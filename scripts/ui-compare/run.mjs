#!/usr/bin/env node
// scripts/ui-compare/run.mjs — side-by-side visual compare: original DSH web
// client (headless Chrome, own vendored Host) vs MyGo native whaleisle.exe
// (isolated profile, UIA). Zero npm deps; node >= 22 for global WebSocket.
//
//   node scripts/ui-compare/run.mjs [--surface workbench] [--landmarks <path>] [--keep-open]
//
// Hard rules: only processes spawned here are touched; every secret stays in
// process memory; the temp root is removed in finally unless --keep-open.

import { spawn, spawnSync } from 'node:child_process';
import { mkdirSync, writeFileSync, readFileSync, existsSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import http from 'node:http';

const REPO = process.cwd();
const TOOL_DIR = new URL('.', import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1');
const RUNTIME = join(REPO, 'runtime-rc4');
const NODE = join(RUNTIME, 'node', 'node.exe');
const HOST_BIN = join(RUNTIME, 'vendor', 'deepseek-harness', 'apps', 'cli', 'lib', 'bin.js');
const CHROME = 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const PS = join(TOOL_DIR, 'native.ps1');
const VIEW = { w: 1280, h: 800 };
const WAIT_MS = 10000;
// One workspace directory, same display name on both sides: the web client
// labels its cwd workspace "默认工作区" and the native sidebar shows the
// configured directory's basename.
const WS_DIR_NAME = '默认工作区';

const args = process.argv.slice(2);
const arg = (k, d) => { const i = args.indexOf('--' + k); return i >= 0 ? args[i + 1] : d; };
const surface = arg('surface', 'workbench');
const keepOpen = args.includes('--keep-open');
const landmarksPath = arg('landmarks', join(TOOL_DIR, 'landmarks.json'));

const runId = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
const tmp = join(process.env.LOCALAPPDATA, 'Temp', 'wi-ui-compare-' + runId);
const outDir = join(REPO, 'tmp', 'ui-compare', runId);
mkdirSync(tmp, { recursive: true });
mkdirSync(outDir, { recursive: true });

const sleep = (ms) => new Promise(r => setTimeout(r, ms));
const pids = []; // every pid this script started
const tracked = (child, tag) => { pids.push({ pid: child.pid, tag }); return child; };
const freePort = () => new Promise((res, rej) => {
  const s = http.createServer();
  s.listen(0, '127.0.0.1', () => { const p = s.address().port; s.close(() => res(p)); });
  s.on('error', rej);
});

function ps1(command, extra = []) {
  const r = spawnSync('powershell.exe', ['-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', PS, '-ProcId', String(nativePid), '-Command', command, ...extra], { encoding: 'utf8', timeout: 60000 });
  if (r.status !== 0) throw new Error(`native.ps1 ${command}: ${r.stderr || r.stdout}`);
  return JSON.parse(r.stdout.replace(/^﻿/, ''));
}
const find = (name, type, index = 0, spec = {}) => {
  const extra = ['-Name', name || '', '-Type', type || '', '-Index', String(index)];
  if (spec.namePrefix) extra.push('-Prefix', spec.namePrefix);
  if (spec.up) extra.push('-Up', String(spec.up));
  if (spec.upToType) extra.push('-UpToType', spec.upToType);
  return ps1('find', extra);
};
const act = (cmd, spec) => {
  const extra = ['-Name', spec.name || '', '-Type', spec.type || '', '-Index', String(spec.index || 0)];
  if (spec.namePrefix) extra.push('-Prefix', spec.namePrefix);
  return ps1(cmd, extra);
};

const lm0 = JSON.parse(readFileSync(landmarksPath, 'utf8'));
const surfaceSpec = lm0.surfaces[surface];
if (!surfaceSpec) throw new Error(`unknown surface ${surface}; have ${Object.keys(lm0.surfaces).join(', ')}`);

// A surface may declare "mock": a dependency-free Messages-compatible model
// server (mock.mjs) on 127.0.0.1:0. It is started BEFORE either Host so both
// launch envs can carry DEEPSEEK_BASE_URL/DEEPSEEK_API_KEY into their own
// child env only — nothing is written to any profile; the pid joins the
// tracked set and dies with the rest of the run.
let mockBase = '';
if (surfaceSpec.mock) {
  mockProc = tracked(spawn(process.execPath, [join(TOOL_DIR, 'mock.mjs'), '--log', join(outDir, 'mock.jsonl')], { stdio: ['ignore', 'pipe', 'inherit'] }), 'mock');
  mockProc.stdout.setEncoding('utf8');
  const mockPort = await new Promise((res, rej) => {
    const t = setTimeout(() => rej(new Error('mock.mjs emitted no PORT')), 20000);
    mockProc.stdout.on('data', (d) => { const m = /PORT=(d+)/.exec(d); if (m) { clearTimeout(t); res(+m[1]); } });
  });
  mockBase = `http://127.0.0.1:${mockPort}`;
  console.log('[mock]', mockBase);
}
const modelEnv = mockBase ? { DEEPSEEK_BASE_URL: mockBase + '/v1', DEEPSEEK_API_KEY: 'mock-key' } : {};

let nativePid = 0, chromeProc = null, host2 = null, nativeProc = null, ws = null, cdpPort = 0, cdpSeq = 0, mockProc = null;
const cdpPending = new Map();
const cdp = (method, params = {}) => new Promise((res, rej) => {
  const id = ++cdpSeq;
  cdpPending.set(id, (m) => m.error ? rej(new Error(method + ': ' + JSON.stringify(m.error))) : res(m.result));
  ws.send(JSON.stringify({ id, method, params }));
});
const evalJs = async (expression) => (await cdp('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })).result.value;

function killAll() {
  for (const { pid } of pids) { try { process.kill(pid); } catch {} }
  spawnSync('powershell.exe', ['-NoProfile', '-Command',
    `Get-CimInstance Win32_Process | Where-Object {($_.ProcessId -ne $PID) -and ($_.CommandLine -like '*wi-ui-compare-${runId}*' -or $_.ExecutablePath -like '*wi-ui-compare-${runId}*')} | ForEach-Object {Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue}`], { timeout: 20000 });
}

const summary = { run: runId, surface, scale: 0, scenarios: [], ok: true };
const rows = [];
let fatal = null;

try {
  // ---- Host #2 (original side). Its cwd IS its workspace (the Host uses the
  // child cwd; the web client's initializeDefault labels it 默认工作区), so it
  // is pointed at the same directory the native profile gets. ----
  const wsDir = join(tmp, WS_DIR_NAME);
  mkdirSync(wsDir, { recursive: true });

  // A surface may seed workspace content (file trees, a git repo) into the
  // shared directory before either Host launches.
  if (surfaceSpec.workspace) {
    const s = spawnSync(NODE, [join(TOOL_DIR, surfaceSpec.workspace), wsDir], { encoding: 'utf8' });
    if (s.status !== 0) throw new Error(`workspace seed: ${(s.stderr || s.stdout || '').slice(0, 400)}`);
    console.log((s.stdout || '').trim());
  }

  // A surface may seed a static session history into BOTH Hosts' DSH homes
  // before either launches (native: <profile>/dsh-home; web: host2's
  // DSH_HOME). Sessions land under sessions/<projectKey(wsDir)>/, so both
  // sidebars list the same fixture under the same workspace.
  const nativeHome = join(tmp, 'data', 'dsh-home');
  const host2Home = join(tmp, 'host2-home');
  if (surfaceSpec.fixture) {
    const fx = join(TOOL_DIR, surfaceSpec.fixture);
    for (const home of [host2Home, nativeHome]) {
      const s = spawnSync(NODE, [join(TOOL_DIR, 'seed-chat.mjs'),
        join(RUNTIME, 'vendor', 'deepseek-harness'), home, wsDir, fx], { encoding: 'utf8' });
      if (s.status !== 0) throw new Error(`seed-chat ${home}: ${(s.stderr || s.stdout || '').slice(0, 400)}`);
      console.log('[seed]', (s.stdout || '').trim());
    }
  }
  const port = await freePort();
  host2 = tracked(spawn(NODE, [HOST_BIN, 'web', '--host', '127.0.0.1', '--port', String(port), '--no-open'],
    { env: { ...process.env, DSH_HOME: host2Home, ...modelEnv }, cwd: wsDir, stdio: ['ignore', 'pipe', 'ignore'] }), 'host2');
  let readyUrl = '';
  host2.stdout.setEncoding('utf8');
  host2.stdout.on('data', (d) => { const m = /dsh web:\s*(https?:\/\/\S+)/.exec(d); if (m && !readyUrl) readyUrl = m[1]; });
  const hostDeadline = Date.now() + 120000;
  while (!readyUrl && Date.now() < hostDeadline) { if (host2.exitCode !== null) throw new Error('host2 exited ' + host2.exitCode); await sleep(250); }
  if (!readyUrl) throw new Error('host2 emitted no ready URL');
  console.log('[host2]', new URL(readyUrl).origin);

  // ---- native build + isolated launch ----
  console.log('[build] whaleisle.exe ...');
  const b = spawnSync('go', ['build', '-o', join(tmp, 'whaleisle.exe'), './cmd/whaleisle'], { cwd: REPO, encoding: 'utf8', timeout: 600000 });
  if (b.status !== 0) throw new Error('go build: ' + (b.stderr || '').slice(0, 400));
  const cfgPath = join(tmp, 'data', 'config.json');
  const launchNative = () => tracked(spawn(join(tmp, 'whaleisle.exe'), [], {
    env: { ...process.env, WHALEISLE_ISOLATED: '1', WHALEISLE_ISOLATED_VISIBLE: '1', WHALEISLE_USER_DATA: join(tmp, 'data'), WHALEISLE_LOG_FILE: join(tmp, 'whaleisle.log'), ...modelEnv },
    cwd: REPO, stdio: ['ignore', 'ignore', 'ignore'],
  }), 'whaleisle');
  // Phase 1: let the app write its default config, then patch the workspace
  // in and relaunch — the shell only auto-starts its Host once a workspace
  // is chosen, and a sparse config is not equivalent to the app's defaults.
  nativeProc = launchNative();
  const cfgDeadline = Date.now() + 90000;
  while (Date.now() < cfgDeadline) {
    await sleep(1000);
    if (existsSync(cfgPath) && readFileSync(cfgPath, 'utf8').includes('dshVersion')) break;
  }
  if (!existsSync(cfgPath)) throw new Error('isolated profile did not write config.json');
  try { process.kill(nativeProc.pid); } catch {}
  spawnSync('powershell.exe', ['-NoProfile', '-Command',
    `Get-CimInstance Win32_Process | Where-Object {$_.ExecutablePath -like '*wi-ui-compare-${runId}*'} | ForEach-Object {Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue}`], { timeout: 20000 });
  await sleep(1500);
  const cfg = JSON.parse(readFileSync(cfgPath, 'utf8').replace(/^﻿/, ''));
  cfg.workspace = wsDir;
  // The startup-slowness offer can surface mid-scenario (its observer needs
  // real frames) and covers the pane under test; the dismiss loop only runs
  // during setup, so suppress the offer in the isolated profile outright.
  cfg.goMinimalModeSuggestionsOff = true;
  writeFileSync(cfgPath, JSON.stringify(cfg, null, 2));
  nativeProc = launchNative();

  // The spawned pid may not own the window — whaleisle.exe runs a helper
  // process beside the windowed one. Discover the pid whose MainWindowHandle
  // is set, among OUR exe (temp path) only.
  const winPid = () => {
    const r = spawnSync('powershell.exe', ['-NoProfile', '-Command',
      `Get-CimInstance Win32_Process -Filter "Name='whaleisle.exe'" | Where-Object {$_.ExecutablePath -like '*wi-ui-compare-${runId}*'} | ForEach-Object { if ((Get-Process -Id $_.ProcessId -ErrorAction SilentlyContinue).MainWindowHandle -ne 0) { $_.ProcessId } }`],
      { encoding: 'utf8', timeout: 30000 });
    return parseInt((r.stdout || '').trim(), 10) || 0;
  };

  // Wait for the workbench: window pid + its first landmark.
  const firstNative = surfaceSpec.scenarios[0].landmarks[0].native;
  const bootDeadline = Date.now() + 240000;
  for (;;) {
    await sleep(1500);
    if (!nativePid) nativePid = winPid();
    if (nativePid) {
      try {
        const f = find(firstNative.name, firstNative.type, firstNative.index || 0, firstNative);
        if (f.count > (firstNative.index || 0)) break;
      } catch { /* window not up yet */ }
    }
    if (Date.now() > bootDeadline) throw new Error('native workbench did not appear in 240 s');
  }
  console.log(`[native] window pid ${nativePid}; workbench up`);
  // The shell's own Host is its child (exe lives in runtime-rc4, not the temp
  // root, so the run-id sweep cannot see it) — track it from the host record.
  try {
    const rec = JSON.parse(readFileSync(join(tmp, 'data', 'runtime', 'host-process.json'), 'utf8'));
    if (rec.pid) pids.push({ pid: rec.pid, tag: 'shell-host' });
    if (rec.shellPid && rec.shellPid !== nativePid) pids.push({ pid: rec.shellPid, tag: 'whaleisle' });
  } catch {}

  // Resize the client to the DIP viewport without activating, retrying while
  // the app settles its own initial geometry.
  let scale = 0, client = null, settled = false;
  for (let i = 0; i < 30 && !settled; i++) {
    if (i) await sleep(700);
    client = ps1('setclient', ['-Width', String(VIEW.w), '-Height', String(VIEW.h)]);
    const cdip = client.dpi / 96;
    settled = Math.abs(client.clientW / cdip - VIEW.w) <= 2 && Math.abs(client.clientH / cdip - VIEW.h) <= 2;
  }
  if (!settled) throw new Error(`setclient did not settle: ${client.clientW}x${client.clientH} px`);
  scale = client.dpi / 96;
  console.log(`[native] pid ${nativePid}, scale ${scale}x, client ${client.clientW}x${client.clientH}`);
  summary.scale = scale;

  // ---- Chrome headless at the same DIP viewport ----
  cdpPort = await freePort();
  chromeProc = tracked(spawn(CHROME, [
    '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
    '--user-data-dir=' + join(tmp, 'chrome'), '--remote-debugging-port=' + cdpPort,
    '--remote-debugging-address=127.0.0.1', `--force-device-scale-factor=${scale}`, 'about:blank',
  ], { stdio: ['ignore', 'ignore', 'ignore'] }), 'chrome');
  let target = null;
  const cdl = Date.now() + 30000;
  while (!target && Date.now() < cdl) {
    try { const r = await fetch(`http://127.0.0.1:${cdpPort}/json/list`); target = (await r.json()).find(t => t.type === 'page'); }
    catch { await sleep(300); }
  }
  if (!target) throw new Error('no CDP page target');
  ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise(r => ws.onopen = r);
  ws.onmessage = (e) => { const m = JSON.parse(e.data); if (m.id && cdpPending.has(m.id)) { cdpPending.get(m.id)(m); cdpPending.delete(m.id); } };
  await cdp('Page.enable'); await cdp('Runtime.enable');
  await cdp('Emulation.setDeviceMetricsOverride', { width: VIEW.w, height: VIEW.h, deviceScaleFactor: scale, mobile: false });
  await cdp('Page.navigate', { url: readyUrl }); // Host mints dsh-auth-* and redirects
  await sleep(4500);
  console.log('[web] title:', await evalJs('document.title'));

  // A surface may declare webShell: the seeded workspace's files-shell.json is
  // installed as window.shell — the preload IPC the desktop app provides, fed
  // with the fixture's real listing/content/git data (seed-files.mjs). The
  // served client's desktop pages read window.shell lazily, so installing it
  // here, before any step opens a tab, is enough.
  if (surfaceSpec.webShell) {
    const payload = JSON.parse(readFileSync(join(tmp, 'files-shell.json'), 'utf8'));
    await evalJs(`(()=>{const p=${JSON.stringify(payload)};
      window.shell={
        listDir:async(cwd,rel)=>({ok:true,entries:p.tree[(rel??'').replace(/^[\\\\/]+|[\\\\/]+$/g,'')]??[]}),
        readFile:async(cwd,rel)=>(rel in p.texts?{ok:true,text:p.texts[rel]}:{ok:false,message:'not found'}),
        readFileMedia:async()=>({ok:false,message:'no media'}),
        writeFile:async()=>({ok:true}),
        listEditors:async()=>[],
        gitStatus:async()=>({ok:true}),
        gitDiff:async()=>p.diff,
        gitStatusEntries:async()=>({ok:true,entries:p.entries}),
        gitBranchList:async()=>({ok:true,branches:p.branches,defaultRef:p.defaultRef}),
      };return true})()`);
    console.log('[web] window.shell bridge installed');
  }

  // ---- helpers over both sides ----
  // A web spec: {selector?, text?, within?, upTo?, index?} — scope = within
  // (default document), candidates match selector (default all visible),
  // narrowed to elements whose own text nodes equal text, indexed, then
  // optionally lifted to the closest upTo ancestor.
  const webFinder = (spec, body) => `(()=>{
    const vis=e=>{const r=e.getBoundingClientRect();if(!(r.width>0&&r.height>0))return false;
      for(let n=e;n;n=n.parentElement){const cs=getComputedStyle(n);
        if(cs.display==='none'||cs.visibility==='hidden'||cs.contentVisibility==='hidden')return false;
        if(n.hidden===true||n.hidden==='')return false}
      return true};
    const scope=${spec.within ? `[...document.querySelectorAll(${JSON.stringify(spec.within)})].filter(vis)[${spec.withinIndex||0}]||document` : 'document'};
    let els=[...scope.querySelectorAll(${JSON.stringify(spec.selector || '*')})].filter(vis);
    ${spec.text ? `els=els.filter(e=>{let t='';for(const n of e.childNodes)if(n.nodeType===3)t+=n.textContent;t=t.trim();return ${JSON.stringify([].concat(spec.text))}.includes(t)});` : ''}
    let e=els[${spec.index || 0}];if(!e)return null;
    ${spec.upTo ? `e=e.closest(${JSON.stringify(spec.upTo)})||e;` : ''}
    ${body}
  })()`;
  const webClick = (spec) => evalJs(webFinder(spec, `const r=e.getBoundingClientRect();const x=r.x+r.width/2,y=r.y+r.height/2;const ev=(t)=>e.dispatchEvent(new MouseEvent(t,{bubbles:true,clientX:x,clientY:y,button:0}));ev('mousedown');ev('mouseup');e.click();return true`));
  const webKey = (key) => evalJs(`document.activeElement && document.activeElement.dispatchEvent(new KeyboardEvent('keydown',{key:${JSON.stringify(key)},bubbles:true}))`);
  const webPresent = (spec) => evalJs(webFinder(spec, 'return true'));
  const webMetric = (spec) => evalJs(webFinder(spec, `const r=e.getBoundingClientRect();const cs=getComputedStyle(e);return {x:+r.x.toFixed(2),y:+r.y.toFixed(2),w:+r.width.toFixed(2),h:+r.height.toFixed(2),padding:cs.padding,radius:cs.borderRadius,fontSize:cs.fontSize,lineHeight:cs.lineHeight,fontWeight:cs.fontWeight}`));
  const nativeRect = (spec) => {
    const f = find(spec.name, spec.type || '', spec.index || 0, spec);
    if (f.count <= (spec.index || 0)) return null;
    const r = f.hits[spec.index || 0].rect;
    return { x: +((r.x - client.clientX) / scale).toFixed(2), y: +((r.y - client.clientY) / scale).toFixed(2), w: +(r.w / scale).toFixed(2), h: +(r.h / scale).toFixed(2) };
  };
  const nativePresent = (spec) => find(spec.name, spec.type || '', spec.index || 0, spec).count > (spec.index || 0);

  const report = { run: runId, scale, scenarios: [] };

  // Setup dismiss steps are verified: the `gone` element must be absent after
  // the dismiss, otherwise the scenario reports ERROR rather than a landmark
  // FAIL. `optional` only excuses a target that never appeared.
  // Dismiss is a verified loop, not a single click: a chain of first-run
  // dialogs can appear one after another (预览版说明, then 开始使用), and the
  // wrong button (e.g. 登录, also _primary) must never be clicked. Candidates
  // are tried in order while the `gone` element is present, until it stays
  // gone for a quiet period.
  async function webSetup(s) {
    if (!s.dismiss) return null;
    const gone = typeof s.gone === 'object' ? s.gone : { selector: s.gone || s.dismiss };
    const candidates = (Array.isArray(s.dismiss) ? s.dismiss : [s.dismiss])
      .map(d => typeof d === 'object' ? d : { selector: d });
    const t0 = Date.now();
    let quietSince = 0, sawTarget = false;
    while (Date.now() - t0 < WAIT_MS * 4) {
      if (!(await webPresent(gone))) {
        if (!quietSince) { quietSince = Date.now(); }
        else if (Date.now() - quietSince > 800) {
          return !sawTarget && !s.optional ? `web setup: dismiss target ${JSON.stringify(gone)} never appeared` : null;
        }
        await sleep(200);
        continue;
      }
      sawTarget = true;
      quietSince = 0;
      for (const c of candidates) { if (await webClick(c)) break; }
      await sleep(400);
    }
    if (!sawTarget && s.optional) return null;
    if (!sawTarget) return `web setup: dismiss target ${JSON.stringify(gone)} never appeared`;
    return `web setup: ${JSON.stringify(gone)} still present after dismiss`;
  }
  async function nativeSetup(s) {
    if (!s.dismiss) return null;
    const gone = s.gone || s.dismiss;
    // Same verified loop as webSetup: first-run prompts can arrive seconds
    // after the workbench is up, so watch for the target, dismiss each time
    // it appears, and settle on a quiet period.
    const t0 = Date.now();
    let quietSince = 0, sawTarget = false;
    while (Date.now() - t0 < WAIT_MS * 4) {
      if (!nativePresent(gone)) {
        if (!quietSince) { quietSince = Date.now(); }
        else if (Date.now() - quietSince > 3500) {
          return !sawTarget && !s.optional ? `native setup: dismiss target ${JSON.stringify(gone)} never appeared` : null;
        }
        await sleep(200);
        continue;
      }
      sawTarget = true;
      quietSince = 0;
      act('invoke', s.dismiss);
      await sleep(400);
    }
    return `native setup: ${JSON.stringify(gone)} still present after dismiss`;
  }

  for (const sc of surfaceSpec.scenarios) {
    // setup first, then steps; web before native
    let scError = null;
    for (const s of (sc.web?.setup || [])) { scError = scError || await webSetup(s, sc); await sleep(300); }
    for (const s of (sc.native?.setup || [])) { scError = scError || await nativeSetup(s); await sleep(300); }
    if (!scError) {
      // An expect polls until the target appears: a section swap or lazy page
      // can take longer than a single 500 ms tick to re-mount its content.
      const webExpect = async (spec) => {
        const t0 = Date.now();
        while (Date.now() - t0 < WAIT_MS) { if (await webPresent(spec)) return true; await sleep(250); }
        return false;
      };
      const nativeExpect = async (spec) => {
        const t0 = Date.now();
        while (Date.now() - t0 < WAIT_MS) { if (nativePresent(spec)) return true; await sleep(250); }
        return false;
      };
      for (const s of (sc.web?.steps || [])) {
        if (s.click && !(await webClick(typeof s.click === 'object' ? s.click : { selector: s.click }))) scError = `web step: ${JSON.stringify(s.click)} not found`;
        if (s.key) await webKey(s.key);
        // prompt: focus the ProseMirror composer with a real click, insert
        // the text through CDP trusted input, then submit with Enter. The
        // contenteditable ignores untrusted execCommand/insertText paths.
        if (s.prompt && !scError) {
          const t0 = Date.now();
          let rect = null;
          while (Date.now() - t0 < WAIT_MS) {
            rect = await evalJs(`(()=>{const e=document.querySelector("[role=textbox]");if(!e)return null;const r=e.getBoundingClientRect();return r.width>0?{x:r.x+r.width/2,y:r.y+r.height/2}:null})()`);
            if (rect) break;
            await sleep(250);
          }
          if (!rect) scError = `web step: composer [role=textbox] absent`;
          else {
            await cdp("Page.bringToFront");
            await cdp("Input.dispatchMouseEvent", { type: "mousePressed", x: rect.x, y: rect.y, button: "left", clickCount: 1 });
            await cdp("Input.dispatchMouseEvent", { type: "mouseReleased", x: rect.x, y: rect.y, button: "left", clickCount: 1 });
            await sleep(300);
            await cdp("Input.insertText", { text: s.prompt });
            await sleep(300);
            await cdp("Input.dispatchKeyEvent", { type: "keyDown", key: "Enter", code: "Enter", windowsVirtualKeyCode: 13 });
            await cdp("Input.dispatchKeyEvent", { type: "keyUp", key: "Enter", code: "Enter", windowsVirtualKeyCode: 13 });
          }
        }
        await sleep(500);
        // Steps are verified in order: an expect spec must be present before
        // the next step runs, else the scenario is an ERROR, not a FAIL.
        if (s.expect && !scError && !(await webExpect(s.expect))) scError = `web step expect: ${JSON.stringify(s.expect)} absent`;
        if (scError) break;
      }
      for (const s of (sc.native?.steps || [])) {
        if (s.setvalue && !scError) {
          let r = ps1('setvalue', ['-Name', s.setvalue.name || '', '-Type', s.setvalue.type || '', '-Index', String(s.setvalue.index || 0), '-Text', s.setvalue.text || '']);
          const t0 = Date.now();
          while (!r.ok && /no match/.test(r.error || '') && Date.now() - t0 < WAIT_MS) { await sleep(300); r = ps1('setvalue', ['-Name', s.setvalue.name || '', '-Type', s.setvalue.type || '', '-Index', String(s.setvalue.index || 0), '-Text', s.setvalue.text || '']); }
          if (!r.ok) scError = `native step setvalue ${s.setvalue.name}: ${r.error}`;
        }
        const cmd = s.invoke ? 'invoke' : s.expand ? 'expand' : s.collapse ? 'collapse' : s.select ? 'select' : null;
        // Poll for the step target the way expects do: first-run dialogs and
        // lazy lists can cost the target a few frames.
        if (cmd && !(await nativeExpect(s[cmd]))) scError = `native step ${cmd} ${s[cmd].name}: target absent`;
        // The act runs in a second process; a re-render between the presence
        // poll and the action can drop the element for one spawn, so retry on
        // "no match" the same way the poll does.
        if (cmd && !scError) {
          let r = act(cmd, s[cmd]);
          const t0 = Date.now();
          while (!r.ok && /no match/.test(r.error || '') && Date.now() - t0 < WAIT_MS) { await sleep(300); r = act(cmd, s[cmd]); }
          if (!r.ok) scError = `native step ${cmd} ${s[cmd].name}: ${r.error}`;
        }
        await sleep(500);
        if (s.expect && !scError && !(await nativeExpect(s.expect))) scError = `native step expect: ${JSON.stringify(s.expect)} absent`;
        if (scError) break;
      }
    }

    const lms = [];
    const byKey = new Map();
    if (scError) {
      lms.push({ key: sc.id, status: 'ERROR', error: scError, web: null, native: null, delta: {}, missing: [] });
      rows.push({ sc: sc.id, key: sc.id, status: 'ERROR', error: scError });
      summary.ok = false;
    } else {
      for (const lm of sc.landmarks) {
        // Boot probes exist only to detect the workbench before steps run;
        // they are never measured (e.g. a sidebar button hidden by a modal).
        if (lm.probe) continue;
        const t0 = Date.now();
        let web = null, nat = null;
        while (Date.now() - t0 < WAIT_MS) {
          if (!web) web = await webMetric(lm.web);
          if (!nat) nat = nativeRect(lm.native);
          if (web && nat) break;
          await sleep(250);
        }
        const tol = lm.tolerance ?? 1;
        const entry = { key: lm.key, web, native: nat, delta: {}, pass: true, missing: [] };
        if (!web) entry.missing.push('web');
        if (!nat) entry.missing.push('native');
        if (web && nat) {
          const prev = lms.filter(e => e.web && e.native).pop();
          const parent = lm.parent ? byKey.get(lm.parent) : null;
          for (const f of lm.assert || []) {
            let d = null;
            if (f === 'dy-prev') {
              if (prev) d = +((nat.y - (prev.native.y + prev.native.h)) - (web.y - (prev.web.y + prev.web.h))).toFixed(2);
            } else if (f === 'dx-parent-left') {
              if (parent && parent.web && parent.native) d = +((nat.x - parent.native.x) - (web.x - parent.web.x)).toFixed(2);
            } else if (f === 'dx-parent-right') {
              if (parent && parent.web && parent.native) d = +(((parent.native.x + parent.native.w) - (nat.x + nat.w)) - ((parent.web.x + parent.web.w) - (web.x + web.w))).toFixed(2);
            } else {
              d = +(nat[f] - web[f]).toFixed(2);
            }
            entry.delta[f] = d;
            if (d !== null && Math.abs(d) > tol) entry.pass = false;
          }
        } else {
          entry.pass = false;
        }
        if (!entry.pass) summary.ok = false;
        entry.status = entry.pass ? 'PASS' : entry.missing.length ? 'MISS' : 'FAIL';
        lms.push(entry);
        byKey.set(lm.key, entry);
        rows.push({ sc: sc.id, ...entry });
      }
    }
    // screenshots (+ optional web DOM dump for landmark discovery)
    const shot = await cdp('Page.captureScreenshot', { format: 'png' });
    writeFileSync(join(outDir, `${sc.id}-web.png`), Buffer.from(shot.data, 'base64'));
    if (sc.domDump) {
      const htmlDump = await evalJs(`(document.querySelector(${JSON.stringify(sc.domDump)})||document.body).outerHTML`);
      writeFileSync(join(outDir, `${sc.id}-dom.html`), htmlDump);
    }
    if (sc.nativeDump) {
      try { writeFileSync(join(outDir, `${sc.id}-uia.json`), JSON.stringify(ps1('dump', ['-Depth', String(sc.nativeDumpDepth || 8)]))); }
      catch (e) { writeFileSync(join(outDir, `${sc.id}-uia.error.txt`), String(e)); }
    }
    ps1('capture', ['-Out', join(outDir, `${sc.id}-native.png`)]);
    summary.scenarios.push({ id: sc.id, landmarks: lms, ...(scError ? { error: scError } : {}) });
    report.scenarios.push({ id: sc.id, landmarks: lms, error: scError, webPng: `${sc.id}-web.png`, nativePng: `${sc.id}-native.png` });
  }

  writeFileSync(join(outDir, 'summary.json'), JSON.stringify(summary, null, 2));

  // ---- report.html ----
  let html = `<!doctype html><meta charset=utf-8><title>ui-compare ${runId}</title><style>body{font:13px system-ui}table{border-collapse:collapse}td,th{border:1px solid #ccc;padding:2px 8px}.wrap{display:flex;gap:8px}.pane{position:relative}</style><h1>ui-compare ${runId} — scale ${scale}</h1>`;
  for (const sc of report.scenarios) {
    html += `<h2>${sc.id}</h2><div class="wrap">`;
    for (const side of ['web', 'native']) {
      const rects = sc.landmarks.map(m => {
        const r = side === 'web' ? m.web : m.native;
        if (!r) return '';
        return `<rect x="${r.x * scale}" y="${r.y * scale}" width="${r.w * scale}" height="${r.h * scale}" fill="none" stroke="${m.pass ? '#090' : '#d33'}" stroke-width="3"/><text x="${r.x * scale}" y="${Math.max(12, r.y * scale - 4)}" font-size="16" fill="${m.pass ? '#090' : '#d33'}">${m.key}</text>`;
      }).join('');
      html += `<div class="pane"><img src="${side === 'web' ? sc.webPng : sc.nativePng}" style="display:block"><svg style="position:absolute;inset:0" width="${VIEW.w * scale}" height="${VIEW.h * scale}">${rects}</svg><div>${side}</div></div>`;
    }
    html += `</div>`;
    if (sc.error) html += `<p><b>ERROR:</b> ${sc.error}</p>`;
    html += `<table><tr><th>key</th><th>web x,y,w,h</th><th>native x,y,w,h</th><th>deltas</th><th>web radius/pad/font</th><th></th></tr>`;
    for (const m of sc.landmarks) {
      const f = (r) => r ? `${r.x}, ${r.y}, ${r.w}, ${r.h}` : '—';
      const d = Object.keys(m.delta).length ? Object.entries(m.delta).map(([k, v]) => `${k} ${v === null ? 'n/a' : (v > 0 ? '+' : '') + v}`).join(' ') : '—';
      const ref = m.web ? `${m.web.radius} / ${m.web.padding} / ${m.web.fontSize} ${m.web.fontWeight}` : '—';
      html += `<tr><td>${m.key}</td><td>${f(m.web)}</td><td>${f(m.native)}</td><td>${d}</td><td>${ref}</td><td>${m.status || (m.pass ? 'PASS' : 'MISS')}</td></tr>`;
    }
    html += '</table>';
  }
  writeFileSync(join(outDir, 'report.html'), html);

  // ---- stdout table ----
  console.log('\nkey                          | web (x,y,w,h)      | native (x,y,w,h)   | delta            | result');
  for (const r of rows) {
    if (r.status === 'ERROR') { console.log(`${r.key.padEnd(28)} | ERROR: ${r.error}`); continue; }
    const f = (v) => v ? `${v.x},${v.y},${v.w},${v.h}`.padEnd(18) : '—                 ';
    const d = Object.keys(r.delta).length ? Object.entries(r.delta).map(([k, v]) => k + '=' + (v === null ? 'n/a' : (v > 0 ? '+' : '') + v)).join(',') : '—';
    console.log(`${r.key.padEnd(28)} | ${f(r.web)} | ${f(r.native)} | ${d.padEnd(16)} | ${r.status}`);
  }
  console.log('\nreport:', join('tmp', 'ui-compare', runId, 'report.html'));
} catch (e) {
  fatal = e;
  console.error('ERROR:', e.message);
} finally {
  if (ws) try { ws.close(); } catch {}
  if (keepOpen) {
    console.log('[keep-open] PIDs left running:', pids.map(p => `${p.tag}=${p.pid}`).join(' '));
    console.log('[keep-open] stop with: powershell -c "Stop-Process -Id ' + pids.map(p => p.pid).join(',') + '"');
    console.log('[keep-open] temp root kept at', tmp);
  } else {
    if (fatal) {
      try {
        const log = join(tmp, 'whaleisle.log');
        if (existsSync(log)) writeFileSync(join(outDir, 'whaleisle.log'), readFileSync(log));
      } catch {}
    }
    const trackedIds = pids.map(p => `'${p.pid}'`).join(',');
    let leftover = [];
    for (let i = 0; i < 10; i++) {
      killAll();
      await sleep(1500);
      const check = spawnSync('powershell.exe', ['-NoProfile', '-Command',
        `Get-CimInstance Win32_Process | Where-Object {($_.ProcessId -ne $PID) -and (${trackedIds ? `@(${trackedIds}) -contains "$($_.ProcessId)" -or ` : ''}$_.CommandLine -like '*wi-ui-compare-${runId}*' -or $_.ExecutablePath -like '*wi-ui-compare-${runId}*')} | ForEach-Object {"$($_.ProcessId) $($_.Name)"}`], { encoding: 'utf8', timeout: 20000 });
      leftover = (check.stdout || '').trim().split(/\r?\n/).filter(Boolean);
      if (!leftover.length) break;
    }
    if (leftover.length) console.warn('WARNING leftover pids:', leftover.join(', '));
    try { rmSync(tmp, { recursive: true, force: true }); } catch (e) { console.warn('temp root kept (in use):', tmp, e.message); }
  }
}
process.exit(fatal || !summary.ok ? 1 : 0);
