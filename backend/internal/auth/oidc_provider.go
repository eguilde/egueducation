package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/notification"
	"github.com/eguilde/egueducation/internal/tenant"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luikyv/go-oidc/pkg/goidc"
	"github.com/luikyv/go-oidc/pkg/provider"
)

var localOIDCTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func newOIDCProviderHandler(
	db *pgxpool.Pool,
	cfg *config.Config,
	smsService *notification.SMSService,
	passkeyRedeemNonce func(string) (string, bool),
) (http.Handler, *JWTVerifier, error) {
	keyManager := newKeyManager(db, cfg)
	if err := keyManager.Init(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("oidc key manager init: %w", err)
	}
	keyManager.StartRotationCheck(context.Background())

	if err := seedOIDCClients(context.Background(), db, cfg); err != nil {
		return nil, nil, fmt.Errorf("seed oidc clients: %w", err)
	}

	clientStore := &oidcClientStore{db: db}
	authnStore := &oidcAuthnSessionStore{db: db}
	grantStore := &oidcGrantSessionStore{db: db}
	otp := newOTPService(db)

	op, err := provider.New(
		goidc.ProfileOpenID,
		cfg.OIDCIssuer,
		func(context.Context) (goidc.JSONWebKeySet, error) {
			return keyManager.ActiveJWKS(), nil
		},
		provider.WithAuthorizationCodeGrant(),
		provider.WithRefreshTokenGrant(
			func(context.Context, *goidc.Client, goidc.GrantInfo) bool { return true },
			86400,
		),
		provider.WithPKCE(goidc.CodeChallengeMethodSHA256),
		provider.WithIDTokenSignatureAlgs(goidc.RS256),
		provider.WithTokenOptions(func(context.Context, goidc.GrantInfo, *goidc.Client) goidc.TokenOptions {
			return goidc.NewJWTTokenOptions(goidc.RS256, 3600)
		}),
		provider.WithTokenAuthnMethods(goidc.ClientAuthnNone, goidc.ClientAuthnSecretBasic),
		provider.WithScopes(goidc.ScopeOpenID, goidc.ScopeProfile, goidc.ScopeEmail, goidc.ScopePhone, goidc.ScopeOfflineAccess),
		provider.WithClientStorage(clientStore),
		provider.WithAuthnSessionStorage(authnStore),
		provider.WithGrantSessionStorage(grantStore),
		provider.WithPolicies(buildLoginPolicy(db, cfg, smsService, otp, passkeyRedeemNonce)),
		provider.WithHandleGrantFunc(buildGrantClaimsEnricher(db, cfg)),
		provider.WithDCR(buildDCRHandler(cfg), nil),
		provider.WithNotifyErrorFunc(func(_ context.Context, err error) {
			// Intentionally silent here; the app logger already wraps initialization errors.
			_ = err
		}),
		provider.WithRenderErrorFunc(buildProviderErrorRenderer(cfg)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("provider.New: %w", err)
	}

	handler := wrapRegisterPage(wrapLogoutPage(wrapRefreshTokenCookie(op.Handler(), cfg), cfg), cfg)
	verifier := NewJWTVerifier(cfg.OIDCIssuer, strings.TrimRight(cfg.OIDCIssuer, "/")+"/jwks", cfg.OIDCAudience)
	verifier.loader = func(context.Context) (*jose.JSONWebKeySet, error) {
		active := keyManager.ActiveJWKS()
		keys := make([]jose.JSONWebKey, 0, len(active.Keys))
		for _, key := range active.Keys {
			keys = append(keys, jose.JSONWebKey{
				KeyID:     key.KeyID,
				Key:       key.Key,
				Algorithm: key.Algorithm,
				Use:       key.Use,
			})
		}
		return &jose.JSONWebKeySet{Keys: keys}, nil
	}
	return handler, verifier, nil
}

func seedOIDCClients(ctx context.Context, db *pgxpool.Pool, cfg *config.Config) error {
	origins := append([]string{cfg.FrontendOrigin}, cfg.FrontendOrigins...)
	redirects := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}
		redirect := origin + "/auth/callback"
		if _, ok := seen[redirect]; ok {
			continue
		}
		seen[redirect] = struct{}{}
		redirects = append(redirects, redirect)
	}

	clients := []struct {
		ID           string
		Name         string
		RedirectURIs []string
		AppType      string
	}{
		{
			ID:           cfg.OIDCClientID,
			Name:         "EguEducation SPA",
			RedirectURIs: redirects,
			AppType:      "web",
		},
		{
			ID:           cfg.OIDCDesktopClient,
			Name:         "EguEducation Desktop",
			RedirectURIs: []string{"egueducation://callback", "http://localhost:4300/callback"},
			AppType:      "native",
		},
	}

	for _, client := range clients {
		metadata := map[string]any{
			"client_id":                  client.ID,
			"client_name":                client.Name,
			"grant_types":                []string{"authorization_code", "refresh_token"},
			"response_types":             []string{"code"},
			"redirect_uris":              client.RedirectURIs,
			"scope":                      "openid profile email phone offline_access",
			"token_endpoint_auth_method": "none",
			"application_type":           client.AppType,
		}
		payload, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal client %q: %w", client.ID, err)
		}
		_, err = db.Exec(ctx, `
			insert into oidc_clients (client_id, client_name, public_client, require_pkce, active, data, created_at, updated_at)
			values ($1, $2, true, true, true, $3::jsonb, now(), now())
			on conflict (client_id) do update
			set client_name = excluded.client_name,
				public_client = excluded.public_client,
				require_pkce = excluded.require_pkce,
				active = excluded.active,
				data = excluded.data,
				updated_at = now()
		`, client.ID, client.Name, string(payload))
		if err != nil {
			return fmt.Errorf("upsert client %q: %w", client.ID, err)
		}
	}

	return nil
}

type oidcClientStore struct {
	db *pgxpool.Pool
}

func (s *oidcClientStore) Save(ctx context.Context, client *goidc.Client) error {
	payload, err := json.Marshal(client)
	if err != nil {
		return fmt.Errorf("marshal oidc client: %w", err)
	}
	_, err = s.db.Exec(ctx, `
		insert into oidc_clients (client_id, client_name, public_client, require_pkce, active, data, created_at, updated_at)
		values ($1, coalesce($2, $1), true, true, true, $3::jsonb, now(), now())
		on conflict (client_id) do update
		set data = excluded.data,
			updated_at = now()
	`, client.ID, client.Name, string(payload))
	return err
}

func (s *oidcClientStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `delete from oidc_clients where client_id = $1`, id)
	return err
}

func (s *oidcClientStore) Client(ctx context.Context, id string) (*goidc.Client, error) {
	var payload []byte
	err := s.db.QueryRow(ctx, `
		select data
		from oidc_clients
		where client_id = $1 and active = true
	`, id).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	var client goidc.Client
	if err := json.Unmarshal(payload, &client); err != nil {
		return nil, fmt.Errorf("unmarshal oidc client: %w", err)
	}
	return &client, nil
}

