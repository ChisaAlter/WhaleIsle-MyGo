import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush,plain} from './body-fixture.mjs';
const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
const opaque=()=>{const p=new Uint8ClampedArray(512*512*4);for(let y=40;y<=480;y++)for(let x=80;x<=430;x++)p[(y*512+x)*4+3]=255;return p;};
async function runtime(){const f=fixture({sleepEnabled:true}),trace={prepare:[],present:[],cached:[],commit:[],contents:[]},originals={};
 Object.defineProperty(f.scope,'WhalePetSuperResolution',{configurable:true,get:()=>originals.sr,set:value=>{const next=value.create;originals.sr={...value,create(...args){const api=next(...args);return {...api,prepare(image){trace.prepare.push({at:f.now,image});return api.prepare(image);},present(target,facing){trace.present.push({at:f.now,facing});return api.present(target,facing);}};}};}});
 Object.defineProperty(f.window,'WhalePetSleep',{configurable:true,get:()=>originals.sleep,set:value=>{const next=value.create;originals.sleep={...value,create(...args){const c=next(...args);trace.controller=c;return {...c,get active(){return c.active;},get pickupActive(){return c.pickupActive;},get evidence(){return c.evidence;},presentCommitted(content,token,input,paint){trace.contents.push(content);const ok=c.presentCommitted(content,token,input,paint);trace.cached.push({at:f.now,ok,content,token,input});return ok;},commitRest(...args){const ok=c.commitRest(...args);trace.commit.push({at:f.now,ok});return ok;},commitNeuralBody(...args){const ok=c.commitNeuralBody(...args);trace.commit.push({at:f.now,ok});return ok;}};}};}});
 await f.start();await f.fire(1500);return {f,trace};}

test('cached sleep and pickup body transforms use current RAF with accepted pixels while successful-frame alpha counts stay unchanged',()=>{
 const f=fixture({sleepEnabled:true});f.load('pet-sleep.js');const layout={X:120,Y:180,Width:240,Height:260},api=f.window.WhalePetSleep,c=api.create({canvas:f.node('sprite'),body:f.body(),getLayout:()=>layout}),pixels=opaque();let paints=0;
 c.request(true);c.tick(f.now);const token=c.snapshot({facing:1});assert.equal(c.commit(pixels,token,{},()=>paints++),true);const content={kind:'rest',token,modelPose:Object.freeze(Array(45).fill(0))},frames=c.frames;
 for(let i=0;i<180;i++){f.now+=1000/60;c.tick(f.now);const current=c.snapshot({facing:1});assert.equal(c.presentCommitted(content,current,{},()=>paints++),true);const expected=api.geometry(c.evidence.box,layout,1,current.angle,current.dy,current.sy);assert.deepEqual(plain(f.body().hitRect(f.body().target)),plain(expected.rect));assert.equal(c.frames,frames);}
 const before=paints;c.request(false);c.tick(f.now+10);assert.equal(c.presentCommitted(content,c.snapshot({}),{},()=>paints++),false);assert.equal(paints,before,'old sleep content cannot cross wake state');c.close();
});

test('SR prepares the CNN once per new image and presents cached RGB/alpha at display rate with facing and context-loss restoration',()=>{
 const f=fixture();let uploads=0;class Upscaler{static isSupported(){return true;}attachSource(input,gl){gl.width=input.width*2;gl.height=input.height*2;}upscale(){uploads++;}detachSource(){}}
 f.load('pet-super-resolution.js');const sr=f.scope.WhalePetSuperResolution.create({ImageUpscaler:Upscaler,ANIME4KJS_SIMPLE_M_2X:'fixed'}),source=f.document.createElement('canvas'),canvas=f.node('sprite');canvas.width=780;canvas.height=984;sr.prepare(source);
 for(let i=0;i<90;i++)sr.present(canvas.ctx,i%2?-1:1);assert.equal(uploads,1);assert.equal(f.draw.filter(r=>r[0]==='sprite'&&r[1]==='draw').length,90);assert.deepEqual(f.draw.filter(r=>r[0]==='sprite'&&r[1]==='draw').slice(0,2).map(r=>r.at(-1).map(v=>v||0)),[[1,0,0,1,0,0],[-1,0,0,1,780,0]]);sr.prepare(source);assert.equal(uploads,2);sr.close();const count=f.draw.length;sr.present(canvas.ctx);assert.equal(f.draw.length,count);
});

