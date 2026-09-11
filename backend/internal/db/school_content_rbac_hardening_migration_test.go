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

const schoolContentRBACHardeningMigration = "migrations/0130_school_content_rbac_hardening.sql"

func TestSchoolContentRBACHardeningMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile(schoolContentRBACHardeningMigration)
	if err != nil {
		t.Fatalf("read %s: %v", schoolContentRBACHardeningMigration, err)
	}
	text := strings.ToLower(string(body))
	for _, required := range []string{
		"role_permission.role_code = 'admin'",
		"lower(permission.code) like 'education.%'",
		"role_code = 'director'",
		"permission_code = 'education.portfolios.manage'",
		"role.code in ('super_admin', 'e2e_canary')",
		"role.code = 'admin' and lower(new.code) not like 'education.%'",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("%s is missing hardening invariant %q", schoolContentRBACHardeningMigration, required)
		}
	}
	if strings.Contains(text, "role.code in ('admin', 'super_admin', 'e2e_canary')") {
		t.Fatal("future permissions must not automatically give admin pedagogical access")
	}
}

// TestSchoolContentRBACHardeningIntegration verifies both the reconciliation
// of legacy grants and the replacement trigger.  It intentionally performs
// no HTTP authorization shortcut: role-permission rows are the source from
// which OIDC access-token claims are derived.
func TestSchoolContentRBACHardeningIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("School content RBAC PostgreSQL integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	pool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open School content RBAC integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply current migrations: %v", err)
	}

	assertRolePermissionAbsent(t, ctx, pool, "admin", "education.portfolios.read")
	assertRolePermissionAbsent(t, ctx, pool, "admin", "education.portfolios.manage")
	assertRolePermissionAbsent(t, ctx, pool, "director", "education.portfolios.manage")
	for _, permission := range []string{
		"education.portfolios.school.read",
		"education.portfolios.verify",
		"education.portfolios.request_corrections",
	} {
		assertRolePermissionPresent(t, ctx, pool, "director", permission)
	}
	// Administration remains intact: no School permission is needed to manage
	// tenant users and roles.
	for _, permission := range []string{"admin.users.manage", "admin.roles.manage"} {
		assertRolePermissionPresent(t, ctx, pool, "admin", permission)
	}

	futureEducation := "education.future_content_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, `insert into app_permissions(code, label) values ($1, 'Future School content')`, futureEducation); err != nil {
		t.Fatalf("insert future education permission: %v", err)
	}
	assertRolePermissionAbsent(t, ctx, pool, "admin", futureEducation)
	assertRolePermissionAbsent(t, ctx, pool, "director", futureEducation)
	assertRolePermissionPresent(t, ctx, pool, "super_admin", futureEducation)
	assertRolePermissionPresent(t, ctx, pool, "e2e_canary", futureEducation)
	caseVariantEducation := "Education.future_content_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, `insert into app_permissions(code, label) values ($1, 'Case-variant future School content')`, caseVariantEducation); err != nil {
		t.Fatalf("insert case-variant future education permission: %v", err)
	}
	assertRolePermissionAbsent(t, ctx, pool, "admin", caseVariantEducation)
	assertRolePermissionPresent(t, ctx, pool, "super_admin", caseVariantEducation)

	futureAdministration := "admin.future_capability_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, `insert into app_permissions(code, label) values ($1, 'Future tenant administration capability')`, futureAdministration); err != nil {
		t.Fatalf("insert future administration permission: %v", err)
	}
	assertRolePermissionPresent(t, ctx, pool, "admin", futureAdministration)
	assertRolePermissionPresent(t, ctx, pool, "super_admin", futureAdministration)
}

func assertRolePermissionAbsent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleCode, permissionCode string) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from app_role_permissions where role_code = $1 and permission_code = $2`, roleCode, permissionCode).Scan(&count); err != nil {
		t.Fatalf("read %s/%s role-permission: %v", roleCode, permissionCode, err)
	}
	if count != 0 {
		t.Fatalf("%s unexpectedly has %s", roleCode, permissionCode)
	}
}

func assertRolePermissionPresent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleCode, permissionCode string) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from app_role_permissions where role_code = $1 and permission_code = $2`, roleCode, permissionCode).Scan(&count); err != nil {
		t.Fatalf("read %s/%s role-permission: %v", roleCode, permissionCode, err)
	}
	if count != 1 {
		t.Fatalf("%s permission count for %s = %d, want 1", roleCode, permissionCode, count)
	}
}
