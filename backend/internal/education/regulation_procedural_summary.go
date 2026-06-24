package education

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type RegulationProceduralSummary struct {
	Regulation RegulationProceduralSummaryRegulation `json:"regulation"`
	Versions   RegulationProceduralSummaryVersions   `json:"versions"`
	Workflow   RegulationProceduralSummaryWorkflow   `json:"workflow"`
	Readiness  RegulationProceduralSummaryReadiness  `json:"readiness"`
}

type RegulationProceduralSummaryRegulation struct {
	ID             string `json:"id"`
	RegulationCode string `json:"regulation_code"`
	RegulationType string `json:"regulation_type"`
	Title          string `json:"title"`
	SchoolYear     string `json:"school_year"`
	Status         string `json:"status"`
	ApprovalStatus string `json:"approval_status"`
	ReviewDueOn    string `json:"review_due_on"`
	ApprovedOn     string `json:"approved_on"`
}

type RegulationProceduralSummaryVersions struct {
	TotalVersions        int                                       `json:"total_versions"`
	ConsultationVersions int                                       `json:"consultation_versions"`
	EndorsedVersions     int                                       `json:"endorsed_versions"`
	ApprovedVersions     int                                       `json:"approved_versions"`
	PublishedVersions    int                                       `json:"published_versions"`
	LatestVersion        *RegulationProceduralSummaryLatestVersion `json:"latest_version"`
}

type RegulationProceduralSummaryLatestVersion struct {
	ID            string `json:"id"`
	VersionLabel  string `json:"version_label"`
	VersionStatus string `json:"version_status"`
	ApprovedOn    string `json:"approved_on"`
	EffectiveFrom string `json:"effective_from"`
	PublishedOn   string `json:"published_on"`
	PreparedBy    string `json:"prepared_by"`
	FileReference string `json:"file_reference"`
}

type RegulationProceduralSummaryWorkflow struct {
	TotalPhases     int                                       `json:"total_phases"`
	CompletedPhases int                                       `json:"completed_phases"`
	OpenPhases      int                                       `json:"open_phases"`
	ReturnedPhases  int                                       `json:"returned_phases"`
	CancelledPhases int                                       `json:"cancelled_phases"`
	FeedbackCount   int                                       `json:"feedback_count"`
	CurrentPhase    *RegulationProceduralSummaryWorkflowPhase `json:"current_phase"`
}

type RegulationProceduralSummaryWorkflowPhase struct {
	ID                string `json:"id"`
	PhaseOrder        int    `json:"phase_order"`
	PhaseType         string `json:"phase_type"`
	Status            string `json:"status"`
	Audience          string `json:"audience"`
	StartedOn         string `json:"started_on"`
	DueOn             string `json:"due_on"`
	CompletedOn       string `json:"completed_on"`
	FeedbackCount     int    `json:"feedback_count"`
	DecisionReference string `json:"decision_reference"`
}

type RegulationProceduralSummaryReadiness struct {
	ReadyForCPEndorsement bool     `json:"ready_for_cp_endorsement"`
	ReadyForCAApproval    bool     `json:"ready_for_ca_approval"`
	ReadyForPublication   bool     `json:"ready_for_publication"`
	ReadyForReview        bool     `json:"ready_for_review"`
	Blockers              []string `json:"blockers"`
}

