# Round255 — fish badge and preset chip audit

Date: 2026-10-25. Read-only audit of the two pass302 discrepancies. No product code changed: the only additions
are two diagnostics (`internal/ui/frame/round255_hero_nodes_test.go`,
`internal/ui/composer/round255_preset_chip_test.go`). No commit, no `passN`; native app untouched; temp profiles
and the UIA helper script removed.

## 1. Fish badge — the two numbers are two different nodes, not two versions of one

The task read "UIA fish 34×24.67 at x728.67 vs harness badge 49×15.4 at x936" as a possible stale/duplicate
node. It is not: they are the **fish and the badge of the same headline row**.

Diagnostic (`go test ./internal/ui/frame -run TestRound255HeroRowNodes -v`, 1440×860 headless):

| node | x | w × h (DIP) | source | live UIA / harness |
| --- | --- | --- | --- | --- |
| `Conversation hero fish` | 729.00 | 34.00 × 25.02 | `internal/ui/frame/conversation_fish.go:10` (`Key` + `Label`, `Width(34)`, `Height(34*17.04/23.16)`) | UIA 34 × 24.67 @ 728.67 |
| `探索未至之境` (slogan) | 765.69 | 156.00 × 32.00 | `conversation_phase.go:110` | UIA 765.33 × 156.67 |
| `大肥鱼版` (badge) | **936.00** | **49.00 × 15.40** | `conversation_phase.go:122-124` (`Text`, font 8.5 / line 13.4, padding 1/7, radius 999) | harness **49 × 15.4 @ 936** |

Row arithmetic from the same run: fish end 763.00 + **2.69** = slogan start 765.69; slogan end 921.69 + **14.31**
= badge 936.00 — the round254 distribution, and the row centre does not move.

So, for the three possibilities the task listed:

- **Is the old node still emitted under that name?** No. One `conversationHeroFish` call exists per state
  (`conversation_phase.go:108` for the live hero, `:141` for the settling skeleton); they cannot render together,
  and the fish keeps its `Key`/`Label`, which is why UIA can report it at all.
- **Is the new badge unnamed?** No. The badge is a `ui.Text` whose name is its content (`大肥鱼版`); the frame
  test finds it by that name, so UIA exposes it too.
- **Did the narrowing not reach pass302?** It did: 49.00 × 15.40 in the frame equals the harness 49 × 15.4, and
  the 8.5/13.4 font is the round242/249 value.

The only real deltas are measurement rounding: fish x 729.00 vs UIA 728.67 (0.33), fish height 25.02 vs UIA
24.67 (0.35, UIA truncation), badge height 15.40 vs harness 15.4. Nothing to name, nothing to hand to the
layout owner for the fish — the "discrepancy" was a comparison across two nodes.

## 2. Preset chip — composition measured, delta explained, no single wrong value

Diagnostic (`go test ./internal/ui/composer -run TestRound255PresetChipComposition -v`):

```
workspace chip  x=48.67  w=150.00 h=28.00
preset chip     x=202.00 w=101.47 h=28.00      (native: x 655.33, w 106.67)
preset label    x=230.00 w= 52.00 h=17.34      ("标准模式" = 4 x 13 DIP)
chip - label:   left=28.00  right=21.47        (padding + gap + leading icon / chevron)
chip-to-chip gap=3.33
```

Composition rule (from the code, `internal/ui/composer/preset_picker.go:63-85` and the shared
`chip()` helper at `composer.go:68`):

```
chip width = 8 (left padding) + 4 (gap) + [leading block] + text + 4 (gap) + chevron + 4 (right padding)
           = 28.00 (left block) + 52.00 (text) + 21.47 (chevron + right padding) = 101.47
```

`left=28.00` is the part worth knowing: the chip's own padding+gap is 12, so a **16 DIP leading block**
(`paintPresetIntroduction` / `presetOptions.Leading`) sits before the label. The like-for-like live delta
(MyGo 100 vs native 106.67 = **6.67**) is therefore spread over the leading block, the label metrics and the
chevron — not one obviously wrong constant:

- leading block 16 vs ~19-20 native (≈+3-4),
- label 52 (13 DIP per CJK glyph) vs native's 14 DIP chip font (≈+4 if native is 14),
- chevron ~13.5 vs native's larger caret (≈0-2).

**No single clearly-wrong value**, so nothing was implemented. Handed to the layout owner with the exact path:

> `internal/ui/composer/composer.go:268-335` — hero preset row: `Gap(3.33)`, `Margin(4,0,8,0)`,
> `Padding(0,16,0,32.67)`; the preset control is `presetSelect(...)` (`preset_picker.go:58-93`) whose `trigger`
> comes from `chip()` (`composer.go:68`: `Height(28)`, `Padding(0,4,0,8)`, `Gap(4)`, `MaxWidth(220)`) and whose
> `Leading`/chevron sizes decide the 100 vs 106.67 gap. Round211 pinned the workspace chip at 150, so the preset
> chip's own composition is the only remaining lever.

The `+3.34` x offset follows from the same width shortfall only if the row's left anchor is fixed; the row-level
numbers (workspace x 48.67 row-relative, gap 3.33) are already inside the round217/237 tolerances, so the x
difference should be re-measured after any width change rather than tuned independently.

## 3. What was changed

Nothing in product code. Two diagnostics added (both pass, `go vet ./internal/ui/frame ./internal/ui/composer`
clean):

- `internal/ui/frame/round255_hero_nodes_test.go` — prints the three hero-row nodes and pins x/w/h within 1 DIP
  plus the 2.69 / 14.31 gaps; it is the executable form of the table above.
- `internal/ui/composer/round255_preset_chip_test.go` — prints the workspace chip, preset chip, preset label and
  the left/right decomposition, which is what the layout owner needs to close the 6.67.

## 4. Residual

1. **No live UIA this round.** The installed native app is not running, and pass302 with a minimal throwaway
   profile exits before its main window (the round195/profile-completeness fault); the numbers above are the
   frame's headless rects plus the UIA/harness values the task supplied, not a fresh dump. A dump would only
   confirm the same three rects (one is already exact: badge 49 × 15.4 @ 936).
2. The preset chip's live width (100) is 1.47 narrower than the headless 101.47; that difference is label-metric
   (live font rasterisation vs headless estimator), and it should be measured again on the same build before any
   padding change.
3. Fish and badge are 0.33-0.35 off the UIA numbers purely from truncation; do not "fix" those.
