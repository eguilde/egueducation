package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/notification"
	"github.com/eguilde/egueducation/internal/tenant"
)

type Service struct {
	cfg         config.Config
	smsService  *notification.SMSService
	db          *appdb.SessionPool
	oidcHandler http.Handler
	verifier    *JWTVerifier
}

func NewService(cfg config.Config, smsService *notification.SMSService, db *appdb.SessionPool) (*Service, error) {
	service := &Service{
		cfg:        cfg,
		smsService: smsService,
		db:         db,
	}
	oidcHandler, verifier, err := newOIDCProviderHandler(db.Raw(), &cfg, smsService, service.redeemPasskeyLoginNonce)
	if err != nil {
		return nil, fmt.Errorf("initialize oidc provider: %w", err)
	}
	service.oidcHandler = oidcHandler
	service.verifier = verifier
	return service, nil
}

func (s *Service) logAudit(ctx context.Context, actorSubject string, action string, targetType string, targetID string, status string, summary string, details map[string]any) {
	_ = audit.Log(ctx, s.db, audit.Event{
		ActorSubject: actorSubject,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Status:       status,
		Summary:      summary,
		Details:      details,
	})
}

func (s *Service) ListMethods(w http.ResponseWriter, _ *http.Request) {
	methods, err := s.listConfiguredMethods(context.Background())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "auth_methods_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"methods": methods,
	})
}

func (s *Service) UIConfig(w http.ResponseWriter, _ *http.Request) {
	methods, err := s.listConfiguredMethods(context.Background())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "auth_ui_config_failed"})
		return
	}

	isEnabled := func(code string) bool {
		for _, method := range methods {
			if method.Code == code {
				return method.Enabled
			}
		}
		return false
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"auth_flow":             "redirect",
		"default_locale":        "ro",
		"available_locales":     []string{"ro", "en"},
		"theme_family":          "material3-expressive",
		"theme_brand":           "red-rose",
		"oidc_issuer":           s.cfg.OIDCIssuer,
		"oidc_client_id":        s.cfg.OIDCClientID,
		"desktop_client_id":     s.cfg.OIDCDesktopClient,
		"sms_otp_enabled":       isEnabled("sms_otp"),
		"passkey_enabled":       isEnabled("passkey"),
		"eudi_wallet_enabled":   isEnabled("eudi_wallet"),
		"gdpr_features_enabled": s.cfg.EnableGDPRFeatures,
	})
}

func (s *Service) RoleCatalog(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		select
			r.code,
			r.label,
			coalesce(array_remove(array_agg(distinct rp.permission_code order by rp.permission_code), null), '{}') as permissions,
			coalesce(array_remove(array_agg(distinct pr.position_code order by pr.position_code), null), '{}') as positions
		from app_roles r
		left join app_role_permissions rp on rp.role_code = r.code
		left join app_position_roles pr on pr.role_code = r.code
		group by r.code, r.label
		order by r.code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_catalog_failed"})
		return
	}
	defer rows.Close()

	roles := make([]RoleCatalogItem, 0, 16)
	for rows.Next() {
		var role RoleCatalogItem
		if err := rows.Scan(&role.Code, &role.Label, &role.Permissions, &role.Positions); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_catalog_failed"})
			return
		}
		role.Description = role.Label
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_catalog_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, RoleCatalogResponse{Roles: roles})
}

