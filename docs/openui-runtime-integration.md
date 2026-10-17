# CH12 OpenUI: inline and streaming integration

This stage implements visible inline `openui` / `openui-lang` assistant fences
and bounded same-instance streaming updates, with a separate-window fallback. Original OpenUiFrame.tsx receives a
pinned React viewer through an opaque iframe and exchanges channel/version/
instanceId messages. MyGo now uses that real protocol inside its existing
separately contained preview helper, with a dedicated bounded event channel.
No runtime bundle, original installer or third-party API handler is distributed
by this change.

## Supply an existing runtime

Set `WHALEISLE_OPENUI_VIEWER` to an absolute path to an already-owned viewer.html
or app.asar. With no override the application checks standard existing Whale
Isle installations: macOS /Applications or ~/Applications; Windows LocalAppData
Programs or ProgramFiles. It does not install, download, execute the original
application, scan arbitrary disks or create browser permissions.

Only the audited viewer SHA256 is accepted:
`ee5470ac1f67c50e69e554f6be5d7202625eef572b1f272f2276a1e250876961`.
It originates from ChisaAlter/WhaleIsle head
a8c931da37376ce7dc7989e6f30b0db885e06d26, successful existing Actions run
38927591701 / artifact12059037786. ASAR is read as bounded archive data only.
Other versions fail visibly. Missing resources show a real error and leave the
source accessible; no simulated renderer or empty success is shown.

The original packaged UPSTREAM.md states that the pinned open-intelligent-ui
source lacks a declared license; MIT notices cover its three SDK dependencies,
not necessarily the complete bundle. This is why the runtime is not committed
or included in the publication manifest. Using an existing local runtime does
not represent a new claim about upstream redistribution rights.

## Behavior and ownership

- Visible assistant fences automatically start a real isolated inline renderer
  when the main native parent handle is available. Windows uses the existing
  clipped child surface; macOS uses the existing pixels/height/input bridge.
  Completed fences may also be opened in a separate window after closing the
  inline instance. Retry and Source remain available. Linux is unavailable.
- Streaming program and completion changes update the same iframe/instance.
  The chat polls changes every 200ms; helper transmission is latest-value-only
  and capped at 10Hz. Only program/streaming change: current form state and
  appearance are retained. Old helpers lacking update capability fail visibly.
- Instance identity hashes Host history binding, session, message and block
  source offset. State is JSON-object-only, max48KiB, stored under the OS user
  config directory's WhaleIsle-MyGo/openui-state-v1 using private files. There is
  no fixed renderer count or persisted-record count ceiling; new states do not
  delete previous states. Per-payload limits remain: program 256KiB, state 48KiB,
  event 64KiB, assistant message 16KiB, error text 4096 bytes and resolved program
  32KiB. The existing event queue remains bounded to 16 entries.
- Navigation, Host epoch/follow generation changes, message removal or program
  replacement that removes the captured program prefix revoke the owner and
  close the renderer. Streaming append retains that owner. Reopening restores
  the last successfully received and persisted state. Pending actions are
  cleared when the displayed program changes.
- An inline renderer that has not painted for 300ms hides its Windows surface.
  After 30 seconds offscreen it closes and becomes eligible to restart when
  visible. Explicit Close cancels without automatic restart; Retry is available.
  Active instances have no arbitrary lifetime deadline. Owner revocation,
  explicit Close and offscreen reclamation still cancel them. These checks are timer-driven,
  not an assertion of immediate unmount cleanup. Queued state not delivered
  before close is not guaranteed to persist.
- The iframe cannot access Host credentials, native bindings, network, workers,
  filesystem, external navigation, clipboard or nested frames. Three typed
  resource requests run in the Go parent with a separate credential-free HTTP
  client: Wikipedia summary/media, Wikipedia/Commons image search, and OSRM
  driving geometry. These mirror Electron shell adapters, not Host API routes.
  Only title/q/coords query values are accepted; addresses and request headers
  are parent-owned. Redirects are not followed and no Host cookie/token is used.
  Requests inherit the instance cancellation context; owner is rechecked before
  a request and before its reply. Replies must match the pending instance/request
  ID; duplicate or unknown replies cannot enter another viewer.
- Resource replies use the existing helper large-body budget of 25MiB; input
  events and state retain their existing smaller limits. Oversized results
  explicitly return an error; no fabricated coordinates or placeholder photos.
  The public adapter preserves the original 8s Wikipedia/Commons and 10s OSRM
  request timeouts. Upstream JSON reads reuse the existing 12MiB resource byte
  ceiling. Original shell response caching is not yet mirrored.
- Wiki photos and image-search display arrays now contain data URLs fetched
  solely from the fixed HTTPS upload.wikimedia.org origin. Only those fields
  returned by the selected public adapter are eligible; iframe-supplied URLs,
  credentials, redirects and other origins cannot use this downloader. PNG,
  JPEG, GIF and WebP are detected from their bytes. Unsupported or oversized
  data returns an explicit resource error. Original URLs remain unchanged in
  photoSources/imageSources, and any existing attribution fields are retained.
  The original adapters provide no author/license metadata; none is invented.
  The iframe retains img-src data: and has no direct network access.
