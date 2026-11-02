import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush,plain} from './body-fixture.mjs';
const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
const seeded=seed=>()=>{seed=(Math.imul(seed,1664525)+1013904223)>>>0;return seed/4294967296;};
function drawReference(layout,frame){
 let matrix=[1,0,0,1,0,0],drawn;const multiply=([a,b,c,d,e,f])=>{const [x,y,z,w,u,v]=matrix;matrix=[x*a+z*b,y*a+w*b,x*c+z*d,y*c+w*d,x*e+z*f+u,y*e+w*f+v];};
 const ctx2d={save(){},restore(){},translate(x,y){multiply([1,0,0,1,x,y]);},rotate(a){multiply([Math.cos(a),Math.sin(a),-Math.sin(a),Math.cos(a),0,0]);},scale(x,y){multiply([x,0,0,y,0,0]);},drawImage(_src,x,y){multiply([1,0,0,1,x,y]);drawn=matrix.slice();}};
 const scope={Math,ctx2d,rigT:frame.time||0,settings:{opacity:1},srReady:false,outCanvas:{},stillCtl:{alpha:1},smooth:x=>x*x*(3-2*x),clamp01:x=>Math.max(0,Math.min(1,x)),fx:{rot:0,dx:0,dy:0,sx:1,sy:1},liveFx:{rot:frame.angle,dx:0,dy:frame.dy,sx:1,sy:frame.sy,pivot:null,stars:false},dragging:false,thrown:false,physPoint:null,petW:()=>layout.Width,petH:()=>layout.Height,drawPos:{x:layout.X,y:layout.Y},facing:frame.facing,GRAB_FRAC:.085};
 vm.runInNewContext(source.slice(source.indexOf('function drawLive()'),source.indexOf('function drawRigPart(')),scope);scope.drawLive();return drawn;
}
function original(clock,seed){
 const math=Object.create(Math);math.random=seeded(seed);
 const scope={Math:math,Float32Array,performance:{now:()=>clock.now},TAP_REACTION_MS:1400,dragging:false,dragMoved:false,physVel:{x:0,y:0},drawPos:{x:0},rigT:clock.now/1000,rigStateT0:clock.now/1000,stillCtl:{name:null,alpha:0,target:0},session:{},LIVE_ENTRY:{live:true},RIG_ENTRY:{rig:true},rig:{ready:false},sleeping:false,say(){},action:null};vm.createContext(scope);
 const core=source.slice(source.indexOf('const pose = new Float32Array(45);'),source.indexOf('// ── ONNX runtime'));
 const programs=source.slice(source.indexOf('const liveFx = {'),source.indexOf('// ── live-mode draw'));
 const state=source.slice(source.indexOf('function setStill(name)'),source.indexOf('// ── HD part rig'));
 const rest=source.slice(source.indexOf('function sleepEnter()'),source.indexOf('// Anchor: still'));
 vm.runInContext(core+'\n'+programs+'\n'+state+'\n'+rest+'\nglobalThis.channels=pose;',scope);
 const ticks=source.slice(source.indexOf('function tickStill(now)'),source.indexOf('  // Come-here:',source.indexOf('function tickStill(now)')))+'\n}';
 Object.assign(scope,{STILL_FADE:7,lastFullClear:0,fx:{},landT:-1e9,throwT:-1e9,thrown:false,feed:null,come:null,panel:null,pendingLevelUp:false,pointer:{x:0,y:0},overPet:()=>false,spawn(){},petW:()=>240,petH:()=>260});vm.runInContext(ticks,scope);
 return {scope,enter(){scope.sleepEnter();},wake(){scope.wake();},tick(now,started){scope.tickStill(now);scope.rigT=now/1000;if(started!==undefined)scope.rigStateT0=started;scope.stepPose();return {alpha:scope.stillCtl.alpha,phase:scope.stillCtl.name,pose:Array.from(scope.channels)};}};
}

test('absolute drawLive matrix, tight alpha corners and head match the real feet h-2 origin without pivot normalization at every native scale',()=>{
 const f=fixture({sleepEnabled:true});f.load('pet-sleep.js');const api=f.window.WhalePetSleep,box={x:84,y:36,right:426,bottom:480};
 for(const layout of [{X:120,Y:180,Width:240,Height:260},{X:-96,Y:64,Width:384,Height:416},{X:13.5,Y:-44.25,Width:153.6,Height:166.4}])for(const facing of [-1,1])for(const angle of [0,-.2,-1])for(const sy of [1,1.04]){
  const frame={facing,angle,dy:angle?-7:14,sy},actual=api.geometry(box,layout,facing,angle,frame.dy,sy),expected=drawReference(layout,frame),[a,b,c,d,e,g]=expected;
  // The source reflection is already in the visible canvas. Compose that
  // exact reflection into CSS only for an absolute source-matrix comparison.
  const css=Array.from(actual.matrix);css[4]+=layout.X;css[5]+=layout.Y;if(facing<0){css[4]+=css[0]*layout.Width;css[5]+=css[1]*layout.Width;css[0]*=-1;css[1]*=-1;}
  for(let i=0;i<6;i++)assert.ok(Math.abs(css[i]-expected[i])<1e-10);
  const point=(x,y)=>{const px=(x-60)*layout.Width/390,py=(y-20)*layout.Height/492;return{x:a*px+c*py+e,y:b*px+d*py+g};};
  const corners=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(p=>point(...p)),rect={x:Math.min(...corners.map(p=>p.x)),y:Math.min(...corners.map(p=>p.y)),right:Math.max(...corners.map(p=>p.x)),bottom:Math.max(...corners.map(p=>p.y))};for(const k of ['x','y','right','bottom'])assert.ok(Math.abs(actual.rect[k]-rect[k])<1e-10);
  const head=point((box.x+box.right)/2,box.y);assert.ok(Math.abs(actual.head.x-head.x)<1e-10);assert.ok(Math.abs(actual.head.y-head.y)<1e-10);
  for(const scale of [1,1.25,1.5,2]){const nativeOrigin={x:-2560,y:144},native=value=>({x:nativeOrigin.x+value.x*scale,y:nativeOrigin.y+value.y*scale});assert.ok(Math.abs(native(actual.head).x-native(head).x)<1e-9);assert.ok(Math.abs(native(actual.head).y-native(head).y)<1e-9);}
 }
});

