package db

import (
	"strings"
	"testing"
)

func TestEducationCurrentRosterAccessMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0118_education_current_roster_access.sql")
	if err != nil {
		t.Fatalf("read 0118 migration: %v", err)
	}
	contents := strings.ToLower(string(body))
	for _, required := range []string{
		"education_classes_current_roster_enrolment",
		"education_classes_actor_owns_current_homeroom",
		"student.status = 'active'",
		"enrolment.status = 'active'",
		"enrolment.enrolled_from <= current_date",
		"enrolment.enrolled_until is null or enrolment.enrolled_until >= current_date",
		"education_classes_actor_is_assigned(enrolment.class_id)",
		"create policy education_student_enrolments_read",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("0118 missing current-roster contract %q", required)
		}
	}
	if strings.Contains(contents, "disable row level security") {
		t.Fatal("0118 must not weaken row-level security")
	}
}
