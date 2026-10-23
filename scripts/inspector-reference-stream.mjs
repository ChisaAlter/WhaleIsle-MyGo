// Replay the shared follow frames through installed controller/model/table code.
// node script debugPort isolatedPage inspectorBundle controllerBundle outputPrefix
import {readFileSync,writeFileSync} from 'node:fs';
const [port,pageUrl,inspectorPath,controllerPath,output]=process.argv.slice(2);
if(!port||!pageUrl||!inspectorPath||!controllerPath||!output||new URL(pageUrl).port==='3080')throw Error('Specify the isolated reference page and installed bundles');
const phases=JSON.parse(readFileSync('internal/ui/frame/testdata/inspector-stream.json','utf8'));
function expose(path,slot,exports){let text=readFileSync(path,'utf8').replace('window.__ModuleLoader__.load({',`window.${slot}=({`);const end=text.lastIndexOf('return module.exports;');if(end<0)throw Error('Missing factory return');return text.slice(0,end)+`module.exports.__fixture={${exports}};`+text.slice(end)}
const inspector=expose(inspectorPath,'__streamInspectorRegistration','SessionInspectorView,SessionLogModel,en');
const controller=expose(controllerPath,'__streamControllerRegistration','ClientAssistantStream,MutableSessionEventSource');
const pages=await(await fetch(`http://127.0.0.1:${port}/json/list`)).json();
const page=pages.find(p=>p.url===pageUrl);if(!page)throw Error('Owned page missing');
const ws=new WebSocket(page.webSocketDebuggerUrl);let id=0;const pending=new Map();
await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});
ws.addEventListener('message',e=>{const r=JSON.parse(e.data);if(r.id&&pending.has(r.id)){const p=pending.get(r.id);pending.delete(r.id);r.error?p.reject(Error(JSON.stringify(r.error))):p.resolve(r.result)}});
const send=(method,params={})=>new Promise((resolve,reject)=>{const key=++id;pending.set(key,{resolve,reject});ws.send(JSON.stringify({id:key,method,params}))});
const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value};
const settle=()=>new Promise(resolve=>setTimeout(resolve,200));
try{
 await send('Emulation.setDeviceMetricsOverride',{width:780,height:600,deviceScaleFactor:1,mobile:false});await send('Page.bringToFront');
 await evaluate(`(async()=>{const M=window.__DSH_MODULES__;${inspector}\n${controller}
 const React=await M.import('react'),DOM=await M.import('react-dom/client'),Sync=await M.import('react-dom');
 const I=window.__streamInspectorRegistration,C=window.__streamControllerRegistration;
 const E=I.factory(M.makeRequire(I.id,new Set())).__fixture,K=C.factory(M.makeRequire(C.id,new Set())).__fixture;
 const source=new K.MutableSessionEventSource(),fold=new K.ClientAssistantStream(),model=new E.SessionLogModel(source),phases=${JSON.stringify(phases)};
 const publish=result=>{if(result?.type==='rebaseline')throw Error('Unexpected rebaseline');if(result?.type==='transient')source.append(result.entry);if(result?.type==='publish'){source.append(result.entry);if(result.retireAttemptId!==undefined)source.settleAssistant(result.retireAttemptId)}if(result?.type==='settlement')source.settleAssistant(result.attemptId,result.entry);if(result?.type==='abandonment')source.settleAssistant(result.attemptId)};
 let phase=0;window.__streamAdvance=()=>Sync.flushSync(()=>{for(const frame of phases[phase++].frames){if(frame.type==='snapshot')source.replace(fold.replace(frame.records,frame.assistantStream),false);else if(frame.type==='event')publish(fold.acceptDurable({type:'event',event:frame.event}));else publish(fold.acceptFrame(frame.frame))}});
 window.__streamProjection=()=>model.getSnapshot().map(row=>({...row,...model.row(row.key).getSnapshot()}));
 const t=(key,params={})=>{let text=E.en[key]||key;for(const[k,v]of Object.entries(params))text=text.replaceAll('{'+k+'}',String(v));return text};
 const useRows=select=>select(React.useSyncExternalStore(listener=>model.subscribe(listener),model.getSnapshot));
 const useRecord=key=>{const source=model.row(key);return React.useSyncExternalStore(listener=>source.subscribe(listener),()=>source.getSnapshot())};
 const props={t,useSession:select=>select({hasMore:false,loadingOlder:false,openError:null}),useLogRows:useRows,useChatRows:useRows,useLogRecord:useRecord,useChatRecord:useRecord,loadOlder:async()=>{},recordType:key=>model.row(key).getSnapshot()?.type,chatRoot:()=>undefined,pickChat:()=>()=>{},resolveChatTargets:()=>[],logChatTarget:()=>undefined,pickRow:()=>undefined,objects:{row:()=>undefined}};
 document.querySelector('#r104-stream')?.remove();const mount=document.createElement('div');mount.id='r104-stream';mount.style.cssText='position:fixed;inset:0;z-index:2147483000;display:flex;background:white;color:var(--dsw-alias-label-primary);font-size:11px';document.body.append(mount);
 Sync.flushSync(()=>DOM.createRoot(mount).render(React.createElement(E.SessionInspectorView,props)));window.__streamAdvance();return true})()`);
 const state=()=>evaluate(`(()=>{const root=document.querySelector('#r104-stream');return {records:[...root.querySelectorAll('tbody tr[aria-selected]')].map(row=>({identity:row.cells[0].innerText,type:row.querySelector('td:nth-child(2) button:last-child')?.textContent,preview:row.cells[row.cells.length-1].innerText,expanded:row.querySelector('button[aria-expanded]')?.getAttribute('aria-expanded')??null})),text:root.innerText,projection:window.__streamProjection()}})()`);
 const capture=async(name,want)=>{await settle();const actual=await state();if(actual.records.length!==want)throw Error(name+' row count '+actual.records.length+' != '+want);writeFileSync(output+'-'+name+'-state.json',JSON.stringify(actual,null,2));const shot=await send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});writeFileSync(output+'-'+name+'.png',Buffer.from(shot.data,'base64'));console.log('PASS: '+name+' '+want+' rows');return actual};
 const toggle=async(identity)=>{await evaluate(`(()=>{const row=[...document.querySelectorAll('#r104-stream tbody tr[aria-selected]')].find(row=>row.cells[0].innerText===${JSON.stringify(identity)}&&row.querySelector('td:nth-child(2) button:last-child')?.textContent==='block-start');if(!row)throw Error('Missing block '+${JSON.stringify(identity)});window.__DSH_MODULES__.seed.get('react-dom').flushSync(()=>row.querySelector('button[aria-expanded]').click())})()`);await settle()};
 const click=label=>evaluate(`(()=>{const button=[...document.querySelectorAll('button')].find(e=>e.getAttribute('aria-label')===${JSON.stringify(label)}||e.textContent===${JSON.stringify(label)});if(!button)throw Error('Missing '+${JSON.stringify(label)});window.__DSH_MODULES__.seed.get('react-dom').flushSync(()=>button.click())})()`);
 const filter=async query=>{await click('Filter types');await evaluate(`(()=>{const input=document.querySelector('[role="combobox"]'),set=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;window.__DSH_MODULES__.seed.get('react-dom').flushSync(()=>{set.call(input,${JSON.stringify(query)});input.dispatchEvent(new Event('input',{bubbles:true}))})})()`);await settle();await click('Apply filter');await settle()};
 await capture('open',8);await toggle('0');const folded=await capture('manual-fold',6);if(!folded.text.includes('blockType: reasoning · first second'))throw Error('Missing folded reasoning prefix');
 await toggle('0');await evaluate('window.__streamAdvance()');const closed=await capture('closed',6);if(closed.records.find(row=>row.identity==='0'&&row.type==='block-start').expanded!=='false')throw Error('Open override survived block completion');
 await toggle('0');await capture('reopened',9);await filter('reasoning-delta');await capture('filtered',6);await toggle('0');await capture('filtered-fold',4);await filter('');await capture('cleared',6);await evaluate('window.__streamAdvance()');await capture('settled',10);await evaluate('window.__streamAdvance()');await capture('retired',8);
}finally{ws.close()}
