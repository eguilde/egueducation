package admission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	AdmissionDecisionLegalPayloadVersion         = "egueducation.admission.decision.v1"
	AdmissionAppealResolutionLegalPayloadVersion = "egueducation.admission.appeal-resolution.v1"
)

// AdmissionLegalScope is intentionally server-derived. It is part of the
// signed payload so a valid document cannot be replayed into another tenant or
// institution.
type AdmissionLegalScope struct {
	TenantCode    string `json:"tenant_code"`
	InstitutionID string `json:"institution_id"`
}

// AdmissionDecisionLegalPayload is a fixed-field, versioned signing contract.
// It deliberately contains no maps: encoding/json therefore emits fields in a
// deterministic declaration order and cannot be influenced by map iteration.
type AdmissionDecisionLegalPayload struct {
	SchemaVersion          string              `json:"schema_version"`
	ArtifactKind           string              `json:"artifact_kind"`
	Scope                  AdmissionLegalScope `json:"scope"`
	ArtifactID             string              `json:"artifact_id"`
	ApplicationID          string              `json:"application_id"`
	CampaignID             string              `json:"campaign_id"`
	OfferingID             string              `json:"offering_id"`
	LocationID             string              `json:"location_id"`
	AuthorizationID        string              `json:"authorization_id"`
	ClassOfferingContextID string              `json:"class_offering_context_id"`
	DecisionNo             string              `json:"decision_no"`
	Outcome                string              `json:"outcome"`
	Rationale              string              `json:"rationale"`
	RankingValue           *float64            `json:"ranking_value"`
	AppealDeadline         *string             `json:"appeal_deadline"`
	PolicyEvaluationV2ID   string              `json:"policy_evaluation_v2_id"`
	ActorSubject           string              `json:"actor_subject"`
	DecidedAt              string              `json:"decided_at"`
	CapacityAllocationID   *string             `json:"capacity_allocation_id"`
	SupersedesDecisionID   *string             `json:"supersedes_decision_id"`
}

type AdmissionDecisionLegalPayloadInput struct {
	TenantCode             string
	InstitutionID          string
	ArtifactID             string
	ApplicationID          string
	CampaignID             string
	OfferingID             string
	LocationID             string
	AuthorizationID        string
	ClassOfferingContextID string
	DecisionNo             string
	Outcome                string
	Rationale              string
	RankingValue           *float64
	AppealDeadline         *string
	PolicyEvaluationV2ID   string
	ActorSubject           string
	DecidedAt              time.Time
	CapacityAllocationID   string
	SupersedesDecisionID   string
}

