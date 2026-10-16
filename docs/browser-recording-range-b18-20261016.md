# B18: restore original browser recording range

## Original source contract

- `ui-preview/src/client/browserRecording.ts`: first valid frame fixes the canvas dimensions; later frames are fitted and letterboxed, canvas stream requests 12 fps, encoder target is 4,000,000 bits/second. Preferred formats are WebM, WebM/VP9, then MP4. There is no duration cutoff.
- `src/main/preview-pip-protocol.js`: capture interval is `ceil(1000/12)` milliseconds (84 ms), with one capture in flight.
- `src/main/preview.js:1009–1035`: saved encoded output must not exceed 512 MiB.
- `ui-preview/src/client/viewport.ts:42–43`: viewport max dimension is 3840 and max area is 3840 × 2160.

## Native implementation

Removed the artificial 2-minute capture deadline, 3-minute UI worker deadline, 45-second whole-artifact transfer deadline, 256-chunk limit, 720p downscale, and 64-MiB encoded artifact restriction. Capture stays serial on an 84-ms scheduler, drops queued timer ticks, and releases each PNG after consumption. The isolated canvas uses a 12-fps stream and the original 4-Mbps encoder target. No measured throughput or sustained frame-rate claim is made.

The original 512-MiB encoded-output limit is enforced during data collection and again before transfer/export. Chunk metadata is consolidated every 64 chunks; cancellation/error clears blobs, stops tracks, and shrinks the canvas. Encoded data is retained for explicit export retry only; source changes cancel the result. Export remains exclusive-create and does not overwrite existing files.

Format selection now includes MP4 fallback. Actual negotiated MIME is passed through, WebM and ISO BMFF container headers are checked, and export filters/default filename/extension reflect the actual container. No codec or playback success is inferred from platform identity; missing runtime encoders fail visibly.

The recording-only `CaptureBrowserRecordingFrame` accepts the original 4K logical viewport range and checks actual PNG dimensions against native bounds/display scale. The old shared capture entry point retains its previous limits. The PNG byte bound is 64 MiB, matching the existing Windows native PNG transport bound; decoded pixels have a separate bounded budget, and no page-provided dimensions are trusted. Native HiDPI pixels are retained; runtime codec-specific dimension restrictions can still reject unsupported captures.

## Verification scope

No tests/builds, native GUI, actual browser/camera/microphone capture, or codec execution were performed by the implementation worker. Go files were formatted. `internal/desktop/browserrecording/helper_test.cjs` is an injected synthetic protocol fixture for the coordinating agent: >256 encoded chunks, dimensions including odd pixels and 4K, MP4 fallback, limit enforcement above the old 64-MiB cap, transfer, backpressure, cancellation, and missing codec. Go fixtures additionally describe container validation, export extensions, and source cancellation. These fixtures do not establish playback or performance.
