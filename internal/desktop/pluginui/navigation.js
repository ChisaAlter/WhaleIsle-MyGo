/* Native navigation follows actual registered root owners and their navigation
 * services. It never independently mounts a component lacking owner props. */
/* Go omits an empty Subject name, so a subject can arrive without one. */
function whaleNavigationSubject(subject){return {...subject,name:String(subject.name||'')}}
function whaleSameNavigationSubject(a,b){return !!a&&!!b&&a.kind===b.kind&&String(a.name||'')===String(b.name||'')}
/* The bundle registers its own settings.plugins.tab (the inventory "all" tab); Go renders that natively, so only a plugin's entries are hosted. */
function whaleHostsEntry(slot,known){
 if(known.stock)return false;
 if(slot==='settings.plugins.tab')return true;
 return !(known.nativeDefault||slot==='settings.section'&&known.nativeSetting)
}
function whaleBindNativeNavigation(ctx){
 if(new URLSearchParams(location.search).get('desktopNativeNavigation')!=='1')return;
 ctx.effect(()=>{
  let stopped=false,revision=0,last='',serial=0,pending=false,requested={kind:'navigation',name:''},activate=true;
  const ids=new WeakMap(),actions=new Map(),declared=new Map();
  const handle=node=>{let id=ids.get(node);if(!id){id=String(++serial);ids.set(node,id)}return id};
  const endpoint=new URL('/__whale_plugin_ui/events',location.href);endpoint.protocol=endpoint.protocol==='https:'?'wss:':'ws:';
  const socket=new WebSocket(endpoint.href,'whaleisle.plugin-ui.v1');
  const send=message=>{if(!stopped&&socket.readyState===WebSocket.OPEN)socket.send(JSON.stringify(message))};
  const workspace=whaleWorkspaceBridge(ctx,schedule,send);
  const nativeDefaults=new Set(['@deepseek-ai/dsh-client-ui-layout','@deepseek-ai/dsh-client-ui-conversation','@deepseek-ai/dsh-client-ui-settings','@deepseek-ai/dsh-client-ui-sidebar','@deepseek-ai/dsh-client-ui-sidebar-right','@deepseek-ai/dsh-client-ui-chat','@deepseek-ai/dsh-client-ui-tool','whaleisle-desktop-plugin-views','whaleisle-native-plugin-ui','dsh-whale']);
  const nativeShell=new Set(['@deepseek-ai/dsh-client-ui-git','@deepseek-ai/dsh-client-ui-titlebar','@deepseek-ai/dsh-client-ui-user-terminal']);
  const nativeSettings=new Set(['@deepseek-ai/dsh-client-ui-settings-account','@deepseek-ai/dsh-client-ui-settings-general','@deepseek-ai/dsh-client-ui-theme','@deepseek-ai/dsh-client-ui-settings-models','@deepseek-ai/dsh-client-ui-settings-plugins','@deepseek-ai/dsh-client-ui-settings-skills','@deepseek-ai/dsh-client-ui-agent-preset','@deepseek-ai/dsh-client-ui-settings-mcp','@deepseek-ai/dsh-client-ui-settings-market','@deepseek-ai/dsh-client-ui-settings-remote','dsh-remote','dsh-usage-panel']);
  const label=entry=>{try{return String(typeof entry.options.label==='function'?entry.options.label():entry.options.label||entry.options.id||entry.options.key||'').slice(0,8192)}catch{return ''}};
  const entries=slot=>ctx.slots.entriesOfSlot(slot);const stock=entry=>{const r=String(entry.registrant||entry.options.registrant||'');return r!==''&&r===whaleBundledRegistrant(entries)};
  const publicSlots=whaleCreatePublicSlots(ctx,{entries,handle,schedule});
  function registry(){
   const result=[];declared.clear();
   for(const slot of ['main','sidebar.page','settings.section','settings.remote.tab','settings.plugins.tab'])for(const entry of slot==='settings.plugins.tab'?ctx.slots.entries(slot):entries(slot)){
    const declaredBy=entry.registrant||entry.options.registrant;if(!whaleHostsEntry(slot,{stock:stock(entry),nativeDefault:nativeDefaults.has(declaredBy),nativeSetting:nativeSettings.has(declaredBy)}))continue;
    const id=entry.options.key||entry.options.id;if(!id)continue;
    const identity=[slot,String(id),String(entry.registrant||entry.options.registrant||'')];
    const key=slot==='settings.plugins.tab'?JSON.stringify(identity):identity.join(':');
    const tab=slot==='sidebar.page'?entries('sidebar.nav.tab').find(tab=>tab.options.id===id):null;
    const view={handle:key,slot,id:String(id),label:label(tab||entry)||String(id),registrant:String(entry.registrant||entry.options.registrant||''),order:Number.isFinite(entry.options.order)?Math.trunc(entry.options.order):0};
    declared.set(key,{entry,view});result.push(view);
   }
   return result;
  }
  function open(record){
   if(!activate||!record)return;
   const view=record.view;
   if(view.slot==='main')ctx.layout.selectPanel(view.id);
   else if(view.slot==='settings.section')ctx.get('settingsNavigation')?.open(view.id);
   else if(view.slot==='settings.plugins.tab'){
    const navigation=ctx.get('settingsNavigation');if(!navigation?.open)return;
    navigation.open('plugins');
    const matches=[...document.querySelectorAll('[data-dsh-settings-plugin-tab]')].filter(node=>node.getAttribute('data-dsh-settings-plugin-tab')===view.id&&!node.closest('[hidden],[aria-hidden="true"]'));
    if(matches.length===1)matches[0].click();
    else{
     // The original one-page section has no tab chrome; its only registered
     // page mounts directly when Settings navigation opens Plugins.
     const tabs=ctx.slots.entries('settings.plugins.tab');
     if(tabs.length!==1||tabs[0]!==record.entry)return;
    }
   }
   else{
    if(view.slot==='settings.remote.tab')ctx.get('settingsNavigation')?.open('remote');
    const matches=[...document.querySelectorAll('button[role="tab"]')].filter(node=>view.slot==='settings.remote.tab'?node.getAttribute('data-dsh-settings-remote-tab')===view.id:node.getAttribute('aria-label')===view.label&&!node.hasAttribute('data-dsh-settings-remote-tab'));
    if(matches.length!==1)return;matches[0].click();
   }
   activate=false;
  }
  function publish(){
   if(stopped)return;
   const workspaceState=workspace.snapshot();
   const views=registry(),record=declared.get(requested.name);open(record);
   if(activate&&requested.kind==='navigation'&&(requested.name==='models'||requested.name.startsWith('settings:'))){ctx.get('settingsNavigation')?.open(requested.name==='models'?'models':requested.name.slice(9));activate=false;}
   actions.clear();const documents=[],globalDocuments=[],modelDocuments=[],budget={left:512};
   if(record&&typeof window.__whaleSessionEntryOf==='function'){
    const matches=[...document.querySelectorAll('[data-whale-slot-entry]')].filter(node=>window.__whaleSessionEntryOf(node,record.view.slot)===record.entry&&!node.closest('[hidden],[aria-hidden="true"]'));
    if(matches.length===1)documents.push({handle:record.view.handle,slot:record.view.slot,nodes:whaleNativeDocument(matches[0],handle,actions,{root:matches[0]}, {left:512})});
   }
   if(typeof window.__whaleSessionEntryOf==='function')for(const slot of ['shell.overlay','shell.titlebar.trailing','shell.terminalDrawer','shell.bottom'])for(const entry of entries(slot)){
    if(nativeDefaults.has(entry.registrant||entry.options.registrant)||stock(entry)||nativeShell.has(entry.registrant||entry.options.registrant)||entry.options.id==='dshd-mini-player')continue;
    const matches=[...document.querySelectorAll('[data-whale-slot-entry]')].filter(node=>window.__whaleSessionEntryOf(node,slot)===entry&&!node.hidden&&!node.closest('[hidden],[aria-hidden="true"]'));
    for(const root of matches)globalDocuments.push({slot,handle:handle(root),nodes:whaleNativeDocument(root,handle,actions,{root,globalEntry:entry,globalSlot:slot},budget)});
   }
   if(requested.kind==='navigation'&&requested.name==='models'&&typeof window.__whaleSessionEntryOf==='function')for(const slot of ['settings.models.provider-card','settings.models.footer'])for(const entry of entries(slot)){
    for(const root of [...document.querySelectorAll('[data-whale-slot-entry]')].filter(node=>window.__whaleSessionEntryOf(node,slot)===entry&&!node.closest('[hidden],[aria-hidden="true"]'))){
     modelDocuments.push({slot,handle:handle(root),provider:root.getAttribute('data-whale-provider')||'',namespace:root.getAttribute('data-whale-provider-ns')||'',nodes:whaleNativeDocument(root,handle,actions,{root,globalEntry:entry,globalSlot:slot},{left:512})});
    }
   }
   const electedRoot=entries('root')[0];if(electedRoot&&!nativeDefaults.has(electedRoot.registrant||electedRoot.options.registrant)&&!stock(electedRoot)&&whaleStockReady(entries)&&typeof window.__whaleSessionEntryOf==='function'){
    const roots=[...document.querySelectorAll('[data-whale-slot-entry]')].filter(node=>window.__whaleSessionEntryOf(node,'root')===electedRoot&&!node.closest('[hidden],[aria-hidden="true"]'));
    if(roots.length===1)globalDocuments.push({slot:'root',handle:handle(roots[0]),nodes:whaleNativeDocument(roots[0],handle,actions,{root:roots[0],globalEntry:electedRoot,globalSlot:'root'}, {left:512})});
   }
   const snapshot={...workspaceState,...publicSlots.snapshot(actions),modelDocuments,globalDocuments,locale:ctx.locale.getSnapshot().active,entries:[],subject:record||requested.kind==='navigation'?requested:null,views,documents,actions:[],badges:[],section:false,configuration:false,surface:'',height:0};
   const encoded=JSON.stringify(snapshot);if(encoded===last||socket.readyState!==WebSocket.OPEN)return;last=encoded;revision++;socket.send(JSON.stringify({type:'state',revision,...snapshot}));
  }
  const richDispose=whaleRichSurface.bind(()=>({revision,subject:requested,actions,valid:record=>declared.has(requested.name)||record.owner.globalEntry&&entries(record.owner.globalSlot||'shell.overlay').includes(record.owner.globalEntry)}));
  function schedule(){if(stopped||pending)return;pending=true;queueMicrotask(()=>{pending=false;publish()})}
  socket.addEventListener('open',schedule);
  socket.addEventListener('message',event=>{let command;try{command=JSON.parse(event.data)}catch{return}
   if(command.type==='publicWorkspaceContext'){publicSlots.workspace(command);return;}
   if(command.type==='publicRowContext'){publicSlots.context(command);return;}
   if(command.type==='publicAction'&&command.revision===revision&&whaleSameNavigationSubject(command.subject,requested)){if(publicSlots.invoke(command))schedule();return;}
   if(command.type==='navigationContext'){workspace.context(command.sessionId);return;}
   if(command.type==='modelTargets'&&Array.isArray(command.providers)){window.__whaleModelTargets=command.providers.filter(id=>typeof id==='string');window.dispatchEvent(new Event('whale-model-targets'));schedule();return;}
   if(command.type==='workspaceAction'){
    if(command.revision!==revision){send({type:'workspaceResult',request:command.request,error:'Workspace source changed. Try again.'});return;}
    void workspace.invoke(command);return;
   }
   if(command.type==='richRelease'){whaleRichSurface.release(command);return;}
if(command.type==='richLayout'){whaleRichSurface.ack(command,reply=>socket.readyState===WebSocket.OPEN&&socket.send(JSON.stringify(reply)));return}
   if(command.type==='select'&&['view','navigation'].includes(command.subject?.kind)){whaleRichSurface.clear();const settingsWasOpen=requested.kind==='navigation'&&(requested.name==='models'||requested.name.startsWith('settings:'))||declared.get(requested.name)?.view.slot.startsWith('settings.');requested=whaleNavigationSubject(command.subject);if(settingsWasOpen&&requested.kind==='navigation'&&!requested.name)ctx.get('settingsNavigation')?.close();activate=true;schedule();return}
   if(command.type==='nativeInput'&&command.revision===revision&&whaleSameNavigationSubject(command.subject,requested)){
    const target=actions.get(command.handle);if((declared.has(requested.name)||target?.owner.globalEntry&&entries(target.owner.globalSlot||'shell.overlay').includes(target.owner.globalEntry))&&whaleNativeContains(target?.owner,target?.node)&&whaleNativeInteraction(command,target))schedule();
   }
  });
  const observer=new MutationObserver(schedule);observer.observe(document.documentElement,{childList:true,subtree:true,characterData:true,attributes:true,attributeFilter:['disabled','aria-disabled','aria-expanded','hidden','aria-hidden']});
  document.addEventListener('whale-rich-portals',schedule);
  window.addEventListener('whale-workspace-actions',schedule);
  const cleanup=[()=>{publicSlots.dispose();workspace.dispose();richDispose();document.removeEventListener('whale-rich-portals',schedule);window.removeEventListener('whale-workspace-actions',schedule)},ctx.on('slots/changed',schedule),ctx.locale.subscribe(schedule),ctx.layout.panelInfo.subscribe(schedule),()=>observer.disconnect()];
  function dispose(){if(stopped)return;stopped=true;cleanup.reverse().forEach(fn=>fn());window.removeEventListener('pagehide',dispose);socket.close()}
  socket.addEventListener('close',dispose);socket.addEventListener('error',dispose);window.addEventListener('pagehide',dispose);schedule();return dispose;
 });
}
