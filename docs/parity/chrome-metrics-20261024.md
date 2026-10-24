# Chrome metrics and motion verification — G01 / G02 / G50

Date: 2026-10-24. Branch `main`, HEAD `3ee88ae5`. Read-only investigation: this file and
`chrome-metrics-20261024.json` are the only writes; no product code, no installed-app change, no commit.

Evidence class: **source-only**. The matrix's own limitation applies (`gap-matrix-20261024.md:169`): the
packaged window cannot be screenshotted under the Host provenance and 150% DPI gate, so nothing below is a
pixel measurement of the running reference. Values marked *(rendered)* come from the Go renderer via
`ui.NewTester`, not from the Electron app.

Line numbers under `reference/electron/**` come from the read-only snapshot. Line numbers for
`cmd/whaleisle/main.go` are **HEAD `3ee88ae5`**; the working tree carries other agents' uncommitted edits to
that file, so a dirty checkout shows those anchors shifted by up to ~45 lines (the symbol names are quoted in
every row so the citation survives the shift).

---

## 1. Question 1 — what the native 1440x920 MAIN window actually renders

### 1.1 `TITLEBAR_HEIGHT = 48` is a dead export

| Fact | Evidence |
| --- | --- |
| Declared `const TITLEBAR_HEIGHT = 48` | `reference/electron/src/main/chrome.js:9` |
| Only other occurrence | `reference/electron/src/main/chrome.js:376` — the `module.exports` list |
| Consumers in the whole reference tree (js/mjs/cjs/ts/tsx/css/html, `node_modules` excluded) | **none** — a repo-wide grep for `TITLEBAR_HEIGHT` returns only :9 and :376 |

So `TITLEBAR_HEIGHT` never sizes the main window. It is not the band, and it is not the boot/launcher
caption either (those use `--caption-h`, see 1.2).

### 1.2 Which windows/surfaces own which band

