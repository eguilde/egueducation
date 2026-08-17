package education

import (
	"errors"
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/jackc/pgx/v5"
)

var errGovernanceMeetingNotFound = errors.New("education governance meeting not found")

func (s *Service) RequireInstitutionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r)) == "" {
			httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_institution_context_required"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

type governanceMeetingAccessContext struct {
	ID                string
	SchoolYear        string
	Organism          string
	Chairperson       string
	SecretaryName     string
	ChairpersonUserID string
	SecretaryUserID   string
	Status            string
}

type governanceMeetingActorRule struct {
	RequireVotingRight    bool
	AllowMeetingChair     bool
	AllowMeetingSecretary bool
	MembershipRoleHints   []string
}

func (s *Service) ensureGovernanceMeetingCloseAccess(w http.ResponseWriter, r *http.Request, meetingID string) bool {
	return s.enforceGovernanceMeetingAction(
		w,
		r,
		meetingID,
		"education.governance.meeting.close",
		"education_governance_meeting_close_forbidden",
		governanceMeetingActorRule{
			AllowMeetingChair:     true,
			AllowMeetingSecretary: true,
			MembershipRoleHints:   []string{"presedinte", "director", "coordonator", "secretar"},
		},
	)
}

func (s *Service) ensureGovernanceMeetingVoteAccess(w http.ResponseWriter, r *http.Request, meetingID string) bool {
	return s.enforceGovernanceMeetingAction(
		w,
		r,
		meetingID,
		"education.governance.meeting.vote",
		"education_governance_meeting_vote_forbidden",
		governanceMeetingActorRule{
			RequireVotingRight: true,
		},
	)
}

func (s *Service) ensureGovernancePublicationAccess(w http.ResponseWriter, r *http.Request, meetingID string, permission string, denialCode string) bool {
	return s.enforceGovernanceMeetingAction(
		w,
		r,
		meetingID,
		permission,
		denialCode,
		governanceMeetingActorRule{
			AllowMeetingChair:     true,
			AllowMeetingSecretary: true,
			MembershipRoleHints:   []string{"presedinte", "director", "coordonator", "secretar"},
		},
	)
}

func (s *Service) enforceGovernanceMeetingAction(
	w http.ResponseWriter,
	r *http.Request,
	meetingID string,
	permission string,
	denialCode string,
	rule governanceMeetingActorRule,
) bool {
	allowed, err := s.authorizeGovernanceMeetingAction(r, meetingID, permission, rule)
	if err != nil {
		switch {
		case errors.Is(err, errGovernanceMeetingNotFound):
			writeEducationNotFound(w, "education_meeting_not_found")
		default:
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_governance_permission_check_failed"})
		}
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{
			"code":       denialCode,
			"permission": permission,
		})
		return false
	}
	return true
}

func (s *Service) authorizeGovernanceMeetingAction(r *http.Request, meetingID string, permission string, rule governanceMeetingActorRule) (bool, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	return s.authorizeGovernanceMeetingActionForSubject(r, subject, meetingID, permission, rule)
}

// authorizeGovernanceMeetingActionForSubject contains the complete contextual
// decision path behind the HTTP/auth adapter above. Keeping the subject explicit
// lets the PostgreSQL integration test exercise the exact authorizer with a
// real SessionPool tenant context without manufacturing private auth contexts.
func (s *Service) authorizeGovernanceMeetingActionForSubject(r *http.Request, subject string, meetingID string, permission string, rule governanceMeetingActorRule) (bool, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return false, nil
	}

	allowed, err := s.currentSubjectHasPermission(r, subject, permission)
	if err != nil || !allowed {
		return allowed, err
	}
	meeting, err := s.loadGovernanceMeetingAccessContext(r, meetingID)
	if err != nil {
		return false, err
	}
	actorUserID, err := s.currentActorUserID(r, subject)
	if err != nil || actorUserID == "" {
		return false, err
	}
	memberships, err := s.governanceMembershipAccess(r, meeting)
	if err != nil {
		return false, err
	}
	return governanceActorAllowed(actorUserID, meeting, memberships, rule), nil
}

func (s *Service) currentSubjectHasPermission(r *http.Request, subject string, permission string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(r.Context(), `
		select exists(
			select 1
			from (
				select up.permission_code
				from app_user_permissions up
				join app_users u on u.id = up.user_id
				where (u.id::text = $1 or lower(u.sub) = lower($1))
				  and up.tenant_code = public.current_tenant_code()
				union
				select rp.permission_code
				from app_user_roles ur
				join app_users u on u.id = ur.user_id
				join app_role_permissions rp on rp.role_code = ur.role_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
				  and ur.tenant_code = public.current_tenant_code()
				union
				select pp.permission_code
				from app_memberships m
				join app_users u on u.id = m.user_id
				join app_position_permissions pp on pp.position_code = m.position_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
					and m.active = true
					and m.tenant_code = public.current_tenant_code()
				union
				select rp.permission_code
				from app_memberships m
				join app_users u on u.id = m.user_id
				join app_position_roles pr on pr.position_code = m.position_code
				join app_role_permissions rp on rp.role_code = pr.role_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
					and m.active = true
					and m.tenant_code = public.current_tenant_code()
			) permissions
			where permission_code = $2
		)
	`, subject, permission).Scan(&allowed)
	return allowed, err
}

