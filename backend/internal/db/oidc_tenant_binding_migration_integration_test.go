package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestOIDCTenantBindingUpgradeIntegration runs migration 0084 against a
// pre-0084 schema with both unbound state and an ambiguous tenant/id pair.
// It complements the clean-database OIDC integration test.
func TestOIDCTenantBindingUpgradeIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("EGUEDUCATION_TEST_DATABASE_URL/TEST_DATABASE_URL is not configured")
	}

	ctx := context.Background()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	t.Cleanup(pool.Close)

	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire integration connection: %v", err)
	}
	defer connection.Release()

	schemaName := "oidc_upgrade_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schemaName}.Sanitize()
	if _, err := connection.Exec(ctx, "create schema "+quotedSchema); err != nil {
		t.Fatalf("create isolated upgrade schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "drop schema "+quotedSchema+" cascade")
	})
	if _, err := connection.Exec(ctx, "set search_path to "+quotedSchema+", public"); err != nil {
		t.Fatalf("bind isolated upgrade schema: %v", err)
	}

	if _, err := connection.Exec(ctx, `
		create table app_tenants (code text primary key);
		insert into app_tenants(code) values ('tenant-a'), ('tenant-b');
		create table oidc_authn_sessions (
			tenant_id uuid not null, tenant_code text references app_tenants(code),
			id text not null, data jsonb not null, expires_at timestamptz not null,
			primary key (tenant_id, id)
		);
		create table oidc_grant_sessions (
			tenant_id uuid not null, tenant_code text references app_tenants(code),
			id text not null, data jsonb not null, expires_at timestamptz not null,
			primary key (tenant_id, id)
		);
		insert into oidc_authn_sessions values
			('00000000-0000-0000-0000-000000000001', null, 'unbound', '{"state":"legacy"}', now() + interval '1 minute'),
			('00000000-0000-0000-0000-000000000002', 'tenant-a', 'collision', '{"version":"older"}', now() + interval '2 minutes'),
			('00000000-0000-0000-0000-000000000003', 'tenant-a', 'collision', '{"version":"newest"}', now() + interval '3 minutes');
		insert into oidc_grant_sessions values
			('00000000-0000-0000-0000-000000000011', null, 'unbound', '{"state":"legacy"}', now() + interval '1 minute'),
			('00000000-0000-0000-0000-000000000012', 'tenant-b', 'collision', '{"version":"older"}', now() + interval '2 minutes'),
			('00000000-0000-0000-0000-000000000013', 'tenant-b', 'collision', '{"version":"newest"}', now() + interval '3 minutes');
	`); err != nil {
		t.Fatalf("seed legacy OIDC storage: %v", err)
	}

	migration, err := migrationFiles.ReadFile("migrations/0084_oidc_tenant_binding.sql")
	if err != nil {
		t.Fatalf("read 0084 migration: %v", err)
	}
	if _, err := connection.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply 0084 to legacy OIDC storage: %v", err)
	}

	for _, table := range []string{"oidc_authn_sessions", "oidc_grant_sessions"} {
		var unboundCount, collisionCount int
		var retainedVersion, primaryKeyDefinition string
		if err := connection.QueryRow(ctx, fmt.Sprintf("select count(*) from %s where id = 'unbound'", table)).Scan(&unboundCount); err != nil {
			t.Fatalf("count unbound rows in %s: %v", table, err)
		}
		if err := connection.QueryRow(ctx, fmt.Sprintf("select count(*), max(data->>'version') from %s where id = 'collision'", table)).Scan(&collisionCount, &retainedVersion); err != nil {
			t.Fatalf("read retained collision in %s: %v", table, err)
		}
		if err := connection.QueryRow(ctx, `
			select pg_get_constraintdef(oid)
			from pg_constraint
			where conrelid = $1::regclass and contype = 'p'
		`, table).Scan(&primaryKeyDefinition); err != nil {
			t.Fatalf("read primary key for %s: %v", table, err)
		}
		if unboundCount != 0 || collisionCount != 1 || retainedVersion != "newest" {
			t.Fatalf("unsafe 0084 result for %s: unbound=%d collisions=%d retained=%q", table, unboundCount, collisionCount, retainedVersion)
		}
		if !strings.Contains(primaryKeyDefinition, "(tenant_code, id)") {
			t.Fatalf("%s primary key is not tenant-aware: %s", table, primaryKeyDefinition)
		}
	}

	for _, quarantineTable := range []string{"oidc_legacy_unbound_authn_sessions", "oidc_legacy_unbound_grant_sessions"} {
		var count int
		if err := connection.QueryRow(ctx, fmt.Sprintf("select count(*) from %s", quarantineTable)).Scan(&count); err != nil {
			t.Fatalf("count quarantined rows in %s: %v", quarantineTable, err)
		}
		if count != 2 {
			t.Fatalf("expected one unbound and one duplicate row in %s, got %d", quarantineTable, count)
		}
	}
}
