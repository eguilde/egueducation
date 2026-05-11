package auth

import (
	"context"
	"net/http"

	"github.com/eguilde/egueducation/internal/httpx"
)

type sessionContextKey string

const requestSessionContextKey sessionContextKey = "egueducation.session"

func (s *Service) RequireAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := s.currentSubject(r)
		if subject == "" {
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
			return
		}

		session, err := s.loadSessionContext(r.Context(), subject)
		if err != nil {
			httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "unauthenticated"})
			return
		}

		ctx := context.WithValue(r.Context(), requestSessionContextKey, session)
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

			missing := []string{}
			available := make(map[string]struct{}, len(session.Permissions))
			for _, permission := range session.Permissions {
				available[permission] = struct{}{}
			}

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
