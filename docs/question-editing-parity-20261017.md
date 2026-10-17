# Question editing parity: CH55 and CH58

Implemented 2026-10-17 in MyGo only. Original reference is read-only WhaleIsle
`a8c931da37376ce7dc7989e6f30b0db885e06d26`,
`vendor/deepseek-harness/packages/client/ui-user-questions/src/client/QuestionComposer.tsx`.

## Behavior

- Custom answers, both optionless and alongside choices, use a native TextArea.
  Shift+Enter preserves a newline. Final answer serialization preserves internal
  newlines and trims surrounding whitespace as before.
- Enter requires an answer on the current question. It moves focus to the next
  question; on the last question it either points to the first incomplete answer
  or submits via the existing live/continued route. Skipped earlier questions
  count as complete. Non-Shift modified Enter also continues, matching the
  original. Held-key repeats do not advance repeatedly.
- Input-method composition and its committing Enter are protected. Native
  TextArea composition is observed during the build, because its text adapter
  does not forward composition events to HandleInput. The callback captures
  values rather than retaining expired UI Element handles.
- Sending disables the entire answer subtree: free text, checkbox choices,
  select controls and Skip/Undo. Store setters also reject edits while Sending.
  The UI binds frame-local copies committed through those setters, not mutable
  pointers into a retained draft, so a stale pre-Sending frame cannot bypass the
  guard. The existing frozen response and retry route remain intact.
- Existing CH50 edit/focus/take-time behavior is retained through the same store
  setters and answer-body focus reporting.

Original references: QuestionComposer.tsx:196-203, 340-355, 390-420,
520-525, 583-597, 611-635. These cover busy locking, current-answer validation,
next/final submission, Shift+Enter and IME protection.

## Scope boundary

The current native all-question layout is retained. Enter advances focus within
that layout; this does not claim CH56's paginated wizard, progress counter,
previous/next buttons, option-click auto-advance or collapse/expand parity.
The original option-button Enter behavior is separate from custom-field Enter
and was not changed here. Live Host/desktop acceptance remains unverified.

## Shared CH69 integration seam

`internal/ui/interaction/card.go` adds `Model.ScopeAgentID` and
`Model.DisablePageShortcuts`. The separately owned child-pane module provides
`SyncQuestionFocusForAgent` and frame focus aggregation. The related two store
fields in `internal/store/interaction/store.go` are also CH69-owned. These files
must be published atomically with the child-pane change, so multiple surfaces
cannot clear each other's countdown focus or compete for page-level approval
shortcuts.

## Focused verification

Both missing behaviors were reproduced before implementation:

- `TestQuestionSendingFreezesDraftMutations` showed custom, single-select and
  multi-select values still mutating after answer or cancellation started.
- `TestQuestionCustomShiftEnterPreservesNewline` returned `onetwo`, not
  `one\ntwo`, through the actual native widget.

Final command, using the existing offline toolchain/cache:

```text
GOTOOLCHAIN=local GOPROXY=off \
GOMODCACHE=/workspace/scratch/2d36477d56ec/.gomodcache \
GOCACHE=/workspace/scratch/2d36477d56ec/.toolchain/gocache \
/workspace/scratch/2d36477d56ec/.toolchain/go/bin/go test \
  ./internal/store/interaction ./internal/ui/interaction -count=1

ok whaleisle/internal/store/interaction 0.112s
ok whaleisle/internal/ui/interaction 0.385s
```

The new tests cover multiline text, current-answer validation, next-field focus,
final serialization, missing earlier answers, foreign-session rejection,
one-shot submission, native IME and commit-Enter protection, repeated keys,
stale-frame pointer isolation, and disabled choices/skip. Existing interaction
and CH50 tests also passed in this run. No original code or tests were executed.
