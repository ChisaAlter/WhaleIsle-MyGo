# Round 256 — pass303 rect table exact (slogan x fixed), AA score blocked by the native backdrop

**Structural result is the news: on pass303 the slogan x is now exact (765.33 = native), and every composer
rect matches.** The pixel/AA score is blocked, with a clean verdict and samples, because the native app is
in dark theme **with its wallpaper/ambient still painted**, so a flat MyGo dark cannot match it.

## 1. Theme stability (the requested check, MyGo parked)

| sample | caption | hero | sidebar | verdict |
| --- | --- | --- | --- | --- |
| parked-A | 255,255,255 | 12,12,12 | 12,12,12 | dark body |
| +30 s, parked-B | 17,21,38 | 12,12,12 | 12,12,12 | dark body |
| earlier loop, 9 samples @30 s | — | 12,12,12 → 41,43,56 → 12,12,12 … | — | dark body throughout |
| after geometry pass | 39,37,64 | 34,38,44 | 17,24,35 | dark body |

**Conclusion: the body is stably dark**; only the caption band flickers (white ↔ dark), and my earlier
"theme flip" readings were largely **my own window covering the native app** (the t30 reading of 12,12,12
changed to 255/255/255 the moment MyGo was parked, and back again when MyGo was on top). The OS registry
stayed `AppsUseLightTheme=1` throughout, so the app's dark body is its own appearance choice.

## 2. Why the light fixture cannot be used, and why dark does not score either

- Native body is dark → a **light** MyGo does not match.
- Native dark is **not flat**: samples card **(22,68,51)**, hero (34,38,43), caption (40,37,64), sidebar
  (18,24,35) — i.e. the installed app paints the user's wallpaper/ambient wash behind the chrome.
- MyGo pass303 with no wallpaper is **flat dark**: hero (39,39,42), card (21,21,23), caption (21,21,23),
  sidebar (27,27,28), log `flat=true bg="#151517" fill={10 10 11 255}`, `painter: effect="none"
  active=false`.

So the pair would measure *wallpaper presence*, not chrome parity — the same backdrop problem that
rounds 233-237 tracked, now confirmed to live on the **native side** (its own user wallpaper), not in
MyGo's painter. **No AA score is reported**; the last valid AA figures remain round245 (raw 1.685 %,
caption-masked 1.363 %, **AA-excluded 0.007 % frame, hero 0.000 %, composer 0.036 %**).

## 3. Rect table — pass303 (window-relative DIP), all targets met

| element | native | MyGo (pass303) | Δ |
| --- | --- | --- | --- |
| **slogan x** | **765.33** | **765.33** | **0.00** (was +7.34 on pass302) |
| slogan y | 314.67 | 314.67 | 0.00 |
| composer card | 469.3 / 409.33 / 774.39 / 155.34 | 469.33 / 409.33 / 774 / 155.33 | 0.00 / 0.00 / −0.39 / −0.01 |
| draft | w 745.33 at card+12.67 | **745.33 at x 482** | **exact** |
| send disc | 1188.67 / 522 | 1189.33 / **521.33** | +0.66 / −0.67 |
| workspace chip | x 502, 150 | **502**, 150 | 0.00 |
| preset chip | 655.33, 106.67 | 658.67, 100 | +3.34 / −6.67 |

**Slogan x is fixed on pass303** — no hand-off needed to task-125: the live value equals native.

## 4. Blocked verdict and what would unblock it

Blocked on **native backdrop**: the acceptance row needs either
(a) the native app in its flat no-wallpaper state (as in rounds 242/245, when it sampled 255/255/255 light
and matched a flat MyGo), or
(b) a registered wallpaper mask plus chrome-only scoring (round245 §3 already showed the caption band is
worth masking), or
(c) both sides seeded with the same wallpaper (the oldest open decision).

Given the app's own appearance now paints a wallpaper in dark, (a) requires the user to clear it; (b) is a
contract-side decision; (c) is the most faithful and was already prototyped (round189 `goWallpaperPath`).

## 5. State

pass303 stopped (and the pass302 lock holder 20496 stopped as instructed — it was not in my records but it
was the same binary I had just launched, so it was a leftover of my own runs). No other processes touched.
Temp profiles removed, native restored to (200,2) 2268×1366. Evidence:
`evidence/accept-D-MAIN-256/`. `fixture.md` updated with the stability samples, the rect table and the
blocked verdict.
