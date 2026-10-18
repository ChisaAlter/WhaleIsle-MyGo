// Package an explicitly unverified portable candidate. No target execution.
import fs from 'node:fs';import path from 'node:path';import crypto from 'node:crypto';import {fileURLToPath} from 'node:url';
import {normalizeWindowsLayout} from './offline-windows-layout.mjs';
import {planLinks} from './restore-offline-links.mjs';
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const remap=f=>f.startsWith('host/')?'runtime/vendor/'+f.slice(5):f.startsWith('mobile/')?'runtime/vendor/dshd-remote/'+f.slice(7):f.startsWith('node/')?'runtime/node/'+f.slice(5):f.startsWith('target-packages/')?'runtime/'+f:f;
function copyFile(source,target){fs.mkdirSync(path.dirname(target),{recursive:true});if(/\.(json|ya?ml|toml|ini|conf|env)$/i.test(source))fs.copyFileSync(source,target);else fs.linkSync(source,target);}
function copyTree(source,target){const st=fs.lstatSync(source);if(st.isSymbolicLink())throw Error('Unexpected target package link');if(st.isDirectory()){fs.mkdirSync(target,{recursive:true});for(const f of fs.readdirSync(source))copyTree(path.join(source,f),path.join(target,f));}else copyFile(source,target);}
export function packageCandidate(config){const c=JSON.parse(fs.readFileSync(config)),source=path.resolve(c.source),root=path.resolve(c.destination);if(fs.existsSync(root)||root.startsWith(source+path.sep)||source.startsWith(root+path.sep))throw Error('Fresh separate destination required');const m=JSON.parse(fs.readFileSync(path.join(source,'offline-preparation.json')));if(m.status!=='prepared'||m.runtime_verified!==false)throw Error('Expected unverified prepared graph');
 if(sha(c.exe)!==c.exeSHA256||sha(path.join(c.terminalDir,'ghostty-vt.dll'))!==c.terminalSHA256)throw Error('EXE/terminal input changed');
 // Optional explicit, hash-pinned companion; never discover or execute a helper from a workspace.
 if(c.previewHelper&&(!/^[0-9a-f]{64}$/.test(c.previewHelperSHA256||'')||sha(c.previewHelper)!==c.previewHelperSHA256))throw Error('Preview helper input changed or lacks SHA256');
 if(!c.previewHelper&&c.previewHelperSHA256)throw Error('Preview helper path missing');
 fs.mkdirSync(root,{recursive:true});
 for(const f of Object.keys(m.files)){if(path.isAbsolute(f)||f.includes('\\')||f.split('/').some(x=>x==='..'||x==='.'||x.includes(':')))throw Error('Unsafe source manifest path');if(f.includes('/.bin/'))continue;copyFile(path.join(source,f),path.join(root,remap(f)));}
 const links=[];for(const f of Object.keys(m.links)){if(f.includes('/.bin/'))continue;const target=fs.realpathSync(path.join(source,f)),relative=path.relative(source,target);if(relative==='..'||relative.startsWith('..'+path.sep)||path.isAbsolute(relative))throw Error('Source link escapes graph');let mapped=remap(relative.split(path.sep).join('/'));
 if(f.startsWith('mobile/')&&mapped.startsWith('runtime/target-packages/')){const suffix=mapped.slice('runtime/target-packages/'.length),name=suffix.split('/')[0];const packageTarget=path.join(root,'runtime/vendor/dshd-remote/.target-packages',name);if(!fs.existsSync(packageTarget))copyTree(path.join(root,'runtime/target-packages',name),packageTarget);mapped='runtime/vendor/dshd-remote/.target-packages/'+suffix;}
 links.push({path:remap(f),target:mapped,kind:fs.statSync(target).isDirectory()?'directory':'file'});}
 if(c.previewHelper)copyFile(c.previewHelper,path.join(root,'whaleisle-preview.exe'));
 copyFile(c.exe,path.join(root,'whaleisle.exe'));for(const f of ['ghostty-vt.dll','ghostty-vt.dll.LICENSE','terminal-runtime.json'])copyFile(path.join(c.terminalDir,f),path.join(root,f));
 if(c.resources){fs.cpSync(c.resources,path.join(root,'resources'),{recursive:true,dereference:false});}
 copyFile(fileURLToPath(new URL('./restore-offline-links.mjs',import.meta.url)),path.join(root,'restore-offline-links.mjs'));
 const toolsDir=path.join(root,'candidate-tools');fs.mkdirSync(toolsDir,{recursive:true});
 for(const name of ['activate-offline-candidate.mjs','verify-offline-native.mjs','restore-offline-links.mjs','current-host.mjs','current-host-readiness.mjs','mobile-runtime.mjs','mobile-runtime-assets.json','offline-windows-runtime.mjs','offline-windows-layout.mjs'])fs.copyFileSync(fileURLToPath(new URL('./'+name,import.meta.url)),path.join(toolsDir,name));
 fs.writeFileSync(path.join(root,'VERIFY-AND-ACTIVATE.cmd'),'@echo off\r\n"%~dp0runtime\\node\\node.exe" "%~dp0candidate-tools\\activate-offline-candidate.mjs" "%~dp0."\r\nexit /b %errorlevel%\r\n');
 const linkManifest={format:1,targetPlatform:'win32',targetArch:'x64',runtime_verified:false,links};
 fs.writeFileSync(path.join(root,'candidate-links.json'),JSON.stringify(linkManifest,null,2));
 const layout=normalizeWindowsLayout(root,undefined,'runtime/node/node.exe');
 planLinks(root,linkManifest); // one static path/target validation; no junction creation
 const result={status:'portable-candidate-links-not-restored',runtime_verified:false,targetPlatform:'win32',targetArch:'x64',exeSHA256:c.exeSHA256,previewHelperSHA256:c.previewHelper?c.previewHelperSHA256:null,terminalSHA256:c.terminalSHA256,sourcePreparationSHA256:sha(path.join(source,'offline-preparation.json')),sourceBuilds:m.sourceBuilds,sourceInputs:m.inputs,targetAssets:m.assets,links:links.length,layout};
 fs.writeFileSync(path.join(root,'candidate-status.json'),JSON.stringify(result,null,2));
 fs.writeFileSync(path.join(root,'PREPARE-LINKS.cmd'),'@echo off\r\n"%~dp0runtime\\node\\node.exe" "%~dp0restore-offline-links.mjs" "%~dp0."\r\nif errorlevel 1 exit /b 1\r\necho Links restored. Windows native runtime remains UNVERIFIED.\r\n');
 fs.writeFileSync(path.join(root,'README-CANDIDATE.txt'),'UNVERIFIED WINDOWS X64 PORTABLE CANDIDATE\r\nThis folder contains ordinary files only; ZIP extraction does not need symbolic-link support.\r\nOn Windows, run PREPARE-LINKS.cmd with ordinary user permissions to restore only manifest-listed directory junctions and file hardlinks inside this folder. Do not run as administrator or change ExecutionPolicy. If the filesystem does not support junctions/hardlinks, use a local NTFS folder; do not request elevation.\r\nThis step never starts WhaleIsle, Host, mobile, a model or account. It does not mark runtime_verified true. Real Windows native-load and startup verification is still required. Source Linux manifests are retained as provenance only; no production .runtime-staged marker exists.\r\nMove or ZIP this folder BEFORE restoring links. After relocation, re-extract a clean copy rather than accepting stale junctions.\r\n');
 return {destination:root,status:result.status,runtime_verified:false,links:links.length,layout};}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){try{console.log(JSON.stringify(packageCandidate(process.argv[2]),null,2));}catch(e){console.error(e.stack);process.exitCode=1;}}
