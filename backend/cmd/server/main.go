package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/admin"
	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/earchiva"
	"github.com/eguilde/egueducation/internal/education"
	"github.com/eguilde/egueducation/internal/gdpr"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/notification"
	"github.com/eguilde/egueducation/internal/registratura"
	"github.com/eguilde/egueducation/internal/workflow"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		logger.Fatal("database migration failed", zap.Error(err))
	}

	smsService := notification.NewSMSService(pool, cfg.SMSAPIToken, cfg.SMSSenderName)
	authService, err := auth.NewService(cfg, smsService, pool)
	if err != nil {
		logger.Fatal("auth service initialization failed", zap.Error(err))
	}
	adminService := admin.NewService(cfg, pool)
	educationService := education.NewService(pool)
	earchivaService := earchiva.NewService(pool)
	gdprService := gdpr.NewService(pool)
	registraturaService := registratura.NewService(pool)
	workflowService := workflow.NewService(pool)

	router := chi.NewRouter()
	router.Use(chimw.RequestID)
	router.Use(chimw.RealIP)
	router.Use(chimw.Recoverer)
	router.Use(chimw.Compress(5))
	router.Use(httprate.LimitByIP(120, time.Minute))
	router.Use(cors(cfg.FrontendOrigin))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := pool.Ping(r.Context()); err != nil {
			dbStatus = "error"
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"service":  "egueducation-api",
			"database": dbStatus,
			"time":     time.Now().UTC(),
		})
	})

	router.HandleFunc("/logout", authService.HandleLogoutAlias)

	router.Route("/api", func(r chi.Router) {
		r.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			httpx.JSON(w, http.StatusOK, buildBootstrapConfig(cfg, r))
		})

		r.Get("/meta/app", func(w http.ResponseWriter, r *http.Request) {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"name":              "EguEducation",
				"default_locale":    "ro",
				"available_locales": []string{"ro", "en"},
				"theme": map[string]string{
					"family": "material3-expressive",
					"brand":  "red-rose",
				},
			})
		})

		r.Get("/auth/methods", authService.ListMethods)
		r.Get("/auth/ui-config", authService.UIConfig)
		r.Get("/auth/role-catalog", authService.RoleCatalog)
		r.Get("/auth/role-positions", authService.RolePositions)
		r.Post("/auth/session/exchange", authService.ExchangeSession)
		r.Post("/auth/logout", authService.Logout)
		r.Mount("/oidc", authService.OIDCHandler())
		r.Post("/passkeys/login-options", authService.BeginPasskeyAuthentication)
		r.Post("/passkeys/login-finish", authService.FinishPasskeyAuthentication)

		r.Group(func(r chi.Router) {
			r.Use(authService.RequireAuthenticated)

			r.Get("/me", authService.SessionContext)
			r.Put("/profile", authService.UpdateProfile)
			r.Get("/passkeys", authService.ListPasskeys)
			r.Post("/passkeys/register-options", authService.BeginPasskeyRegistration)
			r.Post("/passkeys/register-finish", authService.FinishPasskeyRegistration)
			r.Post("/eudi-wallet/activate", authService.ActivateEUDIWallet)

			r.With(authService.RequirePermissions("admin.read")).Get("/admin/dashboard", adminService.Dashboard)
			r.With(authService.RequirePermissions("admin.users.read")).Get("/admin/users", adminService.ListUsers)
			r.With(authService.RequirePermissions("admin.users.read")).Get("/admin/users/filters", adminService.UserFilters)
			r.With(authService.RequirePermissions("admin.users.manage")).Post("/admin/users", adminService.UpsertUser)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/roles", adminService.ListRoles)
			r.With(authService.RequirePermissions("admin.roles.manage")).Post("/admin/roles", adminService.UpsertRole)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/role-assignments", adminService.ListUserRoleAssignments)
			r.With(authService.RequirePermissions("admin.roles.manage")).Post("/admin/role-assignments", adminService.UpsertUserRoleAssignment)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/role-permissions", adminService.ListRolePermissionAssignments)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/role-permissions/filters", adminService.RolePermissionAssignmentFilters)
			r.With(authService.RequirePermissions("admin.roles.manage")).Post("/admin/role-permissions", adminService.UpsertRolePermissionAssignment)
			r.With(authService.RequirePermissions("admin.positions.read")).Get("/admin/position-roles", adminService.ListPositionRoleAssignments)
			r.With(authService.RequirePermissions("admin.positions.read")).Get("/admin/position-roles/filters", adminService.PositionRoleAssignmentFilters)
			r.With(authService.RequirePermissions("admin.positions.manage")).Post("/admin/position-roles", adminService.UpsertPositionRoleAssignment)
			r.With(authService.RequirePermissions("admin.org_units.read")).Get("/admin/org-units", adminService.ListOrgUnits)
			r.With(authService.RequirePermissions("admin.org_units.manage")).Post("/admin/org-units", adminService.UpsertOrgUnit)
			r.With(authService.RequirePermissions("admin.memberships.read")).Get("/admin/memberships", adminService.ListMemberships)
			r.With(authService.RequirePermissions("admin.memberships.manage")).Post("/admin/memberships", adminService.UpsertMembership)
			r.With(authService.RequirePermissions("admin.positions.read")).Get("/admin/positions", adminService.ListPositions)
			r.With(authService.RequirePermissions("admin.positions.manage")).Post("/admin/positions", adminService.UpsertPosition)
			r.With(authService.RequirePermissions("admin.permissions.read")).Get("/admin/permissions", adminService.ListPermissions)
			r.With(authService.RequirePermissions("admin.permissions.read")).Get("/admin/permissions/assignments", adminService.ListPermissionAssignments)
			r.With(authService.RequirePermissions("admin.permissions.read")).Get("/admin/permissions/assignments/filters", adminService.PermissionAssignmentFilters)
			r.With(authService.RequirePermissions("admin.permissions.manage")).Post("/admin/permissions/assignments", adminService.UpsertPermissionAssignment)
			r.With(authService.RequirePermissions("admin.auth_methods.read")).Get("/admin/auth-methods", adminService.ListAuthMethods)
			r.With(authService.RequirePermissions("admin.auth_methods.manage")).Post("/admin/auth-methods", adminService.UpsertAuthMethod)
			r.With(authService.RequirePermissions("admin.modules.read")).Get("/admin/modules", adminService.ListModules)
			r.With(authService.RequirePermissions("admin.modules.manage")).Post("/admin/modules", adminService.UpsertModule)
			r.With(authService.RequirePermissions("admin.identity.read")).Get("/admin/oidc/clients", adminService.ListOIDCClients)
			r.With(authService.RequirePermissions("admin.identity.manage")).Post("/admin/oidc/clients", adminService.UpsertOIDCClient)
			r.With(authService.RequirePermissions("admin.audit.read")).Get("/admin/audit", adminService.ListAuditEvents)
			r.With(authService.RequirePermissions("admin.audit.read")).Get("/admin/audit/filters", adminService.AuditFilters)
			r.With(authService.RequirePermissions("admin.gdpr_settings.read")).Get("/admin/gdpr-settings", adminService.ListGdprSettings)
			r.With(authService.RequirePermissions("admin.gdpr_settings.manage")).Post("/admin/gdpr-settings", adminService.UpsertGdprSetting)
			r.With(authService.RequirePermissions("admin.dossier_requirements.read")).Get("/admin/dossier-requirements", adminService.ListDossierRequirements)
			r.With(authService.RequirePermissions("admin.dossier_requirements.read")).Get("/admin/dossier-requirements/filters", adminService.DossierRequirementFilters)
			r.With(authService.RequirePermissions("admin.dossier_requirements.manage")).Post("/admin/dossier-requirements", adminService.UpsertDossierRequirement)
			r.With(authService.RequirePermissions("admin.workflow_definitions.read")).Get("/admin/workflow-definitions", adminService.ListWorkflowDefinitions)
			r.With(authService.RequirePermissions("admin.workflow_definitions.read")).Get("/admin/workflow-definitions/filters", adminService.WorkflowDefinitionFilters)
			r.With(authService.RequirePermissions("admin.workflow_definitions.manage")).Post("/admin/workflow-definitions", adminService.UpsertWorkflowDefinition)
			r.With(authService.RequirePermissions("admin.nomenclatures.read")).Get("/admin/nomenclatures", adminService.ListNomenclatures)
			r.With(authService.RequirePermissions("admin.nomenclatures.read")).Get("/admin/nomenclatures/filters", adminService.NomenclatureFilters)
			r.With(authService.RequirePermissions("admin.nomenclatures.manage")).Post("/admin/nomenclatures", adminService.UpsertNomenclature)
			r.With(authService.RequirePermissions("admin.education_taxonomies.read")).Get("/admin/education-taxonomies", adminService.ListEducationTaxonomies)
			r.With(authService.RequirePermissions("admin.education_taxonomies.read")).Get("/admin/education-taxonomies/filters", adminService.EducationTaxonomyFilters)
			r.With(authService.RequirePermissions("admin.education_taxonomies.manage")).Post("/admin/education-taxonomies", adminService.UpsertEducationTaxonomy)

			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents", registraturaService.ListDocuments)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/filters", registraturaService.DocumentFilters)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/nomenclatures", registraturaService.Nomenclatures)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents", registraturaService.CreateDocument)
			r.With(authService.RequirePermissions("registratura.manage")).Patch("/registratura/documents/{documentID}", registraturaService.UpdateDocument)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/{documentID}/cancel", registraturaService.CancelDocument)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/batch", registraturaService.BatchCreateDocuments)
			r.With(authService.RequirePermissions("registratura.read")).Post("/registratura/documents/export-pdf", registraturaService.ExportPDF)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/lookup", registraturaService.LookupDocuments)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}", registraturaService.GetDocument)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}/versions", registraturaService.ListDocumentVersions)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/{documentID}/versions", registraturaService.CreateDocumentVersion)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}/attachments", registraturaService.ListDocumentAttachments)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/{documentID}/attachments", registraturaService.CreateDocumentAttachment)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/registre", registraturaService.ListRegistries)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/registre/default", registraturaService.GetDefaultRegistry)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/registre/{id}", registraturaService.GetRegistry)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/registre", registraturaService.CreateRegistru)
			r.With(authService.RequirePermissions("registratura.manage")).Patch("/registratura/registre/{id}", registraturaService.UpdateRegistru)
			r.With(authService.RequirePermissions("registratura.manage")).Delete("/registratura/registre/{id}", registraturaService.DeleteRegistru)
			r.With(authService.RequirePermissions("registratura.manage")).Patch("/registratura/registre/{id}/set-default", registraturaService.SetDefaultRegistru)
			r.With(authService.RequirePermissions("registratura.links.read")).Get("/registratura/document-links", registraturaService.ListDocumentLinks)
			r.With(authService.RequirePermissions("registratura.links.manage")).Post("/registratura/document-links", registraturaService.CreateDocumentLink)
			r.With(authService.RequirePermissions("registratura.links.manage")).Delete("/registratura/document-links/{linkID}", registraturaService.DeleteDocumentLink)

			r.With(authService.RequirePermissions("workflow.read")).Get("/workflow/dashboard", workflowService.Dashboard)
			r.With(authService.RequirePermissions("workflow.read")).Get("/workflow/definitions", workflowService.ListDefinitions)
			r.With(authService.RequirePermissions("workflow.read")).Get("/workflow/tasks", workflowService.ListTasks)
			r.With(authService.RequirePermissions("workflow.read")).Get("/workflow/tasks/filters", workflowService.TaskFilters)
			r.With(authService.RequirePermissions("workflow.manage")).Post("/workflow/tasks", workflowService.CreateTask)
			r.With(authService.RequirePermissions("workflow.transition")).Post("/workflow/tasks/{taskID}/transition", workflowService.TransitionTask)

			r.With(authService.RequirePermissions("earchiva.read")).Get("/earchiva/dashboard", earchivaService.Dashboard)
			r.With(authService.RequirePermissions("earchiva.read")).Get("/earchiva/records", earchivaService.ListRecords)
			r.With(authService.RequirePermissions("earchiva.read")).Get("/earchiva/records/filters", earchivaService.Filters)
			r.With(authService.RequirePermissions("earchiva.read")).Get("/earchiva/nomenclatures", earchivaService.Nomenclatures)
			r.With(authService.RequirePermissions("earchiva.manage")).Post("/earchiva/records", earchivaService.CreateRecord)

			r.Group(func(r chi.Router) {
				r.Use(educationService.RequireInstitutionContext)

				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/dashboard", educationService.GovernanceDashboard)
				r.With(authService.RequirePermissions("education.read")).Get("/education/taxonomies", educationService.ListTaxonomies)
				r.With(authService.RequirePermissions("education.read")).Get("/education/requirements", educationService.RequirementCatalog)
				r.With(authService.RequirePermissions("education.compliance.read")).Get("/education/compliance/publications", educationService.PublicationRecords)
				r.With(authService.RequirePermissions("education.compliance.read")).Get("/education/compliance/publications/{recordID}", educationService.PublicationRecordDetail)
				r.With(authService.RequirePermissions("education.compliance.manage")).Post("/education/compliance/publications", educationService.CreatePublicationRecord)
				r.With(authService.RequirePermissions("education.compliance.manage")).Patch("/education/compliance/publications/{recordID}", educationService.UpdatePublicationRecord)
				r.With(authService.RequirePermissions("education.compliance.manage")).Delete("/education/compliance/publications/{recordID}", educationService.DeletePublicationRecord)
				r.With(authService.RequireAnyPermissions(
					"education.read",
					"education.compliance.read",
					"education.governance.read",
					"education.decisions.read",
					"education.decisions.issuance.read",
					"education.managerial.read",
					"education.regulations.read",
					"education.personnel.read",
					"education.personnel.files.read",
					"education.personnel.access.read",
					"education.evaluations.read",
					"education.declarations.read",
					"education.mobility.read",
					"education.gradatii.read",
					"education.portfolios.read",
				)).Post("/education/exports/pdf", educationService.ExportPDF)
				r.With(authService.RequireAnyPermissions(
					"education.read",
					"education.compliance.read",
					"education.governance.read",
					"education.decisions.read",
					"education.decisions.issuance.read",
					"education.managerial.read",
					"education.regulations.read",
					"education.personnel.read",
					"education.personnel.files.read",
					"education.personnel.access.read",
					"education.evaluations.read",
					"education.declarations.read",
					"education.mobility.read",
					"education.gradatii.read",
					"education.portfolios.read",
				)).Post("/education/exports/csv", educationService.ExportCSV)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings", educationService.GovernanceMeetings)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}", educationService.GovernanceMeetingDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/filters", educationService.GovernanceFilters)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/memberships", educationService.GovernanceMemberships)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/memberships/{recordID}", educationService.GovernanceMembershipDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/participants", educationService.GovernanceMeetingParticipants)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/participants/{participantID}", educationService.GovernanceMeetingParticipantDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/documents", educationService.GovernanceMeetingDocuments)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/documents/{documentID}", educationService.GovernanceMeetingDocumentDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/votes", educationService.GovernanceMeetingVotes)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/votes/{voteID}", educationService.GovernanceMeetingVoteDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/minutes", educationService.GovernanceMinuteItems)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/minutes/{recordID}", educationService.GovernanceMinuteItemDetail)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/resolutions", educationService.GovernanceResolutions)
				r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/{meetingID}/resolutions/{recordID}", educationService.GovernanceResolutionDetail)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings", educationService.CreateGovernanceMeeting)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}", educationService.UpdateGovernanceMeeting)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}", educationService.DeleteGovernanceMeeting)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/memberships", educationService.CreateGovernanceMembership)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/memberships/{recordID}", educationService.UpdateGovernanceMembership)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/memberships/{recordID}", educationService.DeleteGovernanceMembership)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings/{meetingID}/participants", educationService.CreateGovernanceMeetingParticipant)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}/participants/{participantID}", educationService.UpdateGovernanceMeetingParticipant)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}/participants/{participantID}", educationService.DeleteGovernanceMeetingParticipant)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings/{meetingID}/documents", educationService.CreateGovernanceMeetingDocument)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}/documents/{documentID}", educationService.UpdateGovernanceMeetingDocument)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}/documents/{documentID}", educationService.DeleteGovernanceMeetingDocument)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings/{meetingID}/votes", educationService.CreateGovernanceMeetingVote)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}/votes/{voteID}", educationService.UpdateGovernanceMeetingVote)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}/votes/{voteID}", educationService.DeleteGovernanceMeetingVote)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings/{meetingID}/minutes", educationService.CreateGovernanceMinuteItem)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}/minutes/{recordID}", educationService.UpdateGovernanceMinuteItem)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}/minutes/{recordID}", educationService.DeleteGovernanceMinuteItem)
				r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings/{meetingID}/resolutions", educationService.CreateGovernanceResolution)
				r.With(authService.RequirePermissions("education.governance.manage")).Patch("/education/governance/meetings/{meetingID}/resolutions/{recordID}", educationService.UpdateGovernanceResolution)
				r.With(authService.RequirePermissions("education.governance.manage")).Delete("/education/governance/meetings/{meetingID}/resolutions/{recordID}", educationService.DeleteGovernanceResolution)
				r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/dashboard", educationService.DecisionDashboard)
				r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/records", educationService.GovernanceDecisions)
				r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/records/{decisionID}", educationService.GovernanceDecisionDetail)
				r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/records/filters", educationService.DecisionFilters)
				r.With(authService.RequirePermissions("education.decisions.issuance.read")).Get("/education/decisions/records/{decisionID}/issuances", educationService.DecisionIssuances)
				r.With(authService.RequirePermissions("education.decisions.issuance.read")).Get("/education/decisions/records/{decisionID}/issuances/{itemID}", educationService.DecisionIssuanceDetail)
				r.With(authService.RequirePermissions("education.compliance.read")).Get("/education/decisions/records/{decisionID}/publication-steps", educationService.DecisionPublicationSteps)
				r.With(authService.RequirePermissions("education.compliance.read")).Get("/education/decisions/records/{decisionID}/publication-steps/{itemID}", educationService.DecisionPublicationStepDetail)
				r.With(authService.RequirePermissions("education.decisions.manage")).Post("/education/decisions/records", educationService.CreateGovernanceDecision)
				r.With(authService.RequirePermissions("education.decisions.manage")).Patch("/education/decisions/records/{decisionID}", educationService.UpdateGovernanceDecision)
				r.With(authService.RequirePermissions("education.decisions.manage")).Delete("/education/decisions/records/{decisionID}", educationService.DeleteGovernanceDecision)
				r.With(authService.RequirePermissions("education.decisions.issuance.manage")).Post("/education/decisions/records/{decisionID}/issuances", educationService.CreateDecisionIssuance)
				r.With(authService.RequirePermissions("education.decisions.issuance.manage")).Patch("/education/decisions/records/{decisionID}/issuances/{itemID}", educationService.UpdateDecisionIssuance)
				r.With(authService.RequirePermissions("education.decisions.issuance.manage")).Delete("/education/decisions/records/{decisionID}/issuances/{itemID}", educationService.DeleteDecisionIssuance)
				r.With(authService.RequirePermissions("education.compliance.manage")).Post("/education/decisions/records/{decisionID}/publication-steps", educationService.CreateDecisionPublicationStep)
				r.With(authService.RequirePermissions("education.compliance.manage")).Patch("/education/decisions/records/{decisionID}/publication-steps/{itemID}", educationService.UpdateDecisionPublicationStep)
				r.With(authService.RequirePermissions("education.compliance.manage")).Delete("/education/decisions/records/{decisionID}/publication-steps/{itemID}", educationService.DeleteDecisionPublicationStep)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/dashboard", educationService.ManagerialDashboard)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records", educationService.ManagerialDossiers)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/{recordID}", educationService.ManagerialDossierDetail)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/filters", educationService.ManagerialFilters)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/{recordID}/documents", educationService.ManagerialDocuments)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/{recordID}/documents/{documentID}", educationService.ManagerialDocumentDetail)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/{recordID}/workflow", educationService.ManagerialWorkflowSteps)
				r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/{recordID}/workflow/{stepID}", educationService.ManagerialWorkflowStepDetail)
				r.With(authService.RequirePermissions("education.managerial.manage")).Post("/education/managerial/records", educationService.CreateManagerialDossier)
				r.With(authService.RequirePermissions("education.managerial.manage")).Patch("/education/managerial/records/{recordID}", educationService.UpdateManagerialDossier)
				r.With(authService.RequirePermissions("education.managerial.manage")).Delete("/education/managerial/records/{recordID}", educationService.DeleteManagerialDossier)
				r.With(authService.RequirePermissions("education.managerial.manage")).Post("/education/managerial/records/{recordID}/documents", educationService.CreateManagerialDocument)
				r.With(authService.RequirePermissions("education.managerial.manage")).Patch("/education/managerial/records/{recordID}/documents/{documentID}", educationService.UpdateManagerialDocument)
				r.With(authService.RequirePermissions("education.managerial.manage")).Delete("/education/managerial/records/{recordID}/documents/{documentID}", educationService.DeleteManagerialDocument)
				r.With(authService.RequirePermissions("education.managerial.manage")).Post("/education/managerial/records/{recordID}/workflow", educationService.CreateManagerialWorkflowStep)
				r.With(authService.RequirePermissions("education.managerial.manage")).Patch("/education/managerial/records/{recordID}/workflow/{stepID}", educationService.UpdateManagerialWorkflowStep)
				r.With(authService.RequirePermissions("education.managerial.manage")).Delete("/education/managerial/records/{recordID}/workflow/{stepID}", educationService.DeleteManagerialWorkflowStep)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/dashboard", educationService.RegulationDashboard)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records", educationService.Regulations)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/{recordID}", educationService.RegulationDetail)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/filters", educationService.RegulationFilters)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/{recordID}/versions", educationService.RegulationVersions)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/{recordID}/versions/{versionID}", educationService.RegulationVersionDetail)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/{recordID}/workflow", educationService.RegulationWorkflowSteps)
				r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/{recordID}/workflow/{stepID}", educationService.RegulationWorkflowStepDetail)
				r.With(authService.RequirePermissions("education.regulations.manage")).Post("/education/regulations/records", educationService.CreateRegulation)
				r.With(authService.RequirePermissions("education.regulations.manage")).Patch("/education/regulations/records/{recordID}", educationService.UpdateRegulation)
				r.With(authService.RequirePermissions("education.regulations.manage")).Delete("/education/regulations/records/{recordID}", educationService.DeleteRegulation)
				r.With(authService.RequirePermissions("education.regulations.manage")).Post("/education/regulations/records/{recordID}/versions", educationService.CreateRegulationVersion)
				r.With(authService.RequirePermissions("education.regulations.manage")).Patch("/education/regulations/records/{recordID}/versions/{versionID}", educationService.UpdateRegulationVersion)
				r.With(authService.RequirePermissions("education.regulations.manage")).Delete("/education/regulations/records/{recordID}/versions/{versionID}", educationService.DeleteRegulationVersion)
				r.With(authService.RequirePermissions("education.regulations.manage")).Post("/education/regulations/records/{recordID}/workflow", educationService.CreateRegulationWorkflowStep)
				r.With(authService.RequirePermissions("education.regulations.manage")).Patch("/education/regulations/records/{recordID}/workflow/{stepID}", educationService.UpdateRegulationWorkflowStep)
				r.With(authService.RequirePermissions("education.regulations.manage")).Delete("/education/regulations/records/{recordID}/workflow/{stepID}", educationService.DeleteRegulationWorkflowStep)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/dashboard", educationService.PersonnelDashboard)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records", educationService.PersonnelRecords)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/{recordID}", educationService.PersonnelRecordDetail)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/filters", educationService.PersonnelFilters)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/{recordID}/assignments", educationService.PersonnelAssignments)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/{recordID}/assignments/{itemID}", educationService.PersonnelAssignmentDetail)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/{recordID}/disciplinary-cases", educationService.PersonnelDisciplinaryCases)
				r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", educationService.PersonnelDisciplinaryCaseDetail)
				r.With(authService.RequirePermissions("education.personnel.files.read")).Get("/education/personnel/records/{recordID}/file-documents", educationService.PersonnelPersonalFileDocuments)
				r.With(authService.RequirePermissions("education.personnel.files.read")).Get("/education/personnel/records/{recordID}/file-documents/{documentID}", educationService.PersonnelPersonalFileDocumentDetail)
				r.With(authService.RequirePermissions("education.personnel.access.read")).Get("/education/personnel/records/{recordID}/access-events", educationService.PersonnelPersonalAccessEvents)
				r.With(authService.RequirePermissions("education.personnel.access.read")).Get("/education/personnel/records/{recordID}/access-events/{eventID}", educationService.PersonnelPersonalAccessEventDetail)
				r.With(authService.RequirePermissions("education.personnel.manage")).Post("/education/personnel/records", educationService.CreatePersonnelRecord)
				r.With(authService.RequirePermissions("education.personnel.manage")).Patch("/education/personnel/records/{recordID}", educationService.UpdatePersonnelRecord)
				r.With(authService.RequirePermissions("education.personnel.manage")).Delete("/education/personnel/records/{recordID}", educationService.DeletePersonnelRecord)
				r.With(authService.RequirePermissions("education.personnel.manage")).Post("/education/personnel/records/{recordID}/assignments", educationService.CreatePersonnelAssignment)
				r.With(authService.RequirePermissions("education.personnel.manage")).Patch("/education/personnel/records/{recordID}/assignments/{itemID}", educationService.UpdatePersonnelAssignment)
				r.With(authService.RequirePermissions("education.personnel.manage")).Delete("/education/personnel/records/{recordID}/assignments/{itemID}", educationService.DeletePersonnelAssignment)
				r.With(authService.RequirePermissions("education.personnel.manage")).Post("/education/personnel/records/{recordID}/disciplinary-cases", educationService.CreatePersonnelDisciplinaryCase)
				r.With(authService.RequirePermissions("education.personnel.manage")).Patch("/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", educationService.UpdatePersonnelDisciplinaryCase)
				r.With(authService.RequirePermissions("education.personnel.manage")).Delete("/education/personnel/records/{recordID}/disciplinary-cases/{itemID}", educationService.DeletePersonnelDisciplinaryCase)
				r.With(authService.RequirePermissions("education.personnel.files.manage")).Post("/education/personnel/records/{recordID}/file-documents", educationService.CreatePersonnelPersonalFileDocument)
				r.With(authService.RequirePermissions("education.personnel.files.manage")).Patch("/education/personnel/records/{recordID}/file-documents/{documentID}", educationService.UpdatePersonnelPersonalFileDocument)
				r.With(authService.RequirePermissions("education.personnel.files.manage")).Delete("/education/personnel/records/{recordID}/file-documents/{documentID}", educationService.DeletePersonnelPersonalFileDocument)
				r.With(authService.RequirePermissions("education.personnel.access.manage")).Post("/education/personnel/records/{recordID}/access-events", educationService.CreatePersonnelPersonalAccessEvent)
				r.With(authService.RequirePermissions("education.personnel.access.manage")).Patch("/education/personnel/records/{recordID}/access-events/{eventID}", educationService.UpdatePersonnelPersonalAccessEvent)
				r.With(authService.RequirePermissions("education.personnel.access.manage")).Delete("/education/personnel/records/{recordID}/access-events/{eventID}", educationService.DeletePersonnelPersonalAccessEvent)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/dashboard", educationService.EvaluationDashboard)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records", educationService.Evaluations)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}", educationService.EvaluationDetail)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/pdf", educationService.EvaluationPDF)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/filters", educationService.EvaluationFilters)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/self-reviews", educationService.EvaluationSelfReviews)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/self-reviews/{itemID}", educationService.EvaluationSelfReviewDetail)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/criteria", educationService.EvaluationCriteria)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/criteria/{itemID}", educationService.EvaluationCriterionDetail)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/appeals", educationService.EvaluationAppeals)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/appeals/{appealID}", educationService.EvaluationAppealDetail)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/appeals/{appealID}/pdf", educationService.EvaluationAppealPDF)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/result-issues", educationService.EvaluationResultIssues)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/result-issues/{itemID}", educationService.EvaluationResultIssueDetail)
				r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/{recordID}/result-issues/{itemID}/pdf", educationService.EvaluationResultIssuePDF)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records", educationService.CreateEvaluation)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Patch("/education/evaluations/records/{recordID}", educationService.UpdateEvaluation)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Delete("/education/evaluations/records/{recordID}", educationService.DeleteEvaluation)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records/{recordID}/self-reviews", educationService.CreateEvaluationSelfReview)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Patch("/education/evaluations/records/{recordID}/self-reviews/{itemID}", educationService.UpdateEvaluationSelfReview)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Delete("/education/evaluations/records/{recordID}/self-reviews/{itemID}", educationService.DeleteEvaluationSelfReview)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records/{recordID}/criteria", educationService.CreateEvaluationCriterion)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Patch("/education/evaluations/records/{recordID}/criteria/{itemID}", educationService.UpdateEvaluationCriterion)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Delete("/education/evaluations/records/{recordID}/criteria/{itemID}", educationService.DeleteEvaluationCriterion)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records/{recordID}/appeals", educationService.CreateEvaluationAppeal)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Patch("/education/evaluations/records/{recordID}/appeals/{appealID}", educationService.UpdateEvaluationAppeal)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Delete("/education/evaluations/records/{recordID}/appeals/{appealID}", educationService.DeleteEvaluationAppeal)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records/{recordID}/result-issues", educationService.CreateEvaluationResultIssue)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Patch("/education/evaluations/records/{recordID}/result-issues/{itemID}", educationService.UpdateEvaluationResultIssue)
				r.With(authService.RequirePermissions("education.evaluations.manage")).Delete("/education/evaluations/records/{recordID}/result-issues/{itemID}", educationService.DeleteEvaluationResultIssue)
				r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/dashboard", educationService.DeclarationDashboard)
				r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/records", educationService.Declarations)
				r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/records/{recordID}", educationService.DeclarationDetail)
				r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/records/filters", educationService.DeclarationFilters)
				r.With(authService.RequirePermissions("education.declarations.manage")).Post("/education/declarations/records", educationService.CreateDeclaration)
				r.With(authService.RequirePermissions("education.declarations.manage")).Patch("/education/declarations/records/{recordID}", educationService.UpdateDeclaration)
				r.With(authService.RequirePermissions("education.declarations.manage")).Delete("/education/declarations/records/{recordID}", educationService.DeleteDeclaration)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/dashboard", educationService.MobilityDashboard)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records", educationService.MobilityCases)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}", educationService.MobilityCaseDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/pdf", educationService.MobilityCasePDF)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/filters", educationService.MobilityFilters)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/documents", educationService.MobilityDocuments)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/documents/{itemID}", educationService.MobilityDocumentDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/scores", educationService.MobilityCriterionScores)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/scores/{itemID}", educationService.MobilityCriterionScoreDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/appeals", educationService.MobilityAppeals)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/appeals/{itemID}", educationService.MobilityAppealDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/appeals/{itemID}/pdf", educationService.MobilityAppealPDF)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/final-decisions", educationService.MobilityFinalDecisions)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/final-decisions/{itemID}", educationService.MobilityFinalDecisionDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/final-decisions/{itemID}/pdf", educationService.MobilityFinalDecisionPDF)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/result-issues", educationService.MobilityResultIssues)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/result-issues/{itemID}", educationService.MobilityResultIssueDetail)
				r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/{recordID}/result-issues/{itemID}/pdf", educationService.MobilityResultIssuePDF)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records", educationService.CreateMobilityCase)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}", educationService.UpdateMobilityCase)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}", educationService.DeleteMobilityCase)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records/{recordID}/documents", educationService.CreateMobilityDocument)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}/documents/{itemID}", educationService.UpdateMobilityDocument)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}/documents/{itemID}", educationService.DeleteMobilityDocument)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records/{recordID}/scores", educationService.CreateMobilityCriterionScore)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}/scores/{itemID}", educationService.UpdateMobilityCriterionScore)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}/scores/{itemID}", educationService.DeleteMobilityCriterionScore)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records/{recordID}/appeals", educationService.CreateMobilityAppeal)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}/appeals/{itemID}", educationService.UpdateMobilityAppeal)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}/appeals/{itemID}", educationService.DeleteMobilityAppeal)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records/{recordID}/final-decisions", educationService.CreateMobilityFinalDecision)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}/final-decisions/{itemID}", educationService.UpdateMobilityFinalDecision)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}/final-decisions/{itemID}", educationService.DeleteMobilityFinalDecision)
				r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records/{recordID}/result-issues", educationService.CreateMobilityResultIssue)
				r.With(authService.RequirePermissions("education.mobility.manage")).Patch("/education/mobility/records/{recordID}/result-issues/{itemID}", educationService.UpdateMobilityResultIssue)
				r.With(authService.RequirePermissions("education.mobility.manage")).Delete("/education/mobility/records/{recordID}/result-issues/{itemID}", educationService.DeleteMobilityResultIssue)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/dashboard", educationService.MeritDashboard)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records", educationService.MeritGrants)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}", educationService.MeritGrantDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/pdf", educationService.MeritGrantPDF)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/filters", educationService.MeritFilters)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/documents", educationService.MeritDocuments)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/documents/{itemID}", educationService.MeritDocumentDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/scores", educationService.MeritCriterionScores)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/scores/{itemID}", educationService.MeritCriterionScoreDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/appeals", educationService.MeritAppeals)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/appeals/{itemID}", educationService.MeritAppealDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/appeals/{itemID}/pdf", educationService.MeritAppealPDF)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/final-decisions", educationService.MeritFinalDecisions)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/final-decisions/{itemID}", educationService.MeritFinalDecisionDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/final-decisions/{itemID}/pdf", educationService.MeritFinalDecisionPDF)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/result-issues", educationService.MeritResultIssues)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/result-issues/{itemID}", educationService.MeritResultIssueDetail)
				r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/{recordID}/result-issues/{itemID}/pdf", educationService.MeritResultIssuePDF)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records", educationService.CreateMeritGrant)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}", educationService.UpdateMeritGrant)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}", educationService.DeleteMeritGrant)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records/{recordID}/documents", educationService.CreateMeritDocument)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}/documents/{itemID}", educationService.UpdateMeritDocument)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}/documents/{itemID}", educationService.DeleteMeritDocument)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records/{recordID}/scores", educationService.CreateMeritCriterionScore)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}/scores/{itemID}", educationService.UpdateMeritCriterionScore)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}/scores/{itemID}", educationService.DeleteMeritCriterionScore)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records/{recordID}/appeals", educationService.CreateMeritAppeal)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}/appeals/{itemID}", educationService.UpdateMeritAppeal)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}/appeals/{itemID}", educationService.DeleteMeritAppeal)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records/{recordID}/final-decisions", educationService.CreateMeritFinalDecision)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}/final-decisions/{itemID}", educationService.UpdateMeritFinalDecision)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}/final-decisions/{itemID}", educationService.DeleteMeritFinalDecision)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records/{recordID}/result-issues", educationService.CreateMeritResultIssue)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Patch("/education/gradatii/records/{recordID}/result-issues/{itemID}", educationService.UpdateMeritResultIssue)
				r.With(authService.RequirePermissions("education.gradatii.manage")).Delete("/education/gradatii/records/{recordID}/result-issues/{itemID}", educationService.DeleteMeritResultIssue)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/dashboard", educationService.PortfolioDashboard)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records", educationService.PortfolioRecords)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}", educationService.PortfolioRecordDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/pdf", educationService.PortfolioRecordPDF)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/filters", educationService.PortfolioFilters)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/documents", educationService.PortfolioDocuments)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/documents/{documentID}", educationService.PortfolioDocumentDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/checklist", educationService.PortfolioChecklistItems)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/checklist/{itemID}", educationService.PortfolioChecklistItemDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/opis", educationService.PortfolioOpisEntries)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/opis/{itemID}", educationService.PortfolioOpisEntryDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/custody", educationService.PortfolioCustodyEvents)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/custody/{itemID}", educationService.PortfolioCustodyEventDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/transfers", educationService.PortfolioTransfers)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/transfers/{itemID}", educationService.PortfolioTransferDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/reviews", educationService.PortfolioReviews)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/{recordID}/reviews/{itemID}", educationService.PortfolioReviewDetail)
				r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/sections", educationService.PortfolioSections)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records", educationService.CreatePortfolioRecord)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}", educationService.UpdatePortfolioRecord)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}", educationService.DeletePortfolioRecord)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/documents", educationService.CreatePortfolioDocument)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/documents/{documentID}", educationService.UpdatePortfolioDocument)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/documents/{documentID}", educationService.DeletePortfolioDocument)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/checklist", educationService.CreatePortfolioChecklistItem)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/checklist/{itemID}", educationService.UpdatePortfolioChecklistItem)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/checklist/{itemID}", educationService.DeletePortfolioChecklistItem)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/opis", educationService.CreatePortfolioOpisEntry)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/opis/{itemID}", educationService.UpdatePortfolioOpisEntry)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/opis/{itemID}", educationService.DeletePortfolioOpisEntry)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/custody", educationService.CreatePortfolioCustodyEvent)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/custody/{itemID}", educationService.UpdatePortfolioCustodyEvent)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/custody/{itemID}", educationService.DeletePortfolioCustodyEvent)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/transfers", educationService.CreatePortfolioTransfer)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/transfers/{itemID}", educationService.UpdatePortfolioTransfer)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/transfers/{itemID}", educationService.DeletePortfolioTransfer)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records/{recordID}/reviews", educationService.CreatePortfolioReview)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Patch("/education/portfolios/records/{recordID}/reviews/{itemID}", educationService.UpdatePortfolioReview)
				r.With(authService.RequirePermissions("education.portfolios.manage")).Delete("/education/portfolios/records/{recordID}/reviews/{itemID}", educationService.DeletePortfolioReview)
			})

			r.With(authService.RequirePermissions("gdpr.read")).Get("/gdpr/dashboard", gdprService.Dashboard)
			r.With(authService.RequirePermissions("gdpr.read")).Get("/gdpr/config", gdprService.Config)
			r.With(authService.RequirePermissions("gdpr.policies.read")).Get("/gdpr/retention-policies", gdprService.RetentionPolicies)
			r.With(authService.RequirePermissions("gdpr.policies.read")).Get("/gdpr/retention-policies/filters", gdprService.RetentionPolicyFilters)
			r.With(authService.RequirePermissions("gdpr.policies.manage")).Post("/gdpr/retention-policies", gdprService.CreateRetentionPolicy)
			r.With(authService.RequirePermissions("gdpr.requests.read")).Get("/gdpr/subject-requests", gdprService.SubjectRequests)
			r.With(authService.RequirePermissions("gdpr.requests.read")).Get("/gdpr/subject-requests/filters", gdprService.SubjectRequestFilters)
			r.With(authService.RequirePermissions("gdpr.requests.manage")).Post("/gdpr/subject-requests", gdprService.CreateSubjectRequest)
			r.With(authService.RequirePermissions("gdpr.exports.read")).Get("/gdpr/exports/dashboard", gdprService.ExportDashboard)
			r.With(authService.RequirePermissions("gdpr.exports.read")).Get("/gdpr/exports", gdprService.SubjectExports)
			r.With(authService.RequirePermissions("gdpr.exports.read")).Get("/gdpr/exports/filters", gdprService.SubjectExportFilters)
			r.With(authService.RequirePermissions("gdpr.exports.manage")).Post("/gdpr/exports", gdprService.CreateSubjectExport)
			r.With(authService.RequirePermissions("gdpr.publication.read")).Get("/gdpr/publication-reviews/dashboard", gdprService.PublicationDashboard)
			r.With(authService.RequirePermissions("gdpr.publication.read")).Get("/gdpr/publication-reviews", gdprService.PublicationReviews)
			r.With(authService.RequirePermissions("gdpr.publication.read")).Get("/gdpr/publication-reviews/filters", gdprService.PublicationReviewFilters)
			r.With(authService.RequirePermissions("gdpr.publication.manage")).Post("/gdpr/publication-reviews", gdprService.CreatePublicationReview)
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("egueducation api listening", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}
}

