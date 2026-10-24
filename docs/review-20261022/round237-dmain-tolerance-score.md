# Round 237 — definitive D-MAIN score on pass298, three ways, with what each proves

Honest headline: **the base delta the slice expected (9 levels) is not what this build renders.** The
measured surface still carries a painted gradient (hero ~43,53,49; bottom band 83,71,118; caption
36,95,123), so the tolerance scores do **not** isolate text — they stay wash-contaminated. The wash-mask
ratio (10.88 %) is the closest thing to a chrome measure and is reported as such.

## 1. HWND evidence (normal harness launch, pid 33196)

| HWND | class | visible | rect | centre sample |
| --- | --- | --- | --- | --- |
| 9372302 | `MyGoWindow` | yes | 2160×1380 | **(41,50,49)** |
| 13829054 | `MyGoCaption` (child) | yes | 2160×11 | — |
| 18678284 | `MyGoCaption` (child) | yes | 144×48 | — |
| 25822260 | **`MyGoSurface` (child)** | yes | 2160×1379 | paints the app surface |
| 10553290 | `MyGoApp` | **no** | 202×56 | (208,207,208) |
| 8784798 | `PseudoConsoleWindow` | yes | 0×0 | no paint |

So the painted surface is `MyGoWindow` → `MyGoSurface`; there is no second candidate HWND for the
fixture row (the hidden `MyGoApp` 202×56 is the tray/app helper). Capture of that window is the right
target — no HWND confusion this time.

## 2. Direct samples (client 1440×860 DIP)

| point | pass298 | native | Δ |
| --- | --- | --- | --- |
| hero | (43,53,49) | (12,12,12) | **+31…+41** |
| hero (2nd) | (46,115,72) | (12,12,12) | +34…+103 |
| card interior | (31,38,39) | (12,12,12) | +19…+27 |
| card interior (2nd) | (34,57,45) | (12,12,12) | +22…+45 |
| caption | (36,95,123) | (19,43,61) | +17…+62 |
| sidebar | (26,29,29) | (40,40,40) | −14…−11 |
| bottom band | (83,71,118) | (12,12,12) | +59…+106 |
| cyan band | **3/9** | 0/9 | — |

The gradient is not a uniform 9-level offset: it is a spatial wash from green-teal (upper) through
purple (lower), which is why every tolerance below stays dominated by it.

## 3. The three scores, clearly labelled

| score | value | what it proves |
| --- | --- | --- |
| threshold 0, whole frame | 2,786,047 / 2,786,400 = **99.99 %** | only that two renderers differ somewhere; unusable as parity |
| **Δ > 8**, whole frame | **98.64 %** | *fails to isolate text here*: the painted wash exceeds 8 levels over most of the frame (expected to isolate text only if the base delta were ≤8) |
| **Δ > 16**, whole frame | **90.40 %** | same, stronger cut; still wash-dominated |
| mean │Δ│ (whole frame) | **59.55** | magnitude of the systematic difference |
| **wash-mask ratio** | masked 89.11 % (2,482,850 px) → **unmasked mismatch 10.88 %** (30,197 px) | the closest available chrome measure: pixels that are not low-frequency wash still differ on ~1 in 9 |

Per-region (mean │Δ│ / Δ>8 / Δ>16):

| region | mean │Δ│ | Δ>8 | Δ>16 |
| --- | --- | --- | --- |
| sidebar | 71.73 | 93.60 % | 71.42 % |
| header | 37.34 | 98.84 % | 75.62 % |
| hero | 50.20 | 99.91 % | 94.62 % |
| composer | 58.17 | 100 % | 100 % |
| footer | 76.28 | 99.89 % | 99.78 % |

## 4. Structure (improved matcher)

| space | matched | within 1 DIP | outside | native-only | MyGo-only |
| --- | --- | --- | --- | --- | --- |
| window-relative | 23 | **12** (was 11) | 11 | 36 | 22 |
| caption-relative | 23 | **8** | 15 | 36 | 22 |

The extra within-tolerance pair is the hero block: round232's foot change put `探索未至之境`, the preset
row and the card top on their native y values, and the matcher now keeps them paired (scaffolding
excluded, composer nodes aliased by role).

Remaining clusters (window-relative, outside 1 DIP):

| control | delta (x, y, w, h) | owner / note |
| --- | --- | --- |
| 新建会话 (sidebar button) | y **+303.33**, w +4 | the native one is the *session-tree row*; different widget in a different place — data/tree, not chrome |
| 新建会话 (second pair) | y +54.00, w +83.33 | same name, two MyGo nodes (nav + hero) — needs an id rule |
| 标准模式 (preset chip) | x +3.34 | known cosmetic inter-chip gap |
| 发送消息 | y **−42.67** | send disc sits higher than native (card interior rows) |
| composer Edit (placeholder name) | x −12.67, w **+28.67** | textarea inset/width differs by ~29 DIP horizontally |
| conversation fish | x −7.33, w +14 | mark width |
| 视图选项 / 多选会话 / 收起 | ~0.7 | already inside tolerance, listed for completeness |

## 5. Residual

1. **The wash is the blocker**, and it is a painted gradient, not a 9-level composition placeholder:
   until the paint path stops it, no tolerance score can isolate text. The next product step is to log
   the paint decision (pass298 added a decision log — capture it next run) and disable the ambient layer
   for hero/dark/no-wallpaper.
2. Tolerance scoring is ready and reusable: `tmp/score-thresholds.py` emits Δ>0/8/16 + per-region +
   mean Δ for any row; once the wash is gone, Δ>8 becomes the text/AA/layout metric the contract wants.
3. Composer Edit width (+28.67) and send-disc y (−42.67) are real chrome deltas worth a slice after the
   wash.
4. Session-tree rows need their own id rule in the matcher (name collision 新建会话).

## 6. State

pass298 stopped, temp profile removed, native restored to (200,2) 2268×1366 on the empty 新会话. No
commit, no passN, no contract edit. Artifacts: `evidence/accept-D-MAIN-237/` (+`hwnd-mygowindow.png`,
`score-tolerance.json`). `fixture.md` updated with the round237 block.
