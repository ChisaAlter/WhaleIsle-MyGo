'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { run, runGit, describeWindowsWorktreeFailure } = require('./source/main/git-exec');
const { createProjectEnvironment } = require('./source/main/project-environment');
const { createWorkspaceAuthority, readProjectExecutionPaths } = require('./source/main/workspace-authority');
const { worktreePath } = require('./source/shared/project-worktree-layout');

test('long working directory uses actual Git capability and preserves interrupted work', { skip: process.platform !== 'win32' }, async t => {
  const parent = fs.realpathSync.native(os.tmpdir());
  const owned = fs.mkdtempSync(path.join(parent, 'whale-project-path-'));
  const previousProfile = process.env.USERPROFILE;
  process.env.USERPROFILE = owned;
  try {
    const source = path.join(owned, 'source');
    fs.mkdirSync(source);
    let home = path.join(owned, 'home');
    while (home.length < 175) home = path.join(home, 'padding');
    fs.mkdirSync(home, { recursive: true });
    const projectId = 'project-01234567-0123-0123-0123-012345678901';
    const workstreamId = 'work-01234567-0123-0123-0123-012345678901';
    const legacyTarget = path.join(home, 'project-worktrees', projectId, workstreamId);
    const target = worktreePath(home, projectId, workstreamId, 'compact-v1');
    assert.ok(legacyTarget.length >= 260, 'original ID-based layout exceeds the Git cwd limit');
    assert.ok(target.length < 260, 'physical layout leaves cwd inside the supported range');
    const env = { GIT_CONFIG_NOSYSTEM: '1', GIT_CONFIG_GLOBAL: path.join(owned, 'absent-config') };
    const git = async (...args) => {
      const result = await runGit(source, args, { env });
      assert.equal(result.code, 0, result.stderr);
      return result.stdout.trim();
    };
    await git('init', '--quiet');
    await git('config', 'core.autocrlf', 'false');
    fs.writeFileSync(path.join(source, 'result.txt'), 'base\n');
    await git('add', '--', 'result.txt');
    await git('-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgSign=false', '-c', `core.hooksPath=${owned}`, 'commit', '--quiet', '-m', 'fixture');
    const head = await git('rev-parse', 'HEAD');
    const bridge = createProjectEnvironment({ home, authority: createWorkspaceAuthority({ workspace: source }) });
    const payload = { action: 'prepare-workspace', projectId, workstreamId, workingDirectory: source, mode: 'worktree' };
    const result = await bridge.dispatch(payload);
    assert.equal(await git('rev-parse', 'HEAD'), head);
    assert.equal(fs.readFileSync(path.join(source, 'result.txt'), 'utf8'), 'base\n');
    const receiptPath = path.join(home, 'projects', projectId, 'internal', 'desktop-workspaces', `${workstreamId}.json`);
    const receipt = JSON.parse(fs.readFileSync(receiptPath, 'utf8'));
    assert.equal(receipt.canonicalPath, target);
    assert.equal(result.ok, true);
    assert.equal(receipt.state, 'ready');
    assert.equal(receipt.layout, 'compact-v1');
    assert.equal(receipt.projectId, projectId);
    assert.equal(receipt.workstreamId, workstreamId);
    assert.equal(receipt.branch, `project/${projectId}/${workstreamId}`);
    assert.equal(fs.readFileSync(path.join(target, 'result.txt'), 'utf8'), 'base\n');
    fs.writeFileSync(path.join(target, 'unique-edit.txt'), 'keep my work\n');
    assert.equal((await bridge.dispatch(payload)).receipt, receipt.receipt);
    assert.equal(fs.readFileSync(path.join(target, 'unique-edit.txt'), 'utf8'), 'keep my work\n');
    assert.equal(JSON.parse(fs.readFileSync(receiptPath, 'utf8')).receipt, receipt.receipt);

    assert.deepEqual(readProjectExecutionPaths(home), [target]);
    const bound = { projectId, workstreamId, workspace: result };
    const baseline = await bridge.dispatch({ ...bound, action: 'baseline' });
    fs.writeFileSync(path.join(target, 'result.txt'), 'delivered change\n');
    const changes = await bridge.dispatch({ ...bound, action: 'changes', base: baseline.tree, diff: true });
    assert.match(changes.diff, /delivered change/);
    assert.equal((await bridge.dispatch({ ...bound, action: 'revert', base: baseline.tree, end: changes.end })).applied, true);
    assert.equal(fs.readFileSync(path.join(target, 'result.txt'), 'utf8'), 'base\n');
    assert.equal(fs.readFileSync(path.join(target, 'unique-edit.txt'), 'utf8'), 'keep my work\n');
    const snapshot = await bridge.dispatch({ ...bound, action: 'snapshot' });
    assert.equal(snapshot.dirty, true);
    assert.ok(snapshot.files.some(row => row.path === 'unique-edit.txt'));
    await assert.rejects(bridge.dispatch({ ...bound, action: 'cleanup-workspace' }), /未提交/);
    fs.unlinkSync(path.join(target, 'unique-edit.txt'));

    // Git hooks still receive the real, supported cwd and can launch child Git.
    const hooks = path.join(owned, 'hooks');
    fs.mkdirSync(hooks);
    fs.writeFileSync(path.join(hooks, 'pre-commit'), '#!/bin/sh\ngit rev-parse --show-toplevel > hook-cwd.txt\n');
    const commit = await runGit(target, ['-c', `core.hooksPath=${hooks}`, '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgSign=false', 'commit', '--allow-empty', '--quiet', '-m', 'child Git hook']);
    assert.equal(commit.code, 0, commit.stderr);
    assert.equal(path.resolve(fs.readFileSync(path.join(target, 'hook-cwd.txt'), 'utf8').trim()), target);
    fs.unlinkSync(path.join(target, 'hook-cwd.txt'));
    const removed = await bridge.dispatch({ ...bound, action: 'cleanup-workspace' });
    assert.equal(removed.removed, true);
    assert.equal(fs.existsSync(target), false);
    assert.deepEqual(readProjectExecutionPaths(home), []);
    assert.equal(fs.readFileSync(path.join(source, 'result.txt'), 'utf8'), 'base\n');

    t.diagnostic(JSON.stringify({ legacyPathLength: legacyTarget.length, compactPathLength: target.length,
      idsAndBranchRetained: true, prepareReuseSnapshotBaselineChangesRevertCleanup: true, childGitHook: true }));

    const successful = { code: 0, stdout: 'supported', stderr: '', missing: false };
    assert.equal(describeWindowsWorktreeFailure(target, successful, true), successful, 'successful long-path operations must not be rejected by a platform rule');
  } finally {
    if (previousProfile === undefined) delete process.env.USERPROFILE;
    else process.env.USERPROFILE = previousProfile;
    const resolved = fs.realpathSync.native(owned);
    assert.equal(path.dirname(resolved), parent);
    assert.ok(path.basename(resolved).startsWith('whale-project-path-'));
    fs.rmSync(resolved, { recursive: true, force: true });
  }
});

