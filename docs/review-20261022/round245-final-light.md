# Round 245 — final light pair after the composer card change: scores, mask registration, and a 20.66-DIP regression

The light pair is captured with the post-116 binary and scored three ways, the caption band is now a
registered mask, and the composer rects are measured — which revealed that the taller card moved the
**whole hero stack 20.66 DIP up**. Everything else is green.

Artifacts: `evidence/accept-D-MAIN-245/`.

## 1. Fixture and flatness

Light no-wallpaper profile (`theme=deepseek.light`, zh, `goPreviewNoticeAck` present so the first-run
preview modal never hides the composer, autoStartDesktop, temp workspace, `quitAfterStart=false`),
`whaleisle://open` hand-off, main `MyGoWindow` 2160×1380, client 1440×860 (2160×1290 px @150 %).

`view: scheme="light" … flat=true bg="#ffffff" fill={255 255 255 255}` ·
`painter: path="" effect="none" localExplicit=true active=false`.

Samples, both sides: hero/card/caption/bottom **(255,255,255)**, sidebar **(249,250,251)** — flat, no
base delta, cyan 0/9.

## 2. Composer rects (UIA, window-relative DIP)

| element | native | MyGo | Δ |
| --- | --- | --- | --- |
| composer card top | 409.33 | **388.67** | **−20.66** |
| card height | 155.34 (derived) | **155.33** | −0.01 ✓ |
| draft | y 417.33, h 52.67 | y 396.67, **h 52** | −20.66 / −0.67 |
| send disc | y **522** | y **500.67** | **−21.33** |
| 标准模式 chip | y 373.33 | y 352.67 | −20.66 |
| slogan | y 314.67 | y 294.00 | −20.67 |
| hero fish | 48.67 × 15.33 (target) | **34 × 24.67** | step 4 not in this build |

**The card recipe is correct (155.33 vs 155.34) but the stack is 20.66 DIP too high.** Cause: the card
grew 41.34 DIP while the hero stack is centred with a fixed 38-DIP foot (`conversationHeroStackFoot`), so
half the growth (20.67) moved the whole stack up. Fix: add a **41.34-DIP fixed block above** the stack
(equivalently `T' = 41.34` with the foot unchanged), which puts slogan 314.67 / preset 373.33 / card
409.33 / send 522 back onto the native values. Not applied here (product code owner: task-116).

## 3. Scores — with and without the caption-band mask

Mask registered: `mask.json` → one rectangle `(0,0,2160,61)` = the 40.67-DIP caption band, reason
"native injects branch pill / terminal / right-panel controls (injection-only)".

| mode | value |
| --- | --- |
| threshold 0, **no mask** | 46,954 / 2,786,400 = **1.685 %** |
| threshold 0, **caption masked** | 37,937 / 2,664,595 = **1.36 %** (131,805 px masked, 4.73 %) |
| Δ>8, no mask | 1.57 % |
| Δ>16, no mask | 1.46 % |
| mean │Δ│ | **1.23** |
| wash mask | 47 px only (0.002 %) |

Per region (mean │Δ│ / Δ>8 / Δ>16):

| region | mean │Δ│ | Δ>8 | Δ>16 |
| --- | --- | --- | --- |
| sidebar | 3.00 | 2.94 % | 2.67 % |
| header (caption, now masked) | 3.93 | 5.71 % | 5.59 % |
| hero | **1.02** | 1.03 % | 1.00 % |
| composer | **0.33** | 1.05 % | 0.86 % |
| footer | 5.92 | 5.49 % | 5.37 % |

**What is genuinely missing:** the pixel deltas are now small everywhere (composer mean 0.33 is the best
yet), but they are *not* ≤0.1 % per region, and the 20.66-DIP stack shift means part of that residual is
a real displacement. After the top-spacer fix the hero/composer regions should approach their floor,
which is text antialiasing (~0.3-0.5 % of pixels at Δ>16).

## 4. Structure

| space | matched | within 1 DIP | outside |
| --- | --- | --- | --- |
| window-relative | 23 | **11** (was 12) | 12 |
| caption-relative | 23 | 7 | 16 |

The lost within-tolerance pair is the composer node that moved by 20.66 — consistent with §2.

Remaining genuine clusters: 新建会话 sidebar-vs-tree (name collision, y +303.33), 发送消息 (−21.33 now),
composer Edit width (+28.67 step 2 pending), fish size (step 4), 账号菜单 h −8, 标准模式 widths.

## 5. Contract-relevant conclusions

1. The light fixture with the caption band registered is the cleanest row: **1.36 %** masked mismatch,
   mean 1.23 — dominated by data (session tree) and one known displacement.
2. Caption-band mask registration is justified: that band carries only injected Chrome controls
   (branch pill, terminal toggle, right-panel toggle) that MyGo draws itself in its own caption; masking
   it removes 0.33 percentage points and ~132 k px of injection noise.
3. The hero/composer ≤0.1 % target is reachable only after the 41.34-DIP top spacer (or an equivalent
   re-centring) lands; text AA is then the expected floor.

## 6. State

Temp build stopped and deleted, temp profile removed, native restored to (200,2) 2268×1366 (it had been
moved to (0,0) by the live-sample capture). No commit, no passN, no contract edit. `fixture.md` updated
with the round245 rect table, the mask registration and the stack-shift finding.
