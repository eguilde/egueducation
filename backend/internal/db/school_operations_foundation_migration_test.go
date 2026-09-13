package db

import (
	"os"
	"strings"
	"testing"
)

func TestSchoolOperationsFoundationHasScopedImmutableContracts(t *testing.T) {
	b, err := os.ReadFile("migrations/0136_school_operations_foundation.sql")
	if err != nil { t.Fatal(err) }
	s := string(b)
	for _, required := range []string{
		"create table school_contracts", "policy_evaluation_id uuid not null", "expected_version integer not null", "school_operation_idempotency",
		"school_contract_versions", "school_utility_readings", "school_compliance_corrective_actions", "school_operations_outbox",
		"force row level security", "foreign key(tenant_code,institution_id)", "school_contracts_no_final_delete",
	} { if !strings.Contains(s, required) { t.Fatalf("migration lacks %q", required) } }
}
