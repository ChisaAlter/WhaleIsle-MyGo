#!/usr/bin/env node
/**
 * Round112: run the exact installed InspectorChatPicker in an isolated real Chrome.
 * No packages, application server, original GUI/profile, or Go tooling are used.
 * The only writable paths are this script's fixed evidence/fixture paths and a
 * fresh child of .devdata/r112-nested-picker-profiles. All browser input uses CDP Input.
 */
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { createHash, randomUUID } from 'node:crypto';
import net from 'node:net';
import vm from 'node:vm';
import { spawn } from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts', 'inspector-reference-nested-tools.mjs');
const INSTALLED = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness\\packages\\experimental\\session-inspector\\lib\\client.js';
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const RUN_ID = `${new Date().toISOString().replace(/[:.]/g, '-')}-${randomUUID().slice(0, 8)}`;
const REFERENCE_RUNNER = path.join(ROOT, 'scripts', 'inspector-reference-picker-browser.mjs');
const INSTALLED_TREE = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness\\packages\\client\\ui-tool\\lib\\types\\client\\tool\\ToolCallTree.js';
const FIXTURE = path.join(ROOT, '.devdata', `r112-nested-picker-${RUN_ID}.html`);
const PROFILE_BASE = path.join(ROOT, '.devdata', 'r112-nested-picker-profiles');
const PREFIX = path.join(ROOT, 'docs', 'review-20261022', 'evidence', `inspector-pass256-reference-nested-tools-${RUN_ID}`);
const OUTPUT = { json:`${PREFIX}.json`, text:`${PREFIX}.txt`, leaf:`${PREFIX}-leaf.png`, subtree:`${PREFIX}-subtree.png`, root:`${PREFIX}-root.png`, clipped:`${PREFIX}-clipped.png` };
const PROFILE = path.join(PROFILE_BASE, RUN_ID);
const PROTECTED_PIDS = new Set([20012, 16008]);
const PROTECTED_PORTS = new Set([20012, 16008, 3080]);
const allowedFiles = new Set([FIXTURE, ...Object.values(OUTPUT)].map(normalize));
const evidence = {
  schema: 'inspector-pass256-reference-nested-tools-browser/v1', round: 112, runId: RUN_ID,
  startedAt: new Date().toISOString(), status: 'running',
  scope: {
    description: 'Exact installed InspectorChatPicker, seven verbatim installed functions and exact installed picker CSS in a controlled nested ToolCallTree-style DOM, file URL in separate installed Chrome. Additional original Log pick and forward-target functions run separately in a bounded Node VM.',
    workspace: ROOT, script: SCRIPT, installedBundle: INSTALLED, fixture: FIXTURE,
    uniqueUserDataDir: PROFILE, allowedProfileRoot: PROFILE_BASE,
    protected: { electronPid: 20012, backendPid: 16008, gui: 'http://127.0.0.1:3080', policy: 'Never signal, restart, navigate, read profile, or mutate. No operation in this runner targets these processes, port, or profiles.' },
    dependencies: 'Node >=24 built-ins, installed Chrome executable; no dependency install, builds, Go tests, model calls, application server, or external fetch.',
    input: 'CDP Input.dispatchMouseEvent and Input.dispatchKeyEvent only. No DOM dispatchEvent or direct picker event-handler calls.',
    network: 'The runner connects only to the allocated 127.0.0.1 CDP endpoint. Fixture CSP denies network; page Network.setBlockedURLs blocks HTTP(S)/WS(S). Chrome background networking is disabled and DNS resolution is blocked.'
  },
  writes: [], extraction: {}, browser: {}, cases: [], errors: [], attempts: {
    policy: 'Each invocation uses a new evidence/fixture/profile runId and runs each case once. Failed attempts remain under their original runId. No passing run is repeated.'
  }, limitations: [
    'Controlled DOM reproduces original ToolCallTree wrapper containment, not full app, live data, original GUI, React rendering, or Electron Chromium.',
    'Seven functions, POINTER_ACTIONS, picker class and exact picker CSS execute in Chrome. Original Log pick method and forward-target functions execute separately in Node VM with fixture source state; full SessionLogModel integration is outside scope.',
    'Original Log reverse pick only prioritizes exact tool/call. Nested PTC-start rows fall back to owning root anchor; no child PTC reverse-selection semantics are added.',
    'Shift-wheel, touch, platform menus, window-blur and live application lifecycle are outside this nested-call evidence scope.'
  ]
};
let browser, page, chrome, chromeExit, chromeExitInfo, port, ownershipVerified = false;
let sourceBefore, vmFunctions;
const immutableBefore = {};
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
  for (const target of [ROOT, SCRIPT, REFERENCE_RUNNER, INSTALLED, INSTALLED_TREE, CHROME, FIXTURE, PROFILE_BASE, ...Object.values(OUTPUT)]) await noLinks(target);
  for (const directory of [ROOT, path.dirname(SCRIPT), path.dirname(PREFIX), path.dirname(FIXTURE)]) {
    invariant(normalize(await fs.realpath(directory)) === normalize(directory), `Refuse redirected directory: ${directory}`);
  }
  invariant(normalize(PROFILE).startsWith(`${normalize(PROFILE_BASE)}${path.sep}`), 'Profile is outside its owned root');
  for (const target of [PROFILE, FIXTURE, ...Object.values(OUTPUT)]) {
    try { await fs.lstat(target); throw new Error(`New output/profile already exists; refuse reuse: ${target}`); } catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
}
async function writeOwned(target, value) {
  invariant(allowedFiles.has(normalize(target)), `Refuse out-of-scope write: ${target}`);
  await noLinks(target);
  try { await fs.readFile(target); } catch (error) { if (error.code !== 'ENOENT') throw error; }
  await fs.writeFile(target, value, {flag:'wx'});
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
  const functions = ['displayed', 'inChat', 'findChatTargets', 'matchChatElement', 'matchChatNodeRow', 'visibleChatRect', 'placeChatHighlight'].map(name => {
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
  const vmRegions = ['partOf', 'sessionLogChatTarget', 'nearest', 'primaryTarget', 'nodeTarget', 'targetKey', 'resolveChatRevealTargets'].map(name => extractLines(source, name, new RegExp(`^\\t\\tfunction ${name}\\(`), /^\t\t}\r?$/));
  const logClassLine = source.split('\n').findIndex(line => /^\t\tvar SessionLogModel = class \{/.test(line));
  invariant(logClassLine >= 0, 'Installed SessionLogModel not found');
  const logPick = extractLines(source, 'SessionLogModel.pick', /^\t\t\tpick\(target\) \{/, /^\t\t\t}\r?$/, logClassLine);
  vmRegions.push(logPick);
  vmFunctions = { regions:vmRegions, code:vmRegions.filter(region=>region!==logPick).map(region=>region.verbatim).join('\n'), pick:logPick.verbatim };
  for (const region of regions) invariant(!/<\/script/i.test(region.verbatim), `Unsafe inline script delimiter in ${region.name}`);
  evidence.extraction = {
    installedBundleSha256: hash(source), installedBundleUtf8Bytes: Buffer.byteLength(source),
    regions, nodeVmRegions:vmRegions, css: { sourceLine: source.slice(0, cssMatch.index).split('\n').length, generatedLiteralSha256: hash(cssMatch[1]), decodedCssSha256: hash(css), mapping, mappingRegions, rules, appliedCssSha256: hash(rules.map(rule => rule.verbatim).join('\n')) },
    method: 'Functions, POINTER_ACTIONS, and class sliced verbatim at exact original indentation boundaries. CSS decoded with JSON.parse from the installed generated literal; only the exact two installed picker rules are applied. The class and functions are not transformed, mocked, transpiled, or reimplemented.'
  };
  return { code: regions.map(region => region.verbatim).join('\n'), css: rules.map(rule => rule.verbatim).join('\n'), mapping };
}
function makeFixture(extracted) {
  return `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; connect-src 'none'; frame-src 'none'"><title>Round112 installed nested-tool picker</title>
<style>
:root{--dsw-alias-state-business-primary:#2f74c8;--dsw-radius-sm:4px;--dsw-alias-border-l2:#b7c5d8;color-scheme:light;font:15px Arial,sans-serif;color:#172235;background:#e9eef6}
*{box-sizing:border-box}body{margin:0;overflow:hidden}h1{font-size:19px;margin:0 0 6px}p{margin:0}#caption{position:fixed;left:40px;top:22px;right:30px}#caption p{font-size:13px;color:#58677f}
[data-slot="main.conversation"]{position:fixed;left:40px;right:320px;top:100px;bottom:60px;background:white;border-radius:8px}[data-slot="conversation.view"]{width:100%;height:100%}[data-conversation-scroll]{position:relative;width:100%;height:100%;overflow:auto;overflow-anchor:none;line-height:20px}[data-chat-flow]{position:relative;width:100%;height:1900px}
#turn{position:absolute;top:24px;left:24px;right:24px}.callRow{border-radius:var(--dsw-radius-sm)}.subCalls{display:flex;flex-direction:column;gap:4px;margin:4px 0 2px 22px;padding-left:8px;border-left:.5px solid var(--dsw-alias-border-l2)}
.toolCard{height:56px;background:#f1f4f9;border:1px solid #b7c5d8;border-radius:5px;padding:8px;display:flex;align-items:center}.toolCard button{width:100%;height:38px}button{font:inherit;padding:7px 12px;border:1px solid #a3b2c7;background:white;border-radius:5px;cursor:pointer}
#clipper{position:absolute;left:24px;right:24px;top:500px;height:220px;overflow:hidden;background:#dbe4f0}#clippedNode{position:relative;left:-18px;top:-28px;width:calc(100% + 36px);height:260px}#clipRoot,#clipChild{height:100%;width:100%;background:#edf3fc}#clipAction{margin:80px 50px;width:calc(100% - 100px);height:60px}
#offscreenNode{position:absolute;top:1400px;left:24px;right:24px;height:80px}#offscreen{height:100%;background:#f1f4f9}[data-composer-seat]{position:sticky;bottom:0;height:72px;margin-top:-72px;background:#d6e0ed;z-index:3;display:flex;align-items:center;justify-content:center;border-top:1px solid #b7c5d8}
#inspectorPane{position:fixed;right:24px;top:100px;width:264px;padding:18px;background:white;border-radius:8px}#inspectorPane p{font-size:13px;margin-top:14px;line-height:1.5}
</style><style data-installed-picker-css="${evidence.extraction.css.appliedCssSha256}">${extracted.css}</style></head><body>
<header id="caption"><h1>Installed nested-tool picker · isolated Chrome</h1><p>Outer Chat node tool:root. Identical labels; root → child → grandchild, plus sibling. Wrappers include descendants.</p></header>
<main data-slot="main.conversation"><div data-slot="conversation.view"><div id="scroller" data-conversation-scroll><div id="flow" data-chat-flow>
<div id="turn" data-chat-turn="7"><div id="group" data-chat-group-key="group-seven"><section id="rootNode" data-chat-node-key="tool:root" data-chat-flow-kind="tool-call">
<div id="rootCall" class="callRow" data-chat-anchor-key="call:root" data-chat-call-id="root"><div id="rootCard" class="toolCard"><button id="rootAction">Identical tool action</button></div><div id="rootSubcalls" class="subCalls" data-subcalls>
<div id="childCall" class="callRow" data-chat-anchor-key="call:child" data-chat-call-id="child"><div id="childCard" class="toolCard"><button id="childAction">Identical tool action</button></div><div id="childSubcalls" class="subCalls" data-subcalls>
<div id="grandchildCall" class="callRow" data-chat-anchor-key="call:grandchild" data-chat-call-id="grandchild"><div id="grandchildCard" class="toolCard"><button id="grandchildAction">Identical tool action</button></div></div></div></div>
<div id="siblingCall" class="callRow" data-chat-anchor-key="call:sibling" data-chat-call-id="sibling"><div id="siblingCard" class="toolCard"><button id="siblingAction">Identical tool action</button></div></div>
</div></div></section></div></div>
<div id="clipper" data-step-process-body><section id="clippedNode" data-chat-node-key="tool:clipped" data-chat-flow-kind="tool-call"><div id="clipRoot" data-chat-call-id="clip-root"><div id="clipChild" data-chat-call-id="clip-child"><button id="clipAction">Identical tool action</button></div></div></section></div>
<section id="offscreenNode" data-chat-node-key="tool:offscreen" data-chat-flow-kind="tool-call"><div id="offscreen" data-chat-call-id="offscreen"><button id="offscreenAction">Identical tool action</button></div></section>
</div><div id="composer" data-composer-seat><button id="composerAction">Composer</button></div></div></div></main>
<aside id="inspectorPane"><button id="owner">Inspector picker owner</button><p>Actual installed picker/functions/CSS. Layout reproduces recursive wrapper containment; all four tool labels are equal.</p><p>Leaf = exact box. Child = subtree. Root = full tree.</p></aside>
<script>
'use strict';
const inspector_module_css_default=${JSON.stringify(extracted.mapping)};
${extracted.code}
(() => {
  const flow=document.getElementById('flow'),owner=document.getElementById('owner'),scroller=document.getElementById('scroller');
  const rows=new Map([['node:tool:root:',{nodeKey:'tool:root',turn:7,step:2}],['group:group-seven',{groupKey:'group-seven',turn:7}]]);
  let active;
  const state={accepts:true,attempts:[],starts:0,releases:0,cancellations:0,events:[],underlying:[]},observed=[];
  for(const type of ['pointermove','pointerdown','pointerup','mousedown','mouseup','click','auxclick','contextmenu','keydown','keyup'])window.addEventListener(type,event=>{
    const entry={type,trusted:event.isTrusted,targetId:event.target.id||null,mask:event.target.hasAttribute?.('data-session-inspector-picker')||false,button:event.button??null,key:event.key??null,defaultPrevented:event.defaultPrevented};observed.push({event,entry});state.events.push(entry);
  },true);
  for(const id of ['rootAction','childAction','grandchildAction','siblingAction','clipAction','offscreenAction'])for(const type of ['pointerdown','pointerup','mousedown','mouseup','click'])document.getElementById(id).addEventListener(type,event=>state.underlying.push({id,type,trusted:event.isTrusted}));
  const rect=element=>{const b=element.getBoundingClientRect();return {left:b.left,top:b.top,right:b.right,bottom:b.bottom,width:b.width,height:b.height};};
  const styles=element=>{const s=getComputedStyle(element);return Object.fromEntries(['position','zIndex','pointerEvents','cursor','touchAction','backgroundColor','borderRadius','outlineWidth','outlineStyle','outlineColor'].map(k=>[k,s[k]]));};
  const ids=['flow','scroller','composer','turn','rootNode','rootCall','rootCard','rootAction','rootSubcalls','childCall','childCard','childAction','childSubcalls','grandchildCall','grandchildCard','grandchildAction','siblingCall','siblingCard','siblingAction','clipper','clippedNode','clipRoot','clipChild','clipAction','offscreenNode','offscreen'];
  function snapshot(){
    for(const {event,entry} of observed)entry.defaultPrevented=event.defaultPrevented;
    const mask=document.querySelector('[data-session-inspector-picker]'),overlay=document.querySelector('[data-session-inspector-highlight="pick"]');
    return {active:!!active?.mask,accepts:state.accepts,viewport:{width:innerWidth,height:innerHeight},counts:{attempts:state.attempts.length,starts:state.starts,releases:state.releases,cancellations:state.cancellations,events:state.events.length,underlying:state.underlying.length,underlyingClicks:state.underlying.filter(e=>e.type==='click').length},mask:mask?{rect:rect(mask),style:styles(mask),parent:mask.parentElement.tagName,className:mask.className,ariaHidden:mask.getAttribute('aria-hidden')}:null,overlay:overlay?{rect:rect(overlay),style:styles(overlay),parent:overlay.parentElement.tagName,className:overlay.className,ariaHidden:overlay.getAttribute('aria-hidden')}:null,rects:Object.fromEntries(ids.map(id=>[id,rect(document.getElementById(id))])),rootHTML:flow.outerHTML,layout:{scrollHeight:scroller.scrollHeight,scrollWidth:scroller.scrollWidth,clientHeight:scroller.clientHeight,clientWidth:scroller.clientWidth,flowOffsetWidth:flow.offsetWidth,flowOffsetHeight:flow.offsetHeight},scroll:{top:scroller.scrollTop,left:scroller.scrollLeft,windowX:scrollX,windowY:scrollY},attempts:state.attempts.slice(),events:state.events.slice(),underlying:state.underlying.slice()};
  }
  window.harness={ready:true,state,snapshot,reset(){if(active?.mask)active.dispose();scroller.scrollTop=0;scroller.scrollLeft=0;},arm(accepts){state.accepts=accepts;const picker=new InspectorChatPicker(()=>flow,owner,target=>{const row=matchChatNodeRow(target,rows),accepted=state.accepts&&row!==undefined;state.attempts.push({target,row:row??null,accepted});return accepted;},()=>state.cancellations++,()=>{state.releases++;if(active===picker)active=undefined;});active=picker;const started=picker.start();if(started)state.starts++;return started;},setAccepted(value){state.accepts=value;},diagnostic(id){const element=document.getElementById(id),match=matchChatElement(element,flow),box=visibleChatRect(element);return {match:match?{elementId:match.element.id,target:match.target}:null,visible:box??null};},find(targets){return findChatTargets(flow,targets).map(e=>({id:e.id,callId:e.dataset.chatCallId??null,nodeKey:e.dataset.chatNodeKey??null}));}};
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
  const {rootHTML,events,underlying,...rest}=snapshot;
  return {...rest,rootHTMLSha256:hash(rootHTML)};
}
async function runCase(id,action,{optional=false}={}) {
  const entry={id,execution:id.startsWith('node-vm-')?'Node VM only; not browser':'Real installed Chrome; CDP trusted input for interaction cases',startedAt:new Date().toISOString(),status:'running',assertions:[],actual:{}};
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

function containsRect(parent,child){return !!parent&&!!child&&parent.left<=child.left+0.75&&parent.top<=child.top+0.75&&parent.left+parent.width>=child.right-0.75&&parent.top+parent.height>=child.bottom-0.75;}
function expectedTarget(callId){return {nodeKey:'tool:root',nodeKind:'tool-call',callId,groupKey:'group-seven',turn:7};}
async function capture(name,entry){const shot=await page.send('Page.captureScreenshot',{format:'png',captureBeyondViewport:false,fromSurface:true});const bytes=Buffer.from(shot.data,'base64');await writeOwned(OUTPUT[name],bytes);entry.actual.screenshot={path:OUTPUT[name],sha256:hash(bytes),description:'Actual installed picker overlay in isolated nested fixture; not original GUI'};}
async function acceptance(){
  await runCase('nested-fixture-wrapper-shape-and-mask-no-layout-change',async(entry,check)=>{
    const before=await snap(),after=await prepare();entry.actual={before:compact(before),after:compact(after)};
    const dom=await evaluate("({labels:[...document.querySelectorAll('#rootNode button')].map(b=>b.textContent),ids:[...document.querySelectorAll('#rootNode [data-chat-call-id]')].map(e=>e.dataset.chatCallId),rootIncludes:document.getElementById('rootCall').contains(document.getElementById('siblingCall'))&&document.getElementById('rootCall').contains(document.getElementById('grandchildCall')),childIncludes:document.getElementById('childCall').contains(document.getElementById('grandchildCall')),nodeKeys:[...document.querySelectorAll('#rootNode [data-chat-call-id]')].map(e=>e.closest('[data-chat-node-key]').dataset.chatNodeKey)})");entry.actual.dom=dom;
    check('four equal labels cannot disambiguate identities',dom.labels.length===4&&new Set(dom.labels).size===1,dom.labels,'four identical labels');
    check('root/child/grandchild/sibling share outer native-alias Chat node tool:root',JSON.stringify(dom.ids)===JSON.stringify(['root','child','grandchild','sibling'])&&dom.nodeKeys.every(k=>k==='tool:root'),dom,'four CallIDs; outer nodeKey tool:root');
    check('wrappers include descendants as original ToolCallTree',dom.rootIncludes&&dom.childIncludes&&containsRect(after.rects.childCall,after.rects.grandchildCall)&&containsRect(after.rects.rootCall,after.rects.siblingCall),{dom,rects:after.rects},'child contains grandchild; root contains all');
    check('installed mask is fixed body child over visible transcript',after.mask?.parent==='BODY'&&after.mask.className===evidence.extraction.css.mapping.chatPickMask&&after.mask.style.position==='fixed'&&after.mask.style.zIndex==='1200'&&sameRect(after.mask.rect,expectedRect(after,'flow')),after.mask,expectedRect(after,'flow'));
    check('arming leaves flow markup, layout and nested boxes unchanged',before.rootHTML===after.rootHTML&&JSON.stringify(before.layout)===JSON.stringify(after.layout)&&['rootCall','childCall','grandchildCall','siblingCall'].every(id=>sameRect(before.rects[id],after.rects[id])),{beforeHash:hash(before.rootHTML),afterHash:hash(after.rootHTML),beforeLayout:before.layout,afterLayout:after.layout},'identical');
  });
  for(const spec of [{id:'grandchild-leaf',call:'grandchild',box:'grandchildCall',action:'grandchildAction',shot:'leaf'},{id:'child-subtree',call:'child',box:'childCall',action:'childAction',shot:'subtree'},{id:'root-full-tree',call:'root',box:'rootCall',action:'rootAction',shot:'root'},{id:'sibling-leaf',call:'sibling',box:'siblingCall',action:'siblingAction'}]){
    await runCase(spec.id+'-preview-callid-and-chat-root-record',async(entry,check)=>{
      const before=await prepare(),point=center(before.rects[spec.action]);await move(point);const hovered=await snap();
      const diagnostic=await evaluate('window.harness.diagnostic('+JSON.stringify(spec.action)+')');entry.actual={before:compact(before),hovered:compact(hovered),point,diagnostic};
      check('actual hover preview equals exact nearest CallID wrapper',sameRect(hovered.overlay?.rect,expectedRect(hovered,spec.box)),hovered.overlay?.rect,expectedRect(hovered,spec.box));
      check('original matcher retains exact CallID plus same outer nodeKey',diagnostic.match?.elementId===spec.box&&JSON.stringify(diagnostic.match.target)===JSON.stringify(expectedTarget(spec.call)),diagnostic.match,{elementId:spec.box,target:expectedTarget(spec.call)});
      if(spec.call==='child')check('child preview includes grandchild and excludes sibling',containsRect(hovered.overlay.rect,hovered.rects.grandchildCall)&&hovered.overlay.rect.top+hovered.overlay.rect.height<=hovered.rects.siblingCall.top,hovered.overlay.rect,{grandchild:hovered.rects.grandchildCall,sibling:hovered.rects.siblingCall});
      if(spec.call==='root')check('root preview includes full child/grandchild/sibling tree',['childCall','grandchildCall','siblingCall'].every(id=>containsRect(hovered.overlay.rect,hovered.rects[id])),hovered.overlay.rect,['childCall','grandchildCall','siblingCall'].map(id=>hovered.rects[id]));
      check('installed preview is above mask and ignores hit tests',hovered.overlay?.parent==='BODY'&&hovered.overlay.style.position==='fixed'&&hovered.overlay.style.pointerEvents==='none'&&hovered.overlay.style.zIndex==='1201'&&hovered.overlay.ariaHidden==='true'&&hovered.overlay.className===evidence.extraction.css.mapping.chatPick,hovered.overlay,{position:'fixed',pointerEvents:'none',zIndex:'1201',ariaHidden:'true'});
      if(spec.shot)await capture(spec.shot,entry);
      await click(point);const after=await snap(),attempt=after.attempts.at(-1),events=after.events.slice(before.counts.events);entry.actual.after=compact(after);entry.actual.events=events;
      check('actual click returns exact CallID and tool:root nodeKey',after.counts.attempts===before.counts.attempts+1&&JSON.stringify(attempt?.target)===JSON.stringify(expectedTarget(spec.call)),attempt,expectedTarget(spec.call));
      check('original matchChatNodeRow selects owning root Chat row',attempt?.row==='node:tool:root:'&&attempt.accepted===true,attempt,{row:'node:tool:root:',accepted:true});
      check('trusted native click is prevented and underlying action suppressed',events.some(e=>e.type==='click'&&e.trusted&&e.mask&&e.defaultPrevented)&&after.counts.underlying===before.counts.underlying,{events,underlyingDelta:after.counts.underlying-before.counts.underlying},'trusted prevented mask click; zero underlying events');
      check('accepted pick releases exactly once without layout change',!after.active&&!after.mask&&!after.overlay&&after.counts.releases===before.counts.releases+1&&after.counts.cancellations===before.counts.cancellations&&before.rootHTML===after.rootHTML&&JSON.stringify(before.layout)===JSON.stringify(after.layout),{after:compact(after),beforeHash:hash(before.rootHTML)},'one release, zero cancel, unchanged markup/layout');
    });
  }
  await runCase('nested-gaps-use-nearest-containing-parent',async(entry,check)=>{
    const before=await prepare(false),points=[{name:'child indentation gutter',call:'child',box:'childCall',point:{x:before.rects.childSubcalls.left+3,y:center(before.rects.grandchildCall).y}},{name:'root gap between child subtree and sibling',call:'root',box:'rootCall',point:{x:center(before.rects.siblingCall).x,y:(before.rects.childCall.bottom+before.rects.siblingCall.top)/2}},{name:'root gap before first child',call:'root',box:'rootCall',point:{x:center(before.rects.childCall).x,y:(before.rects.rootCard.bottom+before.rects.childCall.top)/2}}];
    entry.actual={before:compact(before),samples:[]};
    for(const sample of points){await move(sample.point);const hovered=await snap();await click(sample.point);const after=await snap(),attempt=after.attempts.at(-1);entry.actual.samples.push({...sample,preview:hovered.overlay,attempt,after:compact(after)});check(sample.name+' previews containing parent',sameRect(hovered.overlay?.rect,expectedRect(hovered,sample.box)),hovered.overlay?.rect,expectedRect(hovered,sample.box));check(sample.name+' click keeps parent CallID and root nodeKey',attempt?.target.callId===sample.call&&attempt.target.nodeKey==='tool:root'&&attempt.row==='node:tool:root:'&&!attempt.accepted&&after.active,attempt,{callId:sample.call,nodeKey:'tool:root',row:'node:tool:root:',accepted:false});}
    const after=await snap();entry.actual.after=compact(after);check('three rejected gaps keep gesture and suppress underlying events',after.counts.attempts===before.counts.attempts+3&&after.counts.underlying===before.counts.underlying&&after.counts.releases===before.counts.releases&&after.active,{before:before.counts,after:after.counts,active:after.active},'three attempts; zero underlying/release; active');await escapeKey();
  });
  await rejectionAndClipping();
  await runVmMatchers();
  await forwardAndIntegrity();
}

async function rejectionAndClipping(){
  await runCase('rejected-child-stays-armed-then-grandchild-accepts',async(entry,check)=>{
    const before=await prepare(false);await click(center(before.rects.childAction));const rejected=await snap();await evaluate('window.harness.setAccepted(true)');await click(center(rejected.rects.grandchildAction));const after=await snap();entry.actual={before:compact(before),rejected:compact(rejected),after:compact(after),events:after.events.slice(before.counts.events)};
    check('rejected child keeps mask and does not release',rejected.active&&!!rejected.mask&&rejected.attempts.at(-1)?.target.callId==='child'&&!rejected.attempts.at(-1).accepted&&rejected.counts.releases===before.counts.releases,{active:rejected.active,attempt:rejected.attempts.at(-1),releaseDelta:rejected.counts.releases-before.counts.releases},'rejected child; active mask; zero releases');
    check('next grandchild click completes same gesture with distinct CallID',after.counts.starts===before.counts.starts&&after.counts.attempts===before.counts.attempts+2&&after.attempts.at(-1)?.target.callId==='grandchild'&&after.attempts.at(-1).target.nodeKey==='tool:root'&&after.attempts.at(-1).accepted&&after.counts.releases===before.counts.releases+1&&!after.active,after.attempts.slice(before.counts.attempts),'same start; rejected child then accepted grandchild; one release');
    const clicks=after.events.slice(before.counts.events).filter(e=>e.type==='click');check('both clicks trusted/prevented and underlying action suppressed',clicks.length===2&&clicks.every(e=>e.trusted&&e.mask&&e.defaultPrevented)&&after.counts.underlying===before.counts.underlying,{clicks,underlyingDelta:after.counts.underlying-before.counts.underlying},'two trusted prevented clicks; zero underlying');
    await click(center(after.rects.grandchildAction));const released=await snap();entry.actual.afterReleasedControl=compact(released);check('same underlying control works after mask release',released.counts.underlyingClicks===after.counts.underlyingClicks+1&&released.underlying.slice(after.counts.underlying).every(e=>e.trusted),released.underlying.slice(after.counts.underlying),'one trusted underlying click');
  });
  await runCase('clipped-nested-call-offscreen-and-no-layout-change',async(entry,check)=>{
    const before=await prepare(false),expected=expectedRect(before,'clipChild',true);invariant(expected.width>0&&expected.height>0,'Clipped nested target is not visible');await move(center(expected));const after=await snap(),offscreen=await evaluate("window.harness.diagnostic('offscreen')"),clipped=await evaluate("window.harness.diagnostic('clipChild')");entry.actual={before:compact(before),after:compact(after),expectedIndependentIntersection:expected,offscreen,clipped};
    check('actual clip-child preview equals independent intersection',sameRect(after.overlay?.rect,expected),after.overlay?.rect,expected);
    check('example clips both axes and lower edge at composer',expected.width<before.rects.clipChild.width&&expected.height<before.rects.clipChild.height&&near(expected.bottom,before.rects.composer.top),{expected,unclipped:before.rects.clipChild,composer:before.rects.composer},'smaller width/height; lower edge at composer');
    check('clipped child exact CallID retains owning nodeKey',clipped.match?.target.callId==='clip-child'&&clipped.match.target.nodeKey==='tool:clipped'&&clipped.match.elementId==='clipChild',clipped,{callId:'clip-child',nodeKey:'tool:clipped',elementId:'clipChild'});
    check('mounted offscreen identity has no visible rectangle',offscreen.match?.target.callId==='offscreen'&&offscreen.visible===null&&before.rects.offscreen.top>before.viewport.height,offscreen,'mounted identity; visibleChatRect undefined; below viewport');
    check('hover/diagnostics do not change content layout or scroll',before.rootHTML===after.rootHTML&&JSON.stringify(before.layout)===JSON.stringify(after.layout)&&JSON.stringify(before.scroll)===JSON.stringify(after.scroll),{beforeHash:hash(before.rootHTML),afterHash:hash(after.rootHTML),beforeLayout:before.layout,afterLayout:after.layout,beforeScroll:before.scroll,afterScroll:after.scroll},'identical');
    const style=after.overlay?.style,parsed=style?.backgroundColor.match(/^color\(srgb\s+([\d.]+)\s+([\d.]+)\s+([\d.]+)\s*\/\s*([\d.]+)\)$/),channels=parsed?.slice(1).map(Number);check('installed CSS computes theme 15% fill and 1px outline',channels?.every((v,i)=>Math.abs(v-[47/255,116/255,200/255,0.15][i])<0.00001)&&style.borderRadius==='4px'&&style.outlineWidth==='1px'&&style.outlineStyle==='solid'&&style.outlineColor==='rgb(47, 116, 200)',style,{fill:[47/255,116/255,200/255,0.15],radius:'4px',outline:'1px solid rgb(47, 116, 200)'});
    await capture('clipped',entry);await escapeKey();
  });
}
async function runVmMatchers(){
  await runCase('node-vm-original-log-pick-root-anchor-fallback-with-ptc-start',async(entry,check)=>{
    const context=vm.createContext({}, {codeGeneration:{strings:false,wasm:false}}),definitions=vmFunctions.code+'\nclass OriginalLogPick {\n'+vmFunctions.pick+'\n}\n({OriginalLogPick,sessionLogChatTarget,resolveChatRevealTargets})';
    const original=new vm.Script(definitions,{filename:'installed-session-inspector-verbatim-node-vm.js'}).runInContext(context,{timeout:1000});
    const settled=(seq,type,data)=>({type:'settled',event:{seq,type,data}}),entries=[settled(90,'turn/start',{turn:7}),settled(95,'step/start',{turn:7,step:2}),settled(100,'tool/call',{callId:'root',name:'identical_tool',arguments:'{}',turn:7,step:2}),settled(110,'tool/ptc-dispatch-start',{parentCallId:'root',rootCallId:'root',subCallId:'child',name:'identical_tool',arguments:{}}),settled(120,'tool/ptc-dispatch-start',{parentCallId:'child',rootCallId:'root',subCallId:'grandchild',name:'identical_tool',arguments:{}}),settled(130,'tool/ptc-dispatch-start',{parentCallId:'root',rootCallId:'root',subCallId:'sibling',name:'identical_tool',arguments:{}})];
    const model=new original.OriginalLogPick();let accepts=0;model.source={getSnapshot:()=>({entries})};model.accept=()=>{accepts++;};model.settledRoots=new Map(entries.map(e=>[e.event.seq,'event:'+e.event.seq]));model.rows={getSnapshot:()=>[]};model.records=new Map();
    const picked=(await snap()).attempts.filter(a=>['child','grandchild','sibling'].includes(a.target.callId)),targets=['child','grandchild','sibling'].map(callId=>({...picked.find(a=>a.target.callId===callId).target,anchorSeq:100,step:2}));
    context.model=model;context.targets=targets;const reverse=new vm.Script('targets.map(target=>({target,key:model.pick(target)}))').runInContext(context,{timeout:1000});entry.actual={entries:structuredClone(entries),targets,reverse,acceptCalls:accepts,execution:'Node VM only; original pick method with fixture source/accept/settledRoots/rows/records, not full model or browser'};
    check('PTC-start children exist without child tool/call',entries.filter(e=>e.event.type==='tool/ptc-dispatch-start').length===3&&['child','grandchild','sibling'].every(id=>!entries.some(e=>e.event.type==='tool/call'&&e.event.data.callId===id)),entries,'three nested PTC-start entries; only root tool/call');
    check('original Log matcher chooses root anchor for all nested CallIDs',reverse.length===3&&reverse.every(r=>r.key==='event:100'),reverse,'event:100 for child/grandchild/sibling');
    check('original matcher accepts source once per selection',accepts===3,accepts,3);
    const exact=settled(150,'tool/call',{callId:'child',turn:7,step:2,name:'identical_tool',arguments:'{}'});entries.push(exact);model.settledRoots.set(150,'event:150');context.target=targets[0];const exactControl=new vm.Script('model.pick(target)').runInContext(context,{timeout:1000});entries.pop();entry.actual.exactToolCallControl={entry:exact,result:exactControl};check('exact tool/call priority still precedes root anchor',exactControl==='event:150',exactControl,'event:150');
    const rootNode={key:'tool:root',kind:'tool-call',visibility:'visible',anchorSeq:100,location:{kind:'step',turn:{turn:7},step:{step:2}},data:{root:{callId:'root',subCalls:[{callId:'child',subCalls:[{callId:'grandchild',subCalls:[]}]},{callId:'sibling',subCalls:[]}]}}};context.ptc=entries.find(e=>e.event.seq===110).event;context.rootNode=rootNode;context.sessionLogChatTarget=original.sessionLogChatTarget;context.resolveChatRevealTargets=original.resolveChatRevealTargets;
    const forward=new vm.Script('(()=>{const target=sessionLogChatTarget({event:ptc});return {target,resolved:resolveChatRevealTargets(target,[rootNode])};})()').runInContext(context,{timeout:1000});entry.actual.forward=forward;evidence.nodeVm={execution:'Node VM only',forwardPtcTarget:forward.target,forwardResolved:forward.resolved,reverse};
    check('original forward Log target retains PTC child/root identity',forward.target.anchorSeq===110&&forward.target.callId==='child'&&forward.target.rootCallId==='root',forward.target,{anchorSeq:110,callId:'child',rootCallId:'root'});
    check('original forward resolver retains child and adds root nodeKey',forward.resolved[0]?.callId==='child'&&forward.resolved[0].rootCallId==='root'&&forward.resolved[0].nodeKey==='tool:root',forward.resolved,{callId:'child',rootCallId:'root',nodeKey:'tool:root'});
  });
}
async function forwardAndIntegrity(){
  await runCase('forward-original-findChatTargets-exact-child-before-root',async(entry,check)=>{
    const before=await snap(),targets=evidence.nodeVm.forwardResolved,found=await evaluate('window.harness.find('+JSON.stringify(targets)+')');entry.actual={targets,found};
    check('first forward DOM target is exact child wrapper',found[0]?.id==='childCall'&&found[0].callId==='child',found,'childCall first');
    check('exact child precedes root Chat node and root-call fallback',found.findIndex(e=>e.id==='childCall')<found.findIndex(e=>e.id==='rootNode')&&found.findIndex(e=>e.id==='childCall')<found.findIndex(e=>e.id==='rootCall'),found,'childCall before rootNode and rootCall');
    const after=await snap();check('forward enumeration does not mutate or scroll Chat',before.rootHTML===after.rootHTML&&JSON.stringify(before.scroll)===JSON.stringify(after.scroll),{before:before.scroll,after:after.scroll},'unchanged');
  });
  await runCase('page-input-and-original-source-integrity',async(entry,check)=>{
    const final=await snap(),external=requests.filter(r=>/^(https?|wss?):/i.test(r.url)),after={};for(const file of [INSTALLED,INSTALLED_TREE,REFERENCE_RUNNER,SCRIPT])after[file]=hash(await fs.readFile(file));entry.actual={final:compact(final),pageExceptions,protocolErrors,requests,immutableBefore,immutableAfter:after};
    check('installed bundle/tree and both scripts unchanged during execution',Object.keys(immutableBefore).every(file=>immutableBefore[file]===after[file]),{before:immutableBefore,after},'identical SHA256');
    check('no remote requests runtime exceptions or protocol errors',external.length===0&&pageExceptions.length===0&&protocolErrors.length===0,{external,pageExceptions,protocolErrors},'none');
    check('all recorded input is trusted Chrome events',final.events.length>0&&final.events.every(e=>e.trusted),{eventCount:final.events.length,untrusted:final.events.filter(e=>!e.trusted)},'all isTrusted=true');
    check('final gesture has no mask or preview',!final.active&&!final.mask&&!final.overlay,{active:final.active,mask:final.mask,overlay:final.overlay},'cleared');evidence.inputAudit={events:final.events,underlyingEvents:final.underlying};
  });
}

async function portIsFree() {
  return new Promise(resolve=>{
    const server=net.createServer();server.once('error',()=>resolve(false));server.listen({host:'127.0.0.1',port,exclusive:true},()=>server.close(()=>resolve(true)));
  });
}
async function closeOwned(){
  const cleanup={target:'Only own spawned Chrome and verified own CDP process IDs/profile/port',protectedProcessesSignalled:false,protectedGuiRequested:false};
  if(chrome){
    invariant(!PROTECTED_PIDS.has(chrome.pid),'Refuse cleanup of protected PID');
    if(ownershipVerified&&browser?.socket.readyState===WebSocket.OPEN){
      try{cleanup.lastOwnedProcesses=(await browser.send('SystemInfo.getProcessInfo')).processInfo;invariant(cleanup.lastOwnedProcesses.every(p=>!PROTECTED_PIDS.has(p.id)),'Protected PID in own-process list');}catch(error){cleanup.processInfoError=String(error);}
      try{await browser.send('Browser.close');cleanup.browserClose='sent only to profile/PID/port-verified own browser';}catch(error){cleanup.browserClose=String(error);}
    }
    const deadline=Date.now()+7000;while(!chromeExitInfo&&Date.now()<deadline)await delay(50);
    if(!chromeExitInfo){cleanup.spawnedChildKill=chrome.kill();await Promise.race([chromeExit,delay(1000)]);}
    cleanup.exit=chromeExitInfo||{pending:true};if(browser?.socket.readyState===WebSocket.OPEN)browser.socket.close();
    if(port){const deadline=Date.now()+4000;do{cleanup.debugPortReleased=await portIsFree();if(cleanup.debugPortReleased)break;await delay(50);}while(Date.now()<deadline);}
    const owned=[...new Set([...(cleanup.lastOwnedProcesses||[]),...(evidence.browser.initialOwnedProcesses||[])].map(p=>p.id))];
    const alive=id=>{invariant(!PROTECTED_PIDS.has(id),'Refuse protected PID existence probe');try{process.kill(id,0);return true;}catch(error){if(error.code==='ESRCH')return false;throw error;}};
    const deadlineProcesses=Date.now()+4000;do{cleanup.ownedProcessLiveness=owned.map(id=>({pid:id,alive:alive(id)}));if(cleanup.ownedProcessLiveness.every(p=>!p.alive))break;await delay(50);}while(Date.now()<deadlineProcesses);
    cleanup.allObservedOwnedProcessesExited=cleanup.ownedProcessLiveness.length>0&&cleanup.ownedProcessLiveness.every(p=>!p.alive);
    if(chromeExitInfo&&cleanup.debugPortReleased&&cleanup.allObservedOwnedProcessesExited){
      invariant(normalize(PROFILE).startsWith(normalize(PROFILE_BASE)+path.sep)&&path.basename(PROFILE)===RUN_ID,'Profile cleanup target is not exact own run path');await noLinks(PROFILE);invariant(normalize(await fs.realpath(PROFILE))===normalize(PROFILE),'Profile cleanup target redirected');cleanup.profileCleanupTarget=path.resolve(PROFILE);await fs.rm(PROFILE,{recursive:true,force:false});cleanup.ownedProfileRemoved=true;
    }
    cleanup.pass=!!chromeExitInfo&&cleanup.debugPortReleased===true&&cleanup.allObservedOwnedProcessesExited&&cleanup.ownedProfileRemoved===true;
    if(!cleanup.pass)evidence.errors.push('Own Chrome processes, debug port or profile cleanup not confirmed');
  }else cleanup.pass=true;
  evidence.cleanup=cleanup;
}
function makeText() {
  const lines=[
    `Round112 isolated installed nested-tool picker acceptance: ${evidence.status.toUpperCase()}`,
    `Started: ${evidence.startedAt}`,`Finished: ${evidence.finishedAt}`,`Node: ${process.version}`,
    `Chrome: ${evidence.browser.version?.product || 'not available'}`,`Owned Chrome PID: ${evidence.browser.launch?.pid || 'not launched'}`,
    `Scope: ${evidence.scope.description}`,`Fixture: ${FIXTURE}`,`Unique profile: ${PROFILE}`,
    `Installed bundle SHA256: ${evidence.extraction.installedBundleSha256 || 'not extracted'}`,
    `Applied installed picker CSS SHA256: ${evidence.extraction.css?.appliedCssSha256 || 'not extracted'}`,
    '', 'Verbatim installed regions:'
  ];
  for(const region of evidence.extraction.regions || [])lines.push(`  ${region.name}: lines ${region.startLine}-${region.endLine}; SHA256 ${region.sha256}`);
  for(const region of evidence.extraction.nodeVmRegions || [])lines.push('  Node VM ONLY '+region.name+': lines '+region.startLine+'-'+region.endLine+'; SHA256 '+region.sha256);
  for(const rule of evidence.extraction.css?.rules || [])lines.push(`  Exact CSS: ${rule.verbatim}`);
  lines.push('','Cases (Node VM separately labelled):');
  for(const entry of evidence.cases){lines.push(`  ${entry.status.toUpperCase()} ${entry.id}: ${entry.assertions.filter(item=>item.pass).length}/${entry.assertions.length} assertions`);for(const item of entry.assertions.filter(item=>!item.pass))lines.push(`    FAIL ${item.name}: actual=${JSON.stringify(item.actual)} expected=${JSON.stringify(item.expected)}`);if(entry.error)lines.push(entry.error);if(entry.reason)lines.push(`    ${entry.reason}`);}
  lines.push('',`Owned Chrome cleanup: ${JSON.stringify(evidence.cleanup)}`,'','Remaining scope limits:',...evidence.limitations.map(value=>`  - ${value}`),'','Errors:',...(evidence.errors.length?evidence.errors:['  none']),'',`Full per-case measured values and trusted input events: ${OUTPUT.json}`,'Screenshots: '+[OUTPUT.leaf,OUTPUT.subtree,OUTPUT.root,OUTPUT.clipped].join(', '),'');
  return lines.join('\n');
}
async function main() {
  await checkScope();
  try {
    sourceBefore=await fs.readFile(INSTALLED,'utf8');
    for(const file of [INSTALLED,INSTALLED_TREE,REFERENCE_RUNNER,SCRIPT])immutableBefore[file]=hash(await fs.readFile(file));
    evidence.immutableBefore=immutableBefore;evidence.runnerSha256=immutableBefore[SCRIPT];evidence.browser.executableSha256=hash(await fs.readFile(CHROME));
    const treeSource=await fs.readFile(INSTALLED_TREE,'utf8');
    invariant(treeSource.includes('children] }')&&treeSource.includes('data-subcalls'),'Installed ToolCallTree no longer wraps descendants; review fixture');
    evidence.installedTree={path:INSTALLED_TREE,sha256:immutableBefore[INSTALLED_TREE],wrapperLines:[31,36,40],fixtureRelationship:'callRow div contains own toolview plus children; data-subcalls recursively contains ToolCallBranch wrappers. Layout/theme are fixture-only.'};
    const extracted=extractInstalled(sourceBefore),fixture=makeFixture(extracted);
    await writeOwned(FIXTURE,fixture);evidence.fixtureSha256=hash(fixture);evidence.node=process.version;
    await startBrowser();await acceptance();
  } catch(error) {evidence.errors.push(error.stack || String(error));console.error(error.stack || error);}
  finally {try{await closeOwned();}catch(error){evidence.errors.push(`Cleanup: ${error.stack || error}`);}}
  evidence.finishedAt=new Date().toISOString();
  evidence.summary={passedCases:evidence.cases.filter(entry=>entry.status==='passed').length,failedCases:evidence.cases.filter(entry=>entry.status==='failed').length,notVerifiedCases:evidence.cases.filter(entry=>entry.status==='not_verified').length,passedAssertions:evidence.cases.flatMap(entry=>entry.assertions).filter(item=>item.pass).length,failedAssertions:evidence.cases.flatMap(entry=>entry.assertions).filter(item=>!item.pass).length};
  evidence.fixtureFileSha256=await fs.readFile(FIXTURE).then(hash).catch(error=>{if(error.code==='ENOENT')return null;throw error;});
  evidence.status=evidence.errors.length===0 && evidence.summary.failedCases===0 && evidence.summary.passedCases===11 && evidence.cleanup?.pass ? 'passed':'failed';
  evidence.executionCounts={browser:{cases:evidence.cases.filter(e=>!e.id.startsWith('node-vm-')).length,assertions:evidence.cases.filter(e=>!e.id.startsWith('node-vm-')).flatMap(e=>e.assertions).length},nodeVm:{cases:evidence.cases.filter(e=>e.id.startsWith('node-vm-')).length,assertions:evidence.cases.filter(e=>e.id.startsWith('node-vm-')).flatMap(e=>e.assertions).length}};
  await writeOwned(OUTPUT.text,makeText());
  if(!evidence.writes.includes(OUTPUT.json))evidence.writes.push(OUTPUT.json);
  await writeOwned(OUTPUT.json,JSON.stringify(evidence,null,2)+'\n');
  console.log(JSON.stringify({status:evidence.status,...evidence.summary,chrome:evidence.browser.version?.product,cleanup:evidence.cleanup?.pass,evidence:OUTPUT.json}));
  process.exitCode=evidence.status==='passed'?0:1;
}
await main();
