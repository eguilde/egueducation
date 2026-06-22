package education

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) MobilityCasePDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type mobilityCaseDocument struct {
		CaseCode          string
		EmployeeCode      string
		FullName          string
		SchoolYear        string
		RequestType       string
		Stage             string
		Status            string
		SourceSchool      string
		DestinationSchool string
		SubmittedOn       string
		ReviewedBy        string
		Notes             string
	}

	var item mobilityCaseDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			case_code,
			employee_code,
			full_name,
			school_year,
			request_type,
			stage,
			status,
			source_school,
			destination_school,
			to_char(submitted_on, 'YYYY-MM-DD'),
			reviewed_by,
			notes
		from education_mobility_cases
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&item.CaseCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.SchoolYear,
		&item.RequestType,
		&item.Stage,
		&item.Status,
		&item.SourceSchool,
		&item.DestinationSchool,
		&item.SubmittedOn,
		&item.ReviewedBy,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_case_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Dosar mobilitate",
		"",
		"Date generale",
		fmt.Sprintf("Cod caz: %s", item.CaseCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("Cod angajat: %s", item.EmployeeCode),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Tip solicitare: %s", item.RequestType),
		fmt.Sprintf("Etapa: %s", item.Stage),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Unitate sursa: %s", valueOrDash(item.SourceSchool)),
		fmt.Sprintf("Unitate destinatie: %s", valueOrDash(item.DestinationSchool)),
		fmt.Sprintf("Data depunerii: %s", item.SubmittedOn),
		fmt.Sprintf("Analizat de: %s", valueOrDash(item.ReviewedBy)),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	if summaryLines, err := s.mobilityAppealSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Contestatii")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.mobilityFinalDecisionSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Decizii finale")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.mobilityResultIssueSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Comunicare rezultat")
		lines = append(lines, summaryLines...)
	}

	writeEducationPDFDownload(w, "Dosar mobilitate", fmt.Sprintf("mobilitate-%s", item.CaseCode), lines)
}

func (s *Service) MobilityAppealPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type mobilityAppealDocument struct {
		CaseCode          string
		FullName          string
		SchoolYear        string
		RequestType       string
		Stage             string
		DestinationSchool string
		AppealCode        string
		SubmittedBy       string
		SubmittedOn       string
		Status            string
		Grounds           string
		HearingOn         string
		ResolvedOn        string
		DecisionSummary   string
		Notes             string
	}

	var item mobilityAppealDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emc.case_code,
			emc.full_name,
			emc.school_year,
			emc.request_type,
			emc.stage,
			emc.destination_school,
			ema.appeal_code,
			ema.submitted_by,
			to_char(ema.submitted_on, 'YYYY-MM-DD'),
			ema.status,
			ema.grounds,
			coalesce(to_char(ema.hearing_on, 'YYYY-MM-DD'), ''),
			coalesce(to_char(ema.resolved_on, 'YYYY-MM-DD'), ''),
			ema.decision_summary,
			ema.notes
		from education_mobility_appeals ema
		join education_mobility_cases emc on emc.id = ema.mobility_case_id
		where ema.id = $1 and ema.mobility_case_id = $2 and ema.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.CaseCode,
		&item.FullName,
		&item.SchoolYear,
		&item.RequestType,
		&item.Stage,
		&item.DestinationSchool,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.HearingOn,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_appeal_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Contestatie mobilitate",
		"",
		"Referinta caz",
		fmt.Sprintf("Cod caz: %s", item.CaseCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Tip solicitare: %s", item.RequestType),
		fmt.Sprintf("Etapa: %s", item.Stage),
		fmt.Sprintf("Unitate destinatie: %s", valueOrDash(item.DestinationSchool)),
		"",
		"Date contestatie",
		fmt.Sprintf("Cod contestatie: %s", item.AppealCode),
		fmt.Sprintf("Depusa de: %s", item.SubmittedBy),
		fmt.Sprintf("Data depunerii: %s", item.SubmittedOn),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Sedinta: %s", valueOrDash(item.HearingOn)),
		fmt.Sprintf("Solutionata la: %s", valueOrDash(item.ResolvedOn)),
		"",
		"Motive",
		item.Grounds,
	}
	if strings.TrimSpace(item.DecisionSummary) != "" {
		lines = append(lines, "", "Decizie", item.DecisionSummary)
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Contestatie mobilitate", fmt.Sprintf("contestatie-mobilitate-%s", item.AppealCode), lines)
}

func (s *Service) MobilityFinalDecisionPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type mobilityFinalDecisionDocument struct {
		CaseCode        string
		FullName        string
		SchoolYear      string
		RequestType     string
		Stage           string
		DecisionCode    string
		DecisionType    string
		Outcome         string
		ApprovedOn      string
		EffectiveFrom   string
		PanelName       string
		LegalBasis      string
		DestinationUnit string
		Notes           string
	}

	var item mobilityFinalDecisionDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emc.case_code,
			emc.full_name,
			emc.school_year,
			emc.request_type,
			emc.stage,
			emfd.decision_code,
			emfd.decision_type,
			emfd.outcome,
			to_char(emfd.approved_on, 'YYYY-MM-DD'),
			to_char(emfd.effective_from, 'YYYY-MM-DD'),
			emfd.panel_name,
			emfd.legal_basis,
			emfd.destination_unit,
			emfd.notes
		from education_mobility_final_decisions emfd
		join education_mobility_cases emc on emc.id = emfd.mobility_case_id
		where emfd.id = $1 and emfd.mobility_case_id = $2 and emfd.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.CaseCode,
		&item.FullName,
		&item.SchoolYear,
		&item.RequestType,
		&item.Stage,
		&item.DecisionCode,
		&item.DecisionType,
		&item.Outcome,
		&item.ApprovedOn,
		&item.EffectiveFrom,
		&item.PanelName,
		&item.LegalBasis,
		&item.DestinationUnit,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_final_decision_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Decizie finala mobilitate",
		"",
		"Referinta caz",
		fmt.Sprintf("Cod caz: %s", item.CaseCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Tip solicitare: %s", item.RequestType),
		fmt.Sprintf("Etapa: %s", item.Stage),
		"",
		"Date decizie",
		fmt.Sprintf("Cod decizie: %s", item.DecisionCode),
		fmt.Sprintf("Tip decizie: %s", item.DecisionType),
		fmt.Sprintf("Rezultat: %s", item.Outcome),
		fmt.Sprintf("Aprobata la: %s", item.ApprovedOn),
		fmt.Sprintf("Aplicare de la: %s", item.EffectiveFrom),
		fmt.Sprintf("Comisie / organism: %s", item.PanelName),
		fmt.Sprintf("Unitate destinatie: %s", valueOrDash(item.DestinationUnit)),
	}
	if strings.TrimSpace(item.LegalBasis) != "" {
		lines = append(lines, "", "Baza legala", item.LegalBasis)
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Decizie finala mobilitate", fmt.Sprintf("decizie-mobilitate-%s", item.DecisionCode), lines)
}

func (s *Service) MobilityResultIssuePDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type mobilityResultIssueDocument struct {
		CaseCode          string
		FullName          string
		SchoolYear        string
		RequestType       string
		Stage             string
		IssueCode         string
		DocumentType      string
		RecipientName     string
		RecipientRole     string
		DeliveryChannel   string
		DeliveryStatus    string
		IssuedOn          string
		DeliveredOn       string
		RegistryReference string
		Notes             string
	}

	var item mobilityResultIssueDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emc.case_code,
			emc.full_name,
			emc.school_year,
			emc.request_type,
			emc.stage,
			emri.issue_code,
			emri.document_type,
			emri.recipient_name,
			emri.recipient_role,
			emri.delivery_channel,
			emri.delivery_status,
			to_char(emri.issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(emri.delivered_on, 'YYYY-MM-DD'), ''),
			emri.registry_reference,
			emri.notes
		from education_mobility_result_issues emri
		join education_mobility_cases emc on emc.id = emri.mobility_case_id
		where emri.id = $1 and emri.mobility_case_id = $2 and emri.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.CaseCode,
		&item.FullName,
		&item.SchoolYear,
		&item.RequestType,
		&item.Stage,
		&item.IssueCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.IssuedOn,
		&item.DeliveredOn,
		&item.RegistryReference,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_result_issue_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Comunicare rezultat mobilitate",
		"",
		"Referinta caz",
		fmt.Sprintf("Cod caz: %s", item.CaseCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Tip solicitare: %s", item.RequestType),
		fmt.Sprintf("Etapa: %s", item.Stage),
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
		fmt.Sprintf("Referinta registratura: %s", valueOrDash(item.RegistryReference)),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Comunicare rezultat mobilitate", fmt.Sprintf("comunicare-mobilitate-%s", item.IssueCode), lines)
}

