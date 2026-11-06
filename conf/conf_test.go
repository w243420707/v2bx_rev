package conf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConf_LoadFromPath(t *testing.T) {
	c := New()
	t.Log(c.LoadFromPath("../example/config.json"), c.NodeConfig)
}

// 原来的写法是 `select {}`——永远不结束，跑 go test ./... 会一直挂到超时。
// 这里改成真跑一次：建临时配置 → 起 Watch → 改文件 → 等回调。
// 注意 Watch 内部对事件有 5 秒 debounce（sleep 5s 才 reload），所以超时给到 15 秒。
func TestConf_Watch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"Log":{"Level":"info"}}`), 0o644); err != nil {
		t.Fatalf("写临时配置失败: %v", err)
	}

	c := New()
	fired := make(chan struct{}, 1)
	if err := c.Watch(path, "", "", func() { fired <- struct{}{} }); err != nil {
		t.Fatalf("Watch 启动失败: %v", err)
	}

	// 等 watcher 的 Add 生效，否则紧接着的写入可能赶在监听之前。
	time.Sleep(200 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"Log":{"Level":"debug"}}`), 0o644); err != nil {
		t.Fatalf("修改临时配置失败: %v", err)
	}

	select {
	case <-fired:
	case <-time.After(15 * time.Second):
		t.Fatal("配置文件变化后 15 秒内没有触发 reload 回调")
	}
}
