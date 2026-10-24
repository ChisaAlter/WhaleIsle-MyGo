# Round 233 — opaque-canvas verification on pass297: the transparency theory is refuted, the wash is painted

Goal: verify the live opaque canvas on pass297 and capture the definitive flat pair. **The surface is not
flat, so no pair and no masked ratio are reported** (as instructed). What this round does establish is
that the round230 "transparent window composites the desktop" hypothesis is **wrong** — the tint is
painted by the app itself.

## 1. Method

- Lock check: no `whaleisle*` process before start; a fresh throwaway profile (dark, zh, ack,
  autoStartDesktop, `quitAfterStart=false`, temp workspace, `welcome-shown`, **no** `goWallpaperPath`,
  no `go-resume.json`).
- `whaleisle-pass297.exe` started with `WHALEISLE_USER_DATA=<profile>`, waited 42 s,
  `whaleisle://open` hand-off, UIA asserted `探索未至之境`/`Composer card`, then the client was set to
  1440×860 DIP (2160×1290 px) and **sampled directly from the screen** (not through the capture
  pipeline).

## 2. Direct live samples (RGB, pass297)

| point | sample | expectation |
| --- | --- | --- |
| hero | **(43,54,50)** | 12,12,12 |
| hero (second point) | **(39,84,57)** | 12,12,12 |
| card interior | **(27,27,29)** | 12,12,12 |
| card interior (second) | (26,26,28) | 12,12,12 |
| caption band | **(37,100,129)** | token composite (~19,43,61) |
| sidebar | (25,25,27) | token fill |
| bottom band | **(58,52,80)** | token fill |
| cyan band | **3/9** | 0/9 |

So the hero still carries a green/teal wash, the bottom band a purple one, and the caption a strong blue —
the flat canvas is **not** visible anywhere except that the card interior is closer to the MyGo base
(27,27,29 ≈ #1b1b1d, not 12,12,12).

## 3. Transparency hypothesis: refuted by a position test

The decisive test was to move the window and re-sample the **same client point**:

| window at | hero sample |
| --- | --- |
| (0, 0) | (43,54,50) |
| (320, 0) | (43,55,50) |

Identical within ±1 per channel ⇒ the window does **not** composite the desktop wallpaper. The colour is
painted by the app. (`Transparent: true` on the main window at `cmd/whaleisle/main.go:2345` is still
worth revisiting for rounded corners, but it is not the cause of the tint.)

## 4. Remaining suspects (for the product slice)

1. **The hero page's own ambient layer.** `chrome/background.go` paints an ambient wash when the
   background is "active"; the round230 trace pointed at the first-run preset, and `c23c96fc` cleared the
   first-run preset — but something still paints in the *conversation hero*: the hero sample has a
   green-teal cast (43,54,50 / 39,84,57) while the card interior is flat grey (27,27,29). A page-level
   gradient/bloom would produce exactly that split.
2. **The caption's blue is too strong** (37,100,129 vs native 19,43,61) — that may be the same layer
   showing through the caption, or an independent caption tint.
3. **The flat masked fill may be gated on something my profile does not satisfy** — the card interior at
   27,27,29 is the MyGo base (#151517 → 21,21,23 plus panel lightening) rather than the masked
   canvas value (12,12,12), so `FlatBackdropFill` is still not applied on this route even at alpha 255.
   Worth logging the gate conditions (resolved scheme, `Effect == ""`, wallpaper path, `alpha`) at the
   paint site for one run.

## 5. What is *not* claimed

- No baseline/actual pair, no structure rescore and **no masked ratio**: the pair would score the wash,
  not the chrome (the round230 masked number, 9.96 %, was taken with a cyan wash present on MyGo and is
  therefore still a wash-contaminated figure, not a chrome parity figure).
- Geometry is unaffected by this round: round232's live UIA already pinned slogan 314.67, preset 373.33
  and card 409.33 exactly.

## 6. State

pass297 stopped, temp profile removed, native restored to (200,2) 2268×1366 on the empty 新会话. No
commit, no passN, no contract edit. `fixture.md` updated with the round233 samples and the refuted
transparency hypothesis.
