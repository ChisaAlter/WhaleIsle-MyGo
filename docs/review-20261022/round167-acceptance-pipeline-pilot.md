# Round 167 — acceptance capture pipeline and D-MAIN pilot

Contract: `docs/acceptance-ui-parity.md` (E1–E7, §2 rows, threshold 0 with registered masks).
This round builds the artifact pipeline and runs one pilot row so the contract can be reviewed with
real numbers. No product code, contract, `runtime/`, `reference/` or install file was touched; the
installed app was resized and restored, never written to.

## 1. What was built

| file | role |
| --- | --- |
| `scripts/acceptance-capture.ps1` | one row → `baseline.png`, `actual.png`, `geom.json`, `controls.json`, `accept-log.txt`, `mask.json` under `%TEMP%\whaleisle-accept-<UTC date>\<ID>\`. PowerShell 5.1, per-monitor-v2, resizes both windows to one fixture client size, foregrounds + proves no occlusion, copies the client area, dumps UIA, restores the windows. |
| `scripts/acceptance-diff.py` | PIL diff → `diff.png` (mask grey, mismatch red), `diff.json` with mismatch count/ratio, masked pixel count, per-channel max delta. Honours `mask.json`; allows the contract's 1-px size tolerance by cropping to the common size and recording it. |

Pipeline command used for the pilot:

```powershell
# delivered build (working tree, not a passN exe) on its own profile, dark theme attempted
powershell -File scripts/acceptance-capture.ps1 -Screen D-MAIN -NativePid 15560 `
  -MyGoPid <mygo> -ClientWidth 1440 -ClientHeight 860
python scripts/acceptance-diff.py "%TEMP%\whaleisle-accept-2026-10-24\D-MAIN"
```

Artifacts were also copied to `docs/review-20261022/evidence/accept-D-MAIN/` so a reviewer can open
them without hunting `%TEMP%`.

## 2. Pilot row: D-MAIN (main shell, terminal off, empty session)

| item | value |
| --- | --- |
| native | `C:\软件\Whale Isle\Whale Isle.exe` 0.3.10, pid 15560, class `Chrome_WidgetWin_1` |
| MyGo | `tmp\whaleisle-accept.exe` built from git `e96c0543` (working tree, 23 modified files), class `MyGoWindow` |
| fixture client | 1440×860 DIP @ 150 % = 2160×1290 px (see §4 for why not 920 DIP) |
| captured | baseline 2160×1290, actual 2160×1291 (1 px, inside the contract tolerance; diff crops to 2160×1290) |
| UIA nodes | native 84, MyGo 49 |
| mismatch (threshold 0) | **2 786 097 / 2 786 400 = 99.99 %**, masked 0, channel max Δ **240** |
| region ratios | sidebar 99.97 %, header 100 %, hero 100 %, composer 100 %, below 99.99 % |

### Why the ratio is ~100 %: the pilot captured two different states

1. **Theme.** The native app runs dark (`theme: system` on a dark desktop); the MyGo profile rendered
   light. Mean colour per region: sidebar native `rgb(30,38,47)` vs MyGo `rgb(146,186,191)`, hero
   `rgb(17,36,45)` vs light. The theme alone makes every pixel differ, so threshold 0 has nothing to
   say about layout here.
2. **Data state.** Native has the user's real profile: workspaces `whale-code`, `WhaleBridge`,
   `Whale Isle`, `Deepseek-Harness-Desktop`, `Whalelsle-MyGo`, `无工作目录`, a long session list and a
   signed-in account (Ayase); MyGo ran an empty throwaway profile (one workspace, no sessions, signed
   out). The sidebar therefore cannot match by construction.

Both are fixture problems, not renderer problems, and both are exactly what the contract's E5
(`fixture.md`) and per-row data states are for.

### Geometry comparison from the two UIA dumps (name → width/height DIP, Δy)

Controls present on both sides with a stable name still line up, which is the useful signal:

| control | native | MyGo | Δ |
| --- | --- | --- | --- |
| 自动化任务 row | 252×36 | 252×36 | Δy 0.7 |
| 鲸鱼娘 row | 252×36 | 252×36 | Δy 0.7 |
| 视图选项 (workspace toolbar) | 28×28 | 28×28 | Δy 6.7 (the native sidebar had one extra session row above) |
| 新建会话 | 252×38 | 252×38 | Δy 0 |
| 收起侧边栏 | 28×28 | 28×28 | Δy 0 |
| 账号菜单 (footer seat) | 252×**44** | 252×**36** | Δy 8.7 — **state**, not a bug: the native trigger is 44 DIP when a profile is signed in, the merged settings seat is 36 DIP when signed out |
| 标准模式 chip | 106.67×28.67 | 100×28 | width −6.67 (text metrics of the 13 px label; a real candidate for a follow-up row) |

