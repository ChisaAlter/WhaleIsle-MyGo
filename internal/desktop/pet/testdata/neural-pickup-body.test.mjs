import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush,plain} from './body-fixture.mjs';

const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
function originalDraw(frame,layout){
 let matrix=[1,0,0,1,0,0],drawn;
 const mul=([a,b,c,d,e,f])=>{const [x,y,z,w,u,v]=matrix;matrix=[x*a+z*b,y*a+w*b,x*c+z*d,y*c+w*d,x*e+z*f+u,y*e+w*f+v];};
 const ctx2d={save(){},restore(){},translate(x,y){mul([1,0,0,1,x,y]);},rotate(a){mul([Math.cos(a),Math.sin(a),-Math.sin(a),Math.cos(a),0,0]);},scale(x,y){mul([x,0,0,y,0,0]);},drawImage(_image,x,y){mul([1,0,0,1,x,y]);drawn=matrix.slice();}};
 const scope={Math,ctx2d,rigT:frame.time||0,settings:{opacity:1},srReady:false,outCanvas:{},stillCtl:{alpha:frame.alpha},smooth:t=>t*t*(3-2*t),clamp01:t=>Math.max(0,Math.min(1,t)),fx:{rot:frame.angle-frame.rotation*frame.a,dx:0,dy:frame.dy, sx:frame.sx,sy:frame.sy/(1+((frame.scaleY||1)-1)*frame.a)},liveFx:{rot:frame.rotation,dx:0,dy:0,sx:1,sy:frame.scaleY||1,pivot:frame.pivot,stars:false},dragging:frame.dragging,thrown:frame.thrown,physPoint:{x:frame.cursorX,y:frame.cursorY+12},petW:()=>layout.Width,petH:()=>layout.Height,drawPos:{x:layout.X,y:layout.Y},facing:frame.facing,GRAB_FRAC:.085};
 vm.runInNewContext(source.slice(source.indexOf('function drawLive()'),source.indexOf('function drawRigPart(')),scope);scope.drawLive();return drawn;
}
const pixels=()=>{const value=new Uint8ClampedArray(512*512*4);for(let y=40;y<=480;y++)for(let x=80;x<=430;x++)value[(y*512+x)*4+3]=255;return value;};
function local(){
 const f=fixture({sleepEnabled:true});f.load('pet-sleep.js');const canvas=f.node('sprite'),body=f.body();let layout={X:120,Y:180,Width:240,Height:260},paints=0;
 const sleep=f.window.WhalePetSleep.create({canvas,body,getLayout:()=>layout});
 return Object.assign(f,{canvas,sleep,setLayout(next){layout=next;},paint(){paints++;},getPaints(){return paints;}});
}
const frame=(fields={})=>Object.freeze({phase:'pick-up',alpha:1,a:1,angle:.2,rotation:.2,scaleY:1.06,sx:1,sy:1.06,dy:-6,pivot:'grab',hanging:true,dragging:true,thrown:false,cursorX:250,cursorY:200,facing:1,...fields});

test('pickup whole-canvas matrix and tight alpha corners match extracted reference drawLive for grab/feet and facing',()=>{
 const f=local(),api=f.window.WhalePetSleep,layout={X:120,Y:180,Width:240,Height:260},box={x:80,y:40,right:430,bottom:480};
 for(const hanging of [true,false])for(const facing of [-1,1])for(const angle of [-.3,0,.25]){
  const value=frame({hanging,facing,angle,pivot:hanging?'grab':null}),actual=api.pickupGeometry(box,layout,value),expected=originalDraw(value,layout);
  const css=Array.from(actual.matrix);css[4]+=layout.X;css[5]+=layout.Y;
  if(!hanging&&facing===-1){const [a,b,c,d,e,g]=expected;expected.splice(0,6,-a,-b,c,d,e+a*layout.Width,g+b*layout.Width);}
  for(let i=0;i<6;i++)assert.ok(Math.abs(css[i]-expected[i])<1e-9,`matrix${i}: ${css[i]} != ${expected[i]}`);
  const corners=[[80,40],[430,40],[80,480],[430,480]].map(([x,y])=>{let px=(x-60)*240/390;if(!hanging&&facing<0)px=240-px;const py=(y-20)*260/492;return {x:css[0]*px+css[2]*py+css[4],y:css[1]*px+css[3]*py+css[5]};});
  assert.ok(Math.abs(actual.rect.x-Math.min(...corners.map(p=>p.x)))<1e-9);assert.ok(Math.abs(actual.rect.bottom-Math.max(...corners.map(p=>p.y)))<1e-9);
 }
 f.sleep.close();
});

