// Native harness reference screenshots through Chrome DevTools Protocol.
//
// Plain `chrome --screenshot` hangs on the DeepSeek Harness web app, so this
// drives headless Chrome over CDP instead: it sets exact emulated device
// metrics, navigates, optionally clicks text (e.g. the first-run 预览版说明
// 继续 button), optionally dumps a layout map, and writes the PNG.
//
// Usage:
//   node scripts/native-web-shot.mjs <url> <out.png> [width] [height] [scale]
//        [--click-text <text>]...  [--settle <ms>] [--map <out.json>]
//        [--eval <js-file>] [--eval-out <out.json>]
//
// `url` is the token URL printed by `dsh web` (plain / answers 401).
import {spawn} from 'node:child_process';
import {writeFileSync, rmSync, mkdirSync, readFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';

const argv = process.argv.slice(2);
const positional = [];
const clicks = [];
let mapOut = null;
let evalFile = null;
let evalOut = null;
let settle = 9000;
for (let i = 0; i < argv.length; i++) {
  const arg = argv[i];
  if (arg === '--click-text') clicks.push(argv[++i]);
  else if (arg === '--settle') settle = Number(argv[++i]);
  else if (arg === '--map') mapOut = argv[++i];
  else if (arg === '--eval') evalFile = argv[++i];
  else if (arg === '--eval-out') evalOut = argv[++i];
  else positional.push(arg);
}
const [url, out, widthArg, heightArg, scaleArg] = positional;
if (!url || !out) {
  console.error('usage: node scripts/native-web-shot.mjs <url> <out.png> [width] [height] [scale] [--click-text <text>] [--map <out.json>]');
  process.exit(2);
}
const width = Number(widthArg || 1440);
const height = Number(heightArg || 920);
const scale = Number(scaleArg || 1.5);
const chrome = process.env.WHALEISLE_CHROME || 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const profile = path.join(tmpdir(), 'whaleisle-cdp-' + process.pid);
mkdirSync(profile, {recursive: true});
const port = 9223 + (process.pid % 500);
const child = spawn(chrome, [
  '--headless=new', '--disable-gpu', '--hide-scrollbars', '--no-first-run',
  `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`, 'about:blank',
], {stdio: 'ignore'});
const sleep = ms => new Promise(r => setTimeout(r, ms));
const deadline = Date.now() + 45000;
try {
  let target;
  while (Date.now() < deadline) {
    try {
      const list = await fetch(`http://127.0.0.1:${port}/json/list`).then(r => r.json());
      target = list.find(t => t.type === 'page');
      if (target) break;
    } catch {}
    await sleep(200);
  }
  if (!target) throw new Error('devtools did not start');
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  const pending = new Map();
  let id = 0;
  const send = (method, params = {}) => new Promise((resolve, reject) => {
    const messageId = ++id;
    pending.set(messageId, {resolve, reject});
    ws.send(JSON.stringify({id: messageId, method, params}));
  });
  ws.addEventListener('message', event => {
    const msg = JSON.parse(event.data);
    if (msg.id && pending.has(msg.id)) {
      const {resolve, reject} = pending.get(msg.id);
      pending.delete(msg.id);
      msg.error ? reject(new Error(JSON.stringify(msg.error))) : resolve(msg.result);
    }
  });
  await new Promise((resolve, reject) => { ws.addEventListener('open', resolve); ws.addEventListener('error', reject); });
  await send('Page.enable');
  await send('Emulation.setDeviceMetricsOverride', {width, height, deviceScaleFactor: scale, mobile: false});
  await send('Page.navigate', {url});
  await sleep(settle);
  for (const text of clicks) {
    const expr = `(() => {
      const target = ${JSON.stringify(text)};
      const hits = [...document.querySelectorAll('body *')].filter(el => el.children.length === 0 && (el.textContent||'').trim() === target);
      if (!hits.length) return {clicked: false};
      const el = hits[hits.length - 1];
      const button = el.closest('button, [role=button]') || el;
      button.click();
      return {clicked: true, tag: button.tagName.toLowerCase()};
    })()`;
    const res = await send('Runtime.evaluate', {expression: expr, returnByValue: true});
    console.log(JSON.stringify({clickText: text, result: res.result.value}));
    await sleep(1200);
  }
  if (mapOut || evalFile) {
    const expression = evalFile ? readFileSync(evalFile, 'utf8') : '({body: document.body.innerText.slice(0, 200)})';
    const res = await send('Runtime.evaluate', {expression, returnByValue: true, awaitPromise: true});
    if (res.exceptionDetails) throw new Error('eval failed: ' + JSON.stringify(res.exceptionDetails.exception?.description || res.exceptionDetails.text));
    writeFileSync(mapOut || evalOut, JSON.stringify(res.result.value, null, 2));
  }
  const shot = await send('Page.captureScreenshot', {format: 'png'});
  const bytes = Buffer.from(shot.data, 'base64');
  writeFileSync(out, bytes);
  console.log(JSON.stringify({ok: true, out, width, height, scale, bytes: bytes.length}));
  ws.close();
} finally {
  child.kill();
  await sleep(500);
  try { rmSync(profile, {recursive: true, force: true}); } catch {}
}
