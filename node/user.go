package node

import (
	"github.com/InazumaV/V2bX/api/panel"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) reportUserTrafficTask() (err error) {
	// 原子地「读取并清零」，不会重复计；推送失败就把这批挂在 pendingTraffic 上，
	// 下一轮和新的用量合并后再报。既不丢量，也不会像「推成功后再读一次清零」
	// 那样把推送期间新产生的流量一起清掉。
	fresh, _ := c.server.GetUserTrafficSlice(c.Tag(), true)
	batch := mergeUserTraffic(c.pendingTraffic, fresh)
	if len(batch) > 0 {
		err = c.apiClient.ReportUserTraffic(batch)
		if err != nil {
			c.pendingTraffic = batch
			log.WithFields(log.Fields{
				"tag":     c.Tag(),
				"err":     err,
				"pending": len(batch),
			}).Error("Report user traffic failed, keep for retry")
		} else {
			c.pendingTraffic = nil
			log.WithField("tag", c.Tag()).Infof("Report %d users traffic", len(batch))
			log.WithField("tag", c.Tag()).Debugf("User traffic: %+v", batch)
		}
	}

	if onlineDevice, err := c.Limiter().GetOnlineDevice(); err != nil {
		log.Print(err)
	} else if len(*onlineDevice) > 0 {
		// Only report user has traffic > 100kb to allow ping test
		var result []panel.OnlineUser
		var nocountUID = make(map[int]struct{})
		for _, traffic := range fresh {
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

// mergeUserTraffic 把「上一轮没报成功的」和「这一轮新增的」按用户合并。
// 只按 uid 累加，不同用户直接并存；返回切片顺序无关紧要。
func mergeUserTraffic(pending, fresh []panel.UserTraffic) []panel.UserTraffic {
	if len(pending) == 0 {
		return fresh
	}
	merged := make(map[int]panel.UserTraffic, len(pending)+len(fresh))
	for _, t := range pending {
		merged[t.UID] = t
	}
	for _, t := range fresh {
		if old, ok := merged[t.UID]; ok {
			merged[t.UID] = panel.UserTraffic{
				UID:      t.UID,
				Upload:   old.Upload + t.Upload,
				Download: old.Download + t.Download,
			}
			continue
		}
		merged[t.UID] = t
	}
	out := make([]panel.UserTraffic, 0, len(merged))
	for _, t := range merged {
		out = append(out, t)
	}
	return out
}
