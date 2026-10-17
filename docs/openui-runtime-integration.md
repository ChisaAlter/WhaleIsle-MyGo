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
  filesystem, external navigation, clipboard or nested frames. Original
  external wiki/image-search/routing adapters return explicit unavailable errors.
  Map/network-dependent components therefore remain unsupported.
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

Still partial: complete original network/map/clipboard/link capabilities and
gesture-attested immediate action sending. The default distribution lacks a
bundled runtime. These limitations must not be reported as full CH12 completion.

Count regression fixtures save and read the 257th state while retaining the first,
and keep five independent cancellable contexts alive. Source removes the global
renderer semaphore. This is not a five-native-window physical acceptance test.
