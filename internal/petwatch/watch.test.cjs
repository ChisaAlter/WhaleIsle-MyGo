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
  } finally { fs.rmSync(home,{recursive:true,force:true}); }
});
