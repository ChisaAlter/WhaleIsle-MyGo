# Build the original mobile runtime with plain Node ABI, never Electron ABI.
param(
    [string]$SourceRoot = $env:WHALEISLE_ELECTRON_SOURCE,
    [string]$SileroVadFile = $env:WHALEISLE_SILERO_VAD_FILE
)
$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
if (-not $SourceRoot) { $SourceRoot = Join-Path $ProjectRoot 'reference\electron' }
$NodeDir = $env:WHALEISLE_NODE_DIR
if (-not $NodeDir) { $NodeDir = 'C:\AI\tools\node-v24.21.0-win-x64' }
$Node = Join-Path $NodeDir 'node.exe'
$Npm = Join-Path $NodeDir 'npm.cmd'
if (-not (Test-Path $Node) -or -not (Test-Path $Npm)) { throw 'Set WHALEISLE_NODE_DIR to an official Windows Node distribution (node.exe and npm.cmd).' }
$env:PATH = "$NodeDir;$env:PATH"
$env:npm_config_runtime = 'node'
$env:npm_config_target = (& $Node -p 'process.versions.node')
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect bundled Node version' }
$Work = Join-Path $ProjectRoot 'runtime\build\mobile'
$Helper = Join-Path $PSScriptRoot 'mobile-runtime.mjs'
& $Node $Helper prepare $SourceRoot $Work $SileroVadFile
if ($LASTEXITCODE -ne 0) { throw 'Mobile source preparation failed' }
$Upstream = Join-Path $Work 'vendor\chisacode-remote'
Push-Location $Upstream
try {
    # Locked dependency graph; enable only the required native build scripts.
    & $Npm ci --ignore-scripts --no-audit --no-fund
    if ($LASTEXITCODE -ne 0) { throw 'Mobile npm ci failed' }
    & $Npm rebuild esbuild better-sqlite3 node-pty
    if ($LASTEXITCODE -ne 0) { throw 'Mobile Node ABI rebuild failed' }
    & $Npm run build:server
    if ($LASTEXITCODE -ne 0) { throw 'Mobile server build failed' }
} finally { Pop-Location }
& $Node $Helper bundle $Work
if ($LASTEXITCODE -ne 0) { throw 'Mobile browser bundle build failed' }
# Load native modules only. No daemon, listening socket, home or credentials.
$Probe = @'
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto');
const root=process.argv[1], nm=path.join(root,'vendor/chisacode-remote/node_modules');
const Database=require(path.join(nm,'better-sqlite3'));
const db=new Database(':memory:'); db.prepare('select 1').get(); db.close();
require(path.join(nm,'node-pty'));
fs.writeFileSync(path.join(root,'node-build.json'),JSON.stringify({
  platform:process.platform,arch:process.arch,node:process.version,abi:process.versions.modules,
  nodeSHA256:crypto.createHash('sha256').update(fs.readFileSync(process.execPath)).digest('hex')
},null,2));
'@
& $Node -e $Probe $Work
if ($LASTEXITCODE -ne 0) { throw 'Mobile native modules failed plain Node ABI verification' }
& $Node $Helper stage $Work (Join-Path $ProjectRoot 'runtime\vendor\deepseek-harness') $NodeDir (Join-Path $ProjectRoot 'runtime\vendor\dshd-remote')
if ($LASTEXITCODE -ne 0) { throw 'Mobile runtime staging failed' }
Write-Host 'Mobile runtime built and staged; live phone/relay acceptance still required.'
