// Real installed InspectorChatRevealer / Chromium trajectory evidence; no bundle writes.
// Launch the installed app separately with the exact .devdata-ref-r100 profile, then:
// node scripts/inspector-reference-smooth-scroll.mjs 19444 http://127.0.0.1:3083/ <installed-client.js> docs/review-20261022/evidence/inspector-reference-pass251-smooth <owned-main-pid>
import {readFileSync, writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {execFileSync} from 'node:child_process';
import path from 'node:path';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const PROFILE = ROOT + '\\.devdata-ref-r100';
const EXECUTABLE = 'C:\\软件\\Whale Isle\\Whale Isle.exe';
const BUNDLE = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness\\packages\\experimental\\session-inspector\\lib\\client.js';
const NODE = 'C:\\Users\\48818\\AppData\\Roaming\\Deepseek-Harness-Desktop\\dsh-home\\dsh-runtimes\\dsh-primary-runtime\\dependencies\\node\\bin\\node.exe';
const [port, pageUrl, bundle, output, ownedPidArg] = process.argv.slice(2);
const ownedPid = Number(ownedPidArg);
const samePath = (a, b) => path.resolve(a).toLowerCase() === path.resolve(b).toLowerCase();
if (port !== '19444' || pageUrl !== 'http://127.0.0.1:3083/' || !samePath(bundle || '.', BUNDLE)) throw Error('Use the exact installed bundle and isolated 3083/19444 reference only');
if (!Number.isSafeInteger(ownedPid) || ownedPid < 1 || [20012, 16008].includes(ownedPid)) throw Error('Specify the independently launched main PID');
if (!samePath(process.cwd(), ROOT) || !samePath(process.execPath, NODE)) throw Error('Use the requested workspace and Node executable');
const outputPath = path.resolve(output || '.');
const allowedDir = path.join(ROOT, 'docs', 'review-20261022', 'evidence');
if (!samePath(path.dirname(outputPath), allowedDir) || !/^inspector-reference-pass251-smooth[\w-]*$/.test(path.basename(outputPath))) throw Error('Evidence output is outside the requested prefix');

function ownershipSnapshot() {
  // This command reads identities/listeners only. It does not signal any process.
  const command = `[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false); $OutputEncoding = [Console]::OutputEncoding; $all = @(Get-CimInstance Win32_Process); $ids = @(${ownedPid}); do { $more = @($all | Where-Object { $_.ParentProcessId -in $ids -and $_.ProcessId -notin $ids } | Select-Object -ExpandProperty ProcessId); $ids += $more } while ($more.Count); $ports = @(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue | Where-Object { $_.LocalPort -in 3080,3083,19444 } | Select-Object LocalAddress,LocalPort,OwningProcess); [pscustomobject]@{ ports=$ports; main=($all | Where-Object ProcessId -eq ${ownedPid} | Select-Object ProcessId,ParentProcessId,ExecutablePath,CommandLine); owned=@($all | Where-Object { $_.ProcessId -in $ids } | Select-Object ProcessId,ParentProcessId,ExecutablePath,CommandLine); protected=@($all | Where-Object { $_.ProcessId -in 20012,16008 } | Select-Object ProcessId,ParentProcessId,ExecutablePath) } | ConvertTo-Json -Depth 6 -Compress`;
  const snapshot = JSON.parse(execFileSync('powershell.exe', ['-NoLogo', '-NoProfile', '-NonInteractive', '-Command', command], {encoding: 'utf8', windowsHide: true, timeout: 10000}).replace(/^\uFEFF/, ''));
  if (!snapshot.main || !samePath(snapshot.main.ExecutablePath, EXECUTABLE) || !snapshot.main.CommandLine.includes(`--user-data-dir="${PROFILE}"`) || !snapshot.main.CommandLine.includes('--remote-debugging-port=19444')) throw Error('Owned main process identity/profile/debugger mismatch');
  const listening = p => snapshot.ports.filter(x => x.LocalPort === p);
  const ownedIds = new Set(snapshot.owned.map(x => x.ProcessId));
  if (listening(19444).length !== 1 || listening(19444)[0].OwningProcess !== ownedPid || listening(3083).length !== 1 || !ownedIds.has(listening(3083)[0].OwningProcess)) throw Error('Isolated listener is not owned by the specified tree');
  const backend = snapshot.owned.find(x => x.ProcessId === listening(3083)[0].OwningProcess);
  if (!backend.CommandLine.includes(PROFILE + '\\dsh-home\\profiles\\web\\') || !backend.CommandLine.includes('--port 3083')) throw Error('Backend profile/3083 command mismatch');
  if (listening(3080).length !== 1 || listening(3080)[0].OwningProcess !== 16008 || !snapshot.protected.some(x => x.ProcessId === 16008 && x.ParentProcessId === 20012)) throw Error('Protected original identities/listener changed; stop the probe');
  const config = JSON.parse(readFileSync(path.join(PROFILE, 'config.json'), 'utf8'));
  if (config.port !== 3083 || config.workspace !== ROOT + '\\.devdata-clean\\workspace') throw Error('Reference profile configuration mismatch');
  return snapshot;
}

const beforeOwnership = ownershipSnapshot();
const original = readFileSync(bundle, 'utf8');
const bundleHash = createHash('sha256').update(original).digest('hex');
const registration = 'window.__ModuleLoader__.load({';
if (!original.includes(registration)) throw Error('Installed module registration missing');
let inMemory = original.replace(registration, 'window.__smoothRegistration=({');
const factoryEnd = inMemory.lastIndexOf('return module.exports;');
if (factoryEnd < 0) throw Error('Installed factory return missing');
inMemory = inMemory.slice(0, factoryEnd) + 'module.exports.__smoothFixture={InspectorChatRevealer,visibleChatRect};' + inMemory.slice(factoryEnd);

const fetchJson = async suffix => {
  const response = await fetch(`http://127.0.0.1:${port}${suffix}`, {signal: AbortSignal.timeout(4000)});
  if (!response.ok) throw Error('Debugger endpoint failed: ' + response.status);
  return response.json();
};
const browser = await fetchJson('/json/version');
const pages = await fetchJson('/json/list');
const matches = pages.filter(p => p.type === 'page' && p.url === pageUrl);
if (matches.length !== 1) throw Error('Expected one exact owned 3083 page: ' + JSON.stringify(pages.map(p => ({id: p.id, url: p.url}))));
const page = matches[0];
const ws = new WebSocket(page.webSocketDebuggerUrl);
let sequence = 0;
const pending = new Map();
await new Promise((resolve, reject) => {
  const timeout = setTimeout(() => reject(Error('CDP connect timeout')), 5000);
  ws.addEventListener('open', () => { clearTimeout(timeout); resolve(); }, {once: true});
  ws.addEventListener('error', e => { clearTimeout(timeout); reject(e); }, {once: true});
});
ws.addEventListener('message', e => {
  const message = JSON.parse(e.data), request = pending.get(message.id);
  if (!request) return;
  pending.delete(message.id);
  clearTimeout(request.timeout);
  message.error ? request.reject(Error(JSON.stringify(message.error))) : request.resolve(message.result);
});
const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = ++sequence;
  const timeout = setTimeout(() => { pending.delete(id); reject(Error('CDP timeout: ' + method)); }, 12000);
  pending.set(id, {resolve, reject, timeout});
  ws.send(JSON.stringify({id, method, params}));
});
const evaluate = async expression => {
  const response = await send('Runtime.evaluate', {expression, awaitPromise: true, returnByValue: true});
  if (response.exceptionDetails) throw Error(JSON.stringify(response.exceptionDetails));
  return response.result.value;
};

