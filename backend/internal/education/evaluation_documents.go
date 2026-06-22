package education

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) EvaluationPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	institutionID := s.institutionID(r)

	evaluation, err := s.loadEvaluationFileContext(r.Context(), recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_pdf_failed"})
		return
	}
	if evaluation == nil {
		writeEducationNotFound(w, "education_evaluation_not_found")
		return
	}

	lines := []string{
		"Document: Fisa evaluare anuala",
		"",
		"Date generale",
		fmt.Sprintf("Cod evaluare: %s", evaluation.EvaluationCode),
		fmt.Sprintf("Cadru didactic: %s", evaluation.FullName),
		fmt.Sprintf("Cod angajat: %s", evaluation.EmployeeCode),
		fmt.Sprintf("Functie: %s", evaluation.RoleTitle),
		fmt.Sprintf("An scolar: %s", evaluation.SchoolYear),
		fmt.Sprintf("Status: %s", evaluation.Status),
		fmt.Sprintf("Punctaj final: %.2f", evaluation.Score),
		fmt.Sprintf("Calificativ: %s", firstNonEmpty(evaluation.Qualification, evaluationQualification(evaluation.Score))),
		fmt.Sprintf("Evaluator: %s", valueOrDash(evaluation.EvaluatorName)),
		fmt.Sprintf("Data finalizarii: %s", formatOptionalTime(evaluation.FinalizedOn)),
	}
	if strings.TrimSpace(evaluation.Summary) != "" {
		lines = append(lines,
			"",
			"Rezumat evaluare",
			evaluation.Summary,
		)
	}

	selfReviewLines, err := s.evaluationSelfReviewPDFLines(r, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_pdf_failed"})
		return
	}
	if len(selfReviewLines) > 0 {
		lines = append(lines, "", "Autoevaluari")
		lines = append(lines, selfReviewLines...)
	}
	criterionLines, err := s.evaluationCriterionPDFLines(r, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_pdf_failed"})
		return
	}
	if len(criterionLines) > 0 {
		lines = append(lines, "", "Criterii si punctaje")
		lines = append(lines, criterionLines...)
	}
	appealLines, err := s.evaluationAppealSummaryPDFLines(r, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_pdf_failed"})
		return
	}
	if len(appealLines) > 0 {
		lines = append(lines, "", "Contestatii")
		lines = append(lines, appealLines...)
	}
	resultLines, err := s.evaluationResultIssueSummaryPDFLines(r, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_pdf_failed"})
		return
	}
	if len(resultLines) > 0 {
		lines = append(lines, "", "Comunicare rezultat")
		lines = append(lines, resultLines...)
	}

	writeEducationPDFDownload(w, "Fisa evaluare anuala", fmt.Sprintf("fisa-evaluare-%s", evaluation.EvaluationCode), lines)
}

