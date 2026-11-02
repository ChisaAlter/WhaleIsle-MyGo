import assert from 'node:assert/strict';
import test from 'node:test';
import {readFileSync} from 'node:fs';
import path from 'node:path';
import {asset, baseline, fixture, flush, plain} from './body-fixture.mjs';

const modes = baseline ? [{name:'pre-change',old:true},{name:'body unavailable',bodyScript:false},{name:'shared body',bodyScript:true}] : [{name:'body unavailable',bodyScript:false},{name:'shared body',bodyScript:true}];
async function compare(options, run) {
  const results = [];
  for (const mode of modes) {
    const f = fixture({...options,...mode}); const detail = await run(f); results.push({mode:mode.name,observation:f.observations(),detail:plain(detail || {})});
  }
  for (const result of results.slice(1)) {
    assert.deepEqual(result, {...results[0],mode:result.mode}, result.mode + ' must preserve the same source-driven observed output');
  }
  return results.at(-1);
}

// Every branch is read from the real shipped document; no fixture-only order.
test('body script is a single local defer dependency before every consumer; all markup/CSS stays unchanged', () => {
  const html = asset('pet.html');
  const scripts = [...html.matchAll(/<script\b([^>]*)><\/script>/g)].map(match => ({attributes:match[1],src:/src="([^"]+)"/.exec(match[1])?.[1]}));
  const body = scripts.findIndex(tag => tag.src === 'pet-body.js');
  assert.equal(scripts.filter(tag => tag.src === 'pet-body.js').length, 1);
  assert.ok(body >= 0);
  for (const name of ['pet-care-animation.js','pet-drop.js','pet-status.js','pet-interaction.js','pet-behavior.js','pet-layout.js','pet-runtime.js']) {
    const index = scripts.findIndex(tag => tag.src === name);
    assert.ok(index > body, name + ' must load after shared body');
    assert.match(scripts[index].attributes, /\bdefer\b/);
  }
  assert.match(scripts[body].attributes, /\bdefer\b/);
  assert.equal(/https?:|nonce=|async/.test(scripts[body].attributes), false);
  if (baseline) {
    const oldHTML = readFileSync(path.join(baseline, 'pet.html'), 'utf8');
    const withoutSleep=html.replace('<script src="pet-sleep.js" defer></script>','').replace('<div id="pet-body-hit" hidden aria-label="Desktop pet body"></div>','').replace('#pet-body-hit:hover~#pet-status-toggle,','').replace(/#pet-body-hit\{position:fixed;background:transparent;--app-region:no-drag;touch-action:none\}\r?\n/,'');
    assert.equal(withoutSleep.replace('<script src="pet-body.js" defer></script>', ''), oldHTML, 'all prior HTML/CSS bytes except the precise sleep target/style/script additions stay unchanged');
  }
});

test('body access reads lazily, keeps both visible renderers and native anchors, and uses element identity', () => {
  const f = fixture(), body = f.body(), sprite = f.node('sprite'), fallback = f.node('fallback');
  assert.deepEqual(f.rectReads, {}, 'creating body must not measure layout');
  assert.equal(body.active(), fallback);
  sprite.hidden = false; assert.equal(body.active(), sprite);
  fallback.hidden = false; assert.equal(body.contains(120,180), true);
  assert.equal(body.contains(360,440), true, 'old inclusive edges retained');
  assert.equal(body.contains(361,440), false);
  const layout = {X:10,Y:20,Width:30,Height:40}; assert.equal(body.anchorRect(layout), layout);
  assert.equal(body.matches(sprite), true);
  assert.equal(body.matches({id:'sprite',isConnected:true}), false, 'same id cannot steal another create instance');
  const local = f.window.WhalePetBody.create({characters:[null]});
  assert.equal(local.active(), null); assert.equal(local.rect(), null); assert.equal(local.contains(0,0), false); assert.equal(local.matches(sprite), false);
  sprite.hidden = fallback.hidden = true; assert.equal(body.active(), sprite, 'existing first-hidden fallback preserved for decoration/gaze');
});

for (const renderer of ['neural','rig','sprite']) test('runtime/interaction/care/layout keep ' + renderer + ' rendering, hit timing, wake and close output', async () => {
  const result = await compare({renderer}, async f => {
    await f.start();
    const before = f.observations();
    assert.equal(f.document.body.dataset.renderer, renderer);
    assert.equal(f.node('sprite').hidden, false); assert.equal(f.node('fallback').hidden, true);
    assert.equal(f.rectReads.sprite || 0, renderer === 'sprite' ? 1 : 0, 'only the pre-existing startup poll may read the body; differential observations also compare every BCR count');
    await f.fire(1500); await f.frame(1050); await f.fire(250);
    const hit = () => f.nav.filter(url => url.includes('interactive-regions?')).map(url=>JSON.parse(decodeURIComponent(url.split('data=')[1])));
    assert.equal(hit().at(-1).Epoch, 7);
    assert.equal(hit().at(-1).Regions[0].Width, 240);
    // Both visible elements are separate islands, and display:none is excluded.
    f.node('fallback').hidden = false; f.node('fallback').box = {x:500,y:200,width:120,height:130};
    await f.fire(250); assert.equal(hit().at(-1).Regions.length, 4);
    f.node('sprite').style.display = 'none'; await f.fire(250); assert.equal(hit().at(-1).Regions[0].X, 500);
    f.node('sprite').style.display = ''; f.node('fallback').hidden = true;
    f.now = 2000; f.setHover(f.node('sprite')); await f.fire(1000);
    assert.equal(f.nav.at(-1), 'whale-pet://localhost/motion-contact');
    f.setTask({Rest:{Sleeping:true}}); await f.fire(1500);
    const careStyle = [f.node('sprite').style.rotate,f.node('sprite').style.scale]; assert.deepEqual(careStyle,['18deg','1 .82']);
    const beforeWake = f.nav.length;
    f.emit(f.node('pet-status-toggle'),'pointerenter'); assert.equal(f.nav.length,beforeWake,'controls never wake sleeping pet');
    f.emit(f.node('sprite'),'pointerenter'); assert.equal(f.nav.at(-1),'whale-pet://localhost/wake-contact');
    f.setTask({Rest:{Sleeping:false}}); await f.fire(1500); await f.frame(2100);
    assert.equal(f.node('sprite').style.rotate,''); assert.equal(f.node('sprite').style.scale,'');
    const preClose = f.nav.length; f.close(); await flush();
    f.emit(f.node('sprite'),'pointerenter'); assert.equal(f.nav.length,preClose,'closed runtime cannot request wake');
    assert.equal(f.rafs.size,0); assert.equal([...f.timers.values()].filter(t=>[50,120,1500,250,1000].includes(t.ms)).length,0);
    return {hit:hit(),careStyle,initialSize:[before.dataset.renderer,before.hidden.sprite],finalSize:[f.node('sprite').width,f.node('sprite').height]};
  });
  assert.ok(result.detail.hit.length>=3);
});

for (const options of [{failScripts:['tha4.js']},{failScripts:['tha4.js','pet-rig.js']},{failScripts:['tha4.js','pet-rig.js'],failImages:['sprite.webp']},{failScripts:['tha4.js','pet-rig.js'],failImages:['sprite.webp','fallback.png']}]) test('source failures preserve renderer fallback: ' + JSON.stringify(options), async () => {
  await compare(options, async f => {
    await f.start(); await f.fire(1500); await f.frame(); await f.fire(250);
    const visible = !f.node('sprite').hidden ? 'sprite' : 'fallback';
    f.emit(f.node(visible),'pointerdown'); f.emit(f.node(visible),'pointerup');
    assert.deepEqual(f.captures.slice(-2).map(x=>x[0]),['set','release']);
    const result = {renderer:f.document.body.dataset.renderer || '',visible,notice:f.node('render-notice').textContent}; f.close(); await flush(); return result;
  });
});

test('late neural failure returns to the real rig without changing body targets, drawing or hit reports', async () => {
  await compare({}, async f => {
    await f.start(); await f.fire(1500); await f.frame();
    f.failFrame(); await f.frame();
    assert.equal(f.document.body.dataset.renderer,'rig');
    await f.fire(250); f.emit(f.node('sprite'),'pointerdown'); f.emit(f.node('sprite'),'pointerup'); f.close(); await flush();
  });
});

test('interaction retains top-45% pat thresholds, pointer target capture, drag settings, release/cancel/blur and late close', async () => {
  await compare({}, async f => {
    f.load('pet-interaction.js'); const sprite=f.node('sprite'), fallback=f.node('fallback'); sprite.hidden=false; fallback.hidden=true;
    let taps=0; const settings={lockPosition:false,shiftToDrag:false};
    const c=f.window.WhalePetInteraction.create({characters:[sprite,fallback],body:f.body(),getSettings:()=>settings,isSleeping:()=>false,onTap:()=>taps++});
    await flush();
    // Three direction flips, including the first positive stroke, trigger pat.
    for(const [i,x] of [140,160,140,160].entries()){f.now=1100+i*100;f.emit(sprite,'pointermove',{clientX:x,clientY:200,buttons:0});}
    assert.equal(f.nav.filter(url=>url.endsWith('/pat-pet')).length,1);
    f.now=5000; for(const x of [160,140,160,140])f.emit(sprite,'pointermove',{clientX:x,clientY:400,buttons:0});
    assert.equal(f.nav.filter(url=>url.endsWith('/pat-pet')).length,1,'lower body strokes cannot pat');
    f.emit(sprite,'pointerdown'); f.emit(sprite,'pointerup'); assert.equal(taps,1);
    settings.lockPosition=true; f.emit(sprite,'pointerdown'); f.emit(sprite,'pointermove',{screenX:200}); f.emit(sprite,'pointerup'); assert.equal(taps,2,'locked movement retains prior tap semantics');
    settings.lockPosition=false; settings.shiftToDrag=true;
    f.emit(sprite,'pointerdown'); f.emit(sprite,'pointermove',{screenX:200}); f.emit(sprite,'pointermove',{screenX:200,shiftKey:true}); f.emit(sprite,'pointerup'); assert.equal(taps,2);
    f.emit(sprite,'pointerdown'); f.emit(sprite,'pointercancel'); f.emit(sprite,'pointerup');
    f.emit(sprite,'pointerdown'); f.emit(f.window,'blur'); f.emit(sprite,'pointerup'); assert.equal(taps,2);
    f.emit(sprite,'pointerdown',{button:2}); f.emit(sprite,'pointerup'); assert.equal(taps,2);
    // Capture remains owned by the pressed DOM element when renderer visibility changes.
    f.emit(sprite,'pointerdown'); sprite.hidden=true; fallback.hidden=false;
    f.emit(sprite,'pointermove',{screenX:200,shiftKey:true}); f.emit(sprite,'pointerup');
    assert.equal(f.captures.at(-1)[1],'sprite');
    f.emit(fallback,'pointerdown'); f.emit(fallback,'pointerup'); assert.equal(taps,3);
    await f.frame();
    const pending=f.gateInteraction(); const poll=[...f.timers.values()].find(t=>t.ms===50).fn(); await flush();
    f.emit(fallback,'pointerdown'); c.close(); const before=f.nav.length; pending.resolve(); await poll; await flush();
    f.emit(fallback,'pointerup'); f.emit(sprite,'pointermove'); await f.frame(); assert.equal(f.nav.length,before);
    assert.equal(fallback.captured.has(1),true,'close keeps existing DOM disposal capture ownership; it must not synthesize a release/action');
    return {taps,state:c.renderState()};
  });
});

test('status right-click, per-target 600ms long press, cancellation, focus and outside/Escape behavior stay unchanged', async () => {
  await compare({}, async f => {
    f.load('pet-status.js'); const sprite=f.node('sprite'), fallback=f.node('fallback'), card=f.node('pet-status-card'), toggle=f.node('pet-status-toggle');
    assert.equal(f.emit(sprite,'contextmenu').defaultPrevented,true); assert.equal(card.hidden,false);
    f.emit(f.document,'keydown',{key:'Escape'}); assert.equal(card.hidden,true);
    f.emit(fallback,'pointerdown'); assert.equal(card.hidden,true); await f.fire(600); assert.equal(card.hidden,false);
    f.emit(f.node('pet-status-close'),'click'); assert.equal(card.hidden,true);
    for(const event of ['pointerup','pointerleave','pointercancel']){f.emit(sprite,'pointerdown');f.emit(sprite,event);assert.equal([...f.timers.values()].some(t=>t.ms===600),false);}
    f.document.activeElement=toggle;f.emit(toggle,'click');assert.equal(f.document.activeElement.id,'pet-status-close');
    f.emit(f.document,'pointerdown',{target:f.node('feed-file')});assert.equal(card.hidden,false);
    f.emit(f.node('pet-status-close'),'click');assert.equal(f.document.activeElement,toggle);
    f.emit(fallback,'contextmenu'); f.emit(f.document,'pointerdown',{target:f.document.body});assert.equal(card.hidden,true);
    const listeners=Object.fromEntries([...sprite.handlers].map(([name,list])=>[name,list.length]));
    f.close(); return {listeners};
  });
});

test('file drag/drop keeps inclusive visible BCR checks and metadata-only handling', async () => {
  await compare({}, async f => {
    f.load('pet-drop.js'); const sprite=f.node('sprite'), fallback=f.node('fallback'); sprite.hidden=false; fallback.hidden=true;
    const file={name:'鲸鱼 <b>雪</b>.txt',get path(){throw Error('path read');},arrayBuffer(){throw Error('content read');},text(){throw Error('content read');}};
    const transfer={types:['Files'],files:[file]};
    assert.equal(f.emit(f.document,'dragover',{clientX:120,clientY:180,dataTransfer:transfer}).defaultPrevented,true); assert.equal(transfer.dropEffect,'copy');
    assert.equal(f.emit(f.document,'drop',{clientX:360,clientY:440,dataTransfer:transfer}).defaultPrevented,true);assert.match(f.nav.at(-1),/feed-drop\?name=/);
    for(const extra of [{clientX:361,clientY:440,dataTransfer:transfer},{clientX:0,clientY:0,dataTransfer:transfer},{clientX:140,clientY:200,dataTransfer:{types:['text/plain'],files:[]}}])assert.equal(f.emit(f.document,'drop',extra).defaultPrevented,false);
    sprite.hidden=true;fallback.hidden=false;fallback.box={x:500,y:200,width:120,height:130};
    assert.equal(f.emit(f.document,'drop',{clientX:500,clientY:200,dataTransfer:transfer}).defaultPrevented,true);
    fallback.hidden=true;assert.equal(f.emit(f.document,'drop',{clientX:500,clientY:200,dataTransfer:transfer}).defaultPrevented,false);
    f.close();
  });
});

test('care decoration/WAAPI/pose and native layout anchors do not change after renderer visibility or CSS changes', async () => {
  await compare({}, async f => {
    f.load('pet-care-animation.js');f.load('pet-layout.js');const sprite=f.node('sprite'),fallback=f.node('fallback');
    sprite.hidden=false;fallback.hidden=true;
    const care=f.window.WhalePetCareAnimation.create({characters:[sprite,fallback],body:f.body()});
    for(const [i,kind] of ['pat','file','token'].entries()){
      const action={ID:f.scope.Date.now()+i,Kind:kind,Name:'本地.txt',Level:kind==='token'?2:0};
      care.update({Care:{Action:action}});const count=f.observations().animation.length;care.update({Care:{Action:action}});assert.equal(f.observations().animation.length,count,'ID dedup retained');
      const pose=new Float32Array(45);care.pose(pose);assert.ok(pose.every(Number.isFinite));await f.fire(2800);if(kind==='token')await f.fire(2200);
    }
    sprite.hidden=true;fallback.hidden=false;fallback.box={x:500,y:400,width:120,height:130};
    care.update({Rest:{Dozing:true}});assert.equal(fallback.style.rotate,'7deg');
    care.update({Rest:{Sleeping:true},Care:{Action:{ID:f.scope.Date.now()+20,Kind:'pat'}}});assert.equal(fallback.style.rotate,'18deg');
    const pose=new Float32Array(45);care.pose(pose);assert.equal(pose[12],1);assert.equal(pose[40],Math.fround(.15));
    f.node('pet-status-card').hidden=false;f.node('pet-status-card').textContent='status';f.node('pet-status-card').box={x:0,y:0,width:248,height:360};
    const layout={X:300,Y:500,Width:240,Height:260,Area:{X:0,Y:0,Width:1200,Height:800}};
    const controller=f.window.WhalePetLayout.create({body:f.body()});controller.update(layout);
    assert.equal(f.node('pet-status-card').style.left,'296px');assert.equal(f.node('pet-status-card').style.top,'132px','anchor stays native nominal layout even though visible fallback BCR differs');
    care.close();f.close();assert.equal(fallback.style.rotate,'');assert.equal(fallback.style.scale,'');
    return {pose:Array.from(pose)};
  });
});

test('behavior hover/focus/power saving and self-talk suppression keep the same live-state ownership', async () => {
  await compare({}, async f => {
    f.load('pet-behavior.js'); const sprite=f.node('sprite'), bubble=f.node('pet-bubble');
    let sleeping=false; const c=f.window.WhalePetBehavior.create({body:f.body(),bubble,getSettings:()=>({powerSave:true,selfTalk:true}),getStatus:()=>({Rest:{Sleeping:sleeping}}),isSleeping:()=>sleeping});
    await flush();f.now=301001;assert.equal(c.powerSaving(f.now),true);
    f.emit(f.document,'pointerover',{target:sprite});assert.equal(c.powerSaving(f.now+400000),false);
    f.emit(f.document,'pointerout',{target:sprite,relatedTarget:f.document.body});f.now+=301000;assert.equal(c.powerSaving(f.now),true);
    f.document.activeElement=f.node('feed-file');assert.equal(c.powerSaving(f.now),false);f.document.activeElement=f.document.body;
    sleeping=true;await f.fire(90000);assert.equal(bubble.hidden,true);
    f.close();assert.equal(c.powerSaving(f.now+600000),false);
  });
});

test('no body elements, custom create arrays and zero-size gaze are finite and independent of global current', async () => {
  for(const bodyScript of [true,false]) {
    const f=fixture({bodyScript,empty:true});f.load('pet-interaction.js');f.load('pet-care-animation.js');f.load('pet-drop.js');f.load('pet-status.js');
    const interaction=f.window.WhalePetInteraction.create({characters:[null],body:f.body(),getSettings:()=>({}),isSleeping:()=>false});
    const care=f.window.WhalePetCareAnimation.create({characters:[null],body:f.body()});
    await flush();f.emit(f.document,'pointermove',{clientX:0,clientY:0});care.update({Rest:{Sleeping:true}});await f.frame();
    const pose=new Float32Array(45);interaction.pose(pose);assert.ok(pose.every(Number.isFinite));care.close();interaction.close();
    assert.equal(f.emit(f.document,'drop',{dataTransfer:{files:[{name:'x'}]}}).defaultPrevented,false);
    const custom=f.node('custom');custom.hidden=false;custom.box={x:0,y:0,width:0,height:0};
    const privateInteraction=f.window.WhalePetInteraction.create({characters:[custom],getSettings:()=>({}),isSleeping:()=>false});
    const privateCare=f.window.WhalePetCareAnimation.create({characters:[custom]});await flush();
    f.emit(custom,'pointermove',{clientX:0,clientY:0});await f.frame();privateInteraction.pose(pose);assert.ok(pose.every(Number.isFinite),'zero-size gaze cannot poison neural pose');
    privateCare.update({Rest:{Dozing:true}});assert.equal(custom.style.rotate,'7deg');assert.equal(f.node('sprite').style.rotate,undefined);
    f.emit(custom,'pointerdown');f.emit(custom,'pointerup');assert.equal(f.captures.at(-1)[1],'custom');privateInteraction.close();privateCare.close();
  }
});
