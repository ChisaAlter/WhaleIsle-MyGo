"use strict";
const {test}=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),os=require('node:os'),path=require('node:path'),z=require('node:zlib');
const {createWatcher}=require('./watch.cjs');
test('daily greeting and gateway milestone use current source once per day',()=>{
 const home=fs.mkdtempSync(path.join(os.tmpdir(),'mygo-pet-greeting-'));
 try{const dir=path.join(home,'sessions','project','s');fs.mkdirSync(dir,{recursive:true});const file=path.join(dir,'session.v4.jsonl.zstd'),usage=path.join(home,'gateway.jsonl');
 const frame=rows=>z.zstdCompressSync(Buffer.from(rows.map(r=>JSON.stringify(r)).join('\n')+'\n'));
 fs.writeFileSync(file,frame([{type:'session',id:'s'}]));let now=Date.parse('2026-11-02T12:00:00Z');const watcher=createWatcher(home,()=>now,'',usage);assert.equal(watcher.poll().events.length,0);
 fs.writeFileSync(usage,JSON.stringify({agent:'dsh',route_id:'one',in:100000,out:1,t:new Date(now).toISOString()})+'\n');
 fs.appendFileSync(file,frame([{seq:1,type:'turn/start',data:{turn:1}}]));let snap=watcher.poll();assert.equal(snap.events.filter(e=>e.type==='greet').length,1);assert.equal(snap.events.filter(e=>e.type==='milestone').length,1);
 assert.equal(watcher.poll().events.length,snap.events.length);for(const e of snap.events)watcher.ack(e.id);
 now+=86400000;snap=watcher.poll();assert.equal(snap.events.filter(e=>e.type==='greet').length,1);assert.equal(snap.events.filter(e=>e.type==='milestone').length,0);
 now+=4*86400000;snap=watcher.poll();assert.equal(snap.events.filter(e=>e.type==='miss').length,1);
 }finally{fs.rmSync(home,{recursive:true,force:true})}
});
