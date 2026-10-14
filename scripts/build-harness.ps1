# build-harness.ps1 - Build the vendored DeepSeek Harness runtime for WhaleIsle MyGo.
#
# Upstream pin: reference/electron/vendor/harness-upstream.json
#   repo https://github.com/deepseek-ai/deepseek-harness.git
#   ref  dsh-v0.2.1-alpha.1
#   sha  5badb15009ae1756c3afe0ae0cef1faafc290ccc
#
# The sources in runtime/vendor/deepseek-harness are a verbatim copy of the
# pinned snapshot; this script NEVER modifies them. It only runs package
# managers (node_modules writes) and build scripts (lib/ and dist/ outputs).
# Re-running is safe: every step is idempotent (package managers converge,
# build scripts overwrite their own outputs).

param(
    [ValidateSet('custom', 'current')][string]$SourceMode = 'custom',
    [string]$CurrentSourceRoot = $env:WHALEISLE_CURRENT_ELECTRON_SOURCE,
    [switch]$BuildOnly
)
if ($SourceMode -eq 'current') {
    if (-not $CurrentSourceRoot) { throw 'CurrentSourceRoot must point to the fixed 90465b67 official source checkout.' }
    & (Join-Path $PSScriptRoot 'build-current-host.ps1') -SourceRoot $CurrentSourceRoot -BuildOnly:$BuildOnly
    return
}
if ($BuildOnly) { throw 'BuildOnly is supported by the isolated current source mode only.' }

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$VendorDir  = Join-Path $ProjectRoot 'runtime\vendor'
$HarnessDir = Join-Path $VendorDir 'deepseek-harness'
$SourceRoot = $env:WHALEISLE_ELECTRON_SOURCE
if (-not $SourceRoot) { $SourceRoot = Join-Path $ProjectRoot 'reference\electron' }
$PinFile = Join-Path $SourceRoot 'vendor\harness-upstream.json'
# Populate only missing source directories from the explicitly selected snapshot.
# Existing work trees remain build inputs; remove runtime/vendor to switch snapshots.
foreach ($name in @('deepseek-harness', 'dsh-usage-panel', 'dsh-im', 'dshbot', 'dsh-whale', 'dsh-remote', 'dsh-task-control', 'dsh-platform-session')) {
    $target = Join-Path $VendorDir $name
    if (-not (Test-Path $target)) {
        $source = Join-Path $SourceRoot "vendor\$name"
        if (-not (Test-Path (Join-Path $source 'package.json'))) { throw "Source package missing: $source" }
        New-Item -ItemType Directory -Force -Path $VendorDir | Out-Null
        Copy-Item $source $target -Recurse -Force
    }
}
# Node.js 24 toolchain location (add to PATH if not already present).
$NodeDir = 'C:\AI\tools\node-v24.21.0-win-x64'
if (Test-Path $NodeDir) { $env:PATH = "$NodeDir;$env:PATH" }

if (-not (Test-Path (Join-Path $HarnessDir 'package.json'))) {
    throw "Harness sources not found at $HarnessDir - copy them from reference\electron\vendor first (robocopy /E, see docs/plan.md M0)."
}

# The build embeds a short commit hash into client artifacts and reads it via
# `git rev-parse HEAD`. The vendored copy is not a git checkout, so we pass
# the pinned upstream sha through DSH_CLIENT_COMMIT_HASH - the documented
# explicit source for non-Git build environments (scripts/client-build-environment.ts).
$Pin = Get-Content $PinFile -Raw | ConvertFrom-Json
$env:DSH_CLIENT_COMMIT_HASH = $Pin.sha

# --- pnpm bootstrap ---------------------------------------------------------
# Install the exact pnpm version the harness pins in its package.json
# "packageManager" field, if the global pnpm does not already match.
$Pkg = Get-Content (Join-Path $HarnessDir 'package.json') -Raw | ConvertFrom-Json
$PinnedPnpm = $Pkg.packageManager -replace '^pnpm@', ''
$HavePnpm = $null
try { $HavePnpm = (pnpm --version) 2>$null } catch { }
if ($HavePnpm -ne $PinnedPnpm) {
    Write-Host "Installing pnpm $PinnedPnpm (harness packageManager pin)..."
    npm i -g "pnpm@$PinnedPnpm"
    if ($LASTEXITCODE -ne 0) { throw "npm i -g pnpm@$PinnedPnpm failed (exit $LASTEXITCODE)" }
}
Write-Host "pnpm $(pnpm --version) / node $(node --version)"