test('cached sleep reflection stays with its content while a facing change awaits the next inference',async()=>{
 const {f,trace}=await runtime();f.setInteraction({Facing:1});await f.fire(50);f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();await f.frame(f.now+50);const before=trace.cached.at(-1).content;f.setInteraction({Facing:-1});await f.fire(50);await f.frame(f.now+17);const row=trace.cached.at(-1);assert.equal(row.ok,true);assert.equal(row.token.facing,1);assert.equal(row.content,before);assert.equal(trace.present.at(-1).facing,1);await f.frame(f.now+50);await f.frame(f.now+17);assert.equal(trace.cached.at(-1).token.facing,-1);assert.equal(trace.present.at(-1).facing,-1);f.close();
});

test('real runtime presents cached content every visible RAF without changing the in-flight model input or rerunning SR',async()=>{
 const {f,trace}=await runtime();f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();await f.frame(f.now+50);const start=f.now,gate=f.gateFrame(),pending=f.frame(start+50);await flush();const input=f.neuralInputs.at(-1),saved=input.before.slice(),prepared=trace.prepare.length,count=f.neuralInputs.length,contents=trace.contents.at(-1),matrix=f.node('sprite').style.transform;
 for(let i=1;i<=18;i++)await f.frame(start+50+i*1000/60);
 assert.equal(f.neuralInputs.length,count);assert.equal(trace.prepare.length,prepared);assert.deepEqual(Array.from(input.value),saved);assert.ok(trace.cached.slice(-18).every(row=>row.ok));assert.ok(trace.cached.slice(-18).every(row=>row.content===contents));assert.notEqual(f.node('sprite').style.transform,matrix);assert.equal(Object.isFrozen(contents.modelPose),true);gate.resolve();await pending;assert.deepEqual(input.after,saved);assert.equal(trace.prepare.length,prepared+1);f.close();
});

test('reference candidate clock consumes busy slots at50/110ms and runtime starts match without immediate busy catch-up',async()=>{
 for(const gap of [50,110]){
  const {f,trace}=await runtime();f.scope.Math.random=()=>.2;
  // Exercise the real power-save clock only; settings and inactivity source
  // are unchanged, so110ms requires5 minutes without interaction.
  if(gap===110){f.now+=300001;await f.frame(f.now);}else await f.frame(f.now+50);
  const start=f.now,lastStart=f.neuralInputs.length;
  let busy=false,last=start;
  const gate=f.gateFrame();const pending=f.frame(start+gap);await flush();busy=true;last=start+gap;
  const points=[start+gap+17,start+gap+50,start+gap+100,start+gap+117,start+gap+137,start+gap+149,start+gap+150,start+gap+200,start+gap+220];
  const scope={inferBusy:true,session:{},petPerf:{record(){}},performance:{now:()=>0},calls:0};vm.createContext(scope);
  const body=source.slice(source.indexOf('async function renderFrame()'),source.indexOf('  const perfStart',source.indexOf('async function renderFrame()')));vm.runInContext(body+'calls++;inferBusy=true;}',scope);
  const slot=source.slice(source.indexOf('        if (now - lastFrameAt >= gap)'),source.indexOf('        paint();',source.indexOf('        if (now - lastFrameAt >= gap)')));vm.runInContext('let lastFrameAt='+last+';globalThis.slot=(now,gap)=>{'+slot+'};',scope);
  for(const at of points){if(at>=start+gap+137&&busy){gate.resolve();await pending;busy=false;scope.inferBusy=false;}const before=f.neuralInputs.length;scope.slot(at,gap);await f.frame(at);if(f.neuralInputs.length>before){scope.inferBusy=false;}assert.equal(f.neuralInputs.length-lastStart,1+scope.calls);}
  assert.ok(trace.present.length>trace.prepare.length);f.close();
 }
});

test('cached old mode/capture and pending first transparent frame cannot steal a new body kind or replace alpha bounds',async()=>{
 const {f,trace}=await runtime();f.setInteraction({Mode:'pickup',CursorX:250,CursorY:220});await f.fire(50);await f.frame(f.now+50);const c=trace.controller,oldContent=trace.contents.at(-1),box=plain(c.pickupEvidence.box);
 f.setPixels(new Uint8ClampedArray(512*512*4));const prepares=trace.prepare.length;await f.frame(f.now+50);assert.equal(trace.prepare.length,prepares);assert.deepEqual(plain(c.pickupEvidence.box),box);
 f.setInteraction({Mode:'throw',VX:200});await f.fire(50);const matrix=f.node('sprite').style.transform;await f.frame(f.now+17);assert.equal(f.node('sprite').style.transform,matrix);assert.equal(trace.cached.at(-1).ok,false,'old mode content cannot republish after mode epoch changed');
 f.emit(f.node('sprite'),'pointerdown',{pointerId:23});await f.frame(f.now+17);assert.equal(trace.cached.at(-1).ok,false);f.close();
});