func buildBootstrapConfig(cfg config.Config, r *http.Request) map[string]any {
	modules := []string{"registratura", "workflow", "earchiva", "education", "admin"}
	if cfg.EnableGDPRFeatures {
		modules = append(modules, "gdpr")
	}

	frontendOrigin := strings.TrimSpace(cfg.FrontendOrigin)
	if frontendOrigin == "" && r != nil {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		frontendOrigin = scheme + "://" + r.Host
	}

	customerName := "Școala Bălotești"
	customerShortName := "Bălotești"
	customerWebsite := frontendOrigin
	if parsed, err := url.Parse(frontendOrigin); err == nil && parsed.Hostname() != "" {
		host := parsed.Hostname()
		switch {
		case strings.Contains(host, "scoalabalotesti"):
			customerName = "Școala Bălotești"
			customerShortName = "Bălotești"
		default:
			customerName = "EguEducation"
			customerShortName = "EguEducation"
		}
	}

	return map[string]any{
		"version":             "1.0.0",
		"environment":         cfg.Environment,
		"apiBaseUrl":          "/api",
		"oidcAuthority":       "/api/oidc",
		"oidcIssuer":          cfg.OIDCIssuer,
		"oidcClientId":        cfg.OIDCClientID,
		"oidcDesktopClientId": cfg.OIDCDesktopClient,
		"authScope":           "openid profile email phone offline_access",
		"defaultLocale":       "ro",
		"availableLocales":    []string{"ro", "en"},
		"theme": map[string]string{
			"family": "material3-expressive",
			"brand":  "red-rose",
		},
		"features": map[string]any{
			"offlineMode":       false,
			"pushNotifications": false,
			"passkeys":          cfg.EnablePasskeys,
			"wallet":            cfg.EnableWallet,
			"smsOtp":            cfg.EnableSMSOTP,
			"gdpr":              cfg.EnableGDPRFeatures,
		},
		"license": map[string]any{
			"licenseType":     "education",
			"uatId":           nil,
			"uatName":         nil,
			"uatSiruta":       nil,
			"enabledFeatures": modules,
			"isActive":        true,
		},
		"customer": map[string]any{
			"name":            customerName,
			"shortName":       customerShortName,
			"logoUrl":         nil,
			"defaultLanguage": "ro",
			"websiteUrl":      customerWebsite,
			"websiteLabel":    customerName,
		},
		"map": map[string]any{
			"defaultCenter": []float64{26.0765, 44.6204},
			"defaultZoom":   13,
		},
		"modules": map[string]any{
			"enabled": modules,
		},
		"institutionTypes": []string{"scoala", "liceu", "gradinita", "other"},
		"isMaster":         false,
		"service": map[string]any{
			"id":    "egueducation",
			"title": customerName,
		},
	}
}

func cors(frontendOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, DPoP, X-Request-ID, Accept-Language")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
