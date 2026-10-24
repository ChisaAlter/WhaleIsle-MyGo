# Round193 — S03 on a real Host page, and the pet devtools decision

Date: 2026-10-25. Read-only native app; no commit, no `passN`. Temporary S03 instrumentation was added to
`internal/desktop/webview/devtools_shortcut.go` (+ a temp `g71_debug.go`) and **reverted**: `git status
--porcelain internal/desktop/webview` is empty and `g71log` no longer appears in the tree. A throwaway probe
app lived in `tmp/hostprobe/` and was deleted.

---

## A) Pressing Ctrl+Shift+I on a real Host page — settled

### A.1 Why a probe app instead of the shipped window

The premise "the main window is a `webview.Host`" does **not** hold: the main window is a Go-drawn
`mygo.Window` whose content is `ui.View(mainView)` (`cmd/whaleisle/main.go:2223-2242`), so it has no page and
no chord. `Host` pages are the *child* windows: plugin/browser panel (`main.go:1346`), document viewer
(`document_viewer.go:25`), whalebridge (`whalebridge_window.go:125`), account platform
(`account_platform.go:86`). The chord is installed only inside `Host` (`webview/host.go:287`, `:444`).

Opening one of those panels in a fresh profile is not reproducible in the timebox (the `browser`/`webview`
accept-open hooks need a persisted panel URL, `main.go:1589-1592` + `RestorePanelURL`), so the press test used
a minimal probe that goes through the **same** Host constructor:

```
mygo.NewWindow(WindowOptions{Title:"HostProbe", 900x640, Page: PageOptions{DevTools: DevToolsEnabled}})
desktopwebview.NewHost(HostOptions{Parent: win, URL: "http://127.0.0.1:3999/probe", Title: "HostProbe page",
                                   Untrusted: true, OnChange: func(){ win.Update(func(){}) }})
```

`WL_HOSTPROBE_GATE=deny` calls `webview.SetDevToolsShortcutAllowed(func() bool { return false })` before the
Host is created, so the two runs differ only in the gate.

### A.2 Results

Instrumentation wrote one line when the installer ran (`install ok` / `install skipped`), and one line per
`OnBeforeInput` event. Keys: `keybd_event` Ctrl↓ Shift↓ I↓ I↑ Shift↑ Ctrl↑, 6 s settle, `EnumWindows` diff.

| Run | Gate | Installer log | DevTools after the press | Go `OnBeforeInput` lines |
| --- | --- | --- | --- | --- |
| allow | open | `install ok packaged=false allowed=true` | **opened**: `DevTools - about.blank`, class `Chrome_WidgetWin_1` | **none** |
| deny | closed | `install skipped h=true pg=true allowed=false` | **opened anyway**: same title/class | n/a (not installed) |
| allow (second attempt) | open | `install ok …` | none — `GetForegroundWindow() != HostProbe`, so the keys never reached the page | n/a |

Screenshots: `evidence/round193/hostprobe-devtools-allow.png` and `hostprobe-devtools-deny.png` both show the
Microsoft Edge DevTools welcome page ("欢迎使用 Microsoft Edge DevTools"), captured from the window's rect.
`hostprobe-before-allow.png` is the probe window before the press (no inspector).

### A.3 What this settles

1. **Synthetic `keybd_event` keys DO reach the WebView2 on a Host page** and open the inspector. Round188's
   "unproven" is now answered: the delivery works when the target window actually owns the foreground.
2. **The mechanism is WebView2's built-in accelerator, not MyGo's chord.** In the successful allow run the
   installer logged `install ok`, yet the Go hook logged **zero** input events for the same chord — Chromium
   consumed it first and opened DevTools itself. MyGo's `OnBeforeInput` handler is therefore redundant on a
   page whose `AreDevToolsEnabled` is true, and would be dead code if it were the only entrypoint.
3. **The Go gate does not gate the inspector.** With `SetDevToolsShortcutAllowed(false)` the installer is
   skipped (`install skipped … allowed=false`) and DevTools still opens, because Host pages do not set
   `PageOptions.DevTools`, whose zero value is `DevToolsAuto` (`third_party/mygo/window.go:40-45`:
   "enables the inspector unless the app was built for production with `mygo build`"). In a normal `go build`
   the WebView2 `AreDevToolsEnabled` flag is on, so the config flag `openDevTools`
   (`cmd/whaleisle/main.go:2265-2266`) only removes our handler — it does not remove the accelerator.