func (s *oidcClientStore) Clients(ctx context.Context) ([]*goidc.Client, error) {
	rows, err := s.db.Query(ctx, `
		select data
		from oidc_clients
		where active = true
		order by client_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]*goidc.Client, 0, 8)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var client goidc.Client
		if err := json.Unmarshal(payload, &client); err != nil {
			return nil, err
		}
		clients = append(clients, &client)
	}
	return clients, rows.Err()
}

type oidcAuthnSessionStore struct {
	db *pgxpool.Pool
}

func (s *oidcAuthnSessionStore) Save(ctx context.Context, session *goidc.AuthnSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal authn session: %w", err)
	}
	_, err = s.db.Exec(ctx, `
		insert into oidc_authn_sessions (tenant_id, id, data, expires_at)
		values ($1::uuid, $2, $3::jsonb, $4)
		on conflict (tenant_id, id) do update
		set data = excluded.data,
			expires_at = excluded.expires_at
	`, localOIDCTenantID, session.ID, string(payload), time.Now().Add(10*time.Minute))
	return err
}

func (s *oidcAuthnSessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		delete from oidc_authn_sessions
		where tenant_id = $1::uuid and id = $2
	`, localOIDCTenantID, id)
	return err
}

func (s *oidcAuthnSessionStore) SessionByCallbackID(ctx context.Context, id string) (*goidc.AuthnSession, error) {
	return s.query(ctx, "callback_id", id)
}

func (s *oidcAuthnSessionStore) SessionByAuthCode(ctx context.Context, code string) (*goidc.AuthnSession, error) {
	return s.query(ctx, "auth_code", code)
}

func (s *oidcAuthnSessionStore) SessionByPushedAuthReqID(ctx context.Context, id string) (*goidc.AuthnSession, error) {
	return s.query(ctx, "pushed_auth_req_id", id)
}

func (s *oidcAuthnSessionStore) SessionByCIBAAuthID(ctx context.Context, id string) (*goidc.AuthnSession, error) {
	return s.query(ctx, "ciba_auth_req_id", id)
}

func (s *oidcAuthnSessionStore) query(ctx context.Context, field, value string) (*goidc.AuthnSession, error) {
	var payload []byte
	err := s.db.QueryRow(ctx, `
		select data
		from oidc_authn_sessions
		where tenant_id = $1::uuid
			and data->>$2 = $3
			and expires_at > now()
	`, localOIDCTenantID, field, value).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	var session goidc.AuthnSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("unmarshal authn session: %w", err)
	}
	return &session, nil
}

type oidcGrantSessionStore struct {
	db *pgxpool.Pool
}

func (s *oidcGrantSessionStore) Save(ctx context.Context, grant *goidc.GrantSession) error {
	payload, err := json.Marshal(grant)
	if err != nil {
		return fmt.Errorf("marshal grant session: %w", err)
	}
	_, err = s.db.Exec(ctx, `
		insert into oidc_grant_sessions (tenant_id, id, data, expires_at)
		values ($1::uuid, $2, $3::jsonb, $4)
		on conflict (tenant_id, id) do update
		set data = excluded.data,
			expires_at = excluded.expires_at
	`, localOIDCTenantID, grant.ID, string(payload), time.Unix(int64(grant.ExpiresAtTimestamp), 0))
	return err
}

func (s *oidcGrantSessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `
		delete from oidc_grant_sessions
		where tenant_id = $1::uuid and id = $2
	`, localOIDCTenantID, id)
	return err
}

func (s *oidcGrantSessionStore) DeleteByAuthCode(ctx context.Context, code string) error {
	_, err := s.db.Exec(ctx, `
		delete from oidc_grant_sessions
		where tenant_id = $1::uuid
			and data->>'authorization_code' = $2
	`, localOIDCTenantID, code)
	return err
}

func (s *oidcGrantSessionStore) SessionByTokenID(ctx context.Context, tokenID string) (*goidc.GrantSession, error) {
	return s.query(ctx, "token_id", tokenID)
}

func (s *oidcGrantSessionStore) SessionByRefreshToken(ctx context.Context, token string) (*goidc.GrantSession, error) {
	return s.query(ctx, "refresh_token", token)
}

func (s *oidcGrantSessionStore) query(ctx context.Context, field, value string) (*goidc.GrantSession, error) {
	var payload []byte
	err := s.db.QueryRow(ctx, `
		select data
		from oidc_grant_sessions
		where tenant_id = $1::uuid
			and data->>$2 = $3
			and expires_at > now()
	`, localOIDCTenantID, field, value).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	var grant goidc.GrantSession
	if err := json.Unmarshal(payload, &grant); err != nil {
		return nil, fmt.Errorf("unmarshal grant session: %w", err)
	}
	return &grant, nil
}

func buildGrantClaimsEnricher(db *pgxpool.Pool, cfg *config.Config) goidc.HandleGrantFunc {
	return func(r *http.Request, grant *goidc.GrantInfo) error {
		userID, err := uuid.Parse(strings.TrimSpace(grant.Subject))
		if err != nil || userID == uuid.Nil {
			return nil
		}

		var (
			subject              string
			name                 string
			email                string
			phone                string
			locale               string
			emailVerified        bool
			phoneVerified        bool
			preferredInstitution string
		)
		err = db.QueryRow(r.Context(), `
			select
				sub,
				name,
				email,
				phone_number,
				locale,
				email_verified,
				phone_number_verified,
				sc.institution_id
			from app_users u
			join app_session_context sc on sc.user_id = u.id
			where u.id = $1::uuid and u.status = 'active'
		`, userID).Scan(&subject, &name, &email, &phone, &locale, &emailVerified, &phoneVerified, &preferredInstitution)
		if err != nil {
			return nil
		}

		roles, _ := loadRolesForSubject(r.Context(), db, subject)
		audience := tokenAudiences(grant, cfg.OIDCAudience)

		if grant.AdditionalTokenClaims == nil {
			grant.AdditionalTokenClaims = make(map[string]any)
		}
		if grant.AdditionalIDTokenClaims == nil {
			grant.AdditionalIDTokenClaims = make(map[string]any)
		}
		if grant.AdditionalUserInfoClaims == nil {
			grant.AdditionalUserInfoClaims = make(map[string]any)
		}

		shared := map[string]any{
			"user_id":               userID.String(),
			"tenant_id":             localOIDCTenantID.String(),
			"institution_id":        preferredInstitution,
			"aud":                   audience,
			"name":                  name,
			"email":                 email,
			"email_verified":        emailVerified,
			"phone_number":          phone,
			"phone_number_verified": phoneVerified,
			"locale":                locale,
			"roles":                 roles,
		}
		for key, value := range shared {
			grant.AdditionalTokenClaims[key] = value
			grant.AdditionalIDTokenClaims[key] = value
			grant.AdditionalUserInfoClaims[key] = value
		}
		return nil
	}
}

func tokenAudiences(grant *goidc.GrantInfo, configuredAudience string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	add(configuredAudience)
	if grant != nil {
		for _, resource := range grant.ActiveResources {
			add(string(resource))
		}
		if len(out) == 0 {
			add(grant.ClientID)
		}
	}
	return out
}

func buildDCRHandler(cfg *config.Config) goidc.HandleDynamicClientFunc {
	known := map[string]bool{
		cfg.OIDCClientID:      true,
		cfg.OIDCDesktopClient: true,
	}
	return func(_ *http.Request, id string, _ *goidc.ClientMeta) error {
		if cfg.IsProduction() && !known[id] {
			return fmt.Errorf("oidc: DCR not permitted for client %q in production", id)
		}
		return nil
	}
}

