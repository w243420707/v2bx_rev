package rate

import (
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// PacketConn 给 UDP 转发套一层限速。
//
// 只限 WritePacket（发回客户端的方向）。
//
// ⚠️ 不要在这里包 ReadPacket：v0.1.12 试过「超速就 FullReset 丢包」，
// 被清空的 buffer 会继续往下游走，vmess 的 mux 要往前面插 16 字节头时
// 已经没有空间，直接 panic: buffer overflow (start 0, need 16)，节点反复重启。
// 上行限速要么在更外层做（不破坏 buffer），要么就不做。
type PacketConn struct {
	N.PacketConn
	limiter *DynamicBucket
}

func NewPacketConnRateLimiter(c N.PacketConn, l *DynamicBucket) *PacketConn {
	return &PacketConn{
		PacketConn: c,
		limiter:    l,
	}
}

func (c *PacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	if buffer != nil {
		c.limiter.Wait(int64(buffer.Len()))
	}
	return c.PacketConn.WritePacket(buffer, destination)
}
