package admission

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Service) ListCampaigns(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "campaign_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"code": {}, "title": {}, "opens_on": {}, "status": {}}, []string{"code", "title", "school_year", "status"})
	sort := map[string]string{"code": "code", "title": "title", "opens_on": "opens_on", "status": "status"}[q.Sort]
	if sort == "" {
		sort = "opens_on"
	}
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"code", "title", "school_year", "status"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "code" || f == "title" {
				where += fmt.Sprintf(" and %s ilike '%%'||$%d||'%%'", f, n)
			} else {
				where += fmt.Sprintf(" and %s=$%d", f, n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_campaigns"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,source_id::text,code,title,school_year,offering_id::text,location_id::text,authorization_id::text,class_offering_context_id::text,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,decision_due_on,status,expected_version from school_admission_campaigns"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	defer rows.Close()
	items := []Campaign{}
	for rows.Next() {
		var x Campaign
		var opens, closes time.Time
		var due *time.Time
		if err = rows.Scan(&x.ID, &x.SourceID, &x.Code, &x.Title, &x.SchoolYear, &x.OfferingID, &x.LocationID, &x.AuthorizationID, &x.ClassOfferingContextID, &x.CapacityLimit, &x.CapacityUnit, &x.StudentPlaceLimit, &x.CapacityBasis, &x.Shift, &opens, &closes, &due, &x.Status, &x.ExpectedVersion); err != nil {
			writeError(w, err, "campaign_list_failed")
			return
		}
		x.OpensOn = opens.Format(time.DateOnly)
		x.ClosesOn = closes.Format(time.DateOnly)
		if due != nil {
			v := due.Format(time.DateOnly)
			x.DecisionDueOn = &v
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "campaign_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) ListApplications(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "application_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "registratura.read")
	}
	if err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"application_no": {}, "status": {}, "submitted_at": {}}, []string{"campaign_id", "application_no", "status"})
	sort := map[string]string{"application_no": "application_no", "status": "status", "submitted_at": "submitted_at"}[q.Sort]
	if sort == "" {
		sort = "created_at"
	}
	where := " where a.tenant_code=$1 and a.institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"campaign_id", "application_no", "status"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "campaign_id" {
				where += fmt.Sprintf(" and a.campaign_id=$%d::uuid", n)
			} else if f == "application_no" {
				where += fmt.Sprintf(" and a.application_no ilike '%%'||$%d||'%%'", n)
			} else {
				where += fmt.Sprintf(" and a.status=$%d", n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_applications a"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), `select a.id::text,a.campaign_id::text,a.application_no,a.candidate_party_id::text,a.student_id::text,a.submitted_at,a.status,a.consent_snapshot,a.expected_version,c.code,p.display_name,
		not exists(select 1 from school_admission_document_requirements requirement where requirement.tenant_code=a.tenant_code and requirement.institution_id=a.institution_id and requirement.campaign_id=a.campaign_id and requirement.required and not exists(select 1 from school_admission_application_documents d where d.tenant_code=a.tenant_code and d.institution_id=a.institution_id and d.application_id=a.id and d.document_requirement_id=requirement.id and d.status in ('accepted','waived'))),
		not exists(select 1 from school_admission_criteria criterion left join school_admission_criterion_assessments assessment on assessment.tenant_code=criterion.tenant_code and assessment.institution_id=criterion.institution_id and assessment.application_id=a.id and assessment.criterion_id=criterion.id where criterion.tenant_code=a.tenant_code and criterion.institution_id=a.institution_id and criterion.campaign_id=a.campaign_id and criterion.required and coalesce(assessment.outcome,'pending')<>'met')
		from school_admission_applications a join school_admission_campaigns c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.campaign_id join app_parties p on p.tenant_code=a.tenant_code and p.institution_id=a.institution_id and p.id=a.candidate_party_id`+where+fmt.Sprintf(" order by a.%s %s nulls last,a.id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	defer rows.Close()
	items := []Application{}
	for rows.Next() {
		var x Application
		var submitted *time.Time
		if err = rows.Scan(&x.ID, &x.CampaignID, &x.ApplicationNo, &x.CandidatePartyID, &x.StudentID, &submitted, &x.Status, &x.ConsentSnapshot, &x.ExpectedVersion, &x.CampaignCode, &x.CandidateName, &x.DocumentsComplete, &x.CriteriaComplete); err != nil {
			writeError(w, err, "application_list_failed")
			return
		}
		if submitted != nil {
			v := submitted.Format(time.RFC3339Nano)
			x.SubmittedAt = &v
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "application_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) ListDecisions(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "decision_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "registratura.read")
	}
	if err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"decision_no": {}, "outcome": {}, "decided_at": {}}, []string{"application_id", "decision_no", "outcome"})
	sort := map[string]string{"decision_no": "decision_no", "outcome": "outcome", "decided_at": "decided_at"}[q.Sort]
	if sort == "" {
		sort = "decided_at"
	}
	where := " where d.tenant_code=$1 and d.institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"application_id", "decision_no", "outcome"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "application_id" {
				where += fmt.Sprintf(" and d.application_id=$%d::uuid", n)
			} else if f == "decision_no" {
				where += fmt.Sprintf(" and d.decision_no ilike '%%'||$%d||'%%'", n)
			} else {
				where += fmt.Sprintf(" and d.outcome=$%d", n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_decisions d"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select d.id::text,d.application_id::text,d.capacity_allocation_id::text,d.policy_evaluation_v2_id::text,d.decision_no,d.outcome,d.rationale,d.ranking_value,d.appeal_deadline,d.decided_at,a.application_no,p.display_name from school_admission_decisions d join school_admission_applications a on a.tenant_code=d.tenant_code and a.institution_id=d.institution_id and a.id=d.application_id join app_parties p on p.tenant_code=a.tenant_code and p.institution_id=a.institution_id and p.id=a.candidate_party_id"+where+fmt.Sprintf(" order by d.%s %s,d.id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	defer rows.Close()
	items := []Decision{}
	for rows.Next() {
		var x Decision
		var deadline *time.Time
		var decided time.Time
		if err = rows.Scan(&x.ID, &x.ApplicationID, &x.CapacityAllocationID, &x.PolicyEvaluationV2ID, &x.DecisionNo, &x.Outcome, &x.Rationale, &x.RankingValue, &deadline, &decided, &x.ApplicationNo, &x.CandidateName); err != nil {
			writeError(w, err, "decision_list_failed")
			return
		}
		if deadline != nil {
			v := deadline.Format(time.DateOnly)
			x.AppealDeadline = &v
		}
		x.DecidedAt = decided.Format(time.RFC3339Nano)
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "decision_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) ListAppeals(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "appeal_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "registratura.read")
	}
	if err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"appeal_no": {}, "status": {}, "submitted_at": {}}, []string{"application_id", "appeal_no", "status"})
	sort := map[string]string{"appeal_no": "appeal_no", "status": "status", "submitted_at": "submitted_at"}[q.Sort]
	if sort == "" {
		sort = "submitted_at"
	}
	where := " where appeal.tenant_code=$1 and appeal.institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"application_id", "appeal_no", "status"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "application_id" {
				where += fmt.Sprintf(" and appeal.application_id=$%d::uuid", n)
			} else if f == "appeal_no" {
				where += fmt.Sprintf(" and appeal.appeal_no ilike '%%'||$%d||'%%'", n)
			} else {
				where += fmt.Sprintf(" and appeal.status=$%d", n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_appeals appeal"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select appeal.id::text,appeal.application_id::text,appeal.decision_id::text,appeal.appeal_no,appeal.status,appeal.submitted_at,appeal.submitted_by_party_id::text,appeal.expected_version,a.application_no from school_admission_appeals appeal join school_admission_applications a on a.tenant_code=appeal.tenant_code and a.institution_id=appeal.institution_id and a.id=appeal.application_id"+where+fmt.Sprintf(" order by appeal.%s %s nulls last,appeal.id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	defer rows.Close()
	items := []Appeal{}
	for rows.Next() {
		var x Appeal
		var submitted *time.Time
		if err = rows.Scan(&x.ID, &x.ApplicationID, &x.DecisionID, &x.AppealNo, &x.Status, &submitted, &x.SubmittedByPartyID, &x.ExpectedVersion, &x.ApplicationNo); err != nil {
			writeError(w, err, "appeal_list_failed")
			return
		}
		if submitted != nil {
			v := submitted.Format(time.RFC3339Nano)
			x.SubmittedAt = &v
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "appeal_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) GetApplication(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "application_read_failed")
		return
	}
	id := chi.URLParam(r, "applicationID")
	if !validUUID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_id"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "registratura.read")
	}
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	var app Application
	var submitted *time.Time
	err = tx.QueryRow(r.Context(), `select id::text,campaign_id::text,application_no,candidate_party_id::text,student_id::text,submitted_at,status,consent_snapshot,expected_version from school_admission_applications where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&app.ID, &app.CampaignID, &app.ApplicationNo, &app.CandidatePartyID, &app.StudentID, &submitted, &app.Status, &app.ConsentSnapshot, &app.ExpectedVersion)
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	if submitted != nil {
		v := submitted.Format(time.RFC3339Nano)
		app.SubmittedAt = &v
	}
	assessmentRows, err := tx.Query(r.Context(), `select a.id::text,a.criterion_id::text,c.code,c.title,c.required,a.outcome,a.score,a.rationale,a.evidence_snapshot,a.expected_version from school_admission_criterion_assessments a join school_admission_criteria c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.criterion_id where a.tenant_code=$1 and a.institution_id=$2 and a.application_id=$3::uuid order by c.ordinal`, sc.tenant, sc.institution, id)
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	assessments := []CriterionAssessment{}
	for assessmentRows.Next() {
		var x CriterionAssessment
		if err = assessmentRows.Scan(&x.ID, &x.CriterionID, &x.Code, &x.Title, &x.Required, &x.Outcome, &x.Score, &x.Rationale, &x.EvidenceSnapshot, &x.ExpectedVersion); err != nil {
			assessmentRows.Close()
			writeError(w, err, "application_read_failed")
			return
		}
		assessments = append(assessments, x)
	}
	err = assessmentRows.Err()
	assessmentRows.Close()
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	documentRows, err := tx.Query(r.Context(), `select id::text,document_requirement_id::text,document_kind,status,archive_document_id::text,archive_version_id::text,expected_version from school_admission_application_documents where tenant_code=$1 and institution_id=$2 and application_id=$3::uuid order by created_at`, sc.tenant, sc.institution, id)
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	documents := []ApplicationDocument{}
	for documentRows.Next() {
		var x ApplicationDocument
		if err = documentRows.Scan(&x.ID, &x.RequirementID, &x.DocumentKind, &x.Status, &x.ArchiveDocumentID, &x.ArchiveVersionID, &x.ExpectedVersion); err != nil {
			documentRows.Close()
			writeError(w, err, "application_read_failed")
			return
		}
		documents = append(documents, x)
	}
	err = documentRows.Err()
	documentRows.Close()
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	decisionRows, err := tx.Query(r.Context(), `select id::text,application_id::text,capacity_allocation_id::text,policy_evaluation_v2_id::text,decision_no,outcome,rationale,ranking_value,appeal_deadline,decided_at from school_admission_decisions where tenant_code=$1 and institution_id=$2 and application_id=$3::uuid order by decided_at desc`, sc.tenant, sc.institution, id)
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	decisions := []Decision{}
	for decisionRows.Next() {
		var x Decision
		var deadline *time.Time
		var decided time.Time
		if err = decisionRows.Scan(&x.ID, &x.ApplicationID, &x.CapacityAllocationID, &x.PolicyEvaluationV2ID, &x.DecisionNo, &x.Outcome, &x.Rationale, &x.RankingValue, &deadline, &decided); err != nil {
			decisionRows.Close()
			writeError(w, err, "application_read_failed")
			return
		}
		if deadline != nil {
			v := deadline.Format(time.DateOnly)
			x.AppealDeadline = &v
		}
		x.DecidedAt = decided.Format(time.RFC3339Nano)
		decisions = append(decisions, x)
	}
	err = decisionRows.Err()
	decisionRows.Close()
	if err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "application_read_failed")
		return
	}
	httpx.JSON(w, 200, ApplicationDetail{Application: app, Assessments: assessments, Documents: documents, Decisions: decisions})
}

