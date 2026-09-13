package regulatorysource

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeResolver struct {
	lookup func(context.Context, string, string) ([]netip.Addr, error)
}

func (r fakeResolver) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return r.lookup(ctx, network, host)
}

func TestFetcherValidateURL(t *testing.T) {
	fetcher, err := NewFetcher([]string{"publisher.example"})
	if err != nil {
		t.Fatalf("NewFetcher() error = %v", err)
	}

	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "exact approved host", raw: "https://publisher.example/notices?id=7", want: true},
		{name: "approved host case insensitive", raw: "https://PUBLISHER.EXAMPLE/notices", want: true},
		{name: "unapproved subdomain", raw: "https://cdn.publisher.example/notices", want: false},
		{name: "unapproved suffix", raw: "https://publisher.example.attacker.invalid/notices", want: false},
		{name: "http", raw: "http://publisher.example/notices", want: false},
		{name: "nonstandard port", raw: "https://publisher.example:8443/notices", want: false},
		{name: "credentials", raw: "https://user@publisher.example/notices", want: false},
		{name: "fragment", raw: "https://publisher.example/notices#section", want: false},
		{name: "ip literal", raw: "https://203.0.113.10/notices", want: false},
		{name: "wildcard-like hostname", raw: "https://*.publisher.example/notices", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetcher.validateURL(tt.raw)
			if tt.want {
				if err != nil {
					t.Fatalf("validateURL(%q) error = %v", tt.raw, err)
				}
				if got.Hostname() != "publisher.example" && got.Hostname() != "PUBLISHER.EXAMPLE" {
					t.Fatalf("validateURL(%q) hostname = %q", tt.raw, got.Hostname())
				}
				return
			}
			if err == nil {
				t.Fatalf("validateURL(%q) succeeded with URL %q", tt.raw, got)
			}
		})
	}
}

func TestNewFetcherRejectsNonHostAllowlistEntries(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{name: "empty", host: ""},
		{name: "single label", host: "localhost"},
		{name: "ip literal", host: "8.8.8.8"},
		{name: "wildcard", host: "*.example.com"},
		{name: "leading dash", host: "-publisher.example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewFetcher([]string{tt.host}); err == nil {
				t.Fatalf("NewFetcher(%q) succeeded", tt.host)
			}
		})
	}
}

func TestFetcherFetchFailsClosedForZeroValue(t *testing.T) {
	var fetcher Fetcher
	_, err := fetcher.Fetch(context.Background(), "https://publisher.example/notices")
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("zero-value Fetch() error = %v, want not configured", err)
	}
}

func TestFetcherFetchReturnsAlreadyCancelledContextWithoutNetwork(t *testing.T) {
	fetcher, err := NewFetcher([]string{"publisher.example"})
	if err != nil {
		t.Fatalf("NewFetcher() error = %v", err)
	}
	resolverCalls := 0
	dialCalls := 0
	fetcher.resolver = fakeResolver{lookup: func(context.Context, string, string) ([]netip.Addr, error) {
		resolverCalls++
		return nil, errors.New("DNS must not be reached for a cancelled request")
	}}
	fetcher.dial = func(context.Context, string, string) (net.Conn, error) {
		dialCalls++
		return nil, errors.New("dial must not be reached for a cancelled request")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = fetcher.Fetch(ctx, "https://publisher.example/notices")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
	if resolverCalls != 0 || dialCalls != 0 {
		t.Fatalf("cancelled Fetch() used network: resolver calls = %d, dial calls = %d", resolverCalls, dialCalls)
	}
}

func TestFetcherCheckRedirect(t *testing.T) {
	fetcher, err := NewFetcher([]string{"publisher.example", "alternate.example"})
	if err != nil {
		t.Fatalf("NewFetcher() error = %v", err)
	}
	sensitiveReferer := "https://publisher.example/notices?access_token=do-not-disclose"

	tests := []struct {
		name       string
		target     string
		viaCount   int
		wantErr    bool
		wantErrMsg string
	}{
		{name: "approved cross host clears Referer", target: "https://alternate.example/notices/2", viaCount: 1},
		{name: "unapproved host clears Referer before rejection", target: "https://attacker.invalid/notices", viaCount: 1, wantErr: true},
		{name: "HTTP target clears Referer before rejection", target: "http://alternate.example/notices", viaCount: 1, wantErr: true},
		{name: "redirect limit clears Referer before rejection", target: "https://alternate.example/notices", viaCount: 5, wantErr: true, wantErrMsg: "redirect limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL:    mustURL(t, tt.target),
				Header: http.Header{"Referer": []string{sensitiveReferer}},
			}
			via := make([]*http.Request, tt.viaCount)
			for i := range via {
				via[i] = &http.Request{URL: mustURL(t, "https://publisher.example/notices")}
			}

			err := fetcher.checkRedirect(req, via)
			if tt.wantErr && err == nil {
				t.Fatal("checkRedirect() succeeded, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("checkRedirect() error = %v", err)
			}
			if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Fatalf("checkRedirect() error = %v, want containing %q", err, tt.wantErrMsg)
			}
			if got := req.Header.Get("Referer"); got != "" {
				t.Fatalf("redirect Referer = %q, want removed", got)
			}
		})
	}
}

