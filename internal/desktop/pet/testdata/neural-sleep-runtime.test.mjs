import assert from 'node:assert/strict';
import test from 'node:test';
import {fixture,flush,plain,deferred} from './body-fixture.mjs';
const proxy=f=>f.node('pet-body-hit');
const visibleDraws=f=>f.draw.filter(row=>row[0]==='sprite'&&row[1]==='draw').length;
async function enter(f){f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame(f.now+50);assert.equal(proxy(f).hidden,false);}

test('real runtime commits sleep matrix/hit together, only proxy wakes and normal care/interaction cannot overwrite it',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);await enter(f);await f.frame(f.now+1400);
  const rest=f.document.body.children.find(node=>node.textContent==='Z');assert.ok(rest?.isConnected,'a neural particle starts after a valid committed sleep frame');  const matrix=f.node('sprite').style.transform;assert.match(matrix,/^matrix\(/);assert.equal(f.node('sprite').style.pointerEvents,'none');assert.equal(f.node('sprite').style.rotate,'');assert.equal(f.node('sprite').style.scale,'');
  const b=f.body(),hit=b.hitRect(proxy(f)),r=b.rect(proxy(f));assert.equal(r.width-hit.width,16);assert.equal(b.hitRect(f.node('sprite')),null);
  const report=f.nav.filter(url=>url.includes('interactive-regions?')).at(-1),regions=JSON.parse(decodeURIComponent(report.split('data=')[1])).Regions;
  assert.ok(regions.some(box=>Math.abs(box.Width-hit.width)<1e-8));assert.equal(regions.some(box=>box.Width===240&&box.Height===260),true,'status toggle retains its separate test rectangle');
  const before=f.nav.length;f.emit(f.node('pet-status-toggle'),'pointerenter');assert.equal(f.nav.length,before);f.emit(proxy(f),'pointerenter');assert.equal(f.nav.at(-1),'whale-pet://localhost/wake-contact');
  f.emit(proxy(f),'contextmenu');assert.equal(f.node('pet-status-card').hidden,false);f.emit(f.document,'keydown',{key:'Escape'});f.emit(proxy(f),'pointerdown');await f.fire(600);assert.equal(f.node('pet-status-card').hidden,false);f.emit(proxy(f),'pointerup');
  const file={name:'local.txt',text(){throw Error('file read');}};const inside={clientX:r.left+1,clientY:r.top+1,dataTransfer:{files:[file]}};assert.equal(f.emit(f.document,'drop',inside).defaultPrevented,true);assert.equal(f.emit(f.document,'drop',{...inside,clientX:r.right+1}).defaultPrevented,false);
  f.setPixels(new Uint8ClampedArray(512*512*4));await f.frame();assert.equal(f.node('sprite').style.transform,matrix,'interaction RAF cannot overwrite the last valid sleep matrix');f.close();await flush();assert.equal(proxy(f).hidden,true);
});

test('pending sleep keeps the old CSS pair; inflight old state and transparent frames cannot publish',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);await f.frame();const style=f.node('sprite').style.transform;
  const gate=f.gateFrame(),pending=f.frame();await flush();f.setTask({Rest:{Sleeping:true}});await f.fire(1500);
  assert.equal(f.node('sprite').style.transform,style);assert.equal(f.node('sprite').style.rotate,undefined,'care must not write old 18deg before commit');
  const draws=visibleDraws(f);gate.resolve();await pending;assert.equal(visibleDraws(f),draws);assert.equal(proxy(f).hidden,true);
  const zero=new Uint8ClampedArray(512*512*4);f.setPixels(zero);await f.frame();assert.equal(proxy(f).hidden,true);assert.equal(f.node('sprite').style.transform,style);
  const pixels=new Uint8ClampedArray(zero.length);for(let y=40;y<480;y+=2)for(let x=80;x<430;x+=2)pixels[(y*512+x)*4+3]=255;f.setPixels(pixels);await f.frame();assert.equal(proxy(f).hidden,false);
  const before=plain({style:f.node('sprite').style,hit:f.body().hitRect(proxy(f)),draws:visibleDraws(f)});f.setPixels(zero);await f.frame();const current=f.body().hitRect(proxy(f));assert.equal(visibleDraws(f),before.draws+1,'cached accepted image is presented without publishing transparent output');assert.notEqual(f.node('sprite').style.transform,before.style.transform);assert.equal(current.width,f.body().rect(proxy(f)).width-16,'current transform and tight hit remain paired');
  f.close();
});

test('dataset sleep and delayed provider open cannot retain old care CSS after a wake before first paint',async()=>{
  const gate=deferred(),f=fixture({sleepEnabled:true,openGate:gate,initialSleeping:true});const starting=f.start();await flush();assert.equal(f.body().managed(f.node('sprite')),true);
  f.setTask({Rest:{Sleeping:false}});await f.fire(1500);f.now+=1300;gate.resolve();await starting;await f.frame();assert.equal(f.body().managed(f.node('sprite')),false);assert.equal(f.node('sprite').style.rotate,'');assert.equal(f.node('sprite').style.scale,'');assert.equal(proxy(f).hidden,true);f.close();
});

