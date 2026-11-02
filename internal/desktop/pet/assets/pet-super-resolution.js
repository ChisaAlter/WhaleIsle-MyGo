'use strict';
// Port of native pet-live2d.js initSr/srFrame (Chisa 2026, MIT).
// Keep feathered alpha separate from the RGB CNN input, with two RGB bleed
// passes to avoid dark halos from the transparent-black model background.
(function (root) {
  const crop = {x:60,y:20,width:390,height:492};
  function surface(width, height) {
    const canvas = document.createElement('canvas'); canvas.width = width; canvas.height = height;
    return canvas;
  }
  function create(library = root.Anime4KJS) {
    let up = null, closed = false, ready = false;
    const input = surface(crop.width, crop.height), blur = surface(crop.width, crop.height);
    const gl = surface(1, 1), output = surface(crop.width * 2, crop.height * 2);
    function disable() {
      ready = false;
      try { up?.detachSource?.(); } catch {}
      up = null;
    }
    try {
      if (library?.ImageUpscaler?.isSupported()) {
        up = new library.ImageUpscaler(library.ANIME4KJS_SIMPLE_M_2X);
        up.attachSource(input, gl);
        output.width = gl.width; output.height = gl.height;
        ready = output.width === crop.width * 2 && output.height === crop.height * 2;
        if (!ready) disable();
      }
    } catch { disable(); }
    function drawVisible(target, image, facing, raw = false) {
      const w = target.canvas.width, h = target.canvas.height;
      target.clearRect(0, 0, w, h);
      target.save();
      try {
        // Mirror only the final visible frame. The SR input and alpha mask
        // retain their original crop, coordinates and premultiplication.
        if (facing === -1) { target.translate(w, 0); target.scale(-1, 1); }
        if (raw) target.drawImage(image, crop.x, crop.y, crop.width, crop.height, 0, 0, w, h);
        else target.drawImage(image, 0, 0, w, h);
      } finally { target.restore(); }
    }
    let cachedSource = null;
    function prepare(source) {
        if (closed) return;
        cachedSource = source;
        if (ready) {
          try {
            const c = input.getContext('2d'), b = blur.getContext('2d'), o = output.getContext('2d');
            c.clearRect(0, 0, crop.width, crop.height);
            c.drawImage(source, crop.x, crop.y, crop.width, crop.height, 0, 0, crop.width, crop.height);
            for (let i = 0; i < 2; i++) {
              b.clearRect(0, 0, crop.width, crop.height);
              b.filter = 'blur(10px)'; b.drawImage(input, 0, 0); b.filter = 'none';
              c.globalCompositeOperation = 'destination-over'; c.drawImage(blur, 0, 0);
              c.globalCompositeOperation = 'source-over';
            }
            up.upscale();
            o.clearRect(0, 0, output.width, output.height); o.drawImage(gl, 0, 0);
            o.globalCompositeOperation = 'destination-in';
            o.drawImage(source, crop.x, crop.y, crop.width, crop.height, 0, 0, output.width, output.height);
            o.globalCompositeOperation = 'source-over';
            return;
          } catch { disable(); }
        }
    }
    function present(target, facing = 1) {
      if (closed || !cachedSource) return;
      if (ready) {
        try { drawVisible(target, output, facing); return; } catch { disable(); }
      }
      drawVisible(target, cachedSource, facing, true);
    }
    return {
      get ready() { return ready && !closed; },
      prepare, present,
      paint(source, target, facing = 1) { prepare(source); present(target, facing); },
      close() {
        if (closed) return; closed = true; cachedSource=null; disable();
        // The vendored API has no destroy method. Release its attached source
        // and the owned context explicitly rather than leaking a WebGL context.
        try { gl.getContext('webgl')?.getExtension('WEBGL_lose_context')?.loseContext(); } catch {}
        for (const canvas of [input, blur, gl, output]) canvas.width = canvas.height = 1;
      }
    };
  }
  root.WhalePetSuperResolution = {create};
})(globalThis);
