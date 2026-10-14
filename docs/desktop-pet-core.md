# Native desktop companion core

The MyGo shell now owns a small independent transparent, frameless, topmost sprite window. Settings → Pet provides an explicit enable switch, local skin selection, refresh and save. It works offline, without a Harness connection. Default is off; no scans, timer or pet window are started while off. Refresh scans only when explicitly requested, or when opening an enabled companion. Position restoration and disconnected-screen clamping use MyGo's existing `StateKey` window-state implementation. Drag the character to move the window; use Settings to hide it. Closing the app destroys the companion without changing its enabled preference.

Configuration is stored under `mygoDesktopPet` in the existing conflict-checked config store. Electron `pet` and `live2dPet`, including enabled state, growth and unknown fields, are not migrated or rewritten. A saved unavailable skin is preserved and falls back visually to the bundled Whale image. Select the empty entry for that image. Local Codex packages are read from `${CODEX_HOME:-~/.codex}/pets`, using directory names as stable IDs. Only bounded `pet.json` and confined 1536×1872 / 1536×2288 WebP sheets are read; scripts are never executed, symlink escapes and traversal are rejected, and nothing is uploaded or downloaded.

The page is served from a fixed `whale-pet://pet` resource allowlist. Navigation and new windows are blocked. The new opt-in MyGo `PageOptions.WindowControlsOnly` mode only permits DOM readiness and native CSS dragging; application methods, broadcasts, file drops, maximize and other bridge messages are blocked. `DisableBindings` retains its original behavior. This is a small rectangular window, not an all-screen click-through overlay: its rectangle can intercept clicks. Fine-grained shaped hit regions and nonfocusable native windows are not claimed here.

## Source and licensing

`internal/desktop/pet/assets/pet-head.png` is copied unchanged from the supplied WhaleIsle-Electron `assets/pet-head.png` snapshot. Its original repository MIT license, copyright 2026 Chisa, is retained beside the asset. This is the existing fallback badge, not newly generated animation art. The sprite atlas dimensions/idle-row convention follow Electron `src/main/desktop-pets.js` and `src/renderer/pet.js`; the image-only renderer is a minimal new adapter. User-selected packages retain their own original licenses and remain in their directories.

## Renderer parity boundary

Electron's feature named `desktop-live2d-pet` actually uses THA4/ONNX neural animation; it is not evidence of a Cubism implementation. The supplied source includes `pet-live2d.js/html`, `ort.all.min.js`, `ort-wasm-simd-threaded*.mjs`, `anime4k.js` and rig JSON manifests. It does not include `avatar/model.onnx`, `avatar/character.png`, ORT `.wasm`, rig PNG parts or `states/*.webp`. No Cubism SDK, `.moc3` or `.model3.json` references/assets were found in the checked source. This module does not claim THA4, Cubism, feeding/growth, live expressions, gaze, chat or pixel-shape click-through completion. It neither downloads replacements nor accepts new licenses.

## Verification

One offline fixture checks opt-in persistence, legacy config preservation, stale-write refusal, local v2 atlas discovery, path/symlink confinement and resource routing. A focused framework fixture checks native dragging while rejecting native/application calls and events. Windows compilation is performed by the integration owner. Interactive Windows acceptance remains required: off → on → select skin → drag → restart → off, main-window quit, offline settings entry, missing skin fallback, transparency and multi-monitor restore. No real user pet directories or Host were used during verification.
