import test from 'node:test';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { loadContract, verifyComposition, diffComposition } from './session-runtime-composition.mjs';

const compositionFile = 'packages/bundle/base/cordis.patch.yml';
const hash = value => crypto.createHash('sha256').update(value).digest('hex');

const upstream = [
  '- insert:',
  '    - id: alpha',
  "      name: '@deepseek-ai/dsh-alpha'",
  '',
  '    - id: beta',
  "      name: '@deepseek-ai/dsh-beta'",
  '      config:',
  '        openAt: never',
  '',
].join('\n');

const stagedBody = [
  '- insert:',
  '    - id: native-alpha',
  "      name: '@deepseek-ai/dsh-alpha'",
  '',
  '    - id: beta',
  "      name: '@deepseek-ai/dsh-beta'",
  '      config:',
  '        openAt: never',
  '        preparedSessionCacheSize: 1',
  '',
  '    - id: mcp-servers-file',
  "      name: '@deepseek-ai/dsh-mcp-servers-file'",
  '',
].join('\n');

const scratch = t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-composition-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
};

const fixture = (t, overrides = {}) => {
  const root = scratch(t);
  const upstreamFile = path.join(root, 'upstream.cordis.patch.yml');
  fs.writeFileSync(upstreamFile, upstream);
  const contract = {
    schemaVersion: 1,
    file: compositionFile,
    versions: {
      '9.9.9': {
        harnessPin: { ref: 'dsh-v9.9.9', sha: 'a'.repeat(40) },
        upstream: { file: upstreamFile, sha256: hash(upstream) },
        desktop: {
          reference: 'fixture desktop 0123abc',
          rows: ['mcp-servers-file'],
          configs: ['preparedSessionCacheSize: 1'],
          renamedRows: { alpha: 'native-alpha' },
        },
        sha256: hash(stagedBody),
        alternates: [],
      },
    },
  };
  const version = contract.versions['9.9.9'];
  Object.assign(version, overrides.version ?? {});
  return { root, contract, version };
};

const stage = (t, body, { version = '9.9.9', pin = null } = {}) => {
  const runtime = scratch(t);
  const harness = path.join(runtime, 'vendor', 'deepseek-harness');
  const composition = path.join(harness, ...path.posix.dirname(compositionFile).split('/'));
  fs.mkdirSync(composition, { recursive: true });
  fs.writeFileSync(path.join(harness, 'package.json'), JSON.stringify({ name: '@deepseek-ai/dsh-root', version }));
  fs.writeFileSync(path.join(composition, path.basename(compositionFile)), body);
  if (pin) fs.writeFileSync(path.join(runtime, 'vendor', 'harness-upstream.json'), JSON.stringify(pin));
  return runtime;
};

const rejectsWith = async (promise, text) => {
  await assert.rejects(promise, error => {
    assert.ok(error.message.includes(text), `expected the failure to name ${JSON.stringify(text)}, got:\n${error.message}`);
    return true;
  });
};

test('verify accepts a stage whose composition matches the contract', async t => {
  const { contract } = fixture(t);
  const result = await verifyComposition(stage(t, stagedBody), { contract });
  assert.equal(result.version, '9.9.9');
  assert.equal(result.sha256, hash(stagedBody));
  assert.equal(result.alternate, false);
});

test('verify accepts a recorded alternate and reports why it is one', async t => {
  const alternate = stagedBody.replace('        preparedSessionCacheSize: 1\n', '');
  const { contract } = fixture(t, {
    version: { alternates: [{ sha256: hash(alternate), why: 'source predates the desktop cache bound' }] },
  });
  const result = await verifyComposition(stage(t, alternate), { contract });
  assert.equal(result.alternate, true);
  assert.equal(result.expected, 'source predates the desktop cache bound');
});

