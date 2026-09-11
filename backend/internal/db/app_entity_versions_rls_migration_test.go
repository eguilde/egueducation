package db

import (
	"strings"
	"testing"
)

const appEntityVersionsRLSMigration = "migrations/0132_app_entity_versions_rls.sql"

func TestAppEntityVersionsRLSMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile(appEntityVersionsRLSMigration)
	if err != nil {
		t.Fatalf("read %s: %v", appEntityVersionsRLSMigration, err)
	}
	text := strings.ToLower(string(body))
	for _, required := range []string{
		"alter table public.app_entity_versions enable row level security",
		"alter table public.app_entity_versions force row level security",
		"create policy app_entity_versions_tenant_read",
		"create policy app_entity_versions_tenant_append",
		"tenant_code = public.current_tenant_code()",
		"institution_id = public.current_institution_id()",
		"for select",
		"for insert",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("%s is missing RLS invariant %q", appEntityVersionsRLSMigration, required)
		}
	}
	if strings.Contains(text, "for update") || strings.Contains(text, "for delete") {
		t.Fatal("entity-version history must remain append-only for application roles")
	}
}
