package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

type OfferEducationDelegationRequest struct {
	DelegateUserID string `json:"delegate_user_id"`
	PermissionCode string `json:"permission_code"`
	ResourceType   string `json:"resource_type"`
	ResourceID     string `json:"resource_id"`
	ValidFrom      string `json:"valid_from"`
	ValidUntil     string `json:"valid_until"`
	Notes          string `json:"notes"`
}

type EducationDelegation struct {
	ID               string `json:"id"`
	TenantCode       string `json:"tenant_code"`
	InstitutionID    string `json:"institution_id"`
	DelegatorUserID  string `json:"delegator_user_id"`
	DelegateUserID   string `json:"delegate_user_id"`
	DelegatorName    string `json:"delegator_name"`
	DelegateName     string `json:"delegate_name"`
	PermissionCode   string `json:"permission_code"`
	ResourceType     string `json:"resource_type"`
	ResourceID       string `json:"resource_id"`
	Status           string `json:"status"`
	ValidFrom        string `json:"valid_from"`
	ValidUntil       string `json:"valid_until"`
	OfferedByUserID  string `json:"offered_by_user_id"`
	OfferedAt        string `json:"offered_at"`
	AcceptedByUserID string `json:"accepted_by_user_id"`
	AcceptedAt       string `json:"accepted_at"`
	RevokedByUserID  string `json:"revoked_by_user_id"`
	RevokedAt        string `json:"revoked_at"`
	ExpiredByUserID  string `json:"expired_by_user_id"`
	ExpiredAt        string `json:"expired_at"`
	Notes            string `json:"notes"`
}

