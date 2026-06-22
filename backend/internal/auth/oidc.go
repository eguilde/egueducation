package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/jackc/pgx/v5"
)

const (
	authorizationCodeTTL = 10 * time.Minute
	refreshTokenTTL      = 8 * time.Hour
	accessTokenTTL       = 15 * time.Minute
	idTokenTTL           = 15 * time.Minute
)

func (s *Service) Discovery(w http.ResponseWriter, _ *http.Request) {
	issuer := s.cfg.OIDCIssuer
	httpx.JSON(w, http.StatusOK, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/authorize",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"jwks_uri":                              issuer + "/jwks",
		"revocation_endpoint":                   issuer + "/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "offline_access"},
		"claims_supported": []string{
			"sub", "name", "email", "email_verified", "phone_number", "phone_number_verified", "locale", "roles",
		},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	})
}

func (s *Service) JWKS(w http.ResponseWriter, _ *http.Request) {
	if s.oidc == nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "oidc_key_unavailable"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": s.oidc.keyID,
				"n":   s.oidc.modulus,
				"e":   s.oidc.exponent,
			},
		},
	})
}

func (s *Service) Authorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	clientID := strings.TrimSpace(query.Get("client_id"))
	redirectURI := strings.TrimSpace(query.Get("redirect_uri"))
	responseType := strings.TrimSpace(query.Get("response_type"))
	scope := strings.TrimSpace(query.Get("scope"))
	state := strings.TrimSpace(query.Get("state"))
	codeChallenge := strings.TrimSpace(query.Get("code_challenge"))
	codeChallengeMethod := strings.TrimSpace(query.Get("code_challenge_method"))
	nonce := strings.TrimSpace(query.Get("nonce"))
	scope = strings.Join(uniqueScopes(strings.Fields(scope)), " ")

	if clientID == "" || redirectURI == "" || responseType != "code" || state == "" || scope == "" || codeChallenge == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_authorization_request"})
		return
	}
	if codeChallengeMethod != "" && codeChallengeMethod != "S256" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "unsupported_code_challenge_method"})
		return
	}
	valid, err := s.validateClientRedirect(r.Context(), clientID, redirectURI)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "oidc_client_validation_failed"})
		return
	}
	if !valid {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_redirect_uri"})
		return
	}

	subject := s.currentSubject(r)
	if subject == "" {
		loginURL, err := url.Parse(s.cfg.FrontendOrigin + "/auth/login")
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "login_redirect_failed"})
			return
		}
		values := loginURL.Query()
		values.Set("returnUrl", s.cfg.OIDCIssuer+"/authorize?"+r.URL.RawQuery)
		loginURL.RawQuery = values.Encode()
		http.Redirect(w, r, loginURL.String(), http.StatusFound)
		return
	}

	if granted, err := s.hasConsent(r.Context(), clientID, subject, scope); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_validation_failed"})
		return
	} else if !granted {
		requestID, createErr := s.createConsentRequest(r.Context(), clientID, subject, redirectURI, scope, state, nonce, codeChallenge)
		if createErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_request_store_failed"})
			return
		}
		http.Redirect(w, r, s.cfg.FrontendOrigin+"/auth/consent?request="+url.QueryEscape(requestID), http.StatusFound)
		return
	}

	redirectTarget, err := s.buildAuthorizationRedirect(r.Context(), clientID, subject, redirectURI, scope, state, nonce, codeChallenge)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "authorization_code_store_failed"})
		return
	}
	http.Redirect(w, r, redirectTarget, http.StatusFound)
}