func (s *Service) RegulationProceduralSummary(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	institutionID := s.institutionID(r)

	var summary RegulationProceduralSummary
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			regulation_code,
			regulation_type,
			title,
			school_year,
			status,
			approval_status,
			to_char(review_due_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), '')
		from education_regulations
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Regulation.ID,
		&summary.Regulation.RegulationCode,
		&summary.Regulation.RegulationType,
		&summary.Regulation.Title,
		&summary.Regulation.SchoolYear,
		&summary.Regulation.Status,
		&summary.Regulation.ApprovalStatus,
		&summary.Regulation.ReviewDueOn,
		&summary.Regulation.ApprovedOn,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_versions,
			count(*) filter (where version_status = 'consultation') as consultation_versions,
			count(*) filter (where version_status = 'endorsed') as endorsed_versions,
			count(*) filter (where version_status = 'approved') as approved_versions,
			count(*) filter (where version_status = 'published') as published_versions
		from education_regulation_versions
		where regulation_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Versions.TotalVersions,
		&summary.Versions.ConsultationVersions,
		&summary.Versions.EndorsedVersions,
		&summary.Versions.ApprovedVersions,
		&summary.Versions.PublishedVersions,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	}

	var latestVersion RegulationProceduralSummaryLatestVersion
	err = s.pool.QueryRow(r.Context(), `
		select
			id::text,
			version_label,
			version_status,
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			to_char(effective_from, 'YYYY-MM-DD'),
			coalesce(to_char(published_on, 'YYYY-MM-DD'), ''),
			prepared_by,
			file_reference
		from education_regulation_versions
		where regulation_id = $1::uuid and institution_id = $2
		order by effective_from desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(
		&latestVersion.ID,
		&latestVersion.VersionLabel,
		&latestVersion.VersionStatus,
		&latestVersion.ApprovedOn,
		&latestVersion.EffectiveFrom,
		&latestVersion.PublishedOn,
		&latestVersion.PreparedBy,
		&latestVersion.FileReference,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		summary.Versions.LatestVersion = nil
	case err != nil:
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	default:
		summary.Versions.LatestVersion = &latestVersion
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_phases,
			count(*) filter (where status = 'completed') as completed_phases,
			count(*) filter (where status in ('pending', 'active')) as open_phases,
			count(*) filter (where status = 'returned') as returned_phases,
			count(*) filter (where status = 'cancelled') as cancelled_phases,
			coalesce(sum(feedback_count), 0) as feedback_count
		from education_regulation_workflow_steps
		where regulation_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Workflow.TotalPhases,
		&summary.Workflow.CompletedPhases,
		&summary.Workflow.OpenPhases,
		&summary.Workflow.ReturnedPhases,
		&summary.Workflow.CancelledPhases,
		&summary.Workflow.FeedbackCount,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	}

	var currentPhase RegulationProceduralSummaryWorkflowPhase
	err = s.pool.QueryRow(r.Context(), `
		select
			id::text,
			phase_order,
			phase_type,
			status,
			audience,
			to_char(started_on, 'YYYY-MM-DD'),
			to_char(due_on, 'YYYY-MM-DD'),
			coalesce(to_char(completed_on, 'YYYY-MM-DD'), ''),
			feedback_count,
			decision_reference
		from education_regulation_workflow_steps
		where regulation_id = $1::uuid and institution_id = $2
		order by
			case when status in ('active', 'returned', 'pending') then 0 else 1 end,
			phase_order desc,
			created_at desc
		limit 1
	`, recordID, institutionID).Scan(
		&currentPhase.ID,
		&currentPhase.PhaseOrder,
		&currentPhase.PhaseType,
		&currentPhase.Status,
		&currentPhase.Audience,
		&currentPhase.StartedOn,
		&currentPhase.DueOn,
		&currentPhase.CompletedOn,
		&currentPhase.FeedbackCount,
		&currentPhase.DecisionReference,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		summary.Workflow.CurrentPhase = nil
	case err != nil:
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	default:
		summary.Workflow.CurrentPhase = &currentPhase
	}

	var consultationPhaseCompleted bool
	var cpPhaseCompleted bool
	var caPhaseCompleted bool
	var registrationPhaseCompleted bool
	var publicationPhaseCompleted bool
	if err := s.pool.QueryRow(r.Context(), `
		select
			coalesce(bool_or(phase_type = 'consultare_publica' and status = 'completed'), false),
			coalesce(bool_or(phase_type = 'avizare_cp' and status = 'completed'), false),
			coalesce(bool_or(phase_type = 'aprobare_ca' and status = 'completed'), false),
			coalesce(bool_or(phase_type = 'inregistrare' and status = 'completed'), false),
			coalesce(bool_or(phase_type = 'publicare' and status = 'completed'), false)
		from education_regulation_workflow_steps
		where regulation_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&consultationPhaseCompleted,
		&cpPhaseCompleted,
		&caPhaseCompleted,
		&registrationPhaseCompleted,
		&publicationPhaseCompleted,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_procedural_summary_failed"})
		return
	}

	consultationCovered := summary.Versions.ConsultationVersions > 0 || consultationPhaseCompleted
	cpCovered := summary.Versions.EndorsedVersions > 0 || summary.Versions.ApprovedVersions > 0 || summary.Versions.PublishedVersions > 0 || cpPhaseCompleted
	caCovered := summary.Versions.ApprovedVersions > 0 || summary.Versions.PublishedVersions > 0 || caPhaseCompleted || summary.Regulation.ApprovalStatus == "ca_approved"
	publicationCovered := summary.Versions.PublishedVersions > 0 || publicationPhaseCompleted || summary.Regulation.Status == "published"
	registrationCovered := registrationPhaseCompleted || summary.Regulation.ApprovedOn != ""

	blockers := make([]string, 0)
	if summary.Versions.TotalVersions == 0 {
		blockers = append(blockers, "regulation_version_missing")
	}
	if !consultationCovered {
		blockers = append(blockers, "public_consultation_missing")
	}
	if !cpCovered {
		blockers = append(blockers, "cp_endorsement_missing")
	}
	if !caCovered {
		blockers = append(blockers, "ca_approval_missing")
	}
	if caCovered && !registrationCovered {
		blockers = append(blockers, "registration_missing")
	}
	if caCovered && !publicationCovered {
		blockers = append(blockers, "publication_missing")
	}
	if reviewDueOn, err := time.Parse("2006-01-02", summary.Regulation.ReviewDueOn); err == nil && reviewDueOn.Before(time.Now()) {
		blockers = append(blockers, "review_overdue")
	}

	summary.Readiness = RegulationProceduralSummaryReadiness{
		ReadyForCPEndorsement: summary.Versions.TotalVersions > 0 && consultationCovered && !cpCovered,
		ReadyForCAApproval:    cpCovered && !caCovered,
		ReadyForPublication:   caCovered && registrationCovered && !publicationCovered,
		ReadyForReview:        summary.Regulation.ReviewDueOn != "",
		Blockers:              blockers,
	}

	httpx.JSON(w, http.StatusOK, summary)
}
