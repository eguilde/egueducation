package education

// Institution-owned, versioned procedures for the professional portfolio.
// The database migration is intentionally the final lifecycle authority: these
// handlers validate the public contract and never try to bypass its immutable
// evidence rules.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

type PortfolioProcedure struct {
	ID                string         `json:"id"`
	ProcedureCode     string         `json:"procedure_code"`
	VersionNo         int            `json:"version_no"`
	Title             string         `json:"title"`
	SourceRef         string         `json:"source_ref"`
	LifecycleStatus   string         `json:"lifecycle_status"`
	EffectiveFrom     string         `json:"effective_from,omitempty"`
	EffectiveTo       string         `json:"effective_to,omitempty"`
	CalendarRules     map[string]any `json:"calendar_rules"`
	AccessRules       map[string]any `json:"access_rules"`
	AcceptedFormats   map[string]any `json:"accepted_formats"`
	RetentionRules    map[string]any `json:"retention_rules"`
	TransferRules     map[string]any `json:"transfer_rules"`
	ApprovedAt        string         `json:"approved_at,omitempty"`
	ApprovedByUserID  string         `json:"approved_by_user_id,omitempty"`
	PublishedAt       string         `json:"published_at,omitempty"`
	PublishedByUserID string         `json:"published_by_user_id,omitempty"`
	CreatedByUserID   string         `json:"created_by_user_id"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	SupersededAt      string         `json:"superseded_at,omitempty"`
	WithdrawnAt       string         `json:"withdrawn_at,omitempty"`
	InstitutionID     string         `json:"institution_id"`
}

type PortfolioProcedureSectionRule struct {
	ID                   string `json:"id"`
	ProcedureID          string `json:"procedure_id"`
	SectionCode          string `json:"section_code"`
	LabelRO              string `json:"label_ro"`
	LabelEN              string `json:"label_en"`
	SourceCatalogVersion string `json:"source_catalog_version"`
	Required             bool   `json:"required"`
	SortOrder            int    `json:"sort_order"`
	Active               bool   `json:"active"`
}

type CreatePortfolioProcedureRequest struct {
	ProcedureCode   string         `json:"procedure_code"`
	Title           string         `json:"title"`
	SourceRef       string         `json:"source_ref"`
	EffectiveFrom   string         `json:"effective_from,omitempty"`
	EffectiveTo     string         `json:"effective_to,omitempty"`
	CalendarRules   map[string]any `json:"calendar_rules,omitempty"`
	AccessRules     map[string]any `json:"access_rules,omitempty"`
	AcceptedFormats map[string]any `json:"accepted_formats,omitempty"`
	RetentionRules  map[string]any `json:"retention_rules,omitempty"`
	TransferRules   map[string]any `json:"transfer_rules,omitempty"`
}

type UpdatePortfolioProcedureRequest struct {
	CreatePortfolioProcedureRequest
	ExpectedUpdatedAt string `json:"expected_updated_at"`
}

type PortfolioProcedureEvidenceRequest struct {
	ExpectedUpdatedAt string         `json:"expected_updated_at"`
	Evidence          map[string]any `json:"evidence"`
}

type ReplacePortfolioProcedureSectionRulesRequest struct {
	ExpectedUpdatedAt string                          `json:"expected_updated_at"`
	Rules             []PortfolioProcedureSectionRule `json:"rules"`
}

func (s *Service) PortfolioProcedures(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"procedure_code": {}, "version_no": {}, "title": {}, "lifecycle_status": {}, "effective_from": {}, "updated_at": {},
	}, []string{"procedure_code", "title", "lifecycle_status"})
	if query.Sort == "" {
		query.Sort = "updated_at"
		query.Direction = "desc"
	}
	where, args := s.portfolioProcedureWhere(r, query)
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_procedure_versions "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_procedure_list_failed"})
		return
	}
	order := map[string]string{"procedure_code": "procedure_code", "version_no": "version_no", "title": "title", "lifecycle_status": "lifecycle_status", "effective_from": "effective_from", "updated_at": "updated_at"}[query.Sort]
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select `+portfolioProcedureColumns+` from education_portfolio_procedure_versions `+where+
		fmt.Sprintf(" order by %s %s, procedure_code, version_no desc limit $%d offset $%d", order, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_procedure_list_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioProcedure, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioProcedure
		if err := scanPortfolioProcedure(rows, &item); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioProcedureDetail(w http.ResponseWriter, r *http.Request) {
	item, err := s.loadPortfolioProcedure(r, chi.URLParam(r, "procedureID"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "portfolio_procedure_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreatePortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	var req CreatePortfolioProcedureRequest
	if !decodePortfolioProcedureRequest(w, r, &req) {
		return
	}
	actorID, err := s.currentActorUserID(r, strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r)))
	if err != nil || actorID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_procedure_actor_required"})
		return
	}
	tenantCode, err := s.portfolioProcedureTenantCode(r)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_tenant_failed"})
		return
	}
	var item PortfolioProcedure
	err = s.pool.QueryRow(r.Context(), `
		insert into education_portfolio_procedure_versions (
			institution_id, tenant_code, procedure_code, version_no, title, source_ref, effective_from, effective_to,
			calendar_rules, access_rules, accepted_formats, retention_rules, transfer_rules, created_by_user_id
		) select $1, $2, $3, coalesce(max(version_no), 0) + 1, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::uuid
		from education_portfolio_procedure_versions where institution_id = $1 and procedure_code = $3
		returning `+portfolioProcedureColumns,
		s.institutionID(r), tenantCode, req.ProcedureCode, req.Title, req.SourceRef, optionalProcedureDate(req.EffectiveFrom), optionalProcedureDate(req.EffectiveTo), jsonObject(req.CalendarRules), jsonObject(req.AccessRules), jsonObject(req.AcceptedFormats), jsonObject(req.RetentionRules), jsonObject(req.TransferRules), actorID,
	).Scan(portfolioProcedureScanTargets(&item)...)
	if err != nil {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_procedure_create_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.procedures.create", "portfolio_procedure", item.ID, "Portfolio procedure draft created.", map[string]any{"procedure_code": item.ProcedureCode, "version_no": item.VersionNo})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdatePortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	var req UpdatePortfolioProcedureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_payload"})
		return
	}
	if !validatePortfolioProcedureRequest(w, &req.CreatePortfolioProcedureRequest) {
		return
	}
	if !validExpectedUpdatedAt(req.ExpectedUpdatedAt) {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_expected_updated_at_required"})
		return
	}
	procedureID := strings.TrimSpace(chi.URLParam(r, "procedureID"))
	var item PortfolioProcedure
	err := s.pool.QueryRow(r.Context(), `update education_portfolio_procedure_versions set title=$1, source_ref=$2, effective_from=$3, effective_to=$4, calendar_rules=$5::jsonb, access_rules=$6::jsonb, accepted_formats=$7::jsonb, retention_rules=$8::jsonb, transfer_rules=$9::jsonb, updated_at=now()
		where id=$10::uuid and institution_id=$11 and lifecycle_status='draft' and updated_at=$12::timestamptz returning `+portfolioProcedureColumns,
		req.Title, req.SourceRef, optionalProcedureDate(req.EffectiveFrom), optionalProcedureDate(req.EffectiveTo), jsonObject(req.CalendarRules), jsonObject(req.AccessRules), jsonObject(req.AcceptedFormats), jsonObject(req.RetentionRules), jsonObject(req.TransferRules), procedureID, s.institutionID(r), req.ExpectedUpdatedAt).Scan(portfolioProcedureScanTargets(&item)...)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_procedure_stale_or_not_draft"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_update_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.procedures.update", "portfolio_procedure", item.ID, "Portfolio procedure draft updated.", map[string]any{"expected_updated_at": req.ExpectedUpdatedAt})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) ApprovePortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	s.transitionPortfolioProcedure(w, r, "approved")
}
func (s *Service) PublishPortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	s.transitionPortfolioProcedure(w, r, "published")
}
func (s *Service) SupersedePortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	s.transitionPortfolioProcedure(w, r, "superseded")
}
func (s *Service) WithdrawPortfolioProcedure(w http.ResponseWriter, r *http.Request) {
	s.transitionPortfolioProcedure(w, r, "withdrawn")
}

