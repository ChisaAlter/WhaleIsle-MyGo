# Actual NSIS transaction fixture under an owned temporary directory only.
# No product registration, shortcuts, installed app, profile or network access.
param([string]$OutputRoot)
$ErrorActionPreference='Stop'
$project = Split-Path -Parent $PSScriptRoot
. (Join-Path $PSScriptRoot 'env.ps1')
if (-not $OutputRoot) { $OutputRoot = Join-Path $project ('tmp\nsis-runtime-links-' + [guid]::NewGuid().ToString('N')) }
$fixtureRoot = [IO.Path]::GetFullPath($OutputRoot)
if (Test-Path -LiteralPath $fixtureRoot) { throw 'Fixture output must be fresh.' }
New-Item -ItemType Directory -Path $fixtureRoot | Out-Null
$stage = Join-Path $fixtureRoot 'stage'
$vendor = Join-Path $stage 'runtime\vendor'
New-Item -ItemType Directory -Path (Join-Path $vendor 'one\node_modules'),(Join-Path $vendor 'two\node_modules') -Force | Out-Null
[IO.File]::WriteAllText((Join-Path $vendor 'one\index.js'),'// one')
[IO.File]::WriteAllText((Join-Path $vendor 'two\index.js'),'// two')
[IO.File]::WriteAllText((Join-Path $vendor 'z-blocked.bin'),'owned late cleanup fixture')
$files = [ordered]@{}
$files['one/index.js'] = (Get-FileHash (Join-Path $vendor 'one\index.js') -Algorithm SHA256).Hash.ToLowerInvariant()
$files['two/index.js'] = (Get-FileHash (Join-Path $vendor 'two\index.js') -Algorithm SHA256).Hash.ToLowerInvariant()
$files['z-blocked.bin'] = (Get-FileHash (Join-Path $vendor 'z-blocked.bin') -Algorithm SHA256).Hash.ToLowerInvariant()
$graph = [ordered]@{schemaVersion=1;files=$files;links=[ordered]@{'one/node_modules/two'=@{target='two';kind='directory'};'two/node_modules/one'=@{target='one';kind='directory'}}}
[IO.File]::WriteAllText((Join-Path $vendor 'runtime-graph.json'),($graph | ConvertTo-Json -Depth 8))
$helper = Join-Path $stage 'runtime\runtime-links.exe'
& go build -ldflags '-s -w -H windowsgui' -o $helper ./scripts/session-runtime-links
if ($LASTEXITCODE -ne 0) { throw 'Helper build failed' }
$makensis = Join-Path $project '.toolchain\nsis\makensis.exe'
$include = Join-Path $PSScriptRoot 'nsis'
function Build-Fixture([string]$Name,[bool]$Fail) {
    $out=Join-Path $fixtureRoot ($Name+'.exe')
    $source=Join-Path $fixtureRoot ($Name+'.nsi')
    $failure=if($Fail){'Abort "Fixture rollback requested"'}else{'Call wiCommitInstallation'}
    $text=@'
Unicode true
Name "Runtime link transaction fixture"
OutFile "__OUT__"
RequestExecutionLevel user
InstallDir "$TEMP\runtime-links-fixture-unused"
SilentInstall silent
SilentUnInstall silent
!define APP_ID "whaleisle-runtime-link-fixture"
!define PRODUCT_NAME "Runtime link fixture"
!define STAGE_DIR "__STAGE__"
!addincludedir "__INCLUDE__"
!include "payload-transaction.nsh"
Section
 Call wiBeginStage
 SetOutPath "$INSTDIR"
 File /oname=whaleisle.exe "__HELPER__"
 SetOutPath "$INSTDIR\runtime"
 File "__HELPER__"
 SetOutPath "$INSTDIR\runtime\vendor"
 File "__VENDOR__\runtime-graph.json"
 File /oname=z-blocked.bin "__VENDOR__\z-blocked.bin"
 SetOutPath "$INSTDIR\runtime\vendor\one"
 File "__VENDOR__\one\index.js"
 CreateDirectory "$INSTDIR\runtime\vendor\one\node_modules"
 SetOutPath "$INSTDIR\runtime\vendor\two"
 File "__VENDOR__\two\index.js"
 CreateDirectory "$INSTDIR\runtime\vendor\two\node_modules"
 SetOutPath "$INSTDIR"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 Call wiCreateManifest
 ${If} $wiError != 0
  Abort "Fixture ownership failed"
 ${EndIf}
 Call wiPromote
 __FAILURE__
SectionEnd
Section Uninstall
 Call un.wiRemovePayload
SectionEnd
'@
    $text=$text.Replace('__OUT__',$out).Replace('__STAGE__',$stage).Replace('__INCLUDE__',$include).Replace('__HELPER__',$helper).Replace('__VENDOR__',$vendor).Replace('__FAILURE__',$failure)
    [IO.File]::WriteAllText($source,$text)
    & $makensis /V2 $source | Out-Host
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $out)) { throw 'NSIS fixture compilation failed' }
    return $out
}
function Run-HiddenExecutable([string]$Exe,[string[]]$Arguments) {
    $psi = New-Object Diagnostics.ProcessStartInfo
    $psi.FileName=$Exe
    $psi.UseShellExecute=$false
    $psi.CreateNoWindow=$true
    $psi.WindowStyle=[Diagnostics.ProcessWindowStyle]::Hidden
    $psi.Arguments=($Arguments -join ' ')
    $process = New-Object Diagnostics.Process
    $process.StartInfo=$psi
    if (-not $process.Start()) { throw 'Fixture process failed to start' }
    $process.WaitForExit()
    return $process.ExitCode
}
$normal=Build-Fixture 'normal' $false
$rollback=Build-Fixture 'rollback' $true
$installed=Join-Path $fixtureRoot 'installed'
$installCode = Run-HiddenExecutable $normal @('/S',('/D='+$installed))
if ($installCode -ne 0) { throw "Fixture installation failed: exit=$installCode root=$fixtureRoot" }
if ((Run-HiddenExecutable $helper @('verify',('"'+(Join-Path $installed 'runtime\vendor')+'"'))) -ne 0) { throw 'Installed graph failed verification' }
$before=(Get-FileHash (Join-Path $installed 'runtime\vendor\one\index.js') -Algorithm SHA256).Hash
if ((Run-HiddenExecutable $rollback @('/S',('/D='+$installed))) -eq 0) { throw 'Rollback fixture unexpectedly succeeded' }
if ((Run-HiddenExecutable $helper @('verify',('"'+(Join-Path $installed 'runtime\vendor')+'"'))) -ne 0) { throw 'Rollback did not restore original junction graph' }
if ((Get-FileHash (Join-Path $installed 'runtime\vendor\one\index.js') -Algorithm SHA256).Hash -ne $before) { throw 'Rollback changed original bytes' }
if ((Run-HiddenExecutable $normal @('/S',('/D='+$installed))) -ne 0) { throw 'Fixture upgrade failed' }
if ((Run-HiddenExecutable $helper @('verify',('"'+(Join-Path $installed 'runtime\vendor')+'"'))) -ne 0) { throw 'Upgrade graph failed verification' }
$uninstall=Join-Path $fixtureRoot 'fixture-uninstall.exe'
Copy-Item -LiteralPath (Join-Path $installed 'Uninstall.exe') -Destination $uninstall
# Run the management copy outside the renamed payload; _?= keeps it synchronous.
$unknown=Join-Path $installed 'runtime\vendor\unknown'
New-Item -ItemType Junction -Path $unknown -Target (Join-Path $installed 'runtime\vendor\two') | Out-Null
$uninstallCode=Run-HiddenExecutable $uninstall @('/S',('_?='+$installed))
if ($uninstallCode -eq 0) { throw 'Unknown runtime junction was accepted by the uninstaller' }
if (-not (Test-Path -LiteralPath (Join-Path $installed 'runtime\vendor\one\index.js'))) { throw 'Rejected uninstall changed owned payload bytes' }
# Remove only the exact junction created above, never recurse into its target.
$unknownItem=Get-Item -LiteralPath $unknown
if ($unknownItem.LinkType -ne 'Junction' -or $unknownItem.Target[0] -ne (Join-Path $installed 'runtime\vendor\two')) { throw 'Unexpected test junction target' }
[IO.Directory]::Delete($unknown)
$uninstallCode=Run-HiddenExecutable $uninstall @('/S',('_?='+$installed))
if ($uninstallCode -ne 0) { throw "Fixture uninstall failed: exit=$uninstallCode root=$fixtureRoot" }
if (Test-Path -LiteralPath $installed) { throw 'Successful uninstall retained the owned installation directory' }
$retired=Get-ChildItem -LiteralPath $fixtureRoot -Directory -Filter 'installed.uninstall-*'
if (@($retired).Count -ne 0) { throw 'Successful uninstall retained an owned retired directory' }
# Exercise the real failed-cleanup retry, including a detached graph whose
# physical targets and alias parent directories are already gone.
if ((Run-HiddenExecutable $normal @('/S',('/D='+$installed))) -ne 0) { throw 'Retry fixture installation failed' }
Copy-Item -LiteralPath (Join-Path $installed 'Uninstall.exe') -Destination $uninstall -Force
$sentinel=Join-Path $installed 'user-kept.txt'
[IO.File]::WriteAllText($sentinel,'unowned sentinel must survive failed cleanup')
$firstRetryCode=Run-HiddenExecutable $uninstall @('/S',('_?='+$installed))
if ($firstRetryCode -eq 0) { throw 'Unowned sentinel did not cause the first real uninstall cleanup failure' }
$retired=@(Get-ChildItem -LiteralPath $fixtureRoot -Directory -Filter 'installed.uninstall-*')
if ($retired.Count -ne 1) { throw 'Failed uninstall did not retain one exact manageable retired tree' }
$retiredRoot=$retired[0].FullName
$retainedSentinel=Join-Path $retiredRoot 'user-kept.txt'
if ((Get-Content -LiteralPath $retainedSentinel -Raw) -ne 'unowned sentinel must survive failed cleanup') { throw 'First failed cleanup removed or changed unowned bytes' }
$retryVendor=Join-Path $retiredRoot 'runtime\vendor'
if (Test-Path -LiteralPath (Join-Path $retryVendor 'runtime-graph.json')) {
    if ((Run-HiddenExecutable $helper @('detach',('"'+$retryVendor+'"'))) -ne 0) { throw 'Failed uninstall retained graph cannot be detached' }
} else {
    if (Test-Path -LiteralPath $retryVendor) { throw 'Expected the actual partial cleanup to have removed its complete runtime graph subtree' }
    Write-Output 'Actual partial cleanup removed runtime graph; product retry safely skips graph helper. Graph-present missing-target behavior is covered by three real helper fixtures.'
}
foreach($owned in @('one','two')) {
    if (Test-Path -LiteralPath (Join-Path $retryVendor $owned)) { throw 'First real cleanup did not remove the manifest-owned dependency target subtree' }
}
if ([IO.Path]::GetFullPath($retainedSentinel) -ne (Join-Path $retiredRoot 'user-kept.txt')) { throw 'Sentinel cleanup path changed' }
Remove-Item -LiteralPath $retainedSentinel
Write-Output ('Actual partial cleanup retained unowned bytes and removed owned targets before retry: '+$retiredRoot)
$retryCopy=Join-Path $fixtureRoot 'fixture-retry-uninstall.exe'
Copy-Item -LiteralPath (Join-Path $retiredRoot 'Uninstall.exe') -Destination $retryCopy
if ((Run-HiddenExecutable $retryCopy @('/S',('_?='+$retiredRoot))) -ne 0) { throw 'Partial physical subtree uninstall retry failed' }
if ((Test-Path -LiteralPath $installed) -or (Test-Path -LiteralPath $retiredRoot)) { throw 'Successful retry retained an owned installed or retired root' }
Write-Output ('NSIS runtime links fixture PASS: install, upgrade, rollback, unknown-link refusal, owned uninstall and actual partial-cleanup retry; root='+$fixtureRoot)
