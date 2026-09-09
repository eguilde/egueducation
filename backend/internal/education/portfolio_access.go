package education

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	portfolioReadOwnPermission   = "education.portfolios.read_own"
	portfolioManageOwnPermission = "education.portfolios.manage_own"
)

// portfolioAdminAllowed recognizes the historical portfolio permissions during
// migration, while new deployments use the explicitly school-scoped grants.
func (s *Service) portfolioAdminAllowed(r *http.Request, permission string) (bool, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return false, nil
	}
	for _, candidate := range []string{permission, "education.portfolios." + strings.TrimPrefix(permission, "education.portfolios.school."), "education.portfolios.manage"} {
		allowed, err := s.currentSubjectHasPermission(r, subject, candidate)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) ownPortfolioActorID(r *http.Request, requiredPermission string) (string, bool, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return "", false, nil
	}
	allowed, err := s.currentSubjectHasPermission(r, subject, requiredPermission)
	if err != nil || !allowed {
		return "", allowed, err
	}
	actorID, err := s.currentActorUserID(r, subject)
	return actorID, actorID != "", err
}

func (s *Service) requireOwnPortfolio(r *http.Request, recordID string, permission string) (string, bool, error) {
	actorID, allowed, err := s.ownPortfolioActorID(r, permission)
	if err != nil || !allowed {
		return "", allowed, err
	}
	var exists bool
	err = s.pool.QueryRow(r.Context(), `
		select exists(
			select 1 from education_portfolios
			where id = $1::uuid and institution_id = $2 and owner_user_id = $3::uuid
		)
	`, recordID, s.institutionID(r), actorID).Scan(&exists)
	return actorID, exists, err
}

func writePortfolioAccessFailure(w http.ResponseWriter, err error) {
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_authorization_failed"})
		return
	}
	httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_portfolio_access_denied"})
}

func scanPortfolioRecord(row pgx.Row, item *PortfolioRecord) error {
	return row.Scan(
		&item.ID, &item.PortfolioCode, &item.OwnerUserID, &item.OwnerPersonnelID,
		&item.OwnerName, &item.OwnerRole, &item.SchoolYear, &item.Status,
		&item.SectionCount, &item.LastUpdatedOn, &item.RetentionUntil,
		&item.TransferStatus, &item.AuthenticityDeclared, &item.ConsentCaptured,
		&item.Custodian, &item.InstitutionID, &item.Notes,
	)
}

const portfolioRecordColumns = `
	id::text, portfolio_code, coalesce(owner_user_id::text, ''),
	coalesce(owner_personnel_id::text, ''), owner_name, owner_role, school_year,
	status, section_count, to_char(last_updated_on, 'YYYY-MM-DD'),
	to_char(retention_until, 'YYYY-MM-DD'), transfer_status,
	authenticity_declared, consent_captured, custodian, institution_id, notes`

func (s *Service) loadOwnPortfolio(r *http.Request, recordID, actorID string) (PortfolioRecord, error) {
	var item PortfolioRecord
	err := scanPortfolioRecord(s.pool.QueryRow(r.Context(), `select `+portfolioRecordColumns+`
		from education_portfolios where id = $1::uuid and institution_id = $2 and owner_user_id = $3::uuid`,
		recordID, s.institutionID(r), actorID), &item)
	return item, err
}

