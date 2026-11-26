# package.ps1 - Stage Whale Isle desktop artifacts for NSIS / MyGo packager (M11).
#
# Produces dist/whaleisle/ (exe + resources/) and dist/README.md with next-step notes.
# Does NOT download NSIS or build an installer when tools are missing.
#
# Usage (from repo root):  . .\scripts\env.ps1; .\scripts\package.ps1
param(
    [string]$OutputRoot = $env:WHALEISLE_PACKAGE_OUTPUT,
    [string]$RuntimeRoot,
    [switch]$StageOnly,
    # Opt in: preserve the installer/metadata, reclaim only this run's payload.
    [switch]$CleanStage,
    [string]$GoExecutable = 'go',
    [string]$SigningScript = $env:WHALEISLE_SIGNING_SCRIPT,
    [string]$ExpectedPublisher = $env:WHALEISLE_UPDATE_PUBLISHER,
    [switch]$RequireSignature = (([string]$env:WHALEISLE_UPDATE_REQUIRE_SIGNATURE).Trim() -notin @('', '0', 'false', 'no'))
)

$ErrorActionPreference = 'Stop'
if ($StageOnly -and $CleanStage) { throw 'StageOnly retains its payload; use CleanStage only when building an installer.' }

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

. (Join-Path $PSScriptRoot 'env.ps1')

if (-not $OutputRoot) { $OutputRoot = Join-Path $ProjectRoot 'dist' }
$DistRoot = [System.IO.Path]::GetFullPath($OutputRoot)
$Stage = Join-Path $DistRoot 'whaleisle'
$ExeOut = Join-Path $Stage 'whaleisle.exe'

if ($env:WHALEISLE_VERSION) {
    $Version = $env:WHALEISLE_VERSION
} else {
    $Version = '0.0.0-dev'
}

if (Test-Path -LiteralPath $Stage) {
    throw "Package stage already exists; choose a fresh -OutputRoot rather than deleting existing artifacts: $Stage"
}
New-Item -ItemType Directory -Force -Path $Stage | Out-Null

Write-Host "Building whaleisle.exe (buildVersion=$Version) -> $ExeOut"
$BuildRelease = Join-Path $PSScriptRoot 'build-release.ps1'
& $BuildRelease -Package 'cmd/whaleisle' -Output $ExeOut -Version $Version -GoExecutable $GoExecutable

# Package the separate helper, but never run it during build/staging.
# Its presence does not enable scripts: the product requires an explicit
# per-document gesture followed by the runtime AppContainer/job gate.
$PreviewOS = (& $GoExecutable env GOOS).Trim()
$PreviewArch = (& $GoExecutable env GOARCH).Trim()
if ($PreviewOS -eq 'windows') {
    if ($PreviewArch -notin @('amd64', 'arm64')) { throw 'Interactive preview helper supports Windows amd64/arm64 only.' }
    $PreviewOut = Join-Path $Stage 'whaleisle-preview.exe'
    & $BuildRelease -Package 'cmd/whaleisle-preview' -Output $PreviewOut -Version $Version -GoExecutable $GoExecutable
    $IntegrationOut = Join-Path $Stage 'whaleisle-integration.exe'
    & $BuildRelease -Package 'cmd/whaleisle-integration' -Output $IntegrationOut -Version $Version -GoExecutable $GoExecutable
}

$ResSrc = Join-Path $ProjectRoot 'resources'
$ResDst = Join-Path $Stage 'resources'
if (Test-Path $ResSrc) {
    Write-Host "Copying resources/ -> $ResDst"
    Copy-Item -Path $ResSrc -Destination $ResDst -Recurse -Force
} else {
    Write-Host "Note: resources/ not found at repo root; skipping copy"
}

# The executable embeds the native skin fonts. Ship their notices alongside it,
# including in shell-only packages, from the same authoritative font sources.
$FontNotices = Join-Path $ResDst 'licenses\fonts'
New-Item -ItemType Directory -Force -Path $FontNotices | Out-Null
foreach ($Name in @('Inter-OFL.txt', 'SourceSerif4-OFL.txt', 'sources.json')) {
    Copy-Item -LiteralPath (Join-Path $ProjectRoot "internal\ui\theme\fonts\$Name") -Destination $FontNotices
}

