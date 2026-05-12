package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
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
	authService := auth.NewService(cfg, smsService, pool)
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

	router.Route("/api", func(r chi.Router) {
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
		r.Post("/auth/request-sms", authService.RequestSMSOTP)
		r.Post("/auth/verify-sms", authService.VerifySMSOTP)
		r.Post("/auth/session/exchange", authService.ExchangeSession)
		r.Post("/auth/logout", authService.Logout)
		r.Get("/oidc/.well-known/openid-configuration", authService.Discovery)
		r.Get("/oidc/jwks", authService.JWKS)
		r.Get("/oidc/authorize", authService.Authorize)
		r.Get("/oidc/consent/request", authService.ConsentRequest)
		r.Post("/oidc/consent/decision", authService.ConsentDecision)
		r.Post("/oidc/token", authService.Token)
		r.Post("/oidc/revoke", authService.Revoke)
		r.Get("/oidc/userinfo", authService.UserInfo)

		r.Group(func(r chi.Router) {
			r.Use(authService.RequireAuthenticated)

			r.Get("/me", authService.SessionContext)

			r.With(authService.RequirePermissions("admin.read")).Get("/admin/dashboard", adminService.Dashboard)
			r.With(authService.RequirePermissions("admin.users.read")).Get("/admin/users", adminService.ListUsers)
			r.With(authService.RequirePermissions("admin.users.read")).Get("/admin/users/filters", adminService.UserFilters)
			r.With(authService.RequirePermissions("admin.users.manage")).Post("/admin/users", adminService.UpsertUser)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/roles", adminService.ListRoles)
			r.With(authService.RequirePermissions("admin.roles.manage")).Post("/admin/roles", adminService.UpsertRole)
			r.With(authService.RequirePermissions("admin.roles.read")).Get("/admin/role-assignments", adminService.ListUserRoleAssignments)
			r.With(authService.RequirePermissions("admin.roles.manage")).Post("/admin/role-assignments", adminService.UpsertUserRoleAssignment)
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
			r.With(authService.RequirePermissions("admin.identity.read")).Get("/admin/oidc/consents", adminService.ListOIDCConsents)
			r.With(authService.RequirePermissions("admin.identity.manage")).Post("/admin/oidc/consents/revoke", adminService.RevokeOIDCConsent)
			r.With(authService.RequirePermissions("admin.identity.read")).Get("/admin/oidc/sessions", adminService.ListOIDCSessions)
			r.With(authService.RequirePermissions("admin.identity.manage")).Post("/admin/oidc/sessions/revoke", adminService.RevokeOIDCSession)
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
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/lookup", registraturaService.LookupDocuments)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}", registraturaService.GetDocument)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}/versions", registraturaService.ListDocumentVersions)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/{documentID}/versions", registraturaService.CreateDocumentVersion)
			r.With(authService.RequirePermissions("registratura.read")).Get("/registratura/documents/{documentID}/attachments", registraturaService.ListDocumentAttachments)
			r.With(authService.RequirePermissions("registratura.manage")).Post("/registratura/documents/{documentID}/attachments", registraturaService.CreateDocumentAttachment)
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

			r.With(authService.RequirePermissions("education.governance.read")).Get("/education/dashboard", educationService.GovernanceDashboard)
			r.With(authService.RequirePermissions("education.read")).Get("/education/taxonomies", educationService.ListTaxonomies)
			r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings", educationService.GovernanceMeetings)
			r.With(authService.RequirePermissions("education.governance.read")).Get("/education/governance/meetings/filters", educationService.GovernanceFilters)
			r.With(authService.RequirePermissions("education.governance.manage")).Post("/education/governance/meetings", educationService.CreateGovernanceMeeting)
			r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/dashboard", educationService.DecisionDashboard)
			r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/records", educationService.GovernanceDecisions)
			r.With(authService.RequirePermissions("education.decisions.read")).Get("/education/decisions/records/filters", educationService.DecisionFilters)
			r.With(authService.RequirePermissions("education.decisions.manage")).Post("/education/decisions/records", educationService.CreateGovernanceDecision)
			r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/dashboard", educationService.ManagerialDashboard)
			r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records", educationService.ManagerialDossiers)
			r.With(authService.RequirePermissions("education.managerial.read")).Get("/education/managerial/records/filters", educationService.ManagerialFilters)
			r.With(authService.RequirePermissions("education.managerial.manage")).Post("/education/managerial/records", educationService.CreateManagerialDossier)
			r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/dashboard", educationService.RegulationDashboard)
			r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records", educationService.Regulations)
			r.With(authService.RequirePermissions("education.regulations.read")).Get("/education/regulations/records/filters", educationService.RegulationFilters)
			r.With(authService.RequirePermissions("education.regulations.manage")).Post("/education/regulations/records", educationService.CreateRegulation)
			r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/dashboard", educationService.PersonnelDashboard)
			r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records", educationService.PersonnelRecords)
			r.With(authService.RequirePermissions("education.personnel.read")).Get("/education/personnel/records/filters", educationService.PersonnelFilters)
			r.With(authService.RequirePermissions("education.personnel.manage")).Post("/education/personnel/records", educationService.CreatePersonnelRecord)
			r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/dashboard", educationService.EvaluationDashboard)
			r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records", educationService.Evaluations)
			r.With(authService.RequirePermissions("education.evaluations.read")).Get("/education/evaluations/records/filters", educationService.EvaluationFilters)
			r.With(authService.RequirePermissions("education.evaluations.manage")).Post("/education/evaluations/records", educationService.CreateEvaluation)
			r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/dashboard", educationService.DeclarationDashboard)
			r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/records", educationService.Declarations)
			r.With(authService.RequirePermissions("education.declarations.read")).Get("/education/declarations/records/filters", educationService.DeclarationFilters)
			r.With(authService.RequirePermissions("education.declarations.manage")).Post("/education/declarations/records", educationService.CreateDeclaration)
			r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/dashboard", educationService.MobilityDashboard)
			r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records", educationService.MobilityCases)
			r.With(authService.RequirePermissions("education.mobility.read")).Get("/education/mobility/records/filters", educationService.MobilityFilters)
			r.With(authService.RequirePermissions("education.mobility.manage")).Post("/education/mobility/records", educationService.CreateMobilityCase)
			r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/dashboard", educationService.MeritDashboard)
			r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records", educationService.MeritGrants)
			r.With(authService.RequirePermissions("education.gradatii.read")).Get("/education/gradatii/records/filters", educationService.MeritFilters)
			r.With(authService.RequirePermissions("education.gradatii.manage")).Post("/education/gradatii/records", educationService.CreateMeritGrant)
			r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/dashboard", educationService.PortfolioDashboard)
			r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records", educationService.PortfolioRecords)
			r.With(authService.RequirePermissions("education.portfolios.read")).Get("/education/portfolios/records/filters", educationService.PortfolioFilters)
			r.With(authService.RequirePermissions("education.portfolios.manage")).Post("/education/portfolios/records", educationService.CreatePortfolioRecord)

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
