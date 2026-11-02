import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {inventoryGraph,verifyGraph,cloneGraph} from './session-runtime-graph.mjs';
import {emitRuntimeInclude} from './session-runtime-nsis.mjs';

test('physical NSIS include preserves closed cyclic graph without expanding aliases',()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'runtime-nsis-'));
 try {
  const stage=path.join(root,'stage'),vendor=path.join(stage,'runtime/vendor');
  for(const name of ['one','two']){fs.mkdirSync(path.join(vendor,name,'node_modules'),{recursive:true});fs.writeFileSync(path.join(vendor,name,'index.js'),'// '+name);}
  fs.symlinkSync(path.join(vendor,'two'),path.join(vendor,'one/node_modules/two'),process.platform==='win32'?'junction':'dir');
  fs.symlinkSync(path.join(vendor,'one'),path.join(vendor,'two/node_modules/one'),process.platform==='win32'?'junction':'dir');
  const actual=inventoryGraph(vendor),manifest={schemaVersion:1,files:actual.files,links:actual.links};
  fs.writeFileSync(path.join(vendor,'runtime-graph.json'),JSON.stringify(manifest));
  assert.equal(verifyGraph(vendor,manifest,{exclude:['runtime-graph.json']}).files['one/index.js'],manifest.files['one/index.js']);
  const include=path.join(root,'physical.nsh');const result=emitRuntimeInclude(stage,include);
  assert.equal(result.files,3);assert.equal(result.links,2);
  const text=fs.readFileSync(include,'utf8');assert.equal(text.split('\n').filter(line=>line.startsWith('File ')).length,3);
   assert.match(text,/!insertmacro WI_SetPayloadOutPath "\$INSTDIR\\runtime\\vendor\\one"/);
   assert.doesNotMatch(text,/^SetOutPath /m);
  assert.doesNotMatch(text,/File \/r/);
  const copy=path.join(root,'copy');cloneGraph(vendor,copy);assert.equal(fs.realpathSync(path.join(copy,'one/node_modules/two')),path.join(copy,'two'));
  fs.writeFileSync(path.join(copy,'extra.js'),'// untracked');assert.throws(()=>verifyGraph(copy,manifest,{exclude:['runtime-graph.json']}),/Untracked/);
 } finally {fs.rmSync(root,{recursive:true,force:true});}
});
