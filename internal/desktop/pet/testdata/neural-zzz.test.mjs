import assert from 'node:assert/strict';
import test from 'node:test';
import vm from 'node:vm';
import {readFileSync} from 'node:fs';
import {fixture,flush,plain} from './body-fixture.mjs';
const source=readFileSync(new URL('../../../../reference/electron/src/renderer/pet-live2d.js',import.meta.url),'utf8');
const seeded=seed=>()=>{seed=(Math.imul(seed,1664525)+1013904223)>>>0;return seed/4294967296;};
const zzz=f=>f.document.body.children.filter(node=>node.textContent==='Z'&&node.isConnected);
function original(clock,seed){
 const random=seeded(seed),paint=[];const ctx={fillStyle:'',font:'',globalAlpha:1,fillText(text,x,y){paint.push({text,x,y,font:this.font,opacity:this.globalAlpha,color:this.fillStyle});}};
 const scope={Math,performance:{now:()=>clock.now},rng:(a,b)=>a+random()*(b-a),particles:[],ctx2d:ctx,sleeping:false,session:{},rig:{ready:false},stillCtl:{name:null},drawPos:{x:120,y:180},petW:()=>240,petH:()=>260};vm.createContext(scope);
 const functions=source.slice(source.indexOf('function spawn(type, x, y, vx, vy)'),source.indexOf('// ──',source.indexOf('function drawParticles(now)')));
 const spawn=source.slice(source.indexOf('    if (!tickStill._nextZzz'),source.indexOf('    if (overPet(pointer.x, pointer.y))',source.indexOf('    if (!tickStill._nextZzz')));
 const lifecycle=source.slice(source.indexOf('  // Particle lifecycle.'),source.indexOf('  if (stillCtl.alpha > 0.01',source.indexOf('  // Particle lifecycle.')));
 vm.runInContext(functions+'\nfunction tickStill(now){if(sleeping){'+spawn+'}\n'+lifecycle+'}\n',scope);
 return {tick(now,state){scope.sleeping=state?.sleeping===true;scope.stillCtl.name=state?.phase;scope.drawPos={x:state?.layout?.X??120,y:state?.layout?.Y??180};scope.petW=()=>state?.layout?.Width??240;scope.petH=()=>state?.layout?.Height??260;scope.tickStill(now);paint.length=0;scope.drawParticles(now);return paint;}};
}
function direct(seed=177){const f=fixture();f.load('pet-care-animation.js');const care=f.window.WhalePetCareAnimation.create({body:f.body()});care.update({Rest:{Sleeping:true}});assert.ok(f.document.body.children.some(n=>n.textContent==='Z z z'));const driver=care.useNeuralParticles(seeded(seed));assert.equal(f.document.body.children.some(n=>n.textContent==='Z z z'),false);return {f,care,driver};}

test('actual reference spawn, perRAF lifecycle and draw agree over32s dualseeds and30/60/120/144Hz including wake tails, respawn and stage movement',()=>{
 for(const hz of [30,60,120,144])for(const seed of [177,905]){
  const {f,care,driver}=direct(seed),ref=original(f,seed);const start=f.now;
  for(let i=0;i<hz*32;i++){
   const elapsed=i*1000/hz;f.now=start+elapsed;
   const sleeping=elapsed<6500||elapsed>=8000&&elapsed<12700||elapsed>=14900&&elapsed<27600;
   const since=elapsed<6500?elapsed:elapsed<12700?elapsed-8000:elapsed-14900;
   const state=sleeping?{sleeping:true,phase:since<1200?'sleep-enter':'sleep',layout:{X:elapsed<17000?120:280,Y:elapsed<17000?180:240,Width:240,Height:260}}:null;
   driver.frame(f.now,state);const expected=ref.tick(f.now,state),actual=zzz(f).filter(n=>Number(n.style.opacity)>0);
   assert.equal(actual.length,expected.filter(p=>p.opacity>0).length,`hz${hz} seed${seed} at${elapsed}`);
   for(let n=0;n<actual.length;n++){const node=actual[n],p=expected.filter(p=>p.opacity>0)[n];assert.ok(Math.abs(parseFloat(node.style.left)-p.x)<1e-10);assert.ok(Math.abs(parseFloat(node.style.top)-p.y)<1e-10);assert.ok(Math.abs(Number(node.style.opacity)-p.opacity)<1e-12);assert.equal(node.style.fontSize,p.font.match(/\d+px/)[0]);assert.equal(node.style.color,p.color);}
  }
  assert.equal(zzz(f).length,0);care.close();driver.frame(f.now+50,{sleeping:true,phase:'sleep'});assert.equal(zzz(f).length,0);
 }
});