# --- harness install + official build ---------------------------------------
# Mirrors reference/electron/scripts/setup-harness.js:
#   pnpm install --frozen-lockfile
#   pnpm run build:official   (tsx scripts/build.ts --profile official:
#     native-system [no-op on Windows], build:lib:host, build:lib:client
#     [copies Ghostty wasm/font assets], build:web)
Push-Location $HarnessDir
try {
    Write-Host '==> pnpm install --frozen-lockfile'
    pnpm install --frozen-lockfile
    if ($LASTEXITCODE -ne 0) { throw "pnpm install --frozen-lockfile failed (exit $LASTEXITCODE)" }

    Write-Host '==> pnpm run build:official'
    pnpm run build:official
    if ($LASTEXITCODE -ne 0) { throw "pnpm run build:official failed (exit $LASTEXITCODE)" }
} finally {
    Pop-Location
}

# --- Ghostty terminal assets -------------------------------------------------
# build:lib:client already copies them; assert the paths setup-harness.js checks.
$GhosttyAssets = @(
    'packages\client\ui-user-terminal\lib\assets\ghostty-vt.wasm',
    'packages\client\ui-user-terminal\lib\assets\ghostty-write-pty.wasm',
    'packages\client\ui-user-terminal\lib\assets\SymbolsNerdFontMono-Regular.woff2'
)
foreach ($a in $GhosttyAssets) {
    if (-not (Test-Path (Join-Path $HarnessDir $a))) { throw "Ghostty asset missing: $a" }
}

# --- Desktop plugin packages -------------------------------------------------
# setup-harness.js installs runtime npm deps (no dev/peer, no scripts) for
# dsh-usage-panel and dsh-im via `npm ci` (both ship package-lock.json).
# Idempotency: skip when a previous run already produced a node_modules tree.
$NpmCiPlugins = @('dsh-usage-panel', 'dsh-im')
foreach ($name in $NpmCiPlugins) {
    $dir = Join-Path $VendorDir $name
    if (-not (Test-Path (Join-Path $dir 'package.json'))) { throw "Plugin package.json missing: $name" }
    if (Test-Path (Join-Path $dir 'node_modules')) {
        Write-Host "== $name : node_modules present, skipping npm ci"
        continue
    }
    Write-Host "== $name : npm ci --omit=dev --omit=peer --ignore-scripts"
    Push-Location $dir
    try {
        npm ci --omit=dev --omit=peer --ignore-scripts --no-fund --no-audit
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed in $name (exit $LASTEXITCODE)" }
    } finally {
        Pop-Location
    }
}

# dshbot, dsh-whale, dsh-remote ship their runtime node_modules committed in
# the vendor tree (Git-tracked, per after-pack.js); the original app links
# them into profiles\web\node_modules via junctions (desktop-plugin-link.js)
# and never installs them. dsh-task-control and dsh-platform-session declare
# only peerDependencies, which the harness host provides at runtime - no
# node_modules needed. Verify instead of installing.
$CommittedPlugins = @('dshbot', 'dsh-whale', 'dsh-remote', 'dsh-task-control', 'dsh-platform-session')
foreach ($name in $CommittedPlugins) {
    $pkg = Join-Path $VendorDir "$name\package.json"
    if (-not (Test-Path $pkg)) { throw "Plugin package.json missing: $name" }
    $manifest = Get-Content $pkg -Raw | ConvertFrom-Json
    $hasRuntimeDeps = $manifest.dependencies -and ($manifest.dependencies.PSObject.Properties.Count -gt 0)
    if ($hasRuntimeDeps -and -not (Test-Path (Join-Path $VendorDir "$name\node_modules"))) {
        throw "$name declares runtime dependencies but its committed node_modules tree is missing - re-copy it from reference\electron\vendor."
    }
}

# --- Final artifact verification ----------------------------------------------
$Required = @(
    'apps\cli\lib\bin.js',
    'apps\web\dist\index.html'
)
foreach ($r in $Required) {
    if (-not (Test-Path (Join-Path $HarnessDir $r))) { throw "Build artifact missing: $r" }
}

Write-Host ''
Write-Host 'Harness runtime OK:'
Write-Host "  upstream pin : $($Pin.ref) @ $($Pin.sha)"
Write-Host "  CLI entry    : $HarnessDir\apps\cli\lib\bin.js"
Write-Host "  Web dist     : $HarnessDir\apps\web\dist\index.html"
Write-Host "  Ghostty      : $($GhosttyAssets.Count) assets present"

# The phone bridge uses original ChisaCode sources and its own plain-Node ABI.
& (Join-Path $PSScriptRoot 'build-mobile-runtime.ps1')
