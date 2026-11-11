// Describe already built/signed artifacts. This command does not build, publish,
// upload, install, activate a candidate or run a target executable.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {verifyRuntime as verifyHost, supportedCommits} from './current-host.mjs';
import {guardID as sessionGuard} from './session-runtime-readiness.mjs';
import {verify as verifyMobile} from './mobile-runtime.mjs';

const application = 'whaleisle-mygo';
const guard = 'mygo-visual-readiness-v1';
const tagPattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/;
const maxInstallerBytes = 8 * 2**30;
const blockBytes = 4 * 2**20;
async function hash(file) {
  const digest = crypto.createHash('sha256');
  for await (const chunk of fs.createReadStream(file)) digest.update(chunk);
  return digest.digest('hex');
}
function regular(file) {
  const st = fs.lstatSync(file);
  if (!st.isFile() || st.isSymbolicLink()) throw Error('Expected regular artifact: ' + file);
  return st;
}
function peMachine(file) {
  regular(file);
  const fd = fs.openSync(file, 'r');
  try {
    const dos = Buffer.alloc(64);
    if (fs.readSync(fd, dos, 0, 64, 0) !== 64 || dos.toString('ascii', 0, 2) !== 'MZ') throw Error('Missing Windows executable header');
    const pe = Buffer.alloc(6);
    if (fs.readSync(fd, pe, 0, 6, dos.readUInt32LE(60)) !== 6 || pe.readUInt32LE(0) !== 0x4550) throw Error('Invalid PE header');
    return pe.readUInt16LE(4);
  } finally { fs.closeSync(fd); }
}
function fullID(release) {
  return [release.tag, release.platform, release.arch, 'full', release.runtimeIncluded ? 'runtime' : 'desktop'].join('/');
}
function packageID(release) {
  if (release.packageKind === 'delta') return fullID(release).replace('/full/', '/delta/') + '/' + release.delta.baseVersion + '/' + release.delta.baseSHA256.toLowerCase();
  return fullID(release);
}
function assetURL(raw, name, tag) {
  const base = new URL(raw);
  if (base.protocol !== 'https:' || base.username || base.password || base.search || base.hash || !base.pathname.endsWith('/')) throw Error('HTTPS release-directory baseURL is required');
  if (base.hostname === 'github.com' && base.pathname !== '/ChisaAlter/WhaleIsle-MyGo/releases/download/'+tag+'/') throw Error('GitHub URL must name the exact native repository/tag');
  return new URL(encodeURIComponent(name), base).href;
}
function downloadURLs(config, installer, tag) {
  const route = config.route || 'github';
  if (!['github', 'cnb'].includes(route)) throw Error('Release route must be github or cnb');
  const urls = {[route]: assetURL(config.baseURL, path.basename(installer), tag)};
  for (const [mirror, base] of Object.entries(config.mirrorBaseURLs || {})) {
    if (!['github', 'cnb'].includes(mirror)) throw Error('Mirror route must be github or cnb');
    urls[mirror] = assetURL(base, path.basename(installer), tag);
  }
  return {url: urls[route], downloadURLs: urls};
}
async function inspectWindows(stage, installer, arch) {
  const expectedMachine = arch === 'amd64' ? 0x8664 : 0xaa64;
  if (peMachine(path.join(stage, 'whaleisle.exe')) !== expectedMachine) throw Error('MyGo executable target mismatch');
  // NSIS may use a 32-bit loader for a 64-bit payload.
  peMachine(installer);
  const runtimeIncluded = fs.existsSync(path.join(stage, '.runtime-staged'));
  if (!runtimeIncluded) return {runtimeIncluded};
  if (fs.readFileSync(path.join(stage, '.runtime-staged'), 'utf8').trim() !== '1') throw Error('Invalid runtime staging marker');
  const vendor = path.join(stage, 'runtime/vendor');
  const host = await verifyHost(vendor);
  const installedProfile = {
    '687f282e8625c6b8d11ff58211a90524be2e4317': 'installed-rc2',
    '504a01a9578abc2c6f1fb896b1dc322c9fe24f66': 'installed-rc4',
  }[host.sourceCommit];
  const guarded = supportedCommits.includes(host.sourceCommit) && host.safetyPatch?.id === guard || installedProfile !== undefined && host.buildProfile === installedProfile && host.safetyPatch?.id === sessionGuard;
  if (!guarded || host.platform !== 'win32' || host.arch !== (arch === 'amd64' ? 'x64' : 'arm64')) throw Error('Host is not the native MyGo-compatible runtime');
  verifyMobile(path.join(vendor, 'dshd-remote'));
  if (peMachine(path.join(stage, 'runtime/node/node.exe')) !== expectedMachine) throw Error('Bundled Node target mismatch');
  return {runtimeIncluded, hostSourceCommit: host.sourceCommit, hostReadinessGuard: host.safetyPatch.id};
}

