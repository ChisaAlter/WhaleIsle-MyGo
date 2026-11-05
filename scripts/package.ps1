# package.ps1 - Stage Whale Isle desktop artifacts for NSIS / MyGo packager (M11).
#
# Produces dist/whaleisle/ (exe + resources/) and dist/README.md with next-step notes.
# Does NOT download NSIS or build an installer when tools are missing.
#
# Usage (from repo root):  . .\scripts\env.ps1; .\scripts\package.ps1
param(
    [string]$OutputRoot = $env:WHALEISLE_PACKAGE_OUTPUT,
    [switch]$StageOnly,
    [string]$GoExecutable = 'go'
)

$ErrorActionPreference = 'Stop'

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
$ldflags = "-s -w -X main.buildVersion=$Version"
if ((& $GoExecutable env GOOS).Trim() -eq 'windows') { $ldflags += ' -H windowsgui' }
& $GoExecutable build -ldflags $ldflags -o $ExeOut ./cmd/whaleisle
if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }

# Package the separate helper, but never run it during build/staging.
# Its presence does not enable scripts: the product requires an explicit
# per-document gesture followed by the runtime AppContainer/job gate.
$PreviewOS = (& $GoExecutable env GOOS).Trim()
$PreviewArch = (& $GoExecutable env GOARCH).Trim()
if ($PreviewOS -eq 'windows') {
    if ($PreviewArch -notin @('amd64', 'arm64')) { throw 'Interactive preview helper supports Windows amd64/arm64 only.' }
    $PreviewOut = Join-Path $Stage 'whaleisle-preview.exe'
    & $GoExecutable build -ldflags "-s -w -H windowsgui" -o $PreviewOut ./cmd/whaleisle-preview
    if ($LASTEXITCODE -ne 0) { throw "Preview helper build failed (exit $LASTEXITCODE)" }
    $IntegrationOut = Join-Path $Stage 'whaleisle-integration.exe'
    & $GoExecutable build -ldflags '-s -w -H windowsgui' -o $IntegrationOut ./cmd/whaleisle-integration
    if ($LASTEXITCODE -ne 0) { throw "Integration cleanup helper build failed (exit $LASTEXITCODE)" }
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

# Every package includes the fixed THA4 assets, even shell-only development packages.
# Keep the input outside dist/whaleisle: that directory is recreated above.
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

# Host runtime is required unless explicitly producing a shell-only development package.
if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -ne '1') {
    # Refuse incomplete phone runtime before copying or writing .runtime-staged.
    $VendorSrc = $env:WHALEISLE_VENDOR
    if (-not $VendorSrc) { $VendorSrc = Join-Path $ProjectRoot 'runtime\vendor' }
    if (Test-Path (Join-Path $VendorSrc 'current-host-build.json')) {
        & node (Join-Path $PSScriptRoot 'current-host.mjs') verify $VendorSrc
        if ($LASTEXITCODE -ne 0) { throw 'Selected current Host artifacts failed verification' }
    }
    # An executable rebuild does not update the adjacent Host. Require the
    # actual built reader's rc.2 event vocabulary and preserved readiness gate.
    if (-not (Test-Path (Join-Path $VendorSrc 'current-host-build.json'))) {
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') verify $VendorSrc
        if ($LASTEXITCODE -ne 0) { throw 'Host session runtime is stale, mixed or unverified. Stage a complete rc.2 runtime before packaging.' }
    } else {
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') inspect (Join-Path $VendorSrc 'deepseek-harness')
        if ($LASTEXITCODE -ne 0) { throw 'Selected current Host cannot read the shared desktop session schema.' }
    }
    $VendorDst = Join-Path $Stage 'runtime\vendor'
    $SessionInput = Test-Path -LiteralPath (Join-Path $VendorSrc 'host-session-runtime.json') -PathType Leaf
    if ($SessionInput) {
        & node (Join-Path $PSScriptRoot 'session-runtime.mjs') release (Split-Path -Parent $VendorSrc) (Join-Path $Stage 'runtime')
        if ($LASTEXITCODE -ne 0) { throw 'Complete rc.2 runtime staging or actual mobile native ABI validation failed.' }
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
    # An explicitly selected input must not silently fall back to another Node.
    $NodeSrc = $env:WHALEISLE_NODE_DIR
    $SessionManifestPath = Join-Path $VendorSrc 'host-session-runtime.json'
    $SessionManifest = $null
    if (Test-Path -LiteralPath $SessionManifestPath -PathType Leaf) {
        $SessionManifest = Get-Content -LiteralPath $SessionManifestPath -Raw | ConvertFrom-Json
        if (-not $NodeSrc) { $NodeSrc = Join-Path (Split-Path -Parent $VendorSrc) 'node' }
    }
    if (-not $NodeSrc) {
        $NodeSrc = @(
            'C:\AI\tools\node-v24.21.0-win-x64',
            (Join-Path $ProjectRoot '.toolchain\node')
        ) | Where-Object { Test-Path (Join-Path $_ 'node.exe') -PathType Leaf } | Select-Object -First 1
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
    & node (Join-Path $PSScriptRoot 'mobile-runtime.mjs') verify (Join-Path $Stage 'runtime\vendor\dshd-remote')
    if ($LASTEXITCODE -ne 0) { throw 'Packaged mobile runtime verification failed' }
    # The original installed distribution may contain no optional sample catalog.
    # An explicit selection or an existing catalog still receives full validation.
    $SamplesExplicit = [bool]$env:WHALEISLE_COMPONENT_SAMPLES
    $SamplesSrc = $env:WHALEISLE_COMPONENT_SAMPLES
    if (-not $SamplesSrc) { $SamplesSrc = Join-Path (Split-Path -Parent $VendorSrc) 'components\samples' }
    $SamplesDst = Join-Path $Stage 'runtime\components\samples'
    & node (Join-Path $PSScriptRoot 'session-runtime-samples.mjs') $SamplesSrc $SamplesDst $SamplesExplicit.ToString().ToLowerInvariant()
    if ($LASTEXITCODE -ne 0) { throw 'Bundled component staging failed; the selected components/samples must contain valid real payloads.' }
    if ($SessionInput) {
        & node (Join-Path $PSScriptRoot 'session-runtime-nsis.mjs') $Stage (Join-Path $DistRoot 'runtime-files.nsh')
        if ($LASTEXITCODE -ne 0) { throw 'Verified physical runtime include generation failed.' }
    }
    # Both desktop Node and the separately verified phone runtime are now staged.
    $Marker = Join-Path $Stage '.runtime-staged'
    Set-Content -Path $Marker -Value '1' -NoNewline
    Write-Host "Wrote runtime staging marker -> $Marker"
}

$Readme = Join-Path $DistRoot 'README.md'
$readmeText = @'
# 鲸屿go - staged packaging output (M11)

This folder holds **staging artifacts** only. `scripts/package.ps1` does not download NSIS or emit an installer when packager tools are absent.

## Staged layout

- `whaleisle/whaleisle.exe` - MyGo desktop shell (link-time `main.buildVersion` via `go build -ldflags`)
- `whaleisle/whaleisle-preview.exe` - separate Windows AppContainer helper; scripts remain off until an explicit per-document action and successful runtime checks
- `whaleisle/ghostty-vt.dll` - verified native terminal ABI, with `.LICENSE` and `terminal-runtime.json` provenance
- `whaleisle/resources/` - copied from repo `resources/` when present (e.g. tray icon)
- `whaleisle/resources/licenses/fonts/` - Inter and Source Serif 4 OFL notices and native font provenance
- `whaleisle/runtime/vendor/` - vendored harness when `runtime/vendor` exists (skip with `WHALEISLE_PACKAGE_SKIP_RUNTIME=1`)
- `whaleisle/runtime/node/node.exe` - required Host Node from `WHALEISLE_NODE_DIR` or a known local Node distribution
- `whaleisle/pet-render-assets/` - required manifest-verified THA4 resources and provenance from `WHALEISLE_PET_RENDER_ASSETS` (default `runtime/pet-render-assets`)

## Next steps (NSIS / MyGo packager)

1. Build the vendored harness runtime: `scripts/build-harness.ps1` (`runtime/vendor/`).
2. `mygo.json` `identifier` is **`ai.deepseek.harness.gui`** (same appId as Electron for cover upgrades).
3. NSIS script: `scripts/nsis/whaleisle.nsi` installs exe, resources, and optional `runtime/` trees.
4. Sign the setup executable before release.

## Application identifier

| Field | Release value |
| --- | --- |
| App ID / identifier | `ai.deepseek.harness.gui` |
| Product name | 鲸屿go |

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
    $LocalMakensis = Join-Path $ProjectRoot '.toolchain\nsis\makensis.exe'
    if (Test-Path $LocalMakensis) { $MakensisPath = $LocalMakensis }
}
if ($MakensisPath -and (Test-Path $Nsi)) {
    $Setup = Join-Path $DistRoot ("WhaleIsle-Setup-{0}.exe" -f $Version)
    Write-Host "NSIS found: building $Setup"
    Push-Location (Join-Path $PSScriptRoot 'nsis')
    try {
        $NsisArguments = @("/DVERSION=$Version", "/DSTAGE_DIR=$Stage", "/DSETUP_OUT=$Setup")
        if ($SessionInput) { $NsisArguments += "/DRUNTIME_INCLUDE=$(Join-Path $DistRoot 'runtime-files.nsh')" }
        if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') { $NsisArguments += '/DSHELL_ONLY=1' }
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
		# Emit the same offset/size/SHA256 blockmap consumed by the MyGo
		# incremental updater. The setup remains the signed release artifact;
		# this sidecar is metadata only and is safe to regenerate after signing.
		$blockSize = 65536
		$setupBytes = [System.IO.File]::ReadAllBytes($Setup)
		$blocks = @()
		for ($offset = 0; $offset -lt $setupBytes.Length; $offset += $blockSize) {
			$size = [Math]::Min($blockSize, $setupBytes.Length - $offset)
			$hash = [System.Security.Cryptography.SHA256]::Create()
			try {
				$digest = $hash.ComputeHash($setupBytes, $offset, $size)
			} finally {
				$hash.Dispose()
			}
			$blocks += [ordered]@{
				offset = [int64]$offset
				size = [int64]$size
				sha256 = ([BitConverter]::ToString($digest) -replace '-', '').ToLowerInvariant()
			}
		}
		$blockmapPath = "${Setup}.blockmap.json"
		$blockmap = [ordered]@{ fileSize = [int64]$setupBytes.Length; blocks = $blocks } |
			ConvertTo-Json -Depth 4
		[System.IO.File]::WriteAllText($blockmapPath, $blockmap + "`n")
		$sumHash = [System.Security.Cryptography.SHA256]::Create()
		try {
			$sumBytes = $sumHash.ComputeHash([System.IO.File]::ReadAllBytes($Setup))
		} finally {
			$sumHash.Dispose()
		}
		$sum = ([BitConverter]::ToString($sumBytes) -replace '-', '').ToLowerInvariant()
		$sumPath = Join-Path $DistRoot 'SHA256SUMS.txt'
		[System.IO.File]::WriteAllText($sumPath, "$sum  $(Split-Path -Leaf $Setup)`n")
		Write-Host "Wrote blockmap: $blockmapPath"
		Write-Host "Wrote SHA256 manifest: $sumPath"
    }
} else {
    Write-Host "Note: makensis not on PATH; NSIS script at scripts/nsis/whaleisle.nsi (run after staging)"
}

Write-Host "package.ps1: done -> $Stage"
