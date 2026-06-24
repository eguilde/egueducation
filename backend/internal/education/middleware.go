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
	ID            string
	SchoolYear    string
	Organism      string
	Chairperson   string
	SecretaryName string
	Status        string
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
	if subject == "" {
		return false, nil
	}

	allowed, err := s.currentSubjectHasPermission(r, subject, permission)
	if err != nil || allowed {
		return allowed, err
	}

	actorName, err := s.currentActorName(r, subject)
	if err != nil || actorName == "" {
		return false, err
	}

	meeting, err := s.loadGovernanceMeetingAccessContext(r, meetingID)
	if err != nil {
		return false, err
	}

	if rule.AllowMeetingChair && normalizedEducationIdentity(actorName) == normalizedEducationIdentity(meeting.Chairperson) {
		return true, nil
	}
	if rule.AllowMeetingSecretary && normalizedEducationIdentity(actorName) == normalizedEducationIdentity(meeting.SecretaryName) {
		return true, nil
	}

	return s.actorMatchesGovernanceMembership(r, actorName, meeting, rule)
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
				union
				select rp.permission_code
				from app_user_roles ur
				join app_users u on u.id = ur.user_id
				join app_role_permissions rp on rp.role_code = ur.role_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
				union
				select pp.permission_code
				from app_memberships m
				join app_users u on u.id = m.user_id
				join app_position_permissions pp on pp.position_code = m.position_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
					and m.active = true
				union
				select rp.permission_code
				from app_memberships m
				join app_users u on u.id = m.user_id
				join app_position_roles pr on pr.position_code = m.position_code
				join app_role_permissions rp on rp.role_code = pr.role_code
				where (u.id::text = $1 or lower(u.sub) = lower($1))
					and m.active = true
			) permissions
			where permission_code = $2
		)
	`, subject, permission).Scan(&allowed)
	return allowed, err
}

func (s *Service) currentActorName(r *http.Request, subject string) (string, error) {
	var actorName string
	err := s.pool.QueryRow(r.Context(), `
		select name
		from app_users
		where id::text = $1 or lower(sub) = lower($1)
	`, subject).Scan(&actorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return actorName, err
}

func (s *Service) loadGovernanceMeetingAccessContext(r *http.Request, meetingID string) (governanceMeetingAccessContext, error) {
	var meeting governanceMeetingAccessContext
	err := s.pool.QueryRow(r.Context(), `
		select id::text, school_year, organism, chairperson, secretary_name, status
		from education_meetings
		where id = $1::uuid and institution_id = $2
	`, meetingID, s.institutionID(r)).Scan(
		&meeting.ID,
		&meeting.SchoolYear,
		&meeting.Organism,
		&meeting.Chairperson,
		&meeting.SecretaryName,
		&meeting.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return governanceMeetingAccessContext{}, errGovernanceMeetingNotFound
	}
	return meeting, err
}

func (s *Service) actorMatchesGovernanceMembership(
	r *http.Request,
	actorName string,
	meeting governanceMeetingAccessContext,
	rule governanceMeetingActorRule,
) (bool, error) {
	rows, err := s.pool.Query(r.Context(), `
		select full_name, role_name, voting_right
		from education_governance_memberships
		where institution_id = $1
			and school_year = $2
			and organism = $3
			and status = 'activ'
	`, s.institutionID(r), meeting.SchoolYear, meeting.Organism)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	normalizedActor := normalizedEducationIdentity(actorName)
	for rows.Next() {
		var fullName string
		var roleName string
		var votingRight bool
		if err := rows.Scan(&fullName, &roleName, &votingRight); err != nil {
			return false, err
		}
		if normalizedEducationIdentity(fullName) != normalizedActor {
			continue
		}
		if rule.RequireVotingRight && !votingRight {
			continue
		}
		if len(rule.MembershipRoleHints) > 0 && !containsAnyNormalizedWord(roleName, rule.MembershipRoleHints...) {
			continue
		}
		return true, nil
	}

	return false, rows.Err()
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
