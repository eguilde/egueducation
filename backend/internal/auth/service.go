package auth

import (
	"context"
	"net/http"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/notification"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	cfg        config.Config
	smsService *notification.SMSService
	db         *pgxpool.Pool
}

func NewService(cfg config.Config, smsService *notification.SMSService, db *pgxpool.Pool) *Service {
	return &Service{
		cfg:        cfg,
		smsService: smsService,
		db:         db,
	}
}

func (s *Service) ListMethods(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"methods": []map[string]any{
			{"code": "oidc_redirect", "enabled": true, "primary": true},
			{"code": "sms_otp", "enabled": s.cfg.EnableSMSOTP && s.smsService.Configured(), "primary": false},
			{"code": "passkey", "enabled": s.cfg.EnablePasskeys, "primary": false},
			{"code": "eudi_wallet", "enabled": s.cfg.EnableWallet, "primary": false},
		},
	})
}

func (s *Service) UIConfig(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"auth_flow":             "redirect",
		"default_locale":        "ro",
		"available_locales":     []string{"ro", "en"},
		"theme_family":          "material3-expressive",
		"theme_brand":           "red-rose",
		"oidc_issuer":           s.cfg.OIDCIssuer,
		"oidc_client_id":        s.cfg.OIDCClientID,
		"desktop_client_id":     s.cfg.OIDCDesktopClient,
		"sms_otp_enabled":       s.cfg.EnableSMSOTP && s.smsService.Configured(),
		"passkey_enabled":       s.cfg.EnablePasskeys,
		"eudi_wallet_enabled":   s.cfg.EnableWallet,
		"gdpr_features_enabled": s.cfg.EnableGDPRFeatures,
	})
}

func (s *Service) SessionContext(w http.ResponseWriter, r *http.Request) {
	session, err := s.loadSessionContext(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]string{
			"code":    "session_load_failed",
			"message": err.Error(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, session)
}

func (s *Service) loadSessionContext(ctx context.Context) (SessionContext, error) {
	var session SessionContext
	err := s.db.QueryRow(ctx, `
		select
			u.id::text,
			u.sub,
			u.name,
			u.email,
			u.phone_number,
			u.locale,
			sc.institution_id,
			sc.institution_name,
			sc.auth_methods,
			sc.gdpr_capabilities
		from app_users u
		join app_session_context sc on sc.user_id = u.id
		where u.sub = 'usr-001'
	`).Scan(
		&session.User.ID,
		&session.User.Sub,
		&session.User.Name,
		&session.User.Email,
		&session.User.PhoneNumber,
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
		where u.sub = 'usr-001'
		order by ur.role_code
	`)
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
		select up.permission_code
		from app_user_permissions up
		join app_users u on u.id = up.user_id
		where u.sub = 'usr-001'
		order by up.permission_code
	`)
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
		where u.sub = 'usr-001'
		order by m.code
	`)
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
