package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantGrantRLSMigrationContract(t *testing.T) {
	contents, err := migrationFiles.ReadFile("migrations/0087_tenant_grant_rls.sql")
	if err != nil {
		t.Fatalf("read tenant-grant RLS migration: %v", err)
	}
	text := string(contents)

	for _, required := range []string{
		"'app_user_roles'",
		"'app_user_permissions'",
		"'app_user_modules'",
		"enable row level security",
		"force row level security",
		"drop policy if exists",
		"create policy tenant_grant_isolation",
		"tenant_code is not null and tenant_code = public.current_tenant_code()",
		"app_unresolved_legacy_grants",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("tenant-grant RLS migration is missing %q", required)
		}
	}
	if strings.Contains(text, "can_bypass_tenant_rls") {
		t.Fatal("tenant-grant RLS must not provide a platform-superadmin database bypass")
	}
}

// TestTenantGrantRLSIntegration proves the enforcement boundary with a real
// NOINHERIT/NOBYPASSRLS application role. It uses one physical connection so a
// failed cleanup of app.tenant_id would deterministically leak grants into the
// next simulated request.
func TestTenantGrantRLSIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("tenant-grant PostgreSQL integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	adminPool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open tenant-grant integration database: %v", err)
	}
	t.Cleanup(adminPool.Close)
	if err := Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var userID uuid.UUID
	if err := adminPool.QueryRow(ctx, `select id from app_users where sub = 'usr-001'`).Scan(&userID); err != nil {
		t.Fatalf("read shared global fixture user: %v", err)
	}
	fixtureSuffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	grantCodes := tenantGrantCodes{
		role:       "tenant_rls_role_" + fixtureSuffix,
		permission: "tenant.rls.permission." + fixtureSuffix,
		module:     "tenant-rls-module-" + fixtureSuffix,
	}
	if _, err := adminPool.Exec(ctx, `insert into app_roles (code, label) values ($1, 'Tenant RLS integration role')`, grantCodes.role); err != nil {
		t.Fatalf("seed tenant-grant role catalog code: %v", err)
	}
	if _, err := adminPool.Exec(ctx, `insert into app_permissions (code, label) values ($1, 'Tenant RLS integration permission')`, grantCodes.permission); err != nil {
		t.Fatalf("seed tenant-grant permission catalog code: %v", err)
	}
	if _, err := adminPool.Exec(ctx, `insert into app_modules (code, active) values ($1, true)`, grantCodes.module); err != nil {
		t.Fatalf("seed tenant-grant module catalog code: %v", err)
	}
	grantTenantGrantRLSApplicationRole(t, ctx, adminPool, it.roleName)

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationConfig.MinConns = 0
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open restricted application pool: %v", err)
	}
	t.Cleanup(applicationPool.Close)
	if err := applicationPool.Ping(ctx); err != nil {
		t.Fatalf("ping restricted application pool: %v", err)
	}
	sessionPool := NewSessionPool(applicationPool)

	// The simulated application session explicitly carries a platform-admin
	// flag. Grant isolation must still be tenant-only.
	tenantAContext, releaseTenantA := acquireTenantGrantRequest(t, ctx, applicationPool, "tenant-egueducation", "inst-001")
	versionBefore := tenantGrantAuthorizationVersion(t, tenantAContext, sessionPool, "tenant-egueducation", userID)
	insertTenantGrantSet(t, tenantAContext, sessionPool, "tenant-egueducation", userID, grantCodes)
	versionAfter := tenantGrantAuthorizationVersion(t, tenantAContext, sessionPool, "tenant-egueducation", userID)
	if versionAfter < versionBefore+3 {
		releaseTenantA()
		t.Fatalf("tenant-bound grant changes did not invalidate authorization version: before=%d after=%d", versionBefore, versionAfter)
	}
	releaseTenantA()

	tenantBContext, releaseTenantB := acquireTenantGrantRequest(t, ctx, applicationPool, "tenant-balotesti", "inst-balotesti")
	assertTenantGrantSetCount(t, tenantBContext, sessionPool, "tenant-egueducation", userID, grantCodes, 0)
	assertTenantGrantCrossTenantWritesDenied(t, tenantBContext, sessionPool, "tenant-egueducation", userID, grantCodes)
	insertTenantGrantSet(t, tenantBContext, sessionPool, "tenant-balotesti", userID, grantCodes)
	assertTenantGrantSetCount(t, tenantBContext, sessionPool, "tenant-balotesti", userID, grantCodes, 1)
	releaseTenantB()

	// The same global profile can legitimately have grants in both tenants,
	// but each request sees only its own tenant's grants.
	tenantAContext, releaseTenantA = acquireTenantGrantRequest(t, ctx, applicationPool, "tenant-egueducation", "inst-001")
	assertTenantGrantSetCount(t, tenantAContext, sessionPool, "tenant-egueducation", userID, grantCodes, 1)
	assertTenantGrantSetCount(t, tenantAContext, sessionPool, "tenant-balotesti", userID, grantCodes, 0)
	releaseTenantA()

	assertTenantGrantPoolContextCleared(t, ctx, applicationPool, userID, grantCodes)
}

