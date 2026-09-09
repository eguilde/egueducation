package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPost0060EducationRLSIntegration proves that the three institution
// tables added after the original RLS baseline cannot leak through a
// NOBYPASSRLS application role.  It intentionally sets IsSuperAdmin=false:
// a platform role must not turn this test into a database bypass.
func TestPost0060EducationRLSIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("post-0060 education RLS integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	adminPool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open post-0060 RLS integration admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)
	if err := Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	seedPost0060EducationRLSRows(t, ctx, adminPool)
	grantPost0060EducationRLSReadAccess(t, ctx, adminPool, it.roleName)

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open post-0060 RLS application pool: %v", err)
	}
	t.Cleanup(applicationPool.Close)
	servicePool := NewSessionPool(applicationPool)

	for _, scope := range []struct {
		tenantCode, institutionID string
	}{
		{"tenant-egueducation", "inst-001"},
		{"tenant-balotesti", "inst-balotesti"},
	} {
		requestContext, release, err := AcquireRequestConn(ctx, applicationPool, SessionConfig{
			TenantID: scope.tenantCode, InstitutionID: scope.institutionID,
			InstitutionName: scope.tenantCode, ActorSubject: "post-0060-rls-test", IsSuperAdmin: false,
		})
		if err != nil {
			t.Fatalf("bind %s request context: %v", scope.tenantCode, err)
		}
		for _, table := range []string{
			"education_portfolio_valorifications",
			"education_committees",
			"education_committee_members",
		} {
			var visible int
			if err := servicePool.QueryRow(requestContext, fmt.Sprintf("select count(*) from %s", table)).Scan(&visible); err != nil {
				release()
				t.Fatalf("read %s as %s: %v", table, scope.tenantCode, err)
			}
			if visible != 1 {
				release()
				t.Fatalf("%s visible rows for %s=%d, want exactly 1", table, scope.tenantCode, visible)
			}
		}
		release()
	}
}

func seedPost0060EducationRLSRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin post-0060 RLS fixture: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin','true',true)`); err != nil {
		t.Fatalf("enable fixture RLS bypass: %v", err)
	}
	for _, scope := range []struct {
		tenantCode, institutionID string
	}{
		{"tenant-egueducation", "inst-001"},
		{"tenant-balotesti", "inst-balotesti"},
	} {
		portfolioID := uuid.New()
		committeeID := uuid.New()
		if _, err := tx.Exec(ctx, `
			insert into education_portfolios (
				id, portfolio_code, owner_name, owner_role, school_year, status,
				section_count, last_updated_on, transfer_status, institution_id
			) values ($1, $2, 'RLS integration owner', 'Profesor', '2026-2027', 'draft', 0, current_date, 'none', $3)
		`, portfolioID, "RLS-"+scope.tenantCode+"-"+uuid.NewString(), scope.institutionID); err != nil {
			t.Fatalf("seed portfolio for %s: %v", scope.tenantCode, err)
		}
		if _, err := tx.Exec(ctx, `
			insert into education_portfolio_valorifications (
				portfolio_id, valorification_code, scope, status, started_on, institution_id
			) values ($1, $2, 'evaluare_profesionala', 'planificat', current_date, $3)
		`, portfolioID, "VAL-RLS-"+uuid.NewString(), scope.institutionID); err != nil {
			t.Fatalf("seed valorification for %s: %v", scope.tenantCode, err)
		}
		if _, err := tx.Exec(ctx, `
			insert into education_committees (
				id, committee_code, school_year, committee_type, title, status, starts_on, institution_id
			) values ($1, $2, '2026-2027', 'curriculum', 'RLS committee', 'active', current_date, $3)
		`, committeeID, "COM-RLS-"+uuid.NewString(), scope.institutionID); err != nil {
			t.Fatalf("seed committee for %s: %v", scope.tenantCode, err)
		}
		if _, err := tx.Exec(ctx, `
			insert into education_committee_members (
				committee_id, full_name, role_name, member_type, status, appointed_on, institution_id
			) values ($1, 'RLS committee member', 'Profesor', 'membru', 'active', current_date, $2)
		`, committeeID, scope.institutionID); err != nil {
			t.Fatalf("seed committee member for %s: %v", scope.tenantCode, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit post-0060 RLS fixture: %v", err)
	}
}

func grantPost0060EducationRLSReadAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	quotedRole := tenantGrantQuoteIdentifier(roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + quotedRole,
		"grant select on education_portfolio_valorifications to " + quotedRole,
		"grant select on education_committees to " + quotedRole,
		"grant select on education_committee_members to " + quotedRole,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant application read access: %v", err)
		}
	}
}
