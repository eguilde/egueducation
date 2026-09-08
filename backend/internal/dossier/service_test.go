package dossier

import (
	"context"
	"strings"
	"testing"
)

func TestEvaluateAllowsRecordsWithoutSourceAndFailsClosedWhenNoRequirements(t *testing.T) {
	ready, missing, err := Evaluate(context.Background(), nil, "workflow", false, RelationCounts{}, PurposeSubmit)
	if err != nil || !ready || len(missing) != 0 {
		t.Fatalf("records without a source must not be blocked: ready=%v missing=%v err=%v", ready, missing, err)
	}
}

func TestRequirementFilteringAndRelationCounts(t *testing.T) {
	requirements := []Requirement{
		{RelationType: "primary", RequiredForReadiness: true},
		{RelationType: "decision", RequiredForSubmit: true},
		{RelationType: "gdpr_basis", RequiredForApprove: true},
	}
	if got := filterRequirements(requirements, PurposeReadiness); len(got) != 1 || got[0].RelationType != "primary" {
		t.Fatalf("readiness filter = %#v", got)
	}
	if got := relationCount(RelationCounts{Decision: 2}, "decision"); got != 2 {
		t.Fatalf("decision count = %d", got)
	}
	if got := relationCount(RelationCounts{}, "untrusted"); got != 0 {
		t.Fatalf("unknown relation must fail closed, got %d", got)
	}
}

func TestDossierSQLUsesSuppliedExpressionsAndAllRelationCategories(t *testing.T) {
	counts := CountSQL("stats")
	for _, expected := range []string{"stats.total_count", "stats.primary_count", "stats.gdpr_basis_count"} {
		if !strings.Contains(counts, expected) {
			t.Fatalf("CountSQL missing %q: %s", expected, counts)
		}
	}
	join := LateralJoinSQL("stats", "wi.source_module", "wi.source_record_id")
	for _, expected := range []string{"wi.source_module", "wi.source_record_id", ") stats on true"} {
		if !strings.Contains(join, expected) {
			t.Fatalf("LateralJoinSQL missing %q: %s", expected, join)
		}
	}
}