type EducationDelegationPage struct {
	Items    []EducationDelegation `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type EducationDelegationEligibleAdjunct struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

func parseEducationDelegationDate(value string, required bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		if required {
			return nil, errors.New("required date")
		}
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func scanEducationDelegation(row pgx.Row, item *EducationDelegation) error {
	return row.Scan(
		&item.ID, &item.TenantCode, &item.InstitutionID, &item.DelegatorUserID, &item.DelegateUserID,
		&item.DelegatorName, &item.DelegateName,
		&item.PermissionCode, &item.ResourceType, &item.ResourceID, &item.Status, &item.ValidFrom, &item.ValidUntil,
		&item.OfferedByUserID, &item.OfferedAt, &item.AcceptedByUserID, &item.AcceptedAt,
		&item.RevokedByUserID, &item.RevokedAt, &item.ExpiredByUserID, &item.ExpiredAt, &item.Notes,
	)
}

const educationDelegationColumns = `
	id::text, tenant_code, institution_id, delegator_user_id::text, delegate_user_id::text,
	coalesce((select account.name from app_users account where account.id=delegator_user_id), ''),
	coalesce((select account.name from app_users account where account.id=delegate_user_id), ''),
	permission_code, resource_type, coalesce(resource_id::text, ''), status,
	to_char(valid_from, 'YYYY-MM-DD'), coalesce(to_char(valid_until, 'YYYY-MM-DD'), ''),
	offered_by_user_id::text, to_char(offered_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	coalesce(accepted_by_user_id::text, ''), coalesce(to_char(accepted_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
	coalesce(revoked_by_user_id::text, ''), coalesce(to_char(revoked_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
	coalesce(expired_by_user_id::text, ''), coalesce(to_char(expired_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''), notes`

func (s *Service) EducationDelegations(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"offered_at": {}, "valid_until": {}, "status": {}, "permission_code": {},
	}, []string{"status", "permission_code", "delegate_user_id", "delegate_name"})
	clauses := []string{"tenant_code=public.current_tenant_code()", "institution_id=public.current_institution_id()"}
	args := make([]any, 0, 5)
	for _, filter := range []struct{ key, column string }{
		{"status", "status"}, {"permission_code", "permission_code"}, {"delegate_user_id", "delegate_user_id::text"},
		{"delegate_name", "coalesce((select account.name from app_users account where account.id=education_role_delegations.delegate_user_id), '')"},
	} {
		if value := strings.TrimSpace(query.Filters[filter.key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", filter.column, len(args)))
		}
	}
	where := " where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_role_delegations"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegations_list_failed"})
		return
	}
	sortColumns := map[string]string{
		"offered_at": "offered_at", "valid_from": "valid_from", "valid_until": "valid_until",
		"status": "status", "permission_code": "permission_code",
		"delegator_name": "coalesce((select account.name from app_users account where account.id=education_role_delegations.delegator_user_id), '')",
		"delegate_name":  "coalesce((select account.name from app_users account where account.id=education_role_delegations.delegate_user_id), '')",
	}
	sortColumn := sortColumns[query.Sort]
	if sortColumn == "" {
		sortColumn = "offered_at"
	}
	direction := "desc"
	if strings.EqualFold(query.Direction, "asc") {
		direction = "asc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), "select "+educationDelegationColumns+" from education_role_delegations"+where+fmt.Sprintf(" order by %s %s, id desc limit $%d offset $%d", sortColumn, direction, len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegations_list_failed"})
		return
	}
	defer rows.Close()
	items := make([]EducationDelegation, 0, query.PageSize)
	for rows.Next() {
		var item EducationDelegation
		if err := scanEducationDelegation(rows, &item); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegations_list_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegations_list_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EducationDelegationEligibleAdjuncts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select user_account.id::text, user_account.name, user_account.email
		from app_memberships membership
		join app_users user_account on user_account.id=membership.user_id
		where membership.tenant_code=public.current_tenant_code()
			and user_account.status='active'
			and membership.position_code='director_adjunct' and membership.active
			and membership.start_date<=current_date
			and (membership.end_date is null or membership.end_date>=current_date)
		group by user_account.id, user_account.name, user_account.email
		order by user_account.name, user_account.id
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_adjuncts_failed"})
		return
	}
	defer rows.Close()
	items := make([]EducationDelegationEligibleAdjunct, 0)
	for rows.Next() {
		var item EducationDelegationEligibleAdjunct
		if err := rows.Scan(&item.UserID, &item.Name, &item.Email); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_adjuncts_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_adjuncts_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// EducationDelegationEligiblePermissions returns only education permissions
// that the authenticated delegator currently holds. The database trigger
// rechecks the same condition when the offer is persisted.
func (s *Service) EducationDelegationEligiblePermissions(w http.ResponseWriter, r *http.Request) {
	actorID, err := s.currentActorUserID(r, authruntime.CurrentSubjectFromRequest(r))
	if err != nil || actorID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_delegation_actor_required"})
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		select distinct permission_code from (
			select permission.permission_code
			from app_user_permissions permission
			where permission.user_id=$1::uuid and permission.tenant_code=public.current_tenant_code()
			union
			select role_permission.permission_code
			from app_user_roles user_role
			join app_role_permissions role_permission on role_permission.role_code=user_role.role_code
			where user_role.user_id=$1::uuid and user_role.tenant_code=public.current_tenant_code()
			union
			select position_permission.permission_code
			from app_memberships membership
			join app_position_permissions position_permission on position_permission.position_code=membership.position_code
			where membership.user_id=$1::uuid and membership.tenant_code=public.current_tenant_code()
				and membership.active and membership.start_date<=current_date
				and (membership.end_date is null or membership.end_date>=current_date)
			union
			select role_permission.permission_code
			from app_memberships membership
			join app_position_roles position_role on position_role.position_code=membership.position_code
			join app_role_permissions role_permission on role_permission.role_code=position_role.role_code
			where membership.user_id=$1::uuid and membership.tenant_code=public.current_tenant_code()
				and membership.active and membership.start_date<=current_date
				and (membership.end_date is null or membership.end_date>=current_date)
		) effective
		where permission_code like 'education.%' and permission_code not like 'education.delegations.%'
		order by permission_code
	`, actorID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_permissions_failed"})
		return
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_permissions_failed"})
			return
		}
		items = append(items, code)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_permissions_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// OfferEducationDelegation validates the public contract while the database
// trigger remains authoritative for tenant, role, scope, and provenance.
func (s *Service) OfferEducationDelegation(w http.ResponseWriter, r *http.Request) {
	var req OfferEducationDelegationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_payload"})
		return
	}
	req.DelegateUserID = strings.TrimSpace(req.DelegateUserID)
	req.PermissionCode = strings.TrimSpace(req.PermissionCode)
	req.ResourceType = strings.ToLower(strings.TrimSpace(req.ResourceType))
	req.ResourceID = strings.TrimSpace(req.ResourceID)
	if req.ResourceType == "" {
		req.ResourceType = "institution"
	}
	if _, err := uuid.Parse(req.DelegateUserID); err != nil || !strings.HasPrefix(req.PermissionCode, "education.") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_scope"})
		return
	}
	if (req.ResourceType == "institution" && req.ResourceID != "") || (req.ResourceType != "institution" && !validEducationDelegationResource(req.ResourceType, req.ResourceID)) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_resource_scope"})
		return
	}
	if req.ResourceType != "institution" {
		exists, err := s.educationDelegationResourceExists(r, req.ResourceType, req.ResourceID)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_resource_check_failed"})
			return
		}
		if !exists {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_delegation_resource_not_found"})
			return
		}
	}
	validFrom, err := parseEducationDelegationDate(req.ValidFrom, false)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_valid_from"})
		return
	}
	validUntil, err := parseEducationDelegationDate(req.ValidUntil, false)
	if err != nil || (validFrom != nil && validUntil != nil && validUntil.Before(*validFrom)) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_valid_until"})
		return
	}
	actorID, err := s.currentActorUserID(r, authruntime.CurrentSubjectFromRequest(r))
	if err != nil || actorID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_delegation_actor_required"})
		return
	}
	var item EducationDelegation
	sql := `
		insert into education_role_delegations (
			tenant_code, institution_id, delegator_user_id, delegate_user_id, permission_code,
			resource_type, resource_id, valid_from, valid_until, offered_by_user_id, notes
		) values (
			public.current_tenant_code(), public.current_institution_id(), $1::uuid, $2::uuid, $3,
			$4, nullif($5, '')::uuid, coalesce($6::date, current_date), $7::date, $1::uuid, $8
		) returning ` + educationDelegationColumns
	row := s.pool.QueryRow(r.Context(), sql, actorID, req.DelegateUserID, req.PermissionCode, req.ResourceType, req.ResourceID, validFrom, validUntil, strings.TrimSpace(req.Notes))
	err = scanEducationDelegation(row, &item)
	if err != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_delegation_offer_forbidden"})
		return
	}
	s.logAudit(r, "education.delegations.offer", "education_role_delegation", item.ID, "Director offered delegated authority.", map[string]any{"delegate_user_id": item.DelegateUserID, "permission_code": item.PermissionCode, "resource_type": item.ResourceType, "resource_id": item.ResourceID})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) educationDelegationResourceExists(r *http.Request, resourceType, resourceID string) (bool, error) {
	query := ""
	switch resourceType {
	case "portfolio":
		query = `select exists(select 1 from education_portfolios where id=$1::uuid and institution_id=public.current_institution_id())`
	case "meeting":
		query = `select exists(select 1 from education_meetings where id=$1::uuid and institution_id=public.current_institution_id())`
	case "decision":
		query = `select exists(select 1 from education_decisions where id=$1::uuid and institution_id=public.current_institution_id())`
	case "regulation":
		query = `select exists(select 1 from education_regulations where id=$1::uuid and institution_id=public.current_institution_id())`
	case "personnel":
		query = `select exists(select 1 from education_personnel where id=$1::uuid and institution_id=public.current_institution_id())`
	default:
		return false, nil
	}
	var exists bool
	err := s.pool.QueryRow(r.Context(), query, resourceID).Scan(&exists)
	return exists, err
}

