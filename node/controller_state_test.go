package node

import (
	"sync"
	"testing"

	"github.com/InazumaV/V2bX/limiter"
)

// tag 与 limiter 会被「拉配置」和「上报流量」两个定时任务同时访问。
// 这个用例在 -race 下能抓到裸读写的竞争（本机没 gcc，跑不了 -race，CI 上跑）。
func TestControllerStateConcurrentAccess(t *testing.T) {
	c := &Controller{}
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				_ = c.Tag()
				_ = c.Limiter()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 2000; j++ {
			c.setTag("node-tag")
			c.setLimiter(&limiter.Limiter{})
		}
	}()

	wg.Wait()

	if c.Tag() != "node-tag" {
		t.Fatalf("tag 期望 node-tag，得到 %q", c.Tag())
	}
	if c.Limiter() == nil {
		t.Fatal("limiter 不该为空")
	}
}
