# Native desktop companion core

The MyGo shell now owns a small independent transparent, frameless, topmost sprite window. Settings → Pet provides an explicit enable switch, local skin selection, refresh and save. It works offline, without a Harness connection. Default is off; no scans, timer or pet window are started while off. Refresh scans only when explicitly requested, or when opening an enabled companion. Position restoration and disconnected-screen clamping use MyGo's existing `StateKey` window-state implementation. Drag the character to move the window; use Settings to hide it. Closing the app destroys the companion without changing its enabled preference.

Configuration is stored under `mygoDesktopPet` in the existing conflict-checked config store. Electron `pet` and `live2dPet` enabled state, preferences and unknown fields are preserved. Explicit care/token-feeding actions update the shared `live2dPet.stats` and `live2dPet.growth` fields described below. A saved unavailable skin is preserved and falls back visually to the bundled Whale image. Select the empty entry for that image. Local Codex packages are read from `${CODEX_HOME:-~/.codex}/pets`, using directory names as stable IDs. Only bounded `pet.json` and confined 1536×1872 / 1536×2288 WebP sheets are read; scripts are never executed, symlink escapes and traversal are rejected, and nothing is uploaded or downloaded.

The page is served from a fixed `whale-pet://localhost` resource allowlist. Page navigation and new windows are blocked; four exact local navigation intents (open the displayed current session, open the dedicated pet conversation, choose a file meal, hide this instance) are intercepted without navigating. The new opt-in MyGo `PageOptions.WindowControlsOnly` mode only permits DOM readiness and native CSS dragging; application methods, broadcasts, file drops, maximize and other bridge messages are blocked. `DisableBindings` retains its original behavior. This is a small rectangular window, not an all-screen click-through overlay: its rectangle can intercept clicks. Fine-grained shaped hit regions and nonfocusable native windows are not claimed here.

## Source and licensing

`internal/desktop/pet/assets/pet-head.png` is copied unchanged from the supplied WhaleIsle-Electron `assets/pet-head.png` snapshot. Its original repository MIT license, copyright 2026 Chisa, is retained beside the asset. This is the existing fallback badge, not newly generated animation art. The sprite atlas dimensions/idle-row convention follow Electron `src/main/desktop-pets.js` and `src/renderer/pet.js`; the image-only renderer is a minimal new adapter. User-selected packages retain their own original licenses and remain in their directories.

## Renderer parity boundary

Electron's feature named `desktop-live2d-pet` uses THA4/ONNX neural animation, not Cubism. The optional `mygoDesktopPet.neural` setting now connects the fixed 512px THA4 model with WebNN → WebGPU → WASM provider fallback. Both the companion and neural rendering remain off by default. The adapter is based on official WhaleIsle commit `90465b67ec16765631d54c195af19d9ff3c3a770`, including premultiplied image input, reusable 45-channel pose tensor, byte-scale CHW output, no concurrent inference/backlog, and deduplicated output disposal in `finally`. It uses the original crop and 50ms inference-start budget, slowing to 110ms after five minutes without pointer interaction. Hidden documents pause inference; close/unload releases the session after any in-flight run, and native destruction ends the webview.

This is a bounded small-window implementation, not full Electron rendering/performance parity. Graph capture, GPU packed readback, Anime4K super-resolution, rig/state sprites, full-screen wandering, click-through regions, floating quick-chat input and broad Electron APIs are not ported. The dedicated conversation entry is described below. GPU uses default CPU readback; browser/provider limitations fall back through providers. Missing/changed assets or model/runtime failures show an explicit notice and the selected local sprite or bundled image. A transparent first frame is a failure, never a success notice.

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
the floating overlay input/model picker remains out of scope. Explicit local token feeding is described below.

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
behavior. Token feeding uses the separate real settlement ledger below; its food
is derived from settled usage rather than simulated.

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


## Local token-food ledger

The companion exposes **刷新粮碗** and **喂 Token**, porting the real
`src/main/pet-growth.js` and `pet-stats.js` contracts. These are local game
points, not priced usage, currency, prepaid API credit, or account balances.
No model, payment, billing API, network request or extra plugin is invoked.
Refresh is explicit; status polling never scans logs or writes preferences.
Feeding consumes only already-recorded local food, so a scan error does not
prevent using an existing bowl. Refresh first to include newly-used tokens.

