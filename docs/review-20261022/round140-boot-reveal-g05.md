# Round 140 — boot reveal: timing, coverage, easing and scene (G05)

Date: 2026-10-24. Task: task-15. Scope: `internal/ui/boot/`, this doc. No commit.

## 1. Reference evidence (read-only)

`reference/electron/src/renderer/boot-reveal.css`:

- `.dshd-boot-curtain { transition: transform calc(var(--ds-transition-duration, 0.2s) * 3) var(--ds-ease-in-out, ease-out); }` → **600 ms** on the design curve `cubic-bezier(.4,0,.2,1)`.
- `.dshd-boot-curtain-sky { top:0; height:62% }` / `.dshd-boot-curtain-sea { bottom:0; height:38% }`, moved by `translateY(-100%)` / `translateY(100%)` when `html[data-dshd-harness-fade="in"]`.
- `#dshd-boot-curtains { position: fixed; inset: 0; border-radius: inherit; overflow: hidden; pointer-events: none; }` → the layer never takes input.
- `@media (prefers-reduced-motion: reduce) { .dshd-boot-curtain { transition: none !important } }` → the release lands on its end state at once.
- Each curtain holds an `<img>` of the **same snapshot**, `width:100vw;height:100vh`, so the two halves are rigid slices of one page image.

`reference/electron/src/main/window.js:446-509` `revealHarnessView`: the snapshot is `win.webContents.capturePage()` (the boot page itself, caption strip included), the curtain is its own `parent, show:false, frame:false, transparent, focusable:false` BrowserWindow with `setIgnoreMouseEvents(true)`, sized `setBounds(win.getContentBounds())`, and the release starts only after the harness view is mounted and warmed behind it.

`reference/electron/src/renderer/boot.css`: `.scene { position: relative; height: 100%; border-radius: 20px; overflow: hidden }` and `.caption { position: fixed; top: 0; height: 48px }` — the boot page's scene is **full-bleed over the window content box**; the 48 DIP caption is a transparent drag strip *inside* the page, not chrome above it.

## 2. Coverage decision

Native: curtain window = content bounds, layer `inset:0` → covers the whole client area including the 48 DIP caption strip, because the snapshot *is* the page and the page paints the caption strip's background (the scene behind the transparent caption).

