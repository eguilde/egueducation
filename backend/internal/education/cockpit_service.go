package education

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) DirectorCockpit(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	schoolYear, err := s.resolveDirectorCockpitSchoolYear(r, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}

	response := DirectorCockpitResponse{
		SchoolYear:    schoolYear,
		InstitutionID: institutionID,
		RecommendedLinks: []DirectorCockpitQuickLink{
			{Key: "governance", Label: "Sedinte si hotarari", Route: "/education/governance"},
			{Key: "portfolios", Label: "Portofolii profesionale", Route: "/education/portfolio"},
			{Key: "personnel", Label: "Evaluari si personal", Route: "/education/personnel"},
			{Key: "compliance", Label: "Conformitate si publicare", Route: "/education/compliance"},
		},
	}

	if err := s.loadDirectorCockpitGovernance(r, institutionID, schoolYear, &response.Governance); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}
	if err := s.loadDirectorCockpitPortfolios(r, institutionID, schoolYear, &response.Portfolios); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}
	if err := s.loadDirectorCockpitEvaluations(r, institutionID, schoolYear, &response.Evaluations); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}
	if err := s.loadDirectorCockpitManagerial(r, institutionID, schoolYear, &response.Managerial); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}
	if err := s.loadDirectorCockpitPersonnel(r, institutionID, schoolYear, &response.Personnel); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}
	if err := s.loadDirectorCockpitCompliance(r, institutionID, &response.Compliance); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_director_cockpit_failed"})
		return
	}

	response.Alerts = buildDirectorCockpitAlerts(response)
	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) resolveDirectorCockpitSchoolYear(r *http.Request, institutionID string) (string, error) {
	if value := strings.TrimSpace(r.URL.Query().Get("school_year")); value != "" {
		return value, nil
	}

	var schoolYear string
	err := s.pool.QueryRow(r.Context(), `
		select code
		from education_taxonomies
		where domain = 'school_year' and active = true
		order by sort_order desc, code desc
		limit 1
	`).Scan(&schoolYear)
	if err == nil {
		return schoolYear, nil
	}

	err = s.pool.QueryRow(r.Context(), `
		select school_year
		from education_meetings
		where institution_id = $1
		order by school_year desc
		limit 1
	`, institutionID).Scan(&schoolYear)
	if err == nil {
		return schoolYear, nil
	}

	return "", nil
}

func (s *Service) loadDirectorCockpitGovernance(r *http.Request, institutionID string, schoolYear string, target *DirectorCockpitGovernance) error {
	return s.pool.QueryRow(r.Context(), `
		with filtered_meetings as (
			select id, status
			from education_meetings
			where institution_id = $1
				and ($2 = '' or school_year = $2)
		)
		select
			count(*) as total_meetings,
			count(*) filter (where status = 'scheduled') as scheduled_meetings,
			count(*) filter (
				where not exists (
					select 1 from education_meeting_minutes emm where emm.meeting_id = filtered_meetings.id and emm.institution_id = $1
				)
			) as meetings_without_minute,
			count(*) filter (
				where not exists (
					select 1 from education_meeting_votes emv where emv.meeting_id = filtered_meetings.id and emv.institution_id = $1
				)
			) as meetings_without_vote,
			coalesce((
				select count(*)
				from education_meeting_resolutions emr
				join education_meetings em on em.id = emr.meeting_id
				where emr.institution_id = $1
					and em.institution_id = $1
					and ($2 = '' or em.school_year = $2)
					and emr.publication_status = 'publicat'
			), 0) as published_resolutions
		from filtered_meetings
	`, institutionID, schoolYear).Scan(
		&target.TotalMeetings,
		&target.ScheduledMeetings,
		&target.MeetingsWithoutMinute,
		&target.MeetingsWithoutVote,
		&target.PublishedResolutions,
	)
}

