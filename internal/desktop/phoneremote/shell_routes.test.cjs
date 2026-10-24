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

  for (const name of ['gitCreateChangeRequest', 'saveConfig', 'openSettings', 'openGallery']) {
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

// Gated mutations: the desktop gate stays in charge. A denied authorisation is
// 403 with the native error text; missing/invalid arguments are 400; a fake
// authoriser (the desktop saying yes) yields the native 200 {ok,result}.
function denialGit() {
  const deny = async () => { throw new Error('Mobile Git desktop workspace permission expired'); };
  return { gitStatus: deny, gitCommit: deny, gitSwitchBranch: deny, gitCreateBranch: deny };
}

test('mutations without a desktop authorisation are 403, never run', async () => {
  const calls = { commit: 0, create: 0, switch: 0 };
  const git = {
    async gitCommit() { calls.commit++; throw new Error('Mobile Git desktop workspace permission expired'); },
    async gitCreateBranch() { calls.create++; throw new Error('Mobile Git desktop workspace permission expired'); },
    async gitSwitchBranch() { calls.switch++; throw new Error('Mobile Git desktop workspace permission expired'); },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  for (const [name, body] of [
    ['gitCommit', { cwd: ROOT, message: 'probe' }],
    ['gitCreateBranch', { cwd: ROOT, name: 'probe-branch' }],
    ['gitSwitchBranch', { cwd: ROOT, ref: 'probe-branch' }],
  ]) {
    const res = await run(handler, { url: '/__remote__/shell/' + name, headers, body: JSON.stringify(body) });
    assert.equal(res.statusCode, 403, name);
    assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'Mobile Git desktop workspace permission expired' });
  }
  assert.equal(calls.commit + calls.create + calls.switch, 3, 'the gate was consulted for each call');
});

test('mutations validate arguments before touching the gate', async () => {
  let reached = 0;
  const git = { async gitCommit() { reached++; return {}; }, async gitCreateBranch() { reached++; return {}; }, async gitSwitchBranch() { reached++; return {}; } };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const cases = [
    ['gitCommit', { cwd: ROOT }, 'invalid args: message'],
    ['gitCommit', { cwd: ROOT, message: 'x', filePaths: 'nope' }, 'invalid args: filePaths'],
    ['gitCreateBranch', { cwd: ROOT }, 'invalid args: name'],
    ['gitSwitchBranch', { cwd: ROOT }, 'invalid args: ref'],
  ];
  for (const [name, body, error] of cases) {
    const res = await run(handler, { url: '/__remote__/shell/' + name, headers, body: JSON.stringify(body) });
    assert.equal(res.statusCode, 400, name + ' ' + error);
    assert.deepEqual(JSON.parse(res.body), { ok: false, error });
  }
  // A cwd outside the configured roots is a denial, not an argument error.
  const outside = await run(handler, { url: '/__remote__/shell/gitCommit', headers, body: JSON.stringify({ cwd: require('node:path').parse(ROOT).root, message: 'x' }) });
  assert.equal(outside.statusCode, 403);
  assert.equal(JSON.parse(outside.body).error, 'Mobile Git is not authorized for this workspace');
  assert.equal(reached, 0, 'invalid requests never reached the gate');
});

