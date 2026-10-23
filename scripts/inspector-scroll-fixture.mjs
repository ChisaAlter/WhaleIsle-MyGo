import {writeFileSync} from 'node:fs';
// Normal follow frames shared by the installed controller and native Store.
const event=(type,seq,data={})=>({type:'event',event:{type,seq,time:1792759200000+seq*100,data}});
const chunk=(index,value)=>({type:'assistant-stream',frame:{type:'chunk',revision:index+2,attemptId:'scroll-a',index,time:1792759203000+index*10,chunk:value}});
const records=Array.from({length:20},(_,seq)=>event('private/before',seq));
records.push(event('turn/start',20,{turn:1}),event('step/start',21,{turn:1,step:0}));
const frames=[{type:'snapshot',records},{type:'assistant-stream',frame:{type:'start',revision:1,attemptId:'scroll-a',startedAfterSeq:21,turn:1,step:0}},chunk(0,{type:'block-start',index:0,blockType:'reasoning'})];
for(let i=0;i<200;i++)frames.push(chunk(i+1,{type:'reasoning-delta',index:0,text:'delta '+i+' '}));
frames.push(event('private/next',22,{turn:1,step:0}));
writeFileSync('internal/ui/frame/testdata/inspector-scroll.json',JSON.stringify([{name:'open',frames},{name:'closed',frames:[chunk(201,{type:'block-end',index:0,block:{type:'reasoning',text:'completed'}})]},{name:'appended',frames:[event('private/after',23,{turn:1,step:0})]}],null,2)+'\n');
