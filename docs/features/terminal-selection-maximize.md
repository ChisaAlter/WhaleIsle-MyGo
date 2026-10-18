# Terminal selection and maximize actions

Original contract: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`,
vendored `ui-user-terminal/TerminalWorkspace.tsx`, `selection.ts` and `draft.ts`.

- Maximize expands the native drawer to at most 75% of its center column, while
  preserving the previous split position. Restore returns to that position;
  dragging the divider exits maximize and explicitly sets a new saved size.
- The native terminal binding exposes only its current selected text through a
  locked getter. It does not infer a selection from scrollback or clipboard.
- A nonempty selection exposes Copy selection and Add to chat. The latter
  appends a fenced terminal fragment to the exact selected conversation draft,
  preserving existing draft text and attachments.
- Missing/changing conversation, an active send or an unfinished IME composition
  blocks the local insertion. No model request or message send is made.
- CRLF/blank-line normalization follows the original contract; a sufficiently
  long backtick fence preserves arbitrary selected output inside that block.
- Existing multi-PTY lifecycle and per-pane focus binding are unchanged.

Validation: bounded source/data-flow and layout-state checks only. No native
PTY, clipboard action, model or user-file operation was executed. Compiler and
Windows native runtime checks remain blocked by the full cloud filesystem.
