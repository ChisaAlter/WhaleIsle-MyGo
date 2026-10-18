# Assistant and scheduler workflow parity

Source baseline: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`.
Native baseline: WhaleIsle-MyGo `d32f5767e25a46f885ec7ef153b9a04ef1599062`.

## Implemented

- Dshbot prompts now require an explicit Bot/room chosen from the current Host roster. The original `describe(botId)` and `bot/open({id, revision, scratchCwd})` contract resolves or restores its canonical session. The unchanged prompt is submitted to that captured authenticated session and only an `accepted:true` receipt clears the unchanged draft. No prefix-based routing to an arbitrary selected chat remains.
- Bot opening also uses `bot/open`, including cold or not-yet-created canonical sessions. Fresh returned item/session identities are checked. Host/client/plugin changes reject delayed results; persisted roster rows are displayed only for their originating Host.
- The assistant retains pending and failed drafts, preserves edits made during a request, and exposes the complete roster in a scrollable panel.
- Schedule refreshes rebase the captured expected rule onto the latest authoritative task while retaining edited fields. Host conflict/ended/not-found responses trigger a catalog refresh; inactive or unavailable tasks cannot be saved.
- Tasks link back to their original session only when it remains in the current live list and is not archived. Opening history never activates or restores a session.
- History requests use request identity and connection epochs, so an old response cannot replace a newer request for the same task or a new Host's result. Reattach clears obsolete busy/loading state.

## Original source contracts

- `vendor/dshbot/client/client.js`: `ensureCanonicalSession`, `profileCommand`, `describeCapabilities`.
- `vendor/dshbot/lib/profile-ops.js`: `open` resolves stable session identity. Although the original client sends revision, this operation does not enforce catalog CAS.
- `vendor/deepseek-harness/packages/api/session-controller/src/types.ts`: `SessionPromptValue` is `{ accepted: true }`.
- `vendor/deepseek-harness/packages/client/ui-schedule/src/client/TaskDetail.tsx` and `task-timing.ts`: authoritative refresh, retained local edits, complete expected rule.
- `vendor/deepseek-harness/packages/client/ui-schedule/src/client/session-link.ts`: live-list and archive navigation gate.
- `vendor/deepseek-harness/packages/client/ui-schedule/src/client/DeliveryHistory.tsx`: latest-request/epoch history semantics.

## Scope and verification

IM reconnect, workspace and agent-preset actions already use the real Cordis methods; full account setup and other plugin management remain available through the existing plugin page. Their absence from the compact native status panel is not counted as a missing action.

Targeted regression source covers selected Bot/open identity, the describe payload, one-shot draft acknowledgement, and schedule refresh/rebase behavior. Shell startup is blocked by the cloud filesystem being full, so these Go tests and compilation were not run in this module. This delivery is code implementation, not Windows, account, model, IM, or whole-project acceptance. No external messages, account connections, or real task execution were performed.
