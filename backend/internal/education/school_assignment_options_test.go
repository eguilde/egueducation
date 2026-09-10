package education

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestParseSchoolAssignmentOptionsQuery(t *testing.T) {
	query, err := parseSchoolAssignmentOptionsQuery(url.Values{
		"kind":     {" Teachers "},
		"q":        {" Popescu "},
		"page":     {"3"},
		"pageSize": {"500"},
	})
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if query.Kind != schoolAssignmentOptionTeachers || query.Search != "Popescu" || query.Page != 3 || query.PageSize != 100 {
		t.Fatalf("unexpected normalized query: %#v", query)
	}
	for _, kind := range []string{schoolAssignmentOptionClasses, schoolAssignmentOptionStudents, schoolAssignmentOptionTeachers} {
		if _, err := parseSchoolAssignmentOptionsQuery(url.Values{"kind": {kind}}); err != nil {
			t.Errorf("valid kind %q rejected: %v", kind, err)
		}
	}
	if _, err := parseSchoolAssignmentOptionsQuery(url.Values{"kind": {"users"}}); err == nil {
		t.Fatal("unsupported kind must be rejected")
	}
}

func TestSchoolAssignmentOptionPlansRemainTenantAndInstitutionScoped(t *testing.T) {
	for _, kind := range []string{schoolAssignmentOptionClasses, schoolAssignmentOptionStudents, schoolAssignmentOptionTeachers} {
		plan, err := schoolAssignmentOptionsPlan(kind)
		if err != nil {
			t.Fatalf("plan %q: %v", kind, err)
		}
		for _, required := range []string{"public.current_tenant_code()", "institution_id = $1", "$2"} {
			if !strings.Contains(plan.fromWhere+plan.searchClause, required) {
				t.Errorf("plan %q missing scope/search predicate %q", kind, required)
			}
		}
	}
	teachers, _ := schoolAssignmentOptionsPlan(schoolAssignmentOptionTeachers)
	for _, required := range []string{"person.status = 'active'", "person.app_user_id is not null", "education_membership_is_eligible", "person.app_user_id::text", "person.id::text"} {
		if !strings.Contains(teachers.fromWhere+teachers.selectColumns, required) {
			t.Errorf("teacher plan missing canonical active identity condition %q", required)
		}
	}
}

func TestSchoolAssignmentOptionsHandlerRechecksManagePermission(t *testing.T) {
	assertSourceContains(t, "school_assignment_options.go", "requireSchoolClassesAccess(w, r, true)")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/education/classes/assignment-options?kind=classes", nil)
	new(Service).SchoolAssignmentOptions(recorder, request)
	if recorder.Code != http.StatusForbidden || !strings.Contains(recorder.Body.String(), "education_classes_access_denied") {
		t.Fatalf("unauthenticated direct handler call = %d %s, want rechecked 403", recorder.Code, recorder.Body.String())
	}
}

func assertSourceContains(t *testing.T, path string, expected string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(source), expected) {
		t.Fatalf("%s missing %q", path, expected)
	}
}
