# Round 148 — Motion.Scale, per-direction exit duration, native overlay recipe

Closes the last G50 gaps by extending the toolkit where it was missing, then wiring the native recipe.
Native values: `ui-theme/src/styles/base.css:16-23` (`overlay` 0.2s in, `overlay-out` 0.1s, distance 8px,
scale 0.96, popover 0.16s / 4px) and `ui-theme/src/styles/motion.css:20-44`.

## 1. Toolkit API (`third_party/mygo/ui/transition.go`)

```go
type Motion struct {
    X, Y     float32
    Opacity  float32
    Collapse bool
    // Scale shrinks (below 1) or grows (above 1) the element's box around its
    // centre there: 1, and 0, leave it at its size.
    Scale float32
}

type ElementTransition struct {
    Duration     time.Duration // 200 ms when zero
    // ExitDuration is how long an exit takes: Duration when zero.
    ExitDuration time.Duration
    Ease         Easing
    Position, Size, Colors bool
    Enter, Exit  *Motion
}
```

`Scale` is additive: `0` means identity, so every existing `&ui.Motion{Opacity: 0}` call site keeps its
behaviour without a change.

## 2. Interpolation

- `tbox` gained `scale`; `Motion.Scale` is read in `tbox.moved` (values ≤ 0 normalise to 1) and lerped in
  `tbox.at` under a new `moveScale` bit, which enters and exits always move. So an enter from 0.96 runs
  0.96 → 1 across the duration along the transition's easing, and an exit runs 1 → 0.96.
- `apply` keeps the box's centre and shrinks the painted box: `w' = w·scale`, `x' = x + (w − w')/2` (same for
  height), and sets `flags |= flagClip`. The element is not re-laid out at the smaller size: the toolkit has
  no paint-time transform (no per-op matrix in `internal/scene`), so a scaled box shows its content clipped
  rather than 4% smaller. The enter is also mostly transparent at that point, and the animation lands on the
  exact layout box (`scale == 1` ⇒ no clip, no size override).
- Exits take the same path in `animateExits`: the ghost copy shrinks around its centre and clips.
- Exits progress over `exitDuration()` (`ExitDuration`, else `Duration`); `progressExit` is a separate call
  from `progress`, so a layout move or a color fade still uses `Duration`.
- Reduced motion is unchanged: the engine sets `moving = 0` under `ReduceMotion` (and on window resize), so
  both the box and its scale jump straight to the settled value. Tests cover that path for scale.

## 3. App wiring (`internal/ui/chrome/motion.go`)

| Surface | Duration in | Exit | Rise | Scale | Ease |
|---|---|---|---|---|---|
| Overlay | 200 ms | **100 ms** (`OverlayExitDuration`) | 8 | 0.96 | `widgets.DesignEase` |
| Popover | 160 ms | 160 ms (no `ExitDuration`) | 4 | none | `widgets.DesignEase` |

`Reveal` is unchanged: it is the enter-only, hand-run path for `DialogBase`/`PopoverBase` panels, and it still
drives the panel and its dim mask from one progress value (`DialogReveal`), so mask and panel stay in sync. The
100 ms exit applies to elements that carry `OverlayTransition` and whose parent survives — those are the only
ones whose exit the framework plays at all.

## 4. Tests

Toolkit (`third_party/mygo/ui`):
- `TestTransitionScalesTheBoxAroundItsCentre` — an element entering at `Scale: 0.5` into a 200-wide column
  starts 100 wide at x=50 and clipped, is 150 at x=25 halfway, and lands at 200/x=0 unclipped; its content never
  moves; under `ReduceMotion` it appears at the full 200 at once.
- `TestTransitionExitUsesItsOwnDuration` — `Duration: 200 ms`, `ExitDuration: 100 ms`: the leaving copy starts
  at its own width, is 190 wide (of 200) at +50 ms, and is gone at +110 ms; a fresh element entering still
  takes the full 200 ms (mid-flight at +110 ms, settled at +210 ms).
- Full `go test . ./ui/...` in `third_party/mygo`, plus `GOOS=linux|darwin|windows go build ./...`.

App (`internal/ui/chrome`):
- `TestOverlayRecipeMatchesTheDesignSystem` — overlay Y=8/scale 0.96 in and out, 200 ms in, 100 ms out; popover
  Y=4, no scale, 160 ms both ways.
- `TestOverlayTransitionScalesInAndExitsFast` — renderer-driven: the panel finds itself 96% wide and centred on
  the first frame, between that and full halfway, exact at +200 ms; after closing, the copy is still painted
  50 ms into the exit and gone at 100 ms, i.e. before the enter token would have finished.
- Existing chrome/widgets/frame suites and `go vet` on the three packages.

## 5. Residuals

- `Motion.Scale` shrinks the box and clips its content; it does not scale glyphs or images, because the
  renderers have no per-element transform (`internal/scene.Op` has no matrix, `Painter.scale` is the device
  DPI). A true 4% glyph scale needs a paint-time transform in the scene and all four renderers; recorded, not
  faked with layout.
- `DialogBase`/`PopoverBase` panels still cannot play either direction (they return nil while closed), so
  dialogs get the enter through `Reveal` and no exit at all — the 100 ms exit is reachable only through
  `ElementTransition` today. A real dialog exit needs the base to keep the panel mounted for one exit
  duration, a lifecycle change of its own.
- `browserpreview`'s `TopInset` use of `chrome.TitleBarHeight` (12) against the painted 48 DIP band is still
  open (round 133/134).
