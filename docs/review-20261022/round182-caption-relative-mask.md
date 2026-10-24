# Round 182 — caption-relative structure and a wallpaper mask

Date: 2026-10-25. Branch `main` at HEAD `3ee88ae5` plus the uncommitted teammate work in the tree.
Read-only native app; no product code, no commit, no `passN`. The acceptance contract
(`docs/acceptance-ui-parity.md`) is **untouched**; only the two harness scripts and this document changed.

Question: rounds 170/175/177 reported `0/17 matched within 1 DIP` and a pixel mismatch of 99.99 %. Both
numbers are measurements of the *capture pipeline*, not of the two UIs. This round registers the coordinate
spaces, masks the background material, and re-scores the same matched dark pair
(`evidence/accept-D-MAIN`, native 0.3.10 vs MyGo `whaleisle-pass283.exe` @ `3e8fd30b`, 1440x860 DIP @ 144 dpi).

---

## 1. Method

### 1.1 Three coordinate spaces (`scripts/acceptance-structure.py`)

| Space | Transform | What it answers |
| --- | --- | --- |
| `absolute` | none — the recorded `dip` rect | nothing about layout: kept only for continuity with earlier rounds |
| `window` | rect − its own window/pane client origin | "is the control in the same place in the window" |
| `caption` | `window` − that side's caption bottom | "is the page laid out the same below the caption" |

Detection, all reported in `registration` of `structure-relative.json`:

- **Window origin**: the first `Window`/`Pane` node at depth ≤ 1.
  Native `Window "Whale Isle"` is at `x = -2800, y = 0` (screen coordinates); MyGo's is at `0, 0`
  (client coordinates). That single fact produced the `dx = +2800` on every control in earlier rounds — it is
  a capture-space bug, not a UI difference. `--native-origin X,Y` / `--mygo-origin X,Y` override it.
- **Caption bottom**: the bottom edge of the window-control triad (minimize/maximize/close) inside the top
  64 DIP. Native `最小化/最大化/关闭` sit at `y = 8, h = 32` → **40.0**; MyGo's at `y = 12.67, h = 32` →
  **44.67**. Difference (`caption_offset_dip`) = **+4.67**, absorbed by the caption-relative space and
  reported, never hidden. `--native-caption` / `--mygo-caption` override it. A document-level `关闭` at
  `y = 864` would otherwise define the origin; the 64 DIP band guard excludes it (this was a real defect of
  the first draft of the detector).
- **Alias table**: the four historical aliases plus 收起侧边栏→收起左侧栏, 切换右侧栏→切换右侧面板,
  展开面板→展开右侧面板, 添加工作区/选择工作区→选择工作区…, 搜索会话→搜索会话… . Seven of these fire on this
  row (`structure-relative.json → window.alias_matches`). A suffix-normalized fallback
  (ellipsis / trailing parenthetical) is also applied; it did not fire here (`normalized_matches: 0`).
- **Pairing**: same `(type, name)` key, paired in reading order by closest `(Δy, Δw)`, each node used once.
  Duplicate native names therefore pair twice; the two artifacts this produces here are called out in §3.2.

### 1.2 Wallpaper / mica mask (`scripts/acceptance-diff.py --mask-mode auto|both`)

The two clients do not paint the same background material: native shows the user's wallpaper through
translucent surfaces, MyGo paints its own gradient. Those differences are **broad and slowly varying**; a
misplaced control produces **thin, edge-rich** disagreement. The mask is therefore a low-frequency wash
detector, not a guessed geometry list:

```
D  = max channel |baseline − actual|
R  = 5x5 local range of D        (MaxFilter(5) − MinFilter(5))
M  = 5x5 local mean of D         (BoxBlur(2))
wash = D > threshold AND R <= 6 AND M >= 2
```

Parameters: `WASH_EDGE = 6` of 255, `WASH_MEAN = 2`. It is deliberately conservative — a 1 DIP text
displacement has `R` of tens of levels and is never masked. Registered rectangles from `mask.json` are still
honoured and can be combined with `--mask-mode both`. The resulting mask is written into the summary as
`wash_blocks` (32 px blocks the wash covers ≥ 90 % of) so a reviewer can register or challenge it without
re-deriving the image. Defaults are unchanged: `--mask-mode registered`, `--out diff.json`, so the contract's
existing flow keeps its behaviour; `--out diff-masked.json --mask-mode both` produces the new artifact.

Pixel geometry: window client is 1440x860 DIP, captured at 2160x1290 px (×1.5). Block coordinates below are
**capture pixels**; divide by 1.5 for DIP.

---

## 2. Numbers

### 2.1 Structure — the same 22 matched controls in three spaces

| Space | Matched | Within 1 DIP | Outside | Dominant delta (count) |
| --- | --- | --- | --- | --- |
| absolute (as recorded) | 22 | **0** | 22 | `dx +2800, dy +5` ×5 |
| window-relative | 22 | **4** | 18 | `dx 0, dy +5` ×5 |
| caption-relative | 22 | **5** | 17 | `dx 0, dy 0` ×5 |

