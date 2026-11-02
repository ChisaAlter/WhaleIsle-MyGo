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
      return {x:layout.X + w / 2 + px * cos - py * sin, y:layout.Y + h + dy + px * sin + py * cos};
    };
    const points = [[box.x,box.y],[box.right,box.y],[box.x,box.bottom],[box.right,box.bottom]].map(([x,y])=>map(x,y));
    const x = Math.min(...points.map(p=>p.x)), y = Math.min(...points.map(p=>p.y));
    const right = Math.max(...points.map(p=>p.x)), bottom = Math.max(...points.map(p=>p.y));
    // The visible canvas already contains task-29's facing reflection. CSS only
    // rotates that stage; hit coordinates apply facing exactly once above.
    return {head:map((box.x+box.right)/2,box.y),matrix:[cos,sin,-sin*sy,cos*sy,w/2-cos*w/2+sin*h*sy,h+dy-sin*w/2-cos*h*sy],rect:{x,y,left:x,top:y,right,bottom,width:right-x,height:bottom-y}};
  }
  function create({canvas, body, getLayout, onChange = () => {}, onRelease = () => {}, sleeping = false}) {
    const target = body?.target;
    if (!target) return null;
    const handle = body.acquire(canvas);
    let closed = false, retired = false, requested = sleeping, started = performance.now(), revision = 0, frames = 0, box = null, active = false;
    handle.reserve(sleeping);
    let alpha = 0, dozing = false, waking = false, angle = 0, offset = 0, scaleY = 1, last = null;
    function request(value) {
      if (closed || retired || value === requested) return;
      waking = !value && (requested || active);requested = value; started = performance.now(); revision++;
      handle.reserve(requested || active || waking);
    }
    function tick(now) {
      if (closed || retired) return;
      if(waking && now-started>=1100)waking=false;
      alpha += ((requested || dozing || waking ? 1 : 0)-alpha)*.35;
      const a=smooth(alpha);
      if(requested) {const p=smooth(clamp((now-started)/1200));angle=-p*a;offset=14*p*a;scaleY=1;}
      else if(waking) {const p=clamp((now-started)/1100), wave=Math.sin(Math.PI*p);angle=0;offset=-9*wave*a;scaleY=1+.04*wave*a;}
      else {angle=0;offset=0;scaleY=1;}
    }
    function snapshot(input) {
      return {revision,sleeping:requested,held:input.state==='pick-up',facing:input.facing===-1?-1:1,capture:body.captureVersion,time:performance.now(),angle,dy:offset,sy:scaleY,waking};
    }
    function reset() {
      if (!handle.current()) return;
      const wasManaged=body.managed(canvas);handle.clear();
      if (!active) {if(wasManaged)onRelease();return;}
      active = false; last = null;
      target.hidden = true; target.style.pointerEvents = 'none';
      canvas.style.pointerEvents = ''; canvas.style.transform = ''; canvas.style.transformOrigin = '';
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
        canvas.style.rotate = ''; canvas.style.scale = ''; canvas.style.transformOrigin = '0 0';
        canvas.style.transform = 'matrix('+next.matrix.join(',')+')'; canvas.style.pointerEvents = 'none';
        Object.assign(target.style,{left:(next.rect.x-8)+'px',top:(next.rect.y-8)+'px',width:(next.rect.width+16)+'px',height:(next.rect.height+16)+'px',pointerEvents:'auto'});
        target.hidden = false; last = {...next,frames,box:{...box},revision}; onChange(next);
      } else if (active || body.managed(canvas)) reset();
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
      if(retired)return;retired=true;
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
      if (closed) return; closed = true; revision++;
      offRelease?.();offRelease=null;reset();if(handle.current())target.hidden=true;handle.close();
    }
    return Object.freeze({request,tick,snapshot,commit,reproject,stop,close,doze(value){if(!retired&&!closed)dozing=value;}, get active(){return active;},get evidence(){return last;},get frames(){return frames;}});
  }
  root.WhalePetSleep = Object.freeze({create,measure,geometry});
})(window);
