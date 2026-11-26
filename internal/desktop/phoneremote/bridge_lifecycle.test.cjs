'use strict';
// The production adapter runs in a VM with a fictional constructor, pipes and
// timers. No original daemon, home/profile, listener, relay or credentials load.
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const {EventEmitter} = require('node:events');
const bridge = fs.readFileSync(path.join(__dirname, 'bridge.cjs'), 'utf8');
const deviceActions = fs.readFileSync(path.join(__dirname, 'device-actions.cjs'), 'utf8');
const settle = async () => { await new Promise(setImmediate); await new Promise(setImmediate); };

function fixture({stopError = false, pairingUrl} = {}) {
  const input = new EventEmitter(), processStub = new EventEmitter();
  const events = [], exits = [], intervals = new Set(), reads = [], syncs = [];
  const deviceCalls = [];
  let remote, constructorOptions, stops = 0, revocations = 0, mountedShell = null;
  let deviceRelease;
  const devicePending = new Promise(resolve => { deviceRelease = resolve; });
  let currentUrl = pairingUrl !== undefined ? pairingUrl : ('https://example.test/#offer=' + Buffer.from(JSON.stringify({
    v: 2, authBootstrap: {expiresAtMs: Date.now() + 60_000},
  })).toString('base64url'));
  class FakeRemote extends EventEmitter {
    constructor(options) { super(); remote = this; constructorOptions = options; this.daemon = null; }
    sync() {
      const pending = {};
      const promise = new Promise((resolve, reject) => { pending.resolve = resolve; pending.reject = reject; });
      syncs.push(pending);
      return promise;
    }
    snapshot() {
      return {listening: !!this.daemon, relayConnected: true, urls: [{pairingUrl: currentUrl}], devices: this.unbound ? [] : [{id: 'fictional-device', name: this.deviceName || 'Phone'}]};
    }
    async renameDevice(id, name) { deviceCalls.push('rename'); await devicePending; this.deviceName = name; }
    async rotateToken() { deviceCalls.push('rotate'); await devicePending; }
    async revokePairingLink() { deviceCalls.push('revoke-link'); await devicePending; }
    async unbindDevice(id) { deviceCalls.push('unbind'); await devicePending; this.unbound = true; }
    async stopDaemon() {
      ++stops;
      // Match the original manager's readyReject on cancellation. A separate
      // test resolves late instead, ensuring the adapter itself suppresses it.
      if (stopError) throw Error('fictional-private-stop-detail');
      for (const pending of syncs) pending.reject(Error('fictional-cancelled-start'));
      this.daemon = null;
    }
  }
  processStub.argv = ['node', '/fixture/src/main/dshd-remote.js', 'fictional-home'];
  processStub.execPath = 'fictional-node'; processStub.stdin = {};
  processStub.stdout = {write: text => events.push(JSON.parse(text))};
  processStub.exit = code => exits.push(code);
  const context = {
    __MYGO_GIT_SOURCES__: {}, __MYGO_MOBILE_DRAFT_UI__: '', installMobileDraftUI() {},
    __mygoShellRoutes: {mountShellRoutes(server, opts) { mountedShell = opts; }},
    loadOriginalGit: () => ({git: {setChangeRequestGenerator() {}}, authority: {}}),
    createGitGate: () => ({git: {}, configure() {}, revoke() { ++revocations; }}),
    process: processStub, console: {}, Buffer, URL,
    setInterval: fn => { intervals.add(fn); return fn; },
    clearInterval: fn => intervals.delete(fn),
    setTimeout: () => { throw Error('unexpected timer in fake runtime lifecycle'); }, clearTimeout() {},
    require(id) {
      reads.push(id);
      if (id === 'node:path') return path;
      if (id === 'node:module') return {createRequire: () => () => { throw Error('optional QR unavailable'); }};
      if (id === 'node:readline') return {createInterface: () => input};
      if (id === processStub.argv[1]) return {DshdRemote: FakeRemote};
      throw Error('unexpected runtime import: ' + id);
    },
  };
  vm.runInNewContext(deviceActions + '\n' + bridge, context, {filename: 'production-bridge.cjs'});
  return {
    input, processStub, events, exits, syncs, reads, deviceCalls,
    releaseDevice: () => deviceRelease(),
    get remote() { return remote; }, get options() { return constructorOptions; },
    get stops() { return stops; }, get revocations() { return revocations; },
    get mountedShell() { return mountedShell; },
    setPairingUrl(next) { currentUrl = next; if (remote) remote.pairing = {url: next}; },
    send(command) { input.emit('line', JSON.stringify(command)); },
    tick() { for (const fn of intervals) fn(); },
    ready(index = 0) { remote.daemon = {}; remote.emit('listening'); syncs[index].resolve(); },
  };
}

