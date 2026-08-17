package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
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
		EnableTestOTPFixture:             true,
		EnableProductionE2ECanary:        true,
		ProductionE2ECanaryActivationKey: strings.Repeat("a", 32),
		ProductionE2ECanarySigningKey:    strings.Repeat("s", 32),
		TestOTPFixtureCode:               "482615",
		TestOTPFixtureIdentifier:         "test@eguilde.cloud",
		TestOTPFixtureSubject:            "production-browser-fixture-subject",
		TestOTPFixtureTenantCode:         "tenant-school",
	}
}

func TestProductionE2ECanaryUsesTheDedicatedRBACIdentityThroughNormalOTP(t *testing.T) {
	cfg := productionCanaryConfig()
	if err := cfg.ValidateTestOTPFixture(); err != nil {
		t.Fatalf("valid production canary rejected: %v", err)
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
}

func TestProductionCanaryIdentityRemainsReservedWhenFeatureIsDisabled(t *testing.T) {
	cfg := productionCanaryConfig()
	cfg.EnableProductionE2ECanary = false
	if !isReservedProductionE2ECanaryUser(&cfg, oidcTestFixtureUserID) {
		t.Fatal("a provisioned production canary must never fall back to normal SMS when the feature is disabled")
	}
	if isReservedProductionE2ECanaryUser(&cfg, uuid.New()) {
		t.Fatal("ordinary production users must not be treated as the reserved canary")
	}
}

func TestBeginProductionE2ECanaryDoesNotPutGateKeyInCookie(t *testing.T) {
	cfg := productionCanaryConfig()
	service := &Service{cfg: cfg}

	request := httptest.NewRequest(http.MethodPost, "https://school.example.test/api/oidc/e2e-canary/session", nil)
	request.Header.Set("Origin", cfg.FrontendOrigin)
	request.Header.Set("Authorization", "Bearer "+cfg.ProductionE2ECanaryActivationKey)
	response := httptest.NewRecorder()
	service.BeginProductionE2ECanary(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one canary cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if strings.Contains(cookie.Value, cfg.ProductionE2ECanaryActivationKey) || strings.Contains(cookie.Value, cfg.ProductionE2ECanarySigningKey) || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api/oidc" {
		t.Fatal("canary cookie must be opaque, Secure, HttpOnly, SameSite=Strict, and OIDC-path scoped")
	}

	request = httptest.NewRequest(http.MethodPost, "https://school.example.test/api/oidc/e2e-canary/session", nil)
	request.Header.Set("Origin", cfg.FrontendOrigin)
	request.Header.Set("Authorization", "Bearer wrong")
	response = httptest.NewRecorder()
	service.BeginProductionE2ECanary(response, request)
	if response.Code != http.StatusUnauthorized || len(response.Result().Cookies()) != 0 {
		t.Fatal("wrong canary gate must not create a cookie")
	}
}

func TestBeginProductionE2ECanaryRequiresExactHostAndOrigin(t *testing.T) {
	cfg := productionCanaryConfig()
	service := &Service{cfg: cfg}
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "https://other.example.test/api/oidc/e2e-canary/session", nil),
		httptest.NewRequest(http.MethodPost, "https://school.example.test/api/oidc/e2e-canary/session", nil),
	} {
		request.Header.Set("Authorization", "Bearer "+cfg.ProductionE2ECanaryActivationKey)
		if request.Host != "other.example.test" {
			request.Header.Set("Origin", "https://other.example.test")
		} else {
			request.Header.Set("Origin", cfg.FrontendOrigin)
		}
		response := httptest.NewRecorder()
		service.BeginProductionE2ECanary(response, request)
		if response.Code != http.StatusUnauthorized || len(response.Result().Cookies()) != 0 {
			t.Fatal("wrong canary host or origin must fail without a cookie")
		}
	}
}

func TestBeginProductionE2ECanaryThrottlesRepeatedInvalidSecrets(t *testing.T) {
	cfg := productionCanaryConfig()
	service := &Service{cfg: cfg}
	const remote = "198.51.100.77:12345"
	clearProductionE2ECanaryFailures(&http.Request{RemoteAddr: remote})
	t.Cleanup(func() { clearProductionE2ECanaryFailures(&http.Request{RemoteAddr: remote}) })

	for attempt := 1; attempt <= productionE2ECanaryLimit+1; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "https://school.example.test/api/oidc/e2e-canary/session", nil)
		request.RemoteAddr = remote
		request.Header.Set("Origin", cfg.FrontendOrigin)
		request.Header.Set("Authorization", "Bearer invalid")
		response := httptest.NewRecorder()
		service.BeginProductionE2ECanary(response, request)
		expected := http.StatusUnauthorized
		if attempt > productionE2ECanaryLimit {
			expected = http.StatusTooManyRequests
		}
		if response.Code != expected {
			t.Fatalf("attempt %d status=%d, want %d", attempt, response.Code, expected)
		}
	}
}

func TestProductionE2ECanaryConfigurationFailsClosed(t *testing.T) {
	tests := []config.Config{
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.EnableProductionE2ECanary = false
			return cfg
		}(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.ProductionE2ECanaryActivationKey = "short"
			return cfg
		}(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.ProductionE2ECanarySigningKey = "short"
			return cfg
		}(),
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.ProductionE2ECanarySigningKey = cfg.ProductionE2ECanaryActivationKey
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
		func() config.Config {
			cfg := productionCanaryConfig()
			cfg.TestOTPFixtureCode = "173829"
			return cfg
		}(),
	}
	for _, cfg := range tests {
		if err := cfg.ValidateTestOTPFixture(); err == nil {
			t.Fatal("unsafe production canary configuration must fail startup validation")
		}
	}
}
