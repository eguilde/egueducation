package education

import (
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
)

// EducationDelegationScope describes the boundary of a delegated permission.
// An institution scope authorizes the permission only in the active tenant and
// institution; a resource scope additionally requires the exact resource ID.
type EducationDelegationScope struct {
	PermissionCode string
	ResourceType   string
	ResourceID     string
}

func normalizeEducationDelegationScope(scope EducationDelegationScope) EducationDelegationScope {
	scope.PermissionCode = strings.TrimSpace(scope.PermissionCode)
	scope.ResourceType = strings.ToLower(strings.TrimSpace(scope.ResourceType))
	scope.ResourceID = strings.TrimSpace(scope.ResourceID)
	if scope.ResourceType == "" {
		scope.ResourceType = "institution"
	}
	return scope
}

// educationDelegationScopeMatches is deliberately pure so handlers and future
// route middleware cannot accidentally broaden an institution-bound grant.
func educationDelegationScopeMatches(granted, requested EducationDelegationScope) bool {
	granted = normalizeEducationDelegationScope(granted)
	requested = normalizeEducationDelegationScope(requested)
	if granted.PermissionCode == "" || granted.PermissionCode != requested.PermissionCode {
		return false
	}
	if granted.ResourceType == "institution" {
		return granted.ResourceID == ""
	}
	return granted.ResourceType == requested.ResourceType && granted.ResourceID != "" && granted.ResourceID == requested.ResourceID
}

// authorizeEducationPermission authorizes direct tenant permissions first,
// then consults only active accepted delegations in the current DB session.
// The latter is intentionally request-time state and is never copied to JWTs.
func (s *Service) authorizeEducationPermission(r *http.Request, scope EducationDelegationScope) (bool, error) {
	return s.authorizeEducationPermissionForSubject(r, authruntime.CurrentSubjectFromRequest(r), scope)
}

func (s *Service) authorizeEducationPermissionForSubject(r *http.Request, subject string, scope EducationDelegationScope) (bool, error) {
	scope = normalizeEducationDelegationScope(scope)
	if !strings.HasPrefix(scope.PermissionCode, "education.") || scope.PermissionCode == "" {
		return false, nil
	}
	direct, err := s.currentSubjectHasPermission(r, strings.TrimSpace(subject), scope.PermissionCode)
	if err != nil || direct {
		return direct, err
	}
	actorID, err := s.currentActorUserID(r, strings.TrimSpace(subject))
	if err != nil || actorID == "" {
		return false, err
	}
	var granted []EducationDelegationScope
	rows, err := s.pool.Query(r.Context(), `
		select permission_code, resource_type, coalesce(resource_id::text, '')
		from education_role_delegations
		where tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
			and delegate_user_id = $1::uuid
			and status = 'accepted'
			and valid_from <= current_date
			and (valid_until is null or valid_until >= current_date)
			and permission_code = $2
	`, actorID, scope.PermissionCode)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate EducationDelegationScope
		if err := rows.Scan(&candidate.PermissionCode, &candidate.ResourceType, &candidate.ResourceID); err != nil {
			return false, err
		}
		granted = append(granted, candidate)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	for _, candidate := range granted {
		if educationDelegationScopeMatches(candidate, scope) {
			return true, nil
		}
	}
	return false, nil
}

// RequireEducationPermission is the HTTP authorization boundary for School
// routes. It resolves direct RBAC from the database first and then active
// delegations, so revocation is effective immediately without minting a new
// access token. The concrete resource scope is derived only from chi route
// parameters, never from request JSON.
func (s *Service) RequireEducationPermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope := educationDelegationScopeForRequest(permission, r)
			allowed, err := s.authorizeEducationPermission(r, scope)
			if err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_authorization_failed"})
				return
			}
			if !allowed {
				httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_permission_required", "permission": permission})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) RequireAnyEducationPermissions(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, permission := range permissions {
				allowed, err := s.authorizeEducationPermission(r, educationDelegationScopeForRequest(permission, r))
				if err != nil {
					httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_authorization_failed"})
					return
				}
				if allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_permission_required", "permissions": permissions})
		})
	}
}

func educationDelegationScopeForRequest(permission string, r *http.Request) EducationDelegationScope {
	path := r.URL.Path
	for _, candidate := range []struct {
		prefix       string
		resourceType string
		parameter    string
	}{
		{"/api/education/portfolios/", "portfolio", "recordID"},
		{"/api/education/governance/meetings/", "meeting", "meetingID"},
		{"/api/education/decisions/", "decision", "decisionID"},
		{"/api/education/regulations/", "regulation", "recordID"},
		{"/api/education/personnel/", "personnel", "recordID"},
	} {
		if !strings.HasPrefix(path, candidate.prefix) {
			continue
		}
		if resourceID := strings.TrimSpace(chi.URLParam(r, candidate.parameter)); resourceID != "" {
			return EducationDelegationScope{PermissionCode: permission, ResourceType: candidate.resourceType, ResourceID: resourceID}
		}
	}
	return EducationDelegationScope{PermissionCode: permission, ResourceType: "institution"}
}

// delegationActiveAt centralizes the expiry interpretation used by API
// consumers. A record may still be persisted as accepted after its end date,
// but it never authorizes after that date.
func delegationActiveAt(status string, validFrom time.Time, validUntil *time.Time, now time.Time) bool {
	if status != "accepted" || validFrom.After(now) {
		return false
	}
	return validUntil == nil || !validUntil.Before(now)
}
