// The harness fork as a replayable git history.
//
// Upstream tags become one commit each, 本体's vendored tree becomes one commit,
// and every bump is a merge whose human resolution is a patch. Replaying this is
// what makes a harness bump one merge: the trees are checked by hash, so a
// resolution that no longer applies stops before a build starts instead of
// leaving a silently different source tree behind.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {importArchiveTree} from './archive.mjs';

export const directory = path.dirname(fileURLToPath(import.meta.url));
export const projectRoot = path.resolve(directory, '..', '..');
export const defaultWork = path.join(projectRoot, '.harness-fork');
export const pinFile = path.join(directory, 'pin.json');

/** Paths whose conflicts a bump resolves by keeping the fork's side. */
export const unownedPatterns = [
  /^docs\//, /^website\//, /^examples\//, /^benchmarks\//, /^snapshots\//, /^python\//,
  /^\.agents\//, /^\.github\//,
  /(^|\/)tests\//, /(^|\/)__tests__\//, /\.spec\.[cm]?[jt]sx?$/, /\.snap$/, /(^|\/)README(\.[a-z-]+)?\.(md|yaml)$/,
];

/** Conflicts a bump always starts from the upstream side: the lockfile is regenerated, never merged. */
const regenerated = /^pnpm-lock\.yaml$/;

export const isUnowned = file => unownedPatterns.some(pattern => pattern.test(file));

export function loadPin(file = pinFile) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

export function writePin(pin, file = pinFile) {
  fs.writeFileSync(file, `${JSON.stringify(pin, null, 2)}\n`);
}

export function tagEntry(pin, ref) {
  const entry = pin.upstream.tags.find(tag => tag.ref === ref);
  if (!entry) throw new Error(`${ref} is not in the fork pin. Add it with: node scripts/harness-fork/fork.mjs bump ${ref}`);
  return entry;
}

function run(command, args, options = {}) {
  return spawnSync(command, args, {
    cwd: options.cwd,
    env: {...process.env, ...options.env},
    encoding: 'utf8',
    windowsHide: true,
    maxBuffer: 64 * 1024 * 1024,
  });
}

export function git(args, options = {}) {
  const result = run('git', args, options);
  if (result.error) throw result.error;
  if (result.status !== 0 && !options.allow) {
    throw new Error(`git ${args.join(' ')} failed in ${options.cwd ?? process.cwd()}: ${(result.stderr || result.stdout).trim()}`);
  }
  return result;
}

const stdout = result => result.stdout.trim();
const revParse = (repo, rev) => stdout(git(['rev-parse', '--verify', '--quiet', rev], {cwd: repo, allow: true}));
const treeOf = (repo, rev) => stdout(git(['rev-parse', `${rev}^{tree}`], {cwd: repo}));

let indexCounter = 0;

/** A scratch index over objects the record repo already holds; no worktree is consulted. */
function scratchIndex(repo) {
  const index = path.join(os.tmpdir(), `harness-fork-${process.pid}-${indexCounter++}.index`);
  fs.rmSync(index, {force: true});
  return {index, env: {GIT_DIR: path.join(repo, '.git'), GIT_INDEX_FILE: index}};
}

/** Commit a tree the replay produced. Reading no worktree is what keeps hosts out of the identity. */
function commitReplay({repo, tree, ref, parents, message, expectTree, label}) {
  if (expectTree && tree !== expectTree) {
    throw new Error(`${label} content is not the recorded tree: the replay built ${tree}, the pin records ${expectTree}. The download is not the source this fork was recorded against; do not record over it.`);
  }
  const commit = stdout(git(['commit-tree', tree, '--no-gpg-sign', ...parents.flatMap(parent => ['-p', parent]), '-m', message], {cwd: repo}));
  git(['update-ref', ref, commit], {cwd: repo});
  return {tree, commit};
}

