'use strict';
// Only neural sleep owns this CSS transform. The backing canvas and fixed THA4
// crop retain their pixels; the browser transforms the whole composited layer.
(function (root) {
  const smooth = value => value * value * (3 - 2 * value);
  const clamp = value => Math.max(0, Math.min(1, value));
  function measure(pixels) {
    let x = 512, y = 512, right = -1, bottom = -1;
    for (let py = 0; py < 512; py += 2) for (let px = 0; px < 512; px += 2) {
      if (pixels[(py * 512 + px) * 4 + 3] > 16) {
        x = Math.min(x, px); y = Math.min(y, py); right = Math.max(right, px); bottom = Math.max(bottom, py);
      }
    }
    return right < 0 ? null : {x, y, right, bottom};
  }
  function geometry(box, layout, facing, angle, dy, sy = 1) {
    const w = layout.Width, h = layout.Height, cos = Math.cos(angle), sin = Math.sin(angle);
    const map = (x, y) => {
      const px = ((x - 60) * w / 390 - w / 2) * facing, py = ((y - 20) * h / 492 - h) * sy;
      return {x:layout.X + w / 2 + px * cos - py * sin, y:layout.Y + h - 2 + dy + px * sin + py * cos};
    };
    const points = [[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>map(x,y));
    const x = Math.min(...points.map(p=>p.x)), y = Math.min(...points.map(p=>p.y));
    const right = Math.max(...points.map(p=>p.x)), bottom = Math.max(...points.map(p=>p.y));
    // The visible canvas already contains task-29's facing reflection. CSS only
    // rotates that stage; hit coordinates apply facing exactly once above.
    return {head:map((box.x+box.right)/2,box.y),matrix:[cos,sin,-sin*sy,cos*sy,w/2-cos*w/2+sin*h*sy,h-2+dy-sin*w/2-cos*h*sy],rect:{x,y,left:x,top:y,right,bottom,width:right-x,height:bottom-y}};
  }
  // Pickup shares the same lease, but has an independent committed frame pair.
  function pickupGeometry(box, layout, frame) {
    const w=layout.Width,h=layout.Height,cos=Math.cos(frame.angle),sin=Math.sin(frame.angle),sx=frame.sx,sy=frame.sy;
    const facing=frame.hanging?1:frame.facing,anchorY=frame.hanging?h*.085:h;
    const originX=frame.hanging?frame.cursorX:layout.X+w/2,originY=(frame.hanging?frame.cursorY+12:layout.Y+h-2)+frame.dy;
    const map=(x,y)=>{const px=((x-60)*w/390-w/2)*facing*sx,py=((y-20)*h/492-anchorY)*sy;return {x:originX+px*cos-py*sin,y:originY+px*sin+py*cos};};
    const points=[[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>map(x,y));
    const x=Math.min(...points.map(p=>p.x)),y=Math.min(...points.map(p=>p.y)),right=Math.max(...points.map(p=>p.x)),bottom=Math.max(...points.map(p=>p.y));
    return {matrix:[cos*sx,sin*sx,-sin*sy,cos*sy,originX-layout.X-cos*sx*w/2+sin*sy*anchorY,originY-layout.Y-sin*sx*w/2-cos*sy*anchorY],head:map((box.x+box.right)/2,box.y),rect:{x,y,left:x,top:y,right,bottom,width:right-x,height:bottom-y}};
  }
  function create({canvas, body, getLayout, onChange = () => {}, onRelease = () => {}, sleeping = false, normalFeet = false}) {
    const target = body?.target;
    if (!target) return null;
    const handle = body.acquire(canvas);
    const placeNormalFeet = () => {if(normalFeet)canvas.style.translate=closed||retired?'':'0px -2px';};
    const placeActiveFeet = () => {if(normalFeet)canvas.style.translate='';};
    let closed = false, retired = false, requested = sleeping, started = performance.now(), revision = 0, frames = 0, box = null, active = false;
    handle.reserve(sleeping);
    let alpha = 0, dozing = false, waking = false, angle = 0, offset = 0, scaleY = 1, last = null;
    // Pose metadata reads the body's envelope; it never advances a second clock.
    let posePhase = sleeping ? 'sleep-enter' : null, tickTime = null;
    function restFrame() {
      if(closed || retired)return null;
      const now=tickTime ?? performance.now(), elapsed=Math.max(0,(now-started)/1000);
      const phase=requested ? (elapsed<1.2?'sleep-enter':'sleep') : waking ? 'wake' : dozing ? 'doze' : alpha>=.005 ? posePhase : null;
      return Object.freeze({phase,alpha,a:smooth(alpha),time:now/1000,started:started/1000,elapsed,revision,sleeping:requested});
    }
    function request(value) {
      if (closed || retired || value === requested) return;
      waking = !value && (requested || active);requested = value; started = performance.now(); revision++;
      posePhase=value?'sleep-enter':waking?'wake':posePhase;tickTime=null;
      handle.reserve(requested || active || waking);
    }
    function tick(now) {
      if (closed || retired) return;
      tickTime=now;
      alpha += ((requested || dozing || waking ? 1 : 0)-alpha)*.35;
      if(!requested&&!dozing&&!waking&&alpha<.005)alpha=0;
      // The source clears a timed action's target after this RAF's envelope.
      if(waking && now>started+1100)waking=false;
      const a=smooth(alpha);
      if(requested) {const p=smooth(clamp((now-started)/1200));angle=-p*a;offset=14*p*a;scaleY=1;}
      else if(waking) {const p=clamp((now-started)/1100), wave=Math.sin(Math.PI*p);angle=0;offset=-9*wave*a;scaleY=1+.04*wave*a;}
      else {angle=0;offset=0;scaleY=1;}
      if(requested)posePhase=now-started<1200?'sleep-enter':'sleep';
      else if(waking)posePhase='wake';
      else if(dozing)posePhase='doze';
      else if(alpha<.005)posePhase=null;
    }
    function snapshot(input, frame = null) {
      const state=frame || {revision,sleeping:requested,time:performance.now(),angle,dy:offset,sy:scaleY,waking,rest:restFrame()};
      return Object.freeze({...state,held:input.state==='pick-up',facing:input.facing===-1?-1:1,capture:body.captureVersion});
    }
    let pickupLast=null,pickupFrames=0,pickupBox=null,pickupReserved=false,offPickupRelease=null;
    const validLayout=value=>value&&['X','Y','Width','Height'].every(key=>Number.isFinite(value[key]))&&value.Width>0&&value.Height>0;
    const sameLayout=(a,b)=>validLayout(a)&&validLayout(b)&&['X','Y','Width','Height'].every(key=>a[key]===b[key]);
    function prepareNeuralBody(frame) {
      if(closed||retired||!handle.current())return;
      if(frame?.phase&&!requested&&!waking&&!active){handle.reserve(true);pickupReserved=true;}
    }
    function bodySnapshot(frame,input) {
      if(closed||retired)return null;
      const layout=getLayout();
      return Object.freeze({frame,revision,sleeping:requested,waking,capture:body.captureVersion,epoch:input.epoch,mode:input.mode,layout:layout?Object.freeze({...layout}):null});
    }
    function clearPickup() {
      if((!pickupLast&&!pickupReserved)||!handle.current())return;
      handle.clear();pickupLast=null;pickupReserved=false;
      canvas.style.pointerEvents='';canvas.style.transform='';canvas.style.transformOrigin='';placeNormalFeet();
      if(body.capturing(target)){target.style.pointerEvents='none';target.style.width='0px';target.style.height='0px';if(!offPickupRelease)offPickupRelease=body.onRelease(()=>{if(!body.capturing(target)&&!pickupLast&&!active){if(handle.current())target.hidden=true;offPickupRelease?.();offPickupRelease=null;}});}
      else {target.hidden=true;target.style.pointerEvents='none';}
      onRelease();onChange();
    }
    function commitNeuralBody(pixels,token,input,paint) {
      if(closed||retired||!handle.current()||!token||token.revision!==revision||requested||waking||active||token.sleeping||token.waking)return false;
      if(token.capture!==body.captureVersion||token.epoch!==input.epoch||token.mode!==input.mode||(token.frame?.epoch!==undefined&&token.frame.epoch!==token.epoch)||!sameLayout(token.layout,getLayout()))return false;
      const frame=token.frame;
      if(!frame?.phase&&!pickupLast&&!pickupReserved)return false;
      if(body.capturing()&&!body.capturing(canvas)&&!body.capturing(target))return false;
      let maximum=0;for(let i=3;i<pixels.length;i+=4){maximum=Math.max(maximum,pixels[i]);if(maximum>=4)break;}if(maximum<4)return false;
      const nextCount=pickupFrames+1,nextBox=!pickupBox||nextCount%240===0?measure(pixels)||pickupBox:pickupBox;
      if(frame?.phase&&(!nextBox||![frame.angle,frame.sx,frame.sy].every(Number.isFinite)||(frame.hanging&&![frame.cursorX,frame.cursorY].every(Number.isFinite))))return false;
      const next=frame?.phase?pickupGeometry(nextBox,token.layout,frame):null;
      paint();if(closed||retired||!handle.current()||token.revision!==revision)return false;
      pickupFrames=nextCount;pickupBox=nextBox;
      if(!next){clearPickup();return true;}
      if(!handle.publish(next.rect))return false;
      placeActiveFeet();canvas.style.rotate='';canvas.style.scale='';canvas.style.transformOrigin='0 0';canvas.style.transform='matrix('+next.matrix.join(',')+')';canvas.style.pointerEvents='none';
      Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px',pointerEvents:'auto'});target.hidden=false;
      pickupReserved=true;pickupLast={...next,frame,layout:token.layout,box:{...nextBox},frames:pickupFrames};onChange({...next,kind:'pickup'});return true;
    }
    function commitRest(pixels,token,input,paint) {
      if((pickupLast||pickupReserved)&&body.capturing())return false;
      return commit(pixels,token,input,()=>{paint();if((pickupLast||pickupReserved)&&!token.sleeping&&!token.waking)clearPickup();else {pickupLast=null;pickupReserved=false;}});
    }
    // A display frame reuses accepted model pixels and alpha bounds. Its body
    // clock is current; content.modelPose remains the sampled inference input.
    function presentCommitted(content,token,input,paint) {
      if(closed||retired||!handle.current()||!content?.token||!token)return false;
      const base=content.token,layout=getLayout();
      if(base.revision!==revision||token.revision!==revision||base.capture!==body.captureVersion||token.capture!==body.captureVersion)return false;
      let next=null;
      if(content.kind==='pickup'){
        const frame=token.frame;
        if(requested||waking||active||!pickupLast||!frame?.phase||base.epoch!==token.epoch||base.mode!==token.mode||token.epoch!==input.epoch||token.mode!==input.mode||(frame.epoch!==undefined&&frame.epoch!==token.epoch)||!sameLayout(token.layout,layout))return false;
        if(body.capturing()&&!body.capturing(canvas)&&!body.capturing(target))return false;
        if(![frame.angle,frame.sx,frame.sy].every(Number.isFinite)||(frame.hanging&&![frame.cursorX,frame.cursorY].every(Number.isFinite)))return false;
        next=pickupGeometry(pickupLast.box,layout,frame);
      } else {
        if(base.sleeping!==requested||token.sleeping!==requested||base.waking!==token.waking||token.held!==(input.state==='pick-up')||pickupLast||pickupReserved)return false;
        if((requested||active||token.waking)&&(body.capturing()||token.held))return false;
        if(active){if(!last||!validLayout(layout))return false;next={...geometry(last.box,layout,token.facing,token.angle,token.dy,token.sy),angle:token.angle,dy:token.dy,sy:token.sy,waking:token.waking,elapsed:Math.max(0,(token.time-started)/1000),facing:token.facing};}
        else if(requested||token.waking)return false;
      }
      paint();if(closed||retired||!handle.current()||base.revision!==revision)return false;
      if(!next)placeNormalFeet();
      if(next){
        if(!handle.publish(next.rect))return false;
        placeActiveFeet();canvas.style.rotate='';canvas.style.scale='';canvas.style.transformOrigin='0 0';canvas.style.transform='matrix('+next.matrix.join(',')+')';canvas.style.pointerEvents='none';
        Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px',pointerEvents:'auto'});target.hidden=false;
        if(content.kind==='pickup')pickupLast={...pickupLast,...next,frame:token.frame,layout:token.layout};else last={...last,...next};
        onChange(content.kind==='pickup'?{...next,kind:'pickup'}:next);
      }
      return true;
    }
    function reprojectPickup() {
      if(closed||retired||!pickupLast||!handle.current())return;
      const layout=getLayout();if(!validLayout(layout))return;
      const previous=pickupLast.layout,frame={...pickupLast.frame};
      frame.cursorX+=layout.X-previous.X;frame.cursorY+=layout.Y-previous.Y;
      const next=pickupGeometry(pickupLast.box,layout,frame);if(!handle.publish(next.rect))return;
      canvas.style.transform='matrix('+next.matrix.join(',')+')';Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px'});
      pickupLast={...pickupLast,...next,frame,layout:{...layout}};onChange({...next,kind:'pickup'});
    }
    function reset() {
      if (!handle.current()) return;
      const wasManaged=body.managed(canvas);handle.clear();
      if (!active) {if(wasManaged)onRelease();return;}
      active = false; last = null;
      target.hidden = true; target.style.pointerEvents = 'none';
      canvas.style.pointerEvents = ''; canvas.style.transform = ''; canvas.style.transformOrigin = '';placeNormalFeet();
      onRelease(); onChange();
    }
    function commit(pixels, token, input, paint) {
      if (closed || retired || !handle.current() || token.revision !== revision || token.sleeping !== requested || token.held !== (input.state==='pick-up')) return false;
      if ((requested || active || token.waking) && (token.capture !== body.captureVersion || body.capturing() || token.held)) return false;
      let alpha = 0;
      for (let i = 3; i < pixels.length; i += 4) {if(pixels[i]>alpha)alpha=pixels[i];if(alpha>=4)break;}
      if (alpha < 4) return false; // Preserve pixels, matrix and hit as one pair.
      const nextFrames = frames + 1;
      const nextBox = !box || nextFrames % 240 === 0 ? measure(pixels) || box : box;
      const layout = getLayout();
      if ((requested || waking) && (!nextBox || !layout || !['X','Y','Width','Height'].every(key=>Number.isFinite(layout[key])) || !(layout.Width>0) || !(layout.Height>0))) return false;
      let next = null;
      if (requested || token.waking) {
        const elapsed = Math.max(0,(token.time-started)/1000);
        next = {...geometry(nextBox,layout,token.facing,token.angle,token.dy,token.sy),angle:token.angle,dy:token.dy,sy:token.sy,waking:token.waking,elapsed,facing:token.facing};
      }
      paint(); // A drawing error is handled by the renderer's existing fallback.
      if(!handle.current() || closed || token.revision!==revision)return false;
      frames = nextFrames; box = nextBox;
      if (next) {
        if(!handle.publish(next.rect))return false; active = true;
        placeActiveFeet();canvas.style.rotate = ''; canvas.style.scale = ''; canvas.style.transformOrigin = '0 0';
        canvas.style.transform = 'matrix('+next.matrix.join(',')+')'; canvas.style.pointerEvents = 'none';
        Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px',pointerEvents:'auto'});
        target.hidden = false; last = {...next,frames,box:{...box},revision}; onChange(next);
      } else if (active || body.managed(canvas)) reset();
      else placeNormalFeet();
      return true;
    }
    let offRelease = null;
    function reproject() {
      if(closed||retired||!active||!last||!handle.current())return;
      const layout=getLayout();if(!layout||!['X','Y','Width','Height'].every(key=>Number.isFinite(layout[key]))||!(layout.Width>0)||!(layout.Height>0))return;
      const next=geometry(last.box,layout,last.facing,last.angle,last.dy,last.sy);
      if(!handle.publish(next.rect))return;
      canvas.style.transform='matrix('+next.matrix.join(',')+')';
      Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px'});
      last={...last,...next};onChange(next);
    }
    function stop() {
      if(retired)return;retired=true;placeActiveFeet();
      revision++; requested=false;
      if (!handle.current()) return;
      if (!body.capturing(target)) {reset();offRelease?.();handle.close();return;}
      // The renderer may fail while this exact target owns pointer capture.
      // Return its visual transform immediately, keep only that captured target
      // alive, and retire it on its real release without blocking fallback.
      handle.clear();active=false;last=null;
      canvas.style.transform='';canvas.style.transformOrigin='';canvas.style.pointerEvents='';
      target.style.pointerEvents='none';target.style.width='0px';target.style.height='0px';
      onRelease();onChange();
      if (!offRelease) offRelease=body.onRelease(()=>{if(!body.capturing(target)){if(handle.current())target.hidden=true;offRelease?.();offRelease=null;handle.close();}});
    }
    function close() {
      if (closed) return; closed = true;placeActiveFeet(); revision++;
      offRelease?.();offRelease=null;reset();if(handle.current())target.hidden=true;handle.close();
    }
    return Object.freeze({request,tick,snapshot,commit,commitRest,prepareNeuralBody,bodySnapshot,commitNeuralBody,presentCommitted,reproject,reprojectPickup,stop(){clearPickup();stop();},close(){clearPickup();offPickupRelease?.();offPickupRelease=null;close();},get pickupActive(){return !!pickupLast||pickupReserved;},get pickupEvidence(){return pickupLast;},doze(value){if(!retired&&!closed){dozing=value;if(value&&!requested&&!waking)posePhase='doze';}}, get active(){return active;},get evidence(){return last;},get frames(){return frames;}});
  }
  root.WhalePetSleep = Object.freeze({create,measure,geometry,pickupGeometry});
})(window);
