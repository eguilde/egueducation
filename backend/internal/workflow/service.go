package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/dossier"
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

func (s *Service) Dashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response DashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) filter (where status <> 'archived') as active_tasks,
			count(*) filter (where due_at is not null and due_at < current_date and status not in ('approved', 'archived')) as overdue_tasks,
			count(*) filter (where status = 'waiting_approval') as waiting_approval,
			(select count(*) from workflow_definitions where active = true) as active_definitions
		from workflow_instances
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.ActiveTasks,
		&response.Stats.OverdueTasks,
		&response.Stats.WaitingApproval,
		&response.Stats.ActiveDefinitions,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_dashboard_failed"})
		return
	}

	response.Stats.ReadyDossiers, response.Stats.BlockedDossiers, err = s.readinessStats(r.Context(), s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select code, name, category, initial_step, sla_hours, active
		from workflow_definitions
		order by name
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_definitions_failed"})
		return
	}
	defer rows.Close()

	definitions := []Definition{}
	for rows.Next() {
		var definition Definition
		if err := rows.Scan(
			&definition.Code,
			&definition.Name,
			&definition.Category,
			&definition.InitialStep,
			&definition.SLAHours,
			&definition.Active,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_definitions_failed"})
			return
		}
		definitions = append(definitions, definition)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_definitions_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, definitions)
}

