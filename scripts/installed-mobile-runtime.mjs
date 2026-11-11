// Complete a staged Host graph from the installed desktop's shipped resources.
// Reads only application files; no user profile, pairing identity or daemon starts.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {shellFiles, recordInstalled} from './mobile-runtime.mjs';
import {verifyStagedSessionRuntime} from './session-runtime.mjs';

function archiveReader(filename) {
  const bytes = fs.readFileSync(filename);
  const headerSize = bytes.readUInt32LE(4);
  const header = JSON.parse(bytes.subarray(16, 16 + bytes.readUInt32LE(12)).toString('utf8'));
  function entry(relative) {
    let value = header;
    for (const part of relative.split('/')) {
      if (!part || part === '.' || part === '..') throw Error('Invalid archive path');
      value = value?.files?.[part];
    }
    if (!value || value.link) throw Error(`Missing archive entry: ${relative}`);
    return value;
  }
  function read(relative) {
    const value = entry(relative);
    if (!Number.isSafeInteger(value.size)) throw Error(`Not an archive file: ${relative}`);
    if (value.unpacked) {
      const data = fs.readFileSync(path.join(filename + '.unpacked', relative));
      if (data.length !== value.size) throw Error(`Unpacked archive entry changed: ${relative}`);
      return data;
    }
    const offset = 8 + headerSize + Number(value.offset);
    if (!Number.isSafeInteger(offset) || offset < 8 + headerSize || offset + value.size > bytes.length) throw Error('Invalid archive offset');
    return bytes.subarray(offset, offset + value.size);
  }
  function files(relative) {
    const result = [];
    function walk(prefix, value) {
      if (value.files) for (const [name, child] of Object.entries(value.files)) walk(prefix + '/' + name, child);
      else result.push(prefix);
    }
    walk(relative, entry(relative));
    return result;
  }
  return {read, files};
}

export async function completeInstalledMobile(resources, stageRoot, webDestination = '') {
  resources = fs.realpathSync(resources);
  stageRoot = fs.realpathSync(stageRoot);
  const host = await verifyStagedSessionRuntime(path.join(stageRoot, 'vendor'));
  const archive = archiveReader(path.join(resources, 'app.asar'));
  const identity = JSON.parse(archive.read('package.json'));
  if (identity.name !== 'deepseek-harness-desktop' || identity.version !== '0.3.14' || host.cliVersion !== '0.2.1-rc.4') throw Error('Expected shipped Whale Isle 0.3.14 with the staged rc.4 Host');
  const root = path.join(stageRoot, 'vendor/dshd-remote');
  for (const relative of ['src', 'mobile', 'vendor', 'node.exe', 'staging-manifest.json']) {
    if (fs.existsSync(path.join(root, relative))) throw Error(`Mobile completion requires an unassembled stage: ${relative}`);
  }
  const write = (relative, bytes) => {
    const target = path.join(root, relative);
    fs.mkdirSync(path.dirname(target), {recursive: true});
    fs.writeFileSync(target, bytes, {flag: 'wx'});
  };
  const modifications = [];
  for (const relative of shellFiles) {
    let bytes = archive.read(relative);
    if (relative === 'src/main/dshd-remote.js') {
      const before = "return path.join(__dirname, '..', '..', 'vendor', 'deepseek-harness');";
      const source = bytes.toString('utf8');
      if (source.split(before).length !== 2) throw Error('Unrecognized mobile Host resolution');
      bytes = Buffer.from(source.replace(before, "return path.join(__dirname, '..', '..', '..', 'deepseek-harness');"));
      modifications.push({path: relative, change: 'Resolve the staged sibling Host used by the native desktop.'});
    }
    if (relative === 'src/main/mobile-web-server.js') {
      const source = bytes.toString('utf8'), before = "'x-frame-options': 'DENY',";
      if (source.split(before).length !== 2) throw Error('Unrecognized mobile framing policy');
      bytes = Buffer.from(source.replace(before, "'x-frame-options': asset.file === require('path').join(root, 'whale-ui', 'viewer.html') ? 'SAMEORIGIN' : 'DENY',"));
      modifications.push({path: relative, change: 'Allow the same-origin Whale UI sandbox viewer; other pages retain DENY.'});
    }
    write(relative, bytes);
  }
  for (const relative of ['src/shared/dshd-account-callback.js', 'src/shared/dshd-workspace-image.js']) write(relative, archive.read(relative));
  const webFiles = archive.files('mobile/web').filter(name => !name.endsWith('.map') && !name.includes('.fixture.') && !name.includes('.test.'));
  for (const relative of webFiles) {
    const bytes = archive.read(relative);
    write(relative, bytes);
    if (webDestination) {
      const target = path.join(webDestination, relative.slice('mobile/web/'.length));
      fs.mkdirSync(path.dirname(target), {recursive: true});
      fs.writeFileSync(target, bytes);
    }
  }
  // Reuse the installed SQLite/PTY binaries with their own Electron ABI.
  const appRoot = path.dirname(resources);
  write('node.exe', fs.readFileSync(path.join(appRoot, identity.productName + '.exe')));
  const runtimeFiles = fs.readdirSync(appRoot).filter(name => /\.(?:dll|pak|bin)$/.test(name) || ['icudtl.dat', 'LICENSE.electron.txt', 'LICENSES.chromium.html', 'vk_swiftshader_icd.json'].includes(name));
  for (const name of runtimeFiles) write(name, fs.readFileSync(path.join(appRoot, name)));
  const nested = path.join(root, 'vendor/chisacode-remote');
  fs.mkdirSync(nested, {recursive: true});
  fs.symlinkSync(process.platform === 'win32' ? path.join(root, 'node_modules') : '../../node_modules', path.join(nested, 'node_modules'), process.platform === 'win32' ? 'junction' : 'dir');
  for (const name of ['LICENSE', 'NOTICE']) fs.copyFileSync(path.join(root, name), path.join(nested, name), fs.constants.COPYFILE_EXCL);
  write('source-provenance.json', JSON.stringify({schemaVersion: 1, profile: 'installed-rc4', source: {product: identity.name, version: identity.version, resources}, modifications}, null, 2) + '\n');
  return recordInstalled(root);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [resources, stageRoot, webDestination, extra] = process.argv.slice(2);
  if (!resources || !stageRoot || extra) throw Error('Usage: installed-mobile-runtime.mjs <installed resources> <staged runtime> [web destination]');
  console.log(JSON.stringify(await completeInstalledMobile(resources, stageRoot, webDestination), null, 2));
}
