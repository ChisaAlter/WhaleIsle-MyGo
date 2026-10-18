// Inert provenance/input gates only. No target binary is run and no package is emitted.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {artifacts, supportedCommits} from './current-host.mjs';
import {toolsPath,indexPath,marker,guardID,baseSHA256,patchSHA256} from './current-host-readiness.mjs';
import {buildCatalog} from './native-release-catalog.mjs';
import {inspectMacOSRuntime} from './macos-native-artifacts.mjs';
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-native-source-'));
const platform=Object.getOwnPropertyDescriptor(process,'platform'),arch=Object.getOwnPropertyDescriptor(process,'arch');
const digest=b=>crypto.createHash('sha256').update(b).digest('hex');
try {
 const stage=path.join(root,'stage'),runtime=path.join(stage,'runtime'),vendor=path.join(runtime,'vendor');
 const write=(name,data)=>{const file=path.join(vendor,name);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,data);};
 const hashes={};
 for(const name of artifacts){const data=name===indexPath?marker+' /desktop/html-preview/status':'inert';write(name,data);hashes[name]=digest(data);}
 write(toolsPath,'inert source');
 const safetyPatch={id:guardID,baseSHA256,patchSHA256,sourceSHA256:digest('inert source')};
 write('visual-readiness-patch.json',JSON.stringify(safetyPatch));
 const m={schemaVersion:1,sourceCommit:supportedCommits[0],buildProfile:'official',platform:'win32',arch:'x64',safetyPatch,artifacts:hashes};
 const save=()=>write('current-host-build.json',JSON.stringify(m));
 const pe=Buffer.alloc(128);pe.write('MZ');pe.writeUInt32LE(64,60);pe.writeUInt32LE(0x4550,64);pe.writeUInt16LE(0x8664,68);
 fs.writeFileSync(path.join(stage,'whaleisle.exe'),pe);fs.writeFileSync(path.join(stage,'.runtime-staged'),'1');
 const installer=path.join(root,'setup.exe');fs.writeFileSync(installer,pe);
 const config={version:'1.2.3',stage,installer,baseURL:'https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/'};
 for(const target of ['win32','darwin']){
  Object.defineProperty(process,'platform',{value:target});Object.defineProperty(process,'arch',{value:'x64'});m.platform=target;
  const inspect=()=>target==='win32'?buildCatalog(config):inspectMacOSRuntime(runtime,'amd64');
  for(const pin of supportedCommits){m.sourceCommit=pin;save();await assert.rejects(inspect,/Mobile runtime resource missing: node-build\.json/,`${target}/${pin} must reach independent mobile gate`);}
  m.sourceCommit='f'.repeat(40);save();await assert.rejects(inspect,/Unsupported current Host build provenance/);
  m.sourceCommit=supportedCommits[1];m.safetyPatch=null;save();await assert.rejects(inspect,/matching visual readiness safety patch/);
  m.safetyPatch=safetyPatch;m.platform='linux';save();await assert.rejects(inspect,/platform\/architecture/);
  m.platform=target;save();write(artifacts[0],'changed');await assert.rejects(inspect,/artifact missing or changed/);write(artifacts[0],'inert');
 }
 console.log('Native release source gate PASS: Windows catalog and macOS runtime accept both exact pins; source, guard, platform and hash rejection retained. No native runtime executed.');
} finally {Object.defineProperty(process,'platform',platform);Object.defineProperty(process,'arch',arch);fs.rmSync(root,{recursive:true,force:true});}
