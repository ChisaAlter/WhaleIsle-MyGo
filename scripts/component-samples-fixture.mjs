// Offline fixture only: synthetic source files, never executed as components.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {createRequire} from 'node:module';
import {stageComponentSamples} from './component-samples.mjs';
const require=createRequire(import.meta.url);
const {scanCatalog}=require('../internal/desktop/components/generic-source/registry.js');
const root=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-samples-stage-'));
try {
 const source=path.join(root,'source'), stage=path.join(root,'stage');
 fs.mkdirSync(source);
 assert.throws(()=>stageComponentSamples(source,stage),/catalog is empty/);
 assert.equal(fs.existsSync(stage),false);
 const payload=path.join(source,'notes','1.0.0');fs.mkdirSync(payload,{recursive:true});
 fs.writeFileSync(path.join(payload,'manifest.json'),JSON.stringify({id:'notes',version:'1.0.0',entry:'entry.js'}));
 fs.writeFileSync(path.join(payload,'entry.js'),'// fixture, never execute');
 stageComponentSamples(source,stage);
 assert.equal(scanCatalog({samplesRoot:stage})[0].id,'notes');
 assert.equal(fs.readFileSync(path.join(stage,'notes','1.0.0','entry.js'),'utf8'),'// fixture, never execute');
 fs.rmSync(stage,{recursive:true});
 stageComponentSamples(source,stage); // fresh packaging after cleanup restores catalog
 fs.writeFileSync(path.join(payload,'manifest.json'),JSON.stringify({id:'notes',version:'2.0.0',entry:'entry.js'}));
 assert.throws(()=>stageComponentSamples(source,path.join(root,'bad')),/Invalid component manifest/);
 assert.equal(fs.existsSync(path.join(root,'bad')),false);
 const windows=fs.readFileSync(new URL('./package.ps1',import.meta.url),'utf8');
 assert.ok(windows.indexOf("'component-samples.mjs'")>=0);
 assert.ok(windows.indexOf("'component-samples.mjs'")<windows.indexOf('Set-Content -Path $Marker'));
 const mac=fs.readFileSync(new URL('./package-macos.mjs',import.meta.url),'utf8');
 assert.match(mac,/stageComponentSamples\(.*path.join\(runtimeDir, 'components', 'samples'\)/);
 console.log('Bundled component clean-stage catalog fixture passed; no native packaging or payload execution.');
} finally {fs.rmSync(root,{recursive:true,force:true});}
