# Round 213 (addendum to round208) — fresh pass291 L-HOME launcher pair

Date: 2026-10-25. Task: task-90. Complements `round208-lhome-launcher-evidence.md`, whose live half was blocked by a shared pass290 instance. No product code change (no new divergence confirmed), no commit, no passN exe, native app untouched, no `runtime/`/`third_party/` edits. Temp profiles removed and all captured windows closed.

## 1. Lock assertion

Before launching: `Get-CimInstance Win32_Process -Filter "Name LIKE 'whaleisle%'"` → **no process**; `Get-Process -like 'whaleisle*'` count **0**. The single-instance lock was free for the whole run, so this pair is a real, uncontended pass291 launch (contrast round208, where pid 11564 from another slice held it and three launches exited in <0.5 s).

## 2. Recipe (the two things round208 got wrong)

Throwaway `WHALEISLE_USER_DATA` profile, `config.json` written BOM-free:

```json
{
  "theme": "deepseek.dark",            // second run: "deepseek.light"
  "locale": "zh", "host": "127.0.0.1", "port": 3099,
  "workspace": "", "autoStartDesktop": false, "quitAfterStart": false,
  "goPreviewNoticeAck": "2026-09-28.1",
  "goUnifiedShortcuts": { "schemaVersion": 2, "profiles": { "desktop:windows": {} } }
}
```

plus an empty `welcome-shown` marker; launch `whaleisle-pass291.exe` and wait for the separate launcher window. Both fixes from round208 are in force: **`port` is a JSON number** (a quoted `"3099"` reproduces the 549×460 config-read-failure dialog) and the file is written with `UTF8Encoding($false)` (a BOM is rejected — round195 added tolerance in `internal/config`, but this build may predate it).

## 3. Captures (real windows, PrintWindow, per-monitor-v2)

| capture | bytes | measured |
| --- | --- | --- |
| `evidence/screens/lhome-pass291-launcher-dark-150dpi.png` | 60,549 | 1590×990 px @144 dpi = **1060×660 DIP** |
| `evidence/screens/lhome-pass291-launcher-light-150dpi.png` | 61,723 | 1590×990 px @144 dpi = **1060×660 DIP** |

Both show the separate launcher window only (no main window beside it), the native nav glyphs, the 48 px whale avatar in the home card, the `鲸屿桌面端` title and the `vdev` version pill.

## 4. Diff against the native reference

`evidence/screens/reference-launcher-1060x660.png` (headless Chromium render of `launcher.html`, **light**, preload data absent) is downscaled to 1060×660 and compared with each capture after the same high-quality downscale, `scripts/acceptance-diff.py --mask-mode auto`:

| pair | unmasked mismatch | wash-masked | auto ratio | note |
| --- | --- | --- | --- | --- |
| light vs reference | 60,456 / 699,600 = **8.64 %** | 4,400 px (0.63 %) | **8.01 %** | comparable to round208 (8.53 / 7.90) on the older build |
| dark vs reference | 699,129 = 99.93 % | 606,713 px (86.72 %) | **13.21 %** | the reference render is light-only, so 86.7 % of the frame is low-frequency wash; the dark pair is structure-only, not a pixel claim |

The surviving light disagreement is data/state, not structure: the live capture shows the in-flight catalog line `正在读取 MyGo 发布目录…` plus its `取消下载/读取` button and a `vdev` pill, where the reference has the preload placeholders `读取版本…` / `正在读取状态…` and no catalog row at all.

## 5. Metric table (live probes on the dark pass291 capture + deterministic tester values)

| metric | native | pass291 capture/measurement | verdict |
| --- | --- | --- | --- |
| window | 1060×660 DIP, radius 20 | 1590×990 px = 1060×660 DIP (window rect) | equal |
| rail width | 188 + 1 px `border-l1` | sidebar/content boundary at x = 282–284 px = **188.0–189.3 DIP** | equal |
| nav pitch | h 40 + gap 4 = 44 | active pill column transitions y = 175 / 235 px = **116.7 → 156.7 DIP**: top 116.7, height 40; tester rows 116/160/204/248/292/336 | equal |
| nav icons | 16×16 launcher.html paths | same paths (round151), visibly filled house/cube/list/tray/puzzle/gear | equal |
| home card | padding 20 24, radius 16, border-l2 | same box in both captures | equal |
| card product row | avatar 48, gap 12, eyebrow 10/16 ls 1, title 16/24 | 48 px avatar (72 px at 150 %), title starts at x ≈ 297 DIP (tester) | equal |
| action buttons | 34 DIP pill + outline sibling, gap 8 | same pair, same gap in both captures | equal |
| caption | 48 drag strip, right−120 reserve, 32×32 controls | same; window controls at the top right in both captures | equal |
| footer | rail-foot border, 12/18 route chip | `线路 GitHub` chip present at the bottom | equal |

## 6. Fixed / blocked

- **Fixed this round:** nothing new — no metric differs beyond data/state, so no code change and no new test was warranted (round147 closed the card product row, round151 the nav SVG paths and the `鲸屿桌面端` copy).
- **Blocked:** the dark pixel ratio needs a **light-independent reference** — the headless render is light-only, so dark is compared structurally (a dark reference render would need the page's theme attribute/preload stub, or the live native launcher under D2).
- **D2 note (unchanged, explicit):** the installed Electron app cannot be driven to its own launcher window from this environment (a second instance focuses the running main window; the launcher is only reachable through the cold-start gate). Until the D2 cold-start decision is made, the native half of L-HOME is the headless `launcher.html` render.

## 7. Gates

No code changed. `go test ./internal/ui/launcher ./internal/ui/boot ./internal/ui/frame` was blocked during round208 by a concurrent `internal/ui/chrome` edit; it should be re-run now that slice has landed — not re-run here because this round produced evidence only.