func validEducationDelegationResource(resourceType, resourceID string) bool {
	if _, err := uuid.Parse(strings.TrimSpace(resourceID)); err != nil {
		return false
	}
	switch resourceType {
	case "portfolio", "meeting", "decision", "regulation", "personnel":
		return true
	default:
		return false
	}
}

func (s *Service) transitionEducationDelegation(w http.ResponseWriter, r *http.Request, status, auditAction string) {
	id := strings.TrimSpace(chi.URLParam(r, "delegationID"))
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_education_delegation_id"})
		return
	}
	var item EducationDelegation
	sql := `
		update education_role_delegations
		set status = $1
		where id = $2::uuid
			and tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
		returning ` + educationDelegationColumns
	row := s.pool.QueryRow(r.Context(), sql, status, id)
	err := scanEducationDelegation(row, &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_delegation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_delegation_transition_forbidden"})
		return
	}
	s.logAudit(r, auditAction, "education_role_delegation", item.ID, "Education delegation state transitioned.", map[string]any{"status": item.Status})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) AcceptEducationDelegation(w http.ResponseWriter, r *http.Request) {
	s.transitionEducationDelegation(w, r, "accepted", "education.delegations.accept")
}

func (s *Service) RevokeEducationDelegation(w http.ResponseWriter, r *http.Request) {
	s.transitionEducationDelegation(w, r, "revoked", "education.delegations.revoke")
}

func (s *Service) ExpireEducationDelegation(w http.ResponseWriter, r *http.Request) {
	s.transitionEducationDelegation(w, r, "expired", "education.delegations.expire")
}
