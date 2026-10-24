// Render the native launcher page in DARK through CDP.
// The page's dark theme is the host-set attribute `data-ds-dark-theme`, read by
// reference/electron/src/shared/dsh-webui-tokens.css:80-81
// (`html[data-ds-dark-theme], body[data-ds-dark-theme] { color-scheme: dark; … }`);
// the launcher window gets it from the injected chrome, so a static headless
// screenshot needs it applied explicitly, exactly as the host does.
import { writeFileSync } from 'node:fs';

const [debugPort, pageUrl, out] = process.argv.slice(2);
if (!debugPort || !pageUrl || !out) throw Error('usage: node launcher-dark-reference.mjs <port> <file-url> <out.png>');

const targets = await (await fetch(`http://127.0.0.1:${debugPort}/json/list`)).json();
const page = targets.find(t => t.type === 'page' && t.url === pageUrl) || targets.find(t => t.type === 'page');
if (!page) throw Error('no page target');
const ws = new WebSocket(page.webSocketDebuggerUrl);
let next = 1;
const pending = new Map();
await new Promise((resolve, reject) => {
  ws.addEventListener('open', resolve, { once: true });
  ws.addEventListener('error', reject, { once: true });
});
ws.addEventListener('message', e => {
  const r = JSON.parse(e.data);
  if (r.id && pending.has(r.id)) {
    const p = pending.get(r.id);
    pending.delete(r.id);
    r.error ? p.reject(Error(JSON.stringify(r.error))) : p.resolve(r.result);
  }
});
const send = (method, params = {}) => new Promise((resolve, reject) => {
  const id = next++;
  pending.set(id, { resolve, reject });
  ws.send(JSON.stringify({ id, method, params }));
});
const evaluate = async expression => {
  const r = await send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true });
  if (r.exceptionDetails) throw Error(JSON.stringify(r.exceptionDetails));
  return r.result.value;
};

await send('Page.enable');
await send('Emulation.setDeviceMetricsOverride', { width: 1060, height: 660, deviceScaleFactor: 1, mobile: false });
// Let the deferred launcher scripts settle so the static markup is styled.
await new Promise(resolve => setTimeout(resolve, 800));
const state = await evaluate(`(() => {
  document.documentElement.setAttribute('data-ds-dark-theme', '');
  document.body.setAttribute('data-ds-dark-theme', '');
  const bg = getComputedStyle(document.body).backgroundColor;
  const rail = document.querySelector('.rail');
  return {
    html: document.documentElement.getAttribute('data-ds-dark-theme'),
    body: document.body.getAttribute('data-ds-dark-theme'),
    colorScheme: getComputedStyle(document.documentElement).colorScheme,
    bodyBg: bg,
    railBg: rail ? getComputedStyle(rail).backgroundColor : null,
    size: [window.innerWidth, window.innerHeight]
  };
})()`);
console.log(JSON.stringify(state));
await new Promise(resolve => setTimeout(resolve, 400));
const shot = await send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
writeFileSync(out, Buffer.from(shot.data, 'base64'));
console.log(JSON.stringify({ out, bytes: Buffer.from(shot.data, 'base64').length }));
ws.close();
