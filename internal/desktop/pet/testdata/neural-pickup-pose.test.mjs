import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush} from './body-fixture.mjs';

const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
const seeded=seed=>()=>{seed=(Math.imul(seed,1664525)+1013904223)>>>0;return seed/4294967296;};
function reference(clock,seed){
 const math=Object.create(Math);math.random=seeded(seed);
 const scope={Math:math,Float32Array,performance:{now:()=>clock.now},TAP_REACTION_MS:1400,dragging:false,dragMoved:false,physVel:{x:0,y:0},drawPos:{x:0},rigT:clock.now/1000,rigStateT0:clock.now/1000,rigSwing:.3,rigSwingV:0,stillCtl:{name:null,alpha:0,target:0},session:{},LIVE_ENTRY:{},RIG_ENTRY:{},rig:{ready:false},sleeping:false,say(){},action:null,STILL_FADE:7,lastFullClear:0,fx:{},landT:-1e9,throwT:-1e9,thrown:false,feed:null,come:null,panel:null,pendingLevelUp:false,pointer:{x:0,y:0},overPet:()=>false,spawn(){},petW:()=>240,petH:()=>260};vm.createContext(scope);
 const core=source.slice(source.indexOf('const pose = new Float32Array(45);'),source.indexOf('// ── ONNX runtime'));
 const programs=source.slice(source.indexOf('const liveFx = {'),source.indexOf('// ── live-mode draw'));
 const states=source.slice(source.indexOf('function setStill(name)'),source.indexOf('// ── HD part rig'));
 const fade=source.slice(source.indexOf('function tickStill(now)'),source.indexOf('  // Come-here:',source.indexOf('function tickStill(now)')))+'\n}';
 vm.runInContext(core+'\n'+programs+'\n'+states+'\n'+fade+'\nglobalThis.channels=pose;globalThis.stateFx=liveFx;',scope);
 let mode='';return {scope,mode(next){if((next==='pickup'||next==='throw')&&mode!=='pickup'&&mode!=='throw')vm.runInContext("setStill('pick-up')",scope);if(next!== 'pickup'&&next!=='throw'&&(mode==='pickup'||mode==='throw')){vm.runInContext(mode==='throw'?"playStill('recover',1100)":"clearStill()",scope);}mode=next;scope.dragging=scope.dragMoved=next==='pickup';scope.thrown=next==='throw';},frame(at){scope.tickStill(at);scope.rigT=at/1000;vm.runInContext('stepPose()',scope);return {pose:Array.from(scope.channels),phase:scope.stillCtl.name,alpha:scope.stillCtl.alpha,rotation:scope.stateFx.rot,scaleY:scope.stateFx.sy,swing:scope.rigSwing,velocity:scope.rigSwingV};}};
}
async function setup(seed=311){
 const f=fixture({sleepEnabled:true});f.scope.Math.random=seeded(seed);f.setInteraction({CursorX:240,CursorY:237.2});f.load('pet-interaction.js');const interaction=f.window.WhalePetInteraction.create({body:f.body(),getSettings:()=>({}),isSleeping:()=>false}),driver=interaction.useNeuralRest();await flush();const ref=reference(f,seed),pose=new Float32Array(45);let previousMode='';
 return {f,interaction,driver,ref,async mode(mode,vx=0){f.setInteraction({Mode:mode,VX:vx,VY:0});await f.fire(50);ref.mode(mode);ref.scope.physVel.x=vx;previousMode=mode;},frame(at){f.now=at;const rest=Object.freeze({phase:null,alpha:0,a:0,time:at/1000,started:0,elapsed:0,revision:0,sleeping:false});driver.frame(rest);interaction.pose(pose);const expected=ref.frame(at),actual=driver.pickup;for(let c=0;c<45;c++)assert.ok(Math.abs(pose[c]-expected.pose[c])<2e-7,`mode=${previousMode} phase=${actual.phase} t=${at} ch${c} ${pose[c]} != ${expected.pose[c]}`);assert.equal(actual.phase,expected.phase);assert.ok(Math.abs(actual.alpha-expected.alpha)<1e-12);if(actual.phase){assert.ok(Math.abs(actual.rotation-expected.rotation)<1e-12);assert.ok(Math.abs(actual.scaleY-expected.scaleY)<1e-12);assert.ok(Math.abs(actual.swing-expected.swing)<1e-12);}return {pose:Array.from(pose),state:actual};},close(){driver.close();interaction.close();}};
}

