import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {auditFork} from '../runtime-fork-audit.mjs';
import {assembleInstanceGraph, compositionSeeds, dependencyEntries, verifyInstanceGraph} from './graph.mjs';
import {baselineMerge, configureScratchRepo, directory as forkDirectory, isUnowned, loadPin, tagEntry} from './fork.mjs';

const tmp = (t, prefix) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  t.after(() => fs.rmSync(root, {recursive: true, force: true}));
  return root;
};

const write = (root, relative, content) => {
  const file = path.join(root, ...relative.split('/'));
  fs.mkdirSync(path.dirname(file), {recursive: true});
  fs.writeFileSync(file, content);
  return file;
};

const link = (target, at) => {
  fs.mkdirSync(path.dirname(at), {recursive: true});
  fs.symlinkSync(target, at, process.platform === 'win32' ? 'junction' : 'dir');
};

const git = (cwd, ...args) => {
  const result = spawnSync('git', ['-c', 'core.autocrlf=false', '-c', 'core.symlinks=true', ...args], {cwd, encoding: 'utf8', windowsHide: true});
  if (result.status !== 0) throw new Error(`git ${args.join(' ')}: ${result.stderr}`);
  return result.stdout.trim();
};

test('the pin lists every tag before it is used, and each bump points at a recorded patch', () => {
  const pin = loadPin();
  const refs = pin.upstream.tags.map(tag => tag.ref);
  assert.equal(new Set(refs).size, refs.length, 'a tag is listed twice');
  for (const tag of pin.upstream.tags) {
    assert.match(tag.sha, /^[a-f0-9]{40}$/, `${tag.ref} has no commit hash`);
    assert.match(tag.tree, /^[a-f0-9]{40}$/, `${tag.ref} has no tree hash`);
  }
  assert.ok(refs.includes(pin.carry.base), 'the carry base tag must be imported before the carry commit');
  assert.match(pin.carry.tree, /^[a-f0-9]{40}$/);
  assert.ok(pin.carry.packages.length > 0);
  for (const bump of pin.bumps) {
    assert.ok(refs.includes(bump.ref), `${bump.ref} is not an upstream tag`);
    assert.match(bump.tree, /^[a-f0-9]{40}$/, `${bump.ref} records no resolved tree`);
    assert.ok(bump.why && bump.why.length > 20, `${bump.ref} records no decision`);
    const patch = fs.readFileSync(path.join(forkDirectory, bump.resolution), 'utf8');
    assert.match(patch, /^diff --git /m, `${bump.resolution} is not a patch`);
  }
});

test('bump tags are ordered the way they merge', () => {
  const pin = loadPin();
  const order = pin.upstream.tags.map(tag => tag.ref);
  const bumps = pin.bumps.map(bump => order.indexOf(bump.ref));
  assert.deepEqual(bumps, [...bumps].sort((a, b) => a - b), 'a bump is recorded before the tag it merges');
  assert.ok(tagEntry(pin, pin.upstream.tags.at(-1).ref));
  assert.throws(() => tagEntry(pin, 'dsh-v9.9.9-rc.1'), /not in the fork pin/);
});

test('a recorded resolution replays to the same tree git merged', t => {
  const root = tmp(t, 'harness-fork-resolution-');
  git(root, 'init', '-q', '-b', 'main');
  configureScratchRepo(root);
  write(root, 'file.txt', 'base\n');
  write(root, 'other.txt', 'untouched\n');
  git(root, 'add', '-A');
  git(root, 'commit', '-qm', 'base');
  const base = git(root, 'rev-parse', 'HEAD');
  git(root, 'checkout', '-q', '-b', 'ours');
  write(root, 'file.txt', 'ours\n');
  git(root, 'commit', '-qam', 'ours');
  const ours = git(root, 'rev-parse', 'HEAD');
  git(root, 'checkout', '-q', '-b', 'theirs', base);
  write(root, 'file.txt', 'theirs\n');
  git(root, 'commit', '-qam', 'theirs');
  const theirs = git(root, 'rev-parse', 'HEAD');

  const baseline = baselineMerge({repo: root, ours, theirs});
  git(root, 'checkout', '-q', 'ours');
  git(root, 'read-tree', '-u', '--reset', baseline);
  write(root, 'file.txt', 'resolved\n');
  git(root, 'add', '-A');
  const resolved = git(root, 'write-tree');
  const patch = path.join(root, 'resolution.patch');
  fs.writeFileSync(patch, spawnSync('git', ['-C', root, 'diff', '--binary', baseline, resolved], {encoding: 'utf8'}).stdout);

  const replay = tmp(t, 'harness-fork-replay-');
  const environment = {GIT_DIR: path.join(root, '.git'), GIT_WORK_TREE: replay};
  const run = (...args) => spawnSync('git', args, {cwd: root, env: {...process.env, ...environment}, encoding: 'utf8'});
  assert.equal(run('read-tree', '-u', '--reset', baseline).status, 0);
  assert.equal(fs.readFileSync(path.join(replay, 'file.txt'), 'utf8'), 'ours\n', 'the baseline takes the fork side, so the patch is exactly the decision');
  assert.equal(run('-C', replay, 'apply', '--binary', patch).status, 0);
  assert.equal(fs.readFileSync(path.join(replay, 'file.txt'), 'utf8'), 'resolved\n');
});

