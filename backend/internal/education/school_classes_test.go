package education

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func TestSchoolClassRequestValidationAndAllowedValues(t *testing.T) {
	valid := CreateSchoolClassRequest{ClassCode: "CLS-2026-a", ClassName: "  IV A ", SchoolYear: "2026-2027", GradeLevel: "IV", StudyShift: ""}
	if !normalizeSchoolClassRequest(&valid) || valid.StudyShift != "day" || valid.ClassName != "IV A" {
		t.Fatalf("valid class request was not normalized: %#v", valid)
	}
	invalid := CreateSchoolClassRequest{ClassCode: "CLS", ClassName: "A", SchoolYear: "2026-2027", GradeLevel: "IV", StudyShift: "night"}
	if normalizeSchoolClassRequest(&invalid) {
		t.Fatal("unsupported study shift must be rejected")
	}
}

// The API and RLS must share the database's one canonical temporal roster
// predicate. This guards list/count/detail paths against a future shortcut
// that could disclose historical or future pupils to a read_assigned teacher.
func TestAssignedTeacherHandlersUseCanonicalCurrentRosterPredicates(t *testing.T) {
	for _, test := range []struct {
		file, predicate string
		minimum         int
	}{
		{"school_classes_handlers.go", "public.education_classes_actor_is_assigned(class_row.id)", 2},
		{"school_classes_handlers.go", "public.education_classes_current_roster_enrolment(enrolment.id)", 3},
		{"school_classes_assignments.go", "public.education_classes_current_roster_enrolment(enrolment.id)", 2},
		{"school_classes_assignments.go", "public.education_classes_actor_owns_current_homeroom(assignment.id)", 2},
	} {
		body, err := os.ReadFile(test.file)
		if err != nil {
			t.Fatalf("read %s: %v", test.file, err)
		}
		if got := strings.Count(string(body), test.predicate); got < test.minimum {
			t.Errorf("%s has %d canonical predicate occurrences, want at least %d", test.file, got, test.minimum)
		}
	}
}

func TestSchoolStudentAndTemporalCommandValidation(t *testing.T) {
	student := CreateSchoolStudentRequest{StudentCode: "STU-1", FirstName: " Ana ", LastName: " Pop ", Status: ""}
	if !normalizeSchoolStudentRequest(&student) || student.Status != "active" {
		t.Fatalf("student normalization failed: %#v", student)
	}
	enrolment := CreateSchoolEnrolmentRequest{StudentID: "11111111-1111-4111-8111-111111111111", ClassID: "22222222-2222-4222-8222-222222222222", EnrolledFrom: "2026-09-01", Status: ""}
	if !normalizeSchoolEnrolment(&enrolment) || enrolment.Status != "active" {
		t.Fatalf("enrolment normalization failed: %#v", enrolment)
	}
	assignment := CreateSchoolHomeroomAssignmentRequest{ClassID: "22222222-2222-4222-8222-222222222222", PersonnelID: "33333333-3333-4333-8333-333333333333", AppUserID: "44444444-4444-4444-8444-444444444444", AssignedFrom: "2026-09-01"}
	if !normalizeSchoolHomeroom(&assignment) {
		t.Fatal("complete homeroom assignment must be accepted")
	}
	assignment.AppUserID = ""
	if normalizeSchoolHomeroom(&assignment) {
		t.Fatal("homeroom assignment without canonical application user must be rejected")
	}
}

func TestSchoolUUIDRejectsMalformedAndNonCanonicalValues(t *testing.T) {
	valid, ok := schoolUUID("AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA")
	if !ok || valid != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("canonical UUID was not normalized: %q, %t", valid, ok)
	}
	for _, value := range []string{"", "not-a-uuid", "{aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa}", "aaaaaaaaaaaaaaaa4aaa8aaaaaaaaaaaaaaaa", "00000000-0000-0000-0000-000000000000"} {
		if _, ok := schoolUUID(value); ok {
			t.Errorf("invalid UUID %q was accepted", value)
		}
	}
}

func TestSchoolPathUUIDWritesBadRequestBeforeDatabaseAccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/education/school/classes/not-a-uuid", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("classID", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()
	if _, ok := requireSchoolPathUUID(recorder, req, "classID", "education_class_id_invalid"); ok {
		t.Fatal("malformed path UUID was accepted")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), "education_class_id_invalid") {
		t.Fatalf("validation response lacks error code: %s", recorder.Body.String())
	}
}

