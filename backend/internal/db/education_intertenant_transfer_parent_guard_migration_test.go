package db

import (
	"strings"
	"testing"
)

func TestEducationIntertenantTransferParentGuardMigrationContract(t *testing.T) {
	const migrationName = "migrations/0109_education_intertenant_transfer_parent_guard.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)
	for _, required := range []string{
		"security definer",
		"set search_path = pg_catalog, public",
		"rolsuper or rolbypassrls",
		"for update of portfolio",
		"old.routing_version is distinct from new.routing_version",
		"portfolio transfer routing version is immutable",
		"portfolio.withdrawn_at",
		"portfolio.legal_hold_active",
		"old.source_tenant_code",
		"new.source_tenant_code",
		"revoke all on function public.enforce_education_intertenant_transfer_parent_guard() from public",
		"when (new.routing_version = 1)",
		"when (old.routing_version = 1 or new.routing_version = 1)",
		"when (old.routing_version = 1)",
		"before insert or update or delete on public.education_portfolio_transfers",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("inter-tenant transfer parent guard migration is missing %q", required)
		}
	}
}
