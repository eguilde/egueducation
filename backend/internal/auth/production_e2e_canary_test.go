package auth

import (
	"testing"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/google/uuid"
)

func productionCanaryConfig() config.Config {
	return config.Config{
		Environment:                      "production",
		FrontendOrigin:                   "https://school.example.test",
		BackendURL:                       "https://school.example.test",
		OIDCIssuer:                       "https://school.example.test/api/oidc",
		EnableProductionFixedOTPTestUser: true,
		TestOTPFixtureCode:               "482615",
		TestOTPFixtureIdentifier:         "test@eguilde.cloud",
		TestOTPFixtureSubject:            "production-browser-fixture-subject",
		TestOTPFixtureTenantCode:         "tenant-school",
	}
}

func TestProductionFixedOTPTestUserUsesTheDedicatedRBACIdentityThroughNormalOTP(t *testing.T) {
	cfg := productionCanaryConfig()
	if err := cfg.ValidateTestOTPFixture(); err != nil {
		t.Fatalf("valid production fixed-OTP identity rejected: %v", err)
	}
	user := oidcLoginUser{ID: oidcTestFixtureUserID, Email: cfg.TestOTPFixtureIdentifier, Subject: cfg.TestOTPFixtureSubject}
	if !testOTPFixtureAllowed(nil, &cfg, user, cfg.TestOTPFixtureTenantCode) {
		t.Fatal("the exact dedicated production identity must receive its configured OTP")
	}
	wrongUser := user
	wrongUser.ID = uuid.New()
	if testOTPFixtureAllowed(nil, &cfg, wrongUser, cfg.TestOTPFixtureTenantCode) {
		t.Fatal("matching email and subject on a different user must not receive the fixed OTP")
	}
	if testOTPFixtureAllowed(nil, &cfg, user, "other-tenant") {
		t.Fatal("the fixed OTP must not authenticate the identity on another tenant")
	}
}

func TestProductionCanaryIdentityRemainsReservedWhenFeatureIsDisabled(t *testing.T) {
	cfg := productionCanaryConfig()
	cfg.EnableProductionFixedOTPTestUser = false
	if !isReservedProductionFixedOTPTestUser(&cfg, oidcTestFixtureUserID) {
		t.Fatal("a provisioned production test user must never fall back to normal SMS when the feature is disabled")
	}
	if isReservedProductionFixedOTPTestUser(&cfg, uuid.New()) {
		t.Fatal("ordinary production users must not be treated as the reserved test identity")
	}
}

func TestProductionFixedOTPTestUserConfigurationFailsClosed(t *testing.T) {
	tests := []config.Config{
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.EnableProductionFixedOTPTestUser = false
			return cfg
		}(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.FrontendOrigin = "http://school.example.test"
			return cfg
		}(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.BackendURL = "https://other.example.test"
			return cfg
		}(),
		func() config.Config { cfg := productionCanaryConfig(); cfg.TestOTPFixtureCode = "173829"; return cfg }(),
		func() config.Config { cfg := productionCanaryConfig(); cfg.TestOTPFixtureCode = "12x615"; return cfg }(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.TestOTPFixtureIdentifier = "other@eguilde.cloud"
			return cfg
		}(),
		func() config.Config { cfg := productionCanaryConfig(); cfg.TestOTPFixtureSubject = ""; return cfg }(),
		func() config.Config { cfg := productionCanaryConfig(); cfg.TestOTPFixtureTenantCode = ""; return cfg }(),
	}
	for _, cfg := range tests {
		if err := cfg.ValidateTestOTPFixture(); err == nil {
			t.Fatal("unsafe production fixed-OTP configuration must fail startup validation")
		}
	}
}
