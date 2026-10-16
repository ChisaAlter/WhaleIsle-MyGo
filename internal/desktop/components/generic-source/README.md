# Generic component service

The four service modules (`index`, `registry`, `store`, `lifecycle`) are preserved
from original commit `e9f584be6d3b491ce777bb6823f30bc2bbc6f195` source `src/launcher/components`. They use
only Node core modules. `worker.js` loads this embedded closure without extracting
source or installing dependencies and exposes it to the Go launcher over stdio.

The Go launcher supplies its Host-resolved Node and the matching runtime's
`components/samples` directory. No catalog entries or payloads are manufactured
when that directory is absent. Historical `components/registry.json` records,
including orphan installed components, remain visible. Normal catalog rendering
never launches Node or starts components. Actions initialize the worker on demand;
committed quit (or ownership pipe EOF) stops supervised components. Closing to tray
or cancelling task protection does not stop them.

Install copies and verifies a real bundled manifest, update retains one rollback
payload, rollback swaps active/previous versions, and uninstall preserves data.
The native UI retains the original rollback/uninstall confirmations. WhaleBridge
continues through its separate existing lifecycle handlers.

Prepared checks (not executed by the implementing worker):
- `node --test internal/desktop/components/generic-source/service.fixture.test.js`
  uses private fixture files, injected spawn/taskkill and never executes payloads.
- Go `TestGenericCatalogAndOriginalRegistry` and
  `TestGenericManifestEntryContainment` read private fixtures only.
