# Round 220 — DARK native launcher reference for L-HOME

Date: 2026-10-25. Task: task-96. Completes round213, whose dark half could only be scored structurally because the headless reference render was light. No product code change (no confirmed style delta), no commit, no passN, native app read-only, no `runtime/`/`third_party/` edits; temp files cleaned.

## 1. How the dark reference is produced (the theme mechanism)

- The launcher page's dark theme is a **host-set attribute**, not a media query: `reference/electron/src/shared/dsh-webui-tokens.css:80-81` — `html[data-ds-dark-theme], body[data-ds-dark-theme] { color-scheme: dark; … }` (the file is the one `launcher.html` links as `../shared/dsh-webui-tokens.css`). In the app the launcher window receives it from the injected chrome (`attachIntegratedChrome`), i.e. the attribute the shell writes after it reads the OS/theme preference.
- New script `scripts/launcher-dark-reference.mjs` reproduces that through CDP on the same page: headless Chrome with `--remote-debugging-port`, `Emulation.setDeviceMetricsOverride` 1060×660 @1, then `Runtime.evaluate` sets `data-ds-dark-theme` on **both** `<html>` and `<body>` (as the host does), waits for the deferred launcher scripts, and captures `Page.captureScreenshot`.
- Verified state at capture: `html/body data-ds-dark-theme=""`, `color-scheme: dark`, `.rail` computed background `rgb(27,27,28)`, viewport `[1060,660]`.
- Output: `evidence/screens/reference-launcher-1060x660-dark.png` (35,170 bytes, 1060×660) — dark rail/stage, the same layout and metrics as the light render, with the reference's preload placeholders (`读取版本…`, `正在读取状态…`) and only `线路` in the rail footer (no GitHub route chip without preload data).

## 2. Dark pair diff (round213 dark capture, downscaled to 1060×660)

`scripts/acceptance-diff.py --mask-mode auto` against `reference-launcher-1060x660-dark.png`:

| metric | dark (pass291) | light (round213, for comparison) |
| --- | --- | --- |
| unmasked mismatch | 71,022 / 699,600 = **10.15 %** | 8.64 % |
| wash-masked | 10,086 px (**1.44 %**) | 4,400 px (0.63 %) |
| auto-masked ratio | **8.71 %** | **8.01 %** |
| max channel delta | 254 | 255 |

Per-region (max-channel delta > 16 / > 64, ignoring the wash mask; region boxes in DIP on the 1060×660 frame):

| region | > 16 | > 64 | what differs |
| --- | --- | --- | --- |
| rail (x < 188) | 6.65 % | 3.83 % | nav label/icon anti-aliasing and the active pill wash; the 188 DIP boundary matches |
| stage header (x ≥ 188, y < 110) | 4.81 % | 2.04 % | `首页`/lede + `版本管理` text anti-aliasing |
| home card (y 110–300) | 10.44 % | 5.02 % | card copy + the `vdev` pill vs the reference's `读取版本…`; the 48 px avatar, 12 gap, eyebrow and 16/24 title sit on the same rows |
| below the card (y 300–630) | 1.45 % | 0.41 % | only the live `正在读取 MyGo 发布目录…` line + `取消下载/读取` button (a thin text band) |
| rail footer (y > 630) | 13.37 % | 8.26 % | MyGo shows `线路 GitHub ⌄` (config route), the reference only `线路` (no preload data) |

## 3. Data-only versus style

No structural or style delta is confirmed: the only regions above noise are text/anti-aliasing plus **data/state** differences the reference cannot carry without the preload bridge (version pill placeholder, status placeholder, footer route label, live catalog row). Structure was already measured live on the same capture in round213 (rail boundary 282–284 px = 188.0–189.3 DIP; nav pill top 116.7 DIP, height 40 → pitch 44), and the dark render reproduces the light render's layout exactly. **No code change, no new test.**

## 4. Recipe (both halves, reproducible)

1. Dark reference: `chrome --headless=new --window-size=1060,660 --force-device-scale-factor=1 --remote-debugging-port=9223 file:///…/reference/electron/src/renderer/launcher.html`, then `node scripts/launcher-dark-reference.mjs 9223 <file-url> <out.png>`.
2. MyGo dark capture: round213 — pass291, no-workspace throwaway profile (`workspace=""`, `autoStartDesktop=false`, `quitAfterStart=false`, `theme=deepseek.dark`, `port: 3099` as a **number**, BOM-free, `welcome-shown`, `goPreviewNoticeAck`), `scripts/window-capture.ps1` on the 1590×990 launcher window, then downscale to 1060×660 with high-quality bicubic before diffing.
3. Diff: `scripts/acceptance-diff.py <row-dir> --mask-mode auto --out diff.json` (bundled Python with Pillow).

## 5. Fixed / blocked

- **Fixed:** the missing dark reference (this round). No MyGo change was warranted.
- **Blocked / residual:** the reference still cannot carry live data (preload), so a *pixel* score below ~8 % for this row would need the native launcher running (D2) — and **D2 note**: the installed Electron app still cannot be driven to its own launcher window from this environment (a second instance focuses the running main window; the launcher is only reachable through the cold-start gate). Until that decision, L-HOME's native half is the headless render, now available in both light and dark.
- The remaining ~8.7 % dark / ~8.0 % light disagreement is the text-and-data floor described in §2, not a geometry error.

## 6. Files

`scripts/launcher-dark-reference.mjs` (new), `docs/review-20261022/evidence/screens/reference-launcher-1060x660-dark.png` (new), `docs/review-20261022/round220-lhome-dark-reference.md`. Diff summaries: `evidence/lhome-pass291-diff-dark.json` (round213, light-reference pair) — the dark pair's summary lives in this doc's table; the temp row dir was deleted with the capture scratch.