func TestPublicAddress(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "8.8.8.8", want: true},
		{name: "private IPv4", ip: "10.20.30.40", want: false},
		{name: "loopback IPv4", ip: "127.0.0.1", want: false},
		{name: "link local IPv4", ip: "169.254.1.1", want: false},
		{name: "this network IPv4", ip: "0.1.2.3", want: false},
		{name: "shared address space", ip: "100.64.0.1", want: false},
		{name: "protocol assignment IPv4", ip: "192.0.0.1", want: false},
		{name: "documentation IPv4", ip: "192.0.2.1", want: false},
		{name: "deprecated relay anycast IPv4", ip: "192.88.99.1", want: false},
		{name: "benchmark IPv4", ip: "198.18.0.1", want: false},
		{name: "documentation two IPv4", ip: "198.51.100.1", want: false},
		{name: "documentation three IPv4", ip: "203.0.113.1", want: false},
		{name: "reserved IPv4", ip: "240.0.0.1", want: false},
		{name: "multicast IPv4", ip: "224.0.0.1", want: false},
		{name: "public IPv4 mapped IPv6", ip: "::ffff:8.8.8.8", want: true},
		{name: "private IPv4 mapped IPv6", ip: "::ffff:10.0.0.1", want: false},
		{name: "public IPv6", ip: "2606:4700:4700::1111", want: true},
		{name: "private IPv6", ip: "fd00::1", want: false},
		{name: "loopback IPv6", ip: "::1", want: false},
		{name: "link local IPv6", ip: "fe80::1", want: false},
		{name: "documentation IPv6", ip: "2001:db8::1", want: false},
		{name: "translation IPv6", ip: "2001::1", want: false},
		{name: "6to4 IPv6", ip: "2002::1", want: false},
		{name: "special use IPv6", ip: "3fff::1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := netip.MustParseAddr(tt.ip)
			if got := publicAddress(ip); got != tt.want {
				t.Errorf("publicAddress(%s) = %t, want %t", ip, got, tt.want)
			}
		})
	}
}

func TestFetcherDialPublisherDeniesMixedDNSWithoutDial(t *testing.T) {
	dialCalls := 0
	fetcher := &Fetcher{
		hosts: map[string]struct{}{"publisher.example": {}},
		resolver: fakeResolver{lookup: func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")}, nil
		}},
		dial: func(context.Context, string, string) (net.Conn, error) {
			dialCalls++
			return nil, errors.New("must not dial mixed DNS results")
		},
	}

	_, err := fetcher.dialPublisher(context.Background(), "tcp", "publisher.example:443")
	if err == nil || !strings.Contains(err.Error(), "non-public") {
		t.Fatalf("dialPublisher() error = %v, want non-public DNS denial", err)
	}
	if dialCalls != 0 {
		t.Fatalf("dial called %d times, want 0", dialCalls)
	}
}

