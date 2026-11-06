package rate

import (
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
	limiter *DynamicBucket
}

func NewPacketConnRateLimiter(c N.PacketConn, l *DynamicBucket) *PacketConn {
	return &PacketConn{
		PacketConn: c,
		limiter:    l,
	}
}

func (c *PacketConn) ReadPacket(buffer *buf.Buffer) (M.Socksaddr, error) {
	dest, err := c.PacketConn.ReadPacket(buffer)
	if err != nil || buffer == nil {
		return dest, err
	}
	size := int64(buffer.Len())
	if size <= 0 {
		return dest, nil
	}
	// 上行也限速。UDP 没有重传、也没有背压可用，能做的只有「超速就丢」：
	// 这里用非阻塞的 TakeAvailable，拿不到配额就把这一包清空丢掉。
	// 千万不要在收包循环里 Wait —— 那会把整条读链路堵住，
	// 内核缓冲区一满就是连片丢包，比限速本身更糟。
	if got := c.limiter.Get().TakeAvailable(size); got < size {
		buffer.FullReset()
	}
	return dest, nil
}

func (c *PacketConn) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	if buffer != nil {
		c.limiter.Wait(int64(buffer.Len()))
	}
	return c.PacketConn.WritePacket(buffer, destination)
}
