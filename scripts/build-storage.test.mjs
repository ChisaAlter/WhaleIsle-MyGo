import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { after, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { removeBuildWorkspace, removeOwnedDir, removePackageStage } from './build-storage.mjs';

const scripts = path.dirname(fileURLToPath(import.meta.url));
const temporary = fs.mkdtempSync(path.join(fs.realpathSync.native(os.tmpdir()), 'whale-storage-test-'));
after(() => fs.rmSync(temporary, { recursive: true, force: true }));
let sequence = 0;
const fresh = () => fs.mkdtempSync(path.join(temporary, 'case-'));
function resolvePowerShell() {
  for (const candidate of process.platform === 'win32' ? ['pwsh', 'powershell.exe'] : ['pwsh']) {
    const probe = spawnSync(candidate, ['-NoProfile', '-Command', '$PSVersionTable.PSVersion.Major'], { encoding: 'utf8', windowsHide: true });
    if (!probe.error && probe.status === 0) return candidate;
  }
  return 'pwsh';
}
const powershellExecutable = resolvePowerShell();
function powershell(source, env = {}) {
  const script = path.join(temporary, `probe-${sequence++}.ps1`);
  fs.writeFileSync(script, '$ErrorActionPreference = "Stop"\n' + source);
  return spawnSync(powershellExecutable, ['-NoProfile', '-File', script], { cwd: temporary, env: { ...process.env, ...env }, encoding: 'utf8', windowsHide: true });
}
const quote = value => "'" + value.replaceAll("'", "''") + "'";

test('stage cleanup unlinks external junctions and retains installer, metadata and source', () => {
  const output = fresh(); const stage = path.join(output, 'whaleisle'); const source = fresh();
  fs.mkdirSync(stage); fs.writeFileSync(path.join(source, 'source.go'), 'keep');
  fs.writeFileSync(path.join(output, 'setup.exe'), 'installer');
  fs.writeFileSync(path.join(output, 'setup.exe.blockmap.json'), '{}');
  fs.symlinkSync(source, path.join(stage, 'runtime-link'), process.platform === 'win32' ? 'junction' : 'dir');
  fs.writeFileSync(path.join(stage, 'generated.exe'), 'temporary');
  removePackageStage(stage, output);
  assert.equal(fs.existsSync(stage), false);
  assert.equal(fs.readFileSync(path.join(source, 'source.go'), 'utf8'), 'keep');
  assert.equal(fs.existsSync(path.join(output, 'setup.exe')), true);
  assert.equal(fs.existsSync(path.join(output, 'setup.exe.blockmap.json')), true);
});

test('cleanup refuses unrelated targets, linked roots and linked ancestors', () => {
  const output = fresh(); const source = fresh(); fs.writeFileSync(path.join(source, 'keep'), 'keep');
  assert.throws(() => removePackageStage(source, output), /output directory/);
  fs.symlinkSync(source, path.join(output, 'whaleisle'), process.platform === 'win32' ? 'junction' : 'dir');
  assert.throws(() => removePackageStage(path.join(output, 'whaleisle'), output), /redirected/);
  const parentLink = path.join(output, 'linked-parent');
  fs.symlinkSync(source, parentLink, process.platform === 'win32' ? 'junction' : 'dir');
  fs.mkdirSync(path.join(source, 'whaleisle'));
  assert.throws(() => removePackageStage(path.join(parentLink, 'whaleisle'), parentLink), /redirected/);
  assert.throws(() => removeBuildWorkspace(source, output), /owned whaleisle-build/);
  assert.equal(fs.readFileSync(path.join(source, 'keep'), 'utf8'), 'keep');
});

test('environment uses native shared caches and preserves explicit isolation', { skip: process.platform !== 'win32' }, () => {
  const base = `foreach ($key in @('GOPATH','GOCACHE','GOMODCACHE','GOTMPDIR')) { Remove-Item "Env:$key" -ErrorAction SilentlyContinue }\n`;
  const probe = `. ${quote(path.join(scripts, 'env.ps1'))}\n$pin=$env:GOTOOLCHAIN\n$env:GOTOOLCHAIN='local'\ngo env -json GOPATH GOCACHE GOMODCACHE GOTMPDIR\nWrite-Output "PIN=$pin"`;
  const shared = powershell(base + probe);
  assert.equal(shared.status, 0, shared.stderr);
  const json = JSON.parse(shared.stdout.slice(0, shared.stdout.indexOf('PIN=')));
  assert.ok(!JSON.stringify(json).includes('.toolchain'));
  const pin = /GOTOOLCHAIN\s*=\s*'([^']+)'/.exec(fs.readFileSync(path.join(scripts, 'env.ps1'), 'utf8'))[1];
  assert.match(shared.stdout, new RegExp('PIN=' + pin.replaceAll('.', '\\.').replaceAll('+', '\\+')));
  const isolated = fresh();
  const explicit = powershell(base + `\n$env:GOCACHE=${quote(path.join(isolated, 'cache'))}\n$env:GOMODCACHE=${quote(path.join(isolated, 'modules'))}\n$env:GOTMPDIR=${quote(path.join(isolated, 'temp'))}\n` + probe);
  assert.equal(explicit.status, 0, explicit.stderr);
  const values = JSON.parse(explicit.stdout.slice(0, explicit.stdout.indexOf('PIN=')));
  assert.equal(values.GOCACHE, path.join(isolated, 'cache'));
  assert.equal(values.GOMODCACHE, path.join(isolated, 'modules'));
  assert.equal(values.GOTMPDIR, path.join(isolated, 'temp'));
});

