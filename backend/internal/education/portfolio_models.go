package education

type PortfolioRecord struct {
	ID                   string `json:"id"`
	PortfolioCode        string `json:"portfolio_code"`
	OwnerUserID          string `json:"owner_user_id,omitempty"`
	OwnerPersonnelID     string `json:"owner_personnel_id,omitempty"`
	OwnerName            string `json:"owner_name"`
	OwnerRole            string `json:"owner_role"`
	SchoolYear           string `json:"school_year"`
	Status               string `json:"status"`
	SectionCount         int    `json:"section_count"`
	LastUpdatedOn        string `json:"last_updated_on"`
	RetentionUntil       string `json:"retention_until"`
	ActivityCeasedOn     string `json:"activity_ceased_on,omitempty"`
	RetentionPeriodDays  int    `json:"retention_period_days"`
	LegalHoldActive      bool   `json:"legal_hold_active"`
	LegalHoldReason      string `json:"legal_hold_reason,omitempty"`
	WithdrawnAt          string `json:"withdrawn_at,omitempty"`
	WithdrawalReason     string `json:"withdrawal_reason,omitempty"`
	AppliedProcedureID   string `json:"applied_procedure_id,omitempty"`
	TransferStatus       string `json:"transfer_status"`
	AuthenticityDeclared bool   `json:"authenticity_declared"`
	ConsentCaptured      bool   `json:"consent_captured"`
	Custodian            string `json:"custodian"`
	InstitutionID        string `json:"institution_id"`
	Notes                string `json:"notes"`
}

type PortfolioFiltersResponse struct {
	SchoolYears    []string `json:"school_years"`
	Statuses       []string `json:"statuses"`
	TransferStatus []string `json:"transfer_statuses"`
}

type PortfolioDashboardResponse struct {
	Stats PortfolioStats `json:"stats"`
}

type PortfolioStats struct {
	TotalPortfolios     int `json:"total_portfolios"`
	ValidatedPortfolios int `json:"validated_portfolios"`
	TransferPortfolios  int `json:"transfer_portfolios"`
	DeclaredPortfolios  int `json:"declared_portfolios"`
}

type CreatePortfolioRecordRequest struct {
	// OwnerUserID is accepted only for an institution administrator. Own-portfolio
	// endpoints ignore it and derive the immutable owner from the session.
	OwnerUserID          string `json:"owner_user_id,omitempty"`
	OwnerPersonnelID     string `json:"owner_personnel_id,omitempty"`
	OwnerName            string `json:"owner_name"`
	OwnerRole            string `json:"owner_role"`
	SchoolYear           string `json:"school_year"`
	Status               string `json:"status"`
	SectionCount         int    `json:"section_count"`
	LastUpdatedOn        string `json:"last_updated_on"`
	TransferStatus       string `json:"transfer_status"`
	AuthenticityDeclared bool   `json:"authenticity_declared"`
	ConsentCaptured      bool   `json:"consent_captured"`
	Custodian            string `json:"custodian"`
	Notes                string `json:"notes"`
}

// UpdatePortfolioRecordRequest is deliberately separate from creation. The
// owner identifiers are optional assertions of the immutable stored pairing;
// lifecycle, retention and transfer changes remain dedicated commands.
type UpdatePortfolioRecordRequest struct {
	OwnerUserID          string `json:"owner_user_id,omitempty"`
	OwnerPersonnelID     string `json:"owner_personnel_id,omitempty"`
	OwnerName            string `json:"owner_name"`
	OwnerRole            string `json:"owner_role"`
	SchoolYear           string `json:"school_year"`
	Status               string `json:"status"`
	SectionCount         int    `json:"section_count"`
	LastUpdatedOn        string `json:"last_updated_on"`
	TransferStatus       string `json:"transfer_status"`
	AuthenticityDeclared bool   `json:"authenticity_declared"`
	ConsentCaptured      bool   `json:"consent_captured"`
	Custodian            string `json:"custodian"`
	Notes                string `json:"notes"`
}

// PortfolioCessationRequest is an institution-controlled lifecycle command.
// Retention is calculated in persistence from this event; it is never an
// input supplied by a portfolio client.
type PortfolioCessationRequest struct {
	ActivityCeasedOn string `json:"activity_ceased_on"`
	Reason           string `json:"reason"`
}

// PortfolioLegalHoldRequest records or releases a legal hold. A route is
// deliberately expected to protect this command with a dedicated lifecycle
// permission once the public contract is published.
type PortfolioLegalHoldRequest struct {
	Active bool   `json:"active"`
	Reason string `json:"reason"`
}

