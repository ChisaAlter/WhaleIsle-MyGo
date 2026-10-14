// Ordinary-user Windows junction/hardlink restoration. No elevation or policy changes.
import fs from 'node:fs';import path from 'node:path';import {fileURLToPath} from 'node:url';
export function planLinks(root,manifest){
 root=path.resolve(root);if(manifest.runtime_verified!==false||manifest.targetPlatform!=='win32'||manifest.targetArch!=='x64')throw Error('Expected unverified Windows x64 candidate');
 const safe=relative=>{if(typeof relative!=='string'||!relative||relative.includes('\\')||relative.split('/').some(p=>!p||p==='.'||p==='..'||p.includes(':')||/[<>"|?*]/.test(p)||/[. ]$/.test(p)||/^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\.|$)/i.test(p))||path.isAbsolute(relative))throw Error('Unsafe candidate path');return path.join(root,...relative.split('/'));};
 const planned=[];const names=new Set();
 for(const item of manifest.links){const link=safe(item.path),target=safe(item.target);if(names.has(link)||link===target||!['directory','file'].includes(item.kind))throw Error('Invalid duplicate/self link');names.add(link);planned.push({...item,link,target});}
 function parents(p){let current=path.dirname(p);while(current!==root){const st=fs.lstatSync(current,{throwIfNoEntry:false});if(st?.isSymbolicLink()||names.has(current))throw Error('Link path has redirected ancestor');if(st&&!st.isDirectory())throw Error('Non-directory parent');current=path.dirname(current);}}
 const rootReal=fs.realpathSync(root);if(rootReal!==root)throw Error('Candidate root must not be redirected');
 for(const item of planned){parents(item.link);parents(item.target);const source=fs.lstatSync(item.target,{throwIfNoEntry:false});if(!source||source.isSymbolicLink()||(item.kind==='directory'?!source.isDirectory():!source.isFile()))throw Error(`Missing physical link target: ${item.path}`);
  const existing=fs.lstatSync(item.link,{throwIfNoEntry:false});if(existing){if(item.kind==='directory'){if(!existing.isSymbolicLink()||fs.realpathSync(item.link)!==item.target)throw Error(`Refuse unexpected existing directory: ${item.path}`);}else if(!existing.isFile()||existing.isSymbolicLink()||existing.ino!==source.ino||existing.dev!==source.dev)throw Error(`Refuse unexpected existing file: ${item.path}`);item.exists=true;}
 }
 return planned;
}
export function restore(root){if(process.platform!=='win32')throw Error('Restore only on Windows; static planning is cross-platform');const file=path.join(root,'candidate-links.json'),manifest=JSON.parse(fs.readFileSync(file));const plan=planLinks(root,manifest);let created=0;for(const item of plan){if(item.exists)continue;fs.mkdirSync(path.dirname(item.link),{recursive:true});if(item.kind==='directory')fs.symlinkSync(item.target,item.link,'junction');else fs.linkSync(item.target,item.link);created++;}return {created,runtime_verified:false,status:'links-restored-native-runtime-unverified'};}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){try{console.log(JSON.stringify(restore(path.resolve(process.argv[2]||path.dirname(fileURLToPath(import.meta.url)))),null,2));}catch(e){console.error(e.message);process.exitCode=1;}}
