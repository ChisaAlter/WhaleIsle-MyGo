/* Product-only accepted registry adapter. The authenticated Gateway exposes
 * keys and revisions, never command callbacks, native services or Host RPC. */
function whaleBindProductKeyboard(ctx) {
 const owner=ctx.shortcuts;
 if(!owner?.catalog?.getSnapshot||!owner.config?.getSnapshot||typeof WebSocket==='undefined')return;
 ctx.effect(()=>{
  let stopped=false,composing=false,recording=false,serial=0,revision='',accepted=[],off=[];
  const endpoint=new URL('/__whale_keyboard',location.href);endpoint.protocol=endpoint.protocol==='https:'?'wss:':'ws:';
  const socket=new WebSocket(endpoint.href,'whaleisle.product-keyboard.v1');
  const send=value=>{if(!stopped&&socket.readyState===WebSocket.OPEN)socket.send(JSON.stringify(value))};
  function pageOwnsInput(){
   if(recording||composing||!document.hasFocus()||window!==window.top)return false;
   let target=document.activeElement;
   while(target?.shadowRoot?.activeElement)target=target.shadowRoot.activeElement;
   if(target?.closest?.('input,textarea,select,[contenteditable],.xterm,.terminal,[data-terminal],[data-shortcut-owner],iframe,webview'))return false;
   if(document.querySelector('dialog[open],[aria-modal="true"]'))return false;
   return true;
  }
  function bindings(){
   const rows=owner.catalog.getSnapshot(),out=[];
   if(owner.config.getSnapshot().status!=='ready')return out;
   for(const row of rows){
    const b=row.binding;
    if(!b||b.secondCode||row.issue!==null||row.conflicts?.length)continue;
    // These are already accepted web-registry bindings. Keep its current
    // runtime defaults and persistence, without pretending to be dshDesktop.
    const mods=(b.modifiers.includes('control')?1:0)|(b.modifiers.includes('alt')?2:0)|(b.modifiers.includes('shift')?4:0);
    if(!mods||b.modifiers.includes('meta'))continue;
    if(!/^(Key[A-Z]|Digit[0-9]|Comma|Period|Slash|Backslash|Backquote|Semicolon|Quote|BracketLeft|BracketRight|Minus|Equal|Enter|F([1-9]|1[0-2]))$/.test(b.code))continue;
    out.push({code:b.code,mods});
   }
   return out;
  }
  function publish(){
   accepted=bindings();revision=String(owner.config.getSnapshot().revision)+':'+(++serial);
   send({type:'state',revision,active:pageOwnsInput(),bindings:accepted});
  }
  const originalRecording=owner.recording;
  const record=async function(active){recording=!!active;publish();try{return await originalRecording.call(owner,active)}finally{publish()}};
  if(typeof originalRecording==='function')owner.recording=record;
  const onCompositionStart=()=>{composing=true;publish()},onCompositionEnd=()=>{composing=false;publish()};
  const events=[['focusin',publish],['focusout',publish],['compositionstart',onCompositionStart],['compositionend',onCompositionEnd]];
  for(const [event,fn]of events)document.addEventListener(event,fn,true);
  window.addEventListener('focus',publish);window.addEventListener('blur',publish);
  const observer=typeof MutationObserver==='undefined'?null:new MutationObserver(publish);
  observer?.observe(document.documentElement,{childList:true,subtree:true,attributes:true,attributeFilter:['role','aria-modal','contenteditable','inert']});
  off.push(owner.catalog.subscribe(publish),owner.config.subscribe(publish));
  socket.addEventListener('open',publish);
  socket.addEventListener('message',event=>{
   let input;try{input=JSON.parse(event.data)}catch{return}
   if(stopped||input.type!=='key'||input.revision!==revision||!pageOwnsInput()||!accepted.some(k=>k.code===input.code&&k.mods===input.mods))return;
   // Re-enter the original DOM dispatcher and feature-owned resolvers. This
   // synthetic event has no browser default, so native interception cannot
   // also open Print/Find/etc. Local page listeners retain preventDefault.
   window.dispatchEvent(new KeyboardEvent('keydown',{code:input.code,key:input.code.startsWith('Key')?input.code.slice(3):input.code,
    ctrlKey:!!(input.mods&1),altKey:!!(input.mods&2),shiftKey:!!(input.mods&4),metaKey:false,
    repeat:false,isComposing:false,bubbles:true,cancelable:true,composed:true}));
  });
  function dispose(){if(stopped)return;stopped=true;for(const fn of off)fn();observer?.disconnect();for(const [event,fn]of events)document.removeEventListener(event,fn,true);window.removeEventListener('focus',publish);window.removeEventListener('blur',publish);window.removeEventListener('pagehide',dispose);if(owner.recording===record)owner.recording=originalRecording;socket.close()}
  socket.addEventListener('close',dispose);socket.addEventListener('error',dispose);window.addEventListener('pagehide',dispose);
  return dispose;
 });
}
