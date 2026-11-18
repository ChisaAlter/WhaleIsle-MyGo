# Compile and run the production installer with a tiny synthetic payload and
# unique product identity. HealthAcceptance additionally runs a valid Windows
# candidate that deliberately exits before Host ready; uninstallers never run.
# Run inside with-build-workspace.ps1 so its finally block owns disk cleanup.
param(
    [Parameter(Mandatory=$true)][string]$WorkRoot,
    [Parameter(Mandatory=$true)][string]$ReportPath,
    [switch]$HealthAcceptance
)
$ErrorActionPreference = 'Stop'
$project = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$root = [IO.Path]::GetFullPath($WorkRoot)
$report = [IO.Path]::GetFullPath($ReportPath)
if (-not [IO.Path]::IsPathFullyQualified($WorkRoot) -or
    -not [IO.Path]::IsPathFullyQualified($ReportPath) -or
    (Test-Path -LiteralPath $root)) { throw 'Use absolute paths and a fresh fixture directory.' }
if ($report.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase)) { throw 'Keep the report outside the disposable fixture.' }
$makensis = Join-Path $env:LOCALAPPDATA 'mygo\nsis-3.13\Bin\makensis.exe'
if (-not (Test-Path -LiteralPath $makensis -PathType Leaf)) { throw 'Shared NSIS 3.13 is required.' }
$id = [guid]::NewGuid().ToString('N')
$appID = 'whaleisle.nsis-identity.' + $id
$productName = 'WhaleIsle Identity Fixture ' + $id
$appKey = 'HKCU:\Software\' + $appID
$uninstallKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\' + $appID
$shortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) ($productName + '.lnk')
$menu = Join-Path ([Environment]::GetFolderPath('Programs')) $productName
$menuShortcut = Join-Path $menu ($productName + '.lnk')
foreach ($path in @($appKey,$uninstallKey,$shortcut,$menu)) {
    if (Test-Path -LiteralPath $path) { throw "Fixture identity already exists: $path" }
}
$result = [ordered]@{
    schemaVersion=1
    scope='Production whaleisle.nsi with shell-only synthetic bytes; real silent NSIS install, identity rejection and upgrade'
    appID=$appID
    workRoot=$root
    productScript='scripts/nsis/whaleisle.nsi'
    checks=@()
    passed=$false
    cleanup=@{}
}
$steps = [Collections.Generic.List[object]]::new()
$failure = $null
$running = [Collections.Generic.List[Diagnostics.Process]]::new()
function Write-Synthetic([string]$Path,[string]$Text) {
    $null = [IO.Directory]::CreateDirectory((Split-Path -Parent $Path))
    [IO.File]::WriteAllText($Path,$Text,[Text.UTF8Encoding]::new($false))
}
function Tree-Bytes([string]$Directory) {
    $items = [ordered]@{}
    foreach ($item in @(Get-ChildItem -LiteralPath $Directory -Force -Recurse | Sort-Object FullName)) {
        if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Synthetic tree unexpectedly contains a link.' }
        $relative = [IO.Path]::GetRelativePath($Directory,$item.FullName)
        $items[$relative] = if ($item.PSIsContainer) { 'directory' } else { [Convert]::ToBase64String([IO.File]::ReadAllBytes($item.FullName)) }
    }
    return ($items | ConvertTo-Json -Depth 5 -Compress)
}
function Registration-State {
    $items = [ordered]@{}
    foreach ($path in @($appKey,$uninstallKey)) {
        if (-not (Test-Path -LiteralPath $path)) { $items[$path]=$null; continue }
        $key = Get-Item -LiteralPath $path
        $values = [ordered]@{}
        foreach ($name in @($key.GetValueNames() | Sort-Object)) {
            $values[$name]=@{kind=[string]$key.GetValueKind($name);value=$key.GetValue($name)}
        }
        $items[$path]=$values
    }
    foreach ($path in @($shortcut,$menuShortcut)) {
        $items[$path]=if(Test-Path -LiteralPath $path){[Convert]::ToBase64String([IO.File]::ReadAllBytes($path))}else{$null}
    }
    return ($items | ConvertTo-Json -Depth 8 -Compress)
}
function Transaction-Siblings([string]$Target) {
    $siblings=@(Get-ChildItem -LiteralPath (Split-Path -Parent $Target) -Force |
        Where-Object { $_.Name.StartsWith((Split-Path -Leaf $Target)+'.',[StringComparison]::OrdinalIgnoreCase) })
    return $siblings
}
function Assert-NoTransactionSibling([string]$Target) {
    $siblings=@(Transaction-Siblings $Target)
    if ($siblings.Count) { throw "Unexpected transaction sibling: $($siblings.FullName -join ', ')" }
}
function Invoke-OwnedInstaller([string]$Setup,[string]$Target) {
    if (-not $Setup.StartsWith($root+'\',[StringComparison]::OrdinalIgnoreCase) -or
        -not $Target.StartsWith($root+'\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Installer/target escaped fixture.' }
    $info=[Diagnostics.ProcessStartInfo]::new()
    $info.FileName=$Setup
    # NSIS requires /D to be the final unquoted argument, including spaces.
    $info.Arguments='/S /D='+$Target
    $info.UseShellExecute=$false
    $info.CreateNoWindow=$true
    $info.WindowStyle=[Diagnostics.ProcessWindowStyle]::Hidden
    $process=[Diagnostics.Process]::new()
    $process.StartInfo=$info
    if (-not $process.Start()) { throw 'Owned installer did not start.' }
    $running.Add($process)
    if (-not $process.WaitForExit(45000)) { throw "Owned installer did not exit; retained PID $($process.Id)." }
    return $process.ExitCode
}
function Build-Installer([string]$Stage,[string]$Name,[string]$Version,[switch]$HealthRequired) {
    $setup=Join-Path $root ($Name+'.exe')
    Push-Location $PSScriptRoot
    try {
        $compilerArgs = @('/V2','/NOCD')
        if (-not $HealthRequired) { $compilerArgs += '/DSHELL_ONLY=1' }
        $compilerArgs += @("/DVERSION=$Version", "/DSTAGE_DIR=$Stage", "/DSETUP_OUT=$setup", "/DAPP_ID=$appID", "/DPRODUCT_NAME=$productName", (Join-Path $PSScriptRoot 'whaleisle.nsi'))
        & $makensis @compilerArgs | Out-Host
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $setup)) { throw 'Production NSIS script did not compile.' }
    } finally { Pop-Location }
    return $setup
}
try {
    $null=[IO.Directory]::CreateDirectory($root)
    $stage=Join-Path $root 'stage'
    # The setup never runs these files. The notification cleanup helper is
    # embedded only to compile the uninstaller, which this fixture never runs.
    foreach ($file in @('whaleisle.exe','whaleisle-preview.exe','whaleisle-integration.exe','ghostty-vt.dll','ghostty-vt.dll.LICENSE','terminal-runtime.json','resources\fixture.txt')) {
        Write-Synthetic (Join-Path $stage $file) ('synthetic v1 '+$file)
    }
    $first=Build-Installer $stage 'install-v1' '0.0.1-identity-fixture'
    Write-Synthetic (Join-Path $stage 'whaleisle.exe') 'synthetic v2 whaleisle.exe'
    $second=Build-Installer $stage 'install-v2' '0.0.2-identity-fixture'
    $result.installerBytes=@((Get-Item -LiteralPath $first).Length,(Get-Item -LiteralPath $second).Length)
    $installed=Join-Path $root 'installed'
    $null=[IO.Directory]::CreateDirectory($installed)
    $code=Invoke-OwnedInstaller $first $installed
    if ($code -ne 0) { throw "Empty directory install exited $code." }
    if ([IO.File]::ReadAllText((Join-Path $installed 'whaleisle.exe')) -cne 'synthetic v1 whaleisle.exe') { throw 'Install did not publish v1 payload.' }
    $registration=Get-ItemProperty -LiteralPath $uninstallKey
    if ($registration.InstallLocation -cne $installed -or $registration.NativeRuntime -cne 'WhaleIsle-MyGo') { throw 'Install published incorrect native identity.' }
    $firstInstallID=$registration.NativeInstallID
    $manifest=[IO.File]::ReadAllText((Join-Path $installed '.whaleisle-owned-v1'),[Text.Encoding]::Unicode)
    if (-not $manifest.Contains('I|'+$firstInstallID)) { throw 'Manifest and registry identities differ.' }
    # The existing transaction deliberately retains old directories lacking a
    # manifest, even an initially empty target. Record that contract instead of
    # mistaking it for a failed installation or removing this acceptance case.
    $initialSiblings=@(Transaction-Siblings $installed)
    if ($initialSiblings.Count -ne 1 -or -not $initialSiblings[0].Name.StartsWith('installed.old-') -or
        @(Get-ChildItem -LiteralPath $initialSiblings[0].FullName -Force).Count -ne 0) { throw 'Initial install retained more than its original empty directory.' }
    $initialEmptyBackup=$initialSiblings[0].FullName
    $steps.Add(@{name='empty-target-install';pass=$true;exitCode=$code;manifestAndRegistryAgree=$true;initialEmptyDirectoryRetained=$true})
    $registrationBefore=Registration-State
    $installedBefore=Tree-Bytes $installed
    # An Electron marker wins even if native-looking payload and a valid MyGo
    # manifest were copied beside it. This is the original overwrite risk.
    $electron=Join-Path $root 'electron'
    Write-Synthetic (Join-Path $electron 'Whale Isle.exe') 'synthetic Electron main'
    Write-Synthetic (Join-Path $electron 'user-kept.txt') 'synthetic existing user file'
    Write-Synthetic (Join-Path $electron 'whaleisle.exe') 'native-looking filename grants no authority'
    Copy-Item -LiteralPath (Join-Path $installed '.whaleisle-owned-v1') -Destination (Join-Path $electron '.whaleisle-owned-v1')
    $before=Tree-Bytes $electron
    $code=Invoke-OwnedInstaller $second $electron
    if ($code -eq 0) { throw 'Electron target was accepted.' }
    if ((Tree-Bytes $electron) -cne $before -or (Tree-Bytes $installed) -cne $installedBefore -or (Registration-State) -cne $registrationBefore) { throw 'Rejected Electron install changed existing files or registration.' }
    Assert-NoTransactionSibling $electron
    $steps.Add(@{name='electron-with-native-spoof-rejected';pass=$true;exitCode=$code;exactFilesPreserved=$true;nativeRegistrationPreserved=$true;noStageOrBackup=$true})
    $unowned=Join-Path $root 'unowned-native'
    Write-Synthetic (Join-Path $unowned 'whaleisle.exe') 'unowned same filename'
    Write-Synthetic (Join-Path $unowned 'user-kept.txt') 'unowned sentinel'
    $before=Tree-Bytes $unowned
    $code=Invoke-OwnedInstaller $second $unowned
    if ($code -eq 0 -or (Tree-Bytes $unowned) -cne $before -or (Registration-State) -cne $registrationBefore) { throw 'Unowned native filename was accepted or modified.' }
    Assert-NoTransactionSibling $unowned
    $steps.Add(@{name='unowned-native-filename-rejected';pass=$true;exitCode=$code;exactFilesPreserved=$true;nativeRegistrationPreserved=$true;noStageOrBackup=$true})
    $code=Invoke-OwnedInstaller $second $installed
    if ($code -ne 0) { throw "Owned MyGo upgrade exited $code." }
    if ([IO.File]::ReadAllText((Join-Path $installed 'whaleisle.exe')) -cne 'synthetic v2 whaleisle.exe') { throw 'Upgrade did not publish v2 payload.' }
    $registration=Get-ItemProperty -LiteralPath $uninstallKey
    $secondInstallID=$registration.NativeInstallID
    $manifest=[IO.File]::ReadAllText((Join-Path $installed '.whaleisle-owned-v1'),[Text.Encoding]::Unicode)
    if ($registration.InstallLocation -cne $installed -or $registration.DisplayVersion -cne '0.0.2-identity-fixture' -or
        $secondInstallID -eq $firstInstallID -or -not $manifest.Contains('I|'+$secondInstallID)) { throw 'Upgrade identity/publication is inconsistent.' }
    $siblings=@(Transaction-Siblings $installed)
    if ($siblings.Count -ne 1 -or $siblings[0].FullName -cne $initialEmptyBackup -or
        @(Get-ChildItem -LiteralPath $initialEmptyBackup -Force).Count -ne 0) { throw 'Upgrade retained a new transaction directory.' }
    $steps.Add(@{name='owned-native-upgrade';pass=$true;exitCode=$code;newPayloadConfirmed=$true;newManifestAndRegistryAgree=$true;noNewStageOrBackup=$true})
    if ($HealthAcceptance) {
    # Exercise a real post-promotion startup failure, not a fake health flag.
    # This tiny executable is valid Windows code that exits before Host ready.
    # The production embedded health helper must reject it and restore v2.
    $failedStage=Join-Path $root 'failed-stage'
    $null=[IO.Directory]::CreateDirectory($failedStage)
    foreach ($file in @('whaleisle-preview.exe','ghostty-vt.dll.LICENSE','terminal-runtime.json','resources\fixture.txt','runtime\node\node.exe','.runtime-staged')) {
        Write-Synthetic (Join-Path $failedStage $file) ('synthetic failed candidate '+$file)
    }
    $source=Join-Path $root 'failed-main.go'
    Write-Synthetic $source "package main`nimport `"os`"`nfunc main(){os.Exit(17)}`n"
    & go build -o (Join-Path $failedStage 'whaleisle.exe') $source
    if ($LASTEXITCODE -ne 0) { throw 'Failed-candidate executable did not build.' }
    & go build -tags release -ldflags '-s -w -H windowsgui' -o (Join-Path $failedStage 'whaleisle-integration.exe') (Join-Path $project 'cmd\whaleisle-integration')
    if ($LASTEXITCODE -ne 0) { throw 'Actual installation-health helper did not build.' }
    $terminal=Join-Path $env:LOCALAPPDATA 'Programs\WhaleIsle-MyGo\ghostty-vt.dll'
    if (-not (Test-Path -LiteralPath $terminal -PathType Leaf)) { throw 'Pinned installed terminal is required for real health-failure acceptance.' }
    Copy-Item -LiteralPath $terminal -Destination (Join-Path $failedStage 'ghostty-vt.dll')
    $failed=Build-Installer $failedStage 'install-health-failure' '0.0.3-health-failure' -HealthRequired
    $registrationBefore=Registration-State
    $installedBefore=Tree-Bytes $installed
    $code=Invoke-OwnedInstaller $failed $installed
    if ($code -eq 0) { throw 'Startup-failing candidate was committed.' }
    if ((Tree-Bytes $installed) -cne $installedBefore -or (Registration-State) -cne $registrationBefore) { throw 'Health failure did not restore exact prior payload and management entries.' }
    $health=Get-Content -LiteralPath (Join-Path $env:LOCALAPPDATA 'mygo\installation-health.json') -Raw|ConvertFrom-Json
    if ($health.passed -or -not $health.processesExited -or -not $health.profileRemoved -or $health.error -notlike '*exited before*') { throw 'The failed candidate did not reach the actual health process/cleanup boundary.' }
    $steps.Add(@{name='post-promotion-startup-failure-rolls-back';pass=$true;exitCode=$code;previousPayloadExact=$true;previousRegistrationAndShortcutsExact=$true;health=$health})
    }
    $result.passed=$true
} catch {
    $failure=$_
    $result.error=$_.Exception.Message
} finally {
    $result.checks=$steps.ToArray()
    $active=@($running | Where-Object { -not $_.HasExited })
    if ($active.Count) {
        $result.cleanup=@{complete=$false;retainedRoot=$root;activePIDs=@($active | ForEach-Object Id)}
    } else {
        # These names were absent before this run and contain this run's GUID.
        # Avoid launching the product's notification/uninstall helper here.
        foreach ($path in @($appKey,$uninstallKey)) {
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Recurse -Force }
        }
        foreach ($path in @($shortcut,$menuShortcut)) {
            if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path -Force }
        }
        if (Test-Path -LiteralPath $menu) { [IO.Directory]::Delete($menu,$false) }
        $left=@($appKey,$uninstallKey,$shortcut,$menu | Where-Object { Test-Path -LiteralPath $_ })
        $result.cleanup=@{registrationAndShortcutsRemoved=$left.Count -eq 0;disk='caller-owned build workspace finally';runningInstallers=0}
        if ($left.Count -and -not $failure) { $failure='Owned fixture management cleanup failed.'; $result.passed=$false; $result.error=[string]$failure }
    }
    foreach ($process in $running) { $process.Dispose() }
    [IO.File]::WriteAllText($report,($result|ConvertTo-Json -Depth 10),[Text.UTF8Encoding]::new($false))
}
if ($failure) { throw $failure }
Write-Output ('Installer identity acceptance passed: '+$report)
