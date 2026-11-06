package node

import (
	"github.com/InazumaV/V2bX/api/panel"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) reportUserTrafficTask() (err error) {
	// 先读不清零：推送失败时计数器还在，下一轮会把这批流量一起报上去。
	// 以前是「先清零再推」，面板一抖这一批就永久丢了，用户等于白用。
	userTraffic, _ := c.server.GetUserTrafficSlice(c.Tag(), false)
	if len(userTraffic) > 0 {
		err = c.apiClient.ReportUserTraffic(userTraffic)
		if err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Error("Report user traffic failed, keep counters for retry")
		} else {
			// 推成功再清零。两次读取之间的几十毫秒新流量会一并清零，
			// 相比「整批丢失」，这点误差可以接受。
			_, _ = c.server.GetUserTrafficSlice(c.Tag(), true)
			log.WithField("tag", c.Tag()).Infof("Report %d users traffic", len(userTraffic))
			log.WithField("tag", c.Tag()).Debugf("User traffic: %+v", userTraffic)
		}
	}

	if onlineDevice, err := c.Limiter().GetOnlineDevice(); err != nil {
		log.Print(err)
	} else if len(*onlineDevice) > 0 {
		// Only report user has traffic > 100kb to allow ping test
		var result []panel.OnlineUser
		var nocountUID = make(map[int]struct{})
		for _, traffic := range userTraffic {
			total := traffic.Upload + traffic.Download
			if total < int64(c.Options.DeviceOnlineMinTraffic*1000) {
				nocountUID[traffic.UID] = struct{}{}
			}
		}
		for _, online := range *onlineDevice {
			if _, ok := nocountUID[online.UID]; !ok {
				result = append(result, online)
			}
		}
		data := make(map[int][]string)
		for _, onlineuser := range result {
			// json structure: { UID1:["ip1","ip2"],UID2:["ip3","ip4"] }
			data[onlineuser.UID] = append(data[onlineuser.UID], onlineuser.IP)
		}
		if err = c.apiClient.ReportNodeOnlineUsers(&data); err != nil {
			log.WithFields(log.Fields{
				"tag": c.Tag(),
				"err": err,
			}).Info("Report online users failed")
		} else {
			log.WithField("tag", c.Tag()).Infof("Total %d online users, %d Reported", len(*onlineDevice), len(result))
			log.WithField("tag", c.Tag()).Debugf("Online users: %+v", data)
		}
	}

	userTraffic = nil
	return nil
}

// compareUserList 比对两轮用户名单。
//
// 旧实现把 SpeedLimit 拼进 key：限速一变，旧记录就被当成"删号"、新记录算"新增"，
// 而删号会 CloseAll 掉该用户所有连接 —— 日限/小时限一超额，表现是用户突然掉线，
// 不是被降速。现在 key 只用 UUID，限速/设备数的变化走 modified，就地更新，不断连。
func compareUserList(old, new []panel.UserInfo) (deleted, added, modified []panel.UserInfo) {
	oldMap := make(map[string]panel.UserInfo, len(old))
	for _, user := range old {
		oldMap[user.Uuid] = user
	}

	for _, user := range new {
		prev, exists := oldMap[user.Uuid]
		if !exists {
			added = append(added, user)
			continue
		}
		if prev.SpeedLimit != user.SpeedLimit || prev.DeviceLimit != user.DeviceLimit {
			modified = append(modified, user)
		}
		delete(oldMap, user.Uuid)
	}

	for _, user := range oldMap {
		deleted = append(deleted, user)
	}

	return deleted, added, modified
}
