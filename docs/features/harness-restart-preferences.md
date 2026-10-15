# Harness restart preferences

Settings → Desktop exposes the existing `harnessAutoRestart`,
`harnessRestartMaxAttempts` and `harnessRestartBaseDelayMs` configuration.
The native editor accepts integer counts from 1 through 10 and base delays from
500 through 30000 milliseconds, matching the existing configuration loader.
The default remains enabled, three attempts, and 1000 ms.

The page shows the committed policy separately from its editable draft. Pressing
Save validates all three values before the existing atomic configuration update,
then checks the committed receipt. Invalid values, a disk write failure or an
external configuration conflict retain the draft and the old committed policy.
Unknown configuration fields remain preserved by `config.Store.Update`.

For a new ordinary failure/retry decision, the running Supervisor reads one
committed configuration snapshot. It invokes that provider outside its own
mutex. Saving does not call Start, Stop or Restart, reset attempts, or alter a
running task. An already scheduled retry timer keeps its original decision,
including its delay. New decisions use the new enabled flag, attempt limit and
base delay. Existing exponential backoff and reset-on-ready behavior remain.
A policy change alone cannot restart a Harness that has already stopped or
failed. Plugin-tree recovery and manually requested restarts are separate paths
and are not controlled by this preference.

The desktop save continues to include the existing other desktop preferences.
The restart-policy change adds no network permission, install, launch, process
termination, or workspace operation. Its runtime source is the same committed
Store used by the saved-policy display; an unsuccessful write cannot publish a
new policy through this provider.

## Verification boundary

The SP06 checks use temporary configuration directories, a synthetic Supervisor
without a launched child, and the native UI's headless tester. They cover input,
strict ranges, draft versus saved values, repeated saves, atomic rename failure,
external conflict, unknown-field preservation, provider lock order and retry
calculation. A single scheduled fake retry is generation-invalidated before it
can run; no Host or child process is launched. Windows compilation is a separate
integration check. These checks do not claim a real Windows interactive or live
Host acceptance run.
