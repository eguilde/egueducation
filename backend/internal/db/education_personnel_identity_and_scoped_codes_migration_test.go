package db

import (
	"strings"
	"testing"
)

func TestEducationPersonnelIdentityAndScopedCodesMigrationHasDurableBackstops(t *testing.T) {
	eligibilityMigration, err := migrationFiles.ReadFile("migrations/0110_education_temporal_membership_contract.sql")
	if err != nil {
		t.Fatalf("read temporal membership migration: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(eligibilityMigration)), "create or replace function public.education_membership_is_eligible") {
		t.Fatal("0110 must define the canonical eligibility helper before 0111 consumes it")
	}
	contents, err := migrationFiles.ReadFile("migrations/0111_education_personnel_identity_and_scoped_codes.sql")
	if err != nil {
		t.Fatalf("read personnel identity migration: %v", err)
	}
	text := strings.ToLower(string(contents))
	for _, required := range []string{
		"add column if not exists app_user_id uuid references app_users",
		"uq_education_personnel_institution_app_user",
		"identity_row.verified_at is not null",
		"public.education_membership_is_eligible(identity_row.user_id, tenant.code, person.institution_id, null)",
		"public.education_membership_is_eligible(new.owner_user_id, tenant.code, new.institution_id, null)",
		"min(user_id::text)::uuid",
		"duplicate code already exists within an institution",
		"education_portfolios', 'portfolio_code",
		"education_decisions', 'decision_code",
		"education_managerial_dossiers', 'dossier_code",
		"education_regulations', 'regulation_code",
		"education_evaluations', 'evaluation_code",
		"education_declarations', 'declaration_code",
		"education portfolio owner personnel must be canonically linked",
		"education portfolio owner_user_id is immutable",
		"education personnel identity cannot be reassigned while referenced by a portfolio",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("0111 migration is missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") {
		t.Fatal("0111 must not weaken row-level security")
	}
}