func (s *Service) MeritGrantPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type meritGrantDocument struct {
		GrantCode     string
		FullName      string
		RoleTitle     string
		SchoolYear    string
		Category      string
		Status        string
		Score         float64
		CommitteeName string
		DecisionDate  string
		Funded        bool
		Notes         string
	}

	var item meritGrantDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			grant_code,
			full_name,
			role_title,
			school_year,
			category,
			status,
			score,
			committee_name,
			to_char(decision_date, 'YYYY-MM-DD'),
			funded,
			notes
		from education_merit_grants
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&item.GrantCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.Category,
		&item.Status,
		&item.Score,
		&item.CommitteeName,
		&item.DecisionDate,
		&item.Funded,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_grant_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Dosar gradatie de merit",
		"",
		"Date generale",
		fmt.Sprintf("Cod dosar: %s", item.GrantCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("Functie: %s", item.RoleTitle),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Categorie: %s", item.Category),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Punctaj: %.2f", item.Score),
		fmt.Sprintf("Comisie: %s", valueOrDash(item.CommitteeName)),
		fmt.Sprintf("Data deciziei: %s", item.DecisionDate),
		fmt.Sprintf("Finantat: %t", item.Funded),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	if summaryLines, err := s.meritAppealSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Contestatii")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.meritFinalDecisionSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Decizii finale")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.meritResultIssueSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Comunicare rezultat")
		lines = append(lines, summaryLines...)
	}

	writeEducationPDFDownload(w, "Dosar gradatie de merit", fmt.Sprintf("gradatie-merit-%s", item.GrantCode), lines)
}

func (s *Service) MeritAppealPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type meritAppealDocument struct {
		GrantCode       string
		FullName        string
		RoleTitle       string
		SchoolYear      string
		Score           float64
		AppealCode      string
		SubmittedBy     string
		SubmittedOn     string
		Status          string
		Grounds         string
		ResolvedOn      string
		DecisionSummary string
		Notes           string
	}

	var item meritAppealDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emg.grant_code,
			emg.full_name,
			emg.role_title,
			emg.school_year,
			emg.score,
			ema.appeal_code,
			ema.submitted_by,
			to_char(ema.submitted_on, 'YYYY-MM-DD'),
			ema.status,
			ema.grounds,
			coalesce(to_char(ema.resolved_on, 'YYYY-MM-DD'), ''),
			ema.decision_summary,
			ema.notes
		from education_merit_appeals ema
		join education_merit_grants emg on emg.id = ema.grant_id
		where ema.id = $1 and ema.grant_id = $2 and ema.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.GrantCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.Score,
		&item.AppealCode,
		&item.SubmittedBy,
		&item.SubmittedOn,
		&item.Status,
		&item.Grounds,
		&item.ResolvedOn,
		&item.DecisionSummary,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_appeal_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_appeal_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Contestatie gradatie de merit",
		"",
		"Referinta dosar",
		fmt.Sprintf("Cod dosar: %s", item.GrantCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("Functie: %s", item.RoleTitle),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Punctaj: %.2f", item.Score),
		"",
		"Date contestatie",
		fmt.Sprintf("Cod contestatie: %s", item.AppealCode),
		fmt.Sprintf("Depusa de: %s", item.SubmittedBy),
		fmt.Sprintf("Data depunerii: %s", item.SubmittedOn),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Solutionata la: %s", valueOrDash(item.ResolvedOn)),
		"",
		"Motive",
		item.Grounds,
	}
	if strings.TrimSpace(item.DecisionSummary) != "" {
		lines = append(lines, "", "Decizie", item.DecisionSummary)
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Contestatie gradatie de merit", fmt.Sprintf("contestatie-gradatie-%s", item.AppealCode), lines)
}

