// One isolated configuration fixture; never executes the Host or build toolchain.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {artifacts,commit,finalize,verify,prepare} from './current-host.mjs';
import {applyReadinessGuard,toolsPath,marker} from './current-host-readiness.mjs';
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-current-host-fixture-'));
try {
 const lock=path.join(root,'deepseek-harness/pnpm-lock.yaml');
 fs.mkdirSync(path.dirname(lock),{recursive:true});fs.writeFileSync(lock,'fixture-lock');
 fs.writeFileSync(path.join(root,'source-selection.json'),JSON.stringify({sourceCommit:commit,sourcePath:'fixture',packageManager:'pnpm@11.8.0',lockSHA256:crypto.createHash('sha256').update('fixture-lock').digest('hex')}));
 for(const f of artifacts){fs.mkdirSync(path.dirname(path.join(root,f)),{recursive:true});fs.writeFileSync(path.join(root,f),'fixture-artifact');}
 const original=process.argv[2] || process.env.WHALEISLE_CURRENT_ELECTRON_SOURCE;
 if(!original) throw new Error('Pass the fixed official source checkout to this configuration fixture.');
 fs.mkdirSync(path.dirname(path.join(root,toolsPath)),{recursive:true});
 fs.copyFileSync(path.join(original,'vendor/deepseek-harness/packages/client/ui-visual-replies/src/tools.ts'),path.join(root,toolsPath));
 fs.chmodSync(path.join(root,toolsPath),0o644);
 applyReadinessGuard(root);
 fs.writeFileSync(path.join(root,'deepseek-harness/packages/client/ui-visual-replies/lib/index.js'),marker+' /desktop/html-preview/status');
 for (const sourceCommit of [commit, 'e9f584be6d3b491ce777bb6823f30bc2bbc6f195']) {
 const selection=JSON.parse(fs.readFileSync(path.join(root,'source-selection.json'),'utf8'));
 selection.sourceCommit=sourceCommit;fs.writeFileSync(path.join(root,'source-selection.json'),JSON.stringify(selection));
 const m=finalize(root);assert.equal(m.sourceCommit,sourceCommit);assert.deepEqual(m.capabilities,{project:false,team:false,visualReplies:false});
 assert.equal(verify(root).buildProfile,'official');
 }
 const manifestPath=path.join(root,'current-host-build.json');
 const valid=fs.readFileSync(manifestPath,'utf8');
 const unsupported=JSON.parse(valid);unsupported.sourceCommit='f'.repeat(40);fs.writeFileSync(manifestPath,JSON.stringify(unsupported));
 assert.throws(()=>verify(root),/Unsupported/);fs.writeFileSync(manifestPath,valid);
 assert.throws(()=>prepare(original,path.join(root,'unsupported-source'),'f'.repeat(40)),/Unsupported/);
 fs.writeFileSync(path.join(root,artifacts[0]),'changed');assert.throws(()=>verify(root),/missing or changed/);
 console.log('Configuration fixture OK: 904/e9 manifests, capability defaults off, unknown provenance and changed artifact rejected.');
} finally {fs.rmSync(root,{recursive:true,force:true});}
