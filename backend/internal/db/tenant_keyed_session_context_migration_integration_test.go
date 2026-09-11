package db

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantKeyedSessionContextUpgradeIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("tenant-keyed session-context integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	tests := []struct {
		name             string
		institutionID    string
		addSecondTenant  bool
		wantTenantCode   string
		wantErrorMessage string
	}{
		{name: "one tenant binds deterministically", institutionID: "inst-001", wantTenantCode: "tenant-egueducation"},
		{name: "two tenant codes fail closed", institutionID: "inst-001", addSecondTenant: true, wantErrorMessage: "0124 session context mapping is ambiguous"},
		{name: "missing tenant fails closed", institutionID: "inst-unmapped", wantErrorMessage: "0124 session context mapping is ambiguous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			isolated := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
			pool, err := pgxpool.NewWithConfig(ctx, isolated.databaseConfig.Copy())
			if err != nil {
				t.Fatalf("open isolated migration database: %v", err)
			}
			t.Cleanup(pool.Close)
			if err := Migrate(ctx, pool); err != nil {
				t.Fatalf("apply current migrations: %v", err)
			}

			if _, err := pool.Exec(ctx, `
				delete from app_session_context;
				alter table app_session_context drop constraint app_session_context_tenant_institution_fkey;
				alter table app_session_context drop constraint app_session_context_pkey;
				alter table app_session_context alter column tenant_code drop not null;
				alter table app_session_context add constraint app_session_context_pkey primary key (user_id);
				delete from schema_migrations where version = '0124_tenant_keyed_session_context.sql';
			`); err != nil {
				t.Fatalf("restore legacy session-context shape: %v", err)
			}

			fixtureSuffix := strings.ReplaceAll(uuid.NewString(), "-", "")
			if tt.addSecondTenant {
				if _, err := pool.Exec(ctx, `alter table app_tenants drop constraint app_tenants_institution_id_key`); err != nil {
					t.Fatalf("remove historical one-tenant-per-institution constraint: %v", err)
				}
				if _, err := pool.Exec(ctx, `
					insert into app_tenants (
						code, subdomain, institution_id, display_name, short_name, root_org_unit_code, active
					) values ($1, $2, 'inst-001', 'Ambiguous tenant', 'Ambiguous', 'unit-root', true)
				`, "tenant-ambiguous-"+fixtureSuffix, "ambiguous-"+fixtureSuffix); err != nil {
					t.Fatalf("seed ambiguous tenant mapping: %v", err)
				}
			}

			userID := uuid.New()
			if _, err := pool.Exec(ctx, `
				insert into app_users (id, sub, name, email, phone_number, status, email_verified, phone_number_verified)
				values ($1, $2, 'Legacy session user', $3, '', 'active', false, false)
			`, userID, "legacy-session-"+fixtureSuffix, "legacy-session-"+fixtureSuffix+"@example.test"); err != nil {
				t.Fatalf("seed legacy user: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				insert into app_session_context (
					user_id, tenant_code, institution_id, institution_name, auth_methods, gdpr_capabilities
				) values ($1, null, $2, 'Legacy institution', '{}', '{}')
			`, userID, tt.institutionID); err != nil {
				t.Fatalf("seed legacy session context: %v", err)
			}

			migrationErr := Migrate(ctx, pool)
			if tt.wantErrorMessage == "" {
				if migrationErr != nil {
					t.Fatalf("apply deterministic 0124 upgrade: %v", migrationErr)
				}
			} else if migrationErr == nil || !strings.Contains(migrationErr.Error(), tt.wantErrorMessage) {
				t.Fatalf("migration error = %v, want message containing %q", migrationErr, tt.wantErrorMessage)
			}

			var tenantCode *string
			if err := pool.QueryRow(ctx, `select tenant_code from app_session_context where user_id = $1`, userID).Scan(&tenantCode); err != nil {
				t.Fatalf("read upgraded session context: %v", err)
			}
			var ledgerCount int
			if err := pool.QueryRow(ctx, `select count(*) from schema_migrations where version = '0124_tenant_keyed_session_context.sql'`).Scan(&ledgerCount); err != nil {
				t.Fatalf("read 0124 ledger state: %v", err)
			}

			if tt.wantErrorMessage != "" {
				if tenantCode != nil {
					t.Fatalf("failed migration guessed tenant_code %q", *tenantCode)
				}
				if ledgerCount != 0 {
					t.Fatalf("failed migration ledger count = %d, want 0", ledgerCount)
				}
				return
			}
			if tenantCode == nil || *tenantCode != tt.wantTenantCode {
				t.Fatalf("tenant_code = %v, want %q", tenantCode, tt.wantTenantCode)
			}
			if ledgerCount != 1 {
				t.Fatalf("successful migration ledger count = %d, want 1", ledgerCount)
			}
		})
	}
}
