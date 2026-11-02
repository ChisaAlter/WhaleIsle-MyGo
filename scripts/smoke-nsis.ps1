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
    # Silent-install using a unique product identity; never use real registration.
    [switch] $SilentInstall,
    [string] $OutputRoot,
    # Assert the installed desktop executable is exactly these bytes.
    [string] $ExpectedAppSHA256,
    # Run the same silent installer twice to exercise the upgrade path.
    [switch] $Upgrade
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

if (-not $OutputRoot) {
    if ($SkipPackage) { $OutputRoot = Join-Path $ProjectRoot 'dist' }
    else { $OutputRoot = Join-Path $ProjectRoot ('tmp\nsis-smoke-' + [guid]::NewGuid().ToString('n')) }
}
$OutputRoot = [System.IO.Path]::GetFullPath($OutputRoot)
$StageRoot = Join-Path $OutputRoot 'whaleisle'
if (-not $SkipPackage) {
    if (-not $env:WHALEISLE_PACKAGE_SKIP_RUNTIME) { $env:WHALEISLE_PACKAGE_SKIP_RUNTIME = '1' }
    Write-Host "Running package.ps1 in fresh owned output $OutputRoot ..."
    & (Join-Path $PSScriptRoot 'package.ps1') -OutputRoot $OutputRoot
}

$StageExe = Join-Path $StageRoot 'whaleisle.exe'
if (-not (Test-Path $StageExe)) {
    throw "Staging missing: $StageExe (run package.ps1 first)"
}

$StagePreview = Join-Path $StageRoot 'whaleisle-preview.exe'
if (-not (Test-Path $StagePreview)) { throw 'Staged preview helper missing; rebuild with package.ps1' }

$Nsi = Join-Path $PSScriptRoot 'nsis\whaleisle.nsi'
if (-not (Test-Path $Nsi)) {
    throw "NSIS script missing: $Nsi"
}

# Compilation is separate from package.ps1's normal development artifact.
# Even an explicitly reused stage never overwrites an existing installer.
$SmokeID = [guid]::NewGuid().ToString('n')
$SmokeAppID = 'whaleisle.nsis-smoke.' + $SmokeID
$SmokeName = 'WhaleIsle NSIS Smoke ' + $SmokeID
$Setup = Join-Path $OutputRoot ('WhaleIsle-Smoke-' + $SmokeID + '.exe')

Write-Host "Compiling NSIS with $Makensis (VERSION=$Version) ..."
Push-Location (Join-Path $PSScriptRoot 'nsis')
try {
    $NsisArguments = @("/DVERSION=$Version", "/DSTAGE_DIR=$StageRoot", "/DSETUP_OUT=$Setup", "/DAPP_ID=$SmokeAppID", "/DPRODUCT_NAME=$SmokeName")
    $RuntimeInclude = Join-Path $OutputRoot 'runtime-files.nsh'
    if (Test-Path -LiteralPath $RuntimeInclude -PathType Leaf) { $NsisArguments += "/DRUNTIME_INCLUDE=$RuntimeInclude" }
    if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -eq '1') { $NsisArguments += '/DSHELL_ONLY=1' }
    & $Makensis @NsisArguments (Split-Path -Leaf $Nsi)
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

function Invoke-OwnedExecutable([string]$Executable, [string]$Arguments) {
    $info = New-Object Diagnostics.ProcessStartInfo
    $info.FileName = $Executable
    $info.Arguments = $Arguments
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.WindowStyle = [Diagnostics.ProcessWindowStyle]::Hidden
    $process = New-Object Diagnostics.Process
    $process.StartInfo = $info
    if (-not $process.Start()) { throw "Could not start owned fixture: $Executable" }
    if (-not $process.WaitForExit(900000)) { throw "Owned fixture did not exit within 900 seconds: $Executable" }
    $result = $process.ExitCode
    $process.Dispose()
    return $result
}

