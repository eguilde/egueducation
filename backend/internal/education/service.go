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

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

type Service struct {
	pool *appdb.SessionPool
}

func NewService(pool *appdb.SessionPool) *Service {
	return &Service{pool: pool}
}

func (s *Service) logAudit(r *http.Request, action string, targetType string, targetID string, summary string, details map[string]any) {
	_ = audit.Log(r.Context(), s.pool, audit.Event{
		ActorSubject: authruntime.CurrentSubjectFromRequest(r),
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Summary:      summary,
		Details:      details,
	})
}

func (s *Service) institutionID(r *http.Request) string {
	return strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
}

func writeEducationNotFound(w http.ResponseWriter, code string) {
	httpx.JSON(w, http.StatusNotFound, map[string]any{"code": code})
}

func (s *Service) ListTaxonomies(w http.ResponseWriter, r *http.Request) {
	domains := []string{}
	if raw := strings.TrimSpace(r.URL.Query().Get("domains")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			if value != "" {
				domains = append(domains, value)
			}
		}
	}

	sql := `
		select
			id::text,
			domain,
			code,
			label_ro,
			label_en,
			active,
			sort_order
		from education_taxonomies
		where active = true
	`
	args := []any{}
	if len(domains) > 0 {
		sql += " and domain = any($1)"
		args = append(args, domains)
	}
	sql += " order by domain, sort_order, code"

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_taxonomies_failed"})
		return
	}
	defer rows.Close()

	response := TaxonomyCatalogResponse{Items: map[string][]TaxonomyItem{}}
	for rows.Next() {
		var item TaxonomyItem
		if err := rows.Scan(
			&item.ID,
			&item.Domain,
			&item.Code,
			&item.LabelRO,
			&item.LabelEN,
			&item.Active,
			&item.SortOrder,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_taxonomies_failed"})
			return
		}
		response.Items[item.Domain] = append(response.Items[item.Domain], item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_taxonomies_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) GovernanceDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response GovernanceDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_meetings,
			count(*) filter (where status = 'scheduled') as scheduled_meetings,
			count(*) filter (where status = 'held') as held_meetings,
			count(*) filter (where status = 'published') as published_meetings
		from education_meetings
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalMeetings,
		&response.Stats.ScheduledMeetings,
		&response.Stats.HeldMeetings,
		&response.Stats.PublishedMeetings,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) GovernanceMeetings(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"title":          {},
			"school_year":    {},
			"organism":       {},
			"meeting_type":   {},
			"status":         {},
			"meeting_date":   {},
			"chairperson":    {},
			"secretary_name": {},
		},
		[]string{"title", "school_year", "organism", "meeting_type", "status", "meeting_date"},
	)

	whereClause, args := buildMeetingFilters(query.Filters, s.institutionID(r))

	var total int
	countSQL := "select count(*) from education_meetings em " + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meetings_failed"})
		return
	}

	sortField := meetingSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			em.id::text,
			em.school_year,
			em.organism,
			em.title,
			em.meeting_type,
			em.status,
			em.quorum_required,
			em.participants_count,
			to_char(em.meeting_date, 'YYYY-MM-DD') as meeting_date,
			em.location,
			em.chairperson,
			em.secretary_name,
			em.institution_id,
			em.summary
		from education_meetings em
		%s
		order by %s %s, em.meeting_date desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meetings_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceMeeting, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceMeeting
		if err := rows.Scan(
			&item.ID,
			&item.SchoolYear,
			&item.Organism,
			&item.Title,
			&item.MeetingType,
			&item.Status,
			&item.QuorumRequired,
			&item.ParticipantsCount,
			&item.MeetingDate,
			&item.Location,
			&item.Chairperson,
			&item.SecretaryName,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meetings_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meetings_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceFilters(w http.ResponseWriter, r *http.Request) {
	response := GovernanceFiltersResponse{
		SchoolYears:  []string{},
		Organisms:    []string{},
		MeetingTypes: []string{},
		Statuses:     []string{},
	}

	var err error
	if response.SchoolYears, err = s.taxonomyCodes(r, "school_year"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_filters_failed"})
		return
	}
	if response.Organisms, err = s.taxonomyCodes(r, "governance_organism"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_filters_failed"})
		return
	}
	if response.MeetingTypes, err = s.taxonomyCodes(r, "governance_meeting_type"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_filters_failed"})
		return
	}
	if response.Statuses, err = s.taxonomyCodes(r, "governance_status"); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateGovernanceMeeting(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateGovernanceMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Organism = strings.TrimSpace(req.Organism)
	req.Title = strings.TrimSpace(req.Title)
	req.MeetingType = strings.TrimSpace(req.MeetingType)
	req.Status = strings.TrimSpace(req.Status)
	req.MeetingDate = strings.TrimSpace(req.MeetingDate)
	req.Location = strings.TrimSpace(req.Location)
	req.Chairperson = strings.TrimSpace(req.Chairperson)
	req.SecretaryName = strings.TrimSpace(req.SecretaryName)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.Organism == "" || req.Title == "" || req.MeetingType == "" || req.Status == "" || req.MeetingDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_fields"})
		return
	}
	validOrganism, err := s.taxonomyExists(r, "governance_organism", req.Organism)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_create_failed"})
		return
	}
	validMeetingType, err := s.taxonomyExists(r, "governance_meeting_type", req.MeetingType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_create_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "governance_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_create_failed"})
		return
	}
	if !validOrganism || !validMeetingType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_fields"})
		return
	}
	if req.QuorumRequired < 1 || req.ParticipantsCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_counts"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.MeetingDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_date"})
		return
	}

	var item GovernanceMeeting
	err = s.pool.QueryRow(r.Context(), `
		insert into education_meetings (
			school_year,
			organism,
			title,
			meeting_type,
			status,
			quorum_required,
			participants_count,
			meeting_date,
			location,
			chairperson,
			secretary_name,
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		returning
			id::text,
			school_year,
			organism,
			title,
			meeting_type,
			status,
			quorum_required,
			participants_count,
			to_char(meeting_date, 'YYYY-MM-DD'),
			location,
			chairperson,
			secretary_name,
			institution_id,
			summary
	`,
		req.SchoolYear,
		req.Organism,
		req.Title,
		req.MeetingType,
		req.Status,
		req.QuorumRequired,
		req.ParticipantsCount,
		req.MeetingDate,
		req.Location,
		req.Chairperson,
		req.SecretaryName,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.MeetingType,
		&item.Status,
		&item.QuorumRequired,
		&item.ParticipantsCount,
		&item.MeetingDate,
		&item.Location,
		&item.Chairperson,
		&item.SecretaryName,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_create_failed"})
		return
	}

	s.logAudit(r, "education.governance.create", "governance_meeting", item.ID, "Governance meeting created.", map[string]any{
		"school_year":        item.SchoolYear,
		"organism":           item.Organism,
		"title":              item.Title,
		"meeting_type":       item.MeetingType,
		"status":             item.Status,
		"meeting_date":       item.MeetingDate,
		"participants_count": item.ParticipantsCount,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) DecisionDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response GovernanceDecisionDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_decisions,
			count(*) filter (where status = 'approved') as approved_decisions,
			count(*) filter (where publication_status = 'published') as published_decisions,
			count(*) filter (where publication_status <> 'published') as pending_publication
		from education_decisions
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalDecisions,
		&response.Stats.ApprovedDecisions,
		&response.Stats.PublishedDecisions,
		&response.Stats.PendingPublication,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decisions_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) GovernanceDecisions(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"decision_code":      {},
			"title":              {},
			"school_year":        {},
			"organism":           {},
			"status":             {},
			"publication_status": {},
			"decision_date":      {},
			"signed_by":          {},
		},
		[]string{"decision_code", "title", "school_year", "organism", "status", "publication_status", "decision_date"},
	)

	whereClause, args := buildDecisionFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_decisions ed "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decisions_failed"})
		return
	}

	sortField := decisionSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			ed.id::text,
			ed.decision_code,
			ed.school_year,
			ed.organism,
			ed.title,
			ed.status,
			ed.publication_status,
			to_char(ed.decision_date, 'YYYY-MM-DD'),
			ed.legal_basis,
			ed.signed_by,
			ed.institution_id,
			ed.summary
		from education_decisions ed
		%s
		order by %s %s, ed.decision_date desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decisions_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceDecision, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceDecision
		if err := rows.Scan(
			&item.ID,
			&item.DecisionCode,
			&item.SchoolYear,
			&item.Organism,
			&item.Title,
			&item.Status,
			&item.PublicationStatus,
			&item.DecisionDate,
			&item.LegalBasis,
			&item.SignedBy,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decisions_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decisions_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) DecisionFilters(w http.ResponseWriter, r *http.Request) {
	response := GovernanceDecisionFiltersResponse{
		SchoolYears:         []string{},
		Organisms:           []string{},
		Statuses:            []string{},
		PublicationStatuses: []string{},
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = s.loadDistinctValues(r, "select distinct school_year from education_decisions where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_filters_failed"})
		return
	}
	if response.Organisms, err = s.loadDistinctValues(r, "select distinct organism from education_decisions where institution_id = $1 order by organism", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadDistinctValues(r, "select distinct status from education_decisions where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_filters_failed"})
		return
	}
	if response.PublicationStatuses, err = s.loadDistinctValues(r, "select distinct publication_status from education_decisions where institution_id = $1 order by publication_status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateGovernanceDecision(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateGovernanceDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Organism = strings.TrimSpace(req.Organism)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.DecisionDate = strings.TrimSpace(req.DecisionDate)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.SignedBy = strings.TrimSpace(req.SignedBy)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.Organism == "" || req.Title == "" || req.Status == "" || req.PublicationStatus == "" || req.DecisionDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_fields"})
		return
	}
	validOrganism, err := s.taxonomyExists(r, "governance_organism", req.Organism)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_create_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "governance_decision_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_create_failed"})
		return
	}
	validPublicationStatus, err := s.taxonomyExists(r, "governance_publication_status", req.PublicationStatus)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_create_failed"})
		return
	}
	if !validOrganism || !validStatus || !validPublicationStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DecisionDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_date"})
		return
	}

	decisionCode := fmt.Sprintf("DEC-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item GovernanceDecision
	err = s.pool.QueryRow(r.Context(), `
		insert into education_decisions (
			decision_code,
			school_year,
			organism,
			title,
			status,
			publication_status,
			decision_date,
			legal_basis,
			signed_by,
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		returning
			id::text,
			decision_code,
			school_year,
			organism,
			title,
			status,
			publication_status,
			to_char(decision_date, 'YYYY-MM-DD'),
			legal_basis,
			signed_by,
			institution_id,
			summary
	`,
		decisionCode,
		req.SchoolYear,
		req.Organism,
		req.Title,
		req.Status,
		req.PublicationStatus,
		req.DecisionDate,
		req.LegalBasis,
		req.SignedBy,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.DecisionCode,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.Status,
		&item.PublicationStatus,
		&item.DecisionDate,
		&item.LegalBasis,
		&item.SignedBy,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_create_failed"})
		return
	}

	s.logAudit(r, "education.decisions.create", "governance_decision", item.ID, "Governance decision created.", map[string]any{
		"decision_code":      item.DecisionCode,
		"school_year":        item.SchoolYear,
		"organism":           item.Organism,
		"status":             item.Status,
		"publication_status": item.PublicationStatus,
		"decision_date":      item.DecisionDate,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) ManagerialDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response ManagerialDossierDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_dossiers,
			count(*) filter (where status = 'in_review') as review_dossiers,
			count(*) filter (where status = 'published') as published_dossiers,
			count(*) filter (where due_on < current_date and status not in ('published', 'archived')) as overdue_dossiers
		from education_managerial_dossiers
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalDossiers,
		&response.Stats.ReviewDossiers,
		&response.Stats.PublishedDossiers,
		&response.Stats.OverdueDossiers,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) ManagerialDossiers(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"dossier_code":         {},
			"title":                {},
			"school_year":          {},
			"dossier_type":         {},
			"status":               {},
			"due_on":               {},
			"publication_required": {},
			"owner_name":           {},
		},
		[]string{"dossier_code", "title", "school_year", "dossier_type", "status", "due_on"},
	)

	whereClause, args := buildManagerialFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_managerial_dossiers emd "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_failed"})
		return
	}

	sortField := managerialSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			emd.id::text,
			emd.dossier_code,
			emd.school_year,
			emd.dossier_type,
			emd.title,
			emd.status,
			emd.owner_name,
			to_char(emd.due_on, 'YYYY-MM-DD'),
			emd.publication_required,
			emd.institution_id,
			emd.summary
		from education_managerial_dossiers emd
		%s
		order by %s %s, emd.due_on asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_failed"})
		return
	}
	defer rows.Close()

	items := make([]ManagerialDossier, 0, query.PageSize)
	for rows.Next() {
		var item ManagerialDossier
		if err := rows.Scan(
			&item.ID,
			&item.DossierCode,
			&item.SchoolYear,
			&item.DossierType,
			&item.Title,
			&item.Status,
			&item.OwnerName,
			&item.DueOn,
			&item.PublicationRequired,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) ManagerialFilters(w http.ResponseWriter, r *http.Request) {
	response := ManagerialDossierFiltersResponse{
		SchoolYears:  []string{},
		DossierTypes: []string{},
		Statuses:     []string{},
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = s.loadDistinctValues(r, "select distinct school_year from education_managerial_dossiers where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_filters_failed"})
		return
	}
	if response.DossierTypes, err = s.loadDistinctValues(r, "select distinct dossier_type from education_managerial_dossiers where institution_id = $1 order by dossier_type", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadDistinctValues(r, "select distinct status from education_managerial_dossiers where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateManagerialDossier(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateManagerialDossierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.DossierType = strings.TrimSpace(req.DossierType)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.DossierType == "" || req.Title == "" || req.Status == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "managerial_dossier_type", req.DossierType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_create_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "managerial_dossier_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_create_failed"})
		return
	}
	if !validType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_due_on"})
		return
	}

	dossierCode := fmt.Sprintf("MGR-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item ManagerialDossier
	err = s.pool.QueryRow(r.Context(), `
		insert into education_managerial_dossiers (
			dossier_code,
			school_year,
			dossier_type,
			title,
			status,
			owner_name,
			due_on,
			publication_required,
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		returning
			id::text,
			dossier_code,
			school_year,
			dossier_type,
			title,
			status,
			owner_name,
			to_char(due_on, 'YYYY-MM-DD'),
			publication_required,
			institution_id,
			summary
	`,
		dossierCode,
		req.SchoolYear,
		req.DossierType,
		req.Title,
		req.Status,
		req.OwnerName,
		req.DueOn,
		req.PublicationRequired,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.DossierCode,
		&item.SchoolYear,
		&item.DossierType,
		&item.Title,
		&item.Status,
		&item.OwnerName,
		&item.DueOn,
		&item.PublicationRequired,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_create_failed"})
		return
	}

	s.logAudit(r, "education.managerial.create", "managerial_dossier", item.ID, "Managerial dossier created.", map[string]any{
		"dossier_code":         item.DossierCode,
		"school_year":          item.SchoolYear,
		"dossier_type":         item.DossierType,
		"status":               item.Status,
		"due_on":               item.DueOn,
		"publication_required": item.PublicationRequired,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) RegulationDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response RegulationDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_regulations,
			count(*) filter (where status = 'consultation') as consultation_items,
			count(*) filter (where status in ('approved', 'published')) as approved_regulations,
			count(*) filter (where status = 'published') as published_regulations
		from education_regulations
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalRegulations,
		&response.Stats.ConsultationItems,
		&response.Stats.ApprovedRegulations,
		&response.Stats.PublishedRegulations,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulations_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Regulations(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"regulation_code": {},
			"title":           {},
			"school_year":     {},
			"regulation_type": {},
			"status":          {},
			"approval_status": {},
			"review_due_on":   {},
			"owner_name":      {},
		},
		[]string{"regulation_code", "title", "school_year", "regulation_type", "status", "approval_status", "review_due_on"},
	)

	whereClause, args := buildRegulationFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_regulations er "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulations_failed"})
		return
	}

	sortField := regulationSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			er.id::text,
			er.regulation_code,
			er.school_year,
			er.regulation_type,
			er.title,
			er.status,
			er.approval_status,
			er.owner_name,
			to_char(er.review_due_on, 'YYYY-MM-DD'),
			coalesce(to_char(er.approved_on, 'YYYY-MM-DD'), ''),
			er.institution_id,
			er.summary
		from education_regulations er
		%s
		order by %s %s, er.review_due_on asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulations_failed"})
		return
	}
	defer rows.Close()

	items := make([]RegulationRecord, 0, query.PageSize)
	for rows.Next() {
		var item RegulationRecord
		if err := rows.Scan(
			&item.ID,
			&item.RegulationCode,
			&item.SchoolYear,
			&item.RegulationType,
			&item.Title,
			&item.Status,
			&item.ApprovalStatus,
			&item.OwnerName,
			&item.ReviewDueOn,
			&item.ApprovedOn,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulations_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulations_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) RegulationFilters(w http.ResponseWriter, r *http.Request) {
	response := RegulationFiltersResponse{
		SchoolYears:      []string{},
		RegulationTypes:  []string{},
		Statuses:         []string{},
		ApprovalStatuses: []string{},
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = s.loadDistinctValues(r, "select distinct school_year from education_regulations where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_filters_failed"})
		return
	}
	if response.RegulationTypes, err = s.loadDistinctValues(r, "select distinct regulation_type from education_regulations where institution_id = $1 order by regulation_type", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadDistinctValues(r, "select distinct status from education_regulations where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_filters_failed"})
		return
	}
	if response.ApprovalStatuses, err = s.loadDistinctValues(r, "select distinct approval_status from education_regulations where institution_id = $1 order by approval_status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateRegulation(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateRegulationRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.RegulationType = strings.TrimSpace(req.RegulationType)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.ApprovalStatus = strings.TrimSpace(req.ApprovalStatus)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.ReviewDueOn = strings.TrimSpace(req.ReviewDueOn)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.RegulationType == "" || req.Title == "" || req.Status == "" || req.ApprovalStatus == "" || req.ReviewDueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "education_regulation_type", req.RegulationType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_create_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_regulation_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_create_failed"})
		return
	}
	validApprovalStatus, err := s.taxonomyExists(r, "education_regulation_approval_status", req.ApprovalStatus)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_create_failed"})
		return
	}
	if !validType || !validStatus || !validApprovalStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReviewDueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_review_due_on"})
		return
	}
	if req.ApprovedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ApprovedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_approved_on"})
			return
		}
	}

	regulationCode := fmt.Sprintf("REGL-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item RegulationRecord
	err = s.pool.QueryRow(r.Context(), `
		insert into education_regulations (
			regulation_code,
			school_year,
			regulation_type,
			title,
			status,
			approval_status,
			owner_name,
			review_due_on,
			approved_on,
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,$8,nullif($9, '')::date,$10,$11)
		returning
			id::text,
			regulation_code,
			school_year,
			regulation_type,
			title,
			status,
			approval_status,
			owner_name,
			to_char(review_due_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		regulationCode,
		req.SchoolYear,
		req.RegulationType,
		req.Title,
		req.Status,
		req.ApprovalStatus,
		req.OwnerName,
		req.ReviewDueOn,
		req.ApprovedOn,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.RegulationCode,
		&item.SchoolYear,
		&item.RegulationType,
		&item.Title,
		&item.Status,
		&item.ApprovalStatus,
		&item.OwnerName,
		&item.ReviewDueOn,
		&item.ApprovedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_create_failed"})
		return
	}

	s.logAudit(r, "education.regulations.create", "education_regulation", item.ID, "Education regulation created.", map[string]any{
		"regulation_code": item.RegulationCode,
		"school_year":     item.SchoolYear,
		"regulation_type": item.RegulationType,
		"status":          item.Status,
		"approval_status": item.ApprovalStatus,
		"review_due_on":   item.ReviewDueOn,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) PersonnelDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response PersonnelDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status = 'active') as active_records,
			count(*) filter (where has_portfolio = true) as portfolios_enabled,
			count(*) filter (where mobility_stage <> 'none') as mobility_cases
		from education_personnel
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalRecords,
		&response.Stats.ActiveRecords,
		&response.Stats.PortfoliosEnabled,
		&response.Stats.MobilityCases,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) PersonnelRecords(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"employee_code":     {},
			"full_name":         {},
			"role_title":        {},
			"employment_type":   {},
			"status":            {},
			"evaluation_status": {},
			"mobility_stage":    {},
			"school_year":       {},
			"has_portfolio":     {},
		},
		[]string{"employee_code", "full_name", "employment_type", "status", "evaluation_status", "mobility_stage", "school_year", "has_portfolio"},
	)

	whereClause, args := buildPersonnelFilters(query.Filters, s.institutionID(r))

	var total int
	countSQL := "select count(*) from education_personnel ep " + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_failed"})
		return
	}

	sortField := personnelSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			ep.id::text,
			ep.employee_code,
			ep.full_name,
			ep.role_title,
			ep.employment_type,
			ep.status,
			ep.evaluation_status,
			ep.mobility_stage,
			ep.school_year,
			ep.assigned_unit,
			ep.phone,
			ep.email,
			ep.has_portfolio,
			ep.institution_id,
			ep.notes
		from education_personnel ep
		%s
		order by %s %s, ep.full_name asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelRecord, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelRecord
		if err := rows.Scan(
			&item.ID,
			&item.EmployeeCode,
			&item.FullName,
			&item.RoleTitle,
			&item.EmploymentType,
			&item.Status,
			&item.EvaluationStatus,
			&item.MobilityStage,
			&item.SchoolYear,
			&item.AssignedUnit,
			&item.Phone,
			&item.Email,
			&item.HasPortfolio,
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PersonnelFilters(w http.ResponseWriter, r *http.Request) {
	response := PersonnelFiltersResponse{
		SchoolYears:      []string{},
		EmploymentTypes:  []string{},
		Statuses:         []string{},
		EvaluationStatus: []string{},
		MobilityStages:   []string{},
	}

	load := func(sql string, args ...any) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = load("select distinct school_year from education_personnel where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_filters_failed"})
		return
	}
	if response.EmploymentTypes, err = load("select distinct employment_type from education_personnel where institution_id = $1 order by employment_type", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_filters_failed"})
		return
	}
	if response.Statuses, err = load("select distinct status from education_personnel where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_filters_failed"})
		return
	}
	if response.EvaluationStatus, err = load("select distinct evaluation_status from education_personnel where institution_id = $1 order by evaluation_status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_filters_failed"})
		return
	}
	if response.MobilityStages, err = load("select distinct mobility_stage from education_personnel where institution_id = $1 order by mobility_stage", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreatePersonnelRecord(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreatePersonnelRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.EmploymentType = strings.TrimSpace(req.EmploymentType)
	req.Status = strings.TrimSpace(req.Status)
	req.EvaluationStatus = strings.TrimSpace(req.EvaluationStatus)
	req.MobilityStage = strings.TrimSpace(req.MobilityStage)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.AssignedUnit = strings.TrimSpace(req.AssignedUnit)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.TrimSpace(req.Email)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleTitle == "" || req.EmploymentType == "" || req.Status == "" || req.EvaluationStatus == "" || req.MobilityStage == "" || req.SchoolYear == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_fields"})
		return
	}
	if !contains([]string{"titular", "suplinitor", "plata_cu_ora", "auxiliar"}, req.EmploymentType) ||
		!contains([]string{"active", "on_leave", "vacant", "inactive"}, req.Status) ||
		!contains([]string{"draft", "in_review", "finalized"}, req.EvaluationStatus) ||
		!contains([]string{"none", "transfer", "detasare", "restrangere"}, req.MobilityStage) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_fields"})
		return
	}

	employeeCode := fmt.Sprintf("PER-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item PersonnelRecord
	err := s.pool.QueryRow(r.Context(), `
		insert into education_personnel (
			employee_code,
			full_name,
			role_title,
			employment_type,
			status,
			evaluation_status,
			mobility_stage,
			school_year,
			assigned_unit,
			phone,
			email,
			has_portfolio,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		returning
			id::text,
			employee_code,
			full_name,
			role_title,
			employment_type,
			status,
			evaluation_status,
			mobility_stage,
			school_year,
			assigned_unit,
			phone,
			email,
			has_portfolio,
			institution_id,
			notes
	`,
		employeeCode,
		req.FullName,
		req.RoleTitle,
		req.EmploymentType,
		req.Status,
		req.EvaluationStatus,
		req.MobilityStage,
		req.SchoolYear,
		req.AssignedUnit,
		req.Phone,
		req.Email,
		req.HasPortfolio,
		institutionID,
		req.Notes,
	).Scan(
		&item.ID,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.EmploymentType,
		&item.Status,
		&item.EvaluationStatus,
		&item.MobilityStage,
		&item.SchoolYear,
		&item.AssignedUnit,
		&item.Phone,
		&item.Email,
		&item.HasPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_create_failed"})
		return
	}

	s.logAudit(r, "education.personnel.create", "personnel_record", item.ID, "Personnel record created.", map[string]any{
		"employee_code":     item.EmployeeCode,
		"full_name":         item.FullName,
		"role_title":        item.RoleTitle,
		"employment_type":   item.EmploymentType,
		"status":            item.Status,
		"evaluation_status": item.EvaluationStatus,
		"mobility_stage":    item.MobilityStage,
		"school_year":       item.SchoolYear,
		"has_portfolio":     item.HasPortfolio,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) EvaluationDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response PersonnelEvaluationDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_evaluations,
			count(*) filter (where status = 'submitted') as submitted_evaluations,
			count(*) filter (where status = 'approved') as approved_evaluations,
			count(*) filter (where status = 'contested') as contested_evaluations,
			count(distinct ee.id) filter (where exists (
				select 1
				from education_evaluation_result_issues eeri
				where eeri.evaluation_id = ee.id
					and eeri.institution_id = ee.institution_id
					and eeri.delivery_status in ('transmis', 'confirmat')
			)) as communicated_results
		from education_evaluations
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalEvaluations,
		&response.Stats.SubmittedEvaluations,
		&response.Stats.ApprovedEvaluations,
		&response.Stats.ContestedEvaluations,
		&response.Stats.CommunicatedResults,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluations_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Evaluations(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"evaluation_code": {},
			"employee_code":   {},
			"full_name":       {},
			"school_year":     {},
			"status":          {},
			"qualification":   {},
			"finalized_on":    {},
			"evaluator_name":  {},
		},
		[]string{"evaluation_code", "employee_code", "full_name", "school_year", "status", "qualification", "finalized_on", "score"},
	)

	whereClause, args := buildEvaluationFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_evaluations ee "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluations_failed"})
		return
	}

	sortField := evaluationSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			ee.id::text,
			ee.evaluation_code,
			ee.employee_code,
			ee.full_name,
			ee.role_title,
			ee.school_year,
			ee.status,
			ee.score,
			ee.qualification,
			ee.evaluator_name,
			coalesce(to_char(ee.finalized_on, 'YYYY-MM-DD'), ''),
			ee.institution_id,
			ee.summary
		from education_evaluations ee
		%s
		order by %s %s, ee.full_name asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluations_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelEvaluation, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelEvaluation
		if err := rows.Scan(
			&item.ID,
			&item.EvaluationCode,
			&item.EmployeeCode,
			&item.FullName,
			&item.RoleTitle,
			&item.SchoolYear,
			&item.Status,
			&item.Score,
			&item.Qualification,
			&item.EvaluatorName,
			&item.FinalizedOn,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluations_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluations_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) EvaluationFilters(w http.ResponseWriter, r *http.Request) {
	response := PersonnelEvaluationFiltersResponse{
		SchoolYears: []string{},
		Statuses:    []string{},
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = s.loadDistinctValues(r, "select distinct school_year from education_evaluations where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadDistinctValues(r, "select distinct status from education_evaluations where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateEvaluation(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreatePersonnelEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Status = strings.TrimSpace(req.Status)
	req.EvaluatorName = strings.TrimSpace(req.EvaluatorName)
	req.FinalizedOn = strings.TrimSpace(req.FinalizedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.EmployeeCode == "" || req.FullName == "" || req.RoleTitle == "" || req.SchoolYear == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_fields"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_evaluation_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_create_failed"})
		return
	}
	if !validStatus || !validSchoolYear || req.Score < 0 || req.Score > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_fields"})
		return
	}
	if inStringSet(req.Status, "reviewed", "approved") && req.EvaluatorName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_evaluator"})
		return
	}
	if req.Status == "approved" && req.FinalizedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_finalized_on"})
		return
	}
	if req.FinalizedOn != "" {
		if _, err := time.Parse("2006-01-02", req.FinalizedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_finalized_on"})
			return
		}
	}

	evaluationCode := fmt.Sprintf("EVAL-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)
	qualification := evaluationQualification(req.Score)

	var item PersonnelEvaluation
	err = s.pool.QueryRow(r.Context(), `
		insert into education_evaluations (
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
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,nullif($10, '')::date,$11,$12)
		returning
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
			coalesce(to_char(finalized_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		evaluationCode,
		req.EmployeeCode,
		req.FullName,
		req.RoleTitle,
		req.SchoolYear,
		req.Status,
		req.Score,
		qualification,
		req.EvaluatorName,
		req.FinalizedOn,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.EvaluationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.Status,
		&item.Score,
		&item.Qualification,
		&item.EvaluatorName,
		&item.FinalizedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_create_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.create", "personnel_evaluation", item.ID, "Personnel evaluation created.", map[string]any{
		"evaluation_code": item.EvaluationCode,
		"employee_code":   item.EmployeeCode,
		"full_name":       item.FullName,
		"status":          item.Status,
		"score":           item.Score,
	})
	if err := s.syncEvaluationDocumentToPersonnelFile(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_personnel_file_sync_failed"})
		return
	}
	if err := s.syncEvaluationPersonnelStateByID(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_personnel_state_sync_failed"})
		return
	}
	if err := s.syncEvaluationResultIssueEffects(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) DeclarationDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response PersonnelDeclarationDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_declarations,
			count(*) filter (where status = 'submitted') as submitted_declarations,
			count(*) filter (where status = 'validated') as validated_declarations,
			count(*) filter (where status = 'expired') as expired_declarations
		from education_declarations
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalDeclarations,
		&response.Stats.SubmittedDeclarations,
		&response.Stats.ValidatedDeclarations,
		&response.Stats.ExpiredDeclarations,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declarations_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) Declarations(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"declaration_code": {},
			"employee_code":    {},
			"full_name":        {},
			"declaration_type": {},
			"status":           {},
			"school_year":      {},
			"submitted_on":     {},
			"valid_until":      {},
		},
		[]string{"declaration_code", "employee_code", "full_name", "declaration_type", "status", "school_year", "submitted_on", "valid_until"},
	)

	whereClause, args := buildDeclarationFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_declarations ed "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declarations_failed"})
		return
	}

	sortField := declarationSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			ed.id::text,
			ed.declaration_code,
			ed.employee_code,
			ed.full_name,
			ed.declaration_type,
			ed.status,
			ed.school_year,
			to_char(ed.submitted_on, 'YYYY-MM-DD'),
			coalesce(to_char(ed.valid_until, 'YYYY-MM-DD'), ''),
			ed.institution_id,
			ed.summary
		from education_declarations ed
		%s
		order by %s %s, ed.submitted_on desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declarations_failed"})
		return
	}
	defer rows.Close()

	items := make([]PersonnelDeclaration, 0, query.PageSize)
	for rows.Next() {
		var item PersonnelDeclaration
		if err := rows.Scan(
			&item.ID,
			&item.DeclarationCode,
			&item.EmployeeCode,
			&item.FullName,
			&item.DeclarationType,
			&item.Status,
			&item.SchoolYear,
			&item.SubmittedOn,
			&item.ValidUntil,
			&item.InstitutionID,
			&item.Summary,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declarations_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declarations_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) DeclarationFilters(w http.ResponseWriter, r *http.Request) {
	response := PersonnelDeclarationFiltersResponse{
		SchoolYears:      []string{},
		DeclarationTypes: []string{},
		Statuses:         []string{},
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = s.loadDistinctValues(r, "select distinct school_year from education_declarations where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declaration_filters_failed"})
		return
	}
	if response.DeclarationTypes, err = s.loadDistinctValues(r, "select distinct declaration_type from education_declarations where institution_id = $1 order by declaration_type", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declaration_filters_failed"})
		return
	}
	if response.Statuses, err = s.loadDistinctValues(r, "select distinct status from education_declarations where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declaration_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateDeclaration(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreatePersonnelDeclarationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.DeclarationType = strings.TrimSpace(req.DeclarationType)
	req.Status = strings.TrimSpace(req.Status)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.ValidUntil = strings.TrimSpace(req.ValidUntil)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.EmployeeCode == "" || req.FullName == "" || req.DeclarationType == "" || req.Status == "" || req.SchoolYear == "" || req.SubmittedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_declaration_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "education_declaration_type", req.DeclarationType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_create_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_declaration_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_create_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_create_failed"})
		return
	}
	if !validType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.SubmittedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_submitted_on"})
		return
	}
	if req.ValidUntil != "" {
		if _, err := time.Parse("2006-01-02", req.ValidUntil); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_valid_until"})
			return
		}
	}

	declarationCode := fmt.Sprintf("DECL-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item PersonnelDeclaration
	err = s.pool.QueryRow(r.Context(), `
		insert into education_declarations (
			declaration_code,
			employee_code,
			full_name,
			declaration_type,
			status,
			school_year,
			submitted_on,
			valid_until,
			institution_id,
			summary
		) values ($1,$2,$3,$4,$5,$6,$7,nullif($8, '')::date,$9,$10)
		returning
			id::text,
			declaration_code,
			employee_code,
			full_name,
			declaration_type,
			status,
			school_year,
			to_char(submitted_on, 'YYYY-MM-DD'),
			coalesce(to_char(valid_until, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		declarationCode,
		req.EmployeeCode,
		req.FullName,
		req.DeclarationType,
		req.Status,
		req.SchoolYear,
		req.SubmittedOn,
		req.ValidUntil,
		institutionID,
		req.Summary,
	).Scan(
		&item.ID,
		&item.DeclarationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.DeclarationType,
		&item.Status,
		&item.SchoolYear,
		&item.SubmittedOn,
		&item.ValidUntil,
		&item.InstitutionID,
		&item.Summary,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_create_failed"})
		return
	}

	s.logAudit(r, "education.declarations.create", "personnel_declaration", item.ID, "Personnel declaration created.", map[string]any{
		"declaration_code": item.DeclarationCode,
		"employee_code":    item.EmployeeCode,
		"full_name":        item.FullName,
		"declaration_type": item.DeclarationType,
		"status":           item.Status,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) PortfolioDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response PortfolioDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_portfolios,
			count(*) filter (where status = 'validated') as validated_portfolios,
			count(*) filter (where transfer_status <> 'none') as transfer_portfolios,
			count(*) filter (where authenticity_declared = true) as declared_portfolios
		from education_portfolios
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalPortfolios,
		&response.Stats.ValidatedPortfolios,
		&response.Stats.TransferPortfolios,
		&response.Stats.DeclaredPortfolios,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) PortfolioRecords(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"portfolio_code":        {},
			"owner_name":            {},
			"school_year":           {},
			"status":                {},
			"transfer_status":       {},
			"authenticity_declared": {},
			"consent_captured":      {},
			"retention_until":       {},
		},
		[]string{"portfolio_code", "owner_name", "school_year", "status", "transfer_status", "authenticity_declared", "consent_captured", "retention_until"},
	)

	whereClause, args := buildPortfolioFilters(query.Filters, s.institutionID(r))

	var total int
	countSQL := "select count(*) from education_portfolios epf " + whereClause
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolios_failed"})
		return
	}

	sortField := portfolioSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			epf.id::text,
			epf.portfolio_code,
			epf.owner_name,
			epf.owner_role,
			epf.school_year,
			epf.status,
			epf.section_count,
			to_char(epf.last_updated_on, 'YYYY-MM-DD'),
			to_char(epf.retention_until, 'YYYY-MM-DD'),
			epf.transfer_status,
			epf.authenticity_declared,
			epf.consent_captured,
			epf.custodian,
			epf.institution_id,
			epf.notes
		from education_portfolios epf
		%s
		order by %s %s, epf.owner_name asc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolios_failed"})
		return
	}
	defer rows.Close()

	items := make([]PortfolioRecord, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioRecord
		if err := rows.Scan(
			&item.ID,
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
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolios_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolios_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioFilters(w http.ResponseWriter, r *http.Request) {
	response := PortfolioFiltersResponse{
		SchoolYears:    []string{},
		Statuses:       []string{},
		TransferStatus: []string{},
	}

	load := func(sql string, args ...any) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = load("select distinct school_year from education_portfolios where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_filters_failed"})
		return
	}
	if response.Statuses, err = load("select distinct status from education_portfolios where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_filters_failed"})
		return
	}
	if response.TransferStatus, err = load("select distinct transfer_status from education_portfolios where institution_id = $1 order by transfer_status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreatePortfolioRecord(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreatePortfolioRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_payload"})
		return
	}

	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.OwnerRole = strings.TrimSpace(req.OwnerRole)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Status = strings.TrimSpace(req.Status)
	req.LastUpdatedOn = strings.TrimSpace(req.LastUpdatedOn)
	req.RetentionUntil = strings.TrimSpace(req.RetentionUntil)
	req.TransferStatus = strings.TrimSpace(req.TransferStatus)
	req.Custodian = strings.TrimSpace(req.Custodian)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.OwnerName == "" || req.OwnerRole == "" || req.SchoolYear == "" || req.Status == "" || req.LastUpdatedOn == "" || req.RetentionUntil == "" || req.TransferStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_fields"})
		return
	}
	if !contains([]string{"draft", "submitted", "validated", "transferred", "archived"}, req.Status) ||
		!contains([]string{"none", "prepared", "sent", "received"}, req.TransferStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_fields"})
		return
	}
	if req.SectionCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_sections"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.LastUpdatedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_last_updated"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.RetentionUntil); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention"})
		return
	}

	portfolioCode := fmt.Sprintf("PORT-CD-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item PortfolioRecord
	err := s.pool.QueryRow(r.Context(), `
		insert into education_portfolios (
			portfolio_code,
			owner_name,
			owner_role,
			school_year,
			status,
			section_count,
			last_updated_on,
			retention_until,
			transfer_status,
			authenticity_declared,
			consent_captured,
			custodian,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		returning
			id::text,
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
			institution_id,
			notes
	`,
		portfolioCode,
		req.OwnerName,
		req.OwnerRole,
		req.SchoolYear,
		req.Status,
		req.SectionCount,
		req.LastUpdatedOn,
		req.RetentionUntil,
		req.TransferStatus,
		req.AuthenticityDeclared,
		req.ConsentCaptured,
		req.Custodian,
		institutionID,
		req.Notes,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_create_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.create", "portfolio_record", item.ID, "Portfolio record created.", map[string]any{
		"portfolio_code":        item.PortfolioCode,
		"owner_name":            item.OwnerName,
		"owner_role":            item.OwnerRole,
		"school_year":           item.SchoolYear,
		"status":                item.Status,
		"transfer_status":       item.TransferStatus,
		"authenticity_declared": item.AuthenticityDeclared,
		"consent_captured":      item.ConsentCaptured,
		"retention_until":       item.RetentionUntil,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) MobilityDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response MobilityDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_cases,
			count(*) filter (where status in ('open', 'pending')) as open_cases,
			count(*) filter (where status in ('approved', 'completed')) as approved_cases,
			count(*) filter (where request_type = 'transfer') as transfer_cases,
			(select count(*) from education_mobility_final_decisions emfd where emfd.institution_id = $1) as final_decisions,
			(select count(*) from education_mobility_result_issues emri where emri.institution_id = $1 and emri.delivery_status in ('emis', 'transmis', 'confirmat')) as communicated_results
		from education_mobility_cases
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalCases,
		&response.Stats.OpenCases,
		&response.Stats.ApprovedCases,
		&response.Stats.TransferCases,
		&response.Stats.FinalDecisions,
		&response.Stats.CommunicatedResults,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) MobilityCases(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"case_code":     {},
			"employee_code": {},
			"full_name":     {},
			"school_year":   {},
			"request_type":  {},
			"stage":         {},
			"status":        {},
			"submitted_on":  {},
		},
		[]string{"case_code", "employee_code", "full_name", "school_year", "request_type", "stage", "status", "submitted_on"},
	)

	whereClause, args := buildMobilityFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_mobility_cases emc "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_failed"})
		return
	}

	sortField := mobilitySortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			emc.id::text,
			emc.case_code,
			emc.employee_code,
			emc.full_name,
			emc.school_year,
			emc.request_type,
			emc.stage,
			emc.status,
			emc.source_school,
			emc.destination_school,
			to_char(emc.submitted_on, 'YYYY-MM-DD'),
			emc.reviewed_by,
			emc.institution_id,
			emc.notes
		from education_mobility_cases emc
		%s
		order by %s %s, emc.submitted_on desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_failed"})
		return
	}
	defer rows.Close()

	items := make([]MobilityCase, 0, query.PageSize)
	for rows.Next() {
		var item MobilityCase
		if err := rows.Scan(
			&item.ID,
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
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MobilityFilters(w http.ResponseWriter, r *http.Request) {
	response := MobilityFiltersResponse{
		SchoolYears:  []string{},
		RequestTypes: []string{},
		Stages:       []string{},
		Statuses:     []string{},
	}

	load := func(sql string, args ...any) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = load("select distinct school_year from education_mobility_cases where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_filters_failed"})
		return
	}
	if response.RequestTypes, err = load("select distinct request_type from education_mobility_cases where institution_id = $1 order by request_type", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_filters_failed"})
		return
	}
	if response.Stages, err = load("select distinct stage from education_mobility_cases where institution_id = $1 order by stage", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_filters_failed"})
		return
	}
	if response.Statuses, err = load("select distinct status from education_mobility_cases where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateMobilityCase(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateMobilityCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.RequestType = strings.TrimSpace(req.RequestType)
	req.Stage = strings.TrimSpace(req.Stage)
	req.Status = strings.TrimSpace(req.Status)
	req.SourceSchool = strings.TrimSpace(req.SourceSchool)
	req.DestinationSchool = strings.TrimSpace(req.DestinationSchool)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.ReviewedBy = strings.TrimSpace(req.ReviewedBy)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.EmployeeCode == "" || req.FullName == "" || req.SchoolYear == "" || req.RequestType == "" || req.Stage == "" || req.Status == "" || req.SubmittedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_fields"})
		return
	}
	if !contains([]string{"transfer", "detasare", "pretransfer", "restrangere"}, req.RequestType) ||
		!contains([]string{"draft", "submitted", "review", "approved", "completed"}, req.Stage) ||
		!contains([]string{"open", "pending", "approved", "rejected", "completed"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.SubmittedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_date"})
		return
	}

	caseCode := fmt.Sprintf("MOB-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item MobilityCase
	err := s.pool.QueryRow(r.Context(), `
		insert into education_mobility_cases (
			case_code,
			employee_code,
			full_name,
			school_year,
			request_type,
			stage,
			status,
			source_school,
			destination_school,
			submitted_on,
			reviewed_by,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		returning
			id::text,
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
			institution_id,
			notes
	`,
		caseCode,
		req.EmployeeCode,
		req.FullName,
		req.SchoolYear,
		req.RequestType,
		req.Stage,
		req.Status,
		req.SourceSchool,
		req.DestinationSchool,
		req.SubmittedOn,
		req.ReviewedBy,
		institutionID,
		req.Notes,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_create_failed"})
		return
	}

	s.logAudit(r, "education.mobility.create", "mobility_case", item.ID, "Mobility case created.", map[string]any{
		"case_code":          item.CaseCode,
		"employee_code":      item.EmployeeCode,
		"full_name":          item.FullName,
		"school_year":        item.SchoolYear,
		"request_type":       item.RequestType,
		"stage":              item.Stage,
		"status":             item.Status,
		"submitted_on":       item.SubmittedOn,
		"destination_school": item.DestinationSchool,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) MeritDashboard(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var response MeritGrantDashboardResponse
	err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_records,
			count(*) filter (where status in ('approved', 'funded')) as approved_records,
			count(*) filter (where funded = true) as funded_records,
			coalesce(avg(score), 0),
			(select count(*) from education_merit_final_decisions emfd where emfd.institution_id = $1) as final_decisions,
			(select count(*) from education_merit_result_issues emri where emri.institution_id = $1 and emri.delivery_status in ('emis', 'transmis', 'confirmat')) as communicated_results
		from education_merit_grants
		where institution_id = $1
	`, institutionID).Scan(
		&response.Stats.TotalRecords,
		&response.Stats.ApprovedRecords,
		&response.Stats.FundedRecords,
		&response.Stats.AverageScore,
		&response.Stats.FinalDecisions,
		&response.Stats.CommunicatedResults,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_dashboard_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) MeritGrants(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"grant_code":    {},
			"full_name":     {},
			"school_year":   {},
			"category":      {},
			"status":        {},
			"decision_date": {},
			"funded":        {},
		},
		[]string{"grant_code", "full_name", "school_year", "category", "status", "decision_date", "funded"},
	)

	whereClause, args := buildMeritFilters(query.Filters, s.institutionID(r))

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_merit_grants emg "+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_failed"})
		return
	}

	sortField := meritSortColumn(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	args = append(args, query.PageSize, offset)

	sql := fmt.Sprintf(`
		select
			emg.id::text,
			emg.grant_code,
			emg.full_name,
			emg.role_title,
			emg.school_year,
			emg.category,
			emg.status,
			emg.score,
			emg.committee_name,
			to_char(emg.decision_date, 'YYYY-MM-DD'),
			emg.funded,
			emg.institution_id,
			emg.notes
		from education_merit_grants emg
		%s
		order by %s %s, emg.decision_date desc
		limit $%d offset $%d
	`, whereClause, sortField, strings.ToUpper(query.Direction), len(args)-1, len(args))

	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_failed"})
		return
	}
	defer rows.Close()

	items := make([]MeritGrant, 0, query.PageSize)
	for rows.Next() {
		var item MeritGrant
		if err := rows.Scan(
			&item.ID,
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
			&item.InstitutionID,
			&item.Notes,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) MeritFilters(w http.ResponseWriter, r *http.Request) {
	response := MeritGrantFiltersResponse{
		SchoolYears: []string{},
		Categories:  []string{},
		Statuses:    []string{},
	}

	load := func(sql string, args ...any) ([]string, error) {
		rows, err := s.pool.Query(r.Context(), sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		values := []string{}
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		return values, rows.Err()
	}

	var err error
	institutionID := s.institutionID(r)
	if response.SchoolYears, err = load("select distinct school_year from education_merit_grants where institution_id = $1 order by school_year desc", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_filters_failed"})
		return
	}
	if response.Categories, err = load("select distinct category from education_merit_grants where institution_id = $1 order by category", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_filters_failed"})
		return
	}
	if response.Statuses, err = load("select distinct status from education_merit_grants where institution_id = $1 order by status", institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_filters_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) CreateMeritGrant(w http.ResponseWriter, r *http.Request) {
	institutionID := s.institutionID(r)
	var req CreateMeritGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Category = strings.TrimSpace(req.Category)
	req.Status = strings.TrimSpace(req.Status)
	req.CommitteeName = strings.TrimSpace(req.CommitteeName)
	req.DecisionDate = strings.TrimSpace(req.DecisionDate)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleTitle == "" || req.SchoolYear == "" || req.Category == "" || req.Status == "" || req.DecisionDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_fields"})
		return
	}
	if !contains([]string{"predare", "management", "consiliere", "auxiliar"}, req.Category) ||
		!contains([]string{"draft", "submitted", "evaluated", "approved", "funded"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_fields"})
		return
	}
	if req.Score < 0 || req.Score > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DecisionDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_date"})
		return
	}

	grantCode := fmt.Sprintf("GRM-%d-%04d", time.Now().UTC().Year(), time.Now().Unix()%10000)

	var item MeritGrant
	err := s.pool.QueryRow(r.Context(), `
		insert into education_merit_grants (
			grant_code,
			full_name,
			role_title,
			school_year,
			category,
			status,
			score,
			committee_name,
			decision_date,
			funded,
			institution_id,
			notes
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		returning
			id::text,
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
			institution_id,
			notes
	`,
		grantCode,
		req.FullName,
		req.RoleTitle,
		req.SchoolYear,
		req.Category,
		req.Status,
		req.Score,
		req.CommitteeName,
		req.DecisionDate,
		req.Funded,
		institutionID,
		req.Notes,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_create_failed"})
		return
	}

	s.logAudit(r, "education.gradatii.create", "merit_grant", item.ID, "Merit grant record created.", map[string]any{
		"grant_code":    item.GrantCode,
		"full_name":     item.FullName,
		"role_title":    item.RoleTitle,
		"school_year":   item.SchoolYear,
		"category":      item.Category,
		"status":        item.Status,
		"score":         item.Score,
		"decision_date": item.DecisionDate,
		"funded":        item.Funded,
	})

	httpx.JSON(w, http.StatusCreated, item)
}

func (s *Service) UpdateGovernanceMeeting(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateGovernanceMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Organism = strings.TrimSpace(req.Organism)
	req.Title = strings.TrimSpace(req.Title)
	req.MeetingType = strings.TrimSpace(req.MeetingType)
	req.Status = strings.TrimSpace(req.Status)
	req.MeetingDate = strings.TrimSpace(req.MeetingDate)
	req.Location = strings.TrimSpace(req.Location)
	req.Chairperson = strings.TrimSpace(req.Chairperson)
	req.SecretaryName = strings.TrimSpace(req.SecretaryName)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.Organism == "" || req.Title == "" || req.MeetingType == "" || req.Status == "" || req.MeetingDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_meeting_fields"})
		return
	}
	validOrganism, err := s.taxonomyExists(r, "governance_organism", req.Organism)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_update_failed"})
		return
	}
	validMeetingType, err := s.taxonomyExists(r, "governance_meeting_type", req.MeetingType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_update_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "governance_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_update_failed"})
		return
	}
	if !validOrganism || !validMeetingType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_fields"})
		return
	}
	if req.QuorumRequired < 1 || req.ParticipantsCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_counts"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.MeetingDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_date"})
		return
	}

	var item GovernanceMeeting
	err = s.pool.QueryRow(r.Context(), `
		update education_meetings
		set
			school_year = $1,
			organism = $2,
			title = $3,
			meeting_type = $4,
			status = $5,
			quorum_required = $6,
			participants_count = $7,
			meeting_date = $8,
			location = $9,
			chairperson = $10,
			secretary_name = $11,
			summary = $12,
			updated_at = now()
		where id = $13::uuid and institution_id = $14
		returning
			id::text,
			school_year,
			organism,
			title,
			meeting_type,
			status,
			quorum_required,
			participants_count,
			to_char(meeting_date, 'YYYY-MM-DD'),
			location,
			chairperson,
			secretary_name,
			institution_id,
			summary
	`,
		req.SchoolYear,
		req.Organism,
		req.Title,
		req.MeetingType,
		req.Status,
		req.QuorumRequired,
		req.ParticipantsCount,
		req.MeetingDate,
		req.Location,
		req.Chairperson,
		req.SecretaryName,
		req.Summary,
		meetingID,
		institutionID,
	).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.MeetingType,
		&item.Status,
		&item.QuorumRequired,
		&item.ParticipantsCount,
		&item.MeetingDate,
		&item.Location,
		&item.Chairperson,
		&item.SecretaryName,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "meeting_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_update_failed"})
		return
	}

	s.logAudit(r, "education.governance.update", "governance_meeting", item.ID, "Governance meeting updated.", map[string]any{
		"school_year":        item.SchoolYear,
		"organism":           item.Organism,
		"title":              item.Title,
		"meeting_type":       item.MeetingType,
		"status":             item.Status,
		"meeting_date":       item.MeetingDate,
		"participants_count": item.ParticipantsCount,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceMeeting(w http.ResponseWriter, r *http.Request) {
	meetingID := strings.TrimSpace(chi.URLParam(r, "meetingID"))
	if meetingID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_meeting_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_meetings
		where id = $1::uuid and institution_id = $2
	`, meetingID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "meeting_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "meeting_not_found")
		return
	}

	s.logAudit(r, "education.governance.delete", "governance_meeting", meetingID, "Governance meeting deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateGovernanceDecision(w http.ResponseWriter, r *http.Request) {
	decisionID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	if decisionID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateGovernanceDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Organism = strings.TrimSpace(req.Organism)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.PublicationStatus = strings.TrimSpace(req.PublicationStatus)
	req.DecisionDate = strings.TrimSpace(req.DecisionDate)
	req.LegalBasis = strings.TrimSpace(req.LegalBasis)
	req.SignedBy = strings.TrimSpace(req.SignedBy)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.Organism == "" || req.Title == "" || req.Status == "" || req.PublicationStatus == "" || req.DecisionDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_decision_fields"})
		return
	}
	validOrganism, err := s.taxonomyExists(r, "governance_organism", req.Organism)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_update_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "governance_decision_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_update_failed"})
		return
	}
	validPublicationStatus, err := s.taxonomyExists(r, "governance_publication_status", req.PublicationStatus)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_update_failed"})
		return
	}
	if !validOrganism || !validStatus || !validPublicationStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DecisionDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_date"})
		return
	}

	var item GovernanceDecision
	err = s.pool.QueryRow(r.Context(), `
		update education_decisions
		set
			school_year = $1,
			organism = $2,
			title = $3,
			status = $4,
			publication_status = $5,
			decision_date = $6,
			legal_basis = $7,
			signed_by = $8,
			summary = $9,
			updated_at = now()
		where id = $10::uuid and institution_id = $11
		returning
			id::text,
			decision_code,
			school_year,
			organism,
			title,
			status,
			publication_status,
			to_char(decision_date, 'YYYY-MM-DD'),
			legal_basis,
			signed_by,
			institution_id,
			summary
	`,
		req.SchoolYear,
		req.Organism,
		req.Title,
		req.Status,
		req.PublicationStatus,
		req.DecisionDate,
		req.LegalBasis,
		req.SignedBy,
		req.Summary,
		decisionID,
		institutionID,
	).Scan(
		&item.ID,
		&item.DecisionCode,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.Status,
		&item.PublicationStatus,
		&item.DecisionDate,
		&item.LegalBasis,
		&item.SignedBy,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_update_failed"})
		return
	}

	s.logAudit(r, "education.decisions.update", "governance_decision", item.ID, "Governance decision updated.", map[string]any{
		"decision_code":      item.DecisionCode,
		"school_year":        item.SchoolYear,
		"organism":           item.Organism,
		"status":             item.Status,
		"publication_status": item.PublicationStatus,
		"decision_date":      item.DecisionDate,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteGovernanceDecision(w http.ResponseWriter, r *http.Request) {
	decisionID := strings.TrimSpace(chi.URLParam(r, "decisionID"))
	if decisionID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_decision_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_decisions
		where id = $1::uuid and institution_id = $2
	`, decisionID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "decision_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "decision_not_found")
		return
	}

	s.logAudit(r, "education.decisions.delete", "governance_decision", decisionID, "Governance decision deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateManagerialDossier(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateManagerialDossierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.DossierType = strings.TrimSpace(req.DossierType)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.DueOn = strings.TrimSpace(req.DueOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.DossierType == "" || req.Title == "" || req.Status == "" || req.DueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_managerial_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "managerial_dossier_type", req.DossierType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_update_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "managerial_dossier_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_update_failed"})
		return
	}
	if !validType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_due_on"})
		return
	}

	var item ManagerialDossier
	err = s.pool.QueryRow(r.Context(), `
		update education_managerial_dossiers
		set
			school_year = $1,
			dossier_type = $2,
			title = $3,
			status = $4,
			owner_name = $5,
			due_on = $6,
			publication_required = $7,
			summary = $8,
			updated_at = now()
		where id = $9::uuid and institution_id = $10
		returning
			id::text,
			dossier_code,
			school_year,
			dossier_type,
			title,
			status,
			owner_name,
			to_char(due_on, 'YYYY-MM-DD'),
			publication_required,
			institution_id,
			summary
	`,
		req.SchoolYear,
		req.DossierType,
		req.Title,
		req.Status,
		req.OwnerName,
		req.DueOn,
		req.PublicationRequired,
		req.Summary,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
		&item.DossierCode,
		&item.SchoolYear,
		&item.DossierType,
		&item.Title,
		&item.Status,
		&item.OwnerName,
		&item.DueOn,
		&item.PublicationRequired,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "managerial_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_update_failed"})
		return
	}

	s.logAudit(r, "education.managerial.update", "managerial_dossier", item.ID, "Managerial dossier updated.", map[string]any{
		"dossier_code":         item.DossierCode,
		"school_year":          item.SchoolYear,
		"dossier_type":         item.DossierType,
		"status":               item.Status,
		"due_on":               item.DueOn,
		"publication_required": item.PublicationRequired,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteManagerialDossier(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_managerial_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_managerial_dossiers
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "managerial_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "managerial_record_not_found")
		return
	}

	s.logAudit(r, "education.managerial.delete", "managerial_dossier", recordID, "Managerial dossier deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateRegulation(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateRegulationRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_payload"})
		return
	}

	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.RegulationType = strings.TrimSpace(req.RegulationType)
	req.Title = strings.TrimSpace(req.Title)
	req.Status = strings.TrimSpace(req.Status)
	req.ApprovalStatus = strings.TrimSpace(req.ApprovalStatus)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.ReviewDueOn = strings.TrimSpace(req.ReviewDueOn)
	req.ApprovedOn = strings.TrimSpace(req.ApprovedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.SchoolYear == "" || req.RegulationType == "" || req.Title == "" || req.Status == "" || req.ApprovalStatus == "" || req.ReviewDueOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_regulation_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "education_regulation_type", req.RegulationType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_update_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_regulation_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_update_failed"})
		return
	}
	validApprovalStatus, err := s.taxonomyExists(r, "education_regulation_approval_status", req.ApprovalStatus)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_update_failed"})
		return
	}
	if !validType || !validStatus || !validApprovalStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReviewDueOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_review_due_on"})
		return
	}
	if req.ApprovedOn != "" {
		if _, err := time.Parse("2006-01-02", req.ApprovedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_approved_on"})
			return
		}
	}

	var item RegulationRecord
	err = s.pool.QueryRow(r.Context(), `
		update education_regulations
		set
			school_year = $1,
			regulation_type = $2,
			title = $3,
			status = $4,
			approval_status = $5,
			owner_name = $6,
			review_due_on = $7,
			approved_on = nullif($8, '')::date,
			summary = $9,
			updated_at = now()
		where id = $10::uuid and institution_id = $11
		returning
			id::text,
			regulation_code,
			school_year,
			regulation_type,
			title,
			status,
			approval_status,
			owner_name,
			to_char(review_due_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		req.SchoolYear,
		req.RegulationType,
		req.Title,
		req.Status,
		req.ApprovalStatus,
		req.OwnerName,
		req.ReviewDueOn,
		req.ApprovedOn,
		req.Summary,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
		&item.RegulationCode,
		&item.SchoolYear,
		&item.RegulationType,
		&item.Title,
		&item.Status,
		&item.ApprovalStatus,
		&item.OwnerName,
		&item.ReviewDueOn,
		&item.ApprovedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "regulation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_update_failed"})
		return
	}

	s.logAudit(r, "education.regulations.update", "education_regulation", item.ID, "Education regulation updated.", map[string]any{
		"regulation_code": item.RegulationCode,
		"school_year":     item.SchoolYear,
		"regulation_type": item.RegulationType,
		"status":          item.Status,
		"approval_status": item.ApprovalStatus,
		"review_due_on":   item.ReviewDueOn,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteRegulation(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulation_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_regulations
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulation_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "regulation_not_found")
		return
	}

	s.logAudit(r, "education.regulations.delete", "education_regulation", recordID, "Education regulation deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdatePersonnelRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreatePersonnelRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.EmploymentType = strings.TrimSpace(req.EmploymentType)
	req.Status = strings.TrimSpace(req.Status)
	req.EvaluationStatus = strings.TrimSpace(req.EvaluationStatus)
	req.MobilityStage = strings.TrimSpace(req.MobilityStage)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.AssignedUnit = strings.TrimSpace(req.AssignedUnit)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.TrimSpace(req.Email)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleTitle == "" || req.EmploymentType == "" || req.Status == "" || req.EvaluationStatus == "" || req.MobilityStage == "" || req.SchoolYear == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_personnel_fields"})
		return
	}
	if !contains([]string{"titular", "suplinitor", "plata_cu_ora", "auxiliar"}, req.EmploymentType) ||
		!contains([]string{"active", "on_leave", "vacant", "inactive"}, req.Status) ||
		!contains([]string{"draft", "in_review", "finalized"}, req.EvaluationStatus) ||
		!contains([]string{"none", "transfer", "detasare", "restrangere"}, req.MobilityStage) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_fields"})
		return
	}

	var item PersonnelRecord
	err := s.pool.QueryRow(r.Context(), `
		update education_personnel
		set
			full_name = $1,
			role_title = $2,
			employment_type = $3,
			status = $4,
			evaluation_status = $5,
			mobility_stage = $6,
			school_year = $7,
			assigned_unit = $8,
			phone = $9,
			email = $10,
			has_portfolio = $11,
			notes = $12,
			updated_at = now()
		where id = $13::uuid and institution_id = $14
		returning
			id::text,
			employee_code,
			full_name,
			role_title,
			employment_type,
			status,
			evaluation_status,
			mobility_stage,
			school_year,
			assigned_unit,
			phone,
			email,
			has_portfolio,
			institution_id,
			notes
	`,
		req.FullName,
		req.RoleTitle,
		req.EmploymentType,
		req.Status,
		req.EvaluationStatus,
		req.MobilityStage,
		req.SchoolYear,
		req.AssignedUnit,
		req.Phone,
		req.Email,
		req.HasPortfolio,
		req.Notes,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.EmploymentType,
		&item.Status,
		&item.EvaluationStatus,
		&item.MobilityStage,
		&item.SchoolYear,
		&item.AssignedUnit,
		&item.Phone,
		&item.Email,
		&item.HasPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "personnel_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_update_failed"})
		return
	}

	s.logAudit(r, "education.personnel.update", "personnel_record", item.ID, "Personnel record updated.", map[string]any{
		"employee_code":     item.EmployeeCode,
		"full_name":         item.FullName,
		"role_title":        item.RoleTitle,
		"employment_type":   item.EmploymentType,
		"status":            item.Status,
		"evaluation_status": item.EvaluationStatus,
		"mobility_stage":    item.MobilityStage,
		"school_year":       item.SchoolYear,
		"has_portfolio":     item.HasPortfolio,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePersonnelRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_personnel_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_personnel
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "personnel_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "personnel_record_not_found")
		return
	}

	s.logAudit(r, "education.personnel.delete", "personnel_record", recordID, "Personnel record deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateEvaluation(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreatePersonnelEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Status = strings.TrimSpace(req.Status)
	req.EvaluatorName = strings.TrimSpace(req.EvaluatorName)
	req.FinalizedOn = strings.TrimSpace(req.FinalizedOn)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.EmployeeCode == "" || req.FullName == "" || req.RoleTitle == "" || req.SchoolYear == "" || req.Status == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_fields"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_evaluation_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_update_failed"})
		return
	}
	if !validStatus || !validSchoolYear || req.Score < 0 || req.Score > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_fields"})
		return
	}
	if inStringSet(req.Status, "reviewed", "approved") && req.EvaluatorName == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_evaluator"})
		return
	}
	if req.Status == "approved" && req.FinalizedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_evaluation_finalized_on"})
		return
	}
	if req.FinalizedOn != "" {
		if _, err := time.Parse("2006-01-02", req.FinalizedOn); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_finalized_on"})
			return
		}
	}
	qualification := evaluationQualification(req.Score)

	var item PersonnelEvaluation
	err = s.pool.QueryRow(r.Context(), `
		update education_evaluations
		set
			employee_code = $1,
			full_name = $2,
			role_title = $3,
			school_year = $4,
			status = $5,
			score = $6,
			qualification = $7,
			evaluator_name = $8,
			finalized_on = nullif($9, '')::date,
			summary = $10,
			updated_at = now()
		where id = $11::uuid and institution_id = $12
		returning
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
			coalesce(to_char(finalized_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		req.EmployeeCode,
		req.FullName,
		req.RoleTitle,
		req.SchoolYear,
		req.Status,
		req.Score,
		qualification,
		req.EvaluatorName,
		req.FinalizedOn,
		req.Summary,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
		&item.EvaluationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.Status,
		&item.Score,
		&item.Qualification,
		&item.EvaluatorName,
		&item.FinalizedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "evaluation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_update_failed"})
		return
	}

	s.logAudit(r, "education.evaluations.update", "personnel_evaluation", item.ID, "Personnel evaluation updated.", map[string]any{
		"evaluation_code": item.EvaluationCode,
		"employee_code":   item.EmployeeCode,
		"full_name":       item.FullName,
		"status":          item.Status,
		"score":           item.Score,
	})
	if err := s.syncEvaluationDocumentToPersonnelFile(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_personnel_file_sync_failed"})
		return
	}
	if err := s.syncEvaluationPersonnelStateByID(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_personnel_state_sync_failed"})
		return
	}
	if err := s.syncEvaluationResultIssueEffects(r.Context(), item.ID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_result_issue_sync_failed"})
		return
	}

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteEvaluation(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_evaluation_id"})
		return
	}

	institutionID := s.institutionID(r)
	var evaluationCode string
	prefetchErr := s.pool.QueryRow(r.Context(), `
		select evaluation_code
		from education_evaluations
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(&evaluationCode)
	if prefetchErr != nil && !errors.Is(prefetchErr, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_delete_failed"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_evaluations
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "evaluation_not_found")
		return
	}
	if evaluationCode != "" {
		if _, err := s.pool.Exec(r.Context(), `
			delete from education_personnel_file_documents
			where institution_id = $1 and document_category = 'evaluare' and (file_reference = $2 or file_reference like $3)
		`, institutionID, evaluationCode, evaluationCode+"/%"); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "evaluation_personnel_file_cleanup_failed"})
			return
		}
	}

	s.logAudit(r, "education.evaluations.delete", "personnel_evaluation", recordID, "Personnel evaluation deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateDeclaration(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreatePersonnelDeclarationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.DeclarationType = strings.TrimSpace(req.DeclarationType)
	req.Status = strings.TrimSpace(req.Status)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.ValidUntil = strings.TrimSpace(req.ValidUntil)
	req.Summary = strings.TrimSpace(req.Summary)

	if req.EmployeeCode == "" || req.FullName == "" || req.DeclarationType == "" || req.Status == "" || req.SchoolYear == "" || req.SubmittedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_declaration_fields"})
		return
	}
	validType, err := s.taxonomyExists(r, "education_declaration_type", req.DeclarationType)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_update_failed"})
		return
	}
	validStatus, err := s.taxonomyExists(r, "education_declaration_status", req.Status)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_update_failed"})
		return
	}
	validSchoolYear, err := s.taxonomyExists(r, "school_year", req.SchoolYear)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_update_failed"})
		return
	}
	if !validType || !validStatus || !validSchoolYear {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.SubmittedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_submitted_on"})
		return
	}
	if req.ValidUntil != "" {
		if _, err := time.Parse("2006-01-02", req.ValidUntil); err != nil {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_valid_until"})
			return
		}
	}

	var item PersonnelDeclaration
	err = s.pool.QueryRow(r.Context(), `
		update education_declarations
		set
			employee_code = $1,
			full_name = $2,
			declaration_type = $3,
			status = $4,
			school_year = $5,
			submitted_on = $6,
			valid_until = nullif($7, '')::date,
			summary = $8,
			updated_at = now()
		where id = $9::uuid and institution_id = $10
		returning
			id::text,
			declaration_code,
			employee_code,
			full_name,
			declaration_type,
			status,
			school_year,
			to_char(submitted_on, 'YYYY-MM-DD'),
			coalesce(to_char(valid_until, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
	`,
		req.EmployeeCode,
		req.FullName,
		req.DeclarationType,
		req.Status,
		req.SchoolYear,
		req.SubmittedOn,
		req.ValidUntil,
		req.Summary,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
		&item.DeclarationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.DeclarationType,
		&item.Status,
		&item.SchoolYear,
		&item.SubmittedOn,
		&item.ValidUntil,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "declaration_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_update_failed"})
		return
	}

	s.logAudit(r, "education.declarations.update", "personnel_declaration", item.ID, "Personnel declaration updated.", map[string]any{
		"declaration_code": item.DeclarationCode,
		"employee_code":    item.EmployeeCode,
		"full_name":        item.FullName,
		"declaration_type": item.DeclarationType,
		"status":           item.Status,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteDeclaration(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_declaration_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_declarations
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "declaration_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "declaration_not_found")
		return
	}

	s.logAudit(r, "education.declarations.delete", "personnel_declaration", recordID, "Personnel declaration deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdatePortfolioRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreatePortfolioRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_payload"})
		return
	}

	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.OwnerRole = strings.TrimSpace(req.OwnerRole)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Status = strings.TrimSpace(req.Status)
	req.LastUpdatedOn = strings.TrimSpace(req.LastUpdatedOn)
	req.RetentionUntil = strings.TrimSpace(req.RetentionUntil)
	req.TransferStatus = strings.TrimSpace(req.TransferStatus)
	req.Custodian = strings.TrimSpace(req.Custodian)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.OwnerName == "" || req.OwnerRole == "" || req.SchoolYear == "" || req.Status == "" || req.LastUpdatedOn == "" || req.RetentionUntil == "" || req.TransferStatus == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_fields"})
		return
	}
	if !contains([]string{"draft", "submitted", "validated", "transferred", "archived"}, req.Status) ||
		!contains([]string{"none", "prepared", "sent", "received"}, req.TransferStatus) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_fields"})
		return
	}
	if req.SectionCount < 0 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_sections"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.LastUpdatedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_last_updated"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.RetentionUntil); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention"})
		return
	}

	var item PortfolioRecord
	err := s.pool.QueryRow(r.Context(), `
		update education_portfolios
		set
			owner_name = $1,
			owner_role = $2,
			school_year = $3,
			status = $4,
			section_count = $5,
			last_updated_on = $6,
			retention_until = $7,
			transfer_status = $8,
			authenticity_declared = $9,
			consent_captured = $10,
			custodian = $11,
			notes = $12,
			updated_at = now()
		where id = $13::uuid and institution_id = $14
		returning
			id::text,
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
			institution_id,
			notes
	`,
		req.OwnerName,
		req.OwnerRole,
		req.SchoolYear,
		req.Status,
		req.SectionCount,
		req.LastUpdatedOn,
		req.RetentionUntil,
		req.TransferStatus,
		req.AuthenticityDeclared,
		req.ConsentCaptured,
		req.Custodian,
		req.Notes,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_update_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.update", "portfolio_record", item.ID, "Portfolio record updated.", map[string]any{
		"portfolio_code":        item.PortfolioCode,
		"owner_name":            item.OwnerName,
		"owner_role":            item.OwnerRole,
		"school_year":           item.SchoolYear,
		"status":                item.Status,
		"transfer_status":       item.TransferStatus,
		"authenticity_declared": item.AuthenticityDeclared,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeletePortfolioRecord(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_portfolios
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "portfolio_not_found")
		return
	}

	s.logAudit(r, "education.portfolios.delete", "portfolio_record", recordID, "Portfolio record deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateMobilityCase(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateMobilityCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_payload"})
		return
	}

	req.EmployeeCode = strings.TrimSpace(req.EmployeeCode)
	req.FullName = strings.TrimSpace(req.FullName)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.RequestType = strings.TrimSpace(req.RequestType)
	req.Stage = strings.TrimSpace(req.Stage)
	req.Status = strings.TrimSpace(req.Status)
	req.SourceSchool = strings.TrimSpace(req.SourceSchool)
	req.DestinationSchool = strings.TrimSpace(req.DestinationSchool)
	req.SubmittedOn = strings.TrimSpace(req.SubmittedOn)
	req.ReviewedBy = strings.TrimSpace(req.ReviewedBy)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.EmployeeCode == "" || req.FullName == "" || req.SchoolYear == "" || req.RequestType == "" || req.Stage == "" || req.Status == "" || req.SubmittedOn == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_mobility_fields"})
		return
	}
	if !contains([]string{"transfer", "detasare", "pretransfer", "restrangere"}, req.RequestType) ||
		!contains([]string{"draft", "submitted", "review", "approved", "completed"}, req.Stage) ||
		!contains([]string{"open", "pending", "approved", "rejected", "completed"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_fields"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.SubmittedOn); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_date"})
		return
	}

	var item MobilityCase
	err := s.pool.QueryRow(r.Context(), `
		update education_mobility_cases
		set
			employee_code = $1,
			full_name = $2,
			school_year = $3,
			request_type = $4,
			stage = $5,
			status = $6,
			source_school = $7,
			destination_school = $8,
			submitted_on = $9,
			reviewed_by = $10,
			notes = $11
		where id = $12::uuid and institution_id = $13
		returning
			id::text,
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
			institution_id,
			notes
	`,
		req.EmployeeCode,
		req.FullName,
		req.SchoolYear,
		req.RequestType,
		req.Stage,
		req.Status,
		req.SourceSchool,
		req.DestinationSchool,
		req.SubmittedOn,
		req.ReviewedBy,
		req.Notes,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "mobility_case_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_update_failed"})
		return
	}

	s.logAudit(r, "education.mobility.update", "mobility_case", item.ID, "Mobility case updated.", map[string]any{
		"case_code":          item.CaseCode,
		"employee_code":      item.EmployeeCode,
		"full_name":          item.FullName,
		"school_year":        item.SchoolYear,
		"request_type":       item.RequestType,
		"stage":              item.Stage,
		"status":             item.Status,
		"submitted_on":       item.SubmittedOn,
		"destination_school": item.DestinationSchool,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMobilityCase(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_mobility_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_mobility_cases
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "mobility_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "mobility_case_not_found")
		return
	}

	s.logAudit(r, "education.mobility.delete", "mobility_case", recordID, "Mobility case deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) UpdateMeritGrant(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_id"})
		return
	}

	institutionID := s.institutionID(r)
	var req CreateMeritGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_payload"})
		return
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.RoleTitle = strings.TrimSpace(req.RoleTitle)
	req.SchoolYear = strings.TrimSpace(req.SchoolYear)
	req.Category = strings.TrimSpace(req.Category)
	req.Status = strings.TrimSpace(req.Status)
	req.CommitteeName = strings.TrimSpace(req.CommitteeName)
	req.DecisionDate = strings.TrimSpace(req.DecisionDate)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.FullName == "" || req.RoleTitle == "" || req.SchoolYear == "" || req.Category == "" || req.Status == "" || req.DecisionDate == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_merit_fields"})
		return
	}
	if !contains([]string{"predare", "management", "consiliere", "auxiliar"}, req.Category) ||
		!contains([]string{"draft", "submitted", "evaluated", "approved", "funded"}, req.Status) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_fields"})
		return
	}
	if req.Score < 0 || req.Score > 100 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_score"})
		return
	}
	if _, err := time.Parse("2006-01-02", req.DecisionDate); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_date"})
		return
	}

	var item MeritGrant
	err := s.pool.QueryRow(r.Context(), `
		update education_merit_grants
		set
			full_name = $1,
			role_title = $2,
			school_year = $3,
			category = $4,
			status = $5,
			score = $6,
			committee_name = $7,
			decision_date = $8,
			funded = $9,
			notes = $10
		where id = $11::uuid and institution_id = $12
		returning
			id::text,
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
			institution_id,
			notes
	`,
		req.FullName,
		req.RoleTitle,
		req.SchoolYear,
		req.Category,
		req.Status,
		req.Score,
		req.CommitteeName,
		req.DecisionDate,
		req.Funded,
		req.Notes,
		recordID,
		institutionID,
	).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "merit_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_update_failed"})
		return
	}

	s.logAudit(r, "education.gradatii.update", "merit_grant", item.ID, "Merit grant record updated.", map[string]any{
		"grant_code":    item.GrantCode,
		"full_name":     item.FullName,
		"role_title":    item.RoleTitle,
		"school_year":   item.SchoolYear,
		"category":      item.Category,
		"status":        item.Status,
		"score":         item.Score,
		"decision_date": item.DecisionDate,
		"funded":        item.Funded,
	})

	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeleteMeritGrant(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_merit_id"})
		return
	}

	commandTag, err := s.pool.Exec(r.Context(), `
		delete from education_merit_grants
		where id = $1::uuid and institution_id = $2
	`, recordID, s.institutionID(r))
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "merit_delete_failed"})
		return
	}
	if commandTag.RowsAffected() == 0 {
		writeEducationNotFound(w, "merit_record_not_found")
		return
	}

	s.logAudit(r, "education.gradatii.delete", "merit_grant", recordID, "Merit grant record deleted.", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) GovernanceMeetingDetail(w http.ResponseWriter, r *http.Request) {
	var item GovernanceMeeting
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			school_year,
			organism,
			title,
			meeting_type,
			status,
			quorum_required,
			participants_count,
			to_char(meeting_date, 'YYYY-MM-DD'),
			location,
			chairperson,
			secretary_name,
			institution_id,
			summary
		from education_meetings
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "meetingID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.MeetingType,
		&item.Status,
		&item.QuorumRequired,
		&item.ParticipantsCount,
		&item.MeetingDate,
		&item.Location,
		&item.Chairperson,
		&item.SecretaryName,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_meeting_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_meeting_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) GovernanceDecisionDetail(w http.ResponseWriter, r *http.Request) {
	var item GovernanceDecision
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			decision_code,
			school_year,
			organism,
			title,
			status,
			publication_status,
			to_char(decision_date, 'YYYY-MM-DD'),
			legal_basis,
			signed_by,
			institution_id,
			summary
		from education_decisions
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "decisionID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.DecisionCode,
		&item.SchoolYear,
		&item.Organism,
		&item.Title,
		&item.Status,
		&item.PublicationStatus,
		&item.DecisionDate,
		&item.LegalBasis,
		&item.SignedBy,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_decision_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_decision_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) ManagerialDossierDetail(w http.ResponseWriter, r *http.Request) {
	var item ManagerialDossier
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			dossier_code,
			school_year,
			dossier_type,
			title,
			status,
			owner_name,
			to_char(due_on, 'YYYY-MM-DD'),
			publication_required,
			institution_id,
			summary
		from education_managerial_dossiers
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.DossierCode,
		&item.SchoolYear,
		&item.DossierType,
		&item.Title,
		&item.Status,
		&item.OwnerName,
		&item.DueOn,
		&item.PublicationRequired,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) RegulationDetail(w http.ResponseWriter, r *http.Request) {
	var item RegulationRecord
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			regulation_code,
			school_year,
			regulation_type,
			title,
			status,
			approval_status,
			owner_name,
			to_char(review_due_on, 'YYYY-MM-DD'),
			coalesce(to_char(approved_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
		from education_regulations
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.RegulationCode,
		&item.SchoolYear,
		&item.RegulationType,
		&item.Title,
		&item.Status,
		&item.ApprovalStatus,
		&item.OwnerName,
		&item.ReviewDueOn,
		&item.ApprovedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_regulation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_regulation_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PersonnelRecordDetail(w http.ResponseWriter, r *http.Request) {
	var item PersonnelRecord
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			employee_code,
			full_name,
			role_title,
			employment_type,
			status,
			evaluation_status,
			mobility_stage,
			school_year,
			assigned_unit,
			phone,
			email,
			has_portfolio,
			institution_id,
			notes
		from education_personnel
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.EmploymentType,
		&item.Status,
		&item.EvaluationStatus,
		&item.MobilityStage,
		&item.SchoolYear,
		&item.AssignedUnit,
		&item.Phone,
		&item.Email,
		&item.HasPortfolio,
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) EvaluationDetail(w http.ResponseWriter, r *http.Request) {
	var item PersonnelEvaluation
	err := s.pool.QueryRow(r.Context(), `
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
			coalesce(to_char(finalized_on, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
		from education_evaluations
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.EvaluationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.RoleTitle,
		&item.SchoolYear,
		&item.Status,
		&item.Score,
		&item.Qualification,
		&item.EvaluatorName,
		&item.FinalizedOn,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_evaluation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_evaluation_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) DeclarationDetail(w http.ResponseWriter, r *http.Request) {
	var item PersonnelDeclaration
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			declaration_code,
			employee_code,
			full_name,
			declaration_type,
			status,
			school_year,
			to_char(submitted_on, 'YYYY-MM-DD'),
			coalesce(to_char(valid_until, 'YYYY-MM-DD'), ''),
			institution_id,
			summary
		from education_declarations
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
		&item.DeclarationCode,
		&item.EmployeeCode,
		&item.FullName,
		&item.DeclarationType,
		&item.Status,
		&item.SchoolYear,
		&item.SubmittedOn,
		&item.ValidUntil,
		&item.InstitutionID,
		&item.Summary,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_declaration_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_declaration_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) PortfolioRecordDetail(w http.ResponseWriter, r *http.Request) {
	var item PortfolioRecord
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
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
			institution_id,
			notes
		from education_portfolios
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) MobilityCaseDetail(w http.ResponseWriter, r *http.Request) {
	var item MobilityCase
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
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
			institution_id,
			notes
		from education_mobility_cases
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_mobility_case_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_mobility_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func (s *Service) MeritGrantDetail(w http.ResponseWriter, r *http.Request) {
	var item MeritGrant
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
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
			institution_id,
			notes
		from education_merit_grants
		where id = $1::uuid and institution_id = $2
	`, chi.URLParam(r, "recordID"), s.institutionID(r)).Scan(
		&item.ID,
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
		&item.InstitutionID,
		&item.Notes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_merit_grant_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_merit_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

func buildMeetingFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"em.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["title"]; value != "" {
		addContains("em.title", value)
	}
	if value := filters["school_year"]; value != "" {
		addEqual("em.school_year", value)
	}
	if value := filters["organism"]; value != "" {
		addEqual("em.organism", value)
	}
	if value := filters["meeting_type"]; value != "" {
		addEqual("em.meeting_type", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("em.status", value)
	}
	if value := filters["meeting_date"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("em.meeting_date = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func (s *Service) taxonomyCodes(r *http.Request, domain string) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), `
		select code
		from education_taxonomies
		where domain = $1 and active = true
		order by sort_order, code
	`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Service) taxonomyExists(r *http.Request, domain string, code string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(r.Context(), `
		select exists(
			select 1
			from education_taxonomies
			where domain = $1 and code = $2 and active = true
		)
	`, domain, code).Scan(&exists)
	return exists, err
}

func (s *Service) loadDistinctValues(r *http.Request, sql string, args ...any) ([]string, error) {
	rows, err := s.pool.Query(r.Context(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func buildDecisionFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"ed.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["decision_code"]); value != "" {
		addContains("ed.decision_code", value)
	}
	if value := strings.TrimSpace(filters["title"]); value != "" {
		addContains("ed.title", value)
	}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		addEqual("ed.school_year", value)
	}
	if value := strings.TrimSpace(filters["organism"]); value != "" {
		addEqual("ed.organism", value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" {
		addEqual("ed.status", value)
	}
	if value := strings.TrimSpace(filters["publication_status"]); value != "" {
		addEqual("ed.publication_status", value)
	}
	if value := strings.TrimSpace(filters["decision_date"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("ed.decision_date = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["signed_by"]); value != "" {
		addContains("ed.signed_by", value)
	}

	return "where " + strings.Join(clauses, " and "), args
}

func decisionSortColumn(field string) string {
	switch field {
	case "decision_code":
		return "ed.decision_code"
	case "title":
		return "ed.title"
	case "school_year":
		return "ed.school_year"
	case "organism":
		return "ed.organism"
	case "status":
		return "ed.status"
	case "publication_status":
		return "ed.publication_status"
	case "decision_date":
		return "ed.decision_date"
	default:
		return "ed.decision_date"
	}
}

func buildManagerialFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"emd.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["dossier_code"]); value != "" {
		addContains("emd.dossier_code", value)
	}
	if value := strings.TrimSpace(filters["title"]); value != "" {
		addContains("emd.title", value)
	}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		addEqual("emd.school_year", value)
	}
	if value := strings.TrimSpace(filters["dossier_type"]); value != "" {
		addEqual("emd.dossier_type", value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" {
		addEqual("emd.status", value)
	}
	if value := strings.TrimSpace(filters["due_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("emd.due_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["publication_required"]); value != "" {
		addEqual("emd.publication_required::text", value)
	}
	if value := strings.TrimSpace(filters["owner_name"]); value != "" {
		addContains("emd.owner_name", value)
	}

	return "where " + strings.Join(clauses, " and "), args
}

func managerialSortColumn(field string) string {
	switch field {
	case "dossier_code":
		return "emd.dossier_code"
	case "title":
		return "emd.title"
	case "school_year":
		return "emd.school_year"
	case "dossier_type":
		return "emd.dossier_type"
	case "status":
		return "emd.status"
	case "due_on":
		return "emd.due_on"
	default:
		return "emd.due_on"
	}
}

func meetingSortColumn(field string) string {
	switch field {
	case "title":
		return "em.title"
	case "school_year":
		return "em.school_year"
	case "organism":
		return "em.organism"
	case "meeting_type":
		return "em.meeting_type"
	case "status":
		return "em.status"
	case "meeting_date":
		return "em.meeting_date"
	default:
		return "em.meeting_date"
	}
}

func buildPersonnelFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"ep.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["employee_code"]; value != "" {
		addContains("ep.employee_code", value)
	}
	if value := filters["full_name"]; value != "" {
		addContains("ep.full_name", value)
	}
	if value := filters["employment_type"]; value != "" {
		addEqual("ep.employment_type", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("ep.status", value)
	}
	if value := filters["evaluation_status"]; value != "" {
		addEqual("ep.evaluation_status", value)
	}
	if value := filters["mobility_stage"]; value != "" {
		addEqual("ep.mobility_stage", value)
	}
	if value := filters["school_year"]; value != "" {
		addEqual("ep.school_year", value)
	}
	if value := filters["has_portfolio"]; value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("ep.has_portfolio = $%d", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func personnelSortColumn(field string) string {
	switch field {
	case "employee_code":
		return "ep.employee_code"
	case "full_name":
		return "ep.full_name"
	case "role_title":
		return "ep.role_title"
	case "employment_type":
		return "ep.employment_type"
	case "status":
		return "ep.status"
	case "evaluation_status":
		return "ep.evaluation_status"
	case "mobility_stage":
		return "ep.mobility_stage"
	case "school_year":
		return "ep.school_year"
	default:
		return "ep.full_name"
	}
}

func buildRegulationFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"er.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["regulation_code"]); value != "" {
		addContains("er.regulation_code", value)
	}
	if value := strings.TrimSpace(filters["title"]); value != "" {
		addContains("er.title", value)
	}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		addEqual("er.school_year", value)
	}
	if value := strings.TrimSpace(filters["regulation_type"]); value != "" {
		addEqual("er.regulation_type", value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" {
		addEqual("er.status", value)
	}
	if value := strings.TrimSpace(filters["approval_status"]); value != "" {
		addEqual("er.approval_status", value)
	}
	if value := strings.TrimSpace(filters["review_due_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("er.review_due_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["owner_name"]); value != "" {
		addContains("er.owner_name", value)
	}

	return "where " + strings.Join(clauses, " and "), args
}

func regulationSortColumn(field string) string {
	switch field {
	case "regulation_code":
		return "er.regulation_code"
	case "title":
		return "er.title"
	case "school_year":
		return "er.school_year"
	case "regulation_type":
		return "er.regulation_type"
	case "status":
		return "er.status"
	case "approval_status":
		return "er.approval_status"
	case "review_due_on":
		return "er.review_due_on"
	default:
		return "er.review_due_on"
	}
}

func buildEvaluationFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"ee.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["evaluation_code"]); value != "" {
		addContains("ee.evaluation_code", value)
	}
	if value := strings.TrimSpace(filters["employee_code"]); value != "" {
		addContains("ee.employee_code", value)
	}
	if value := strings.TrimSpace(filters["full_name"]); value != "" {
		addContains("ee.full_name", value)
	}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		addEqual("ee.school_year", value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" {
		addEqual("ee.status", value)
	}
	if value := strings.TrimSpace(filters["qualification"]); value != "" {
		addEqual("ee.qualification", value)
	}
	if value := strings.TrimSpace(filters["finalized_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("ee.finalized_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["evaluator_name"]); value != "" {
		addContains("ee.evaluator_name", value)
	}

	return "where " + strings.Join(clauses, " and "), args
}

func evaluationSortColumn(field string) string {
	switch field {
	case "evaluation_code":
		return "ee.evaluation_code"
	case "employee_code":
		return "ee.employee_code"
	case "full_name":
		return "ee.full_name"
	case "school_year":
		return "ee.school_year"
	case "status":
		return "ee.status"
	case "score":
		return "ee.score"
	case "qualification":
		return "ee.qualification"
	case "finalized_on":
		return "ee.finalized_on"
	default:
		return "ee.full_name"
	}
}

func buildDeclarationFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"ed.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := strings.TrimSpace(filters["declaration_code"]); value != "" {
		addContains("ed.declaration_code", value)
	}
	if value := strings.TrimSpace(filters["employee_code"]); value != "" {
		addContains("ed.employee_code", value)
	}
	if value := strings.TrimSpace(filters["full_name"]); value != "" {
		addContains("ed.full_name", value)
	}
	if value := strings.TrimSpace(filters["declaration_type"]); value != "" {
		addEqual("ed.declaration_type", value)
	}
	if value := strings.TrimSpace(filters["status"]); value != "" {
		addEqual("ed.status", value)
	}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		addEqual("ed.school_year", value)
	}
	if value := strings.TrimSpace(filters["submitted_on"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("ed.submitted_on = $%d::date", len(args)))
	}
	if value := strings.TrimSpace(filters["valid_until"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("ed.valid_until = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func declarationSortColumn(field string) string {
	switch field {
	case "declaration_code":
		return "ed.declaration_code"
	case "employee_code":
		return "ed.employee_code"
	case "full_name":
		return "ed.full_name"
	case "declaration_type":
		return "ed.declaration_type"
	case "status":
		return "ed.status"
	case "school_year":
		return "ed.school_year"
	case "submitted_on":
		return "ed.submitted_on"
	case "valid_until":
		return "ed.valid_until"
	default:
		return "ed.submitted_on"
	}
}

func buildPortfolioFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"epf.institution_id = $1"}
	args := []any{institutionID}

	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["portfolio_code"]; value != "" {
		addContains("epf.portfolio_code", value)
	}
	if value := filters["owner_name"]; value != "" {
		addContains("epf.owner_name", value)
	}
	if value := filters["school_year"]; value != "" {
		addEqual("epf.school_year", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("epf.status", value)
	}
	if value := filters["transfer_status"]; value != "" {
		addEqual("epf.transfer_status", value)
	}
	if value := filters["authenticity_declared"]; value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("epf.authenticity_declared = $%d", len(args)))
	}
	if value := filters["consent_captured"]; value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("epf.consent_captured = $%d", len(args)))
	}
	if value := filters["retention_until"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("epf.retention_until = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func portfolioSortColumn(field string) string {
	switch field {
	case "portfolio_code":
		return "epf.portfolio_code"
	case "owner_name":
		return "epf.owner_name"
	case "school_year":
		return "epf.school_year"
	case "status":
		return "epf.status"
	case "section_count":
		return "epf.section_count"
	case "retention_until":
		return "epf.retention_until"
	case "transfer_status":
		return "epf.transfer_status"
	default:
		return "epf.last_updated_on"
	}
}

func buildMobilityFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"emc.institution_id = $1"}
	args := []any{institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["case_code"]; value != "" {
		addContains("emc.case_code", value)
	}
	if value := filters["employee_code"]; value != "" {
		addContains("emc.employee_code", value)
	}
	if value := filters["full_name"]; value != "" {
		addContains("emc.full_name", value)
	}
	if value := filters["school_year"]; value != "" {
		addEqual("emc.school_year", value)
	}
	if value := filters["request_type"]; value != "" {
		addEqual("emc.request_type", value)
	}
	if value := filters["stage"]; value != "" {
		addEqual("emc.stage", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("emc.status", value)
	}
	if value := filters["submitted_on"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("emc.submitted_on = $%d::date", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func mobilitySortColumn(field string) string {
	switch field {
	case "case_code":
		return "emc.case_code"
	case "employee_code":
		return "emc.employee_code"
	case "full_name":
		return "emc.full_name"
	case "school_year":
		return "emc.school_year"
	case "request_type":
		return "emc.request_type"
	case "stage":
		return "emc.stage"
	case "status":
		return "emc.status"
	case "submitted_on":
		return "emc.submitted_on"
	default:
		return "emc.submitted_on"
	}
}

func buildMeritFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"emg.institution_id = $1"}
	args := []any{institutionID}
	addContains := func(column string, value string) {
		args = append(args, "%"+strings.ToLower(value)+"%")
		clauses = append(clauses, fmt.Sprintf("lower(%s) like $%d", column, len(args)))
	}
	addEqual := func(column string, value string) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if value := filters["grant_code"]; value != "" {
		addContains("emg.grant_code", value)
	}
	if value := filters["full_name"]; value != "" {
		addContains("emg.full_name", value)
	}
	if value := filters["school_year"]; value != "" {
		addEqual("emg.school_year", value)
	}
	if value := filters["category"]; value != "" {
		addEqual("emg.category", value)
	}
	if value := filters["status"]; value != "" {
		addEqual("emg.status", value)
	}
	if value := filters["decision_date"]; value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("emg.decision_date = $%d::date", len(args)))
	}
	if value := filters["funded"]; value != "" {
		args = append(args, value == "true")
		clauses = append(clauses, fmt.Sprintf("emg.funded = $%d", len(args)))
	}

	return "where " + strings.Join(clauses, " and "), args
}

func meritSortColumn(field string) string {
	switch field {
	case "grant_code":
		return "emg.grant_code"
	case "full_name":
		return "emg.full_name"
	case "school_year":
		return "emg.school_year"
	case "category":
		return "emg.category"
	case "status":
		return "emg.status"
	case "score":
		return "emg.score"
	case "decision_date":
		return "emg.decision_date"
	default:
		return "emg.decision_date"
	}
}

func (s *Service) RequirementCatalog(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"domain":                {},
		"priority":              {},
		"implementation_status": {},
		"requirement_type":      {},
	}, []string{"domain", "implementation_status", "requirement_type", "source_ref"})
	if query.Sort == "" {
		query.Sort = "priority"
	}

	where := []string{"1=1"}
	args := []any{}
	for key, value := range query.Filters {
		args = append(args, "%"+strings.ToLower(value)+"%")
		switch key {
		case "domain":
			where = append(where, fmt.Sprintf("lower(domain) like $%d", len(args)))
		case "implementation_status":
			where = append(where, fmt.Sprintf("lower(implementation_status) like $%d", len(args)))
		case "requirement_type":
			where = append(where, fmt.Sprintf("lower(requirement_type) like $%d", len(args)))
		case "source_ref":
			where = append(where, fmt.Sprintf("lower(source_ref) like $%d", len(args)))
		}
	}
	whereSQL := strings.Join(where, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_requirement_catalog where "+whereSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_requirements_count_failed"})
		return
	}

	orderBy := map[string]string{
		"domain":                "domain",
		"priority":              "priority",
		"implementation_status": "implementation_status",
		"requirement_type":      "requirement_type",
	}[query.Sort]
	if orderBy == "" {
		orderBy = "priority"
	}
	direction := "asc"
	if query.Direction == "desc" {
		direction = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select id::text, domain, code, title_ro, title_en, source_ref, requirement_type, implementation_status, priority, notes
		from education_requirement_catalog
		where `+whereSQL+`
		order by `+orderBy+` `+direction+`, domain, code
		limit $`+fmt.Sprint(len(args)-1)+` offset $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_requirements_failed"})
		return
	}
	defer rows.Close()

	items := []EducationRequirement{}
	for rows.Next() {
		var item EducationRequirement
		if err := rows.Scan(&item.ID, &item.Domain, &item.Code, &item.TitleRO, &item.TitleEN, &item.SourceRef, &item.RequirementType, &item.ImplementationStatus, &item.Priority, &item.Notes); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_requirements_scan_failed"})
			return
		}
		items = append(items, item)
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioSections(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"section_code":   {},
		"component_code": {},
		"sort_order":     {},
	}, []string{"section_code", "component_code", "label"})
	if query.Sort == "" {
		query.Sort = "sort_order"
	}

	where := []string{"active = true"}
	args := []any{}
	for key, value := range query.Filters {
		args = append(args, "%"+strings.ToLower(value)+"%")
		switch key {
		case "section_code":
			where = append(where, fmt.Sprintf("lower(section_code) like $%d", len(args)))
		case "component_code":
			where = append(where, fmt.Sprintf("lower(component_code) like $%d", len(args)))
		case "label":
			where = append(where, fmt.Sprintf("(lower(label_ro) like $%d or lower(label_en) like $%d)", len(args), len(args)))
		}
	}
	whereSQL := strings.Join(where, " and ")

	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_sections where "+whereSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_sections_count_failed"})
		return
	}

	orderBy := map[string]string{
		"section_code":   "section_code",
		"component_code": "component_code",
		"sort_order":     "sort_order",
	}[query.Sort]
	if orderBy == "" {
		orderBy = "sort_order"
	}
	direction := "asc"
	if query.Direction == "desc" {
		direction = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), `
		select id::text, section_code, component_code, label_ro, label_en, example_documents, required, sensitive_data, retention_rule, sort_order, active
		from education_portfolio_sections
		where `+whereSQL+`
		order by `+orderBy+` `+direction+`, section_code, component_code
		limit $`+fmt.Sprint(len(args)-1)+` offset $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_sections_failed"})
		return
	}
	defer rows.Close()

	items := []PortfolioSection{}
	for rows.Next() {
		var item PortfolioSection
		if err := rows.Scan(&item.ID, &item.SectionCode, &item.ComponentCode, &item.LabelRO, &item.LabelEN, &item.ExampleDocuments, &item.Required, &item.SensitiveData, &item.RetentionRule, &item.SortOrder, &item.Active); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_sections_scan_failed"})
			return
		}
		items = append(items, item)
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func contains(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
