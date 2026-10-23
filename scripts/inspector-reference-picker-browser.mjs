#!/usr/bin/env node
/**
 * Round111: run the exact installed InspectorChatPicker in an isolated real Chrome.
 * No packages, application server, original GUI/profile, or Go tooling are used.
 * The only writable paths are this script's fixed evidence/fixture paths and a
 * fresh child of .devdata/r111-picker-chrome. All browser input uses CDP Input.
 */
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createHash, randomUUID } from 'node:crypto';
import net from 'node:net';
import { spawn } from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts', 'inspector-reference-picker-browser.mjs');
const INSTALLED = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness\\packages\\experimental\\session-inspector\\lib\\client.js';
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const FIXTURE = path.join(ROOT, '.devdata', 'r111-original-picker.html');
const PROFILE_BASE = path.join(ROOT, '.devdata', 'r111-picker-chrome');
const PREFIX = path.join(ROOT, 'docs', 'review-20261022', 'evidence', 'inspector-pass255-reference-picker');
const OUTPUT = { json: `${PREFIX}-browser.json`, text: `${PREFIX}-browser.txt`, screenshot: `${PREFIX}-browser.png` };
const RUN_ID = `${new Date().toISOString().replace(/[:.]/g, '-')}-${randomUUID().slice(0, 8)}`;
const PROFILE = path.join(PROFILE_BASE, RUN_ID);
const PROTECTED_PIDS = new Set([20012, 16008]);
const PROTECTED_PORTS = new Set([3080]);
const allowedFiles = new Set([FIXTURE, ...Object.values(OUTPUT)].map(normalize));
const evidence = {
  schema: 'inspector-pass255-reference-picker-browser/v1', round: 111, runId: RUN_ID,
  startedAt: new Date().toISOString(), status: 'running',
  scope: {
    description: 'Actual installed InspectorChatPicker and five verbatim installed functions in a controlled DOM, navigated by file URL in a separate headless Chrome. This is not the original GUI or full original application.',
    workspace: ROOT, script: SCRIPT, installedBundle: INSTALLED, fixture: FIXTURE,
    uniqueUserDataDir: PROFILE, allowedProfileRoot: PROFILE_BASE,
    protected: { electronPid: 20012, backendPid: 16008, gui: 'http://127.0.0.1:3080', policy: 'Never signal, restart, navigate, read profile, or mutate. No operation in this runner targets these processes, port, or profiles.' },
    dependencies: 'Node >=24 built-ins, installed Chrome executable; no dependency install, builds, Go tests, model calls, application server, or external fetch.',
    input: 'CDP Input.dispatchMouseEvent and Input.dispatchKeyEvent only. No DOM dispatchEvent or direct picker event-handler calls.',
    network: 'The runner connects only to the allocated 127.0.0.1 CDP endpoint. Fixture CSP denies network; page Network.setBlockedURLs blocks HTTP(S)/WS(S). Chrome background networking is disabled and DNS resolution is blocked.'
  },
  writes: [], extraction: {}, browser: {}, cases: [], errors: [], harnessRevision: {
    priorDiagnosticRun: `${PREFIX}-browser-attempt1.json`,
    correction: 'First-run logger sampled defaultPrevented too early in a capture-listener microtask, missed Escape at document capture after window suppression, and required compatibility mouse events suppressed by cancelled pointerdown. Final logger observes at window capture before picker registration and reads retained native Event.defaultPrevented during the later snapshot. Assertions allow absent compatibility mouse events. Installed picker/functions/CSS were never changed.'
  }, limitations: [
    'Controlled main conversation DOM and fixture theme, not the full original application, current original GUI, live session data, React integration, or original Electron Chromium.',
    'Only the extracted installed picker and its required functions/CSS are executed; unrelated installed module registration and app lifecycle are outside this acceptance scope.',
    'Pixel-mode wheel forwarding is exercised. Line/page wheel delta modes, touch input, and platform-native menus are outside this run.'
  ]
};
let browser, page, chrome, chromeExit, chromeExitInfo, port, ownershipVerified = false;
let sourceBefore;
const protocolErrors = [], pageExceptions = [], requests = [];

