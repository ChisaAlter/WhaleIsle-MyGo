/* Product-only accepted registry adapter. The authenticated Gateway exposes
 * keys and revisions, never command callbacks, native services or Host RPC. */
function whaleBindProductKeyboard(ctx) {
 const owner=ctx.shortcuts;
 if(!owner?.catalog?.getSnapshot||!owner.config?.getSnapshot||typeof WebSocket==='undefined')return;
 ctx.effect(()=>{
  let stopped=false,composing=false,recording=false,serial=0,revision='',accepted=[],off=[],held=new Set(),consumed=new Set(),lastMods=-1,acceptedFrame=null,focusTimer; 
  const endpoint=new URL('/__whale_keyboard',location.href);endpoint.protocol=endpoint.protocol==='https:'?'wss:':'ws:';
  const socket=new WebSocket(endpoint.href,'whaleisle.product-keyboard.v1');
  const send=value=>{if(!stopped&&socket.readyState===WebSocket.OPEN)socket.send(JSON.stringify(value))};
  function focusedFrame(){
   let target=document.activeElement;
   while(target?.shadowRoot?.activeElement)target=target.shadowRoot.activeElement;
   if(!window.whaleisleShortcutProvider?.guestKeyboard?.ready||!target?.isConnected
    ||!target.matches?.('iframe[data-sidebar-browser-frame],iframe[data-html-preview]')
    ||typeof target.name!=='string'||!target.name)return null;
   return {element:target,name:target.name};
  }
  function pageOwnsInput(){
   if(recording||composing||!document.hasFocus()||window!==window.top)return false;
   let target=document.activeElement;
   while(target?.shadowRoot?.activeElement)target=target.shadowRoot.activeElement;
   if(document.querySelector('dialog[open],[aria-modal="true"]'))return false;
   if(focusedFrame())return true;
   if(target?.closest?.('input,textarea,select,[contenteditable],.xterm,.terminal,[data-terminal],[data-shortcut-owner],iframe,webview'))return false;
   return true;
  }
  function bindings(){
   const rows=owner.catalog.getSnapshot(),out=[];
   if(owner.config.getSnapshot().status!=='ready')return out;
   for(const row of rows){
    const b=row.binding;
    if(!b||row.issue!==null||row.conflicts?.length)continue;
    // These are already accepted product-registry bindings. Keep its current
    // runtime defaults and persistence, without pretending to be dshDesktop.
    const mods=(b.modifiers.includes('control')?1:0)|(b.modifiers.includes('alt')?2:0)|(b.modifiers.includes('shift')?4:0)|(b.modifiers.includes('meta')?8:0);
    const codeOK=c=>/^(Key[A-Z]|Digit[0-9]|Comma|Period|Slash|Backslash|Backquote|Semicolon|Quote|BracketLeft|BracketRight|Minus|Equal|Enter|Escape|Space|Tab|Backspace|Delete|Arrow(Up|Down|Left|Right)|F([1-9]|1[0-9]|2[0-4]))$/.test(c);
    if(!codeOK(b.code)||(b.secondCode&&(!codeOK(b.secondCode)||b.code>=b.secondCode)))continue;
    out.push({code:b.code,mods,...(b.secondCode?{secondCode:b.secondCode}:{})});
   }
   return out;
  }
  function publish(){
   held.clear();consumed.clear();lastMods=-1;
   acceptedFrame=focusedFrame();
   accepted=bindings();revision=String(owner.config.getSnapshot().revision)+':'+(++serial);
   send({type:'state',revision,active:pageOwnsInput(),bindings:accepted});
  }
  const originalRecording=owner.recording;
	const originalCloseWindow=owner.closeWindow;
	const closeWindow=async function(){if(pageOwnsInput()&&revision)send({type:'close-window',revision})};
	if(typeof originalCloseWindow==='function')owner.closeWindow=closeWindow;
  const record=async function(active){recording=!!active;publish();try{return await originalRecording.call(owner,active)}finally{publish()}};
  if(typeof originalRecording==='function')owner.recording=record;
  const onCompositionStart=()=>{composing=true;publish()},onCompositionEnd=()=>{composing=false;publish()};
  const events=[['focusin',publish],['focusout',publish],['compositionstart',onCompositionStart],['compositionend',onCompositionEnd]];
  for(const [event,fn]of events)document.addEventListener(event,fn,true);
  const onBlur=()=>{publish();clearTimeout(focusTimer);focusTimer=setTimeout(publish,0)};
  window.addEventListener('focus',publish);window.addEventListener('blur',onBlur);
  window.addEventListener('whaleisle-guest-keyboard-ready',publish);
  const observer=typeof MutationObserver==='undefined'?null:new MutationObserver(publish);
  observer?.observe(document.documentElement,{childList:true,subtree:true,attributes:true,attributeFilter:['role','aria-modal','contenteditable','inert','name','src','data-html-preview','data-sidebar-browser-frame']});
  off.push(owner.catalog.subscribe(publish),owner.config.subscribe(publish));
  socket.addEventListener('open',publish);
  socket.addEventListener('message',event=>{
   let input;try{input=JSON.parse(event.data)}catch{return}
   if(stopped||input.type!=='key'||input.revision!==revision||!pageOwnsInput()||!accepted.some(k=>k.code===input.code&&k.mods===input.mods&&k.secondCode===input.secondCode))return;
   const frame=focusedFrame();
   if(acceptedFrame||frame){
    if(!acceptedFrame||!frame||acceptedFrame.element!==frame.element||acceptedFrame.name!==frame.name)return;
    held.clear();consumed.clear();
    // Top-product subscription only. No postMessage, guest script or preview RPC.
    // The original native.ts subscriber independently rechecks frame ownership.
    window.dispatchEvent(new CustomEvent('whaleisle-native-frame-key',{detail:{kind:'iframe',
     revision:String(owner.config.getSnapshot().revision),frameName:frame.name,code:input.code,
     ...(input.secondCode?{secondCode:input.secondCode}:{}),control:!!(input.mods&1),alt:!!(input.mods&2),
     shift:!!(input.mods&4),meta:!!(input.mods&8),repeat:false}}));
    return;
   }
   // Re-enter the original DOM dispatcher and feature-owned resolvers. This
   // synthetic event has no browser default, so native interception cannot
   // also open Print/Find/etc. Local page listeners retain preventDefault.
   held.clear();consumed.clear();
   const key=new KeyboardEvent('keydown',{code:input.code,key:input.code.startsWith('Key')?input.code.slice(3):input.code,
    ctrlKey:!!(input.mods&1),altKey:!!(input.mods&2),shiftKey:!!(input.mods&4),metaKey:!!(input.mods&8),
    repeat:false,isComposing:false,bubbles:true,cancelable:true,composed:true});
   if(input.secondCode)Object.defineProperty(key,'whaleisleSecondCode',{value:input.secondCode});
   window.dispatchEvent(key);
  });
  // WebView2 only reports accelerator keys. Ordinary unmodified physical
  // keys therefore retain DOM delivery; annotate only an accepted held pair
  // before the original dispatcher runs, after which local handlers can veto.
  function physical(event){
   if(!event.isTrusted)return;
   const mods=(event.ctrlKey?1:0)|(event.altKey?2:0)|(event.shiftKey?4:0)|(event.metaKey?8:0);
   const modifier=/^(Control|Alt|Shift|Meta)(Left|Right)$/.test(event.code);
   if(!pageOwnsInput()||event.isComposing||event.key==='Dead'||event.getModifierState?.('AltGraph')){held.clear();consumed.clear();return}
   if(event.type==='keyup'){held.delete(event.code);consumed.delete(event.code);if(modifier)held.clear();return}
   if(mods!==lastMods||modifier){held.clear();lastMods=mods}
   if(modifier)return;
   if(event.repeat){if(consumed.has(event.code))event.preventDefault();return}
   consumed.delete(event.code);held.add(event.code);
   if(held.size!==2)return;
   const codes=[...held].sort();
   if(!accepted.some(k=>k.mods===mods&&k.code===codes[0]&&k.secondCode===codes[1]))return;
   // Preserve the triggering event's code; the registry canonicalizes the pair.
   Object.defineProperty(event,'whaleisleSecondCode',{value:codes.find(c=>c!==event.code)});
   for(const code of codes)consumed.add(code);
   held.clear();
  }
  window.addEventListener('keydown',physical,true);window.addEventListener('keyup',physical,true);
  function dispose(){if(stopped)return;stopped=true;held.clear();consumed.clear();window.removeEventListener('keydown',physical,true);window.removeEventListener('keyup',physical,true);for(const fn of off)fn();observer?.disconnect();for(const [event,fn]of events)document.removeEventListener(event,fn,true);window.removeEventListener('focus',publish);window.removeEventListener('blur',onBlur);clearTimeout(focusTimer);window.removeEventListener('whaleisle-guest-keyboard-ready',publish);window.removeEventListener('pagehide',dispose);if(owner.recording===record)owner.recording=originalRecording;if(owner.closeWindow===closeWindow)owner.closeWindow=originalCloseWindow;socket.close()}
  socket.addEventListener('close',dispose);socket.addEventListener('error',dispose);window.addEventListener('pagehide',dispose);
  return dispose;
 });
}