test('legacy owned receipts retain their physical directory and dirty work', { skip: process.platform !== 'win32' }, async () => {
  const parent = fs.realpathSync.native(os.tmpdir());
  const owned = fs.mkdtempSync(path.join(parent, 'whale-project-legacy-'));
  try {
    const source = path.join(owned, 'source'), home = path.join(owned, 'home');
    fs.mkdirSync(source); fs.mkdirSync(home);
    const git = async (cwd, ...args) => {
      const result = await runGit(cwd, args);
      assert.equal(result.code, 0, result.stderr);
      return result.stdout.trim();
    };
    await git(source, 'init', '--quiet');
    await git(source, 'config', 'core.autocrlf', 'false');
    fs.writeFileSync(path.join(source, 'result.txt'), 'base\n');
    await git(source, 'add', '--', 'result.txt');
    await git(source, '-c', 'user.name=Fixture', '-c', 'user.email=fixture@example.invalid', '-c', 'commit.gpgSign=false', '-c', `core.hooksPath=${owned}`, 'commit', '--quiet', '-m', 'fixture');
    const projectId = 'legacy-project', workstreamId = 'legacy-work';
    const target = path.join(home, 'project-worktrees', projectId, workstreamId);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    await git(source, 'worktree', 'add', '--quiet', '-b', `project/${projectId}/${workstreamId}`, target);
    const bridge = createProjectEnvironment({ home, authority: createWorkspaceAuthority({ workspace: source }) });
    const repository = await bridge.repository(source);
    const record = { projectId, workstreamId, mode: 'worktree', sourceIdentity: source.toLowerCase(), canonicalPath: target,
      directoryIdentity: target.toLowerCase(), ownership: 'project', branch: `project/${projectId}/${workstreamId}`,
      baseCommit: repository.headCommit, repoIdentity: repository.repoIdentity, repositoryRoot: source,
      requestedBase: null, state: 'ready', receipt: 'retained-legacy-receipt' };
    const receiptPath = path.join(home, 'projects', projectId, 'internal', 'desktop-workspaces', workstreamId + '.json');
    fs.mkdirSync(path.dirname(receiptPath), { recursive: true });
    fs.writeFileSync(receiptPath, JSON.stringify(record));
    fs.writeFileSync(path.join(target, 'result.txt'), 'legacy dirty edit\n');
    const resumed = await bridge.dispatch({ action: 'prepare-workspace', projectId, workstreamId, workingDirectory: source, mode: 'worktree' });
    assert.equal(resumed.canonicalPath, target);
    assert.equal(resumed.receipt, record.receipt);
    assert.equal(fs.readFileSync(path.join(target, 'result.txt'), 'utf8'), 'legacy dirty edit\n');
    assert.deepEqual(readProjectExecutionPaths(home), [target]);
    for (const modified of [{ ...record, canonicalPath: source, directoryIdentity: source.toLowerCase() }, { ...record, layout: 'unknown-layout' }]) {
      fs.writeFileSync(receiptPath, JSON.stringify(modified));
      assert.deepEqual(readProjectExecutionPaths(home), [], 'changed receipts grant no arbitrary execution root');
      await assert.rejects(bridge.checkWorkspace(projectId, workstreamId), /不属于|布局/);
    }
  } finally {
    assert.equal(path.dirname(fs.realpathSync.native(owned)), parent);
    fs.rmSync(owned, { recursive: true, force: true });
  }
});

test('timed out command is settled only after its child exits', async () => {
  const result = await run(process.execPath, ['-e', 'console.log(process.pid); setInterval(() => {}, 1000)'], process.cwd(), { timeoutMs: 250 });
  assert.equal(result.timedOut, true);
  const childPid = Number(result.stdout.trim());
  assert.ok(Number.isSafeInteger(childPid) && childPid > 0, 'child started before timeout');
  assert.throws(() => process.kill(childPid, 0), { code: 'ESRCH' });
});