function normalize(value) { return path.resolve(value).toLowerCase(); }
function invariant(ok, message) { if (!ok) throw new Error(message); }
function hash(value) { return createHash('sha256').update(value).digest('hex'); }
function delay(ms) { invariant(ms >= 0 && ms <= 1000, 'Short bounded delay only'); return new Promise(resolve => setTimeout(resolve, ms)); }
function escapeRegExp(value) { return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }
async function noLinks(target) {
  const resolved = path.resolve(target), parsed = path.parse(resolved);
  let cursor = parsed.root;
  for (const part of resolved.slice(parsed.root.length).split(path.sep)) {
    if (!part) continue;
    cursor = path.join(cursor, part);
    try { invariant(!(await fs.lstat(cursor)).isSymbolicLink(), `Refuse link/junction: ${cursor}`); }
    catch (error) { if (error.code === 'ENOENT') break; throw error; }
  }
}
async function checkScope() {
  invariant(process.platform === 'win32', 'This scoped runner is Windows-only');
  invariant(Number(process.versions.node.split('.')[0]) >= 24, 'Node24 or newer is required');
  invariant(process.argv.length === 2, 'No path, port, executable, or profile overrides are accepted');
  invariant(normalize(fileURLToPath(import.meta.url)) === normalize(SCRIPT), 'Script must run from the exact owned script path');
  invariant(normalize(process.cwd()) === normalize(ROOT), 'Working directory must be the exact authorized workspace');
  for (const target of [ROOT, SCRIPT, INSTALLED, CHROME, FIXTURE, PROFILE_BASE, ...Object.values(OUTPUT)]) await noLinks(target);
  for (const directory of [ROOT, path.dirname(SCRIPT), path.dirname(PREFIX), path.dirname(FIXTURE)]) {
    invariant(normalize(await fs.realpath(directory)) === normalize(directory), `Refuse redirected directory: ${directory}`);
  }
  invariant(normalize(PROFILE).startsWith(`${normalize(PROFILE_BASE)}${path.sep}`), 'Profile is outside its owned root');
  try { await fs.lstat(PROFILE); throw new Error('Unique profile already exists; refuse reuse'); } catch (error) { if (error.code !== 'ENOENT') throw error; }
}
async function writeOwned(target, value) {
  invariant(allowedFiles.has(normalize(target)), `Refuse out-of-scope write: ${target}`);
  await noLinks(target);
  try { await fs.readFile(target); } catch (error) { if (error.code !== 'ENOENT') throw error; }
  await fs.writeFile(target, value);
  if (!evidence.writes.includes(target)) evidence.writes.push(target);
}
function extractLines(source, name, startPattern, endPattern, afterLine = 0) {
  const lines = source.split('\n');
  const start = lines.findIndex((line, index) => index >= afterLine && startPattern.test(line.replace(/\r$/, '')));
  invariant(start >= 0, `Missing installed region ${name}`);
  const end = lines.findIndex((line, index) => index >= start && endPattern.test(line.replace(/\r$/, '')));
  invariant(end >= start, `Unterminated installed region ${name}`);
  let startByte = 0;
  for (let index = 0; index < start; index++) startByte += lines[index].length + 1;
  let endByte = startByte;
  for (let index = start; index <= end; index++) endByte += lines[index].length + (index < lines.length - 1 ? 1 : 0);
  const verbatim = source.slice(startByte, endByte);
  return { name, startLine: start + 1, endLine: end + 1, startUtf16Offset: startByte, endUtf16OffsetExclusive: endByte, utf8Bytes: Buffer.byteLength(verbatim), sha256: hash(verbatim), verbatim };
}
function extractInstalled(source) {
  const functions = ['displayed', 'inChat', 'matchChatElement', 'visibleChatRect', 'placeChatHighlight'].map(name => {
    const startPattern = new RegExp(`^\\t\\tfunction ${name}\\(`);
    return extractLines(source, name, startPattern, /^\t\t}\r?$/);
  });
  const pointer = extractLines(source, 'POINTER_ACTIONS', /^\t\tconst POINTER_ACTIONS = \[/, /^\t\t];\r?$/);
  const picker = extractLines(source, 'InspectorChatPicker', /^\t\tvar InspectorChatPicker = class \{/, /^\t\t};\r?$/);
  const cssMatch = source.match(/^\t\tconst css = ("(?:[^"\\]|\\.)*");\r?$/m);
  invariant(cssMatch, 'Installed generated CSS literal not found');
  const css = JSON.parse(cssMatch[1]);
  const mapping = {};
  const mappingRegions = [];
  for (const key of ['chatPick', 'chatPickMask']) {
    const match = source.match(new RegExp(`^\\t\\t\\t"${key}": "([A-Za-z0-9_-]+)",?\\r?$`, 'm'));
    invariant(match, `Installed CSS mapping ${key} not found`);
    mapping[key] = match[1];
    mappingRegions.push({ name: key, sourceLine: source.slice(0, match.index).split('\n').length, verbatim: match[0], sha256: hash(match[0]) });
  }
  const rules = [];
  for (const match of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (Object.values(mapping).some(className => new RegExp(`\\.${escapeRegExp(className)}(?=[\\s,:.#\\[]|$)`).test(match[1]))) {
      rules.push({ selector: match[1], declarations: match[2], verbatim: match[0], sha256: hash(match[0]) });
    }
  }
  for (const name of Object.values(mapping)) invariant(rules.some(rule => rule.selector.split(',').includes(`.${name}`)), `Missing exact installed CSS rule ${name}`);
  invariant(rules.length === 2, 'Review changed installed picker CSS before running');
  const preview = rules.find(rule => rule.selector === `.${mapping.chatPick}`);
  invariant(preview.declarations.includes('color-mix(in srgb, var(--dsw-alias-state-business-primary) 15%, transparent)'), 'Installed preview no longer uses the expected 15% color rule');
  const regions = [...functions, pointer, picker];
  for (const region of regions) invariant(!/<\/script/i.test(region.verbatim), `Unsafe inline script delimiter in ${region.name}`);
  evidence.extraction = {
    installedBundleSha256: hash(source), installedBundleUtf8Bytes: Buffer.byteLength(source),
    regions, css: { sourceLine: source.slice(0, cssMatch.index).split('\n').length, generatedLiteralSha256: hash(cssMatch[1]), decodedCssSha256: hash(css), mapping, mappingRegions, rules, appliedCssSha256: hash(rules.map(rule => rule.verbatim).join('\n')) },
    method: 'Functions, POINTER_ACTIONS, and class sliced verbatim at exact original indentation boundaries. CSS decoded with JSON.parse from the installed generated literal; only the exact two installed picker rules are applied. The class and functions are not transformed, mocked, transpiled, or reimplemented.'
  };
  return { code: regions.map(region => region.verbatim).join('\n'), css: rules.map(rule => rule.verbatim).join('\n'), mapping };
}
function makeFixture(extracted) {
  return `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; connect-src 'none'; frame-src 'none'"><title>Round111 isolated installed InspectorChatPicker acceptance</title>
<style>
/* Fixture layout/theme only. Picker declarations below are copied from the installed bundle. */
:root { --dsw-alias-state-business-primary:#2f74c8; color-scheme:light; font:15px Arial,sans-serif; color:#172235; background:#e9eef6; }
* { box-sizing:border-box; } body { margin:0; overflow:hidden; } h1 { font-size:19px; margin:0 0 6px; } p { margin:0; }
#caption { position:fixed; left:40px; top:22px; right:30px; } #caption p { color:#58677f; font-size:13px; }
[data-slot="main.conversation"] { position:fixed; left:40px; right:320px; top:100px; bottom:80px; background:white; border-radius:8px; }
[data-slot="conversation.view"] { width:100%; height:100%; }
[data-conversation-scroll] { position:relative; width:100%; height:100%; overflow:auto; overflow-anchor:none; line-height:20px; }
[data-chat-flow] { position:relative; width:100%; height:1800px; }
#turn { position:absolute; top:32px; left:24px; right:24px; height:140px; }
#primary { height:108px; background:#f1f4f9; border:1px solid #b7c5d8; border-radius:7px; }
#call { width:100%; height:100%; display:grid; place-items:center; }
button { font:inherit; padding:9px 13px; border:1px solid #a3b2c7; background:white; border-radius:5px; cursor:pointer; }
#clipper { position:absolute; left:24px; right:24px; top:500px; height:220px; overflow:hidden; background:#dbe4f0; }
#partial { position:relative; left:-18px; top:-28px; width:calc(100% + 36px); height:260px; background:#edf3fc; padding:50px; }
#nestedSidebar { position:absolute; top:220px; left:30px; width:240px; height:60px; background:#fff0d4; }
#nestedView { position:absolute; top:320px; left:30px; width:240px; height:60px; background:#f6e8f8; }
#nestedSidebar button,#nestedView button { width:100%; height:100%; }
[data-composer-seat] { position:sticky; bottom:0; height:84px; margin-top:-84px; background:#d6e0ed; z-index:3; display:flex; align-items:center; justify-content:center; border-top:1px solid #b7c5d8; }
#inspectorPane { position:fixed; right:24px; top:100px; width:264px; padding:18px; background:white; border-radius:8px; } #outside { margin-top:22px; } #inspectorPane p { font-size:13px; margin-top:14px; line-height:1.5; }
</style><style data-installed-picker-css="${evidence.extraction.css.appliedCssSha256}">${extracted.css}</style></head>
<body><header id="caption"><h1>Installed InspectorChatPicker · isolated browser acceptance</h1><p>Controlled fixture, file URL, fresh Chrome profile. Original application and GUI are not loaded.</p></header>
<main data-slot="main.conversation"><div data-slot="conversation.view"><div id="scroller" data-conversation-scroll>
<div id="flow" data-chat-flow>
<div id="turn" data-chat-turn="7"><div id="group" data-chat-group-key="group-seven"><section id="primary" data-chat-node-key="node-primary" data-chat-group-part="assistant" data-chat-flow-kind="message"><div id="call" data-chat-call-id="call-primary"><button id="underlying">Underlying action (must be caught while picking)</button></div></section></div></div>
<aside id="nestedSidebar" data-sidebar-right-session><button id="sidebarContent" data-chat-node-key="excluded-sidebar">Excluded nested sidebar</button></aside>
<div id="nestedView" data-slot="conversation.view"><button id="nestedContent" data-chat-node-key="excluded-nested-view">Excluded nested conversation view</button></div>
<div id="clipper" data-step-process-body><section id="partial" data-chat-node-key="node-clipped" data-chat-group-part="result">Clipped node extends outside its step box and below the composer.</section></div>
</div><div id="composer" data-composer-seat><button id="composerButton">Composer outside transcript mask</button></div>
</div></div></main>
<aside id="inspectorPane"><button id="owner">Inspector picker owner</button><button id="outside">Outside control</button><p>The picker class and five functions are verbatim installed source. Preview and mask declarations are the installed generated CSS.</p></aside>
<script>
'use strict';
const inspector_module_css_default = ${JSON.stringify(extracted.mapping)};
${extracted.code}
(() => {
  const root = document.getElementById('flow');
  const owner = document.getElementById('owner');
  const scroller = document.getElementById('scroller');
  let active;
  const state = { ready:false, accepts:true, attempts:[], cancellations:0, releases:0, starts:0, underlying:[], outsideClicks:0, composerClicks:0, events:[], windowEvents:[], hiddenChanges:[] };
  const types = ['pointerdown','pointerup','mousedown','mouseup','click','auxclick','dblclick','contextmenu','wheel','keydown','keyup'];
  const observedInputs = [];
  // Window capture is registered before picker.start(), including its Escape listener.
  // Retain each native Event and sample defaultPrevented in the later snapshot;
  // a microtask inside an early listener can run before target listeners in Chrome.
  for (const type of types) window.addEventListener(type, event => {
    const entry = { type, trusted:event.isTrusted, targetId:event.target.id || null, mask:event.target.hasAttribute?.('data-session-inspector-picker') || false, button:event.button ?? null, key:event.key ?? null, deltaY:event.deltaY ?? null, defaultPrevented:event.defaultPrevented };
    state.events.push(entry);
    observedInputs.push({ event, entry });
  }, true);
  for (const type of types.filter(type => !type.startsWith('key'))) document.getElementById('underlying').addEventListener(type, event => state.underlying.push({ type, trusted:event.isTrusted, button:event.button }));
  document.getElementById('outside').addEventListener('click', () => state.outsideClicks++);
  document.getElementById('composerButton').addEventListener('click', () => state.composerClicks++);
  for (const type of ['blur','focus','resize','scroll']) window.addEventListener(type, event => state.windowEvents.push({ type, trusted:event.isTrusted, at:performance.now() }), true);
  const rect = element => { const box = element.getBoundingClientRect(); return { left:box.left, top:box.top, right:box.right, bottom:box.bottom, width:box.width, height:box.height }; };
  const styles = element => { const style = getComputedStyle(element); return Object.fromEntries(['position','zIndex','pointerEvents','cursor','touchAction','backgroundColor','borderRadius','outlineWidth','outlineStyle','outlineColor'].map(key => [key,style[key]])); };
  function snapshot() {
    for (const {event,entry} of observedInputs) entry.defaultPrevented=event.defaultPrevented;
    const mask = document.querySelector('[data-session-inspector-picker]');
    const overlay = document.querySelector('[data-session-inspector-highlight="pick"]');
    return {
      counts:{ attempts:state.attempts.length, cancellations:state.cancellations, releases:state.releases, starts:state.starts, underlying:state.underlying.length, underlyingClicks:state.underlying.filter(event => event.type === 'click').length, outsideClicks:state.outsideClicks, composerClicks:state.composerClicks, events:state.events.length, windowEvents:state.windowEvents.length },
      active:!!active?.mask, accepts:state.accepts, focus:document.hasFocus(), viewport:{width:innerWidth,height:innerHeight},
      mask:mask ? { rect:rect(mask), style:styles(mask), parent:mask.parentElement.tagName, ariaHidden:mask.getAttribute('aria-hidden'), className:mask.className } : null,
      overlay:overlay ? { rect:rect(overlay), style:styles(overlay), parent:overlay.parentElement.tagName, ariaHidden:overlay.getAttribute('aria-hidden'), className:overlay.className } : null,
      rects:Object.fromEntries(['flow','scroller','composer','call','underlying','clipper','partial','outside','composerButton','sidebarContent','nestedContent'].map(id => [id,rect(document.getElementById(id))])),
      layout:{ scrollHeight:scroller.scrollHeight,scrollWidth:scroller.scrollWidth,clientHeight:scroller.clientHeight,clientWidth:scroller.clientWidth,flowOffsetWidth:root.offsetWidth,flowOffsetHeight:root.offsetHeight },
      scroll:{top:scroller.scrollTop,left:scroller.scrollLeft,windowX:scrollX,windowY:scrollY}, rootHTML:root.outerHTML,
      attempts:state.attempts.slice(), events:state.events.slice(), underlying:state.underlying.slice(), windowEvents:state.windowEvents.slice(),
      owner:{hidden:owner.hidden,ancestorHidden:document.getElementById('inspectorPane').hidden,display:getComputedStyle(owner).display},
      nativeObservers:{mutation:String(MutationObserver),resize:String(ResizeObserver)}
    };
  }
  window.harness = {
    state, snapshot,
    reset() { if (active?.mask) active.dispose(); document.getElementById('inspectorPane').hidden=false; owner.hidden=false; scroller.scrollTop=0; scroller.scrollLeft=0; },
    arm(accepts) {
      if (active?.mask) active.dispose();
      state.accepts=accepts;
      const picker = new InspectorChatPicker(() => root, owner, target => { state.attempts.push({ target,accepted:state.accepts }); return state.accepts; }, () => state.cancellations++, () => { state.releases++; if (active === picker) active=undefined; });
      active=picker; const started=picker.start(); if (started) state.starts++; return started;
    },
    hideOwnerAncestor() { state.hiddenChanges.push({at:performance.now(),action:'set Inspector owner ancestor.hidden=true'}); document.getElementById('inspectorPane').hidden=true; },
    ready:true
  };
  state.ready=true;
})();
</script></body></html>`;
}

