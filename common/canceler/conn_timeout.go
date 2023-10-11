package canceler

import (
	"context"
	"errors"
	"net"
	"time"
)

type ConnWithDeadline struct {
	net.Conn
	timeout time.Duration
	lastSet time.Time
	cancel  context.CancelCauseFunc
}

func NewConnWithDeadline(ctx context.Context, conn net.Conn, timeout time.Duration) (context.Context, net.Conn) {
	ctx, cancel := context.WithCancelCause(ctx)
	return ctx, &ConnWithDeadline{
		Conn:    conn,
		timeout: timeout,
		lastSet: time.Time{},
		cancel:  cancel,
	}
}

func (c *ConnWithDeadline) Read(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= time.Second*10 {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Read(b)
}

func (c *ConnWithDeadline) Write(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= time.Second*10 {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Write(b)
}

func (c *ConnWithDeadline) Close() error {
	c.cancel(errors.New("connection force closed"))
	return c.Conn.Close()
}

func (c *ConnWithDeadline) Upstream() any {
	return c.Conn
}
