package admission

import "encoding/json"

type RegulatorySourceOption struct {
	ID               string  `json:"id"`
	SourceKind       string  `json:"source_kind"`
	Citation         string  `json:"citation"`
	ArticleReference string  `json:"article_reference"`
	Issuer           string  `json:"issuer"`
	EffectiveFrom    *string `json:"effective_from"`
	EffectiveTo      *string `json:"effective_to"`
}
type CandidatePartyOption struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}
type StudentOption struct {
	ID          string  `json:"id"`
	PartyID     *string `json:"party_id"`
	StudentCode string  `json:"student_code"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Status      string  `json:"status"`
}
type ArchiveVersionOption struct {
	DocumentID       string `json:"document_id"`
	VersionID        string `json:"version_id"`
	VersionNo        int    `json:"version_no"`
	Title            string `json:"title"`
	OriginalFileName string `json:"original_file_name"`
	MIMEType         string `json:"mime_type"`
	SHA256           string `json:"sha256"`
	RetentionUntil   string `json:"retention_until"`
}

type CreateClassOfferingContextRequest struct {
	ClassID         string  `json:"class_id"`
	OfferingID      string  `json:"offering_id"`
	LocationID      string  `json:"location_id"`
	AuthorizationID string  `json:"authorization_id"`
	SchoolYear      string  `json:"school_year"`
	Shift           string  `json:"shift"`
	EffectiveFrom   string  `json:"effective_from"`
	EffectiveTo     *string `json:"effective_to"`
}

type ClassOfferingContext struct {
	ID              string  `json:"id"`
	ClassID         string  `json:"class_id"`
	OfferingID      string  `json:"offering_id"`
	LocationID      string  `json:"location_id"`
	AuthorizationID string  `json:"authorization_id"`
	SchoolYear      string  `json:"school_year"`
	Shift           string  `json:"shift"`
	Active          bool    `json:"active"`
	EffectiveFrom   string  `json:"effective_from"`
	EffectiveTo     *string `json:"effective_to"`
	ExpectedVersion int     `json:"expected_version"`
}

type CriterionInput struct {
	Code         string          `json:"code"`
	Title        string          `json:"title"`
	Kind         string          `json:"kind"`
	Required     bool            `json:"required"`
	Weight       float64         `json:"weight"`
	Ordinal      int             `json:"ordinal"`
	RuleSnapshot json.RawMessage `json:"rule_snapshot"`
}

type DocumentRequirementInput struct {
	Code             string   `json:"code"`
	Title            string   `json:"title"`
	Required         bool     `json:"required"`
	AllowedMIMETypes []string `json:"allowed_mime_types"`
	Ordinal          int      `json:"ordinal"`
}

// Scope is intentionally absent from all request models. It is established by
// the authenticated host and pinned on the request connection.
type CreateCampaignRequest struct {
	SourceID               string                     `json:"source_id"`
	Code                   string                     `json:"code"`
	Title                  string                     `json:"title"`
	SchoolYear             string                     `json:"school_year"`
	OfferingID             string                     `json:"offering_id"`
	LocationID             string                     `json:"location_id"`
	AuthorizationID        string                     `json:"authorization_id"`
	ClassOfferingContextID string                     `json:"class_offering_context_id"`
	CapacityLimit          int                        `json:"capacity_limit"`
	CapacityUnit           string                     `json:"capacity_unit"`
	StudentPlaceLimit      int                        `json:"student_place_limit"`
	CapacityBasis          json.RawMessage            `json:"capacity_basis"`
	Shift                  string                     `json:"shift"`
	OpensOn                string                     `json:"opens_on"`
	ClosesOn               string                     `json:"closes_on"`
	DecisionDueOn          *string                    `json:"decision_due_on"`
	Criteria               []CriterionInput           `json:"criteria"`
	DocumentRequirements   []DocumentRequirementInput `json:"document_requirements"`
}

type Campaign struct {
	ID                     string          `json:"id"`
	SourceID               string          `json:"source_id"`
	Code                   string          `json:"code"`
	Title                  string          `json:"title"`
	SchoolYear             string          `json:"school_year"`
	OfferingID             string          `json:"offering_id"`
	LocationID             string          `json:"location_id"`
	AuthorizationID        string          `json:"authorization_id"`
	ClassOfferingContextID string          `json:"class_offering_context_id"`
	CapacityLimit          int             `json:"capacity_limit"`
	CapacityUnit           string          `json:"capacity_unit"`
	StudentPlaceLimit      int             `json:"student_place_limit"`
	CapacityBasis          json.RawMessage `json:"capacity_basis"`
	Shift                  string          `json:"shift"`
	OpensOn                string          `json:"opens_on"`
	ClosesOn               string          `json:"closes_on"`
	DecisionDueOn          *string         `json:"decision_due_on"`
	Status                 string          `json:"status"`
	ExpectedVersion        int             `json:"expected_version"`
}

type Criterion struct {
	ID              string          `json:"id"`
	CampaignID      string          `json:"campaign_id"`
	Code            string          `json:"code"`
	Title           string          `json:"title"`
	Kind            string          `json:"kind"`
	Required        bool            `json:"required"`
	Weight          float64         `json:"weight"`
	Ordinal         int             `json:"ordinal"`
	RuleSnapshot    json.RawMessage `json:"rule_snapshot"`
	ExpectedVersion int             `json:"expected_version"`
}

type DocumentRequirement struct {
	ID               string   `json:"id"`
	CampaignID       string   `json:"campaign_id"`
	Code             string   `json:"code"`
	Title            string   `json:"title"`
	Required         bool     `json:"required"`
	AllowedMIMETypes []string `json:"allowed_mime_types"`
	Ordinal          int      `json:"ordinal"`
	ExpectedVersion  int      `json:"expected_version"`
}

type TransitionRequest struct {
	Status          string `json:"status"`
	ExpectedVersion int    `json:"expected_version"`
}

type CreateApplicationRequest struct {
	CampaignID       string          `json:"campaign_id"`
	ApplicationNo    string          `json:"application_no"`
	CandidatePartyID string          `json:"candidate_party_id"`
	ConsentSnapshot  json.RawMessage `json:"consent_snapshot"`
}

type Application struct {
	ID                string          `json:"id"`
	CampaignID        string          `json:"campaign_id"`
	ApplicationNo     string          `json:"application_no"`
	CandidatePartyID  string          `json:"candidate_party_id"`
	StudentID         *string         `json:"student_id"`
	SubmittedAt       *string         `json:"submitted_at"`
	Status            string          `json:"status"`
	ConsentSnapshot   json.RawMessage `json:"consent_snapshot"`
	ExpectedVersion   int             `json:"expected_version"`
	CampaignCode      string          `json:"campaign_code"`
	CandidateName     string          `json:"candidate_name"`
	DocumentsComplete bool            `json:"documents_complete"`
	CriteriaComplete  bool            `json:"criteria_complete"`
}

type AssessCriterionRequest struct {
	CriterionID      string          `json:"criterion_id"`
	Outcome          string          `json:"outcome"`
	Score            *float64        `json:"score"`
	Rationale        string          `json:"rationale"`
	EvidenceSnapshot json.RawMessage `json:"evidence_snapshot"`
	ExpectedVersion  int             `json:"expected_version"`
}

type CriterionAssessment struct {
	ID               string          `json:"id"`
	CriterionID      string          `json:"criterion_id"`
	Code             string          `json:"code"`
	Title            string          `json:"title"`
	Required         bool            `json:"required"`
	Outcome          string          `json:"outcome"`
	Score            *float64        `json:"score"`
	Rationale        string          `json:"rationale"`
	EvidenceSnapshot json.RawMessage `json:"evidence_snapshot"`
	ExpectedVersion  int             `json:"expected_version"`
}

type ApplicationDocument struct {
	ID                string  `json:"id"`
	RequirementID     *string `json:"requirement_id"`
	DocumentKind      string  `json:"document_kind"`
	Status            string  `json:"status"`
	ArchiveDocumentID *string `json:"archive_document_id"`
	ArchiveVersionID  *string `json:"archive_version_id"`
	ExpectedVersion   int     `json:"expected_version"`
}

type ApplicationDetail struct {
	Application Application           `json:"application"`
	Assessments []CriterionAssessment `json:"assessments"`
	Documents   []ApplicationDocument `json:"documents"`
	Decisions   []Decision            `json:"decisions"`
}

type ReviewApplicationDocumentRequest struct {
	Status          string            `json:"status"`
	ReviewNote      string            `json:"review_note"`
	ExpectedVersion int               `json:"expected_version"`
	Archive         *ArchiveReference `json:"archive"`
}

type ArchiveReference struct {
	DocumentID string `json:"document_id"`
	VersionID  string `json:"version_id"`
}

type IssueDecisionRequest struct {
	DecisionNo      string           `json:"decision_no"`
	Outcome         string           `json:"outcome"`
	Rationale       string           `json:"rationale"`
	RankingValue    *float64         `json:"ranking_value"`
	AppealDeadline  *string          `json:"appeal_deadline"`
	ExpectedVersion int              `json:"expected_version"`
	Archive         ArchiveReference `json:"archive"`
}

type PrepareDecisionRequest struct {
	DecisionNo      string   `json:"decision_no"`
	Outcome         string   `json:"outcome"`
	Rationale       string   `json:"rationale"`
	RankingValue    *float64 `json:"ranking_value"`
	AppealDeadline  *string  `json:"appeal_deadline"`
	ExpectedVersion int      `json:"expected_version"`
}

type FinalizeAdmissionLegalPreparationRequest struct {
	PreparationID            string            `json:"preparation_id"`
	Archive                  ArchiveReference  `json:"archive"`
	ResultingDecisionArchive *ArchiveReference `json:"resulting_decision_archive,omitempty"`
}

type AdmissionLegalPreparation struct {
	ID                             string          `json:"id"`
	ArtifactKind                   string          `json:"artifact_kind"`
	ArtifactID                     string          `json:"artifact_id"`
	ApplicationID                  string          `json:"application_id"`
	AppealID                       *string         `json:"appeal_id,omitempty"`
	ResultingDecisionID            *string         `json:"resulting_decision_id,omitempty"`
	PolicyEvaluationV2ID           string          `json:"policy_evaluation_v2_id"`
	RetentionPolicyID              string          `json:"retention_policy_id"`
	RetentionRuleVersionID         string          `json:"retention_rule_version_id"`
	RetentionSourceID              string          `json:"retention_source_id"`
	RetentionAnchorAt              string          `json:"retention_anchor_at"`
	MinimumRetentionDays           int             `json:"minimum_retention_days"`
	RequiredRetentionUntil         string          `json:"required_retention_until"`
	CanonicalPayload               json.RawMessage `json:"canonical_payload"`
	CanonicalPayloadBase64         string          `json:"canonical_payload_base64"`
	CanonicalPayloadSHA256         string          `json:"canonical_payload_sha256"`
	ResultingDecisionPayload       json.RawMessage `json:"resulting_decision_payload,omitempty"`
	ResultingDecisionPayloadBase64 string          `json:"resulting_decision_payload_base64,omitempty"`
	ResultingDecisionPayloadSHA256 string          `json:"resulting_decision_payload_sha256,omitempty"`
	PreparedBySubject              string          `json:"prepared_by_subject"`
	PreparedAt                     string          `json:"prepared_at"`
	ExpiresAt                      string          `json:"expires_at"`
	Status                         string          `json:"status"`
	Replayed                       bool            `json:"replayed,omitempty"`
}

type Decision struct {
	ID                   string   `json:"id"`
	ApplicationID        string   `json:"application_id"`
	CapacityAllocationID *string  `json:"capacity_allocation_id"`
	PolicyEvaluationV2ID string   `json:"policy_evaluation_v2_id"`
	DecisionNo           string   `json:"decision_no"`
	Outcome              string   `json:"outcome"`
	Rationale            string   `json:"rationale"`
	RankingValue         *float64 `json:"ranking_value"`
	AppealDeadline       *string  `json:"appeal_deadline"`
	DecidedAt            string   `json:"decided_at"`
	ApplicationNo        string   `json:"application_no"`
	CandidateName        string   `json:"candidate_name"`
}

type CreateAppealRequest struct {
	DecisionID         string            `json:"decision_id"`
	AppealNo           string            `json:"appeal_no"`
	SubmittedByPartyID string            `json:"submitted_by_party_id"`
	Statement          string            `json:"statement"`
	Archive            *ArchiveReference `json:"archive"`
}

type Appeal struct {
	ID                 string  `json:"id"`
	ApplicationID      string  `json:"application_id"`
	DecisionID         string  `json:"decision_id"`
	AppealNo           string  `json:"appeal_no"`
	Status             string  `json:"status"`
	SubmittedAt        *string `json:"submitted_at"`
	SubmittedByPartyID *string `json:"submitted_by_party_id"`
	ExpectedVersion    int     `json:"expected_version"`
	ApplicationNo      string  `json:"application_no"`
}

type ResolveAppealRequest struct {
	Outcome                    string           `json:"outcome"`
	Rationale                  string           `json:"rationale"`
	ExpectedVersion            int              `json:"expected_version"`
	ApplicationExpectedVersion int              `json:"application_expected_version"`
	ResultingDecisionNo        string           `json:"resulting_decision_no"`
	ResultingOutcome           string           `json:"resulting_outcome"`
	Archive                    ArchiveReference `json:"archive"`
}

type PrepareAppealResolutionRequest struct {
	Outcome                    string `json:"outcome"`
	Rationale                  string `json:"rationale"`
	ExpectedVersion            int    `json:"expected_version"`
	ApplicationExpectedVersion int    `json:"application_expected_version"`
	ResultingDecisionNo        string `json:"resulting_decision_no"`
	ResultingOutcome           string `json:"resulting_outcome"`
}

type ProposeAdmissionSignerAuthorizationRequest struct {
	CertificateSHA256 string `json:"certificate_sha256"`
	UserID            string `json:"user_id"`
	PermissionCode    string `json:"permission_code"`
	ValidUntil        string `json:"valid_until"`
}

type ApproveAdmissionSignerAuthorizationRequest struct {
	ProposalID string `json:"proposal_id"`
}

type RevokeAdmissionSignerAuthorizationRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
}

type AdmissionSignerAuthorization struct {
	ID                string `json:"id"`
	ProposalID        string `json:"proposal_id,omitempty"`
	CertificateSHA256 string `json:"certificate_sha256"`
	UserID            string `json:"user_id"`
	ActorSubject      string `json:"actor_subject"`
	PermissionCode    string `json:"permission_code"`
	ValidUntil        string `json:"valid_until"`
	ProposedBySubject string `json:"proposed_by_subject"`
	ApprovedBySubject string `json:"approved_by_subject,omitempty"`
	Status            string `json:"status"`
	ExpectedVersion   int    `json:"expected_version"`
	Replayed          bool   `json:"replayed,omitempty"`
}

type EnrolApplicationRequest struct {
	StudentCode     string `json:"student_code"`
	EnrolledFrom    string `json:"enrolled_from"`
	ExpectedVersion int    `json:"expected_version"`
}

type CommandResult struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	ExpectedVersion int    `json:"expected_version,omitempty"`
	Replayed        bool   `json:"replayed,omitempty"`
}

type AdmissionDSSRetentionPolicy struct {
	ID                   string `json:"id"`
	Status               string `json:"status"`
	RuleVersionID        string `json:"rule_version_id"`
	MinimumRetentionDays int    `json:"minimum_retention_days"`
	EffectiveFrom        string `json:"effective_from"`
	SourceID             string `json:"source_id"`
	Replayed             bool   `json:"replayed,omitempty"`
}

type AdmissionRetentionRuleVersion struct {
	ID                   string  `json:"id"`
	ArtifactKind         string  `json:"artifact_kind"`
	Status               string  `json:"status"`
	MinimumRetentionDays int     `json:"minimum_retention_days"`
	EffectiveFrom        string  `json:"effective_from"`
	EffectiveTo          *string `json:"effective_to,omitempty"`
	SourceID             string  `json:"source_id"`
	SourceChecksumSHA256 string  `json:"source_checksum_sha256"`
	ProposedBySubject    string  `json:"proposed_by_subject"`
	ApprovedBySubject    string  `json:"approved_by_subject,omitempty"`
	Replayed             bool    `json:"replayed,omitempty"`
}
