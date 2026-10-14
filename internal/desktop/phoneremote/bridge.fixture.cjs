'use strict';
// Fictional manager and encoder only: no real daemon, credentials or network.
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const {EventEmitter} = require('node:events');
const input = new EventEmitter(), processStub = new EventEmitter();
const events = [], jobs = [];
let manager, tick;
function offer(id, expires = Date.now() + 60000) {
 return 'https://example.test/#offer=' + Buffer.from(JSON.stringify({v:2,serverId:id,authBootstrap:{expiresAtMs:expires}})).toString('base64url');
}
class FakeRemote extends EventEmitter {
 constructor(){super();manager=this;this.url=offer('a');this.connected=true;}
 async sync(){this.daemon={};this.emit('listening');}
 snapshot(){return {listening:!!this.daemon,relayConnected:this.connected,urls:[{pairingUrl:this.url}]};}
 async stopDaemon(){this.daemon=null;}
}
processStub.argv=['node','/fixture/src/main/dshd-remote.js','fixture-home'];
processStub.execPath='fictional-node';processStub.stdin={};
processStub.stdout={write:s=>events.push(JSON.parse(s))};processStub.exit=()=>{};
vm.runInNewContext(fs.readFileSync(__dirname+'/bridge.cjs','utf8'), {
 __MYGO_GIT_SOURCES__:{},loadOriginalGit:()=>({git:{},authority:{}}),
 createGitGate:()=>({git:{},configure(){},revoke(){}}),
 process:processStub,console:{},Buffer,URL,setInterval:f=>{tick=f;return 1;},clearInterval:()=>{},
 require:id=>{
  if(id==='node:path')return path;
  if(id==='node:module')return {createRequire:p=>{assert.equal(p,'/fixture/vendor/chisacode-remote/node_modules/@chisacode/server/package.json');return id=>{assert.equal(id,'qrcode');return {toBuffer:url=>new Promise(resolve=>jobs.push({url,resolve}))};};}};
  if(id==='node:readline')return {createInterface:()=>input};
  if(id===processStub.argv[1])return {DshdRemote:FakeRemote};
  throw Error('unexpected import');
 }
});
const settle=()=>new Promise(setImmediate);
(async()=>{
 input.emit('line',JSON.stringify({action:'start',config:{remoteEnabled:true}}));await settle();
 assert.equal(jobs.length,1);
 manager.url=offer('b');tick();await settle();
 assert.equal(jobs.length,2);
 jobs[0].resolve(Buffer.from('obsolete'));await settle();
 assert.equal(events.at(-1).qrPNG,'');
 jobs[1].resolve(Buffer.from('current'));await settle();
 assert.equal(events.at(-1).url,manager.url);
 assert.equal(Buffer.from(events.at(-1).qrPNG,'base64').toString(),'current');
 manager.connected=false;tick();assert.equal(events.at(-1).qrPNG,'');assert.equal(events.at(-1).url,'');
 manager.connected=true;manager.url=offer('expired',Date.now()-1);tick();assert.equal(events.at(-1).url,'');
 manager.url=offer('c');tick();await settle();
 input.emit('line',JSON.stringify({action:'stop'}));await settle();
 jobs.at(-1).resolve(Buffer.from('stopped'));await settle();
 assert.equal(events.at(-1).qrPNG,undefined);assert.equal(events.at(-1).url,'');
 assert(!events.some(e=>e.qrPNG===Buffer.from('obsolete').toString('base64')));
 console.log('QR bridge fixture passed: same offer, stale async, disconnect, expiry, stop');
})().catch(e=>{console.error(e);process.exitCode=1;});
