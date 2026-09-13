package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolPolicyV2CutoverExpandIsAdditiveAndScoped(t *testing.T) {
	b, err := os.ReadFile("migrations/0145_school_policy_v2_cutover_expand.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(b)
	for _, needle := range []string{
		"school_profile_cutover_identity", "school_institution_profile_api_projection",
		"school_operation_policy_bindings_v2", "school_operation_policy_overrides_v2",
		"school_operation_policy_input_bindings", "school_policy_evaluation_cutover_identity",
		"school_policy_cutover_state", "school_regulatory_migration_issues",
		"status in ('unclassified', 'draft', 'approved', 'active', 'superseded')",
		"policy_evaluation_v2_id uuid", "education_publications_policy_evaluation_v2_fk",
		"school_operation_policy_evaluations_v2_scope_id_input_unique", "effective_on date;",
		"school_policy_overrides_scope_id_unique", "input_id, profile_v2_id, profile_v2_version, profile_series_id",
		"binding_id, profile_v2_id, profile_v2_version, profile_series_id, policy_pack_version_id",
		"school_policy_pack_versions_scope_id_checksum_unique", "exact_profile_pack_unique",
		"school_institution_profiles_v2_scope_id_version_series_unique",
		"school_policy_pack_versions_scope_id_code_version_unique",
		"profile_v2_id, profile_v2_version, profile_series_id) references school_institution_profiles_v2",
		"policy_pack_version_id, pack_code, policy_pack_version) references school_policy_pack_versions",
		"school_profile_api_projection_immutable", "school_profile_cutover_identity_versioning",
		"id uuid not null default gen_random_uuid()",
		"decision_kind in ('operation', 'publication', 'contract', 'compliance', 'migration', 'legacy_import')",
		"schema_version integer not null default 1", "active_effective_excl",
		"policy_evaluation_id is not null or policy_evaluation_v2_id is not null",
		"accreditation_reference", "has_legal_personality", "tax_identifier", "vat_profile",
		"force row level security", "school_stage1b_snapshot_immutable", "not valid",
		"foreign key (tenant_code, institution_id)",
	} {
		if !strings.Contains(strings.ToLower(sql), strings.ToLower(needle)) {
			t.Fatalf("0145 cutover contract lacks %q", needle)
		}
	}
	lowerSQL := strings.ToLower(sql)
	if strings.Contains(lowerSQL, "drop table") || strings.Contains(lowerSQL, "delete from school_policy") || strings.Contains(lowerSQL, "update school_operation_policy_") {
		t.Fatal("0145 must preserve legacy policy evidence")
	}
}
