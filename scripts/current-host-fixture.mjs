// One isolated configuration fixture; never executes the Host or build toolchain.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import assert from 'node:assert/strict';
import {artifacts,commit,finalize,verify} from './current-host.mjs';
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-current-host-fixture-'));
try {
 const lock=path.join(root,'deepseek-harness/pnpm-lock.yaml');
 fs.mkdirSync(path.dirname(lock),{recursive:true});fs.writeFileSync(lock,'fixture-lock');
 fs.writeFileSync(path.join(root,'source-selection.json'),JSON.stringify({sourceCommit:commit,sourcePath:'fixture',packageManager:'pnpm@11.8.0',lockSHA256:crypto.createHash('sha256').update('fixture-lock').digest('hex')}));
 for(const f of artifacts){fs.mkdirSync(path.dirname(path.join(root,f)),{recursive:true});fs.writeFileSync(path.join(root,f),'fixture-artifact');}
 const m=finalize(root);assert.equal(m.sourceCommit,commit);assert.deepEqual(m.capabilities,{project:false,team:false,visualReplies:false});
 assert.equal(verify(root).buildProfile,'official');
 fs.writeFileSync(path.join(root,artifacts[0]),'changed');assert.throws(()=>verify(root),/missing or changed/);
 console.log('Configuration fixture OK: matching artifacts, capability defaults off, changed artifact rejected.');
} finally {fs.rmSync(root,{recursive:true,force:true});}