const start = f => f.send({action: 'start', config: {remoteEnabled: true, remoteMode: 'lan'}});
const assertNoOffer = f => assert.equal(f.events.some(event => event.url || event.qrPNG), false);

test('constructor uses a finite 60s ready budget and preserves existing adapter contracts', () => {
  const f = fixture();
  assert.equal(f.options.readyTimeoutMs, 60_000);
  assert.equal(f.options.stopTimeoutMs, undefined, 'do not extend the independent vendor stop budget');
  assert.equal(f.options.execPath, 'fictional-node');
  assert.equal(f.options.getHomeDir(), 'fictional-home');
  assert.equal(f.options.getHarnessOrigin(), '');
  assert.equal(f.options.getSessionCookie(), '');
  assert.equal(f.reads.filter(id => id === '/fixture/src/main/dshd-remote.js').length, 1);
});

for (const operation of ['stop', 'stdin-close', 'SIGTERM']) {
  test(`${operation} cancels an in-flight ready wait without waiting in its queue`, async () => {
    const f = fixture(); start(f); await settle();
    assert.equal(f.syncs.length, 1);
    f.send({action: 'refresh', config: {remoteEnabled: true}}); // queued behind cold start
    if (operation === 'stop') f.send({action: 'stop'});
    if (operation === 'stdin-close') f.input.emit('close');
    if (operation === 'SIGTERM') f.processStub.emit('SIGTERM');
    assert.equal(f.stops, 1, 'stopDaemon must run immediately, before pending sync settles');
    // Concurrent/repeated shutdown channels consume the same idempotent stop.
    f.send({action: 'stop'}); f.input.emit('close'); f.processStub.emit('SIGTERM');
    await settle(); f.tick();
    assert.equal(f.stops, 1);
    assert.equal(f.revocations, 1);
    assert.equal(f.syncs.length, 1, 'queued refresh must not restart the stopped daemon');
    assert.deepEqual(f.exits, [0]);
    assertNoOffer(f);
    assert.equal(f.events.some(event => event.kind === 'error'), false, 'cancelled startup is not a startup failure');
    assert.equal(f.events.at(-1).kind, 'snapshot');
    assert.equal(f.events.at(-1).listening, false);
  });
}

test('late ready resolution cannot publish or restart after stop', async () => {
  const f = fixture({stopError: true}); start(f); await settle();
  f.send({action: 'stop'}); await settle();
  assert.deepEqual(f.exits, [1]);
  f.ready(); await settle(); f.tick();
  f.send({action: 'start', config: {remoteEnabled: true}}); await settle();
  assert.equal(f.syncs.length, 1);
  assertNoOffer(f);
  assert.deepEqual(f.events.map(event => ({kind: event.kind, code: event.code})), [{kind: 'error', code: 'stop'}]);
  assert.equal(JSON.stringify(f.events).includes('fictional-private-stop-detail'), false);
});

test('normal startup failures stay visible and sanitized without automatic retries', async () => {
  const f = fixture(); start(f); await settle();
  f.syncs[0].reject(Error('fictional-private-startup-detail'));
  await settle(); f.tick(); f.tick(); await settle();
  assert.equal(f.syncs.length, 1);
  const errors = f.events.filter(event => event.kind === 'error');
  assert.equal(errors.length, 1);
  assert.equal(errors[0].code, 'operation');
  assert.equal(JSON.stringify(f.events).includes('fictional-private-startup-detail'), false);
  assertNoOffer(f);
  f.input.emit('close'); await settle();
  assert.deepEqual(f.exits, [0]);
});

