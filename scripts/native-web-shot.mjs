// Headless Chrome screenshot through CDP, with an explicit device scale and a
// hard deadline. Usage: node cdp-shot.mjs <url> <out.png> [width] [height] [scale]
import {spawn} from 'node:child_process';
import {writeFileSync, rmSync, mkdirSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';

const [url, out, widthArg, heightArg, scaleArg] = process.argv.slice(2);
if (!url || !out) {
  console.error('usage: node cdp-shot.mjs <url> <out.png> [width] [height] [scale]');
  process.exit(2);
}
const width = Number(widthArg || 1440);
const height = Number(heightArg || 920);
const scale = Number(scaleArg || 1.5);
const chrome = 'C:/Program Files/Google/Chrome/Application/chrome.exe';
const profile = path.join(tmpdir(), 'whaleisle-cdp-' + process.pid);
mkdirSync(profile, {recursive: true});
const port = 9223 + (process.pid % 200);
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
  await sleep(9000);
  const shot = await send('Page.captureScreenshot', {format: 'png'});
  writeFileSync(out, Buffer.from(shot.data, 'base64'));
  console.log(JSON.stringify({ok: true, out, width, height, scale, bytes: Buffer.from(shot.data, 'base64').length}));
  ws.close();
} finally {
  child.kill();
  await sleep(500);
  try { rmSync(profile, {recursive: true, force: true}); } catch {}
}
