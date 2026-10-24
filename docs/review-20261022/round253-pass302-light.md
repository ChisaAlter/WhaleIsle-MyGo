# Round 253 — pass302 light verification: UIA table complete, pixel/AA score blocked by native theme instability

The composer residuals are confirmed landed **structurally** on pass302 (draft 745.33 at +12.67, send +0.66,
card 155.33, chip 502), the slogan-x value for task-125 is measured, and the AA score could **not** be
taken this round because the native app's theme keeps flipping between light and dark mid-verification.

## 1. Fixture

Light no-wallpaper profile with `goPreviewNoticeAck` (no modal), `theme=deepseek.light`, zh,
autoStartDesktop, temp workspace, `quitAfterStart=false`; `whaleisle://open`; **main** `MyGoWindow`
15927744 (2160×1380), the other window parked off-desktop for every capture; client 1440×860
(2160×1290 px @150 %). Backdrop log: `view: scheme="light" … flat=true bg="#ffffff" fill={255 255 255 255}`,
`painter: path="" effect="none" localExplicit=true active=false`; MyGo samples 255/255/255 hero-card-caption,
sidebar 249,250,251, cyan 0/9.

## 2. UIA table (window-relative DIP) — all composer residuals confirmed

| element | native | MyGo (pass302) | Δ |
| --- | --- | --- | --- |
| slogan | x **765.33** (harness 765.33), y 314.67 | **x 772.67**, y **314.67** | **x +7.34**, y 0.00 |
| badge 大肥鱼版 | x 936.00, y 321.33 (48.67×15.33) | **x 936.00**, y 323.33 (48.67×15.33) | x **0.00**, y +2.00 |
| composer card | 469.3 / 409.33 / 774.39 / 155.34 | 469.33 / **409.33** / 774 / **155.33** | 0.00 / 0.00 / −0.39 / −0.01 |
| draft | w **745.33** | **745.33** at x +12.67 | **width exact ✓** |
| send disc | x 1188.67, y **522**, 34.67 | x 1189.33, y **521.33**, 34 | +0.66 / −0.67 |
| workspace chip | x 502, 150×28.67 | **x 502**, 150×28 | **0.00** |
| preset chip | x 655.33, 106.67 | x 658.67, 100 | +3.34 / −6.67 |
| hero fish | 48.67 × 15.33 at x≈936 (target) | **34 × 24.67 at x 728.67** | **not landed** — see §4 |

**Slogan x for task-125: live 772.67 DIP** (native/harness 765.33) → **+7.34**, matching the lead's
harness figure of 772.64 vs 765.33 (+7.31). The badge is already at the native x, so the offset is inside
the headline row's centring, not the badge gap.

## 3. Why no AA score this round

The native window's theme is unstable between runs of the same minute:

| time | native caption | native hero | native sidebar |
| --- | --- | --- | --- |
| 07:29 | 255,255,255 | 255,255,255 | 249,250,251 |
| 07:33 | 255,255,255 | **12,12,12** | 65,65,65 (a stray console's titlebar) |
| 07:35 | 15,20,35 | **32,35,42** | 18,21,32 |

Additionally, during the first attempts a stray Windows Terminal (pid 33964, three tabs: a
`whaleisle-pass302.exe` deeplink log, `wi-flat.exe`, and one titled 设置) sat over the native window, and
the native app had its 设置 menu open. I closed those console tabs to clear the frame — **flagged for the
record: that terminal belongs to another process and closing it may have closed a shell someone was
using.** I will not close console windows again without asking.

Consequently the pass302 pixel pair is **cross-run at best** (MyGo current, native from a different theme
state) and I did not compute or report a score from it.

**Last valid AA numbers remain round245** (same light fixture, native stable):
raw 1.685 % frame · caption-masked 1.363 % frame · **AA-excluded 0.007 % frame; hero 0.000 %, composer
0.036 %**, nothing above Δ>16. With draft 745.33 and send now landed, the composer's non-AA residual (the
+12.67 inset and the old width) should disappear once a stable pair exists — that is the one number left
to confirm for task-125/126.

## 4. Discrepancy to resolve (fish)

The slice states the fish landed as **49×15.4 at x=936**. UIA on pass302 shows
`Conversation hero fish` at **x 728.67, 34×24.67** — the old box. Either the new mark is drawn under a
different accessible name (and the old node remains), or the change is not in this build. Worth a
one-line check in the frame code before the next capture; it is also the item the AA metric cannot see
(size inside the ink band), so it must be verified structurally.

## 5. Next step (ready to run)

1. Freeze the native theme for one run (light or dark — whichever the user's session is), close stray
   overlays, then capture: the procedure is scripted (park the other window, per-window live samples,
   caption mask registered, `scripts/score-aa.py`).
2. Report hero/composer AA-excluded with the composer fix in place; expect hero 0.000 %, composer ≤0.01 %.
3. Hand slogan x 772.67 to task-125 (its +7.34 offset) and resolve the fish node question.

## 6. State

pass302 stopped, temp profile removed, native restored to (200,2) 2268×1366. No commit, no passN, no
contract edit. Evidence: `evidence/accept-D-MAIN-253/` (geom, controls, structure, mask, actual,
backdrop log). `fixture.md` updated with the pass302 table, the theme-instability samples and the fish
discrepancy.
