package talk

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	defaultPort       = 4010
	defaultFrameBytes = 640
)

// Client streams 8000 Hz mono signed 16-bit little-endian PCM to atomtalkd.
type Client struct {
	host  string
	port  int
	token string
}

// NewClient creates a talk client for one camera.
func NewClient(host string, port int, token string) *Client {
	if port == 0 {
		port = defaultPort
	}
	return &Client{
		host:  host,
		port:  port,
		token: token,
	}
}

// Stream reads raw PCM from r and forwards it to atomtalkd over UDP.
func (c *Client) Stream(ctx context.Context, r io.Reader) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.host, c.port))
	if err != nil {
		return fmt.Errorf("resolve talk address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("connect talk udp: %w", err)
	}
	defer conn.Close()
	defer c.sendStop(conn)

	if err := c.sendControl(conn, ""); err != nil {
		return err
	}

	buf := make([]byte, defaultFrameBytes)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := io.ReadFull(r, buf)
		if n > 0 {
			if n&1 == 1 {
				n--
			}
			if n > 0 {
				if _, err := conn.Write(buf[:n]); err != nil {
					return fmt.Errorf("send talk packet: %w", err)
				}
			}
		}
		if readErr == nil {
			continue
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			return nil
		}
		return fmt.Errorf("read pcm: %w", readErr)
	}
}

func (c *Client) sendControl(conn *net.UDPConn, command string) error {
	line := "ATOMTALK"
	if c.token != "" {
		line += " " + c.token
	}
	if command != "" {
		line += " " + command
	}
	if _, err := conn.Write([]byte(line + "\n")); err != nil {
		return fmt.Errorf("send talk control: %w", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var reply [128]byte
	n, _, err := conn.ReadFromUDP(reply[:])
	_ = conn.SetReadDeadline(time.Time{})
	if err != nil {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil
		}
		return fmt.Errorf("read talk control reply: %w", err)
	}
	if n >= 2 && string(reply[:2]) == "OK" {
		return nil
	}
	return fmt.Errorf("talk control rejected: %s", string(reply[:n]))
}

func (c *Client) sendStop(conn *net.UDPConn) {
	_ = c.sendControl(conn, "STOP")
}