test('stop ahead of a queued start prevents the constructor from starting its daemon', async () => {
  const f = fixture(); start(f); f.send({action: 'stop'});
  await settle();
  assert.equal(f.syncs.length, 0);
  assert.equal(f.stops, 1);
  assert.deepEqual(f.exits, [0]);
  assertNoOffer(f);
});

for (const operation of ['rename', 'unbind', 'rotate', 'revoke-link']) {
  test(`stopping pending ${operation} suppresses late results, snapshots and restart`, async () => {
    const f = fixture(); start(f); await settle(); f.ready(); await settle();
    f.send({action: 'device-action', id: '1', operation, deviceId: 'fictional-device', name: 'Renamed'});
    await settle();
    assert.deepEqual(f.deviceCalls, [operation]);
    f.send({action: 'stop'}); await settle();
    const emittedAtStop = f.events.length;
    f.releaseDevice(); await settle(); f.tick();
    assert.equal(f.events.length, emittedAtStop, 'no late action result or snapshot after stop');
    assert.equal(f.events.some(event => event.kind === 'device-action-result'), false);
    assert.equal(f.syncs.length, 1, 'unbind must not sync/restart after the desktop stopped');
    assert.deepEqual(f.deviceCalls, [operation], 'do not retry or replay the mutation');
    assert.deepEqual(f.exits, [0]);
  });
}

test('explicit reusable-link revocation does not stop the daemon or revoke paired-device Git access', async () => {
  const f = fixture(); start(f); await settle(); f.ready(); await settle();
  f.send({action:'device-action', id:'1', operation:'revoke-link'}); await settle();
  assert.deepEqual(f.deviceCalls, ['revoke-link']);
  f.releaseDevice(); await settle();
  assert.equal(f.stops, 0);
  assert.equal(f.revocations, 0);
  assert.equal(f.syncs.length, 1);
  assert.equal(f.events.at(-1).kind, 'device-action-result');
  assert.equal(f.events.at(-1).applied, true);
  f.send({action:'stop'}); await settle();
});

test('successful explicit start still emits the existing snapshot protocol', async () => {
  const f = fixture(); start(f); await settle();
  assert.equal(f.events.length, 0, 'do not publish readiness while sync is pending');
  f.ready(); await settle();
  assert.equal(f.events.at(-1).kind, 'snapshot');
  assert.equal(f.events.at(-1).listening, true);
  assert.equal(f.events.at(-1).relayConnected, true);
  assert.equal(f.events.at(-1).url.startsWith('https://example.test/#offer='), true);
  f.send({action: 'stop'}); await settle();
  assert.deepEqual(f.exits, [0]);
  assert.equal(f.events.at(-1).url, '');
});

