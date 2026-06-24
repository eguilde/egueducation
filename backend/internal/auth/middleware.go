package auth

import (
	"context"
	"net/http"
	"strings"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/tenant"
)

type sessionContextKey string
type accessClaimsContextKey string

const (
	requestSessionContextKey sessionContextKey      = "egueducation.session"
	requestClaimsContextKey  accessClaimsContextKey = "egueducation.access_claims"
)

func (s *Service) RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.verifier == nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "auth_unavailable"})
			return
		}

		token, scheme, ok := ExtractAccessToken(r)
		if !ok || token == "" {
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
			return
		}

		var proof *DPoPProof
		if scheme == AccessTokenDPoP {
			var err error
			proof, err = VerifyDPoPProof(r, token)
			if err != nil {
				WriteDPoPNonce(w)
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_dpop_proof"})
				return
			}
		}

		claims, err := s.verifier.Verify(r.Context(), token)
		if err != nil {
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_token"})
			return
		}
		if scheme == AccessTokenDPoP && claims.Cnf.JKT != "" && (proof == nil || claims.Cnf.JKT != proof.Thumbprint) {
			WriteDPoPNonce(w)
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "invalid_dpop_binding"})
			return
		}

		session, err := s.loadSessionContext(r.Context(), r.Host, claims.Subject)
		if err != nil {
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
			return
		}

		branding := tenant.ResolveBranding(r.Host, session.InstitutionName, session.InstitutionID)
		tenantCode := tenant.DefaultTenantCode(session.InstitutionID, branding.Subdomain)
		isSuperAdmin := false
		for _, role := range session.User.Roles {
			if strings.EqualFold(role, "super_admin") {
				isSuperAdmin = true
				break
			}
		}
		sessionCtx, release, err := appdb.AcquireRequestConn(r.Context(), s.db.Raw(), appdb.SessionConfig{
			TenantID:        tenantCode,
			InstitutionID:   session.InstitutionID,
			InstitutionName: session.InstitutionName,
			TenantSubdomain: branding.Subdomain,
			ActorSubject:    session.User.Sub,
			IsSuperAdmin:    isSuperAdmin,
		})
		if err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "tenant_session_unavailable"})
			return
		}
		defer release()

		ctx := context.WithValue(sessionCtx, requestClaimsContextKey, claims)
		ctx = context.WithValue(ctx, requestSessionContextKey, session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) RequirePermissions(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := sessionFromContext(r.Context())
			if !ok {
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
				return
			}

			available := make(map[string]struct{}, len(session.Permissions))
			for _, permission := range session.Permissions {
				available[permission] = struct{}{}
			}

			missing := make([]string, 0, len(required))
			for _, permission := range required {
				if _, ok := available[permission]; !ok {
					missing = append(missing, permission)
				}
			}
			if len(missing) > 0 {
				httpx.JSON(w, http.StatusForbidden, map[string]any{
					"code":                 "permission_denied",
					"required_permissions": required,
					"missing_permissions":  missing,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) RequireAnyPermissions(required ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := sessionFromContext(r.Context())
			if !ok {
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
				return
			}

			available := make(map[string]struct{}, len(session.Permissions))
			for _, permission := range session.Permissions {
				available[permission] = struct{}{}
			}

			for _, permission := range required {
				if _, ok := available[permission]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			httpx.JSON(w, http.StatusForbidden, map[string]any{
				"code":                  "permission_denied",
				"required_any_of":       required,
				"available_permissions": session.Permissions,
			})
		})
	}
}

func sessionFromContext(ctx context.Context) (SessionContext, bool) {
	session, ok := ctx.Value(requestSessionContextKey).(SessionContext)
	return session, ok
}

func accessTokenClaimsFromContext(ctx context.Context) *AccessTokenClaims {
	claims, _ := ctx.Value(requestClaimsContextKey).(*AccessTokenClaims)
	return claims
}
