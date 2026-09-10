package auth

import (
	"bytes"
	"encoding/json"
	"html/template"
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

func TestOIDCLoginHTMLExposesThePasskeyInteractionAction(t *testing.T) {
	const action = "/api/oidc/authorize/interaction-id/login"
	tmpl := template.Must(template.New("login-passkey-action").Parse(oidcLoginHTML))
	var page bytes.Buffer
	if err := tmpl.Execute(&page, oidcLoginData{
		Step:       "methods",
		FormAction: action,
		Theme:      resolveOIDCThemeSettings(nil, nil),
	}); err != nil {
		t.Fatalf("render login page: %v", err)
	}
	if !strings.Contains(page.String(), `data-oidc-action="`+action+`"`) {
		t.Fatalf("login page does not expose the passkey interaction action")
	}
}

func TestTestOTPFixtureFailsClosedUnlessExplicitLoopbackConfigurationAndExactIdentity(t *testing.T) {
	valid := config.Config{
		Environment:              "test",
		FrontendOrigin:           "http://127.0.0.1:4173",
		BackendURL:               "http://127.0.0.1:8080",
		OIDCIssuer:               "http://127.0.0.1:8080/api/oidc",
		EnableTestOTPFixture:     true,
		TestOTPFixtureCode:       "173829",
		TestOTPFixtureIdentifier: "oidc.browser.fixture@example.test",
		TestOTPFixtureSubject:    "oidc-browser-fixture-subject",
		TestOTPFixtureTenantCode: "tenant-egueducation",
	}
	user := oidcLoginUser{Email: valid.TestOTPFixtureIdentifier, Subject: valid.TestOTPFixtureSubject}
	if !testOTPFixtureAllowed(nil, &valid, user, valid.TestOTPFixtureTenantCode) {
		t.Fatal("explicit loopback fixture configuration must allow the exact synthetic identity")
	}

	tests := []struct {
		name   string
		cfg    config.Config
		user   oidcLoginUser
		tenant string
	}{
		{name: "production is rejected", cfg: withFixtureEnvironment(valid, "production"), user: user, tenant: valid.TestOTPFixtureTenantCode},
		{name: "public frontend is rejected", cfg: withFixtureFrontend(valid, "https://school.example.test"), user: user, tenant: valid.TestOTPFixtureTenantCode},
		{name: "malformed code is rejected", cfg: withFixtureCode(valid, "12x829"), user: user, tenant: valid.TestOTPFixtureTenantCode},
		{name: "wrong user is rejected", cfg: valid, user: oidcLoginUser{Email: "other@example.test", Subject: valid.TestOTPFixtureSubject}, tenant: valid.TestOTPFixtureTenantCode},
		{name: "wrong tenant is rejected", cfg: valid, user: user, tenant: "tenant-other"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.cfg.ValidateTestOTPFixture(); test.name == "wrong user is rejected" || test.name == "wrong tenant is rejected" {
				if err != nil {
					t.Fatalf("valid fixture configuration rejected: %v", err)
				}
			} else if err == nil {
				t.Fatal("unsafe fixture configuration must fail startup validation")
			} else if service, serviceErr := NewService(test.cfg, nil, nil); serviceErr == nil || service != nil {
				t.Fatal("unsafe fixture configuration must fail service initialization")
			}
			if testOTPFixtureAllowed(nil, &test.cfg, test.user, test.tenant) {
				t.Fatal("fixture must fail closed")
			}
		})
	}
}

func TestLoopbackOTPFixturesUseStableDistinctGlobalIdentities(t *testing.T) {
	first := config.Config{Environment: "test", TestOTPFixtureTenantCode: "tenant-egueducation", TestOTPFixtureSubject: "fixture-one"}
	second := first
	second.TestOTPFixtureSubject = "fixture-two"
	if testOTPFixtureUserID(first) != testOTPFixtureUserID(first) {
		t.Fatal("the same loopback fixture must keep a stable user ID across restarts")
	}
	if testOTPFixtureUserID(first) == testOTPFixtureUserID(second) {
		t.Fatal("independent loopback fixtures must never overwrite the same global user")
	}
	if testOTPFixturePhone(first) != testOTPFixturePhone(first) {
		t.Fatal("the same loopback fixture must keep a stable synthetic phone across restarts")
	}
	if testOTPFixturePhone(first) == testOTPFixturePhone(second) {
		t.Fatal("independent loopback fixtures must never share a global phone identity")
	}
	production := first
	production.Environment = "production"
	if testOTPFixtureUserID(production) != oidcTestFixtureUserID {
		t.Fatal("production must retain its single permanently reserved test identity")
	}
	if testOTPFixturePhone(production) != "+40100000000" {
		t.Fatal("production must retain its single auditable reserved test phone")
	}
}

