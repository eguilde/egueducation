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

func isDelegableEducationPermission(permission string) bool {
	permission = strings.TrimSpace(permission)
	return strings.HasPrefix(permission, "education.") && !strings.HasPrefix(permission, "education.delegations.")
}

// educationPermissionImplies keeps the conventional read subset explicit:
// managing a School resource necessarily includes reading that same resource
// family, while no other permission is broadened implicitly.
func educationPermissionImplies(granted, requested string) bool {
	granted = strings.TrimSpace(granted)
	requested = strings.TrimSpace(requested)
	if granted == requested && granted != "" {
		return true
	}
	if !strings.HasPrefix(granted, "education.") || !strings.HasPrefix(requested, "education.") {
		return false
	}
	return strings.HasSuffix(granted, ".manage") && requested == strings.TrimSuffix(granted, ".manage")+".read"
}

// educationDelegationScopeMatches is deliberately pure so handlers and future
// route middleware cannot accidentally broaden an institution-bound grant.
func educationDelegationScopeMatches(granted, requested EducationDelegationScope) bool {
	granted = normalizeEducationDelegationScope(granted)
	requested = normalizeEducationDelegationScope(requested)
	if !educationPermissionImplies(granted.PermissionCode, requested.PermissionCode) {
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
	if strings.HasSuffix(scope.PermissionCode, ".read") {
		managePermission := strings.TrimSuffix(scope.PermissionCode, ".read") + ".manage"
		direct, err = s.currentSubjectHasPermission(r, strings.TrimSpace(subject), managePermission)
		if err != nil || direct {
			return direct, err
		}
	}
	actorID, err := s.currentActorUserID(r, strings.TrimSpace(subject))
	if err != nil || actorID == "" {
		return false, err
	}
	// Delegation administration is intentionally never delegable. The direct
	// permission check above still permits an authorized director/administrator
	// to manage the ledger, but a grant can only authorize operational School
	// work.
	if !isDelegableEducationPermission(scope.PermissionCode) {
		return false, nil
	}
	granted, err := s.activeEducationDelegationGrantsForActor(r, actorID)
	if err != nil {
		return false, err
	}
	for _, candidate := range granted {
		if educationDelegationScopeMatches(candidate, scope) {
			return true, nil
		}
	}
	return false, nil
}

// activeEducationDelegationGrantsForActor is the sole request-time predicate
// for a usable delegation. Both authorization and the caller-facing active
// grant catalogue use it, so a catalogue entry can never claim authority that
// RequireEducationPermission would not grant. The SessionPool binds tenant,
// institution and actor context before this query runs.
func (s *Service) activeEducationDelegationGrantsForActor(r *http.Request, actorID string) ([]EducationDelegationScope, error) {
	granted := make([]EducationDelegationScope, 0)
	rows, err := s.pool.Query(r.Context(), `
		select permission_code, resource_type, coalesce(resource_id::text, '')
		from education_role_delegations delegation
		where delegation.tenant_code = public.current_tenant_code()
			and delegation.institution_id = public.current_institution_id()
			and delegation.delegate_user_id = $1::uuid
			and delegation.status = 'accepted'
			and delegation.valid_from <= current_date
			and (delegation.valid_until is null or delegation.valid_until >= current_date)
			and delegation.permission_code like 'education.%'
			and delegation.permission_code not like 'education.delegations.%'
			and exists (
				select 1
				from app_memberships delegate_membership
				where delegate_membership.user_id = delegation.delegate_user_id
					and delegate_membership.tenant_code = delegation.tenant_code
					and delegate_membership.position_code = 'director_adjunct'
					and delegate_membership.active = true
					and delegate_membership.start_date <= current_date
					and (delegate_membership.end_date is null or delegate_membership.end_date >= current_date)
			)
			and exists (
				select 1
				from app_memberships delegator_membership
				where delegator_membership.user_id = delegation.delegator_user_id
					and delegator_membership.tenant_code = delegation.tenant_code
					and delegator_membership.position_code = 'director'
					and delegator_membership.active = true
					and delegator_membership.start_date <= current_date
					and (delegator_membership.end_date is null or delegator_membership.end_date >= current_date)
			)
			and exists (
				select 1
				from (
					select direct_permission.permission_code
					from app_user_permissions direct_permission
					where direct_permission.user_id = delegation.delegator_user_id
						and direct_permission.tenant_code = delegation.tenant_code
					union
					select role_permission.permission_code
					from app_user_roles user_role
					join app_role_permissions role_permission on role_permission.role_code = user_role.role_code
					where user_role.user_id = delegation.delegator_user_id
						and user_role.tenant_code = delegation.tenant_code
					union
					select position_permission.permission_code
					from app_memberships permission_membership
					join app_position_permissions position_permission on position_permission.position_code = permission_membership.position_code
					where permission_membership.user_id = delegation.delegator_user_id
						and permission_membership.tenant_code = delegation.tenant_code
						and permission_membership.active = true
						and permission_membership.start_date <= current_date
						and (permission_membership.end_date is null or permission_membership.end_date >= current_date)
					union
					select role_permission.permission_code
					from app_memberships permission_membership
					join app_position_roles position_role on position_role.position_code = permission_membership.position_code
					join app_role_permissions role_permission on role_permission.role_code = position_role.role_code
					where permission_membership.user_id = delegation.delegator_user_id
						and permission_membership.tenant_code = delegation.tenant_code
						and permission_membership.active = true
						and permission_membership.start_date <= current_date
						and (permission_membership.end_date is null or permission_membership.end_date >= current_date)
				) delegator_permissions
				where delegator_permissions.permission_code = delegation.permission_code
			)
		order by delegation.permission_code asc, delegation.resource_type asc, coalesce(delegation.resource_id::text, '') asc, delegation.id asc
	`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate EducationDelegationScope
		if err := rows.Scan(&candidate.PermissionCode, &candidate.ResourceType, &candidate.ResourceID); err != nil {
			return nil, err
		}
		granted = append(granted, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return granted, nil
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
