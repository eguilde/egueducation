package education

import (
	"context"
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

func evaluationQualification(score float64) string {
	switch {
	case score >= 90:
		return "foarte_bine"
	case score >= 75:
		return "bine"
	case score >= 60:
		return "satisfacator"
	default:
		return "nesatisfacator"
	}
}

func personnelEvaluationStatusFromEvaluation(status string) string {
	switch status {
	case "approved":
		return "finalized"
	case "draft":
		return "draft"
	default:
		return "in_review"
	}
}

func (s *Service) syncEvaluationPersonnelStateByID(ctx context.Context, evaluationID string, institutionID string) error {
	evaluation, err := s.loadEvaluationFileContext(ctx, evaluationID, institutionID)
	if err != nil {
		return err
	}
	if evaluation == nil {
		return nil
	}
	return s.syncEvaluationPersonnelStatus(ctx, evaluation, institutionID)
}

func (s *Service) syncEvaluationPersonnelStatus(ctx context.Context, evaluation *evaluationFileContext, institutionID string) error {
	if evaluation.PersonnelID == "" {
		return nil
	}

	if _, err := s.pool.Exec(ctx, `
		update education_personnel
		set evaluation_status = $1, updated_at = now()
		where id = $2::uuid and institution_id = $3
	`, personnelEvaluationStatusFromEvaluation(evaluation.Status), evaluation.PersonnelID, institutionID); err != nil {
		return fmt.Errorf("update personnel evaluation status: %w", err)
	}

	return nil
}

func (s *Service) EvaluationResultIssues(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"issue_code":                 {},
		"document_type":              {},
		"recipient_name":             {},
		"recipient_role":             {},
		"delivery_channel":           {},
		"delivery_status":            {},
		"issued_on":                  {},
		"delivered_on":               {},
		"acknowledged_on":            {},
		"attached_to_personnel_file": {},
	}, []string{"issue_code", "document_type", "recipient_name", "recipient_role", "delivery_channel", "delivery_status", "issued_on", "delivered_on", "acknowledged_on"})
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildEvaluationResultIssueFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_evaluation_result_issues eeri "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issues_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			eeri.id::text,
			eeri.evaluation_id::text,
			eeri.issue_code,
			eeri.document_type,
			eeri.recipient_name,
			eeri.recipient_role,
			eeri.delivery_channel,
			eeri.delivery_status,
			to_char(eeri.issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(eeri.delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(eeri.acknowledged_on, 'YYYY-MM-DD'), ''),
			eeri.registry_reference,
			eeri.attached_to_personnel_file,
			eeri.institution_id,
			eeri.notes
		from education_evaluation_result_issues eeri
		%s
		order by %s %s, eeri.issue_code asc
		limit $%d offset $%d
	`, whereClause, evaluationResultIssueSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issues_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelEvaluationResultIssue, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelEvaluationResultIssue
		if err := rows.Scan(
			&item.ID,
			&item.EvaluationID,
			&item.IssueCode,
			&item.DocumentType,
			&item.RecipientName,
			&item.RecipientRole,
			&item.DeliveryChannel,
			&item.DeliveryStatus,
			&item.IssuedOn,
			&item.DeliveredOn,
			&item.AcknowledgedOn,
			&item.RegistryReference,
			&item.AttachedToPersonnelFile,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issues_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issues_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EvaluationResultIssueDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	var item PersonnelEvaluationResultIssue
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			evaluation_id::text,
			issue_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			registry_reference,
			attached_to_personnel_file,
			institution_id,
			notes
		from education_evaluation_result_issues
		where id = $1 and evaluation_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.IssueCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.IssuedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.RegistryReference,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issue_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateEvaluationResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelEvaluationResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_payload"})
		return
	}
	normalizeEvaluationResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "fisa_evaluare", "comunicare", "decizie", "raport_final") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_fields"})
		return
	}

	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_delivered_on"})
		return
	}
	acknowledgedOn, err := parseOptionalEducationDate(req.AcknowledgedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_acknowledged_on"})
		return
	}
	if deliveredOn != nil && deliveredOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_delivery_interval"})
		return
	}
	if acknowledgedOn != nil && acknowledgedOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_acknowledged_interval"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_delivered_on"})
		return
	}
	if req.DeliveryStatus == "confirmat" && acknowledgedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_acknowledged_on"})
		return
	}

	issueCode := fmt.Sprintf("EVRES-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelEvaluationResultIssue
	err = s.pool.QueryRow(r.Context(), `
		insert into education_evaluation_result_issues (
			evaluation_id,
			issue_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			issued_on,
			delivered_on,
			acknowledged_on,
			registry_reference,
			attached_to_personnel_file,
			institution_id,
			notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		returning
			id::text,
			evaluation_id::text,
			issue_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			registry_reference,
			attached_to_personnel_file,
			institution_id,
			notes
	`, recordID, issueCode, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, acknowledgedOn, req.RegistryReference, req.AttachedToPersonnelFile, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.IssueCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.IssuedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.RegistryReference,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_create_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.result_issue.create", "evaluation_result_issue", item.ID, "Evaluation result issue created.", map[string]any{
		"evaluation_id":              recordID,
		"issue_code":                 item.IssueCode,
		"document_type":              item.DocumentType,
		"delivery_status":            item.DeliveryStatus,
		"attached_to_personnel_file": item.AttachedToPersonnelFile,
	})
	if err := s.syncEvaluationResultIssueEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateEvaluationResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePersonnelEvaluationResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_payload"})
		return
	}
	normalizeEvaluationResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "fisa_evaluare", "comunicare", "decizie", "raport_final") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_fields"})
		return
	}

	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_delivered_on"})
		return
	}
	acknowledgedOn, err := parseOptionalEducationDate(req.AcknowledgedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_acknowledged_on"})
		return
	}
	if deliveredOn != nil && deliveredOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_delivery_interval"})
		return
	}
	if acknowledgedOn != nil && acknowledgedOn.Before(*issuedOn) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_result_issue_acknowledged_interval"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_delivered_on"})
		return
	}
	if req.DeliveryStatus == "confirmat" && acknowledgedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_result_issue_acknowledged_on"})
		return
	}

	var item PersonnelEvaluationResultIssue
	err = s.pool.QueryRow(r.Context(), `
		update education_evaluation_result_issues
		set
			document_type = $1,
			recipient_name = $2,
			recipient_role = $3,
			delivery_channel = $4,
			delivery_status = $5,
			issued_on = $6,
			delivered_on = $7,
			acknowledged_on = $8,
			registry_reference = $9,
			attached_to_personnel_file = $10,
			notes = $11,
			updated_at = now()
		where id = $12 and evaluation_id = $13 and institution_id = $14
		returning
			id::text,
			evaluation_id::text,
			issue_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			registry_reference,
			attached_to_personnel_file,
			institution_id,
			notes
	`, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, acknowledgedOn, req.RegistryReference, req.AttachedToPersonnelFile, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationID,
		&item.IssueCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.IssuedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.RegistryReference,
		&item.AttachedToPersonnelFile,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_update_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.result_issue.update", "evaluation_result_issue", item.ID, "Evaluation result issue updated.", map[string]any{
		"evaluation_id":              recordID,
		"issue_code":                 item.IssueCode,
		"document_type":              item.DocumentType,
		"delivery_status":            item.DeliveryStatus,
		"attached_to_personnel_file": item.AttachedToPersonnelFile,
	})
	if err := s.syncEvaluationResultIssueEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteEvaluationResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	tag, err := s.pool.Exec(r.Context(), `
		delete from education_evaluation_result_issues
		where id = $1 and evaluation_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_evaluation_result_issue_not_found")
		return
	}

	s.logAudit(r, "education.evaluations.result_issue.delete", "evaluation_result_issue", itemID, "Evaluation result issue deleted.", map[string]any{"evaluation_id": recordID})
	if err := s.syncEvaluationResultIssueEffects(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_sync_failed"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) syncEvaluationResultIssueEffects(ctx context.Context, evaluationID string, institutionID string) error {
	evaluation, err := s.loadEvaluationFileContext(ctx, evaluationID, institutionID)
	if err != nil {
		return err
	}
	if evaluation == nil {
		return nil
	}
	return s.syncEvaluationResultIssueDocumentsToPersonnelFile(ctx, evaluation, institutionID)
}

type evaluationResultIssueMirrorRecord struct {
	IssueCode               string
	DocumentType            string
	RecipientName           string
	RecipientRole           string
	DeliveryChannel         string
	DeliveryStatus          string
	IssuedOn                time.Time
	DeliveredOn             *time.Time
	AcknowledgedOn          *time.Time
	RegistryReference       string
	AttachedToPersonnelFile bool
	Notes                   string
}

func (s *Service) syncEvaluationResultIssueDocumentsToPersonnelFile(ctx context.Context, evaluation *evaluationFileContext, institutionID string) error {
	if evaluation.PersonnelID == "" {
		return nil
	}

	rows, err := s.pool.Query(ctx, `
		select
			issue_code, document_type, recipient_name, recipient_role, delivery_channel, delivery_status,
			issued_on, delivered_on, acknowledged_on, registry_reference, attached_to_personnel_file, notes
		from education_evaluation_result_issues
		where evaluation_id = $1 and institution_id = $2
		order by issued_on, issue_code
	`, evaluation.ID, institutionID)
	if err != nil {
		return fmt.Errorf("list evaluation result issues for personnel file sync: %w", err)
	}
	defer rows.Close()

	activeFileReferences := make([]string, 0)
	for rows.Next() {
		var issue evaluationResultIssueMirrorRecord
		if err := rows.Scan(
			&issue.IssueCode,
			&issue.DocumentType,
			&issue.RecipientName,
			&issue.RecipientRole,
			&issue.DeliveryChannel,
			&issue.DeliveryStatus,
			&issue.IssuedOn,
			&issue.DeliveredOn,
			&issue.AcknowledgedOn,
			&issue.RegistryReference,
			&issue.AttachedToPersonnelFile,
			&issue.Notes,
		); err != nil {
			return fmt.Errorf("scan evaluation result issue for personnel file sync: %w", err)
		}
		if !issue.AttachedToPersonnelFile {
			continue
		}

		fileReference := fmt.Sprintf("%s/%s", evaluation.EvaluationCode, issue.IssueCode)
		activeFileReferences = append(activeFileReferences, fileReference)
		if err := s.upsertEvaluationResultIssueDocument(ctx, evaluation, issue, fileReference, institutionID); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate evaluation result issues for personnel file sync: %w", err)
	}

	deleteQuery := `
		delete from education_personnel_file_documents
		where personnel_id = $1 and institution_id = $2 and document_category = 'evaluare' and file_reference like $3
	`
	deleteArgs := []any{evaluation.PersonnelID, institutionID, evaluation.EvaluationCode + "/EVRES-%"}
	if len(activeFileReferences) > 0 {
		deleteQuery += ` and not (file_reference = any($4::text[]))`
		deleteArgs = append(deleteArgs, activeFileReferences)
	}

	if _, err := s.pool.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("delete stale mirrored evaluation result issue documents: %w", err)
	}

	return nil
}

