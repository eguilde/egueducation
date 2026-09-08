package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestOIDCPostgresIntegration exercises the actual provider with PostgreSQL,
// rather than a mocked issuer. It is deliberately opt-in locally: use an
// empty disposable database and set TEST_DATABASE_URL. CI supplies one via a
// PostgreSQL service and runs this test as a required job.
func TestOIDCPostgresIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("OIDC PostgreSQL integration test requires TEST_DATABASE_URL (use a disposable database)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()

	adminConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatalf("open integration admin database: %v", err)
	}
	// Cleanups execute in LIFO order. Register pool shutdown before any fixture
	// cleanup so tenant-scoped deletes can always use a live connection.
	t.Cleanup(adminPool.Close)
	if err := adminPool.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := appdb.ValidateSchemaContract(ctx, adminPool); err != nil {
		t.Fatalf("validate migrated schema: %v", err)
	}
	provisionOIDCIntegrationRole(t, ctx, adminPool)

	poolConfig := adminConfig.Copy()
	// A single physical connection makes a stale tenant-setting leak
	// deterministic if AcquireRequestConn ever fails to clear the session.
	poolConfig.MaxConns = 1
	poolConfig.MinConns = 0
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `set role egueducation_integration_app`)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	tenant.ConfigureResolver(pool, tenant.ResolverOptions{Environment: "test", BaseDomain: "egueducation.test"})

	var application http.Handler = http.NotFoundHandler()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		application.ServeHTTP(w, r)
	}))
	server.Start()
	defer server.Close()

	cfg := config.Config{
		Environment:              "test",
		FrontendOrigin:           server.URL,
		BackendURL:               server.URL,
		OIDCIssuer:               server.URL + "/api/oidc",
		OIDCClientID:             "oidc-postgres-integration-spa",
		OIDCDesktopClient:        "oidc-postgres-integration-desktop",
		OIDCAudience:             "egueducation-api",
		CustomerName:             "EguEducation",
		EnableSMSOTP:             true,
		EnableTestOTPFixture:     true,
		TestOTPFixtureCode:       "173829",
		TestOTPFixtureIdentifier: oidcTestFixtureIdentifier,
		TestOTPFixtureSubject:    oidcTestFixtureSubject,
		TestOTPFixtureTenantCode: "tenant-egueducation",
	}
	sessionPool := appdb.NewSessionPool(pool)
	service, err := NewService(cfg, nil, sessionPool)
	if err != nil {
		t.Fatalf("initialize OIDC provider: %v", err)
	}
	user, err := EnsureOIDCTestFixtureUser(ctx, pool, cfg)
	if err != nil {
		t.Fatalf("seed test OTP fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := removeOIDCIntegrationUser(pool, user.ID); err != nil {
			t.Errorf("remove test OTP fixture: %v", err)
		}
	})

	mux := http.NewServeMux()
	mux.Handle("/api/oidc/", http.StripPrefix("/api/oidc", service.OIDCHandler()))
	mux.Handle("/api/oidc", http.StripPrefix("/api/oidc", service.OIDCHandler()))
	mux.Handle("/api/me", service.RequireAuthenticated(http.HandlerFunc(service.SessionContext)))
	mux.HandleFunc("/api/oidc/session/logout", service.Logout)
	application = mux

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 20 * time.Second,
	}

	discoveryResponse, err := client.Get(cfg.OIDCIssuer + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatalf("load OIDC discovery: %v", err)
	}
	defer discoveryResponse.Body.Close()
	if discoveryResponse.StatusCode != http.StatusOK {
		t.Fatalf("OIDC discovery status = %d, want 200", discoveryResponse.StatusCode)
	}
	var discovery struct {
		RevocationEndpoint string `json:"revocation_endpoint"`
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	decodeJSONResponse(t, discoveryResponse.Body, &discovery)
	if discovery.RevocationEndpoint != cfg.OIDCIssuer+"/revoke" || discovery.EndSessionEndpoint != cfg.OIDCIssuer+"/session/end" {
		t.Fatalf("OIDC discovery did not publish RFC 7009/RP logout endpoints: %#v", discovery)
	}

	verifier := "oidc-postgres-integration-pkce-verifier-0123456789"
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	state := "oidc-postgres-integration-state"
	nonce := "oidc-postgres-integration-nonce"
	redirectURI := server.URL + "/auth/callback"
	authorizeURL := cfg.OIDCIssuer + "/authorize?" + url.Values{
		"client_id":             {cfg.OIDCClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}.Encode()

	methodsPage := getHTMLFollowingRedirects(t, client, authorizeURL)
	formAction := oidcFormAction(t, methodsPage, server.URL)
	otpIdentifierPage := postOIDCForm(t, client, formAction, url.Values{"method": {"otp"}})
	if !strings.Contains(otpIdentifierPage, `name="identifier"`) {
		t.Fatal("OIDC provider did not advance to the OTP identifier interaction")
	}
	if !strings.Contains(otpIdentifierPage, `name="delivery_channel"`) || !strings.Contains(otpIdentifierPage, `value="sms"`) {
		t.Fatal("OIDC provider did not offer the standards-compliant SMS delivery selection")
	}

	// The fixture signs in by its verified email identifier while explicitly
	// selecting its verified SMS identity. This exercises the same channel
	// selection and verification rules as a real user; it does not treat an
	// email identifier as an SMS-capable address.
	otpPage := postOIDCForm(t, client, formAction, url.Values{
		"identifier":       {user.Identifier},
		"delivery_channel": {"sms"},
	})
	if strings.Contains(otpPage, cfg.TestOTPFixtureCode) {
		t.Fatal("test OTP fixture must not be exposed in the browser")
	}
	otp := cfg.TestOTPFixtureCode
	consentPage := postOIDCForm(t, client, formAction, url.Values{"code": {otp}})
	if !strings.Contains(consentPage, `id="consentForm"`) {
		t.Fatal("OIDC provider did not advance to consent after valid loopback OTP")
	}

	consentResponse := postFormResponse(t, client, formAction, url.Values{
		"action":        {"allow"},
		"granted_scope": {"profile", "email", "offline_access"},
	})
	defer consentResponse.Body.Close()
	if consentResponse.StatusCode < http.StatusMultipleChoices || consentResponse.StatusCode >= http.StatusBadRequest {
		t.Fatalf("consent status = %d, want redirect to client callback", consentResponse.StatusCode)
	}
	callbackURL, err := consentResponse.Location()
	if err != nil {
		t.Fatalf("read OIDC callback location: %v", err)
	}
	callbackQuery := callbackURL.Query()
	if callbackURL.Scheme+"://"+callbackURL.Host+callbackURL.Path != redirectURI {
		t.Fatal("OIDC callback redirect URI differs from the registered URI")
	}
	if callbackQuery.Get("state") != state {
		t.Fatal("OIDC callback state does not match the authorization request")
	}
	code := callbackQuery.Get("code")
	if code == "" {
		t.Fatal("OIDC callback did not contain an authorization code")
	}

	tokenResponse := postFormResponse(t, client, cfg.OIDCIssuer+"/token", url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {cfg.OIDCClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	})
	defer tokenResponse.Body.Close()
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("authorization-code token exchange status = %d, want 200", tokenResponse.StatusCode)
	}
	refreshCookieSeen := false
	for _, cookie := range tokenResponse.Cookies() {
		if cookie.Name != "egueducation_rt" {
			continue
		}
		refreshCookieSeen = true
		if !cookie.HttpOnly || cookie.Path != "/api/oidc" || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("refresh cookie is not HttpOnly, path-scoped, and SameSite=Lax")
		}
	}
	if !refreshCookieSeen {
		t.Fatal("token exchange did not issue the HttpOnly refresh cookie")
	}
	var tokenPayload map[string]json.RawMessage
	decodeJSONResponse(t, tokenResponse.Body, &tokenPayload)
	if _, exposed := tokenPayload["refresh_token"]; exposed {
		t.Fatal("token endpoint exposed refresh_token in JSON instead of only the HttpOnly cookie")
	}
	var accessToken, idToken string
	if err := json.Unmarshal(tokenPayload["access_token"], &accessToken); err != nil || accessToken == "" {
		t.Fatal("token endpoint did not return an access token")
	}
	if err := json.Unmarshal(tokenPayload["id_token"], &idToken); err != nil || idToken == "" {
		t.Fatal("token endpoint did not return an ID token")
	}
	if got := jwtStringClaim(t, idToken, "nonce"); got != nonce {
		t.Fatal("ID token nonce does not match the authorization request")
	}
	for _, claim := range []string{"name", "email", "locale"} {
		if jwtStringClaim(t, idToken, claim) == "" {
			t.Fatalf("ID token is missing granted identity claim %q", claim)
		}
		if jwtStringClaim(t, accessToken, claim) != "" {
			t.Fatalf("access token leaked identity claim %q", claim)
		}
	}
	if jwtStringClaim(t, idToken, "phone_number") != "" || jwtStringClaim(t, accessToken, "phone_number") != "" {
		t.Fatal("phone_number was released without the phone scope")
	}
	if got := jwtStringClaim(t, accessToken, "tenant_id"); got != "tenant-egueducation" {
		t.Fatalf("access token tenant_id = %q, want tenant-egueducation", got)
	}
	if got := jwtStringClaim(t, accessToken, "tenant_code"); got != "tenant-egueducation" {
		t.Fatalf("access token tenant_code = %q, want tenant-egueducation", got)
	}
	accessRoles := jwtStringArrayClaim(t, accessToken, "roles")
	accessPlatformRoles := jwtStringArrayClaim(t, accessToken, "platform_roles")
	accessPermissions := jwtStringArrayClaim(t, accessToken, "permissions")
	if len(accessRoles) == 0 || len(accessPermissions) == 0 {
		t.Fatal("access token must contain effective tenant roles and permissions")
	}
	if accessPlatformRoles == nil {
		t.Fatal("access token must encode platform_roles as an array")
	}
	if jwtStringClaim(t, accessToken, "jti") == "" || jwtStringClaim(t, accessToken, "sid") == "" || jwtStringClaim(t, accessToken, "acr") == "" {
		t.Fatal("access token must contain jti, sid and acr")
	}
	if len(jwtStringArrayClaim(t, accessToken, "amr")) == 0 {
		t.Fatal("access token must contain at least one authentication method")
	}
	if got := jwtInt64Claim(t, accessToken, "authz_version"); got <= 0 {
		t.Fatalf("access token authz_version = %d, want positive value", got)
	}
	// Authorization codes live in authn sessions and are consumed by a
	// successful token exchange. The durable post-exchange state is the
	// tenant-bound grant session containing the refresh token. Do not assert
	// that the consumed authn session remains stored.
	var consumedAuthCodeCount int
	if err := adminPool.QueryRow(ctx, `
		select count(*)
		from oidc_authn_sessions
		where tenant_code = 'tenant-egueducation'
			and data->>'auth_code' = $1
	`, code).Scan(&consumedAuthCodeCount); err != nil {
		t.Fatalf("query consumed tenant-bound OIDC authorization code: %v", err)
	}
	if consumedAuthCodeCount != 0 {
		t.Fatal("authorization code authn session remained after successful exchange")
	}
	var tenantBoundRefreshGrantCount int
	if err := adminPool.QueryRow(ctx, `
		select count(*)
		from oidc_grant_sessions
		where tenant_code = 'tenant-egueducation'
			and data->>'sub' = $1
			and coalesce(data->>'refresh_token', '') <> ''
	`, user.ID.String()).Scan(&tenantBoundRefreshGrantCount); err != nil {
		t.Fatalf("query tenant-bound OIDC refresh grant: %v", err)
	}
	if tenantBoundRefreshGrantCount == 0 {
		t.Fatal("refresh grant was not persisted under the resolved tenant")
	}

	// The same opaque protocol identifier may safely exist in two tenant
	// namespaces. Before migration 0084 the shared sentinel tenant_id made this
	// an overwrite; now tenant_code is the primary conflict boundary.
	const storageBoundaryID = "oidc-integration-cross-tenant-id"
	for _, tenantCode := range []string{"tenant-egueducation", "tenant-balotesti"} {
		if _, err := adminPool.Exec(ctx, `
			insert into oidc_authn_sessions (tenant_id, tenant_code, id, data, expires_at)
			values ($1::uuid, $2, $3, '{}'::jsonb, now() + interval '5 minutes')
		`, localOIDCTenantID, tenantCode, storageBoundaryID); err != nil {
			t.Fatalf("insert tenant-bound OIDC authn session for %s: %v", tenantCode, err)
		}
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), `delete from oidc_authn_sessions where id = $1`, storageBoundaryID)
	})
	var storageBoundaryCount int
	if err := adminPool.QueryRow(ctx, `select count(*) from oidc_authn_sessions where id = $1`, storageBoundaryID).Scan(&storageBoundaryCount); err != nil {
		t.Fatalf("count tenant-bound OIDC authn sessions: %v", err)
	}
	if storageBoundaryCount != 2 {
		t.Fatalf("same OIDC authn id must remain isolated per tenant, got %d rows", storageBoundaryCount)
	}

	meRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/me", nil)
	if err != nil {
		t.Fatalf("create /api/me request: %v", err)
	}
	meRequest.Header.Set("Authorization", "Bearer "+accessToken)
	meResponse, err := client.Do(meRequest)
	if err != nil {
		t.Fatalf("call /api/me: %v", err)
	}
	defer meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("/api/me status = %d, want 200", meResponse.StatusCode)
	}
	var me SessionContext
	decodeJSONResponse(t, meResponse.Body, &me)
	if me.InstitutionID != "inst-001" || me.User.Sub != user.Subject {
		t.Fatal("/api/me did not resolve the host and authenticated tenant membership")
	}
	if me.User.Roles == nil || me.Permissions == nil || me.Modules == nil || me.Authentication == nil || me.GDPRCapabilities == nil {
		t.Fatal("/api/me must encode collection fields as JSON arrays, never null")
	}
	if len(me.Modules) == 0 {
		t.Fatal("browser fixture must be assigned active modules for full UI coverage")
	}
	if me.TenantCode != "tenant-egueducation" || me.AuthzVersion <= 0 {
		t.Fatal("/api/me did not expose the tenant authorization version")
	}
	if !sameStringSet(accessRoles, me.User.Roles) || !sameStringSet(accessPlatformRoles, me.PlatformRoles) || !sameStringSet(accessPermissions, me.Permissions) {
		t.Fatal("access token and /api/me must expose the same effective authorization")
	}

	mismatchRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/me", nil)
	if err != nil {
		t.Fatalf("create tenant-mismatch /api/me request: %v", err)
	}
	// Host selection is server-side tenant selection. The same bearer token must
	// never gain a Balotesti session merely because a caller changes its Host.
	mismatchRequest.Host = "scoalabalotesti.egueducation.test"
	mismatchRequest.Header.Set("Authorization", "Bearer "+accessToken)
	mismatchResponse, err := client.Do(mismatchRequest)
	if err != nil {
		t.Fatalf("call tenant-mismatch /api/me: %v", err)
	}
	mismatchResponse.Body.Close()
	if mismatchResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("cross-tenant /api/me status = %d, want 401", mismatchResponse.StatusCode)
	}

	// RFC 7009 must not reinterpret the sentinel browser value as the refresh
	// cookie. An unknown literal token receives the required 200 response, and
	// the following cookie refresh proves the real grant remained usable.
	revokeCookieLiteral := postFormResponse(t, client, cfg.OIDCIssuer+"/revoke", url.Values{
		"client_id": {cfg.OIDCClientID},
		"token":     {"cookie"},
	})
	defer revokeCookieLiteral.Body.Close()
	if revokeCookieLiteral.StatusCode != http.StatusOK {
		t.Fatalf("RFC 7009 literal token status = %d, want 200", revokeCookieLiteral.StatusCode)
	}

	refreshResponse := postFormResponse(t, client, cfg.OIDCIssuer+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.OIDCClientID},
		"refresh_token": {"cookie"},
	})
	defer refreshResponse.Body.Close()
	if refreshResponse.StatusCode != http.StatusOK {
		t.Fatalf("cookie refresh status = %d, want 200", refreshResponse.StatusCode)
	}
	var refreshed map[string]json.RawMessage
	decodeJSONResponse(t, refreshResponse.Body, &refreshed)
	if _, exposed := refreshed["refresh_token"]; exposed {
		t.Fatal("refresh response exposed refresh_token in JSON")
	}

	invalidRPLogout, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.OIDCIssuer+"/session/end?"+url.Values{
		"id_token_hint":            {idToken},
		"post_logout_redirect_uri": {"https://attacker.example/logout"},
		"state":                    {"must-not-redirect"},
	}.Encode(), nil)
	if err != nil {
		t.Fatalf("create invalid RP logout request: %v", err)
	}
	invalidRPLogoutResponse, err := client.Do(invalidRPLogout)
	if err != nil {
		t.Fatalf("invalid RP logout request: %v", err)
	}
	invalidRPLogoutResponse.Body.Close()
	if invalidRPLogoutResponse.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid RP logout redirect status = %d, want 400", invalidRPLogoutResponse.StatusCode)
	}

	rpState := "oidc-rp-logout-state"
	rpLogout, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.OIDCIssuer+"/session/end?"+url.Values{
		"id_token_hint":            {idToken},
		"post_logout_redirect_uri": {server.URL + "/auth/logout"},
		"state":                    {rpState},
		"client_id":                {cfg.OIDCClientID},
	}.Encode(), nil)
	if err != nil {
		t.Fatalf("create RP logout request: %v", err)
	}
	rpLogoutResponse, err := client.Do(rpLogout)
	if err != nil {
		t.Fatalf("RP logout request: %v", err)
	}
	defer rpLogoutResponse.Body.Close()
	if rpLogoutResponse.StatusCode < http.StatusMultipleChoices || rpLogoutResponse.StatusCode >= http.StatusBadRequest {
		t.Fatalf("RP logout status = %d, want redirect", rpLogoutResponse.StatusCode)
	}
	rpLocation, err := rpLogoutResponse.Location()
	if err != nil {
		t.Fatalf("read RP logout redirect: %v", err)
	}
	if rpLocation.String() != server.URL+"/auth/logout?state="+rpState {
		t.Fatalf("RP logout redirect = %q, want exact registered redirect with state", rpLocation.String())
	}
	if !hasExpiredRefreshCookie(rpLogoutResponse.Cookies()) {
		t.Fatal("RP logout did not expire the refresh cookie before redirect")
	}

	logoutRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/oidc/session/logout", nil)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}
	logoutRequest.Header.Set("Authorization", "Bearer "+accessToken)
	logoutResponse, err := client.Do(logoutRequest)
	if err != nil {
		t.Fatalf("logout request: %v", err)
	}
	defer logoutResponse.Body.Close()
	if logoutResponse.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d, want 200", logoutResponse.StatusCode)
	}
	if !hasExpiredRefreshCookie(logoutResponse.Cookies()) {
		t.Fatal("logout did not expire the refresh cookie")
	}

	revokedRefresh := postFormResponse(t, client, cfg.OIDCIssuer+"/token", url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {cfg.OIDCClientID},
		"refresh_token": {"cookie"},
	})
	defer revokedRefresh.Body.Close()
	if revokedRefresh.StatusCode < http.StatusBadRequest {
		t.Fatalf("refresh after logout status = %d, want OAuth error", revokedRefresh.StatusCode)
	}

	assertPooledTenantIsolation(t, ctx, pool, user.ID)
}