# The native terminal uses MyGo's pinned DLL, not the Host's Ghostty WASM.
# Require verified local input before marking a package ready. This never fetches
# or executes the library; WHALEISLE_TERMINAL_LIBRARY can name the official asset.
$TerminalOS = (& $GoExecutable env GOOS).Trim()
$TerminalArch = (& $GoExecutable env GOARCH).Trim()
& $GoExecutable run ./tools/stage-terminal -target "$TerminalOS-$TerminalArch" -out $Stage
if ($LASTEXITCODE -ne 0) { throw 'Native terminal runtime staging failed; set WHALEISLE_TERMINAL_LIBRARY to the pinned official library.' }

# Route A (decision 3A): default excludes bundled pet-render-assets, bundle only when explicitly requested.
if ($env:WHALEISLE_INCLUDE_PET_ASSETS -eq '1') {
    $PetAssets = $env:WHALEISLE_PET_RENDER_ASSETS
    if (-not $PetAssets) { $PetAssets = Join-Path $ProjectRoot 'runtime\pet-render-assets' }
    if (Test-Path $PetAssets -PathType Container) {
        & node (Join-Path $PSScriptRoot 'pet-render-assets.mjs') $PetAssets $Stage
        if ($LASTEXITCODE -ne 0) { throw 'THA4 resource staging failed; run scripts/restore-pet-render-assets.ps1 or set WHALEISLE_PET_RENDER_ASSETS to the fixed manifest resource directory.' }
    } elseif ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') {
        Write-Host 'Note: verified THA4 assets unavailable; shell-only package skips runtime pet staging.'
    } else {
        throw 'THA4 resource staging failed; run scripts/restore-pet-render-assets.ps1 (pinned URLs/sizes/SHA256 from internal/desktop/pet/assets/render-manifest.json) or set WHALEISLE_PET_RENDER_ASSETS to the fixed manifest resource directory.'
    }
} else {
    Write-Host 'Note: Route A default excludes bundled pet-render-assets (set WHALEISLE_INCLUDE_PET_ASSETS=1 to bundle).'
}