test('pickup commits under original canvas/proxy capture without releasing or transferring; stale mode/capture/layout keep last pair',()=>{
 const f=local(),p=pixels(),input={epoch:1,mode:'pickup'};f.body().capture(f.canvas,11,true);
 const token=f.sleep.bodySnapshot(frame(),input);assert.equal(f.sleep.commitNeuralBody(p,token,input,f.paint),true);assert.equal(f.body().capturing(f.canvas),true);assert.equal(f.captures.length,0);const target=f.body().target;assert.equal(target.hidden,false);assert.equal(f.canvas.style.pointerEvents,'none');assert.equal(f.body().hitRect(f.canvas),null);
 const before=plain({matrix:f.canvas.style.transform,hit:f.body().hitRect(target),paints:f.getPaints()});
 assert.equal(f.sleep.commitNeuralBody(p,token,{epoch:2,mode:'throw'},f.paint),false);f.body().capture(f.canvas,11,false);assert.equal(f.sleep.commitNeuralBody(p,token,input,f.paint),false);f.body().capture(target,12,true);
 assert.equal(f.sleep.commitNeuralBody(p,f.sleep.bodySnapshot(frame(),input),input,f.paint),true);assert.equal(f.body().capturing(target),true);
 const next=f.sleep.bodySnapshot(frame(),input);f.setLayout({X:121,Y:180,Width:240,Height:260});const count=f.getPaints();assert.equal(f.sleep.commitNeuralBody(p,next,input,f.paint),false);assert.equal(f.getPaints(),count);f.body().capture(target,12,false);f.sleep.close();
});

test('crossing sleep kind retains the last pickup pixels/matrix/hit while captured; actual release permits original sleep commit',()=>{
 const f=local(),p=pixels(),input={epoch:1,mode:'pickup'},target=f.body().target;f.body().capture(f.canvas,3,true);assert.equal(f.sleep.commitNeuralBody(p,f.sleep.bodySnapshot(frame(),input),input,f.paint),true);
 const before=plain({style:f.canvas.style,hit:f.body().hitRect(target),paints:f.getPaints()});f.sleep.request(true);f.sleep.tick(f.now+500);assert.equal(f.sleep.commitRest(p,f.sleep.snapshot({state:'pick-up'}),{state:'pick-up'},f.paint),false);assert.deepEqual(plain({style:f.canvas.style,hit:f.body().hitRect(target),paints:f.getPaints()}),before);
 f.body().capture(f.canvas,3,false);assert.equal(f.sleep.commitRest(p,f.sleep.snapshot({}),{},f.paint),true);assert.equal(f.sleep.pickupActive,false);assert.equal(f.sleep.active,true);f.sleep.close();
});

test('transparent/rejected pickup frames retain the last pair; layout reprojects it and fade retires a captured proxy only on real release',()=>{
 const f=local(),p=pixels(),input={epoch:1,mode:'pickup'},target=f.body().target;assert.equal(f.sleep.commitNeuralBody(p,f.sleep.bodySnapshot(frame(),input),input,f.paint),true);
 const before=plain({matrix:f.canvas.style.transform,hit:f.body().hitRect(target),paints:f.getPaints()});assert.equal(f.sleep.commitNeuralBody(new Uint8ClampedArray(p.length),f.sleep.bodySnapshot(frame({angle:.6}),input),input,f.paint),false);assert.deepEqual(plain({matrix:f.canvas.style.transform,hit:f.body().hitRect(target),paints:f.getPaints()}),before);
 f.setLayout({X:220,Y:230,Width:240,Height:260});f.sleep.reprojectPickup();const moved=f.body().hitRect(target);assert.ok(Math.abs(moved.x-before.hit.x-100)<1e-9);assert.ok(Math.abs(moved.y-before.hit.y-50)<1e-9);assert.equal(f.getPaints(),before.paints);
 f.body().capture(target,9,true);const neutral=frame({phase:null,hanging:false,mode:''}),awake={epoch:2,mode:''};assert.equal(f.sleep.commitNeuralBody(p,f.sleep.bodySnapshot(neutral,awake),awake,f.paint),true);assert.equal(f.sleep.pickupActive,false);assert.equal(f.body().managed(f.canvas),false);assert.equal(target.hidden,false);assert.equal(target.style.pointerEvents,'none');assert.equal(f.body().hitRect(target),null);f.body().capture(target,9,false);assert.equal(target.hidden,true);f.sleep.close();
});

