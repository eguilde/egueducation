package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	productionE2ECanaryCookie = "__Secure-egueducation-e2e-canary"
	productionE2ECanaryTTL    = 15 * time.Minute
	productionE2ECanaryWindow = 5 * time.Minute
	productionE2ECanaryLimit  = 5
)

var productionE2ECanaryFailures = struct {
	sync.Mutex
	bySource map[string][]time.Time
}{bySource: make(map[string][]time.Time)}

// BeginProductionE2ECanary creates the short-lived second factor required
// before the production deployment will generate its deterministic OTP. The
// long random gate key is never stored in the browser cookie.
func (s *Service) BeginProductionE2ECanary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !s.cfg.ProductionE2ECanaryEnabled() {
		http.NotFound(w, r)
		return
	}
	if !productionE2ECanaryRequestOriginAllowed(r, &s.cfg, true) {
		s.auditProductionE2ECanary(r, "failure", "Production E2E canary activation rejected.")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	presented := ""
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(authorization, "Bearer ") {
		presented = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	}
	expected := s.cfg.ProductionE2ECanaryActivationKey
	if !constantTimeSecretEqual(presented, expected) {
		if !recordProductionE2ECanaryFailure(r, time.Now().UTC()) {
			s.auditProductionE2ECanary(r, "failure", "Production E2E canary activation rate limited.")
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		s.auditProductionE2ECanary(r, "failure", "Production E2E canary activation rejected.")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	clearProductionE2ECanaryFailures(r)

	token, err := issueProductionE2ECanaryToken(&s.cfg, time.Now().UTC())
	if err != nil {
		s.auditProductionE2ECanary(r, "failure", "Production E2E canary capability generation failed.")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     productionE2ECanaryCookie,
		Value:    token,
		Path:     "/api/oidc",
		MaxAge:   int(productionE2ECanaryTTL.Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	s.auditProductionE2ECanary(r, "success", "Production E2E canary capability issued.")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) auditProductionE2ECanary(r *http.Request, status, summary string) {
	if s == nil || s.db == nil || r == nil {
		return
	}
	branding := tenant.ResolveBranding(r.Host, s.cfg.CustomerName, tenant.DefaultInstitutionID(s.cfg.CustomerName))
	if branding.TenantCode != s.cfg.TestOTPFixtureTenantCode || branding.InstitutionID == "" {
		return
	}
	auditCtx, release, err := appdb.AcquireRequestConn(r.Context(), s.db.Raw(), appdb.SessionConfig{
		TenantID: branding.TenantCode, InstitutionID: branding.InstitutionID,
		InstitutionName: branding.Name, TenantSubdomain: branding.Subdomain,
		ActorSubject: "production-e2e-canary", IsSuperAdmin: false,
	})
	if err != nil {
		return
	}
	defer release()
	s.logAudit(auditCtx, "production-e2e-canary", "auth.e2e_canary.activate", "tenant", s.cfg.TestOTPFixtureTenantCode, status, summary, map[string]any{
		"tenant_code": s.cfg.TestOTPFixtureTenantCode,
	})
}

func constantTimeSecretEqual(presented, expected string) bool {
	presentedHash := sha256.Sum256([]byte(presented))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(presentedHash[:], expectedHash[:]) == 1
}

func productionE2ECanaryRequestOriginAllowed(r *http.Request, cfg *config.Config, requireOrigin bool) bool {
	if r == nil || cfg == nil {
		return false
	}
	expected, err := url.Parse(strings.TrimSpace(cfg.FrontendOrigin))
	if err != nil || expected.Scheme != "https" || expected.Hostname() == "" {
		return false
	}
	requestHost := strings.TrimSpace(r.Host)
	if host, _, splitErr := net.SplitHostPort(requestHost); splitErr == nil {
		requestHost = host
	}
	requestHost = strings.Trim(strings.ToLower(requestHost), "[]")
	if requestHost != strings.ToLower(expected.Hostname()) {
		return false
	}
	origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
	if origin == "" {
		return !requireOrigin
	}
	return origin == strings.TrimRight(strings.TrimSpace(cfg.FrontendOrigin), "/")
}

func productionE2ECanaryRateKey(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func recordProductionE2ECanaryFailure(r *http.Request, now time.Time) bool {
	key := productionE2ECanaryRateKey(r)
	cutoff := now.Add(-productionE2ECanaryWindow)
	productionE2ECanaryFailures.Lock()
	defer productionE2ECanaryFailures.Unlock()
	for source, failures := range productionE2ECanaryFailures.bySource {
		kept := failures[:0]
		for _, failure := range failures {
			if failure.After(cutoff) {
				kept = append(kept, failure)
			}
		}
		if len(kept) == 0 {
			delete(productionE2ECanaryFailures.bySource, source)
		} else {
			productionE2ECanaryFailures.bySource[source] = kept
		}
	}
	productionE2ECanaryFailures.bySource[key] = append(productionE2ECanaryFailures.bySource[key], now)
	return len(productionE2ECanaryFailures.bySource[key]) <= productionE2ECanaryLimit
}

func clearProductionE2ECanaryFailures(r *http.Request) {
	productionE2ECanaryFailures.Lock()
	delete(productionE2ECanaryFailures.bySource, productionE2ECanaryRateKey(r))
	productionE2ECanaryFailures.Unlock()
}

func issueProductionE2ECanaryToken(cfg *config.Config, now time.Time) (string, error) {
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate production E2E canary nonce: %w", err)
	}
	payload := strconv.FormatInt(now.Add(productionE2ECanaryTTL).Unix(), 10) + "." + base64.RawURLEncoding.EncodeToString(nonce)
	return payload + "." + signProductionE2ECanary(cfg, payload), nil
}

func validProductionE2ECanaryCookie(r *http.Request, cfg *config.Config) bool {
	_, ok := productionE2ECanaryClaimsFromRequest(r, cfg)
	return ok
}

type productionE2ECanaryClaims struct {
	JTI       string
	ExpiresAt time.Time
}

func productionE2ECanaryClaimsFromRequest(r *http.Request, cfg *config.Config) (productionE2ECanaryClaims, bool) {
	if r == nil || cfg == nil || !cfg.ProductionE2ECanaryEnabled() {
		return productionE2ECanaryClaims{}, false
	}
	if !productionE2ECanaryRequestOriginAllowed(r, cfg, false) {
		return productionE2ECanaryClaims{}, false
	}
	cookie, err := r.Cookie(productionE2ECanaryCookie)
	if err != nil {
		return productionE2ECanaryClaims{}, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return productionE2ECanaryClaims{}, false
	}
	expiresAt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().UTC().Unix() >= expiresAt || expiresAt > time.Now().UTC().Add(productionE2ECanaryTTL+time.Minute).Unix() {
		return productionE2ECanaryClaims{}, false
	}
	payload := parts[0] + "." + parts[1]
	expected := signProductionE2ECanary(cfg, payload)
	if len(parts[2]) != len(expected) || subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expected)) != 1 {
		return productionE2ECanaryClaims{}, false
	}
	return productionE2ECanaryClaims{JTI: parts[1], ExpiresAt: time.Unix(expiresAt, 0).UTC()}, true
}

func signProductionE2ECanary(cfg *config.Config, payload string) string {
	mac := hmac.New(sha256.New, []byte(cfg.ProductionE2ECanarySigningKey))
	_, _ = mac.Write([]byte("egueducation-production-e2e-canary-v1\x00" + payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func storeProductionE2ECanaryChallenge(ctx context.Context, db *pgxpool.Pool, claims productionE2ECanaryClaims, sessionID string, userID uuid.UUID, tenantCode string) error {
	if db == nil || claims.JTI == "" || sessionID == "" || userID == uuid.Nil || tenantCode == "" {
		return fmt.Errorf("invalid production E2E canary challenge")
	}
	_, err := db.Exec(ctx, `
		insert into oidc_production_e2e_challenges (jti, authn_session_id, user_id, tenant_code, expires_at)
		values ($1, $2, $3, $4, $5)
	`, claims.JTI, sessionID, userID, tenantCode, claims.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store production E2E canary challenge: %w", err)
	}
	return nil
}

func consumeProductionE2ECanaryChallenge(ctx context.Context, db *pgxpool.Pool, jti, sessionID string, userID uuid.UUID, tenantCode string) bool {
	if db == nil || jti == "" || sessionID == "" || userID == uuid.Nil || tenantCode == "" {
		return false
	}
	result, err := db.Exec(ctx, `
		update oidc_production_e2e_challenges
		set consumed_at=now()
		where jti=$1 and authn_session_id=$2 and user_id=$3 and tenant_code=$4
			and consumed_at is null and expires_at > now()
	`, jti, sessionID, userID, tenantCode)
	return err == nil && result.RowsAffected() == 1
}

func validProductionE2ECanaryChallenge(ctx context.Context, db *pgxpool.Pool, jti, sessionID string, userID uuid.UUID, tenantCode string) bool {
	if db == nil || jti == "" || sessionID == "" || userID == uuid.Nil || tenantCode == "" {
		return false
	}
	var valid bool
	err := db.QueryRow(ctx, `
		select exists(
			select 1 from oidc_production_e2e_challenges
			where jti=$1 and authn_session_id=$2 and user_id=$3 and tenant_code=$4
				and consumed_at is null and expires_at > now()
		)
	`, jti, sessionID, userID, tenantCode).Scan(&valid)
	return err == nil && valid
}

func deleteProductionE2ECanaryChallenge(ctx context.Context, db *pgxpool.Pool, jti, sessionID string) {
	if db == nil || jti == "" || sessionID == "" {
		return
	}
	_, _ = db.Exec(ctx, `delete from oidc_production_e2e_challenges where jti=$1 and authn_session_id=$2 and consumed_at is null`, jti, sessionID)
}

func clearProductionE2ECanaryCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     productionE2ECanaryCookie,
		Value:    "",
		Path:     "/api/oidc",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
