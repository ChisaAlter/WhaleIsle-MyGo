'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { EventEmitter } = require('node:events');
const { createComponentsService } = require('./index');

test('failed termination, removal and startup never publish successful lifecycle', async t => {
 const root = fs.mkdtempSync(path.join(os.tmpdir(),'mygo-component-failures-'));
 t.after(() => fs.rmSync(root,{recursive:true,force:true}));
 const samplesRoot=path.join(root,'samples'), userDataDir=path.join(root,'user');
 const versions=path.join(userDataDir,'components','notes','versions');
 function payload(version) {
  const dir=path.join(samplesRoot,'notes',version);fs.mkdirSync(dir,{recursive:true});
  fs.writeFileSync(path.join(dir,'manifest.json'),JSON.stringify({id:'notes',version,entry:'entry.js'}));
  fs.writeFileSync(path.join(dir,'entry.js'),'// Never executed');
 }
 payload('1.0.0');
 let live, killFails=true, removeFails=false, dieDuringWait=false, starts=0;
 const fakeFS={...fs,promises:{...fs.promises,rm:async (p,o)=>{
  if(removeFails&&p===versions)throw Object.assign(new Error('locked'),{code:'EPERM'});
  return fs.promises.rm(p,o);
 }},createWriteStream:()=>({end(){}}),readFileSync:(p,...args)=>{
  if(dieDuringWait&&path.basename(p)==='state.json'&&live){live.exitCode=1;const dead=live;live=null;dead.emit('exit',1);}
  return fs.readFileSync(p,...args);
 }};
 const svc=createComponentsService({fs:fakeFS,userDataDir,samplesRoot,platform:'win32',
  startGraceMs:0,stateWaitMs:1,stopGraceMs:0,stopSoftGraceMs:0,
  processKill:()=>{if(!live)throw Object.assign(new Error('gone'),{code:'ESRCH'});},
  spawn:()=>{starts++;live=new EventEmitter();live.pid=123;live.exitCode=null;return live;},
  execFileSync:()=>{if(killFails)throw new Error('access denied');if(live){const dead=live;live=null;dead.exitCode=0;dead.emit('exit',0);}}
 });
 assert.equal((await svc.install('notes')).ok,true);
 assert.equal((await svc.start('notes')).ok,true);
 payload('2.0.0');
 assert.equal((await svc.stop('notes')).ok,false,'live PID must not be reported stopped');
 let row=svc.list().components[0];assert.equal(row.pid,123);assert.equal(row.state,'running');assert.ok(row.message);
 assert.equal((await svc.update('notes')).ok,false,'failed stop blocks version switch');
 assert.equal(svc.list().components[0].installedVersion,'1.0.0');assert.equal(starts,1);
 assert.equal((await svc.uninstall('notes')).ok,false,'failed stop blocks uninstall');
 killFails=false;assert.equal((await svc.stop('notes')).ok,true);
 removeFails=true;assert.equal((await svc.uninstall('notes')).ok,false,'failed removal retains registry');
 assert.equal(svc.list().components[0].installedVersion,'1.0.0');
 removeFails=false;dieDuringWait=true;
 assert.equal((await svc.start('notes')).ok,false,'death during state wait is not running');
 assert.equal(svc.list().components[0].state,'error');
 assert.equal((await svc.uninstall('notes')).ok,true);
});
