package timeout

import (
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"time"
)

type PacketConnWithTimeout struct {
	N.PacketConn
	timeout time.Duration
	lastSet time.Time
}

func NewPacketConnWithTimeout(conn N.PacketConn, timeout time.Duration) N.PacketConn {
	return &PacketConnWithTimeout{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (c *PacketConnWithTimeout) ReadPacket(buffer *buf.Buffer) (destination metadata.Socksaddr, err error) {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.ReadPacket(buffer)
}

func (c *PacketConnWithTimeout) WritePacket(buffer *buf.Buffer, destination metadata.Socksaddr) error {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.PacketConn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.WritePacket(buffer, destination)
}

func (c *PacketConnWithTimeout) Upstream() any {
	return c.PacketConn
}
