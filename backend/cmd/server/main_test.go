package main

import (
	"net/http/httptest"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
)

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
