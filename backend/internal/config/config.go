package config

import (
	"bufio"
	"fmt"
	"net/url"
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
	Port                                    string
	HTTPRateLimitPerMinute                  int
	FrontendOrigin                          string
	FrontendOrigins                         []string
	Environment                             string
	DatabaseURL                             string
	MigrationDatabaseURL                    string
	BackendURL                              string
	CustomerName                            string
	CustomerDomain                          string
	TenantHostBaseDomain                    string
	OIDCIssuer                              string
	OIDCClientID                            string
	OIDCDesktopClient                       string
	OIDCAudience                            string
	OIDCDevMode                             bool
	AdditionalOIDCClients                   []OIDCClientConfig
	SMSAPIToken                             string
	SMSSenderName                           string
	EnablePasskeys                          bool
	EnableWallet                            bool
	EnableSMSOTP                            bool
	OTPHMACKey                              string
	EnableTestOTPFixture                    bool
	EnableProductionFixedOTPTestUser        bool
	TestOTPFixtureCode                      string
	TestOTPFixtureIdentifier                string
	TestOTPFixtureSubject                   string
	TestOTPFixtureTenantCode                string
	EnableGDPRFeatures                      bool
	ForceSecureCookies                      bool
	JWTKeyRotationDays                      int
	JWTKeyOverlapHours                      int
	Origin                                  string
	RPID                                    string
	WalletVerifierURL                       string
	WalletRPClientID                        string
	ArchiveStorageEndpoint                  string
	ArchiveStorageRegion                    string
	ArchiveStorageBucket                    string
	ArchiveStorageAccessKey                 string
	ArchiveStorageSecretKey                 string
	ArchiveStorageUsePathStyle              bool
	ArchiveStorageCreateBucket              bool
	ArchiveStorageRequireObjectLock         bool
	ArchiveTextractBucket                   string
	ArchiveTextractRegion                   string
	AzureDocumentIntelligenceEndpoint       string
	AzureDocumentIntelligenceKey            string
	AzureDocumentIntelligenceModel          string
	AzureDocumentIntelligenceAPIVersion     string
	AzureDocumentIntelligenceTimeoutMinutes int
	ArchiveWorkerEnabled                    bool
	ArchiveWorkerPollInterval               int
	ArchiveWorkerMaxAttempts                int
	ClamdAddress                            string
	SignatureVerifierURL                    string
	SignatureVerifierToken                  string
	SignatureVerifierTimeoutSeconds         int
}

