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

type GovernanceMeetingParticipant struct {
	ID               string `json:"id"`
	MeetingID        string `json:"meeting_id"`
	FullName         string `json:"full_name"`
	RoleName         string `json:"role_name"`
	MemberType       string `json:"member_type"`
	AttendanceStatus string `json:"attendance_status"`
	VotingRight      bool   `json:"voting_right"`
	SignaturePresent bool   `json:"signature_present"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreateGovernanceMeetingParticipantRequest struct {
	FullName         string `json:"full_name"`
	RoleName         string `json:"role_name"`
	MemberType       string `json:"member_type"`
	AttendanceStatus string `json:"attendance_status"`
	VotingRight      bool   `json:"voting_right"`
	SignaturePresent bool   `json:"signature_present"`
	Notes            string `json:"notes"`
}

type GovernanceMeetingDocument struct {
	ID                string `json:"id"`
	MeetingID         string `json:"meeting_id"`
	DocumentType      string `json:"document_type"`
	Title             string `json:"title"`
	DocumentNumber    string `json:"document_number"`
	RegistryNumber    string `json:"registry_number"`
	PublicationStatus string `json:"publication_status"`
	CustodyOwner      string `json:"custody_owner"`
	SignedBy          string `json:"signed_by"`
	IssuedOn          string `json:"issued_on"`
	InstitutionID     string `json:"institution_id"`
	Summary           string `json:"summary"`
}

type CreateGovernanceMeetingDocumentRequest struct {
	DocumentType      string `json:"document_type"`
	Title             string `json:"title"`
	DocumentNumber    string `json:"document_number"`
	RegistryNumber    string `json:"registry_number"`
	PublicationStatus string `json:"publication_status"`
	CustodyOwner      string `json:"custody_owner"`
	SignedBy          string `json:"signed_by"`
	IssuedOn          string `json:"issued_on"`
	Summary           string `json:"summary"`
}

type GovernanceMeetingVote struct {
	ID               string `json:"id"`
	MeetingID        string `json:"meeting_id"`
	SubjectTitle     string `json:"subject_title"`
	AgendaOrder      int    `json:"agenda_order"`
	DecisionType     string `json:"decision_type"`
	VotesFor         int    `json:"votes_for"`
	VotesAgainst     int    `json:"votes_against"`
	Abstentions      int    `json:"abstentions"`
	Outcome          string `json:"outcome"`
	RequiresFollowUp bool   `json:"requires_follow_up"`
	LegalBasis       string `json:"legal_basis"`
	InstitutionID    string `json:"institution_id"`
	Notes            string `json:"notes"`
}

type CreateGovernanceMeetingVoteRequest struct {
	SubjectTitle     string `json:"subject_title"`
	AgendaOrder      int    `json:"agenda_order"`
	DecisionType     string `json:"decision_type"`
	VotesFor         int    `json:"votes_for"`
	VotesAgainst     int    `json:"votes_against"`
	Abstentions      int    `json:"abstentions"`
	Outcome          string `json:"outcome"`
	RequiresFollowUp bool   `json:"requires_follow_up"`
	LegalBasis       string `json:"legal_basis"`
	Notes            string `json:"notes"`
}

type GovernanceMembership struct {
	ID            string `json:"id"`
	SchoolYear    string `json:"school_year"`
	Organism      string `json:"organism"`
	FullName      string `json:"full_name"`
	RoleName      string `json:"role_name"`
	MandateFrom   string `json:"mandate_from"`
	MandateTo     string `json:"mandate_to"`
	VotingRight   bool   `json:"voting_right"`
	Status        string `json:"status"`
	InstitutionID string `json:"institution_id"`
	Notes         string `json:"notes"`
}

type CreateGovernanceMembershipRequest struct {
	SchoolYear  string `json:"school_year"`
	Organism    string `json:"organism"`
	FullName    string `json:"full_name"`
	RoleName    string `json:"role_name"`
	MandateFrom string `json:"mandate_from"`
	MandateTo   string `json:"mandate_to"`
	VotingRight bool   `json:"voting_right"`
	Status      string `json:"status"`
	Notes       string `json:"notes"`
}

type GovernanceResolution struct {
	ID                 string `json:"id"`
	MeetingID          string `json:"meeting_id"`
	VoteID             string `json:"vote_id"`
	ResolutionCode     string `json:"resolution_code"`
	Title              string `json:"title"`
	ResolutionType     string `json:"resolution_type"`
	PublicationStatus  string `json:"publication_status"`
	AnonymizationState string `json:"anonymization_state"`
	IssuedOn           string `json:"issued_on"`
	SignedBy           string `json:"signed_by"`
	InstitutionID      string `json:"institution_id"`
	Notes              string `json:"notes"`
}

type CreateGovernanceResolutionRequest struct {
	VoteID             string `json:"vote_id"`
	Title              string `json:"title"`
	ResolutionType     string `json:"resolution_type"`
	PublicationStatus  string `json:"publication_status"`
	AnonymizationState string `json:"anonymization_state"`
	IssuedOn           string `json:"issued_on"`
	SignedBy           string `json:"signed_by"`
	Notes              string `json:"notes"`
}

type GovernanceMinuteItem struct {
	ID                  string `json:"id"`
	MeetingID           string `json:"meeting_id"`
	AgendaOrder         int    `json:"agenda_order"`
	TopicTitle          string `json:"topic_title"`
	DiscussionSummary   string `json:"discussion_summary"`
	DecisionSummary     string `json:"decision_summary"`
	ResponsibleParty    string `json:"responsible_party"`
	DueOn               string `json:"due_on"`
	FollowUpStatus      string `json:"follow_up_status"`
	RequiresPublication bool   `json:"requires_publication"`
	InstitutionID       string `json:"institution_id"`
	Notes               string `json:"notes"`
}

type CreateGovernanceMinuteItemRequest struct {
	AgendaOrder         int    `json:"agenda_order"`
	TopicTitle          string `json:"topic_title"`
	DiscussionSummary   string `json:"discussion_summary"`
	DecisionSummary     string `json:"decision_summary"`
	ResponsibleParty    string `json:"responsible_party"`
	DueOn               string `json:"due_on"`
	FollowUpStatus      string `json:"follow_up_status"`
	RequiresPublication bool   `json:"requires_publication"`
	Notes               string `json:"notes"`
}

type PublicationRecord struct {
	ID                  string `json:"id"`
	PublicationCode     string `json:"publication_code"`
	Domain              string `json:"domain"`
	EntityType          string `json:"entity_type"`
	EntityLabel         string `json:"entity_label"`
	PublicationChannel  string `json:"publication_channel"`
	PublicationStatus   string `json:"publication_status"`
	AnonymizationStatus string `json:"anonymization_status"`
	Mandatory           bool   `json:"mandatory"`
	PublishedOn         string `json:"published_on"`
	ReviewedBy          string `json:"reviewed_by"`
	InstitutionID       string `json:"institution_id"`
	Notes               string `json:"notes"`
}

type CreatePublicationRecordRequest struct {
	Domain              string `json:"domain"`
	EntityType          string `json:"entity_type"`
	EntityLabel         string `json:"entity_label"`
	PublicationChannel  string `json:"publication_channel"`
	PublicationStatus   string `json:"publication_status"`
	AnonymizationStatus string `json:"anonymization_status"`
	Mandatory           bool   `json:"mandatory"`
	PublishedOn         string `json:"published_on"`
	ReviewedBy          string `json:"reviewed_by"`
	Notes               string `json:"notes"`
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

type DecisionIssuance struct {
	ID              string `json:"id"`
	DecisionID      string `json:"decision_id"`
	IssuanceCode    string `json:"issuance_code"`
	DocumentType    string `json:"document_type"`
	RecipientName   string `json:"recipient_name"`
	RecipientRole   string `json:"recipient_role"`
	DeliveryChannel string `json:"delivery_channel"`
	DeliveryStatus  string `json:"delivery_status"`
	SignedOn        string `json:"signed_on"`
	DeliveredOn     string `json:"delivered_on"`
	AcknowledgedOn  string `json:"acknowledged_on"`
	FileReference   string `json:"file_reference"`
	InstitutionID   string `json:"institution_id"`
	Notes           string `json:"notes"`
}

type CreateDecisionIssuanceRequest struct {
	DocumentType    string `json:"document_type"`
	RecipientName   string `json:"recipient_name"`
	RecipientRole   string `json:"recipient_role"`
	DeliveryChannel string `json:"delivery_channel"`
	DeliveryStatus  string `json:"delivery_status"`
	SignedOn        string `json:"signed_on"`
	DeliveredOn     string `json:"delivered_on"`
	AcknowledgedOn  string `json:"acknowledged_on"`
	FileReference   string `json:"file_reference"`
	Notes           string `json:"notes"`
}

type DecisionPublicationStep struct {
	ID                   string `json:"id"`
	DecisionID           string `json:"decision_id"`
	StepOrder            int    `json:"step_order"`
	StepType             string `json:"step_type"`
	Status               string `json:"status"`
	ResponsibleName      string `json:"responsible_name"`
	PublicationChannel   string `json:"publication_channel"`
	DueOn                string `json:"due_on"`
	CompletedOn          string `json:"completed_on"`
	PublicationReference string `json:"publication_reference"`
	InstitutionID        string `json:"institution_id"`
	Notes                string `json:"notes"`
}

type CreateDecisionPublicationStepRequest struct {
	StepOrder            int    `json:"step_order"`
	StepType             string `json:"step_type"`
	Status               string `json:"status"`
	ResponsibleName      string `json:"responsible_name"`
	PublicationChannel   string `json:"publication_channel"`
	DueOn                string `json:"due_on"`
	CompletedOn          string `json:"completed_on"`
	PublicationReference string `json:"publication_reference"`
	Notes                string `json:"notes"`
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

type ManagerialDocument struct {
	ID                  string `json:"id"`
	DossierID           string `json:"dossier_id"`
	DocumentCode        string `json:"document_code"`
	DocumentCategory    string `json:"document_category"`
	Title               string `json:"title"`
	DocumentStatus      string `json:"document_status"`
	VersionLabel        string `json:"version_label"`
	Mandatory           bool   `json:"mandatory"`
	PublicationRequired bool   `json:"publication_required"`
	RegisteredOn        string `json:"registered_on"`
	ApprovedOn          string `json:"approved_on"`
	OwnerName           string `json:"owner_name"`
	FileReference       string `json:"file_reference"`
	InstitutionID       string `json:"institution_id"`
	Notes               string `json:"notes"`
}

type CreateManagerialDocumentRequest struct {
	DocumentCategory    string `json:"document_category"`
	Title               string `json:"title"`
	DocumentStatus      string `json:"document_status"`
	VersionLabel        string `json:"version_label"`
	Mandatory           bool   `json:"mandatory"`
	PublicationRequired bool   `json:"publication_required"`
	RegisteredOn        string `json:"registered_on"`
	ApprovedOn          string `json:"approved_on"`
	OwnerName           string `json:"owner_name"`
	FileReference       string `json:"file_reference"`
	Notes               string `json:"notes"`
}

type ManagerialWorkflowStep struct {
	ID                string `json:"id"`
	DossierID         string `json:"dossier_id"`
	StageOrder        int    `json:"stage_order"`
	StageType         string `json:"stage_type"`
	Status            string `json:"status"`
	AssignedTo        string `json:"assigned_to"`
	DueOn             string `json:"due_on"`
	CompletedOn       string `json:"completed_on"`
	RequiresSignature bool   `json:"requires_signature"`
	DecisionReference string `json:"decision_reference"`
	InstitutionID     string `json:"institution_id"`
	OutcomeNote       string `json:"outcome_note"`
}

type CreateManagerialWorkflowStepRequest struct {
	StageOrder        int    `json:"stage_order"`
	StageType         string `json:"stage_type"`
	Status            string `json:"status"`
	AssignedTo        string `json:"assigned_to"`
	DueOn             string `json:"due_on"`
	CompletedOn       string `json:"completed_on"`
	RequiresSignature bool   `json:"requires_signature"`
	DecisionReference string `json:"decision_reference"`
	OutcomeNote       string `json:"outcome_note"`
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

type RegulationVersion struct {
	ID            string `json:"id"`
	RegulationID  string `json:"regulation_id"`
	VersionLabel  string `json:"version_label"`
	VersionStatus string `json:"version_status"`
	ChangeSummary string `json:"change_summary"`
	ApprovedOn    string `json:"approved_on"`
	EffectiveFrom string `json:"effective_from"`
	PublishedOn   string `json:"published_on"`
	PreparedBy    string `json:"prepared_by"`
	FileReference string `json:"file_reference"`
	InstitutionID string `json:"institution_id"`
	Notes         string `json:"notes"`
}

type CreateRegulationVersionRequest struct {
	VersionLabel  string `json:"version_label"`
	VersionStatus string `json:"version_status"`
	ChangeSummary string `json:"change_summary"`
	ApprovedOn    string `json:"approved_on"`
	EffectiveFrom string `json:"effective_from"`
	PublishedOn   string `json:"published_on"`
	PreparedBy    string `json:"prepared_by"`
	FileReference string `json:"file_reference"`
	Notes         string `json:"notes"`
}

type RegulationWorkflowStep struct {
	ID                string `json:"id"`
	RegulationID      string `json:"regulation_id"`
	PhaseOrder        int    `json:"phase_order"`
	PhaseType         string `json:"phase_type"`
	Status            string `json:"status"`
	Audience          string `json:"audience"`
	StartedOn         string `json:"started_on"`
	DueOn             string `json:"due_on"`
	CompletedOn       string `json:"completed_on"`
	FeedbackCount     int    `json:"feedback_count"`
	DecisionReference string `json:"decision_reference"`
	InstitutionID     string `json:"institution_id"`
	Notes             string `json:"notes"`
}

type CreateRegulationWorkflowStepRequest struct {
	PhaseOrder        int    `json:"phase_order"`
	PhaseType         string `json:"phase_type"`
	Status            string `json:"status"`
	Audience          string `json:"audience"`
	StartedOn         string `json:"started_on"`
	DueOn             string `json:"due_on"`
	CompletedOn       string `json:"completed_on"`
	FeedbackCount     int    `json:"feedback_count"`
	DecisionReference string `json:"decision_reference"`
	Notes             string `json:"notes"`
}
