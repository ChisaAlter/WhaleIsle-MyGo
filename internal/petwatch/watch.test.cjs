'use strict';
// One bounded fixture, exclusively temporary synthetic files. Never starts Host.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const zlib = require('node:zlib');
const { createWatcher } = require('./watch.cjs');
test('all-session tails, durable acknowledgement, torn writes, restart and boundaries', () => {
  const home = fs.mkdtempSync(path.join(os.tmpdir(), 'mygo-pet-events-'));
  const frame = rows => zlib.zstdCompressSync(Buffer.from(rows.map(x=>JSON.stringify(x)).join('\n')+'\n'));
  const event = (seq,type,data={}) => ({seq,type,data});
  const session = id => {
    const dir=path.join(home,'sessions','project',id);fs.mkdirSync(dir,{recursive:true});
    const file=path.join(dir,'session.v4.jsonl.zstd');
    fs.writeFileSync(file,frame([{type:'session',id,version:4}]));return file;
  };
  try {
    const a=session('a'), b=session('b');
    fs.appendFileSync(a,frame([event(1,'turn/end',{reason:{kind:'completed'}})]));
    const whale=path.join(home,'data','whale');fs.mkdirSync(whale,{recursive:true});
    const outbox=path.join(whale,'pet-outbox.jsonl');
    fs.writeFileSync(outbox,JSON.stringify({kind:'notify',text:'old history'})+'\n');
    let now=1800000000000;
    let w=createWatcher(home,()=>now);
    assert.equal(w.poll().events.length,0,'first run establishes baseline only');
    fs.appendFileSync(a,frame([event(2,'turn/start',{turn:1}),event(3,'turn/end',{reason:{kind:'completed'}})]));
    const tail=frame([event(1,'turn/start',{turn:1}),event(2,'turn/end',{reason:{kind:'failed',message:'SECRET'}})]);
    fs.appendFileSync(b,tail.subarray(0,tail.length-3));
    let r=w.poll();assert(r.events.some(e=>e.type==='done'&&e.sessionID==='a'));
    assert(!r.events.some(e=>e.type==='error'));
    fs.appendFileSync(b,tail.subarray(tail.length-3));
    fs.appendFileSync(outbox,'{"kind":"notify","text":"<img src=x onerror=alert(1)>');
    r=w.poll();assert(r.events.some(e=>e.type==='error'&&e.sessionID==='b'));
    assert(!JSON.stringify(r).includes('SECRET'));
    assert(!r.events.some(e=>e.type==='whale'));
    fs.appendFileSync(outbox,'"}\n');
    r=w.poll();const whaleEvent=r.events.find(e=>e.type==='whale');assert(whaleEvent&&whaleEvent.pinned);
    const count=r.events.length;assert.equal(w.poll().events.length,count,'unchanged tails never re-emit');
    w=createWatcher(home,()=>now);assert.equal(w.poll().events.length,count,'unacknowledged queue survives restart');
    assert.equal(w.ack('invented'),false);assert.equal(w.ack(whaleEvent.id),true);
    w=createWatcher(home,()=>now);assert(!w.poll().events.some(e=>e.id===whaleEvent.id));
    const c=session('c');fs.appendFileSync(c,frame([event(1,'assistant/message',{turn:1,step:1,usage:{inputTokens:100000},content:'SECRET2'}),event(2,'turn/end',{reason:{kind:'completed'}})]));
    r=w.poll();assert(r.events.some(e=>e.type==='milestone'));assert(r.events.some(e=>e.type==='done'&&e.sessionID==='c'));assert(!JSON.stringify(r).includes('SECRET2'));
    const before=r.events.length;fs.appendFileSync(c,frame([event(2,'turn/end',{reason:{kind:'completed'}})]));assert.equal(w.poll().events.length,before,'sequence dedup');
    const escape=path.join(home,'outside.jsonl.zstd');fs.writeFileSync(escape,frame([event(9,'turn/end',{reason:{kind:'completed'}})]));
    fs.symlinkSync(escape,path.join(home,'sessions','project','a','session.v5.jsonl.zstd'));
    assert.equal(w.poll().events.length,before,'symlink source rejected');
    fs.truncateSync(outbox,0);w.poll();fs.appendFileSync(outbox,'{"kind":"say","text":"new after truncation"}\n');assert(w.poll().events.some(e=>e.text==='new after truncation'));
    now+=46*60000;fs.appendFileSync(a,frame([event(4,'turn/start',{turn:2})]));w.poll();now+=46*60000;assert(w.poll().events.some(e=>e.type==='rest'));
    const stickers=path.join(whale,'stickers');fs.mkdirSync(stickers);
    const png=path.join(stickers,'safe.png');fs.writeFileSync(png,Buffer.from('89504e470d0a1a0a','hex'));
    fs.appendFileSync(outbox,JSON.stringify({kind:'sticker',text:'safe sticker',path:png})+'\n');
    assert.equal(w.poll().events.find(e=>e.text==='safe sticker').mediaPath,png);
    fs.appendFileSync(outbox,JSON.stringify({kind:'sticker',text:'outside',path:escape})+'\n');
    assert.equal(w.poll().events.find(e=>e.text==='outside').mediaPath,'');
    const linked=path.join(stickers,'linked.png');fs.symlinkSync(escape,linked);
    fs.appendFileSync(outbox,JSON.stringify({kind:'sticker',text:'linked',path:linked})+'\n');
    assert.equal(w.poll().events.find(e=>e.text==='linked').mediaPath,'');
    const html=path.join(stickers,'unsafe.svg');fs.writeFileSync(html,'<svg onload="alert(1)"/>');
    fs.appendFileSync(outbox,JSON.stringify({kind:'sticker',text:'vector',path:html})+'\n');
    assert.equal(w.poll().events.find(e=>e.text==='vector').mediaPath,'');
    for(const e of w.snapshot().events)w.ack(e.id);
    const d=session('d');fs.appendFileSync(d,frame(Array.from({length:300},(_,i)=>event(i+1,'turn/end',{reason:{kind:'completed'}}))));
    const first=w.poll().events.filter(e=>e.sessionID==='d');assert.equal(first.length,256,'bounded queue');
    for(const e of first)w.ack(e.id);
    const second=w.poll().events.filter(e=>e.sessionID==='d');assert.equal(second.length,44,'large complete frame resumes without dropping or repeating');
    assert.equal(new Set([...first,...second].map(e=>e.id)).size,300);
    const oldSize=fs.statSync(outbox).size;
    fs.writeFileSync(outbox,JSON.stringify({kind:'say',text:'same-inode replacement',padding:'x'.repeat(oldSize)})+'\n');
    assert(w.poll().events.some(e=>e.text==='same-inode replacement'),'same-inode rewrite beyond prior size is not lost');



  } finally { fs.rmSync(home,{recursive:true,force:true}); }
});
