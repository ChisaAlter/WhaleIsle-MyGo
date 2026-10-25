import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import * as vm from 'node:vm';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {
  applyWindowsBackgroundRuntime, patchBackgroundSource, patches, patchID,
} from './windows-background-runtime.mjs';

const projectRoot = fileURLToPath(new URL('../', import.meta.url));
const temporaryPrefix = 'mygo-windows-background-runtime-test-';
const createdDirectories = new Set();
const sha256 = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const expectedBaseHashes = {
  'dsh-remote/lib/credential.js': '2bdd220dac68da8391692fb03c671915a01cc8a50189b6e77f4efab9f35a6887',
  'dsh-remote/lib/index.js': '817b6d0823e7e340ec85c2408d1f2aa54b3f96209ae0ea05ebd9e530f21404e9',
};
const oldManifest = {
  schemaVersion: 1,
  artifacts: {
    'deepseek-harness/apps/cli/lib/bin.js': '1'.repeat(64),
    'deepseek-harness/packages/client/ui-visual-replies/lib/index.js': '2'.repeat(64),
    'unrelated-fixture-artifact.bin': '3'.repeat(64),
  },
  safetyPatch: {
    id: 'mygo-session-runtime-readiness-v1',
    baseSHA256: '4'.repeat(64),
    patchSHA256: '5'.repeat(64),
    outputSHA256: '2'.repeat(64),
  },
  requiredEventTypes: ['job/registered', 'job/settled'],
  node: {path: 'node/node.exe', sha256: '6'.repeat(64)},
  unrelated: {keep: ['existing', 'provenance']},
};

// Keep the actual vendor bytes, including their line endings. Reverse precisely
// the pinned one-field change; a source imitation is not an adequate fixture.
const fixtures = await Promise.all(patches.map(async patch => {
  const fixed = await fs.readFile(path.join(projectRoot, 'runtime/vendor', patch.file));
  const after = Buffer.from(patch.after);
  const position = fixed.indexOf(after);
  assert.notEqual(position, -1, `missing fixed execFile call in ${patch.file}`);
  assert.equal(fixed.indexOf(after, position + after.length), -1, `ambiguous fixed call in ${patch.file}`);
  assert.equal(patch.after.replace(' windowsHide: true,', ''), patch.before);
  const original = Buffer.concat([
    fixed.subarray(0, position),
    Buffer.from(patch.before),
    fixed.subarray(position + after.length),
  ]);
  assert.equal(sha256(original), expectedBaseHashes[patch.file], `original byte hash for ${patch.file}`);
  assert.equal(sha256(fixed), patch.outputSHA256, `fixed byte hash for ${patch.file}`);
  assert.deepEqual(Buffer.from(original.toString('utf8')), original, 'fixture must round-trip as UTF-8');
  return {patch, original, fixed};
}));
const expectedArtifacts = Object.fromEntries(fixtures.map(({patch}) => [patch.file, patch.outputSHA256]));

async function temporaryDirectory(t) {
  const parent = path.resolve(os.tmpdir());
  const created = path.resolve(await fs.mkdtemp(path.join(parent, temporaryPrefix)));
  assert.equal(path.dirname(created), parent);
  assert.ok(path.basename(created).startsWith(temporaryPrefix));
  const createdRealPath = await fs.realpath(created);
  createdDirectories.add(created);
  t.after(async () => {
    // Resolve and verify the exact directory this helper created before removal.
    const target = path.resolve(created);
    assert.ok(createdDirectories.has(target), 'cleanup target must be our recorded temporary directory');
    assert.equal(path.dirname(target), parent);
    assert.ok(path.basename(target).startsWith(temporaryPrefix));
    assert.equal((await fs.lstat(target)).isSymbolicLink(), false, 'cleanup root must not be replaced by a link');
    assert.equal(await fs.realpath(target), createdRealPath, 'cleanup root must still resolve to the created directory');
    await fs.rm(target, {recursive: true, force: true});
    createdDirectories.delete(target);
  });
  return created;
}

