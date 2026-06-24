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

func (s *Service) GovernanceMinuteItemPDF(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type minuteDocument struct {
		MeetingTitle        string
		Organism            string
		MeetingDate         string
		AgendaOrder         int
		TopicTitle          string
		DiscussionSummary   string
		DecisionSummary     string
		ResponsibleParty    string
		DueOn               string
		FollowUpStatus      string
		RequiresPublication bool
		Notes               string
	}

	var item minuteDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			em.title,
			em.organism,
			to_char(em.meeting_date, 'YYYY-MM-DD'),
			emm.agenda_order,
			emm.topic_title,
			emm.discussion_summary,
			emm.decision_summary,
			emm.responsible_party,
			coalesce(to_char(emm.due_on, 'YYYY-MM-DD'), ''),
			emm.follow_up_status,
			emm.requires_publication,
			emm.notes
		from education_meeting_minutes emm
		join education_meetings em on em.id = emm.meeting_id
		where emm.id = $1 and emm.meeting_id = $2 and emm.institution_id = $3
	`, recordID, meetingID, s.institutionID(r)).Scan(
		&item.MeetingTitle,
		&item.Organism,
		&item.MeetingDate,
		&item.AgendaOrder,
		&item.TopicTitle,
		&item.DiscussionSummary,
		&item.DecisionSummary,
		&item.ResponsibleParty,
		&item.DueOn,
		&item.FollowUpStatus,
		&item.RequiresPublication,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_meeting_minute_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_minute_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Extras proces-verbal sedinta",
		"",
		"Referinta sedinta",
		fmt.Sprintf("Sedinta: %s", item.MeetingTitle),
		fmt.Sprintf("Organism: %s", strings.ToUpper(item.Organism)),
		fmt.Sprintf("Data sedintei: %s", item.MeetingDate),
		"",
		"Element proces-verbal",
		fmt.Sprintf("Ordine pe agenda: %d", item.AgendaOrder),
		fmt.Sprintf("Subiect: %s", item.TopicTitle),
		fmt.Sprintf("Responsabil: %s", valueOrDash(item.ResponsibleParty)),
		fmt.Sprintf("Termen: %s", valueOrDash(item.DueOn)),
		fmt.Sprintf("Status urmarire: %s", item.FollowUpStatus),
		fmt.Sprintf("Necesita publicare: %t", item.RequiresPublication),
		"",
		"Rezumat dezbateri",
		item.DiscussionSummary,
		"",
		"Rezumat decizie / masura",
		item.DecisionSummary,
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Extras proces-verbal sedinta", fmt.Sprintf("proces-verbal-%s-pct-%d", meetingID, item.AgendaOrder), lines)
}

func (s *Service) GovernanceResolutionPDF(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))

	type resolutionDocument struct {
		MeetingTitle       string
		Organism           string
		MeetingDate        string
		ResolutionCode     string
		Title              string
		ResolutionType     string
		PublicationStatus  string
		AnonymizationState string
		IssuedOn           string
		SignedBy           string
		VoteSubjectTitle   string
		VoteOutcome        string
		LegalBasis         string
		Notes              string
	}

	var item resolutionDocument
	err := s.pool.QueryRow(r.Context(), `
		select
			em.title,
			em.organism,
			to_char(em.meeting_date, 'YYYY-MM-DD'),
			emr.resolution_code,
			emr.title,
			emr.resolution_type,
			emr.publication_status,
			emr.anonymization_state,
			to_char(emr.issued_on, 'YYYY-MM-DD'),
			emr.signed_by,
			emv.subject_title,
			emv.outcome,
			emv.legal_basis,
			emr.notes
		from education_meeting_resolutions emr
		join education_meetings em on em.id = emr.meeting_id
		join education_meeting_votes emv on emv.id = emr.vote_id
		where emr.id = $1 and emr.meeting_id = $2 and emr.institution_id = $3
	`, recordID, meetingID, s.institutionID(r)).Scan(
		&item.MeetingTitle,
		&item.Organism,
		&item.MeetingDate,
		&item.ResolutionCode,
		&item.Title,
		&item.ResolutionType,
		&item.PublicationStatus,
		&item.AnonymizationState,
		&item.IssuedOn,
		&item.SignedBy,
		&item.VoteSubjectTitle,
		&item.VoteOutcome,
		&item.LegalBasis,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_meeting_resolution_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_resolution_pdf_failed"})
		return
	}

	lines := []string{
		"Document: Hotarare / aviz sedinta",
		"",
		"Referinta sedinta",
		fmt.Sprintf("Sedinta: %s", item.MeetingTitle),
		fmt.Sprintf("Organism: %s", strings.ToUpper(item.Organism)),
		fmt.Sprintf("Data sedintei: %s", item.MeetingDate),
		"",
		"Date act",
		fmt.Sprintf("Cod: %s", item.ResolutionCode),
		fmt.Sprintf("Titlu: %s", item.Title),
		fmt.Sprintf("Tip act: %s", item.ResolutionType),
		fmt.Sprintf("Data emiterii: %s", item.IssuedOn),
		fmt.Sprintf("Semnat de: %s", valueOrDash(item.SignedBy)),
		fmt.Sprintf("Status publicare: %s", item.PublicationStatus),
		fmt.Sprintf("Status anonimizare: %s", item.AnonymizationState),
		"",
		"Referinta vot",
		fmt.Sprintf("Subiect: %s", item.VoteSubjectTitle),
		fmt.Sprintf("Rezultat vot: %s", item.VoteOutcome),
	}
	if strings.TrimSpace(item.LegalBasis) != "" {
		lines = append(lines, fmt.Sprintf("Temei legal: %s", item.LegalBasis))
	}
	if strings.TrimSpace(item.Notes) != "" {
		lines = append(lines, "", "Note", item.Notes)
	}

	writeEducationPDFDownload(w, "Hotarare / aviz sedinta", fmt.Sprintf("hotarare-%s", item.ResolutionCode), lines)
}
