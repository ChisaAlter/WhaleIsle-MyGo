// Build/staging utilities only. Never import the daemon or create a device identity.
import fs from 'node:fs';
import {prepare as prepareOfflineWindows} from './offline-windows-runtime.mjs';
import path from 'node:path';
import crypto from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import {spawnSync} from 'node:child_process';
import {inventoryGraph, verifyGraph, safeChild, inside} from './session-runtime-graph.mjs';

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
  ...['exports', 'pairing-offer', 'relay-device-credential-store', 'relay-pairing-link-store'].map(n => `${server}${n}.js`),
  'vendor/chisacode-remote/node_modules/@chisacode/protocol/dist/connection-offer.js',
  'vendor/chisacode-remote/node_modules/@chisacode/relay/dist/e2ee.js',
  'vendor/chisacode-remote/node_modules/better-sqlite3/build/Release/better_sqlite3.node',
  'vendor/chisacode-remote/LICENSE', 'vendor/chisacode-remote/NOTICE',
  'vendor/deepseek-harness/apps/cli/lib/bin.js',
  `${server}speech/providers/local/sherpa/assets/silero_vad.onnx`,
  ...['dsh-llm-deepseek','dsh-sandbox-local','dsh-sandbox-policy','dsh-subprocess-local','dsh-bash-sandbox','dsh-user-approval','dsh-fs-sandbox','dsh-fs-observation-policy','dsh-tool-fs','dsh-token-meter','dsh-compaction-basic','dsh-repeat-tool-reminder'].map(n => `vendor/deepseek-harness/node_modules/@deepseek-ai/${n}/lib/index.js`),
];
const assetPin = JSON.parse(fs.readFileSync(new URL('./mobile-runtime-assets.json', import.meta.url))).sileroVad;
const nodeRelative = platform => platform === 'win32' ? 'node.exe' : 'bin/node';
const platformRequired = platform => required.map(f => f === 'node/node.exe' ? `node/${nodeRelative(platform)}` : f);
function supportedPlatform(platform) { if (!['win32','linux','darwin'].includes(platform)) throw new Error(`Unsupported mobile build platform: ${platform}`); }
const sha = p => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
function assertFiles(root, files) {
  for (const f of files) if (!fs.statSync(path.join(root, f), {throwIfNoEntry:false})?.isFile())
    throw new Error(`Mobile runtime resource missing: ${f}`);
}
function copy(from, to, filter) {
  fs.mkdirSync(path.dirname(to), {recursive:true});
  fs.cpSync(from, to, {recursive:true, verbatimSymlinks:true, filter});
  // Keep dependency topology without expanding pnpm's graph into duplicate trees.
  // Relocate absolute npm workspace junctions; relative links stay relative.
  function relocate(source, target) {
    const st=fs.lstatSync(target);
    if(st.isSymbolicLink()) {
      const link=fs.readlinkSync(target);
      if(path.isAbsolute(link)) {
        const relative=path.relative(path.resolve(from),path.resolve(link));
        if(relative.startsWith('..'+path.sep) || relative==='..' || path.isAbsolute(relative)) throw new Error(`External dependency link cannot be staged: ${source}`);
        const resolved=path.resolve(to,relative);
        fs.unlinkSync(target);
        fs.symlinkSync(path.relative(path.dirname(target),resolved),target,process.platform==='win32' && fs.statSync(source).isDirectory()?'junction':undefined);
      }
    } else if(st.isDirectory()) for(const name of fs.readdirSync(target)) relocate(path.join(source,name),path.join(target,name));
  }
  relocate(from,to);
}
export function prepare(source, work, modelFile = '') {
  const model = modelFile || path.join(source, sourceAssets[0]);
  if (!fs.statSync(model, {throwIfNoEntry:false})?.isFile()) {
    throw new Error(`Mobile runtime resource missing: ${sourceAssets[0]}. Supply the verified local model using -SileroVadFile or WHALEISLE_SILERO_VAD_FILE.`);
  }
  if (fs.statSync(model).size !== assetPin.bytes || sha(model) !== assetPin.sha256) {
    throw new Error('Silero VAD model checksum mismatch; expected the pinned original asset. Source/work trees were not modified.');
  }
  assertFiles(source, [...shellFiles, 'mobile/web/index.html', 'scripts/bundle-chisacode-mobile-client.mjs',
    'vendor/chisacode-remote/package.json', 'vendor/chisacode-remote/package-lock.json']);
  // A fresh work tree prevents stale dist/native outputs from qualifying as a build.
  fs.rmSync(work, {recursive:true, force:true});
  for (const f of shellFiles) copy(path.join(source,f), path.join(work,f));
  copy(path.join(source,'mobile/web'), path.join(work,'mobile/web'));
  copy(path.join(source,'scripts/bundle-chisacode-mobile-client.mjs'), path.join(work,'scripts/bundle-chisacode-mobile-client.mjs'));
  copy(path.join(source,'vendor/chisacode-remote'), path.join(work,'vendor/chisacode-remote'),
    p => !['node_modules','.git','.tmp','dist'].includes(path.basename(p)));
  copy(model, path.join(work, sourceAssets[0]));
  const supplements = [];
  const qrPath='vendor/chisacode-remote/packages/server/src/share/QRCode.ts';
  if(!fs.existsSync(path.join(work,qrPath))) {
    const anchor=path.join(work,'vendor/chisacode-remote/packages/server/src/server/pairing-qr.ts');
    const original=fileURLToPath(new URL('./mobile-runtime-source/QRCode.ts',import.meta.url));
    if(sha(anchor)!=='dccaf9a9b2d992cfe40e7e816aa6f3c27bf093e9b7bf8aca9fa1630eaa03737d' || sha(original)!=='fc57907f2bbd8797c8ed9f3cb8b2f7d2d413f4bc30e2e56b4e7869206603fcf0') throw new Error('Missing QRCode source does not match the pinned supported snapshot');
    copy(original,path.join(work,qrPath));
    supplements.push({path:qrPath,repository:'https://github.com/ChisaAlter/WhaleIsle',commit:'90465b67ec16765631d54c195af19d9ff3c3a770',sha256:sha(original),reason:'Missing original shared source; importing pairing-qr.ts matches the fixed source byte-for-byte.'});
  }
  const pkg = JSON.parse(fs.readFileSync(path.join(work,'vendor/chisacode-remote/package.json')));
  fs.writeFileSync(path.join(work,'source-provenance.json'), JSON.stringify({
    repository:pkg.repository, version:pkg.version, sourceSupplements:supplements,
    lockSHA256:sha(path.join(work,'vendor/chisacode-remote/package-lock.json')),
    managerSHA256:sha(path.join(work,'src/main/dshd-remote.js')),
    assets: [{...assetPin, suppliedVia: modelFile ? 'explicit-local-file' : 'source-snapshot', stagedPath: sourceAssets[0]}],
  },null,2));
}
// Use the official JS API: esbuild/bin/esbuild can become a native ELF/Mach-O
// executable after install, so passing that entry to `node` is not portable.
export function bundle(work) {
  work=path.resolve(work);
  const nm=path.join(work,'vendor/chisacode-remote/node_modules');
  const require=createRequire(path.join(work,'vendor/chisacode-remote/package.json'));
  require('esbuild').buildSync({entryPoints:[path.join(work,'mobile/web/chisacode/entry.mjs')],
    outfile:path.join(work,'mobile/web/chisacode/daemon-client.bundle.js'),bundle:true,
    format:'esm',platform:'browser',target:'es2022',sourcemap:true,nodePaths:[nm]});
}
export function record(work) {
  work=path.resolve(work);
  const require=createRequire(path.join(work,'vendor/chisacode-remote/package.json'));
  const Database=require('better-sqlite3');
  const db=new Database(':memory:');
  try {db.prepare('select 1').get();} finally {db.close();}
  require('node-pty'); // Load the binding, never create a terminal/process.
  fs.writeFileSync(path.join(work,'node-build.json'),JSON.stringify({platform:process.platform,
    arch:process.arch,node:process.version,abi:process.versions.modules,nodeSHA256:sha(process.execPath)},null,2));
}
function composeHarnessScope(root) {
  const names=required.filter(f=>f.startsWith('vendor/deepseek-harness/node_modules/@deepseek-ai/')).map(f=>f.split('/')[4]);
  const packages=new Map();
  for(const category of fs.readdirSync(path.join(root,'packages'))) {
    const dir=path.join(root,'packages',category);
    if(!fs.statSync(dir).isDirectory()) continue;
    for(const name of fs.readdirSync(dir)) {
      const packageDir=path.join(dir,name), manifest=path.join(packageDir,'package.json');
      if(fs.statSync(manifest,{throwIfNoEntry:false})?.isFile()) packages.set(JSON.parse(fs.readFileSync(manifest)).name,packageDir);
    }
  }
  for(const name of names) {
    const link=path.join(root,'node_modules/@deepseek-ai',name);
    if(fs.existsSync(path.join(link,'lib/index.js'))) continue;
    const source=packages.get('@deepseek-ai/'+name);
    if(!source) throw new Error(`Built Harness package missing: ${name}`);
    assertFiles(source,['lib/index.js']);
    if(fs.lstatSync(link,{throwIfNoEntry:false})) throw new Error(`Incomplete existing Harness package link: ${name}`);
    fs.mkdirSync(path.dirname(link),{recursive:true});
    fs.symlinkSync(path.relative(path.dirname(link),source),link,process.platform==='win32'?'junction':undefined);
  }
}
export function stage(work, harness, nodeDir, destination) {
  assertFiles(work, [...shellFiles,'mobile/web/index.html','mobile/web/chisacode/daemon-client.bundle.js','node-build.json']);
  assertFiles(harness, ['apps/cli/lib/bin.js']);
  const build = JSON.parse(fs.readFileSync(path.join(work,'node-build.json')));
  supportedPlatform(build.platform);
  const node=nodeRelative(build.platform);
  assertFiles(nodeDir,[node]);
  if (build.nodeSHA256 !== sha(path.join(nodeDir,node)))
    throw new Error('Mobile runtime must use the exact same-platform Node executable used to build its native modules');
  destination=path.resolve(destination);
  const temp = `${destination}.staging`;
  fs.rmSync(temp,{recursive:true,force:true});
  try {
    for (const f of shellFiles) copy(path.join(work,f),path.join(temp,f));
    copy(path.join(work,'mobile/web'),path.join(temp,'mobile/web'));
    // Preserve the lockfile workspace topology, dereferencing workspace junctions.
    // Deliberately no production pruning: it can silently remove native optional deps.
    copy(path.join(work,'vendor/chisacode-remote'),path.join(temp,'vendor/chisacode-remote'));
    // CLI test profiles are not runtime inputs; one fixed-source fixture is an
    // intentionally external snapshot link and must not escape the package.
    const cliTests=path.resolve(harness,'apps/cli/tests');
    copy(harness,path.join(temp,'vendor/deepseek-harness'),p=>path.resolve(p)!==cliTests);
    composeHarnessScope(path.join(temp,'vendor/deepseek-harness'));
    copy(nodeDir,path.join(temp,'node'));
    for (const f of ['source-provenance.json','node-build.json']) copy(path.join(work,f),path.join(temp,f));
    assertFiles(temp,platformRequired(build.platform));
    const pty = path.join(temp,'vendor/chisacode-remote/node_modules/node-pty');
    const ptyName=build.platform==='win32'?'conpty.node':'pty.node';
    const nativePty=['build/Release','build/Debug',`prebuilds/${build.platform}-${build.arch}`].map(dir=>path.join(pty,dir,ptyName)).find(file=>fs.statSync(file,{throwIfNoEntry:false})?.isFile());
    if (!nativePty) throw new Error(`Mobile runtime node-pty native output missing for ${build.platform}-${build.arch}`);
    const files = {}, links = {};
    function scan(dir) {
      for(const e of fs.readdirSync(dir,{withFileTypes:true})) {
        const p = path.join(dir,e.name);
        const key=path.relative(temp,p).split(path.sep).join('/');
        if(e.isSymbolicLink()) links[key]=fs.readlinkSync(p);
        else if(e.isDirectory()) scan(p); else files[key] = sha(p);
      }
    }
    scan(temp);
    fs.writeFileSync(path.join(temp,'staging-manifest.json'), JSON.stringify({format:2,platform:build.platform,arch:build.arch,files,links},null,2));
    verify(temp);
    fs.rmSync(destination,{recursive:true,force:true});
    fs.renameSync(temp,destination);
    // Windows directory junctions store absolute targets. Rebase those after
    // the atomic directory move, then verify the final package again.
    for(const [name,link] of Object.entries(links)) {
      if(!path.isAbsolute(link)) continue;
      const target=path.join(destination,path.relative(temp,link));
      const item=path.join(destination,name);
      fs.unlinkSync(item);
      fs.symlinkSync(target,item,process.platform==='win32' && fs.statSync(target).isDirectory()?'junction':undefined);
      links[name]=fs.readlinkSync(item);
    }
    fs.writeFileSync(path.join(destination,'staging-manifest.json'),JSON.stringify({format:2,platform:build.platform,arch:build.arch,files,links},null,2));
    verify(destination);
  } catch(e) { fs.rmSync(temp,{recursive:true,force:true}); throw e; }
}
const installedRequired = [
  ...shellFiles, 'mobile/web/index.html', 'mobile/web/chisacode/daemon-client.bundle.js',
  ...['exports','pairing-offer','relay-device-credential-store','relay-pairing-link-store'].map(name => `${server}${name}.js`),
  'vendor/chisacode-remote/node_modules/@chisacode/protocol/dist/connection-offer.js',
  'vendor/chisacode-remote/node_modules/@chisacode/relay/dist/e2ee.js',
  'vendor/chisacode-remote/node_modules/better-sqlite3/build/Release/better_sqlite3.node',
  `${server}speech/providers/local/sherpa/assets/silero_vad.onnx`, 'LICENSE', 'NOTICE',
];
const electronResources = ['icudtl.dat','resources.pak','chrome_100_percent.pak','chrome_200_percent.pak','v8_context_snapshot.bin','snapshot_blob.bin'];
const peerArtifacts = ['apps/cli/lib/bin.js','packages/core/session/lib/index.js','packages/session/session-persistence/lib/index.js'];
function bundledNode(root) {
  return ['node.exe','node/node.exe','node/bin/node'].map(relative => path.join(root,relative)).find(file => fs.statSync(file,{throwIfNoEntry:false})?.isFile());
}
export function probeNative(root, output) {
  root=fs.realpathSync(root);
  const executable=bundledNode(root);
  if(!executable)throw Error('Mobile runtime resource missing: bundled Node');
  const env={...process.env,ELECTRON_RUN_AS_NODE:'1'};
  // Inherited stdio avoids named-pipe restrictions; windowsHide suppresses the
  // Electron-as-Node console as well as the normal Node console.
  const result=spawnSync(executable,[fileURLToPath(new URL('./mobile-runtime-probe.cjs',import.meta.url)),root,path.resolve(output)],{env,stdio:'inherit',windowsHide:true,timeout:30000});
  if(result.error||result.status!==0)throw Error(`Mobile native ABI probe failed: ${result.error?.message||result.status}`);
  const build=JSON.parse(fs.readFileSync(output,'utf8'));
  if(build.platform!==process.platform||build.arch!==process.arch||!['node','electron'].includes(build.runtime)||build.nodeSHA256!==sha(executable))throw Error('Mobile native probe identity mismatch');
  return build;
}
export function recordInstalled(root) {
  root=fs.realpathSync(root);
  assertFiles(root,installedRequired);
  if(fs.existsSync(path.join(root,'staging-manifest.json'))||fs.existsSync(path.join(root,'node-build.json')))throw Error('Record installed mobile runtime only in a fresh unrecorded copy');
  const buildPath=path.join(root,'node-build.json');
  try {
    const build=probeNative(root,buildPath);
    if(build.runtime==='electron')assertFiles(root,electronResources);
    const peer=path.join(path.dirname(root),'deepseek-harness'),harnessArtifacts={};
    for(const relative of peerArtifacts) {const filename=path.join(peer,relative);if(!inside(path.dirname(root),fs.realpathSync(filename)))throw Error('Mobile Host peer escapes vendor graph');harnessArtifacts[relative]=sha(filename);}
    const graph=inventoryGraph(root,['staging-manifest.json','debug.log']);
    const version=JSON.parse(fs.readFileSync(path.join(peer,'package.json'),'utf8')).version;
    const profile=version==='0.2.1-rc.4'?'installed-rc4':version==='0.2.1-rc.2'?'installed-rc2':null;
    if(!profile)throw Error('Unsupported installed mobile Host version');
    const manifest={format:3,profile,platform:build.platform,arch:build.arch,nodePath:path.relative(root,bundledNode(root)).split(path.sep).join('/'),files:graph.files,links:graph.links,harnessPeer:{path:'../deepseek-harness',artifacts:harnessArtifacts}};
    fs.writeFileSync(path.join(root,'staging-manifest.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx'});
    verify(root);
    return {profile:manifest.profile,runtime:build.runtime,node:build.node,abi:build.abi,files:Object.keys(graph.files).length,links:Object.keys(graph.links).length};
  } catch(error) {
    for(const filename of [buildPath,path.join(root,'staging-manifest.json')])if(fs.existsSync(filename))fs.unlinkSync(filename);
    throw error;
  }
}
export function verifyInstalled(root,manifest,build) {
  if(!['installed-rc2','installed-rc4'].includes(manifest.profile)||manifest.platform!==build.platform||manifest.arch!==build.arch||!['node','electron'].includes(build.runtime)||!/^v\d+\.\d+\.\d+/.test(build.node)||!/^\d+$/.test(String(build.abi)))throw Error('Unsupported installed mobile runtime identity');
  supportedPlatform(build.platform);
  const executable=safeChild(root,manifest.nodePath);
  if(!manifest.files[manifest.nodePath]||sha(executable)!==build.nodeSHA256)throw Error('Mobile Node build provenance mismatch');
  verifyGraph(root,{schemaVersion:1,files:manifest.files,links:manifest.links},{exclude:['staging-manifest.json','debug.log']});
  assertFiles(root,installedRequired);
  if(build.runtime==='electron') {if(!build.electron)throw Error('Electron mobile runtime is missing its runtime version');assertFiles(root,electronResources);}
  const native=Object.keys(build.nativeArtifacts||{});
  if(!native.some(name=>name.endsWith('better_sqlite3.node'))||!native.some(name=>/(?:conpty|pty)\.node$/.test(name)))throw Error('Mobile runtime has no actual SQLite/PTY ABI load proof');
  for(const [relative,expected] of Object.entries(build.nativeArtifacts))if(manifest.files[relative]!==expected||sha(safeChild(root,relative))!==expected)throw Error(`Mobile native binding proof changed: ${relative}`);
  if(manifest.harnessPeer?.path!=='../deepseek-harness')throw Error('Mobile runtime is missing its exact Host peer');
  const vendor=fs.realpathSync(path.dirname(root)),peer=path.join(vendor,'deepseek-harness');
  if(manifest.profile==='installed-rc4') {
    if(JSON.parse(fs.readFileSync(path.join(peer,'package.json'),'utf8')).version!=='0.2.1-rc.4')throw Error('Mobile rc.4 Host version mismatch');
    assertFiles(root,['mobile/web/whale-ui/viewer.html', ...['standalone-runtime','icons','mermaid','three','echarts-core','echarts-full'].map(name=>`mobile/web/whale-ui/assets/${name}.js`)]);
  }
  for(const relative of peerArtifacts) {
    const filename=safeChild(peer,relative);
    if(!inside(vendor,fs.realpathSync(filename))||sha(filename)!==manifest.harnessPeer.artifacts?.[relative])throw Error(`Mobile Host peer changed: ${relative}`);
  }
  return build;
}
export function verify(root) {
  assertFiles(root,['node-build.json','staging-manifest.json']);
  root=fs.realpathSync(root);
  const build = JSON.parse(fs.readFileSync(path.join(root,'node-build.json')));
  const manifest = JSON.parse(fs.readFileSync(path.join(root,'staging-manifest.json')));
  if(manifest.format===3)return verifyInstalled(root,manifest,build);
  supportedPlatform(build.platform);
  const requiredForBuild=platformRequired(build.platform);
  assertFiles(root,requiredForBuild);
  if(![1,2].includes(manifest.format) || !manifest.files) throw new Error('Invalid mobile staging manifest');
  for(const f of [...requiredForBuild,'node-build.json']) if(!manifest.files[f] && !Object.keys(manifest.links || {}).some(link => f.startsWith(link+'/'))) throw new Error(`Untracked mobile resource: ${f}`);
  for(const [f,hash] of Object.entries(manifest.files)) {
    if(path.isAbsolute(f) || f.split('/').includes('..')) throw new Error('Unsafe mobile staging manifest path');
    assertFiles(root,[f]);
    if(sha(path.join(root,f)) !== hash) throw new Error(`Mobile runtime changed after build: ${f}`);
  }
  if(manifest.format===2 && (manifest.platform!==build.platform || manifest.arch!==build.arch)) throw new Error('Mobile staging platform mismatch');
  for(const [f,link] of Object.entries(manifest.links || {})) {
    if(path.isAbsolute(f) || f.split('/').includes('..')) throw new Error('Unsafe mobile dependency path');
    const p=path.join(root,f);
    if(!fs.lstatSync(p).isSymbolicLink() || fs.readlinkSync(p)!==link) throw new Error(`Mobile dependency link changed: ${f}`);
    const relative=path.relative(fs.realpathSync(root),fs.realpathSync(p));
    if(relative==='..' || relative.startsWith('..'+path.sep) || path.isAbsolute(relative)) throw new Error(`Mobile dependency escapes staged runtime: ${f}`);
  }
  if(build.nodeSHA256 !== sha(path.join(root,'node',nodeRelative(build.platform)))) throw new Error('Mobile Node build provenance mismatch');
}
if(process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [mode,...args] = process.argv.slice(2);
  if(mode === 'prepare-offline-windows') console.log(JSON.stringify(prepareOfflineWindows(...args),null,2));
  else if(mode === 'prepare') prepare(...args);
  else if(mode === 'bundle') bundle(...args);
  else if(mode === 'record') record(...args);
  else if(mode === 'stage') stage(...args);
  else if(mode === 'verify') console.log(JSON.stringify(verify(...args) || {verified:true}));
  else if(mode === 'record-installed') console.log(JSON.stringify(recordInstalled(...args),null,2));
  else if(mode === 'probe') console.log(JSON.stringify(probeNative(...args),null,2));
  else throw new Error('Expected prepare-offline-windows, prepare, bundle, record, stage, verify, record-installed, or probe');
}
