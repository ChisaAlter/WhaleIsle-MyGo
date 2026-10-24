# Round 223 — flat D-MAIN pair attempt on pass294: geometry all green, backdrop still tinted on MyGo

Captured the hand-off pair on pass294 (git `0322e73d`) with a no-wallpaper throwaway profile. Every
geometry target the last rounds chased is now **met**; the pixels are still not comparable because MyGo
paints a cyan backdrop where native paints flat dark. Artifacts: `evidence/accept-D-MAIN-223/`.

## 1. Stability and flatness checks (as requested)

| check | result |
| --- | --- |
| UIA stability, two walks | native **83 / 83** nodes (63 named) · MyGo **48 / 48** (43 named) — both stable, `Composer card` present on MyGo |
| hand-off | `whaleisle://open` after ~42 s; chat asserted before capture |
| native flatness | caption (17,31,48) · sidebar (65,65,65) · hero (12,12,12) · card (12,12,12) — flat, no cyan band (0/9 cyan samples across the footer band) |
| MyGo flatness | caption (**35,92,119**) · hero (42,48,46) · footer (**33,145,163**) · card (27,27,28) — **cyan present, 3/9 samples** |

## 2. Geometry: all targets met

| element | native (DIP) | MyGo (DIP) | Δ |
| --- | --- | --- | --- |
| composer card | 774.39 × 114 at (469.3, 430) | 774 × 114 at (469.33, 429.33) | **−0.39** ✓ |
| workspace chip | 150 × 28.67 at (502, 373.33) | 150 × 28 at (**502**, 393.33) | **x exact** ✓ |
| 标准模式 chip | 106.67 × 28.67 at (655.33, 373.33) | 100 × 28 at (**658.67**, 393.33) | **+3.34** (was +20) ✓ |
| caption triad | x 1336/1368/1400, y 8 | x 1336/1368/1400, y 8.67 | x exact, **+0.67** ✓ |
| sidebar nav (插件/自动化任务/鲸鱼娘) | y 120/160/200, 252×36 | y 120.67/160.67/200.67, 252×36 | **+0.67** ✓ (<1 DIP) |

The chip row fixes from task-98 land exactly as specified (inset 32.67 → x=502; gap 3.33 → +3.34 on the
preset chip). The remaining +20 DIP on the whole preset row is vertical (MyGo y 393.33 vs native
373.33) and is the hero-block offset, not the chip row.

## 3. Structure

| space | matched | within 1 DIP | outside | native-only | MyGo-only |
| --- | --- | --- | --- | --- | --- |
| window-relative | 22 | **11** | 11 | 37 | 27 |
| caption-relative | 22 | **7** | 15 | 37 | 27 |
| absolute | 22 | 0 | 22 | 37 | 27 |

Same as round217 — expected, since round217's pair also had a stable walk. The clusters behind the 11
misses are the preset row (+20 DIP vertical, i.e. hero block) and the hero title/badge block.

## 4. Masked ratio: still not meaningful

Raw diff **99.99 %**, channel max Δ 243; regions sidebar 0.9995, header 0.9996, hero 1.0, composer 1.0,
footer 0.9993; `mask_regions` 0, so the wash mask has nothing registered and the "masked" number equals
the raw one. A masked ratio is only worth computing once the backdrop matches — otherwise the mask would
have to cover the whole window.

## 5. Why MyGo still tints (product finding, exact site)

Both defaults are already flat in the tree:

- `internal/ui/theme/first_run.go:20-26` → `p.Effect = "none"`;
- `internal/ui/theme/host_preferences.go:53` → `BackgroundEffect: ""`.

But `internal/ui/chrome/background.go:96` activates the backdrop on **either** condition:

```go
backgroundState.active = path != "" || p.Effect == "gradient"
```

With `Effect = "none"` the only way to be active is a **non-empty wallpaper path**. So the aurora-like
pixels must come from a seeded wallpaper *image* path (a built-in preset image written into the profile),
not from the gradient effect. That is the next two-line probe: log `chrome.backgroundState.wallpaper`
and `p.Effect` at paint time, and if the path is non-empty, find who seeds it (profile seeding vs
`wallpapers.Defaults()`), then either clear it for a no-wallpaper fixture or stop seeding it on first
run. Everything else on this row is already within its tolerance.

## 6. Residual

1. Seed/probe the wallpaper path (above). After that, re-capture → first meaningful masked ratio.
2. Hero block +20 DIP vertical (preset row 373.33 native vs 393.33 MyGo) — the round181 headline-foot
   change moved the preset row; with the flat pair this is the next number to fix.
3. Caption-relative 7/22 will improve with (1)+(2); clusters are the hero rows.

## 7. State

pass294 stopped, temp profile deleted, native restored to (200,2) 2268×1366 on the empty 新会话.
`fixture.md` updated with the round223 status (hand-off, geometry, structure, the backdrop finding).
