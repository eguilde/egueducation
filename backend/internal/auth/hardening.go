package auth

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

const (
	passkeyChallengeTTL = 5 * time.Minute
)

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
	PublicKey *passkeyPublicKey `json:"public_key,omitempty"`
	SignCount uint32            `json:"sign_count,omitempty"`
}

type passkeyRecord struct {
	Subject    string
	UserID     string
	DeviceName string
	PublicKey  *passkeyPublicKey
	SignCount  uint32
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
