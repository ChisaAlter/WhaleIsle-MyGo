import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {asset,fixture,flush,plain} from './body-fixture.mjs';

const ref = readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
function reference() {
  const scope={Math,pose:null,rigStateT0:0,STILL_FADE:7,stillCtl:{alpha:0,target:0,name:null,entry:null},tickStill:{},lastFullClear:0,smooth:p=>p*p*(3-2*p),clamp01:p=>Math.max(0,Math.min(1,p))};
  vm.createContext(scope);
  const states=ref.slice(ref.indexOf('const LIVE_STATES = {'),ref.indexOf('// Facial programs'));
  vm.runInContext(states+'\nglobalThis.states=LIVE_STATES;',scope);
  const fade=ref.slice(ref.indexOf('  stillCtl.alpha +='),ref.indexOf('  fx.dx = 0;',ref.indexOf('function tickStill')));
  scope.fade=()=>vm.runInContext('{const dt=.05;'+fade+'}',scope);
  return scope;
}
const opaque = (x=80,y=40,right=430,bottom=480) => {
  const pixels=new Uint8ClampedArray(512*512*4);
  for(let py=y;py<=bottom;py++)for(let px=x;px<=right;px++)pixels[(py*512+px)*4+3]=255;
  return pixels;
};
function local() {
  const f=fixture({sleepEnabled:true});f.load('pet-sleep.js');let paints=0,changes=0,releases=0;
  const canvas=f.node('sprite');canvas.hidden=false;f.node('fallback').hidden=true;
  const sleep=f.window.WhalePetSleep.create({canvas,body:f.body(),getLayout:()=>({X:120,Y:180,Width:240,Height:260}),onChange:()=>changes++,onRelease:()=>releases++});
  return Object.assign(f,{sleep,canvas,paint:()=>paints++,counts:()=>({paints,changes,releases})});
}

test('sleep body formula uses the real feet reference, original alpha box and both facing directions',()=>{
  const f=local(), API=f.window.WhalePetSleep, r=reference(), pixels=opaque(4,6,502,508);
  const box=API.measure(pixels);assert.deepEqual(plain(box),{x:4,y:6,right:502,bottom:508},'alpha boxes are not tightened to the crop');
  for(const facing of [1,-1])for(const angle of [0,-.2,-1])for(const sy of [1,1.02,1.04]){
    const layout={X:120,Y:180,Width:240,Height:260};const actual=API.geometry(box,layout,facing,angle,14,sy);
    const char={x:(box.x-60)*240/390,y:(box.y-20)*260/492,right:(box.right-60)*240/390,bottom:(box.bottom-20)*260/492};
    const corners=[[char.x,char.y],[char.right,char.y],[char.x,char.bottom],[char.right,char.bottom]].map(([x,y])=>{const px=(x-120)*facing,py=(y-260)*sy;return{x:240+px*Math.cos(angle)-py*Math.sin(angle),y:452+px*Math.sin(angle)+py*Math.cos(angle)};});
    const expected={x:Math.min(...corners.map(p=>p.x)),y:Math.min(...corners.map(p=>p.y)),right:Math.max(...corners.map(p=>p.x)),bottom:Math.max(...corners.map(p=>p.y))};
    for(const key of ['x','y','right','bottom'])assert.ok(Math.abs(actual.rect[key]-expected[key])<1e-10);
    // The CSS feet point remains fixed except for the reference dy. Task-29 has
    // already applied facing to the canvas pixels; CSS must not mirror twice.
    const [a,b,c,d,e,g]=actual.matrix;assert.ok(Math.abs(a*120+c*260+e-120)<1e-10);assert.ok(Math.abs(b*120+d*260+g-272)<1e-10);
  }
  assert.ok(r.states.sleep);f.sleep.close();
});

test('per-rAF alpha and sleep-enter source programs match reference at 30/60/120/144Hz including doze carry',()=>{
  for(const hz of [30,60,120,144])for(const doze of [false,true]){
    const f=local(), r=reference();
    if(doze){r.stillCtl.target=1;f.sleep.doze(true);for(let i=0;i<7;i++){f.now+=1000/hz;f.sleep.tick(f.now);r.fade();}}
    f.sleep.request(true);r.stillCtl.name='sleep-enter';r.stillCtl.target=1;r.rigStateT0=f.now/1000;
    for(let i=0;i<hz*2;i++){
      f.now+=1000/hz;f.sleep.tick(f.now);r.fade();
      const s={fx:{},set(){}};r.states['sleep-enter'](s,f.now/1000);
      const token=f.sleep.snapshot({facing:1});
      assert.ok(Math.abs(token.angle-s.fx.rot*(r.stillCtl.alpha*r.stillCtl.alpha*(3-2*r.stillCtl.alpha)))<1e-12);
      assert.ok(Math.abs(token.dy-s.fx.dy*(r.stillCtl.alpha*r.stillCtl.alpha*(3-2*r.stillCtl.alpha)))<1e-11);
    }
    f.sleep.close();
  }
});

