// Bring the desktop-owned plugins from an explicit clean revision into an
// isolated installed Host stage. Preserve the prior packages for local rollback.
import fs from 'node:fs';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {verifyStagedSessionRuntime} from './session-runtime.mjs';

const packages = ['dsh-whale', 'dsh-im', 'dsh-task-control', 'dsh-platform-session'];
const inside = (root, file) => { const relative = path.relative(root, file); return relative === '' || (!path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep)); };
function copyOwned(source, target, root) {
  const stat = fs.lstatSync(source);
  if (stat.isSymbolicLink()) {
    const real = fs.realpathSync(source);
    if (!inside(root, real)) throw Error(`Plugin dependency escapes package: ${source}`);
    copyOwned(real, target, root);
  } else if (stat.isDirectory()) {
    fs.mkdirSync(target, {recursive: true});
    for (const name of fs.readdirSync(source)) copyOwned(path.join(source, name), path.join(target, name), root);
  } else if (stat.isFile()) {
    fs.mkdirSync(path.dirname(target), {recursive: true});
    fs.copyFileSync(source, target, fs.constants.COPYFILE_EXCL);
  } else throw Error(`Unsupported plugin entry: ${source}`);
}

export function copyDesktopPackage(source, destination) {
  const manifest = JSON.parse(fs.readFileSync(path.join(source, 'package.json'), 'utf8'));
  // A package without an npm files allowlist ships its files, including the
  // implementation behind main. Treating an omitted list as [] drops lib/.
  const declared = manifest.files ?? fs.readdirSync(source).filter(name => name !== '.git');
  const includes = new Set(['package.json', ...declared, 'node_modules']);
  for (const optional of ['LICENSE', 'NOTICE', 'THIRD_PARTY_NOTICES.md']) if (fs.existsSync(path.join(source, optional))) includes.add(optional);
  for (const relative of includes) {
    const input = path.resolve(source, relative), output = path.resolve(destination, relative);
    if (!inside(source, input) || !inside(destination, output)) throw Error('Unsafe package file entry');
    if (!fs.existsSync(input)) { if (relative === 'node_modules') continue; throw Error(`Missing plugin package file: ${relative}`); }
    copyOwned(input, output, source);
  }
  for (const entry of [manifest.main, manifest.dsh?.bundle?.patch].filter(Boolean)) {
    if (!fs.statSync(path.join(destination, entry)).isFile()) throw Error(`Missing staged plugin entry: ${entry}`);
  }
  return manifest;
}

export async function stageDesktopPlugins(sourceRoot, revision, stageRoot) {
  sourceRoot = fs.realpathSync(sourceRoot);
  stageRoot = fs.realpathSync(stageRoot);
  if (inside(sourceRoot, stageRoot) || inside(stageRoot, sourceRoot)) throw Error('Source and runtime stage must be disjoint');
  const vendor = path.join(stageRoot, 'vendor');
  if (fs.existsSync(path.join(vendor, 'runtime-graph.json'))) throw Error('Cannot mutate a closed release; create a new editable stage');
  const host = await verifyStagedSessionRuntime(vendor);
  if (host.cliVersion !== '0.2.1-rc.4') throw Error('Desktop task plugins require the selected rc.4 stage');
  const git = args => execFileSync('git', ['-C', sourceRoot, ...args], {encoding: 'utf8', windowsHide: true});
  const commit = git(['rev-parse', '--verify', revision + '^{commit}']).trim();
  const paths = packages.map(name => 'vendor/' + name);
  if (git(['diff', '--name-only', commit, '--', ...paths]).trim()) throw Error('Selected desktop plugin sources differ from the requested revision');
  const backup = path.join(stageRoot, 'desktop-plugins-before-' + commit.slice(0, 12));
  const prepared = path.join(stageRoot, 'desktop-plugins-prepared-' + commit.slice(0, 12));
  if (fs.existsSync(backup) || fs.existsSync(prepared)) throw Error('Prior plugin preparation retained; use a fresh runtime stage');
  fs.mkdirSync(prepared);
  for (const name of packages) {
    const source = path.join(sourceRoot, 'vendor', name), destination = path.join(prepared, name);
    const manifest = copyDesktopPackage(source, destination);
    fs.writeFileSync(path.join(destination, 'mygo-desktop-source.json'), JSON.stringify({schemaVersion: 1, sourceRoot, commit, package: name, version: manifest.version, host: host.cliVersion}, null, 2) + '\n', {flag: 'wx'});
  }
  // All source files/dependencies have been prepared before swapping any package.
  fs.mkdirSync(backup);
  for (const name of packages) {
    const current = path.resolve(vendor, name), saved = path.resolve(backup, name), next = path.resolve(prepared, name);
    if (!inside(stageRoot, current) || !inside(stageRoot, saved) || !inside(stageRoot, next)) throw Error('Plugin move escapes owned runtime stage');
    fs.renameSync(current, saved);
    try { fs.renameSync(next, current); } catch (error) { fs.renameSync(saved, current); throw error; }
  }
  fs.rmdirSync(prepared);
  return {commit, packages, backup};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [source, revision, stage, extra] = process.argv.slice(2);
  if (!source || !revision || !stage || extra) throw Error('Usage: session-runtime-desktop-plugins.mjs <desktop source> <revision> <staged runtime>');
  console.log(JSON.stringify(await stageDesktopPlugins(source, revision, stage), null, 2));
}
