# Round 210 — axis root cause fixed (card 774.39) and a settled hand-off capture

Two results: the composer axis is **fixed in code with a test** (card now 774 at 469.33, native 774.39
at 469.3), and the hand-off capture now runs native-first with a repaint nudge. The native page in this
session still painted a flat hero, so the pixel numbers remain wallpaper-limited.

## 1. Axis root cause (S27 final)

MyGo's width formulas are percentages of the conversation column, and that column was the
**divider-shrunk** `Sizes.Center`:

```go
// internal/ui/frame/frame.go (before)
cols := computeColumns(maxf(0, viewport-sidebarDivider-SplitterWidth), sidebar, pref, viewport)
sizes.Center = cols.Center            // 1440 − 1 − 1 − 280 = 1158
contentAxis  := conversationContentWidth(colW)   // 0.64 × 1158 = 741.12
composerOuterW := min(contentAxis+64, viewport)  // 805.12 → card 773.12
```

Native has no divider reservation between the tracks: its conversation root is exactly
`viewport − sidebar = 1160`, so `0.64 × 1160 = 742.4` → seat 806.4 → card **774.4**. The 2-DIP divider
reservation (one per splitter) is therefore the whole 1.06-DIP gap, not a composer inset and not a
persisted preference (round186's hypothesis was a red herring — `go-resume.json` was absent this run and
the card was still 773.33).

**Fix** (`internal/ui/frame/frame.go`): the width formulas are fed the *native* column when no right dock
is open — `axisColumn = max(colW, viewport − Sidebar)`; with a dock open the drawn column stays
authoritative (the track arithmetic is the geometry). `m.Sizes.conversationWidth(axisColumn)` is used for
the persisted-preference path too.

**Test** (`internal/ui/frame/hero_axis_test.go`): at 1440 wide, 280 sidebar, `Center` 1158 (the real
divider-shrunk value) → card **774.39 ± 0.6**; the dock case (Right 320) → 773.12, i.e. unchanged by the
dock rule. Both pass, and the whole frame suite is green.

**Live confirmation** (pass290-lineage temp build, git `304d5f35`): `Composer card` UIA rect
**x=469.33 y=429.33 w=774 h=114** vs native **469.3 / 430 / 774.39 / 114** → width **−0.39 DIP** (was
−1.06), y **−0.67**. The workspace chip also widened to **150** (native 150), and `标准模式` now sits at
**675.33** in the left flow.

## 2. The settled capture (native first, repaint nudge)

Sequence used: `tmp/row-capture.ps1` parks MyGo at x=−4200, places native at (0,0), foregrounds it,
**nudges its height by +1 DIP and back** to force a DWM repaint, waits 2.2 s, captures the baseline, then
parks native and captures MyGo at (0,0). Artifacts: `evidence/accept-D-MAIN-210/`
(baseline/actual/diff/diff.json/geom/controls/structure/mask/accept-log).

Hand-off used: `whaleisle://open` after ~42 s (asserted `探索未至之境` + `Composer card` in UIA on
handle 20646960).

Numbers:

| metric | value |
| --- | --- |
| raw pixel diff | **99.98 %** (channel max Δ 249) — unusable, see below |
| regions | sidebar 0.999, header 0.9996, hero 1.0, composer 1.0, footer 0.996 |
| structure (window-relative) | matched 22, **within 1 DIP 0**, outside 22, native-only 37, MyGo-only 27 |

**Why the pixels are still not comparable:** the native baseline samples
`(12,12,12)` at the hero and card, `(19,23,31)` low in the sidebar — i.e. the native page painted a flat
dark surface with only faint wallpaper bleed, while MyGo painted the seeded crop. The size nudge did
**not** restore it, so this is not a stale DWM layer; the native chat page in this session has no
wallpaper layer (the round175/186 baselines that showed the aurora were captured on the same empty
`新会话`, so something in the user's app changed in between — worth one targeted check: does the native
page still have `#dsh-wallpaper` with an image on the *running* window? That needs the app's own CDP, not
the harness web page).

Consequence: the run this round proves the **geometry** (card 774 vs 774.39, chip 150/left flow, triad
x-alignment), not the pixels. The wash-masked ratio is still pending a native side that paints.

## 3. Residual / next slice

1. **Native paint check**: query the running native window (its own devtools endpoint) for
   `#dsh-wallpaper`'s background-image; if it is empty, the user's appearance setting changed and the
   acceptance fixture must seed *both* sides (or the wallpaper must be masked and chrome scored only).
2. **Caption-relative rescore**: `acceptance-structure.py` reports `absolute` only in this run
   (`caption_relative` came back empty); rerun once the script's registration block is stable and report
   the caption-relative within-1-DIP count (the `caption_offset_dip` was 0.67, from the triads).
3. **Structure 0/22**: still dominated by the hero block whose native numbers were walked mid-switch;
   with the axis fixed, a settled re-walk should move it — that is the number the next round must move.

## 4. State

Temp build stopped and deleted, temp profile removed, native restored to (200,2) 2268×1366 on the empty
`新会话`. Gates run: `go test ./internal/ui/composer ./internal/ui/chat ./internal/ui/frame` → ok;
`go vet` on the same three → clean.
