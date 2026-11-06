package node

import (
	"testing"

	"github.com/InazumaV/V2bX/api/panel"
)

func u(id int, uuid string, speed, device int) panel.UserInfo {
	return panel.UserInfo{Id: id, Uuid: uuid, SpeedLimit: speed, DeviceLimit: device}
}

// 只改限速：必须走 modified，不能算成「删号 + 新号」。
// 旧实现把 SpeedLimit 拼进 key，这里会得到 deleted=1、added=1，
// 而删号会让该用户所有连接被 CloseAll —— 日限超额时用户会突然掉线。
func TestCompareUserListSpeedChangeIsModified(t *testing.T) {
	old := []panel.UserInfo{u(1, "aaa", 5, 1)}
	next := []panel.UserInfo{u(1, "aaa", 1, 1)}

	deleted, added, modified := compareUserList(old, next)

	if len(deleted) != 0 || len(added) != 0 {
		t.Fatalf("改限速不应产生删/增，得到 deleted=%d added=%d", len(deleted), len(added))
	}
	if len(modified) != 1 || modified[0].SpeedLimit != 1 {
		t.Fatalf("期望 1 个 modified（新限速 1），得到 %+v", modified)
	}
}

// 只改设备数也要被识别（旧实现的 key 里没有 device_limit，改了等于没改）。
func TestCompareUserListDeviceChangeIsModified(t *testing.T) {
	old := []panel.UserInfo{u(1, "aaa", 5, 3)}
	next := []panel.UserInfo{u(1, "aaa", 5, 1)}

	deleted, added, modified := compareUserList(old, next)

	if len(deleted) != 0 || len(added) != 0 {
		t.Fatalf("改设备数不应产生删/增，得到 deleted=%d added=%d", len(deleted), len(added))
	}
	if len(modified) != 1 || modified[0].DeviceLimit != 1 {
		t.Fatalf("期望 1 个 modified（新设备数 1），得到 %+v", modified)
	}
}

// 完全没变时不产生任何变更。
func TestCompareUserListUnchanged(t *testing.T) {
	same := []panel.UserInfo{u(1, "aaa", 5, 1), u(2, "bbb", 0, 0)}

	deleted, added, modified := compareUserList(same, same)

	if len(deleted)+len(added)+len(modified) != 0 {
		t.Fatalf("不该有变更，得到 deleted=%d added=%d modified=%d", len(deleted), len(added), len(modified))
	}
}

// 真正的增删仍然要正确。
func TestCompareUserListAddRemove(t *testing.T) {
	old := []panel.UserInfo{u(1, "aaa", 5, 1)}
	next := []panel.UserInfo{u(2, "bbb", 5, 1)}

	deleted, added, modified := compareUserList(old, next)

	if len(deleted) != 1 || deleted[0].Id != 1 {
		t.Fatalf("期望删除 id=1，得到 %+v", deleted)
	}
	if len(added) != 1 || added[0].Id != 2 {
		t.Fatalf("期望新增 id=2，得到 %+v", added)
	}
	if len(modified) != 0 {
		t.Fatalf("不该有 modified，得到 %+v", modified)
	}
}

// 没有待补报数据时，原样返回本轮的。
func TestMergeUserTrafficNoPending(t *testing.T) {
	fresh := []panel.UserTraffic{{UID: 1, Upload: 10, Download: 20}}
	got := mergeUserTraffic(nil, fresh)
	if len(got) != 1 || got[0].Upload != 10 {
		t.Fatalf("期望原样返回，得到 %+v", got)
	}
}

// 同一个用户：待补报的与新一轮相加，不能覆盖。
func TestMergeUserTrafficSumsSameUser(t *testing.T) {
	pending := []panel.UserTraffic{{UID: 1, Upload: 10, Download: 20}}
	fresh := []panel.UserTraffic{{UID: 1, Upload: 5, Download: 7}}
	got := mergeUserTraffic(pending, fresh)
	if len(got) != 1 {
		t.Fatalf("期望 1 条，得到 %d 条", len(got))
	}
	if got[0].Upload != 15 || got[0].Download != 27 {
		t.Fatalf("期望 up=15 down=27，得到 %+v", got[0])
	}
}

// 不同用户：并存，不互相吞。
func TestMergeUserTrafficKeepsDistinctUsers(t *testing.T) {
	pending := []panel.UserTraffic{{UID: 1, Upload: 10, Download: 20}}
	fresh := []panel.UserTraffic{{UID: 2, Upload: 1, Download: 2}}
	got := mergeUserTraffic(pending, fresh)
	if len(got) != 2 {
		t.Fatalf("期望 2 条，得到 %d 条", len(got))
	}
}
