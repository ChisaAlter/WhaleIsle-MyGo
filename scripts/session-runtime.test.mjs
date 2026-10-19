// Tiny synthetic module graphs test the stager, not installed provenance. The
// one real visual bundle is read as bytes from an explicit fixture path solely
// to exercise the exact pinned readiness patch; it is never imported/executed.
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import {inspectSessionRuntime, stageSessionRuntime, cloneStagedSessionRuntime, verifyStagedSessionRuntime, sessionAdmissionTypes} from './session-runtime.mjs';
import {outputSHA256, bundlePath} from './session-runtime-readiness.mjs';

const pin = {repo: 'https://github.com/deepseek-ai/deepseek-harness.git', ref: 'dsh-v0.2.1-rc.2', sha: '687f282e8625c6b8d11ff58211a90524be2e4317', npm: '0.2.1-rc.2'};
async function write(filename, contents) {
  await fs.mkdir(path.dirname(filename), {recursive: true});
  await fs.writeFile(filename, typeof contents === 'string' ? contents : JSON.stringify(contents));
}
async function fixture(t, {coreTypes = sessionAdmissionTypes, readerTypes, allowUnknown = false, version = pin.npm} = {}) {
  const temporaryRoot = await fs.realpath(os.tmpdir());
  const root = await fs.mkdtemp(path.join(temporaryRoot, 'session-runtime-fixture-'));
  t.after(async () => {
    const relative = path.relative(temporaryRoot, root);
    assert.ok(!path.isAbsolute(relative) && relative.startsWith('session-runtime-fixture-') && !relative.includes(path.sep));
    await fs.rm(root, {recursive: true, force: true});
  });
  const resources = path.join(root, 'resources');
  const vendor = path.join(resources, 'vendor');
  const harness = path.join(vendor, 'deepseek-harness');
  for (const relative of ['package.json', 'apps/cli/package.json', 'packages/core/session/package.json', 'packages/session/session-persistence/package.json']) {
    await write(path.join(harness, relative), {type: 'module', version, description: 'Synthetic compatibility tool fixture'});
  }
  await write(path.join(harness, 'apps/cli/lib/bin.js'), '// Synthetic fixture; never executed.');
  await write(path.join(harness, 'packages/core/session/lib/index.js'), `export const KNOWN_SESSION_EVENT_TYPES = new Set(${JSON.stringify(coreTypes)});`);
  await write(path.join(harness, 'packages/session/session-persistence/lib/index.js'), `
    import {KNOWN_SESSION_EVENT_TYPES} from '../../../core/session/lib/index.js';
    const known = ${readerTypes === undefined ? 'KNOWN_SESSION_EVENT_TYPES' : `new Set(${JSON.stringify(readerTypes)})`};
    export function validateStoredEvents(meta, events) {
      for (const event of events) if (!known.has(event.type) && event.ignorable !== true && ${!allowUnknown}) {
        const error = new Error('session "' + meta.id + '" contains event type "' + event.type + '" (seq ' + event.seq + ') unknown to this harness and not marked ignorable; refusing to interpret the log');
        error.name = 'SessionFormatUnsupportedError'; throw error;
      }
      return events;
    }`);
  await write(path.join(vendor, 'harness-upstream.json'), pin);
  await write(path.join(resources, 'runtime/primary-runtime/dependencies/node/bin/node.exe'), 'Synthetic Node bytes; never executed.');
  return {root, resources, vendor, harness, destination: path.join(root, 'staged')};
}
async function addVisualBytes(f) {
  const source = process.env.WHALEISLE_SESSION_RUNTIME_VISUAL_FIXTURE;
  assert.ok(source, 'Set WHALEISLE_SESSION_RUNTIME_VISUAL_FIXTURE to the explicitly selected rc.2 visual bundle');
  const target = path.join(f.vendor, bundlePath);
  await fs.mkdir(path.dirname(target), {recursive: true});
  await fs.copyFile(source, target);
}
async function linkDirectory(target, link) {
  await fs.mkdir(path.dirname(link), {recursive: true});
  await fs.symlink(target, link, process.platform === 'win32' ? 'junction' : 'dir');
}

