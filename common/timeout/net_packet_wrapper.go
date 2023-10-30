package timeout

import (
	"net"
	"time"
)

type NetPacketConnWithTimeout struct {
	net.PacketConn
	timeout time.Duration
	lastSet time.Time
}

func NewNetPacketConnWithTimeout(conn net.PacketConn, timeout time.Duration) net.PacketConn {
	return &NetPacketConnWithTimeout{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *NetPacketConnWithTimeout) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.ReadFrom(p)
}

func (c *NetPacketConnWithTimeout) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.WriteTo(p, addr)
}

func (c *NetPacketConnWithTimeout) Upstream() any {
	return c.PacketConn
}
