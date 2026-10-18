# Browser WebM recording

## Scope and upstream reference

The original `src/main/preview.js` (capture frame pump and recording IPC) and
`vendor/deepseek-harness/packages/client/ui-preview/src/client/browserRecording.ts`
(host canvas + MediaRecorder) supply the design. The recorded upstream comparison
from `90465b67ec16765631d54c195af19d9ff3c3a770` to
`e9f584be6d3b491ce777bb6823f30bc2bbc6f195` contains no change to either file.

Reference links:
- https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/preview.js
- https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/deepseek-harness/packages/client/ui-preview/src/client/browserRecording.ts

This implementation encodes a real video stream with the installed Windows
WebView2 MediaRecorder. It does not present a PNG sequence or PiP loop as video.
Only WebM is implemented, using an available VP8/VP9 encoder. MP4 and audio are
not implemented. No extra codec, external program, Electron or model is used.

## Explicit controls and ownership

- Click **录制 WebM** in the active ordinary Browser tab. Local PDF, HTML,
  audio/video previews and authenticated plugin pages are excluded.
- The UI shows encoder preparation, elapsed time/frame count, stop, cancel,
  finalization and export state. Duplicate starts are blocked.
- **停止录制** finalizes encoded bytes in memory. **导出视频…** opens a native
  save dialog and writes a new `.webm` file. Existing files are never replaced.
  Canceling the save dialog preserves the in-memory result for another attempt.
- **取消录制 / 丢弃视频**, hiding/switching the Browser panel or tab, navigation,
  replacement/closure of its Host, or owner closure cancels and drops pending
  output. These events never trigger export.
- The source is pinned by tab object, Host object and navigation epoch, with
  checks before/after native frame capture and before accepting output. The
  source URL is never evaluated as code or passed to the encoder.
- Starting recording closes PiP to avoid competing capture consumers. Other
  pixel-consuming actions are disabled while recording/awaiting export.

## Encoding and security boundary

`CaptureBrowserFrame` obtains only the original WebView's visible PNG pixels,
never the desktop, other application windows, camera or microphone. There is
one source capture/decode/consume operation in flight, with no queued backlog.

A separate hidden `IsolatedPreview` window navigates only to `about:blank` and
runs the application-embedded `helper.js`. The generic isolated-preview policy
is unchanged: private temporary profile, no application bindings/preloads,
permissions, downloads, popup navigation, file access or network resources.
It never loads site HTML, DOM, scripts or cookies. Host-side `EvalContext` is
already an asynchronous native API; it does not expose a bridge to the source.

Only base64 PNG bytes enter the helper. It uses `createImageBitmap(Blob)` in
memory, draws a fixed-size letterboxed canvas and invokes
`canvas.captureStream(0)` plus one `requestFrame()` per accepted frame.
`MediaRecorder` encodes the stream at a requested 4 Mbps. Unsupported capabilities
fail before recording is reported active, with an actionable runtime error.

Output is taken only from that MediaRecorder. Every 256 KiB transfer has an
expected offset and decoded size. The caller checks negotiated video-only
WebM MIME, byte limit, EBML header DocType `webm` and Segment signature before
making it exportable. This is container validation, not playback verification.

## Bounds and cleanup

- At most 8 source frames/second; output at most 1280 × 720.
- Input viewport at most 2048 × 2048 and 4 megapixels; PNG at most 8 MiB.
- Two-minute recording limit, enforced by the native control loop and by an
  encoder-side stop timer. If a delayed native capture finishes after the
  encoder deadline, it is not accepted and the result is finalized.
- 64 MiB encoded-byte cap, at most 256 MediaRecorder chunks. Size/error failures
  discard rather than export a misleading partial movie.
- Startup deadline 15 seconds, each Eval wait at most 10 seconds, finalization
  and transfer at most 45 seconds, total worker deadline three minutes.
- Encoded data can briefly exist in both helper and Go memory (up to 64 MiB
  each); input PNG/base64 and decoder/codec working memory are additional.
- All termination paths destroy the private helper. Existing WebView2 private
  profile cleanup waits for BrowserProcessExited. Go drops bytes on cancel.
  Interrupted export removes only the new file whose identity matches the
  file opened by that export; no pre-existing file is deleted or overwritten.

## Verification boundary

The development checks are synthetic: state/cancellation/container/export
fixtures in Go and a fake MediaRecorder protocol fixture in Node. They use no
real Browser page, user pixels, microphone, desktop capture or running Host.
No page was actually recorded and no output was played back during this work.

Windows compilation is a separate integration gate. Compilation and these
fixtures do not establish native WebView2 encoding, hidden-canvas capture,
codec availability, navigation/closure behavior or successful playback. Those
remain runtime acceptance checks on Windows, only with explicit test content
and recording authorization. No runtime-verified claim is made here.
