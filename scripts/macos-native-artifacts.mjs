// Native macOS package contract. Reads bytes/provenance; never starts an app,
// loads a native module, signs anything, mounts a disk image, or publishes it.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {verify as verifyHost} from './current-host.mjs';
import {verify as verifyMobile} from './mobile-runtime.mjs';

export const macOSApplication = 'whaleisle-mygo';
export const macOSBundleIdentifier = 'ai.deepseek.harness.gui';
export const macOSExecutable = 'Contents/MacOS/whaleisle';
export const macOSPackageManifest = 'Contents/Resources/native-macos-package.json';
export const hostSourceCommit = '90465b67ec16765631d54c195af19d9ff3c3a770';
export const hostReadinessGuard = 'mygo-visual-readiness-v1';
const readJSON = file => JSON.parse(fs.readFileSync(file, 'utf8'));

export async function sha256(file) {
  const digest = crypto.createHash('sha256');
  for await (const chunk of fs.createReadStream(file)) digest.update(chunk);
  return digest.digest('hex');
}

export function inspectMachO(file, arch, fileType = 2) {
  const cpu = {amd64: 0x01000007, arm64: 0x0100000c}[arch];
  if (!cpu) throw Error('macOS packages require amd64 or arm64');
  const stat = fs.statSync(file);
  if (!stat.isFile() || stat.size < 32) throw Error('Missing Mach-O file: ' + file);
  const fd = fs.openSync(file, 'r');
  try {
    const header = Buffer.alloc(32);
    const matches = offset => fs.readSync(fd, header, 0, 32, offset) === 32 &&
      header.readUInt32LE(0) === 0xfeedfacf && header.readUInt32LE(4) === cpu &&
      header.readUInt32LE(12) === fileType;
    if (matches(0)) return {arch, size: stat.size};
    const magic = header.readUInt32BE(0);
    if (magic === 0xcafebabe || magic === 0xcafebabf) {
      const count = header.readUInt32BE(4), width = magic === 0xcafebabf ? 32 : 20;
      if (count > 0 && count <= 32) {
        const entry = Buffer.alloc(width);
        for (let i = 0; i < count; i++) {
          if (fs.readSync(fd, entry, 0, width, 8 + i * width) !== width) break;
          if (entry.readUInt32BE(0) !== cpu) continue;
          const offset = width === 32 ? Number(entry.readBigUInt64BE(8)) : entry.readUInt32BE(8);
          if (Number.isSafeInteger(offset) && offset <= stat.size - 32 && matches(offset)) return {arch, size: stat.size};
        }
      }
    }
    throw Error(`Expected ${arch} Mach-O ${fileType === 2 ? 'executable' : 'library'}: ${file}`);
  } finally { fs.closeSync(fd); }
}

export function inspectDMG(file) {
  const stat = fs.lstatSync(file);
  if (!stat.isFile() || stat.size < 512 || stat.size > 8 * 2**30 || !file.toLowerCase().endsWith('.dmg')) throw Error('Expected a native DMG artifact');
  const fd = fs.openSync(file, 'r');
  try {
    const trailer = Buffer.alloc(4);
    if (fs.readSync(fd, trailer, 0, 4, stat.size - 512) !== 4 || trailer.toString('ascii') !== 'koly') throw Error('Missing UDIF disk-image trailer');
  } finally { fs.closeSync(fd); }
  return {size: stat.size};
}

export async function inspectMacOSRuntime(runtimeDir, arch) {
  const nodeArch = arch === 'amd64' ? 'x64' : arch;
  const vendor = path.join(runtimeDir, 'vendor');
  const host = verifyHost(vendor);
  if (host.platform !== 'darwin' || host.arch !== nodeArch || host.sourceCommit !== hostSourceCommit || host.safetyPatch?.id !== hostReadinessGuard) throw Error('Host does not match the native macOS package target');
  const mobile = path.join(vendor, 'dshd-remote');
  verifyMobile(mobile);
  const build = readJSON(path.join(mobile, 'node-build.json'));
  if (build.platform !== 'darwin' || build.arch !== nodeArch) throw Error('Mobile native modules target a different platform/architecture');
  const node = path.join(runtimeDir, 'node', 'node');
  inspectMachO(node, arch);
  if (await sha256(node) !== build.nodeSHA256) throw Error('Host must bundle the same Darwin Node used for mobile native modules');
  return {hostSourceCommit: host.sourceCommit, hostReadinessGuard: host.safetyPatch.id,
    nodeVersion: build.node, nodeABI: build.abi};
}

export async function inspectMacOSApp(app, arch) {
  const manifest = readJSON(path.join(app, macOSPackageManifest));
  if (manifest.schemaVersion !== 1 || manifest.application !== macOSApplication ||
      manifest.bundleIdentifier !== macOSBundleIdentifier || manifest.platform !== 'darwin' ||
      manifest.arch !== arch || manifest.executable !== macOSExecutable ||
      typeof manifest.runtimeIncluded !== 'boolean' || !manifest.version) throw Error('Unsupported native macOS app manifest');
  const plist = fs.readFileSync(path.join(app, 'Contents/Info.plist'), 'utf8');
  for (const [key, value] of Object.entries({CFBundleIdentifier: macOSBundleIdentifier, CFBundleExecutable: 'whaleisle', CFBundleShortVersionString: manifest.bundleVersion})) {
    if (!plist.includes(`<key>${key}</key><string>${xml(value)}</string>`)) throw Error('App Info.plist differs from its native package manifest: ' + key);
  }
  const binDir = path.join(app, 'Contents/MacOS');
  inspectMachO(path.join(app, macOSExecutable), arch);
  const terminal = readJSON(path.join(binDir, 'terminal-runtime.json'));
  const pins = readJSON(new URL('../internal/desktop/terminalruntime/native-manifest.json', import.meta.url));
  const spec = pins.libraries.find(l => l.name === 'libghostty-vt')?.files?.['darwin-' + arch];
  if (!spec || terminal.Target !== 'darwin-' + arch || terminal.sha256 !== spec.sha256 ||
      await sha256(path.join(binDir, spec.name)) !== spec.sha256) throw Error('Packaged terminal library differs from the pinned MyGo ABI');
  const runtime = manifest.runtimeIncluded ? await inspectMacOSRuntime(path.join(binDir, 'runtime'), arch) : {};
  if (manifest.runtimeIncluded && fs.readFileSync(path.join(binDir, '.runtime-staged'), 'utf8').trim() !== '1') throw Error('Missing native runtime staging marker');
  return {version: manifest.version, platform: 'darwin', arch, executable: macOSExecutable,
    runtimeIncluded: manifest.runtimeIncluded, ...runtime};
}

export function xml(value) {
  return String(value).replace(/[&<>"']/g, c => ({'&':'&amp;', '<':'&lt;', '>':'&gt;', '"':'&quot;', "'":'&apos;'}[c]));
}
