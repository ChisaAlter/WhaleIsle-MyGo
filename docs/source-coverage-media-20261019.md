# Media source coverage, 2026-10-19

Scope: original desktop image behavior in Files, composer attachments/lightbox,
and Market README. Original source was read only at
`C:/AI/Deepseek-Harness-Desktop` (`e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`).
Native changes belong to `codex/session-runtime-compat-20261019`, inspected from
`363c6504be35987d1c628bc2ec96df60a9206571`. This is a source implementation
record, not runtime/visual acceptance or a claim that all original modules were
reviewed. No tests were executed; no build, app launch, service, paid/model/account/device action,
push, CI or release was performed by this delegated pass.

## Actual source comparison

Original paths below are relative to
`vendor/deepseek-harness/packages/client/` unless prefixed `src/`.

| Surface/contract | Original source evidence | Current Go source and status |
| --- | --- | --- |
| Files formats and complete image bytes | `ui-sidebar-documentpreview/src/client/image/ImageBody.tsx:14` registers PNG/JPEG/GIF/WebP/BMP/ICO/SVG; `:76` creates an image Blob; `:121` mounts browser `<img>` inside the shared zoom viewport | `internal/ui/files/media.go:isPreviewImage`, `loadImagePreviewContext`, `paintImagePreview`; root-scoped bounded reads, bitmap/SVG/ICO/AVIF decode and native zoom/pan remain. GIF timeline already existed before this audit. Animated WebP routing is added in this batch. |
| Composer thumbnail and original image | `ui-attachment/src/client/ComposerAttachments.tsx:83` mounts `<img>`; `:100` mounts `ImageLightbox`; `ui-primitives/src/ImageLightbox.tsx:69` mounts original `<img>` | `internal/ui/composer/attachment_display.go:readAttachmentDisplay`, `paintAttachmentRail`, `paintAttachmentOriginal`; `attachment_image.go` already animated GIF in both thumbnail and original seats. Animated WebP now uses exactly that timeline/bitmap path. |
| Market README image semantics | `ui-settings-market/src/client/MarketExtras.tsx:104` mounts `MarkdownText`; `ui-primitives/src/markdown/render.tsx:imageSource` allows explicit HTTP(S), retaining authored alt for unsupported sources; `MarkdownText.tsx:181` documents direct absolute HTTP(S) images | `internal/ui/market/readme_images.go`, `readme_media.go` keep existing HTTP(S) cache, natural size/aspect ratio, max width and authored-alt fallback. Before this batch, `ui.DecodeBitmap` meant first-frame GIF and failure for animated WebP. Both now use the shared native animation decoder. No new URL scheme, guessed base URL or resource access is admitted. |
| Static SVG | Original Files/README use browser image elements, including browser-supported static SVG content | Files/composer already use `internal/desktop/imagepreview/svg_native.go:DecodeSVGContext` and `svg_native_bounds.go`. Market previously used `ui.ParseSVG` plus a gate that rejected even text/filter/image documents. Market now retains that existing narrow fast vector path and uses the same bounded isolated rasterizer for additional accepted static SVG. It is not full SVG support. |
| Separate desktop file window | `src/renderer/file-preview.js:render` mounts `<img>` for image and media elements for audio/video | Read to identify the original image contract only. This pass did not audit every separate-window, audio/video/PDF/HTML operation. Their implementation/acceptance is **not reviewed here**, rather than inferred unimplemented. |

## Implemented delta

`internal/desktop/imagepreview/webp_animation.go` parses the existing 16 MiB
bounded RIFF input. Leading VP8X canvas dimensions, ANIM/ANMF ordering, frame
rectangles, actual compressed bitstream dimensions, at most 256 frames and at
most 32 million retained composited canvas pixels are checked before pixel
decoding. ANMF VP8/VP8L and optional ALPH chunks are rewrapped as still WebP and
decoded by the already-pinned `golang.org/x/image/webp`; no dependency changed.
Unknown ancillary RIFF chunks stay inert. Composition supports replace/over and
dispose/keep, WebP total-cycle loop counts, and the existing timeline's 100 ms
handling for delays at most 10 ms.

