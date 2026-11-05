'use strict';
// All text is explicitly authored outbox text or fixed native status text.
// No innerHTML, script URLs, external images, sound, model calls or screenshots.
(() => {
  const panel=document.createElement('section');panel.id='pet-notifications';panel.className='pet-surface';
  panel.setAttribute('aria-live','polite');panel.setAttribute('aria-label','全部会话动态 / All conversation events');
  Object.assign(panel.style,{position:'fixed',zIndex:'11',maxWidth:'240px',maxHeight:'260px',overflowY:'auto',padding:'9px',textAlign:'center','--app-region':'no-drag'});
  panel.hidden=true;
  const body=document.createElement('div'),open=document.createElement('a'),close=document.createElement('a');
  open.textContent='打开会话 / Open chat';close.textContent='知道了 / Dismiss';
  const picture=document.createElement('img');picture.alt='鲸鱼娘贴纸 / Whale sticker';picture.hidden=true;Object.assign(picture.style,{width:'128px',height:'128px',borderRadius:'4px',objectFit:'contain',display:'block',margin:'auto','--app-region':'no-drag'});picture.addEventListener('error',()=>{picture.hidden=true;});panel.append(body,picture,open,close);document.body.append(panel);
  let stopped=false,timer=0,current='',owner='',shownAt=0,visibleMs=0,lastTick=0,acknowledging=false;
  const encode=value=>encodeURIComponent(value).replace(/%20/g,'+').replace(/[!'()*]/g,c=>'%'+c.charCodeAt(0).toString(16).toUpperCase());
  const action=(kind,id)=>kind+'?owner='+encode(owner)+'&id='+encode(id);
  const visiblyShown=()=>{
    if(document.hidden||panel.hidden||getComputedStyle(panel).visibility==='hidden')return false;
    const r=panel.getBoundingClientRect();
    return r.width>0&&r.height>0&&Math.min(r.bottom,innerHeight)>Math.max(r.top,0)&&Math.min(r.right,innerWidth)>Math.max(r.left,0);
  };
  function clear(){current='';shownAt=0;visibleMs=0;lastTick=0;acknowledging=false;panel.hidden=true;body.textContent='';picture.hidden=true;picture.removeAttribute('src');open.removeAttribute('href');close.removeAttribute('href');}
  function ack(){if(!current||!visiblyShown()||acknowledging)return;acknowledging=true;location.href=action('notification-ack',current);}
  close.addEventListener('click',e=>{e.preventDefault();ack();});
  document.addEventListener('visibilitychange',()=>{lastTick=0;});
  async function tick(){
    try{
      const response=await fetch('notifications.json',{cache:'no-store'});if(!response.ok)throw Error();
      const snap=await response.json();if(stopped)return;
      if(owner!==snap.owner){clear();owner=snap.owner||'';}
      const events=Array.isArray(snap.events)?snap.events:[];
      if(window.whalePetLookPinned){panel.hidden=true;lastTick=0;if(!stopped)timer=setTimeout(tick,1000);return;}
      let event=events.find(e=>e.id===current);
      const important=events.find(e=>e.pinned||e.type==='error');
      if(event&&!event.pinned&&important&&important.id!==event.id){clear();event=important;}
      if(!event){clear();event=events.find(e=>e.type==='error'||e.pinned)||events[0];}
      if(event&&owner){
        panel.hidden=false;
        if(current!==event.id){current=event.id;body.textContent=event.text;shownAt=performance.now();lastTick=shownAt;panel.hidden=false;if(typeof event.image==='string'&&/^notification-image\/[0-9]+\/[a-f0-9]{32}$/.test(event.image)){picture.src=event.image;picture.hidden=false;}window.dispatchEvent(new Event('whale-pet-surfaces'));}
        open.hidden=!event.sessionID;if(event.sessionID)open.href=action('notification-open',event.id);
        close.href=action('notification-ack',event.id);
        const now=performance.now();if(visiblyShown()&&lastTick)visibleMs+=Math.min(now-lastTick,2000);lastTick=now;
        // Only acknowledge after actual visible display; pinned notices wait
        // for an explicit dismissal. Repeated GET never advances the cursor.
        if(!event.pinned&&visibleMs>=9000)ack();
        // A save failure keeps this same receipt visible and allows retry.
        if(acknowledging&&now-shownAt>12000)acknowledging=false;
      }
    }catch{/* Offline/failed fetch is not an acknowledgement. */}
    if(!stopped)timer=setTimeout(tick,1000);
  }
  window.addEventListener('pagehide',()=>{stopped=true;clearTimeout(timer);clear();},{once:true});
  tick();
})();
