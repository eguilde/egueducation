$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $repoRoot 'backend\.env'

function Decode-SecretValue {
  param(
    [Parameter(Mandatory = $true)]$Secret,
    [Parameter(Mandatory = $true)][string]$Key
  )

  return [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($Secret.data.$Key))
}

$dbSecret = kubectl -n education get secret egueducation-scoalabalotesti-db -o json | ConvertFrom-Json
$smsSecret = kubectl -n costesti get secret costesti-registratura-env -o json | ConvertFrom-Json

$databaseName = Decode-SecretValue $dbSecret 'database'
$databaseUsername = Decode-SecretValue $dbSecret 'username'
$databasePassword = Decode-SecretValue $dbSecret 'password'
$databaseSslMode = Decode-SecretValue $dbSecret 'sslmode'
$encodedPassword = [Uri]::EscapeDataString($databasePassword)
$smsApiToken = Decode-SecretValue $smsSecret 'SMSAPI_TOKEN'

$lines = @(
  'PORT=8080',
  'FRONTEND_ORIGIN=http://localhost:4200',
  'APP_ENV=development',
  '',
  'DATABASE_HOST=db.eguilde.cloud',
  'DATABASE_PORT=5432',
  "DATABASE_NAME=$databaseName",
  "DATABASE_USERNAME=$databaseUsername",
  "DATABASE_PASSWORD=$databasePassword",
  "DATABASE_SSLMODE=$databaseSslMode",
  "DATABASE_URL=postgres://${databaseUsername}:${encodedPassword}@db.eguilde.cloud:5432/${databaseName}?sslmode=${databaseSslMode}",
  '',
  'OIDC_ISSUER=http://localhost:8080/api/oidc',
  'OIDC_CLIENT_ID=egueducation-spa',
  'OIDC_DESKTOP_CLIENT_ID=egueducation-desktop',
  '',
  "SMSAPI_TOKEN=$smsApiToken",
  'SMS_SENDER_NAME=EguEducation',
  '',
  'ENABLE_SMS_OTP=true',
  'ENABLE_PASSKEYS=true',
  'ENABLE_EUDI_WALLET=true',
  'ENABLE_GDPR_FEATURES=true'
)

Set-Content -LiteralPath $envPath -Value $lines -Encoding UTF8
Write-Host 'backend/.env updated. Secret values were not printed.'
