package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type otpPurpose string

const (
	otpPurposeLogin otpPurpose = "login"
	otpLength                  = 6
	otpTTL                     = 10 * time.Minute
	otpMaxAttempts             = 5
)

type otpService struct {
	db *pgxpool.Pool
}

func newOTPService(db *pgxpool.Pool) *otpService {
	return &otpService{db: db}
}

func (s *otpService) Generate(ctx context.Context, userID uuid.UUID, purpose otpPurpose) (string, error) {
	code, err := randomDigits(otpLength)
	if err != nil {
		return "", fmt.Errorf("otp: generate: %w", err)
	}

	_, err = s.db.Exec(ctx, `
		insert into oidc_otp_codes (user_id, purpose, code_hash, expires_at, attempts)
		values ($1::uuid, $2, $3, $4, 0)
		on conflict (user_id, purpose) do update
		set code_hash = excluded.code_hash,
			expires_at = excluded.expires_at,
			attempts = 0,
			updated_at = now()
	`, userID, string(purpose), hashOTPCode(code), time.Now().Add(otpTTL))
	if err != nil {
		return "", fmt.Errorf("otp: store: %w", err)
	}

	return code, nil
}

func (s *otpService) Verify(ctx context.Context, userID uuid.UUID, purpose otpPurpose, code string) error {
	var (
		storedCode string
		expiresAt  time.Time
		attempts   int
	)

	err := s.db.QueryRow(ctx, `
		select code_hash, expires_at, attempts
		from oidc_otp_codes
		where user_id = $1::uuid and purpose = $2
	`, userID, string(purpose)).Scan(&storedCode, &expiresAt, &attempts)
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
	if subtle.ConstantTimeCompare([]byte(hashOTPCode(code)), []byte(storedCode)) != 1 {
		_, _ = s.db.Exec(ctx, `
			update oidc_otp_codes
			set attempts = attempts + 1,
				updated_at = now()
			where user_id = $1::uuid and purpose = $2
		`, userID, string(purpose))
		return errors.New("otp: invalid code")
	}

	_, err = s.db.Exec(ctx, `
		delete from oidc_otp_codes
		where user_id = $1::uuid and purpose = $2
	`, userID, string(purpose))
	if err != nil {
		return fmt.Errorf("otp: delete after verify: %w", err)
	}
	return nil
}

func hashOTPCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
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
