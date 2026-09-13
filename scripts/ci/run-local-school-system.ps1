[CmdletBinding()]
param(
    [string]$Spec = 'e2e/system/teacher-portfolio-real-stack.spec.ts',
    [string]$StorageEndpoint = 'http://127.0.0.1:59000',
    [ValidatePattern('^egueducation_system_e2e(?:_[a-z0-9_]+)?$')]
    [string]$DatabaseName = 'egueducation_system_e2e',
    [ValidateRange(1024,65535)]
    [int]$DatabasePort = 55438,
    [ValidateRange(1024,65535)]
    [int]$FrontendPort = 4173,
    [ValidateRange(1024,65535)]
    [int]$TeacherFrontendPort = 4174,
    [ValidateRange(1024,65535)]
    [int]$AlternateFrontendPort = 4175
)

$ErrorActionPreference = 'Stop'
if (@($FrontendPort, $TeacherFrontendPort, $AlternateFrontendPort | Select-Object -Unique).Count -ne 3) { throw 'E2E frontend ports must be distinct.' }
$workspace = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
# Requires disposable PG17 and MinIO, forwarded only to these loopback ports.
# Never use an application database or archive bucket with this test runner.
$env:APP_ENV = 'test'
$env:GOCACHE = Join-Path $workspace '.cache/go-verifier-v2'
$env:PLAYWRIGHT_BROWSERS_PATH = Join-Path $workspace '.cache/playwright-browsers'
$env:PORT = '8080'
$env:DATABASE_URL = "postgres://egueducation_app:egueducation_app@127.0.0.1:${DatabasePort}/${DatabaseName}?sslmode=disable"
$env:MIGRATION_DATABASE_URL = "postgres://egueducation:egueducation@127.0.0.1:${DatabasePort}/${DatabaseName}?sslmode=disable"
$env:TEST_DATABASE_URL = $env:MIGRATION_DATABASE_URL
$env:E2E_PRIMARY_FRONTEND_PORT = [string]$FrontendPort
$env:E2E_TEACHER_FRONTEND_PORT = [string]$TeacherFrontendPort
$env:E2E_ALTERNATE_FRONTEND_PORT = [string]$AlternateFrontendPort
$env:FRONTEND_ORIGIN = "http://localhost:$FrontendPort"
$env:BACKEND_URL = 'http://127.0.0.1:8080'
$env:OIDC_ISSUER = "http://localhost:$FrontendPort/api/oidc"
$env:OIDC_CLIENT_ID = 'egueducation-spa'
$env:OIDC_AUDIENCE = 'egueducation-api'
$env:CUSTOMER_NAME = 'EguEducation Test Fixture'
$env:ENABLE_TEST_OTP_FIXTURE = 'true'
$env:ENABLE_PRODUCTION_FIXED_OTP_TEST_USER = 'false'
foreach ($name in @('PRODUCTION_TEST_USER_OTP','PRODUCTION_TEST_USER_IDENTIFIER','PRODUCTION_TEST_USER_SUBJECT','PRODUCTION_TEST_USER_TENANT')) {
    [Environment]::SetEnvironmentVariable($name, $null, 'Process')
}
$env:TEST_OTP_FIXTURE_CODE = '173829'
$env:TEST_OTP_FIXTURE_IDENTIFIER = 'oidc.browser.fixture@example.test'
$env:TEST_OTP_FIXTURE_SUBJECT = 'oidc-browser-fixture-subject'
$env:TEST_OTP_FIXTURE_TENANT_CODE = 'tenant-egueducation'
$env:SMSAPI_TOKEN = ''
$env:OTP_HMAC_KEY = 'isolated-system-e2e-only-not-a-production-secret'
$env:ENABLE_EUDI_WALLET = 'false'
$env:FORCE_SECURE_COOKIES = 'false'
$env:ARCHIVE_WORKER_ENABLED = 'true'
$env:ARCHIVE_WORKER_POLL_INTERVAL_SECONDS = '1'
$env:ARCHIVE_STORAGE_ENDPOINT = $StorageEndpoint
$env:ARCHIVE_STORAGE_REGION = 'us-east-1'
$env:ARCHIVE_STORAGE_BUCKET = 'registratura-system-e2e'
$env:ARCHIVE_STORAGE_ACCESS_KEY = 'minioadmin'
$env:ARCHIVE_STORAGE_SECRET_KEY = 'minioadmin123'
$env:ARCHIVE_STORAGE_USE_PATH_STYLE = 'true'
$env:ARCHIVE_STORAGE_CREATE_BUCKET = 'true'
$env:ARCHIVE_STORAGE_REQUIRE_OBJECT_LOCK = 'true'
$env:AZURE_DOCUMENT_INTELLIGENCE_ENDPOINT = 'http://127.0.0.1:9090'
$env:AZURE_DOCUMENT_INTELLIGENCE_KEY = 'system-e2e-azure-key'
$env:AZURE_DOCUMENT_INTELLIGENCE_MODEL = 'prebuilt-layout'
$env:AZURE_DOCUMENT_INTELLIGENCE_API_VERSION = '2024-11-30'
$env:CLAMD_ADDRESS = '127.0.0.1:3310'
$env:ARCHIVE_PDF_VALIDATOR_PATH = Join-Path $workspace '.cache/archive-pdf-validator.exe'
Push-Location (Join-Path $workspace 'backend')
try {
    & go build -o $env:ARCHIVE_PDF_VALIDATOR_PATH ./cmd/archive-pdf-validator
    if ($LASTEXITCODE -ne 0) { throw 'Archive PDF validator build failed.' }
} finally { Pop-Location }
Push-Location (Join-Path $workspace 'frontend-react')
try {
    # Parse the selected spec before starting any backend or browser servers.
    & npx playwright test '--config=playwright.system.config.ts' $Spec '--list'
    if ($LASTEXITCODE -ne 0) { throw 'E2E specification preflight failed.' }
    & npx playwright test '--config=playwright.system.config.ts' $Spec '--workers=1' '--reporter=line'
    $testExit = $LASTEXITCODE
} finally { Pop-Location }
exit $testExit
