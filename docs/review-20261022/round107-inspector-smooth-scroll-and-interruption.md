# Round107 — Inspector to Chat smooth scrolling and interruption

The native Inspector now scrolls an offscreen Chat target smoothly with Chromium152's desktop programmatic curve, then confirms the completed position before queuing the existing700ms overlay. Positive clipped target visibility still flashes in place without scrolling; reduced motion centers immediately without a flash.

This closes the immediate-center/global-interruption gap recorded in Round106 for the current native transcript's one scrolling ancestor. Whole-product parity remains incomplete.

## Implementation

- `third_party/mygo/ui/list_motion.go`: measured `ListState.ScrollToAnimated`, application-supplied duration/easing, stable-key target tracking, exact final row alignment and `ScrollOffset`. Target/header measurement happens without a first-frame jump. Ordinary requests, external offsets and `StopScrollAnimation` replace motion. Stopping adds no frozen-position request, so the interrupting wheel/key still performs its normal operation.
- `third_party/mygo/ui/input_observer.go`: passive window input observation before inspection, control dispatch and shortcut handling; registrations belong to the final build pass and are cleared on the next build or close. Window-DIP pointer coordinates, zero/precise/Ctrl wheel and key modifiers/repeat/shortcut-blocking fields are preserved.
- `internal/ui/chat/session_inspector_smooth.go`: cubic-bezier(.4,0,0,1), with duration `min(sqrt(abs(physical delta))/60,1.5)`seconds. Duration receives DIP distance multiplied by the painted display scale. While queued, moving or confirming, any wheel, supported pointerdown or navigation key interrupts, including events outside Chat. Ordinary A/Left/Right/Enter continue. Capture ends after confirmation, before queued flash; the active fixed overlay survives generic input. No-op movement uses the150ms quiet fallback.
- `internal/ui/frame/conversation_width.go`: width rails use the same `ScrollState` as Chat. Previously they rebound the List to another tracker and permanently changed configured `FollowEnd` on a rail wheel. The shared tracker now preserves regular scrolling and following behavior.
- Toolkit documentation describes the new APIs. No platform/native callback implementation or cgo was added.

Independent read-only review found six List layout/request defects plus a later replacement stage. All were repaired and reproduced by focused regressions: final centering through a parent's second size-transition layout; post-request height measurement translating the animation origin; target/header builder replacement; explicit offset adoption after remount; duration initialized from the displayed first-frame geometry; provisional no-op completion changing into movement in another layout; and an intermediate/overscan builder cancelling or replacing a request after the animated offset was calculated. The last case now re-resolves the request before committing the offset, with a bounded layout restart.

## Real reference evidence

The installed unmodified Inspector factory ran in the independently owned Electron44.3.0/Chrome152.0.7977.78 page on3083/debug19444. This uses synthetic loaded Chat boxes and window-dispatched reveal/interruption events; native DOM scroll/scrollend and real WAAPI run without clock overrides. It does not invoke the production Host/model. The recording passed12cases/106checks. Exact raw times, options, pending/overlay states, origin-alignment method and ownership are in:

- [Reference report](evidence/inspector-reference-pass251-smooth-report.md)
- [Raw browser recording](evidence/inspector-reference-pass251-smooth-state.json)
- [Browser trajectory plot](evidence/inspector-reference-pass251-smooth-trajectories.svg)
- [Interruption plot](evidence/inspector-reference-pass251-smooth-interruptions.svg)
- [Reference lifecycle](evidence/inspector-reference-pass251-smooth-lifecycle.json)

Captured native/CSS viewport ratio is1.5. Browser DPR emulation alone does not change the native compositor scale. The installed bundle SHA256 remained62443553c0dec20f1d1a21b02a55822877c734059ea1c18d5a414ddafac657a0.

Chromium sources pin the curve and duration rather than fitting a duration to observations:

- [Desktop curve defaults](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/cc/base/features.cc)
- [Delta-based duration and curve evaluation](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/cc/animation/scroll_offset_animation_curve.cc)
- [Programmatic duration behavior](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/cc/animation/scroll_offset_animation_curve_factory.cc)
- [Programmatic animator](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/third_party/blink/renderer/core/scroll/programmatic_scroll_animator.cc)

## Native/browser trajectory comparison

The native test replays recorded browser frame times through the production List and Inspector revealer with a fixed clock, using the source recording's bounded first-offset phase alignment. Duration is never fitted. This measures offsets on equal constant-height geometry; it does not establish equal OS scheduling, rasterization, variable-height production content or whole-window parity.

| Offset/DIP at scale1.5 | Source duration/ms | Native vs browser RMSE/DIP | Maximum error/DIP |
|---|---:|---:|---:|
|200→500|353.553|0.193|0.380|
|200→800|500.000|0.235|0.587|
|100→1400|735.980|0.235|0.973|
|1400→20|758.288|0.225|0.845|