type tenantGrantCodes struct {
	role       string
	permission string
	module     string
}

type tenantGrantRLSIntegrationDatabase struct {
	databaseConfig *pgxpool.Config
	roleName       string
	rolePassword   string
}

func newTenantGrantRLSIntegrationDatabase(t *testing.T, ctx context.Context, databaseURL string) tenantGrantRLSIntegrationDatabase {
	t.Helper()
	baseConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse tenant-grant integration database URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, baseConfig)
	if err != nil {
		t.Fatalf("connect tenant-grant integration database admin: %v", err)
	}

	fixtureSuffix := strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	databaseName := "tenant_grant_rls_" + fixtureSuffix
	roleName := "tenant_grant_app_" + fixtureSuffix
	rolePassword := uuid.NewString()
	if _, err := admin.Exec(ctx, "create database "+tenantGrantQuoteIdentifier(databaseName)+" template template0"); err != nil {
		admin.Close(ctx)
		t.Fatalf("create tenant-grant integration database: %v", err)
	}
	if _, err := admin.Exec(ctx, "create role "+tenantGrantQuoteIdentifier(roleName)+" login nosuperuser nocreatedb nocreaterole noinherit nobypassrls password "+tenantGrantQuoteLiteral(rolePassword)); err != nil {
		_, _ = admin.Exec(ctx, "drop database if exists "+tenantGrantQuoteIdentifier(databaseName)+" with (force)")
		admin.Close(ctx)
		t.Fatalf("create non-bypass tenant-grant application role: %v", err)
	}
	admin.Close(ctx)

	targetConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse target tenant-grant database URL: %v", err)
	}
	targetConfig.ConnConfig.Database = databaseName
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		cleanup, cleanupErr := pgx.ConnectConfig(cleanupContext, baseConfig)
		if cleanupErr != nil {
			t.Errorf("connect to remove tenant-grant integration database: %v", cleanupErr)
			return
		}
		defer cleanup.Close(cleanupContext)
		if _, err := cleanup.Exec(cleanupContext, "drop database if exists "+tenantGrantQuoteIdentifier(databaseName)+" with (force)"); err != nil {
			t.Errorf("drop tenant-grant integration database: %v", err)
		}
		if _, err := cleanup.Exec(cleanupContext, "drop role if exists "+tenantGrantQuoteIdentifier(roleName)); err != nil {
			t.Errorf("drop tenant-grant application role: %v", err)
		}
	})
	return tenantGrantRLSIntegrationDatabase{
		databaseConfig: targetConfig,
		roleName:       roleName,
		rolePassword:   rolePassword,
	}
}

func grantTenantGrantRLSApplicationRole(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	quotedRole := tenantGrantQuoteIdentifier(roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + quotedRole,
		"grant select on app_users to " + quotedRole,
		"grant select, insert, update, delete on app_user_roles, app_user_permissions, app_user_modules, app_tenant_authorization_versions to " + quotedRole,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant restricted tenant-grant application role access: %v", err)
		}
	}
}

