// Emit a native installer catalog only after ordinary package/signing steps.
// This tool reads and hashes local artifacts. It does not run an installer,
// change a candidate's readiness, publish anything, or execute target packages.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {verify as verifyHost} from './current-host.mjs';
import {verify as verifyMobile} from './mobile-runtime.mjs';

const application = 'whaleisle-mygo';
const sourceCommit = '90465b67ec16765631d54c195af19d9ff3c3a770';
const guard = 'mygo-visual-readiness-v1';
const tagPattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/;
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
export async function buildCatalog(config) {
  const stage = path.resolve(config.stage);
  const installer = path.resolve(config.installer);
  const version = String(config.version || '');
  const tag = String(config.tag || 'v' + version);
  if (!version.trim() || version.trim() !== version || !tagPattern.test(tag) || ![version, 'v'+version, 'mygo-v'+version].includes(tag)) throw Error('Exact version and matching release tag are required');
  const base = new URL(config.baseURL);
  if (base.protocol !== 'https:' || base.username || base.password || base.search || base.hash || !base.pathname.endsWith('/')) throw Error('HTTPS release-directory baseURL is required');
  if (base.hostname === 'github.com' && base.pathname !== '/ChisaAlter/WhaleIsle-MyGo/releases/download/'+tag+'/') throw Error('GitHub URL must name the exact native repository/tag');
  const arch = config.arch || 'amd64';
  if (!['amd64', 'arm64'].includes(arch)) throw Error('Unsupported native target');
  const shell = path.join(stage, 'whaleisle.exe');
  const expectedMachine = arch === 'amd64' ? 0x8664 : 0xaa64;
  if (peMachine(shell) !== expectedMachine) throw Error('MyGo executable target mismatch');
  // NSIS itself may use a 32-bit loader for a 64-bit payload.
  peMachine(installer);
  const stat = regular(installer);
  if (!installer.toLowerCase().endsWith('.exe') || stat.size < 1 || stat.size > 8 * 2**30) throw Error('Invalid native installer size/type');
  const runtimeIncluded = fs.existsSync(path.join(stage, '.runtime-staged'));
  let hostSourceCommit, hostReadinessGuard;
  if (runtimeIncluded) {
    if (fs.readFileSync(path.join(stage, '.runtime-staged'), 'utf8').trim() !== '1') throw Error('Invalid runtime staging marker');
    const vendor = path.join(stage, 'runtime/vendor');
    const host = verifyHost(vendor);
    if (host.sourceCommit !== sourceCommit || host.safetyPatch?.id !== guard || host.platform !== 'win32' || host.arch !== (arch === 'amd64' ? 'x64' : 'arm64')) throw Error('Host is not the native MyGo-compatible runtime');
    verifyMobile(path.join(vendor, 'dshd-remote'));
    if (peMachine(path.join(stage, 'runtime/node/node.exe')) !== expectedMachine) throw Error('Bundled Node target mismatch');
    hostSourceCommit = host.sourceCommit;
    hostReadinessGuard = host.safetyPatch.id;
  }
  const release = {version, tag, url: new URL(path.basename(installer), base).href,
    sha256: await hash(installer), size: stat.size, notes: String(config.notes || ''),
    platform: 'windows', arch, format: 'nsis', prerelease: config.prerelease === true,
    runtimeIncluded, ...(runtimeIncluded ? {hostSourceCommit, hostReadinessGuard} : {})};
  let previous = [];
  if (config.previous) {
    const old = JSON.parse(fs.readFileSync(config.previous, 'utf8'));
    if (old.schemaVersion !== 1 || old.application !== application || !Array.isArray(old.releases)) throw Error('Previous file is not a native catalog');
    previous = old.releases.filter(r => !(r.tag === tag && r.platform === 'windows' && r.arch === arch));
  }
  const catalog = {schemaVersion: 1, application, releases: [release, ...previous]};
  if (catalog.releases.length > 500) throw Error('Catalog exceeds 500 releases');
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
