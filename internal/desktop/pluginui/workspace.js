/* Public uiWorkspace registries remain the executable authority. Only labels,
 * matched presentation membership and action availability cross to native. */
function whaleWorkspaceBridge(ctx,schedule,send){
 let service=null,off=[],contextSession='',contextPending=false;
 const directoryIDs=new WeakMap();let directorySerial=0;
 const directoryHandle=action=>{let id=directoryIDs.get(action);if(!id){id=String(++directorySerial);directoryIDs.set(action,id)}return id};
 const rows=()=>{
  const list=ctx.sessions.list.getSnapshot();
  return (service?.sessionNavigation.getSnapshot()||[]).flatMap(entry=>{
   const session=list.byId[entry.sessionId],presentation=session?.presentation;
   if(entry.owner==='project'||presentation?.owner!==entry.owner||presentation.workingDirectory!==entry.workingDirectory)return [];
   return [{sessionId:entry.sessionId,owner:entry.owner,workingDirectory:entry.workingDirectory,title:presentation.title||session.displayTitle||entry.sessionId,archived:entry.archived===true,
    canRename:!!service.resolveSessionAction(entry.sessionId,'rename'),canArchive:!!service.resolveSessionAction(entry.sessionId,'archive'),canRestore:!!service.resolveSessionAction(entry.sessionId,'restore')}];
  });
 };
 function sync(){
  const next=ctx.get('uiWorkspace');if(next===service)return;
  off.forEach(fn=>fn());off=[];service=next;
  if(service)off=[service.directoryActions.subscribe(schedule),service.sessionNavigation.subscribe(schedule),ctx.sessions.list.subscribe(schedule),ctx.get('workspaces')?.list.subscribe(schedule)].filter(Boolean);
 }
 function snapshot(){
  sync();
  if(contextPending&&service&&contextSession&&ctx.sessions.list.getSnapshot().byId[contextSession]){contextPending=false;service.openSession(contextSession)}
  const directories=ctx.get('workspaces')?.list.getSnapshot().items||[];
  const navigation=rows();
  const paths=[...new Set([...directories.map(row=>row.path),...navigation.map(row=>row.workingDirectory)])];
  const directoryActions=(service?.directoryActions.getSnapshot()||[]).filter(action=>action.id!=='dsh-project').map(action=>{
   const rowLabels={};for(const path of paths){const label=action.rowLabel?.(path);if(typeof label==='string')rowLabels[path]=label}
   return {handle:directoryHandle(action),id:action.id,label:action.label,order:action.order||0,picker:action.picker!==false,rowLabels};
  });
  return {directoryActions,workspaceNavigation:navigation};
 }
 async function invoke(command){
  sync();let error='';
  try{
   if(!service)throw new Error('Workspace service is unavailable');
   if(command.op==='directory'){
    const action=service.directoryActions.getSnapshot().find(row=>directoryHandle(row)===command.id&&row.id!=='dsh-project');
    if(!action||typeof command.path!=='string'||!command.path)throw new Error('Directory action is no longer available');
    await action.adopt(command.path);
   }else{
    const row=rows().find(row=>row.sessionId===command.id&&row.owner===command.owner);
    if(!row)throw new Error('Session owner navigation changed');
    if(command.op==='open'){service.openSession(row.sessionId)}
    else if(['rename','archive','restore'].includes(command.op)){
     const action=service.resolveSessionAction(row.sessionId,command.op);if(!action)throw new Error('Session owner action was withdrawn');await action();
    }else throw new Error('Unsupported workspace action');
   }
  }catch(cause){error=String(cause?.message||cause).slice(0,8192)}
  schedule();send({type:'workspaceResult',request:command.request,error});
 }
 return {snapshot,invoke,context:id=>{contextSession=String(id||'');contextPending=true;schedule()},dispose:()=>off.forEach(fn=>fn())};
}