async function stageFixture(t, versions = fixtures.map(() => 'original'), {manifest = oldManifest} = {}) {
  const temporary = await temporaryDirectory(t);
  const vendor = path.join(temporary, 'vendor');
  await fs.mkdir(path.join(vendor, 'dsh-remote/lib'), {recursive: true});
  await fs.writeFile(path.join(vendor, 'dsh-remote/package.json'), '{"name":"dsh-remote","type":"module"}\n');
  for (const [index, fixture] of fixtures.entries()) {
    const version = versions[index];
    if (version !== null) await fs.writeFile(path.join(vendor, fixture.patch.file), fixture[version]);
  }
  const manifestPath = path.join(vendor, 'host-session-runtime.json');
  if (manifest !== null) await fs.writeFile(manifestPath, JSON.stringify(manifest, null, 2) + '\n');
  return {temporary, vendor, manifestPath};
}

async function sourceBytes(vendor) {
  return Promise.all(fixtures.map(({patch}) => fs.readFile(path.join(vendor, patch.file))));
}

async function assertFixedSources(vendor) {
  const actual = await sourceBytes(vendor);
  for (const [index, fixture] of fixtures.entries()) {
    assert.deepEqual(actual[index], fixture.fixed, `exact patched bytes for ${fixture.patch.file}`);
    assert.equal(sha256(actual[index]), fixture.patch.outputSHA256);
  }
}

async function assertManifestExtended(manifestPath) {
  const actual = JSON.parse(await fs.readFile(manifestPath, 'utf8'));
  assert.deepEqual(actual, {
    ...oldManifest,
    artifacts: {...oldManifest.artifacts, ...expectedArtifacts},
    windowsBackgroundPatch: {id: patchID, artifacts: expectedArtifacts},
  });
  assert.deepEqual(actual.safetyPatch, oldManifest.safetyPatch);
  for (const [file, digest] of Object.entries(oldManifest.artifacts)) assert.equal(actual.artifacts[file], digest);
}

test('pins exactly the two real vendor sources and reconstructs their original byte hashes', () => {
  assert.equal(patchID, 'mygo-windows-background-process-v1');
  assert.deepEqual(patches.map(patch => patch.file).sort(), Object.keys(expectedBaseHashes).sort());
  for (const {patch, original, fixed} of fixtures) {
    assert.equal(patch.baseSHA256, expectedBaseHashes[patch.file]);
    assert.match(patch.outputSHA256, /^[a-f0-9]{64}$/);
    assert.equal(sha256(original), patch.baseSHA256);
    assert.equal(sha256(fixed), patch.outputSHA256);
  }
});

for (const fixture of fixtures) {
  const {patch, original, fixed} = fixture;
  test(`patchBackgroundSource repairs and accepts the exact bytes of ${patch.file}`, () => {
    const output = patchBackgroundSource(patch, original.toString('utf8'));
    assert.deepEqual(Buffer.from(output), fixed);
    assert.equal(patchBackgroundSource(patch, output), output);
  });

  test(`patchBackgroundSource rejects unpinned or ambiguous bytes of ${patch.file}`, () => {
    const unknownSources = [
      original.toString('utf8') + '\n// unknown original revision\n',
      fixed.toString('utf8') + '\n// unknown fixed revision\n',
      original.toString('utf8').replace(patch.before, 'execFile(unrecognized, args, {'),
      original.toString('utf8') + '\n' + patch.before,
    ];
    for (const source of unknownSources) {
      assert.throws(() => patchBackgroundSource(patch, source), error => {
        assert.match(error.message, /requires the pinned original or patched bytes/);
        assert.ok(error.message.includes(patch.file));
        return true;
      });
    }
  });
}

test('staging repairs both originals, extends provenance and is byte-idempotent on repeat', async t => {
  const {vendor, manifestPath} = await stageFixture(t);
  const first = await applyWindowsBackgroundRuntime(vendor);
  assert.deepEqual(first, {id: patchID, artifacts: expectedArtifacts});
  await assertFixedSources(vendor);
  await assertManifestExtended(manifestPath);
  const firstManifest = await fs.readFile(manifestPath);
  const firstSources = await sourceBytes(vendor);

  const repeated = await applyWindowsBackgroundRuntime(vendor);
  assert.deepEqual(repeated, first);
  assert.deepEqual(await sourceBytes(vendor), firstSources);
  assert.deepEqual(await fs.readFile(manifestPath), firstManifest);
});

