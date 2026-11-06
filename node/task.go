package node

import (
	"time"

	"github.com/InazumaV/V2bX/api/panel"
	"github.com/InazumaV/V2bX/common/task"
	vCore "github.com/InazumaV/V2bX/core"
	"github.com/InazumaV/V2bX/limiter"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) startTasks(node *panel.NodeInfo) {
	// fetch node info task
	c.nodeInfoMonitorPeriodic = &task.Task{
		Interval: node.PullInterval,
		Execute:  c.nodeInfoMonitor,
	}
	// fetch user list task
	c.userReportPeriodic = &task.Task{
		Interval: node.PushInterval,
		Execute:  c.reportUserTrafficTask,
	}
	log.WithField("tag", c.Tag()).Info("Start monitor node status")
	// delay to start nodeInfoMonitor
	_ = c.nodeInfoMonitorPeriodic.Start(false)
	log.WithField("tag", c.Tag()).Info("Start report node status")
	_ = c.userReportPeriodic.Start(false)
	if node.Security == panel.Tls {
		switch c.CertConfig.CertMode {
		case "none", "", "file", "self":
		default:
			c.renewCertPeriodic = &task.Task{
				Interval: time.Hour * 24,
				Execute:  c.renewCertTask,
			}
			log.WithField("tag", c.Tag()).Info("Start renew cert")
			// delay to start renewCert
			_ = c.renewCertPeriodic.Start(true)
		}
	}
	if c.LimitConfig.EnableDynamicSpeedLimit {
		c.traffic = make(map[string]int64)
		c.dynamicSpeedLimitPeriodic = &task.Task{
			Interval: time.Duration(c.LimitConfig.DynamicSpeedLimitConfig.Periodic) * time.Second,
			Execute:  c.SpeedChecker,
		}
		log.Printf("[%s: %d] Start dynamic speed limit", c.apiClient.NodeType, c.apiClient.NodeId)
	}
}