func (s *Service) transitionPortfolioProcedure(w http.ResponseWriter, r *http.Request, target string) {
	var req PortfolioProcedureEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_evidence_payload"})
		return
	}
	if !validExpectedUpdatedAt(req.ExpectedUpdatedAt) {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_expected_updated_at_required"})
		return
	}
	if len(req.Evidence) == 0 {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_evidence_required"})
		return
	}
	actorID, err := s.currentActorUserID(r, strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r)))
	if err != nil || actorID == "" {
		httpx.JSON(w, 403, map[string]any{"code": "portfolio_procedure_actor_required"})
		return
	}
	procedureID := strings.TrimSpace(chi.URLParam(r, "procedureID"))
	if target == "published" {
		var missingMandatorySections int
		if err := s.pool.QueryRow(r.Context(), `
			select count(*) from education_portfolio_sections catalog
			where catalog.active and catalog.required
				and not exists (
					select 1 from education_portfolio_procedure_section_rules rule
					where rule.procedure_id=$1::uuid and rule.institution_id=$2
						and rule.section_code=catalog.section_code and rule.active and rule.required
				)
		`, procedureID, s.institutionID(r)).Scan(&missingMandatorySections); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_readiness_failed"})
			return
		}
		if missingMandatorySections > 0 {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "portfolio_procedure_mandatory_sections_missing", "missing_count": missingMandatorySections})
			return
		}
	}
	allowed := map[string]string{"approved": "draft", "published": "approved", "superseded": "published", "withdrawn": "draft,approved,published"}[target]
	var item PortfolioProcedure
	var statement string
	args := []any{procedureID, s.institutionID(r), req.ExpectedUpdatedAt}
	switch target {
	case "approved":
		args = append(args, actorID, jsonObject(req.Evidence))
		statement = `update education_portfolio_procedure_versions set lifecycle_status='approved', approval_evidence=$5::jsonb, approved_at=now(), approved_by_user_id=$4::uuid, updated_at=now() where id=$1::uuid and institution_id=$2 and updated_at=$3::timestamptz and lifecycle_status='draft' returning ` + portfolioProcedureColumns
	case "published":
		args = append(args, actorID, jsonObject(req.Evidence))
		statement = `update education_portfolio_procedure_versions set lifecycle_status='published', publication_evidence=$5::jsonb, published_at=now(), published_by_user_id=$4::uuid, updated_at=now() where id=$1::uuid and institution_id=$2 and updated_at=$3::timestamptz and lifecycle_status='approved' returning ` + portfolioProcedureColumns
	case "superseded":
		statement = `update education_portfolio_procedure_versions set lifecycle_status='superseded', superseded_at=now(), updated_at=now() where id=$1::uuid and institution_id=$2 and updated_at=$3::timestamptz and lifecycle_status='published' returning ` + portfolioProcedureColumns
	case "withdrawn":
		statement = `update education_portfolio_procedure_versions set lifecycle_status='withdrawn', withdrawn_at=now(), updated_at=now() where id=$1::uuid and institution_id=$2 and updated_at=$3::timestamptz and lifecycle_status in ('draft','approved','published') returning ` + portfolioProcedureColumns
	}
	_ = allowed // Documents the allowed state set above; the SQL predicates are the enforcement boundary.
	err = s.pool.QueryRow(r.Context(), statement, args...).Scan(portfolioProcedureScanTargets(&item)...)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_procedure_invalid_transition_or_stale"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_transition_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.procedures."+target, "portfolio_procedure", item.ID, "Portfolio procedure transitioned to "+target+".", map[string]any{"evidence": req.Evidence, "previous_status": allowed})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioProcedureSectionRules(w http.ResponseWriter, r *http.Request) {
	procedureID := strings.TrimSpace(chi.URLParam(r, "procedureID"))
	if _, err := s.loadPortfolioProcedure(r, procedureID); errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "portfolio_procedure_not_found")
		return
	} else if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_detail_failed"})
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"section_code": {}, "label_ro": {}, "sort_order": {}}, []string{"section_code", "label_ro", "sort_order"})
	if query.Sort == "" {
		query.Sort = "sort_order"
	}
	where := "where procedure_id=$1::uuid and institution_id=$2"
	args := []any{procedureID, s.institutionID(r)}
	for _, key := range []string{"section_code", "label_ro"} {
		if value := strings.TrimSpace(query.Filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where += fmt.Sprintf(" and lower(%s) like $%d", key, len(args))
		}
	}
	if value := strings.TrimSpace(query.Filters["sort_order"]); value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and sort_order::text = $%d", len(args))
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_procedure_section_rules "+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_failed"})
		return
	}
	order := map[string]string{"section_code": "section_code", "label_ro": "label_ro", "sort_order": "sort_order"}[query.Sort]
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `select id::text,procedure_id::text,section_code,label_ro,label_en,source_catalog_version,required,sort_order,active from education_portfolio_procedure_section_rules `+where+fmt.Sprintf(" order by %s %s, section_code limit $%d offset $%d", order, strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_failed"})
		return
	}
	defer rows.Close()
	items := []PortfolioProcedureSectionRule{}
	for rows.Next() {
		var item PortfolioProcedureSectionRule
		if err := rows.Scan(&item.ID, &item.ProcedureID, &item.SectionCode, &item.LabelRO, &item.LabelEN, &item.SourceCatalogVersion, &item.Required, &item.SortOrder, &item.Active); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) ReplacePortfolioProcedureSectionRules(w http.ResponseWriter, r *http.Request) {
	procedureID := strings.TrimSpace(chi.URLParam(r, "procedureID"))
	var req ReplacePortfolioProcedureSectionRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_section_rules_payload"})
		return
	}
	if !validExpectedUpdatedAt(req.ExpectedUpdatedAt) {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_expected_updated_at_required"})
		return
	}
	if len(req.Rules) == 0 {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_section_rules_required"})
		return
	}
	seen := map[string]bool{}
	for index := range req.Rules {
		rule := &req.Rules[index]
		rule.SectionCode = strings.TrimSpace(rule.SectionCode)
		rule.LabelRO = strings.TrimSpace(rule.LabelRO)
		rule.LabelEN = strings.TrimSpace(rule.LabelEN)
		rule.SourceCatalogVersion = strings.TrimSpace(rule.SourceCatalogVersion)
		if rule.SectionCode == "" || rule.LabelRO == "" || rule.SourceCatalogVersion == "" || rule.SortOrder < 1 || seen[rule.SectionCode] {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_section_rule"})
			return
		}
		seen[rule.SectionCode] = true
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_transaction_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	var exists bool
	err = tx.QueryRow(r.Context(), `select exists(select 1 from education_portfolio_procedure_versions where id=$1::uuid and institution_id=$2 and lifecycle_status='draft' and updated_at=$3::timestamptz for update)`, procedureID, s.institutionID(r), req.ExpectedUpdatedAt).Scan(&exists)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_failed"})
		return
	}
	if !exists {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_procedure_stale_or_not_draft"})
		return
	}
	if _, err = tx.Exec(r.Context(), `delete from education_portfolio_procedure_section_rules where procedure_id=$1::uuid and institution_id=$2`, procedureID, s.institutionID(r)); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_replace_failed"})
		return
	}
	tenantCode, err := s.portfolioProcedureTenantCode(r)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_tenant_failed"})
		return
	}
	for _, rule := range req.Rules {
		if _, err = tx.Exec(r.Context(), `insert into education_portfolio_procedure_section_rules (procedure_id,institution_id,tenant_code,section_code,label_ro,label_en,source_catalog_version,required,sort_order,active) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, procedureID, s.institutionID(r), tenantCode, rule.SectionCode, rule.LabelRO, rule.LabelEN, rule.SourceCatalogVersion, rule.Required, rule.SortOrder, rule.Active); err != nil {
			httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_section_rules_replace_failed"})
			return
		}
	}
	if _, err = tx.Exec(r.Context(), `update education_portfolio_procedure_versions set updated_at=now() where id=$1::uuid and institution_id=$2`, procedureID, s.institutionID(r)); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_replace_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_procedure_section_rules_replace_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.procedures.section_rules.replace", "portfolio_procedure", procedureID, "Portfolio procedure section rules replaced.", map[string]any{"rule_count": len(req.Rules)})
	httpx.JSON(w, http.StatusOK, map[string]any{"procedure_id": procedureID, "rule_count": len(req.Rules)})
}

const portfolioProcedureColumns = `id::text, procedure_code, version_no, title, source_ref, lifecycle_status, coalesce(to_char(effective_from,'YYYY-MM-DD'),''), coalesce(to_char(effective_to,'YYYY-MM-DD'),''), calendar_rules, access_rules, accepted_formats, retention_rules, transfer_rules, coalesce(to_char(approved_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''), coalesce(approved_by_user_id::text,''), coalesce(to_char(published_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''), coalesce(published_by_user_id::text,''), created_by_user_id::text, to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), coalesce(to_char(superseded_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''), coalesce(to_char(withdrawn_at,'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''), institution_id`

func portfolioProcedureScanTargets(item *PortfolioProcedure) []any {
	return []any{&item.ID, &item.ProcedureCode, &item.VersionNo, &item.Title, &item.SourceRef, &item.LifecycleStatus, &item.EffectiveFrom, &item.EffectiveTo, &item.CalendarRules, &item.AccessRules, &item.AcceptedFormats, &item.RetentionRules, &item.TransferRules, &item.ApprovedAt, &item.ApprovedByUserID, &item.PublishedAt, &item.PublishedByUserID, &item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt, &item.SupersededAt, &item.WithdrawnAt, &item.InstitutionID}
}
func scanPortfolioProcedure(row pgx.Row, item *PortfolioProcedure) error {
	return row.Scan(portfolioProcedureScanTargets(item)...)
}
func (s *Service) loadPortfolioProcedure(r *http.Request, id string) (PortfolioProcedure, error) {
	var item PortfolioProcedure
	err := scanPortfolioProcedure(s.pool.QueryRow(r.Context(), `select `+portfolioProcedureColumns+` from education_portfolio_procedure_versions where id=$1::uuid and institution_id=$2`, strings.TrimSpace(id), s.institutionID(r)), &item)
	return item, err
}
func (s *Service) portfolioProcedureWhere(r *http.Request, query httpx.PageQuery) (string, []any) {
	where := "where institution_id=$1"
	args := []any{s.institutionID(r)}
	for _, key := range []string{"procedure_code", "title", "lifecycle_status"} {
		if value := strings.TrimSpace(query.Filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where += fmt.Sprintf(" and lower(%s) like $%d", key, len(args))
		}
	}
	return where, args
}
func (s *Service) portfolioProcedureTenantCode(r *http.Request) (string, error) {
	var code string
	err := s.pool.QueryRow(r.Context(), `select public.current_tenant_code()`).Scan(&code)
	return strings.TrimSpace(code), err
}
func decodePortfolioProcedureRequest(w http.ResponseWriter, r *http.Request, req *CreatePortfolioProcedureRequest) bool {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_payload"})
		return false
	}
	return validatePortfolioProcedureRequest(w, req)
}

func validatePortfolioProcedureRequest(w http.ResponseWriter, req *CreatePortfolioProcedureRequest) bool {
	req.ProcedureCode = strings.TrimSpace(req.ProcedureCode)
	req.Title = strings.TrimSpace(req.Title)
	req.SourceRef = strings.TrimSpace(req.SourceRef)
	if req.ProcedureCode == "" || req.Title == "" {
		httpx.JSON(w, 400, map[string]any{"code": "portfolio_procedure_code_and_title_required"})
		return false
	}
	if req.SourceRef == "" {
		req.SourceRef = "Ordinul nr. 3.858/2026, metodologia-cadru"
	}
	from, err := parseOptionalProcedureDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_effective_from"})
		return false
	}
	to, err := parseOptionalProcedureDate(req.EffectiveTo)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_effective_to"})
		return false
	}
	if from != nil && to != nil && to.Before(*from) {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_procedure_effective_period"})
		return false
	}
	return true
}
func parseOptionalProcedureDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
func optionalProcedureDate(value string) *time.Time {
	parsed, _ := parseOptionalProcedureDate(value)
	return parsed
}
func validExpectedUpdatedAt(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return err == nil
}
func jsonObject(value map[string]any) string {
	if value == nil {
		return "{}"
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
