package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/eguilde/egueducation/internal/config"
)

type testPinger struct {
	err error
}

func TestEducationPortfolioValorificationCRUDRoutesAreRegistered(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read router source: %v", err)
	}
	router := string(source)
	permissionGuard := `RequireAnyEducationPermissions("education.portfolios.school.manage", "education.portfolios.manage")`
	for _, route := range []string{
		`.Post("/education/portfolios/records/{recordID}/valorifications", educationService.CreatePortfolioValorification)`,
		`.Patch("/education/portfolios/records/{recordID}/valorifications/{itemID}", educationService.UpdatePortfolioValorification)`,
		`.Delete("/education/portfolios/records/{recordID}/valorifications/{itemID}", educationService.DeletePortfolioValorification)`,
	} {
		position := strings.Index(router, route)
		if position < 0 {
			t.Fatalf("missing valorification route %s", route)
		}
		lineStart := strings.LastIndex(router[:position], "\n") + 1
		if !strings.Contains(router[lineStart:position], permissionGuard) {
			t.Fatalf("valorification route is missing school/legacy manage RBAC: %s", route)
		}
	}
}

func TestSchoolOperationalSurfacesAreRegisteredWithRBAC(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read router source: %v", err)
	}
	router := string(source)
	tests := []struct {
		route      string
		permission string
	}{
		{`.Get("/education/classes", educationService.SchoolClasses)`, "education.classes.read_assigned"},
		{`.Get("/education/classes/assignment-options", educationService.SchoolAssignmentOptions)`, "education.classes.manage"},
		{`.Get("/education/classes/{classID}", educationService.SchoolClassDetail)`, "education.classes.read_assigned"},
		{`.Post("/education/classes", educationService.CreateSchoolClass)`, "education.classes.manage"},
		{`.Patch("/education/classes/{classID}", educationService.UpdateSchoolClass)`, "education.classes.manage"},
		{`.Delete("/education/classes/{classID}", educationService.DeleteSchoolClass)`, "education.classes.manage"},
		{`.Get("/education/students", educationService.SchoolStudents)`, "education.classes.read_assigned"},
		{`.Get("/education/students/{studentID}", educationService.SchoolStudentDetail)`, "education.classes.read_assigned"},
		{`.Post("/education/students", educationService.CreateSchoolStudent)`, "education.classes.manage"},
		{`.Get("/education/class-enrolments", educationService.SchoolEnrolments)`, "education.classes.read_assigned"},
		{`.Get("/education/class-enrolments/{enrolmentID}", educationService.SchoolEnrolmentDetail)`, "education.classes.read_assigned"},
		{`.Post("/education/class-enrolments", educationService.CreateSchoolEnrolment)`, "education.classes.manage"},
		{`.Get("/education/homeroom-assignments", educationService.SchoolHomeroomAssignments)`, "education.classes.read_assigned"},
		{`.Get("/education/homeroom-assignments/{assignmentID}", educationService.SchoolHomeroomAssignmentDetail)`, "education.classes.read_assigned"},
		{`.Post("/education/homeroom-assignments", educationService.CreateSchoolHomeroomAssignment)`, "education.classes.manage"},
		{`.Get("/education/reports", educationService.SchoolReportCatalog)`, "education.compliance.read"},
		{`.Get("/education/reports/{reportCode}", educationService.SchoolReport)`, "education.compliance.read"},
		{`.Get("/education/reports/{reportCode}/csv", educationService.SchoolReportCSV)`, "education.reports.export_sensitive"},
		{`.Get("/education/reports/{reportCode}/pdf", educationService.SchoolReportPDF)`, "education.reports.export_sensitive"},
		{`.Get("/education/signatures", educationService.ListSignedArtifactEvidence)`, "education.signatures.read"},
		{`.Get("/education/signatures/eligible-artifacts", educationService.EligibleSignedArtifacts)`, "education.signatures.manage"},
		{`.Get("/education/signatures/eligible-archive-versions", educationService.EligibleSignatureArchiveVersions)`, "education.signatures.manage"},
		{`.Get("/education/signatures/{evidenceID}", educationService.SignedArtifactEvidenceDetail)`, "education.signatures.read"},
		{`.Post("/education/signatures", educationService.SubmitSignedArtifactEvidence)`, "education.signatures.manage"},
		{`.Post("/education/signatures/{evidenceID}/revalidate", educationService.RevalidateSignedArtifactEvidence)`, "education.signatures.validate"},
		{`.Get("/education/secretariat/cockpit", educationService.SecretariatCockpit)`, "education.cockpit.secretariat.read"},
		{`.Get("/education/hr/cockpit", educationService.HRCockpit)`, "education.cockpit.hr.read"},
		{`.Get("/education/committee/cockpit", educationService.CommitteeCockpit)`, "education.cockpit.committee.read"},
		{`.Get("/education/inspector/cockpit", educationService.InspectorCockpit)`, "education.cockpit.inspector.read"},
	}
	for _, tt := range tests {
		position := strings.Index(router, tt.route)
		if position < 0 {
			t.Errorf("missing route %s", tt.route)
			continue
		}
		contextStart := position - 1200
		if contextStart < 0 {
			contextStart = 0
		}
		if !strings.Contains(router[contextStart:position], tt.permission) {
			t.Errorf("route %s is missing RBAC permission %s", tt.route, tt.permission)
		}
	}
}