MyGo: the shell paints the caption strip as chrome and the boot page starts under it (`cmd/whaleisle/main.go` mounts the boot body in a `Grow(1)` column below the title bar; `boot.View`'s `boot-root` fills that body). The boot page's box is therefore `(0, 48, w, h-48)`, and the curtain must cover **exactly the page it stands in for**. Covering 0..48 would paint boot scene over shell chrome the page never paints — a jump the reference does not have.

Decision: `curtainCoverage(w,h) = {0, min(48,h), w, h-48}` — full width, from the page's top to the window bottom, 48 DIP top offset kept and pinned by `TestBootCurtainCoverage`. If the boot page ever becomes full-bleed (its own scene at y=0, as in the reference), the constant moves with it; the parity gap in the *page* itself is `cmd/`-owned and out of this task's scope.

## 3. Changes in `internal/ui/boot/curtain.go`

- **Timing/easing**: `anchor.AnimateWith(cycle, target, 600ms, widgets.DesignEase)` (was `Animate` → framework `EaseOut`/symmetric curve). 600 ms = `0.2s * 3`; `widgets.DesignEase` is the app's `cubic-bezier(.4,0,.2,1)` (`internal/ui/widgets/easing.go`).
- **Coverage**: one `curtainCoverage` helper for the surface rect, used by the overlay and the tests; the 48 DIP caption constant is documented with the CSS/page evidence above.
- **Scene, not a look-alike**: the synthetic gradient + hard-coded star field (`paintBootBackdrop`) is gone. The curtain now paints `bootBackdrop(bg, accent, m.SystemDark)` — the exact painter `boot.View` uses, with the same `th.Active(m.SystemDark)` palette — so the covered area is the page's own scene.
- **Rigid slices**: the sky/sea windows are the translated page-box slices (`curtainRects`), and the scene is painted shifted by exactly the window's translation (`curtainShifts`) and clipped to the window, then to the page box, so the horizon stays glued to the slice instead of being stretched into a shorter rect. `TestBootCurtainRigidSlices` pins the shifts.
- **Input**: unchanged rule, now documented — covering stage `Modal()` (its layer carries the page's own retry/export/cancel actions), releasing curtain `PassThrough()` (= `pointer-events:none`); the release can never hold a click.
- **Reduced motion**: `ui.AnimateWith` snaps to the target when `Preferences.ReduceMotion` is set, matching `transition: none`; `TestBootCurtainReducedMotionJumps` asserts the finished offsets on the first frame.
- One test seam: `curtainProgressHook` (nil in the shell) samples the painted progress so the timing test can assert the curve on a fixed clock.

### Snapshot limitation (explicit)

The reference freezes a real page capture and slides two `<img>` halves of it, so the brand block, status line and actions slide away with the scene. MyGo has no offscreen capture for native-UI content, and the toolkit has no element transform (no translate/group primitive — only `Painter.Clip`), so the release carries the **scene only**: the boot page's centre content (brand, status, actions) is painted while the stage is covered and disappears at the first release frame, while the split scene slides over the desktop that is already painted underneath. This is the best-effort substitute; a true frozen page needs either a page capture API or a clipped translate group. Also, the reference rounds the curtain layer (`border-radius: inherit`); MyGo keeps the window's own 20 DIP silhouette, which already clips the layer.

## 4. Tests

`internal/ui/boot/curtain_test.go` (all pass, deterministic clock via `ui.Tester.SetTime`):

| test | asserts |
| --- | --- |
| `TestBootCurtainSplitAndExit` (kept) | 62/38 at p=0 (496/304 of 800), both halves leave the box at p=1 |
| `TestBootCurtainCoverage` | box `0,48 1060x612`; frame covers 0..660 with no gap; degenerate small window has no negative box |
| `TestBootCurtainRigidSlices` | at p ∈ {0,.25,.5,.78,1}: sky shift `-0.62·H·p`, sea shift `+0.38·H·p`, window tops equal those shifts, split stays 62/38 |
| `TestBootCurtainDesignTiming` | +0 ms → 0, +150 ms → `DesignEase(.25)`, +300 ms → `DesignEase(.5)` (≈0.78), +600 ms → 1; offsets linear in progress at each sample; no frame after 600 ms |
| `TestBootCurtainReducedMotionJumps` | first frame of the release = 1, sky ends above the box, sea below |

Commands: `go test ./internal/ui/boot -count=1` → ok; `go test ./internal/ui/boot ./internal/ui/frame -count=1` → ok (boot 2.1 s, frame 12.0 s); `go vet ./internal/ui/boot` → clean.

Frame check (scratch tester render, 1060×660, deleted afterwards): at +300 ms (progress `DesignEase(.5)` ≈ 0.78) the sky slice is visible only in rows 48..131 and the sea strip only in 608..660, with the desk underneath between them — the window tops land exactly on `box.Y - 0.62·H·p` and `box.Y + 0.62·H + 0.38·H·p`, and each slice shows the scene rows the reference's anchored `<img>` shows (sky rows `0.62p..0.62`, sea rows `0.62..1-0.38p` of the page box).

## 5. Residuals

- Boot page is not full-bleed (starts at 48 DIP vs the reference's 0): the curtain matches the MyGo page; aligning the page itself is a `cmd/whaleisle` boot-layout change (G05 follow-up).
- Release carries the scene without the page's text/actions (see the snapshot limitation).
- The optional workspace smoke (port 3099) was **not** run: the Lead owns the integration smoke, and task-15 marks it optional. Note for the Lead: in a profile with no workspace the app stops processing posted main-thread tasks after the launcher reveals the main window (round139 §5); the `start → boot → ready` hand-off needs the workspace smoke to confirm.
