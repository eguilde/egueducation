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

func (s *Service) DecisionIssuances(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"issuance_code":    {},
		"document_type":    {},
		"recipient_name":   {},
		"recipient_role":   {},
		"delivery_channel": {},
		"delivery_status":  {},
	}, []string{"issuance_code", "document_type", "recipient_name", "recipient_role", "delivery_channel", "delivery_status", "signed_on", "delivered_on"})
	if query.Sort == "" {
		query.Sort = "signed_on"
	}

	whereClause, args := buildDecisionIssuanceFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_decision_issuances edi "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_issuances_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			edi.id::text,
			edi.decision_id::text,
			edi.issuance_code,
			edi.document_type,
			edi.recipient_name,
			edi.recipient_role,
			edi.delivery_channel,
			edi.delivery_status,
			coalesce(to_char(edi.signed_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(edi.delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(edi.acknowledged_on, 'YYYY-MM-DD'), ''),
			edi.file_reference,
			edi.institution_id,
			edi.notes
		from education_decision_issuances edi
		%s
		order by %s %s, edi.issuance_code desc
		limit $%d offset $%d
	`, whereClause, decisionIssuanceSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_issuances_failed"})
		return
	}
	defer rows.Close()

	items := make([]DecisionIssuance, 0, query.PageSize)
	for rows.Next() {
		var item DecisionIssuance
		if err := rows.Scan(
			&item.ID,
			&item.DecisionID,
			&item.IssuanceCode,
			&item.DocumentType,
			&item.RecipientName,
			&item.RecipientRole,
			&item.DeliveryChannel,
			&item.DeliveryStatus,
			&item.SignedOn,
			&item.DeliveredOn,
			&item.AcknowledgedOn,
			&item.FileReference,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_issuances_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_issuances_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) DecisionIssuanceDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	var item DecisionIssuance
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			decision_id::text,
			issuance_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			coalesce(to_char(signed_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			file_reference,
			institution_id,
			notes
		from education_decision_issuances
		where id = $1 and decision_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DecisionID,
		&item.IssuanceCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.SignedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_decision_issuance_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_issuance_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateDecisionIssuance(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	var req CreateDecisionIssuanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_payload"})
		return
	}

	normalizeDecisionIssuanceRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "extras", "comunicare", "adeverinta", "dispozitie") ||
		!inStringSet(req.DeliveryChannel, "intern", "email", "registratura", "avizier", "site") ||
		!inStringSet(req.DeliveryStatus, "draft", "semnat", "transmis", "confirmat", "returnat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_fields"})
		return
	}

	signedOn, err := parseOptionalEducationDate(req.SignedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_signed_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_delivered_on"})
		return
	}
	acknowledgedOn, err := parseOptionalEducationDate(req.AcknowledgedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_acknowledged_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "semnat", "transmis", "confirmat", "returnat") && signedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_signed_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat", "returnat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_delivered_on"})
		return
	}
	if req.DeliveryStatus == "confirmat" && acknowledgedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_acknowledged_on"})
		return
	}

	code := fmt.Sprintf("DEC-OUT-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item DecisionIssuance
	err = s.pool.QueryRow(r.Context(), `
		insert into education_decision_issuances (
			decision_id,
			issuance_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			signed_on,
			delivered_on,
			acknowledged_on,
			file_reference,
			institution_id,
			notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		returning
			id::text,
			decision_id::text,
			issuance_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			coalesce(to_char(signed_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			file_reference,
			institution_id,
			notes
	`, recordID, code, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, signedOn, deliveredOn, acknowledgedOn, req.FileReference, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.DecisionID,
		&item.IssuanceCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.SignedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_issuance_create_failed"})
		return
	}

	s.logAudit(r, "education.decisions.issuance.create", "decision_issuance", item.ID, "Decision issuance created.", map[string]any{
		"decision_id":      recordID,
		"issuance_code":    item.IssuanceCode,
		"delivery_status":  item.DeliveryStatus,
		"delivery_channel": item.DeliveryChannel,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateDecisionIssuance(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateDecisionIssuanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_payload"})
		return
	}

	normalizeDecisionIssuanceRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "extras", "comunicare", "adeverinta", "dispozitie") ||
		!inStringSet(req.DeliveryChannel, "intern", "email", "registratura", "avizier", "site") ||
		!inStringSet(req.DeliveryStatus, "draft", "semnat", "transmis", "confirmat", "returnat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_fields"})
		return
	}

	signedOn, err := parseOptionalEducationDate(req.SignedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_signed_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_delivered_on"})
		return
	}
	acknowledgedOn, err := parseOptionalEducationDate(req.AcknowledgedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_issuance_acknowledged_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "semnat", "transmis", "confirmat", "returnat") && signedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_signed_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat", "returnat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_delivered_on"})
		return
	}
	if req.DeliveryStatus == "confirmat" && acknowledgedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_issuance_acknowledged_on"})
		return
	}

	var item DecisionIssuance
	err = s.pool.QueryRow(r.Context(), `
		update education_decision_issuances
		set document_type = $1, recipient_name = $2, recipient_role = $3, delivery_channel = $4,
			delivery_status = $5, signed_on = $6, delivered_on = $7, acknowledged_on = $8,
			file_reference = $9, notes = $10, updated_at = now()
		where id = $11 and decision_id = $12 and institution_id = $13
		returning
			id::text,
			decision_id::text,
			issuance_code,
			document_type,
			recipient_name,
			recipient_role,
			delivery_channel,
			delivery_status,
			coalesce(to_char(signed_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''),
			file_reference,
			institution_id,
			notes
	`, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, signedOn, deliveredOn, acknowledgedOn, req.FileReference, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DecisionID,
		&item.IssuanceCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.SignedOn,
		&item.DeliveredOn,
		&item.AcknowledgedOn,
		&item.FileReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_decision_issuance_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_issuance_update_failed"})
		return
	}

	s.logAudit(r, "education.decisions.issuance.update", "decision_issuance", item.ID, "Decision issuance updated.", map[string]any{
		"decision_id":      recordID,
		"issuance_code":    item.IssuanceCode,
		"delivery_status":  item.DeliveryStatus,
		"delivery_channel": item.DeliveryChannel,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteDecisionIssuance(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_decision_issuances where id = $1 and decision_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_issuance_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_decision_issuance_not_found")
		return
	}

	s.logAudit(r, "education.decisions.issuance.delete", "decision_issuance", itemID, "Decision issuance deleted.", map[string]any{
		"decision_id": recordID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) DecisionPublicationSteps(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"step_type":             {},
		"status":                {},
		"responsible_name":      {},
		"publication_channel":   {},
		"publication_reference": {},
	}, []string{"step_order", "step_type", "status", "responsible_name", "publication_channel", "due_on", "completed_on"})
	if query.Sort == "" {
		query.Sort = "step_order"
	}

	whereClause, args := buildDecisionPublicationStepFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_decision_publication_steps edps "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_publication_steps_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			edps.id::text,
			edps.decision_id::text,
			edps.step_order,
			edps.step_type,
			edps.status,
			edps.responsible_name,
			edps.publication_channel,
			to_char(edps.due_on, 'YYYY-MM-DD'),
			coalesce(to_char(edps.completed_on, 'YYYY-MM-DD'), ''),
			edps.publication_reference,
			edps.institution_id,
			edps.notes
		from education_decision_publication_steps edps
		%s
		order by %s %s, edps.step_order, edps.id
		limit $%d offset $%d
	`, whereClause, decisionPublicationStepSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_publication_steps_failed"})
		return
	}
	defer rows.Close()

	items := make([]DecisionPublicationStep, 0, query.PageSize)
	for rows.Next() {
		var item DecisionPublicationStep
		if err := rows.Scan(
			&item.ID,
			&item.DecisionID,
			&item.StepOrder,
			&item.StepType,
			&item.Status,
			&item.ResponsibleName,
			&item.PublicationChannel,
			&item.DueOn,
			&item.CompletedOn,
			&item.PublicationReference,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_publication_steps_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_publication_steps_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) DecisionPublicationStepDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	var item DecisionPublicationStep
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			decision_id::text,
			step_order,
			step_type,
			status,
			responsible_name,
			publication_channel,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			publication_reference,
			institution_id,
			notes
		from education_decision_publication_steps
		where id = $1 and decision_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DecisionID,
		&item.StepOrder,
		&item.StepType,
		&item.Status,
		&item.ResponsibleName,
		&item.PublicationChannel,
		&item.DueOn,
		&item.CompletedOn,
		&item.PublicationReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_decision_publication_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_publication_step_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateDecisionPublicationStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	var req CreateDecisionPublicationStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_payload"})
		return
	}

	normalizeDecisionPublicationStepRequest(&req)
	if req.StepOrder <= 0 || req.StepType == "" || req.Status == "" || req.ResponsibleName == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_publication_step_fields"})
		return
	}
	if !inStringSet(req.StepType, "analiza_juridica", "anonimizare", "aprobare_publicare", "publicare", "retragere") ||
		!inStringSet(req.Status, "pending", "in_progress", "completed", "blocked") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_fields"})
		return
	}
	if req.PublicationChannel != "" && !inStringSet(req.PublicationChannel, "site_public", "avizier", "intranet", "registratura") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_channel"})
		return
	}

	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_completed_on"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_publication_step_completed_on"})
		return
	}

	var item DecisionPublicationStep
	err = s.pool.QueryRow(r.Context(), `
		insert into education_decision_publication_steps (
			decision_id,
			step_order,
			step_type,
			status,
			responsible_name,
			publication_channel,
			due_on,
			completed_on,
			publication_reference,
			institution_id,
			notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning
			id::text,
			decision_id::text,
			step_order,
			step_type,
			status,
			responsible_name,
			publication_channel,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			publication_reference,
			institution_id,
			notes
	`, recordID, req.StepOrder, req.StepType, req.Status, req.ResponsibleName, req.PublicationChannel, dueOn, completedOn, req.PublicationReference, s.institutionID(r), req.Notes).Scan(
		&item.ID,
		&item.DecisionID,
		&item.StepOrder,
		&item.StepType,
		&item.Status,
		&item.ResponsibleName,
		&item.PublicationChannel,
		&item.DueOn,
		&item.CompletedOn,
		&item.PublicationReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_publication_step_create_failed"})
		return
	}

	s.logAudit(r, "education.decisions.publication.create", "decision_publication_step", item.ID, "Decision publication step created.", map[string]any{
		"decision_id": recordID,
		"step_type":   item.StepType,
		"status":      item.Status,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateDecisionPublicationStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateDecisionPublicationStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_payload"})
		return
	}

	normalizeDecisionPublicationStepRequest(&req)
	if req.StepOrder <= 0 || req.StepType == "" || req.Status == "" || req.ResponsibleName == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_publication_step_fields"})
		return
	}
	if !inStringSet(req.StepType, "analiza_juridica", "anonimizare", "aprobare_publicare", "publicare", "retragere") ||
		!inStringSet(req.Status, "pending", "in_progress", "completed", "blocked") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_fields"})
		return
	}
	if req.PublicationChannel != "" && !inStringSet(req.PublicationChannel, "site_public", "avizier", "intranet", "registratura") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_channel"})
		return
	}

	dueOn, err := parseRequiredEducationDate(req.DueOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_due_on"})
		return
	}
	completedOn, err := parseOptionalEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_publication_step_completed_on"})
		return
	}
	if req.Status == "completed" && completedOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_publication_step_completed_on"})
		return
	}

	var item DecisionPublicationStep
	err = s.pool.QueryRow(r.Context(), `
		update education_decision_publication_steps
		set step_order = $1, step_type = $2, status = $3, responsible_name = $4, publication_channel = $5,
			due_on = $6, completed_on = $7, publication_reference = $8, notes = $9, updated_at = now()
		where id = $10 and decision_id = $11 and institution_id = $12
		returning
			id::text,
			decision_id::text,
			step_order,
			step_type,
			status,
			responsible_name,
			publication_channel,
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			publication_reference,
			institution_id,
			notes
	`, req.StepOrder, req.StepType, req.Status, req.ResponsibleName, req.PublicationChannel, dueOn, completedOn, req.PublicationReference, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID,
		&item.DecisionID,
		&item.StepOrder,
		&item.StepType,
		&item.Status,
		&item.ResponsibleName,
		&item.PublicationChannel,
		&item.DueOn,
		&item.CompletedOn,
		&item.PublicationReference,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_decision_publication_step_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_publication_step_update_failed"})
		return
	}

	s.logAudit(r, "education.decisions.publication.update", "decision_publication_step", item.ID, "Decision publication step updated.", map[string]any{
		"decision_id": recordID,
		"step_type":   item.StepType,
		"status":      item.Status,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteDecisionPublicationStep(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_decision_publication_steps where id = $1 and decision_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_publication_step_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_decision_publication_step_not_found")
		return
	}

	s.logAudit(r, "education.decisions.publication.delete", "decision_publication_step", itemID, "Decision publication step deleted.", map[string]any{
		"decision_id": recordID,
	})

	w.WriteHeader(http.StatusNoContent)
}

func normalizeDecisionIssuanceRequest(req *CreateDecisionIssuanceRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.RecipientRole = strings.TrimSpace(req.RecipientRole)
	req.DeliveryChannel = strings.TrimSpace(req.DeliveryChannel)
	req.DeliveryStatus = strings.TrimSpace(req.DeliveryStatus)
	req.SignedOn = strings.TrimSpace(req.SignedOn)
	req.DeliveredOn = strings.TrimSpace(req.DeliveredOn)
	req.AcknowledgedOn = strings.TrimSpace(req.AcknowledgedOn)
	req.FileReference = strings.TrimSpace(req.FileReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeDecisionPublicationStepRequest(req *CreateDecisionPublicationStepRequest) {
	req.StepType = strings.TrimSpace(req.StepType)
	req.Status = strings.TrimSpace(req.Status)
	req.ResponsibleName = strings.TrimSpace(req.ResponsibleName)
	req.PublicationChannel = strings.TrimSpace(req.PublicationChannel)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.CompletedOn = strings.TrimSpace(req.CompletedOn)
	req.PublicationReference = strings.TrimSpace(req.PublicationReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func buildDecisionIssuanceFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"edi.decision_id = $1", "edi.institution_id = $2"}
	args := []any{recordID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}

	if value := filters["issuance_code"]; value != "" {
		addContains("edi.issuance_code", value)
	}
	if value := filters["document_type"]; value != "" {
		addContains("edi.document_type", value)
	}
	if value := filters["recipient_name"]; value != "" {
		addContains("edi.recipient_name", value)
	}
	if value := filters["recipient_role"]; value != "" {
		addContains("edi.recipient_role", value)
	}
	if value := filters["delivery_channel"]; value != "" {
		addContains("edi.delivery_channel", value)
	}
	if value := filters["delivery_status"]; value != "" {
		addContains("edi.delivery_status", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func buildDecisionPublicationStepFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"edps.decision_id = $1", "edps.institution_id = $2"}
	args := []any{recordID, institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}

	if value := filters["step_type"]; value != "" {
		addContains("edps.step_type", value)
	}
	if value := filters["status"]; value != "" {
		addContains("edps.status", value)
	}
	if value := filters["responsible_name"]; value != "" {
		addContains("edps.responsible_name", value)
	}
	if value := filters["publication_channel"]; value != "" {
		addContains("edps.publication_channel", value)
	}
	if value := filters["publication_reference"]; value != "" {
		addContains("edps.publication_reference", value)
	}

	return "where " + strings.Join(where, " and "), args
}

func decisionIssuanceSortColumn(value string) string {
	switch value {
	case "issuance_code":
		return "edi.issuance_code"
	case "document_type":
		return "edi.document_type"
	case "recipient_name":
		return "edi.recipient_name"
	case "recipient_role":
		return "edi.recipient_role"
	case "delivery_channel":
		return "edi.delivery_channel"
	case "delivery_status":
		return "edi.delivery_status"
	case "delivered_on":
		return "edi.delivered_on"
	default:
		return "edi.signed_on"
	}
}

func decisionPublicationStepSortColumn(value string) string {
	switch value {
	case "step_order":
		return "edps.step_order"
	case "step_type":
		return "edps.step_type"
	case "status":
		return "edps.status"
	case "responsible_name":
		return "edps.responsible_name"
	case "publication_channel":
		return "edps.publication_channel"
	case "completed_on":
		return "edps.completed_on"
	default:
		return "edps.due_on"
	}
}
