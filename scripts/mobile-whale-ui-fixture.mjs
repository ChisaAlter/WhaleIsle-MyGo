// Owned headless browser exercise of the staged mobile renderer and HTTP server.
import fs from 'node:fs';
import path from 'node:path';
import {createRequire} from 'node:module';
import assert from 'node:assert/strict';

const [mobileRoot, playwrightRoot, chrome, output] = process.argv.slice(2);
if (!output) throw Error('Usage: mobile-whale-ui-fixture.mjs <mobile runtime> <playwright-core> <chrome> <output directory>');
const require = createRequire(import.meta.url);
const {chromium} = require(path.resolve(playwrightRoot));
const {createMobileWebServer} = require(path.join(path.resolve(mobileRoot), 'src/main/mobile-web-server.js'));
const root = path.join(path.resolve(mobileRoot), 'mobile/web');
fs.mkdirSync(output, {recursive: true});
const report = {scope: 'staged mobile server, full app renderer, synthetic card, isolated headless Chrome; no pairing or external model', checks: [], errors: []};
const server = createMobileWebServer({root});
const listener = server.listeners('request')[0];
server.removeAllListeners('request');
server.on('request', (req, res) => {
  if (new URL(req.url, 'http://localhost').pathname === '/app.js') {
    // Expose existing functions only in this owned server response, not in product files.
    res.writeHead(200, {'content-type': 'text/javascript'});
    res.end(fs.readFileSync(path.join(root, 'app.js'), 'utf8') + '\nwindow.__mobileQA={whaleUiCardNode,whaleUiInstances,state};');
  } else listener(req, res);
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
let browser;
try {
  assert.equal((await fetch(origin)).headers.get('x-frame-options'), 'DENY');
  assert.equal((await fetch(origin + '/whale-ui/viewer.html')).headers.get('x-frame-options'), 'SAMEORIGIN');
  report.checks.push('viewer-only same-origin framing');
  browser = await chromium.launch({executablePath: chrome, headless: true});
  const page = await browser.newPage({viewport: {width: 390, height: 700}, deviceScaleFactor: 1});
  await page.route('**/*', route => {
    const url = route.request().url();
    return url.startsWith(origin + '/') || url.startsWith('data:') || url.startsWith('blob:') ? route.continue() : route.abort();
  });
  page.on('pageerror', error => report.errors.push(error.message));
  await page.goto(origin, {waitUntil: 'load'});
  await page.waitForFunction(() => Boolean(window.__mobileQA));
  const program = 'root = Col([Title("手机交互卡片","md"), Input($quantity,"数量","number"), Text("Total " + $quantity*2)])\n$quantity = 2';
  await page.evaluate(program => {
    const panel = document.createElement('section');
    panel.id = 'owned-card-fixture';
    panel.style.cssText = 'position:fixed;inset:16px;z-index:9999;background:var(--bg,#fff);padding:8px';
    document.body.append(panel);
    panel.append(window.__mobileQA.whaleUiCardNode({cardKey: 'quantity', program}));
    window.__firstWhaleFrame = panel.querySelector('iframe');
  }, program);
  const frame = page.frameLocator('#owned-card-fixture iframe');
  await frame.getByText('Total 4', {exact: true}).waitFor({timeout: 20000});
  assert.equal(await page.locator('#owned-card-fixture iframe').getAttribute('sandbox'), 'allow-scripts');
  const viewer = page.frames().find(frame => frame.url().includes('/whale-ui/viewer.html'));
  assert.equal(await viewer.evaluate(() => window.origin), 'null');
  report.checks.push('sandboxed production viewer renders full bundle');
  await frame.locator('input').fill('7');
  await frame.getByText('Total 14', {exact: true}).waitFor();
  await page.waitForFunction(() => window.__mobileQA.whaleUiInstances.get('local:quantity')?.state.values?.['$quantity'] === 7);
  report.checks.push('interactive input state reaches parent');
  await page.getByRole('tab', {name: '源码', exact: true}).click();
  assert.equal(await page.locator('#owned-card-fixture .whale-ui-source code').textContent(), program);
  await page.getByRole('tab', {name: '预览', exact: true}).click();
  await frame.getByText('Total 14', {exact: true}).waitFor();
  report.checks.push('source/preview preserves viewer and state');
  await page.evaluate(program => {
    window.__mobileQA.whaleUiCardNode({cardKey: 'quantity', program: program.replace('手机交互卡片', '流式更新完成'), streaming: true});
  }, program);
  await frame.getByText('流式更新完成', {exact: true}).waitFor();
  await frame.getByText('Total 14', {exact: true}).waitFor();
  assert.equal(await page.evaluate(() => window.__firstWhaleFrame === document.querySelector('#owned-card-fixture iframe')), true);
  report.checks.push('streaming update reuses iframe and edited state');
  await page.screenshot({path: path.join(output, 'mobile-whale-light.png')});
  await page.emulateMedia({colorScheme: 'dark'});
  await page.setViewportSize({width: 320, height: 700});
  await page.screenshot({path: path.join(output, 'mobile-whale-narrow-dark.png')});
  const overflow = await page.evaluate(() => {
    const panel = document.querySelector('#owned-card-fixture');
    return panel.scrollWidth > panel.clientWidth;
  });
  assert.equal(overflow, false);
  assert.deepEqual(report.errors, []);
  report.checks.push('narrow viewport without horizontal overflow');
} catch (error) {
  report.failure = error.stack;
  throw error;
} finally {
  if (browser) await browser.close();
  await new Promise(resolve => server.close(resolve));
  report.ownedBrowserClosed = true;
  report.ownedServerClosed = true;
  fs.writeFileSync(path.join(output, 'mobile-whale-result.json'), JSON.stringify(report, null, 2) + '\n');
  console.log(JSON.stringify(report, null, 2));
}
