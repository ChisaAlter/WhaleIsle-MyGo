// Optional, already-signed preview bundle packaging. This module never builds,
// signs, launches, requests credentials, or claims WebKit runtime containment.
import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
import {inspectMachO} from './macos-native-artifacts.mjs';

export const previewBundleName = 'WhaleIsle Preview.app';
export const previewBundleIdentifier = 'org.whaleisle.preview';

export function validatePreviewEntitlements(value) {
  if (!value || Array.isArray(value) || Object.keys(value).length !== 1 || value['com.apple.security.app-sandbox'] !== true)
    throw Error('Preview helper requires exactly com.apple.security.app-sandbox=true; no network, user-selected files, inheritance, exceptions or other entitlements');
}
export function signingTeam(details) {
  const matches = details.split('\n').filter(line => line.startsWith('TeamIdentifier='));
  if (matches.length !== 1 || !/^TeamIdentifier=[A-Z0-9]{10}$/.test(matches[0]))
    throw Error('Preview packaging requires an unambiguous Apple-issued signing team');
  return matches[0].slice('TeamIdentifier='.length);
}
function command(executable, args, input) {
  const result = spawnSync(executable, args, {encoding: 'utf8', input, timeout: 15000, maxBuffer: 1024 * 1024,
    env: {PATH: '/usr/bin:/bin', LANG: 'C', LC_ALL: 'C'}});
  if (result.error || result.status !== 0) throw Error(`${executable} failed: ${result.error?.message || result.stderr || result.status}`);
  return result;
}
function plistJSON(file) {
  return JSON.parse(command('/usr/bin/plutil', ['-convert', 'json', '-o', '-', file]).stdout);
}
function noSymlinks(root) {
  const stat = fs.lstatSync(root);
  if (stat.isSymbolicLink()) throw Error('Preview bundle must not contain symbolic links: ' + root);
  if (stat.isDirectory()) for (const name of fs.readdirSync(root)) noSymlinks(path.join(root, name));
  else if (!stat.isFile()) throw Error('Preview bundle contains a nonregular file: ' + root);
}
export function inspectPreviewBundle(bundle, arch) {
  if (process.platform !== 'darwin') throw Error('Preview signature inspection requires macOS');
  bundle = path.resolve(bundle);
  noSymlinks(bundle);
  const info = plistJSON(path.join(bundle, 'Contents/Info.plist'));
  if (info.CFBundleIdentifier !== previewBundleIdentifier || info.CFBundleExecutable !== 'whaleisle-preview' || info.CFBundlePackageType !== 'APPL')
    throw Error('Unexpected preview helper bundle identity');
  const executable = path.join(bundle, 'Contents/MacOS/whaleisle-preview');
  inspectMachO(executable, arch);
  command('/usr/bin/codesign', ['--verify', '--deep', '--strict', '-R',
    `anchor apple generic and identifier "${previewBundleIdentifier}"`, bundle]);
  const detail = command('/usr/bin/codesign', ['-d', '--verbose=4', bundle]);
  const team = signingTeam(detail.stderr);
  const entitlement = command('/usr/bin/codesign', ['-d', '--entitlements', ':-', executable]).stdout;
  // Only codesign's stdout is parsed; diagnostics on stderr cannot become a plist.
  const value = JSON.parse(command('/usr/bin/plutil', ['-convert', 'json', '-o', '-', '--', '-'], entitlement).stdout);
  validatePreviewEntitlements(value);
  return {bundleIdentifier: previewBundleIdentifier, teamIdentifier: team};
}
export function stagePreviewBundle(source, app, arch) {
  inspectPreviewBundle(source, arch);
  const destination = path.join(app, 'Contents/Helpers', previewBundleName);
  if (fs.existsSync(destination)) throw Error('Preview helper destination already exists');
  fs.mkdirSync(path.dirname(destination), {recursive: true});
  // ditto preserves the existing signature and executable metadata. Verify the
  // copied artifact too; source inspection alone does not validate delivery.
  command('/usr/bin/ditto', [path.resolve(source), destination]);
  return inspectPreviewBundle(destination, arch);
}
export function inspectEmbeddedPreview(app, arch) {
  const bundle = path.join(app, 'Contents/Helpers', previewBundleName);
  if (!fs.existsSync(bundle)) return null; // Existing packages remain unchanged.
  const preview = inspectPreviewBundle(bundle, arch);
  command('/usr/bin/codesign', ['--verify', '--deep', '--strict', '-R', 'anchor apple generic', app]);
  const detail = command('/usr/bin/codesign', ['-d', '--verbose=4', app]);
  if (signingTeam(detail.stderr) !== preview.teamIdentifier) throw Error('Preview helper and host must have the same signing team');
  return preview;
}
