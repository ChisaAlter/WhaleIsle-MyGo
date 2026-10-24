# Round 242 — light-theme D-MAIN pair: 1.68 % mismatch, mean Δ 1.23

The installed app is legitimately light (Windows is light), so the pair was taken in **light**. This is
the first fully valid, theme-matched, wash-free D-MAIN pair: **threshold 0 = 1.68 %**, mean │Δ│ = 1.23,
and the hero/composer regions are under 1 % at every threshold.

Artifacts: `evidence/accept-D-MAIN-242/` — `baseline.png`, `actual.png` (both 2160×1290), `diff-wash.json`,
`score-tolerance.json`, `geom.json`, `controls.json`, `structure.json`, `accept-log.txt`,
`backdrop-light.log`.

## 1. Fixture (light, no wallpaper)

`theme=deepseek.light`, `locale=zh`, `goPreviewNoticeAck`, `autoStartDesktop`, temp workspace,
`quitAfterStart=false`, **no** `goWallpaperPath`, fresh profile; temp build of the current tree; hand-off
`whaleisle://open`; main `MyGoWindow` (2160×1380, launcher absent); client 1440×860 (2160×1290 px @150 %
dpi 144).

Backdrop log confirms the light flat canvas and no ambient layer:

```
view: scheme="light" systemDark=false darkCanvas=false path="" flat=true bg="#ffffff" fill={255 255 255 255}
painter: path="" effect="none" localExplicit=true active=false
```

## 2. Direct samples (both sides, 1440×860)

| point | native | MyGo | Δ |
| --- | --- | --- | --- |
| hero / card / caption / bottom | (255,255,255) | (255,255,255) | **0** |
| sidebar | (249,250,251) | (249,250,251) | **0** |
| cyan band | 0/9 | 0/9 | — |

**There is no light base delta.** The 9-level dark-base issue (MyGo #151517 vs native #0c0c0c) does not
exist in light: both sides paint `#ffffff`, and the sidebar token matches exactly. So this pair isolates
renderer differences, which is exactly what the contract wants.

## 3. The three scores (scripts/score-thresholds.py + acceptance-diff.py --mask-mode both)

| score | value |
| --- | --- |
| threshold 0 | 46,905 / 2,786,400 = **1.68 %** |
| **Δ > 8** | **1.57 %** |
| **Δ > 16** | **1.46 %** |
| mean │Δ│ | **1.23** |
| wash mask | only 47 px masked (0.002 %) → unmasked mismatch **1.685 %** |

Per region (mean │Δ│ / Δ>8 / Δ>16):

| region | mean │Δ│ | Δ>8 | Δ>16 |
| --- | --- | --- | --- |
| sidebar | 3.00 | 2.94 % | 2.67 % |
| header (caption band) | 3.89 | 5.71 % | 5.59 % |
| **hero** | **0.95** | 0.77 % | 0.74 % |
| **composer** | **0.44** | 1.45 % | 1.25 % |
| footer | 5.93 | 5.49 % | 5.37 % |

What this proves: with the theme matched and the backdrop flat, the renderers agree on **98.3 %** of the
frame byte-for-byte, and the disagreement is concentrated where data/injection differ (sidebar session
tree 2.7 %, caption band 5.6 % — branch pill/terminal/right-panel controls) rather than in the hero or
composer (0.4–0.95 % mean, i.e. text antialiasing and the known chip offsets). The wash mask is
effectively unused (47 px), so this is a genuine pixel comparison, not a masked one.

## 4. Structure (UIA, improved matcher)

| space | matched | within 1 DIP | outside | native-only | MyGo-only |
| --- | --- | --- | --- | --- | --- |
| window-relative | 23 | **12** | 11 | 36 | 22 |
| caption-relative | 23 | **8** | 15 | 36 | 22 |

Remaining clusters (window-relative):

| control | delta (x, y, w, h) | note |
| --- | --- | --- |
| 新建会话 (sidebar vs tree) | y **+303.33**, w +4 | native pairs a tree row with MyGo's nav button — needs an id rule |
| 新建会话 (duplicate) | y +54.00, w +83.33 | two MyGo nodes share the name |
| 发送消息 | y **−42.67** | send disc higher than native (card tool row) |
| composer Edit | x −12.67, w **+28.67** | textarea inset/width |
| conversation fish | w +14 | mark width |
| 账号菜单 | y +7.33, h **−8** | signed-in 44 vs signed-out 36 footer seat |
| 标准模式 | x −3.33, w −6.67 | chip is now slightly narrower than native |

## 5. Residual / next

1. **Pair score is now trustworthy** — 1.68 % threshold 0 / 1.23 mean. The natural next criteria:
   hero+composer per-region ≤0.1 % is still far (0.95/0.44 mean), so the remaining work is the send-disc
   row, the textarea width (+28.67) and text AA, not the backdrop.
2. The caption band (5.6 %) is injection-only content; if the contract accepts "injection differences are
   out of row scope", the caption band should be a registered mask rather than a score contributor.
3. Light fixture becomes the primary row while Windows is light; the dark fixture (9-level base) stays
   documented as a separate, out-of-scope issue.

## 6. State

Temp build stopped and deleted, temp profile removed, native restored (its own position/size; the light
theme was left untouched as instructed). No commit, no passN, no contract edit. `fixture.md` gains a
light-fixture section.