func (s *Service) MeritFinalDecisionPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type meritFinalDecisionDocument struct {
		GrantCode     string
		FullName      string
		RoleTitle     string
		SchoolYear    string
		DecisionCode  string
		DecisionStage string
		Outcome       string
		ApprovedOn    string
		EffectiveFrom string
		PanelName     string
		Funded        bool
		LegalBasis    string
		Notes         string
	}

	var item meritFinalDecisionDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emg.grant_code,
			emg.full_name,
			emg.role_title,
			emg.school_year,
			emfd.decision_code,
			emfd.decision_stage,
			emfd.outcome,
			to_char(emfd.approved_on, 'YYYY-MM-DD'),
			to_char(emfd.effective_from, 'YYYY-MM-DD'),
			emfd.panel_name,
			emfd.funded,
			emfd.legal_basis,
			emfd.notes
		from education_merit_final_decisions emfd
		join education_merit_grants emg on emg.id = emfd.grant_id
		where emfd.id = $1 and emfd.grant_id = $2 and emfd.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.GrantCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.DecisionCode,
		&item.DecisionStage,
		&item.Outcome,
		&item.ApprovedOn,
		&item.EffectiveFrom,
		&item.PanelName,
		&item.Funded,
		&item.LegalBasis,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_final_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_final_decision_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Decizie finala gradatie de merit",
		"",
		"Referinta dosar",
		fmt.Sprintf("Cod dosar: %s", item.GrantCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("Functie: %s", item.RoleTitle),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		"",
		"Date decizie",
		fmt.Sprintf("Cod decizie: %s", item.DecisionCode),
		fmt.Sprintf("Etapa: %s", item.DecisionStage),
		fmt.Sprintf("Rezultat: %s", item.Outcome),
		fmt.Sprintf("Aprobata la: %s", item.ApprovedOn),
		fmt.Sprintf("Aplicare de la: %s", item.EffectiveFrom),
		fmt.Sprintf("Comisie / organism: %s", item.PanelName),
		fmt.Sprintf("Include finantare: %t", item.Funded),
	}
	if strings.TrimSpace(item.LegalBasis) != "" {
		lines = append(lines, "", "Baza legala", item.LegalBasis)
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Decizie finala gradatie de merit", fmt.Sprintf("decizie-gradatie-%s", item.DecisionCode), lines)
}

