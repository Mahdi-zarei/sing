package canceler

import (
	"net"
	"time"
)

type NetPacketConnWithDeadline struct {
	net.PacketConn
	timeout time.Duration
}

func NewNetPacketConnWithDeadline(conn net.PacketConn, timeout time.Duration) net.PacketConn {
	return &NetPacketConnWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *NetPacketConnWithDeadline) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))

	return c.PacketConn.ReadFrom(p)
}

func (c *NetPacketConnWithDeadline) Upstream() any {
	return c.PacketConn
}
