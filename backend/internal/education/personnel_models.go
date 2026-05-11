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

type PersonnelEvaluation struct {
	ID             string  `json:"id"`
	EvaluationCode string  `json:"evaluation_code"`
	EmployeeCode   string  `json:"employee_code"`
	FullName       string  `json:"full_name"`
	RoleTitle      string  `json:"role_title"`
	SchoolYear     string  `json:"school_year"`
	Status         string  `json:"status"`
	Score          float64 `json:"score"`
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
