package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	loginSessionTTL     = 8 * time.Hour
	passkeyChallengeTTL = 5 * time.Minute
)

type oidcKeyPair struct {
	privateKey *rsa.PrivateKey
	keyID      string
	modulus    string
	exponent   string
}

type signedSessionClaims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	ExpiresAt int64  `json:"exp"`
}

type passkeyClientData struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}

type passkeyAssertionData struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}

type passkeyPublicKey struct {
	KeyType  string `json:"kty"`
	Alg      int    `json:"alg"`
	Curve    int    `json:"crv,omitempty"`
	X        string `json:"x,omitempty"`
	Y        string `json:"y,omitempty"`
	Modulus  string `json:"n,omitempty"`
	Exponent string `json:"e,omitempty"`
}

type passkeyStoredCredential struct {
	Challenge string            `json:"challenge,omitempty"`
	PublicKey *passkeyPublicKey  `json:"public_key,omitempty"`
	SignCount uint32            `json:"sign_count,omitempty"`
}

type passkeyRecord struct {
	Subject    string
	UserID     string
	DeviceName string
	PublicKey  *passkeyPublicKey
	SignCount  uint32
}

var activeOIDCKeyPair atomic.Pointer[oidcKeyPair]

func setActiveOIDCKeyPair(keyPair *oidcKeyPair) {
	activeOIDCKeyPair.Store(keyPair)
}

func activeSessionKeyPair() *oidcKeyPair {
	return activeOIDCKeyPair.Load()
}

func loadOrCreateOIDCKeyPair(ctx context.Context, db *pgxpool.Pool) (*oidcKeyPair, error) {
	var (
		keyID      string
		privatePEM string
	)
	err := db.QueryRow(ctx, `
		select key_id, private_key_pem
		from oidc_signing_keys
		where active = true
		order by created_at desc
		limit 1
	`).Scan(&keyID, &privatePEM)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("load oidc signing key: %w", err)
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generate oidc signing key: %w", err)
		}
		keyID = keyIDFromPublicKey(&privateKey.PublicKey)
		privatePEM, err = marshalPrivateKeyToPEM(privateKey)
		if err != nil {
			return nil, err
		}
		if _, err := db.Exec(ctx, `
			insert into oidc_signing_keys (key_id, private_key_pem, active, created_at, updated_at)
			values ($1, $2, true, now(), now())
			on conflict (key_id) do update
			set private_key_pem = excluded.private_key_pem,
				active = true,
				updated_at = now()
		`, keyID, privatePEM); err != nil {
			return nil, fmt.Errorf("store oidc signing key: %w", err)
		}
		return newOIDCKeyPairFromPEM(keyID, privatePEM)
	}

	return newOIDCKeyPairFromPEM(keyID, privatePEM)
}

func newOIDCKeyPairFromPEM(keyID, privatePEM string) (*oidcKeyPair, error) {
	block, _ := pem.Decode([]byte(privatePEM))
	if block == nil {
		return nil, errors.New("invalid oidc signing key pem")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse oidc signing key: %w", err)
	}

	return newOIDCKeyPairFromPrivateKey(keyID, privateKey), nil
}

func newOIDCKeyPairFromPrivateKey(keyID string, privateKey *rsa.PrivateKey) *oidcKeyPair {
	return &oidcKeyPair{
		privateKey: privateKey,
		keyID:      keyID,
		modulus:    base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
		exponent:   base64.RawURLEncoding.EncodeToString(bigIntToBytes(int64(privateKey.PublicKey.E))),
	}
}