func (s *Service) MeritResultIssuePDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))

	type meritResultIssueDocument struct {
		GrantCode         string
		FullName          string
		RoleTitle         string
		SchoolYear        string
		IssueCode         string
		DocumentType      string
		RecipientName     string
		RecipientRole     string
		DeliveryChannel   string
		DeliveryStatus    string
		IssuedOn          string
		DeliveredOn       string
		RegistryReference string
		Notes             string
	}

	var item meritResultIssueDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emg.grant_code,
			emg.full_name,
			emg.role_title,
			emg.school_year,
			emri.issue_code,
			emri.document_type,
			emri.recipient_name,
			emri.recipient_role,
			emri.delivery_channel,
			emri.delivery_status,
			to_char(emri.issued_on, 'YYYY-MM-DD'),
			coalesce(to_char(emri.delivered_on, 'YYYY-MM-DD'), ''),
			emri.registry_reference,
			emri.notes
		from education_merit_result_issues emri
		join education_merit_grants emg on emg.id = emri.grant_id
		where emri.id = $1 and emri.grant_id = $2 and emri.institution_id = $3
	`, itemID, recordID, s.institutionID(r)).Scan(
		&item.GrantCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.IssueCode,
		&item.DocumentType,
		&item.RecipientName,
		&item.RecipientRole,
		&item.DeliveryChannel,
		&item.DeliveryStatus,
		&item.IssuedOn,
		&item.DeliveredOn,
		&item.RegistryReference,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_result_issue_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_result_issue_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Comunicare rezultat gradatie de merit",
		"",
		"Referinta dosar",
		fmt.Sprintf("Cod dosar: %s", item.GrantCode),
		fmt.Sprintf("Cadru didactic: %s", item.FullName),
		fmt.Sprintf("Functie: %s", item.RoleTitle),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
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
		fmt.Sprintf("Referinta registratura: %s", valueOrDash(item.RegistryReference)),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Comunicare rezultat gradatie de merit", fmt.Sprintf("comunicare-gradatie-%s", item.IssueCode), lines)
}

func (s *Service) mobilityAppealSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select appeal_code, submitted_by, status, to_char(submitted_on, 'YYYY-MM-DD'), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), '')
		from education_mobility_appeals
		where mobility_case_id = $1 and institution_id = $2
		order by submitted_on desc, appeal_code desc
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
		if err := rows.Scan(&appealCode, &submittedBy, &status, &submittedOn, &resolvedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / depusa %s / solutionata %s", appealCode, submittedBy, status, submittedOn, valueOrDash(resolvedOn)))
	}

	return lines, rows.Err()
}

func (s *Service) mobilityFinalDecisionSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select decision_code, decision_type, outcome, to_char(approved_on, 'YYYY-MM-DD'), coalesce(destination_unit, '')
		from education_mobility_final_decisions
		where mobility_case_id = $1 and institution_id = $2
		order by approved_on desc, decision_code desc
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var decisionCode string
		var decisionType string
		var outcome string
		var approvedOn string
		var destinationUnit string
		if err := rows.Scan(&decisionCode, &decisionType, &outcome, &approvedOn, &destinationUnit); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / aprobata %s / destinatie %s", decisionCode, decisionType, outcome, approvedOn, valueOrDash(destinationUnit)))
	}

	return lines, rows.Err()
}

func (s *Service) mobilityResultIssueSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select issue_code, document_type, recipient_name, delivery_status, to_char(issued_on, 'YYYY-MM-DD')
		from education_mobility_result_issues
		where mobility_case_id = $1 and institution_id = $2
		order by issued_on desc, issue_code desc
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
		if err := rows.Scan(&issueCode, &documentType, &recipientName, &deliveryStatus, &issuedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / %s / emis %s", issueCode, documentType, recipientName, deliveryStatus, issuedOn))
	}

	return lines, rows.Err()
}

func (s *Service) meritAppealSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select appeal_code, submitted_by, status, to_char(submitted_on, 'YYYY-MM-DD'), coalesce(to_char(resolved_on, 'YYYY-MM-DD'), '')
		from education_merit_appeals
		where grant_id = $1 and institution_id = $2
		order by submitted_on desc, appeal_code desc
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
		if err := rows.Scan(&appealCode, &submittedBy, &status, &submittedOn, &resolvedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / depusa %s / solutionata %s", appealCode, submittedBy, status, submittedOn, valueOrDash(resolvedOn)))
	}

	return lines, rows.Err()
}

func (s *Service) meritFinalDecisionSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select decision_code, decision_stage, outcome, to_char(approved_on, 'YYYY-MM-DD'), funded
		from education_merit_final_decisions
		where grant_id = $1 and institution_id = $2
		order by approved_on desc, decision_code desc
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var decisionCode string
		var decisionStage string
		var outcome string
		var approvedOn string
		var funded bool
		if err := rows.Scan(&decisionCode, &decisionStage, &outcome, &approvedOn, &funded); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / aprobata %s / finantare %t", decisionCode, decisionStage, outcome, approvedOn, funded))
	}

	return lines, rows.Err()
}

func (s *Service) meritResultIssueSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select issue_code, document_type, recipient_name, delivery_status, to_char(issued_on, 'YYYY-MM-DD')
		from education_merit_result_issues
		where grant_id = $1 and institution_id = $2
		order by issued_on desc, issue_code desc
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
		if err := rows.Scan(&issueCode, &documentType, &recipientName, &deliveryStatus, &issuedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / %s / emis %s", issueCode, documentType, recipientName, deliveryStatus, issuedOn))
	}

	return lines, rows.Err()
}
