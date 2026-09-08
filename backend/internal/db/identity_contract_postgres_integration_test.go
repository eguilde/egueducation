package db

import (
	"context"
	"os"
	"strings"
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

	var memberCount, roleCount, identityCount, verifiedPhoneCount int
	if err := pool.QueryRow(ctx, `
		select
			(select count(*) from app_memberships m where m.user_id = u.id and m.tenant_code = 'tenant-balotesti' and m.active),
			(select count(*) from app_user_roles r where r.user_id = u.id and r.tenant_code = 'tenant-balotesti' and r.role_code = 'admin'),
			(select count(*) from app_user_identities i where i.user_id = u.id),
			(select count(*) from app_user_identities i where i.user_id = u.id and i.identity_type = 'phone' and i.verified_at is not null)
		from app_users u
		where u.sub = 'user-stelian-fedorca'
	`).Scan(&memberCount, &roleCount, &identityCount, &verifiedPhoneCount); err != nil {
		t.Fatalf("read Stelian tenant authorization: %v", err)
	}
	if memberCount != 1 || roleCount != 1 || identityCount < 1 || verifiedPhoneCount != 0 {
		t.Fatalf("Stelian tenant authorization is incomplete or auto-verified: memberships=%d roles=%d identities=%d verified_phones=%d", memberCount, roleCount, identityCount, verifiedPhoneCount)
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

// TestPublished0083ReconciliationIntegration proves the forward-only repair
// against the exact dangerous upgrade state: a reserved operator subject that
// existed long before 0083, but whose phone was marked verified by the old
// bootstrap without a durable SMS possession-proof audit event.
func TestPublished0083ReconciliationIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("published 0083 reconciliation requires TEST_DATABASE_URL")
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
		t.Fatalf("apply current migrations: %v", err)
	}

	const subject = "user-stelian-fedorca"
	const phone = "+40744652476"
	var userID string
	if err := pool.QueryRow(ctx, `select id::text from app_users where sub=$1`, subject).Scan(&userID); err != nil {
		t.Fatalf("resolve reserved operator: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin vulnerable 0083 state: %v", err)
	}
	if _, err = tx.Exec(ctx, `select set_config('app.tenant_id','tenant-balotesti',true), set_config('app.institution_id','inst-balotesti',true)`); err == nil {
		_, err = tx.Exec(ctx, `update app_users set created_at=now()-interval '2 years' where id=$1::uuid`, userID)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `update app_user_identities set verified_at=now() where user_id=$1::uuid and identity_type='phone' and normalized_value=$2`, userID, phone)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `update app_users set phone_number_verified=true where id=$1::uuid`, userID)
	}
	if err == nil {
		err = tx.Commit(ctx)
	} else {
		_ = tx.Rollback(ctx)
	}
	if err != nil {
		t.Fatalf("seed vulnerable published-0083 state: %v", err)
	}

	reconciliation, err := migrationFiles.ReadFile("migrations/0095_identity_contract_published_0083_reconciliation.sql")
	if err != nil {
		t.Fatalf("read reconciliation migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(reconciliation)); err != nil {
		t.Fatalf("reapply forward reconciliation to published-0083 state: %v", err)
	}

	var profileVerified, identityVerified bool
	if err := pool.QueryRow(ctx, `
		select profile.phone_number_verified, identity.verified_at is not null
		from app_users profile
		join app_user_identities identity on identity.user_id=profile.id
		where profile.id=$1::uuid and identity.identity_type='phone' and identity.normalized_value=$2
	`, userID, phone).Scan(&profileVerified, &identityVerified); err != nil {
		t.Fatalf("read reconciled verification state: %v", err)
	}
	if profileVerified || identityVerified {
		t.Fatalf("published 0083 verification survived without SMS proof: profile=%t identity=%t", profileVerified, identityVerified)
	}
}