func Load() Config {
	_ = godotenv.Load(".env", "backend/.env")
	loadPlainEnvFile(".env")
	loadPlainEnvFile("backend/.env")

	frontendOrigin := env("FRONTEND_ORIGIN", "http://localhost:4200")
	desktopClientID := env("OIDC_DESKTOP_CLIENT_ID", env("DESKTOP_CLIENT_ID", "egueducation-desktop"))

	return Config{
		Port:                                    env("PORT", "8080"),
		HTTPRateLimitPerMinute:                  boundedEnvInt("HTTP_RATE_LIMIT_PER_MINUTE", 0, 0, 100000),
		FrontendOrigin:                          frontendOrigin,
		FrontendOrigins:                         parseCSV(os.Getenv("FRONTEND_ORIGINS")),
		Environment:                             env("APP_ENV", env("NODE_ENV", "development")),
		DatabaseURL:                             databaseURL(),
		MigrationDatabaseURL:                    env("MIGRATION_DATABASE_URL", ""),
		BackendURL:                              env("BACKEND_URL", "http://localhost:8080"),
		CustomerName:                            env("CUSTOMER_NAME", "EguEducation"),
		CustomerDomain:                          env("CUSTOMER_DOMAIN", ""),
		TenantHostBaseDomain:                    env("TENANT_HOST_BASE_DOMAIN", "eguilde.cloud"),
		OIDCIssuer:                              env("OIDC_ISSUER", "http://localhost:8080/api/oidc"),
		OIDCClientID:                            env("OIDC_CLIENT_ID", "egueducation-spa"),
		OIDCDesktopClient:                       desktopClientID,
		OIDCAudience:                            env("OIDC_AUDIENCE", "egueducation-api"),
		OIDCDevMode:                             envBool("OIDC_DEV_MODE", false),
		SMSAPIToken:                             os.Getenv("SMSAPI_TOKEN"),
		SMSSenderName:                           env("SMS_SENDER_NAME", env("SMSAPI_SENDER", "EguEducation")),
		EnablePasskeys:                          envBool("ENABLE_PASSKEYS", true),
		EnableWallet:                            envBool("ENABLE_EUDI_WALLET", true),
		EnableSMSOTP:                            envBool("ENABLE_SMS_OTP", true),
		OTPHMACKey:                              strings.TrimSpace(os.Getenv("OTP_HMAC_KEY")),
		EnableTestOTPFixture:                    envBool("ENABLE_TEST_OTP_FIXTURE", false),
		EnableProductionFixedOTPTestUser:        envBool("ENABLE_PRODUCTION_FIXED_OTP_TEST_USER", false),
		TestOTPFixtureCode:                      preferredEnv("PRODUCTION_TEST_USER_OTP", "TEST_OTP_FIXTURE_CODE"),
		TestOTPFixtureIdentifier:                preferredEnv("PRODUCTION_TEST_USER_IDENTIFIER", "TEST_OTP_FIXTURE_IDENTIFIER"),
		TestOTPFixtureSubject:                   preferredEnv("PRODUCTION_TEST_USER_SUBJECT", "TEST_OTP_FIXTURE_SUBJECT"),
		TestOTPFixtureTenantCode:                preferredEnv("PRODUCTION_TEST_USER_TENANT", "TEST_OTP_FIXTURE_TENANT_CODE"),
		EnableGDPRFeatures:                      envBool("ENABLE_GDPR_FEATURES", true),
		ForceSecureCookies:                      envBool("FORCE_SECURE_COOKIES", false),
		JWTKeyRotationDays:                      envInt("JWT_KEY_ROTATION_DAYS", 90),
		JWTKeyOverlapHours:                      envInt("JWT_KEY_OVERLAP_HOURS", 24),
		Origin:                                  env("ORIGIN", env("BACKEND_URL", "http://localhost:8080")),
		RPID:                                    env("RP_ID", defaultRPID(frontendOrigin)),
		WalletVerifierURL:                       env("WALLET_VERIFIER_URL", ""),
		WalletRPClientID:                        env("WALLET_RP_CLIENT_ID", "egueducation"),
		ArchiveStorageEndpoint:                  env("ARCHIVE_STORAGE_ENDPOINT", ""),
		ArchiveStorageRegion:                    env("ARCHIVE_STORAGE_REGION", "us-east-1"),
		ArchiveStorageBucket:                    env("ARCHIVE_STORAGE_BUCKET", "archive-documents"),
		ArchiveStorageAccessKey:                 env("ARCHIVE_STORAGE_ACCESS_KEY", ""),
		ArchiveStorageSecretKey:                 env("ARCHIVE_STORAGE_SECRET_KEY", ""),
		ArchiveStorageUsePathStyle:              envBool("ARCHIVE_STORAGE_USE_PATH_STYLE", false),
		ArchiveStorageCreateBucket:              envBool("ARCHIVE_STORAGE_CREATE_BUCKET", false),
		ArchiveStorageRequireObjectLock:         envBool("ARCHIVE_STORAGE_REQUIRE_OBJECT_LOCK", false),
		ArchiveTextractBucket:                   env("ARCHIVE_TEXTRACT_BUCKET", ""),
		ArchiveTextractRegion:                   env("ARCHIVE_TEXTRACT_REGION", "us-east-1"),
		AzureDocumentIntelligenceEndpoint:       strings.TrimSpace(os.Getenv("AZURE_DOCUMENT_INTELLIGENCE_ENDPOINT")),
		AzureDocumentIntelligenceKey:            strings.TrimSpace(os.Getenv("AZURE_DOCUMENT_INTELLIGENCE_KEY")),
		AzureDocumentIntelligenceModel:          strings.TrimSpace(os.Getenv("AZURE_DOCUMENT_INTELLIGENCE_MODEL")),
		AzureDocumentIntelligenceAPIVersion:     strings.TrimSpace(os.Getenv("AZURE_DOCUMENT_INTELLIGENCE_API_VERSION")),
		AzureDocumentIntelligenceTimeoutMinutes: boundedEnvInt("AZURE_DOCUMENT_INTELLIGENCE_TIMEOUT_MINUTES", 60, 1, 180),
		ArchiveWorkerEnabled:                    envBool("ARCHIVE_WORKER_ENABLED", true),
		ArchiveWorkerPollInterval:               envInt("ARCHIVE_WORKER_POLL_INTERVAL_SECONDS", 5),
		ArchiveWorkerMaxAttempts:                boundedEnvInt("ARCHIVE_WORKER_MAX_ATTEMPTS", 5, 1, 20),
		ClamdAddress:                            env("CLAMD_ADDRESS", ""),
		SignatureVerifierURL:                    strings.TrimSpace(os.Getenv("SIGNATURE_VERIFIER_URL")),
		SignatureVerifierToken:                  strings.TrimSpace(os.Getenv("SIGNATURE_VERIFIER_TOKEN")),
		SignatureVerifierTimeoutSeconds:         boundedEnvInt("SIGNATURE_VERIFIER_TIMEOUT_SECONDS", 30, 1, 180),
	}
}

