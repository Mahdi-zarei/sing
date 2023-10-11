package canceler

import (
	"context"
	"errors"
	"github.com/sagernet/sing/common/buf"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"time"
)

type PacketWithDeadline struct {
	N.PacketConn
	timeout time.Duration
	lastSet time.Time
	cancel  context.CancelCauseFunc
}

func NewPacketWithDeadline(ctx context.Context, conn N.PacketConn, timeout time.Duration) (context.Context, N.PacketConn) {
	ctx, cancel := context.WithCancelCause(ctx)
	return ctx, &PacketWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
		lastSet:    time.Time{},
		cancel:     cancel,
	}
}

func (p *PacketWithDeadline) ReadPacket(buffer *buf.Buffer) (destination M.Socksaddr, err error) {
	if time.Since(p.lastSet) >= time.Second*10 {
		_ = p.PacketConn.SetDeadline(time.Now().Add(p.timeout))
		p.lastSet = time.Now()
	}

	return p.PacketConn.ReadPacket(buffer)
}

func (p *PacketWithDeadline) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) (err error) {
	if time.Since(p.lastSet) >= time.Second*10 {
		_ = p.PacketConn.SetDeadline(time.Now().Add(p.timeout))
		p.lastSet = time.Now()
	}

	return p.PacketConn.WritePacket(buffer, destination)
}

func (p *PacketWithDeadline) Close() error {
	p.cancel(errors.New("connection force closed"))
	return p.PacketConn.Close()
}

func (p *PacketWithDeadline) Upstream() any {
	return p.PacketConn
}