func keyIDFromPublicKey(publicKey *rsa.PublicKey) string {
	sum := sha256.Sum256(publicKey.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}

func marshalPrivateKeyToPEM(privateKey *rsa.PrivateKey) (string, error) {
	der := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

func bigIntToBytes(value int64) []byte {
	return new(big.Int).SetInt64(value).Bytes()
}

func signSessionClaims(claims signedSessionClaims) (string, error) {
	keyPair := activeSessionKeyPair()
	if keyPair == nil {
		return "", errors.New("oidc_key_unavailable")
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal session claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, keyPair.privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("sign session claims: %w", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func verifySessionClaims(token string) (signedSessionClaims, error) {
	keyPair := activeSessionKeyPair()
	if keyPair == nil {
		return signedSessionClaims{}, errors.New("oidc_key_unavailable")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}
	sum := sha256.Sum256([]byte(parts[0]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}
	if err := rsa.VerifyPKCS1v15(&keyPair.privateKey.PublicKey, crypto.SHA256, sum[:], signature); err != nil {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}

	var claims signedSessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}
	if claims.Subject == "" || claims.SessionID == "" || claims.ExpiresAt <= time.Now().Unix() {
		return signedSessionClaims{}, errors.New("invalid_session_cookie")
	}
	return claims, nil
}

func normalizeOrigin(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return strings.TrimSpace(value)
	}
	return parsed.Scheme + "://" + parsed.Host
}

func parsePasskeyClientData(encoded string) (passkeyClientData, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return passkeyClientData{}, fmt.Errorf("decode client data: %w", err)
	}

	var data passkeyClientData
	if err := json.Unmarshal(raw, &data); err != nil {
		return passkeyClientData{}, fmt.Errorf("parse client data: %w", err)
	}
	return data, nil
}

func parsePasskeyAssertionData(encoded string) (passkeyAssertionData, error) {
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return passkeyAssertionData{}, fmt.Errorf("decode client data: %w", err)
	}

	var data passkeyAssertionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return passkeyAssertionData{}, fmt.Errorf("parse client data: %w", err)
	}
	return data, nil
}

func decodePasskeyPayload(encoded string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
}

func passkeyRPID(frontendOrigin string) string {
	parsed, err := url.Parse(strings.TrimSpace(frontendOrigin))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(frontendOrigin)
	}
	return parsed.Hostname()
}

