package institution

// RegulatoryProfile is the effective, host-scoped classification of one
// school. Tenant and institution identifiers are response-only provenance.
type RegulatoryProfile struct {
	ID                     string   `json:"id"`
	TenantCode             string   `json:"tenant_code"`
	InstitutionID          string   `json:"institution_id"`
	Version                int      `json:"version"`
	Status                 string   `json:"status"`
	SchoolLegalForm        *string  `json:"school_legal_form"`
	RegulatoryProfile      string   `json:"regulatory_profile"`
	AuthorizationStatus    string   `json:"authorization_status"`
	AccreditationReference string   `json:"accreditation_reference"`
	AuthorizedLevels       []string `json:"authorized_levels"`
	HasLegalPersonality    bool     `json:"has_legal_personality"`
	TaxIdentifier          string   `json:"tax_identifier"`
	FounderName            string   `json:"founder_name"`
	FunderName             string   `json:"funder_name"`
	BudgetAuthorityName    string   `json:"budget_authority_name"`
	// Compatibility projection only; legal truth is an immutable procurement assessment.
	IsContractingAuthority bool     `json:"is_contracting_authority"`
	AccountingProfile      string   `json:"accounting_profile"`
	ProcurementProfile     string   `json:"procurement_profile"`
	PayrollProfile         string   `json:"payroll_profile"`
	VATProfile             string   `json:"vat_profile"`
	TreasuryRequired       bool     `json:"treasury_required"`
	PublicFunding          bool     `json:"public_funding"`
	ProgramCodes           []string `json:"program_codes"`
	EffectiveFrom          *string  `json:"effective_from"`
	EffectiveTo            *string  `json:"effective_to"`
	SourceReference        string   `json:"source_reference"`
	ApprovedBySubject      string   `json:"approved_by_subject"`
	ApprovedAt             *string  `json:"approved_at"`
	CreatedBySubject       string   `json:"created_by_subject"`
	CreatedAt              string   `json:"created_at"`
	UpdatedBySubject       string   `json:"updated_by_subject"`
	UpdatedAt              string   `json:"updated_at"`
}

// PutRegulatoryProfileRequest is deliberately closed by the HTTP decoder and
// OpenAPI schema. Security scope and approval actor are never client fields.
type PutRegulatoryProfileRequest struct {
	ExpectedVersion        int                         `json:"expected_version"`
	Status                 string                      `json:"status"`
	SchoolLegalForm        string                      `json:"school_legal_form"`
	RegulatoryProfile      string                      `json:"regulatory_profile"`
	AuthorizationStatus    string                      `json:"authorization_status"`
	AccreditationReference string                      `json:"accreditation_reference"`
	AuthorizedLevels       []string                    `json:"authorized_levels"`
	HasLegalPersonality    bool                        `json:"has_legal_personality"`
	TaxIdentifier          string                      `json:"tax_identifier"`
	FounderName            string                      `json:"founder_name"`
	FunderName             string                      `json:"funder_name"`
	BudgetAuthorityName    string                      `json:"budget_authority_name"`
	AccountingProfile      string                      `json:"accounting_profile"`
	ProcurementProfile     string                      `json:"procurement_profile"`
	PayrollProfile         string                      `json:"payroll_profile"`
	VATProfile             string                      `json:"vat_profile"`
	ProgramCodes           []string                    `json:"program_codes"`
	EffectiveFrom          string                      `json:"effective_from"`
	EffectiveTo            *string                     `json:"effective_to"`
	Source                 RegulatorySourceRequest     `json:"source"`
	ConfessionalOverlay    *ConfessionalOverlayRequest `json:"confessional_overlay,omitempty"`
}

type RegulatorySourceRequest struct {
	SourceKind       string  `json:"source_kind"`
	Citation         string  `json:"citation"`
	ArticleReference string  `json:"article_reference"`
	Issuer           string  `json:"issuer"`
	SourceURL        string  `json:"source_url"`
	PublishedOn      *string `json:"published_on"`
	ConsolidatedOn   *string `json:"consolidated_on"`
	ChecksumSHA256   string  `json:"checksum_sha256"`
}

