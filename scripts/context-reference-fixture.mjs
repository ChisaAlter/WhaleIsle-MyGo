// Run from the repository root against an independently owned Electron profile.
// Example: node scripts/context-reference-fixture.mjs 19444 http://127.0.0.1:3083/ docs/review-20261022/evidence/context-components-reference-pass245
import {readFileSync,writeFileSync} from 'node:fs';
const [debugPort='19444',pageUrl='http://127.0.0.1:3083/',output='docs/review-20261022/evidence/context-components-reference-pass245']=process.argv.slice(2);
if(new URL(pageUrl).port==='3080')throw Error('Use an isolated reference profile, not the user GUI');
const fixtures=JSON.parse(readFileSync('internal/ui/chat/testdata/context-components.json','utf8'));
const pages=await(await fetch(`http://127.0.0.1:${debugPort}/json/list`)).json();
const page=pages.find(p=>p.url===pageUrl);
if(!page)throw Error('Isolated reference page unavailable');
const ws=new WebSocket(page.webSocketDebuggerUrl);let next=1;const pending=new Map();
await new Promise((resolve,reject)=>{ws.addEventListener('open',resolve,{once:true});ws.addEventListener('error',reject,{once:true})});
ws.addEventListener('message',e=>{const r=JSON.parse(e.data);if(r.id&&pending.has(r.id)){const p=pending.get(r.id);pending.delete(r.id);r.error?p.reject(Error(JSON.stringify(r.error))):p.resolve(r.result)}});
const send=(method,params={})=>new Promise((resolve,reject)=>{const id=next++;pending.set(id,{resolve,reject});ws.send(JSON.stringify({id,method,params}))});
const evaluate=async expression=>{const r=await send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(JSON.stringify(r.exceptionDetails));return r.result.value};
await send('Emulation.setDeviceMetricsOverride',{width:900,height:800,deviceScaleFactor:1,mobile:false});
await send('Page.bringToFront');
const setup=await evaluate(`(async()=>{
const M=window.__DSH_MODULES__,registration=M.factories.get('@deepseek-ai/dsh-client-ui-chat');
let source=registration.factory.toString(); const end=source.lastIndexOf('return module.exports;');
if(end<0)throw Error('Reference factory return missing');
source=source.slice(0,end)+'module.exports.__fixture={ContextInjectionRow,TurnTriggerNodeView,zh,en};'+source.slice(end);
const exported=(0,eval)('('+source+')')(M.makeRequire('@deepseek-ai/dsh-client-ui-chat',new Set())).__fixture;
const React=await M.import('react'),DOM=await M.import('react-dom/client');
const fixture=${JSON.stringify(fixtures)};
const t=(key,params={})=>{let value=exported.en[key]||key;for(const[k,v]of Object.entries(params))value=value.replaceAll('{'+k+'}',String(v));return value};
document.querySelector('#r101-component-fixture')?.remove();const mount=document.createElement('div');mount.id='r101-component-fixture';mount.style.cssText='position:fixed;inset:0;z-index:2147483000;background:white;color:var(--dsw-alias-label-primary);overflow:auto;--dsh-content-font-size:14px;--dsh-content-font-delta:0px;--dsh-content-font-size-secondary:13px;';document.body.append(mount);
const host=React.createElement('div',{style:{width:600,boxSizing:'border-box',padding:24,display:'flex',flexDirection:'column',gap:12,fontSize:14}},fixture.map(f=>React.createElement('div',{key:f.id,'data-fixture':f.id},f.id==='trigger'?React.createElement(exported.TurnTriggerNodeView,{node:{data:{time:f.time,content:f.content,source:f.source}},t}):React.createElement(exported.ContextInjectionRow,{content:f.content,source:f.source,form:['instructions','catalog','snapshot','notice','relay','recall'].includes(f.source.form)?f.source.form:null,producer:{role:'inject',label:f.id==='instructions'?'AGENTS.md, rules.md':f.source.kind},t}))));
const SyncDOM=await M.import('react-dom');SyncDOM.flushSync(()=>DOM.createRoot(mount).render(host));return {exported:Object.keys(exported),fixtures:fixture.map(f=>f.id)};
})()`);
console.log('setup',JSON.stringify(setup));
await new Promise(resolve=>setTimeout(resolve,100));
await evaluate(`(()=>{const root=document.querySelector('#r101-component-fixture');const DOM=window.__DSH_MODULES__.seed.get('react-dom');DOM.flushSync(()=>{for(const row of root.querySelectorAll('[role="button"][aria-expanded="false"],button[aria-expanded="false"]'))row.click()});return root.innerText})()`);
await new Promise(resolve=>setTimeout(resolve,300));
const geometry=await evaluate(`(()=>{const root=document.querySelector('#r101-component-fixture');const rect=e=>{const r=e.getBoundingClientRect(),s=getComputedStyle(e);return {x:r.x,y:r.y,w:r.width,h:r.height,font:s.font,background:s.backgroundColor,padding:s.padding,scrollWidth:e.scrollWidth,scrollHeight:e.scrollHeight}};return [...root.querySelectorAll('[data-fixture]')].map(e=>({id:e.dataset.fixture,rect:rect(e),text:e.innerText,header:rect(e.querySelector('[aria-expanded]')),body:e.querySelector('[data-context-injection-body]')?rect(e.querySelector('[data-context-injection-body]')):null}))})()`);
writeFileSync(output+'.json',JSON.stringify(geometry,null,2));
const shot=await send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});
writeFileSync(output+'.png',Buffer.from(shot.data,'base64'));
console.log(JSON.stringify(geometry));
ws.close();
