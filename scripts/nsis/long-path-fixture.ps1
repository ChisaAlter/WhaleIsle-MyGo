# Small real NSIS regression using the product script and prebuilt transaction
# helpers. Synthetic payload only; never starts desktop, Host, Node, or a model.
# Usage: .\scripts\nsis\long-path-fixture.ps1 -HelperStage dist\0.2.1-rc.3\whaleisle
param([Parameter(Mandatory=$true)][string]$HelperStage, [string]$OutputRoot)
$ErrorActionPreference='Stop'
$project=Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$helperRoot=(Resolve-Path -LiteralPath $HelperStage).Path
foreach($relative in @('whaleisle-integration.exe','runtime\runtime-links.exe')) {
    if(-not(Test-Path -LiteralPath (Join-Path $helperRoot $relative) -PathType Leaf)){throw "Prebuilt helper missing: $relative"}
}
$id=[guid]::NewGuid().ToString('N')
if(-not $OutputRoot){$OutputRoot=Join-Path $project ('tmp\nsis-long-path-'+$id)}
$root=[IO.Path]::GetFullPath($OutputRoot)
if(Test-Path -LiteralPath $root){throw 'Fixture output must be fresh'}
New-Item -ItemType Directory -Path $root | Out-Null
$stage=Join-Path $root 'stage'
$include=Join-Path $root 'runtime-files.nsh'
$appID='whaleisle.nsis-longpath.'+$id
$productName='WhaleIsle Long Path '+$id
# Match the failing smoke install-root length, retaining the transaction GUID.
$installed=Join-Path $env:TEMP ('whaleisle-nsis-smoke-'+$id)
$deepDirectory='runtime\vendor\deepseek-harness\node_modules\.dsh-instances\07dc5704d1df52eb26dd31ab\build\esm\detectors\platform\node\machine-id'
$transactionDirectoryLength=($installed+'.new-{00000000-0000-0000-0000-000000000000}\'+$deepDirectory).Length
if($transactionDirectoryLength -lt 260){throw 'TEMP is too short to exercise the SetCurrentDirectoryW MAX_PATH failure'}
$appKey='HKCU:\Software\'+$appID
$uninstallKey='HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\'+$appID
$shortcut=Join-Path ([Environment]::GetFolderPath('Desktop')) ($productName+'.lnk')
$menu=Join-Path ([Environment]::GetFolderPath('Programs')) $productName
foreach($path in @($installed,$appKey,$uninstallKey,$shortcut,$menu)){if(Test-Path -LiteralPath $path){throw "Fixture target already exists: $path"}}
$prepare=@'
import fs from 'node:fs';import path from 'node:path';import {pathToFileURL} from 'node:url';
const [stage,project,helpers,include]=process.argv.slice(1);
const {inventoryGraph}=await import(pathToFileURL(path.join(project,'scripts/session-runtime-graph.mjs')));
const {emitRuntimeInclude}=await import(pathToFileURL(path.join(project,'scripts/session-runtime-nsis.mjs')));
const put=(name,body)=>{const file=path.join(stage,name);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,body);};
for(const name of ['whaleisle.exe','whaleisle-preview.exe','ghostty-vt.dll','ghostty-vt.dll.LICENSE','terminal-runtime.json','resources/font-license.txt','runtime/node/node.exe','.runtime-staged','.pet-render-staged'])put(name,'synthetic '+name);
const manifest=JSON.parse(fs.readFileSync(path.join(project,'internal/desktop/pet/assets/render-manifest.json'),'utf8'));
for(const file of manifest.files)put('pet-render-assets/'+file.path,'synthetic '+file.path);
put('pet-render-assets/provenance.json','synthetic provenance');
for(const name of ['whaleisle-integration.exe','runtime/runtime-links.exe'])fs.copyFileSync(path.join(helpers,name),path.join(stage,name));
const vendor=path.join(stage,'runtime/vendor');
const deep='deepseek-harness/node_modules/.dsh-instances/07dc5704d1df52eb26dd31ab/build/esm/detectors/platform/node/machine-id/execAsync.js';
put('runtime/vendor/'+deep,'synthetic original long-path payload');
fs.mkdirSync(path.join(vendor,'alias-parent'),{recursive:true});
fs.symlinkSync(path.dirname(path.join(vendor,deep)),path.join(vendor,'alias-parent/machine-id'),'junction');
const graph=inventoryGraph(vendor);fs.writeFileSync(path.join(vendor,'runtime-graph.json'),JSON.stringify({schemaVersion:1,files:graph.files,links:graph.links}));
console.log(JSON.stringify(emitRuntimeInclude(stage,include)));
'@
& node --input-type=module -e $prepare $stage $project $helperRoot $include
if($LASTEXITCODE -ne 0){throw 'Fixture payload preparation failed'}
$makensis=Join-Path $project '.toolchain\nsis\makensis.exe'
$source=Join-Path $PSScriptRoot 'whaleisle.nsi'
$normal=Join-Path $root 'normal.exe'
function Compile([string]$Nsi,[string]$Setup) {
    Push-Location $PSScriptRoot
    try {
        & $makensis /V2 /NOCD '/DVERSION=0.2.1-longpath-fixture' "/DSTAGE_DIR=$stage" "/DSETUP_OUT=$Setup" "/DAPP_ID=$appID" "/DPRODUCT_NAME=$productName" "/DRUNTIME_INCLUDE=$include" $Nsi
        if($LASTEXITCODE -ne 0){throw 'Small NSIS fixture compile failed'}
    } finally {Pop-Location}
}
function Invoke-Owned([string]$Exe,[string]$Arguments) {
    $info=New-Object Diagnostics.ProcessStartInfo
    $info.FileName=$Exe;$info.Arguments=$Arguments;$info.UseShellExecute=$false;$info.CreateNoWindow=$true
    $info.WindowStyle=[Diagnostics.ProcessWindowStyle]::Hidden
    $process=New-Object Diagnostics.Process;$process.StartInfo=$info
    if(-not $process.Start()){throw 'Owned fixture process did not start'}
    if(-not $process.WaitForExit(120000)){throw 'Small fixture timed out; owned output retained'}
    $code=$process.ExitCode;$process.Dispose();return $code
}
$check=@'
import fs from 'node:fs';import path from 'node:path';import assert from 'node:assert/strict';
const root=process.argv[1],deep='runtime/vendor/deepseek-harness/node_modules/.dsh-instances/07dc5704d1df52eb26dd31ab/build/esm/detectors/platform/node/machine-id/execAsync.js';
assert.equal(fs.readFileSync(path.join(root,deep),'utf8'),'synthetic original long-path payload');
assert.equal(fs.readFileSync(path.join(root,'runtime/vendor/alias-parent/machine-id/execAsync.js'),'utf8'),'synthetic original long-path payload');
console.log('Long file preserved: '+path.join(root,deep).length+' characters; declared junction restored');
'@
function Check-Installed {
    & node --input-type=module -e $check $installed
    if($LASTEXITCODE -ne 0){throw 'Installed long path or runtime alias mismatch'}
}
Compile $source $normal
$code=Invoke-Owned $normal ('/S /D='+$installed)
if($code -ne 0){throw "Long-path install exit $code"}
Check-Installed
$before=[IO.File]::ReadAllText((Join-Path $installed '.whaleisle-owned-v1'),[Text.Encoding]::Unicode)
$rollbackSource=Join-Path $root 'rollback.nsi'
$text=[IO.File]::ReadAllText($source)
if(([regex]::Matches($text,'Call wiCommitInstallation')).Count -ne 1){throw 'Unexpected transaction commit seam'}
$text=$text.Replace('Call wiCommitInstallation','SetErrorLevel 2'+"`r`n"+'  Abort "Owned long-path rollback requested"')
[IO.File]::WriteAllText($rollbackSource,$text)
$rollback=Join-Path $root 'rollback.exe'
Compile $rollbackSource $rollback
$code=Invoke-Owned $rollback ('/S /D='+$installed)
if($code -ne 2){throw "Expected forced rollback exit 2, got $code"}
Check-Installed
if([IO.File]::ReadAllText((Join-Path $installed '.whaleisle-owned-v1'),[Text.Encoding]::Unicode) -ne $before){throw 'Rollback replaced the previous installation identity'}
$code=Invoke-Owned $normal ('/S /D='+$installed)
if($code -ne 0){throw "Long-path upgrade exit $code"}
Check-Installed
if([IO.File]::ReadAllText((Join-Path $installed '.whaleisle-owned-v1'),[Text.Encoding]::Unicode) -eq $before){throw 'Upgrade did not promote a new transaction identity'}
$uninstaller=Join-Path $root 'uninstall.exe'
Copy-Item -LiteralPath (Join-Path $installed 'Uninstall.exe') -Destination $uninstaller
$code=Invoke-Owned $uninstaller ('/S _?='+$installed)
if($code -ne 0){throw "Long-path uninstall exit $code"}
foreach($path in @($installed,$appKey,$uninstallKey,$shortcut,$menu)){if(Test-Path -LiteralPath $path){throw "Owned fixture object retained after uninstall: $path"}}
$result=[ordered]@{result='PASS-long-path-install-rollback-upgrade-uninstall';outputRoot=$root;installRoot=$installed;transactionDirectoryLength=$transactionDirectoryLength;appID=$appID;normalSetup=$normal;rollbackSetup=$rollback;syntheticPayloadOnly=$true}
[IO.File]::WriteAllText((Join-Path $root 'receipt.json'),($result|ConvertTo-Json),[Text.UTF8Encoding]::new($false))
$result|ConvertTo-Json
