package timeout

import (
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"time"
)

type PacketConnWithTimeout struct {
	N.PacketConn
	timeout             time.Duration
	manualReadDeadline  time.Time
	manualWriteDeadline time.Time
}

func NewPacketConnWithTimeout(conn N.PacketConn, timeout time.Duration) N.PacketConn {
	return &PacketConnWithTimeout{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *PacketConnWithTimeout) ReadPacket(buffer *buf.Buffer) (destination metadata.Socksaddr, err error) {
	if time.Now().After(c.manualReadDeadline) {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
	}

	return c.PacketConn.ReadPacket(buffer)
}

func (c *PacketConnWithTimeout) WritePacket(buffer *buf.Buffer, destination metadata.Socksaddr) error {
	if time.Now().After(c.manualWriteDeadline) {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
	}

	return c.PacketConn.WritePacket(buffer, destination)
}

func (c *PacketConnWithTimeout) SetReadDeadline(t time.Time) error {
	c.manualReadDeadline = t
	return c.PacketConn.SetReadDeadline(t)
}

func (c *PacketConnWithTimeout) SetWriteDeadline(t time.Time) error {
	c.manualWriteDeadline = t
	return c.PacketConn.SetWriteDeadline(t)
}

func (c *PacketConnWithTimeout) SetDeadline(t time.Time) error {
	c.manualReadDeadline = t
	c.manualWriteDeadline = t
	return c.PacketConn.SetDeadline(t)
}

func (c *PacketConnWithTimeout) Upstream() any {
	return c.PacketConn
}
