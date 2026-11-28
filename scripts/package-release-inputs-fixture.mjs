// Narrow offline source-contract fixture. No build, packaging, target binary,
// Host, ORT, model, installer, network or user data is executed.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
const root = fileURLToPath(new URL('../', import.meta.url));
const read = name => fs.readFileSync(path.join(root, name), 'utf8');
const windows = read('scripts/package.ps1');
const nsis = read('scripts/nsis/whaleisle.nsi');
const mac = read('scripts/package-macos.mjs');
assert.match(windows, /\[switch\]\$StageOnly/);
assert.match(windows, /\[string\]\$GoExecutable = 'go'/);
assert.doesNotMatch(windows, /& go (?:build|run|env)/);
assert.match(windows, /& \$BuildRelease -Package 'cmd\/whaleisle' -Output \$ExeOut/);
const buildContract = JSON.parse(read('scripts/release-build.json'));
assert.deepEqual(buildContract.tags, ['release']);
assert.deepEqual(buildContract.ldflags, ['-s', '-w']);
assert.match(windows, /finalize-windows-release\.ps1/);
const stageOnly = windows.indexOf('if ($StageOnly)');
assert.ok(stageOnly > windows.indexOf('Write-Host "Wrote $Readme"'));
assert.ok(stageOnly < windows.indexOf('$MakensisCmd = Get-Command makensis'));
assert.match(windows.slice(stageOnly, windows.indexOf('$Nsi =', stageOnly)), /\breturn\b/);
assert.match(windows, /Test-Path \(Join-Path \$NodeSrc 'node.exe'\) -PathType Leaf/);
assert.match(windows, /Get-FileHash \$StagedNode -Algorithm SHA256/);
assert.ok(windows.indexOf('Packaged Host node.exe is missing or changed.') < windows.indexOf("Set-Content -Path $Marker"));
assert.ok(windows.indexOf('Packaged mobile runtime verification failed') < windows.indexOf("Set-Content -Path $Marker"));
assert.match(windows, /throw 'Host Node runtime missing/);
assert.doesNotMatch(windows, /Note: Node directory not found/);
// The staged README is user-facing release copy: it must describe the current
// independent data directory, never the retired shared-userData contract.
assert.doesNotMatch(windows, /sharing `%APPDATA%\\Deepseek-Harness-Desktop` userData/);
assert.match(windows, /%APPDATA%\\WhaleIsle-MyGo\\data/);
assert.doesNotMatch(windows, /Remove-Item -Recurse -Force \$Stage/);
assert.match(windows, /Package stage already exists; choose a fresh -OutputRoot/);
assert.match(windows, /session-runtime\.mjs'\) release/);
assert.match(nsis, /!ifdef RUNTIME_INCLUDE/);
assert.ok(windows.indexOf("'pet-render-assets.mjs'") < windows.indexOf("if ($env:WHALEISLE_PACKAGE_SKIP_RUNTIME -ne '1')"));
assert.match(windows, /throw 'THA4 resource staging failed/);
assert.match(windows, /\$NsisArguments \+= '\/DSHELL_ONLY=1'/);
assert.match(nsis, /!ifndef SHELL_ONLY/);
assert.match(nsis, /!error "Required Host runtime\/node\/node.exe is missing/);
assert.match(nsis, /!error "THA4 resources not staged/);
assert.match(mac, /config\.includePetAssets === true \|\| process\.env\.WHALEISLE_INCLUDE_PET_ASSETS === '1'/);
assert.match(mac, /run\(process.execPath, \[path.join\(projectRoot, 'scripts\/pet-render-assets.mjs'\), petAssets, bin\]\)/);
const manifest = JSON.parse(read('internal/desktop/pet/assets/render-manifest.json'));
assert.equal(manifest.commit, '90465b67ec16765631d54c195af19d9ff3c3a770');
for (const {path: file} of manifest.files) {
  assert.ok(nsis.includes('File "${STAGE_DIR}\\pet-render-assets\\' + file.replaceAll('/', '\\') + '"'), file);
}
assert.ok(nsis.includes('File "${STAGE_DIR}\\pet-render-assets\\provenance.json"'));
// Exercise the real existing stager's failure path with empty local inputs.
// It must fail before writing a success marker; no supplied asset is executed.
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-release-inputs-'));
try {
  const source = path.join(temporary, 'source'), stage = path.join(temporary, 'stage');
  fs.mkdirSync(source); fs.mkdirSync(stage);
  const result = spawnSync(process.execPath, [path.join(root, 'scripts/pet-render-assets.mjs'), source, stage], {encoding: 'utf8', windowsHide: true});
  assert.equal(result.status, 1);
  assert.equal(fs.existsSync(path.join(stage, '.pet-render-staged')), false);
  assert.equal(fs.existsSync(path.join(stage, 'pet-render-assets')), false);
} finally { fs.rmSync(temporary, {recursive: true, force: true}); }
console.log('Release-input source contract and missing-THA4 staging fixture passed; Windows/native packaging not exercised.');

// Runtime pin single-source contract: the pin file is the only place that
// names the product runtime version, the scripted consumers read it instead
// of hardcoding stage directories, and the staging tool's pin table must
// actually know the pinned version.
const pin = JSON.parse(read('scripts/runtime-pin.json'));
assert.match(pin.version, /^0\.2\.2-rc\./);
assert.ok(Array.isArray(pin.stageDirNames) && pin.stageDirNames.length >= 1);
const sessionRuntime = read('scripts/session-runtime.mjs');
assert.ok(sessionRuntime.includes(`'${pin.version}'`), `session-runtime.mjs pin table does not know ${pin.version}`);
const perf = read('scripts/perf-interaction.ps1');
const uiCompare = read('scripts/ui-compare/run.mjs');
assert.match(perf, /runtime-pin\.json/);
assert.match(uiCompare, /runtime-pin\.json/);
assert.match(windows, /runtime-pin\.json/);
assert.doesNotMatch(windows, /@\('runtime-0\.2\.2-rc5', 'runtime-0\.2\.2-rc2'\)/);
console.log('Runtime pin single-source contract passed.');
