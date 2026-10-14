// One isolated configuration fixture; never executes the Host or build toolchain.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {artifacts,commit,finalize,verify} from './current-host.mjs';
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
 const m=finalize(root);assert.equal(m.sourceCommit,commit);assert.deepEqual(m.capabilities,{project:false,team:false,visualReplies:false});
 assert.equal(verify(root).buildProfile,'official');
 fs.writeFileSync(path.join(root,artifacts[0]),'changed');assert.throws(()=>verify(root),/missing or changed/);
 console.log('Configuration fixture OK: matching artifacts, capability defaults off, changed artifact rejected.');
} finally {fs.rmSync(root,{recursive:true,force:true});}
