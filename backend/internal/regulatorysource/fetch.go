// Package regulatorysource retrieves publisher content without trusting client-provided hashes.
package regulatorysource

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const maxEvidenceBytes int64 = 20 << 20

// Evidence proves retrieval of bytes, not their legal applicability or authenticity of a signature.
// The caller must persist Content and its metadata atomically with the unchanged source version.
type Evidence struct {
	Content     []byte
	SHA256      string
	URL         string
	ContentType string
	RetrievedAt time.Time
}

type resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// Fetcher permits exact, operator-configured publisher hosts; wildcards and IP literals are rejected.
// Its zero value fails closed. Do not construct its configuration from browser input.
type Fetcher struct {
	hosts    map[string]struct{}
	resolver resolver
	dial     func(context.Context, string, string) (net.Conn, error)
}

func NewFetcher(publisherHosts []string) (*Fetcher, error) {
	f := &Fetcher{
		hosts:    make(map[string]struct{}, len(publisherHosts)),
		resolver: net.DefaultResolver,
		dial:     (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
	for _, host := range publisherHosts {
		host = strings.ToLower(host)
		if !validHostname(host) {
			return nil, errors.New("invalid publisher hostname")
		}
		f.hosts[host] = struct{}{}
	}
	if len(f.hosts) == 0 {
		return nil, errors.New("publisher hosts required")
	}
	return f, nil
}

func validHostname(host string) bool {
	if len(host) > 253 || !strings.Contains(host, ".") {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

func (f *Fetcher) validateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.New("invalid publisher URL")
	}
	invalid := u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Opaque != ""
	invalidPort := u.Port() != "" && u.Port() != "443"
	if invalid || invalidPort || !validHostname(strings.ToLower(u.Hostname())) {
		return nil, errors.New("publisher URL must use an approved HTTPS host on port 443")
	}
	if _, allowed := f.hosts[strings.ToLower(u.Hostname())]; !allowed {
		return nil, errors.New("publisher host is not approved")
	}
	return u, nil
}

// Fetch uses a dedicated transport: no environment proxy, cookies, credentials or inherited TLS settings.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (Evidence, error) {
	if f == nil || f.resolver == nil || f.dial == nil {
		return Evidence{}, errors.New("publisher fetcher is not configured")
	}
	u, err := f.validateURL(rawURL)
	if err != nil {
		return Evidence{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	transport := &http.Transport{
		DialContext:            f.dialPublisher,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
		DisableCompression:     true,
		DisableKeepAlives:      true,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport:     transport,
		CheckRedirect: f.checkRedirect,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Evidence{}, errors.New("cannot construct publisher request")
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Evidence{}, ctx.Err()
		}
		// Do not include a publisher URL query (potential credentials) in the public error.
		return Evidence{}, errors.New("publisher retrieval failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Evidence{}, fmt.Errorf("publisher returned HTTP %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Encoding") != "" && resp.Header.Get("Content-Encoding") != "identity" {
		return Evidence{}, errors.New("encoded publisher content is not supported")
	}
	evidence, err := readEvidence(resp, time.Now().UTC())
	if ctx.Err() != nil {
		return Evidence{}, ctx.Err()
	}
	return evidence, err
}

func (f *Fetcher) checkRedirect(req *http.Request, via []*http.Request) error {
	// net/http synthesizes Referer before invoking this hook. Never disclose
	// the previous URL's path/query to another publisher, even an approved one.
	req.Header.Del("Referer")
	if len(via) >= 5 {
		return errors.New("publisher redirect limit exceeded")
	}
	_, err := f.validateURL(req.URL.String())
	return err
}

func readEvidence(resp *http.Response, retrievedAt time.Time) (Evidence, error) {
	if resp.ContentLength > maxEvidenceBytes {
		return Evidence{}, errors.New("publisher content exceeds size limit")
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxEvidenceBytes+1))
	if err != nil {
		return Evidence{}, errors.New("publisher content read failed")
	}
	if len(content) == 0 || int64(len(content)) > maxEvidenceBytes {
		return Evidence{}, errors.New("publisher content is empty or exceeds size limit")
	}
	digest := sha256.Sum256(content)
	return Evidence{
		Content:     content,
		SHA256:      hex.EncodeToString(digest[:]),
		URL:         resp.Request.URL.String(),
		ContentType: resp.Header.Get("Content-Type"),
		RetrievedAt: retrievedAt,
	}, nil
}

func (f *Fetcher) dialPublisher(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" {
		return nil, errors.New("publisher connection target denied")
	}
	if _, allowed := f.hosts[strings.ToLower(host)]; !allowed {
		return nil, errors.New("publisher connection host denied")
	}
	ips, err := f.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("publisher DNS resolution failed")
	}
	for _, ip := range ips {
		if !publicAddress(ip) {
			return nil, errors.New("publisher DNS contains a non-public address")
		}
	}
	// Dial the validated numeric address, never resolve the hostname a second time.
	// TLS still verifies the original request hostname through net/http.
	for _, ip := range ips {
		conn, err := f.dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, errors.New("publisher connection failed")
}

var excludedNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	// Restrict IPv6 to ordinary global unicast, excluding translation and transition ranges.
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, prefix := range excludedNetworks {
		if prefix.Contains(ip) {
			return false
		}
	}
	return ip.Zone() == ""
}
