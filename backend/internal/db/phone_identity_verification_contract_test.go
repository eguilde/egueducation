package db

import (
	"strings"
	"testing"
)

func TestPhoneIdentityVerificationMigrationEnforcesPossessionProof(t *testing.T) {
	const migrationName = "migrations/0086_phone_identity_verification_contract.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)
	for _, required := range []string{
		"create constraint trigger trg_app_users_phone_identity_contract",
		"deferrable initially deferred",
		"Phone numbers are globally-owned login credentials",
		"identity_type = 'phone'",
		"i.is_primary",
		"i.verified_at is not null",
		"phone verification requires successful SMS OTP identity verification",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("phone verification migration is missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("phone verification migration must not weaken tenant isolation")
	}
}
