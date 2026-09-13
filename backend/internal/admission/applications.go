package admission

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) CreateApplication(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "application_create_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in CreateApplicationRequest
	if decode(w, r, &in) != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_application"})
		return
	}
	in.ApplicationNo = strings.TrimSpace(in.ApplicationNo)
	if !validUUID(in.CampaignID) || !validUUID(in.CandidatePartyID) || !codePattern.MatchString(in.ApplicationNo) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application"})
		return
	}
	var consent map[string]any
	if json.Unmarshal(rawObject(in.ConsentSnapshot), &consent) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_consent"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "application_create_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "application_create_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.application.create", key, fingerprint(in), uuid.NewString())
	if err != nil {
		writeError(w, err, "application_create_failed")
		return
	}
	if !replay {
		var candidatePhysical bool
		err = tx.QueryRow(r.Context(), `select exists(select 1 from app_parties where tenant_code=$1 and institution_id=$2 and id=$3::uuid and party_type='physical' and active)`, sc.tenant, sc.institution, in.CandidatePartyID).Scan(&candidatePhysical)
		if err == nil && !candidatePhysical {
			err = errInvalidInput
		}
		var campaignStatus string
		var inWindow bool
		if err == nil {
			err = tx.QueryRow(r.Context(), `select status,current_date between opens_on and closes_on from school_admission_campaigns where tenant_code=$1 and institution_id=$2 and id=$3::uuid for share`, sc.tenant, sc.institution, in.CampaignID).Scan(&campaignStatus, &inWindow)
		}
		if err == nil && (campaignStatus != "open" || !inWindow) {
			err = errInvalidState
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `insert into school_admission_applications(id,tenant_code,institution_id,campaign_id,application_no,candidate_party_id,consent_snapshot,created_by_subject,updated_by_subject) values($1::uuid,$2,$3,$4::uuid,$5,$6::uuid,$7::jsonb,$8,$8)`, id, sc.tenant, sc.institution, in.CampaignID, in.ApplicationNo, in.CandidatePartyID, rawObject(in.ConsentSnapshot), sc.actor)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `insert into school_admission_criterion_assessments(tenant_code,institution_id,application_id,criterion_id,outcome,created_by_subject,updated_by_subject) select $1,$2,$3::uuid,c.id,'pending',$4,$4 from school_admission_criteria c where c.tenant_code=$1 and c.institution_id=$2 and c.campaign_id=$5::uuid`, sc.tenant, sc.institution, id, sc.actor, in.CampaignID)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `insert into school_admission_application_documents(tenant_code,institution_id,application_id,document_requirement_id,document_kind,status,created_by_subject,updated_by_subject) select $1,$2,$3::uuid,d.id,d.code,'requested',$4,$4 from school_admission_document_requirements d where d.tenant_code=$1 and d.institution_id=$2 and d.campaign_id=$5::uuid`, sc.tenant, sc.institution, id, sc.actor, in.CampaignID)
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_application", id, "admission.application.created", map[string]any{"application_id": id, "campaign_id": in.CampaignID})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.application.created", "school_admission_application", id)
		}
	}
	if err != nil {
		writeError(w, err, "application_create_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "application_create_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "draft", ExpectedVersion: 1, Replayed: replay})
}

var applicationTransitions = map[string]map[string]bool{"draft": {"submitted": true, "cancelled": true}, "submitted": {"under_review": true, "withdrawn": true, "cancelled": true}, "under_review": {"withdrawn": true, "cancelled": true}, "waitlisted": {"withdrawn": true, "cancelled": true}}

