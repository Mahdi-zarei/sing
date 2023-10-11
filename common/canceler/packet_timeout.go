package canceler

import (
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"time"
)

type PacketWithDeadline struct {
	N.PacketConn
	timeout      time.Duration
	lastReadSet  time.Time
	lastWriteSet time.Time
}

func NewPacketWithDeadline(conn N.PacketConn, timeout time.Duration) N.PacketConn {
	return &PacketWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
	}
}

func (p *PacketWithDeadline) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	if time.Since(p.lastReadSet) >= time.Second*10 {
		_ = p.PacketConn.SetReadDeadline(time.Now().Add(p.timeout))
		p.lastReadSet = time.Now()
	}

	return p.PacketConn.ReadPacket(buffer)
}

func (p *PacketWithDeadline) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) (err error) {
	if time.Since(p.lastWriteSet) >= time.Second*10 {
		_ = p.PacketConn.SetWriteDeadline(time.Now().Add(p.timeout))
		p.lastWriteSet = time.Now()
	}

	return p.PacketConn.WritePacket(buffer, destination)
}

func (p *PacketWithDeadline) Close() error {
	return p.PacketConn.Close()
}

func (p *PacketWithDeadline) Upstream() any {
	return p.PacketConn
}