func provisionOIDCIntegrationRole(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `create role egueducation_integration_app nologin nosuperuser nocreatedb nocreaterole noinherit nobypassrls`); err != nil {
		t.Fatalf("create non-bypass integration role: %v", err)
	}
	for _, statement := range []string{
		`grant usage on schema public to egueducation_integration_app`,
		`grant select, insert, update, delete on all tables in schema public to egueducation_integration_app`,
		`grant usage, select on all sequences in schema public to egueducation_integration_app`,
		`grant execute on all functions in schema public to egueducation_integration_app`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant integration role privileges: %v", err)
		}
	}
}

const (
	oidcTestFixtureIdentifier = "oidc.browser.fixture@example.test"
	oidcTestFixtureSubject    = "oidc-browser-fixture-subject"
)

func removeOIDCIntegrationUser(pool *pgxpool.Pool, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin fixture cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		select
			set_config('app.tenant_id', 'tenant-egueducation', true),
			set_config('app.is_super_admin', 'true', true)
	`); err != nil {
		return fmt.Errorf("scope fixture cleanup: %w", err)
	}
	if _, err := tx.Exec(ctx, `delete from app_users where id = $1`, userID); err != nil {
		return fmt.Errorf("delete fixture user: %w", err)
	}
	var remainingAuthorizationVersions int
	if err := tx.QueryRow(ctx, `
		select count(*)
		from app_tenant_authorization_versions
		where tenant_code = 'tenant-egueducation' and user_id = $1
	`, userID).Scan(&remainingAuthorizationVersions); err != nil {
		return fmt.Errorf("verify fixture authorization cleanup: %w", err)
	}
	if remainingAuthorizationVersions != 0 {
		return fmt.Errorf("fixture cleanup recreated %d authorization version rows", remainingAuthorizationVersions)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit fixture cleanup: %w", err)
	}
	return nil
}

func getHTMLFollowingRedirects(t *testing.T, client *http.Client, rawURL string) string {
	t.Helper()
	for redirects := 0; redirects < 8; redirects++ {
		response, err := client.Get(rawURL)
		if err != nil {
			t.Fatalf("OIDC GET interaction: %v", err)
		}
		if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < http.StatusBadRequest {
			location, err := response.Location()
			response.Body.Close()
			if err != nil {
				t.Fatalf("read OIDC redirect: %v", err)
			}
			rawURL = location.String()
			continue
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			t.Fatalf("OIDC interaction status = %d, want 200", response.StatusCode)
		}
		body := readResponse(t, response.Body)
		response.Body.Close()
		return body
	}
	t.Fatal("OIDC interaction exceeded redirect limit")
	return ""
}

func postOIDCForm(t *testing.T, client *http.Client, action string, values url.Values) string {
	t.Helper()
	response := postFormResponse(t, client, action, values)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("OIDC interaction POST status = %d, want 200", response.StatusCode)
	}
	return readResponse(t, response.Body)
}

func postFormResponse(t *testing.T, client *http.Client, action string, values url.Values) *http.Response {
	t.Helper()
	response, err := client.PostForm(action, values)
	if err != nil {
		t.Fatalf("OIDC POST interaction: %v", err)
	}
	return response
}

var oidcActionPattern = regexp.MustCompile(`<form[^>]+action="([^"]+)"`)