if ($SilentInstall) {
    $InstallRoot = Join-Path $env:TEMP ('whaleisle-nsis-smoke-' + $SmokeID)
    Write-Host "Silent install -> $InstallRoot ..."
    # NSIS: /S silent; /D= must be last and unquoted.
    $InstallCode = Invoke-OwnedExecutable $Setup "/S /D=$InstallRoot"
    if ($InstallCode -ne 0) { throw "Silent install exit $InstallCode" }
    $InstalledExe = Join-Path $InstallRoot 'whaleisle.exe'
    if (-not (Test-Path $InstalledExe)) {
        throw "Silent install missing $InstalledExe"
    }
    $exeLen = (Get-Item $InstalledExe).Length
    if ($exeLen -le 0) {
        throw "Installed exe size invalid: $exeLen"
    }
    $InstalledPreview = Join-Path $InstallRoot 'whaleisle-preview.exe'
    if (-not (Test-Path $InstalledPreview) -or (Get-Item $InstalledPreview).Length -le 0) {
        throw "Silent install missing preview helper: $InstalledPreview"
    }
    Write-Host "OK installed: $InstalledExe ($exeLen bytes)"
    foreach ($Name in @('Inter-OFL.txt', 'SourceSerif4-OFL.txt', 'sources.json')) {
        $Relative = "resources\licenses\fonts\$Name"
        $StagedNotice = [IO.File]::ReadAllBytes((Join-Path $StageRoot $Relative))
        $InstalledNotice = [IO.File]::ReadAllBytes((Join-Path $InstallRoot $Relative))
        if ([Convert]::ToBase64String($InstalledNotice) -cne [Convert]::ToBase64String($StagedNotice)) {
            throw "Installed font notice differs from the staged source: $Relative"
        }
    }
    Write-Host 'OK installed font notices and provenance'
    if ($ExpectedAppSHA256) {
        $actualApp = (Get-FileHash -LiteralPath $InstalledExe -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actualApp -ne $ExpectedAppSHA256.ToLowerInvariant()) {
            throw "Installed desktop executable differs from the verified build: $actualApp"
        }
        Write-Host "OK installed desktop bytes: $actualApp"
    }
    if ($Upgrade) {
        # Reinstalling over the owned installation exercises the real upgrade
        # path: old payload detached/retired, new payload promoted, one product
        # registration retained. The second run must still succeed.
        Write-Host 'Running the same owned installer again (upgrade) ...'
        $UpgradeCode = Invoke-OwnedExecutable $Setup "/S /D=$InstallRoot"
        if ($UpgradeCode -ne 0) { throw "Silent upgrade exit $UpgradeCode" }
        if (-not (Test-Path -LiteralPath $InstalledExe)) { throw 'Upgrade removed the installed desktop executable.' }
        if ($ExpectedAppSHA256) {
            $upgradedApp = (Get-FileHash -LiteralPath $InstalledExe -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($upgradedApp -ne $ExpectedAppSHA256.ToLowerInvariant()) {
                throw "Upgraded desktop executable differs from the verified build: $upgradedApp"
            }
        }
        $RetiredAfterUpgrade = Get-ChildItem -LiteralPath (Split-Path -Parent $InstallRoot) -Directory -Filter ((Split-Path -Leaf $InstallRoot) + '.uninstall-*')
        if ($RetiredAfterUpgrade) { throw 'Upgrade retained a retired payload; inspect fixture output.' }
        Write-Host 'OK upgrade: payload replaced and no retired tree kept'
    }

    $Uninst = Join-Path $InstallRoot 'Uninstall.exe'
    if (-not (Test-Path -LiteralPath $Uninst -PathType Leaf)) { throw 'Owned installation has no uninstaller.' }
    # The real NSIS uninstaller runs from a temporary copy. Keep the fixture
    # synchronous with _?= while avoiding a loaded executable in the renamed tree.
    $ManagementCopy = Join-Path $OutputRoot ('uninstall-smoke-' + $SmokeID + '.exe')
    Copy-Item -LiteralPath $Uninst -Destination $ManagementCopy
    Write-Host 'Running isolated silent uninstall ...'
    $UninstallCode = Invoke-OwnedExecutable $ManagementCopy "/S _?=$InstallRoot"
    if ($UninstallCode -ne 0) { throw "Silent uninstall exit $UninstallCode" }
    if (Test-Path -LiteralPath $InstallRoot) { throw "Uninstall retained owned directory: $InstallRoot" }
    $Retired = Get-ChildItem -LiteralPath (Split-Path -Parent $InstallRoot) -Directory -Filter ((Split-Path -Leaf $InstallRoot) + '.uninstall-*')
    if ($Retired) { throw 'Uninstall retained a retired payload; inspect fixture output.' }
    $Registration = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\' + $SmokeAppID
    $ProductRegistration = 'HKCU:\Software\' + $SmokeAppID
    if ((Test-Path -LiteralPath $Registration) -or (Test-Path -LiteralPath $ProductRegistration)) { throw 'Isolated installation registration was not removed.' }
    Write-Host 'Silent install/uninstall smoke: passed'
}

Write-Host 'smoke-nsis.ps1: passed'