func passkeyResponseString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func passkeyAuthenticatorVerified(authenticatorData string, rpID string) (bool, error) {
	_, err := parsePasskeyAuthenticatorData(authenticatorData, rpID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func parsePasskeyAuthenticatorData(authenticatorData string, rpID string) (uint32, error) {
	data, err := decodePasskeyPayload(authenticatorData)
	if err != nil {
		return 0, fmt.Errorf("decode authenticator data: %w", err)
	}
	if len(data) < 37 {
		return 0, errors.New("authenticator_data_invalid")
	}
	expectedHash := sha256.Sum256([]byte(rpID))
	if !bytes.Equal(data[:32], expectedHash[:]) {
		return 0, errors.New("authenticator_rp_mismatch")
	}
	flags := data[32]
	if flags&0x01 == 0 {
		return 0, errors.New("authenticator_user_not_present")
	}
	if flags&0x04 == 0 {
		return 0, errors.New("authenticator_user_not_verified")
	}
	return binary.BigEndian.Uint32(data[33:37]), nil
}

type cborItem struct {
	Key   any
	Value any
}

func parseCBOR(encoded []byte) (any, int, error) {
	return parseCBORAt(encoded, 0)
}

func parseCBORAt(encoded []byte, offset int) (any, int, error) {
	if offset >= len(encoded) {
		return nil, offset, errors.New("cbor_unexpected_eof")
	}

	major := encoded[offset] >> 5
	additional := encoded[offset] & 0x1f
	offset++

	readLength := func() (uint64, error) {
		switch {
		case additional <= 23:
			return uint64(additional), nil
		case additional == 24:
			if offset >= len(encoded) {
				return 0, errors.New("cbor_unexpected_eof")
			}
			value := uint64(encoded[offset])
			offset++
			return value, nil
		case additional == 25:
			if offset+2 > len(encoded) {
				return 0, errors.New("cbor_unexpected_eof")
			}
			value := uint64(binary.BigEndian.Uint16(encoded[offset:]))
			offset += 2
			return value, nil
		case additional == 26:
			if offset+4 > len(encoded) {
				return 0, errors.New("cbor_unexpected_eof")
			}
			value := uint64(binary.BigEndian.Uint32(encoded[offset:]))
			offset += 4
			return value, nil
		case additional == 27:
			if offset+8 > len(encoded) {
				return 0, errors.New("cbor_unexpected_eof")
			}
			value := binary.BigEndian.Uint64(encoded[offset:])
			offset += 8
			return value, nil
		default:
			return 0, errors.New("cbor_length_unsupported")
		}
	}

	switch major {
	case 0:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		return int64(length), offset, nil
	case 1:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		return int64(-1) - int64(length), offset, nil
	case 2:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		if uint64(len(encoded)-offset) < length {
			return nil, offset, errors.New("cbor_unexpected_eof")
		}
		value := make([]byte, int(length))
		copy(value, encoded[offset:offset+int(length)])
		offset += int(length)
		return value, offset, nil
	case 3:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		if uint64(len(encoded)-offset) < length {
			return nil, offset, errors.New("cbor_unexpected_eof")
		}
		value := string(encoded[offset : offset+int(length)])
		offset += int(length)
		return value, offset, nil
	case 4:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		values := make([]any, 0, length)
		for i := uint64(0); i < length; i++ {
			value, nextOffset, err := parseCBORAt(encoded, offset)
			if err != nil {
				return nil, offset, err
			}
			offset = nextOffset
			values = append(values, value)
		}
		return values, offset, nil
	case 5:
		length, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		values := make([]cborItem, 0, length)
		for i := uint64(0); i < length; i++ {
			key, nextOffset, err := parseCBORAt(encoded, offset)
			if err != nil {
				return nil, offset, err
			}
			offset = nextOffset
			value, nextOffset, err := parseCBORAt(encoded, offset)
			if err != nil {
				return nil, offset, err
			}
			offset = nextOffset
			values = append(values, cborItem{Key: key, Value: value})
		}
		return values, offset, nil
	case 6:
		_, err := readLength()
		if err != nil {
			return nil, offset, err
		}
		return parseCBORAt(encoded, offset)
	case 7:
		switch additional {
		case 20:
			return false, offset, nil
		case 21:
			return true, offset, nil
		case 22, 23:
			return nil, offset, nil
		case 24:
			if offset >= len(encoded) {
				return nil, offset, errors.New("cbor_unexpected_eof")
			}
			simple := encoded[offset]
			offset++
			return simple, offset, nil
		default:
			return nil, offset, fmt.Errorf("cbor_simple_type_unsupported:%d", additional)
		}
	default:
		return nil, offset, fmt.Errorf("cbor_major_type_unsupported:%d", major)
	}
}

func cborMapValue(entries []cborItem, key any) (any, bool) {
	for _, entry := range entries {
		if cborKeyEquals(entry.Key, key) {
			return entry.Value, true
		}
	}
	return nil, false
}

func cborKeyEquals(left any, right any) bool {
	switch l := left.(type) {
	case string:
		r, ok := right.(string)
		return ok && l == r
	case int64:
		switch r := right.(type) {
		case int64:
			return l == r
		case int:
			return l == int64(r)
		}
	case uint64:
		switch r := right.(type) {
		case uint64:
			return l == r
		case int64:
			return l == uint64(r)
		}
	}
	return false
}

func passkeyPublicKeyFromCOSE(encoded []byte) (*passkeyPublicKey, error) {
	root, _, err := parseCBOR(encoded)
	if err != nil {
		return nil, err
	}
	entries, ok := root.([]cborItem)
	if !ok {
		return nil, errors.New("cose_key_invalid")
	}

	publicKey := &passkeyPublicKey{}
	if value, ok := cborMapValue(entries, int64(1)); ok {
		if v, valid := cborNumber(value); valid {
			switch v {
			case 2:
				publicKey.KeyType = "ec2"
			case 3:
				publicKey.KeyType = "rsa"
			default:
				return nil, errors.New("cose_key_type_unsupported")
			}
		}
	}
	if value, ok := cborMapValue(entries, int64(3)); ok {
		if v, valid := cborNumber(value); valid {
			publicKey.Alg = int(v)
		}
	}

	switch publicKey.KeyType {
	case "ec2":
		if publicKey.Alg != -7 {
			return nil, errors.New("cose_key_algorithm_unsupported")
		}
		crvValue, ok := cborMapValue(entries, int64(-1))
		if !ok {
			return nil, errors.New("cose_key_curve_missing")
		}
		crv, valid := cborNumber(crvValue)
		if !valid {
			return nil, errors.New("cose_key_curve_invalid")
		}
		publicKey.Curve = int(crv)
		xValue, ok := cborMapValue(entries, int64(-2))
		if !ok {
			return nil, errors.New("cose_key_x_missing")
		}
		yValue, ok := cborMapValue(entries, int64(-3))
		if !ok {
			return nil, errors.New("cose_key_y_missing")
		}
		xBytes, valid := xValue.([]byte)
		if !valid {
			return nil, errors.New("cose_key_x_invalid")
		}
		yBytes, valid := yValue.([]byte)
		if !valid {
			return nil, errors.New("cose_key_y_invalid")
		}
		publicKey.X = base64.RawURLEncoding.EncodeToString(xBytes)
		publicKey.Y = base64.RawURLEncoding.EncodeToString(yBytes)
		return publicKey, nil
	case "rsa":
		if publicKey.Alg != -257 {
			return nil, errors.New("cose_key_algorithm_unsupported")
		}
		nValue, ok := cborMapValue(entries, int64(-1))
		if !ok {
			return nil, errors.New("cose_key_modulus_missing")
		}
		eValue, ok := cborMapValue(entries, int64(-2))
		if !ok {
			return nil, errors.New("cose_key_exponent_missing")
		}
		nBytes, valid := nValue.([]byte)
		if !valid {
			return nil, errors.New("cose_key_modulus_invalid")
		}
		eBytes, valid := eValue.([]byte)
		if !valid {
			return nil, errors.New("cose_key_exponent_invalid")
		}
		publicKey.Modulus = base64.RawURLEncoding.EncodeToString(nBytes)
		publicKey.Exponent = base64.RawURLEncoding.EncodeToString(eBytes)
		return publicKey, nil
	default:
		return nil, errors.New("cose_key_material_unsupported")
	}
}

func cborNumber(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint64:
		return int64(v), true
	}
	return 0, false
}

func extractPasskeyRegistrationMaterial(attestationEncoded string, rpID string) (*passkeyPublicKey, uint32, error) {
	raw, err := decodePasskeyPayload(attestationEncoded)
	if err != nil {
		return nil, 0, fmt.Errorf("decode attestation object: %w", err)
	}

	root, _, err := parseCBOR(raw)
	if err != nil {
		return nil, 0, fmt.Errorf("parse attestation object: %w", err)
	}
	entries, ok := root.([]cborItem)
	if !ok {
		return nil, 0, errors.New("attestation_object_invalid")
	}
	authDataRaw, ok := cborMapValue(entries, "authData")
	if !ok {
		return nil, 0, errors.New("attestation_auth_data_missing")
	}
	authData, ok := authDataRaw.([]byte)
	if !ok {
		return nil, 0, errors.New("attestation_auth_data_invalid")
	}
	signCount, err := parsePasskeyAuthenticatorData(base64.RawURLEncoding.EncodeToString(authData), rpID)
	if err != nil {
		return nil, 0, err
	}
	if len(authData) < 55 {
		return nil, 0, errors.New("attestation_authenticator_data_invalid")
	}
	flags := authData[32]
	if flags&0x40 == 0 {
		return nil, 0, errors.New("attestation_credential_data_missing")
	}
	offset := 37
	if len(authData) < offset+18 {
		return nil, 0, errors.New("attestation_credential_data_invalid")
	}
	offset += 16
	credentialLength := int(binary.BigEndian.Uint16(authData[offset : offset+2]))
	offset += 2
	if credentialLength <= 0 || len(authData) < offset+credentialLength {
		return nil, 0, errors.New("attestation_credential_length_invalid")
	}
	offset += credentialLength
	publicKey, err := passkeyPublicKeyFromCOSE(authData[offset:])
	if err != nil {
		return nil, 0, err
	}
	return publicKey, signCount, nil
}

func passkeyAssertionVerified(authenticatorData string, clientDataJSON string, signature string, publicKey *passkeyPublicKey, rpID string, storedSignCount uint32) (uint32, error) {
	if publicKey == nil {
		return 0, errors.New("passkey_public_key_missing")
	}

	authData, err := decodePasskeyPayload(authenticatorData)
	if err != nil {
		return 0, fmt.Errorf("decode authenticator data: %w", err)
	}
	clientDataRaw, err := decodePasskeyPayload(clientDataJSON)
	if err != nil {
		return 0, fmt.Errorf("decode client data: %w", err)
	}
	if len(authData) < 37 {
		return 0, errors.New("authenticator_data_invalid")
	}
	expectedHash := sha256.Sum256([]byte(rpID))
	if !bytes.Equal(authData[:32], expectedHash[:]) {
		return 0, errors.New("authenticator_rp_mismatch")
	}
	flags := authData[32]
	if flags&0x01 == 0 {
		return 0, errors.New("authenticator_user_not_present")
	}
	if flags&0x04 == 0 {
		return 0, errors.New("authenticator_user_not_verified")
	}
	signCount := binary.BigEndian.Uint32(authData[33:37])
	clientHash := sha256.Sum256(clientDataRaw)
	signedData := make([]byte, 0, len(authData)+len(clientHash))
	signedData = append(signedData, authData...)
	signedData = append(signedData, clientHash[:]...)
	signatureBytes, err := decodePasskeyPayload(signature)
	if err != nil {
		return 0, fmt.Errorf("decode signature: %w", err)
	}

	switch publicKey.KeyType {
	case "ec2":
		if publicKey.Alg != -7 {
			return 0, errors.New("passkey_algorithm_unsupported")
		}
		if publicKey.Curve != 1 {
			return 0, errors.New("passkey_curve_unsupported")
		}
		xBytes, err := decodePasskeyPayload(publicKey.X)
		if err != nil {
			return 0, fmt.Errorf("decode public key x: %w", err)
		}
		yBytes, err := decodePasskeyPayload(publicKey.Y)
		if err != nil {
			return 0, fmt.Errorf("decode public key y: %w", err)
		}
		curve := elliptic.P256()
		pubKey := &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}
		digest := sha256.Sum256(signedData)
		if !ecdsa.VerifyASN1(pubKey, digest[:], signatureBytes) {
			return 0, errors.New("passkey_signature_invalid")
		}
	case "rsa":
		if publicKey.Alg != -257 {
			return 0, errors.New("passkey_algorithm_unsupported")
		}
		nBytes, err := decodePasskeyPayload(publicKey.Modulus)
		if err != nil {
			return 0, fmt.Errorf("decode public key modulus: %w", err)
		}
		eBytes, err := decodePasskeyPayload(publicKey.Exponent)
		if err != nil {
			return 0, fmt.Errorf("decode public key exponent: %w", err)
		}
		exponent := 0
		for _, b := range eBytes {
			exponent = exponent<<8 + int(b)
		}
		if exponent <= 0 {
			return 0, errors.New("passkey_exponent_invalid")
		}
		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: exponent,
		}
		digest := sha256.Sum256(signedData)
		if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest[:], signatureBytes); err != nil {
			return 0, errors.New("passkey_signature_invalid")
		}
	default:
		return 0, errors.New("passkey_public_key_unsupported")
	}

	if storedSignCount > 0 && signCount > 0 && signCount < storedSignCount {
		return 0, errors.New("passkey_sign_count_invalid")
	}
	return signCount, nil
}

