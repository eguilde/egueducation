package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	}, []string{"valorification_code", "scope", "status", "requested_by", "target_institution"})
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

	code := fmt.Sprintf("VAL-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
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
	tag, err := s.pool.Exec(r.Context(), `delete from education_portfolio_valorifications where id = $1 and portfolio_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_valorification_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_portfolio_valorification_not_found")
		return
	}
	s.logAudit(r, "education.portfolios.valorification.delete", "portfolio_valorification", itemID, "Portfolio valorification flow deleted.", map[string]any{"portfolio_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func buildPortfolioValorificationFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"epv.portfolio_id = $1", "epv.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"valorification_code": "epv.valorification_code",
		"scope":               "epv.scope",
		"status":              "epv.status",
		"requested_by":        "epv.requested_by",
		"target_institution":  "epv.target_institution",
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

func nullableDate(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
