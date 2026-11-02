// Explicit offline Host runtime inspection/staging. Never starts Host or reads
// user sessions; admission probes contain only synthetic lifecycle metadata.
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import {constants} from 'node:fs';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {applyWindowsBackgroundRuntime} from './windows-background-runtime.mjs';
import {cloneGraph, inventoryGraph, verifyGraph} from './session-runtime-graph.mjs';
import {recordInstalled, verify as verifyMobile} from './mobile-runtime.mjs';
import {
  applySessionRuntimeReadiness, guardID, baseSHA256, patchSHA256, outputSHA256,
  bundlePath as readinessBundlePath, recordFile as readinessRecordFile,
} from './session-runtime-readiness.mjs';

export const sessionAdmissionTypes = Object.freeze([
  'job/registered', 'job/settled', 'conversation/next-prompt-request',
  'team/control', 'working-directory/change',
]);
const artifactPaths = Object.freeze({
  cli: 'apps/cli/lib/bin.js',
  session: 'packages/core/session/lib/index.js',
  persistence: 'packages/session/session-persistence/lib/index.js',
});
const supportedHarnessPin = Object.freeze({
  repo: 'https://github.com/deepseek-ai/deepseek-harness.git',
  ref: 'dsh-v0.2.1-rc.2', sha: '687f282e8625c6b8d11ff58211a90524be2e4317', npm: '0.2.1-rc.2',
});
const requiredArtifacts = [...Object.values(artifactPaths).map(value => 'deepseek-harness/' + value), readinessBundlePath, 'harness-upstream.json'];
const fixtureHeader = () => ({version: 4, id: 'runtime-compat-fixture', createdAt: 0, isSeeded: false});
function fixtureEvent(type) {
  const registered = {jobId: 'fixture-job', owner: 'runtime-compat-fixture', kind: 'fixture', label: 'fixture', startedAt: 0};
  const data = type === 'job/registered' ? registered
    : type === 'job/settled' ? {...registered, finishedAt: 1, status: 'completed', cause: 'producer', producerReported: true}
    : type === 'working-directory/change' ? {cwd: 'C:/runtime-compat-fixture'}
    : {};
  return {seq: 20, time: 0, type, data};
}
const readJSON = async filename => JSON.parse(await fs.readFile(filename, 'utf8'));
const sha = async filename => crypto.createHash('sha256').update(await fs.readFile(filename)).digest('hex');
function within(root, target) {
  const relative = path.relative(root, target);
  return relative === '' || (!path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep));
}
function child(root, relative) {
  if (typeof relative !== 'string' || path.isAbsolute(relative)) throw new Error(`Unsafe runtime path: ${relative}`);
  const target = path.resolve(root, relative);
  if (!within(root, target) || target === root) throw new Error(`Unsafe runtime path: ${relative}`);
  return target;
}
async function fileInside(root, relative) {
  const filename = child(root, relative);
  const real = await fs.realpath(filename);
  if (!within(root, real) || !(await fs.stat(real)).isFile()) throw new Error(`Runtime artifact is not an internal file: ${relative}`);
  return filename;
}

