# Build the standalone native launcher only. Never launches it or an installer.
param([string]$Version = '0.0.0-dev')
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
. (Join-Path $PSScriptRoot 'env.ps1')
$Stage = Join-Path $Root 'dist/whaleisle-launcher'
New-Item -ItemType Directory -Force -Path $Stage | Out-Null
$Exe = Join-Path $Stage 'whaleisle-launcher.exe'
& go build -ldflags "-s -w -X main.buildVersion=$Version" -o $Exe ./cmd/whaleisle-launcher
if ($LASTEXITCODE -ne 0) { throw 'Launcher build failed' }
# Own executable and directory: no desktop vendor, credentials, or runtime copy.
Copy-Item (Join-Path $Root 'docs/standalone-launcher.md') (Join-Path $Stage 'README.md')
Write-Host "Standalone launcher staged: $Stage"
