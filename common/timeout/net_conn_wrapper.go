package timeout

import (
	"net"
	"time"
)

type NetConnWithTimeout struct {
	net.Conn
	timeout        time.Duration
	lastSet        time.Time
	manualDeadline time.Time
}

func NewNetConnWithTimeout(conn net.Conn, timeout time.Duration) net.Conn {
	return &NetConnWithTimeout{
		Conn:    conn,
		timeout: timeout,
	}
}

func (c *NetConnWithTimeout) Read(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= c.timeout/4 && time.Now().After(c.manualDeadline) {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Read(b)
}

func (c *NetConnWithTimeout) Write(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= c.timeout/4 && time.Now().After(c.manualDeadline) {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Write(b)
}

func (c *NetConnWithTimeout) SetReadDeadline(t time.Time) error {
	c.manualDeadline = t
	return c.Conn.SetReadDeadline(t)
}

func (c *NetConnWithTimeout) SetWriteDeadline(t time.Time) error {
	c.manualDeadline = t
	return c.Conn.SetWriteDeadline(t)
}

func (c *NetConnWithTimeout) SetDeadline(t time.Time) error {
	c.manualDeadline = t
	return c.Conn.SetDeadline(t)
}

func (c *NetConnWithTimeout) Upstream() any {
	return c.Conn
}
