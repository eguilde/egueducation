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

func (s *Service) ManagerialDossierPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type dossierDocument struct {
		DossierCode         string
		SchoolYear          string
		DossierType         string
		Title               string
		Status              string
		OwnerName           string
		DueOn               string
		PublicationRequired bool
		Summary             string
	}

	var item dossierDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			dossier_code,
			school_year,
			dossier_type,
			title,
			status,
			owner_name,
			to_char(due_on, 'YYYY-MM-DD'),
			publication_required,
			summary
		from education_managerial_dossiers
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r)).Scan(
		&item.DossierCode,
		&item.SchoolYear,
		&item.DossierType,
		&item.Title,
		&item.Status,
		&item.OwnerName,
		&item.DueOn,
		&item.PublicationRequired,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Dosar managerial",
		"",
		"Date generale",
		fmt.Sprintf("Cod dosar: %s", item.DossierCode),
		fmt.Sprintf("An scolar: %s", item.SchoolYear),
		fmt.Sprintf("Tip dosar: %s", item.DossierType),
		fmt.Sprintf("Titlu: %s", item.Title),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Responsabil: %s", valueOrDash(item.OwnerName)),
		fmt.Sprintf("Termen: %s", valueOrDash(item.DueOn)),
		fmt.Sprintf("Publicare necesara: %t", item.PublicationRequired),
	}
	if strings.TrimSpace(item.Summary) != "" {
		lines = append(lines, "", "Rezumat", item.Summary)
	}

	writeEducationPDFDownload(w, "Dosar managerial", fmt.Sprintf("dosar-managerial-%s", item.DossierCode), lines)
}

func (s *Service) ManagerialDocumentPDF(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))

	type documentPDF struct {
		DossierCode         string
		DossierTitle        string
		DocumentCode        string
		DocumentCategory    string
		Title               string
		DocumentStatus      string
		VersionLabel        string
		Mandatory           bool
		PublicationRequired bool
		RegisteredOn        string
		ApprovedOn          string
		OwnerName           string
		FileReference       string
		Notes               string
	}

	var item documentPDF
	err := s.pool.QueryRow(r.Context(), `
		select
			emdos.dossier_code,
			emdos.title,
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
			emd.notes
		from education_managerial_documents emd
		join education_managerial_dossiers emdos on emdos.id = emd.dossier_id
		where emd.id = $1::uuid and emd.dossier_id = $2::uuid and emd.institution_id = $3
	`, documentID, recordID, s.institutionID(r)).Scan(
		&item.DossierCode,
		&item.DossierTitle,
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
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_document_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Act managerial",
		"",
		"Referinta dosar",
		fmt.Sprintf("Cod dosar: %s", item.DossierCode),
		fmt.Sprintf("Titlu dosar: %s", item.DossierTitle),
		"",
		"Date document",
		fmt.Sprintf("Cod document: %s", item.DocumentCode),
		fmt.Sprintf("Categorie: %s", item.DocumentCategory),
		fmt.Sprintf("Titlu: %s", item.Title),
		fmt.Sprintf("Status: %s", item.DocumentStatus),
		fmt.Sprintf("Versiune: %s", item.VersionLabel),
		fmt.Sprintf("Obligatoriu: %t", item.Mandatory),
		fmt.Sprintf("Publicare necesara: %t", item.PublicationRequired),
		fmt.Sprintf("Inregistrat la: %s", valueOrDash(item.RegisteredOn)),
		fmt.Sprintf("Aprobat la: %s", valueOrDash(item.ApprovedOn)),
		fmt.Sprintf("Responsabil: %s", valueOrDash(item.OwnerName)),
		fmt.Sprintf("Referinta fisier: %s", valueOrDash(item.FileReference)),
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Document managerial", fmt.Sprintf("document-managerial-%s", item.DocumentCode), lines)
}
