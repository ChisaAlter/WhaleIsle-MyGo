# Native desktop companion core

The MyGo shell now owns a small independent transparent, frameless, topmost sprite window. Settings → Pet provides an explicit enable switch, local skin selection, refresh and save. It works offline, without a Harness connection. Default is off; no scans, timer or pet window are started while off. Refresh scans only when explicitly requested, or when opening an enabled companion. Position restoration and disconnected-screen clamping use MyGo's existing `StateKey` window-state implementation. Drag the character to move the window; use Settings to hide it. Closing the app destroys the companion without changing its enabled preference.

Configuration is stored under `mygoDesktopPet` in the existing conflict-checked config store. Electron `pet` and `live2dPet` enabled state, growth, preferences and unknown fields are not migrated or rewritten. Explicit file feeding updates only the shared `live2dPet.stats` care fields described below. A saved unavailable skin is preserved and falls back visually to the bundled Whale image. Select the empty entry for that image. Local Codex packages are read from `${CODEX_HOME:-~/.codex}/pets`, using directory names as stable IDs. Only bounded `pet.json` and confined 1536×1872 / 1536×2288 WebP sheets are read; scripts are never executed, symlink escapes and traversal are rejected, and nothing is uploaded or downloaded.

The page is served from a fixed `whale-pet://localhost` resource allowlist. Page navigation and new windows are blocked; four exact local navigation intents (open the displayed current session, open the dedicated pet conversation, choose a file meal, hide this instance) are intercepted without navigating. The new opt-in MyGo `PageOptions.WindowControlsOnly` mode only permits DOM readiness and native CSS dragging; application methods, broadcasts, file drops, maximize and other bridge messages are blocked. `DisableBindings` retains its original behavior. This is a small rectangular window, not an all-screen click-through overlay: its rectangle can intercept clicks. Fine-grained shaped hit regions and nonfocusable native windows are not claimed here.

## Source and licensing

`internal/desktop/pet/assets/pet-head.png` is copied unchanged from the supplied WhaleIsle-Electron `assets/pet-head.png` snapshot. Its original repository MIT license, copyright 2026 Chisa, is retained beside the asset. This is the existing fallback badge, not newly generated animation art. The sprite atlas dimensions/idle-row convention follow Electron `src/main/desktop-pets.js` and `src/renderer/pet.js`; the image-only renderer is a minimal new adapter. User-selected packages retain their own original licenses and remain in their directories.

## Renderer parity boundary

Electron's feature named `desktop-live2d-pet` uses THA4/ONNX neural animation, not Cubism. The optional `mygoDesktopPet.neural` setting now connects the fixed 512px THA4 model with WebNN → WebGPU → WASM provider fallback. Both the companion and neural rendering remain off by default. The adapter is based on official WhaleIsle commit `90465b67ec16765631d54c195af19d9ff3c3a770`, including premultiplied image input, reusable 45-channel pose tensor, byte-scale CHW output, no concurrent inference/backlog, and deduplicated output disposal in `finally`. It uses the original crop and 50ms inference-start budget, slowing to 110ms after five minutes without pointer interaction. Hidden documents pause inference; close/unload releases the session after any in-flight run, and native destruction ends the webview.

This is a bounded small-window implementation, not full Electron rendering/performance parity. Graph capture, GPU packed readback, Anime4K super-resolution, rig/state sprites, full-screen wandering, click-through regions, growth, floating quick-chat input and broad Electron APIs are not ported. The dedicated conversation entry is described below. GPU uses default CPU readback; browser/provider limitations fall back through providers. Missing/changed assets or model/runtime failures show an explicit notice and the selected local sprite or bundled image. A transparent first frame is a failure, never a success notice.

### Fixed assets and staging

`internal/desktop/pet/assets/render-manifest.json` pins exact source URLs, sizes and SHA256 values for seven runtime/model files plus both MIT licenses. Assets are restored separately from that fixed official commit; binary assets are not in Git. The executable accepts only `pet-render-assets/` beside itself. On opt-in, `os.Root` confines reads, bounded sizes and SHA256 are checked, and only those seven verified byte buffers are published through exact resource routes. A sidecar manifest cannot add paths. No network download, file selection API or arbitrary asset path is exposed. Closing drops these buffers.

