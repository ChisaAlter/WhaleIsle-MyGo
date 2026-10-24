# Round212 — S03: direct keyboard deny observation

Date: 2026-10-25. Tree at HEAD `304d5f35` (pass291). No product code changed; no commit; no `passN`;
read-only native app. The probe lived in `tmp/pagegate/` and is deleted; every `%TEMP%` profile and log from
the runs was removed.

Goal: close the last S03 gap — press Ctrl+Shift+I at a page whose inspector flag is **known**, and show a
**deny** case (keys delivered, no DevTools) rather than only an allow case.

---

## 1. Method

### 1.1 Probe (`tmp/pagegate`, deleted)

A plain `mygo.Window` loading a local page — deliberately **not** a `webview.Host`, so the shell's Go chord
(`installDevToolsShortcut`) cannot confound the result:

```
win := mygo.NewWindow(WindowOptions{Title:"PageGate", 760x520,
        Page: PageOptions{DevTools: DevToolsEnabled | DevToolsDisabled}})
win.Page().LoadHTML(page, "")     // local page with an autofocus input and a keydown logger
```

- `WL_PAGE_GATE=allow` → `mygo.DevToolsEnabled`; `WL_PAGE_GATE=deny` → `mygo.DevToolsDisabled`.
- The page reports each `keydown` through a bound Go method (`Probe.Key`) into `%TEMP%/wl-pg-<gate>-keys.log`,
  so **delivery is observable independently of whether DevTools opens**. The page also prints the last key
  inline (`key=I ctrl=true shift=true`), which is visible in the window screenshots.

### 1.2 Harness (`tmp/pg.ps1`, deleted)

1. Start the probe with a throwaway profile, find the `PageGate` window, `ShowWindow(SW_RESTORE)` +
   `SetWindowPos(HWND_TOPMOST)`.
2. `AutomationElement.SetFocus()` on the window — **fails**: "Target element cannot receive focus" (see §3.2).
3. Real click at the window centre (`SetCursorPos` + `mouse_event`), then `AttachThreadInput` + ALT-tap +
   `SetForegroundWindow` retry loop until `GetForegroundWindow() == target`.
4. Assert the page has keyboard focus with `GetGUIThreadInfo(thread of target).hwndFocus` being a **child** of
   the window (the WebView2 HWND), not the window itself.
5. Send the chord, read the page key log, then count top-level windows whose title matches `*DevTools*`
   (class `Chrome_WidgetWin_1`, owned by `msedgewebview2`), and capture.

---

## 2. Results

Both runs: `foregroundIsTarget=True`, `focusIsDescendant=True` — the preconditions the earlier slices never
proved.

| Step | allow (`DevToolsEnabled`) | deny (`DevToolsDisabled`) |
| --- | --- | --- |
| foreground == target | True | True |
| page focus is a child HWND | True (`focusHwnd=21892148`) | True (`focusHwnd=16911406`) |
| SendInput chord delivered (page log) | **False** (no lines) | **False** (no lines) |
| `keybd_event` chord delivered (page log) | **True**: `Control ctrl=true`, `Shift ctrl+shift`, `I ctrl=true shift=true` | **True**: **identical three lines** |
| DevTools windows after the press | **1** — `DevTools - about:blank`, class `Chrome_WidgetWin_1`, `pid=33488`, 960x960 | **0** |
| Screenshot | `pagegate-allow-devtools.png` (DevTools welcome page), `pagegate-allow-window.png` (page shows `key=I ctrl=true shift=true`) | `pagegate-deny-window.png` (page shows `key=I ctrl=true shift=true`) |

Evidence files: `docs/review-20261022/evidence/round212/` — the two window screenshots, the allow DevTools
capture, both text reports, both page key logs.

**Direct deny observation: achieved.** The same chord, delivered by the same method into the same probe page,
opens the inspector when `PageOptions.DevTools` is enabled and produces **no** DevTools window when it is
disabled — with the page proving receipt in the deny run.

---

## 3. What this settles, and the two automation facts worth keeping

### 3.1 The page option is the effective S03 switch on Windows

Combined with round193 (`docs/review-20261022/round193-s03-host-press.md`):

- round193: on a `Host` page the shell's Go gate (`SetDevToolsShortcutAllowed(false)`, installer skipped) did
  **not** prevent DevTools — Host pages leave `PageOptions.DevTools` at its zero value `DevToolsAuto`
  (`third_party/mygo/window.go:40-45`), so WebView2's own accelerator was on.
- this round: at a page with `DevToolsDisabled` the chord is delivered and ignored (deny), and with
  `DevToolsEnabled` it opens (allow).

So the real switch for Ctrl+Shift+I is WebView2's `AreDevToolsEnabled`, i.e. `PageOptions.DevTools`, not the Go
chord. Consequences: (a) the shell's Go handler is redundant on pages with devtools enabled and dead on pages
without; (b) any packaged-build gating must set `PageOptions.DevTools` from the product gate
(`!IsPackaged() || cfgStore.OpenDevTools`, `cmd/whaleisle/main.go:2265-2266`) or apply a runtime
`put_AreDevToolsEnabled`, because the config flag alone no longer controls the entrypoint.

### 3.2 Automation facts (these cost the earlier slices their results)

- **`AutomationElement.SetFocus()` on the top-level window fails** ("Target element cannot receive focus").
  UIA focus must target the page/document element, not the window; the working recipe is topmost + a real
  click into the page, then verify `GetGUIThreadInfo().hwndFocus` is a *child* HWND.
- **`SendInput` with `wVk` and `wScan = 0` delivered nothing to the WebView2** (page log empty, both runs),
  while **`keybd_event` delivered** the identical chord (page log three lines). Chromium's keyboard path wants
  scan codes; `keybd_event` fills them, the naive `SendInput` key record does not. Future press tests should
  either use `keybd_event`, or `SendInput` with `KEYEVENTF_SCANCODE` + `MapVirtualKey`.
- The page-level key log is the delivery proof; the shell's `OnBeforeInput` never sees the chord (round193), so
  it cannot be used to infer delivery.

---

## 4. Residual

1. **Packaged artifact not tested.** These runs used a `go build` binary. In a production `mygo build`,
   `DevToolsAuto` resolves the other way; whether `openDevTools=false` then yields deny end-to-end is unproven
   on a packaged exe.
2. **Linux/darwin not tested.** The probe is Windows/WebView2-specific (`DevTools` maps to WebKitGTK/WKWebView
   settings there, with a different accelerator story); the deny observation should be repeated per platform if
   the matrix claims it cross-platform.
3. **Host pages are still DevToolsAuto.** This round proves the page-option behaviour, not that the product
   uses it: wiring `PageOptions.DevTools` to the config gate across Host pages and the pet
   (`internal/desktop/pet/companion.go:195`) remains the open product decision from round193.
4. The probe pressed a local page (`about:blank` in the DevTools title, because the loaded document is a
   `LoadHTML` data page); per-page scoping on a real plugin page is implied by the Host path but not re-tested
   here.
