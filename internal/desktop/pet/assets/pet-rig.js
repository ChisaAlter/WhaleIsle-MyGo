"use strict";
// Adapted from ChisaAlter/WhaleIsle e9f584be6d3b491ce777bb6823f30bc2bbc6f195
// src/renderer/pet-live2d.js. Copyright (c) 2026 Chisa; MIT, see rig/LICENSE.
// Actual original tail/shell composition. No SDK, remote code or model required.
window.WhalePetRig = {async create({ctx: ctx2d, width=390, height=492, manifest: mf, loadImage}) {
  const entries = [mf?.body, mf?.tail, ...Object.values(mf?.shells || {})];
  const validBox = box => Array.isArray(box) && box.length===4 && box.every(Number.isFinite) && box[2]>box[0] && box[3]>box[1];
  if (!validBox(mf?.char_bbox) || entries.some(e => !e || !validBox(e.bbox_in_master) || !['shell.png','tail.png'].includes(e.file))) throw new Error('Invalid rig resource manifest');
  const byFile=new Map();
  for(const entry of entries) if(!byFile.has(entry.file)) byFile.set(entry.file,Promise.resolve().then(()=>loadImage(entry.file)));
  await Promise.all(byFile.values());
  const imgs={body:await byFile.get(mf.body.file),tail:await byFile.get(mf.tail.file)};
  for(const [key,entry] of Object.entries(mf.shells||{})) imgs['shell:'+key]=await byFile.get(entry.file);
  if(!imgs.body?.naturalWidth || !imgs.tail?.naturalWidth || !imgs['shell:neutral']?.naturalWidth) throw new Error('Rig images missing');
  const rig={mf,imgs,ready:true}, pose=new Float32Array(45);
  const petW=()=>width, petH=()=>height, smooth=t=>t*t*(3-2*t), clamp01=t=>Math.max(0,Math.min(1,t));
  const drawPos={x:0,y:0}, pointer={x:0,y:0}, idle={sleepy:0}, settings={opacity:1};
  const stillCtl={name:'',alpha:0}, fx={rot:0,dy:0,sx:1,sy:1}, patTrack={dir:0};
  let dragging=false, thrown=false, physPoint=null, feed=null, facing=1, charRect=null, closed=false, previousTime=null;
const RIG_NECK = [1018, 621];
const RIG_TAIL_ROOT = [1152, 741];
const RIG_FEET = [1018, 955];
const RIG_GRAB = [1018, 330];
const RIG_NUM_KEYS = ['bodyRot', 'bodySx', 'bodySy', 'headRot', 'headDx', 'headDy',
  'tailRot', 'allRot', 'allDx', 'allDy'];
let rigT = 0;
let rigStateT0 = 0; // rigT when the current rig state was entered
// Head-expression crossfade: expression-category swaps blend the outgoing
// head out over ~140ms instead of hard-cutting (the stillCtl alpha envelope
// only lerps numeric pose params, so the face would pop at the discrete
// flip). Blink-band steps (open/half/closed) are excluded — the three bands
// already are the in-between frames, and a 140ms dissolve over the ~20ms
// ramp steps just smears her eyes instead of snapping the lid.
const BLINK_BANDS = new Set(['neutral', 'half-closed', 'eyes-closed']);
let rigShell = { cur: null, prev: null, swapT: 0 };
let rigSwing = 0.3;
let rigSwingV = 0;
const RIG_ENTRY = { rig: true }; // non-null stillCtl.entry marker

function rigScale() {
  const b = rig.mf.char_bbox;
  // Original paints onto a whole-screen surface; the native canvas is bounded.
  // Fit the sideways sleeping silhouette instead of clipping its head/feet.
  const sleepFit=stillCtl.name==='sleep'?1-.4*smooth(stillCtl.alpha):1;
  return (petH() * 0.96 * sleepFit) / Math.max(1, b[3] - b[1]);
}

// Idle program: pose channels + cursor gaze + breathing → part params.
// State programs override selected fields; stillCtl.alpha blends them in.
function rigBase() {
  const t = rigT;
  const breathe = Math.sin((t * 2 * Math.PI) / 3.6);
  const P = {
    expr: 'neutral', pivot: null, body: null,
    bodyRot: 0.008 * Math.sin(t * 1.1),
    bodySx: 1 - 0.010 * breathe, bodySy: 1 + 0.016 * breathe,
    headRot: 0.02 * Math.sin(t * 1.7),
    headDx: 0, headDy: 1.8 * breathe,
    tailRot: 0.14 * Math.sin(t * 2.4) + 0.05 * Math.sin(t * 5.1),
    allRot: 0, allDx: 0, allDy: 0,
  };
  if (pointer.x || pointer.y) {
    const gx = Math.max(-1, Math.min(1,
      (pointer.x - (drawPos.x + petW() / 2)) / (petW() * 0.9)));
    const gy = Math.max(-1, Math.min(1,
      (pointer.y - (drawPos.y + petH() * 0.25)) / (petH() * 0.9)));
    P.headRot += 0.06 * gx;
    P.headDx += 5 * gx;
    P.headDy += 2.5 * gy;
  }
  // Micro-acts / taps / yawn ride the pose channels — mapped to the
  // nearest head variant + head offsets.
  P.headRot += pose[41] * 0.035 + pose[39] * 0.02;
  P.headDx += pose[40] * 4;
  P.headDy += pose[38] * 2.5 - pose[42] * 6;
  P.allDy -= pose[42] * 5;
  // Blink channel is analog (60ms ramp each way): three bands give the
  // blink a real closing phase instead of a one-frame snap.
  const eyeV = Math.max(pose[12], pose[13]);
  if (eyeV > 0.78) { P.expr = 'eyes-closed'; } else if (eyeV > 0.35) { P.expr = 'half-closed'; } else if (pose[14] > 0.55 || pose[15] > 0.55 || pose[20] > 0.5) { P.expr = 'happy'; } else if (pose[26] > 0.35 || pose[18] > 0.5 || pose[19] > 0.5 || pose[16] > 0.6 || pose[17] > 0.6) { P.expr = 'mouth-open'; } else if (pose[0] > 0.5 && pose[1] > 0.5) { P.expr = 'angry'; }
  const s = smooth(idle.sleepy || 0);
  if (s > 0) {
    P.headRot += 0.14 * s;
    P.headDy += 9 * s;
    if (s > 0.55) { P.expr = 'eyes-closed'; } else if (s > 0.3) { P.expr = 'half-closed'; }
  }
  return P;
}

// Per-state motion programs. Each mutates a params object: expression
// head, part transforms, and the global pivot ('feet' rotates the whole
// rig about the foot line; 'grab' hangs her from RIG_GRAB at physPoint).
const RIG_STATES = {
  'sleep': (P, t) => {
    P.expr = 'eyes-closed';
    P.pivot = 'feet';
    P.allRot = -1.0;
    P.allDy = 12;
    P.headRot += 0.05 + 0.03 * Math.sin(t * 1.2);
    P.bodySy = 1 + 0.030 * Math.sin((t * 2 * Math.PI) / 4.4);
    // Tail curls forward across her front — sells "curled up asleep"
    // without needing a dedicated torso asset.
    P.tailRot = 0.55 + 0.04 * Math.sin(t * 1.1);
  },
  'pick-up': (P, t, dt) => {
    P.pivot = 'grab';
    // Raised-arms torso drops in as manifest `bodies.pickup` — a no-op
    // (default body) until the asset lands.
    P.body = 'pickup';
    // Grab arc: a surprised gasp in the first ~0.7s, then worried with
    // periodic wails (squeezed eyes + open mouth).
    const heldFor = t - rigStateT0;
    P.expr = heldFor < 0.7 ? 'surprised'
      : (Math.sin(t * 1.3) > 0.93 ? 'wail' : 'worried');
    rigSwingV += (-16 * rigSwing) * dt - rigSwingV * 1.7 * dt;
    rigSwing += rigSwingV * dt;
    if (!dragging && Math.abs(rigSwing) < 0.02 && Math.abs(rigSwingV) < 0.02) {
      rigSwingV = 2.4;
    }
    P.allRot = rigSwing * 0.45;
    // Held by the scruff: body hangs slightly stretched, chin drops,
    // head does a small scared wobble, tail trails limply.
    P.headRot += 0.12 + 0.06 * Math.sin(t * 9);
    P.headDy += 5 + 2 * Math.sin(t * 8);
    P.bodySy = 1.07 + 0.012 * Math.sin(t * 3.1);
    P.tailRot = 0.15 * Math.sin(t * 2.5) - 0.08;
  },
  'eat': (P, t) => {
    // Food arriving gets a brief delighted gasp before she starts chewing.
    P.expr = (t - rigStateT0) < 0.8 ? 'surprised'
      : (Math.sin(t * 7) > 0 ? 'mouth-open' : 'neutral');
    P.headRot += 0.05 * Math.sin(t * 2.2) + 0.03;
    P.headDy += 3 * Math.abs(Math.sin(t * 7));
    P.tailRot = 0.10 * Math.sin(t * 3);
    // Lean her head toward the bowl while it's on screen.
    if (feed) {
      const dir = Math.sign(feed.bowlX - (drawPos.x + petW() / 2)) || 1;
      P.headRot += dir * 0.07;
      P.headDx += dir * 4;
    }
  },
  'running': (P, t) => {
    P.expr = 'happy';
    P.pivot = 'feet';
    P.allRot = 0.07;
    P.allDy = -6 * Math.abs(Math.sin(t * 9));
    P.bodySy = 1 + 0.03 * Math.sin(t * 18);
    P.headRot += 0.04 * Math.sin(t * 9);
    P.headDy += 2.5 * Math.sin(t * 18);
    P.tailRot = 0.35 * Math.sin(t * 9);
  },
  'react-head': (P, t) => {
    // A pat lands as a tiny "oh!" before melting into the happy wobble.
    P.expr = (t - rigStateT0) < 0.45 ? 'surprised'
      : (Math.sin(t * 5) > 0.7 ? 'mouth-open' : 'happy');
    P.headRot += 0.10 * Math.sin(t * 5);
    P.headDx += (patTrack.dir || 0) * 7;
    P.bodySy = 1 + 0.02 * Math.sin(t * 5);
    P.tailRot = 0.3 * Math.sin(t * 5.5);
  },
  'angry': (P, t) => {
    // Mostly furious, with brief squeezed-eye frustration beats.
    P.expr = Math.sin(t * 0.9) > 0.94 ? 'eyes-closed' : 'angry';
    P.allDx = 2.5 * Math.sin(t * 22);
    P.headRot += 0.06 * Math.sin(t * 14);
    P.tailRot = 0.1 * Math.sin(t * 11);
  },
  'celebrate': (P, t) => {
    // Mouth opens at the top of each bounce — cheering, not a frozen grin.
    P.expr = Math.abs(Math.sin(t * 5.2)) > 0.6 ? 'mouth-open' : 'happy';
    P.allDy = -14 * Math.abs(Math.sin(t * 5.2));
    P.bodySy = 1 + 0.03 * Math.sin(t * 10.4);
    P.headRot += 0.05 * Math.sin(t * 5.2);
    P.tailRot = 0.25 * Math.sin(t * 6.5);
  },
  'star': (P, t) => {
    P.expr = 'mouth-open'; // open eyes + mouth, star pupils drawn on top
    P.allDy = -8 * Math.abs(Math.sin(t * 4.4));
    P.headRot += 0.09 * Math.sin(t * 3.1);
    P.tailRot = 0.4 * Math.sin(t * 5.5);
  },
  'greet': (P, t) => {
    // Alternating open mouth — waving hello reads as actually saying it —
    // plus a playful wink once per wave cycle.
    const winkPh = t % 5;
    P.expr = (winkPh > 1 && winkPh < 1.5) ? 'wink'
      : (Math.sin(t * 2.4) > 0.3 ? 'mouth-open' : 'happy');
    P.pivot = 'feet';
    P.allRot = 0.06 * Math.sin(t * 2.4);
    P.headRot += 0.10 * Math.sin(t * 2.4);
    P.allDy = -5 * Math.abs(Math.sin(t * 2.4));
    P.tailRot = 0.2 * Math.sin(t * 4);
  },
  'startle': (P, t) => {
    // Jolt awake: surprised face, rapid little hops, tail flicks up.
    P.expr = 'surprised';
    P.pivot = 'feet';
    P.allDy = -9 * Math.abs(Math.sin(t * 8));
    P.headRot += 0.05 * Math.sin(t * 16);
    P.tailRot = 0.45;
  },
  'tail-swing': (P, t) => {
    P.expr = 'happy';
    P.tailRot = 0.45 * Math.sin(t * 7);
    P.headRot += 0.03 * Math.sin(t * 2.4);
    P.bodyRot += 0.02 * Math.sin(t * 3.5);
  },
};

function drawRigPart(img, bb, pivot, o) {
  if (!img || !img.naturalWidth) {
    return;
  }
  ctx2d.save();
  ctx2d.translate(pivot[0] + (o.dx || 0), pivot[1] + (o.dy || 0));
  ctx2d.rotate(o.rot || 0);
  ctx2d.scale(o.sx || 1, o.sy || 1);
  ctx2d.drawImage(img, bb[0] - pivot[0], bb[1] - pivot[1]);
  ctx2d.restore();
}

// Four-pointed sparkle (star-pupil overlay), in current transform space.
function drawSparkle(x, y, r) {
  ctx2d.beginPath();
  for (let i = 0; i < 8; i += 1) {
    const a = (i * Math.PI) / 4;
    const rr = i % 2 === 0 ? r : r * 0.32;
    const px = x + Math.cos(a) * rr;
    const py = y + Math.sin(a) * rr;
    if (i === 0) { ctx2d.moveTo(px, py); } else { ctx2d.lineTo(px, py); }
  }
  ctx2d.closePath();
  ctx2d.fillStyle = '#ffdf6b';
  ctx2d.fill();
  ctx2d.beginPath();
  ctx2d.arc(x, y, r * 0.22, 0, Math.PI * 2);
  ctx2d.fillStyle = '#fff';
  ctx2d.fill();
}

function drawRig() {
  const mf = rig.mf;
  const box = mf.char_bbox;
  const cw = box[2] - box[0];
  const S = rigScale();
  const base = rigBase();
  const P = { ...base };
  const prog = stillCtl.name && RIG_STATES[stillCtl.name];
  if (prog && stillCtl.alpha > 0.01) {
    const over = { ...base };
    prog(over, rigT, 1 / 60);
    const a = smooth(clamp01(stillCtl.alpha));
    for (const k of RIG_NUM_KEYS) {
      P[k] = base[k] + (over[k] - base[k]) * a;
    }
    if (a > 0.4) {
      P.expr = over.expr;
      P.pivot = over.pivot;
      P.body = over.body;
    }
  }
  P.allRot += fx.rot;
  P.allDy += fx.dy / S;
  const opacity = Math.max(0.3, Math.min(1, settings.opacity || 1));
  ctx2d.globalAlpha = opacity;
  const hanging = P.pivot === 'grab' && (dragging || thrown) && physPoint;
  ctx2d.save();
  if (hanging) {
    // GRAB sits at the physics point; the rig swings about it.
    ctx2d.translate(physPoint.x, physPoint.y);
    ctx2d.scale(S, S);
    ctx2d.rotate(P.allRot);
    ctx2d.translate(P.allDx - RIG_GRAB[0], P.allDy - RIG_GRAB[1]);
  } else {
    // Feet anchored to the pet's bottom line, centered in the draw box.
    ctx2d.translate(drawPos.x + petW() / 2, drawPos.y + petH() - 2);
    ctx2d.scale(S * (facing < 0 ? -1 : 1), S);
    const pv = RIG_FEET;
    // Parts draw at absolute master coords — pre-subtract pv so the feet
    // pivot (not the master 0,0) lands on the anchor.
    ctx2d.translate(P.allDx - pv[0], P.allDy - pv[1]);
    ctx2d.translate(pv[0], pv[1]);
    ctx2d.rotate(P.allRot);
    ctx2d.scale(fx.sx, fx.sy);
    ctx2d.translate(-pv[0], -pv[1]);
  }
  drawRigPart(rig.imgs.tail, mf.tail.bbox_in_master, RIG_TAIL_ROOT, { rot: P.tailRot });
  // Expression = whole-shell variant swap (head+body are fused — there is
  // no neck seam). Head tilt/bob channels fold into the shell transform;
  // blink-band swaps stay instant because blink shells are pixel-identical
  // outside the baked face rect.
  // P.body (state programs) may name a manifest `shells` variant that
  // replaces the expression shell wholesale (e.g. a raised-arms pickup
  // pose) — the extension hook for future pose assets.
  const shellKey = (P.body && mf.shells && mf.shells[P.body]) ? P.body
    : (mf.shells && mf.shells[P.expr] ? P.expr : 'neutral');
  if (shellKey !== rigShell.cur) {
    const bandStep = BLINK_BANDS.has(shellKey) && BLINK_BANDS.has(rigShell.cur);
    rigShell.prev = bandStep ? null : rigShell.cur;
    rigShell.cur = shellKey;
    rigShell.swapT = rigT;
  }
  const shellBlend = rigShell.prev ? Math.min(1, (rigT - rigShell.swapT) / 0.14) : 1;
  const prevImg = rigShell.prev ? rig.imgs[`shell:${rigShell.prev}`] : null;
  const shellCand = rig.imgs[`shell:${shellKey}`];
  const shellDflt = rig.imgs['shell:neutral'];
  const shellImg = (shellCand && shellCand.naturalWidth) ? shellCand
    : (shellDflt && shellDflt.naturalWidth ? shellDflt : rig.imgs.body);
  const shellBb = ((mf.shells && mf.shells[shellKey]) || mf.body).bbox_in_master;
  const pvX = RIG_NECK[0], pvY = RIG_NECK[1] + 280;
  ctx2d.save();
  ctx2d.translate(pvX + P.headDx, pvY + P.headDy);
  ctx2d.rotate(P.bodyRot + P.headRot * 0.7);
  ctx2d.scale(P.bodySx, P.bodySy);
  if (prevImg && prevImg.naturalWidth && shellBlend < 1) {
    const pb = ((mf.shells && mf.shells[rigShell.prev]) || mf.body).bbox_in_master;
    ctx2d.globalAlpha = opacity * (1 - shellBlend);
    ctx2d.drawImage(prevImg, pb[0] - pvX, pb[1] - pvY);
  }
  ctx2d.globalAlpha = opacity * shellBlend;
  ctx2d.drawImage(shellImg, shellBb[0] - pvX, shellBb[1] - pvY);
  ctx2d.globalAlpha = opacity;
  // 'star' draws literal star pupils over the happy face — the only
  // expression whose marker can't come from the model's own vocabulary.
  if (stillCtl.name === 'star' && stillCtl.alpha > 0.4) {
    const tw = 0.8 + 0.2 * Math.sin(rigT * 14);
    for (const ex of [912, 1012]) {
      drawSparkle(ex - pvX, 550 - pvY, 20 * tw);
    }
  }
  ctx2d.restore();
  ctx2d.restore();
  ctx2d.globalAlpha = 1;
  // Hit-test box in drawPos-local coords (idle pose; states widen it via
  // stillDrawRect).
  charRect = {
    x: petW() / 2 + (box[0] - RIG_FEET[0]) * S,
    right: petW() / 2 + (box[2] - RIG_FEET[0]) * S,
    y: petH() - 2 + (box[1] - RIG_FEET[1]) * S,
    bottom: petH() - 2 + (box[3] - RIG_FEET[1]) * S,
  };
}


  return {
    frame(time, input={}) {
      if(closed) return;
      rigT=Number.isFinite(time)?time:0;
      const dt=previousTime===null?0:Math.max(0,Math.min(.1,rigT-previousTime));previousTime=rigT;
      pose.fill(0);if(input.pose)pose.set(input.pose.subarray(0,45));
      const name=Object.hasOwn(RIG_STATES,input.state)?input.state:'';
      if(name!==stillCtl.name){stillCtl.name=name;rigStateT0=rigT;stillCtl.alpha=0;}
      stillCtl.alpha=Math.min(1,stillCtl.alpha+dt/.14);
      idle.sleepy=clamp01(Number(input.sleepy)||0);
      const sleepBlend=name==='sleep'?smooth(stillCtl.alpha):0;
      drawPos.x=width*.30*sleepBlend;drawPos.y=-height*.20*sleepBlend;
      dragging=input.state==='pick-up'; thrown=false;
      // Native window owns trajectory; local grab pivot stays inside its box.
      physPoint=dragging?{x:width/2,y:height*.12}:null;
      patTrack.dir=Number(input.patDirection)||0;
      facing=input.facing===-1?-1:1;
      ctx2d.clearRect(0,0,width,height);drawRig();
      return charRect;
    },
    close(){closed=true;rig.ready=false;for(const key of Object.keys(imgs))delete imgs[key];byFile.clear();},
  };
}};
