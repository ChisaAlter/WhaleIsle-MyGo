# Provider profile removal

Source parity: Electron `ui-settings-models/src/client/ModelsSection.tsx`
`targetOf` / `removeProviderProfile`, and `store.ts` removal eligibility.

The Models settings page now lists configurable profiles by provider route,
namespace and exact settings path. Only writable, non-root profiles present in
`user` and absent in `base` offer removal. Account and protected base profiles
are excluded from destructive actions. Session model selection is unchanged.

Confirmation names the exact profile and warns when its page-managed credential
will also be removed. A credential is targeted only when the profile reference
matches the derived route reference and metadata confirms configured + writable.
No credential literals are requested or held by this controller.

Mutation order: `credentials/unset` (when applicable), `settings/mutate` with one
`unset` at the exact profile path, then reload directory/settings/credential
metadata. No namespace replacement and no expectedRevision are sent, matching
the original removal contract. Busy calls cannot issue duplicate mutations;
refresh/cancel invalidate pending confirmation. Snapshots copy target paths.

Failures stop the sequence. If the credential was removed but settings failed,
the UI says the profile may no longer work and supports retry. If settings were
removed but reload failed, it explicitly reports stale rows and unverified
availability. Neither partial outcome reports complete success. Failed refreshes
leave removal disabled until metadata can be verified again.

Verification: `go test ./internal/desktop/providerprofiles` uses only in-process
mock callers for exact payload/order, failure boundaries, busy protection,
confirmation, ownership and base/read-only restrictions. No live Host requests
or real credential access were used. Windows compile belongs to integration.
