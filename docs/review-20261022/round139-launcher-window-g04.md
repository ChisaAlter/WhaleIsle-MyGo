# Round 139 — launcher as its own native window (G04 phase 1)

Date: 2026-10-24. Task: task-12. Plan: `docs/parity/launcher-boot-plan-20261024.md` §3.1–3.3.
Scope used: `cmd/whaleisle/` (main.go launch block, new `launcher_window.go`, tests, welcome hand-over, quit cleanup) + this doc. No commit.

## 1. Reference behaviour (read-only evidence)

- `reference/electron/src/main/window.js:900-937` `createLauncherWindow`: `shellWindowChrome({width:1060,height:660,minWidth:860,minHeight:560, show:false})`, `markWindowTransparent`, `attachIntegratedChrome({role:'launcher'})`, `hideNativeMenu` on `ready-to-show`, `launcherWindow.on('closed', … = null)`.
- `window.js:939-952` `showLauncher`: reuse the window if it exists, else load `launcher.html`; then `restore()` if minimized → `show()` → `focus()`.
- `index.js:1364-1366`: `prepareLauncher()` creates it **hidden** before the cold-start gate; `index.js:340-348` `openLauncher()` shows it for tray/deep link.
- `index.js:313-336` `bindLauncherClose`: closing the launcher closes it plainly **when a main window exists**; with no main window it `preventDefault()` + `quitApp()`. Cold start therefore never creates the main window while the launcher has focus; `HarnessController` creates it at start (`index.js:597`).
- `index.js:486-497`: on a confirmed healthy start (`shouldCloseLauncherAfterDesktopStart`) the launcher is closed; `lastStartFailed`/import holds/`quitAfterStart:false` keep it.
- `launcher.css:21-42`: 48 DIP caption strip, 32×32 buttons, radius 8.

## 2. Implementation

`cmd/whaleisle/launcher_window.go` (new):

