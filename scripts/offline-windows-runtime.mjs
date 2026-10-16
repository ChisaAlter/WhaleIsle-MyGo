// Offline cross-target preparation only. Never executes a target binary.
import fs from 'node:fs';
import {normalizeWindowsLayout} from './offline-windows-layout.mjs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const within=(root,p)=>p===root||p.startsWith(root+path.sep);
function clone(from,to,filter=()=>true) {
 if(!filter(from))return;
 const s=fs.lstatSync(from);fs.mkdirSync(path.dirname(to),{recursive:true});
 if(s.isSymbolicLink()) {const target=fs.readlinkSync(from);if(path.isAbsolute(target))throw Error(`Absolute source link: ${from}`);fs.symlinkSync(target,to);}
 else if(s.isDirectory()){fs.mkdirSync(to,{recursive:true});for(const n of fs.readdirSync(from))clone(path.join(from,n),path.join(to,n),filter);}
 else if(/\.(json|ya?ml|toml|ini|conf|env)$/i.test(from)||path.basename(from).startsWith('.env'))fs.copyFileSync(from,to);
 else fs.linkSync(from,to); // immutable code/assets; replacements always unlink first
}
function walk(root,fn){for(const e of fs.readdirSync(root,{withFileTypes:true})){const p=path.join(root,e.name);if(e.isDirectory())walk(p,fn);else fn(p,e);}}
function peX64(p){const fd=fs.openSync(p,'r');try{const b=Buffer.alloc(64);if(fs.readSync(fd,b,0,64,0)<64||b.toString('ascii',0,2)!=='MZ')return false;const o=b.readUInt32LE(60),h=Buffer.alloc(6);fs.readSync(fd,h,0,6,o);return h.toString('ascii',0,4)==='PE\0\0'&&h.readUInt16LE(4)===0x8664;}finally{fs.closeSync(fd);}}
function nonTarget(p){const fd=fs.openSync(p,'r'),b=Buffer.alloc(4);try{fs.readSync(fd,b,0,4,0);}finally{fs.closeSync(fd);}return b.equals(Buffer.from([127,69,76,70]))||[0xfeedface,0xfeedfacf,0xcafebabe,0xcefaedfe,0xcffaedfe].includes(b.readUInt32BE(0));}
export function prepare(configFile) {
 const c=read(configFile);
 if(c.targetPlatform!=='win32'||c.targetArch!=='x64')throw Error('Only explicit win32/x64 supported');
 const hostManifest=read(path.join(c.inputs.host,'current-host-build.json'));
 if(!['90465b67ec16765631d54c195af19d9ff3c3a770','e9f584be6d3b491ce777bb6823f30bc2bbc6f195'].includes(hostManifest.sourceCommit)||!hostManifest.safetyPatch)throw Error('Expected guarded fixed Host build');
 for(const [file,hash] of Object.entries(hostManifest.artifacts||{}))if(sha(path.join(c.inputs.host,file))!==hash)throw Error(`Changed Host input: ${file}`);
 const mobileBuild=read(path.join(c.inputs.mobile,'node-build.json'));
 if(mobileBuild.node!=='v24.19.0'||String(mobileBuild.abi)!=='137')throw Error('This offline recipe requires the verified Node 24.19.0 / ABI137 source build');
 const target=path.resolve(c.destination);if(fs.existsSync(target))throw Error('Use a fresh candidate destination');
 const assets=read(c.assetsManifest);if(assets.some(a=>a.status!=='available'))throw Error('Missing target assets; inspect assets manifest');
 for(const a of assets){const archive=path.join(path.dirname(a.path),'package.tgz');if(sha(archive)!==a.sha256)throw Error(`Target asset archive changed: ${a.name}`);}
 const assetsByName=new Map(assets.map(a=>[a.name+'@'+a.version,a]));
 const result={schemaVersion:1,status:'preparing',targetPlatform:c.targetPlatform,targetArch:c.targetArch,runtime_verified:false,assemblyPlatform:process.platform,sourceBuilds:{host:hostManifest,mobile:mobileBuild},inputs:{},assets,removed:[],links:{},files:{}};
 for(const [name,source] of Object.entries(c.inputs)){
  const root=path.resolve(source);if(within(root,target)||within(target,root))throw Error('Candidate must be separate from source inputs');result.inputs[name]={path:root,provenance:{}};
  for(const f of ['current-host-build.json','visual-readiness-patch.json','node-build.json','source-provenance.json','deepseek-harness/pnpm-lock.yaml','vendor/chisacode-remote/package-lock.json','vendor/deepseek-harness/pnpm-lock.yaml'])if(fs.existsSync(path.join(root,f)))result.inputs[name].provenance[f]=sha(path.join(root,f));
  clone(root,path.join(target,name),p=>!p.endsWith('/apps/cli/tests'));
 }
 // Keep old build provenance as evidence, never as an accepted Windows manifest.
 for(const [name,f] of [['host','current-host-build.json'],['mobile','node-build.json'],['mobile','staging-manifest.json']]){
  const p=path.join(target,name,f);if(fs.existsSync(p))fs.renameSync(p,p+'.source-linux.json');
 }
 const node=path.join(target,'node');clone(path.resolve(c.nodeDir),node);
 if(!peX64(path.join(node,'node.exe'))||sha(path.join(node,'node.exe'))!==c.nodeSHA256)throw Error('Windows Node identity mismatch');
 // Mobile's node directory is replaced wholesale; never reuse Linux executable.
 fs.rmSync(path.join(target,'mobile/node'),{recursive:true,force:true});clone(node,path.join(target,'mobile/node'));
 const manifests=[];walk(target,(p,e)=>{if(e.isFile()&&path.basename(p)==='package.json')manifests.push(p);});
 for(const p of manifests){if(!fs.existsSync(p))continue;let m;try{m=read(p);}catch{continue;}
  const incompatible=(m.os?.length&&!m.os.includes('win32')&&!m.os.every(x=>x.startsWith('!')))||(m.cpu?.length&&!m.cpu.includes('x64')&&!m.cpu.every(x=>x.startsWith('!')));
  if(incompatible){result.removed.push(path.relative(target,path.dirname(p)));fs.rmSync(path.dirname(p),{recursive:true,force:true});}
 }
 for(const p of manifests){if(!fs.existsSync(p))continue;let m;try{m=read(p);}catch{continue;}
  for(const [name,spec] of Object.entries(m.optionalDependencies||{})){
   if(!((name.includes('win32')||name.includes('-win-'))&&name.includes('x64')))continue;
   let version=spec.startsWith('npm:')?spec.slice(spec.lastIndexOf('@')+1):spec.replace(/^[~^]/,'');
   let a=assetsByName.get(name+'@'+version);if(!a && /^[~^]\d+\.\d+\.\d+$/.test(spec)){const base=version.split('.').map(Number);const choices=assets.filter(x=>x.name===name&&/^\d+\.\d+\.\d+$/.test(x.version)&&x.version.split('.')[0]===String(base[0])&&(spec[0]==='^'||x.version.split('.')[1]===String(base[1]))&&x.version.localeCompare(version,undefined,{numeric:true})>=0);if(choices.length===1){a=choices[0];version=a.version;}}if(!a)throw Error(`Missing exact target dependency ${name}@${version}`);
   const cache=path.join(target,'target-packages',name.replaceAll('/','__')+'@'+version);
   if(!fs.existsSync(cache))clone(a.path,cache);
   const dependency=path.join(path.dirname(p),'node_modules',name);
   fs.rmSync(dependency,{recursive:true,force:true});fs.mkdirSync(path.dirname(dependency),{recursive:true});fs.symlinkSync(path.relative(path.dirname(dependency),cache),dependency);
  }
 }
 // Exact Node-ABI binding, replacing only mobile's matching package version.
 const sqlite=path.join(target,'mobile/vendor/chisacode-remote/node_modules/better-sqlite3');
 if(read(path.join(sqlite,'package.json')).version!=='12.11.1')throw Error('SQLite version mismatch');
 const binding=path.join(sqlite,'build/Release/better_sqlite3.node');fs.rmSync(binding,{force:true});clone(c.sqliteBinding,binding);
 if(!peX64(binding)||sha(binding)!==c.sqliteSHA256)throw Error('SQLite target binding mismatch');
 // Remove all ELF/Mach-O binaries, all non-x64 PE/addons and non-target prebuilds.
 walk(target,(p,e)=>{if(!e.isFile())return;if(nonTarget(p)||((p.endsWith('.node')||p.endsWith('.exe')||p.endsWith('.dll'))&&!peX64(p))){result.removed.push(path.relative(target,p));fs.unlinkSync(p);}});
 // Remove dangling platform-optional links. Existing external links are rejected.
 walk(target,(p,e)=>{if(!e.isSymbolicLink())return;const resolved=path.resolve(path.dirname(p),fs.readlinkSync(p));if(!within(target,resolved))throw Error(`External link: ${p}`);if(!fs.existsSync(p)){result.removed.push(path.relative(target,p));fs.unlinkSync(p);}});
 result.layout=normalizeWindowsLayout(target);
 const required=['node/node.exe','host/deepseek-harness/apps/cli/lib/bin.js','host/dsh-project/lib/index.js','mobile/node/node.exe','mobile/vendor/chisacode-remote/node_modules/better-sqlite3/build/Release/better_sqlite3.node','mobile/vendor/chisacode-remote/node_modules/node-pty/prebuilds/win32-x64/conpty.node','mobile/vendor/deepseek-harness/apps/cli/lib/bin.js'];
 for(const f of required)if(!fs.statSync(path.join(target,f),{throwIfNoEntry:false})?.isFile())throw Error(`Missing runtime entry ${f}`);
 walk(target,(p,e)=>{const key=path.relative(target,p).split(path.sep).join('/');if(e.isSymbolicLink()){if(!within(target,fs.realpathSync(p)))throw Error(`Link escapes candidate: ${p}`);result.links[key]=fs.readlinkSync(p);}else {if(p.endsWith('.node')&&!peX64(p))throw Error(`Non-target addon: ${p}`);result.files[key]=sha(p);}});
 result.status='prepared';result.staticEntryChecks=true;result.nodeSHA256=c.nodeSHA256;
 fs.writeFileSync(path.join(target,'offline-preparation.json'),JSON.stringify(result,null,2));return {destination:target,status:result.status,runtime_verified:false,files:Object.keys(result.files).length,links:Object.keys(result.links).length};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){try{console.log(JSON.stringify(prepare(process.argv[2]),null,2));}catch(e){console.error(e.stack);process.exitCode=1;}}
