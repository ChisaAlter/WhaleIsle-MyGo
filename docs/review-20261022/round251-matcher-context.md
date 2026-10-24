# Round251 — structural matcher context disambiguation

Date: 2026-10-25. Scripts/docs only: `scripts/acceptance-structure.py` changed, no product code, no commit,
no `passN`; the native app was not touched. Re-scored `evidence/accept-D-MAIN-247` and `…-245`.

## 1. The defect

`(type, name)` alone is not a key when a name is reused. In both rows:

| native `Button 新建会话` | rect (window-relative DIP) | MyGo counterpart |
| --- | --- | --- |
| rail/harness pill inside the caption band | x 16, y 16.67, **w 168.67**, h 38.67 | **none** |
| session header button under the band | x 14, y 70, **w 252**, h 38 | `Button 新会话` x 14, y 70.67, w 252, h 38 |
| — | — | `Button 新会话` x 12, **y 373.33**, w 256 (a session row titled 新会话) |

Name-only pairing scored the rail pill against the sidebar header button (nearest by `|Δy|+|Δw|`), that header
button against the session row, and produced the phantom **+303.33 DIP** pair reported in rounds 237/245 — while
the one real pair (header button ↔ header button, Δ 0.67) was never measured.

## 2. The rule (script header documents it)

Same-name nodes are bucketed by **context** before any ordering, with context always measured in the
**window-relative** frame (so the absolute space's −2800 DIP screen offset cannot move every native control into
one bucket):

```
region   sidebar (x_window < 280 DIP) or main
size     narrow (w < 200 DIP) or wide   # rail icon/pill vs full-width column row
id       automationId, ignored when empty or "RootWebArea"
```

- Buckets present on both sides pair **inside themselves**, nearest `(Δy, Δw)` first — ordinal order therefore
  survives only within one context, which keeps the old behaviour for duplicates that are genuinely equivalent.
- Leftovers stay on their side and are reported as `missing` / `extra` with `unpaired_after_context: true`.
  There is deliberately **no cross-context fallback**: after bucket pairing the remaining tokens on the two
  sides are necessarily different, so pairing them by order is exactly the phantom-pair bug.
- The alias table, the role-prefix aliases and the suffix-normalised name fallback are unchanged.

Schema: every existing key keeps its meaning. New fields only —
`context_rules` (top level), `pairing_compare` per space (`context_pairs`, `legacy_pairs`,
`added_by_context`, `removed_by_context`), and per match `pair_method` + `context`.

## 3. Before / after (same dumps, same tolerance)

| Row | Space | matched | within 1 DIP | missing | extra |
| --- | --- | --- | --- | --- | --- |
| **247** | absolute | 23 → 22 | 0 → 0 | 36 → 38 | 22 → 23 |
| **247** | window | 23 → 22 | **12 → 13** | 36 → 38 | 22 → 23 |
| **247** | caption | 23 → 22 | **8 → 9** | 36 → 38 | 22 → 23 |
| **245** | absolute | 23 → 22 | 0 → 0 | 36 → 38 | 22 → 23 |
| **245** | window | 23 → 22 | **11 → 12** | 36 → 38 | 22 → 23 |
| **245** | caption | 23 → 22 | **7 → 8** | 36 → 38 | 22 → 23 |

Pairs added by the rule (both rows, window space):

```
+ 新建会话  native {x:-2786, y:70,   w:252, h:38}  ->  mygo {x:14, y:70.67, w:252, h:38}   (Δ 0, 0.67, 0, 0)
```

Pairs removed by the rule (both rows):

```
- 新建会话  native {y:70,   w:252}   ->  mygo {y:373.33, w:256}   (the +303.33 phantom)
- 新建会话  native {y:16.67, w:168.67} -> mygo {y:70.67, w:252}    (rail pill vs header button)
```

Reported instead, honestly: `missing` gains the rail pill
(`新建会话`, context `sidebar/narrow`) and a surplus native `最大化` (its 28.67-DIP copy at y 868 has no MyGo
counterpart); `extra` gains the session row (`新会话`, context `sidebar/wide`).

`matched` falls by one because two wrong pairs became one right pair, one honest missing and one honest extra.

## 4. Verdict on the `新建会话` mismatch

It was a matcher artefact, not a UI delta: the real pair is within 1 DIP (Δy 0.67). What remains after the fix
is a **genuine** structural difference — MyGo has no control at the rail pill's position (native `新建会话` at
y 16.67, w 168.67 inside the caption band), and MyGo's sidebar carries a 60-DIP brand row before the header
button, whose own y is already aligned (70 vs 70.67). That belongs to the sidebar/caption family, not to the
matcher.

`标准模式` has no duplicate in these two rows (one node per side); the same rule covers it wherever several
preset rows share the name, since they differ by `region`/`size` and, when the dump carries it,
`automationId`.

## 5. Files

- `scripts/acceptance-structure.py` — context rule, `pair_with_context`, `pair_signature` diff,
  `pairing_compare`, `context_rules`, header section. `pair()` (nearest y/width) is now used only inside a
  context bucket.
- `docs/review-20261022/evidence/accept-D-MAIN-247/structure-relative.json` and `…-245/structure-relative.json`
  — the re-scored output (new schema fields included).
- `…/before-accept-D-MAIN-247.json`, `…/before-accept-D-MAIN-245.json` — the pre-change output kept for the
  comparison above.

## 6. Residual

1. `absolute` space still reports 0 within 1 DIP by construction (native screen offset −2800 DIP); only
   `window`/`caption` are meaningful. `within 1 DIP` should stay quoted in those two spaces.
2. One leftover native `最大化` (y 868.67, w 28.67) is unpaired on both rows — a MyGo control at that position
   does not exist under that name; worth a look from the terminal/caption family, not from the matcher.
3. Context buckets are geometric (`x < 280`, `w < 200`); a future row with a very wide rail control or a narrow
   main-column control would need the thresholds revisited. The dump's `automationId` is empty for these nodes,
   so the id leg of the rule is untested in practice.
4. Round245's own numbers were produced by the pre-change script, so its report keeps its original counts; the
   table above is the re-score of the same dumps.
