package config

import "testing"

func TestLoadProductionFixedOTPTestUserContract(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("FRONTEND_ORIGIN", "https://school.example.test")
	t.Setenv("BACKEND_URL", "https://school.example.test")
	t.Setenv("OIDC_ISSUER", "https://school.example.test/api/oidc")
	t.Setenv("ENABLE_TEST_OTP_FIXTURE", "false")
	t.Setenv("ENABLE_PRODUCTION_FIXED_OTP_TEST_USER", "true")
	t.Setenv("PRODUCTION_TEST_USER_OTP", "482615")
	t.Setenv("PRODUCTION_TEST_USER_IDENTIFIER", "test@eguilde.cloud")
	t.Setenv("PRODUCTION_TEST_USER_SUBJECT", "production-fixed-otp-test-subject")
	t.Setenv("PRODUCTION_TEST_USER_TENANT", "tenant-school")
	// Production names are authoritative and cannot be shadowed by the local
	// integration-fixture contract.
	t.Setenv("TEST_OTP_FIXTURE_CODE", "111111")
	t.Setenv("TEST_OTP_FIXTURE_IDENTIFIER", "local@example.test")

	cfg := Load()
	if !cfg.EnableProductionFixedOTPTestUser || cfg.TestOTPFixtureCode != "482615" || cfg.TestOTPFixtureIdentifier != "test@eguilde.cloud" || cfg.TestOTPFixtureSubject != "production-fixed-otp-test-subject" || cfg.TestOTPFixtureTenantCode != "tenant-school" {
		t.Fatal("production fixed-OTP env contract was not loaded")
	}
	if err := cfg.ValidateTestOTPFixture(); err != nil {
		t.Fatalf("production fixed-OTP env contract rejected: %v", err)
	}
}
