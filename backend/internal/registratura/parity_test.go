package registratura

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func requestWithDocumentID(request *http.Request, documentID string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("documentID", documentID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func TestCanonicalWorkflowTransitions(t *testing.T) {
	cases := []struct{ status, action, want string }{
		{"INCOMING", "assign_department", "ALOCAT_COMPARTIMENT"},
		{"ALOCAT_COMPARTIMENT", "assign_user", "IN_LUCRU"},
		{"ALOCAT_COMPARTIMENT", "claim", "IN_LUCRU"},
		{"IN_LUCRU", "send_for_approval", "FLUX_APROBARE"},
		{"FLUX_APROBARE", "approve", "FINALIZAT"},
		{"FLUX_APROBARE", "reject", "IN_LUCRU"},
	}
	for _, tc := range cases {
		got, ok := workflowTransition(tc.status, tc.action)
		if !ok || got != tc.want {
			t.Fatalf("%s/%s = %q, %v", tc.status, tc.action, got, ok)
		}
	}
	if _, ok := workflowTransition("FINALIZAT", "claim"); ok {
		t.Fatal("finalized document must not be claimable")
	}
}

func TestWorkflowActionRequiresExactPositiveVersionBeforeDatabaseAccess(t *testing.T) {
	for _, body := range []string{
		`{"action":"claim"}`,
		`{"action":"claim","expected_version":0}`,
		`{"action":"claim","expected_version":-1}`,
	} {
		recorder := httptest.NewRecorder()
		request := requestWithDocumentID(httptest.NewRequest(http.MethodPost, "/api/registratura/documents/document-id/workflow-actions", strings.NewReader(body)), "document-id")
		(&Service{}).ApplyDocumentWorkflowAction(recorder, request)
		if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"code":"workflow_expected_version_required"`) {
			t.Fatalf("payload %s returned %d %s", body, recorder.Code, recorder.Body.String())
		}
	}
}

func TestValidateWorkflowActionFieldsRejectsForgedAssignmentFields(t *testing.T) {
	departmentID, userID := "department-id", "user-id"
	cases := []struct {
		name string
		req  DocumentWorkflowActionRequest
		code string
	}{
		{"department with user", DocumentWorkflowActionRequest{Action: "assign_department", DepartmentID: &departmentID, UserID: &userID}, "workflow_user_not_allowed"},
		{"user with department", DocumentWorkflowActionRequest{Action: "assign_user", DepartmentID: &departmentID, UserID: &userID}, "workflow_department_not_allowed"},
		{"approval with assignment", DocumentWorkflowActionRequest{Action: "approve", UserID: &userID}, "workflow_assignment_not_allowed"},
		{"rejection with assignment", DocumentWorkflowActionRequest{Action: "reject", DepartmentID: &departmentID}, "workflow_assignment_not_allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWorkflowActionFields(tc.req)
			if err == nil || err.Error() != tc.code {
				t.Fatalf("validateWorkflowActionFields() = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestWorkflowActionRejectsOverlongNoteBeforeDatabaseAccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"action":"claim","expected_version":1,"note":"` + strings.Repeat("x", 501) + `"}`
	request := requestWithDocumentID(httptest.NewRequest(http.MethodPost, "/api/registratura/documents/document-id/workflow-actions", strings.NewReader(body)), "document-id")
	(&Service{}).ApplyDocumentWorkflowAction(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"code":"workflow_note_too_long"`) {
		t.Fatalf("returned %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestFluxQueryIsBoundedAndUsesOnlyWhitelistedSorts(t *testing.T) {
	q := parseFluxQuery(url.Values{
		"page":      {"2"},
		"limit":     {"101"},
		"sortField": {"status; drop table registratura_documents"},
		"sortOrder": {"desc"},
		"continut":  {"transfer"},
	})
	if q.page != 2 || q.pageSize != 100 || q.direction != "desc" || q.sort != "" {
		t.Fatalf("unexpected bounded flux query: %#v", q)
	}
	if q.filters["continut"] != "transfer" {
		t.Fatalf("flux content filter missing: %#v", q.filters)
	}
	if got := fluxSortColumn("status; drop table"); got != "d.registered_at" {
		t.Fatalf("unsafe sort fallback = %q", got)
	}
}

func TestDocumentGlobalSearchSurvivesCostestiAliasNormalization(t *testing.T) {
	q := documentListPageQuery(url.Values{"q": {"Popescu"}})
	if q.Filters["q"] != "Popescu" {
		t.Fatalf("q filter must be allowlisted and forwarded, got %#v", q.Filters)
	}
}

func TestNormalizeDocumentStatus(t *testing.T) {
	for in, want := range map[string]string{"": "INCOMING", "registered": "INCOMING", "in_workflow": "IN_LUCRU", "archived": "FINALIZAT", "anulat": "ANULAT"} {
		if got := normalizeDocumentStatus(in); got != want {
			t.Fatalf("%q = %q, want %q", in, got, want)
		}
	}
}

func TestStageDocumentAttachmentFailsClosed(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/registratura/documents/document-id/attachments/stage", strings.NewReader(`{"file_name":"orphan.pdf"}`))

	(&Service{}).StageDocumentAttachment(recorder, request)

	if recorder.Code != http.StatusGone {
		t.Fatalf("stage status = %d, want %d", recorder.Code, http.StatusGone)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"attachment_upload_required"`) {
		t.Fatalf("stage response = %s, want attachment_upload_required", recorder.Body.String())
	}
}

func TestDocumentListPageQueryAcceptsContractAliasesWithExplicitPrecedence(t *testing.T) {
	aliased := documentListPageQuery(url.Values{
		"page":                 {"2"},
		"limit":                {"20"},
		"sortBy":               {"subject"},
		"sortDir":              {"desc"},
		"q":                    {"contract"},
		"filter.registered_at": {"2026-09-08"},
		"filter.due_date":      {"2026-09-30"},
	})
	if aliased.Page != 2 || aliased.PageSize != 20 || aliased.Sort != "subject" || aliased.Direction != "desc" {
		t.Fatalf("aliases parsed as %+v", aliased)
	}
	if aliased.Filters["registered_at"] != "2026-09-08" || aliased.Filters["due_date"] != "2026-09-30" {
		t.Fatalf("date filters parsed as %+v", aliased.Filters)
	}
	if aliased.Filters["q"] != "contract" {
		t.Fatalf("global search filter parsed as %+v", aliased.Filters)
	}

	legacy := documentListPageQuery(url.Values{
		"pageSize":  {"30"},
		"limit":     {"20"},
		"sort":      {"status"},
		"sortBy":    {"subject"},
		"direction": {"asc"},
		"sortDir":   {"desc"},
	})
	if legacy.PageSize != 30 || legacy.Sort != "status" || legacy.Direction != "asc" {
		t.Fatalf("legacy parameters must take precedence, got %+v", legacy)
	}
}

func TestCreateDocumentRejectsPriorCamelCaseAndUUIDRegistryContracts(t *testing.T) {
	cases := []string{
		`{"registryId":"c8400d62-0167-4b5c-a02d-e4d1ce123456","subject":"Cerere","document_type":"DOCUMENT","direction":"intrare","confidentiality":"normal"}`,
		`{"registru_id":"c8400d62-0167-4b5c-a02d-e4d1ce123456","subject":"Cerere","document_type":"DOCUMENT","direction":"intrare","confidentiality":"normal"}`,
	}
	for _, body := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/registratura/documents", strings.NewReader(body))
		(&Service{}).CreateDocument(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("payload %s returned %d, want %d", body, recorder.Code, http.StatusBadRequest)
		}
		if !strings.Contains(recorder.Body.String(), `"code":"invalid_document_payload"`) {
			t.Fatalf("payload %s response = %s", body, recorder.Body.String())
		}
	}
}

func TestDocumentDateFiltersMatchTheirDocumentedNames(t *testing.T) {
	where, args := buildDocumentFilters("institution-1", map[string]string{
		"registered_at": "2026-09-08",
		"due_date":      "2026-09-30",
	})
	if !strings.Contains(where, "d.registered_at::date =") || !strings.Contains(where, "d.due_date =") {
		t.Fatalf("documented date filters missing from SQL: %s", where)
	}
	if len(args) != 3 || args[1] != "2026-09-08" || args[2] != "2026-09-30" {
		t.Fatalf("documented date filter arguments = %#v", args)
	}
}

func TestDocumentGlobalSearchCoversVisibleTextColumns(t *testing.T) {
	where, args := buildDocumentFilters("institution-1", map[string]string{"q": "Cerere"})
	for _, column := range []string{"d.registry_number", "d.external_number", "d.subject", "d.correspondent", "d.assigned_to"} {
		if !strings.Contains(where, column) {
			t.Fatalf("global search SQL does not cover %s: %s", column, where)
		}
	}
	if len(args) != 2 || args[1] != "%cerere%" {
		t.Fatalf("global search arguments = %#v", args)
	}
}

func TestWorkflowResponseQueryReliesOnTenantRLSNotRegistryReadVisibility(t *testing.T) {
	workflowQuery := documentByIDQuery(false)
	if strings.Contains(workflowQuery, "app.actor_subject") || strings.Contains(workflowQuery, "registratura_registry_departments") {
		t.Fatalf("authorized workflow response query unexpectedly requires registry visibility: %s", workflowQuery)
	}
	if !strings.Contains(workflowQuery, "from registratura_documents") || !strings.Contains(workflowQuery, "where id::text = $1") {
		t.Fatalf("workflow response query is not document-scoped: %s", workflowQuery)
	}

	standardQuery := documentByIDQuery(true)
	if !strings.Contains(standardQuery, "app.actor_subject") || !strings.Contains(standardQuery, "registratura_registry_departments") {
		t.Fatalf("standard document query lost registry visibility enforcement: %s", standardQuery)
	}
}
