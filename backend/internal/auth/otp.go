package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type otpPurpose string

const (
	otpPurposeLogin   otpPurpose = "login"
	otpLength                    = 6
	otpTTL                       = 10 * time.Minute
	otpMaxAttempts               = 5
	otpResendCooldown            = 60 * time.Second
	otpHashPrefix                = "hmac-sha256:v1:"
)

type otpService struct {
	db      *pgxpool.Pool
	hmacKey []byte
}

func newOTPService(db *pgxpool.Pool, hmacKey string) (*otpService, error) {
	key := []byte(strings.TrimSpace(hmacKey))
	if len(key) < 32 {
		return nil, errors.New("otp: HMAC key must contain at least 32 bytes")
	}
	// Copy the key so the caller cannot mutate the in-process key material.
	return &otpService{db: db, hmacKey: append([]byte(nil), key...)}, nil
}

func (s *otpService) Generate(ctx context.Context, userID uuid.UUID, purpose otpPurpose, tenantCode, authnSessionID string) (string, error) {
	code, err := randomDigits(otpLength)
	if err != nil {
		return "", fmt.Errorf("otp: generate: %w", err)
	}
	return s.store(ctx, userID, purpose, tenantCode, authnSessionID, code)
}

// GenerateFixture stores an already validated deterministic code for an
// automated test. Callers must apply their own test-environment and synthetic
// identity gate before using it; this method has no production configuration.
func (s *otpService) GenerateFixture(ctx context.Context, userID uuid.UUID, purpose otpPurpose, tenantCode, authnSessionID, code string) (string, error) {
	if len(code) != otpLength {
		return "", errors.New("otp: invalid fixture code")
	}
	for _, digit := range code {
		if digit < '0' || digit > '9' {
			return "", errors.New("otp: invalid fixture code")
		}
	}
	return s.store(ctx, userID, purpose, tenantCode, authnSessionID, code)
}

func (s *otpService) store(ctx context.Context, userID uuid.UUID, purpose otpPurpose, tenantCode, authnSessionID, code string) (string, error) {
	tenantCode = strings.TrimSpace(tenantCode)
	authnSessionID = strings.TrimSpace(authnSessionID)
	if tenantCode == "" || authnSessionID == "" {
		return "", errors.New("otp: tenant and authentication session are required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("otp: begin store: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var identityID uuid.UUID
	err = tx.QueryRow(ctx, `
		select identity.id
		from app_user_identities identity
		join app_users profile on profile.id=identity.user_id
		where identity.user_id=$1::uuid
			and identity.identity_type='phone'
			and identity.is_primary
			and identity.normalized_value = case
				when regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g') like '+%' then regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(profile.phone_number), '[^0-9]', '', 'g') like '07%' then '+40' || substr(regexp_replace(btrim(profile.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g')
			end
		for key share
	`, userID).Scan(&identityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("otp: assigned phone identity is missing")
	}
	if err != nil {
		return "", fmt.Errorf("otp: resolve phone identity: %w", err)
	}
	result, err := tx.Exec(ctx, `
		insert into oidc_otp_challenges (user_id, purpose, tenant_code, authn_session_id, code_hash, expires_at, attempts, identity_id)
		values ($1::uuid, $2, $3, $4, $5, $6, 0, $8)
		on conflict (authn_session_id, purpose) do update
		set code_hash = excluded.code_hash,
			expires_at = excluded.expires_at,
			attempts = 0,
			user_id = excluded.user_id,
			tenant_code = excluded.tenant_code,
			identity_id = excluded.identity_id,
			updated_at = now()
		where oidc_otp_challenges.updated_at <= now() - make_interval(secs => $7)
	`, userID, string(purpose), tenantCode, authnSessionID, s.hashCode(code), time.Now().Add(otpTTL), int(otpResendCooldown.Seconds()), identityID)
	if err != nil {
		return "", fmt.Errorf("otp: store: %w", err)
	}
	if result.RowsAffected() == 0 {
		return "", errors.New("otp: resend cooldown active")
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("otp: commit store: %w", err)
	}

	return code, nil
}

// VerifyPhoneLogin consumes a one-time SMS code and promotes precisely the
// assigned primary phone identity in the same database transaction.
func (s *otpService) VerifyPhoneLogin(ctx context.Context, userID uuid.UUID, tenantCode, authnSessionID, code string) error {
	tenantCode = strings.TrimSpace(tenantCode)
	authnSessionID = strings.TrimSpace(authnSessionID)
	if tenantCode == "" || authnSessionID == "" {
		return errors.New("otp: tenant and authentication session are required")
	}
	return s.verify(ctx, userID, otpPurposeLogin, tenantCode, authnSessionID, code, func(tx pgx.Tx, challengedIdentityID uuid.UUID) error {
		institutionID, err := bindPhoneOTPUserTenant(ctx, tx, tenantCode, userID)
		if err != nil {
			return err
		}

		var (
			wasVerified bool
		)
		err = tx.QueryRow(ctx, `
			select i.verified_at is not null
			from app_user_identities i join app_users u on i.user_id=u.id
			where i.id=$2 and i.user_id = $1 and i.identity_type = 'phone' and i.is_primary
			  and i.normalized_value = case
				when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then '+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
			  end
			for update
		`, userID, challengedIdentityID).Scan(&wasVerified)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("otp: assigned phone identity is missing")
			}
			return fmt.Errorf("otp: promote phone identity: %w", err)
		}
		if _, err := tx.Exec(ctx, `update app_user_identities set verified_at=coalesce(verified_at, now()), updated_at=now() where id=$1`, challengedIdentityID); err != nil {
			return fmt.Errorf("otp: promote phone identity: %w", err)
		}
		if _, err := tx.Exec(ctx, `update app_users set phone_number_verified = true, updated_at = now() where id = $1`, userID); err != nil {
			return fmt.Errorf("otp: promote profile phone: %w", err)
		}
		action := "identity.phone.enrollment_verified"
		summary := "Verified assigned phone through SMS OTP possession proof."
		if wasVerified {
			action = "identity.phone.login_authenticated"
			summary = "Authenticated with an already verified phone through SMS OTP."
		}
		if _, err := tx.Exec(ctx, `
			insert into app_audit_log (institution_id, actor_subject, action, target_type, target_id, status, summary, details)
			select $1, u.sub, $4, 'app_user_identity', $2::text,
			       'success', $5,
			       jsonb_build_object('user_id', $3::text, 'identity_id', $2::text, 'method', 'sms_otp', 'purpose', 'login', 'outcome', 'success')
			from app_users u where u.id = $3::uuid
		`, institutionID, challengedIdentityID, userID, action, summary); err != nil {
			return fmt.Errorf("otp: append phone verification audit: %w", err)
		}
		return nil
	})
}

