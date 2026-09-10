package db

import (
	"strings"
	"testing"
)

func TestTenantRBACAndFixedOTPTestUserMigration(t *testing.T) {
	const migrationName = "migrations/0125_tenant_rbac_and_fixed_otp_test_user.sql"
	body, err := migrationFiles.ReadFile(migrationName)
	if err != nil {
		t.Fatalf("read %s: %v", migrationName, err)
	}
	text := string(body)
	for _, required := range []string{
		"role.code in ('admin', 'super_admin', 'e2e_canary')",
		"permission.code like 'education.%'",
		"apply_complete_tenant_role_mapping",
		"trg_app_permissions_complete_tenant_roles",
		"20c36b31-d7e9-4a4b-b6df-42adc5b2913d",
		"drop table if exists oidc_production_e2e_challenges",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("%s is missing %q", migrationName, required)
		}
	}
	if strings.Contains(text, "app_user_platform_roles (user_id, role_code)") || strings.Contains(text, "platform_super_admin'") {
		t.Fatal("tenant role completion must never grant a platform role")
	}
}
