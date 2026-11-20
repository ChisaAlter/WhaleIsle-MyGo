// Turn a built pnpm workspace into the relocatable instance graph the runtime
// and the installer replay: one physical copy per package, one link per edge.
// A pnpm node_modules stores the same package under `.pnpm/<pkg>@<version>` and
// reaches it through relative links, which cannot survive the installer moving
// the tree; the shipped graph lists every link in .dsh-runtime-links.json so
// runtime-links can rebuild them at the destination.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

export const runtimeLinkManifest = '.dsh-runtime-links.json';
const workspaceRoots = new Set(['apps', 'packages', 'vendor', 'native']);
const devOnlyNames = new Set(['@deepseek-ai/dsh-client-test-runtime', '@deepseek-ai/dsh-session-snapshot']);

export function dependencyEntries(manifest) {
  const entries = new Map();
  for (const name of Object.keys(manifest.dependencies ?? {})) entries.set(name, 'required');
  for (const name of Object.keys(manifest.optionalDependencies ?? {})) entries.set(name, 'optional');
  for (const name of Object.keys(manifest.peerDependencies ?? {})) if (!entries.has(name)) entries.set(name, 'peer');
  return entries;
}

/** Node resolution with preserve-symlinks off: a linked importer resolves from its real path. */
export function resolvePackageFrom(from, name, stop) {
  const segments = String(name).split('/');
  const limit = path.resolve(stop);
  let current = fs.realpathSync(path.resolve(from));
  for (;;) {
    const candidate = path.join(current, 'node_modules', ...segments);
    if (fs.existsSync(path.join(candidate, 'package.json'))) return candidate;
    if (current === limit) return null;
    const parent = path.dirname(current);
    if (parent === current) return null;
    current = parent;
  }
}

const manifestOf = dir => JSON.parse(fs.readFileSync(path.join(dir, 'package.json'), 'utf8'));
const relativeOf = (root, dir) => path.relative(root, dir).split(path.sep).join('/');

/** Every package reachable from the seeds through runtime edges, keyed by real path. */
export function productionClosure(seeds, stop) {
  const packages = new Map();
  const visit = (dir) => {
    const real = fs.realpathSync(dir);
    if (packages.has(real)) return packages.get(real);
    const manifest = manifestOf(real);
    const node = {source: real, name: manifest.name, manifest, deps: []};
    packages.set(real, node);
    for (const [name, kind] of dependencyEntries(manifest)) {
      const dependency = resolvePackageFrom(real, name, stop);
      if (!dependency) {
        if (kind === 'required') throw new Error(`Missing runtime dependency: ${manifest.name} -> ${name}`);
        continue;
      }
      node.deps.push({name, node: visit(dependency)});
    }
    return node;
  };
  for (const seed of seeds) visit(seed);
  return packages;
}

/** Package names the composition mounts by name; a row naming a desktop plugin resolves to nothing and is skipped. */
export function compositionSeeds(harnessRoot, compositionFile) {
  const text = fs.readFileSync(path.join(harnessRoot, ...compositionFile.split('/')), 'utf8');
  const names = [...text.matchAll(/^\s*name:\s*'([^']+)'\s*$/gm)].map(match => match[1]);
  if (!names.length) throw new Error(`${compositionFile} names no plugin rows; the fork cannot tell which packages the Host mounts.`);
  const seeds = [fs.realpathSync(path.join(harnessRoot, 'apps', 'cli'))];
  for (const name of names) {
    const resolved = resolvePackageFrom(harnessRoot, name, harnessRoot);
    if (resolved) seeds.push(fs.realpathSync(resolved));
  }
  return seeds;
}

function omitRoots(relative) {
  const top = relative.split('/')[0];
  return workspaceRoots.has(top) ? new Set(['node_modules', 'src', 'tests', '__tests__']) : new Set(['node_modules']);
}

function collectFiles(source, omit) {
  const files = [];
  const walk = (relative) => {
    for (const entry of fs.readdirSync(path.join(source, relative), {withFileTypes: true})) {
      const child = relative ? `${relative}/${entry.name}` : entry.name;
      if (!relative && omit.has(entry.name)) continue;
      if (entry.name === 'node_modules') continue;
      const absolute = path.join(source, child);
      const stat = entry.isSymbolicLink() ? fs.statSync(absolute) : entry;
      if (stat.isDirectory()) { walk(child); continue; }
      if (!stat.isFile()) throw new Error(`Unsupported runtime file: ${absolute}`);
      files.push(child);
    }
  };
  walk('');
  return files;
}

function linkPath(target, link) {
  const relative = path.relative(path.dirname(link), target);
  return process.platform === 'win32' ? target : relative;
}