func buildProviderErrorRenderer(cfg *config.Config) goidc.RenderErrorFunc {
	tmpl := template.Must(template.New("oidc-error").Parse(oidcErrorHTML))
	return func(w http.ResponseWriter, r *http.Request, err error) error {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return tmpl.Execute(w, map[string]string{
			"CustomerName": tenant.ResolveBranding(r.Host, cfg.CustomerName, "").Name,
			"Error":        "Sesiunea de autentificare a expirat sau cererea OIDC este invalidă. Vă rugăm să încercați din nou.",
			"Detail":       err.Error(),
		})
	}
}

func buildLoginPolicy(
	db *pgxpool.Pool,
	cfg *config.Config,
	smsService *notification.SMSService,
	otp *otpService,
	passkeyRedeemNonce func(string) (string, bool),
) goidc.AuthnPolicy {
	tmpl := template.Must(template.New("login").Parse(oidcLoginHTML))

	return goidc.NewPolicy(
		"login",
		func(_ *http.Request, _ *goidc.Client, _ *goidc.AuthnSession) bool { return true },
		func(w http.ResponseWriter, r *http.Request, sess *goidc.AuthnSession) (goidc.Status, error) {
			if err := r.ParseForm(); err != nil {
				return goidc.StatusFailure, err
			}
			if r.FormValue("action") == "abort" {
				return goidc.StatusFailure, nil
			}

			step, _ := sess.StoredParameter("step").(string)
			formAction := strings.TrimRight(cfg.OIDCIssuer, "/") + "/authorize/" + sess.CallbackID + "/login"

			switch step {
			case "":
				return renderMethodStep(w, r, sess, db, cfg, smsService, tmpl, formAction, otp, passkeyRedeemNonce)
			case "otp_identifier":
				return renderOTPIdentifierStep(w, r, sess, db, cfg, smsService, tmpl, formAction, otp)
			case "otp":
				return renderOTPStep(w, r, sess, cfg, tmpl, formAction, otp)
			case "consent":
				return renderConsentStep(w, r, sess, cfg, tmpl, formAction)
			default:
				return renderMethodStep(w, r, sess, db, cfg, smsService, tmpl, formAction, otp, passkeyRedeemNonce)
			}
		},
	)
}

func renderMethodStep(
	w http.ResponseWriter,
	r *http.Request,
	sess *goidc.AuthnSession,
	db *pgxpool.Pool,
	cfg *config.Config,
	smsService *notification.SMSService,
	tmpl *template.Template,
	formAction string,
	otp *otpService,
	passkeyRedeemNonce func(string) (string, bool),
) (goidc.Status, error) {
	customerName := tenant.ResolveBranding(r.Host, cfg.CustomerName, "").Name
	data := oidcLoginData{
		Step:           "methods",
		StepLabel:      "Pasul 1",
		CustomerName:   customerName,
		FormAction:     formAction,
		FrontendOrigin: cfg.FrontendOrigin,
		WalletEnabled:  cfg.EnableWallet,
		Theme:          resolveOIDCThemeSettings(r, sess),
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}

	switch r.FormValue("method") {
	case "otp":
		sess.StoreParameter("step", "otp_identifier")
		return renderOIDCStep(w, tmpl, oidcLoginData{
			Step:           "otp_identifier",
			StepLabel:      "Pasul 2",
			CustomerName:   customerName,
			FormAction:     formAction,
			FrontendOrigin: cfg.FrontendOrigin,
			WalletEnabled:  cfg.EnableWallet,
			Theme:          resolveOIDCThemeSettings(r, sess),
		})
	case "eudi_wallet":
		data.Error = "Autentificarea cu EUDI Wallet nu este disponibilă încă în acest flux."
		return renderOIDCStep(w, tmpl, data)
	case "passkey_done":
		if passkeyRedeemNonce == nil {
			data.Error = "Autentificarea cu passkey nu este disponibilă."
			return renderOIDCStep(w, tmpl, data)
		}
		userID, ok := passkeyRedeemNonce(strings.TrimSpace(r.FormValue("nonce")))
		if !ok || userID == "" {
			data.Error = "Sesiunea passkey a expirat. Reîncercați."
			return renderOIDCStep(w, tmpl, data)
		}
		sess.SetUserID(userID)
		sess.StoreParameter("step", "consent")
		return renderConsentStep(w, r, sess, cfg, tmpl, formAction)
	default:
		return renderOIDCStep(w, tmpl, data)
	}
}

func renderOTPIdentifierStep(
	w http.ResponseWriter,
	r *http.Request,
	sess *goidc.AuthnSession,
	db *pgxpool.Pool,
	cfg *config.Config,
	smsService *notification.SMSService,
	tmpl *template.Template,
	formAction string,
	otp *otpService,
) (goidc.Status, error) {
	customerName := tenant.ResolveBranding(r.Host, cfg.CustomerName, "").Name
	identifier, _ := sess.StoredParameter("identifier").(string)
	data := oidcLoginData{
		Step:           "otp_identifier",
		StepLabel:      "Pasul 2",
		CustomerName:   customerName,
		FormAction:     formAction,
		FrontendOrigin: cfg.FrontendOrigin,
		Identifier:     identifier,
		WalletEnabled:  cfg.EnableWallet,
		Theme:          resolveOIDCThemeSettings(r, sess),
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}
	if r.FormValue("action") == "back" {
		sess.StoreParameter("step", "")
		return renderOIDCStep(w, tmpl, oidcLoginData{
			Step:           "methods",
			StepLabel:      "Pasul 1",
			CustomerName:   customerName,
			FormAction:     formAction,
			FrontendOrigin: cfg.FrontendOrigin,
			WalletEnabled:  cfg.EnableWallet,
			Theme:          resolveOIDCThemeSettings(r, sess),
		})
	}

	identifier = strings.TrimSpace(r.FormValue("identifier"))
	if identifier == "" {
		data.Error = "Introduceți utilizatorul, emailul sau numărul de telefon."
		return renderOIDCStep(w, tmpl, data)
	}
	user, err := findLoginUser(r.Context(), db, identifier)
	if err != nil {
		data.Error = "Contul nu a putut fi localizat pentru autentificare."
		data.Identifier = identifier
		return renderOIDCStep(w, tmpl, data)
	}
	code, err := otp.Generate(r.Context(), user.ID, otpPurposeLogin)
	if err != nil {
		data.Error = "Nu am putut genera codul OTP."
		data.Identifier = identifier
		return renderOIDCStep(w, tmpl, data)
	}
	message := fmt.Sprintf("Codul dumneavoastră de autentificare este: %s. Valabil 10 minute.", code)
	if smsService != nil && smsService.Configured() {
		if _, err := smsService.Send(r.Context(), user.PhoneNumber, message); err != nil {
			data.Error = "Nu am putut trimite codul OTP prin SMS."
			data.Identifier = identifier
			return renderOIDCStep(w, tmpl, data)
		}
	} else if !cfg.IsProduction() {
		message = fmt.Sprintf("Mediu de dezvoltare: codul OTP este %s. Valabil 10 minute.", code)
	}
	if smsService == nil || !smsService.Configured() {
		if cfg.IsProduction() {
			data.Error = "Serviciul SMS nu este configurat pe acest mediu."
			data.Identifier = identifier
			return renderOIDCStep(w, tmpl, data)
		}
	}

	sess.StoreParameter("step", "otp")
	sess.StoreParameter("otp_user_id", user.ID.String())
	sess.StoreParameter("identifier", identifier)
	return renderOIDCStep(w, tmpl, oidcLoginData{
		Step:         "otp",
		StepLabel:    "Pasul 3",
		CustomerName: customerName,
		FormAction:   formAction,
		Identifier:   identifier,
		Message:      message,
		Theme:        resolveOIDCThemeSettings(r, sess),
	})
}

