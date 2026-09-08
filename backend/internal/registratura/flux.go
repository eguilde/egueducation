package registratura

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

// Flux endpoints mirror Costesti's Queue, Mapa and Pipeline projections. All
// predicates run after the session tenant has been bound by RequireAuthenticated.
type fluxQuery struct {
	page, pageSize  int
	sort, direction string
	filters         map[string]string
}

func parseFluxQuery(values url.Values) fluxQuery {
	copyValues := make(url.Values, len(values))
	for key, value := range values {
		copyValues[key] = append([]string(nil), value...)
	}
	if copyValues.Get("pageSize") == "" {
		copyValues.Set("pageSize", copyValues.Get("limit"))
	}
	if copyValues.Get("sort") == "" {
		copyValues.Set("sort", copyValues.Get("sortField"))
	}
	if copyValues.Get("direction") == "" {
		copyValues.Set("direction", copyValues.Get("sortOrder"))
	}
	for _, key := range []string{"nr_doc", "continut", "emitent", "compartiment", "tip", "status", "mapa_filter"} {
		if value := strings.TrimSpace(copyValues.Get(key)); value != "" {
			copyValues.Set("filter."+key, value)
		}
	}
	page := httpx.ParsePageQuery(copyValues, map[string]struct{}{
		"registry_number": {}, "entry_at": {}, "registered_at": {}, "subject": {}, "correspondent": {}, "department": {}, "status": {}, "document_type": {},
	}, []string{"nr_doc", "continut", "emitent", "compartiment", "tip", "status", "mapa_filter"})
	return fluxQuery{page: page.Page, pageSize: page.PageSize, sort: page.Sort, direction: page.Direction, filters: page.Filters}
}

func fluxSortColumn(field string) string {
	switch field {
	case "registry_number":
		return "d.registry_number"
	case "entry_at":
		return "d.entry_at"
	case "subject":
		return "d.subject"
	case "correspondent":
		return "d.correspondent"
	case "department":
		return "department.name"
	case "status":
		return "d.status"
	case "document_type":
		return "d.document_type"
	default:
		return "d.registered_at"
	}
}

func fluxFilters(q fluxQuery, clauses []string, args []any) ([]string, []any) {
	contains := func(column, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := q.filters["nr_doc"]; value != "" {
		contains("d.registry_number", value)
	}
	if value := q.filters["continut"]; value != "" {
		contains("d.subject", value)
	}
	if value := q.filters["emitent"]; value != "" {
		contains("d.correspondent", value)
	}
	if value := q.filters["compartiment"]; value != "" {
		contains("department.name", value)
	}
	if value := q.filters["tip"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("d.document_type=$%d", len(args)))
	}
	if value := q.filters["status"]; value != "" {
		args = append(args, normalizeDocumentStatus(value))
		clauses = append(clauses, fmt.Sprintf("d.status=$%d", len(args)))
	}
	return clauses, args
}

const fluxJoins = `
 from registratura_documents d
 left join registratura_departments department on department.id=d.workflow_department_id
 left join app_users assignee on assignee.id=d.workflow_assigned_user_id
 left join app_users approver on approver.id=d.workflow_target_approver_id`

