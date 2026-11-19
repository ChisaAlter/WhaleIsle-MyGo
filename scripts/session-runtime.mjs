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
  applySessionRuntimeReadiness, readinessProfile,
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
const supportedHarnessPins = Object.freeze([{
  repo: 'https://github.com/deepseek-ai/deepseek-harness.git',
  ref: 'dsh-v0.2.1-rc.2', sha: '687f282e8625c6b8d11ff58211a90524be2e4317', npm: '0.2.1-rc.2',
}, {
  repo: 'https://github.com/deepseek-ai/deepseek-harness.git',
  ref: 'dsh-v0.2.1-rc.4', sha: '504a01a9578abc2c6f1fb896b1dc322c9fe24f66', npm: '0.2.1-rc.4',
}, {
  repo: 'https://github.com/deepseek-ai/deepseek-harness.git',
  ref: 'dsh-v0.2.2-rc.2', sha: 'ddac96760b6edf8fa0fe17cbdcf0b512f4c12b2c', npm: '0.2.2-rc.2',
}]);
const sharedDesktopAdmissionTypes = ['team/mode', 'subagent/catalog', 'assistant/attempt'];
function admissionTypesFor(version) {
  return version === '0.2.2-rc.2' ? [...sessionAdmissionTypes, ...sharedDesktopAdmissionTypes] : sessionAdmissionTypes;
}
function supportedPin(version, pin) {
  return supportedHarnessPins.find(candidate => candidate.npm === version &&
    Object.entries(candidate).every(([key, value]) => pin?.[key] === value));
}
const requiredArtifacts = [...Object.values(artifactPaths).map(value => 'deepseek-harness/' + value), readinessBundlePath, 'harness-upstream.json'];
const fixtureHeader = () => ({version: 4, id: 'runtime-compat-fixture', createdAt: 0, isSeeded: false});
function fixtureEvent(type) {
  const registered = {jobId: 'fixture-job', owner: 'runtime-compat-fixture', kind: 'fixture', label: 'fixture', startedAt: 0};
  const data = type === 'job/registered' ? registered
    : type === 'job/settled' ? {...registered, finishedAt: 1, status: 'completed', cause: 'producer', producerReported: true}
    : type === 'working-directory/change' ? {cwd: 'C:/runtime-compat-fixture'}
    : type === 'team/mode' ? {mode: 'active'}
    : type === 'subagent/catalog' ? {version: 3, childId: 'fixture-child', childCreatedAt: 1, mode: 'continuable', label: 'Fixture child', name: 'fixture-child'}
    : type === 'assistant/attempt' ? {turn: 1, step: 1, stream: []}
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
  const requiredEventTypes = admissionTypesFor(cliPackage.version);
  for (const type of requiredEventTypes) {
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
    rootVersion: rootPackage.version, requiredEventTypes: [...requiredEventTypes],
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
  if (!supportedPin(metadata.cliVersion, metadata.harnessPin)) throw new Error('Unsupported source Session runtime profile or Harness pin');
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
  const safetyPatch = await applySessionRuntimeReadiness(targetVendor, graph.root, metadata.cliVersion);
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
    if (relative === 'host-session-runtime.json' || relative === 'runtime-graph.json') continue;
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
  // A release's closure includes its Host manifest. Relocation rewrites that
  // manifest, so publish the new copy's closure after those changes complete.
  if (graph.files.includes('runtime-graph.json')) {
    const copied = inventoryGraph(vendor, ['runtime-graph.json', 'dshd-remote/debug.log']);
    await fs.writeFile(path.join(vendor, 'runtime-graph.json'), JSON.stringify({schemaVersion: 1, files: copied.files, links: copied.links}, null, 2) + '\n', {flag: 'wx'});
  }
  return verifyStagedSessionRuntime(vendor);
}

// A capability ships a loader package plus the engine that loader runs. Slimming
// either half leaves a plugin the Host mounts and cannot run: office-to-pdf keeps
// a 127.8 MB WASM engine whose 300 KB loader was stripped, and Sherpa keeps 17 JS
// files with no native addon to load. Both halves are required; only the optional
// per-platform native office engine may go, because the kit falls back to WASM
// when its native package is absent.
function platformPackageTag() {
  const family = process.platform === 'win32' ? 'win' : process.platform === 'darwin' ? 'darwin' : 'linux';
  return `${family}-${process.arch}`;
}

export const capabilityPairs = Object.freeze([Object.freeze({
  capability: 'office-to-pdf',
  loader: '@deepseek-ai/libreoffice-kit',
  engines: Object.freeze([
    Object.freeze({name: '@deepseek-ai/libreoffice-kit-wasm', asset: 'assets/dsoffice.wasm', optionalNative: false}),
    Object.freeze({name: `@deepseek-ai/libreoffice-kit-${platformPackageTag()}`, asset: null, optionalNative: true}),
  ]),
}), Object.freeze({
  capability: 'speech-to-text-sensevoice',
  loader: 'sherpa-onnx-node',
  engines: Object.freeze([Object.freeze({name: `sherpa-onnx-${platformPackageTag()}`, asset: 'addon', optionalNative: false})]),
})]);

export const optionalNativeEnginesToSlim = Object.freeze(capabilityPairs
  .flatMap(pair => pair.engines.filter(engine => engine.optionalNative).map(engine => engine.name)));

export const criticalPackagesToPreserve = Object.freeze([
  '@img/sharp-win32-x64',
  ...capabilityPairs.map(pair => pair.loader),
  ...capabilityPairs.flatMap(pair => pair.engines.filter(engine => !engine.optionalNative).map(engine => engine.name)),
]);

export async function applyReleaseSlim(vendorDir) {
  const root = await fs.realpath(vendorDir);
  const harnessRoot = path.join(root, 'deepseek-harness');
  const dshInstancesDir = path.join(harnessRoot, 'node_modules/.dsh-instances');
  const dshLinksFile = path.join(harnessRoot, '.dsh-runtime-links.json');

  let instanceEntries = [];
  try {
    instanceEntries = await fs.readdir(dshInstancesDir, { withFileTypes: true });
  } catch (err) {
    if (err.code === 'ENOENT') return { removedPackages: [], removedInstances: [], removedLinks: 0 };
    throw err;
  }

  const packageToInstances = new Map();
  const instancePackages = new Map();
  for (const entry of instanceEntries) {
    if (!entry.isDirectory()) continue;
    const instanceId = entry.name;
    const instancePath = path.join(dshInstancesDir, instanceId);

    async function collectPackageNames(dir, depth = 0) {
      if (depth > 3) return [];
      const found = [];
      try {
        const sub = await fs.readdir(dir, { withFileTypes: true });
        for (const s of sub) {
          const full = path.join(dir, s.name);
          if (s.isFile() && s.name === 'package.json') {
            try {
              const parsed = JSON.parse(await fs.readFile(full, 'utf8'));
              if (parsed.name) found.push(parsed.name);
            } catch {}
          } else if (s.isDirectory() && s.name !== 'node_modules') {
            found.push(...await collectPackageNames(full, depth + 1));
          }
        }
      } catch {}
      return found;
    }

    const pkgNames = await collectPackageNames(instancePath);
    instancePackages.set(instanceId, new Set(pkgNames));
    for (const name of pkgNames) {
      if (!packageToInstances.has(name)) packageToInstances.set(name, new Set());
      packageToInstances.get(name).add(instanceId);
    }
  }

  const instancesToRemove = new Set();
  const identifiedPackages = [];
  for (const pkgName of optionalNativeEnginesToSlim) {
    const ids = packageToInstances.get(pkgName);
    if (ids) {
      for (const id of ids) instancesToRemove.add(id);
      identifiedPackages.push(pkgName);
    }
  }

  for (const critPkg of criticalPackagesToPreserve) {
    const ids = packageToInstances.get(critPkg);
    if (ids) {
      for (const id of ids) {
        if (instancesToRemove.has(id)) {
          throw new Error(`Safety violation: critical package ${critPkg} marked for removal`);
        }
      }
    }
  }

  let linkManifest;
  try {
    linkManifest = JSON.parse(await fs.readFile(dshLinksFile, 'utf8'));
  } catch (err) {
    if (err.code !== 'ENOENT') throw err;
  }

  if (linkManifest && Array.isArray(linkManifest.links)) {
    const isInstanceRemoved = targetOrPath => {
      for (const id of instancesToRemove) {
        const prefix = `node_modules/.dsh-instances/${id}`;
        if (targetOrPath === prefix || targetOrPath.startsWith(prefix + '/')) return true;
      }
      return false;
    };

    // Verify no retained instance depends on removed instances. A dependency a
    // retained package declares optional is the one exception: the LibreOffice kit
    // documents that it selects WASM when its native engine package is absent, so
    // pruning that link is the supported configuration rather than a broken graph.
    const declaredOptional = new Map();
    const optionalDependenciesOf = async ownerId => {
      if (!declaredOptional.has(ownerId)) {
        let names = new Set();
        try {
          const parsed = JSON.parse(await fs.readFile(path.join(dshInstancesDir, ownerId, 'package.json'), 'utf8'));
          names = new Set(Object.keys(parsed.optionalDependencies ?? {}));
        } catch {}
        declaredOptional.set(ownerId, names);
      }
      return declaredOptional.get(ownerId);
    };

    for (const link of linkManifest.links) {
      const match = link.path.match(/^node_modules\/\.dsh-instances\/([^/]+)/);
      const ownerId = match ? match[1] : null;
      if (ownerId && !instancesToRemove.has(ownerId) && isInstanceRemoved(link.target)) {
        const optional = await optionalDependenciesOf(ownerId);
        const targetNames = [...(instancePackages.get(link.target.match(/^node_modules\/\.dsh-instances\/([^/]+)/)?.[1]) ?? [])];
        if (!(targetNames.length > 0 && targetNames.every(name => optional.has(name)))) {
          throw new Error(`Safety violation: retained instance ${ownerId} depends on removed target: ${link.target}`);
        }
      }
    }

    const linksToRemove = [];
    const remainingLinks = [];
    for (const link of linkManifest.links) {
      if (isInstanceRemoved(link.path) || isInstanceRemoved(link.target)) {
        linksToRemove.push(link);
      } else {
        remainingLinks.push(link);
      }
    }

    for (const link of linksToRemove) {
      const linkFsPath = path.join(harnessRoot, link.path);
      try {
        const stat = await fs.lstat(linkFsPath);
        if (stat.isSymbolicLink()) {
          await fs.unlink(linkFsPath);
        } else if (stat.isDirectory()) {
          await fs.rm(linkFsPath, { recursive: true, force: true });
        }
      } catch (err) {
        if (err.code !== 'ENOENT') throw err;
      }
    }

    for (const id of instancesToRemove) {
      const instDir = path.join(dshInstancesDir, id);
      await fs.rm(instDir, { recursive: true, force: true });
    }

    linkManifest.links = remainingLinks;
    await fs.writeFile(dshLinksFile, JSON.stringify(linkManifest, null, 2) + '\n');

    const hostManifestPath = path.join(root, 'host-session-runtime.json');
    try {
      const hostManifest = JSON.parse(await fs.readFile(hostManifestPath, 'utf8'));
      hostManifest.relocatedLinks = remainingLinks.length;
      await fs.writeFile(hostManifestPath, JSON.stringify(hostManifest, null, 2) + '\n');
    } catch (err) {
      if (err.code !== 'ENOENT') throw err;
    }

    return {
      removedPackages: identifiedPackages,
      removedInstances: [...instancesToRemove],
      removedLinks: linksToRemove.length,
      remainingLinks: remainingLinks.length,
    };
  }

  return { removedPackages: identifiedPackages, removedInstances: [...instancesToRemove], removedLinks: 0 };
}

async function packageNameAt(dir) {
  try {
    const parsed = JSON.parse(await fs.readFile(path.join(dir, 'package.json'), 'utf8'));
    return typeof parsed.name === 'string' ? parsed.name : null;
  } catch { return null; }
}

async function indexNodeModules(installed, nodeModules) {
  let entries;
  try { entries = await fs.readdir(nodeModules, { withFileTypes: true }); }
  catch (error) { if (error.code !== 'ENOENT') throw error; return; }
  const dirs = [];
  for (const entry of entries) {
    if (!(entry.isDirectory() || entry.isSymbolicLink()) || entry.name === '.dsh-instances') continue;
    if (entry.name.startsWith('@')) {
      try {
        for (const inner of await fs.readdir(path.join(nodeModules, entry.name), { withFileTypes: true })) {
          if (inner.isDirectory() || inner.isSymbolicLink()) dirs.push(path.join(nodeModules, entry.name, inner.name));
        }
      } catch {}
    } else dirs.push(path.join(nodeModules, entry.name));
  }
  for (const dir of dirs) {
    const name = await packageNameAt(dir);
    if (name) installed.set(name, [...(installed.get(name) ?? []), dir]);
  }
}

// Every package the Host can require from either its top-level tree or one of the
// instance stores the link manifest points at. Resolution order is the Host's, so
// presence here is what the plugin sees at run time.
export async function indexHarnessPackages(harnessRoot) {
  const installed = new Map();
  for (const nodeModules of [path.join(harnessRoot, 'node_modules'), path.join(harnessRoot, 'apps/cli/node_modules')]) {
    await indexNodeModules(installed, nodeModules);
    let instances;
    try { instances = await fs.readdir(path.join(nodeModules, '.dsh-instances'), { withFileTypes: true }); }
    catch (error) { if (error.code !== 'ENOENT') throw error; continue; }
    for (const entry of instances) {
      if (!entry.isDirectory()) continue;
      const dir = path.join(nodeModules, '.dsh-instances', entry.name);
      const name = await packageNameAt(dir);
      if (name) installed.set(name, [...(installed.get(name) ?? []), dir]);
      await indexNodeModules(installed, path.join(dir, 'node_modules'));
    }
  }
  return installed;
}

async function engineAssetsPresent(dir, asset) {
  if (asset === null) return true;
  if (asset === 'addon') {
    try { return (await fs.readdir(dir)).some(name => name.endsWith('.node')); } catch { return false; }
  }
  try { await fs.stat(path.join(dir, asset)); return true; } catch { return false; }
}

export async function inspectCapabilities(vendorDir) {
  const root = await fs.realpath(path.resolve(vendorDir));
  const harnessRoot = path.join(root, 'deepseek-harness');
  const installed = await indexHarnessPackages(harnessRoot);
  const capabilities = [];
  for (const pair of capabilityPairs) {
    const engines = [];
    for (const engine of pair.engines) {
      const usable = [];
      for (const dir of installed.get(engine.name) ?? []) {
        if (await engineAssetsPresent(dir, engine.asset)) usable.push(dir);
      }
      engines.push({ name: engine.name, optionalNative: engine.optionalNative, dirs: usable });
    }
    const loaderDirs = installed.get(pair.loader) ?? [];
    capabilities.push({
      capability: pair.capability, loader: pair.loader, loaderDirs, engines,
      complete: loaderDirs.length > 0 && engines.some(engine => engine.dirs.length > 0),
    });
  }
  return { harnessRoot, capabilities };
}

// Refuse a stage whose plugin/engine pair is incomplete. A plugin that needs a
// loader and cannot find one must never reach an installer: the Host mounts it,
// the user enables it, and the capability fails at the moment it is used.
export async function assertCapabilities(vendorDir) {
  const report = await inspectCapabilities(vendorDir);
  const broken = report.capabilities.filter(capability => !capability.complete);
  if (broken.length === 0) return report;
  const detail = broken.map(capability => [
    `${capability.capability}: loader ${capability.loader} ${capability.loaderDirs.length ? 'present' : 'missing'}`,
    `engines ${capability.engines.map(engine => `${engine.name} ${engine.dirs.length ? 'present' : 'missing'}`).join(', ')}`,
  ].join(', ')).join(' | ');
  throw new Error(`Packaged runtime cannot run a capability it ships (${detail})`);
}

// Package from a verified stage into a fresh directory. Existing source bytes
// and selection remain untouched; only this independent copy receives records.
export async function stageReleaseRuntime(stageRoot, freshDest, {slim = false} = {}) {
  const source = await fs.realpath(path.resolve(stageRoot));
  const target = path.resolve(freshDest);
  if (within(source, target) || within(target, source)) throw Error('Release runtime destination overlaps source');
  try { await fs.lstat(target); throw Error('Release runtime destination must be fresh'); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
  const metadata = await verifyStagedSessionRuntime(path.join(source, 'vendor'));
  await fs.mkdir(target);
  const vendor = path.join(target, 'vendor');
  cloneGraph(path.join(source, 'vendor'), vendor, {exclude: ['runtime-graph.json', 'dshd-remote/debug.log']});
  await fs.mkdir(path.join(target, 'node'));
  await fs.copyFile(await fileInside(source, 'node/node.exe'), path.join(target, 'node/node.exe'), constants.COPYFILE_EXCL);
  const mobile = path.join(vendor, 'dshd-remote');
  if (!(await fs.stat(path.join(mobile, 'staging-manifest.json')).catch(error => { if (error.code !== 'ENOENT') throw error; }))) {
    // The original manager's plain-Node fallback expects a nested Harness tree.
    // The released complete runtime instead owns one exact sibling Host graph.
    const manager = path.join(mobile, 'src/main/dshd-remote.js');
    const original = await fs.readFile(manager, 'utf8');
    const before = "return path.join(__dirname, '..', '..', 'vendor', 'deepseek-harness');";
    const after = "return path.join(__dirname, '..', '..', '..', 'deepseek-harness');";
    if (original.split(before).length !== 2) throw Error('Installed mobile manager has no unique supported Harness fallback');
    const patched = original.replace(before, after);
    await fs.writeFile(manager, patched);
    await fs.writeFile(path.join(mobile, 'source-provenance.json'), JSON.stringify({schemaVersion: 1, profile: metadata.cliVersion === '0.2.1-rc.4' ? 'installed-rc4' : 'installed-rc2', sourceRoot: path.join(source, 'vendor/dshd-remote'), managerRelocation: {path: 'src/main/dshd-remote.js', baseSHA256: crypto.createHash('sha256').update(original).digest('hex'), outputSHA256: crypto.createHash('sha256').update(patched).digest('hex'), reason: 'Use the same verified Host graph as the desktop rather than an absent nested source-build copy.'}}, null, 2) + '\n', {flag: 'wx'});
    recordInstalled(mobile);
  }
  verifyMobile(mobile);
  if (slim) {
    await applyReleaseSlim(vendor);
  }
  await assertCapabilities(vendor);
  const graph = inventoryGraph(vendor, ['runtime-graph.json', 'dshd-remote/debug.log']);
  const closure = {schemaVersion: 1, files: graph.files, links: graph.links};
  await fs.writeFile(path.join(vendor, 'runtime-graph.json'), JSON.stringify(closure, null, 2) + '\n', {flag: 'wx'});
  await verifyStagedSessionRuntime(vendor);
  return {vendor, files: Object.keys(closure.files).length, links: Object.keys(closure.links).length, slim};
}

/** Verify the MyGo-specific staged profile before packaging or selecting it. */
export async function verifyStagedSessionRuntime(vendorDir) {
  const root = await fs.realpath(path.resolve(vendorDir));
  const manifestPath = await fileInside(root, 'host-session-runtime.json');
  if ((await fs.lstat(manifestPath)).isSymbolicLink()) throw new Error('Session runtime manifest must be a regular staged file');
  const manifest = await readJSON(manifestPath);
  if (manifest.schemaVersion !== 1 || !supportedPin(manifest.cliVersion, manifest.harnessPin)) {
    throw new Error('Unsupported staged Session runtime profile or Harness pin');
  }
  const requiredTypes = admissionTypesFor(manifest.cliVersion);
  if (!Array.isArray(manifest.requiredEventTypes) || manifest.requiredEventTypes.length !== requiredTypes.length ||
      requiredTypes.some(type => !manifest.requiredEventTypes.includes(type))) {
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
  const expectedPatch = readinessProfile(manifest.cliVersion);
  const recordPath = await fileInside(root, readinessRecordFile);
  if ((await fs.lstat(recordPath)).isSymbolicLink()) throw new Error('Session readiness record must be a regular staged file');
  const record = await readJSON(recordPath);
  for (const [key, expected] of Object.entries(expectedPatch)) {
    if (manifest.safetyPatch?.[key] !== expected || record[key] !== expected) throw new Error('Staged Session runtime readiness record mismatch');
  }
  if (manifest.artifacts[readinessBundlePath] !== expectedPatch.outputSHA256) throw new Error('Staged Session runtime readiness output mismatch');
  const inspected = await inspectSessionRuntime(path.join(root, 'deepseek-harness'));
  if (inspected.cliVersion !== manifest.cliVersion || !supportedPin(manifest.cliVersion, inspected.harnessPin)) {
    throw new Error('Staged Session runtime Harness pin changed');
  }
  if (manifest.node?.path !== 'node/node.exe' || !/^[a-f0-9]{64}$/.test(manifest.node?.sha256 ?? '') ||
      await sha(await fileInside(path.dirname(root), 'node/node.exe')) !== manifest.node.sha256) {
    throw new Error('Staged Session runtime Node artifact changed');
  }
  return manifest;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const args = process.argv.slice(2);
  const slim = args.includes('--slim');
  const [command, source, destination, ...extra] = args.filter(a => a !== '--slim');
  const usage = 'Usage: session-runtime.mjs inspect <harnessRoot> | verify <vendorDir> | capabilities <vendorDir> | stage <resources> <fresh runtimeDir> | clone <stageRoot> <fresh runtimeDir> | release <stageRoot> <fresh runtimeDir> [--slim]';
  const takesDirectory = command === 'inspect' || command === 'verify' || command === 'capabilities';
  try {
    if (!source || extra.length || (takesDirectory ? destination !== undefined : !['stage', 'clone', 'release'].includes(command) || !destination)) {
      throw new Error(usage);
    }
    if (command === 'inspect') {
      console.log(JSON.stringify(await inspectSessionRuntime(source), null, 2));
    } else if (command === 'capabilities') {
      const report = await assertCapabilities(source);
      console.log(JSON.stringify(report, null, 2));
    } else {
      console.log(JSON.stringify(command === 'verify' ? await verifyStagedSessionRuntime(source)
        : command === 'clone' ? await cloneStagedSessionRuntime(source, destination)
        : command === 'release' ? await stageReleaseRuntime(source, destination, {slim})
        : await stageSessionRuntime(source, destination), null, 2));
    }
  } catch (error) {
    console.error(`Session runtime: ${error.message}`);
    process.exitCode = 1;
  }
}