After packaging (which recreates `dist/whaleisle`), stage the restored fixed source directory:

    node scripts/pet-render-assets.mjs <restored-asset-directory> dist/whaleisle

For a development executable, replace `dist/whaleisle` with the directory containing that executable. The staging command verifies every file before writing, copies licenses, and emits the provenance manifest. It never executes the model. An ordinary `go run` uses a temporary executable path and therefore intentionally reports missing assets unless staged there. Installer builds include only the manifest-listed files when the staging marker exists.

Windows uses MyGo's existing `whale-pet://localhost` mapping to `http://whale-pet.localhost`; relative JS/model/module URLs retain that exact origin. The old `//pet` host did not match Windows' protocol mapper and has been corrected without changing the framework. The pet-only CSP permits same-origin scripts/fetch and WebAssembly compilation, forbids workers, remote connections and frames, and no longer permits inline scripts. `WindowControlsOnly` and the exact open/hide navigation whitelist are retained.

Source licensing: upstream root MIT (Chisa 2026), ORT 1.22.0 Microsoft MIT; both files are staged. No separate avatar model card, training-data provenance or art-rights chain is established here.

## Verification

One offline fixture checks opt-in persistence, legacy config preservation, stale-write refusal, local v2 atlas discovery, path/symlink confinement and resource routing. A focused framework fixture checks native dragging while rejecting native/application calls and events. Windows compilation is performed by the integration owner. Interactive Windows acceptance remains required: off → on → select skin → drag → restart → off, main-window quit, offline settings entry, missing skin fallback, transparency and multi-monitor restore. No real user pet directories or Host were used during verification.


## Current-session task status

The companion shows the currently selected session from MyGo's existing session store. A live Harness and mux plus an available roster row yield Running or Idle (not a completion/success claim). Loading, failed roster, or no matching selected row yields Unknown; a missing connection yields Offline. No job success, failure, queue, permission-wait, or cross-workspace total is inferred from Running=false. This intentionally covers less than Electron's `src/main/pet-dsh-watch.js` log watcher and does not scan session logs or monitor all workspaces. Only the session title, ID and this bounded status reach the local page; message content and tool arguments do not.

Open chat validates the exact displayed session ID again against the live current selection, then reveals/focuses the main window and switches to that chat. Deleted/stale targets cannot redirect to another chat. Hide destroys only the companion instance, leaving the persisted opt-in unchanged; Settings → Pet → Save restores it. OS close has the same session-only behavior. Main app exit destroys it. Store/mux refresh callbacks update the pet independently of main-window painting, including when the main window is hidden.

The local HTML uses escaped text and URL components, no application bridge and no external content. A fixed, same-origin task.json endpoint refreshes only the current-chat card every 1.5 seconds; changes do not reload the model. The retained WindowControlsOnly bridge whitelist is unchanged; action URLs are accepted only in the navigation handler, not through the resource handler.

One additional offline fixture covers running/idle/unknown/offline, missing and failed source, HTML escaping, exact open-session targeting, foreign origins/extra parameters and hide. No real Host or user data was used. Interactive Windows checks remain: run/idle and disconnect transitions while main window is hidden, click to reveal the matching chat from settings, stale/deleted target rejection, hide → state change stays hidden → settings Save restores, and OS close/quit. Cumulative Windows compilation belongs to the integrator.

## THA4 verification boundary

One synthetic JS fixture passed for image normalization, named inputs/shapes, byte-scale CHW conversion, malformed output, duplicate-output disposal, single in-flight run and close-during-run/input/session cleanup. A single focused Go invocation passed the existing core/status fixtures and new resource-boundary fixture: missing resource rejection, exact routing, query rejection and closing removal. JS syntax checks passed. The offline staging command also verified and copied all nine pinned resources successfully without executing them. No real ORT runtime/model, user skin, user Host or Windows GUI was run. Cumulative Windows compilation belongs to the integration owner; interactive opt-in/load/animate/provider-fallback/hide/quit and actual performance remain unverified.

## Dedicated pet conversation entry

