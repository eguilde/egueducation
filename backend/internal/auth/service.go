package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/notification"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	cfg        config.Config
	smsService *notification.SMSService
	db         *pgxpool.Pool
	oidc       *oidcKeyPair
}

func NewService(cfg config.Config, smsService *notification.SMSService, db *pgxpool.Pool) *Service {
	keyPair, _ := newOIDCKeyPair()
	return &Service{
		cfg:        cfg,
		smsService: smsService,
		db:         db,
		oidc:       keyPair,
	}
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

const (
	smsOTPExpiry      = 10 * time.Minute
	smsOTPMaxAttempts = 5
	smsOTPRateLimit   = 3
	sessionCookieName = "egueducation_subject"
)

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

func (s *Service) SessionContext(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	if subject == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
		return
	}

	session, err := s.loadSessionContext(r.Context(), subject)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]string{
			"code":    "session_load_failed",
			"message": err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, session)
}

func (s *Service) RequestSMSOTP(w http.ResponseWriter, r *http.Request) {
	var req RequestSMSOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_sms_otp_request"})
		return
	}

	identifier := normalizeIdentifier(req.Identifier)
	if identifier == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_sms_otp_request"})
		return
	}
	if !s.cfg.EnableSMSOTP || !s.smsService.Configured() {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "sms_otp_not_available"})
		return
	}

	phone, err := s.lookupVerifiedPhone(r.Context(), identifier)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "sms_otp_unavailable"})
		return
	}

	var recentCount int
	_ = s.db.QueryRow(r.Context(), `
		select count(*)
		from sms_otp_codes
		where identifier = $1 and created_at > now() - interval '10 minutes'
	`, phone).Scan(&recentCount)
	if recentCount >= smsOTPRateLimit {
		s.logAudit(r.Context(), identifier, "auth.sms_otp.request", "authentication", identifier, "failed", "SMS OTP request rate limited.", map[string]any{
			"channel":    "sms",
			"identifier": identifier,
		})
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"code": "sms_otp_rate_limited"})
		return
	}

	code, err := generateOTP()
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "sms_otp_generation_failed"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "sms_otp_generation_failed"})
		return
	}

	_, err = s.db.Exec(r.Context(), `
		insert into sms_otp_codes (identifier, code_hash, purpose, expires_at, created_at, updated_at)
		values ($1, $2, 'sms_login', $3, now(), now())
	`, phone, string(hash), time.Now().Add(smsOTPExpiry))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "sms_otp_store_failed"})
		return
	}

	message := fmt.Sprintf("Codul dumneavoastra de autentificare EguEducation este: %s. Valabil 10 minute.", code)
	if _, err := s.smsService.Send(r.Context(), phone, message); err != nil {
		s.logAudit(r.Context(), identifier, "auth.sms_otp.request", "authentication", identifier, "failed", "SMS OTP send failed.", map[string]any{
			"channel":      "sms",
			"identifier":   identifier,
			"masked_phone": maskPhone(phone),
		})
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "sms_otp_send_failed"})
		return
	}

	s.logAudit(r.Context(), identifier, "auth.sms_otp.request", "authentication", identifier, "success", "SMS OTP requested.", map[string]any{
		"channel":      "sms",
		"identifier":   identifier,
		"masked_phone": maskPhone(phone),
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":       "sent",
		"channel":      "sms",
		"masked_phone": maskPhone(phone),
	})
}