func oidcFormAction(t *testing.T, page, baseURL string) string {
	t.Helper()
	matches := oidcActionPattern.FindStringSubmatch(page)
	if len(matches) != 2 {
		t.Fatal("OIDC login page did not contain an interaction form action")
	}
	action, err := url.Parse(strings.ReplaceAll(matches[1], "&amp;", "&"))
	if err != nil {
		t.Fatalf("parse OIDC form action: %v", err)
	}
	base, _ := url.Parse(baseURL)
	return base.ResolveReference(action).String()
}

func decodeJSONResponse(t *testing.T, body io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
}

func readResponse(t *testing.T, body io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		t.Fatalf("read HTTP response: %v", err)
	}
	return string(data)
}

func jwtStringClaim(t *testing.T, token, name string) string {
	t.Helper()
	claims := jwtClaims(t, token)
	value, _ := claims[name].(string)
	return value
}

func jwtStringArrayClaim(t *testing.T, token, name string) []string {
	t.Helper()
	claims := jwtClaims(t, token)
	raw, _ := claims[name].([]any)
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if !ok {
			t.Fatalf("JWT claim %q contains a non-string value", name)
		}
		values = append(values, value)
	}
	return values
}

func jwtInt64Claim(t *testing.T, token, name string) int64 {
	t.Helper()
	value, ok := jwtClaims(t, token)[name].(float64)
	if !ok {
		t.Fatalf("JWT claim %q is not numeric", name)
	}
	return int64(value)
}

func jwtClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("token is not a compact JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode token claims: %v", err)
	}
	return claims
}

func hasExpiredRefreshCookie(cookies []*http.Cookie) bool {
	for _, cookie := range cookies {
		if cookie.Name == "egueducation_rt" && cookie.MaxAge < 0 {
			return true
		}
	}
	return false
}

func assertPooledTenantIsolation(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	sessionPool := appdb.NewSessionPool(pool)
	firstCtx, firstRelease, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001", InstitutionName: "EguEducation", ActorSubject: "oidc-integration",
	})
	if err != nil {
		t.Fatalf("acquire first tenant-scoped connection: %v", err)
	}
	var visibleInMemberTenant int
	err = sessionPool.QueryRow(firstCtx, `select count(*) from app_memberships where user_id = $1 and active`, userID).Scan(&visibleInMemberTenant)
	firstRelease()
	if err != nil || visibleInMemberTenant != 1 {
		t.Fatalf("member tenant RLS visibility = %d, err = %v, want 1", visibleInMemberTenant, err)
	}

	secondCtx, secondRelease, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{
		TenantID: "tenant-balotesti", InstitutionID: "inst-balotesti", InstitutionName: "Balotesti", ActorSubject: "oidc-integration",
	})
	if err != nil {
		t.Fatalf("acquire second tenant-scoped connection: %v", err)
	}
	var visibleInOtherTenant int
	err = sessionPool.QueryRow(secondCtx, `select count(*) from app_memberships where user_id = $1 and active`, userID).Scan(&visibleInOtherTenant)
	secondRelease()
	if err != nil || visibleInOtherTenant != 0 {
		t.Fatalf("other tenant RLS visibility = %d, err = %v, want 0", visibleInOtherTenant, err)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("reacquire pooled connection: %v", err)
	}
	defer conn.Release()
	var tenantID, institutionID, bypass string
	err = conn.QueryRow(ctx, `select current_setting('app.tenant_id', true), current_setting('app.institution_id', true), current_setting('app.is_super_admin', true)`).Scan(&tenantID, &institutionID, &bypass)
	if err != nil {
		t.Fatalf("inspect released pooled session: %v", err)
	}
	if tenantID != "" || institutionID != "" || bypass != "" {
		t.Fatal("pooled PostgreSQL connection retained tenant session state after release")
	}
}