func acquireTenantGrantRequest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantCode, institutionID string) (context.Context, func()) {
	t.Helper()
	requestContext, release, err := AcquireRequestConn(ctx, pool, SessionConfig{
		TenantID: tenantCode, InstitutionID: institutionID, InstitutionName: tenantCode,
		ActorSubject: "tenant-grant-rls-integration", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatalf("acquire tenant-bound application connection: %v", err)
	}
	return requestContext, release
}

func insertTenantGrantSet(t *testing.T, ctx context.Context, pool *SessionPool, tenantCode string, userID uuid.UUID, codes tenantGrantCodes) {
	t.Helper()
	for _, grant := range []struct {
		table  string
		column string
		code   string
	}{
		{"app_user_roles", "role_code", codes.role},
		{"app_user_permissions", "permission_code", codes.permission},
		{"app_user_modules", "module_code", codes.module},
	} {
		statement := fmt.Sprintf("insert into %s (tenant_code, user_id, %s) values ($1, $2, $3)", grant.table, grant.column)
		if _, err := pool.Exec(ctx, statement, tenantCode, userID, grant.code); err != nil {
			t.Fatalf("insert tenant-bound %s grant: %v", grant.table, err)
		}
	}
}

func tenantGrantAuthorizationVersion(t *testing.T, ctx context.Context, pool *SessionPool, tenantCode string, userID uuid.UUID) int64 {
	t.Helper()
	var version int64
	if err := pool.QueryRow(ctx, `
	select coalesce((
		select version
		from app_tenant_authorization_versions
		where tenant_code = $1 and user_id = $2
	), 0)
	`, tenantCode, userID).Scan(&version); err != nil {
		t.Fatalf("read tenant authorization version: %v", err)
	}
	return version
}

func assertTenantGrantSetCount(t *testing.T, ctx context.Context, pool *SessionPool, tenantCode string, userID uuid.UUID, codes tenantGrantCodes, expected int) {
	t.Helper()
	for _, grant := range []struct {
		table  string
		column string
		code   string
	}{
		{"app_user_roles", "role_code", codes.role},
		{"app_user_permissions", "permission_code", codes.permission},
		{"app_user_modules", "module_code", codes.module},
	} {
		var count int
		statement := fmt.Sprintf("select count(*) from %s where tenant_code = $1 and user_id = $2 and %s = $3", grant.table, grant.column)
		if err := pool.QueryRow(ctx, statement, tenantCode, userID, grant.code).Scan(&count); err != nil {
			t.Fatalf("read %s grant visibility: %v", grant.table, err)
		}
		if count != expected {
			t.Fatalf("%s grants visible for %s = %d, want %d", grant.table, tenantCode, count, expected)
		}
	}
}

func assertTenantGrantCrossTenantWritesDenied(t *testing.T, ctx context.Context, pool *SessionPool, otherTenant string, userID uuid.UUID, codes tenantGrantCodes) {
	t.Helper()
	for _, grant := range []struct {
		table  string
		column string
		code   string
	}{
		{"app_user_roles", "role_code", codes.role},
		{"app_user_permissions", "permission_code", codes.permission},
		{"app_user_modules", "module_code", codes.module},
	} {
		insertStatement := fmt.Sprintf("insert into %s (tenant_code, user_id, %s) values ($1, $2, $3)", grant.table, grant.column)
		if _, err := pool.Exec(ctx, insertStatement, otherTenant, userID, grant.code); err == nil {
			t.Fatalf("cross-tenant insert into %s unexpectedly succeeded", grant.table)
		}

		updateStatement := fmt.Sprintf("update %s set %s = %s where tenant_code = $1 and user_id = $2 and %s = $3", grant.table, grant.column, grant.column, grant.column)
		updateResult, err := pool.Exec(ctx, updateStatement, otherTenant, userID, grant.code)
		if err != nil {
			t.Fatalf("cross-tenant update on %s returned an unexpected error: %v", grant.table, err)
		}
		if updateResult.RowsAffected() != 0 {
			t.Fatalf("cross-tenant update on %s changed %d rows", grant.table, updateResult.RowsAffected())
		}

		deleteStatement := fmt.Sprintf("delete from %s where tenant_code = $1 and user_id = $2 and %s = $3", grant.table, grant.column)
		deleteResult, err := pool.Exec(ctx, deleteStatement, otherTenant, userID, grant.code)
		if err != nil {
			t.Fatalf("cross-tenant delete on %s returned an unexpected error: %v", grant.table, err)
		}
		if deleteResult.RowsAffected() != 0 {
			t.Fatalf("cross-tenant delete on %s changed %d rows", grant.table, deleteResult.RowsAffected())
		}
	}
}

func assertTenantGrantPoolContextCleared(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, codes tenantGrantCodes) {
	t.Helper()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("reacquire restricted application connection: %v", err)
	}
	defer conn.Release()

	var tenantCode, superAdmin string
	if err := conn.QueryRow(ctx, `
		select coalesce(current_setting('app.tenant_id', true), ''), coalesce(current_setting('app.is_super_admin', true), '')
	`).Scan(&tenantCode, &superAdmin); err != nil {
		t.Fatalf("inspect released tenant-grant connection: %v", err)
	}
	if tenantCode != "" || superAdmin != "" {
		t.Fatalf("pooled application connection retained tenant context: tenant=%q superadmin=%q", tenantCode, superAdmin)
	}

	for _, grant := range []struct {
		table  string
		column string
		code   string
	}{
		{"app_user_roles", "role_code", codes.role},
		{"app_user_permissions", "permission_code", codes.permission},
		{"app_user_modules", "module_code", codes.module},
	} {
		var count int
		statement := fmt.Sprintf("select count(*) from %s where user_id = $1 and %s = $2", grant.table, grant.column)
		if err := conn.QueryRow(ctx, statement, userID, grant.code).Scan(&count); err != nil {
			t.Fatalf("read unscoped %s grants: %v", grant.table, err)
		}
		if count != 0 {
			t.Fatalf("unscoped pooled connection exposed %d %s grant rows", count, grant.table)
		}
	}
}

func tenantGrantQuoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func tenantGrantQuoteLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
