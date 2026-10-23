# Round 104 — Inspector stream lifecycle, summaries and settlement

## Result

The native Raw Log now shows accepted live chunks, groups interleaved chunks beneath their own block headers without renumbering, closes each block when its matching block-end arrives, and shows the bounded reasoning prefix on a folded header. Manual disclosures carry the lifecycle state in which they were made. A previous open-state override cannot keep a completed block open, while a new override can reopen the completed block. Active type filtering overrides source lifecycle defaults for its retained hierarchy, so matches are initially visible and subsequent manual folding remains effective.

Successful settlement retains a separate live root alongside the durable message until the matching Step end. Failed/interrupted settlement preserves the observed attempt key and original chunk identities. These are distinct reference behaviors, verified against the real installed controller/model and native Store with the same [follow frames](../../internal/ui/frame/testdata/inspector-stream.json).

The [default executable](../../whaleisle.exe) is updated from pass248. The whole product parity goal remains active.

## Reference and implementation

The reference [stream](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/session-log/stream.ts), [log model](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/session-log/model.ts), [range groups](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/session-log/groups.ts), [disclosure resolution](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/table-model.ts), [filter](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/type-filter.ts) and [Assistant controller](../../reference/electron/vendor/deepseek-harness/packages/api/session-controller/src/client/sessions/assistant-stream.ts) define the source behavior. Both reference source and installed bundles remained read only.

- Turn/Step start headers have an explicit open default and retain it after end. Assistant roots have no explicit default. Each block-start is open; only its matching block-end closes it. An unfinished block in a failed durable attempt stays open. Turn end belongs directly to Turn even when a Step has no end; missing starts create no invented headers.
- A chunk keeps its original receive ordinal, raw value, timestamp and location. Interleaved blocks become contiguous depth-first subtrees. A chunk whose indexed start was not loaded stays under Assistant; later starts do not move it. Unindexed finish/usage chunks remain Assistant children, rather than accidentally attaching to block zero.
- A reasoning header joins only matching reasoning-delta text, including while unfinished. Its whole collapsed preview is bounded to 240 UTF-16 units without ellipsis. The final block's text cannot replace that delta prefix; raw values are unchanged. Text/tool headers keep their normal previews.
- Every accepted live chunk is retained once after revision/attempt/dense-index validation and invalidates the view, even if it changes no Chat body. Baseline replay folds the already retained compact stream without appending duplicates. A start alone creates no row. Unknown suffixes are ignored and protocol faults preserve the last good prefix until replacement.
- A live root anchors in the loaded event gap at its first actual chunk. Later chunks stay at that root even after newer durable events arrive. An empty reconnect prefix leaves that gap undecided until a real chunk arrives; a reconstructed nonempty prefix appears after the replacement window.
- Live metadata is assistant/live with attempt identity, Turn/Step, latest chunk time and the compact value {attemptId,turn,step,chunks}; private fold internals do not enter that value. Successful messages remain event:<seq> while attempt:<id> stays live until matching Step end. Failed/interrupted settlement replaces only an observed live root under its attempt key. Loaded failure aliases survive same-generation replacement; eviction removes them before subsequent append/prepend can reintroduce a durable event. New Host/session generations cannot borrow aliases.
- The table stores {state,collapsed} choices. Active filters strip source disclosure and assign open only to retained parents. Matching headers without retained descendants have no empty arrow. Reveal writes an explicit open override using the resulting hierarchy state, so it can reveal inside completed blocks. Binding/mode/filter confirmation resets these choices as before.
- A live record targets actual native Message IDs. Indexed rows use exact block identity; root and unindexed/empty blocks can reveal an existing visible block from the same attempt. Forward reveal fallback is separate from exact reverse-pick ownership, so an empty header cannot intercept a pick for another block. Retained success uses the canonical settlement sequence rather than retired prefix IDs, and reverse durable picks skip the retained assistant/live root. Settled aliases remain proper roots in the native object graph.
- The newly populated canonical stream also feeds trajectory live usage. That reader now expands timed stream records; its regression drives real start/chunk usage frames rather than injecting an incompatible top-level usage object.