func (s *Service) ConsentRequest(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	if subject == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("request"))
	if requestID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_consent_request"})
		return
	}

	var (
		clientID   string
		clientName string
		scope      string
		expiresAt  time.Time
	)
	err := s.db.QueryRow(r.Context(), `
		select cr.client_id, c.client_name, cr.scope, cr.expires_at
		from oidc_consent_requests cr
		join oidc_clients c on c.client_id = cr.client_id
		where cr.id::text = $1
			and lower(cr.subject) = lower($2)
			and cr.status = 'pending'
			and cr.expires_at > now()
	`, requestID, subject).Scan(&clientID, &clientName, &scope, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "consent_request_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_request_load_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, ConsentRequestResponse{
		RequestID:  requestID,
		ClientID:   clientID,
		ClientName: clientName,
		Scopes:     consentScopes(scope),
		ExpiresAt:  expiresAt.UTC().Format(time.RFC3339),
	})
}

func (s *Service) ConsentDecision(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	if subject == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	var req ConsentDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_consent_decision"})
		return
	}

	requestID := strings.TrimSpace(req.RequestID)
	decision := strings.TrimSpace(req.Decision)
	if requestID == "" || (decision != "allow" && decision != "deny") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_consent_decision"})
		return
	}

	var (
		clientID            string
		redirectURI         string
		scope               string
		state               string
		nonce               *string
		codeChallenge       string
		codeChallengeMethod string
	)
	err := s.db.QueryRow(r.Context(), `
		select client_id, redirect_uri, scope, state, nonce, code_challenge, code_challenge_method
		from oidc_consent_requests
		where id::text = $1
			and lower(subject) = lower($2)
			and status = 'pending'
			and expires_at > now()
	`, requestID, subject).Scan(&clientID, &redirectURI, &scope, &state, &nonce, &codeChallenge, &codeChallengeMethod)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "consent_request_not_found"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_request_load_failed"})
		return
	}

	if decision == "deny" {
		if _, updateErr := s.db.Exec(r.Context(), `
			update oidc_consent_requests
			set status = 'denied', updated_at = now()
			where id::text = $1
		`, requestID); updateErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_request_update_failed"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status":      "denied",
			"redirect_to": s.buildErrorRedirect(redirectURI, state, "access_denied"),
		})
		return
	}

	grantedScope := mergeGrantedScope(scope, req.GrantedScopes)
	if grantedScope == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_granted_scope"})
		return
	}
	if !isSubsetOfRequested(grantedScope, scope) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_granted_scope"})
		return
	}
	if codeChallengeMethod != "S256" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "unsupported_code_challenge_method"})
		return
	}

	if _, err := s.db.Exec(r.Context(), `
		insert into oidc_consents (client_id, subject, scope)
		values ($1, $2, $3)
		on conflict (client_id, subject, scope)
		do update set updated_at = now(), granted_at = now()
	`, clientID, subject, grantedScope); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_store_failed"})
		return
	}

	if _, err := s.db.Exec(r.Context(), `
		update oidc_consent_requests
		set status = 'approved', updated_at = now()
		where id::text = $1
	`, requestID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "consent_request_update_failed"})
		return
	}

	redirectTarget, err := s.buildAuthorizationRedirect(r.Context(), clientID, subject, redirectURI, grantedScope, state, derefString(nonce), codeChallenge)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "authorization_code_store_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":      "approved",
		"redirect_to": redirectTarget,
	})
}

