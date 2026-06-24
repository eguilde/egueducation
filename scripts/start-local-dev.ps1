$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot

& (Join-Path $repoRoot 'scripts\update-local-env.ps1')

Start-Process powershell -WindowStyle Hidden -ArgumentList @(
  '-NoProfile',
  '-ExecutionPolicy', 'Bypass',
  '-File', (Join-Path $repoRoot 'scripts\start-backend.ps1')
)

Start-Process powershell -WindowStyle Hidden -ArgumentList @(
  '-NoProfile',
  '-ExecutionPolicy', 'Bypass',
  '-File', (Join-Path $repoRoot 'scripts\start-frontend.ps1')
)

Write-Host 'Local development processes were started for tenant Scoala Balotesti.'
Write-Host 'Frontend: http://localhost:4200'
Write-Host 'Backend:  http://localhost:8080'
