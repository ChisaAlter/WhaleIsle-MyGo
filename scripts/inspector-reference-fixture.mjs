// Mount installed Inspector components in an independently owned Electron renderer.
// node scripts/inspector-reference-fixture.mjs 19444 http://127.0.0.1:3083/ C:/.../session-inspector/lib/client.js output-prefix
import {readFileSync,writeFileSync} from 'node:fs';
const [debugPort,pageUrl,bundlePath,output]=process.argv.slice(2);
if(!debugPort||!pageUrl||!bundlePath||!output||new URL(pageUrl).port==='3080')throw Error('Specify an isolated reference page, installed bundle and output prefix');
const fixtures=JSON.parse(readFileSync('internal/ui/frame/testdata/inspector-records.json','utf8'));
let bundle=readFileSync(bundlePath,'utf8');
bundle=bundle.replace('window.__ModuleLoader__.load({','window.__r102Registration=({');
const end=bundle.lastIndexOf('return module.exports;');
if(end<0)throw Error('Installed inspector factory return missing');
bundle=bundle.slice(0,end)+'module.exports.__fixture={SessionInspectorView,InspectorTable,en,zh};'+bundle.slice(end);
const targets=await(await fetch(`http://127.0.0.1:${debugPort}/json/list`)).json();
const page=targets.find(p=>p.url===pageUrl);if(!page)throw Error('Owned reference page missing');
const ws=new WebSocket(page.webSocketDebuggerUrl);let next=1;const pending=new Map();
await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});
ws.addEventListener('message',e=>{const r=JSON.parse(e.data);if(r.id&&pending.has(r.id)){const p=pending.get(r.id);pending.delete(r.id);r.error?p.reject(Error(JSON.stringify(r.error))):p.resolve(r.result)}});
const send=(method,params={})=>new Promise((resolve,reject)=>{const id=next++;pending.set(id,{resolve,reject});ws.send(JSON.stringify({id,method,params}))});
const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value};
try{
 await send('Emulation.setDeviceMetricsOverride',{width:780,height:600,deviceScaleFactor:1,mobile:false});await send('Page.bringToFront');
 const setup=await evaluate(`(async()=>{
 const M=window.__DSH_MODULES__;${bundle}
 const registration=window.__r102Registration;
 const React=await M.import('react'),DOM=await M.import('react-dom/client'),SyncDOM=await M.import('react-dom');
 const E=registration.factory(M.makeRequire(registration.id,new Set())).__fixture;
 const fixtures=${JSON.stringify(fixtures)};
 const depths=new Map(),records=new Map();const rows=fixtures.map(f=>{const depth=f.parent?(depths.get(f.parent)||0)+1:0;depths.set(f.key,depth);const data=f.value.data||{};records.set(f.key,{identity:String(f.seq),type:f.type,time:f.time,location:'turn'in data?String(data.turn)+('step'in data?'/'+data.step:''):'',value:f.value});return {key:f.key,parent:f.parent,depth}});
 const t=(key,params={})=>{let text=E.en[key]||key;for(const[k,v]of Object.entries(params))text=text.replaceAll('{'+k+'}',String(v));return text};
 document.querySelector('#r102-inspector-fixture')?.remove();const mount=document.createElement('div');mount.id='r102-inspector-fixture';mount.style.cssText='position:fixed;inset:0;z-index:2147483000;background:white;display:flex;color:var(--dsw-alias-label-primary);font-size:11px;';document.body.append(mount);
 const useRows=select=>select(rows),useRecord=key=>records.get(key);
 const props={t,useSession:select=>select({hasMore:false,loadingOlder:false,openError:null}),useLogRows:useRows,useChatRows:useRows,useLogRecord:useRecord,useChatRecord:useRecord,loadOlder:async()=>{},recordType:key=>records.get(key)?.type,chatRoot:()=>undefined,pickChat:()=>undefined,resolveChatTargets:()=>[],logChatTarget:()=>undefined,pickRow:()=>undefined,objects:{row:()=>undefined}};
 SyncDOM.flushSync(()=>DOM.createRoot(mount).render(React.createElement(E.SessionInspectorView,props)));
 return {rows:rows.length,components:Object.keys(E)};
 })()`);
 console.log(JSON.stringify(setup));await new Promise(resolve=>setTimeout(resolve,300));
 const capture=async suffix=>{const geometry=await evaluate(`(()=>{const root=document.querySelector('#r102-inspector-fixture');const rect=e=>{if(!e)return null;const r=e.getBoundingClientRect(),s=getComputedStyle(e);return {x:r.x,y:r.y,w:r.width,h:r.height,font:s.font,background:s.backgroundColor,padding:s.padding,scrollWidth:e.scrollWidth,scrollHeight:e.scrollHeight}};return {root:rect(root.querySelector('section')),text:root.innerText,toolbar:rect(root.querySelector('section>div')),header:rect(root.querySelector('thead tr')),columns:[...root.querySelectorAll('th')].map(e=>({text:e.innerText,...rect(e)})),rows:[...root.querySelectorAll('tbody tr[aria-selected]')].map(e=>({text:e.innerText,...rect(e)})),details:rect(root.querySelector('[role="separator"]')?.parentElement)}})()`);writeFileSync(output+suffix+'.json',JSON.stringify(geometry,null,2));const shot=await send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});writeFileSync(output+suffix+'.png',Buffer.from(shot.data,'base64'));return geometry};
 console.log(JSON.stringify(await capture('')));
 await evaluate(`(()=>{const root=document.querySelector('#r102-inspector-fixture');const DOM=window.__DSH_MODULES__.seed.get('react-dom');DOM.flushSync(()=>[...root.querySelectorAll('button')].find(e=>e.textContent==='future/custom').click());return root.innerText})()`);
 await new Promise(resolve=>setTimeout(resolve,150));console.log(JSON.stringify(await capture('-details')));
 await send('DOM.enable');await send('CSS.enable');
 const doc=await send('DOM.getDocument');const fonts=[];
 for(const selector of ['#r102-inspector-fixture tbody td button','#r102-inspector-fixture th','#r102-inspector-fixture select','#r102-inspector-fixture summary']){
  const node=await send('DOM.querySelector',{nodeId:doc.root.nodeId,selector});
  if(node.nodeId)fonts.push({selector,...await send('CSS.getPlatformFontsForNode',{nodeId:node.nodeId})});
 }
 writeFileSync(output+'-fonts.json',JSON.stringify(fonts,null,2));
}finally{ws.close()}
