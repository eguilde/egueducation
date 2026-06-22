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

func (s *Service) MobilityDocuments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"document_code":     {},
		"document_type":     {},
		"stage_scope":       {},
		"document_title":    {},
		"validation_status": {},
		"submitted_by":      {},
	}, []string{"document_code", "document_type", "stage_scope", "document_title", "validation_status", "registered_on"})
	if query.Sort == "" {
		query.Sort = "registered_on"
	}
	whereClause, args := buildMobilityDocumentFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_documents emd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_documents_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			emd.id::text, emd.mobility_case_id::text, emd.document_code, emd.document_type, emd.stage_scope,
			emd.document_title, to_char(emd.registered_on, 'YYYY-MM-DD'), emd.submitted_by, emd.verified_by,
			emd.validation_status, emd.mandatory, emd.institution_id, emd.notes
		from education_mobility_documents emd
		%s
		order by %s %s, emd.document_code
		limit $%d offset $%d
	`, whereClause, mobilityDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_documents_failed"})
		return
	}
	defer rows.Close()
	items := make([]MobilityDocument, 0, query.PageSize)
	for rows.Next() {
		var item MobilityDocument
		if err := rows.Scan(&item.ID, &item.MobilityCaseID, &item.DocumentCode, &item.DocumentType, &item.StageScope, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.VerifiedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_documents_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityDocumentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MobilityDocument
	err := s.pool.QueryRow(r.Context(), `
		select id::text, mobility_case_id::text, document_code, document_type, stage_scope, document_title,
			to_char(registered_on, 'YYYY-MM-DD'), submitted_by, verified_by, validation_status, mandatory, institution_id, notes
		from education_mobility_documents
		where id = $1 and mobility_case_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.MobilityCaseID, &item.DocumentCode, &item.DocumentType, &item.StageScope, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.VerifiedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_document_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMobilityDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMobilityDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_payload"})
		return
	}
	normalizeMobilityDocumentRequest(&req)
	if req.DocumentType == "" || req.StageScope == "" || req.DocumentTitle == "" || req.RegisteredOn == "" || req.ValidationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_document_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "cerere", "adeverinta", "aviz", "fisa_evaluare", "decizie", "anexa") ||
		!inStringSet(req.StageScope, "depunere", "verificare", "sedinta", "aprobare", "emitere") ||
		!inStringSet(req.ValidationStatus, "draft", "submitted", "validated", "rejected") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_fields"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_registered_on"})
		return
	}
	code := fmt.Sprintf("MOBDOC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MobilityDocument
	err = s.pool.QueryRow(r.Context(), `
		insert into education_mobility_documents (
			mobility_case_id, document_code, document_type, stage_scope, document_title, registered_on,
			submitted_by, verified_by, validation_status, mandatory, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id::text, mobility_case_id::text, document_code, document_type, stage_scope, document_title,
			to_char(registered_on, 'YYYY-MM-DD'), submitted_by, verified_by, validation_status, mandatory, institution_id, notes
	`, recordID, code, req.DocumentType, req.StageScope, req.DocumentTitle, registeredOn, req.SubmittedBy, req.VerifiedBy, req.ValidationStatus, req.Mandatory, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.MobilityCaseID, &item.DocumentCode, &item.DocumentType, &item.StageScope, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.VerifiedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_document_create_failed"})
		return
	}
	s.logAudit(r, "education.mobility.document.create", "mobility_document", item.ID, "Mobility document created.", map[string]any{"mobility_case_id": recordID, "document_code": item.DocumentCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMobilityDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMobilityDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_payload"})
		return
	}
	normalizeMobilityDocumentRequest(&req)
	if req.DocumentType == "" || req.StageScope == "" || req.DocumentTitle == "" || req.RegisteredOn == "" || req.ValidationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_document_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "cerere", "adeverinta", "aviz", "fisa_evaluare", "decizie", "anexa") ||
		!inStringSet(req.StageScope, "depunere", "verificare", "sedinta", "aprobare", "emitere") ||
		!inStringSet(req.ValidationStatus, "draft", "submitted", "validated", "rejected") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_fields"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_document_registered_on"})
		return
	}
	var item MobilityDocument
	err = s.pool.QueryRow(r.Context(), `
		update education_mobility_documents
		set document_type=$1, stage_scope=$2, document_title=$3, registered_on=$4, submitted_by=$5,
			verified_by=$6, validation_status=$7, mandatory=$8, notes=$9, updated_at=now()
		where id=$10 and mobility_case_id=$11 and institution_id=$12
		returning id::text, mobility_case_id::text, document_code, document_type, stage_scope, document_title,
			to_char(registered_on, 'YYYY-MM-DD'), submitted_by, verified_by, validation_status, mandatory, institution_id, notes
	`, req.DocumentType, req.StageScope, req.DocumentTitle, registeredOn, req.SubmittedBy, req.VerifiedBy, req.ValidationStatus, req.Mandatory, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.DocumentCode, &item.DocumentType, &item.StageScope, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.VerifiedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_document_update_failed"})
		return
	}
	s.logAudit(r, "education.mobility.document.update", "mobility_document", item.ID, "Mobility document updated.", map[string]any{"mobility_case_id": recordID, "document_code": item.DocumentCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_mobility_documents where id = $1 and mobility_case_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_mobility_document_not_found")
		return
	}
	s.logAudit(r, "education.mobility.document.delete", "mobility_document", itemID, "Mobility document deleted.", map[string]any{"mobility_case_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MobilityCriterionScores(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"criterion_code":     {},
		"criterion_category": {},
		"criterion_label":    {},
		"validated_by":       {},
		"contested":          {},
	}, []string{"criterion_code", "criterion_category", "criterion_label", "awarded_score", "max_score"})
	if query.Sort == "" {
		query.Sort = "criterion_code"
	}
	whereClause, args := buildMobilityScoreFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_scores ems "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_scores_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, mobility_case_id::text, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
			evidence_reference, validated_by, contested, institution_id, notes
		from education_mobility_scores ems
		%s
		order by %s %s, ems.criterion_code
		limit $%d offset $%d
	`, whereClause, mobilityScoreSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_scores_failed"})
		return
	}
	defer rows.Close()
	items := make([]MobilityCriterionScore, 0, query.PageSize)
	for rows.Next() {
		var item MobilityCriterionScore
		if err := rows.Scan(&item.ID, &item.MobilityCaseID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.MaxScore, &item.AwardedScore, &item.EvidenceReference, &item.ValidatedBy, &item.Contested, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_scores_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_scores_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityCriterionScoreDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MobilityCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		select id::text, mobility_case_id::text, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
			evidence_reference, validated_by, contested, institution_id, notes
		from education_mobility_scores
		where id = $1 and mobility_case_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.MobilityCaseID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.MaxScore, &item.AwardedScore, &item.EvidenceReference, &item.ValidatedBy, &item.Contested, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_score_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_score_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMobilityCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMobilityCriterionScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_payload"})
		return
	}
	normalizeMobilityScoreRequest(&req)
	if req.CriterionCode == "" || req.CriterionLabel == "" || req.CriterionCategory == "" || req.MaxScore <= 0 || req.AwardedScore < 0 || req.AwardedScore > req.MaxScore {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "studii", "vechime", "performanta", "social", "administrativ") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_category"})
		return
	}
	var item MobilityCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		insert into education_mobility_scores (
			mobility_case_id, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
			evidence_reference, validated_by, contested, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning id::text, mobility_case_id::text, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
			evidence_reference, validated_by, contested, institution_id, notes
	`, recordID, req.CriterionCode, req.CriterionLabel, req.CriterionCategory, req.MaxScore, req.AwardedScore, req.EvidenceReference, req.ValidatedBy, req.Contested, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.MobilityCaseID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.MaxScore, &item.AwardedScore, &item.EvidenceReference, &item.ValidatedBy, &item.Contested, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_score_create_failed"})
		return
	}
	s.logAudit(r, "education.mobility.score.create", "mobility_score", item.ID, "Mobility score created.", map[string]any{"mobility_case_id": recordID, "criterion_code": item.CriterionCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMobilityCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMobilityCriterionScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_payload"})
		return
	}
	normalizeMobilityScoreRequest(&req)
	if req.CriterionCode == "" || req.CriterionLabel == "" || req.CriterionCategory == "" || req.MaxScore <= 0 || req.AwardedScore < 0 || req.AwardedScore > req.MaxScore {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "studii", "vechime", "performanta", "social", "administrativ") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_score_category"})
		return
	}
	var item MobilityCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		update education_mobility_scores
		set criterion_code=$1, criterion_label=$2, criterion_category=$3, max_score=$4, awarded_score=$5,
			evidence_reference=$6, validated_by=$7, contested=$8, notes=$9, updated_at=now()
		where id=$10 and mobility_case_id=$11 and institution_id=$12
		returning id::text, mobility_case_id::text, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
			evidence_reference, validated_by, contested, institution_id, notes
	`, req.CriterionCode, req.CriterionLabel, req.CriterionCategory, req.MaxScore, req.AwardedScore, req.EvidenceReference, req.ValidatedBy, req.Contested, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.MaxScore, &item.AwardedScore, &item.EvidenceReference, &item.ValidatedBy, &item.Contested, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_score_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_score_update_failed"})
		return
	}
	s.logAudit(r, "education.mobility.score.update", "mobility_score", item.ID, "Mobility score updated.", map[string]any{"mobility_case_id": recordID, "criterion_code": item.CriterionCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_mobility_scores where id = $1 and mobility_case_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_score_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_mobility_score_not_found")
		return
	}
	s.logAudit(r, "education.mobility.score.delete", "mobility_score", itemID, "Mobility score deleted.", map[string]any{"mobility_case_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MobilityAppeals(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"appeal_code":  {},
		"submitted_by": {},
		"status":       {},
	}, []string{"appeal_code", "submitted_by", "status", "submitted_on", "hearing_on", "resolved_on"})
	if query.Sort == "" {
		query.Sort = "submitted_on"
	}
	whereClause, args := buildMobilityAppealFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_appeals ema "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeals_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, mobility_case_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary, institution_id, notes
		from education_mobility_appeals ema
		%s
		order by %s %s, ema.submitted_on desc
		limit $%d offset $%d
	`, whereClause, mobilityAppealSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeals_failed"})
		return
	}
	defer rows.Close()
	items := make([]MobilityAppeal, 0, query.PageSize)
	for rows.Next() {
		var item MobilityAppeal
		if err := rows.Scan(&item.ID, &item.MobilityCaseID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.HearingOn, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeals_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeals_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityAppealDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MobilityAppeal
	err := s.pool.QueryRow(r.Context(), `
		select id::text, mobility_case_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary, institution_id, notes
		from education_mobility_appeals
		where id = $1 and mobility_case_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.MobilityCaseID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.HearingOn, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeal_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMobilityAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMobilityAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_payload"})
		return
	}
	normalizeMobilityAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_submitted_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_resolved_on"})
		return
	}
	code := fmt.Sprintf("MOBAPL-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MobilityAppeal
	err = s.pool.QueryRow(r.Context(), `
		insert into education_mobility_appeals (
			mobility_case_id, appeal_code, submitted_by, submitted_on, status, grounds, hearing_on, resolved_on, decision_summary, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning id::text, mobility_case_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary, institution_id, notes
	`, recordID, code, req.SubmittedBy, submittedOn, req.Status, req.Grounds, hearingOn, resolvedOn, req.DecisionSummary, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.MobilityCaseID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.HearingOn, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_appeal_create_failed"})
		return
	}
	s.logAudit(r, "education.mobility.appeal.create", "mobility_appeal", item.ID, "Mobility appeal created.", map[string]any{"mobility_case_id": recordID, "appeal_code": item.AppealCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMobilityAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMobilityAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_payload"})
		return
	}
	normalizeMobilityAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_submitted_on"})
		return
	}
	hearingOn, err := parseOptionalEducationDate(req.HearingOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_hearing_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_appeal_resolved_on"})
		return
	}
	var item MobilityAppeal
	err = s.pool.QueryRow(r.Context(), `
		update education_mobility_appeals
		set submitted_by=$1, submitted_on=$2, status=$3, grounds=$4, hearing_on=$5, resolved_on=$6, decision_summary=$7, notes=$8, updated_at=now()
		where id=$9 and mobility_case_id=$10 and institution_id=$11
		returning id::text, mobility_case_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(hearing_on, 'YYYY-MM-DD'), ''), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''),
			decision_summary, institution_id, notes
	`, req.SubmittedBy, submittedOn, req.Status, req.Grounds, hearingOn, resolvedOn, req.DecisionSummary, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.MobilityCaseID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.HearingOn, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_appeal_update_failed"})
		return
	}
	s.logAudit(r, "education.mobility.appeal.update", "mobility_appeal", item.ID, "Mobility appeal updated.", map[string]any{"mobility_case_id": recordID, "appeal_code": item.AppealCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_mobility_appeals where id = $1 and mobility_case_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_appeal_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_mobility_appeal_not_found")
		return
	}
	s.logAudit(r, "education.mobility.appeal.delete", "mobility_appeal", itemID, "Mobility appeal deleted.", map[string]any{"mobility_case_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MeritDocuments(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"document_code":     {},
		"document_type":     {},
		"document_title":    {},
		"validation_status": {},
		"submitted_by":      {},
	}, []string{"document_code", "document_type", "document_title", "validation_status", "registered_on"})
	if query.Sort == "" {
		query.Sort = "registered_on"
	}
	whereClause, args := buildMeritDocumentFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_documents emd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_documents_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, grant_id::text, document_code, document_type, document_title, to_char(registered_on, 'YYYY-MM-DD'),
			submitted_by, validation_status, mandatory, institution_id, notes
		from education_merit_documents emd
		%s
		order by %s %s, emd.document_code
		limit $%d offset $%d
	`, whereClause, meritDocumentSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_documents_failed"})
		return
	}
	defer rows.Close()
	items := make([]MeritDocument, 0, query.PageSize)
	for rows.Next() {
		var item MeritDocument
		if err := rows.Scan(&item.ID, &item.GrantID, &item.DocumentCode, &item.DocumentType, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_documents_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_documents_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritDocumentDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MeritDocument
	err := s.pool.QueryRow(r.Context(), `
		select id::text, grant_id::text, document_code, document_type, document_title, to_char(registered_on, 'YYYY-MM-DD'),
			submitted_by, validation_status, mandatory, institution_id, notes
		from education_merit_documents
		where id = $1 and grant_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.GrantID, &item.DocumentCode, &item.DocumentType, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_document_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMeritDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMeritDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_payload"})
		return
	}
	normalizeMeritDocumentRequest(&req)
	if req.DocumentType == "" || req.DocumentTitle == "" || req.RegisteredOn == "" || req.ValidationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_document_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "cerere", "declaratie", "autoevaluare", "adeverinta", "portofoliu", "anexa") ||
		!inStringSet(req.ValidationStatus, "draft", "submitted", "validated", "rejected") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_fields"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_registered_on"})
		return
	}
	code := fmt.Sprintf("GRDOC-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MeritDocument
	err = s.pool.QueryRow(r.Context(), `
		insert into education_merit_documents (
			grant_id, document_code, document_type, document_title, registered_on, submitted_by, validation_status, mandatory, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning id::text, grant_id::text, document_code, document_type, document_title, to_char(registered_on, 'YYYY-MM-DD'),
			submitted_by, validation_status, mandatory, institution_id, notes
	`, recordID, code, req.DocumentType, req.DocumentTitle, registeredOn, req.SubmittedBy, req.ValidationStatus, req.Mandatory, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.GrantID, &item.DocumentCode, &item.DocumentType, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_document_create_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.document.create", "merit_document", item.ID, "Merit document created.", map[string]any{"grant_id": recordID, "document_code": item.DocumentCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMeritDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMeritDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_payload"})
		return
	}
	normalizeMeritDocumentRequest(&req)
	if req.DocumentType == "" || req.DocumentTitle == "" || req.RegisteredOn == "" || req.ValidationStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_document_fields"})
		return
	}
	if !inStringSet(req.DocumentType, "cerere", "declaratie", "autoevaluare", "adeverinta", "portofoliu", "anexa") ||
		!inStringSet(req.ValidationStatus, "draft", "submitted", "validated", "rejected") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_fields"})
		return
	}
	registeredOn, err := parseRequiredEducationDate(req.RegisteredOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_document_registered_on"})
		return
	}
	var item MeritDocument
	err = s.pool.QueryRow(r.Context(), `
		update education_merit_documents
		set document_type=$1, document_title=$2, registered_on=$3, submitted_by=$4, validation_status=$5, mandatory=$6, notes=$7, updated_at=now()
		where id=$8 and grant_id=$9 and institution_id=$10
		returning id::text, grant_id::text, document_code, document_type, document_title, to_char(registered_on, 'YYYY-MM-DD'),
			submitted_by, validation_status, mandatory, institution_id, notes
	`, req.DocumentType, req.DocumentTitle, registeredOn, req.SubmittedBy, req.ValidationStatus, req.Mandatory, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.DocumentCode, &item.DocumentType, &item.DocumentTitle, &item.RegisteredOn, &item.SubmittedBy, &item.ValidationStatus, &item.Mandatory, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_document_update_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.document.update", "merit_document", item.ID, "Merit document updated.", map[string]any{"grant_id": recordID, "document_code": item.DocumentCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritDocument(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_merit_documents where id = $1 and grant_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_document_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_merit_document_not_found")
		return
	}
	s.logAudit(r, "education.gradatii.document.delete", "merit_document", itemID, "Merit document deleted.", map[string]any{"grant_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MeritCriterionScores(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"criterion_code":     {},
		"criterion_category": {},
		"panel_stage":        {},
		"reviewer_name":      {},
		"contested":          {},
	}, []string{"criterion_code", "criterion_category", "panel_stage", "awarded_score", "max_score"})
	if query.Sort == "" {
		query.Sort = "criterion_code"
	}
	whereClause, args := buildMeritScoreFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_scores ems "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_scores_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, grant_id::text, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
			reviewer_name, evidence_reference, contested, institution_id, notes
		from education_merit_scores ems
		%s
		order by %s %s, ems.criterion_code
		limit $%d offset $%d
	`, whereClause, meritScoreSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_scores_failed"})
		return
	}
	defer rows.Close()
	items := make([]MeritCriterionScore, 0, query.PageSize)
	for rows.Next() {
		var item MeritCriterionScore
		if err := rows.Scan(&item.ID, &item.GrantID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.PanelStage, &item.MaxScore, &item.AwardedScore, &item.ReviewerName, &item.EvidenceReference, &item.Contested, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_scores_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_scores_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritCriterionScoreDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MeritCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		select id::text, grant_id::text, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
			reviewer_name, evidence_reference, contested, institution_id, notes
		from education_merit_scores
		where id = $1 and grant_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.GrantID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.PanelStage, &item.MaxScore, &item.AwardedScore, &item.ReviewerName, &item.EvidenceReference, &item.Contested, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_score_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_score_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMeritCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMeritCriterionScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_payload"})
		return
	}
	normalizeMeritScoreRequest(&req)
	if req.CriterionCode == "" || req.CriterionLabel == "" || req.CriterionCategory == "" || req.PanelStage == "" || req.MaxScore <= 0 || req.AwardedScore < 0 || req.AwardedScore > req.MaxScore {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "performanta", "impact", "dezvoltare", "management", "incluziune") ||
		!inStringSet(req.PanelStage, "autoevaluare", "evaluare_comisie", "validare_finala") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_fields"})
		return
	}
	var item MeritCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		insert into education_merit_scores (
			grant_id, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
			reviewer_name, evidence_reference, contested, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning id::text, grant_id::text, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
			reviewer_name, evidence_reference, contested, institution_id, notes
	`, recordID, req.CriterionCode, req.CriterionLabel, req.CriterionCategory, req.PanelStage, req.MaxScore, req.AwardedScore, req.ReviewerName, req.EvidenceReference, req.Contested, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.GrantID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.PanelStage, &item.MaxScore, &item.AwardedScore, &item.ReviewerName, &item.EvidenceReference, &item.Contested, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_score_create_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.score.create", "merit_score", item.ID, "Merit score created.", map[string]any{"grant_id": recordID, "criterion_code": item.CriterionCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMeritCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMeritCriterionScoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_payload"})
		return
	}
	normalizeMeritScoreRequest(&req)
	if req.CriterionCode == "" || req.CriterionLabel == "" || req.CriterionCategory == "" || req.PanelStage == "" || req.MaxScore <= 0 || req.AwardedScore < 0 || req.AwardedScore > req.MaxScore {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "performanta", "impact", "dezvoltare", "management", "incluziune") ||
		!inStringSet(req.PanelStage, "autoevaluare", "evaluare_comisie", "validare_finala") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score_fields"})
		return
	}
	var item MeritCriterionScore
	err := s.pool.QueryRow(r.Context(), `
		update education_merit_scores
		set criterion_code=$1, criterion_label=$2, criterion_category=$3, panel_stage=$4, max_score=$5, awarded_score=$6,
			reviewer_name=$7, evidence_reference=$8, contested=$9, notes=$10, updated_at=now()
		where id=$11 and grant_id=$12 and institution_id=$13
		returning id::text, grant_id::text, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
			reviewer_name, evidence_reference, contested, institution_id, notes
	`, req.CriterionCode, req.CriterionLabel, req.CriterionCategory, req.PanelStage, req.MaxScore, req.AwardedScore, req.ReviewerName, req.EvidenceReference, req.Contested, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.CriterionCode, &item.CriterionLabel, &item.CriterionCategory, &item.PanelStage, &item.MaxScore, &item.AwardedScore, &item.ReviewerName, &item.EvidenceReference, &item.Contested, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_score_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_score_update_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.score.update", "merit_score", item.ID, "Merit score updated.", map[string]any{"grant_id": recordID, "criterion_code": item.CriterionCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritCriterionScore(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_merit_scores where id = $1 and grant_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_score_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_merit_score_not_found")
		return
	}
	s.logAudit(r, "education.gradatii.score.delete", "merit_score", itemID, "Merit score deleted.", map[string]any{"grant_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) MeritAppeals(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"appeal_code":  {},
		"submitted_by": {},
		"status":       {},
	}, []string{"appeal_code", "submitted_by", "status", "submitted_on", "resolved_on"})
	if query.Sort == "" {
		query.Sort = "submitted_on"
	}
	whereClause, args := buildMeritAppealFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_appeals ema "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeals_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select id::text, grant_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''), decision_summary, institution_id, notes
		from education_merit_appeals ema
		%s
		order by %s %s, ema.submitted_on desc
		limit $%d offset $%d
	`, whereClause, meritAppealSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeals_failed"})
		return
	}
	defer rows.Close()
	items := make([]MeritAppeal, 0, query.PageSize)
	for rows.Next() {
		var item MeritAppeal
		if err := rows.Scan(&item.ID, &item.GrantID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeals_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeals_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritAppealDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item MeritAppeal
	err := s.pool.QueryRow(r.Context(), `
		select id::text, grant_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''), decision_summary, institution_id, notes
		from education_merit_appeals
		where id = $1 and grant_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(&item.ID, &item.GrantID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeal_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateMeritAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreateMeritAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_payload"})
		return
	}
	normalizeMeritAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_submitted_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_resolved_on"})
		return
	}
	code := fmt.Sprintf("GRAPL-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item MeritAppeal
	err = s.pool.QueryRow(r.Context(), `
		insert into education_merit_appeals (
			grant_id, appeal_code, submitted_by, submitted_on, status, grounds, resolved_on, decision_summary, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning id::text, grant_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''), decision_summary, institution_id, notes
	`, recordID, code, req.SubmittedBy, submittedOn, req.Status, req.Grounds, resolvedOn, req.DecisionSummary, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.GrantID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_appeal_create_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.appeal.create", "merit_appeal", item.ID, "Merit appeal created.", map[string]any{"grant_id": recordID, "appeal_code": item.AppealCode})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateMeritAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreateMeritAppealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_payload"})
		return
	}
	normalizeMeritAppealRequest(&req)
	if req.SubmittedBy == "" || req.SubmittedOn == "" || req.Status == "" || req.Grounds == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_appeal_fields"})
		return
	}
	if !inStringSet(req.Status, "submitted", "review", "accepted", "rejected", "resolved") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_status"})
		return
	}
	submittedOn, err := parseRequiredEducationDate(req.SubmittedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_submitted_on"})
		return
	}
	resolvedOn, err := parseOptionalEducationDate(req.ResolvedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_appeal_resolved_on"})
		return
	}
	var item MeritAppeal
	err = s.pool.QueryRow(r.Context(), `
		update education_merit_appeals
		set submitted_by=$1, submitted_on=$2, status=$3, grounds=$4, resolved_on=$5, decision_summary=$6, notes=$7, updated_at=now()
		where id=$8 and grant_id=$9 and institution_id=$10
		returning id::text, grant_id::text, appeal_code, submitted_by, to_char(submitted_on, 'YYYY-MM-DD'),
			status, grounds, coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''), decision_summary, institution_id, notes
	`, req.SubmittedBy, submittedOn, req.Status, req.Grounds, resolvedOn, req.DecisionSummary, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.GrantID, &item.AppealCode, &item.SubmittedBy, &item.SubmittedOn, &item.Status, &item.Grounds, &item.ResolvedOn, &item.DecisionSummary, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_appeal_update_failed"})
		return
	}
	s.logAudit(r, "education.gradatii.appeal.update", "merit_appeal", item.ID, "Merit appeal updated.", map[string]any{"grant_id": recordID, "appeal_code": item.AppealCode})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritAppeal(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_merit_appeals where id = $1 and grant_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_appeal_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_merit_appeal_not_found")
		return
	}
	s.logAudit(r, "education.gradatii.appeal.delete", "merit_appeal", itemID, "Merit appeal deleted.", map[string]any{"grant_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func buildMobilityDocumentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emd.mobility_case_id = $1", "emd.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"document_code":     "emd.document_code",
		"document_type":     "emd.document_type",
		"stage_scope":       "emd.stage_scope",
		"document_title":    "emd.document_title",
		"validation_status": "emd.validation_status",
		"submitted_by":      "emd.submitted_by",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMobilityScoreFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"ems.mobility_case_id = $1", "ems.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"criterion_code":     "ems.criterion_code",
		"criterion_category": "ems.criterion_category",
		"criterion_label":    "ems.criterion_label",
		"validated_by":       "ems.validated_by",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	if value := strings.TrimSpace(filters["contested"]); value != "" {
		args = append(args, value == "true")
		where = append(where, fmt.Sprintf("ems.contested = $%d", len(args)))
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMobilityAppealFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"ema.mobility_case_id = $1", "ema.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"appeal_code":  "ema.appeal_code",
		"submitted_by": "ema.submitted_by",
		"status":       "ema.status",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMeritDocumentFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"emd.grant_id = $1", "emd.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"document_code":     "emd.document_code",
		"document_type":     "emd.document_type",
		"document_title":    "emd.document_title",
		"validation_status": "emd.validation_status",
		"submitted_by":      "emd.submitted_by",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMeritScoreFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"ems.grant_id = $1", "ems.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"criterion_code":     "ems.criterion_code",
		"criterion_category": "ems.criterion_category",
		"panel_stage":        "ems.panel_stage",
		"reviewer_name":      "ems.reviewer_name",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	if value := strings.TrimSpace(filters["contested"]); value != "" {
		args = append(args, value == "true")
		where = append(where, fmt.Sprintf("ems.contested = $%d", len(args)))
	}
	return "where " + strings.Join(where, " and "), args
}

func buildMeritAppealFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"ema.grant_id = $1", "ema.institution_id = $2"}
	args := []any{recordID, institutionID}
	for key, column := range map[string]string{
		"appeal_code":  "ema.appeal_code",
		"submitted_by": "ema.submitted_by",
		"status":       "ema.status",
	} {
		if value := strings.TrimSpace(filters[key]); value != "" {
			args = append(args, "%"+strings.ToLower(value)+"%")
			where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
		}
	}
	return "where " + strings.Join(where, " and "), args
}

func mobilityDocumentSortColumn(value string) string {
	switch value {
	case "document_code":
		return "emd.document_code"
	case "document_type":
		return "emd.document_type"
	case "stage_scope":
		return "emd.stage_scope"
	case "document_title":
		return "emd.document_title"
	case "validation_status":
		return "emd.validation_status"
	case "registered_on":
		return "emd.registered_on"
	default:
		return "emd.registered_on"
	}
}

func mobilityScoreSortColumn(value string) string {
	switch value {
	case "criterion_code":
		return "ems.criterion_code"
	case "criterion_category":
		return "ems.criterion_category"
	case "criterion_label":
		return "ems.criterion_label"
	case "max_score":
		return "ems.max_score"
	case "awarded_score":
		return "ems.awarded_score"
	default:
		return "ems.criterion_code"
	}
}

func mobilityAppealSortColumn(value string) string {
	switch value {
	case "appeal_code":
		return "ema.appeal_code"
	case "submitted_by":
		return "ema.submitted_by"
	case "status":
		return "ema.status"
	case "submitted_on":
		return "ema.submitted_on"
	case "hearing_on":
		return "ema.hearing_on"
	case "resolved_on":
		return "ema.resolved_on"
	default:
		return "ema.submitted_on"
	}
}

func meritDocumentSortColumn(value string) string {
	switch value {
	case "document_code":
		return "emd.document_code"
	case "document_type":
		return "emd.document_type"
	case "document_title":
		return "emd.document_title"
	case "validation_status":
		return "emd.validation_status"
	case "registered_on":
		return "emd.registered_on"
	default:
		return "emd.registered_on"
	}
}

func meritScoreSortColumn(value string) string {
	switch value {
	case "criterion_code":
		return "ems.criterion_code"
	case "criterion_category":
		return "ems.criterion_category"
	case "panel_stage":
		return "ems.panel_stage"
	case "max_score":
		return "ems.max_score"
	case "awarded_score":
		return "ems.awarded_score"
	default:
		return "ems.criterion_code"
	}
}

func meritAppealSortColumn(value string) string {
	switch value {
	case "appeal_code":
		return "ema.appeal_code"
	case "submitted_by":
		return "ema.submitted_by"
	case "status":
		return "ema.status"
	case "submitted_on":
		return "ema.submitted_on"
	case "resolved_on":
		return "ema.resolved_on"
	default:
		return "ema.submitted_on"
	}
}

func normalizeMobilityDocumentRequest(req *CreateMobilityDocumentRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.StageScope = strings.TrimSpace(req.StageScope)
	req.DocumentTitle = strings.TrimSpace(req.DocumentTitle)
	req.RegisteredOn = strings.TrimSpace(req.RegisteredOn)
	req.SubmittedBy = strings.TrimSpace(req.SubmittedBy)
	req.VerifiedBy = strings.TrimSpace(req.VerifiedBy)
	req.ValidationStatus = strings.TrimSpace(req.ValidationStatus)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMobilityScoreRequest(req *CreateMobilityCriterionScoreRequest) {
	req.CriterionCode = strings.TrimSpace(req.CriterionCode)
	req.CriterionLabel = strings.TrimSpace(req.CriterionLabel)
	req.CriterionCategory = strings.TrimSpace(req.CriterionCategory)
	req.EvidenceReference = strings.TrimSpace(req.EvidenceReference)
	req.ValidatedBy = strings.TrimSpace(req.ValidatedBy)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMobilityAppealRequest(req *CreateMobilityAppealRequest) {
	req.SubmittedBy = strings.TrimSpace(req.SubmittedBy)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.Status = strings.TrimSpace(req.Status)
	req.Grounds = strings.TrimSpace(req.Grounds)
	req.HearingOn = strings.TrimSpace(req.HearingOn)
	req.ResolvedOn = strings.TrimSpace(req.ResolvedOn)
	req.DecisionSummary = strings.TrimSpace(req.DecisionSummary)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMeritDocumentRequest(req *CreateMeritDocumentRequest) {
	req.DocumentType = strings.TrimSpace(req.DocumentType)
	req.DocumentTitle = strings.TrimSpace(req.DocumentTitle)
	req.RegisteredOn = strings.TrimSpace(req.RegisteredOn)
	req.SubmittedBy = strings.TrimSpace(req.SubmittedBy)
	req.ValidationStatus = strings.TrimSpace(req.ValidationStatus)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMeritScoreRequest(req *CreateMeritCriterionScoreRequest) {
	req.CriterionCode = strings.TrimSpace(req.CriterionCode)
	req.CriterionLabel = strings.TrimSpace(req.CriterionLabel)
	req.CriterionCategory = strings.TrimSpace(req.CriterionCategory)
	req.PanelStage = strings.TrimSpace(req.PanelStage)
	req.ReviewerName = strings.TrimSpace(req.ReviewerName)
	req.EvidenceReference = strings.TrimSpace(req.EvidenceReference)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeMeritAppealRequest(req *CreateMeritAppealRequest) {
	req.SubmittedBy = strings.TrimSpace(req.SubmittedBy)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.Status = strings.TrimSpace(req.Status)
	req.Grounds = strings.TrimSpace(req.Grounds)
	req.ResolvedOn = strings.TrimSpace(req.ResolvedOn)
	req.DecisionSummary = strings.TrimSpace(req.DecisionSummary)
	req.Notes = strings.TrimSpace(req.Notes)
}
