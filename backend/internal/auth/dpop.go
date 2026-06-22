package auth

import (
	gocrypto "crypto"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	if accessToken != "" && claims.ATH != "" {
		expected := accessTokenHash(accessToken)
		if claims.ATH != expected {
			return nil, errors.New("dpop: ath mismatch")
		}
	}

	tp, err := jwk.Thumbprint(gocrypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("dpop: thumbprint: %w", err)
	}

	return &DPoPProof{
		PublicKey:  jwk,
		Thumbprint: base64.RawURLEncoding.EncodeToString(tp),
	}, nil
}

func WriteDPoPNonce(w http.ResponseWriter) {
	w.Header().Set(headerNonce, base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().Unix()))))
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