func renderOTPStep(
	w http.ResponseWriter,
	r *http.Request,
	sess *goidc.AuthnSession,
	cfg *config.Config,
	tmpl *template.Template,
	formAction string,
	otp *otpService,
) (goidc.Status, error) {
	customerName := tenant.ResolveBranding(r.Host, cfg.CustomerName, "").Name
	identifier, _ := sess.StoredParameter("identifier").(string)
	data := oidcLoginData{
		Step:         "otp",
		StepLabel:    "Pasul 3",
		CustomerName: customerName,
		FormAction:   formAction,
		Identifier:   identifier,
		Message:      "Am trimis un cod de verificare către telefonul asociat contului.",
		Theme:        resolveOIDCThemeSettings(r, sess),
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}
	if r.FormValue("action") == "back" {
		sess.StoreParameter("step", "otp_identifier")
		return renderOIDCStep(w, tmpl, oidcLoginData{
			Step:           "otp_identifier",
			StepLabel:      "Pasul 2",
			CustomerName:   customerName,
			FormAction:     formAction,
			FrontendOrigin: cfg.FrontendOrigin,
			Identifier:     identifier,
			WalletEnabled:  cfg.EnableWallet,
			Theme:          resolveOIDCThemeSettings(r, sess),
		})
	}

	userIDText, _ := sess.StoredParameter("otp_user_id").(string)
	userID, err := uuid.Parse(userIDText)
	if err != nil {
		return goidc.StatusFailure, fmt.Errorf("invalid otp user id: %w", err)
	}
	code := strings.TrimSpace(r.FormValue("code"))
	if code == "" {
		data.Error = "Introduceți codul primit prin SMS."
		return renderOIDCStep(w, tmpl, data)
	}
	if err := otp.Verify(r.Context(), userID, otpPurposeLogin, code); err != nil {
		data.Error = "Cod invalid sau expirat."
		return renderOIDCStep(w, tmpl, data)
	}

	sess.SetUserID(userID.String())
	sess.StoreParameter("step", "consent")
	return renderConsentStep(w, r, sess, cfg, tmpl, formAction)
}

func renderConsentStep(
	w http.ResponseWriter,
	r *http.Request,
	sess *goidc.AuthnSession,
	cfg *config.Config,
	tmpl *template.Template,
	formAction string,
) (goidc.Status, error) {
	customerName := tenant.ResolveBranding(r.Host, cfg.CustomerName, "").Name
	data := oidcLoginData{
		Step:         "consent",
		StepLabel:    "Pasul 4",
		CustomerName: customerName,
		FormAction:   formAction,
		ClientName:   customerName,
		Scopes:       buildScopeItems(sess.Scopes),
		Theme:        resolveOIDCThemeSettings(r, sess),
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}
	if r.FormValue("action") == "deny" {
		return goidc.StatusFailure, nil
	}

	granted := append([]string{"openid"}, r.Form["granted_scope"]...)
	hasOffline := false
	for _, scope := range strings.Fields(sess.Scopes) {
		if scope == "offline_access" {
			hasOffline = true
			break
		}
	}
	if hasOffline {
		granted = append(granted, "offline_access")
	}
	if len(granted) == 0 {
		data.Error = "Selectați cel puțin un permisiu."
		return renderOIDCStep(w, tmpl, data)
	}
	sess.GrantScopes(strings.Join(granted, " "))
	return goidc.StatusSuccess, nil
}

func renderOIDCStep(w http.ResponseWriter, tmpl *template.Template, data oidcLoginData) (goidc.Status, error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return goidc.StatusInProgress, tmpl.Execute(w, data)
}

type oidcLoginUser struct {
	ID          uuid.UUID
	Subject     string
	PhoneNumber string
}

func findLoginUser(ctx context.Context, db *pgxpool.Pool, identifier string) (oidcLoginUser, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return oidcLoginUser{}, errors.New("missing identifier")
	}

	candidates := phoneNumberCandidates(identifier)

	var user oidcLoginUser
	err := db.QueryRow(ctx, `
		select id, sub, phone_number
		from app_users
		where status = 'active'
			and phone_number_verified = true
			and preferred_otp_channel = 'sms'
			and (
				lower(sub) = lower($1)
				or lower(email) = lower($1)
				or regexp_replace(phone_number, '[^0-9]+', '', 'g') = any($2::text[])
			)
		order by updated_at desc
		limit 1
	`, identifier, candidates).Scan(&user.ID, &user.Subject, &user.PhoneNumber)
	if err != nil {
		return oidcLoginUser{}, err
	}
	user.PhoneNumber = notification.NormalizePhone(user.PhoneNumber)
	return user, nil
}

func loadRolesForSubject(ctx context.Context, db *pgxpool.Pool, subject string) ([]string, error) {
	rows, err := db.Query(ctx, `
		select distinct role_code
		from (
			select ur.role_code
			from app_user_roles ur
			join app_users u on u.id = ur.user_id
			where lower(u.sub) = lower($1)
			union
			select pr.role_code
			from app_memberships m
			join app_users u on u.id = m.user_id
			join app_position_roles pr on pr.position_code = m.position_code
			where lower(u.sub) = lower($1)
				and m.active = true
		) roles
		order by role_code
	`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]string, 0, 8)
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

type scopeItem struct {
	ID    string
	Label string
}

func buildScopeItems(scopes string) []scopeItem {
	labels := map[string]string{
		"profile": "Profil",
		"email":   "Adresă de email",
		"phone":   "Număr de telefon",
	}
	items := make([]scopeItem, 0, 4)
	for _, scope := range strings.Fields(scopes) {
		if scope == "openid" || scope == "offline_access" {
			continue
		}
		label, ok := labels[scope]
		if !ok {
			label = scope
		}
		items = append(items, scopeItem{ID: scope, Label: label})
	}
	return items
}

type oidcLoginData struct {
	Step           string
	StepLabel      string
	CustomerName   string
	FormAction     string
	FrontendOrigin string
	ClientName     string
	Identifier     string
	Message        string
	Error          string
	WalletEnabled  bool
	Scopes         []scopeItem
	Theme          oidcThemeSettings
}

type oidcThemeSettings struct {
	Scheme     string
	Dark       bool
	Primary    string
	Surface    string
	Primary500 string
	Primary600 string
	Primary700 string
	Surface0   string
	Surface50  string
	Surface100 string
	Surface200 string
	Surface300 string
	Surface500 string
	Surface700 string
	Surface900 string
	Bg         string
	Card       string
	CardSoft   string
	Border     string
	Text       string
	Muted      string
	Soft       string
	Focus      string
	Shadow     string
}