// The explicit guarded fixture supplies only pinned visual bundle/record bytes;
// all validator, Node and lifecycle bytes remain the tiny synthetic fixture.
async function cloneFixture(t) {
  const f=await fixture(t);
  const stage=process.env.WHALEISLE_SESSION_RUNTIME_STAGED_FIXTURE;
  assert.ok(stage,'Set WHALEISLE_SESSION_RUNTIME_STAGED_FIXTURE to the verified rc.2 stage');
  const real=JSON.parse(await fs.readFile(path.join(stage,'vendor/host-session-runtime.json'),'utf8'));
  await fs.mkdir(path.dirname(path.join(f.vendor,bundlePath)),{recursive:true});
  await fs.copyFile(path.join(stage,'vendor',bundlePath),path.join(f.vendor,bundlePath));
  await fs.copyFile(path.join(stage,'vendor/session-runtime-readiness.json'),path.join(f.vendor,'session-runtime-readiness.json'));
  const node=await fs.readFile(path.join(f.resources,'runtime/primary-runtime/dependencies/node/bin/node.exe'));
  await write(path.join(f.resources,'node/node.exe'),node.toString());
  const inspected=await inspectSessionRuntime(f.harness);
  await write(path.join(f.vendor,'host-session-runtime.json'),{...inspected,safetyPatch:real.safetyPatch,
    node:{path:'node/node.exe',sha256:crypto.createHash('sha256').update(node).digest('hex')},relocatedLinks:0});
  return f;
}

test('clone keeps guarded bytes and relocates links independently of the old stage',async t=>{
  const f=await cloneFixture(t);
  await linkDirectory(path.join(f.harness,'packages/core/session'),path.join(f.vendor,'session-alias'));
  const sourceManifest=await fs.readFile(path.join(f.vendor,'host-session-runtime.json'),'utf8');
  const result=await cloneStagedSessionRuntime(f.resources,f.destination);
  assert.equal(result.harnessRoot,path.join(f.destination,'vendor/deepseek-harness'));
  assert.equal(result.relocatedLinks,1);
  assert.equal(await fs.realpath(path.join(f.destination,'vendor/session-alias')),path.join(f.destination,'vendor/deepseek-harness/packages/core/session'));
  assert.equal(await fs.readFile(path.join(f.vendor,'host-session-runtime.json'),'utf8'),sourceManifest);
  assert.deepEqual(await fs.readFile(path.join(f.destination,'vendor',bundlePath)),await fs.readFile(path.join(f.vendor,bundlePath)));
  await fs.rename(f.resources,path.join(f.root,'old-stage-unavailable'));
  assert.equal((await verifyStagedSessionRuntime(path.join(f.destination,'vendor'))).cliVersion,pin.npm);
});

test('clone refuses existing, overlapping and external-link destinations without changing source',async t=>{
  const f=await cloneFixture(t);
  await fs.mkdir(f.destination);
  await write(path.join(f.destination,'keep.txt'),'keep');
  await assert.rejects(cloneStagedSessionRuntime(f.resources,f.destination),/must be fresh/);
  assert.equal(await fs.readFile(path.join(f.destination,'keep.txt'),'utf8'),'keep');
  await assert.rejects(cloneStagedSessionRuntime(f.resources,path.join(f.resources,'nested')),/overlaps source stage/);
  const external=path.join(f.root,'outside');await fs.mkdir(external);
  await linkDirectory(external,path.join(f.vendor,'external-link'));
  const fresh=path.join(f.root,'never-created');
  await assert.rejects(cloneStagedSessionRuntime(f.resources,fresh),/External or ancestor runtime link/);
  await assert.rejects(fs.lstat(fresh),{code:'ENOENT'});
});