func (c *Controller) nodeInfoMonitor() (err error) {
	// get node info
	newN, err := c.apiClient.GetNodeInfo()
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.Tag(),
			"err": err,
		}).Error("Get node info failed")
		return nil
	}
	// get user info
	newU, err := c.apiClient.GetUserList()
	if err != nil {
		log.WithFields(log.Fields{
			"tag": c.Tag(),
			"err": err,
		}).Error("Get user list failed")
		return nil
	}
	// get user alive
	newA, err := c.apiClient.GetUserAlive()
	if err != nil {
		// 拉不到就沿用上一轮的在线 IP 数据。
		// 这里不能 return：那会让用户名单也停在这一轮不更新，
		// 「设备数可能旧一点」比「流量/封禁/限速全都不生效」轻得多。
		log.WithFields(log.Fields{
			"tag": c.Tag(),
			"err": err,
		}).Warn("Get alive list failed, keep last alive list")
		newA = nil
	}
	if newN == nil && c.needRebuild && c.info != nil {
		// 上一轮重建失败，但面板那边已经 304 了（ETag + body hash 双重去重），
		// 配置不再变化就永远不会重新下发，节点会一直停在「半重建」状态。
		// 这里用最后一次收到的配置强制重试一次。
		newN = c.info
		log.WithField("tag", c.Tag()).Warn("retry node rebuild with last known config")
	}
	if newN != nil {
		c.info = newN
		// 本轮的名单临时变量：只有重建全部成功才写回 c.userList，
		// 否则失败后 diff 会认为"没有新增用户"，他们再也加不进来。
		users := c.userList
		if newU != nil {
			users = newU
		}
		c.traffic = make(map[string]int64)
		// Remove old node
		log.WithField("tag", c.Tag()).Info("Node changed, reload")
		err = c.server.DelNode(c.Tag())
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Delete node failed, retry next round")
			c.needRebuild = true
			return nil
		}

		// Update limiter
		if len(c.Options.Name) == 0 {
			c.setTag(c.buildNodeTag(newN))
			// Remove Old limiter
			limiter.DeleteLimiter(c.Tag())
			// Add new Limiter
			l := limiter.AddLimiter(c.Tag(), &c.LimitConfig, users, newA)
			c.setLimiter(l)
		}
		// update alive list
		if newA != nil {
			c.Limiter().SetAlive(newA)
		}
		// Update rule
		err = c.Limiter().UpdateRule(&newN.Rules)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Update Rule failed")
			c.needRebuild = true
			return nil
		}

		// check cert
		if newN.Security == panel.Tls {
			err = c.requestCert()
			if err != nil {
				log.WithFields(log.Fields{
					"tag": c.Tag(),
					"err": err,
				}).Error("Request cert failed")
				return nil
			}
		}
		// add new node
		err = c.server.AddNode(c.Tag(), newN, c.Options)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Add node failed, retry next round")
			c.needRebuild = true
			return nil
		}
		_, err = c.server.AddUsers(&vCore.AddUsersParams{
			Tag:      c.Tag(),
			Users:    users,
			NodeInfo: newN,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Add users failed")
			c.needRebuild = true
			return nil
		}
		// 全部成功，才把新名单写回
		c.userList = users
		c.needRebuild = false
		// Check interval
		if c.nodeInfoMonitorPeriodic.Interval != newN.PullInterval &&
			newN.PullInterval != 0 {
			c.nodeInfoMonitorPeriodic.Interval = newN.PullInterval
			c.nodeInfoMonitorPeriodic.Close()
			_ = c.nodeInfoMonitorPeriodic.Start(false)
		}
		if c.userReportPeriodic.Interval != newN.PushInterval &&
			newN.PushInterval != 0 {
			c.userReportPeriodic.Interval = newN.PushInterval
			c.userReportPeriodic.Close()
			_ = c.userReportPeriodic.Start(false)
		}
		log.WithField("tag", c.Tag()).Infof("Added %d new users", len(c.userList))
		// exit
		return nil
	}
	// update alive list
	if newA != nil {
		c.Limiter().SetAlive(newA)
	}
	// node no changed, check users
	if len(newU) == 0 {
		return nil
	}
	deleted, added, modified := compareUserList(c.userList, newU)
	if len(deleted) > 0 {
		// have deleted users
		err = c.server.DelUsers(deleted, c.Tag(), c.info)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Delete users failed")
			return nil
		}
	}
	if len(added) > 0 {
		// have added users
		_, err = c.server.AddUsers(&vCore.AddUsersParams{
			Tag:      c.Tag(),
			NodeInfo: c.info,
			Users:    added,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Add users failed")
			return nil
		}
	}
	if len(added) > 0 || len(deleted) > 0 || len(modified) > 0 {
		// update Limiter
		c.Limiter().UpdateUser(c.Tag(), added, deleted, modified)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("limiter users failed")
			return nil
		}
		// clear traffic record
		if c.LimitConfig.EnableDynamicSpeedLimit {
			for i := range deleted {
				delete(c.traffic, deleted[i].Uuid)
			}
		}
	}
	c.userList = newU
	if len(added)+len(deleted) != 0 {
		log.WithField("tag", c.Tag()).
			Infof("%d user deleted, %d user added", len(deleted), len(added))
	}
	return nil
}

func (c *Controller) SpeedChecker() error {
	for u, t := range c.traffic {
		if t >= c.LimitConfig.DynamicSpeedLimitConfig.Traffic {
			err := c.Limiter().UpdateDynamicSpeedLimit(c.Tag(), u,
				c.LimitConfig.DynamicSpeedLimitConfig.SpeedLimit,
				time.Now().Add(time.Duration(c.LimitConfig.DynamicSpeedLimitConfig.ExpireTime)*time.Minute))
			log.WithField("err", err).Error("Update dynamic speed limit failed")
			delete(c.traffic, u)
		}
	}
	return nil
}
