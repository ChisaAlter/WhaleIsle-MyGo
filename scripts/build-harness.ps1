# build-harness.ps1 - Build the vendored DeepSeek Harness runtime for WhaleIsle MyGo.
#
# The shared desktop home contains rc.2 required events. The historical
# alpha.1 reference cannot read them, despite also claiming session format v4.
# Stage an explicit installed rc.2 or complete rc.4 runtime with -RuntimeSourceResources;
# never merge a newer snapshot into an existing older dependency graph.
#
# The sources in runtime/vendor/deepseek-harness are a verbatim copy of the
# pinned snapshot; this script NEVER modifies them. It only runs package
# managers (node_modules writes) and build scripts (lib/ and dist/ outputs).
# Re-running is safe: every step is idempotent (package managers converge,
# build scripts overwrite their own outputs).
#
# The composition a stage carries is the desktop app's, not upstream's: it
# vendors extra rows (mcp-servers-file, llm-vision-fallback, its tools plugin)
# on top of the pin, so every stage is verified against
# scripts/runtime-composition/contract.json and refused when it came from any
# other source.
#
# -SourceMode fork builds the Host from the recorded harness fork
# (scripts/harness-fork/pin.json) instead of copying an installed graph, so a
# harness bump is one merge plus one build. It needs no desktop source checkout;
# -DesktopResources still supplies the desktop plugins and mobile bridge, which
# are not harness source. See scripts/runtime-composition/README.md.

param(
    [string]$RuntimeSourceResources,
    [string]$RuntimeDestination,
    [string]$DesktopSourceRoot = $env:WHALEISLE_CURRENT_ELECTRON_SOURCE,
    [string]$DesktopSourceRevision = 'c9d805a7de0e0b7b80d5768cd30ab489275d1b3b',
    [string]$LegacyVendor,
    [string]$AcpSource,
    [ValidateSet('custom', 'current', 'fork')][string]$SourceMode = 'custom',
    [string]$HarnessForkTag,
    [string]$HarnessForkWork,
    [string]$DesktopResources,
    [string]$CurrentSourceRoot = $env:WHALEISLE_CURRENT_ELECTRON_SOURCE,
    [ValidateSet('90465b67ec16765631d54c195af19d9ff3c3a770', 'e9f584be6d3b491ce777bb6823f30bc2bbc6f195')]
    [string]$SourceCommit = '90465b67ec16765631d54c195af19d9ff3c3a770',
    [string]$NodeDir = $env:WHALEISLE_NODE_DIR,
    [switch]$UnifiedShortcuts,
    [switch]$BuildOnly,
    [switch]$WithoutPaidAPIs
)
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'env.ps1')

