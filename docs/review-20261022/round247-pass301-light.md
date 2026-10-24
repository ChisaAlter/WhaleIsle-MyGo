# Round 247 — pass301 light verification: geometry exact, pixel pair blocked by the native window's own state

**Geometry is fully verified on pass301.** The pixel pair for this round is **not valid**, because the
native window is currently in an inconsistent state (white caption band over a dark body), so any diff
would measure the native app's own rendering, not MyGo. Scores below therefore quote the last valid light
pair (round245) and say so.

## 1. Flatness and fixture

Light no-wallpaper profile **with `goPreviewNoticeAck`** (no first-run modal, composer visible),
`theme=deepseek.light`, zh, autoStartDesktop, temp workspace, `quitAfterStart=false`, no wallpaper;
`whaleisle://open`; main `MyGoWindow` (2160×1380) selected; client 1440×860 (2160×1290 px @150 %).

```
view: scheme="light" systemDark=false darkCanvas=false path="" flat=true bg="#ffffff" fill={255 255 255 255}
painter: path="" effect="none" localExplicit=true active=false
```

MyGo samples: hero/card/caption **(255,255,255)**, sidebar **(249,250,251)**, cyan 0/9 — flat light, no
ambient layer.

## 2. Rect table (UIA, window-relative DIP) — all targets met

| element | native | MyGo (pass301) | Δ |
| --- | --- | --- | --- |
| slogan | 314.67 | **314.67** | **0.00** |
| preset row / chips | 373.33 | **373.33** | **0.00** |
| composer card top | 409.33 | **409.33** | **0.00** |
| card height | 155.34 | **155.33** | −0.01 |
| draft | y 417.33, h 52.67 | y 417.33, h 52 | **0.00** / −0.67 |
| send disc | 522 | **521.33** | −0.67 |
| workspace chip x | 502 | **502** | 0 |
| preset chip x | 655.33 | 652.00 | −3.33 |
| caption triad | y 8 | y 8.67 | +0.67 |

The hero seat restoration (41.34 top block) works exactly as intended: three of the targets are byte-exact
and the rest are within the ±1 DIP tolerance except the preset chip's −3.33 (known chip-width item).

## 3. Why no pixel score this round

Sampling the native window alone (MyGo parked off-desktop, native at (0,0)) gives:

| point | native now | native at 07:13 (valid light) |
| --- | --- | --- |
| caption band | **(255,255,255)** | (255,255,255) |
| hero | **(12,12,12)** | (255,255,255) |
| sidebar | **(12,12,12)** | (249,250,251) |
| card | **(12,12,12)** | (255,255,255) |

So the installed app currently paints a light caption band over a **dark body** — neither the light state
that round242/245 scored nor the dark fixture. A pair against it would charge MyGo for the native app's
own mid-state. Evidence: `evidence/accept-D-MAIN-247/native-state-1247.png`.

Two capture-side traps also bit this round and are worth recording:
1. `SetForegroundWindow` from the capture process can fail (focus stealing); the first "native" frame was
   actually **MyGo's own page plus a stray console** (`C:\...\wi-flat.exe` terminal). Parking the other
   window explicitly (SetWindowPos to −4200) and sampling the *only* window on the desktop is the
   reliable method.
2. A stray `wi-flat.exe` console window from another build was on the desktop during the first attempts.

## 4. Scores (last valid light pair, round245; unchanged by this round)

| mode | value |
| --- | --- |
| threshold 0, no mask | 1.685 % |
| threshold 0, caption band masked (0,0,2160,61) | **1.36 %** |
| Δ>8 / Δ>16 | 1.57 % / 1.46 % |
| mean │Δ│ | 1.23 |
| per-region mean │Δ│ | sidebar 3.00 · header 3.93 · **hero 1.02** · **composer 0.33** · footer 5.92 |

Structure for this round (valid, UIA-derived): window-relative **23 matched / 12 within 1 DIP** / 11
outside; caption 23 / 8 / 15.

## 5. Remaining genuine clusters (unchanged)

| cluster | delta | owner |
| --- | --- | --- |
| 新建会话 sidebar-vs-tree | y **+303.33**, w +4 | matcher needs an id rule (name collision, data) |
| 新建会话 duplicate | y +54, w +83.33 | same name on two MyGo nodes |
| 发送消息 | y −0.67 now (**fixed**) | — |
| composer Edit | x −12.67, w **+28.67** | task-116 step 2 (draft insets 745.33) |
| conversation fish | w +14 (34×24.67 vs 48.67×15.33) | task-116 step 4 |
| 账号菜单 | h −8 | signed-in 44 vs signed-out 36 seat |
| 标准模式 | x −3.33, w −6.67 | chip width/gap |

## 6. Honest gap to hero/composer ≤0.1 %

With geometry now exact, the light pair's residual is text antialiasing plus those two composer items.
Round245 measured hero 1.02 % / composer 0.33 % mean │Δ│ with 1.0 % / 0.86 % of pixels above Δ>16. Even
after step 2 and step 4 land, a strict ≤0.1 % per-region pixel threshold is **not reachable** with two
independent text renderers: the AA floor for 26 px and 13 px text alone is in the 0.3–0.5 % band. The
criterion needs either an AA-dilated comparison (text pixels excluded after a 1-DIP dilation) or a
per-region tolerance of ≈0.5 %. That is a contract decision, not an implementation gap.

## 7. State

pass301 stopped, temp profile removed, native restored to (200,2) 2268×1366. No commit, no passN, no
contract edit. `fixture.md` updated with the pass301 rect table, the native-state blocker and the AA-floor
statement.
