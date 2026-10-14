# Native lifecycle protection (2026-10-14)

Reference: Electron `docs/features/task-protection.md`, `close-behavior.md`, and actual `src/main/task-protection.js` / `index.js`.

## Implemented scope

- Window close with `closeToTray` hides the window immediately, including while a session runs. It does not enter quit confirmation, resume-state failure dialogs, or resource/Host cleanup. Explicit tray quit continues through the existing quit and resume-save gates.
- Native restart entry points retain import-busy priority. Known running sessions prompt with Cancel as both default and Escape action. Cancel or dialog failure never calls `Supervisor.Restart` or cancels a session.
- Concurrent restart requests are ignored while confirmation is pending; the import, quit and supervisor identity checks run again after confirmation. Dialog errors appear in the launcher without changing the existing shutdown implementation.

## Deliberate limits

This is a local session-snapshot guard, not complete Electron task-protection parity. It does not inspect Host jobs, schedules, external connections, in-flight admission, PTYs, or components, and it does not perform inspect/acquire/drain/inspect. Unknown or stale session snapshots can therefore miss work. Launcher stop, updates, Host stop semantics and existing explicit-quit/resume recovery are unchanged. The first-hide system toast is not added here.

Electron's current task-protection implementation treats explicit quit/launcher stop as preconfirmed, while the older close-behavior feature text still mentions running-task quit confirmation. This patch preserves MyGo's existing explicit-quit confirmation rather than deleting protection without the Host coordinator.

## Verification

A focused platform-independent decision fixture covers all close preference/quit combinations, idle restart, active-task cancel, explicit consent, and dialog failure. No real Host or user process was started or stopped. Native dialog rendering and cumulative Windows compilation are separate integration checks; this is not a full-suite or real-process certification.
