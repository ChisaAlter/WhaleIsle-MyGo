# Round 181 — S27 finished with a native 1440×860 DOM probe

Probe: the staged harness web UI (`dsh web`, token URL, CDP via `scripts/native-web-shot.mjs --eval`)
at **1440×860** and the round152 numbers at **1440×920**. No product code outside the two frame files,
no contract edit, no commit, no passN.

## 1. Probe results (DOM, devicePixelRatio 1, DIP)

| element | 1440×860 | 1440×920 | notes |
| --- | --- | --- | --- |
| composer card (`gVwOaG_card`) | **774.39 × 114** at (469.3, **430**) | 774.39 × 114 at (469.3, 460) | `padding: 8px 0 0`, `gap 12`, `radius 28`, **`max-width: 774.4px`** |
| beam layer / inner | 782.39 × 122 at (465.3, 426) | 782.39 × 122 | card + 4 DIP halo each side |
| headline row (badge parent) | (764.98, **337**) 227 × 32 | (765, 367) 227 × 32 | 探索未至之境 26/32 + 大肥鱼版 badge 12/18 |
| 大肥鱼版 badge | (927.98, 339) 64 × 21 | (928, 369) 64 × 21 | |
| 标准模式 chip | (638.3, **398**) 52 × 20 | (638.3, 428) 52 × 20 | left-aligned after the workspace chip, **not** right-aligned |
| CSS vars | `--dsh-composer-card-max-width` etc. all empty | same | the ceiling is a literal `max-width`, not a token |

**Diagnosis correction:** the round177 claim "native card ≈155 DIP at 860" was wrong — it came from a
UIA dump taken while the native window was parked off-screen at x=−4200, which mixed coordinate frames.
The live DOM says the card is **114 DIP at both 860 and 920**, so MyGo's 114-DIP recipe
(`8 + 52 + 12 + 34 + 8`) is correct and no responsive card height is needed. The ±1.06 DIP width and
the hero seat offsets are the only real deltas.

## 2. What was changed

1. **Hero headline foot 12 → 25 DIP** (`internal/ui/frame/conversation_phase.go`). The installed hero
   has the headline row 337..369 and the preset row at 398 → 29 DIP between them; MyGo's 12-DIP foot
   produced a 16-DIP gap, so the preset row and card sat 13 DIP high.
2. **Recorded the card ceiling** as `composerCardMaxWidth = 774.39` (`conversation_phase.go`) with the
   probe citation, and corrected the seat comment in `frame.go` so the next reader knows the installed
   card is `max-width: 774.4px` and where the 1.06 DIP lives.

## 3. Composer width: tried, measured, reverted — the gap is in the column

Three variants were built and tested:

| attempt | result |
| --- | --- |
| `composerOuterW = min(min(contentAxis+64, 774.39), viewport)` | hero card 774.39 ✓ but the docked composer was clamped too → `TestPopulatedComposerCardKeepsSharedAxisAndFitsCenter` failed (742.39 vs 913.92 at a 2200-wide window) |
| cap only when `emptyHero` | hero 774.39 ✓, docked unchanged, but broke the hero↔docked width identity: `TestConversationHeroFirstMessageKeepsCardAndDraftIdentity` (742.39 vs 952) and `TestConversationFirstSubmissionPinsFirstFrame…` (742.39 vs 712) |
| floor when `emptyHero && composerOuterW < 774.39` | same identity failures at narrow columns (900) |

So the hero card must keep the shared seat width, and the 1.06 DIP must be fixed **upstream**:
`contentAxis = min(920, max(680, colW*0.64))` gives 709.33 at 1440 where the installed card implies
710.39, i.e. MyGo's column is ≈1.65 DIP narrower than the native page's at the same window (sidebar
280 + gutters). That is a column/inset measurement, not a composer formula, and it is the next slice.
The three experiments and their failures are recorded here so nobody repeats them.

## 4. Verification

- `go test ./internal/ui/frame ./internal/ui/composer ./internal/ui/chat -count=1` → **ok**
  (including the hero/docked identity tests and `TestSidebarChromeMatchesTheNativePage`).
- `go vet` on the same three packages → clean.
- **S28 confirmed intact**: the nav label assertion still requires a **22-DIP** line box at
  **y≈127** (the same test that round177 added), and the suite passes.
- The headline-foot change was verified structurally by the widget suite, not by a fresh dark capture:
  the headless fixture does not carry the workspace/model state the real hero needs (a probe at
  1440×860 with an empty `ViewState` renders a 641×50 card), so a pixel/rect confirmation of the
  +13 DIP shift needs the full acceptance capture — folded into the next round together with the
  column-width fix.

## 5. Residual

| item | value | next step |
| --- | --- | --- |
| composer width | 773.33 vs 774.39 (**1.06 DIP**) | measure the native column width at 1440 (sidebar + gutter + right inset) and align `conversationWidth` |
| hero seat, preset row | was +13 (now expected ≈0) | confirm with an acceptance capture |
| hero seat, card top | was +6.7 (expected ≈+6.7−13 → ≈−6) | same capture; then adjust the preset→card spacing |
| 标准模式 chip position | native left-aligned after the workspace chip (x=638.3); MyGo pushes it right (x≈1088) | separate slice: remove the auto-margin spacer in the hero preset row |
| card height | none — 114 is correct at both heights | — |

## 6. State

Harness web server stopped; no temp profile created this round; native app untouched (still the empty
`新会话` from the acceptance row).
