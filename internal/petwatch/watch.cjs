'use strict';
// Structural tail reader adapted from ChisaAlter/WhaleIsle@90465b67,
// src/main/pet-dsh-watch.js and pet-growth.js (MIT, copyright 2026 Chisa).
// No Host RPC, network, model, shell, command arguments or message bodies leave
// this worker. Only fixed status text and the explicit Whale outbox text do.
const fs=require('node:fs'), path=require('node:path'), zlib=require('node:zlib');
const {createHash}=require('node:crypto');
const MAX=8*1024*1024, QUEUE=256;
const hash=x=>createHash('sha256').update(x).digest('hex').slice(0,32);
const clone=x=>JSON.parse(JSON.stringify(x));
const day=t=>new Date(t).toLocaleDateString('en-CA');
const object=()=>Object.create(null);
function segment(raw) {
  if(raw==='.'||raw==='..')return [...raw].map(()=> '~002E').join('');
  return raw.split('').map(c=>/^[A-Za-z0-9._-]$/.test(c)?c:'~'+c.charCodeAt(0).toString(16).toUpperCase().padStart(4,'0')).join('');
}
function within(root,file) {const r=path.relative(root,file);return !path.isAbsolute(r)&&r!=='..'&&!r.startsWith('..'+path.sep);}
function safe(root,file,dir=false) {
  if(!within(root,file))throw Error('source boundary');
  let p=root;
  for(const part of path.relative(root,file).split(path.sep).filter(Boolean)) {
    p=path.join(p,part);const s=fs.lstatSync(p);
    if(s.isSymbolicLink())throw Error('source symlink');
  }
  const s=fs.lstatSync(file);
  if(s.isSymbolicLink()||!(dir?s.isDirectory():s.isFile())||!within(root,fs.realpathSync(file)))throw Error('source type');
  return s;
}
function read(fd,offset,length) {
  const b=Buffer.alloc(length);const n=fs.readSync(fd,b,0,length,offset);return b.subarray(0,n);
}
// Walk physical frame headers without decoding historical event bodies.
function frameEnd(fd,start,size) {
  const h=read(fd,start,18);if(h.length<4)return 0;
  const magic=h.readUInt32LE(0);
  if((magic&0xfffffff0)===0x184d2a50){if(h.length<8)return 0;const end=start+8+h.readUInt32LE(4);return end<=size?end:0;}
  if(magic!==0xfd2fb528||h.length<5)return 0;
  const f=h[4];if(f&8)return 0;
  const single=(f>>5)&1, flag=f>>6;
  let pos=start+5+(single?0:1)+[0,1,2,4][f&3]+(flag===0?(single?1:0):[0,2,4,8][flag]);
  for(let blocks=0;blocks<100000;blocks++) {
    if(pos+3>size)return 0;const b=read(fd,pos,3);if(b.length!==3)return 0;
    const v=b[0]|b[1]<<8|b[2]<<16, type=(v>>1)&3;if(type===3)return 0;
    pos+=3+(type===1?1:v>>3);if(pos>size)return 0;
    if(v&1){pos+=(f&4)?4:0;return pos<=size?pos:0;}
  }
  return 0;
}
function rows(fd,start,end) {
  if(end-start>MAX)throw Error('frame limit');
  const raw=read(fd,start,end-start);if(raw.readUInt32LE(0)!==0xfd2fb528)return [];
  const text=zlib.zstdDecompressSync(raw,{maxOutputLength:MAX}).toString('utf8');
  return text.split('\n').filter(x=>x.startsWith('{')).flatMap(x=>{try{return [JSON.parse(x)];}catch{return [];}});
}
function list(root) {
  const base=path.join(root,'sessions'), files=[];
  try {safe(root,base,true);}catch(e){if(e.code==='ENOENT')return files;throw e;}
  const stack=[[base,0]];let dirs=0;
  while(stack.length){const [dir,depth]=stack.pop();if(++dirs>25000)throw Error('directory limit');
    safe(root,dir,true);
    const entries=fs.readdirSync(dir,{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name));
    const logs=entries.filter(e=>e.isFile()&&/^session(?:\.v\d+)?\.jsonl\.zstd$/.test(e.name));
    // The current backend selects the highest physical generation; old format
    // copies must not be replayed as independent live conversations.
    logs.sort((a,b)=>(Number(b.name.match(/\.v(\d+)/)?.[1])||0)-(Number(a.name.match(/\.v(\d+)/)?.[1])||0));
    if(logs[0])files.push(path.join(dir,logs[0].name));
    for(const e of entries)if(e.isDirectory()&&depth<3)stack.push([path.join(dir,e.name),depth+1]);
    if(files.length>20000)throw Error('file limit');
  }return files.sort();
}
function createWatcher(home,now=Date.now) {
  if(!path.isAbsolute(home)||typeof zlib.zstdDecompressSync!=='function')throw Error('runtime unavailable');
  const st=fs.lstatSync(home);if(!st.isDirectory()||st.isSymbolicLink())throw Error('invalid home');
  const root=fs.realpathSync(home), ledger=path.join(root,'mygo-pet-events.json'), outbox=path.join(root,'data','whale','pet-outbox.jsonl');
  let state;
  try {safe(root,ledger);const raw=fs.readFileSync(ledger);if(raw.length>16*MAX)throw Error('checkpoint limit');state=JSON.parse(raw);
    if(state.version!==1||state.root!==root||!state.files||!state.sessions||!Array.isArray(state.pending)||state.pending.length>QUEUE)throw Error('invalid checkpoint');
  }catch(e){if(e.code!=='ENOENT')throw e;state={version:1,root,initialized:false,files:object(),sessions:object(),pending:[],used:0,marks:[],day:'',lastActive:0,lastSeen:0,activeSince:0,lastRest:0,lastGreet:''};}
  function save(next){
    const tmp=ledger+'.tmp-'+process.pid;
    const fd=fs.openSync(tmp,'wx',0o600);
    try{fs.writeFileSync(fd,JSON.stringify(next));fs.fsyncSync(fd);}finally{fs.closeSync(fd);}
    try{fs.renameSync(tmp,ledger);}catch(e){try{fs.unlinkSync(tmp);}catch{}throw e;}
    state=next;
  }
  function emit(s,id,type,text,extra={}){
    if(s.pending.some(e=>e.id===id))return;
    if(s.pending.length>=QUEUE)throw Error('queue full');
    s.pending.push({id,type,text,...extra});
  }
  function usage(ev){
    const d=ev.data||{};let u;
    if(ev.type==='assistant/message'||ev.type==='compaction/summary')u=d.usage;
    else if(ev.type==='assistant/attempt'||ev.type==='assistant/chunk')u=d.chunk?.type==='usage'?d.chunk.usage:Array.isArray(d.stream)?[...d.stream].reverse().find(c=>c?.type==='usage')?.usage:undefined;
    if(!u)return null;
    let total=0;for(const k of ['inputTokens','cacheReadTokens','cacheWriteTokens','outputTokens']){const n=u[k]??0;if(!Number.isSafeInteger(n)||n<0||!Number.isSafeInteger(total+n))return null;total+=n;}
    const turn=ev.type==='compaction/summary'?-1:d.turn,step=ev.type==='compaction/summary'?ev.seq:d.step;
    return Number.isSafeInteger(turn)&&Number.isSafeInteger(step)?{slot:turn+':'+step,total}:null;
  }
  function feed(s,ev,id,t){
    if(!Number.isSafeInteger(ev.seq)||ev.seq<0)return;
    const key=hash(id), sess=s.sessions[key]||(s.sessions[key]={id,seq:-1,open:false,usage:object()});
    if(ev.seq<=sess.seq)return;sess.seq=ev.seq;
    const d=ev.data||{},eid=hash(id+':'+ev.seq);
    if(ev.type==='turn/start'){
      sess.open=true;s.lastActive=t;emit(s,eid,'working','有会话开始工作了。',{sessionID:id});
    }else if(ev.type==='turn/end'){
      sess.open=false;s.lastActive=t;const done=d.reason?.kind==='completed';
      emit(s,eid,done?'done':'error',done?'这一轮完成了。':'这一轮未完成，请打开会话查看。',{sessionID:id});
    }else if(['user/message','tool/call','step/start','assistant/chunk'].includes(ev.type))s.lastActive=t;
    const sample=usage(ev);
    if(sample){const prev=sess.usage[sample.slot]||0;if(sample.total>prev){s.used+=sample.total-prev;sess.usage[sample.slot]=sample.total;}
      for(const mark of [100000,500000,1000000,5000000,10000000])if(s.used>=mark&&!s.marks.includes(mark)){
        emit(s,hash(s.day+':'+mark),'milestone','今天的工作用量达到 '+mark.toLocaleString('en-US')+' token。');s.marks.push(mark);
      }
    }
    s.lastSeen=t;
  }
  function sessionFile(s,file,initial,t){
    const stat=safe(root,file), rel=path.relative(root,file), key=hash(rel), identity=String(stat.dev)+':'+stat.ino+':'+stat.birthtimeMs;
    const fd=fs.openSync(file,'r');
    try{
      const opened=fs.fstatSync(fd);if(opened.dev!==stat.dev||opened.ino!==stat.ino)throw Error('source changed');
      const first=frameEnd(fd,0,stat.size);if(!first)return;
      const header=rows(fd,0,first)[0];if(header?.type!=='session'||typeof header.id!=='string'||!header.id||header.id.length>512||segment(header.id)!==path.basename(path.dirname(file)))return;
      let c=s.files[key];
      if(!c||c.identity!==identity||stat.size<c.offset)c={identity,offset:0};
      if(initial){
        let off=0,last=0,end;while((end=frameEnd(fd,off,stat.size))){last=off;off=end;}
        const sess=s.sessions[hash(header.id)]||(s.sessions[hash(header.id)]={id:header.id,seq:-1,open:false,usage:object()});
        if(off>last)for(const ev of rows(fd,last,off))if(Number.isSafeInteger(ev.seq))sess.seq=Math.max(sess.seq,ev.seq);
        s.files[key]={identity,offset:off,sessionID:header.id};return;
      }
      let budget=0;
      while(c.offset<stat.size&&budget<MAX){
        const end=frameEnd(fd,c.offset,stat.size);if(!end)break;
        const next=clone(s);
        try{for(const ev of rows(fd,c.offset,end))feed(next,ev,header.id,t);}catch(e){if(e.message==='queue full')break;throw e;}
        budget+=end-c.offset;c={identity,offset:end,sessionID:header.id};next.files[key]=c;Object.assign(s,next);
      }
      s.files[key]=c;
    }finally{fs.closeSync(fd);}
  }
  function outboxFile(s,initial,t){
    let stat;try{stat=safe(root,outbox);}catch(e){if(e.code==='ENOENT')return;throw e;}
    const identity=String(stat.dev)+':'+stat.ino+':'+stat.birthtimeMs;
    let c=s.files.outbox;
    const fd=fs.openSync(outbox,'r');
    try{
      const opened=fs.fstatSync(fd);if(opened.dev!==stat.dev||opened.ino!==stat.ino)throw Error('source changed');
      if(initial){s.files.outbox={identity,offset:stat.size,dropPartial:stat.size>0&&read(fd,stat.size-1,1)[0]!==10};return;}
      if(!c||c.identity!==identity||stat.size<c.offset)c={identity,offset:0,epoch:(c?.epoch||0)+1};
      const data=read(fd,c.offset,Math.min(MAX,stat.size-c.offset));let start=0;
      while(start<data.length){const end=data.indexOf(10,start);if(end<0)break;
        const nextOffset=c.offset+end+1;
        if(c.dropPartial){c.dropPartial=false;}else if(end-start<65536){
          let ev;try{ev=JSON.parse(data.subarray(start,end).toString('utf8'));}catch{}
          if(typeof ev?.text==='string'&&ev.text.trim()){
            if(s.pending.length>=QUEUE)break;
            const text=ev.text.replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/g,'').slice(0,512);
            emit(s,hash('outbox:'+identity+':'+(c.epoch||0)+':'+nextOffset),'whale',text,{pinned:ev.kind==='notify'});s.lastSeen=t;
          }
        }
        start=end+1;
      }
      s.files.outbox={...c,offset:c.offset+start};
    }finally{fs.closeSync(fd);}
  }
  function snapshot(){return {state:Object.values(state.sessions).some(s=>s.open)||now()-state.lastActive<60000?'working':'idle',events:state.pending};}
  function poll(){
    let next=clone(state);const t=now(),today=day(t),initial=!next.initialized,lastSeen=next.lastSeen;
    if(next.day!==today){next.day=today;next.used=0;next.marks=[];}
    const files=list(root),live=new Set(files.map(file=>hash(path.relative(root,file))));
    for(const [key,cursor] of Object.entries(next.files))if(key!=='outbox'&&!live.has(key))delete next.files[key];
    const sessionIDs=new Set(Object.values(next.files).map(c=>c.sessionID).filter(Boolean));
    for(const sess of Object.values(next.sessions))if(!sessionIDs.has(sess.id))sess.open=false;
    for(const file of files){
      const candidate=clone(next);
      try{sessionFile(candidate,file,initial,t);next=candidate;}catch(e){if(e.message==='queue full')break;/* Retry unread malformed/torn source next tick. */}
    }
    const candidate=clone(next);try{outboxFile(candidate,initial,t);next=candidate;}catch{/* Keep cursor on failed or backpressured read. */}
    next.initialized=true;
    const active=Object.values(next.sessions).some(s=>s.open)||t-next.lastActive<60000;
    if(!active&&t-next.lastActive>600000)next.activeSince=0;
    if(active){
      if(!next.activeSince)next.activeSince=t;
      if(t-next.activeSince>=45*60000&&t-next.lastRest>=60*60000&&next.pending.length<QUEUE){emit(next,hash('rest:'+t),'rest','已经连续工作 45 分钟，起来休息一下吧。');next.lastRest=t;}
      if(lastSeen&&t-lastSeen>=3*86400000&&next.lastGreet!==today&&next.pending.length<QUEUE){emit(next,hash('miss:'+today),'miss','好几天没见，欢迎回来。');next.lastGreet=t;}
    }
    if(JSON.stringify(next)!==JSON.stringify(state))save(next);
    return snapshot();
  }
  function ack(id){if(typeof id!=='string'||!state.pending.some(e=>e.id===id))return false;const next=clone(state);next.pending=next.pending.filter(e=>e.id!==id);save(next);return true;}
  return {poll,ack,snapshot};
}
module.exports={createWatcher};
// The native wrapper supplies only the actual current owned Host home.
if(process.argv[1]==='--pet-event-worker'){
  const watcher=createWatcher(process.argv[2]);
  const send=()=>process.stdout.write(JSON.stringify(watcher.poll())+'\n');
  const tick=()=>{try{send();}catch{process.stdout.write('{"state":"unavailable","events":[]}\n');}};
  tick();const timer=setInterval(tick,2000);
  require('node:readline').createInterface({input:process.stdin}).on('line',line=>{
    if(line.length>128)return;try{const m=JSON.parse(line);if(watcher.ack(m.ack))send();}catch{/* Pending receipt remains durable if save failed. */}
  }).on('close',()=>{clearInterval(timer);process.exit(0);});
}
