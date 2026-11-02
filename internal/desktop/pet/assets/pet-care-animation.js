'use strict';
window.WhalePetCareAnimation = {create({characters, body}) {
  characters = (body?.elements || characters || []).filter(Boolean);
  let last = 0, kind = '', until = 0, sleeping = false, dozing = false, timer = 0;
  const clips = [], decorations = [];
  let restDecoration = null, neuralParticles = null;
  const active = () => body ? body.active() : characters.find(c => !c.hidden) || characters[0];
  function clear() {
    clearTimeout(timer);
    for (const clip of clips.splice(0)) clip.cancel();
    for (const item of decorations.splice(0)) item.remove();restDecoration=null;
  }
  function decorate(text, dx = 0, dy = -12) {
    const character = active(), r = body ? body.rect(character) : character?.getBoundingClientRect();
    const item = document.createElement('span');
    item.textContent = text;
    if (!r) return item;
    Object.assign(item.style, {position:'fixed',left:(r.left+r.width/2+dx)+'px',top:(r.top+dy)+'px',pointerEvents:'none',zIndex:'8',font:'bold 19px system-ui',color:'#e68baa',textShadow:'0 1px 2px #fff'});
    document.body.appendChild(item); decorations.push(item); return item;
  }
  function animate(frames, duration) {
    for (const character of characters) if (!body?.managed(character)) clips.push(character.animate(frames,{duration,fill:'none',easing:'ease-in-out'}));
  }
  function restStyles() {
    for (const character of characters) {
      if (body?.managed(character)) continue;
      character.style.rotate = sleeping ? '18deg' : dozing ? '7deg' : '';
      character.style.scale = sleeping ? '1 .82' : '';
    }
  }
  function rest(next, nap) {
    if (sleeping === next && dozing === nap) return;
    sleeping = next; dozing = nap; clear(); kind = '';
    restStyles();
    if (!neuralParticles && (sleeping || dozing)) restDecoration=decorate(sleeping ? 'Z z z' : 'z', 18, 0);
  }
  function play(action) {
    clear(); kind = action.Kind; until = performance.now() + 3600;
    if (kind === 'pat') {
      decorate('♥ ♥ ♥');
      animate([{rotate:'0deg'},{rotate:'-7deg',scale:'1.04 .96'},{rotate:'7deg'},{rotate:'0deg'}],1600);
    } else {
      // Walk toward a small local bowl, chew, then return to the anchor.
      decorate(kind === 'token' ? '◡ ◆' : '◡ ◇', 30, 75);
      if (action.Name) decorate('谢谢「'+action.Name.slice(0,24)+'」', -55, -16).style.fontSize='11px';
      animate([{translate:'0px 0px'},{translate:'23px 0px',offset:.25},{translate:'23px 4px',scale:'1.05 .95',offset:.4},{translate:'23px 0px',scale:'.97 1.03',offset:.55},{translate:'23px 4px',scale:'1.05 .95',offset:.7},{translate:'0px 0px',scale:'1'}],2600);
    }
    timer = setTimeout(() => {
      clear();
      if (action.Level > 0) {
        kind='level';until=performance.now()+2200;
        decorate('✦ Lv.'+action.Level+' ✦');
        animate([{translate:'0 0'},{translate:'0 -18px',rotate:'-8deg'},{translate:'0 0',rotate:'8deg'},{translate:'0 -12px'},{translate:'0 0',rotate:'0deg'}],1800);
        timer=setTimeout(()=>{clear();kind='';},2200);
      } else kind='';
    },2800);
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
    releaseBody(){restStyles();if(restDecoration){const r=body ? body.rect(active()) : active()?.getBoundingClientRect();if(r)Object.assign(restDecoration.style,{left:(r.left+r.width/2+18)+'px',top:r.top+'px'});}},
    placeRestBody(point){if(restDecoration&&point)Object.assign(restDecoration.style,{left:(point.x+18)+'px',top:(point.y-12)+'px'});},
    claimBody(){for(const clip of clips.splice(0))clip.cancel();restStyles();},
    update(task) {
      rest(task.Rest?.Sleeping === true, task.Rest?.Dozing === true);
      const action=task.Care?.Action;
      if (!action || !action.ID || action.ID===last) return;
      last=action.ID;
      if (!sleeping && Date.now()-action.ID < 10000 && ['pat','file','token'].includes(action.Kind)) play(action);
    },
    pose(pose, neuralRest = false) {
      if (sleeping || dozing) {if(neuralRest)return;pose[12]=pose[13]=sleeping?1:.55;pose[40]=.15;return;}
      if (performance.now()>until) return;
      if (kind==='pat' || kind==='level') {pose[14]=pose[15]=.8;pose[8]=pose[9]=.6;pose[26]=.25;}
      if (kind==='file' || kind==='token') {pose[26]=.25+.2*Math.sin(performance.now()/100);pose[40]=.1*Math.sin(performance.now()/170);}
    },
    close(){neuralParticles?.close();clear();for(const character of characters){character.style.rotate='';character.style.scale='';}}
  };
}};
