# Round188 — S03 live press on pass286 and the G10 panel-toggle glyph

Date: 2026-10-25. Read-only native app (screen capture + window enumeration only), no product code, no
commit, no `passN`. Everything below ran against `whaleisle-pass286.exe` (04:11) and, for the instrumented
run, a temporary build of HEAD `8798e572` (`docs: record pass286 and the rounds 181-185 work`). The
instrumentation was reverted and the worktree removed; `git status internal/desktop/webview` is clean.

---

## A) S03 — pressing Ctrl+Shift+I on the pet webview

### A.1 Method

1. **Fresh profile per run.** Unique `%TEMP%\wl-g71-<guid>` with `config.json` (real config + temp workspace +
   `mygoDesktopPet.enabled=true`), `welcome-shown`, `WHALEISLE_USER_DATA=<profile>`,
   `WHALEISLE_VENDOR=C:\AI\WhaleIsle-MyGo\runtime\vendor`. No profile is reused, which also removes round183's
   `dsh-usage-panel.sqlite` lock: the lock came from a *surviving* harness child of the previous run, not from
   the profile path (killing the app with `Stop-Process` leaves its node child holding the sqlite; the fix is
   a new profile **and** no reuse of the old directory).
2. **Instrumented build.** Two temporary edits, removed afterwards:
   - `internal/desktop/webview/g71_debug.go` (new file, deleted): appends to `$env:WL_G71_LOG` when set;
   - `internal/desktop/webview/devtools_shortcut.go`: one line on the install-ok and install-skipped branches
     plus one line per `OnBeforeInput` key event.
   Build: `go build -o tmp/whaleisle-g71.exe ./cmd/whaleisle` from a `git worktree` at HEAD, because the live
   working tree does not compile (concurrent teammate edits: `application_menu.go:254` `plain(...)` arity).
3. **Window and focus.** 22 s after start, `EnumWindows` found `Whale Isle` (main, 2160x1380 @ 144 dpi) and
   `Whale Isle companion` (pet, 360x657 DIP). The pet was restored (`ShowWindow SW_RESTORE=9`),
   `SetForegroundWindow` + `SetFocus`; `GetForegroundWindow()` returned the pet hwnd (7015356) — verified
   foreground, not assumed.
4. **Keys and observation.** `keybd_event` Ctrl↓ Shift↓ I↓ I↑ Shift↑ Ctrl↑, wait 5 s, then a full
   `EnumWindows` diff (every visible and invisible top-level window, with pid → process name, so
   `msedgewebview2.exe` windows are included). Then the same chord again, wait 5 s, second diff.

### A.2 Result — no chord on the pet window, and that is the code, not the keys

- **No DevTools/inspector window appeared.** The only "new" windows in the diff were the PowerShell host's
  own `GDI+ Window (powershell.exe)` (2x2, invisible) and `Default IME` (0x0, invisible).
- **No log file was created at all** — not an empty log: the instrumented `installDevToolsShortcut` never
  even reached its "install skipped" branch in that process.
- **Why (source):** the pet is not a `webview.Host`. `internal/desktop/pet/companion.go:195` opens it as a
  plain `mygo.Window` with
  `Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}`, and the chord is
  installed only for `Host` pages (`internal/desktop/webview/host.go:287` in `openPage`,
  `internal/desktop/webview/host.go:444` in the navigate path). Both facts together mean the pet page has no
  chord and no devtools to open.

So the negative live result is **not** evidence that synthetic keys fail to reach WebView2 — it is evidence
that this window has no S03 wiring at all. The task's "no new window + no log" criterion is met, and the
cause is stronger than a key-delivery failure.

### A.3 The Host-webview press is still unproven

To exercise the chord where it *is* wired, `%TEMP%\whaleisle-accept-open` was set to `browser` in a fresh
profile. No separate browser-preview window was created (only `Whale Isle` and the hidden `MyGoApp` helper,
202x56), and the log stayed absent, so no `Host` page existed and the chord path could not be pressed in the
timebox.

**Verdict for the synthetic-key question: UNPROVEN.** What is proven is narrower and precise:
`synthetic keybd_event + focus on the pet window → nothing, because the pet has no chord`.

To settle it (≈10 min, needs one interactive step):
1. Open a real `Host` webview (plugin page or browser preview) so `installDevToolsShortcut` runs — the
   instrumented build proves it by logging `install ok packaged=false allowed=true`.
2. Focus the WebView2 child (`Chrome_WidgetWin_1`/`Chrome_WidgetWin_0` under the host window; note the
   wrapper HWND is not the one that receives keys) and send the chord, or press it on a real keyboard.
3. Decide from the log: `input key="i" down=true ctrl=true shift=true` ⇒ keys arrive and the remaining issue
   is `IsDevToolsToggle`/devtools availability; no line ⇒ `keybd_event` does not cross into the WebView2
   accelerator path and a real keyboard (or `SendInput` after child focus) is required.

### A.4 Gating

