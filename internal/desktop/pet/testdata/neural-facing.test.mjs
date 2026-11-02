import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import test from 'node:test';

const asset = name => readFileSync(new URL('../assets/' + name, import.meta.url), 'utf8');
const identity = [1, 0, 0, 1, 0, 0];
const reflected = [-1, 0, 0, 1, 780, 0];
const flush = async () => { for (let i = 0; i < 100; i++) await Promise.resolve(); };
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r; }); return {promise, resolve}; };

function context(canvas, trace = []) {
  let matrix = identity.slice();
  const stack = [], draws = [];
  const ctx = {
    canvas, draws, trace,
    get matrix() { return matrix; },
    get stackDepth() { return stack.length; },
    save() { trace.push(['save']); stack.push(matrix.slice()); },
    restore() { trace.push(['restore']); matrix = stack.pop() || identity.slice(); },
    translate(x, y) { trace.push(['translate', x, y]); matrix[4] += matrix[0] * x + matrix[2] * y; matrix[5] += matrix[1] * x + matrix[3] * y; },
    scale(x, y) { trace.push(['scale', x, y]); matrix[0] *= x; matrix[1] *= x; matrix[2] *= y; matrix[3] *= y; matrix = matrix.map(v => v || 0); },
    rotate(radians) { const [a, b, c, d, e, f] = matrix, cos = Math.cos(radians), sin = Math.sin(radians); matrix = [a * cos + c * sin, b * cos + d * sin, c * cos - a * sin, d * cos - b * sin, e, f].map(v => v || 0); },
    clearRect(...args) { trace.push(['clearRect', ...args]); assert.deepEqual(matrix, identity, 'each clear precedes this frame transform'); },
    drawImage(image, ...args) { trace.push(['drawImage', ...args]); if (ctx.failNextDraw) {ctx.failNextDraw = false; throw Error('one target draw failure');} if (ctx.failDraw) throw Error('target draw failure'); draws.push({image, args, matrix: matrix.slice()}); },
    getImageData() { return {data: new Uint8ClampedArray(512 * 512 * 4)}; },
    createImageData() { return {data: new Uint8ClampedArray(512 * 512 * 4)}; },
    putImageData() {},
  };
  return ctx;
}

function srFixture({supported = true, fail = false} = {}) {
  const canvases = [], setting = {fail};
  const document = {createElement() {
    const canvas = {width: 0, height: 0};
    canvas.ctx = context(canvas); canvas.getContext = () => canvas.ctx; canvases.push(canvas); return canvas;
  }};
  class Upscaler {
    static isSupported() { return supported; }
    attachSource(source, target) { target.width = source.width * 2; target.height = source.height * 2; }
    upscale() { if (setting.fail) throw Error('WebGL lost'); }
    detachSource() {}
  }
  const scope = {document};
  vm.runInNewContext(asset('pet-super-resolution.js'), scope);
  const sr = scope.WhalePetSuperResolution.create({ImageUpscaler: Upscaler, ANIME4KJS_SIMPLE_M_2X: 'fixed-original-preset'});
  const target = document.createElement('canvas'); target.width = 780; target.height = 984;
  return {sr, target, canvases, setting};
}

test('production SR and raw paths reflect only the visible frame around its centre', () => {
  for (const supported of [true, false]) for (const facing of [1, -1]) {
    const f = srFixture({supported}), source = {rawFrame: true};
    f.sr.paint(source, f.target.ctx, facing);
    const draw = f.target.ctx.draws.at(-1);
    assert.deepEqual(draw.matrix, facing === -1 ? reflected : identity);
    assert.deepEqual(draw.args, supported ? [0, 0, 780, 984] : [60, 20, 390, 492, 0, 0, 780, 984]);
    assert.deepEqual(f.target.ctx.matrix, identity);
    assert.equal(f.target.ctx.stackDepth, 0);
    assert.deepEqual(f.target.ctx.trace[0], ['clearRect', 0, 0, 780, 984]);
    assert.equal(f.target.ctx.trace[1][0], 'save');
    assert.equal(f.target.ctx.trace.at(-1)[0], 'restore');
    assert.deepEqual(f.canvases.flatMap(c => c === f.target ? [] : c.ctx.draws).map(d => d.matrix), supported ? Array(7).fill(identity) : [], 'RGB crop/bleed and independent alpha retain their original coordinates');
    assert.equal(f.target.width, 780); assert.equal(f.target.height, 984);
    f.sr.close();
  }
});

