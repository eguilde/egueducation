package gdpr

type RetentionPolicy struct {
	ID             string `json:"id"`
	PolicyCode     string `json:"policy_code"`
	DomainCode     string `json:"domain_code"`
	RecordCategory string `json:"record_category"`
	RetentionYears int    `json:"retention_years"`
	LegalBasis     string `json:"legal_basis"`
	Status         string `json:"status"`
	ReviewDueOn    string `json:"review_due_on"`
	OwnerName      string `json:"owner_name"`
	InstitutionID  string `json:"institution_id"`
	Notes          string `json:"notes"`
}

type SubjectRequest struct {
	ID                    string `json:"id"`
	RequestCode           string `json:"request_code"`
	SubjectName           string `json:"subject_name"`
	RequestType           string `json:"request_type"`
	Status                string `json:"status"`
	SubmittedOn           string `json:"submitted_on"`
	DueOn                 string `json:"due_on"`
	HandledBy             string `json:"handled_by"`
	SourceModule          string `json:"source_module"`
	AnonymizationRequired bool   `json:"anonymization_required"`
	InstitutionID         string `json:"institution_id"`
	Notes                 string `json:"notes"`
}

type SubjectExport struct {
	ID             string `json:"id"`
	ExportCode     string `json:"export_code"`
	RequestID      string `json:"request_id"`
	SubjectName    string `json:"subject_name"`
	SourceModule   string `json:"source_module"`
	Status         string `json:"status"`
	ExportFormat   string `json:"export_format"`
	ApprovedBy     string `json:"approved_by"`
	ApprovedOn     string `json:"approved_on"`
	GeneratedOn    string `json:"generated_on"`
	PackageSummary string `json:"package_summary"`
	InstitutionID  string `json:"institution_id"`
	Notes          string `json:"notes"`
}

type PublicationReview struct {
	ID                  string `json:"id"`
	ReviewCode          string `json:"review_code"`
	SourceModule        string `json:"source_module"`
	SourceRecordID      string `json:"source_record_id"`
	SourceLabel         string `json:"source_label"`
	AnonymizationStatus string `json:"anonymization_status"`
	PublicationStatus   string `json:"publication_status"`
	ReviewedBy          string `json:"reviewed_by"`
	ReviewedOn          string `json:"reviewed_on"`
	LegalBasis          string `json:"legal_basis"`
	InstitutionID       string `json:"institution_id"`
	Notes               string `json:"notes"`
}

type DashboardResponse struct {
	Stats DashboardStats `json:"stats"`
}

type DashboardStats struct {
	ActivePolicies     int `json:"active_policies"`
	PendingRequests    int `json:"pending_requests"`
	OverdueRequests    int `json:"overdue_requests"`
	AnonymizationCases int `json:"anonymization_cases"`
}

type ExportDashboardResponse struct {
	Stats struct {
		TotalExports     int `json:"total_exports"`
		PendingApproval  int `json:"pending_approval"`
		GeneratedExports int `json:"generated_exports"`
		DeliveredExports int `json:"delivered_exports"`
	} `json:"stats"`
}

type PublicationDashboardResponse struct {
	Stats struct {
		TotalReviews         int `json:"total_reviews"`
		PendingAnonymization int `json:"pending_anonymization"`
		ReadyForPublication  int `json:"ready_for_publication"`
		PublishedItems       int `json:"published_items"`
	} `json:"stats"`
}

type Settings struct {
	PublicationAnonymizationRequired bool `json:"publication_anonymization_required"`
	SubjectExportRequiresApproval    bool `json:"subject_export_requires_approval"`
	DefaultResponseSLADays           int  `json:"default_response_sla_days"`
	RetentionReviewNoticeDays        int  `json:"retention_review_notice_days"`
	PortfolioConsentRequired         bool `json:"portfolio_consent_required"`
	PortfolioAuthenticityRequired    bool `json:"portfolio_authenticity_required"`
}

type ConfigResponse struct {
	Settings Settings `json:"settings"`
	Catalogs struct {
		Domains       []string `json:"domains"`
		PolicyStatus  []string `json:"policy_status"`
		RequestTypes  []string `json:"request_types"`
		RequestStatus []string `json:"request_status"`
		SourceModules []string `json:"source_modules"`
	} `json:"catalogs"`
}

type RetentionPolicyFiltersResponse struct {
	Domains  []string `json:"domains"`
	Statuses []string `json:"statuses"`
}

type SubjectRequestFiltersResponse struct {
	RequestTypes  []string `json:"request_types"`
	Statuses      []string `json:"statuses"`
	SourceModules []string `json:"source_modules"`
}

type SubjectExportFiltersResponse struct {
	Statuses      []string `json:"statuses"`
	ExportFormats []string `json:"export_formats"`
	SourceModules []string `json:"source_modules"`
}

type PublicationReviewFiltersResponse struct {
	SourceModules         []string `json:"source_modules"`
	AnonymizationStatuses []string `json:"anonymization_statuses"`
	PublicationStatuses   []string `json:"publication_statuses"`
}

type CreateRetentionPolicyRequest struct {
	DomainCode     string `json:"domain_code"`
	RecordCategory string `json:"record_category"`
	RetentionYears int    `json:"retention_years"`
	LegalBasis     string `json:"legal_basis"`
	Status         string `json:"status"`
	ReviewDueOn    string `json:"review_due_on"`
	OwnerName      string `json:"owner_name"`
	Notes          string `json:"notes"`
}

type CreateSubjectRequestRequest struct {
	SubjectName           string `json:"subject_name"`
	RequestType           string `json:"request_type"`
	Status                string `json:"status"`
	SubmittedOn           string `json:"submitted_on"`
	DueOn                 string `json:"due_on"`
	HandledBy             string `json:"handled_by"`
	SourceModule          string `json:"source_module"`
	AnonymizationRequired bool   `json:"anonymization_required"`
	Notes                 string `json:"notes"`
}

type CreateSubjectExportRequest struct {
	RequestID      string `json:"request_id"`
	SubjectName    string `json:"subject_name"`
	SourceModule   string `json:"source_module"`
	Status         string `json:"status"`
	ExportFormat   string `json:"export_format"`
	ApprovedBy     string `json:"approved_by"`
	ApprovedOn     string `json:"approved_on"`
	GeneratedOn    string `json:"generated_on"`
	PackageSummary string `json:"package_summary"`
	Notes          string `json:"notes"`
}

type CreatePublicationReviewRequest struct {
	SourceModule        string `json:"source_module"`
	SourceRecordID      string `json:"source_record_id"`
	SourceLabel         string `json:"source_label"`
	AnonymizationStatus string `json:"anonymization_status"`
	PublicationStatus   string `json:"publication_status"`
	ReviewedBy          string `json:"reviewed_by"`
	ReviewedOn          string `json:"reviewed_on"`
	LegalBasis          string `json:"legal_basis"`
	Notes               string `json:"notes"`
}
