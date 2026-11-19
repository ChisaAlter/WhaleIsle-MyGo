import test from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { auditFork } from './runtime-fork-audit.mjs';

const cli = fileURLToPath(new URL('./runtime-fork-audit.mjs', import.meta.url));

const tree = (t, packages) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'whale-fork-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.mkdirSync(path.join(root, 'packages'), { recursive: true });
  for (const name of packages) fs.mkdirSync(path.join(root, 'packages', ...name.split('/')), { recursive: true });
  return root;
};

const run = (...args) => spawnSync(process.execPath, [cli, ...args], { encoding: 'utf8' });

test('the audit names packages each side carries alone', t => {
  const upstream = tree(t, ['core/session', 'mcp/mcp-client']);
  const desktop = tree(t, ['core/session', 'mcp/mcp-client', 'mcp/mcp-servers-file', 'client/ui-visual-replies']);
  const report = auditFork(upstream, desktop);
  assert.deepEqual(report.forkOnly, ['client/ui-visual-replies', 'mcp/mcp-servers-file']);
  assert.deepEqual(report.upstreamOnly, []);
  assert.equal(report.upstreamPackages, 2);
  assert.equal(report.desktopPackages, 4);
});

test('a vendor carrying extras is a non-zero result that names them', t => {
  const upstream = tree(t, ['core/session']);
  const desktop = tree(t, ['core/session', 'mcp/mcp-servers-file', 'llm/llm-vision-fallback', 'client/ui-visual-replies']);
  const result = run(upstream, desktop);
  assert.equal(result.status, 1);
  for (const name of ['client/ui-visual-replies', 'llm/llm-vision-fallback', 'mcp/mcp-servers-file']) {
    assert.ok(result.stderr.includes(name), `expected the audit to name ${name}, got:\n${result.stderr}`);
  }
  assert.match(result.stderr, /must copy that vendor/);
});

test('identical trees pass', t => {
  const upstream = tree(t, ['core/session', 'mcp/mcp-client']);
  const desktop = tree(t, ['core/session', 'mcp/mcp-client']);
  const result = run(upstream, desktop);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /adds no package over the upstream tree/);
});

test('missing arguments are a usage error, not a pass', t => {
  const result = run(tree(t, ['core/session']));
  assert.equal(result.status, 2);
  assert.match(result.stderr, /Usage: runtime-fork-audit\.mjs/);
});
