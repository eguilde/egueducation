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

func (s *Service) GovernanceMeetingDocumentPDF(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))

	var item GovernanceMeetingDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			emd.id::text,
			emd.meeting_id::text,
			emd.document_type,
			emd.title,
			emd.document_number,
			emd.registry_number,
			emd.publication_status,
			emd.custody_owner,
			emd.signed_by,
			to_char(emd.issued_on, 'YYYY-MM-DD'),
			emd.institution_id,
			emd.summary
		from education_meeting_documents emd
		where emd.id = $1 and emd.meeting_id = $2 and emd.institution_id = $3
	`, documentID, meetingID, s.institutionID(r)).Scan(
		&item.ID,
		&item.MeetingID,
		&item.DocumentType,
		&item.Title,
		&item.DocumentNumber,
		&item.RegistryNumber,
		&item.PublicationStatus,
		&item.CustodyOwner,
		&item.SignedBy,
		&item.IssuedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_meeting_document_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_document_pdf_failed"})
		return
	}

	titleByType := map[string]string{
		"convocator": "Convocator sedinta",
		"convocator_ca": "Convocator CA",
		"convocator_cp": "Convocator CP",
		"ordine_de_zi": "Ordine de zi",
		"prezenta": "Lista prezenta",
		"proces_verbal": "Proces-verbal oficial",
		"proces_verbal_ca": "Proces-verbal CA",
		"proces_verbal_cp": "Proces-verbal CP",
		"registru_ca": "Registru sedinte CA",
		"registru_cp": "Registru sedinte CP",
		"numire_secretar_cp": "Decizie de numire secretar CP",
		"anexa": "Anexa sedinta",
		"hotarare": "Hotarare sedinta",
		"material_sedinta": "Material sedinta",
		"delegare": "Delegare / imputernicire",
	}

	lines := []string{
		"Document: " + titleByType[item.DocumentType],
		"",
		"Date document",
		fmt.Sprintf("Tip: %s", item.DocumentType),
		fmt.Sprintf("Titlu: %s", item.Title),
		fmt.Sprintf("Numar document: %s", valueOrDash(item.DocumentNumber)),
		fmt.Sprintf("Numar registratura: %s", valueOrDash(item.RegistryNumber)),
		fmt.Sprintf("Status publicare: %s", item.PublicationStatus),
		fmt.Sprintf("Custode: %s", valueOrDash(item.CustodyOwner)),
		fmt.Sprintf("Semnat de: %s", valueOrDash(item.SignedBy)),
		fmt.Sprintf("Data emiterii: %s", valueOrDash(item.IssuedOn)),
	}
	if strings.TrimSpace(item.Summary) != "" {
		lines = append(lines, "", "Rezumat", item.Summary)
	}

	if title, ok := titleByType[item.DocumentType]; ok {
		writeEducationPDFDownload(w, title, fmt.Sprintf("document-sedinta-%s", item.DocumentNumber), lines)
		return
	}

	writeEducationPDFDownload(w, "Document sedinta", fmt.Sprintf("document-sedinta-%s", item.DocumentNumber), lines)
}