// Executed in the one exact reference target. These wrappers only log and delegate
// to the original native DOM methods. No scroll timeline/clock is mocked.
async function installFixture(source, expectedUrl) {
  if (location.href !== expectedUrl) throw Error('Page URL changed');
  window.__r107Smooth?.dispose();
  (0, eval)(source);
  const registration = window.__smoothRegistration, modules = window.__DSH_MODULES__;
  const exposed = registration.factory(modules.makeRequire(registration.id, new Set())).__smoothFixture;
  const host = document.createElement('div');
  host.id = 'r107-smooth';
  host.style.cssText = 'position:fixed;inset:0;z-index:2147483500;background:white;color:#1d2535;font:12px sans-serif;--dsw-specific-bubble-highlight:rgb(211,226,255);--dsw-alias-state-business-primary:rgb(65,118,230)';
  host.innerHTML = '<div style="height:40px;box-sizing:border-box">Reference smooth-scroll probe</div><div data-conversation-scroll tabindex="-1" style="height:260px;overflow-y:auto;overflow-x:hidden;position:relative;scroll-behavior:auto;scroll-snap-type:none;overscroll-behavior:contain;box-sizing:border-box"><div data-chat-flow style="display:block;padding:0;margin:0"></div></div><div data-composer-seat style="height:60px;box-sizing:border-box">60 DIP composer seat</div>';
  document.body.append(host);
  const scroller = host.querySelector('[data-conversation-scroll]'), flow = host.querySelector('[data-chat-flow]');
  for (let index = 0; index < 40; index++) {
    const row = document.createElement('div');
    row.dataset.chatNodeKey = String(index);
    row.style.cssText = 'display:block;height:60px;min-height:60px;max-height:60px;margin:0 20px;padding:0;box-sizing:border-box;border-bottom:1px solid #d5dae1';
    row.textContent = 'Chat row ' + index;
    flow.append(row);
  }
  const revealer = new exposed.InspectorChatRevealer(() => flow.isConnected ? flow : undefined);
  const nativeIntoView = Element.prototype.scrollIntoView, nativeTo = Element.prototype.scrollTo;
  let active;
  const log = (kind, fields = {}) => {
    if (active) active.events.push({kind, now: performance.now(), scrollTop: scroller.scrollTop, ...fields});
  };
  for (const row of flow.children) row.scrollIntoView = function(options) {
    log('native-scrollIntoView', {nodeKey: this.dataset.chatNodeKey, options});
    return nativeIntoView.call(this, options);
  };
  scroller.scrollTo = function(...args) {
    log('native-scrollTo', {args});
    return nativeTo.apply(this, args);
  };
  const frames = async count => { for (let i = 0; i < count; i++) await new Promise(requestAnimationFrame); };
  const overlay = () => document.querySelector('[data-session-inspector-highlight="reveal"]');
  const rect = element => { const r = element.getBoundingClientRect(); return {x:r.x, y:r.y, width:r.width, height:r.height}; };
  const dispatch = input => {
    if (input.type === 'wheel') return window.dispatchEvent(new WheelEvent('wheel', {deltaY:120, bubbles:true, cancelable:true}));
    if (input.type === 'pointerdown') return window.dispatchEvent(new PointerEvent('pointerdown', {pointerType:'mouse', button:0, bubbles:true, cancelable:true}));
    if (input.type === 'keydown') return window.dispatchEvent(new KeyboardEvent('keydown', {key:input.key, code:input.code || input.key, bubbles:true, cancelable:true}));
    return window.dispatchEvent(new MouseEvent('click', {button:0, bubbles:true, cancelable:true}));
  };
  window.__r107Smooth = {
    geometry: {viewport: rect(scroller), composer: rect(host.querySelector('[data-composer-seat]')), rows: flow.children.length, rowHeight: flow.children[0].getBoundingClientRect().height, contentHeight:scroller.scrollHeight, maxOffset:scroller.scrollHeight-scroller.clientHeight, devicePixelRatio, visibility:document.visibilityState, reducedMotion:matchMedia('(prefers-reduced-motion: reduce)').matches},
    async run(spec) {
      if (location.href !== expectedUrl) throw Error('Page URL changed during probe');
      revealer.dispose();
      scroller.scrollTo({top:spec.start, left:0, behavior:'instant'});
      await frames(5); // Let the prior instant reset's scroll/scrollend drain before tracing.
      const record = {spec, startedAt:performance.now(), startOffset:scroller.scrollTop, frames:[], events:[], firstMovement:null, overlayFirstOn:null, overlayFirstOff:null, interruption:null, trigger:null};
      active = record;
      const onScroll = event => {
        if (event.target !== scroller) return;
        log(event.type, {isTrusted:event.isTrusted, target:'fixture-scroller'});
      };
      const onInput = event => log('window-input', {type:event.type, key:event.key || null, isTrusted:event.isTrusted, targetIsWindow:event.target === window});
      window.addEventListener('scroll', onScroll, true);
      window.addEventListener('scrollend', onScroll, true);
      for (const type of ['click','wheel','pointerdown','keydown']) window.addEventListener(type, onInput, true);
      let hadOverlay = false;
      const observeOverlay = () => {
        const element = overlay();
        if (element && !hadOverlay) {
          hadOverlay = true;
          const animation = element.getAnimations()[0];
          record.overlayFirstOn = {now:performance.now(), scrollTop:scroller.scrollTop, rect:rect(element), duration:animation?.effect.getTiming().duration, animationStartTime:animation?.startTime};
          log('overlay-on', record.overlayFirstOn);
        } else if (!element && hadOverlay && record.overlayFirstOff === null) {
          record.overlayFirstOff = {now:performance.now(), scrollTop:scroller.scrollTop};
          log('overlay-off', record.overlayFirstOff);
        }
      };
      const observer = new MutationObserver(observeOverlay);
      observer.observe(document.body, {childList:true});
      let frame, hardTimeout;
      return new Promise(resolve => {
        const finish = error => {
          cancelAnimationFrame(frame);
          clearTimeout(hardTimeout);
          observeOverlay();
          observer.disconnect();
          window.removeEventListener('scroll', onScroll, true);
          window.removeEventListener('scrollend', onScroll, true);
          for (const type of ['click','wheel','pointerdown','keydown']) window.removeEventListener(type, onInput, true);
          record.finishedAt = performance.now();
          record.final = {scrollTop:scroller.scrollTop, overlay:!!overlay(), pending:typeof revealer.cancelScroll === 'function', framePending:revealer.frame !== undefined};
          if (error) record.error = error;
          active = undefined;
          revealer.dispose();
          resolve(record);
        };
        const tick = timestamp => {
          observeOverlay();
          const now = performance.now(), offset = scroller.scrollTop;
          const sample = {raf:timestamp, now, timeline:document.timeline.currentTime, scrollTop:offset, overlay:!!overlay(), pending:typeof revealer.cancelScroll === 'function'};
          record.frames.push(sample);
          if (record.firstMovement === null && Math.abs(offset - spec.start) > 0.25) record.firstMovement = {...sample};
          if (spec.input && record.firstMovement && !record.interruption) {
            const progress = Math.abs((offset-spec.start)/(spec.target-spec.start));
            const interruption = {type:spec.input.type, key:spec.input.key || null, now:performance.now(), raf:timestamp, beforeOffset:scroller.scrollTop, pendingBefore:typeof revealer.cancelScroll === 'function', overlayBefore:!!overlay(), progress};
            dispatch(spec.input); // Synthetic window event has no browser scrolling default.
            interruption.afterOffset = scroller.scrollTop;
            interruption.pendingAfter = typeof revealer.cancelScroll === 'function';
            interruption.overlayAfter = !!overlay();
            record.interruption = interruption;
            log('input-dispatched', interruption);
          }
          if (spec.mode === 'idle-click' && now-record.startedAt >= 350) return finish();
          if (spec.cancel && record.interruption && now-record.interruption.now >= 1250) return finish();
          if (!spec.cancel && spec.mode !== 'idle-click' && record.overlayFirstOff && now-record.overlayFirstOff.now >= 100) return finish();
          if (now-record.startedAt >= 4200) return finish('4200ms trace deadline reached');
          frame = requestAnimationFrame(tick);
        };
        hardTimeout = setTimeout(() => finish('5500ms wall deadline reached (rAF unavailable)'), 5500);
        frame = requestAnimationFrame(tick);
        if (spec.mode !== 'idle-click') window.addEventListener('click', () => revealer.reveal([{nodeKey:String(spec.row)}]), {capture:true, once:true});
        const beforeOffset = scroller.scrollTop, beforeNow = performance.now();
        dispatch({type:'click'});
        record.trigger = {method:'window.dispatchEvent(new MouseEvent("click"))', beforeNow, afterNow:performance.now(), beforeOffset, afterOffset:scroller.scrollTop, revealListener:spec.mode !== 'idle-click'};
      });
    },
    dispose() { revealer.dispose(); host.remove(); delete window.__r107Smooth; delete window.__smoothRegistration; }
  };
  return window.__r107Smooth.geometry;
}

