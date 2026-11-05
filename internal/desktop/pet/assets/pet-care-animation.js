'use strict';
// State programs adapted from ChisaAlter/WhaleIsle installed 0.3.13,
// src/renderer/pet-live2d.js. Copyright (c) 2026 Chisa; MIT, see rig/LICENSE.
// One presentation owner serializes explicit care, queued level-up and idle
// flourishes. Native care/food persistence remains the only source of rewards.
window.WhalePetCareAnimation={create({characters,body}){
  characters=(body?.elements||characters||[]).filter(Boolean);
  let last=0,sleeping=false,dozing=false,restDecoration=null,neuralParticles=null,stage=null,task={},closed=false;
  let nextIdle=performance.now()+25000+Math.random()*20000,lastFrame=0,offset=0,bodyBlocked=false;
  const queue=[],clips=[],decorations=[];
  const active=()=>body?body.active():characters.find(c=>!c.hidden)||characters[0];
  const clamp=(n,a,b)=>Math.max(a,Math.min(b,n)),smooth=t=>t*t*(3-2*t);
  function decorate(text,dx=0,dy=-12){
    const character=active(),r=body?body.rect(character):character?.getBoundingClientRect(),item=document.createElement('span');item.textContent=text;
    if(!r)return item;
    Object.assign(item.style,{position:'fixed',left:(r.left+r.width/2+dx)+'px',top:(r.top+dy)+'px',pointerEvents:'none',zIndex:'8',font:'bold 19px system-ui',color:'#e9b750',textShadow:'0 1px 2px #0003'});
    document.body.append(item);decorations.push(item);return item;
  }
  function clear(){for(const clip of clips.splice(0))clip.cancel();for(const item of decorations.splice(0))item.remove();restDecoration=null;}
  function busy(value){if(window.whalePetStageBusy!==value){window.whalePetStageBusy=value;window.dispatchEvent(new Event('whale-pet-surfaces'));}}
  function restStyles(){for(const character of characters){if(body?.managed(character))continue;character.style.translate='';character.style.rotate=sleeping?'18deg':dozing?'7deg':'';character.style.scale=sleeping?'1 .82':'';}}
  function say(category,values={}){window.dispatchEvent(new CustomEvent('whale-pet-reaction',{detail:{category,values}}));}
  function reset(){clear();stage=null;offset=0;busy(false);restStyles();nextIdle=performance.now()+25000+Math.random()*20000;}
  function phase(name,now){stage.phase=name;stage.at=now;}
  function stars(count=1){for(let i=0;i<count;i++){const item=decorate('✦',(Math.random()-.5)*130,Math.random()*70);item.animate([{opacity:0,translate:'0 0',scale:'.5'},{opacity:1,offset:.2},{opacity:0,translate:'0 -50px',scale:'1.4'}],{duration:1800,easing:'ease-out'});}}
  function start(action,now){
    clear();offset=0;stage={action,phase:action.Kind==='token'?'drop':action.Kind==='file'?'celebrate':action.Kind==='pat'?'react-head':action.Kind,at:now,started:now};busy(true);
    if(action.Kind==='token'){
      const r=active()?.getBoundingClientRect(),layout=task.Layout||{},area=layout.Area||{X:0,Width:innerWidth};
      const center=(r?.x||layout.X||0)+(r?.width||layout.Width||240)/2;
      stage.target=clamp(center+(Math.random()-.5)*440,area.X+60,area.X+area.Width-60)-center;
      stage.ground=(r?.bottom||layout.Y+layout.Height)-10;
      stage.bowl=decorate('◆');stage.bowl.style.color='#a9dff7';stage.bowl.style.fontSize='25px';
      stage.bowlX=center+stage.target;
    }else if(action.Kind==='file'){say('fileEat',{file:action.Name||'文件'});stars();}
    else if(action.Kind==='pat'){say('pat');const hearts=decorate('♥ ♥ ♥');hearts.style.color='#e88ca6';}
    else if(action.Kind==='level'){say('levelUp');stars(5);decorate('Lv.'+action.Level,-20,-22);}
    else if(action.Kind==='star')stars(2);
  }
  function interrupt(){
    // Picking her up or putting her to sleep may end the current clip, but a
    // persisted level-up must still be celebrated once the body is free.
    if(stage?.action.Level>0)queue.unshift({Kind:'level',Level:stage.action.Level});
    reset();
  }
  function end(now){
    const previous=stage?.action;reset();
    if(previous?.Level>0&&previous.Kind!=='level')queue.unshift({Kind:'level',Level:previous.Level});
    else if(previous?.Kind==='token')say('feedTokenDone');
    if(queue.length)start(queue.shift(),now);
  }
  function advance(now){
    if(closed)return;
    const dt=clamp((now-lastFrame)/1000,0,.1);lastFrame=now;
    if(!stage&&queue.length&&!sleeping&&!body?.capturing())start(queue.shift(),now);
    if(!stage)return;
    const elapsed=(now-stage.at)/1000;
    if(stage.action.Kind==='token'){
      if(stage.phase==='drop'&&elapsed>=.4){phase('running',now);say('feedToken');}
      else if(stage.phase==='running'){
        offset+=(stage.target-offset)*Math.min(dt*5,1);
        if(Math.abs(stage.target-offset)<=8){offset=stage.target;phase('food-notice',now);say('feedTokenEat');}
      }else if(stage.phase==='food-notice'&&elapsed>.5)phase('eat',now);
      else if(stage.phase==='eat'&&elapsed>1.5)phase('chew',now);
      else if(stage.phase==='chew'&&elapsed>.8)phase('full',now);
      else if(stage.phase==='full'&&elapsed>.6)phase('return',now);
      else if(stage.phase==='return'){offset=stage.target*(1-smooth(clamp(elapsed/.35,0,1)));if(elapsed>=.35)end(now);}
    }else if(elapsed>(stage.action.Kind==='file'?1.4:stage.action.Kind==='pat'||stage.action.Kind==='level'?2.6:1.6))end(now);
  }
  function program(pose,now){
    const state=stage?.phase,t=now/1000,age=stage?(now-stage.at)/1000:0,fx={rot:0,dy:0,sx:1,sy:1};
    const a=stage?smooth(clamp((now-stage.started)/140,0,1)):0,set=(ch,v)=>{if(pose)pose[ch]+=(v-pose[ch])*a;};
    const happy=()=>{set(14,1);set(15,1);set(30,.7);set(31,.7);};
    switch(state){
      case 'running':happy();set(39,.05);fx.rot=.07;fx.dy=-Math.abs(Math.sin(now/90))*10;break;
      case 'food-notice':set(16,.7);set(17,.7);set(22,.35);fx.dy=-3*Math.sin(Math.PI*clamp(age/1.1,0,1));break;
      case 'eat':if(age<.8){set(16,1);set(17,1);set(22,.5);}else set(26,.05+.75*(.5+.5*Math.sin(t*7)));set(39,.04+.03*Math.sin(t*2.2));set(40,Math.sign(stage.target)*.3);set(37,Math.sign(stage.target)*.4);fx.rot=Math.sign(stage.target)*.04;break;
      case 'chew':set(26,.08+.62*(.5+.5*Math.sin(t*8)));set(39,.035*Math.sin(t*8));fx.dy=1.2*Math.sin(t*8);break;
      case 'full':happy();fx.sy=1+.02*Math.sin(t*2);break;
      case 'react-head':if(age<.45){set(16,1);set(17,1);set(22,.4);}else happy();fx.rot=.08*Math.sin(t*5);break;
      case 'celebrate':case 'level':happy();set(26,Math.abs(Math.sin(t*5.2))>.6?.8:.15);fx.dy=-14*Math.abs(Math.sin(t*5.2));fx.sy=1+.03*Math.sin(t*10.4);break;
      case 'star':set(16,1);set(17,1);set(26,.8);fx.dy=-8*Math.abs(Math.sin(t*4.4));break;
      case 'twirl':{const p=clamp(age/1.3,0,1);set(40,.6*Math.sin(2*Math.PI*p));set(42,.5*Math.sin(2*Math.PI*p));fx.rot=.14*Math.sin(2*Math.PI*p);fx.sx=1-.06*Math.sin(Math.PI*p)**2;fx.dy=-5*Math.sin(Math.PI*p);break;}
      case 'look':set(37,.5*Math.sin(t*2));set(38,.6*Math.sin(t*2));set(40,.35*Math.sin(t*2));fx.rot=.025*Math.sin(t*2);break;
      case 'wave':happy();if(t%5>1&&t%5<1.5)set(12,1);set(26,Math.sin(t*2.4)>.3?.7:.1);fx.rot=.06*Math.sin(t*2.4);fx.dy=-5*Math.abs(Math.sin(t*2.4));break;
      case 'shy':set(12,.45);set(13,.45);set(40,.25);fx.rot=.045*Math.sin(t*2);fx.dy=2*Math.sin(t*2);break;
      case 'happy-tail':happy();set(43,.25*Math.sin(t*7));fx.rot=.04*Math.sin(t*3.5);break;
      case 'sad':set(2,.7);set(3,.7);set(18,.35);set(19,.35);set(39,.2);fx.dy=3;fx.sy=.97;break;
    }
    return fx;
  }
  function present(now){
    if(bodyBlocked||sleeping||body?.capturing()||document.hidden)return;
    advance(now);if(!stage)return;
    const fx=program(null,now),rig=document.body.dataset.renderer==='rig';
    for(const character of characters){if(body?.managed(character))continue;character.style.translate=offset+'px '+(rig?0:fx.dy)+'px';character.style.rotate=(rig?0:fx.rot*180/Math.PI)+'deg';character.style.scale=rig?'1':fx.sx+' '+fx.sy;}
    if(stage.bowl){const age=(now-stage.at)/1000,y=stage.phase==='drop'?stage.ground-(1-clamp(age/.4,0,1)**2)*260:stage.ground;Object.assign(stage.bowl.style,{left:stage.bowlX+'px',top:y+'px',opacity:stage.phase==='full'||stage.phase==='return'?'0':'1'});}
  }
  return {
    useNeuralParticles(random = Math.random){
      neuralParticles?.close();
      if(restDecoration){restDecoration.remove();const index=decorations.indexOf(restDecoration);if(index>=0)decorations.splice(index,1);restDecoration=null;}
      const particles=[],rng=(a,b)=>a+random()*(b-a);let nextZzz=0,closed=false;
      const driver={
        frame(now,state){
          if(closed||document.hidden)return;
          const layout=state?.layout;
          if(state?.sleeping&&['sleep-enter','sleep'].includes(state.phase)&&layout&&['X','Y','Width','Height'].every(key=>Number.isFinite(layout[key]))&&layout.Width>0&&layout.Height>0&&(!nextZzz||now>nextZzz)){
            nextZzz=now+rng(1400,2200);
            const settled=state.phase==='sleep',node=document.createElement('span');node.textContent='Z';
            Object.assign(node.style,{position:'fixed',pointerEvents:'none',zIndex:'8',color:'#8fb8e8',fontWeight:'bold',fontFamily:'sans-serif',lineHeight:'normal'});document.body.appendChild(node);
            particles.push({node,x:layout.X+layout.Width*(settled ? .08 : .62),y:layout.Y+layout.Height*(settled ? .42 : .18),vx:rng(-.15,.15),vy:rng(-.6,-.3),born:now,life:2400,seed:rng(0,Math.PI*2)});
          }
          for(let i=particles.length-1;i>=0;i--){const p=particles[i];if(now-p.born>p.life){p.node.remove();particles.splice(i,1);continue;}p.x+=p.vx;p.y+=p.vy;p.vx+=.004;}
          for(const p of particles){const age=now-p.born,k=age/p.life;if(k>=1){p.node.style.opacity='0';continue;}const opacity=k<.15?k/.15:1-(k-.15)/.85;Object.assign(p.node.style,{left:(p.x+Math.sin(age/300+p.seed)*6)+'px',top:p.y+'px',opacity:String(Math.max(0,opacity)),fontSize:Math.round(14+k*10)+'px',transform:'translateY(-100%)'});}
        },
        close(){if(closed)return;closed=true;for(const p of particles)p.node.remove();particles.length=0;if(neuralParticles===driver){neuralParticles=null;if(sleeping||dozing)restDecoration=decorate(sleeping?'Z z z':'z',18,0);}}
      };
      neuralParticles=driver;return driver;
    },
    releaseBody(){restStyles();if(restDecoration){const r=body?body.rect(active()):active()?.getBoundingClientRect();if(r)Object.assign(restDecoration.style,{left:(r.left+r.width/2+18)+'px',top:r.top+'px'});}},
    placeRestBody(point){if(restDecoration&&point)Object.assign(restDecoration.style,{left:(point.x+18)+'px',top:(point.y-12)+'px'});},
    claimBody(){restStyles();},
    update(next){
      task=next;
      const sleep=next.Rest?.Sleeping===true,nap=next.Rest?.Dozing===true;
      if(sleep!==sleeping||nap!==dozing){sleeping=sleep;dozing=nap;if(sleeping)interrupt();restStyles();if(!neuralParticles&&(sleeping||dozing)&&!restDecoration)restDecoration=decorate(sleeping?'Z z z':'z',18,0);if(!sleeping&&!dozing&&restDecoration){restDecoration.remove();restDecoration=null;}}
      const action=next.Care?.Action;
      if(!action?.ID||action.ID===last)return;last=action.ID;
      if(!sleeping&&Date.now()-action.ID<10000&&['pat','file','token'].includes(action.Kind)&&!body?.capturing()){
        if(stage?.action.Idle)reset();queue.push(action);
      }
    },
    frame(now,input,status){
      task=status;
      bodyBlocked=!!(input.sleeping||input.pickup||body?.capturing()||document.hidden);
      if(bodyBlocked){if(stage)interrupt();nextIdle=now+25000+Math.random()*20000;return null;}
      const occupied=input.state||status.QuickChatOpen||status.State==='running'||!document.getElementById('pet-status-card').hidden||['Care','Chat','Look','Growth'].some(k=>status[k]?.Busy)||input.powerSaving;
      if(occupied){nextIdle=now+25000+Math.random()*20000;if(stage?.action.Idle)reset();}
      else if(!stage&&!queue.length&&now>nextIdle){const pool=['star','twirl','look','wave','happy-tail'];if(status.Care?.AffectionLevel>=3)pool.push('shy');if(status.Care?.Mood<35)pool.push('sad');start({Kind:pool[Math.floor(Math.random()*pool.length)],Idle:true},now);}
      present(now);if(!stage)return null;
      return {state:stage.phase==='drop'||stage.phase==='return'?'':stage.phase==='level'?'celebrate':stage.phase,facing:stage.action.Kind==='token'?(Math.sign(stage.target)||1):input.facing};
    },
    present,
    pose(pose,neuralRest=false){if(sleeping||dozing){if(neuralRest)return;pose[12]=pose[13]=sleeping?1:.55;pose[40]=.15;return;}if(stage)program(pose,performance.now());},
    close(){closed=true;queue.length=0;neuralParticles?.close();reset();for(const character of characters){character.style.translate='';character.style.rotate='';character.style.scale='';}}
  };
}};
