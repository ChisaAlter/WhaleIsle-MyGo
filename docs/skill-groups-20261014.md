# Native skill groups

Settings → Skills uses the existing `skillInventory/list` catalog, independently of the composer `skills/list` snapshot. Tags retain inventory order; a multi-tag skill appears in each group and ungrouped skills appear last.

- Inspect an existing skill to edit description, content and comma-separated group tags. Existing tag buttons add/remove tags; an empty field explicitly sends `groups: []` to clear them. Read-only skills remain inspectable but have no editing or mutation controls.
- Group and row invocation switches operate only on writable skills and retain each skill's `userInvocable` flag. Rows update optimistically; failed writes roll back individually and show their errors. Pending changes block further group/save writes.
- Session catalog failure retries the global catalog with a visible notice. Detail/save/group requests retain the catalog's effective scope.
- Catalog replies from superseded sessions or requests are discarded. Repeated detail inspections only display the latest result. Save preserves the current catalog invocation flags so an older editor cannot undo a newer toggle.
- No skill invocation, installation, Host process, plugin revival, merge or deployment is performed by validation.

Verification: `go test ./internal/store/session -run TestSkillGroupsCoreFixture -count=1` passes against a local mock RPC server. The single fixture covers session fallback, read-only exclusion, multi-group membership, preserved user invocation, partial failure rollback, and explicit empty-group save. Windows UI/application compilation is handled by the integrator. Pixel styling, native menu parity and live Host acceptance are outside this change.
