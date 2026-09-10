package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (s *Service) PortfolioValorifications(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"valorification_code": {},
		"scope":               {},
		"status":              {},
		"requested_by":        {},
		"target_institution":  {},
		"target_reference":    {},
		"started_on":          {},
		"completed_on":        {},
		"notes":               {},
	}, []string{"valorification_code", "scope", "status", "requested_by", "target_institution", "target_reference", "started_on", "completed_on", "notes"})
	if query.Sort == "" {
		query.Sort = "started_on"
	}

	whereClause, args := buildPortfolioValorificationFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_valorifications epv "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_valorifications_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			id::text,
			portfolio_id::text,
			valorification_code,
			scope,
			status,
			requested_by,
			target_institution,
			target_reference,
			to_char(started_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			institution_id,
			notes
		from education_portfolio_valorifications epv
		%s
		order by %s %s, started_on desc, valorification_code
		limit $%d offset $%d
	`, whereClause, portfolioValorificationSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_valorifications_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioValorificationEvent, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioValorificationEvent
		if err := rows.Scan(
			&item.ID,
			&item.PortfolioID,
			&item.ValorificationCode,
			&item.Scope,
			&item.Status,
			&item.RequestedBy,
			&item.TargetInstitution,
			&item.TargetReference,
			&item.StartedOn,
			&item.CompletedOn,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_valorifications_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_valorifications_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioValorificationDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PortfolioValorificationEvent
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			portfolio_id::text,
			valorification_code,
			scope,
			status,
			requested_by,
			target_institution,
			target_reference,
			to_char(started_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			institution_id,
			notes
		from education_portfolio_valorifications
		where id = $1 and portfolio_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.ValorificationCode,
		&item.Scope,
		&item.Status,
		&item.RequestedBy,
		&item.TargetInstitution,
		&item.TargetReference,
		&item.StartedOn,
		&item.CompletedOn,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_valorification_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_valorification_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioValorification(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioValorificationEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_payload"})
		return
	}

	normalizePortfolioValorificationRequest(&req)
	if req.Scope == "" || req.Status == "" || req.StartedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_valorification_fields"})
		return
	}
	if !containsString(portfolioValorificationScopes(), req.Scope) || !containsString([]string{"planificat", "in_pregatire", "transmis", "validat", "finalizat"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.StartedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_started_on"})
		return
	}
	if req.CompletedOn != "" {
		completedOn, err := time.Parse("2006-01-02", req.CompletedOn)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_completed_on"})
			return
		}
		startedOn, _ := time.Parse("2006-01-02", req.StartedOn)
		if completedOn.Before(startedOn) {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_interval"})
			return
		}
	}

	code := newEducationCode("VAL")
	var item PortfolioValorificationEvent
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_valorifications (
			portfolio_id, valorification_code, scope, status, requested_by, target_institution, target_reference, started_on, completed_on, institution_id, notes
		)
		select ep.id, $2, $3, $4, $5, $6, $7, $8, $9, ep.institution_id, $10
		from education_portfolios ep
		where ep.id = $1 and ep.institution_id = $11
		returning
			id::text,
			portfolio_id::text,
			valorification_code,
			scope,
			status,
			requested_by,
			target_institution,
			target_reference,
			to_char(started_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			institution_id,
			notes
	`, recordID, code, req.Scope, req.Status, req.RequestedBy, req.TargetInstitution, req.TargetReference, req.StartedOn, nullableDate(req.CompletedOn), req.Notes, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.ValorificationCode,
		&item.Scope,
		&item.Status,
		&item.RequestedBy,
		&item.TargetInstitution,
		&item.TargetReference,
		&item.StartedOn,
		&item.CompletedOn,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_create_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.valorification.create", "portfolio_valorification", item.ID, "Portfolio valorification flow created.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"valorification_code": item.ValorificationCode,
		"scope":               item.Scope,
		"status":              item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioValorification(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePortfolioValorificationEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_payload"})
		return
	}

	normalizePortfolioValorificationRequest(&req)
	if req.Scope == "" || req.Status == "" || req.StartedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_valorification_fields"})
		return
	}
	if !containsString(portfolioValorificationScopes(), req.Scope) || !containsString([]string{"planificat", "in_pregatire", "transmis", "validat", "finalizat"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.StartedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_started_on"})
		return
	}
	if req.CompletedOn != "" {
		completedOn, err := time.Parse("2006-01-02", req.CompletedOn)
		if err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_completed_on"})
			return
		}
		startedOn, _ := time.Parse("2006-01-02", req.StartedOn)
		if completedOn.Before(startedOn) {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_interval"})
			return
		}
	}

	var item PortfolioValorificationEvent
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolio_valorifications
		set scope = $1,
			status = $2,
			requested_by = $3,
			target_institution = $4,
			target_reference = $5,
			started_on = $6,
			completed_on = $7,
			notes = $8,
			updated_at = now()
		where id = $9 and portfolio_id = $10 and institution_id = $11
		returning
			id::text,
			portfolio_id::text,
			valorification_code,
			scope,
			status,
			requested_by,
			target_institution,
			target_reference,
			to_char(started_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			institution_id,
			notes
	`, req.Scope, req.Status, req.RequestedBy, req.TargetInstitution, req.TargetReference, req.StartedOn, nullableDate(req.CompletedOn), req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.PortfolioID,
		&item.ValorificationCode,
		&item.Scope,
		&item.Status,
		&item.RequestedBy,
		&item.TargetInstitution,
		&item.TargetReference,
		&item.StartedOn,
		&item.CompletedOn,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_portfolio_valorification_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_update_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.valorification.update", "portfolio_valorification", item.ID, "Portfolio valorification flow updated.", map[string]any{
		"portfolio_id":        item.PortfolioID,
		"valorification_code": item.ValorificationCode,
		"scope":               item.Scope,
		"status":              item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioValorification(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `
		update education_portfolio_valorifications valorification
		set withdrawn_at = now(), withdrawn_by_subject = $1,
			withdrawal_reason = 'withdrawn through legacy delete command', updated_at = now()
		from education_portfolios portfolio
		where valorification.id = $2::uuid and valorification.portfolio_id = $3::uuid and valorification.institution_id = $4
			and portfolio.id = valorification.portfolio_id and portfolio.institution_id = valorification.institution_id
			and portfolio.status in ('draft', 'returned') and portfolio.activity_ceased_on is null
			and portfolio.retention_until is null and not portfolio.legal_hold_active
			and valorification.withdrawn_at is null
	`, strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r)), itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(r.Context(), `select exists(select 1 from education_portfolio_valorifications where id = $1::uuid and portfolio_id = $2::uuid and institution_id = $3)`, itemID, recordID, s.institutionID(r)).Scan(&exists); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_delete_failed"})
			return
		}
		if !exists {
			writeEducationNotFound(w, "education_portfolio_valorification_not_found")
			return
		}
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_valorification_withdrawal_blocked"})
		return
	}
	s.logAudit(r, "education.portfolios.valorification.withdraw", "portfolio_valorification", itemID, "Portfolio valorification withdrawn; evidence tombstone retained.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func buildPortfolioValorificationFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epv.portfolio_id = $1", "epv.institution_id = $2", "epv.withdrawn_at is null"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"valorification_code": "epv.valorification_code",
		"scope":               "epv.scope",
		"status":              "epv.status",
		"requested_by":        "epv.requested_by",
		"target_institution":  "epv.target_institution",
		"target_reference":    "epv.target_reference",
		"started_on":          "to_char(epv.started_on, 'YYYY-MM-DD')",
		"completed_on":        "to_char(epv.completed_on, 'YYYY-MM-DD')",
		"notes":               "epv.notes",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func portfolioValorificationSortColumn(value string) string {
	switch value {
	case "valorification_code":
		return "epv.valorification_code"
	case "scope":
		return "epv.scope"
	case "status":
		return "epv.status"
	case "requested_by":
		return "epv.requested_by"
	case "target_institution":
		return "epv.target_institution"
	case "started_on":
		return "epv.started_on"
	case "completed_on":
		return "epv.completed_on"
	default:
		return "epv.started_on"
	}
}

func normalizePortfolioValorificationRequest(req *CreatePortfolioValorificationEventRequest) {
	req.Scope = strings.TrimSpace(req.Scope)
	req.Status = strings.TrimSpace(req.Status)
	req.RequestedBy = strings.TrimSpace(req.RequestedBy)
	req.TargetInstitution = strings.TrimSpace(req.TargetInstitution)
	req.TargetReference = strings.TrimSpace(req.TargetReference)
	req.StartedOn = strings.TrimSpace(req.StartedOn)
	req.CompletedOn = strings.TrimSpace(req.CompletedOn)
	req.Notes = strings.TrimSpace(req.Notes)
}

func portfolioValorificationScopes() []string {
	return []string{
		"licentiere",
		"debut",
		"definitivat",
		"grad_ii",
		"grad_i",
		"evaluare_profesionala",
		"mobilitate",
		"dezvoltare_profesionala",
		"inspectie_scolara",
		"evaluare_externa_calitate",
		"gradatie_merit",
		"distinctie_premiu",
	}
}

func portfolioValorificationPurposeAllowed(sourceScope, purpose string) bool {
	evaluationPurposes := []string{
		"licentiere", "debut", "definitivat", "grad_ii", "grad_i",
		"evaluare_profesionala", "dezvoltare_profesionala", "inspectie_scolara",
		"evaluare_externa_calitate", "distinctie_premiu",
	}
	switch sourceScope {
	case "evaluare_profesionala":
		return containsString(evaluationPurposes, purpose)
	case "mobilitate":
		return purpose == "mobilitate"
	case "gradatie_merit":
		return purpose == "gradatie_merit" || purpose == "distinctie_premiu"
	default:
		return false
	}
}

func nullableDate(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// PortfolioValorificationPackage is the strict successor to the legacy
// free-text valorification event. Its source is always a real record in the
// corresponding education domain and document evidence is pinned to an
// eArhiva version by the database.
type PortfolioValorificationPackage struct {
	ID                   string `json:"id"`
	TenantCode           string `json:"tenant_code"`
	InstitutionID        string `json:"institution_id"`
	PortfolioID          string `json:"portfolio_id"`
	Scope                string `json:"scope"`
	Purpose              string `json:"purpose"`
	SourceEvaluationID   string `json:"source_evaluation_id,omitempty"`
	SourceMobilityCaseID string `json:"source_mobility_case_id,omitempty"`
	SourceMeritGrantID   string `json:"source_merit_grant_id,omitempty"`
	Status               string `json:"status"`
	CreatedBySubject     string `json:"created_by_subject"`
	CreatedAt            string `json:"created_at"`
	SubmittedBySubject   string `json:"submitted_by_subject,omitempty"`
	SubmittedAt          string `json:"submitted_at,omitempty"`
	ValidatedBySubject   string `json:"validated_by_subject,omitempty"`
	ValidatedAt          string `json:"validated_at,omitempty"`
	CompletedBySubject   string `json:"completed_by_subject,omitempty"`
	CompletedAt          string `json:"completed_at,omitempty"`
}

type CreatePortfolioValorificationPackageRequest struct {
	Scope                string `json:"scope"`
	Purpose              string `json:"purpose"`
	SourceEvaluationID   string `json:"source_evaluation_id"`
	SourceMobilityCaseID string `json:"source_mobility_case_id"`
	SourceMeritGrantID   string `json:"source_merit_grant_id"`
}

type AddPortfolioValorificationPackageDocumentRequest struct {
	ArchiveDocumentID string `json:"archive_document_id"`
	ArchiveVersionID  string `json:"archive_version_id"`
}

type AdvancePortfolioValorificationPackageRequest struct {
	Action string `json:"action"`
}

type PortfolioValorificationPackageDocument struct {
	ID                     string `json:"id"`
	PackageID              string `json:"package_id"`
	ArchiveDocumentID      string `json:"archive_document_id"`
	ArchiveVersionID       string `json:"archive_version_id"`
	ArchiveVersionNo       int    `json:"archive_version_no"`
	ArchiveSourceBucket    string `json:"archive_source_bucket"`
	ArchiveSourceObjectKey string `json:"archive_source_object_key"`
	ArchiveSHA256          string `json:"archive_sha256"`
	CreatedBySubject       string `json:"created_by_subject"`
	CreatedAt              string `json:"created_at"`
}

type PortfolioValorificationEligibleSource struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
	Label string `json:"label"`
}

type PortfolioValorificationEligibleArchiveVersion struct {
	ArchiveDocumentID string `json:"archive_document_id"`
	ArchiveVersionID  string `json:"archive_version_id"`
	VersionNo         int    `json:"version_no"`
	Title             string `json:"title"`
}

const portfolioValorificationPackageColumns = `
	id::text, tenant_code, institution_id, portfolio_id::text, scope, purpose,
	coalesce(source_evaluation_id::text,''), coalesce(source_mobility_case_id::text,''), coalesce(source_merit_grant_id::text,''),
	status, created_by_subject, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
	submitted_by_subject, coalesce(to_char(submitted_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
	validated_by_subject, coalesce(to_char(validated_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),
	completed_by_subject, coalesce(to_char(completed_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'')`

func scanPortfolioValorificationPackage(row interface{ Scan(...any) error }) (PortfolioValorificationPackage, error) {
	var item PortfolioValorificationPackage
	err := row.Scan(&item.ID, &item.TenantCode, &item.InstitutionID, &item.PortfolioID, &item.Scope, &item.Purpose,
		&item.SourceEvaluationID, &item.SourceMobilityCaseID, &item.SourceMeritGrantID,
		&item.Status, &item.CreatedBySubject, &item.CreatedAt, &item.SubmittedBySubject, &item.SubmittedAt,
		&item.ValidatedBySubject, &item.ValidatedAt, &item.CompletedBySubject, &item.CompletedAt)
	return item, err
}

const portfolioValorificationPackageDocumentColumns = `
	id::text, package_id::text, archive_document_id::text, archive_version_id::text,
	archive_version_no, archive_source_bucket, archive_source_object_key, archive_sha256,
	created_by_subject, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`

func scanPortfolioValorificationPackageDocument(row interface{ Scan(...any) error }) (PortfolioValorificationPackageDocument, error) {
	var item PortfolioValorificationPackageDocument
	err := row.Scan(&item.ID, &item.PackageID, &item.ArchiveDocumentID, &item.ArchiveVersionID,
		&item.ArchiveVersionNo, &item.ArchiveSourceBucket, &item.ArchiveSourceObjectKey, &item.ArchiveSHA256,
		&item.CreatedBySubject, &item.CreatedAt)
	return item, err
}

func (s *Service) PortfolioValorificationPackages(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"scope": {}, "purpose": {}, "status": {}}, []string{"scope", "purpose", "status"})
	if query.Sort == "" {
		query.Sort = "created_at"
	}
	where := []string{"portfolio_id = $1::uuid", "institution_id = $2"}
	args := []any{recordID, s.institutionID(r)}
	for _, filter := range []string{"scope", "purpose", "status"} {
		if value := strings.TrimSpace(query.Filters[filter]); value != "" {
			args = append(args, value)
			where = append(where, filter+" = $"+fmt.Sprint(len(args)))
		}
	}
	whereSQL := " where " + strings.Join(where, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_valorification_packages"+whereSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_packages_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	sortColumn := map[string]string{"created_at": "created_at", "scope": "scope", "purpose": "purpose", "status": "status"}[query.Sort]
	if sortColumn == "" {
		sortColumn = "created_at"
	}
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf("select %s from education_portfolio_valorification_packages%s order by %s %s, id limit $%d offset $%d", portfolioValorificationPackageColumns, whereSQL, sortColumn, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_packages_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioValorificationPackage, 0, query.PageSize)
	for rows.Next() {
		item, err := scanPortfolioValorificationPackage(rows)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_packages_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_packages_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioValorificationPackageDetail(w http.ResponseWriter, r *http.Request) {
	recordID, itemID := strings.TrimSpace(chi.URLParam(r, "recordID")), strings.TrimSpace(chi.URLParam(r, "itemID"))
	item, err := scanPortfolioValorificationPackage(s.pool.QueryRow(r.Context(), `
		select `+portfolioValorificationPackageColumns+`
		from education_portfolio_valorification_packages
		where id=$1::uuid and portfolio_id=$2::uuid and institution_id=$3
	`, itemID, recordID, s.institutionID(r)))
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_valorification_package_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_package_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioValorificationPackageDocuments(w http.ResponseWriter, r *http.Request) {
	recordID, itemID := strings.TrimSpace(chi.URLParam(r, "recordID")), strings.TrimSpace(chi.URLParam(r, "itemID"))
	rows, err := s.pool.Query(r.Context(), `
		select `+portfolioValorificationPackageDocumentColumns+`
		from education_portfolio_valorification_package_documents document
		join education_portfolio_valorification_packages package on package.id=document.package_id
		where package.id=$1::uuid and package.portfolio_id=$2::uuid and package.institution_id=$3
		order by document.created_at, document.id
	`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_package_documents_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioValorificationPackageDocument, 0)
	for rows.Next() {
		item, err := scanPortfolioValorificationPackageDocument(rows)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_package_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_package_documents_scan_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) PortfolioValorificationEligibleSources(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	scope := strings.TrimSpace(r.URL.Query().Get("scope"))
	query := ""
	switch scope {
	case "evaluare_profesionala":
		query = `select source.id::text, 'evaluare_profesionala', source.evaluation_code || ' · ' || source.full_name || ' · ' || source.status
			from education_evaluations source join education_portfolios portfolio on portfolio.owner_personnel_id=source.personnel_id and portfolio.school_year=source.school_year
			where portfolio.id=$1::uuid and portfolio.institution_id=$2 and source.institution_id=$2 order by source.evaluation_code`
	case "mobilitate":
		query = `select source.id::text, 'mobilitate', source.case_code || ' · ' || source.full_name || ' · ' || source.status
			from education_mobility_cases source join education_portfolios portfolio on portfolio.owner_personnel_id=source.personnel_id and portfolio.school_year=source.school_year
			where portfolio.id=$1::uuid and portfolio.institution_id=$2 and source.institution_id=$2 order by source.case_code`
	case "gradatie_merit":
		query = `select source.id::text, 'gradatie_merit', source.grant_code || ' · ' || source.full_name || ' · ' || source.status
			from education_merit_grants source join education_portfolios portfolio on portfolio.owner_personnel_id=source.personnel_id and portfolio.school_year=source.school_year
			where portfolio.id=$1::uuid and portfolio.institution_id=$2 and source.institution_id=$2 order by source.grant_code`
	default:
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_scope"})
		return
	}
	rows, err := s.pool.Query(r.Context(), query, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_sources_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioValorificationEligibleSource, 0)
	for rows.Next() {
		var item PortfolioValorificationEligibleSource
		if err := rows.Scan(&item.ID, &item.Scope, &item.Label); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_sources_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_sources_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) PortfolioValorificationEligibleArchiveVersions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	rows, err := s.pool.Query(r.Context(), `
		select version.document_id::text, version.id::text, version.version_no, coalesce(nullif(version.title,''), document.title)
		from archive_document_versions version
		join archive_documents document on document.id=version.document_id and document.institution_id=version.institution_id
		where version.institution_id=$2 and version.status='active'
			and btrim(version.source_bucket)<>'' and btrim(version.source_object_key)<>'' and lower(btrim(version.source_sha256)) ~ '^[0-9a-f]{64}$'
			and exists (select 1 from education_portfolios portfolio where portfolio.id=$1::uuid and portfolio.institution_id=$2)
		order by coalesce(nullif(version.title,''), document.title), version.version_no desc
		limit 100
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_archive_versions_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioValorificationEligibleArchiveVersion, 0)
	for rows.Next() {
		var item PortfolioValorificationEligibleArchiveVersion
		if err := rows.Scan(&item.ArchiveDocumentID, &item.ArchiveVersionID, &item.VersionNo, &item.Title); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_archive_versions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_archive_versions_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

// CreatePortfolioValorificationPackage deliberately accepts only source UUIDs,
// never a free-text "target reference". The migration trigger repeats and
// enforces every scope/source/institution invariant at the database boundary.
func (s *Service) CreatePortfolioValorificationPackage(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePortfolioValorificationPackageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_package_payload"})
		return
	}
	req.Scope, req.Purpose = strings.TrimSpace(req.Scope), strings.TrimSpace(req.Purpose)
	req.SourceEvaluationID, req.SourceMobilityCaseID, req.SourceMeritGrantID = strings.TrimSpace(req.SourceEvaluationID), strings.TrimSpace(req.SourceMobilityCaseID), strings.TrimSpace(req.SourceMeritGrantID)
	valid := (req.Scope == "evaluare_profesionala" && req.SourceEvaluationID != "" && req.SourceMobilityCaseID == "" && req.SourceMeritGrantID == "") ||
		(req.Scope == "mobilitate" && req.SourceEvaluationID == "" && req.SourceMobilityCaseID != "" && req.SourceMeritGrantID == "") ||
		(req.Scope == "gradatie_merit" && req.SourceEvaluationID == "" && req.SourceMobilityCaseID == "" && req.SourceMeritGrantID != "")
	if !valid || !portfolioValorificationPurposeAllowed(req.Scope, req.Purpose) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_package_source"})
		return
	}
	item, err := scanPortfolioValorificationPackage(s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_valorification_packages (tenant_code,institution_id,portfolio_id,scope,purpose,source_evaluation_id,source_mobility_case_id,source_merit_grant_id)
		select public.current_tenant_code(), $2, portfolio.id, $3, $4,
			nullif($5,'')::uuid, nullif($6,'')::uuid, nullif($7,'')::uuid
		from education_portfolios portfolio
		where portfolio.id=$1::uuid and portfolio.institution_id=$2 and portfolio.withdrawn_at is null
		returning `+portfolioValorificationPackageColumns, recordID, s.institutionID(r), req.Scope, req.Purpose, req.SourceEvaluationID, req.SourceMobilityCaseID, req.SourceMeritGrantID))
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_valorification_package_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.valorification_package.create", "portfolio_valorification_package", item.ID, "Scope-bound portfolio valorification package created.", map[string]any{"portfolio_id": item.PortfolioID, "scope": item.Scope, "purpose": item.Purpose})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) AdvancePortfolioValorificationPackage(w http.ResponseWriter, r *http.Request) {
	recordID, itemID := strings.TrimSpace(chi.URLParam(r, "recordID")), strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req AdvancePortfolioValorificationPackageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_package_advance_payload"})
		return
	}
	status := map[string]string{"submit": "submitted", "validate": "validated", "complete": "completed"}[strings.TrimSpace(req.Action)]
	if status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_package_action"})
		return
	}
	item, err := scanPortfolioValorificationPackage(s.pool.QueryRow(r.Context(), `
		update education_portfolio_valorification_packages package set status=$1
		where package.id=$2::uuid and package.portfolio_id=$3::uuid and package.institution_id=$4
		returning `+portfolioValorificationPackageColumns, status, itemID, recordID, s.institutionID(r)))
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_valorification_package_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_valorification_package_transition_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.valorification_package.advance", "portfolio_valorification_package", item.ID, "Portfolio valorification package lifecycle advanced.", map[string]any{"portfolio_id": recordID, "status": item.Status})
	httpx.JSON(w, http.StatusOK, item)
}

// AddPortfolioValorificationPackageDocument accepts only immutable archive
// identifiers. Placeholder metadata is overwritten by the database trigger
// from the referenced archive version, so neither browser nor handler owns the
// hash or storage provenance.
func (s *Service) AddPortfolioValorificationPackageDocument(w http.ResponseWriter, r *http.Request) {
	recordID, itemID := strings.TrimSpace(chi.URLParam(r, "recordID")), strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req AddPortfolioValorificationPackageDocumentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_valorification_package_document_payload"})
		return
	}
	req.ArchiveDocumentID, req.ArchiveVersionID = strings.TrimSpace(req.ArchiveDocumentID), strings.TrimSpace(req.ArchiveVersionID)
	if req.ArchiveDocumentID == "" || req.ArchiveVersionID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_valorification_package_document_source"})
		return
	}
	var documentID string
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_valorification_package_documents (
			package_id,institution_id,archive_document_id,archive_version_id,
			archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256
		)
		select package.id, $3, $4::uuid, $5::uuid, 1, '', '', repeat('0',64)
		from education_portfolio_valorification_packages package
		where package.id=$1::uuid and package.portfolio_id=$2::uuid and package.institution_id=$3
		returning id::text`, itemID, recordID, s.institutionID(r), req.ArchiveDocumentID, req.ArchiveVersionID).Scan(&documentID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_valorification_package_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_valorification_package_document_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.valorification_package.document.add", "portfolio_valorification_package_document", documentID, "Immutable archive version attached to valorification package.", map[string]any{"package_id": itemID})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": documentID})
}