func (s *Service) ListTasks(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"title":           {},
			"definition_name": {},
			"status":          {},
			"priority":        {},
			"assigned_to":     {},
			"current_step":    {},
			"due_at":          {},
			"started_at":      {},
		},
		[]string{"title", "definition_code", "status", "priority", "assigned_to", "due_on"},
	)

	whereClause, args := buildTaskFilters(s.institutionID(r), query.Filters)

	var total int
	countSQL := `
		select count(*)
		from workflow_instances wi
		join workflow_definitions wd on wd.code = wi.definition_code
		` + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_list_failed"})
		return
	}

	sortField := sortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			wi.id::text,
			wi.definition_code,
			wd.name,
			wi.title,
			wi.document_number,
			wi.source_module,
			case when wi.source_record_id is null then null else wi.source_record_id::text end as source_record_id,
			wi.status,
			wi.priority,
			wi.assigned_to,
			wi.current_step,
			case when wi.due_at is null then null else to_char(wi.due_at, 'YYYY-MM-DD') end as due_at,
			to_char(wi.started_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as started_at,
			to_char(wi.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as updated_at,
			wi.institution_id,
			wi.summary,
			%s
		from workflow_instances wi
		join workflow_definitions wd on wd.code = wi.definition_code
		%s
		%s
		order by %s %s, wi.updated_at desc
		limit $%d offset $%d
	`, dossier.CountSQL("link_stats"), dossier.LateralJoinSQL("link_stats", "wi.source_module", "wi.source_record_id"), whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_list_failed"})
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0, query.PageSize)
	for rows.Next() {
		var task Task
		var counts dossier.RelationCounts
		if err := rows.Scan(
			&task.ID,
			&task.DefinitionCode,
			&task.DefinitionName,
			&task.Title,
			&task.DocumentNumber,
			&task.SourceModule,
			&task.SourceRecordID,
			&task.Status,
			&task.Priority,
			&task.AssignedTo,
			&task.CurrentStep,
			&task.DueAt,
			&task.StartedAt,
			&task.UpdatedAt,
			&task.InstitutionID,
			&task.Summary,
			&counts.Total,
			&counts.Primary,
			&counts.Supporting,
			&counts.Decision,
			&counts.ArchiveBasis,
			&counts.GDPRBasis,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_list_failed"})
			return
		}
		task.LinkedDocumentsCount = counts.Total
		task.DossierReady, task.MissingRelations, err = dossier.Evaluate(r.Context(), s.pool, task.SourceModule, task.SourceRecordID != nil, counts, dossier.PurposeReadiness)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_list_failed"})
			return
		}
		task.AvailableActions = availableActions(task.Status)
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_list_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, tasks, total, query.Page, query.PageSize)
}

func (s *Service) TaskFilters(w http.ResponseWriter, r *http.Request) {
	response := FiltersResponse{
		Statuses:   []string{},
		Priorities: []string{},
		Assignees:  []string{},
	}

	load := func(sql string) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql)
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
	if response.Statuses, err = load("select distinct status from workflow_instances order by status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_filters_failed"})
		return
	}
	if response.Priorities, err = load("select distinct priority from workflow_instances order by priority"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_filters_failed"})
		return
	}
	if response.Assignees, err = load("select distinct assigned_to from workflow_instances where assigned_to <> '' order by assigned_to"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_payload"})
		return
	}

	req.DefinitionCode = strings.TrimSpace(req.DefinitionCode)
	req.Title = strings.TrimSpace(req.Title)
	req.DocumentNumber = strings.TrimSpace(req.DocumentNumber)
	req.SourceModule = strings.TrimSpace(req.SourceModule)
	req.SourceRecordID = strings.TrimSpace(req.SourceRecordID)
	req.Priority = strings.TrimSpace(req.Priority)
	req.AssignedTo = strings.TrimSpace(req.AssignedTo)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.DefinitionCode == "" || req.Title == "" || req.Priority == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_workflow_fields"})
		return
	}
	if !contains([]string{"low", "medium", "high", "urgent"}, req.Priority) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_priority"})
		return
	}

	var task Task
	err := s.pool.QueryRow(r.Context(), `
		with inserted as (
			insert into workflow_instances (
				definition_code,
				title,
				document_number,
				source_module,
				source_record_id,
				status,
				priority,
				assigned_to,
				current_step,
				due_at,
				institution_id,
				summary
			)
			select
				definition.code,
				$2,
				$3,
				coalesce(nullif($4, ''), definition.category),
				case when nullif($5, '') is null then null else $5::uuid end,
				'new',
				$6,
				$7,
				definition.initial_step,
				case when nullif($8, '') is null then null else $8::date end,
				$10,
				$9
			from workflow_definitions definition
			where definition.code = $1 and definition.active = true
			returning
				id::text,
				definition_code,
				title,
				document_number,
				source_module,
				case when source_record_id is null then null else source_record_id::text end as source_record_id,
				status,
				priority,
				assigned_to,
				current_step,
				case when due_at is null then null else to_char(due_at, 'YYYY-MM-DD') end as due_at,
				to_char(started_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as started_at,
				to_char(updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as updated_at,
				institution_id,
				summary
		)
		select
			inserted.id,
			inserted.definition_code,
			definition.name,
			inserted.title,
			inserted.document_number,
			inserted.source_module,
			inserted.source_record_id,
			inserted.status,
			inserted.priority,
			inserted.assigned_to,
			inserted.current_step,
			inserted.due_at,
			inserted.started_at,
			inserted.updated_at,
			inserted.institution_id,
			inserted.summary
		from inserted
		join workflow_definitions definition on definition.code = inserted.definition_code
	`,
		req.DefinitionCode,
		req.Title,
		req.DocumentNumber,
		req.SourceModule,
		req.SourceRecordID,
		req.Priority,
		req.AssignedTo,
		trimOrEmpty(req.DueDate),
		req.Summary,
		s.institutionID(r),
	).Scan(
		&task.ID,
		&task.DefinitionCode,
		&task.DefinitionName,
		&task.Title,
		&task.DocumentNumber,
		&task.SourceModule,
		&task.SourceRecordID,
		&task.Status,
		&task.Priority,
		&task.AssignedTo,
		&task.CurrentStep,
		&task.DueAt,
		&task.StartedAt,
		&task.UpdatedAt,
		&task.InstitutionID,
		&task.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "workflow_create_failed"})
		return
	}
	task.DossierReady, task.MissingRelations, err = dossier.Evaluate(r.Context(), s.pool, task.SourceModule, task.SourceRecordID != nil, dossier.RelationCounts{}, dossier.PurposeReadiness)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_create_failed"})
		return
	}
	task.AvailableActions = availableActions(task.Status)

	s.logAudit(r, "workflow.tasks.create", "workflow_task", task.ID, "Workflow task created.", map[string]any{
		"definition_code":  task.DefinitionCode,
		"definition_name":  task.DefinitionName,
		"title":            task.Title,
		"status":           task.Status,
		"priority":         task.Priority,
		"current_step":     task.CurrentStep,
		"source_module":    task.SourceModule,
		"source_record_id": task.SourceRecordID,
		"dossier_ready":    task.DossierReady,
	})

	httpx.JSON(w, http.StatusCreated, task)
}

func (s *Service) TransitionTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if strings.TrimSpace(taskID) == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_workflow_task_id"})
		return
	}

	var req TransitionTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_transition"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_workflow_action"})
		return
	}

	var currentStatus string
	var sourceModule string
	var sourceRecordID *string
	var counts dossier.RelationCounts
	if err := s.pool.QueryRow(r.Context(), `
		select
			wi.status,
			wi.source_module,
			case when wi.source_record_id is null then null else wi.source_record_id::text end as source_record_id,
			`+dossier.CountSQL("link_stats")+`
		from workflow_instances wi
		`+dossier.LateralJoinSQL("link_stats", "wi.source_module", "wi.source_record_id")+`
		where wi.id::text = $1
	`, taskID).Scan(
		&currentStatus,
		&sourceModule,
		&sourceRecordID,
		&counts.Total,
		&counts.Primary,
		&counts.Supporting,
		&counts.Decision,
		&counts.ArchiveBasis,
		&counts.GDPRBasis,
	); err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "workflow_task_not_found"})
		return
	}

	purpose := dossier.PurposeSubmit
	if req.Action == "approve" {
		purpose = dossier.PurposeApprove
	}
	dossierReady, missingRelations, err := dossier.Evaluate(r.Context(), s.pool, sourceModule, sourceRecordID != nil, counts, purpose)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_transition_failed"})
		return
	}
	if actionRequiresReady(req.Action) && !dossierReady {
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"code":              "workflow_dossier_incomplete",
			"missing_relations": missingRelations,
		})
		return
	}

	nextStatus, nextStep, ok := resolveTransition(currentStatus, req.Action)
	if !ok {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_action"})
		return
	}

	var task Task
	err = s.pool.QueryRow(r.Context(), `
		update workflow_instances wi
		set
			status = $2,
			current_step = $3,
			updated_at = now()
		from workflow_definitions wd
		where wi.id::text = $1
			and wd.code = wi.definition_code
		returning
			wi.id::text,
			wi.definition_code,
			wd.name,
			wi.title,
			wi.document_number,
			wi.source_module,
			case when wi.source_record_id is null then null else wi.source_record_id::text end as source_record_id,
			wi.status,
			wi.priority,
			wi.assigned_to,
			wi.current_step,
			case when wi.due_at is null then null else to_char(wi.due_at, 'YYYY-MM-DD') end as due_at,
			to_char(wi.started_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as started_at,
			to_char(wi.updated_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as updated_at,
			wi.institution_id,
			wi.summary
	`, taskID, nextStatus, nextStep).Scan(
		&task.ID,
		&task.DefinitionCode,
		&task.DefinitionName,
		&task.Title,
		&task.DocumentNumber,
		&task.SourceModule,
		&task.SourceRecordID,
		&task.Status,
		&task.Priority,
		&task.AssignedTo,
		&task.CurrentStep,
		&task.DueAt,
		&task.StartedAt,
		&task.UpdatedAt,
		&task.InstitutionID,
		&task.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_transition_failed"})
		return
	}
	task.LinkedDocumentsCount = counts.Total
	task.DossierReady, task.MissingRelations, err = dossier.Evaluate(r.Context(), s.pool, task.SourceModule, task.SourceRecordID != nil, counts, dossier.PurposeReadiness)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_transition_failed"})
		return
	}
	task.AvailableActions = availableActions(task.Status)

	s.logAudit(r, "workflow.tasks.transition", "workflow_task", task.ID, "Workflow task transitioned.", map[string]any{
		"action":            req.Action,
		"previous_status":   currentStatus,
		"status":            task.Status,
		"current_step":      task.CurrentStep,
		"source_module":     task.SourceModule,
		"source_record_id":  task.SourceRecordID,
		"linked_documents":  task.LinkedDocumentsCount,
		"dossier_ready":     task.DossierReady,
		"missing_relations": task.MissingRelations,
	})

	httpx.JSON(w, http.StatusOK, task)
}

func buildTaskFilters(institutionID string, filters map[string]string) (string, []any) {
	clauses := []string{"wi.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["title"]; value != "" {
		addContains("wi.title", value)
	}
	if value := filters["definition_code"]; value != "" {
		addEqual("wi.definition_code", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("wi.status", value)
	}
	if value := filters["priority"]; value != "" {
		addEqual("wi.priority", value)
	}
	if value := filters["assigned_to"]; value != "" {
		addEqual("wi.assigned_to", value)
	}
	if value := filters["due_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("wi.due_at = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func sortColumn(field string) string {
	switch field {
	case "title":
		return "wi.title"
	case "definition_name":
		return "wd.name"
	case "status":
		return "wi.status"
	case "priority":
		return "wi.priority"
	case "assigned_to":
		return "wi.assigned_to"
	case "current_step":
		return "wi.current_step"
	case "due_at":
		return "wi.due_at"
	default:
		return "wi.started_at"
	}
}

func availableActions(status string) []string {
	switch status {
	case "new":
		return []string{"start", "archive"}
	case "in_progress":
		return []string{"submit", "archive"}
	case "waiting_approval":
		return []string{"approve", "return"}
	case "approved":
		return []string{"archive", "return"}
	default:
		return []string{}
	}
}

func resolveTransition(status string, action string) (string, string, bool) {
	switch {
	case status == "new" && action == "start":
		return "in_progress", "Verificare și completare", true
	case status == "new" && action == "archive":
		return "archived", "Arhivat", true
	case status == "in_progress" && action == "submit":
		return "waiting_approval", "Aprobare finală", true
	case status == "in_progress" && action == "archive":
		return "archived", "Arhivat", true
	case status == "waiting_approval" && action == "approve":
		return "approved", "Finalizat", true
	case status == "waiting_approval" && action == "return":
		return "in_progress", "Remediere și completare", true
	case status == "approved" && action == "archive":
		return "archived", "Arhivat", true
	case status == "approved" && action == "return":
		return "in_progress", "Remediere și completare", true
	default:
		return "", "", false
	}
}

func actionRequiresReady(action string) bool {
	switch action {
	case "submit", "approve":
		return true
	default:
		return false
	}
}

func (s *Service) readinessStats(ctx context.Context, institutionID string) (int, int, error) {
	rows, err := s.pool.Query(ctx, `
		select
			wi.source_module,
			case when wi.source_record_id is null then null else wi.source_record_id::text end as source_record_id,
			`+dossier.CountSQL("link_stats")+`
		from workflow_instances wi
		`+dossier.LateralJoinSQL("link_stats", "wi.source_module", "wi.source_record_id")+`
		where wi.institution_id = $1
			and wi.status <> 'archived'
	`, institutionID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	ready := 0
	blocked := 0
	for rows.Next() {
		var sourceModule string
		var sourceRecordID *string
		var counts dossier.RelationCounts
		if err := rows.Scan(
			&sourceModule,
			&sourceRecordID,
			&counts.Total,
			&counts.Primary,
			&counts.Supporting,
			&counts.Decision,
			&counts.ArchiveBasis,
			&counts.GDPRBasis,
		); err != nil {
			return 0, 0, err
		}
		dossierReady, _, err := dossier.Evaluate(ctx, s.pool, sourceModule, sourceRecordID != nil, counts, dossier.PurposeReadiness)
		if err != nil {
			return 0, 0, err
		}
		if dossierReady {
			ready++
		} else {
			blocked++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	return ready, blocked, nil
}

func trimOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