4. **The main window can never show this chord**, which is exactly why round188's pet-only press produced no
   log: neither window was a Host.

### A.4 Recommended (not implemented) gate fix

Make the *page* follow the same gate instead of the callback alone: pass
`PageOptions.DevTools: DevToolsEnabled|DevToolsDisabled` into the Host's child window from
`DevToolsShortcutAllowed()`, or add a runtime `ICoreWebView2Settings::put_AreDevToolsEnabled` toggle bound to
the config. Without that, a packaged build that sets `openDevTools=false` still exposes Ctrl+Shift+I through
WebView2 unless the production `mygo build` DevToolsAuto behaviour happens to disable it — a security-relevant
difference from the native app, which does gate `webContents.openDevTools`. Recorded here; changing Host page
creation is a product decision, not a measurement.

### A.5 Automation precondition (for future press tests)

`SetForegroundWindow` is silently refused for a newly shown window in this session: the third run reported
`foregroundIsProbe=False` and produced no inspector. A press test must assert
`GetForegroundWindow() == target` before sending keys (or attach to the foreground thread first); otherwise a
"no window" result is a focus failure, not a wiring failure. This is the trap round188 fell into.

---

## B) Pet devtools — decision: deliberate gap, no code

Facts:

- Native attaches the chord **per `webContents`** (`devtools-shortcut.js`), so the pet window has devtools
  natively.
- MyGo's pet is a plain `mygo.Window`, not a `Host`, created with
  `Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}`
  (`internal/desktop/pet/companion.go:195`). `installDevToolsShortcut` therefore never runs for it, and the
  WebView2 accelerator is off on that page as well — the pet has no inspector by either route.
- Technically MyGo *can* open devtools on that window (`mygo.Page.OpenDevTools` /
  `third_party/mygo/internal/windows/browser_devtools.go:18-26`, and `PageOptions.DevTools` at window
  creation).

Why this is not "small, behind the same gating":

1. The effective entrypoint in this build is the WebView2 accelerator, and the Go chord cannot gate it (§A.3).
   Adding the chord to the pet would add a second handler while the real switch (`AreDevToolsEnabled`) stays
   whatever the page option said — i.e. cosmetic parity, not the gate the task asked for.
2. `PageOptions.DevTools` is a **static** page option evaluated at creation, while the product gate is
   `!IsPackaged() || cfgStore.OpenDevTools` (a runtime function). Enabling devtools on the pet page
   unconditionally would give a packaged build with `openDevTools=false` an inspector it must not have; the
   same problem as A.4.
3. Doing it correctly needs the runtime `AreDevToolsEnabled` toggle from A.4 first, then one line on the pet
   page (same option) plus the chord for parity of behaviour. That is a two-step product change with a
   packaging decision, not a one-file patch.

Decision: **documented gap, no half-wiring.** The pet remains diagnostics-exempt in MyGo; the correct fix is
the shared gate from A.4, applied to every page (Host and pet) in one change. No tests were added and no
gates run because nothing in the product changed.

---

## C) Files and state

- Evidence: `docs/review-20261022/evidence/round193/` — `hostprobe-before-allow.png`,
  `hostprobe-devtools-allow.png`, `hostprobe-devtools-deny.png`, `hostprobe-report-allow.txt`,
  `hostprobe-report-deny.txt`, `hostprobe-log-allow.log`.
- Instrumentation reverted (`git status --porcelain internal/desktop/webview` empty), probe app, exe, scripts
  and `%TEMP%\wl-g71*` profiles removed, probe processes stopped.
- Native app: untouched (no window was moved or closed by this slice).

## D) Residuals

1. **Packaged-mode gating is not proven**: the deny run itself opened DevTools (A.2). A packaged artifact with
   `openDevTools=false` was not built or run; the gate's real effect there depends on `DevToolsAuto` under
   `mygo build`, which should be confirmed before claiming S03 gating is correct.
2. **Second-press close** was not observed cleanly: the successful runs' follow-up press happened while focus
   had moved to the DevTools window, which is its own window; toggling from the host page again needs a
   foreground assertion per A.5. The DevTools window is closed by its own chrome in the meantime.
3. The probe inspected `about:blank` (the `127.0.0.1:3999` page is not served), which is enough for the
   accelerator question but does not prove per-page scoping on a real plugin page.
