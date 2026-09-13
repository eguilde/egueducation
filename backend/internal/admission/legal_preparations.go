package admission

// The prepare/finalize flow deliberately keeps the signed document out of the
// preparation request.  The browser receives only a canonical, server-built
// payload to place in the signing ceremony; archive provenance is accepted at
// finalization and resolved again from eArhiva.

import (
	"context"
	"encoding/base64"
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

const legalPreparationTTL = 15 * time.Minute

// expireAdmissionPreparationsTx is deliberately tenant/institution scoped and
// runs only inside a request transaction already bound to the caller's scope.
// It cannot release another tenant's capacity; each preparation and its hold
// are locked before changing either lifecycle state.
func expireAdmissionPreparationsTx(ctx context.Context, tx pgx.Tx, sc scope) error {
	rows, err := tx.Query(ctx, `update school_admission_legal_preparations p set status='expired',cancellation_reason='expired' where p.tenant_code=$1 and p.institution_id=$2 and p.status='prepared' and p.expires_at<=now() returning p.capacity_allocation_id::text`, sc.tenant, sc.institution)
	if err != nil {
		return err
	}
	allocationIDs := make([]string, 0)
	for rows.Next() {
		var allocationID *string
		if err = rows.Scan(&allocationID); err != nil {
			rows.Close()
			return err
		}
		if allocationID != nil {
			allocationIDs = append(allocationIDs, *allocationID)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, allocationID := range allocationIDs {
		if _, err = tx.Exec(ctx, `update school_admission_capacity_allocations set status='released',expected_version=expected_version+1,updated_at=now(),updated_by_subject=$1 where tenant_code=$2 and institution_id=$3 and id=$4::uuid and status='held'`, sc.actor, sc.tenant, sc.institution, allocationID); err != nil {
			return err
		}
	}
	return nil
}

func validPrepareDecision(in PrepareDecisionRequest) bool {
	return validDecision(IssueDecisionRequest{DecisionNo: in.DecisionNo, Outcome: in.Outcome, Rationale: in.Rationale, RankingValue: in.RankingValue, AppealDeadline: in.AppealDeadline, ExpectedVersion: in.ExpectedVersion, Archive: ArchiveReference{DocumentID: uuid.NewString(), VersionID: uuid.NewString()}})
}

func preparationResponse(row AdmissionLegalPreparation) AdmissionLegalPreparation {
	row.CanonicalPayloadBase64 = base64.RawStdEncoding.EncodeToString(row.CanonicalPayload)
	if len(row.ResultingDecisionPayload) > 0 {
		row.ResultingDecisionPayloadBase64 = base64.RawStdEncoding.EncodeToString(row.ResultingDecisionPayload)
	}
	return row
}

func (s *Service) PrepareDecision(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "decision_prepare_failed")
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
	var in PrepareDecisionRequest
	if decode(w, r, &in) != nil || !validPrepareDecision(in) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_decision_preparation"})
		return
	}
	in.DecisionNo = strings.TrimSpace(in.DecisionNo)
	in.Outcome = strings.TrimSpace(in.Outcome)
	in.Rationale = strings.TrimSpace(in.Rationale)
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionDecide); err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	if err = expireAdmissionPreparationsTx(r.Context(), tx, sc); err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	preparationID, replay, err := reserve(r.Context(), tx, sc, "admission.decision.prepare", key, fingerprint(struct {
		ApplicationID string
		Input         PrepareDecisionRequest
	}{applicationID, in}), uuid.NewString())
	if err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	var out AdmissionLegalPreparation
	if replay {
		err = loadPreparation(r.Context(), tx, sc, preparationID, &out)
		out.Replayed = true
	} else {
		err = s.prepareDecisionTx(r.Context(), tx, sc, preparationID, applicationID, in, &out)
	}
	if err == nil && !replay {
		err = auditEvent(r.Context(), tx, sc, "admission.decision.prepared", "school_admission_legal_preparation", preparationID)
	}
	if err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "decision_prepare_failed")
		return
	}
	httpx.JSON(w, 201, preparationResponse(out))
}

