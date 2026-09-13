package admission

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes exposes the complete admission vertical. The server still owns the
// authenticated host middleware; handlers independently recheck live RBAC.
func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/class-offering-contexts", s.ListClassOfferingContexts)
	r.Post("/class-offering-contexts", s.CreateClassOfferingContext)
	r.Get("/regulatory-sources", s.ListRegulatorySources)
	r.Get("/candidate-parties", s.ListCandidateParties)
	r.Get("/students", s.ListStudents)
	r.Get("/eligible-archive-versions", s.ListEligibleArchiveVersions)
	r.Post("/retention-rule-versions", s.ProposeAdmissionRetentionRule)
	r.Post("/retention-rule-versions/approve", s.ApproveAdmissionRetentionRule)
	r.Post("/dss-retention-policies", s.ConfigureDSSRetentionPolicy)
	r.Get("/dss-retention-policies/current", s.CurrentDSSRetentionPolicy)
	r.Post("/signer-authorizations", s.ProposeAdmissionSignerAuthorization)
	r.Post("/signer-authorizations/approve", s.ApproveAdmissionSignerAuthorization)
	r.Get("/signer-authorizations", s.ListAdmissionSignerAuthorizations)
	r.Post("/signer-authorizations/{authorizationID}/revoke", s.RevokeAdmissionSignerAuthorization)
	r.Get("/campaigns", s.ListCampaigns)
	r.Post("/campaigns", s.CreateCampaign)
	r.Post("/campaigns/{campaignID}/transitions", s.TransitionCampaign)
	r.Get("/campaigns/{campaignID}/criteria", s.ListCriteria)
	r.Post("/campaigns/{campaignID}/criteria", s.AddCriterion)
	r.Get("/campaigns/{campaignID}/document-requirements", s.ListDocumentRequirements)
	r.Post("/campaigns/{campaignID}/document-requirements", s.AddDocumentRequirement)
	r.Get("/applications", s.ListApplications)
	r.Post("/applications", s.CreateApplication)
	r.Get("/applications/{applicationID}", s.GetApplication)
	r.Post("/applications/{applicationID}/transitions", s.TransitionApplication)
	r.Post("/applications/{applicationID}/assessments", s.AssessCriterion)
	r.Post("/applications/{applicationID}/documents/{documentID}", s.ReviewApplicationDocument)
	// Legacy direct legal issue endpoints intentionally fail closed. A legal
	// document has to be prepared before it is signed and finalized.
	r.Post("/applications/{applicationID}/decisions", s.IssueDecision)
	r.Post("/applications/{applicationID}/decision-preparations", s.PrepareDecision)
	r.Post("/applications/{applicationID}/appeals", s.CreateAppeal)
	r.Post("/applications/{applicationID}/enrolment", s.EnrolApplication)
	r.Get("/decisions", s.ListDecisions)
	r.Get("/appeals", s.ListAppeals)
	r.Post("/appeals/{appealID}/resolution", s.ResolveAppeal)
	r.Post("/appeals/{appealID}/resolution-preparations", s.PrepareAppealResolution)
	r.Post("/legal-preparations/finalize", s.FinalizeAdmissionLegalPreparation)
	r.Post("/legal-preparations/{preparationID}/cancel", s.CancelAdmissionLegalPreparation)
	return r
}