// Native deltas reuse exact byte ranges of a cached, signed base installer. The
// changed bytes live in the same final installer asset and are requested with
// HTTP Range; no fictitious binary patch is published or executed.
async function rangeDelta(installer, baseInstaller, release, base) {
  const oldStat = regular(baseInstaller);
  if (oldStat.size !== base.size || await hash(baseInstaller) !== base.sha256.toLowerCase()) throw Error('Delta base does not match its previous catalog SHA-256/size');
  const next = fs.openSync(installer, 'r'), old = fs.openSync(baseInstaller, 'r');
  const blocks = [];
  let downloadSize = 0, reused = 0;
  try {
    for (let offset = 0; offset < release.size; offset += blockBytes) {
      const size = Math.min(blockBytes, release.size - offset);
      const bytes = Buffer.alloc(size);
      if (fs.readSync(next, bytes, 0, size, offset) !== size) throw Error('Installer changed while generating native block map');
      const sha256 = crypto.createHash('sha256').update(bytes).digest('hex');
      blocks.push({offset, size, sha256});
      const previous = Buffer.alloc(size);
      if (offset + size <= oldStat.size && fs.readSync(old, previous, 0, size, offset) === size && crypto.createHash('sha256').update(previous).digest('hex') === sha256) reused += size;
      else downloadSize += size;
    }
  } finally { fs.closeSync(next); fs.closeSync(old); }
  if (!reused || downloadSize >= release.size) return null;
  const delta = {algorithm: 'sha256-range-v1', basePackageID: fullID(base), baseVersion: base.version,
    baseSHA256: base.sha256.toLowerCase(), baseSize: base.size, fullPackageID: release.packageID,
    downloadSize, blockMap: {fileSize: release.size, blocks}};
  const entry = {...release, packageKind: 'delta', delta};
  entry.packageID = packageID(entry);
  return entry;
}

export async function buildCatalog(config) {
  const version = String(config.version || '');
  const tag = String(config.tag || 'v' + version);
  if (!version.trim() || version.trim() !== version || !tagPattern.test(tag) || ![version, 'v'+version, 'mygo-v'+version].includes(tag)) throw Error('Exact version and matching release tag are required');
  let previous = [];
  if (config.previous) {
    const old = JSON.parse(fs.readFileSync(config.previous, 'utf8'));
    if (![1, 2].includes(old.schemaVersion) || old.application !== application || !Array.isArray(old.releases)) throw Error('Previous file is not a native catalog');
    previous = old.releases.map(r => ({...r, packageKind: r.packageKind || 'full', packageID: packageID(r)}));
  }
  const emitted = [];
  // packages permits desktop-only and runtime installers at the same tag.
  for (const item of config.packages || [config]) {
    const spec = {...config, ...item};
    const platform = spec.platform || 'windows', arch = spec.arch || 'amd64';
    if (!['amd64', 'arm64'].includes(arch) || !['windows', 'darwin'].includes(platform)) throw Error('Unsupported native target');
    const stage = path.resolve(spec.stage), installer = path.resolve(spec.installer);
    const stat = regular(installer);
    const format = platform === 'darwin' ? 'dmg' : 'nsis';
    if (!installer.toLowerCase().endsWith(format === 'dmg' ? '.dmg' : '.exe') || stat.size < 1 || stat.size > maxInstallerBytes) throw Error('Invalid native installer size/type');
    let runtime;
    if (platform === 'darwin') {
      const {inspectMacOSApp, inspectDMG} = await import('./macos-native-artifacts.mjs');
      inspectDMG(installer);
      const app = await inspectMacOSApp(stage, arch);
      if (app.version !== version) throw Error('macOS application version does not match catalog');
      runtime = {runtimeIncluded: app.runtimeIncluded, ...(app.runtimeIncluded ? {hostSourceCommit: app.hostSourceCommit, hostReadinessGuard: app.hostReadinessGuard} : {})};
    } else runtime = await inspectWindows(stage, installer, arch);
    const publishedAt = spec.publishedAt ? String(spec.publishedAt) : new Date().toISOString();
    if (!/^\d{4}-\d{2}-\d{2}T/.test(publishedAt) || !Number.isFinite(Date.parse(publishedAt))) throw Error('publishedAt must be an ISO release timestamp');
    const release = {version, tag, ...downloadURLs(spec, installer, tag), sha256: await hash(installer), size: stat.size,
      publishedAt, notes: String(spec.notes || ''), notesZH: String(spec.notesZH || ''), notesEN: String(spec.notesEN || ''),
      platform, arch, format, prerelease: spec.prerelease === true, ...runtime, packageKind: 'full'};
    release.packageID = fullID(release);
    emitted.push(release);
    for (const input of spec.deltaBases || []) {
      if (platform !== 'windows') throw Error('Native range deltas currently support Windows NSIS only');
      const base = previous.find(r => r.packageID === input.packageID && r.packageKind === 'full');
      if (!base || base.platform !== platform || base.arch !== arch || base.runtimeIncluded !== release.runtimeIncluded || base.version === version) throw Error('Delta base must identify a previous full package with the same target/runtime variant');
      const delta = await rangeDelta(installer, path.resolve(input.installer), release, base);
      if (delta) emitted.push(delta);
    }
  }
  const ids = new Set();
  for (const release of emitted) {
    if (ids.has(release.packageID)) throw Error('Duplicate native package identity: ' + release.packageID);
    ids.add(release.packageID);
  }
  // Replacing one variant replaces its old range recipes, but keeps the other
  // runtime variant and architectures at this tag.
  const replaced = new Set(emitted.filter(r => r.packageKind === 'full').map(fullID));
  const releases = [...emitted, ...previous.filter(r => !replaced.has(fullID(r)))];
  if (releases.length > 500) throw Error('Catalog exceeds 500 releases');
  const catalog = {schemaVersion: 2, application, releases};
  if (Buffer.byteLength(JSON.stringify(catalog, null, 2)+'\n') > 2**21) throw Error('Catalog exceeds client 2 MiB limit; retain fewer historical delta recipes');
  return catalog;
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const config = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
    if (!config.output) throw Error('config.output is required');
    const catalog = await buildCatalog(config);
    fs.writeFileSync(config.output, JSON.stringify(catalog, null, 2)+'\n', {flag:'wx'});
    console.log('Native catalog created: ' + config.output);
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
