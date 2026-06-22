package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const dpopClockSkew = 5 * time.Minute

type dpopJWTHeader struct {
	Alg string       `json:"alg"`
	Typ string       `json:"typ"`
	JWK *dpopJWK     `json:"jwk"`
	Kid string       `json:"kid,omitempty"`
}

type dpopJWTClaims struct {
	HTM string `json:"htm"`
	HTU string `json:"htu"`
	JTI string `json:"jti"`
	IAT int64  `json:"iat"`
	ATH string `json:"ath,omitempty"`
}

type dpopJWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func verifyDPoPProof(r *http.Request, accessToken string) (string, error) {
	rawProof := strings.TrimSpace(r.Header.Get("DPoP"))
	if rawProof == "" {
		return "", errors.New("dpop_proof_missing")
	}

	parts := strings.Split(rawProof, ".")
	if len(parts) != 3 {
		return "", errors.New("dpop_proof_invalid")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("dpop_proof_invalid")
	}
	var header dpopJWTHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", errors.New("dpop_proof_invalid")
	}
	if !strings.EqualFold(header.Typ, "dpop+jwt") || !strings.EqualFold(header.Alg, "ES256") || header.JWK == nil || header.Kid != "" {
		return "", errors.New("dpop_proof_invalid")
	}

	pub, err := dpopPublicKey(header.JWK)
	if err != nil {
		return "", errors.New("dpop_proof_invalid")
	}

	signingInput := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signingInput))
	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signatureBytes) != 64 {
		return "", errors.New("dpop_proof_invalid")
	}
	rValue := new(big.Int).SetBytes(signatureBytes[:32])
	sValue := new(big.Int).SetBytes(signatureBytes[32:])
	if !ecdsa.Verify(pub, sum[:], rValue, sValue) {
		return "", errors.New("dpop_proof_invalid")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("dpop_proof_invalid")
	}
	var claims dpopJWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", errors.New("dpop_proof_invalid")
	}
	if claims.JTI == "" || claims.IAT == 0 || claims.HTM == "" || claims.HTU == "" {
		return "", errors.New("dpop_proof_invalid")
	}
	if !strings.EqualFold(claims.HTM, r.Method) {
		return "", errors.New("dpop_proof_invalid")
	}
	if !dpopURLMatches(claims.HTU, requestURLForDPoP(r)) {
		return "", errors.New("dpop_proof_invalid")
	}
	now := time.Now()
	iat := time.Unix(claims.IAT, 0)
	if iat.After(now.Add(dpopClockSkew)) || iat.Before(now.Add(-dpopClockSkew)) {
		return "", errors.New("dpop_proof_invalid")
	}
	if accessToken != "" {
		if claims.ATH == "" || claims.ATH != dpopAccessTokenHash(accessToken) {
			return "", errors.New("dpop_proof_invalid")
		}
	}

	thumbprint, err := dpopJWKThumbprint(header.JWK)
	if err != nil {
		return "", errors.New("dpop_proof_invalid")
	}
	return thumbprint, nil
}

func dpopPublicKey(jwk *dpopJWK) (*ecdsa.PublicKey, error) {
	if jwk == nil || jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, errors.New("unsupported_jwk")
	}
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, err
	}
	if len(xBytes) == 0 || len(yBytes) == 0 {
		return nil, errors.New("unsupported_jwk")
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func dpopJWKThumbprint(jwk *dpopJWK) (string, error) {
	if jwk == nil {
		return "", errors.New("unsupported_jwk")
	}
	canonical := fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`, jwk.Crv, jwk.Kty, jwk.X, jwk.Y)
	sum := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func dpopAccessTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func dpopURLMatches(expected, actual string) bool {
	exp, err := url.Parse(expected)
	if err != nil {
		return false
	}
	act, err := url.Parse(actual)
	if err != nil {
		return false
	}
	if !strings.EqualFold(exp.Scheme, act.Scheme) {
		return false
	}
	if !strings.EqualFold(exp.Host, act.Host) {
		return false
	}
	if exp.Path != act.Path {
		return false
	}
	return exp.RawQuery == act.RawQuery
}

func requestURLForDPoP(r *http.Request) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	return (&url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}).String()
}

func tokenThumbprint(claims map[string]any) (string, bool) {
	cnf, ok := claims["cnf"].(map[string]any)
	if !ok {
		return "", false
	}
	thumbprint, _ := cnf["jkt"].(string)
	if thumbprint == "" {
		return "", false
	}
	return thumbprint, true
}
