package education

type PersonnelRecord struct {
	ID               string `json:"id"`
	EmployeeCode     string `json:"employee_code"`
	FullName         string `json:"full_name"`
	RoleTitle        string `json:"role_title"`
	EmploymentType   string `json:"employment_type"`
	Status           string `json:"status"`
	EvaluationStatus string `json:"evaluation_status"`
	MobilityStage    string `json:"mobility_stage"`
	SchoolYear       string `json:"school_year"`
	AssignedUnit     string `json:"assigned_unit"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	HasPortfolio     bool   `json:"has_portfolio"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type PersonnelFiltersResponse struct {
	SchoolYears      []string `json:"school_years"`
	EmploymentTypes  []string `json:"employment_types"`
	Statuses         []string `json:"statuses"`
	EvaluationStatus []string `json:"evaluation_statuses"`
	MobilityStages   []string `json:"mobility_stages"`
}

type PersonnelDashboardResponse struct {
	Stats PersonnelStats `json:"stats"`
}

type PersonnelStats struct {
	TotalRecords      int `json:"total_records"`
	ActiveRecords     int `json:"active_records"`
	PortfoliosEnabled int `json:"portfolios_enabled"`
	MobilityCases     int `json:"mobility_cases"`
}

type PersonnelPortfolioDossierSummary struct {
	Personnel PersonnelPortfolioDossierSubject    `json:"personnel"`
	Dossier   PersonnelPortfolioDossierStats      `json:"dossier"`
	Portfolio PersonnelPortfolioAggregate         `json:"portfolio"`
	Relation  PersonnelPortfolioRelationRules     `json:"relation"`
	Readiness PersonnelPortfolioRelationReadiness `json:"readiness"`
}

type PersonnelPortfolioDossierSubject struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	RoleTitle    string `json:"role_title"`
	SchoolYear   string `json:"school_year"`
	HasPortfolio bool   `json:"has_portfolio"`
}

type PersonnelPortfolioDossierStats struct {
	TotalDocuments               int `json:"total_documents"`
	PersonalFileDocuments        int `json:"personal_file_documents"`
	DirectorFileDocuments        int `json:"director_file_documents"`
	AdjunctDirectorFileDocuments int `json:"adjunct_director_file_documents"`
	SensitiveDocuments           int `json:"sensitive_documents"`
	DocumentsMarkedForPortfolio  int `json:"documents_marked_for_portfolio"`
	EvaluationDocuments          int `json:"evaluation_documents"`
	AdministrativeCareerDocs     int `json:"administrative_career_documents"`
}

type PersonnelPortfolioAggregate struct {
	MatchedRecords          int    `json:"matched_records"`
	ValidatedRecords        int    `json:"validated_records"`
	TotalDocuments          int    `json:"total_documents"`
	PortfolioScopeDocuments int    `json:"portfolio_scope_documents"`
	PersonnelScopeDocuments int    `json:"personnel_scope_documents"`
	VerifiedDocuments       int    `json:"verified_documents"`
	LastUpdatedOn           string `json:"last_updated_on"`
}

type PersonnelPortfolioRelationRules struct {
	MirroredFileReferences               int      `json:"mirrored_file_references"`
	EvaluationResultsEnterPersonnelFile  bool     `json:"evaluation_results_enter_personnel_file"`
	AdministrativeDocsEnterPersonnelFile bool     `json:"administrative_docs_enter_personnel_file"`
	InstitutionMayDuplicateOrSeparate    bool     `json:"institution_may_duplicate_or_separate"`
	DuplicationMode                      string   `json:"duplication_mode"`
	Rules                                []string `json:"rules"`
}

type PersonnelPortfolioRelationReadiness struct {
	ClearDelimitation bool     `json:"clear_delimitation"`
	Blockers          []string `json:"blockers"`
}

