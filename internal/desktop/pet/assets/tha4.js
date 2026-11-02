'use strict';
// Adapted from ChisaAlter/WhaleIsle 90465b67 pet-live2d.js (MIT, Chisa 2026).
// Same 512px premultiplied input, 45-channel pose, byte-scale CHW output.
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
  function createEngine(ort, session, image, options = {}) {
    if (image.length !== N * 4) throw new Error('THA4 image shape');
    const buffers = [];
    let input, pose, poseBuffer;
    // ort.env.webgpu.device may survive a prior provider attempt. Only the
    // successfully selected WebGPU session may receive external GPU tensors.
    const device = options.provider === 'webgpu' ? options.device : null;
    const usage = root.GPUBufferUsage;
    const onGpu = !!device && !!usage && typeof ort.Tensor.fromGpuBuffer === 'function';
    function gpuTensor(data, dims) {
      const buffer = device.createBuffer({size: data.byteLength, usage: usage.STORAGE | usage.COPY_DST});
      buffers.push(buffer);
      device.queue.writeBuffer(buffer, 0, data);
      return [ort.Tensor.fromGpuBuffer(buffer, {dataType:'float32', dims}), buffer];
    }
    try {
      if (onGpu) {
        [input] = gpuTensor(image, [1, 4, 512, 512]);
        [pose, poseBuffer] = gpuTensor(new Float32Array(45), [1, 45]);
      } else {
        if (options.capture) throw new Error('THA4 capture needs GPU inputs');
        input = new ort.Tensor('float32', image, [1, 4, 512, 512]);
        pose = new ort.Tensor('float32', new Float32Array(45), [1, 45]);
      }
    } catch (error) {
      try { input?.dispose(); } catch {}
      try { pose?.dispose(); } catch {}
      for (const buffer of buffers) buffer.destroy();
      throw error;
    }
    const pixels = new Uint8ClampedArray(N * 4);
    let active = null, stopped = false, closing = null;
    async function run(values) {
      let results;
      try {
        if (!values || values.length !== 45) throw new Error('THA4 pose shape');
        if (onGpu) device.queue.writeBuffer(poseBuffer, 0, values);
        else pose.data.set(values);
        results = await session.run({ image: input, pose });
        if (stopped) return null;
        const out = results.rgba_f || results.rgba;
        if (!out || out.dims.join(',') !== '1,4,512,512') throw new Error('THA4 output shape');
        // The compositor still needs RGBA bytes. GPU inputs remove repeated
        // image uploads; this output conversion intentionally retains readback.
        const raw = out.location === 'gpu-buffer' ? await out.getData() : out.data;
        if (!raw || raw.length !== N * 4) throw new Error('THA4 output shape');
        if (stopped) return null;
        for (let i = 0; i < N; i++) for (let c = 0; c < 4; c++) pixels[i * 4 + c] = raw[c * N + i];
        return pixels;
      } finally {
        if (results) for (const out of new Set(Object.values(results))) { try { out?.dispose?.(); } catch {} }
      }
    }
    return {
      gpuInputs: onGpu,
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
          try { input.dispose(); } finally {
            try { pose.dispose(); } finally {
              try { await session.release(); } finally { for (const buffer of buffers) buffer.destroy(); }
            }
          }
        })();
        return closing;
      }
    };
  }
  // Validate a real first frame before accepting a provider. A successful
  // session.create alone does not prove graph capture or its GPU bindings work.
  async function open(ort, url, image, isClosed = () => false) {
    const attempts = [{provider:'webnn',capture:false}, {provider:'webgpu',capture:true},
      {provider:'webgpu',capture:false}, {provider:'wasm',capture:false}];
    let failure = null;
    for (const attempt of attempts) {
      if (isClosed()) throw new Error('closed');
      let session = null, engine = null;
      try {
        const options = {executionProviders:[attempt.provider]};
        if (attempt.provider === 'webgpu') {
          options.enableGraphCapture = attempt.capture;
          options.preferredOutputLocation = 'gpu-buffer';
        }
        session = await ort.InferenceSession.create(url, options);
        if (isClosed()) throw new Error('closed');
        engine = createEngine(ort, session, image, {...attempt, device:ort.env.webgpu?.device});
        session = null; // The engine owns the successful session.
        const firstPixels = await engine.frame(new Float32Array(45));
        if (isClosed()) throw new Error('closed');
        let visible = false;
        for (let i = 3; i < firstPixels.length; i += 4) if (firstPixels[i] >= 4) {visible = true; break;}
        if (!visible) throw new Error('transparent first frame');
        return {engine, provider:attempt.provider, capture:attempt.capture, firstPixels};
      } catch (error) {
        failure = error;
        if (engine) { try { await engine.close(); } catch {} }
        else if (session) { try { await session.release(); } catch {} }
      }
    }
    throw failure || new Error('No supported inference provider');
  }
  root.WhaleTHA4 = { imageCHW, createEngine, open };
})(globalThis);
