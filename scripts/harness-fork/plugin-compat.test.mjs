import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {CARRIED_COMPATIBILITY, carryPluginCompatibility, widenPluginCompatibility} from './plugin-compat.mjs';

const plugin = (name, peers, extra = {}) => ({name, version: '0.1.0', peerDependencies: peers, ...extra});

const scratch = t => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'plugin-compat-'));
  t.after(() => fs.rmSync(root, {recursive: true, force: true}));
  return root;
};

test('a recorded plugin is widened to the shipped version, and only on its dsh requirements', () => {
  const manifest = plugin('dsh-whale', {
    '@deepseek-ai/cordis': '^4.0.1',
    '@deepseek-ai/dsh-agent': '^0.1.5-rc.2 || 0.2.1-rc.2 || 0.2.2-rc.2',
    '@deepseek-ai/dsh-session': '^0.1.5-rc.2 || 0.2.2-rc.1',
  });
  const changes = widenPluginCompatibility(manifest, '0.2.2-rc.5');
  assert.deepEqual(changes, [
    '@deepseek-ai/dsh-agent: ^0.1.5-rc.2 || 0.2.1-rc.2 || 0.2.2-rc.2 -> ^0.1.5-rc.2 || 0.2.1-rc.2 || 0.2.2-rc.2 || 0.2.2-rc.5',
    '@deepseek-ai/dsh-session: ^0.1.5-rc.2 || 0.2.2-rc.1 -> ^0.1.5-rc.2 || 0.2.2-rc.1 || 0.2.2-rc.5',
  ]);
  assert.equal(manifest.peerDependencies['@deepseek-ai/cordis'], '^4.0.1', 'a non-dsh peer is not touched');
});

test('widening the same manifest twice changes nothing the second time', () => {
  const manifest = plugin('dsh-remote', {'@deepseek-ai/dsh-tools': '^0.1.0-rc.6 || 0.2.2-rc.2'});
  widenPluginCompatibility(manifest, '0.2.2-rc.5');
  const written = JSON.stringify(manifest);
  assert.deepEqual(widenPluginCompatibility(manifest, '0.2.2-rc.5'), []);
  assert.equal(JSON.stringify(manifest), written);
});

test('a plugin whose ranges already admit the version is left alone', () => {
  const manifest = plugin('dshbot', {
    '@deepseek-ai/dsh-agent': '*',
    '@deepseek-ai/dsh-tools': '^0.1.5-rc.2 || ^0.2.2-rc.5',
  });
  assert.deepEqual(widenPluginCompatibility(manifest, '0.2.2-rc.5'), []);
});

test('a requirement that needs widening without a record stops the stage', () => {
  const manifest = plugin('dsh-unrecorded', {'@deepseek-ai/dsh-agent': '0.2.2-rc.2'});
  assert.throws(() => widenPluginCompatibility(manifest, '0.2.2-rc.5'), /add it to CARRIED_COMPATIBILITY/);
  const untabled = plugin('dsh-whale', {'@deepseek-ai/dsh-unlisted': '0.2.2-rc.2'});
  assert.throws(() => widenPluginCompatibility(untabled, '0.2.2-rc.5'), /dsh-unlisted/);
});

test('engines.dsh decides compatibility alone and is refused rather than widened', () => {
  const manifest = plugin('dsh-whale', {}, {engines: {dsh: '0.2.2-rc.2'}});
  assert.throws(() => widenPluginCompatibility(manifest, '0.2.2-rc.5'), /engines\.dsh/);
});

test('the carry widens carried manifests, skips the rest of the vendor directory, and reports each change', t => {
  const vendor = scratch(t);
  fs.mkdirSync(path.join(vendor, 'dsh-whale'));
  fs.mkdirSync(path.join(vendor, 'dshd-remote'));
  fs.writeFileSync(path.join(vendor, 'dsh-whale', 'package.json'), `${JSON.stringify(plugin('dsh-whale', {'@deepseek-ai/dsh-agent': '0.2.2-rc.2'}), null, 2)}\n`);
  fs.writeFileSync(path.join(vendor, 'dshd-remote', 'other.txt'), 'no manifest here\n');

  const widened = carryPluginCompatibility(vendor, ['dsh-whale', 'dshd-remote', 'missing'], '0.2.2-rc.5');
  assert.deepEqual(widened, ['dsh-whale@0.1.0: @deepseek-ai/dsh-agent: 0.2.2-rc.2 -> 0.2.2-rc.2 || 0.2.2-rc.5']);
  const written = JSON.parse(fs.readFileSync(path.join(vendor, 'dsh-whale', 'package.json'), 'utf8'));
  assert.equal(written.peerDependencies['@deepseek-ai/dsh-agent'], '0.2.2-rc.2 || 0.2.2-rc.5');
});

test('every recorded row names a real plugin and a real dsh requirement', () => {
  for (const row of CARRIED_COMPATIBILITY) {
    assert.ok(row.plugin.length > 0);
    assert.ok(row.requirements.length > 0, `${row.plugin} records no requirements`);
    for (const requirement of row.requirements) assert.match(requirement, /^@deepseek-ai\/dsh(-|$)/);
  }
});