| Surface | Band owner | Height |
| --- | --- | --- |
| Boot page (`boot.html`, shown while the harness loads) | `boot.css:2 --caption-h: 48px`; `.caption` `boot.css:39-47` (fixed, `right: 120px`, `app-region: drag`) | 48 |
| Launcher page (`launcher.html`) | `launcher.css:21-29 .caption` (fixed, `right: 120px`, drag) | 48 |
| Shared defaults for both pages | `window-controls.css:9 height: var(--caption-h, 48px)` | 48 |
| **Main window (1440x920), harness covered** | `harness-chrome-inject.js` injected into the harness page | **48** |
| Harness page drag rectangle | `AppFrame.module.css:277-284 .captionDrag` (`height: 48px`, `grid-column: 1 / -1`) | 48 |
| Conversation header content row inside that rect | `ConversationRoot.module.css:49-66` (`padding: 12px 28px 0 20px`) + `.titleRow` `min-height: 32px` (`:169-173`) | 44 |
| DSH standalone desktop app (NOT this product's main window) | `apps/desktop/src/windows-layout.ts:4 WINDOWS_TITLEBAR_HEIGHT = 40`; `apps/desktop/src/main.ts:215`; `apps/desktop/src/preload-windows.ts:13`; consumed by `AppFrame.module.css:26-31,74-77` | 40 |

The 40 DIP harness titlebar is **not** in play: `data-windows-titlebar` is set only by the DSH desktop
preload (`apps/desktop/src/preload-windows.ts:13`), and Whale Isle loads the harness Web UI into a plain
`BrowserView` with its own preload (`window.js:633-658`) whose bounds are the full content box
(`window.js:379-387`: `{0,0,content.width,content.height}`). Therefore `AppFrame.module.css:74-77` does not
match, and the harness page keeps its default 48 DIP caption geometry.

The boot document is hidden once the harness covers the window, so the boot page's own 48 caption and 48
`--caption-h` are not what the user sees in the main window (`boot.css:64-67`
`body[data-harness-covered] .caption { no-drag; pointer-events: none }`; `boot.css:78-80`
`visibility: hidden`).

### 1.3 The main-window chrome that is actually painted

`reference/electron/src/main/harness-chrome-inject.js`:

| Metric | Value | Line |
| --- | --- | --- |
| `CAPTION_HEIGHT` (plate height and published `--dshd-wco-caption`) | 48 | `:11`, `:75`, `:88` |
| `CONTROL_SIZE` | 32 | `:4`, `:96-97` |
| `CONTROL_GAP` | 0 | `:5`, `:86` |
| `EDGE` (plate padding-left/right, i.e. the right inset) | 8 | `:8`, `:89` |
| `CLUSTER` (air after the buttons) | 8 | `:9` |
| Plate padding | `12px 8px 4px` | `:89`, `:269` |
| Button radius | 8 | `:104` |
| Glyph box | 12x12 | `:111-115` |
| `--dshd-wco-controls` (cluster reserve) | `8 + 32*3 + 0*2 + 8` = **112** | `:61-63`, `:74` |
| `--dshd-wco-caption` | 48 | `:75` |
| Button vertical placement | plate is 48 with `align-items: center` and 12 top / 4 bottom padding, so the content box is exactly 32 and a 32px button occupies **y = 12..44** | `:83-89`, `:96` |
| Close hover | `#e81123` background, white glyph | `:120-123` |
| Page-side cluster | `AppFrame.module.css:333-348 .titlebarTrailing` height 32, `margin-top: 12px`, `margin-right: var(--dshd-wco-controls, 8px)` | — |

Page-side consumers of the reserve: `AppFrame.module.css:346`, `ConversationRoot.module.css:76-93`,
`SidebarRight.module.css:196`, `SurfaceTabs.module.css:11`, and `ImageLightbox.module.css:37` for the caption
strip. The inject also hides `[data-sidebar-right-expand]` (`harness-chrome-inject.js:126-128`).

### 1.4 The 30x30 pill is a default that no shipped shell page uses

`window-controls.css` declares `gap: 2px` (`:14`), `padding: 0 10px` (`:16`), buttons `30x30` (`:25-27`) and
`border-radius: 999px` (`:30`). Both pages that link it override the whole geometry:

- `boot.html:2` carries `data-boot-theme`; `boot.css:50-59` sets `gap: 0; padding: 12px 8px 4px` and buttons
  `32x32; border-radius: 8px`, under the comment "Match main-window geometry".
- `launcher.css:33-42` sets the same three declarations unconditionally, under the comment "matching the
  main-window chrome".
- `window-harness-cover.test.js:280-281` pins the boot override text (plate `padding: 12px 8px 4px`, button
  `width: 32px; height: 32px; border-radius: 8px`).

The main window does not load `window-controls.css` at all; it gets the injected plate. Conclusion: **30x30 /
gap 2 / padding 0 10 / radius 999 is dead CSS for the shipped shell**; a matrix row quoting it is reading the
shared default, not the rendered chrome.

### 1.5 MyGo's actual numbers

| Item | Evidence | Value |
| --- | --- | --- |
| `chrome.TitleBarHeight` | `internal/ui/chrome/titlebar.go:36` | 12 |
| `chrome.EffectiveTitleBarHeight()` | `internal/ui/chrome/titlebar.go:39-41` | 12 |
| Window option passed to the platform | `cmd/whaleisle/main.go:2202` (`TitleBarHeight: int(chrome.EffectiveTitleBarHeight())`) | 12 |
| Windows caption overlay options | `cmd/whaleisle/main.go:2208` (`WindowsCaptionButtons`) | 32x32, Top 8, Right 8, Corner 8, Glyph 12 |
| Platform `titleBarHeight()` | `third_party/mygo/internal/windows/titlebar.go:158` | `opts.TitleBarHeight` = **12** |
| Platform `captionGeometry()` | `third_party/mygo/internal/windows/titlebar.go:162-167` | (32, 32, 8, 8) |
| Platform `TitleBar()` | `third_party/mygo/internal/windows/titlebar.go:169-175` | `{Height: 12, Right: 32*shown + 8}` = 104 for three buttons |
| Go-drawn caption buttons (fallback painter) | `internal/ui/frame/caption_buttons.go:46-70` | `Absolute().Top(8).Right(8/40/72).Size(32,32).Radius(8)` |
| Windows main-window title row | `cmd/whaleisle/main.go:3903-3909` (`sharedNativeCaption` at `:3875` → `ui.Box(c).Height(0)` at `:3906`) | **no 12 DIP row is drawn** |
| Shared caption band | `internal/ui/frame/workspace_caption.go:31` `workspaceCaptionHeight = 48`; `:36` `MinHeight(48)`; `:88` trailing `Top(12) Right(112) Height(32)`; `:111-113` hairline at row bottom; `:114` returns `max(48, row.H)`; `:39` `controls = max(112, c.TitleBar().Right+8)` | **48** |
| Rendered proof | `go test ./internal/ui/frame -run "TestSharedCaptionIsOneRowAcrossEveryRoute|TestConversationViewTabsStayInsideTheBand" -count=1` → **PASS**; asserts `center.Y == details.Y == 48` and trailing `Y=12 H=32` ending at `width-112` (`workspace_caption_test.go:48-55`), and `band == 48` (`:75-77`) | 48 |

`ui.Box(c).Height(0)` at `main.go:3906` is the decisive line: on Windows with the harness ready, MyGo adds no
title strip above the shell, and the workspace caption row is the only band. The `12` therefore reaches the
platform (`TitleBar.Height`, read by `chrome.TitleBarWithActions`; and `TopInset` for child webviews) but
**not the painted band**.

---

## 2. Table (a) — native vs MyGo vs verdict

| Metric | Native (1440x920 main window) | MyGo | Verdict |
| --- | --- | --- | --- |
| Title band height | 48 drag rect (`AppFrame.module.css:277-284`), 48 controls plate (`harness-chrome-inject.js:11,88`) | 48 (`workspace_caption.go:31,36,114`), render-verified | **match** |
| Band content row inside the band | 44 = 12 top padding + 32 title row (`ConversationRoot.module.css:49-66,169-173`); the last 4 DIP of the 48 rect overlays content | 48 box with contents vertically centred (`conversation_header.go:41`, `AlignItems(Center)`) | **difference 4 DIP** (source-only) |
| Caption button box | 32x32, radius 8, glyph 12 | 32x32, radius 8, glyph 12 (`main.go:2208`; `caption_buttons.go:46`) | **match** |
| Caption button gap | 0 (`harness-chrome-inject.js:5`; effective pitch 32 → Right 8/40/72) | 0 (`caption_buttons.go:54-70`) | **match** |
| Caption button right inset | 8 (`:8,89`) | 8 (`main.go:2208`) | **match** |
| Caption button top inset | **12** (`padding-top: 12px` on a 48 plate) | **8** (`main.go:2208`; `caption_buttons.go:46`; `titlebar.go:164`) | **difference 4 DIP** — the one real G02 geometric delta |
| Cluster reserve before the buttons | 112 (`:61-63,74`; page margins consume it at `AppFrame.module.css:346`) | 112 (`workspace_caption.go:39,88`) | **match** |
| Close hover treatment | plate `#e81123`, white glyph (`harness-chrome-inject.js:120-123`) | no hover fill; glyph in theme muted ink (`caption_buttons.go:55-58`) | **difference** (visual only) |
| Title font / size / line | 14px / 20px crumb (`ConversationRoot.module.css:245-246`), selected crumb weight 500 + primary ink (`:262-266`), 4px vertical crumb padding (`:241`) inside the 32 DIP row | 14 / 20, weight 500 (`conversation_header.go:47`, `theme.InterfaceSize`) | **match** |
| Title vertical position | title row y = 12..44, centre 28 | centred in the 48 row, centre 24 | **difference 4 DIP** (same offset as the buttons) |
| Bottom hairline | conversation header `::after` at `bottom: 1px`, 0.5px (`ConversationRoot.module.css:104-114`) | `workspace_caption.go:111-113`, `H - .5`, 0.5px | **match** |
| Drag region shape | one 48 DIP rectangle across **all** columns (`grid-column: 1 / -1`, `AppFrame.module.css:277-284`), with no-drag boxes punching holes (`:295-307`, `ConversationRoot.module.css:151-154`) | 48 DIP row across chat + details (`workspace_caption.go:36`), nested title band also drag (`conversation_header.go:41`); sidebar has its own 60 DIP brand drag row (`internal/ui/sidebar/sidebar.go:182-183`) and a 6 DIP footer drag (`:378`) | **difference (structural, not a miss)** — hit area is ≥ native, but there is no single full-width rect |
| Window button ownership | page-painted overlay `#dshd-shell-controls`, z-index max, no-drag (`harness-chrome-inject.js:77-110`) | native Win32 layered caption bar (`third_party/mygo/internal/windows/titlebar.go:195+`) **plus** Go-painted duplicates at identical geometry (`caption_buttons.go:13-22,46`) | **difference (implementation)**, functionally equivalent; the two layers must stay in sync (§4.1) |
| Maximize corner radius | OS rounding off, page draws radius 20 (`chrome.js:41-43`; `harness-chrome-inject.js:14,151-153`) | `WindowsFrameCornerRadius: 20` (`main.go:2209`) | **match** |

---

## 3. Table (b) — exact motion tokens (G50)

Native paths below are relative to
`reference/electron/vendor/deepseek-harness/packages/client/` for `ui-*` packages unless prefixed with
`apps/`. Every native value is the token declaration itself, not a consumer's copy.

| Token | Native value | Native definition | MyGo value | MyGo definition | Verdict |
| --- | --- | --- | --- | --- | --- |
| shared duration | `0.2s` | `ui-theme/src/styles/base.css:13` | `200ms` | `internal/ui/widgets/motion.go:27,37`; `internal/ui/chrome/motion.go:15` | **match** |
| fast (swap / overlay-out) | `0.1s` | `base.css:14`; `:17` (`--ds-motion-duration-overlay-out`) | swap `120ms`; overlay exit `200ms` (reuses enter) | `widgets/motion.go:29,41`; `chrome/motion.go:59-65` | **difference** (swap +20ms, overlay-out +100ms) |
| slow (frame tracks) | `0.3s` | `base.css:15` | no token; per-site 150/200ms | `internal/ui/sidebar/sidebar.go:167-175` (150ms rail) | **difference** (only where tracks animate) |
| popover duration | `0.16s` | `base.css:18 --ds-motion-duration-popover`; used at `ui-primitives/src/MenuSurface.module.css:53,58` and `motion.css:34-40` | `150ms` | `widgets/motion.go:28,39`; `chrome/motion.go:17` | **difference −10ms** (matrix correct) |
| popover distance | `translateY(4px)` | `base.css:22`; `motion.css:44` | enter `Y = 6`, exit `Y = 3` | `chrome/motion.go:23,45-46` | **difference** (enter 6 vs 4, exit 3 vs 4) |
| overlay duration | `0.2s` | `base.css:16` = `--ds-transition-duration`; `motion.css:20-26` | `200ms` | `chrome/motion.go:15,59-65` | **match** |
| overlay distance + scale | `translateY(8px) scale(0.96)` | `base.css:21,23`; `motion.css:30` | `Y = 2`, no scale | `chrome/motion.go:24,63-64`; `ui.Motion` has only `X, Y, Opacity, Collapse` (`third_party/mygo/ui/transition.go:40-49`) | **difference** (8/0.96 vs 2/1.0; scale needs a toolkit field) |
| overlay exit | `0.1s` (`overlay-out`) | `base.css:17`; `motion.css:28-32` | `200ms`, same distance as enter | `chrome/motion.go:61-65` | **difference** |
| modal enter | `modalEnter var(--ds-transition-duration) var(--ds-ease-in-out)` = 0.2s, **opacity only**; mask `::after` uses the same | `ui-primitives/src/Modal.module.css:25-31,48,51-54` | `200ms`, `Y=2`, opacity | `chrome/motion.go:59-66,140-144`; `chrome/dialog_surface.go:43` | **partial**: duration match, MyGo adds a 2 DIP travel the reference modal does not have |
| shared ease | `cubic-bezier(0.4, 0, 0.2, 1)` | `base.css:12 --ds-ease-in-out` | cubic in-out: `t<.5 ? 4t³ : 1-(2-2t)³/2` | `third_party/mygo/ui/widgets.go:128-135` | **difference (curve)**: same endpoints and midpoint, different slope; at t=0.25 MyGo ≈ 0.063 vs ≈ 0.29 for the reference bezier |
| flip | `0.4s`, rotateX ±80deg | `base.css:20`; `motion.css:66-109` | `300ms` | `widgets/motion.go:30,43` | **difference −100ms** |
| reduced motion | all tokens zeroed | `base.css:42-49`; `motion.css:111-116`; `Modal.module.css:56-58` | framework skips transitions; `Reveal` returns 1 | `chrome/motion.go:92-94`; `ui.Preferences.ReduceMotion` | **match** |
| consumers | — | `MenuSurface.module.css:53,58`; `ConnectionIndicator.module.css:18-22` (160/150ms mix) | `PopoverTransition()` at `chrome/glass.go:327`, `composer/chip.go:36,44`, `sidebar/sidebar.go:247`, `settings/settings.go:209`, `sidebar/workspace_heading.go:88`; `Reveal(...PopoverDuration)` | — | **wired; values differ as above** |

Nothing here was measured from a screen recording; every native value is a CSS token read from the installed
source and every MyGo value a Go constant. Rendered timing is **unverified** for both sides.

One nuance worth recording: the reference mix of 160ms and 150ms is not an inconsistency. `0.16s` is the
popover token; the 150ms values in `SidebarRoot.module.css:135,157,161` and
`ConnectionIndicator.module.css:21-22` are the rail settle and indicator timings, which are not popover
surfaces.

---

## 4. Table (c) — minimal-risk fix list

Ordered by value per risk. Each row names the exact edit, the test that would prove it, and the risk.

### 4.1 Caption button top: 8 → 12 (the only true G02 geometry delta)

- `cmd/whaleisle/main.go:2208` — `WindowsCaptionButtons{... Top: 12 ...}` (keep 32/32/Right 8/Corner 8/Glyph 12).
- `internal/ui/frame/caption_buttons.go:46` — `.Top(8)` → `.Top(12)`; the Go-painted trio must move with the
  native overlay or the two layers stop coinciding.
- Test: extend `third_party/mygo/internal/windows/caption_geometry_windows_test.go` to require `top == 12`, and
  add a `ui.NewTester` case in `internal/ui/frame/caption_buttons_test.go` asserting the close rect is
  `Y == 12, H == 32, X + W == width - 8`.
- Risk: low. Hit target unchanged; the bottom edge lands at 44, which is the reference plate's 4px bottom
  inset. `win.Close/Minimize/ToggleMaximize` behaviour untouched.

### 4.2 Band content vertical position: centre 24 → 28 (4 DIP)

- `internal/ui/frame/workspace_caption.go:36` — keep `MinHeight(48)` but give the row `Padding(12, 0, 0, 0)`
  (or `AlignItems(ui.Start)` plus an inner 32-tall row) so title and view tabs occupy y = 12..44 the way
  `ConversationRoot.module.css:49-66,169-173` does.
- Test: assert the conversation title seat `Y == 12 && H == 32` in
  `TestSharedCaptionIsOneRowAcrossEveryRoute` (today it only asserts the columns start at 48 and the trailing
  cluster at 12/32).
- Risk: medium-low. The band stays 48 with no second row and the hairline stays at the row bottom, but the
  view tabs (`workspace_caption.go:135-151`) and the hero header (`:59-64`) share this row and need the same
  inset; goldens encoding header text positions may move 4 DIP.

### 4.3 Popover token: 150 → 160ms, rise 6 → 4

- `internal/ui/widgets/motion.go:28,39` — `PopoverMotion`, `PopoverDuration` → `160 * time.Millisecond`.
- `internal/ui/chrome/motion.go:17` — `PopoverDuration` → `160 * time.Millisecond` (equality is pinned by
  `chrome/motion_test.go:15-19`).
- `internal/ui/chrome/motion.go:23,45-46` — `popoverRise = 4`, exit `Y: 4` (the reference exits at the same
  distance, `motion.css:44`).
- Tests: `internal/ui/widgets/motion_test.go:15` and `internal/ui/chrome/motion_test.go:41` pin 150ms today and
  must change in the same edit; then a `Reveal` timing case at +80ms/+160ms
  (`chrome/motion_test.go:64-66`) proves the new settle point.
- Risk: low (timing only), but it is user-visible across menus, chips and floating cards, so land it alone.

### 4.4 Overlay recipe: distance 2 → 8, exit on the fast token

- `internal/ui/chrome/motion.go:24,63-64` — `overlayRise = 8`; give the exit its own `100ms` (the reference's
  `overlay-out`, `base.css:17`; `motion.css:28-32`) instead of reusing `OverlayDuration`.
- Test: overlay progress in `chrome/motion_test.go` — settled at +100ms on close, enter offset == 8.
- Risk: medium. `Reveal` drives both directions from one duration today, and mask and panel must not
  desynchronise (`chrome/motion.go:116-123`). The reference's `scale(0.96)` cannot be reproduced: `ui.Motion`
  has no scale field (`third_party/mygo/ui/transition.go:40-49`), so this row stays a **partial** fix unless the
  toolkit gains one. Do not fake the scale with layout.

### 4.5 Residual hazards from `TitleBarHeight = 12` (not the band, but the same constant)

- `internal/ui/browserpreview/{media,pdf,html}.go:23-26` and `controller.go:128` pass
  `TopInset: chrome.TitleBarHeight + BrowserChromeHeight` to `desktop.NewHost`, so an embedded child window is
  positioned from a 12 DIP title reserve while the real chrome is 48 DIP. **Unverified** whether the panel rect
  is already below the band before the inset is applied; if not, the preview starts 36 DIP too high and
  overlaps the caption. A fix would use `frame.workspaceCaptionHeight` (or a shared exported constant) and is
  not minimal-risk until the offset is confirmed on a running window.
- `internal/ui/chrome/titlebar.go:84-90` — `h = bar.Height` (= 12 via `titlebar.go:174`) when `bar.Height > 0`,
  and only a non-nil `leading` lifts it to 32 (`:88-90`). On Windows this path is reachable only when the
  harness is not ready (`main.go:3901-3902`), so the visible effect is a 12 DIP strip holding a 13px label.
  **Unverified visually**; the cheap guard is an unconditional `if h < 32 { h = 32 }`, which also changes
  macOS/Linux titles and therefore needs its own parity pass.
- Leave `chrome.TitleBarHeight = 12` itself alone for now: it is the platform caption reserve, and the Windows
  caption overlay reads only `Right` from `TitleBar()` (`workspace_caption.go:39`), so changing it would move
  the reserved cluster width and the platform overlay geometry at once.

### 4.6 Documentation of the dead export

`reference/electron/src/main/chrome.js:9,376` — `TITLEBAR_HEIGHT` has no consumer. Either cite
`harness-chrome-inject.js:11` as the band owner in the matrix, or delete the constant upstream. Not a MyGo
change; recorded so the next parity pass does not re-derive 48 from a dead export.

---

## 5. Verdict

**G01 is a measurement error — not a 36 DIP defect and not an 8 DIP difference.**

- Native main window: a **48 DIP** chrome band (48 controls plate, `harness-chrome-inject.js:11,88`; 48 DIP
  full-width drag rectangle, `AppFrame.module.css:277-284`), with the conversation header's content row at
  **44 DIP** inside it. `TITLEBAR_HEIGHT = 48` (`chrome.js:9`) is exported and never consumed, so it is not the
  evidence; the inject is.
- MyGo: a **48 DIP** shared caption band (`internal/ui/frame/workspace_caption.go:31,36,114`), rendered and
  pinned by `TestSharedCaptionIsOneRowAcrossEveryRoute` (PASS at HEAD). The `12` in
  `internal/ui/chrome/titlebar.go:36` is the platform caption reserve; on Windows the main window draws
  `ui.Box(c).Height(0)` (`main.go:3906`) and never a 12 DIP strip.
- So the band height is **48 vs 48: match**. The genuine deltas are **4 DIP**, twice: caption buttons sit at
  y=8 instead of y=12, and the title/trailing content is centred at 24 instead of 28. Those four DIP, plus the
  popover 150 vs 160ms and the popover rise 6 vs 4, are the whole of G01/G02/G50's real geometry-and-timing
  content.
- The "30x30 pill / gap 2 / padding 0 10" numbers come from `window-controls.css:14-34`, which every shipped
  shell page overrides (`boot.css:50-59` via `boot.html:2 data-boot-theme`; `launcher.css:33-42`) and which the
  main window never loads. The real caption button box is **32x32, radius 8, gap 0, right 8** on both sides;
  only the **top inset (12 native vs 8 MyGo)** differs.
