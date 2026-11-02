import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import {stage, verify, prepare, required, shellFiles} from './mobile-runtime.mjs';

test('one isolated staging fixture: complete inventory, repeat build, missing resources fail closed',()=>{
  const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'mobile-staging-'));
  const work=path.join(tmp,'work'), harness=path.join(tmp,'harness'), node=path.join(tmp,'node'), dest=path.join(tmp,'out');
  const put=(p,value='fixture, never executable')=>{fs.mkdirSync(path.dirname(p),{recursive:true});fs.writeFileSync(p,value);};
  try {
    for(const f of required) {
      if(f.startsWith('node/')) put(path.join(node,f.slice(5)));
      else if(f.startsWith('vendor/deepseek-harness/')) put(path.join(harness,f.slice('vendor/deepseek-harness/'.length)));
      else put(path.join(work,f));
    }
    fs.mkdirSync(path.join(harness,'packages'),{recursive:true});
    put(path.join(work,'source-provenance.json'),'{}');
    put(path.join(work,'node-build.json'),JSON.stringify({platform:'win32',arch:'x64',nodeSHA256:crypto.createHash('sha256').update(fs.readFileSync(path.join(node,'node.exe'))).digest('hex')}));
    put(path.join(work,'vendor/chisacode-remote/node_modules/node-pty/prebuilds/win32-x64/conpty.node'));
    stage(work,harness,node,dest); verify(dest);
    stage(work,harness,node,dest); verify(dest);
    const m=JSON.parse(fs.readFileSync(path.join(dest,'staging-manifest.json')));
    for(const f of required) assert.ok(m.files[f],f);
    fs.unlinkSync(path.join(dest,shellFiles[0]));
    assert.throws(()=>verify(dest),/resource missing/);
    fs.unlinkSync(path.join(work,'mobile/web/chisacode/daemon-client.bundle.js'));
    assert.throws(()=>stage(work,harness,node,dest),/resource missing/);
    assert.throws(()=>prepare(work,path.join(tmp,'new-work')),/silero_vad.onnx/);
  } finally {fs.rmSync(tmp,{recursive:true,force:true});}
});
