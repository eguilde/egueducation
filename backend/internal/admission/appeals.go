package admission

import (
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// appealResolutionAllocationID retains the original decision's allocation for
// a non-favourable resolution. A favourable replacement uses only the
// allocation created for that replacement; carrying an admitted allocation to
// a non-admitted replacement would violate the resolution aggregate contract.
func appealResolutionAllocationID(originalAllocationID string, favourable bool, resulting Decision) string {
	if !favourable {
		return originalAllocationID
	}
	return optional(resulting.CapacityAllocationID)
}

func (s *Service) CreateAppeal(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "appeal_create_failed")
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
	var in CreateAppealRequest
	if decode(w, r, &in) != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_appeal"})
		return
	}
	in.AppealNo = strings.TrimSpace(in.AppealNo)
	in.Statement = strings.TrimSpace(in.Statement)
	if !validUUID(in.DecisionID) || !validUUID(in.SubmittedByPartyID) || !codePattern.MatchString(in.AppealNo) || in.Statement == "" || (in.Archive != nil && (!validUUID(in.Archive.DocumentID) || !validUUID(in.Archive.VersionID))) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_appeal"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionAppeals); err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	if err = requirePermission(r.Context(), tx, sc, "earchiva.read"); err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.appeal.submit", key, fingerprint(struct {
		ApplicationID string
		Input         CreateAppealRequest
	}{applicationID, in}), uuid.NewString())
	if err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	if !replay {
		var late bool
		var authorizedAppellant bool
		err = tx.QueryRow(r.Context(), `select appeal_deadline is not null and current_date>appeal_deadline,exists(select 1 from school_admission_applications application where application.tenant_code=$1 and application.institution_id=$2 and application.id=$4::uuid and (application.candidate_party_id=$5::uuid or exists(select 1 from school_admission_application_representatives representative where representative.tenant_code=application.tenant_code and representative.institution_id=application.institution_id and representative.application_id=application.id and representative.representative_party_id=$5::uuid))) from school_admission_decisions where tenant_code=$1 and institution_id=$2 and id=$3::uuid and application_id=$4::uuid for share`, sc.tenant, sc.institution, in.DecisionID, applicationID, in.SubmittedByPartyID).Scan(&late, &authorizedAppellant)
		if err == nil && late {
			err = errInvalidState
		}
		if err == nil && !authorizedAppellant {
			err = errInvalidInput
		}
		if err == nil {
			_, err = tx.Exec(r.Context(), `insert into school_admission_appeals(id,tenant_code,institution_id,application_id,decision_id,appeal_no,status,submitted_at,submitted_by_party_id,created_by_subject,updated_by_subject) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6,'submitted',now(),$7::uuid,$8,$8)`, id, sc.tenant, sc.institution, applicationID, in.DecisionID, in.AppealNo, in.SubmittedByPartyID, sc.actor)
		}
		if err == nil && in.Archive == nil {
			_, err = tx.Exec(r.Context(), `insert into school_admission_appeal_submissions(tenant_code,institution_id,appeal_id,submission_no,submitted_by_party_id,statement,created_by_subject) values($1,$2,$3::uuid,1,$4::uuid,$5,$6)`, sc.tenant, sc.institution, id, in.SubmittedByPartyID, in.Statement, sc.actor)
		}
		if err == nil && in.Archive != nil {
			var a archiveSnapshot
			a, err = loadArchiveSnapshot(r.Context(), tx, sc.institution, *in.Archive)
			if err == nil && !strings.EqualFold(a.mime, "application/pdf") {
				err = errInvalidInput
			}
			if err == nil {
				_, err = tx.Exec(r.Context(), `insert into school_admission_appeal_submissions(tenant_code,institution_id,appeal_id,submission_no,submitted_by_party_id,statement,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256,created_by_subject) values($1,$2,$3::uuid,1,$4::uuid,$5,$6::uuid,$7::uuid,$8,$9,$10,$11,$12,$13,$14)`, sc.tenant, sc.institution, id, in.SubmittedByPartyID, in.Statement, in.Archive.DocumentID, in.Archive.VersionID, a.versionNo, a.bucket, a.objectKey, a.objectVersion, a.retention, a.sha, sc.actor)
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_appeal", id, "admission.appeal.submitted", map[string]any{"appeal_id": id, "application_id": applicationID})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.appeal.submitted", "school_admission_appeal", id)
		}
	}
	if err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "appeal_create_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "submitted", ExpectedVersion: 1, Replayed: replay})
}

