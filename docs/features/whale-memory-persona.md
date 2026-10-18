# Native Whale memory and persona

The Assistant panel's Whale mode now has actual native controls for:

- Loading and editing the current Host's long-term memory.
- Saving memory through `dsh-whale/memory/replace`.
- Clearing memory through `dsh-whale/memory/clear`, only after the dedicated second confirmation. Cancel retains the draft. The Host writes its default template, not an empty file.
- Editing/saving extra persona, name, and how Whale addresses the user through field-scoped `dsh-whale/settings/update` patches.
- Explicitly restoring a local draft from the last successful Host read.

The existing original plugin WebView remains available, and desktop pet notification/care behavior is untouched.

## Contract and isolation

Verified against `ChisaAlter/WhaleIsle@e9f584be6d3b491ce777bb6823f30bc2bbc6f195`, `vendor/dsh-whale/client/client.js` and `vendor/dsh-whale/lib/index.js`.

`catalog` returns `memory`, `name`, `userTitle`, `personaText`, `homeDir`, and `sessionId`. It does not expose the dshbot revision or task interface. The native editor's request generation, draft revision, authenticated Host/client generation and canonical Whale home/session identity fence stale results. Drafts are in-memory only. Different Host or Whale identities never share drafts.

Writes make a read-only preflight against the loaded edited fields and identity, then send exactly one mutation and read back the current catalog. This catches already-observable conflicts. Upstream provides no compare-and-swap revision or idempotency key, so a concurrent change between preflight and mutation cannot be made atomic by this client.

No write is retried automatically. An uncertain response requires a fresh read before another write, and clear requires another explicit confirmation. A successful mutation followed by a failed read is reported as confirmed-but-not-yet-refreshed, rather than clean success. Dirty and newer drafts survive same-owner refreshes, failures and late receipts. Close, mode change, disable and navigation revoke outstanding UI tickets.

The UI checks upstream UTF-16 limits (memory 40000, persona 4000, name/title 64) to avoid silent truncation. Name/title normalization is confirmed by reading the Host value.

## Verification

Six new synthetic fixtures ran once each with `-count=1`: exact RPC envelopes and payloads, no write retries, limits, clear confirmation/cancel/failure, identity and draft revision isolation, accepted-write/read-failure status, field isolation, preflight conflicts, read-back races, and native clickable controls. No live Host, model, local user memory file or network service was accessed. The UI fixture passed with Fontconfig cache-directory warnings. Windows product compilation is performed by the integration owner; no old/full/race suite was run for this module.
