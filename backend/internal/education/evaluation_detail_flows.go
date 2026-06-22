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

func (s *Service) EvaluationSelfReviews(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"review_code":    {},
		"section_title":  {},
		"narrative_type": {},
		"status":         {},
	}, []string{"review_code", "section_title", "narrative_type", "status", "completed_on", "assumed_score"})
	if query.Sort == "" {
		query.Sort = "completed_on"
	}

	whereClause, args := buildEvaluationSelfReviewFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_evaluation_self_reviews eesr "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_self_reviews_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			eesr.id::text, eesr.evaluation_id::text, eesr.review_code, eesr.section_title, eesr.narrative_type,
			eesr.status, to_char(eesr.completed_on, 'YYYY-MM-DD'), eesr.evidence_summary,
			eesr.strengths, eesr.improvement_needs, eesr.assumed_score, eesr.institution_id, eesr.notes
		from education_evaluation_self_reviews eesr
		%s
		order by %s %s, eesr.review_code desc
		limit $%d offset $%d
	`, whereClause, evaluationSelfReviewSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_self_reviews_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelEvaluationSelfReview, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelEvaluationSelfReview
		if err := rows.Scan(
			&item.ID, &item.EvaluationID, &item.ReviewCode, &item.SectionTitle, &item.NarrativeType,
			&item.Status, &item.CompletedOn, &item.EvidenceSummary, &item.Strengths, &item.ImprovementNeeds,
			&item.AssumedScore, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_self_reviews_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_self_reviews_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EvaluationSelfReviewDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PersonnelEvaluationSelfReview
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text, evaluation_id::text, review_code, section_title, narrative_type,
			status, to_char(completed_on, 'YYYY-MM-DD'), evidence_summary, strengths,
			improvement_needs, assumed_score, institution_id, notes
		from education_evaluation_self_reviews
		where id = $1 and evaluation_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.EvaluationID, &item.ReviewCode, &item.SectionTitle, &item.NarrativeType,
		&item.Status, &item.CompletedOn, &item.EvidenceSummary, &item.Strengths, &item.ImprovementNeeds,
		&item.AssumedScore, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_self_review_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_self_review_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateEvaluationSelfReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelEvaluationSelfReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_payload"})
		return
	}
	normalizeEvaluationSelfReviewRequest(&req)
	if req.SectionTitle == "" || req.NarrativeType == "" || req.Status == "" || req.CompletedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_self_review_fields"})
		return
	}
	if !inStringSet(req.NarrativeType, "autoevaluare", "performanta", "dezvoltare", "impact") ||
		!inStringSet(req.Status, "draft", "submitted", "validated", "returned") ||
		req.AssumedScore < 0 || req.AssumedScore > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_fields"})
		return
	}
	completedOn, err := parseRequiredEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_completed_on"})
		return
	}

	code := fmt.Sprintf("AUTO-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelEvaluationSelfReview
	err = s.pool.QueryRow(r.Context(), `
		insert into education_evaluation_self_reviews (
			evaluation_id, review_code, section_title, narrative_type, status, completed_on,
			evidence_summary, strengths, improvement_needs, assumed_score, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning
			id::text, evaluation_id::text, review_code, section_title, narrative_type,
			status, to_char(completed_on, 'YYYY-MM-DD'), evidence_summary, strengths,
			improvement_needs, assumed_score, institution_id, notes
	`, recordID, code, req.SectionTitle, req.NarrativeType, req.Status, completedOn, req.EvidenceSummary, req.Strengths, req.ImprovementNeeds, req.AssumedScore, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.EvaluationID, &item.ReviewCode, &item.SectionTitle, &item.NarrativeType,
		&item.Status, &item.CompletedOn, &item.EvidenceSummary, &item.Strengths, &item.ImprovementNeeds,
		&item.AssumedScore, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_self_review_create_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.self_review.create", "evaluation_self_review", item.ID, "Evaluation self review created.", map[string]any{
		"evaluation_id":  recordID,
		"review_code":    item.ReviewCode,
		"narrative_type": item.NarrativeType,
		"status":         item.Status,
		"assumed_score":  item.AssumedScore,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateEvaluationSelfReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePersonnelEvaluationSelfReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_payload"})
		return
	}
	normalizeEvaluationSelfReviewRequest(&req)
	if req.SectionTitle == "" || req.NarrativeType == "" || req.Status == "" || req.CompletedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_self_review_fields"})
		return
	}
	if !inStringSet(req.NarrativeType, "autoevaluare", "performanta", "dezvoltare", "impact") ||
		!inStringSet(req.Status, "draft", "submitted", "validated", "returned") ||
		req.AssumedScore < 0 || req.AssumedScore > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_fields"})
		return
	}
	completedOn, err := parseRequiredEducationDate(req.CompletedOn)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_self_review_completed_on"})
		return
	}

	var item PersonnelEvaluationSelfReview
	err = s.pool.QueryRow(r.Context(), `
		update education_evaluation_self_reviews
		set
			section_title = $1,
			narrative_type = $2,
			status = $3,
			completed_on = $4,
			evidence_summary = $5,
			strengths = $6,
			improvement_needs = $7,
			assumed_score = $8,
			notes = $9,
			updated_at = now()
		where id = $10 and evaluation_id = $11 and institution_id = $12
		returning
			id::text, evaluation_id::text, review_code, section_title, narrative_type,
			status, to_char(completed_on, 'YYYY-MM-DD'), evidence_summary, strengths,
			improvement_needs, assumed_score, institution_id, notes
	`, req.SectionTitle, req.NarrativeType, req.Status, completedOn, req.EvidenceSummary, req.Strengths, req.ImprovementNeeds, req.AssumedScore, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.EvaluationID, &item.ReviewCode, &item.SectionTitle, &item.NarrativeType,
		&item.Status, &item.CompletedOn, &item.EvidenceSummary, &item.Strengths, &item.ImprovementNeeds,
		&item.AssumedScore, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_self_review_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_self_review_update_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.self_review.update", "evaluation_self_review", item.ID, "Evaluation self review updated.", map[string]any{
		"evaluation_id":  recordID,
		"review_code":    item.ReviewCode,
		"narrative_type": item.NarrativeType,
		"status":         item.Status,
		"assumed_score":  item.AssumedScore,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteEvaluationSelfReview(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_evaluation_self_reviews where id = $1 and evaluation_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_self_review_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_evaluation_self_review_not_found")
		return
	}

	s.logAudit(r, "education.evaluations.self_review.delete", "evaluation_self_review", itemID, "Evaluation self review deleted.", map[string]any{"evaluation_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) EvaluationCriteria(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"criterion_code":     {},
		"criterion_category": {},
		"criterion_label":    {},
		"status":             {},
	}, []string{"criterion_code", "criterion_category", "criterion_label", "status", "max_score", "final_score"})
	if query.Sort == "" {
		query.Sort = "criterion_code"
	}

	whereClause, args := buildEvaluationCriteriaFilters(query.Filters, recordID, s.institutionID(r))
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_evaluation_criteria eec "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_criteria_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select
			eec.id::text, eec.evaluation_id::text, eec.criterion_code, eec.criterion_category, eec.criterion_label,
			eec.max_score, eec.self_score, eec.reviewer_score, eec.final_score, eec.status,
			eec.evidence_summary, eec.institution_id, eec.notes
		from education_evaluation_criteria eec
		%s
		order by %s %s, eec.criterion_code asc
		limit $%d offset $%d
	`, whereClause, evaluationCriteriaSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_criteria_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelEvaluationCriterion, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelEvaluationCriterion
		if err := rows.Scan(
			&item.ID, &item.EvaluationID, &item.CriterionCode, &item.CriterionCategory, &item.CriterionLabel,
			&item.MaxScore, &item.SelfScore, &item.ReviewerScore, &item.FinalScore, &item.Status,
			&item.EvidenceSummary, &item.InstitutionID, &item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_criteria_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_criteria_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EvaluationCriterionDetail(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var item PersonnelEvaluationCriterion
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text, evaluation_id::text, criterion_code, criterion_category, criterion_label,
			max_score, self_score, reviewer_score, final_score, status,
			evidence_summary, institution_id, notes
		from education_evaluation_criteria
		where id = $1 and evaluation_id = $2 and institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.EvaluationID, &item.CriterionCode, &item.CriterionCategory, &item.CriterionLabel,
		&item.MaxScore, &item.SelfScore, &item.ReviewerScore, &item.FinalScore, &item.Status,
		&item.EvidenceSummary, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_criterion_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_criterion_detail_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) CreateEvaluationCriterion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	var req CreatePersonnelEvaluationCriterionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_payload"})
		return
	}
	normalizeEvaluationCriterionRequest(&req)
	if req.CriterionCategory == "" || req.CriterionLabel == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_criterion_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "proiectare", "predare", "evaluare", "management_clasa", "dezvoltare", "parteneriat") ||
		!inStringSet(req.Status, "draft", "reviewed", "validated", "contested") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_fields"})
		return
	}
	if !validEvaluationCriterionScores(req.MaxScore, req.SelfScore, req.ReviewerScore, req.FinalScore) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_scores"})
		return
	}

	code := fmt.Sprintf("CRIT-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
	var item PersonnelEvaluationCriterion
	err := s.pool.QueryRow(r.Context(), `
		insert into education_evaluation_criteria (
			evaluation_id, criterion_code, criterion_category, criterion_label, max_score,
			self_score, reviewer_score, final_score, status, evidence_summary, institution_id, notes
		) values ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning
			id::text, evaluation_id::text, criterion_code, criterion_category, criterion_label,
			max_score, self_score, reviewer_score, final_score, status, evidence_summary, institution_id, notes
	`, recordID, code, req.CriterionCategory, req.CriterionLabel, req.MaxScore, req.SelfScore, req.ReviewerScore, req.FinalScore, req.Status, req.EvidenceSummary, s.institutionID(r), req.Notes).Scan(
		&item.ID, &item.EvaluationID, &item.CriterionCode, &item.CriterionCategory, &item.CriterionLabel,
		&item.MaxScore, &item.SelfScore, &item.ReviewerScore, &item.FinalScore, &item.Status,
		&item.EvidenceSummary, &item.InstitutionID, &item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_create_failed"})
		return
	}
	if err := s.syncEvaluationScore(r, recordID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_sync_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.criterion.create", "evaluation_criterion", item.ID, "Evaluation criterion created.", map[string]any{
		"evaluation_id":      recordID,
		"criterion_code":     item.CriterionCode,
		"criterion_category": item.CriterionCategory,
		"final_score":        item.FinalScore,
		"status":             item.Status,
	})
	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateEvaluationCriterion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req CreatePersonnelEvaluationCriterionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_payload"})
		return
	}
	normalizeEvaluationCriterionRequest(&req)
	if req.CriterionCategory == "" || req.CriterionLabel == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_criterion_fields"})
		return
	}
	if !inStringSet(req.CriterionCategory, "proiectare", "predare", "evaluare", "management_clasa", "dezvoltare", "parteneriat") ||
		!inStringSet(req.Status, "draft", "reviewed", "validated", "contested") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_fields"})
		return
	}
	if !validEvaluationCriterionScores(req.MaxScore, req.SelfScore, req.ReviewerScore, req.FinalScore) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_criterion_scores"})
		return
	}

	var item PersonnelEvaluationCriterion
	err := s.pool.QueryRow(r.Context(), `
		update education_evaluation_criteria
		set
			criterion_category = $1,
			criterion_label = $2,
			max_score = $3,
			self_score = $4,
			reviewer_score = $5,
			final_score = $6,
			status = $7,
			evidence_summary = $8,
			notes = $9,
			updated_at = now()
		where id = $10 and evaluation_id = $11 and institution_id = $12
		returning
			id::text, evaluation_id::text, criterion_code, criterion_category, criterion_label,
			max_score, self_score, reviewer_score, final_score, status, evidence_summary, institution_id, notes
	`, req.CriterionCategory, req.CriterionLabel, req.MaxScore, req.SelfScore, req.ReviewerScore, req.FinalScore, req.Status, req.EvidenceSummary, req.Notes, itemID, recordID, s.institutionID(r)).Scan(
		&item.ID, &item.EvaluationID, &item.CriterionCode, &item.CriterionCategory, &item.CriterionLabel,
		&item.MaxScore, &item.SelfScore, &item.ReviewerScore, &item.FinalScore, &item.Status,
		&item.EvidenceSummary, &item.InstitutionID, &item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_criterion_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_update_failed"})
		return
	}
	if err := s.syncEvaluationScore(r, recordID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_sync_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.criterion.update", "evaluation_criterion", item.ID, "Evaluation criterion updated.", map[string]any{
		"evaluation_id":      recordID,
		"criterion_code":     item.CriterionCode,
		"criterion_category": item.CriterionCategory,
		"final_score":        item.FinalScore,
		"status":             item.Status,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteEvaluationCriterion(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	tag, err := s.pool.Exec(r.Context(), `delete from education_evaluation_criteria where id = $1 and evaluation_id = $2 and institution_id = $3`, itemID, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_delete_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		writeEducationNotFound(w, "education_evaluation_criterion_not_found")
		return
	}
	if err := s.syncEvaluationScore(r, recordID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_criterion_sync_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.criterion.delete", "evaluation_criterion", itemID, "Evaluation criterion deleted.", map[string]any{"evaluation_id": recordID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) syncEvaluationScore(r *http.Request, recordID string) error {
	var score float64
	if err := s.pool.QueryRow(r.Context(), `
		select coalesce(sum(final_score), 0)
		from education_evaluation_criteria
		where evaluation_id = $1 and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(&score); err != nil {
		return fmt.Errorf("calculate evaluation score: %w", err)
	}
	qualification := evaluationQualification(score)
	if _, err := s.pool.Exec(r.Context(), `
		update education_evaluations
		set score = $1, qualification = $2, updated_at = now()
		where id = $3 and institution_id = $4
	`, score, qualification, recordID, s.institutionID(r)); err != nil {
		return fmt.Errorf("update evaluation score: %w", err)
	}
	return nil
}

func (s *Service) syncEvaluationDocumentToPersonnelFile(ctx context.Context, evaluationID string, institutionID string) error {
	evaluation, err := s.loadEvaluationFileContext(ctx, evaluationID, institutionID)
	if err != nil {
		return err
	}
	if evaluation == nil || evaluation.PersonnelID == "" {
		return nil
	}

	if evaluation.FinalizedOn == nil {
		if _, err := s.pool.Exec(ctx, `
			delete from education_personnel_file_documents
			where personnel_id = $1 and institution_id = $2 and file_reference = $3 and document_category = 'evaluare'
		`, evaluation.PersonnelID, institutionID, evaluation.EvaluationCode); err != nil {
			return fmt.Errorf("delete mirrored evaluation document: %w", err)
		}
		return nil
	}

	title := fmt.Sprintf("Fisa evaluare anuala %s", evaluation.SchoolYear)
	fileScope := personnelFileScope(evaluation.PersonnelRoleTitle, evaluation.RoleTitle)
	var existingID string
	err = s.pool.QueryRow(ctx, `
		select id::text
		from education_personnel_file_documents
		where personnel_id = $1 and institution_id = $2 and file_reference = $3 and document_category = 'evaluare'
	`, evaluation.PersonnelID, institutionID, evaluation.EvaluationCode).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		documentCode := fmt.Sprintf("PFD-%d-%05d", time.Now().UTC().Year(), time.Now().UTC().UnixNano()%100000)
		if _, err := s.pool.Exec(ctx, `
			insert into education_personnel_file_documents (
				personnel_id, document_code, document_category, document_title, file_scope, confidentiality_level,
				issued_on, expires_on, file_reference, sensitive_data, included_in_portfolio, institution_id, notes
			) values ($1::uuid,$2,'evaluare',$3,$4,'confidential',$5,null,$6,false,false,$7,$8)
		`, evaluation.PersonnelID, documentCode, title, fileScope, evaluation.FinalizedOn, evaluation.EvaluationCode, institutionID, evaluation.Summary); err != nil {
			return fmt.Errorf("insert mirrored evaluation document: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("find mirrored evaluation document: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
		update education_personnel_file_documents
		set
			document_title = $1,
			file_scope = $2,
			confidentiality_level = 'confidential',
			issued_on = $3,
			file_reference = $4,
			sensitive_data = false,
			included_in_portfolio = false,
			notes = $5,
			updated_at = now()
		where id = $6 and personnel_id = $7 and institution_id = $8
	`, title, fileScope, evaluation.FinalizedOn, evaluation.EvaluationCode, evaluation.Summary, existingID, evaluation.PersonnelID, institutionID); err != nil {
		return fmt.Errorf("update mirrored evaluation document: %w", err)
	}

	return nil
}

type evaluationFileContext struct {
	ID                 string
	EvaluationCode     string
	EmployeeCode       string
	FullName           string
	RoleTitle          string
	SchoolYear         string
	Status             string
	Score              float64
	Qualification      string
	EvaluatorName      string
	FinalizedOn        *time.Time
	Summary            string
	PersonnelID        string
	PersonnelRoleTitle string
}

func (s *Service) loadEvaluationFileContext(ctx context.Context, evaluationID string, institutionID string) (*evaluationFileContext, error) {
	var evaluation evaluationFileContext
	err := s.pool.QueryRow(ctx, `
		select
			id::text,
			evaluation_code,
			employee_code,
			full_name,
			role_title,
			school_year,
			status,
			score,
			qualification,
			evaluator_name,
			finalized_on,
			summary
		from education_evaluations
		where id = $1 and institution_id = $2
	`, evaluationID, institutionID).Scan(
		&evaluation.ID,
		&evaluation.EvaluationCode,
		&evaluation.EmployeeCode,
		&evaluation.FullName,
		&evaluation.RoleTitle,
		&evaluation.SchoolYear,
		&evaluation.Status,
		&evaluation.Score,
		&evaluation.Qualification,
		&evaluation.EvaluatorName,
		&evaluation.FinalizedOn,
		&evaluation.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load evaluation for personnel file sync: %w", err)
	}

	err = s.pool.QueryRow(ctx, `
		select id::text, role_title
		from education_personnel
		where institution_id = $1 and (employee_code = $2 or full_name = $3)
		order by case when employee_code = $2 then 0 else 1 end, id
		limit 1
	`, institutionID, evaluation.EmployeeCode, evaluation.FullName).Scan(&evaluation.PersonnelID, &evaluation.PersonnelRoleTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		return &evaluation, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load personnel for evaluation file sync: %w", err)
	}

	return &evaluation, nil
}

func personnelFileScope(personnelRoleTitle string, fallbackRoleTitle string) string {
	roleTitle := strings.ToLower(strings.TrimSpace(personnelRoleTitle))
	if roleTitle == "" {
		roleTitle = strings.ToLower(strings.TrimSpace(fallbackRoleTitle))
	}

	switch {
	case strings.Contains(roleTitle, "director adjunct"):
		return "dosar_director_adjunct"
	case strings.Contains(roleTitle, "director"):
		return "dosar_director"
	default:
		return "dosar_personal"
	}
}

func (s *Service) syncEvaluationAppealEffects(ctx context.Context, evaluationID string, institutionID string) error {
	evaluation, err := s.loadEvaluationFileContext(ctx, evaluationID, institutionID)
	if err != nil {
		return err
	}
	if evaluation == nil {
		return nil
	}

	if err := s.syncEvaluationStatusFromAppeals(ctx, evaluation, institutionID); err != nil {
		return err
	}
	if err := s.syncEvaluationAppealDocumentsToPersonnelFile(ctx, evaluation, institutionID); err != nil {
		return err
	}
	if err := s.syncEvaluationPersonnelStatus(ctx, evaluation, institutionID); err != nil {
		return err
	}

	return nil
}

func (s *Service) syncEvaluationStatusFromAppeals(ctx context.Context, evaluation *evaluationFileContext, institutionID string) error {
	var activeAppeals int
	if err := s.pool.QueryRow(ctx, `
		select count(*)
		from education_evaluation_appeals
		where evaluation_id = $1 and institution_id = $2 and status in ('submitted', 'review')
	`, evaluation.ID, institutionID).Scan(&activeAppeals); err != nil {
		return fmt.Errorf("count active evaluation appeals: %w", err)
	}

	nextStatus := evaluation.Status
	if activeAppeals > 0 {
		nextStatus = "contested"
	} else if evaluation.Status == "contested" {
		nextStatus = deriveEvaluationResolvedStatus(evaluation)
	}

	if nextStatus == "" || nextStatus == evaluation.Status {
		return nil
	}

	if _, err := s.pool.Exec(ctx, `
		update education_evaluations
		set status = $1, updated_at = now()
		where id = $2::uuid and institution_id = $3
	`, nextStatus, evaluation.ID, institutionID); err != nil {
		return fmt.Errorf("update evaluation status from appeals: %w", err)
	}
	evaluation.Status = nextStatus

	return nil
}

func deriveEvaluationResolvedStatus(evaluation *evaluationFileContext) string {
	if evaluation.FinalizedOn != nil {
		return "approved"
	}
	if strings.TrimSpace(evaluation.EvaluatorName) != "" || evaluation.Score > 0 {
		return "reviewed"
	}
	return "submitted"
}

func (s *Service) syncEvaluationAppealDocumentsToPersonnelFile(ctx context.Context, evaluation *evaluationFileContext, institutionID string) error {
	if evaluation.PersonnelID == "" {
		return nil
	}

	rows, err := s.pool.Query(ctx, `
		select appeal_code, submitted_on, status, grounds, resolved_on, decision_summary, committee_note, attached_to_personnel_file
		from education_evaluation_appeals
		where evaluation_id = $1 and institution_id = $2
		order by submitted_on, appeal_code
	`, evaluation.ID, institutionID)
	if err != nil {
		return fmt.Errorf("list evaluation appeals for personnel file sync: %w", err)
	}
	defer rows.Close()

	activeFileReferences := make([]string, 0)
	for rows.Next() {
		var appeal evaluationAppealMirrorRecord
		if err := rows.Scan(
			&appeal.AppealCode,
			&appeal.SubmittedOn,
			&appeal.Status,
			&appeal.Grounds,
			&appeal.ResolvedOn,
			&appeal.DecisionSummary,
			&appeal.CommitteeNote,
			&appeal.AttachedToPersonnelFile,
		); err != nil {
			return fmt.Errorf("scan evaluation appeal for personnel file sync: %w", err)
		}
		if !appeal.AttachedToPersonnelFile {
			continue
		}

		fileReference := fmt.Sprintf("%s/%s", evaluation.EvaluationCode, appeal.AppealCode)
		activeFileReferences = append(activeFileReferences, fileReference)
		if err := s.upsertEvaluationAppealDocument(ctx, evaluation, appeal, fileReference, institutionID); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate evaluation appeals for personnel file sync: %w", err)
	}

	deleteQuery := `
		delete from education_personnel_file_documents
		where personnel_id = $1 and institution_id = $2 and document_category = 'evaluare' and file_reference like $3
	`
	deleteArgs := []any{evaluation.PersonnelID, institutionID, evaluation.EvaluationCode + "/APEL-%"}
	if len(activeFileReferences) > 0 {
		deleteQuery += ` and not (file_reference = any($4::text[]))`
		deleteArgs = append(deleteArgs, activeFileReferences)
	}

	if _, err := s.pool.Exec(ctx, deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("delete stale mirrored evaluation appeal documents: %w", err)
	}

	return nil
}

type evaluationAppealMirrorRecord struct {
	AppealCode              string
	SubmittedOn             time.Time
	Status                  string
	Grounds                 string
	ResolvedOn              *time.Time
	DecisionSummary         string
	CommitteeNote           string
	AttachedToPersonnelFile bool
}

func (s *Service) upsertEvaluationAppealDocument(ctx context.Context, evaluation *evaluationFileContext, appeal evaluationAppealMirrorRecord, fileReference string, institutionID string) error {
	title := fmt.Sprintf("Contestatie evaluare %s - %s", evaluation.SchoolYear, appeal.AppealCode)
	notesParts := []string{
		fmt.Sprintf("Evaluare: %s", evaluation.EvaluationCode),
		fmt.Sprintf("Status contestatie: %s", appeal.Status),
		fmt.Sprintf("Motive: %s", appeal.Grounds),
	}
	if appeal.ResolvedOn != nil {
		notesParts = append(notesParts, "Solutionata la: "+appeal.ResolvedOn.Format("2006-01-02"))
	}
	if strings.TrimSpace(appeal.DecisionSummary) != "" {
		notesParts = append(notesParts, "Decizie: "+appeal.DecisionSummary)
	}
	if strings.TrimSpace(appeal.CommitteeNote) != "" {
		notesParts = append(notesParts, "Nota comisie: "+appeal.CommitteeNote)
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
			) values ($1::uuid, $2, 'evaluare', $3, $4, 'strict_confidential', $5, null, $6, true, false, $7, $8)
		`, evaluation.PersonnelID, documentCode, title, fileScope, appeal.SubmittedOn, fileReference, institutionID, strings.Join(notesParts, " | ")); err != nil {
			return fmt.Errorf("insert mirrored evaluation appeal document: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("find mirrored evaluation appeal document: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
		update education_personnel_file_documents
		set
			document_title = $1,
			file_scope = $2,
			confidentiality_level = 'strict_confidential',
			issued_on = $3,
			file_reference = $4,
			sensitive_data = true,
			included_in_portfolio = false,
			notes = $5,
			updated_at = now()
		where id = $6 and personnel_id = $7 and institution_id = $8
	`, title, fileScope, appeal.SubmittedOn, fileReference, strings.Join(notesParts, " | "), existingID, evaluation.PersonnelID, institutionID); err != nil {
		return fmt.Errorf("update mirrored evaluation appeal document: %w", err)
	}

	return nil
}

func normalizeEvaluationSelfReviewRequest(req *CreatePersonnelEvaluationSelfReviewRequest) {
	req.SectionTitle = strings.TrimSpace(req.SectionTitle)
	req.NarrativeType = strings.TrimSpace(req.NarrativeType)
	req.Status = strings.TrimSpace(req.Status)
	req.CompletedOn = strings.TrimSpace(req.CompletedOn)
	req.EvidenceSummary = strings.TrimSpace(req.EvidenceSummary)
	req.Strengths = strings.TrimSpace(req.Strengths)
	req.ImprovementNeeds = strings.TrimSpace(req.ImprovementNeeds)
	req.Notes = strings.TrimSpace(req.Notes)
}

func normalizeEvaluationCriterionRequest(req *CreatePersonnelEvaluationCriterionRequest) {
	req.CriterionCategory = strings.TrimSpace(req.CriterionCategory)
	req.CriterionLabel = strings.TrimSpace(req.CriterionLabel)
	req.Status = strings.TrimSpace(req.Status)
	req.EvidenceSummary = strings.TrimSpace(req.EvidenceSummary)
	req.Notes = strings.TrimSpace(req.Notes)
}

func validEvaluationCriterionScores(maxScore float64, selfScore float64, reviewerScore float64, finalScore float64) bool {
	if maxScore < 0 || maxScore > 100 {
		return false
	}
	if selfScore < 0 || reviewerScore < 0 || finalScore < 0 {
		return false
	}
	if selfScore > maxScore || reviewerScore > maxScore || finalScore > maxScore {
		return false
	}
	return true
}

func buildEvaluationSelfReviewFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"eesr.evaluation_id = $1", "eesr.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["review_code"]; value != "" {
		addContains("eesr.review_code", value)
	}
	if value := filters["section_title"]; value != "" {
		addContains("eesr.section_title", value)
	}
	if value := filters["narrative_type"]; value != "" {
		addContains("eesr.narrative_type", value)
	}
	if value := filters["status"]; value != "" {
		addContains("eesr.status", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func buildEvaluationCriteriaFilters(filters map[string]string, recordID string, institutionID string) (string, []any) {
	where := []string{"eec.evaluation_id = $1", "eec.institution_id = $2"}
	args := []any{recordID, institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(value))+"%")
		where = append(where, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	if value := filters["criterion_code"]; value != "" {
		addContains("eec.criterion_code", value)
	}
	if value := filters["criterion_category"]; value != "" {
		addContains("eec.criterion_category", value)
	}
	if value := filters["criterion_label"]; value != "" {
		addContains("eec.criterion_label", value)
	}
	if value := filters["status"]; value != "" {
		addContains("eec.status", value)
	}
	return "where " + strings.Join(where, " and "), args
}

func evaluationSelfReviewSortColumn(value string) string {
	switch value {
	case "review_code":
		return "eesr.review_code"
	case "section_title":
		return "eesr.section_title"
	case "narrative_type":
		return "eesr.narrative_type"
	case "status":
		return "eesr.status"
	case "assumed_score":
		return "eesr.assumed_score"
	default:
		return "eesr.completed_on"
	}
}

func evaluationCriteriaSortColumn(value string) string {
	switch value {
	case "criterion_category":
		return "eec.criterion_category"
	case "criterion_label":
		return "eec.criterion_label"
	case "status":
		return "eec.status"
	case "max_score":
		return "eec.max_score"
	case "final_score":
		return "eec.final_score"
	default:
		return "eec.criterion_code"
	}
}
