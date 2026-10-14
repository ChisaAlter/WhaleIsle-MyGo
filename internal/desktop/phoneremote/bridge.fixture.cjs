'use strict';
// No subprocesses, network, runtime module, files containing credentials, or
// crypto are used. The original manager boundary is a fictional EventEmitter.
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const {EventEmitter}=require('node:events');
const input=new EventEmitter(), processStub=new EventEmitter();
const events=[],calls=[];
let manager;
class FakeRemote extends EventEmitter {
 constructor(options){super();this.options=options;manager=this;}
 async sync(){calls.push('sync');this.daemon={};this.emit('listening');}
 snapshot(){return {listening:!!this.daemon,relayConnected:true,urls:[{pairingUrl:'https://example.test/#offer=fictional'}]};}
 async stopDaemon(){calls.push('stop');this.daemon=null;}
}
processStub.argv=['node','fixture-module','fixture-home'];
processStub.execPath='fictional-node';
processStub.stdin={};
processStub.stdout={write:s=>events.push(JSON.parse(s))};
processStub.exit=code=>calls.push('exit:'+code);
vm.runInNewContext(fs.readFileSync(__dirname+'/bridge.cjs','utf8'),{
 process:processStub,console:{},setInterval:()=>1,clearInterval:()=>{},
 require:id=>{if(id==='node:readline')return {createInterface:()=>input};if(id==='fixture-module')return {DshdRemote:FakeRemote};throw Error('unexpected import');}
});
(async()=>{
 input.emit('line',JSON.stringify({action:'start',config:{remoteEnabled:true}}));
 await new Promise(setImmediate);
 assert.equal(calls[0],'sync');assert.equal(manager.options.getConfig().remoteEnabled,true);
 assert.equal(events.at(-1).kind,'snapshot');
 input.emit('line',JSON.stringify({action:'refresh',config:{remoteEnabled:true}}));
 await new Promise(setImmediate);assert.equal(calls.filter(x=>x==='sync').length,2);
 input.emit('line',JSON.stringify({action:'stop'}));
 await new Promise(setImmediate);assert.equal(calls.includes('stop'),true);
 assert.equal(events.at(-1).url,'');assert.equal(events.at(-1).listening,false);
 console.log('bridge mocked manager fixture passed');
})().catch(e=>{console.error(e);process.exitCode=1;});
