import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush,plain} from './body-fixture.mjs';

const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
const seeded=seed=>()=>{seed=(Math.imul(seed,1664525)+1013904223)>>>0;return seed/4294967296;};
function original(clock,seed){
 const math=Object.create(Math);math.random=seeded(seed);
 const scope={Math:math,Float32Array,performance:{now:()=>clock.now},TAP_REACTION_MS:1400,dragging:false,dragMoved:false,physVel:{x:0,y:0},drawPos:{x:0},rigT:clock.now/1000,rigStateT0:clock.now/1000,stillCtl:{name:null,alpha:0,target:0},session:{},LIVE_ENTRY:{live:true},RIG_ENTRY:{rig:true},rig:{ready:false},sleeping:false,say(){},action:null};
 vm.createContext(scope);
 const core=source.slice(source.indexOf('const pose = new Float32Array(45);'),source.indexOf('// ── ONNX runtime'));
 const programs=source.slice(source.indexOf('const liveFx = {'),source.indexOf('// ── live-mode draw'));
 const state=source.slice(source.indexOf('function setStill(name)'),source.indexOf('// ── HD part rig'));
 const rest=source.slice(source.indexOf('function sleepEnter()'),source.indexOf('// Anchor: still'));
 vm.runInContext(core+'\n'+programs+'\n'+state+'\n'+rest+'\nglobalThis.channels=pose;globalThis.idleState=idle;',scope);
 const ticks=source.slice(source.indexOf('function tickStill(now)'),source.indexOf('  // Come-here:',source.indexOf('function tickStill(now)')))+'\n}';
 Object.assign(scope,{STILL_FADE:7,lastFullClear:0,fx:{},landT:-1e9,throwT:-1e9,thrown:false,feed:null,come:null,panel:null,pendingLevelUp:false,pointer:{x:0,y:0},overPet:()=>false,spawn(){},petW:()=>240,petH:()=>260});vm.runInContext(ticks,scope);
 return {scope,tick(at){scope.tickStill(at);scope.rigT=at/1000;return {phase:scope.stillCtl.name,alpha:scope.stillCtl.alpha};},set(name){vm.runInContext(`setStill('${name}')`,scope);},enter(){vm.runInContext('sleepEnter()',scope);},wake(){vm.runInContext('wake()',scope);},render(frame){scope.rigT=frame.time;scope.rigStateT0=frame.started;scope.stillCtl.name=frame.phase;scope.stillCtl.alpha=frame.alpha;vm.runInContext('stepPose()',scope);return Array.from(scope.channels);}};
}
async function pair(seed=177){
 const f=fixture({sleepEnabled:true});f.scope.Math.random=seeded(seed);f.setInteraction({CursorX:240,CursorY:237.2});f.load('pet-interaction.js');f.load('pet-sleep.js');
 const inter=f.window.WhalePetInteraction.create({body:f.body(),getSettings:()=>({}),isSleeping:()=>false}),driver=inter.useNeuralRest();await flush();
 const sleep=f.window.WhalePetSleep.create({canvas:f.node('sprite'),body:f.body(),getLayout:()=>({X:120,Y:180,Width:240,Height:260})});
 const ref=original(f,seed);const pose=new Float32Array(45);let samples=0;
 function compare(at){f.now=at;sleep.tick(at);const token=sleep.snapshot({}),frame=token.rest;assert.equal(Object.isFrozen(frame),true);driver.frame(frame);inter.pose(pose);const expected=ref.render(frame);for(let channel=0;channel<45;channel++)assert.ok(Math.abs(pose[channel]-expected[channel])<2e-7,`sample${samples} phase${frame.phase} time${at} channel${channel}: ${pose[channel]} != ${expected[channel]}`);samples++;return {token,pose:Array.from(pose)};}
 return {f,sleep,inter,driver,ref,compare,close(){driver.close();inter.close();sleep.close();}};
}

