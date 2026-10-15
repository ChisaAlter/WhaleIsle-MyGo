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

The sourceEventSeq reference is retained for richer domain presentation. The native generic command card does not combine manual-compaction summary and command lifecycle into the original single specialized card.

## Verification boundary

Authored command_contracts_test.go covers event pairing/history pages, output copying isolation, failure state, empty-array submission, multiline dispatch, skill fallback, and stale catalog replies/events. One in-memory source-contract/delimiter/JSON-fixture check was performed. The cloud filesystem failure prevents shell startup, so Go compilation and these repository tests have not run. No real Host/model/session data was exercised and no acceptance conclusion is implied.
