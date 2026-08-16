package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
)

type testPinger struct {
	err error
}

func (p testPinger) Ping(context.Context) error {
	return p.err
}

func TestReadinessHandlerReportsDatabaseFailure(t *testing.T) {
	tests := []struct {
		name     string
		pinger   testPinger
		wantCode int
	}{
		{name: "database ready", pinger: testPinger{}, wantCode: http.StatusOK},
		{name: "database unavailable", pinger: testPinger{err: errors.New("unavailable")}, wantCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			readinessHandler(tt.pinger).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantCode)
			}
		})
	}
}

func TestLivenessHandlerDoesNotRequireDatabase(t *testing.T) {
	recorder := httptest.NewRecorder()

	livenessHandler(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestBuildBootstrapConfigIncludesLegacyAndRuntimeFields(t *testing.T) {
	cfg := config.Config{
		FrontendOrigin:     "https://scoalabalotesti.eguilde.cloud",
		Environment:        "production",
		OIDCIssuer:         "https://scoalabalotesti.eguilde.cloud/api/oidc",
		OIDCClientID:       "egueducation-spa",
		OIDCDesktopClient:  "egueducation-desktop",
		EnablePasskeys:     true,
		EnableWallet:       true,
		EnableSMSOTP:       true,
		EnableGDPRFeatures: true,
	}

	req := httptest.NewRequest("GET", "https://scoalabalotesti.eguilde.cloud/api/config", nil)
	payload := buildBootstrapConfig(cfg, req)

	if got, ok := payload["oidcClientId"].(string); !ok || got != "egueducation-spa" {
		t.Fatalf("oidcClientId = %#v, want eg educ client", payload["oidcClientId"])
	}

	if got, ok := payload["apiBaseUrl"].(string); !ok || got != "/api" {
		t.Fatalf("apiBaseUrl = %#v, want /api", payload["apiBaseUrl"])
	}

	if got, ok := payload["institutionId"].(string); !ok || got == "" {
		t.Fatalf("institutionId = %#v, want non-empty string", payload["institutionId"])
	}

	customer, ok := payload["customer"].(map[string]any)
	if !ok {
		t.Fatalf("customer = %#v, want object", payload["customer"])
	}
	if got, ok := customer["name"].(string); !ok || got == "" {
		t.Fatalf("customer.name = %#v, want non-empty string", customer["name"])
	}

	modules, ok := payload["modules"].(map[string]any)
	if !ok {
		t.Fatalf("modules = %#v, want object", payload["modules"])
	}
	enabled, ok := modules["enabled"].([]string)
	if !ok {
		t.Fatalf("modules.enabled = %#v, want []string", modules["enabled"])
	}
	if len(enabled) == 0 {
		t.Fatal("modules.enabled should not be empty")
	}

	features, ok := payload["features"].(map[string]any)
	if !ok {
		t.Fatalf("features = %#v, want object", payload["features"])
	}
	if got, ok := features["gdpr"].(bool); !ok || !got {
		t.Fatalf("features.gdpr = %#v, want true", features["gdpr"])
	}
}