func (s *Service) upsertEvaluationResultIssueDocument(ctx context.Context, evaluation *evaluationFileContext, issue evaluationResultIssueMirrorRecord, fileReference string, institutionID string) error {
	title := fmt.Sprintf("Comunicare rezultat evaluare %s - %s", evaluation.SchoolYear, issue.IssueCode)
	notesParts := []string{
		fmt.Sprintf("Evaluare: %s", evaluation.EvaluationCode),
		fmt.Sprintf("Document: %s", issue.DocumentType),
		fmt.Sprintf("Destinatar: %s", issue.RecipientName),
		fmt.Sprintf("Canal: %s", issue.DeliveryChannel),
		fmt.Sprintf("Status livrare: %s", issue.DeliveryStatus),
	}
	if strings.TrimSpace(issue.RecipientRole) != "" {
		notesParts = append(notesParts, "Rol destinatar: "+issue.RecipientRole)
	}
	if strings.TrimSpace(issue.RegistryReference) != "" {
		notesParts = append(notesParts, "Referinta registratura: "+issue.RegistryReference)
	}
	if issue.DeliveredOn != nil {
		notesParts = append(notesParts, "Predat la: "+issue.DeliveredOn.Format("2006-01-02"))
	}
	if issue.AcknowledgedOn != nil {
		notesParts = append(notesParts, "Confirmat la: "+issue.AcknowledgedOn.Format("2006-01-02"))
	}
	if strings.TrimSpace(issue.Notes) != "" {
		notesParts = append(notesParts, "Note: "+issue.Notes)
	}

	fileScope := personnelFileScope(evaluation.PersonnelRoleTitle, evaluation.RoleTitle)
	var existingID string
	err := s.pool.QueryRow(ctx, `
		select id::text
		from education_personnel_file_documents
		where personnel_id = $1 and institution_id = $2 and file_reference = $3 and document_category = 'evaluare'
	`, evaluation.PersonnelID, institutionID, fileReference).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		documentCode := fmt.Sprintf("PFD-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
		if _, err := s.pool.Exec(ctx, `
			insert into education_personnel_file_documents (
				personnel_id, document_code, document_category, document_title, file_scope, confidentiality_level,
				issued_on, expires_on, file_reference, sensitive_data, included_in_portfolio, institution_id, notes
			) values ($1::uuid, $2, 'evaluare', $3, $4, 'confidential', $5, null, $6, true, false, $7, $8)
		`, evaluation.PersonnelID, documentCode, title, fileScope, issue.IssuedOn, fileReference, institutionID, strings.Join(notesParts, " | ")); err != nil {
			return fmt.Errorf("insert mirrored evaluation result issue document: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("find mirrored evaluation result issue document: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
		update education_personnel_file_documents
		set
			document_title = $1,
			file_scope = $2,
			confidentiality_level = 'confidential',
			issued_on = $3,
			file_reference = $4,
			sensitive_data = true,
			included_in_portfolio = false,
			notes = $5,
			updated_at = now()
		where id = $6 and personnel_id = $7 and institution_id = $8
	`, title, fileScope, issue.IssuedOn, fileReference, strings.Join(notesParts, " | "), existingID, evaluation.PersonnelID, institutionID); err != nil {
		return fmt.Errorf("update mirrored evaluation result issue document: %w", err)
	}

	return nil
}

func normalizeEvaluationResultIssueRequest(req *CreatePersonnelEvaluationResultIssueRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.RecipientRole = strings.TrimSpace(req.RecipientRole)
	req.DeliveryChannel = strings.TrimSpace(req.DeliveryChannel)
	req.DeliveryStatus = strings.TrimSpace(req.DeliveryStatus)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.DeliveredOn = strings.TrimSpace(req.DeliveredOn)
	req.AcknowledgedOn = strings.TrimSpace(req.AcknowledgedOn)
	req.RegistryReference = strings.TrimSpace(req.RegistryReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func buildEvaluationResultIssueFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	clauses := []string{"eeri.evaluation_id = $1", "eeri.institution_id = $2"}
	args := []any{recordID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["issue_code"]); value != "" {
		addContains("eeri.issue_code", value)
	}
	if value := strings.TrimSpace(filters["document_type"]); value != "" {
		addEqual("eeri.document_type", value)
	}
	if value := strings.TrimSpace(filters["recipient_name"]); value != "" {
		addContains("eeri.recipient_name", value)
	}
	if value := strings.TrimSpace(filters["recipient_role"]); value != "" {
		addContains("eeri.recipient_role", value)
	}
	if value := strings.TrimSpace(filters["delivery_channel"]); value != "" {
		addEqual("eeri.delivery_channel", value)
	}
	if value := strings.TrimSpace(filters["delivery_status"]); value != "" {
		addEqual("eeri.delivery_status", value)
	}
	if value := strings.TrimSpace(filters["issued_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("eeri.issued_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["delivered_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("eeri.delivered_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["acknowledged_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("eeri.acknowledged_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["attached_to_personnel_file"]); value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("eeri.attached_to_personnel_file = $%d", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func evaluationResultIssueSortColumn(value string) string {
	switch value {
	case "issue_code":
		return "eeri.issue_code"
	case "document_type":
		return "eeri.document_type"
	case "recipient_name":
		return "eeri.recipient_name"
	case "recipient_role":
		return "eeri.recipient_role"
	case "delivery_channel":
		return "eeri.delivery_channel"
	case "delivery_status":
		return "eeri.delivery_status"
	case "issued_on":
		return "eeri.issued_on"
	case "delivered_on":
		return "eeri.delivered_on"
	case "acknowledged_on":
		return "eeri.acknowledged_on"
	default:
		return "eeri.issued_on"
	}
}