The earlier round-152/156 headless measurements (sidebar 280, rows 252 at x=14, pitch 40, collapse
28×28 at (240,22)) are confirmed on the real native window: UIA reports 新建会话 at screen
(221,107) physical → (14,70) DIP with 252×38, i.e. the same numbers.

## 3. Findings that the contract must absorb

1. **Electron's UIA tree is lazy.** The first UIA query on the native window returns 0 descendants;
   the second (after ~600 ms) returns 84–300. The pipeline warms it up before walking. A reviewer
   running a UIA tool by hand will otherwise conclude the native app has no controls.
2. **Electron renders into a child HWND** (`Chrome_RenderWidgetHostHWND`), so `WindowFromPoint`
   returns the child, not the top-level window. The occlusion check accepts any descendant.
3. **A window taller than the monitor working area cannot be captured cleanly.** 1440×920 DIP at
   150 % is 2160×1380 px; with the Windows taskbar that reaches into `TrayNotifyWnd`. The pilot
   anchored both windows at (0,0) and used 1440×860 DIP. The contract's E3/E4 should fix the fixture
   size to something that fits the working area at the session DPI.
4. **Sizes can land 1 px apart** (2160×1290 vs 2160×1291) even through `SetWindowPos` with identical
   physical sizes — the contract's ≤1 px rule is right, and the diff tool crops to the common size.
5. **PowerShell 5.1 `Set-Content -Encoding UTF8` writes a BOM**, which broke JSON parsing; the
   pipeline now writes BOM-free JSON and the diff tool reads `utf-8-sig`.

## 4. Verdict on the strict threshold-0 rule (Go-native vs Chromium)

**Threshold 0 across a whole window is not achievable, and the pilot shows why in one number: the
channel max delta is 240 and 99.99 % of pixels differ once theme or data state differs.** Even with a
matching fixture the rule fails for reasons that are not defects:

- Text is rasterised by two different stacks (Chromium/Skia subpixel AA vs MyGo's Go glyph rasteriser
  with its own hinting), so glyph edges differ even at identical fonts, sizes and positions.
- The wallpaper/aurora layer is animated in the reference; any frame that is not the exact same
  instant differs, and its own CSS gradient dithering is not reproducible.
- Electron composites with GPU mipmapping/rounded-corner AA; MyGo draws AA for the same shapes with a
  different coverage rule.

Proposed practical criterion (for review, not applied to the contract here):

1. **Masked-pixel mismatch = 0** for every registered volatile region (caret, spinner, toast, hover,
   cursor, animated wallpaper bands) — unchanged from the contract.
2. **Structural parity instead of full-frame equality**: every control both dumps expose must match
   within **1 DIP** in x/y/w/h (the UIA comparison above already produces this table), and every
   control the native exposes must exist on the MyGo side (no missing/extra inventory).
3. **Pixel parity on the chrome, not on text edges**: require `mismatch_ratio ≤ 0.1 %` per region
   *after* the theme and data fixture match, plus **0** mismatches outside a 1-DIP dilation of text
   bounding boxes; text-antialiasing-only differences are tolerated but counted and reported.
4. Fixture: both sides on the same theme, same workspace/session inventory, same fixture client size
   that fits the working area, and one image per row captured at a defined "steady" moment (the
   pipeline's own settle + mask list).

## 5. What the pilot proves and what it does not

Proved: the artifact set can be produced end to end for a row (6 files + diff), the same client size
and DPI can be forced on both apps and restored, UIA inventories are obtainable on both sides, the
1-px tolerance works, and the geometry numbers agree with the headless measurements.

Not proved / still required:

- L-HOME (native launcher) is **not reachable** without changing the app: the launcher window is not
  currently open, a second instance focuses the existing main window, and the launcher is a separate
  1060×660 window while MyGo's launcher is a panel inside the 1440×920 main window — so L-HOME needs a
  decision on the fixture window before it can be captured.
- A matched fixture (theme + workspace/session inventory) — `fixture.md` per row, and a MyGo profile
  seeded from the native profile's workspace list.
- Dark-theme parity: MyGo's profile kept rendering light even with `theme: deepseek.dark` written to
  the profile config; the theme resolution path needs its own probe before any dark row is claimed.
- Per-row states/cases (`cases-<ID>.md`), the user's independent review signatures, and the E1–E7
  double-signature table. Nothing in this round is signed.

## 6. State restoration

Both windows were resized back (native restored to 2268×1366 at (200,2), verified in the log). The
native app was left on the empty 新会话 that was created for the capture; its previous session
(`dsh 最新版本合并`) was **not** re-selected because that row was not reachable in the UIA tree from the
collapsed workspace group — the user should click it back. The MyGo pilot process and its temp profile
were stopped/removed.
