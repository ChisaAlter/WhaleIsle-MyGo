# Round 232 — hero stack height: the 2.68-DIP correction lands exactly

## 1. Arithmetic

Round230 live UIA at 1440×860 (all DIP):

| element | native | MyGo before | Δ |
| --- | --- | --- | --- |
| slogan | 314.67 | 313.33 | **−1.34** |
| preset row / chips | 373.33 | 374.67 | **+1.34** |
| composer card top | 409.33 | 410.67 | **+1.34** |

Opposite signs at the ends ⇒ the centred stack was **2.68 DIP taller** than native's, not offset. The
stack foot would move both ends together, so the correction has to come out of an internal gap.

## 2. Fix

`internal/ui/frame/conversation_phase.go`: the hero headline row's bottom pad goes from 25 DIP to
**22.3 DIP** (`conversationHeroHeadlineFoot`), i.e. 25 − 2.68 = 22.32 rounded to the layout's 0.1 DIP
unit. Because the stack is centred, removing 2.68 of height moves the slogan down by 1.34 and the
preset row / card up by 1.34 — both ends straight onto the native values in one change. The comment on
the function now records the native rects and the derivation instead of the old 398-DIP page probe.

## 3. Live verification (temp dark build, direct window UIA at 1440×860)

`whaleisle://open` hand-off, then the capture helper's UIA dump (window-relative DIP):

| element | native | MyGo after | Δ |
| --- | --- | --- | --- |
| slogan 探索未至之境 | 314.67 | **314.67** | **0.00** |
| badge 大肥鱼版 | 321.33 | 320.67 | −0.66 |
| preset row 选择工作区/标准模式 | 373.33 | **373.33** | **0.00** |
| composer card top | 409.33 | **409.33** | **0.00** |
| workspace chip x | 502 | 502 | 0 |
| preset chip x | 655.33 | 658.67 | +3.34 |

All three re-pinned values are inside ±1 DIP (two of them exact), and the chip row kept its
inset/gap. Artifacts: `evidence/accept-D-MAIN-232/` (geom, controls, actual, accept-log).

## 4. Tests

- New `internal/ui/frame/hero_headline_height_test.go` guards the constant: it fails if
  `conversationHeroHeadlineFoot` drifts more than 0.1 from 22.32, and renders the headline row to prove
  the hero still lays out at 1440 wide (the bare tester does not expose the row key, so the rect
  assertion logs the slogan box; the guard is the real pin — the live UIA values above are the
  authoritative check and are recorded in the function comment).
- `go test ./internal/ui/frame ./internal/ui/chat ./internal/ui/composer` → **ok**
- `go vet` on the same three → clean.

## 5. Residual

1. Preset chip x is +3.34 (native 655.33): the gap between workspace and preset chips is ~3 DIP wide.
   Cosmetic and separate from the stack; one constant in the hero row.
2. Badge height 20 vs native 15.33 (box difference, ink is aligned).
3. Pixel parity still waits on the opaque-canvas work (task-105 on the board): the live window still
   composites the desktop, so the masked ratio cannot yet be read as a pass/fail.

## 6. State

Temp build stopped and deleted, temp profile removed, native restored to (200,2) 2268×1366 on the empty
新会话. No commit, no passN, no contract edit.
