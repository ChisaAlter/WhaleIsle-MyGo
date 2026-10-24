# Round 173 — dark-matched D-MAIN attempt, alias structure report, L-HOME second-instance result

Follow-up to round170. Goal: get the fixture-matched pair by making **MyGo dark** instead of switching
the native app to light. Result: **the matched pair is still not captured** — and the reason is now
pinned to MyGo's own theme path, not to the acceptance pipeline.

No product code, no contract edit, no commit, no passN exe. Native app untouched on disk; the stray
second instance was killed; the user's main window (pid 15560) is intact and unchanged.

## 1. Dark matching: what was tried and what the evidence says

1. Throwaway profile with `theme: "deepseek.dark"` (also tried `"dark"` in round170), `locale=zh`,
   `autoStartDesktop`, `goPreviewNoticeAck`, temp workspace, `welcome-shown`; started with
   `WHALEISLE_USER_DATA=<profile>`; waited ~48 s.
2. Captured the window with CopyFromScreen and sampled the same points as the native baseline:
   - MyGo sidebar `rgb(196,196,197)`, hero `rgb(237,246,241)` → **light**.
   - Native sidebar `rgb(30,38,47)`, hero `rgb(17,36,45)` → dark.
3. After the run, the profile's `config.json` still contained `theme = deepseek.dark`, so the config was
   **not** rewritten; the value is simply not what the window resolves. Candidate sites, read but not
   modified:
   - startup applies it (`cmd/whaleisle/main.go:103` `s.themeID = cfg.Theme`, `:121`
     `applyAppearance(cfg.Theme)`);
   - `syncHostAppearance` runs every frame (`main.go:3103`) and the save path only keeps the desktop
     draft when the host appearance was **not** accepted (`main.go:845-880`:
     `if !s.hostAppearanceAccepted { c.Theme = s.desktopDraft.theme }`, `s.themeID = cfg.Theme`);
   - so a light window implies either `hostAppearanceAccepted` became true with a light host scheme,
     or the draft's theme won. Both are inside MyGo — task-52 owns the mapping; the acceptance
     pipeline cannot work around it without editing product code.
4. The host appearance namespace was not found on disk to seed it: `dsh-home/profiles/web` has no
   settings JSON containing a scheme, and no appearance file exists anywhere under `dsh-home`, so
   "seed the host preference" has no documented on-disk target today.

**Consequence:** a dark-matched pair needs task-52's fix (or an accepted way to seed the host
appearance). The fixture recipe in `evidence/accept-D-MAIN/fixture.md` stands; only its theme step is
open.

## 2. Environment limitation found (pipeline note)

Two other MyGo builds were running in this workspace during the attempt (`whaleisle-panel-mounted`,
plus earlier captures). My throwaway instance **exited a few seconds after starting** and a
`MyGoWindow` remained owned by the other build: the single-instance lock is workspace-scoped, so a
concurrent MyGo process steals the handover and the capture target disappears. The acceptance runner
must assert "no other MyGo window" before capturing (the pipeline already fails with
`no running MyGo window`, which is the right failure, but the root cause is contention, not code).

## 3. Structure report with the alias table (unmatched pair, 1440×860 DIP)

`scripts/acceptance-structure.py` now carries an explicit alias table for accessible-name drift
(`新建会话` ↔ `新会话`, `探索未至之鲸` ↔ `探索未至之境`, the two composer resize separators), and
reports alias-matched pairs separately from true missing/extra. Numbers on the round167 pair:

| metric | round170 (no aliases) | round173 (aliases) |
| --- | --- | --- |
| named native / MyGo controls | 64 / 44 | 64 / 44 |
| matched by (type, name+alias) | 15 | **17** |
| within 1 DIP | 4 | **4** |
| outside 1 DIP | 11 | 13 |
| native-only (missing) | 48 | **45** |
| MyGo-only (extra) | 29 | **26** |

Per-region pixel ratios are unchanged (sidebar 99.97 %, header/hero/composer/footer 100 %) because the
theme still differs; they remain a measurement of the fixture, not of the renderer. The alias table
confirms that part of the "missing/extra" gap was naming, not structure — but only 2 of 45 pairs, so
the dominant gap is still data state (the native session tree + signed-in account) and the caption
plate injection.

Conclusion for the criterion: the structural leg cannot pass on this pair (4/17 within 1 DIP), and the
pixel leg cannot be evaluated at all until the theme matches. The alias table is a prerequisite
improvement, not a fix for the score.

## 4. L-HOME: native launcher is not reachable from the running instance

Attempts this round, all recorded:

| route | result |
| --- | --- |
| second-instance launch (`Start-Process "Whale Isle.exe"`) | spawned a second main process (pid 4584) plus a child main (25284); **no new window** appeared — enumeration found only the existing 1512×911 `Chrome_WidgetWin_1`, no 1060×660 window. The strays were killed; the user's main window was untouched. |
| app menu item / tray | not reachable: the app menu is not exposed to UIA (round170: `Alt+F` walk left the tree unchanged), and the tray menu requires shell interaction that this headless pass cannot drive |
| cold start with no launcher gate | the launcher window only exists on a cold start with no workspace/launcher marker — i.e. a **separate launch profile**, not the running instance (this is the round139 `launcher_window.go` path MyGo also uses) |

MyGo's own launcher **is** a separate 1060×660 window on a cold start with a no-workspace profile, so
its half of L-HOME is capturable today; the native half is not, from a running instance.

**Fixture decision needed (unchanged from round170, now with the second-instance result):**

- **Option A (recommended):** capture L-HOME from **cold starts on both sides** — native with a
  no-workspace profile (so its launcher window opens) and MyGo with a no-workspace profile (its
  launcher window opens), both at their native 1060×660, then diff window-to-window. This needs a
  dedicated launch recipe per app (profile prep, cold start, wait for the 1060×660 window) and no
  product change.
- **Option B:** score only the launcher's inner panel at a common crop, documented as
  "panel-vs-window" — cheaper but weaker evidence.
- **Option C:** defer L-HOME until the launcher windows are structurally comparable.

Until the Lead picks one, L-HOME is unmeasured.

## 5. State and cleanup

- Native: user's main window (pid 15560) unchanged at 200,2 2268×1366 px @144 dpi, theme untouched
  (`跟随系统`, dark), still on the empty `新会话` created in round167 — the user should re-select their
  previous session.
- Stray second-instance main processes (4584, 25284) killed; only 15560 and its renderer/gpu children
  remain.
- MyGo throwaway profile deleted; no MyGo process left from this round.
- `structure.json` (with aliases) copied to both the evidence dir and the `%TEMP%` row dir.
