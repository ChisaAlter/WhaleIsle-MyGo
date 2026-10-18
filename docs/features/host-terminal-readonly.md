# Retained Host read-only selection

The Host terminal now uses a read-only input policy instead of disabling its
entire native view. Real Ghostty drag/word/line selection, Select All, Copy and
local scrollback remain available without input control. Copy selection reads
`Terminal.SelectionText()`; it never falls back to full-screen/scrollback text.
The previous whole-content toolbar copy is replaced by this explicit selection
copy. Existing local PTY features remain unchanged by the default-writable API.

## Pinned source and real renderer capability

Original `ChisaAlter/WhaleIsle@90465b67ec16765631d54c195af19d9ff3c3a770`:
`vendor/deepseek-harness/packages/client/ui-sidebar-terminal/src/client/terminal.tsx`
(Git blob `99f2d19fc22a8de605f68710031dbbf74a3b7aa9`) uses xterm's
`disableStdin = !state.writable`, leaving the selected text and viewport usable.
It keeps the authoritative Host grid when read-only and does not call fit/resize
without control. The native Host store already preserves that grid and validates
write/resize capability against the active attachment.

Actual native selection chain:
`mygo/plugins/terminal/view.go` pointer gesture -> Ghostty SetSelection ->
`mygo/plugins/terminal/selection.go` SelectionText ->
`internal/vt/terminal.go` selection formatter (soft-wrap joining and trailing-space
trimming). This is an existing real API, not reconstructed text or screen OCR.

Important evidence boundary: the original **local PTY** drawer has selection-to-
chat formatting and draft insertion in ui-user-terminal/selection.ts and draft.ts
(blobs `2866e8fcbccc1e48c672d5d2ec1aaf7a986eb5ec` and
`4c746d1a841a9d15b9ee47c88e9c7a132ff2eba6`). Those are not evidence of a retained
Host quote action. This increment does not add a Host quote/draft button or alter
the existing local PTY Composer bridge. Navigation and multi-occurrence remain
out of scope.

## Input and identity boundaries

- Options.ReadOnly starts the Host renderer blocked before it parses a snapshot.
  SetReadOnly follows the current Host attachment's writable state before Feed
  and before building the view. Its default is false for existing local PTYs.
- The shared input queue rejects every push while read-only, including terminal
  query replies. Entering read-only clears queued input; restoring control does
  not replay it. IME text/preedit, key press/release, direct send/paste, menu or
  shortcut paste, focus reports, mouse reports, and alternate-screen wheel key
  synthesis are blocked. Read-only wheel/selection autoscroll stays local.
- Resize changes still update the local emulator, but read-only renderers do not
  send new resize requests. Host grid constraints remain authoritative.
- A write already handed to a Conn cannot be recalled. The Host Attachment also
  validates its current controller/identity immediately before RPC; it remains
  the final capability guard. This module adds no input retry/replay mechanism.
- Copy toolbar, keyboard and context-menu actions revalidate the captured Host
  epoch, Session, terminal and attachment, including current visibility, parsed
  snapshot and not-closing status. Switching/hiding/reconnecting cannot copy a
  stale renderer into a new target. Selection never invokes Host write/create.

## Verification boundary

Four new synthetic fixtures run once with
`-run '^TestHostTerminal(ReadOnly|Selection)' -count=1`:
queued input dropped/no replay; key/IME/paste/report suppression; selection
command routes plus copy authorization; and read-only Host copy identity across
wrong Host/session/terminal, close, hide and attachment replacement.
The renderer fixtures instantiate an empty Terminal and never call New, Load,
Feed, any native Ghostty function, or any actual clipboard service. The Host
fixture uses in-memory streams only.

The real Ghostty gesture/scroll chain is source-inspected, not runtime-tested.
Windows UI compilation belongs to the integrator. No native DLL/GUI, actual
terminal/shell, live Host, old T05/rename/shell fixture or race run was executed.
This focused evidence does not constitute full native terminal UI acceptance.
