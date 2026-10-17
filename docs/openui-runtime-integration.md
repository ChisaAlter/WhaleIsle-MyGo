# CH12 OpenUI: inline and streaming integration

This stage implements visible inline `openui` / `openui-lang` assistant fences
and bounded same-instance streaming updates, with a separate-window fallback. Original OpenUiFrame.tsx receives a
pinned React viewer through an opaque iframe and exchanges channel/version/
instanceId messages. MyGo now uses that real protocol inside its existing
separately contained preview helper, with a dedicated bounded event channel.
The default viewer is independently authored application code using explicitly
licensed OpenUI SDK packages. Its maintainable sources and pinned dependency
lock live in `tools/openui-runtime/`; production HTML and third-party notices
live in `internal/desktop/openui/viewer/` and are embedded in the Go application.
An original Whale Isle installation is not required or probed on default startup.

## Built-in runtime and explicit legacy import

Build the viewer using the production build instructions in
`tools/openui-runtime/README.md`. The checked-in production artifact includes
its dependency notices; no network is required to load its scripts, styles or
fonts. The application uses the same opaque iframe and native preview bridge.
Source integration is not Windows/macOS runtime acceptance.

An explicit `WHALEISLE_OPENUI_VIEWER` absolute path remains a compatibility
option for an already-owned original viewer.html or app.asar, not a default
installation prerequisite. That legacy path accepts only SHA256
`ee5470ac1f67c50e69e554f6be5d7202625eef572b1f272f2276a1e250876961`.
It reads the archive as data and does not execute the original application.
The upstream application bundle is not redistributed; its unclear license
status is separate from the independently authored built-in implementation.

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
  filesystem, direct external navigation or direct clipboard APIs. The owned
  runtime may request typed clipboard/HTTP(S) link actions through the current
  Host after a real UI gesture; legacy runtime overrides do not receive this
  capability. Local nested HtmlView documents
  require explicit native capability and actual frame-identity admission; arbitrary
  nested navigation remains unavailable. Fixed typed
  resource requests run in the Go parent with a separate credential-free HTTP
  client: Wikipedia summary/media, Wikipedia/Commons image search, and OSRM
  driving geometry, plus raster map tiles for the owned Map component. The first
  three mirror Electron shell adapters; none are Host API routes.
  Only title/q/coords or tile z/x/y/style values are accepted; addresses and request headers
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

Current remaining scope: external CDN scripts, external
Image/Markdown image URLs and gesture-attested immediate assistant sending.
The default distribution now includes the independently authored runtime,
owned raster map tiles and typed clipboard/link actions. This does not establish
full component-variant, platform-runtime or visual parity. Prior stage compile
results above do not cover the newly embedded runtime, HtmlView or Host actions.

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

An independently authored raster Map is integrated at source level with
the owned SDK-based viewer. Its parent resource adapter uses the fixed
`/api/map-tile` operation (`z`, `x`, `y`, `style=light|dark`) and returns
bitmap data URLs from the two existing ArcGIS raster services. This path does
not require iframe network access or a worker. The checked-in owned viewer includes this component and its resource bridge.
Native compilation/runtime verification is pending; source implementation and
production asset generation do not establish completed runtime parity.

## Public adapter process cache

The parent independently implements successful-response reuse matching the
original adapters' observable cache scope: 500 Wikipedia titles, 500 normalized
image queries, and 200 normalized route coordinate sets. Entries live for the
application process lifetime and evict oldest insertion first. Reads do not
change eviction order. Failures are not cached.

Cache entries contain only public response JSON. A cache hit receives the
current instance/request IDs; no prior owner envelope is reused. Image source
JSON is cached before data-URL expansion, so large bitmap payloads are not
retained in this process cache. This does not claim the browser HTTP image-cache
semantics of the original Electron implementation or any runtime verification.

## Independently authored HtmlView

The built-in registry provides `HtmlView(html, height?, title?)` for local
HTML/CSS/inline JavaScript/Canvas. The content lives in a separate opaque
`sandbox=allow-scripts` srcdoc. Fixed height or bounded automatic content height
is supported; changed content/streaming/removal destroys the old document.
CDN scripts, network and workers remain disabled inside HtmlView. The separate
YouTube component uses the private native media route described below.

The trusted helper reports a default-disabled HtmlView capability to the shell.
macOS tracks the admitted viewer identity and its immediate srcdoc child.
Windows uses actual Frame2/Frame7 identities and direct-child events. Missing
Frame7 support leaves the base viewer working and reports HtmlView unavailable.
Ordinary HTML behavior, OS AppContainer/App Sandbox settings and native
credential/binding restrictions are unchanged.

Only source integration and production asset generation are complete. Native
compilation, frame lifecycle/event timing, inline script execution, Canvas,
height and input behavior have not been verified by running either platform.
See `openui-htmlview-windows-source-20261017.md` for exact API requirements and
remaining source-level assumptions.

## Typed clipboard and link actions

Only the default owned embedded viewer can receive `hostActions`; an explicit
legacy runtime override remains disabled. Code copying, Link, Markdown/Text
HTTP(S) links and button action plans use the existing bounded event/response
transport. They cannot claim success before Clipboard.Write/OpenExternal
returns from the MyGo UI-thread operation. A successful URL reply means the OS
browser handoff returned successfully, not that a remote page loaded.

The request is bound to its current instance/session/owner and one captured
payload. Source replacement, component removal or lost capability cancels
remaining work; actual completed system effects cannot be undone by removing
the component. No clipboard contents or URLs were executed during development.
See `tools/openui-runtime/NATIVE-ACTIONS.md` and
`openui-native-actions-host-20261017.md` for source behavior and limits.

## YouTube private native media route

`YouTube(video, title?)` now has source integration with a native browser seat
separate from the isolated OpenUI helper. Only a fixed parsed video ID crosses
the typed media bridge; no arbitrary URL, HTML, headers or app cookies are
accepted from the renderer. The first actual activation creates a fresh
TransientProfile/Untrusted/DisableBindings host. Existing browser/application
cookies are not shared; YouTube may set its own cookies in that temporary store.

Host-assigned source revision and paint-owned viewport/clip bind the seat to
the current component. Old queued revisions cannot recreate removed seats.
Hidden, removed or expired seats are destroyed; returning creates a new paused
player and does not preserve playback position or temporary cookies. There is
no arbitrary concurrent-player count cap.

The identified player uses a fixed, script-free local iframe wrapper with the
current shipped `ai.deepseek.harness.gui` HTTPS baseURL, via existing LoadHTML.
This follows the desktop application identity path in the
[YouTube embedded-player requirements](https://developers.google.com/youtube/terms/required-minimum-functionality#api-client-identity-and-credentials).
The ordinary browser constructor is unchanged. Host-triggered reload reinstalls
the identity document; engine-internal refresh behavior remains unverified.

`native-view-created` acknowledges a native seat, not playback. No video request,
native build or playback test was performed. Actual Referer delivery, service
acceptance, codec/input/clip behavior and remote-client presentation remain
unverified. See `tools/openui-runtime/YOUTUBE-MEDIA.md` for the wire contract.
