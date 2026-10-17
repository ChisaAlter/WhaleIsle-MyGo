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

$ResSrc = Join-Path $ProjectRoot 'resources'
$ResDst = Join-Path $Stage 'resources'
if (Test-Path $ResSrc) {
    Write-Host "Copying resources/ -> $ResDst"
    Copy-Item -Path $ResSrc -Destination $ResDst -Recurse -Force
} else {
    Write-Host "Note: resources/ not found at repo root; skipping copy"
}

# Optional vendored harness + Node for NSIS (skip when absent / WHALEISLE_PACKAGE_SKIP_RUNTIME=1).
if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -ne '1') {
    $VendorSrc = Join-Path $ProjectRoot 'runtime\vendor'
    if (Test-Path $VendorSrc) {
        $VendorDst = Join-Path $Stage 'runtime\vendor'
        Write-Host "Copying runtime/vendor -> $VendorDst (may take a while)"
        New-Item -ItemType Directory -Force -Path (Split-Path $VendorDst) | Out-Null
        Copy-Item -Path $VendorSrc -Destination $VendorDst -Recurse -Force
    } else {
        Write-Host "Note: runtime/vendor missing; NSIS will ship without harness runtime"
    }
    $NodeCandidates = @(
        $env:WHALEISLE_NODE_DIR,
        'C:\AI\tools\node-v24.21.0-win-x64',
        (Join-Path $ProjectRoot '.toolchain\node')
    ) | Where-Object { $_ -and (Test-Path $_) }
    if ($NodeCandidates.Count -gt 0) {
        $NodeSrc = $NodeCandidates[0]
        $NodeDst = Join-Path $Stage 'runtime\node'
        Write-Host "Copying Node -> $NodeDst from $NodeSrc"
        New-Item -ItemType Directory -Force -Path (Split-Path $NodeDst) | Out-Null
        Copy-Item -Path $NodeSrc -Destination $NodeDst -Recurse -Force
    } else {
        Write-Host "Note: Node directory not found; set WHALEISLE_NODE_DIR to include it in staging"
    }
    $RuntimeRoot = Join-Path $Stage 'runtime'
    if ((Test-Path $RuntimeRoot) -and (Get-ChildItem $RuntimeRoot -Recurse -File -ErrorAction SilentlyContinue | Select-Object -First 1)) {
        $Marker = Join-Path $Stage '.runtime-staged'
        Set-Content -Path $Marker -Value '1' -NoNewline
        Write-Host "Wrote runtime staging marker -> $Marker"
    }
}

$Readme = Join-Path $DistRoot 'README.md'
$readmeText = @'
# Whale Isle - staged packaging output (M11)

This folder holds **staging artifacts** only. `scripts/package.ps1` does not download NSIS or emit an installer when packager tools are absent.

## Staged layout

- `whaleisle/whaleisle.exe` - MyGo desktop shell (link-time `main.buildVersion` via `go build -ldflags`)
- `whaleisle/resources/` - copied from repo `resources/` when present (e.g. tray icon)
- `whaleisle/runtime/vendor/` - vendored harness when `runtime/vendor` exists (skip with `WHALEISLE_PACKAGE_SKIP_RUNTIME=1`)
- `whaleisle/runtime/node/` - Node tree when `WHALEISLE_NODE_DIR` or a known local Node path exists
- `WhaleIsle-Setup-<version>.exe.blockmap.json` - offset/size/SHA256 blocks for incremental update staging (when NSIS is available)
- `SHA256SUMS.txt` - SHA256 manifest for the generated installer (when NSIS is available)

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
        & $MakensisPath "/DVERSION=$Version" (Split-Path -Leaf $Nsi)
    } finally {
        Pop-Location
    }
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "makensis failed (exit $LASTEXITCODE); staging artifacts kept"
    } elseif (Test-Path $Setup) {
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
