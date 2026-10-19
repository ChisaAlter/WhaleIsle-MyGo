# Original-source parity implementation batch

# Current source follow-up (2026-10-19, concentrated regression corrections)

Earlier “batch closed” statements below describe their recorded source snapshots. They do not close the complete original application. The registered 91-ID inventory still contains sampled or unread states; see [the coverage index](source-coverage-index-20261019.md).

Run01 at `3b37750` and run02 at `221dba0` exited 1. Run03 at `6fec511` passed every root package except the main package (cached Activity field migration compile error), composer/Files (cross-synctest raster fixture failure) and Market (float geometry equality). The corrected composer and Market packages passed at `c5be241`; Files passed at `92d0327`. The main package's duplicate native theme query was removed in `63aeeb7`, and its theme/owner tests now pass; its completed latest run still failed one IME checkpoint fixture. Source attribution found that fixture reused the original selection instead of the current marked range; `66c84f1` corrects the input sequence and adds native-text assertions without changing the SDK or checkpoint product. Its single-main-package recheck is pending. No current Windows binary has yet been built.

| Evidence snapshot | Actual outcome / reuse boundary |
|---|---|
| `../evidence/source-parity-unified-20261019T161408Z/` | Run02 root exit 1; nested UI SDK `go test ./ui` exit 0, tree `79a93d817ed8f2c9a7c72a202cb07093037e2ce8` unchanged through `92d0327` |
| `../evidence/source-parity-unified-20261019T163227Z/` | Run03 root exit 1; all packages except the four listed above passed |
| `../evidence/source-parity-recheck-20261019T164454Z/` | Four affected packages at `c5be241`: composer and Market pass; main theme-query timeout and Files fixture alias fail |
| `../evidence/source-parity-targeted-20261019T165904Z/` | Main/Files at `92d0327`: Files pass; main finishes in 4.730s, only selection-marked-commit visibility checkpoint fails |

There have been six failed local validation rounds including the earlier admitted compile attempt. These are local results, not CI attempts: remote CI remains untriggered (zero attempts). The earlier pre-source-closure compile request was denied before execution and generated no result. Passed component receipts are retained only while their source/dependency trees are unchanged; this table does not claim a single all-green repository invocation or actual desktop acceptance.

Additional source comparison found implementable gaps, so source work resumed before a second unified run. Native Market screenshot catalog/gallery, GIF/WebP README animation, broader bounded static SVG, compact child composer and original Workflow row rendering are committed. `8e1694c` now preserves actual subagent descriptor identity and permits both original workflow navigation modes; `c6854e8` restores desktop applyConfig’s public config response. These are source changes, not new regression or desktop acceptance claims.

The identified source follow-up was frozen for run02; corrective source work and the two-axis review now address its observed failures and newly exposed paths. Verification is deferred while those source corrections and necessary fixtures are being prepared. The current source ledger records 52 rows with bounded behavior comparisons and 39 with entry-point sampling. These 91 labels are not an exhaustive original-feature/state review: each linked report retains unread states, dynamic contributions and platform-specific limits.

| Follow-up source behavior | Concrete implementation commits |
|---|---|
| Jobs ambient state/hover counts, Schedule visibility/edit/delete/receipt ownership | `4c5318f`, `6336ea7`, `291b23e`, `f38841b`, `b6e33e5`, `e63a1c6` |
| Question profile-owned drafts, Host continuation/queue-discard facts across coalesced control frames | `2c21cc0`, `531a15d`, `6c8a569`, `d9a1372` |
| Automatic delivery reads, recorded native application choices, history references and local retry repaint | `2c21cc0`, `0f4e9ea`, `e160cb7`, `8281fd9`, `9b72ad6` |
| Mounted visual reply lifetime, typed permanent errors and declared initial height | `2c21cc0`, `9cf03fc`, `e6e842c` |
| Recovery checkpoint debounce/max-wait/retry and hide/minimize capture; terminal drawer restore without replay | `59c3582`, `02c6bd3`, `6755c17`, `3a3064f` |
| Files committed native input before paint, plus selected-range IME capture without persisting preedit | `bca41bb`, `99f732e`, `e11ebc0` |
| Original desktop Files local source restored after withdrawing an over-broad Host-only inference | `5e918af`, `be7a34e`; [exact original route and correction](source-review-native-file-references-20261019.md) |
| CSV/TSV default desktop edit/save and source-consistent observation; XLS/XLSX remain table previews | `8b25c42`, `4f5055e`, `e6db66c`, `d9c99ca`; prepared round-trip/default chooser fixtures `5744f60` |
| Desktop Files AVIF sequence route uses the already bounded shared decoder; composer/Market route retained | `ce67a12`, `3d61894`; [format limits](source-coverage-media-20261019.md) |
| Pet transparent-region input gating and Room timeout aliases | `af11d25`, `4f08402` |
| Narrow shortcut source reload without rebasing unrelated stale desktop drafts | `9cf5e01`, `1ba34d5` |
| Transactional installer/rollback/partial-uninstall registration and legacy uninstaller retirement | `b62ec7f`, `472ea92`, `863ce35` |

