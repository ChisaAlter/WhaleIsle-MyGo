'use strict';
// Request table for the remote shell routes: native statuses and auth semantics
// (reference/electron/src/main/remote.js:708-730, 779-807).
const test = require('node:test');
const assert = require('node:assert');
const routes = require('./shell_routes.cjs');

const TOKEN = 'fictional-shell-token';
const ROOT = require('node:path').resolve(require('node:os').tmpdir());
const calls = { status: 0, diff: 0, branches: 0, fetch: 0 };

const git = {
  async gitStatus(cwd) { calls.status++; return { cwd, branch: 'probe-branch', clean: true, files: [] }; },
  async gitFetchForStatus(cwd) { calls.fetch++; return { cwd, fetched: true }; },
  async gitDiff(cwd, options) { calls.diff++; return { cwd, options, patch: '' }; },
  async gitBranchList(cwd) { calls.branches++; return { cwd, branches: ['probe-branch'] }; },
};

function fakeReq({ url, method = 'POST', headers = {}, body = '', chunked = false, total = 0 }) {
  const { Readable } = require('node:stream');
  const req = new Readable({ read() {} });
  req.url = url;
  req.method = method;
  req.headers = headers;
  req.on = req.on.bind(req);
  process.nextTick(() => {
    if (chunked) {
      req.push(Buffer.alloc(total, 0x61)); // 'a' * total
    } else if (body) {
      req.push(body);
    }
    req.push(null);
  });
  return req;
}

function fakeRes() {
  const res = {
    statusCode: 0,
    headers: {},
    body: '',
    headersSent: false,
    writeHead(status, headers) { this.statusCode = status; this.headers = headers || {}; this.headersSent = true; },
    end(payload) { this.body = payload || ''; },
  };
  return res;
}

async function run(handler, options) {
  const res = fakeRes();
  await handler(fakeReq(options), res);
  return res;
}

test('anonymous and wrong-token requests are 401 plain text', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  for (const headers of [{}, { authorization: 'Bearer nope' }, { cookie: 'dsh_remote=nope' }]) {
    const res = await run(handler, { url: '/__remote__/shell/gitStatus', headers, body: '{}' });
    assert.equal(res.statusCode, 401, JSON.stringify(headers));
    assert.equal(res.body, 'unauthorized');
    assert.match(res.headers['content-type'], /text\/plain/);
  }
  assert.equal(calls.status, 0, 'unauthenticated request must not reach git');
});

test('read-only routes answer 200 through the gate with Bearer or cookie', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  for (const headers of [{ authorization: 'Bearer ' + TOKEN }, { cookie: 'x=1; dsh_remote=' + TOKEN }]) {
    const status = await run(handler, { url: '/__remote__/shell/gitStatus', headers, body: JSON.stringify({ cwd: ROOT }) });
    assert.equal(status.statusCode, 200);
    const parsed = JSON.parse(status.body);
    assert.equal(parsed.ok, true);
    assert.equal(parsed.result.branch, 'probe-branch');

    const diff = await run(handler, { url: '/__remote__/shell/gitDiff', headers, body: JSON.stringify({ cwd: ROOT, options: { staged: true } }) });
    assert.equal(diff.statusCode, 200);
    assert.deepEqual(JSON.parse(diff.body).result.options, { staged: true });

    const branches = await run(handler, { url: '/__remote__/shell/gitBranchList', headers, body: JSON.stringify({ cwd: ROOT }) });
    assert.equal(branches.statusCode, 200);
    assert.deepEqual(JSON.parse(branches.body).result.branches, ['probe-branch']);

    const fetch = await run(handler, { url: '/__remote__/shell/gitFetchForStatus', headers, body: JSON.stringify({ cwd: ROOT }) });
    assert.equal(fetch.statusCode, 200);
  }
  assert.ok(calls.status > 0 && calls.diff > 0 && calls.branches > 0 && calls.fetch > 0);
});

