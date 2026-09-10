package education

import (
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) requireOperationalCockpit(w http.ResponseWriter, r *http.Request, permission string) bool {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_cockpit_access_denied"})
		return false
	}
	allowed, err := s.currentSubjectHasPermission(r, subject, permission)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_cockpit_permission_check_failed"})
		return false
	}
	if !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_cockpit_access_denied", "permission": permission})
		return false
	}
	return true
}

func (s *Service) SecretariatCockpit(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperationalCockpit(w, r, "education.cockpit.secretariat.read") {
		return
	}
	var x SecretariatCockpitResponse
	x.InstitutionID = s.institutionID(r)
	err := s.pool.QueryRow(r.Context(), `select (select count(*) from education_school_classes where institution_id=$1),(select count(*) from education_students where institution_id=$1),(select count(*) from education_student_enrolments where institution_id=$1 and status='active'),(select count(*) from education_portfolios where institution_id=$1 and status='submitted')`, x.InstitutionID).Scan(&x.Classes, &x.Students, &x.ActiveEnrolments, &x.PortfoliosInReview)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_secretariat_cockpit_failed"})
		return
	}
	httpx.JSON(w, 200, x)
}
func (s *Service) HRCockpit(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperationalCockpit(w, r, "education.cockpit.hr.read") {
		return
	}
	var x HRCockpitResponse
	x.InstitutionID = s.institutionID(r)
	err := s.pool.QueryRow(r.Context(), `select (select count(*) from education_personnel where institution_id=$1),(select count(*) from education_personnel_file_documents where institution_id=$1 and expires_on between current_date and current_date+30),(select count(*) from education_personnel_file_documents where institution_id=$1 and expires_on < current_date),(select count(*) from education_evaluations where institution_id=$1 and status in ('draft','submitted','reviewed'))`, x.InstitutionID).Scan(&x.Personnel, &x.ExpiringDocuments, &x.ExpiredDocuments, &x.PendingEvaluations)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_hr_cockpit_failed"})
		return
	}
	httpx.JSON(w, 200, x)
}
func (s *Service) CommitteeCockpit(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperationalCockpit(w, r, "education.cockpit.committee.read") {
		return
	}
	var x CommitteeCockpitResponse
	x.InstitutionID = s.institutionID(r)
	err := s.pool.QueryRow(r.Context(), `select (select count(*) from education_committees where institution_id=$1),(select count(*) from education_committee_members where institution_id=$1 and status='active'),(select count(*) from education_meetings where institution_id=$1),(select count(*) from education_meeting_documents where institution_id=$1)`, x.InstitutionID).Scan(&x.Committees, &x.ActiveMembers, &x.Meetings, &x.EvidenceDocuments)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_committee_cockpit_failed"})
		return
	}
	httpx.JSON(w, 200, x)
}
func (s *Service) InspectorCockpit(w http.ResponseWriter, r *http.Request) {
	if !s.requireOperationalCockpit(w, r, "education.cockpit.inspector.read") {
		return
	}
	var x InspectorCockpitResponse
	x.InstitutionID = s.institutionID(r)
	err := s.pool.QueryRow(r.Context(), `select (select count(*) from education_meetings m where m.institution_id=$1 and (not exists(select 1 from education_meeting_minutes v where v.meeting_id=m.id and v.institution_id=$1) or not exists(select 1 from education_meeting_votes v where v.meeting_id=m.id and v.institution_id=$1))),(select count(*) from education_publications where institution_id=$1 and publication_status<>'publicat'),(select count(*) from education_publications where institution_id=$1 and mandatory and publication_status<>'publicat'),(select count(*) from education_requirement_catalog where implementation_status<>'implemented'),(select count(*) from education_evaluations where institution_id=$1 and status in ('submitted','reviewed','contested'))`, x.InstitutionID).Scan(&x.ReadinessOpen, &x.PendingPublications, &x.MandatoryPublicationPending, &x.RequirementsPending, &x.EvaluationsInReview)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "education_inspector_cockpit_failed"})
		return
	}
	httpx.JSON(w, 200, x)
}