func (s *Service) loadDirectorCockpitPortfolios(r *http.Request, institutionID string, schoolYear string, target *DirectorCockpitPortfolios) error {
	return s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status = 'draft') as draft_records,
			count(*) filter (where status = 'in_verificare') as review_records,
			count(*) filter (where status = 'returnat_pentru_completari') as returned_records,
			count(*) filter (where status = 'validated') as validated_records,
			count(*) filter (where transfer_status in ('trimis', 'transfer_solicitat', 'transferat')) as transfer_in_progress
		from education_portfolios
		where institution_id = $1
			and ($2 = '' or school_year = $2)
	`, institutionID, schoolYear).Scan(
		&target.TotalRecords,
		&target.DraftRecords,
		&target.ReviewRecords,
		&target.ReturnedRecords,
		&target.ValidatedRecords,
		&target.TransferInProgress,
	)
}

func (s *Service) loadDirectorCockpitEvaluations(r *http.Request, institutionID string, schoolYear string, target *DirectorCockpitEvaluations) error {
	return s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status = 'submitted') as submitted_records,
			count(*) filter (where status = 'reviewed') as reviewed_records,
			count(*) filter (where status = 'contested') as contested_records,
			count(*) filter (where status = 'approved') as approved_records,
			count(*) filter (
				where exists (
					select 1
					from education_evaluation_result_issues eeri
					where eeri.evaluation_id = ee.id
						and eeri.institution_id = $1
				)
			) as communicated_documents
		from education_evaluations ee
		where institution_id = $1
			and ($2 = '' or school_year = $2)
	`, institutionID, schoolYear).Scan(
		&target.TotalRecords,
		&target.SubmittedRecords,
		&target.ReviewedRecords,
		&target.ContestedRecords,
		&target.ApprovedRecords,
		&target.CommunicatedDocuments,
	)
}

func (s *Service) loadDirectorCockpitManagerial(r *http.Request, institutionID string, schoolYear string, target *DirectorCockpitManagerial) error {
	return s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_dossiers,
			count(*) filter (where status = 'draft') as draft_dossiers,
			count(*) filter (where status in ('in_review', 'consultation')) as review_dossiers,
			count(*) filter (where status = 'approved') as approved_dossiers,
			coalesce((
				select count(*)
				from education_managerial_documents emd
				join education_managerial_dossiers emds on emds.id = emd.dossier_id
				where emd.institution_id = $1
					and emds.institution_id = $1
					and ($2 = '' or emds.school_year = $2)
					and emd.status = 'published'
			), 0) as published_documents,
			coalesce((
				select count(*)
				from education_managerial_workflow_steps emws
				join education_managerial_dossiers emds on emds.id = emws.dossier_id
				where emws.institution_id = $1
					and emds.institution_id = $1
					and ($2 = '' or emds.school_year = $2)
					and emws.status in ('pending', 'in_progress', 'returned')
			), 0) as workflow_open_steps
		from education_managerial_dossiers
		where institution_id = $1
			and ($2 = '' or school_year = $2)
	`, institutionID, schoolYear).Scan(
		&target.TotalDossiers,
		&target.DraftDossiers,
		&target.ReviewDossiers,
		&target.ApprovedDossiers,
		&target.PublishedDocuments,
		&target.WorkflowOpenSteps,
	)
}

func (s *Service) loadDirectorCockpitPersonnel(r *http.Request, institutionID string, schoolYear string, target *DirectorCockpitPersonnel) error {
	return s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status = 'active') as active_records,
			count(*) filter (where has_portfolio = true) as portfolio_enabled,
			count(*) filter (where evaluation_status in ('draft', 'in_review')) as evaluation_pending,
			count(*) filter (where mobility_stage <> 'none') as mobility_cases
		from education_personnel
		where institution_id = $1
			and ($2 = '' or school_year = $2)
	`, institutionID, schoolYear).Scan(
		&target.TotalRecords,
		&target.ActiveRecords,
		&target.PortfolioEnabled,
		&target.EvaluationPending,
		&target.MobilityCases,
	)
}

