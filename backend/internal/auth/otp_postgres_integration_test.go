package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPhoneOTPPostgresIntegration is deliberately a database behaviour test,
// not a SQL-text test.  It proves that an administrator cannot assert phone
// possession, while the ordinary one-time OTP path is the only route that
// promotes both the global identity and the profile projection.  CI always
// provides TEST_DATABASE_URL for this mandatory test.
func TestPhoneOTPPostgresIntegration(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("phone OTP PostgreSQL integration requires TEST_DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	userID := uuid.New()
	unique := strings.ReplaceAll(userID.String(), "-", "")
	subject := "otp-postgres-" + unique
	phone := fmt.Sprintf("+407%08d", time.Now().UnixNano()%100000000)
	if len(phone) < 10 || len(phone) > 15 { // E.164 Romanian test value.
		t.Fatalf("invalid generated fixture phone %q", phone)
	}
	seedPhoneOTPUser(t, ctx, pool, userID, subject, "otp-postgres-"+unique+"@example.test", phone)
	t.Cleanup(func() { cleanupPhoneOTPUser(t, pool, userID) })

	// Direct profile promotion must fail at transaction commit because the
	// matching primary identity has not been verified by an OTP transaction.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin direct promotion test: %v", err)
	}
	if _, err = tx.Exec(ctx, `select set_config('app.is_super_admin','true',true)`); err != nil {
		t.Fatalf("scope direct promotion test: %v", err)
	}
	if _, err = tx.Exec(ctx, `update app_users set phone_number_verified=true where id=$1`, userID); err != nil {
		t.Fatalf("set unverified profile phone flag: %v", err)
	}
	if err = tx.Commit(ctx); err == nil {
		t.Fatal("direct phone_verified promotion unexpectedly committed")
	}

	var verified bool
	if err := pool.QueryRow(ctx, `select phone_number_verified from app_users where id=$1`, userID).Scan(&verified); err != nil {
		t.Fatalf("read profile after rejected direct promotion: %v", err)
	}
	if verified {
		t.Fatal("rejected direct promotion persisted phone_number_verified")
	}

	// The global login-identity unique constraint prevents a phone being bound
	// to a second account, regardless of tenant membership.
	secondID := uuid.New()
	if _, err := pool.Exec(ctx, `insert into app_users (id,sub,name,email,phone_number,phone_number_verified) values ($1,$2,'Second OTP test user',$3,'',false)`, secondID, "otp-postgres-second-"+unique, "otp-postgres-second-"+unique+"@example.test"); err != nil {
		t.Fatalf("seed second phone uniqueness user: %v", err)
	}
	t.Cleanup(func() { cleanupPhoneOTPUser(t, pool, secondID) })
	if _, err := pool.Exec(ctx, `insert into app_user_identities(user_id,identity_type,normalized_value,display_value,is_primary) values($1,'phone',$2,$2,true)`, secondID, phone); err == nil {
		t.Fatal("duplicate global phone identity unexpectedly inserted")
	}

	service, err := newOTPService(pool, otpHMACTestKeyOne)
	if err != nil {
		t.Fatalf("create OTP service: %v", err)
	}
	const authnSessionID = "otp-postgres-session-primary"
	if _, err := service.GenerateFixture(ctx, userID, otpPurposeLogin, "tenant-egueducation", authnSessionID, "482615"); err != nil {
		t.Fatalf("generate fixed OTP through regular storage: %v", err)
	}
	// A challenge is proof for this exact immutable phone identity, not merely
	// for the mutable profile/user row.  A later phone replacement must not be
	// able to consume a code delivered to the former number.
	var challengeIdentityID, primaryIdentityID uuid.UUID
	if err := pool.QueryRow(ctx, `select identity_id from oidc_otp_challenges where user_id=$1 and purpose='login' and tenant_code='tenant-egueducation' and authn_session_id=$2`, userID, authnSessionID).Scan(&challengeIdentityID); err != nil {
		t.Fatalf("read OTP identity binding: %v", err)
	}
	if err := pool.QueryRow(ctx, `select id from app_user_identities where user_id=$1 and identity_type='phone' and is_primary`, userID).Scan(&primaryIdentityID); err != nil {
		t.Fatalf("read primary phone identity: %v", err)
	}
	if challengeIdentityID != primaryIdentityID {
		t.Fatalf("OTP identity binding = %s, want primary phone identity %s", challengeIdentityID, primaryIdentityID)
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-egueducation", authnSessionID, "000000"); err == nil {
		t.Fatal("invalid OTP unexpectedly verified the phone")
	} else {
		service.RecordPhoneLoginOTPFailure(ctx, userID, "tenant-egueducation", err)
	}
	// Audit is evidence, not an unbounded attacker-controlled log sink. A
	// repeated invalid submission can create at most the OTP attempt budget's
	// worth of retained failed events for this user/outcome window.
	for range 12 {
		service.RecordPhoneLoginOTPFailure(ctx, userID, "tenant-egueducation", errors.New("otp: invalid code"))
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-balotesti", authnSessionID, "482615"); err == nil {
		t.Fatal("OTP issued for tenant-egueducation authenticated on another tenant")
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-egueducation", "otp-postgres-session-other", "482615"); err == nil {
		t.Fatal("OTP issued for one OIDC session authenticated another session")
	}
	// Even a correctly hashed challenge naming another tenant cannot authorize
	// a user who has no active membership there. This exercises the narrowly
	// bypassed global tenant-directory lookup rather than relying on the earlier
	// challenge tenant mismatch check.
	const noMembershipSessionID = "otp-postgres-session-no-membership"
	if _, err := service.GenerateFixture(ctx, userID, otpPurposeLogin, "tenant-balotesti", noMembershipSessionID, "731946"); err != nil {
		t.Fatalf("generate cross-tenant no-membership OTP: %v", err)
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-balotesti", noMembershipSessionID, "731946"); err == nil || !strings.Contains(err.Error(), "no active tenant membership") {
		t.Fatalf("cross-tenant OTP without membership error = %v, want no active tenant membership", err)
	}
	if err := pool.QueryRow(ctx, `select phone_number_verified from app_users where id=$1`, userID).Scan(&verified); err != nil {
		t.Fatalf("read phone after rejected cross-tenant OTP: %v", err)
	}
	if verified {
		t.Fatal("cross-tenant OTP without membership promoted phone verification")
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-egueducation", authnSessionID, "482615"); err != nil {
		t.Fatalf("verify valid OTP: %v", err)
	}
	if err := service.VerifyPhoneLogin(ctx, userID, "tenant-egueducation", authnSessionID, "482615"); err == nil {
		t.Fatal("replayed OTP unexpectedly authenticated")
	} else {
		service.RecordPhoneLoginOTPFailure(ctx, userID, "tenant-egueducation", err)
	}

	if err := pool.QueryRow(ctx, `
		select u.phone_number_verified and exists(
			select 1 from app_user_identities i
			where i.user_id=u.id and i.identity_type='phone' and i.is_primary and i.verified_at is not null
		)
		from app_users u where u.id=$1
	`, userID).Scan(&verified); err != nil {
		t.Fatalf("read verified phone projections: %v", err)
	}
	if !verified {
		t.Fatal("successful OTP did not atomically verify both phone projections")
	}

	var successCount, invalidCount, replayCount int
	auditTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin OTP audit assertion: %v", err)
	}
	defer auditTx.Rollback(ctx) //nolint:errcheck
	if _, err := auditTx.Exec(ctx, `select set_config('app.tenant_id','tenant-egueducation',true), set_config('app.institution_id','inst-001',true)`); err != nil {
		t.Fatalf("scope OTP audit assertion: %v", err)
	}
	if err := auditTx.QueryRow(ctx, `
		select
			count(*) filter (where action='identity.phone.enrollment_verified' and status='success'),
			count(*) filter (where action='identity.phone.otp_verification' and details->>'outcome'='invalid' and status='failed'),
			count(*) filter (where action='identity.phone.otp_verification' and details->>'outcome'='replayed_or_missing' and status='failed')
		from app_audit_log
		where institution_id='inst-001' and details->>'user_id'=$1::text
	`, userID).Scan(&successCount, &invalidCount, &replayCount); err != nil {
		t.Fatalf("read OTP audit evidence: %v", err)
	}
	if successCount != 1 || invalidCount < 1 || invalidCount > otpMaxAttempts || replayCount != 1 {
		t.Fatalf("OTP audit evidence success=%d invalid=%d replay=%d, want success=1 invalid=1..%d replay=1", successCount, invalidCount, replayCount, otpMaxAttempts)
	}
}

func seedPhoneOTPUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, subject, email, phone string) {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin OTP fixture: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `select set_config('app.tenant_id','tenant-egueducation',true), set_config('app.institution_id','inst-001',true), set_config('app.is_super_admin','true',true)`); err != nil {
		t.Fatalf("scope OTP fixture: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into app_users(id,sub,name,email,phone_number,phone_number_verified,preferred_otp_channel) values($1,$2,'OTP PostgreSQL integration',$3,'',false,'sms')`, userID, subject, email); err != nil {
		t.Fatalf("insert OTP fixture user: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into app_user_identities(user_id,identity_type,normalized_value,display_value,is_primary) values($1,'phone',$2,$2,true)`, userID, phone); err != nil {
		t.Fatalf("insert OTP fixture phone identity: %v", err)
	}
	if _, err = tx.Exec(ctx, `update app_users set phone_number=$2 where id=$1`, userID, phone); err != nil {
		t.Fatalf("project OTP fixture phone: %v", err)
	}
	if _, err = tx.Exec(ctx, `insert into app_memberships(user_id,tenant_code,position_code,organization_name,org_unit_code,is_primary,active,start_date) values($1,'tenant-egueducation','registrator','EguEducation OTP integration','unit-root',true,true,current_date)`, userID); err != nil {
		t.Fatalf("insert OTP fixture membership: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit OTP fixture: %v", err)
	}
}

func cleanupPhoneOTPUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Errorf("begin OTP fixture cleanup: %v", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err = tx.Exec(ctx, `select set_config('app.tenant_id','tenant-egueducation',true), set_config('app.institution_id','inst-001',true), set_config('app.is_super_admin','true',true)`); err != nil {
		t.Errorf("scope OTP fixture cleanup: %v", err)
		return
	}
	if _, err = tx.Exec(ctx, `delete from app_audit_log where details->>'user_id'=$1::text`, userID); err != nil {
		t.Errorf("remove OTP fixture audits: %v", err)
		return
	}
	if _, err = tx.Exec(ctx, `delete from app_users where id=$1`, userID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("remove OTP fixture user: %v", err)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		t.Errorf("commit OTP fixture cleanup: %v", err)
	}
}