const CHROMIUM_TAG = '152.0.7977.78';
const sourceBase = `https://raw.githubusercontent.com/chromium/chromium/${CHROMIUM_TAG}/`;
const chromiumSources = [
  {url:sourceBase+'third_party/blink/renderer/core/scroll/programmatic_scroll_animator.cc', evidence:'ProgrammaticScrollAnimator::AnimateToOffset creates kProgrammatic; main-thread TickAnimation evaluates monotonic_time - start_time_.'},
  {url:sourceBase+'cc/animation/scroll_offset_animation_curve_factory.cc', evidence:'GetDurationBehaviorFromScrollType(kProgrammatic) returns kDeltaBased.'},
  {url:sourceBase+'cc/animation/scroll_offset_animation_curve.cc', evidence:'EaseInOutSegmentDuration: min(sqrt(abs(maximum delta)), maxDurationSeconds*60)/60 seconds; GetValue uses timing_function_->GetValue(t/duration).'},
  {url:sourceBase+'cc/base/features.cc', evidence:'Desktop defaults PROGRAMMATIC_SCROLL_ANIMATION_CURVE(0.4,0.0,0.0,1.0,1500); Android defaults differ.'}
];
const result = {
  schema:'inspector-reference-smooth-scroll/v1', capturedAt:new Date().toISOString(),
  method:'Installed factory exported only in page memory; original DOM methods delegated without clock/timeline overrides; native Chromium smooth scroll and WAAPI highlight run in real time.',
  scope:'Synthetic loaded Chat boxes only; no Host/model callback, conversation action, production/test edit, or bundle write.',
  browser, target:{id:page.id, url:page.url, debuggerPort:Number(port)}, ownership:{before:beforeOwnership},
  installedBundle:{path:bundle, sha256:bundleHash, sourcePointers:{reveal:'3624-3656', cancellation:'3684-3769', nativeCall:'3765-3769', overlay:'3771-3793'}},
  timeBasis:'Every rAF records its browser high-resolution timestamp, performance.now(), document.timeline.currentTime, and scrollTop. Curve comparison begins with first nonzero movement, fixes duration using captured physical/CSS viewport ratio, and aligns only origin within two observed frame gaps; no frame-exact duration/offset assertions.',
  chromiumSources, cases:[], checks:[], errors:[]
};

