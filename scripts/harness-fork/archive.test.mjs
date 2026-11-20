// The record's identity is the archive: path bytes, the mode class the tar header
// states, and the content. These tests build an archive that has everything a real
// tag has — a link, an executable, a non-ASCII name, a CRLF file the repo's
// .gitattributes normalizes — and prove materialize reproduces it on the host
// running the suite, with no tar binary and no symlink privilege required.
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import zlib from 'node:zlib';
import {spawnSync} from 'node:child_process';
import {importArchiveTree} from './archive.mjs';
import {configureScratchRepo, materialize} from './fork.mjs';

const tmp = (t, prefix) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  t.after(() => fs.rmSync(root, {recursive: true, force: true}));
  return root;
};

function blocks(entry) {
  const body = entry.body ?? Buffer.alloc(0);
  const header = Buffer.alloc(512);
  header.write(entry.name, 0, 100, 'utf8');
  header.write(`${(entry.mode ?? 0o644).toString(8).padStart(7, '0')}\0`, 100, 8, 'latin1');
  header.write('0000000\0', 108, 8, 'latin1');
  header.write('0000000\0', 116, 8, 'latin1');
  header.write(`${body.length.toString(8).padStart(11, '0')}\0`, 124, 12, 'latin1');
  header.write('00000000000\0', 136, 12, 'latin1');
  header.fill(' ', 148, 156);
  header.write(entry.type ?? '0', 156, 1, 'latin1');
  header.write(entry.link ?? '', 157, 100, 'utf8');
  header.write('ustar\0', 257, 6, 'latin1');
  header.write('00', 263, 2, 'latin1');
  header.write(`${header.reduce((sum, byte) => sum + byte, 0).toString(8).padStart(6, '0')}\0 `, 148, 8, 'latin1');
  const pad = Buffer.alloc((512 - (body.length % 512)) % 512);
  return [header, body, pad];
}

/** A tarball with the shapes the record has to survive, written without a tar binary. */
function fixtureArchive(target, {root, entries}) {
  const parts = [];
  for (const entry of entries) {
    parts.push(...blocks({...entry, name: `${root}/${entry.name}`}));
  }
  parts.push(Buffer.alloc(1024));
  fs.mkdirSync(path.dirname(target), {recursive: true});
  fs.writeFileSync(target, zlib.gzipSync(Buffer.concat(parts)));
  return target;
}

function scratchRepo(t) {
  const root = tmp(t, 'harness-fork-archive-');
  const repo = path.join(root, 'repo');
  fs.mkdirSync(repo);
  const raw = (...args) => {
    const result = spawnSync('git', ['-c', 'core.quotePath=false', ...args], {cwd: repo, encoding: 'utf8', windowsHide: true});
    if (result.status !== 0) throw new Error(`git ${args.join(' ')}: ${result.stderr}`);
    return result.stdout;
  };
  const run = (...args) => raw(...args).trim();
  run('init', '-q', '-b', 'mygo');
  configureScratchRepo(repo);
  return {root, repo, run, raw};
}

const attributes = Buffer.from('* text=auto eol=lf\n*.cmd text eol=crlf\n');

function tagEntries() {
  return [
    {name: '.gitattributes', body: attributes},
    {name: 'AGENTS.md', body: Buffer.from('canonical\n')},
    {name: 'CLAUDE.md', type: '2', mode: 0o777, link: 'AGENTS.md'},
    {name: 'scripts/run.sh', mode: 0o755, body: Buffer.from('#!/bin/sh\necho hi\n')},
    {name: 'win.cmd', body: Buffer.from('@echo off\r\nrem crlf\r\n')},
    {name: 'docs/说明.md', body: Buffer.from('# 说明\n')},
    {name: 'packages/core/package.json', body: Buffer.from('{"name":"@fixture/core"}\n')},
  ];
}

test('an archive import is the identity, and no host decides it', async t => {
  const {root, repo, run, raw} = scratchRepo(t);
  const archive = fixtureArchive(path.join(root, 'tag.tar.gz'), {root: 'upstream-deadbeef', entries: tagEntries()});
  const {tree, entries} = await importArchiveTree({archive, repo, strip: 1, label: 'fixture', work: path.join(root, 'work')});
  assert.equal(entries, tagEntries().length);

  const listed = new Map();
  for (const line of run('ls-tree', '-r', tree).split('\n').filter(Boolean)) {
    const match = /^(\d+) blob ([0-9a-f]{40})\t(.*)$/.exec(line);
    assert.ok(match, `unexpected ls-tree line: ${line}`);
    listed.set(match[3], {mode: match[1], sha: match[2]});
  }
  assert.equal(listed.get('CLAUDE.md').mode, '120000', 'the link is recorded as a link, not as a file holding its target');
  assert.equal(listed.get('scripts/run.sh').mode, '100755', 'the execute bit comes from the tar header');
  assert.equal(listed.get('win.cmd').mode, '100644');
  assert.ok(listed.has('docs/说明.md'), 'the non-ASCII path survives byte for byte');
  assert.equal(raw('cat-file', 'blob', listed.get('CLAUDE.md').sha), 'AGENTS.md');
  assert.equal(raw('cat-file', 'blob', listed.get('win.cmd').sha), '@echo off\nrem crlf\n', 'the archive .gitattributes normalizes the repo form');
  assert.equal(raw('cat-file', 'blob', listed.get('docs/说明.md').sha), '# 说明\n');

  const again = await importArchiveTree({archive, repo, strip: 1, label: 'fixture-again', work: path.join(root, 'work')});
  assert.equal(again.tree, tree, 'the same archive imports to the same tree twice');
});