func resolveOIDCThemeSettings(r *http.Request, sess *goidc.AuthnSession) oidcThemeSettings {
	settings := oidcThemeSettings{
		Scheme:  "system",
		Primary: "rose",
		Surface: "slate",
		Dark:    false,
	}

	read := func(key string) string {
		if r != nil {
			if value := strings.TrimSpace(r.URL.Query().Get(key)); value != "" {
				return value
			}
		}
		if sess != nil {
			if value, ok := sess.StoredParameter(key).(string); ok {
				return strings.TrimSpace(value)
			}
		}
		return ""
	}

	if value := read("ui_theme_scheme"); value == "light" || value == "dark" || value == "system" {
		settings.Scheme = value
	}
	if value := read("ui_theme_primary"); value != "" {
		settings.Primary = value
	}
	if value := read("ui_theme_surface"); value != "" {
		settings.Surface = value
	}
	if value := read("ui_theme_dark"); value != "" {
		settings.Dark = value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
	}
	if settings.Scheme == "dark" {
		settings.Dark = true
	} else if settings.Scheme == "light" {
		settings.Dark = false
	}

	primary := resolveOIDCPrimaryPalette(settings.Primary)
	surface := resolveOIDCSurfacePalette(settings.Surface)
	settings.Primary500 = primary[500]
	settings.Primary600 = primary[600]
	settings.Primary700 = primary[700]
	settings.Surface0 = surface[0]
	settings.Surface50 = surface[50]
	settings.Surface100 = surface[100]
	settings.Surface200 = surface[200]
	settings.Surface300 = surface[300]
	settings.Surface500 = surface[500]
	settings.Surface700 = surface[700]
	settings.Surface900 = surface[900]

	if settings.Dark {
		settings.Bg = settings.Surface900
		settings.Card = settings.Surface800()
		settings.CardSoft = settings.Surface700
		settings.Border = settings.Surface700
		settings.Text = settings.Surface0
		settings.Muted = settings.Surface300
		settings.Shadow = "0 24px 60px rgba(2,6,23,.45)"
	} else {
		settings.Bg = settings.Surface50
		settings.Card = settings.Surface0
		settings.CardSoft = settings.Surface50
		settings.Border = settings.Surface200
		settings.Text = settings.Surface900
		settings.Muted = settings.Surface500
		settings.Shadow = "0 24px 60px rgba(15,23,42,.14)"
	}
	settings.Soft = fmt.Sprintf("color-mix(in srgb, %s 12%%, %s)", settings.Primary500, settings.Surface0)
	settings.Focus = fmt.Sprintf("color-mix(in srgb, %s 22%%, transparent)", settings.Primary500)
	return settings
}

func (s oidcThemeSettings) Surface800() string {
	return resolveOIDCSurfacePalette(s.Surface)[800]
}

func storeOIDCThemeSettings(sess *goidc.AuthnSession, r *http.Request) {
	if sess == nil || r == nil {
		return
	}
	for _, key := range []string{"ui_theme_scheme", "ui_theme_primary", "ui_theme_surface", "ui_theme_dark"} {
		if value := strings.TrimSpace(r.URL.Query().Get(key)); value != "" {
			sess.StoreParameter(key, value)
		}
	}
}

func resolveOIDCPrimaryPalette(name string) map[int]string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "emerald":
		return map[int]string{50: "#ecfdf5", 100: "#d1fae5", 200: "#a7f3d0", 300: "#6ee7b7", 400: "#34d399", 500: "#10b981", 600: "#059669", 700: "#047857", 800: "#065f46", 900: "#064e3b", 950: "#022c22"}
	case "green":
		return map[int]string{50: "#f0fdf4", 100: "#dcfce7", 200: "#bbf7d0", 300: "#86efac", 400: "#4ade80", 500: "#22c55e", 600: "#16a34a", 700: "#15803d", 800: "#166534", 900: "#14532d", 950: "#052e16"}
	case "lime":
		return map[int]string{50: "#f7fee7", 100: "#ecfccb", 200: "#d9f99d", 300: "#bef264", 400: "#a3e635", 500: "#84cc16", 600: "#65a30d", 700: "#4d7c0f", 800: "#3f6212", 900: "#365314", 950: "#1a2e05"}
	case "orange":
		return map[int]string{50: "#fff7ed", 100: "#ffedd5", 200: "#fed7aa", 300: "#fdba74", 400: "#fb923c", 500: "#f97316", 600: "#ea580c", 700: "#c2410c", 800: "#9a3412", 900: "#7c2d12", 950: "#431407"}
	case "amber":
		return map[int]string{50: "#fffbeb", 100: "#fef3c7", 200: "#fde68a", 300: "#fcd34d", 400: "#fbbf24", 500: "#f59e0b", 600: "#d97706", 700: "#b45309", 800: "#92400e", 900: "#78350f", 950: "#451a03"}
	case "yellow":
		return map[int]string{50: "#fefce8", 100: "#fef9c3", 200: "#fef08a", 300: "#fde047", 400: "#facc15", 500: "#eab308", 600: "#ca8a04", 700: "#a16207", 800: "#854d0e", 900: "#713f12", 950: "#422006"}
	case "teal":
		return map[int]string{50: "#f0fdfa", 100: "#ccfbf1", 200: "#99f6e4", 300: "#5eead4", 400: "#2dd4bf", 500: "#14b8a6", 600: "#0d9488", 700: "#0f766e", 800: "#115e59", 900: "#134e4a", 950: "#042f2e"}
	case "cyan":
		return map[int]string{50: "#ecfeff", 100: "#cffafe", 200: "#a5f3fc", 300: "#67e8f9", 400: "#22d3ee", 500: "#06b6d4", 600: "#0891b2", 700: "#0e7490", 800: "#155e75", 900: "#164e63", 950: "#083344"}
	case "sky":
		return map[int]string{50: "#f0f9ff", 100: "#e0f2fe", 200: "#bae6fd", 300: "#7dd3fc", 400: "#38bdf8", 500: "#0ea5e9", 600: "#0284c7", 700: "#0369a1", 800: "#075985", 900: "#0c4a6e", 950: "#082f49"}
	case "blue":
		return map[int]string{50: "#eff6ff", 100: "#dbeafe", 200: "#bfdbfe", 300: "#93c5fd", 400: "#60a5fa", 500: "#3b82f6", 600: "#2563eb", 700: "#1d4ed8", 800: "#1e40af", 900: "#1e3a8a", 950: "#172554"}
	case "indigo":
		return map[int]string{50: "#eef2ff", 100: "#e0e7ff", 200: "#c7d2fe", 300: "#a5b4fc", 400: "#818cf8", 500: "#6366f1", 600: "#4f46e5", 700: "#4338ca", 800: "#3730a3", 900: "#312e81", 950: "#1e1b4b"}
	case "violet":
		return map[int]string{50: "#f5f3ff", 100: "#ede9fe", 200: "#ddd6fe", 300: "#c4b5fd", 400: "#a78bfa", 500: "#8b5cf6", 600: "#7c3aed", 700: "#6d28d9", 800: "#5b21b6", 900: "#4c1d95", 950: "#2e1065"}
	case "purple":
		return map[int]string{50: "#faf5ff", 100: "#f3e8ff", 200: "#e9d5ff", 300: "#d8b4fe", 400: "#c084fc", 500: "#a855f7", 600: "#9333ea", 700: "#7e22ce", 800: "#6b21a8", 900: "#581c87", 950: "#3b0764"}
	case "fuchsia":
		return map[int]string{50: "#fdf4ff", 100: "#fae8ff", 200: "#f5d0fe", 300: "#f0abfc", 400: "#e879f9", 500: "#d946ef", 600: "#c026d3", 700: "#a21caf", 800: "#86198f", 900: "#701a75", 950: "#4a044e"}
	case "pink":
		return map[int]string{50: "#fdf2f8", 100: "#fce7f3", 200: "#fbcfe8", 300: "#f9a8d4", 400: "#f472b6", 500: "#ec4899", 600: "#db2777", 700: "#be185d", 800: "#9d174d", 900: "#831843", 950: "#500724"}
	case "rose":
		fallthrough
	default:
		return map[int]string{50: "#fff1f2", 100: "#ffe4e6", 200: "#fecdd3", 300: "#fda4af", 400: "#fb7185", 500: "#f43f5e", 600: "#e11d48", 700: "#be123c", 800: "#9f1239", 900: "#881337", 950: "#4c0519"}
	}
}