type ConfessionalOverlayRequest struct {
	CultPartyID       string `json:"cult_party_id"`
	CultCode          string `json:"cult_code"`
	ProtocolReference string `json:"protocol_reference"`
}

type PolicyPackSummary struct {
	ID                string   `json:"id"`
	Code              string   `json:"code"`
	Version           int      `json:"version"`
	RegulatoryProfile string   `json:"regulatory_profile"`
	EffectiveFrom     string   `json:"effective_from"`
	EffectiveTo       *string  `json:"effective_to"`
	ChecksumSHA256    string   `json:"checksum_sha256"`
	SourceReferences  []string `json:"source_references"`
}

type PolicyCapability struct {
	Code               string   `json:"code"`
	Enabled            bool     `json:"enabled"`
	Reason             string   `json:"reason"`
	RequiredPermission string   `json:"required_permission"`
	RequiredDocuments  []string `json:"required_documents"`
	RequiredApprovals  []string `json:"required_approvals"`
	WizardSteps        []string `json:"wizard_steps"`
}

type InstitutionCapabilitiesResponse struct {
	TenantCode        string              `json:"tenant_code"`
	InstitutionID     string              `json:"institution_id"`
	EvaluatedAt       string              `json:"evaluated_at"`
	EvaluationID      *string             `json:"evaluation_id"`
	ProfileID         string              `json:"profile_id"`
	ProfileVersion    int                 `json:"profile_version"`
	ProfileStatus     string              `json:"profile_status"`
	SchoolLegalForm   *string             `json:"school_legal_form"`
	Blocked           bool                `json:"blocked"`
	BlockReason       string              `json:"block_reason"`
	Warnings          []string            `json:"warnings"`
	EffectivePolicies []PolicyPackSummary `json:"effective_policies"`
	Capabilities      []PolicyCapability  `json:"capabilities"`
}

// PolicyCutoverPreflightResponse exposes only aggregate migration readiness
// for the authenticated institution. It never exposes rows, legal evidence or
// another tenant's identifiers.
type PolicyCutoverPreflightResponse struct {
	TenantCode                   string `json:"tenant_code"`
	InstitutionID                string `json:"institution_id"`
	Phase                        string `json:"phase"`
	LegacyProfiles               int64  `json:"legacy_profiles"`
	UnmappedProfiles             int64  `json:"unmapped_profiles"`
	LegacyAssignments            int64  `json:"legacy_assignments"`
	UnmappedAssignments          int64  `json:"unmapped_assignments"`
	LegacyOverrides              int64  `json:"legacy_overrides"`
	UnmappedOverrides            int64  `json:"unmapped_overrides"`
	LegacyEvaluations            int64  `json:"legacy_evaluations"`
	UnmappedEvaluations          int64  `json:"unmapped_evaluations"`
	MissingPackProvenance        int64  `json:"missing_pack_provenance"`
	InputsWithoutEffectiveDate   int64  `json:"inputs_without_effective_date"`
	InputsWithMultipleDecisions  int64  `json:"inputs_with_multiple_decisions"`
	ConsumerProvenanceMismatches int64  `json:"consumer_provenance_mismatches"`
	OpenBlockingIssues           int64  `json:"open_blocking_issues"`
	StructurallyReadyForDual     bool   `json:"structurally_ready_for_dual"`
}

type SchoolLocation struct {
	ID              string  `json:"id"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	Active          bool    `json:"active"`
	EffectiveFrom   string  `json:"effective_from"`
	EffectiveTo     *string `json:"effective_to"`
	ExpectedVersion int     `json:"expected_version"`
}

type CreateSchoolLocationRequest struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Address        string  `json:"address"`
	Active         bool    `json:"active"`
	EffectiveFrom  string  `json:"effective_from"`
	EffectiveTo    *string `json:"effective_to"`
	IdempotencyKey string  `json:"idempotency_key"`
}

// UpdateSchoolLocationRequest changes mutable catalog metadata under
// optimistic concurrency. Code and effective_from are immutable identity and
// history fields; a correction that changes them is represented by a new row.
type UpdateSchoolLocationRequest struct {
	ExpectedVersion int     `json:"expected_version"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	Active          *bool   `json:"active"`
	EffectiveTo     *string `json:"effective_to"`
}

