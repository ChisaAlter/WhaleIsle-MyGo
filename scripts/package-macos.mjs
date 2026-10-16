// 功能优先；全部功能完成前不运行测试、不做独立审查、不新增防御性扩展
// Manual packaging only: node scripts/package-macos.mjs stage|archive config.json
// Inputs are already-built native artifacts. This entry never builds source,
// downloads dependencies, starts Host, signs/notarizes, installs, or uploads.
// stage config: {version, arch, output, shell, terminalLibrary, vendor, nodeDir,
//                runtimeIncluded:true, icon?:"local.icns"}
// archive config: {app, arch, output, hostBundle:true}
// Use a fresh output directory for each phase. Apply the existing external
// signing workflow between stage/archive and before publishing the catalog.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';
import {inspectMacOSApp, inspectMacOSRuntime, inspectMachO, inspectDMG, sha256, xml,
  macOSApplication, macOSBundleIdentifier, macOSExecutable, macOSPackageManifest} from './macos-native-artifacts.mjs';

const projectRoot = fileURLToPath(new URL('../', import.meta.url));
const readJSON = file => JSON.parse(fs.readFileSync(file, 'utf8'));
const writeJSON = (file, value) => fs.writeFileSync(file, JSON.stringify(value, null, 2) + '\n', {flag: 'wx'});

function target(arch) {
  if (process.platform !== 'darwin' || !['amd64', 'arm64'].includes(arch) ||
      process.arch !== (arch === 'amd64' ? 'x64' : arch)) throw Error('Run packaging with matching-architecture Node on macOS; native Host dependencies cannot be cross-staged');
}
function fresh(output) {
  if (!output) throw Error('A fresh output directory is required');
  const root = path.resolve(output);
  if (fs.existsSync(root)) throw Error('Output already exists; choose a fresh directory: ' + root);
  fs.mkdirSync(root, {recursive: true});
  return root;
}
function input(value, label) {
  if (!value) throw Error(label + ' is required');
  return fs.realpathSync(value);
}
function run(command, args, options = {}) {
  const result = spawnSync(command, args, {stdio: 'inherit', ...options});
  if (result.error || result.status !== 0) throw Error(`${command} failed: ${result.error?.message || result.status}`);
}
function copyTree(source, destination) {
  source = fs.realpathSync(source);
  fs.cpSync(source, destination, {recursive: true, verbatimSymlinks: true});
  // Keep pnpm/workspace links usable after the app is moved to /Applications.
  function relocate(relative) {
    const from = path.join(source, relative), to = path.join(destination, relative);
    const stat = fs.lstatSync(from);
    if (stat.isSymbolicLink()) {
      const link = fs.readlinkSync(from);
      const resolved = path.relative(source, fs.realpathSync(from));
      if (resolved === '..' || resolved.startsWith('..' + path.sep) || path.isAbsolute(resolved)) throw Error('Runtime dependency must be inside the staged tree: ' + from);
      if (path.isAbsolute(link)) {
        fs.unlinkSync(to);
        fs.symlinkSync(path.relative(path.dirname(to), path.join(destination, resolved)), to);
      }
    } else if (stat.isDirectory()) {
      for (const name of fs.readdirSync(from)) relocate(path.join(relative, name));
    }
  }
  relocate('');
}

