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
	TotalRecords    int     `json:"total_records"`
	ApprovedRecords int     `json:"approved_records"`
	FundedRecords   int     `json:"funded_records"`
	AverageScore    float64 `json:"average_score"`
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
