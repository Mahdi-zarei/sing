package canceler

import (
	"context"
	"errors"
	"net"
	"time"
)

type NetPacketConnWithDeadline struct {
	net.PacketConn
	timeout time.Duration
	lastSet time.Time
	cancel  context.CancelCauseFunc
}

func NewNetPacketConnWithDeadline(ctx context.Context, conn net.PacketConn, timeout time.Duration) (context.Context, net.PacketConn) {
	ctx, cancel := context.WithCancelCause(ctx)
	return ctx, &NetPacketConnWithDeadline{
		PacketConn: conn,
		timeout:    timeout,
		lastSet:    time.Time{},
		cancel:     cancel,
	}
}

func (c *NetPacketConnWithDeadline) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if time.Since(c.lastSet) >= time.Second*10 {
		_ = c.PacketConn.SetReadDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.ReadFrom(p)
}

func (c *NetPacketConnWithDeadline) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if time.Since(c.lastSet) >= time.Second*10 {
		_ = c.PacketConn.SetReadDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.PacketConn.WriteTo(p, addr)
}

func (c *NetPacketConnWithDeadline) Close() error {
	c.cancel(errors.New("connection force closed"))
	return c.PacketConn.Close()
}

func (c *NetPacketConnWithDeadline) Upstream() any {
	return c.PacketConn
}