for (const versions of [['fixed', 'fixed'], ['original', 'fixed'], ['fixed', 'original']]) {
  test(`staging accepts pinned ${versions.join('/')} source pairs`, async t => {
    const {vendor, manifestPath} = await stageFixture(t, versions);
    assert.deepEqual(await applyWindowsBackgroundRuntime(vendor), {id: patchID, artifacts: expectedArtifacts});
    await assertFixedSources(vendor);
    await assertManifestExtended(manifestPath);
  });
}

for (const badFixture of fixtures) {
  test(`unknown ${badFixture.patch.file} rejects before either source or manifest changes`, async t => {
    const {vendor, manifestPath} = await stageFixture(t);
    const badPath = path.join(vendor, badFixture.patch.file);
    await fs.writeFile(badPath, Buffer.concat([badFixture.original, Buffer.from('\n// unpinned fixture revision\n')]));
    const before = await sourceBytes(vendor);
    const manifestBefore = await fs.readFile(manifestPath);

    await assert.rejects(applyWindowsBackgroundRuntime(vendor), error => {
      assert.match(error.message, /requires the pinned original or patched bytes/);
      assert.ok(error.message.includes(badFixture.patch.file));
      return true;
    });
    assert.deepEqual(await sourceBytes(vendor), before, 'both source files must survive failed preflight byte-for-byte');
    assert.deepEqual(await fs.readFile(manifestPath), manifestBefore);
  });
}

test('a missing second source rejects before the first source or manifest changes', async t => {
  const {vendor, manifestPath} = await stageFixture(t, ['original', null]);
  const firstPath = path.join(vendor, fixtures[0].patch.file);
  const before = await fs.readFile(firstPath);
  const manifestBefore = await fs.readFile(manifestPath);
  await assert.rejects(applyWindowsBackgroundRuntime(vendor), {code: 'ENOENT'});
  assert.deepEqual(await fs.readFile(firstPath), before);
  assert.deepEqual(await fs.readFile(manifestPath), manifestBefore);
});

test('a linked lib directory outside the copied vendor is rejected without changing its targets', async t => {
  const temporary = await temporaryDirectory(t);
  const vendor = path.join(temporary, 'vendor');
  const remote = path.join(vendor, 'dsh-remote');
  const outside = path.join(temporary, 'outside-vendor');
  await fs.mkdir(remote, {recursive: true});
  await fs.mkdir(outside);
  await fs.writeFile(path.join(remote, 'package.json'), '{"name":"dsh-remote"}\n');
  for (const fixture of fixtures) await fs.writeFile(path.join(outside, path.basename(fixture.patch.file)), fixture.original);
  const manifestPath = path.join(vendor, 'host-session-runtime.json');
  await fs.writeFile(manifestPath, JSON.stringify(oldManifest) + '\n');
  const manifestBefore = await fs.readFile(manifestPath);
  const link = path.join(remote, 'lib');
  // Junctions exercise realpath containment on Windows without symlink privileges.
  await fs.symlink(outside, link, process.platform === 'win32' ? 'junction' : 'dir');
  assert.equal((await fs.lstat(link)).isSymbolicLink(), true);
  assert.equal(await fs.realpath(link), await fs.realpath(outside));

  await assert.rejects(applyWindowsBackgroundRuntime(vendor), /target escapes the copied runtime/);
  for (const fixture of fixtures) {
    assert.deepEqual(await fs.readFile(path.join(outside, path.basename(fixture.patch.file))), fixture.original);
  }
  assert.deepEqual(await fs.readFile(manifestPath), manifestBefore);
});