func (s *Service) PortfolioOwnRecords(w http.ResponseWriter, r *http.Request) {
	actorID, allowed, err := s.ownPortfolioActorID(r, portfolioReadOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	rows, err := s.pool.Query(r.Context(), `select `+portfolioRecordColumns+`
		from education_portfolios
		where institution_id = $1 and owner_user_id = $2::uuid
		order by school_year desc, updated_at desc`, s.institutionID(r), actorID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolios_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioRecord, 0)
	for rows.Next() {
		var item PortfolioRecord
		if err := scanPortfolioRecord(rows, &item); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolios_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolios_failed"})
		return
	}
	pageSize := len(items)
	if pageSize < 1 {
		pageSize = 1
	}
	httpx.WritePage(w, http.StatusOK, items, len(items), 1, pageSize)
}

// listPortfolioArchiveAttachments intentionally queries the archive only by
// active institution and by a current version that has a real stored source.
// This provides attachment selection without granting generic eArhiva access.
func (s *Service) listPortfolioArchiveAttachments(r *http.Request, actorID string, query httpx.PageQuery) ([]PortfolioArchiveAttachment, int, error) {
	where := `where document.institution_id = $1
		and version.institution_id = document.institution_id
		and version.version_no = document.current_version_no
		and btrim(version.source_bucket) <> ''
		and btrim(version.source_object_key) <> ''
		and document.status = 'ready'
		and version.status = 'active'
		and exists (
			select 1 from education_portfolio_archive_attachment_grants attachment_grant
			where attachment_grant.institution_id = document.institution_id
				and attachment_grant.archive_document_id = document.id
				and attachment_grant.grantee_user_id = $2::uuid
		)`
	args := []any{s.institutionID(r), actorID}
	if title := strings.TrimSpace(query.Filters["title"]); title != "" {
		args = append(args, "%"+strings.ToLower(title)+"%")
		where += fmt.Sprintf(" and lower(document.title) like $%d", len(args))
	}

	var total int
	if err := s.pool.QueryRow(r.Context(), `
		select count(*)
		from archive_documents document
		join archive_document_versions version on version.document_id = document.id
		`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortColumn := "document.title"
	if query.Sort == "current_version_no" {
		sortColumn = "document.current_version_no"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select document.id::text, document.title, document.current_version_no
		from archive_documents document
		join archive_document_versions version on version.document_id = document.id
		%s
		order by %s %s, document.id
		limit $%d offset $%d
	`, where, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]PortfolioArchiveAttachment, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioArchiveAttachment
		if err := rows.Scan(&item.ID, &item.Title, &item.CurrentVersionNo); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) PortfolioOwnArchiveDocuments(w http.ResponseWriter, r *http.Request) {
	actorID, allowed, err := s.ownPortfolioActorID(r, portfolioReadOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"title": {}, "current_version_no": {},
	}, []string{"title"})
	if query.Sort == "" {
		query.Sort = "title"
	}
	items, total, err := s.listPortfolioArchiveAttachments(r, actorID, query)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_archive_documents_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

const portfolioArchiveGrantsManagePermission = "education.portfolios.archive_grants.manage"

func scanPortfolioArchiveAttachmentGrant(row pgx.Row, item *PortfolioArchiveAttachmentGrant) error {
	return row.Scan(&item.ID, &item.ArchiveDocumentID, &item.DocumentTitle, &item.GranteeUserID, &item.GranteeName, &item.GrantedByUserID, &item.CreatedAt)
}

const portfolioArchiveAttachmentGrantColumns = `
	grant.id::text, grant.archive_document_id::text, document.title,
	grant.grantee_user_id::text, grantee.name, coalesce(grant.granted_by_user_id::text, ''),
	to_char(grant.created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func (s *Service) PortfolioArchiveAttachmentGrants(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"document_title": {}, "grantee_name": {}}, []string{"document_title", "grantee_name"})
	if query.Sort == "" {
		query.Sort = "document_title"
	}
	where := "where grant.institution_id = $1"
	args := []any{s.institutionID(r)}
	if value := strings.TrimSpace(query.Filters["document_title"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where += fmt.Sprintf(" and lower(document.title) like $%d", len(args))
	}
	if value := strings.TrimSpace(query.Filters["grantee_name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where += fmt.Sprintf(" and lower(grantee.name) like $%d", len(args))
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from education_portfolio_archive_attachment_grants grant join archive_documents document on document.id = grant.archive_document_id join app_users grantee on grantee.id = grant.grantee_user_id `+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_grants_failed"})
		return
	}
	sortColumn := "document.title"
	if query.Sort == "grantee_name" {
		sortColumn = "grantee.name"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`select %s from education_portfolio_archive_attachment_grants grant join archive_documents document on document.id = grant.archive_document_id join app_users grantee on grantee.id = grant.grantee_user_id %s order by %s %s, grant.id limit $%d offset $%d`, portfolioArchiveAttachmentGrantColumns, where, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_grants_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioArchiveAttachmentGrant, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioArchiveAttachmentGrant
		if err := scanPortfolioArchiveAttachmentGrant(rows, &item); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_grants_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_grants_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioArchiveEligibleDocuments(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"title": {}, "current_version_no": {}}, []string{"title"})
	if query.Sort == "" {
		query.Sort = "title"
	}
	where := `where document.institution_id = $1 and version.institution_id = document.institution_id and version.version_no = document.current_version_no and document.status = 'ready' and version.status = 'active' and btrim(version.source_bucket) <> '' and btrim(version.source_object_key) <> ''`
	args := []any{s.institutionID(r)}
	if value := strings.TrimSpace(query.Filters["title"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where += fmt.Sprintf(" and lower(document.title) like $%d", len(args))
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), `select count(*) from archive_documents document join archive_document_versions version on version.document_id=document.id `+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_eligible_documents_failed"})
		return
	}
	sortColumn := "document.title"
	if query.Sort == "current_version_no" {
		sortColumn = "document.current_version_no"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`select document.id::text,document.title,document.current_version_no from archive_documents document join archive_document_versions version on version.document_id=document.id %s order by %s %s,document.id limit $%d offset $%d`, where, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_eligible_documents_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioArchiveAttachment, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioArchiveAttachment
		if err := rows.Scan(&item.ID, &item.Title, &item.CurrentVersionNo); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_eligible_documents_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_eligible_documents_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) listPortfolioArchiveEligibleUsers(r *http.Request, query httpx.PageQuery) ([]EligibleGovernanceUser, int, error) {
	where := `where tenant.institution_id = $1 and tenant.active and membership.active
		and user_row.status = 'active' and nullif(btrim(user_row.name), '') is not null`
	args := []any{s.institutionID(r)}
	if value := strings.TrimSpace(query.Filters["name"]); value != "" {
		args = append(args, "%"+strings.ToLower(value)+"%")
		where += fmt.Sprintf(" and lower(user_row.name) like $%d", len(args))
	}
	base := ` from app_users user_row join app_memberships membership on membership.user_id = user_row.id join app_tenants tenant on tenant.code = membership.tenant_code ` + where
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(distinct user_row.id)"+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`select distinct user_row.id::text, btrim(user_row.name)%s order by btrim(user_row.name) %s, user_row.id limit $%d offset $%d`, base, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]EligibleGovernanceUser, 0, query.PageSize)
	for rows.Next() {
		var item EligibleGovernanceUser
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// PortfolioArchiveEligibleUsers is intentionally independent of governance
// permissions: custodians can grant evidence without gaining governance data.
func (s *Service) PortfolioArchiveEligibleUsers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"name": {}}, []string{"name"})
	if query.Sort == "" {
		query.Sort = "name"
	}
	items, total, err := s.listPortfolioArchiveEligibleUsers(r, query)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_archive_eligible_users_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) CreatePortfolioArchiveAttachmentGrant(w http.ResponseWriter, r *http.Request) {
	var req CreatePortfolioArchiveAttachmentGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_archive_grant_payload"})
		return
	}
	if _, err := uuid.Parse(strings.TrimSpace(req.ArchiveDocumentID)); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_document_id"})
		return
	}
	if _, err := uuid.Parse(strings.TrimSpace(req.GranteeUserID)); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_grantee_user_id"})
		return
	}
	actorID, err := s.currentActorUserID(r, authruntime.CurrentSubjectFromRequest(r))
	if err != nil || actorID == "" {
		writePortfolioAccessFailure(w, err)
		return
	}
	var grantID string
	err = s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_archive_attachment_grants (institution_id,archive_document_id,grantee_user_id,granted_by_user_id)
		select $1,document.id,membership.user_id,$4::uuid from archive_documents document join archive_document_versions version on version.document_id=document.id and version.version_no=document.current_version_no join app_memberships membership on membership.user_id=$3::uuid join app_tenants tenant on tenant.code=membership.tenant_code and tenant.institution_id=$1 and tenant.active
		where document.id=$2::uuid and document.institution_id=$1 and document.status='ready' and version.status='active' and btrim(version.source_bucket)<>'' and btrim(version.source_object_key)<>'' and membership.active
		on conflict (institution_id,archive_document_id,grantee_user_id) do update set granted_by_user_id=excluded.granted_by_user_id
		returning id::text`, s.institutionID(r), req.ArchiveDocumentID, req.GranteeUserID, actorID).Scan(&grantID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_archive_grant_target_not_eligible"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_archive_grant_create_failed"})
		return
	}
	var item PortfolioArchiveAttachmentGrant
	err = scanPortfolioArchiveAttachmentGrant(s.pool.QueryRow(r.Context(), `select `+portfolioArchiveAttachmentGrantColumns+` from education_portfolio_archive_attachment_grants grant join archive_documents document on document.id=grant.archive_document_id join app_users grantee on grantee.id=grant.grantee_user_id where grant.id=$1::uuid and grant.institution_id=$2`, grantID, s.institutionID(r)), &item)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_archive_grant_load_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.archive_grant.create", "portfolio_archive_attachment_grant", item.ID, "Portfolio archive attachment grant created.", map[string]any{"archive_document_id": item.ArchiveDocumentID, "grantee_user_id": item.GranteeUserID})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) DeletePortfolioArchiveAttachmentGrant(w http.ResponseWriter, r *http.Request) {
	grantID := strings.TrimSpace(chi.URLParam(r, "grantID"))
	if _, err := uuid.Parse(grantID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_archive_grant_id"})
		return
	}
	var exists, referenced bool
	err := s.pool.QueryRow(r.Context(), `select exists(select 1 from education_portfolio_archive_attachment_grants where id=$1::uuid and institution_id=$2),exists(select 1 from education_portfolio_archive_attachment_grants grant join education_portfolio_documents evidence on evidence.archive_document_id=grant.archive_document_id join education_portfolios portfolio on portfolio.id=evidence.portfolio_id where grant.id=$1::uuid and grant.institution_id=$2 and portfolio.status in ('submitted','validated','transferred','archived'))`, grantID, s.institutionID(r)).Scan(&exists, &referenced)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_archive_grant_revoke_failed"})
		return
	}
	if !exists {
		writeEducationNotFound(w, "portfolio_archive_grant_not_found")
		return
	}
	if referenced {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_archive_grant_revoke_blocked_by_submitted_evidence"})
		return
	}
	if _, err := s.pool.Exec(r.Context(), `delete from education_portfolio_archive_attachment_grants where id=$1::uuid and institution_id=$2`, grantID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_archive_grant_revoke_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.archive_grant.revoke", "portfolio_archive_attachment_grant", grantID, "Portfolio archive attachment grant revoked.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) PortfolioOwnRecordDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioReadOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	item, err := s.loadOwnPortfolio(r, recordID, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_own_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func normalizeOwnPortfolioRequest(req *OwnPortfolioRequest) error {
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.LastUpdatedOn = strings.TrimSpace(req.LastUpdatedOn)
	req.Notes = strings.TrimSpace(req.Notes)
	if req.SchoolYear == "" || req.LastUpdatedOn == "" {
		return errors.New("missing fields")
	}
	if req.SectionCount < 0 {
		return errors.New("negative sections")
	}
	if _, err := time.Parse("2006-01-02", req.LastUpdatedOn); err != nil {
		return err
	}
	return nil
}

// Own routes never accept a teacher-controlled retention date. The existing
// three-year product baseline is calculated by the server and can only be
// extended later by an institution retention workflow.
func ownPortfolioRetentionUntil() string {
	return time.Now().UTC().AddDate(3, 0, 0).Format("2006-01-02")
}

func (s *Service) PortfolioOwnCreate(w http.ResponseWriter, r *http.Request) {
	actorID, allowed, err := s.ownPortfolioActorID(r, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	var req OwnPortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || normalizeOwnPortfolioRequest(&req) != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_own_portfolio_payload"})
		return
	}
	name, err := s.currentActorName(r, authruntime.CurrentSubjectFromRequest(r))
	if err != nil || strings.TrimSpace(name) == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_portfolio_owner_identity_required"})
		return
	}
	role := "Profesor"
	portfolioCode := fmt.Sprintf("PORT-CD-%d-%d", time.Now().UTC().Year(), time.Now().UTC().UnixNano())
	var item PortfolioRecord
	err = scanPortfolioRecord(s.pool.QueryRow(r.Context(), `
		insert into education_portfolios (
			portfolio_code, owner_user_id, owner_name, owner_role, school_year,
			status, section_count, last_updated_on, retention_until, transfer_status,
			authenticity_declared, consent_captured, custodian, institution_id, notes
		) values ($1, $2::uuid, $3, $4, $5, 'draft', $6, $7, $8, 'none', $9, $10, '', $11, $12)
		returning `+portfolioRecordColumns,
		portfolioCode, actorID, name, role, req.SchoolYear, req.SectionCount,
		req.LastUpdatedOn, ownPortfolioRetentionUntil(), req.AuthenticityDeclared,
		req.ConsentCaptured, s.institutionID(r), req.Notes), &item)
	if err != nil {
		var databaseError *pgconn.PgError
		if errors.As(err, &databaseError) && databaseError.Code == "23505" {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_own_portfolio_school_year_exists"})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.own.create", "portfolio_record", item.ID, "Own professional portfolio created.", map[string]any{"owner_user_id": actorID, "portfolio_code": item.PortfolioCode})
	httpx.JSON(w, http.StatusCreated, item)
}

// PortfolioOwnUpdate deliberately does not expose owner, status, transfer, or
// custody fields. Those fields require an institution-level command.
func (s *Service) PortfolioOwnUpdate(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	var req OwnPortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || normalizeOwnPortfolioRequest(&req) != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_own_portfolio_payload"})
		return
	}
	var item PortfolioRecord
	err = scanPortfolioRecord(s.pool.QueryRow(r.Context(), `
		update education_portfolios set school_year = $1, section_count = $2,
			last_updated_on = $3, retention_until = $4, authenticity_declared = $5,
			consent_captured = $6, notes = $7, updated_at = now()
		where id = $8::uuid and institution_id = $9 and owner_user_id = $10::uuid and status in ('draft', 'returned')
		returning `+portfolioRecordColumns,
		req.SchoolYear, req.SectionCount, req.LastUpdatedOn, ownPortfolioRetentionUntil(),
		req.AuthenticityDeclared, req.ConsentCaptured, req.Notes, recordID,
		s.institutionID(r), actorID), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_own_portfolio_not_editable"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.own.update", "portfolio_record", item.ID, "Own professional portfolio content updated.", map[string]any{"owner_user_id": actorID})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioOwnSubmit(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	var item PortfolioRecord
	err = scanPortfolioRecord(tx.QueryRow(r.Context(), `select `+portfolioRecordColumns+`
		from education_portfolios where id = $1::uuid and institution_id = $2 and owner_user_id = $3::uuid for update`,
		recordID, s.institutionID(r), actorID), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_own_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_failed"})
		return
	}
	// Snapshot the precise active archive version while holding the portfolio
	// lock. A later archive current-version change cannot change submitted
	// evidence, and the document-state trigger serializes concurrent edits.
	if _, err = tx.Exec(r.Context(), `
		update education_portfolio_documents evidence
		set archive_document_id = document.id,
			archive_version_id = version.id,
			archive_version_no = version.version_no,
			archive_source_bucket = version.source_bucket,
			archive_source_object_key = version.source_object_key,
			archive_sha256 = version.source_sha256
		from archive_documents document
		join archive_document_versions version on version.document_id = document.id and version.version_no = document.current_version_no
		join education_portfolio_archive_attachment_grants attachment_grant
			on attachment_grant.archive_document_id = document.id
			and attachment_grant.institution_id = document.institution_id
			and attachment_grant.grantee_user_id = $3::uuid
		where evidence.portfolio_id = $1::uuid and evidence.institution_id = $2
			and evidence.file_reference = 'archive://' || document.id::text
			and document.institution_id = $2 and document.status = 'ready' and version.status = 'active'
			and btrim(version.source_bucket) <> '' and btrim(version.source_object_key) <> ''
	`, recordID, s.institutionID(r), actorID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_snapshot_failed"})
		return
	}
	var evidenceCount, snapshottedCount int
	err = tx.QueryRow(r.Context(), `
		select count(*), count(*) filter (where archive_document_id is not null and archive_version_id is not null and archive_sha256 <> '')
		from education_portfolio_documents where portfolio_id = $1::uuid and institution_id = $2 and source_scope = 'portofoliu'
	`, recordID, s.institutionID(r)).Scan(&evidenceCount, &snapshottedCount)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_readiness_failed"})
		return
	}
	var missingComponents []string
	if err = tx.QueryRow(r.Context(), `
		select coalesce(array_agg(section.section_code || '/' || section.component_code order by section.sort_order), '{}')
		from education_portfolio_sections section
		where section.active and section.required and not exists (
			select 1 from education_portfolio_documents evidence
			where evidence.portfolio_id = $1::uuid and evidence.institution_id = $2
				and evidence.section_code = section.section_code and evidence.component_code = section.component_code
				and evidence.source_scope = 'portofoliu' and evidence.archive_version_id is not null
		)
	`, recordID, s.institutionID(r)).Scan(&missingComponents); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_readiness_failed"})
		return
	}
	if !item.AuthenticityDeclared || !item.ConsentCaptured || evidenceCount == 0 || evidenceCount != snapshottedCount || len(missingComponents) > 0 {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_submit_incomplete", "blockers": map[string]any{
			"authenticity_declared":  item.AuthenticityDeclared,
			"consent_captured":       item.ConsentCaptured,
			"unsnapshotted_evidence": evidenceCount - snapshottedCount,
			"missing_components":     missingComponents,
		}})
		return
	}
	err = scanPortfolioRecord(tx.QueryRow(r.Context(), `
		update education_portfolios set status = 'submitted', section_count = $4, updated_at = now()
		where id = $1::uuid and institution_id = $2 and owner_user_id = $3::uuid and status in ('draft', 'returned')
		returning `+portfolioRecordColumns, recordID, s.institutionID(r), actorID, evidenceCount), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_portfolio_submit_transition_invalid"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_submit_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.submit", "portfolio_record", item.ID, "Professional portfolio submitted for verification.", map[string]any{"owner_user_id": actorID})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioAdminTransition(w http.ResponseWriter, r *http.Request, nextStatus string, permission string) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	allowed, err := s.portfolioAdminAllowed(r, permission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	previous := "submitted"
	if nextStatus == "returned" {
		previous = "submitted"
	}
	var item PortfolioRecord
	err = scanPortfolioRecord(s.pool.QueryRow(r.Context(), `
		update education_portfolios set status = $1, updated_at = now()
		where id = $2::uuid and institution_id = $3 and status = $4
		returning `+portfolioRecordColumns, nextStatus, recordID, s.institutionID(r), previous), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_portfolio_transition_invalid"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transition_failed"})
		return
	}
	s.logAudit(r, "education.portfolios."+nextStatus, "portfolio_record", item.ID, "Professional portfolio state changed by institution administrator.", map[string]any{"status": nextStatus})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioAdminVerify(w http.ResponseWriter, r *http.Request) {
	s.PortfolioAdminTransition(w, r, "validated", "education.portfolios.verify")
}