test('verify names a desktop row the stage is missing', async t => {
  const { contract } = fixture(t);
  const row = "    - id: mcp-servers-file\n      name: '@deepseek-ai/dsh-mcp-servers-file'\n";
  const withoutRow = stagedBody.split(row).join('');
  await rejectsWith(verifyComposition(stage(t, withoutRow), { contract }), 'missing rows the desktop composes: mcp-servers-file');
});

test('verify names a row left under its upstream id', async t => {
  const { contract } = fixture(t);
  const unrenamed = stagedBody.replace('- id: native-alpha', '- id: alpha');
  await rejectsWith(verifyComposition(stage(t, unrenamed), { contract }), 'rows still under their upstream id: native-alpha');
});

test('verify names a row the contract cannot account for', async t => {
  const { contract } = fixture(t);
  const row = "    - id: mcp-servers-file\n      name: '@deepseek-ai/dsh-mcp-servers-file'\n";
  const stray = stagedBody.split(row).join(`${row}\n    - id: stray-tool\n      name: '@deepseek-ai/dsh-stray'\n`);
  await rejectsWith(verifyComposition(stage(t, stray), { contract }), 'rows this contract does not account for: stray-tool');
});

test('verify names a missing desktop config value', async t => {
  const { contract } = fixture(t);
  const withoutConfig = stagedBody.replace('        preparedSessionCacheSize: 1\n', '');
  await rejectsWith(verifyComposition(stage(t, withoutConfig), { contract }), 'missing desktop config: preparedSessionCacheSize: 1');
});

test('verify names CRLF line endings as the difference', async t => {
  const { contract } = fixture(t);
  const crlf = stagedBody.split('\n').join('\r\n');
  await rejectsWith(verifyComposition(stage(t, crlf), { contract }), 'uses CRLF line endings');
});

test('verify refuses a harness version with no recorded contract', async t => {
  const { contract } = fixture(t);
  await rejectsWith(verifyComposition(stage(t, stagedBody, { version: '8.8.8' }), { contract }), 'has no composition contract');
});

test('verify refuses a stage whose pin record names another harness pin', async t => {
  const { contract } = fixture(t);
  const runtime = stage(t, stagedBody, { pin: { ref: 'dsh-v9.9.9', sha: 'b'.repeat(40) } });
  await rejectsWith(verifyComposition(runtime, { contract }), 'mixes a harness version with another pin');
});

test('verify refuses when the tracked upstream copy is not the recorded pin', async t => {
  const { contract, version } = fixture(t);
  version.upstream.sha256 = 'c'.repeat(64);
  await rejectsWith(verifyComposition(stage(t, stagedBody), { contract }), 'is not the recorded pin');
});

test('diff reports rows added on one side and configs that differ', async t => {
  const { contract } = fixture(t);
  const left = stage(t, stagedBody);
  const right = stage(t, upstream);
  const result = await diffComposition(left, right, { contract });
  assert.deepEqual(result.rowsOnlyLeft.sort(), ['mcp-servers-file', 'native-alpha']);
  assert.deepEqual(result.rowsOnlyRight, ['alpha']);
  assert.deepEqual(result.rowsWithDifferentConfig, ['beta']);
});

test('the shipped contract pins the upstream files it carries', async () => {
  const contract = await loadContract();
  const versions = Object.entries(contract.versions);
  assert.ok(versions.length > 0);
  for (const [version, entry] of versions) {
    assert.match(entry.sha256, /^[a-f0-9]{64}$/, `${version} sha256`);
    assert.match(entry.harnessPin.sha, /^[a-f0-9]{40}$/, `${version} harness pin`);
    const filename = path.join(path.dirname(fileURLToPath(import.meta.url)), 'runtime-composition', entry.upstream.file);
    assert.equal(hash(fs.readFileSync(filename, 'utf8')), entry.upstream.sha256, `${version} upstream copy`);
    for (const alternate of entry.alternates) {
      assert.match(alternate.sha256, /^[a-f0-9]{64}$/, `${version} alternate`);
      assert.ok(alternate.why.length > 20, `${version} alternate reason`);
    }
  }
});
