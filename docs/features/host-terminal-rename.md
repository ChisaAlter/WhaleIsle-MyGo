# Retained Host terminal rename

The native Host terminal panel now has a selected-title double-click editor and
Rename terminal button. Save/Enter sends one request; Cancel/Escape discards the
local edit. RPC failure, disconnection, or an uncertain response preserves the
exact draft for inspection and explicit retry. There is no automatic retry.
Blur does not save in the native editor. This is not full terminal UI acceptance.

## Pinned source and protocol

Original: `ChisaAlter/WhaleIsle@90465b67ec16765631d54c195af19d9ff3c3a770`.
Paths below are relative to `vendor/deepseek-harness/`:

- `packages/api/terminal-controller/src/index.ts`: real `terminal/rename`
  handler takes `{agentId, id, title}`, checks trimmed nonempty and at most
  120 JavaScript UTF-16 code units, then renames the specified terminal.
- `packages/api/terminal-controller/src/client/model.ts`: client trims the
  user title, avoids unchanged requests, and patches local title on success.
- `packages/client/ui-sidebar-terminal/src/client/TerminalTitle.tsx`:
  original double-click editor, Enter/Escape, 120-unit maximum, blur-save.

The native store binds the draft and request to captured Host epoch, session
(`agentId`), and terminal `id`. It has no attachment-ID or local-process rename
substitute. Host replacement invalidates results. Explicit close invalidates
pending rename completion without discarding the draft or closing another
terminal. Rename never creates, connects, writes, or replays shell input.
The native editor trims before the 120-UTF-16-unit validation and calls the
existing RPC only after Save/Enter. Failures never update the displayed title.

## Verification boundary

Five new in-memory fixtures run once with `-run '^TestHostTerminalRename'
-count=1`: wire identity/success, failure with preserved draft and explicit retry,
UTF-16 validation/cancel, replaced Host/stale result, and close/stale result.
No old T05 test, race test, live Host, shell, GUI, or native library was run.
Windows/UI compilation belongs to the integrator. Shell selection/remember,
multiple sidebar occurrences, selection-to-chat and navigation remain separate.
