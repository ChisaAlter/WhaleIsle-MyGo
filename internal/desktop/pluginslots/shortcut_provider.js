/* A real, narrow product capability. Only the build-adapted shortcuts service
 * reads it. This neither creates dshDesktop nor changes global platform marks. */
(()=>{
 if(window!==window.top||location.origin!==__MYGO_ORIGIN__)return;
 let queue=Promise.resolve(),listeners=new Set(),timer=null,latest=-1,closed=false;
 const publish=s=>{if(s&&s.sequence>latest){latest=s.sequence;for(const f of listeners)f(s)};return s};
 const request=body=>{
  const run=async()=>{
   if(closed)throw Error('Shortcut source closed');
   const r=await fetch('/__whale_shortcuts',{method:'POST',credentials:'same-origin',cache:'no-store',redirect:'error',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
   if(!r.ok)throw Error('Shortcut preferences unavailable');
   return r.json();
  };
  const result=queue.then(run);queue=result.catch(()=>{});return result;
 };
 // A top-product-only subscription, not a command invocation API. The key
 // transport supplies physical gestures; original native.ts checks live frame
 // identity, focus and accepted preference revision again before dispatch.
 const guestListeners=new Set();
 const guestKeyboard=Object.freeze({
  subscribe(listener){guestListeners.add(listener);window.dispatchEvent(new CustomEvent('whaleisle-guest-keyboard-ready'));return ()=>{guestListeners.delete(listener);window.dispatchEvent(new CustomEvent('whaleisle-guest-keyboard-ready'))}},
  get ready(){return guestListeners.size>0},
 });
 window.addEventListener('whaleisle-native-frame-key',event=>{
  const input=event.detail;
  if(closed||!input||input.kind!=='iframe'||typeof input.frameName!=='string'||!input.frameName||typeof input.revision!=='string'||typeof input.code!=='string')return;
  // Never accept menu/commandId, postMessage, frame RPC, or arbitrary actions.
  const gesture={kind:'iframe',revision:input.revision,frameName:input.frameName,code:input.code,
   ...(typeof input.secondCode==='string'?{secondCode:input.secondCode}:{}),
   control:input.control===true,alt:input.alt===true,shift:input.shift===true,meta:input.meta===true,repeat:false};
  for(const listener of guestListeners)listener(gesture);
 });
 const api=Object.freeze({
  get:definitions=>request({operation:'get',definitions}).then(publish),
  edit:(edit,revision)=>request({operation:'edit',edit,revision}).then(r=>{publish(r.snapshot);return r}),
  subscribe:listener=>{
   listeners.add(listener);
   if(timer===null)timer=setInterval(()=>{request({operation:'snapshot'}).then(publish).catch(()=>{})},750);
   return ()=>{listeners.delete(listener);if(!listeners.size&&timer!==null){clearInterval(timer);timer=null}};
  },
  // Recording ownership remains with the existing accepted-registry adapter.
  recording:async active=>{window.dispatchEvent(new CustomEvent('whaleisle-shortcut-recording',{detail:!!active}))},
 });
 Object.defineProperty(window,'whaleisleShortcutProvider',{value:Object.freeze({version:1,platform:__MYGO_PLATFORM__,shortcuts:api,guestKeyboard}),writable:false,configurable:false});
 window.addEventListener('pagehide',()=>{closed=true;listeners.clear();guestListeners.clear();if(timer!==null)clearInterval(timer)},{once:true});
})();