func (s *Service) PortfolioAdminReturn(w http.ResponseWriter, r *http.Request) {
	s.PortfolioAdminTransition(w, r, "returned", "education.portfolios.request_corrections")
}

func (s *Service) requireOwnPortfolioContent(w http.ResponseWriter, r *http.Request, permission string, editable bool) bool {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	_, allowed, err := s.requireOwnPortfolio(r, recordID, permission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return false
	}
	if !editable {
		return true
	}
	var state string
	err = s.pool.QueryRow(r.Context(), `select status from education_portfolios where id = $1::uuid and institution_id = $2`, recordID, s.institutionID(r)).Scan(&state)
	if err != nil {
		writePortfolioAccessFailure(w, err)
		return false
	}
	if state != "draft" && state != "returned" {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "education_own_portfolio_not_editable"})
		return false
	}
	return true
}

// The established content handlers retain their DTOs and data validation; own
// wrappers add the owner + institution + editable-state authorization before
// delegating. The delegated queries remain bound to recordID/institution_id.
func (s *Service) PortfolioOwnDocuments(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioReadOwnPermission, false) {
		s.PortfolioDocuments(w, r)
	}
}
func (s *Service) PortfolioOwnDocumentCreate(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioManageOwnPermission, true) && s.prepareOwnPortfolioDocumentRequest(w, r) {
		s.CreatePortfolioDocument(w, r)
	}
}
func (s *Service) PortfolioOwnDocumentUpdate(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioManageOwnPermission, true) && s.prepareOwnPortfolioDocumentRequest(w, r) {
		s.UpdatePortfolioDocument(w, r)
	}
}
func (s *Service) PortfolioOwnDocumentDelete(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioManageOwnPermission, true) {
		s.DeletePortfolioDocument(w, r)
	}
}
func (s *Service) PortfolioOwnChecklist(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioReadOwnPermission, false) {
		s.PortfolioChecklistItems(w, r)
	}
}
func (s *Service) PortfolioOwnOpis(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioReadOwnPermission, false) {
		s.PortfolioOpisEntries(w, r)
	}
}
func (s *Service) PortfolioOwnOpisRegenerate(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioManageOwnPermission, true) {
		s.RegeneratePortfolioOpis(w, r)
	}
}
func (s *Service) PortfolioOwnReviews(w http.ResponseWriter, r *http.Request) {
	if s.requireOwnPortfolioContent(w, r, portfolioReadOwnPermission, false) {
		s.PortfolioReviews(w, r)
	}
}

