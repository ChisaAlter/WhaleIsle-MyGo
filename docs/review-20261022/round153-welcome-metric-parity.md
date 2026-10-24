# Round 153 — welcome window metric parity (G04/G05 follow-up)

Date: 2026-10-25. Task: task-27. Scope: `internal/ui/welcome/`, this doc and its screen. No commit, no passN exe, native app untouched.

## 1. Method

The native welcome page cannot be screenshotted statically (round147: `welcome.html` is an empty `#root` filled by `welcome/welcome.js` through the preload bridge), so the pass is metric-based: every number below is read from `reference/electron/src/renderer/welcome-layout.css` (the page's layout authority, imported by `welcome.html`), then measured in MyGo with `ui.Tester` at 600×700 and compared with the live capture `evidence/screens/mygo-welcome-pass275-150dpi.png` (900×1050 px = 600×700 DIP at 150%).

## 2. Metric table

| metric | native CSS | MyGo before | MyGo after |
| --- | --- | --- | --- |
| window size / radius | 600×700 (welcome-window.js), 20 | 600×700, 20 | equal (unchanged) |
| `#root` padding / gap | `6px 16px`, gap 8 | `16 24 20 24`, gap 12 | `6 16 6 16`, gap 8 |
| `.titlebar` | 36px, drag only (no logo/name) | 24px row with fish logo + name | 36px empty drag strip |
| `.brand` | 40px block in grid row 1 | none (logo lived in the caption) | 40px row: fish logo 27×20 + name 12 muted |
| `.welcome` padding | `24px 48px 76px` | none (root padding only) | `24 48 76 48` |
| grid rows | `40px 1fr 104px` | flex column with gaps | grow row + fixed 40 / 104 |
| `.tagline` (sentence) | 24/40 weight 300, secondary | separate 13px line | 24/40 weight 300 secondary |
| `.tagline em` (product) | 26/40 weight 500, primary | 26 weight 500 primary (equal) | 26/40 weight 500 primary |
| `.actions` box | 240 wide, gap 16, row 3 fixed 104 | 280 wide, gap 8 | 240 wide, gap 16 |
| `.actions button` | height 44, radius 10, 1px transparent border | height 40, radius 10 | height 44, radius 10 |
| `.actions .secondary` | l2 border + elevated fill | transparent + border | unchanged (border l2, transparent fill; the native elevated fill is the window tint, see §4) |
| `.actions .back` (skip/back) | absolute 16px below the buttons, 28 tall, min 58, radius 24, text only | full-width 32 tall row inside the column | own 28px row, 16 below, min 58, radius 24, text only |
| `.key-form` | 440 wide, gap 40 | 280 wide column | 440 wide, gap 40 |
| `.key-heading` | gap 8; h1 20/32 w400, ls .01em; p 16/22 w300, ls .02em | h1 15, p 12, no line boxes | 20/32 and 16/22 with the tracking |
| `.key-field input` | height 48, radius 12, border l4, 14/24, padding 7 14 | height 40, radius 10 | 48, radius 12 |
| `.key-error` | 14/22 under the field | 12px note | 14/22 |
| save button | `.action .primary` in the actions row | inside the key column | actions row, primary |

Measured on the after render (`ui.Tester`, 600×700): tagline line box 40, sub line box 40, button label pitch 60 (= 44 + 16), skip label centre 54.7 below the last button label centre (22 + 16 + 14 plus the label's intrinsic box), key heading line box 32, key description 22.

## 3. Fixed

`internal/ui/welcome/welcome.go` rewritten to the reference grid: `#root` padding/gap, an empty 36px `.titlebar` with the brand moved to its 40px row, the `.welcome` padding and 40/1fr/104 rows, the `.tagline` type pair (26/40 `em` over 24/40 sentence), the 240×44/radius-10 `.actions` column with 16 gaps, the 28px `.actions .back` text action 16 below, and the `.key-form` (440 wide, 40 gap, 20/32 + 16/22 heading, 48/radius-12 input, 14/22 error) with its save button in the actions row.

Behaviour is unchanged: same model, same callbacks, same labels. Two adjustments were needed to keep everything inside the fixed 104px actions row: the running sign-in's copy/cancel pair now replaces the `.back` action, and the key page's "later" shortcut is gone (the native key page has the save button plus `.back` only); `welcome_test.go` was updated for the removed label.

Tests: `metrics_test.go` — `TestWelcomeLayoutMetrics` (line boxes, 44+16 pitch, `.back` offset), `TestWelcomeKeyFormMetrics` (20/32 heading, 16/22 line, save action present) and `TestWelcomeMetricConstants` (pins the 18 ported CSS numbers).

## 4. Blocked / inferred (documented, not silently guessed)

- **`.brand` composition**: the CSS gives the 40px box but the element's content comes from the bundled, minified `welcome/welcome.js` (the page is React-built), so I kept MyGo's existing brand content (fish logo + product name) inside the 40px row instead of inventing native art.
- **`.tagline p` size**: `.tagline h1, .tagline p { font: inherit }` inherits the block's 24/40 weight 300, so the sentence is 24px in the after capture. That is CSS-faithful but large; if the installed page overrides the paragraph elsewhere it is hidden inside the bundle and needs a captured native window to confirm.
- **`.actions .secondary` fill**: the native `--dsw-alias-button-elevated-fill` is white on the welcome tint; MyGo keeps the transparent background with the l2 border, so the outlined look is preserved but not the exact fill token.
- **`--welcome-tint` material**: MyGo paints the theme background instead of the native `window-material.css` tint/blur (a chrome material concern owned outside this package).
- Native pixel proof of the whole page still needs a native welcome capture (Electron window: never PrintWindow, per the capture tool's rule).

## 5. Evidence

- Before: `docs/review-20261022/evidence/screens/mygo-welcome-pass275-150dpi.png` (900×1050).
- After: `docs/review-20261022/evidence/screens/welcome-after-600x700.png` — temp build `.devdata\pixel3\whaleisle-task27.exe` (deleted, pass277 untouched), temp profile without the marker, `scripts/window-capture.ps1`, 900×1050 px @144 dpi = 600×700 DIP.
- `go test ./internal/ui/welcome ./internal/ui/frame -count=1` → ok; `go vet ./internal/ui/welcome` → clean.