Changes are in [Store projection](../../internal/store/session/session_inspector.go), [accepted stream handling](../../internal/store/session/assistant_stream.go), [replacement reconciliation](../../internal/store/session/follow.go), [trajectory usage](../../internal/store/session/trajectory_requests.go), [filtered disclosure state](../../internal/ui/frame/session_inspector_pane.go), [table](../../internal/ui/frame/session_inspector_table.go), [navigation](../../internal/ui/frame/session_inspector_navigation.go) and [object references](../../internal/ui/frame/session_inspector_objects.go). There is no Session format, Host API, credentials or toolkit change. Complete raw accumulation and snapshot projection add work proportional to the retained stream/window; large-stream repaint performance remains unmeasured.

## Same-frame real-window comparison

The [reference probe](../../scripts/inspector-reference-stream.mjs) mounts installed SessionInspectorView, SessionLogModel, MutableSessionEventSource and ClientAssistantStream by exposing their existing factory symbols in memory. It passes normal snapshot/event/assistant-stream frames through the installed controller and publishes decisions exactly as its Session integration does. React subscriptions use the actual observable model and record sources; table interactions use real buttons/filter input. No bundle is written or plugin registered.

The [opt-in native fixture](../../internal/ui/frame/session_inspector_fixture_windows_test.go) feeds the same frames through the real Store follow parser via a [build-tagged helper](../../internal/store/session/session_inspector_fixture.go), then renders production inspector presentation. F8 advances a phase; the [Windows probe](../../scripts/inspector-native-stream.ps1) uses pointer/key messages and UIA ValuePattern to interact and rereads real visible row identities/bounds. The final fixture was rebuilt after navigation review repairs, and the complete nine-state replay passed again. Neither fixture opens a model connection; reference desktop boot uses its separate existing profile at 3083/debugger19444.

| State | Visible rows | Reference capture | Native capture |
| --- | ---: | --- | --- |
| Open interleaved reasoning/text blocks | 8 | [open](evidence/inspector-reference-pass248-open.png) | [open](evidence/inspector-native-pass248-open.png) |
| Manually fold unfinished reasoning | 6 | [manual fold](evidence/inspector-reference-pass248-manual-fold.png) | [manual fold](evidence/inspector-native-pass248-manual-fold.png) |
| Reopen, then receive both block ends and finish | 6 | [closed](evidence/inspector-reference-pass248-closed.png) | [closed](evidence/inspector-native-pass248-closed.png) |
| Reopen completed reasoning | 9 | [reopened](evidence/inspector-reference-pass248-reopened.png) | [reopened](evidence/inspector-native-pass248-reopened.png) |
| Confirm reasoning-delta filter | 6 | [filtered](evidence/inspector-reference-pass248-filtered.png) | [filtered](evidence/inspector-native-pass248-filtered.png) |
| Manually fold filtered reasoning | 4 | [filtered fold](evidence/inspector-reference-pass248-filtered-fold.png) | [filtered fold](evidence/inspector-native-pass248-filtered-fold.png) |
| Clear filter, restore lifecycle defaults | 6 | [cleared](evidence/inspector-reference-pass248-cleared.png) | [cleared](evidence/inspector-native-pass248-cleared.png) |
| Successful settlement retains live + durable | 10 | [settled](evidence/inspector-reference-pass248-settled.png) | [settled](evidence/inspector-native-pass248-settled.png) |
| Step/Turn end retires live, retains durable | 8 | [retired](evidence/inspector-reference-pass248-retired.png) | [retired](evidence/inspector-native-pass248-retired.png) |

Each capture has an adjacent state JSON. The [comparison script](../../scripts/inspector-stream-compare.mjs) and [13-check output](evidence/inspector-pass248-comparison.json) verify four complete projections and all nine actual visible-row sequences/previews. Complete projections contain 8/11/20/13 rows in open/closed/settled/retired phases. The check compares stable key, parent, type, identity, location, timestamp, disclosure, collapsedSummary and decoded original value, independent of JSON object property order. Native-only Chat navigation IDs and source Seq are excluded because the reference's table model stores different navigation metadata. Every reference visible preview is also present in actual native UIA names. Unfolded reasoning rows and collapsed prefix were inspected visually.

Reference captures are 780×600 at scale1; native is 1193×912 physical at Windows150%, with thin external fixture margins. Row heights are 30 CSS/DIP = 45 native physical pixels. These are matching runtime states/data, not pixel equality: native text rasterization/antialiasing and some controls still differ. The fixture omits production window caption/sidebar/Chat, so it does not establish whole-window or pointer-hover picker parity.

## Tests and build

