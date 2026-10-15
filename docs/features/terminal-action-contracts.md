# Native terminal action contracts

Original reference: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`,
vendored `client/ui-user-terminal/README.md` and `TerminalWorkspace.tsx`.

The native workspace panel now supports independent New terminal groups,
selecting any shell, horizontal/vertical split groups capped at four panes,
closing one shell, and restarting only the active shell. Existing processes
remain alive across group/workspace switching and hide/show. Native Ghostty
continues to own input, output, resizing, selection, copy/paste and process exit.

Each shell has a stable keyed native view and focus binding. New/Split failures
leave existing processes intact and show their error. Closing the last shell
uses the existing panel-hide callback. Application shutdown closes all native
processes. A failed active-shell restart remains visible and retryable without
closing neighboring panes.

The old environment-refresh path no longer creates an invisible second Host
terminal, writes/resizes/renames/retains it, or starts a short-lived follow probe.
Opening the native panel owns only the native PTY(s) that are displayed.
No web terminal is claimed as connected to that view.

Validation: one bounded source lifecycle/binding review only. No shell, PTY,
Host session, Git operation or model call was executed. Runtime/compiler checks
remain unavailable because the cloud filesystem is full. Native Windows IME,
ConPTY focus and rendering are not claimed as exercised.
