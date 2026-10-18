# Host task protection (native quit/restart)

- The supervised Host receives a per-launch, in-memory DSHD_TASK_CONTROL_TOKEN. Parent values are discarded; the token is absent from status/snapshots and redacted from child output. No credential is written to disk.
- The loopback-only client sends POST /dshd-task-control/inspect, acquire (Host-internal drain), inspect, renew, then the lifecycle action. Redirects are refused and Authorization uses Bearer.
- Running local sessions, Host active/scheduled work and unknown coverage require a cancel-default warning. An unavailable first inspection requires a separate explicit warning that unseen work may be interrupted; it never implies idle or completion.
- Acquire/drain failure, changed Host generation, expired lease, recheck failure or renewal failure blocks the action. Cancel/error after acquisition releases the lock; a successful quit/restart leaves the old Host locked until it stops. No cancel endpoint is called.
- Import busy checks and close-to-tray behavior remain. This change covers native restart and quit, not a complete port of Electron's update/install/reload or peer-coordination paths. The launcher's standalone Stop action now uses the same coordinator, with a warning that the Host stops while the desktop window remains open.

Verification: one focused fixture invocation with fake tokens and HTTP mock, no real Host or account started:
`go test ./internal/desktop/taskcontrol ./internal/harness -run '^TestHostProtection' -count=1`
Both packages passed. Cases cover clean, cancellation before/after acquire, unavailable inspection, drain/renew/commit failure, changed generation, expired lease, missing coverage, child-environment isolation, log/status token boundary and local-session consent while stopped. Windows compilation is delegated to the integration owner; no full suite or live GUI run is claimed.


## Launcher Stop increment

Verified real entry: `internal/ui/launcher/panel.go` renders the Close desktop button and dispatches `Model.OnStop`; the native handler now invokes `stopProtected`, sharing the inspect/acquire/drain/recheck/renew sequence. Duplicate stop/restart actions are excluded, import/quitting/supervisor identity is checked before the first side effect, and startup-error state is cleared only after an accepted stop. Cancellation preserves the window and Host. No quit, updater, installer, reload or peer path is broadened.

One incremental fake-token/mock check passed: `go test ./internal/desktop/taskcontrol -run '^TestStopProtectionContract$' -count=1`, using operation `stop` and verifying the same eleven success/failure branches. No real process was stopped. Windows compile remains the integration owner's check.
