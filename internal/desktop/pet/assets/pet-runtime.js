'use strict';
// Fixed-origin presentation only; no application bridge or user package code.
(async function() {
  const fallback = document.getElementById('fallback');
  const canvas = document.getElementById('sprite'), ctx = canvas.getContext('2d');
  const notice = document.getElementById('render-notice');
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
      character.style.setProperty('--app-region', settings.lockPosition || (settings.shiftToDrag && !shiftHeld) ? 'no-drag' : 'drag');
    }
  }
  function modifier(event) { const next = !!event.shiftKey; if (shiftHeld !== next) {shiftHeld = next; applySettings();} }
  document.addEventListener('pointermove', modifier);
  document.addEventListener('pointerenter', modifier, true);
  document.addEventListener('keydown', modifier);
  document.addEventListener('keyup', modifier);
  window.addEventListener('blur', () => {shiftHeld = false; applySettings();});
  document.addEventListener('click', event => {
    if (!settings.clickSound || !event.target.closest('a[href]')) return;
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
  });
  applySettings(settings);
  let stopped = false, engine = null, raf = 0, spriteTimer = 0, taskTimer = 0;
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
  let sleeping = document.body.dataset.sleeping === 'true', resumeRenderer = null;
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
    if (clickAudio) void clickAudio.close().catch(() => {});
    cancelAnimationFrame(raf); clearInterval(spriteTimer); clearTimeout(taskTimer); clearInterval(contactTimer);
    if (engine) void engine.close().catch(() => {});
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
  function spriteFallback() {
    const sheet = new Image(); let frame = 0;
    const paint = () => { ctx.clearRect(0,0,192,208); ctx.drawImage(sheet,frame*192,0,192,208,0,0,192,208); };
    function start() {
      if (stopped || sleeping || document.hidden || spriteTimer || !sheet.naturalWidth) return;
      spriteTimer=setTimeout(() => {
        spriteTimer=0;
        if (stopped || sleeping || document.hidden) return;
        frame=(frame+1)%8;paint();start();
      },behaviorController?.powerSaving(performance.now()) ? 240 : 120);
    }
    sheet.onload = () => {
      if (stopped) return;
      canvas.width=192; canvas.height=208;
      paint();canvas.hidden=false;fallback.hidden=true;start();
    };
    document.addEventListener('pointermove',start);
    document.addEventListener('visibilitychange',()=>{if(!document.hidden)start();});
    resumeRenderer = start;
    sheet.src='sprite.webp';
  }
  if (document.body.dataset.neural !== 'true') { spriteFallback(); return; }
  function script(src) { return new Promise((resolve,reject) => { const tag=document.createElement('script');tag.src=src;tag.onload=resolve;tag.onerror=()=>reject(new Error('runtime unavailable'));document.head.appendChild(tag); }); }
  let session = null;
  try {
    notice.textContent='正在载入 THA4… / Loading THA4…';
    await script('tha4.js'); await script('render/ort/ort.all.min.js');
    if (stopped) return;
    ort.env.wasm.wasmPaths=new URL('render/ort/', location.href).href;
    ort.env.wasm.numThreads=1; ort.env.wasm.proxy=false; ort.env.logLevel='warning';
    let provider='wasm';
    for (const ep of ['webnn','webgpu','wasm']) {
      try { session=await ort.InferenceSession.create(new URL('render/avatar/model.onnx', location.href).href,{executionProviders:[ep]}); provider=ep; break; }
      catch { if (stopped) return; }
    }
    if (!session) throw new Error('No supported inference provider');
    if (stopped) { await session.release(); return; }
    const img = new Image();
    await new Promise((resolve,reject) => {img.onload=resolve;img.onerror=()=>reject(new Error('character unavailable'));img.src='render/avatar/character.png';});
    if (img.naturalWidth !== 512 || img.naturalHeight !== 512) throw new Error('character shape');
    if (stopped) { await session.release(); return; }
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
        const t=now/1000;
        pose.fill(0); pose[44]=0.4*Math.sin(t*1.5);pose[42]=0.35*Math.sin(t*0.7);pose[43]=0.3*Math.sin(t*0.6);
        pose[39]=0.18*Math.sin(t*1.1);pose[40]=0.14*Math.sin(t*1.3);pose[41]=0.08*Math.sin(t*0.55);
        const blink=t%4;pose[12]=pose[13]=blink<0.24?Math.sin(blink/0.24*Math.PI):0;
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
        if (!stopped) {canvas.hidden=true;fallback.hidden=false;notice.textContent='THA4 运行失败，已回退 / Renderer failed; fallback';spriteFallback();}
      } finally { busy=false; }
    }
    // No concurrent session.run; rAF callbacks already queued are canceled on failure.
    resumeRenderer = () => {
      if (!stopped && !sleeping && !raf) {
        last=0; raf=requestAnimationFrame(tick);
      }
    };
    resumeRenderer();
  } catch {
    if (session) {try {await session.release();}catch{}}
    if (engine) {try {await engine.close();}catch{} engine=null;}
    if (!stopped) { notice.textContent='THA4 资源或运行时不可用，已回退 / THA4 unavailable; fallback';spriteFallback(); }
  }
})();
