# Round136 · 标题带内容内边距、设计缓动与 overlay 退场

Closes the G02 content-position remainder and the G50 curve remainder from
[chrome metrics](<C:/AI/WhaleIsle-MyGo/docs/parity/chrome-metrics-20261024.md>) §4.2 and §4.4. The
caption band stays 48; the overlay-exit half of §4.4 is blocked in the toolkit and is recorded, not faked.

## 1. Caption content row: centre 24 → the plate's 12 / 32 / 4

Native: the 48 DIP plate carries `padding: 12px 8px 4px` (`harness-chrome-inject.js:89`), so its content
box is exactly 32 and the conversation header's row occupies `y = 12..44` (`ConversationRoot.module.css:49-66`
`padding: 12px 28px 0 20px` + `.titleRow { min-height: 32px }` at `:169-173`). MyGo centred a 48-tall title
row, so every title, tab and utility sat 4 DIP too high.

| Item | Before | After |
|---|---|---|
| `workspaceCaptionHeight` (band, hairline floor, `workspaceHeaderBand`) | 48 | 48 (unchanged) |
| Caption row padding | 0 | `12` top / `4` bottom |
| Content row (`workspaceCaptionContent`) | 48 | `48 - 12 - 4` = **32** |
| Conversation title row (`conversation_header.go`) | `Height(48)`, text 14..34 | `Height(32)` at `y=12..44`, text line 18..38, centre **28** |
| Title crumb seat (`Conversation session title`) | hug 20 (`Y=18,H=20`) | spans the content row (`Y=12,H=32`), line centred |
| View tabs (`conversation-view-tabs`) | `Height(48)`, underline at 46..48 | `Height(32)`, underline at 42..44 |
| Hero row | `Height(48)`, centred | `Height(32)` at `y=12..44` |
| Global page seat | `Box.Height(48)` | `Box.Height(32)` |
| Subagent child header | `MinHeight(40)`, `Padding(8,…)` | `MinHeight(32)`, `Padding(0,…)` |
| Trailing cluster / window buttons | `Absolute().Top(12).Height(32)` | unchanged: absolute insets resolve against the padding box, so the 12 DIP band padding does not move them (`ui/layout.go:917-980`) |
| Hairline | `y + H - .5` | unchanged, still the band's bottom edge |

`m.captionRightInset` math is untouched (`workspace_caption.go:39,47-48`): it is measured from the trailing
cluster's painted width plus the native reserve, which did not move.

## 2. Design easing `cubic-bezier(0.4, 0, 0.2, 1)`

New `internal/ui/widgets/easing.go`: `DesignEase(t float32) float32` is the app-local implementation of
`--ds-ease-in-out` (`ui-theme/src/styles/base.css:12`). CSS fixes such a curve by two control points and
moves the parameter with x, so x is solved by 24 bisection steps and y is read at the solution
(`X(s) = 3(1-s)²s·0.4 + 3(1-s)s²·0.2 + s³`, `Y(s) = 3(1-s)s² + s³`). It clamps outside 0..1, stays
non-decreasing, and returns exactly 0 and 1 at the ends.

| t | 0 | 0.10 | 0.20 | 0.25 | 0.35 | 0.50 | 0.75 | 0.90 | 1 |
|---|---|---|---|---|---|---|---|---|---|
| `DesignEase(t)` | 0 | 0.025863 | 0.133947 | 0.236587 | 0.500000 | 0.775561 | 0.959368 | 0.994354 | 1 |
| `ui.EaseInOut(t)` (before) | 0 | 0.004000 | 0.032000 | 0.062500 | 0.171500 | 0.500000 | 0.828500 | 0.968000 | 1 |

The metrics doc's "≈0.29 at t=0.25" was an approximation; the exact solution of the curve is 0.2366
(`x(s)=0.35` is exactly the half-way point of the progress axis, so the curve is front-loaded).