func (s *Service) RolePositions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `
		select
			p.code,
			p.name,
			r.code,
			r.label
		from app_position_roles pr
		join app_positions p on p.code = pr.position_code
		join app_roles r on r.code = pr.role_code
		order by p.code, r.code
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_positions_failed"})
		return
	}
	defer rows.Close()

	items := make([]RolePositionItem, 0, 16)
	for rows.Next() {
		var item RolePositionItem
		if err := rows.Scan(&item.PositionCode, &item.PositionName, &item.RoleCode, &item.RoleLabel); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_positions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "role_positions_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, RolePositionResponse{Items: items})
}

func (s *Service) SessionContext(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	if subject == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	session, err := s.loadSessionContext(r.Context(), r.Host, subject)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]string{
			"code":    "session_load_failed",
			"message": err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, session)
}

func (s *Service) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_profile_request"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Locale = strings.TrimSpace(req.Locale)
	if req.Name == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "profile_name_required"})
		return
	}
	if req.Locale != "ro" && req.Locale != "en" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "profile_locale_invalid"})
		return
	}

	_, err := s.db.Exec(r.Context(), `
		update app_users
		set name = $2,
			phone_number = $3,
			locale = $4,
			updated_at = now()
		where id = $1::uuid
	`, session.User.ID, req.Name, req.PhoneNumber, req.Locale)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "profile_update_failed"})
		return
	}

	s.logAudit(r.Context(), session.User.Sub, "profile.update", "user", session.User.ID, "success", "User updated own profile.", map[string]any{
		"locale": req.Locale,
	})

	updated, err := s.loadSessionContext(r.Context(), r.Host, session.User.Sub)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "session_load_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}

func (s *Service) ListPasskeys(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	rows, err := s.db.Query(r.Context(), `
		select id::text, credential_id, device_name, created_at::text, coalesce(last_used_at::text, '')
		from app_passkeys
		where user_id = $1::uuid
		order by created_at desc
	`, session.User.ID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkeys_list_failed"})
		return
	}
	defer rows.Close()

	passkeys := []PasskeyCredentialSummary{}
	for rows.Next() {
		var passkey PasskeyCredentialSummary
		if err := rows.Scan(&passkey.ID, &passkey.CredentialID, &passkey.DeviceName, &passkey.CreatedAt, &passkey.LastUsedAt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkeys_list_failed"})
			return
		}
		passkeys = append(passkeys, passkey)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkeys_list_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, passkeys)
}

func (s *Service) BeginPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_challenge_failed"})
		return
	}
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes)
	expiresAt := time.Now().Add(passkeyChallengeTTL)

	_, _ = s.db.Exec(r.Context(), `
		delete from app_passkey_challenges
		where user_id = $1::uuid and kind = 'registration'
	`, session.User.ID)

	_, err := s.db.Exec(r.Context(), `
		insert into app_passkey_challenges (user_id, challenge, kind, expires_at)
		values ($1::uuid, $2, 'registration', $3)
	`, session.User.ID, challenge, expiresAt)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_challenge_failed"})
		return
	}

	var opts PasskeyRegistrationOptions
	opts.Challenge = challenge
	opts.RP.Name = "EguEducation"
	opts.RP.ID = strings.Split(strings.TrimPrefix(strings.TrimPrefix(s.cfg.FrontendOrigin, "https://"), "http://"), ":")[0]
	if opts.RP.ID == "" {
		opts.RP.ID = r.Host
	}
	opts.User.ID = base64.RawURLEncoding.EncodeToString([]byte(session.User.ID))
	opts.User.Name = session.User.Email
	if opts.User.Name == "" {
		opts.User.Name = session.User.Name
	}
	if opts.User.Name == "" {
		opts.User.Name = session.User.Sub
	}
	opts.User.DisplayName = session.User.Name
	opts.PubKeyCredParams = []map[string]any{
		{"type": "public-key", "alg": -7},
		{"type": "public-key", "alg": -257},
	}
	opts.Timeout = 60000
	opts.Attestation = "none"

	httpx.JSON(w, http.StatusOK, opts)
}

func (s *Service) FinishPasskeyRegistration(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	var req FinishPasskeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_passkey_request"})
		return
	}
	req.CredentialID = strings.TrimSpace(req.CredentialID)
	req.DeviceName = strings.TrimSpace(req.DeviceName)
	req.Challenge = strings.TrimSpace(req.Challenge)
	if req.CredentialID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_credential_required"})
		return
	}
	if req.Challenge == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_challenge_required"})
		return
	}
	if req.DeviceName == "" {
		req.DeviceName = "Passkey"
	}

	var challenge string
	err := s.db.QueryRow(r.Context(), `
		select challenge
		from app_passkey_challenges
		where user_id = $1::uuid and kind = 'registration' and expires_at > now()
		order by created_at desc
		limit 1
	`, session.User.ID).Scan(&challenge)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_challenge_expired"})
		return
	}
	if challenge != req.Challenge {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_challenge_mismatch"})
		return
	}

	clientDataEncoded, _ := req.Response["clientDataJSON"].(string)
	if clientDataEncoded == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_client_data_required"})
		return
	}
	clientData, err := parsePasskeyClientData(clientDataEncoded)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_client_data_invalid"})
		return
	}
	if clientData.Type != "webauthn.create" || clientData.Challenge != req.Challenge {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_client_data_invalid"})
		return
	}
	if normalizeOrigin(clientData.Origin) != normalizeOrigin(s.cfg.FrontendOrigin) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_origin_invalid"})
		return
	}
	if responseType, _ := req.Response["type"].(string); responseType != "public-key" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_response_invalid"})
		return
	}
	attestationObject, _ := req.Response["attestationObject"].(string)
	if attestationObject == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_attestation_required"})
		return
	}
	publicKey, signCount, err := extractPasskeyRegistrationMaterial(attestationObject, passkeyRPID(s.cfg.FrontendOrigin))
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_attestation_invalid"})
		return
	}
	payload, err := json.Marshal(map[string]any{
		"challenge":  req.Challenge,
		"clientData": clientData,
		"response":   req.Response,
		"passkey": map[string]any{
			"public_key": publicKey,
			"sign_count": signCount,
		},
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_save_failed"})
		return
	}

	var saved PasskeyCredentialSummary
	err = s.db.QueryRow(r.Context(), `
		insert into app_passkeys (user_id, credential_id, device_name, credential_payload)
		values ($1::uuid, $2, $3, $4::jsonb)
		on conflict (credential_id) do update
		set device_name = excluded.device_name,
			credential_payload = excluded.credential_payload
		returning id::text, credential_id, device_name, created_at::text, coalesce(last_used_at::text, '')
	`, session.User.ID, req.CredentialID, req.DeviceName, string(payload)).Scan(
		&saved.ID, &saved.CredentialID, &saved.DeviceName, &saved.CreatedAt, &saved.LastUsedAt,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_save_failed"})
		return
	}
	_, _ = s.db.Exec(r.Context(), `delete from app_passkey_challenges where user_id = $1::uuid and kind = 'registration'`, session.User.ID)

	s.logAudit(r.Context(), session.User.Sub, "profile.passkey.register", "user", session.User.ID, "success", "User registered a passkey.", map[string]any{
		"credential_id": req.CredentialID,
		"device_name":   req.DeviceName,
	})

	httpx.JSON(w, http.StatusOK, saved)
}

func (s *Service) BeginPasskeyAuthentication(w http.ResponseWriter, r *http.Request) {
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_challenge_failed"})
		return
	}

	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes)
	expiresAt := time.Now().Add(passkeyChallengeTTL)

	_, _ = s.db.Exec(r.Context(), `delete from app_passkey_login_challenges`)

	_, err := s.db.Exec(r.Context(), `
		insert into app_passkey_login_challenges (challenge, expires_at)
		values ($1, $2)
	`, challenge, expiresAt)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_challenge_failed"})
		return
	}

	var opts PasskeyAuthenticationOptions
	opts.Challenge = challenge
	opts.RP.Name = "EguEducation"
	opts.RP.ID = passkeyRPID(s.cfg.FrontendOrigin)
	opts.Timeout = 60000
	opts.UserVerification = "required"

	httpx.JSON(w, http.StatusOK, BeginPasskeyAuthenticationResponse{
		Status:  "ready",
		Options: opts,
	})
}

func (s *Service) FinishPasskeyAuthentication(w http.ResponseWriter, r *http.Request) {
	var req FinishPasskeyAuthenticationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_passkey_request"})
		return
	}
	req.CredentialID = strings.TrimSpace(req.CredentialID)
	req.Challenge = strings.TrimSpace(req.Challenge)
	if req.CredentialID == "" || req.Challenge == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_passkey_request"})
		return
	}

	var storedChallenge string
	err := s.db.QueryRow(r.Context(), `
		select challenge
		from app_passkey_login_challenges
		where expires_at > now()
		order by created_at desc
		limit 1
	`).Scan(&storedChallenge)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_challenge_expired"})
		return
	}
	if storedChallenge != req.Challenge {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_challenge_mismatch"})
		return
	}

	clientDataEncoded := passkeyResponseString(req.Response, "clientDataJSON")
	authenticatorDataEncoded := passkeyResponseString(req.Response, "authenticatorData")
	if clientDataEncoded == "" || authenticatorDataEncoded == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_response_invalid"})
		return
	}
	clientData, err := parsePasskeyAssertionData(clientDataEncoded)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_client_data_invalid"})
		return
	}
	if clientData.Type != "webauthn.get" || clientData.Challenge != req.Challenge {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_client_data_invalid"})
		return
	}
	if normalizeOrigin(clientData.Origin) != normalizeOrigin(s.cfg.FrontendOrigin) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_origin_invalid"})
		return
	}
	if responseType, _ := req.Response["type"].(string); responseType != "public-key" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_response_invalid"})
		return
	}

	subject, userID, deviceName, publicKey, storedSignCount, err := s.lookupPasskeySubjectByCredentialID(r.Context(), req.CredentialID)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_invalid"})
		return
	}
	newSignCount, err := passkeyAssertionVerified(authenticatorDataEncoded, clientDataEncoded, passkeyResponseString(req.Response, "signature"), publicKey, passkeyRPID(s.cfg.FrontendOrigin), storedSignCount)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "passkey_authenticator_invalid"})
		return
	}

	_, _ = s.db.Exec(r.Context(), `
		update app_passkeys
		set last_used_at = now(),
			credential_payload = jsonb_set(
				coalesce(credential_payload, '{}'::jsonb),
				'{passkey,sign_count}',
				to_jsonb($2::bigint),
				true
			)
		where credential_id = $1
	`, req.CredentialID, int64(newSignCount))
	_, _ = s.db.Exec(r.Context(), `
		update app_users
		set last_login_at = now(),
			updated_at = now()
		where lower(sub) = lower($1)
	`, subject)

	_, _ = s.db.Exec(r.Context(), `delete from app_passkey_login_challenges where challenge = $1`, req.Challenge)

	nonce, err := s.storePasskeyLoginNonce(r.Context(), userID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "passkey_login_failed"})
		return
	}

	s.logAudit(r.Context(), subject, "auth.passkey.verify", "session", userID, "success", "Passkey verified and nonce issued for OIDC login.", map[string]any{
		"credential_id": req.CredentialID,
		"device_name":   deviceName,
		"user_id":       userID,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{"nonce": nonce})
}

func (s *Service) ActivateEUDIWallet(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	_, err := s.db.Exec(r.Context(), `
		insert into app_eudi_wallets (user_id, status, activated_at, updated_at)
		values ($1::uuid, 'active', now(), now())
		on conflict (user_id) do update
		set status = 'active',
			activated_at = coalesce(app_eudi_wallets.activated_at, now()),
			updated_at = now()
	`, session.User.ID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "eudi_activation_failed"})
		return
	}

	s.logAudit(r.Context(), session.User.Sub, "profile.eudi.activate", "user", session.User.ID, "success", "User activated EUDI wallet status.", nil)
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "active"})
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	s.logAudit(r.Context(), subject, "auth.session.logout", "session", subject, "success", "OIDC-backed API session signed out.", nil)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": "signed_out",
	})
}

func (s *Service) loadSessionContext(ctx context.Context, host string, subject string) (SessionContext, error) {
	var session SessionContext
	err := s.db.QueryRow(ctx, `
		select
			u.id::text,
			u.sub,
			u.name,
			u.email,
			u.email_verified,
			u.phone_number,
			u.phone_number_verified,
			u.preferred_otp_channel,
			u.locale,
			sc.institution_id,
			sc.institution_name,
			sc.auth_methods,
			sc.gdpr_capabilities
		from app_users u
		join app_session_context sc on sc.user_id = u.id
		where lower(u.sub) = lower($1)
		   or u.id::text = $1
	`, subject).Scan(
		&session.User.ID,
		&session.User.Sub,
		&session.User.Name,
		&session.User.Email,
		&session.User.EmailVerified,
		&session.User.PhoneNumber,
		&session.User.PhoneNumberVerified,
		&session.User.PreferredOTPChannel,
		&session.User.Locale,
		&session.InstitutionID,
		&session.InstitutionName,
		&session.Authentication,
		&session.GDPRCapabilities,
	)
	if err != nil {
		return SessionContext{}, err
	}

	branding := tenant.ResolveBranding(host, session.InstitutionName, session.InstitutionID)
	session.InstitutionID = branding.InstitutionID
	session.InstitutionName = branding.Name

	roleRows, err := s.db.Query(ctx, `
		select distinct role_code
		from (
			select ur.role_code
			from app_user_roles ur
			join app_users u on u.id = ur.user_id
			where (u.id::text = $1 or lower(u.sub) = lower($1))
			union
			select pr.role_code
			from app_memberships m
			join app_users u on u.id = m.user_id
			join app_position_roles pr on pr.position_code = m.position_code
			left join app_org_units ou on ou.code = m.org_unit_code
			where (u.id::text = $1 or lower(u.sub) = lower($1))
				and m.active = true
				and coalesce(ou.tenant_code, '') = $2
		) roles
		order by role_code
	`, session.User.ID, session.InstitutionID)
	if err != nil {
		return SessionContext{}, err
	}
	defer roleRows.Close()

	for roleRows.Next() {
		var role string
		if err := roleRows.Scan(&role); err != nil {
			return SessionContext{}, err
		}
		session.User.Roles = append(session.User.Roles, role)
	}
	if err := roleRows.Err(); err != nil {
		return SessionContext{}, err
	}

	permissionRows, err := s.db.Query(ctx, `
		select distinct permission_code
		from (
			select up.permission_code
			from app_user_permissions up
			join app_users u on u.id = up.user_id
			where (u.id::text = $1 or lower(u.sub) = lower($1))
			union
			select rp.permission_code
			from app_user_roles ur
			join app_users u on u.id = ur.user_id
			join app_role_permissions rp on rp.role_code = ur.role_code
			where (u.id::text = $1 or lower(u.sub) = lower($1))
			union
			select pp.permission_code
			from app_memberships m
			join app_users u on u.id = m.user_id
			join app_position_permissions pp on pp.position_code = m.position_code
			where (u.id::text = $1 or lower(u.sub) = lower($1))
				and m.active = true
			union
			select rp.permission_code
			from app_memberships m
			join app_users u on u.id = m.user_id
			join app_position_roles pr on pr.position_code = m.position_code
			join app_role_permissions rp on rp.role_code = pr.role_code
			where (u.id::text = $1 or lower(u.sub) = lower($1))
				and m.active = true
		) permissions
		order by permission_code
	`, session.User.ID)
	if err != nil {
		return SessionContext{}, err
	}
	defer permissionRows.Close()

	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			return SessionContext{}, err
		}
		session.Permissions = append(session.Permissions, permission)
	}
	if err := permissionRows.Err(); err != nil {
		return SessionContext{}, err
	}

	moduleRows, err := s.db.Query(ctx, `
		select m.code, m.active
		from app_user_modules um
		join app_modules m on m.code = um.module_code
		join app_users u on u.id = um.user_id
		where lower(u.sub) = lower($1)
		order by m.code
	`, subject)
	if err != nil {
		return SessionContext{}, err
	}
	defer moduleRows.Close()

	for moduleRows.Next() {
		var module SessionModule
		if err := moduleRows.Scan(&module.Code, &module.Active); err != nil {
			return SessionContext{}, err
		}
		session.Modules = append(session.Modules, module)
	}
	if err := moduleRows.Err(); err != nil {
		return SessionContext{}, err
	}

	return session, nil
}

type configuredMethod struct {
	Code    string `json:"code"`
	Enabled bool   `json:"enabled"`
	Primary bool   `json:"primary"`
}

func (s *Service) listConfiguredMethods(ctx context.Context) ([]configuredMethod, error) {
	rows, err := s.db.Query(ctx, `
		select code, enabled, primary_method
		from app_auth_methods
		order by sort_order, code
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	methods := []configuredMethod{}
	for rows.Next() {
		var method configuredMethod
		if err := rows.Scan(&method.Code, &method.Enabled, &method.Primary); err != nil {
			return nil, err
		}
		method.Enabled = method.Enabled && s.runtimeCapability(method.Code)
		methods = append(methods, method)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return methods, nil
}

