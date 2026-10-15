# Retained Host terminal

Reference: `ChisaAlter/WhaleIsle@90465b67ec16765631d54c195af19d9ff3c3a770`.

The native bottom drawer is still the local MyGo/ConPTY terminal. The separate
right-sidebar **Host terminal** tab now uses the Host terminal controller; it
never substitutes a local shell for a Host terminal.

## Exact source and wire chain

- `vendor/deepseek-harness/packages/client/ui-sidebar-terminal/src/client/terminal.tsx` mounts `TerminalView`, renders complete snapshots plus ordered output, and offers reconnect/input takeover.
- `vendor/deepseek-harness/packages/api/terminal-controller/src/client/model.ts` binds terminal identity to a Session and Host lifetime; each follow generation has a new attachment; missing terminals are never recreated during recovery.
- `vendor/deepseek-harness/packages/api/terminal-controller/src/client/retention.ts` waits for a physical `retained` acknowledgement before following output.
- `vendor/deepseek-harness/packages/api/terminal-controller/src/index.ts` defines `terminal/list` and `terminal/retain` with `sessionId`, and `terminal/environment`, `terminal/create`, `terminal/follow`, `terminal/write`, `terminal/resize`, and `terminal/close` with `agentId`. Follow/write/resize additionally bind the terminal `id` and `attachmentId`.
- `vendor/deepseek-harness/packages/api/terminal-controller/src/types.ts` defines snapshot/output/state frames, nullable exit code, controller identity, and monotonically ordered output sequence.
- Pinned source blobs checked through GitHub: index.ts `24bf1a1f01179a807cff541c2a9e9ffde7cd6bd5`; types.ts `5e1df56868e7884f68308e2946409ffe556bc27e`.

The MyGo generated endpoint registry already describes this wire protocol, but
before this change its native terminal call chain ended at `myterm.New` without
`Options.Conn`, creating only local PTYs. A generated RPC definition and a web
runtime were not native retained-terminal functionality.

## Native path

`frame.HostTerminalStore` → `hostterminal.Store` → authenticated existing DSH
client/mux → terminal retain/follow and one-shot control calls →
`terminal.HostView` → native Ghostty `Options.Conn`.

Opening the tab alone starts no process. Refresh lists retained identities;
Connect explicitly attaches to one; New explicitly creates a Host shell using
the Host system-user environment. Close requires the selected identity's close
confirmation. Input/control is bounded to the displayed Host epoch, Session,
terminal and fresh attachment. Host replacement rejects stale UI actions and
callbacks. Hidden/session-switched views keep their physical window holds but
release output/input attachments. App detach releases holds without sending
terminal/close; Host retention/reclamation policy continues to apply.

The existing mux supervisor re-establishes desired holds after carrier recovery.
Only acknowledged holds can open a new follow. Every follow must begin with a
complete snapshot, followed by contiguous sequence numbers. There is no cursor
parameter or invented reconnect RPC: cursor/modes come from the snapshot's VT
screen bytes. Ghostty parses the complete frame before the next output frame is
consumed. A fresh native renderer and immutable connection are used for each
attachment. A read-only renderer keeps the Host's authoritative grid rather than
reflowing it to local view width; writable resize proposals respect Host limits.

Writes and resizes are serialized. Any uncertain input response disables that
attachment's input, and no queued input is replayed on recovery. A failed create
keeps its caller-generated ID available for list recovery or explicit close;
it is never recreated automatically. A failed close stays visibly retryable and
is never automatically repeated. Host exit does not create a replacement shell.

## Validation boundary

New in-memory synthetic stream fixtures cover retention-before-follow, correct
wire scopes, fresh snapshot/attachment recovery, local render acknowledgement,
sequence/cursor-byte fidelity, invalid frame rejection, controller takeover,
Host/session isolation, stale UI capabilities, hidden retention, explicit close,
input limits and uncertain create/write/close outcomes. Tests use `-count=1`.
No live Host, shell, native terminal process, native DLL/GUI, race run or old test
suite was started. Windows compile verification belongs to integration; actual
visual/keyboard/IME/remote-environment behavior remains a future explicit user
validation step. Native local-PTY behavior and its existing entry remain intact.

## Deliberate interaction scope

Selected-title editing is added by [Host terminal rename](host-terminal-rename.md).
Shell discovery and client-local memory are added by [Host terminal shells](host-terminal-shells.md).

This is the narrow retained-Host-session path, not acceptance of all original
terminal interactions. These original interactions remain outside this module:

- The original multiple sidebar occurrences, saved occurrence/navigation
  bindings, and the dedicated new-Host-terminal keyboard command. The native
  view presents one active screen with a retained-identity list per Session.
- Read-only terminal text selection parity. The native read-only screen disables
  interactive input/selection and provides Copy terminal content; it does not
  reproduce xterm's separate read-only selection interaction.
- Retained-screen selection-to-chat and application-specific URL/file routing.
  Existing local-PTY selection/chat and split-pane controls are preserved, but
  are not evidence that the new Host screen has those same commands.

The source also contains recovery/cleanup UI components whose registration is
commented out in the pinned sidebar plugin. Their mere presence is not treated
as an active original feature or used to claim parity. This module offers
explicit Refresh/Connect and an in-tab error/explicit-close retry path.

The actual main-window OnHide/OnClosed lifecycle release is integrated by the
sole integrator: hide drops the output/input renderer outside mainState.mu while
keeping Host retain; close detaches the Store without terminal/close and drops
the renderer. Existing OnShow invalidation restores only through normal view
state. These wiring-only corrections do not constitute Windows runtime testing.