/** Placement is a pure function of the built tree, so verification can recompute it instead of trusting the manifest. */
export function planInstanceGraph({harnessRoot, seeds}) {
  const root = fs.realpathSync(harnessRoot);
  const packages = productionClosure(seeds, root);
  const store = 'node_modules/.dsh-instances';
  const placements = [];
  for (const [real, node] of packages) {
    if (devOnlyNames.has(node.name)) throw new Error(`A runtime dependency closure reaches the test-only package ${node.name}; the built tree is not a release tree.`);
    const relative = relativeOf(root, real);
    const parts = relative.split('/');
    const inWorkspace = workspaceRoots.has(parts[0]) && !parts.includes('node_modules');
    placements.push({
      node,
      relative,
      inWorkspace,
      placed: inWorkspace ? relative : `${store}/${crypto.createHash('sha256').update(relative).digest('hex').slice(0, 24)}`,
    });
  }
  return {root, placements};
}

/**
 * Copy each reachable package once and link every dependency edge.
 * @returns {{targetRoot: string, packages: number, files: number, links: number}}
 */
export function assembleInstanceGraph({harnessRoot, targetRoot, seeds}) {
  const {root, placements} = planInstanceGraph({harnessRoot, seeds});
  const target = path.resolve(targetRoot);
  if (target === root || target.startsWith(`${root}${path.sep}`) || root.startsWith(`${target}${path.sep}`)) {
    throw new Error('The instance graph target must be outside the built tree.');
  }
  fs.rmSync(target, {recursive: true, force: true});
  fs.mkdirSync(target, {recursive: true});
  const absolute = new Map(placements.map(placement => [placement.node.source, path.join(target, ...placement.placed.split('/'))]));
  let files = 0;
  for (const placement of placements) {
    for (const file of collectFiles(placement.node.source, omitRoots(placement.relative))) {
      const destination = path.join(absolute.get(placement.node.source), ...file.split('/'));
      fs.mkdirSync(path.dirname(destination), {recursive: true});
      fs.copyFileSync(path.join(placement.node.source, ...file.split('/')), destination);
      files += 1;
    }
  }
  const links = [];
  const link = (owner, name, placed) => {
    if (!/^(@[^/@]+\/)?[^/@]+$/.test(name)) throw new Error(`Invalid runtime package name: ${name}`);
    const location = path.join(owner, 'node_modules', ...name.split('/'));
    fs.mkdirSync(path.dirname(location), {recursive: true});
    fs.symlinkSync(linkPath(placed, location), location, process.platform === 'win32' ? 'junction' : 'dir');
    const from = relativeOf(target, location);
    const to = relativeOf(target, placed);
    if (from.startsWith('..') || to.startsWith('..')) throw new Error(`Runtime link escapes the staged tree: ${from}`);
    links.push({path: from, target: to});
  };
  for (const placement of placements) {
    for (const dep of placement.node.deps) link(absolute.get(placement.node.source), dep.name, absolute.get(dep.node.source));
  }
  const defaulted = new Set();
  for (const placement of placements) {
    const name = placement.node.manifest.name;
    if (!name || defaulted.has(name)) continue;
    defaulted.add(name);
    link(target, name, absolute.get(placement.node.source));
  }
  fs.writeFileSync(path.join(target, runtimeLinkManifest), `${JSON.stringify({version: 1, links})}\n`);
  return {targetRoot: target, packages: placements.length, files, links: links.length};
}

/** Follow every recorded edge and compare bytes against the recomputed plan, so a graph that only looks complete fails here. */
export function verifyInstanceGraph({harnessRoot, targetRoot, seeds}) {
  const {placements} = planInstanceGraph({harnessRoot, seeds});
  const target = fs.realpathSync(path.resolve(targetRoot));
  const manifest = JSON.parse(fs.readFileSync(path.join(target, runtimeLinkManifest), 'utf8'));
  if (manifest.version !== 1 || !Array.isArray(manifest.links)) throw new Error('Unsupported runtime link manifest');
  const recorded = new Map(manifest.links.map(entry => [entry.path, entry.target]));
  let files = 0;
  for (const placement of placements) {
    const placed = path.join(target, ...placement.placed.split('/'));
    if (!fs.existsSync(placed)) throw new Error(`Runtime package is missing from the staged tree: ${placement.placed}`);
    for (const file of collectFiles(placement.node.source, omitRoots(placement.relative))) {
      const copy = path.join(placed, ...file.split('/'));
      if (!fs.existsSync(copy) || !fs.readFileSync(copy).equals(fs.readFileSync(path.join(placement.node.source, ...file.split('/'))))) {
        throw new Error(`Runtime package bytes differ from the built tree: ${placement.placed}/${file}`);
      }
      files += 1;
    }
    for (const dep of placement.node.deps) {
      const location = path.join(placed, 'node_modules', ...dep.name.split('/'));
      const expected = path.join(target, ...placements.find(candidate => candidate.node.source === dep.node.source).placed.split('/'));
      if (!fs.existsSync(location) || fs.realpathSync(location) !== fs.realpathSync(expected)) {
        throw new Error(`Runtime edge does not resolve to its instance: ${dep.name} from ${placement.placed}`);
      }
      if (recorded.get(relativeOf(target, location)) !== relativeOf(target, expected)) throw new Error(`Runtime edge is not recorded in ${runtimeLinkManifest}: ${relativeOf(target, location)}`);
    }
  }
  return {packages: placements.length, files, links: manifest.links.length};
}