test('import refuses a tree that folds on a case-insensitive host', async t => {
  const {root, repo} = scratchRepo(t);
  const archive = fixtureArchive(path.join(root, 'case.tar.gz'), {
    root: 'upstream-deadbeef',
    entries: [{name: 'Readme.md', body: Buffer.from('a\n')}, {name: 'README.md', body: Buffer.from('b\n')}],
  });
  await assert.rejects(
    importArchiveTree({archive, repo, strip: 1, label: 'case', work: path.join(root, 'work')}),
    /differ only by case/,
  );
});

test('import skips an entry it does not want without losing the stream', async t => {
  const {root, repo, run} = scratchRepo(t);
  const archive = fixtureArchive(path.join(root, 'skip.tar.gz'), {
    root: 'carry-cafe',
    entries: [
      {name: 'docs/large.bin', body: Buffer.alloc(5000, 7)},
      {name: 'vendor/deepseek-harness/package.json', body: Buffer.from('{"name":"@fixture/core"}\n')},
      {name: 'vendor/deepseek-harness/packages/core/package.json', body: Buffer.from('{"name":"@fixture/core"}\n')},
      {name: 'README.md', body: Buffer.from('outside the prefix\n')},
    ],
  });
  const {tree} = await importArchiveTree({archive, repo, strip: 1, prefix: 'vendor/deepseek-harness/', label: 'carry', work: path.join(root, 'work')});
  assert.equal(run('ls-tree', '-r', '--name-only', tree).split('\n').length, 2, 'only the carried subtree is recorded');
  assert.match(run('ls-tree', '-r', '--name-only', tree), /package\.json/);
});

test('materialize replays a pin offline and reproduces the recorded trees', async t => {
  const root = tmp(t, 'harness-fork-materialize-');
  const tag = fixtureArchive(path.join(root, 'downloads', 'tag.tar.gz'), {root: 'upstream-deadbeef', entries: tagEntries()});
  const carry = fixtureArchive(path.join(root, 'downloads', 'carry.tar.gz'), {
    root: 'carry-cafe',
    entries: [
      {name: 'vendor/deepseek-harness/AGENTS.md', body: Buffer.from('canonical\n')},
      {name: 'vendor/deepseek-harness/CLAUDE.md', type: '2', mode: 0o777, link: 'AGENTS.md'},
      {name: 'vendor/deepseek-harness/packages/core/package.json', body: Buffer.from('{"name":"@fixture/core","fork":true}\n')},
    ],
  });
  const fetch = (url, destination) => {
    fs.mkdirSync(path.dirname(destination), {recursive: true});
    fs.copyFileSync(url.includes('carry') ? carry : tag, destination);
    return destination;
  };
  const pin = {
    schemaVersion: 1,
    note: 'fixture',
    upstream: {repo: 'https://github.com/fixture/upstream.git', tags: [{ref: 'dsh-v0.0.1', sha: 'a'.repeat(40)}]},
    carry: {repo: 'https://github.com/fixture/carry', commit: 'b'.repeat(40), path: 'vendor/deepseek-harness', base: 'dsh-v0.0.1', packages: ['core']},
    bumps: [],
  };
  const pinFile = path.join(root, 'pin.json');
  fs.writeFileSync(pinFile, JSON.stringify(pin, null, 2));

  const first = await materialize({work: path.join(root, 'first'), pin: pinFile, fetch, log: () => {}});
  assert.match(first.tree, /^[0-9a-f]{40}$/);

  const claude = path.join(first.repo, 'CLAUDE.md');
  const stat = fs.lstatSync(claude);
  assert.ok(stat.isSymbolicLink() || fs.statSync(claude).isFile(), 'the link resolves to the file the tag points at');
  assert.equal(fs.readFileSync(claude, 'utf8'), 'canonical\n', 'a plain file holding the target text would not be the same checkout');

  const recorded = {...pin, upstream: {repo: pin.upstream.repo, tags: [{...pin.upstream.tags[0], tree: (await importArchiveTree({archive: tag, repo: first.repo, strip: 1, label: 'tag', work: path.join(root, 'probe')})).tree}]}};
  recorded.carry = {...pin.carry, tree: (await importArchiveTree({archive: carry, repo: first.repo, strip: 1, prefix: 'vendor/deepseek-harness/', label: 'carry', work: path.join(root, 'probe')})).tree};
  const recordedFile = path.join(root, 'pin-recorded.json');
  fs.writeFileSync(recordedFile, JSON.stringify(recorded, null, 2));

  const second = await materialize({work: path.join(root, 'second'), pin: recordedFile, fetch, log: () => {}});
  assert.match(second.head, /^[0-9a-f]{40}$/, 'the replay built a commit');
  assert.equal(second.tree, first.tree, 'the same pin materializes the same tree twice');
});