Full affected packages passed before the final navigation-only review repairs:

```text
go test ./internal/store/session ./internal/ui/frame ./internal/ui/chat -count=1
# session 13.787s, frame 27.331s, chat 11.254s
go vet ./internal/store/session ./internal/ui/frame ./internal/ui/chat
```

After final empty-prefix/forward-vs-reverse ownership repairs, affected stream/inspector/Assistant/trajectory tests passed in session8.703s and frame5.138s; vet passed. Chat focused tests had also passed after live IDs/usage changes (6.918s). Final commands:

```text
go test ./internal/store/session ./internal/ui/frame -run 'TestSessionInspector|TestInspector|TestAssistant|TestTrajectory' -count=1
go vet ./internal/store/session ./internal/ui/frame
CGO_ENABLED=0 go test -tags inspectorfixture -c -o .devdata/r104-inspector-fixture.exe ./internal/ui/frame
CGO_ENABLED=0 go build -o whaleisle-pass248.exe ./cmd/whaleisle
```

[Store regressions](../../internal/store/session/session_inspector_stream_test.go) cover interleaved block parents/order, chunk/time/original identity, default closure, two independent reasoning prefixes, UTF-16 bound and untouched raw values, unindexed finish, first actual gap, raw-only invalidation, failed dense suffix retention, success coexistence/matching retirement, failed/interrupted stable aliases, same-generation replacement and eviction-before-republish, compact baseline plus suffix exactly once, stale attempt, abandonment, empty baseline and empty first block ownership, and Turn end without Step end. [Frame regressions](../../internal/ui/frame/session_inspector_stream_test.go) cover real disclosure buttons, state transition invalidation, manual reopened completion, unchanged Turn defaults, filtering over completed blocks, filtered manual folds, same-query reset, closed-ancestor reveal, empty header arrow, live forward/reverse IDs and canonical pick preference, and settled root object references. Existing Round103 cases were migrated to state-tagged choices and passed. [Trajectory regression](../../internal/store/session/trajectory_requests_test.go) uses canonical live usage frames.

Both independent read-only audits completed. The first traced source semantics; the second followed parser/Store/UI/Chat/trajectory consumers and found live navigation, usage-reader, empty-baseline and reverse-pick issues that were repaired and reread. Its final result found no remaining concrete commit blocker. Neither reviewer ran tests or applications.

The final default and pass248 executables both have SHA256 806061A4C747479CF95CB2215F667636D33B11F188EC78600F5D97599AB3E502, length85,042,176 bytes. The production Go file list excludes the opt-in fixture helper; no fixture environment handling enters the shipped build. Native protocol registration still points at the stable default executable. Git whitespace check passed.

Verification-only failures were corrected and excluded from accepted runs: a new test initially treated the void applyFollowFrame helper as a bool, then used the locked production parser for notification assertions; the first Windows loop stopped early by checking an inherited LASTEXITCODE; duplicate visible Index0 labels required exact record identity plus geometry rather than selecting the first named disclosure; UIA PropertyCondition did not resolve the concatenated action name used in the helper, so the driver now filters actual enumerated names/bounds; WaitForInputIdle rejected the console test executable before its GUI was ready, so final orchestration waits800ms then asserts actual UIA rows. An initial go-list template lost quoted arguments in Windows PowerShell; the production exclusion check was rerun using the direct GoFiles field. None of those failures are counted as behavioral passes.

The owned native fixture and independent reference PID17432/backend1068 were closed after evidence. The protected original PID20012/backend16008 remained on3080. Model requests in this round:0. No full production Host desktop replay was repeated; Round103's reproduced exit hang remains unresolved.

## Remaining parity

Fold scroll anchoring, trailing extent, 40px near-tail behavior, 700ms arrival/reveal pulse and reduced motion remain. Full typed tool/Turn/Step/location matching, settled reasoning-part reveal, native Chat group pick fallback, lazy/accessor/cycle object graphs and paging50 are incomplete. Scalar lexical semantics, empty JSON container copy/label behavior, filter combobox/listbox/active-descendant semantics, breadcrumb overflow/stickiness and exact typography/controls still have gaps. Large-stream repaint cost and accumulated raw snapshot work need profiling. Whole-window caption/tabs/sidebar/material differences, shortcut preference I/O, previous toolkit baseline failures and the production shutdown hang remain open. The goal is not complete.