The [WebP container specification](https://developers.google.com/speed/webp/docs/riff_container)
defines frame rectangles, blending, disposal and total-cycle loop counts.
For original-browser behavior, [Chromium 152's decoder](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/third_party/blink/renderer/platform/image-decoders/webp/webp_image_decoder.cc)
uses transparent initial/disposed pixels and ignores animated ICC profiles. The
native timeline likewise clears transparently instead of applying ANIM's
optional background-color hint. This source comparison is not pixel equivalence
evidence for the Go VP8 codec versus libwebp.

Files and composer retain their existing mounted-seat ownership and paint-driven
scheduling: hidden or closed surfaces schedule no recurring animation work;
late paints skip elapsed frames; finite loops stop. Market background decoding
uses two decode slots, a one-result channel and one invalidation. The mounted
image seat retains its own bitmaps/timing; a dismissed seat retains no animation
timer. The shared `readmeDecodedImage.load/poll/advance` methods also provide the
separate gallery implementation with the same decoder contract.

Market additionally recognizes bounded ICO and still AVIF from cached bytes,
using the existing shared decoders. Cache paths are hashes without extensions,
so this does not guess a filename or change network/cache policy.

## Support and remaining limits

| Capability | Files | Composer thumbnail/original | Market README |
| --- | --- | --- | --- |
| PNG/JPEG/BMP/static WebP | Existing native bitmap | Existing native bitmap | Existing bitmap, now dimension-gated before allocation |
| GIF delay/disposal/finite or infinite loop | Existing composited timeline | Existing composited timeline | Added shared timeline |
| Animated WebP VP8/VP8L, ALPH, offsets, blend/dispose, loops | Added shared timeline | Added shared timeline | Added shared timeline |
| ICO / bounded still AVIF | Existing shared decoder | Existing shared decoder | Added shared decoder |
| SVG shapes, text, gradients, local references, supported static filters/patterns/markers and bounded embedded raster | Existing isolated rasterizer | Existing isolated rasterizer | Added same rasterizer for documents beyond the existing vector subset |
| SVG scripts, SMIL/CSS animation, foreignObject, external images/use/paint, nested embedded SVG, unsupported stylesheet constructs | Rejected by existing bounded native gate | Same | Rejected/retains authored alt; no external-resource exception added |
| Animated AVIF | Explicitly unsupported by existing shared AVIF gate | Same | Same |

SVG is still a bounded static subset. Its validator rejects CSS at-rules,
variables and unsupported resource functions; it bounds XML structure, geometry
and references. Acceptance of an SVG is not a full conformance claim. Market
keeps the old vector parser for its existing supported subset (including the
unconditional switch/metadata cases); broader static artwork uses the isolated
rasterizer. Animated WebP is implemented within existing resource limits;
exact browser codec color/rounding, timing under actual window occlusion, and
native visible playback remain unverified in this pass.

GPU fluid/Beam depth stays paused. Source still explicitly describes native
fluid short-edge resolution 64 in
`third_party/mygo/ui/decorative_fluid_worker.go`, versus the source's deeper
GPU profile. `internal/ui/composer/beam.go` remains native paint. This pass did
not change those renderers or reclassify their recorded visual differences as
environment blockers.

## Verification handoff

Implementation and prepared regression files were formatted with gofmt only. Parent owns unified compilation,
targeted decode/surface checks, and actual native verification after parallel
source implementations finish. Relevant boundaries include malformed RIFF,
frame/byte/canvas budgets, alpha/blend/dispose rectangles, finite/infinite loops,
hidden/closed seats, Market async readiness and broader SVG visible content.
After the source commit, the parent requested focused regression preparation.
`webp_animation_test.go` and the 260-byte `testdata/animated-native.webp` cover
actual composited pixels, alpha, offsets, replace/over, disposal, loops/duration,
container/frame/canvas rejection, still VP8/VP8L and lossy ALPH.
`image_webp_test.go`, `attachment_webp_test.go`, `readme_media_test.go` cover
native surface playback, pixel rendering, staging cleanup, finite stopping,
inactive/dismissed/hidden seats and async one-result notification. Static SVG
tests now assert visible text/filter/embedded-image/pattern/marker/CSS content;
the older negative table retains malformed embedded, SMIL, foreignObject,
external-resource and unsupported stylesheet failures. These tests are prepared
but **not run** here. They do not replace actual Windows playback/occlusion QA.

The older `docs/native-workspace-parity-20261019.md` statement “GIF remains first
frame” and its broad “full SVG unsupported” boundary describe an earlier source
state. GIF was already implemented at this audit's starting revision, and
static SVG breadth already existed in Files/composer. The specific unimplemented
gaps found here were animated WebP and Market's narrower media integration;
this batch supplies their source implementation. Other modules remain under
their own audits and acceptance records.
