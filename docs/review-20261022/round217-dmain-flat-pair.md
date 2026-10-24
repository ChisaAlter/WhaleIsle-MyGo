# Round 217 — settled flat D-MAIN pair: structure 11/22, wallpaper still on MyGo

The hand-off pair is now **structurally scored on a stable walk**: window-relative **11/22 within 1 DIP**
(was 0/22) and caption-relative **7/22**. The composer card is at its native size. The remaining blocker
is that MyGo still paints the aurora gradient, so the pixels are not comparable.

Artifacts: `evidence/accept-D-MAIN-217/` (baseline, actual, diff, diff.json, geom, controls, structure,
mask, accept-log). Build: pass292, git `69988cec` (in-tree, includes `32ed319c` flat-base fix).

## 1. Structure (the deliverable that moved)

| space | matched | within 1 DIP | outside | native-only | MyGo-only |
| --- | --- | --- | --- | --- | --- |
| window-relative | 22 | **11** | 11 | 37 | 27 |
| caption-relative | 22 | **7** | 15 | 37 | 27 |
| absolute (screen) | 22 | 0 | 22 | 37 | 27 |

The improvement came from the hand-off capture order (`whaleisle://open`, then native-first capture with
the repaint nudge) giving a stable UIA walk on both sides — the earlier 0/22 was an artefact of walking
the native tree mid-switch. Window-relative is the honest number for same-size windows (both framed
differently by their own captions); caption-relative penalises the hero block, whose native rows sit
0.67 DIP higher, so it is the stricter of the two.

## 2. Live geometry vs native (DIP)

| element | native | MyGo | Δ |
| --- | --- | --- | --- |
| composer card | 774.39 × 114 at (469.3, 430) | **774 × 114 at (469.33, 429.33)** | width **−0.39** ✓ (inside ±0.6), y −0.67 |
| workspace chip | 150 × 28.67 at (502, 373.33) | 150 × 28 at (**489.33**, 393.33) | x **−12.67**, y +20 |
| 标准模式 chip | 106.67 × 28.67 at (655.33, 373.33) | 100 × 28 at (**675.33**, 393.33) | x **+20**, y +20 |
| caption triad | x 1336/1368/1400, y 8 | x 1336/1368/1400, y 8.67 | **x exact**, y +0.67 |

So the chip row needs two separate corrections, both measured: the row's **left inset** (MyGo 20 DIP
inside the card, native 32.7 DIP) and the **gap between the chips** (MyGo 36 DIP, native 3.3 DIP —
`655.33 − (502+150)`). The earlier DOM probe's 638.3 is a different box (text span, not the chip), so the
UIA numbers above are the ones to fix against. Not fixed in this round: the pixel pair is unstable, and
changing the row without a comparable pair would be guessing.

## 3. Wallpaper: MyGo still paints the aurora (blocker)

Both captures sampled:

| side | sidebar | hero | footer band | card interior |
| --- | --- | --- | --- | --- |
| native | (12,12,12) | (12,12,12) | (19,23,31) | (12,12,12) |
| MyGo | (25,25,27) | (41,46,46) | (**33,130,145**) | (36,35,44) |

Native is flat; MyGo still shows the aurora (the cyan footer band is the tell). Two attempts:

1. No `goWallpaperPath` at all → aurora (round 210/217 first run).
2. `"goWallpaperBackground": {"backgroundEffect":"none", …}` in the throwaway config → **still aurora**.

The effect therefore does not come from the config path: `internal/ui/theme/host_preferences.go:49` seeds
`BackgroundEffect: "gradient"` with aurora colours, and `internal/ui/theme/first_run.go:23` sets
`p.Effect = "gradient"`. `internal/ui/chrome/background.go:96` treats `p.Effect == "gradient"` as an
active background even with no wallpaper path, so `32ed319c`'s flat-base fix never runs for a first-run
profile. **That is the remaining product gap**: a profile with no wallpaper *and* no explicit host
background must paint the flat theme base, not the gradient — i.e. the default should be "none" unless
the host/user asks for an effect (native's `cordis.patch.yml` has no `wallpaperImage` and no gradient).

Consequence: raw diff 99.99 %, regions sidebar 0.9996, header 0.9996, hero 1.0, composer 1.0, footer
0.9993 — all dominated by the gradient, so the wash-masked ratio is not yet meaningful.

## 4. Residual

1. **Default background** (product): make "no wallpaper + no explicit effect" paint flat; keep the
   gradient only when the host sets it. Then the pair becomes comparable and the masked ratio lands.
2. **Chip row**: apply the two measured corrections (left inset 20 → 32.67, chip gap 36 → 3.3) with a
   headless test at 1440×860 once the pair is stable.
3. **Caption-relative 7/22**: re-check after (1) — the hero rows are the remaining cluster.

## 5. State

pass292 stopped, both temp profiles deleted, native restored to (200,2) 2268×1366 on the empty 新会话.
