'use strict';
(function () {
  const card=document.getElementById('pet-status-card'),toggle=document.getElementById('pet-status-toggle'),close=document.getElementById('pet-status-close');
  const characters=window.WhalePetBody?.current.targets||[document.getElementById('sprite'),document.getElementById('fallback')];
  let zh=true;
  const copy=(a,b)=>zh?a:b, byId=id=>document.getElementById(id), set=(id,text)=>{byId(id).textContent=text;};
  const count=n=>new Intl.NumberFormat(zh?'zh-CN':'en',{notation:'compact',maximumFractionDigits:1}).format(Number(n)||0);
  function show(open){
    card.hidden=!open;toggle.setAttribute('aria-expanded',String(open));
    window.dispatchEvent(new Event('whale-pet-surfaces'));
    if(open&&document.activeElement===toggle)close.focus();
    else if(!open&&card.contains(document.activeElement))toggle.focus();
  }
  toggle.addEventListener('click',()=>show(card.hidden));close.addEventListener('click',()=>show(false));
  for(const target of characters){
    if(!target)continue;
    target.addEventListener('contextmenu',event=>{event.preventDefault();show(card.hidden);});
    let timer=0;
    target.addEventListener('pointerdown',()=>{clearTimeout(timer);timer=setTimeout(()=>show(true),600);});
    for(const event of ['pointerup','pointerleave','pointercancel'])target.addEventListener(event,()=>clearTimeout(timer));
  }
  document.addEventListener('keydown',event=>{if(event.key==='Escape'&&!card.hidden){event.preventDefault();show(false);}});
  card.addEventListener('click',event=>{if(event.target.closest('a[href]'))show(false);});
  const labels={chat:['聊聊','Chat'],look:['看看','Look'],pat:['摸摸','Pat'],settings:['设置','Settings'],hide:['隐藏','Hide'],more:['更多','More'],file:['喂文件','Feed a file'],refresh:['刷新粮仓','Refresh food'],mainChat:['打开主会话','Open main chat']};
  const stats=[['Satiety','饱食','Satiety'],['Mood','心情','Mood'],['Affection','亲密','Bond']].map(([id,a,b])=>{
    const row=document.createElement('div');row.className='pet-stat';
    const label=document.createElement('span'),track=document.createElement('div'),fill=document.createElement('div'),value=document.createElement('span');
    track.className='pet-track';fill.className='pet-fill';value.className='pet-stat-value';track.append(fill);row.append(label,track,value);byId('pet-stats').append(row);
    return {id,a,b,label,fill,value};
  });
  function update(task){
    zh=!task.Presentation?.Locale||task.Presentation.Locale.startsWith('zh');
    set('pet-status-toggle',copy('状态','Status'));card.setAttribute('aria-label',copy('宠物状态','Pet status'));close.setAttribute('aria-label',copy('关闭','Close'));
    for(const node of card.querySelectorAll('[data-label]')){const label=labels[node.dataset.label];node.textContent=copy(...label);}
    const g=task.Growth||{},care=task.Care||{},levels=g.Levels||[],level=Math.max(1,g.Level||1),tier=levels[level-1],base=tier?.At||0;
    card.style.setProperty('--pet-tier',tier?.Color||'var(--pet-accent)');
    set('pet-name',copy('鲸鱼娘','Whale'));set('pet-level',`Lv.${level} ${g.LevelName||''}`);
    const bond=Math.max(1,Math.min(5,care.AffectionLevel||1)),thresholds=[0,40,120,260,520];
    set('pet-hearts','♥'.repeat(bond)+'♡'.repeat(5-bond));
    set('pet-relation',copy(['陌生','相识','亲近','信赖','形影不离'][bond-1],['Acquainted','Familiar','Close','Attached','Beloved'][bond-1]));
    byId('pet-work-dot').classList.toggle('idle',task.State!=='running');byId('pet-work-dot').title=task.State==='running'?copy('工作中','Working'):copy('待机','Idle');
    set('pet-tier-name',copy('成长值','Growth'));set('pet-growth-points',count(g.Points)+(g.NextAt?' / '+count(g.NextAt):''));
    byId('pet-growth-fill').style.width=(g.NextAt?Math.max(0,Math.min(1,(g.Points-base)/(g.NextAt-base)))*100:100)+'%';
    byId('pet-growth-rungs').replaceChildren(...levels.map((t,i)=>{const dot=document.createElement('span');dot.className='pet-rung';dot.style.left=(i/Math.max(1,levels.length-1)*100)+'%';dot.style.opacity=g.Points>=t.At?'1':'.35';dot.title=`Lv.${i+1} ${t.Name} · ${count(t.At)}`;return dot;}));
    const next=levels[level]?.Name||'';
    set('pet-growth-next',g.NextAt?copy(`→ 再长 ${count(g.NextAt-g.Points)} 点升「${next}」`,`→ ${count(g.NextAt-g.Points)} to ${next}`):copy(`${g.LevelName||''} · 已加满`,`${g.LevelName||''} · Max level`));
    set('pet-today',copy('⚡ 今日 ','⚡ Today ')+count(g.TodayUsed));set('pet-fed',copy('🍚 已投喂 ','🍚 Fed ')+count(g.TokensFed));
    set('feed-tokens',g.NextFeed>0?copy('投喂 +','Feed +')+count(g.NextFeed):copy('无算力可喂','No food yet'));
    for(const s of stats){
      const n=Number(care[s.id])||0;s.label.textContent=copy(s.a,s.b);
      const fraction=s.id==='Affection'?(bond===5?1:(n-thresholds[bond-1])/(thresholds[bond]-thresholds[bond-1])):n/100;
      s.fill.style.width=(Math.max(0,Math.min(1,fraction))*100)+'%';
      s.fill.style.background=s.id==='Affection'?'var(--pet-accent)':n<(s.id==='Mood'?30:20)?'var(--pet-danger)':n<(s.id==='Mood'?80:75)?'#e0ac51':'var(--pet-accent)';
      s.value.textContent=s.id==='Affection'?`Lv${bond}`:s.id==='Mood'?copy(n<30?'闹别扭中':n<55?'蔫蔫的':n<80?'心情不错':'超开心',n<30?'Grumpy':n<55?'Low':n<80?'Good':'Happy'):copy(n<20?'饿扁了':n<45?'有点饿':n<75?'不饿':'肚肚圆滚滚',n<20?'Starving':n<45?'Hungry':n<75?'Okay':'Full');
      s.value.title=String(Math.round(n));
    }
    set('rest-pet',task.Rest?.Sleeping?copy('☀ 叫醒','☀ Wake'):copy('☾ 睡觉','☾ Sleep'));
    // Routine explanations belong on the action, not permanent rows in a game HUD.
    byId('feed-file').title=care.Message||'';byId('feed-tokens').title=g.Message||'';
    byId('care-message').hidden=!care.Busy&&!(care.Action?.ID&&Date.now()-care.Action.ID<8000)&&!care.Message?.match(/失败|错误|取消|failed|error|cancel/i);
    byId('pet-chat-status').hidden=!task.Chat?.Busy&&!task.Chat?.Message?.match(/失败|错误|取消|failed|error|cancel/i);
    byId('growth-message').hidden=!g.Busy&&!g.Message?.match(/失败|错误|取消|failed|error|cancel/i);
    window.dispatchEvent(new Event('whale-pet-surfaces'));
  }
  window.WhalePetStatus={show,update};
})();
