package education

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type GovernanceBodyRecord struct {
	ID                 string `json:"id"`
	SchoolYear         string `json:"school_year"`
	Organism           string `json:"organism"`
	ActiveMembers      int    `json:"active_members"`
	VotingMembers      int    `json:"voting_members"`
	ChairpersonCovered bool   `json:"chairperson_covered"`
	SecretaryCovered   bool   `json:"secretary_covered"`
	ExpiredMandates    int    `json:"expired_mandates"`
	TotalMeetings      int    `json:"total_meetings"`
	ScheduledMeetings  int    `json:"scheduled_meetings"`
	HeldMeetings       int    `json:"held_meetings"`
	PublishedMeetings  int    `json:"published_meetings"`
	LatestMeetingOn    string `json:"latest_meeting_on"`
	ReadinessStatus    string `json:"readiness_status"`
	InstitutionID      string `json:"institution_id"`
}

type GovernanceBodyCompletenessSummary struct {
	Body       GovernanceBodyRecord                      `json:"body"`
	Membership GovernanceBodyCompletenessMembershipBlock `json:"membership"`
	Meetings   GovernanceBodyCompletenessMeetingBlock    `json:"meetings"`
	Readiness  GovernanceBodyCompletenessReadiness       `json:"readiness"`
}

type GovernanceBodyCompletenessMembershipBlock struct {
	ActiveMembers      int      `json:"active_members"`
	VotingMembers      int      `json:"voting_members"`
	ChairpersonCovered bool     `json:"chairperson_covered"`
	SecretaryCovered   bool     `json:"secretary_covered"`
	ExpiredMandates    int      `json:"expired_mandates"`
	MemberNames        []string `json:"member_names"`
}

type GovernanceBodyCompletenessMeetingBlock struct {
	TotalMeetings     int    `json:"total_meetings"`
	ScheduledMeetings int    `json:"scheduled_meetings"`
	HeldMeetings      int    `json:"held_meetings"`
	PublishedMeetings int    `json:"published_meetings"`
	LastMeetingTitle  string `json:"last_meeting_title"`
	LastMeetingOn     string `json:"last_meeting_on"`
}

type GovernanceBodyCompletenessReadiness struct {
	ReadyForOperation bool     `json:"ready_for_operation"`
	ReadyForMeetings  bool     `json:"ready_for_meetings"`
	Blockers          []string `json:"blockers"`
}

