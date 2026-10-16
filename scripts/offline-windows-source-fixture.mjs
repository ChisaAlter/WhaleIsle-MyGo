// Input-gate fixture only: no target executable, asset download or runtime assembly.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {prepare} from './offline-windows-runtime.mjs';
import {supportedCommits} from './current-host.mjs';
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-offline-source-fixture-'));
try {
 const host=path.join(root,'host'),mobile=path.join(root,'mobile'),destination=path.join(root,'candidate');
 fs.mkdirSync(host);fs.mkdirSync(mobile);
 fs.writeFileSync(path.join(host,'artifact.js'),'fixture');
 const digest=crypto.createHash('sha256').update('fixture').digest('hex');
 // This deliberately stops preparation at the next independent gate, before any clone.
 fs.writeFileSync(path.join(mobile,'node-build.json'),JSON.stringify({node:'not-a-runtime',abi:'0'}));
 const config={targetPlatform:'win32',targetArch:'x64',inputs:{host,mobile},destination};
 const configFile=path.join(root,'config.json');
 const writeConfig=()=>fs.writeFileSync(configFile,JSON.stringify(config));writeConfig();
 const manifest={sourceCommit:'',safetyPatch:{id:'mygo-visual-readiness-v1'},artifacts:{'artifact.js':digest}};
 const writeManifest=()=>fs.writeFileSync(path.join(host,'current-host-build.json'),JSON.stringify(manifest));
 for(const sourceCommit of supportedCommits){
  manifest.sourceCommit=sourceCommit;writeManifest();
  assert.throws(()=>prepare(configFile),/requires the verified Node 24\.19\.0/,`Allowed ${sourceCommit} must reach the next gate`);
 }
 manifest.sourceCommit='f'.repeat(40);writeManifest();assert.throws(()=>prepare(configFile),/Expected guarded fixed Host build/);
 manifest.sourceCommit=supportedCommits[1];manifest.safetyPatch=null;writeManifest();assert.throws(()=>prepare(configFile),/Expected guarded fixed Host build/);
 manifest.safetyPatch={id:'mygo-visual-readiness-v1'};writeManifest();
 fs.writeFileSync(path.join(host,'artifact.js'),'changed');assert.throws(()=>prepare(configFile),/Changed Host input/);
 config.targetPlatform='linux';writeConfig();assert.throws(()=>prepare(configFile),/Only explicit win32\/x64/);
 assert.equal(fs.existsSync(destination),false);
 console.log('Offline source gate fixture PASS: 904/e9 accepted; unknown source, missing guard, changed artifact and non-Windows target rejected; no assembly created.');
}finally{fs.rmSync(root,{recursive:true,force:true});}
