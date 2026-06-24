package education

type PortfolioRecord struct {
	ID                   string `json:"id"`
	PortfolioCode        string `json:"portfolio_code"`
	OwnerName            string `json:"owner_name"`
	OwnerRole            string `json:"owner_role"`
	SchoolYear           string `json:"school_year"`
	Status               string `json:"status"`
	SectionCount         int    `json:"section_count"`
	LastUpdatedOn        string `json:"last_updated_on"`
	RetentionUntil       string `json:"retention_until"`
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
	OwnerName            string `json:"owner_name"`
	OwnerRole            string `json:"owner_role"`
	SchoolYear           string `json:"school_year"`
	Status               string `json:"status"`
	SectionCount         int    `json:"section_count"`
	LastUpdatedOn        string `json:"last_updated_on"`
	RetentionUntil       string `json:"retention_until"`
	TransferStatus       string `json:"transfer_status"`
	AuthenticityDeclared bool   `json:"authenticity_declared"`
	ConsentCaptured      bool   `json:"consent_captured"`
	Custodian            string `json:"custodian"`
	Notes                string `json:"notes"`
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
	ID                 string `json:"id"`
	PortfolioID        string `json:"portfolio_id"`
	SectionCode        string `json:"section_code"`
	ComponentCode      string `json:"component_code"`
	DocumentTitle      string `json:"document_title"`
	SourceScope        string `json:"source_scope"`
	EvidenceType       string `json:"evidence_type"`
	IssuedOn           string `json:"issued_on"`
	AddedOn            string `json:"added_on"`
	ChronologicalIndex int    `json:"chronological_index"`
	SensitiveData      bool   `json:"sensitive_data"`
	AuthenticityStatus string `json:"authenticity_status"`
	FileReference      string `json:"file_reference"`
	InstitutionID      string `json:"institution_id"`
	Notes              string `json:"notes"`
}

type CreatePortfolioDocumentRequest struct {
	SectionCode        string `json:"section_code"`
	ComponentCode      string `json:"component_code"`
	DocumentTitle      string `json:"document_title"`
	SourceScope        string `json:"source_scope"`
	EvidenceType       string `json:"evidence_type"`
	IssuedOn           string `json:"issued_on"`
	AddedOn            string `json:"added_on"`
	ChronologicalIndex int    `json:"chronological_index"`
	SensitiveData      bool   `json:"sensitive_data"`
	AuthenticityStatus string `json:"authenticity_status"`
	FileReference      string `json:"file_reference"`
	Notes              string `json:"notes"`
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
	ID                     string `json:"id"`
	PortfolioID            string `json:"portfolio_id"`
	TransferCode           string `json:"transfer_code"`
	TransferType           string `json:"transfer_type"`
	SourceInstitution      string `json:"source_institution"`
	DestinationInstitution string `json:"destination_institution"`
	Status                 string `json:"status"`
	HandoverOn             string `json:"handover_on"`
	ReceivedOn             string `json:"received_on"`
	HandoverBy             string `json:"handover_by"`
	ReceivedBy             string `json:"received_by"`
	InstitutionID          string `json:"institution_id"`
	Notes                  string `json:"notes"`
}

type CreatePortfolioTransferEventRequest struct {
	TransferType           string `json:"transfer_type"`
	SourceInstitution      string `json:"source_institution"`
	DestinationInstitution string `json:"destination_institution"`
	Status                 string `json:"status"`
	HandoverOn             string `json:"handover_on"`
	ReceivedOn             string `json:"received_on"`
	HandoverBy             string `json:"handover_by"`
	ReceivedBy             string `json:"received_by"`
	Notes                  string `json:"notes"`
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
