package education

type MeritGrant struct {
	ID            string  `json:"id"`
	GrantCode     string  `json:"grant_code"`
	FullName      string  `json:"full_name"`
	RoleTitle     string  `json:"role_title"`
	SchoolYear    string  `json:"school_year"`
	Category      string  `json:"category"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	CommitteeName string  `json:"committee_name"`
	DecisionDate  string  `json:"decision_date"`
	Funded        bool    `json:"funded"`
	InstitutionID string  `json:"institution_id"`
	Notes         string  `json:"notes"`
}

type MeritGrantFiltersResponse struct {
	SchoolYears []string `json:"school_years"`
	Categories  []string `json:"categories"`
	Statuses    []string `json:"statuses"`
}

type MeritGrantDashboardResponse struct {
	Stats MeritGrantStats `json:"stats"`
}

type MeritGrantStats struct {
	TotalRecords        int     `json:"total_records"`
	ApprovedRecords     int     `json:"approved_records"`
	FundedRecords       int     `json:"funded_records"`
	AverageScore        float64 `json:"average_score"`
	FinalDecisions      int     `json:"final_decisions"`
	CommunicatedResults int     `json:"communicated_results"`
}

type CreateMeritGrantRequest struct {
	FullName      string  `json:"full_name"`
	RoleTitle     string  `json:"role_title"`
	SchoolYear    string  `json:"school_year"`
	Category      string  `json:"category"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	CommitteeName string  `json:"committee_name"`
	DecisionDate  string  `json:"decision_date"`
	Funded        bool    `json:"funded"`
	Notes         string  `json:"notes"`
}

type MeritDocument struct {
	ID               string `json:"id"`
	GrantID          string `json:"grant_id"`
	DocumentCode     string `json:"document_code"`
	DocumentType     string `json:"document_type"`
	DocumentTitle    string `json:"document_title"`
	RegisteredOn     string `json:"registered_on"`
	SubmittedBy      string `json:"submitted_by"`
	ValidationStatus string `json:"validation_status"`
	Mandatory        bool   `json:"mandatory"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreateMeritDocumentRequest struct {
	DocumentType     string `json:"document_type"`
	DocumentTitle    string `json:"document_title"`
	RegisteredOn     string `json:"registered_on"`
	SubmittedBy      string `json:"submitted_by"`
	ValidationStatus string `json:"validation_status"`
	Mandatory        bool   `json:"mandatory"`
	Notes            string `json:"notes"`
}

type MeritCriterionScore struct {
	ID                string  `json:"id"`
	GrantID           string  `json:"grant_id"`
	CriterionCode     string  `json:"criterion_code"`
	CriterionLabel    string  `json:"criterion_label"`
	CriterionCategory string  `json:"criterion_category"`
	PanelStage        string  `json:"panel_stage"`
	MaxScore          float64 `json:"max_score"`
	AwardedScore      float64 `json:"awarded_score"`
	ReviewerName      string  `json:"reviewer_name"`
	EvidenceReference string  `json:"evidence_reference"`
	Contested         bool    `json:"contested"`
	InstitutionID     string  `json:"institution_id"`
	Notes             string  `json:"notes"`
}

type CreateMeritCriterionScoreRequest struct {
	CriterionCode     string  `json:"criterion_code"`
	CriterionLabel    string  `json:"criterion_label"`
	CriterionCategory string  `json:"criterion_category"`
	PanelStage        string  `json:"panel_stage"`
	MaxScore          float64 `json:"max_score"`
	AwardedScore      float64 `json:"awarded_score"`
	ReviewerName      string  `json:"reviewer_name"`
	EvidenceReference string  `json:"evidence_reference"`
	Contested         bool    `json:"contested"`
	Notes             string  `json:"notes"`
}

type MeritAppeal struct {
	ID              string `json:"id"`
	GrantID         string `json:"grant_id"`
	AppealCode      string `json:"appeal_code"`
	SubmittedBy     string `json:"submitted_by"`
	SubmittedOn     string `json:"submitted_on"`
	Status          string `json:"status"`
	Grounds         string `json:"grounds"`
	ResolvedOn      string `json:"resolved_on"`
	DecisionSummary string `json:"decision_summary"`
	InstitutionID   string `json:"institution_id"`
	Notes           string `json:"notes"`
}

type CreateMeritAppealRequest struct {
	SubmittedBy     string `json:"submitted_by"`
	SubmittedOn     string `json:"submitted_on"`
	Status          string `json:"status"`
	Grounds         string `json:"grounds"`
	ResolvedOn      string `json:"resolved_on"`
	DecisionSummary string `json:"decision_summary"`
	Notes           string `json:"notes"`
}

type MeritFinalDecision struct {
	ID            string `json:"id"`
	GrantID       string `json:"grant_id"`
	DecisionCode  string `json:"decision_code"`
	DecisionStage string `json:"decision_stage"`
	Outcome       string `json:"outcome"`
	ApprovedOn    string `json:"approved_on"`
	EffectiveFrom string `json:"effective_from"`
	PanelName     string `json:"panel_name"`
	Funded        bool   `json:"funded"`
	LegalBasis    string `json:"legal_basis"`
	InstitutionID string `json:"institution_id"`
	Notes         string `json:"notes"`
}

type CreateMeritFinalDecisionRequest struct {
	DecisionStage string `json:"decision_stage"`
	Outcome       string `json:"outcome"`
	ApprovedOn    string `json:"approved_on"`
	EffectiveFrom string `json:"effective_from"`
	PanelName     string `json:"panel_name"`
	Funded        bool   `json:"funded"`
	LegalBasis    string `json:"legal_basis"`
	Notes         string `json:"notes"`
}

type MeritResultIssue struct {
	ID                string `json:"id"`
	GrantID           string `json:"grant_id"`
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

type CreateMeritResultIssueRequest struct {
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