const testOTPHMACKey = "egueducation-test-otp-hmac-key-32-bytes-minimum"

// OTPHMACKeyValue returns the configured OTP hashing key.  Tests have a
// deterministic synthetic key so in-memory and integration fixtures never
// depend on a developer secret.  Deployments must always supply their own
// key; in particular production never falls back to this test value.
func (c Config) OTPHMACKeyValue() string {
	if key := strings.TrimSpace(c.OTPHMACKey); key != "" {
		return key
	}
	if strings.EqualFold(strings.TrimSpace(c.Environment), "test") {
		return testOTPHMACKey
	}
	return ""
}

// ValidateOTPStorage makes keyed OTP hashing a startup invariant.  The
// material is deliberately not included in an error, audit record, or log.
func (c Config) ValidateOTPStorage() error {
	key := c.OTPHMACKeyValue()
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("OTP_HMAC_KEY is required")
	}
	if len([]byte(key)) < 32 {
		return fmt.Errorf("OTP_HMAC_KEY must contain at least 32 bytes")
	}
	return nil
}

// AzureDocumentIntelligenceEnabled reports whether the complete, explicit Azure
// OCR configuration is present. Values are intentionally never logged.
func (c Config) AzureDocumentIntelligenceEnabled() bool {
	return strings.TrimSpace(c.AzureDocumentIntelligenceEndpoint) != "" &&
		strings.TrimSpace(c.AzureDocumentIntelligenceKey) != "" &&
		strings.TrimSpace(c.AzureDocumentIntelligenceModel) != "" &&
		strings.TrimSpace(c.AzureDocumentIntelligenceAPIVersion) != ""
}