/** Validate the actual installed exports, not a version string or format number. */
export async function inspectSessionRuntime(harnessRoot) {
  const root = await fs.realpath(path.resolve(harnessRoot));
  const [rootPackage, cliPackage, sessionPackage, persistencePackage] = await Promise.all([
    readJSON(path.join(root, 'package.json')), readJSON(path.join(root, 'apps/cli/package.json')),
    readJSON(path.join(root, 'packages/core/session/package.json')),
    readJSON(path.join(root, 'packages/session/session-persistence/package.json')),
  ]);
  if ([rootPackage, cliPackage, sessionPackage, persistencePackage].some(pkg => typeof pkg.version !== 'string' || pkg.version === '')) {
    throw new Error('Session runtime package version metadata is missing');
  }
  const files = {};
  for (const [name, relative] of Object.entries(artifactPaths)) files[name] = await fileInside(root, relative);
  // Hashes are part of import identity as well as provenance. A repeated inspect
  // must not reuse a cached validator after its own critical bundle changed.
  const artifacts = {};
  for (const [name, relative] of Object.entries(artifactPaths)) artifacts['deepseek-harness/' + relative] = await sha(files[name]);
  const importArtifact = async name => {
    const url = pathToFileURL(files[name]);
    url.searchParams.set('runtime-sha256', artifacts['deepseek-harness/' + artifactPaths[name]]);
    return import(url.href);
  };
  const [session, persistence] = await Promise.all([importArtifact('session'), importArtifact('persistence')]);
  if (!(session.KNOWN_SESSION_EVENT_TYPES instanceof Set) || typeof persistence.validateStoredEvents !== 'function') {
    throw new Error('Runtime does not export the Session vocabulary and strict storage validator');
  }
  for (const type of sessionAdmissionTypes) {
    try {
      await persistence.validateStoredEvents(fixtureHeader(), [fixtureEvent(type)]);
    } catch (error) {
      throw new Error(`Session runtime rejects required event ${type}: ${error.message}`, {cause: error});
    }
    if (!session.KNOWN_SESSION_EVENT_TYPES.has(type)) throw new Error(`Session runtime does not know required event ${type}`);
  }
  const unknownType = 'runtime-compat-fixture/unknown-required';
  let unknownRequiredRejected = false;
  try {
    await persistence.validateStoredEvents(fixtureHeader(), [fixtureEvent(unknownType)]);
  } catch (error) {
    unknownRequiredRejected = error.name === 'SessionFormatUnsupportedError' && error.message.includes(unknownType);
  }
  if (!unknownRequiredRejected) throw new Error('Session runtime does not strictly reject unknown required events');
  if ([rootPackage.version, sessionPackage.version, persistencePackage.version].some(version => version !== cliPackage.version)) {
    throw new Error('Mixed Session runtime package versions');
  }
  let harnessPin = null;
  try {
    const filename = await fileInside(path.dirname(root), 'harness-upstream.json');
    harnessPin = await readJSON(filename);
    artifacts['harness-upstream.json'] = await sha(filename);
  }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  try {
    const filename = await fileInside(root, 'packages/client/ui-visual-replies/lib/index.js');
    artifacts['deepseek-harness/packages/client/ui-visual-replies/lib/index.js'] = await sha(filename);
  } catch (error) { if (error.code !== 'ENOENT') throw error; }
  return {
    schemaVersion: 1, harnessRoot: root, cliVersion: cliPackage.version,
    rootVersion: rootPackage.version, requiredEventTypes: [...sessionAdmissionTypes],
    unknownRequiredRejected, artifacts, harnessPin,
  };
}

