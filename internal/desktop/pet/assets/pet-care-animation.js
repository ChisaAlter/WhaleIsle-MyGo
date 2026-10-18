'use strict';
window.WhalePetCareAnimation = {create({characters}) {
  let last = 0, kind = '', until = 0, sleeping = false, dozing = false, timer = 0;
  const clips = [], decorations = [];
  const active = () => characters.find(c => !c.hidden) || characters[0];
  function clear() {
    clearTimeout(timer);
    for (const clip of clips.splice(0)) clip.cancel();
    for (const item of decorations.splice(0)) item.remove();
  }
  function decorate(text, dx = 0, dy = -12) {
    const r = active().getBoundingClientRect(), item = document.createElement('span');
    item.textContent = text;
    Object.assign(item.style, {position:'fixed',left:(r.left+r.width/2+dx)+'px',top:(r.top+dy)+'px',pointerEvents:'none',zIndex:'8',font:'bold 19px system-ui',color:'#e68baa',textShadow:'0 1px 2px #fff'});
    document.body.appendChild(item); decorations.push(item); return item;
  }
  function animate(frames, duration) {
    for (const character of characters) clips.push(character.animate(frames,{duration,fill:'none',easing:'ease-in-out'}));
  }
  function rest(next, nap) {
    if (sleeping === next && dozing === nap) return;
    sleeping = next; dozing = nap; clear(); kind = '';
    for (const character of characters) {
      character.style.rotate = sleeping ? '18deg' : dozing ? '7deg' : '';
      character.style.scale = sleeping ? '1 .82' : '';
    }
    if (sleeping || dozing) decorate(sleeping ? 'Z z z' : 'z', 18, 0);
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
    update(task) {
      rest(task.Rest?.Sleeping === true, task.Rest?.Dozing === true);
      const action=task.Care?.Action;
      if (!action || !action.ID || action.ID===last) return;
      last=action.ID;
      if (!sleeping && Date.now()-action.ID < 10000 && ['pat','file','token'].includes(action.Kind)) play(action);
    },
    pose(pose) {
      if (sleeping || dozing) {pose[12]=pose[13]=sleeping?1:.55;pose[40]=.15;return;}
      if (performance.now()>until) return;
      if (kind==='pat' || kind==='level') {pose[14]=pose[15]=.8;pose[8]=pose[9]=.6;pose[26]=.25;}
      if (kind==='file' || kind==='token') {pose[26]=.25+.2*Math.sin(performance.now()/100);pose[40]=.1*Math.sin(performance.now()/170);}
    },
    close(){clear();for(const character of characters){character.style.rotate='';character.style.scale='';}}
  };
}};
