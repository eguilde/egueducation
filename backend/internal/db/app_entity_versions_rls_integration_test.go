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

	for _, scope := range []appEntityVersionsRLSScope{fixture.tenantA, fixture.tenantAOtherInstitution, fixture.tenantB} {
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
	tenantA                 appEntityVersionsRLSScope
	tenantAOtherInstitution appEntityVersionsRLSScope
	tenantB                 appEntityVersionsRLSScope
	legacy                  uuid.UUID
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
		tenantA:                 appEntityVersionsRLSScope{tenantCode: "tenant-egueducation", institutionID: "inst-001", versionID: uuid.New()},
		tenantAOtherInstitution: appEntityVersionsRLSScope{tenantCode: "tenant-egueducation", institutionID: "inst-002", versionID: uuid.New()},
		tenantB:                 appEntityVersionsRLSScope{tenantCode: "tenant-balotesti", institutionID: "inst-balotesti", versionID: uuid.New()},
		legacy:                  uuid.New(),
	}
	sharedEntityID := uuid.New()
	for _, scope := range []appEntityVersionsRLSScope{fixture.tenantA, fixture.tenantAOtherInstitution, fixture.tenantB} {
		if _, err := tx.Exec(ctx, `
			insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
			values ($1, 'app_entity_versions_rls_fixture', $2, 1, 'insert', $3, $4, jsonb_build_object('private_scope', $3), 'fixture')
		`, scope.versionID, sharedEntityID, scope.tenantCode, scope.institutionID); err != nil {
			t.Fatalf("seed %s scoped history row: %v", scope.tenantCode, err)
		}
	}
	if _, err := tx.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_fixture', $2, 1, 'insert', '', '', '{"private_scope":"legacy"}'::jsonb, 'legacy-fixture')
	`, fixture.legacy, sharedEntityID); err != nil {
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
	hiddenIDs := []uuid.UUID{fixture.legacy}
	for _, candidate := range []appEntityVersionsRLSScope{fixture.tenantA, fixture.tenantAOtherInstitution, fixture.tenantB} {
		if candidate.versionID != scope.versionID {
			hiddenIDs = append(hiddenIDs, candidate.versionID)
		}
	}
	for _, hiddenID := range hiddenIDs {
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

	if _, err := pool.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_write_fixture', $2, 1, 'insert', $3, $4, '{}'::jsonb, 'cross-institution-attempt')
	`, uuid.New(), uuid.New(), scope.tenantCode, scope.institutionID+"-other"); err == nil {
		t.Fatalf("cross-institution history insert from %s/%s unexpectedly succeeded", scope.tenantCode, scope.institutionID)
	} else {
		var databaseError *pgconn.PgError
		if !errors.As(err, &databaseError) || databaseError.Code != "42501" {
			t.Fatalf("cross-institution history insert error=%v, want PostgreSQL RLS 42501", err)
		}
	}

	result, err := pool.Exec(ctx, `update app_entity_versions set changed_by='tampered' where id=$1`, scope.versionID)
	if err != nil {
		t.Fatalf("append-only history update returned unexpected error: %v", err)
	}
	if result.RowsAffected() != 0 {
		t.Fatalf("append-only history update affected %d rows", result.RowsAffected())
	}
	result, err = pool.Exec(ctx, `delete from app_entity_versions where id=$1`, scope.versionID)
	if err != nil {
		t.Fatalf("append-only history delete returned unexpected error: %v", err)
	}
	if result.RowsAffected() != 0 {
		t.Fatalf("append-only history delete affected %d rows", result.RowsAffected())
	}

	if _, err := pool.Exec(ctx, `
		insert into app_entity_versions (id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by)
		values ($1, 'app_entity_versions_rls_write_fixture', $2, 1, 'insert', $3, $4, '{}'::jsonb, 'same-tenant')
	`, uuid.New(), uuid.New(), scope.tenantCode, scope.institutionID); err != nil {
		t.Fatalf("same-scope history insert failed: %v", err)
	}
}