# Host runtime is required unless explicitly producing a shell-only development package.
if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -ne '1') {
    # Refuse incomplete phone runtime before copying or writing .runtime-staged.
    $VendorSrc = $env:WHALEISLE_VENDOR
    if ($RuntimeRoot) { $VendorSrc = Join-Path ([System.IO.Path]::GetFullPath($RuntimeRoot)) 'vendor' }
    if (-not $VendorSrc) {
        # Newest staged 0.2.2 runtime wins; rc.2 stays the fallback so a machine
        # that has only the older stage keeps packaging.
        foreach ($name in @('runtime-0.2.2-rc5', 'runtime-0.2.2-rc2')) {
            $candidate = Join-Path $ProjectRoot $name
            if (Test-Path -LiteralPath (Join-Path $candidate 'vendor') -PathType Container) { $VendorSrc = Join-Path $candidate 'vendor'; break }
        }
        if (-not $VendorSrc) {
            throw 'Stage a 0.2.2 Host with build-harness.ps1 or explicitly select an alternate runtime with -RuntimeRoot.'
        }
    }
    if (Test-Path (Join-Path $VendorSrc 'current-host-build.json')) {
        & node (Join-Path $PSScriptRoot 'current-host.mjs') verify $VendorSrc
        if ($LASTEXITCODE -ne 0) { throw 'Selected current Host artifacts failed verification' }
    }
    # An executable rebuild does not update the adjacent Host. Require the
    # actual built reader's event vocabulary and preserved readiness gate.
    if (-not (Test-Path (Join-Path $VendorSrc 'current-host-build.json'))) {
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') verify $VendorSrc
        if ($LASTEXITCODE -ne 0) { throw 'Host session runtime is stale, mixed or unverified. Stage a complete supported runtime before packaging.' }
    } else {
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') inspect (Join-Path $VendorSrc 'deepseek-harness')
        if ($LASTEXITCODE -ne 0) { throw 'Selected current Host cannot read the shared desktop session schema.' }
    }
    & node (Join-Path $PSScriptRoot 'session-runtime-composition.mjs') verify (Split-Path -Parent $VendorSrc)
    if ($LASTEXITCODE -ne 0) { throw 'Staged Host composition does not match scripts/runtime-composition/contract.json. Restage from the 本体 source that contract names.' }
    $VendorDst = Join-Path $Stage 'runtime\vendor'
    $SessionInput = Test-Path -LiteralPath (Join-Path $VendorSrc 'host-session-runtime.json') -PathType Leaf
    if ($SessionInput) {
        $InputManifest = Get-Content -LiteralPath (Join-Path $VendorSrc 'host-session-runtime.json') -Raw -Encoding UTF8 | ConvertFrom-Json
        if ($InputManifest.cliVersion -eq '0.2.1-rc.4') {
            & node (Join-Path $PSScriptRoot 'session-runtime-complete.mjs') verify (Split-Path -Parent $VendorSrc)
            if ($LASTEXITCODE -ne 0) { throw 'rc.4 packaging requires the complete phone, retained dshbot and desktop main plugin assembly.' }
        }
        $releaseArgs = @((Split-Path -Parent $VendorSrc), (Join-Path $Stage 'runtime'))
        if ($env:WHALEISLE_INCLUDE_HEAVY_OPTIONAL -ne '1') {
            $releaseArgs += '--slim'
        }
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') release @releaseArgs
        if ($LASTEXITCODE -ne 0) { throw 'Complete runtime staging or actual mobile native ABI validation failed.' }
        $LinksHelper = Join-Path $Stage 'runtime\runtime-links.exe'
        & $GoExecutable build -ldflags '-s -w -H windowsgui' -o $LinksHelper ./scripts/session-runtime-links
        if ($LASTEXITCODE -ne 0) { throw 'Runtime dependency transaction helper build failed.' }
    } elseif (Test-Path $VendorSrc) {
        $VendorDst = Join-Path $Stage 'runtime\vendor'
        Write-Host "Copying runtime/vendor -> $VendorDst (may take a while)"
        New-Item -ItemType Directory -Force -Path (Split-Path $VendorDst) | Out-Null
        Copy-Item -Path $VendorSrc -Destination $VendorDst -Recurse -Force
        & node (Join-Path $PSScriptRoot 'windows-background-runtime.mjs') $VendorDst
        if ($LASTEXITCODE -ne 0) { throw 'Windows background process patch failed for the staged runtime.' }
    } else {
        throw 'Required runtime/vendor is missing.'
    }
    # A capability the package ships must arrive complete: its loader package plus
    # the engine that loader runs. Slimming one half leaves a plugin the Host mounts
    # and that fails the moment a user enables it.
    & node (Join-Path $PSScriptRoot 'session-runtime.mjs') capabilities (Join-Path $Stage 'runtime\vendor')
    if ($LASTEXITCODE -ne 0) { throw 'Packaged runtime cannot run a capability it ships. office-to-pdf needs @deepseek-ai/libreoffice-kit with its WASM or native engine; speech-to-text-sensevoice needs sherpa-onnx-node with its platform package.' }
    # An explicitly selected input must not silently fall back to another Node.
    $NodeSrc = $env:WHALEISLE_NODE_DIR
    $SessionManifestPath = Join-Path $VendorSrc 'host-session-runtime.json'
    $SessionManifest = $null
    if (Test-Path -LiteralPath $SessionManifestPath -PathType Leaf) {
        $SessionManifest = Get-Content -LiteralPath $SessionManifestPath -Raw | ConvertFrom-Json
        if (-not $NodeSrc) { $NodeSrc = Join-Path (Split-Path -Parent $VendorSrc) 'node' }
    }
    if (-not $NodeSrc) {
        $pathNode = Get-Command node -ErrorAction SilentlyContinue
        $NodeSrc = @(
            (Join-Path $ProjectRoot 'runtime-rc4\node'),
            (Join-Path $ProjectRoot 'runtime\node'),
            (Join-Path $ProjectRoot '.toolchain\node'),
            $(if ($pathNode) { Split-Path -Parent $pathNode.Source })
        ) | Where-Object { $_ -and (Test-Path (Join-Path $_ 'node.exe') -PathType Leaf) } | Select-Object -First 1
    }
    if (-not $NodeSrc -or -not (Test-Path (Join-Path $NodeSrc 'node.exe') -PathType Leaf)) {
        throw 'Host Node runtime missing; WHALEISLE_NODE_DIR must contain node.exe. Use WHALEISLE_PACKAGE_SKIP_RUNTIME=1 only for a shell-only development package.'
    }
    $NodeInput = Join-Path $NodeSrc 'node.exe'
    if ((Get-Item $NodeInput).Length -eq 0) { throw 'Host node.exe is empty.' }
    $NodeHash = (Get-FileHash $NodeInput -Algorithm SHA256).Hash
    if ($SessionManifest -and $NodeHash.ToLowerInvariant() -ne $SessionManifest.node.sha256) {
        throw 'Selected Node differs from the verified Host session runtime; use the matching bundled Node.'
    }
    $NodeDst = Join-Path $Stage 'runtime\node'
    if (-not $SessionInput) {
        Write-Host "Copying Node -> $NodeDst from $NodeSrc"
        New-Item -ItemType Directory -Force -Path (Split-Path $NodeDst) | Out-Null
        Copy-Item -Path $NodeSrc -Destination $NodeDst -Recurse -Force
    }
    $StagedNode = Join-Path $NodeDst 'node.exe'
    if (-not (Test-Path $StagedNode -PathType Leaf) -or (Get-FileHash $StagedNode -Algorithm SHA256).Hash -ne $NodeHash) {
        throw 'Packaged Host node.exe is missing or changed.'
    }
    if ($SessionManifest) {
        & $StagedNode (Join-Path $PSScriptRoot 'session-runtime.mjs') verify $VendorDst
        if ($LASTEXITCODE -ne 0) { throw 'Packaged Host session runtime failed final verification.' }
    }
    & $StagedNode (Join-Path $PSScriptRoot 'mobile-runtime.mjs') verify (Join-Path $Stage 'runtime\vendor\dshd-remote')
    if ($LASTEXITCODE -ne 0) { throw 'Packaged mobile runtime verification failed' }
    # The original installed distribution may contain no optional sample catalog.
    # An explicit selection or an existing catalog still receives full validation.
    $SamplesExplicit = [bool]$env:WHALEISLE_COMPONENT_SAMPLES
    $SamplesSrc = $env:WHALEISLE_COMPONENT_SAMPLES
    if (-not $SamplesSrc) { $SamplesSrc = Join-Path (Split-Path -Parent $VendorSrc) 'components\samples' }
    $SamplesDst = Join-Path $Stage 'runtime\components\samples'
    & $StagedNode (Join-Path $PSScriptRoot 'session-runtime-samples.mjs') $SamplesSrc $SamplesDst $SamplesExplicit.ToString().ToLowerInvariant()
    if ($LASTEXITCODE -ne 0) { throw 'Bundled component staging failed; the selected components/samples must contain valid real payloads.' }
    if ($SessionInput) {
        & $StagedNode (Join-Path $PSScriptRoot 'session-runtime-nsis.mjs') $Stage (Join-Path $DistRoot 'runtime-files.nsh')
        if ($LASTEXITCODE -ne 0) { throw 'Verified physical runtime include generation failed.' }
    }
    # Both desktop Node and the separately verified phone runtime are now staged.
    $Marker = Join-Path $Stage '.runtime-staged'
    Set-Content -Path $Marker -Value '1' -NoNewline
    Write-Host "Wrote runtime staging marker -> $Marker"
}