The runtime uses the existing configured/PATH Node resolver and an embedded,
builtin-only scanner. The sole path comes from native `paths.DshHome()/sessions`
(the same authorized application data-home as the Host), never the page. It
never reads other applications' histories. Session bodies are decompressed in
a child process; only usage counts and SHA256 settlement identities are returned,
with no message text, filenames, paths, credentials or session titles persisted
or exposed. No NODE_OPTIONS/NODE_PATH loader or shell command is accepted.
Node with builtin zstd support is required, as for the existing Host runtime.

The original zstd concatenated-frame format, assistant message/attempt/chunk
last-wins settlements, separate compaction usage, cross-version log dedup and
SHA256(JSON.stringify([session basename, "turn:step"])) identities are retained.
Input + cache-read + cache-write + output count once; reasoning is already in
output. The original `food.version=1`, `food.seen` high-water marks and available
balance remain compatible, without a second source/grain ledger. Moving a
workspace, deleting logs, restoring logs or reopening the app cannot re-credit
an observed settlement. Decreased usage does not revoke earned food. Missing
high-water entries remain saved. The first successful scan seeds the baseline
without awarding the historical corpus; legacy unspent food and lifetime points
survive migration. Today uses local calendar days and records observed deltas,
not an estimate of historical daily spend.

A click consumes at most the next original level gap (100M first promotion,
20B final tier), one local food token per growth point. A positive meal also
applies the original feedToken care: +45 satiety, +12 mood, +5 affection, or +1
affection within ten minutes of the previous token meal. Empty bowls grant
nothing. Points, food, lifetime feeding and care are merged atomically into the
existing config.json with optimistic conflict checking. Save failure changes
neither disk nor memory. Other-shell settings, enable flags and unknown fields
remain intact. Invalid/unsupported persisted ledgers are refused instead of
silently reinitialized and potentially awarding old usage again.

The scan has a 30-second window-owned deadline (the process itself also has a
60-second cap); close/replacement cancels it. A Host generation, data-home or
runtime change discards its result before persistence. Symlinks/junctions below
the authorized root are not followed. Limits are 20,000 log files, 50,000
directories, 128 MiB compressed per file, 256 MiB per decoded frame, 512 MiB total
decoded corpus, 200,000 settlements and 32 MiB process output. Unreadable,
truncated, oversized or invalid data aborts the refresh without a partial
baseline/award; retry after an active write finishes. There is no extrapolation
or fallback to incomplete rankings. Windows native rendering remains unverified
by the targeted synthetic fixture and compile alone.

## Explicit head-pat care

The supplied `pet-live2d.js` really exposes `pat` in `PANEL_ACTIONS` (1588–1595),
dispatches it through `runAction` (4306–4312), and persists `care('pat')` via
`desktop-live2d.js`. The native **摸摸头 / Pat** entry now performs that actual
care action: +10 mood (capped at 100), +4 affection, reduced to +1 affection
when another pat happened within the previous ten minutes. Cooldowns are keyed
by action, so file/token meals do not suppress or reset the pat cooldown.
Existing elapsed hunger/mood decay is folded before the award.

The fixed native `pat-pet` navigation is the only new action. No query-supplied
care kind or application bridge is exposed; fetching its resource route cannot
mutate state. A closed/disabled pet or active native file picker refuses it.
It uses the same atomic, conflict-checked stats writer as file care and the
same original affection-cooldown calculation as token feeding. Only stats and
`care.pat` change; local food, points, native preferences, other care timestamps
and unknown fields are preserved. Failed saves show failure and grant nothing.
The existing care status reflects the saved mood/affection immediately.

This deliberately does not restore the removed play/tease menu entries or
invent a persistent rest reward. Original sleep is a renderer-only temporary
state; explicit waking has a separate -7 mood action. Sleep/wake animation and
its temporary state are not part of this small persistent-pat increment.
