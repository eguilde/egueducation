package admission

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/institution"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type decisionContext struct{ campaignID, offeringID, locationID, authorizationID, classContextID, status, campaignStatus string }

func validDecision(in IssueDecisionRequest) bool {
	in.DecisionNo = strings.TrimSpace(in.DecisionNo)
	in.Outcome = strings.TrimSpace(in.Outcome)
	in.Rationale = strings.TrimSpace(in.Rationale)
	if !codePattern.MatchString(in.DecisionNo) || in.Rationale == "" || in.ExpectedVersion < 1 || !validUUID(in.Archive.DocumentID) || !validUUID(in.Archive.VersionID) {
		return false
	}
	if in.Outcome != "admitted" && in.Outcome != "waitlisted" && in.Outcome != "rejected" && in.Outcome != "withdrawn" && in.Outcome != "cancelled" {
		return false
	}
	return in.AppealDeadline == nil || validDate(*in.AppealDeadline)
}

func loadDecisionContext(ctx context.Context, tx pgx.Tx, sc scope, applicationID string, expectedVersion int) (decisionContext, error) {
	var c decisionContext
	err := tx.QueryRow(ctx, `select a.campaign_id::text,c.offering_id::text,c.location_id::text,c.authorization_id::text,c.class_offering_context_id::text,a.status,c.status from school_admission_applications a join school_admission_campaigns c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.campaign_id where a.tenant_code=$1 and a.institution_id=$2 and a.id=$3::uuid and a.expected_version=$4 for update`, sc.tenant, sc.institution, applicationID, expectedVersion).Scan(&c.campaignID, &c.offeringID, &c.locationID, &c.authorizationID, &c.classContextID, &c.status, &c.campaignStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, errVersionConflict
	}
	return c, err
}

func (s *Service) IssueDecision(w http.ResponseWriter, r *http.Request) {
	// A browser cannot safely provide the server-derived canonical signing
	// payload. Keep the legacy endpoint explicit and fail closed rather than
	// allowing an unsigned/direct legal mutation during migration.
	httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_legal_preparation_required"})
	return
	/*
		sc, ok := requestScope(r)
		if !ok {
			writeError(w, errBadScope, "decision_failed")
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
		var in IssueDecisionRequest
		if decode(w, r, &in) != nil || !validDecision(in) {
			httpx.JSON(w, 422, map[string]any{"code": "invalid_decision"})
			return
		}
		tx, err := s.begin(r.Context(), sc)
		if err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		defer tx.Rollback(r.Context())
		if err = requirePermission(r.Context(), tx, sc, permissionDecide); err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		if err = requirePermission(r.Context(), tx, sc, "earchiva.read"); err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		id, replay, err := reserve(r.Context(), tx, sc, "admission.decision.issue", key, fingerprint(struct {
			ApplicationID string
			Input         IssueDecisionRequest
		}{applicationID, in}), uuid.NewString())
		if err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		var decision Decision
		if !replay {
			decision, err = s.issueDecisionTx(r.Context(), tx, sc, id, applicationID, in, "")
			if err == nil {
				err = outbox(r.Context(), tx, sc, "admission_application", applicationID, "admission.decision.issued", map[string]any{"applicationId": applicationID, "decisionId": id, "outcome": in.Outcome})
			}
			if err == nil {
				err = auditEvent(r.Context(), tx, sc, "admission.decision.issued", "school_admission_decision", id)
			}
		} else {
			err = tx.QueryRow(r.Context(), `select id::text,application_id::text,capacity_allocation_id::text,policy_evaluation_v2_id::text,decision_no,outcome,rationale,ranking_value,appeal_deadline::text,decided_at::text from school_admission_decisions where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&decision.ID, &decision.ApplicationID, &decision.CapacityAllocationID, &decision.PolicyEvaluationV2ID, &decision.DecisionNo, &decision.Outcome, &decision.Rationale, &decision.RankingValue, &decision.AppealDeadline, &decision.DecidedAt)
		}
		if err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeError(w, err, "decision_failed")
			return
		}
		httpx.JSON(w, 201, decision)
	*/
}