[Per-frame native/browser comparison](evidence/inspector-pass251-smooth-comparison.json) retains all samples. [Native scale1/1.5/2 samples](evidence/inspector-native-pass251-smooth-state.json) also verify physical-distance duration without changing DIP target positions.

## Real native window

An opt-in fixture runs the real native Inspector presentation and Chat with30recorded assistant messages, an80DIP composer seat, and three Inspector records. It has no backend/model connection. The driver posts clicks to its exact MyGoSurface, finds Inspector row buttons through UIA, sends window key events, and captures only that owned window with PrintWindow. Passive per-paint fixture diagnostics establish that motion has begun before interruption; they are absent from production builds.

Accepted final capture from the final implementation: initialY2531.5332; actual moving frameY2528.3567/pending+moving true. After Escape, offset remained2528.3567, pending/moving/highlight cleared, and was still exactly2528.3567 at the next sample1.842seconds later. Ordinary A then allowed answer7 to land at361.40805 and expire; answer30 landed at2525.5747, equal to its updated maximum, then expired. See [desktop state](evidence/inspector-native-pass251-smooth-desktop-state.json), [cancel](evidence/inspector-native-pass251-smooth-desktop-cancel.png), [landing](evidence/inspector-native-pass251-smooth-desktop-landed.png) and [latest](evidence/inspector-native-pass251-smooth-desktop-latest.png). The driver samples for bounded completion; desktop latency is not used as a frame-exact curve measurement.

Earlier diagnostic runs were rejected as evidence: first metrics read preceded the fixture's frame; fixed sleep observed an active514ms flash; an Escape issued100ms after click sometimes preceded movement and therefore only tested queued cancellation. The accepted run waits for a real moving paint before Escape. Original ElectronPID20012/backend16008 on3080 was not modified, restarted or signalled. Owned referencePID16764/backend8040 and every exact-path native fixture were closed;3083/19444 are free.

## Validation and build

Executed affected checks with CGO_ENABLED=0:

```text
go test ./internal/ui/chat ./internal/ui/frame ./internal/ui/theme -count=1
go test ./cmd/whaleisle -run 'TestWorkspaceCaption|TestNativeConversation' -count=1
go vet ./internal/ui/chat ./internal/ui/frame ./internal/ui/theme ./cmd/whaleisle
cd third_party/mygo
go test ./ui -run '^TestList|^TestInputObserver|^TestScroll|^TestTransition|^TestHandle|^TestPhysical|^TestKey|^TestScope' -count=1
go build ./...
go vet ./...
go vet -tags mygo_noinspector ./...
GOOS=linux go build ./...
```

Full affected app suites passed Chat5.154s/frame12.579s/theme0.971s; command selection17.658s. After the final covered-row replacement repair, focused Chat8.152s/frame6.439s and toolkit1.760s passed. Windows toolkit build/vet, production-noinspector vet and Linux build passed. Full toolkit tests and Linux vet are not clean; the separate baseline comparison below records the failures. No JS bridge/native shader change requires rebuilding bundles.

Final default and pass251 executable SHA256:A849175A1D391E065BE70C62A9C7C32A9AE3F7898E657D9B4C13AE9B8D297074; length85,129,216bytes. The default protocol path still targets `whaleisle.exe`. No model request was made. A full production Host boot/shutdown smoke was not repeated; the previously observed shutdown hang remains open.

Baseline comparison completed against a fresh isolated archive of pre-change commit a7c58fda's toolkit. All four failures reproduce unchanged: `TestDiffFrameAllocs` (steady6/within-rows13), `TestShapeTextLineNativeFallbackAndGraphemeEllipsis`, `TestPreviewCleanupOwnsOriginalDirectory`, and Linux vet's unsafe.Pointer diagnostic at internal/linux/browser_seat.go55. The current allocation counts exactly match the archived baseline. [Baseline check record](evidence/inspector-pass251-toolkit-baseline.json) distinguishes passing affected checks from the unclean full suite. These existing failures remain separate work; this round does not claim a clean full-toolkit result.

## Remaining parity work

Native input has no global touchstart or side-button pointerdown event (side buttons currently translate Back/Forward release), and the current transcript exposes one scrolling ancestor rather than the reference's arbitrary nested ancestor tracking. Virtual List positions for unseen variable-height messages remain estimated until measured, unlike a fully laid-out DOM. Per-monitor scale changes during an already running motion have not been compared. These limits are not certified by the constant-height recording.

Typed Turn/Step/tool/nested-call/group-part target ranking and settled part rectangles, picker presentation, graph/lazy accessor/cycle/paging50 behavior remain unfinished. Whole-window caption/tabs/sidebar/material/fonts/control parity, performance budgets, baseline toolkit failures and production shutdown remain open. The goal stays active.