`cmd/whaleisle/main.go:2265-2266` installs
`!mygo.App.IsPackaged() || cfgStore.Get().OpenDevTools`, so on a dev build (`IsPackaged()==false`) the chord is
always allowed and the `openDevTools=false` branch cannot be exercised without a packaged artifact. The pure
gate is already covered by `internal/desktop/webview/devtools_shortcut_test.go`
(`DevToolsShortcutAllowed`). No packaged-like run was attempted; recorded as a residual, not as a pass.

### A.5 Side finding worth a parity row

If the native `devtools-shortcut.js` is attached per `webContents` (it is: one listener per page, including
the live2d/pet window), then the pet's devtools **are** reachable natively, while MyGo marks the pet page
`DevToolsDisabled` and gives it no chord. That is a small, real parity gap (S03-family) rather than a
measurement problem; it should be decided deliberately (enable devtools for the pet page, or document that
the pet is diagnostics-exempt).

---

## B) G10 — the caption panel-toggle glyph, measured

### B.1 Method

- Fresh pass286 run, default 1440x860 window at 150 % (`dpi=144`, `s=1.5`), dark theme.
- Screen crop of the caption band at window-relative `x = 1236..1300 DIP`, `y = 0..56 DIP` for **both**
  windows (native `Whale Isle` 2268x1366 at (200,2); MyGo 2160x1380 at (200,0)).
- The terminal-toggle button box is `1260..1292 DIP` wide; its top is **8 DIP (native)** and **12.67 DIP
  (MyGo)** — the round182 caption registration — so each side is cropped with its own top.
- 32 DIP button → 48 physical px, nearest-neighbour 4x upscale, then a binary scan of the inner 28 px
  window (excludes the button edge and the neighbouring `Commit` pill): threshold = plate median − 10
  (plate median 156, glyph darkest 101, threshold 146). Ink is *darker* than the plate, not brighter — the
  first attempt assumed the opposite and measured the plate wash; that attempt is why the numbers below are
  the second, ASCII-verified pass.

### B.2 Result — equal within 1 physical pixel

| | native | MyGo (pass286) | Δ |
| --- | --- | --- | --- |
| button top (DIP) | 8.00 | 12.67 | +4.67 (the known caption offset, not a glyph fact) |
| glyph bbox w (DIP) | **14.00** | **14.00** | **0.00** |
| glyph bbox h (DIP) | **12.67** | **14.00** | **1.33** |
| glyph bbox (physical px) | 21 × 19 | 21 × 21 | 0 × 2 px |
| ink pixels | 179 | 211 | +32 |

Both ASCII maps are the same rounded-square terminal glyph with the same filled bar in the lower third; the
1-row height difference is the anti-aliased first row at 150 % (19 vs 21 px at a 146 threshold), i.e. below
one DIP of real shape difference and well below the 2 DIP bar the task set.

**Verdict: no change.** The icon is already at the native size (≈14 DIP on both sides), so the open question
"14 vs 16/18" resolves as **14 = 14, match**; `internal/ui/frame/workspace_trailing.go` must not be touched
(no test added, because no code changed).

Evidence: `evidence/round188/g10-band-native.png`, `g10-band-mygo.png` (band crops),
`g10-native-glyph@4x.png` / `g10-mygo-glyph@4x.png` (the measured inner region),
`g10-native-context.png` / `g10-mygo-context.png` (160x64 context),
`g10-measure.json` (bboxes, thresholds, ink counts), `g10-band-diff.json` (band-wide difference: 811 of 8064
pixels over 8 levels, dominated by the 4.67 DIP vertical offset, which is why a raw band diff cannot answer
the glyph question).

---

## C) Cleanup and state

- Instrumentation reverted: `internal/desktop/webview/devtools_shortcut.go` restored, `g71_debug.go` deleted,
  `git status --porcelain internal/desktop/webview` empty; the HEAD worktree `tmp/wt-g71` removed.
- Processes: pass286 / g71 instances stopped; surviving harness `node.exe` children reaped by killing the
  app and then removing the profile after their handles closed.
- Temp: `%TEMP%\wl-g71*` profiles, `wl-g71*-result.json`, `wl-g71*-input-*.log`, `tmp/g71*.ps1` and
  `tmp/whaleisle-g71.exe` removed.
- Native app: captured and enumerated only; window rect restored to 200,2 2268x1366 (it was not moved).

## D) Residuals

1. **S03 synthetic-key delivery to WebView2 is UNPROVEN** (A.3). The evidence gathered says the pet window has
   no chord; the Host path needs one interactive press or a child-HWND focus test.
2. **openDevTools=false gating** was not exercised (dev build always allows the chord); pure function is
   unit-tested.
3. **Pet devtools parity**: MyGo sets `DevTools: mygo.DevToolsDisabled` on the pet page while native attaches
   the chord per `webContents` — decide deliberately (§A.5).
4. G10 is settled in the "no change" direction at this resolution; a 3x/4x crop of a *hovered* button (the
   hover plate changes the threshold) was not taken.
