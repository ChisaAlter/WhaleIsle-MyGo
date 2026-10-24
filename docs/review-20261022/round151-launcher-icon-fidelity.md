# Round 151 — launcher nav icons and brand fidelity (G04 follow-up)

Date: 2026-10-25. Task: task-25. Scope: `internal/ui/launcher/`, `internal/ui/i18n/i18n.go`, this doc and its screens. No commit, no passN exe.

## 1. Nav icons — native SVG paths ported (fixed)

`launcher.html:26-50` draws the six tabs from exact 16×16 `<path d="…">` data (one `fill="currentColor"` path each); MyGo drew hand-drawn approximations (`iconPath`), with extra stroke seams for the cube and versions glyphs.

- New `internal/ui/launcher/svgpath.go`: the six `d` strings copied **verbatim** into `navIconPaths`, plus an SVG path parser (`M/L/H/V/C/S/Q/T/A/Z`, absolute and relative, compact number forms like `.9.9` and `0-.92`) that flattens arcs to cubic segments the way browsers do, and emits into `ui.Path` (`MoveTo/LineTo/CubeTo/QuadTo/Close`). No approximate glyphs remain.
- `tabIcon` (icons.go): draws `FillPath(svgIcon(d, …))` clipped to the 16 DIP box — the SVG viewport's clip, which the settings path needs because the launcher.html data carries a few coordinates past 16 (17.07). `iconPath` deleted. `routeChevron` and `globeIcon` are unchanged (separate glyphs, not part of the six).
- Before: `evidence/screens/mygo-launcher-pass276-150dpi.png` (versions read as three loose lines, components/settings were strokes/coarse gear).
  After: `evidence/screens/launcher-task25-icons-150dpi.png` (temp build, real window, 1590×990 = 1060×660 DIP): filled house, faceted cube, framed list, tray arrow, puzzle piece and the native gear, matching `reference-launcher-1060x660.png`.
- Tests (`internal/ui/launcher/icons_test.go`): `TestNavIconsAreTheNativePaths` (all six parse, ≥4 segments each, every emitted point inside the 16-unit viewport ±1) and `TestNavIconsDrawInsideTheIconBox` (the painter actually receives the glyphs: >150 opaque pixels).

## 2. Brand avatar — already the native asset (verified, no change)

`railHead` (36) and the home product row (48) both draw `brandAvatar()` → `chrome.BrandAvatar()` → `//go:embed assets/whale-head.png`.

- `sha256` comparison: `internal/ui/chrome/assets/whale-head.png` = `reference/electron/assets/whale-head.png` (`DEB725BF…1BA7`, 1,281,699 bytes, identical). No hand-drawn avatar remains; the drawn fallback is the existing `ui.Box` circle when the bitmap fails to decode.

## 3. Home product title — native copy (fixed)

`launcher.html:68` static title is `鲸屿桌面端`; the zh catalogue had `Whale Isle 桌面端`.

- `internal/ui/i18n/i18n.go` `launcher.desktop.title` zh: `Whale Isle 桌面端` → `鲸屿桌面端`. The en string stays `Whale Isle Desktop`: the native launcher ships the zh static page only (`launcher.html`/`launcher.js` have no en title source), so there is nothing to copy from.
- The visible title now matches the reference render; no i18n test asserts the old text.

## 4. Update-error line (blocked, documented)

Under the card, MyGo prints the raw update-check error (this run: `MyGo release catalog HTTP 404; a compatible published native installer is required`), where the native launcher shows a short localized line (`更新未完成：…`). The text is produced by the update-check path (`internal/desktop/update`, `state.launcherUpdate` formatting), not by `internal/ui/launcher`: the panel only renders the string it is given. Mapping those errors to localized copy belongs to the update owner; the launcher side has no way to know which failure it is beyond the message.

## 5. Verification

- Temp build `.devdata\pixel2\whaleisle-task25.exe` (deleted; pass276 untouched), temp profile with `welcome-shown`, no workspace, `scripts/window-capture.ps1` → 1590×990 px @144 dpi = 1060×660 DIP, compared with `reference-launcher-1060x660.png` (headless Chrome, 1060×660 DIP).
- Icons and the home product row now match the reference; remaining launcher diffs are the update-error text (§4) and content-only values (version pill `vdev`).
- `go test ./internal/ui/launcher ./internal/ui/welcome ./internal/ui/i18n -count=1` → ok; `go vet` on the same three packages → clean.

## 6. Files

`internal/ui/launcher/svgpath.go` (new), `internal/ui/launcher/icons.go`, `internal/ui/launcher/icons_test.go` (new), `internal/ui/i18n/i18n.go`, `docs/review-20261022/round151-launcher-icon-fidelity.md`, `docs/review-20261022/evidence/screens/launcher-task25-icons-150dpi.png`.
