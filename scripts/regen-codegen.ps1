# regen-codegen.ps1 - Regenerate internal/dsh/api and internal/dsh/events
# from the vendored harness build artifacts.
#
# Prerequisites: runtime/vendor/deepseek-harness built (see build-harness.ps1).
# Usage (from repo root):  . .\scripts\env.ps1; .\scripts\regen-codegen.ps1

param([string]$HarnessRoot, [switch]$ApiOnly)

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

$EnvScript = Join-Path $PSScriptRoot 'env.ps1'
. $EnvScript

if (-not $HarnessRoot) { $HarnessRoot = Join-Path $ProjectRoot 'runtime\vendor\deepseek-harness' }
$HarnessRoot = (Resolve-Path -LiteralPath $HarnessRoot).Path
$HarnessPackages = Join-Path $HarnessRoot 'packages'
$Schema = Join-Path $HarnessRoot 'docs\persistence-schema.json'
$ApiOut = Join-Path $ProjectRoot 'internal\dsh\api'
$EventsOut = Join-Path $ProjectRoot 'internal\dsh\events'

if (-not (Test-Path $HarnessPackages)) {
    throw "Harness packages not found at $HarnessPackages - run scripts\build-harness.ps1 first."
}
if (-not $ApiOnly -and -not (Test-Path $Schema)) {
    throw "persistence-schema.json not found at $Schema"
}

# Prefer packages under deepseek-harness/packages; dedupe by TYPERT package
# identity is handled inside gen-api (same package name keeps the first).
$typert = Get-ChildItem -Path $HarnessPackages -Recurse -Filter 'typert.remote-client.js' |
    Where-Object { $_.Directory.Name -eq 'lib' } |
    ForEach-Object { $_.Directory.Parent.FullName } |
    Sort-Object -Unique

if ($typert.Count -eq 0) {
    throw "No lib/typert.remote-client.js under $HarnessPackages"
}

$inArgs = foreach ($root in $typert) { '-in'; $root }

Write-Host "gen-api: $($typert.Count) package root(s) -> $ApiOut"
& go run ./tools/gen-api @inArgs -out $ApiOut
if ($LASTEXITCODE -ne 0) { throw "gen-api failed with exit $LASTEXITCODE" }

if (-not $ApiOnly) {
    Write-Host "gen-events: $Schema -> $EventsOut"
    & go run ./tools/gen-events -in $Schema -out $EventsOut
    if ($LASTEXITCODE -ne 0) { throw "gen-events failed with exit $LASTEXITCODE" }
}

Write-Host 'regen-codegen: done'
