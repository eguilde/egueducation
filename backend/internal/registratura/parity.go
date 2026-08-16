package registratura

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) StageDocumentAttachment(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var req StageDocumentAttachmentRequest
	if documentID == "" || json.NewDecoder(r.Body).Decode(&req) != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_attachment_stage"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.FileName = strings.TrimSpace(req.FileName)
	req.MimeType = strings.TrimSpace(req.MimeType)
	req.Category = strings.TrimSpace(req.Category)
	req.ChecksumSHA256 = strings.ToLower(strings.TrimSpace(req.ChecksumSHA256))
	if req.Title == "" || req.FileName == "" || req.MimeType == "" || req.Category == "" || req.SizeBytes <= 0 || req.SizeBytes > 100*1024*1024 || len(req.ChecksumSHA256) != 64 {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_attachment_stage"})
		return
	}
	if _, err := s.loadDocument(r.Context(), documentID); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, 404, map[string]any{"code": "document_not_found"})
		} else {
			httpx.JSON(w, 500, map[string]any{"code": "attachment_stage_failed"})
		}
		return
	}
	var item DocumentAttachment
	err := s.pool.QueryRow(r.Context(), `with key as (select 'tenants/' || public.current_tenant_code() || '/registratura/staged/' || gen_random_uuid()::text as storage_key) insert into registratura_document_attachments(document_id,tenant_code,institution_id,title,file_name,mime_type,storage_key,size_bytes,category,status,uploaded_by,checksum_sha256,scan_status,storage_state) select $1::uuid,public.current_tenant_code(),public.current_institution_id(),$2,$3,$4,key.storage_key,$5,$6,'staged',$7,$8,'pending','staged' from key returning id::text,document_id::text,title,file_name,mime_type,storage_key,size_bytes,category,status,uploaded_by,to_char(uploaded_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`, documentID, req.Title, req.FileName, req.MimeType, req.SizeBytes, req.Category, authruntime.CurrentSubjectFromRequest(r), req.ChecksumSHA256).Scan(&item.ID, &item.DocumentID, &item.Title, &item.FileName, &item.MimeType, &item.StorageKey, &item.SizeBytes, &item.Category, &item.Status, &item.UploadedBy, &item.UploadedAt)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "attachment_stage_failed"})
		return
	}
	httpx.JSON(w, 201, item)
}

// EnrichDocumentParity loads the new, tenant-scoped registratura read model.
// It deliberately keeps legacy list contracts intact while clients migrate.
func (s *Service) EnrichDocumentParity(ctx context.Context, document *Document) error {
	if document == nil || strings.TrimSpace(document.ID) == "" {
		return nil
	}
	var externalDate, entryAt, exitAt, cancelledAt sql.NullString
	err := s.pool.QueryRow(ctx, `
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
	rows, err := s.pool.Query(ctx, `
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
	var departmentID, userID sql.NullString
	err = s.pool.QueryRow(ctx, `select department_id::text, assigned_user_id::text from registratura_document_workflow_events where document_id=$1::uuid and action in ('assign_department','assign_user','claim') order by created_at desc limit 1`, document.ID).Scan(&departmentID, &userID)
	if err == nil && (departmentID.Valid || userID.Valid) {
		document.WorkflowAssignment = &WorkflowAssignment{}
		if departmentID.Valid {
			document.WorkflowAssignment.DepartmentID = &departmentID.String
		}
		if userID.Valid {
			document.WorkflowAssignment.UserID = &userID.String
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
		return "ALOCAT_COMPARTIMENT", status == "ALOCAT_COMPARTIMENT"
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
	if _, err := s.loadDocument(r.Context(), documentID); err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		return
	}
	var req DocumentWorkflowActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_workflow_action"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	req.Note = strings.TrimSpace(req.Note)
	if req.Action == "claim" {
		var actorUserID string
		if err := s.pool.QueryRow(r.Context(), `select id::text from app_users where sub=$1`, authruntime.CurrentSubjectFromRequest(r)).Scan(&actorUserID); err != nil {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_claim_identity_missing"})
			return
		}
		if req.UserID != nil && strings.TrimSpace(*req.UserID) != "" && strings.TrimSpace(*req.UserID) != actorUserID {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_claim_must_be_self"})
			return
		}
		req.UserID = &actorUserID
	}
	if req.Action == "reject" && req.Note == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "workflow_note_required"})
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
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var status string
	var version int
	if err := tx.QueryRow(r.Context(), `select status, workflow_version from registratura_documents where id=$1::uuid for update`, documentID).Scan(&status, &version); err != nil {
		if err == pgx.ErrNoRows {
			httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "document_not_found"})
		} else {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		}
		return
	}
	if req.ExpectedVersion > 0 && req.ExpectedVersion != version {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "stale_workflow_version", "current_version": version})
		return
	}
	next, ok := workflowTransition(status, req.Action)
	if !ok {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "invalid_workflow_transition", "status": status})
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
		err = tx.QueryRow(r.Context(), `select exists(select 1 from app_memberships where user_id=$1::uuid and active)`, strings.TrimSpace(*req.UserID)).Scan(&exists)
		if err != nil || !exists {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "workflow_user_not_found"})
			return
		}
	}
	if req.Action == "assign_user" || req.Action == "claim" {
		var departmentID string
		err := tx.QueryRow(r.Context(), `select department_id::text from registratura_document_workflow_events where document_id=$1::uuid and department_id is not null order by created_at desc limit 1`, documentID).Scan(&departmentID)
		if err != nil {
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
		if err := tx.QueryRow(r.Context(), `select exists(select 1 from registratura_user_departments ud join app_memberships m on m.user_id=ud.user_id where ud.user_id=$1::uuid and ud.department_id=$2::uuid and m.active)`, userID, departmentID).Scan(&allowed); err != nil || !allowed {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "workflow_department_membership_required"})
			return
		}
	}
	actor := authruntime.CurrentSubjectFromRequest(r)
	_, err = tx.Exec(r.Context(), `insert into registratura_document_workflow_events(tenant_code,institution_id,document_id,action,from_status,to_status,department_id,assigned_user_id,note,actor_subject) values (public.current_tenant_code(),public.current_institution_id(),$1::uuid,$2,$3,$4,$5::uuid,$6::uuid,$7,$8)`, documentID, req.Action, status, next, req.DepartmentID, req.UserID, req.Note, actor)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	_, err = tx.Exec(r.Context(), `update registratura_documents set status=$2, workflow_version=workflow_version+1, updated_at=now() where id=$1::uuid`, documentID, next)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	doc, err := s.loadDocument(r.Context(), documentID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "workflow_action_failed"})
		return
	}
	_ = s.EnrichDocumentParity(r.Context(), &doc)
	s.logAudit(r, "registratura.documents.workflow."+req.Action, "document", documentID, "Document workflow transitioned.", map[string]any{"from": status, "to": next})
	httpx.JSON(w, http.StatusOK, doc)
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
