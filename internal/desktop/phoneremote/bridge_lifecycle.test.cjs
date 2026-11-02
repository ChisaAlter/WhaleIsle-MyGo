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

function fixture({stopError = false} = {}) {
  const input = new EventEmitter(), processStub = new EventEmitter();
  const events = [], exits = [], intervals = new Set(), reads = [], syncs = [];
  const deviceCalls = [];
  let remote, constructorOptions, stops = 0, revocations = 0;
  let deviceRelease;
  const devicePending = new Promise(resolve => { deviceRelease = resolve; });
  const url = 'https://example.test/#offer=' + Buffer.from(JSON.stringify({
    v: 2, authBootstrap: {expiresAtMs: Date.now() + 60_000},
  })).toString('base64url');
  class FakeRemote extends EventEmitter {
    constructor(options) { super(); remote = this; constructorOptions = options; this.daemon = null; }
    sync() {
      const pending = {};
      const promise = new Promise((resolve, reject) => { pending.resolve = resolve; pending.reject = reject; });
      syncs.push(pending);
      return promise;
    }
    snapshot() {
      return {listening: !!this.daemon, relayConnected: true, urls: [{pairingUrl: url}], devices: this.unbound ? [] : [{id: 'fictional-device', name: this.deviceName || 'Phone'}]};
    }
    async renameDevice(id, name) { deviceCalls.push('rename'); await devicePending; this.deviceName = name; }
    async rotateToken() { deviceCalls.push('rotate'); await devicePending; }
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

for (const operation of ['rename', 'unbind', 'rotate']) {
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