func (s *Service) currentActorName(r *http.Request, subject string) (string, error) {
	var actorName string
	err := s.pool.QueryRow(r.Context(), `
		select u.name
		from app_users u
		where (u.id::text = $1 or lower(u.sub) = lower($1))
		  and exists (
			select 1 from app_memberships m
			where m.user_id = u.id
			  and m.tenant_code = public.current_tenant_code()
			  and m.active = true
		  )
	`, subject).Scan(&actorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return actorName, err
}

func (s *Service) currentActorUserID(r *http.Request, subject string) (string, error) {
	var userID string
	err := s.pool.QueryRow(r.Context(), `
		select u.id::text
		from app_users u
		where lower(u.sub) = lower($1)
		  and exists (
			select 1 from app_memberships m
			where m.user_id = u.id
			  and m.tenant_code = public.current_tenant_code()
			  and m.active = true
		  )
	`, strings.TrimSpace(subject)).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return userID, err
}

func (s *Service) tenantAppUserName(r *http.Request, userID string) (string, error) {
	var name string
	err := s.pool.QueryRow(r.Context(), `
		select u.name
		from app_users u
		where u.id = $1::uuid
		  and exists (
			select 1 from app_memberships m
			where m.user_id = u.id
			  and m.tenant_code = public.current_tenant_code()
			  and m.active = true
		  )
	`, strings.TrimSpace(userID)).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return strings.TrimSpace(name), err
}

func (s *Service) loadGovernanceMeetingAccessContext(r *http.Request, meetingID string) (governanceMeetingAccessContext, error) {
	var meeting governanceMeetingAccessContext
	err := s.pool.QueryRow(r.Context(), `
		select id::text, school_year, organism, chairperson, secretary_name,
			coalesce(chairperson_user_id::text, ''), coalesce(secretary_user_id::text, ''), status
		from education_meetings
		where id = $1::uuid and institution_id = $2
	`, meetingID, s.institutionID(r)).Scan(
		&meeting.ID,
		&meeting.SchoolYear,
		&meeting.Organism,
		&meeting.Chairperson,
		&meeting.SecretaryName,
		&meeting.ChairpersonUserID,
		&meeting.SecretaryUserID,
		&meeting.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return governanceMeetingAccessContext{}, errGovernanceMeetingNotFound
	}
	return meeting, err
}

type governanceMembershipAccess struct {
	AppUserID   string
	RoleName    string
	VotingRight bool
}

func (s *Service) governanceMembershipAccess(r *http.Request, meeting governanceMeetingAccessContext) ([]governanceMembershipAccess, error) {
	rows, err := s.pool.Query(r.Context(), `
		select role_name, voting_right, coalesce(app_user_id::text, '')
		from education_governance_memberships
		where institution_id = $1
			and school_year = $2
			and organism = $3
			and status = 'activ'
	`, s.institutionID(r), meeting.SchoolYear, meeting.Organism)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberships := make([]governanceMembershipAccess, 0)
	for rows.Next() {
		var membership governanceMembershipAccess
		if err := rows.Scan(&membership.RoleName, &membership.VotingRight, &membership.AppUserID); err != nil {
			return nil, err
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return memberships, nil
}

// governanceActorAllowed is deliberately pure: contextual privileges require an
// immutable app_users.id. Display names and legacy membership rows without an ID
// can never authorize a privileged governance action.
func governanceActorAllowed(actorUserID string, meeting governanceMeetingAccessContext, memberships []governanceMembershipAccess, rule governanceMeetingActorRule) bool {
	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" {
		return false
	}
	if rule.AllowMeetingChair && strings.TrimSpace(meeting.ChairpersonUserID) != "" && strings.EqualFold(meeting.ChairpersonUserID, actorUserID) {
		return true
	}
	if rule.AllowMeetingSecretary && strings.TrimSpace(meeting.SecretaryUserID) != "" && strings.EqualFold(meeting.SecretaryUserID, actorUserID) {
		return true
	}
	for _, membership := range memberships {
		if strings.TrimSpace(membership.AppUserID) == "" || !strings.EqualFold(membership.AppUserID, actorUserID) {
			continue
		}
		if rule.RequireVotingRight && !membership.VotingRight {
			continue
		}
		if len(rule.MembershipRoleHints) > 0 && !containsAnyNormalizedWord(membership.RoleName, rule.MembershipRoleHints...) {
			continue
		}
		return true
	}
	return false
}

func normalizedEducationIdentity(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(
		"ă", "a",
		"â", "a",
		"î", "i",
		"ș", "s",
		"ş", "s",
		"ț", "t",
		"ţ", "t",
	).Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func containsAnyNormalizedWord(value string, keywords ...string) bool {
	normalized := normalizedEducationIdentity(value)
	for _, keyword := range keywords {
		if strings.Contains(normalized, normalizedEducationIdentity(keyword)) {
			return true
		}
	}
	return false
}