test('listDir stays inside the configured roots', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const ok = await run(handler, { url: '/__remote__/shell/listDir', headers, body: JSON.stringify({ cwd: ROOT }) });
  assert.equal(ok.statusCode, 200);
  assert.equal(JSON.parse(ok.body).result.ok, true);

  const root = require('node:path').parse(ROOT).root;
  const outside = await run(handler, { url: '/__remote__/shell/listDir', headers, body: JSON.stringify({ cwd: root }) });
  assert.equal(outside.statusCode, 200);
  assert.equal(JSON.parse(outside.body).ok, false, 'a cwd outside the roots is refused');

  const traversal = await run(handler, { url: '/__remote__/shell/listDir', headers, body: JSON.stringify({ cwd: ROOT, relativePath: '../../..' }) });
  assert.equal(JSON.parse(traversal.body).ok, false);
});

test('getConfig returns only non-secret fields', async () => {
  const handler = routes.createShellHandler({
    tokens: () => [TOKEN], git, roots: () => [ROOT],
    config: () => ({ remoteEnabled: true, remoteMode: 'lan', remotePort: 3499, remoteBindAddress: '127.0.0.1', remoteToken: 'secret', remoteRelayToken: 'secret2', remoteGitRoots: [ROOT] }),
  });
  const res = await run(handler, { url: '/__remote__/shell/getConfig', headers: { authorization: 'Bearer ' + TOKEN }, body: '{}' });
  assert.equal(res.statusCode, 200);
  const result = JSON.parse(res.body).result;
  assert.equal(result.remoteEnabled, true);
  assert.equal(result.remoteGitEnabled, true);
  assert.equal(result.remoteToken, undefined);
  assert.ok(!res.body.includes('secret'));
});

test('unknown name 404, bad JSON 400, oversized body 413, mutations 501', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };

  const unknown = await run(handler, { url: '/__remote__/shell/writeFile', headers, body: '{}' });
  assert.equal(unknown.statusCode, 404);
  assert.deepEqual(JSON.parse(unknown.body), { ok: false, error: 'not found' });

  const bad = await run(handler, { url: '/__remote__/shell/gitStatus', headers, body: '{not json' });
  assert.equal(bad.statusCode, 400);
  assert.deepEqual(JSON.parse(bad.body), { ok: false, error: 'invalid json' });

  const big = await run(handler, { url: '/__remote__/shell/gitStatus', headers, chunked: true, total: routes.MAX_BODY + 1 });
  assert.equal(big.statusCode, 413);
  assert.deepEqual(JSON.parse(big.body), { ok: false, error: '请求过大' });

  for (const name of ['gitCommit', 'gitPush', 'gitPull', 'gitSwitchBranch', 'gitCreateBranch', 'gitCreateChangeRequest', 'saveConfig', 'openSettings']) {
    const res = await run(handler, { url: '/__remote__/shell/' + name, headers, body: '{}' });
    assert.equal(res.statusCode, 501, name);
    assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'shell unavailable' });
  }
});

test('non-POST and other paths are not handled', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  assert.equal(await handler(fakeReq({ url: '/__remote__/shell/gitStatus', method: 'GET', headers: { authorization: 'Bearer ' + TOKEN } }), fakeRes()), false);
  assert.equal(await handler(fakeReq({ url: '/api/session.list', headers: { authorization: 'Bearer ' + TOKEN } }), fakeRes()), false);
});

test('mountShellRoutes keeps the existing listener working', async () => {
  const { EventEmitter } = require('node:events');
  const server = new EventEmitter();
  server.listeners = EventEmitter.prototype.listeners;
  let spaCalls = 0;
  server.on('request', (req, res) => { spaCalls++; res.writeHead(200, {}); res.end('spa'); });
  const mounted = routes.mountShellRoutes(server, { tokens: () => [TOKEN], git, roots: () => [ROOT] });
  assert.equal(mounted, true);
  assert.equal(routes.mountShellRoutes(server, { tokens: () => [TOKEN] }), false, 'mount is idempotent');

  const res = fakeRes();
  server.emit('request', fakeReq({ url: '/__remote__/shell/gitStatus', headers: { authorization: 'Bearer ' + TOKEN }, body: '{}' }), res);
  await new Promise((resolve) => setTimeout(resolve, 10));
  assert.equal(res.statusCode, 200);
  assert.equal(spaCalls, 0, 'shell request did not fall through');

  const spa = fakeRes();
  server.emit('request', fakeReq({ url: '/', method: 'GET' }), spa);
  await new Promise((resolve) => setTimeout(resolve, 10));
  assert.equal(spaCalls, 1);
  assert.equal(spa.body, 'spa');
});
