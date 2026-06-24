package education

type DirectorCockpitResponse struct {
	SchoolYear       string                      `json:"school_year"`
	InstitutionID    string                      `json:"institution_id"`
	Governance       DirectorCockpitGovernance   `json:"governance"`
	Portfolios       DirectorCockpitPortfolios   `json:"portfolios"`
	Evaluations      DirectorCockpitEvaluations  `json:"evaluations"`
	Managerial       DirectorCockpitManagerial   `json:"managerial"`
	Personnel        DirectorCockpitPersonnel    `json:"personnel"`
	Compliance       DirectorCockpitCompliance   `json:"compliance"`
	Alerts           []DirectorCockpitAlert      `json:"alerts"`
	RecommendedLinks []DirectorCockpitQuickLink  `json:"recommended_links"`
}

type DirectorCockpitGovernance struct {
	TotalMeetings         int `json:"total_meetings"`
	ScheduledMeetings     int `json:"scheduled_meetings"`
	MeetingsWithoutMinute int `json:"meetings_without_minute"`
	MeetingsWithoutVote   int `json:"meetings_without_vote"`
	PublishedResolutions  int `json:"published_resolutions"`
}

type DirectorCockpitPortfolios struct {
	TotalRecords       int `json:"total_records"`
	DraftRecords       int `json:"draft_records"`
	ReviewRecords      int `json:"review_records"`
	ReturnedRecords    int `json:"returned_records"`
	ValidatedRecords   int `json:"validated_records"`
	TransferInProgress int `json:"transfer_in_progress"`
}

type DirectorCockpitEvaluations struct {
	TotalRecords          int `json:"total_records"`
	SubmittedRecords      int `json:"submitted_records"`
	ReviewedRecords       int `json:"reviewed_records"`
	ContestedRecords      int `json:"contested_records"`
	ApprovedRecords       int `json:"approved_records"`
	CommunicatedDocuments int `json:"communicated_documents"`
}

type DirectorCockpitManagerial struct {
	TotalDossiers       int `json:"total_dossiers"`
	DraftDossiers       int `json:"draft_dossiers"`
	ReviewDossiers      int `json:"review_dossiers"`
	ApprovedDossiers    int `json:"approved_dossiers"`
	PublishedDocuments  int `json:"published_documents"`
	WorkflowOpenSteps   int `json:"workflow_open_steps"`
}

type DirectorCockpitPersonnel struct {
	TotalRecords      int `json:"total_records"`
	ActiveRecords     int `json:"active_records"`
	PortfolioEnabled  int `json:"portfolio_enabled"`
	EvaluationPending int `json:"evaluation_pending"`
	MobilityCases     int `json:"mobility_cases"`
}

type DirectorCockpitCompliance struct {
	TotalRequirements      int `json:"total_requirements"`
	ImplementedRequirements int `json:"implemented_requirements"`
	PartialRequirements    int `json:"partial_requirements"`
	PendingPublications    int `json:"pending_publications"`
	AnonymizationPending   int `json:"anonymization_pending"`
}

type DirectorCockpitAlert struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Status   string `json:"status"`
	Route    string `json:"route"`
	Priority int    `json:"priority"`
}

type DirectorCockpitQuickLink struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Route string `json:"route"`
}
