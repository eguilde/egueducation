package education

type GovernanceMeeting struct {
	ID                string `json:"id"`
	SchoolYear        string `json:"school_year"`
	Organism          string `json:"organism"`
	Title             string `json:"title"`
	MeetingType       string `json:"meeting_type"`
	Status            string `json:"status"`
	QuorumRequired    int    `json:"quorum_required"`
	ParticipantsCount int    `json:"participants_count"`
	MeetingDate       string `json:"meeting_date"`
	Location          string `json:"location"`
	Chairperson       string `json:"chairperson"`
	SecretaryName     string `json:"secretary_name"`
	InstitutionID     string `json:"institution_id"`
	Summary           string `json:"summary"`
}

type TaxonomyItem struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	Code      string `json:"code"`
	LabelRO   string `json:"label_ro"`
	LabelEN   string `json:"label_en"`
	Active    bool   `json:"active"`
	SortOrder int    `json:"sort_order"`
}

type TaxonomyCatalogResponse struct {
	Items map[string][]TaxonomyItem `json:"items"`
}

type GovernanceFiltersResponse struct {
	SchoolYears  []string `json:"school_years"`
	Organisms    []string `json:"organisms"`
	MeetingTypes []string `json:"meeting_types"`
	Statuses     []string `json:"statuses"`
}

type GovernanceDashboardResponse struct {
	Stats GovernanceStats `json:"stats"`
}

type GovernanceStats struct {
	TotalMeetings     int `json:"total_meetings"`
	ScheduledMeetings int `json:"scheduled_meetings"`
	HeldMeetings      int `json:"held_meetings"`
	PublishedMeetings int `json:"published_meetings"`
}

type CreateGovernanceMeetingRequest struct {
	SchoolYear        string `json:"school_year"`
	Organism          string `json:"organism"`
	Title             string `json:"title"`
	MeetingType       string `json:"meeting_type"`
	Status            string `json:"status"`
	QuorumRequired    int    `json:"quorum_required"`
	ParticipantsCount int    `json:"participants_count"`
	MeetingDate       string `json:"meeting_date"`
	Location          string `json:"location"`
	Chairperson       string `json:"chairperson"`
	SecretaryName     string `json:"secretary_name"`
	Summary           string `json:"summary"`
}

type GovernanceDecision struct {
	ID                string `json:"id"`
	DecisionCode      string `json:"decision_code"`
	SchoolYear        string `json:"school_year"`
	Organism          string `json:"organism"`
	Title             string `json:"title"`
	Status            string `json:"status"`
	PublicationStatus string `json:"publication_status"`
	DecisionDate      string `json:"decision_date"`
	LegalBasis        string `json:"legal_basis"`
	SignedBy          string `json:"signed_by"`
	InstitutionID     string `json:"institution_id"`
	Summary           string `json:"summary"`
}

type GovernanceDecisionFiltersResponse struct {
	SchoolYears         []string `json:"school_years"`
	Organisms           []string `json:"organisms"`
	Statuses            []string `json:"statuses"`
	PublicationStatuses []string `json:"publication_statuses"`
}

type GovernanceDecisionDashboardResponse struct {
	Stats GovernanceDecisionStats `json:"stats"`
}

type GovernanceDecisionStats struct {
	TotalDecisions     int `json:"total_decisions"`
	ApprovedDecisions  int `json:"approved_decisions"`
	PublishedDecisions int `json:"published_decisions"`
	PendingPublication int `json:"pending_publication"`
}

type CreateGovernanceDecisionRequest struct {
	SchoolYear        string `json:"school_year"`
	Organism          string `json:"organism"`
	Title             string `json:"title"`
	Status            string `json:"status"`
	PublicationStatus string `json:"publication_status"`
	DecisionDate      string `json:"decision_date"`
	LegalBasis        string `json:"legal_basis"`
	SignedBy          string `json:"signed_by"`
	Summary           string `json:"summary"`
}

type ManagerialDossier struct {
	ID                  string `json:"id"`
	DossierCode         string `json:"dossier_code"`
	SchoolYear          string `json:"school_year"`
	DossierType         string `json:"dossier_type"`
	Title               string `json:"title"`
	Status              string `json:"status"`
	OwnerName           string `json:"owner_name"`
	DueOn               string `json:"due_on"`
	PublicationRequired bool   `json:"publication_required"`
	InstitutionID       string `json:"institution_id"`
	Summary             string `json:"summary"`
}

type ManagerialDossierFiltersResponse struct {
	SchoolYears  []string `json:"school_years"`
	DossierTypes []string `json:"dossier_types"`
	Statuses     []string `json:"statuses"`
}

type ManagerialDossierDashboardResponse struct {
	Stats ManagerialDossierStats `json:"stats"`
}

type ManagerialDossierStats struct {
	TotalDossiers     int `json:"total_dossiers"`
	ReviewDossiers    int `json:"review_dossiers"`
	PublishedDossiers int `json:"published_dossiers"`
	OverdueDossiers   int `json:"overdue_dossiers"`
}

type CreateManagerialDossierRequest struct {
	SchoolYear          string `json:"school_year"`
	DossierType         string `json:"dossier_type"`
	Title               string `json:"title"`
	Status              string `json:"status"`
	OwnerName           string `json:"owner_name"`
	DueOn               string `json:"due_on"`
	PublicationRequired bool   `json:"publication_required"`
	Summary             string `json:"summary"`
}

type RegulationRecord struct {
	ID             string `json:"id"`
	RegulationCode string `json:"regulation_code"`
	SchoolYear     string `json:"school_year"`
	RegulationType string `json:"regulation_type"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	ApprovalStatus string `json:"approval_status"`
	OwnerName      string `json:"owner_name"`
	ReviewDueOn    string `json:"review_due_on"`
	ApprovedOn     string `json:"approved_on"`
	InstitutionID  string `json:"institution_id"`
	Summary        string `json:"summary"`
}

type RegulationFiltersResponse struct {
	SchoolYears      []string `json:"school_years"`
	RegulationTypes  []string `json:"regulation_types"`
	Statuses         []string `json:"statuses"`
	ApprovalStatuses []string `json:"approval_statuses"`
}

type RegulationDashboardResponse struct {
	Stats RegulationStats `json:"stats"`
}

type RegulationStats struct {
	TotalRegulations     int `json:"total_regulations"`
	ConsultationItems    int `json:"consultation_items"`
	ApprovedRegulations  int `json:"approved_regulations"`
	PublishedRegulations int `json:"published_regulations"`
}

type CreateRegulationRecordRequest struct {
	SchoolYear     string `json:"school_year"`
	RegulationType string `json:"regulation_type"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	ApprovalStatus string `json:"approval_status"`
	OwnerName      string `json:"owner_name"`
	ReviewDueOn    string `json:"review_due_on"`
	ApprovedOn     string `json:"approved_on"`
	Summary        string `json:"summary"`
}
