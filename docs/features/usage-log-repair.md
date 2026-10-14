# Native usage log repair

Source contract: WhaleIsle-Electron/vendor/dsh-usage-panel/src/shared/contract.ts,
client/api.ts, host/index.ts and host/session-repair.ts (current-source snapshot).

- Only IDs reported by overview.coverage.failedSessionIds can be selected.
- Selecting Repair opens native confirmation; it sends nothing. The confirmation
  names the exact session and explains decoding, zero-based event renumbering,
  timestamped backup, and rewriting/replacing the on-disk log.
- Confirmation calls `usage-stats/repair.session` with `{sessionId}` through the
  existing authenticated plugin RPC client. No invented preview/rollback route.
- The host owns backup and mutation. Native code never opens session log files.
- Success shows returned `repaired`, `backup`, `bytesBefore`, and `bytesAfter`.
  Rescan updates statistics; if the failure hint persists, restarting dsh may be
  necessary to clear host memory. A successful session cannot be repaired again
  in the same attachment from an old overview.
- While repairing, duplicate repairs and overview scans are blocked. Refresh
  invalidates pending confirmation. Detach/reattach cancels the request locally
  and discards any old response; cancellation does not undo a host-side write.
- Failures preserve the last statistics, explain that timeout/disconnect does not
  prove no write occurred, and offer explicit re-confirmation after rescan.
  There are no automatic retries.

Validation: one mock-only `TestRepairNativeFlow` covers exact RPC/auth/payload,
confirmation/cancel/ID guards, refresh invalidation, duplicate suppression,
failure retention, successful result fields, stale-candidate suppression, and
attachment changes. No real host or real log file is accessed by this fixture.
Windows compilation is performed by the integration task.
