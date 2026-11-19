# polite — real-machine runs that never take the operator's mouse

One run, one instance, no focus theft. This is the only sanctioned way to drive the MyGo desktop on Ayase.

The operator works on this computer. A test run that raises a window, moves the pointer, or sends a keystroke is a defect in our process, not an acceptable cost of testing.

## Invocation

```powershell
cd C:\AI\WhaleIsle-MyGo-polite
go build -trimpath -ldflags '-H windowsgui' -o dist/polite/whaleisle.exe ./cmd/whaleisle
.\tools\polite\selftest.ps1 -Exe .\dist\polite\whaleisle.exe
```

A lane drives the app through an ops file:

```powershell
.\tools\polite\run.ps1 -Exe .\dist\polite\whaleisle.exe -Ops .\qa\my-lane-ops.ps1 -EvidenceDir .\qa-evidence\my-lane
```

The ops file is dot-sourced inside the run, so it calls the primitives directly:

```powershell
# qa/my-lane-ops.ps1
function Invoke-PoliteOps {
    param($Session)

    $shot = Invoke-PoliteStep -Session $Session -Name 'snapshot' -Action {
        Read-PoliteSnapshot -Session $Session
    }
    $node = Find-PoliteNodes -Snapshot $shot | Where-Object { $_.role -eq 'button' -and $_.name -eq '设置' } | Select-Object -First 1
    if (-not $node) { throw 'no settings button in the tree' }

    Invoke-PoliteStep -Session $Session -Name 'open settings' -Action {
        Invoke-PoliteAction -Session $Session -Ref $node.ref_id -Action 'click'
    }
    Invoke-PoliteStep -Session $Session -Name 'snapshot settings' -Action {
        Read-PoliteSnapshot -Session $Session | Out-Null
    } | Out-Null
}
```

Rules that are not negotiable in an ops file:

- One `Invoke-PoliteStep` wraps **one** action, so the foreground is sampled on both sides of it.
- Actions go through `Invoke-PoliteAction`, which shells out to `agent-desktop` **without** `--headed`. Never call a coordinate action, `SendInput`, `focus-window`, or `mouse-click`.
- A step must be idempotent. The profile is throwaway; the operator's screen is not. Opening a modal or a menu raises a window in front of them even when the foreground handle does not change.

From PowerShell directly:

```powershell
. .\tools\polite\run.ps1
$s = Start-PoliteInstance -ExePath .\dist\polite\whaleisle.exe
Invoke-PoliteStep -Session $s -Name 'snapshot' -Action { Read-PoliteSnapshot -Session $s }
Close-PoliteInstance -Session $s
Show-PoliteEvidence -Session $s
```

Dot-sourcing gives you `Start-PoliteInstance`, `Wait-PoliteWindow`, `Read-PoliteSnapshot`, `Get-PoliteNodes`, `Find-PoliteHarmlessTarget`, `Invoke-PoliteStep`, `Invoke-PoliteAction`, `Find-HangWindow`, `Close-PoliteInstance`, `Show-PoliteEvidence`, `Test-PoliteSession`. Running `run.ps1` without `-Ops` and without `-TargetRef` does one snapshot step and prints the evidence block.

## What each guard prevents