test('alpha<4 and stale state keep the successful pixels, CSS matrix and unpadded hit paired',()=>{
  const f=local(),pixels=opaque();f.sleep.request(true);f.now=2200;f.sleep.tick(f.now);
  let token=f.sleep.snapshot({facing:-1});assert.equal(f.sleep.commit(pixels,token,{},f.paint),true);
  const before=plain({evidence:f.sleep.evidence,style:f.canvas.style,hit:f.body().hitRect(f.body().target),count:f.counts()});
  assert.equal(f.body().hitRect(f.canvas),null);const hit=f.body().hitRect(f.body().target),padded=f.body().rect(f.body().target);assert.equal(padded.width-hit.width,16);
  f.now=2300;f.sleep.tick(f.now);assert.equal(f.sleep.commit(new Uint8ClampedArray(pixels.length),f.sleep.snapshot({}),{},f.paint),false);
  assert.deepEqual(plain({evidence:f.sleep.evidence,style:f.canvas.style,hit:f.body().hitRect(f.body().target),count:f.counts()}),before);
  token=f.sleep.snapshot({});f.sleep.request(false);assert.equal(f.sleep.commit(pixels,token,{},f.paint),false);
  assert.equal(f.sleep.commit(new Uint8ClampedArray(pixels.length),f.sleep.snapshot({}),{},f.paint),false);assert.equal(f.sleep.active,true);
  f.now=3500;f.sleep.tick(f.now);
  assert.equal(f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint),true);assert.equal(f.sleep.active,false);assert.equal(f.body().target.hidden,true);assert.equal(f.canvas.style.pointerEvents,'');
  f.sleep.close();
});

test('wake dy/sy match extracted source at each RAF and pending-only wake releases reservation after a valid awake frame',()=>{
  const f=local(),r=reference(),pixels=opaque();f.sleep.request(true);for(let i=0;i<50;i++){f.now+=17;f.sleep.tick(f.now);}
  f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint);f.sleep.request(false);r.rigStateT0=f.now/1000;r.stillCtl.alpha=1;r.stillCtl.target=1;
  for(let i=0;i<65;i++){f.now+=17;f.sleep.tick(f.now);const token=f.sleep.snapshot({});if(!token.waking)break;r.fade();const s={fx:{},set(){}};r.states.wake(s,f.now/1000);assert.equal(token.angle,0);assert.ok(Math.abs(token.dy-s.fx.dy)<1e-8);assert.ok(Math.abs(token.sy-s.fx.sy)<1e-10);}
  f.sleep.close();
  const p=local();p.sleep.request(true);assert.equal(p.body().managed(p.canvas),true);p.sleep.request(false);p.body().capture(p.canvas,8,true);assert.equal(p.sleep.commit(pixels,p.sleep.snapshot({}),{},p.paint),false);p.body().capture(p.canvas,8,false);p.now+=1200;p.sleep.tick(p.now);assert.equal(p.sleep.commit(pixels,p.sleep.snapshot({}),{},p.paint),true);assert.equal(p.body().managed(p.canvas),false);p.sleep.close();
});

test('an old owner cannot close or publish into a replacement owner, and a retired owner cannot reserve again',()=>{
  const f=local(),pixels=opaque();f.sleep.request(true);f.sleep.tick(f.now);f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint);
  const newer=f.window.WhalePetSleep.create({canvas:f.canvas,body:f.body(),getLayout:()=>({X:120,Y:180,Width:240,Height:260}),sleeping:true});newer.tick(f.now+600);newer.commit(pixels,newer.snapshot({}),{},f.paint);const before=plain({style:f.canvas.style,hit:f.body().hitRect(f.body().target)});f.sleep.close();assert.deepEqual(plain({style:f.canvas.style,hit:f.body().hitRect(f.body().target)}),before);
  f.body().capture(f.body().target,4,true);newer.stop();newer.request(true);newer.doze(true);newer.tick(f.now+500);assert.equal(f.body().managed(f.canvas),false);assert.equal(newer.commit(pixels,newer.snapshot({}),{},f.paint),false);newer.close();assert.equal(f.body().target.hidden,true);
});

test('alpha measurement happens first and every 240 valid commits, not per display tick or transparent frame',()=>{
  const f=local(),first=opaque(80,40,430,480),next=opaque(100,60,400,460);f.sleep.request(true);
  for(let i=0;i<239;i++){f.now+=17;f.sleep.tick(f.now);assert.equal(f.sleep.commit(i?next:first,f.sleep.snapshot({}),{},f.paint),true);}
  assert.equal(f.sleep.frames,239);assert.equal(f.sleep.evidence.box.x,80);
  f.sleep.commit(new Uint8ClampedArray(first.length),f.sleep.snapshot({}),{},f.paint);assert.equal(f.sleep.frames,239);
  f.sleep.commit(next,f.sleep.snapshot({}),{},f.paint);assert.equal(f.sleep.frames,240);assert.equal(f.sleep.evidence.box.x,100);f.sleep.close();
});

test('normal captured neural frames paint; sleep and wake handoff waits for original target release',async()=>{
  const f=local(),pixels=opaque(),b=f.body(),target=f.canvas;
  b.capture(target,1,true);assert.equal(f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint),true,'awake drag paints without sleep ownership');
  f.sleep.request(true);const token=f.sleep.snapshot({});assert.equal(f.sleep.commit(pixels,token,{},f.paint),false);assert.equal(target.style.pointerEvents,undefined);
  b.capture(target,1,false);assert.equal(f.sleep.commit(pixels,token,{},f.paint),false,'capture generation changed');
  f.sleep.tick(2300);assert.equal(f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint),true);
  b.capture(b.target,2,true);f.sleep.request(false);assert.equal(f.sleep.commit(pixels,f.sleep.snapshot({}),{},f.paint),false);assert.equal(b.target.hidden,false);
  f.sleep.stop();assert.equal(b.target.hidden,false,'captured target stays alive while fallback can paint');assert.equal(f.canvas.style.transform,'');
  b.capture(b.target,2,false);assert.equal(b.target.hidden,true);f.sleep.close();
});
