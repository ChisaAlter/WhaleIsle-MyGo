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
| Desktop Files formats and complete image bytes | `ui-files/src/client/desktop-files.ts:63-74,123-133` takes supported session-relative workspace files before the generic document preview; `workspacePreview.ts:4-13` includes `.avif`; `FilePreview.tsx:241,324-342,689-692` reads image media and mounts `data:${media.mime};base64,...` in `<img>`. Original `src/main/workspace-fs.js:15-24,257-293` authorizes local bytes and assigns AVIF MIME. The separate fallback `ui-sidebar-documentpreview` format list does not govern this desktop route. | `internal/ui/files/media.go:isPreviewImage`, `loadImagePreviewContext`, `paintImagePreview`; root-scoped bounded reads, bitmap/SVG/ICO/AVIF decode and native zoom/pan remain. GIF timeline already existed before this audit. Animated WebP routing was added earlier; `3d61894` adds the existing AVIF sequence decoder to the actual desktop `.avif` route. |
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
| Animated AVIF | Added existing bounded AV1 sequence timeline for `.avif`; `.avifs` remains unregistered | Added bounded AV1 sequence timeline | Added same sequence timeline |

SVG is still a bounded static subset. Its validator rejects CSS at-rules,
variables and unsupported resource functions; it bounds XML structure, geometry
and references. Acceptance of an SVG is not a full conformance claim. Market
keeps the old vector parser for its existing supported subset (including the
unconditional switch/metadata cases); broader static artwork uses the isolated
rasterizer. Animated WebP is implemented within existing resource limits;
exact browser codec color/rounding, timing under actual window occlusion, and
native visible playback remain unverified in this pass.

### AVIF sequence follow-up

The initial audit classified AVIF animation as unsupported. A follow-up source
inspection found the existing bundled ABI already implements `decode_all` and
writes every frame plus float64 seconds. No new codec dependency or WASM rebuild
was necessary. The still decoder remains unchanged. The first sequence wiring
was restricted to composer/Market based on an incorrect reading of the generic
`ui-sidebar-documentpreview/src/client/document/unviewable.ts:27` list. The
actual original desktop Files extension takes the supported workspace image
route first: `ui-files/src/client/workspacePreview.ts:4-13` includes `.avif`,
and `FilePreview.tsx` receives AVIF MIME/bytes from the desktop workspace reader
and mounts an image element. That fallback exclusion therefore cannot be used
to exclude desktop Files AVIF sequences. Source correction `3d61894` routes
`.avif` sequence bytes through the existing `DecodeAVIFAnimationContext` and
the same Files GIF/WebP frame/bitmap/timing path. Static `.avif` keeps its
existing decoder; `.avifs` remains unregistered and mislabeled `.gif` does not
gain an AVIF entry. No codec, dependency, resource permission or configuration
was added by this route correction.

`avif_animation_bounds.go` validates bounded BMFF track/sample tables, dimensions,
byte extents inside mdat, self-contained data references, per-frame timing and
edit lists. `avif_animation.go` uses the existing isolated module's `decode_all`
argument and copies full-canvas 8/10/12-bit frames into the current Animation
contract. Its frame-duration output is initialized to NaN; each duration is
written by the bundled C wrapper only after the corresponding complete pixel
copy, so missing/corrupt later frames fail rather than exposing an unfinished
buffer. Frame/canvas/16 MiB/10-second/512 MiB limits remain. Matching alpha tracks
are accepted for sequences, including fully transparent valid frames; the still
item/alpha policy is unchanged.

The [bundled upstream DecodeAll implementation](https://github.com/gen2brain/avif/blob/v0.6.0/avif_wazero.go#L118)
shows the existing all-frame and duration ABI. Its public LoopCount field is not
populated during decode, so Go reads the same track-duration/edit-list ratio as
[libavif 1.4.2](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/src/read.c#L3566).
The [original-version Chromium decoder](https://github.com/chromium/chromium/blob/152.0.7977.78/third_party/blink/renderer/platform/image-decoders/avif/avif_image_decoder.cc#L403)
maps absent repetition metadata to infinite looping; the native timeline now
uses that behavior, preserving explicit finite and infinite edit-list values.
Explicit `avif` major-brand still selection remains separate from `avis` track
selection. This is source evidence for the lockfile's browser version, not
acceptance evidence from the user's running original installation.

The sequence gate currently supports one color track and an optional matching
alpha track. Extra color tracks, fragmented movie layouts, clean-aperture crop,
layer selection, ICC profiles and unsupported sample descriptions fail explicitly.
Exact libavif-versus-Chromium/crabbyavif pixels, HDR/color management and actual
native visible AVIF playback remain unverified. GPU fluid/Beam remains paused.

After AVIF source commit `ce67a12`, focused regressions were prepared, not run:
`avif_animation_test.go` uses the pinned 17-frame sequence to check full pixel
output, frame variation, 80 ms timing, unknown/infinite loops, frame ownership,
metadata limits, external sample references, corrupt final-frame completion and
cancellation. Edit-list fixtures cover explicit once/finite/infinite and rounded
track durations. A tiny transparent/half-alpha finite sequence is constructed
inside the test using the already-bundled encoder ABI, so it requires no added
dependency, pre-generated binary or external tool. Its execution is still pending.
`attachment_avif_test.go` and `readme_avif_test.go` prepare native surface paint,
advance, size, dismiss/hide and staging-release checks. The earlier Files
negative `.avif` assertion was incorrect for the actual desktop route and has
been replaced by a prepared positive `.avif` playback fixture. It uses the
existing pinned 17-frame sequence and actual `paintImagePreview` seat to check
canvas dimensions, bitmap/staging ownership, 80 ms timing, visible frame
variation, hidden-seat inactivity and inactive-tab release, plus corrupt later
frame and cancellation rejection. A separate boundary fixture retains the
unregistered `.avifs` and mislabeled `.gif` rejection. Existing GIF/WebP and
static AVIF regressions remain in place. All of these new/updated fixtures
remain unexecuted; no native playback acceptance is claimed.

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
