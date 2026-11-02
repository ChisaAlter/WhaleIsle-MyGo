// Original DOM focus is the sole text owner. This bridge reads its native text
// context and validates trusted browser input; it never dispatches DOM events.
(() => {
 const create = env => {
  const ids=new WeakMap();let armed=null,cancelled=false,composition=null,measureRoot=null;
  function measurementRoot(){if(!measureRoot){const host=document.createElement('span');host.setAttribute('aria-hidden','true');host.style.cssText='position:fixed;left:-100000px;top:0;visibility:hidden;pointer-events:none;width:0;height:0;overflow:hidden;';measureRoot=host.attachShadow({mode:'closed'});document.documentElement.appendChild(host)}return measureRoot}
  const receipts=new Map();
  const finite=n=>Number.isFinite(n),clamp=(n,a,b)=>Math.max(a,Math.min(b,n));
  const id=node=>{let value=ids.get(node);if(!value){value=crypto.randomUUID();ids.set(node,value)}return value};
  const editable=node=>node instanceof HTMLTextAreaElement || node instanceof HTMLInputElement && ['text','search','email','url','tel','password',''].includes(node.type) || node instanceof HTMLElement && node.isContentEditable;
  function selection(node,value){
   if(node instanceof HTMLInputElement||node instanceof HTMLTextAreaElement){return {start:clamp(node.selectionStart??value.length,0,value.length),end:clamp(node.selectionEnd??value.length,0,value.length),reversed:node.selectionDirection==='backward'}}
   const selected=getSelection();if(!selected?.rangeCount||!node.contains(selected.anchorNode)||!node.contains(selected.focusNode))return {start:value.length,end:value.length,reversed:false};
   const index=(container,offset)=>{const r=document.createRange();r.selectNodeContents(node);r.setEnd(container,offset);return r.toString().length};
   const a=index(selected.anchorNode,selected.anchorOffset),b=index(selected.focusNode,selected.focusOffset);return {start:Math.min(a,b),end:Math.max(a,b),reversed:a>b};
  }
  function caretRect(node,value,selected){
   const box=node.getBoundingClientRect(),style=getComputedStyle(node),height=parseFloat(style.lineHeight)||parseFloat(style.fontSize)||16;
   if(node.isContentEditable){const s=getSelection();if(s?.rangeCount){const range=s.getRangeAt(0).cloneRange();range.collapse(selected.reversed);const r=range.getClientRects()[0];if(r?.height)return {x:r.x,y:r.y,w:Math.max(1,r.width),h:r.height}}}
   // A text-only measurement box has no component, handlers or copied DOM.
   // It follows the browser's own word-wrap, bidi and shaping layout.
   const measure=document.createElement('div'),marker=document.createElement('span');
   for(const name of ['font','fontFamily','fontSize','fontWeight','fontStyle','letterSpacing','lineHeight','textTransform','textIndent','direction','wordSpacing','tabSize','overflowWrap','wordBreak','paddingTop','paddingRight','paddingBottom','paddingLeft','borderTopWidth','borderRightWidth','borderBottomWidth','borderLeftWidth','boxSizing'])measure.style[name]=style[name];
   measure.style.cssText+=';position:fixed;left:-100000px;top:0;visibility:hidden;pointer-events:none;margin:0;border-style:solid;border-color:transparent;';
   measure.style.width=box.width+'px';measure.style.height='auto';
   measure.style.whiteSpace=node instanceof HTMLTextAreaElement?'pre-wrap':'pre';
   measure.style.overflow='visible';
   const edge=selected.reversed?selected.start:selected.end;
   measure.textContent=value.slice(0,edge);marker.textContent='\u200b';measure.appendChild(marker);
   measure.appendChild(document.createTextNode(value.slice(edge)||'\u200b'));measurementRoot().appendChild(measure);
   const measured=measure.getBoundingClientRect(),point=marker.getBoundingClientRect();
   const rect={x:box.x+point.x-measured.x-node.scrollLeft,y:box.y+point.y-measured.y-node.scrollTop,w:Math.max(1,point.width),h:point.height||height};
   measure.remove();return rect;
  }
  function focus(){
   const source=env.source(),node=document.activeElement;if(!source||!editable(node)||!node.isConnected||node.disabled||node.readOnly)return null;
   const record=source.owners.find(row=>row.node.contains(node));if(!record)return null;
   const value=node instanceof HTMLInputElement||node instanceof HTMLTextAreaElement?node.value:node.textContent||'';
   if(value.length>65536)return null;
   const selected=selection(node,value),rect=caretRect(node,value,selected);
   if(![rect.x,rect.y,rect.w,rect.h].every(finite))return null;
   return {documentKey:env.documentKey,subject:source.subject,handle:record.handle,entry:record.entry,turn:record.turn,seq:record.seq,focus:id(node),value,selection:selected,marked:composition?.node===node?composition.range:null,caret:rect};
  }
  function matches(expected,now){return !!expected&&!!now&&expected.documentKey===now.documentKey&&env.same(expected.subject,now.subject)&&expected.handle===now.handle&&expected.entry===now.entry&&expected.turn===now.turn&&expected.seq===now.seq&&expected.focus===now.focus}
  function armValid(value){const allowed=['keydown','keyup','beforeinput','input','compositionstart','compositionupdate','compositionend'];return typeof value?.nonce==='string'&&value.nonce.length>0&&value.nonce.length<=128&&Array.isArray(value.types)&&value.types.length>0&&value.types.length<=8&&value.types.every(type=>allowed.includes(type))}
  function check(event){
   if(cancelled){event.preventDefault();event.stopImmediatePropagation();return}
   if(!armed)return;
   // Original libraries may emit their own untrusted composition notifications
   // while processing native input. Suppress those, without poisoning a receipt
   // already earned by the separately validated trusted browser operation.
   if(!event.isTrusted){event.preventDefault();event.stopImmediatePropagation();return}
   const now=focus(),record=env.find(armed.handle),node=document.activeElement;
   let ok=event.isTrusted&&matches(armed,now)&&record?.node.isConnected&&event.composedPath().includes(node)&&armed.types.includes(event.type);
   if(ok&&(event.type==='keydown'||event.type==='keyup'))ok=(!armed.code||event.code===armed.code)&&(!armed.key||event.key===armed.key);
   if(!ok){event.preventDefault();event.stopImmediatePropagation();receipts.set(armed.nonce,{ok:false,reason:'original-focus-changed:'+event.type+':'+JSON.stringify({trusted:event.isTrusted,focusMatches:matches(armed,now),connected:record?.node.isConnected,path:event.composedPath().includes(node),allowed:armed.types.includes(event.type),active:node?.tagName,target:event.target?.tagName,types:armed.types})});return}
   if(event.type==='compositionstart')composition={node,range:{start:now.selection.start,end:now.selection.end}};
   if(event.type==='compositionupdate'&&composition?.node===node)composition.range.end=composition.range.start+(event.data||'').length;
   if(event.type==='compositionend')composition=null;
   const receipt=receipts.get(armed.nonce)||{ok:true,events:[]};receipt.events.push(event.type);receipts.set(armed.nonce,receipt);
  }
  for(const name of ['keydown','keyup','beforeinput','input','compositionstart','compositionupdate','compositionend'])addEventListener(name,check,{capture:true,passive:false});
  return {
   textFocus:focus,
   textArm(value){const now=focus();if(armed||!armValid(value)||!matches(value,now))return {ok:false};armed={...value};return {ok:true}},
   textDisarm(nonce){const operation=armed?.nonce===nonce?armed:null;if(operation)armed=null;const receipt=receipts.get(nonce)||{ok:false,reason:'no-original-text-event'};receipts.delete(nonce);if(receipt.ok&&operation?.endComposition)composition=null;return {...receipt,focus:focus()}},
   textCancel(key){if(key!==env.documentKey)return false;cancelled=true;armed=null;receipts.clear();return true},
   textResume(key){if(key!==env.documentKey)return false;cancelled=false;composition=null;return true}
  };
 };
 Object.defineProperty(window,'__whaleCreateInlineTextBridge',{value:create,writable:false,configurable:false});
})();