type EducationOffering struct {
	ID                 string  `json:"id"`
	Code               string  `json:"code"`
	EducationLevel     string  `json:"education_level"`
	SpecializationCode string  `json:"specialization_code"`
	LanguageCode       string  `json:"language_code"`
	Title              string  `json:"title"`
	Active             bool    `json:"active"`
	EffectiveFrom      string  `json:"effective_from"`
	EffectiveTo        *string `json:"effective_to"`
	ExpectedVersion    int     `json:"expected_version"`
}

type CreateEducationOfferingRequest struct {
	Code               string  `json:"code"`
	EducationLevel     string  `json:"education_level"`
	SpecializationCode string  `json:"specialization_code"`
	LanguageCode       string  `json:"language_code"`
	Title              string  `json:"title"`
	Active             bool    `json:"active"`
	EffectiveFrom      string  `json:"effective_from"`
	EffectiveTo        *string `json:"effective_to"`
	IdempotencyKey     string  `json:"idempotency_key"`
}

// UpdateEducationOfferingRequest changes mutable presentation/lifecycle data
// without rewriting the offering's level, specialization, language or start.
type UpdateEducationOfferingRequest struct {
	ExpectedVersion int     `json:"expected_version"`
	Title           string  `json:"title"`
	Active          *bool   `json:"active"`
	EffectiveTo     *string `json:"effective_to"`
}

type OfferingAuthorization struct {
	ID                      string  `json:"id"`
	OfferingID              string  `json:"offering_id"`
	OfferingCode            string  `json:"offering_code"`
	OfferingTitle           string  `json:"offering_title"`
	LocationID              string  `json:"location_id"`
	LocationCode            string  `json:"location_code"`
	LocationName            string  `json:"location_name"`
	Status                  string  `json:"status"`
	AuthorityName           string  `json:"authority_name"`
	DecisionReference       string  `json:"decision_reference"`
	Capacity                *int    `json:"capacity"`
	CapacityUnit            string  `json:"capacity_unit"`
	Shift                   string  `json:"shift"`
	EffectiveFrom           string  `json:"effective_from"`
	EffectiveTo             *string `json:"effective_to"`
	ExpectedVersion         int     `json:"expected_version"`
	SourceCitation          string  `json:"source_citation"`
	SourceURL               string  `json:"source_url"`
	ReplacesAuthorizationID *string `json:"replaces_authorization_id"`
}

// CreateOfferingAuthorizationRequest creates an initial decision or appends a
// prospective replacement. Scope, actor and the resulting policy decision are
// server-owned and therefore deliberately absent.
type CreateOfferingAuthorizationRequest struct {
	OfferingID              string                  `json:"offering_id"`
	LocationID              string                  `json:"location_id"`
	Status                  string                  `json:"status"`
	AuthorityName           string                  `json:"authority_name"`
	DecisionReference       string                  `json:"decision_reference"`
	Capacity                *int                    `json:"capacity"`
	CapacityUnit            string                  `json:"capacity_unit"`
	Shift                   string                  `json:"shift"`
	EffectiveFrom           string                  `json:"effective_from"`
	EffectiveTo             *string                 `json:"effective_to"`
	Source                  RegulatorySourceRequest `json:"source"`
	ReplacesAuthorizationID *string                 `json:"replaces_authorization_id"`
	ExpectedVersion         *int                    `json:"expected_version"`
	IdempotencyKey          string                  `json:"idempotency_key"`
}
type storedPolicyRuleSet struct {
	Capabilities []storedPolicyCapability `json:"capabilities"`
}

type storedPolicyCapability struct {
	Code              string   `json:"code"`
	Enabled           bool     `json:"enabled"`
	Reason            string   `json:"reason"`
	RequiredDocuments []string `json:"required_documents"`
	RequiredApprovals []string `json:"required_approvals"`
	WizardSteps       []string `json:"wizard_steps"`
}

type assignedPolicyPack struct {
	PolicyPackSummary
	AssignmentID     string
	AssignmentKind   string
	ConfigurableKeys []string
	Rules            storedPolicyRuleSet
}
