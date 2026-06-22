package education

type MobilityCase struct {
	ID                string `json:"id"`
	CaseCode          string `json:"case_code"`
	EmployeeCode      string `json:"employee_code"`
	FullName          string `json:"full_name"`
	SchoolYear        string `json:"school_year"`
	RequestType       string `json:"request_type"`
	Stage             string `json:"stage"`
	Status            string `json:"status"`
	SourceSchool      string `json:"source_school"`
	DestinationSchool string `json:"destination_school"`
	SubmittedOn       string `json:"submitted_on"`
	ReviewedBy        string `json:"reviewed_by"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type MobilityFiltersResponse struct {
	SchoolYears  []string `json:"school_years"`
	RequestTypes []string `json:"request_types"`
	Stages       []string `json:"stages"`
	Statuses     []string `json:"statuses"`
}

type MobilityDashboardResponse struct {
	Stats MobilityStats `json:"stats"`
}

type MobilityStats struct {
	TotalCases          int `json:"total_cases"`
	OpenCases           int `json:"open_cases"`
	ApprovedCases       int `json:"approved_cases"`
	TransferCases       int `json:"transfer_cases"`
	FinalDecisions      int `json:"final_decisions"`
	CommunicatedResults int `json:"communicated_results"`
}

type CreateMobilityCaseRequest struct {
	EmployeeCode      string `json:"employee_code"`
	FullName          string `json:"full_name"`
	SchoolYear        string `json:"school_year"`
	RequestType       string `json:"request_type"`
	Stage             string `json:"stage"`
	Status            string `json:"status"`
	SourceSchool      string `json:"source_school"`
	DestinationSchool string `json:"destination_school"`
	SubmittedOn       string `json:"submitted_on"`
	ReviewedBy        string `json:"reviewed_by"`
	Notes             string `json:"notes"`
}

type MobilityDocument struct {
	ID               string `json:"id"`
	MobilityCaseID   string `json:"mobility_case_id"`
	DocumentCode     string `json:"document_code"`
	DocumentType     string `json:"document_type"`
	StageScope       string `json:"stage_scope"`
	DocumentTitle    string `json:"document_title"`
	RegisteredOn     string `json:"registered_on"`
	SubmittedBy      string `json:"submitted_by"`
	VerifiedBy       string `json:"verified_by"`
	ValidationStatus string `json:"validation_status"`
	Mandatory        bool   `json:"mandatory"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreateMobilityDocumentRequest struct {
	DocumentType     string `json:"document_type"`
	StageScope       string `json:"stage_scope"`
	DocumentTitle    string `json:"document_title"`
	RegisteredOn     string `json:"registered_on"`
	SubmittedBy      string `json:"submitted_by"`
	VerifiedBy       string `json:"verified_by"`
	ValidationStatus string `json:"validation_status"`
	Mandatory        bool   `json:"mandatory"`
	Notes            string `json:"notes"`
}

type MobilityCriterionScore struct {
	ID                string  `json:"id"`
	MobilityCaseID    string  `json:"mobility_case_id"`
	CriterionCode     string  `json:"criterion_code"`
	CriterionLabel    string  `json:"criterion_label"`
	CriterionCategory string  `json:"criterion_category"`
	MaxScore          float64 `json:"max_score"`
	AwardedScore      float64 `json:"awarded_score"`
	EvidenceReference string  `json:"evidence_reference"`
	ValidatedBy       string  `json:"validated_by"`
	Contested         bool    `json:"contested"`
	InstitutionID     string  `json:"institution_id"`
	Notes             string  `json:"notes"`
}

type CreateMobilityCriterionScoreRequest struct {
	CriterionCode     string  `json:"criterion_code"`
	CriterionLabel    string  `json:"criterion_label"`
	CriterionCategory string  `json:"criterion_category"`
	MaxScore          float64 `json:"max_score"`
	AwardedScore      float64 `json:"awarded_score"`
	EvidenceReference string  `json:"evidence_reference"`
	ValidatedBy       string  `json:"validated_by"`
	Contested         bool    `json:"contested"`
	Notes             string  `json:"notes"`
}

type MobilityAppeal struct {
	ID              string `json:"id"`
	MobilityCaseID  string `json:"mobility_case_id"`
	AppealCode      string `json:"appeal_code"`
	SubmittedBy     string `json:"submitted_by"`
	SubmittedOn     string `json:"submitted_on"`
	Status          string `json:"status"`
	Grounds         string `json:"grounds"`
	HearingOn       string `json:"hearing_on"`
	ResolvedOn      string `json:"resolved_on"`
	DecisionSummary string `json:"decision_summary"`
	InstitutionID   string `json:"institution_id"`
	Notes           string `json:"notes"`
}

type CreateMobilityAppealRequest struct {
	SubmittedBy     string `json:"submitted_by"`
	SubmittedOn     string `json:"submitted_on"`
	Status          string `json:"status"`
	Grounds         string `json:"grounds"`
	HearingOn       string `json:"hearing_on"`
	ResolvedOn      string `json:"resolved_on"`
	DecisionSummary string `json:"decision_summary"`
	Notes           string `json:"notes"`
}

type MobilityFinalDecision struct {
	ID              string `json:"id"`
	MobilityCaseID  string `json:"mobility_case_id"`
	DecisionCode    string `json:"decision_code"`
	DecisionType    string `json:"decision_type"`
	Outcome         string `json:"outcome"`
	ApprovedOn      string `json:"approved_on"`
	EffectiveFrom   string `json:"effective_from"`
	PanelName       string `json:"panel_name"`
	LegalBasis      string `json:"legal_basis"`
	DestinationUnit string `json:"destination_unit"`
	InstitutionID   string `json:"institution_id"`
	Notes           string `json:"notes"`
}

type CreateMobilityFinalDecisionRequest struct {
	DecisionType    string `json:"decision_type"`
	Outcome         string `json:"outcome"`
	ApprovedOn      string `json:"approved_on"`
	EffectiveFrom   string `json:"effective_from"`
	PanelName       string `json:"panel_name"`
	LegalBasis      string `json:"legal_basis"`
	DestinationUnit string `json:"destination_unit"`
	Notes           string `json:"notes"`
}

type MobilityResultIssue struct {
	ID                string `json:"id"`
	MobilityCaseID    string `json:"mobility_case_id"`
	IssueCode         string `json:"issue_code"`
	DocumentType      string `json:"document_type"`
	RecipientName     string `json:"recipient_name"`
	RecipientRole     string `json:"recipient_role"`
	DeliveryChannel   string `json:"delivery_channel"`
	DeliveryStatus    string `json:"delivery_status"`
	IssuedOn          string `json:"issued_on"`
	DeliveredOn       string `json:"delivered_on"`
	RegistryReference string `json:"registry_reference"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type CreateMobilityResultIssueRequest struct {
	DocumentType      string `json:"document_type"`
	RecipientName     string `json:"recipient_name"`
	RecipientRole     string `json:"recipient_role"`
	DeliveryChannel   string `json:"delivery_channel"`
	DeliveryStatus    string `json:"delivery_status"`
	IssuedOn          string `json:"issued_on"`
	DeliveredOn       string `json:"delivered_on"`
	RegistryReference string `json:"registry_reference"`
	Notes             string `json:"notes"`
}