test('old runtime rejects the required seq20 event through its actual validator export', async t => {
  const f = await fixture(t, {version: '0.2.1-alpha.1', coreTypes: []});
  await assert.rejects(inspectSessionRuntime(f.harness), /rejects required event job\/registered:.*seq 20/);
});
test('new runtime admits all five required types and retains strict unknown rejection', async t => {
  const f = await fixture(t);
  const metadata = await inspectSessionRuntime(f.harness);
  assert.deepEqual(metadata.requiredEventTypes, sessionAdmissionTypes);
  assert.equal(metadata.unknownRequiredRejected, true);
  assert.equal(metadata.cliVersion, pin.npm);
  assert.match(metadata.artifacts['deepseek-harness/packages/core/session/lib/index.js'], /^[a-f0-9]{64}$/);
});
test('a reader that accepts unknown required events is refused', async t => {
  const f = await fixture(t, {allowUnknown: true});
  await assert.rejects(inspectSessionRuntime(f.harness), /does not strictly reject unknown required events/);
});
test('rc2 labels and a new core vocabulary cannot conceal an old persistence reader', async t => {
  const f = await fixture(t, {readerTypes: []});
  await assert.rejects(inspectSessionRuntime(f.harness), /rejects required event job\/registered/);
});
test('missing package version metadata is refused before runtime admission', async t => {
  const f = await fixture(t);
  await write(path.join(f.harness, 'apps/cli/package.json'), {type: 'module'});
  await assert.rejects(inspectSessionRuntime(f.harness), /package version metadata is missing/);
});
test('fresh destination and source overlap are checked without creating links', async t => {
  const f = await fixture(t);
  await fs.mkdir(f.destination);
  await write(path.join(f.destination, 'preserve.txt'), 'existing destination');
  await assert.rejects(stageSessionRuntime(f.resources, f.destination), /destination must be fresh/);
  assert.equal(await fs.readFile(path.join(f.destination, 'preserve.txt'), 'utf8'), 'existing destination');
  await assert.rejects(stageSessionRuntime(f.resources, path.join(f.resources, 'nested-copy')), /overlaps source resources/);
  await assert.rejects(fs.lstat(path.join(f.resources, 'nested-copy')), {code: 'ENOENT'});
});
test('stage copies a complete tiny graph, relocates internal links and verifies the guarded result', async t => {
  const f = await fixture(t);
  await addVisualBytes(f);
  const originalBundle = await fs.readFile(path.join(f.vendor, bundlePath));
  const sourceCore = path.join(f.harness, 'packages/core/session');
  const coreLink = path.join(f.harness, 'node_modules/@deepseek-ai/dsh-session');
  await linkDirectory(sourceCore, coreLink);
  await write(path.join(f.harness, '.dsh-runtime-links.json'), {version: 1, links: [{path: 'node_modules/@deepseek-ai/dsh-session', target: 'packages/core/session'}]});
  await write(path.join(f.vendor, 'dsh-fixture/package.json'), {name: 'same-batch-desktop-plugin'});
  const result = await stageSessionRuntime(f.resources, f.destination);
  assert.equal(result.relocatedLinks, 1);
  assert.equal(result.safetyPatch.outputSHA256, outputSHA256);
  assert.deepEqual(result.harnessPin, pin);
  assert.equal(result.artifacts[bundlePath], outputSHA256);
  const stagedVendor = path.join(f.destination, 'vendor');
  const relocated = path.join(stagedVendor, 'deepseek-harness/node_modules/@deepseek-ai/dsh-session');
  assert.equal(await fs.realpath(relocated), await fs.realpath(path.join(stagedVendor, 'deepseek-harness/packages/core/session')));
  assert.deepEqual(await fs.readFile(path.join(f.vendor, bundlePath)), originalBundle, 'source visual bundle changed');
  assert.equal((await fs.readFile(path.join(stagedVendor, 'dsh-fixture/package.json'), 'utf8')).includes('same-batch'), true);
  assert.deepEqual(await verifyStagedSessionRuntime(stagedVendor), result);
  // Mutating the copy proves the bytes were copied, not hard-linked to source.
  await fs.writeFile(path.join(stagedVendor, 'dsh-fixture/package.json'), 'changed target');
  assert.equal((await fs.readFile(path.join(f.vendor, 'dsh-fixture/package.json'), 'utf8')).includes('same-batch'), true);
  await fs.appendFile(path.join(stagedVendor, 'deepseek-harness/packages/core/session/lib/index.js'), '\n// tampered');
  await assert.rejects(verifyStagedSessionRuntime(stagedVendor), /artifact hash mismatch/);
  await assert.rejects(stageSessionRuntime(f.resources, f.destination), /destination must be fresh/);
});
test('external links and manifest disagreement stop staging before a destination exists', async t => {
  const f = await fixture(t);
  const outside = path.join(f.root, 'outside');
  await fs.mkdir(outside);
  const link = path.join(f.harness, 'node_modules/external');
  await linkDirectory(outside, link);
  await assert.rejects(stageSessionRuntime(f.resources, f.destination), /External or ancestor runtime link/);
  await assert.rejects(fs.lstat(f.destination), {code: 'ENOENT'});
  await fs.unlink(link);
  await linkDirectory(path.join(f.harness, 'packages/core/session'), link);
  await write(path.join(f.harness, '.dsh-runtime-links.json'), {version: 1, links: [{path: 'node_modules/external', target: 'packages/session/session-persistence'}]});
  await assert.rejects(stageSessionRuntime(f.resources, f.destination), /manifest disagrees/);
  await assert.rejects(fs.lstat(f.destination), {code: 'ENOENT'});
});
test('verification refuses a changed readiness record even when event admission still passes', async t => {
  const f = await fixture(t);
  await addVisualBytes(f);
  await stageSessionRuntime(f.resources, f.destination);
  const vendor = path.join(f.destination, 'vendor');
  const filename = path.join(vendor, 'host-session-runtime.json');
  const manifest = JSON.parse(await fs.readFile(filename, 'utf8'));
  manifest.safetyPatch.baseSHA256 = '0'.repeat(64);
  await write(filename, manifest);
  await assert.rejects(verifyStagedSessionRuntime(vendor), /readiness record mismatch/);
});