class CDP {
  constructor(socket) {
    this.socket=socket; this.nextId=0; this.pending=new Map(); this.listeners=[];
    socket.addEventListener('message', event => {
      const message=JSON.parse(String(event.data));
      if (message.id) {
        const entry=this.pending.get(message.id); if (!entry) return;
        clearTimeout(entry.timer); this.pending.delete(message.id);
        if (message.error) { protocolErrors.push({method:entry.method,error:message.error}); entry.reject(new Error(`${entry.method}: ${JSON.stringify(message.error)}`)); }
        else entry.resolve(message.result);
      } else for (const listener of this.listeners) listener(message);
    });
    socket.addEventListener('close', () => { for (const entry of this.pending.values()) { clearTimeout(entry.timer); entry.reject(new Error(`${entry.method}: owned CDP closed`)); } this.pending.clear(); });
  }
  static async connect(url) {
    const parsed=new URL(url);
    invariant(parsed.protocol === 'ws:' && parsed.hostname === '127.0.0.1' && Number(parsed.port) === port, 'Refuse non-owned CDP WebSocket');
    const socket=new WebSocket(url);
    await new Promise((resolve,reject) => {
      const timer=setTimeout(() => reject(new Error('Owned CDP WebSocket open timeout')),10000);
      socket.addEventListener('open',() => { clearTimeout(timer);resolve(); },{once:true});
      socket.addEventListener('error',() => { clearTimeout(timer);reject(new Error('Owned CDP WebSocket failed')); },{once:true});
    });
    return new CDP(socket);
  }
  send(method,params={},sessionId) {
    invariant(this.socket.readyState === WebSocket.OPEN, `CDP is closed: ${method}`);
    const id=++this.nextId;
    return new Promise((resolve,reject) => {
      const timer=setTimeout(() => {this.pending.delete(id);reject(new Error(`${method}: 10s command timeout`));},10000);
      this.pending.set(id,{resolve,reject,timer,method});
      this.socket.send(JSON.stringify({id,method,params,...(sessionId ? {sessionId}: {})}));
    });
  }
  session(sessionId) { return {send:(method,params={}) => this.send(method,params,sessionId)}; }
}
async function waitFor(probe,label,timeout=10000) {
  invariant(timeout <= 15000, 'Wait exceeds bounded operation limit');
  const deadline=Date.now()+timeout;
  do { if (await probe()) return; await delay(50); } while (Date.now()<deadline);
  throw new Error(`Timeout waiting for ${label}`);
}
async function evaluate(expression) {
  const result=await page.send('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});
  invariant(!result.exceptionDetails, `Fixture evaluation failed: ${JSON.stringify(result.exceptionDetails)}`);
  return result.result.value;
}
async function snap() { return evaluate('window.harness.snapshot()'); }
async function frames() { await evaluate('new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))'); }
async function prepare(accepts=true) {
  await evaluate('window.harness.reset()'); await frames();
  invariant(await evaluate(`window.harness.arm(${accepts})`),'Installed picker did not start');
  await frames(); return snap();
}
function center(rect) { return {x:(rect.left+rect.right)/2,y:(rect.top+rect.bottom)/2}; }
async function move(point) { await page.send('Input.dispatchMouseEvent',{type:'mouseMoved',...point,button:'none',buttons:0}); await frames(); }
async function click(point,button='left') {
  await move(point);
  const buttons={left:1,right:2,middle:4};
  await page.send('Input.dispatchMouseEvent',{type:'mousePressed',...point,button,buttons:buttons[button],clickCount:1});
  await page.send('Input.dispatchMouseEvent',{type:'mouseReleased',...point,button,buttons:0,clickCount:1});
  await frames();
}
async function escapeKey() {
  for (const type of ['keyDown','keyUp']) await page.send('Input.dispatchKeyEvent',{type,key:'Escape',code:'Escape',windowsVirtualKeyCode:27,nativeVirtualKeyCode:27});
  await frames();
}
function near(a,b) { return Math.abs(a-b) <= 0.75; }
function sameRect(actual,expected) { return !!actual && ['left','top','width','height'].every(key=>near(actual[key],expected[key])); }
function intersect(rects) {
  const left=Math.max(...rects.map(rect=>rect.left)),top=Math.max(...rects.map(rect=>rect.top));
  const right=Math.min(...rects.map(rect=>rect.right)),bottom=Math.min(...rects.map(rect=>rect.bottom));
  return {left,top,right,bottom,width:Math.max(0,right-left),height:Math.max(0,bottom-top)};
}
function expectedRect(snapshot,name,withStep=false) {
  const viewport={left:0,top:0,right:snapshot.viewport.width,bottom:snapshot.viewport.height};
  const transcript={...snapshot.rects.scroller,bottom:snapshot.rects.composer.top};
  return intersect([snapshot.rects[name],viewport,transcript,...(withStep ? [snapshot.rects.clipper]:[])]);
}
function compact(snapshot) {
  const {rootHTML,events,windowEvents,underlying,...rest}=snapshot;
  return {...rest,rootHTMLSha256:hash(rootHTML)};
}
async function runCase(id,action,{optional=false}={}) {
  const entry={id,startedAt:new Date().toISOString(),status:'running',assertions:[],actual:{}};
  evidence.cases.push(entry);
  const check=(name,ok,actual,expected) => {entry.assertions.push({name,pass:!!ok,actual,expected});return !!ok;};
  try {
    await action(entry,check);
    if (entry.status === 'running') entry.status=entry.assertions.every(item=>item.pass) ? 'passed':'failed';
  } catch (error) {entry.status='failed';entry.error=error.stack || String(error);}
  entry.optional=optional; entry.finishedAt=new Date().toISOString();
  console.log(`${entry.status.toUpperCase()} ${id}: ${entry.assertions.filter(item=>item.pass).length}/${entry.assertions.length} assertions${entry.error ? `; ${entry.error.split('\n')[0]}`:''}`);
}
async function allocatePort() {
  const server=net.createServer();
  await new Promise((resolve,reject)=>{server.once('error',reject);server.listen({host:'127.0.0.1',port:0,exclusive:true},resolve);});
  const assigned=server.address().port;
  invariant(assigned >= 1024 && !PROTECTED_PORTS.has(assigned),'Allocated port is protected or privileged');
  await new Promise((resolve,reject)=>server.close(error=>error ? reject(error):resolve()));
  evidence.browser.portAllocation={address:'127.0.0.1',port:assigned,checkedBy:'Exclusive loopback TCP bind with OS-assigned port; released immediately before owned Chrome launch'};
  return assigned;
}
async function startBrowser() {
  port=await allocatePort();
  await noLinks(PROFILE_BASE); await fs.mkdir(PROFILE_BASE,{recursive:true});
  await fs.mkdir(PROFILE,{recursive:false}); evidence.writes.push(`${PROFILE}${path.sep} (Chrome-owned profile directory)`);
  const args=[
    '--headless=new',`--user-data-dir=${PROFILE}`,'--remote-debugging-address=127.0.0.1',`--remote-debugging-port=${port}`,
    '--enable-automation','--no-first-run','--no-default-browser-check','--disable-background-networking','--disable-background-timer-throttling','--disable-backgrounding-occluded-windows','--disable-renderer-backgrounding',
    '--disable-component-update','--disable-sync','--disable-extensions','--disable-default-apps','--disable-domain-reliability','--disable-client-side-phishing-detection','--safebrowsing-disable-auto-update',
    '--disable-features=MediaRouter,OptimizationHints,Translate,AutofillServerCommunication','--metrics-recording-only','--password-store=basic','--no-proxy-server','--host-resolver-rules=MAP * ~NOTFOUND',
    '--disable-breakpad','--disable-gpu','--force-device-scale-factor=1','--window-size=1200,900','about:blank'
  ];
  chrome=spawn(CHROME,args,{cwd:ROOT,stdio:'ignore',windowsHide:true,shell:false});
  invariant(Number.isSafeInteger(chrome.pid) && !PROTECTED_PIDS.has(chrome.pid),'Owned Chrome PID unavailable or protected');
  evidence.browser.launch={executable:CHROME,args,pid:chrome.pid,stdio:'ignore',shell:false};
  chromeExit=new Promise(resolve=>{
    chrome.once('error',error=>{chromeExitInfo={error:String(error)};resolve(chromeExitInfo);});
    chrome.once('exit',(code,signal)=>{chromeExitInfo={code,signal,at:new Date().toISOString()};resolve(chromeExitInfo);});
  });
  let version;
  await waitFor(async()=>{
    invariant(!chromeExitInfo,`Owned Chrome exited during startup: ${JSON.stringify(chromeExitInfo)}`);
    try { const response=await fetch(`http://127.0.0.1:${port}/json/version`,{signal:AbortSignal.timeout(750)}); if(response.ok){version=await response.json();return !!version.webSocketDebuggerUrl;} } catch {}
    return false;
  },'owned Chrome debug endpoint',15000);
  browser=await CDP.connect(version.webSocketDebuggerUrl);
  const [browserVersion,commandLine,processInfo]=await Promise.all([browser.send('Browser.getVersion'),browser.send('Browser.getBrowserCommandLine'),browser.send('SystemInfo.getProcessInfo')]);
  invariant(commandLine.arguments.includes(`--user-data-dir=${PROFILE}`),'CDP endpoint profile does not match owned profile');
  invariant(commandLine.arguments.includes(`--remote-debugging-port=${port}`),'CDP endpoint port does not match owned port');
  const browserProcess=processInfo.processInfo.find(info=>info.type==='browser');
  invariant(browserProcess?.id === chrome.pid,'CDP browser PID does not match the spawned owned process');
  ownershipVerified=true;
  evidence.browser.version=browserVersion; evidence.browser.commandLine=commandLine.arguments; evidence.browser.ownership={verified:true,spawnedPid:chrome.pid,cdpBrowserPid:browserProcess.id,uniqueProfileMatched:true,debugPortMatched:true};
  evidence.browser.initialOwnedProcesses=processInfo.processInfo;
  const targets=await browser.send('Target.getTargets');
  const target=targets.targetInfos.find(info=>info.type==='page' && info.url==='about:blank');
  invariant(target,'Owned blank Chrome target not found');
  const attached=await browser.send('Target.attachToTarget',{targetId:target.targetId,flatten:true});
  evidence.browser.targetId=target.targetId;
  page=browser.session(attached.sessionId);
  browser.listeners.push(message=>{
    if(message.sessionId!==attached.sessionId)return;
    if(message.method==='Runtime.exceptionThrown')pageExceptions.push(message.params);
    if(message.method==='Network.requestWillBeSent')requests.push({url:message.params.request.url,type:message.params.type});
  });
  await page.send('Page.enable'); await page.send('Runtime.enable'); await page.send('Network.enable');
  await page.send('Network.setBlockedURLs',{urls:['http://*','https://*','ws://*','wss://*']});
  await page.send('Emulation.setDeviceMetricsOverride',{width:1200,height:900,deviceScaleFactor:1,mobile:false});
  await page.send('Page.bringToFront');
  const navigation=await page.send('Page.navigate',{url:pathToFileURL(FIXTURE).href});
  invariant(!navigation.errorText,`Fixture navigation failed: ${navigation.errorText}`);
  await waitFor(()=>evaluate('window.harness?.ready === true'),'isolated fixture ready');
  evidence.browser.fixtureUrl=await evaluate('location.href');
  invariant(evidence.browser.fixtureUrl === pathToFileURL(FIXTURE).href,'Page is not the exact owned file URL');
  evidence.browser.userAgent=await evaluate('navigator.userAgent');
}

async function acceptance() {
  await runCase('mask-main-scope-and-no-reflow',async(entry,check)=>{
    const before=await snap(),after=await prepare();
    entry.actual={before:compact(before),after:compact(after)};
    check('mask is a real body child with installed class',after.mask?.parent==='BODY' && after.mask?.className===evidence.extraction.css.mapping.chatPickMask,after.mask,{parent:'BODY',className:evidence.extraction.css.mapping.chatPickMask});
    check('mask is clipped to transcript above composer',sameRect(after.mask?.rect,expectedRect(after,'flow')),after.mask?.rect,expectedRect(after,'flow'));
    check('mask fixed/crosshair/auto pointer/touch none/z1200',after.mask?.style.position==='fixed' && after.mask.style.cursor==='crosshair' && after.mask.style.pointerEvents==='auto' && after.mask.style.touchAction==='none' && after.mask.style.zIndex==='1200',after.mask?.style,{position:'fixed',cursor:'crosshair',pointerEvents:'auto',touchAction:'none',zIndex:'1200'});
    check('mask is aria-hidden',after.mask?.ariaHidden==='true',after.mask?.ariaHidden,'true');
    check('flow DOM and layout are unchanged',before.rootHTML===after.rootHTML && JSON.stringify(before.layout)===JSON.stringify(after.layout) && sameRect(before.rects.call,after.rects.call),{beforeHTML:hash(before.rootHTML),afterHTML:hash(after.rootHTML),beforeLayout:before.layout,afterLayout:after.layout},'identical flow markup, dimensions, scroll extents, and call box');
    const hits=await evaluate(`['outside','composerButton'].map(id=>{const element=document.getElementById(id),rect=element.getBoundingClientRect();const hit=document.elementFromPoint((rect.left+rect.right)/2,(rect.top+rect.bottom)/2);return {id,actualId:hit.id,mask:hit.hasAttribute('data-session-inspector-picker')};})`);
    entry.actual.outsideHits=hits;
    check('outside main and composer controls remain browser-hit-testable',hits.every(hit=>hit.actualId===hit.id && !hit.mask),hits,'outside and composerButton, never mask');
  });
  await runCase('clipped-preview-original-css-and-no-reflow',async(entry,check)=>{
    const before=await prepare(),expected=expectedRect(before,'partial',true);
    invariant(expected.height>0 && expected.width>0,'Clipped target is not visible in controlled fixture');
    await move(center(expected));const after=await snap();
    entry.actual={before:compact(before),after:compact(after),expectedIndependentIntersection:expected};
    check('preview exists above mask and ignores pointer hits',after.overlay?.parent==='BODY' && after.overlay.style.pointerEvents==='none' && after.overlay.style.position==='fixed' && after.overlay.style.zIndex==='1201',after.overlay,{parent:'BODY',pointerEvents:'none',position:'fixed',zIndex:'1201'});
    check('preview clips to step viewport, scroller, composer, and browser viewport',sameRect(after.overlay?.rect,expected),after.overlay?.rect,expected);
    check('fixture really exercises horizontal and vertical clipping',expected.width<before.rects.partial.width && expected.height<before.rects.partial.height && near(expected.bottom,before.rects.composer.top),{expected,target:before.rects.partial,composerTop:before.rects.composer.top},'preview smaller on both axes; lower edge at composer top');
    const style=after.overlay?.style;
    const color=style?.backgroundColor;
    const parsed=color?.match(/^color\(srgb\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*\/\s*([\d.]+)\)$/);
    const channels=parsed ? parsed.slice(1).map(Number):null;
    check('actual installed 15% theme color is computed by Chrome',!!channels && channels.every((value,index)=>Math.abs(value-[47/255,116/255,200/255,0.15][index])<0.00001),{computedColor:color,channels,fixtureTheme:'#2f74c8',installedDeclaration:evidence.extraction.css.rules.find(rule=>rule.selector.endsWith('_chatPick')).declarations},[47/255,116/255,200/255,0.15]);
    check('actual radius and theme outline match installed declarations',style?.borderRadius==='4px' && style.outlineWidth==='1px' && style.outlineStyle==='solid' && style.outlineColor==='rgb(47, 116, 200)',style,{borderRadius:'4px',outlineWidth:'1px',outlineStyle:'solid',outlineColor:'rgb(47, 116, 200)'});
    check('preview is aria-hidden',after.overlay?.ariaHidden==='true',after.overlay?.ariaHidden,'true');
    check('hover causes no flow DOM mutation or reflow',before.rootHTML===after.rootHTML && JSON.stringify(before.layout)===JSON.stringify(after.layout) && sameRect(before.rects.partial,after.rects.partial),{beforeHTML:hash(before.rootHTML),afterHTML:hash(after.rootHTML),beforeLayout:before.layout,afterLayout:after.layout},'identical flow markup, layout, and underlying target box');
    const screenshot=await page.send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false,fromSurface:true});
    await writeOwned(OUTPUT.screenshot,Buffer.from(screenshot.data,'base64'));
    entry.actual.screenshot={path:OUTPUT.screenshot,sha256:hash(Buffer.from(screenshot.data,'base64')),description:'Actual clipped installed picker preview in isolated fixture; not original GUI'};
  });
  await runCase('secondary-and-context-suppressed',async(entry,check)=>{
    const before=await prepare(false),point=center(before.rects.underlying);
    await click(point,'right');await click(point,'middle');const after=await snap();
    const events=after.events.slice(before.counts.events);
    entry.actual={before:compact(before),after:compact(after),events};
    check('secondary buttons cause no underlying action or pick',after.counts.underlying===before.counts.underlying && after.counts.attempts===before.counts.attempts && after.active,{underlyingDelta:after.counts.underlying-before.counts.underlying,attemptDelta:after.counts.attempts-before.counts.attempts,active:after.active},{underlyingDelta:0,attemptDelta:0,active:true});
    for(const type of ['pointerdown','pointerup','contextmenu','auxclick']) {
      const matching=events.filter(event=>event.type===type);
      check(`trusted ${type} reaches mask and is prevented`,matching.length>0 && matching.every(event=>event.trusted && event.mask && event.defaultPrevented),matching,'at least one trusted mask-targeted event, defaultPrevented=true');
    }
    const compatibility=events.filter(event=>['mousedown','mouseup'].includes(event.type));
    check('compatibility mouse events are absent or caught and prevented',compatibility.every(event=>event.trusted && event.mask && event.defaultPrevented),compatibility,'a cancelled pointerdown suppresses compatibility mousedown/up; any emitted mouse events must be caught and prevented');
  });
  await runCase('rejected-pick-stays-active-and-outside-controls-work',async(entry,check)=>{
    const before=await prepare(false);
    await click(center(before.rects.underlying));const rejected=await snap();
    await click(center(rejected.rects.outside));await click(center(rejected.rects.composerButton));const after=await snap();
    entry.actual={before:compact(before),afterRejected:compact(rejected),afterOutsideControls:compact(after),events:after.events.slice(before.counts.events)};
    check('rejected callback keeps mask and gesture active',rejected.counts.attempts===before.counts.attempts+1 && rejected.attempts.at(-1)?.accepted===false && rejected.active && !!rejected.mask && rejected.counts.releases===before.counts.releases,{lastAttempt:rejected.attempts.at(-1),active:rejected.active,releaseDelta:rejected.counts.releases-before.counts.releases},'one rejected real primary pick; active mask; no release');
    check('rejected pick does not trigger underlying events',rejected.counts.underlying===before.counts.underlying,rejected.counts.underlying-before.counts.underlying,0);
    check('outside and composer real clicks work while picker remains active',after.counts.outsideClicks===before.counts.outsideClicks+1 && after.counts.composerClicks===before.counts.composerClicks+1 && after.active,{outsideDelta:after.counts.outsideClicks-before.counts.outsideClicks,composerDelta:after.counts.composerClicks-before.counts.composerClicks,active:after.active},{outsideDelta:1,composerDelta:1,active:true});
    check('leaving transcript clears preview',after.overlay===null,after.overlay,null);
  });
  await runCase('nested-sidebar-and-nested-view-excluded',async(entry,check)=>{
    const before=await prepare(false);
    for(const id of ['sidebarContent','nestedContent']) {await move(center(before.rects[id]));const hover=await snap();check(`${id} gets no preview`,hover.overlay===null,hover.overlay,null);await click(center(before.rects[id]));}
    const after=await snap();entry.actual={before:compact(before),after:compact(after),events:after.events.slice(before.counts.events)};
    check('excluded occurrences do not enter selection callback',after.counts.attempts===before.counts.attempts && after.active,{attemptDelta:after.counts.attempts-before.counts.attempts,active:after.active},{attemptDelta:0,active:true});
  });
  await runCase('primary-selects-releases-and-catches-underlying-action',async(entry,check)=>{
    const before=await prepare(true);await click(center(before.rects.underlying));const selected=await snap();
    const expectedTarget={nodeKey:'node-primary',groupPart:'assistant',nodeKind:'message',callId:'call-primary',groupKey:'group-seven',turn:7};
    const events=selected.events.slice(before.counts.events);
    entry.actual={before:compact(before),afterSelection:compact(selected),events};
    check('real primary click selects exact installed match identities',selected.counts.attempts===before.counts.attempts+1 && JSON.stringify(selected.attempts.at(-1)?.target)===JSON.stringify(expectedTarget),selected.attempts.at(-1),{target:expectedTarget,accepted:true});
    check('selection releases exactly once without cancellation',selected.counts.releases===before.counts.releases+1 && selected.counts.cancellations===before.counts.cancellations,{releaseDelta:selected.counts.releases-before.counts.releases,cancelDelta:selected.counts.cancellations-before.counts.cancellations},{releaseDelta:1,cancelDelta:0});
    check('mask and preview are removed',!selected.active && selected.mask===null && selected.overlay===null,{active:selected.active,mask:selected.mask,overlay:selected.overlay},{active:false,mask:null,overlay:null});
    check('selection triggers no underlying pointer, mouse, or click action',selected.counts.underlying===before.counts.underlying,selected.underlying.slice(before.counts.underlying),[]);
    const actualClick=events.find(event=>event.type==='click');
    check('selection used a trusted browser click caught and prevented by mask',actualClick?.trusted && actualClick.mask && actualClick.defaultPrevented,actualClick,{trusted:true,mask:true,defaultPrevented:true});
    await click(center(selected.rects.underlying));const released=await snap();entry.actual.afterReleasedControlClick=compact(released);entry.actual.releasedUnderlyingEvents=released.underlying.slice(selected.counts.underlying);
    check('same underlying control works after release',released.counts.underlyingClicks===selected.counts.underlyingClicks+1 && released.underlying.slice(selected.counts.underlying).every(event=>event.trusted),released.underlying.slice(selected.counts.underlying),'one trusted underlying click after release');
  });
  await runCase('wheel-forwarded-to-main-scroller-clears-preview',async(entry,check)=>{
    const before=await prepare();const point=center(before.rects.underlying);await move(point);const hovered=await snap();
    await page.send('Input.dispatchMouseEvent',{type:'mouseWheel',...point,deltaX:0,deltaY:140,button:'none',buttons:0});
    await waitFor(async()=> (await snap()).scroll.top>before.scroll.top,'native main scroller wheel movement');await frames();const after=await snap();
    const events=after.events.slice(hovered.counts.events);entry.actual={before:compact(hovered),after:compact(after),events,windowEvents:after.windowEvents.slice(hovered.counts.windowEvents)};
    check('preview was visible before wheel',!!hovered.overlay,hovered.overlay,'visible installed preview');
    check('wheel forwards exact pixel delta to main scroller',near(after.scroll.top-before.scroll.top,140),{before:before.scroll,after:after.scroll},'main scroller +140px');
    check('native scroll event clears preview and preserves active mask',after.overlay===null && after.active && sameRect(after.mask?.rect,expectedRect(after,'flow')),{overlay:after.overlay,active:after.active,mask:after.mask?.rect},expectedRect(after,'flow'));
    check('window does not scroll',after.scroll.windowX===0 && after.scroll.windowY===0,after.scroll,'windowX=0, windowY=0');
    const wheel=events.find(event=>event.type==='wheel');check('trusted browser wheel is caught and prevented by mask',wheel?.trusted && wheel.mask && wheel.defaultPrevented,wheel,{trusted:true,mask:true,defaultPrevented:true});
  });
  await runCase('escape-cancels-with-real-key-input',async(entry,check)=>{
    const before=await prepare();await move(center(before.rects.underlying));await escapeKey();const after=await snap();
    const events=after.events.slice(before.counts.events);entry.actual={before:compact(before),after:compact(after),events};
    check('Escape cancels and releases exactly once',after.counts.cancellations===before.counts.cancellations+1 && after.counts.releases===before.counts.releases+1,{cancelDelta:after.counts.cancellations-before.counts.cancellations,releaseDelta:after.counts.releases-before.counts.releases},{cancelDelta:1,releaseDelta:1});
    check('Escape removes mask and preview',!after.active && !after.mask && !after.overlay,{active:after.active,mask:after.mask,overlay:after.overlay},{active:false,mask:null,overlay:null});
    const key=events.find(event=>event.type==='keydown' && event.key==='Escape');check('Escape is a trusted prevented browser key event',key?.trusted && key.defaultPrevented,key,{trusted:true,key:'Escape',defaultPrevented:true});
  });
  await runCase('real-viewport-resize-clears-and-repositions',async(entry,check)=>{
    const before=await prepare();await move(center(before.rects.underlying));const hovered=await snap();
    await page.send('Emulation.setDeviceMetricsOverride',{width:1000,height:760,deviceScaleFactor:1,mobile:false});
    await waitFor(()=>evaluate('innerWidth === 1000 && innerHeight === 760'),'resized viewport');await frames();const after=await snap();
    entry.actual={before:compact(hovered),afterResize:compact(after),windowEvents:after.windowEvents.slice(hovered.counts.windowEvents)};
    check('real trusted resize event was observed',after.windowEvents.slice(hovered.counts.windowEvents).some(event=>event.type==='resize' && event.trusted),entry.actual.windowEvents,'trusted resize');
    check('resize clears existing preview without cancelling',!!hovered.overlay && after.overlay===null && after.active && after.counts.cancellations===before.counts.cancellations,{beforeOverlay:!!hovered.overlay,afterOverlay:after.overlay,active:after.active,cancelDelta:after.counts.cancellations-before.counts.cancellations},'preview cleared, gesture active');
    check('mask repositions to changed transcript geometry',sameRect(after.mask?.rect,expectedRect(after,'flow')) && after.mask.rect.width!==hovered.mask.rect.width && after.mask.rect.height!==hovered.mask.rect.height,{before:hovered.mask?.rect,after:after.mask?.rect},expectedRect(after,'flow'));
    await move(center(after.rects.underlying));const again=await snap();entry.actual.afterNewHover=compact(again);
    check('subsequent hover uses resized target geometry',sameRect(again.overlay?.rect,expectedRect(again,'call')),again.overlay?.rect,expectedRect(again,'call'));
    await page.send('Emulation.setDeviceMetricsOverride',{width:1200,height:900,deviceScaleFactor:1,mobile:false});await frames();
  });
  await runCase('hidden-owner-cancels-via-real-mutation-observer',async(entry,check)=>{
    const before=await prepare();await move(center(before.rects.underlying));const hovered=await snap();
    await evaluate('window.harness.hideOwnerAncestor()');
    await waitFor(async()=> (await snap()).counts.cancellations>before.counts.cancellations,'native MutationObserver hidden-owner cancellation');
    const after=await snap();entry.actual={before:compact(hovered),after:compact(after),trigger:await evaluate('window.harness.state.hiddenChanges.at(-1)')};
    check('browser native observers are used unmodified',after.nativeObservers.mutation.includes('[native code]') && after.nativeObservers.resize.includes('[native code]'),after.nativeObservers,'native MutationObserver and ResizeObserver');
    check('owner ancestor is actually hidden',after.owner.ancestorHidden,after.owner,{ancestorHidden:true});
    check('attribute mutation cancels and releases exactly once',after.counts.cancellations===before.counts.cancellations+1 && after.counts.releases===before.counts.releases+1,{cancelDelta:after.counts.cancellations-before.counts.cancellations,releaseDelta:after.counts.releases-before.counts.releases},{cancelDelta:1,releaseDelta:1});
    check('observer cancellation removes real mask and preview',!after.active && !after.mask && !after.overlay,{active:after.active,mask:after.mask,overlay:after.overlay},{active:false,mask:null,overlay:null});
  });
  await runCase('window-blur-by-owned-target-activation',async(entry,check)=>{
    const before=await prepare();await move(center(before.rects.underlying));const hovered=await snap();let other;
    try {
      other=await browser.send('Target.createTarget',{url:'about:blank',newWindow:false,background:false});
      await browser.send('Target.activateTarget',{targetId:other.targetId});
      const deadline=Date.now()+1200;let after;
      do {after=await snap();if(after.windowEvents.slice(hovered.counts.windowEvents).some(event=>event.type==='blur'))break;await delay(50);} while(Date.now()<deadline);
      const events=after.windowEvents.slice(hovered.counts.windowEvents),blur=events.find(event=>event.type==='blur' && event.trusted);
      entry.actual={before:compact(hovered),after:compact(after),ownedOtherTargetId:other.targetId,windowEvents:events};
      if(!blur) {
        entry.status='not_verified';entry.reason='Owned headless target activation did not produce a real trusted window blur; no synthetic blur was dispatched.';
        evidence.limitations.push('Window blur cancellation was attempted by activating another owned headless target but no real trusted blur was observed; this path is not claimed as browser-verified.');
      } else {
        check('real trusted window blur was observed',true,blur,'trusted window blur');
        check('real blur cancels and releases exactly once',after.counts.cancellations===before.counts.cancellations+1 && after.counts.releases===before.counts.releases+1,{cancelDelta:after.counts.cancellations-before.counts.cancellations,releaseDelta:after.counts.releases-before.counts.releases},{cancelDelta:1,releaseDelta:1});
        check('real blur removes mask and preview',!after.active && !after.mask && !after.overlay,{active:after.active,mask:after.mask,overlay:after.overlay},{active:false,mask:null,overlay:null});
      }
    } finally {
      if(other)await browser.send('Target.closeTarget',{targetId:other.targetId});
      await page.send('Page.bringToFront');if((await snap()).active)await escapeKey();
    }
  },{optional:true});
  await runCase('page-and-source-integrity',async(entry,check)=>{
    const final=await snap();const sourceAfter=await fs.readFile(INSTALLED,'utf8');
    const external=requests.filter(request=>/^(https?|wss?):/i.test(request.url));
    entry.actual={final:compact(final),pageExceptions,requests,protocolErrors,sourceBeforeSha256:hash(sourceBefore),sourceAfterSha256:hash(sourceAfter)};
    check('installed source remains unchanged',hash(sourceBefore)===hash(sourceAfter),{before:hash(sourceBefore),after:hash(sourceAfter)},'identical SHA256');
    check('no remote page requests or runtime exceptions',external.length===0 && pageExceptions.length===0,{externalRequests:external,pageExceptions},'none');
    check('all recorded fixture input is trusted browser input',final.events.length>0 && final.events.every(event=>event.trusted),{eventCount:final.events.length,untrusted:final.events.filter(event=>!event.trusted)},'every input isTrusted=true');
    check('final gesture has no leaked mask or preview',!final.active && !final.mask && !final.overlay,{active:final.active,mask:final.mask,overlay:final.overlay},{active:false,mask:null,overlay:null});
    evidence.inputAudit={events:final.events,windowEvents:final.windowEvents,underlyingEvents:final.underlying};
  });
}
async function portIsFree() {
  return new Promise(resolve=>{
    const server=net.createServer();server.once('error',()=>resolve(false));server.listen({host:'127.0.0.1',port,exclusive:true},()=>server.close(()=>resolve(true)));
  });
}
async function closeOwned() {
  const cleanup={target:'Only the Chrome process spawned by this runner and its verified owned CDP browser',protectedProcessesSignalled:false,protectedGuiRequested:false};
  if(chrome) {
    invariant(!PROTECTED_PIDS.has(chrome.pid),'Refuse cleanup of protected PID');
    if(ownershipVerified && browser?.socket.readyState===WebSocket.OPEN) {
      try {cleanup.lastOwnedProcesses=(await browser.send('SystemInfo.getProcessInfo')).processInfo;} catch(error){cleanup.processInfoError=String(error);}
      try {await browser.send('Browser.close');cleanup.browserClose='sent to profile/PID/port-verified owned browser';} catch(error){cleanup.browserClose=String(error);}
    }
    const deadline=Date.now()+7000;
    while(!chromeExitInfo && Date.now()<deadline)await delay(50);
    if(!chromeExitInfo) {cleanup.spawnedChildKill=chrome.kill();await Promise.race([chromeExit,delay(1000)]);}
    cleanup.exit=chromeExitInfo || {pending:true};
    if(browser?.socket.readyState===WebSocket.OPEN)browser.socket.close();
    if(port){const freeDeadline=Date.now()+4000;do{cleanup.debugPortReleased=await portIsFree();if(cleanup.debugPortReleased)break;await delay(50);}while(Date.now()<freeDeadline);}
    cleanup.pass=!!chromeExitInfo && cleanup.debugPortReleased===true;
    if(!cleanup.pass)evidence.errors.push('Owned Chrome exit or debug-port release was not confirmed');
  } else cleanup.pass=true;
  evidence.cleanup=cleanup;
}
function makeText() {
  const lines=[
    `Round111 isolated installed InspectorChatPicker browser acceptance: ${evidence.status.toUpperCase()}`,
    `Started: ${evidence.startedAt}`,`Finished: ${evidence.finishedAt}`,`Node: ${process.version}`,
    `Chrome: ${evidence.browser.version?.product || 'not available'}`,`Owned Chrome PID: ${evidence.browser.launch?.pid || 'not launched'}`,
    `Scope: ${evidence.scope.description}`,`Fixture: ${FIXTURE}`,`Unique profile: ${PROFILE}`,
    `Installed bundle SHA256: ${evidence.extraction.installedBundleSha256 || 'not extracted'}`,
    `Applied installed picker CSS SHA256: ${evidence.extraction.css?.appliedCssSha256 || 'not extracted'}`,
    '', 'Verbatim installed regions:'
  ];
  for(const region of evidence.extraction.regions || [])lines.push(`  ${region.name}: lines ${region.startLine}-${region.endLine}; SHA256 ${region.sha256}`);
  for(const rule of evidence.extraction.css?.rules || [])lines.push(`  Exact CSS: ${rule.verbatim}`);
  lines.push('','Browser cases:');
  for(const entry of evidence.cases){lines.push(`  ${entry.status.toUpperCase()} ${entry.id}: ${entry.assertions.filter(item=>item.pass).length}/${entry.assertions.length} assertions`);for(const item of entry.assertions.filter(item=>!item.pass))lines.push(`    FAIL ${item.name}: actual=${JSON.stringify(item.actual)} expected=${JSON.stringify(item.expected)}`);if(entry.error)lines.push(entry.error);if(entry.reason)lines.push(`    ${entry.reason}`);}
  lines.push('',`Owned Chrome cleanup: ${JSON.stringify(evidence.cleanup)}`,'','Remaining scope limits:',...evidence.limitations.map(value=>`  - ${value}`),'','Errors:',...(evidence.errors.length?evidence.errors:['  none']),'',`Full per-case measured values and trusted input events: ${OUTPUT.json}`,`Screenshot: ${OUTPUT.screenshot}`,'');
  return lines.join('\n');
}
async function main() {
  await checkScope();
  try {
    sourceBefore=await fs.readFile(INSTALLED,'utf8');
    const extracted=extractInstalled(sourceBefore),fixture=makeFixture(extracted);
    await writeOwned(FIXTURE,fixture);evidence.fixtureSha256=hash(fixture);evidence.node=process.version;
    await startBrowser();await acceptance();
  } catch(error) {evidence.errors.push(error.stack || String(error));console.error(error.stack || error);}
  finally {try{await closeOwned();}catch(error){evidence.errors.push(`Cleanup: ${error.stack || error}`);}}
  evidence.finishedAt=new Date().toISOString();
  evidence.summary={passedCases:evidence.cases.filter(entry=>entry.status==='passed').length,failedCases:evidence.cases.filter(entry=>entry.status==='failed').length,notVerifiedCases:evidence.cases.filter(entry=>entry.status==='not_verified').length,passedAssertions:evidence.cases.flatMap(entry=>entry.assertions).filter(item=>item.pass).length,failedAssertions:evidence.cases.flatMap(entry=>entry.assertions).filter(item=>!item.pass).length};
  evidence.status=evidence.errors.length===0 && evidence.summary.failedCases===0 && evidence.summary.passedCases>=11 && evidence.cleanup?.pass ? 'passed':'failed';
  await writeOwned(OUTPUT.text,makeText());
  if(!evidence.writes.includes(OUTPUT.json))evidence.writes.push(OUTPUT.json);
  await writeOwned(OUTPUT.json,JSON.stringify(evidence,null,2)+'\n');
  console.log(JSON.stringify({status:evidence.status,...evidence.summary,chrome:evidence.browser.version?.product,cleanup:evidence.cleanup?.pass,evidence:OUTPUT.json}));
  process.exitCode=evidence.status==='passed'?0:1;
}
await main();