test('QA wrapper removes success/failure workspaces and restores caller TEMP', { skip: process.platform !== 'win32' }, () => {
  for (const fail of [false, true]) {
    const witness = path.join(temporary, `witness-${sequence++}.txt`);
    const result = powershell(`$before=$env:TEMP\ntry { & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} { param($work)\n[IO.File]::WriteAllText(${quote(witness)}, $work)\n[IO.File]::WriteAllText((Join-Path $work 'generated.bin'), 'temporary')\n${fail ? "throw 'expected-fixture-failure'" : "Write-Output 'qa-ok'"}\n} } catch { Write-Output $_.Exception.Message }\nif ($env:TEMP -ne $before) { throw 'TEMP not restored' }`);
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, fail ? /expected-fixture-failure/ : /qa-ok/);
    const workspace = fs.readFileSync(witness, 'utf8');
    assert.equal(fs.existsSync(workspace), false, result.stdout + result.stderr);
  }
});

test('QA wrapper propagates a native nonzero exit after reclaiming its workspace', { skip: process.platform !== 'win32' }, () => {
  const witness = path.join(temporary, 'native-failure-workspace.txt');
  const result = powershell(`try {
  & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
    param($work)
    [IO.File]::WriteAllText(${quote(witness)}, $work)
    & node -e "process.exit(7)"
  }
} catch { Write-Output $_.Exception.Message }`);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /failed \(exit 7\)/);
  assert.equal(fs.existsSync(fs.readFileSync(witness, 'utf8')), false);
});

test('real Go builds share native caches when isolated profiles change LOCALAPPDATA', { skip: process.platform !== 'win32' }, () => {
  const source = path.join(temporary, 'storage-smoke.go');
  fs.writeFileSync(source, 'package main\nimport "fmt"\nfunc main() { fmt.Println("storage-smoke-ok") }\n');
  const result = powershell(`foreach ($key in @('GOPATH','GOCACHE','GOMODCACHE','GOTMPDIR')) { Remove-Item "Env:$key" -ErrorAction SilentlyContinue }
. ${quote(path.join(scripts, 'env.ps1'))}
$cacheBefore = go env GOCACHE
$modulesBefore = go env GOMODCACHE
foreach ($iteration in 1..2) {
  & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
    param($work)
    $callerLocalData = $env:LOCALAPPDATA
    $profile = Join-Path $work 'profile'
    $null = New-Item -ItemType Directory $profile
    try {
      $env:LOCALAPPDATA = $profile
      if ((go env GOCACHE) -ne $cacheBefore -or (go env GOMODCACHE) -ne $modulesBefore) { throw 'Profile redirected the shared caches' }
      $exe = Join-Path $work 'smoke.exe'
      go build -o $exe ${quote(source)}
      if ($LASTEXITCODE -ne 0) { throw 'Real Go compilation failed' }
      & $exe
      if (Test-Path -LiteralPath (Join-Path $profile 'go-build')) { throw 'Isolated profile contains a compile cache' }
      [IO.File]::AppendAllText(${quote(path.join(temporary, 'go-workspaces.txt'))}, $work + [Environment]::NewLine)
    } finally { $env:LOCALAPPDATA = $callerLocalData }
  }
}
if ((go env GOCACHE) -ne $cacheBefore -or (go env GOMODCACHE) -ne $modulesBefore) { throw 'Shared cache paths changed' }
if ($env:GOCACHE -or $env:GOMODCACHE) { throw 'Wrapper did not restore initially unset cache variables' }
Write-Output "SHARED_GOCACHE=$cacheBefore"
Write-Output "SHARED_GOMODCACHE=$modulesBefore"`);
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.equal(result.stdout.match(/storage-smoke-ok/g)?.length, 2);
  assert.ok(!result.stdout.includes('.toolchain'));
  for (const workspace of fs.readFileSync(path.join(temporary, 'go-workspaces.txt'), 'utf8').trim().split(/\r?\n/)) assert.equal(fs.existsSync(workspace), false);
  console.log(result.stdout.trim());
});

