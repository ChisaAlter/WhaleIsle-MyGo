# Round 227 — post-fix pair: hero stack verified, backdrop still tinted on MyGo

Built a temp pass295 from `a14d8cd9` (hero lift + "only an explicit Host wallpaper may override the
flat backdrop"), ran the hand-off pair at 1440×860, and scored it. **The hero stack fix is verified live.**
The flat backdrop is **not** — MyGo still paints a cyan wash, so the masked ratio remains blocked.

Artifacts: `evidence/accept-D-MAIN-227/`.

## 1. Hero stack — fixed and verified live

| element | native (UIA) | MyGo (pass295) | Δ |
| --- | --- | --- | --- |
| slogan 探索未至之境 | 314.67 | **313.33** | −1.34 |
| badge | 321.33 | 319.33 | −2.00 |
| preset row | 373.33 | **374.67** | **+1.34** |
| composer card | 409.33 (top) | **410.67** | **+1.34** |
| workspace chip x | 502 | **502** | 0 |
| preset chip x | 655.33 | 658.67 | +3.34 |
| card size | 774.39 × 114 | 774 × 114 | −0.39 |

The 19-DIP lift landed exactly as predicted (round226 expected 313.67 / 374.33 / 410.33). Chip row inset
and card recipe are untouched.

## 2. Stability and hand-off

`whaleisle://open` hand-off; UIA walks twice: native **83/83** nodes (63 named), MyGo **48/48** (43 named),
`Composer card` present, both stable. Single MyGo instance, one `MyGoWindow` handle.

## 3. Backdrops — native flat, MyGo not

| side | sidebar | hero | caption | card | cyan band |
| --- | --- | --- | --- | --- | --- |
| native | (12,12,12) | (12,12,12) | (23,28,45) | (12,12,12) | **0/9** |
| MyGo | (20,23,22) | (42,52,48) | (27,74,94) | (29,28,35) | **4/9** |

The same wash is visible in a **direct window capture** (not through the capture pipeline): sidebar
(25,25,27), hero (44,56,50), caption (34,84,107), bottom (75,67,108) — so this is the app's real paint,
not a stale DWM layer or a capture artefact.

Checks on the running profile: `goWallpaperPath` is **empty** and the profile has no appearance document
(only `config.json`, `go-resume.json`, `last-desktop-start.json`, `welcome-shown`). So neither the config
path nor a seeded image explains it.

**Most likely remaining site** (for the next product slice): `theme/first_run.go` now sets
`p.Effect = "none"` but still seeds `p.Preset = "dune"`, `p.Variant = "orbs"` and
`p.Colors = PresetColors("dune")`. If the ambient painter keys off `Preset != ""` (or off
`EffectExplicit == false`), the wash is painted even though `Effect == "none"`. The check is two log
lines at paint time (`p.Effect`, `p.Preset`, `backgroundState.active`, `backgroundState.wallpaper`) or a
grep for the painter's call site in `chrome/background.go`; then either clear the preset colours with the
flat default or gate the painter on `Effect` only.

## 4. Structure and the ratio

| space | matched | within 1 DIP | outside |
| --- | --- | --- | --- |
| window-relative | 22 | **11** | 11 |
| caption-relative | 22 | **7** | 15 |

Raw diff **99.99 %**, channel max Δ 243; regions sidebar 0.9995, header 0.9996, hero 1.0, composer 1.0,
footer 0.9993; `mask_regions` 0, so masked == raw — **still not a meaningful masked ratio**, because the
only mask that could make it meaningful would cover the whole window.

Note the leads' framing holds: even when both sides paint, the theme bases differ (MyGo #151517 vs native
#0c0c0c) and MyGo's panels are opaque where native composites glass, so the ratio will always carry that
systematic component; it must be reported separately (as a per-region mean Δ over opaque chrome) rather
than folded into the score.

Structure is unchanged at 11/22 because the paired control set does not include the hero nodes that moved
into place — they are the native-only/MyGo-only names, so the count needs a matching rule change (or the
hero pair added by alias) to reflect the fix.

## 5. Residual

1. Flat backdrop (product): the `Preset`/`Colors` seed above. Until then no masked ratio is meaningful.
2. Structure: count the hero nodes by alias so the verified fix shows up in the within-1-DIP number.
3. Badge height native 15.33 vs MyGo 20 (cosmetic), preset chip width 106.67 vs 100.

## 6. State

Temp build stopped and deleted, both temp profiles removed, native restored to (200,2) 2268×1366 on the
empty 新会话.
