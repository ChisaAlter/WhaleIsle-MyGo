# Round 186 — composer column axis and 标准模式 chip

Two S27 items. The chip is fixed in code; the composer's 1.06 DIP turned out **not** to be a formula
bug at all — it is a persisted transcript-width preference in the throwaway profile. Native probe at
1440×860 (staged harness, CDP). No contract edit, no commit, no passN.

## 1. Native column probe (1440×860, DIP)

| element | rect | computed |
| --- | --- | --- |
| conversation root | (280, 0) **1160 × 860** | sidebar 280, no right dock |
| body | (280, 52) 1160 × 808 | |
| scrollBody | (280, 52) **1158 × 808** | 1160 − 2 scroll margin |
| composer seat | (280, 336) **1153 × 240** | 1158 − 5 scrollbar gutter |
| composer stack | (453.3, 336) **806.39** × 240 | `padding: 0 0 32px` |
| hero root | (453.3, 430) 806.39 × 114 | `padding: 0 16px` |
| **composer card** | (469.3, 430) **774.39 × 114** | `max-width: 774.4px`, `padding: 8px 0 0` |

Derived formulas (all exact):

- `axis = 0.64 × 1160 = 742.4` → stack `= axis + 64 = 806.4` → card `= stack − 2×16 = 774.4`.
  So the native axis is the plain 0.64 × column, and the card's 16-DIP side clearance comes from the
  hero root's padding, not from a separate composer inset.

## 2. Why MyGo measured 773.33 — it is the profile, not the formula

MyGo's chain is the same: `contentAxis` → stack `= axis+64` → `composer-region` padding `16/16` → card.
With `contentAxis = 742.4` the card is 774.4 exactly. The capture measured 773.33 because `contentAxis`
was **741.33**, and the reason is the *persisted* transcript width:

```go
// internal/ui/frame/conversation_width.go:23
preference := s.TranscriptWidth
return min(max(640, column-176), max(640, preference))   // when preference > 0
```

`TranscriptWidth` is restored from the session-recovery layout (`cmd/whaleisle/recovery.go:54`), and the
throwaway profile was seeded from the real user config, so it inherited a local preference of ≈741.33
instead of falling back to `conversationContentWidth(column)` (0.64 × column = 742.4).

**Consequence: no product change is needed for the composer width.** The acceptance fixture must reset
the persisted layout for the throwaway profile (transcript width 0 → 0.64 formula), otherwise the row
scores a user preference as a renderer delta. That is a `fixture.md` correction, and it also explains
why the round181 cap/floor experiments broke the hero↔docked identity: they were papering over a
preference, not a formula.

## 3. 标准模式 chip — fixed in code

Native keeps the preset chip in the left flow right after the workspace chip (probe: `标准模式` at
**x = 638.3**, workspace chip ends near 620). MyGo pushed it right with a grow spacer
(`ui.Box(c).Grow(1).MinWidth(0)`), which put it at **x ≈ 1127**.

- `internal/ui/composer/composer.go`: the spacer is gone; the preset chip now follows the workspace
  chip in the row's flow (row gap 2 DIP + chip padding), with the native behaviour cited in the
  comment.
- Test updated: `TestHeroRowCarriesWorkspaceAndPreset` used to require the chip flush to the row's right
  edge (inset 0); it now requires the chip to sit 0–24 DIP after the workspace chip and to stay inside
  the row. All four width/root combinations pass.

## 4. Verification

- `go test ./internal/ui/composer ./internal/ui/chat ./internal/ui/frame -count=1` → **ok**
  (hero/docked identity tests, hero row test, S28 nav-label test all pass).
- `go vet` on the same three packages → clean.
- **S28 re-run**: the nav-label assertion still requires the 22-DIP line box at y≈127 — unchanged and
  passing.
- Round181's hero offsets are untouched (headline foot 25 DIP stays).
- No new capture this round: with the profile preference reset, the expected values are card **774.39**
  (was 773.33, +1.06), hero title y and preset y unchanged from round181's capture, chip x moving from
  ≈1127 to ≈492+chip width — the next acceptance capture should confirm all four at once.

## 5. Residual / next

| item | expectation | action |
| --- | --- | --- |
| composer card width | 773.33 → **774.39** | reset `TranscriptWidth` in the throwaway profile (fixture), then capture |
| 标准模式 chip x | ≈1127 → left flow (≈640 area) | confirm in the same capture |
| hero title / preset y | unchanged (round181) | same capture |
| docked card width at very wide windows | unchanged (MyGo keeps the transcript-axis identity; the native card caps at 774.4) | open question for the contract: is the native docked card also capped? needs a probe with the right dock open |

## 6. State

Harness web server stopped; no temp profile created (the probe needed none for MyGo); native app
untouched.
