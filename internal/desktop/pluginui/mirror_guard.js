// Installed at document creation before original renderer/plugin listeners.
// All native projections refer to the retained original DOM and ownership.
(() => {
 if(window!==window.top)return;
 const documentKey=crypto.randomUUID();
 let source=null,armed=null,revision=0,last='',closed=false,cancelled=false,dirty=0;
 const receipts=new Map(),intersect=(a,b)=>{const x=Math.max(a.x,b.x),y=Math.max(a.y,b.y),right=Math.min(a.x+a.w,b.x+b.w),bottom=Math.min(a.y+a.h,b.y+b.h);return {x,y,w:Math.max(0,right-x),h:Math.max(0,bottom-y)}};
 const portalBridge=window.__whaleCreateInlinePortalBridge({documentKey,source:()=>source?.(),same:(a,b)=>same(a,b)});
 const asRect=r=>({x:r.x,y:r.y,w:r.width,h:r.height});
 function fragment(node,rect){
  let visible=intersect(rect,{x:0,y:0,w:innerWidth,h:innerHeight});
  for(let parent=node;parent instanceof Element;parent=parent.parentElement){const style=getComputedStyle(parent);if(parent.hidden||style.display==='none'||style.visibility==='hidden'||Number(style.opacity)===0)return {x:0,y:0,w:0,h:0};if(parent!==node&&(/hidden|clip|auto|scroll/.test(style.overflowX)||/hidden|clip|auto|scroll/.test(style.overflowY))){const bounds=parent.getBoundingClientRect(),clip={x:/hidden|clip|auto|scroll/.test(style.overflowX)?bounds.x:visible.x,y:/hidden|clip|auto|scroll/.test(style.overflowY)?bounds.y:visible.y,w:/hidden|clip|auto|scroll/.test(style.overflowX)?bounds.width:visible.w,h:/hidden|clip|auto|scroll/.test(style.overflowY)?bounds.height:visible.h};visible=intersect(visible,clip)}}
  return visible;
 }
 function snapshot(){
  const value=source?.();
  if(!value)return {documentKey,revision,closed:true,owners:[],portals:[],dirty};
  const viewport={w:innerWidth,h:innerHeight,offsetX:visualViewport?.offsetLeft||0,offsetY:visualViewport?.offsetTop||0,scale:visualViewport?.scale||1};
  const owners=value.owners.map(record=>{const range=document.createRange();range.selectNodeContents(record.node);const r=range.getBoundingClientRect();range.detach();const rect=asRect(r),visibleRect=fragment(record.node,rect);return {handle:record.handle,entry:record.entry,label:record.label,turn:record.turn,seq:record.seq,rect,visibleRect,connected:record.node.isConnected,visible:record.node.isConnected&&visibleRect.w>0&&visibleRect.h>0,text:record.node.textContent.slice(0,65536)}});
  const portals=portalBridge.portals().map(row=>({...row,visibleRect:intersect(row.rect,{x:0,y:0,w:innerWidth,h:innerHeight})}));
  const state={subject:value.subject,viewport,owners,portals};
  const encoded=JSON.stringify(state);if(encoded!==last){last=encoded;revision++}
  // Geometry revision stays stable while pixels animate; dirty controls capture.
  const animated=value.owners.some(record=>record.node.getAnimations?.({subtree:true}).some(animation=>animation.playState==='running'));
  return {documentKey,revision,closed,subject:value.subject,viewport,owners,portals,dirty,animated};
 }
 const find=handle=>source?.()?.owners.find(row=>row.handle===handle);
 function same(a,b){return a?.kind==='session'&&b?.kind==='session'&&a.sessionId===b.sessionId&&a.epoch===b.epoch}
 function current(a,now){return !closed&&a&&a.documentKey===now.documentKey&&a.revision===now.revision&&same(a.subject,now.subject)&&now.viewport?.scale===1&&now.viewport.offsetX===0&&now.viewport.offsetY===0}
 function target(a,now,hit){const record=find(a.handle),row=now.owners.find(row=>row.handle===a.handle);if(!record?.node.isConnected||row?.entry!==a.entry||row.turn!==a.turn||row.seq!==a.seq)return false;const area=a.portal?now.portals.find(row=>row.token===a.portal&&row.handle===a.handle):row;if(!area?.visible)return false;const r=area.visibleRect;if(a.x<r.x||a.y<r.y||a.x>r.x+r.w||a.y>r.y+r.h)return false;return a.portal?portalBridge.portalTarget(a.portal,hit):record.node.contains(hit)}
 function check(event){
  if(cancelled){event.preventDefault();event.stopImmediatePropagation();return}
  if(!armed)return;
  const a=armed,now=snapshot();
  const ok=event.isTrusted&&current(a,now)&&target(a,now,event.target)&&Math.abs(event.clientX-a.x)<=1&&Math.abs(event.clientY-a.y)<=1&&a.types.includes(event.type);
  if(!ok){event.preventDefault();event.stopImmediatePropagation();receipts.set(a.nonce,{ok:false,reason:'target-or-layout-changed'});return}
  const receipt=receipts.get(a.nonce)||{ok:true,events:[]};receipt.events.push(event.type);receipts.set(a.nonce,receipt);
 }
 for(const type of ['pointerdown','pointerup','pointermove','mousedown','mouseup','mousemove','click','dblclick','contextmenu','wheel'])addEventListener(type,check,{capture:true,passive:false});
 const textBridge=window.__whaleCreateInlineTextBridge({documentKey,source:()=>source?.(),find,same});
 new MutationObserver(records=>{if(records.some(record=>source?.()?.owners.some(owner=>owner.node===record.target||owner.node.contains(record.target))))dirty++}).observe(document,{childList:true,subtree:true,characterData:true,attributes:true});
 for(const type of ['input','change','scroll','resize','load','transitionend','animationend'])addEventListener(type,()=>{dirty++},{capture:true,passive:true});
 const api=Object.freeze({
  ...textBridge,...portalBridge,snapshot,
  bind(get){if(source)throw Error('inline source already bound');source=get},
  arm(a){const now=snapshot();if(armed||!a.nonce||!current(a,now)||!target(a,now,document.elementFromPoint(a.x,a.y)))return {ok:false};armed={...a};return {ok:true}},
  disarm(nonce){if(armed?.nonce===nonce)armed=null;const receipt=receipts.get(nonce)||{ok:false,reason:'no-browser-event'};receipts.delete(nonce);return receipt},
  cancel(key){if(key!==documentKey)return false;cancelled=true;armed=null;receipts.clear();return true},
  resume(key){if(key!==documentKey)return false;cancelled=false;return true},
  close(){closed=true;portalBridge.close();source=null;armed=null;receipts.clear()}
 });
 Object.defineProperty(window,'__whaleInlineMirror',{value:api,writable:false,configurable:false});
})();