func TestSchoolUUIDRequestFieldsAreRejectedBeforeSQL(t *testing.T) {
	enrolment := CreateSchoolEnrolmentRequest{StudentID: "student", ClassID: "class", EnrolledFrom: "2026-09-01", Status: "active"}
	if normalizeSchoolEnrolment(&enrolment) {
		t.Fatal("enrolment request with malformed identifiers was accepted")
	}
	homeroom := CreateSchoolHomeroomAssignmentRequest{ClassID: "class", PersonnelID: "personnel", AppUserID: "user", AssignedFrom: "2026-09-01"}
	if normalizeSchoolHomeroom(&homeroom) {
		t.Fatal("homeroom request with malformed identifiers was accepted")
	}
}

func TestSchoolEnrolmentNamesUseTheAssignmentOptionOrder(t *testing.T) {
	if !strings.Contains(schoolEnrolmentColumns, "concat_ws(' ', student.first_name, student.last_name)") {
		t.Fatalf("enrolment label must use given-name then family-name: %s", schoolEnrolmentColumns)
	}
	body, err := os.ReadFile("school_classes_assignments.go")
	if err != nil {
		t.Fatalf("read enrolment handler: %v", err)
	}
	for _, order := range []string{
		"concat_ws(' ', student.first_name, student.last_name)",
		"concat_ws(' ', student.last_name, student.first_name)",
	} {
		if !strings.Contains(string(body), order) {
			t.Errorf("student-name filter must accept %q", order)
		}
	}
}

func TestSchoolListSortMappingsAreExplicitAndSafe(t *testing.T) {
	for _, test := range []struct {
		mapping                   map[string]string
		requested, fallback, want string
	}{
		{schoolClassSortColumns, "school_year", "class_name", "class_row.school_year"},
		{schoolStudentSortColumns, "status", "last_name", "student.status"},
		{schoolStudentSortColumns, "birth_date", "last_name", "student.birth_date"},
		{schoolEnrolmentSortColumns, "enrolled_from", "enrolled_from", "enrolment.enrolled_from"},
		{schoolEnrolmentSortColumns, "enrolled_until", "enrolled_from", "enrolment.enrolled_until"},
		{schoolHomeroomSortColumns, "assigned_from", "assigned_from", "assignment.assigned_from"},
		{schoolHomeroomSortColumns, "class_name", "assigned_from", "class_row.class_name"},
		{schoolHomeroomSortColumns, "assigned_until", "assigned_from", "assignment.assigned_until"},
		{schoolEnrolmentSortColumns, "unknown", "enrolled_from", "enrolment.enrolled_from"},
	} {
		if got := schoolSortColumn(test.mapping, test.requested, test.fallback); got != test.want {
			t.Fatalf("sort %q = %q, want %q", test.requested, got, test.want)
		}
	}
}

func TestSchoolTemporalAndRelationshipFiltersAreAccepted(t *testing.T) {
	values := url.Values{"filter.class_id": {"class-id"}, "filter.student_id": {"student-id"}, "filter.enrolled_from": {"2026-09-01"}, "filter.enrolled_until": {"2026-10-01"}, "filter.assigned_from": {"2026-09-01"}, "filter.assigned_until": {"2026-10-01"}, "filter.class_name": {"IV A"}, "filter.birth_date": {"2018-01-01"}}
	enrolment := httpx.ParsePageQuery(values, map[string]struct{}{"enrolled_from": {}, "enrolled_until": {}, "status": {}, "student_name": {}, "class_name": {}}, []string{"enrolled_from", "enrolled_until", "status", "student_name", "class_name", "class_id", "student_id"})
	if enrolment.Filters["class_id"] != "class-id" || enrolment.Filters["student_id"] != "student-id" || enrolment.Filters["enrolled_from"] != "2026-09-01" || enrolment.Filters["enrolled_until"] != "2026-10-01" {
		t.Fatalf("enrolment filters lost: %#v", enrolment.Filters)
	}
	assignment := httpx.ParsePageQuery(values, map[string]struct{}{"assigned_from": {}, "assigned_until": {}, "teacher_name": {}, "class_name": {}}, []string{"assigned_from", "assigned_until", "teacher_name", "class_name", "class_id"})
	if assignment.Filters["class_id"] != "class-id" || assignment.Filters["assigned_from"] != "2026-09-01" || assignment.Filters["assigned_until"] != "2026-10-01" || assignment.Filters["class_name"] != "IV A" {
		t.Fatalf("assignment filters lost: %#v", assignment.Filters)
	}
	student := httpx.ParsePageQuery(values, map[string]struct{}{"student_code": {}, "first_name": {}, "last_name": {}, "status": {}, "birth_date": {}}, []string{"student_code", "first_name", "last_name", "status", "birth_date", "class_id"})
	if student.Filters["birth_date"] != "2018-01-01" {
		t.Fatalf("student birth date filter lost: %#v", student.Filters)
	}
}