func (s *Service) ResolveAppeal(w http.ResponseWriter, r *http.Request) {
	// Appeal resolution is a legal act; direct resolution is intentionally
	// closed until its preparation carries the independently signed artifacts.
	httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_legal_preparation_required"})
	return
	/*
		sc, ok := requestScope(r)
		if !ok {
			writeError(w, errBadScope, "appeal_resolution_failed")
			return
		}
		appealID := chi.URLParam(r, "appealID")
		if !validUUID(appealID) {
			httpx.JSON(w, 422, map[string]any{"code": "invalid_appeal_id"})
			return
		}
		key, ok := idempotencyKey(r)
		if !ok {
			httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
			return
		}
		var in ResolveAppealRequest
		if decode(w, r, &in) != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_appeal_resolution"})
			return
		}
		in.Outcome = strings.TrimSpace(in.Outcome)
		in.ResultingOutcome = strings.TrimSpace(in.ResultingOutcome)
		in.Rationale = strings.TrimSpace(in.Rationale)
		favourable := in.Outcome == "upheld" || in.Outcome == "partially_upheld"
		validResult := in.ResultingOutcome == "admitted" || in.ResultingOutcome == "waitlisted" || in.ResultingOutcome == "rejected" || in.ResultingOutcome == "withdrawn" || in.ResultingOutcome == "cancelled"
		if in.ExpectedVersion < 1 || in.ApplicationExpectedVersion < 1 || in.Rationale == "" || !validUUID(in.Archive.DocumentID) || !validUUID(in.Archive.VersionID) || (in.Outcome != "upheld" && in.Outcome != "partially_upheld" && in.Outcome != "dismissed" && in.Outcome != "withdrawn") || (favourable && (!validResult || !codePattern.MatchString(in.ResultingDecisionNo))) {
			httpx.JSON(w, 422, map[string]any{"code": "invalid_appeal_resolution"})
			return
		}
		tx, err := s.begin(r.Context(), sc)
		if err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		defer tx.Rollback(r.Context())
		if err = requirePermission(r.Context(), tx, sc, permissionAppeals); err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		if err = requirePermission(r.Context(), tx, sc, "earchiva.read"); err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		if favourable {
			if err = requirePermission(r.Context(), tx, sc, permissionDecide); err != nil {
				writeError(w, err, "appeal_resolution_failed")
				return
			}
		}
		resolutionID, replay, err := reserve(r.Context(), tx, sc, "admission.appeal.resolve", key, fingerprint(struct {
			AppealID string
			Input    ResolveAppealRequest
		}{appealID, in}), uuid.NewString())
		if err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		if !replay {
			var applicationID, originalDecisionID, appealStatus, originalOutcome, originalAllocationID string
			err = tx.QueryRow(r.Context(), `select appeal.application_id::text,appeal.decision_id::text,appeal.status,decision.outcome,coalesce(decision.capacity_allocation_id::text,'') from school_admission_appeals appeal join school_admission_decisions decision on decision.tenant_code=appeal.tenant_code and decision.institution_id=appeal.institution_id and decision.id=appeal.decision_id where appeal.tenant_code=$1 and appeal.institution_id=$2 and appeal.id=$3::uuid and appeal.expected_version=$4 for update`, sc.tenant, sc.institution, appealID, in.ExpectedVersion).Scan(&applicationID, &originalDecisionID, &appealStatus, &originalOutcome, &originalAllocationID)
			if errors.Is(err, pgx.ErrNoRows) {
				err = errVersionConflict
			}
			if err == nil && appealStatus != "submitted" && appealStatus != "under_review" {
				err = errInvalidState
			}
			var c decisionContext
			if err == nil {
				c, err = loadDecisionContext(r.Context(), tx, sc, applicationID, in.ApplicationExpectedVersion)
			}
			if err == nil && favourable && in.ResultingOutcome != "admitted" {
				var activelyEnrolled bool
				err = tx.QueryRow(r.Context(), `select exists(select 1 from education_student_enrolments enrolment where enrolment.tenant_code=$1 and enrolment.institution_id=$2 and enrolment.admission_application_id=$3::uuid and enrolment.status='active')`, sc.tenant, sc.institution, applicationID).Scan(&activelyEnrolled)
				if err == nil && activelyEnrolled {
					err = errInvalidState
				}
			}
			now := time.Now().UTC()
			var policy institution.OperationPolicyV2Decision
			if err == nil {
				policy, err = institution.EvaluateOperationPolicyV2Tx(r.Context(), tx, institution.OperationPolicyV2Request{OperationCode: "admission.appeal.resolve", EffectiveOn: now, DecisionKind: "operation", OfferingID: c.offeringID, LocationID: c.locationID, ActorSubject: sc.actor, Context: map[string]any{"authorization_id": c.authorizationID, "campaign_id": c.campaignID, "application_id": applicationID, "appeal_id": appealID, "outcome": in.Outcome}})
			}
			if err == nil && !policy.Allowed {
				err = errPolicyDenied
			}
			resultingOutcome := originalOutcome
			resultingID, allocationID := "", originalAllocationID
			releasedAllocationID := ""
			if err == nil && favourable {
				resultingID = uuid.NewString()
				resultingOutcome = in.ResultingOutcome
				decisionIn := IssueDecisionRequest{DecisionNo: in.ResultingDecisionNo, Outcome: resultingOutcome, Rationale: in.Rationale, ExpectedVersion: in.ApplicationExpectedVersion, Archive: in.Archive}
				var decision Decision
				decision, err = s.issueDecisionTx(r.Context(), tx, sc, resultingID, applicationID, decisionIn, originalDecisionID)
				if err == nil {
					allocationID = appealResolutionAllocationID(originalAllocationID, true, decision)
				}
			}
			if err == nil && favourable && resultingOutcome != "admitted" && originalAllocationID != "" {
				tag, e := tx.Exec(r.Context(), `update school_admission_capacity_allocations set status='released',expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$2 and institution_id=$3 and id=$4::uuid and application_id=$5::uuid and status='held'`, sc.actor, sc.tenant, sc.institution, originalAllocationID, applicationID)
				err = e
				if err == nil && tag.RowsAffected() == 1 {
					releasedAllocationID = originalAllocationID
				}
			}
			var a archiveSnapshot
			if err == nil {
				a, err = loadArchiveSnapshot(r.Context(), tx, sc.institution, in.Archive)
				if err == nil && !strings.EqualFold(a.mime, "application/pdf") {
					err = errInvalidInput
				}
			}
			if err == nil {
				snapshot, _ := json.Marshal(map[string]any{"appeal_id": appealID, "application_id": applicationID, "outcome": in.Outcome, "resulting_outcome": resultingOutcome, "original_capacity_allocation_id": originalAllocationID, "released_capacity_allocation_id": releasedAllocationID, "policy_evaluation_v2_id": policy.EvaluationID})
				_, err = tx.Exec(r.Context(), `insert into school_admission_appeal_resolutions(id,tenant_code,institution_id,appeal_id,resulting_decision_id,resulting_outcome,capacity_allocation_id,policy_evaluation_v2_id,outcome,rationale,resolved_at,resolved_by_subject,resolution_snapshot,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256) values($1::uuid,$2,$3,$4::uuid,nullif($5,'')::uuid,$6,nullif($7,'')::uuid,$8::uuid,$9,$10,$11,$12,$13::jsonb,$14::uuid,$15::uuid,$16,$17,$18,$19,$20,$21)`, resolutionID, sc.tenant, sc.institution, appealID, resultingID, resultingOutcome, allocationID, policy.EvaluationID, in.Outcome, in.Rationale, now, sc.actor, snapshot, in.Archive.DocumentID, in.Archive.VersionID, a.versionNo, a.bucket, a.objectKey, a.objectVersion, a.retention, a.sha)
			}
			if err == nil {
				err = persistDSSBinding(r.Context(), tx, sc, "admission_appeal_resolution", resolutionID, policy.EvaluationID, in.Archive, a, sc.actor)
			}
			if err == nil {
				tag, e := tx.Exec(r.Context(), `update school_admission_appeals set status='resolved',expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$2 and institution_id=$3 and id=$4::uuid and expected_version=$5`, sc.actor, sc.tenant, sc.institution, appealID, in.ExpectedVersion)
				err = e
				if err == nil && tag.RowsAffected() != 1 {
					err = errVersionConflict
				}
			}
			if err == nil {
				err = outbox(r.Context(), tx, sc, "admission_appeal", appealID, "admission.appeal.resolved", map[string]any{"appeal_id": appealID, "resolution_id": resolutionID, "outcome": in.Outcome, "resulting_decision_id": resultingID})
			}
			if err == nil {
				err = auditEvent(r.Context(), tx, sc, "admission.appeal.resolved", "school_admission_appeal_resolution", resolutionID)
			}
		}
		if err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeError(w, err, "appeal_resolution_failed")
			return
		}
		httpx.JSON(w, 201, CommandResult{ID: resolutionID, Status: "resolved", Replayed: replay})
	*/
}
