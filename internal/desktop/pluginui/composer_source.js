/* Installed ui-conversation inputActions are rebound only in the private native
 * Session source. The native draft remains the sole text/attachment/send owner. */
(function(){
 if(new URLSearchParams(location.search).get('desktopNativeSession')!=='1'||window.__whaleNativeComposer)return;
 const providers=new Set(),bars=new Map(),files=new Map(),reading=new Set(),requests=new Map(); let transport=null,selected=null,draft=null,serial=0,page=0,importing=false,operations=Promise.resolve(),queued=0;
 const same=(a,b)=>a?.kind==='session'&&b?.kind==='session'&&a.sessionId===b.sessionId&&a.epoch===b.epoch;
 const ready=binding=>!!(transport&&draft&&binding.sessionId===selected?.sessionId&&same(draft.owner?.subject,selected));
 const notify=()=>{for(const p of providers)p.notify()};
 function clearFiles(){for(const value of files.values())if(value.previewUrl)URL.revokeObjectURL(value.previewUrl);files.clear();reading.clear()}
 function remember(id,file,image){id=String(id);const old=files.get(id);if(old?.previewUrl)URL.revokeObjectURL(old.previewUrl);files.set(id,{kind:image?'image':'file',id,file,...image?{previewUrl:URL.createObjectURL(file)}:{}})}
 function update(next){if(!next||!same(next.owner?.subject,selected))return;draft=next;const ids=new Set(next.attachments.map(f=>String(f.id)));for(const [id,value] of files)if(!ids.has(id)){if(value.previewUrl)URL.revokeObjectURL(value.previewUrl);files.delete(id)}notify()}
 function failure(error,subject){if(same(subject,selected))window.dispatchEvent(new CustomEvent('whale-plugin-composer-error',{detail:String(error?.message||error)}))}
 async function responseOK(response){if(!response.ok)throw new Error(await response.text()||'Native attachment transfer failed');return response}
 async function importFiles(binding,provided){
  const owner=draft.owner,subject=selected,originPage=page,before=new Set(draft.attachments.map(f=>String(f.id))),body=new FormData();
  body.append('owner',JSON.stringify({page:originPage,owner}));for(const file of provided){body.append('metadata',JSON.stringify({name:file.name||'file',mediaType:file.type,size:file.size}));body.append('file',file,file.name||'file')}
  importing=true;notify();
  try{const response=await responseOK(await fetch('/__whale_plugin_ui/composer/files',{method:'POST',credentials:'same-origin',body}));const reply=await response.json();if(!same(subject,selected)||page!==originPage)return;const added=reply.draft.attachments.filter(f=>!before.has(String(f.id)));if(added.length!==provided.length)throw new Error('Native attachment transfer returned an incomplete batch');added.forEach((f,i)=>remember(f.id,provided[i],f.image));update(reply.draft)}
  catch(error){failure(error,subject)}finally{if(same(subject,selected)&&page===originPage){importing=false;notify()}}
 }
 async function readFile(file){
  const id=String(file.id);if(reading.has(id))return;reading.add(id);const owner=draft.owner,subject=selected,originPage=page;
  try{const response=await responseOK(await fetch('/__whale_plugin_ui/composer/file',{method:'POST',credentials:'same-origin',headers:{'content-type':'application/json'},body:JSON.stringify({page:originPage,owner,id:file.id})}));const blob=await response.blob();if(!same(subject,selected)||page!==originPage||!draft.attachments.some(f=>String(f.id)===id))return;remember(id,new File([blob],file.name,{type:file.mediaType??blob.type}),file.image);notify()}
  catch(error){failure(error,subject)}finally{reading.delete(id)}
 }
 function retireRequests(){for(const pending of requests.values())pending.resolve(false);requests.clear()}
 function request(binding,op,extra){
  const subject=selected;
  const run=async()=>{
   if(!same(subject,selected)||!ready(binding))return false;
   const fields=typeof extra==='function'?extra():extra;if(fields===false)return false;
   const id=String(++serial);
   return new Promise(resolve=>{requests.set(id,{subject,page,resolve});try{transport({type:'composer',composer:{request:id,op,subject,...fields}})}catch(error){requests.delete(id);failure(error,subject);resolve(false)}});
  };
  queued++;const result=operations.then(run,run).finally(()=>{queued--});operations=result.then(()=>{},()=>{});return result;
 }
 window.__whaleNativeComposer={
  connect(send,subject){if(!same(subject,selected)){retireRequests();clearFiles();draft=null;selected=subject;importing=false;notify()}transport=send},
  receive(message){if(message.page)page=message.page;if(message.draft)update(message.draft);const pending=requests.get(message.request);if(pending){requests.delete(message.request);const current=same(pending.subject,selected)&&(!pending.page||pending.page===page);pending.resolve(current&&message.ok===true);if(current&&(message.error||message.syncError))failure(message.error||message.syncError,pending.subject)}else if(message.error||message.syncError)failure(message.error||message.syncError,selected)},
  disconnect(){retireRequests();clearFiles();transport=null;draft=null;selected=null;page=0;importing=false;notify()},
  pickFiles(sessionId,image=false){const binding={sessionId};if(!ready(binding))return false;void request(binding,'edit',()=>({edit:{owner:draft.owner,...image?{pickImage:true}:{pickFiles:true}}}));return true},
  barProps(sessionId,original){
   const binding={sessionId};let bar=bars.get(sessionId);if(!bar){
    const listeners=new Set();let uploadSnapshot=null;const notify={notify(){uploadSnapshot=null;for(const listener of listeners)listener()}};providers.add(notify);
    const file=id=>ready(binding)&&draft.attachments.find(file=>String(file.id)===String(id));
    const edit=extra=>request(binding,'edit',()=>({edit:{owner:draft.owner,...extra}}));
    const uploads={getSnapshot(){if(uploadSnapshot)return uploadSnapshot;const out={};if(ready(binding))for(const file of draft.attachments)out[String(file.id)]=file.error?{status:'error',message:file.error}:file.preparing?{status:'uploading',loaded:file.loaded||0,...file.size?{total:file.size}:{}}:{status:'ready'};return uploadSnapshot=out},subscribe(fn){listeners.add(fn);return()=>listeners.delete(fn)}};
    bar={
     addFiles(provided,directories=new Set()){if(!ready(binding)||!page)return 'The native composer owner is unavailable';if(importing||draft.sending||draft.preparing)return 'The native composer is sending or preparing attachments';const incoming=[...provided||[]];if(!incoming.length)return null;if(incoming.some(file=>directories.has(file)))return 'Directory references require native desktop file selection';if(incoming.some(file=>!(file instanceof Blob)||!Number.isSafeInteger(file.size)||file.size<0))return 'The provided attachment is not a browser File';void importFiles(binding,incoming);return null},
     removeAttachment(id){const current=file(id);return !!current&&edit({removeFileId:current.id})},
     resolveDraftAttachments(ids){if(!ready(binding))return [];const wanted=new Set(ids.map(String)),out=[];for(const file of draft.attachments)if(wanted.has(String(file.id))){const cached=files.get(String(file.id));if(cached)out.push(cached);else void readFile(file)}return out},
     retryFileUpload(id){const current=file(id);return !!current&&!!current.error&&edit({retryFileId:current.id})},uploads
    };bars.set(sessionId,bar);
   }
   return {...original,addFiles:bar.addFiles,removeAttachment:bar.removeAttachment,resolveDraftAttachments:bar.resolveDraftAttachments,retryFileUpload:bar.retryFileUpload,hooks:{...original.hooks,fileUploads:bar.uploads}};
  },
  provide(binding,shell){
   if(shell.__whaleNativeInput)return shell.__whaleNativeInput;
   const listeners=new Set();let snapshot=null;
   const current=()=>{
    if(snapshot)return snapshot;
    const active=ready(binding),d=active?draft:null;
    return snapshot={draft:d?.text||'',draftRev:d?.owner?.revision||0,attachmentIds:(d?.attachments||[]).map(file=>String(file.id)),phase:!active||d.sending||d.preparing||importing?'submitting':'plain',occurrences:[],queue:shell.state.getSnapshot().queue||[]};
   };
   const state={getSnapshot:current,subscribe(listener){listeners.add(listener);return()=>listeners.delete(listener)}};
   const provider={notify(){snapshot=null;for(const listener of listeners)listener()}};providers.add(provider);
   const edit=extra=>request(binding,'edit',()=>({edit:{owner:draft.owner,...extra}}));
   const actions={
    captureInsertion(){return ready(binding)?{start:draft.selectionStart,end:draft.selectionEnd,draftRev:draft.owner.revision,owner:{...draft.owner}}:{start:0,end:0,draftRev:-1}},
    insertText(text,span){
     if(!ready(binding)||!page||queued||span?.draftRev!==draft.owner.revision||span.owner?.binding!==draft.owner.binding)return false;
     const subject=selected;
     try{const xhr=new XMLHttpRequest();xhr.open('POST','/__whale_plugin_ui/composer/insert',false);xhr.setRequestHeader('Content-Type','application/json');xhr.send(JSON.stringify({page,request:String(++serial),owner:span.owner,replace:{start:span.start,end:span.end,text:String(text)}}));if(xhr.status!==200){failure(xhr.responseText||'Native insertion unavailable',subject);return false}const reply=JSON.parse(xhr.responseText);window.__whaleNativeComposer.receive(reply);return same(subject,selected)&&reply.ok===true}catch(error){failure(error,subject);return false}
    },
    insertTextAsync(text,span){return request(binding,'edit',()=>span?.draftRev===draft.owner.revision&&span.owner?.binding===draft.owner.binding?{edit:{owner:span.owner,replace:{start:span.start,end:span.end,text:String(text)}}}:false)},
    setDraft(text){return edit({text:String(text)})},
    persistDraft(){return ready(binding)},
    removeAttachment(id){const file=ready(binding)&&draft.attachments.find(file=>String(file.id)===String(id));return !!file&&edit({removeFileId:file.id})},
    // Browser attachment registry IDs do not identify native prepared files.
    addAttachments(){return false},pruneAttachments(){return false},
    pickFiles(){return edit({pickFiles:true})},pickImage(){return edit({pickImage:true})},
    submit(){return request(binding,'submit',()=>({gesture:{owner:draft.owner,accelerated:false}}))}
   };
   return shell.__whaleNativeInput={state,actions};
  }
 };
})();