test('real reference base, state, expression and final eye-isolation match all45 channels per RAF at30/60/120/144Hz',async()=>{
 for(const hz of [30,60,120,144])for(const seed of [177,905]){
  const p=await pair(seed);let at=p.f.now;
  for(let i=0;i<8;i++)p.compare(at+=1000/hz);
  p.sleep.doze(true);for(let i=0;i<7;i++)p.compare(at+=1000/hz);
  p.sleep.request(true);p.sleep.doze(false);p.ref.enter();let stable;
  for(let i=0;i<hz*4;i++)stable=p.compare(at+=1000/hz);
  assert.equal(stable.token.rest.phase,'sleep');assert.ok(Math.abs(stable.pose[39]-.16)<1e-7);assert.equal(stable.pose[12],1);assert.equal(stable.pose[13],1);for(let c=0;c<39;c++)if(c!==12&&c!==13)assert.ok(Math.abs(stable.pose[c])<1e-7);
  p.sleep.request(false);p.ref.wake();for(let i=0;i<hz*2;i++)p.compare(at+=1000/hz);p.close();
 }
});

test('sleep breathing remains live and has original1.5 amplitude without cumulative multiplication or a new frequency',async()=>{
 const p=await pair(922);let at=p.f.now;p.sleep.request(true);p.ref.enter();const values=[];
 for(let i=0;i<480;i++){const sample=p.compare(at+=1000/60);if(i>120)values.push(sample.pose[44]);}
 assert.ok(Math.min(...values)<-.45);assert.ok(Math.max(...values)>.45);assert.ok(Math.max(...values)<=.600001);assert.ok(Math.min(...values)>=-.600001);p.close();
});

test('rest metadata keeps source wake expression through envelope fade; reads do not advance the body clock',async()=>{
 const p=await pair();let at=p.f.now;p.sleep.request(true);p.ref.enter();for(let i=0;i<80;i++)p.compare(at+=17);p.sleep.request(false);p.ref.wake();
 let phase;for(let i=0;i<64;i++)phase=p.compare(at+=17).token.rest.phase;assert.equal(phase,'wake');
 const before=p.sleep.snapshot({});for(let i=0;i<20;i++)assert.deepEqual(plain(p.sleep.snapshot({}).rest),plain(before.rest));
 const faded=p.compare(at+=34);assert.equal(faded.token.waking,false);assert.equal(faded.token.rest.phase,'wake');assert.ok(faded.token.rest.alpha>0);
 for(let i=0;i<20;i++)phase=p.compare(at+=17).token.rest.phase;assert.equal(phase,null);p.close();
});

test('independent original tickStill confirms the same sleep and wake-expiry trajectory through final alpha zero',async()=>{
 for(const hz of [30,60,120,144]){
  const p=await pair(),r=p.ref;let at=p.f.now;p.sleep.request(true);r.enter();
  for(let i=0;i<hz*3;i++){at+=1000/hz;p.f.now=at;p.sleep.tick(at);const rest=p.sleep.snapshot({}).rest,expected=r.tick(at);assert.ok(Math.abs(rest.alpha-expected.alpha)<1e-12);assert.equal(rest.phase,expected.phase);}
  p.sleep.request(false);r.wake();
  for(let i=0;i<hz*2;i++){at+=1000/hz;p.f.now=at;p.sleep.tick(at);const rest=p.sleep.snapshot({}).rest,expected=r.tick(at);assert.ok(Math.abs(rest.alpha-expected.alpha)<1e-12);assert.equal(rest.phase,expected.phase);}
  assert.equal(p.sleep.snapshot({}).rest.alpha,0);p.close();
 }
});

