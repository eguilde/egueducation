package education

type SecretariatCockpitResponse struct {
	Classes            int    `json:"classes"`
	Students           int    `json:"students"`
	ActiveEnrolments   int    `json:"active_enrolments"`
	PortfoliosInReview int    `json:"portfolios_in_review"`
	InstitutionID      string `json:"institution_id"`
}
type HRCockpitResponse struct {
	Personnel          int    `json:"personnel"`
	ExpiringDocuments  int    `json:"expiring_documents"`
	ExpiredDocuments   int    `json:"expired_documents"`
	PendingEvaluations int    `json:"pending_evaluations"`
	InstitutionID      string `json:"institution_id"`
}
type CommitteeCockpitResponse struct {
	Committees        int    `json:"committees"`
	ActiveMembers     int    `json:"active_members"`
	Meetings          int    `json:"meetings"`
	EvidenceDocuments int    `json:"evidence_documents"`
	InstitutionID     string `json:"institution_id"`
}
type InspectorCockpitResponse struct {
	ReadinessOpen               int    `json:"readiness_open"`
	PendingPublications         int    `json:"pending_publications"`
	MandatoryPublicationPending int    `json:"mandatory_publication_pending"`
	RequirementsPending         int    `json:"requirements_pending"`
	EvaluationsInReview         int    `json:"evaluations_in_review"`
	InstitutionID               string `json:"institution_id"`
}