func withFixtureEnvironment(cfg config.Config, environment string) config.Config {
	cfg.Environment = environment
	return cfg
}
func withFixtureFrontend(cfg config.Config, origin string) config.Config {
	cfg.FrontendOrigin = origin
	return cfg
}
func withFixtureCode(cfg config.Config, code string) config.Config {
	cfg.TestOTPFixtureCode = code
	return cfg
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

func TestRefreshCookieBridgeRejectsMissingBrowserSessionBeforeProvider(t *testing.T) {
	called := false
	handler := wrapRefreshTokenCookie(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}), &config.Config{})
	request := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {"cookie"}, "client_id": {"first-party-spa"},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if called {
		t.Fatal("missing browser refresh session was forwarded to the OIDC provider")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode OAuth error: %v", err)
	}
	if payload["error"] != "invalid_grant" {
		t.Fatalf("error = %q, want invalid_grant", payload["error"])
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
		`<script src="/api/oidc/ui/login.js?v=20260817-otp-autoadvance" defer></script>`,
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
		`document.documentElement.dataset.oidcUiReady='true'`,
	} {
		if !strings.Contains(oidcLoginScript, required) {
			t.Fatalf("OTP login is missing required keyboard behavior %q", required)
		}
	}
	if strings.Contains(oidcLoginHTML, "<script>") || strings.Contains(oidcLoginHTML, " onchange=") || strings.Contains(oidcLoginHTML, " onclick=") {
		t.Fatal("OIDC login must not depend on inline script or event handlers blocked by the frontend CSP")
	}
}

func TestOIDCLoginRendersSelectedDarkTheme(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/oidc/authorize?ui_theme_scheme=dark&ui_theme_preset=nora&ui_theme_primary=rose&ui_theme_surface=slate&ui_theme_dark=1", nil)
	data := oidcLoginData{
		CustomerName: "Școala Test",
		Step:         "methods",
		Theme:        resolveOIDCThemeSettings(request, nil),
	}
	loginTemplate := template.Must(template.New("login-theme-test").Parse(oidcLoginHTML))
	var rendered bytes.Buffer
	if err := loginTemplate.Execute(&rendered, data); err != nil {
		t.Fatalf("render OIDC login: %v", err)
	}
	page := rendered.String()
	for _, expected := range []string{
		"color-scheme:dark",
		"--bg:#0f172a",
		"--card:#1e293b",
		"--text:#ffffff",
		"--panel-radius:6px",
		"--control-radius:4px",
	} {
		if !strings.Contains(page, expected) {
			t.Fatalf("dark OIDC login is missing theme token %q", expected)
		}
	}
	if strings.Contains(page, "ZgotmplZ") {
		t.Fatal("dark OIDC login contains a rejected template CSS value")
	}
}

func TestOIDCLoginHidesSMSMethodWhenOTPIsDisabled(t *testing.T) {
	loginTemplate := template.Must(template.New("login-otp-disabled-test").Parse(oidcLoginHTML))
	render := func(enabled bool) string {
		t.Helper()
		var rendered bytes.Buffer
		if err := loginTemplate.Execute(&rendered, oidcLoginData{
			CustomerName: "Școala Test",
			Step:         "methods",
			OTPEnabled:   enabled,
		}); err != nil {
			t.Fatalf("render OIDC login: %v", err)
		}
		return rendered.String()
	}

	if page := render(false); strings.Contains(page, `name="method" value="otp"`) {
		t.Fatal("OIDC login exposes the SMS OTP method while ENABLE_SMS_OTP is false")
	}
	if page := render(true); !strings.Contains(page, `name="method" value="otp"`) {
		t.Fatal("OIDC login does not expose the SMS OTP method while ENABLE_SMS_OTP is true")
	}
}