func (s *Service) VerifySMSOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifySMSOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_sms_otp_verify"})
		return
	}

	identifier := normalizeIdentifier(req.Identifier)
	code := strings.TrimSpace(req.Code)
	if identifier == "" || code == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_sms_otp_verify"})
		return
	}

	phone, err := s.lookupVerifiedPhone(r.Context(), identifier)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "sms_otp_invalid"})
		return
	}

	var (
		otpID    int64
		codeHash string
		attempts int
	)
	err = s.db.QueryRow(r.Context(), `
		select id, code_hash, attempts
		from sms_otp_codes
		where identifier = $1 and used = false and expires_at > now()
		order by created_at desc
		limit 1
	`, phone).Scan(&otpID, &codeHash, &attempts)
	if err != nil {
		s.logAudit(r.Context(), identifier, "auth.sms_otp.verify", "authentication", identifier, "failed", "SMS OTP verification failed.", map[string]any{
			"channel":    "sms",
			"identifier": identifier,
			"reason":     "missing_or_expired_code",
		})
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "sms_otp_invalid"})
		return
	}
	if attempts >= smsOTPMaxAttempts {
		s.logAudit(r.Context(), identifier, "auth.sms_otp.verify", "authentication", identifier, "failed", "SMS OTP verification blocked.", map[string]any{
			"channel":    "sms",
			"identifier": identifier,
			"reason":     "too_many_attempts",
		})
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"code": "sms_otp_too_many_attempts"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(codeHash), []byte(code)); err != nil {
		_, _ = s.db.Exec(r.Context(), `update sms_otp_codes set attempts = attempts + 1, updated_at = now() where id = $1`, otpID)
		s.logAudit(r.Context(), identifier, "auth.sms_otp.verify", "authentication", identifier, "failed", "SMS OTP verification failed.", map[string]any{
			"channel":    "sms",
			"identifier": identifier,
			"reason":     "invalid_code",
		})
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "sms_otp_invalid"})
		return
	}

	_, _ = s.db.Exec(r.Context(), `update sms_otp_codes set used = true, updated_at = now() where id = $1`, otpID)

	subject, err := s.lookupSubjectByIdentifier(r.Context(), identifier)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "session_load_failed"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    subject,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Environment == "production",
		MaxAge:   60 * 60 * 8,
	})

	session, err := s.loadSessionContext(r.Context(), subject)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "session_load_failed"})
		return
	}

	s.logAudit(r.Context(), subject, "auth.sms_otp.verify", "session", session.User.ID, "success", "SMS OTP verified and session established.", map[string]any{
		"channel":          "sms",
		"identifier":       identifier,
		"user_id":          session.User.ID,
		"institution_id":   session.InstitutionID,
		"preferred_locale": session.User.Locale,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":  "verified",
		"channel": "sms",
		"session": session,
	})
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Environment == "production",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	s.logAudit(r.Context(), subject, "auth.session.logout", "session", subject, "success", "User session signed out.", nil)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": "signed_out",
	})
}

func (s *Service) loadSessionContext(ctx context.Context, subject string) (SessionContext, error) {
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

	roleRows, err := s.db.Query(ctx, `
		select ur.role_code
		from app_user_roles ur
		join app_users u on u.id = ur.user_id
		where lower(u.sub) = lower($1)
		order by ur.role_code
	`, subject)
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
		select permission_code
		from (
			select up.permission_code
			from app_user_permissions up
			join app_users u on u.id = up.user_id
			where lower(u.sub) = lower($1)
			union
			select pp.permission_code
			from app_memberships m
			join app_users u on u.id = m.user_id
			join app_position_permissions pp on pp.position_code = m.position_code
			where lower(u.sub) = lower($1)
				and m.active = true
		) permissions
		order by permission_code
	`, subject)
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

func (s *Service) lookupVerifiedPhone(ctx context.Context, identifier string) (string, error) {
	email, phoneCandidates := identifierCandidates(identifier)

	var phone string
	err := s.db.QueryRow(ctx, `
		select phone_number
		from app_users
		where (
			($1 <> '' and lower(email) = lower($1))
			or regexp_replace(phone_number, '[^0-9]+', '', 'g') = any($2::text[])
		)
			and status = 'active'
			and phone_number_verified = true
			and preferred_otp_channel = 'sms'
	`, email, phoneCandidates).Scan(&phone)
	if err != nil {
		return "", err
	}
	return notification.NormalizePhone(phone), nil
}

func (s *Service) lookupSubjectByIdentifier(ctx context.Context, identifier string) (string, error) {
	email, phoneCandidates := identifierCandidates(identifier)

	var subject string
	err := s.db.QueryRow(ctx, `
		select sub
		from app_users
		where (
			($1 <> '' and lower(email) = lower($1))
			or regexp_replace(phone_number, '[^0-9]+', '', 'g') = any($2::text[])
		)
			and status = 'active'
	`, email, phoneCandidates).Scan(&subject)
	if err != nil {
		return "", err
	}
	return subject, nil
}

func (s *Service) currentSubject(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		subject := strings.TrimSpace(cookie.Value)
		if subject != "" {
			return subject
		}
	}
	return ""
}

func CurrentSubjectFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		subject := strings.TrimSpace(cookie.Value)
		if subject != "" {
			return subject
		}
	}
	return ""
}

func normalizeIdentifier(input string) string {
	value := strings.TrimSpace(strings.ToLower(input))
	if strings.Contains(value, "@") {
		return value
	}
	return notification.NormalizePhone(value)
}

func identifierCandidates(identifier string) (string, []string) {
	if strings.Contains(identifier, "@") {
		return identifier, []string{}
	}

	normalized := notification.NormalizePhone(identifier)
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

	return "", candidates
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

func generateOTP() (string, error) {
	const digits = "0123456789"
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for index, value := range bytes {
		bytes[index] = digits[int(value)%len(digits)]
	}
	return string(bytes), nil
}

func maskPhone(phone string) string {
	value := notification.NormalizePhone(phone)
	if len(value) <= 4 {
		return value
	}
	middle := len(value) - 5
	if middle < 0 {
		middle = 0
	}
	return value[:3] + strings.Repeat("*", middle) + value[len(value)-2:]
}