// OwnPortfolioRequest is the public command contract for a portfolio owner.
// Identity, lifecycle, custody, transfer and retention are server-controlled.
type OwnPortfolioRequest struct {
	SchoolYear    string `json:"school_year"`
	LastUpdatedOn string `json:"last_updated_on"`
	Notes         string `json:"notes"`
}

type EducationRequirement struct {
	ID                   string `json:"id"`
	Domain               string `json:"domain"`
	Code                 string `json:"code"`
	TitleRO              string `json:"title_ro"`
	TitleEN              string `json:"title_en"`
	SourceRef            string `json:"source_ref"`
	RequirementType      string `json:"requirement_type"`
	ImplementationStatus string `json:"implementation_status"`
	Priority             int    `json:"priority"`
	Notes                string `json:"notes"`
}

type PortfolioSection struct {
	ID               string   `json:"id"`
	SectionCode      string   `json:"section_code"`
	ComponentCode    string   `json:"component_code"`
	LabelRO          string   `json:"label_ro"`
	LabelEN          string   `json:"label_en"`
	ExampleDocuments []string `json:"example_documents"`
	Required         bool     `json:"required"`
	SensitiveData    bool     `json:"sensitive_data"`
	RetentionRule    string   `json:"retention_rule"`
	SortOrder        int      `json:"sort_order"`
	Active           bool     `json:"active"`
}

type PortfolioDocument struct {
	ID                 string   `json:"id"`
	PortfolioID        string   `json:"portfolio_id"`
	SectionCode        string   `json:"section_code"`
	ComponentCode      string   `json:"component_code"`
	DocumentTitle      string   `json:"document_title"`
	Description        string   `json:"description"`
	SchoolYear         string   `json:"school_year"`
	SubjectDiscipline  string   `json:"subject_discipline"`
	ApplicableClass    string   `json:"applicable_class"`
	Competencies       []string `json:"competencies"`
	SourceScope        string   `json:"source_scope"`
	EvidenceType       string   `json:"evidence_type"`
	IssuedOn           string   `json:"issued_on"`
	AddedOn            string   `json:"added_on"`
	ChronologicalIndex int      `json:"chronological_index"`
	SensitiveData      bool     `json:"sensitive_data"`
	AuthenticityStatus string   `json:"authenticity_status"`
	FileReference      string   `json:"file_reference"`
	ArchiveDocumentID  string   `json:"archive_document_id"`
	ArchiveVersionID   string   `json:"archive_version_id"`
	ArchiveVersionNo   int      `json:"archive_version_no"`
	ArchiveSHA256      string   `json:"archive_sha256"`
	InstitutionID      string   `json:"institution_id"`
	Notes              string   `json:"notes"`
}

type CreatePortfolioDocumentRequest struct {
	SectionCode        string   `json:"section_code"`
	ComponentCode      string   `json:"component_code"`
	DocumentTitle      string   `json:"document_title"`
	Description        string   `json:"description"`
	SchoolYear         string   `json:"school_year"`
	SubjectDiscipline  string   `json:"subject_discipline"`
	ApplicableClass    string   `json:"applicable_class"`
	Competencies       []string `json:"competencies"`
	SourceScope        string   `json:"source_scope"`
	EvidenceType       string   `json:"evidence_type"`
	IssuedOn           string   `json:"issued_on"`
	AddedOn            string   `json:"added_on"`
	ChronologicalIndex int      `json:"chronological_index"`
	SensitiveData      bool     `json:"sensitive_data"`
	AuthenticityStatus string   `json:"authenticity_status"`
	FileReference      string   `json:"file_reference"`
	Notes              string   `json:"notes"`
}

// OwnPortfolioDocumentRequest excludes institution-controlled authenticity
// and provenance fields. Own-document handlers set those values server-side.
type OwnPortfolioDocumentRequest struct {
	SectionCode        string   `json:"section_code"`
	ComponentCode      string   `json:"component_code"`
	DocumentTitle      string   `json:"document_title"`
	Description        string   `json:"description"`
	SchoolYear         string   `json:"school_year"`
	SubjectDiscipline  string   `json:"subject_discipline"`
	ApplicableClass    string   `json:"applicable_class"`
	Competencies       []string `json:"competencies"`
	EvidenceType       string   `json:"evidence_type"`
	IssuedOn           string   `json:"issued_on"`
	AddedOn            string   `json:"added_on"`
	ChronologicalIndex int      `json:"chronological_index"`
	SensitiveData      bool     `json:"sensitive_data"`
	FileReference      string   `json:"file_reference"`
	Notes              string   `json:"notes"`
}

type PortfolioDocumentVersion struct {
	VersionNo  int            `json:"version_no"`
	ChangeType string         `json:"change_type"`
	ChangedBy  string         `json:"changed_by"`
	ChangedAt  string         `json:"changed_at"`
	Reason     string         `json:"reason"`
	Snapshot   map[string]any `json:"snapshot"`
}