test('stageII270-frame capture, alpha pairing and layout projection retain the frozen fixture with the explicit absolute feet correction',()=>{
 const f=fixture({sleepEnabled:true});f.load('pet-sleep.js');const current=f.window.WhalePetSleep;
 const priorText=readFileSync(new URL('./fixtures/pet-sleep-stage2.js',import.meta.url),'utf8');vm.runInContext(priorText,f.context);const prior=f.window.WhalePetSleep;
 function make(api){const canvas=f.scope.document.createElement('canvas'),target=f.scope.document.createElement('div'),body=f.window.WhalePetBody.create({characters:[canvas],target});let layout={X:120,Y:180,Width:240,Height:260},paints=0;return {canvas,target,body,sleep:api.create({canvas,body,getLayout:()=>layout}),paint:()=>paints++,layout(next){layout=next;},observe(){return plain({style:canvas.style,target:target.style,hidden:target.hidden,hit:body.hitRect(target),frames:this.sleep.frames,paints});}};}
 const a=make(current),b=make(prior),pixels=new Uint8ClampedArray(512*512*4);for(let y=4;y<510;y+=2)for(let x=2;x<508;x+=2)pixels[(y*512+x)*4+3]=255;
 for(const each of [a,b])each.sleep.request(true);
 for(let i=0;i<270;i++){f.now+=17;for(const each of [a,b]){if(i===8)each.body.capture(each.canvas,1,true);if(i===13)each.body.capture(each.canvas,1,false);if(i===80)each.sleep.request(false);if(i===90)each.sleep.request(true);if(i===170){each.layout({X:180,Y:240,Width:384,Height:416});each.sleep.reproject();}each.sleep.tick(f.now);each.sleep.commit(i%31===0?new Uint8ClampedArray(pixels.length):pixels,each.sleep.snapshot({facing:i%2?-1:1}),{},each.paint);}const actual=a.observe(),legacy=b.observe();if(actual.hit){legacy.hit.y-=2;legacy.hit.top-=2;legacy.hit.bottom-=2;legacy.target.top=(parseFloat(legacy.target.top)-2)+'px';const matrix=legacy.style.transform.slice(7,-1).split(',').map(Number);matrix[5]-=2;legacy.style.transform='matrix('+matrix.join(',')+')';}function compare(actual,legacy,label=''){if(typeof actual==='number'&&typeof legacy==='number'){assert.ok(Math.abs(actual-legacy)<1e-10,label);return;}if(typeof actual==='string'&&typeof legacy==='string'&&(actual.endsWith('px')||actual.startsWith('matrix('))){const numbers=value=>value.match(/-?\d+(?:\.\d+)?(?:e[+-]?\d+)?/gi)?.map(Number)||[];assert.equal(numbers(actual).length,numbers(legacy).length);numbers(actual).forEach((value,index)=>assert.ok(Math.abs(value-numbers(legacy)[index])<1e-10,label));return;}if(actual&&legacy&&typeof actual==='object'&&typeof legacy==='object'){assert.deepEqual(Object.keys(actual),Object.keys(legacy));for(const key of Object.keys(actual))compare(actual[key],legacy[key],label+'.'+key);return;}assert.deepEqual(actual,legacy,label);}compare(actual,legacy);}
 a.sleep.close();b.sleep.close();
});

test('hidden, delayed open and fast sleep-wake transitions cannot leave the private pose driver attached after failure/close',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();const count=f.neuralInputs.length;f.document.hidden=true;for(let i=0;i<10;i++)await f.frame(f.now+50);assert.equal(f.neuralInputs.length,count);f.setTask({Rest:{Sleeping:false}});await f.fire(1500);f.setTask({Rest:{Sleeping:true}});await f.fire(1500);f.document.hidden=false;await f.frame(f.now+50);assert.equal(f.neuralInputs.length,count+1);assert.ok(f.neuralInputs.at(-1).before.every(Number.isFinite));f.close();const closed=f.neuralInputs.length;await f.frame(f.now+100);assert.equal(f.neuralInputs.length,closed);assert.equal(f.rafs.size,0);
});

test('private perRAF pose never mutates an in-flight model input; detach restores sleeping fallback care values',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();
 const gate=f.gateFrame(),pending=f.frame(f.now+50);await flush();const captured=f.neuralInputs.at(-1),before=captured.before.slice();
 for(let i=0;i<9;i++)await f.frame(f.now+17);assert.deepEqual(Array.from(captured.value),before,'inference input is a snapshot, not the private RAF pose');gate.resolve();await pending;assert.deepEqual(captured.after,before);
 await f.frame(f.now+50);assert.notEqual(f.neuralInputs.at(-1).before[44],before[44]);
 f.failFrame();await f.frame(f.now+50);assert.equal(f.document.body.dataset.renderer,'rig');assert.equal(f.node('sprite').style.rotate,'18deg');assert.equal(f.node('sprite').style.scale,'1 .82');f.close();
});
