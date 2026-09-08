package registratura

import (
	"context"
	"database/sql"
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

func (s *Service) StageDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	// Metadata-only staging cannot be completed: there is no upload token or
	// follow-up operation that binds bytes to the staged row. Keep the route
	// fail-closed so clients cannot create orphaned evidence records. The scanned
	// multipart upload endpoint is the only supported attachment write path.
	httpx.JSON(w, http.StatusGone, map[string]any{"code": "attachment_upload_required"})
}

// EnrichDocumentParity loads the new, tenant-scoped registratura read model.
// It deliberately keeps legacy list contracts intact while clients migrate.
func (s *Service) EnrichDocumentParity(ctx context.Context, document *Document) error {
	return enrichDocumentParity(ctx, s.pool, document)
}

func enrichDocumentParity(ctx context.Context, queryer documentQuerier, document *Document) error {
	if document == nil || strings.TrimSpace(document.ID) == "" {
		return nil
	}
	var externalDate, entryAt, exitAt, cancelledAt sql.NullString
	err := queryer.QueryRow(ctx, `
		select external_number,
		 case when external_number_date is null then null else to_char(external_number_date,'YYYY-MM-DD') end,
		 case when entry_at is null then null else to_char(entry_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end,
		 case when exit_at is null then null else to_char(exit_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end,
		 activity, record_kind,
		 case when cancelled_at is null then null else to_char(cancelled_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end,
		 cancelled_by, cancellation_reason, workflow_version
		from registratura_documents where id = $1::uuid`, document.ID).Scan(
		&document.ExternalNumber, &externalDate, &entryAt, &exitAt, &document.Activity,
		&document.RecordKind, &cancelledAt, &document.CancelledBy, &document.CancellationReason,
		&document.WorkflowVersion,
	)
	if err != nil {
		return err
	}
	if externalDate.Valid {
		document.ExternalNumberDate = &externalDate.String
	}
	if entryAt.Valid {
		document.EntryAt = &entryAt.String
	}
	if exitAt.Valid {
		document.ExitAt = &exitAt.String
	}
	if cancelledAt.Valid {
		document.CancelledAt = &cancelledAt.String
	}
	rows, err := queryer.Query(ctx, `
		select d.id::text, d.name
		from registratura_document_departments dd join registratura_departments d on d.id = dd.department_id
		where dd.document_id = $1::uuid order by d.name`, document.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		document.DepartmentIDs = append(document.DepartmentIDs, id)
		document.DepartmentNames = append(document.DepartmentNames, name)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var departmentID, userID, targetApproverID sql.NullString
	err = queryer.QueryRow(ctx, `select workflow_department_id::text, workflow_assigned_user_id::text, workflow_target_approver_id::text from registratura_documents where id=$1::uuid`, document.ID).Scan(&departmentID, &userID, &targetApproverID)
	if err == nil && (departmentID.Valid || userID.Valid || targetApproverID.Valid) {
		document.WorkflowAssignment = &WorkflowAssignment{}
		if departmentID.Valid {
			document.WorkflowAssignment.DepartmentID = &departmentID.String
		}
		if userID.Valid {
			document.WorkflowAssignment.UserID = &userID.String
		}
		if targetApproverID.Valid {
			document.WorkflowAssignment.TargetApproverID = &targetApproverID.String
		}
	}
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

func (s *Service) GetDocumentWorkflowHistory(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}
	if _, err := s.loadDocument(r.Context(), documentID); err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		return
	}
	rows, err := s.pool.Query(r.Context(), `select id::text, document_id::text, action, from_status, to_status, department_id::text, assigned_user_id::text, note, actor_subject, to_char(created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') from registratura_document_workflow_events where document_id=$1::uuid order by created_at asc`, documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_history_failed"})
		return
	}
	defer rows.Close()
	items := []DocumentWorkflowEvent{}
	for rows.Next() {
		var item DocumentWorkflowEvent
		var dep, user sql.NullString
		if err := rows.Scan(&item.ID, &item.DocumentID, &item.Action, &item.FromStatus, &item.ToStatus, &dep, &user, &item.Note, &item.ActorSubject, &item.CreatedAt); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_history_failed"})
			return
		}
		if dep.Valid {
			item.DepartmentID = &dep.String
		}
		if user.Valid {
			item.AssignedUserID = &user.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_history_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func workflowTransition(status, action string) (string, bool) {
	switch action {
	case "assign_department":
		return "ALOCAT_COMPARTIMENT", status == "INCOMING" || status == "ALOCAT_COMPARTIMENT"
	case "assign_user":
		return "IN_LUCRU", status == "ALOCAT_COMPARTIMENT"
	case "claim":
		return "IN_LUCRU", status == "ALOCAT_COMPARTIMENT"
	case "send_for_approval":
		return "FLUX_APROBARE", status == "IN_LUCRU"
	case "approve":
		return "FINALIZAT", status == "FLUX_APROBARE"
	case "reject":
		return "IN_LUCRU", status == "FLUX_APROBARE"
	default:
		return "", false
	}
}

func (s *Service) ApplyDocumentWorkflowAction(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_document_id"})
		return
	}
	var req DocumentWorkflowActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_action"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	req.Note = strings.TrimSpace(req.Note)
	if req.ExpectedVersion <= 0 {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "workflow_expected_version_required"})
		return
	}
	if len([]rune(req.Note)) > 500 {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "workflow_note_too_long", "max": 500})
		return
	}
	actor := authruntime.CurrentSubjectFromRequest(r)
	if actor == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "workflow_actor_missing"})
		return
	}
	var actorUserID string
	if err := s.pool.QueryRow(r.Context(), `select id::text from app_users where sub=$1`, actor).Scan(&actorUserID); err != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_actor_identity_missing"})
		return
	}
	var activeMembership bool
	if err := s.pool.QueryRow(r.Context(), `select exists(select 1 from app_memberships where user_id=$1::uuid and tenant_code=public.current_tenant_code() and active)`, actorUserID).Scan(&activeMembership); err != nil || !activeMembership {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_actor_membership_missing"})
		return
	}
	if req.Action == "claim" {
		if req.UserID != nil && strings.TrimSpace(*req.UserID) != "" && strings.TrimSpace(*req.UserID) != actorUserID {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_claim_must_be_self"})
			return
		}
		req.UserID = &actorUserID
	}
	if req.Action == "reject" && len([]rune(req.Note)) < 10 {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "workflow_rejection_note_length", "min": 10, "max": 500})
		return
	}
	if req.Action == "assign_department" && (req.DepartmentID == nil || strings.TrimSpace(*req.DepartmentID) == "") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "department_required"})
		return
	}
	if req.Action == "assign_user" && (req.UserID == nil || strings.TrimSpace(*req.UserID) == "") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "workflow_user_required"})
		return
	}
	if err := validateWorkflowActionFields(req); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": err.Error()})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var status, assignedUserID, targetApproverID, departmentID sql.NullString
	var version int
	var lockedUntil sql.NullTime
	if err := tx.QueryRow(r.Context(), `select status, workflow_version, workflow_assigned_user_id::text, workflow_target_approver_id::text, workflow_department_id::text, workflow_locked_until from registratura_documents where id=$1::uuid for update`, documentID).Scan(&status, &version, &assignedUserID, &targetApproverID, &departmentID, &lockedUntil); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		} else {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		}
		return
	}
	if lockedUntil.Valid && lockedUntil.Time.After(time.Now()) {
		httpx.JSON(w, http.StatusLocked, map[string]any{"code": "workflow_locked", "locked_until": lockedUntil.Time.UTC().Format(time.RFC3339)})
		return
	}
	if req.ExpectedVersion != version {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "stale_workflow_version", "current_version": version})
		return
	}
	next, ok := workflowTransition(status.String, req.Action)
	if !ok {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "invalid_workflow_transition", "status": status.String})
		return
	}
	if req.DepartmentID != nil {
		var exists bool
		err = tx.QueryRow(r.Context(), `select exists(select 1 from registratura_departments where id=$1::uuid and active)`, strings.TrimSpace(*req.DepartmentID)).Scan(&exists)
		if err != nil || !exists {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "department_not_found"})
			return
		}
	}
	if req.UserID != nil {
		var exists bool
		err = tx.QueryRow(r.Context(), `select exists(select 1 from app_memberships where user_id=$1::uuid and tenant_code=public.current_tenant_code() and active)`, strings.TrimSpace(*req.UserID)).Scan(&exists)
		if err != nil || !exists {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "workflow_user_not_found"})
			return
		}
	}
	if req.Action == "assign_user" || req.Action == "claim" {
		if !departmentID.Valid {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "workflow_department_required"})
			return
		}
		var userID string
		if req.Action == "claim" {
			userID = *req.UserID
		} else {
			userID = *req.UserID
		}
		var allowed bool
		if err := tx.QueryRow(r.Context(), `select exists(select 1 from registratura_user_departments ud join app_memberships m on m.user_id=ud.user_id and m.tenant_code=ud.tenant_code where ud.user_id=$1::uuid and ud.department_id=$2::uuid and ud.tenant_code=public.current_tenant_code() and m.active)`, userID, departmentID.String).Scan(&allowed); err != nil || !allowed {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_department_membership_required"})
			return
		}
	}
	if req.Action == "send_for_approval" {
		if !assignedUserID.Valid || assignedUserID.String != actorUserID {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_sender_must_be_assignee"})
			return
		}
		if req.UserID == nil || strings.TrimSpace(*req.UserID) == "" {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "target_approver_required"})
			return
		}
		if strings.TrimSpace(*req.UserID) == actorUserID {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "workflow_self_approval_forbidden"})
			return
		}
	}
	if req.Action == "approve" || req.Action == "reject" {
		if !targetApproverID.Valid || targetApproverID.String != actorUserID {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_target_approver_required"})
			return
		}
	}
	if req.Action == "approve" {
		var hasArchivablePDF bool
		if err := tx.QueryRow(r.Context(), `
			select exists(
				select 1 from registratura_document_attachments
				where document_id=$1::uuid and status='ready'
					and storage_state='ready' and scan_status='clean'
					and mime_type='application/pdf' and size_bytes>0 and checksum_sha256<>''
			)
		`, documentID).Scan(&hasArchivablePDF); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_attachment_check_failed"})
			return
		}
		if !hasArchivablePDF {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "workflow_clean_pdf_required"})
			return
		}
	}
	eventDepartmentID := req.DepartmentID
	eventUserID := req.UserID
	if req.Action != "assign_department" {
		eventDepartmentID = nil
	}
	if req.Action == "approve" || req.Action == "reject" {
		eventUserID = &actorUserID
	}
	_, err = tx.Exec(r.Context(), `insert into registratura_document_workflow_events(tenant_code,institution_id,document_id,action,from_status,to_status,department_id,assigned_user_id,note,actor_subject) values (public.current_tenant_code(),public.current_institution_id(),$1::uuid,$2,$3,$4,$5::uuid,$6::uuid,$7,$8)`, documentID, req.Action, status.String, next, eventDepartmentID, eventUserID, req.Note, actor)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	var persistedVersion int
	err = tx.QueryRow(r.Context(), `
		update registratura_documents set
			status=$2,
			workflow_version=workflow_version+1,
			workflow_department_id=case when $3='assign_department' then $4::uuid else workflow_department_id end,
			workflow_assigned_user_id=case when $3 in ('assign_user','claim') then $5::uuid when $3='send_for_approval' then null else workflow_assigned_user_id end,
			workflow_target_approver_id=case when $3='send_for_approval' then $5::uuid when $3 in ('approve','reject') then null else workflow_target_approver_id end,
			rejection_count=case when $3='reject' then rejection_count+1 else rejection_count end,
			workflow_locked_until=case
				when $3='approve' then now()-interval '1 second'
				-- Costesti-compatible repeat-rejection guard: the third rejection
				-- keeps the document in remediation but prevents immediate resend.
				when $3='reject' and rejection_count+1 >= 3 then now()+interval '24 hours'
				when $3='reject' then now()-interval '1 second'
				else workflow_locked_until end,
			updated_at=now()
		where id=$1::uuid and workflow_version=$6
		returning workflow_version`, documentID, next, req.Action, req.DepartmentID, req.UserID, req.ExpectedVersion).Scan(&persistedVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "stale_workflow_version"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	if req.Action == "approve" {
		if _, err := tx.Exec(r.Context(), `insert into registratura_archive_outbox(tenant_code,institution_id,document_id,event_type,payload) values(public.current_tenant_code(),public.current_institution_id(),$1::uuid,'document_finalized',jsonb_build_object('document_id',$1::text,'workflow_version',$2)) on conflict (tenant_code,document_id,event_type) do nothing`, documentID, persistedVersion); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_finalization_outbox_failed"})
			return
		}
	}
	doc, err := s.loadWorkflowDocumentTx(r.Context(), tx, documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	if err := enrichDocumentParity(r.Context(), tx, &doc); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	s.logAudit(r, "registratura.documents.workflow."+req.Action, "document", documentID, "Document workflow transitioned.", map[string]any{"from": status.String, "to": next})
	httpx.JSON(w, http.StatusOK, doc)
}

