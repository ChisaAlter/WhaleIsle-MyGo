# Per-process HTML generation gate

A saved `visual-replies.enabled=true` is user intent, not proof that this desktop
process has passed isolation/capture verification. MyGo preserves that preference.
The explicit settings enable flow can run the bounded synthetic native Probe.
New desktop processes start with readiness=false; no implicit Probe runs at boot.

The MyGo safety patch changes only the existing shared requireEnabled checkpoints
in the pinned visual-replies tools. Each preview/publication checkpoint also reads
GET /desktop/html-preview/status using the existing desktop bearer token, a
3-second bounded signal, no cache, and no redirects. HTTP errors, absent service,
malformed status, available other than boolean true, or failed Probe block the
operation. Both tools recheck before attachment persistence and before returning
published output. The status handler reads current-process readiness without
probing, changing preferences, or accessing accounts. Historical authorized
readVisualReply remains independent.

The original 90465b67 source checkout is untouched. Build preparation applies the
small, exact-base-SHA256 patch from scripts/current-host-readiness.mjs to its own
copy. visual-readiness-patch.json records ID, original hash, patch hash, and patched
source hash. The successful build manifest records this patch explicitly: this
is fixed official source plus an audited MyGo safety patch, not an unmodified
upstream binary. finalize/verify require compiled gate markers and matching patch
provenance; stage copies the patch record with the artifacts.

The MyGo launcher refuses a current Host with visual-replies but without verified
matching safety-patch provenance. Merely suppressing a UI capability would leave
an old Host able to restore persisted enabled=true, so launch gating is required.
Legacy customized runtimes without visual-replies remain unchanged. Reader-only
capability detection does not authorize generation.

No real Host, account, model, or network service is used by the gate fixture. It
checks saved true/current-process false rejection, successful current readiness,
strict boolean status handling, both tools' checkpoints, and untouched preference.
Configuration fixtures use the fixed source checkout as their explicit argument.