test('pending pickup reserves prior CSS against interaction before a transparent first inference; close releases only its original pointer',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);const canvas=f.node('sprite');await f.frame();const previous=canvas.style.transform;
 f.emit(canvas,'pointerdown',{pointerId:31});f.emit(canvas,'pointermove',{pointerId:31,screenX:220});f.setInteraction({Mode:'pickup',CursorX:260,CursorY:220,VX:80});await f.fire(50);f.setPixels(new Uint8ClampedArray(512*512*4));await f.frame(f.now+17);const reserved=canvas.style.transform;for(let i=0;i<6;i++)await f.frame(f.now+17);assert.equal(canvas.style.transform,reserved);assert.equal(f.body().managed(canvas),true);assert.equal(f.body().target.hidden,true);assert.equal(canvas.captured.has(31),true);f.close();assert.equal(canvas.captured.has(31),false);assert.equal(f.nav.filter(url=>url.endsWith('gesture-cancel')).length,1);
});

test('throw anchor is continuous from last held native layout offset and then follows native layout rather than the stationary cursor',async()=>{
 const f=fixture({sleepEnabled:true});f.load('pet-interaction.js');const interaction=f.window.WhalePetInteraction.create({body:f.body(),getSettings:()=>({}),isSleeping:()=>false}),driver=interaction.useNeuralRest();await flush();
 const tick=()=>driver.frame({phase:null,alpha:0,a:0,time:f.now/1000,started:0,elapsed:0,revision:0,sleeping:false});
 f.setInteraction({Mode:'pickup',CursorX:250,CursorY:220,Layout:{X:120,Y:180,Width:240,Height:260}});await f.fire(50);tick();const held=driver.pickup;
 f.setInteraction({Mode:'throw',Layout:{X:120,Y:180,Width:240,Height:260}});await f.fire(50);f.now+=17;tick();assert.equal(driver.pickup.cursorX,held.cursorX);assert.equal(driver.pickup.cursorY,held.cursorY);
 f.setInteraction({Mode:'throw',Layout:{X:180,Y:210,Width:240,Height:260}});await f.fire(50);f.now+=17;tick();assert.equal(driver.pickup.cursorX,held.cursorX+60);assert.equal(driver.pickup.cursorY,held.cursorY+30);driver.close();interaction.close();
});

test('real runtime keeps original pointerid through pickup, second pointer, stale inference, fallback and late release',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);const canvas=f.node('sprite'),target=f.body().target;
 f.emit(canvas,'pointerdown',{pointerId:7});f.emit(canvas,'pointermove',{pointerId:7,screenX:200});f.setInteraction({Mode:'pickup',CursorX:250,CursorY:220,VX:40});await f.fire(50);await f.frame(f.now+50);
 assert.equal(f.body().capturing(canvas),true);assert.equal(target.hidden,false);assert.match(canvas.style.transform,/^matrix\(/);assert.equal(f.emit(target,'pointerdown',{pointerId:8}).defaultPrevented,false);assert.equal(target.captured.has(8),false);
 const gate=f.gateFrame(),pending=f.frame(f.now+50);await flush();const old=canvas.style.transform;f.setInteraction({Mode:'throw',VX:300});await f.fire(50);gate.resolve();await pending;assert.equal(canvas.style.transform,old,'Mode epoch invalidates awaited frame');
 f.failFrame();await f.frame(f.now+50);assert.equal(f.document.body.dataset.renderer,'rig');assert.equal(f.body().managed(canvas),false);assert.equal(canvas.captured.has(7),true,'fallback retains original node/id');assert.equal(f.body().hitRect(target),null);assert.equal(canvas.style.pointerEvents,'');f.emit(target,'pointerup',{pointerId:8});assert.equal(canvas.captured.has(7),true);f.emit(canvas,'pointerup',{pointerId:7});assert.equal(canvas.captured.has(7),false);f.close();
});