test('conflicts the fork does not own resolve by policy, the rest wait for a decision', () => {
  for (const file of ['docs/architecture.md', 'packages/x/tests/a.spec.ts', 'packages/x/README.md', '.agents/notes/a.md', 'snapshots/web/a.expected.md']) {
    assert.ok(isUnowned(file), `${file} should resolve without a decision`);
  }
  for (const file of ['packages/client/ui-sidebar/src/client/SidebarRoot.tsx', 'packages/bundle/base/cordis.patch.yml', 'apps/cli/package.json', 'scripts/package-dependency-policy.ts']) {
    assert.ok(!isUnowned(file), `${file} carries behaviour and needs a decision`);
  }
});

/** A pnpm-shaped workspace: workspace packages linked from the root, externals under .pnpm. */
function workspace(t) {
  const root = tmp(t, 'harness-fork-workspace-');
  write(root, 'package.json', JSON.stringify({name: '@deepseek-ai/dsh-root', version: '0.2.2-rc.5'}));
  write(root, 'apps/cli/package.json', JSON.stringify({name: '@deepseek-ai/dsh-cli', version: '0.2.2-rc.5', dependencies: {'@deepseek-ai/dsh-core': 'workspace:*'}}));
  write(root, 'apps/cli/lib/bin.js', 'require("@deepseek-ai/dsh-core")\n');
  write(root, 'packages/core/package.json', JSON.stringify({name: '@deepseek-ai/dsh-core', version: '0.2.2-rc.5', dependencies: {zod: '^4.0.0'}}));
  write(root, 'packages/core/lib/index.js', 'module.exports = require("zod")\n');
  write(root, 'packages/core/src/index.ts', 'export {}\n');
  write(root, 'packages/core/tests/core.spec.ts', 'test\n');
  write(root, 'node_modules/.pnpm/zod@4.0.0/node_modules/zod/package.json', JSON.stringify({name: 'zod', version: '4.0.0', main: 'index.js'}));
  write(root, 'node_modules/.pnpm/zod@4.0.0/node_modules/zod/index.js', 'module.exports = {}\n');
  link(path.join(root, 'node_modules/.pnpm/zod@4.0.0/node_modules/zod'), path.join(root, 'node_modules/zod'));
  link(path.join(root, 'packages/core'), path.join(root, 'node_modules/@deepseek-ai/dsh-core'));
  link(path.join(root, 'apps/cli'), path.join(root, 'node_modules/@deepseek-ai/dsh-cli'));
  return root;
}

test('the instance graph copies each package once and links every edge', t => {
  const harness = workspace(t);
  const target = path.join(tmp(t, 'harness-fork-stage-'), 'deepseek-harness');
  const seeds = [path.join(harness, 'apps', 'cli')];
  const assembled = assembleInstanceGraph({harnessRoot: harness, targetRoot: target, seeds});
  assert.equal(assembled.packages, 3);
  assert.deepEqual(verifyInstanceGraph({harnessRoot: harness, targetRoot: target, seeds}).packages, 3);
  const manifest = JSON.parse(fs.readFileSync(path.join(target, '.dsh-runtime-links.json'), 'utf8'));
  assert.equal(manifest.version, 1);
  const paths = manifest.links.map(entry => entry.path);
  assert.ok(paths.includes('node_modules/@deepseek-ai/dsh-core'), 'a workspace package is reachable by name from the root');
  assert.ok(paths.includes('node_modules/zod'));
  assert.ok(paths.includes('packages/core/node_modules/zod'), 'the external package is linked from the package that depends on it');
  assert.ok(!fs.existsSync(path.join(target, 'packages/core/src')), 'source and tests are not shipped');
  assert.ok(fs.existsSync(path.join(target, 'packages/core/lib/index.js')));
  assert.equal(fs.readFileSync(path.join(target, 'packages/core/lib/index.js'), 'utf8'), 'module.exports = require("zod")\n');
});

