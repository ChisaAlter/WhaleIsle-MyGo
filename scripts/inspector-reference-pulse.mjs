// Sample the installed InspectorTable's actual WAAPI row pulse at fixed offsets.
import {readFileSync,writeFileSync} from 'node:fs';
const [port,pageUrl,bundle,output]=process.argv.slice(2);if(!output||new URL(pageUrl).port==='3080')throw Error('Use isolated reference');
let source=readFileSync(bundle,'utf8').replace('window.__ModuleLoader__.load({','window.__pulseRegistration=({');const end=source.lastIndexOf('return module.exports;');source=source.slice(0,end)+'module.exports.__fixture={InspectorTable,en};'+source.slice(end);
const pages=await(await fetch(`http://127.0.0.1:${port}/json/list`)).json(),page=pages.find(p=>p.url===pageUrl);if(!page)throw Error('Owned page missing');
const ws=new WebSocket(page.webSocketDebuggerUrl);let id=0;const pending=new Map();await new Promise((r,j)=>{ws.addEventListener('open',r,{once:true});ws.addEventListener('error',j,{once:true})});ws.addEventListener('message',e=>{const v=JSON.parse(e.data),p=pending.get(v.id);if(p){pending.delete(v.id);v.error?p.j(Error(JSON.stringify(v.error))):p.r(v.result)}});
const send=(method,params={})=>new Promise((r,j)=>{const key=++id;pending.set(key,{r,j});ws.send(JSON.stringify({id:key,method,params}))});const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value};
try{
 await send('Emulation.setDeviceMetricsOverride',{width:780,height:600,deviceScaleFactor:1,mobile:false});await send('Emulation.setEmulatedMedia',{features:[{name:'prefers-reduced-motion',value:'no-preference'}]});
 await evaluate(`(async()=>{const M=window.__DSH_MODULES__;${source}\nconst React=await M.import('react'),DOM=await M.import('react-dom/client'),Sync=await M.import('react-dom'),I=window.__pulseRegistration,E=I.factory(M.makeRequire(I.id,new Set())).__fixture;
 window.__scrollRoot?.unmount();document.querySelector('#r105-scroll')?.remove();document.querySelector('#r105-pulse')?.remove();const mount=document.createElement('div');mount.id='r105-pulse';mount.style.cssText='position:fixed;inset:0;z-index:2147483001;display:flex;background:white';document.body.append(mount);const root=DOM.createRoot(mount),rows=[{key:'row',depth:0}];let record={type:'text',identity:'row',location:'',value:{text:'initial'}},flash=true,pickedRow;
 const t=(key,params={})=>{let s=E.en[key]||key;for(const[k,v]of Object.entries(params))s=s.replaceAll('{'+k+'}',v);return s};let generation=0;
 const render=()=>Sync.flushSync(()=>root.render(React.createElement(E.InspectorTable,{title:'Pulse fixture',flash,showTime:false,pickedRow,useRows:select=>select(rows),useRecord:key=>key==='row'?record:undefined,useSession:select=>select({hasMore:false,loadingOlder:false,openError:null}),loadOlder:async()=>{},typeOf:()=>record.type,t})));
 window.__pulseUpdate=()=>{record={...record,value:{text:'updated '+(++generation)}};render()};window.__pulseReveal=()=>{pickedRow={key:'row'};render()};window.__pulseRaw=()=>{flash=false;render()};window.__pulseRender=render;render();return true})()`);
 const rowExpression="document.querySelector('#r105-pulse tr[aria-selected]')";
 const count=()=>evaluate(`${rowExpression}.getAnimations().length`);
 const outputData={source:'Installed InspectorTable through its real hooks and WAAPI',initialAnimations:await count(),samples:[]};if(outputData.initialAnimations!==0)throw Error('History flashed');
 await evaluate('window.__pulseUpdate()');
 for(const offset of [0,350,700]){
  const state=await evaluate(`(()=>{const row=${rowExpression},a=row.getAnimations()[0];if(!a)throw Error('Missing Chat Group update animation');a.pause();a.currentTime=${offset};return {offset:${offset},duration:a.effect.getTiming().duration,easing:a.effect.getTiming().easing,color:getComputedStyle(row).backgroundColor,keyframes:a.effect.getKeyframes().map(k=>k.backgroundColor)}})()`);
  outputData.samples.push(state);if(state.duration!==700||state.easing!=='linear')throw Error('Pulse timing mismatch');
  const shot=await send('Page.captureScreenshot',{format:'png'});writeFileSync(output+'-'+offset+'.png',Buffer.from(shot.data,'base64'));
 }
 await evaluate('window.__pulseRaw();window.__pulseUpdate()');outputData.rawUpdateAnimations=await count();if(outputData.rawUpdateAnimations!==0)throw Error('Raw update flashed');
 await evaluate('window.__pulseReveal()');outputData.rawRevealAnimations=await count();if(outputData.rawRevealAnimations!==1)throw Error('Raw reveal missed');
 await send('Emulation.setEmulatedMedia',{features:[{name:'prefers-reduced-motion',value:'reduce'}]});await evaluate('window.__pulseReveal()');outputData.reducedRevealAnimations=await count();if(outputData.reducedRevealAnimations!==0)throw Error('Reduced motion flashed');
 writeFileSync(output+'-state.json',JSON.stringify(outputData,null,2)+'\n');console.log('PASS installed pulse initial/update/raw/reveal/reduced; linear700ms sampled0/350/700');
}finally{await send('Emulation.setEmulatedMedia',{features:[]});ws.close()}
