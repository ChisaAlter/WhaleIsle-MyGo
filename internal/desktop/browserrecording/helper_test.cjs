// One synthetic, in-memory protocol fixture. No browser, page pixels or codec.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const script = fs.readFileSync(__dirname + '/helper.js', 'utf8');
(async () => {
  let stopped = 0, requested = 0, lastRecorder, imageGate = null, supported = true, supportedMime = 'video/webm;codecs=vp8', dimensions = { width: 1921, height: 1081 };
  const track = { requestFrame() { requested++; }, stop() { stopped++; } };
  const stream = { getTracks: () => [track], getVideoTracks: () => [track] };
  class Recorder extends EventTarget {
    static isTypeSupported(t) { return supported && t === supportedMime; }
    constructor(s, opts) { super(); assert.equal(s, stream); this.mimeType = opts.mimeType; this.state = 'inactive'; lastRecorder = this; }
    start() { this.state = 'recording'; }
    stop() {
      this.state = 'inactive';
      queueMicrotask(() => {
        const e = new Event('dataavailable'); e.data = new Blob([Uint8Array.of(1, 2, 3, 4)]);
        this.dispatchEvent(e); this.dispatchEvent(new Event('stop'));
      });
    }
  }
  const canvas = { width: 0, height: 0, captureStream(fps) { assert.equal(fps, 12); return stream; }, getContext: () => ({ fillRect() {}, drawImage() {} }) };
  const makeContext = () => vm.createContext({ location: { href: 'about:blank' }, document: { head: { appendChild() {} }, createElement: tag => tag === 'canvas' ? canvas : {} }, MediaRecorder: Recorder, Blob, Uint8Array, createImageBitmap: async () => imageGate || ({ ...dimensions, close() {} }), atob: s => Buffer.from(s, 'base64').toString('binary'), btoa: s => Buffer.from(s, 'binary').toString('base64'), setTimeout: (fn, ms) => { assert.notEqual(ms, 120000, 'original recording has no duration cutoff'); return setTimeout(fn, ms); }, clearTimeout: timer => { if (typeof timer !== 'symbol') clearTimeout(timer); }, Date, Error, Object });
  const context = makeContext();
  assert.equal(vm.runInContext(script, context), true);
  const api = context.__whaleBrowserRecorder;
  assert.equal(await api.start('AA=='), 'video/webm;codecs=vp8');
  assert.equal(canvas.width, 1921); assert.equal(canvas.height, 1081);
  // More than the former 256 chunks must remain valid without a time cutoff.
  for (let i = 0; i < 300; i++) { const e = new Event('dataavailable'); e.data = new Blob([Uint8Array.of(7)]); lastRecorder.dispatchEvent(e); }
  await api.frame('AA==');
  assert.equal(requested, 2);
  const info = await api.finish(); assert.equal(info.bytes, 304); assert.equal(info.mime, 'video/webm;codecs=vp8');
  await assert.rejects(api.read(1), /offset/);
  const part = await api.read(0); assert.equal(part.offset, 0); assert.equal(part.length, 304); assert.equal(Buffer.from(part.data, 'base64').length, 304);
  await assert.rejects(api.read(4), /offset/);
  await assert.rejects(api.frame('AA=='), /phase/);
  api.cancel(); assert.ok(stopped >= 1);
  await assert.rejects(api.read(0), /phase/);
  // Bound enforcement uses a synthetic Blob-sized event, allocating no video.
  const oversized = makeContext(); vm.runInContext(script, oversized);
  await oversized.__whaleBrowserRecorder.start('AA==');
  const allowed = new Event('dataavailable'); allowed.data = { size: 65 * 1024 * 1024 }; lastRecorder.dispatchEvent(allowed);
  assert.equal(await oversized.__whaleBrowserRecorder.frame('AA=='), true);
  const huge = new Event('dataavailable'); huge.data = { size: 512 * 1024 * 1024 + 1 }; lastRecorder.dispatchEvent(huge);
  await assert.rejects(oversized.__whaleBrowserRecorder.frame('AA=='), /512 MiB/);
  // One decode pending rejects another frame, and cancellation drops the first.
  const pressure = makeContext(); vm.runInContext(script, pressure);
  await pressure.__whaleBrowserRecorder.start('AA==');
  let release; imageGate = new Promise(resolve => { release = resolve; });
  const pending = pressure.__whaleBrowserRecorder.frame('AA==');
  await assert.rejects(pressure.__whaleBrowserRecorder.frame('AA=='), /backpressure/);
  pressure.__whaleBrowserRecorder.cancel(); release({ ...dimensions, close() {} });
  await assert.rejects(pending, /phase/); imageGate = null;
  supportedMime = 'video/mp4'; dimensions = { width: 3840, height: 2160 };
  const fallback = makeContext(); vm.runInContext(script, fallback);
  assert.equal(await fallback.__whaleBrowserRecorder.start('AA=='), 'video/mp4');
  assert.equal(canvas.width, 3840); assert.equal(canvas.height, 2160);
  assert.equal((await fallback.__whaleBrowserRecorder.finish()).mime, 'video/mp4');
  fallback.__whaleBrowserRecorder.cancel();
  supported = false;
  assert.throws(() => vm.runInContext(script, makeContext()), /codec unavailable/);
  console.log('Synthetic helper protocol passed (transfer, limits, backpressure, cancellation, missing codec); real MediaRecorder/playback not exercised.');
})().catch(e => { console.error(e); process.exitCode = 1; });