func TestOIDCLoginRejectsOTPPostsAndExistingOTPStepsWhenDisabled(t *testing.T) {
	loginTemplate := template.Must(template.New("login-otp-incident-disable-test").Parse(oidcLoginHTML))
	cfg := &config.Config{CustomerName: "Școala Test", EnableSMSOTP: false}
	sess := &goidc.AuthnSession{CallbackID: "authn-session-disabled-otp"}

	request := httptest.NewRequest(http.MethodPost, "/api/oidc/authorize/callback/login", strings.NewReader(url.Values{"method": {"otp"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	if _, err := renderMethodStep(recorder, request, sess, nil, cfg, nil, loginTemplate, "/login", nil, nil); err != nil {
		t.Fatalf("reject direct OTP selection: %v", err)
	}
	if strings.Contains(recorder.Body.String(), `name="method" value="otp"`) || !strings.Contains(recorder.Body.String(), "SMS OTP este dezactivată") {
		t.Fatal("disabled OTP POST did not fail closed on the method page")
	}

	sess.StoreParameter("step", "otp")
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/oidc/authorize/callback/login", strings.NewReader(url.Values{"code": {"123456"}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := renderOTPStep(recorder, request, sess, nil, cfg, loginTemplate, "/login", nil); err != nil {
		t.Fatalf("reject existing OTP step: %v", err)
	}
	if step, _ := sess.StoredParameter("step").(string); step != "" {
		t.Fatalf("disabled OTP interaction retained step %q, want reset", step)
	}
	if !strings.Contains(recorder.Body.String(), "SMS OTP este dezactivată") {
		t.Fatal("existing OTP interaction was not rejected after incident disable")
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
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("cache control = %q, want no-store", cacheControl)
	}
	if !strings.Contains(recorder.Body.String(), `box.addEventListener('input'`) {
		t.Fatal("served OIDC script is missing OTP input behavior")
	}
}

func TestOIDCGrantClaimReleaseSeparatesAuthorizationFromScopedIdentity(t *testing.T) {
	subject := oidcClaimSubject{
		UserID:        "user-123",
		TenantCode:    "tenant-balotesti",
		InstitutionID: "inst-balotesti",
		Name:          "Tenant Operator",
		Email:         "operator@example.test",
		PhoneNumber:   "+40100000001",
		Locale:        "ro",
		EmailVerified: true,
		PhoneVerified: true,
		Roles:         []string{"admin"},
		PlatformRoles: []string{"platform_super_admin"},
		Permissions:   []string{"admin.read"},
		AuthzVersion:  7,
	}

	grant := &goidc.GrantInfo{ActiveScopes: "openid"}
	applyOIDCGrantClaimRelease(grant, subject, []string{"egueducation-api"})
	for _, key := range []string{"user_id", "tenant_code", "institution_id", "roles", "platform_roles", "permissions", "authz_version", "token_use"} {
		if _, ok := grant.AdditionalTokenClaims[key]; !ok {
			t.Fatalf("access token is missing authorization claim %q", key)
		}
	}
	for _, key := range []string{"name", "email", "phone_number", "locale", "email_verified", "phone_number_verified"} {
		if _, ok := grant.AdditionalTokenClaims[key]; ok {
			t.Fatalf("access token leaked identity claim %q", key)
		}
		if _, ok := grant.AdditionalIDTokenClaims[key]; ok {
			t.Fatalf("ID token released %q without its scope", key)
		}
		if _, ok := grant.AdditionalUserInfoClaims[key]; ok {
			t.Fatalf("UserInfo released %q without its scope", key)
		}
	}

	grant = &goidc.GrantInfo{ActiveScopes: "openid profile email phone"}
	applyOIDCGrantClaimRelease(grant, subject, []string{"egueducation-api"})
	for _, key := range []string{"name", "locale", "email", "email_verified", "phone_number", "phone_number_verified"} {
		if _, ok := grant.AdditionalIDTokenClaims[key]; !ok {
			t.Fatalf("ID token is missing scoped identity claim %q", key)
		}
		if _, ok := grant.AdditionalUserInfoClaims[key]; !ok {
			t.Fatalf("UserInfo is missing scoped identity claim %q", key)
		}
		if _, ok := grant.AdditionalTokenClaims[key]; ok {
			t.Fatalf("access token leaked scoped identity claim %q", key)
		}
	}
}

func TestOTPDeliveryKeepsSMSDefaultAndFailsClosedForUnconfiguredEmail(t *testing.T) {
	verified := oidcLoginUser{
		Email:               "operator@example.test",
		PhoneNumber:         "+40100000001",
		EmailVerified:       true,
		PhoneNumberVerified: true,
	}

	channel, err := resolveOTPDelivery(verified, normalizeOTPDeliveryChannel(""), nil, true)
	if err != nil || channel != otpDeliverySMS {
		t.Fatalf("fixture-backed default SMS selection = (%q, %v), want sms with no error", channel, err)
	}
	if _, err := resolveOTPDelivery(verified, otpDeliveryEmail, nil, false); err == nil || !strings.Contains(err.Error(), "email") {
		t.Fatalf("unconfigured verified-email OTP must fail closed with a clear email error, got %v", err)
	}
	if _, err := resolveOTPDelivery(oidcLoginUser{Email: verified.Email}, otpDeliveryEmail, nil, false); err == nil || !strings.Contains(err.Error(), "verificată") {
		t.Fatalf("unverified email must not be selected for OTP, got %v", err)
	}
}
