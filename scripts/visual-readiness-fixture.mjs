// Deterministic gate fixture; no HTTP server, Host, models, or real credentials.
import assert from 'node:assert/strict';
import {replacement,patchSource} from './current-host-readiness.mjs';
import {createRequire} from 'node:module';
import fs from 'node:fs';
import path from 'node:path';
const [source,build]=process.argv.slice(2);
if(!source || !build)throw new Error('Pass fixed source checkout and prepared Host build root');
const original=fs.readFileSync(path.join(source,'vendor/deepseek-harness/packages/client/ui-visual-replies/src/tools.ts'),'utf8');
const patched=patchSource(original);
assert.equal((patched.match(/await requireEnabled\(config, exec.signal\)/g)||[]).length,7);
const require=createRequire(path.resolve(build,'deepseek-harness/package.json'));
const ts=require('typescript');
const js=ts.transpileModule(replacement+'\nexport {requireEnabled};',{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ESNext}}).outputText;
const {requireEnabled}=await import('data:text/javascript;base64,'+Buffer.from(js).toString('base64'));
const previousFetch=globalThis.fetch, oldURL=process.env.DSH_DESKTOP_INSTALL_URL, oldToken=process.env.DSH_DESKTOP_INSTALL_TOKEN;
let enabled=true, calls=0;
const config={enabled:{get:()=>enabled}};
try {
 process.env.DSH_DESKTOP_INSTALL_URL='http://127.0.0.1:1';
 process.env.DSH_DESKTOP_INSTALL_TOKEN='synthetic-fixture-only';
 globalThis.fetch=async(url,options)=>{calls++;assert.equal(url.pathname,'/desktop/html-preview/status');assert.equal(options.method,'GET');assert.equal(options.redirect,'error');return {ok:true,json:async()=>({available:false,reason:'not verified in this process'})};};
 await assert.rejects(requireEnabled(config,new AbortController().signal),/not verified in this process/);
 assert.equal(enabled,true); // persisted user preference is never overwritten
 globalThis.fetch=async()=>({ok:true,json:async()=>({available:true})});
 await requireEnabled(config,new AbortController().signal);
 globalThis.fetch=async()=>({ok:true,json:async()=>({available:'true'})});
 await assert.rejects(requireEnabled(config,new AbortController().signal),/verify HTML preview/);
 enabled=false;
 await assert.rejects(requireEnabled(config,new AbortController().signal),/disabled/);
 assert.equal(calls,1);
 console.log('Readiness fixture OK: persisted true fails closed until current-process available; both tools have checkpoints; preference unchanged.');
} finally {globalThis.fetch=previousFetch;if(oldURL===undefined)delete process.env.DSH_DESKTOP_INSTALL_URL;else process.env.DSH_DESKTOP_INSTALL_URL=oldURL;if(oldToken===undefined)delete process.env.DSH_DESKTOP_INSTALL_TOKEN;else process.env.DSH_DESKTOP_INSTALL_TOKEN=oldToken;}
