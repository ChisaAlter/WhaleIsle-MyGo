# Round 177 — S27/S28 on the dark-matched pair

Inputs: `evidence/accept-D-MAIN` (native dark 1440×860 / MyGo pass283 dark, round175) and a fresh
after-fix capture from a temp build of this tree (`tmp\whaleisle-s2728.exe`, git `ab9984da`).
No product code beyond the two chat-chrome files below, no contract edit, no commit, no passN.

## 1. S28 (nav label baseline) — fixed

Native rows are a 36-DIP button with 7-DIP padding around a **22-DIP line box**; MyGo's label was a plain
18.7-DIP text box, so its centred ink sat ~1.7 DIP lower.

- `internal/ui/sidebar/sidebar.go`: the page-row label and the 鲸鱼娘 row label now carry
  `FixedLineHeight(22)`.
- Measured after the fix (headless, 1440 wide): `插件` label box **y=127, h=22**, `自动化任务` y=167,
  `鲸鱼娘` y=207 → pitch **40** and the label origin is exactly the native 127.
- Test: `internal/ui/frame/chat_chrome_metrics_test.go` now asserts the label box is 22 DIP tall in
  addition to x=46/y≈127 (it previously passed only because 18.7 also fell in the tolerance window).

## 2. S27 (composer width) — measured, still open at 1.06 DIP

`geom.json` of the matched pair gives the card as **773.33 DIP** on MyGo and **774.39 DIP** on the
native page (the number round156 measured in the DOM). The width comes from

```go
// internal/ui/frame/frame.go:867
composerOuterW := min(contentAxis+64, contentViewportW)
```

so `contentAxis` is 709.33 where the native page implies 710.39. `contentAxis` is also the transcript
axis, so changing it for the composer alone would move the message column; the honest fix is to
measure the native transcript axis at 1440×860 and align both from one number, or to give the hero
seat its own `min(774.39, viewport)` term. Not changed in this round: 1.06 DIP is at the contract's
1-DIP boundary (0.06 over) and it is a frame-level axis decision, not a composer padding bug. Recorded
as still open with the exact line.

## 3. Hero vertical seat — measured, not yet corrected

From the matched pair's `geom.json` (DIP, window-relative):

| element | native | MyGo | Δ |
| --- | --- | --- | --- |
| 探索未至之鲸 headline | y=314.67, h=35.33 | y=339.33, h=32 | **+24.7** |
| 大肥鱼版 badge | y=321.33, h=15.33 | y=345.33, h=20 | +24.0 |
| hero preset row (选择工作区 / 标准模式) | y=373.33, h=28.67 | y=387.33, h=28 | +14.0 |
| composer card top | ~409.3 | 423.33 | +14.0 |
| composer draft | y=417.33, h=52.67 | y=431.33, h=52 | +14.0 |
| send disc | y=522, h=34.67 | y=493.33, h=34 | **−28.7** |

Both stacks are centred in the column, so the offset is not a seat error: the **native stack is taller**
(send disc at 522 vs 493 means the native card is ≈155 DIP tall at this window, while MyGo keeps the
114-DIP recipe measured at 920 browser height). Two consequences:

- the headline sits 24.7 DIP higher on the native side because the taller card pushes the centred stack
  up;
- the card's internal recipe (well + gap + 34-DIP row) sums to 114 on MyGo but the native card at
  1440×860 is ~155, so the 114 number from the 920-high browser DOM does not hold at 860.

Correcting this needs the native **card's own rect and internal rows at 860** (a DOM probe at
1440×860, the same `--eval` path used in round152), because the recipe must explain 155 before the
stack offset can be attributed. That probe is the next slice; changing the hero offsets now would
just move the mismatch.

## 4. After-fix capture and per-region delta

Captured the fixed build dark at 1440×860 (both windows 2160×1290 exactly, no 1-px crop):
`evidence/screens/mygo-dmain-after-s2728-150dpi.png`. Per-region mean |Δ| (threshold-0 ratio and ratio
above 8 in brackets):

| region | round175 | after S27/S28 |
| --- | --- | --- |
| hero (raw) | 57.9 (1.00 / 0.99) | **57.2** (1.00 / 0.99) |
| composer (raw) | 37.7 (1.00 / 1.00) | **58.8** (1.00 / 1.00) |
| composer card (chrome-only) | 38.8 | 46.7 |
| headline + badge (chrome-only) | 100.9 | 106.6 |

The numbers do not track the two fixes, and the composer got worse: **the wallpaper still dominates**
(native user wallpaper vs MyGo's default gradient, with MyGo's translucent mica letting it through the
card), and the two captures were taken minutes apart with the native app on a different session/scroll
state. Pixel metrics are therefore not a usable signal for 1–2 DIP text/width deltas until the
wallpaper is masked or seeded (round175 §4 proposal). The structural numbers above are the reliable
evidence for this round.

## 5. Gates and state

`go test ./internal/ui/composer ./internal/ui/chat ./internal/ui/frame` → ok;
`go vet ./internal/ui/composer ./internal/ui/chat ./internal/ui/frame` → clean.
Temp build process stopped, temp profile deleted, native window restored (2268×1366 at (200,2)) and
left on the empty `新会话`.