$Readme = Join-Path $DistRoot 'README.md'
$readmeText = @'
# 鲸屿 MyGo - staged packaging output (M11)

This folder holds **staging artifacts** only. `scripts/package.ps1` does not download NSIS or emit an installer when packager tools are absent.

## Staged layout

- `whaleisle/whaleisle.exe` - Whale Isle MyGo desktop shell (link-time `main.buildVersion` via `go build -ldflags`)
- `whaleisle/whaleisle-preview.exe` - separate Windows AppContainer helper; scripts remain off until an explicit per-document action and successful runtime checks
- `whaleisle/ghostty-vt.dll` - verified native terminal ABI, with `.LICENSE` and `terminal-runtime.json` provenance
- `whaleisle/resources/` - copied from repo `resources/` when present (e.g. tray icon)
- `whaleisle/resources/licenses/fonts/` - Inter and Source Serif 4 OFL notices and native font provenance
- `whaleisle/runtime/vendor/` - vendored harness when `runtime/vendor` exists (skip with `WHALEISLE_PACKAGE_SKIP_RUNTIME=1`; default `--slim` drops only the optional native LibreOffice engine and keeps every capability loader with a working engine, so `office-to-pdf` and `speech-to-text-sensevoice` stay runnable; set `WHALEISLE_INCLUDE_HEAVY_OPTIONAL=1` to retain the native engine too. Packaging refuses a stage whose loader/engine pair is incomplete.)
- `whaleisle/runtime/node/node.exe` - required Host Node from `WHALEISLE_NODE_DIR` or a known local Node distribution
- `whaleisle/pet-render-assets/` - THA4 pet render resources (default excluded under Route A decision 3A; set `WHALEISLE_INCLUDE_PET_ASSETS=1` to bundle)

## Next steps (NSIS / MyGo packager)

