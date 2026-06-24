$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$backendPath = Join-Path $repoRoot 'backend'
$envPath = Join-Path $backendPath '.env'

if (-not (Test-Path $envPath)) {
  throw "Missing backend/.env. Run scripts/update-local-env.ps1 first."
}

Get-Content -LiteralPath $envPath | ForEach-Object {
  $line = $_.Trim()
  if (-not $line -or $line.StartsWith('#')) {
    return
  }

  $separatorIndex = $line.IndexOf('=')
  if ($separatorIndex -lt 1) {
    return
  }

  $name = $line.Substring(0, $separatorIndex).Trim()
  $value = $line.Substring($separatorIndex + 1)
  [System.Environment]::SetEnvironmentVariable($name, $value, 'Process')
}

Set-Location $backendPath
go run ./cmd/server