/** Every link the head carries, so a worktree that stores one differently still compares clean. */
function linkEntries(repo, rev) {
  const links = new Map();
  for (const line of git(['-c', 'core.quotePath=false', 'ls-tree', '-r', rev], {cwd: repo}).stdout.split('\n').filter(Boolean)) {
    const match = /^120000 blob ([0-9a-f]{40})\t(.*)$/.exec(line);
    if (match) links.set(match[2], git(['cat-file', 'blob', match[1]], {cwd: repo}).stdout);
  }
  return links;
}

/** Whether this host can hold a symbolic link is a privilege, not part of the record. */
function symlinkSupport(repo) {
  const probe = path.join(repo, `.harness-fork-link-probe-${process.pid}`);
  try {
    fs.rmSync(probe, {force: true});
    fs.symlinkSync('no-such-target', probe);
    fs.unlinkSync(probe);
    return true;
  } catch {
    fs.rmSync(probe, {force: true});
    return false;
  }
}

/**
 * Put the tag's links on disk where the host cannot make symbolic ones. A junction
 * (directories) and a hard link (files) both keep the path resolving to the node
 * the tag points at; a plain file holding the target text is a different checkout,
 * so an unrepresentable link stops the materialize instead.
 */
function materializeLinks({repo, links, symlinks, log}) {
  for (const [name, target] of links) {
    const at = path.join(repo, ...name.split('/'));
    const resolved = path.resolve(path.dirname(at), target);
    if (!fs.existsSync(resolved)) {
      throw new Error(`Cannot materialize the link ${name} -> ${target}: its target is not in the tree, so the recorded tree is not a usable checkout`);
    }
    const directory = fs.statSync(resolved).isDirectory();
    fs.rmSync(at, {recursive: true, force: true});
    if (symlinks) {
      fs.symlinkSync(target, at, directory ? 'dir' : 'file');
      continue;
    }
    if (directory) fs.symlinkSync(resolved, at, 'junction');
    else fs.linkSync(resolved, at);
    log(`link ${name} -> ${target} materialized as ${directory ? 'junction' : 'hard link'}`);
  }
}

/**
 * A scratch repo must not inherit the machine's line endings, symlink settings or
 * identity: the fork is compared by tree hash, and a checkout that rewrites LF to
 * CRLF is a different tree.
 */
export function configureScratchRepo(repo) {
  for (const [key, value] of [
    ['core.autocrlf', 'false'], ['core.symlinks', 'true'], ['commit.gpgsign', 'false'],
    ['gc.auto', '0'], ['fetch.recurseSubmodules', 'false'],
  ]) git(['config', key, value], {cwd: repo});
  for (const [key, value] of [['user.name', 'harness-fork'], ['user.email', 'harness-fork@invalid']]) {
    if (!stdout(git(['config', '--get', key], {cwd: repo, allow: true}))) git(['config', key, value], {cwd: repo});
  }
}

export function ensureWork(work = defaultWork) {
  const repo = path.join(work, 'tree');
  fs.mkdirSync(work, {recursive: true});
  if (!fs.existsSync(path.join(repo, '.git'))) {
    fs.mkdirSync(repo, {recursive: true});
    git(['init', '-q', '-b', 'mygo'], {cwd: repo});
    configureScratchRepo(repo);
  }
  return {work, repo};
}

/** The proxy curl would use on its own, so a failed download can name it instead of the record. */
function environmentProxy(environment) {
  for (const name of ['HTTPS_PROXY', 'https_proxy', 'HTTP_PROXY', 'http_proxy', 'ALL_PROXY', 'all_proxy']) {
    if (environment[name]) return {name, value: environment[name]};
  }
  return null;
}