export async function stage(config) {
  target(config.arch);
  const version = String(config.version || '');
  if (!/^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$/.test(version)) throw Error('A release version such as 1.2.3 or 1.2.3-beta.1 is required');
  const bundleVersion = String(config.bundleVersion || version.split(/[-+]/)[0]);
  if (!/^[0-9]+\.[0-9]+\.[0-9]+$/.test(bundleVersion)) throw Error('bundleVersion must contain three numeric components');
  const shell = input(config.shell, 'Prebuilt MyGo shell');
  inspectMachO(shell, config.arch);
  const library = input(config.terminalLibrary || process.env.WHALEISLE_TERMINAL_LIBRARY, 'Pinned terminalLibrary');
  const manifest = readJSON(path.join(projectRoot, 'internal/desktop/terminalruntime/native-manifest.json'));
  const binding = readJSON(path.join(projectRoot, 'third_party/mygo/plugins/terminal/mygo-plugin.json'));
  const librarySpec = manifest.libraries.find(l => l.name === 'libghostty-vt');
  const spec = librarySpec?.files?.['darwin-' + config.arch];
  const bound = binding.libraries.find(l => l.name === 'libghostty-vt');
  const boundSpec = bound?.files?.['darwin-' + config.arch];
  if (!spec || !boundSpec || ['name', 'sha256', 'url'].some(key => spec[key] !== boundSpec[key]) ||
      librarySpec.source !== bound.source || await sha256(library) !== spec.sha256) throw Error('Supply the pinned native terminal library matching the checked-in MyGo binding');
  inspectMachO(library, config.arch, 6);
  const runtimeIncluded = config.runtimeIncluded !== false;
  const vendor = runtimeIncluded ? input(config.vendor, 'Darwin vendor directory') : '';
  const nodeDir = runtimeIncluded ? input(config.nodeDir, 'Official Darwin Node distribution') : '';
  if (runtimeIncluded) inspectMachO(path.join(nodeDir, 'bin/node'), config.arch);
  const output = fresh(config.output);
  const app = path.join(output, 'Whale Isle.app');
  const bin = path.join(app, 'Contents/MacOS');
  const resources = path.join(app, 'Contents/Resources');
  fs.mkdirSync(bin, {recursive: true});
  fs.mkdirSync(resources, {recursive: true});
  fs.copyFileSync(shell, path.join(bin, 'whaleisle'));
  fs.chmodSync(path.join(bin, 'whaleisle'), 0o755);
  fs.copyFileSync(library, path.join(bin, spec.name));
  fs.copyFileSync(path.join(projectRoot, 'internal/desktop/terminalruntime/Ghostty-LICENSE'), path.join(bin, spec.name + '.LICENSE'));
  writeJSON(path.join(bin, 'terminal-runtime.json'), {...spec, Target: 'darwin-' + config.arch, Source: librarySpec.source});
  copyTree(path.join(projectRoot, 'resources'), path.join(bin, 'resources'));
  let runtime = {};
  if (runtimeIncluded) {
    const runtimeDir = path.join(bin, 'runtime');
    copyTree(vendor, path.join(runtimeDir, 'vendor'));
    copyTree(nodeDir, path.join(runtimeDir, 'node'));
    // Desktop Host resolution uses runtime/node/node; mobile keeps its own
    // exact-ABI distribution under runtime/vendor/dshd-remote/node/bin/node.
    fs.symlinkSync('bin/node', path.join(runtimeDir, 'node/node'));
    runtime = await inspectMacOSRuntime(runtimeDir, config.arch);
    fs.writeFileSync(path.join(bin, '.runtime-staged'), '1', {flag: 'wx'});
  }
  const values = {
    CFBundleName: 'Whale Isle', CFBundleDisplayName: 'Whale Isle',
    CFBundleIdentifier: macOSBundleIdentifier, CFBundleExecutable: 'whaleisle',
    CFBundleShortVersionString: bundleVersion, CFBundleVersion: bundleVersion,
    CFBundlePackageType: 'APPL', CFBundleInfoDictionaryVersion: '6.0',
    NSPrincipalClass: 'NSApplication', LSMinimumSystemVersion: '11.0',
  };
  if (config.icon) {
    fs.copyFileSync(input(config.icon, 'ICNS icon'), path.join(resources, 'AppIcon.icns'));
    values.CFBundleIconFile = 'AppIcon.icns';
  }
  const plist = '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0"><dict>\n' +
    Object.entries(values).map(([key, value]) => `<key>${key}</key><string>${xml(value)}</string>`).join('\n') +
    '\n<key>NSHighResolutionCapable</key><true/><key>NSSupportsAutomaticGraphicsSwitching</key><true/>\n' +
    '<key>CFBundleURLTypes</key><array><dict><key>CFBundleURLName</key><string>' + macOSBundleIdentifier +
    '</string><key>CFBundleURLSchemes</key><array><string>whaleisle</string></array></dict></array>\n</dict></plist>\n';
  fs.writeFileSync(path.join(app, 'Contents/Info.plist'), plist, {flag: 'wx'});
  fs.writeFileSync(path.join(app, 'Contents/PkgInfo'), 'APPL????', {flag: 'wx'});
  writeJSON(path.join(app, macOSPackageManifest), {schemaVersion: 1, application: macOSApplication,
    bundleIdentifier: macOSBundleIdentifier, version, bundleVersion, platform: 'darwin', arch: config.arch,
    executable: macOSExecutable, runtimeIncluded, ...runtime});
  await inspectMacOSApp(app, config.arch);
  return {app, version, arch: config.arch, runtimeIncluded,
    next: 'Use the existing external signing workflow, then run archive explicitly. No signing or publishing was performed.'};
}

