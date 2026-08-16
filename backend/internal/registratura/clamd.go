package registratura

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Scanner interface {
	Scan(context.Context, io.Reader) (clean bool, err error)
}
type ClamdScanner struct {
	Address string
	Timeout time.Duration
}

func (s ClamdScanner) Scan(ctx context.Context, body io.Reader) (bool, error) {
	if strings.TrimSpace(s.Address) == "" {
		return false, fmt.Errorf("clamd unavailable")
	}
	d := net.Dialer{Timeout: s.Timeout}
	c, err := d.DialContext(ctx, "tcp", s.Address)
	if err != nil {
		return false, err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(s.Timeout))
	if _, err = c.Write([]byte("zINSTREAM\000")); err != nil {
		return false, err
	}
	buf := make([]byte, 32*1024)
	for {
		n, e := body.Read(buf)
		if n > 0 {
			var h [4]byte
			h[0] = byte(n >> 24)
			h[1] = byte(n >> 16)
			h[2] = byte(n >> 8)
			h[3] = byte(n)
			if _, err = c.Write(h[:]); err != nil {
				return false, err
			}
			if _, err = c.Write(buf[:n]); err != nil {
				return false, err
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return false, e
		}
	}
	if _, err = c.Write([]byte{0, 0, 0, 0}); err != nil {
		return false, err
	}
	response, err := bufio.NewReader(io.LimitReader(c, 4096)).ReadString(0)
	if err != nil {
		return false, err
	}
	if strings.Contains(response, "FOUND") {
		return false, nil
	}
	if strings.Contains(response, "OK") {
		return true, nil
	}
	return false, fmt.Errorf("unexpected clamd response: %s", response)
}
func scanBuffer(scanner Scanner, data []byte) (bool, error) {
	return scanner.Scan(context.Background(), bytes.NewReader(data))
}
