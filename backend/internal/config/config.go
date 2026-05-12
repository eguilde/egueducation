package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	FrontendOrigin     string
	Environment        string
	DatabaseURL        string
	OIDCIssuer         string
	OIDCClientID       string
	OIDCDesktopClient  string
	SMSAPIToken        string
	SMSSenderName      string
	EnablePasskeys     bool
	EnableWallet       bool
	EnableSMSOTP       bool
	EnableGDPRFeatures bool
}

func Load() Config {
	_ = godotenv.Load(".env", "backend/.env")

	return Config{
		Port:               env("PORT", "8080"),
		FrontendOrigin:     env("FRONTEND_ORIGIN", "http://localhost:4200"),
		Environment:        env("APP_ENV", "development"),
		DatabaseURL:        env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/egueducation?sslmode=disable"),
		OIDCIssuer:         env("OIDC_ISSUER", "http://localhost:8080/api/oidc"),
		OIDCClientID:       env("OIDC_CLIENT_ID", "egueducation-spa"),
		OIDCDesktopClient:  env("OIDC_DESKTOP_CLIENT_ID", "egueducation-desktop"),
		SMSAPIToken:        os.Getenv("SMSAPI_TOKEN"),
		SMSSenderName:      env("SMS_SENDER_NAME", "EguEducation"),
		EnablePasskeys:     envBool("ENABLE_PASSKEYS", true),
		EnableWallet:       envBool("ENABLE_EUDI_WALLET", true),
		EnableSMSOTP:       envBool("ENABLE_SMS_OTP", true),
		EnableGDPRFeatures: envBool("ENABLE_GDPR_FEATURES", true),
	}
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