function bezier(t, p1, p2) { const u = 1-t; return 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t; }
function easing(progress, points = [0.4,0,0,1]) {
  if (progress <= 0) return 0;
  if (progress >= 1) return 1;
  let low = 0, high = 1;
  for (let i = 0; i < 45; i++) { const mid = (low+high)/2; if (bezier(mid, points[0], points[2]) < progress) low = mid; else high = mid; }
  return bezier((low+high)/2, points[1], points[3]);
}
function inverseEasing(value) {
  let low = 0, high = 1;
  for (let i = 0; i < 45; i++) { const mid = (low+high)/2; if (easing(mid) < value) low = mid; else high = mid; }
  return (low+high)/2;
}
function summarize(record) {
  const {spec} = record;
  const first = record.firstMovement;
  const moved = record.frames.filter(f => Math.abs(f.scrollTop-spec.start)>0.25);
  const lastMovingIndex = record.frames.findLastIndex((f,i,all) => i>0 && Math.abs(f.scrollTop-all[i-1].scrollTop)>0.25);
  const lastMovement = record.frames[lastMovingIndex];
  const ends = record.events.filter(e => e.kind === 'scrollend');
  const afterInput = record.interruption ? record.frames.filter(f => f.now >= record.interruption.now) : [];
  const offsetSpan = frames => frames.length ? Math.max(...frames.map(f=>f.scrollTop))-Math.min(...frames.map(f=>f.scrollTop)) : 0;
  const summary = {
    name:spec.name, start:spec.start, target:spec.target ?? null, distance:spec.target === undefined ? 0 : Math.abs(spec.target-spec.start), final:record.final.scrollTop,
    frames:record.frames.length, firstMovementAfterTriggerMs:first ? first.now-record.trigger.beforeNow : null,
    observedFirstToFinalMovementMs:first && lastMovement ? lastMovement.raf-first.raf : null,
    firstScrollendAfterMovementMs:first && ends[0] ? ends[0].now-first.now : null,
    scrollendEvents:ends.length, overlayFirstOnAfterMovementMs:first && record.overlayFirstOn ? record.overlayFirstOn.now-first.now : null,
    overlayFirstOnAfterScrollendMs:ends[0] && record.overlayFirstOn ? record.overlayFirstOn.now-ends[0].now : null,
    observedOverlayLifetimeMs:record.overlayFirstOn && record.overlayFirstOff ? record.overlayFirstOff.now-record.overlayFirstOn.now : null,
    maxRafGapMs:record.frames.length > 1 ? Math.max(...record.frames.slice(1).map((f,i)=>f.raf-record.frames[i].raf)) : null,
    offsetSpan:offsetSpan(record.frames), afterInputOffsetSpan:offsetSpan(afterInput), afterInputObservationMs:record.interruption ? record.finishedAt-record.interruption.now : null,
    triggerImmediateOffsetDelta:record.trigger.afterOffset-record.trigger.beforeOffset,
    anyOverlayBeforeInput:!!record.interruption?.overlayBefore, anyOverlayAfterInput:afterInput.some(f=>f.overlay)
  };
  if (first && !spec.cancel) {
    const delta = spec.target-spec.start;
    const physicalScale = result.browserViewportMetrics.layoutViewport.clientHeight/result.browserViewportMetrics.cssLayoutViewport.clientHeight;
    const durationMs = Math.min(Math.sqrt(Math.abs(delta)*physicalScale)/60, 1.5)*1000;
    const cssOnlyDurationMs = Math.min(Math.sqrt(Math.abs(delta))/60, 1.5)*1000;
    const initialProgress = (first.scrollTop-spec.start)/delta;
    const firstPhase = inverseEasing(initialProgress);
    const firstPhaseMs = firstPhase*durationMs;
    const firstSampleOriginRaf = first.raf-firstPhaseMs;
    const compared = moved.filter(f=>f.raf>=first.raf && f.raf<=firstSampleOriginRaf+durationMs+40);
    const score = (origin, duration, points=[0.4,0,0,1]) => {
      const errors=compared.map(f=>Math.abs(f.scrollTop-(spec.start+delta*easing((f.raf-origin)/duration,points))));
      return {rmseDip:Math.sqrt(errors.reduce((n,e)=>n+e*e,0)/errors.length), maxAbsErrorDip:Math.max(...errors)};
    };
    // The first nonzero offset is physically quantized (2/3 DIP in this 1.5x
    // Electron window), so its inverse phase has uncertainty. Refine only the
    // origin in a bounded +/-two-observed-frame interval; never fit duration.
    const originWindowMs=Math.min(40,Math.max(12,summary.maxRafGapMs*2));
    let alignedOrigin=firstSampleOriginRaf, best=score(alignedOrigin,durationMs);
    for(let shift=-originWindowMs;shift<=originWindowMs;shift+=0.1) {
      const candidate=score(firstSampleOriginRaf+shift,durationMs);
      if(candidate.rmseDip<best.rmseDip) {best=candidate;alignedOrigin=firstSampleOriginRaf+shift;}
    }
    summary.sourceCurveComparison = {
      basis:'browser rAF time from first nonzero movement; duration fixed from source and captured physical/CSS viewport ratio; only origin aligned within two observed frame gaps for first-offset quantization',
      cubicBezier:[0.4,0,0,1], physicalDeltaScale:physicalScale, nominalDurationMs:durationMs, cssOnlyNominalDurationMs:cssOnlyDurationMs,
      firstNonzeroPhaseMs:firstPhaseMs, firstSampleOriginRaf, inferredOriginRaf:alignedOrigin, originAlignmentMs:alignedOrigin-firstSampleOriginRaf,
      originSearchBoundMs:originWindowMs, samples:compared.length,
      firstSampleAnchorScore:score(firstSampleOriginRaf,durationMs), cssOnlyAnchorScore:score(first.raf-firstPhase*cssOnlyDurationMs,cssOnlyDurationMs),
      ...best, legacyEaseInOutRmseDip:score(alignedOrigin,durationMs,[0.42,0,0.58,1]).rmseDip
    };
  }
  return summary;
}
const escaped = text => String(text).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;');
function chart(records, cancellation) {
  const width=1040, panelHeight=230, height=55+records.length*panelHeight;
  const elements=[`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><rect width="100%" height="100%" fill="#fff"/><style>text{font:13px sans-serif;fill:#192638}.muted{fill:#596579}.title{font-size:18px;font-weight:600}</style><text x="24" y="29" class="title">${cancellation?'Early window-dispatched input during native smooth scroll':'Native Chromium smooth-scroll trajectories (DIP)'}</text>`];
  records.forEach((record,index)=>{
    const y=55+index*panelHeight, left=80, right=1000, top=y+28, bottom=y+170;
    const first=record.firstMovement, zero=first?.raf ?? record.frames[0].raf;
    const duration=cancellation?Math.max(1300,record.frames.at(-1).raf-zero):Math.max(record.summary.sourceCurveComparison?.nominalDurationMs||0, record.summary.observedFirstToFinalMovementMs||0)+100;
    const min=Math.min(record.spec.start,record.spec.target??record.spec.start), max=Math.max(record.spec.start,record.spec.target??record.spec.start);
    const span=Math.max(10,max-min);
    const x=ms=>left+Math.max(0,ms)/duration*(right-left), yy=v=>bottom-(v-min)/span*(bottom-top);
    elements.push(`<text x="24" y="${y+13}" font-weight="600">${escaped(record.spec.name)}: ${record.spec.start} → ${record.spec.target??record.spec.start}; final ${record.final.scrollTop}</text><path d="M${left},${top}V${bottom}H${right}" fill="none" stroke="#b7c2d2"/>`);
    for(const t of [0,duration/2,duration]) elements.push(`<text x="${x(t)}" y="${bottom+20}" text-anchor="middle" class="muted">${Math.round(t)} ms</text>`);
    for(const v of [min,max]) elements.push(`<text x="${left-8}" y="${yy(v)+4}" text-anchor="end" class="muted">${v}</text>`);
    if (!cancellation && record.summary.sourceCurveComparison) {
      const curve=record.summary.sourceCurveComparison, delta=record.spec.target-record.spec.start;
      const alignedPhaseMs=first.raf-curve.inferredOriginRaf;
      const points=Array.from({length:121},(_,i)=>{const ms=i*duration/120; return `${x(ms).toFixed(2)},${yy(record.spec.start+delta*easing((ms+alignedPhaseMs)/curve.nominalDurationMs)).toFixed(2)}`;}).join(' ');
      elements.push(`<polyline points="${points}" fill="none" stroke="#b1bac8" stroke-width="4" stroke-dasharray="5 4"/>`);
    }
    const trace=record.frames.filter(f=>f.raf>=zero && f.raf-zero<=duration).map(f=>`${x(f.raf-zero).toFixed(2)},${yy(f.scrollTop).toFixed(2)}`).join(' ');
    elements.push(`<polyline points="${trace}" fill="none" stroke="#2167b4" stroke-width="2"/>`);
    for(const event of record.events.filter(e=>e.kind==='scrollend')) if(event.now-zero<=duration) elements.push(`<line x1="${x(event.now-zero)}" x2="${x(event.now-zero)}" y1="${top}" y2="${bottom}" stroke="#27835a" stroke-dasharray="3 3"/>`);
    if(record.overlayFirstOn && record.overlayFirstOn.now-zero<=duration) elements.push(`<line x1="${x(record.overlayFirstOn.now-zero)}" x2="${x(record.overlayFirstOn.now-zero)}" y1="${top}" y2="${bottom}" stroke="#9455a6"/>`);
    if(record.interruption) elements.push(`<circle cx="${x(record.interruption.raf-zero)}" cy="${yy(record.interruption.afterOffset)}" r="5" fill="#ce7330"/>`);
    const foot=cancellation?`Input ${record.interruption?.type} ${record.interruption?.key||''}; pending ${record.interruption?.pendingBefore} → ${record.interruption?.pendingAfter}; post-input span ${record.summary.afterInputOffsetSpan} DIP; observed ${Math.round(record.summary.afterInputObservationMs)} ms`:`Blue: rAF offset · dashed gray: source curve · green: scrollend · purple: overlay-on · curve RMSE ${record.summary.sourceCurveComparison.rmseDip.toFixed(2)} DIP`;
    elements.push(`<text x="80" y="${bottom+43}" class="muted">${escaped(foot)}</text>`);
  });
  return elements.join('')+'</svg>\n';
}
function reportMarkdown() {
  const trajectory=result.cases.filter(c=>c.summary.sourceCurveComparison && !c.spec.input);
  const inputs=result.cases.filter(c=>c.spec.input);
  const r=n=>n===null||n===undefined?'—':Number(n).toFixed(2);
  const lines=[
    '# Installed InspectorChatRevealer smooth-scroll evidence', '',
    `Captured ${result.capturedAt}. Result: **${result.errors.length?'FAIL':'PASS'}**. Chromium ${browser.Browser}; Electron 44.3.0.`, '',
    'The installed factory was read from disk, exported only from an in-memory copy, and run in the exact owned reference page. Chat geometry: 40 rows × 60 DIP; scroll viewport 260 DIP high at Y=40; 60 DIP composer at Y=300. The original bundle hash was unchanged. All reveal triggers and interruption events were synthetic window dispatches; no browser-default wheel/key scrolling or Host/model action contributed to the offsets.', '',
    '## Trajectories', '',
    '| Case | Offset | First movement → last movement (ms) | First movement → scrollend (ms) | Scrollend → overlay (ms) | Source nominal (ms) | Source curve RMSE / max (DIP) |',
    '|---|---:|---:|---:|---:|---:|---:|',
    ...trajectory.map(c=>{const s=c.summary,q=s.sourceCurveComparison;return `| ${c.spec.name} | ${s.start} → ${s.final} | ${r(s.observedFirstToFinalMovementMs)} | ${r(s.firstScrollendAfterMovementMs)} | ${r(s.overlayFirstOnAfterScrollendMs)} | ${r(q.nominalDurationMs)} | ${r(q.rmseDip)} / ${r(q.maxAbsErrorDip)} |`;}), '',
    `Captured Page.getLayoutMetrics reports native layout viewport ${result.browserViewportMetrics?.layoutViewport.clientWidth} × ${result.browserViewportMetrics?.layoutViewport.clientHeight}, versus CSS viewport ${result.browserViewportMetrics?.cssLayoutViewport.clientWidth} × ${result.browserViewportMetrics?.cssLayoutViewport.clientHeight}: physical/CSS scale ${result.browserViewportMetrics ? result.browserViewportMetrics.layoutViewport.clientHeight/result.browserViewportMetrics.cssLayoutViewport.clientHeight : 'unavailable'}. Electron was launched with native --device-scale-factor=1.5. CDP DPR emulation changes the web-exposed DPR to about 1 but leaves this native viewport ratio at 1.5. The source delta must use that captured physical scale.`, '',
    'Every rAF includes its high-resolution browser timestamp, performance.now(), document.timeline.currentTime, scrollTop, pending state and overlay presence. The comparison starts at first nonzero movement and fixes duration from physical delta. Because the first web offset is quantized, only its time origin is aligned within two observed frame gaps (at most 40 ms); duration is never fitted. The JSON retains both the raw first-sample anchor score and CSS-only prediction error, as well as the bounded origin alignment. Durations/offsets are descriptive measurements, not frame-exact assertions. Scheduler gaps and physical-pixel quantization remain in the raw trace.', '',
    '![Native trajectories](./'+path.basename(outputPath)+'-trajectories.svg)', '',
    '## Interruption and controls', '',
    '| Input after first movement | Offset before → after → final | Pending before → after | Post-input offset span (DIP) | Observation after input (ms) | Overlay ever after input |',
    '|---|---:|---|---:|---:|---|',
    ...inputs.map(c=>{const i=c.interruption,s=c.summary;return `| ${i.type}${i.key?' '+JSON.stringify(i.key):''} | ${i.beforeOffset} → ${i.afterOffset} → ${c.final.scrollTop} | ${i.pendingBefore} → ${i.pendingAfter} | ${r(s.afterInputOffsetSpan)} | ${r(s.afterInputObservationMs)} | ${s.anyOverlayAfterInput} |`;}), '',
    'Cancellation events fired on the first nonzero movement rAF, while the installed revealer still had a pending scroll and before the overlay existed. The delegated native instant scrollTo call froze the offset at its current value. Cancellation cases remained sampled for at least 1250 ms; ordinary “a” and a second window click retained the pending reveal, completed the trajectory, flashed, and removed the overlay.', '',
    'The idle click control had no reveal listener, no scrollIntoView call, no movement, and no overlay. Every reveal-trigger click also left the offset unchanged immediately after window.dispatchEvent returned; movement began on subsequent native frames.', '',
    '![Cancellation traces](./'+path.basename(outputPath)+'-interruptions.svg)', '',
    '## Source pointers', '',
    `Installed bundle: \`${BUNDLE}\` (SHA-256 \`${bundleHash}\`). Lines 3684–3769 own native scrolling, scrollend/quiet completion and interruption; 3765–3769 call smooth/center/nearest; 3771–3793 own the 700 ms overlay.`, '',
    ...chromiumSources.map(s=>`- [${s.url.slice(sourceBase.length)}](${s.url}): ${s.evidence}`), '',
    'This browser version’s desktop defaults are cubic-bezier(0.4, 0, 0, 1) and min(sqrt(abs(physical delta))/60, 1.5) seconds. At the captured native 1.5 scale, 300/600/1300/1380 DIP predict 353.55/500.00/735.98/758.29 ms. The legacy ease-in-out curve is also scored in the JSON as a comparison; no feature/curve override was installed.', '',
    '## Ownership and reproduction', '',
    `Only debugger 19444 target ${page.id} at ${pageUrl} was used. Owned main PID ${ownedPid}; backend PID ${beforeOwnership.ports.find(p=>p.LocalPort===3083).OwningProcess}. Exact profile: \`${PROFILE}\`. Protected main PID 20012/backend PID 16008 on 3080 were only read for identity checks.`, '',
    'Launch command (after verifying ports 3083 and 19444 are free; capture the newly returned main PID):', '',
    '```powershell',
    'Remove-Item Env:ELECTRON_RUN_AS_NODE -ErrorAction SilentlyContinue',
    `$owned = Start-Process -FilePath '${EXECUTABLE}' -ArgumentList '--user-data-dir="${PROFILE}"','--remote-debugging-port=19444' -PassThru`,
    '```', '',
    'Exact probe command for this run:', '',
    '```powershell',
    `& '${NODE}' 'scripts/inspector-reference-smooth-scroll.mjs' '19444' '${pageUrl}' '${BUNDLE}' '${output.replaceAll('\\','/')}' '${ownedPid}'`,
    '```', '',
    `The probe does not launch or terminate the app. The outer bounded run closes the validated owned PID tree and records teardown in the [lifecycle evidence](./${path.basename(outputPath)}-lifecycle.json). Refer to that file for the final port and protected-original checks.`, '',
    `Checks: ${result.checks.length}; errors: ${result.errors.length}.`,
    ...result.errors.map(error=>'- '+error), ''
  ];
  return lines.join('\n');
}