func BuildAdmissionDecisionLegalPayload(in AdmissionDecisionLegalPayloadInput) (AdmissionDecisionLegalPayload, error) {
	if err := validateLegalPayloadCommon(in.TenantCode, in.InstitutionID, in.ActorSubject, in.DecidedAt); err != nil {
		return AdmissionDecisionLegalPayload{}, err
	}
	for _, id := range []string{in.ArtifactID, in.ApplicationID, in.CampaignID, in.OfferingID, in.LocationID, in.AuthorizationID, in.ClassOfferingContextID, in.PolicyEvaluationV2ID} {
		if !validUUID(id) {
			return AdmissionDecisionLegalPayload{}, errors.New("admission decision legal payload contains an invalid required identifier")
		}
	}
	if !optionalUUID(in.CapacityAllocationID) || !optionalUUID(in.SupersedesDecisionID) {
		return AdmissionDecisionLegalPayload{}, errors.New("admission decision legal payload contains an invalid optional identifier")
	}
	decisionNo, outcome, rationale := strings.TrimSpace(in.DecisionNo), strings.TrimSpace(in.Outcome), strings.TrimSpace(in.Rationale)
	if decisionNo == "" || outcome == "" || rationale == "" {
		return AdmissionDecisionLegalPayload{}, errors.New("admission decision legal payload is incomplete")
	}
	appealDeadline := normalizedOptionalString(in.AppealDeadline)
	if appealDeadline != nil && !validDate(*appealDeadline) {
		return AdmissionDecisionLegalPayload{}, errors.New("admission decision legal payload contains an invalid appeal deadline")
	}
	var rankingValue *float64
	if in.RankingValue != nil {
		value := *in.RankingValue
		rankingValue = &value
	}
	return AdmissionDecisionLegalPayload{
		SchemaVersion: AdmissionDecisionLegalPayloadVersion, ArtifactKind: "admission_decision",
		Scope:      AdmissionLegalScope{TenantCode: strings.TrimSpace(in.TenantCode), InstitutionID: strings.TrimSpace(in.InstitutionID)},
		ArtifactID: strings.TrimSpace(in.ArtifactID), ApplicationID: strings.TrimSpace(in.ApplicationID), CampaignID: strings.TrimSpace(in.CampaignID),
		OfferingID: strings.TrimSpace(in.OfferingID), LocationID: strings.TrimSpace(in.LocationID), AuthorizationID: strings.TrimSpace(in.AuthorizationID), ClassOfferingContextID: strings.TrimSpace(in.ClassOfferingContextID),
		DecisionNo: decisionNo, Outcome: outcome, Rationale: rationale, RankingValue: rankingValue, AppealDeadline: appealDeadline,
		PolicyEvaluationV2ID: strings.TrimSpace(in.PolicyEvaluationV2ID), ActorSubject: strings.TrimSpace(in.ActorSubject), DecidedAt: canonicalLegalTimestamp(in.DecidedAt),
		CapacityAllocationID: normalizedOptionalUUID(in.CapacityAllocationID), SupersedesDecisionID: normalizedOptionalUUID(in.SupersedesDecisionID),
	}, nil
}

func (payload AdmissionDecisionLegalPayload) CanonicalJSON() ([]byte, error) {
	return json.Marshal(payload)
}

func AdmissionDecisionLegalPayloadSHA256(payload AdmissionDecisionLegalPayload) (string, error) {
	return canonicalAdmissionLegalPayloadSHA256(payload)
}

type AdmissionAppealResolutionLegalPayload struct {
	SchemaVersion                string              `json:"schema_version"`
	ArtifactKind                 string              `json:"artifact_kind"`
	Scope                        AdmissionLegalScope `json:"scope"`
	ArtifactID                   string              `json:"artifact_id"`
	AppealID                     string              `json:"appeal_id"`
	ApplicationID                string              `json:"application_id"`
	OriginalDecisionID           string              `json:"original_decision_id"`
	Outcome                      string              `json:"outcome"`
	Rationale                    string              `json:"rationale"`
	ResultingOutcome             string              `json:"resulting_outcome"`
	PolicyEvaluationV2ID         string              `json:"policy_evaluation_v2_id"`
	ActorSubject                 string              `json:"actor_subject"`
	ResolvedAt                   string              `json:"resolved_at"`
	ResultingDecisionID          *string             `json:"resulting_decision_id"`
	SupersedesDecisionID         *string             `json:"supersedes_decision_id"`
	CapacityAllocationID         *string             `json:"capacity_allocation_id"`
	OriginalCapacityAllocationID *string             `json:"original_capacity_allocation_id"`
	ReleasedCapacityAllocationID *string             `json:"released_capacity_allocation_id"`
}

type AdmissionAppealResolutionLegalPayloadInput struct {
	TenantCode                   string
	InstitutionID                string
	ArtifactID                   string
	AppealID                     string
	ApplicationID                string
	OriginalDecisionID           string
	Outcome                      string
	Rationale                    string
	ResultingOutcome             string
	PolicyEvaluationV2ID         string
	ActorSubject                 string
	ResolvedAt                   time.Time
	ResultingDecisionID          string
	SupersedesDecisionID         string
	CapacityAllocationID         string
	OriginalCapacityAllocationID string
	ReleasedCapacityAllocationID string
}

