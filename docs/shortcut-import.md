# Reviewed legacy web shortcut import

After the exact-source provider adaptation is built, product Settings → General
includes “Import legacy web shortcuts”. The application-owned adapter reads
`dsh.keybindings.v1` only when the user chooses Review. It does not read/write
preferences merely because the settings item mounted.

The authenticated Gateway previews current-platform web overrides against the
single MyGo desktop profile. Each saved override with a different existing
value shows both values and a Keep existing / Import legacy choice, defaulting
to Keep existing. Missing overrides are additions, including dormant plugin IDs
and two-key bindings; missing defaults are not synthesized. Null means an
explicit unbinding and is distinguished from an absent override.

Confirm sends the reviewed document, revision and explicit replacement IDs.
The Store rejects stale reviews, invalid/future documents, unreviewed choice
IDs and active command-key conflicts. A rejected batch does not partially save.
Only a successful atomic write becomes visible to native and product readers.
The old localStorage value is never deleted or rewritten. Re-import requires a
new explicit review and confirmation; there is no automatic migration loop.

The endpoint supports only preview/import on this shortcut document. The
profile is selected by the host platform, not a request parameter. Existing
Gateway session cookie and exact Origin checks apply. No file paths, arbitrary
Host calls or command invocation are accepted.

Targeted Go tests cover preview preservation, null/missing comparison, unknown
and double-key imports, replacement choices, CAS, key conflicts and failed
writes; authenticated-route tests cover foreign origins and profile injection.
The Node consumer fixture exercises Review → choose conflict → Confirm and
asserts no localStorage writes. Existing client-module lifecycle runs the real
packaged adapter including its helpers. Actual adapted Host bundle compilation
and live UI verification are tracked separately by the build workstream.
