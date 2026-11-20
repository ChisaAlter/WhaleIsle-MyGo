# ui-compare — original web client vs MyGo native, side by side

Runs the same workbench scene twice — the original DSH web client in headless
Chrome, and the native `whaleisle.exe` in an isolated throwaway profile — then
measures the same landmarks in DIP on both, screenshots both at the same pixel
size, and reports deltas.

```
node scripts/ui-compare/run.mjs [--surface workbench] [--landmarks <path>] [--keep-open]
```

Outputs land in `tmp/ui-compare/<run-id>/` (gitignored): `summary.json`,
`report.html` (both screenshots with landmark boxes overlaid + a delta table),
and `<scenario>-web.png` / `<scenario>-native.png`. Exit code is non-zero when
any asserted landmark fails its tolerance or is missing.

## What it measures

Per landmark (`landmarks.json`): `x, y, w, h` in DIP relative to the web
viewport / native client area, within `tolerance` (default 1 DIP). The web
side's computed `border-radius`, `padding`, `font-size`, `line-height` and
`font-weight` are recorded as reference values — UIA cannot expose the native
side's equivalents — and shown beside the native crop in the report.

## How it works

- **Native**: `go build` of `./cmd/whaleisle` into a task temp root, launched
  with `WHALEISLE_ISOLATED=1 WHALEISLE_ISOLATED_VISIBLE=1
  WHALEISLE_USER_DATA=<tmp>/data` (non-activatable; `config.json` is seeded
  with a workspace so the Host starts). Driven through Windows UI Automation
  (`native.ps1`: Name + ControlType + index); resized via `SetWindowPos`
  `SWP_NOACTIVATE` to a 1280×800 DIP client; captured with `PrintWindow`.
- **Web**: a second Host is started from `runtime-rc4` (`bin.js web --no-open`,
  own `DSH_HOME` in the temp root) and its stdout `dsh web: <url>?token=…`
  is kept in memory. Headless Chrome (own `--user-data-dir`, loopback CDP,
  no npm deps) navigates to that URL, so the Host mints the `dsh-auth-*`
  cookie and redirects — the same redeem flow the embedded webview uses.
  The shell's own Host cookie is **memory-only by design** and is not
  extractable, hence the second Host; the shell's Host is left alone.
- **DPI**: `--force-device-scale-factor` is set to the native window's
  `GetDpiForWindow/96` (150 % here), so Chrome CSS px == native DIP. UIA
  `BoundingRectangle`s come back in physical pixels and are divided by the
  same scale, anchored to the window's `ClientToScreen` origin.

## Limits (v1)

- Radius/padding/typography come only from the web side (UIA exposes geometry,
  not styles); they are reported as reference values, not asserted.
- Fonts rasterize differently across the two renderers — pixel diffing the
  PNGs is informational only; the asserted numbers are geometry.
- Content-dependent views (real conversations, repos) need identical fixtures
  on both sides — v1 covers the empty workbench.
- `--theme official` is intentionally absent: switching both sides' theme
  through their real preference paths is not wired yet.
- Settings page is not a scenario: on both sides it needs two steps
  (open the account menu, then its settings item), not the one the schema
  budgets.
- Only processes the run started are stopped (tracked PIDs plus a
  command-line sweep for the run id); the temp root is removed unless
  `--keep-open`, which prints how to stop them.
