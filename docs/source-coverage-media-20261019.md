> Integrated Windows update at `da79d7df`: current ordinary regression and matching main/helper builds passed after the fixed contribution work. SVG/Beam/fluid/SDK implementation and dependencies are unchanged by that continuation, so prior actual SVG helper and two-device GPU receipts retain their own exact revisions and were not relabeled as new-binary acceptance. The prior SVG startup/intrinsic/retirement and GPU lease fixes remain integrated. See [exact receipts, representation differences and visible-motion/input gaps](source-depth-integration-20261019.md#current-windows-checkpoint). Older source-phase statements below remain historical; native screenshots and visible-motion equivalence are still unverified.

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

### Parent unified run02 follow-up

Parent-owned unified run02 at `221dba0` ended with exit 1. Its log is
`../evidence/source-parity-unified-20261019T161408Z/go-test-all.log`; this
delegated follow-up did not rerun it. The generated transparent/half-alpha
finite AVIF fixture reached a genuine preflight omission: the pinned libavif
1.4.2 writer emits an AV1 sequence `ccst` coding-constraint FullBox for both
color and alpha descriptions, which the Go property switch previously rejected.
The [pinned writer](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/src/write.c#L3420)
defines the exact stable box bytes as
`00000010 63637374 00000000 7c000000`; these bytes were derived from its writer,
not claimed as a dump from the prior test's in-memory sample. Read-only
inspection also found the `ccst` constant in the bundled WASM and no such box
in the older pinned 17-frame fixture, explaining why that fixture did not
expose the omission. Product correction `0ca36ee` accepts only an eight-byte
payload, version/flags zero and the 26 reserved bits zero. The six defined
constraint bits add no resource or decoding entry. Other unknown properties,
duplicate rejection and all existing decoder bounds remain unchanged.

Prepared metadata fixtures now cover both color and alpha descriptions,
the writer's valid constraints, all six defined constraint bits, truncated or
extended payloads, invalid version/flags/reserved bits, and duplicates. The
actual generated transparency/completion and finite-loop pixel/time assertions
were not weakened or removed.

Run02's two AVIF UI timeouts identified `raster.help` workers created by
`raster.hire` inside a synctest bubble. Those global workers intentionally live
for the process lifetime; this was a fixture initialization/lifetime problem,
not evidence of an animation loop. The first attempted warm-up still caused a
cross-bubble WaitGroup failure in run03 and was replaced by `c5be241`: full UI
rendering now runs entirely outside synctest. Bounded frame polling observes
the actual frame bitmap and changed canvas pixels; exact frame timing and
finite-loop completion remain deterministic decoder/advance checks. The
separate Market async-decoder fixture does not render. The production pool,
processor settings and renderer paths were not changed. Canvas, green WebP
frame, hide and Escape/dismiss assertions remain intact. Composer and Market
passed at `c5be241` in `../evidence/source-parity-recheck-20261019T164454Z/`;
Files, including its image sequence fixtures, passed at `92d0327` in
`../evidence/source-parity-targeted-20261019T165904Z/`. These are software
render/regression observations, not original/Go Windows visual acceptance.

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

## Continued SET-06 / fluid / Beam / SVG source pass

The continued 2026-10-19 request explicitly resumed the fluid/Beam/SVG work.
The earlier “paused” and static-only descriptions above are historical and are
superseded by this section for the changed paths. The source baseline remains
original `e353e8e`; this pass started on native `ffe7ab6`. Other owners' local
commits were preserved. This section records implementation, **not** compiled
shader, automated regression, helper containment runtime or native visual
acceptance. No test/fixture work, shader compilation, build or app launch was
performed during this source phase.

Because the original checkout contains unrelated dirty files, the exact Beam,
typing, submission-settings, effects-modal, cursor-fluid/cursor-fx, theme-token,
locale and cited image-element source paths were explicitly compared with
`git diff e353e8e --name-only -- <exact paths>`; it returned no differing path.
The source decisions in this section do not adopt an uncommitted original-tree
behavior as the fixed baseline.

### SET-06 actual settings and typing call chains

Original `ui-conversation/src/submission-settings.ts:267-348,485-496` normalizes
the active style at renderer adoption, independently of JSON-import validation.
`ComposerBeam.tsx:92-167` maps percentages to three independently rounded layer
opacities and samples local night time on mount/activation and a 60-second
interval. `TypingFxLayer.tsx:96-201` and `input/editor/typing-fx.ts:17-122` own
the 16 UTF-16-unit insertion/24-echo limits, IME-start snapshot and commit diff,
history/paste/programmatic exclusions, collapsed selection and listener
retirement. Native `beam.go:readBeamStyle` and
`typing_effects.go:readTypingStyle` now also clamp/adopt enums, palette/color
rules and numeric ranges at render adoption. Existing typing event exclusions,
IME commit handling, echo count and reduced-motion gates remain; echo scaling
now preserves the source's unscaled right-hand anchor and centered transform,
and its host-space spawn position no longer moves when the input scrolls.
Underline caret corners are square as in the source.

Original `settings/BeamSettingsModal.tsx:270-388,424-545` and
`TypingFxModal.tsx:139-187,190-284,364-443` were read through actual callbacks:
Save/Cancel use isolated style/preset drafts; JSON has core/version/byte/type
validation; at most five presets have 32 UTF-16-unit names; selecting a mode
changes only its specified fields; other Beam edits mark mode custom; typing's
custom color chip remembers an independent triplet. Native
`internal/ui/settings/effect_preferences.go` retains save acknowledgement/error
ownership and now adds the missing custom-mode marking, palette-derived seed,
typing custom-triplet memory, native-caret blink disablement, preset-name
truncation/name clearing and reset selection clearing. The existing source
field ranges, presets, JSON contract and Save/Cancel scope were present.

The source modal's typing preview actually types `输入特效` / `Typing effects`
every 150ms; the old native preview was an empty editable input.
`composer/settings_preview.go:36` now paints the automatic sample with the
shared keyframes, theme/preset/custom text/echo/caret colors, blink and reduced
motion. Beam preview uses the source status text and shared Beam painter.
Actual native layout, font rasterization and settings interaction remain for
the coordinated acceptance phase.

### Beam: independent source layers rather than widened path strokes

Original `ComposerBeam.module.css:17-254` has a frame at -4px/26px radius,
children at +4px/22px radius, fixed-position radial inner washes and inset
shadows, two additive conic inner masks, a separately masked track with a
white travelling highlight, and an independently colored 1.5px bloom ring.
Its remaining rules define spin direction, hue/breathing combinations, easing,
420ms opacity transitions and reduced-motion hiding. The original tokens are
`ui-theme/src/styles/base.css:38-39` (22/26, not 28).

`composer/beam_raster.go:48,145,214` replaces the old 160-segment broad-stroke
approximation with those independent premultiplied layers, separate masks,
palette closure/stops, fixed radial geometry, track inset, highlighted edge and
blurred bloom. It rebuilds geometry on actual size/DPI/track/palette changes;
`ui.DecorativeRaster` retains one texture rather than allocating a new scene
bitmap every frame. Zero blur still paints the independent bloom ring.
`effect_timing.go` retains independent opacity transition reversal, spin/hue/
breathing clocks, minute sampling and cancellation behavior. Main-card paint
order/radius is a root-owned integration hunk, not overwritten by this owner.

The blur uses three running-sum box passes approximating a Gaussian; inset
shadow falloff uses the Gaussian edge integral. Corner convolution, coverage,
gradient quantization and native glyph scaling are not claimed pixel-identical
to Chromium. This is an actual three-layer implementation, not a changed stroke
width or an assertion of browser visual equivalence.

### Windows fluid: actual persistent GPU texture passes

Original `ui-theme/src/cursor-fluid.ts:17-43,302-545,687-779,781-1024` was read
through shader programs, texture creation/format probing, all numerical passes,
resize, input, live retuning, visibility, idle and dispose. It uses short-edge
128 velocity and 1440 dye half-float targets scaled by aspect; R16F/RG16F may
expand to RGBA16F, never a lower-resolution dye grid. Curl/confinement,
reflecting-edge divergence, 0.1 pressure decay, 20 Jacobi steps, gradient
subtraction, velocity and dye advection precede shaded premultiplied display.
Motion keeps the most recent event delta; each press splats immediately before
the following animation step. Four seconds after input it parks, retaining its
last frame; hidden documents stop rAF; dispose releases the context.

`third_party/mygo/internal/gpu/d3d11/fluid_windows.go:32,105,187,248,262,288`
and `fluid.hlsl` implement those actual half-float ping-pong GPU passes on the
SDK's existing pure-Go D3D11 device/COM/compiler infrastructure. They create no
window or swap chain. Texture-format probing follows the source's channel
fallback order; pressure/curl/divergence restart on drawing-buffer changes,
while velocity/dye are linearly resampled. Only the final drawing-buffer-sized
RGBA display is read back to the existing native bitmap bridge. It does not
run the CPU solver at larger constants or claim CPU work is GPU work.

`ui/decorative_fluid_windows.go` selects that GPU backend; the worker keeps one
in-flight job/result and owns all device state, texture mutation and disposal.
Press splats now retain FIFO events and update the source pointer coordinate
baseline before subsequent motion. Speed/size/palette remain live updates;
hidden/4-second idle scheduling and effect/reduced-motion disposal remain in
the existing owner. GPU initialization/shader/texture/map/device-removal
failures return an error and retire the worker, exposing no fake CPU success
or repeated recreation attempts. Windows pacing is one 60Hz job opportunity,
without the older CPU 24%-core cap.

Non-Windows deliberately retains its existing 64/64 float32 CPU profile and
24%-core pacing in `decorative_fluid_other.go`; cross-platform GPU equivalence
is **not** implemented. Windows has bounded texture extents and a 256MiB
retained-resource gate, and requires hardware D3D11 rather than claiming WARP
as hardware. Those resource/renderer boundaries and final-frame readback latency
remain differences from the original WebGL presentation. Hardware shader
compilation, visible motion/performance, occlusion and release are unverified.

### SVG: preserve the actual image context, including inline animation

Original Files `FilePreview.tsx:689-692`, composer
`ComposerAttachments.tsx:83,100` / `ImageLightbox.tsx:69`, and README
`MarkdownText.tsx` render image elements; they do not grant a navigated SVG
document's script/network authority. The [SVG processing-mode specification](https://www.w3.org/TR/SVG2/conform.html#secure-animated-mode) defines this as
secure animated image use: declarative SMIL/CSS animations are allowed, including
foreignObject content under equivalent restrictions; scripts, interaction and
external resource loads are disabled. Data resources are distinct from external
references. This is why supporting original SVG does not require URL fetching,
running author scripts or inventing an SMIL interpreter.

`imagepreview/svg_image.go:22,36,76,113,121` adds a separate bounded image-only
entry. It keeps the SVG solely in a base64 data-URL `img`, with an additional
script-src-none CSP for the prepared image document. Native resvg still handles
its existing static subset; SMIL/CSS/foreign content/unsupported static content
and embedded GIF/animated WebP resources select the existing isolated inline
helper. The public OpenUI `ResourceDataURL` gate is unchanged. Native static
text's absent initial family/size now matches browser serif/16px rather than
forcing the bundled Go sans font, while explicit authors' choices remain.

`internal/ui/svgimage/seat.go:17,52,177,227` binds each image to immutable bytes,
its own generation and visible paint lease, plus the root's window/Host/session
`Owner` context and thread-safe validity callback. Helper startup is asynchronous;
owner/seat revocation cancels it during startup and closes late results. The
existing `previewprocess.OpenExperimental(Inline:true)` retains its original OS
containment and one-document IPC. New `inert_image_surface*.go` reuses its
single-in-flight pixel stream and verified Windows child, without new focus or
input handlers; disabling image-child input keeps native click/zoom/pan control.
Windows image-only regions also intersect the trusted native rounded seat and
subtract the visible remove/retry button shapes, so the child no longer covers
those native overlays. That geometry is generated by the parent, never SVG or
the helper protocol; pixel-stream platforms keep ordinary canvas overlay order.
No per-frame browser-capture loop or new browser shell was introduced.

Actual inline callers are Files `media.go` / `image_view.go` (including saved
zoom/pan), composer `attachment_display.go` thumbnail and original seats, and
Market `readme_media.go` / `readme_images.go` / gallery's shared image decoder.
They preserve existing GIF/WebP/AVIF/static SVG paths. Removing/dismissing/replacing
an image closes its own surface; inactive local seats expire after 400ms, while
root owner cancellation covers Host/session/window changes, hiding/minimization
and close. Surface errors remain explicit image errors, never a static image
mislabelled as live animation. The root owns model/window-provider and separate
`htmldocument` SVG entry wiring.

Remaining bounds are the existing 1MiB XML, 40,000 tokens, 10,000 nodes, depth64,
64KiB attribute and bounded dimension/geometry policy. The previous browser-seat
px/viewBox sizing restriction is closed by the intrinsic-size continuation below.
Declarative frame rendering on a real helper,
containment/readiness, theme/transparent background, native child clipping,
parent input delivery, thumbnail/lightbox transitions and process release all
remain unverified until the coordinated runtime phase. Source wiring alone is
not evidence those native operations passed.

### DESK03: opt-in native caption and transparent frame geometry

Fixed source evidence is `e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`:
`src/renderer/window-controls.css` and `window-controls.js`,
`src/main/harness-chrome-inject.js`, `src/main/native-window-motion.js`,
`src/main/chrome.js`, and ui-theme `styles/design-platform.css`. The referenced
paths were also compared with the original worktree and have no dirty diff.
The controls are a fixed 48px overlay, not 48px added to document flow: their
buttons are 32x32 at top/right8, radius8, with actual rounded 12-unit SVG glyphs.
The main body retains its existing title strip instead of shifting down36px.

`mygo.WindowOptions.WindowsCaptionButtons` is copied into platform options;
nil keeps the existing SDK controls unchanged. For the app's explicit
32/32/8/8/r8/g12 option, `titlebar.go` uses the same dimensions for positioning,
paint and native hit testing; reports right104 independently of title-strip
height; preserves `HTMAXBUTTON`/`DefWindowProc` Snap handling; uses the original
pointerdown action and pointer cursor. `caption_rounded.go` draws the actual
min/max/restore/close vector geometry, caches its coverage, and uses source
light/dark label/hover tokens and close `#e81123`. Source inject reserves an
additional8px cluster gap (112px total); the app's existing right padding owns
that content spacing. Its anti-aliasing uses a4x4 coverage grid, not a claim of
identical Chromium rasterization. A one-alpha native hit plate keeps each
transparent rectangular button target operable.

`Transparent:true` plus native `Content` plus
`WindowsFrameCornerRadius:20` explicitly selects existing DirectComposition
alpha presentation; zero leaves SDK window defaults unchanged. CPU bitmap and
GPU swap-chain visuals share a circular `IDCompositionRectangleClip` through
`internal/gpu/d3d11/roundclip`; DPI resize updates the physical radius, and
maximize/fullscreen remove it. Clip, visual and device lifetimes stay on their
existing render thread, with clip release before visual/device release. COM
errors are returned into the real presentation path. No new shader, browser,
external dependency, download or retry mechanism is added.

The native window keeps caption/thick-frame styles and native state animation,
disables DWM non-client paint and the Windows11 OS corner mask, and restores
that policy after an enabled-DWM notification using one coalesced posted
message. App-only frame calculation removes the side/bottom native insets;
the native surface forwards the actual resize-edge hit codes to its parent.
The SDK's existing one-physical-pixel Windows11 top inset remains for native
Snap behavior. The original one-physical-pixel inside frame ring is app/root
paint wiring: BorderL2 is black.10 in light appearance, white.12 in dark, and
the ring is hidden when maximized/fullscreen.

COM slots are derived from the installed Windows SDK10.0.26100 dcomp.h and
MSVC overload ordering: device CreateRectangleClip24, Visual.SetClip(Clip*)13,
RectangleClip float properties4,6,...,26. AMD64 uses the existing SyscallN XMM
argument mirroring. ARM64 has a narrow static assembly thunk that transfers
float bits to S0 then tail-calls the COM method through the normal SyscallN
foreign-call entry; it adds no executable allocation or cgo. Source formatting
and whitespace checks are the only checks in this phase. Compilation on both
architectures, COM ABI execution, GPU availability, visual alpha/DPI/resize,
Snap/animation, pointer interaction, and app ring/layout remain unverified
until the coordinated validation phase; no hardware result is inferred here.

### SVG intrinsic sizing continuation

Fixed original conversation `service.ts`'s `probeDimensions` constructs an
`Image`, waits for onload and records `naturalWidth`/`naturalHeight` from its
attachment blob preview URL; it does not parse a viewBox as pixel dimensions.
This source path is unchanged from e353e8e in the original worktree.

`svg_intrinsic.go` separates browser image admission from the existing static
SVG gate. It converts absolute px/in/cm/mm/Q/pt/pc lengths at CSS96dpi for an
initial bounded layout hint, accepts percentages/auto/missing dimensions without
requiring a viewBox, and leaves font-relative/CSS sizing to the browser. A
300x150 default-object box is only provisional; a viewBox is never falsely
reported as the final natural pixel size. SVGs requiring that browser intrinsic
resolution, including root CSS geometry, use the existing live image seat.
The static native SVG parser and WASM capability/resource boundary are unchanged.

The same ongoing inert helper now reports bounded `imageWidth`/`imageHeight`
on its nonce-checked loaded packet. Its existing trusted child bootstrap waits
for the sole img to decode, reads natural dimensions, and passes only numbers
through the opaque iframe's existing readiness message. The parent reads outer
`__previewState` only; it never accesses the opaque child DOM. For positive
subpixel SVG dimensions truncated to zero by the naturalWidth/Height integer
API, the hidden startup temporarily sets that same img to auto sizing, measures
its browser-used rectangle and restores its style before presentation. This
does not create another image/helper, change SVG bytes, guess font metrics, or
add a frame/capture loop. Finite positive dimensions remain bounded to32768 per
axis and32MP. Ordinary HTML/OpenUI/capture sessions cannot request this mode.

Windows and Linux existing Go helpers return that metadata after their current
readiness path. The macOS existing single-process legacy WebView helper returns
the same two numbers from the state it already reads; no new evaluator or
script grant was introduced. Its source branch still requires a separately
signed real helper and has not been compiled or run on macOS here.

Seat generation/owner checks retain the original byte snapshot, cancel stale
results and update mutex-protected natural size only for the current helper.
Its existing invalidation updates Files zoom/pan and displayed dimensions,
composer lightbox sizing, and Market README/gallery image ratios. Standalone
`BuildSVGImage` documents also pass the browser dimension gate and retain the
ordinary browser img intrinsic behavior. No source dimension-unit omission
remains in this browser route. Actual physical/font-relative/default-size
rendering, helper handshake, font availability, owner transitions and the three
platform renderers remain unverified until the coordinated validation phase.

The source-only root-integration review also found and closed the same size
refresh in Market's generic `plugin_artwork.go` consumer: it now copies the
current seat dimensions after Prepare before computing an auto-height aspect
ratio. Plugin-manager `packagePreview` uses this path at width640/height:auto;
it must not keep a2:1 placeholder after the browser resolves another ratio.
This is a source correction, not a claim of rendered-image validation.