1. Build the installed rc.4 runtime with `scripts/build-harness.ps1 -RuntimeSourceResources <resources> -RuntimeDestination <fresh-runtime> -DesktopSourceRoot <desktop-source>`. This includes phone resources, retained dshbot and the pinned main plugins.
2. Select that exact runtime for packaging: `scripts/package.ps1 -RuntimeRoot <fresh-runtime> -OutputRoot <fresh-output>`. Existing rc.2 inputs remain available through the original defaults.
3. `mygo.json` `identifier` is **`ai.deepseek.harness.gui.mygo`** (independent side-by-side install under `%LOCALAPPDATA%\Programs\WhaleIsle-MyGo` while sharing `%APPDATA%\Deepseek-Harness-Desktop` userData).
4. NSIS script: `scripts/nsis/whaleisle.nsi` installs exe, resources, and optional `runtime/` trees.
5. Windows releases are unsigned by default. Finalize emits SHA-256 and blockmap metadata from the final installer bytes. Optional -RequireSignature, WHALEISLE_UPDATE_REQUIRE_SIGNATURE, an expected publisher or a supplied WHALEISLE_SIGNING_SCRIPT enforces the requested signature before metadata is emitted. Hashes verify file integrity; an unsigned package has no verified platform publisher.

## Application identifier

| Field | Release value |
| --- | --- |
| App ID / identifier | `ai.deepseek.harness.gui.mygo` |
| Product name | 鲸屿 MyGo |

'@
[System.IO.File]::WriteAllText($Readme, $readmeText)

Write-Host "Wrote $Readme"

if ($StageOnly) {
    Write-Host "package.ps1: stage ready; installer not requested -> $Stage"
    return
}

$Nsi = Join-Path $PSScriptRoot 'nsis\whaleisle.nsi'
$MakensisPath = $null
$MakensisCmd = Get-Command makensis -ErrorAction SilentlyContinue
if ($MakensisCmd) {
    $MakensisPath = $MakensisCmd.Source
} else {
    $NsisCandidates = @()
    if ($env:LOCALAPPDATA) { $NsisCandidates += (Join-Path $env:LOCALAPPDATA 'mygo\nsis-3.13\Bin\makensis.exe') }
    $NsisCandidates += (Join-Path $ProjectRoot '.toolchain\nsis\makensis.exe')
    $MakensisPath = $NsisCandidates | Where-Object { Test-Path -LiteralPath $_ -PathType Leaf } | Select-Object -First 1
}
if ($MakensisPath -and (Test-Path $Nsi)) {
    $Setup = Join-Path $DistRoot ("WhaleIsle-MyGo-Setup-{0}.exe" -f $Version)
    Write-Host "NSIS found: building $Setup"
    Push-Location (Join-Path $PSScriptRoot 'nsis')
    try {
        $NsisArguments = @("/DVERSION=$Version", "/DSTAGE_DIR=$Stage", "/DSETUP_OUT=$Setup")
        if ($SessionInput) { $NsisArguments += "/DRUNTIME_INCLUDE=$(Join-Path $DistRoot 'runtime-files.nsh')" }
        if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') { $NsisArguments += '/DSHELL_ONLY=1' }
        if ($env:WHALEISLE_INCLUDE_PET_ASSETS -eq '1') { $NsisArguments += '/DINCLUDE_PET_ASSETS=1' }
        & $MakensisPath @NsisArguments (Split-Path -Leaf $Nsi)
    } finally {
        Pop-Location
    }
    if ($LASTEXITCODE -ne 0) {
        throw "makensis failed (exit $LASTEXITCODE); staging artifacts kept at $Stage"
    }
    if (-not (Test-Path -LiteralPath $Setup -PathType Leaf) -or (Get-Item -LiteralPath $Setup).Length -le 0) {
        throw "makensis reported success but did not produce a non-empty installer: $Setup"
    }
    if (Test-Path $Setup) {
        Write-Host "Installer: $Setup"
        $developmentBuild = $Version -match '^0\.0\.0(?:-|$)'
        & (Join-Path $PSScriptRoot 'finalize-windows-release.ps1') -Installer $Setup -SigningScript $SigningScript -ExpectedPublisher $ExpectedPublisher -RequireSignature:$RequireSignature -Development:$developmentBuild
        if ($CleanStage) {
            & node (Join-Path $PSScriptRoot 'build-storage.mjs') package-stage $Stage $DistRoot
            if ($LASTEXITCODE -ne 0) { throw "Installer is ready at $Setup; package stage cleanup failed at $Stage" }
            Write-Host "Reclaimed package stage; installer and update metadata retained -> $Setup"
        }
    }
} else {
    Write-Host "Note: makensis not on PATH; NSIS script at scripts/nsis/whaleisle.nsi (run after staging)"
    if ($CleanStage) { throw "CleanStage requires a successful installer build; staging retained at $Stage" }
}

if ($CleanStage) { Write-Host "package.ps1: done -> $Setup" }
else { Write-Host "package.ps1: done -> $Stage" }