func (p testPinger) Ping(context.Context) error {
	return p.err
}

func TestReadinessHandlerReportsDatabaseFailure(t *testing.T) {
	tests := []struct {
		name     string
		pinger   testPinger
		wantCode int
	}{
		{name: "database ready", pinger: testPinger{}, wantCode: http.StatusOK},
		{name: "database unavailable", pinger: testPinger{err: errors.New("unavailable")}, wantCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			readinessHandler(tt.pinger, nil).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantCode)
			}
		})
	}
}

func TestReadinessHandlerReportsBuildRevision(t *testing.T) {
	recorder := httptest.NewRecorder()
	readinessHandler(testPinger{}, nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	var payload struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if payload.Revision != sourceRevision {
		t.Fatalf("revision = %q, want %q", payload.Revision, sourceRevision)
	}
}

func TestReadinessHandlerReportsSignatureVerifierFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	readinessHandler(testPinger{}, func(context.Context) error { return errors.New("verifier unavailable") }).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	var payload struct {
		SignatureVerifier string `json:"signature_verifier"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if payload.SignatureVerifier != "error" {
		t.Fatalf("signature_verifier = %q, want error", payload.SignatureVerifier)
	}
}

func TestLivenessHandlerDoesNotRequireDatabase(t *testing.T) {
	recorder := httptest.NewRecorder()

	livenessHandler(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestBuildBootstrapConfigIncludesLegacyAndRuntimeFields(t *testing.T) {
	cfg := config.Config{
		FrontendOrigin:     "https://scoalabalotesti.eguilde.cloud",
		Environment:        "production",
		OIDCIssuer:         "https://scoalabalotesti.eguilde.cloud/api/oidc",
		OIDCClientID:       "egueducation-spa",
		OIDCDesktopClient:  "egueducation-desktop",
		EnablePasskeys:     true,
		EnableWallet:       true,
		EnableSMSOTP:       true,
		EnableGDPRFeatures: true,
	}

	req := httptest.NewRequest("GET", "https://scoalabalotesti.eguilde.cloud/api/config", nil)
	payload := buildBootstrapConfig(cfg, req)

	if got, ok := payload["oidcClientId"].(string); !ok || got != "egueducation-spa" {
		t.Fatalf("oidcClientId = %#v, want eg educ client", payload["oidcClientId"])
	}

	if got, ok := payload["apiBaseUrl"].(string); !ok || got != "/api" {
		t.Fatalf("apiBaseUrl = %#v, want /api", payload["apiBaseUrl"])
	}

	if got, ok := payload["institutionId"].(string); !ok || got == "" {
		t.Fatalf("institutionId = %#v, want non-empty string", payload["institutionId"])
	}

	customer, ok := payload["customer"].(map[string]any)
	if !ok {
		t.Fatalf("customer = %#v, want object", payload["customer"])
	}
	if got, ok := customer["name"].(string); !ok || got == "" {
		t.Fatalf("customer.name = %#v, want non-empty string", customer["name"])
	}

	modules, ok := payload["modules"].(map[string]any)
	if !ok {
		t.Fatalf("modules = %#v, want object", payload["modules"])
	}
	enabled, ok := modules["enabled"].([]string)
	if !ok {
		t.Fatalf("modules.enabled = %#v, want []string", modules["enabled"])
	}
	if len(enabled) == 0 {
		t.Fatal("modules.enabled should not be empty")
	}

	features, ok := payload["features"].(map[string]any)
	if !ok {
		t.Fatalf("features = %#v, want object", payload["features"])
	}
	if got, ok := features["gdpr"].(bool); !ok || !got {
		t.Fatalf("features.gdpr = %#v, want true", features["gdpr"])
	}
}