export function download(url, destination, options = {}) {
  const runner = options.runner ?? run;
  const environment = options.environment ?? process.env;
  fs.mkdirSync(path.dirname(destination), {recursive: true});
  const args = ['-fsSL', '--retry', '3', '--retry-delay', '2'];
  if (options.noProxy) args.push('--noproxy', '*');
  else if (options.proxy) args.push('--proxy', options.proxy);
  const result = runner('curl', [...args, '-o', destination, url]);
  if (result.error) throw new Error(`curl is required to fetch ${url}: ${result.error.message}`);
  if (result.status === 0) return destination;
  const detail = (result.stderr || `curl exited ${result.status}`).trim();
  if (options.noProxy) throw new Error(`Fetching ${url} directly failed: ${detail}. This network may need --proxy <url>.`);
  const used = options.proxy ? {name: '--proxy', value: options.proxy} : environmentProxy(environment);
  if (used) throw new Error(`Fetching ${url} failed: ${detail}. That request went through the proxy at ${used.value} (${used.name}); if it cannot reach this host, retry with --no-proxy to download directly.`);
  throw new Error(`Fetching ${url} failed: ${detail}. Pass --proxy <url> when this network needs one.`);
}

/** Resolve a tag to its commit through the public API, so no clone of the upstream repository is needed. */
export function remoteTagSha(repoUrl, ref, options = {}) {
  const api = `${repoUrl.replace(/\.git$/, '').replace('https://github.com/', 'https://api.github.com/repos/')}/git/ref/tags/${ref}`;
  const file = path.join(os.tmpdir(), `harness-fork-tag-${process.pid}-${indexCounter++}.json`);
  try {
    const value = JSON.parse(fs.readFileSync(download(api, file, options), 'utf8')).object;
    if (value.type !== 'tag') return value.sha;
    const tagApi = `${repoUrl.replace(/\.git$/, '').replace('https://github.com/', 'https://api.github.com/repos/')}/git/tags/${value.sha}`;
    return JSON.parse(fs.readFileSync(download(tagApi, file, options), 'utf8')).object.sha;
  } finally {
    fs.rmSync(file, {force: true});
  }
}

async function importUpstreamTag({work, repo, pin, entry, proxy, noProxy, fetch, log}) {
  const ref = `refs/harness/upstream/${entry.ref}`;
  const existing = revParse(repo, ref);
  if (existing && (!entry.tree || treeOf(repo, existing) === entry.tree)) return existing;
  const sha = entry.sha || remoteTagSha(pin.upstream.repo, entry.ref, {proxy});
  const archive = fetch(`https://codeload.github.com/deepseek-ai/deepseek-harness/tar.gz/${sha}`, path.join(work, 'downloads', `${entry.ref}.tar.gz`), {proxy, noProxy});
  const {tree} = await importArchiveTree({archive, repo, strip: 1, expectTree: entry.tree, label: entry.ref, work, log});
  const previous = pin.upstream.tags.slice(0, pin.upstream.tags.indexOf(entry)).pop();
  const parent = previous ? revParse(repo, `refs/harness/upstream/${previous.ref}`) : null;
  const {commit} = commitReplay({
    repo, tree, ref,
    parents: parent ? [parent] : [],
    message: `upstream: ${entry.ref} @ ${sha.slice(0, 10)}`,
  });
  return commit;
}

async function importCarry({work, repo, pin, proxy, noProxy, fetch, log}) {
  const existing = revParse(repo, 'refs/harness/carry');
  if (existing && treeOf(repo, existing) === pin.carry.tree) return existing;
  const base = revParse(repo, `refs/harness/upstream/${pin.carry.base}`);
  if (!base) throw new Error(`The carry's base tag ${pin.carry.base} has not been imported; pin.upstream.tags must list it before the carry.`);
  const archive = fetch(`https://codeload.github.com/${pin.carry.repo.replace(/^https:\/\/github\.com\//, '')}/tar.gz/${pin.carry.commit}`, path.join(work, 'downloads', `carry-${pin.carry.commit.slice(0, 10)}.tar.gz`), {proxy, noProxy});
  const {tree} = await importArchiveTree({archive, repo, strip: 1, prefix: `${pin.carry.path}/`, expectTree: pin.carry.tree, label: 'carry', work, log});
  const {commit} = commitReplay({
    repo, tree, ref: 'refs/harness/carry', parents: [base],
    message: `carry: 本体 ${pin.carry.commit.slice(0, 10)} ${pin.carry.path}`,
  });
  return commit;
}

