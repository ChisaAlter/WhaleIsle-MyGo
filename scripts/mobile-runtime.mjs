// Build/staging utilities only. Never import the daemon or create a device identity.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';

export const shellFiles = [
  'src/main/dshd-remote.js', 'src/main/dshd-daemon-runner.mjs',
  'src/main/dshd-daemon-hooks.mjs', 'src/main/mobile-web-server.js',
  'src/main/mobile-web.js', 'src/main/dshd-git-tunnel.js', 'src/main/dshd-git-dispatch.js',
  ...['official-deepseek-env', 'lan', 'offer', 'dshd-host-tunnel', 'dshd-mux-sse'].map(n => `src/shared/${n}.js`),
];
const server = 'vendor/chisacode-remote/node_modules/@chisacode/server/dist/server/server/';
export const sourceAssets = ['vendor/chisacode-remote/packages/server/src/server/speech/providers/local/sherpa/assets/silero_vad.onnx'];
export const required = [
  ...shellFiles, 'node/node.exe', 'mobile/web/index.html',
  'mobile/web/chisacode/daemon-client.bundle.js',
  ...['exports', 'pairing-offer', 'relay-device-credential-store'].map(n => `${server}${n}.js`),
  'vendor/chisacode-remote/node_modules/@chisacode/protocol/dist/connection-offer.js',
  'vendor/chisacode-remote/node_modules/@chisacode/relay/dist/e2ee.js',
  'vendor/chisacode-remote/node_modules/better-sqlite3/build/Release/better_sqlite3.node',
  'vendor/chisacode-remote/LICENSE', 'vendor/chisacode-remote/NOTICE',
  'vendor/deepseek-harness/apps/cli/lib/bin.js',
  `${server}speech/providers/local/sherpa/assets/silero_vad.onnx`,
  ...['dsh-llm-deepseek','dsh-sandbox-local','dsh-sandbox-policy','dsh-subprocess-local','dsh-bash-sandbox','dsh-user-approval','dsh-fs-sandbox','dsh-fs-observation-policy','dsh-tool-fs','dsh-token-meter','dsh-compaction-basic','dsh-repeat-tool-reminder'].map(n => `vendor/deepseek-harness/node_modules/@deepseek-ai/${n}/lib/index.js`),
];
const sha = p => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
function assertFiles(root, files) {
  for (const f of files) if (!fs.statSync(path.join(root, f), {throwIfNoEntry:false})?.isFile())
    throw new Error(`Mobile runtime resource missing: ${f}`);
}
function copy(from, to, filter) {
  fs.mkdirSync(path.dirname(to), {recursive:true});
  fs.cpSync(from, to, {recursive:true, dereference:true, filter});
}
export function prepare(source, work) {
  assertFiles(source, [...sourceAssets, ...shellFiles, 'mobile/web/index.html', 'scripts/bundle-chisacode-mobile-client.mjs',
    'vendor/chisacode-remote/package.json', 'vendor/chisacode-remote/package-lock.json']);
  // A fresh work tree prevents stale dist/native outputs from qualifying as a build.
  fs.rmSync(work, {recursive:true, force:true});
  for (const f of shellFiles) copy(path.join(source,f), path.join(work,f));
  copy(path.join(source,'mobile/web'), path.join(work,'mobile/web'));
  copy(path.join(source,'scripts/bundle-chisacode-mobile-client.mjs'), path.join(work,'scripts/bundle-chisacode-mobile-client.mjs'));
  copy(path.join(source,'vendor/chisacode-remote'), path.join(work,'vendor/chisacode-remote'),
    p => !['node_modules','.git','.tmp','dist'].includes(path.basename(p)));
  const pkg = JSON.parse(fs.readFileSync(path.join(work,'vendor/chisacode-remote/package.json')));
  fs.writeFileSync(path.join(work,'source-provenance.json'), JSON.stringify({
    repository:pkg.repository, version:pkg.version,
    lockSHA256:sha(path.join(work,'vendor/chisacode-remote/package-lock.json')),
    managerSHA256:sha(path.join(work,'src/main/dshd-remote.js')),
  },null,2));
}
export function stage(work, harness, nodeDir, destination) {
  assertFiles(work, [...shellFiles,'mobile/web/index.html','mobile/web/chisacode/daemon-client.bundle.js','node-build.json']);
  assertFiles(harness, ['apps/cli/lib/bin.js']);
  assertFiles(nodeDir,['node.exe']);
  const build = JSON.parse(fs.readFileSync(path.join(work,'node-build.json')));
  if (build.platform !== 'win32' || build.nodeSHA256 !== sha(path.join(nodeDir,'node.exe')))
    throw new Error('Mobile runtime must use the exact Windows Node executable used to build its native modules');
  const temp = `${destination}.staging`;
  fs.rmSync(temp,{recursive:true,force:true});
  try {
    for (const f of shellFiles) copy(path.join(work,f),path.join(temp,f));
    copy(path.join(work,'mobile/web'),path.join(temp,'mobile/web'));
    // Preserve the lockfile workspace topology, dereferencing workspace junctions.
    // Deliberately no production pruning: it can silently remove native optional deps.
    copy(path.join(work,'vendor/chisacode-remote'),path.join(temp,'vendor/chisacode-remote'));
    copy(harness,path.join(temp,'vendor/deepseek-harness'));
    copy(nodeDir,path.join(temp,'node'));
    for (const f of ['source-provenance.json','node-build.json']) copy(path.join(work,f),path.join(temp,f));
    assertFiles(temp,required);
    const pty = path.join(temp,'vendor/chisacode-remote/node_modules/node-pty/prebuilds',`win32-${build.arch}`);
    if (!fs.existsSync(pty)) throw new Error(`Mobile runtime node-pty prebuild missing: ${pty}`);
    const files = {};
    function scan(dir) {
      for(const e of fs.readdirSync(dir,{withFileTypes:true})) {
        const p = path.join(dir,e.name);
        if(e.isDirectory()) scan(p); else files[path.relative(temp,p).split(path.sep).join('/')] = sha(p);
      }
    }
    scan(temp);
    fs.writeFileSync(path.join(temp,'staging-manifest.json'), JSON.stringify({format:1,files},null,2));
    verify(temp);
    fs.rmSync(destination,{recursive:true,force:true});
    fs.renameSync(temp,destination);
  } catch(e) { fs.rmSync(temp,{recursive:true,force:true}); throw e; }
}
export function verify(root) {
  assertFiles(root,[...required,'staging-manifest.json','node-build.json']);
  const manifest = JSON.parse(fs.readFileSync(path.join(root,'staging-manifest.json')));
  if(manifest.format !== 1 || !manifest.files) throw new Error('Invalid mobile staging manifest');
  for(const f of [...required,'node-build.json']) if(!manifest.files[f]) throw new Error(`Untracked mobile resource: ${f}`);
  for(const [f,hash] of Object.entries(manifest.files)) {
    if(path.isAbsolute(f) || f.split('/').includes('..')) throw new Error('Unsafe mobile staging manifest path');
    assertFiles(root,[f]);
    if(sha(path.join(root,f)) !== hash) throw new Error(`Mobile runtime changed after build: ${f}`);
  }
  const build = JSON.parse(fs.readFileSync(path.join(root,'node-build.json')));
  if(build.platform !== 'win32' || build.nodeSHA256 !== sha(path.join(root,'node/node.exe')))
    throw new Error('Mobile Node build provenance mismatch');
}
if(process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [mode,...args] = process.argv.slice(2);
  if(mode === 'prepare') prepare(...args);
  else if(mode === 'stage') stage(...args);
  else if(mode === 'verify') verify(...args);
  else throw new Error('Expected prepare, stage, or verify');
}
