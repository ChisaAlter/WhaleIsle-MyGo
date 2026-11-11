// Only mocked readiness calls and temporary copies of the pinned compiled bundle.
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import vm from 'node:vm';
import crypto from 'node:crypto';
import {applySessionRuntimeReadiness, patchSessionReadinessSource, replacement,
  readinessProfile, guardID, bundlePath, recordFile} from './session-runtime-readiness.mjs';

const filename = process.argv[2];
const version = process.argv[3] || '0.2.1-rc.2';
const {baseSHA256, outputSHA256, patchSHA256} = readinessProfile(version);
assert.ok(filename, 'Pass the read-only installed visual-reply bundle and optional Host version');
const source = await fs.readFile(filename, 'utf8');
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
assert.equal(hash(source), baseSHA256);
const output = patchSessionReadinessSource(source, version);
assert.equal(hash(output), outputSHA256);
assert.equal(output.split('await requireEnabled(config, exec.signal);').length - 1, 7);
assert.throws(() => patchSessionReadinessSource(source + '\n', version), /exact rc\.[24]/);

function gate({enabled = true, env = true, fetchImpl = async () => ({ok: true, json: async () => ({available: true})})} = {}) {
  const calls = [];
  const timeouts = [];
  const config = {enabled: {get: () => enabled}};
  const scope = vm.createContext({URL, Error, process: {env: env ? {
    DSH_DESKTOP_INSTALL_URL: 'http://127.0.0.1:1', DSH_DESKTOP_INSTALL_TOKEN: 'fixture-token',
  } : {}}, AbortSignal: {timeout(ms) {timeouts.push(ms); return new AbortController().signal;},
    any(signals) {assert.equal(signals.length, 2); return signals[0];}},
  fetch: async (url, options) => {calls.push({url: url.href, options}); return fetchImpl(config, options, () => {enabled = false;});}});
  vm.runInContext(replacement + '\nglobalThis.check = requireEnabled;', scope);
  return {check: signal => scope.check(config, signal), calls, timeouts};
}

let test = gate({enabled: false});
await assert.rejects(test.check(new AbortController().signal), /Visual replies are disabled/);
assert.equal(test.calls.length, 0);
test = gate({env: false});
await assert.rejects(test.check(new AbortController().signal), /runtime is not verified/);
assert.equal(test.calls.length, 0);
test = gate();
const signal = new AbortController().signal;
await test.check(signal);
await test.check(signal);
assert.equal(test.calls.length, 2, 'Readiness must be checked per operation, never cached');
assert.deepEqual(test.timeouts, [3000, 3000]);
for (const call of test.calls) {
  assert.equal(call.url, 'http://127.0.0.1:1/desktop/html-preview/status');
  assert.equal(call.options.method, 'GET');
  assert.equal(call.options.headers.authorization, 'Bearer fixture-token');
  assert.equal(call.options.signal, signal);
  assert.equal(call.options.redirect, 'error');
  assert.equal(call.options.cache, 'no-store');
}
for (const response of [{ok: false, available: true}, {ok: true, available: false}]) {
  test = gate({fetchImpl: async () => ({ok: response.ok, json: async () => ({available: response.available, reason: 'fixture denied'})})});
  await assert.rejects(test.check(signal), /fixture denied/);
}
test = gate({fetchImpl: async (_config, _options, disable) => ({ok: true, json: async () => {disable(); return {available: true};}})});
await assert.rejects(test.check(signal), /disabled while checking/);
const controller = new AbortController();
test = gate({fetchImpl: async () => ({ok: true, json: async () => {controller.abort(); return {available: true};}})});
await assert.rejects(test.check(controller.signal), error => error.name === 'AbortError');
const alreadyAborted = new AbortController();
alreadyAborted.abort();
test = gate({fetchImpl: async (_config, options) => {options.signal.throwIfAborted(); throw new Error('Unreachable');}});
await assert.rejects(test.check(alreadyAborted.signal), error => error.name === 'AbortError');

const root = await fs.mkdtemp(path.join(os.tmpdir(), 'mygo-session-readiness-'));
try {
  const sourceVendor = path.join(root, 'source');
  const targetVendor = path.join(root, 'target');
  const original = path.join(sourceVendor, bundlePath);
  const target = path.join(targetVendor, bundlePath);
  await fs.mkdir(path.dirname(original), {recursive: true});
  await fs.mkdir(path.dirname(target), {recursive: true});
  await fs.writeFile(original, source);
  await fs.writeFile(target, source);
  await assert.rejects(applySessionRuntimeReadiness(sourceVendor, sourceVendor), /disjoint/);
  await assert.rejects(applySessionRuntimeReadiness(path.dirname(target), sourceVendor), /ENOENT/);
  const record = await applySessionRuntimeReadiness(targetVendor, sourceVendor, version);
  assert.deepEqual(record, {id: guardID, baseSHA256, patchSHA256, outputSHA256});
  assert.equal(hash(await fs.readFile(target)), outputSHA256);
  assert.equal(hash(await fs.readFile(original)), baseSHA256, 'Read-only source must stay unchanged');
  assert.deepEqual(await applySessionRuntimeReadiness(targetVendor, sourceVendor, version), record);
  assert.deepEqual(JSON.parse(await fs.readFile(path.join(targetVendor, recordFile), 'utf8')), record);
  await fs.appendFile(target, '\n');
  await assert.rejects(applySessionRuntimeReadiness(targetVendor, sourceVendor, version), /record does not match/);
  await fs.writeFile(target, output);
  await fs.writeFile(path.join(targetVendor, recordFile), JSON.stringify({...record, outputSHA256: '0'.repeat(64)}));
  await assert.rejects(applySessionRuntimeReadiness(targetVendor, sourceVendor, version), /record does not match/);
} finally {
  // This directory was created above and is confined to the temporary workspace.
  const resolved = path.resolve(root);
  assert.equal(path.dirname(resolved), path.resolve(os.tmpdir()));
  assert.ok(path.basename(resolved).startsWith('mygo-session-readiness-'));
  await fs.rm(resolved, {recursive: true, force: true});
}
console.log(JSON.stringify({result: 'PASS', guardID, baseSHA256, outputSHA256,
  checks: '7 await sites; per-operation status/auth/3s/cancel/disabled guards; exact hashes; disjoint copy; idempotence/tampering rejection; source unchanged; no real fetch/tool/Host'}));
