import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {stageOptionalSamples} from './session-runtime-samples.mjs';
test('absent implicit catalog adds no files; explicit missing, empty and invalid payloads remain errors',()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'runtime-samples-'));
 try {
  const absent=path.join(root,'absent'),out=path.join(root,'out');
  const result=stageOptionalSamples(absent,out);
  assert.equal(result.versions,0);assert.equal(fs.existsSync(out),false);
  assert.throws(()=>stageOptionalSamples(absent,out,true),/ENOENT/);
  const empty=path.join(root,'empty');fs.mkdirSync(empty);
  assert.throws(()=>stageOptionalSamples(empty,out),/catalog is empty/);
  const invalid=path.join(root,'invalid');fs.mkdirSync(path.join(invalid,'real-component/1.0.0'),{recursive:true});
  fs.writeFileSync(path.join(invalid,'real-component/1.0.0/manifest.json'),JSON.stringify({id:'real-component',version:'1.0.0',entry:'missing.js'}));
  assert.throws(()=>stageOptionalSamples(invalid,out),/Invalid component manifest/);
  assert.equal(fs.existsSync(out),false);
 } finally {fs.rmSync(root,{recursive:true,force:true});}
});
