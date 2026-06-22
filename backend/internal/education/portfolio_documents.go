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

func (s *Service) PortfolioRecordPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type portfolioDocument struct {
		PortfolioCode        string
		OwnerName            string
		OwnerRole            string
		SchoolYear           string
		Status               string
		SectionCount         int
		LastUpdatedOn        string
		RetentionUntil       string
		TransferStatus       string
		AuthenticityDeclared bool
		ConsentCaptured      bool
		Custodian            string
		Notes                string
	}

	var item portfolioDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			portfolio_code,
			owner_name,
			owner_role,
			school_year,
			status,
			section_count,
			to_char(last_updated_on, 'YYYY-MM-DD'),
			to_char(retention_until, 'YYYY-MM-DD'),
			transfer_status,
			authenticity_declared,
			consent_captured,
			custodian,
			notes
		from education_portfolios
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&item.PortfolioCode,
		&item.OwnerName,
		&item.OwnerRole,
		&item.SchoolYear,
		&item.Status,
		&item.SectionCount,
		&item.LastUpdatedOn,
		&item.RetentionUntil,
		&item.TransferStatus,
		&item.AuthenticityDeclared,
		&item.ConsentCaptured,
		&item.Custodian,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Portofoliu profesional CD",
		"",
		"Date generale",
		fmt.Sprintf("Cod portofoliu: %s", item.PortfolioCode),
		fmt.Sprintf("Titular: %s", item.OwnerName),
		fmt.Sprintf("Functie: %s", item.OwnerRole),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Numar sectiuni: %d", item.SectionCount),
		fmt.Sprintf("Ultima actualizare: %s", item.LastUpdatedOn),
		fmt.Sprintf("Retentie pana la: %s", item.RetentionUntil),
		fmt.Sprintf("Status transfer: %s", item.TransferStatus),
		fmt.Sprintf("Declaratie autenticitate: %t", item.AuthenticityDeclared),
		fmt.Sprintf("Consimtamant capturat: %t", item.ConsentCaptured),
		fmt.Sprintf("Custode: %s", valueOrDash(item.Custodian)),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	if summaryLines, err := s.portfolioDocumentSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Documente")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.portfolioChecklistSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Checklist conformitate")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.portfolioOpisSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Opis si index")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.portfolioCustodySummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Custodie si acces")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.portfolioTransferSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Predare si transfer")
		lines = append(lines, summaryLines...)
	}
	if summaryLines, err := s.portfolioReviewSummaryPDFLines(r, recordID, s.institutionID(r)); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_pdf_failed"})
		return
	} else if len(summaryLines) > 0 {
		lines = append(lines, "", "Verificari si validari")
		lines = append(lines, summaryLines...)
	}

	writeEducationPDFDownload(w, "Portofoliu profesional CD", fmt.Sprintf("portofoliu-cd-%s", item.PortfolioCode), lines)
}

func (s *Service) portfolioDocumentSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select section_code, component_code, document_title, source_scope, to_char(issued_on, 'YYYY-MM-DD')
		from education_portfolio_documents
		where portfolio_id = $1 and institution_id = $2
		order by chronological_index, issued_on, document_title
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var sectionCode string
		var componentCode string
		var documentTitle string
		var sourceScope string
		var issuedOn string
		if err := rows.Scan(&sectionCode, &componentCode, &documentTitle, &sourceScope, &issuedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s / %s - %s (%s, emis %s)", sectionCode, componentCode, documentTitle, sourceScope, issuedOn))
	}

	return lines, rows.Err()
}

func (s *Service) portfolioChecklistSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select requirement_code, requirement_label, status, document_count, to_char(last_checked_on, 'YYYY-MM-DD')
		from education_portfolio_checklist
		where portfolio_id = $1 and institution_id = $2
		order by section_code, requirement_code
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var requirementCode string
		var requirementLabel string
		var status string
		var documentCount int
		var lastCheckedOn string
		if err := rows.Scan(&requirementCode, &requirementLabel, &status, &documentCount, &lastCheckedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / documente %d / verificat %s", requirementCode, requirementLabel, status, documentCount, lastCheckedOn))
	}

	return lines, rows.Err()
}

func (s *Service) portfolioOpisSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select chronological_index, entry_title, document_reference, included_in_transfer
		from education_portfolio_opis
		where portfolio_id = $1 and institution_id = $2
		order by chronological_index, entry_title
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var chronologicalIndex int
		var entryTitle string
		var documentReference string
		var includedInTransfer bool
		if err := rows.Scan(&chronologicalIndex, &entryTitle, &documentReference, &includedInTransfer); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%d - %s / ref. %s / transfer %t", chronologicalIndex, entryTitle, documentReference, includedInTransfer))
	}

	return lines, rows.Err()
}

func (s *Service) portfolioCustodySummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select event_type, holder_name, holder_role, to_char(started_on, 'YYYY-MM-DD'), coalesce(to_char(ended_on, 'YYYY-MM-DD'), ''), access_mode
		from education_portfolio_custody
		where portfolio_id = $1 and institution_id = $2
		order by started_on desc, holder_name
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var eventType string
		var holderName string
		var holderRole string
		var startedOn string
		var endedOn string
		var accessMode string
		if err := rows.Scan(&eventType, &holderName, &holderRole, &startedOn, &endedOn, &accessMode); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / %s -> %s / acces %s", eventType, holderName, valueOrDash(holderRole), startedOn, valueOrDash(endedOn), accessMode))
	}

	return lines, rows.Err()
}

func (s *Service) portfolioTransferSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select transfer_code, transfer_type, source_institution, destination_institution, status, to_char(handover_on, 'YYYY-MM-DD'), coalesce(to_char(received_on, 'YYYY-MM-DD'), '')
		from education_portfolio_transfers
		where portfolio_id = $1 and institution_id = $2
		order by handover_on desc, transfer_code desc
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var transferCode string
		var transferType string
		var sourceInstitution string
		var destinationInstitution string
		var status string
		var handoverOn string
		var receivedOn string
		if err := rows.Scan(&transferCode, &transferType, &sourceInstitution, &destinationInstitution, &status, &handoverOn, &receivedOn); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s -> %s / %s / predat %s / receptionat %s", transferCode, transferType, sourceInstitution, destinationInstitution, status, handoverOn, valueOrDash(receivedOn)))
	}

	return lines, rows.Err()
}

func (s *Service) portfolioReviewSummaryPDFLines(r *http.Request, recordID string, institutionID string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select review_code, review_stage, outcome, reviewer_name, to_char(reviewed_on, 'YYYY-MM-DD'), missing_documents, compliance_score
		from education_portfolio_reviews
		where portfolio_id = $1 and institution_id = $2
		order by reviewed_on desc, review_code desc
	`, recordID, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]string, 0)
	for rows.Next() {
		var reviewCode string
		var reviewStage string
		var outcome string
		var reviewerName string
		var reviewedOn string
		var missingDocuments int
		var complianceScore int
		if err := rows.Scan(&reviewCode, &reviewStage, &outcome, &reviewerName, &reviewedOn, &missingDocuments, &complianceScore); err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s - %s / %s / %s / lipsuri %d / scor %d / %s", reviewCode, reviewStage, outcome, reviewerName, missingDocuments, complianceScore, reviewedOn))
	}

	return lines, rows.Err()
}