func (s *Service) prepareDecisionTx(ctx context.Context, tx pgx.Tx, sc scope, preparationID, applicationID string, in PrepareDecisionRequest, out *AdmissionLegalPreparation) error {
	c, err := loadDecisionContext(ctx, tx, sc, applicationID, in.ExpectedVersion)
	if err != nil {
		return err
	}
	if c.status != "submitted" && c.status != "under_review" {
		return errInvalidState
	}
	if c.campaignStatus != "open" && c.campaignStatus != "closed" {
		return errInvalidState
	}
	if in.Outcome == "admitted" {
		var incomplete bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from school_admission_document_requirements requirement where requirement.tenant_code=$1 and requirement.institution_id=$2 and requirement.campaign_id=$3::uuid and requirement.required and not exists(select 1 from school_admission_application_documents d where d.tenant_code=requirement.tenant_code and d.institution_id=requirement.institution_id and d.application_id=$4::uuid and d.document_requirement_id=requirement.id and d.status in ('accepted','waived')))`, sc.tenant, sc.institution, c.campaignID, applicationID).Scan(&incomplete); err != nil {
			return err
		}
		if incomplete {
			return errInvalidInput
		}
	}
	now := time.Now().UTC()
	policy, err := institution.EvaluateOperationPolicyV2Tx(ctx, tx, institution.OperationPolicyV2Request{OperationCode: "admission.decision.issue", EffectiveOn: now, DecisionKind: "operation", OfferingID: c.offeringID, LocationID: c.locationID, ActorSubject: sc.actor, Context: decisionPolicyContext(c, applicationID, in.Outcome)})
	if err != nil {
		return err
	}
	if !policy.Allowed {
		return errPolicyDenied
	}
	decisionID := uuid.NewString()
	allocationID := ""
	if in.Outcome == "admitted" {
		allocationID = uuid.NewString()
		_, err = tx.Exec(ctx, `insert into school_admission_capacity_allocations(id,tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,status,created_by_subject,updated_by_subject) select $1::uuid,$2,$3,$4::uuid,c.id,c.class_offering_context_id,c.authorization_id,c.capacity_unit,c.shift,'held',$5,$5 from school_admission_campaigns c where c.tenant_code=$2 and c.institution_id=$3 and c.id=$6::uuid`, allocationID, sc.tenant, sc.institution, applicationID, sc.actor, c.campaignID)
		if err != nil {
			return err
		}
	}
	payload, err := BuildAdmissionDecisionLegalPayload(AdmissionDecisionLegalPayloadInput{TenantCode: sc.tenant, InstitutionID: sc.institution, ArtifactID: decisionID, ApplicationID: applicationID, CampaignID: c.campaignID, OfferingID: c.offeringID, LocationID: c.locationID, AuthorizationID: c.authorizationID, ClassOfferingContextID: c.classContextID, DecisionNo: in.DecisionNo, Outcome: in.Outcome, Rationale: in.Rationale, RankingValue: in.RankingValue, AppealDeadline: in.AppealDeadline, PolicyEvaluationV2ID: policy.EvaluationID, ActorSubject: sc.actor, DecidedAt: now, CapacityAllocationID: allocationID})
	if err != nil {
		return err
	}
	raw, err := payload.CanonicalJSON()
	if err != nil {
		return err
	}
	digest, err := AdmissionDecisionLegalPayloadSHA256(payload)
	if err != nil {
		return err
	}
	snapshot, _ := json.Marshal(map[string]any{"decision_no": in.DecisionNo, "outcome": in.Outcome, "rationale": in.Rationale, "ranking_value": in.RankingValue, "appeal_deadline": in.AppealDeadline, "campaign_id": c.campaignID, "offering_id": c.offeringID, "location_id": c.locationID, "authorization_id": c.authorizationID, "class_offering_context_id": c.classContextID, "decided_at": now.Format(time.RFC3339Nano)})
	expires := now.Add(legalPreparationTTL)
	retention, err := snapshotPreparationRetention(ctx, tx, sc, expires)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,capacity_allocation_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) values($1::uuid,$2,$3,'admission_decision',$4::uuid,$5::uuid,nullif($6,'')::uuid,$7::uuid,$8,$9::jsonb,$10,$11,$12::jsonb,$13,$14,$15::uuid,$16::uuid,$17::uuid,$18,$19,$20) returning id::text,artifact_kind,artifact_id::text,application_id::text,policy_evaluation_v2_id::text,canonical_payload_bytes,canonical_payload_sha256,prepared_by_subject,to_char(prepared_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(expires_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),status,retention_policy_id::text,retention_rule_version_id::text,retention_source_id::text,to_char(retention_anchor_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),minimum_retention_days,to_char(required_retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, preparationID, sc.tenant, sc.institution, decisionID, applicationID, allocationID, policy.EvaluationID, in.ExpectedVersion, string(raw), string(raw), digest, string(snapshot), sc.actor, expires, retention.PolicyID, retention.RuleID, retention.SourceID, retention.Anchor, retention.MinimumDays, retention.RequiredUntil).Scan(&out.ID, &out.ArtifactKind, &out.ArtifactID, &out.ApplicationID, &out.PolicyEvaluationV2ID, &out.CanonicalPayload, &out.CanonicalPayloadSHA256, &out.PreparedBySubject, &out.PreparedAt, &out.ExpiresAt, &out.Status, &out.RetentionPolicyID, &out.RetentionRuleVersionID, &out.RetentionSourceID, &out.RetentionAnchorAt, &out.MinimumRetentionDays, &out.RequiredRetentionUntil)
	if allocationID != "" {
		out.ResultingDecisionID = &decisionID
	}
	return err
}

func loadPreparation(ctx context.Context, tx pgx.Tx, sc scope, id string, out *AdmissionLegalPreparation) error {
	var appeal, resulting *string
	err := tx.QueryRow(ctx, `select id::text,artifact_kind,artifact_id::text,application_id::text,appeal_id::text,resulting_decision_id::text,policy_evaluation_v2_id::text,canonical_payload_bytes,canonical_payload_sha256,resulting_decision_payload_bytes,resulting_decision_payload_sha256,prepared_by_subject,to_char(prepared_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(expires_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),status,coalesce(retention_policy_id::text,''),coalesce(retention_rule_version_id::text,''),coalesce(retention_source_id::text,''),coalesce(to_char(retention_anchor_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),coalesce(minimum_retention_days,0),coalesce(to_char(required_retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'') from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&out.ID, &out.ArtifactKind, &out.ArtifactID, &out.ApplicationID, &appeal, &resulting, &out.PolicyEvaluationV2ID, &out.CanonicalPayload, &out.CanonicalPayloadSHA256, &out.ResultingDecisionPayload, &out.ResultingDecisionPayloadSHA256, &out.PreparedBySubject, &out.PreparedAt, &out.ExpiresAt, &out.Status, &out.RetentionPolicyID, &out.RetentionRuleVersionID, &out.RetentionSourceID, &out.RetentionAnchorAt, &out.MinimumRetentionDays, &out.RequiredRetentionUntil)
	out.AppealID, out.ResultingDecisionID = appeal, resulting
	return err
}

func (s *Service) FinalizeAdmissionLegalPreparation(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "legal_finalize_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in FinalizeAdmissionLegalPreparationRequest
	if decode(w, r, &in) != nil || !validUUID(in.PreparationID) || !validUUID(in.Archive.DocumentID) || !validUUID(in.Archive.VersionID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_legal_finalization"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	defer tx.Rollback(r.Context())
	var artifactKind string
	if err = tx.QueryRow(r.Context(), `select artifact_kind from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, in.PreparationID).Scan(&artifactKind); err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	if artifactKind == "admission_decision" {
		err = requirePermission(r.Context(), tx, sc, permissionDecide)
	} else if artifactKind == "admission_appeal_resolution" {
		err = requirePermission(r.Context(), tx, sc, permissionAppeals)
	} else {
		err = errInvalidState
	}
	if err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	if err = expireAdmissionPreparationsTx(r.Context(), tx, sc); err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.legal.finalize", key, fingerprint(in), in.PreparationID)
	if err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	var out CommandResult
	if replay {
		out = CommandResult{ID: id, Status: "finalized", Replayed: true}
	} else {
		err = s.finalizeDecisionTx(r.Context(), tx, sc, in)
		out = CommandResult{ID: id, Status: "finalized"}
	}
	if err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "legal_finalize_failed")
		return
	}
	httpx.JSON(w, 201, out)
}

func (s *Service) finalizeDecisionTx(ctx context.Context, tx pgx.Tx, sc scope, in FinalizeAdmissionLegalPreparationRequest) error {
	var p AdmissionLegalPreparation
	var allocationID *string
	var expectedVersion int
	var snapshot json.RawMessage
	err := tx.QueryRow(ctx, `select id::text,artifact_kind,artifact_id::text,application_id::text,capacity_allocation_id::text,policy_evaluation_v2_id::text,aggregate_expected_version,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,status,prepared_by_subject,to_char(expires_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),coalesce(retention_policy_id::text,''),coalesce(retention_rule_version_id::text,''),coalesce(retention_source_id::text,''),coalesce(to_char(retention_anchor_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),''),coalesce(minimum_retention_days,0),coalesce(to_char(required_retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'') from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, sc.tenant, sc.institution, in.PreparationID).Scan(&p.ID, &p.ArtifactKind, &p.ArtifactID, &p.ApplicationID, &allocationID, &p.PolicyEvaluationV2ID, &expectedVersion, &p.CanonicalPayload, &p.CanonicalPayloadSHA256, &snapshot, &p.Status, &p.PreparedBySubject, &p.ExpiresAt, &p.RetentionPolicyID, &p.RetentionRuleVersionID, &p.RetentionSourceID, &p.RetentionAnchorAt, &p.MinimumRetentionDays, &p.RequiredRetentionUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err != nil {
		return err
	}
	if p.ArtifactKind != "admission_decision" || p.Status != "prepared" || p.PreparedBySubject != sc.actor || time.Now().UTC().After(mustParseTimestamp(p.ExpiresAt)) {
		if p.ArtifactKind == "admission_appeal_resolution" && p.Status == "prepared" && p.PreparedBySubject == sc.actor && !time.Now().UTC().After(mustParseTimestamp(p.ExpiresAt)) {
			return s.finalizeAppealPreparedTx(ctx, tx, sc, p, allocationID, expectedVersion, snapshot, in)
		}
		return errInvalidState
	}
	var facts struct {
		DecisionNo             string   `json:"decision_no"`
		Outcome                string   `json:"outcome"`
		Rationale              string   `json:"rationale"`
		CampaignID             string   `json:"campaign_id"`
		OfferingID             string   `json:"offering_id"`
		LocationID             string   `json:"location_id"`
		AuthorizationID        string   `json:"authorization_id"`
		ClassOfferingContextID string   `json:"class_offering_context_id"`
		DecidedAt              string   `json:"decided_at"`
		RankingValue           *float64 `json:"ranking_value"`
		AppealDeadline         *string  `json:"appeal_deadline"`
	}
	if json.Unmarshal(snapshot, &facts) != nil {
		return errInvalidInput
	}
	c, err := loadDecisionContext(ctx, tx, sc, p.ApplicationID, expectedVersion)
	if err != nil {
		return err
	}
	if c.campaignID != facts.CampaignID || c.status != "submitted" && c.status != "under_review" {
		return errInvalidState
	}
	archive, err := loadArchiveSnapshot(ctx, tx, sc.institution, in.Archive)
	if err != nil {
		return err
	}
	if err = requirePreparationArtifactIntent(ctx, tx, sc, p.ID, "primary", in.Archive); err != nil {
		return err
	}
	if !strings.EqualFold(archive.mime, "application/pdf") {
		return errInvalidInput
	}
	if err = validatePreparationRetention(ctx, tx, sc, preparationRetentionFromModel(p), archive.retention); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into school_admission_decisions(id,tenant_code,institution_id,application_id,capacity_allocation_id,policy_evaluation_v2_id,decision_no,outcome,rationale,ranking_value,appeal_deadline,decided_at,decided_by_subject,decision_snapshot,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11::date,$12::timestamptz,$13,$14::jsonb,$15::uuid,$16::uuid,$17,$18,$19,$20,$21,$22)`, p.ArtifactID, sc.tenant, sc.institution, p.ApplicationID, allocationID, p.PolicyEvaluationV2ID, facts.DecisionNo, facts.Outcome, facts.Rationale, facts.RankingValue, optional(facts.AppealDeadline), facts.DecidedAt, sc.actor, string(snapshot), in.Archive.DocumentID, in.Archive.VersionID, archive.versionNo, archive.bucket, archive.objectKey, archive.objectVersion, archive.retention, archive.sha)
	if err != nil {
		return err
	}
	if err = persistDSSBinding(ctx, tx, sc, "admission_decision", p.ArtifactID, p.PolicyEvaluationV2ID, in.Archive, archive, sc.actor, p.CanonicalPayloadSHA256); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `update school_admission_applications set status=$1,expected_version=expected_version+1,updated_at=now(),updated_by_subject=$2 where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, facts.Outcome, sc.actor, sc.tenant, sc.institution, p.ApplicationID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errVersionConflict
	}
	_, err = tx.Exec(ctx, `update school_admission_legal_preparations set status='finalized' where tenant_code=$1 and institution_id=$2 and id=$3::uuid and status='prepared'`, sc.tenant, sc.institution, p.ID)
	if err != nil {
		return err
	}
	if err = outbox(ctx, tx, sc, "admission_application", p.ApplicationID, "admission.decision.finalized", map[string]any{"decisionId": p.ArtifactID, "preparationId": p.ID}); err != nil {
		return err
	}
	return auditEvent(ctx, tx, sc, "admission.decision.finalized", "school_admission_decision", p.ArtifactID)
}

func mustParseTimestamp(v string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999-07", "2006-01-02 15:04:05-07"} {
		if parsed, err := time.Parse(layout, v); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

// requirePreparationArtifactIntent prevents a legal finalizer from binding an
// arbitrary tenant WORM document. The exact document/version must be the
// committed, preparation-bound immutable ingestion intent for that slot.
func requirePreparationArtifactIntent(ctx context.Context, tx pgx.Tx, sc scope, preparationID, slot string, ref ArchiveReference) error {
	var ok bool
	err := tx.QueryRow(ctx, `select exists(select 1 from archive_ingestion_intents i join archive_document_versions v on v.ingestion_intent_id=i.id where i.tenant_code=$1 and i.institution_id=$2 and i.preparation_id=$3::uuid and i.artifact_slot=$4 and i.purpose='admission_legal_preparation' and i.status='committed' and i.reserved_document_id=$5::uuid and i.reserved_version_id=$6::uuid and v.document_id=i.reserved_document_id and v.id=i.reserved_version_id and v.source_object_version_id=i.stored_version_id and v.source_object_etag=i.stored_etag and lower(v.source_sha256)=lower(i.expected_sha256) and v.source_size_bytes=i.expected_size_bytes and v.retention_until=i.stored_retention_until)`, sc.tenant, sc.institution, preparationID, slot, ref.DocumentID, ref.VersionID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errInvalidState
	}
	return nil
}

func (s *Service) CancelAdmissionLegalPreparation(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "legal_cancel_failed")
		return
	}
	id := chi.URLParam(r, "preparationID")
	if !validUUID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_preparation_id"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "legal_cancel_failed")
		return
	}
	defer tx.Rollback(r.Context())
	var artifactKind string
	if err = tx.QueryRow(r.Context(), `select artifact_kind from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid and prepared_by_subject=$4`, sc.tenant, sc.institution, id, sc.actor).Scan(&artifactKind); err != nil {
		writeError(w, err, "legal_cancel_failed")
		return
	}
	if artifactKind == "admission_decision" {
		err = requirePermission(r.Context(), tx, sc, permissionDecide)
	} else if artifactKind == "admission_appeal_resolution" {
		err = requirePermission(r.Context(), tx, sc, permissionAppeals)
	} else {
		err = errInvalidState
	}
	if err != nil {
		writeError(w, err, "legal_cancel_failed")
		return
	}
	_, replay, err := reserve(r.Context(), tx, sc, "admission.legal.cancel", key, fingerprint(id), id)
	if err == nil && !replay {
		var allocationID *string
		err = tx.QueryRow(r.Context(), `update school_admission_legal_preparations p set status='cancelled',cancellation_reason='cancelled_by_preparer' where p.tenant_code=$1 and p.institution_id=$2 and p.id=$3::uuid and p.status='prepared' and p.prepared_by_subject=$4 returning capacity_allocation_id::text`, sc.tenant, sc.institution, id, sc.actor).Scan(&allocationID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = errInvalidState
		}
		if err == nil && allocationID != nil {
			_, err = tx.Exec(r.Context(), `update school_admission_capacity_allocations set status='released',expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$2 and institution_id=$3 and id=$4::uuid and status='held'`, sc.actor, sc.tenant, sc.institution, *allocationID)
		}
	}
	if err != nil {
		writeError(w, err, "legal_cancel_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "legal_cancel_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "cancelled", Replayed: replay})
}

// PrepareAppealResolution reserves the full resolution aggregate and, for a
// favourable appeal, a distinct replacement-decision legal artifact.
func (s *Service) PrepareAppealResolution(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "appeal_prepare_failed")
		return
	}
	appealID := chi.URLParam(r, "appealID")
	key, ok := idempotencyKey(r)
	if !validUUID(appealID) || !ok {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_appeal_preparation"})
		return
	}
	var in PrepareAppealResolutionRequest
	if decode(w, r, &in) != nil || !validPrepareAppeal(in) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_appeal_preparation"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionAppeals); err == nil && (in.Outcome == "upheld" || in.Outcome == "partially_upheld") {
		err = requirePermission(r.Context(), tx, sc, permissionDecide)
	}
	if err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	if err = expireAdmissionPreparationsTx(r.Context(), tx, sc); err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.appeal.prepare", key, fingerprint(struct {
		AppealID string
		In       PrepareAppealResolutionRequest
	}{appealID, in}), uuid.NewString())
	if err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	var out AdmissionLegalPreparation
	if replay {
		err = loadPreparation(r.Context(), tx, sc, id, &out)
		out.Replayed = true
	} else {
		err = s.prepareAppealTx(r.Context(), tx, sc, id, appealID, in, &out)
	}
	if err == nil && !replay {
		err = auditEvent(r.Context(), tx, sc, "admission.appeal.prepared", "school_admission_legal_preparation", id)
	}
	if err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "appeal_prepare_failed")
		return
	}
	httpx.JSON(w, 201, preparationResponse(out))
}

