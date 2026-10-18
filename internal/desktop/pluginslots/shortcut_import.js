/* Explicit legacy import. Reading and changing preferences starts only after
 * the user opens the review; the legacy localStorage value is never deleted. */
function whaleRegisterShortcutImport(ctx, React) {
 if(ctx.shortcuts?.runtime!=='desktop')return;
 const h=React.createElement;
 const request=async body=>{
  const response=await fetch('/__whale_shortcut_import',{method:'POST',credentials:'same-origin',cache:'no-store',redirect:'error',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  if(!response.ok)throw Error('无法读取统一快捷键服务 / Shared shortcut service unavailable');
  return response.json();
 };
 function View(){
  const [review,setReview]=React.useState(null),[replace,setReplace]=React.useState({}),[busy,setBusy]=React.useState(false),[message,setMessage]=React.useState('');
  const alive=React.useRef(true);React.useEffect(()=>{alive.current=true;return ()=>{alive.current=false}},[]);
  async function inspect(){
   setBusy(true);setMessage('');
   try{
    const raw=localStorage.getItem('dsh.keybindings.v1');
    if(raw===null){setMessage('没有旧网页快捷键 / No legacy web shortcuts');setReview(null);return}
    const document=JSON.parse(raw);
    const plan=await request({operation:'preview',document});
    if(!alive.current)return;
    if(plan.status!=='ready')throw Error('旧配置无法导入 / Cannot import legacy preferences: '+plan.status);
    setReview({document,plan});setReplace({});
   }catch(error){if(alive.current)setMessage(String(error.message||error))}finally{if(alive.current)setBusy(false)}
  }
  async function commit(){
   if(!review)return;setBusy(true);setMessage('');
   try{
    const result=await request({operation:'import',document:review.document,revision:review.plan.snapshot.revision,replaceIds:Object.keys(replace).filter(id=>replace[id])});
    if(!alive.current)return;
    if(result.status==='saved'){setMessage('已写入统一快捷键；旧网页配置仍保留 / Imported into shared shortcuts; original web data retained');setReview(null)}
    else setMessage((result.status==='stale'?'配置已变化，请重新预览 / Preferences changed; review again':result.status==='conflict'?'组合键冲突，未写入 / Key conflict; nothing written':'导入未保存 / Import not saved')+(result.conflicts?.length?': '+result.conflicts.join(', '):''));
   }catch(error){if(alive.current)setMessage(String(error.message||error))}finally{if(alive.current)setBusy(false)}
  }
  const format=b=>b===null?'未绑定 / Unbound':b===undefined?'继承默认 / Default':[...(b.modifiers||[]),b.code,...(b.secondCode?[b.secondCode]:[])].join(' + ');
  return h('section',{'aria-label':'Import legacy web shortcuts',style:{padding:'12px 0'}},
   h('strong',null,'导入旧网页快捷键 / Import legacy web shortcuts'),
   h('p',null,'仅导入本平台显式改键。已有绑定默认保留；未知插件条目保留但不会自动执行。原网页配置不会被删除。'),
   h('button',{type:'button',disabled:busy,onClick:inspect},review?'重新预览 / Review again':'预览旧配置 / Review legacy preferences'),
   message?h('p',{role:'status'},message):null,
   review?h('div',null,
    h('p',null,'目标 / Target: '+review.plan.targetProfile),
    review.plan.entries.map(entry=>h('div',{key:entry.id,style:{padding:'6px 0'}},
     h('code',null,entry.id),h('span',null,' · '+format(entry.incoming)),
     entry.conflict?h('label',null,' · 现有 / Existing: '+format(entry.existing)+' ',h('select',{'aria-label':'Import choice '+entry.id,disabled:busy,value:replace[entry.id]?'import':'keep',onChange:event=>setReplace(old=>({...old,[entry.id]:event.target.value==='import'}))},h('option',{value:'keep'},'保留现有 / Keep existing'),h('option',{value:'import'},'导入旧值 / Import legacy'))):h('span',null,entry.hasExisting?' · 已一致 / Same':' · 新增 / Add'))),
    h('button',{type:'button',disabled:busy||!review.plan.entries.length,onClick:commit},'确认导入所选项 / Import reviewed choices'),
    h('button',{type:'button',disabled:busy,onClick:()=>{setReview(null);setReplace({})}},'取消 / Cancel')):null);
 }
 ctx.effect(()=>ctx.slots.inject('settings.general.item',()=>ctx.slots.register({name:'settings.general.item',id:'whaleisle-shortcut-import',order:17},View)),'mygo: legacy shortcut import');
}