func (s *Service) ListCriteria(w http.ResponseWriter, r *http.Request) {
	s.listCampaignConfig(w, r, "criteria")
}
func (s *Service) ListDocumentRequirements(w http.ResponseWriter, r *http.Request) {
	s.listCampaignConfig(w, r, "documents")
}
func (s *Service) listCampaignConfig(w http.ResponseWriter, r *http.Request, kind string) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "campaign_config_failed")
		return
	}
	campaignID := chi.URLParam(r, "campaignID")
	if !validUUID(campaignID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_campaign_id"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	if kind == "criteria" {
		q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"code": {}, "title": {}, "kind": {}, "required": {}, "ordinal": {}}, []string{"code", "title", "kind", "required"})
		sort := map[string]string{"code": "code", "title": "title", "kind": "criterion_kind", "required": "required", "ordinal": "ordinal"}[q.Sort]
		if sort == "" {
			sort = "ordinal"
		}
		where := " where tenant_code=$1 and institution_id=$2 and campaign_id=$3::uuid"
		args := []any{sc.tenant, sc.institution, campaignID}
		for _, f := range []string{"code", "title", "kind", "required"} {
			if v := q.Filters[f]; v != "" {
				args = append(args, v)
				n := len(args)
				if f == "code" || f == "title" {
					where += fmt.Sprintf(" and %s ilike '%%'||$%d||'%%'", f, n)
				} else if f == "kind" {
					where += fmt.Sprintf(" and criterion_kind=$%d", n)
				} else {
					where += fmt.Sprintf(" and required=$%d::boolean", n)
				}
			}
		}
		var total int
		if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_criteria"+where, args...).Scan(&total); err != nil {
			writeError(w, err, "campaign_config_failed")
			return
		}
		args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
		rows, e := tx.Query(r.Context(), "select id::text,campaign_id::text,code,title,criterion_kind,required,weight,ordinal,rule_snapshot,expected_version from school_admission_criteria"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
		if e != nil {
			writeError(w, e, "campaign_config_failed")
			return
		}
		defer rows.Close()
		items := []Criterion{}
		for rows.Next() {
			var x Criterion
			if e = rows.Scan(&x.ID, &x.CampaignID, &x.Code, &x.Title, &x.Kind, &x.Required, &x.Weight, &x.Ordinal, &x.RuleSnapshot, &x.ExpectedVersion); e != nil {
				writeError(w, e, "campaign_config_failed")
				return
			}
			items = append(items, x)
		}
		if e = rows.Err(); e != nil {
			writeError(w, e, "campaign_config_failed")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			writeError(w, e, "campaign_config_failed")
			return
		}
		httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"code": {}, "title": {}, "required": {}, "ordinal": {}}, []string{"code", "title", "required"})
	sort := map[string]string{"code": "code", "title": "title", "required": "required", "ordinal": "ordinal"}[q.Sort]
	if sort == "" {
		sort = "ordinal"
	}
	where := " where tenant_code=$1 and institution_id=$2 and campaign_id=$3::uuid"
	args := []any{sc.tenant, sc.institution, campaignID}
	for _, f := range []string{"code", "title", "required"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			n := len(args)
			if f == "code" || f == "title" {
				where += fmt.Sprintf(" and %s ilike '%%'||$%d||'%%'", f, n)
			} else {
				where += fmt.Sprintf(" and required=$%d::boolean", n)
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_document_requirements"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, e := tx.Query(r.Context(), "select id::text,campaign_id::text,code,title,required,allowed_mime_types,ordinal,expected_version from school_admission_document_requirements"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if e != nil {
		writeError(w, e, "campaign_config_failed")
		return
	}
	defer rows.Close()
	items := []DocumentRequirement{}
	for rows.Next() {
		var x DocumentRequirement
		if e = rows.Scan(&x.ID, &x.CampaignID, &x.Code, &x.Title, &x.Required, &x.AllowedMIMETypes, &x.Ordinal, &x.ExpectedVersion); e != nil {
			writeError(w, e, "campaign_config_failed")
			return
		}
		items = append(items, x)
	}
	if e = rows.Err(); e != nil {
		writeError(w, e, "campaign_config_failed")
		return
	}
	if e = tx.Commit(r.Context()); e != nil {
		writeError(w, e, "campaign_config_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) AddCriterion(w http.ResponseWriter, r *http.Request) {
	var in CriterionInput
	s.addCampaignConfig(w, r, "criterion", &in)
}
func (s *Service) AddDocumentRequirement(w http.ResponseWriter, r *http.Request) {
	var in DocumentRequirementInput
	s.addCampaignConfig(w, r, "document_requirement", &in)
}
func (s *Service) addCampaignConfig(w http.ResponseWriter, r *http.Request, kind string, input any) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "campaign_config_failed")
		return
	}
	campaignID := chi.URLParam(r, "campaignID")
	key, keyOK := idempotencyKey(r)
	if !validUUID(campaignID) || !keyOK || decode(w, r, input) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_campaign_config"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.campaign."+kind+".add", key, fingerprint(struct {
		CampaignID string
		Input      any
	}{campaignID, input}), uuid.NewString())
	if err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	if !replay {
		var status string
		err = tx.QueryRow(r.Context(), `select status from school_admission_campaigns where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, sc.tenant, sc.institution, campaignID).Scan(&status)
		if err == nil && status != "draft" {
			err = errInvalidState
		}
		if err == nil && kind == "criterion" {
			c := input.(*CriterionInput)
			c.Code = strings.TrimSpace(c.Code)
			c.Title = strings.TrimSpace(c.Title)
			if !codePattern.MatchString(c.Code) || c.Title == "" || c.Ordinal < 1 {
				err = errInvalidInput
			} else {
				_, err = tx.Exec(r.Context(), `insert into school_admission_criteria(id,tenant_code,institution_id,campaign_id,code,title,criterion_kind,required,weight,ordinal,rule_snapshot,created_by_subject,updated_by_subject) values($1::uuid,$2,$3,$4::uuid,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$12)`, id, sc.tenant, sc.institution, campaignID, c.Code, c.Title, c.Kind, c.Required, c.Weight, c.Ordinal, rawObject(c.RuleSnapshot), sc.actor)
			}
		}
		if err == nil && kind == "document_requirement" {
			d := input.(*DocumentRequirementInput)
			d.Code = strings.TrimSpace(d.Code)
			d.Title = strings.TrimSpace(d.Title)
			if !codePattern.MatchString(d.Code) || d.Title == "" || d.Ordinal < 1 {
				err = errInvalidInput
			} else {
				_, err = tx.Exec(r.Context(), `insert into school_admission_document_requirements(id,tenant_code,institution_id,campaign_id,code,title,required,allowed_mime_types,ordinal,created_by_subject,updated_by_subject) values($1::uuid,$2,$3,$4::uuid,$5,$6,$7,$8,$9,$10,$10)`, id, sc.tenant, sc.institution, campaignID, d.Code, d.Title, d.Required, d.AllowedMIMETypes, d.Ordinal, sc.actor)
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_campaign", campaignID, "admission.campaign."+kind+".added", map[string]any{"campaign_id": campaignID, "id": id})
		}
	}
	if err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "campaign_config_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "active", ExpectedVersion: 1, Replayed: replay})
}