func (s *Service) issueDecisionTx(ctx context.Context, tx pgx.Tx, sc scope, id, applicationID string, in IssueDecisionRequest, supersedes string) (Decision, error) {
	c, err := loadDecisionContext(ctx, tx, sc, applicationID, in.ExpectedVersion)
	if err != nil {
		return Decision{}, err
	}
	if c.status != "submitted" && c.status != "under_review" && c.status != "waitlisted" && c.status != "rejected" && !(supersedes != "" && c.status == "admitted") {
		return Decision{}, errInvalidState
	}
	if supersedes == "" && (c.status == "waitlisted" || c.status == "rejected") {
		return Decision{}, errInvalidState
	}
	if c.campaignStatus != "open" && c.campaignStatus != "closed" {
		return Decision{}, errInvalidState
	}
	if in.Outcome == "admitted" {
		var incomplete bool
		err = tx.QueryRow(ctx, `select exists(select 1 from school_admission_document_requirements requirement where requirement.tenant_code=$1 and requirement.institution_id=$2 and requirement.campaign_id=$3::uuid and requirement.required and not exists(select 1 from school_admission_application_documents d where d.tenant_code=requirement.tenant_code and d.institution_id=requirement.institution_id and d.application_id=$4::uuid and d.document_requirement_id=requirement.id and d.status in ('accepted','waived')))`, sc.tenant, sc.institution, c.campaignID, applicationID).Scan(&incomplete)
		if err != nil {
			return Decision{}, err
		}
		if incomplete {
			return Decision{}, errInvalidInput
		}
	}
	now := time.Now().UTC()
	policy, err := institution.EvaluateOperationPolicyV2Tx(ctx, tx, institution.OperationPolicyV2Request{OperationCode: "admission.decision.issue", EffectiveOn: now, DecisionKind: "operation", OfferingID: c.offeringID, LocationID: c.locationID, ActorSubject: sc.actor, Context: decisionPolicyContext(c, applicationID, in.Outcome)})
	if err != nil {
		return Decision{}, err
	}
	if !policy.Allowed {
		return Decision{}, errPolicyDenied
	}
	snap, err := loadArchiveSnapshot(ctx, tx, sc.institution, in.Archive)
	if err != nil {
		return Decision{}, err
	}
	if !strings.EqualFold(snap.mime, "application/pdf") {
		return Decision{}, errInvalidInput
	}
	var allocationID *string
	if in.Outcome == "admitted" {
		// A favourable appeal supersedes a decision for the same application.  Its
		// existing active allocation is the legal capacity commitment; reusing it
		// avoids both double-booking and the active-allocation unique index.
		if supersedes != "" {
			var existingID string
			err = tx.QueryRow(ctx, `select id::text from school_admission_capacity_allocations where tenant_code=$1 and institution_id=$2 and application_id=$3::uuid and status in ('held','consumed') for update`, sc.tenant, sc.institution, applicationID).Scan(&existingID)
			if err == nil {
				allocationID = &existingID
			} else if errors.Is(err, pgx.ErrNoRows) {
				err = nil
			} else {
				return Decision{}, err
			}
		}
		if allocationID == nil {
			aID := uuid.NewString()
			_, err = tx.Exec(ctx, `insert into school_admission_capacity_allocations(id,tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,status,created_by_subject,updated_by_subject) select $1::uuid,$2,$3,$4::uuid,c.id,c.class_offering_context_id,c.authorization_id,c.capacity_unit,c.shift,'held',$5,$5 from school_admission_campaigns c where c.tenant_code=$2 and c.institution_id=$3 and c.id=$6::uuid`, aID, sc.tenant, sc.institution, applicationID, sc.actor, c.campaignID)
			if err != nil {
				return Decision{}, err
			}
			allocationID = &aID
		}
	}
	decisionSnapshot, _ := json.Marshal(map[string]any{"application_id": applicationID, "campaign_id": c.campaignID, "authorization_id": c.authorizationID, "outcome": in.Outcome, "policy_evaluation_v2_id": policy.EvaluationID})
	_, err = tx.Exec(ctx, `insert into school_admission_decisions(id,tenant_code,institution_id,application_id,capacity_allocation_id,policy_evaluation_v2_id,decision_no,outcome,rationale,ranking_value,appeal_deadline,decided_at,decided_by_subject,supersedes_decision_id,decision_snapshot,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256)
		values($1::uuid,$2,$3,$4::uuid,nullif($5,'')::uuid,$6::uuid,$7,$8,$9,$10,nullif($11,'')::date,$12,$13,nullif($14,'')::uuid,$15::jsonb,$16::uuid,$17::uuid,$18,$19,$20,$21,$22,$23)`, id, sc.tenant, sc.institution, applicationID, optional(allocationID), policy.EvaluationID, in.DecisionNo, in.Outcome, in.Rationale, in.RankingValue, optional(in.AppealDeadline), now, sc.actor, supersedes, decisionSnapshot, in.Archive.DocumentID, in.Archive.VersionID, snap.versionNo, snap.bucket, snap.objectKey, snap.objectVersion, snap.retention, snap.sha)
	if err != nil {
		return Decision{}, err
	}
	if err = persistDSSBinding(ctx, tx, sc, "admission_decision", id, policy.EvaluationID, in.Archive, snap, sc.actor); err != nil {
		return Decision{}, err
	}
	tag, err := tx.Exec(ctx, `update school_admission_applications set status=$1,expected_version=expected_version+1,updated_by_subject=$2,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, in.Outcome, sc.actor, sc.tenant, sc.institution, applicationID, in.ExpectedVersion)
	if err != nil {
		return Decision{}, err
	}
	if tag.RowsAffected() != 1 {
		return Decision{}, errVersionConflict
	}
	return Decision{ID: id, ApplicationID: applicationID, CapacityAllocationID: allocationID, PolicyEvaluationV2ID: policy.EvaluationID, DecisionNo: in.DecisionNo, Outcome: in.Outcome, Rationale: in.Rationale, RankingValue: in.RankingValue, AppealDeadline: in.AppealDeadline, DecidedAt: now.Format(time.RFC3339Nano)}, nil
}

func decisionPolicyContext(c decisionContext, applicationID, outcome string) map[string]any {
	return map[string]any{"authorization_id": c.authorizationID, "campaign_id": c.campaignID, "application_id": applicationID, "outcome": outcome}
}

func (s *Service) EnrolApplication(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "enrolment_failed")
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
	var in EnrolApplicationRequest
	if decode(w, r, &in) != nil || in.ExpectedVersion < 1 || !validDate(in.EnrolledFrom) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_enrolment"})
		return
	}
	in.StudentCode = strings.TrimSpace(in.StudentCode)
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "enrolment_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "enrolment_failed")
		return
	}
	enrolmentID, replay, err := reserve(r.Context(), tx, sc, "admission.application.enrol", key, fingerprint(struct {
		ApplicationID string
		Input         EnrolApplicationRequest
	}{applicationID, in}), uuid.NewString())
	if err != nil {
		writeError(w, err, "enrolment_failed")
		return
	}
	if !replay {
		var status, candidatePartyID, classID, schoolYear string
		var studentID *string
		err = tx.QueryRow(r.Context(), `select a.status,a.candidate_party_id::text,a.student_id::text,x.class_id::text,c.school_year from school_admission_applications a join school_admission_campaigns c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.campaign_id join school_admission_class_offering_contexts x on x.tenant_code=c.tenant_code and x.institution_id=c.institution_id and x.id=c.class_offering_context_id where a.tenant_code=$1 and a.institution_id=$2 and a.id=$3::uuid and a.expected_version=$4 and x.effective_from<=$5::date and (x.effective_to is null or x.effective_to>=$5::date) for update`, sc.tenant, sc.institution, applicationID, in.ExpectedVersion, in.EnrolledFrom).Scan(&status, &candidatePartyID, &studentID, &classID, &schoolYear)
		if errors.Is(err, pgx.ErrNoRows) {
			err = errVersionConflict
		}
		if err == nil && status != "admitted" {
			err = errors.New("only admitted application can be enrolled")
		}
		if err == nil && studentID == nil {
			if !codePattern.MatchString(in.StudentCode) {
				err = errInvalidInput
			} else {
				newID := uuid.NewString()
				var first, last, display string
				err = tx.QueryRow(r.Context(), `select first_name,last_name,display_name from app_parties where tenant_code=$1 and institution_id=$2 and id=$3::uuid and active`, sc.tenant, sc.institution, candidatePartyID).Scan(&first, &last, &display)
				if err == nil {
					if strings.TrimSpace(first) == "" {
						first = display
					}
					if strings.TrimSpace(last) == "" {
						last = "-"
					}
					_, err = tx.Exec(r.Context(), `insert into education_students(id,tenant_code,institution_id,student_code,first_name,last_name,status,party_id) values($1::uuid,$2,$3,$4,$5,$6,'active',$7::uuid)`, newID, sc.tenant, sc.institution, in.StudentCode, first, last, candidatePartyID)
					studentID = &newID
				}
			}
		}
		var allocationID string
		var allocationVersion int
		if err == nil {
			err = tx.QueryRow(r.Context(), `select id::text,expected_version from school_admission_capacity_allocations where tenant_code=$1 and institution_id=$2 and application_id=$3::uuid and status='held' for update`, sc.tenant, sc.institution, applicationID).Scan(&allocationID, &allocationVersion)
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `insert into education_student_enrolments(id,tenant_code,institution_id,student_id,class_id,enrolled_from,status,admission_application_id) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6::date,'active',$7::uuid)`, enrolmentID, sc.tenant, sc.institution, *studentID, classID, in.EnrolledFrom, applicationID)
		}
		if err == nil {
			tag, e := tx.Exec(r.Context(), `update school_admission_capacity_allocations set status='consumed',expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$2 and institution_id=$3 and id=$4::uuid and expected_version=$5`, sc.actor, sc.tenant, sc.institution, allocationID, allocationVersion)
			err = e
			if err == nil && tag.RowsAffected() != 1 {
				err = errVersionConflict
			}
		}
		if err == nil {
			tag, e := tx.Exec(r.Context(), `update school_admission_applications set student_id=$1::uuid,expected_version=expected_version+1,updated_by_subject=$2,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, *studentID, sc.actor, sc.tenant, sc.institution, applicationID, in.ExpectedVersion)
			err = e
			if err == nil && tag.RowsAffected() != 1 {
				err = errVersionConflict
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_application", applicationID, "admission.application.enrolled", map[string]any{"applicationId": applicationID, "enrolmentId": enrolmentID, "schoolYear": schoolYear})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.application.enrolled", "education_student_enrolment", enrolmentID)
		}
	}
	if err != nil {
		writeError(w, err, "enrolment_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "enrolment_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: enrolmentID, Status: "active", Replayed: replay})
}
