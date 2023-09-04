package canceler

import (
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"time"
)

type PacketWithDeadline struct {
	N.PacketConn
	timeout time.Duration
}

func NewPacketWithDeadline(conn N.PacketConn, timeout time.Duration) N.PacketConn {
	return &PacketWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (p *PacketWithDeadline) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	_ = p.PacketConn.SetDeadline(time.Now().Add(p.timeout))

	return p.PacketConn.ReadPacket(buffer)
}

func (p *PacketWithDeadline) Upstream() any {
	return p.PacketConn
}