func normalizeOwnPortfolioArchiveReference(reference string) (string, bool) {
	value := strings.TrimSpace(reference)
	if !strings.HasPrefix(value, "archive://") {
		return "", false
	}
	documentID, err := uuid.Parse(strings.TrimPrefix(value, "archive://"))
	if err != nil {
		return "", false
	}
	return "archive://" + documentID.String(), true
}

func ownPortfolioDocumentCommand(own OwnPortfolioDocumentRequest, archiveReference string) CreatePortfolioDocumentRequest {
	return CreatePortfolioDocumentRequest{
		SectionCode: own.SectionCode, ComponentCode: own.ComponentCode,
		DocumentTitle: own.DocumentTitle, SourceScope: "portofoliu",
		EvidenceType: own.EvidenceType, IssuedOn: own.IssuedOn, AddedOn: own.AddedOn,
		ChronologicalIndex: own.ChronologicalIndex, SensitiveData: own.SensitiveData,
		AuthenticityStatus: "declarat", FileReference: archiveReference, Notes: own.Notes,
	}
}

func (s *Service) prepareOwnPortfolioDocumentRequest(w http.ResponseWriter, r *http.Request) bool {
	var own OwnPortfolioDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&own); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_own_portfolio_document_payload"})
		return false
	}
	archiveReference, valid := normalizeOwnPortfolioArchiveReference(own.FileReference)
	if !valid {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "education_own_portfolio_document_archive_reference_required"})
		return false
	}
	actorID, allowed, err := s.ownPortfolioActorID(r, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return false
	}
	var archiveAuthorized bool
	if err := s.pool.QueryRow(r.Context(), `
		select exists(
			select 1
			from archive_documents document
			join archive_document_versions version
				on version.document_id = document.id
				and version.version_no = document.current_version_no
			where document.id = $1::uuid
				and document.institution_id = $2
				and document.status = 'ready'
				and version.status = 'active'
				and btrim(version.source_bucket) <> ''
				and btrim(version.source_object_key) <> ''
				and exists (
					select 1 from education_portfolio_archive_attachment_grants attachment_grant
					where attachment_grant.institution_id = document.institution_id
						and attachment_grant.archive_document_id = document.id
						and attachment_grant.grantee_user_id = $3::uuid
				)
		)
	`, strings.TrimPrefix(archiveReference, "archive://"), s.institutionID(r), actorID).Scan(&archiveAuthorized); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_document_archive_lookup_failed"})
		return false
	}
	if !archiveAuthorized {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_own_portfolio_document_archive_access_denied"})
		return false
	}
	command := ownPortfolioDocumentCommand(own, archiveReference)
	payload, err := json.Marshal(command)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_own_portfolio_document_prepare_failed"})
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(payload))
	r.ContentLength = int64(len(payload))
	return true
}
