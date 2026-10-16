'use strict';
// Isolated fixture: private temporary files and injected child processes only.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { EventEmitter } = require('node:events');
const { createComponentsService } = require('./index');

test('original generic lifecycle: catalog, staging, supervision, rollback, data-preserving uninstall', async t => {
 const root = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-components-fixture-'));
 let svc;
 t.after(() => { if(svc)svc.shutdown();fs.rmSync(root,{recursive:true,force:true}); });
 const samplesRoot = path.join(root,'samples');
 const userDataDir = path.join(root,'user');
 const componentRoot = path.join(userDataDir,'components','notes');
 function payload(version) {
  const dir = path.join(samplesRoot,'notes',version);
  fs.mkdirSync(dir,{recursive:true});
  fs.writeFileSync(path.join(dir,'manifest.json'),JSON.stringify({id:'notes',name:'Notes',version,entry:'entry.js'}));
  fs.writeFileSync(path.join(dir,'entry.js'),'// Fixture payload is never executed.');
 }
 payload('1.0.0');
 let pid = 100;
 const live = new Map();
 const launches = [];
 const deps = {
  fs: {...fs, createWriteStream: () => ({end() {}})}, userDataDir, samplesRoot, platform:'win32', execPath:'injected-node',
  startGraceMs:0, stateWaitMs:0, restartBaseMs:0, maxRestarts:3,
  processKill(id, signal) {
   assert.equal(signal,0);
   if (!live.has(id)) throw Object.assign(new Error('absent'),{code:'ESRCH'});
  },
  spawn(exe,args,options) {
   assert.equal(exe,'injected-node');
   assert.equal(options.env.LAUNCHER_COMPONENT_ID,'notes');
   const child = new EventEmitter();child.pid=++pid;child.exitCode=null;
   live.set(child.pid,child);launches.push({args,options});return child;
  },
  execFileSync(exe,args) {
   assert.equal(exe,'taskkill');
   const id = Number(args[1]);const child=live.get(id);live.delete(id);
   if(child){child.exitCode=0;child.emit('exit',0);}
  },
 };
 svc = createComponentsService(deps);
 assert.equal(svc.list().components[0].state,'available');
 assert.equal((await svc.install('notes')).ok,true);
 assert.equal(launches.length,0,'installation must not auto-start');
 const data=path.join(componentRoot,'data','notes.txt');fs.writeFileSync(data,'keep my notes');
 assert.equal((await svc.start('notes')).ok,true);
 assert.equal((await svc.start('notes')).ok,true);
 assert.equal(launches.length,1,'start is idempotent');
 payload('2.0.0');
 assert.equal(svc.list().components[0].updateAvailable,true);
 assert.equal((await svc.update('notes')).to,'2.0.0');
 assert.equal(svc.list().components[0].previousVersion,'1.0.0');
 assert.equal(svc.list().components[0].state,'running');
 assert.equal((await svc.rollback('notes')).to,'1.0.0');
 assert.equal(svc.list().components[0].previousVersion,'2.0.0');
 assert.equal(fs.readFileSync(data,'utf8'),'keep my notes');
 assert.equal((await svc.stop('notes')).ok,true);
 assert.equal(live.size,0);
 assert.equal((await svc.uninstall('notes')).ok,true);
 assert.equal(fs.existsSync(path.join(componentRoot,'versions')),false);
 assert.equal(fs.readFileSync(data,'utf8'),'keep my notes');
 assert.equal(svc.list().components[0].state,'available');
});