The local-file correction is material: the original `desktop-file` extension-priority resource type takes over session-relative non-Office files before the generic Host document viewer. It uses desktop file/media/write IPC, includes AVIF, and deliberately keeps CSV/TSV editable. Earlier blanket Host-only and fallback-only format conclusions were withdrawn and their mistaken fixtures removed before execution. A session resource address does not alone establish the bytes source.

Native Windows/original screenshots and physical input remain blocked by unavailable supported native capture/input. NSIS compilation is separately blocked by compiler acquisition: all three official archive/mirror retrievals returned HTML and failed the published SHA-256; no bytes were extracted or run. `../evidence/nsis-toolchain-preparation-20261019.json` records that tool-acquisition failure. This is not a product regression round. Local checks and CI counts remain separate as recorded above.

The existing delivered folder and user instances have not been replaced. No new package, release, remote push, main merge or installer execution has occurred. Source implementation, ordinary regression, real Host operation and Windows/original same-state visual acceptance remain separate evidence levels.

Branch: `codex/session-runtime-compat-20261019`. Repository: the task-local `WhaleIsle-MyGo` checkout. Original `C:/AI/Deepseek-Harness-Desktop` is read-only. The installed original, `C:/AI/whale-code`, dirty user checkout, user instances and delivered `WhaleIsle-MyGo-local-9232f52` folder/data are untouched.

The latest instruction is source implementation first, then one concentrated validation phase. New startup runs, mock services, package copying and staged deliveries are paused. These commits record implementation, not product acceptance.

| Area | Original source / implemented difference | Source commits |
|---|---|---|
| Launcher/components/import | `src/renderer/launcher-components.js`, `launcher.js`: running controls, per-component pending/status, real management menu, unsupported states, workspace groups and partial selection, keyboard navigation, diagnostic/recovery actions; component list runs outside the input/render thread | `55beab7`, `12ea3db`, integration `b48c10d` |
| Task lifecycle | `src/main/index.js`, `launcher-service.js`: asynchronous Stop/Restart/Close/update/uninstall, original task confirmation retained; live local PTY work is counted and approved runtime changes clean up its resources | `62c20f3`, `600ceaa`, `edba766`, `54a25bc`, integration `b48c10d` |
| Conversation/composer | Original session-controller, input facade and QueueDock: late Stop ownership, convergent steer results, command echo/engaging behavior, canceled splice receipts, failed attachment restoration and stale queue editor retirement | `2b7dca9`, `7d37fea`, `5924f4c` |
| Bot/Room/routine | Original `vendor/dshbot/client/client.js`: cached model/reasoning/default selection at creation, cached Host workspace selection, routine context/silent/watch values submitted together in the original save operation | `f7f49f7`, `ed79b3e`, `3b28db7` |
| MCP/Skills/plugins | Original client modules: explicit failed-managed-connection retry, connection/tool summaries, full skill creation, source buckets and metadata search, scope retirement, preset-provided plugin status | `898c89e`, `23d2617`, `d00faab`, `176e06c`, `e585ad4`, integration `b48c10d` |
| Presets/install | Original preset descriptions/Markdown guides; install close cancellation, aborting inspection, too-late and lost-response reconciliation of the existing request, without resending install | `59fdb1d`, `0f2485b`, `60512f5` |
| General/About/Usage/Appearance | Original settings CSS/components and usage styles: role-specific font controls, narrow field saves, pending family blur/navigation flush, immediate narrow scheme selection, real Harness version, typography, heatmap/KPI density, responsive layout and loading/error states | `92a9b56`, `412914c`, `98060f3`, integration `b48c10d` |
| Project | Original `ProjectTaskCard`, `RequestDiff`, `ProjectTaskDelivery`, `ProjectVisualResults`: missing requirement changes/revert paths, compact Undo, released/settled work projection, human status priority, real page attachment object/height decoding and published-page reader | `1b9ba61`, `7c3ec88`, `8fe62b1`, `6fb6ac1`, `56eaa6a`, `9b2ceb4`, integration `b48c10d`; final cache wiring in the source-closure commit |

Static inspection also prevented duplicate implementations: existing Bot route editing, Room reply recovery, notification suppression and task retry already had product paths. Original Agent Presets has no direct edit/delete API, so none was invented.