func resolveOIDCSurfacePalette(name string) map[int]string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "gray":
		return map[int]string{0: "#ffffff", 50: "#f9fafb", 100: "#f3f4f6", 200: "#e5e7eb", 300: "#d1d5db", 400: "#9ca3af", 500: "#6b7280", 600: "#4b5563", 700: "#374151", 800: "#1f2937", 900: "#111827", 950: "#030712"}
	case "zinc":
		return map[int]string{0: "#ffffff", 50: "#fafafa", 100: "#f4f4f5", 200: "#e4e4e7", 300: "#d4d4d8", 400: "#a1a1aa", 500: "#71717a", 600: "#52525b", 700: "#3f3f46", 800: "#27272a", 900: "#18181b", 950: "#09090b"}
	case "neutral":
		return map[int]string{0: "#ffffff", 50: "#fafafa", 100: "#f5f5f5", 200: "#e5e5e5", 300: "#d4d4d4", 400: "#a3a3a3", 500: "#737373", 600: "#525252", 700: "#404040", 800: "#262626", 900: "#171717", 950: "#0a0a0a"}
	case "stone":
		return map[int]string{0: "#ffffff", 50: "#fafaf9", 100: "#f5f5f4", 200: "#e7e5e4", 300: "#d6d3d1", 400: "#a8a29e", 500: "#78716c", 600: "#57534e", 700: "#44403c", 800: "#292524", 900: "#1c1917", 950: "#0c0a09"}
	case "slate":
		fallthrough
	default:
		return map[int]string{0: "#ffffff", 50: "#f8fafc", 100: "#f1f5f9", 200: "#e2e8f0", 300: "#cbd5e1", 400: "#94a3b8", 500: "#64748b", 600: "#475569", 700: "#334155", 800: "#1e293b", 900: "#0f172a", 950: "#020617"}
	}
}

const oidcErrorHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Eroare autentificare</title>
  <style>
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--soft:#fff1f5;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--danger:#b91c1c;--danger-bg:#fef2f2;--danger-border:#fecaca;--shadow:0 28px 72px rgba(15,23,42,.16)}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 48%,#ffe4ec 100%);font-family:Inter,system-ui,sans-serif;color:var(--text)}
    .card{width:min(520px,100%);padding:28px;border:1px solid var(--border);border-radius:28px;background:linear-gradient(180deg,var(--card),#fff8fa);box-shadow:var(--shadow)}
    .eyebrow{display:inline-flex;padding:8px 12px;border-radius:999px;background:var(--soft);color:var(--primary);font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
    h1{margin:14px 0 10px;font-size:1.95rem;letter-spacing:-.04em}
    p{margin:0;color:var(--muted);line-height:1.7}
    .detail{margin-top:18px;padding:14px 16px;border-radius:16px;background:var(--danger-bg);border:1px solid var(--danger-border);color:var(--danger);font-size:.95rem;line-height:1.6}
    .actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:22px}
    .btn{display:inline-flex;align-items:center;justify-content:center;padding:13px 16px;border-radius:14px;text-decoration:none;font-weight:800}
    .primary{background:linear-gradient(180deg,var(--primary),#be123c);color:#fff}
    .secondary{border:1px solid var(--border);background:#fff;color:var(--text)}
  </style>
</head>
<body>
  <main class="card">
    <span class="eyebrow">OIDC Provider</span>
    <h1>{{.CustomerName}}</h1>
    <p>{{.Error}}</p>
    <div class="detail">{{.Detail}}</div>
    <div class="actions">
      <a class="btn primary" href="/">Înapoi la autentificare</a>
    </div>
  </main>
</body>
</html>`

const oidcLoginHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Autentificare</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    :root{color-scheme:light;--primary-50:#fff1f2;--primary-500:#f43f5e;--primary-600:#e11d48;--primary-700:#be123c;--surface-0:#fff;--surface-50:#f8fafc;--surface-100:#f1f5f9;--surface-200:#e2e8f0;--surface-300:#cbd5e1;--surface-500:#64748b;--surface-700:#334155;--surface-900:#0f172a;--bg:var(--surface-50);--card:var(--surface-0);--card-soft:var(--surface-50);--border:var(--surface-200);--text:var(--surface-900);--muted:var(--surface-500);--soft:color-mix(in srgb,var(--primary-500) 12%,var(--surface-0));--focus:color-mix(in srgb,var(--primary-500) 22%,transparent);--shadow:0 24px 60px rgba(15,23,42,.14)}
    html,body{height:100%}
    body{font-family:'Inter Variable','Inter',ui-sans-serif,system-ui,-apple-system,sans-serif;font-size:14px;background:var(--bg);color:var(--text);min-height:100%;overflow:hidden}
    .auth-shell{min-height:100vh;width:100%;display:grid;grid-template-columns:minmax(360px,440px) minmax(0,1fr);background:var(--card)}
    .auth-panel{display:flex;align-items:center;justify-content:center;min-height:100vh;padding:clamp(24px,4vw,64px);overflow-y:auto}
    .auth-visual{position:relative;display:flex;align-items:flex-end;min-height:100vh;padding:clamp(40px,6vw,80px);color:#fff;background:linear-gradient(135deg,rgba(15,23,42,.92),rgba(190,18,60,.82) 46%,rgba(8,47,73,.48));overflow:hidden}
    .auth-visual::after{content:"";position:absolute;inset:0;background:radial-gradient(circle at 24% 20%,rgba(255,255,255,.18),transparent 28%),linear-gradient(180deg,transparent 45%,rgba(14,8,18,.36));pointer-events:none}
    .visual-copy{position:relative;z-index:1;max-width:560px}
    .visual-kicker{font-size:13px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;opacity:.8;margin-bottom:14px}
    .visual-copy h2{font-size:clamp(34px,4.5vw,60px);line-height:1.02;font-weight:750;margin-bottom:18px;color:#fff}
    .visual-copy p{max-width:460px;font-size:16px;line-height:1.6;color:rgba(255,255,255,.86)}
    #content{width:100%;max-width:420px;padding:24px 24px 20px;border:1px solid var(--border);border-radius:18px;background:var(--card);box-shadow:var(--shadow)}
    .header{text-align:center;margin-bottom:18px}
    .step{display:inline-flex;align-items:center;justify-content:center;padding:6px 10px;border-radius:999px;background:var(--soft);color:var(--primary-700);font-size:11px;font-weight:700;letter-spacing:.08em;text-transform:uppercase;margin-bottom:10px}
    h1{font-size:20px;font-weight:700;color:var(--text)}
    .subtitle{font-size:13px;color:var(--muted);margin-top:4px;line-height:1.5}
    .error-banner{background:#fef2f2;border-left:4px solid #ef4444;padding:12px;margin-bottom:16px;border-radius:0 12px 12px 0;display:flex;align-items:center;gap:8px}
    .error-banner p{color:#b91c1c;font-size:12px;font-weight:500;line-height:1.5}
    .info-banner{background:var(--soft);border-left:4px solid var(--primary-500);padding:10px 12px;margin-bottom:16px;border-radius:0 10px 10px 0;font-size:12px;color:var(--text);line-height:1.5}
    .method-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-bottom:12px}
    .method-card{display:flex;flex-direction:column;align-items:center;gap:8px;padding:14px 10px;border-radius:12px;border:1px solid var(--border);background:var(--card);text-align:center;cursor:pointer;color:inherit;font:inherit;transition:transform .2s,box-shadow .2s,border-color .2s,background .2s;width:100%}
    .method-card:hover{transform:translateY(-1px);box-shadow:0 14px 30px rgba(15,23,42,.12);border-color:var(--primary-500);background:var(--soft)}
    .method-card:disabled{cursor:not-allowed;opacity:.6;box-shadow:none;transform:none}
    .method-icon{display:flex;align-items:center;justify-content:center;width:36px;height:36px;border-radius:12px;background:var(--soft);color:var(--primary-600)}
    .method-title{font-size:14px;font-weight:600;color:var(--text)}
    .method-subtitle{font-size:12px;color:var(--muted);margin-top:2px}
    .passkey-banner{display:flex;align-items:center;gap:8px;padding:10px;margin-top:10px;border-radius:12px;border:1px solid var(--border);background:var(--card-soft);font-size:12px;color:var(--text)}
    .field{margin-bottom:16px}
    .field label{display:block;font-size:13px;font-weight:500;color:var(--text);margin-bottom:6px}
    .field input{width:100%;padding:12px 14px;border:1px solid var(--border);border-radius:10px;font-size:14px;font-family:inherit;color:var(--text);background:var(--card-soft);outline:none;transition:border-color .15s,box-shadow .15s,background .15s}
    .field input:focus{border-color:var(--primary-500);box-shadow:0 0 0 3px var(--focus);background:var(--card)}
    .hint{font-size:12px;line-height:1.5;color:var(--muted);margin-top:6px}
    .btn{display:flex;align-items:center;justify-content:center;gap:6px;width:100%;padding:11px 16px;background:var(--primary-500);color:#fff;border:none;border-radius:10px;font-size:14px;font-weight:600;font-family:inherit;cursor:pointer;transition:background .15s,box-shadow .15s;text-decoration:none}
    .btn:hover{background:var(--primary-600);box-shadow:0 10px 20px rgba(15,23,42,.16)}
    .btn:disabled{background:var(--surface-300);color:var(--surface-500);cursor:not-allowed;box-shadow:none}
    .btn-secondary{background:transparent;border:1px solid var(--border);color:var(--text);margin-top:8px}
    .btn-secondary:hover{background:var(--card-soft)}
    .btn-deny{background:transparent;border:1px solid var(--border);color:#b3261e;margin-top:8px}
    .btn-deny:hover{background:#fef2f2;border-color:#ef4444}
    .back-link{display:inline-flex;align-items:center;gap:4px;font-size:12px;color:var(--muted);background:transparent;border:none;cursor:pointer;font-family:inherit;padding:0;margin-bottom:16px;text-decoration:none}
    .back-link:hover{color:var(--primary-600)}
    .otp-boxes{display:flex;gap:8px;justify-content:center;margin-bottom:20px}
    .otp-box{width:44px;height:52px;border:1.5px solid var(--border);border-radius:10px;font-size:22px;font-weight:600;text-align:center;color:var(--text);font-family:inherit;outline:none;transition:border-color .15s,box-shadow .15s,background .15s;caret-color:transparent;background:var(--card-soft)}
    .otp-box:focus{border-color:var(--primary-500);box-shadow:0 0 0 3px var(--focus)}
    .otp-box.filled{border-color:var(--primary-500);background:var(--soft)}
    .client-card{display:flex;align-items:center;gap:12px;padding:12px;border:1px solid var(--border);border-radius:12px;background:var(--card-soft);margin-bottom:16px}
    .client-logo{width:40px;height:40px;border-radius:10px;background:var(--primary-500);display:flex;align-items:center;justify-content:center;flex-shrink:0}
    .client-name{font-size:15px;font-weight:600;color:var(--text)}
    .client-sub{font-size:12px;color:var(--muted)}
    .select-all-row{display:flex;align-items:center;gap:8px;margin-bottom:12px;font-size:13px;color:var(--muted)}
    .scope-list{margin-bottom:16px}
    .scope-item{display:flex;align-items:center;gap:10px;padding:10px 12px;border-radius:10px;border:1px solid var(--border);margin-bottom:8px;cursor:pointer;user-select:none;transition:background .15s,border-color .15s}
    .scope-item:hover{background:var(--soft);border-color:var(--primary-500)}
    .scope-item input[type=checkbox]{width:16px;height:16px;accent-color:var(--primary-500);cursor:pointer}
    .scope-label{font-size:14px;font-weight:500;color:var(--text)}
    .gdpr-notice{font-size:11px;color:var(--muted);text-align:center;margin-top:12px;line-height:1.5}
    @media (max-width: 960px){body{overflow-y:auto}.auth-shell{grid-template-columns:1fr}.auth-visual{display:none}.auth-panel{min-height:auto;padding:24px 16px}.method-grid{grid-template-columns:1fr}.otp-box{width:40px;height:48px}}
    @media (max-width: 520px){.auth-panel{padding:0}#content{max-width:none;min-height:100vh;border:0;border-radius:0;box-shadow:none;padding:24px 18px}.otp-boxes{gap:6px}}
  </style>
</head>
<body>
  <div class="auth-shell">
    <main class="auth-panel">
      <div id="content">
        <div class="header">
          <div class="step">{{.StepLabel}}</div>
          <h1>{{.CustomerName}}</h1>
          <p class="subtitle">Autentificare OIDC securizată pentru acces la platformă.</p>
        </div>

        {{if .Error}}
        <div class="error-banner"><p>{{.Error}}</p></div>
        {{end}}

        {{if eq .Step "methods"}}
        <form action="{{.FormAction}}" method="POST">
          <div class="method-grid">
            <button type="submit" name="method" value="otp" class="method-card">
              <div class="method-icon">✉</div>
              <div>
                <div class="method-title">SMS</div>
                <div class="method-subtitle">Cod OTP</div>
              </div>
            </button>
            <button type="button" id="biometricBtn" class="method-card">
              <div class="method-icon">◈</div>
              <div>
                <div class="method-title">Passkey</div>
                <div class="method-subtitle">Cheie de acces</div>
              </div>
            </button>
            <button type="submit" name="method" value="eudi_wallet" class="method-card" {{if not .WalletEnabled}}disabled{{end}}>
              <div class="method-icon">▣</div>
              <div>
                <div class="method-title">EUDI Wallet</div>
                <div class="method-subtitle">{{if .WalletEnabled}}Portofel digital{{else}}Indisponibil{{end}}</div>
              </div>
            </button>
          </div>
        </form>
        <div id="passkey-banner" class="passkey-banner" style="display:none">
          Passkey folosește autentificarea WebAuthn a dispozitivului și continuă direct fluxul OIDC.
        </div>
        <script>
        (function(){
          var btn=document.getElementById('biometricBtn');
          if(!btn||!window.PublicKeyCredential){if(btn){btn.disabled=true;}return;}
          document.getElementById('passkey-banner').style.display='flex';
          btn.addEventListener('click',function(){
            fetch('/api/passkeys/login-options',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'})
              .then(function(r){if(!r.ok)throw new Error('options failed');return r.json();})
              .then(function(payload){
                var opts=payload.options||payload;
                var challenge=opts.challenge;
                opts.challenge=b64u(opts.challenge);
                if(opts.allowCredentials){opts.allowCredentials=opts.allowCredentials.map(function(item){return Object.assign({},item,{id:b64u(item.id)});});}
                return navigator.credentials.get({publicKey:opts}).then(function(cred){return {cred:cred,challenge:challenge};});
              })
              .then(function(result){
                var cred=result.cred;
                var resp={clientDataJSON:u8b64(new Uint8Array(cred.response.clientDataJSON)),authenticatorData:u8b64(new Uint8Array(cred.response.authenticatorData)),signature:u8b64(new Uint8Array(cred.response.signature)),type:cred.type};
                return fetch('/api/passkeys/login-finish',{method:'POST',headers:{'Content-Type':'application/json'},credentials:'include',body:JSON.stringify({challenge:result.challenge,credential_id:cred.id,response:resp})});
              })
              .then(function(r){if(!r.ok)throw new Error('finish failed');return r.json();})
              .then(function(data){
                var f=document.createElement('form');f.method='POST';f.action='{{.FormAction}}';
                var im=document.createElement('input');im.type='hidden';im.name='method';im.value='passkey_done';
                var nonce=document.createElement('input');nonce.type='hidden';nonce.name='nonce';nonce.value=data.nonce||'';
                f.appendChild(im);f.appendChild(nonce);document.body.appendChild(f);f.submit();
              })
              .catch(function(err){if(err&&err.name!=='NotAllowedError'){alert(err.message||'Autentificarea cu passkey a esuat');}});
          });
          function b64u(b){var s=atob(b.replace(/-/g,'+').replace(/_/g,'/'));return new Uint8Array([].map.call(s,function(c){return c.charCodeAt(0);}));}
          function u8b64(a){return btoa(String.fromCharCode.apply(null,a)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/g,'');}
        })();
        </script>

        {{else if eq .Step "otp_identifier"}}
        <form action="{{.FormAction}}" method="POST">
          <button type="submit" name="action" value="back" class="back-link">Inapoi la metode</button>
          <div class="field">
            <label for="identifier">Utilizator, email sau numar de telefon</label>
            <input id="identifier" name="identifier" value="{{.Identifier}}" autocomplete="username" placeholder="utilizator / email / telefon" autofocus>
            <div class="hint">Dupa identificare, sistemul trimite codul de 6 cifre catre telefonul asociat contului.</div>
          </div>
          <button type="submit" class="btn">Trimite codul prin SMS</button>
        </form>

        {{else if eq .Step "otp"}}
        <div class="info-banner">{{.Message}}</div>
        <form action="{{.FormAction}}" method="POST" id="otpForm">
          <div class="otp-boxes">
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1" autocomplete="one-time-code" autofocus>
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1">
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1">
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1">
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1">
            <input class="otp-box" type="text" inputmode="numeric" maxlength="1">
          </div>
          <input type="hidden" id="code" name="code">
          <button type="submit" class="btn" id="verifyBtn" disabled>Verifica codul</button>
        </form>
        <div style="margin-top:16px;text-align:center">
          <form method="POST" action="{{.FormAction}}" style="display:inline">
            <input type="hidden" name="action" value="back">
            <button type="submit" class="back-link">Inapoi la identificare</button>
          </form>
        </div>
        <script>
        (function(){
          var boxes=document.querySelectorAll('.otp-box'),code=document.getElementById('code'),btn=document.getElementById('verifyBtn'),form=document.getElementById('otpForm');
          function sync(){
            var v=Array.prototype.map.call(boxes,function(b){return b.value;}).join('');
            code.value=v;btn.disabled=v.length<6;
            Array.prototype.forEach.call(boxes,function(b){b.classList.toggle('filled',b.value!=='');});
            if(v.length===6){btn.focus();}
          }
          Array.prototype.forEach.call(boxes,function(box,i){
            box.addEventListener('paste',function(e){
              e.preventDefault();
              var paste=(e.clipboardData||window.clipboardData).getData('text').replace(/\D/g,'').slice(0,6);
              if(paste){paste.split('').forEach(function(c,j){if(boxes[j])boxes[j].value=c;});(boxes[Math.min(paste.length,5)]||boxes[5]).focus();sync();}
            });
            box.addEventListener('input',function(e){
              var v=e.target.value.replace(/\D/g,'');
              if(v.length>1){var d=v.slice(0,6).split('');d.forEach(function(c,j){if(boxes[j])boxes[j].value=c;});(boxes[Math.min(d.length,5)]||boxes[5]).focus();sync();return;}
              e.target.value=v;if(v&&i<5){boxes[i+1].focus();}sync();
            });
            box.addEventListener('keydown',function(e){
              if(e.key==='Backspace'&&!box.value&&i>0){boxes[i-1].value='';boxes[i-1].focus();sync();}
              if(e.key==='ArrowLeft'&&i>0){boxes[i-1].focus();}
              if(e.key==='ArrowRight'&&i<5){boxes[i+1].focus();}
              if(e.key==='Enter'&&code.value.length===6){form.submit();}
            });
          });
          form.addEventListener('submit',sync);
        })();
        </script>

        {{else}}
        <div class="client-card">
          <div class="client-logo">✓</div>
          <div>
            <div class="client-name">{{.ClientName}}</div>
            <div class="client-sub">solicita acces la datele tale</div>
          </div>
        </div>
        <form action="{{.FormAction}}" method="POST" id="consentForm">
          {{if .Scopes}}
          <div class="select-all-row">
            <input type="checkbox" id="selectAll" checked onchange="toggleAll(this)">
            <label for="selectAll" style="cursor:pointer">Selecteaza toate</label>
          </div>
          <div class="scope-list">
            {{range .Scopes}}
            <label class="scope-item">
              <input type="checkbox" name="granted_scope" value="{{.ID}}" checked>
              <span class="scope-label">{{.Label}}</span>
            </label>
            {{end}}
          </div>
          {{end}}
          <button type="submit" name="action" value="allow" class="btn">Accepta si continua</button>
          <button type="submit" name="action" value="deny" class="btn btn-deny">Refuza</button>
          <p class="gdpr-notice">Prin acceptare, esti de acord cu procesarea datelor in conformitate cu Regulamentul (UE) 2016/679 (GDPR).</p>
        </form>
        <script>
        function toggleAll(cb){
          document.querySelectorAll('#consentForm input[name=granted_scope]').forEach(function(i){i.checked=cb.checked;});
        }
        </script>
        {{end}}
      </div>
    </main>
    <aside class="auth-visual" aria-hidden="true">
      <div class="visual-copy">
        <div class="visual-kicker">eGuilde Identity</div>
        <h2>Acces securizat la servicii educationale digitale.</h2>
        <p>Autentificare moderna pentru OTP, passkey si EUDI Wallet, cu pasi clari si consistenti pentru fiecare institutie.</p>
      </div>
    </aside>
  </div>
</body>
</html>`
