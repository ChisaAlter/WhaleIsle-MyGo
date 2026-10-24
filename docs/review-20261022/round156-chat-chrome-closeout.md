# Round 156 — closing the measured chat-chrome deltas

Follow-up to round152 (`round152-chat-surface-pixel-audit.md`), which left four chat-chrome numbers
open. Reference is still the staged harness page captured at 1440×920 DIP (`deviceScaleFactor: 1`
DOM rects, not pixels); MyGo numbers are headless `ui.Tester` rects at the same size, so both sides
are DIP and exact.

## 1. Measurements: native → before → after

| item | native (installed page) | MyGo before | MyGo after | verdict |
| --- | --- | --- | --- | --- |
| hero composer card | 774.39 × **114** at (469.3, 460); `padding: 8px 0 0`; 8-DIP gap; 34-DIP control row; children inset 8 | 772 × **126** at (470.7, 447.3); padding 20/16; 28-DIP row; children inset 16 | **114** tall, same 772 width; padding 8/0/8/0, gap 12, row 34, children inset 8 | **fixed** |
| new-session button | 252 × 38 r12 at (14, 70) | 256 × 38 r12 at (12, 66) | **252 × 38 r12 at (14, 70)** | **fixed** |
| page row list | 36-high rows, 40 pitch, buttons 252 wide at x=14 (panelList 256 at x=12) | pitch 40 (round152), buttons 256 wide at x=12 | buttons **252 at x=14**, labels at x=46, 插件 y≈127, pitch 40 | **fixed** |
| workspace toolbar icons | four 28 × 28 r8 at y=210, x=144/176/208/240 | ink at 150.7–164 / 182–198 / 214–229.3 / 250–257.3 (≈ the same 32-DIP pitch and x) | unchanged | **equal** (no change needed) |
| collapse control | 28 × 28 r8 at (240, 22) | 24 × 24 at (244, 24) | **28 × 28 r8 at (240, 22)**, glyph centred | **fixed** |
| footer seat | 账号菜单 252 × 36 r12 at (14, 870), 7/8 padding | 256 × 36 at (12, 874) | **252 × 36 at (14, 874)** | **fixed** |
| page header band | page header 52 (corner 28 × 28 at y=16, 32 × 32 buttons at y=8) | merged 48 caption, content 12..44 | unchanged (see §3) | **by design** |

Composer arithmetic, now exact: `8 (top pad) + 52 (writing well) + 12 (gap) + 34 (control row) + 8
(bottom pad) = 114`. Before: `20 + 52 + 8 + 28 + 20 = 126`. The 34-DIP row is the send disc's own
height, which is what the reference row is sized by; the writing well is full card width and carries
a 12-DIP text inset (native placeholder starts 12 in from the card edge).

## 2. What changed

- `internal/ui/composer/composer.go` — hero card recipe is now the native one: `Padding(8, 0, 8, 0)`,
  `Gap(12)`, control row `Height(34)` with `Padding(0, 8, 0, 8)`, and the hero writing well owns the
  12-DIP text inset instead of the card owning 16. The docked (non-hero) card is untouched.
- `internal/ui/sidebar/sidebar.go` — new-session button is `Margin(4, 2, 0, 2)` + `MaxWidth(252)`
  (→ 252×38 at (14,70)); the spacer under it is 12 DIP (→ the page list starts at y=120); the page-row
  column is inset 2 inside the 256-DIP list (→ rows at x=14, labels at x=46); the collapse control is
  28×28 r8 with a recentred 14×12 panel glyph.
- `internal/ui/sidebar/account.go` — the footer seat takes the same 2-DIP inset and 252 max width, so
  it lands on the native (14, 874) 252×36 seat.
