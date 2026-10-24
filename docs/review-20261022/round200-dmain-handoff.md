# Round 200 — D-MAIN chat hand-off on pass289: the gate, and why the row still is not capturable

Goal: reach the chat window from a seeded throwaway profile and score the seeded pair. **The pair was
again not produced**; what was produced is a precise map of the hand-off. No product code, no contract
edit, no commit, no passN; native restored, temp cleaned.

## 1. Measured facts

| # | fact | evidence |
| --- | --- | --- |
| 1 | `quitAfterStart` defaults true and kills the app after ~50 s | round192; with `false` a run stayed alive at 45 s and again at 60 s on pass289 |
| 2 | A plain launch opens **two** top-level `MyGoWindow`s: 2160×1380 (1440×920 DIP) and 1590×990 (1060×660 DIP) | enumeration by class: `12388822|21912|True|2160x1380`, `11537570|21912|True|1590x990` |
| 3 | The 2160-wide window renders the **launcher page** (首页/组件/版本/导入/插件排查/设置, 桌面端已就绪, 关闭桌面端, 线路 GitHub, catalog warning) | UIA text dump of the handle |
| 4 | Closing the launcher (`WM_CLOSE`) removes the 1590×990 window but the remaining window **still shows the launcher page** | post-close dump: one `MyGoWindow` left, same texts |
| 5 | `launcherFirstWindow = !isolated && !hidden && !background && !settings` | `cmd/whaleisle/main.go` |
| 6 | `WHALEISLE_ISOLATED=1` + absolute `WHALEISLE_USER_DATA` produced **one** window (2160×1296) that still showed the launcher page | isolated run, single `MyGoWindow 4655816`, same texts |
| 7 | A second instance does **not** hand over: it starts a complete second desktop (two more windows, new pid) | enumeration showed pids 21912 and 13608 each with their own windows |
| 8 | The hand-off routes exist: `applyLauncherStage(stage, false)` → `showMain()`; tray item `i18n.KeyCmdShowWindow` ('open' → `a.open`); deeplink `ActionDesktop` (`main.go` `case "desktop", "d-main"` clears all overlays) | code |

## 2. What this means for the capture

The chat window is not a window that exists hidden and can be activated — it is the **same window with
`s.launcherOpen = false`**. So the capture must first drive one of the hand-off routes (tray menu
"显示窗口" through UIA, or the app-menu action that calls `applyDeeplink(ActionDesktop)`), and only then
set 1440×860 and capture. Simply locating/activating handles cannot work, which is why the two attempts
this round (launcher close, isolated start) both captured launcher content.

`fixture.md` now records this (hand-off section, items 1–7 above) together with the earlier
`quitAfterStart` and `go-resume.json` facts.

## 3. Attempt artifacts (deliberately not promoted)

`%TEMP%\whaleisle-accept-2026-10-24\D-MAIN-289` holds baseline/actual/geom/controls/mask from the
2160×1380 launcher capture at 1440×860. It is **not** a D-MAIN pair (launcher page vs chat), so it was
not copied into `evidence/accept-D-MAIN` and no structure/caption-relative/masked-ratio numbers are
reported from it. The one observable there (raw 46.2 % mismatch) is meaningless for the row.

## 4. Residual and next slice (concrete)

1. Implement the hand-off in the capture helper: after start, find the tray icon (`Shell_TrayWnd` →
   `TrayNotifyWnd` → toolbar) or the app menu, invoke `显示窗口` / `desktop` route, wait for the same
   window handle to render the chat (assert `探索未至之境` or `Composer card` in UIA), then resize to
   1440×860 and capture. A UIA retry loop on the *existing* handle is the whole of it.
2. Keep: `quitAfterStart=false`, no `go-resume.json`, wallpaper crop seed, single-instance assert
   (kill any `whaleisle*` before start, re-enumerate after).
3. Then the round's real deliverables land: structure window/caption-relative counts, wash-masked ratio,
   card 774.39, caption triad 0, sidebar heading <1 DIP, chip left.

## 5. State

Both pass289 instances stopped, both temp profiles deleted, native window restored to (200,2)
2268×1366 and still on the empty `新会话`.
