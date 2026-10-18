# Pet-specific Look model

Settings → Pet now contains a separate **桌宠看看模型 / Pet Look model** picker.
It reads only `session/modelCatalog`, lists image-capable provider/model pairs,
and saves the selected pair immediately under
`mygoDesktopPet.behavior.lookProvider` / `lookModel`. Both default to empty (Off).
Opening/refreshing the picker does not write configuration or take a screenshot.
The global `llm-vision-fallback` value and credentials are neither changed nor
used as a fallback. Existing native installs without these fields start Off.

An unavailable stored pair stays visible and is not cleared by a failed catalog
read. Off clears both fields and is available offline. Selection checks the fresh
catalog again before committing; a failed or conflicting save retains the prior
pair and all unrelated form drafts. The behavior reset confirmation now explicitly
says it clears this pair and turns Look off, alongside the eleven existing defaults.

Look is hidden when the pair is empty. Its native handler also verifies the
committed pair against the live image catalog before confirmation or capture.
The existing per-screenshot confirmation names the exact provider, model and Host.
After capture, it checks the same pair, catalog, Host identity, selected conversation
and pet window again. Changing/clearing/resetting the pair cancels an in-flight
attempt; a late frame cannot be repurposed for a newly selected provider.
The existing sender posts once to authenticated `dsh-whale/pet/look`, with the
captured pair unchanged, rejects redirects and never switches to direct API or
another provider after a failure.

## Original evidence and boundaries

All source references use commit `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`:

- [PetSection.tsx:130–163,195–230](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/deepseek-harness/packages/client/ui-settings-general/src/client/PetSection.tsx#L130-L230): image-only model catalog, paired saves, Off clears both, stale pair retained.
- [pet-settings.js:26–54,69–90](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/pet-settings.js#L26-L90): empty defaults, trimmed/capped IDs and independently normalized fields.
- [desktop-live2d.js:341–357](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/desktop-live2d.js#L341-L357) and [1252–1288](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/desktop-live2d.js#L1252-L1288): pet-specific route and refusal before capture when no look model is configured.
- [pet-chat.js:177–214](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/pet-chat.js#L177-L214): provider-bound request never falls back to generic desktop credentials.
- [vendor/dsh-whale/lib/index.js:446–563](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/dsh-whale/lib/index.js#L446-L563): actual upstream consumer only admits the screenshot into the resident session when its route exactly matches and supports images; otherwise it calls the selected provider/model directly, without using a foreign global vision fallback.

MyGo deliberately has no wiring-time model fallback or provider-less direct vision
request. A legacy/bare model remains display-only until the user explicitly picks
a provider/model pair. The existing Whale Host/plugin remains the consumer; it
must provide the above selected-route contract. This change adds no plugin,
credential loader, direct API call, screenshot format or transmission destination.
The original `approvalButtons` is a reserved field with no available approval
bridge, not a missing shipped approval workflow; see [the source boundary](pet-approval-reserved.md).

## Verification

One synthetic Go fixture invocation uses temporary config and a local fake HTTP
Host with an artificial 2×2 JPEG. It covers empty defaults, exact pair persistence,
unchanged legacy/global settings and drafts, unavailable/stale/provider-less routes,
conflict failure, Off/reset, cancelled late sends and one-target/no-retry failure.
No real screenshot, Host, model, account, credentials or user config was accessed.
Native UI/window operations are stubbed; catalog UI and Windows screenshot flow
remain interactive checks. The integration owner performs one Windows main build.
