# Round 147 — launcher/welcome pixel parity with real-window capture

Date: 2026-10-25. Task: task-23. Scope: `internal/ui/launcher/`, this doc and its screens. No commit, no exe.

## 1. Method

- **MyGo baselines** (pass275, physical pixels, per-monitor-v2): `scripts/window-capture.ps1` (PrintWindow `PW_RENDERFULLCONTENT`) against the HWND enumerated for the process, temp profiles under `.devdata\pixel` (deleted afterwards; processes killed).
  - `launcher-pass275.png` — 1590×990 px at 144 dpi = **1060×660 DIP**, profile with `welcome-shown`, no workspace (launcher stays).
  - `welcome-pass275.png` — 900×1050 px at 144 dpi = **600×700 DIP**, profile without the marker (first-run welcome). Only the welcome window was visible in that run.
- **Native reference render**: headless Chrome 1060×660 DIP, `--force-device-scale-factor=1 --virtual-time-budget=3000`, `file:///…/reference/electron/src/renderer/launcher.html` → `reference-launcher-1060x660.png`. The launcher page paints statically (only preload data is missing: "读取版本…", "正在读取状态…"), so geometry and type are comparable.
- **Welcome reference render is blocked**: `welcome.html` is an empty `<div id="root">` filled at runtime by `welcome/welcome.js` from the preload bridge, so a static headless render is blank (`reference-welcome-600x700.png`, kept as evidence of the attempt). The welcome comparison therefore uses the source/DOM metrics only, and the MyGo capture is recorded for a later visual pass.
- All comparisons in **DIP** (MyGo captures are 1.5× physical).

## 2. Reference metrics (source) vs MyGo (tester-measured)

`launcher.css` → `internal/ui/launcher/panel.go`:

| metric | native (launcher.css) | MyGo | verdict |
| --- | --- | --- | --- |
| shell radius / hairline | 20 + inset 1px `border-l2` | `Radius(20).Border(1, colBorderL2)` | equal |
| rail width | 188 + 1px right `border-l1` | `Width(188)`, `BorderWidth(0,1,0,0)` | equal |
| rail padding / gap | 56 12 16 / 18 | `Padding(56,12,16,12)`, `Gap(18)` | equal |
| rail head | avatar 36, gap 8, padding 0 8 | `Width(36).Height(36)`, `Gap(8)`, `Padding(0,8,0,8)` | equal |
| brand type | cn 18/22 700 −0.04em, en 10/12 600 .12em, role 12/18 500 .06em | `18/22 700 ls −0.72`, `10 600 ls 1.2`, `12/18 500 ls 0.72` | equal |
| nav item | h 40, padding 9 16 9 12, radius 12, gap 8, icon 16, 14/22 | `Height(40)`, `Padding(9,16,9,12)`, `Radius(12)`, `Gap(8)`, 16px icon, 14 | equal |
| nav pitch | 40 + 4 gap = 44 | measured rows y = 116, 160, 204, 248, 292, 336 (h 40, pitch 44) | equal (first row 116 vs ≈114: ≤2 DIP) |
| active pill / hover | `sidebar-nav-item-active-accent` wash, no bar | `colNavActive` wash only | equal |
| stage padding | 56 24 24 | `Padding(56,24,24,24)` | equal |
| home card | padding 20 24, radius 16, 1px `border-l2`, `sidebar-fill` | same | equal |
| **home product row** | **48×48 avatar, gap 12, eyebrow 10/16 ls 1px, title 16/24 600 +2px** | **avatar missing**, eyebrow 10 600 (12/18 default line), title 16 without a 24 line | **divergent → fixed** |
| caption / drag strip | 48, 32×32 controls, 12 8 4, drag region right−120 | 48, 32×32, Top 12 Right 8, drag `Right(120)` | equal |
| footer route chip | `.rail-foot` margin-top auto, padding 10 12 0, 1px top border, 12/18 | same block, `Padding(10,12,0,12)` + top border | equal |

## 3. Fixed

**Home desktop card product row** (`internal/ui/launcher/panel.go` `homePanel`): the reference markup is
`<div class="home-product"><img width="48" height="48"><div><span class="home-eyebrow">WHALE ISLE</span><h3>…</h3></div></div>`,
with `.home-product { gap: 12px; align-items: center }`, `.home-eyebrow { 10/16, letter-spacing 1px, tertiary }`,
`.home-product h3 { 16/24, 600, margin-top 2px }`. MyGo had a comment claiming the reference has "no 48px avatar"
and rendered a bare text column.

- before: `launcher-pass275.png` (live window) / after: `launcher-after-homecard-1060x660.png` (tester render, same 1060×660 widget).
- now: 48×48 `brandAvatar()` + 12 DIP gap, eyebrow 10/16 with 1px tracking, title 16/24 semibold; the version pill and status/CTA rows are untouched (they already matched `.home-desktop-head` gap 16 and `.home-desktop-controls`).
- test: `TestHomeCardProductGeometry` — asserts the card title starts at x ≈ 297 DIP (189 rail + hairline + 24 stage + 24 card + 48 avatar + 12 gap; without the avatar it would be 249) and that the title keeps a 16/24 line box.

## 4. Remaining / blocked

- **Welcome pixel comparison blocked**: the reference page renders only through `welcome/welcome.js` + the preload bridge (`#root` is empty in the HTML), so no static headless screenshot exists. MyGo's welcome capture is recorded; its structure (48 caption with mini logo + close, centred heading/subline, two 34-ish DIP full-width buttons, skip link) still needs a metric pass against `welcome/welcome.css`, which this round did not finish.
- **Update error text under the home card**: MyGo prints the raw Go/network error (proxy + GitHub URL) where the reference shows a short localized message. Data-formatting issue in the update-check path (`internal/desktop/update`), outside this task's scope; listed for the owner.
- **Version pill content**: MyGo shows `vdev`/the installed label, the reference `读取版本…` while its preload data is absent — content, not layout.
- **Scrolling**: both sides scroll inside the stage; the launcher capture had no overflow, so scrollbar behaviour was not compared.

## 5. Tests / evidence

- `go test ./internal/ui/launcher ./internal/ui/welcome -count=1` → ok; `go vet ./internal/ui/launcher ./internal/ui/welcome` → clean.
- Screens: `docs/review-20261022/evidence/screens/` — `launcher-pass275.png`, `welcome-pass275.png`, `reference-launcher-1060x660.png`, `reference-welcome-600x700.png` (blank attempt), `launcher-after-homecard-1060x660.png`. (`mygo-launcher-pass275-150dpi.png` / `mygo-welcome-pass275-150dpi.png` in the same directory are the same windows from the Lead's capture run.)
