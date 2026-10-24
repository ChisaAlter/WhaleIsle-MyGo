# Round 226 — hero vertical stack: probe, reconciliation, and the 19-DIP lift

## 1. The 373.33 vs 398 discrepancy, resolved

| source | element measured | rect (1440×860) |
| --- | --- | --- |
| round 223 native **UIA of the running window** | preset chip **button** (标准模式) | y = **373.33**, h = 28.67 |
| round 181 DOM probe of the harness **web page** | preset chip **text span** | y = **398**, h = 20 |
| this round, same DOM page | preset chip **button** | y = **394**, h = 28 (text 398..418) |

So both numbers were right about different things: 398 is the *text span* inside the chip, and the
harness page's chip button sits at 394, while the **running window's** chip is at 373.33 — the page and
the window differ by ~20 DIP in that row. The window is the acceptance target, so **373.33 is the
number to match**, and round181's 25-DIP headline foot was tuned against a page-only measurement.

## 2. Native window hero (UIA, 1440×860, authoritative)

| element | y (DIP) | h |
| --- | --- | --- |
| slogan 探索未至之境 | 314.67 | 35.33 |
| badge 大肥鱼版 | 321.33 | 15.33 |
| workspace chip 选择工作区 | 373.33 | 28.67 |
| preset chip 标准模式 | 373.33 | 28.67 |
| composer drag separator | 405.33 | 8.67 |
| draft textarea | 417.33 | 52.67 (card top = 417.33 − 8 = **409.33**) |

MyGo before the fix (same pair, `evidence/accept-D-MAIN-223`): slogan 332.67, preset row 393.33,
card top 429.33. Every element is **+18…+20 DIP low** — not a gap error between chips, and not the chip
inset (which is already exact at x=502). The stack is simply centred ~19 DIP lower than native's.

## 3. Fix

`internal/ui/frame/frame.go`: under the hero's lower `Grow` spacer, add a fixed foot
`conversationHeroStackFoot = 38` (`conversation_phase.go`, documented against the native rects). With two
equal grow spacers above and below, a fixed 38-DIP box at the bottom shortens both by 19 and **lifts the
whole stack by 19 DIP**, which is exactly the measured offset. The chip row (inset 32.67 / gap 3.33) and
the composer card recipe (774.39×114) are untouched.

Expected after: slogan 313.67 (native 314.67), preset row 374.33 (373.33), card top 410.33 (409.33) —
each within 1 DIP.

## 4. Verification state

- `go build ./internal/ui/frame` ✓
- `go test ./internal/ui/frame ./internal/ui/composer` ✓ (hero/docked identity, hero row, nav label,
  axis tests all still pass)
- `go vet ./internal/ui/frame ./internal/ui/composer ./internal/ui/chat` ✓

**Not yet verified live**: the temp dark capture at 1440×860 was not run in this slice (the probe plus
the reconciliation consumed the box). The three expected y values above are the acceptance check for the
next capture — if any lands outside ±1 DIP, the foot constant is the single knob to adjust.

## 5. Residual

1. Live capture (D-MAIN hand-off pair) → confirm slogan/preset/card at 314.67/373.33/409.33 ±1, then
   re-run structure (window-relative 11/22, caption-relative 7/22) — the hero cluster is what those
   misses were.
2. Add a pinning test for the foot: a headless 1440×860 hero with workspace + model state must place
   `Composer hero preset row` at 374 ± 1 — needs a fixture with a real `ViewState` (the bare fixture
   renders a 641-wide card and cannot express the hero stack).
3. Badge height: native 15.33 vs MyGo 20 (separate, cosmetic).

## 6. State

Harness web server stopped, no temp profile created this round, native untouched at (200,2) 2268×1366.
Changed files: `internal/ui/frame/frame.go`, `internal/ui/frame/conversation_phase.go` (both uncommitted
by design).