// Inventory physical entries without following dependency junctions. Validate
// every link's final target before copying anything, then recreate links inside
// the new graph. This keeps source bytes independent and avoids graph expansion.
async function inventoryVendor(vendor) {
  const root = await fs.realpath(vendor);
  const directories = [], files = [], links = [];
  const pending = [root];
  while (pending.length) {
    const directory = pending.pop();
    for (const entry of await fs.readdir(directory, {withFileTypes: true})) {
      const filename = path.join(directory, entry.name);
      const relative = path.relative(root, filename);
      if (entry.isSymbolicLink()) {
        const real = await fs.realpath(filename);
        if (!within(root, real) || within(real, filename)) throw new Error(`External or ancestor runtime link: ${filename}`);
        const stat = await fs.stat(real);
        if (!stat.isDirectory() && !stat.isFile()) throw new Error(`Unsupported runtime link: ${filename}`);
        links.push({path: relative, target: path.relative(root, real), directory: stat.isDirectory()});
      } else if (entry.isDirectory()) {
        directories.push(relative);
        pending.push(filename);
      } else if (entry.isFile()) files.push(relative);
      else throw new Error(`Unsupported runtime entry: ${filename}`);
    }
  }
  // Installed Harness records its relocatable dependency graph. Validate the
  // manifest against actual link targets; never invoke the original repair tool.
  const manifestPath = path.join(root, 'deepseek-harness/.dsh-runtime-links.json');
  let manifest;
  try { manifest = await readJSON(manifestPath); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (manifest !== undefined) {
    if (manifest.version !== 1 || !Array.isArray(manifest.links)) throw new Error('Unsupported runtime link manifest');
    const actual = new Map(links.map(link => [path.normalize(link.path), link]));
    for (const link of manifest.links) {
      const origin = child(path.join(root, 'deepseek-harness'), link.path);
      if (typeof link.target !== 'string' || path.isAbsolute(link.target)) throw new Error(`Unsafe runtime manifest target: ${link.target}`);
      // Manifest targets are Harness-relative; a desktop sibling is allowed
      // only when it still belongs to the same copied vendor graph.
      const declared = child(root, path.relative(root, path.resolve(root, 'deepseek-harness', link.target)));
      const actualLink = actual.get(path.relative(root, origin));
      if (!actualLink || await fs.realpath(declared) !== path.join(root, actualLink.target)) {
        throw new Error(`Runtime link manifest disagrees with installed graph: ${link.path}`);
      }
    }
  }
  return {root, directories, files, links};
}

/** Copy an explicitly selected complete vendor graph into an absent directory. */
export async function stageSessionRuntime(resources, destination) {
  const source = await fs.realpath(path.resolve(resources));
  const destinationParent = await fs.realpath(path.dirname(path.resolve(destination)));
  const target = path.join(destinationParent, path.basename(path.resolve(destination)));
  if (within(source, target) || within(target, source)) throw new Error('Runtime destination overlaps source resources');
  try { await fs.lstat(target); throw new Error(`Runtime destination must be fresh: ${target}`); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  const vendor = path.join(source, 'vendor');
  const graph = await inventoryVendor(vendor);
  if (!within(source, graph.root)) throw new Error('Source vendor graph is outside selected resources');
  const nodeSource = await fileInside(source, 'runtime/primary-runtime/dependencies/node/bin/node.exe');
  const metadata = await inspectSessionRuntime(path.join(vendor, 'deepseek-harness'));
  if (metadata.harnessPin === null) throw new Error('Runtime source has no harness-upstream.json pin');
  const nodeHash = await sha(nodeSource);
  await fs.mkdir(target);
  const targetVendor = path.join(target, 'vendor');
  await fs.mkdir(targetVendor);
  // Create all real directories first; links are last and are never traversed.
  graph.directories.sort((a, b) => a.split(path.sep).length - b.split(path.sep).length);
  for (const relative of graph.directories) await fs.mkdir(child(targetVendor, relative));
  for (const relative of graph.files) await fs.copyFile(child(graph.root, relative), child(targetVendor, relative), constants.COPYFILE_EXCL);
  for (const link of graph.links) {
    const relocated = child(targetVendor, link.path);
    const targetPath = child(targetVendor, link.target);
    await fs.symlink(process.platform === 'win32' && link.directory ? targetPath : path.relative(path.dirname(relocated), targetPath), relocated,
      link.directory ? (process.platform === 'win32' ? 'junction' : 'dir') : 'file');
  }
  await fs.mkdir(path.join(target, 'node'));
  await fs.copyFile(nodeSource, path.join(target, 'node/node.exe'), constants.COPYFILE_EXCL);
  const safetyPatch = await applySessionRuntimeReadiness(targetVendor, graph.root);
  const staged = await inspectSessionRuntime(path.join(targetVendor, 'deepseek-harness'));
  for (const relative of [...Object.values(artifactPaths).map(value => 'deepseek-harness/' + value), 'harness-upstream.json']) {
    if (staged.artifacts[relative] !== metadata.artifacts[relative]) throw new Error(`Staged runtime artifact changed: ${relative}`);
  }
  if (await sha(path.join(target, 'node/node.exe')) !== nodeHash) throw new Error('Staged Node artifact changed');
  const result = {...staged, harnessPin: metadata.harnessPin, safetyPatch, node: {path: 'node/node.exe', sha256: nodeHash}, relocatedLinks: graph.links.length};
  await fs.writeFile(path.join(targetVendor, 'host-session-runtime.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx'});
  await applyWindowsBackgroundRuntime(targetVendor);
  return verifyStagedSessionRuntime(targetVendor);
}

/** Relocate an already guarded stage. No source repair, patching, or link following. */
export async function cloneStagedSessionRuntime(stageRoot, freshDest) {
  const source = await fs.realpath(path.resolve(stageRoot));
  const absolute = path.resolve(freshDest);
  const target = path.join(await fs.realpath(path.dirname(absolute)), path.basename(absolute));
  if (within(source, target) || within(target, source)) throw new Error('Runtime destination overlaps source stage');
  try { await fs.lstat(target); throw new Error(`Runtime destination must be fresh: ${target}`); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  const metadata = await verifyStagedSessionRuntime(path.join(source, 'vendor'));
  const graph = await inventoryVendor(path.join(source, 'vendor'));
  if (!within(source, graph.root)) throw new Error('Source vendor graph is outside selected stage');
  const nodeSource = await fileInside(source, 'node/node.exe');
  await fs.mkdir(target);
  const vendor = path.join(target, 'vendor');
  await fs.mkdir(vendor);
  graph.directories.sort((a, b) => a.split(path.sep).length - b.split(path.sep).length);
  for (const relative of graph.directories) await fs.mkdir(child(vendor, relative));
  for (const relative of graph.files) {
    if (relative === 'host-session-runtime.json') continue;
    await fs.copyFile(child(graph.root, relative), child(vendor, relative), constants.COPYFILE_EXCL);
  }
  for (const link of graph.links) {
    const relocated = child(vendor, link.path), targetPath = child(vendor, link.target);
    await fs.symlink(process.platform === 'win32' && link.directory ? targetPath : path.relative(path.dirname(relocated), targetPath), relocated,
      link.directory ? (process.platform === 'win32' ? 'junction' : 'dir') : 'file');
  }
  await fs.mkdir(path.join(target, 'node'));
  await fs.copyFile(nodeSource, path.join(target, 'node/node.exe'), constants.COPYFILE_EXCL);
  const result = {...metadata, harnessRoot: path.join(vendor, 'deepseek-harness'), relocatedLinks: graph.links.length};
  await fs.writeFile(path.join(vendor, 'host-session-runtime.json'), JSON.stringify(result, null, 2) + '\n', {flag: 'wx'});
  await applyWindowsBackgroundRuntime(vendor);
  return verifyStagedSessionRuntime(vendor);
}

// Package from a verified stage into a fresh directory. Existing source bytes
// and selection remain untouched; only this independent copy receives records.
export async function stageReleaseRuntime(stageRoot, freshDest) {
  const source = await fs.realpath(path.resolve(stageRoot));
  const target = path.resolve(freshDest);
  if (within(source, target) || within(target, source)) throw Error('Release runtime destination overlaps source');
  try { await fs.lstat(target); throw Error('Release runtime destination must be fresh'); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  await verifyStagedSessionRuntime(path.join(source, 'vendor'));
  await fs.mkdir(target);
  const vendor = path.join(target, 'vendor');
  cloneGraph(path.join(source, 'vendor'), vendor, {exclude: ['runtime-graph.json', 'dshd-remote/debug.log']});
  await fs.mkdir(path.join(target, 'node'));
  await fs.copyFile(await fileInside(source, 'node/node.exe'), path.join(target, 'node/node.exe'), constants.COPYFILE_EXCL);
  const mobile = path.join(vendor, 'dshd-remote');
  if (!(await fs.stat(path.join(mobile, 'staging-manifest.json')).catch(error => { if (error.code !== 'ENOENT') throw error; }))) {
    // The original manager's plain-Node fallback expects a nested Harness tree.
    // The released complete runtime instead owns one exact rc.2 sibling graph.
    const manager = path.join(mobile, 'src/main/dshd-remote.js');
    const original = await fs.readFile(manager, 'utf8');
    const before = "return path.join(__dirname, '..', '..', 'vendor', 'deepseek-harness');";
    const after = "return path.join(__dirname, '..', '..', '..', 'deepseek-harness');";
    if (original.split(before).length !== 2) throw Error('Installed mobile manager has no unique supported Harness fallback');
    const patched = original.replace(before, after);
    await fs.writeFile(manager, patched);
    await fs.writeFile(path.join(mobile, 'source-provenance.json'), JSON.stringify({schemaVersion: 1, profile: 'installed-rc2', sourceRoot: path.join(source, 'vendor/dshd-remote'), managerRelocation: {path: 'src/main/dshd-remote.js', baseSHA256: crypto.createHash('sha256').update(original).digest('hex'), outputSHA256: crypto.createHash('sha256').update(patched).digest('hex'), reason: 'Use the same verified rc.2 Host graph as the desktop rather than an absent nested source-build copy.'}}, null, 2) + '\n', {flag: 'wx'});
    recordInstalled(mobile);
  }
  verifyMobile(mobile);
  const graph = inventoryGraph(vendor, ['runtime-graph.json', 'dshd-remote/debug.log']);
  const closure = {schemaVersion: 1, files: graph.files, links: graph.links};
  await fs.writeFile(path.join(vendor, 'runtime-graph.json'), JSON.stringify(closure, null, 2) + '\n', {flag: 'wx'});
  await verifyStagedSessionRuntime(vendor);
  return {vendor, files: Object.keys(closure.files).length, links: Object.keys(closure.links).length};
}

/** Verify the MyGo-specific staged profile before packaging or selecting it. */
export async function verifyStagedSessionRuntime(vendorDir) {
  const root = await fs.realpath(path.resolve(vendorDir));
  const manifestPath = await fileInside(root, 'host-session-runtime.json');
  if ((await fs.lstat(manifestPath)).isSymbolicLink()) throw new Error('Session runtime manifest must be a regular staged file');
  const manifest = await readJSON(manifestPath);
  if (manifest.schemaVersion !== 1 || manifest.cliVersion !== supportedHarnessPin.npm ||
      !manifest.harnessPin || Object.entries(supportedHarnessPin).some(([key, value]) => manifest.harnessPin[key] !== value)) {
    throw new Error('Unsupported staged Session runtime profile or Harness pin');
  }
  if (!Array.isArray(manifest.requiredEventTypes) || manifest.requiredEventTypes.length !== sessionAdmissionTypes.length ||
      sessionAdmissionTypes.some(type => !manifest.requiredEventTypes.includes(type))) {
    throw new Error('Staged Session runtime admission vocabulary is incomplete');
  }
  if (!manifest.artifacts || typeof manifest.artifacts !== 'object' || Array.isArray(manifest.artifacts) ||
      requiredArtifacts.some(relative => typeof manifest.artifacts[relative] !== 'string')) {
    throw new Error('Staged Session runtime artifact hashes are incomplete');
  }
  await inventoryVendor(root);
  const closurePath = path.join(root, 'runtime-graph.json');
  let closure;
  try { closure = await readJSON(closurePath); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  if (closure !== undefined) verifyGraph(root, closure, {exclude: ['runtime-graph.json', 'dshd-remote/debug.log']});
  for (const [relative, expected] of Object.entries(manifest.artifacts)) {
    if (!/^[a-f0-9]{64}$/.test(expected) || await sha(await fileInside(root, relative)) !== expected) {
      throw new Error(`Staged Session runtime artifact hash mismatch: ${relative}`);
    }
  }
  const expectedPatch = {id: guardID, baseSHA256, patchSHA256, outputSHA256};
  const recordPath = await fileInside(root, readinessRecordFile);
  if ((await fs.lstat(recordPath)).isSymbolicLink()) throw new Error('Session readiness record must be a regular staged file');
  const record = await readJSON(recordPath);
  for (const [key, expected] of Object.entries(expectedPatch)) {
    if (manifest.safetyPatch?.[key] !== expected || record[key] !== expected) throw new Error('Staged Session runtime readiness record mismatch');
  }
  if (manifest.artifacts[readinessBundlePath] !== outputSHA256) throw new Error('Staged Session runtime readiness output mismatch');
  const inspected = await inspectSessionRuntime(path.join(root, 'deepseek-harness'));
  if (Object.entries(supportedHarnessPin).some(([key, value]) => inspected.harnessPin?.[key] !== value)) {
    throw new Error('Staged Session runtime Harness pin changed');
  }
  if (manifest.node?.path !== 'node/node.exe' || !/^[a-f0-9]{64}$/.test(manifest.node?.sha256 ?? '') ||
      await sha(await fileInside(path.dirname(root), 'node/node.exe')) !== manifest.node.sha256) {
    throw new Error('Staged Session runtime Node artifact changed');
  }
  return manifest;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [command, source, destination, ...extra] = process.argv.slice(2);
  try {
    if (!source || extra.length || (command === 'inspect' || command === 'verify' ? destination !== undefined : !['stage', 'clone', 'release'].includes(command) || !destination)) {
      throw new Error('Usage: session-runtime.mjs inspect <harnessRoot> | verify <vendorDir> | stage <resources> <fresh runtimeDir> | clone <stageRoot> <fresh runtimeDir>');
    }
    console.log(JSON.stringify(command === 'inspect' ? await inspectSessionRuntime(source)
      : command === 'verify' ? await verifyStagedSessionRuntime(source)
      : command === 'clone' ? await cloneStagedSessionRuntime(source, destination) : command === 'release' ? await stageReleaseRuntime(source, destination) : await stageSessionRuntime(source, destination), null, 2));
  } catch (error) {
    console.error(`Session runtime: ${error.message}`);
    process.exitCode = 1;
  }
}