type CreatePersonnelRecordRequest struct {
	FullName         string `json:"full_name"`
	RoleTitle        string `json:"role_title"`
	EmploymentType   string `json:"employment_type"`
	Status           string `json:"status"`
	EvaluationStatus string `json:"evaluation_status"`
	MobilityStage    string `json:"mobility_stage"`
	SchoolYear       string `json:"school_year"`
	AssignedUnit     string `json:"assigned_unit"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	HasPortfolio     bool   `json:"has_portfolio"`
	Notes            string `json:"notes"`
}

type PersonnelPersonalFileDocument struct {
	ID                   string `json:"id"`
	PersonnelID          string `json:"personnel_id"`
	DocumentCode         string `json:"document_code"`
	DocumentCategory     string `json:"document_category"`
	DocumentTitle        string `json:"document_title"`
	FileScope            string `json:"file_scope"`
	ConfidentialityLevel string `json:"confidentiality_level"`
	IssuedOn             string `json:"issued_on"`
	ExpiresOn            string `json:"expires_on"`
	FileReference        string `json:"file_reference"`
	SensitiveData        bool   `json:"sensitive_data"`
	IncludedInPortfolio  bool   `json:"included_in_portfolio"`
	InstitutionID        string `json:"institution_id"`
	Notes                string `json:"notes"`
}

type CreatePersonnelPersonalFileDocumentRequest struct {
	DocumentCategory     string `json:"document_category"`
	DocumentTitle        string `json:"document_title"`
	FileScope            string `json:"file_scope"`
	ConfidentialityLevel string `json:"confidentiality_level"`
	IssuedOn             string `json:"issued_on"`
	ExpiresOn            string `json:"expires_on"`
	FileReference        string `json:"file_reference"`
	SensitiveData        bool   `json:"sensitive_data"`
	IncludedInPortfolio  bool   `json:"included_in_portfolio"`
	Notes                string `json:"notes"`
}

type PersonnelPersonalAccessEvent struct {
	ID             string `json:"id"`
	PersonnelID    string `json:"personnel_id"`
	EventType      string `json:"event_type"`
	ActorName      string `json:"actor_name"`
	ActorRole      string `json:"actor_role"`
	Purpose        string `json:"purpose"`
	AccessChannel  string `json:"access_channel"`
	AccessedOn     string `json:"accessed_on"`
	ClosedOn       string `json:"closed_on"`
	SensitiveScope bool   `json:"sensitive_scope"`
	InstitutionID  string `json:"institution_id"`
	Notes          string `json:"notes"`
}

type CreatePersonnelPersonalAccessEventRequest struct {
	EventType      string `json:"event_type"`
	ActorName      string `json:"actor_name"`
	ActorRole      string `json:"actor_role"`
	Purpose        string `json:"purpose"`
	AccessChannel  string `json:"access_channel"`
	AccessedOn     string `json:"accessed_on"`
	ClosedOn       string `json:"closed_on"`
	SensitiveScope bool   `json:"sensitive_scope"`
	Notes          string `json:"notes"`
}

type PersonnelAssignment struct {
	ID                string `json:"id"`
	PersonnelID       string `json:"personnel_id"`
	AssignmentCode    string `json:"assignment_code"`
	AssignmentType    string `json:"assignment_type"`
	AssignmentTitle   string `json:"assignment_title"`
	Status            string `json:"status"`
	AssignedOn        string `json:"assigned_on"`
	EndedOn           string `json:"ended_on"`
	WeeklyHours       int    `json:"weekly_hours"`
	DecisionReference string `json:"decision_reference"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type CreatePersonnelAssignmentRequest struct {
	AssignmentType    string `json:"assignment_type"`
	AssignmentTitle   string `json:"assignment_title"`
	Status            string `json:"status"`
	AssignedOn        string `json:"assigned_on"`
	EndedOn           string `json:"ended_on"`
	WeeklyHours       int    `json:"weekly_hours"`
	DecisionReference string `json:"decision_reference"`
	Notes             string `json:"notes"`
}

type PersonnelDisciplinaryCase struct {
	ID            string `json:"id"`
	PersonnelID   string `json:"personnel_id"`
	CaseCode      string `json:"case_code"`
	CaseType      string `json:"case_type"`
	Status        string `json:"status"`
	ReportedOn    string `json:"reported_on"`
	HearingOn     string `json:"hearing_on"`
	ResolvedOn    string `json:"resolved_on"`
	CommitteeName string `json:"committee_name"`
	Sanction      string `json:"sanction"`
	LegalBasis    string `json:"legal_basis"`
	InstitutionID string `json:"institution_id"`
	Notes         string `json:"notes"`
}

type CreatePersonnelDisciplinaryCaseRequest struct {
	CaseType      string `json:"case_type"`
	Status        string `json:"status"`
	ReportedOn    string `json:"reported_on"`
	HearingOn     string `json:"hearing_on"`
	ResolvedOn    string `json:"resolved_on"`
	CommitteeName string `json:"committee_name"`
	Sanction      string `json:"sanction"`
	LegalBasis    string `json:"legal_basis"`
	Notes         string `json:"notes"`
}

type PersonnelEvaluation struct {
	ID             string  `json:"id"`
	EvaluationCode string  `json:"evaluation_code"`
	EmployeeCode   string  `json:"employee_code"`
	FullName       string  `json:"full_name"`
	RoleTitle      string  `json:"role_title"`
	SchoolYear     string  `json:"school_year"`
	Status         string  `json:"status"`
	Score          float64 `json:"score"`
	Qualification  string  `json:"qualification"`
	EvaluatorName  string  `json:"evaluator_name"`
	FinalizedOn    string  `json:"finalized_on"`
	InstitutionID  string  `json:"institution_id"`
	Summary        string  `json:"summary"`
}

type PersonnelEvaluationFiltersResponse struct {
	SchoolYears []string `json:"school_years"`
	Statuses    []string `json:"statuses"`
}

type PersonnelEvaluationDashboardResponse struct {
	Stats PersonnelEvaluationStats `json:"stats"`
}

type PersonnelEvaluationStats struct {
	TotalEvaluations     int `json:"total_evaluations"`
	SubmittedEvaluations int `json:"submitted_evaluations"`
	ApprovedEvaluations  int `json:"approved_evaluations"`
	ContestedEvaluations int `json:"contested_evaluations"`
	CommunicatedResults  int `json:"communicated_results"`
}

type CreatePersonnelEvaluationRequest struct {
	EmployeeCode  string  `json:"employee_code"`
	FullName      string  `json:"full_name"`
	RoleTitle     string  `json:"role_title"`
	SchoolYear    string  `json:"school_year"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	EvaluatorName string  `json:"evaluator_name"`
	FinalizedOn   string  `json:"finalized_on"`
	Summary       string  `json:"summary"`
}

type PersonnelEvaluationSelfReview struct {
	ID               string  `json:"id"`
	EvaluationID     string  `json:"evaluation_id"`
	ReviewCode       string  `json:"review_code"`
	SectionTitle     string  `json:"section_title"`
	NarrativeType    string  `json:"narrative_type"`
	Status           string  `json:"status"`
	CompletedOn      string  `json:"completed_on"`
	EvidenceSummary  string  `json:"evidence_summary"`
	Strengths        string  `json:"strengths"`
	ImprovementNeeds string  `json:"improvement_needs"`
	AssumedScore     float64 `json:"assumed_score"`
	InstitutionID    string  `json:"institution_id"`
	Notes            string  `json:"notes"`
}

type CreatePersonnelEvaluationSelfReviewRequest struct {
	SectionTitle     string  `json:"section_title"`
	NarrativeType    string  `json:"narrative_type"`
	Status           string  `json:"status"`
	CompletedOn      string  `json:"completed_on"`
	EvidenceSummary  string  `json:"evidence_summary"`
	Strengths        string  `json:"strengths"`
	ImprovementNeeds string  `json:"improvement_needs"`
	AssumedScore     float64 `json:"assumed_score"`
	Notes            string  `json:"notes"`
}

type PersonnelEvaluationCriterion struct {
	ID                string  `json:"id"`
	EvaluationID      string  `json:"evaluation_id"`
	CriterionCode     string  `json:"criterion_code"`
	CriterionCategory string  `json:"criterion_category"`
	CriterionLabel    string  `json:"criterion_label"`
	MaxScore          float64 `json:"max_score"`
	SelfScore         float64 `json:"self_score"`
	ReviewerScore     float64 `json:"reviewer_score"`
	FinalScore        float64 `json:"final_score"`
	Status            string  `json:"status"`
	EvidenceSummary   string  `json:"evidence_summary"`
	InstitutionID     string  `json:"institution_id"`
	Notes             string  `json:"notes"`
}

type CreatePersonnelEvaluationCriterionRequest struct {
	CriterionCategory string  `json:"criterion_category"`
	CriterionLabel    string  `json:"criterion_label"`
	MaxScore          float64 `json:"max_score"`
	SelfScore         float64 `json:"self_score"`
	ReviewerScore     float64 `json:"reviewer_score"`
	FinalScore        float64 `json:"final_score"`
	Status            string  `json:"status"`
	EvidenceSummary   string  `json:"evidence_summary"`
	Notes             string  `json:"notes"`
}

type PersonnelEvaluationAppeal struct {
	ID                      string `json:"id"`
	EvaluationID            string `json:"evaluation_id"`
	AppealCode              string `json:"appeal_code"`
	SubmittedBy             string `json:"submitted_by"`
	SubmittedOn             string `json:"submitted_on"`
	Status                  string `json:"status"`
	Grounds                 string `json:"grounds"`
	HearingOn               string `json:"hearing_on"`
	ResolvedOn              string `json:"resolved_on"`
	DecisionSummary         string `json:"decision_summary"`
	CommitteeNote           string `json:"committee_note"`
	AttachedToPersonnelFile bool   `json:"attached_to_personnel_file"`
	InstitutionID           string `json:"institution_id"`
}

type CreatePersonnelEvaluationAppealRequest struct {
	SubmittedBy             string `json:"submitted_by"`
	SubmittedOn             string `json:"submitted_on"`
	Status                  string `json:"status"`
	Grounds                 string `json:"grounds"`
	HearingOn               string `json:"hearing_on"`
	ResolvedOn              string `json:"resolved_on"`
	DecisionSummary         string `json:"decision_summary"`
	CommitteeNote           string `json:"committee_note"`
	AttachedToPersonnelFile bool   `json:"attached_to_personnel_file"`
}

type PersonnelEvaluationResultIssue struct {
	ID                      string `json:"id"`
	EvaluationID            string `json:"evaluation_id"`
	IssueCode               string `json:"issue_code"`
	DocumentType            string `json:"document_type"`
	RecipientName           string `json:"recipient_name"`
	RecipientRole           string `json:"recipient_role"`
	DeliveryChannel         string `json:"delivery_channel"`
	DeliveryStatus          string `json:"delivery_status"`
	IssuedOn                string `json:"issued_on"`
	DeliveredOn             string `json:"delivered_on"`
	AcknowledgedOn          string `json:"acknowledged_on"`
	RegistryReference       string `json:"registry_reference"`
	AttachedToPersonnelFile bool   `json:"attached_to_personnel_file"`
	InstitutionID           string `json:"institution_id"`
	Notes                   string `json:"notes"`
}

type CreatePersonnelEvaluationResultIssueRequest struct {
	DocumentType            string `json:"document_type"`
	RecipientName           string `json:"recipient_name"`
	RecipientRole           string `json:"recipient_role"`
	DeliveryChannel         string `json:"delivery_channel"`
	DeliveryStatus          string `json:"delivery_status"`
	IssuedOn                string `json:"issued_on"`
	DeliveredOn             string `json:"delivered_on"`
	AcknowledgedOn          string `json:"acknowledged_on"`
	RegistryReference       string `json:"registry_reference"`
	AttachedToPersonnelFile bool   `json:"attached_to_personnel_file"`
	Notes                   string `json:"notes"`
}

type PersonnelDeclaration struct {
	ID              string `json:"id"`
	DeclarationCode string `json:"declaration_code"`
	EmployeeCode    string `json:"employee_code"`
	FullName        string `json:"full_name"`
	DeclarationType string `json:"declaration_type"`
	Status          string `json:"status"`
	SchoolYear      string `json:"school_year"`
	SubmittedOn     string `json:"submitted_on"`
	ValidUntil      string `json:"valid_until"`
	InstitutionID   string `json:"institution_id"`
	Summary         string `json:"summary"`
}

type PersonnelDeclarationFiltersResponse struct {
	SchoolYears      []string `json:"school_years"`
	DeclarationTypes []string `json:"declaration_types"`
	Statuses         []string `json:"statuses"`
}

type PersonnelDeclarationDashboardResponse struct {
	Stats PersonnelDeclarationStats `json:"stats"`
}

type PersonnelDeclarationStats struct {
	TotalDeclarations     int `json:"total_declarations"`
	SubmittedDeclarations int `json:"submitted_declarations"`
	ValidatedDeclarations int `json:"validated_declarations"`
	ExpiredDeclarations   int `json:"expired_declarations"`
}

type CreatePersonnelDeclarationRequest struct {
	EmployeeCode    string `json:"employee_code"`
	FullName        string `json:"full_name"`
	DeclarationType string `json:"declaration_type"`
	Status          string `json:"status"`
	SchoolYear      string `json:"school_year"`
	SubmittedOn     string `json:"submitted_on"`
	ValidUntil      string `json:"valid_until"`
	Summary         string `json:"summary"`
}