func (s *Service) GovernanceBodies(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(
		r.URL.Query(),
		map[string]struct{}{
			"school_year": {},
			"organism":    {},
		},
		[]string{"school_year", "organism", "active_members", "voting_members", "held_meetings", "latest_meeting_on"},
	)
	if query.Sort == "" {
		query.Sort = "school_year"
	}

	whereClause, args := buildGovernanceBodyFilters(query.Filters, s.institutionID(r))

	var total int
	countSQL := `
		select count(*)
		from (
			select gm.school_year, gm.organism
			from education_governance_memberships gm
			` + whereClause + `
			group by gm.school_year, gm.organism
		) bodies
	`
	if err := s.pool.QueryRow(r.Context(), countSQL, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_bodies_failed"})
		return
	}

	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		with body_base as (
			select
				gm.school_year,
				gm.organism,
				count(*) filter (where gm.status = 'activ') as active_members,
				count(*) filter (where gm.status = 'activ' and gm.voting_right) as voting_members,
				count(*) filter (where gm.status = 'activ' and lower(gm.role_name) like '%%presed%%') as chairperson_matches,
				count(*) filter (where gm.status = 'activ' and lower(gm.role_name) like '%%secretar%%') as secretary_matches,
				count(*) filter (where gm.status = 'activ' and gm.mandate_to < current_date) as expired_mandates
			from education_governance_memberships gm
			%s
			group by gm.school_year, gm.organism
		),
		meeting_stats as (
			select
				em.school_year,
				em.organism,
				count(*) as total_meetings,
				count(*) filter (where em.status = 'scheduled') as scheduled_meetings,
				count(*) filter (where em.status = 'held') as held_meetings,
				count(*) filter (where em.status = 'published') as published_meetings,
				max(em.meeting_date) as latest_meeting_on
			from education_meetings em
			where em.institution_id = $1
			group by em.school_year, em.organism
		)
		select
			body_base.school_year || '__' || body_base.organism as id,
			body_base.school_year,
			body_base.organism,
			body_base.active_members,
			body_base.voting_members,
			body_base.chairperson_matches > 0 as chairperson_covered,
			body_base.secretary_matches > 0 as secretary_covered,
			body_base.expired_mandates,
			coalesce(meeting_stats.total_meetings, 0) as total_meetings,
			coalesce(meeting_stats.scheduled_meetings, 0) as scheduled_meetings,
			coalesce(meeting_stats.held_meetings, 0) as held_meetings,
			coalesce(meeting_stats.published_meetings, 0) as published_meetings,
			coalesce(to_char(meeting_stats.latest_meeting_on, 'YYYY-MM-DD'), '') as latest_meeting_on,
			case
				when body_base.active_members = 0 or body_base.voting_members = 0 or body_base.chairperson_matches = 0 or body_base.secretary_matches = 0 then 'critic'
				when body_base.expired_mandates > 0 or coalesce(meeting_stats.total_meetings, 0) = 0 then 'partial'
				else 'complet'
			end as readiness_status,
			$1 as institution_id
		from body_base
		left join meeting_stats
			on meeting_stats.school_year = body_base.school_year
			and meeting_stats.organism = body_base.organism
		order by %s %s, body_base.organism, body_base.school_year desc
		limit $%d offset $%d
	`, whereClause, governanceBodySortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_bodies_failed"})
		return
	}
	defer rows.Close()

	items := make([]GovernanceBodyRecord, 0, query.PageSize)
	for rows.Next() {
		var item GovernanceBodyRecord
		if err := rows.Scan(
			&item.ID,
			&item.SchoolYear,
			&item.Organism,
			&item.ActiveMembers,
			&item.VotingMembers,
			&item.ChairpersonCovered,
			&item.SecretaryCovered,
			&item.ExpiredMandates,
			&item.TotalMeetings,
			&item.ScheduledMeetings,
			&item.HeldMeetings,
			&item.PublishedMeetings,
			&item.LatestMeetingOn,
			&item.ReadinessStatus,
			&item.InstitutionID,
		); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_bodies_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_bodies_scan_failed"})
		return
	}

	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) GovernanceBodyDetail(w http.ResponseWriter, r *http.Request) {
	body, err := s.loadGovernanceBodyRecord(r, strings.TrimSpace(chi.URLParam(r, "bodyID")))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_governance_body_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_body_detail_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, body)
}

func (s *Service) GovernanceBodyCompletenessSummary(w http.ResponseWriter, r *http.Request) {
	body, err := s.loadGovernanceBodyRecord(r, strings.TrimSpace(chi.URLParam(r, "bodyID")))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "education_governance_body_not_found")
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_body_summary_failed"})
		return
	}

	summary := GovernanceBodyCompletenessSummary{Body: body}
	summary.Membership = GovernanceBodyCompletenessMembershipBlock{
		ActiveMembers:      body.ActiveMembers,
		VotingMembers:      body.VotingMembers,
		ChairpersonCovered: body.ChairpersonCovered,
		SecretaryCovered:   body.SecretaryCovered,
		ExpiredMandates:    body.ExpiredMandates,
		MemberNames:        []string{},
	}
	summary.Meetings = GovernanceBodyCompletenessMeetingBlock{
		TotalMeetings:     body.TotalMeetings,
		ScheduledMeetings: body.ScheduledMeetings,
		HeldMeetings:      body.HeldMeetings,
		PublishedMeetings: body.PublishedMeetings,
		LastMeetingOn:     body.LatestMeetingOn,
	}

	rows, err := s.pool.Query(r.Context(), `
		select full_name
		from education_governance_memberships
		where institution_id = $1 and school_year = $2 and organism = $3 and status = 'activ'
		order by full_name
	`, s.institutionID(r), body.SchoolYear, body.Organism)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_body_summary_failed"})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var fullName string
		if err := rows.Scan(&fullName); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_body_summary_failed"})
			return
		}
		summary.Membership.MemberNames = append(summary.Membership.MemberNames, fullName)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_body_summary_failed"})
		return
	}

	_ = s.pool.QueryRow(r.Context(), `
		select
			coalesce(title, ''),
			coalesce(to_char(meeting_date, 'YYYY-MM-DD'), '')
		from education_meetings
		where institution_id = $1 and school_year = $2 and organism = $3
		order by meeting_date desc, created_at desc
		limit 1
	`, s.institutionID(r), body.SchoolYear, body.Organism).Scan(&summary.Meetings.LastMeetingTitle, &summary.Meetings.LastMeetingOn)

	blockers := make([]string, 0)
	if body.ActiveMembers == 0 {
		blockers = append(blockers, "no_active_members")
	}
	if body.VotingMembers == 0 {
		blockers = append(blockers, "no_voting_members")
	}
	if !body.ChairpersonCovered {
		blockers = append(blockers, "chairperson_missing")
	}
	if !body.SecretaryCovered {
		blockers = append(blockers, "secretary_missing")
	}
	if body.ExpiredMandates > 0 {
		blockers = append(blockers, "expired_mandates")
	}
	if body.TotalMeetings == 0 {
		blockers = append(blockers, "no_meeting_recorded")
	}
	if body.HeldMeetings == 0 {
		blockers = append(blockers, "no_held_meeting")
	}

	summary.Readiness = GovernanceBodyCompletenessReadiness{
		ReadyForOperation: body.ActiveMembers > 0 && body.VotingMembers > 0 && body.ChairpersonCovered && body.SecretaryCovered && body.ExpiredMandates == 0,
		ReadyForMeetings:  body.ActiveMembers > 0 && body.VotingMembers > 0 && body.TotalMeetings > 0,
		Blockers:          blockers,
	}

	httpx.JSON(w, http.StatusOK, summary)
}

func (s *Service) loadGovernanceBodyRecord(r *http.Request, bodyID string) (GovernanceBodyRecord, error) {
	schoolYear, organism := parseGovernanceBodyID(bodyID)
	if schoolYear == "" || organism == "" {
		return GovernanceBodyRecord{}, pgx.ErrNoRows
	}

	var item GovernanceBodyRecord
	err := s.pool.QueryRow(r.Context(), `
		with body_base as (
			select
				gm.school_year,
				gm.organism,
				count(*) filter (where gm.status = 'activ') as active_members,
				count(*) filter (where gm.status = 'activ' and gm.voting_right) as voting_members,
				count(*) filter (where gm.status = 'activ' and lower(gm.role_name) like '%presed%') as chairperson_matches,
				count(*) filter (where gm.status = 'activ' and lower(gm.role_name) like '%secretar%') as secretary_matches,
				count(*) filter (where gm.status = 'activ' and gm.mandate_to < current_date) as expired_mandates
			from education_governance_memberships gm
			where gm.institution_id = $1 and gm.school_year = $2 and gm.organism = $3
			group by gm.school_year, gm.organism
		),
		meeting_stats as (
			select
				em.school_year,
				em.organism,
				count(*) as total_meetings,
				count(*) filter (where em.status = 'scheduled') as scheduled_meetings,
				count(*) filter (where em.status = 'held') as held_meetings,
				count(*) filter (where em.status = 'published') as published_meetings,
				max(em.meeting_date) as latest_meeting_on
			from education_meetings em
			where em.institution_id = $1 and em.school_year = $2 and em.organism = $3
			group by em.school_year, em.organism
		)
		select
			body_base.school_year || '__' || body_base.organism as id,
			body_base.school_year,
			body_base.organism,
			body_base.active_members,
			body_base.voting_members,
			body_base.chairperson_matches > 0 as chairperson_covered,
			body_base.secretary_matches > 0 as secretary_covered,
			body_base.expired_mandates,
			coalesce(meeting_stats.total_meetings, 0) as total_meetings,
			coalesce(meeting_stats.scheduled_meetings, 0) as scheduled_meetings,
			coalesce(meeting_stats.held_meetings, 0) as held_meetings,
			coalesce(meeting_stats.published_meetings, 0) as published_meetings,
			coalesce(to_char(meeting_stats.latest_meeting_on, 'YYYY-MM-DD'), '') as latest_meeting_on,
			case
				when body_base.active_members = 0 or body_base.voting_members = 0 or body_base.chairperson_matches = 0 or body_base.secretary_matches = 0 then 'critic'
				when body_base.expired_mandates > 0 or coalesce(meeting_stats.total_meetings, 0) = 0 then 'partial'
				else 'complet'
			end as readiness_status,
			$1 as institution_id
		from body_base
		left join meeting_stats
			on meeting_stats.school_year = body_base.school_year
			and meeting_stats.organism = body_base.organism
	`, s.institutionID(r), schoolYear, organism).Scan(
		&item.ID,
		&item.SchoolYear,
		&item.Organism,
		&item.ActiveMembers,
		&item.VotingMembers,
		&item.ChairpersonCovered,
		&item.SecretaryCovered,
		&item.ExpiredMandates,
		&item.TotalMeetings,
		&item.ScheduledMeetings,
		&item.HeldMeetings,
		&item.PublishedMeetings,
		&item.LatestMeetingOn,
		&item.ReadinessStatus,
		&item.InstitutionID,
	)
	return item, err
}

func parseGovernanceBodyID(bodyID string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(bodyID), "__", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func buildGovernanceBodyFilters(filters map[string]string, institutionID string) (string, []any) {
	clauses := []string{"where gm.institution_id = $1"}
	args := []any{institutionID}
	if value := strings.TrimSpace(filters["school_year"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gm.school_year = $%d", len(args)))
	}
	if value := strings.TrimSpace(filters["organism"]); value != "" {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("gm.organism = $%d", len(args)))
	}
	return " " + strings.Join(clauses, " and "), args
}

func governanceBodySortColumn(value string) string {
	switch value {
	case "organism":
		return "body_base.organism"
	case "active_members":
		return "body_base.active_members"
	case "voting_members":
		return "body_base.voting_members"
	case "held_meetings":
		return "coalesce(meeting_stats.held_meetings, 0)"
	case "latest_meeting_on":
		return "meeting_stats.latest_meeting_on"
	case "school_year":
		fallthrough
	default:
		return "body_base.school_year"
	}
}