func validateWorkflowActionFields(req DocumentWorkflowActionRequest) error {
	hasDepartment := req.DepartmentID != nil && strings.TrimSpace(*req.DepartmentID) != ""
	hasUser := req.UserID != nil && strings.TrimSpace(*req.UserID) != ""
	switch req.Action {
	case "assign_department":
		if hasUser {
			return errors.New("workflow_user_not_allowed")
		}
	case "assign_user", "claim", "send_for_approval":
		if hasDepartment {
			return errors.New("workflow_department_not_allowed")
		}
	case "approve", "reject":
		if hasDepartment || hasUser {
			return errors.New("workflow_assignment_not_allowed")
		}
	}
	return nil
}

func (s *Service) PrintDocumentPDF(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	doc, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		} else {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "document_print_failed"})
		}
		return
	}
	_ = s.EnrichDocumentParity(r.Context(), &doc)
	pdf := buildSimplePDF("Registratura - "+doc.RegistryNumber, []string{doc.RegistryNumber, doc.Subject, doc.DocumentType + " | " + doc.Direction, doc.Status, doc.Correspondent + " -> " + doc.AssignedTo, doc.ExternalNumber, doc.Activity})
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="document-%s.pdf"`, strings.ReplaceAll(doc.RegistryNumber, "/", "-")))
	_, _ = w.Write(pdf)
	s.logAudit(r, "registratura.documents.print", "document", documentID, "Single document PDF generated.", nil)
}

func (s *Service) enrichDocumentOrError(ctx context.Context, document *Document) error {
	return s.EnrichDocumentParity(ctx, document)
}
