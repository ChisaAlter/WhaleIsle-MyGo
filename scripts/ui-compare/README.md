# ui-compare — original web client vs MyGo native, side by side

Runs the same workbench scene twice — the original DSH web client in headless
Chrome, and the native `whaleisle.exe` in an isolated throwaway profile — then
measures the same landmarks in DIP on both, screenshots both at the same pixel
size, and reports deltas.

```
node scripts/ui-compare/run.mjs [--surface workbench|chat|settings] [--landmarks <path>] [--keep-open]
```

Outputs land in `tmp/ui-compare/<run-id>/` (gitignored): `summary.json`,
`report.html` (both screenshots with landmark boxes overlaid + a delta table),
and `<scenario>-web.png` / `<scenario>-native.png`. Exit code is non-zero when
any asserted landmark fails its tolerance, is missing, or a scenario's setup
fails (reported as `ERROR`, distinct from a landmark `FAIL`/`MISS`).

## What it measures

Per landmark (`landmarks.json`), in DIP relative to the web viewport / native
client area, within `tolerance` (default 1 DIP):

- `x`, `y`, `w`, `h` — absolute geometry.
- `dy-prev` — this landmark's top minus the *previous* landmark's bottom (the
  gap), so a local spacing difference lands on the boundary where it happens
  instead of cascading down a stack.
- `dx-parent-left` — left inset from a named container landmark
  (`"parent": "<key>"`).
- `dx-parent-right` — right inset from the parent's right edge.

A landmark may carry `"probe": true` when it exists only to prove the
workbench is up before steps run (e.g. a sidebar button that a modal then
covers); probes are found but never measured.

## Steps are ordered and verified

Scenarios list `web.steps`/`native.steps` in order; each step may carry an
`expect` spec (`{"name": "...", "type": "...", "within": {...}}`) that must
resolve to a visible element before the next step runs — a failed
expectation aborts the scenario as `ERROR` before landmarks are measured.
The `settings` surface uses this to click the rail cells
(`[data-dsh-settings-section]` / the native nav Buttons) so each scenario
lands on a verified page. `nativeDump: true` on a scenario also writes the
native UIA tree to `<scenario>-uia.json`; `domDump` writes the web DOM for
the same purpose.

The web side's computed `border-radius`, `padding`, `font-size`,
`line-height` and `font-weight` are recorded as reference values — UIA cannot
expose the native side's equivalents — and shown beside the native crop in
the report.

## Setup is verified

`web.setup`/`native.setup` entries are not best-effort: each `dismiss` step
waits until its `gone` element is actually absent (with a quiet period that
catches a second dialog appearing behind the first), retrying the candidate
dismiss targets in order. `optional` only excuses a target that never
appeared; a still-present element is a scenario `ERROR`. The web's first-run
chain — `预览版说明`, then `开始使用` — is dismissed through
`[role=dialog] button[aria-label='关闭']` (×) with the `_primary` button as
fallback, ordered so the `开始使用` dialog's `登录` button (also `_primary`)
can never be clicked. The native isolated profile shows no first-run overlay
(`offerFirstRunImport` is skipped for isolated runs), so it needs no dismiss.

## Fixture equalization

Both sides must show the same workspace. The native profile's
`config.json` `workspace` is pointed at one task-owned directory named
`默认工作区`, and Host #2 is spawned with that same directory as its cwd —
the Host uses the child cwd as its workspace and the web client's
`initializeDefault` labels it `默认工作区`, while the native sidebar shows
the directory's basename. One directory, one name, both products' real paths.
Product-specific rows (native `鲸鱼娘`/`Project`/`初始化 Git`/mic, web `更多`,
extra native tree rows like `在“…”中新建会话`) are legitimate feature
differences and are simply not mapped as landmarks.

## How it works

- **Native**: `go build` of `./cmd/whaleisle` into a task temp root, launched
  with `WHALEISLE_ISOLATED=1 WHALEISLE_ISOLATED_VISIBLE=1
  WHALEISLE_USER_DATA=<tmp>/data` (non-activatable). Driven through Windows
  UI Automation (`native.ps1`: Name + ControlType + index); resized via
  `SetWindowPos` `SWP_NOACTIVATE` to a 1280×800 DIP client; captured with
  `PrintWindow`.
- **Web**: a second Host is started from `runtime-rc4` (`bin.js web --no-open`,
  own `DSH_HOME` in the temp root, cwd = the shared workspace dir) and its
  stdout `dsh web: <url>?token=…` is kept in memory. Headless Chrome (own
  `--user-data-dir`, loopback CDP, no npm deps) navigates to that URL, so the
  Host mints the `dsh-auth-*` cookie and redirects — the same redeem flow the
  embedded webview uses. The shell's own Host cookie is **memory-only by
  design** and is not extractable, hence the second Host; the shell's Host is
  left alone.
- **DPI**: `--force-device-scale-factor` is set to the native window's
  `GetDpiForWindow/96` (150 % here), so Chrome CSS px == native DIP. UIA
  `BoundingRectangle`s come back in physical pixels and are divided by the
  same scale, anchored to the window's `ClientToScreen` origin.

## Limits (v1)

- Radius/padding/typography come only from the web side (UIA exposes geometry,
  not styles); they are reported as reference values, not asserted.
- Fonts rasterize differently across the two renderers — pixel diffing the
  PNGs is informational only; the asserted numbers are geometry.
- Content-dependent views need identical fixtures on both sides. The `chat`
  surface covers this: `seed-chat.mjs` writes the static fixture
  (`fixtures/chat-basic.json`) as a real `session.v4.jsonl.zstd` log into
  BOTH DSH homes (the isolated native profile's and host2's `DSH_HOME`)
  before either side launches, and both sides open that seeded session — no
  model calls, no credentials. Member-level disclosure rows can only be
  measured expanded if the product exposes a working UIA ExpandCollapse
  action on them.
- `--theme official` is intentionally absent: the native side is seedable via
  `goAppearance` in config.json, but the web side's theme lives in the Host
  settings store and switching it through the real client path (settings API
  or settings UI) is not wired yet.
- The `settings` surface opens the modal via the real entry on both sides
  (account menu → settings). Its rail is not compared group-for-group: the
  native rail carries extra groups (`账户`, `功能`) that the installed web
  build does not, so only the shared cells and per-cell rhythm are mapped.
- Only processes the run started are stopped (tracked PIDs plus a
  command-line sweep for the run id); the temp root is removed unless
  `--keep-open`, which prints how to stop them.
