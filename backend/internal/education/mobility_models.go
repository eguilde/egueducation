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
	TotalCases    int `json:"total_cases"`
	OpenCases     int `json:"open_cases"`
	ApprovedCases int `json:"approved_cases"`
	TransferCases int `json:"transfer_cases"`
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
