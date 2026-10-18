'use strict';
// Adapted from ChisaAlter/WhaleIsle 90465b67 pet-live2d.js (MIT, Chisa 2026).
// Same 512px premultiplied input, 45-channel pose, byte-scale CHW output.
// Deliberately no Electron bridge, full-screen surface, rig, growth or chat.
(function(root) {
  const N = 512 * 512;
  function imageCHW(rgba) {
    if (rgba.length !== N * 4) throw new Error('THA4 image shape');
    const data = new Float32Array(N * 4);
    for (let i = 0; i < N; i++) {
      const a = rgba[i * 4 + 3] / 255;
      for (let c = 0; c < 3; c++) data[c * N + i] = rgba[i * 4 + c] / 255 * a * 2 - 1;
      data[3 * N + i] = a * 2 - 1;
    }
    return data;
  }
  function createEngine(ort, session, image) {
    const input = new ort.Tensor('float32', image, [1, 4, 512, 512]);
    const pose = new ort.Tensor('float32', new Float32Array(45), [1, 45]);
    const pixels = new Uint8ClampedArray(N * 4);
    let active = null, stopped = false, closing = null;
    async function run(values) {
      let results;
      try {
        pose.data.set(values);
        results = await session.run({ image: input, pose });
        if (stopped) return null;
        const out = results.rgba_f || results.rgba;
        if (!out || out.dims.join(',') !== '1,4,512,512') throw new Error('THA4 output shape');
        const raw = out.location === 'gpu-buffer' ? await out.getData() : out.data;
        if (!raw || raw.length !== N * 4) throw new Error('THA4 output shape');
        if (stopped) return null;
        for (let i = 0; i < N; i++) for (let c = 0; c < 4; c++) pixels[i * 4 + c] = raw[c * N + i];
        return pixels;
      } finally {
        // Latest upstream fix: every output, once, even on readback errors.
        if (results) for (const out of new Set(Object.values(results))) { try { out?.dispose?.(); } catch {} }
      }
    }
    return {
      async frame(values) {
        if (stopped || active) return null;
        active = run(values);
        try { return await active; } finally { active = null; }
      },
      close() {
        if (closing) return closing;
        stopped = true;
        closing = (async () => {
          try { await active; } catch {}
          try { input.dispose(); } finally { try { pose.dispose(); } finally { await session.release(); } }
        })();
        return closing;
      }
    };
  }
  root.WhaleTHA4 = { imageCHW, createEngine };
})(globalThis);
