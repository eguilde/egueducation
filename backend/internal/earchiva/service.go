package earchiva

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

type Service struct {
	pool *appdb.SessionPool
}

func NewService(pool *appdb.SessionPool) *Service {
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

func (s *Service) institutionID(r *http.Request) string {
	return strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
}

func (s *Service) Nomenclatures(w http.ResponseWriter, r *http.Request) {
	response := FiltersResponse{}

	load := func(domain string) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), `
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

	var err error
	if response.Fonds, err = load("archive_fond"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}
	if response.Series, err = load("archive_series"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}
	if response.Statuses, err = load("archive_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}
	if response.SourceModules, err = load("archive_source_module"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), "select distinct assigned_archivist from archive_records where assigned_archivist <> '' order by assigned_archivist")
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}
	defer rows.Close()
	response.Archivists = []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
			return
		}
		response.Archivists = append(response.Archivists, value)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_nomenclatures_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Dashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response DashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status = 'validated') as validated_records,
			count(*) filter (where status = 'draft') as draft_records,
			count(distinct fond) as unique_fonds
		from archive_records
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalRecords,
		&response.Stats.ValidatedRecords,
		&response.Stats.DraftRecords,
		&response.Stats.UniqueFonds,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) ListRecords(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"record_number":      {},
			"title":              {},
			"fond":               {},
			"series":             {},
			"source_module":      {},
			"status":             {},
			"assigned_archivist": {},
			"retention_years":    {},
			"archived_at":        {},
		},
		[]string{"record_number", "title", "fond", "series", "source_module", "status", "assigned_archivist", "archived_on"},
	)

	whereClause, args := buildRecordFilters(s.institutionID(r), query.Filters)

	var total int
	countSQL := "select count(*) from archive_records ar " + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_list_failed"})
		return
	}

	sortField := sortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			ar.id::text,
			ar.record_number,
			ar.title,
			ar.fond,
			ar.series,
			ar.source_module,
			ar.source_reference,
			ar.status,
			ar.retention_years,
			ar.assigned_archivist,
			ar.box_number,
			ar.location_code,
			to_char(ar.archived_at, 'YYYY-MM-DD') as archived_at,
			ar.institution_id,
			ar.notes
		from archive_records ar
		%s
		order by %s %s, ar.record_number asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_list_failed"})
		return
	}
	defer rows.Close()

	records := make([]Record, 0, query.PageSize)
	for rows.Next() {
		var record Record
		if err := rows.Scan(
			&record.ID,
			&record.RecordNumber,
			&record.Title,
			&record.Fond,
			&record.Series,
			&record.SourceModule,
			&record.SourceReference,
			&record.Status,
			&record.RetentionYears,
			&record.AssignedArchivist,
			&record.BoxNumber,
			&record.LocationCode,
			&record.ArchivedAt,
			&record.InstitutionID,
			&record.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_list_failed"})
			return
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_list_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, records, total, query.Page, query.PageSize)
}

func (s *Service) Filters(w http.ResponseWriter, r *http.Request) {
	s.Nomenclatures(w, r)
}

func (s *Service) CreateRecord(w http.ResponseWriter, r *http.Request) {
	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_payload"})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Fond = strings.TrimSpace(req.Fond)
	req.Series = strings.TrimSpace(req.Series)
	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.SourceReference = strings.TrimSpace(req.SourceReference)
	req.Status = strings.TrimSpace(req.Status)
	req.AssignedArchivist = strings.TrimSpace(req.AssignedArchivist)
	req.BoxNumber = strings.TrimSpace(req.BoxNumber)
	req.LocationCode = strings.TrimSpace(req.LocationCode)
	req.ArchivedAt = strings.TrimSpace(req.ArchivedAt)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.Title == "" || req.Fond == "" || req.Series == "" || req.SourceModule == "" || req.Status == "" || req.ArchivedAt == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_archive_fields"})
		return
	}
	ctx := r.Context()
	if !s.isNomenclatureAllowed(ctx, "archive_status", req.Status) ||
		!s.isNomenclatureAllowed(ctx, "archive_fond", req.Fond) ||
		!s.isNomenclatureAllowed(ctx, "archive_series", req.Series) ||
		!s.isNomenclatureAllowed(ctx, "archive_source_module", req.SourceModule) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_status"})
		return
	}
	if req.RetentionYears < 1 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_retention"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ArchivedAt); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_archive_date"})
		return
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_create_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	recordNumber, err := nextRecordNumber(ctx, tx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_number_failed"})
		return
	}

	var record Record
	err = tx.QueryRow(ctx, `
		insert into archive_records (
			record_number,
			title,
			fond,
			series,
			source_module,
			source_reference,
			status,
			retention_years,
			assigned_archivist,
			box_number,
			location_code,
			archived_at,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		returning
			id::text,
			record_number,
			title,
			fond,
			series,
			source_module,
			source_reference,
			status,
			retention_years,
			assigned_archivist,
			box_number,
			location_code,
			to_char(archived_at, 'YYYY-MM-DD'),
			institution_id,
			notes
	`,
		recordNumber,
		req.Title,
		req.Fond,
		req.Series,
		req.SourceModule,
		req.SourceReference,
		req.Status,
		req.RetentionYears,
		req.AssignedArchivist,
		req.BoxNumber,
		req.LocationCode,
		req.ArchivedAt,
		s.institutionID(r),
		req.Notes,
	).Scan(
		&record.ID,
		&record.RecordNumber,
		&record.Title,
		&record.Fond,
		&record.Series,
		&record.SourceModule,
		&record.SourceReference,
		&record.Status,
		&record.RetentionYears,
		&record.AssignedArchivist,
		&record.BoxNumber,
		&record.LocationCode,
		&record.ArchivedAt,
		&record.InstitutionID,
		&record.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_create_failed"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_create_failed"})
		return
	}

	s.logAudit(r, "earchiva.records.create", "archive_record", record.ID, "Archive record created.", map[string]any{
		"record_number":      record.RecordNumber,
		"fond":               record.Fond,
		"series":             record.Series,
		"status":             record.Status,
		"source_module":      record.SourceModule,
		"source_reference":   record.SourceReference,
		"retention_years":    record.RetentionYears,
		"assigned_archivist": record.AssignedArchivist,
	})

	httpx.JSON(w, http.StatusCreated, record)
}

func buildRecordFilters(institutionID string, filters map[string]string) (string, []any) {
	clauses := []string{"ar.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["record_number"]; value != "" {
		addContains("ar.record_number", value)
	}
	if value := filters["title"]; value != "" {
		addContains("ar.title", value)
	}
	if value := filters["fond"]; value != "" {
		addEqual("ar.fond", value)
	}
	if value := filters["series"]; value != "" {
		addEqual("ar.series", value)
	}
	if value := filters["source_module"]; value != "" {
		addEqual("ar.source_module", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("ar.status", value)
	}
	if value := filters["assigned_archivist"]; value != "" {
		addEqual("ar.assigned_archivist", value)
	}
	if value := filters["archived_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("ar.archived_at = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func sortColumn(field string) string {
	switch field {
	case "record_number":
		return "ar.record_number"
	case "title":
		return "ar.title"
	case "fond":
		return "ar.fond"
	case "series":
		return "ar.series"
	case "source_module":
		return "ar.source_module"
	case "status":
		return "ar.status"
	case "retention_years":
		return "ar.retention_years"
	case "assigned_archivist":
		return "ar.assigned_archivist"
	case "archived_at":
		return "ar.archived_at"
	default:
		return "ar.archived_at"
	}
}

func nextRecordNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	year := time.Now().UTC().Year()
	var count int
	if err := tx.QueryRow(ctx, "select count(*) from archive_records where extract(year from archived_at) = $1", year).Scan(&count); err != nil {
		return "", err
	}
	return fmt.Sprintf("ARH-%d-%04d", year, count+1), nil
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
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