test('cold-cache overrides do not leak deleted paths into the next command', { skip: process.platform !== 'win32' }, () => {
  const witness = path.join(temporary, 'cold-cache-workspaces.txt');
  const result = powershell(`$names = @('GOPATH','GOCACHE','GOMODCACHE','npm_config_cache')
$env:GOCACHE = ${quote(path.join(temporary, 'caller-cache'))}
Remove-Item Env:GOPATH -ErrorAction SilentlyContinue
$before = @{}
foreach ($name in $names) { $before[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
foreach ($fail in @($false, $true)) {
  try {
    & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
      param($work)
      [IO.File]::AppendAllText(${quote(witness)}, $work + [Environment]::NewLine)
      foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, (Join-Path $work $name), 'Process') }
      if ($fail) { throw 'expected-cold-cache-failure' }
    }
  } catch { if ($_.Exception.Message -ne 'expected-cold-cache-failure') { throw } }
  foreach ($name in $names) { if ([Environment]::GetEnvironmentVariable($name, 'Process') -cne $before[$name]) { throw "Leaked cache variable: $name" } }
}
Write-Output 'cold-cache-restored'`);
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.match(result.stdout, /cold-cache-restored/);
  for (const workspace of fs.readFileSync(witness, 'utf8').trim().split(/\r?\n/)) assert.equal(fs.existsSync(workspace), false);
});

test('packaging keeps legacy StageOnly, cleans successful installer stage, retains a failed build', { skip: process.platform !== 'win32' }, () => {
  const nsis = path.join(process.env.LOCALAPPDATA, 'mygo', 'nsis-3.13', 'Bin');
  assert.equal(fs.existsSync(path.join(nsis, 'makensis.exe')), true, 'Pinned local NSIS is required for real packaging verification');
  const fakeGo = path.join(temporary, 'fixture-go.ps1');
  fs.writeFileSync(fakeGo, String.raw`
$global:LASTEXITCODE=0
if ($args[0] -eq 'env') { if ($args[1] -in @('GOOS','GOHOSTOS')) { 'windows' } else { 'amd64' }; return }
if ($args[0] -eq 'tool') { $out=$args[[Array]::IndexOf($args,'-out')+1]; [IO.File]::WriteAllText($out,'fixture resource'); return }
if ($args[0] -eq 'build') {
 if ($env:WHALE_STORAGE_FAIL -eq '1') { $global:LASTEXITCODE=7; return }
 $out=$args[[Array]::IndexOf($args,'-o')+1]; [IO.File]::WriteAllText($out,'fixture payload, never executed'); return
}
if ($args[0] -eq 'run') {
 $out=$args[[Array]::IndexOf($args,'-out')+1]
 [IO.File]::WriteAllText((Join-Path $out 'ghostty-vt.dll'),'fixture terminal')
 [IO.File]::WriteAllText((Join-Path $out 'ghostty-vt.dll.LICENSE'),'fixture license')
 [IO.File]::WriteAllText((Join-Path $out 'terminal-runtime.json'),'{}')
}
`);
  const env = { PATH: nsis + path.delimiter + process.env.PATH, WHALEISLE_PACKAGE_SKIP_RUNTIME: '1', WHALEISLE_PET_RENDER_ASSETS: path.join(temporary, 'no-pet-input'), WHALEISLE_VERSION: '0.0.0-dev' };
  for (const mode of ['stage', 'clean', 'failure']) {
    const output = fresh();
    const result = powershell(`& ${quote(path.join(scripts, 'package.ps1'))} -GoExecutable ${quote(fakeGo)} -OutputRoot ${quote(output)} ${mode === 'stage' ? '-StageOnly' : '-CleanStage'}`, { ...env, WHALEISLE_VERSION: mode === 'clean' ? '1.2.3' : env.WHALEISLE_VERSION, WHALE_STORAGE_FAIL: mode === 'failure' ? '1' : '0' });
    if (mode === 'failure') {
      assert.notEqual(result.status, 0);
      assert.match(result.stderr, /go build failed/);
      assert.equal(fs.existsSync(path.join(output, 'whaleisle')), true);
    } else {
      assert.equal(result.status, 0, result.stdout + result.stderr);
      assert.equal(fs.existsSync(path.join(output, 'whaleisle')), mode === 'stage');
      if (mode === 'clean') {
        const setup = fs.readdirSync(output).find(name => name.endsWith('.exe'));
        assert.ok(setup); assert.ok(fs.statSync(path.join(output, setup)).size > 0);
        assert.equal(fs.existsSync(path.join(output, setup + '.blockmap.json')), true);
        assert.equal(fs.existsSync(path.join(output, 'SHA256SUMS.txt')), true);
      }
    }
  }
});

