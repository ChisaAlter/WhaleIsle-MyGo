# Round 192 — seeded fixture on pass287: what the fixture needs, and what blocked the row

Attempted the seeded-fixture acceptance pair (layout reset + wallpaper seed + pass287 at 1440×860).
**The pair was not produced**, but the fixture's three real unknowns were resolved and the exit that
blocked rounds 194/195 is explained. No product code, no commit, no passN; native restored, temp cleaned.

## 1. `quitAfterStart` — the reason a "minimal profile" opens and dies

A throwaway config written with only `host/port/workspace/theme/locale/autoStartDesktop/ack` gets
**rewritten on first start with `"quitAfterStart": true`** (observed in the rewritten
`<profile>/config.json`). The app then opens its window and **exits after ~50 s** — exactly the
549×460-then-gone behaviour reported for pass286. Two launches reproduced it (with and without
`goWallpaperPath`), and with `"quitAfterStart": false` the same profile **stayed alive** (verified at
45 s). This is a fixture field, not a crash: `quitAfterStart` is the smoke-profile's "start the desktop
then quit" switch, and it is on by default when the key is absent.

**Fixture recipe addition:** set `"quitAfterStart": false` explicitly. Tasks that lost windows after
~50 s were hitting this, not a wallpaper or harness failure.

## 2. The persisted transcript layout is `<profile>/go-resume.json`

The profile gains `go-resume.json` (4.8 KB) on first start and it carries `layout` — including the
`TranscriptWidth` that made the composer 773.33 instead of 774.39 (round186). **Fixture recipe:** do
not seed from the real profile, and delete `go-resume.json` (or zero its `layout.TranscriptWidth`)
before the capture run; with no preference the axis falls back to `0.64 × column = 742.4` and the card
is 774.4.

Also worth knowing: a minimal `config.json` is **expanded with defaults** on start (all the
`remote*`, `harness*`, `go*` fields appear), so the fixture must re-read the file after launch rather
than assume what it wrote survived.

## 3. The window it opens is the launcher, not the chat

With `quitAfterStart: false` and `autoStartDesktop: true`, the visible `MyGoWindow` is the **launcher**
(首页/组件/版本/导入/插件排查/设置, "桌面端已就绪", "关闭桌面端", and the catalog warning) — the capture
therefore produced a launcher frame at 1440×860 (2160×1290), which is not D-MAIN. Raw diff on that pair:
67.25 % mismatch (down from 99.99 % only because both sides are flat dark), channel max Δ 243 — not a
usable acceptance number, reported only to show the capture path worked.

**Next fixture step:** the row needs the chat window. Either the launcher must be dismissed/closed
(find the marker or the "进入桌面端" path that focuses the chat window), or the profile must be seeded
so the chat window is the one that opens (as pass283 did when seeded from the real config), or the
capture must target the second `MyGoWindow` after the launcher hands off. This is the single remaining
fixture blocker.

## 4. Wallpaper seed: no on-disk data URL

The native `ui-theme.wallpaperImage` data URL is not in any text store of the user's `dsh-home`
(searched 400 JSON/DB/log files by content, plus the `storages/` tree — no `wallpaperImage` hit); the
harness keeps it in a non-text client store. The harness page served from a scratch `DSH_HOME` has
`#dsh-wallpaper` present but with **no background image**, confirming the value is per-profile data, not
a token. For the capture I used a crop of the native wallpaper from the baseline image
(`tmp/native-wallpaper-crop.png`, 1080×420, temporary, not committed) as `goWallpaperPath`; it never
reached the pair because the launcher window was captured instead.

**Decision needed:** either (a) read the value from the running native app through its own CDP/devtools
endpoint, (b) accept a crop of the native capture as the fixture wallpaper (approximate palette), or
(c) keep the wallpaper as a registered wash mask and stop trying to seed it. My recommendation is (c)
plus (a) later: the wash mask is what the contract's mask registry was for, and the seed only helps
visual review.

## 5. Numbers this round

| item | value |
| --- | --- |
| capture produced | 2160×1290 baseline + actual (launcher frame), geom/controls/mask/accept-log |
| raw mismatch on that (non-)pair | 67.25 %, channel max Δ 243, mask regions 0 |
| structure / caption-relative rescore | **not run** — the pair is not D-MAIN, so the rescores would be meaningless |
| single-instance assert | 0 other `whaleisle*` processes before start; one `MyGoWindow` during |
| process lifetime | alive at 45 s with `quitAfterStart:false`; exited at ~50 s with the default |

## 6. State

pass287 process stopped, both temp profiles deleted, harness web stopped, native window restored to
(200,2) 2268×1366 and left on the empty `新会话`. Artifacts from the launcher capture live only in
`%TEMP%\whaleisle-accept-2026-10-24\D-MAIN-287` and were **not** copied into `evidence/accept-D-MAIN`
(deliberately: they are not a D-MAIN pair).
