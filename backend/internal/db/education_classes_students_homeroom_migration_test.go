package db

import (
	"strings"
	"testing"
)

func TestEducationClassesStudentsHomeroomMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0112_education_classes_students_homeroom.sql")
	if err != nil {
		t.Fatalf("read 0112 migration: %v", err)
	}
	contents := string(body)
	for _, required := range []string{
		"education_school_classes", "education_students", "education_student_enrolments", "education_class_homeroom_assignments",
		"exclude using gist", "education_membership_is_eligible", "canonical institutional identity pair",
		"force row level security", "education_classes_actor_is_assigned",
		"migration owner must be superuser or bypassrls",
		"education_classes_actor_has_permission('education.classes.read_assigned')",
		"education.classes.read", "education.classes.manage", "education.classes.read_assigned",
	} {
		if !strings.Contains(strings.ToLower(contents), strings.ToLower(required)) {
			t.Errorf("0112 missing required School contract %q", required)
		}
	}
	if strings.Contains(contents, "disable row level security") {
		t.Fatal("0112 must not weaken RLS")
	}
}
