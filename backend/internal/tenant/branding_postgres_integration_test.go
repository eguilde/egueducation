package tenant

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabaseResolverIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("tenant PostgreSQL integration requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	// One physical connection makes the pooled-session assertion below
	// deterministic: the resolver's transaction must not leak its RLS bypass.
	poolConfig.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	if err := appdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin seed: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin', 'true', true)`); err != nil {
		t.Fatalf("scope seed: %v", err)
	}
	_, err = tx.Exec(ctx, `insert into app_tenants(code, subdomain, institution_id, display_name, short_name, root_org_unit_code, active) values ('tenant-third', 'third', 'inst-third', 'Third Tenant', 'Third', 'unit-root', true) on conflict (code) do update set active=true`)
	if err != nil {
		t.Fatalf("seed third tenant: %v", err)
	}
	_, err = tx.Exec(ctx, `insert into app_tenant_hostnames(tenant_code, hostname, active) values ('tenant-third', 'third.example.test', true) on conflict (hostname) do update set tenant_code=excluded.tenant_code, active=true`)
	if err != nil {
		t.Fatalf("seed third host: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into app_tenant_hostnames(tenant_code, hostname, active) values ('tenant-third', 'THIRD.EXAMPLE.TEST.', true)`); err == nil {
		t.Fatal("canonicalized duplicate hostname must be rejected")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit seed: %v", err)
	}
	defer func() {
		cleanup, cleanupErr := pool.Begin(context.Background())
		if cleanupErr != nil {
			return
		}
		defer func() { _ = cleanup.Rollback(context.Background()) }()
		if _, cleanupErr = cleanup.Exec(context.Background(), `select set_config('app.is_super_admin', 'true', true)`); cleanupErr != nil {
			return
		}
		_, _ = cleanup.Exec(context.Background(), `delete from app_tenants where code='tenant-third'`)
		_ = cleanup.Commit(context.Background())
	}()

	resolver := &Resolver{pool: pool, baseHost: "example.test"}
	branding, err := resolver.Resolve(ctx, "THIRD.EXAMPLE.TEST:443")
	if err != nil || branding.TenantCode != "tenant-third" || branding.InstitutionID != "inst-third" {
		t.Fatalf("third-tenant resolution = %#v, %v", branding, err)
	}
	var inheritedBypass string
	if err := pool.QueryRow(ctx, `select coalesce(current_setting('app.is_super_admin', true), '')`).Scan(&inheritedBypass); err != nil {
		t.Fatalf("inspect pooled resolver connection: %v", err)
	}
	if inheritedBypass != "" {
		t.Fatalf("resolver leaked privileged RLS setting to pooled request connection")
	}
	if _, err = resolver.Resolve(ctx, "unknown.example.test"); err == nil {
		t.Fatal("unknown hostname must fail closed")
	}
	deactivate, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin deactivate: %v", err)
	}
	if _, err = deactivate.Exec(ctx, `select set_config('app.is_super_admin', 'true', true)`); err != nil {
		t.Fatalf("scope deactivate: %v", err)
	}
	if _, err = deactivate.Exec(ctx, `update app_tenant_hostnames set active=false where hostname='third.example.test'`); err != nil {
		t.Fatalf("deactivate mapping: %v", err)
	}
	if err = deactivate.Commit(ctx); err != nil {
		t.Fatalf("commit deactivate: %v", err)
	}
	if _, err = resolver.Resolve(ctx, "third.example.test"); err == nil {
		t.Fatal("inactive hostname mapping must not resolve")
	}
}
