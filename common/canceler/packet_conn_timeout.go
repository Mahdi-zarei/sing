package canceler

import (
	"net"
	"time"
)

type NetPacketConnWithDeadline struct {
	net.PacketConn
	timeout      time.Duration
	lastReadSet  time.Time
	lastWriteSet time.Time
}

func NewNetPacketConnWithDeadline(conn net.PacketConn, timeout time.Duration) net.PacketConn {
	return &NetPacketConnWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *NetPacketConnWithDeadline) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if time.Since(c.lastReadSet) >= time.Second*10 {
		_ = c.PacketConn.SetReadDeadline(time.Now().Add(c.timeout))
		c.lastReadSet = time.Now()
	}

	return c.PacketConn.ReadFrom(p)
}

func (c *NetPacketConnWithDeadline) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if time.Since(c.lastWriteSet) >= time.Second*10 {
		_ = c.PacketConn.SetWriteDeadline(time.Now().Add(c.timeout))
		c.lastWriteSet = time.Now()
	}

	return c.PacketConn.WriteTo(p, addr)
}

func (c *NetPacketConnWithDeadline) Close() error {
	return c.PacketConn.Close()
}

func (c *NetPacketConnWithDeadline) Upstream() any {
	return c.PacketConn
}
