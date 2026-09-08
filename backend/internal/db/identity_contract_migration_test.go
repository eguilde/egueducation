package db

import (
	"strings"
	"testing"
)

func TestIdentityContractMigrationHasRequiredFoundations(t *testing.T) {
	const migrationName = "migrations/0083_identity_contract_foundation.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)

	for _, required := range []string{
		"create table if not exists app_user_identities",
		"unique (identity_type, normalized_value)",
		"create table if not exists app_platform_roles",
		"create table if not exists app_user_platform_roles",
		"create table if not exists app_tenant_authorization_versions",
		"alter table app_tenant_authorization_versions force row level security",
		"tenant_code = public.current_tenant_code()",
		"record_tenant_authorization_change",
		"trg_app_user_roles_authz_version",
		"trg_app_user_permissions_authz_version",
		"trg_app_user_modules_authz_version",
		"trg_app_memberships_authz_version",
		"trg_app_user_platform_roles_authz_version",
		"set_config('app.tenant_id', membership_row.tenant_code, true)",
		"set_config('app.tenant_id', tenant_row.tenant_code, true)",
		"tenant-balotesti",
		"inst-balotesti",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("identity migration is missing %q", required)
		}
	}

	for _, operator := range []string{
		"user-thomas-galambos",
		"user-stelian-fedorca",
		"user-diana-ilhan",
	} {
		if !strings.Contains(text, operator) {
			t.Errorf("identity migration is missing %s", operator)
		}
	}

	if strings.Contains(text, "tenant_code = public.current_tenant_code())\n\twith check (public.can_bypass_tenant_rls") {
		t.Error("authorization-version RLS must not grant an interactive super-admin bypass")
	}
}

func TestAuthorizationVersionCascadeSafetyMigrationPreservesTenantRLS(t *testing.T) {
	const migrationName = "migrations/0085_authorization_version_cascade_safety.sql"
	contents, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(contents)

	for _, required := range []string{
		"create or replace function public.bump_tenant_authorization_version",
		"not exists (select 1 from app_users where id = p_user_id)",
		"on conflict (tenant_code, user_id) do update",
		"exception when foreign_key_violation",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("authorization cascade migration is missing %q", required)
		}
	}
	if strings.Contains(text, "disable row level security") || strings.Contains(text, "no force row level security") {
		t.Error("authorization cascade migration must not weaken authorization-version RLS")
	}
}