try {
  await send('Page.bringToFront');
  await send('Emulation.setFocusEmulationEnabled', {enabled:true});
  await send('Emulation.setDeviceMetricsOverride', {width:360, height:360, deviceScaleFactor:1, mobile:false});
  await send('Emulation.setEmulatedMedia', {features:[{name:'prefers-reduced-motion', value:'no-preference'}]});
  result.browserViewportMetrics = await send('Page.getLayoutMetrics');
  result.geometry = await evaluate(`(${installFixture.toString()})(${JSON.stringify(inMemory)},${JSON.stringify(pageUrl)})`);
  if (result.geometry.viewport.y!==40 || result.geometry.viewport.height!==260 || result.geometry.composer.height!==60 || result.geometry.composer.y!==300 || result.geometry.rows!==40 || result.geometry.rowHeight!==60 || result.geometry.reducedMotion) throw Error('Real fixture geometry/media mismatch');
  const specs = [
    {name:'idle-window-click-no-scroll', mode:'idle-click', start:100},
    {name:'distance-300-row10', start:200, target:500, row:10},
    {name:'distance-600-row15', start:200, target:800, row:15},
    {name:'distance-1300-row25', start:100, target:1400, row:25},
    {name:'reverse-1380-row2', start:1400, target:20, row:2},
    {name:'cancel-wheel', start:100, target:1400, row:25, input:{type:'wheel'}, cancel:true},
    {name:'cancel-pointerdown', start:100, target:1400, row:25, input:{type:'pointerdown'}, cancel:true},
    {name:'cancel-ArrowDown', start:100, target:1400, row:25, input:{type:'keydown', key:'ArrowDown'}, cancel:true},
    {name:'cancel-PageDown', start:100, target:1400, row:25, input:{type:'keydown', key:'PageDown'}, cancel:true},
    {name:'cancel-Escape', start:100, target:1400, row:25, input:{type:'keydown', key:'Escape'}, cancel:true},
    {name:'ordinary-a-does-not-cancel', start:100, target:1400, row:25, input:{type:'keydown', key:'a', code:'KeyA'}},
    {name:'pending-window-click-does-not-cancel', start:100, target:1400, row:25, input:{type:'click'}}
  ];
  for (const spec of specs) {
    const record = await evaluate(`window.__r107Smooth.run(${JSON.stringify(spec)})`);
    record.summary = summarize(record);
    result.cases.push(record);
    const check = (condition, description) => { result.checks.push({case:spec.name, description, pass:!!condition}); if (!condition) result.errors.push(spec.name+': '+description); };
    check(!record.error, record.error || 'completed before bounded deadline');
    check(record.summary.triggerImmediateOffsetDelta===0, 'window-dispatched trigger click does not itself move offset');
    check(!record.final.overlay && !record.final.pending && !record.final.framePending, 'no lingering overlay or pending revealer work');
    const nativeCalls=record.events.filter(e=>e.kind==='native-scrollIntoView');
    if (spec.mode==='idle-click') {
      check(record.summary.offsetSpan===0 && !record.overlayFirstOn && nativeCalls.length===0, 'idle click has no scrolling default');
    } else {
      check(nativeCalls.length===1 && nativeCalls[0].options.behavior==='smooth' && nativeCalls[0].options.block==='center' && nativeCalls[0].options.inline==='nearest', 'installed code invoked real native smooth/center/nearest exactly once');
      check(!!record.firstMovement, 'native movement observed');
      if (spec.input) check(record.interruption?.pendingBefore && !record.interruption.overlayBefore && record.interruption.progress>0 && record.interruption.progress<0.15, 'input dispatched early while pending and before overlay');
      if (spec.cancel) {
        check(!record.interruption?.pendingAfter && record.interruption?.afterOffset===record.interruption?.beforeOffset, 'interrupt cancels pending reveal without moving offset');
        check(record.summary.afterInputOffsetSpan===0 && record.final.scrollTop===record.interruption.afterOffset, 'offset never resumes after cancellation');
        check(!record.overlayFirstOn && !record.summary.anyOverlayAfterInput && record.summary.afterInputObservationMs>=1250, 'overlay remains absent throughout cancellation observation');
        check(record.events.some(e=>e.kind==='native-scrollTo' && e.args[0]?.behavior==='instant' && e.args[0]?.top===record.interruption.afterOffset), 'installed cancellation delegates instant freeze to native scrollTo');
      } else {
        check(Math.abs(record.final.scrollTop-spec.target)<=0.5, 'native center target reached');
        check(record.events.some(e=>e.kind==='scrollend' && e.isTrusted), 'real browser scrollend observed');
        check(!!record.overlayFirstOn && !!record.overlayFirstOff && record.overlayFirstOn.scrollTop===record.final.scrollTop && record.overlayFirstOn.duration===700, 'native settle precedes real 700ms overlay and removal');
        if (spec.input) check(record.interruption.pendingAfter && record.summary.afterInputOffsetSpan>100, 'ordinary key/click does not cancel native movement');
      }
    }
    console.log(`${result.errors.some(e=>e.startsWith(spec.name+':'))?'FAIL':'PASS'} ${JSON.stringify(record.summary)}`);
  }
} catch (error) {
  result.errors.push(String(error.stack || error));
} finally {
  try { await evaluate('window.__r107Smooth?.dispose();true'); result.fixtureDisposed = true; } catch (error) { result.errors.push('Fixture cleanup: '+error); }
  try { await send('Emulation.setEmulatedMedia', {features:[]}); await send('Emulation.clearDeviceMetricsOverride'); await send('Emulation.setFocusEmulationEnabled', {enabled:false}); } catch (error) { result.errors.push('Emulation cleanup: '+error); }
  for (const request of pending.values()) clearTimeout(request.timeout);
  ws.close();
  try { result.ownership.after = ownershipSnapshot(); } catch (error) { result.errors.push('Final ownership: '+error); }
  result.installedBundle.sha256After = createHash('sha256').update(readFileSync(bundle)).digest('hex');
  if (result.installedBundle.sha256After !== bundleHash) result.errors.push('Installed bundle changed on disk');
  result.passed = result.errors.length===0;
  writeFileSync(outputPath+'-state.json', JSON.stringify(result,null,2)+'\n');
  const trajectory = result.cases.filter(c=>c.summary.sourceCurveComparison && !c.spec.input);
  const cancellations = result.cases.filter(c=>c.spec.cancel);
  if (trajectory.length) writeFileSync(outputPath+'-trajectories.svg', chart(trajectory,false));
  if (cancellations.length) writeFileSync(outputPath+'-interruptions.svg', chart(cancellations,true));
  writeFileSync(outputPath+'-report.md', reportMarkdown());
}
if (!result.passed) { console.error(result.errors.join('\n')); process.exitCode=1; }
else console.log(`PASS ${result.cases.length} real-time cases, ${result.checks.length} checks; evidence ${outputPath}-state.json`);