test('SR context loss and alternating facing preserve raw fallback orientation without accumulated transforms', () => {
  const f = srFixture(), source = {};
  f.sr.paint(source, f.target.ctx, -1);
  f.setting.fail = true;
  f.sr.paint(source, f.target.ctx, -1);
  f.sr.paint(source, f.target.ctx, 1);
  f.sr.paint(source, f.target.ctx, -1);
  assert.equal(f.sr.ready, false);
  assert.deepEqual(f.target.ctx.draws.map(d => d.matrix), [reflected, reflected, identity, reflected]);
  assert.deepEqual(f.target.ctx.draws.slice(1).map(d => d.args), Array(3).fill([60, 20, 390, 492, 0, 0, 780, 984]));
  assert.deepEqual(f.target.ctx.matrix, identity); assert.equal(f.target.ctx.stackDepth, 0);
  f.sr.close(); const count = f.target.ctx.draws.length; f.sr.paint(source, f.target.ctx, -1); assert.equal(f.target.ctx.draws.length, count);
});

test('a failed visible SR draw restores the transform before raw fallback', () => {
  const f = srFixture(), source = {}; f.target.ctx.failNextDraw = true;
  f.sr.paint(source, f.target.ctx, -1);
  assert.equal(f.sr.ready, false);
  assert.equal(f.target.ctx.draws.length, 1);
  assert.deepEqual(f.target.ctx.draws[0].matrix, reflected);
  assert.deepEqual(f.target.ctx.draws[0].args, [60, 20, 390, 492, 0, 0, 780, 984]);
  assert.equal(f.target.ctx.trace.filter(e => e[0] === 'save').length, 2);
  assert.equal(f.target.ctx.trace.filter(e => e[0] === 'restore').length, 2);
  assert.deepEqual(f.target.ctx.matrix, identity); assert.equal(f.target.ctx.stackDepth, 0);
  f.sr.close();
});

test('final draw failures always restore the caller transform', () => {
  const f = srFixture({supported: false}); f.target.ctx.failDraw = true;
  assert.throws(() => f.sr.paint({}, f.target.ctx, -1), /target draw failure/);
  assert.deepEqual(f.target.ctx.matrix, identity); assert.equal(f.target.ctx.stackDepth, 0);
  f.sr.close();
});

