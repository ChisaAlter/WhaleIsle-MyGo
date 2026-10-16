# Native release inputs (B33 / B34)

`package.ps1` now requires a desktop Host `node.exe` whenever runtime inclusion is enabled. `WHALEISLE_NODE_DIR` selects the local Node distribution directory; an explicit missing selection fails instead of falling back. Without it, the existing known local distribution locations are tried only when they contain `node.exe`. The copied executable must exist, be nonempty and have the same SHA256 as the selected input before `.runtime-staged` is written. The existing current-Host provenance verifier and separate mobile-runtime verifier remain unchanged. This is an input/copy contract, not a claim of Node ABI/native execution validation.

`WHALEISLE_PACKAGE_SKIP_RUNTIME=1` remains an explicit shell-only development mode. It passes `SHELL_ONLY=1` to NSIS. Direct NSIS compilation otherwise requires both `.runtime-staged` and `runtime/node/node.exe`; missing runtime inputs cannot silently produce a normal installer.

Every package, including shell-only packages, stages THA4 using the existing `pet-render-assets.mjs` verifier. Set `WHALEISLE_PET_RENDER_ASSETS` to a local resource directory laid out as `avatar/`, `ort/` and `LICENSE`; the default is `runtime/pet-render-assets`. Keep it outside `dist/whaleisle`, which is cleared by Windows packaging. After that cleanup, the stager verifies and writes all nine manifest-selected files, their provenance and the marker. Missing or changed resources stop packaging. NSIS requires the marker and includes all selected files plus `provenance.json`.

macOS `stage` accepts `petRenderAssets` in its config, with the same environment/default fallback, and invokes the same stager beside `Contents/MacOS/whaleisle` before app inspection. Its existing recursive app archive therefore contains the resources too.

The THA4 selection remains pinned to `90465b67ec16765631d54c195af19d9ff3c3a770` as recorded in `internal/desktop/pet/assets/render-manifest.json`; the e9 reconstruction baseline does not authorize silently changing resource hashes or the Host source pin. Model/license caveats in that manifest are unchanged.

No assets are downloaded by these changes. Required external inputs remain the target-platform Node distribution, complete verified Host/mobile vendor tree, pinned native terminal library, and the nine locally restored THA4 resources. Full packaging, target binary/ABI validation, GPU rendering, signing and installation are separate acceptance steps, not established by this wiring.

Narrow offline fixture: `node scripts/package-release-inputs-fixture.mjs`. It checks the source wiring and the existing THA4 stager's empty-input failure, without building or running native/model binaries. It does not claim PowerShell or NSIS execution coverage.