Controls inside 1 DIP:

- window space: `插件`, `收起侧边栏`, `自动化任务`, `鲸鱼娘` — all `|Δ| ≤ 0.67` (sub-pixel UIA rounding).
- caption space: `最小化`, `最大化`, `关闭`, `切换右侧栏`, `切换终端抽屉` — all `Δ = 0`; the caption cluster is
  exact once the caption is removed from the origin, and the sidebar rows pick up the 4.67 the registration
  absorbed (`插件` etc. become −4.0).

Controls: 64 named native nodes, 44 named MyGo nodes; 37 native-only, 27 MyGo-only (see §3.3).

The two clean clusters explain the whole residual, i.e. after fixing the origin there is **no 36 DIP band
shift**: the caption triad sits +4.67 lower in MyGo (8 vs 12.67 DIP button top) and the sidebar header row
sits +6.67 lower (250 vs 256.67). Both are 4–7 DIP, matching the chrome findings in
`docs/parity/chrome-metrics-20261024.md` (native caption button top 12 vs MyGo 8; band equal at 48).

### 2.2 Pixel — masked vs unmasked

| Metric | Value |
| --- | --- |
| Pixels compared | 2 786 400 (2160x1290) |
| Mismatch at threshold 0, `registered` mask (mask.json has **0** regions) | 2 786 143 = **99.9908 %** |
| Wash mask coverage | 2 545 451 px = **91.3527 %** |
| Mismatch at threshold 0, wash mask applied | 240 692 px = **8.6381 %** |
| Worst channel delta | 243 (of 255) |
| Size crop | actual was 2160x1291, baseline 2160x1290 → common 2160x1290 |

Per region, after masking (`remaining_regions` in `diff-masked.json`):

| Region (DIP) | Mismatch after mask |
| --- | --- |
| sidebar `0,0–285,860` | 18.99 % |
| header `285,0–1440,60` | 18.02 % |
| footer `0,800–285,860` | 30.44 % |
| composer `285,400–1440,620` | 9.57 % |
| hero `285,60–1440,400` | 4.69 % |

Largest surviving blocks (capture px): `(32,32)` 1005, `(1792,32)` 978, `(96,32)` 917, `(1152,512)` 901,
`(1344,512)` 874, `(1184,512)` 872, `(704,640)` 872, `(1824,32)` 871, `(1760,32)` 869, `(32,384)` 867.
In DIP: the sidebar brand/lock at (21,21), the caption/trailing cluster around (1170–1220, 21), the hero
preset row at (768–896, 341), the composer text at (469, 427), and the sidebar tree at (21, 256).
Those are exactly the places where the two clients either differ structurally (caption triad, sidebar header,
composer row) or show **different content**.

---

## 3. What is genuinely mismatched

### 3.1 Real, clustered (each is one fix, not N)

| Cluster | Native | MyGo | Δ | Evidence |
| --- | --- | --- | --- | --- |
| caption window buttons | top 8, 32 tall → bottom 40 | top 12.67, 32 tall → bottom 44.67 | +4.67 | 5 controls identically offset (`切换终端抽屉`, `切换右侧栏`, `最小化`, `最大化`, `关闭`); matches chrome-metrics F1 (Top 8 → 12) |
| sidebar heading row (工作区/多选会话/视图选项/添加工作区/搜索会话) | y 250 | y 256.67 | +6.67 (工作区 +7.33, also `dw +82`) | 5 controls at the same y in window space; +2.0 in caption space after the 4.67 is absorbed |

### 3.2 Real, individual

| Control | Native (DIP) | MyGo (DIP) | Δ |
| --- | --- | --- | --- |
| 发送消息 (send disc) | y 522, 34.67² | y 493.33, 34² | `dy −28.67` |
| 探索未至之鲸 (hero title) | y 314.67, h 35.33 | y 339.33, h 32 | `dy +24.66`, `dh −3.33` |
| 大肥鱼版 (preset label) | y 321.33, w 48.67 | y 345.33, w 62.67 | `dy +24`, `dw +14` |
| 新建会话 (sidebar first row) | y 16.67, w 168.67 | y 70.67, w 252 | `dy +54`, `dw +83.33` — MyGo paints a 60 DIP brand row first (its own texts at y 20–42) and a full-width pill |
| 工作区 (heading) | w 42 | w 124 | `dw +82` — container vs text node |
| 账号菜单 | h 44 | h 36 | `dh −8`, `dy +8.67` |

### 3.3 Not layout, and why the matched set is small

- **37 native-only**: 30 are data content — 15 session `TreeItem`s (`whale-code`, `WhaleBridge`, …), the
  `会话` tree, the terminal cards below the fold — plus composer controls whose accessible names are long
  sentences (`描述你想要构建的内容, / 调用指令, @ 文件或对话`, `选择模型，当前 DeepSeek V4.1 Flash 1017，推理等级 Max`).
  The remaining 7 are selector drift in the trailing cluster (`切换分支`, `Commit & push`, `Git 操作` vs MyGo's
  `初始化 Git`) and `展开面板`.
