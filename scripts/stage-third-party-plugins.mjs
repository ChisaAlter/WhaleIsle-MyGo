// Copy a vendored third-party desktop plugin into a runtime stage's vendor
// directory. The tracked tree under third_party/ is the only source; a stage that
// cannot read it refuses instead of producing a runtime that silently lacks the
// plugin, and a build that intentionally ships without paid application APIs says
// so on the command line rather than by omission.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';

// Each entry names the tracked source and the runtime directory it must appear
// as. A runtime keeps the upstream package name because the Host resolves the
// composition row by that name.
export const THIRD_PARTY_PLUGINS = ['dsh-deepseek-tools'];

const inside = (root, file) => {
  const relative = path.relative(root, file);
  return relative === '' || (!path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep));
};

function digest(root) {
  const hash = crypto.createHash('sha256');
  const walk = (relative) => {
    const absolute = path.join(root, relative);
    const stat = fs.lstatSync(absolute);
    if (stat.isSymbolicLink()) throw new Error(`plugin source contains a link: ${relative}`);
    if (stat.isDirectory()) {
      for (const name of fs.readdirSync(absolute).sort()) walk(path.join(relative, name));
      return;
    }
    hash.update(relative.split(path.sep).join('/'));
    hash.update('\0');
    hash.update(fs.readFileSync(absolute));
    hash.update('\0');
  };
  walk('');
  return hash.digest('hex');
}

// A package is stageable when the manifest names it and every entry it declares
// is a real file: a half-copied plugin that cannot load is worse than a stage
// that stopped.
export function verifyPluginSource(source, name) {
  const manifestPath = path.join(source, 'package.json');
  if (!fs.statSync(manifestPath, {throwIfNoEntry: false})?.isFile()) throw new Error(`${name}: source package.json is missing at ${source}`);
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.name !== name) throw new Error(`${name}: source package.json names ${JSON.stringify(manifest.name)}`);
  const missing = [];
  for (const entry of [manifest.main, manifest.dsh?.bundle?.patch].filter(Boolean)) {
    if (!fs.statSync(path.join(source, entry), {throwIfNoEntry: false})?.isFile()) missing.push(entry);
  }
  for (const [key, value] of Object.entries(manifest.exports ?? {})) {
    const relative = String(value).replace(/^\.\//, '');
    if (!fs.statSync(path.join(source, relative), {throwIfNoEntry: false})?.isFile()) missing.push(key === '.' ? `${relative} (exports ".")` : `${relative} (exports ${key})`);
  }
  if (missing.length) throw new Error(`${name}: source package is incomplete: ${missing.join(', ')}`);
  return manifest;
}

/**
 * Stage one plugin into vendorDir. Idempotent: an identical copy is a no-op, a
 * different copy is replaced as a unit, and a failed swap leaves the previous
 * copy in place.
 * @param {{source: string, vendorDir: string, name: string}} input
 * @returns {{name: string, action: 'copied' | 'unchanged' | 'replaced', vendorPath: string}}
 */
export function stagePlugin({source, vendorDir, name}) {
  if (!fs.statSync(vendorDir, {throwIfNoEntry: false})?.isDirectory()) throw new Error(`runtime vendor directory is missing: ${vendorDir}`);
  verifyPluginSource(source, name);
  const target = path.join(vendorDir, name);
  if (!inside(vendorDir, target)) throw new Error(`plugin target escapes the runtime: ${target}`);
  const sourceDigest = digest(source);
  if (fs.existsSync(target)) {
    if (digest(target) === sourceDigest) return {name, action: 'unchanged', vendorPath: target};
    const parked = path.join(vendorDir, `.${name}.previous`);
    const fresh = path.join(vendorDir, `.${name}.staged`);
    for (const leftover of [parked, fresh]) if (fs.existsSync(leftover)) fs.rmSync(leftover, {recursive: true, force: true});
    fs.cpSync(source, fresh, {recursive: true});
    fs.renameSync(target, parked);
    try {
      fs.renameSync(fresh, target);
    } catch (error) {
      fs.renameSync(parked, target);
      fs.rmSync(fresh, {recursive: true, force: true});
      throw error;
    }
    fs.rmSync(parked, {recursive: true, force: true});
    return {name, action: 'replaced', vendorPath: target};
  }
  fs.cpSync(source, target, {recursive: true});
  return {name, action: 'copied', vendorPath: target};
}

/**
 * Stage every tracked plugin into vendorDir. `without` names the plugins a build
 * deliberately leaves out; anything else that cannot be staged throws.
 * @param {{projectRoot: string, vendorDir: string, without?: string[]}} input
 */
export function stageThirdPartyPlugins({projectRoot, vendorDir, without = []}) {
  const staged = [];
  for (const name of THIRD_PARTY_PLUGINS) {
    if (without.includes(name)) {
      staged.push({name, action: 'skipped', vendorPath: '', reason: 'excluded by request'});
      continue;
    }
    staged.push(stagePlugin({source: path.join(projectRoot, 'third_party', name), vendorDir, name}));
  }
  return staged;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [vendorDir, ...flags] = process.argv.slice(2);
  const unknown = flags.filter(flag => !flag.startsWith('--without='));
  if (!vendorDir || vendorDir.startsWith('--') || unknown.length) {
    throw new Error('Usage: stage-third-party-plugins.mjs <runtime vendor dir> [--without=<plugin>]');
  }
  const without = flags.map(flag => flag.slice('--without='.length)).filter(Boolean);
  const projectRoot = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
  const staged = stageThirdPartyPlugins({projectRoot, vendorDir, without});
  for (const entry of staged) {
    console.log(`${entry.name}: ${entry.action}${entry.reason ? ` (${entry.reason})` : ''}`);
  }
}
