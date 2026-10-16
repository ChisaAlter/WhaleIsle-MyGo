// Package supplied original component payloads; never execute or download them.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {createRequire} from 'node:module';
const require=createRequire(import.meta.url);
const {readManifest}=require('../internal/desktop/components/generic-source/registry.js');
const {validComponentId}=require('../internal/desktop/components/generic-source/store.js');

export function stageComponentSamples(source, destination) {
 source=fs.realpathSync(source);destination=path.resolve(destination);
 if(destination===source||destination.startsWith(source+path.sep)||source.startsWith(destination+path.sep))throw Error('Separate source and destination required');
 if(fs.existsSync(destination))throw Error('Component samples destination must be fresh');
 // Snapshot all ordinary payload bytes first. No source link can escape the input.
 const files=[];
 function collect(relative='') {
  const absolute=path.join(source,relative),st=fs.lstatSync(absolute);
  if(st.isSymbolicLink())throw Error('Component payload symlinks are not supported: '+relative);
  if(st.isDirectory()){for(const name of fs.readdirSync(absolute))collect(path.join(relative,name));}
  else if(st.isFile())files.push({relative,bytes:fs.readFileSync(absolute)});
  else throw Error('Component payload must contain ordinary files: '+relative);
 }
 collect();
 let count=0;
 for(const id of fs.readdirSync(source,{withFileTypes:true})) {
  if(!id.isDirectory())continue;
  if(!validComponentId(id.name)||['whalebridge','whaleisle-host'].includes(id.name))throw Error('Invalid generic component ID: '+id.name);
  for(const version of fs.readdirSync(path.join(source,id.name),{withFileTypes:true})) {
   if(!version.isDirectory())continue;
   const manifest=readManifest(path.join(source,id.name,version.name));
   if(!manifest||manifest.id!==id.name||manifest.version!==version.name)throw Error('Invalid component manifest: '+id.name+'/'+version.name);
   count++;
  }
 }
 if(!count)throw Error('Bundled component catalog is empty; supply the original components/samples payloads');
 fs.mkdirSync(path.dirname(destination),{recursive:true});
 const temporary=fs.mkdtempSync(path.join(path.dirname(destination),'.samples-stage-'));
 try {
  for(const {relative,bytes} of files){const out=path.join(temporary,relative);fs.mkdirSync(path.dirname(out),{recursive:true});fs.writeFileSync(out,bytes,{flag:'wx'});}
  // Validate the actual snapshotted bytes, not just their source metadata.
  for(const id of fs.readdirSync(temporary,{withFileTypes:true}))if(id.isDirectory()){
   for(const version of fs.readdirSync(path.join(temporary,id.name),{withFileTypes:true}))if(version.isDirectory()){
    const manifest=readManifest(path.join(temporary,id.name,version.name));
    if(!manifest||manifest.id!==id.name||manifest.version!==version.name)throw Error('Component source changed during staging');
   }
  }
  fs.renameSync(temporary,destination);
 } finally {fs.rmSync(temporary,{recursive:true,force:true});}
 return {versions:count,files:files.map(({relative,bytes})=>({path:relative.split(path.sep).join('/'),sha256:crypto.createHash('sha256').update(bytes).digest('hex')}))};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 try{if(process.argv.length!==4)throw Error('Usage: node component-samples.mjs SOURCE_SAMPLES DESTINATION_SAMPLES');console.log(JSON.stringify(stageComponentSamples(process.argv[2],process.argv[3])));}
 catch(error){console.error(error.message);process.exitCode=1;}
}
