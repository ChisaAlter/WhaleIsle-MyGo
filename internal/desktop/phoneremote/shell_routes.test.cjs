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

// gitCreateChangeRequest (native remote-shell.js:80-81, git.js:949-1120): the
// gate owns authorisation and desktop review; unsupported providers and gh
// failures come back as native failures, never as fake success.
test('change request is 403 without a desktop authorisation', async () => {
  const calls = { create: 0 };
  const git = {
    async gitPush() { throw new Error('Mobile Git desktop workspace permission expired'); },
    async gitCreateChangeRequest() { calls.create++; throw new Error('Mobile Git desktop workspace permission expired'); },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const res = await run(handler, { url: '/__remote__/shell/gitCreateChangeRequest', headers: { authorization: 'Bearer ' + TOKEN }, body: JSON.stringify({ cwd: ROOT, input: { title: 'x' } }) });
  assert.equal(res.statusCode, 403);
  assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'Mobile Git desktop workspace permission expired' });
  assert.equal(calls.create, 1, 'the gate was consulted for the create call');
});

test('change request validates cwd and input before the gate', async () => {
  let reached = 0;
  const git = { async gitPush() { reached++; return { ok: true }; }, async gitCreateChangeRequest() { reached++; return { ok: true }; } };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const root = require('node:path').parse(ROOT).root;
  const outside = await run(handler, { url: '/__remote__/shell/gitCreateChangeRequest', headers, body: JSON.stringify({ cwd: root, input: {} }) });
  assert.equal(outside.statusCode, 403);
  assert.equal(JSON.parse(outside.body).error, 'Mobile Git is not authorized for this workspace');

  for (const [input, error] of [
    ['nope', 'invalid args: input'],
    [[], 'invalid args: input'],
    [{ title: '' }, 'invalid args: input.title'],
    [{ body: 42 }, 'invalid args: input.body'],
    [{ base: '' }, 'invalid args: input.base'],
  ]) {
    const res = await run(handler, { url: '/__remote__/shell/gitCreateChangeRequest', headers, body: JSON.stringify({ cwd: ROOT, input }) });
    assert.equal(res.statusCode, 400, JSON.stringify(input));
    assert.deepEqual(JSON.parse(res.body), { ok: false, error });
  }
  assert.equal(reached, 0, 'invalid requests never reached the gate');
});

test('change request pushes first, then returns the created URL', async () => {
  const order = [];
  const git = {
    async gitPush(cwd) { order.push(['push', cwd]); return { ok: true, status: 'pushed' }; },
    async gitCreateChangeRequest(cwd, input) {
      order.push(['create', cwd, input]);
      return { ok: true, status: 'created', url: 'https://github.com/o/r/pull/7', number: 7 };
    },
  };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const res = await run(handler, { url: '/__remote__/shell/gitCreateChangeRequest', headers: { authorization: 'Bearer ' + TOKEN }, body: JSON.stringify({ cwd: ROOT, input: { title: 'Probe PR', body: 'body', base: 'main', preserveProvided: true } }) });
  assert.equal(res.statusCode, 200);
  assert.deepEqual(JSON.parse(res.body), { ok: true, result: { ok: true, status: 'created', url: 'https://github.com/o/r/pull/7', number: 7 } });
  assert.deepEqual(order[0], ['push', ROOT], 'the branch is pushed before the create call');
  assert.equal(order[1][0], 'create');
  assert.deepEqual(order[1][2], { title: 'Probe PR', body: 'body', base: 'main', preserveProvided: true });
});

test('change request failures keep the native reason (no fake success)', async () => {
  const unsupported = { async gitPush() { return { ok: true }; }, async gitCreateChangeRequest() { return { ok: false, message: 'Creating a pull request for local is not supported yet (requires GitHub via gh).' }; } };
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git: unsupported, roots: () => [ROOT] });
  const res = await run(handler, { url: '/__remote__/shell/gitCreateChangeRequest', headers: { authorization: 'Bearer ' + TOKEN }, body: JSON.stringify({ cwd: ROOT, input: { title: 'x' } }) });
  assert.equal(res.statusCode, 400);
  assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'Creating a pull request for local is not supported yet (requires GitHub via gh).' });

  const review = { async gitPush() { return { ok: true }; }, async gitCreateChangeRequest() { throw new Error('Review and confirm a change-request title/body before creating it'); } };
  const reviewHandler = routes.createShellHandler({ tokens: () => [TOKEN], git: review, roots: () => [ROOT] });
  const reviewRes = await run(reviewHandler, { url: '/__remote__/shell/gitCreateChangeRequest', headers: { authorization: 'Bearer ' + TOKEN }, body: JSON.stringify({ cwd: ROOT, input: {} }) });
  assert.equal(reviewRes.statusCode, 400);
  assert.deepEqual(JSON.parse(reviewRes.body), { ok: false, error: 'Review and confirm a change-request title/body before creating it' });
});