// TestAppEntityVersionsScopedIdentityUpgradeIntegration recreates the schema
// shape deployed after 0132 and proves that 0133 is applied from the migration
// ledger, accepts identical version tuples in separate scopes, and serializes
// concurrent trigger writers without bypassing RLS in the function definition.
func TestAppEntityVersionsScopedIdentityUpgradeIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("app entity versions upgrade integration test requires EGUEDUCATION_TEST_DATABASE_URL or TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	it := newTenantGrantRLSIntegrationDatabase(t, ctx, databaseURL)
	pool, err := pgxpool.NewWithConfig(ctx, it.databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open app entity versions upgrade database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply current migrations before upgrade simulation: %v", err)
	}

	entityID := uuid.New()
	if _, err := pool.Exec(ctx, `
		alter table public.app_entity_versions drop constraint if exists app_entity_versions_scope_entity_version_key;
		alter table public.app_entity_versions add constraint app_entity_versions_entity_table_entity_id_version_no_key
			unique (entity_table, entity_id, version_no);
		delete from public.schema_migrations where version = '0133_app_entity_versions_scoped_identity.sql';
		insert into public.app_entity_versions (
			id, entity_table, entity_id, version_no, change_type, tenant_code, institution_id, snapshot, changed_by
		) values ($1, 'app_entity_versions_upgrade_fixture', $2, 1, 'insert', '', '', '{"scope":"legacy"}'::jsonb, 'legacy');
	`, uuid.New(), entityID); err != nil {
		t.Fatalf("restore post-0132 schema shape: %v", err)
	}

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply 0133 upgrade: %v", err)
	}
	var ledgerCount int
	if err := pool.QueryRow(ctx, `select count(*) from schema_migrations where version = '0133_app_entity_versions_scoped_identity.sql'`).Scan(&ledgerCount); err != nil {
		t.Fatalf("read 0133 ledger state: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("0133 migration ledger count = %d, want 1", ledgerCount)
	}
	var constraintDefinition string
	if err := pool.QueryRow(ctx, `
		select pg_get_constraintdef(oid)
		from pg_constraint
		where conrelid = 'public.app_entity_versions'::regclass
			and conname = 'app_entity_versions_scope_entity_version_key'
	`).Scan(&constraintDefinition); err != nil {
		t.Fatalf("read scoped version constraint: %v", err)
	}
	for _, column := range []string{"tenant_code", "institution_id", "entity_table", "entity_id", "version_no"} {
		if !strings.Contains(strings.ToLower(constraintDefinition), column) {
			t.Fatalf("scoped version constraint %q is missing %s", constraintDefinition, column)
		}
	}

	if _, err := pool.Exec(ctx, `
		create table public.app_entity_versions_upgrade_fixture (
			id uuid not null,
			tenant_code text not null,
			institution_id text not null,
			payload text not null default '',
			primary key (tenant_code, institution_id, id)
		);
		create trigger trg_app_entity_versions_upgrade_fixture_versions
			after insert or update or delete on public.app_entity_versions_upgrade_fixture
			for each row execute function public.record_entity_version();
	`); err != nil {
		t.Fatalf("create version trigger upgrade fixture: %v", err)
	}
	for _, scope := range []appEntityVersionsRLSScope{
		{tenantCode: "tenant-upgrade-a", institutionID: "institution-a"},
		{tenantCode: "tenant-upgrade-a", institutionID: "institution-b"},
		{tenantCode: "tenant-upgrade-b", institutionID: "institution-a"},
	} {
		if _, err := pool.Exec(ctx, `
			insert into public.app_entity_versions_upgrade_fixture (id, tenant_code, institution_id, payload)
			values ($1, $2, $3, 'inserted')
		`, entityID, scope.tenantCode, scope.institutionID); err != nil {
			t.Fatalf("insert scoped trigger fixture %s/%s: %v", scope.tenantCode, scope.institutionID, err)
		}
	}

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	for _, payload := range []string{"updated-one", "updated-two"} {
		payload := payload
		go func() {
			<-start
			_, updateErr := pool.Exec(ctx, `
				update public.app_entity_versions_upgrade_fixture
				set payload = $1
				where id = $2 and tenant_code = 'tenant-upgrade-a' and institution_id = 'institution-a'
			`, payload, entityID)
			errorsChannel <- updateErr
		}()
	}
	close(start)
	for range 2 {
		if err := <-errorsChannel; err != nil {
			t.Fatalf("concurrent scoped version update: %v", err)
		}
	}

	var count, distinctVersions, minVersion, maxVersion int
	if err := pool.QueryRow(ctx, `
		select count(*), count(distinct version_no), min(version_no), max(version_no)
		from public.app_entity_versions
		where entity_table = 'app_entity_versions_upgrade_fixture'
			and entity_id = $1
			and tenant_code = 'tenant-upgrade-a'
			and institution_id = 'institution-a'
	`, entityID).Scan(&count, &distinctVersions, &minVersion, &maxVersion); err != nil {
		t.Fatalf("read concurrent scoped version history: %v", err)
	}
	if count != 3 || distinctVersions != 3 || minVersion != 1 || maxVersion != 3 {
		t.Fatalf("concurrent scoped versions count/distinct/range = %d/%d/%d-%d, want 3/3/1-3", count, distinctVersions, minVersion, maxVersion)
	}
}
