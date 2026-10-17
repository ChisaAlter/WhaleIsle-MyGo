# CH12 OpenUI: isolated-window integration stage

This stage implements a usable `openui` / `openui-lang` assistant fence entry,
not full original inline/streaming parity. Original OpenUiFrame.tsx receives a
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

- Completed assistant fences expose Open interactive UI and Source controls.
  Streaming blocks wait until completion. Windows/macOS use a real standalone
  isolated helper window. Linux is explicitly unavailable.
- Instance identity hashes Host history binding, session, message and block
  source offset. State is JSON-object-only, max48KiB, stored under the OS user
  config directory's WhaleIsle-MyGo/openui-state-v1 using private files. At most
  256 state records and four live windows are allowed.
- Navigation, Host epoch/follow generation changes, message removal or program
  replacement revoke the captured owner and close the old renderer. Reopening
  the same completed block restores its state.
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

One focused Go invocation passed protocol/boundary/resource/state fixtures,
owner revocation/queue-target fixtures and native fence-stage UI checks. Tests
used synthetic source/ASAR/JSON only and did not execute the original viewer.
The helper transport has its own targeted checks and platform compile report.
True SDK builds and real Windows/macOS renderer acceptance are distinct evidence.

Still partial: automatic inline rendering, same-instance streaming updates,
complete original network/map/clipboard/link capabilities, and gesture-attested
immediate action sending. The default distribution lacks a bundled runtime.
These limitations must not be reported as full CH12 completion.
