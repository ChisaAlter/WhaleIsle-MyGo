'use strict';
// Fixed-origin presentation only; no application bridge or user package code.
(async function() {
  const fallback = document.getElementById('fallback');
  const canvas = document.getElementById('sprite'), ctx = canvas.getContext('2d');
  const body = window.WhalePetBody?.current;
  const characters = body?.elements || [canvas, fallback];
  const notice = document.getElementById('render-notice');
  // Text the shell rendered server-side reports a degraded renderer. Anything
  // the page writes itself is transient progress and is cleared after painting.
  const serverNotice = notice ? (notice.textContent || '').trim() : '';
  let settings = {}, shiftHeld = false, clickAudio = null;
  let behaviorController = null, currentTask = {}, neuralSleep = null, neuralRest = null, neuralParticles = null, restVersion = 0, latestLayout = null;
  const layoutController = window.WhalePetLayout?.create({body}) || {update(){}};
  function updateLayout(next) {
    if(!next || !['X','Y','Width','Height'].every(key=>Number.isFinite(next[key])) || !(next.Width>0) || !(next.Height>0))return;
    latestLayout=next;layoutController.update(next,currentTask.QuickChatBounds);neuralSleep?.reproject();neuralSleep?.reprojectPickup?.();
  }
  window.addEventListener('whale-pet-layout', event => updateLayout(event.detail));
  try { settings = JSON.parse(document.body.dataset.behavior || '{}'); } catch {}
  let presentation={};
  function applyPresentation(next){
    if(next&&typeof next==='object')presentation=next;
    const dark=presentation.Scheme==='dark'||(presentation.Scheme==='system'&&matchMedia('(prefers-color-scheme: dark)').matches);
    const colors=dark?presentation.Dark:presentation.Light;
    if(colors)for(const [name,key] of Object.entries({bg:'BG',fg:'FG',muted:'Muted',border:'Border',hover:'Hover',accent:'Accent',danger:'Danger'}))document.body.style.setProperty('--pet-'+name,colors[key]);
    if(presentation.Font)document.body.style.setProperty('--pet-font',JSON.stringify(presentation.Font)+',system-ui');
    document.documentElement.style.colorScheme=dark?'dark':'light';
  }
  try{applyPresentation(JSON.parse(document.body.dataset.presentation||'{}'));}catch{}
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change',()=>applyPresentation());
  function applySettings(next) {
    if (next && typeof next === 'object') settings = next;
    const look = document.getElementById('pet-look');
    look.hidden = !settings.lookModel;
    if (look.hidden) look.removeAttribute('href');
    const scale = Math.max(.6, Math.min(1.6, Number(settings.scale) || 1));
    const opacity = Math.max(.3, Math.min(1, Number(settings.opacity) || 1));
    document.documentElement.style.setProperty('--pet-scale', String(scale));
    // Layout comes from the native logical character rectangle, not a page
    // column that reserves space below a shrunken character.
    for (const character of characters) {
      if (!character) continue;
      character.style.opacity = String(opacity);
      character.style.setProperty('--app-region', 'no-drag');
    }
  }
  function modifier(event) { const next = !!event.shiftKey; if (shiftHeld !== next) {shiftHeld = next; applySettings();} }
  document.addEventListener('pointermove', modifier);
  document.addEventListener('pointerenter', modifier, true);
  document.addEventListener('keydown', modifier);
  document.addEventListener('keyup', modifier);
  window.addEventListener('blur', () => {shiftHeld = false; applySettings();});
  function playClickSound() {
    if (!settings.clickSound) return;
    try {
      const Audio = window.AudioContext || window.webkitAudioContext;
      if (!Audio) return;
      clickAudio ||= new Audio();
      if (clickAudio.state === 'suspended') void clickAudio.resume().catch(() => {});
      const time = clickAudio.currentTime, oscillator = clickAudio.createOscillator(), gain = clickAudio.createGain();
      oscillator.type = 'sine'; oscillator.frequency.setValueAtTime(880, time);
      oscillator.frequency.linearRampToValueAtTime(1320, time + .06);
      gain.gain.setValueAtTime(.08, time); gain.gain.exponentialRampToValueAtTime(.0001, time + .09);
      oscillator.connect(gain); gain.connect(clickAudio.destination);
      oscillator.onended = () => {oscillator.disconnect(); gain.disconnect();};
      oscillator.start(time); oscillator.stop(time + .1);
    } catch {}
  }
  document.addEventListener('click', event => {
    if (event.target.closest('a[href]')) playClickSound();
  });
  applySettings(settings);
  let stopped = false, engine = null, raf = 0, spriteTimer = 0, taskTimer = 0;
  // Report only visible islands in this page, never desktop coordinates.
  // Native cursor sampling can restore interaction even while this window is
  // ignoring mouse events. A short periodic report also restores a shown page.
  const hitElements = [...(body?.targets || characters), document.getElementById('pet-status-toggle'),
    document.getElementById('pet-status-card'), document.getElementById('pet-bubble'), notice];
  let hitReportFrame = 0;
  function reportHitRegions() {
    if (stopped || document.hidden || !currentTask.HitEpoch) return;
    const regions = [...hitElements, document.getElementById('pet-notifications'), document.getElementById('pet-look-bubble')]
      .flatMap(element => {
        if (!element || element.hidden || getComputedStyle(element).display === 'none' || getComputedStyle(element).visibility === 'hidden') return [];
        const r = body?.targets.includes(element) ? body.hitRect(element) : element.getBoundingClientRect();
        return r && r.width > 0 && r.height > 0 ? [{X:r.x, Y:r.y, Width:r.width, Height:r.height}] : [];
      });
    const data = JSON.stringify({Width:innerWidth, Height:innerHeight, Regions:regions, Epoch:currentTask.HitEpoch, MenuOpen:!document.getElementById('pet-status-card').hidden, StageBusy:window.whalePetStageBusy===true});
    location.href = 'whale-pet://localhost/interactive-regions?data=' + encodeURIComponent(data);
  }
  function scheduleHitRegions() {
    if (!stopped && !hitReportFrame) hitReportFrame = requestAnimationFrame(() => {
      hitReportFrame = 0; reportHitRegions();
    });
  }
  const hitResizeObserver = new ResizeObserver(scheduleHitRegions);
  for (const element of hitElements) if (element) hitResizeObserver.observe(element);
  const hitVisibilityObserver = new MutationObserver(scheduleHitRegions);
  hitVisibilityObserver.observe(document.body, {subtree:true, childList:true, attributes:true, attributeFilter:['hidden']});
  window.addEventListener('whale-pet-surfaces', () => {layoutController.refresh?.();scheduleHitRegions();});
  const hitRegionTimer = setInterval(reportHitRegions, 250);
  document.addEventListener('visibilitychange', scheduleHitRegions);
  scheduleHitRegions();
  let contactTimer = 0, lastNativeContact = -Infinity;
  function motionContact() {
    const now = performance.now();
    if (stopped || document.hidden || now - lastNativeContact < 1000) return;
    lastNativeContact = now;
    location.href = "whale-pet://localhost/motion-contact";
  }
  for (const event of ["pointerdown", "pointerover", "focusin", "keydown"]) document.addEventListener(event, motionContact);
  contactTimer = setInterval(() => {
    if (document.querySelector((body?.hoverSelector || '#sprite:hover,#fallback:hover') + ',.task:hover,#pet-bubble:hover')) motionContact();
  }, 1000);
  let sleeping = document.body.dataset.sleeping === 'true', resumeRenderer = null, paintSleeping = null;
  const lookBubble=document.createElement('section'),lookText=document.createElement('span'),lookClose=document.createElement('button');
  lookBubble.className='pet-surface';lookBubble.id='pet-look-bubble';lookBubble.hidden=true;lookBubble.setAttribute('aria-live','polite');
  Object.assign(lookBubble.style,{position:'fixed',maxWidth:'240px',padding:'9px',zIndex:'12',overflowY:'auto','--app-region':'no-drag'});
  lookClose.type='button';lookClose.textContent='×';lookClose.setAttribute('aria-label','关闭看看结果 / Dismiss look result');
  lookBubble.append(lookText,lookClose);document.body.append(lookBubble);
  let dismissedLook=0;
  lookClose.addEventListener('click',()=>{dismissedLook=currentTask.Look?.ID||0;lookBubble.hidden=true;window.whalePetLookPinned=false;window.dispatchEvent(new Event('whale-pet-surfaces'));});
  function updateLookBubble(status){
    const visible=!!status?.Message&&(status.Busy||status.Pinned||status.Expires>Date.now())&&status.ID!==dismissedLook;
    window.whalePetLookPinned=visible;lookBubble.hidden=!visible;lookText.textContent=visible?status.Message:'';lookClose.hidden=!!status?.Busy;
    window.dispatchEvent(new Event('whale-pet-surfaces'));
  }
  const careController = window.WhalePetCareAnimation.create({characters, body});
  const interactionController = window.WhalePetInteraction.create({characters, body, getSettings: () => settings, isSleeping: () => sleeping, onTap: playClickSound});
  behaviorController = window.WhalePetBehavior?.create({
    bubble: document.getElementById('pet-bubble'), getSettings: () => settings,
    getStatus: () => currentTask, isSleeping: () => sleeping, body,
  });
  function setSleeping(next) {
    if (stopped || sleeping === next) return;
    sleeping = next;
    neuralSleep?.request(next);
    if (sleeping) {
      cancelAnimationFrame(raf); raf = 0;
      clearInterval(spriteTimer); spriteTimer = 0;
      paintSleeping?.();
    } else if (resumeRenderer) resumeRenderer();
  }
  // Only contact with the character wakes without a care charge, as upstream.
  // Moving over buttons/card text must not immediately undo explicit sleep.
  function contactWake() {
    if (sleeping && !stopped) location.href = 'whale-pet://localhost/wake-contact';
  }
  for (const character of body?.targets || characters) character?.addEventListener('pointerenter', contactWake);
  const stop = () => {
    stopped = true; resumeRenderer = null;
    behaviorController?.close();
    neuralRest?.close();neuralRest=null;
    interactionController.close();
    neuralSleep?.close(); neuralSleep=null;
    careController.close();
    if (clickAudio) void clickAudio.close().catch(() => {});
    cancelAnimationFrame(raf); clearInterval(spriteTimer); clearTimeout(taskTimer); clearInterval(contactTimer);
    clearInterval(hitRegionTimer);
    cancelAnimationFrame(hitReportFrame);
    hitResizeObserver.disconnect(); hitVisibilityObserver.disconnect();
    document.removeEventListener('visibilitychange', scheduleHitRegions);
    if (engine) void Promise.resolve(engine.close()).catch(() => {});
  };
  window.addEventListener('pagehide', stop, { once: true });
  // Updating task text must not reload/reallocate a multi-megabyte model.
  async function taskTick() {
    try {
      const response = await fetch('task.json');
      if (!response.ok) throw new Error('status unavailable');
      const task = await response.json();
      if (stopped) return;
      const nextSleeping=task.Rest?.Sleeping === true;
      if(nextSleeping!==sleeping||(currentTask.HitEpoch&&currentTask.HitEpoch!==task.HitEpoch))restVersion++;
      currentTask = task;
      neuralSleep?.request(nextSleeping);neuralSleep?.doze(task.Rest?.Dozing === true);
      updateLayout(task.Layout);
      careController.update(task);
      applySettings(task.Behavior);
      applyPresentation(task.Presentation);
      document.getElementById('task-status').textContent = ({running:'运行中 / Running',idle:'未运行 / Idle',offline:'离线 / Offline'})[task.State] || '状态未知 / Unknown';
      document.getElementById('task-title').textContent = task.Title || '';
      document.getElementById('care-label').textContent = task.Care?.Label || '';
      document.getElementById('care-message').textContent = task.Care?.Message || '';
      const feed = document.getElementById('feed-file');
      feed.setAttribute('aria-disabled', String(!!task.Care?.Busy));
      if (task.Care?.Busy) feed.removeAttribute('href'); else feed.setAttribute('href', 'feed-file');
      document.getElementById('growth-label').textContent = task.Growth?.Label || '';
      document.getElementById('growth-detail').textContent = task.Growth?.Detail || '';
      document.getElementById('growth-message').textContent = task.Growth?.Message || '';
      for (const [id, enabled] of [['refresh-growth', !task.Growth?.Busy], ['feed-tokens', !!task.Growth?.CanFeed]]) {
        const link = document.getElementById(id);
        link.setAttribute('aria-disabled', String(!enabled));
        if (enabled) link.setAttribute('href', id); else link.removeAttribute('href');
      }
      const pat = document.getElementById('pat-pet');
      pat.setAttribute('aria-disabled', String(!!task.Care?.Busy));
      if (task.Care?.Busy) pat.removeAttribute('href'); else pat.setAttribute('href', 'pat-pet');
      const quickChat = document.getElementById('pet-quick-chat');
      quickChat.setAttribute('aria-disabled', String(settings.chatEnabled === false));
      if (settings.chatEnabled === false) quickChat.removeAttribute('href'); else quickChat.setAttribute('href', 'quick-chat');
      const chat = document.getElementById('pet-chat');
      chat.setAttribute('aria-disabled', String(!!task.Chat?.Busy || settings.chatEnabled === false));
      if (task.Chat?.Busy || settings.chatEnabled === false) chat.removeAttribute('href'); else chat.setAttribute('href', 'chat');
      document.getElementById('pet-chat-status').textContent = task.Chat?.Message || '';
      const look = document.getElementById('pet-look');
      const lookDisabled = !!task.Look?.Busy || !settings.lookModel;
      look.setAttribute('aria-disabled', String(lookDisabled));
      if (lookDisabled) look.removeAttribute('href'); else look.setAttribute('href', 'look-screen');
      document.getElementById('pet-look-status').textContent = task.Look?.Message || '';
      updateLookBubble(task.Look);
      setSleeping(task.Rest?.Sleeping === true);
      behaviorController?.applySettings(settings);
      const rest = document.getElementById('rest-pet');
      rest.textContent = task.Rest?.Label || '睡觉 / Sleep';
      rest.setAttribute('aria-disabled', String(!!task.Care?.Busy));
      if (task.Care?.Busy) rest.removeAttribute('href');
      else rest.setAttribute('href', sleeping ? 'wake-pet' : 'sleep-pet');
      document.getElementById('rest-message').textContent = task.Rest?.Message || '';
      window.WhalePetStatus?.update(task);
      const open = document.getElementById('task-open'); open.replaceChildren();
      if (task.SessionID) {
        const link = document.createElement('a'); link.textContent = '回到会话 / Open chat';
        link.href = 'open?session=' + encodeURIComponent(task.SessionID).replace(/%20/g, '+').replace(/[!'()*]/g, c => '%' + c.charCodeAt(0).toString(16).toUpperCase());
        open.appendChild(link);
      }
    } catch { if (!stopped) document.getElementById('task-status').textContent = '状态未知 / Unknown'; }
    if (!stopped) taskTimer = setTimeout(taskTick, 1500);
  }
  taskTimer = setTimeout(taskTick, 1500);
  function script(src) { return new Promise((resolve,reject) => {
    const tag=document.createElement('script');let timer=setTimeout(()=>done(new Error('runtime timeout')),12000);
    function done(error){clearTimeout(timer);tag.onload=tag.onerror=null;if(error){tag.remove();reject(error);}else resolve();}
    tag.src=src;tag.onload=()=>done();tag.onerror=()=>done(new Error('runtime unavailable'));document.head.appendChild(tag);
  }); }
  function image(src) { return new Promise((resolve,reject) => {
    const img=new Image();let timer=setTimeout(()=>done(new Error('image timeout')),8000);
    function done(error){clearTimeout(timer);img.onload=img.onerror=null;if(error)reject(error);else resolve(img);}
    img.onload=()=>done();img.onerror=()=>done(new Error('image unavailable'));img.src=src;
  }); }
  function stateInput(pose) {
    const pickup=neuralRest?.pickup?.phase;
    interactionController.pose(pose);
    const input=interactionController.renderState?.() || {};
    const stage=careController.frame(performance.now(),{...input,sleeping,pickup:!!pickup,powerSaving:behaviorController?.powerSaving(performance.now())},currentTask);
    if(sleeping)input.state='sleep';else if(!pickup&&stage)Object.assign(input,stage);
    if(!pickup)careController.pose(pose,!!neuralRest);
    input.pose=pose;input.sleepy=currentTask.Rest?.Dozing?0.7:0;
    return input;
  }
  async function startRig() {
    await script('pet-rig.js');
    if(stopped) return;
    const response=await fetch('rig/manifest.json');if(!response.ok)throw new Error('rig manifest');
    const rig=await window.WhalePetRig.create({ctx,width:390,height:492,manifest:await response.json(),loadImage:name=>image('rig/'+name)});
    if(stopped){rig.close();return;}
    engine=rig;canvas.width=390;canvas.height=492;
    const pose=new Float32Array(45);let last=-Infinity;
    function paint(now){rig.frame(now/1000,stateInput(pose));canvas.hidden=false;fallback.hidden=true;if(neuralSleep)reportHitRegions();}
    paintSleeping=()=>{try{const now=performance.now();paint(now);paint(now+100);paint(now+200);}catch{void nextRenderer();}};
    function tick(now) {
      raf=0;if(stopped||sleeping)return;
      if(!document.hidden && now-last>=(behaviorController?.powerSaving(now)?110:1000/60)) {
        last=now;try{paint(now);}catch{void nextRenderer();return;}
      }
      raf=requestAnimationFrame(tick);
    }
    document.body.dataset.renderer='rig';
    // The native pet shows only the character at rest (pet-live2d.html:275-303
    // keeps every overlay hidden until asked). The rig label is a diagnostic, so
    // it only stays visible when the shell already had to report a degraded
    // renderer; otherwise a successful paint clears it and the :empty CSS rule
    // hides the line.
    paint(performance.now());if(sleeping)paintSleeping();
    if(!serverNotice){notice.textContent='';}
    resumeRenderer=()=>{if(!stopped&&!sleeping&&!raf)raf=requestAnimationFrame(tick);};resumeRenderer();
  }
  async function startNeural() {
    notice.textContent='正在载入 THA4… / Loading THA4…';
    await script('tha4.js'); await script('render/ort/ort.all.min.js');
    if (stopped) throw new Error('closed');
    ort.env.wasm.wasmPaths=new URL('render/ort/', location.href).href;
    ort.env.wasm.numThreads=1; ort.env.wasm.proxy=false; ort.env.logLevel='warning';
    const img = await image('render/avatar/character.png');
    if (img.naturalWidth !== 512 || img.naturalHeight !== 512) throw new Error('character shape');
    if (stopped) throw new Error('closed');
    const source=document.createElement('canvas');source.width=source.height=512;
    const sourceCtx=source.getContext('2d',{willReadFrequently:true});sourceCtx.drawImage(img,0,0);
    const pose=new Float32Array(45);
    // A returned frame keeps the direction sampled when its inference began.
    // The reference hanging pickup/throw pivots do not mirror around the feet.
    const frameFacing=input=>input.state==='pick-up'||input.facing!==-1?1:-1;
    const sleep=window.WhalePetSleep?.create({canvas,body,sleeping,normalFeet:true,getLayout:()=>latestLayout || JSON.parse(document.body.dataset.layout || '{}'),onChange:value=>{if(value?.head)careController.placeRestBody?.(value.head);reportHitRegions();},onRelease:()=>careController.releaseBody?.()});
    neuralSleep=sleep;sleep?.doze(currentTask.Rest?.Dozing === true);
    if(sleeping&&!currentTask.Rest)careController.update({Rest:{Sleeping:true}});
    const firstInput=stateInput(pose), firstFacing=frameFacing(firstInput), firstRestVersion=restVersion, firstToken=sleep?.snapshot(firstInput);
    const opened=await WhaleTHA4.open(ort,new URL('render/avatar/model.onnx',location.href).href,
      WhaleTHA4.imageCHW(sourceCtx.getImageData(0,0,512,512).data),()=>stopped);
    const neural=opened.engine, sr=WhalePetSuperResolution.create();
    engine={frame:values=>neural.frame(values),close:async()=>{try{await neural.close();}finally{sr.close();}}};
    if(stopped){await engine.close();engine=null;return;}
    const output=sourceCtx.createImageData(512,512);
    canvas.width=780;canvas.height=984;
    let content=null, contentID=0;
    function present(facing) {
      sr.present(ctx,facing);
      fallback.hidden=true;canvas.hidden=false;
      document.body.dataset.superResolution=sr.ready?'anime4k':'raw';
      notice.textContent=serverNotice;
    }
    function paint(pixels, facing) {
      output.data.set(pixels);sourceCtx.putImageData(output,0,0);sr.prepare(source);present(facing);
    }
    function remember(kind,token,facing,modelPose,version) {
      content=Object.freeze({kind,token,facing,modelPose:Object.freeze(Array.from(modelPose)),version,id:++contentID});
    }
    if(firstRestVersion===restVersion&&!(sleep&&sleeping)) {
      const accepted=sleep?sleep.commit(opened.firstPixels,firstToken,interactionController.renderState?.()||{},()=>paint(opened.firstPixels,firstFacing)):(paint(opened.firstPixels,firstFacing),true);
      // THA4.open's warmup frame uses a zero pose, independently of firstInput.
      if(accepted){remember('rest',firstToken,firstFacing,new Float32Array(45),firstRestVersion);if(sleep?.active)careController.claimBody?.();}
    }
    if(sleep){neuralRest=interactionController.useNeuralRest?.() || null;neuralParticles=careController.useNeuralParticles?.() || null;}
    let last=0,busy=false;
    // Candidate slots follow the source50/110ms clock, including busy slots.
    // Cached presentations do not upload inputs, rerun SR, or synthesize faces.
    async function tick(now) {
      raf=0;if(stopped)return;
      raf=requestAnimationFrame(tick);
      if(document.hidden)return;
      try {
        careController.present(now);
        sleep?.tick(now);const restFrame=sleep?.snapshot({});
        if(restFrame?.rest)neuralRest?.frame(restFrame.rest);
        sleep?.prepareNeuralBody?.(neuralRest?.pickup);
        const displayInput=interactionController.renderState?.()||{},displayRest=sleep?.snapshot({...displayInput,facing:content?.kind==='rest'?content.facing:displayInput.facing},restFrame),pickup=neuralRest?.pickup;
        const usePickup=!!sleep&&!displayRest?.sleeping&&!displayRest?.waking&&!displayRest?.rest?.phase&&!sleep.active&&(!!pickup?.phase||sleep.pickupActive);
        const pickupToken=usePickup?sleep.bodySnapshot(pickup,neuralRest.input):null;
        const displayToken=usePickup?pickupToken:displayRest;
        const displayFacing=usePickup?(pickup?.hanging?1:displayInput.facing===-1?-1:1):content?.facing??frameFacing(displayInput);
        if(sleep&&content?.version===restVersion)sleep.presentCommitted(content,displayToken,usePickup?neuralRest.input:displayInput,()=>present(displayFacing));
        neuralParticles?.frame(now,sleep?.active&&sleeping&&!body?.capturing()&&sleep.evidence?.revision===restFrame?.revision?{sleeping:true,phase:sleep.evidence.elapsed<1.2?'sleep-enter':'sleep',layout:latestLayout}:null);
        if(now-last<(behaviorController?.powerSaving(now)?110:50))return;
        last=now;if(busy)return;busy=true;
        const input=stateInput(pose),version=restVersion,token=sleep?.snapshot(input,restFrame),facing=usePickup?(pickup?.hanging?1:input.facing===-1?-1:1):frameFacing(input);
        const modelPose=Object.freeze(Array.from(pose));
        try {
          const pixels=await engine.frame(pose);
          if(pixels&&!stopped&&!document.hidden&&version===restVersion){
            const accepted=usePickup?sleep.commitNeuralBody(pixels,pickupToken,neuralRest.input,()=>paint(pixels,facing)):sleep?sleep.commitRest(pixels,token,interactionController.renderState?.()||{},()=>paint(pixels,facing)):(paint(pixels,facing),true);
            if(accepted){
              remember(usePickup?'pickup':'rest',usePickup?pickupToken:token,facing,modelPose,version);
              // Completion may arrive after several displayed body ticks. Finish
              // this task with the current geometry, without rerunning the model.
              if(sleep){const current=interactionController.renderState?.()||{},latest=usePickup?sleep.bodySnapshot(neuralRest.pickup,neuralRest.input):sleep.snapshot({...current,facing});const latestFacing=usePickup?(latest.frame?.hanging?1:latest.frame?.facing===-1?-1:1):facing;sleep.presentCommitted(content,latest,usePickup?neuralRest.input:current,()=>present(latestFacing));}
              if(sleep?.active||usePickup)careController.claimBody?.();
            }
          }
        } finally {busy=false;}
      } catch {
        resumeRenderer=null;cancelAnimationFrame(raf);raf=0;
        if(!stopped)void nextRenderer();
      }
    }
    resumeRenderer=()=>{if(!stopped&&!raf)raf=requestAnimationFrame(tick);};
    paintSleeping=resumeRenderer;
    document.body.dataset.renderer='neural';
    document.body.dataset.provider=opened.provider;
    document.body.dataset.gpuInputs=String(neural.gpuInputs);
    document.body.dataset.graphCapture=String(opened.capture);
    resumeRenderer();
  }
  async function startSprite() {
    let sheet=null;
    try{sheet=await image('sprite.webp');if(sheet.naturalWidth!==1536||![1872,2288].includes(sheet.naturalHeight))sheet=null;}catch{}
    if(stopped)return;
    const head=sheet?null:await image('fallback.png');if(stopped)return;
    canvas.width=192;canvas.height=208;let frame=0;
    let spritePointer=null;
    const trackSprite=event=>{const rect=canvas.getBoundingClientRect();spritePointer={x:event.clientX-rect.left-rect.width/2,y:event.clientY-rect.top-rect.height/2};};
    for(const target of body?.targets||characters){target?.addEventListener('pointermove',trackSprite);target?.addEventListener('pointerleave',()=>{spritePointer=null;});}
    const pose=new Float32Array(45);
    function paint(now) {
      const input=stateInput(pose), breath=Math.sin(now/1000*2*Math.PI/3.6);
      ctx.clearRect(0,0,192,208);ctx.save();
      ctx.translate(96,204);ctx.rotate(pose[41]*.035+pose[39]*.02);
      ctx.scale(1-.01*breath,1+.016*breath);ctx.translate(-96,-204-pose[42]*5);
      if(sheet){
        const state=input.state==='running'?(input.facing===-1?'run-left':'run-right'):input.state;
        const {row,col}=window.WhalePetSprite.cell(sheet.naturalHeight/208,frame,state,currentTask,spritePointer);
        ctx.drawImage(sheet,col*192,row*208,192,208,0,0,192,208);
      }
      else ctx.drawImage(head,36,40+pose[38]*2.5,120,120);
      ctx.restore();canvas.hidden=false;fallback.hidden=true;if(neuralSleep)reportHitRegions();
    }
    function start(){
      if(stopped||sleeping||spriteTimer)return;
      spriteTimer=setTimeout(()=>{spriteTimer=0;if(stopped||sleeping)return;if(!document.hidden){frame=(frame+1)%8;paint(performance.now());}start();},behaviorController?.powerSaving(performance.now())?240:120);
    }
    document.body.dataset.renderer=sheet?'sprite':'dynamic-fallback';
    notice.textContent=sheet?'图集动画 / Sprite animation':'简化动态后备 / Basic animated fallback';
    resumeRenderer=start;paintSleeping=()=>paint(performance.now());paint(performance.now());start();
  }
  // Renderer construction and failure transitions are serialized. A failed
  // runtime is never retried forever and a late load cannot resurrect a page.
  const candidates=[];
  if(document.body.dataset.neural==='true') candidates.push(startNeural);
  if(document.body.dataset.rig==='true') candidates.push(startRig);
  candidates.push(startSprite);
  let nextIndex=0, changing=false;
  async function nextRenderer() {
    if(stopped||changing)return;changing=true;
    try {
      resumeRenderer=null;paintSleeping=null;cancelAnimationFrame(raf);raf=0;clearTimeout(spriteTimer);spriteTimer=0;
      neuralRest?.close();neuralRest=null;
      neuralParticles?.close();neuralParticles=null;
      if(neuralSleep){neuralSleep.stop();}
      if(engine){try{await engine.close();}catch{}engine=null;}
      while(!stopped&&nextIndex<candidates.length) {
        try{await candidates[nextIndex++]();return;}
        catch {neuralRest?.close();neuralRest=null;neuralParticles?.close();neuralParticles=null;if(neuralSleep){neuralSleep.stop();}if(engine){try{await engine.close();}catch{}engine=null;}}
      }
      if(!stopped){canvas.hidden=true;fallback.hidden=false;notice.textContent='绘制资源不可用 / Renderer unavailable';}
    } finally {changing=false;}
  }
  await nextRenderer();
})();