export async function archive(config) {
  target(config.arch);
  const app = input(config.app, 'Staged MyGo .app');
  const info = await inspectMacOSApp(app, config.arch);
  const output = fresh(config.output);
  const stem = `WhaleIsle-MyGo-${info.version}-darwin-${config.arch}`;
  const disk = path.join(output, 'disk');
  fs.mkdirSync(disk);
  // ditto retains the app's existing signatures and resource metadata. Neither
  // this command nor hdiutil signs or grants permissions to the application.
  run('/usr/bin/ditto', [app, path.join(disk, 'Whale Isle.app')]);
  fs.symlinkSync('/Applications', path.join(disk, 'Applications'));
  const dmg = path.join(output, stem + '.dmg');
  run('/usr/bin/hdiutil', ['create', '-volname', 'Whale Isle', '-srcfolder', disk, '-format', 'UDZO', '-o', dmg]);
  inspectDMG(dmg);
  const artifacts = [{file: path.basename(dmg), format: 'dmg', platform: 'darwin', arch: config.arch,
    runtimeIncluded: info.runtimeIncluded, sha256: await sha256(dmg), size: fs.statSync(dmg).size}];
  if (config.hostBundle !== false && info.runtimeIncluded) {
    const work = path.join(output, 'host-bundle');
    fs.mkdirSync(work);
    copyTree(path.join(app, 'Contents/MacOS/runtime'), path.join(work, 'runtime'));
    writeJSON(path.join(work, 'WhaleIsle-Host-bundle.json'), {schemaVersion: 1, application: 'whaleisle-mygo-host',
      version: info.version, platform: 'darwin', arch: config.arch, runtimeDirectory: 'runtime',
      hostSourceCommit: info.hostSourceCommit, hostReadinessGuard: info.hostReadinessGuard,
      nodeVersion: info.nodeVersion, nodeABI: info.nodeABI});
    const host = path.join(output, `WhaleIsle-Host-${info.version}-darwin-${config.arch}.tar.gz`);
    run('/usr/bin/tar', ['-czf', host, '-C', work, 'runtime', 'WhaleIsle-Host-bundle.json'],
      {env: {...process.env, COPYFILE_DISABLE: '1'}});
    // The native launcher component page imports this offline archive, then
    // selects its runtime/ tree for the existing Host supervisor. It is not a DMG.
    artifacts.push({file: path.basename(host), format: 'tar.gz', role: 'offline-host-bundle',
      sha256: await sha256(host), size: fs.statSync(host).size});
  }
  writeJSON(path.join(output, 'macos-artifacts.json'), {schemaVersion: 1, application: macOSApplication,
    version: info.version, ...info, artifacts});
  return {app, output, artifacts, next: 'Sign/notarize through the existing release process, then emit the native catalog from the final bytes. Nothing was published or installed.'};
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [mode, file] = process.argv.slice(2);
    if (!['stage', 'archive'].includes(mode) || !file) throw Error('Usage: package-macos.mjs stage|archive config.json');
    const result = await ({stage, archive})[mode](readJSON(file));
    console.log(JSON.stringify(result, null, 2));
  } catch (error) { console.error(error.message); process.exitCode = 1; }
}
