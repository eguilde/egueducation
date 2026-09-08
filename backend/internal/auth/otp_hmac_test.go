package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const (
	otpHMACTestKeyOne = "otp-hmac-test-key-one-must-be-at-least-32-bytes"
	otpHMACTestKeyTwo = "otp-hmac-test-key-two-must-be-at-least-32-bytes"
)

func TestOTPStorageUsesVersionedKeyedHMAC(t *testing.T) {
	one, err := newOTPService(nil, otpHMACTestKeyOne)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}
	two, err := newOTPService(nil, otpHMACTestKeyTwo)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}

	const code = "482615"
	firstHash := one.hashCode(code)
	if !strings.HasPrefix(firstHash, otpHashPrefix) {
		t.Fatalf("OTP hash %q has no explicit current-version prefix", firstHash)
	}
	if firstHash == two.hashCode(code) {
		t.Fatal("different HMAC keys produced the same OTP storage value")
	}
	plain := sha256.Sum256([]byte(code))
	if firstHash == hex.EncodeToString(plain[:]) || strings.HasSuffix(firstHash, hex.EncodeToString(plain[:])) {
		t.Fatal("OTP storage must not be a plain SHA-256 of the submitted code")
	}
}

func TestOTPStorageDeniesWrongKeyAndWrongCode(t *testing.T) {
	issuer, err := newOTPService(nil, otpHMACTestKeyOne)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}
	otherKey, err := newOTPService(nil, otpHMACTestKeyTwo)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}
	stored := issuer.hashCode("482615")

	if match, current := issuer.matchesStoredCode(stored, "482616"); !current || match {
		t.Fatalf("wrong code: match=%v current=%v, want false true", match, current)
	}
	if match, current := otherKey.matchesStoredCode(stored, "482615"); !current || match {
		t.Fatalf("wrong key: match=%v current=%v, want false true", match, current)
	}
	if match, current := issuer.matchesStoredCode(stored, "482615"); !current || !match {
		t.Fatalf("correct key and code: match=%v current=%v, want true true", match, current)
	}
}

func TestOTPLegacySHAStorageIsDeniedAndRequiresRegeneration(t *testing.T) {
	service, err := newOTPService(nil, otpHMACTestKeyOne)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}
	plain := sha256.Sum256([]byte("482615"))
	legacySHA := hex.EncodeToString(plain[:])
	if match, current := service.matchesStoredCode(legacySHA, "482615"); current || match {
		t.Fatalf("legacy plain SHA row was accepted: match=%v current=%v", match, current)
	}
	if match, current := service.matchesStoredCode("hmac-sha256:v0:"+legacySHA, "482615"); current || match {
		t.Fatalf("unknown OTP hash version was accepted: match=%v current=%v", match, current)
	}
}

func TestFixtureOTPUsesTheSameKeyedStorage(t *testing.T) {
	service, err := newOTPService(nil, otpHMACTestKeyOne)
	if err != nil {
		t.Fatalf("newOTPService() error = %v", err)
	}
	fixtureStored := service.hashCode("482615")
	if match, current := service.matchesStoredCode(fixtureStored, "482615"); !current || !match {
		t.Fatalf("fixed E2E OTP was not accepted through keyed storage: match=%v current=%v", match, current)
	}
}
