# package.ps1 - Stage Whale Isle desktop artifacts for NSIS / MyGo packager (M11).
#
# Produces dist/whaleisle/ (exe + resources/) and dist/README.md with next-step notes.
# Does NOT download NSIS or build an installer when tools are missing.
#
# Usage (from repo root):  . .\scripts\env.ps1; .\scripts\package.ps1

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

. (Join-Path $PSScriptRoot 'env.ps1')

$DistRoot = Join-Path $ProjectRoot 'dist'
$Stage = Join-Path $DistRoot 'whaleisle'
$ExeOut = Join-Path $Stage 'whaleisle.exe'

if ($env:WHALEISLE_VERSION) {
    $Version = $env:WHALEISLE_VERSION
} else {
    $Version = '0.0.0-dev'
}

if (Test-Path $Stage) {
    Remove-Item -Recurse -Force $Stage
}
New-Item -ItemType Directory -Force -Path $Stage | Out-Null

Write-Host "Building whaleisle.exe (buildVersion=$Version) -> $ExeOut"
$ldflags = "-s -w -X main.buildVersion=$Version"
& go build -ldflags $ldflags -o $ExeOut ./cmd/whaleisle
if ($LASTEXITCODE -ne 0) { throw "go build failed (exit $LASTEXITCODE)" }

# Package the separate helper, but never run it during build/staging.
# Its presence does not enable scripts: the product requires an explicit
# per-document gesture followed by the runtime AppContainer/job gate.
$PreviewOS = (& go env GOOS).Trim()
$PreviewArch = (& go env GOARCH).Trim()
if ($PreviewOS -eq 'windows') {
    if ($PreviewArch -notin @('amd64', 'arm64')) { throw 'Interactive preview helper supports Windows amd64/arm64 only.' }
    $PreviewOut = Join-Path $Stage 'whaleisle-preview.exe'
    & go build -ldflags "-s -w -H windowsgui" -o $PreviewOut ./cmd/whaleisle-preview
    if ($LASTEXITCODE -ne 0) { throw "Preview helper build failed (exit $LASTEXITCODE)" }
}

$ResSrc = Join-Path $ProjectRoot 'resources'
$ResDst = Join-Path $Stage 'resources'
if (Test-Path $ResSrc) {
    Write-Host "Copying resources/ -> $ResDst"
    Copy-Item -Path $ResSrc -Destination $ResDst -Recurse -Force
} else {
    Write-Host "Note: resources/ not found at repo root; skipping copy"
}

# The native terminal uses MyGo's pinned DLL, not the Host's Ghostty WASM.
# Require verified local input before marking a package ready. This never fetches
# or executes the library; WHALEISLE_TERMINAL_LIBRARY can name the official asset.
$TerminalOS = (& go env GOOS).Trim()
$TerminalArch = (& go env GOARCH).Trim()
& go run ./tools/stage-terminal -target "$TerminalOS-$TerminalArch" -out $Stage
if ($LASTEXITCODE -ne 0) { throw 'Native terminal runtime staging failed; set WHALEISLE_TERMINAL_LIBRARY to the pinned official library.' }

# Every package includes the fixed THA4 assets, even shell-only development packages.
# Keep the input outside dist/whaleisle: that directory is recreated above.
$PetAssets = $env:WHALEISLE_PET_RENDER_ASSETS
if (-not $PetAssets) { $PetAssets = Join-Path $ProjectRoot 'runtime\pet-render-assets' }
& node (Join-Path $PSScriptRoot 'pet-render-assets.mjs') $PetAssets $Stage
if ($LASTEXITCODE -ne 0) { throw 'THA4 resource staging failed; set WHALEISLE_PET_RENDER_ASSETS to the fixed manifest resource directory.' }