- `launcherWindowOptions(cfg)` — `Width/Height 1060×660`, `MinWidth/MinHeight 860×560`, `Hidden:true`, `Transparent:true`, `TitleBarHidden`, `TitleBarHeight 48`, `AutoHideMenuBar`, `WindowsCaptionButtons{32,32,Top 12,Right 8,Corner 8,Glyph 12}`, `WindowsFrameCornerRadius 20`, `StateKey "launcher"` (its own geometry, never the main window's).
- `launcherFirstWindow(isolated, hidden, background, settings)` — cold-start gate: only a foreground cold start hands the window to the launcher.
- `launcherStage` interface + `applyLauncherStage(stage, open)` — the one transition table (open reuses the single window; a run with no launcher window keeps the in-window panel on the main window; close destroys the launcher window and reveals the main window once; a quitting shell never reveals main).
- `syncLauncherStage()` — the single entry: creates/reuses/shows, or closes + reveals.
- `ensureLauncherWindow` / `closeLauncherWindow` / `launcherWindowCloseRequested` (`bindLauncherClose`): one window maximum; user close with a visible main window just closes, otherwise it goes through `requestQuitProtected`.
- `launcherPanel(c, win, host)` — **one** model: the whole former `mainView` launcher block (model, callbacks, component activity, dialogs, import/update/prefs) moved here verbatim; both surfaces call it. `launcherPanelHost` carries the render snapshot (`ThemeID/Version/Status/Logs/Workspace/Supervisor/Sessions`) so the launcher window reads the same values as `mainView`.
- `launcherWindowBody` paints the panel full-window (transparent window, panel's 20 radius shell = silhouette) — no main chrome.

`cmd/whaleisle/main.go`:

- launch block: `launcherFirst` hides the main window only on that gate (`Hidden: hidden||background||launcherFirst`) and stores `launcherWindowAllowed` with `state.win`.
- `mainView`: the launcher branch is now `state.launcherPanel(c, win, launcherPanelHost{…})` (fallback panel), the ~460 moved lines deleted; the now-unused per-render launcher locals and the `desktopupdate` import removed (it moved with the block).
- `beginLauncherBoot`: syncs the stage right after setting `launcherOpen` (launcher.js order: launcher first).
- `refreshBoot`: reveals the main window when `Starting` (native creates it at start, boot page behind the launcher) and syncs the stage after every status change.
- `applyDeeplink`: `ActionLauncher` → `syncLauncherStage()` (raises/reuses the launcher window, panel fallback still shows main); every other route closes the launcher window and shows main.
- accept-open QA hook (`pollAcceptOpenHook` watcher) syncs after a route.

Other files: `welcome_window.go` `showParent` does not reveal the main window when the launcher owns its own window (welcome → launcher window, then start); `quit_lifecycle.go` `finalizeQuit` destroys the launcher window with the other independent windows.

Fallback kept everywhere: isolated/QA desktop, login start, `--background`, `--settings`, no native window, failed creation (`launcherWindowAllowed=false` or `ensureLauncherWindow` returns nil → `applyLauncherStage` shows the main window with the panel; headless tests never call `mygo.NewWindow`).

## 3. Tests

`cmd/whaleisle/launcher_window_test.go` (all pass):

- `TestLauncherFirstWindowGatesColdStart` — cold start true; isolated / login / `--background` / `--settings` false.
- `TestLauncherWindowOptionsMatchNative` — 1060×660, min 860×560, hidden, transparent, hidden title bar, 48 caption, auto-hide menu, `launcher` state key, 32×32 top 12 right 8 radius 8 glyph 12, radius 20.
- `TestLauncherStageColdStartKeepsOneLauncher` — one creation, repeated syncs create nothing, main stays hidden.
- `TestLauncherStageStartMovesToMainOnce` — close launcher + show main exactly once; a late refresh is a no-op. Event order `show-launcher → close-launcher → show-main`.
- `TestLauncherStageKeepsLauncherOnCancelAndError` — cancel/failure keeps the launcher window.
- `TestLauncherStageFallsBackToPanelWindow` — no launcher window → main shown, nothing created.
- `TestLauncherStageQuittingNeverRevealsMain`.

Commands: `go test ./cmd/whaleisle -count=1` → ok (19.5 s, whole package green); `go vet ./cmd/whaleisle` → clean; `go build ./cmd/whaleisle` → ok.
Module note: the repo `.toolchain` module cache was missing some `.mod` files and the local proxy (127.0.0.1:10808) refused connections; `go build` needed `golang.org/x/net@v0.34.0.mod` (copied from the user cache) before it resolved offline.

## 4. Smoke evidence (real windows, temp profile)

Method: temp binary + temp profile under `.devdata\launcher-smoke` (deleted afterwards; `WHALEISLE_USER_DATA` there, `welcome-shown` marker, **no workspace** so boot cannot start the harness), `EnumWindows` + `GetWindowThreadProcessId` + `GetWindowRect` + `IsWindowVisible` + `GetDpiForWindow` from a per-monitor-v2 process (physical px). **No PrintWindow.** DPI 144 (150%), so 1 DIP = 1.5 px.

Cold start (launcher stage):

| hwnd | visible | rect (px) | size (px) | size (DIP) | class | title |
| --- | --- | --- | --- | --- | --- | --- |
| 8061450 | true | 485,189,2075,1179 | 1590×990 | **1060×660** | MyGoWindow | Whale Isle |
| 1444326 | false | 200,0,2360,1380 | 2160×1380 | 1440×920 | MyGoWindow | Whale Isle |

Re-raise through the launcher route (tray/deep link path): same hwnd `8061450`, same 1590×990 rect → **no second launcher window**.

Leave the stage (`desktop` route, the same code path as a successful start's hand-off): hwnd `8061450` is gone and `1444326` is visible at 2160×1380 (1440×920) → launcher destroyed, main revealed once.

## 5. Risks and open items

- **Liveness after the reveal is unproven in this profile.** With no workspace the harness never starts, so after the launcher is destroyed the main window paints the desktop shell for the first time in that process; in this environment the process then stops executing posted main-thread tasks (`Get-Process.Responding` false, later accept-open routes are consumed but never applied). The same `Responding=false` appears for the **pre-existing** `--settings` main-window path in the same no-workspace profile, so it is not introduced by this change, but the real `start → boot → ready` shape (harness running) is only covered by the unit tests here. The Lead's pass273 run with a workspace should confirm the hand-off and the boot page.
- Native parity gap: the reference does not create the main window at all during the launcher stage; MyGo pre-creates it hidden (single window object owns all shell state). Consequence: `--background`/login/isolated paths keep their existing behaviour and the fallback panel.
- Closing the launcher window at cold start routes through `requestQuitProtected` (MyGo task-guard) instead of native's bare `quitApp`; at cold start there is no runtime, so no prompt appears, but the main window shows the closing layer before exit. Documented rather than special-cased.
- The launcher window uses `cfg.Workspace` (no selected-session cwd) for its workspace seed; the in-window panel keeps the session-aware root. A cold start has no session, so the two agree.
- `launcherPanel` now reads the launcher route flags from `state` under the lock instead of `mainView` locals; `desktop_tray`/`main_view` render paths are otherwise unchanged.
- Geometry before the change (in-window panel) was 1440×920 for the launcher; the launcher is now 1060×660 DIP as in the reference. `internal/ui/launcher` was not touched; its panel is fluid and the smoke painted it at 1060×660 without visible clipping (no pixels captured — PrintWindow hangs WebView2).
