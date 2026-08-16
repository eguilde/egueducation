package registratura

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClamdScannerResponses(t *testing.T) {
	for _, tc := range []struct {
		response string
		clean    bool
		wantErr  bool
		partial  bool
	}{{"stream: OK\000", true, false, true}, {"stream: Eicar-Test-Signature FOUND\000", false, false, false}, {"bad\000", false, true, false}, {"", false, true, false}} {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			c, _ := ln.Accept()
			if c == nil {
				return
			}
			defer c.Close()
			buf := make([]byte, 10)
			_, _ = c.Read(buf) // command
			for {
				var h [4]byte
				if _, err := io.ReadFull(c, h[:]); err != nil {
					return
				}
				n := int(h[0])<<24 | int(h[1])<<16 | int(h[2])<<8 | int(h[3])
				if n == 0 {
					break
				}
				_, _ = io.CopyN(io.Discard, c, int64(n))
			}
			response := []byte(tc.response)
			if tc.partial {
				_, _ = c.Write(response[:3])
				time.Sleep(10 * time.Millisecond)
				response = response[3:]
			}
			_, _ = c.Write(response)
		}()
		s := ClamdScanner{Address: ln.Addr().String(), Timeout: time.Second}
		clean, err := s.Scan(context.Background(), bytes.NewReader([]byte("x")))
		ln.Close()
		if clean != tc.clean || (err != nil) != tc.wantErr {
			t.Fatalf("response %q clean=%v err=%v", tc.response, clean, err)
		}
	}
}

func TestAttachmentObjectKeyIsUniqueAndNamespaced(t *testing.T) {
	first := attachmentObjectKey("school-a", "doc-a", "my file.pdf")
	second := attachmentObjectKey("school-a", "doc-a", "my file.pdf")
	if first == second {
		t.Fatal("attachment keys must not overwrite same-name attachments")
	}
	if !strings.HasPrefix(first, "tenants/school-a/registratura/doc-a/attachments/") || !strings.HasSuffix(first, "/my-file.pdf") {
		t.Fatalf("unexpected key %q", first)
	}
}
