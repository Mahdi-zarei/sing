package timeout

import (
	"net"
	"time"
)

type NetPacketConnWithTimeout struct {
	net.PacketConn
	timeout             time.Duration
	manualReadDeadline  time.Time
	manualWriteDeadline time.Time
}

func NewNetPacketConnWithTimeout(conn net.PacketConn, timeout time.Duration) net.PacketConn {
	return &NetPacketConnWithTimeout{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *NetPacketConnWithTimeout) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if time.Now().After(c.manualReadDeadline) {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
	}

	return c.PacketConn.ReadFrom(p)
}

func (c *NetPacketConnWithTimeout) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if time.Now().After(c.manualWriteDeadline) {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
	}

	return c.PacketConn.WriteTo(p, addr)
}

func (c *NetPacketConnWithTimeout) SetReadDeadline(t time.Time) error {
	c.manualReadDeadline = t
	return c.PacketConn.SetReadDeadline(t)
}

func (c *NetPacketConnWithTimeout) SetWriteDeadline(t time.Time) error {
	c.manualWriteDeadline = t
	return c.PacketConn.SetWriteDeadline(t)
}

func (c *NetPacketConnWithTimeout) SetDeadline(t time.Time) error {
	c.manualReadDeadline = t
	c.manualWriteDeadline = t
	return c.PacketConn.SetDeadline(t)
}

func (c *NetPacketConnWithTimeout) Upstream() any {
	return c.PacketConn
}
