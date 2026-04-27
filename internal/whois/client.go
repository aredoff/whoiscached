package whois

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	MaxResponseBytes int64
}

func (c *Client) Query(ctx context.Context, server, query string, timeout time.Duration) ([]byte, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil, fmt.Errorf("empty server")
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host = server
		port = "43"
	}
	addr := net.JoinHostPort(host, port)
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(timeout)
	}
	if err = conn.SetDeadline(deadline); err != nil {
		return nil, err
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("empty query")
	}
	msg := q + "\r\n"
	if _, err = io.Copy(conn, strings.NewReader(msg)); err != nil {
		return nil, fmt.Errorf("write query: %w", err)
	}

	max := c.MaxResponseBytes
	if max <= 0 {
		max = 2 << 20
	}
	var out []byte
	buf := make([]byte, 32<<10)
	for {
		n, rerr := conn.Read(buf)
		if n > 0 {
			remaining := max - int64(len(out))
			if remaining <= 0 {
				break
			}
			if int64(n) > remaining {
				out = append(out, buf[:remaining]...)
				break
			}
			out = append(out, buf[:n]...)
		}
		if rerr != nil {
			if rerr == io.EOF {
				break
			}
			return nil, fmt.Errorf("read response: %w", rerr)
		}
	}
	return out, nil
}

func HostPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		if _, _, err := net.SplitHostPort(host); err == nil {
			return host
		}
	}
	return net.JoinHostPort(host, "43")
}

func ParsePort(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 43
	}
	p, err := strconv.Atoi(s)
	if err != nil || p <= 0 || p > 65535 {
		return 43
	}
	return p
}