/** A merge with the fork's side winning every conflict: the baseline a resolution patch is a delta from. */
export function baselineMerge({repo, ours, theirs}) {
  try {
    return stdout(git(['merge-tree', '--write-tree', '-X', 'ours', ours, theirs], {cwd: repo})).split('\n')[0].trim();
  } catch (error) {
    if (!/unknown switch `X'/.test(error.message)) throw error;
    throw new Error(`This git's merge-tree does not take -X, so there is no "our side wins" baseline to record a resolution against; git 2.56 has it and 2.39 does not. Bump on a newer git rather than resolving against a different baseline.`);
  }
}

function applyBaseline({repo, tree, patch}) {
  const {index, env} = scratchIndex(repo);
  try {
    git(['read-tree', tree], {cwd: repo, env});
    git(['apply', '--cached', '--binary', '--whitespace=nowarn', patch], {cwd: repo, env});
    return stdout(git(['write-tree'], {cwd: repo, env}));
  } finally {
    fs.rmSync(index, {force: true});
  }
}

function replayBump({repo, pin, bump, head}) {
  const tagCommit = revParse(repo, `refs/harness/upstream/${bump.ref}`);
  if (!tagCommit) throw new Error(`Bump ${bump.ref} has no imported tag; pin.upstream.tags must list it.`);
  const ref = `refs/harness/bump/${bump.ref}`;
  const existing = revParse(repo, ref);
  if (existing && treeOf(repo, existing) === bump.tree) return existing;
  const baseline = baselineMerge({repo, ours: head, theirs: tagCommit});
  const tree = applyBaseline({repo, tree: baseline, patch: path.resolve(directory, bump.resolution)});
  if (bump.tree && tree !== bump.tree) {
    throw new Error(`Applying ${bump.resolution} to the ${bump.ref} merge produced ${tree}, the pin records ${bump.tree}. The recorded resolution no longer applies cleanly; re-record it with: node scripts/harness-fork/fork.mjs record ${bump.ref}`);
  }
  const commit = stdout(git(['commit-tree', tree, '--no-gpg-sign', '-p', head, '-p', tagCommit, '-m', `merge upstream ${bump.ref}`], {cwd: repo}));
  git(['update-ref', ref, commit], {cwd: repo});
  return commit;
}

/**
 * Replay the whole pin into a git worktree. Idempotent: commits that already
 * exist with the recorded tree are reused, and nothing is downloaded twice.
 * @returns {{work: string, repo: string, head: string, tree: string}}
 */
export async function materialize({work = defaultWork, proxy, noProxy = false, pin: pinPath = pinFile, fetch = download, log = () => {}} = {}) {
  const pin = loadPin(pinPath);
  const repo = ensureWork(work).repo;
  let upstreamTip = null;
  for (const entry of pin.upstream.tags) {
    upstreamTip = await importUpstreamTag({work, repo, pin, entry, proxy, noProxy, fetch, log});
    log(`upstream ${entry.ref} -> ${upstreamTip.slice(0, 10)}`);
  }
  git(['update-ref', 'refs/heads/upstream', upstreamTip], {cwd: repo});
  let head = await importCarry({work, repo, pin, proxy, noProxy, fetch, log});
  log(`carry -> ${head.slice(0, 10)}`);
  for (const bump of pin.bumps) {
    head = replayBump({repo, pin, bump, head});
    log(`bump ${bump.ref} -> ${head.slice(0, 10)}`);
  }
  const links = linkEntries(repo, head);
  const dirty = git(['-c', 'core.quotePath=false', 'status', '--porcelain', '--untracked-files=no'], {cwd: repo, allow: true})
    .stdout.split('\n').filter(Boolean).filter(line => !links.has(line.slice(3)));
  if (dirty.length) throw new Error(`The fork worktree has tracked changes before checkout:\n${dirty.join('\n')}\nResolve or discard them first; materialize must reproduce the recorded tree exactly.`);
  const symlinks = symlinkSupport(repo);
  git(['config', 'core.symlinks', String(symlinks)], {cwd: repo});
  git(['checkout', '-f', '-B', 'mygo', head], {cwd: repo});
  git(['reset', '-q', '--hard', head], {cwd: repo});
  materializeLinks({repo, links, symlinks, log});
  return {work, repo, head, tree: treeOf(repo, head)};
}

/** Import one new tag and merge it, leaving conflicts for the operator to resolve. */
export async function bump(ref, {work = defaultWork, proxy, noProxy = false, log = () => {}} = {}) {
  const pin = loadPin();
  if (!pin.upstream.tags.some(tag => tag.ref === ref)) {
    const sha = remoteTagSha(pin.upstream.repo, ref, {proxy});
    const archive = download(`https://codeload.github.com/deepseek-ai/deepseek-harness/tar.gz/${sha}`, path.join(work, 'downloads', `${ref}.tar.gz`), {proxy, noProxy});
    const repo = ensureWork(work).repo;
    const {tree: measured} = await importArchiveTree({archive, repo, strip: 1, label: ref, work, log});
    pin.upstream.tags.push({ref, sha, tree: measured});
    writePin(pin);
    log(`pinned upstream ${ref} @ ${sha.slice(0, 10)} (${measured.slice(0, 10)})`);
  }
  const {repo} = ensureWork(work);
  const current = revParse(repo, 'refs/heads/mygo');
  if (!current) await materialize({work, proxy, noProxy, log});
  const tagCommit = await importUpstreamTag({work, repo, pin: loadPin(), entry: tagEntry(loadPin(), ref), proxy, noProxy});
  git(['checkout', '-f', 'mygo'], {cwd: repo});
  const merge = git(['merge', '--no-ff', '--no-commit', tagCommit], {cwd: repo, allow: true});
  const conflicted = stdout(git(['diff', '--name-only', '--diff-filter=U'], {cwd: repo}))
    .split('\n').filter(Boolean).sort();
  const auto = [];
  for (const file of conflicted) {
    if (!regenerated.test(file) && !isUnowned(file)) continue;
    git(['checkout', '--ours', '--', file], {cwd: repo});
    git(['add', '--', file], {cwd: repo});
    auto.push(file);
  }
  if (regenerated.test(conflicted.join('\n')) || auto.some(file => regenerated.test(file))) log('pnpm-lock.yaml takes the upstream side; run pnpm install in the fork before building so it resolves 本体\'s packages again.');
  const remaining = stdout(git(['diff', '--name-only', '--diff-filter=U'], {cwd: repo})).split('\n').filter(Boolean);
  log(merge.status === 0 ? `${ref} merged with no conflicts` : `${ref} merged; ${remaining.length} conflict(s) need a decision, ${auto.length} resolved by policy`);
  if (remaining.length) log(`resolve these in ${repo}, then run: node scripts/harness-fork/fork.mjs record ${ref}\n${remaining.map(file => `  ${file}`).join('\n')}`);
  return {ref, repo, conflicted, auto, remaining};
}

/** Record the resolution of the merge in the worktree as this bump's patch. */
export function record(ref, {why, work = defaultWork, log = () => {}} = {}) {
  const {repo} = ensureWork(work);
  const pin = loadPin();
  if (!why) throw new Error('record needs the decision it encodes: --why "<what you kept and what upstream lost>"');
  const pending = stdout(git(['diff', '--name-only', '--diff-filter=U'], {cwd: repo}));
  if (pending) throw new Error(`Unresolved conflicts remain:\n${pending}`);
  const merging = fs.existsSync(path.join(repo, '.git', 'MERGE_HEAD'));
  if (merging) git(['commit', '--no-edit', '--no-gpg-sign'], {cwd: repo});
  const head = revParse(repo, 'HEAD');
  const parents = stdout(git(['rev-list', '--parents', '-n', '1', head], {cwd: repo})).split(' ').slice(1);
  if (parents.length !== 2) throw new Error('record expects the merge commit of one bump; HEAD is not a merge.');
  const [ours, theirs] = parents;
  const baseline = baselineMerge({repo, ours, theirs});
  const patchFile = path.join(directory, 'resolution', `${ref}.patch`);
  fs.mkdirSync(path.dirname(patchFile), {recursive: true});
  const diff = git(['diff', '--binary', '--full-index', baseline, head], {cwd: repo});
  fs.writeFileSync(patchFile, diff.stdout);
  const tree = treeOf(repo, head);
  const replayed = applyBaseline({repo, tree: baseline, patch: patchFile});
  if (replayed !== tree) throw new Error(`The recorded patch does not reproduce ${head.slice(0, 10)}: replay gave ${replayed}, the merge is ${tree}. Nothing was pinned.`);
  const entry = {ref, resolution: path.relative(directory, patchFile).split(path.sep).join('/'), tree, why};
  pin.bumps = [...pin.bumps.filter(bump => bump.ref !== ref), entry];
  writePin(pin);
  log(`recorded ${entry.resolution}: ${diff.stdout.split('\n').filter(line => line.startsWith('diff --git')).length} file(s), tree ${tree.slice(0, 10)}`);
  return entry;
}

export function auditCarry({work = defaultWork} = {}) {
  const pin = loadPin();
  const {repo} = ensureWork(work);
  const base = revParse(repo, `refs/harness/upstream/${pin.carry.base}`);
  const carry = revParse(repo, 'refs/harness/carry');
  if (!base || !carry) throw new Error('Materialize the fork before auditing the carry.');
  const packages = root => {
    const listed = git(['ls-tree', '-r', '--name-only', root, 'packages/'], {cwd: repo}).stdout.split('\n').filter(Boolean);
    const found = new Set();
    for (const file of listed) {
      const parts = file.split('/');
      if (parts.length > 3) found.add(`${parts[1]}/${parts[2]}`);
    }
    return found;
  };
  const upstream = packages(base);
  const carried = packages(carry);
  return {
    forkOnly: [...carried].filter(name => !upstream.has(name)).sort(),
    upstreamOnly: [...upstream].filter(name => !carried.has(name)).sort(),
    recorded: pin.carry.packages,
  };
}

async function main(argv) {
  const [command, ...rest] = argv;
  const options = {log: message => console.log(message)};
  const flags = new Map();
  const positional = [];
  for (let index = 0; index < rest.length; index += 1) {
    if (!rest[index].startsWith('--')) { positional.push(rest[index]); continue; }
    const key = rest[index].slice(2);
    const value = rest[index + 1] && !rest[index + 1].startsWith('--') ? rest[++index] : true;
    flags.set(key, value);
  }
  if (flags.has('work')) options.work = path.resolve(flags.get('work'));
  if (flags.has('proxy')) options.proxy = flags.get('proxy');
  if (flags.has('no-proxy')) options.noProxy = true;
  if (command === 'materialize') {
    const result = await materialize(options);
    console.log(JSON.stringify({repo: result.repo, head: result.head, tree: result.tree}, null, 2));
    return;
  }
  if (command === 'bump' && positional.length === 1) {
    console.log(JSON.stringify(await bump(positional[0], options), null, 2));
    return;
  }
  if (command === 'record' && positional.length === 1) {
    console.log(JSON.stringify(record(positional[0], {...options, why: flags.get('why')}), null, 2));
    return;
  }
  if (command === 'audit') {
    const report = auditCarry(options);
    const missing = report.recorded.filter(name => !report.forkOnly.includes(name));
    const unrecorded = report.forkOnly.filter(name => !report.recorded.includes(name));
    console.log(JSON.stringify({...report, missing, unrecorded}, null, 2));
    if (missing.length || unrecorded.length) process.exitCode = 1;
    return;
  }
  throw new Error('Usage: fork.mjs materialize [--work <dir>] [--proxy <url> | --no-proxy] | bump <tag> [--proxy <url> | --no-proxy] | record <tag> --why <decision> | audit');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch(error => {
    console.error(`harness fork: ${error.message}`);
    process.exitCode = 1;
  });
}
