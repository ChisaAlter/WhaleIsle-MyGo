// One synthetic, in-memory protocol fixture. No browser, page pixels or codec.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const script = fs.readFileSync(__dirname + '/helper.js', 'utf8');
(async () => {
  let stopped = 0, requested = 0, durationStop, lastRecorder, imageGate = null, supported = true;
  const track = { requestFrame() { requested++; }, stop() { stopped++; } };
  const stream = { getTracks: () => [track], getVideoTracks: () => [track] };
  class Recorder extends EventTarget {
    static isTypeSupported(t) { return supported && t === 'video/webm;codecs=vp8'; }
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
  const canvas = { width: 0, height: 0, captureStream(fps) { assert.equal(fps, 0); return stream; }, getContext: () => ({ fillRect() {}, drawImage() {} }) };
  const makeContext = () => vm.createContext({ location: { href: 'about:blank' }, document: { head: { appendChild() {} }, createElement: tag => tag === 'canvas' ? canvas : {} }, MediaRecorder: Recorder, Blob, Uint8Array, createImageBitmap: async () => imageGate || ({ width: 2, height: 2, close() {} }), atob: s => Buffer.from(s, 'base64').toString('binary'), btoa: s => Buffer.from(s, 'binary').toString('base64'), setTimeout: (fn, ms) => { if (ms === 120000) { durationStop = fn; return Symbol(); } return setTimeout(fn, ms); }, clearTimeout: timer => { if (typeof timer !== 'symbol') clearTimeout(timer); }, Date, Error, Object });
  const context = makeContext();
  assert.equal(vm.runInContext(script, context), true);
  const api = context.__whaleBrowserRecorder;
  assert.equal(await api.start('AA=='), 'video/webm;codecs=vp8');
  await api.frame('AA==');
  assert.equal(requested, 2);
  const info = await api.finish(); assert.equal(info.bytes, 4); assert.equal(info.mime, 'video/webm;codecs=vp8');
  await assert.rejects(api.read(1), /offset/);
  const part = await api.read(0); assert.equal(part.offset, 0); assert.equal(part.length, 4); assert.equal(part.data, 'AQIDBA==');
  await assert.rejects(api.read(4), /offset/);
  await assert.rejects(api.frame('AA=='), /phase/);
  api.cancel(); assert.ok(stopped >= 1);
  await assert.rejects(api.read(0), /phase/);
  // The helper itself stops encoding at the duration deadline, even if its
  // native owner has not finished one capture/consumption cycle.
  const timed = makeContext(); vm.runInContext(script, timed);
  await timed.__whaleBrowserRecorder.start('AA=='); durationStop();
  assert.equal(await timed.__whaleBrowserRecorder.frame('AA=='), 'limit');
  assert.equal((await timed.__whaleBrowserRecorder.finish()).bytes, 4);
  timed.__whaleBrowserRecorder.cancel();
  // Bound enforcement uses a synthetic Blob-sized event, allocating no video.
  const oversized = makeContext(); vm.runInContext(script, oversized);
  await oversized.__whaleBrowserRecorder.start('AA==');
  const huge = new Event('dataavailable'); huge.data = { size: 64 * 1024 * 1024 + 1 }; lastRecorder.dispatchEvent(huge);
  await assert.rejects(oversized.__whaleBrowserRecorder.frame('AA=='), /64 MiB/);
  // One decode pending rejects another frame, and cancellation drops the first.
  const pressure = makeContext(); vm.runInContext(script, pressure);
  await pressure.__whaleBrowserRecorder.start('AA==');
  let release; imageGate = new Promise(resolve => { release = resolve; });
  const pending = pressure.__whaleBrowserRecorder.frame('AA==');
  await assert.rejects(pressure.__whaleBrowserRecorder.frame('AA=='), /backpressure/);
  pressure.__whaleBrowserRecorder.cancel(); release({ width: 2, height: 2, close() {} });
  await assert.rejects(pending, /phase/); imageGate = null;
  supported = false;
  assert.throws(() => vm.runInContext(script, makeContext()), /codec unavailable/);
  console.log('Synthetic helper protocol passed (transfer, limits, backpressure, cancellation, missing codec); real MediaRecorder/playback not exercised.');
})().catch(e => { console.error(e); process.exitCode = 1; });