// ValidateArchiveOCR fails closed whenever any Azure OCR setting is supplied
// without the complete set. This prevents an accidental fallback to another
// OCR provider in a deployment that intended to use Azure.
func (c Config) ValidateArchiveOCR() error {
	values := []string{
		strings.TrimSpace(c.AzureDocumentIntelligenceEndpoint),
		strings.TrimSpace(c.AzureDocumentIntelligenceKey),
		strings.TrimSpace(c.AzureDocumentIntelligenceModel),
		strings.TrimSpace(c.AzureDocumentIntelligenceAPIVersion),
	}
	configured := 0
	for _, value := range values {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(values) {
		return fmt.Errorf("incomplete Azure Document Intelligence configuration: set AZURE_DOCUMENT_INTELLIGENCE_ENDPOINT, AZURE_DOCUMENT_INTELLIGENCE_KEY, AZURE_DOCUMENT_INTELLIGENCE_MODEL, and AZURE_DOCUMENT_INTELLIGENCE_API_VERSION together")
	}
	return nil
}

// ValidateSignatureVerifier rejects half-configured remote trust validation.
// Keeping the token separate from the URL avoids accidental credential
// disclosure in logs and deployment manifests.
func (c Config) ValidateSignatureVerifier() error {
	endpoint := strings.TrimSpace(c.SignatureVerifierURL)
	token := strings.TrimSpace(c.SignatureVerifierToken)
	if endpoint == "" && token == "" {
		return nil
	}
	if endpoint == "" || token == "" {
		return fmt.Errorf("SIGNATURE_VERIFIER_URL and SIGNATURE_VERIFIER_TOKEN must be configured together")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(strings.EqualFold(c.Environment, "test") && parsed.Scheme == "http")) {
		return fmt.Errorf("SIGNATURE_VERIFIER_URL must be an absolute HTTPS URL")
	}
	return nil
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

// AutoMigrateOnStartup preserves the frictionless developer and test setup
// while making production schema changes an explicit, one-shot deployment
// operation. Production API instances must only consume an already validated
// schema using the restricted runtime database role.
func (c Config) AutoMigrateOnStartup() bool {
	return !c.IsProduction()
}

// MigrationURL returns the database connection dedicated to schema changes.
// A production migration must never silently fall back to the runtime URL:
// the Kubernetes pre-sync Job receives this value while the API Deployment
// intentionally does not.
func (c Config) MigrationURL() (string, error) {
	if migrationURL := strings.TrimSpace(c.MigrationDatabaseURL); migrationURL != "" {
		return migrationURL, nil
	}
	if c.IsProduction() {
		return "", fmt.Errorf("MIGRATION_DATABASE_URL is required for production schema migration")
	}
	if runtimeURL := strings.TrimSpace(c.DatabaseURL); runtimeURL != "" {
		return runtimeURL, nil
	}
	return "", fmt.Errorf("DATABASE_URL is required for non-production schema migration")
}

// TestOTPFixtureEnabled reports whether the deterministic OTP identity passed
// either the isolated loopback-test policy or the explicitly enabled,
// tenant-bound production fixed-OTP test-user policy.
func (c Config) TestOTPFixtureEnabled() bool {
	return c.ValidateTestOTPFixture() == nil && c.testOTPFixtureConfigured()
}

// ValidateTestOTPFixture rejects a test fixture configuration unless it can
// only be used by a loopback test server. NewService invokes this during
// startup, preventing a public deployment from silently accepting test OTPs.
func (c Config) ValidateTestOTPFixture() error {
	if !c.testOTPFixtureConfigured() {
		return nil
	}
	if !c.EnableTestOTPFixture && !(c.IsProduction() && c.EnableProductionFixedOTPTestUser) {
		return fmt.Errorf("test OTP fixture requires ENABLE_TEST_OTP_FIXTURE=true")
	}
	environment := strings.ToLower(strings.TrimSpace(c.Environment))
	switch environment {
	case "test":
		if !loopbackURL(c.FrontendOrigin) || !loopbackURL(c.BackendURL) || !loopbackURL(c.OIDCIssuer) {
			return fmt.Errorf("test OTP fixture requires loopback FRONTEND_ORIGIN, BACKEND_URL, and OIDC_ISSUER")
		}
	case "production", "prod":
		if !c.EnableProductionFixedOTPTestUser {
			return fmt.Errorf("production OTP fixture requires ENABLE_PRODUCTION_FIXED_OTP_TEST_USER=true")
		}
		if !securePublicURL(c.FrontendOrigin) || !securePublicURL(c.BackendURL) || !securePublicURL(c.OIDCIssuer) {
			return fmt.Errorf("production E2E canary requires HTTPS FRONTEND_ORIGIN, BACKEND_URL, and OIDC_ISSUER")
		}
		if !sameURLHost(c.FrontendOrigin, c.BackendURL) || !sameURLHost(c.FrontendOrigin, c.OIDCIssuer) {
			return fmt.Errorf("production E2E canary frontend, backend, and issuer must use the same host")
		}
		if strings.EqualFold(strings.TrimSpace(c.TestOTPFixtureIdentifier), "oidc.browser.fixture@example.test") || c.TestOTPFixtureCode == "173829" {
			return fmt.Errorf("production E2E canary must not reuse the public loopback fixture identity or OTP")
		}
		if !strings.EqualFold(strings.TrimSpace(c.TestOTPFixtureIdentifier), "test@eguilde.cloud") {
			return fmt.Errorf("production E2E fixture requires the dedicated test@eguilde.cloud identity")
		}
	default:
		return fmt.Errorf("test OTP fixture is allowed only in test or explicitly gated production")
	}
	if environment == "test" && !strings.HasSuffix(strings.ToLower(c.TestOTPFixtureIdentifier), "@example.test") {
		return fmt.Errorf("loopback test OTP fixture requires a synthetic example.test identifier")
	}
	if strings.TrimSpace(c.TestOTPFixtureSubject) == "" || strings.TrimSpace(c.TestOTPFixtureTenantCode) == "" {
		return fmt.Errorf("test OTP fixture requires a subject and tenant code")
	}
	if len(c.TestOTPFixtureCode) != 6 {
		return fmt.Errorf("test OTP fixture code must contain six digits")
	}
	for _, digit := range c.TestOTPFixtureCode {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("test OTP fixture code must contain six digits")
		}
	}
	return nil
}

func (c Config) testOTPFixtureConfigured() bool {
	return c.EnableTestOTPFixture || c.EnableProductionFixedOTPTestUser || c.TestOTPFixtureCode != "" || c.TestOTPFixtureIdentifier != "" || c.TestOTPFixtureSubject != "" || c.TestOTPFixtureTenantCode != ""
}

func (c Config) ProductionFixedOTPTestUserEnabled() bool {
	return c.IsProduction() && c.EnableProductionFixedOTPTestUser && c.TestOTPFixtureEnabled()
}

func loopbackURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func securePublicURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == ""
}