func BuildAdmissionAppealResolutionLegalPayload(in AdmissionAppealResolutionLegalPayloadInput) (AdmissionAppealResolutionLegalPayload, error) {
	if err := validateLegalPayloadCommon(in.TenantCode, in.InstitutionID, in.ActorSubject, in.ResolvedAt); err != nil {
		return AdmissionAppealResolutionLegalPayload{}, err
	}
	for _, id := range []string{in.ArtifactID, in.AppealID, in.ApplicationID, in.OriginalDecisionID, in.PolicyEvaluationV2ID} {
		if !validUUID(id) {
			return AdmissionAppealResolutionLegalPayload{}, errors.New("admission appeal-resolution legal payload contains an invalid required identifier")
		}
	}
	for _, id := range []string{in.ResultingDecisionID, in.SupersedesDecisionID, in.CapacityAllocationID, in.OriginalCapacityAllocationID, in.ReleasedCapacityAllocationID} {
		if !optionalUUID(id) {
			return AdmissionAppealResolutionLegalPayload{}, errors.New("admission appeal-resolution legal payload contains an invalid optional identifier")
		}
	}
	outcome, rationale, resultingOutcome := strings.TrimSpace(in.Outcome), strings.TrimSpace(in.Rationale), strings.TrimSpace(in.ResultingOutcome)
	if outcome == "" || rationale == "" || resultingOutcome == "" {
		return AdmissionAppealResolutionLegalPayload{}, errors.New("admission appeal-resolution legal payload is incomplete")
	}
	return AdmissionAppealResolutionLegalPayload{
		SchemaVersion: AdmissionAppealResolutionLegalPayloadVersion, ArtifactKind: "admission_appeal_resolution",
		Scope:      AdmissionLegalScope{TenantCode: strings.TrimSpace(in.TenantCode), InstitutionID: strings.TrimSpace(in.InstitutionID)},
		ArtifactID: strings.TrimSpace(in.ArtifactID), AppealID: strings.TrimSpace(in.AppealID), ApplicationID: strings.TrimSpace(in.ApplicationID), OriginalDecisionID: strings.TrimSpace(in.OriginalDecisionID),
		Outcome: outcome, Rationale: rationale, ResultingOutcome: resultingOutcome, PolicyEvaluationV2ID: strings.TrimSpace(in.PolicyEvaluationV2ID), ActorSubject: strings.TrimSpace(in.ActorSubject), ResolvedAt: canonicalLegalTimestamp(in.ResolvedAt),
		ResultingDecisionID: normalizedOptionalUUID(in.ResultingDecisionID), SupersedesDecisionID: normalizedOptionalUUID(in.SupersedesDecisionID), CapacityAllocationID: normalizedOptionalUUID(in.CapacityAllocationID),
		OriginalCapacityAllocationID: normalizedOptionalUUID(in.OriginalCapacityAllocationID), ReleasedCapacityAllocationID: normalizedOptionalUUID(in.ReleasedCapacityAllocationID),
	}, nil
}

func (payload AdmissionAppealResolutionLegalPayload) CanonicalJSON() ([]byte, error) {
	return json.Marshal(payload)
}

func AdmissionAppealResolutionLegalPayloadSHA256(payload AdmissionAppealResolutionLegalPayload) (string, error) {
	return canonicalAdmissionLegalPayloadSHA256(payload)
}

func canonicalAdmissionLegalPayloadSHA256(payload any) (string, error) {
	canonical, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func validateLegalPayloadCommon(tenantCode, institutionID, actorSubject string, occurredAt time.Time) error {
	if strings.TrimSpace(tenantCode) == "" || strings.TrimSpace(institutionID) == "" || strings.TrimSpace(actorSubject) == "" || occurredAt.IsZero() {
		return errors.New("admission legal payload scope, actor and timestamp are required")
	}
	return nil
}

func canonicalLegalTimestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func optionalUUID(value string) bool { return strings.TrimSpace(value) == "" || validUUID(value) }

func normalizedOptionalUUID(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