func (s *Service) buildAuthorizationRedirect(ctx context.Context, clientID, subject, redirectURI, scope, state, nonce, codeChallenge string) (string, error) {
	code, err := randomToken(32)
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(ctx, `
		insert into oidc_authorization_codes (
			code,
			client_id,
			subject,
			redirect_uri,
			scope,
			nonce,
			code_challenge,
			code_challenge_method,
			expires_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, code, clientID, subject, redirectURI, scope, nullableString(nonce), codeChallenge, "S256", time.Now().Add(authorizationCodeTTL))
	if err != nil {
		return "", err
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	values := redirectURL.Query()
	values.Set("code", code)
	values.Set("state", state)
	redirectURL.RawQuery = values.Encode()
	return redirectURL.String(), nil
}

func (s *Service) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_token_request"})
		return
	}

	clientID := strings.TrimSpace(r.FormValue("client_id"))
	grantType := strings.TrimSpace(r.FormValue("grant_type"))
	if clientID == "" || grantType == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_token_request"})
		return
	}
	if ok, err := s.clientExists(r.Context(), clientID); err != nil || !ok {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_client"})
		return
	}

	var response map[string]any
	var err error
	switch grantType {
	case "authorization_code":
		response, err = s.exchangeAuthorizationCode(r.Context(), clientID, r.FormValue("code"), r.FormValue("redirect_uri"), r.FormValue("code_verifier"))
	case "refresh_token":
		response, err = s.exchangeRefreshToken(r.Context(), clientID, r.FormValue("refresh_token"))
	default:
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "unsupported_grant_type"})
		return
	}
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusInternalServerError
		}
		httpx.JSON(w, status, map[string]any{"code": err.Error()})
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_revoke_request"})
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	if token == "" {
		httpx.JSON(w, http.StatusOK, map[string]any{})
		return
	}

	_, _ = s.db.Exec(r.Context(), `
		update oidc_refresh_tokens
		set revoked = true, updated_at = now()
		where token = $1
	`, token)
	httpx.JSON(w, http.StatusOK, map[string]any{})
}

func (s *Service) UserInfo(w http.ResponseWriter, r *http.Request) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "missing_authorization"})
		return
	}
	parts := strings.Fields(authorization)
	if len(parts) != 2 {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_authorization"})
		return
	}

	claims, err := s.verifyToken(parts[1], "access_token")
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_token"})
		return
	}

	subject, _ := claims["sub"].(string)
	session, err := s.loadSessionContext(r.Context(), subject)
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_token"})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"sub":                   session.User.Sub,
		"name":                  session.User.Name,
		"email":                 session.User.Email,
		"email_verified":        session.User.EmailVerified,
		"phone_number":          session.User.PhoneNumber,
		"phone_number_verified": session.User.PhoneNumberVerified,
		"locale":                session.User.Locale,
		"roles":                 session.User.Roles,
	})
}

func (s *Service) exchangeAuthorizationCode(ctx context.Context, clientID, code, redirectURI, verifier string) (map[string]any, error) {
	if code == "" || redirectURI == "" || verifier == "" {
		return nil, errors.New("invalid_grant")
	}

	var (
		subject             string
		scope               string
		nonce               *string
		codeChallenge       string
		codeChallengeMethod string
	)
	err := s.db.QueryRow(ctx, `
		select subject, scope, nonce, code_challenge, code_challenge_method
		from oidc_authorization_codes
		where code = $1
			and client_id = $2
			and redirect_uri = $3
			and used = false
			and expires_at > now()
	`, code, clientID, redirectURI).Scan(&subject, &scope, &nonce, &codeChallenge, &codeChallengeMethod)
	if err != nil {
		return nil, errors.New("invalid_grant")
	}
	if codeChallengeMethod != "S256" || codeChallenge != codeChallengeS256(verifier) {
		return nil, errors.New("invalid_grant")
	}

	if _, err := s.db.Exec(ctx, `update oidc_authorization_codes set used = true, updated_at = now() where code = $1`, code); err != nil {
		return nil, errors.New("authorization_code_update_failed")
	}

	return s.issueTokenSet(ctx, clientID, subject, scope, derefString(nonce))
}

func (s *Service) exchangeRefreshToken(ctx context.Context, clientID, refreshToken string) (map[string]any, error) {
	if refreshToken == "" {
		return nil, errors.New("invalid_grant")
	}

	var (
		subject string
		scope   string
	)
	err := s.db.QueryRow(ctx, `
		select subject, scope
		from oidc_refresh_tokens
		where token = $1
			and client_id = $2
			and revoked = false
			and expires_at > now()
	`, refreshToken, clientID).Scan(&subject, &scope)
	if err != nil {
		return nil, errors.New("invalid_grant")
	}

	return s.issueTokenSet(ctx, clientID, subject, scope, "")
}

func (s *Service) issueTokenSet(ctx context.Context, clientID, subject, scope, nonce string) (map[string]any, error) {
	session, err := s.loadSessionContext(ctx, subject)
	if err != nil {
		return nil, errors.New("session_load_failed")
	}

	now := time.Now().UTC()
	accessToken, err := s.signToken(map[string]any{
		"iss":   s.cfg.OIDCIssuer,
		"sub":   session.User.Sub,
		"aud":   clientID,
		"iat":   now.Unix(),
		"exp":   now.Add(accessTokenTTL).Unix(),
		"scope": scope,
		"jti":   mustRandomToken(16),
		"typ":   "access_token",
	})
	if err != nil {
		return nil, errors.New("token_sign_failed")
	}

	idTokenClaims := map[string]any{
		"iss":                   s.cfg.OIDCIssuer,
		"sub":                   session.User.Sub,
		"aud":                   clientID,
		"iat":                   now.Unix(),
		"exp":                   now.Add(idTokenTTL).Unix(),
		"auth_time":             now.Unix(),
		"name":                  session.User.Name,
		"email":                 session.User.Email,
		"email_verified":        session.User.EmailVerified,
		"phone_number":          session.User.PhoneNumber,
		"phone_number_verified": session.User.PhoneNumberVerified,
		"locale":                session.User.Locale,
		"roles":                 session.User.Roles,
	}
	if nonce != "" {
		idTokenClaims["nonce"] = nonce
	}
	idToken, err := s.signToken(idTokenClaims)
	if err != nil {
		return nil, errors.New("token_sign_failed")
	}

	refreshToken, err := randomToken(48)
	if err != nil {
		return nil, errors.New("refresh_token_generation_failed")
	}
	if _, err := s.db.Exec(ctx, `
		insert into oidc_refresh_tokens (token, client_id, subject, scope, expires_at)
		values ($1, $2, $3, $4, $5)
	`, refreshToken, clientID, subject, scope, now.Add(refreshTokenTTL)); err != nil {
		return nil, errors.New("refresh_token_store_failed")
	}

	return map[string]any{
		"access_token":  accessToken,
		"token_type":    "DPoP",
		"expires_in":    int(accessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"id_token":      idToken,
		"scope":         scope,
	}, nil
}

func (s *Service) validateClientRedirect(ctx context.Context, clientID, redirectURI string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		select exists(
			select 1
			from oidc_client_redirect_uris cru
			join oidc_clients c on c.client_id = cru.client_id
			where c.client_id = $1
				and c.active = true
				and cru.redirect_uri = $2
		)
	`, clientID, redirectURI).Scan(&exists)
	return exists, err
}

