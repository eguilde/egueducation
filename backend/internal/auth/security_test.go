package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

func TestJWTVerifierRejectsIDTokenAndWrongAudience(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewJWTVerifier("https://issuer.example", "", "api")
	verifier.loader = func(context.Context) (*jose.JSONWebKeySet, error) {
		return &jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{KeyID: "test", Key: &key.PublicKey}}}, nil
	}

	makeToken := func(tokenUse string, audience []string) string {
		t.Helper()
		raw, err := josejwt.Signed(signer).Claims(map[string]any{
			"sub": "user", "iss": "https://issuer.example", "aud": audience,
			"iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Minute).Unix(),
			"token_use": tokenUse,
		}).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}

	if _, err := verifier.Verify(context.Background(), makeToken("id", []string{"api"})); err == nil {
		t.Fatal("ID token must not be accepted as an API access token")
	}
	if _, err := verifier.Verify(context.Background(), makeToken("access", []string{"different"})); err == nil {
		t.Fatal("wrong-audience access token must be rejected")
	}
	if _, err := verifier.Verify(context.Background(), makeToken("access", []string{"api"})); err != nil {
		t.Fatalf("valid access token rejected: %v", err)
	}
}

func TestDPoPReplayCacheRejectsReuse(t *testing.T) {
	key := "thumbprint:jti"
	now := time.Now()
	if !rememberDPoPProof(key, now) {
		t.Fatal("first proof use should be accepted")
	}
	if rememberDPoPProof(key, now.Add(time.Second)) {
		t.Fatal("replayed proof should be rejected")
	}
}

func TestAllowedLogoutReturnToRequiresFrontendOrigin(t *testing.T) {
	frontend := "https://school.example/app"
	if !allowedLogoutReturnTo("https://school.example/signed-out", frontend) {
		t.Fatal("same-origin return URL should be allowed")
	}
	for _, candidate := range []string{"https://evil.example/", "//evil.example/", "javascript:alert(1)", ""} {
		if allowedLogoutReturnTo(candidate, frontend) {
			t.Fatalf("unsafe return URL accepted: %q", candidate)
		}
	}
}
