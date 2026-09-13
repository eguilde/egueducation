package db

import (
	"context"
	"strings"
	"testing"
)

func TestSchoolPolicyCutoverPreflightReadyForDual(t *testing.T) {
	complete := SchoolPolicyCutoverPreflight{Phase: "legacy"}
	if !complete.ReadyForDual() {
		t.Fatal("gap-free legacy scope should pass the structural preflight")
	}

	cases := map[string]func(*SchoolPolicyCutoverPreflight){
		"wrong phase":                  func(p *SchoolPolicyCutoverPreflight) { p.Phase = "dual" },
		"profile gap":                  func(p *SchoolPolicyCutoverPreflight) { p.UnmappedProfiles = 1 },
		"assignment gap":               func(p *SchoolPolicyCutoverPreflight) { p.UnmappedAssignments = 1 },
		"override gap":                 func(p *SchoolPolicyCutoverPreflight) { p.UnmappedOverrides = 1 },
		"evaluation gap":               func(p *SchoolPolicyCutoverPreflight) { p.UnmappedEvaluations = 1 },
		"pack provenance gap":          func(p *SchoolPolicyCutoverPreflight) { p.MissingPackProvenance = 1 },
		"missing effective date":       func(p *SchoolPolicyCutoverPreflight) { p.InputsWithoutEffectiveDate = 1 },
		"duplicate decision":           func(p *SchoolPolicyCutoverPreflight) { p.InputsWithMultipleDecisions = 1 },
		"consumer provenance mismatch": func(p *SchoolPolicyCutoverPreflight) { p.ConsumerProvenanceMismatches = 1 },
		"blocking issue":               func(p *SchoolPolicyCutoverPreflight) { p.OpenBlockingIssues = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			report := complete
			mutate(&report)
			if report.ReadyForDual() {
				t.Fatalf("preflight accepted unsafe state: %+v", report)
			}
		})
	}
}

func TestSchoolPolicyCutoverPreflightIsSessionScopedAndComplete(t *testing.T) {
	lower := strings.ToLower(schoolPolicyCutoverPreflightSQL)
	for _, needle := range []string{
		"current_tenant_code()",
		"current_institution_id()",
		"school_profile_cutover_identity",
		"school_operation_policy_bindings_v2",
		"school_operation_policy_overrides_v2",
		"school_policy_evaluation_cutover_identity",
		"school_operation_policy_input_bindings",
		"effective_on is null",
		"having count(*) > 1",
		"consumer_mismatches",
		"severity='blocking'",
	} {
		if !strings.Contains(lower, strings.ToLower(needle)) {
			t.Fatalf("preflight query lacks %q", needle)
		}
	}
	if strings.Contains(lower, "$1") || strings.Contains(lower, "$2") {
		t.Fatal("preflight must derive scope from the pinned PostgreSQL session")
	}
}

func TestSchoolPolicyCutoverPreflightRejectsMissingSessionPool(t *testing.T) {
	_, err := ReadSchoolPolicyCutoverPreflight(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "requires a database session") {
		t.Fatalf("missing pool error = %v", err)
	}
}