test('production desktop reports size without enforcing the retired EXE limit', { skip: process.platform !== 'win32' }, () => {
  const fakeGo = path.join(temporary, 'large-build-go.ps1');
  fs.writeFileSync(fakeGo, String.raw`
$global:LASTEXITCODE=0
if ($args[0] -eq 'env') { if ($args[1] -in @('GOOS','GOHOSTOS')) { 'windows' } else { 'amd64' }; return }
if ($args[0] -eq 'tool') { $out=$args[[Array]::IndexOf($args,'-out')+1]; [IO.File]::WriteAllText($out,'fixture resource'); return }
if ($args[0] -eq 'build') {
 $out=$args[[Array]::IndexOf($args,'-o')+1]
 $file=[IO.File]::Create($out)
 try { $file.SetLength(100*1024*1024) } finally { $file.Dispose() }
}
`);
  const output = path.join(temporary, 'large-desktop.exe');
  const result = powershell(`& ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
    & ${quote(path.join(scripts, 'build-release.ps1'))} -Package cmd/whaleisle -Output ${quote(output)} -GoExecutable ${quote(fakeGo)}
  }`);
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.equal(fs.statSync(output).size, 100 * 1024 * 1024);
  assert.match(result.stdout, /Production desktop: 104857600 bytes/);
  assert.equal(Object.hasOwn(JSON.parse(fs.readFileSync(path.join(scripts, 'release-build.json'))), 'windowsAmd64MaxBytes'), false);
});

test('owned-dir cleanup unlinks nested junctions without descending and enforces prefix', () => {
  const parent = fresh();
  const owned = path.join(parent, 'whaleisle-owned-123');
  const nested = path.join(owned, 'deep', 'sub');
  const source = fresh();
  fs.mkdirSync(nested, { recursive: true });
  fs.writeFileSync(path.join(source, 'keep.txt'), 'protected-data');
  fs.symlinkSync(source, path.join(nested, 'ext-link'), process.platform === 'win32' ? 'junction' : 'dir');
  assert.throws(() => removeOwnedDir(owned, parent, 'other-'), /prefixed/);
  removeOwnedDir(owned, parent, 'whaleisle-owned-');
  assert.equal(fs.existsSync(owned), false);
  assert.equal(fs.readFileSync(path.join(source, 'keep.txt'), 'utf8'), 'protected-data');
  removeOwnedDir(owned, parent, 'whaleisle-owned-');
});

const releaseFixtureGo = name => {
  const fakeGo = path.join(temporary, name);
  fs.writeFileSync(fakeGo, String.raw`
$global:LASTEXITCODE=0
if ($args[0] -eq 'env') { if ($args[1] -in @('GOOS','GOHOSTOS')) { 'windows' } else { 'amd64' }; return }
if ($args[0] -eq 'tool') { $out=$args[[Array]::IndexOf($args,'-out')+1]; [IO.File]::WriteAllText($out,'fixture resource'); return }
if ($args[0] -eq 'build') { $out=$args[[Array]::IndexOf($args,'-o')+1]; [IO.File]::WriteAllText($out,'fixture payload, never executed'); return }
`);
  return fakeGo;
};
// The script names its cross-process lock sha256(upper(normalized candidate)):
// an ASCII-only repo path keeps ToUpperInvariant equal to toUpperCase.
const sysoLockPath = candidate => path.join(
  fs.realpathSync.native(os.tmpdir()).replaceAll('\\', '/'),
  'whaleisle-syso-' + crypto.createHash('sha256').update(candidate.toUpperCase(), 'utf8').digest('hex') + '.lock');