# Host runtime is required unless explicitly producing a shell-only development package.
if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -ne '1') {
    # Refuse incomplete phone runtime before copying or writing .runtime-staged.
    $VendorSrc = $env:WHALEISLE_VENDOR
    if (-not $VendorSrc) { $VendorSrc = Join-Path $ProjectRoot 'runtime\vendor' }
    if (Test-Path (Join-Path $VendorSrc 'current-host-build.json')) {
        & node (Join-Path $PSScriptRoot 'current-host.mjs') verify $VendorSrc
        if ($LASTEXITCODE -ne 0) { throw 'Selected current Host artifacts failed verification' }
    }
    $MobileRoot = Join-Path $VendorSrc 'dshd-remote'
    $MobileNode = Join-Path $MobileRoot 'node\node.exe'
    if (-not (Test-Path $MobileNode)) { throw 'Mobile runtime missing. Run scripts/build-harness.ps1 first, or explicitly set WHALEISLE_PACKAGE_SKIP_RUNTIME=1 for a shell-only package.' }
    & $MobileNode (Join-Path $PSScriptRoot 'mobile-runtime.mjs') verify $MobileRoot
    if ($LASTEXITCODE -ne 0) { throw 'Mobile runtime incomplete or changed; rebuild before packaging.' }
    if (Test-Path $VendorSrc) {
        $VendorDst = Join-Path $Stage 'runtime\vendor'
        Write-Host "Copying runtime/vendor -> $VendorDst (may take a while)"
        New-Item -ItemType Directory -Force -Path (Split-Path $VendorDst) | Out-Null
        Copy-Item -Path $VendorSrc -Destination $VendorDst -Recurse -Force
    } else {
        throw 'Required runtime/vendor is missing.'
    }
    # An explicitly selected input must not silently fall back to another Node.
    $NodeSrc = $env:WHALEISLE_NODE_DIR
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
    $NodeDst = Join-Path $Stage 'runtime\node'
    Write-Host "Copying Node -> $NodeDst from $NodeSrc"
    New-Item -ItemType Directory -Force -Path (Split-Path $NodeDst) | Out-Null
    Copy-Item -Path $NodeSrc -Destination $NodeDst -Recurse -Force
    $StagedNode = Join-Path $NodeDst 'node.exe'
    if (-not (Test-Path $StagedNode -PathType Leaf) -or (Get-FileHash $StagedNode -Algorithm SHA256).Hash -ne $NodeHash) {
        throw 'Packaged Host node.exe is missing or changed.'
    }
    & $MobileNode (Join-Path $PSScriptRoot 'mobile-runtime.mjs') verify (Join-Path $Stage 'runtime\vendor\dshd-remote')
    if ($LASTEXITCODE -ne 0) { throw 'Packaged mobile runtime verification failed' }
    # Both desktop Node and the separately verified phone runtime are now staged.
    $Marker = Join-Path $Stage '.runtime-staged'
    Set-Content -Path $Marker -Value '1' -NoNewline
    Write-Host "Wrote runtime staging marker -> $Marker"
}

$Readme = Join-Path $DistRoot 'README.md'
$readmeText = @'
# Whale Isle - staged packaging output (M11)

This folder holds **staging artifacts** only. `scripts/package.ps1` does not download NSIS or emit an installer when packager tools are absent.

## Staged layout

- `whaleisle/whaleisle.exe` - MyGo desktop shell (link-time `main.buildVersion` via `go build -ldflags`)
- `whaleisle/whaleisle-preview.exe` - separate Windows AppContainer helper; scripts remain off until an explicit per-document action and successful runtime checks
- `whaleisle/ghostty-vt.dll` - verified native terminal ABI, with `.LICENSE` and `terminal-runtime.json` provenance
- `whaleisle/resources/` - copied from repo `resources/` when present (e.g. tray icon)
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
| Product name | Whale Isle |

'@
[System.IO.File]::WriteAllText($Readme, $readmeText)

Write-Host "Wrote $Readme"

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
        $NsisArguments = @("/DVERSION=$Version")
        if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') { $NsisArguments += '/DSHELL_ONLY=1' }
        & $MakensisPath @NsisArguments (Split-Path -Leaf $Nsi)
    } finally {
        Pop-Location
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "makensis failed (exit $LASTEXITCODE); staging artifacts kept"
    } elseif (Test-Path $Setup) {
        Write-Host "Installer: $Setup"
    }
} else {
    Write-Host "Note: makensis not on PATH; NSIS script at scripts/nsis/whaleisle.nsi (run after staging)"
}

Write-Host "package.ps1: done -> $Stage"