func ensurePasskeyPublicKey(payload map[string]any, rpID string) (*passkeyPublicKey, uint32, error) {
	if existing, ok := payload["passkey"].(map[string]any); ok {
		encoded, err := json.Marshal(existing)
		if err == nil {
			var material struct {
				PublicKey *passkeyPublicKey `json:"public_key"`
				SignCount uint32            `json:"sign_count"`
			}
			if err := json.Unmarshal(encoded, &material); err == nil && material.PublicKey != nil {
				return material.PublicKey, material.SignCount, nil
			}
		}
	}

	response, _ := payload["response"].(map[string]any)
	if response == nil {
		return nil, 0, errors.New("passkey_material_missing")
	}
	attestationObject, _ := response["attestationObject"].(string)
	if attestationObject == "" {
		return nil, 0, errors.New("passkey_attestation_missing")
	}
	publicKey, signCount, err := extractPasskeyRegistrationMaterial(attestationObject, rpID)
	if err != nil {
		return nil, 0, err
	}
	return publicKey, signCount, nil
}

func passkeyPublicKeyFromStoredPayload(raw []byte, rpID string) (*passkeyPublicKey, uint32, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, 0, fmt.Errorf("parse passkey payload: %w", err)
	}
	return ensurePasskeyPublicKey(payload, rpID)
}

