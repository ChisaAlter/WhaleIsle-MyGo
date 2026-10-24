# Round 241 — ambient-gate verification blocked: the fresh build exits at startup

Attempted the requested verification (fresh temp exe from the current tree → normal harness launch →
main `MyGoWindow` → 1440×860 → log/samples/score). **It could not run: the binary built from the current
tree exits cleanly within seconds of starting**, on the same profile shape that pass298 launches from.
No samples, no capture, no score — reporting the blocker and the two environment traps instead of
fabricating numbers.

## 1. What was tried

| attempt | result |
| --- | --- |
| `go build -o tmp/whaleisle-p241.exe ./cmd/whaleisle` at `09d8c719` | build ok |
| `Start-Process -RedirectStandardError/-RedirectStandardOutput` | **fails before launch**: `Item has already been added. Key in dictionary: 'NO_PROXY' / 'no_proxy'`, then `'https_proxy' / 'HTTPS_PROXY'` — this session's environment carries case-variant duplicates and `Start-Process` with redirection builds a case-insensitive env dictionary |
| `cmd /c` with inline redirection (two quotings) | wrapper ran, app never appeared, no log files created |
| CRLF `.bat` with `set` + redirect + `echo exit=%ERRORLEVEL%` | app ran and **exited with code 0** in a few seconds; `p241.err` and `p241.out` both 0 bytes |
| plain `Start-Process` + `WHALEISLE_USER_DATA` | alive at 3 s check → gone; profile left with only `config.json` + `welcome-shown` (no `go-resume.json`, no `last-desktop-start.json`) |
| same exe copied to the repo root (so `runtime\…` resolves like the passN builds) | same: alive at 8 s? **no** — exited before the check |

Profile contents confirm the app never reached the window stage: the only files are the seeded
`config.json` (307 bytes: host/port/workspace/theme/locale/autoStartDesktop/**quitAfterStart=false**/
goPreviewNoticeAck) and `welcome-shown`. Workspace directory exists; no listener on 3098 or 3097; the
bundled runtime and node are present.

## 2. Diagnosis (corrected after a control build)  ---  IMPORTANT: my first hypothesis was wrong. A go build of the **known-good commit 4bc6c5c2** (built in a throwaway worktree) also exits within 8 s, leaving only config.json + welcome-shown, while the lead-built whaleisle-pass298.exe from that same commit runs for >50 s. So this is **not a tree regression**: the difference is in how the binary is produced or invoked, not in the source.

- The exit is **clean (code 0)**, so it is not a panic — the process decides to stop before creating a
  window. Candidates, in order of likelihood:
  1. **A regression between `4bc6c5c2` (pass298, which runs) and `09d8c719`** — the two commits in that
     range are the ambient-gate change ("stop a host-seeded gradient from painting without a wallpaper")
     and the preset-chip row-gap change. A gate that returns early in the startup path, or an
     `applyAppearance`/`paint` call made before the main state exists, would produce exactly this.
  2. A build-stamp/version requirement the passN binaries satisfy (they are built by the lead; a missing
     ldflags stamp can make the shell shut down quietly).
  3. A first-run path that decides the desktop is not needed because no harness hand-off happens (but
     that path used to keep the process alive, and `quitAfterStart` is explicitly false here).
- **Not** the environment: the earlier failure mode (`NO_PROXY` duplicate) is a `Start-Process` issue
  only, and the plain launch that worked today for pass298 works for no build now — pass298 is still the
  last known-good binary.

## 3. Environment notes for the pipeline (worth fixing once)

1. `Start-Process -RedirectStandard*` is unusable in this session until the duplicate case-variant proxy
   variables are normalised (only one of `NO_PROXY`/`no_proxy` and `HTTPS_PROXY`/`https_proxy` should
   exist). A CRLF `.bat` wrapper is the workaround; note that LF-only `.bat` files silently misbehave.
2. Temp builds must be the **same directory class** as the passN binaries when checking startup
   (`runtime\` resolves relative to the exe); the run-from-root retest above already covered that.
3. Worth adding to the launch helper: if the process is gone within ~10 s, dump `%ERRORLEVEL%` and the
   profile file list, and fail loudly — that would have caught this in one line instead of five attempts.

## 4. What this round does **not** claim

- No ambient-gate verdict: I could not observe `painter:`/`view:` log lines, so I cannot say whether the
  gate works on the real MAIN window.
- No 9-point samples, no capture, no three-way score, no structure — the last trustworthy D-MAIN numbers
  remain round237's (threshold 0 99.99 %, Δ>8 98.64 %, Δ>16 90.40 %, wash-mask 10.88 %, structure
  window 23 matched / 12 within 1 DIP).

## 5. Next step (concrete)

Rebuild the ambient gate on top of `4bc6c5c2` (known-good startup) in two commits — gate first, probe
log second — and re-run this exact slice; if the gate commit alone reintroduces the silent exit, the
startup path call order is the bug. The launch/verification procedure above is ready to reuse.

## 6. State

No shadow build left behind (root copy deleted), temp profile removed, native restored to (200,2)
2268×1366 on the empty 新会话. No commit, no passN, no contract edit.
