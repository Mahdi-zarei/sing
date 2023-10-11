package canceler

import (
	"net"
	"time"
)

type ConnWithDeadline struct {
	net.Conn
	timeout      time.Duration
	lastReadSet  time.Time
	lastWriteSet time.Time
}

func NewConnWithDeadline(conn net.Conn, timeout time.Duration) net.Conn {
	return &ConnWithDeadline{
		Conn:    conn,
		timeout: timeout,
	}
}

func (c *ConnWithDeadline) Read(b []byte) (n int, err error) {
	if time.Since(c.lastReadSet) >= time.Second*10 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))
		c.lastReadSet = time.Now()
	}

	return c.Conn.Read(b)
}

func (c *ConnWithDeadline) Write(b []byte) (n int, err error) {
	if time.Since(c.lastWriteSet) >= time.Second*10 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
		c.lastWriteSet = time.Now()
	}

	return c.Conn.Write(b)
}

func (c *ConnWithDeadline) Close() error {
	return c.Conn.Close()
}

func (c *ConnWithDeadline) Upstream() any {
	return c.Conn
}
