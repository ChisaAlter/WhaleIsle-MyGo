# Round 250 — AA-dilated acceptance metric

New tooling: **`scripts/score-aa.py`** scores one acceptance row three ways and answers whether the
hero/composer ≤0.1 % per-region target is reachable. Short answer: **yes, and it is met on the light pair
once glyph antialiasing is excluded** — hero 0.000 %, composer 0.035 %. The contract itself is untouched.

## 1. Method

For a row's `baseline.png` / `actual.png`:

1. **Ink mask** — per channel, the 3×3 local range (max − min) in *either* image, max over channels;
   pixels above `--ink` (default 10 levels) are "text ink / glyph edge".
2. **Dilation** — `--dilate` (default 2) 3×3 max passes ≈ 2 px, covering a 1-px glyph shift plus AA.
3. **Modes**
   - `raw`: every pixel (the contract's literal form).
   - `caption_masked`: the registered `mask.json` rectangles removed (the 40.67-DIP caption band, which
     only carries injected Chrome controls).
   - `aa_excluded`: caption band **and** dilated ink removed.
4. Reports per mode: kept pixels, mean │Δ│, mismatch at Δ>0 / Δ>8 / Δ>16, and each ratio over **two
   denominators** — the kept set and the whole frame. The second one exists because a mode that removes
   only *matching* pixels would otherwise look better than it is (this actually happens to the caption
   band: masking it raises the header's kept-set ratio from 6.06 % to 7.35 % while lowering the count).

Limitations, stated for the record: the ink test is heuristic (local contrast, not OCR); it excludes any
high-contrast graphic, not only text; it cannot see a *size* change that stays inside the ink band (the
hero fish is exactly that case — caught by structure, invisible here); and the dilation radius trades
coverage against generosity, so coverage is reported and capped in the proposed criterion.

## 2. Numbers — `evidence/accept-D-MAIN-245` (last valid light pair)

Ink coverage after dilation: **6.11 %** of the frame; caption band **4.73 %**. Kept: raw 2,786,400 ·
caption-masked 2,654,640 · AA-excluded 2,508,721.

| mode | whole frame Δ>0 | Δ>16 | mean │Δ│ |
| --- | --- | --- | --- |
| raw | 1.685 % | 1.457 % | 1.234 |
| caption-masked (kept-set / frame) | 1.431 % / 1.363 % | 1.242 % / 1.183 % | 1.005 |
| **AA-excluded (kept-set / frame)** | **0.008 % / 0.007 %** | **0 %** | **0.000** |

Per region (kept-set Δ>0 / frame Δ>0):

| region | raw | caption-masked | **AA-excluded** |
| --- | --- | --- | --- |
| sidebar | 3.15 % | 2.69 % | **0.000 %** |
| header (caption) | 6.06 % | 7.35 % | **0.000 %** |
| **hero** | 1.07 % | 1.07 % | **0.000 %** |
| **composer** | 1.26 % | 1.26 % | **0.037 %** (frame 0.035 %) |
| footer | 5.63 % | 5.63 % | **0.000 %** |

Every mismatch left outside the ink mask is in the composer, and there are none above Δ>16 anywhere.

## 3. Is hero/composer ≤0.1 % reachable?

**Yes, on the evidence**: hero 0.000 %, composer 0.037 % under AA exclusion, with zero pixels above Δ>16
in either region. What remains is exactly the known draft-inset displacement (composer Edit width
+28.67 DIP, task-120) — a real, non-text difference; once that lands the composer should also read 0.000 %.
Two caveats belong in any claim:

1. The fish (34×24.67 vs native 48.67×15.33) is a **size** difference that lives inside the ink band, so
   the AA metric cannot see it — it must stay a *structural* requirement (task-120 step 4).
2. The pass301 row (`accept-D-MAIN-247`) could **not** be scored: its native baseline is unusable (the
   installed app was painting a light caption over a dark body). Its geometry was verified exactly and is
   recorded in round247; only the pixel score is missing.

## 4. Proposed criterion wording (not applied to the contract)

> **Pixel parity (per region).** With the row's registered mask regions removed and with all pixels
> within 2 px of a high-contrast ink edge excluded (3×3 local contrast > 10 levels in either image), the
> mismatch ratio must be ≤ 0.1 % of the region's kept pixels and no kept pixel may differ by more than
> 16 levels. The ink mask must cover ≤ 10 % of the region; otherwise the row is void and must be
> recaptured with cleaner content. Every row reports three numbers — raw, registered-mask, AA-excluded —
> plus the ink coverage, so a reviewer can see what the exclusion removed.

Rationale for each clause: the AA exclusion is what makes two independent text renderers comparable;
the Δ>16 clause keeps "strong disagreements" impossible to hide; the coverage cap stops a generous mask
from being used to pass a bad row; and the three-number report prevents the masked figure from being
quoted without context. Structural parity (1-DIP UIA matching) remains the only guard against size
changes inside the ink band.

## 5. Residual

1. Re-run the light pair after task-120 (draft insets, fish) → expect composer 0.000 % and the fish fixed
   structurally; publish that as the first pair that satisfies both legs.
2. Re-score pass301 when the native window is uniformly themed (the 247 capture is not usable).
3. Optional: `score-aa.py` could emit a `--plot-mask` PNG for reviewer eyeballing; not needed for the
   criterion.

## 6. State

Scripts/docs only; no product code, no commit, no passN, no contract edit. No temp profiles were created
this round; native untouched.

## 7. Sensitivity check (dilation radius)

Re-running with --dilate 1 (≈1 px, ink coverage 4.89 % instead of 6.11 %) leaves the conclusion
unchanged: absolute AA-excluded Δ>0 = 0.008 % of kept, **hero 0.000 %**, **composer 0.036 %** (frame
0.035 %), Δ>16 = 0. So the verdict does not depend on the exact dilation radius, which is what makes the
proposed criterion safe to adopt at either 1 or 2 px.
