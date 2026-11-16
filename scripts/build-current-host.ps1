# Optional pinned Host build. Default build-harness.ps1 remains the dirty/custom source.
param(
    [Parameter(Mandatory=$true)][string]$SourceRoot,
    [string]$WorkRoot,
    [ValidateSet('90465b67ec16765631d54c195af19d9ff3c3a770', 'e9f584be6d3b491ce777bb6823f30bc2bbc6f195')]
    [string]$SourceCommit = '90465b67ec16765631d54c195af19d9ff3c3a770',
    [string]$VendorTarget,
    [switch]$UnifiedShortcuts,
    [switch]$BuildOnly
)
$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot 'env.ps1')
$ownsWork = -not $PSBoundParameters.ContainsKey('WorkRoot')
$tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/')
$workPrefix = 'whaleisle-current-host-'
if (-not $WorkRoot) { $WorkRoot = Join-Path $tempParent ($workPrefix + [guid]::NewGuid().ToString('N')) }
$WorkRoot = [IO.Path]::GetFullPath($WorkRoot)
if (-not $VendorTarget) { $VendorTarget = Join-Path $ProjectRoot ('runtime\vendor-current-' + $SourceCommit.Substring(0, 8)) }
$Helper = Join-Path $PSScriptRoot 'current-host.mjs'
$PreviousCommit = $env:DSH_CLIENT_COMMIT_HASH
try {
    & node $Helper prepare $SourceRoot $WorkRoot $SourceCommit
    if ($LASTEXITCODE -ne 0) { throw 'Current Host source preparation failed' }
    if ($UnifiedShortcuts) {
        & node (Join-Path $PSScriptRoot 'current-host-shortcuts.mjs') (Join-Path $WorkRoot 'deepseek-harness')
        if ($LASTEXITCODE -ne 0) { throw 'Pinned shortcut provider adaptation failed.' }
    }
    # Install pinned pnpm locally, never alter the global toolchain.
    $Tools = Join-Path $WorkRoot 'build-tools'
    # Reuse npm's configured content cache; only the pinned tool installation is
    # task-local. Explicit npm_config_cache still supports cold-cache acceptance.
    & npm install --prefix $Tools --ignore-scripts --no-audit --no-fund --registry=https://registry.npmjs.org 'pnpm@11.8.0'
    if ($LASTEXITCODE -ne 0) { throw 'Pinned pnpm bootstrap failed' }
    $Pnpm = Join-Path $Tools 'node_modules\pnpm\bin\pnpm.cjs'
    if (-not (Test-Path $Pnpm)) { $Pnpm = Join-Path $Tools 'node_modules\pnpm\bin\pnpm.mjs' }
    $env:DSH_CLIENT_COMMIT_HASH = $SourceCommit
    Push-Location (Join-Path $WorkRoot 'deepseek-harness')
    try {
        & node $Pnpm install --frozen-lockfile --registry=https://registry.npmjs.org --fetch-retries=0
        if ($LASTEXITCODE -ne 0) { throw 'Pinned install failed. Do not bypass denied upstream dependencies or rewrite the lockfile.' }
        & node $Pnpm run build:official
        if ($LASTEXITCODE -ne 0) { throw 'Official Host build failed; no runtime was selected or staged' }
    } finally { Pop-Location }
    & node $Helper finalize $WorkRoot
    if ($LASTEXITCODE -ne 0) { throw 'Current Host artifact verification failed' }
    if (-not $BuildOnly) {
        $baseVendor = Join-Path $ProjectRoot 'runtime-rc4\vendor'
        if (-not (Test-Path $baseVendor)) { $baseVendor = Join-Path $ProjectRoot 'runtime\vendor' }
        & node $Helper stage $WorkRoot $baseVendor $VendorTarget
        if ($LASTEXITCODE -ne 0) { throw 'Current Host staging failed' }
        Write-Host "Staged $VendorTarget. Select explicitly with WHALEISLE_VENDOR; desktop bridges and overlay readiness are still required."
    } else {
        Write-Host "BuildOnly current Host output retained at $WorkRoot"
    }
} finally {
    if ($null -eq $PreviousCommit) { Remove-Item Env:DSH_CLIENT_COMMIT_HASH -ErrorAction SilentlyContinue } else { $env:DSH_CLIENT_COMMIT_HASH = $PreviousCommit }
    if ($ownsWork -and -not $BuildOnly) {
        $active = Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -ne $PID -and (($_.ExecutablePath + ' ' + $_.CommandLine).IndexOf($WorkRoot, [StringComparison]::OrdinalIgnoreCase) -ge 0) }
        if ($active) { Write-Warning "Owned build tree still referenced by another process; retained: $WorkRoot" }
        else { & node (Join-Path $PSScriptRoot 'build-storage.mjs') owned-dir $WorkRoot $tempParent $workPrefix }
    }
}
