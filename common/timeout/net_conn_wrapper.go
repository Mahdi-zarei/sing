package timeout

import (
	"net"
	"time"
)

type NetConnWithTimeout struct {
	net.Conn
	timeout time.Duration
	lastSet time.Time
}

func NewNetConnWithTimeout(conn net.Conn, timeout time.Duration) net.Conn {
	return &NetConnWithTimeout{
		Conn:    conn,
		timeout: timeout,
	}
}

func (c *NetConnWithTimeout) Read(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Read(b)
}

func (c *NetConnWithTimeout) Write(b []byte) (n int, err error) {
	if time.Since(c.lastSet) >= 10*time.Second {
		_ = c.Conn.SetDeadline(time.Now().Add(c.timeout))
		c.lastSet = time.Now()
	}

	return c.Conn.Write(b)
}

func (c *NetConnWithTimeout) Upstream() any {
	return c.Conn
}
