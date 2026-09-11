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
	ExpectedVersion        int      `json:"expected_version"`
	Status                 string   `json:"status"`
	SchoolLegalForm        string   `json:"school_legal_form"`
	RegulatoryProfile      string   `json:"regulatory_profile"`
	AuthorizationStatus    string   `json:"authorization_status"`
	AccreditationReference string   `json:"accreditation_reference"`
	AuthorizedLevels       []string `json:"authorized_levels"`
	HasLegalPersonality    bool     `json:"has_legal_personality"`
	TaxIdentifier          string   `json:"tax_identifier"`
	FounderName            string   `json:"founder_name"`
	FunderName             string   `json:"funder_name"`
	BudgetAuthorityName    string   `json:"budget_authority_name"`
	IsContractingAuthority bool     `json:"is_contracting_authority"`
	AccountingProfile      string   `json:"accounting_profile"`
	ProcurementProfile     string   `json:"procurement_profile"`
	PayrollProfile         string   `json:"payroll_profile"`
	VATProfile             string   `json:"vat_profile"`
	TreasuryRequired       bool     `json:"treasury_required"`
	PublicFunding          bool     `json:"public_funding"`
	ProgramCodes           []string `json:"program_codes"`
	EffectiveFrom          string   `json:"effective_from"`
	EffectiveTo            *string  `json:"effective_to"`
	SourceReference        string   `json:"source_reference"`
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
