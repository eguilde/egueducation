package auth

import (
	"context"
	"fmt"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OIDCTestFixtureUser is the synthetic identity used by an isolated browser
// test. EnsureOIDCTestFixtureUser refuses every non-loopback test config.
type OIDCTestFixtureUser struct {
	ID         uuid.UUID
	Identifier string
	Subject    string
}

var oidcTestFixtureUserID = uuid.MustParse("20c36b31-d7e9-4a4b-b6df-42adc5b2913d")

// EnsureOIDCTestFixtureUser provisions no data unless the config has already
// passed all fixture safety predicates. It is intentionally a test bootstrap
// helper, not an HTTP endpoint or migration seed.
func EnsureOIDCTestFixtureUser(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) (OIDCTestFixtureUser, error) {
	if pool == nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture requires a database pool")
	}
	if err := cfg.ValidateTestOTPFixture(); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture is not permitted: %w", err)
	}
	if !cfg.TestOTPFixtureEnabled() {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture is not enabled")
	}
	user := OIDCTestFixtureUser{ID: oidcTestFixtureUserID, Identifier: cfg.TestOTPFixtureIdentifier, Subject: cfg.TestOTPFixtureSubject}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("begin test OTP fixture: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin', 'true', true)`); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("scope test OTP fixture: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_users (id, sub, name, email, phone_number, locale, status, email_verified, phone_number_verified, preferred_otp_channel)
		values ($1, $2, 'OIDC Browser Fixture', $3, '+40100000000', 'ro', 'active', true, true, 'sms')
		on conflict (id) do update set sub=excluded.sub, email=excluded.email, status='active', email_verified=true, phone_number_verified=true, preferred_otp_channel='sms', updated_at=now()
	`, user.ID, user.Subject, user.Identifier); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP user: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_session_context (user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
		values ($1, 'inst-001', 'EguEducation Test Fixture', array['oidc_redirect', 'sms_otp'], '{}')
		on conflict (user_id) do update set auth_methods=excluded.auth_methods
	`, user.ID); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP session context: %w", err)
	}
	if _, err = tx.Exec(ctx, `delete from app_memberships where user_id=$1`, user.ID); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("reset test OTP memberships: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_memberships (user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date)
		values ($1, $2, 'super_admin', 'unit-root', 'EguEducation Test Fixture', true, true, current_date)
	`, user.ID, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP membership: %w", err)
	}
	if _, err = tx.Exec(ctx, `delete from app_user_modules where user_id=$1`, user.ID); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("reset test OTP modules: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_user_modules (tenant_code, user_id, module_code)
		select $2, $1, code from app_modules where active = true
		on conflict do nothing
	`, user.ID, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP modules: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("commit test OTP fixture: %w", err)
	}
	return user, nil
}