func (s *Service) issueLoginSession(ctx context.Context, subject string) (string, error) {
	sessionID, err := randomToken(24)
	if err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	claims := signedSessionClaims{
		Subject:   subject,
		SessionID: sessionID,
		ExpiresAt: time.Now().Add(loginSessionTTL).UTC().Unix(),
	}
	token, err := signSessionClaims(claims)
	if err != nil {
		return "", err
	}

	if _, err := s.db.Exec(ctx, `
		insert into app_login_sessions (session_id, subject, expires_at, revoked, created_at, updated_at)
		values ($1, $2, to_timestamp($3), false, now(), now())
		on conflict (session_id) do update
		set subject = excluded.subject,
			expires_at = excluded.expires_at,
			revoked = false,
			updated_at = now()
	`, sessionID, subject, claims.ExpiresAt); err != nil {
		return "", fmt.Errorf("store login session: %w", err)
	}

	return token, nil
}

func (s *Service) revokeLoginSession(ctx context.Context, token string) error {
	claims, err := verifySessionClaims(token)
	if err != nil {
		return nil
	}

	if _, err := s.db.Exec(ctx, `
		update app_login_sessions
		set revoked = true,
			updated_at = now()
		where session_id = $1
			and subject = $2
	`, claims.SessionID, claims.Subject); err != nil {
		return fmt.Errorf("revoke login session: %w", err)
	}
	return nil
}

func (s *Service) setLoginSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Environment == "production",
		MaxAge:   int(loginSessionTTL.Seconds()),
	})
}

func (s *Service) resolveLoginSubject(ctx context.Context, token string) (string, error) {
	claims, err := verifySessionClaims(token)
	if err != nil {
		return "", err
	}

	var active bool
	err = s.db.QueryRow(ctx, `
		select exists(
			select 1
			from app_login_sessions
			where session_id = $1
				and subject = $2
				and revoked = false
				and expires_at > now()
		)
	`, claims.SessionID, claims.Subject).Scan(&active)
	if err != nil {
		return "", fmt.Errorf("lookup login session: %w", err)
	}
	if !active {
		return "", errors.New("session_revoked")
	}

	_, _ = s.db.Exec(ctx, `
		update app_login_sessions
		set last_used_at = now(),
			updated_at = now()
		where session_id = $1
	`, claims.SessionID)

	return claims.Subject, nil
}
