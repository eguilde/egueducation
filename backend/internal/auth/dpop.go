package auth

import (
	gocrypto "crypto"
	gocryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

const (
	headerDPoP   = "DPoP"
	headerNonce  = "DPoP-Nonce"
	maxClockSkew = 60 * time.Second
	proofMaxAge  = 5 * time.Minute
)

type DPoPProof struct {
	PublicKey  *jose.JSONWebKey
	Thumbprint string
}

var dpopReplayCache = struct {
	sync.Mutex
	seen map[string]time.Time
}{seen: make(map[string]time.Time)}

func VerifyDPoPProof(r *http.Request, accessToken string) (*DPoPProof, error) {
	rawProof := r.Header.Get(headerDPoP)
	if rawProof == "" {
		return nil, errors.New("dpop: missing DPoP header")
	}

	tok, err := josejwt.ParseSigned(rawProof, []jose.SignatureAlgorithm{
		jose.RS256, jose.ES256, jose.ES384, jose.PS256,
	})
	if err != nil {
		return nil, fmt.Errorf("dpop: parse: %w", err)
	}
	if len(tok.Headers) == 0 {
		return nil, errors.New("dpop: no headers")
	}

	header := tok.Headers[0]
	if header.KeyID != "" {
		return nil, errors.New("dpop: kid must not be present")
	}
	if header.ExtraHeaders["typ"] != "dpop+jwt" {
		return nil, errors.New("dpop: typ must be dpop+jwt")
	}

	jwk := header.JSONWebKey
	if jwk == nil {
		return nil, errors.New("dpop: missing jwk")
	}
	if !jwk.IsPublic() {
		return nil, errors.New("dpop: jwk must be public key")
	}

	var claims struct {
		HTTPMethod string `json:"htm"`
		HTTPURL    string `json:"htu"`
		IssuedAt   int64  `json:"iat"`
		ATH        string `json:"ath,omitempty"`
		JTI        string `json:"jti"`
	}
	if err := tok.Claims(jwk.Key, &claims); err != nil {
		return nil, fmt.Errorf("dpop: verify signature: %w", err)
	}

	if claims.HTTPMethod != r.Method {
		return nil, fmt.Errorf("dpop: htm mismatch: got %q want %q", claims.HTTPMethod, r.Method)
	}
	if claims.HTTPURL != dpopRequestURL(r) {
		return nil, fmt.Errorf("dpop: htu mismatch: got %q want %q", claims.HTTPURL, dpopRequestURL(r))
	}

	now := time.Now()
	issued := time.Unix(claims.IssuedAt, 0)
	if now.Before(issued.Add(-maxClockSkew)) || now.After(issued.Add(proofMaxAge)) {
		return nil, errors.New("dpop: proof expired or not yet valid")
	}
	if strings.TrimSpace(claims.JTI) == "" {
		return nil, errors.New("dpop: missing jti")
	}
	if accessToken != "" {
		if claims.ATH == "" {
			return nil, errors.New("dpop: missing ath")
		}
		expected := accessTokenHash(accessToken)
		if claims.ATH != expected {
			return nil, errors.New("dpop: ath mismatch")
		}
	}

	tp, err := jwk.Thumbprint(gocrypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("dpop: thumbprint: %w", err)
	}
	thumbprint := base64.RawURLEncoding.EncodeToString(tp)
	if !rememberDPoPProof(thumbprint+":"+claims.JTI, now) {
		return nil, errors.New("dpop: proof replayed")
	}

	return &DPoPProof{
		PublicKey:  jwk,
		Thumbprint: thumbprint,
	}, nil
}

func rememberDPoPProof(key string, now time.Time) bool {
	dpopReplayCache.Lock()
	defer dpopReplayCache.Unlock()
	for existing, expiresAt := range dpopReplayCache.seen {
		if !expiresAt.After(now) {
			delete(dpopReplayCache.seen, existing)
		}
	}
	if expiresAt, exists := dpopReplayCache.seen[key]; exists && expiresAt.After(now) {
		return false
	}
	dpopReplayCache.seen[key] = now.Add(proofMaxAge + maxClockSkew)
	return true
}

func WriteDPoPNonce(w http.ResponseWriter) {
	random := make([]byte, 32)
	if _, err := gocryptoRand.Read(random); err == nil {
		w.Header().Set(headerNonce, base64.RawURLEncoding.EncodeToString(random))
	}
}

func accessTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func dpopRequestURL(r *http.Request) string {
	scheme := "https"
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		if index := strings.Index(proto, ","); index >= 0 {
			proto = proto[:index]
		}
		scheme = strings.TrimSpace(proto)
	} else if r.TLS == nil {
		scheme = "http"
	}

	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		if index := strings.Index(forwardedHost, ","); index >= 0 {
			forwardedHost = forwardedHost[:index]
		}
		host = strings.TrimSpace(forwardedHost)
	}

	path := r.URL.Path
	if forwardedURI := strings.TrimSpace(r.Header.Get("X-Forwarded-Uri")); forwardedURI != "" {
		if index := strings.IndexByte(forwardedURI, '?'); index >= 0 {
			forwardedURI = forwardedURI[:index]
		}
		if forwardedURI != "" {
			path = forwardedURI
		}
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, path)
}
