import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { after, test } from 'node:test';
import { fileURLToPath } from 'node:url';
import { removeBuildWorkspace, removePackageStage } from './build-storage.mjs';

const scripts = path.dirname(fileURLToPath(import.meta.url));
const temporary = fs.mkdtempSync(path.join(fs.realpathSync.native(os.tmpdir()), 'whale-storage-test-'));
after(() => fs.rmSync(temporary, { recursive: true, force: true }));
let sequence = 0;
const fresh = () => fs.mkdtempSync(path.join(temporary, 'case-'));
function powershell(source, env = {}) {
  const script = path.join(temporary, `probe-${sequence++}.ps1`);
  fs.writeFileSync(script, '$ErrorActionPreference = "Stop"\n' + source);
  return spawnSync('pwsh', ['-NoProfile', '-File', script], { cwd: temporary, env: { ...process.env, ...env }, encoding: 'utf8', windowsHide: true });
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
  assert.match(shared.stdout, /PIN=go1\.27\.2\+auto/);
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

test('real Go builds share native caches across two disposable workspaces', { skip: process.platform !== 'win32' }, () => {
  const source = path.join(temporary, 'storage-smoke.go');
  fs.writeFileSync(source, 'package main\nimport "fmt"\nfunc main() { fmt.Println("storage-smoke-ok") }\n');
  const result = powershell(`foreach ($key in @('GOPATH','GOCACHE','GOMODCACHE','GOTMPDIR')) { Remove-Item "Env:$key" -ErrorAction SilentlyContinue }
. ${quote(path.join(scripts, 'env.ps1'))}
$cacheBefore = go env GOCACHE
$modulesBefore = go env GOMODCACHE
foreach ($iteration in 1..2) {
  & ${quote(path.join(scripts, 'with-build-workspace.ps1'))} {
    param($work)
    $exe = Join-Path $work 'smoke.exe'
    go build -o $exe ${quote(source)}
    if ($LASTEXITCODE -ne 0) { throw 'Real Go compilation failed' }
    & $exe
    [IO.File]::AppendAllText(${quote(path.join(temporary, 'go-workspaces.txt'))}, $work + [Environment]::NewLine)
  }
}
if ((go env GOCACHE) -ne $cacheBefore -or (go env GOMODCACHE) -ne $modulesBefore) { throw 'Shared cache paths changed' }
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
if ($args[0] -eq 'env') { if ($args[1] -eq 'GOOS') { 'windows' } else { 'amd64' }; return }
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
    const result = powershell(`& ${quote(path.join(scripts, 'package.ps1'))} -GoExecutable ${quote(fakeGo)} -OutputRoot ${quote(output)} ${mode === 'stage' ? '-StageOnly' : '-CleanStage'}`, { ...env, WHALE_STORAGE_FAIL: mode === 'failure' ? '1' : '0' });
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
