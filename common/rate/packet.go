package rate

import (
	"github.com/juju/ratelimit"
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// PacketConn 给 UDP 转发套一层限速。
//
// 只限 WritePacket（发回客户端的方向）：ReadPacket 上等待会卡住整个收包循环，
// 内核缓冲区一满就是丢包，而不是降速。下行是主要流量，先把它管住，
// 上行与 xray 内核的行为保持一致（xray 的 DispatchLink 也只包了 outbound.Writer）。
type PacketConn struct {
	N.PacketConn
	limiter *ratelimit.Bucket
}

func NewPacketConnRateLimiter(c N.PacketConn, l *ratelimit.Bucket) *PacketConn {
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