| Guard | Failure it prevents |
| --- | --- |
| `GetForegroundWindow()` recorded before anything else | Any later claim "we did not steal focus" becomes measurable instead of asserted. Sampled again after every step and at the end. |
| Refuses to start when `whaleisle`, `whaleisle-gw` (or the binary's own name) is running, and when an `electron`/`cua_node` process belongs to a MyGo tree | A second instance attaching to the first one's window, and a leftover browser seat from an earlier run. The refusal names each process, its pid and its path. An `electron.exe` from another product is listed in the evidence and ignored: it can hold neither our single-instance lock nor our window, and refusing over it would push lanes to close the operator's apps. |
| A named mutex `Local\WhaleIsle-Polite-Desktop-Run` | Two lanes on this machine at once, each believing it owns the display. |
| `--background` on every launch | `mainState.showMainWindow` → `Show()` + `Focus()` → two `SetForegroundWindow` calls, and the first-run import modal. `--background` replaces both with `ShowInactive` (`SW_SHOWNOACTIVATE`). |
| `WHALEISLE_ISOLATED=1` + `WHALEISLE_ISOLATED_VISIBLE=1` + `WHALEISLE_USER_DATA=<%TEMP%>` | Writing the operator's real profile, registering the URL scheme in their registry, deleting their autostart entry, installing a tray icon, and reading their legacy Whale Isle data. A visible window is still created, so the accessibility tree is real. |
| Fresh `%TEMP%\mygo-polite-<guid>` profile per run | Two runs sharing state, and any run reading `%APPDATA%\WhaleIsle-MyGo`. |
| Window discovery by **pid** in the accessibility tree, then `snapshot --window-id` | Matching another app that happens to be called "Whale Isle", and matching a window by title after the title changed. |
| Foreground re-read after every step, failure if it changed | A step that activates the window or raises a modal. The failure names both handles and the owner of the new one, and says whether it is our own process tree. |
| 界面未响应 check after every step and during startup | The `whaleisle-gw` watchdog: 20 s without a UI-thread heartbeat creates a 640×420 AlwaysOnTop window titled 界面未响应 and calls `Focus()` on it. Its presence means the probe was too heavy. The run stops instead of pushing through. |
| Close by `WM_CLOSE` to **our own pid's** window, then wait for that pid | Closing someone else's instance, and leaving an invisible process behind. In isolated mode `closeToTray` is forced off, so the window close is a real quit. |
| Reaping by `ParentProcessId` + executable path | The watchdog child left running after its parent died. Killed by exact pid, only after its path matches the binary we launched. |
| `Stop-Process` only on our own pid, after checking path | Never a process name. Other lanes share this machine. |
| `-H windowsgui` build | A console window appearing on the operator's screen for the app's stdout. |
| `[Console]::OutputEncoding = UTF8` around every `agent-desktop` call | Native output decoded as GBK, which corrupts every non-ASCII window title and breaks `ConvertFrom-Json`. |

`agent-desktop` itself needs a real Windows temp directory; the harness always hands it `%LOCALAPPDATA%\Temp` for `TEMP`/`TMP`, whatever the caller's shell thinks.

## Acceptance evidence a lane must attach

Paste this block (the harness prints it) into the lane report or the PR comment:

1. `foreground before` and `foreground after`, with both hwnd values printed and equal.
2. One `step <name>: <before> -> <after> unchanged` line per step.
3. The `command line` line, showing `--background` and the isolated profile path under `%TEMP%`.
4. `watchdog window : none`.
5. `app pid : <n>` with `pid exited : True` and `survivors : none`.
6. Any failure text verbatim. A failed run is reported, never retried until it goes quiet.

`-EvidenceDir <path>` also writes `evidence.txt` and every snapshot JSON there, so the tree that was actually read is part of the evidence.

A run is void if any of these is missing, or if the lane edited the harness to get past a guard.

## Exit codes

`selftest.ps1` and `run.ps1`:

| Exit | Meaning | What a lane does |
| --- | --- | --- |
| 0 | Pass: no foreground change at all, no watchdog window, our pid exited | Attach the block |
| 1 | Politeness or close failure — an **owned** window took the foreground, or the pid never exited | Stop. Report with the block. Never rerun to see if it goes away |
| 2 | Refused: a MyGo instance is running, the binary is missing, or `agent-desktop` is absent | Fix the machine state, then run |
| 3 | No harmless semantic target, or the action was refused before delivery (`not_delivered`) | Nothing reached the app. Read the dumped tree and pick a target with `-TargetRef` |
| 4 | Inconclusive: a window that is **not** ours took the foreground during a step | Rerun while the machine is quiet. The block names the pid that did it |

An owned change and an external one are different events, and the evidence says which: a change to a pid in our own process tree is a failure of the product; a change to someone else's window means the operator was using their computer, which voids the measurement without excusing or blaming the app.

Dot-sourcing `run.ps1` binds its parameter names (`$CliExe`, `$CliOps`, `$CliTargetRef`, `$CliTargetAction`, `$CliStartupTimeoutSec`, `$CliSnapshotTimeoutMs`, `$CliEvidenceDir`, `$CliKeepProfile`) in your scope. Use the `-Exe`/`-EvidenceDir` aliases on the command line; do not name your own variables `$Cli*`.

## What a lane must not do

- No `--headed`, no `mouse-click`, no `hover`, no `drag`, no `press`, no `type`, no `focus-window`, no `resize-window` on this machine.
- No `Start-Process` of `whaleisle.exe` outside this harness, and never without `--background`.
- No `Stop-Process -Name`, no `taskkill /IM`, no `Get-Process whaleisle | Stop-Process`.
- No writing under `%APPDATA%\WhaleIsle-MyGo`, `%LOCALAPPDATA%`, or the operator's Whale Isle (本体) profile. The isolated profile is disposable and gets deleted by the run.
- No patching the guards to make a run pass. If a guard refuses, the machine state is wrong; fix the state or report it.
- No second run while one is open. If a run leaves something behind, report the pids instead of killing them.

## Why `--background` alone was not enough

`cmd/whaleisle/main.go` computed `background := !isolatedDesktopRun() && slices.Contains(os.Args, "--background")`, so the flag was ignored in isolated mode. There, `WindowOptions.Hidden` stayed false and `mygo.NewWindow` called `win.Show()`, which is `ShowWindow(SW_SHOW)` followed by `SetForegroundWindow` (`third_party/mygo/internal/windows/window.go:801-808`). A polite isolated run was impossible: either the window never appeared (and the accessibility tree was empty), or it took the foreground.

The harness ships with that line changed to `slices.Contains(os.Args, "--background")`, so isolated runs honour the flag and go through `ShowInactive`. Without this change every lane would have had to choose between stealing focus and writing the operator's registry.

That fix was not enough on its own. Measured on `origin/main` with the flag honoured, the app still raised its own main window over another application's during a `snapshot` — see the run log in `tools/polite`'s pull request. The second half of the fix is in the library, not in the launch: `mygo.Window.SetInactiveOnly(true)` makes `Show()` reveal without activating, `Focus()` a no-op and `Restore()` a plain reveal, and `main.go` sets it for every background or isolated launch. Only `showMainWindow` — the choke point for actions a person took at this machine — clears it. A raising path added later cannot take the foreground by accident, because the window itself refuses.

## Why the rival guard is not "refuse on any electron"

Taking that rule literally would forbid testing whenever the operator has the reference desktop open, which is most of the time: five `electron.exe` from `C:\AI\Deepseek-Harness-Desktop` were running the first time this guard fired. A guard that always refuses gets commented out by the next person under time pressure, and then nothing protects the machine. So the guard names the conflict — a process that belongs to a WhaleIsle-MyGo tree — and lists a foreign `electron` as ignored in the evidence. Do not tighten it back without solving that problem first.