func validPrepareAppeal(in PrepareAppealResolutionRequest) bool {
	favourable := in.Outcome == "upheld" || in.Outcome == "partially_upheld"
	result := in.ResultingOutcome == "admitted" || in.ResultingOutcome == "waitlisted" || in.ResultingOutcome == "rejected" || in.ResultingOutcome == "withdrawn" || in.ResultingOutcome == "cancelled"
	return in.ExpectedVersion > 0 && in.ApplicationExpectedVersion > 0 && strings.TrimSpace(in.Rationale) != "" && (favourable && result && codePattern.MatchString(strings.TrimSpace(in.ResultingDecisionNo)) || (!favourable && (in.Outcome == "dismissed" || in.Outcome == "withdrawn")))
}

func (s *Service) prepareAppealTx(ctx context.Context, tx pgx.Tx, sc scope, preparationID, appealID string, in PrepareAppealResolutionRequest, out *AdmissionLegalPreparation) error {
	var applicationID, originalDecisionID, originalOutcome, originalAllocationID, appealStatus string
	err := tx.QueryRow(ctx, `select a.application_id::text,a.decision_id::text,a.status,d.outcome,coalesce(d.capacity_allocation_id::text,'') from school_admission_appeals a join school_admission_decisions d on d.tenant_code=a.tenant_code and d.institution_id=a.institution_id and d.id=a.decision_id where a.tenant_code=$1 and a.institution_id=$2 and a.id=$3::uuid and a.expected_version=$4 for update`, sc.tenant, sc.institution, appealID, in.ExpectedVersion).Scan(&applicationID, &originalDecisionID, &appealStatus, &originalOutcome, &originalAllocationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errVersionConflict
	}
	if err != nil {
		return err
	}
	if appealStatus != "submitted" && appealStatus != "under_review" {
		return errInvalidState
	}
	c, err := loadDecisionContext(ctx, tx, sc, applicationID, in.ApplicationExpectedVersion)
	if err != nil {
		return err
	}
	favourable := in.Outcome == "upheld" || in.Outcome == "partially_upheld"
	if favourable && in.ResultingOutcome == "admitted" {
		var incomplete bool
		err = tx.QueryRow(ctx, `select exists(
 select 1 from school_admission_document_requirements requirement
 where requirement.tenant_code=$1 and requirement.institution_id=$2 and requirement.campaign_id=$3::uuid and requirement.required
 and not exists(select 1 from school_admission_application_documents d where d.tenant_code=requirement.tenant_code and d.institution_id=requirement.institution_id and d.application_id=$4::uuid and d.document_requirement_id=requirement.id and d.status in ('accepted','waived'))
 union all
 select 1 from school_admission_criteria criterion where criterion.tenant_code=$1 and criterion.institution_id=$2 and criterion.campaign_id=$3::uuid and criterion.required
 and not exists(select 1 from school_admission_criterion_assessments assessment where assessment.tenant_code=criterion.tenant_code and assessment.institution_id=criterion.institution_id and assessment.application_id=$4::uuid and assessment.criterion_id=criterion.id and assessment.outcome='met')
)`, sc.tenant, sc.institution, c.campaignID, applicationID).Scan(&incomplete)
		if err != nil {
			return err
		}
		if incomplete {
			return errInvalidInput
		}
	}
	now := time.Now().UTC()
	rp, err := institution.EvaluateOperationPolicyV2Tx(ctx, tx, institution.OperationPolicyV2Request{OperationCode: "admission.appeal.resolve", EffectiveOn: now, DecisionKind: "operation", OfferingID: c.offeringID, LocationID: c.locationID, ActorSubject: sc.actor, Context: map[string]any{"authorization_id": c.authorizationID, "campaign_id": c.campaignID, "application_id": applicationID, "appeal_id": appealID, "outcome": in.Outcome}})
	if err != nil {
		return err
	}
	if !rp.Allowed {
		return errPolicyDenied
	}
	resolutionID := uuid.NewString()
	resultingID, allocationID, resultingPolicyID := "", originalAllocationID, ""
	resultingOutcome := originalOutcome
	var decisionBytes []byte
	decisionHash := ""
	if favourable {
		resultingID = uuid.NewString()
		resultingOutcome = in.ResultingOutcome
		dp, err := institution.EvaluateOperationPolicyV2Tx(ctx, tx, institution.OperationPolicyV2Request{OperationCode: "admission.decision.issue", EffectiveOn: now, DecisionKind: "operation", OfferingID: c.offeringID, LocationID: c.locationID, ActorSubject: sc.actor, Context: decisionPolicyContext(c, applicationID, resultingOutcome)})
		if err != nil {
			return err
		}
		if !dp.Allowed {
			return errPolicyDenied
		}
		resultingPolicyID = dp.EvaluationID
		if resultingOutcome == "admitted" && allocationID == "" {
			allocationID = uuid.NewString()
			_, err = tx.Exec(ctx, `insert into school_admission_capacity_allocations(id,tenant_code,institution_id,application_id,campaign_id,class_offering_context_id,authorization_id,capacity_unit,shift,status,created_by_subject,updated_by_subject) select $1::uuid,$2,$3,$4::uuid,c.id,c.class_offering_context_id,c.authorization_id,c.capacity_unit,c.shift,'held',$5,$5 from school_admission_campaigns c where c.tenant_code=$2 and c.institution_id=$3 and c.id=$6::uuid`, allocationID, sc.tenant, sc.institution, applicationID, sc.actor, c.campaignID)
			if err != nil {
				return err
			}
		}
		if resultingOutcome != "admitted" {
			allocationID = ""
		}
		dpayload, err := BuildAdmissionDecisionLegalPayload(AdmissionDecisionLegalPayloadInput{TenantCode: sc.tenant, InstitutionID: sc.institution, ArtifactID: resultingID, ApplicationID: applicationID, CampaignID: c.campaignID, OfferingID: c.offeringID, LocationID: c.locationID, AuthorizationID: c.authorizationID, ClassOfferingContextID: c.classContextID, DecisionNo: in.ResultingDecisionNo, Outcome: resultingOutcome, Rationale: in.Rationale, PolicyEvaluationV2ID: dp.EvaluationID, ActorSubject: sc.actor, DecidedAt: now, CapacityAllocationID: allocationID, SupersedesDecisionID: originalDecisionID})
		if err != nil {
			return err
		}
		decisionBytes, err = dpayload.CanonicalJSON()
		if err != nil {
			return err
		}
		decisionHash, err = AdmissionDecisionLegalPayloadSHA256(dpayload)
		if err != nil {
			return err
		}
	}
	released := ""
	if favourable && resultingOutcome != "admitted" && originalAllocationID != "" {
		released = originalAllocationID
	}
	payload, err := BuildAdmissionAppealResolutionLegalPayload(AdmissionAppealResolutionLegalPayloadInput{TenantCode: sc.tenant, InstitutionID: sc.institution, ArtifactID: resolutionID, AppealID: appealID, ApplicationID: applicationID, OriginalDecisionID: originalDecisionID, Outcome: in.Outcome, Rationale: in.Rationale, ResultingOutcome: resultingOutcome, PolicyEvaluationV2ID: rp.EvaluationID, ActorSubject: sc.actor, ResolvedAt: now, ResultingDecisionID: resultingID, SupersedesDecisionID: originalDecisionID, CapacityAllocationID: allocationID, OriginalCapacityAllocationID: originalAllocationID, ReleasedCapacityAllocationID: released})
	if err != nil {
		return err
	}
	raw, err := payload.CanonicalJSON()
	if err != nil {
		return err
	}
	hash, err := AdmissionAppealResolutionLegalPayloadSHA256(payload)
	if err != nil {
		return err
	}
	snapshot, _ := json.Marshal(map[string]any{"outcome": in.Outcome, "rationale": in.Rationale, "resulting_outcome": resultingOutcome, "original_decision_id": originalDecisionID, "original_capacity_allocation_id": originalAllocationID, "released_capacity_allocation_id": released, "resulting_decision_no": in.ResultingDecisionNo, "resulting_policy_evaluation_v2_id": resultingPolicyID, "resolved_at": now.Format(time.RFC3339Nano)})
	expires := now.Add(legalPreparationTTL)
	retention, err := snapshotPreparationRetention(ctx, tx, sc, expires)
	if err != nil {
		return err
	}
	err = tx.QueryRow(ctx, `insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,appeal_id,resulting_decision_id,capacity_allocation_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,resulting_decision_payload,resulting_decision_payload_bytes,resulting_decision_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) values($1::uuid,$2,$3,'admission_appeal_resolution',$4::uuid,$5::uuid,$6::uuid,nullif($7,'')::uuid,nullif($8,'')::uuid,$9::uuid,$10,$11::jsonb,$12,$13,$14::jsonb,$15,$16,$17::jsonb,$18,$19,$20::uuid,$21::uuid,$22::uuid,$23,$24,$25) returning id::text,artifact_kind,artifact_id::text,application_id::text,appeal_id::text,resulting_decision_id::text,policy_evaluation_v2_id::text,canonical_payload_bytes,canonical_payload_sha256,resulting_decision_payload_bytes,resulting_decision_payload_sha256,prepared_by_subject,to_char(prepared_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(expires_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),status,retention_policy_id::text,retention_rule_version_id::text,retention_source_id::text,to_char(retention_anchor_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),minimum_retention_days,to_char(required_retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`, preparationID, sc.tenant, sc.institution, resolutionID, applicationID, appealID, resultingID, allocationID, rp.EvaluationID, in.ExpectedVersion, string(raw), string(raw), hash, nullableJSON(decisionBytes), string(decisionBytes), decisionHash, string(snapshot), sc.actor, expires, retention.PolicyID, retention.RuleID, retention.SourceID, retention.Anchor, retention.MinimumDays, retention.RequiredUntil).Scan(&out.ID, &out.ArtifactKind, &out.ArtifactID, &out.ApplicationID, &out.AppealID, &out.ResultingDecisionID, &out.PolicyEvaluationV2ID, &out.CanonicalPayload, &out.CanonicalPayloadSHA256, &out.ResultingDecisionPayload, &out.ResultingDecisionPayloadSHA256, &out.PreparedBySubject, &out.PreparedAt, &out.ExpiresAt, &out.Status, &out.RetentionPolicyID, &out.RetentionRuleVersionID, &out.RetentionSourceID, &out.RetentionAnchorAt, &out.MinimumRetentionDays, &out.RequiredRetentionUntil)
	return err
}