function runtimeFixture({state = {state: '', facing: 1}, firstGate = null, supported = true} = {}) {
  const trace = [], raf = new Map(), timers = new Map(), events = new Map(), nodes = new Map(), sessions = [], poses = [];
  let serial = 0, contextVM, runGate = null, currentState = state, failFrame = false;
  const node = name => {
    if (!nodes.has(name)) {
      const element = {name, width: 0, height: 0, style: {setProperty() {}}, textContent: '', hidden: false,
        addEventListener() {}, removeAttribute() {}, setAttribute() {}, replaceChildren() {}, remove() {}};
      element.ctx = context(element); element.getContext = () => element.ctx; nodes.set(name, element);
    }
    return nodes.get(name);
  };
  class Observer { observe() {} disconnect() {} }
  class Upscaler {
    static isSupported() { return supported; }
    attachSource(source, target) { target.width = source.width * 2; target.height = source.height * 2; }
    upscale() {}
    detachSource() { trace.push(['sr-close']); }
  }
  const window = {addEventListener: (name, fn) => events.set(name, fn)};
  const scope = {window, ResizeObserver: Observer, MutationObserver: Observer, Float32Array, Uint8ClampedArray, Set, Map, Math, Error, Number, Object, URL, Promise, Date,
    performance: {now: () => 1000}, location: {href: 'https://pet.local/index.html'},
    requestAnimationFrame: fn => { raf.set(++serial, fn); return serial; }, cancelAnimationFrame: id => raf.delete(id),
    setTimeout: (fn, ms) => { timers.set(++serial, {fn, ms}); return serial; }, clearTimeout: id => timers.delete(id),
    setInterval: (fn, ms) => { timers.set(++serial, {fn, ms}); return serial; }, clearInterval: id => timers.delete(id),
    Anime4KJS: {ImageUpscaler: Upscaler, ANIME4KJS_SIMPLE_M_2X: 'original'},
  };
  scope.document = {hidden: false, body: {dataset: {behavior: '{}', neural: 'true', rig: 'true', sleeping: 'false'}}, documentElement: node('html'),
    getElementById: node, querySelector: query => query === '.task' ? node('task') : null, addEventListener() {}, removeEventListener() {},
    createElement: type => node('created-' + type + '-' + ++serial), head: {appendChild(tag) { queueMicrotask(() => { if (['tha4.js', 'pet-rig.js'].includes(tag.src)) vm.runInContext(asset(tag.src), contextVM); tag.onload?.(); }); }}};
  scope.Image = class {set src(name) { this.naturalWidth = name.includes('character') ? 512 : 502; this.naturalHeight = name.includes('character') ? 512 : 695; queueMicrotask(() => this.onload?.()); }};
  scope.fetch = async () => ({ok: true, json: async () => JSON.parse(asset('rig/manifest.json'))});
  window.WhalePetInteraction = {create: () => ({pose(target) { target.fill(0); target[12] = .35; target[40] = .6; }, renderState: () => currentState, close() {}})};
  window.WhalePetCareAnimation = {create: () => ({pose(target) { target[26] = .2; }, update() {}, close() {}})};
  window.WhalePetBehavior = {create: () => ({powerSaving: () => false, close() {}})};
  const pixels = new Float32Array(512 * 512 * 4); pixels.fill(255, 512 * 512 * 3);
  scope.ort = {env: {wasm: {}}, Tensor: class {constructor(type, data, dims) {Object.assign(this, {type, data, dims});} dispose() {}}, InferenceSession: {async create() {
    let runCount = 0;
    const session = {async run(feeds) {
      poses.push(Array.from(feeds.pose.data)); const gate = runCount++ === 0 ? firstGate : runGate;
      runGate = null; if (gate) await gate.promise;
      if (failFrame) throw Error('inference failed');
      return {rgba_f: {dims: [1, 4, 512, 512], data: pixels, dispose() {trace.push(['output-close']);}}};
    }, async release() {trace.push(['session-close']);}};
    sessions.push(session); return session;
  }}};
  contextVM = vm.createContext(scope); vm.runInContext(asset('pet-super-resolution.js'), contextVM);
  const start = vm.runInContext(asset('pet-runtime.js'), contextVM);
  return {scope, start, trace, raf, timers, events, poses, canvas: node('sprite'), setState(next) {currentState = next;}, gateRun() { runGate = deferred(); return runGate; }, failNextFrame() {failFrame = true;},
    async tick(now) {const [id, fn] = [...raf].find(([, fn]) => fn.constructor.name === 'AsyncFunction'); raf.delete(id); await fn(now); await flush();},
    close() {events.get('pagehide')?.();}};
}

async function assertReady(f) { await flush(); await f.start; assert.equal(f.scope.document.body.dataset.renderer, 'neural'); }
function lastMatrix(f) { return f.canvas.ctx.draws.at(-1).matrix; }

test('first neural frame keeps its sampled facing when state changes during provider startup', async () => {
  const firstGate = deferred(), state = {state: '', facing: -1}, f = runtimeFixture({firstGate, state});
  await flush(); assert.equal(f.poses.length, 1); state.facing = 1; firstGate.resolve(); await assertReady(f);
  assert.deepEqual(lastMatrix(f), reflected); assert.ok(f.poses[0].every(x => x === 0), 'provider first-frame validation pose remains unchanged');
  await f.tick(1000); assert.deepEqual(lastMatrix(f), identity);
  assert.equal(f.canvas.width, 780); assert.equal(f.canvas.height, 984);
  assert.equal(f.poses[1][12], Math.fround(.35)); assert.equal(f.poses[1][40], Math.fround(.6)); assert.equal(f.poses[1][26], Math.fround(.2));
  f.close(); await flush();
});

