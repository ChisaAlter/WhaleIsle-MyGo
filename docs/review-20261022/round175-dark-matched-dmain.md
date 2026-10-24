# Round 175 — dark-matched D-MAIN on pass283, renderer delta, L-HOME MyGo half

Follow-up to rounds 167/170/173. `pass283` fixes the theme render path, so the D-MAIN pair is finally
**theme-matched (dark/dark)**. Two things remained unmatched, and one of them is new evidence about the
contract: the **wallpaper** layer, and the **data state** (session tree / signed-in account).

No product code, no contract edit, no commit, no passN exe. Native app untouched on disk; both temp
profiles and both processes cleaned up.

## 1. The pair is captured and verified dark

| side | build | window | dark check (sampled from the capture) |
| --- | --- | --- | --- |
| native | `C:\软件\Whale Isle\Whale Isle.exe` 0.3.10 (pid 15560) | `Chrome_WidgetWin_1`, 2160×1290 px @144 dpi | sidebar `rgb(30,38,47)`, hero `rgb(17,36,45)` |
| MyGo | `whaleisle-pass283.exe`, git `3e8fd30b` (throwaway profile: `theme=deepseek.dark`, `locale=zh`, `goPreviewNoticeAck=2026-09-28.1`, `autoStartDesktop`, temp workspace, `welcome-shown`) | `MyGoWindow`, 2160×1291 px @144 dpi | sidebar `rgb(25,25,27)`, hero `rgb(43,55,51)` |

Single-instance was asserted before starting (0 other `whaleisle*` processes; exactly one `MyGoWindow`
after start), which round173 showed is required because the lock is workspace-scoped.

Artifacts refreshed in `evidence/accept-D-MAIN/`: `baseline.png`, `actual.png`, `diff.png`,
`diff-matched.json`, `geom.json`, `controls.json`, `mask.json`, `accept-log.txt`, `structure.json`.
Fixture size 1440×860 DIP (2160×1290 px), the size that fits the working area.

## 2. Numbers

Threshold 0 (`diff-matched.json`): **2 786 143 / 2 786 400 = 99.99 %**, masked 0, channel max Δ **243**.

The pair is theme-matched but not wallpaper-matched, and threshold 0 counts *any* channel delta, so a
gradient that differs by 1 already counts. The useful number is the tolerant one (mean absolute channel
delta and the ratio above thresholds, per region):

| region | mean \|Δ\| | >1 | >8 | >24 | >48 |
| --- | --- | --- | --- | --- | --- |
| sidebar | 65.5 | 99.6 % | 92.1 % | 50.5 % | 39.7 % |
| header | 44.9 | 99.99 % | 95.8 % | 61.0 % | 39.9 % |
| hero | 57.9 | 100 % | 99.4 % | 83.6 % | 62.1 % |
| composer | 37.7 | 100 % | 100 % | 68.2 % | 20.7 % |
| footer | 147.6 | 100 % | 99.97 % | 99.8 % | 99.6 % |

Interpretation, region by region:

- **hero / composer** — the layout is state-independent, but both sit over the wallpaper: the native
  app shows the user's dark green wallpaper, MyGo its own teal/magenta gradient. Because MyGo's panes
  are translucent mica, the wallpaper shows *through* the sidebar and composer too, which is why the
  sidebar mean is 65 and the composer mean 38 even though the chrome colours are close.
- **sidebar / footer** — the native sidebar carries the user's five workspaces, a long session tree
  and the signed-in `Ayase` row; MyGo shows one workspace, one session, the merged signed-out
  `设置与账号` seat. That is data state, not a renderer defect; footer mean 148 is that difference plus
  the wallpaper showing through.
- **header** — native has the caption plate with branch/`Push`/window controls (desktop injection);
  MyGo draws its own 48-DIP caption with `初始化 Git`. Injection difference by design.

## 3. Structure (aliases on)

`structure.json` (name+role with the alias table from round173):

