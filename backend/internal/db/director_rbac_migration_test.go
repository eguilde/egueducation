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

// legacyDirectorSchoolPermissions documents the grants published by 0128.
// 0130 subsequently removes the generic portfolio-content write capability,
// leaving the reviewed director catalog below as the effective state.
var legacyDirectorSchoolPermissions = []string{
	"education.governance.read",
	"education.governance.manage",
	"education.portfolios.read",
	"education.portfolios.manage",
	"education.personnel.read",
	"education.personnel.manage",
	"education.mobility.read",
	"education.mobility.manage",
	"education.gradatii.read",
	"education.gradatii.manage",
	"education.decisions.read",
	"education.decisions.manage",
	"education.managerial.read",
	"education.managerial.manage",
	"education.regulations.read",
	"education.regulations.manage",
	"education.evaluations.read",
	"education.evaluations.manage",
	"education.declarations.read",
	"education.declarations.manage",
}

var approvedDirectorSchoolPermissions = []string{
	"education.governance.read",
	"education.governance.manage",
	"education.portfolios.read",
	"education.personnel.read",
	"education.personnel.manage",
	"education.mobility.read",
	"education.mobility.manage",
	"education.gradatii.read",
	"education.gradatii.manage",
	"education.decisions.read",
	"education.decisions.manage",
	"education.managerial.read",
	"education.managerial.manage",
	"education.regulations.read",
	"education.regulations.manage",
	"education.evaluations.read",
	"education.evaluations.manage",
	"education.declarations.read",
	"education.declarations.manage",
}

func TestDirectorSchoolCatalogMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("migrations/0128_director_school_catalog.sql")
	if err != nil {
		t.Fatalf("read director School catalog migration: %v", err)
	}
	text := strings.ToLower(string(body))
	for _, permission := range legacyDirectorSchoolPermissions {
		if !strings.Contains(text, "('director', '"+permission+"')") {
			t.Errorf("director School catalog migration is missing explicit grant %q", permission)
		}
	}
	for _, forbidden := range []string{"like 'education.%'", "app_platform_roles", "'admin'"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("director School catalog migration contains forbidden broadening token %q", forbidden)
		}
	}
}

// TestDirectorSchoolCatalogUpgradeIntegration proves that an upgrade which
// replays the historical director catalog is reconciled by 0130: directors
// retain reviewed permissions but do not receive generic portfolio-content
// write access or later unreviewed education permissions.
func TestDirectorSchoolCatalogUpgradeIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("director RBAC PostgreSQL integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	pool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open director RBAC integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply current migrations: %v", err)
	}

	var migration0127Count int
	if err := pool.QueryRow(ctx, `select count(*) from schema_migrations where version = '0127_director_rbac_least_privilege.sql'`).Scan(&migration0127Count); err != nil {
		t.Fatalf("read 0127 migration state: %v", err)
	}
	if migration0127Count != 1 {
		t.Fatalf("0127 migration count = %d, want 1", migration0127Count)
	}

	if _, err := pool.Exec(ctx, `delete from app_role_permissions where role_code = 'director' and permission_code = any($1)`, legacyDirectorSchoolPermissions); err != nil {
		t.Fatalf("simulate pre-0128 director grant state: %v", err)
	}
	unreviewedPermission := "education.unreviewed." + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := pool.Exec(ctx, `insert into app_permissions(code, label) values ($1, 'Unreviewed director permission')`, unreviewedPermission); err != nil {
		t.Fatalf("seed unreviewed education permission: %v", err)
	}
	if _, err := pool.Exec(ctx, `delete from schema_migrations where version = any($1)`, []string{"0128_director_school_catalog.sql", "0130_school_content_rbac_hardening.sql"}); err != nil {
		t.Fatalf("simulate pre-0128/0130 upgrade state: %v", err)
	}

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("upgrade database through 0130: %v", err)
	}
	var approvedCount int
	if err := pool.QueryRow(ctx, `select count(*) from app_role_permissions where role_code = 'director' and permission_code = any($1)`, approvedDirectorSchoolPermissions).Scan(&approvedCount); err != nil {
		t.Fatalf("read approved director grants: %v", err)
	}
	if approvedCount != len(approvedDirectorSchoolPermissions) {
		t.Fatalf("approved director grant count = %d, want %d", approvedCount, len(approvedDirectorSchoolPermissions))
	}
	var genericPortfolioManageCount int
	if err := pool.QueryRow(ctx, `select count(*) from app_role_permissions where role_code = 'director' and permission_code = 'education.portfolios.manage'`).Scan(&genericPortfolioManageCount); err != nil {
		t.Fatalf("read generic director portfolio manage grant: %v", err)
	}
	if genericPortfolioManageCount != 0 {
		t.Fatalf("director retained %d generic portfolio-content grants, want 0", genericPortfolioManageCount)
	}
	var unreviewedCount int
	if err := pool.QueryRow(ctx, `select count(*) from app_role_permissions where role_code = 'director' and permission_code = $1`, unreviewedPermission).Scan(&unreviewedCount); err != nil {
		t.Fatalf("read unreviewed director grant: %v", err)
	}
	if unreviewedCount != 0 {
		t.Fatalf("director inherited %d unreviewed education grants, want 0", unreviewedCount)
	}
}
