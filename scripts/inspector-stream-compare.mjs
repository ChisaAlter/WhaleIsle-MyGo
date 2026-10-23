// Compare real native and installed reference projections from one shared replay.
import {readFileSync,writeFileSync} from 'node:fs';
const [directory='docs/review-20261022/evidence']=process.argv.slice(2);
const read=name=>JSON.parse(readFileSync(directory+'/'+name,'utf8').replace(/^\uFEFF/,''));
const canonical=value=>Array.isArray(value)?value.map(canonical):value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(key=>[key,canonical(value[key])])):value;
const equal=(actual,want,label)=>{if(JSON.stringify(canonical(actual))!==JSON.stringify(canonical(want)))throw Error(label+' mismatch\n'+JSON.stringify(actual)+'\n'+JSON.stringify(want))};
const results=[];
for(const phase of ['open','closed','settled','retired']){
 const native=read('inspector-native-pass248-projection-'+phase+'.json');
 const reference=read('inspector-reference-pass248-'+phase+'-state.json').projection;
 equal(native.map(row=>({key:row.Key,parent:row.Parent||'',type:row.Type,identity:row.Identity,location:row.Location,time:row.Time,disclosure:row.Disclosure||'',collapsedSummary:row.CollapsedSummary||'',value:row.Raw})),reference.map(row=>({key:row.key,parent:row.parent||'',type:row.type,identity:row.identity,location:row.location,time:row.time,disclosure:row.disclosure||'',collapsedSummary:row.collapsedSummary||'',value:row.value})),phase+' complete projection');
 results.push({kind:'projection',phase,rows:native.length,pass:true});
}
for(const name of ['open','manual-fold','closed','reopened','filtered','filtered-fold','cleared','settled','retired']){
 const phase=['open','manual-fold'].includes(name)?'open':['closed','reopened','filtered','filtered-fold','cleared'].includes(name)?'closed':name;
 const native=read('inspector-native-pass248-'+name+'-state.json');
 const reference=read('inspector-reference-pass248-'+name+'-state.json');
 const projection=read('inspector-native-pass248-projection-'+phase+'.json');
 const byKey=new Map(projection.map(row=>[row.Key,row]));
 equal([...native.records].sort((a,b)=>a.y-b.y).map(row=>({identity:byKey.get(row.key)?.Identity,type:byKey.get(row.key)?.Type})),reference.records.map(row=>({identity:row.identity,type:row.type})),name+' real visible rows');
 for(const row of reference.records){if(row.preview.trim()!==''&&!native.names.some(name=>name.trim()===row.preview.trim()))throw Error(name+' missing actual native preview '+row.preview)}
 results.push({kind:'visible rows and previews',phase:name,rows:native.records.length,pass:true});
}
writeFileSync(directory+'/inspector-pass248-comparison.json',JSON.stringify({source:'Shared normal follow frames; installed controller/model and native Store; separate real window interactions',checks:results},null,2));
console.log('PASS: '+results.length+' complete-projection and real-window comparisons');