# -SourceMode fork: the harness comes from the recorded fork source tree
# (scripts/harness-fork/pin.json) instead of a copy of an installed graph, so a
# harness bump is one merge there, not a new staging round. Desktop plugins and
# the mobile bridge are not harness source; -DesktopResources names the installed
# desktop they still come from.
function Invoke-HarnessForkStage([string]$ForkWork, [string]$Tag, [string]$Destination, [string]$NodeExe, [string]$Resources, [string]$Legacy) {
    $ForkScripts = Join-Path $PSScriptRoot 'harness-fork'
    $ProjectRoot = Split-Path -Parent $PSScriptRoot
    if (-not $ForkWork) { $ForkWork = Join-Path $ProjectRoot '.harness-fork' }
    $Network = @()
    if ($env:HTTPS_PROXY) { $Network = @('--proxy', $env:HTTPS_PROXY) }
    & node (Join-Path $ForkScripts 'fork.mjs') materialize --work $ForkWork @Network
    if ($LASTEXITCODE -ne 0) { throw 'Replaying the harness fork failed. The pin and its recorded resolutions are the only inputs, so this means the record drifted; do not stage over it.' }
    $Tree = Join-Path $ForkWork 'tree'
    $Pin = Get-Content -LiteralPath (Join-Path $ForkScripts 'pin.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    if (-not $Tag) { $Tag = $Pin.bumps[-1].ref }
    $Entry = $Pin.upstream.tags | Where-Object { $_.ref -eq $Tag } | Select-Object -First 1
    if (-not $Entry) { throw "Tag $Tag is not in scripts/harness-fork/pin.json. Add it with: node scripts/harness-fork/fork.mjs bump $Tag" }
    & node (Join-Path $ForkScripts 'build.mjs') $Tree --commit $Entry.sha --work $ForkWork
    if ($LASTEXITCODE -ne 0) { throw 'build:official failed on the fork source tree; no runtime was staged.' }
    $Version = (Get-Content -LiteralPath (Join-Path $Tree 'apps\cli\package.json') -Raw -Encoding UTF8 | ConvertFrom-Json).version
    if ($Entry.ref -ne ('dsh-v' + $Version)) { throw "The built tree reports $Version but the pin stages $($Entry.ref); the fork records another harness version." }
    if (-not $Destination) { $Destination = Join-Path $ProjectRoot ('runtime-' + ($Version -replace '\.', '')) }
    $Stage = @((Join-Path $ForkScripts 'stage.mjs'), 'stage', $Tree, '--dest', $Destination, '--work', $ForkWork)
    if ($NodeExe) { $Stage += @('--node', $NodeExe) }
    if ($Resources) { $Stage += @('--desktop-resources', $Resources) }
    & node @Stage
    if ($LASTEXITCODE -ne 0) { throw 'Staging the fork build failed; the staged directory was left for diagnosis.' }
    $thirdParty = @((Join-Path $PSScriptRoot 'stage-third-party-plugins.mjs'), (Join-Path $Destination 'vendor'))
    if ($WithoutPaidAPIs) { $thirdParty += '--without=dsh-deepseek-tools' }
    & node @thirdParty
    if ($LASTEXITCODE -ne 0) { throw 'Third-party desktop plugin staging failed; staged files retained.' }
    if ($Resources) {
        $Complete = @((Join-Path $ForkScripts 'stage.mjs'), 'complete', $Destination, '--desktop-resources', $Resources)
        if ($Legacy) { $Complete += @('--legacy-vendor', $Legacy) }
        & node @Complete
        if ($LASTEXITCODE -ne 0) { throw 'Completing the stage with the installed desktop plugins failed; the stage keeps the files it already verified.' }
    }
    Write-Host "Staged $Destination from the harness fork ($($Entry.ref))."
}

if ($SourceMode -eq 'fork') {
    if ($RuntimeSourceResources) { throw 'fork stages the harness from source; pass the installed desktop plugins with -DesktopResources instead of -RuntimeSourceResources.' }
    Invoke-HarnessForkStage -ForkWork $HarnessForkWork -Tag $HarnessForkTag -Destination $RuntimeDestination -NodeExe $(if ($NodeDir) { Join-Path $NodeDir 'node.exe' } else { $null }) -Resources $DesktopResources -Legacy $LegacyVendor
    return
}
if ($RuntimeSourceResources) {
    if ($SourceMode -ne 'custom') { throw 'RuntimeSourceResources and current source build are separate inputs.' }
    $StageNode = Join-Path $RuntimeSourceResources 'runtime\primary-runtime\dependencies\node\bin\node.exe'
    if (-not (Test-Path -LiteralPath $StageNode -PathType Leaf)) { throw 'The selected original resources directory has no bundled Node runtime.' }
    $InstalledHost = Get-Content -LiteralPath (Join-Path $RuntimeSourceResources 'vendor\deepseek-harness\package.json') -Raw -Encoding UTF8 | ConvertFrom-Json
    if (-not $RuntimeDestination) {
        $DestinationName = if ($InstalledHost.version -match '^0\.2\.2-rc\.\d+$') { 'runtime-' + ($InstalledHost.version -replace '\.', '') } else { 'runtime' }
        $RuntimeDestination = Join-Path (Split-Path -Parent $PSScriptRoot) $DestinationName
    }
    if ($InstalledHost.version -eq '0.2.1-rc.4') {
        if (-not $DesktopSourceRoot) { throw 'DesktopSourceRoot is required for rc.4 main task/IM plugins.' }
        if (-not $LegacyVendor) { $LegacyVendor = Join-Path (Split-Path -Parent $PSScriptRoot) 'runtime\vendor' }
        & $StageNode (Join-Path $PSScriptRoot 'session-runtime-complete.mjs') stage $RuntimeSourceResources $RuntimeDestination $DesktopSourceRoot $DesktopSourceRevision $LegacyVendor
    } elseif ($InstalledHost.version -match '^0\.2\.2-rc\.\d+$') {
        if (-not $LegacyVendor) { $LegacyVendor = Join-Path (Split-Path -Parent $PSScriptRoot) 'runtime\vendor' }
        & $StageNode (Join-Path $PSScriptRoot 'session-runtime-complete.mjs') stage-installed $RuntimeSourceResources $RuntimeDestination $LegacyVendor
    } else {
        & $StageNode (Join-Path $PSScriptRoot 'session-runtime.mjs') stage $RuntimeSourceResources $RuntimeDestination
    }
    if ($LASTEXITCODE -ne 0) { throw 'Compatible Host staging failed; the original runtime and session files were not changed.' }
    Assert-HostComposition $RuntimeDestination $StageNode
    # The staged tree has to carry the tracked third-party plugins too: the mount
    # row resolves by package name, so a stage without them boots and silently
    # lacks the tools. -WithoutPaidAPIs names the build that wants exactly that.
    $thirdParty = @((Join-Path $PSScriptRoot 'stage-third-party-plugins.mjs'), (Join-Path $RuntimeDestination 'vendor'))
    if ($WithoutPaidAPIs) { $thirdParty += '--without=dsh-deepseek-tools' }
    & $StageNode @thirdParty
    if ($LASTEXITCODE -ne 0) { throw 'Third-party desktop plugin staging failed; staged files retained.' }
    if ($AcpSource) {
        & $StageNode (Join-Path $PSScriptRoot 'session-runtime-acp.mjs') $AcpSource $RuntimeDestination
        if ($LASTEXITCODE -ne 0) { throw 'ACP package snapshot failed; staged files retained.' }
    }
    return
}
if ($SourceMode -eq 'current') {
    if (-not $CurrentSourceRoot) { throw 'CurrentSourceRoot must point to the selected fixed official source checkout.' }
    & (Join-Path $PSScriptRoot 'build-current-host.ps1') -SourceRoot $CurrentSourceRoot -SourceCommit $SourceCommit -BuildOnly:$BuildOnly -UnifiedShortcuts:$UnifiedShortcuts
    return
}
if ($BuildOnly) { throw 'BuildOnly is supported by the isolated current source mode only.' }

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$VendorDir  = Join-Path $ProjectRoot 'runtime\vendor'
$HarnessDir = Join-Path $VendorDir 'deepseek-harness'
$SourceRoot = $env:WHALEISLE_ELECTRON_SOURCE
if (-not $SourceRoot) { $SourceRoot = Join-Path $ProjectRoot 'reference\electron' }
$PinFile = Join-Path $SourceRoot 'vendor\harness-upstream.json'
$Pin = Get-Content -LiteralPath $PinFile -Raw | ConvertFrom-Json
if ($Pin.npm -ne '0.2.1-rc.2' -or $Pin.sha -ne '687f282e8625c6b8d11ff58211a90524be2e4317') {
    throw 'This source snapshot predates the shared desktop session schema. Select the original rc.2 resources with -RuntimeSourceResources and a fresh -RuntimeDestination; do not reuse the alpha.1 runtime.'
}
$ExistingCLI = Join-Path $HarnessDir 'apps\cli\package.json'
if (Test-Path -LiteralPath $ExistingCLI) {
    $ExistingVersion = (Get-Content -LiteralPath $ExistingCLI -Raw | ConvertFrom-Json).version
    if ($ExistingVersion -ne $Pin.npm) { throw 'Existing runtime differs from the selected source. Use a fresh runtime destination instead of merging dependency graphs.' }
}
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
# Tracked third-party desktop plugins this stage must carry. A source that cannot
# be read stops the stage: a runtime that silently lacks the plugin boots fine and
# hides the gap. Builds that deliberately ship without paid application APIs pass
# -WithoutPaidAPIs instead of dropping the copy by hand.
$thirdParty = @((Join-Path $PSScriptRoot 'stage-third-party-plugins.mjs'), $VendorDir)
if ($WithoutPaidAPIs) { $thirdParty += '--without=dsh-deepseek-tools' }
& node @thirdParty
if ($LASTEXITCODE -ne 0) { throw 'Third-party desktop plugin staging failed.' }
# Node.js 24 toolchain location (prefer explicit NodeDir / WHALEISLE_NODE_DIR, then runtime/node, then PATH).
$savedPath = $env:PATH
$savedCommitHash = $env:DSH_CLIENT_COMMIT_HASH
$tempParent = [IO.Path]::GetFullPath([IO.Path]::GetTempPath()).TrimEnd('\', '/')
$pnpmToolsPrefix = 'whaleisle-pnpm-'
$pnpmToolsDir = $null
try {
if (-not $NodeDir) {
    $repoNode = Join-Path $ProjectRoot 'runtime\node'
    if ((Test-Path -LiteralPath (Join-Path $repoNode 'node.exe') -PathType Leaf) -and (Test-Path -LiteralPath (Join-Path $repoNode 'npm.cmd') -PathType Leaf)) {
        $NodeDir = $repoNode
    } else {
        $pathNode = Get-Command node -ErrorAction SilentlyContinue
        if ($pathNode) { $NodeDir = Split-Path -Parent $pathNode.Source }
    }
}
if ($NodeDir -and (Test-Path -LiteralPath (Join-Path $NodeDir 'node.exe') -PathType Leaf)) {
    $env:PATH = "$NodeDir;$env:PATH"
}
$nodeCmd = Get-Command node -ErrorAction SilentlyContinue
$npmCmd = Get-Command npm -ErrorAction SilentlyContinue
if (-not $nodeCmd -or -not $npmCmd) {
    throw 'Node/npm toolchain missing; set WHALEISLE_NODE_DIR (or pass -NodeDir) containing node.exe and npm.cmd, or put them on PATH.'
}

if (-not (Test-Path (Join-Path $HarnessDir 'package.json'))) {
    throw "Harness sources not found at $HarnessDir - copy them from reference\electron\vendor first (robocopy /E, see docs/plan.md M0)."
}

# The build embeds a short commit hash into client artifacts and reads it via
# `git rev-parse HEAD`. The vendored copy is not a git checkout, so we pass
# the pinned upstream sha through DSH_CLIENT_COMMIT_HASH - the documented
# explicit source for non-Git build environments (scripts/client-build-environment.ts).
$env:DSH_CLIENT_COMMIT_HASH = $Pin.sha

# Explicit opt-in builds only; never silently adapt an existing runtime on startup.
if ($UnifiedShortcuts) {
    & node (Join-Path $PSScriptRoot 'current-host-shortcuts.mjs') $HarnessDir
    if ($LASTEXITCODE -ne 0) { throw 'Pinned shortcut provider adaptation failed.' }
}

# --- pnpm bootstrap ---------------------------------------------------------
# Resolve exact pnpm version pinned in package.json without modifying global packages.
$Pkg = Get-Content (Join-Path $HarnessDir 'package.json') -Raw | ConvertFrom-Json
$PinnedPnpm = $Pkg.packageManager -replace '^pnpm@', ''
$HavePnpm = $null
$pnpmCmd = Get-Command pnpm -ErrorAction SilentlyContinue
if ($pnpmCmd) {
    try { $HavePnpm = (& $pnpmCmd.Source --version) 2>$null } catch { }
}
$InvokePnpm = $null
if ($HavePnpm -eq $PinnedPnpm) {
    $InvokePnpm = { param([Parameter(ValueFromRemainingArguments=$true)][string[]]$PnpmArgs) & $pnpmCmd.Source @PnpmArgs }
} else {
    Write-Host "Bootstrapping isolated pnpm $PinnedPnpm (harness packageManager pin)..."
    $pnpmToolsDir = Join-Path $tempParent ($pnpmToolsPrefix + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $pnpmToolsDir | Out-Null
    & $npmCmd.Source install --prefix $pnpmToolsDir --ignore-scripts --no-audit --no-fund "pnpm@$PinnedPnpm"
    if ($LASTEXITCODE -ne 0) { throw "Isolated pnpm@$PinnedPnpm install failed (exit $LASTEXITCODE)" }
    $pnpmCjs = Join-Path $pnpmToolsDir 'node_modules\pnpm\bin\pnpm.cjs'
    if (-not (Test-Path -LiteralPath $pnpmCjs -PathType Leaf)) { $pnpmCjs = Join-Path $pnpmToolsDir 'node_modules\pnpm\bin\pnpm.mjs' }
    if (-not (Test-Path -LiteralPath $pnpmCjs -PathType Leaf)) { throw "Pinned pnpm entry not found under $pnpmToolsDir" }
    $InvokePnpm = { param([Parameter(ValueFromRemainingArguments=$true)][string[]]$PnpmArgs) & $nodeCmd.Source $pnpmCjs @PnpmArgs }
}
$activePnpmVersion = (& $InvokePnpm --version).Trim()
Write-Host "pnpm $activePnpmVersion / node $(& $nodeCmd.Source --version)"

# --- harness install + official build ---------------------------------------
# Mirrors reference/electron/scripts/setup-harness.js:
#   pnpm install --frozen-lockfile
#   pnpm run build:official   (tsx scripts/build.ts --profile official:
#     native-system [no-op on Windows], build:lib:host, build:lib:client
#     [copies Ghostty wasm/font assets], build:web)
Push-Location $HarnessDir
try {
    Write-Host '==> pnpm install --frozen-lockfile'
    & $InvokePnpm install --frozen-lockfile
    if ($LASTEXITCODE -ne 0) { throw "pnpm install --frozen-lockfile failed (exit $LASTEXITCODE)" }

    Write-Host '==> pnpm run build:official'
    & $InvokePnpm run build:official
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
# Verify lockfile hash instead of blindly skipping a potentially stale node_modules.
$NpmCiPlugins = @('dsh-usage-panel', 'dsh-im')
foreach ($name in $NpmCiPlugins) {
    $dir = Join-Path $VendorDir $name
    if (-not (Test-Path (Join-Path $dir 'package.json'))) { throw "Plugin package.json missing: $name" }
    $lockPath = Join-Path $dir 'package-lock.json'
    $nmDir = Join-Path $dir 'node_modules'
    $stampPath = Join-Path $nmDir '.whaleisle-lock.sha256'
    $lockHash = if (Test-Path -LiteralPath $lockPath -PathType Leaf) { (Get-FileHash -LiteralPath $lockPath -Algorithm SHA256).Hash.ToLowerInvariant() } else { '' }
    $stampHash = if (Test-Path -LiteralPath $stampPath -PathType Leaf) { (Get-Content -LiteralPath $stampPath -Raw).Trim().ToLowerInvariant() } else { '' }
    if ((Test-Path -LiteralPath $nmDir -PathType Container) -and $lockHash -and ($stampHash -eq $lockHash)) {
        Write-Host "== $name : node_modules matches package-lock.json ($lockHash), skipping npm ci"
        continue
    }
    if (Test-Path -LiteralPath $nmDir) {
        Write-Host "== $name : removing stale/unverified node_modules before npm ci"
        & $nodeCmd.Source -e "const fs=require('node:fs');fs.rmSync(process.argv[1],{recursive:true,force:true})" $nmDir
        if ($LASTEXITCODE -ne 0) { throw "Failed to remove stale node_modules in $name" }
    }
    Write-Host "== $name : npm ci --omit=dev --omit=peer --ignore-scripts"
    Push-Location $dir
    try {
        & $npmCmd.Source ci --omit=dev --omit=peer --ignore-scripts --no-fund --no-audit
        if ($LASTEXITCODE -ne 0) { throw "npm ci failed in $name (exit $LASTEXITCODE)" }
    } finally {
        Pop-Location
    }
    if ($lockHash -and (Test-Path -LiteralPath $nmDir -PathType Container)) {
        [IO.File]::WriteAllText($stampPath, $lockHash + "`n")
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
& (Join-Path $PSScriptRoot 'build-mobile-runtime.ps1') -NodeDir $NodeDir
} finally {
    $env:PATH = $savedPath
    if ($null -eq $savedCommitHash) { Remove-Item Env:DSH_CLIENT_COMMIT_HASH -ErrorAction SilentlyContinue } else { $env:DSH_CLIENT_COMMIT_HASH = $savedCommitHash }
    if ($pnpmToolsDir -and (Test-Path -LiteralPath $pnpmToolsDir)) {
        & node (Join-Path $PSScriptRoot 'build-storage.mjs') owned-dir $pnpmToolsDir $tempParent $pnpmToolsPrefix
    }
}
