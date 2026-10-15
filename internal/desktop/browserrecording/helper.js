// Fixed application-owned encoder. This code runs ONLY in a newly created
// isolated about:blank page, never in the captured website. Only PNG bytes cross
// into this realm; there is no host bridge, URL, markup, DOM or filesystem API.
(() => {
  'use strict';
  if (location.href !== 'about:blank' || globalThis.__whaleBrowserRecorder) throw new Error('Invalid encoder owner');
  if (typeof MediaRecorder !== 'function' || typeof createImageBitmap !== 'function') throw new Error('WebM MediaRecorder unavailable');
  const mime = ['video/webm;codecs=vp8', 'video/webm;codecs=vp9', 'video/webm'].find(t => MediaRecorder.isTypeSupported(t));
  if (!mime) throw new Error('WebM codec unavailable');
  const policy = document.createElement('meta');
  policy.httpEquiv = 'Content-Security-Policy';
  policy.content = "default-src 'none'; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; form-action 'none'; base-uri 'none'";
  document.head.appendChild(policy);
  const canvas = document.createElement('canvas');
  const drawing = canvas.getContext('2d', { alpha: false });
  if (!drawing || typeof canvas.captureStream !== 'function') throw new Error('Canvas recording unavailable');
  const MAX_BYTES = 64 * 1024 * 1024, MAX_FRAME = 8 * 1024 * 1024, CHUNK = 256 * 1024;
  let phase = 'idle', busy = false, fatal = '', recorder = null, stream = null, track = null;
  let chunks = [], total = 0, artifact = null, offset = 0, watchdog = null, stopped = null;
  const stopTracks = () => { if (stream) for (const t of stream.getTracks()) t.stop(); stream = null; track = null; };
  const cancel = () => {
    phase = 'cancelled'; clearTimeout(watchdog);
    if (recorder && recorder.state !== 'inactive') { try { recorder.stop(); } catch (_) {} }
    stopTracks(); chunks = []; total = 0; artifact = null; canvas.width = canvas.height = 1;
  };
  const fail = message => { fatal = message; cancel(); };
  const requirePhase = expected => { if (fatal) throw new Error(fatal); if (phase !== expected) throw new Error('Invalid recording phase'); };
  const draw = async (data, first) => {
    if (busy) throw new Error('Frame backpressure violated');
    if (typeof data !== 'string' || data.length === 0 || data.length > Math.ceil(MAX_FRAME / 3) * 4) throw new Error('PNG frame limit exceeded');
    busy = true; let bitmap;
    try {
      const binary = atob(data);
      if (binary.length > MAX_FRAME) throw new Error('PNG frame limit exceeded');
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      bitmap = await createImageBitmap(new Blob([bytes], { type: 'image/png' }));
      if (!first && phase === 'limited') return;
      requirePhase(first ? 'starting' : 'recording');
      if (bitmap.width < 1 || bitmap.height < 1 || bitmap.width * bitmap.height > 4 * 1024 * 1024) throw new Error('PNG dimensions exceeded');
      if (first) {
        const scale = Math.min(1, 1280 / bitmap.width, 720 / bitmap.height);
        canvas.width = Math.max(2, Math.floor(bitmap.width * scale / 2) * 2);
        canvas.height = Math.max(2, Math.floor(bitmap.height * scale / 2) * 2);
      }
      const scale = Math.min(canvas.width / bitmap.width, canvas.height / bitmap.height);
      const w = bitmap.width * scale, h = bitmap.height * scale;
      drawing.fillStyle = '#000'; drawing.fillRect(0, 0, canvas.width, canvas.height);
      drawing.drawImage(bitmap, (canvas.width - w) / 2, (canvas.height - h) / 2, w, h);
      if (track) track.requestFrame();
    } finally { if (bitmap) bitmap.close(); busy = false; }
  };
  const start = async data => {
    requirePhase('idle'); phase = 'starting';
    try {
      await draw(data, true);
      stream = canvas.captureStream(0); track = stream.getVideoTracks()[0];
      if (!track || typeof track.requestFrame !== 'function' || stream.getTracks().length !== 1) throw new Error('Manual canvas frame recording unavailable');
      recorder = new MediaRecorder(stream, { mimeType: mime, videoBitsPerSecond: 4000000 });
      if (!/^video\/webm(?:;\s*codecs=(?:vp8|vp9))?$/i.test(recorder.mimeType)) throw new Error('Unexpected recorder MIME');
      stopped = new Promise(resolve => recorder.addEventListener('stop', resolve, { once: true }));
      recorder.addEventListener('error', () => fail('WebM encoder failed'));
      recorder.addEventListener('dataavailable', event => {
        if (phase === 'cancelled' || event.data.size === 0) return;
        if (event.data.size > MAX_BYTES - total || chunks.length >= 256) { fail('WebM recording exceeded 64 MiB'); return; }
        total += event.data.size; chunks.push(event.data);
      });
      recorder.start(1000); phase = 'recording'; track.requestFrame();
      // The encoder itself enforces the duration bound, even if a native
      // capture or the owner UI is stalled. No frame is accepted after this.
      watchdog = setTimeout(() => {
        if (phase !== 'recording') return;
        phase = 'limited';
        try { recorder.stop(); stopTracks(); } catch (_) { fail('Encoder could not stop at the duration limit'); }
      }, 120000);
      return recorder.mimeType;
    } catch (error) { cancel(); throw error; }
  };
  const frame = async data => {
    if (phase === 'limited') return 'limit';
    requirePhase('recording'); await draw(data, false);
    if (phase === 'limited') return 'limit';
    requirePhase('recording'); return true;
  };
  const finish = async () => {
    if (phase !== 'limited') requirePhase('recording');
    if (busy) throw new Error('Frame still pending');
    const alreadyStopped = phase === 'limited';
    phase = 'stopping'; clearTimeout(watchdog);
    try {
      let timer;
      try {
        if (!alreadyStopped) recorder.stop();
        await Promise.race([stopped, new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Encoder stop timed out')), 8000); })]);
      } finally { clearTimeout(timer); }
      clearTimeout(watchdog); stopTracks();
      if (fatal) throw new Error(fatal);
      if (phase !== 'stopping' || total === 0 || total > MAX_BYTES) throw new Error('No encoded WebM output');
      artifact = new Blob(chunks, { type: recorder.mimeType }); chunks = [];
      if (artifact.size !== total) throw new Error('WebM output length mismatch');
      phase = 'ready';
      return { mime: recorder.mimeType, bytes: total };
    } catch (error) { cancel(); throw error; }
  };
  const read = async requestedOffset => {
    requirePhase('ready');
    if (busy || requestedOffset !== offset || offset >= total) throw new Error('Invalid transfer offset');
    busy = true;
    try {
      const begin = offset;
      const bytes = new Uint8Array(await artifact.slice(begin, Math.min(total, begin + CHUNK)).arrayBuffer());
      requirePhase('ready');
      let binary = ''; for (let i = 0; i < bytes.length; i++) binary += String.fromCharCode(bytes[i]);
      offset += bytes.length;
      return { offset: begin, length: bytes.length, data: btoa(binary) };
    } finally { busy = false; }
  };
  Object.defineProperty(globalThis, '__whaleBrowserRecorder', { value: Object.freeze({ start, frame, finish, read, cancel }) });
  return true;
})()