func (s *Service) listFlux(ctxContext *http.Request, q fluxQuery, clauses []string, args []any) ([]FluxDocument, int, error) {
	clauses, args = fluxFilters(q, clauses, args)
	where := " where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(ctxContext.Context(), "select count(*)"+fluxJoins+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, q.pageSize, (q.page-1)*q.pageSize)
	rows, err := s.pool.Query(ctxContext.Context(), fmt.Sprintf(`select d.id::text,d.registry_number,d.subject,d.document_type,d.status,d.direction,d.correspondent,
	case when d.entry_at is null then null else to_char(d.entry_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end,
	to_char(d.registered_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	d.workflow_department_id::text,department.name,d.workflow_assigned_user_id::text,assignee.name,d.workflow_target_approver_id::text,approver.name,d.rejection_count,d.workflow_version
	%s%s order by %s %s nulls last,d.registry_number asc limit $%d offset $%d`, fluxJoins, where, fluxSortColumn(q.sort), strings.ToUpper(q.direction), len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]FluxDocument, 0, q.pageSize)
	for rows.Next() {
		var item FluxDocument
		var entry, depID, depName, userID, userName, targetID, targetName sql.NullString
		if err := rows.Scan(&item.ID, &item.RegistryNumber, &item.Subject, &item.DocumentType, &item.Status, &item.Direction, &item.Correspondent, &entry, &item.RegisteredAt, &depID, &depName, &userID, &userName, &targetID, &targetName, &item.RejectionCount, &item.WorkflowVersion); err != nil {
			return nil, 0, err
		}
		if entry.Valid {
			item.EntryAt = &entry.String
		}
		if depID.Valid {
			item.DepartmentID = &depID.String
		}
		if depName.Valid {
			item.DepartmentName = &depName.String
		}
		if userID.Valid {
			item.AssignedUserID = &userID.String
		}
		if userName.Valid {
			item.AssignedUserName = &userName.String
		}
		if targetID.Valid {
			item.TargetApproverID = &targetID.String
		}
		if targetName.Valid {
			item.TargetApproverName = &targetName.String
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *Service) FluxMyQueue(w http.ResponseWriter, r *http.Request) {
	var userID string
	if err := s.pool.QueryRow(r.Context(), `select id::text from app_users where sub=$1`, authruntime.CurrentSubjectFromRequest(r)).Scan(&userID); err != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_actor_identity_missing"})
		return
	}
	q := parseFluxQuery(r.URL.Query())
	clauses := []string{"d.institution_id=public.current_institution_id()", `(d.workflow_assigned_user_id=$1::uuid and d.status='IN_LUCRU' or (d.status='ALOCAT_COMPARTIMENT' and exists(select 1 from registratura_user_departments ud where ud.user_id=$1::uuid and ud.department_id=d.workflow_department_id and ud.tenant_code=public.current_tenant_code())))`}
	items, total, err := s.listFlux(r, q, clauses, []any{userID})
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_queue_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.page, q.pageSize)
}

func (s *Service) FluxMapa(w http.ResponseWriter, r *http.Request) {
	q := parseFluxQuery(r.URL.Query())
	clauses := []string{"d.institution_id=public.current_institution_id()"}
	args := []any{}
	if q.filters["mapa_filter"] == "mine" {
		var userID string
		if err := s.pool.QueryRow(r.Context(), `select id::text from app_users where sub=$1`, authruntime.CurrentSubjectFromRequest(r)).Scan(&userID); err != nil {
			httpx.JSON(w, 403, map[string]any{"code": "workflow_actor_identity_missing"})
			return
		}
		args = append(args, userID)
		clauses = append(clauses, fmt.Sprintf("d.workflow_assigned_user_id=$%d::uuid", len(args)))
	}
	items, total, err := s.listFlux(r, q, clauses, args)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_mapa_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.page, q.pageSize)
}

func (s *Service) FluxPipeline(w http.ResponseWriter, r *http.Request) {
	q := parseFluxQuery(r.URL.Query())
	items, total, err := s.listFlux(r, q, []string{"d.institution_id=public.current_institution_id()"}, nil)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_pipeline_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, q.page, q.pageSize)
}

func (s *Service) FluxPipelineStats(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `select status,count(*) from registratura_documents where institution_id=public.current_institution_id() group by status order by status`)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "workflow_pipeline_stats_failed"})
		return
	}
	defer rows.Close()
	items := []FluxPipelineStat{}
	for rows.Next() {
		var item FluxPipelineStat
		if err := rows.Scan(&item.Status, &item.Count); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "workflow_pipeline_stats_failed"})
			return
		}
		items = append(items, item)
	}
	httpx.JSON(w, 200, items)
}