func (s *Service) loadDirectorCockpitCompliance(r *http.Request, institutionID string, target *DirectorCockpitCompliance) error {
	return s.pool.QueryRow(r.Context(), `
		select
			coalesce((select count(*) from education_requirement_catalog), 0) as total_requirements,
			coalesce((select count(*) from education_requirement_catalog where implementation_status = 'implemented'), 0) as implemented_requirements,
			coalesce((select count(*) from education_requirement_catalog where implementation_status = 'partial'), 0) as partial_requirements,
			coalesce((select count(*) from education_publications where institution_id = $1 and publication_status <> 'published'), 0) as pending_publications,
			coalesce((select count(*) from education_publications where institution_id = $1 and anonymization_status in ('necesara', 'pending_anonymization')), 0) as anonymization_pending
	`, institutionID).Scan(
		&target.TotalRequirements,
		&target.ImplementedRequirements,
		&target.PartialRequirements,
		&target.PendingPublications,
		&target.AnonymizationPending,
	)
}

func buildDirectorCockpitAlerts(response DirectorCockpitResponse) []DirectorCockpitAlert {
	alerts := make([]DirectorCockpitAlert, 0, 8)
	appendAlert := func(condition bool, alert DirectorCockpitAlert) {
		if condition {
			alerts = append(alerts, alert)
		}
	}

	appendAlert(response.Governance.MeetingsWithoutMinute > 0, DirectorCockpitAlert{
		ID:       "meetings-without-minute",
		Title:    "Sedinte fara minute",
		Summary:  fmt.Sprintf("%d sedinte au nevoie de minute completate sau verificate.", response.Governance.MeetingsWithoutMinute),
		Status:   "in_verificare",
		Route:    "/education/governance",
		Priority: 1,
	})
	appendAlert(response.Governance.MeetingsWithoutVote > 0, DirectorCockpitAlert{
		ID:       "meetings-without-votes",
		Title:    "Sedinte fara voturi inregistrate",
		Summary:  fmt.Sprintf("%d sedinte nu au voturi consemnate in sistem.", response.Governance.MeetingsWithoutVote),
		Status:   "scheduled",
		Route:    "/education/governance",
		Priority: 2,
	})
	appendAlert(response.Portfolios.ReturnedRecords > 0, DirectorCockpitAlert{
		ID:       "returned-portfolios",
		Title:    "Portofolii returnate pentru completari",
		Summary:  fmt.Sprintf("%d portofolii trebuie completate si reverificate.", response.Portfolios.ReturnedRecords),
		Status:   "returnat",
		Route:    "/education/portfolio",
		Priority: 1,
	})
	appendAlert(response.Evaluations.ContestedRecords > 0, DirectorCockpitAlert{
		ID:       "contested-evaluations",
		Title:    "Evaluari contestate",
		Summary:  fmt.Sprintf("%d evaluari au contestatii active sau recente.", response.Evaluations.ContestedRecords),
		Status:   "contested",
		Route:    "/education/personnel",
		Priority: 1,
	})
	appendAlert(response.Managerial.WorkflowOpenSteps > 0, DirectorCockpitAlert{
		ID:       "open-managerial-workflows",
		Title:    "Workflow-uri manageriale deschise",
		Summary:  fmt.Sprintf("%d pasi de workflow managerial sunt in lucru sau in asteptare.", response.Managerial.WorkflowOpenSteps),
		Status:   "in_progress",
		Route:    "/education/governance",
		Priority: 2,
	})
	appendAlert(response.Compliance.PendingPublications > 0, DirectorCockpitAlert{
		ID:       "pending-publications",
		Title:    "Publicari restante",
		Summary:  fmt.Sprintf("%d documente sunt in asteptarea publicarii sau finalizarii fluxului.", response.Compliance.PendingPublications),
		Status:   "pending",
		Route:    "/education/compliance",
		Priority: 1,
	})
	appendAlert(response.Compliance.AnonymizationPending > 0, DirectorCockpitAlert{
		ID:       "anonymization-pending",
		Title:    "Anonimizare necesara",
		Summary:  fmt.Sprintf("%d inregistrari necesita anonimizare inainte de publicare.", response.Compliance.AnonymizationPending),
		Status:   "pending_anonymization",
		Route:    "/education/compliance",
		Priority: 2,
	})

	return alerts
}