func (s *Service) runtimeCapability(code string) bool {
	switch code {
	case "oidc_redirect":
		return true
	case "sms_otp":
		return s.cfg.EnableSMSOTP && s.smsService.Configured()
	case "passkey":
		return s.cfg.EnablePasskeys
	case "eudi_wallet":
		return s.cfg.EnableWallet
	default:
		return false
	}
}

func (s *Service) lookupPasskeySubjectByCredentialID(ctx context.Context, credentialID string) (string, string, string, *passkeyPublicKey, uint32, error) {
	var (
		subject    string
		userID     string
		deviceName string
		payload    string
	)
	err := s.db.QueryRow(ctx, `
		select u.sub, u.id::text, p.device_name, p.credential_payload::text
		from app_passkeys p
		join app_users u on u.id = p.user_id
		where p.credential_id = $1
			and u.status = 'active'
	`, credentialID).Scan(&subject, &userID, &deviceName, &payload)
	if err != nil {
		return "", "", "", nil, 0, err
	}
	publicKey, signCount, err := passkeyPublicKeyFromStoredPayload([]byte(payload), passkeyRPID(s.cfg.FrontendOrigin))
	if err != nil {
		return "", "", "", nil, 0, err
	}
	return subject, userID, deviceName, publicKey, signCount, nil
}