test('in-flight facing snapshots, hanging exception and return to feet work through SR and raw output', async () => {
  for (const supported of [true, false]) {
    const f = runtimeFixture({supported}); await assertReady(f);
    assert.equal(f.scope.document.body.dataset.superResolution, supported ? 'anime4k' : 'raw');
    const state = {state: '', facing: -1}; f.setState(state);
    const gate = f.gateRun(), pending = f.tick(1000); await flush(); state.facing = 1; gate.resolve(); await pending;
    assert.deepEqual(lastMatrix(f), reflected, 'pending frame keeps the earlier left-facing snapshot');
    await f.tick(1050); assert.deepEqual(lastMatrix(f), identity);
    state.state = 'pick-up'; state.facing = -1;
    await f.tick(1100); assert.deepEqual(lastMatrix(f), identity, 'held/thrown frame does not use feet mirroring');
    state.state = ''; await f.tick(1150); assert.deepEqual(lastMatrix(f), reflected, 'landed feet frame resumes left-facing mirror');
    state.state = 'sleep'; await f.tick(1200); assert.deepEqual(lastMatrix(f), reflected, 'sleep remains a feet-pivot state');
    assert.deepEqual(f.canvas.ctx.matrix, identity); assert.equal(f.canvas.ctx.stackDepth, 0);
    assert.ok(f.poses.slice(1).every(p => p[12] === Math.fround(.35) && p[40] === Math.fround(.6) && p[26] === Math.fround(.2)), 'mirroring never edits pose channels');
    f.close(); await flush();
  }
});

test('close during an in-flight reflected frame cannot repaint or restore a renderer', async () => {
  const f = runtimeFixture(); await assertReady(f); f.setState({state: '', facing: -1});
  const gate = f.gateRun(), pending = f.tick(1000); await flush();
  const painted = f.canvas.ctx.draws.length; f.close(); gate.resolve(); await pending;
  assert.equal(f.canvas.ctx.draws.length, painted); assert.equal(f.raf.size, 0); assert.equal(f.timers.size, 0);
  assert.equal(f.trace.filter(e => e[0] === 'session-close').length, 1); assert.equal(f.trace.filter(e => e[0] === 'sr-close').length, 1);
  assert.deepEqual(f.canvas.ctx.matrix, identity);
});

test('a reflected neural frame followed by inference failure leaves the rig unreflected', async () => {
  const f = runtimeFixture({state: {state: '', facing: -1}}); await assertReady(f);
  assert.deepEqual(lastMatrix(f), reflected);
  f.setState({state: '', facing: 1}); f.failNextFrame(); await f.tick(1000); await flush();
  assert.equal(f.scope.document.body.dataset.renderer, 'rig');
  const rigDraws = f.canvas.ctx.draws.slice(1);
  assert.equal(rigDraws.length, 2, 'actual fallback rig draws tail and shell');
  assert.ok(rigDraws.every(draw => draw.matrix[0] > 0), 'the previous neural reflection does not leak into the rig');
  assert.deepEqual(f.canvas.ctx.matrix, identity); assert.equal(f.canvas.ctx.stackDepth, 0);
  f.close(); await flush();
});

test('provider completion after close cannot paint the initial reflected frame', async () => {
  const firstGate = deferred(), f = runtimeFixture({state: {state: '', facing: -1}, firstGate});
  await flush(); f.close(); firstGate.resolve(); await flush(); await f.start;
  assert.equal(f.canvas.ctx.draws.length, 0); assert.equal(f.raf.size, 0); assert.equal(f.timers.size, 0);
  assert.equal(f.trace.filter(e => e[0] === 'session-close').length, 1);
});
