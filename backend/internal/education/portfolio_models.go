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