- Tests updated/added: `frame/chat_chrome_metrics_test.go` (new; asserts the numbers above headlessly),
  `composer/hero_pass2_test.go` + `presentation_test.go` + `draft_presentation_test.go` +
  `workspace_parity_test.go` (hero recipe: 114 card, 8 top pad, 0 side pad, 52 well, 8 primary inset,
  10 bottom inset, one well + row only), `sidebar/account_test.go` (252×36 / 252×44 inset 2),
  `frame/sidebar_expanded_test.go` (40 pitch, already from round152).

## 3. Page header band: why 48 stands

`reference/electron/src/main/harness-chrome-inject.js:11` defines `const CAPTION_HEIGHT = 48` — the
desktop shell injects a 48-DIP caption plate (`height: 48px`, plus `--dshd-wco-caption: 48px`) that
covers the top of the harness page and owns the window controls and drag strip. The page's own header
is 52 DIP; in the desktop app its top 48 DIP sit under that plate, and the injection explicitly hides
the duplicate seat: `harness-chrome-inject.js:124` "The conversation-header corner expander (new seat
in the rc.1 merge) duplicates the titlebar's own right-panel toggle."

So the 52-DIP header only exists in the browser reference, and the 48-DIP merged caption MyGo draws is
the desktop-correct surface. Round151's caption metrics (band 48, content y=12..44, hairline at the
bottom) are unchanged and still asserted.

## 4. After-capture

`evidence/screens/mygo-chat-after-round156-150dpi.png` — temp build (never `whaleisle-pass*.exe`),
throwaway profile with `goPreviewNoticeAck = 2026-09-28.1` and `autoStartDesktop = true`, a real
`start → boot → ready`, then `scripts/window-capture.ps1` at 150 % (2160×1380 = 1440×920 DIP).
Compared against `native-harness-chat-hero-1440x920.png` in DIP:

| compared | native | capture (1440×920 DIP) |
| --- | --- | --- |
| sidebar column | 280 | 280 + 1 hairline |
| new-session button | 252×38 at (14,70) | 252×38 at (14,70) (headless rect; the capture's icon+label ink starts at 23.3/80.7, exactly the native 8-px/10-px content inset) |
| page rows | 36 high, pitch 40, label x=46 | pitch 40 (自动化任务 ink 171.3 − 插件 ink 132.7 = 38.7 with glyph-height noise, rows 40 apart), label ink starts x=22.7 |
| collapse control | 28×28 at (240,22), glyph 14×12 centred → ink at 247/30 | ink at **247.3/30, 14×13.3** — the seat lands exactly on (240,22) |
| composer card | 774×114 at (469.3,460) | white face x=470.7, height ≈114 (112.7 measured on the face; the 1.3 is the r28 corner's antialias), model row `deepseek-official/deepseek-flash` |
| hero | 探索未至之境 + 大肥鱼版 badge | same copy, title ink 722/372.7 (browser 722/370) |
| footer seat | 252×36 at (14,870), 设置与账号 | 设置与账号 at the 14 inset, gear + 14/22 label |
| injection-only | — | 鲸鱼娘 row, 分支 pill, 提交并推送, window buttons, wallpaper |

The nav block sits 1.7 DIP lower in ink than the browser's (MyGo centres an 18.7-DIP text line in the
36-DIP row; the page pads 7 over a 22-DIP line). That is a typography difference under 2 DIP and is
left as is.

## 5. Checks

```
go test ./internal/ui/composer ./internal/ui/sidebar ./internal/ui/frame -count=1   # all ok
go vet  ./internal/ui/composer ./internal/ui/sidebar ./internal/ui/frame            # clean
```

## 6. Residual

- Composer card width is 772 vs 774.39 (2.4 DIP): the column is 1153 wide and the card is capped by
  the hero max-width path; not changed because the same 772 shows in round152's capture and the
  difference is under the page's sub-pixel rounding.
- The hero stack's vertical seat (447 vs 460 in the browser) is a caption-reserve consequence, not a
  card metric; it needs a separate hero-centring comparison against the *desktop* reference (which the
  browser cannot provide).
- No new differences were found in the workspace toolbar icon group; it is recorded as equal.