test('release build reclaims a stale .syso and removes its resource lock', { skip: process.platform !== 'win32' }, () => {
  const candidate = path.join(path.dirname(scripts), 'cmd', 'whaleisle', 'mygo_windows_amd64.syso');
  const lock = sysoLockPath(candidate);
  fs.rmSync(candidate, { force: true });
  fs.rmSync(lock, { force: true });
  fs.writeFileSync(candidate, 'stale resource from a killed build');
  const output = path.join(temporary, 'stale-syso-desktop.exe');
  try {
    const result = powershell(`& ${quote(path.join(scripts, 'build-release.ps1'))} -Package cmd/whaleisle -Output ${quote(output)} -GoExecutable ${quote(releaseFixtureGo('stale-syso-go.ps1'))}`);
    assert.equal(result.status, 0, result.stdout + result.stderr);
    assert.equal(fs.existsSync(output), true);
    assert.equal(fs.existsSync(candidate), false, 'build left its .syso behind');
    assert.equal(fs.existsSync(lock), false, 'build left its resource lock behind');
  } finally {
    fs.rmSync(candidate, { force: true });
    fs.rmSync(lock, { force: true });
  }
});

test('release build refuses while another process holds the resource lock', { skip: process.platform !== 'win32' }, () => {
  const candidate = path.join(path.dirname(scripts), 'cmd', 'whaleisle', 'mygo_windows_amd64.syso');
  const lock = sysoLockPath(candidate);
  fs.rmSync(candidate, { force: true });
  fs.rmSync(lock, { force: true });
  fs.writeFileSync(candidate, 'sentinel in use by another build');
  const output = path.join(temporary, 'locked-syso-desktop.exe');
  try {
    // The probe holds the same exclusive lock a live build would, then runs
    // a second build: it must fail fast and leave the .syso untouched.
    const probe = powershell(`$candidate = ${quote(candidate)}
$lockPath = ${quote(lock)}
$lock = [IO.File]::Open($lockPath, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
try {
  & ${quote(path.join(scripts, 'build-release.ps1'))} -Package cmd/whaleisle -Output ${quote(output)} -GoExecutable ${quote(releaseFixtureGo('locked-syso-go.ps1'))}
  Write-Output 'BUILD-SUCCEEDED'
} catch {
  Write-Output ('BUILD-FAILED: ' + $_.Exception.Message)
} finally {
  $lock.Dispose()
  Remove-Item -LiteralPath $lockPath -Force -ErrorAction SilentlyContinue
}`);
    assert.match(probe.stdout, /BUILD-FAILED/);
    assert.match(probe.stdout, /resource lock/);
    assert.equal(fs.existsSync(output), false);
    assert.equal(fs.readFileSync(candidate, 'utf8'), 'sentinel in use by another build');
  } finally {
    fs.rmSync(candidate, { force: true });
    fs.rmSync(lock, { force: true });
  }
});

test('env.ps1 pins PSModuleAnalysisCachePath outside cwd and with-build-workspace restores it', { skip: process.platform !== 'win32' }, () => {
  const isolated = fresh();
  const explicitCache = path.join(isolated, 'custom-ps-cache');
  const res = powershell(`
    Remove-Item Env:LOCALAPPDATA -ErrorAction SilentlyContinue
    Remove-Item Env:PSModuleAnalysisCachePath -ErrorAction SilentlyContinue
    . ${quote(path.join(scripts, 'env.ps1'))}
    Write-Output "AUTO=$env:PSModuleAnalysisCachePath"
    $env:PSModuleAnalysisCachePath = ${quote(explicitCache)}
    & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
      param($work)
      $env:PSModuleAnalysisCachePath = Join-Path $work 'inner-cache'
    }
    Write-Output "RESTORED=$env:PSModuleAnalysisCachePath"
  `);
  assert.equal(res.status, 0, res.stderr);
  const autoMatch = res.stdout.match(/AUTO=(.+)/);
  assert.ok(autoMatch);
  assert.equal(path.isAbsolute(autoMatch[1].trim()), true);
  assert.equal(res.stdout.includes(`RESTORED=${explicitCache}`), true);
});
