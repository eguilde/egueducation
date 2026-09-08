package db

import (
	"strings"
	"testing"
)

func TestOTPSessionBindingMigrationUsesRollingSafeChallengeTable(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0093_oidc_otp_session_binding.sql")
	if err != nil {
		t.Fatalf("read OTP session migration: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"create table if not exists oidc_otp_challenges",
		"authn_session_id text not null",
		"tenant_code text not null references app_tenants(code)",
		"identity_id uuid not null references app_user_identities(id)",
		"primary key (authn_session_id, purpose)",
		"delete from oidc_otp_codes",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("OTP session migration is missing %q", required)
		}
	}
}

func TestPhoneHistoricalValidationMigrationFailsClosed(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0094_phone_identity_historical_validation.sql")
	if err != nil {
		t.Fatalf("read historical phone validation migration: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"where u.phone_number_verified",
		"identity.verified_at is not null",
		"identity.normalized_value = case",
		"phone identity historical validation failed",
		"validate constraint app_user_identities_phone_e164",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("historical phone validation migration is missing %q", required)
		}
	}
}

func TestPublishedIdentityMigrationHasForwardReconciliation(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0095_identity_contract_published_0083_reconciliation.sql")
	if err != nil {
		t.Fatalf("read published identity reconciliation migration: %v", err)
	}
	text := string(contents)
	for _, required := range []string{
		"0083_identity_contract_foundation.sql",
		"user-thomas-galambos",
		"user-stelian-fedorca",
		"user-diana-ilhan",
		"set verified_at = null",
		"phone_number_verified = false",
		"identity.phone.enrollment_verified",
		"identity.phone.login_authenticated",
		"details->>'method' = 'sms_otp'",
		"matching_users <> 1 or matching_admins <> 1",
		"will not select an owner automatically",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("published identity reconciliation is missing %q", required)
		}
	}
}
