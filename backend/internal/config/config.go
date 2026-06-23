package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type OIDCClientConfig struct {
	ClientID                string
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	Scope                   string
	TokenEndpointAuthMethod string
	ApplicationType         string
}

type Config struct {
	Port                  string
	FrontendOrigin        string
	FrontendOrigins       []string
	Environment           string
	DatabaseURL           string
	MigrationDatabaseURL  string
	BackendURL            string
	CustomerName          string
	CustomerDomain        string
	OIDCIssuer            string
	OIDCClientID          string
	OIDCDesktopClient     string
	OIDCAudience          string
	OIDCDevMode           bool
	AdditionalOIDCClients []OIDCClientConfig
	SMSAPIToken           string
	SMSSenderName         string
	EnablePasskeys        bool
	EnableWallet          bool
	EnableSMSOTP          bool
	EnableGDPRFeatures    bool
	ForceSecureCookies    bool
	JWTKeyRotationDays    int
	JWTKeyOverlapHours    int
	Origin                string
	RPID                  string
	WalletVerifierURL     string
	WalletRPClientID      string
}

func Load() Config {
	_ = godotenv.Load(".env", "backend/.env")

	frontendOrigin := env("FRONTEND_ORIGIN", "http://localhost:4200")
	desktopClientID := env("OIDC_DESKTOP_CLIENT_ID", env("DESKTOP_CLIENT_ID", "egueducation-desktop"))

	return Config{
		Port:                 env("PORT", "8080"),
		FrontendOrigin:       frontendOrigin,
		FrontendOrigins:      parseCSV(os.Getenv("FRONTEND_ORIGINS")),
		Environment:          env("APP_ENV", env("NODE_ENV", "development")),
		DatabaseURL:          env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/egueducation?sslmode=disable"),
		MigrationDatabaseURL: env("MIGRATION_DATABASE_URL", ""),
		BackendURL:           env("BACKEND_URL", "http://localhost:8080"),
		CustomerName:         env("CUSTOMER_NAME", "EguEducation"),
		CustomerDomain:       env("CUSTOMER_DOMAIN", ""),
		OIDCIssuer:           env("OIDC_ISSUER", "http://localhost:8080/api/oidc"),
		OIDCClientID:         env("OIDC_CLIENT_ID", "egueducation-spa"),
		OIDCDesktopClient:    desktopClientID,
		OIDCAudience:         env("OIDC_AUDIENCE", ""),
		OIDCDevMode:          envBool("OIDC_DEV_MODE", false),
		SMSAPIToken:          os.Getenv("SMSAPI_TOKEN"),
		SMSSenderName:        env("SMS_SENDER_NAME", env("SMSAPI_SENDER", "EguEducation")),
		EnablePasskeys:       envBool("ENABLE_PASSKEYS", true),
		EnableWallet:         envBool("ENABLE_EUDI_WALLET", true),
		EnableSMSOTP:         envBool("ENABLE_SMS_OTP", true),
		EnableGDPRFeatures:   envBool("ENABLE_GDPR_FEATURES", true),
		ForceSecureCookies:   envBool("FORCE_SECURE_COOKIES", false),
		JWTKeyRotationDays:   envInt("JWT_KEY_ROTATION_DAYS", 90),
		JWTKeyOverlapHours:   envInt("JWT_KEY_OVERLAP_HOURS", 24),
		Origin:               env("ORIGIN", env("BACKEND_URL", "http://localhost:8080")),
		RPID:                 env("RP_ID", defaultRPID(frontendOrigin)),
		WalletVerifierURL:    env("WALLET_VERIFIER_URL", ""),
		WalletRPClientID:     env("WALLET_RP_CLIENT_ID", "egueducation"),
	}
}

func (c Config) DesktopClientID() string {
	return c.OIDCDesktopClient
}

func (c Config) TLSEnabled() bool {
	if c.ForceSecureCookies {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.OIDCIssuer)), "https://")
}

func (c Config) IsProduction() bool {
	value := strings.ToLower(strings.TrimSpace(c.Environment))
	return value == "production" || value == "prod"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	case "0", "false", "FALSE", "no", "NO":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	var out int
	for _, char := range value {
		if char < '0' || char > '9' {
			return fallback
		}
		out = out*10 + int(char-'0')
	}
	if out <= 0 {
		return fallback
	}
	return out
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func defaultRPID(frontendOrigin string) string {
	value := strings.TrimSpace(frontendOrigin)
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	if index := strings.IndexByte(value, '/'); index >= 0 {
		value = value[:index]
	}
	if index := strings.IndexByte(value, ':'); index >= 0 {
		value = value[:index]
	}
	if value == "" {
		return "localhost"
	}
	return value
}
