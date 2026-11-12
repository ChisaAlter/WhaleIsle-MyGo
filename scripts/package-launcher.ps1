# Build the standalone native launcher only. Never launches it or an installer.
param([string]$Version = '0.0.0-dev')
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
. (Join-Path $PSScriptRoot 'env.ps1')
$Stage = Join-Path $Root 'dist/whaleisle-launcher'
New-Item -ItemType Directory -Force -Path $Stage | Out-Null
$Exe = Join-Path $Stage 'whaleisle-launcher.exe'
$ldflags = "-s -w -X main.buildVersion=$Version"
if ((& go env GOOS).Trim() -eq 'windows') {
    $ldflags += ' -H windowsgui'
    # Embed icon, manifest and version info; plain go build links .syso in the
    # main package. Without them the launcher shows no icon or file details.
    & go tool mygo winres -pkg cmd/whaleisle-launcher -arch ((& go env GOARCH).Trim()) -version $Version
    if ($LASTEXITCODE -ne 0) { throw 'Windows resource generation failed' }
}
& go build -ldflags $ldflags -o $Exe ./cmd/whaleisle-launcher
if ($LASTEXITCODE -ne 0) { throw 'Launcher build failed' }
Remove-Item -Path "$Root\cmd\*\mygo_windows_*.syso" -Force -ErrorAction SilentlyContinue
# Own executable and directory: no desktop vendor, credentials, or runtime copy.
Copy-Item (Join-Path $Root 'docs/standalone-launcher.md') (Join-Path $Stage 'README.md')
Write-Host "Standalone launcher staged: $Stage"
