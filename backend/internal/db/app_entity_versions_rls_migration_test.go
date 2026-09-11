package db

import (
	"strings"
	"testing"
)

const appEntityVersionsRLSMigration = "migrations/0132_app_entity_versions_rls.sql"
const appEntityVersionsScopedIdentityMigration = "migrations/0133_app_entity_versions_scoped_identity.sql"

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

func TestAppEntityVersionsScopedIdentityMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile(appEntityVersionsScopedIdentityMigration)
	if err != nil {
		t.Fatalf("read %s: %v", appEntityVersionsScopedIdentityMigration, err)
	}
	text := strings.ToLower(string(body))
	for _, required := range []string{
		"drop constraint if exists app_entity_versions_entity_table_entity_id_version_no_key",
		"add constraint app_entity_versions_scope_entity_version_key",
		"unique (tenant_code, institution_id, entity_table, entity_id, version_no)",
		"create or replace function public.record_entity_version()",
		"security invoker",
		"set search_path = pg_catalog, public",
		"pg_advisory_xact_lock",
		"where tenant_code = tenant_value",
		"and institution_id = institution_value",
		"and entity_table = tg_table_name",
		"and entity_id = entity_id_value",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("%s is missing scoped identity invariant %q", appEntityVersionsScopedIdentityMigration, required)
		}
	}
	if strings.Contains(text, "security definer") {
		t.Fatal("entity-version trigger must not bypass tenant RLS")
	}
	if strings.Index(text, "pg_advisory_xact_lock") > strings.Index(text, "select coalesce(max(version_no)") {
		t.Fatal("scoped version allocation must lock before reading the next version")
	}
}
