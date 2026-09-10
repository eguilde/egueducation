package db

import (
	"os"
	"strings"
	"testing"
)

func TestEducationTemporalMembershipContractMigration(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("migrations/0110_education_temporal_membership_contract.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := strings.ToLower(string(body))
	for _, required := range []string{
		"education_membership_is_eligible",
		"user_row.status = 'active'",
		"tenant.active",
		"membership.start_date <= current_date",
		"membership.end_date is null or membership.end_date >= current_date",
		"trg_education_role_delegations_00_temporal_guard",
		"create or replace function public.education_actor_can_transfer_portfolio",
		"public.current_institution_id()",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("0110 is missing temporal authorization contract %q", required)
		}
	}
}
