package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolRegulatoryStage1BHasNormalizedScopedEvidence(t *testing.T) {
	files := []string{"migrations/0137_school_regulatory_profile_v2.sql", "migrations/0138_school_offerings_authorizations.sql", "migrations/0139_school_funding_procurement.sql", "migrations/0140_school_contracts_quality_network.sql", "migrations/0141_school_operation_policy_context.sql", "migrations/0142_school_stage1b_rls_contract.sql", "migrations/0143_school_stage1b_policy_enrichment.sql"}
	all := ""
	for _, f := range files {
		b, e := os.ReadFile(f)
		if e != nil {
			t.Fatal(e)
		}
		all += string(b)
	}
	for _, needle := range []string{"legal_form in ('public','private')", "school_confessional_profiles", "profile_version integer not null", "school_confessional_private_only", "school_offering_authorizations", "replaces_authorization_id", "school_offering_authorization_parent_guard", "school_location_authorization_dependency_conflict", "education_offering_authorization_dependency_conflict", "institution.offerings.read", "institution.offerings.manage", "school_funding_subject_unique", "school_funding_eligibility_evaluations", "determination in ('applicable','not_applicable','indeterminate')", "school_operation_policy_inputs", "foreign key(tenant_code,institution_id,profile_id,profile_version)", "revalidates_evaluation_id", "effective_excl exclude using gist", "force row level security", "school_stage1b_snapshot_immutable", "foreign key(tenant_code,institution_id)"} {
		if !strings.Contains(all, needle) {
			t.Fatalf("Stage 1B contract lacks %q", needle)
		}
	}
	if strings.Contains(all, "applicable boolean") {
		t.Fatal("procurement applicability must have one tri-state source of truth")
	}
	offerings, err := os.ReadFile("migrations/0138_school_offerings_authorizations.sql")
	if err != nil {
		t.Fatal(err)
	}
	offeringsSQL := string(offerings)
	if !strings.Contains(offeringsSQL, "app_permissions(code,label)") || strings.Contains(offeringsSQL, "app_permissions(code,description)") {
		t.Fatal("offering permissions must target the canonical app_permissions.label column")
	}
}