func (s *Service) EvaluationAppealPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	appealID := strings.TrimSpace(chi.URLParam(r, "appealID"))

	type appealDocument struct {
		EvaluationCode          string
		FullName                string
		SchoolYear              string
		Qualification           string
		Score                   float64
		AppealCode              string
		SubmittedBy             string
		SubmittedOn             string
		Status                  string
		Grounds                 string
		HearingOn               string
		ResolvedOn              string
		DecisionSummary         string
		CommitteeNote           string
		AttachedToPersonnelFile bool
	}

	var item appealDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			ee.evaluation_code,
			ee.full_name,
			ee.school_year,
			ee.qualification,
			ee.score,
			eea.appeal_code,
			eea.submitted_by,
			to_char(eea.submitted_on, 'YYYY-MM-DD'),
			eea.status,
			eea.grounds,
			coalesce(to_char(eea.hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(eea.resolved_on, 'YYYY-MM-DD'), ''),
			eea.decision_summary,
			eea.committee_note,
			eea.attached_to_personnel_file
		from education_evaluation_appeals eea
		join education_evaluations ee on ee.id = eea.evaluation_id
		where eea.id = $1 and eea.evaluation_id = $2 and eea.institution_id = $3
	`, appealID, recordID, s.institutionID(r)).Scan(
		&item.EvaluationCode,
		&item.FullName,
		&item.SchoolYear,
		&item.Qualification,
		&item.Score,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.HearingOn,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.CommitteeNote,
		&item.AttachedToPersonnelFile,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_appeal_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Contestatie evaluare anuala",
		"",
		"Referinta evaluare",
		fmt.Sprintf("Cod evaluare: %s", item.EvaluationCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Punctaj: %.2f", item.Score),
		fmt.Sprintf("Calificativ: %s", firstNonEmpty(item.Qualification, evaluationQualification(item.Score))),
		"",
		"Date contestatie",
		fmt.Sprintf("Cod contestatie: %s", item.AppealCode),
		fmt.Sprintf("Depusa de: %s", item.SubmittedBy),
		fmt.Sprintf("Data depunerii: %s", item.SubmittedOn),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Sedinta: %s", valueOrDash(item.HearingOn)),
		fmt.Sprintf("Solutionata la: %s", valueOrDash(item.ResolvedOn)),
		fmt.Sprintf("In dosarul personal: %t", item.AttachedToPersonnelFile),
		"",
		"Motive",
		item.Grounds,
	}
	if strings.TrimSpace(item.DecisionSummary) != "" {
		lines = append(lines, "", "Decizie comisie", item.DecisionSummary)
	}
	if strings.TrimSpace(item.CommitteeNote) != "" {
		lines = append(lines, "", "Nota comisiei", item.CommitteeNote)
	}

	writeEducationPDFDownload(w, "Contestatie evaluare", fmt.Sprintf("contestatie-%s", item.AppealCode), lines)
}

func (s *Service) EvaluationResultIssuePDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type resultIssueDocument struct {
		EvaluationCode          string
		FullName                string
		SchoolYear              string
		Qualification           string
		Score                   float64
		IssueCode               string
		DocumentType            string
		RecipientName           string
		RecipientRole           string
		DeliveryChannel         string
		DeliveryStatus          string
		IssuedOn                string
		DeliveredOn             string
		AcknowledgedOn          string
		RegistryReference       string
		AttachedToPersonnelFile bool
		Notes                   string
	}

	var item resultIssueDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			ee.evaluation_code,
			ee.full_name,
			ee.school_year,
			ee.qualification,
			ee.score,
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
			eeri.notes
		from education_evaluation_result_issues eeri
		join education_evaluations ee on ee.id = eeri.evaluation_id
		where eeri.id = $1 and eeri.evaluation_id = $2 and eeri.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.EvaluationCode,
		&item.FullName,
		&item.SchoolYear,
		&item.Qualification,
		&item.Score,
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
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_result_issue_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Comunicare rezultat evaluare",
		"",
		"Referinta evaluare",
		fmt.Sprintf("Cod evaluare: %s", item.EvaluationCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Punctaj: %.2f", item.Score),
		fmt.Sprintf("Calificativ: %s", firstNonEmpty(item.Qualification, evaluationQualification(item.Score))),
		"",
		"Date comunicare",
		fmt.Sprintf("Cod document: %s", item.IssueCode),
		fmt.Sprintf("Tip document: %s", item.DocumentType),
		fmt.Sprintf("Destinatar: %s", item.RecipientName),
		fmt.Sprintf("Rol destinatar: %s", valueOrDash(item.RecipientRole)),
		fmt.Sprintf("Canal livrare: %s", item.DeliveryChannel),
		fmt.Sprintf("Status livrare: %s", item.DeliveryStatus),
		fmt.Sprintf("Emis la: %s", item.IssuedOn),
		fmt.Sprintf("Predat la: %s", valueOrDash(item.DeliveredOn)),
		fmt.Sprintf("Confirmat la: %s", valueOrDash(item.AcknowledgedOn)),
		fmt.Sprintf("Referinta registratura: %s", valueOrDash(item.RegistryReference)),
		fmt.Sprintf("In dosarul personal: %t", item.AttachedToPersonnelFile),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Comunicare rezultat evaluare", fmt.Sprintf("comunicare-rezultat-%s", item.IssueCode), lines)
}

func (s *Service) evaluationSelfReviewPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select review_code, section_title, narrative_type, status, to_char(completed_on, 'YYYY-MM-DD'), assumed_score,
			evidence_summary, strengths, improvement_needs
		from education_evaluation_self_reviews
		where evaluation_id = $1 and institution_id = $2
		order by completed_on, review_code
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var reviewCode string
		var sectionTitle string
		var narrativeType string
		var status string
		var completedOn string
		var assumedScore float64
		var evidenceSummary string
		var strengths string
		var improvementNeeds string
		if err := rows.Scan(&reviewCode, &sectionTitle, &narrativeType, &status, &completedOn, &assumedScore, &evidenceSummary, &strengths, &improvementNeeds); err != nil {
			return nil, err
		}
		lines = append(lines,
			fmt.Sprintf("%s | %s | %s | %s | %.2f", reviewCode, sectionTitle, narrativeType, status, assumedScore),
		)
		if strings.TrimSpace(evidenceSummary) != "" {
			lines = append(lines, "Dovezi: "+evidenceSummary)
		}
		if strings.TrimSpace(strengths) != "" {
			lines = append(lines, "Puncte forte: "+strengths)
		}
		if strings.TrimSpace(improvementNeeds) != "" {
			lines = append(lines, "Nevoi de imbunatatire: "+improvementNeeds)
		}
		lines = append(lines, "Data completarii: "+completedOn, "")
	}
	return trimTrailingBlankLines(lines), rows.Err()
}

func (s *Service) evaluationCriterionPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select criterion_code, criterion_category, criterion_label, max_score, self_score, reviewer_score, final_score, status, evidence_summary
		from education_evaluation_criteria
		where evaluation_id = $1 and institution_id = $2
		order by criterion_code
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var criterionCode string
		var category string
		var label string
		var maxScore float64
		var selfScore float64
		var reviewerScore float64
		var finalScore float64
		var status string
		var evidenceSummary string
		if err := rows.Scan(&criterionCode, &category, &label, &maxScore, &selfScore, &reviewerScore, &finalScore, &status, &evidenceSummary); err != nil {
			return nil, err
		}
		lines = append(lines,
			fmt.Sprintf("%s | %s | %s", criterionCode, category, label),
			fmt.Sprintf("Scoruri: maxim %.2f | auto %.2f | evaluator %.2f | final %.2f | status %s", maxScore, selfScore, reviewerScore, finalScore, status),
		)
		if strings.TrimSpace(evidenceSummary) != "" {
			lines = append(lines, "Dovezi: "+evidenceSummary)
		}
		lines = append(lines, "")
	}
	return trimTrailingBlankLines(lines), rows.Err()
}

func (s *Service) evaluationAppealSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select appeal_code, submitted_by, status, to_char(submitted_on, 'YYYY-MM-DD'),
			coalesce(to_char(resolved_on, 'YYYY-MM-DD'), ''), attached_to_personnel_file
		from education_evaluation_appeals
		where evaluation_id = $1 and institution_id = $2
		order by submitted_on, appeal_code
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var appealCode string
		var submittedBy string
		var status string
		var submittedOn string
		var resolvedOn string
		var attached bool
		if err := rows.Scan(&appealCode, &submittedBy, &status, &submittedOn, &resolvedOn, &attached); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s | %s | %s | depusa %s | solutionata %s | in dosar %t", appealCode, submittedBy, status, submittedOn, valueOrDash(resolvedOn), attached))
	}
	return lines, rows.Err()
}

func (s *Service) evaluationResultIssueSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select issue_code, document_type, recipient_name, delivery_status, to_char(issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(acknowledged_on, 'YYYY-MM-DD'), ''), attached_to_personnel_file
		from education_evaluation_result_issues
		where evaluation_id = $1 and institution_id = $2
		order by issued_on, issue_code
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var issueCode string
		var documentType string
		var recipientName string
		var deliveryStatus string
		var issuedOn string
		var acknowledgedOn string
		var attached bool
		if err := rows.Scan(&issueCode, &documentType, &recipientName, &deliveryStatus, &issuedOn, &acknowledgedOn, &attached); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s | emis %s | confirmat %s | in dosar %t", issueCode, documentType, recipientName, deliveryStatus, issuedOn, valueOrDash(acknowledgedOn), attached))
	}
	return lines, rows.Err()
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "-"
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.Format("2006-01-02")
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