| metric | value |
| --- | --- |
| named native / MyGo controls | 64 / 44 |
| matched (incl. aliases) | 17 |
| within 1 DIP | **0** |
| outside 1 DIP | 17 |
| native-only | 45 |
| MyGo-only | 26 |

The `0 within 1 DIP` is the direct consequence of the two caption designs: the native page starts at
y=0 with its own 52-DIP header *and* the Electron caption plate, MyGo's merged 48-DIP caption puts the
whole page 4–6 DIP higher, and the hero block centres differently without the session tree. Round176
should compare **caption-relative** rects (subtract each side's own caption/hero origin) instead of
absolute screen rects — that is a keying change, not a renderer fix.

## 4. The round167 criterion, tested

| criterion | result | evidence |
| --- | --- | --- |
| masked mismatch = 0 | **pass** (trivially) | `mask.json` has no regions in this steady state and masked pixels = 0 |
| structural parity, 1 DIP | **fail** | 0/17 within tolerance, 45 native-only, 26 MyGo-only — data + caption offset + name drift (aliases already applied) |
| per-region ≤0.1 % after theme match | **fail** | theme is matched, wallpaper is not: >1 ratios are 99.6–100 % in every region |
| text-AA counted outside a 1-DIP dilation | **not separable yet** | the wallpaper delta dominates the histogram, so text AA cannot be isolated |

New finding for the contract: the **wallpaper is a fixture element like the theme**. Until both sides
render the same wallpaper (or the wallpaper band is a registered mask), no per-region pixel threshold
can be met, even with identical chrome. Two concrete options:

1. Seed MyGo's wallpaper from the same image the native app uses (the harness wallpaper namespace) —
   honest, but needs a way to read the native wallpaper file and write MyGo's store.
2. Register the wallpaper as a mask region and score only the opaque chrome (sidebar surface, caption
   band, composer card) — cheaper, and it is what the contract's mask registry was designed for;
   requires the Lead to accept "wallpaper is masked" as the D-MAIN definition.

Also worth fixing in the criterion: with two renderers, threshold 0 on a *gradient* is unreachable by
construction (a 1/255 difference counts). A tolerance (e.g. >8) plus a mean-delta bound is the only
computable form; that is exactly what the table above reports.

## 5. L-HOME — MyGo half captured, native half still blocked

- **MyGo**: cold start of `whaleisle-pass283.exe` with a **no-workspace profile** (`workspace=""`,
  `autoStartDesktop=false`) opens the launcher as its own **1060×660** window
  (`MyGoWindow`, pid 33676), captured to
  `evidence/screens/mygo-launcher-pass283-1060x660.png` (1590×990 px @150 %). The launcher is dark,
  with the rail (首页/组件/版本/导入/插件排查/设置), the home card, `版本管理`, `启动桌面端` and the
  release-catalog warning line. This is the window L-HOME needs, in the exact native size.
- **Native**: still unreachable from the running instance. Round173 measured that a second-instance
  launch spawns two extra `main` processes and **no window**; there is no tray/app-menu route exposed to
  UIA. The only path is a **cold start with a no-workspace profile** (native launcher window), i.e. the
  app must be restarted with a temporary user-data dir.
- **Fixture decision needed (unchanged, now half-blocked only on native):** Option A — cold-start
  window-to-window at 1060×660, native launcher window vs MyGo launcher window, both from
  no-workspace profiles. MyGo's half exists as of this round; the native half needs the Lead's go-ahead
  to restart the app with a temp profile (the running instance is the user's own).

## 6. State and cleanup

Both `pass283` processes stopped; both temp profiles deleted; `%TEMP%\whaleisle-accept-2026-10-24\D-MAIN`
refreshed and mirrored into `evidence/accept-D-MAIN/`. Native window restored to its original rect
(2268×1366 at (200,2)); the app is still on the empty `新会话` created for the acceptance row, so the
user should re-select their previous session. No other MyGo process left running.
