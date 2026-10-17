# CH68: explicit replies from a retained child occurrence

## Implemented scope

An already explicitly opened, live continuable child can retain its approval and
question UI when the selected root changes. Its submit facade captures the exact
pane occurrence, original root, Host client/mux, lease pointer, navigation token
and complete child address. The original `AgentInteractionTarget` gate remains
unchanged. Restored read-only leases cannot create a reply facade.

Scope creation and rendering do not send an outcome. Approval/deny, question
Next/Submit, free-text Enter, option Enter, final Skip, plan approval/unnamed plan
dismissal and retry each revalidate the captured scope and child pending record.
The original root and direct parent must still exist and be available in the
owner's current Host registry. The card's drafts, multiline/IME handling,
pagination, focus scoping and Sending freeze use the existing implementation.
Named plan discussion continues to hide locally without settling the request.
A retained facade cannot enable global auto-allow.

## Transport boundary

Live replies use the existing single `$events` sender. An `EventDelivery` field
on the result outcome is process-local (`json:"-"`); it binds the Host, event,
lease context and validator. `PostEventResult` rechecks this authority and waiter
identity before HTTP, then joins the event-stream context with the lease's
cancellation. A revoked callback neither falls back to the selected parent's
Store authority nor converts an invalid approval into a denial.

Continued-question replies keep the existing `userQuestions/answer` recovery
path, but bind it to the captured client and lease context, checking again in the
RPC goroutine. Exact occurrence release, accepted owner deletion/removal,
Host replacement/disconnect and event-generation revocation cancel the lease
and therefore its in-flight HTTP/RPC. A request already accepted by the Host
cannot be retracted by local cancellation. Delivery failures preserve the draft
and require an explicit valid retry; late completion from an older scoped result
cannot settle a newer response.

No new Host subscription, credential channel or authority registry was added.
Ordinary main-chat submissions and existing Host-controlled timed-question
behavior are unchanged.

## Focused verification

One focused run passed on 2026-10-17:

- `internal/dsh`: 0.012s, preflight identity validation/non-serialization and
  in-flight HTTP cancellation.
- `internal/store/session`: 0.006s, retained root switch with old gate still
  closed, stale token/child/Host/mux/occurrence/lease, unavailable root, revoked
  lease and restored read-only rejection.
- `internal/store/interaction`: 0.027s, all sending gestures, exact retry outcome,
  stale completion fencing, scoped question/plan outcomes and continued RPC
  cancellation; existing question wizard/plan/late-answer fixtures.
- `internal/ui/interaction`: 0.114s, rendered editor Enter revalidates after paint
  and retains Sending freeze; existing multiline and wizard fixtures.
- `internal/ui/agents` and `internal/ui/frame` compiled; the focused expression
  selected no tests in these two packages.

Command: `go test ./internal/dsh ./internal/store/session
./internal/store/interaction ./internal/ui/interaction ./internal/ui/agents
./internal/ui/frame -run 'Test(EventDelivery|RetainedInteractionScope|ScopedInteraction|QuestionWizard|QuestionCustom|PlanReviewContractFixture|LateQuestionAnswerRPC|RetainedAgent|ChildInteractions)' -count=1`

No real Host/model/tool action, original execution, whole-suite run or target-OS
GUI acceptance was performed. Depends on the retained read lease and restored
read-only marker implementation from the session stage.
