# Round 208 — L-HOME: the separate 1060×660 launcher

Date: 2026-10-25. Task: task-86. No commit, no passN exe, native app untouched, no `runtime/`/`third_party/` edits. No product code changed in this round (nothing new was confirmed divergent beyond what rounds 147/151 already fixed).

## 1. Live capture: blocked by the shared single instance

Attempted three times with pass290 and a throwaway no-workspace profile (`.devdata\lhome86\profile`: `theme=deepseek.dark`, `locale=zh`, `workspace=""`, `autoStartDesktop=false`, `quitAfterStart=false`, `goPreviewNoticeAck=2026-09-28.1`, `welcome-shown`, no BOM). Every launch **exited in under 0.5 s with no window of its own**, i.e. the single-instance hand-over.

Evidence: a `whaleisle-pass290.exe` instance from another slice (pid 11564, started 04:54:10, command line `whaleisle-pass290.exe`, no profile env visible to me) already held the lock and owned a visible 1590×990 launcher plus a 2160×1380 main window at that moment; it later handed over to its desktop (only 2160×1290 remained). Polling my own PIDs at 400 ms never produced a window or a dialog, so the exit is the forward-to-existing-instance path, not a config/recovery dialog (contrast round195, where the BOM dialog was visible).

Consequence: a **fresh pass290 launcher capture was not obtainable** this round; the launcher code path is shared, and the live native launcher half remains blocked as the task predicted (D2 cold-start decision — see §5).

A fourth attempt, made after the other instance exited, did get its own process but only produced a **549×460 dialog** again: the fixture had `"port": "3099"` as a JSON *string*, so the run hit the same config-read-failure dialog as round195 (`internal/config` requires the number). That is a fixture typo, not a product defect — recorded here so the recipe is not repeated with quoted scalars; a clean retry with `port: 3099` was outside the remaining timebox.

## 2. Pair used instead, and the diff

| side | image | size |
| --- | --- | --- |
| native reference | `evidence/screens/reference-launcher-1060x660.png` (headless Chromium render of `launcher.html`, preload data absent) | 1060×660 DIP |
| MyGo live | `evidence/screens/mygo-launcher-pass276-150dpi.png` — pass276, no-workspace profile, real 1590×990 px window at 144 dpi, downscaled to 1060×660 for the diff | 1590×990 px = 1060×660 DIP |

`acceptance-diff.py` (auto wash mask, threshold 0) on the pair:

| metric | value |
| --- | --- |
| mismatched pixels (unmasked) | 59,700 / 699,600 |
| mismatch ratio (unmasked) | **8.53 %** |
| wash-masked pixels | 4,402 (0.63 %) |
| mismatch ratio (auto mask) | **7.90 %** |
| channel max delta | 255 |

The remaining disagreement is data/state, not launcher structure: the MyGo capture carries the live catalog warning (`目录读取失败 … Get "https://github.com/…"`) and a `vdev` version pill where the reference shows the preload placeholders (`读取版本…` / `正在读取状态…`), and the reference has no session data at all. Both are excluded from the L-HOME definition.

## 3. Metric table (unchanged from the last verified build)

| region | native (launcher.css / reference render) | MyGo (pass276/283 capture + tester) | verdict |
| --- | --- | --- | --- |
| window | 1060×660 DIP, radius 20 | 1590×990 px @144 dpi = 1060×660 DIP | equal |
| rail | 188 + 1 px `border-l1`, padding 56 12 16, gap 18 | 188, same padding/gap | equal |
| rail head | avatar 36, gap 8, padding 0 8 | 36, gap 8, 0 8 | equal |
| nav rows | h 40, gap 4 → pitch 44, first row ≈ 116 | measured y 116/160/204/248/292/336 (pitch 44) | equal |
| nav icons | 16×16 `launcher.html` SVG paths | same path data (round151 `svgpath.go`, clipped to the 16 box) | equal |
| stage | padding 56 24 24 | same | equal |
| home card | padding 20 24, radius 16, 1 px `border-l2` | same | equal |
| card product row | avatar 48, gap 12, eyebrow 10/16 ls 1, title 16/24 +2 | avatar 48, gap 12, same type (title starts at x ≈ 297 DIP) | equal |
| action buttons | 34 DIP pill (dark) + outline sibling, gap 8 | same pair, same gap | equal |
| caption | 48 drag strip, right−120 reserve, 32×32 controls | same | equal |
| footer route chip | rail-foot top border, 12/18 text | same | equal |

## 4. Fixed / blocked

- **Fixed this round:** nothing new. The two confirmable launcher divergences were already closed earlier: the home-card product row (round147) and the six nav SVG paths + `鲸屿桌面端` copy (round151). No new metric differs beyond data.
- **Blocked:** the fresh pass290 live pair (§1) — shared single instance; and the live **native** launcher half, which needs the D2 cold-start decision (below).
- **Gates:** `go test ./internal/ui/launcher ./internal/ui/boot ./internal/ui/frame` currently **fails to build for a reason outside this slice**: another agent's in-flight edit left `internal/ui/chrome/wallpaper_screen.go:120: undefined: out` and `background.go: "image/draw" imported and not used` (both files modified at 04:56:15, uncommitted). This slice changed no code, so there is nothing of mine in that failure; the gate should be re-run once internal/ui/chrome compiles.

## 5. D2 note (explicit)

The installed Electron app cannot be driven to its own launcher window from here: a second instance focuses the existing main window, and the launcher is only reachable through the cold-start gate (no workspace / failed last start). Comparing the live native launcher therefore needs the **D2 cold-start decision** (whether to boot the installed app with a profile that holds at the launcher, or to accept the headless `launcher.html` render as the reference). Until then, the native half of L-HOME is the headless render above.

## 6. Files

`docs/review-20261022/round208-lhome-launcher-evidence.md` only (plus the existing reference/capture images it cites). Temp profile `.devdata\lhome86` removed; no windows left behind by this slice.
