package gdpr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) logAudit(r *http.Request, action string, targetType string, targetID string, summary string, details map[string]any) {
	_ = audit.Log(r.Context(), s.pool, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Summary:      summary,
		Details:      details,
	})
}

func (s *Service) Config(w http.ResponseWriter, r *http.Request) {
	settings, err := s.loadSettings(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}

	response := ConfigResponse{Settings: settings}
	if response.Catalogs.Domains, err = s.loadNomenclatures(r.Context(), "gdpr_domain"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}
	if response.Catalogs.PolicyStatus, err = s.loadNomenclatures(r.Context(), "gdpr_policy_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}
	if response.Catalogs.RequestTypes, err = s.loadNomenclatures(r.Context(), "gdpr_request_type"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}
	if response.Catalogs.RequestStatus, err = s.loadNomenclatures(r.Context(), "gdpr_request_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}
	if response.Catalogs.SourceModules, err = s.loadNomenclatures(r.Context(), "gdpr_source_module"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_config_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Dashboard(w http.ResponseWriter, r *http.Request) {
	var response DashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) filter (where status = 'active') as active_policies,
			(select count(*) from gdpr_subject_requests where institution_id = 'inst-001' and status in ('received', 'identity_check', 'in_progress', 'waiting_approval')) as pending_requests,
			(select count(*) from gdpr_subject_requests where institution_id = 'inst-001' and due_on < current_date and status not in ('completed', 'rejected')) as overdue_requests,
			(select count(*) from gdpr_subject_requests where institution_id = 'inst-001' and anonymization_required = true) as anonymization_cases
		from gdpr_retention_policies
		where institution_id = 'inst-001'
	`).Scan(
		&response.Stats.ActivePolicies,
		&response.Stats.PendingRequests,
		&response.Stats.OverdueRequests,
		&response.Stats.AnonymizationCases,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) RetentionPolicies(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"policy_code":     {},
			"domain_code":     {},
			"record_category": {},
			"status":          {},
			"review_due_on":   {},
			"owner_name":      {},
		},
		[]string{"policy_code", "domain_code", "record_category", "status", "review_due_on"},
	)

	whereClause, args := buildRetentionFilters(query.Filters)

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from gdpr_retention_policies grp "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_failed"})
		return
	}

	sortField := retentionSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			grp.id::text,
			grp.policy_code,
			grp.domain_code,
			grp.record_category,
			grp.retention_years,
			grp.legal_basis,
			grp.status,
			to_char(grp.review_due_on, 'YYYY-MM-DD'),
			grp.owner_name,
			grp.institution_id,
			grp.notes
		from gdpr_retention_policies grp
		%s
		order by %s %s, grp.review_due_on asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_failed"})
		return
	}
	defer rows.Close()

	items := make([]RetentionPolicy, 0, query.PageSize)
	for rows.Next() {
		var item RetentionPolicy
		if err := rows.Scan(
			&item.ID,
			&item.PolicyCode,
			&item.DomainCode,
			&item.RecordCategory,
			&item.RetentionYears,
			&item.LegalBasis,
			&item.Status,
			&item.ReviewDueOn,
			&item.OwnerName,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RetentionPolicyFilters(w http.ResponseWriter, r *http.Request) {
	response := RetentionPolicyFiltersResponse{}
	var err error
	if response.Domains, err = s.loadNomenclatures(r.Context(), "gdpr_domain"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadNomenclatures(r.Context(), "gdpr_policy_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_retention_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	var req CreateRetentionPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_retention_policy_payload"})
		return
	}

	req.DomainCode = strings.TrimSpace(req.DomainCode)
	req.RecordCategory = strings.TrimSpace(req.RecordCategory)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.Status = strings.TrimSpace(req.Status)
	req.ReviewDueOn = strings.TrimSpace(req.ReviewDueOn)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.DomainCode == "" || req.RecordCategory == "" || req.LegalBasis == "" || req.Status == "" || req.ReviewDueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_retention_policy_fields"})
		return
	}
	if !s.isNomenclatureAllowed(r.Context(), "gdpr_domain", req.DomainCode) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_policy_status", req.Status) ||
		req.RetentionYears < 1 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_retention_policy_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReviewDueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_retention_review_due"})
		return
	}

	policyCode := fmt.Sprintf("RET-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item RetentionPolicy
	err := s.pool.QueryRow(r.Context(), `
		insert into gdpr_retention_policies (
			policy_code,
			domain_code,
			record_category,
			retention_years,
			legal_basis,
			status,
			review_due_on,
			owner_name,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning
			id::text,
			policy_code,
			domain_code,
			record_category,
			retention_years,
			legal_basis,
			status,
			to_char(review_due_on, 'YYYY-MM-DD'),
			owner_name,
			institution_id,
			notes
	`,
		policyCode,
		req.DomainCode,
		req.RecordCategory,
		req.RetentionYears,
		req.LegalBasis,
		req.Status,
		req.ReviewDueOn,
		req.OwnerName,
		"inst-001",
		req.Notes,
	).Scan(
		&item.ID,
		&item.PolicyCode,
		&item.DomainCode,
		&item.RecordCategory,
		&item.RetentionYears,
		&item.LegalBasis,
		&item.Status,
		&item.ReviewDueOn,
		&item.OwnerName,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "retention_policy_create_failed"})
		return
	}

	s.logAudit(r, "gdpr.retention_policies.create", "retention_policy", item.ID, "GDPR retention policy created.", map[string]any{
		"policy_code":     item.PolicyCode,
		"domain_code":     item.DomainCode,
		"record_category": item.RecordCategory,
		"retention_years": item.RetentionYears,
		"status":          item.Status,
		"review_due_on":   item.ReviewDueOn,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) SubjectRequests(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"request_code":           {},
			"subject_name":           {},
			"request_type":           {},
			"status":                 {},
			"submitted_on":           {},
			"due_on":                 {},
			"source_module":          {},
			"anonymization_required": {},
		},
		[]string{"request_code", "subject_name", "request_type", "status", "submitted_on", "due_on"},
	)

	whereClause, args := buildSubjectRequestFilters(query.Filters)

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from gdpr_subject_requests gsr "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_requests_failed"})
		return
	}

	sortField := subjectRequestSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			gsr.id::text,
			gsr.request_code,
			gsr.subject_name,
			gsr.request_type,
			gsr.status,
			to_char(gsr.submitted_on, 'YYYY-MM-DD'),
			to_char(gsr.due_on, 'YYYY-MM-DD'),
			gsr.handled_by,
			gsr.source_module,
			gsr.anonymization_required,
			gsr.institution_id,
			gsr.notes
		from gdpr_subject_requests gsr
		%s
		order by %s %s, gsr.due_on asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_requests_failed"})
		return
	}
	defer rows.Close()

	items := make([]SubjectRequest, 0, query.PageSize)
	for rows.Next() {
		var item SubjectRequest
		if err := rows.Scan(
			&item.ID,
			&item.RequestCode,
			&item.SubjectName,
			&item.RequestType,
			&item.Status,
			&item.SubmittedOn,
			&item.DueOn,
			&item.HandledBy,
			&item.SourceModule,
			&item.AnonymizationRequired,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_requests_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_requests_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) SubjectRequestFilters(w http.ResponseWriter, r *http.Request) {
	response := SubjectRequestFiltersResponse{}
	var err error
	if response.RequestTypes, err = s.loadNomenclatures(r.Context(), "gdpr_request_type"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_request_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadNomenclatures(r.Context(), "gdpr_request_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_request_filters_failed"})
		return
	}
	if response.SourceModules, err = s.loadNomenclatures(r.Context(), "gdpr_source_module"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_subject_request_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateSubjectRequest(w http.ResponseWriter, r *http.Request) {
	var req CreateSubjectRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_request_payload"})
		return
	}

	req.SubjectName = strings.TrimSpace(req.SubjectName)
	req.RequestType = strings.TrimSpace(req.RequestType)
	req.Status = strings.TrimSpace(req.Status)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.HandledBy = strings.TrimSpace(req.HandledBy)
	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.SubjectName == "" || req.RequestType == "" || req.Status == "" || req.SubmittedOn == "" || req.DueOn == "" || req.SourceModule == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_subject_request_fields"})
		return
	}
	if !s.isNomenclatureAllowed(r.Context(), "gdpr_request_type", req.RequestType) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_request_status", req.Status) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_source_module", req.SourceModule) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_request_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.SubmittedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_request_submitted_on"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_request_due_on"})
		return
	}

	requestCode := fmt.Sprintf("DSR-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item SubjectRequest
	err := s.pool.QueryRow(r.Context(), `
		insert into gdpr_subject_requests (
			request_code,
			subject_name,
			request_type,
			status,
			submitted_on,
			due_on,
			handled_by,
			source_module,
			anonymization_required,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning
			id::text,
			request_code,
			subject_name,
			request_type,
			status,
			to_char(submitted_on, 'YYYY-MM-DD'),
			to_char(due_on, 'YYYY-MM-DD'),
			handled_by,
			source_module,
			anonymization_required,
			institution_id,
			notes
	`,
		requestCode,
		req.SubjectName,
		req.RequestType,
		req.Status,
		req.SubmittedOn,
		req.DueOn,
		req.HandledBy,
		req.SourceModule,
		req.AnonymizationRequired,
		"inst-001",
		req.Notes,
	).Scan(
		&item.ID,
		&item.RequestCode,
		&item.SubjectName,
		&item.RequestType,
		&item.Status,
		&item.SubmittedOn,
		&item.DueOn,
		&item.HandledBy,
		&item.SourceModule,
		&item.AnonymizationRequired,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "subject_request_create_failed"})
		return
	}

	s.logAudit(r, "gdpr.subject_requests.create", "subject_request", item.ID, "GDPR subject request created.", map[string]any{
		"request_code":           item.RequestCode,
		"subject_name":           item.SubjectName,
		"request_type":           item.RequestType,
		"status":                 item.Status,
		"submitted_on":           item.SubmittedOn,
		"due_on":                 item.DueOn,
		"source_module":          item.SourceModule,
		"anonymization_required": item.AnonymizationRequired,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ExportDashboard(w http.ResponseWriter, r *http.Request) {
	var response ExportDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_exports,
			count(*) filter (where status = 'pending_approval') as pending_approval,
			count(*) filter (where status = 'generated') as generated_exports,
			count(*) filter (where status = 'delivered') as delivered_exports
		from gdpr_subject_exports
		where institution_id = 'inst-001'
	`).Scan(
		&response.Stats.TotalExports,
		&response.Stats.PendingApproval,
		&response.Stats.GeneratedExports,
		&response.Stats.DeliveredExports,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_exports_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) SubjectExports(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"export_code":   {},
			"subject_name":  {},
			"source_module": {},
			"status":        {},
			"export_format": {},
			"approved_on":   {},
			"generated_on":  {},
		},
		[]string{"export_code", "subject_name", "source_module", "status", "export_format", "approved_on", "generated_on"},
	)

	whereClause, args := buildSubjectExportFilters(query.Filters)

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from gdpr_subject_exports gse "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_exports_failed"})
		return
	}

	sortField := subjectExportSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			gse.id::text,
			gse.export_code,
			coalesce(gse.request_id::text, ''),
			gse.subject_name,
			gse.source_module,
			gse.status,
			gse.export_format,
			gse.approved_by,
			coalesce(to_char(gse.approved_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(gse.generated_on, 'YYYY-MM-DD'), ''),
			gse.package_summary,
			gse.institution_id,
			gse.notes
		from gdpr_subject_exports gse
		%s
		order by %s %s, gse.created_at desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_exports_failed"})
		return
	}
	defer rows.Close()

	items := make([]SubjectExport, 0, query.PageSize)
	for rows.Next() {
		var item SubjectExport
		if err := rows.Scan(
			&item.ID,
			&item.ExportCode,
			&item.RequestID,
			&item.SubjectName,
			&item.SourceModule,
			&item.Status,
			&item.ExportFormat,
			&item.ApprovedBy,
			&item.ApprovedOn,
			&item.GeneratedOn,
			&item.PackageSummary,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_exports_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_exports_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) SubjectExportFilters(w http.ResponseWriter, r *http.Request) {
	response := SubjectExportFiltersResponse{}
	var err error
	if response.Statuses, err = s.loadNomenclatures(r.Context(), "gdpr_export_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_export_filters_failed"})
		return
	}
	if response.ExportFormats, err = s.loadNomenclatures(r.Context(), "gdpr_export_format"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_export_filters_failed"})
		return
	}
	if response.SourceModules, err = s.loadNomenclatures(r.Context(), "gdpr_source_module"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_export_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateSubjectExport(w http.ResponseWriter, r *http.Request) {
	var req CreateSubjectExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_export_payload"})
		return
	}

	req.RequestID = strings.TrimSpace(req.RequestID)
	req.SubjectName = strings.TrimSpace(req.SubjectName)
	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.Status = strings.TrimSpace(req.Status)
	req.ExportFormat = strings.TrimSpace(req.ExportFormat)
	req.ApprovedBy = strings.TrimSpace(req.ApprovedBy)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.GeneratedOn = strings.TrimSpace(req.GeneratedOn)
	req.PackageSummary = strings.TrimSpace(req.PackageSummary)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.SubjectName == "" || req.SourceModule == "" || req.Status == "" || req.ExportFormat == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_subject_export_fields"})
		return
	}
	if !s.isNomenclatureAllowed(r.Context(), "gdpr_source_module", req.SourceModule) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_export_status", req.Status) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_export_format", req.ExportFormat) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_export_fields"})
		return
	}
	if req.ApprovedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ApprovedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_export_approved_on"})
			return
		}
	}
	if req.GeneratedOn != "" {
		if _, err := time.Parse("2006-01-02", req.GeneratedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_subject_export_generated_on"})
			return
		}
	}

	exportCode := fmt.Sprintf("EXP-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item SubjectExport
	err := s.pool.QueryRow(r.Context(), `
		insert into gdpr_subject_exports (
			export_code,
			request_id,
			subject_name,
			source_module,
			status,
			export_format,
			approved_by,
			approved_on,
			generated_on,
			package_summary,
			institution_id,
			notes
		) values ($1,nullif($2, '')::uuid,$3,$4,$5,$6,$7,nullif($8, '')::date,nullif($9, '')::date,$10,$11,$12)
		returning
			id::text,
			export_code,
			coalesce(request_id::text, ''),
			subject_name,
			source_module,
			status,
			export_format,
			approved_by,
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(generated_on, 'YYYY-MM-DD'), ''),
			package_summary,
			institution_id,
			notes
	`,
		exportCode,
		req.RequestID,
		req.SubjectName,
		req.SourceModule,
		req.Status,
		req.ExportFormat,
		req.ApprovedBy,
		req.ApprovedOn,
		req.GeneratedOn,
		req.PackageSummary,
		"inst-001",
		req.Notes,
	).Scan(
		&item.ID,
		&item.ExportCode,
		&item.RequestID,
		&item.SubjectName,
		&item.SourceModule,
		&item.Status,
		&item.ExportFormat,
		&item.ApprovedBy,
		&item.ApprovedOn,
		&item.GeneratedOn,
		&item.PackageSummary,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "subject_export_create_failed"})
		return
	}

	s.logAudit(r, "gdpr.subject_exports.create", "subject_export", item.ID, "GDPR subject export created.", map[string]any{
		"export_code":   item.ExportCode,
		"subject_name":  item.SubjectName,
		"source_module": item.SourceModule,
		"status":        item.Status,
		"export_format": item.ExportFormat,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) PublicationDashboard(w http.ResponseWriter, r *http.Request) {
	var response PublicationDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_reviews,
			count(*) filter (where anonymization_status = 'pending') as pending_anonymization,
			count(*) filter (where publication_status = 'ready') as ready_for_publication,
			count(*) filter (where publication_status = 'published') as published_items
		from gdpr_publication_reviews
		where institution_id = 'inst-001'
	`).Scan(
		&response.Stats.TotalReviews,
		&response.Stats.PendingAnonymization,
		&response.Stats.ReadyForPublication,
		&response.Stats.PublishedItems,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) PublicationReviews(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"review_code":          {},
			"source_module":        {},
			"source_label":         {},
			"anonymization_status": {},
			"publication_status":   {},
			"reviewed_on":          {},
		},
		[]string{"review_code", "source_module", "source_label", "anonymization_status", "publication_status", "reviewed_on"},
	)

	whereClause, args := buildPublicationReviewFilters(query.Filters)

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from gdpr_publication_reviews gpr "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_reviews_failed"})
		return
	}

	sortField := publicationReviewSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			gpr.id::text,
			gpr.review_code,
			gpr.source_module,
			gpr.source_record_id,
			gpr.source_label,
			gpr.anonymization_status,
			gpr.publication_status,
			gpr.reviewed_by,
			coalesce(to_char(gpr.reviewed_on, 'YYYY-MM-DD'), ''),
			gpr.legal_basis,
			gpr.institution_id,
			gpr.notes
		from gdpr_publication_reviews gpr
		%s
		order by %s %s, gpr.created_at desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_reviews_failed"})
		return
	}
	defer rows.Close()

	items := make([]PublicationReview, 0, query.PageSize)
	for rows.Next() {
		var item PublicationReview
		if err := rows.Scan(
			&item.ID,
			&item.ReviewCode,
			&item.SourceModule,
			&item.SourceRecordID,
			&item.SourceLabel,
			&item.AnonymizationStatus,
			&item.PublicationStatus,
			&item.ReviewedBy,
			&item.ReviewedOn,
			&item.LegalBasis,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_reviews_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_reviews_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PublicationReviewFilters(w http.ResponseWriter, r *http.Request) {
	response := PublicationReviewFiltersResponse{}
	var err error
	if response.SourceModules, err = s.loadNomenclatures(r.Context(), "gdpr_publication_source_module"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_filters_failed"})
		return
	}
	if response.AnonymizationStatuses, err = s.loadNomenclatures(r.Context(), "gdpr_anonymization_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_filters_failed"})
		return
	}
	if response.PublicationStatuses, err = s.loadNomenclatures(r.Context(), "gdpr_publication_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "gdpr_publication_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreatePublicationReview(w http.ResponseWriter, r *http.Request) {
	var req CreatePublicationReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_review_payload"})
		return
	}

	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.SourceRecordID = strings.TrimSpace(req.SourceRecordID)
	req.SourceLabel = strings.TrimSpace(req.SourceLabel)
	req.AnonymizationStatus = strings.TrimSpace(req.AnonymizationStatus)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.ReviewedBy = strings.TrimSpace(req.ReviewedBy)
	req.ReviewedOn = strings.TrimSpace(req.ReviewedOn)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.SourceModule == "" || req.SourceRecordID == "" || req.SourceLabel == "" || req.AnonymizationStatus == "" || req.PublicationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_publication_review_fields"})
		return
	}
	if !s.isNomenclatureAllowed(r.Context(), "gdpr_publication_source_module", req.SourceModule) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_anonymization_status", req.AnonymizationStatus) ||
		!s.isNomenclatureAllowed(r.Context(), "gdpr_publication_status", req.PublicationStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_review_fields"})
		return
	}
	if req.ReviewedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ReviewedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_publication_reviewed_on"})
			return
		}
	}

	reviewCode := fmt.Sprintf("PUB-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item PublicationReview
	err := s.pool.QueryRow(r.Context(), `
		insert into gdpr_publication_reviews (
			review_code,
			source_module,
			source_record_id,
			source_label,
			anonymization_status,
			publication_status,
			reviewed_by,
			reviewed_on,
			legal_basis,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,nullif($8, '')::date,$9,$10,$11)
		returning
			id::text,
			review_code,
			source_module,
			source_record_id,
			source_label,
			anonymization_status,
			publication_status,
			reviewed_by,
			coalesce(to_char(reviewed_on, 'YYYY-MM-DD'), ''),
			legal_basis,
			institution_id,
			notes
	`,
		reviewCode,
		req.SourceModule,
		req.SourceRecordID,
		req.SourceLabel,
		req.AnonymizationStatus,
		req.PublicationStatus,
		req.ReviewedBy,
		req.ReviewedOn,
		req.LegalBasis,
		"inst-001",
		req.Notes,
	).Scan(
		&item.ID,
		&item.ReviewCode,
		&item.SourceModule,
		&item.SourceRecordID,
		&item.SourceLabel,
		&item.AnonymizationStatus,
		&item.PublicationStatus,
		&item.ReviewedBy,
		&item.ReviewedOn,
		&item.LegalBasis,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "publication_review_create_failed"})
		return
	}

	s.logAudit(r, "gdpr.publication_reviews.create", "publication_review", item.ID, "GDPR publication review created.", map[string]any{
		"review_code":          item.ReviewCode,
		"source_module":        item.SourceModule,
		"source_record_id":     item.SourceRecordID,
		"anonymization_status": item.AnonymizationStatus,
		"publication_status":   item.PublicationStatus,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func buildRetentionFilters(filters map[string]string) (string, []any) {
	clauses := []string{"grp.institution_id = 'inst-001'"}
	args := []any{}
	addContains := func(column, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["policy_code"]; value != "" {
		addContains("grp.policy_code", value)
	}
	if value := filters["domain_code"]; value != "" {
		addEqual("grp.domain_code", value)
	}
	if value := filters["record_category"]; value != "" {
		addContains("grp.record_category", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("grp.status", value)
	}
	if value := filters["review_due_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("grp.review_due_on = $%d::date", len(args)))
	}
	return "where " + strings.Join(clauses, " and "), args
}

func retentionSortColumn(field string) string {
	switch field {
	case "policy_code":
		return "grp.policy_code"
	case "domain_code":
		return "grp.domain_code"
	case "record_category":
		return "grp.record_category"
	case "retention_years":
		return "grp.retention_years"
	case "status":
		return "grp.status"
	case "review_due_on":
		return "grp.review_due_on"
	default:
		return "grp.review_due_on"
	}
}

func buildSubjectRequestFilters(filters map[string]string) (string, []any) {
	clauses := []string{"gsr.institution_id = 'inst-001'"}
	args := []any{}
	addContains := func(column, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["request_code"]; value != "" {
		addContains("gsr.request_code", value)
	}
	if value := filters["subject_name"]; value != "" {
		addContains("gsr.subject_name", value)
	}
	if value := filters["request_type"]; value != "" {
		addEqual("gsr.request_type", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("gsr.status", value)
	}
	if value := filters["submitted_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gsr.submitted_on = $%d::date", len(args)))
	}
	if value := filters["due_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gsr.due_on = $%d::date", len(args)))
	}
	if value := filters["source_module"]; value != "" {
		addEqual("gsr.source_module", value)
	}
	if value := filters["anonymization_required"]; value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("gsr.anonymization_required = $%d", len(args)))
	}
	return "where " + strings.Join(clauses, " and "), args
}

func subjectRequestSortColumn(field string) string {
	switch field {
	case "request_code":
		return "gsr.request_code"
	case "subject_name":
		return "gsr.subject_name"
	case "request_type":
		return "gsr.request_type"
	case "status":
		return "gsr.status"
	case "submitted_on":
		return "gsr.submitted_on"
	case "due_on":
		return "gsr.due_on"
	default:
		return "gsr.due_on"
	}
}

func buildSubjectExportFilters(filters map[string]string) (string, []any) {
	clauses := []string{"gse.institution_id = 'inst-001'"}
	args := []any{}
	addContains := func(column, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["export_code"]; value != "" {
		addContains("gse.export_code", value)
	}
	if value := filters["subject_name"]; value != "" {
		addContains("gse.subject_name", value)
	}
	if value := filters["source_module"]; value != "" {
		addEqual("gse.source_module", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("gse.status", value)
	}
	if value := filters["export_format"]; value != "" {
		addEqual("gse.export_format", value)
	}
	if value := filters["approved_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gse.approved_on = $%d::date", len(args)))
	}
	if value := filters["generated_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gse.generated_on = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func subjectExportSortColumn(field string) string {
	switch field {
	case "export_code":
		return "gse.export_code"
	case "subject_name":
		return "gse.subject_name"
	case "source_module":
		return "gse.source_module"
	case "status":
		return "gse.status"
	case "export_format":
		return "gse.export_format"
	case "approved_on":
		return "gse.approved_on"
	case "generated_on":
		return "gse.generated_on"
	default:
		return "gse.created_at"
	}
}

func buildPublicationReviewFilters(filters map[string]string) (string, []any) {
	clauses := []string{"gpr.institution_id = 'inst-001'"}
	args := []any{}
	addContains := func(column, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["review_code"]; value != "" {
		addContains("gpr.review_code", value)
	}
	if value := filters["source_module"]; value != "" {
		addEqual("gpr.source_module", value)
	}
	if value := filters["source_label"]; value != "" {
		addContains("gpr.source_label", value)
	}
	if value := filters["anonymization_status"]; value != "" {
		addEqual("gpr.anonymization_status", value)
	}
	if value := filters["publication_status"]; value != "" {
		addEqual("gpr.publication_status", value)
	}
	if value := filters["reviewed_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gpr.reviewed_on = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func publicationReviewSortColumn(field string) string {
	switch field {
	case "review_code":
		return "gpr.review_code"
	case "source_module":
		return "gpr.source_module"
	case "source_label":
		return "gpr.source_label"
	case "anonymization_status":
		return "gpr.anonymization_status"
	case "publication_status":
		return "gpr.publication_status"
	case "reviewed_on":
		return "gpr.reviewed_on"
	default:
		return "gpr.created_at"
	}
}

func (s *Service) loadSettings(ctx context.Context) (Settings, error) {
	settings := Settings{
		DefaultResponseSLADays:    30,
		RetentionReviewNoticeDays: 90,
	}

	rows, err := s.pool.Query(ctx, `
		select code, coalesce(value_bool, false), coalesce(value_int, 0)
		from gdpr_operational_settings
	`)
	if err != nil {
		return Settings{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var code string
		var boolValue bool
		var intValue int
		if err := rows.Scan(&code, &boolValue, &intValue); err != nil {
			return Settings{}, err
		}
		switch code {
		case "publication_anonymization_required":
			settings.PublicationAnonymizationRequired = boolValue
		case "subject_export_requires_approval":
			settings.SubjectExportRequiresApproval = boolValue
		case "default_response_sla_days":
			settings.DefaultResponseSLADays = intValue
		case "retention_review_notice_days":
			settings.RetentionReviewNoticeDays = intValue
		case "portfolio_consent_required":
			settings.PortfolioConsentRequired = boolValue
		case "portfolio_authenticity_required":
			settings.PortfolioAuthenticityRequired = boolValue
		}
	}
	return settings, rows.Err()
}

func (s *Service) loadNomenclatures(ctx context.Context, domain string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select code
		from app_nomenclatures
		where domain = $1 and active = true
		order by sort_order, code
	`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Service) isNomenclatureAllowed(ctx context.Context, domain string, code string) bool {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		select exists(
			select 1
			from app_nomenclatures
			where domain = $1 and code = $2 and active = true
		)
	`, domain, code).Scan(&exists); err != nil {
		return false
	}
	return exists
}
