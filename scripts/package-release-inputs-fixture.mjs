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
assert.match(windows, /Test-Path \(Join-Path \$NodeSrc 'node.exe'\) -PathType Leaf/);
assert.match(windows, /Get-FileHash \$StagedNode -Algorithm SHA256/);
assert.ok(windows.indexOf('Packaged Host node.exe is missing or changed.') < windows.indexOf("Set-Content -Path $Marker"));
assert.ok(windows.indexOf('Packaged mobile runtime verification failed') < windows.indexOf("Set-Content -Path $Marker"));
assert.match(windows, /throw 'Host Node runtime missing/);
assert.doesNotMatch(windows, /Note: Node directory not found/);
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