test('a missing optional edge is skipped, a missing required edge stops the stage', t => {
  const harness = workspace(t);
  const target = path.join(tmp(t, 'harness-fork-missing-'), 'deepseek-harness');
  write(harness, 'packages/core/package.json', JSON.stringify({name: '@deepseek-ai/dsh-core', dependencies: {zod: '^4.0.0'}}));
  write(harness, 'packages/extra/package.json', JSON.stringify({name: '@deepseek-ai/dsh-extra', optionalDependencies: {absent: '^1.0.0'}}));
  link(path.join(harness, 'packages/extra'), path.join(harness, 'node_modules/@deepseek-ai/dsh-extra'));
  const seeds = [path.join(harness, 'apps', 'cli'), path.join(harness, 'packages/extra')];
  assert.equal(assembleInstanceGraph({harnessRoot: harness, targetRoot: target, seeds}).packages, 4);
  fs.rmSync(path.join(harness, 'node_modules/.pnpm'), {recursive: true, force: true});
  assert.throws(
    () => assembleInstanceGraph({harnessRoot: harness, targetRoot: target, seeds}),
    /Missing runtime dependency: @deepseek-ai\/dsh-core -> zod/,
  );
});

test('the composition file decides which packages the stage keeps', t => {
  const harness = workspace(t);
  write(harness, 'packages/bundle/base/cordis.patch.yml', [
    'plugins:',
    '  host:',
    '    - id: core',
    "      name: '@deepseek-ai/dsh-core'",
    '    - id: visual-replies',
    "      name: '@deepseek-ai/dsh-client-ui-visual-replies'",
    '',
  ].join('\n'));
  const seeds = compositionSeeds(harness, 'packages/bundle/base/cordis.patch.yml');
  assert.deepEqual(seeds.map(seed => path.basename(seed)), ['cli', 'core'], 'a row naming an unshipped desktop plugin is skipped, not invented');
  assert.throws(() => compositionSeeds(harness, 'packages/bundle/base/absent.yml'), /ENOENT/);
  assert.deepEqual([...dependencyEntries({dependencies: {a: '1'}, optionalDependencies: {b: '1', a: '1'}, peerDependencies: {a: '1', c: '1'}})], [['a', 'optional'], ['b', 'optional'], ['c', 'peer']]);
});

test('the audit fails when the carry drops a package the pin records', t => {
  const upstream = tmp(t, 'harness-fork-audit-up-');
  const fork = tmp(t, 'harness-fork-audit-fork-');
  for (const name of ['core/session', 'mcp/mcp-client']) fs.mkdirSync(path.join(upstream, 'packages', ...name.split('/')), {recursive: true});
  for (const name of ['core/session', 'mcp/mcp-client', 'mcp/mcp-servers-file']) fs.mkdirSync(path.join(fork, 'packages', ...name.split('/')), {recursive: true});
  const pin = path.join(fork, 'pin.json');
  fs.writeFileSync(pin, JSON.stringify({carry: {packages: ['mcp/mcp-servers-file']}}));
  assert.deepEqual(auditFork(upstream, fork).forkOnly, ['mcp/mcp-servers-file']);

  const cli = path.join(forkDirectory, '..', 'runtime-fork-audit.mjs');
  const ok = spawnSync(process.execPath, [cli, upstream, fork, `--expect=${pin}`], {encoding: 'utf8', windowsHide: true});
  assert.equal(ok.status, 0, ok.stderr);
  assert.match(ok.stdout, /exactly the recorded carry/);

  fs.writeFileSync(pin, JSON.stringify({carry: {packages: ['mcp/mcp-servers-file', 'client/ui-git']}}));
  const dropped = spawnSync(process.execPath, [cli, upstream, fork, `--expect=${pin}`], {encoding: 'utf8', windowsHide: true});
  assert.equal(dropped.status, 1);
  assert.match(dropped.stderr, /missing {2}packages\/client\/ui-git/);
});