func (s *Service) TransitionApplication(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "application_transition_failed")
		return
	}
	id := chi.URLParam(r, "applicationID")
	if !validUUID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_id"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in TransitionRequest
	if decode(w, r, &in) != nil || in.ExpectedVersion < 1 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_transition"})
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "application_transition_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "application_transition_failed")
		return
	}
	_, replay, err := reserve(r.Context(), tx, sc, "admission.application.transition", key, fingerprint(struct {
		ID    string
		Input TransitionRequest
	}{id, in}), id)
	if err != nil {
		writeError(w, err, "application_transition_failed")
		return
	}
	if !replay {
		var current string
		var campaignOpen bool
		err = tx.QueryRow(r.Context(), `select a.status,(c.status='open' and current_date between c.opens_on and c.closes_on) from school_admission_applications a join school_admission_campaigns c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.campaign_id where a.tenant_code=$1 and a.institution_id=$2 and a.id=$3::uuid and a.expected_version=$4 for update`, sc.tenant, sc.institution, id, in.ExpectedVersion).Scan(&current, &campaignOpen)
		if errors.Is(err, pgx.ErrNoRows) {
			err = errVersionConflict
		}
		if err == nil && !applicationTransitions[current][in.Status] {
			err = errInvalidState
		}
		if err == nil && in.Status == "submitted" && !campaignOpen {
			err = errInvalidState
		}
		if err == nil {
			tag, e := tx.Exec(r.Context(), `update school_admission_applications set status=$1,submitted_at=case when $1='submitted' then now() else submitted_at end,expected_version=expected_version+1,updated_by_subject=$2,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, in.Status, sc.actor, sc.tenant, sc.institution, id, in.ExpectedVersion)
			err = e
			if err == nil && tag.RowsAffected() != 1 {
				err = errVersionConflict
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_application", id, "admission.application."+in.Status, map[string]any{"application_id": id, "status": in.Status})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.application."+in.Status, "school_admission_application", id)
		}
	}
	if err != nil {
		writeError(w, err, "application_transition_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "application_transition_failed")
		return
	}
	httpx.JSON(w, 200, CommandResult{ID: id, Status: in.Status, ExpectedVersion: in.ExpectedVersion + 1, Replayed: replay})
}

func (s *Service) AssessCriterion(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "assessment_failed")
		return
	}
	applicationID := chi.URLParam(r, "applicationID")
	if !validUUID(applicationID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_id"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in AssessCriterionRequest
	if decode(w, r, &in) != nil || !validUUID(in.CriterionID) || in.ExpectedVersion < 1 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_assessment"})
		return
	}
	in.Outcome = strings.TrimSpace(in.Outcome)
	if in.Outcome != "met" && in.Outcome != "not_met" && in.Outcome != "not_applicable" && in.Outcome != "indeterminate" {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_assessment_outcome"})
		return
	}
	var evidence map[string]any
	if json.Unmarshal(rawObject(in.EvidenceSnapshot), &evidence) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_assessment_evidence"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	var assessmentID string
	err = tx.QueryRow(r.Context(), `select id::text from school_admission_criterion_assessments where tenant_code=$1 and institution_id=$2 and application_id=$3::uuid and criterion_id=$4::uuid`, sc.tenant, sc.institution, applicationID, in.CriterionID).Scan(&assessmentID)
	if err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	reserved, replay, err := reserve(r.Context(), tx, sc, "admission.assessment.upsert", key, fingerprint(struct {
		ApplicationID string
		Input         AssessCriterionRequest
	}{applicationID, in}), assessmentID)
	if err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	assessmentID = reserved
	if !replay {
		tag, e := tx.Exec(r.Context(), `update school_admission_criterion_assessments assessment set outcome=$1,score=$2,rationale=$3,evidence_snapshot=$4::jsonb,assessed_by_subject=$5,assessed_at=now(),expected_version=expected_version+1,updated_by_subject=$5,updated_at=now() where assessment.tenant_code=$6 and assessment.institution_id=$7 and assessment.application_id=$8::uuid and assessment.criterion_id=$9::uuid and assessment.expected_version=$10 and exists(select 1 from school_admission_applications app where app.tenant_code=assessment.tenant_code and app.institution_id=assessment.institution_id and app.id=assessment.application_id and app.status not in ('admitted','rejected','withdrawn','cancelled')) and not exists(select 1 from school_admission_decisions decision where decision.tenant_code=assessment.tenant_code and decision.institution_id=assessment.institution_id and decision.application_id=assessment.application_id)`, in.Outcome, in.Score, strings.TrimSpace(in.Rationale), rawObject(in.EvidenceSnapshot), sc.actor, sc.tenant, sc.institution, applicationID, in.CriterionID, in.ExpectedVersion)
		err = e
		if err == nil && tag.RowsAffected() != 1 {
			err = errVersionConflict
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_application", applicationID, "admission.criterion.assessed", map[string]any{"applicationId": applicationID, "criterionId": in.CriterionID, "outcome": in.Outcome})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.criterion.assessed", "school_admission_application", applicationID)
		}
	}
	if err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "assessment_failed")
		return
	}
	httpx.JSON(w, 200, CommandResult{ID: assessmentID, Status: in.Outcome, ExpectedVersion: in.ExpectedVersion + 1, Replayed: replay})
}
