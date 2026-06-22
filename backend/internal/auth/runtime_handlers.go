package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) OIDCHandler() http.Handler {
	return s.oidcHandler
}

func (s *Service) HandleLogoutAlias(w http.ResponseWriter, r *http.Request) {
	if s.oidcHandler == nil {
		http.NotFound(w, r)
		return
	}
	request := r.Clone(r.Context())
	request.URL.Path = "/logout"
	s.oidcHandler.ServeHTTP(w, request)
}

func (s *Service) storePasskeyLoginNonce(ctx context.Context, userID string) (string, error) {
	parsedUserID, err := uuid.Parse(strings.TrimSpace(userID))
	if err != nil {
		return "", err
	}
	return s.storePasskeyLoginNonceUUID(ctx, parsedUserID)
}

func (s *Service) storePasskeyLoginNonceUUID(ctx context.Context, userID uuid.UUID) (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(nonceBytes)
	_, err := s.db.Exec(ctx, `
		insert into oidc_passkey_login_nonces (nonce, user_id, expires_at, created_at)
		values ($1, $2::uuid, $3, now())
	`, nonce, userID, time.Now().Add(2*time.Minute))
	if err != nil {
		return "", err
	}
	return nonce, nil
}

func (s *Service) redeemPasskeyLoginNonce(nonce string) (string, bool) {
	var userID string
	err := s.db.QueryRow(context.Background(), `
		delete from oidc_passkey_login_nonces
		where nonce = $1 and expires_at > now()
		returning user_id::text
	`, strings.TrimSpace(nonce)).Scan(&userID)
	if err != nil || strings.TrimSpace(userID) == "" {
		return "", false
	}
	return userID, true
}