test('neural ownership retires legacy marker; hidden pauses motion, wake preserves particles and fallback restores the original marker',()=>{
 const {f,care,driver}=direct();const state={sleeping:true,phase:'sleep',layout:{X:120,Y:180,Width:240,Height:260}};
 driver.frame(f.now,state);const node=zzz(f)[0];assert.ok(node);f.document.hidden=true;const before=plain(node.style);driver.frame(f.now+1200,state);assert.deepEqual(plain(node.style),before);
 f.document.hidden=false;care.update({Rest:{Sleeping:false}});f.now+=600;driver.frame(f.now,null);assert.equal(zzz(f)[0],node);f.now+=1900;driver.frame(f.now,null);assert.equal(zzz(f).length,0);
 care.update({Rest:{Sleeping:true}});driver.frame(f.now,state);assert.equal(zzz(f).length,1);driver.close();assert.equal(zzz(f).length,0);assert.equal(f.document.body.children.filter(n=>n.textContent==='Z z z').length,1);driver.close();assert.equal(f.document.body.children.filter(n=>n.textContent==='Z z z').length,1);care.close();assert.equal(f.document.body.children.length,0);
});

test('runtime spawns only from committed current rest; capture, transparent pending, stale completion, wake, fallback and close preserve ownership',async()=>{
 const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);const canvas=f.node('sprite');
 f.emit(canvas,'pointerdown',{pointerId:7});f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();await f.frame();assert.equal(zzz(f).length,0);f.emit(canvas,'pointerup',{pointerId:7});
 const zero=new Uint8ClampedArray(512*512*4);f.setPixels(zero);await f.frame();await f.frame();assert.equal(zzz(f).length,0);
 const valid=zero.slice();for(let y=40;y<480;y+=2)for(let x=80;x<430;x+=2)valid[(y*512+x)*4+3]=255;f.setPixels(valid);await f.frame();await f.frame();assert.equal(zzz(f).length,1);assert.equal(f.document.body.children.some(n=>n.textContent==='Z z z'),false);
 const node=zzz(f)[0],before=parseFloat(node.style.left);f.window.dispatchEvent(new f.scope.CustomEvent('whale-pet-layout',{detail:{X:300,Y:260,Width:240,Height:260}}));assert.equal(parseFloat(node.style.left),before,'spawned particle retains its world coordinate on layout updates');
 f.setTask({Rest:{Sleeping:false}});await f.fire(1500);await f.frame(f.now+400);assert.equal(zzz(f)[0],node);await f.frame(f.now+2600);assert.equal(zzz(f).length,0);
 f.setTask({Rest:{Sleeping:true}});await f.fire(1500);const gate=f.gateFrame(),pending=f.frame();await flush();f.setTask({Rest:{Sleeping:false}});await f.fire(1500);gate.resolve();await pending;await f.frame();assert.equal(zzz(f).length,0,'stale sleep completion cannot start particles');
 f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();await f.frame();assert.equal(zzz(f).length,1);f.failFrame();await f.frame();await flush();assert.equal(f.document.body.dataset.renderer,'rig');assert.equal(zzz(f).length,0);assert.ok(f.document.body.children.some(n=>n.textContent==='Z z z'));f.close();assert.equal(zzz(f).length,0);
});