test('offers keep expiry checks but never authorize retired HTTP shell routes', async () => {
  const encodeOffer = (payload, scheme = 'https://example.test/#offer=') =>
    scheme + Buffer.from(JSON.stringify(payload)).toString('base64url');
  const f = fixture();
  f.remote.mobileWebServer = {};
  start(f); await settle(); f.ready(); await settle();
  f.tick();
  assert.ok(f.mountedShell, 'shell routes should mount on remote.mobileWebServer');

  const valid = encodeOffer({
    v: 2,
    authBootstrap: {expiresAtMs: Date.now() + 60_000, pairingToken: 'pair-valid-1'},
  });
  f.setPairingUrl(valid);
  f.tick();
  assert.equal(f.events.at(-1).url, valid);
  assert.deepEqual(Array.from(f.mountedShell.tokens()), [], 'bootstrap token is never a business credential');

  const invalidCases = [
    ['expired', encodeOffer({v: 2, authBootstrap: {expiresAtMs: Date.now() - 1_000, pairingToken: 'pair-expired'}})],
    ['missing expiresAtMs', encodeOffer({v: 2, authBootstrap: {pairingToken: 'pair-missing-exp'}})],
    ['string expiresAtMs', encodeOffer({v: 2, authBootstrap: {expiresAtMs: String(Date.now() + 60_000), pairingToken: 'pair-str-exp'}})],
    ['non-finite NaN', encodeOffer({v: 2, authBootstrap: {expiresAtMs: null, pairingToken: 'pair-null-exp'}})],
    ['zero expiresAtMs', encodeOffer({v: 2, authBootstrap: {expiresAtMs: 0, pairingToken: 'pair-zero-exp'}})],
    ['negative expiresAtMs', encodeOffer({v: 2, authBootstrap: {expiresAtMs: -5, pairingToken: 'pair-neg-exp'}})],
    ['wrong version', encodeOffer({v: 1, authBootstrap: {expiresAtMs: Date.now() + 60_000, pairingToken: 'pair-v1'}})],
    ['non-http protocol', encodeOffer({v: 2, authBootstrap: {expiresAtMs: Date.now() + 60_000, pairingToken: 'pair-js'}}, 'javascript:alert(1)#offer=')],
    ['malformed base64 json', 'https://example.test/#offer=not-valid-json'],
    ['oversized url', 'https://example.test/#offer=' + 'a'.repeat(8200)],
  ];
  for (const [label, badUrl] of invalidCases) {
    f.setPairingUrl(badUrl);
    f.tick();
    assert.equal(f.events.at(-1).url, '', `usableURL must fail closed for ${label}`);
    assert.deepEqual(Array.from(f.mountedShell.tokens()), [], `HTTP shell remains closed for ${label}`);
  }
  f.send({action: 'stop'}); await settle();
});

test('real HTTP shell rejects bootstrap and legacy bearer while the SPA stays available', async () => {
  const http = require('node:http');
  const routes = require('./shell_routes.cjs');
  const f = fixture();
  f.remote.mobileWebServer = {};
  f.send({action:'start',config:{remoteEnabled:true,remoteMode:'lan',remoteToken:'fictional-old-token'}});
  await settle(); f.ready(); await settle();
  f.setPairingUrl('http://example.test/#offer='+Buffer.from(JSON.stringify({v:2,authBootstrap:{expiresAtMs:Date.now()+60000,pairingToken:'fictional-bootstrap'}})).toString('base64url'));
  const server = http.createServer((_req,res)=>res.end('owned static SPA'));
  routes.mountShellRoutes(server,f.mountedShell);
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
  try {
    const base='http://127.0.0.1:'+server.address().port;
    assert.equal(await (await fetch(base)).text(),'owned static SPA');
    for(const token of ['fictional-bootstrap','fictional-old-token']) {
      const response=await fetch(base+'/__remote__/shell/saveConfig',{method:'POST',headers:{authorization:'Bearer '+token,'content-type':'application/json'},body:'{"patch":{"remoteEnabled":false}}'});
      assert.equal(response.status,401);
      assert.equal(await response.text(),'unauthorized');
    }
    assert.equal(f.events.some(e=>e.kind==='host-save-config'),false);
  } finally {
    server.closeAllConnections();
    await new Promise(resolve=>server.close(resolve));
    f.send({action:'stop'});await settle();
  }
});

test('idle snapshot ticks with unchanged state emit zero redundant events', async () => {
  const f = fixture();
  start(f); await settle(); f.ready(); await settle();
  const countAfterReady = f.events.length;
  assert.ok(countAfterReady >= 1);
  // Repeated ticks with identical remote.snapshot() must not churn stdout.
  f.tick(); f.tick(); f.tick();
  assert.equal(f.events.length, countAfterReady, 'idle ticks emitted redundant unchanged snapshots');
  // Changing the offer URL must immediately emit a fresh snapshot on the next tick.
  const nextUrl = 'https://example.test/#offer=' + Buffer.from(JSON.stringify({
    v: 2, authBootstrap: {expiresAtMs: Date.now() + 90_000, pairingToken: 'rotated'},
  })).toString('base64url');
  f.setPairingUrl(nextUrl);
  f.tick();
  assert.equal(f.events.length, countAfterReady + 1);
  assert.equal(f.events.at(-1).url, nextUrl);
  f.send({action: 'stop'}); await settle();
});
