package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func testOTPFixtureUserID(cfg config.Config) uuid.UUID {
	// Production keeps one permanently reserved, auditable identity. Isolated
	// loopback test servers derive a stable identity from their configured
	// subject and tenant, allowing the real-stack suite to authenticate two
	// independent users without sharing or overwriting a global account.
	if cfg.IsProduction() {
		return oidcTestFixtureUserID
	}
	identity := strings.TrimSpace(cfg.TestOTPFixtureTenantCode) + "\x00" + strings.TrimSpace(cfg.TestOTPFixtureSubject)
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("egueducation-test-otp-fixture\x00"+identity))
}

func testOTPFixturePhone(cfg config.Config) string {
	if cfg.IsProduction() {
		return "+40100000000"
	}
	id := testOTPFixtureUserID(cfg)
	// Romania-formatted synthetic E.164 number: +40 + nine national digits.
	// Its stable suffix follows the already tenant/subject-derived fixture ID,
	// so parallel OIDC fixtures cannot collide in the global identity catalog.
	value := uint32(id[0])<<24 | uint32(id[1])<<16 | uint32(id[2])<<8 | uint32(id[3])
	return fmt.Sprintf("+401%08d", value%100_000_000)
}

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
	user := OIDCTestFixtureUser{ID: testOTPFixtureUserID(cfg), Identifier: cfg.TestOTPFixtureIdentifier, Subject: cfg.TestOTPFixtureSubject}
	phone := testOTPFixturePhone(cfg)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("begin test OTP fixture: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// The authorization-version table is FORCE RLS with no administrative
	// bypass. Fixture provisioning changes membership/module rows, whose
	// triggers write that table, so bind this transaction to the configured
	// fixture tenant before any tenant-scoped write. This mirrors a real
	// request session rather than weakening the table policy for test setup.
	if _, err = tx.Exec(ctx, `
		select
			set_config('app.tenant_id', $1, true),
			set_config('app.is_super_admin', 'true', true)
	`, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("scope test OTP fixture: %w", err)
	}
	// Serialize idempotent provisioning across rolling replicas. The lock is
	// transaction-scoped and contains no credential material.
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('egueducation:oidc-canary:' || $1))`, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("lock test OTP fixture: %w", err)
	}

	var existingID uuid.UUID
	var existingSubject, existingEmail string
	err = tx.QueryRow(ctx, `
		select id, sub, email
		from app_users
		where id=$1 or lower(sub)=lower($2) or lower(email)=lower($3)
		order by case when id=$1 then 0 else 1 end
		limit 1
	`, user.ID, user.Subject, user.Identifier).Scan(&existingID, &existingSubject, &existingEmail)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return OIDCTestFixtureUser{}, fmt.Errorf("inspect test OTP identity collision: %w", err)
	}
	if err == nil && (existingID != user.ID || existingSubject != user.Subject) {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture identity collides with an existing user")
	}
	var conflictingIdentity bool
	if err = tx.QueryRow(ctx, `
		select exists(
			select 1 from app_users
			where id<>$1 and (lower(sub)=lower($2) or lower(email)=lower($3))
		)
	`, user.ID, user.Subject, user.Identifier).Scan(&conflictingIdentity); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("inspect test OTP identity uniqueness: %w", err)
	}
	if conflictingIdentity {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture identity collides with an existing user")
	}
	var crossTenantMembership bool
	if err = tx.QueryRow(ctx, `
		select exists(
			select 1 from app_memberships
			where user_id=$1 and tenant_code<>$2
		)
	`, user.ID, cfg.TestOTPFixtureTenantCode).Scan(&crossTenantMembership); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("inspect test OTP cross-tenant membership: %w", err)
	}
	if crossTenantMembership {
		return OIDCTestFixtureUser{}, fmt.Errorf("test OTP fixture user already belongs to another tenant")
	}
	if _, err = tx.Exec(ctx, `
		insert into app_users (id, sub, name, email, phone_number, locale, status, email_verified, phone_number_verified, preferred_otp_channel)
		values ($1, $2, 'Utilizator Test', $3, $4, 'ro', 'active', true, false, 'sms')
		on conflict (id) do update set
			sub=excluded.sub, name=excluded.name, email=excluded.email,
			status='active', email_verified=true, phone_number_verified=false,
			preferred_otp_channel='sms', updated_at=now()
		where app_users.id=excluded.id
	`, user.ID, user.Subject, user.Identifier, phone); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP user: %w", err)
	}
	// The deterministic test code follows the ordinary SMS proof path.  It
	// never pre-marks the synthetic phone as verified.
	if _, err = tx.Exec(ctx, `
		delete from app_user_identities where user_id = $1 and identity_type = 'phone';
		insert into app_user_identities (user_id, identity_type, normalized_value, display_value, is_primary)
		values ($1, 'phone', $2, $2, true)
	`, user.ID, phone); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP phone identity: %w", err)
	}
	var institutionID, institutionName, rootOrgUnitCode string
	if err = tx.QueryRow(ctx, `
		select institution_id, display_name, root_org_unit_code
		from app_tenants
		where code=$1 and active=true
	`, cfg.TestOTPFixtureTenantCode).Scan(&institutionID, &institutionName, &rootOrgUnitCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("resolve test OTP tenant: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_session_context (user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
		values ($1, $2, $3, array['oidc_redirect', 'sms_otp'], '{}')
		on conflict (user_id) do update set institution_id=excluded.institution_id, institution_name=excluded.institution_name, auth_methods=excluded.auth_methods
	`, user.ID, institutionID, institutionName); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP session context: %w", err)
	}
	if _, err = tx.Exec(ctx, `delete from app_memberships where user_id=$1 and tenant_code=$2`, user.ID, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("reset test OTP memberships: %w", err)
	}
	positionCode := "super_admin"
	if cfg.ProductionE2ECanaryEnabled() {
		positionCode = "e2e_canary"
	}
	if _, err = tx.Exec(ctx, `
		insert into app_memberships (user_id, tenant_code, position_code, org_unit_code, organization_name, is_primary, active, start_date)
		values ($1, $2, $3, $4, $5, true, true, current_date)
	`, user.ID, cfg.TestOTPFixtureTenantCode, positionCode, rootOrgUnitCode, institutionName); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP membership: %w", err)
	}
	if _, err = tx.Exec(ctx, `delete from app_user_modules where user_id=$1 and tenant_code=$2`, user.ID, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("reset test OTP modules: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into app_user_modules (tenant_code, user_id, module_code)
		select $2, $1, code
		from app_modules
		where active = true
		on conflict do nothing
	`, user.ID, cfg.TestOTPFixtureTenantCode); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("seed test OTP modules: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return OIDCTestFixtureUser{}, fmt.Errorf("commit test OTP fixture: %w", err)
	}
	return user, nil
}

func stringsEqualFoldTrimmed(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
