package db

import (
	"strings"
	"testing"
)

func TestTenantKeyedSessionContextMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0124_tenant_keyed_session_context.sql")
	if err != nil {
		t.Fatalf("read 0124 migration: %v", err)
	}
	contents := strings.ToLower(string(body))
	for _, required := range []string{
		"add column if not exists tenant_code",
		"drop constraint if exists app_session_context_pkey",
		"primary key (user_id, tenant_code)",
		"foreign key (tenant_code, institution_id)",
		"references app_tenants (code, institution_id)",
		"join app_memberships membership",
		"membership.active = true",
		"start_date <= current_date",
		"idx_app_session_context_tenant_user",
	} {
		if !strings.Contains(contents, required) {
			t.Errorf("0124 missing tenant session-context requirement %q", required)
		}
	}
}
