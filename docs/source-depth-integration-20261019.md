# Native source integration, 2026-10-19

This records the source-first continuation after `ffe7ab6`. The current Windows
checkpoint distinguishes source, ordinary regression, real Host and native
acceptance. Earlier checkpoints and all failed receipts remain historical.

## Current Windows checkpoint

Product/test revision: `aa159b6aade0fa108cfd699cce81338a80a3cf3e` (tree
`a1216fe1a14908e5a777f01577f8de72466acd0d`). The final ledger commit changes
documentation only. Branch: `codex/session-runtime-compat-20261019` in
`C:\Users\48818\Documents\Codex\2026-10-19\task-2\WhaleIsle-MyGo`.
Reference remains the read-only original git object
`e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`.
**This is not complete original-product, visual or all-function acceptance.**

The finite continuation after `2e09236a` closed the following confirmed source
differences. Root integrated shared placement/frame contracts; domain owners
used exclusive file ranges. Implementation preceded new regression preparation.

| Scope | Implemented behavior | Source/limits |
|---|---|---|
| SVG native helper | Contained Windows temp creation uses the actual returned temp path inside owned staging. Hidden inert images complete after font/image decode without waiting for visibility-dependent animation frames. Inert parent/child policies consistently admit embedded data images/fonts. Bounded pre-document startup and fixed authenticated stage/code receipts distinguish handshake, load and metadata failures. | `8763dcf9`, `5d1c5706`, `ae16a847`, `8ff206ef`. Ordinary preview policy and external resource restrictions remain. No global network/security setting change. |
| Beam | Gaussian bloom is filtered before the outer clip; inset shadows use the two-dimensional rounded-hole complement and 4x4 boundary coverage. A single finite off-UI job and two RGBA buffers retain a completed frame and reject stale geometry/palette results. | `0dfb54ea`; [domain record](source-depth-effects-beam-20261019.md). Finite three-sigma/4x4/2048-conic rasterization and asynchronous frame latency still differ from Chromium. |
| Windows fluid | Normal worker presentation transfers a shared D3D11 texture lease to a separate UI device; keyed-mutex admission is nonblocking, and UI copies GPU-to-GPU into the ordinary image layer. No final CPU readback or CPU placeholder upload in that path. Diagnostic readback is explicit and allocates staging lazily. | `f1b43fae`, `3693fbcc`. A GPU copy and bounded hardware/resource profile remain different from direct WebGL presentation. |
| First-send placement | Original 80ms quiet/450ms cap and 300ms easing hold the actual measured card position during first-send layout, then dock it. Same-frame observe/relative-placement hooks move paint, hit, accessibility and IME geometry together without changing sibling flow. Session/Host, scroll, takeover, reduced-motion and cancellation boundaries retire the transition. | `b14353b3`, `df209db5`, `dbcb6fa5`, `cc3574e0`; current-frame and actual Store echo fixtures. No model submission. |
| Preset/compaction registered states | One-click Creator Prepare/Create and exact successful-task Open; per-guide-page scroll and modal focus return; explicit read-only configuration; receiving-session-only introduction with original timing, current theme font and original icon; automatic compaction is collapsed by default and expands full Markdown only on explicit input. | `2d464632`, `414a82c2`; [conversation record](source-coverage-conversation-20261019.md). Host owns automatic pressure/model recovery; no client reimplementation. |
| Remote registered states | Reopening every expanded directory refreshes through the captured Browse owner. Gateway labels preserve saved unavailable NIC choices and distinguish unknown from observed-empty address data; selected labels still save the raw IP. | `574252e3`, `7722ecef`; [finite six-ID record](source-depth-remote-pet-20261019.md). No real pairing/SSH/IM action. |