- Maps still need MapLibre workers, OpenFreeMap glyphs/vector tiles and Esri
  raster fallback; those remain blocked by the unchanged iframe worker/connect
  policy. No map data is simulated.
- Validated assistant actions appear in native UI with their form/context text.
  Send interaction result submits through the captured ComposerTarget in queue
  mode. It never selects a session from renderer-supplied fields. An explicit
  native confirmation remains because postMessage alone cannot attest whether
  an action came from a genuine user gesture rather than programmatic code.
- Helper events are capped at64KiB,16 queued entries and checked twice: trusted
  parent shell validates iframe source/opaque origin/protocol/instance, and Go
  validates envelope/type/size. State messages coalesce. No arbitrary RPC or Eval
  API is added to the renderer. Old helpers must reject missing OpenUI capability.

## Validation and remaining work

Stage one passed synthetic protocol/resource/state/owner/UI fixtures and actual
Windows/macOS SDK compilation in Actions run 39023193323. Stage two adds a Node
fake-DOM fixture executing only our parent-shell script: same iframe identity,
new program/streaming, retained typed state, and foreign-instance rejection.
Native fence/source checks and the affected frame/main packages compile in the
focused Go check. Native helper transport/update tests and platform cross-builds
are reported separately. No original viewer, real account or microphone is
executed by these checks.

Source and synthetic fixtures cover streaming/state preservation and bounded
transport. Automatic visible loading, offscreen timeout, clipping, user input,
scrolling and close/reopen behavior still require real Windows/macOS runtime
acceptance. No screenshot or pixel acceptance is claimed. Stage two native SDK
compilation passed in Actions run 39023879575 at commit
551649d8afe62ff3964dc94e1e0f610faacea2d8; stage one run 39023193323 also passed.
These compile results do not establish signed-build or real UI acceptance.
A focused lifecycle fixture additionally checks that active contexts have no
deadline and explicit cancellation still terminates them; the existing owner
ticket fixture covers owner revocation.

Still partial: map workers/tiles, clipboard/link capabilities and
gesture-attested immediate action sending. The default distribution lacks a
bundled runtime. These limitations must not be reported as full CH12 completion.

Count regression fixtures save and read the 257th state while retaining the first,
and keep five independent cancellable contexts alive. Source removes the global
renderer semaphore. This is not a five-native-window physical acceptance test.

The typed resource adapter fixture uses an in-memory HTTP RoundTripper only.
It checks all three real response schemas, fixed destinations, no authentication
headers, malformed route rejection, cancellation and pending-instance replies.
No public service, real account query or location is contacted by these checks.
Native API-response command compilation is tracked separately from the earlier
successful streaming-stage SDK runs.

The image stage uses a local one-pixel PNG fixture. It checks real data URL
bytes, retained source/attribution data, exact Wikimedia origin restrictions,
and API responses above 64KiB but below the existing 25MiB body budget. No real
image server is contacted. Actual viewer display remains a runtime acceptance
item. This stage needs its own matching native response-budget compilation.

## Map resource boundary (source-only assessment)

The verified viewer embeds its MapLibre module worker as a `data:` URL
(original src/openui-viewer/index.tsx:15–18). Its README explicitly explains
that an opaque iframe cannot load Chromium `blob:null` module workers. Required
public map resources are OpenFreeMap at https://tiles.openfreemap.org (planet,
fonts and the TileJSON's same-origin tiles), plus the two ArcGIS raster fallback
prefixes under https://server.arcgisonline.com/ArcGIS/rest/services/:
World_Street_Map/MapServer/tile/ and
Canvas/World_Dark_Gray_Base/MapServer/tile/. No remote script is necessary for
the already bundled worker.

Merely changing this shell's CSP cannot enable these resources today:

- Windows creates a zero-capability AppContainer in
  previewprocess/launcher_windows.go. token_windows.go rejects capabilities in
  the helper/browser process tree, while MyGo's Windows preview_isolation.go
  separately denies all external resource requests, including worker traffic.
- macOS preview.entitlements grants only App Sandbox. identity_darwin.go and
  native/macos-preview/main.m require exactly that entitlement set, and the
  native resource delegate also rejects external traffic. A signed entitlement
  cannot change with a document flag.

A future working path needs a distinct verified-OpenUI Windows process policy
with fixed-resource filtering, and a separately signed macOS OpenUI helper
bundle/identity with matching entitlement validation and fixed-resource
filtering. A mere OpenUI boolean must not grant these permissions to arbitrary
HTML: both sides must verify the pinned runtime. Ordinary HTML must retain its
existing zero-network/zero-worker policy. No such new mode, entitlement or CSP
permission was added in this assessment.

This assessment read source only. No test source was added, and no tests,
builds, helper launches, permission changes or real map requests were performed.
The recovered upstream MapLibre viewer remains unsupported under the present helper policy.

An independently authored raster Map is being integrated at source level with
the owned SDK-based viewer. Its parent resource adapter uses the fixed
`/api/map-tile` operation (`z`, `x`, `y`, `style=light|dark`) and returns
bitmap data URLs from the two existing ArcGIS raster services. This path does
not require iframe network access or a worker. Component integration, production
asset publication and native/runtime verification are still pending; it is not
recorded as completed parity.