// PortfolioArchiveAttachment is the deliberately minimal eArhiva projection
// exposed to a portfolio owner when choosing existing evidence. It never
// exposes storage paths, metadata, OCR text, or archive workflow state.
type PortfolioArchiveAttachment struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	CurrentVersionNo int    `json:"current_version_no"`
}

type PortfolioArchiveAttachmentGrant struct {
	ID                string `json:"id"`
	ArchiveDocumentID string `json:"archive_document_id"`
	DocumentTitle     string `json:"document_title"`
	GranteeUserID     string `json:"grantee_user_id"`
	GranteeName       string `json:"grantee_name"`
	GrantedByUserID   string `json:"granted_by_user_id,omitempty"`
	CreatedAt         string `json:"created_at"`
}

type CreatePortfolioArchiveAttachmentGrantRequest struct {
	ArchiveDocumentID string `json:"archive_document_id"`
	GranteeUserID     string `json:"grantee_user_id"`
}

type PortfolioChecklistItem struct {
	ID               string `json:"id"`
	PortfolioID      string `json:"portfolio_id"`
	RequirementCode  string `json:"requirement_code"`
	RequirementLabel string `json:"requirement_label"`
	SectionCode      string `json:"section_code"`
	SourceScope      string `json:"source_scope"`
	Mandatory        bool   `json:"mandatory"`
	Status           string `json:"status"`
	DocumentCount    int    `json:"document_count"`
	LastCheckedOn    string `json:"last_checked_on"`
	CheckedBy        string `json:"checked_by"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreatePortfolioChecklistItemRequest struct {
	RequirementCode  string `json:"requirement_code"`
	RequirementLabel string `json:"requirement_label"`
	SectionCode      string `json:"section_code"`
	SourceScope      string `json:"source_scope"`
	Mandatory        bool   `json:"mandatory"`
	Status           string `json:"status"`
	DocumentCount    int    `json:"document_count"`
	LastCheckedOn    string `json:"last_checked_on"`
	CheckedBy        string `json:"checked_by"`
	Notes            string `json:"notes"`
}

type PortfolioTransferEvent struct {
	ID                       string `json:"id"`
	PortfolioID              string `json:"portfolio_id"`
	TransferCode             string `json:"transfer_code"`
	TransferType             string `json:"transfer_type"`
	SourceInstitution        string `json:"source_institution"`
	DestinationInstitution   string `json:"destination_institution"`
	Status                   string `json:"status"`
	HandoverOn               string `json:"handover_on"`
	ReceivedOn               string `json:"received_on"`
	HandoverBy               string `json:"handover_by"`
	ReceivedBy               string `json:"received_by"`
	InstitutionID            string `json:"institution_id"`
	Notes                    string `json:"notes"`
	WithdrawnAt              string `json:"withdrawn_at,omitempty"`
	WithdrawalReason         string `json:"withdrawal_reason,omitempty"`
	RoutingVersion           int    `json:"routing_version"`
	SourceTenantCode         string `json:"source_tenant_code,omitempty"`
	SourceInstitutionID      string `json:"source_institution_id,omitempty"`
	DestinationTenantCode    string `json:"destination_tenant_code,omitempty"`
	DestinationInstitutionID string `json:"destination_institution_id,omitempty"`
	ExportManifestID         string `json:"export_manifest_id,omitempty"`
	SentAt                   string `json:"sent_at,omitempty"`
	SentBySubject            string `json:"sent_by_subject,omitempty"`
	ReceivedAt               string `json:"received_at,omitempty"`
	ReceivedBySubject        string `json:"received_by_subject,omitempty"`
	ClosedAt                 string `json:"closed_at,omitempty"`
	ClosedBySubject          string `json:"closed_by_subject,omitempty"`
}

type PortfolioTransferDestination struct {
	TenantCode    string `json:"tenant_code"`
	InstitutionID string `json:"institution_id"`
	DisplayName   string `json:"display_name"`
	ShortName     string `json:"short_name"`
}

// CreateIntertenantPortfolioTransferRequest contains only source-authored
// draft metadata. Route labels, status and all evidence provenance are derived
// by the server from the authenticated tenant and actor.
type CreateIntertenantPortfolioTransferRequest struct {
	TransferType          string `json:"transfer_type"`
	HandoverOn            string `json:"handover_on"`
	Notes                 string `json:"notes"`
	DestinationTenantCode string `json:"destination_tenant_code"`
}

// UpdatePreparedPortfolioTransferRequest is intentionally narrower than the
// create contract: a prepared transfer cannot be re-routed by a browser.
type UpdatePreparedPortfolioTransferRequest struct {
	TransferType string `json:"transfer_type"`
	HandoverOn   string `json:"handover_on"`
	Notes        string `json:"notes"`
}

type PortfolioReviewEvent struct {
	ID               string `json:"id"`
	PortfolioID      string `json:"portfolio_id"`
	ReviewCode       string `json:"review_code"`
	ReviewStage      string `json:"review_stage"`
	Outcome          string `json:"outcome"`
	ReviewerName     string `json:"reviewer_name"`
	ReviewedOn       string `json:"reviewed_on"`
	MissingDocuments int    `json:"missing_documents"`
	ComplianceScore  int    `json:"compliance_score"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreatePortfolioReviewEventRequest struct {
	ReviewStage      string `json:"review_stage"`
	Outcome          string `json:"outcome"`
	ReviewerName     string `json:"reviewer_name"`
	ReviewedOn       string `json:"reviewed_on"`
	MissingDocuments int    `json:"missing_documents"`
	ComplianceScore  int    `json:"compliance_score"`
	Notes            string `json:"notes"`
}

type PortfolioValorificationEvent struct {
	ID                 string `json:"id"`
	PortfolioID        string `json:"portfolio_id"`
	ValorificationCode string `json:"valorification_code"`
	Scope              string `json:"scope"`
	Status             string `json:"status"`
	RequestedBy        string `json:"requested_by"`
	TargetInstitution  string `json:"target_institution"`
	TargetReference    string `json:"target_reference"`
	StartedOn          string `json:"started_on"`
	CompletedOn        string `json:"completed_on"`
	InstitutionID      string `json:"institution_id"`
	Notes              string `json:"notes"`
	WithdrawnAt        string `json:"withdrawn_at,omitempty"`
	WithdrawalReason   string `json:"withdrawal_reason,omitempty"`
}

type CreatePortfolioValorificationEventRequest struct {
	Scope             string `json:"scope"`
	Status            string `json:"status"`
	RequestedBy       string `json:"requested_by"`
	TargetInstitution string `json:"target_institution"`
	TargetReference   string `json:"target_reference"`
	StartedOn         string `json:"started_on"`
	CompletedOn       string `json:"completed_on"`
	Notes             string `json:"notes"`
}

type PortfolioOpisEntry struct {
	ID                 string `json:"id"`
	PortfolioID        string `json:"portfolio_id"`
	SectionCode        string `json:"section_code"`
	ComponentCode      string `json:"component_code"`
	EntryTitle         string `json:"entry_title"`
	SourceScope        string `json:"source_scope"`
	ChronologicalIndex int    `json:"chronological_index"`
	DocumentReference  string `json:"document_reference"`
	IncludedInTransfer bool   `json:"included_in_transfer"`
	CheckedOn          string `json:"checked_on"`
	CheckedBy          string `json:"checked_by"`
	InstitutionID      string `json:"institution_id"`
	Notes              string `json:"notes"`
}

type CreatePortfolioOpisEntryRequest struct {
	SectionCode        string `json:"section_code"`
	ComponentCode      string `json:"component_code"`
	EntryTitle         string `json:"entry_title"`
	SourceScope        string `json:"source_scope"`
	ChronologicalIndex int    `json:"chronological_index"`
	DocumentReference  string `json:"document_reference"`
	IncludedInTransfer bool   `json:"included_in_transfer"`
	CheckedOn          string `json:"checked_on"`
	CheckedBy          string `json:"checked_by"`
	Notes              string `json:"notes"`
}

type PortfolioCustodyEvent struct {
	ID                  string `json:"id"`
	PortfolioID         string `json:"portfolio_id"`
	EventType           string `json:"event_type"`
	HolderName          string `json:"holder_name"`
	HolderRole          string `json:"holder_role"`
	LocationLabel       string `json:"location_label"`
	AccessReason        string `json:"access_reason"`
	StartedOn           string `json:"started_on"`
	EndedOn             string `json:"ended_on"`
	AccessMode          string `json:"access_mode"`
	SensitiveDataAccess bool   `json:"sensitive_data_access"`
	InstitutionID       string `json:"institution_id"`
	Notes               string `json:"notes"`
}

type CreatePortfolioCustodyEventRequest struct {
	EventType           string `json:"event_type"`
	HolderName          string `json:"holder_name"`
	HolderRole          string `json:"holder_role"`
	LocationLabel       string `json:"location_label"`
	AccessReason        string `json:"access_reason"`
	StartedOn           string `json:"started_on"`
	EndedOn             string `json:"ended_on"`
	AccessMode          string `json:"access_mode"`
	SensitiveDataAccess bool   `json:"sensitive_data_access"`
	Notes               string `json:"notes"`
}
