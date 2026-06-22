package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) ManagerialDocuments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"document_code":     {},
		"document_category": {},
		"title":             {},
		"document_status":   {},
		"version_label":     {},
		"owner_name":        {},
	}, []string{"document_code", "document_category", "title", "document_status", "version_label", "owner_name"})
	if query.Sort == "" {
		query.Sort = "document_code"
	}

	whereClause, args := buildManagerialDocumentFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_managerial_documents emd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_documents_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emd.id::text,
			emd.dossier_id::text,
			emd.document_code,
			emd.document_category,
			emd.title,
			emd.document_status,
			emd.version_label,
			emd.mandatory,
			emd.publication_required,
			to_char(emd.registered_on, 'YYYY-MM-DD'),
			coalesce(to_char(emd.approved_on, 'YYYY-MM-DD'), ''),
			emd.owner_name,
			emd.file_reference,
			emd.institution_id,
			emd.notes
		from education_managerial_documents emd
		%s
		order by %s %s, emd.document_code
		limit $%d offset $%d
	`, whereClause, managerialDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_documents_failed"})
		return
	}
	defer rows.Close()

	items := make([]ManagerialDocument, 0, query.PageSize)
	for rows.Next() {
		var item ManagerialDocument
		if err := rows.Scan(
			&item.ID,
			&item.DossierID,
			&item.DocumentCode,
			&item.DocumentCategory,
			&item.Title,
			&item.DocumentStatus,
			&item.VersionLabel,
			&item.Mandatory,
			&item.PublicationRequired,
			&item.RegisteredOn,
			&item.ApprovedOn,
			&item.OwnerName,
			&item.FileReference,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_documents_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) ManagerialDocumentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var item ManagerialDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			dossier_id::text,
			document_code,
			document_category,
			title,
			document_status,
			version_label,
			mandatory,
			publication_required,
			to_char(registered_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			owner_name,
			file_reference,
			institution_id,
			notes
		from education_managerial_documents
		where id = $1 and dossier_id = $2 and institution_id = $3
	`, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DossierID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.Title,
		&item.DocumentStatus,
		&item.VersionLabel,
		&item.Mandatory,
		&item.PublicationRequired,
		&item.RegisteredOn,
		&item.ApprovedOn,
		&item.OwnerName,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_document_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateManagerialDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateManagerialDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_payload"})
		return
	}
	normalizeManagerialDocumentRequest(&req)
	if req.DocumentCategory == "" || req.Title == "" || req.DocumentStatus == "" || req.VersionLabel == "" || req.RegisteredOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_document_fields"})
		return
	}
	if !inStringSet(req.DocumentCategory, "diagnoza", "prognoza", "evidenta", "planificare", "raport", "anexa", "hotarare", "procedura") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_category"})
		return
	}
	if !inStringSet(req.DocumentStatus, "draft", "in_review", "approved", "published", "archived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_status"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_registered_on"})
		return
	}
	approvedOn, err := parseOptionalEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_approved_on"})
		return
	}
	if inStringSet(req.DocumentStatus, "approved", "published", "archived") && approvedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_document_approved_on"})
		return
	}

	documentCode := fmt.Sprintf("MDOC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item ManagerialDocument
	err = s.pool.QueryRow(r.Context(), `
		insert into education_managerial_documents (
			dossier_id,
			document_code,
			document_category,
			title,
			document_status,
			version_label,
			mandatory,
			publication_required,
			registered_on,
			approved_on,
			owner_name,
			file_reference,
			institution_id,
			notes
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		returning
			id::text,
			dossier_id::text,
			document_code,
			document_category,
			title,
			document_status,
			version_label,
			mandatory,
			publication_required,
			to_char(registered_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			owner_name,
			file_reference,
			institution_id,
			notes
	`, recordID, documentCode, req.DocumentCategory, req.Title, req.DocumentStatus, req.VersionLabel, req.Mandatory, req.PublicationRequired, registeredOn, approvedOn, req.OwnerName, req.FileReference, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.DossierID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.Title,
		&item.DocumentStatus,
		&item.VersionLabel,
		&item.Mandatory,
		&item.PublicationRequired,
		&item.RegisteredOn,
		&item.ApprovedOn,
		&item.OwnerName,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_document_create_failed"})
		return
	}

	s.logAudit(r, "education.managerial.documents.create", "managerial_document", item.ID, "Managerial document created.", map[string]any{
		"dossier_id":      recordID,
		"document_code":   item.DocumentCode,
		"document_status": item.DocumentStatus,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateManagerialDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	var req CreateManagerialDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_payload"})
		return
	}
	normalizeManagerialDocumentRequest(&req)
	if req.DocumentCategory == "" || req.Title == "" || req.DocumentStatus == "" || req.VersionLabel == "" || req.RegisteredOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_document_fields"})
		return
	}
	if !inStringSet(req.DocumentCategory, "diagnoza", "prognoza", "evidenta", "planificare", "raport", "anexa", "hotarare", "procedura") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_category"})
		return
	}
	if !inStringSet(req.DocumentStatus, "draft", "in_review", "approved", "published", "archived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_status"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_registered_on"})
		return
	}
	approvedOn, err := parseOptionalEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_document_approved_on"})
		return
	}
	if inStringSet(req.DocumentStatus, "approved", "published", "archived") && approvedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_document_approved_on"})
		return
	}

	var item ManagerialDocument
	err = s.pool.QueryRow(r.Context(), `
		update education_managerial_documents
		set
			document_category = $1,
			title = $2,
			document_status = $3,
			version_label = $4,
			mandatory = $5,
			publication_required = $6,
			registered_on = $7,
			approved_on = $8,
			owner_name = $9,
			file_reference = $10,
			notes = $11,
			updated_at = now()
		where id = $12 and dossier_id = $13 and institution_id = $14
		returning
			id::text,
			dossier_id::text,
			document_code,
			document_category,
			title,
			document_status,
			version_label,
			mandatory,
			publication_required,
			to_char(registered_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			owner_name,
			file_reference,
			institution_id,
			notes
	`, req.DocumentCategory, req.Title, req.DocumentStatus, req.VersionLabel, req.Mandatory, req.PublicationRequired, registeredOn, approvedOn, req.OwnerName, req.FileReference, req.Notes, documentID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DossierID,
		&item.DocumentCode,
		&item.DocumentCategory,
		&item.Title,
		&item.DocumentStatus,
		&item.VersionLabel,
		&item.Mandatory,
		&item.PublicationRequired,
		&item.RegisteredOn,
		&item.ApprovedOn,
		&item.OwnerName,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_document_update_failed"})
		return
	}

	s.logAudit(r, "education.managerial.documents.update", "managerial_document", item.ID, "Managerial document updated.", map[string]any{
		"dossier_id":      recordID,
		"document_code":   item.DocumentCode,
		"document_status": item.DocumentStatus,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteManagerialDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_managerial_documents where id = $1 and dossier_id = $2 and institution_id = $3`, documentID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_managerial_document_not_found")
		return
	}

	s.logAudit(r, "education.managerial.documents.delete", "managerial_document", documentID, "Managerial document deleted.", map[string]any{"dossier_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) ManagerialWorkflowSteps(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"stage_type":         {},
		"status":             {},
		"assigned_to":        {},
		"decision_reference": {},
	}, []string{"stage_order", "stage_type", "status", "assigned_to", "due_on", "completed_on"})
	if query.Sort == "" {
		query.Sort = "stage_order"
	}

	whereClause, args := buildManagerialWorkflowFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_managerial_workflow_steps emw "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_workflow_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emw.id::text,
			emw.dossier_id::text,
			emw.stage_order,
			emw.stage_type,
			emw.status,
			emw.assigned_to,
			to_char(emw.due_on, 'YYYY-MM-DD'),
			coalesce(to_char(emw.completed_on, 'YYYY-MM-DD'), ''),
			emw.requires_signature,
			emw.decision_reference,
			emw.institution_id,
			emw.outcome_note
		from education_managerial_workflow_steps emw
		%s
		order by %s %s, emw.stage_order
		limit $%d offset $%d
	`, whereClause, managerialWorkflowSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_workflow_failed"})
		return
	}
	defer rows.Close()

	items := make([]ManagerialWorkflowStep, 0, query.PageSize)
	for rows.Next() {
		var item ManagerialWorkflowStep
		if err := rows.Scan(
			&item.ID,
			&item.DossierID,
			&item.StageOrder,
			&item.StageType,
			&item.Status,
			&item.AssignedTo,
			&item.DueOn,
			&item.CompletedOn,
			&item.RequiresSignature,
			&item.DecisionReference,
			&item.InstitutionID,
			&item.OutcomeNote,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_workflow_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_workflow_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) ManagerialWorkflowStepDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	var item ManagerialWorkflowStep
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			dossier_id::text,
			stage_order,
			stage_type,
			status,
			assigned_to,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			requires_signature,
			decision_reference,
			institution_id,
			outcome_note
		from education_managerial_workflow_steps
		where id = $1 and dossier_id = $2 and institution_id = $3
	`, stepID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DossierID,
		&item.StageOrder,
		&item.StageType,
		&item.Status,
		&item.AssignedTo,
		&item.DueOn,
		&item.CompletedOn,
		&item.RequiresSignature,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.OutcomeNote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_workflow_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_workflow_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateManagerialWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateManagerialWorkflowStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_payload"})
		return
	}
	normalizeManagerialWorkflowRequest(&req)
	if req.StageOrder <= 0 || req.StageType == "" || req.Status == "" || req.AssignedTo == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_workflow_fields"})
		return
	}
	if !inStringSet(req.StageType, "elaborare", "verificare_secretariat", "avizare_cp", "aprobare_ca", "publicare", "arhivare") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_stage"})
		return
	}
	if !inStringSet(req.Status, "pending", "in_progress", "completed", "returned", "waived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_status"})
		return
	}
	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_completed_on"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_workflow_completed_on"})
		return
	}

	var item ManagerialWorkflowStep
	err = s.pool.QueryRow(r.Context(), `
		insert into education_managerial_workflow_steps (
			dossier_id,
			stage_order,
			stage_type,
			status,
			assigned_to,
			due_on,
			completed_on,
			requires_signature,
			decision_reference,
			institution_id,
			outcome_note
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		returning
			id::text,
			dossier_id::text,
			stage_order,
			stage_type,
			status,
			assigned_to,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			requires_signature,
			decision_reference,
			institution_id,
			outcome_note
	`, recordID, req.StageOrder, req.StageType, req.Status, req.AssignedTo, dueOn, completedOn, req.RequiresSignature, req.DecisionReference, s.institutionID(r), req.OutcomeNote).Scan(
		&item.ID,
		&item.DossierID,
		&item.StageOrder,
		&item.StageType,
		&item.Status,
		&item.AssignedTo,
		&item.DueOn,
		&item.CompletedOn,
		&item.RequiresSignature,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.OutcomeNote,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_workflow_create_failed"})
		return
	}

	s.logAudit(r, "education.managerial.workflow.create", "managerial_workflow_step", item.ID, "Managerial workflow step created.", map[string]any{
		"dossier_id":  recordID,
		"stage_order": item.StageOrder,
		"stage_type":  item.StageType,
		"status":      item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateManagerialWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	var req CreateManagerialWorkflowStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_payload"})
		return
	}
	normalizeManagerialWorkflowRequest(&req)
	if req.StageOrder <= 0 || req.StageType == "" || req.Status == "" || req.AssignedTo == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_workflow_fields"})
		return
	}
	if !inStringSet(req.StageType, "elaborare", "verificare_secretariat", "avizare_cp", "aprobare_ca", "publicare", "arhivare") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_stage"})
		return
	}
	if !inStringSet(req.Status, "pending", "in_progress", "completed", "returned", "waived") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_status"})
		return
	}
	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_workflow_completed_on"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_workflow_completed_on"})
		return
	}

	var item ManagerialWorkflowStep
	err = s.pool.QueryRow(r.Context(), `
		update education_managerial_workflow_steps
		set
			stage_order = $1,
			stage_type = $2,
			status = $3,
			assigned_to = $4,
			due_on = $5,
			completed_on = $6,
			requires_signature = $7,
			decision_reference = $8,
			outcome_note = $9,
			updated_at = now()
		where id = $10 and dossier_id = $11 and institution_id = $12
		returning
			id::text,
			dossier_id::text,
			stage_order,
			stage_type,
			status,
			assigned_to,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			requires_signature,
			decision_reference,
			institution_id,
			outcome_note
	`, req.StageOrder, req.StageType, req.Status, req.AssignedTo, dueOn, completedOn, req.RequiresSignature, req.DecisionReference, req.OutcomeNote, stepID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DossierID,
		&item.StageOrder,
		&item.StageType,
		&item.Status,
		&item.AssignedTo,
		&item.DueOn,
		&item.CompletedOn,
		&item.RequiresSignature,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.OutcomeNote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_workflow_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_workflow_update_failed"})
		return
	}

	s.logAudit(r, "education.managerial.workflow.update", "managerial_workflow_step", item.ID, "Managerial workflow step updated.", map[string]any{
		"dossier_id":  recordID,
		"stage_order": item.StageOrder,
		"stage_type":  item.StageType,
		"status":      item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteManagerialWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_managerial_workflow_steps where id = $1 and dossier_id = $2 and institution_id = $3`, stepID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_workflow_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_managerial_workflow_step_not_found")
		return
	}

	s.logAudit(r, "education.managerial.workflow.delete", "managerial_workflow_step", stepID, "Managerial workflow step deleted.", map[string]any{"dossier_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) RegulationVersions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"version_label":  {},
		"version_status": {},
		"prepared_by":    {},
		"file_reference": {},
	}, []string{"version_label", "version_status", "prepared_by", "approved_on", "effective_from", "published_on"})
	if query.Sort == "" {
		query.Sort = "effective_from"
	}

	whereClause, args := buildRegulationVersionFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_regulation_versions erv "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_versions_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			erv.id::text,
			erv.regulation_id::text,
			erv.version_label,
			erv.version_status,
			erv.change_summary,
			coalesce(to_char(erv.approved_on, 'YYYY-MM-DD'), ''),
			to_char(erv.effective_from, 'YYYY-MM-DD'),
			coalesce(to_char(erv.published_on, 'YYYY-MM-DD'), ''),
			erv.prepared_by,
			erv.file_reference,
			erv.institution_id,
			erv.notes
		from education_regulation_versions erv
		%s
		order by %s %s, erv.version_label
		limit $%d offset $%d
	`, whereClause, regulationVersionSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_versions_failed"})
		return
	}
	defer rows.Close()

	items := make([]RegulationVersion, 0, query.PageSize)
	for rows.Next() {
		var item RegulationVersion
		if err := rows.Scan(
			&item.ID,
			&item.RegulationID,
			&item.VersionLabel,
			&item.VersionStatus,
			&item.ChangeSummary,
			&item.ApprovedOn,
			&item.EffectiveFrom,
			&item.PublishedOn,
			&item.PreparedBy,
			&item.FileReference,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_versions_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_versions_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RegulationVersionDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	versionID := strings.TrimSpace(chi.URLParam(r, "versionID"))
	var item RegulationVersion
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			regulation_id::text,
			version_label,
			version_status,
			change_summary,
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			to_char(effective_from, 'YYYY-MM-DD'),
			coalesce(to_char(published_on, 'YYYY-MM-DD'), ''),
			prepared_by,
			file_reference,
			institution_id,
			notes
		from education_regulation_versions
		where id = $1 and regulation_id = $2 and institution_id = $3
	`, versionID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.RegulationID,
		&item.VersionLabel,
		&item.VersionStatus,
		&item.ChangeSummary,
		&item.ApprovedOn,
		&item.EffectiveFrom,
		&item.PublishedOn,
		&item.PreparedBy,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_version_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_version_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateRegulationVersion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateRegulationVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_payload"})
		return
	}
	normalizeRegulationVersionRequest(&req)
	if req.VersionLabel == "" || req.VersionStatus == "" || req.ChangeSummary == "" || req.EffectiveFrom == "" || req.PreparedBy == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_version_fields"})
		return
	}
	if !inStringSet(req.VersionStatus, "draft", "consultation", "endorsed", "approved", "published", "retired") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_status"})
		return
	}
	approvedOn, err := parseOptionalEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_effective_from"})
		return
	}
	publishedOn, err := parseOptionalEducationDate(req.PublishedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_published_on"})
		return
	}
	if req.VersionStatus == "published" && publishedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_version_published_on"})
		return
	}

	var item RegulationVersion
	err = s.pool.QueryRow(r.Context(), `
		insert into education_regulation_versions (
			regulation_id,
			version_label,
			version_status,
			change_summary,
			approved_on,
			effective_from,
			published_on,
			prepared_by,
			file_reference,
			institution_id,
			notes
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		returning
			id::text,
			regulation_id::text,
			version_label,
			version_status,
			change_summary,
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			to_char(effective_from, 'YYYY-MM-DD'),
			coalesce(to_char(published_on, 'YYYY-MM-DD'), ''),
			prepared_by,
			file_reference,
			institution_id,
			notes
	`, recordID, req.VersionLabel, req.VersionStatus, req.ChangeSummary, approvedOn, effectiveFrom, publishedOn, req.PreparedBy, req.FileReference, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.RegulationID,
		&item.VersionLabel,
		&item.VersionStatus,
		&item.ChangeSummary,
		&item.ApprovedOn,
		&item.EffectiveFrom,
		&item.PublishedOn,
		&item.PreparedBy,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_version_create_failed"})
		return
	}

	s.logAudit(r, "education.regulations.versions.create", "regulation_version", item.ID, "Regulation version created.", map[string]any{
		"regulation_id":  recordID,
		"version_label":  item.VersionLabel,
		"version_status": item.VersionStatus,
		"effective_from": item.EffectiveFrom,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateRegulationVersion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	versionID := strings.TrimSpace(chi.URLParam(r, "versionID"))
	var req CreateRegulationVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_payload"})
		return
	}
	normalizeRegulationVersionRequest(&req)
	if req.VersionLabel == "" || req.VersionStatus == "" || req.ChangeSummary == "" || req.EffectiveFrom == "" || req.PreparedBy == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_version_fields"})
		return
	}
	if !inStringSet(req.VersionStatus, "draft", "consultation", "endorsed", "approved", "published", "retired") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_status"})
		return
	}
	approvedOn, err := parseOptionalEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_effective_from"})
		return
	}
	publishedOn, err := parseOptionalEducationDate(req.PublishedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_version_published_on"})
		return
	}
	if req.VersionStatus == "published" && publishedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_version_published_on"})
		return
	}

	var item RegulationVersion
	err = s.pool.QueryRow(r.Context(), `
		update education_regulation_versions
		set
			version_label = $1,
			version_status = $2,
			change_summary = $3,
			approved_on = $4,
			effective_from = $5,
			published_on = $6,
			prepared_by = $7,
			file_reference = $8,
			notes = $9,
			updated_at = now()
		where id = $10 and regulation_id = $11 and institution_id = $12
		returning
			id::text,
			regulation_id::text,
			version_label,
			version_status,
			change_summary,
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			to_char(effective_from, 'YYYY-MM-DD'),
			coalesce(to_char(published_on, 'YYYY-MM-DD'), ''),
			prepared_by,
			file_reference,
			institution_id,
			notes
	`, req.VersionLabel, req.VersionStatus, req.ChangeSummary, approvedOn, effectiveFrom, publishedOn, req.PreparedBy, req.FileReference, req.Notes, versionID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.RegulationID,
		&item.VersionLabel,
		&item.VersionStatus,
		&item.ChangeSummary,
		&item.ApprovedOn,
		&item.EffectiveFrom,
		&item.PublishedOn,
		&item.PreparedBy,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_version_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_version_update_failed"})
		return
	}

	s.logAudit(r, "education.regulations.versions.update", "regulation_version", item.ID, "Regulation version updated.", map[string]any{
		"regulation_id":  recordID,
		"version_label":  item.VersionLabel,
		"version_status": item.VersionStatus,
		"effective_from": item.EffectiveFrom,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteRegulationVersion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	versionID := strings.TrimSpace(chi.URLParam(r, "versionID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_regulation_versions where id = $1 and regulation_id = $2 and institution_id = $3`, versionID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_version_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_regulation_version_not_found")
		return
	}

	s.logAudit(r, "education.regulations.versions.delete", "regulation_version", versionID, "Regulation version deleted.", map[string]any{"regulation_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) RegulationWorkflowSteps(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"phase_type":         {},
		"status":             {},
		"audience":           {},
		"decision_reference": {},
	}, []string{"phase_order", "phase_type", "status", "audience", "started_on", "due_on", "completed_on", "feedback_count"})
	if query.Sort == "" {
		query.Sort = "phase_order"
	}

	whereClause, args := buildRegulationWorkflowFilters(query.Filters, recordID, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_regulation_workflow_steps erw "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_workflow_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			erw.id::text,
			erw.regulation_id::text,
			erw.phase_order,
			erw.phase_type,
			erw.status,
			erw.audience,
			to_char(erw.started_on, 'YYYY-MM-DD'),
			to_char(erw.due_on, 'YYYY-MM-DD'),
			coalesce(to_char(erw.completed_on, 'YYYY-MM-DD'), ''),
			erw.feedback_count,
			erw.decision_reference,
			erw.institution_id,
			erw.notes
		from education_regulation_workflow_steps erw
		%s
		order by %s %s, erw.phase_order
		limit $%d offset $%d
	`, whereClause, regulationWorkflowSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_workflow_failed"})
		return
	}
	defer rows.Close()

	items := make([]RegulationWorkflowStep, 0, query.PageSize)
	for rows.Next() {
		var item RegulationWorkflowStep
		if err := rows.Scan(
			&item.ID,
			&item.RegulationID,
			&item.PhaseOrder,
			&item.PhaseType,
			&item.Status,
			&item.Audience,
			&item.StartedOn,
			&item.DueOn,
			&item.CompletedOn,
			&item.FeedbackCount,
			&item.DecisionReference,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_workflow_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_workflow_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RegulationWorkflowStepDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	var item RegulationWorkflowStep
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			regulation_id::text,
			phase_order,
			phase_type,
			status,
			audience,
			to_char(started_on, 'YYYY-MM-DD'),
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			feedback_count,
			decision_reference,
			institution_id,
			notes
		from education_regulation_workflow_steps
		where id = $1 and regulation_id = $2 and institution_id = $3
	`, stepID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.RegulationID,
		&item.PhaseOrder,
		&item.PhaseType,
		&item.Status,
		&item.Audience,
		&item.StartedOn,
		&item.DueOn,
		&item.CompletedOn,
		&item.FeedbackCount,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_workflow_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_workflow_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateRegulationWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateRegulationWorkflowStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_payload"})
		return
	}
	normalizeRegulationWorkflowRequest(&req)
	if req.PhaseOrder <= 0 || req.PhaseType == "" || req.Status == "" || req.Audience == "" || req.StartedOn == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_workflow_fields"})
		return
	}
	if req.FeedbackCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_feedback_count"})
		return
	}
	if !inStringSet(req.PhaseType, "redactare", "consultare_publica", "avizare_cp", "aprobare_ca", "inregistrare", "publicare") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_phase"})
		return
	}
	if !inStringSet(req.Status, "pending", "active", "completed", "returned", "cancelled") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_status"})
		return
	}
	startedOn, err := parseRequiredEducationDate(req.StartedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_started_on"})
		return
	}
	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_completed_on"})
		return
	}
	if completedOn != nil && completedOn.Before(*startedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_interval"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_workflow_completed_on"})
		return
	}

	var item RegulationWorkflowStep
	err = s.pool.QueryRow(r.Context(), `
		insert into education_regulation_workflow_steps (
			regulation_id,
			phase_order,
			phase_type,
			status,
			audience,
			started_on,
			due_on,
			completed_on,
			feedback_count,
			decision_reference,
			institution_id,
			notes
		) values (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		returning
			id::text,
			regulation_id::text,
			phase_order,
			phase_type,
			status,
			audience,
			to_char(started_on, 'YYYY-MM-DD'),
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			feedback_count,
			decision_reference,
			institution_id,
			notes
	`, recordID, req.PhaseOrder, req.PhaseType, req.Status, req.Audience, startedOn, dueOn, completedOn, req.FeedbackCount, req.DecisionReference, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.RegulationID,
		&item.PhaseOrder,
		&item.PhaseType,
		&item.Status,
		&item.Audience,
		&item.StartedOn,
		&item.DueOn,
		&item.CompletedOn,
		&item.FeedbackCount,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_workflow_create_failed"})
		return
	}

	s.logAudit(r, "education.regulations.workflow.create", "regulation_workflow_step", item.ID, "Regulation workflow step created.", map[string]any{
		"regulation_id": recordID,
		"phase_order":   item.PhaseOrder,
		"phase_type":    item.PhaseType,
		"status":        item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateRegulationWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	var req CreateRegulationWorkflowStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_payload"})
		return
	}
	normalizeRegulationWorkflowRequest(&req)
	if req.PhaseOrder <= 0 || req.PhaseType == "" || req.Status == "" || req.Audience == "" || req.StartedOn == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_workflow_fields"})
		return
	}
	if req.FeedbackCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_feedback_count"})
		return
	}
	if !inStringSet(req.PhaseType, "redactare", "consultare_publica", "avizare_cp", "aprobare_ca", "inregistrare", "publicare") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_phase"})
		return
	}
	if !inStringSet(req.Status, "pending", "active", "completed", "returned", "cancelled") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_status"})
		return
	}
	startedOn, err := parseRequiredEducationDate(req.StartedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_started_on"})
		return
	}
	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_completed_on"})
		return
	}
	if completedOn != nil && completedOn.Before(*startedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_workflow_interval"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_workflow_completed_on"})
		return
	}

	var item RegulationWorkflowStep
	err = s.pool.QueryRow(r.Context(), `
		update education_regulation_workflow_steps
		set
			phase_order = $1,
			phase_type = $2,
			status = $3,
			audience = $4,
			started_on = $5,
			due_on = $6,
			completed_on = $7,
			feedback_count = $8,
			decision_reference = $9,
			notes = $10,
			updated_at = now()
		where id = $11 and regulation_id = $12 and institution_id = $13
		returning
			id::text,
			regulation_id::text,
			phase_order,
			phase_type,
			status,
			audience,
			to_char(started_on, 'YYYY-MM-DD'),
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			feedback_count,
			decision_reference,
			institution_id,
			notes
	`, req.PhaseOrder, req.PhaseType, req.Status, req.Audience, startedOn, dueOn, completedOn, req.FeedbackCount, req.DecisionReference, req.Notes, stepID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.RegulationID,
		&item.PhaseOrder,
		&item.PhaseType,
		&item.Status,
		&item.Audience,
		&item.StartedOn,
		&item.DueOn,
		&item.CompletedOn,
		&item.FeedbackCount,
		&item.DecisionReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_workflow_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_workflow_update_failed"})
		return
	}

	s.logAudit(r, "education.regulations.workflow.update", "regulation_workflow_step", item.ID, "Regulation workflow step updated.", map[string]any{
		"regulation_id": recordID,
		"phase_order":   item.PhaseOrder,
		"phase_type":    item.PhaseType,
		"status":        item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteRegulationWorkflowStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	stepID := strings.TrimSpace(chi.URLParam(r, "stepID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_regulation_workflow_steps where id = $1 and regulation_id = $2 and institution_id = $3`, stepID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_workflow_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_regulation_workflow_step_not_found")
		return
	}

	s.logAudit(r, "education.regulations.workflow.delete", "regulation_workflow_step", stepID, "Regulation workflow step deleted.", map[string]any{"regulation_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func buildManagerialDocumentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emd.dossier_id = $1", "emd.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"document_code":     "emd.document_code",
		"document_category": "emd.document_category",
		"title":             "emd.title",
		"document_status":   "emd.document_status",
		"version_label":     "emd.version_label",
		"owner_name":        "emd.owner_name",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildManagerialWorkflowFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emw.dossier_id = $1", "emw.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"stage_type":         "emw.stage_type",
		"status":             "emw.status",
		"assigned_to":        "emw.assigned_to",
		"decision_reference": "emw.decision_reference",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildRegulationVersionFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"erv.regulation_id = $1", "erv.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"version_label":  "erv.version_label",
		"version_status": "erv.version_status",
		"prepared_by":    "erv.prepared_by",
		"file_reference": "erv.file_reference",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildRegulationWorkflowFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"erw.regulation_id = $1", "erw.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"phase_type":         "erw.phase_type",
		"status":             "erw.status",
		"audience":           "erw.audience",
		"decision_reference": "erw.decision_reference",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func managerialDocumentSortColumn(value string) string {
	switch value {
	case "document_code":
		return "emd.document_code"
	case "document_category":
		return "emd.document_category"
	case "title":
		return "emd.title"
	case "document_status":
		return "emd.document_status"
	case "version_label":
		return "emd.version_label"
	case "owner_name":
		return "emd.owner_name"
	case "registered_on":
		return "emd.registered_on"
	case "approved_on":
		return "emd.approved_on"
	default:
		return "emd.document_code"
	}
}

func managerialWorkflowSortColumn(value string) string {
	switch value {
	case "stage_order":
		return "emw.stage_order"
	case "stage_type":
		return "emw.stage_type"
	case "status":
		return "emw.status"
	case "assigned_to":
		return "emw.assigned_to"
	case "due_on":
		return "emw.due_on"
	case "completed_on":
		return "emw.completed_on"
	default:
		return "emw.stage_order"
	}
}

func regulationVersionSortColumn(value string) string {
	switch value {
	case "version_label":
		return "erv.version_label"
	case "version_status":
		return "erv.version_status"
	case "prepared_by":
		return "erv.prepared_by"
	case "approved_on":
		return "erv.approved_on"
	case "effective_from":
		return "erv.effective_from"
	case "published_on":
		return "erv.published_on"
	default:
		return "erv.effective_from"
	}
}

func regulationWorkflowSortColumn(value string) string {
	switch value {
	case "phase_order":
		return "erw.phase_order"
	case "phase_type":
		return "erw.phase_type"
	case "status":
		return "erw.status"
	case "audience":
		return "erw.audience"
	case "started_on":
		return "erw.started_on"
	case "due_on":
		return "erw.due_on"
	case "completed_on":
		return "erw.completed_on"
	case "feedback_count":
		return "erw.feedback_count"
	default:
		return "erw.phase_order"
	}
}

func normalizeManagerialDocumentRequest(req *CreateManagerialDocumentRequest) {
	req.DocumentCategory = strings.TrimSpace(req.DocumentCategory)
	req.Title = strings.TrimSpace(req.Title)
	req.DocumentStatus = strings.TrimSpace(req.DocumentStatus)
	req.VersionLabel = strings.TrimSpace(req.VersionLabel)
	req.RegisteredOn = strings.TrimSpace(req.RegisteredOn)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.FileReference = strings.TrimSpace(req.FileReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeManagerialWorkflowRequest(req *CreateManagerialWorkflowStepRequest) {
	req.StageType = strings.TrimSpace(req.StageType)
	req.Status = strings.TrimSpace(req.Status)
	req.AssignedTo = strings.TrimSpace(req.AssignedTo)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.CompletedOn = strings.TrimSpace(req.CompletedOn)
	req.DecisionReference = strings.TrimSpace(req.DecisionReference)
	req.OutcomeNote = strings.TrimSpace(req.OutcomeNote)
}

func normalizeRegulationVersionRequest(req *CreateRegulationVersionRequest) {
	req.VersionLabel = strings.TrimSpace(req.VersionLabel)
	req.VersionStatus = strings.TrimSpace(req.VersionStatus)
	req.ChangeSummary = strings.TrimSpace(req.ChangeSummary)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.EffectiveFrom = strings.TrimSpace(req.EffectiveFrom)
	req.PublishedOn = strings.TrimSpace(req.PublishedOn)
	req.PreparedBy = strings.TrimSpace(req.PreparedBy)
	req.FileReference = strings.TrimSpace(req.FileReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeRegulationWorkflowRequest(req *CreateRegulationWorkflowStepRequest) {
	req.PhaseType = strings.TrimSpace(req.PhaseType)
	req.Status = strings.TrimSpace(req.Status)
	req.Audience = strings.TrimSpace(req.Audience)
	req.StartedOn = strings.TrimSpace(req.StartedOn)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.CompletedOn = strings.TrimSpace(req.CompletedOn)
	req.DecisionReference = strings.TrimSpace(req.DecisionReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func parseRequiredEducationDate(value string) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalEducationDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func inStringSet(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
