package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/go-jose/go-jose/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luikyv/go-oidc/pkg/goidc"
)

type keyManager struct {
	db  *pgxpool.Pool
	cfg *config.Config

	mu         sync.RWMutex
	currentKey *rsa.PrivateKey
	currentKID string
	activeKeys []goidc.JSONWebKey
}

func newKeyManager(db *pgxpool.Pool, cfg *config.Config) *keyManager {
	return &keyManager{db: db, cfg: cfg}
}

func (km *keyManager) Init(ctx context.Context) error {
	type jwksRow struct {
		KeyID      string
		PrivateKey string
		CreatedAt  time.Time
		RotatedAt  *time.Time
	}

	rows, err := km.db.Query(ctx, `
		select key_id, private_key, created_at, rotated_at
		from oidc_jwks_keys
		where active = true
		order by created_at desc
	`)
	if err != nil {
		return fmt.Errorf("query jwks keys: %w", err)
	}
	defer rows.Close()

	keys := make([]jwksRow, 0, 4)
	for rows.Next() {
		var row jwksRow
		if err := rows.Scan(&row.KeyID, &row.PrivateKey, &row.CreatedAt, &row.RotatedAt); err != nil {
			return fmt.Errorf("scan jwks key: %w", err)
		}
		keys = append(keys, row)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read jwks keys: %w", err)
	}

	if len(keys) == 0 {
		if err := km.generateAndStore(ctx, "sig-rs256"); err != nil {
			return fmt.Errorf("generate initial key: %w", err)
		}
		return km.Init(ctx)
	}

	newest := keys[0]
	currentKey, err := parsePEMPrivateKey(newest.PrivateKey)
	if err != nil {
		return fmt.Errorf("parse current key %q: %w", newest.KeyID, err)
	}

	rotationAge := time.Duration(km.cfg.JWTKeyRotationDays) * 24 * time.Hour
	if newest.RotatedAt == nil && time.Since(newest.CreatedAt) > rotationAge {
		newKID := nextKID(newest.KeyID)
		if _, err := km.db.Exec(ctx, `
			update oidc_jwks_keys
			set rotated_at = now()
			where key_id = $1 and rotated_at is null
		`, newest.KeyID); err != nil {
			return fmt.Errorf("mark old key rotated: %w", err)
		}
		if err := km.generateAndStore(ctx, newKID); err != nil {
			return fmt.Errorf("generate rotated key: %w", err)
		}
		return km.Init(ctx)
	}

	overlapDuration := time.Duration(km.cfg.JWTKeyOverlapHours) * time.Hour
	for _, key := range keys {
		if key.RotatedAt != nil && time.Since(*key.RotatedAt) > overlapDuration {
			_, _ = km.db.Exec(ctx, `
				update oidc_jwks_keys
				set active = false
				where key_id = $1
			`, key.KeyID)
		}
	}

	activeJWKs := make([]goidc.JSONWebKey, 0, len(keys))
	for _, key := range keys {
		if key.RotatedAt != nil && time.Since(*key.RotatedAt) > overlapDuration {
			continue
		}
		privateKey, err := parsePEMPrivateKey(key.PrivateKey)
		if err != nil {
			continue
		}
		activeJWKs = append(activeJWKs, goidc.JSONWebKey{
			KeyID:     key.KeyID,
			Key:       privateKey,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		})
	}

	km.mu.Lock()
	km.currentKey = currentKey
	km.currentKID = newest.KeyID
	km.activeKeys = activeJWKs
	km.mu.Unlock()
	return nil
}

func (km *keyManager) ActiveJWKS() goidc.JSONWebKeySet {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return goidc.JSONWebKeySet{Keys: km.activeKeys}
}

func (km *keyManager) StartRotationCheck(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = km.Init(ctx)
			}
		}
	}()
}

func (km *keyManager) generateAndStore(ctx context.Context, kid string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate RSA key: %w", err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	publicPEM, err := publicKeyPEM(key)
	if err != nil {
		return fmt.Errorf("marshal public PEM: %w", err)
	}
	jwkJSON, err := publicJWKJSON(kid, key)
	if err != nil {
		return fmt.Errorf("marshal JWK: %w", err)
	}

	_, err = km.db.Exec(ctx, `
		insert into oidc_jwks_keys (key_id, use, alg, private_key, public_key, jwk, active, created_at)
		values ($1, 'sig', 'RS256', $2, $3, $4::jsonb, true, now())
	`, kid, string(privatePEM), publicPEM, string(jwkJSON))
	if err != nil {
		return fmt.Errorf("insert new key: %w", err)
	}
	return nil
}

func parsePEMPrivateKey(pemValue string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemValue))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS1/PKCS8: %w", err)
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("parsed key is not RSA (got %T)", parsed)
	}
	return rsaKey, nil
}

func publicKeyPEM(key *rsa.PrivateKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	})), nil
}

func publicJWKJSON(kid string, key *rsa.PrivateKey) (json.RawMessage, error) {
	jwk := jose.JSONWebKey{
		KeyID:     kid,
		Key:       &key.PublicKey,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	data, err := json.Marshal(jwk)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func nextKID(kid string) string {
	lastDash := strings.LastIndex(kid, "-v")
	if lastDash >= 0 {
		suffix := kid[lastDash+2:]
		if version, err := strconv.Atoi(suffix); err == nil {
			return kid[:lastDash] + "-v" + strconv.Itoa(version+1)
		}
	}
	return kid + "-v2"
}
