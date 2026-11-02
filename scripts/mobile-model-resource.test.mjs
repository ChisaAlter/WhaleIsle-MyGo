import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {prepare,shellFiles,sourceAssets} from './mobile-runtime.mjs';

// Supply the independently verified local asset. Never download it in a test.
const model=process.env.WHALEISLE_SILERO_VAD_FILE;
test('local pinned model: match copied only to work, wrong hash/missing input rejected', {skip:!model},()=>{
  const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'mobile-model-'));
  const source=path.join(tmp,'source'),work=path.join(tmp,'work');
  const put=(f,value='fixture')=>{const p=path.join(source,f);fs.mkdirSync(path.dirname(p),{recursive:true});fs.writeFileSync(p,value);};
  try {
    for(const f of [...shellFiles,'mobile/web/index.html','scripts/bundle-chisacode-mobile-client.mjs','vendor/chisacode-remote/package-lock.json','vendor/chisacode-remote/packages/server/src/share/QRCode.ts']) put(f);
    put('vendor/chisacode-remote/package.json',JSON.stringify({version:'1.0.3'}));
    assert.throws(()=>prepare(source,work),/resource missing.*silero_vad/);
    prepare(source,work,model);
    assert.deepEqual(fs.readFileSync(path.join(work,sourceAssets[0])),fs.readFileSync(model));
    assert.equal(fs.existsSync(path.join(source,sourceAssets[0])),false);
    const receipt=JSON.parse(fs.readFileSync(path.join(work,'source-provenance.json')));
    assert.equal(receipt.assets[0].sha256,'9e2449e1087496d8d4caba907f23e0bd3f78d91fa552479bb9c23ac09cbb1fd6');
    assert.equal(receipt.assets[0].suppliedVia,'explicit-local-file');
    const bad=path.join(tmp,'bad.onnx'),bytes=fs.readFileSync(model);bytes[0]^=1;fs.writeFileSync(bad,bytes);
    assert.throws(()=>prepare(source,work,bad),/checksum mismatch/);
    assert.deepEqual(fs.readFileSync(path.join(work,sourceAssets[0])),fs.readFileSync(model));
    assert.equal(fs.existsSync(path.join(source,sourceAssets[0])),false);
  }finally{fs.rmSync(tmp,{recursive:true,force:true});}
});
