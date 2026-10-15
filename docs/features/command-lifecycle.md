# Slash command contracts

Implemented against WhaleIsle source revision 90465b67ec16765631d54c195af19d9ff3c3a770.

## Connected behavior

- Empty command attachments serialize as an array, never null or an omitted argument.
- The current Host catalog owns dispatch. Commands advertising free input accept multiline arguments; unclaimed slash tokens remain prompt/skill input. Discovery failures retain the draft.
- The existing draft owner clears content only on accepted success; command errors preserve text and attachments for correction.
- Durable command/run and command/done records pair by commandId into a running/success/error card. Full input/output remain available, including multiline output and copy. A done-only history page remains visible and is enriched when an older run record is loaded, without duplicating the row or changing the paging journal.
- Command discovery runs asynchronously with visible loading/error/retry. Session/Host changes, reconnect, commands/change, and preset changes invalidate its cache; obsolete replies and event owners cannot repopulate it.

## Source contracts

- vendor/deepseek-harness/packages/interaction/commands/src/types.ts and index.ts
- vendor/deepseek-harness/packages/client/ui-commands/src/client/service.ts
- vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/command.ts
- vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/GenericCommandCard.tsx

Manual compaction now folds into one specialized card at the actual replacement checkpoint. The fold requires sourceCommandId and compactionId; a successful sourceEventSeq selects only the matching command's earlier summary. Log-only start/summary/end events and the checkpoint do not produce duplicate cards. A summary without a landed checkpoint leaves the command's complete generic error/no-op output intact.

Partial history can show a checkpoint before run/done: it retains the same command identity and an unknown/running outcome until settlement is loaded. Missing summaries use the complete command fallback, never borrowed nearby content. The card exposes counts, expandable Markdown summary, copy, and any failed-save/cancellation output after replacement. Shadowed transcript rows and the raw paging journal are retained.

Additional source contracts:
- vendor/deepseek-harness/packages/compaction/command-compact/src/index.ts
- vendor/deepseek-harness/packages/compaction/compaction/src/checkpoint.ts and types.ts
- vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/CompactionCommandCard.tsx and CompactionItem.tsx

## Verification boundary

Authored command_contracts_test.go covers event pairing/history pages, output copying isolation, failure state, empty-array submission, multiline dispatch, skill fallback, and stale catalog replies/events. One in-memory source-contract/delimiter/JSON-fixture check was performed. The cloud filesystem failure prevents shell startup, so Go compilation and these repository tests have not run. No real Host/model/session data was exercised and no acceptance conclusion is implied.

The manual_compaction_test.go fixtures cover one-card correlation, partial-history enrichment, summary-without-checkpoint fallback, post-checkpoint failure retention, source-reference identity isolation, and rejection of append-only checkpoint lookalikes. These tests are authored but have not run; the follow-on patch received one in-memory source/delimiter/JSON-fixture check only.
