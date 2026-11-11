'use strict';
const fs = require('node:fs'), path = require('node:path'), readline = require('node:readline');
const [root, mobile, toolRoot, chrome] = process.argv.slice(2);
const emit = value => process.stdout.write(JSON.stringify(value) + '\n');
const {Miniflare, Log, LogLevel} = require(path.join(toolRoot, 'node_modules/miniflare'));
const esbuild = require(path.join(toolRoot, 'node_modules/wrangler/node_modules/esbuild'));
const {chromium} = require(path.join(toolRoot, 'node_modules/playwright-core'));
let relay, browser, pageServer, pageOrigin, closing;
async function cleanup() {
  if (closing) return closing;
  closing = (async () => { if (browser) await browser.close(); if (pageServer) await new Promise(resolve => pageServer.close(resolve)); if (relay) await relay.dispose(); esbuild.stop(); emit({kind: 'closed'}); })();
  return closing;
}
function safeError(error) { return String(error.stack || error).replace(/#offer=[^\s"']+/g, '#offer=[redacted]'); }
async function browse({offer, sessionId}) {
  const parsed = new URL(offer);
  const pageURL = pageOrigin + '/' + parsed.hash;
  browser = await chromium.launch({executablePath: chrome, headless: true});
  const context = await browser.newContext({viewport: {width: 390, height: 780}});
  let relaySocket;
  await context.routeWebSocket(url => new URL(url).hostname === '127.0.0.1', socket => { relaySocket = socket; socket.connectToServer(); });
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const report = {errors, checks: []};
  try {
    await page.goto(pageURL, {waitUntil: 'load'});
    await page.locator('#screen-chat').waitFor({state: 'visible', timeout: 45000});
    await page.locator('#menu').click();
    await page.locator(`[data-session-id="${sessionId}"] button.session`).first().click({timeout: 15000});
    await page.locator('#log').getByText('QA_REPLY_OK: deterministic local model response.', {exact: true}).waitFor({timeout: 15000});
    report.checks.push('paired and opened existing Host session history');
    await page.locator('#draft').fill('QA_NEXT: continue through the owned phone relay.');
    await page.locator('#send-btn').click();
    await page.locator('#log').getByText('QA_NEXT_OK: next turn after stop completed.', {exact: true}).waitFor({timeout: 30000});
    report.checks.push('phone send traversed real relay/daemon/Host/model');
    await page.locator('#draft').fill('Unsent recovery draft');
    await context.setOffline(true);
    if (!relaySocket) throw Error('No real relay WebSocket was observed');
    await relaySocket.close({code: 1012, reason: 'Owned connection recovery check'});
    await page.waitForFunction(() => window.__phoneQA?.().phase !== 'online', null, {timeout: 15000});
    report.checks.push('relay WebSocket closed and client observed disconnection');
    await context.setOffline(false);
    await page.evaluate(() => window.dispatchEvent(new Event('dshd-resume')));
    await page.waitForFunction(() => window.__phoneQA?.().phase === 'online' && !document.querySelector('#draft').disabled, null, {timeout: 15000});
    if (await page.locator('#draft').inputValue() !== 'Unsent recovery draft') throw Error('draft lost on connection resume');
    await page.reload({waitUntil: 'load'});
    await page.locator('#screen-chat').waitFor({state: 'visible', timeout: 30000});
    if (await page.locator('#log').getByText('QA_NEXT_OK: next turn after stop completed.', {exact: true}).count() === 0) {
      await page.locator('#menu').click();
      await page.locator(`[data-session-id="${sessionId}"] button.session`).first().click();
    }
    await page.locator('#log').getByText('QA_NEXT_OK: next turn after stop completed.', {exact: true}).waitFor();
    if (await page.locator('#draft').inputValue() !== 'Unsent recovery draft') throw Error('draft lost after browser reload');
    report.checks.push('saved device reconnect, history and unsent draft restored');
    await page.locator('#sessions-page').waitFor({state: 'hidden'});
    await page.waitForFunction(() => document.querySelector('.drawer').getBoundingClientRect().right <= 1);
    await page.screenshot({path: path.join(root, 'phone-connected.png')});
    await browser.close(); browser = null;
    fs.writeFileSync(path.join(root, 'phone-browser-result.json'), JSON.stringify(report, null, 2));
    emit({kind: 'browser-complete'});
  } catch (error) {
    report.failure = safeError(error);
    report.catalog = await page.evaluate(() => window.__phoneQA?.()).catch(() => null);
    await page.screenshot({path: path.join(root, 'phone-failed.png')}).catch(() => {});
    fs.writeFileSync(path.join(root, 'phone-browser-result.json'), JSON.stringify(report, null, 2));
    throw error;
  }
}
async function start() {
  const worker = path.join(root, 'phone-relay-worker.mjs');
  await esbuild.build({entryPoints: [path.join(mobile, 'node_modules/@chisacode/relay/dist/cloudflare-adapter.js')], outfile: worker, bundle: true, format: 'esm', platform: 'browser', target: 'es2022', logLevel: 'silent'});
  relay = new Miniflare({rootPath: root, modulesRoot: root, scriptPath: worker, modules: true, compatibilityDate: '2024-12-01', host: '127.0.0.1', port: 0, durableObjects: {RELAY: {className: 'RelayDurableObject', useSQLite: true}}, durableObjectsPersist: path.join(root, 'phone-relay-db'), log: new Log(LogLevel.NONE)});
  const endpoint = (await relay.ready).host;
  // Away mode serves the SPA separately from the daemon, as production does.
  const {createMobileWebServer} = require(path.join(mobile, 'src/main/mobile-web-server.js'));
  pageServer = createMobileWebServer({root: path.join(mobile, 'mobile/web')});
  const handlers = pageServer.listeners('request');
  pageServer.removeAllListeners('request');
  pageServer.on('request', (req, res) => {
    if (new URL(req.url, 'http://localhost').pathname === '/app.js') {
      res.writeHead(200, {'content-type': 'text/javascript'});
      res.end(fs.readFileSync(path.join(mobile, 'mobile/web/app.js'), 'utf8') + '\nwindow.__phoneQA=()=>({sessions:state.sessions,catalog:state.catalogSessions,workspaces:state.workspaces,error:state.sessionsError,phase:state.connPhase});');
    } else for (const handler of handlers) handler(req, res);
  });
  await new Promise(resolve => pageServer.listen(0, '127.0.0.1', resolve));
  pageOrigin = `http://127.0.0.1:${pageServer.address().port}`;
  emit({kind: 'ready', endpoint, pageOrigin});
}
const input = readline.createInterface({input: process.stdin});
let chain = start();
input.on('line', line => { chain = chain.then(async () => { const action = JSON.parse(line); if (action.action === 'browse') await browse(action); else if (action.action === 'stop') await cleanup(); }).catch(async error => { emit({kind: 'error', message: safeError(error)}); await cleanup(); }); });
input.on('close', () => { void chain.finally(cleanup).finally(() => process.exit(0)); });
chain.catch(async error => { emit({kind: 'error', message: safeError(error)}); await cleanup(); });
