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
const RUNTIME = join(REPO, 'runtime-rc4');
const NODE = join(RUNTIME, 'node', 'node.exe');
const HOST_BIN = join(RUNTIME, 'vendor', 'deepseek-harness', 'apps', 'cli', 'lib', 'bin.js');
const CHROME = 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const PS = join(REPO, 'scripts', 'ui-compare', 'native.ps1');
const VIEW = { w: 1280, h: 800 };
const WAIT_MS = 5000;

const args = process.argv.slice(2);
const arg = (k, d) => { const i = args.indexOf('--' + k); return i >= 0 ? args[i + 1] : d; };
const surface = arg('surface', 'workbench');
const keepOpen = args.includes('--keep-open');
const landmarksPath = arg('landmarks', join(REPO, 'scripts', 'ui-compare', 'landmarks.json'));

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
const find = (name, type, index = 0) => ps1('find', ['-Name', name, '-Type', type, '-Index', String(index)]);
const act = (cmd, spec) => ps1(cmd, ['-Name', spec.name, '-Type', spec.type || '', '-Index', String(spec.index || 0)]);

let nativePid = 0, chromeProc = null, host2 = null, nativeProc = null, ws = null, cdpPort = 0, cdpSeq = 0;
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
  // ---- Host #2 (original side) ----
  const port = await freePort();
  host2 = tracked(spawn(NODE, [HOST_BIN, 'web', '--host', '127.0.0.1', '--port', String(port), '--no-open'],
    { env: { ...process.env, DSH_HOME: join(tmp, 'host2-home') }, stdio: ['ignore', 'pipe', 'ignore'] }), 'host2');
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
  mkdirSync(join(tmp, 'ws'), { recursive: true });
  const cfgPath = join(tmp, 'data', 'config.json');
  const launchNative = () => tracked(spawn(join(tmp, 'whaleisle.exe'), [], {
    env: { ...process.env, WHALEISLE_ISOLATED: '1', WHALEISLE_ISOLATED_VISIBLE: '1', WHALEISLE_USER_DATA: join(tmp, 'data'), WHALEISLE_LOG_FILE: join(tmp, 'whaleisle.log') },
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
  cfg.workspace = join(tmp, 'ws');
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
  const lm0 = JSON.parse(readFileSync(landmarksPath, 'utf8'));
  const firstNative = lm0.surfaces[surface].scenarios[0].landmarks[0].native;
  const bootDeadline = Date.now() + 240000;
  for (;;) {
    await sleep(1500);
    if (!nativePid) nativePid = winPid();
    if (nativePid) {
      try {
        const f = find(firstNative.name, firstNative.type, firstNative.index || 0);
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

  // ---- helpers over both sides ----
  const webClick = (sel) => evalJs(`(()=>{const e=[...document.querySelectorAll(${JSON.stringify(sel)})].find(e=>{const r=e.getBoundingClientRect();return r.width>0&&r.height>0});if(!e)return false;const r=e.getBoundingClientRect();const x=r.x+r.width/2,y=r.y+r.height/2;const ev=(t)=>e.dispatchEvent(new MouseEvent(t,{bubbles:true,clientX:x,clientY:y,button:0}));ev('mousedown');ev('mouseup');e.click();return true})()`);
  const webKey = (key) => evalJs(`document.activeElement && document.activeElement.dispatchEvent(new KeyboardEvent('keydown',{key:${JSON.stringify(key)},bubbles:true}))`);
  const webMetric = (sel) => evalJs(`(()=>{const e=[...document.querySelectorAll(${JSON.stringify(sel)})].find(e=>{const r=e.getBoundingClientRect();return r.width>0||r.height>0});if(!e)return null;const r=e.getBoundingClientRect();const cs=getComputedStyle(e);return {x:+r.x.toFixed(2),y:+r.y.toFixed(2),w:+r.width.toFixed(2),h:+r.height.toFixed(2),padding:cs.padding,radius:cs.borderRadius,fontSize:cs.fontSize,lineHeight:cs.lineHeight,fontWeight:cs.fontWeight}})()`);
  const nativeRect = (spec) => {
    const f = find(spec.name, spec.type || '', spec.index || 0);
    if (f.count <= (spec.index || 0)) return null;
    const r = f.hits[spec.index || 0].rect;
    return { x: +((r.x - client.clientX) / scale).toFixed(2), y: +((r.y - client.clientY) / scale).toFixed(2), w: +(r.w / scale).toFixed(2), h: +(r.h / scale).toFixed(2) };
  };

  const report = { run: runId, scale, scenarios: [] };

  for (const sc of lm0.surfaces[surface].scenarios) {
    // setup + steps, web first then native
    for (const s of (sc.web?.setup || [])) {
      if (s.dismiss) { const ok = await webClick(s.dismiss); if (!ok && !s.optional) throw new Error(`web setup dismiss ${s.dismiss} missing`); }
      await sleep(400);
    }
    for (const s of (sc.web?.steps || [])) {
      if (s.click && !(await webClick(s.click))) throw new Error(`web step click ${s.click} missing`);
      if (s.key) await webKey(s.key);
      await sleep(500);
    }
    for (const s of (sc.native?.steps || [])) {
      const cmd = s.invoke ? 'invoke' : s.expand ? 'expand' : s.collapse ? 'collapse' : null;
      if (cmd) { const r = act(cmd, s[cmd]); if (!r.ok) throw new Error(`native ${cmd} ${s[cmd].name}: ${r.error}`); }
      await sleep(500);
    }
    // bounded wait for every landmark on both sides
    const lms = [];
    for (const lm of sc.landmarks) {
      const t0 = Date.now();
      let web = null, nat = null;
      while (Date.now() - t0 < WAIT_MS) {
        if (!web) web = await webMetric(lm.web.selector);
        if (!nat) nat = nativeRect(lm.native);
        if (web && nat) break;
        await sleep(250);
      }
      const tol = lm.tolerance ?? 1;
      const entry = { key: lm.key, web, native: nat, delta: {}, pass: false, missing: [] };
      if (!web) entry.missing.push('web');
      if (!nat) entry.missing.push('native');
      if (web && nat) {
        entry.pass = true;
        for (const f of lm.assert || []) {
          entry.delta[f] = +(nat[f] - web[f]).toFixed(2);
          if (Math.abs(entry.delta[f]) > tol) entry.pass = false;
        }
      }
      if (!entry.pass) summary.ok = false;
      lms.push(entry);
      rows.push({ sc: sc.id, ...entry });
    }
    // screenshots
    const shot = await cdp('Page.captureScreenshot', { format: 'png' });
    writeFileSync(join(outDir, `${sc.id}-web.png`), Buffer.from(shot.data, 'base64'));
    ps1('capture', ['-Out', join(outDir, `${sc.id}-native.png`)]);
    summary.scenarios.push({ id: sc.id, landmarks: lms });
    report.scenarios.push({ id: sc.id, landmarks: lms, webPng: `${sc.id}-web.png`, nativePng: `${sc.id}-native.png` });
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
    html += `</div><table><tr><th>key</th><th>web x,y,w,h</th><th>native x,y,w,h</th><th>Δx Δy Δw Δh</th><th>web radius/pad/font</th><th></th></tr>`;
    for (const m of sc.landmarks) {
      const f = (r) => r ? `${r.x}, ${r.y}, ${r.w}, ${r.h}` : '—';
      const d = Object.keys(m.delta).length ? ['x', 'y', 'w', 'h'].map(k => m.delta[k] ?? '·').join(' ') : '—';
      const ref = m.web ? `${m.web.radius} / ${m.web.padding} / ${m.web.fontSize} ${m.web.fontWeight}` : '—';
      html += `<tr><td>${m.key}</td><td>${f(m.web)}</td><td>${f(m.native)}</td><td>${d}</td><td>${ref}</td><td>${m.pass ? 'PASS' : m.missing.length ? 'MISS ' + m.missing.join('+') : 'FAIL'}</td></tr>`;
    }
    html += '</table>';
  }
  writeFileSync(join(outDir, 'report.html'), html);

  // ---- stdout table ----
  console.log('\nkey                          | web (x,y,w,h)      | native (x,y,w,h)   | delta            | result');
  for (const r of rows) {
    const f = (v) => v ? `${v.x},${v.y},${v.w},${v.h}`.padEnd(18) : '—                 ';
    const d = Object.keys(r.delta).length ? ['x', 'y', 'w', 'h'].map(k => (r.delta[k] > 0 ? '+' : '') + (r.delta[k] ?? 0)).join(',') : '—';
    console.log(`${r.key.padEnd(28)} | ${f(r.web)} | ${f(r.native)} | ${d.padEnd(16)} | ${r.pass ? 'PASS' : r.missing.length ? 'MISS' : 'FAIL'}`);
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
