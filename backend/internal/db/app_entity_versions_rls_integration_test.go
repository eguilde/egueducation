package db

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAppEntityVersionsRLSIntegration proves that the shared version store is
// both tenant- and institution-scoped for a NOBYPASSRLS application role. It
// includes an unscoped legacy row, which must fail closed rather than leak.
func TestAppEntityVersionsRLSIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("app entity versions PostgreSQL integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	adminPool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open app entity versions integration database: %v", err)
	}
	t.Cleanup(adminPool.Close)
	if err := Migrate(ctx, adminPool); err != nil {
		t.Fatalf("apply app entity versions migrations: %v", err)
	}
	fixture := seedAppEntityVersionsRLSRows(t, ctx, adminPool)
	grantAppEntityVersionsRLSAccess(t, ctx, adminPool, it.roleName)

	applicationConfig := it.databaseConfig.Copy()
	applicationConfig.ConnConfig.User = it.roleName
	applicationConfig.ConnConfig.Password = it.rolePassword
	applicationConfig.MaxConns = 1
	applicationPool, err := pgxpool.NewWithConfig(ctx, applicationConfig)
	if err != nil {
		t.Fatalf("open NOBYPASSRLS app entity versions pool: %v", err)
	}
	t.Cleanup(applicationPool.Close)
	servicePool := NewSessionPool(applicationPool)

	for _, scope := range []appEntityVersionsRLSScope{fixture.tenantA, fixture.tenantB} {
		requestContext, release, err := AcquireRequestConn(ctx, applicationPool, SessionConfig{
			TenantID: scope.tenantCode, InstitutionID: scope.institutionID, InstitutionName: scope.tenantCode,
			ActorSubject: "app-entity-versions-rls-integration", IsSuperAdmin: false,
		})
		if err != nil {
			t.Fatalf("bind NOBYPASSRLS %s context: %v", scope.tenantCode, err)
		}

		assertAppEntityVersionsScope(t, requestContext, servicePool, scope, fixture)
		release()
	}
}

type appEntityVersionsRLSScope struct {
	tenantCode    string
	institutionID string
	versionID     uuid.UUID
}

type appEntityVersionsRLSFixture struct {
	tenantA appEntityVersionsRLSScope
	tenantB appEntityVersionsRLSScope
	legacy  uuid.UUID
}

func seedAppEntityVersionsRLSRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool) appEntityVersionsRLSFixture {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin app entity versions fixture: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `select set_config('app.is_super_admin', 'true', true)`); err != nil {
		t.Fatalf("enable app entity versions fixture bypass: %v", err)
	}
	fixture := appEntityVersionsRLSFixture{
		tenantA: appEntityVersionsRLSScope{tenantCode: "tenant-egueducation", institutionID: "inst-001", versionID: uuid.New()},
		tenantB: appEntityVersionsRLSScope{tenantCode: "tenant-balotesti", institutionID: "inst-balotesti", versionID: uuid.New()},
		legacy:  uuid.New(),
	}
	for _, scope := range []appEntityVersionsRLSScope{fixture.tenantA, fixture.tenantB} {
		if _, err := tx.Exec(ctx, `
			insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
			values ($1, 'app_entity_versions_rls_fixture', $2, 1, 'insert', $3, $4, jsonb_build_object('private_scope', $3), 'fixture')
		`, scope.versionID, uuid.New(), scope.tenantCode, scope.institutionID); err != nil {
			t.Fatalf("seed %s scoped history row: %v", scope.tenantCode, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_fixture', $2, 1, 'insert', '', '', '{"private_scope":"legacy"}'::jsonb, 'legacy-fixture')
	`, fixture.legacy, uuid.New()); err != nil {
		t.Fatalf("seed unscoped legacy history row: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit app entity versions fixture: %v", err)
	}
	return fixture
}

func grantAppEntityVersionsRLSAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool, roleName string) {
	t.Helper()
	role := tenantGrantQuoteIdentifier(roleName)
	for _, statement := range []string{
		"grant usage on schema public to " + role,
		"grant select, insert, update, delete on app_entity_versions to " + role,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatalf("grant app entity versions application access: %v", err)
		}
	}
}

func assertAppEntityVersionsScope(t *testing.T, ctx context.Context, pool *SessionPool, scope appEntityVersionsRLSScope, fixture appEntityVersionsRLSFixture) {
	t.Helper()
	var visible int
	if err := pool.QueryRow(ctx, `select count(*) from app_entity_versions where entity_table='app_entity_versions_rls_fixture'`).Scan(&visible); err != nil {
		t.Fatalf("read %s history visibility: %v", scope.tenantCode, err)
	}
	if visible != 1 {
		t.Fatalf("history rows visible in %s = %d, want 1 scoped row", scope.tenantCode, visible)
	}

	foreign := fixture.tenantA
	if scope.tenantCode == foreign.tenantCode {
		foreign = fixture.tenantB
	}
	for _, hiddenID := range []uuid.UUID{foreign.versionID, fixture.legacy} {
		var count int
		if err := pool.QueryRow(ctx, `select count(*) from app_entity_versions where id=$1`, hiddenID).Scan(&count); err != nil {
			t.Fatalf("read hidden history %s: %v", hiddenID, err)
		}
		if count != 0 {
			t.Fatalf("%s exposed history row %s", scope.tenantCode, hiddenID)
		}
	}

	if _, err := pool.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_write_fixture', $2, 1, 'insert', $3, $4, '{}'::jsonb, 'cross-tenant-attempt')
	`, uuid.New(), uuid.New(), foreign.tenantCode, foreign.institutionID); err == nil {
		t.Fatalf("cross-tenant history insert from %s unexpectedly succeeded", scope.tenantCode)
	} else {
		var databaseError *pgconn.PgError
		if !errors.As(err, &databaseError) || databaseError.Code != "42501" {
			t.Fatalf("cross-tenant history insert error=%v, want PostgreSQL RLS 42501", err)
		}
	}

	result, err := pool.Exec(ctx, `update app_entity_versions set changed_by='tampered' where id=$1`, scope.versionID)
	if err != nil {
		t.Fatalf("append-only history update returned unexpected error: %v", err)
	}
	if result.RowsAffected() != 0 {
		t.Fatalf("append-only history update affected %d rows", result.RowsAffected())
	}

	if _, err := pool.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_write_fixture', $2, 1, 'insert', $3, $4, '{}'::jsonb, 'same-tenant')
	`, uuid.New(), uuid.New(), scope.tenantCode, scope.institutionID); err != nil {
		t.Fatalf("same-scope history insert failed: %v", err)
	}
}
