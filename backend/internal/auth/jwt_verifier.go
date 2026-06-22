package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

type AccessTokenClaims struct {
	Subject       string            `json:"sub"`
	Issuer        string            `json:"iss"`
	Audience      []string          `json:"aud"`
	ExpiresAt     int64             `json:"exp"`
	IssuedAt      int64             `json:"iat"`
	Email         string            `json:"email,omitempty"`
	UserUUID      string            `json:"user_id,omitempty"`
	TenantID      string            `json:"tenant_id,omitempty"`
	InstitutionID string            `json:"institution_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	ActorType     string            `json:"actor_type,omitempty"`
	Roles         []string          `json:"roles,omitempty"`
	Cnf           ConfirmationClaim `json:"cnf,omitempty"`
}

type ConfirmationClaim struct {
	JKT string `json:"jkt,omitempty"`
}

type AccessTokenScheme string

const (
	AccessTokenBearer AccessTokenScheme = "Bearer"
	AccessTokenDPoP   AccessTokenScheme = "DPoP"
)

type JWKSLoader func(context.Context) (*jose.JSONWebKeySet, error)

type JWTVerifier struct {
	issuer   string
	jwksURL  string
	audience string
	loader   JWKSLoader

	mu        sync.RWMutex
	keySet    *jose.JSONWebKeySet
	fetchedAt time.Time
	ttl       time.Duration
}

func NewJWTVerifier(issuer, jwksURL string, audience string) *JWTVerifier {
	return &JWTVerifier{
		issuer:   issuer,
		jwksURL:  jwksURL,
		audience: audience,
		ttl:      5 * time.Minute,
	}
}

func (v *JWTVerifier) Verify(ctx context.Context, rawToken string) (*AccessTokenClaims, error) {
	keySet, err := v.getKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("jwt: load JWKS: %w", err)
	}

	tok, err := josejwt.ParseSigned(rawToken, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return nil, fmt.Errorf("jwt: parse: %w", err)
	}
	if len(tok.Headers) == 0 {
		return nil, errors.New("jwt: no headers")
	}

	kid := tok.Headers[0].KeyID
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		v.invalidate()
		keySet, err = v.getKeySet(ctx)
		if err != nil {
			return nil, fmt.Errorf("jwt: refresh JWKS: %w", err)
		}
		keys = keySet.Key(kid)
		if len(keys) == 0 {
			return nil, fmt.Errorf("jwt: unknown key id %q", kid)
		}
	}

	var claims AccessTokenClaims
	if err := tok.Claims(keys[0].Key, &claims); err != nil {
		return nil, fmt.Errorf("jwt: verify claims: %w", err)
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, errors.New("jwt: token expired")
	}
	if claims.Issuer != v.issuer {
		return nil, fmt.Errorf("jwt: unexpected issuer %q", claims.Issuer)
	}
	if v.audience != "" {
		found := false
		for _, item := range claims.Audience {
			if item == v.audience {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("jwt: token audience %v does not contain expected %q", claims.Audience, v.audience)
		}
	}

	return &claims, nil
}

func ExtractAccessToken(r *http.Request) (string, AccessTokenScheme, bool) {
	value := r.Header.Get("Authorization")
	if strings.HasPrefix(value, "Bearer ") {
		return strings.TrimPrefix(value, "Bearer "), AccessTokenBearer, true
	}
	if strings.HasPrefix(value, "DPoP ") {
		return strings.TrimPrefix(value, "DPoP "), AccessTokenDPoP, true
	}
	return "", "", false
}

func (v *JWTVerifier) getKeySet(ctx context.Context) (*jose.JSONWebKeySet, error) {
	v.mu.RLock()
	keySet := v.keySet
	fresh := time.Since(v.fetchedAt) < v.ttl
	v.mu.RUnlock()
	if keySet != nil && fresh {
		return keySet, nil
	}
	return v.fetch(ctx)
}

func (v *JWTVerifier) fetch(ctx context.Context) (*jose.JSONWebKeySet, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.keySet != nil && time.Since(v.fetchedAt) < v.ttl {
		return v.keySet, nil
	}

	if v.loader != nil {
		keySet, err := v.loader(ctx)
		if err != nil {
			return nil, err
		}
		if keySet == nil || len(keySet.Keys) == 0 {
			return nil, errors.New("jwt: empty JWKS")
		}
		v.keySet = keySet
		v.fetchedAt = time.Now()
		return keySet, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", v.jwksURL, err)
	}
	defer resp.Body.Close()

	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(keySet.Keys) == 0 {
		return nil, errors.New("jwt: empty JWKS")
	}

	v.keySet = &keySet
	v.fetchedAt = time.Now()
	return &keySet, nil
}

func (v *JWTVerifier) invalidate() {
	v.mu.Lock()
	v.fetchedAt = time.Time{}
	v.mu.Unlock()
}
