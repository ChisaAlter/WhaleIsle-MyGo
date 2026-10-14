// Explicit Windows-only native verification. Never starts desktop/Host/mobile.
import fs from 'node:fs';import path from 'node:path';import os from 'node:os';import crypto from 'node:crypto';import {spawnSync} from 'node:child_process';import {fileURLToPath,pathToFileURL} from 'node:url';
import {restore} from './restore-offline-links.mjs';
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
export function assertRuntime(platform,arch,node,abi){if(platform!=='win32'||arch!=='x64'||node!=='v24.19.0'||abi!=='137')throw Error('Activation requires actual Windows x64 Node24.19.0 ABI137');}
export async function activate(root){assertRuntime(process.platform,process.arch,process.version,process.versions.modules);root=fs.realpathSync(root);const node=path.join(root,'runtime/node/node.exe');if(fs.realpathSync(process.execPath).toLowerCase()!==fs.realpathSync(node).toLowerCase()||sha(node)!=='3602f2bb1a10f2cbab4c36886218a33c1ab3db87290e73b033c46c77147d0237')throw Error('Run with the exact bundled official Node');
 const vendor=path.join(root,'runtime/vendor'),mobile=path.join(vendor,'dshd-remote'),statusFile=path.join(root,'candidate-status.json'),status=JSON.parse(fs.readFileSync(statusFile));
 const records=[path.join(vendor,'current-host-build.json'),path.join(mobile,'node-build.json'),path.join(mobile,'staging-manifest.json'),path.join(root,'.runtime-staged')];if(status.runtime_verified!==false||records.some(p=>fs.existsSync(p)))throw Error('Expected fresh prepared candidate without activation records');
 const created=[];let temp;
 try{restore(root);temp=fs.mkdtempSync(path.join(root,'.native-verification-'));for(const d of ['home','tmp','config','data','cache'])fs.mkdirSync(path.join(temp,d));
 const env={SystemRoot:process.env.SystemRoot||'C:\\Windows',WINDIR:process.env.WINDIR||'C:\\Windows',PATH:path.dirname(node)+';'+path.join(process.env.SystemRoot||'C:\\Windows','System32'),HOME:path.join(temp,'home'),USERPROFILE:path.join(temp,'home'),APPDATA:path.join(temp,'config'),LOCALAPPDATA:path.join(temp,'data'),DSH_HOME:path.join(temp,'home'),DSHD_HOME:path.join(temp,'home'),XDG_CONFIG_HOME:path.join(temp,'config'),XDG_DATA_HOME:path.join(temp,'data'),XDG_CACHE_HOME:path.join(temp,'cache'),TEMP:path.join(temp,'tmp'),TMP:path.join(temp,'tmp'),DSH_TELEMETRY_DISABLED:'1',DO_NOT_TRACK:'1'};
 const probe=spawnSync(node,[fileURLToPath(new URL('./verify-offline-native.mjs',import.meta.url)),root],{cwd:temp,env,encoding:'utf8',timeout:120000,windowsHide:true,maxBuffer:4*1024*1024});
 if(probe.status!==0||probe.error)throw Error(`Native verification failed: ${probe.error?.message||probe.stderr||probe.stdout}`);
 const actual=JSON.parse(probe.stdout.trim().split(/\r?\n/).at(-1));assertRuntime(actual.platform,actual.arch,actual.node,actual.abi);
 const evidence={scope:'windows-native-load-only',...actual,sourceBuilds:status.sourceBuilds,nodeSHA256:sha(node),hostStartupVerified:false,desktopUIVerified:false};
 const host={...status.sourceBuilds.host,platform:actual.platform,arch:actual.arch,node:actual.node,sourceBuild:status.sourceBuilds.host,windowsVerification:evidence};
 const write=(p,value)=>{fs.writeFileSync(p,JSON.stringify(value,null,2),{flag:'wx'});created.push(p);};write(records[0],host);
 const current=await import('./current-host.mjs');current.verify(vendor); // original source/guard/artifact/platform gate
 write(records[1],{platform:actual.platform,arch:actual.arch,node:actual.node,abi:actual.abi,nodeSHA256:sha(path.join(mobile,'node/node.exe')),sourceBuild:status.sourceBuilds.mobile,windowsVerification:evidence});
 if(sha(path.join(mobile,'node/node.exe'))!==sha(node))throw Error('Mobile Node differs from verified bundled Node');
 const files={},links={};function scan(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){const p=path.join(dir,e.name),name=path.relative(mobile,p).split(path.sep).join('/');if(e.isSymbolicLink())links[name]=fs.readlinkSync(p);else if(e.isDirectory())scan(p);else files[name]=sha(p);}}scan(mobile);
 write(records[2],{format:2,platform:actual.platform,arch:actual.arch,files,links});const mobileTools=await import('./mobile-runtime.mjs');mobileTools.verify(mobile); // original required assets/hashes/link/Node gate
 fs.writeFileSync(records[3],'1',{flag:'wx'});created.push(records[3]);
 fs.writeFileSync(path.join(root,'windows-native-verification.json'),JSON.stringify({...evidence,runtime_verified:true},null,2));
 fs.writeFileSync(statusFile,JSON.stringify({...status,status:'windows-native-verified-host-ui-unverified',runtime_verified:true,verification:evidence},null,2));return {runtime_verified:true,scope:evidence.scope,hostStartupVerified:false,desktopUIVerified:false};
 }catch(error){for(const p of created.reverse())fs.rmSync(p,{force:true});fs.writeFileSync(path.join(root,'windows-native-verification.json'),JSON.stringify({runtime_verified:false,status:'prepared-verification-failed',reason:error.message},null,2));throw error;}finally{if(temp)fs.rmSync(temp,{recursive:true,force:true});}
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){try{console.log(JSON.stringify(await activate(path.resolve(process.argv[2]||'.')),null,2));}catch(error){console.error(error.message);process.exitCode=1;}}