test('mutations succeed through the gate with a fake authorizer', async () => {
  const seen = [];
  const git = {
    async gitCommit(cwd, message, filePaths, onProgress, options) {
      seen.push({ name: 'gitCommit', cwd, message, filePaths, options });
      return { committed: true, refName: 'probe-branch', message };
    },
    async gitCreateBranch(cwd, name) {
      seen.push({ name: 'gitCreateBranch', cwd, name });
      return { created: true, refName: name };
    },
    async gitSwitchBranch(cwd, ref) {
      seen.push({ name: 'gitSwitchBranch', cwd, ref });
      return { switched: true, refName: ref };
    },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const create = await run(handler, { url: '/__remote__/shell/gitCreateBranch', headers, body: JSON.stringify({ cwd: ROOT, name: 'probe-branch' }) });
  assert.equal(create.statusCode, 200);
  assert.deepEqual(JSON.parse(create.body), { ok: true, result: { created: true, refName: 'probe-branch' } });

  const commit = await run(handler, { url: '/__remote__/shell/gitCommit', headers, body: JSON.stringify({ cwd: ROOT, message: 'probe commit', filePaths: ['probe.txt'], options: { signOff: true } }) });
  assert.equal(commit.statusCode, 200);
  assert.equal(JSON.parse(commit.body).result.committed, true);
  const commitSeen = seen.find((entry) => entry.name === 'gitCommit');
  assert.deepEqual(commitSeen.filePaths, ['probe.txt']);
  assert.deepEqual(commitSeen.options, { signOff: true });

  const switchRes = await run(handler, { url: '/__remote__/shell/gitSwitchBranch', headers, body: JSON.stringify({ cwd: ROOT, ref: 'probe-branch' }) });
  assert.equal(switchRes.statusCode, 200);
  assert.deepEqual(JSON.parse(switchRes.body).result, { switched: true, refName: 'probe-branch' });

  const denied = routes.createShellHandler({ tokens: () => [TOKEN], git: denialGit(), roots: () => [ROOT] });
  const deniedRes = await run(denied, { url: '/__remote__/shell/gitCommit', headers, body: JSON.stringify({ cwd: ROOT, message: 'x' }) });
  assert.equal(deniedRes.statusCode, 403);
});

// Push/pull go through the same gate: native remote-shell.js:70-73 delegates
// both with only the workspace cwd, so the branch comes from HEAD.
test('push/pull are 403 without a desktop authorisation', async () => {
  const calls = { push: 0, pull: 0 };
  const git = {
    async gitPush() { calls.push++; throw new Error('Mobile Git desktop workspace permission expired'); },
    async gitPull() { calls.pull++; throw new Error('Mobile Git desktop workspace permission expired'); },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  for (const name of ['gitPush', 'gitPull']) {
    const res = await run(handler, { url: '/__remote__/shell/' + name, headers, body: JSON.stringify({ cwd: ROOT }) });
    assert.equal(res.statusCode, 403, name);
    assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'Mobile Git desktop workspace permission expired' });
  }
  assert.equal(calls.push, 1, 'push reached the gate');
  assert.equal(calls.pull, 1, 'pull reached the gate');
});

test('push/pull validate cwd and optional remote/branch before the gate', async () => {
  let reached = 0;
  const git = { async gitPush() { reached++; return { ok: true }; }, async gitPull() { reached++; return { ok: true }; } };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const root = require('node:path').parse(ROOT).root;
  const outside = await run(handler, { url: '/__remote__/shell/gitPush', headers, body: JSON.stringify({ cwd: root }) });
  assert.equal(outside.statusCode, 403);
  assert.equal(JSON.parse(outside.body).error, 'Mobile Git is not authorized for this workspace');

  const badRemote = await run(handler, { url: '/__remote__/shell/gitPush', headers, body: JSON.stringify({ cwd: ROOT, remote: 42 }) });
  assert.equal(badRemote.statusCode, 400);
  assert.deepEqual(JSON.parse(badRemote.body), { ok: false, error: 'invalid args: remote' });

  const badBranch = await run(handler, { url: '/__remote__/shell/gitPull', headers, body: JSON.stringify({ cwd: ROOT, branch: '' }) });
  assert.equal(badBranch.statusCode, 400);
  assert.deepEqual(JSON.parse(badBranch.body), { ok: false, error: 'invalid args: branch' });
  assert.equal(reached, 0, 'invalid requests never reached the gate');
});

test('push/pull succeed through the gate with a fake authorizer', async () => {
  const seen = [];
  const git = {
    async gitPush(cwd) { seen.push({ name: 'gitPush', cwd }); return { ok: true, status: 'pushed', branch: 'probe-branch', ahead: 1 }; },
    async gitPull(cwd) { seen.push({ name: 'gitPull', cwd }); return { ok: true, status: 'fast-forward', branch: 'probe-branch' }; },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const pushRes = await run(handler, { url: '/__remote__/shell/gitPush', headers, body: JSON.stringify({ cwd: ROOT }) });
  assert.equal(pushRes.statusCode, 200);
  assert.deepEqual(JSON.parse(pushRes.body), { ok: true, result: { ok: true, status: 'pushed', branch: 'probe-branch', ahead: 1 } });

  const pullRes = await run(handler, { url: '/__remote__/shell/gitPull', headers, body: JSON.stringify({ cwd: ROOT, remote: 'origin', branch: 'probe-branch' }) });
  assert.equal(pullRes.statusCode, 200);
  assert.equal(JSON.parse(pullRes.body).result.status, 'fast-forward');

  assert.deepEqual(seen, [
    { name: 'gitPush', cwd: ROOT },
    { name: 'gitPull', cwd: ROOT },
  ], 'native delegates only the workspace cwd');

  // A git-level failure result keeps the native wrapResult shape.
  const failing = routes.createShellHandler({ tokens: () => [TOKEN], git: { async gitPush() { return { ok: false, message: 'no upstream configured' }; } }, roots: () => [ROOT] });
  const failed = await run(failing, { url: '/__remote__/shell/gitPush', headers, body: JSON.stringify({ cwd: ROOT }) });
  assert.equal(failed.statusCode, 400);
  assert.deepEqual(JSON.parse(failed.body), { ok: false, error: 'no upstream configured' });
});
