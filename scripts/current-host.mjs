// Build/staging only. Never starts Host, imports a daemon, or touches user profiles.
import fs from 'node:fs';
import {prepare as prepareOfflineWindows} from './offline-windows-runtime.mjs';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {applyReadinessGuard, inspectReadinessGuard} from './current-host-readiness.mjs';
export const commit = '90465b67ec16765631d54c195af19d9ff3c3a770';
// New pins are opt-in: existing 904 builds and callers retain their selection.
export const supportedCommits = Object.freeze([commit, 'e9f584be6d3b491ce777bb6823f30bc2bbc6f195']);
const manifestName = 'current-host-build.json';
export const artifacts = [
 'deepseek-harness/apps/cli/lib/bin.js', 'deepseek-harness/apps/web/dist/index.html',
 ...['experimental/agent-team','experimental/agent-team-profile','experimental/tool-agent-team',
 'experimental/client-ui-agent-team','client/ui-visual-replies','api/session-controller'].map(p => `deepseek-harness/packages/${p}/lib/index.js`),
 'deepseek-harness/packages/client/ui-visual-replies/lib/client.js',
 'dsh-project/lib/index.js', 'dsh-project/lib/transport.js', 'dsh-project/client/client.js',
];
const digest = p => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const read = p => JSON.parse(fs.readFileSync(p, 'utf8'));
function git(source, args) {
 const r = spawnSync('git', ['-C', source, ...args], {encoding:'utf8'});
 if(r.status !== 0) throw new Error(`Cannot verify fixed source: ${r.stderr}`);
 return r.stdout.trim();
}
function absent(target) {
 if(fs.existsSync(target)) throw new Error(`Destination already exists; use a fresh directory: ${target}`);
}
function copy(source,target) {
 fs.cpSync(source,target,{recursive:true,filter:p => !['.git','node_modules'].includes(path.basename(p))});
 // Source references can be read-only; only the isolated copy becomes writable.
 function writable(p) { const s=fs.lstatSync(p); if(s.isSymbolicLink()) return; fs.chmodSync(p,s.mode|0o200); if(s.isDirectory()) for(const n of fs.readdirSync(p)) writable(path.join(p,n)); }
 writable(target);
}
export function prepare(source, work, sourceCommit = commit) {
 if(!supportedCommits.includes(sourceCommit)) throw new Error(`Unsupported official source commit ${sourceCommit}`);
 source=fs.realpathSync(source); work=path.resolve(work);
 if(work === source || work.startsWith(source+path.sep)) throw new Error('Build directory must be outside the source reference');
 absent(work);
 if(git(source,['rev-parse','HEAD']) !== sourceCommit) throw new Error(`Expected official source commit ${sourceCommit}`);
 if(git(source,['status','--porcelain','--untracked-files=no'])) throw new Error('Fixed official source has tracked changes; preserve custom source via the default legacy build');
 for(const f of ['vendor/deepseek-harness/pnpm-lock.yaml','vendor/deepseek-harness/pnpm-workspace.yaml','vendor/deepseek-harness/apps/desktop/package.json','vendor/dsh-project/package.json'])
  if(!fs.existsSync(path.join(source,f))) throw new Error(`Incomplete fixed source: ${f}`);
 const pkg=read(path.join(source,'vendor/deepseek-harness/package.json'));
 if(pkg.packageManager !== 'pnpm@11.8.0') throw new Error('Unexpected package manager in pinned source');
 fs.mkdirSync(work,{recursive:true});
 for(const name of ['deepseek-harness','dsh-project']) copy(path.join(source,'vendor',name),path.join(work,name));
 // Preserve original desktop composition/bridge code without importing it here.
 if(fs.existsSync(path.join(source,'src'))) copy(path.join(source,'src'),path.join(work,'desktop-source/src'));
 applyReadinessGuard(work);
 fs.writeFileSync(path.join(work,'source-selection.json'),JSON.stringify({sourceCommit,sourcePath:source,packageManager:pkg.packageManager,lockSHA256:digest(path.join(work,'deepseek-harness/pnpm-lock.yaml'))},null,2));
 return {work,sourceCommit,packageManager:pkg.packageManager};
}
export function finalize(work) {
 const selection=read(path.join(work,'source-selection.json'));
 if(!supportedCommits.includes(selection.sourceCommit) || selection.lockSHA256 !== digest(path.join(work,'deepseek-harness/pnpm-lock.yaml'))) throw new Error('Source/lock selection changed during build');
 const safetyPatch=inspectReadinessGuard(work);
 const hashes={};
 for(const f of artifacts) hashes[f]=digest(path.join(work,f));
 const manifest={schemaVersion:1,sourceCommit:selection.sourceCommit,sourcePath:selection.sourcePath,lockSHA256:selection.lockSHA256,packageManager:selection.packageManager,buildProfile:'official',safetyPatch,platform:process.platform,arch:process.arch,node:process.version,artifacts:hashes,
 // A successful build is not a working desktop bridge or an authenticated Host.
 capabilities:{project:false,team:false,visualReplies:false}};
 fs.writeFileSync(path.join(work,manifestName),JSON.stringify(manifest,null,2));
 return manifest;
}
export function verify(root) {
 const m=read(path.join(root,manifestName));
 if(m.schemaVersion!==1 || !supportedCommits.includes(m.sourceCommit) || m.buildProfile!=='official') throw new Error('Unsupported current Host build provenance');
 const safetyPatch=inspectReadinessGuard(root);
 if(JSON.stringify(m.safetyPatch)!==JSON.stringify(safetyPatch)) throw new Error('Current Host build is missing the matching visual readiness safety patch');
 if(m.platform!==process.platform || m.arch!==process.arch) throw new Error('Current Host build platform/architecture does not match this runtime');
 for(const f of artifacts) if(!m.artifacts?.[f] || digest(path.join(root,f))!==m.artifacts[f]) throw new Error(`Current Host artifact missing or changed: ${f}`);
 return m;
}
export function stage(work,legacyVendor,target) {
 const manifest=verify(work); absent(target);
 if(path.resolve(target)===path.resolve(legacyVendor) || path.resolve(target).startsWith(path.resolve(legacyVendor)+path.sep)) throw new Error('Current staging must not overwrite or nest in legacy runtime');
 for(const n of ['dsh-task-control','dsh-platform-session','dsh-usage-panel','dsh-im'])
  if(!fs.existsSync(path.join(legacyVendor,n,'package.json'))) throw new Error(`Build the legacy desktop runtime first: ${n}`);
 fs.mkdirSync(target,{recursive:true});
 // Keep customized desktop/mobile packages. Never replace their source checkout.
 for(const n of fs.readdirSync(legacyVendor)) if(!['deepseek-harness','dsh-project',manifestName,'desktop-source','visual-readiness-patch.json'].includes(n)) fs.cpSync(path.join(legacyVendor,n),path.join(target,n),{recursive:true,verbatimSymlinks:true});
 for(const n of ['deepseek-harness','dsh-project','desktop-source',manifestName,'visual-readiness-patch.json']) fs.cpSync(path.join(work,n),path.join(target,n),{recursive:true,verbatimSymlinks:true});
 verify(target);
 return {vendorDir:target,sourceCommit:manifest.sourceCommit,requiresBridgeVerification:true};
}
if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 try {const [command,...args]=process.argv.slice(2);const fn={prepare,finalize,verify,stage,'prepare-offline-windows':prepareOfflineWindows}[command];if(!fn)throw new Error('Usage: current-host.mjs prepare|finalize|verify|stage|prepare-offline-windows ...');console.log(JSON.stringify(fn(...args),null,2));}
 catch(e){console.error(e.message);process.exitCode=1;}
}
