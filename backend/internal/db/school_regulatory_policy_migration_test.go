package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolRegulatoryPolicyFoundationMigrationContract(t *testing.T) {
	raw, err := os.ReadFile("migrations/0135_school_regulatory_policy_foundation.sql")
	if err != nil {
		t.Fatal(err)
	}
	source := strings.ToLower(string(raw))
	for _, table := range []string{
		"school_institution_profiles", "school_policy_pack_versions", "school_policy_assignments",
		"school_policy_overrides", "school_policy_evaluations",
	} {
		if !strings.Contains(source, "create table "+table) {
			t.Errorf("migration does not create %s", table)
		}
		if !strings.Contains(source, "alter table %i force row level security") {
			t.Error("migration does not force row level security")
		}
	}
	for _, required := range []string{
		"tenant_code = public.current_tenant_code()",
		"institution_id = public.current_institution_id()",
		"school policy evaluations are immutable",
		"school_profiles_scope_id_version_unique",
		"school_policy_assignments_profile_fk",
		"education_publications_policy_evaluation_fk",
		"education.publication.manage",
		"school_profiles_effective_period_excl",
		"education_publications_require_policy_evaluation",
		"drop policy if exists tenant_isolation on education_publications",
		"'unclassified'",
		"legal-form.ro.public",
		"legal-form.ro.private",
		"legal-form.ro.confessional",
		"funding.public",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("migration missing contract fragment %q", required)
		}
	}
}