func (s *Service) clientExists(ctx context.Context, clientID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		select exists(select 1 from oidc_clients where client_id = $1 and active = true)
	`, clientID).Scan(&exists)
	return exists, err
}

func (s *Service) signToken(claims map[string]any) (string, error) {
	if s.oidc == nil {
		return "", errors.New("oidc_key_unavailable")
	}
	header, err := json.Marshal(map[string]any{
		"alg": "RS256",
		"typ": "JWT",
		"kid": s.oidc.keyID,
	})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerEncoded := base64.RawURLEncoding.EncodeToString(header)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := headerEncoded + "." + payloadEncoded
	sum := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.oidc.privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *Service) verifyToken(token string, expectedTypes ...string) (map[string]any, error) {
	if s.oidc == nil {
		return nil, errors.New("oidc_key_unavailable")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid_token")
	}

	signingInput := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signingInput))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	if err := rsa.VerifyPKCS1v15(&s.oidc.privateKey.PublicKey, crypto.SHA256, sum[:], signature); err != nil {
		return nil, err
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if issuer, _ := claims["iss"].(string); issuer != s.cfg.OIDCIssuer {
		return nil, errors.New("token_issuer_invalid")
	}
	if aud, _ := claims["aud"].(string); aud != s.cfg.OIDCClientID && aud != s.cfg.OIDCDesktopClient {
		return nil, errors.New("token_audience_invalid")
	}
	if exp, ok := claims["exp"].(float64); !ok || int64(exp) <= time.Now().Unix() {
		return nil, errors.New("token_expired")
	}
	if len(expectedTypes) > 0 {
		typ, _ := claims["typ"].(string)
		if typ == "" || !slices.Contains(expectedTypes, typ) {
			return nil, errors.New("token_type_invalid")
		}
	}
	return claims, nil
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func mustRandomToken(size int) string {
	value, err := randomToken(size)
	if err != nil {
		return ""
	}
	return value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Service) hasConsent(ctx context.Context, clientID, subject, scope string) (bool, error) {
	var granted bool
	err := s.db.QueryRow(ctx, `
		select exists(
			select 1
			from oidc_consents
			where client_id = $1
				and lower(subject) = lower($2)
				and scope = $3
		)
	`, clientID, subject, scope).Scan(&granted)
	return granted, err
}

func (s *Service) createConsentRequest(ctx context.Context, clientID, subject, redirectURI, scope, state, nonce, codeChallenge string) (string, error) {
	var requestID string
	err := s.db.QueryRow(ctx, `
		insert into oidc_consent_requests (
			client_id,
			subject,
			redirect_uri,
			scope,
			state,
			nonce,
			code_challenge,
			code_challenge_method,
			expires_at
		) values ($1, $2, $3, $4, $5, $6, $7, 'S256', $8)
		returning id::text
	`, clientID, subject, redirectURI, scope, state, nullableString(nonce), codeChallenge, time.Now().Add(authorizationCodeTTL)).Scan(&requestID)
	return requestID, err
}

func (s *Service) buildErrorRedirect(redirectURI, state, errorCode string) string {
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	values := redirectURL.Query()
	values.Set("error", errorCode)
	values.Set("state", state)
	redirectURL.RawQuery = values.Encode()
	return redirectURL.String()
}

func consentScopes(scope string) []ConsentScope {
	scopeLabels := map[string]string{
		"profile": "Profil și roluri",
		"email":   "Adresă de email verificată",
		"phone":   "Număr de telefon verificat",
	}
	scopes := make([]ConsentScope, 0, len(scopeLabels))
	for _, code := range strings.Fields(scope) {
		label, ok := scopeLabels[code]
		if !ok {
			continue
		}
		scopes = append(scopes, ConsentScope{
			Code:  code,
			Label: label,
		})
	}
	return scopes
}

func mergeGrantedScope(requestedScope string, grantedScopes []string) string {
	allowed := make([]string, 0, len(strings.Fields(requestedScope)))
	optionalScopes := consentScopes(requestedScope)
	optionalIndex := make(map[string]bool, len(optionalScopes))
	for _, scope := range optionalScopes {
		optionalIndex[scope.Code] = true
	}
	normalizedGranted := uniqueScopes(grantedScopes)
	for _, code := range strings.Fields(requestedScope) {
		if !optionalIndex[code] || slices.Contains(normalizedGranted, code) {
			allowed = append(allowed, code)
		}
	}
	return strings.Join(uniqueScopes(allowed), " ")
}

func isSubsetOfRequested(grantedScope, requestedScope string) bool {
	requested := uniqueScopes(strings.Fields(requestedScope))
	for _, code := range strings.Fields(grantedScope) {
		if !slices.Contains(requested, code) {
			return false
		}
	}
	return true
}

func uniqueScopes(scopes []string) []string {
	seen := make(map[string]bool, len(scopes))
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		code := strings.TrimSpace(scope)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		normalized = append(normalized, code)
	}
	slices.Sort(normalized)
	return normalized
}