test('actual reference pickup/throw/release/recover compares45 channels and spring perRAF for dualseeds and30/60/120/144Hz',async()=>{
 for(const hz of [30,60,120,144])for(const seed of [311,911]){
  const p=await setup(seed);let now=p.f.now;for(let i=0;i<6;i++)p.frame(now+=1000/hz);
  await p.mode('pickup',80);let sample;for(let i=0;i<hz*2;i++)sample=p.frame(now+=1000/hz);assert.ok(Math.abs(sample.pose[12]-.85)<1e-7);assert.ok(Math.abs(sample.pose[2]-1)<1e-7);assert.ok(Math.abs(sample.pose[26]-.1)<1e-7||Math.abs(sample.pose[26]-.9)<1e-7);
  const started=sample.state.started;await p.mode('throw',400);for(let i=0;i<hz*2;i++)sample=p.frame(now+=1000/hz);assert.equal(sample.state.started,started,'throw retains pickup start and expression phase');
  await p.mode('');for(let i=0;i<hz*2;i++)sample=p.frame(now+=1000/hz);assert.equal(sample.state.phase,null);
  await p.mode('pickup',0);for(let i=0;i<hz;i++)p.frame(now+=1000/hz);await p.mode('');for(let i=0;i<30;i++)sample=p.frame(now+=1000/hz);assert.equal(sample.state.phase,null);p.close();
 }
});

test('held base wins over early surprise before eye isolation; true pick-up key has no panicked alias expression',async()=>{
 const p=await setup();await p.mode('pickup');let sample;for(let i=0;i<20;i++)sample=p.frame(p.f.now+17);assert.equal(sample.pose[12],1);assert.equal(sample.pose[16],0,'reference base-held closure then isolation suppresses wide-eye channel');
 for(let i=0;i<80;i++)sample=p.frame(p.f.now+17);assert.ok(Math.abs(sample.pose[2]-1)<1e-7,'actual key keeps program brow1 rather than alias expression.75');assert.ok(sample.pose[26]===Math.fround(.1)||sample.pose[26]===Math.fround(.9));p.close();
});

test('real runtime pickup bypasses stale care pose and keeps model input fixed while busy; rest has priority',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();f.setTask({Care:{Action:{ID:f.scope.Date.now(),Kind:'pat',Level:1}}});await f.fire(1500);f.setInteraction({Mode:'pickup',VX:0});await f.fire(50);
 for(let i=0;i<25;i++)await f.frame(f.now+50);const pose=f.neuralInputs.at(-1).before;assert.ok(Math.abs(pose[12]-.85)<1e-7);assert.ok(pose[26]===Math.fround(.1)||pose[26]===Math.fround(.9));assert.ok(Math.abs(pose[14])<1e-7,'care happyeye must not override isolated pickup');
 const gate=f.gateFrame(),inflight=f.frame(f.now+50);await flush();const input=f.neuralInputs.at(-1);for(let i=0;i<6;i++)await f.frame(f.now+17);assert.deepEqual(Array.from(input.value),input.before);gate.resolve();await inflight;
 f.setTask({Rest:{Sleeping:true}});await f.fire(1500);for(let i=0;i<35;i++)await f.frame(f.now+50);const sleep=f.neuralInputs.at(-1).before;assert.equal(sleep[12],1);assert.equal(sleep[13],1);assert.ok(Math.abs(sleep[39]-.16)<1e-7);f.close();assert.equal(f.rafs.size,0);
});