The SVG diagnosis retained each failed experiment rather than repeating it
unchanged. Initial route diagnostics showed handshake EOF; pre-document stderr
then identified the contained nested temp directory. Once startup succeeded,
load-stage diagnosis isolated inherited CSP admission. A child policy cannot
relax its inherited parent restriction; the correction is limited to inert
data-image/font resources. See [CSP inheritance](https://www.w3.org/TR/CSP3/#security-inherit-csp)
and [multiple policies](https://www.w3.org/TR/CSP3/#multiple-policies).
The final remaining size failure was an incorrect fixture expectation: CSS-only
SVG has browser natural size300x150 here. The fixture preserves that case, adds
fractional intrinsic dimensions that must refine to actual browser metadata,
requires `LoadStatus.Loaded`, and checks both owner revocations. It cannot pass
using only provisional geometry.

| Evidence class | Actual result and exact source |
|---|---|
| Final complete ordinary Windows regression | **PASS**, `aa159b6a`, `go test -json -count=1 -timeout=3m ./...`, Windows amd64/Go1.27.2/CGO0/offline, exit0; 115 package passes,22 package skips,4073 test/subtest passes,27 test/subtest skips,0 failures. [Receipt](../../evidence/final-svg-closed-regression-20261019/root/receipt.json), [count summary](../../evidence/final-svg-closed-regression-20261019/root/summary.json). Skips are not acceptance. |
| Matching final Windows artifacts | **PASS**, `aa159b6a`, main and helper exit0, version `0.0.0-local.aa159b6a`. [Build receipt, paths, flags and hashes](../../evidence/final-svg-closed-windows-20261019/receipt.json). Main SHA256 `1a208e9008d07105ece36dd0b1ef8e81263ddde3c45ed59614fb2a162e5973a4`; helper `64d700cb983baf83731ccdc9ef129d6e6c32a1fd8012e940ac96909053088de2`. Fresh evidence outputs only; main was not launched, installed or copied to delivery directories. |
| Final native SVG with that exact helper | **PASS**, `aa159b6a`, opt-in actual contained helper plus owned hidden native parent, exit0. CSS-only300x150 gives640x320; fractional hint80.5x160.5 refines to browser81x161 and640x1272.0988. Both report pending=false/loaded=true/retired=false, then both owner cancellations clear ready and loaded. [Receipt and exact helper hash](../../evidence/final-svg-candidate-native-20261019/nativesvg/receipt.json), adjacent JSONL contains observations. Earlier successful different-helper receipt is retained separately. No screenshot or visible animation claim. |
| SDK ordinary and real GPU | Initial coordinated SDK UI run passed494 tests and skipped1, with only the new placement fixture failing. Its label/occlusion assertions were corrected; [final placement receipt](../../evidence/bounded-geometry-r3-20261019/receipt.json) at`ee59f3a0` exits0 and checks exposed hit region, ordinary sibling occlusion and IME bounds. Actual hardware two-device D3D11 test at`414a82c2` exits0 in the [retained handoff receipt](../../evidence/bounded-native-handoff-20261019/receipt.json): red/blue GPU frames, real mutex contention, retained old frame, lease through resize/solver close and no normal staging/readback. That receipt's separate earlier placement step remains FAIL. Shader compile, rounded-clip ABI, caption and inert terminal flush exited0 in [SDK receipt](../../evidence/bounded-final-regression-20261019/sdk/receipt.json); Windows amd64/arm64 SDK builds exited0 in [build receipt](../../evidence/bounded-final-windows-20261019/receipt.json). Subsequent SVG changes do not alter this SDK tree. No HWND/screen capture/input was used by the GPU test. |
| New current-phase actual isolated Host | **PASS within recorded scope**, `ee59f3a0`, [receipt](../../evidence/bounded-current-host-20261019/receipt.json), [combined results](../../evidence/bounded-current-host-20261019/combined-summary.json). Host/CLI0.2.1-rc.2, pin`687f282e8625c6b8d11ff58211a90524be2e4317`, runtime provenance verified. Fresh private homes/profile; 3 durable job fixtures resume/follow, real workspace registry/files, preset/skill persistence/restarts and local Files/Git controllers passed. Owned Host/listeners stopped. No models, accounts, devices, IM, SSH, user-instance restart or session deletion. Settings skips provider/model discovery and MCP execution; counts include explicit not-run entries. Later changes only concern SVG preview/diagnostics/fixture, not these controllers. This is a new execution, not an inherited earlier Host success, and is not an aa159b6a full-app reconnect test. |

### Remaining implementation differences

- Native inline model controls do not execute arbitrary React
  `conversation.input.model.label` or `commandUi action/popupSelect` contributions.
  Original shipped1017 decoration is controlled by the client entry. Host desired
  graphs are not successful mount authority; current snapshots do not expose
  mounted registrations. No unconditional badge or invented universal bridge was
  added. Served full-page paths retain their own React registration/runtime.
  This is an actual architecture gap, not a tool block or proof from unread code.
- Beam raster sampling/one retained asynchronous frame and fluid GPU-copy plus
  hardware/resource boundaries above remain representation differences. Their
  practical visible cost has not been measured against the original.
- The finite registered-state ledger does not certify all transitive
  Markdown/tool/trajectory, pet-renderer or third-party factory implementations.
  Unread branches remain unread; they are not automatically declared missing.

### Verification gaps and exact tool blocks

- **Not executed / pending callable native capability:** same-window/DPI/font/data
  original-versus-Go screenshots and physical menu/hover/typing/IME/window input.
  Although the parent reports Ayase connected and authorized, this executor has
  no callable Ayase tools in its active inventory. CUA native APIs are explicitly
  disabled. Codex screen-context is limited to active voice/current Codex and is
  inapplicable. Raw HWND capture/injection helpers were not used as a bypass.
  This is tool exposure, not a new request for user authorization or a denial.
- Visible Beam/fluid/SVG motion, occlusion, performance, native whole-workspace
  pixel layout and physical input are unverified. Offscreen hardware textures,
  hidden SVG loading and software fixture frames do not establish them.
- Main `MaintainMuxFromBootstrap -> events Ready -> manual reconnect`, Host theme
  write/CAS and onboarding still lack current full-app integration acceptance.
  Paid/model behavior, automatic compression pressure/recovery, external account,
  real device/phone/IM/SSH and served registration runtime remain not run.
- **Installer tool preparation blocked:** three retained fixed-hash NSIS
  retrieval failures, [preparation record](../../evidence/nsis-toolchain-preparation-20261019.json).
  No fourth retry, hash bypass, install or installer/portable refresh claim.

### Review and retained failures

Independent Standards review of `2e09236a..8ff206ef`, with final affected review
`8ff206ef..aa159b6a`, found0 new hard breaches and no necessary new maintenance
finding. Spec review found2 concrete preset font/icon mismatches; `414a82c2`
corrected both and the reviewer rechecked them. SVG policy/error-receipt and final
fixture reviews found no new confirmed mismatch or weakened gate. These code
reviews do not close the architecture or native acceptance gaps above.

The earlier failed checkpoint remains below. This continuation retains eight
additional failed coordinated rounds (20 total with the prior12; successes do
not reset the count):

- `svg-stage-diagnostic-20261019`, `svg-handshake-diagnostic-20261019`;
- `bounded-final-regression-20261019` (114 ordinary package passes,1 settings
  fixture failure; SDK placement fixture failure);
- `bounded-native-handoff-20261019` (placement failure; hardware step passed);
- `bounded-native-svg-fixed-20261019`,
  `svg-decode-readiness-acceptance-20261019`;
- `svg-inherited-policy-acceptance-20261019`,
  `svg-load-state-diagnostic-20261019` (completed load exposed incorrect old size
  expectation; final two-case fixture preserves the browser behavior).

All directories are under `../../evidence/`; none was overwritten. The earlier
full ordinary run at`414a82c2` also exited0 in
`bounded-final-regression-r2-20261019/root/receipt.json`; final aa159b6a execution
above supersedes it for current ordinary acceptance. Remote CI remains0.
No push, PR, merge, release, user-instance replacement, credentials readout or
global network/security setting change. Original WhaleIsle, `C:\AI\whale-code`,
user data and delivery directories were not modified. Four pre-existing
untracked paths remain untouched. Linux validation remains canceled; no new
non-Windows validation was run during this continuation.

## Historical Windows checkpoint at f1fd8837 (superseded)

Validated product/test revision: `f1fd883760697bacc2a5aa90dbace7a864268dca`.
The final ledger commit changes documentation only. **This is not a complete
original-product or full-function acceptance.** User instruction is Windows-only;
Linux validation is canceled. Earlier non-Windows build steps remain historical
facts in their immutable receipts and are not project acceptance evidence.
No further non-Windows check was executed after that clarification.

The former39 entrypoint samples now have these concrete deep-read subpaths:

| Scope | Former sampled IDs | Deep-read behavior and integrated corrections |
|---|---|---|
| Conversation | CHAT-03/04/05/08/09/11/12/16/17 | history/resume/follow/reconnect ownership, loading versus empty, IME/undo, completion, model/preset/permission menus, voice setup, message/tool/workflow/child identity paths |
| Workspace | WORK-04/07/08/09/10/11/12, SAVE-04 | local/remote dirty editor save and close, browser membership/recording, horizontal max-two resource dock, Git review, local and Host terminal identity/resize/recovery |
| Settings | SET-01/03/04/05/14/17/18, SAVE-03 | onboarding, registered navigation/served forms, Host-owned themes/fonts and write receipts, usage/About, same-transport reconnect and Windows availability notifications |
| Remote/pet | SET-12/15/16, REMOTE-01/02/03 | IM/pet narrow save ownership, remote file drafts, observed phone authority and SSH identity/trust/tunnel lifecycle |
| Extensions | SET-10, PLUG-01/02 | physical shortcut recording/dispatch scope, Market/plugin-manager loading/detail/install/image owners |
| Effects/media | SET-06 | Windows fluid passes, layered Beam, animated raster/SVG and actual intrinsic-size metadata continuation |
| Root | PROJ-01, DESK-03/04/05 | Project row/observer/modal origin, shared caption/right-dock row contract, compact rail, native caption geometry and connection/menu integration |

Each row means its recorded selected branches were followed into their producer,
controller and state/lifetime contracts. It does not certify every state or
third-party factory in the entire module. The six domain records preserve unread
branches, including first-send transitions, arbitrary command/model factories,
all preset creator/guide paths, compression triggers, transitive remote/SSH
implementations and full pet-renderer behavior.

Known implementation fidelity limits remain: Beam uses native Gaussian/coverage
approximations; Windows fluid retains a final-frame GPU readback and bounded
hardware/resource profile rather than the original WebGL presentation. Pixel
equivalence, visible motion and performance are not certified. The native SVG
failure below is unresolved; it is not classified as missing implementation or
an external prerequisite without evidence.

| Evidence class | Current result |
|---|---|
| Complete ordinary regression | The expanded-batch `go test ./...` at589143f6 failed:102 package passes,13 failures/build blocks. That complete run remains FAIL. Later affected-package runs resolved its observed failures; they are not a new full-suite pass. |
| Changed-path ordinary checks | Main atf1fd8837 and corrected software frames exited0 in [post-review receipt](../../evidence/source-depth-review-fix-20261019T2033Z/reviewfix/receipt.json). Full frame package at25fb8fda exited0 in [frame receipt](../../evidence/source-depth-final-frame-20261019T2025Z/finalframe/receipt.json). Its later change is only the separately rerun snapshot fixture. Projects/composer/sidebar passed at8d7190cc; settings passed at0c43cea7. Other valid receipts apply only to their unchanged source/dependency trees. |
| Windows SDK | SDK tree `50ac5830db86f9274c1033eb5ecf8e62c70937ed` unchanged. Native UI, actual Windows shader compilation, surface-free rounded-clip ABI, caption and inert terminal resize tests exited0. Windows amd64/arm64 SDK builds exited0; these are build evidence, not full-product or GPU-pixel acceptance. |
| Final Windows artifacts | Main and preview helper atf1fd8837 exited0, Go1.27.2/windows-amd64/CGO0/offline. [Build receipt and hashes](../../evidence/source-depth-final-windows-20261019T2035Z/receipt.json), version `0.0.0-local.f1fd8837`. Fresh evidence output only; not an installer or refreshed delivery folder. Main was not launched. |
| Real Host | Earlier matrix rows retain their exact historical real-Host receipts. No new actual-Host acceptance bound to this expanded revision was executed; old successes are not inherited. No model/account/device/IM/SSH operations were added for verification. |
| Current native SVG | **FAIL**, [receipt](../../evidence/source-depth-native-svg-20261019T2036Z/nativesvg/receipt.json). The actual built helper was supplied to an owned hidden-parent test. After35s the wrapper rectangle was0 andready=false; expected80×160 intrinsic metadata and subsequent owner-revocation assertion were not reached. Cause remains unknown. Read-only diagnosis found no Hidden-parent admission guard; the fixture omits Seat.Error and helper stderr is discarded, so startup/handshake/metadata/paint stages cannot be distinguished. No retry or safeguard bypass. |
| Original/native Windows visual/input | **Unverified**: supported native capture/input APIs are disabled. Same window/DPI/font/data-state original-versus-Go screenshots, actual native control hit testing, GPU motion and physical input remain absent. Software frames cannot substitute. |
| Installer | **Unverified**: NSIS tool preparation retained three failed fixed-hash retrievals. No fourth attempt, hash bypass, installation or packaging claim. |

Root inspected the latest f1fd8837 software frames at900/1200/1600 DIP, including
light/dark, compact rail, populated long message and open terminal/right pane.
The title, Git controls and checkbox labels are readable after the fixture uses
the actual production native palette. The input card and terminal remain in the
center track; the footer stays pinned. Native OS caption controls are absent
from software painting, and the original matching screen is unavailable, so this
observation is only fixture evidence. Files are in
`../../evidence/source-depth-review-fix-20261019T2033Z/reviewfix/software-frames/`.

The retained failed local round count is12: six earlier rounds, five expanded
ordinary failures and this native SVG failure. Later passes do not reset it.
Remote CI remains0; no push, PR, merge, release or current-directory replacement.
Four pre-existing untracked acceptance/tool paths remain untouched.

## Standards

Independent read-only review `ffe7ab66...25fb8fda`, followed only by review of
`25fb8fda..f1fd8837`: no confirmed documented hard-standard breach. Review
covered MyGo thread forwarding, native callback initialization, ownership,
asynchronous Git reads and settings write receipts against global AGENTS/workflow
and MyGo AGENTS/architecture.

Two nonblocking, explicitly heuristic findings remain:

- Possible Primitive Obsession: `third_party/mygo/ui/physical_shortcut.go`
  overloads `identity ...string`, using `identity[1] == "terminal"` as scope.
  A named option/type could make this contract clearer in future work.
- Possible Duplicated Code: settings `host_theme.go` and `instructions_write.go`
  share mutation/owner/namespace receipt-settlement shapes. Their queue rules
  differ; no unrelated extraction was made during parity work.

## Spec

Independent read-only review found one concrete mismatch with “以原版作为产品验收
标准…区域尺寸、位置…菜单…行为”: normal titlebar Commit/Publish changed the right
dock, while original `GitActionsControl.tsx:502` only opens its dialog.
`75cfdc0b` uses the existing owned modal for both normal/global pages and preserves
the selected dock and PR state. `f1fd8837` exercises actual binder/buttons across
both page types and four dock states; opening/closing does not prepare or execute
Git/model actions. The reviewer rechecked and marked this code finding resolved;
the corresponding main run exited0.

The two acknowledged remaining Spec gaps are “同窗口尺寸、DPI、字体和相同页面数据
状态” native comparison, and “全部功能正常用” exhaustive functional acceptance.
No additional substantiated scope creep was found in reviewed paths. Both axes
are code conclusions and do not replace native visual/function acceptance.

Review totals: Standards0 hard breaches plus2 maintenance heuristics; Spec1
concrete mismatch corrected, with2 acceptance/scope gaps still open.

- Checkout: `C:\Users\48818\Documents\Codex\2026-10-19\task-2\WhaleIsle-MyGo`.
- Branch: `codex/session-runtime-compat-20261019`; no merge, push, release, or CI.
- Reference: original repository commit `e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`.
  Its worktree contains pre-existing modifications. Reference behavior is read
  from `git show e353e8e:<path>` or from an explicitly checked clean file.
  Neither the original repository nor the user's other checkout was modified.

## Integration responsibilities

Six exclusive source owners implement conversation/voice, workspace resources,
settings, remote/Pet, extensions, and effects/media. Root owns main/frame and
their cross-domain contracts. Source implementation precedes preparing new
regressions and coordinated validation, as the user explicitly requested.

### Conversation, voice and media entrypoints

- A selected session with a live `Presentation` has active conversation content,
  even before text messages arrive; it does not show the welcome hero.
- Slash-menu selections call the captured session/Host command executor.
- Voice settings navigate to the actual optional voice bundle in the Plugins
  manager. The manager renders the provider/language/preparation/microphone page;
  hiding the page or window relinquishes device preview ownership.
- Speech catalog watching reconnects with the real event transport ready path.
  It is fenced by the current Store/Client/event owner and canceled-attempt
  context; late Ready from the old attempt cannot replace the new catalog.
- The composer card uses the reference 22 DIP radius. Beam draws below children,
  with the hairline above; its inner wash cannot cover text.
- SVG owners carry window, selected session and Host epoch into actual Composer,
  Files and Market callers. The existing preview helper uses inert image mode;
  inline SVG routes use `BuildSVGImage`, not an interactive HTML document.
- Markdown respects explicit secondary font sizes; primary chat callers use the
  accepted content preference rather than an unconditional 14 DIP override.

### Files, Browser and terminal ownership

- Browser workspace membership comes from authoritative direct workspace rows,
  with loading/error readiness and session fallback. No parent/CWD inference or
  config fallback can join unrelated partitions.
- Every Files resource pane owns its remote tree, tabs and drafts persistently.
  The remote renderer receives that pane's actual `session.Store`; hidden pane
  retirement cancels transient requests without erasing drafts. An unresolved
  remote mirror still permits the existing local Files renderer.
- Closing a resource pane checks local and remote drafts. Save-and-close first
  preflights both domains, queues explicit writes once and waits for their
  receipts. Errors, changed owners and newer drafts keep the pane present.
  If one domain fails after the other has started, both receipt lifecycles
  continue to drain, retaining the error and disabling repeated close actions.
  An error cannot strand the remote close cursor or local autosave suppression.
- Host terminal descriptors persist identity only. Restoring adopts a matching
  existing terminal; explicit new terminal tabs carry a separate runtime fresh
  marker. Clones/restored descriptors never allocate a shell implicitly.
- Local terminal resize flushes before hiding the drawer/window or changing its
  workspace, in addition to the terminal's own disposal flush.
- Dock splits are horizontal, at most two docked seats, with .2 minimum fraction
  and a measured 100 DIP chip-fit condition. Center/tab moves do not consume a
  new split. Malformed IDs/trees are rejected before content restoration.
  Document/subagent resource openers use the same admission. The new SDK
  `SplitFraction` updates the persistent fraction itself during dragging and
  keyboard resizing, rather than a discarded per-frame temporary width.
- Valid older Go vertical or larger layouts migrate without dropping resources:
  splits become horizontal and surplus seats fold into retained tab groups.
  Drafts and floating resource identities remain attached to those resources.

### Project and desktop source findings

`PROJ-01` deeper comparison follows pinned `vendor/dsh-project/client/client.js`
`SidebarPage`, `ProjectRow`, `ProjectHeader`, `command` and `refresh`, plus the
native project bridge and downstream create service:

- Project title filtering uses the shared sidebar query, not a second input.
- The row overflow contains Rename, Open Directory, Archive/Restore. Detailed
  Project routes remain accessible through the original observer entrypoints.
- Empty active Projects expose the reference Add action. Selected folder tint
  follows selected foreground; the coordinator header opens its directory.
- A failed list refresh preserves prior projects. A confirmed rename/create
  receipt remains accepted even when the subsequent read fails; mutation is not
  replayed. Rename uses the original 240 UTF-16-unit bound.
- Create already delegates to the original Host `select-directory` bridge. An
  additional native picker would prompt twice and was not added.

`DESK-03` normal window defaults are 1440x920 and minimum 960x640. A second
instance with `--background` does not foreground the active instance; initial
background startup shows inactive when not configured to remain hidden. The
reference 32x32 caption buttons with top/right 8 DIP are an independent overlay,
not another 48 DIP row in the content layout. SDK caption/rounded-frame source
is committed in `915edb4`: app-scoped caption options and a 20 DIP native
DirectComposition clip, including CPU/GPU and ARM64 COM paths. The root's inside
one-physical-pixel frame ring adds no content padding. Snap/resize remain native.

`DESK-04/05` deeper physical chord recording, stale application menu callbacks and
fixed-reference desktop integration are recorded in
[the physical shortcut source record](source-depth-desktop-shortcuts-20261019.md)
and committed in `2ba3b072`. Root wires the actual three Host factories and
defers both native application-menu dispatch and product-close dispatch until
the current frame has built its modal ownership. Product close targets the
captured owning window and respects its existing OnClose draft protection.
No new deeplink routes are inferred: the fixed original has no desktop custom
URI registration matching the existing Go extension.

### Settings, Plugins and Git shared wiring

- Accepted Host `ui-theme` is the connected application authority. Local config
  is the launcher/pre-Host source; ordinary Desktop Save does not mirror Host
  scheme or image data into launcher config. Pending Host writes preserve the
  native editor draft. Wallpaper data URLs decode in memory.
- The account introduction is the original resumable desktop onboarding flow,
  using display-safe account/balance and credential metadata. It sends no model
  request and does not read a credential literal.
- Subagent and web settings use their specialized served forms once, without
  rendering a duplicate generic namespace form below them.
- The desktop curated Market is always available as in the original; unrelated
  disabled plugin preferences remain preserved.
- Plugins Creator uses its actual `marketOpen && marketManager` page owner and
  the existing captured `CreatorTask` admission: create one session, apply
  `cordis`, then open. It does not seed or send a prompt. Unknown outcomes retain
  the exact target with Locate/Open recovery actions.
- Install registry choice persists as a non-secret native preference, with load
  and save errors returned to the manager. No registry action was executed.
- Git titlebar consumes the existing snapshot's local `RepositoryStatus`, shows
  ahead/behind and uses source `CanPush`/`CanCreatePR` gates. No extra fetch,
  account lookup or PR creation is performed by status rendering.
- Assistant configuration failures reach the Pet settings error receipt.
- Desktop update feedback comes from actual staging phases, not byte-count
  inference: checking, downloading, verifying, ready and installing. Source
  producers in `708e442b` cover full and delta/fallback pipelines. The seat's
  available gate excludes prereleases like the existing launcher notice.
- SVG gallery, README, attachment and actual plugin-icon callers share the
  inert image owner. `009c57bf` admits absolute units and lets the existing
  helper report real intrinsic sizes for relative/missing dimensions. The app
  and preview helper must be built together; an older helper's missing metadata
  is an explicit failure, not an assumed SVG success.

### Transport recovery and retained interactions

The reference reconnect action retires the current transport, not the Host
process. Root has replaced both restart callbacks with `requestConnectionReconnect`
and applied the reviewed, attempt-fenced `runDSH` wiring from `32acf481`.
It keeps the same Mux, Client and Stores, publishes ready only after the current
`$events` Ready, and ignores old event/analytics projections. Stream handles
capture their carrier; an old cancel/read failure cannot mutate the replacement.

`1f565763` adds an explicit interaction delivery owner and Retire hook. Transport
retirement releases that attempt's waiter and HTTP authority, retains callId
drafts, and creates no permanent Host-cancel tombstone. A same-eventId replay gets
a new waiter. Old answers, cancellation callbacks and late HTTP results cannot
consume or remove its replacement. Actual Host Cancel remains separate.

`04e7961c` implements the configurable recovery timings from the actual Host
bootstrap global: 500ms base, factor2/max10000ms and 50–100% jitter, plus the
3000ms readiness warning and 15000ms whole-generation deadline including Dial.
Root's second narrow handoff binds the `$events` completion before starting it,
then waits for retirement/join before the next generation. An unexpected events
end uses automatic recovery; only an explicit user retry skips backoff.

The same maintainer now also owns bootstrap loading. Read/parse failure keeps a
visible error and an operational Retry on the same Host/Client. It does not
silently accept defaults for invalid configuration or leave a dead retry button.
The bootstrap document is bounded, same-origin, parsed without execution and
never logged or persisted. No running user configuration was inspected.

`4132225c` closes the Windows network-availability producer using local OS
connectivity hints and change notifications. Unknown stays unknown; offline
pauses automatic attempts, an explicit retry admits one attempt, and online
restarts the initial backoff. Root wires its lifetime to the same Host attempt
owner. No polling service, external probe or network setting is introduced.

The remote close audit additionally found committed native text could precede
the UI's delayed OnChange. `915f9452` gives each captured remote resource its own
native buffer; dirty/preflight/save/check/suspend paths capture committed text
from that owner before close or reset decisions. IME marked text and another
session's buffer are not substituted. The fixed original remote contribution
does not register whole-window persistence/close protection; this work does not
invent or claim remote draft restoration after process exit.

### Shared AppFrame topology and actual workspace open-in-app

The fixed Whale shell injects its own 32x32 controls at top8/right8 and reserves
112 DIP. Its preload does not set the upstream standalone desktop's
`data-windows-titlebar` marker; importing that other branch's extra 40px spacer
was incorrect. There is no extra 12-DIP spacer above the ordinary native shell.

`138abe63` and root integration use one auto-sized shared title row (floor52,
ordinary title/tabs approximately72, title without tabs53). Conversation headers
and the compact Git/panel trailing cluster share that row; right dock tabs start
in row two. Measured trailing width reserves only its overlap with the center,
including native control space. Windows collapsed navigation remains56 DIP.
Project header actions are in the title cluster rather than a separate row.

`ff0d510f` provides compact Git branches/menu/progress and the panel toggles.
Retained main-session cwd, managed composer and actual ui-git/titlebar settings
control their visibility. Global main panels retain this title row and the same
terminal panel/drawer. They intentionally have no active right-dock surface;
the Surfaces button cannot alter hidden state or redirect to chat. The exact
fixed-source chain is recorded in the settings source record (`40490dac`).
`28b0a5b9` follows the actual direct Initialize action and keeps global commit,
push and publish review visible in one existing Workbench modal. Review captures
root/Host/session admission, retires on replacement, and never switches panels.
Global terminal file/loopback targets use the same captured handler and retained
resource state; they open the dock for later conversation visibility while the
global panel remains active. External targets retain the existing OS action.

The trailing width and shared-row height are observed after layout/commit in a
paint receipt, including width zero, with rounding of the controls inset. Only
changed geometry requests another frame; asynchronous Git arrival and all-toggle
hide no longer depend on an unrelated input event or a permanent timer.

`a0a40e39` adds the actual open-in-app application list, split picker, default
choice and captured POST contract. Root supplies once-per-Host lifecycle,
native preference persistence and both command routes in the main conversation
scope. It does not substitute a fixed file-manager launch or a configured cwd
fallback for the original selected-session action. Pending/default/error states
share the header and command controller; Host replacement retires old requests.

SVG restore now invalidates the window after visibility resumes. The frame ring
uses the painter's already-inside stroke directly, keeping its edge at exactly
one physical pixel without an additional half-pixel inset.

## Evidence status

Source-owner commits include `67a6924`, `590e30f`, `b103ae7`, `b2d7273`,
`bcf3d5f`, `5c2e6fc`, `4bfd992`, `ff6ac28`, `91e7916`, `e760f50`, `d93e409`,
`915edb4`, `2ba3b072`, `b8228820`, `009c57bf`, `32acf481`, `1f565763`,
`4132225c`, `915f9452`, `70d4e0d2`, `138abe63`, `a0a40e39` and `28b0a5b9`.
Domain records retain
the exact branch/state evidence and explicit unreviewed branches.

Root has formatted its exclusive source modules and checked their whitespace.
Shared `main.go`/`frame.go` were edited with narrow hunks, preserving concurrent
work. No new-phase tests, fixtures, builds, shader compilation, app/helper
startup, device acquisition, Host mutation or user configuration operation was
performed for this source integration milestone.

All six source owners and root have now closed the confirmed changes of this
expanded source batch. This admits regression preparation next. It does not
change the coverage index's explicit non-exhaustive boundary or certify every
original dynamic contribution/state.

The previous phase's passing component/build receipts belong to its older
source revision. They do not validate these new changes. Real Windows native
visual comparison, GPU/helper containment, full external-service behavior and
packaging remain separate acceptance obligations, not inferred successes.

## First coordinated expanded-batch run

Source `589143f67b3a7f80f993a20d71d6f261305843d9`, tree
`002514352c23ca9a22c338a11aee65d29679f7a0`, ran only after all six source/fixture
owners and root had committed. Receipts are in
`../evidence/source-depth-unified-20261019T1940Z` relative to the checkout's parent.
Each process used Go1.27.2/windows-amd64/CGO0, offline module cache and fresh owned
userdata; inherited product acceptance opt-ins were cleared only in that process.

- Application `go test -json -count=1 -timeout=3m ./...`: exit1, 102 package
  passes and13 package failures/build blocks. Full JSON, build errors, failed
  test output and the settings fatal/timeout remain in `root/root-all.jsonl`,
  `root/failures.txt` and `root/receipt.json`. No passing application conclusion.
- SDK `./ui`: exit0. Includes persistent SplitFraction input/resize and native
  physical-key driver coverage; this is software/headless UI evidence.
- SDK fluid: real Windows D3DCompile and DXBC layout tests exited0, covering
  profile128/1440 and the48-byte/splat/ink ABI. No GPU output comparison claimed.
- SDK rounded clip: real surface-free COM property-tree ABI test exited0.
- SDK native caption checks and the explicit inert terminal resize/close tests
  exited0. No real PTY was started or downloaded.

Three invalid `ui.RoleAlert` references caused several UI packages to be build
blocked; `aa46bc67` replaces them with the existing inline `RoleStatus` role.
Other failures are being traced against fixed original behavior, including
history window/EOF fixtures, two-level model menu paths, asynchronous Host theme
receipts, SVG renderer ownership, and strict generic versus explicit-editor save
authority. A synctest bubble crossed persistent raster workers in a new settings
fixture; its fatal/timeout is a failure, not a skipped or passing check.

This adds one failed local round to the six retained earlier local rounds.
Remote CI remains zero. Corrections must complete before a coordinated recheck;
unaffected SDK results retain their exact source-tree boundary.

Corrections prepared before the second coordinated run:

- `2597798e`: accepted Host defaults/font normalization/registered-page fixtures
  follow their actual consumers; rendered UI is outside the synctest bubble.
- `227c2159`: fixes SVG native/helper routing and failed README image fallback,
  retaining every prior pixel/alt/paragraph assertion.
- `1f7b8059`: generic checked saves reject incomplete rereads. Only the local
  editor's real explicit Flush of a dirty draft may use the original exception;
  later input cannot inherit that gesture. Prefix safety assertions remain.
- `16dddb2b`: validates dense history before accepting/merging its older subset,
  matches the original model menu/current-choice and busy-browsing behavior,
  and prevents unavailable voice feedback from admitting retry recording.

Root's final interaction audit also found synchronous `WorkspaceSnapshot`
directly in the frame/menu callback, with its existing ten-second timeout. The
native Git refresh now admits a bounded background `ReadRepositoryStatus`,
cancels a replaced root/window owner, rejects late results, and waits for
mutations to settle before publishing a fresh status. A completed operation
invalidates the selected frame instead of forcing its old workspace back into
the status reader. No fetch/account lookup or Git mutation is added. New tests
hold the reader to prove admission returns, then inspect the actual completion;
separate receipt cases reject retired and mutation-racing owners. These changes
have only been formatted/diff-checked at this point, not declared passing.

## Second coordinated expanded-batch run

Source `fb5da091b572e72465c722840773e8aedaa5aa6e`, tree
`9289da1f3ac9a7d9127bc643e6478d10b89af502`, rechecked only the fifteen
previously failed/build-blocked or directly changed consumer packages. Receipt:
`../evidence/source-depth-recheck-20261019T2000Z/recheck/receipt.json`.
The process exited1: ten package passes, five failures/build blocks. Its JSON,
stderr and extracted failures remain beside the receipt. This brings the known
retained failed local rounds to eight; remote CI remains zero.

The frame terminal occurrence code referenced a nonexistent method instead of
the existing atomic `Snapshot().ComposerHostGeneration` field, blocking frame
and main. `bcfc353f` reads session and generation from the same snapshot.
The composer keyboard path captured a frame-local Element in a later input
callback; it requires a stable focus owner. The project search/context-menu
fixtures and shortcut clear/default fixture still described removed UI paths.
Their owners are tracing/fixing those exact cases before the next coordinated
execution. No broad SDK rerun is admitted: its tested tree remains unchanged.

The third run, source `0c43cea7`, is retained in
`../evidence/source-depth-narrow-20261019T2006Z/narrow`: exit1, settings/sidebar
passed, main/frame/projects/composer failed or build-blocked. Go1.27 build
diagnostics are `build-output` entries in the JSON, not necessarily stderr.
There are now nine retained failed local rounds; remote CI remains zero.
Main exposed an int-to-uint64 connection-generation assignment and stale menu
gesture/physical-chord fixtures. After frame finally compiled, its older
caption/rail, raw-pointer coordinates and vertical-dock fixtures were reached.
Their assertions must follow the fixed source and real physical input producer.

Two newly reached failures identify production focus gaps: non-bubbling model
menu Escape and a Project Activity child without its task-action focus origin.
The compact sidebar audit additionally found missing real rail actions and a
replaced footer identity. `47de782b` restores the original generic 56-DIP rail
actions and common footer; actual Whale preload does not enable the upstream
standalone Windows titlebar flag. These corrections remain subject to a new
coordinated check, with all original failure receipts retained.