test('layout reprojects the last valid pair during transparent inference and rejects invalid layout',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);await enter(f);await f.frame();const before=plain(f.body().hitRect(proxy(f)));const rest=f.document.body.children.find(node=>node.textContent==='Z'),marker={left:parseFloat(rest.style.left),top:parseFloat(rest.style.top)},count=f.document.body.children.length;
  f.setPixels(new Uint8ClampedArray(512*512*4));const layout={X:220,Y:230,Width:240,Height:260,Area:{X:0,Y:0,Width:1200,Height:800}};
  f.window.dispatchEvent(new f.scope.CustomEvent('whale-pet-layout',{detail:layout}));const moved=f.body().hitRect(proxy(f));assert.ok(Math.abs(moved.x-before.x-100)<1e-9);assert.ok(Math.abs(moved.y-before.y-50)<1e-9);assert.equal(parseFloat(rest.style.left),marker.left);assert.equal(parseFloat(rest.style.top),marker.top);await f.frame();const displayed=plain(f.body().hitRect(proxy(f)));assert.notDeepEqual(displayed,plain(moved),'cached image body clock continues while transparent inference is rejected');assert.equal(displayed.width,f.body().rect(proxy(f)).width-16);assert.ok(Math.abs(parseFloat(rest.style.left)-marker.left)<3);assert.ok(Math.abs(parseFloat(rest.style.top)-marker.top)<1);assert.equal(f.document.body.children.length,count,'layout preserves particle world coordinates without spawning another');
  f.window.dispatchEvent(new f.scope.CustomEvent('whale-pet-layout',{detail:{X:0,Y:0,Width:0,Height:NaN}}));assert.deepEqual(plain(f.body().hitRect(proxy(f))),displayed);f.close();
});

test('pressed-with-empty-mode waits for original pointer; second pointer cannot overwrite capture or block handoff',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);const canvas=f.node('sprite');
  f.emit(canvas,'pointerdown',{pointerId:11});assert.equal(f.body().capturing(canvas),true);
  const second=f.emit(canvas,'pointerdown',{pointerId:12});assert.equal(second.defaultPrevented,false);assert.equal(canvas.captured.has(12),false);
  await enterPending();
  async function enterPending(){f.setTask({Rest:{Sleeping:true}});await f.fire(1500);await f.frame();assert.equal(proxy(f).hidden,true);}
  f.emit(canvas,'pointerup',{pointerId:12});assert.equal(f.body().capturing(canvas),true);
  f.emit(canvas,'pointerup',{pointerId:11});await f.frame();assert.equal(proxy(f).hidden,false);assert.equal(f.body().capturing(),false);f.close();
});

test('held sleep target inference failure reaches real rig immediately, restores fallback geometry then releases original target',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);await enter(f);await f.frame(f.now+1300);
  const target=proxy(f);f.emit(target,'pointerdown',{pointerId:3});f.failFrame();await f.frame();await flush();assert.equal(f.document.body.dataset.renderer,'rig');
  assert.equal(f.node('sprite').style.pointerEvents,'');assert.equal(target.hidden,false);assert.equal(target.style.pointerEvents,'none');assert.equal(f.body().hitRect(target),null);assert.deepEqual(plain(f.body().rect()),plain(f.node('sprite').getBoundingClientRect()));
  f.setTask({Rest:{Sleeping:false}});await f.fire(1500);f.setTask({Rest:{Sleeping:true}});await f.fire(1500);assert.equal(f.body().managed(f.node('sprite')),false,'retired controller cannot reserve the rig canvas again');
  f.emit(target,'pointerup',{pointerId:3});assert.equal(target.hidden,true);assert.equal(f.body().capturing(),false);assert.equal(f.nav.filter(url=>url.endsWith('gesture-release')).length,1);
  f.close();await flush();
});

test('wake body phase uses its own successful frame and retains proxy until completion; hidden/close cannot resurrect it',async()=>{
  const f=fixture({sleepEnabled:true});await f.start();await f.fire(1500);await enter(f);await f.frame(f.now+1400);
  f.setTask({Rest:{Sleeping:false}});await f.fire(1500);await f.frame(f.now+550);assert.equal(proxy(f).hidden,false);assert.match(f.node('sprite').style.transform,/matrix\(/);
  f.document.hidden=true;const before=visibleDraws(f);await f.frame(f.now+1500);assert.equal(visibleDraws(f),before);
  f.document.hidden=false;await f.frame();assert.equal(proxy(f).hidden,true);assert.equal(f.node('sprite').style.pointerEvents,'');
  f.setTask({Rest:{Sleeping:true}});await f.fire(1500);const gate=f.gateFrame(),pending=f.frame();await flush();f.close();gate.resolve();await pending;assert.equal(proxy(f).hidden,true);assert.equal(f.rafs.size,0);
});