func nullableJSON(value []byte) string {
	if len(value) == 0 {
		return "null"
	}
	return string(value)
}

func (s *Service) finalizeAppealPreparedTx(ctx context.Context, tx pgx.Tx, sc scope, p AdmissionLegalPreparation, allocationID *string, expectedVersion int, snapshot json.RawMessage, in FinalizeAdmissionLegalPreparationRequest) error {
	var appealID string
	var resultingID *string
	var resultPayload []byte
	var resultHash string
	err := tx.QueryRow(ctx, `select appeal_id::text,resulting_decision_id::text,resulting_decision_payload_bytes,resulting_decision_payload_sha256 from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, p.ID).Scan(&appealID, &resultingID, &resultPayload, &resultHash)
	if err != nil {
		return err
	}
	var facts struct {
		Outcome              string `json:"outcome"`
		Rationale            string `json:"rationale"`
		ResultingOutcome     string `json:"resulting_outcome"`
		OriginalDecisionID   string `json:"original_decision_id"`
		OriginalAllocationID string `json:"original_capacity_allocation_id"`
		ReleasedAllocationID string `json:"released_capacity_allocation_id"`
		ResultingDecisionNo  string `json:"resulting_decision_no"`
		ResultingPolicyID    string `json:"resulting_policy_evaluation_v2_id"`
		ResolvedAt           string `json:"resolved_at"`
	}
	if json.Unmarshal(snapshot, &facts) != nil {
		return errInvalidInput
	}
	var applicationID, status string
	var version int
	err = tx.QueryRow(ctx, `select application_id::text,status,expected_version from school_admission_appeals where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, sc.tenant, sc.institution, appealID).Scan(&applicationID, &status, &version)
	if err != nil {
		return err
	}
	if version != expectedVersion || (status != "submitted" && status != "under_review") {
		return errVersionConflict
	}
	c, err := loadDecisionContext(ctx, tx, sc, applicationID, expectedVersion)
	if err != nil {
		return err
	}
	if c.campaignID == "" {
		return errInvalidState
	}
	resolutionArchive, err := loadArchiveSnapshot(ctx, tx, sc.institution, in.Archive)
	if err != nil {
		return err
	}
	if err = requirePreparationArtifactIntent(ctx, tx, sc, p.ID, "primary", in.Archive); err != nil {
		return err
	}
	if !strings.EqualFold(resolutionArchive.mime, "application/pdf") {
		return errInvalidInput
	}
	if err = validatePreparationRetention(ctx, tx, sc, preparationRetentionFromModel(p), resolutionArchive.retention); err != nil {
		return err
	}
	if resultingID != nil {
		if in.ResultingDecisionArchive == nil || !validUUID(in.ResultingDecisionArchive.DocumentID) || !validUUID(in.ResultingDecisionArchive.VersionID) {
			return errInvalidInput
		}
		if in.ResultingDecisionArchive.DocumentID == in.Archive.DocumentID && in.ResultingDecisionArchive.VersionID == in.Archive.VersionID {
			return errInvalidInput
		}
	}
	var decisionArchive archiveSnapshot
	if resultingID != nil {
		decisionArchive, err = loadArchiveSnapshot(ctx, tx, sc.institution, *in.ResultingDecisionArchive)
		if err != nil {
			return err
		}
		if err = requirePreparationArtifactIntent(ctx, tx, sc, p.ID, "resulting_decision", *in.ResultingDecisionArchive); err != nil {
			return err
		}
		if !strings.EqualFold(decisionArchive.mime, "application/pdf") {
			return errInvalidInput
		}
		if err = validatePreparationRetention(ctx, tx, sc, preparationRetentionFromModel(p), decisionArchive.retention); err != nil {
			return err
		}
	}
	if resultingID != nil {
		_, err = tx.Exec(ctx, `insert into school_admission_decisions(id,tenant_code,institution_id,application_id,capacity_allocation_id,policy_evaluation_v2_id,decision_no,outcome,rationale,decided_at,decided_by_subject,supersedes_decision_id,decision_snapshot,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10::timestamptz,$11,$12::uuid,$13::jsonb,$14::uuid,$15::uuid,$16,$17,$18,$19,$20,$21)`, *resultingID, sc.tenant, sc.institution, applicationID, allocationID, facts.ResultingPolicyID, facts.ResultingDecisionNo, facts.ResultingOutcome, facts.Rationale, facts.ResolvedAt, sc.actor, facts.OriginalDecisionID, string(resultPayload), in.ResultingDecisionArchive.DocumentID, in.ResultingDecisionArchive.VersionID, decisionArchive.versionNo, decisionArchive.bucket, decisionArchive.objectKey, decisionArchive.objectVersion, decisionArchive.retention, decisionArchive.sha)
		if err != nil {
			return err
		}
		if err = persistDSSBinding(ctx, tx, sc, "admission_decision", *resultingID, facts.ResultingPolicyID, *in.ResultingDecisionArchive, decisionArchive, sc.actor, resultHash); err != nil {
			return err
		}
	}
	if facts.ReleasedAllocationID != "" {
		_, err = tx.Exec(ctx, `update school_admission_capacity_allocations set status='released',expected_version=expected_version+1,updated_at=now(),updated_by_subject=$1 where tenant_code=$2 and institution_id=$3 and id=$4::uuid and status='held'`, sc.actor, sc.tenant, sc.institution, facts.ReleasedAllocationID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `insert into school_admission_appeal_resolutions(id,tenant_code,institution_id,appeal_id,resulting_decision_id,resulting_outcome,capacity_allocation_id,policy_evaluation_v2_id,outcome,rationale,resolved_at,resolved_by_subject,resolution_snapshot,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256) values($1::uuid,$2,$3,$4::uuid,$5::uuid,$6,nullif($7,'')::uuid,$8::uuid,$9,$10,$11::timestamptz,$12,$13::jsonb,$14::uuid,$15::uuid,$16,$17,$18,$19,$20,$21)`, p.ArtifactID, sc.tenant, sc.institution, appealID, resultingID, facts.ResultingOutcome, optional(allocationID), p.PolicyEvaluationV2ID, facts.Outcome, facts.Rationale, facts.ResolvedAt, sc.actor, string(snapshot), in.Archive.DocumentID, in.Archive.VersionID, resolutionArchive.versionNo, resolutionArchive.bucket, resolutionArchive.objectKey, resolutionArchive.objectVersion, resolutionArchive.retention, resolutionArchive.sha)
	if err != nil {
		return err
	}
	if err = persistDSSBinding(ctx, tx, sc, "admission_appeal_resolution", p.ArtifactID, p.PolicyEvaluationV2ID, in.Archive, resolutionArchive, sc.actor, p.CanonicalPayloadSHA256); err != nil {
		return err
	}
	if resultingID != nil {
		tag, e := tx.Exec(ctx, `update school_admission_applications set status=$1,expected_version=expected_version+1,updated_at=now(),updated_by_subject=$2 where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, facts.ResultingOutcome, sc.actor, sc.tenant, sc.institution, applicationID, expectedVersion)
		err = e
		if err == nil && tag.RowsAffected() != 1 {
			return errVersionConflict
		}
	}
	tag, err := tx.Exec(ctx, `update school_admission_appeals set status='resolved',expected_version=expected_version+1,updated_at=now(),updated_by_subject=$1 where tenant_code=$2 and institution_id=$3 and id=$4::uuid and expected_version=$5`, sc.actor, sc.tenant, sc.institution, appealID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errVersionConflict
	}
	_, err = tx.Exec(ctx, `update school_admission_legal_preparations set status='finalized' where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, p.ID)
	if err != nil {
		return err
	}
	if err = outbox(ctx, tx, sc, "admission_appeal", appealID, "admission.appeal.finalized", map[string]any{"resolutionId": p.ArtifactID, "preparationId": p.ID, "resultingDecisionId": resultingID}); err != nil {
		return err
	}
	return auditEvent(ctx, tx, sc, "admission.appeal.finalized", "school_admission_appeal_resolution", p.ArtifactID)
}
