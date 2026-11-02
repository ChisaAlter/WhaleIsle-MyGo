// Isolated ABI probe: only an in-memory database and native binding loads.
// Never import the daemon, create a home/identity, bind a port, or launch a PTY.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const {createRequire} = require('node:module');
const [rootArg, output] = process.argv.slice(2);
const root = fs.realpathSync(rootArg);
const sha = file => crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const inside = file => { const relative = path.relative(root, fs.realpathSync(file)); if (path.isAbsolute(relative) || relative === '..' || relative.startsWith('..' + path.sep)) throw Error('Native binding escapes mobile runtime'); return relative.split(path.sep).join('/'); };
const requireRuntime = createRequire(path.join(root, 'vendor/chisacode-remote/node_modules/@chisacode/server/package.json'));
const Database = requireRuntime('better-sqlite3');
const db = new Database(':memory:');
try { if (db.prepare('select 1 as ok').get().ok !== 1) throw Error('SQLite probe failed'); } finally { db.close(); }
requireRuntime('node-pty');
if (process.platform === 'win32') {
  const utils = requireRuntime('node-pty/lib/utils');
  utils.loadNativeModule('conpty');
  utils.loadNativeModule('conpty_console_list');
}
const nativeArtifacts = {};
for (const file of Object.keys(require.cache)) if (file.endsWith('.node')) nativeArtifacts[inside(file)] = sha(file);
if (!Object.keys(nativeArtifacts).some(file => file.endsWith('better_sqlite3.node')) || !Object.keys(nativeArtifacts).some(file => /(?:conpty|pty)\.node$/.test(file))) throw Error('Required native bindings were not loaded');
const result = {schemaVersion: 1, platform: process.platform, arch: process.arch, node: process.version, abi: process.versions.modules, runtime: process.versions.electron ? 'electron' : 'node', ...(process.versions.electron ? {electron: process.versions.electron} : {}), nodeSHA256: sha(process.execPath), nativeArtifacts};
fs.writeFileSync(output, JSON.stringify(result, null, 2) + '\n', {flag: 'wx'});
console.log(JSON.stringify({platform: result.platform, arch: result.arch, runtime: result.runtime, node: result.node, abi: result.abi, electron: result.electron, nativeArtifacts: Object.keys(nativeArtifacts)}));