func (s *Service) currentSubject(r *http.Request) string {
	if session, ok := sessionFromContext(r.Context()); ok && session.User.Sub != "" {
		return session.User.Sub
	}
	if claims := accessTokenClaimsFromContext(r.Context()); claims != nil && claims.Subject != "" {
		return claims.Subject
	}
	return ""
}

func CurrentSubjectFromRequest(r *http.Request) string {
	if session, ok := sessionFromContext(r.Context()); ok && session.User.Sub != "" {
		return session.User.Sub
	}
	if claims := accessTokenClaimsFromContext(r.Context()); claims != nil && claims.Subject != "" {
		return claims.Subject
	}
	return ""
}

func CurrentInstitutionIDFromRequest(r *http.Request) string {
	if session, ok := sessionFromContext(r.Context()); ok && session.InstitutionID != "" {
		return session.InstitutionID
	}
	if claims := accessTokenClaimsFromContext(r.Context()); claims != nil {
		if claims.InstitutionID != "" {
			return claims.InstitutionID
		}
		if claims.TenantID != "" {
			return claims.TenantID
		}
	}
	return ""
}

func phoneNumberCandidates(phoneNumber string) []string {
	normalized := notification.NormalizePhone(phoneNumber)
	digitsOnly := phoneDigits(normalized)
	candidates := []string{}
	add := func(value string) {
		if value == "" {
			return
		}
		for _, existing := range candidates {
			if existing == value {
				return
			}
		}
		candidates = append(candidates, value)
	}

	add(digitsOnly)
	if strings.HasPrefix(digitsOnly, "40") && len(digitsOnly) > 2 {
		add("0" + digitsOnly[2:])
	}

	return candidates
}

func phoneDigits(value string) string {
	var digits strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	return digits.String()
}