test('a tiny stager without dsh-remote returns empty and leaves existing provenance intact', async t => {
  const temporary = await temporaryDirectory(t);
  const vendor = path.join(temporary, 'vendor');
  await fs.mkdir(path.join(vendor, 'deepseek-harness'), {recursive: true});
  const manifestPath = path.join(vendor, 'host-session-runtime.json');
  const manifestBefore = Buffer.from(JSON.stringify(oldManifest) + '\n');
  await fs.writeFile(manifestPath, manifestBefore);
  const entriesBefore = await fs.readdir(vendor);

  assert.deepEqual(await applyWindowsBackgroundRuntime(vendor), {id: patchID, artifacts: {}});
  assert.deepEqual(await fs.readFile(manifestPath), manifestBefore);
  assert.deepEqual(await fs.readdir(vendor), entriesBefore);
  await assert.rejects(fs.stat(path.join(vendor, 'dsh-remote')), {code: 'ENOENT'});
});

test('pinned sources can stage without creating an absent host manifest', async t => {
  const {vendor, manifestPath} = await stageFixture(t, undefined, {manifest: null});
  assert.deepEqual(await applyWindowsBackgroundRuntime(vendor), {id: patchID, artifacts: expectedArtifacts});
  await assertFixedSources(vendor);
  await assert.rejects(fs.stat(manifestPath), {code: 'ENOENT'});
});

test('real Windows getSecret hides its mocked PowerShell and returns the decoded output', {
  skip: typeof vm.SourceTextModule !== 'function' ? 'run with --experimental-vm-modules for the isolated behavior test' : false,
}, async t => {
  const temporary = await temporaryDirectory(t);
  const secretsDir = path.join(temporary, 'mock-secrets-never-created');
  const encodedFixture = 'Zml4dHVyZS1vbmx5LWJsb2I=';
  const reads = [];
  const calls = [];
  const context = vm.createContext({process: Object.freeze({platform: 'win32'})});
  const fakeExecFile = (bin, args, options, callback) => {
    calls.push({bin, args: Array.from(args), options: {...options}});
    callback(null, '  fixture decoded output\r\n', '');
  };
  const fakeReadFileSync = (filename, encoding) => {
    reads.push({filename, encoding});
    assert.equal(filename, path.join(secretsDir, 'fixture-machine.bin'));
    assert.equal(encoding, 'utf8');
    return `  ${encodedFixture}\n`;
  };
  const forbiddenFsOperation = () => assert.fail('getSecret must only read the in-memory fake .bin file');
  const mockedModules = new Map([
    ['node:child_process', {execFile: fakeExecFile}],
    ['node:fs', {
      mkdirSync: forbiddenFsOperation,
      writeFileSync: forbiddenFsOperation,
      readFileSync: fakeReadFileSync,
      unlinkSync: forbiddenFsOperation,
    }],
    ['node:path', {join: path.join}],
  ]);
  const credentialFixture = fixtures.find(({patch}) => patch.file === 'dsh-remote/lib/credential.js');
  const module = new vm.SourceTextModule(credentialFixture.fixed.toString('utf8'), {
    context,
    identifier: pathToFileURL(path.join(projectRoot, 'runtime/vendor', credentialFixture.patch.file)).href,
  });
  await module.link(specifier => {
    assert.ok(mockedModules.has(specifier), `unexpected real-module import: ${specifier}`);
    const exports = mockedModules.get(specifier);
    return new vm.SyntheticModule(Object.keys(exports), function () {
      for (const [name, value] of Object.entries(exports)) this.setExport(name, value);
    }, {context, identifier: `mock:${specifier}`});
  });
  await module.evaluate();

  assert.equal(await module.namespace.getSecret('fixture-machine', secretsDir), 'fixture decoded output');
  assert.equal(reads.length, 1);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].bin, 'powershell');
  assert.deepEqual(calls[0].args.slice(0, 3), ['-NoProfile', '-STA', '-Command']);
  assert.ok(calls[0].args[3].includes(`FromBase64String('${encodedFixture}')`));
  assert.deepEqual(calls[0].options, {windowsHide: true, timeout: 8000, maxBuffer: 1 << 20});
  await assert.rejects(fs.stat(secretsDir), {code: 'ENOENT'}, 'the fake credential directory must never be created');
});