// Host actions (native remote-shell.js:84-96): openSettings returns the desktop
// opened flag, openGallery opens the appearance section, saveConfig persists a
// whitelisted patch and never echoes tokens.
test('host actions require the token', async () => {
  const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  for (const name of ['openSettings', 'openGallery', 'saveConfig']) {
    const res = await run(handler, { url: '/__remote__/shell/' + name, body: '{}' });
    assert.equal(res.statusCode, 401, name);
  }
});

test('openSettings and openGallery return the desktop shape', async () => {
  const opened = [];
  const handler = routes.createShellHandler({
    tokens: () => [TOKEN], git, roots: () => [ROOT],
    host: { openSettings: (section) => { opened.push(section); return section !== 'blocked'; } },
  });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const remote = await run(handler, { url: '/__remote__/shell/openSettings', headers, body: JSON.stringify({ sectionId: 'remote' }) });
  assert.equal(remote.statusCode, 200);
  assert.deepEqual(JSON.parse(remote.body), { ok: true, result: { opened: true, section: 'remote' } });

  const gallery = await run(handler, { url: '/__remote__/shell/openGallery', headers, body: '{}' });
  assert.equal(gallery.statusCode, 200);
  assert.deepEqual(JSON.parse(gallery.body), { ok: true, result: { opened: true, section: 'appearance' } });
  assert.deepEqual(opened, ['remote', 'appearance']);

  const badSection = await run(handler, { url: '/__remote__/shell/openSettings', headers, body: JSON.stringify({ sectionId: 42 }) });
  assert.equal(badSection.statusCode, 400);
  assert.deepEqual(JSON.parse(badSection.body), { ok: false, error: 'invalid args: sectionId' });

  // Without a desktop host the opened flag is false, not a fake success.
  const noHost = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const plain = await run(noHost, { url: '/__remote__/shell/openSettings', headers, body: '{}' });
  assert.deepEqual(JSON.parse(plain.body), { ok: true, result: { opened: false, section: '' } });
});

test('saveConfig persists a whitelisted patch without echoing tokens', async () => {
  const saved = [];
  const handler = routes.createShellHandler({
    tokens: () => [TOKEN], git, roots: () => [ROOT],
    host: {
      saveConfig: async (patch) => {
        saved.push(patch);
        return { ok: true, config: { remoteEnabled: true, remoteMode: patch.remoteMode || 'lan', remotePort: 3499, remoteBindAddress: '127.0.0.1', remoteAppBaseUrl: '' } };
      },
    },
  });
  const headers = { authorization: 'Bearer ' + TOKEN };
  const res = await run(handler, { url: '/__remote__/shell/saveConfig', headers, body: JSON.stringify({ patch: { remoteMode: 'relay', remotePort: 3499 } }) });
  assert.equal(res.statusCode, 200);
  const body = JSON.parse(res.body);
  assert.equal(body.ok, true);
  assert.equal(body.result.remoteMode, 'relay');
  assert.ok(!JSON.stringify(body).includes('Token'));
  assert.deepEqual(saved, [{ remoteMode: 'relay', remotePort: 3499 }]);

  for (const [patch, error] of [
    ['nope', 'invalid args: patch'],
    [[], 'invalid args: patch'],
    [{ remoteToken: 'secret' }, 'invalid args: patch.remoteToken'],
  ]) {
    const bad = await run(handler, { url: '/__remote__/shell/saveConfig', headers, body: JSON.stringify({ patch }) });
    assert.equal(bad.statusCode, 400, JSON.stringify(patch));
    assert.deepEqual(JSON.parse(bad.body), { ok: false, error });
  }
  assert.equal(saved.length, 1, 'invalid patches never reached the host');

  const noHost = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT] });
  const unavailable = await run(noHost, { url: '/__remote__/shell/saveConfig', headers, body: JSON.stringify({ patch: {} }) });
  assert.equal(unavailable.statusCode, 400);
  assert.deepEqual(JSON.parse(unavailable.body), { ok: false, error: 'saveConfig requires the desktop host' });

  const failing = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT], host: { saveConfig: async () => ({ ok: false, error: 'config store read-only' }) } });
  const failed = await run(failing, { url: '/__remote__/shell/saveConfig', headers, body: JSON.stringify({ patch: {} }) });
  assert.equal(failed.statusCode, 400);
  assert.deepEqual(JSON.parse(failed.body), { ok: false, error: 'config store read-only' });
});

test('saveConfig preserves desktop validation errors without a success result', async () => {
  const headers = { authorization: 'Bearer ' + TOKEN };
  for (const saveConfig of [
    async () => ({ ok: false, error: 'theme must be a valid theme id' }),
    async () => { throw new Error('theme must be a valid theme id'); },
  ]) {
    const handler = routes.createShellHandler({ tokens: () => [TOKEN], git, roots: () => [ROOT], host: { saveConfig } });
    const res = await run(handler, { url: '/__remote__/shell/saveConfig', headers, body: JSON.stringify({ patch: { theme: 42 } }) });
    assert.equal(res.statusCode, 400);
    assert.deepEqual(JSON.parse(res.body), { ok: false, error: 'theme must be a valid theme id' });
  }
});
