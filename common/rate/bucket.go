package rate

import (
	"sync/atomic"
	"time"

	"github.com/juju/ratelimit"
)

// DynamicBucket 是可换速率的令牌桶。
//
// juju/ratelimit 的 Bucket 建好之后速率就固定了。连接建立时拿到的是桶的指针，
// 面板把 speed_limit 调低时（日限 / 小时限超额就是走这条路），
// 已经连着的会话会一直按旧速率跑到断开为止——用户看到的是「限速不生效」。
//
// 这里用 atomic.Value 包一层：换的只是里面的桶，外面持有者不变，
// 已有连接下一次读写自然就按新速率走，不用断线。
type DynamicBucket struct {
	current atomic.Value // *ratelimit.Bucket
	rate    atomic.Int64 // 当前速率（字节/秒），用来判断要不要重建
}

func NewDynamicBucket(rate int64) *DynamicBucket {
	d := &DynamicBucket{}
	d.store(rate)
	return d
}

func (d *DynamicBucket) store(rate int64) {
	if rate < 1 {
		rate = 1
	}
	d.rate.Store(rate)
	d.current.Store(ratelimit.NewBucketWithQuantum(time.Second, rate, rate))
}

// Update 换速率。速率没变就什么都不做，
// 否则每次判定都重建桶，会把桶里攒着的余量清掉，表现是速度一卡一卡的。
func (d *DynamicBucket) Update(rate int64) {
	if rate < 1 {
		rate = 1
	}
	if d.rate.Load() == rate {
		return
	}
	d.store(rate)
}

// Rate 当前速率，字节/秒。
func (d *DynamicBucket) Rate() int64 {
	return d.rate.Load()
}

func (d *DynamicBucket) Get() *ratelimit.Bucket {
	b, _ := d.current.Load().(*ratelimit.Bucket)
	return b
}

// Wait 阻塞到攒够 n 个字节的配额。
func (d *DynamicBucket) Wait(n int64) {
	if b := d.Get(); b != nil {
		b.Wait(n)
	}
}
