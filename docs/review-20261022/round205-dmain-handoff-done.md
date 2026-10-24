# Round 205 — D-MAIN hand-off cracked: `whaleisle://open`

The chat window is reachable, and the working command is now in `fixture.md`. The pair was captured at
1440×860 DIP, though the native side of this particular capture lost its wallpaper, so the **pixel**
numbers are weak; the **geometry** numbers below are valid.

## 1. The hand-off

| attempt | result |
| --- | --- |
| plain launch (pass289/290) | two `MyGoWindow`s (2160×1380 and 1590×990), the wide one renders the **launcher page** |
| `WM_CLOSE` on the launcher window | window removed, remaining window still launcher |
| `WHALEISLE_ISOLATED=1` | one window, still launcher |
| second instance | a whole second desktop, no hand-over |
| **`Start-Process "whaleisle://open"`** | **works** — 6 s later the same handle renders the chat (`探索未至之境`, `Composer hero preset row`, `Composer card`, caption triad, sidebar tree) |

`parse.go:60` maps host `""`/`"open"` to `ActionOpen`; the running instance applies it, clears the
overlays and flips `launcherOpen`, so the already-existing window repaints as the desktop. No hidden
window has to be found or activated.

## 2. Capture (pass290, git `0157ad1c`, 1440×860 DIP = 2160×1290 px @150 %)

- `evidence/accept-D-MAIN-290/`: `baseline.png`, `actual.png`, `geom.json`, `controls.json`,
  `structure.json`, `mask.json`, `accept-log.txt`.
- Native pid 15560, unchanged window; one MyGo instance (`11564`), one `MyGoWindow` handle `9896176`.
- Raw pixel diff: **49.54 %** mismatch, channel max Δ 250. Per-region: sidebar 0.81, header 1.00,
  hero 0.57, composer **0.095**, footer 0.99.
- Caveat (read this before quoting the numbers): in this capture the native window painted a **flat
  dark surface** (hero samples `rgb(12,12,12)`, sidebar `rgb(40,40,40)` — no wallpaper, no aurora), while
  MyGo painted its seeded crop. Earlier rounds' native captures had the wallpaper (`rgb(17,36,45)`), so
  the native side lost its backdrop during the page switch. The pair therefore measures wallpaper
  absence, not chrome parity; only the composer region (0.095) is close to meaningful.

## 3. Geometry (valid regardless of the wallpaper)

| item | native (DIP) | MyGo (DIP) | verdict |
| --- | --- | --- | --- |
| caption triad (最小化/最大化/关闭) | x = 1336 / 1368 / 1400, y = 8, 32×32 | x = 1336 / 1368 / 1400, y = 8.67, 32×32 | **x exact, y +0.67** (F1 confirmed) |
| sidebar nav rows 插件/自动化任务/鲸鱼娘 | y = 120 / 160 / 200, 252×36 | y = 120.67 / 160.67 / 200.67, 252×36 | **+0.67 DIP**, pitch 40 (F2 confirmed) |
| 标准模式 chip | x = 638.3 (left flow, after the workspace chip) | x = **610.67** (left flow, 4 DIP after the workspace chip at 490+116.67) | **left-aligned** (round181/186 fix holds; the absolute x differs only because the workspace chip is 116.67 wide vs the native 150) |
| composer card | 774.39 × 114 at (469.3, 430) | **773.33** × 114 at (470, 429.33) | size **−1.06 DIP** (still the transcript-preference axis), y −0.67 |
| 探索未至之境 | y = 321.33 h 15.33 (badge) / headline row 337 | y = 332.67 h 32 (slogan), badge 338.67 | +11.7 on the slogan box (headline row measurement differs between the two dumps) |

Structure counts on this pair: `absolute` matched 22, **within 1 DIP 0**, outside 22, native-only 37,
MyGo-only 27; the caption-relative section came back empty in this run of the script (the registration
block reports `caption_offset_dip: 0.67`, computed from the triads above). The `0/22` is dominated by
the hero block, whose native numbers in this dump are inconsistent (the native page was mid-switch when
its UIA tree was walked), so I am **not** treating it as a regression signal.

## 4. Still open

1. The axis: card is 773.33 though the profile had no `go-resume.json`, so something else supplies a
   preference (~741.33). Next: log `Sizes.TranscriptWidth`/`contentAxis` at startup, or set
   `TranscriptWidth` explicitly in the config if the field exists, before the next capture.
2. The native paint: re-capture with a forced native repaint (foreground + 3 s did not restore the
   wallpaper) — sequence the native capture *first*, before the MyGo window ever occupies (0,0), or
   park MyGo off-screen from the start.
3. Caption-relative rescore: re-run `acceptance-structure.py` on a pair taken while both sides are
   settled, and report that number instead of the absolute one.

## 5. State

pass290 stopped, temp profile deleted, native window restored to (200,2) 2268×1366 and left on the
empty `新会话`.
