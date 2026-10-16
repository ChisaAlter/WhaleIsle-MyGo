# PET09 / PET10: packaged renderer chain

Source: [WhaleIsle e9f584be](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/renderer/pet-live2d.js).

## Real upstream entry point

`mount()` at lines 4920–4957 creates the neural session first and calls `loadRig()` only after session creation fails. `paint()` likewise checks `session` before `rig.ready`. The earlier rig-section comment claiming rig-first is stale; it is not the executable product contract.

Native startup now follows neural (when enabled and verified sidecar assets are available), then the bundled original rig, then a selected sprite sheet or a basic animated head. An explicitly selected skin skips the default rig. Initial failures and later neural/rig frame failures advance once through the same chain. Closing, hiding the document, sleep and late resource completions retain their lifecycle boundaries. Nothing downloads or installs a neural runtime.

## Actual packaged rig

`rig.go` embeds the original `rig/manifest.json`, `shell.png`, `tail.png`, upstream MIT LICENSE, and the adapted `pet-rig.js`. No separate staging step, external SDK, directory scanning or arbitrary asset URL is required. The fixed-origin GET/HEAD allowlist exposes only the renderer script, manifest and two PNGs.

`pet-rig.js` composes the original raster layers around the original feet, neck, grab and tail anchors. It retains breathing, tail swing, pose-channel head offsets, state transforms and expression-category crossfade from upstream. The native page feeds real care, sleep, pat, angry, pickup/throw, and native roaming/facing state into these transforms. The sleep silhouette is fitted to the bounded native canvas because upstream paints onto a whole-screen overlay.

All ten upstream manifest shell entries point to `shell.png`; even the separately named shell-expression files in that fixed commit have the same Git blob SHA. They do not contain ten distinct expressions. The renderer does not invent replacement art or claim rig facial deformation. Blink/mouth neural deformation still requires a functioning THA4 model/runtime.

Asset provenance at fixed e9f584be6d3b491ce777bb6823f30bc2bbc6f195:

- `manifest.json`: Git blob c7d164fa3cc00d609a37a33d8298012174c38d83
- `shell.png`: Git blob d268cf1c5ce601a45b997b7c90ab1ce3ab76e7a1; SHA-256 8a30393afc9f64914c8f5f51389c827c8e2dee6702c17b0ccdaf6d294258edb3
- `tail.png`: Git blob ae16cf0dc53a4c2e8dd5281939f287a82f1742ae; SHA-256 2e2b60beb5552f3040cf9daa8473c9ed488b3897fa83beff5330964d48bb1836
- Root MIT LICENSE: Git blob aab16e641cf2b3673aca280c5491f6fda3cbf2b5, Copyright 2026 Chisa, retained in the bundle. No distinct license appeared in the rig directory; the underlying art-rights chain was not independently established.

## Dynamic fallback and limits

The last-resort path paints either the user's eight-frame sprite sheet or the existing bundled head, with timed breathing and pose-driven transforms. It is explicitly labeled basic animated fallback, not THA4 or full original-expression parity. It works without a model, and timers stop on sleep/close. No downloaded model was executed in validation.

This change does not replace native window movement with upstream's fullscreen overlay, port GPU-buffer packing/Anime4K super-resolution, fabricate missing expression artwork, or establish pixel-level GUI parity. The existing THA4 CPU tensor path and verified sidecar packaging remain in place.

## Focused verification

- `node scripts/pet-rig.test.mjs`: actual compositor operations, deduplicated image loading, state/time-dependent transforms, manifest URL bounds and close.
- `node scripts/pet-render-chain.test.mjs`: neural provider order before rig, initial failure transitions, sprite/dynamic fallback, repeated painting, close and late-load prevention using an offline DOM/canvas protocol fixture.
- `go test ./internal/desktop/pet -run '^(TestPackagedRigResources|TestRigPageIntegration)$' -count=1`: embedded source-image hashes, PNG decoding, exact routes, GET/HEAD and page integration.

These are focused code/protocol checks. They do not establish successful inference on real WebNN/WebGPU/WASM hardware or live MyGo visual acceptance.
