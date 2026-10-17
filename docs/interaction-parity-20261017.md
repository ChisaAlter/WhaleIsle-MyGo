# Interaction parity: CH50 and B04

Implemented 2026-10-17 against the source audit of original WhaleIsle
`a8c931da37376ce7dc7989e6f30b0db885e06d26` and MyGo remote `e792363`.
The original was read only. The audit inventory remains a pre-fix snapshot.

## CH50: timed question foreground ownership

- `internal/store/interaction/question_wait.go` owns the request's countdown,
  remaining time, focus pause, permanent first-edit/take-time hold, generation,
  repaint wakeups and atomic timeout settlement.
- `user_questions_rpc.go:30-52` opens the existing Host `attachWait` claim and
  starts the state machine from its actual `remainingMs`, without inventing a
  deadline or extending a Host API contract.
- `questions.go`, `store.go` and `internal/ui/interaction/card.go` connect the
  state to answer edits and the visible answer body's focus. The independent
  `internal/ui/interaction/question_wait.go` displays localized remaining time
  and the explicit “Take my time” button.
- A pristine focus pause resumes with its preserved remaining time. First edits
  or take-time hold the claim until explicit answer/cancellation. Hidden and
  remounted cards retain the draft-owned hold. Focus moving to another session
  releases only the old pristine pause.
- Timer callbacks revalidate event, agent, call, draft and generation under the
  same lock as edits. Continued questions cannot acquire a live deadline.

Original references: `ui-user-questions/src/client/QuestionComposer.tsx:254-291,
456-472`; `ui-user-questions/src/client/contract/slots.ts:328-350,450-508`
under `vendor/deepseek-harness/packages/client`.

## B04: complete approval arguments and requester context

- `internal/store/interaction/approval_context.go` decodes complete argument
  values, with no 2,400-byte truncation. It accepts the existing event argument
  fields as fallback, and correlates ordinary approvals to original tool start
  records by exact call ID and selected session.
- `internal/ui/frame/question_history.go:9-18` invokes that correlation using
  the existing non-streaming transcript collection. Result/settlement records
  cannot overwrite start arguments, and once resolved the original is retained.
- Delegated approvals read only Host-provided `requester.call.arguments`, with
  matching call identity. Requester label, session and resolved display-only
  working directory appear above the arguments. Mismatch produces an explicit
  warning and never falls back to the presenting parent's arguments. A request
  ID is never treated as a tool call identity for correlation.
- `internal/ui/interaction/approval_context.go` displays full arguments in a
  bounded scroll viewport and copies the complete value. Existing allow/reject
  actions, keyboard handling, request ownership and approval policy are intact.

Original references: `ui-chat/src/client/chat/ApprovalCommand.tsx:32-48` and
`ui-approval/src/client/contract/slots.ts:59-73` under the same vendor root.

## Focused verification

Toolchain: `.toolchain/go/bin/go`, `GOTOOLCHAIN=local GOPROXY=off`, populated
`GOMODCACHE=/workspace/scratch/2d36477d56ec/.gomodcache`, and
`GOCACHE=/workspace/scratch/2d36477d56ec/.toolchain/gocache`.

Both reported bugs were reproduced before the fix:

- CH50 `TestTimedQuestionEditingRetainsForegroundClaim` failed because editing
  before the fake Host opened a zero-remaining claim still timed out.
- B04 `TestApprovalRequesterArgumentsAreComplete` failed with empty requester
  arguments. The fixed test also checks a 5,000-character command is uncut.

Final module check:

```text
go test ./internal/store/interaction ./internal/ui/interaction ./internal/ui/frame -count=1
ok whaleisle/internal/store/interaction 0.129s
ok whaleisle/internal/ui/interaction 0.389s
FAIL whaleisle/internal/ui/frame
```

The frame package compiled, but its unrelated layout test panicked in concurrent
workspace-pane work: `TestPopulatedComposerCardKeepsSharedAxisAndFitsCenter/1200`,
`WorkspacePanes.syncRoute` at `workspace_panes.go:144`, from
`workspaceColumns` at line 359. This was reported to the integration owner; it
must not be described as a passing full-frame suite.

A precise regression for the modified per-frame bridge was then added and run:

```text
go test ./internal/ui/frame -run '^TestApprovalContextFrameBridge$' -count=1
ok whaleisle/internal/ui/frame 0.021s
```

Coverage includes focus pause/resume, edit-before-claim, take-time ownership,
hide/reopen, untouched timeout, cancellation, delegated mismatch isolation,
portable display paths, cross-session call collisions, PTC start records,
settlement/duplicate rejection, immutable original arguments, and actual
waterfall/frame wiring without changing allow-once decisions.

## Remaining acceptance boundary

These are source implementation and focused fake/in-process fixture results,
not live Host acceptance or a desktop visual check. Runtime behavior remains
unverified. The broader audit's child-pane interaction mounting, parent
availability gates, question busy-state edits and other distinct rows remain
separate work; this change does not claim to close them.