test('normal CSS feet baseline stays exactly-2 through busy/transparent output and sleep-wake handoff without changing canvas capture or normal hit dimensions',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);const canvas=f.node('sprite'),body=f.body(),before=plain(body.hitRect(canvas));assert.equal(canvas.style.translate,'0px -2px');assert.equal(body.managed(canvas),false);assert.equal(body.target.hidden,true);
 f.emit(canvas,'pointerdown',{pointerId:21});const gate=f.gateFrame(),pending=f.frame();await flush();for(let i=0;i<4;i++)await f.frame(f.now+17);assert.equal(canvas.style.translate,'0px -2px');assert.equal(canvas.captured.has(21),true);assert.equal(body.target.hidden,true);assert.deepEqual(plain(body.hitRect(canvas)),before,'browser boundary box dimensions are unchanged; actualCSStranslation is checked in ownedGUI');gate.resolve();await pending;f.emit(canvas,'pointerup',{pointerId:21});
 f.setPixels(new Uint8ClampedArray(512*512*4));await f.frame();assert.equal(canvas.style.translate,'0px -2px');f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();assert.equal(body.target.hidden,true);assert.equal(canvas.style.translate,'0px -2px');const pixels=new Uint8ClampedArray(512*512*4);for(let y=40;y<480;y++)for(let x=80;x<430;x++)pixels[(y*512+x)*4+3]=255;f.setPixels(pixels);await f.frame();assert.equal(canvas.style.translate,'');assert.equal(body.target.hidden,false);
 f.setTask({Rest:{Sleeping:false}});await f.fire(1500);await f.frame(f.now+1200);assert.equal(body.target.hidden,true);assert.equal(canvas.style.translate,'0px -2px');f.failFrame();await f.frame(f.now+50);await flush();assert.equal(f.document.body.dataset.renderer,'rig');assert.equal(canvas.style.translate,'');f.close();
});

test('wake at1100ms keeps its target through that frame; strict post-integration expiry and all45 channels match actual tickStill until alpha0',async()=>{
 for(const hz of [30,60,120,144])for(const seed of [177,905]){
  const f=fixture({sleepEnabled:true});f.scope.Math.random=seeded(seed);f.setInteraction({CursorX:240,CursorY:237.2});f.load('pet-interaction.js');f.load('pet-sleep.js');const inter=f.window.WhalePetInteraction.create({body:f.body(),getSettings:()=>({}),isSleeping:()=>false}),driver=inter.useNeuralRest(),sleep=f.window.WhalePetSleep.create({canvas:f.node('sprite'),body:f.body(),getLayout:()=>({X:120,Y:180,Width:240,Height:260})});await flush();const ref=original(f,seed),pose=new Float32Array(45);
  sleep.request(true);ref.enter();for(let i=0;i<hz*2;i++){f.now+=1000/hz;sleep.tick(f.now);driver.frame(sleep.snapshot({}).rest);ref.tick(f.now);}
  sleep.request(false);ref.wake();const start=f.now,points=[start+1,start+1099.999,start+1100,start+1100.001,...Array.from({length:hz},(_,i)=>start+1100.001+(i+1)*1000/hz)];
  for(const at of points){f.now=at;sleep.tick(at);const token=sleep.snapshot({});driver.frame(token.rest);inter.pose(pose);const expected=ref.tick(at,token.rest.started);assert.ok(Math.abs(token.rest.alpha-expected.alpha)<1e-12,`hz${hz} t${at-start} alpha`);assert.equal(token.rest.phase,expected.phase);for(let c=0;c<45;c++)assert.ok(Math.abs(pose[c]-expected.pose[c])<2e-7,`hz${hz} at${at-start} channel${c}`);if(at===start+1100)assert.equal(token.waking,true);if(at===start+1100.001)assert.equal(token.waking,false);}
  assert.equal(sleep.snapshot({}).rest.alpha,0);assert.equal(sleep.snapshot({}).rest.phase,null);driver.close();inter.close();sleep.close();
 }
});