Correction to earlier evidence: `9da81d3` fixes only the base update-check callback. `withNativeReleases` overrides the actual launcher callback with an already asynchronous release refresh. The gated helper experiment is not the actual launcher button's root-cause or acceptance evidence. `995cbe0` remains the independently demonstrated exact-line-height correction.

An attempted early `go build -o NUL ./cmd/whaleisle` was rejected by automatic approval review before execution because source work was still incomplete. It generated no compile result or binary. No workaround or alternate compile was run. Source work continues; comprehensive regression, Windows build, native/OS operation and packaging remain deferred until the implementation scope is finished.

The identified source implementation batch is closed: responsive inline Project pages, structured unified requirement diff, source-sized revert confirmation, retained independent card pages, member labels, and remote/IM diagnostics, lifecycle, workspace and preset controls are implemented. Remote/IM source commits are `83f2ba7`, `467a29a`, `ab32493`, `bf3e8bf`. This closes the planned source edits, not acceptance or the entire product. Concentrated review and regression/build are next; no new-source compile, runtime or screenshot result has been claimed. GPU fluid/GIF expansion remains paused by the earlier priority decision. Matching original/Go desktop pixels and physical OS input still require the supported native computer-use capability. Existing older receipts apply only to their recorded source and operation.

## Full-function source follow-up

After the identified batch was saved at 0bdb900, the parent reiterated that the target is every function, not only primary paths. Parallel source audits now cover lifecycle/import/components, Project/settings, conversation, Bot/Room/IM and MCP/Skills/plugins. Implementation and acceptance remain separate; no new packaging or service launch has occurred.

A single build was admitted after 0bdb900 supplied materially new evidence that the identified batch was closed. It failed at internal/ui/settings/skill_create.go:103: ui.Local received a Context instead of an Element. The validation state now belongs to the stable Name label element. No executable or passing compile is claimed, and compilation has not been repeated. The earlier rejected attempt remains unexecuted. Further verification is paused while the broadened source audit and confirmed fixes continue.

### Confirmed source gaps addressed by the full-function follow-up

| Gap found by original-source audit | Source implementation | Local commit / integration |
|---|---|---|
| Windows retained session store hid Start after Stop/import; lifecycle actions could overlap | Runtime availability excludes stopped/failed stores; pending lifecycle blocks conflicting Start/Stop | `336b3e1` |
| Desktop config API restarted with old overlay flags and without protected cleanup; saved shell settings did not apply | Dedicated patch module, original import maintenance slot, narrow UI effects and existing protected restart; concurrent save/effect ordering retained | `336b3e1` plus follow-up ordering fix |
| WhaleBridge settings action used the generic runner that rejects this builtin ID | Existing builtin `RunAction(open/manage)` path | `336b3e1` |
| Diagnostics changed the runtime while retaining local PTYs | Existing approved `closeTaskProtectedResources` cleanup before Stop | `336b3e1` |
| Skill Hub had no independent installation scope and skipped the original review/busy lifecycle | User/project roots, source/path/scope review, owned requests and installation result projection | `657696a`, shared wiring `336b3e1` |
| Expanded card A lost Undo when card B was read | Authoritative per-card revision action, per-card conflict and coordinator draft | `38b7565`, shared wiring `336b3e1` |
| Bot task/routine lists lacked original search, task status selection and source sorting; missed-run notice was not dismissible | Existing local snapshots filtered/sorted; dismissal scoped to the canonical session | `95bdfa9` |
| Non-Feishu channel connection test was unreachable | Original supported-platform reconnect-test review/action wired without executing it | `fe3c43a` |
| Attachment send was forgotten at RPC acceptance before canceled/unadmitted retirement | Attachment-only retirement wait restores the captured failed draft; durable observation succeeds, no automatic resend | `f5c3f6d` |
| Workflow disclosure did not follow later run/phase facts or defer clean collapse around focus | Original disclosure state progression and stable content focus ownership | `f5c3f6d` |

The final source follow-up is now committed: independent retained-card file/diff/detail/activity/recovery actions (`9109718`, integration `40374c4`); complete Next Prompt readiness/interaction/managed projection (`0f35dfb`); original Room polling ownership (`27a8cab`); conversation/agents deep-link routing (`e322792`); and settlement of unobserved attachment sends only on durable deletion (`911ff74`, narrowing `1abc48d`). Ordinary removed sessions retain reload semantics. Source review found no further confirmed P1/P2 in these reviewed paths. This is source closure for the identified Windows batch, not full product acceptance. Concentrated regressions are being prepared in separate test commits; no additional compilation/test execution, mock services, packaging or runtime acceptance has occurred after the single recorded compile failure.
