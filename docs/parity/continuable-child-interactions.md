# Continuable child approval and question interactions

Both the Agents navigation pane and an explicitly opened child-chat pane now
mount the existing approval/question cards for a live continuable child. This
reuses the desktop interaction store and its event IDs, waterfall acknowledgments,
question drafts, Sending guards and continued-answer RPC. It does not register a
second event receiver or send requests via the parent session composer.

A pane must still own its Host, navigation token, root and complete child address.
Only live `continuable` children receive controls. One-shot/external children,
paused/restored panes, historical pages and stale navigation remain read-only.
The interaction store's bound client must match the navigation client. Follow
baseline question/inbox projections are retained with their watermarks; current
Host control projections supply newer values. Child projection caches are cleared
when pane ownership changes. Settled child messages support approval call details
and question history without borrowing parent history.

Child cards disable page-wide approval shortcuts, while their focused-card input
and buttons retain normal behavior. The main and child mounts report separate
question-focus scopes. A frame aggregator prevents repeated rendering of the
same child or an unrelated parent from clearing a focused question's pristine
pause, and releases scopes when panes disappear. Explicit editing/take-time
behavior remains owned by the original question wait controller.

## Focused verification

2026-10-17: one invocation passed:

`go test ./internal/store/session ./internal/store/interaction ./internal/ui/agents ./internal/ui/frame ./cmd/whaleisle -run '^(TestContinuableChildInteractionScope|TestChildQuestionFocusFrameIsolation)$' -count=1`

The new tests cover current child ownership/projections; rejection of stale,
one-shot/external/unknown, paused and switched-root targets; parent/child/duplicate
pane focus isolation, hidden-pane release and interaction Host identity.
UI/frame/command packages compiled with no matching tests. No live approval,
question answer, Host session or native GUI interaction was exercised.
