package db

import (
	"strings"
	"testing"
)

func TestEducationClassesDelegationRLSMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0122_education_classes_delegation_rls.sql")
	if err != nil {
		t.Fatalf("read 0122 migration: %v", err)
	}
	contents := strings.ToLower(string(body))
	for _, required := range []string{
		"create or replace function public.education_classes_actor_has_permission",
		"education_role_delegations",
		"delegation.resource_type = 'institution'",
		"delegation.status = 'accepted'",
		"delegation.valid_from <= current_date",
		"delegation.valid_until is null or delegation.valid_until >= current_date",
		"director_adjunct",
		"delegator_user_id",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("0122 missing delegation RLS requirement %q", required)
		}
	}
}
