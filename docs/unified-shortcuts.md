# Unified native/product shortcut preferences

MyGo owns one `shortcutprefs.Store` per main application state. Native settings,
menu/keyboard dispatch and authenticated product Gateways use it. Its original
schema-2 document lives in config.json's `goUnifiedShortcuts` raw key. This does
not open or modify Electron's independent keybindings.json. Ordinary config
updates merge the latest raw keys under the existing Store lock.

The first native initialization atomically copies only explicit known legacy
`goPageShortcuts` values into the new document. Empty strings become null;
missing values inherit defaults. The legacy object and unknown metadata remain
untouched. An existing unified document is never overlaid again, including
after resetting a migrated binding. Damaged preferences are read-only.

The native editor submits only changed IDs with the revision at draft start.
Product saves and catalog changes make older drafts stale; use “Discard drafts
and reload” to review the current snapshot. “Use default” deletes an override;
reset-all affects native-owned, representable rows only. Advanced bindings and
unknown plugin entries survive other edits. Unsupported bindings are displayed
and not silently replaced by an old native default.

## Product build adaptation

`build-harness.ps1 -UnifiedShortcuts` and
`build-current-host.ps1 -UnifiedShortcuts` explicitly opt into the tracked
`current-host-shortcuts.mjs` adaptation on the build copy. It requires the exact
SHA-256 of original e9f584be6d3b491ce777bb6823f30bc2bbc6f195 file
`vendor/deepseek-harness/packages/client/shortcuts/src/client/index.ts`:
`50f36616c5e4af3c4844e2737068e78152e9cc0bb61be0cef9e9d7101409a457`.
A source mismatch fails before patching; it does not guess a replacement.
An idempotent patch record is written alongside the copied Harness source.
The original source checkout is not modified. Existing runtime bundles are not
patched at startup or silently rebuilt.

Only the shortcuts service reads the explicit `whaleisleShortcutProvider`
capability. It sets that service's runtime/profile/storage; it neither creates
`dshDesktop` nor changes the document's global platform flag. The product-only
preload is exact-origin and top-frame scoped. Requests use the existing
Gateway's session cookie and exact Origin check. The bounded API admits only
catalog get, preference edit and snapshot; it has no Host/file/command RPC.
Native accepted-registry physical key delivery remains the preceding module's
separate protocol. Native and product edits read one revisioned document.

Canonical shared IDs are the e9 workspace commands (new/search/rename/fork/
archive/add-workspace), settings.open, shortcuts.open and explicit aliases
sidebar.toggle → sidebar.left.toggle, tools.toggle → surfaces.toggle,
terminal.toggle → terminal.drawer.toggle. Other native actions retain a
`mygo.` namespace until equivalent original actions are confirmed. Compatibility
workspace.open/tools.toggle.alternate retain explicit saved bindings but no
second default owner. Native unsupported macOS Command+Backquote stays visible
without stealing the system chord.

## Verified and remaining

Go Store tests cover strict schema, unknown/double-key preservation, CAS,
conflicts and failure retention. Integration tests cover both editors' shared
save, stale rejection, migration/reset and unrelated settings writes. The Node
fixture verifies exact-source adaptation, idempotence, absence of dshDesktop,
origin isolation and subscribe disposal. Existing DSK10 settings interaction
still passes.

The full adapted Host bundle has NOT been built or exercised in a live GUI.
An unadapted bundle continues using its existing web preferences. Old
`dsh.keybindings.v1` localStorage is left intact; reviewed import with explicit
conflict choices is a separate pending module. Two-key execution, guest/iframe
routing and non-Windows native priority remain outside this storage change.