// RecordPhoneLoginOTPFailure records only an outcome classification and opaque
// identifiers. It intentionally never stores a phone value, submitted code,
// OTP hash, or provider payload.
func (s *otpService) RecordPhoneLoginOTPFailure(ctx context.Context, userID uuid.UUID, tenantCode string, verifyErr error) {
	if s == nil || s.db == nil || strings.TrimSpace(tenantCode) == "" {
		return
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	institutionID, err := bindPhoneOTPUserTenant(ctx, tx, tenantCode, userID)
	if err != nil {
		return
	}
	var subject string
	if err = tx.QueryRow(ctx, `select sub from app_users where id=$1`, userID).Scan(&subject); err != nil {
		return
	}
	outcome := otpFailureOutcome(verifyErr)
	limit := otpMaxAttempts
	if outcome != "invalid" {
		// Expiry, lock and replay are state outcomes, not per-request telemetry.
		// One event per OTP TTL is enough evidence and prevents an unauthenticated
		// caller from amplifying the append-only audit log.
		limit = 1
	}
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1, 0))`, userID.String()+":"+outcome); err != nil {
		return
	}
	if _, err = tx.Exec(ctx, `
		insert into app_audit_log (institution_id, actor_subject, action, target_type, target_id, status, summary, details)
		select $1, $2, 'identity.phone.otp_verification', 'app_user', $3::text, 'failed',
			'Phone OTP verification did not complete.', jsonb_build_object('user_id', $3::text, 'method', 'sms_otp', 'purpose', 'login', 'outcome', $4::text)
		where (
			select count(*) from app_audit_log existing
			where existing.institution_id=$1
				and existing.action='identity.phone.otp_verification'
				and existing.target_type='app_user'
				and existing.target_id=$3::text
				and existing.details->>'outcome'=$4::text
				and existing.created_at >= now() - make_interval(secs => $6)
		) < $5
	`, institutionID, subject, userID, outcome, limit, int(otpTTL.Seconds())); err != nil {
		return
	}
	_ = tx.Commit(ctx)
}

// bindPhoneOTPUserTenant resolves the global tenant directory under a narrowly
// bounded internal bypass because app_tenants intentionally exposes no
// tenant-scoped read policy. The query still requires the exact active tenant,
// user and membership. The bypass is then dropped and verified before any
// identity, profile or audit mutation is allowed to proceed.
func bindPhoneOTPUserTenant(ctx context.Context, tx pgx.Tx, tenantCode string, userID uuid.UUID) (string, error) {
	if _, err := tx.Exec(ctx, `
		select
			set_config('app.tenant_id', $1, true),
			set_config('app.is_super_admin', 'true', true)
	`, tenantCode); err != nil {
		return "", fmt.Errorf("otp: scope tenant directory lookup: %w", err)
	}

	var institutionID string
	err := tx.QueryRow(ctx, `
		select tenant.institution_id
		from app_tenants tenant
		where tenant.code=$1 and tenant.active
		  and exists (
			select 1
			from app_memberships membership
			where membership.user_id=$2
			  and membership.tenant_code=tenant.code
			  and membership.active
			  and membership.start_date <= current_date
			  and (membership.end_date is null or membership.end_date >= current_date)
		  )
	`, tenantCode, userID).Scan(&institutionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("otp: no active tenant membership")
		}
		return "", fmt.Errorf("otp: resolve verification tenant: %w", err)
	}

	// Keep the state transition and its assertion in separate statements:
	// PostgreSQL does not promise left-to-right evaluation of SELECT targets.
	if _, err := tx.Exec(ctx, `
		select
			set_config('app.institution_id', $1, true),
			set_config('app.is_super_admin', 'false', true)
	`, institutionID); err != nil {
		return "", fmt.Errorf("otp: bind verification tenant: %w", err)
	}
	var bypassStillEnabled bool
	if err := tx.QueryRow(ctx, `select public.can_bypass_tenant_rls()`).Scan(&bypassStillEnabled); err != nil {
		return "", fmt.Errorf("otp: verify tenant directory bypass state: %w", err)
	}
	if bypassStillEnabled {
		return "", errors.New("otp: tenant directory bypass remained enabled")
	}
	return institutionID, nil
}

func otpFailureOutcome(err error) string {
	value := strings.ToLower(fmt.Sprint(err))
	switch {
	case strings.Contains(value, "expired"):
		return "expired"
	case strings.Contains(value, "too many attempts"):
		return "locked"
	case strings.Contains(value, "no code found"):
		return "replayed_or_missing"
	default:
		return "invalid"
	}
}

func (s *otpService) verify(ctx context.Context, userID uuid.UUID, purpose otpPurpose, tenantCode, authnSessionID, code string, onSuccess func(pgx.Tx, uuid.UUID) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("otp: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var (
		storedCode string
		expiresAt  time.Time
		attempts   int
		identityID sql.NullString
	)

	err = tx.QueryRow(ctx, `
		select code_hash, expires_at, attempts, identity_id::text
		from oidc_otp_challenges
		where user_id = $1::uuid and purpose = $2 and tenant_code = $3 and authn_session_id = $4
		for update
	`, userID, string(purpose), tenantCode, authnSessionID).Scan(&storedCode, &expiresAt, &attempts, &identityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("otp: no code found")
	}
	if err != nil {
		return fmt.Errorf("otp: query: %w", err)
	}
	if attempts >= otpMaxAttempts {
		return errors.New("otp: too many attempts")
	}
	if time.Now().After(expiresAt) {
		return errors.New("otp: code expired")
	}
	match, currentFormat := s.matchesStoredCode(storedCode, code)
	challengedIdentityID, identityErr := uuid.Parse(identityID.String)
	if !identityID.Valid || identityErr != nil {
		currentFormat = false
	}
	if !currentFormat {
		// SHA-only records from prior versions cannot prove possession after the
		// keyed-hash upgrade.  Remove them so the normal resend flow can issue a
		// current, ten-minute OTP immediately; they are never accepted.
		if _, err := tx.Exec(ctx, `
			delete from oidc_otp_challenges
			where user_id = $1::uuid and purpose = $2 and tenant_code = $3 and authn_session_id = $4
		`, userID, string(purpose), tenantCode, authnSessionID); err != nil {
			return fmt.Errorf("otp: remove legacy code: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("otp: commit legacy removal: %w", err)
		}
		return errors.New("otp: code requires regeneration")
	}
	if !match {
		if _, err := tx.Exec(ctx, `
			update oidc_otp_challenges
			set attempts = attempts + 1,
				updated_at = now()
			where user_id = $1::uuid and purpose = $2 and tenant_code = $3 and authn_session_id = $4
		`, userID, string(purpose), tenantCode, authnSessionID); err != nil {
			return fmt.Errorf("otp: increment attempts: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("otp: commit attempt: %w", err)
		}
		return errors.New("otp: invalid code")
	}

	if onSuccess != nil {
		if err := onSuccess(tx, challengedIdentityID); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `
		delete from oidc_otp_challenges
		where user_id = $1::uuid and purpose = $2 and tenant_code = $3 and authn_session_id = $4
	`, userID, string(purpose), tenantCode, authnSessionID)
	if err != nil {
		return fmt.Errorf("otp: delete after verify: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("otp: commit: %w", err)
	}
	return nil
}

func (s *otpService) hashCode(code string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	_, _ = mac.Write([]byte(code))
	return otpHashPrefix + hex.EncodeToString(mac.Sum(nil))
}

// matchesStoredCode only accepts the explicit current storage format.  A
// malformed or legacy SHA-256 row is rejected before comparison, never
// silently treated as a valid HMAC value.
func (s *otpService) matchesStoredCode(storedCode, code string) (match bool, currentFormat bool) {
	if !strings.HasPrefix(storedCode, otpHashPrefix) {
		return false, false
	}
	encoded := strings.TrimPrefix(storedCode, otpHashPrefix)
	storedMAC, err := hex.DecodeString(encoded)
	if err != nil || len(storedMAC) != sha256.Size {
		return false, false
	}
	expected := hmac.New(sha256.New, s.hmacKey)
	_, _ = expected.Write([]byte(code))
	return subtle.ConstantTimeCompare(storedMAC, expected.Sum(nil)) == 1, true
}

func randomDigits(length int) (string, error) {
	max := big.NewInt(1)
	for index := 0; index < length; index++ {
		max.Mul(max, big.NewInt(10))
	}
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", length, num), nil
}
