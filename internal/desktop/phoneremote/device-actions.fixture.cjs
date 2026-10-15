'use strict';
// Fictional runtime only. No device credentials, daemon, filesystem writes or network.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const context = {};
vm.runInNewContext(fs.readFileSync(__dirname + '/device-actions.cjs', 'utf8'), context);
async function run() {
  const calls = [];
  let devices = [{id:'device-0001', name:'Phone', secret:'not-public', secretHash:'not-public'}];
  const remote = {
    daemon:{},
    snapshot:()=>({devices}),
    renameDevice(id,name){ calls.push('rename'); devices = devices.map(d=>d.id===id?{...d,name}:d); },
    async stopDaemon(){ calls.push('stop'); this.daemon=null; },
    unbindDevice(id){ calls.push('unbind'); devices=devices.filter(d=>d.id!==id); },
    async sync(){ calls.push('sync'); this.daemon={}; },
    async rotateToken(){ calls.push('rotate'); },
  };
  const hooks = {hideOffer:()=>calls.push('hide'), revokeGit:()=>calls.push('revoke'), applied:()=>calls.push('applied')};
  assert.deepEqual(Object.keys(context.publicPhoneDevices(remote.snapshot())[0]), ['id','name','createdAt','lastSeenAt']);
  await context.managePhoneDevice(remote,{operation:'rename',deviceId:'device-0001',name:' New name '},hooks);
  assert.equal(devices[0].name,'New name');
  await assert.rejects(context.managePhoneDevice(remote,{operation:'rename',deviceId:'device-0001',name:' '},hooks));
  await assert.rejects(context.managePhoneDevice(remote,{operation:'unbind',deviceId:'missing'},hooks));
  calls.length=0;
  await context.managePhoneDevice(remote,{operation:'rotate'},hooks);
  assert.deepEqual(calls,['hide','rotate']);
  assert.equal(devices.length,1);
  calls.length=0;
  await context.managePhoneDevice(remote,{operation:'unbind',deviceId:'device-0001'},hooks);
  assert.deepEqual(calls,['hide','revoke','stop','unbind','applied','sync']);
  assert.equal(devices.length,0);
  devices=[{id:'device-0002',name:'Second'}];
  remote.stopDaemon=async()=>{throw Error('fictional stop failure');};
  calls.length=0;
  await assert.rejects(context.managePhoneDevice(remote,{operation:'unbind',deviceId:'device-0002'},hooks));
  assert(!calls.includes('unbind'));
  assert.equal(devices.length,1);
  console.log('Phone device adapter fixture passed');
}
run().catch(error=>{console.error(error);process.exitCode=1;});
