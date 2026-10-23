import {readFileSync,writeFileSync} from 'node:fs';
const prefix='docs/review-20261022/evidence/inspector-';
const phases=['manual-open','manual-fold','manual-latest','auto-open','auto-closed','auto-appended','auto-latest'];
const checks=[];
for(const phase of phases){
 const native=JSON.parse(readFileSync(prefix+'native-pass249-'+phase+'-state.json','utf8').replace(/^\uFEFF/,'')),ref=JSON.parse(readFileSync(prefix+'reference-pass249-'+phase+'-state.json','utf8'));
 const near=(a,b,t=2)=>Math.abs(a-b)<=t;
 if(!near(native.offset,ref.offset)||!near(native.viewportHeight,ref.viewportHeight))throw Error(phase+' viewport mismatch');
 if(ref.padding===0?native.padding!==0:!near(native.padding,ref.padding))throw Error(phase+' trailing extent mismatch');
 const rows=ref.records.filter(r=>r.top>=29.9&&r.top<ref.viewportHeight-.1);
 for(const row of rows){const key=row.type==='assistant/live'?'attempt:scroll-a':row.type==='block-start'?'attempt:scroll-a/chunk:0':row.type==='reasoning-delta'?'attempt:scroll-a/chunk:'+row.identity:'event:'+row.identity;const actual=native.records.find(r=>r.key===key);if(!actual||!near(actual.top,row.top,2)||actual.height!==30)throw Error(phase+' row '+key+' misplaced '+JSON.stringify(actual));}
 if(phase==='manual-fold'&&native.mode!=='manual')throw Error('Manual mode not retained');
 if((phase==='auto-closed'||phase==='auto-appended'||phase.endsWith('latest'))&&native.mode!=='tail')throw Error(phase+' following lost');
 checks.push({phase,pass:true,offset:{reference:ref.offset,native:native.offset},padding:{reference:ref.padding,native:native.padding},checkedRows:rows.length});
}
writeFileSync(prefix+'pass249-scroll-comparison.json',JSON.stringify({source:'Same normal follow frames through installed controller/model and native Store; actual-window disclosure and latest actions',tolerance:'2 DIP absolute viewport/offset/padding due to window and fractional CSS height;2 DIP row tops across separately sized/scaled window surfaces;30 DIP row heights',checks},null,2)+'\n');
console.log('PASS seven real-window scroll/fold/append states');
