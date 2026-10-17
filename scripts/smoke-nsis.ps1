# smoke-nsis.ps1 - M11 NSIS installer smoke (staging + makensis + artifact checks).
#
# Usage (repo root):
#   . .\scripts\env.ps1; .\scripts\smoke-nsis.ps1
#   .\scripts\smoke-nsis.ps1 -SkipPackage          # reuse existing dist/whaleisle/
#   .\scripts\smoke-nsis.ps1 -EnsureNsis           # winget silent install into .toolchain/nsis
#
# Env: WHALEISLE_VERSION, WHALEISLE_PACKAGE_SKIP_RUNTIME (default 1 for speed)

param(
    [switch] $SkipPackage,
    [switch] $EnsureNsis,
    # Silent-install into a TEMP dir (never touches real Electron Programs path), then uninstall.
    [switch] $SilentInstall
)

$ErrorActionPreference = 'Stop'

$ProjectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $ProjectRoot

. (Join-Path $PSScriptRoot 'env.ps1')

function Resolve-Makensis {
    $cmd = Get-Command makensis -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $local = Join-Path $ProjectRoot '.toolchain\nsis\makensis.exe'
    if (Test-Path $local) { return $local }

    return $null
}

function Ensure-ToolchainNsis {
    $local = Join-Path $ProjectRoot '.toolchain\nsis\makensis.exe'
    if (Test-Path $local) { return $local }

    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $winget) {
        throw 'makensis not found and winget unavailable; install NSIS or place makensis.exe under .toolchain/nsis/'
    }

    $dl = Join-Path $ProjectRoot '.toolchain\nsis-dl'
    New-Item -ItemType Directory -Force -Path $dl | Out-Null
    Write-Host 'Ensuring NSIS via winget download + silent install to .toolchain/nsis ...'
    & winget.Source download --id NSIS.NSIS --download-directory $dl --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) { throw "winget download NSIS failed (exit $LASTEXITCODE)" }

    $setup = Get-ChildItem $dl -Filter '*.exe' | Select-Object -First 1
    if (-not $setup) { throw "NSIS installer not found under $dl" }

    $dest = Join-Path $ProjectRoot '.toolchain\nsis'
    if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    & $setup.FullName /S "/D=$dest"
    Start-Sleep -Seconds 3
    if (-not (Test-Path $local)) {
        throw "NSIS silent install did not produce $local"
    }
    return $local
}

if ($EnsureNsis) {
    $null = Ensure-ToolchainNsis
}

$Makensis = Resolve-Makensis
if (-not $Makensis) {
    Write-Error @'
makensis not on PATH and .toolchain/nsis/makensis.exe missing.
Run:  .\scripts\smoke-nsis.ps1 -EnsureNsis
Or install NSIS and retry.
'@
}

if ($env:WHALEISLE_VERSION) {
    $Version = $env:WHALEISLE_VERSION
} else {
    $Version = '0.0.0-dev'
}

if (-not $SkipPackage) {
    if (-not $env:WHALEISLE_PACKAGE_SKIP_RUNTIME) {
        $env:WHALEISLE_PACKAGE_SKIP_RUNTIME = '1'
    }
    Write-Host "Running package.ps1 (WHALEISLE_PACKAGE_SKIP_RUNTIME=$($env:WHALEISLE_PACKAGE_SKIP_RUNTIME)) ..."
    & (Join-Path $PSScriptRoot 'package.ps1')
}

$StageExe = Join-Path $ProjectRoot 'dist\whaleisle\whaleisle.exe'
if (-not (Test-Path $StageExe)) {
    throw "Staging missing: $StageExe (run package.ps1 first)"
}

$Nsi = Join-Path $PSScriptRoot 'nsis\whaleisle.nsi'
if (-not (Test-Path $Nsi)) {
    throw "NSIS script missing: $Nsi"
}

$Setup = Join-Path $ProjectRoot ("dist\WhaleIsle-Setup-{0}.exe" -f $Version)
if (Test-Path $Setup) {
    Remove-Item -Force $Setup
}

Write-Host "Compiling NSIS with $Makensis (VERSION=$Version) ..."
Push-Location (Join-Path $PSScriptRoot 'nsis')
try {
    & $Makensis "/DVERSION=$Version" (Split-Path -Leaf $Nsi)
    if ($LASTEXITCODE -ne 0) {
        throw "makensis failed (exit $LASTEXITCODE)"
    }
} finally {
    Pop-Location
}

if (-not (Test-Path $Setup)) {
    throw "Expected installer not found: $Setup"
}
$len = (Get-Item $Setup).Length
if ($len -le 0) {
    throw "Installer size invalid: $len"
}
Write-Host "OK: $Setup ($len bytes)"

$SevenZip = Get-Command 7z -ErrorAction SilentlyContinue
if (-not $SevenZip) {
    $SevenZip = Get-Command 7za -ErrorAction SilentlyContinue
}
if ($SevenZip) {
    Write-Host "Listing installer contents ($($SevenZip.Source)) ..."
    & $SevenZip.Source l $Setup | Select-Object -First 40
} else {
    Write-Host 'Note: 7z not on PATH; skipped archive listing'
}

if ($SilentInstall) {
    $InstallRoot = Join-Path $env:TEMP ("whaleisle-nsis-smoke-" + [guid]::NewGuid().ToString('n'))
    Write-Host "Silent install -> $InstallRoot ..."
    # NSIS: /S silent; /D= must be last and unquoted.
    $p = Start-Process -FilePath $Setup -ArgumentList @('/S', "/D=$InstallRoot") -Wait -PassThru
    if ($p.ExitCode -ne 0) {
        throw "Silent install exit $($p.ExitCode)"
    }
    $InstalledExe = Join-Path $InstallRoot 'whaleisle.exe'
    if (-not (Test-Path $InstalledExe)) {
        throw "Silent install missing $InstalledExe"
    }
    $exeLen = (Get-Item $InstalledExe).Length
    if ($exeLen -le 0) {
        throw "Installed exe size invalid: $exeLen"
    }
    Write-Host "OK installed: $InstalledExe ($exeLen bytes)"

    $Uninst = Join-Path $InstallRoot 'Uninstall.exe'
    if (Test-Path $Uninst) {
        Write-Host 'Running silent uninstall ...'
        $u = Start-Process -FilePath $Uninst -ArgumentList @('/S') -Wait -PassThru
        if ($u.ExitCode -ne 0) {
            Write-Host "WARN: uninstall exit $($u.ExitCode)"
        }
    }
    if (Test-Path $InstallRoot) {
        Remove-Item -Recurse -Force $InstallRoot -ErrorAction SilentlyContinue
    }
    Write-Host 'Silent install/uninstall smoke: passed'
}

Write-Host 'smoke-nsis.ps1: passed'
