# Import recovery and startup admission

Original contract reference: WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770`, `src/main/data-import.js`, `src/main/import-transaction.js`, and `src/main/launcher-gate.js`.

## Native behavior

- Session, skill, and preset imports prepare a complete private tree before replacing the selected destination. Overwrite replaces that selected tree instead of merging stale files from an earlier version.
- Attachments overlay a staged copy of the destination; destination-only attachments remain. Existing names follow the approved overwrite option.
- A live destination that appeared after scanning is skipped unless overwrite was selected. Selecting two different skill sources for one destination is rejected before any import mutation.
- A process-released OS file lock serializes native import and recovery. Cancellation is checked while staging and between items. Earlier completed items remain.
- The reserved destination-home `.mygo-import` directory owns the journal, staging and backup. Imported payload paths are never searched for recovery instructions.
- Before/after tree fingerprints detect changed or unexpected recovery data. Interrupted swaps restore the old tree or finish cleanup after a verified publication. Unknown, unreadable or unsafe state preserves copies and blocks subsequent import/Host startup.
- Cold startup checks recovery even with auto-start disabled. A recovered import pauses on the import tab once; a later explicit Start can continue. Restart only checks admission and never runs recovery against a live Host.
- Existing archive/snapshot persistence, Host task protection and plugin reinstall/build-approval behavior are unchanged.

## Limits

Recovery covers process interruption. It is not a verified OS/power-loss durability guarantee: file contents are synced, but directory-entry persistence ordering is not enforced. Publication is per selected tree or selected YAML file, not a transaction for the entire import. External tools do not participate in the native importer lock, and adversarial concurrent path replacement is not supported. Symbolic links and other non-regular payload entries are rejected rather than followed.

An unfinished legacy Electron `import-journal.json` is preserved and blocks native import/startup for its destination. Legacy sidecars are not translated: use the original desktop's recovery flow or repair the preserved transaction. Referenced credentials/OAuth migration remains outside this change.

## Verification status

Implementation/source review only at author handoff. No behavioral tests, acceptance run, real user-data import/export, installer execution or application termination was performed. The integration owner formats the changed Go files and compiles the Windows main/helper targets before publication.
