import fs from 'node:fs';import path from 'node:path';import crypto from 'node:crypto';
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
// Remove POSIX/.bin launchers containing build-machine absolute NODE_PATHs.
// Generate Windows wrappers using only candidate-relative paths.
export function normalizeWindowsLayout(root, manifest) {
 const packages=[],bins=[];let detached=0;
 function scan(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){const p=path.join(dir,e.name);if(e.isSymbolicLink())continue;if(e.isDirectory()){if(e.name==='.bin'){bins.push(p);continue;}scan(p);}else {if(e.name==='package.json')packages.push(p);if((/\.(json|ya?ml|toml|ini|conf|env)$/i.test(p)||e.name.startsWith('.env'))&&fs.statSync(p).nlink>1){const b=fs.readFileSync(p),mode=fs.statSync(p).mode;fs.unlinkSync(p);fs.writeFileSync(p,b,{mode});detached++;}}}}
 scan(root);
 if(manifest)for(const group of ['files','links'])for(const key of Object.keys(manifest[group]))if(key.includes('/.bin/'))delete manifest[group][key];
 for(const bin of bins)fs.rmSync(bin,{recursive:true,force:true});
 let wrappers=0;
 for(const file of packages){let m;try{m=JSON.parse(fs.readFileSync(file,'utf8'));}catch{continue;}if(!m.bin)continue;const dir=path.dirname(file);let nm=path.dirname(dir);if(path.basename(nm).startsWith('@'))nm=path.dirname(nm);if(path.basename(nm)!=='node_modules')continue;
 const entries=typeof m.bin==='string'?{[m.name.split('/').pop()]:m.bin}:m.bin;
 for(const [name,entry] of Object.entries(entries)){if(!/^[a-zA-Z0-9_.-]+$/.test(name)||typeof entry!=='string')continue;const executable=path.resolve(dir,entry);if(!fs.statSync(executable,{throwIfNoEntry:false})?.isFile())continue;
 const bin=path.join(nm,'.bin'),out=path.join(bin,name+'.cmd');fs.mkdirSync(bin,{recursive:true});const relative=path.relative(bin,executable).replaceAll('/','\\');const node=path.relative(bin,path.join(root,'node/node.exe')).replaceAll('/','\\');const command=entry.endsWith('.exe')?`"%~dp0${relative}"`:`"%~dp0${node}" "%~dp0${relative}"`;fs.writeFileSync(out,`@echo off\r\n${command} %*\r\n`);wrappers++;if(manifest)manifest.files[path.relative(root,out).split(path.sep).join('/')]=sha(out);
 }}
 return {detachedConfigurationFiles:detached,windowsBinWrappers:wrappers};
}
