/* Rich controls stay on the existing React/DOM owner. Top-layer promotion changes
 * only presentation: no clone, innerHTML, remount, serialized File or canvas. */
var whaleRichSurface = typeof whaleRichSurface === "object" ? whaleRichSurface : (() => {
 let source=null,active=new Map(),css=null,lastToken=0;
 const height=new WeakMap();let portalCandidates=null;
 const same=(a,b)=>['kind','name','rowId','itemId','entryId','sessionId','epoch'].every(key=>(a?.[key]||'')===(b?.[key]||''));
 const rootOf=owner=>owner?.root||owner?.marker;
 function hidden(node){for(let p=node;p?.nodeType===1;p=p.parentElement){if(p.hidden||p.tagName==='DIALOG'&&!p.open||getComputedStyle(p).display==='none')return true}return false}
 function rect(node){
  if(node.hasAttribute?.('data-whale-slot-entry')||getComputedStyle(node).display==='contents'){
   const range=document.createRange();range.selectNodeContents(node);const r=range.getBoundingClientRect();range.detach();if(r.width>0&&r.height>0)return r;
  }
  return node.getBoundingClientRect();
 }
 function restore(node,old){try{if(!old.open&&node.matches(':popover-open'))node.hidePopover();if(node.getAttribute('popover')==='manual'){if(old.popover===null)node.removeAttribute('popover');else node.setAttribute('popover',old.popover)}for(const [name,value] of old.applied){if(node.style.getPropertyValue(name)!==value)continue;const prior=old.properties.get(name);if(prior.value)node.style.setProperty(name,prior.value,prior.priority);else node.style.removeProperty(name)}node.removeAttribute('data-whale-rich-seat')}catch{}}
 function clear(){for(const [node,old] of active)restore(node,old);active.clear();source?.()?.drain?.()}
 // React DOM's actual HostPortal retains its fiber return chain to the source
 // host marker. Time, text, role and document proximity never grant ownership.
 function ownsPortal(node,root){
  if(!root?.isConnected||!node?.isConnected||root.contains(node)||node.contains(root))return false;
  let matched=false;
  for(const host of [node,...node.querySelectorAll('*')].slice(0,64)){
   const key=Object.keys(host).find(key=>key.startsWith('__reactFiber$'));if(!key)continue;let fiber=host[key];if(fiber?.stateNode!==host)continue;
   let belongs=false;const seen=new Set();for(let n=fiber;n&&!seen.has(n)&&seen.size<512;n=n.return){seen.add(n);if(n.stateNode===root){belongs=true;break}}
   if(!belongs)return false;matched=true;
  }
  return matched;
 }
 function portals(owner){const root=owner?.portalRoot||rootOf(owner);if(!portalCandidates){
  const rows=new Set(document.querySelectorAll('[role="dialog"],[role="alertdialog"],[role="menu"],[role="listbox"],[popover]'));
  for(const node of document.querySelectorAll('body *')){const key=Object.keys(node).find(key=>key.startsWith('__reactFiber$'));let f=key&&node[key];if(f?.stateNode!==node)continue;const seen=new Set();for(let n=f;n&&!seen.has(n)&&seen.size<512;n=n.return){seen.add(n);if(n.tag===4){rows.add(node);break}}}portalCandidates=[...rows];
 }const owned=portalCandidates.filter(node=>!hidden(node)&&ownsPortal(node,root));return owned.filter(node=>!owned.some(parent=>parent!==node&&parent.contains(node))).slice(0,64)}
 const cause=()=>{};
 if(typeof MutationObserver==='function')new MutationObserver(()=>{
  portalCandidates=null;const env=source?.();for(const [node,old] of active){const record=[...(env?.actions?.values()||[])].find(record=>record.node===node);if(!node.isConnected||hidden(node)||!record||!env.valid(record)){restore(node,old);active.delete(node)}}
 }).observe(document,{childList:true,subtree:true,attributes:true,attributeFilter:['hidden','aria-hidden']});
 // A file picker inside a proven composer owner is a native draft gesture,
 // not a browser registry attachment. Prevent the browser chooser before it
 // creates an unrelated FileList; generic plugin forms retain their own picker.
 if(typeof document.addEventListener==='function')document.addEventListener('click',event=>{
  const node=event.target;if(node?.tagName!=='INPUT'||node.type!=='file')return;const env=source?.();
  for(const record of env?.actions?.values()||[]){const slot=record.owner?.slot||'';if(!(slot==='conversation.composer'||slot.startsWith('conversation.composer.')||slot==='conversation.input.attachments')||!env.valid(record)||!record.node.contains(node)||!whaleNativeContains(record.owner,node))continue;
   if(window.__whaleNativeComposer?.pickFiles(env.subject.sessionId,/image\//.test(node.accept||''))){event.preventDefault();event.stopImmediatePropagation()}break;
  }
 },true);
 function layout(command){
  const env=source?.();if(!env||!Number.isSafeInteger(command.token)||command.token<lastToken||!Array.isArray(command.seats)||command.seats.length>128)return false;
  if(command.seats.length===0){lastToken=command.token;clear();return true}
  if(command.revision!==env.revision||!same(command.subject,env.subject))return false;
  const next=new Map();
  for(const seat of command.seats){const record=env.actions.get(seat.handle),node=record?.node;if(!node?.isConnected||hidden(node)||!env.valid(record)||!whaleNativeContains(record.owner,node)||typeof node.showPopover!=='function'||![seat.x,seat.y,seat.w,seat.h].every(Number.isFinite)||seat.w<=0||seat.h<=0||seat.w>8192||seat.h>8192||![seat.cl,seat.ct,seat.cr,seat.cb].every(v=>Number.isFinite(v)&&v>=0)||seat.cl+seat.cr>=seat.w||seat.ct+seat.cb>=seat.h||next.has(node))return false;next.set(node,seat)}
  lastToken=command.token;
  for(const [node,old] of active)if(!next.has(node)){restore(node,old);active.delete(node)}
  if(!next.size){clear();return true}
  if(!css){css=document.createElement('style');css.textContent='html[data-whale-rich-mask] body :not([data-whale-rich-seat]):not([data-whale-rich-seat] *){visibility:hidden!important;pointer-events:none!important}:where([data-whale-rich-seat]){visibility:visible;pointer-events:auto;position:fixed;inset:auto;margin:0;border:none;padding:0;background:transparent;color:inherit;overflow:visible}[data-whale-rich-seat]::backdrop{display:none!important}html[data-whale-rich-mask] body [data-whale-rich-seat][hidden],html[data-whale-rich-mask] body [hidden] [data-whale-rich-seat]{visibility:hidden!important;pointer-events:none!important}';document.documentElement.setAttribute('data-whale-rich-mask','');document.head.appendChild(css)}
  const assign=(node,old,name,value)=>{if(!old.properties.has(name))old.properties.set(name,{value:node.style.getPropertyValue(name),priority:node.style.getPropertyPriority(name)});node.style.setProperty(name,value,'important');old.applied.set(name,node.style.getPropertyValue(name))};
  let drifted=false;
  try{for(const [node,seat] of next){const record=env.actions.get(seat.handle);if(!active.has(node)){
   const computed=getComputedStyle(node),ambientFixed=!!(record?.owner?.ambient&&computed.position==='fixed'),baseline={};
   if(!ambientFixed)for(const name of ['border-top','border-right','border-bottom','border-left','padding-top','padding-right','padding-bottom','padding-left','background-color'])baseline[name]=computed.getPropertyValue(name);
   height.set(node,Math.max(1,rect(node).height));const old={popover:node.getAttribute('popover'),open:node.matches(':popover-open'),display:computed.display==='contents'?'block':computed.display,overflow:computed.overflow,ambientFixed,properties:new Map(),applied:new Map()};active.set(node,old);node.setAttribute('popover','manual');node.setAttribute('data-whale-rich-seat','');if(!node.matches(':popover-open'))node.showPopover();for(const [name,value] of Object.entries(baseline))assign(node,old,name,value);
  }
   const old=active.get(node);
   if(old.ambientFixed){
    if(seat.ct||seat.cr||seat.cb||seat.cl)assign(node,old,'clip-path','inset('+seat.ct+'px '+seat.cr+'px '+seat.cb+'px '+seat.cl+'px)');
    else if(old.applied.has('clip-path')){const prior=old.properties.get('clip-path');if(prior?.value)node.style.setProperty('clip-path',prior.value,prior.priority);else node.style.removeProperty('clip-path');old.applied.delete('clip-path')}
    const r=node.getBoundingClientRect(),rx=r.left??r.x,ry=r.top??r.y;if(r.width<=0||r.height<=0)throw Error('top-layer geometry not retained');
    if(Math.abs(rx-seat.x)>1||Math.abs(ry-seat.y)>1||Math.abs(r.width-seat.w)>1||Math.abs(r.height-seat.h)>1)drifted=true;
   }else{
    for(const [name,value] of Object.entries({display:old.display,position:'fixed',inset:'auto',left:seat.x+'px',top:seat.y+'px',width:seat.w+'px',height:seat.h+'px',margin:'0',boxSizing:'border-box',overflow:old.overflow,clipPath:'inset('+seat.ct+'px '+seat.cr+'px '+seat.cb+'px '+seat.cl+'px)'}))assign(node,old,name.replace(/[A-Z]/g,m=>'-'+m.toLowerCase()),value);
    const r=node.getBoundingClientRect(),rx=r.left??r.x,ry=r.top??r.y;if(Math.abs(rx-seat.x)>2||Math.abs(ry-seat.y)>2||Math.abs(r.width-seat.w)>2||Math.abs(r.height-seat.h)>2)throw Error('top-layer geometry not retained');
   }
  }}catch{clear();return false}finally{env.drain?.()}if(drifted)env.schedule?.();return true;
 }
 const ack=(command,send)=>{const ok=layout(command);send({type:'richAck',token:command.token,revision:command.revision,ok:ok&&command.token===lastToken&&(command.seats.length===0||command.revision===source?.().revision&&same(command.subject,source?.().subject))})};
 return {ack,bind(get){clear();lastToken=0;source=get;return ()=>{clear();source=null}},layout,clear,release(command){if(command.token!==lastToken||active.size)return false;css?.remove();css=null;document.documentElement.removeAttribute('data-whale-rich-mask');source?.()?.drain?.();return true},cause,portals,ownsPortal,rect,hidden,applied(node){return active.has(node)},height(node){return height.get(node)||Math.max(1,rect(node).height)}};
})();
