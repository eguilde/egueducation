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

func (s *Service) MobilityFinalDecisions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"decision_code":    {},
		"decision_type":    {},
		"outcome":          {},
		"panel_name":       {},
		"destination_unit": {},
	}, []string{"decision_code", "decision_type", "outcome", "panel_name", "approved_on", "effective_from"})
	if query.Sort == "" {
		query.Sort = "approved_on"
	}

	whereClause, args := buildMobilityFinalDecisionFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_final_decisions emfd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decisions_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emfd.id::text, emfd.mobility_case_id::text, emfd.decision_code, emfd.decision_type, emfd.outcome,
			to_char(emfd.approved_on, 'YYYY-MM-DD'), to_char(emfd.effective_from, 'YYYY-MM-DD'), emfd.panel_name,
			emfd.legal_basis, emfd.destination_unit, emfd.institution_id, emfd.notes
		from education_mobility_final_decisions emfd
		%s
		order by %s %s, emfd.decision_code desc
		limit $%d offset $%d
	`, whereClause, mobilityFinalDecisionSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decisions_failed"})
		return
	}
	defer rows.Close()

	items := make([]MobilityFinalDecision, 0, query.PageSize)
	for rows.Next() {
		var item MobilityFinalDecision
		if err := rows.Scan(
			&item.ID, &item.MobilityCaseID, &item.DecisionCode, &item.DecisionType, &item.Outcome,
			&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.LegalBasis, &item.DestinationUnit, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decisions_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decisions_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityFinalDecisionDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MobilityFinalDecision
	err := s.pool.QueryRow(r.Context(), `
		select id::text, mobility_case_id::text, decision_code, decision_type, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, legal_basis, destination_unit, institution_id, notes
		from education_mobility_final_decisions
		where id = $1 and mobility_case_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.DecisionCode, &item.DecisionType, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.LegalBasis, &item.DestinationUnit, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decision_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMobilityFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMobilityFinalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_payload"})
		return
	}

	normalizeMobilityFinalDecisionRequest(&req)
	if req.DecisionType == "" || req.Outcome == "" || req.ApprovedOn == "" || req.EffectiveFrom == "" || req.PanelName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_final_decision_fields"})
		return
	}
	if !inStringSet(req.DecisionType, "validare_dosar", "repartizare", "transfer", "detasare", "solutionare_contestatie") ||
		!inStringSet(req.Outcome, "admis", "respins", "redistribuit", "rezerva") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_fields"})
		return
	}
	approvedOn, err := parseRequiredEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_effective_from"})
		return
	}

	code := fmt.Sprintf("MOB-DEC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MobilityFinalDecision
	err = s.pool.QueryRow(r.Context(), `
		insert into education_mobility_final_decisions (
			mobility_case_id, decision_code, decision_type, outcome, approved_on, effective_from,
			panel_name, legal_basis, destination_unit, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning id::text, mobility_case_id::text, decision_code, decision_type, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, legal_basis, destination_unit, institution_id, notes
	`, recordID, code, req.DecisionType, req.Outcome, approvedOn, effectiveFrom, req.PanelName, req.LegalBasis, req.DestinationUnit, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.MobilityCaseID, &item.DecisionCode, &item.DecisionType, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.LegalBasis, &item.DestinationUnit, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_create_failed"})
		return
	}

	s.logAudit(r, "education.mobility.final_decision.create", "mobility_final_decision", item.ID, "Mobility final decision created.", map[string]any{
		"mobility_case_id": recordID,
		"decision_code":    item.DecisionCode,
		"outcome":          item.Outcome,
	})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMobilityFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMobilityFinalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_payload"})
		return
	}

	normalizeMobilityFinalDecisionRequest(&req)
	if req.DecisionType == "" || req.Outcome == "" || req.ApprovedOn == "" || req.EffectiveFrom == "" || req.PanelName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_final_decision_fields"})
		return
	}
	if !inStringSet(req.DecisionType, "validare_dosar", "repartizare", "transfer", "detasare", "solutionare_contestatie") ||
		!inStringSet(req.Outcome, "admis", "respins", "redistribuit", "rezerva") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_fields"})
		return
	}
	approvedOn, err := parseRequiredEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_final_decision_effective_from"})
		return
	}

	var item MobilityFinalDecision
	err = s.pool.QueryRow(r.Context(), `
		update education_mobility_final_decisions
		set decision_type=$1, outcome=$2, approved_on=$3, effective_from=$4, panel_name=$5,
			legal_basis=$6, destination_unit=$7, notes=$8, updated_at=now()
		where id=$9 and mobility_case_id=$10 and institution_id=$11
		returning id::text, mobility_case_id::text, decision_code, decision_type, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, legal_basis, destination_unit, institution_id, notes
	`, req.DecisionType, req.Outcome, approvedOn, effectiveFrom, req.PanelName, req.LegalBasis, req.DestinationUnit, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.DecisionCode, &item.DecisionType, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.LegalBasis, &item.DestinationUnit, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_update_failed"})
		return
	}

	s.logAudit(r, "education.mobility.final_decision.update", "mobility_final_decision", item.ID, "Mobility final decision updated.", map[string]any{
		"mobility_case_id": recordID,
		"decision_code":    item.DecisionCode,
		"outcome":          item.Outcome,
	})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_mobility_final_decisions where id = $1 and mobility_case_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_mobility_final_decision_not_found")
		return
	}
	s.logAudit(r, "education.mobility.final_decision.delete", "mobility_final_decision", itemID, "Mobility final decision deleted.", map[string]any{"mobility_case_id": recordID})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_final_decision_sync_failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MobilityResultIssues(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"issue_code":         {},
		"document_type":      {},
		"recipient_name":     {},
		"recipient_role":     {},
		"delivery_channel":   {},
		"delivery_status":    {},
		"registry_reference": {},
	}, []string{"issue_code", "document_type", "recipient_name", "recipient_role", "delivery_channel", "delivery_status", "issued_on", "delivered_on"})
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildMobilityResultIssueFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_result_issues emri "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issues_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emri.id::text, emri.mobility_case_id::text, emri.issue_code, emri.document_type, emri.recipient_name,
			emri.recipient_role, emri.delivery_channel, emri.delivery_status,
			to_char(emri.issued_on, 'YYYY-MM-DD'), coalesce(to_char(emri.delivered_on, 'YYYY-MM-DD'), ''),
			emri.registry_reference, emri.institution_id, emri.notes
		from education_mobility_result_issues emri
		%s
		order by %s %s, emri.issue_code desc
		limit $%d offset $%d
	`, whereClause, mobilityResultIssueSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issues_failed"})
		return
	}
	defer rows.Close()

	items := make([]MobilityResultIssue, 0, query.PageSize)
	for rows.Next() {
		var item MobilityResultIssue
		if err := rows.Scan(
			&item.ID, &item.MobilityCaseID, &item.IssueCode, &item.DocumentType, &item.RecipientName,
			&item.RecipientRole, &item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn,
			&item.RegistryReference, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issues_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issues_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityResultIssueDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MobilityResultIssue
	err := s.pool.QueryRow(r.Context(), `
		select id::text, mobility_case_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
		from education_mobility_result_issues
		where id = $1 and mobility_case_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issue_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMobilityResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMobilityResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_payload"})
		return
	}

	normalizeMobilityResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "comunicare", "adeverinta", "raport_final") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_fields"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_delivered_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_result_issue_delivered_on"})
		return
	}

	code := fmt.Sprintf("MOB-OUT-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MobilityResultIssue
	err = s.pool.QueryRow(r.Context(), `
		insert into education_mobility_result_issues (
			mobility_case_id, issue_code, document_type, recipient_name, recipient_role, delivery_channel,
			delivery_status, issued_on, delivered_on, registry_reference, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id::text, mobility_case_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
	`, recordID, code, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, req.RegistryReference, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.MobilityCaseID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_create_failed"})
		return
	}

	s.logAudit(r, "education.mobility.result_issue.create", "mobility_result_issue", item.ID, "Mobility result issue created.", map[string]any{
		"mobility_case_id": recordID,
		"issue_code":       item.IssueCode,
		"delivery_status":  item.DeliveryStatus,
	})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMobilityResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMobilityResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_payload"})
		return
	}
	normalizeMobilityResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "comunicare", "adeverinta", "raport_final") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_fields"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_result_issue_delivered_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_result_issue_delivered_on"})
		return
	}

	var item MobilityResultIssue
	err = s.pool.QueryRow(r.Context(), `
		update education_mobility_result_issues
		set document_type=$1, recipient_name=$2, recipient_role=$3, delivery_channel=$4, delivery_status=$5,
			issued_on=$6, delivered_on=$7, registry_reference=$8, notes=$9, updated_at=now()
		where id=$10 and mobility_case_id=$11 and institution_id=$12
		returning id::text, mobility_case_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
	`, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, req.RegistryReference, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_update_failed"})
		return
	}

	s.logAudit(r, "education.mobility.result_issue.update", "mobility_result_issue", item.ID, "Mobility result issue updated.", map[string]any{
		"mobility_case_id": recordID,
		"issue_code":       item.IssueCode,
		"delivery_status":  item.DeliveryStatus,
	})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_mobility_result_issues where id = $1 and mobility_case_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_mobility_result_issue_not_found")
		return
	}
	s.logAudit(r, "education.mobility.result_issue.delete", "mobility_result_issue", itemID, "Mobility result issue deleted.", map[string]any{"mobility_case_id": recordID})
	if err := s.syncMobilityCaseOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_result_issue_sync_failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MeritFinalDecisions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"decision_code":  {},
		"decision_stage": {},
		"outcome":        {},
		"panel_name":     {},
	}, []string{"decision_code", "decision_stage", "outcome", "panel_name", "approved_on", "effective_from"})
	if query.Sort == "" {
		query.Sort = "approved_on"
	}

	whereClause, args := buildMeritFinalDecisionFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_final_decisions emfd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decisions_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emfd.id::text, emfd.grant_id::text, emfd.decision_code, emfd.decision_stage, emfd.outcome,
			to_char(emfd.approved_on, 'YYYY-MM-DD'), to_char(emfd.effective_from, 'YYYY-MM-DD'),
			emfd.panel_name, emfd.funded, emfd.legal_basis, emfd.institution_id, emfd.notes
		from education_merit_final_decisions emfd
		%s
		order by %s %s, emfd.decision_code desc
		limit $%d offset $%d
	`, whereClause, meritFinalDecisionSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decisions_failed"})
		return
	}
	defer rows.Close()

	items := make([]MeritFinalDecision, 0, query.PageSize)
	for rows.Next() {
		var item MeritFinalDecision
		if err := rows.Scan(
			&item.ID, &item.GrantID, &item.DecisionCode, &item.DecisionStage, &item.Outcome,
			&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.Funded, &item.LegalBasis, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decisions_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decisions_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritFinalDecisionDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MeritFinalDecision
	err := s.pool.QueryRow(r.Context(), `
		select id::text, grant_id::text, decision_code, decision_stage, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, funded, legal_basis, institution_id, notes
		from education_merit_final_decisions
		where id = $1 and grant_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.DecisionCode, &item.DecisionStage, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.Funded, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decision_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMeritFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMeritFinalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_payload"})
		return
	}
	normalizeMeritFinalDecisionRequest(&req)
	if req.DecisionStage == "" || req.Outcome == "" || req.ApprovedOn == "" || req.EffectiveFrom == "" || req.PanelName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_final_decision_fields"})
		return
	}
	if !inStringSet(req.DecisionStage, "evaluare_initiala", "solutionare_contestatie", "validare_finala", "finantare") ||
		!inStringSet(req.Outcome, "admis", "respins", "rezerva", "finantat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_fields"})
		return
	}
	approvedOn, err := parseRequiredEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_effective_from"})
		return
	}

	code := fmt.Sprintf("MER-DEC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MeritFinalDecision
	err = s.pool.QueryRow(r.Context(), `
		insert into education_merit_final_decisions (
			grant_id, decision_code, decision_stage, outcome, approved_on, effective_from,
			panel_name, funded, legal_basis, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning id::text, grant_id::text, decision_code, decision_stage, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, funded, legal_basis, institution_id, notes
	`, recordID, code, req.DecisionStage, req.Outcome, approvedOn, effectiveFrom, req.PanelName, req.Funded, req.LegalBasis, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.GrantID, &item.DecisionCode, &item.DecisionStage, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.Funded, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_create_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.final_decision.create", "merit_final_decision", item.ID, "Merit final decision created.", map[string]any{
		"grant_id":      recordID,
		"decision_code": item.DecisionCode,
		"outcome":       item.Outcome,
		"funded":        item.Funded,
	})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMeritFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMeritFinalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_payload"})
		return
	}
	normalizeMeritFinalDecisionRequest(&req)
	if req.DecisionStage == "" || req.Outcome == "" || req.ApprovedOn == "" || req.EffectiveFrom == "" || req.PanelName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_final_decision_fields"})
		return
	}
	if !inStringSet(req.DecisionStage, "evaluare_initiala", "solutionare_contestatie", "validare_finala", "finantare") ||
		!inStringSet(req.Outcome, "admis", "respins", "rezerva", "finantat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_fields"})
		return
	}
	approvedOn, err := parseRequiredEducationDate(req.ApprovedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_approved_on"})
		return
	}
	effectiveFrom, err := parseRequiredEducationDate(req.EffectiveFrom)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_final_decision_effective_from"})
		return
	}

	var item MeritFinalDecision
	err = s.pool.QueryRow(r.Context(), `
		update education_merit_final_decisions
		set decision_stage=$1, outcome=$2, approved_on=$3, effective_from=$4, panel_name=$5,
			funded=$6, legal_basis=$7, notes=$8, updated_at=now()
		where id=$9 and grant_id=$10 and institution_id=$11
		returning id::text, grant_id::text, decision_code, decision_stage, outcome,
			to_char(approved_on, 'YYYY-MM-DD'), to_char(effective_from, 'YYYY-MM-DD'),
			panel_name, funded, legal_basis, institution_id, notes
	`, req.DecisionStage, req.Outcome, approvedOn, effectiveFrom, req.PanelName, req.Funded, req.LegalBasis, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.DecisionCode, &item.DecisionStage, &item.Outcome,
		&item.ApprovedOn, &item.EffectiveFrom, &item.PanelName, &item.Funded, &item.LegalBasis, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_update_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.final_decision.update", "merit_final_decision", item.ID, "Merit final decision updated.", map[string]any{
		"grant_id":      recordID,
		"decision_code": item.DecisionCode,
		"outcome":       item.Outcome,
		"funded":        item.Funded,
	})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritFinalDecision(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_merit_final_decisions where id = $1 and grant_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_merit_final_decision_not_found")
		return
	}
	s.logAudit(r, "education.gradatii.final_decision.delete", "merit_final_decision", itemID, "Merit final decision deleted.", map[string]any{"grant_id": recordID})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_final_decision_sync_failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MeritResultIssues(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"issue_code":         {},
		"document_type":      {},
		"recipient_name":     {},
		"recipient_role":     {},
		"delivery_channel":   {},
		"delivery_status":    {},
		"registry_reference": {},
	}, []string{"issue_code", "document_type", "recipient_name", "recipient_role", "delivery_channel", "delivery_status", "issued_on", "delivered_on"})
	if query.Sort == "" {
		query.Sort = "issued_on"
	}

	whereClause, args := buildMeritResultIssueFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_result_issues emri "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issues_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emri.id::text, emri.grant_id::text, emri.issue_code, emri.document_type, emri.recipient_name,
			emri.recipient_role, emri.delivery_channel, emri.delivery_status,
			to_char(emri.issued_on, 'YYYY-MM-DD'), coalesce(to_char(emri.delivered_on, 'YYYY-MM-DD'), ''),
			emri.registry_reference, emri.institution_id, emri.notes
		from education_merit_result_issues emri
		%s
		order by %s %s, emri.issue_code desc
		limit $%d offset $%d
	`, whereClause, meritResultIssueSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issues_failed"})
		return
	}
	defer rows.Close()

	items := make([]MeritResultIssue, 0, query.PageSize)
	for rows.Next() {
		var item MeritResultIssue
		if err := rows.Scan(
			&item.ID, &item.GrantID, &item.IssueCode, &item.DocumentType, &item.RecipientName,
			&item.RecipientRole, &item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn,
			&item.RegistryReference, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issues_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issues_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritResultIssueDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MeritResultIssue
	err := s.pool.QueryRow(r.Context(), `
		select id::text, grant_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
		from education_merit_result_issues
		where id = $1 and grant_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issue_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMeritResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMeritResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_payload"})
		return
	}
	normalizeMeritResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "comunicare", "extras_punctaj", "adeverinta") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_fields"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_delivered_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_result_issue_delivered_on"})
		return
	}
	code := fmt.Sprintf("MER-OUT-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MeritResultIssue
	err = s.pool.QueryRow(r.Context(), `
		insert into education_merit_result_issues (
			grant_id, issue_code, document_type, recipient_name, recipient_role, delivery_channel,
			delivery_status, issued_on, delivered_on, registry_reference, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id::text, grant_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
	`, recordID, code, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, req.RegistryReference, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.GrantID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_create_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.result_issue.create", "merit_result_issue", item.ID, "Merit result issue created.", map[string]any{
		"grant_id":        recordID,
		"issue_code":      item.IssueCode,
		"delivery_status": item.DeliveryStatus,
	})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMeritResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMeritResultIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_payload"})
		return
	}
	normalizeMeritResultIssueRequest(&req)
	if req.DocumentType == "" || req.RecipientName == "" || req.DeliveryChannel == "" || req.DeliveryStatus == "" || req.IssuedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_result_issue_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "decizie", "comunicare", "extras_punctaj", "adeverinta") ||
		!inStringSet(req.DeliveryChannel, "registratura", "email", "intern", "posta") ||
		!inStringSet(req.DeliveryStatus, "pregatit", "emis", "transmis", "confirmat") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_fields"})
		return
	}
	issuedOn, err := parseRequiredEducationDate(req.IssuedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_issued_on"})
		return
	}
	deliveredOn, err := parseOptionalEducationDate(req.DeliveredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_result_issue_delivered_on"})
		return
	}
	if inStringSet(req.DeliveryStatus, "transmis", "confirmat") && deliveredOn == nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_result_issue_delivered_on"})
		return
	}

	var item MeritResultIssue
	err = s.pool.QueryRow(r.Context(), `
		update education_merit_result_issues
		set document_type=$1, recipient_name=$2, recipient_role=$3, delivery_channel=$4, delivery_status=$5,
			issued_on=$6, delivered_on=$7, registry_reference=$8, notes=$9, updated_at=now()
		where id=$10 and grant_id=$11 and institution_id=$12
		returning id::text, grant_id::text, issue_code, document_type, recipient_name, recipient_role,
			delivery_channel, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(delivered_on, 'YYYY-MM-DD'), ''), registry_reference, institution_id, notes
	`, req.DocumentType, req.RecipientName, req.RecipientRole, req.DeliveryChannel, req.DeliveryStatus, issuedOn, deliveredOn, req.RegistryReference, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.IssueCode, &item.DocumentType, &item.RecipientName, &item.RecipientRole,
		&item.DeliveryChannel, &item.DeliveryStatus, &item.IssuedOn, &item.DeliveredOn, &item.RegistryReference, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_update_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.result_issue.update", "merit_result_issue", item.ID, "Merit result issue updated.", map[string]any{
		"grant_id":        recordID,
		"issue_code":      item.IssueCode,
		"delivery_status": item.DeliveryStatus,
	})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_sync_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritResultIssue(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_merit_result_issues where id = $1 and grant_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_merit_result_issue_not_found")
		return
	}
	s.logAudit(r, "education.gradatii.result_issue.delete", "merit_result_issue", itemID, "Merit result issue deleted.", map[string]any{"grant_id": recordID})
	if err := s.syncMeritGrantOutcome(r.Context(), recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_result_issue_sync_failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) syncMobilityCaseOutcome(ctx context.Context, recordID string, institutionID string) error {
	type latestDecision struct {
		Outcome         string
		PanelName       string
		DestinationUnit string
	}

	var (
		decision latestDecision
		hasFinal bool
	)
	err := s.pool.QueryRow(ctx, `
		select outcome, panel_name, destination_unit
		from education_mobility_final_decisions
		where mobility_case_id = $1 and institution_id = $2
		order by approved_on desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(&decision.Outcome, &decision.PanelName, &decision.DestinationUnit)
	if errors.Is(err, pgx.ErrNoRows) {
		hasFinal = false
	} else if err != nil {
		return fmt.Errorf("load latest mobility final decision: %w", err)
	} else {
		hasFinal = true
	}

	var communicated bool
	if err := s.pool.QueryRow(ctx, `
		select exists(
			select 1
			from education_mobility_result_issues
			where mobility_case_id = $1
				and institution_id = $2
				and delivery_status in ('emis', 'transmis', 'confirmat')
		)
	`, recordID, institutionID).Scan(&communicated); err != nil {
		return fmt.Errorf("load mobility communicated result status: %w", err)
	}

	stage := "review"
	status := "pending"
	reviewedBy := ""
	destinationSchool := ""
	if hasFinal {
		stage = "approved"
		status = "approved"
		reviewedBy = decision.PanelName
		destinationSchool = decision.DestinationUnit
		if decision.Outcome == "respins" {
			status = "rejected"
		}
		if communicated {
			stage = "completed"
			if status != "rejected" {
				status = "completed"
			}
		}
	}

	tag, err := s.pool.Exec(ctx, `
		update education_mobility_cases
		set
			stage = $1,
			status = $2,
			reviewed_by = coalesce(nullif($3, ''), reviewed_by),
			destination_school = coalesce(nullif($4, ''), destination_school)
		where id = $5 and institution_id = $6
	`, stage, status, reviewedBy, destinationSchool, recordID, institutionID)
	if err != nil {
		return fmt.Errorf("sync mobility case outcome: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sync mobility case outcome: %w", pgx.ErrNoRows)
	}

	return nil
}

func (s *Service) syncMeritGrantOutcome(ctx context.Context, recordID string, institutionID string) error {
	type latestDecision struct {
		Outcome    string
		PanelName  string
		Funded     bool
		ApprovedOn time.Time
	}

	var (
		decision latestDecision
		hasFinal bool
	)
	err := s.pool.QueryRow(ctx, `
		select outcome, panel_name, funded, approved_on
		from education_merit_final_decisions
		where grant_id = $1 and institution_id = $2
		order by approved_on desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(&decision.Outcome, &decision.PanelName, &decision.Funded, &decision.ApprovedOn)
	if errors.Is(err, pgx.ErrNoRows) {
		hasFinal = false
	} else if err != nil {
		return fmt.Errorf("load latest merit final decision: %w", err)
	} else {
		hasFinal = true
	}

	status := "evaluated"
	funded := false
	committeeName := ""
	var decisionDate any
	if hasFinal {
		funded = decision.Funded || decision.Outcome == "finantat"
		committeeName = decision.PanelName
		decisionDate = decision.ApprovedOn
		switch decision.Outcome {
		case "admis":
			status = "approved"
		case "finantat":
			status = "funded"
		case "rezerva", "respins":
			status = "evaluated"
		}
		if funded {
			status = "funded"
		}
	}

	tag, err := s.pool.Exec(ctx, `
		update education_merit_grants
		set
			status = $1,
			funded = $2,
			committee_name = coalesce(nullif($3, ''), committee_name),
			decision_date = coalesce($4::date, decision_date)
		where id = $5 and institution_id = $6
	`, status, funded, committeeName, decisionDate, recordID, institutionID)
	if err != nil {
		return fmt.Errorf("sync merit grant outcome: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sync merit grant outcome: %w", pgx.ErrNoRows)
	}

	return nil
}

func normalizeMobilityFinalDecisionRequest(req *CreateMobilityFinalDecisionRequest) {
	req.DecisionType = strings.TrimSpace(req.DecisionType)
	req.Outcome = strings.TrimSpace(req.Outcome)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.EffectiveFrom = strings.TrimSpace(req.EffectiveFrom)
	req.PanelName = strings.TrimSpace(req.PanelName)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.DestinationUnit = strings.TrimSpace(req.DestinationUnit)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMobilityResultIssueRequest(req *CreateMobilityResultIssueRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.RecipientRole = strings.TrimSpace(req.RecipientRole)
	req.DeliveryChannel = strings.TrimSpace(req.DeliveryChannel)
	req.DeliveryStatus = strings.TrimSpace(req.DeliveryStatus)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.DeliveredOn = strings.TrimSpace(req.DeliveredOn)
	req.RegistryReference = strings.TrimSpace(req.RegistryReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMeritFinalDecisionRequest(req *CreateMeritFinalDecisionRequest) {
	req.DecisionStage = strings.TrimSpace(req.DecisionStage)
	req.Outcome = strings.TrimSpace(req.Outcome)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.EffectiveFrom = strings.TrimSpace(req.EffectiveFrom)
	req.PanelName = strings.TrimSpace(req.PanelName)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMeritResultIssueRequest(req *CreateMeritResultIssueRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.RecipientRole = strings.TrimSpace(req.RecipientRole)
	req.DeliveryChannel = strings.TrimSpace(req.DeliveryChannel)
	req.DeliveryStatus = strings.TrimSpace(req.DeliveryStatus)
	req.IssuedOn = strings.TrimSpace(req.IssuedOn)
	req.DeliveredOn = strings.TrimSpace(req.DeliveredOn)
	req.RegistryReference = strings.TrimSpace(req.RegistryReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func buildMobilityFinalDecisionFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emfd.mobility_case_id = $1", "emfd.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["decision_code"]; value != "" {
		addContains("emfd.decision_code", value)
	}
	if value := filters["decision_type"]; value != "" {
		addContains("emfd.decision_type", value)
	}
	if value := filters["outcome"]; value != "" {
		addContains("emfd.outcome", value)
	}
	if value := filters["panel_name"]; value != "" {
		addContains("emfd.panel_name", value)
	}
	if value := filters["destination_unit"]; value != "" {
		addContains("emfd.destination_unit", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMobilityResultIssueFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emri.mobility_case_id = $1", "emri.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["issue_code"]; value != "" {
		addContains("emri.issue_code", value)
	}
	if value := filters["document_type"]; value != "" {
		addContains("emri.document_type", value)
	}
	if value := filters["recipient_name"]; value != "" {
		addContains("emri.recipient_name", value)
	}
	if value := filters["recipient_role"]; value != "" {
		addContains("emri.recipient_role", value)
	}
	if value := filters["delivery_channel"]; value != "" {
		addContains("emri.delivery_channel", value)
	}
	if value := filters["delivery_status"]; value != "" {
		addContains("emri.delivery_status", value)
	}
	if value := filters["registry_reference"]; value != "" {
		addContains("emri.registry_reference", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMeritFinalDecisionFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emfd.grant_id = $1", "emfd.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["decision_code"]; value != "" {
		addContains("emfd.decision_code", value)
	}
	if value := filters["decision_stage"]; value != "" {
		addContains("emfd.decision_stage", value)
	}
	if value := filters["outcome"]; value != "" {
		addContains("emfd.outcome", value)
	}
	if value := filters["panel_name"]; value != "" {
		addContains("emfd.panel_name", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMeritResultIssueFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emri.grant_id = $1", "emri.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["issue_code"]; value != "" {
		addContains("emri.issue_code", value)
	}
	if value := filters["document_type"]; value != "" {
		addContains("emri.document_type", value)
	}
	if value := filters["recipient_name"]; value != "" {
		addContains("emri.recipient_name", value)
	}
	if value := filters["recipient_role"]; value != "" {
		addContains("emri.recipient_role", value)
	}
	if value := filters["delivery_channel"]; value != "" {
		addContains("emri.delivery_channel", value)
	}
	if value := filters["delivery_status"]; value != "" {
		addContains("emri.delivery_status", value)
	}
	if value := filters["registry_reference"]; value != "" {
		addContains("emri.registry_reference", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func mobilityFinalDecisionSortColumn(value string) string {
	switch value {
	case "decision_code":
		return "emfd.decision_code"
	case "decision_type":
		return "emfd.decision_type"
	case "outcome":
		return "emfd.outcome"
	case "panel_name":
		return "emfd.panel_name"
	case "effective_from":
		return "emfd.effective_from"
	default:
		return "emfd.approved_on"
	}
}

func mobilityResultIssueSortColumn(value string) string {
	switch value {
	case "issue_code":
		return "emri.issue_code"
	case "document_type":
		return "emri.document_type"
	case "recipient_name":
		return "emri.recipient_name"
	case "recipient_role":
		return "emri.recipient_role"
	case "delivery_channel":
		return "emri.delivery_channel"
	case "delivery_status":
		return "emri.delivery_status"
	case "delivered_on":
		return "emri.delivered_on"
	default:
		return "emri.issued_on"
	}
}

func meritFinalDecisionSortColumn(value string) string {
	switch value {
	case "decision_code":
		return "emfd.decision_code"
	case "decision_stage":
		return "emfd.decision_stage"
	case "outcome":
		return "emfd.outcome"
	case "panel_name":
		return "emfd.panel_name"
	case "effective_from":
		return "emfd.effective_from"
	default:
		return "emfd.approved_on"
	}
}

func meritResultIssueSortColumn(value string) string {
	switch value {
	case "issue_code":
		return "emri.issue_code"
	case "document_type":
		return "emri.document_type"
	case "recipient_name":
		return "emri.recipient_name"
	case "recipient_role":
		return "emri.recipient_role"
	case "delivery_channel":
		return "emri.delivery_channel"
	case "delivery_status":
		return "emri.delivery_status"
	case "delivered_on":
		return "emri.delivered_on"
	default:
		return "emri.issued_on"
	}
}
