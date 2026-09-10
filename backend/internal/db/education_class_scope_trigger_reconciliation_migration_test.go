package db

import (
	"strings"
	"testing"
)

func TestEducationClassScopeTriggerReconciliationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0129_education_class_scope_trigger_reconciliation.sql")
	if err != nil {
		t.Fatalf("read class-scope trigger reconciliation migration: %v", err)
	}
	text := strings.ToLower(string(body))
	for _, required := range []string{
		"create or replace function public.enforce_education_class_scope()",
		"if tg_table_name = 'education_student_enrolments' then",
		"if tg_table_name = 'education_class_homeroom_assignments' then",
		"if tg_op = 'insert' then",
		"identity_changed := true",
		"school record tenant and institution are immutable",
		"new.tenant_code is distinct from old.tenant_code",
		"new.institution_id is distinct from old.institution_id",
		"security definer",
		"set search_path = pg_catalog, public",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("class-scope trigger reconciliation is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"tg_table_name = 'education_student_enrolments' and",
		"tg_op = 'insert' or new.personnel_id",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("class-scope trigger reconciliation retains unsafe compound expression %q", forbidden)
		}
	}
}
