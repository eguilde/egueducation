$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$backendPath = Join-Path $repoRoot 'backend'
$envPath = Join-Path $backendPath '.env'

if (-not (Test-Path $envPath)) {
  throw "Missing backend/.env. Run scripts/update-local-env.ps1 first."
}

Set-Location $backendPath
go run ./cmd/server
