package earchiva

type Record struct {
	ID                string `json:"id"`
	RecordNumber      string `json:"record_number"`
	Title             string `json:"title"`
	Fond              string `json:"fond"`
	Series            string `json:"series"`
	SourceModule      string `json:"source_module"`
	SourceReference   string `json:"source_reference"`
	Status            string `json:"status"`
	RetentionYears    int    `json:"retention_years"`
	AssignedArchivist string `json:"assigned_archivist"`
	BoxNumber         string `json:"box_number"`
	LocationCode      string `json:"location_code"`
	ArchivedAt        string `json:"archived_at"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type FiltersResponse struct {
	Fonds         []string `json:"fonds"`
	Series        []string `json:"series"`
	Statuses      []string `json:"statuses"`
	SourceModules []string `json:"source_modules"`
	Archivists    []string `json:"archivists"`
}

type DashboardResponse struct {
	Stats DashboardStats `json:"stats"`
}

type DashboardStats struct {
	TotalRecords     int `json:"total_records"`
	ValidatedRecords int `json:"validated_records"`
	DraftRecords     int `json:"draft_records"`
	UniqueFonds      int `json:"unique_fonds"`
}

type CreateRecordRequest struct {
	Title             string `json:"title"`
	Fond              string `json:"fond"`
	Series            string `json:"series"`
	SourceModule      string `json:"source_module"`
	SourceReference   string `json:"source_reference"`
	Status            string `json:"status"`
	RetentionYears    int    `json:"retention_years"`
	AssignedArchivist string `json:"assigned_archivist"`
	BoxNumber         string `json:"box_number"`
	LocationCode      string `json:"location_code"`
	ArchivedAt        string `json:"archived_at"`
	Notes             string `json:"notes"`
}
