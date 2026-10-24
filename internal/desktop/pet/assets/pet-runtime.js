'use strict';
// Fixed-origin presentation only; no application bridge or user package code.
(async function() {
  const fallback = document.getElementById('fallback');
  const canvas = document.getElementById('sprite'), ctx = canvas.getContext('2d');
  const notice = document.getElementById('render-notice');
  // Text the shell rendered server-side reports a degraded renderer. Anything
  // the page writes itself is transient progress and is cleared after painting.
  const serverNotice = notice ? (notice.textContent || '').trim() : '';
  let settings = {}, shiftHeld = false, clickAudio = null;
  let behaviorController = null, currentTask = {};
  try { settings = JSON.parse(document.body.dataset.behavior || '{}'); } catch {}
  function applySettings(next) {
    if (next && typeof next === 'object') settings = next;
    const look = document.getElementById('pet-look');
    look.hidden = !(settings.lookProvider && settings.lookModel);
    if (look.hidden) look.removeAttribute('href');
    const scale = Math.max(.6, Math.min(1.6, Number(settings.scale) || 1));
    const opacity = Math.max(.3, Math.min(1, Number(settings.opacity) || 1));
    document.documentElement.style.setProperty('--pet-scale', String(scale));
    document.querySelector('.task').style.maxHeight = 'calc(100vh - ' + Math.ceil(130 * scale) + 'px)';
    for (const character of [canvas, fallback]) {
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
  const hitElements = [canvas, fallback, document.getElementById('pet-status-toggle'),
    document.getElementById('pet-status-card'), document.getElementById('pet-bubble'), notice];
  let hitReportFrame = 0;
  function reportHitRegions() {
    if (stopped || document.hidden || !currentTask.HitEpoch) return;
    const regions = hitElements
      .flatMap(element => {
        if (!element || element.hidden || getComputedStyle(element).display === 'none') return [];
        const r = element.getBoundingClientRect();
        return r.width > 0 && r.height > 0 ? [{X:r.x, Y:r.y, Width:r.width, Height:r.height}] : [];
      });
    const data = JSON.stringify({Width:innerWidth, Height:innerHeight, Regions:regions, Epoch:currentTask.HitEpoch});
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
  hitVisibilityObserver.observe(document.body, {subtree:true, attributes:true, attributeFilter:['hidden']});
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
    if (document.querySelector("#sprite:hover,#fallback:hover,.task:hover,#pet-bubble:hover")) motionContact();
  }, 1000);
  let sleeping = document.body.dataset.sleeping === 'true', resumeRenderer = null, paintSleeping = null;
  const careController = window.WhalePetCareAnimation.create({characters: [canvas, fallback]});
  const interactionController = window.WhalePetInteraction.create({characters: [canvas, fallback], getSettings: () => settings, isSleeping: () => sleeping, onTap: playClickSound});
  behaviorController = window.WhalePetBehavior?.create({
    bubble: document.getElementById('pet-bubble'), getSettings: () => settings,
    getStatus: () => currentTask, isSleeping: () => sleeping,
  });
  function setSleeping(next) {
    if (stopped || sleeping === next) return;
    sleeping = next;
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
  canvas.addEventListener('pointerenter', contactWake);
  fallback.addEventListener('pointerenter', contactWake);
  const stop = () => {
    stopped = true; resumeRenderer = null;
    behaviorController?.close();
    interactionController.close();
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
      currentTask = task;
      careController.update(task);
      applySettings(task.Behavior);
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
      const lookDisabled = !!task.Look?.Busy || !(settings.lookProvider && settings.lookModel);
      look.setAttribute('aria-disabled', String(lookDisabled));
      if (lookDisabled) look.removeAttribute('href'); else look.setAttribute('href', 'look-screen');
      document.getElementById('pet-look-status').textContent = task.Look?.Message || '';
      setSleeping(task.Rest?.Sleeping === true);
      behaviorController?.applySettings(settings);
      const rest = document.getElementById('rest-pet');
      rest.textContent = task.Rest?.Label || '睡觉 / Sleep';
      rest.setAttribute('aria-disabled', String(!!task.Care?.Busy));
      if (task.Care?.Busy) rest.removeAttribute('href');
      else rest.setAttribute('href', sleeping ? 'wake-pet' : 'sleep-pet');
      document.getElementById('rest-message').textContent = task.Rest?.Message || '';
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
    interactionController.pose(pose);careController.pose(pose);
    const input=interactionController.renderState?.() || {};
    const action=currentTask.Care?.Action;
    const age=action?Date.now()-action.ID:Infinity;
    if(sleeping) input.state='sleep';
    else if(age>=0 && age<2800) input.state=action.Kind==='pat'?'react-head':'eat';
    else if(age>=2800 && age<5000 && action?.Level>0) input.state='celebrate';
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
    function paint(now){rig.frame(now/1000,stateInput(pose));canvas.hidden=false;fallback.hidden=true;}
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
    let session=null;
    try {
    notice.textContent='正在载入 THA4… / Loading THA4…';
    await script('tha4.js'); await script('render/ort/ort.all.min.js');
    if (stopped) throw new Error('closed');
    ort.env.wasm.wasmPaths=new URL('render/ort/', location.href).href;
    ort.env.wasm.numThreads=1; ort.env.wasm.proxy=false; ort.env.logLevel='warning';
    let provider='wasm';
    for (const ep of ['webnn','webgpu','wasm']) {
      try { session=await ort.InferenceSession.create(new URL('render/avatar/model.onnx', location.href).href,{executionProviders:[ep]}); provider=ep; break; }
      catch { if (stopped) throw new Error('closed'); }
    }
    if (!session) throw new Error('No supported inference provider');
    if (stopped) { await session.release(); session=null; throw new Error('closed'); }
    const img = new Image();
    await new Promise((resolve,reject) => {img.onload=resolve;img.onerror=()=>reject(new Error('character unavailable'));img.src='render/avatar/character.png';});
    if (img.naturalWidth !== 512 || img.naturalHeight !== 512) throw new Error('character shape');
    if (stopped) { await session.release(); session=null; throw new Error('closed'); }
    const source=document.createElement('canvas');source.width=source.height=512;
    const sourceCtx=source.getContext('2d',{willReadFrequently:true}); sourceCtx.drawImage(img,0,0);
    engine=WhaleTHA4.createEngine(ort,session,WhaleTHA4.imageCHW(sourceCtx.getImageData(0,0,512,512).data));
    session=null; // Engine owns disposal from now on.
    const output=sourceCtx.createImageData(512,512), pose=new Float32Array(45);
    canvas.width=390; canvas.height=492;
    let last=0, busy=false, first=true;
    // Upstream's 20fps inference budget; no backlog when a frame is slower.
    // WASM remains adaptive rather than silently capped to an unusable 4fps.
    async function tick(now) {
      raf=0;
      if (stopped || sleeping) return;
      raf=requestAnimationFrame(tick);
      if (document.hidden || busy || now-last < (behaviorController?.powerSaving(now)?110:50)) return;
      last=now; busy=true;
      try {
        interactionController.pose(pose);
        careController.pose(pose);
        const pixels=await engine.frame(pose);
        if (!pixels || stopped || sleeping || document.hidden) return;
        let alphaMax=0;for(let i=3;i<pixels.length;i+=4)alphaMax=Math.max(alphaMax,pixels[i]);
        if(alphaMax<4){if(first)throw new Error('transparent first frame');return;}
        output.data.set(pixels);sourceCtx.putImageData(output,0,0);
        ctx.clearRect(0,0,390,492);ctx.drawImage(source,60,20,390,492,0,0,390,492);
        fallback.hidden=true;canvas.hidden=false;first=false;
        notice.textContent='THA4 · '+provider;
      } catch {
        resumeRenderer=null;cancelAnimationFrame(raf);raf=0;try {await engine.close();} catch {} engine=null;
        if (!stopped) void nextRenderer();
      } finally { busy=false; }
    }
    // No concurrent session.run; rAF callbacks already queued are canceled on failure.
    resumeRenderer = () => {
      if (!stopped && !sleeping && !raf) {
        last=0; raf=requestAnimationFrame(tick);
      }
    };
    document.body.dataset.renderer='neural';
    // Same as the rig path: no permanent label at rest (native pet-live2d.html
    // shows only #pet). Keep the shell's degradation text when it exists.
    if(!serverNotice){notice.textContent='';}
    resumeRenderer();

    } catch(error) {
      if(session){try{await session.release();}catch{}}
      throw error;
    }
  }
  async function startSprite() {
    let sheet=null;
    try{sheet=await image('sprite.webp');if(sheet.naturalWidth!==1536||sheet.naturalHeight!==208)sheet=null;}catch{}
    if(stopped)return;
    const head=sheet?null:await image('fallback.png');if(stopped)return;
    canvas.width=192;canvas.height=208;let frame=0;
    const pose=new Float32Array(45);
    function paint(now) {
      const input=stateInput(pose), breath=Math.sin(now/1000*2*Math.PI/3.6);
      ctx.clearRect(0,0,192,208);ctx.save();
      ctx.translate(96,204);ctx.rotate(pose[41]*.035+pose[39]*.02);
      ctx.scale(1-.01*breath,1+.016*breath);ctx.translate(-96,-204-pose[42]*5);
      if(sheet)ctx.drawImage(sheet,frame*192,0,192,208,0,0,192,208);
      else ctx.drawImage(head,36,40+pose[38]*2.5,120,120);
      ctx.restore();canvas.hidden=false;fallback.hidden=true;
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
      if(engine){try{await engine.close();}catch{}engine=null;}
      while(!stopped&&nextIndex<candidates.length) {
        try{await candidates[nextIndex++]();return;}
        catch {if(engine){try{await engine.close();}catch{}engine=null;}}
      }
      if(!stopped){canvas.hidden=true;fallback.hidden=false;notice.textContent='绘制资源不可用 / Renderer unavailable';}
    } finally {changing=false;}
  }
  await nextRenderer();
})();
