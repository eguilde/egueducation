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
		redirect := origin + "/callback"
		if _, ok := seen[redirect]; ok {
			continue
		}
		seen[redirect] = struct{}{}
		redirects = append(redirects, redirect)
	}

	clients := []struct {
		ID          string
		Name        string
		RedirectURIs []string
		AppType     string
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
	return func(w http.ResponseWriter, _ *http.Request, err error) error {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return tmpl.Execute(w, map[string]string{
			"CustomerName": cfg.CustomerName,
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
	data := oidcLoginData{
		Step:           "methods",
		CustomerName:   cfg.CustomerName,
		FormAction:     formAction,
		FrontendOrigin: cfg.FrontendOrigin,
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}

	switch r.FormValue("method") {
	case "otp":
		identifier := strings.TrimSpace(r.FormValue("identifier"))
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
		if smsService == nil || !smsService.Configured() {
			data.Error = "Fluxul OTP nu este configurat pe acest mediu."
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
		if _, err := smsService.Send(r.Context(), user.PhoneNumber, message); err != nil {
			data.Error = "Nu am putut trimite codul OTP prin SMS."
			data.Identifier = identifier
			return renderOIDCStep(w, tmpl, data)
		}
		sess.StoreParameter("step", "otp")
		sess.StoreParameter("otp_user_id", user.ID.String())
		sess.StoreParameter("identifier", identifier)
		return renderOIDCStep(w, tmpl, oidcLoginData{
			Step:         "otp",
			CustomerName: cfg.CustomerName,
			FormAction:   formAction,
			Identifier:   identifier,
			Message:      "Am trimis un cod de verificare către telefonul asociat contului.",
		})
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

func renderOTPStep(
	w http.ResponseWriter,
	r *http.Request,
	sess *goidc.AuthnSession,
	cfg *config.Config,
	tmpl *template.Template,
	formAction string,
	otp *otpService,
) (goidc.Status, error) {
	identifier, _ := sess.StoredParameter("identifier").(string)
	data := oidcLoginData{
		Step:         "otp",
		CustomerName: cfg.CustomerName,
		FormAction:   formAction,
		Identifier:   identifier,
		Message:      "Am trimis un cod de verificare către telefonul asociat contului.",
	}
	if r.Method == http.MethodGet {
		return renderOIDCStep(w, tmpl, data)
	}
	if r.FormValue("action") == "back" {
		sess.StoreParameter("step", "")
		return renderOIDCStep(w, tmpl, oidcLoginData{
			Step:           "methods",
			CustomerName:   cfg.CustomerName,
			FormAction:     formAction,
			FrontendOrigin: cfg.FrontendOrigin,
			Identifier:     identifier,
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
	data := oidcLoginData{
		Step:         "consent",
		CustomerName: cfg.CustomerName,
		FormAction:   formAction,
		ClientName:   cfg.CustomerName,
		Scopes:       buildScopeItems(sess.Scopes),
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
	CustomerName   string
	FormAction     string
	FrontendOrigin string
	ClientName     string
	Identifier     string
	Message        string
	Error          string
	Scopes         []scopeItem
}

const oidcErrorHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Eroare autentificare</title>
  <style>
    :root{color-scheme:light;--bg:#fff7f8;--card:#ffffff;--accent:#e11d48;--accent-strong:#be123c;--text:#0f172a;--muted:#64748b;--border:#fecdd3;--shadow:0 28px 70px rgba(15,23,42,.14)}
    body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:radial-gradient(circle at top left,rgba(225,29,72,.12),transparent 30rem),var(--bg);font-family:Inter,system-ui,sans-serif;color:var(--text)}
    .card{width:min(420px,100%);border:1px solid var(--border);border-radius:24px;background:var(--card);box-shadow:var(--shadow);padding:28px}
    h1{margin:0 0 12px;font-size:22px}.msg{color:var(--muted);line-height:1.6}.err{margin-top:16px;padding:12px 14px;border-radius:14px;background:#fff1f2;color:var(--accent-strong)}
    .btn{margin-top:18px;display:inline-flex;align-items:center;justify-content:center;padding:12px 16px;border-radius:14px;background:var(--accent);color:#fff;text-decoration:none;font-weight:700}
  </style>
</head>
<body>
  <main class="card">
    <h1>{{.CustomerName}}</h1>
    <p class="msg">{{.Error}}</p>
    <div class="err">{{.Detail}}</div>
    <a class="btn" href="/">Înapoi la autentificare</a>
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
    :root{color-scheme:light;--bg:#fff7f8;--card:#ffffff;--soft:#fff1f2;--accent:#e11d48;--accent-strong:#be123c;--text:#0f172a;--muted:#64748b;--border:#fecdd3;--shadow:0 28px 70px rgba(15,23,42,.14)}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;background:radial-gradient(circle at top left,rgba(225,29,72,.14),transparent 32rem),linear-gradient(135deg,#fff7f8 0%,#ffffff 48%,#ffe4e9 100%);font-family:Inter,system-ui,sans-serif;color:var(--text)}
    .shell{min-height:100vh;display:grid;grid-template-columns:minmax(0,1.08fr) minmax(320px,.92fr)}@media(max-width:900px){.shell{grid-template-columns:1fr}}
    .hero{padding:48px 40px;display:flex;align-items:center}.hero h1{font-size:clamp(2.5rem,5vw,4.5rem);line-height:.95;margin:0 0 16px;letter-spacing:-.05em}.hero p{max-width:42rem;color:var(--muted);font-size:1rem;line-height:1.8}
    .panel-wrap{display:flex;align-items:center;justify-content:center;padding:24px}.panel{width:min(480px,100%);border:1px solid var(--border);border-radius:30px;background:rgba(255,255,255,.92);box-shadow:var(--shadow);padding:28px}
    .kicker{display:inline-flex;align-items:center;border-radius:999px;padding:9px 14px;background:var(--soft);color:var(--accent-strong);font-size:.76rem;font-weight:800;letter-spacing:.14em;text-transform:uppercase}
    h2{margin:18px 0 8px;font-size:1.7rem;letter-spacing:-.03em}.sub{margin:0 0 18px;color:var(--muted);line-height:1.7}
    form{display:grid;gap:16px}.field{display:grid;gap:8px}.field label{font-size:.88rem;font-weight:700;color:var(--muted)}input{width:100%;padding:14px 16px;border:1px solid var(--border);border-radius:16px;background:#fff;color:var(--text);font:inherit}
    .btn{display:inline-flex;align-items:center;justify-content:center;width:100%;padding:14px 16px;border:0;border-radius:16px;background:var(--accent);color:#fff;font:inherit;font-weight:800;cursor:pointer}.btn.secondary{background:#fff;color:var(--text);border:1px solid var(--border)}
    .grid{display:grid;gap:12px}.methods{grid-template-columns:1fr 1fr}.msg{padding:12px 14px;border-radius:14px;background:#f1f5f9;color:var(--text);font-size:.92rem;line-height:1.6}.err{padding:12px 14px;border-radius:14px;background:#fff1f2;color:var(--accent-strong);font-size:.92rem;line-height:1.6}
    .checks{display:grid;gap:10px}.check{display:flex;align-items:center;gap:10px;padding:12px 14px;border:1px solid var(--border);border-radius:16px}.check input{width:18px;height:18px}
    .minor{font-size:.85rem;color:var(--muted);text-align:center}.otp{display:grid;grid-template-columns:repeat(6,1fr);gap:10px}.otp input{text-align:center;font-size:1.2rem;font-weight:800}
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div>
        <span class="kicker">OIDC Provider</span>
        <h1>{{.CustomerName}}</h1>
        <p>Autentificare OIDC/OAuth2 cu cod de autorizare, PKCE, DPoP și pagini dedicate pentru login, consimțământ și logout.</p>
      </div>
    </section>
    <section class="panel-wrap">
      <main class="panel">
        {{if .Error}}<div class="err">{{.Error}}</div>{{end}}
        {{if eq .Step "methods"}}
          <span class="kicker">Pasul 1</span>
          <h2>Alege autentificarea</h2>
          <p class="sub">Continuă cu OTP prin telefonul asociat contului sau cu passkey.</p>
          <form method="POST" action="{{.FormAction}}">
            <div class="field">
              <label for="identifier">Utilizator, email sau telefon</label>
              <input id="identifier" name="identifier" value="{{.Identifier}}" autocomplete="username" placeholder="thomas.admin / email / telefon" />
            </div>
            <div class="grid methods">
              <button class="btn" type="submit" name="method" value="otp">Continuă cu OTP</button>
              <button class="btn secondary" id="biometricBtn" type="button">Passkey</button>
            </div>
          </form>
          <p class="minor">Passkey folosește fluxul WebAuthn existent al aplicației și predă identitatea înapoi providerului OIDC.</p>
          <script>
          (function(){
            var btn=document.getElementById('biometricBtn');
            if(!btn||!window.PublicKeyCredential){if(btn){btn.disabled=true;}return;}
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
                .then(function(r){if(!r.ok){throw new Error('finish failed');}return r.json();})
                .then(function(data){
                  var f=document.createElement('form');f.method='POST';f.action='{{.FormAction}}';
                  var method=document.createElement('input');method.type='hidden';method.name='method';method.value='passkey_done';
                  var nonce=document.createElement('input');nonce.type='hidden';nonce.name='nonce';nonce.value=data.nonce||'';
                  f.appendChild(method);f.appendChild(nonce);document.body.appendChild(f);f.submit();
                })
                .catch(function(){window.location.reload();});
            });
            function b64u(value){var base64=value.replace(/-/g,'+').replace(/_/g,'/');base64=base64.padEnd(Math.ceil(base64.length/4)*4,'=');var binary=atob(base64);var out=new Uint8Array(binary.length);for(var i=0;i<binary.length;i++){out[i]=binary.charCodeAt(i);}return out;}
            function u8b64(bytes){var binary='';bytes.forEach(function(byte){binary+=String.fromCharCode(byte);});return btoa(binary).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/g,'');}
          })();
          </script>
        {{else if eq .Step "otp"}}
          <span class="kicker">Pasul 2</span>
          <h2>Verificare OTP</h2>
          <p class="sub">{{.Message}}</p>
          <form method="POST" action="{{.FormAction}}">
            <div class="otp">
              <input maxlength="1" inputmode="numeric" class="otp-box"><input maxlength="1" inputmode="numeric" class="otp-box"><input maxlength="1" inputmode="numeric" class="otp-box"><input maxlength="1" inputmode="numeric" class="otp-box"><input maxlength="1" inputmode="numeric" class="otp-box"><input maxlength="1" inputmode="numeric" class="otp-box">
            </div>
            <input type="hidden" id="code" name="code">
            <button class="btn" type="submit">Verifică și continuă</button>
            <button class="btn secondary" type="submit" name="action" value="back">Înapoi</button>
          </form>
          <script>
          (function(){
            var boxes=document.querySelectorAll('.otp-box');var hidden=document.getElementById('code');
            function sync(){var value='';boxes.forEach(function(box){value+=box.value.replace(/\D/g,'');});hidden.value=value;}
            boxes.forEach(function(box,index){
              box.addEventListener('input',function(){box.value=box.value.replace(/\D/g,'').slice(0,1);if(box.value&&index<boxes.length-1){boxes[index+1].focus();}sync();});
              box.addEventListener('keydown',function(event){if(event.key==='Backspace'&&!box.value&&index>0){boxes[index-1].focus();}sync();});
              box.addEventListener('paste',function(event){event.preventDefault();var text=(event.clipboardData||window.clipboardData).getData('text').replace(/\D/g,'').slice(0,6);text.split('').forEach(function(char,i){if(boxes[i]){boxes[i].value=char;}});sync();});
            });
          })();
          </script>
        {{else}}
          <span class="kicker">Pasul 3</span>
          <h2>Consimțământ OIDC</h2>
          <p class="sub">{{.ClientName}} solicită acces la datele selectate.</p>
          <form method="POST" action="{{.FormAction}}">
            <div class="checks">
              {{range .Scopes}}
                <label class="check"><input type="checkbox" name="granted_scope" value="{{.ID}}" checked><span>{{.Label}}</span></label>
              {{end}}
            </div>
            <button class="btn" type="submit" name="action" value="allow">Acceptă și continuă</button>
            <button class="btn secondary" type="submit" name="action" value="deny">Refuză</button>
          </form>
        {{end}}
      </main>
    </section>
  </div>
</body>
</html>`