func TestFetcherDialPublisherPinsValidatedNumericAddress(t *testing.T) {
	lookupCalls := 0
	var dialedAddress string
	fetcher := &Fetcher{
		hosts: map[string]struct{}{"publisher.example": {}},
		resolver: fakeResolver{lookup: func(_ context.Context, network, host string) ([]netip.Addr, error) {
			lookupCalls++
			if network != "ip" || host != "publisher.example" {
				t.Fatalf("LookupNetIP(%q, %q), want (ip, publisher.example)", network, host)
			}
			if lookupCalls > 1 {
				return []netip.Addr{netip.MustParseAddr("10.0.0.1")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}},
		dial: func(_ context.Context, network, address string) (net.Conn, error) {
			if network != "tcp" {
				t.Fatalf("dial network = %q, want tcp", network)
			}
			dialedAddress = address
			client, server := net.Pipe()
			go server.Close()
			return client, nil
		},
	}

	conn, err := fetcher.dialPublisher(context.Background(), "tcp", "publisher.example:443")
	if err != nil {
		t.Fatalf("dialPublisher() error = %v", err)
	}
	defer conn.Close()
	if lookupCalls != 1 {
		t.Fatalf("DNS lookup calls = %d, want 1; hostname must not be resolved again before dialing", lookupCalls)
	}
	if dialedAddress != "8.8.8.8:443" {
		t.Fatalf("dial address = %q, want validated numeric address %q", dialedAddress, "8.8.8.8:443")
	}
}

func TestReadEvidence(t *testing.T) {
	retrievedAt := time.Date(2026, time.September, 12, 10, 0, 0, 0, time.UTC)
	requestURL := mustURL(t, "https://publisher.example/notices/1")

	tests := []struct {
		name          string
		body          io.Reader
		contentLength int64
		wantErr       string
		wantContent   []byte
	}{
		{name: "exact bytes and SHA256", body: bytes.NewBufferString("Regulation \x00 v1\n"), contentLength: -1, wantContent: []byte("Regulation \x00 v1\n")},
		{name: "empty", body: strings.NewReader(""), contentLength: 0, wantErr: "empty"},
		{name: "known oversized content length", body: strings.NewReader("small"), contentLength: maxEvidenceBytes + 1, wantErr: "exceeds size limit"},
		{name: "unknown oversized content length", body: bytes.NewReader(bytes.Repeat([]byte{'x'}, int(maxEvidenceBytes+1))), contentLength: -1, wantErr: "empty or exceeds size limit"},
		{name: "read error", body: errorReader{err: errors.New("source interrupted")}, contentLength: -1, wantErr: "read failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Body:          io.NopCloser(tt.body),
				ContentLength: tt.contentLength,
				Header:        http.Header{"Content-Type": []string{"application/pdf"}},
				Request:       &http.Request{URL: requestURL},
			}
			got, err := readEvidence(resp, retrievedAt)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("readEvidence() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("readEvidence() error = %v", err)
			}
			if !bytes.Equal(got.Content, tt.wantContent) {
				t.Fatalf("content = %q, want %q", got.Content, tt.wantContent)
			}
			const wantSHA256 = "cbec0030868223b0f0fa495f8d06865ecfacec383b2383115017fd44d5360bfa"
			if got.SHA256 != wantSHA256 {
				t.Fatalf("SHA256 = %q, want %q", got.SHA256, wantSHA256)
			}
			if got.URL != requestURL.String() || got.ContentType != "application/pdf" || !got.RetrievedAt.Equal(retrievedAt) {
				t.Fatalf("metadata = %#v, want URL %q, content type application/pdf, retrieved at %s", got, requestURL, retrievedAt)
			}
		})
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}
