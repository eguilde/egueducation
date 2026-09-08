package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestIdentityContractFoundationIntegration exercises the migration against a
// disposable PostgreSQL database when the integration DSN is supplied. It is
// intentionally skipped in ordinary unit-test runs.
func TestIdentityContractFoundationIntegration(t *testing.T) {
	databaseURL := os.Getenv("EGUEDUCATION_TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("TEST_DATABASE_URL")
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

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	if _, err := pool.Exec(ctx, `select set_config('app.is_super_admin', 'true', false)`); err != nil {
		t.Fatalf("bind maintenance test session: %v", err)
	}

	var platformRoleCount int
	if err := pool.QueryRow(ctx, `
		select count(*)
		from app_user_platform_roles upr
		join app_users u on u.id = upr.user_id
		where u.sub = 'user-thomas-galambos'
			and upr.role_code = 'platform_super_admin'
	`).Scan(&platformRoleCount); err != nil {
		t.Fatalf("read Thomas platform role: %v", err)
	}
	if platformRoleCount != 1 {
		t.Fatalf("Thomas must have exactly one platform_super_admin grant, got %d", platformRoleCount)
	}

	var memberCount, roleCount, identityCount int
	if err := pool.QueryRow(ctx, `
		select
			(select count(*) from app_memberships m where m.user_id = u.id and m.tenant_code = 'tenant-balotesti' and m.active),
			(select count(*) from app_user_roles r where r.user_id = u.id and r.tenant_code = 'tenant-balotesti' and r.role_code = 'admin'),
			(select count(*) from app_user_identities i where i.user_id = u.id and i.verified_at is not null)
		from app_users u
		where u.sub = 'user-stelian-fedorca'
	`).Scan(&memberCount, &roleCount, &identityCount); err != nil {
		t.Fatalf("read Stelian tenant authorization: %v", err)
	}
	if memberCount != 1 || roleCount != 1 || identityCount < 1 {
		t.Fatalf("Stelian tenant authorization is incomplete: memberships=%d roles=%d identities=%d", memberCount, roleCount, identityCount)
	}

	if _, err := pool.Exec(ctx, `select set_config('app.tenant_id', 'tenant-balotesti', false)`); err != nil {
		t.Fatalf("bind Balotesti authorization context: %v", err)
	}
	var versionCount int
	if err := pool.QueryRow(ctx, `
		select count(*)
		from app_tenant_authorization_versions v
		join app_users u on u.id = v.user_id
		where v.tenant_code = 'tenant-balotesti'
			and u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
	`).Scan(&versionCount); err != nil {
		t.Fatalf("read tenant authorization versions: %v", err)
	}
	if versionCount != 3 {
		t.Fatalf("expected authorization versions for three Balotesti operators, got %d", versionCount)
	}
}
