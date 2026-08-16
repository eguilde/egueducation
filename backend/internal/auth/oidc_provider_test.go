package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/luikyv/go-oidc/pkg/goidc"
)

func TestOTPLoginUIMessageDoesNotExposeCodeWhenSMSIsConfigured(t *testing.T) {
	const code = "123456"
	message, ok := otpLoginUIMessage(&config.Config{
		Environment:    "development",
		FrontendOrigin: "https://school.example.test",
	}, code, true)
	if !ok {
		t.Fatal("configured SMS delivery should allow the login flow to continue")
	}
	if strings.Contains(message, code) {
		t.Fatal("configured SMS delivery must not expose the OTP in the browser")
	}
}

func TestOIDCRevocationAllowListIncludesOnlyConfiguredFirstPartyClients(t *testing.T) {
	allowed := oidcClientAllowedRevocation(&config.Config{
		OIDCClientID:      "first-party-spa",
		OIDCDesktopClient: "first-party-desktop",
	})
	for _, clientID := range []string{"first-party-spa", "first-party-desktop"} {
		if !allowed(&goidc.Client{ID: clientID}) {
			t.Fatalf("configured client %q must be allowed to revoke its own token", clientID)
		}
	}
	if allowed(&goidc.Client{ID: "dynamically-registered-client"}) || allowed(nil) {
		t.Fatal("unconfigured or absent client must not be allowed to revoke tokens")
	}
}

func TestRefreshCookieBridgeNeverSubstitutesTokenAtRFC7009Revoke(t *testing.T) {
	var receivedToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	})
	handler := wrapRefreshTokenCookie(next, &config.Config{})
	request := httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(url.Values{
		"token": {"cookie"}, "client_id": {"first-party-spa"},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "egueducation_rt", Value: "server-side-refresh-token"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if receivedToken != "cookie" {
		t.Fatalf("RFC 7009 token was rewritten to %q; revocation must receive the submitted token only", receivedToken)
	}
}

func TestOTPLoginUIMessageAllowsCodeOnlyForLoopbackDevelopment(t *testing.T) {
	const code = "123456"
	tests := []struct {
		name    string
		cfg     *config.Config
		allowed bool
		exposes bool
	}{
		{
			name: "localhost development",
			cfg: &config.Config{
				Environment:    "development",
				FrontendOrigin: "http://localhost:4200",
			},
			allowed: true,
			exposes: true,
		},
		{
			name: "loopback test",
			cfg: &config.Config{
				Environment:    "test",
				FrontendOrigin: "http://127.0.0.1:4200",
			},
			allowed: true,
			exposes: true,
		},
		{
			name: "public development host fails closed",
			cfg: &config.Config{
				Environment:    "development",
				FrontendOrigin: "https://school.example.test",
			},
		},
		{
			name: "production loopback fails closed",
			cfg: &config.Config{
				Environment:    "production",
				FrontendOrigin: "http://localhost:4200",
			},
		},
		{name: "missing configuration fails closed"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message, ok := otpLoginUIMessage(test.cfg, code, false)
			if ok != test.allowed {
				t.Fatalf("allowed = %v, want %v", ok, test.allowed)
			}
			if got := strings.Contains(message, code); got != test.exposes {
				t.Fatalf("message exposes code = %v, want %v", got, test.exposes)
			}
		})
	}
}

func TestOTPLoginUIAutoAdvancesAndSupportsPasteAndBackspace(t *testing.T) {
	if got := strings.Count(oidcLoginHTML, `class="otp-box"`); got != 6 {
		t.Fatalf("OTP login renders %d digit inputs, want 6", got)
	}
	for _, required := range []string{
		`autocomplete="one-time-code"`,
		`<script src="/api/oidc/ui/login.js" defer></script>`,
	} {
		if !strings.Contains(oidcLoginHTML, required) {
			t.Fatalf("OTP login is missing required markup %q", required)
		}
	}
	for _, required := range []string{
		`box.addEventListener('input'`,
		`otpBoxes[index+1].focus()`,
		`box.addEventListener('paste'`,
		`box.addEventListener('keydown'`,
		`event.key==='Backspace'`,
	} {
		if !strings.Contains(oidcLoginScript, required) {
			t.Fatalf("OTP login is missing required keyboard behavior %q", required)
		}
	}
	if strings.Contains(oidcLoginHTML, "<script>") || strings.Contains(oidcLoginHTML, " onchange=") || strings.Contains(oidcLoginHTML, " onclick=") {
		t.Fatal("OIDC login must not depend on inline script or event handlers blocked by the frontend CSP")
	}
}

func TestOIDCLoginScriptIsServedAsCSPCompatibleJavaScript(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/oidc/ui/login.js", nil)
	recorder := httptest.NewRecorder()
	(&Service{}).OIDCLoginScript(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/javascript") {
		t.Fatalf("content type = %q, want JavaScript", contentType)
	}
	if !strings.Contains(recorder.Body.String(), `box.addEventListener('input'`) {
		t.Fatal("served OIDC script is missing OTP input behavior")
	}
}
