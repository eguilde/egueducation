package workflow

type Definition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	InitialStep string `json:"initial_step"`
	SLAHours    int    `json:"sla_hours"`
	Active      bool   `json:"active"`
}

type Task struct {
	ID                   string   `json:"id"`
	DefinitionCode       string   `json:"definition_code"`
	DefinitionName       string   `json:"definition_name"`
	Title                string   `json:"title"`
	DocumentNumber       string   `json:"document_number"`
	SourceModule         string   `json:"source_module"`
	SourceRecordID       *string  `json:"source_record_id"`
	Status               string   `json:"status"`
	Priority             string   `json:"priority"`
	AssignedTo           string   `json:"assigned_to"`
	CurrentStep          string   `json:"current_step"`
	DueAt                *string  `json:"due_at"`
	StartedAt            string   `json:"started_at"`
	UpdatedAt            string   `json:"updated_at"`
	InstitutionID        string   `json:"institution_id"`
	Summary              string   `json:"summary"`
	LinkedDocumentsCount int      `json:"linked_documents_count"`
	DossierReady         bool     `json:"dossier_ready"`
	MissingRelations     []string `json:"missing_relations"`
	AvailableActions     []string `json:"available_actions"`
}

type FiltersResponse struct {
	Statuses   []string `json:"statuses"`
	Priorities []string `json:"priorities"`
	Assignees  []string `json:"assignees"`
}

type DashboardResponse struct {
	Stats DashboardStats `json:"stats"`
}

type DashboardStats struct {
	ActiveTasks       int `json:"active_tasks"`
	OverdueTasks      int `json:"overdue_tasks"`
	WaitingApproval   int `json:"waiting_approval"`
	ActiveDefinitions int `json:"active_definitions"`
	ReadyDossiers     int `json:"ready_dossiers"`
	BlockedDossiers   int `json:"blocked_dossiers"`
}

type CreateTaskRequest struct {
	DefinitionCode string  `json:"definition_code"`
	Title          string  `json:"title"`
	DocumentNumber string  `json:"document_number"`
	SourceModule   string  `json:"source_module"`
	SourceRecordID string  `json:"source_record_id"`
	Priority       string  `json:"priority"`
	AssignedTo     string  `json:"assigned_to"`
	DueDate        *string `json:"due_date"`
	Summary        string  `json:"summary"`
}

type TransitionTaskRequest struct {
	Action string `json:"action"`
}