Wired into:
- `internal/ui/widgets/motion.go` — the four tokens (`Overlay` 200, `Popover` 160, `Swap` 120, `Flip` 300 ms)
  and the nil-ease default of `Motion.Animate`.
- `internal/ui/chrome/motion.go` — `PopoverTransition().Ease`, `OverlayTransition().Ease` and `Reveal`'s
  progress (so hand-run Dialog/Popover reveals fade on the same curve as the widget primitives).

Reduced motion is unchanged: `ElementTransition`/`AnimateWith` still jump (`ui.Preferences.ReduceMotion`),
`Reveal` still returns 1 before it reads the clock, and the reduce-motion tests were not touched.

## 3. Overlay exit 100ms — blocked, not implemented

Native `--ds-motion-duration-overlay-out: 0.1s` (`base.css:17`); MyGo's `ui.ElementTransition` carries one
`Duration` for both directions (`third_party/mygo/ui/transition.go:17-19,130-141`: `duration()` is shared by
enter and exit progress), and `ui.Motion` has no per-direction duration. A separate exit duration therefore
needs a toolkit field (`ExitDuration`, or a duration on `ui.Motion`) in `third_party/mygo/ui`, which is
outside this task's write scope and would be a framework change, not an app token.

The affected surface is also smaller than the matrix row suggests: every product overlay (dialogs, lightbox,
settings, projects, file picker, menu popovers, composer chips) enters through the hand-run `chrome.Reveal`,
which is enter-only, and `DialogBase`/`PopoverBase` return nil while closed, so those surfaces have no exit
motion at all. `OverlayTransition()`/`PopoverTransition()` — the only carriers of an `Exit` — have no
non-test consumer in the app today. Recorded rather than faked.

## 4. Tests

- `internal/ui/frame/workspace_caption_test.go` — `TestSharedCaptionIsOneRowAcrossEveryRoute` now asserts, on
  the routes that render the conversation header, that the title seat is `Y == 12 && H == 32` and the
  `Conversation title band` row is `Y == 12 && H == 32`; `TestConversationViewTabsStayInsideTheBand` pins the
  tab label to the `12..44` content row. The band stays 48 in all four routes.
- `internal/ui/widgets/easing_test.go` — new: nine solved samples, exact ends, 1001-step monotonicity,
  clamping of −0.5 / 1.5 / NaN, and a check that the curve is not `ui.EaseInOut` at 0.25.
- `internal/ui/widgets/motion_test.go` — every token now rides `DesignEase` (0.25 and 0.5 sampled); the old
  "0.5 must be the middle" assertion is replaced because that was the cubic's property, not the design
  curve's.
- `internal/ui/chrome/motion_test.go` — transitions must ease on `DesignEase`; `TestOverlayRevealFadesPanelAndMask`
  now also samples a quarter into the fade and requires it visibly ahead of the cubic in-out (0.24 vs 0.06).

Results: `go test ./internal/ui/frame ./internal/ui/widgets ./internal/ui/chrome ./internal/ui/chat -count=1`
→ all ok. Broader sweep `go test ./internal/ui/... ./cmd/whaleisle -count=1` → all ok except
`internal/ui/diffsurface`'s `TestCaptionInitializeIsOneGestureAndCapturesItsRoot`, which fails identically on
a clean HEAD worktree (git-init fixture, unrelated). `go vet` clean on the three touched packages.

No golden image in the repository encodes the old header positions (`*.png` goldens are exported on demand
through `SHELL_SNAPSHOT_DIR`, `frame/snapshot_test.go:22-24`), so no 4 DIP golden shift had to be absorbed;
the hand-exported shell/conversation screenshots will move 4 DIP down with the band's content row.

## 5. Residuals

- Overlay exit 100 ms and the native `scale(0.96)` still need toolkit support (`ElementTransition` exit
  duration; a scale field on `ui.Motion`).
- `browserpreview` still passes `TopInset: chrome.TitleBarHeight(12) + …` while the painted band is 48
  (recorded in the metrics doc §4.5).
- Caption button hover still has no `#e81123` fill (visual only).