func sameURLHost(left, right string) bool {
	leftURL, leftErr := url.Parse(strings.TrimSpace(left))
	rightURL, rightErr := url.Parse(strings.TrimSpace(right))
	return leftErr == nil && rightErr == nil && strings.EqualFold(leftURL.Hostname(), rightURL.Hostname())
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func preferredEnv(primary, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(fallback))
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

func boundedEnvInt(key string, fallback, minimum, maximum int) int {
	value := envInt(key, fallback)
	if value < minimum || value > maximum {
		return fallback
	}
	return value
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

func databaseURL() string {
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}

	host := strings.TrimSpace(os.Getenv("DATABASE_HOST"))
	port := strings.TrimSpace(os.Getenv("DATABASE_PORT"))
	name := strings.TrimSpace(os.Getenv("DATABASE_NAME"))
	username := strings.TrimSpace(os.Getenv("DATABASE_USERNAME"))
	password := os.Getenv("DATABASE_PASSWORD")
	sslMode := strings.TrimSpace(os.Getenv("DATABASE_SSLMODE"))

	if host == "" || name == "" || username == "" {
		return "postgres://postgres:postgres@localhost:5432/egueducation?sslmode=disable"
	}
	if port == "" {
		port = "5432"
	}
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	query := url.Values{}
	query.Set("sslmode", sslMode)
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

func loadPlainEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		index := strings.IndexByte(line, '=')
		if index <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:index])
		if key == "" || os.Getenv(key) != "" {
			continue
		}

		value := strings.TrimSpace(line[index+1:])
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}
