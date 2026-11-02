'use strict';
/*
 * Native desktop-pet idle behaviour, adapted from ChisaAlter/WhaleIsle
 * commit 90465b67ec16765631d54c195af19d9ff3c3a770:
 * src/main/pet-settings.js, src/renderer/pet-live2d.js and pet-dialogue.js.
 *
 * MIT License
 * 
 * Copyright (c) 2026 Chisa
 * 
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 * 
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */
// Pose channels adapted from WhaleIsle e9f584be pet-live2d.js.
// Native Go owns the screen cursor, window trajectory and persisted care.
window.WhalePetInteraction = {create({characters, body, getSettings, isSleeping, onTap}) {
 characters = (body?.elements || characters || []).filter(Boolean);
 const pose = new Float32Array(45);
 const rng=(a,b)=>a+Math.random()*(b-a), clamp01=v=>Math.max(0,Math.min(1,v));
 const smooth=t=>t*t*(3-2*t), actEnv=p=>Math.sin(Math.PI*clamp01(p));
 const TAP_REACTION_MS=1400;
 let dragging=false, dragMoved=false, physVel={x:0,y:0}, drawPos={x:0};
 let snapshot={}, snapshotAt=performance.now(), modeEpoch=0, down=null, taps=[], angryUntil=0, landingAt=-Infinity, throwAt=-Infinity;
 let closed=false, timer=0, raf=0, previousMode='', previousCursor=null, restDriver=null;
 const listeners=[];
 const particles=[];
 let pat={x:0,dir:0,flips:0,since:0}, patUntil=0;
 const listen=(target,type,fn)=>{target.addEventListener(type,fn);listeners.push(()=>target.removeEventListener(type,fn));};
 const action=name=>{location.href='whale-pet://localhost/'+name;};
 const active=()=>body ? body.active() : characters.find(c=>!c.hidden)||characters[0];
const ACTS = [
  { dur: 1.7, apply: (k, p) => { p[41] += 0.55 * k; p[39] += 0.18 * k; } },                    // head tilt
  { dur: 2.0, apply: (k, p) => { p[40] += 0.7 * k; p[38] += 0.55 * k; } },                    // look left
  { dur: 2.0, apply: (k, p) => { p[40] -= 0.7 * k; p[38] -= 0.55 * k; } },                    // look right
  { dur: 1.9, apply: (k, p) => { p[14] = p[15] = Math.max(p[14], 0.8 * k); p[8] = p[9] = 0.6 * k; p[34] = p[35] = 0.35 * k; } }, // happy-eye smile
  { dur: 1.1, apply: (k, p, t) => { p[42] += 0.4 * Math.sin(t * 14) * k; p[44] += 0.35 * k; } }, // happy bounce
  { dur: 0.9, apply: (k, p) => { p[12] = Math.max(p[12], k); } },                             // wink (left)
  { dur: 0.9, apply: (k, p) => { p[13] = Math.max(p[13], k); } },                             // wink (right)
];
const YAWN = { dur: 2.6, apply: (k, p) => { p[12] = p[13] = Math.max(p[12], 0.7 * k); p[18] = p[19] = Math.max(p[18], 0.5 * k); p[6] = p[7] = 0.4 * k; p[39] += 0.2 * k; p[26] = Math.max(p[26], 0.4 * k); } };

// Tap reactions rotate through four expressions.
// Tap reactions rotate through four expressions. Mouth channels are kept
// small: the source art's mouth is a 2px line so the model barely morphs
// it — eyes/brows/head carry the expression instead.
const TAP_KINDS = [
  (k, p) => { p[14] = p[15] = Math.max(p[14], 0.9 * k); p[8] = p[9] = 0.7 * k; p[34] = p[35] = 0.4 * k; p[42] += 0.3 * k; }, // happy ^^ + bounce
  (k, p) => { p[16] = p[17] = 0.9 * k; p[6] = p[7] = 0.8 * k; p[39] -= 0.2 * k; p[42] += 0.35 * k; },                       // surprised + hop
  (k, p) => { p[20] = 0.8 * k; p[21] = 0.4 * k; p[10] = p[11] = 0.6 * k; p[36] = 0.3 * k; p[41] += 0.35 * k; },             // smug + tilt
  (k, p) => { p[0] = p[1] = 0.7 * k; p[2] = p[3] = 0.4 * k; p[22] = p[23] = 0.5 * k; p[40] += 0.3 * Math.sin(k * 6); },     // grumpy shake
];

const idle = {
  t: 0,
  last: performance.now(),
  pby: rng(0, Math.PI * 2), pbz: rng(0, Math.PI * 2), pbr: rng(0, Math.PI * 2),
  phx: rng(0, Math.PI * 2), phy: rng(0, Math.PI * 2),
  bsy: rng(0.6, 1.0), bsz: rng(0.4, 0.7), bsb: rng(1.6, 2.0),
  nextBlink: rng(0.6, 1.6),
  blinkState: 0, blinkTimer: 0,
  blinkDur: 0.06, blinkHold: 0.08,
  // Two-tier gaze: iris snaps fast, head drifts slowly and only partially.
  ix: 0, iy: 0, mx: 0, my: 0,
  targetMx: 0, targetMy: 0,
  tapUntil: 0, tapKind: 0,
  act: null, actStart: 0, nextAct: rng(5, 10),
  lastInteract: performance.now(), sleepy: 0,
  dragVX: 0, lastDrawX: 0,
};

function stepPose() {
  const now = performance.now();
  const dt = Math.min((now - idle.last) / 1000, 0.1);
  idle.last = now;
  idle.t += dt;
  pose.fill(0);

  const breath = 0.40 * Math.sin(idle.t * idle.bsb + idle.pbr);
  const bodyY = 0.35 * Math.sin(idle.t * idle.bsy + idle.pby);
  const bodyZ = 0.30 * Math.sin(idle.t * idle.bsz + idle.pbz);
  const headX = 0.18 * Math.sin(idle.t * 1.1 + idle.phx);
  const headY = 0.14 * Math.sin(idle.t * 1.3 + idle.phy);
  const neck = 0.08 * Math.sin(idle.t * 0.55);
  const irisX = 0.10 * Math.sin(idle.t * 0.45 + idle.phy);
  const irisY = 0.07 * Math.sin(idle.t * 0.55 + idle.phx);

  // Iris leads (fast lerp, full range); head follows slowly and partially —
  // reads as her eyes finding the cursor first, then her head catching up.
  idle.ix += (idle.targetMx - idle.ix) * Math.min(dt * 16, 1);
  idle.iy += (idle.targetMy - idle.iy) * Math.min(dt * 16, 1);
  idle.mx += (idle.targetMx - idle.mx) * Math.min(dt * 5, 1);
  idle.my += (idle.targetMy - idle.my) * Math.min(dt * 5, 1);

  pose[44] = breath;
  pose[42] = bodyY + idle.mx * 0.20;
  pose[43] = bodyZ + idle.my * 0.15;
  pose[39] = headX - idle.my * 0.55;
  pose[40] = headY - idle.mx * 0.60;
  pose[41] = neck;
  pose[37] = irisX - idle.iy * 1.0;
  pose[38] = irisY - idle.ix * 1.0;

  idle.blinkTimer += dt;
  if (idle.blinkState === 0) {
    if (idle.blinkTimer >= idle.nextBlink) {
      idle.blinkState = 1;
      idle.blinkTimer = 0;
      idle.nextBlink = rng(2, 6);
      if (Math.random() < 0.15) { idle.nextBlink = 0.35; } // occasional double-blink
    }
  } else if (idle.blinkState === 1) {
    const v = Math.min(idle.blinkTimer / idle.blinkDur, 1);
    pose[12] = pose[13] = v;
    if (v >= 1) { idle.blinkState = 2; idle.blinkTimer = 0; }
  } else if (idle.blinkState === 2) {
    pose[12] = pose[13] = 1;
    if (idle.blinkTimer >= idle.blinkHold) { idle.blinkState = 3; idle.blinkTimer = 0; }
  } else {
    const v = Math.max(1 - idle.blinkTimer / idle.blinkDur, 0);
    pose[12] = pose[13] = v;
    if (v <= 0) { idle.blinkState = 0; idle.blinkTimer = 0; }
  }

  // Sleepiness: after ~90s without interaction her lids droop, head sags,
  // and she occasionally yawns. Any nearby cursor movement wakes her.
  const idleFor = now - idle.lastInteract;
  idle.sleepy += ((idleFor > 90000 ? 1 : 0) - idle.sleepy) * Math.min(dt * 0.6, 1);
  if (idle.sleepy > 0.01) {
    const s = smooth(idle.sleepy);
    pose[18] = Math.max(pose[18], 0.6 * s);
    pose[19] = Math.max(pose[19], 0.6 * s);
    pose[39] += 0.16 * s;
    pose[44] *= 1 - 0.5 * s;
  }

  // Scheduled micro-action (yawn is much more likely once sleepy).
  if (idle.act) {
    const phase = (now - idle.actStart) / (idle.act.dur * 1000);
    if (phase >= 1) {
      idle.act = null;
    } else {
      idle.act.apply(actEnv(phase), pose, phase * idle.act.dur);
    }
  } else {
    idle.nextAct -= dt;
    if (idle.nextAct <= 0) {
      idle.nextAct = rng(6, 14);
      idle.act = (idle.sleepy > 0.5 && Math.random() < 0.5)
        ? YAWN
        : ACTS[Math.floor(Math.random() * ACTS.length)];
      idle.actStart = now;
    }
  }

  // Tap reaction: rotate through the expression set.
  if (now < idle.tapUntil) {
    const k = actEnv((TAP_REACTION_MS - (idle.tapUntil - now)) / TAP_REACTION_MS);
    TAP_KINDS[idle.tapKind % TAP_KINDS.length](k, pose);
  }

  // Being carried: >< squeezed-shut eyes, brows up, head wobbling, body
  // tilting with the drag — reads as being lifted even though the model's
  // mouth morph is too weak for an open-mouth yell.
  if (dragging && dragMoved) {
    pose[12] = pose[13] = 1.0;
    pose[6] = pose[7] = 0.7;
    pose[26] = 0.35;
    const vx = physVel.x / 20; // px/s → per-frame units the pose expects
    idle.dragVX += (vx - idle.dragVX) * 0.4;
    pose[43] += Math.max(-0.8, Math.min(0.8, idle.dragVX * 0.05));
    pose[40] += 0.18 * Math.sin(idle.t * 9);
    pose[42] -= 0.12;
  }
  if(restDriver?.frame?.phase)applyRestPose(restDriver.frame);
  else if(restDriver?.pickup.frame?.phase)applyPickupPose(restDriver.pickup.frame);
  else applyLiveState();
  isolateClosedEyes();
  idle.lastDrawX = drawPos.x;
}

// Blink/wink must win after idle, tap and live-state eye shapes have composed.
// THA4 distorts eyelids when full natural closure overlaps happy/relaxed eyes.
// Fade only the closing eye's competing shapes; keep the other eye and mouth.
function isolateClosedEyes() {
  const leftClosed = clamp01(pose[12]);
  const rightClosed = clamp01(pose[13]);
  for (let side = 0; side < 2; side += 1) {
    const open = 1 - (side === 0 ? leftClosed : rightClosed);
    for (let ch = 14 + side; ch < 26; ch += 2) { pose[ch] *= open; }
  }
  // Iris rotation is shared by both eyes: retain it for a one-eyed wink.
  const gaze = 1 - Math.min(leftClosed, rightClosed);
  pose[37] *= gaze;
  pose[38] *= gaze;
}


 function burst(kind,count) {
  const c=active(), r=body ? body.rect(c) : c?.getBoundingClientRect();
  if(!r)return;
  for(let i=0;i<count;i++){
   const node=document.createElement("span");node.textContent=kind;
   Object.assign(node.style,{position:"fixed",pointerEvents:"none",zIndex:"3",fontSize:"15px",color:kind==="♥"?"#ee8caa":"#70bfe8"});
   document.body.appendChild(node);
   particles.push({node,x:r.left+r.width*.5,y:r.top+r.height*.3,vx:(i%2?1:-1)*rng(15,65),vy:rng(-100,-40),born:performance.now()});
  }
 }
 function applyLiveState() {
  const now=performance.now();
  if(now<patUntil) {const k=actEnv(1-(patUntil-now)/2600);pose[14]=pose[15]=.9*k;pose[8]=pose[9]=.7*k;}
  if(now<angryUntil) {
   const k=actEnv(1-(angryUntil-now)/1800);
   pose[0]=pose[1]=.9*k;pose[2]=pose[3]=.6*k;
   pose[22]=pose[23]=.7*k;pose[40]+=.3*Math.sin(now/70)*k;
  }
  if(snapshot.Mode==='throw') {
   pose[12]=pose[13]=.9;pose[6]=pose[7]=.8;pose[41]+=.2*Math.sin(now/70);
  }
 }
 // Neural rest composes after the existing base/tap/held programs and before
 // the one closed-eye isolation pass. The body supplies its already-ticked
 // immutable phase/alpha; this layer owns no competing rest envelope.
 function applyRestPose(frame) {
  const {phase,a,time,elapsed}=frame;
  const set=(channel,value)=>{pose[channel]+=(value-pose[channel])*a;};
  const mul=(channel,value)=>{pose[channel]*=1+(value-1)*a;};
  if(phase==='doze') {
   set(18,.7);set(19,.7);set(39,.22+.04*Math.sin(time*1.4));mul(44,.55);
   set(18,.65);set(19,.65);
  } else if(phase==='sleep-enter'||phase==='sleep') {
   const closed=phase==='sleep-enter'?smooth(clamp01(elapsed/1.2)):1;
   if(phase==='sleep-enter')set(39,.2*closed);else {set(39,.16);mul(44,1.5);}
   for(let channel=0;channel<39;channel++)if(channel!==12&&channel!==13)mul(channel,1-closed);
   set(12,closed);set(13,closed);
  } else if(phase==='wake') {
   const p=clamp01(elapsed/1.1);
   set(16,.8*(1-p));set(17,.8*(1-p));
   set(16,.9);set(17,.9);set(22,.5);
  }
 }
 // Pickup inherits the reference's per-display-frame spring step. Its fixed
 // 1/60 step deliberately makes the wall-time swing depend on refresh rate.
 function pickupFrame(frame) {
  const ctl=restDriver.pickup, now=frame.time, mode=snapshot.Mode || '',layout=snapshot.Layout;
  if(mode==='pickup'&&layout){ctl.grabX=snapshot.CursorX-layout.X;ctl.grabY=snapshot.CursorY-layout.Y;}
  const cursorX=mode==='throw'&&layout&&Number.isFinite(ctl.grabX)?layout.X+ctl.grabX:snapshot.CursorX;
  const cursorY=mode==='throw'&&layout&&Number.isFinite(ctl.grabY)?layout.Y+ctl.grabY:snapshot.CursorY;
  const held=mode==='pickup'||mode==='throw';
  const begin=phase=>{ctl.phase=phase;ctl.target=1;ctl.started=ctl.time;ctl.until=phase==='recover'?snapshotAt/1000+1.1:0;};
  if(frame.phase) {ctl.phase=null;ctl.target=0;ctl.alpha=0;}
  else {
   if(held&&(ctl.mode!==mode||!ctl.phase)) {
    if(!(ctl.phase==='pick-up'&&(ctl.mode==='pickup'||ctl.mode==='throw')))begin('pick-up');
    ctl.target=1;
   } else if(!held&&(ctl.mode==='pickup'||ctl.mode==='throw')) {
    if(ctl.mode==='throw')begin('recover');else ctl.target=0;
   }
   ctl.alpha+=(ctl.target-ctl.alpha)*.35;
   if(!ctl.target&&ctl.alpha<.005){ctl.alpha=0;ctl.phase=null;}
   // tickStill clears timed actions after this frame's envelope integration.
   if(ctl.phase==='recover'&&now>ctl.until)ctl.target=0;
  }
  ctl.mode=mode;ctl.time=now;
  ctl.frame=Object.freeze({phase:ctl.phase,alpha:ctl.alpha,a:smooth(clamp01(ctl.alpha)),time:now,started:ctl.started,elapsed:now-ctl.started,dragging:mode==='pickup',thrown:mode==='throw',vx:physVel.x,vy:physVel.y,mode,epoch:modeEpoch,cursorX,cursorY,facing:snapshot.Facing===-1?-1:1});
 }
 function applyPickupPose(frame) {
  const ctl=restDriver.pickup,{a,time,elapsed}=frame;
  const set=(channel,value)=>{pose[channel]+=(value-pose[channel])*a;};
  if(frame.phase==='pick-up') {
   const dt=1/60;
   ctl.velocity+=(-16*ctl.swing)*dt-ctl.velocity*1.7*dt;
   ctl.swing+=ctl.velocity*dt;
   if(!frame.dragging&&Math.abs(ctl.swing)<.02&&Math.abs(ctl.velocity)<.02)ctl.velocity=2.4;
   if(elapsed<.7){set(16,1);set(17,1);set(22,.5);}
   else {set(2,1);set(3,1);set(12,.85);set(13,.85);set(26,Math.sin(time*1.3)>.93?.9:.1);}
   set(40,.12+.06*Math.sin(time*9));
   // The actual 'pick-up' key has no additional panicked expression mapping.
   ctl.frame=Object.freeze({...frame,rotation:ctl.swing*.45,scaleY:1.06,pivot:'grab',swing:ctl.swing,velocity:ctl.velocity});
  } else if(frame.phase==='recover') {
   const p=clamp01(elapsed/1.5);
   set(18,.35*(1-p));set(19,.35*(1-p));
   set(18,.35);set(19,.35);set(41,.4);
   ctl.frame=Object.freeze({...frame,rotation:.2*(1-p)*Math.sin(p*15),dy:3*(1-p),scaleY:1,pivot:null,swing:ctl.swing,velocity:ctl.velocity});
  }
 }
 function driveRest(frame) {
  if(!restDriver||closed||document.hidden)return;
  const changed=restDriver.revision!==frame.revision;
  if(frame.sleeping&&(changed||!restDriver.sleeping))idle.sleepy=0;
  else if(!frame.sleeping&&frame.phase==='wake'&&changed){idle.lastInteract=frame.started*1000;idle.sleepy=0;idle.tapKind=1;idle.tapUntil=frame.started*1000+TAP_REACTION_MS;}
  restDriver.revision=frame.revision;restDriver.sleeping=frame.sleeping;restDriver.frame=frame;pickupFrame(frame);stepPose();
  if(restDriver.pickup.frame?.phase){
   if(down)down.neural=true;
   const value=restDriver.pickup.frame,now=frame.time*1000;
   const land=(now-landingAt)/500,squash=land>=0&&land<1?Math.sin(land*Math.PI):0,tumble=(now-throwAt)/900;
   let rotation=(value.dragging||value.thrown)&&value.alpha>.5?Math.max(-.2,Math.min(.2,value.vx*.0004)):0;
   if(tumble>=0&&tumble<1)rotation+=Math.sin(tumble*Math.PI*2)*.3*(1-tumble);
   restDriver.pickup.frame=Object.freeze({...value,angle:rotation+(value.rotation||0)*value.a,dy:(value.dragging||value.thrown)&&value.alpha>.5?-6:(value.dy||0)*value.a,sx:1+.14*squash,sy:(1-.18*squash)*(1+((value.scaleY||1)-1)*value.a),hanging:value.pivot==='grab'&&value.a>.4&&(value.dragging||value.thrown)});
  }
 }
 function gaze(x,y) {
  const c=active(), r=body ? body.rect(c) : c?.getBoundingClientRect();
  if(!r || !(r.width>0) || !(r.height>0))return;
  idle.targetMx=Math.max(-1,Math.min(1,(x-r.left-r.width*.5)/(r.width*1.4)));
  idle.targetMy=Math.max(-1,Math.min(1,(y-r.top-r.height*.22)/(r.height*1.3)));
  if(x>=r.left&&x<=r.right&&y>=r.top&&y<=r.bottom) idle.lastInteract=performance.now();
 }
 async function poll() {
  try {
   const response=await fetch('interaction.json');
   if(response.ok) {
    const next=await response.json();
    if(closed)return;
    if(next.Landed && next.Landed!==snapshot.Landed) {landingAt=performance.now();if(previousMode==="throw")burst("✦",3);}
    if(next.Mode==='throw'&&previousMode!=='throw')throwAt=performance.now();
    if(next.Mode!==snapshot.Mode||next.Landed!==snapshot.Landed)modeEpoch++;
    previousMode=next.Mode;snapshot=next;snapshotAt=performance.now();
     window.dispatchEvent(new CustomEvent('whale-pet-layout',{detail:next.Layout}));
    dragging=next.Mode==='pickup';dragMoved=dragging;
    physVel={x:next.VX||0,y:next.VY||0};
    if(!previousCursor||previousCursor.x!==next.CursorX||previousCursor.y!==next.CursorY) {
     gaze(next.CursorX,next.CursorY);previousCursor={x:next.CursorX,y:next.CursorY};
    }
   }
  } catch {}
  if(!closed)timer=setTimeout(poll,document.hidden?250:50);
 }
 function end(event,cancelled=false) {
  if(!down || (event.pointerId!=null && event.pointerId!==down.id))return;
  const wasMoved=down.moved, target=down.target, pointerId=down.id;
  down=null;
  action(cancelled?'gesture-cancel':'gesture-release');
  if(target.hasPointerCapture(pointerId))target.releasePointerCapture(pointerId);
   body?.capture(target,pointerId,false);
  if(!wasMoved&&!cancelled) {
   // Only completed taps sound; click/dblclick and capture loss must not replay them.
   onTap?.();
   const now=performance.now();
   taps=taps.filter(t=>now-t<3000);taps.push(now);
   idle.lastInteract=now;
   if(taps.length>=4){angryUntil=now+1800;taps=[];burst("•",8);}
   else {idle.tapKind++;idle.tapUntil=now+TAP_REACTION_MS;}
  }
 }
 for(const c of body?.targets || characters) {
  c.style.touchAction='none';
  listen(c,'pointerdown',e=>{
   if(e.button!==0 || down)return;
   e.preventDefault();
   e.stopPropagation();
   down={x:e.screenX,y:e.screenY,id:e.pointerId,target:c,moved:false};
   c.setPointerCapture(e.pointerId);
   body?.capture(c,e.pointerId,true);
   idle.lastInteract=performance.now();
   action('gesture-press');
  });
  listen(c,'pointermove',e=>{
   gaze(e.clientX,e.clientY);
   if(!down) {
    const r=body ? body.rect(c) : c.getBoundingClientRect(), now=performance.now(), dx=e.clientX-pat.x;
    if(now-pat.since>1600){pat.flips=0;pat.since=now;}
    pat.x=e.clientX;
    if(e.clientY<r.top+r.height*.45 && now>patUntil && Math.abs(dx)>4) {
     const dir=Math.sign(dx);
     if(dir!==pat.dir){pat.dir=dir;pat.flips++;}
     if(pat.flips>=3){pat.flips=0;patUntil=now+2600;action('pat-pet');burst('♥',3);}
    }
    return;
   }
   if(down.id!==e.pointerId)return;
   if(!(e.buttons&1)){end(e,true);return;}
   const settings=getSettings();
   if(!down.moved&&!settings.lockPosition&&(!settings.shiftToDrag||e.shiftKey)&&Math.hypot(e.screenX-down.x,e.screenY-down.y)>5) {
    down.moved=true;action('gesture-pickup');
   }
  });
  listen(c,'pointerup',e=>end(e));
  listen(c,'pointercancel',e=>end(e,true));
  listen(c,'lostpointercapture',e=>end(e,true));
 }
 listen(window,'blur',e=>end(e,true));
 listen(document,'pointermove',e=>gaze(e.clientX,e.clientY));
 function animate(now) {
  if(closed)return;
  if(!document.hidden&&!restDriver&&!isSleeping())stepPose();
  const age=(now-landingAt)/500, squash=age>=0&&age<1?Math.sin(age*Math.PI):0;
  const tumble=(now-throwAt)/900;
  let rotation=(snapshot.Mode==='pickup'||snapshot.Mode==='throw')?Math.max(-.2,Math.min(.2,physVel.x*.0004)):0;
  if(tumble>=0&&tumble<1)rotation+=Math.sin(tumble*Math.PI*2)*.3*(1-tumble);
  const tap=now<idle.tapUntil?actEnv(1-(idle.tapUntil-now)/TAP_REACTION_MS):0;
  for(const c of characters) {
   if(body?.managed(c))continue;
   c.style.transformOrigin='50% 75%';
   c.style.transform='translateY('+(-tap*5)+'px) rotate('+rotation+'rad) scale('+(1+.14*squash)+','+(1-.18*squash)+')';
  }
  for(let i=particles.length-1;i>=0;i--){
   const particle=particles[i], t=(now-particle.born)/1000;
   if(t>1.1){particle.node.remove();particles.splice(i,1);continue;}
   particle.node.style.left=(particle.x+particle.vx*t)+'px';
   particle.node.style.top=(particle.y+particle.vy*t+80*t*t)+'px';
   particle.node.style.opacity=String(1-t/1.1);
  }
  raf=requestAnimationFrame(animate);
 }
 poll();raf=requestAnimationFrame(animate);
 return {
  pose(target){target.set(pose);},
  useNeuralRest(){
   const owner={sleeping:null,frame:null,pickup:{phase:null,mode:'',alpha:0,target:0,started:0,until:0,time:performance.now()/1000,swing:.3,velocity:0,frame:null}};restDriver=owner;
   return Object.freeze({frame(value){if(restDriver===owner)driveRest(value);},get pickup(){return restDriver===owner?owner.pickup.frame:null;},get input(){return {epoch:modeEpoch,mode:snapshot.Mode||''};},close(){if(restDriver===owner){restDriver=null;idle.last=performance.now();}}});
  },
  renderState(){
   const now=performance.now();
   return {state:snapshot.Mode==='pickup'||snapshot.Mode==='throw'?'pick-up':now<angryUntil?'angry':now<patUntil?'react-head':snapshot.Roaming?'running':'',patDirection:pat.dir,facing:snapshot.Facing===-1?-1:1};
  },
  close(){closed=true;restDriver=null;if(down&&(down.neural||body?.managed(active())||down.target===body?.target))end({pointerId:down.id},true);clearTimeout(timer);cancelAnimationFrame(raf);for(const off of listeners)off();for(const p of particles)p.node.remove();},
 };
}};

