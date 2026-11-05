/* Public families omitted by navigation are observed on their real renderer
 * entries. A source menu/hover child exists only while its actual owner mounts. */
const whaleBundledMarker='whaleisle:bundled';
const whaleStockSections=['general','appearance','interface','models','plugins','skills','mcp','agent-presets','usage-stats','about'];
/* Stock conversation fibers that carry their own names instead of the bundle's minified registrant. */
const whaleStockFibers=['conversation-todo-dock','conversation-queue-dock'];
/* The Harness web bundle registers its own UI under one minified registrant that no package list can name; plugins loaded from node_modules keep their own. Go replaces the bundled UI natively, so only entries from other registrants are plugin contributions. The bundled name is recovered from the settings sections that only the bundle registers. */
function whaleBundledRegistrant(entriesOf){
 const counts=new Map();
 for(const entry of entriesOf('settings.section')){
  if(!whaleStockSections.includes(String(entry.options&&entry.options.id||'')))continue;
  const registrant=String(entry.registrant||entry.options.registrant||'');
  if(registrant===''||registrant.startsWith('@deepseek-ai/'))continue;
  counts.set(registrant,(counts.get(registrant)||0)+1);
 }
 let best='',most=2;
 for(const [registrant,count] of counts)if(count>most){best=registrant;most=count}
 return best;
}
function whaleStockReady(entriesOf){return entriesOf('settings.section').some(entry=>whaleStockSections.includes(String(entry.options&&entry.options.id||'')))}
function whaleReportedRegistrant(entry,entriesOf){const registrant=String(entry.registrant||'').trim().slice(0,8192);return registrant!==''&&(registrant===whaleBundledRegistrant(entriesOf)||whaleStockFibers.includes(registrant))?whaleBundledMarker:registrant}
function whaleCreatePublicSlots(ctx,{entries,handle,schedule}){
 const slots=['sidebar.brand.mark','sidebar.brand.name','sidebar.toggle.badge','sidebar.panellist','sidebar.footer.action','sidebar.settings','sidebar.workspaces','sidebar.workspaces.sections','sidebar.session.row.leading','sidebar.session.row.badge','sidebar.session.row.hover','sidebar.workspaces.session.row.action','sidebar.workspaces.session.menu.item','settings.header','settings.action','settings.close','settings.trigger','settings.launcher','settings.launcher.action','settings.general.item','settings.interface.item','settings.appearance.item','settings.conversation.item','settings.features.item','settings.pet.item','settings.usage.item'];
 const menus=new Map(),contexts=new Map();let workspaceContext=null;
 let bundledName='';const module=entry=>{const r=String(entry.registrant||entry.options.registrant||'');return r!==''&&(r===bundledName||whaleStockFibers.includes(r))?whaleBundledMarker:r};
 const builtinSettings=new Set(['@deepseek-ai/dsh-client-locale','@deepseek-ai/dsh-client-ui-git','@deepseek-ai/dsh-client-ui-titlebar','@deepseek-ai/dsh-client-ui-permission-presets','@deepseek-ai/dsh-client-ui-visual-replies','@deepseek-ai/dsh-client-ui-settings-general','@deepseek-ai/dsh-client-ui-settings-session-log','@deepseek-ai/dsh-client-ui-shortcuts','@deepseek-ai/dsh-client-ui-conversation','@deepseek-ai/dsh-client-ui-chat','@deepseek-ai/dsh-client-ui-workspace','@deepseek-ai/dsh-client-ui-theme','dsh-whale','dsh-remote','dsh-project','dsh-usage-panel']);
 const native=(slot,entry)=>{
  const m=module(entry);if(m===whaleBundledMarker)return true;
  if(/^settings\..*\.item$/.test(slot))return builtinSettings.has(m);
  if(slot==='settings.action')return m==='@deepseek-ai/dsh-client-ui-settings-general'&&entry.options.id==='open-document';
  if(['settings.header','settings.close','settings.trigger','sidebar.settings','sidebar.toggle.badge'].includes(slot))return m==='@deepseek-ai/dsh-client-ui-settings-general';
  if(slot==='settings.launcher')return m==='@deepseek-ai/dsh-client-ui-settings-account';
  if(slot==='sidebar.workspaces')return ['@deepseek-ai/dsh-client-ui-workspace','@deepseek-ai/dsh-client-ui-codex-sidebar'].includes(m);
  if(slot==='sidebar.brand.mark'||slot==='sidebar.brand.name')return ['@deepseek-ai/dsh-client-ui-brand-official','@deepseek-ai/dsh-client-ui-codex-sidebar'].includes(m);
  if(slot==='sidebar.panellist')return ['@deepseek-ai/dsh-client-ui-plugin-manager','@deepseek-ai/dsh-client-ui-schedule','dsh-whale'].includes(m);
  if(slot==='sidebar.workspaces.session.menu.item'||slot==='sidebar.workspaces.session.row.action')return m==='@deepseek-ai/dsh-client-ui-workspace';
  if(slot==='sidebar.session.row.leading'||slot==='sidebar.session.row.hover')return m==='@deepseek-ai/dsh-client-ui-schedule';
  return false;
 };
 const label=entry=>{try{return String(typeof entry.options.label==='function'?entry.options.label():entry.options.label||'').slice(0,8192)}catch{return ''}};
 function applyContexts(){if(workspaceContext)window.__whalePublicWorkspace?.apply(workspaceContext);for(const [id,context] of contexts){const owners=window.__whalePublicRows?.get(id);if(owners?.size===1)[...owners][0].apply(context);else if(!context.menu&&!context.hover)for(const owner of owners||[])owner.apply(context)}}
 function snapshot(actions){
  menus.clear();bundledName=whaleBundledRegistrant(entries);const documents=[],menuActions=[],markers=[...document.querySelectorAll('[data-whale-slot-entry]')];
  if(typeof window.__whaleSessionEntryOf!=='function')return {publicDocuments:documents,publicMenus:menuActions};
  for(const slot of slots)for(const entry of entries(slot)){
   if(entry.options.id)documents.push({slot,ownerKey:'',handle:'c'+handle(entry),id:String(entry.options.id),registrant:module(entry),label:label(entry),order:Number.isFinite(entry.options.order)?entry.options.order:0,native:native(slot,entry),catalog:true,nodes:[]});
   for(const root of markers.filter(node=>window.__whaleSessionEntryOf(node,slot)===entry&&!whaleRichSurface.hidden(node))){
    const ownerKey=root.getAttribute('data-whale-public-session')||'';
    const owner={root,globalEntry:entry,globalSlot:slot},isNative=native(slot,entry),id=handle(root);
    documents.push({slot,ownerKey,handle:id,id:String(entry.options.id||entry.options.key||''),registrant:module(entry),label:slot==='settings.close'?String(root.textContent||'').trim().slice(0,8192):label(entry),order:Number.isFinite(entry.options.order)?entry.options.order:0,native:isNative,nodes:isNative?[]:whaleNativeDocument(root,handle,actions,owner,{left:512})});
    if(slot==='sidebar.workspaces.session.menu.item'&&!isNative)for(const node of root.querySelectorAll('button,[role="menuitem"],[role="menuitemcheckbox"],[role="menuitemradio"]')){
     if(whaleRichSurface.hidden(node)||[...root.querySelectorAll('button,[role="menuitem"]')].some(parent=>parent!==node&&parent.contains(node)))continue;
     const key=handle(node),text=String(node.getAttribute('aria-label')||node.textContent||'').trim().slice(0,8192);if(!text)continue;
     const action={slot,ownerKey,handle:key,entryHandle:id,id:String(entry.options.id||''),registrant:module(entry),order:entry.options.order??0,label:text,disabled:node.disabled===true||node.getAttribute('aria-disabled')==='true'};
     menus.set(key,{action,node,owner,entry});menuActions.push(action);
    }
   }
  }
  const launcher=window.__whalePublicLauncher;
  if(launcher)for(const entry of entries('settings.launcher.action')){
   const text=label(entry);if(!text)continue;
   const action={slot:'settings.launcher.action',ownerKey:'',handle:'l'+handle(entry),id:String(entry.options.id||''),registrant:module(entry),order:entry.options.order??0,label:text,disabled:false};
   menus.set(action.handle,{action,entry,launcher});menuActions.push(action);
  }
  menuActions.sort((a,b)=>a.order-b.order);
  return {publicDocuments:documents,publicMenus:menuActions};
 }
 function invoke(command){
  const record=menus.get(command.handle);if(!record||record.action.slot!==command.slot||record.action.ownerKey!==command.ownerKey||record.action.disabled||!entries(command.slot).includes(record.entry))return false;
  if(record.launcher)return record.launcher===window.__whalePublicLauncher&&record.launcher.select(record.entry.options.id);
  if(!whaleNativeContains(record.owner,record.node)||record.node.disabled||whaleRichSurface.hidden(record.node))return false;
  record.node.click();return true;
 }
 function context(command){if(typeof command.sessionId!=='string'||command.sessionId.length>8192||typeof command.menu!=='boolean'||typeof command.hover!=='boolean')return;contexts.set(command.sessionId,{menu:command.menu,hover:command.hover});applyContexts();if(!command.menu&&!command.hover)contexts.delete(command.sessionId);schedule()}
 const change=()=>{applyContexts();schedule()};window.addEventListener('whale-public-owners',change);
 return {snapshot,invoke,context,workspace(command){if(typeof command.query!=='string'||command.query.length>8192||typeof command.archived!=='boolean')return;workspaceContext={query:command.query,archived:command.archived};applyContexts();schedule()},dispose(){for(const [id]of contexts)for(const owner of window.__whalePublicRows?.get(id)||[])owner.apply({menu:false,hover:false});contexts.clear();menus.clear();window.removeEventListener('whale-public-owners',change)}};
}