- **27 MyGo-only**: instrumentation labels MyGo sets deliberately (`Composer card`, `Composer draft`,
  `Composer trailing controls`, `Sidebar divider`, the brand texts) plus its own workspace tree rows.
- **2 pairing artifacts**: native exposes two `新建会话`-named nodes and the greedy pairing produced a
  `dy +303.33` record; native also has several `标准模式` preset nodes and one paired across columns
  (`dx +472`). They are recorded, not silently dropped, but they are artifacts of name-keyed pairing.
- **Content mismatch**: the two clients do not show the same sessions, so the sidebar tree and the hero copy
  can never agree pixel- or name-wise on this pair. This is the dominant reason the pixel ratio cannot reach
  0 and the reason the matched-control denominator is 22.

---

## 4. Revised verdict on the proposed criterion

The proposal was **masked 0 / 1-DIP structure / per-region ≤0.1 %**. Measured on the matched dark pair:

| Criterion | Result | Verdict |
| --- | --- | --- |
| masked mismatch 0 | 8.6381 % after a 91.35 % wash mask | **not met, and not attainable on this pair** |
| structure within 1 DIP | 4/22 window-relative, 5/22 caption-relative (was 0/17) | **partly met**, and now measurable |
| per-region ≤ 0.1 % | composer 9.57 %, hero 4.69 %, sidebar 18.99 % | **not met, and not attainable with data-dependent content** |

Three conclusions, in order of confidence:

1. **The earlier 0/17 was a capture-space defect.** Native UIA reports screen coordinates and MyGo reports
   client coordinates; `dx = +2800` made every control fail regardless of layout. Comparing in window space
   turns that into 4/22, and the residual is two small clusters (+4.67 caption, +6.67 sidebar header) plus
   the known S27 composer/hero offsets — not a 36 DIP band shift. This matches
   `docs/parity/chrome-metrics-20261024.md`: the band is 48 DIP in both clients; only the caption contents
   sit ~4 DIP differently.
2. **The pixel criterion is currently insensitive, not passed.** A 91.35 % mask means the test is mostly
   measuring the background material, and the surviving 8.64 % is concentrated on content that legitimately
   differs. "Masked 0" cannot be reached until (a) both clients are captured with the same fixture/session
   set and (b) the S27 composer/hero and the two clusters are fixed.
3. **A workable replacement criterion**, keeping the contract untouched:

   - **Structure**: 1-DIP tolerance in **window-relative** space, over a *curated* control list (the 20-odd
     controls both clients expose with stable names), reported together with caption-relative counts and the
     `caption_offset_dip`. Gate: 100 % of the curated list within 1 DIP after the caption/sidebar-header
     fixes; alias drift counts as matched, missing/extra names are reported but not gated.
   - **Pixels**: keep threshold 0 but publish **masked ratio with the wash coverage** and gate per region only
     where content is fixture-controlled (empty-state hero, composer chrome, caption band): target
     ≤ 0.1 % there. For content regions (sidebar tree, transcript) publish the ratio as a **regression
     signal** (must not increase between rounds), not as a pass gate.
   - **Fixture**: capture both clients from the same seeded workspace (one archived session, `fixture.md`
     loaded) so the sidebar tree and hero copy agree; without this, no pixel threshold is meaningful.

---

## 5. Files

- `scripts/acceptance-structure.py` — three spaces, origin/caption detection with overrides, duplicate-aware
  pairing, alias + suffix-normalized matching, `dominant_delta`, per-space counts. Default output stays
  `structure.json`.
- `scripts/acceptance-diff.py` — `--mask-mode registered|auto|both`, `--out`, wash method documented at the
  top of the file, `wash_blocks`, `remaining_blocks`, `remaining_regions`, masked/unmasked ratios. Default
  behaviour unchanged.
- `docs/review-20261022/evidence/accept-D-MAIN/structure-relative.json` — structure in all three spaces.
- `docs/review-20261022/evidence/accept-D-MAIN/diff-masked.json` — masked pixel summary (8.6381 % after mask,
  91.3527 % masked).
- `docs/review-20261022/evidence/accept-D-MAIN/diff.json` — contract-default (registered mask, unmasked
  99.9908 %) for continuity; `diff.png` shows the masked run.

## 6. Unverified / limits

- The wash thresholds (6 / 2) are a calibration choice made on one pair; a second pair (light theme, or the
  D-EMPTY row) has not been re-scored, so the 91 % coverage is a property of *this* wallpaper, not a constant.
- The caption bottom is inferred from the control triad, not from a caption node (neither dump exposes one);
  if a future build moves the triad without moving the band, the registration will be off by that amount and
  `caption_offset_dip` will move with it — read it as a signal, not as truth.
- No native recapture was done (read-only, no `passN`): the pair is round-175's, so the native side reflects
  0.3.10 + that session's content.
- Pairing is name-keyed; two artifacts remain (§3.3). A curated control manifest would remove them, but that
  changes the harness contract and was out of scope for this slice.
