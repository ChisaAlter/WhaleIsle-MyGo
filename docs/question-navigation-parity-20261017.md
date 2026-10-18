# CH56: paged question navigation

Implemented 2026-10-17 in MyGo. This supersedes the all-question layout boundary
recorded in the earlier CH55/CH58 note. Original WhaleIsle was read only at
`a8c931da37376ce7dc7989e6f30b0db885e06d26`.

## Original semantics and native mapping

Original references below are within
`vendor/deepseek-harness/packages/client/ui-user-questions/src/client`.

- `QuestionComposer.tsx:607-623`: previous/next arrows navigate without requiring
  an answer. Native `question_navigation.go:17-27` validates exact agent/event
  and bounds; `QuestionCardView.Index` stores the visible page with the draft.
- `QuestionComposer.tsx:328-337`: a single choice advances, a multi-selection
  stays on the current page, and a final choice does not submit. Native
  `question_navigation.go:32-43` and the per-option callback in
  `internal/ui/interaction/question_editing.go:75-94` implement that split.
  All options remain individual controls, including questions with more than
  four choices; no dropdown hides the original option Enter behavior.
- `QuestionComposer.tsx:340-365,390-401`: primary Next/Enter requires a current
  answer; final Submit validates every answer or skip and redirects to the first
  incomplete page. Native `internal/store/interaction/question_editing.go`
  and `question_navigation.go:45-79` preserve the existing answer route.
- `QuestionComposer.tsx:422-432`: Skip clears the current answer, advances, and
  directly validates/submits on the last page. Native
  `question_navigation.go:84-108` uses the shared skip mutation and preserves
  explicit skipped answers. Skip remains available on an already skipped final
  page, so correcting an earlier incomplete answer cannot leave the flow stuck.
- `QuestionComposer.tsx:522-525`: Enter on an option validates the whole batch
  without selecting that option first. This is distinct from free-text Enter.
- `QuestionComposer.tsx:227-237,475-497`: minimize keeps the request/header but
  removes body and question-navigation footer. Native `questionEditorFlow`
  retains component-local minimization; expanding does not steal text focus.
  Hide/reopen remains the separate existing action.

Native UI wiring lives in `internal/ui/interaction/question_navigation.go`.
Only the active question editor is mounted. The card heading follows that
question, and the progress count is visible beside collapse/expand controls.

## Answer retention and review

The page index lives on the same `draftSession` as the answers, not a disposable
UI component. Existing per-agent/per-call recovery reuses this object, so
hide/reopen and live-to-continued preserve both the page and all filled answers.
A stale key from a no-longer-current page cannot continue a different page.

An optional current-draft summary shows answers/skips/unanswered rows and can
navigate back to a chosen question. This is only a navigation aid. It adds no
mandatory confirmation step: the final Next/Skip still submits directly.
The original has separate compact reply summaries and settled-record review
(`QuestionReplyView.tsx:30-85`; `QuestionComposer.tsx:159-203,626-640`). Existing
native recorded/queued history stays separate and read-only; its implementation
file was not modified by this change.

## Preserved boundaries

- Multiline custom answers, Shift+Enter, IME commit protection, and one-shot held
  Enter handling remain in the native answer field.
- Sending freezes answers, pager, Skip, summary navigation and minimize. Store
  methods also revalidate exact request/session ownership and busy state.
- Pager and summary actions do not engage or extend the timed foreground claim.
  Only the active answer body reports focus through the existing scoped focus
  aggregation, and edits still use the CH50 hold path.
- Plan review, allow/reject policy, late-answer RPC fencing, Host cancellation
  and delivery-failure retry contracts are unchanged.
- Retention here is in-process. The original persists draft progress across
  client restart; durable restart storage is not implemented by this module.
  No live Host or desktop acceptance run was performed.

## Focused verification

A new native-headless regression first failed because the later question's
editor was still mounted. After implementation, the final focused package run
passed:

```text
GOTOOLCHAIN=local GOPROXY=off \
GOMODCACHE=/workspace/scratch/2d36477d56ec/.gomodcache \
GOCACHE=/workspace/scratch/2d36477d56ec/.toolchain/gocache \
/workspace/scratch/2d36477d56ec/.toolchain/go/bin/go test \
  ./internal/store/interaction ./internal/ui/interaction -count=1

ok whaleisle/internal/store/interaction 0.117s
ok whaleisle/internal/ui/interaction 0.438s
```

Coverage includes one-page-only mounting, unvalidated pager navigation, bounds
and foreign-session rejection, draft retention, hide/reopen, continued recovery,
single/multi choice behavior, final Skip redirect/submission, option Enter,
minimize/expand without focus stealing, summary navigation, Sending locks, and
the existing multiline/IME/timer/approval suites. Older tests now target the
answer field by its accessible label and explicitly return to the final page
after an incomplete-answer redirect.