The companion now has a separate **和宠物聊聊 / Pet chat** action. It implements
Electron's real `pet-live2d.js` ↗ path (`openWhale` → `__dshWhaleOpen` →
`dsh-whale/assistant/ensure` → open the returned session), rather than relabeling
the existing current-task link. Opening sends no prompt and performs no model
inference. The ordinary native session composer and history remain the chat UI;
the floating overlay input/model picker and feeding/growth are still out of scope.

The existing Whale plugin owns the stable assistant identity, preset, name and
model settings under `${DSH_HOME}/data/whale/settings.json`. The client does not
create a parallel identity store or migrate Electron pet state. Each explicit
open asks the same plugin to ensure/reuse that session; a missing session is
repaired by its existing contract. `whaleAssistantEnabled` must already be true
in the existing configuration, and the authenticated Host/mux must be ready.
Missing/disabled plugins and failed RPCs show an error on the pet, with no
fallback into a task conversation and no automatic plugin enabling/restart.

Only the exact local `whale-pet://localhost/chat` navigation intent can invoke
this action; resource requests cannot. The page receives only busy/message
status, never Host cookies or the returned assistant identity. Duplicate clicks
share one in-flight operation. Hide, OS close, settings recreation and quit
cancel it and invalidate its result. A changed Host generation/origin/cookie,
store, plugin enable flag or current chat selection also prevents late results
from changing the main conversation. The single fixed authenticated ensure RPC
refuses all redirects. No new application binding, file API or remote asset
access is exposed. Saved window position is retained; the fixed card height is
updated so the added action and feedback remain reachable after an old restore.

One temporary synthetic fixture passed for the separate action, rejection of
forged/query URLs, request deduplication, exact RPC/credential header contract,
stable returned identity, unavailable/refused plugin, redirect rejection,
close/cancellation and stale completion, escaped feedback, endpoint separation
and unchanged client configuration. JS syntax and patch whitespace checks
passed. The integration owner performs cumulative Windows compilation. Actual
Windows clicks/focus/layout and a live installed Whale plugin remain unverified;
no real Host, user account, model or pet directory was used.


## File feeding and persisted hunger

The supplied original has a real renderer drop action: `pet-live2d.js:4205–4219`
invokes `care('fileEat')`; `desktop-live2d.js:476–491` persists it through
`pet-stats.js`. The native companion now exposes **喂文件 / Feed file**, an
explicit native single-file picker adapting that same action. A successful
selection grants +2 satiety and +1 mood, clamped to 100, with no affection award.
Cancel does nothing. The selected path/name is not read, saved, displayed,
uploaded, moved or deleted. Only a deliberate choice counts, and repeated
clicks cannot open concurrent pickers; closing/replacing the pet invalidates
an outstanding selection.

The separate original `fileEat(names)` metadata-history endpoint has a daily
cap, but this renderer does **not** call it. This implementation follows the
actually wired care action and does not add that disconnected history/quota
behavior. Token/compute feeding still requires its real usage ledger and is not
simulated or claimed here.

Care uses the same `config.json` and `live2dPet.stats` object as Electron:
`satiety`, `mood`, `affection`, `lastTick` and `care.fileEat`. Unknown nested fields,
other care timestamps, growth, `fileEaten`, preferences and enable flags survive.
Existing atomic/conflict-checked config writes commit all changed stats together;
a failed save grants nothing and reports failure. Ordinary MyGo pet preference
saves retain these values. Missing stats use the original 70/70/0 defaults and
are first persisted by the explicit feeding action, not by status polling.

The existing pet status endpoint shows current satiety/mood. Reads lazily fold
elapsed time: satiety −3/hour; mood approaches 62 at 8/hour; elapsed drift is
capped at 72 hours, and affection never decays. Persisted timestamps retain
hunger across app restarts without new timers or renderer animations.

One temporary fixture passed: defaults/normalization, elapsed hunger/mood,
actual care increments, repeated explicit meals, restart preservation, unknown
nested settings, later native preference saves, read-only polling, stale-write
refusal without a reward, exact action URL, escaped UI and resource/action
separation. No real files were selected or read; no models, accounts or Host
were used. The integration owner alone performs Windows compilation. Actual
native picker/close flow and Windows rendering remain unverified.
