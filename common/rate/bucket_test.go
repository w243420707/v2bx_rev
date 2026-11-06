package rate

import "testing"

// 换速率必须真的换掉里面的桶，否则「面板调低限速，老连接还是旧速度」。
func TestDynamicBucketUpdateReplacesBucket(t *testing.T) {
	b := NewDynamicBucket(1_000_000)
	if got := b.Rate(); got != 1_000_000 {
		t.Fatalf("初始速率期望 1000000，得到 %d", got)
	}
	before := b.Get()

	b.Update(500_000)

	if got := b.Rate(); got != 500_000 {
		t.Fatalf("Update 后速率期望 500000，得到 %d", got)
	}
	if b.Get() == before {
		t.Fatal("Update 后应该换成新的桶，实际还是同一个")
	}
}

// 速率没变时不能重建桶：重建会把桶里攒的余量清掉，表现是速度一卡一卡的。
func TestDynamicBucketUpdateSameRateKeepsBucket(t *testing.T) {
	b := NewDynamicBucket(1_000_000)
	before := b.Get()

	b.Update(1_000_000)

	if b.Get() != before {
		t.Fatal("速率没变时不该换桶")
	}
}

// 速率被压到 0 或负数时按 1 字节/秒处理，不能出现除零或「不限速」的意外。
func TestDynamicBucketClampsToMinimum(t *testing.T) {
	b := NewDynamicBucket(0)
	if got := b.Rate(); got != 1 {
		t.Fatalf("速率 0 应被夹到 1，得到 %d", got)
	}
	b.Update(-5)
	if got := b.Rate(); got != 1 {
		t.Fatalf("负速率应被夹到 1，得到 %d", got)
	}
}
