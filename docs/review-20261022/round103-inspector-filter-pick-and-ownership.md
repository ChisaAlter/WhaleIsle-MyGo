# Round 103 — Inspector filtering, picking and occurrence ownership

## Result

Confirmed filters now obey subsequent manual disclosures. Picking an included record retains the filter; picking an excluded record clears it. Reveal opens only the selected record's ancestors and centers the resulting visible row once. Unrelated folds and the target's own fold remain. Ordinary record selection reveals Chat without changing the inspector's tail-follow policy. These distinctions follow the reference [table](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/InspectorTable.tsx), [type filter](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/type-filter.ts) and [interaction regression](../../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/tests/table-filter.client.spec.tsx).

The original and installed Electron sources remained read only. This round changes the native application and opt-in verification scripts; it does not modify the original 3080 GUI or install a plugin there.

## Behavior changes

- Every filter confirmation canonicalizes optional surrounding stars/whitespace while retaining entered case, clears prior disclosure choices/selection/history, resets vertical scroll and leaves tail-follow. This applies even when confirming the same query or an already-empty query. Horizontal position remains. Clear edits the draft; Cancel/Escape leave confirmed view state unchanged.
- Matching rows plus their contextual ancestors form the filtered hierarchy before disclosure choices are applied. Arrows and sticky branch ownership derive from that retained graph. A matching parent without any retained children has no empty disclosure arrow. The match/total count remains independent of manual folding.
- The pick handoff does not queue a return reveal into Chat. A consumed pick is not replayed on rerender, filter confirmation or details Close; a new request for the same key can center again. A contextual ancestor counts as included by the active filter. Scrolling uses the disclosed, filtered row index, rather than the unfiltered index.
- Object-reference navigation and breadcrumb Back share the same row reveal. Back restores the corresponding table selection and Chat target. References without a row keep the current table selection. Native reference objects remain serialized native data, not the reference's typed runtime object graph.
- Details retain a resized ratio while switching selected records. Closing unmounts that lifetime, so reopening starts at 40%. A selected record removed by settlement/window changes keeps a closable panel with the reference's removed-record message and disabled copy. Initial details layout schedules a second frame when root height is not measured yet; this fixes clicks aimed at provisional geometry.
- Starting a picker cancels the previous Chat target/highlight. Tool-card inspection opens the resource dock persistently and queues only the routed occurrence, with the exact current Host/session-generation binding. It replaces any older inspector gesture. Binding/mode changes, hidden tabs, minimized floats and suspended pages reject stale queued requests. An intentional request queued before its first pane render survives that initial handoff.
- Navigation uses actual resource visibility: dock tabs, visible floating inspectors, and fullscreen siblings all participate correctly. There is one Chat navigation owner. Current Chat Nodes/Groups are shared with every visible same-binding inspector, so changing the chosen pane does not freeze another pane's projection. Beginning an application frame removes old cached projections. End-of-frame rendered/Chat markers cancel gestures when the main Chat or inspector is absent; the newly routed request gets its first render before cancellation. Native store Chat remains the source when main Chat is not displayed.

The implementation is in [pane/filter hierarchy and inspect routing](../../internal/ui/frame/session_inspector_pane.go), [navigation and projection](../../internal/ui/frame/session_inspector_navigation.go), [table](../../internal/ui/frame/session_inspector_table.go), [confirmed filter](../../internal/ui/frame/session_inspector_controls.go), [details](../../internal/ui/frame/session_inspector_details.go), [frame integration](../../internal/ui/frame/frame.go) and [workspace lifecycle](../../internal/ui/frame/workspace_content.go). No toolkit/API, Session persistence format, model path or credentials were changed.

## Same-input real-window evidence

Both sides use the same seven [records](../../internal/ui/frame/testdata/inspector-records.json). The [reference probe](../../scripts/inspector-reference-fixture.mjs) mounts the installed real inspector components by evaluating an in-memory factory copy in the independently owned Electron profile at 3083/debug port 19444. Source hooks are fixed local callbacks, and the picker callback is handed a synthetic identity. The probe does not modify or register the installed bundle.

The [native fixture](../../internal/ui/frame/session_inspector_fixture_windows_test.go) calls production inspector presentation without a Host. With INSPECTOR_NATIVE_FIXTURE_SCENARIO=interaction, F6/F7 supply the same pick handoff identities. The [Windows probe](../../scripts/inspector-native-interaction.ps1) drives actual pointer/key messages and UIA ValuePattern, rereads UIA record identities/SelectionItem state/details geometry and captures the window. This is evidence for the post-pick inspector behavior; it is not a real pointer-hover Chat picker gesture test.

Six sequential states passed on both sides:

| State | Expected visible records and outcome | Reference | Native |
| --- | --- | --- | --- |
| Confirm ` *CUSTOM* ` | turn/start, step/start, future/custom; 1/7; no details | [capture](evidence/inspector-reference-pass247-filtered.png), [state](evidence/inspector-reference-pass247-filtered-state.json) | [capture](evidence/inspector-native-pass247-filtered.png), [state](evidence/inspector-native-pass247-filtered-state.json) |
| Fold step | turn/start, step/start; count remains 1/7 | [capture](evidence/inspector-reference-pass247-folded.png), [state](evidence/inspector-reference-pass247-folded-state.json) | [capture](evidence/inspector-native-pass247-folded.png), [state](evidence/inspector-native-pass247-folded-state.json) |
| Pick included event:4 | ancestor reopened, future/custom selected, filter retained, details open | [capture](evidence/inspector-reference-pass247-picked.png), [state](evidence/inspector-reference-pass247-picked-state.json) | [capture](evidence/inspector-native-pass247-picked.png), [state](evidence/inspector-native-pass247-picked-state.json) |
| Resize to End, close, reopen | same selection; details restart at 40% | [capture](evidence/inspector-reference-pass247-reopened.png), [state](evidence/inspector-reference-pass247-reopened-state.json) | [capture](evidence/inspector-native-pass247-reopened.png), [state](evidence/inspector-native-pass247-reopened-state.json) |
| Confirm same CUSTOM again | all contextual rows open; no selected details; filter retained | [capture](evidence/inspector-reference-pass247-reconfirmed.png), [state](evidence/inspector-reference-pass247-reconfirmed-state.json) | [capture](evidence/inspector-native-pass247-reconfirmed.png), [state](evidence/inspector-native-pass247-reconfirmed-state.json) |
| Pick excluded event:3 | filter cleared; seven records; assistant/message selected | [capture](evidence/inspector-reference-pass247-excluded-pick.png), [state](evidence/inspector-reference-pass247-excluded-pick-state.json) | [capture](evidence/inspector-native-pass247-excluded-pick.png), [state](evidence/inspector-native-pass247-excluded-pick-state.json) |

Reference captures are 780×600 CSS pixels at scale 1. Native captures are 1193×912 physical pixels at 150% Windows scale; the content width is about 780.7 DIP and includes thin fixture frame margins. Native row height is 45 physical pixels = 30 DIP. Reopened detail divider is y=540 physical ≈360 DIP, matching reference y=360. Reference picked-row screenshots capture its timed blue pulse; the native selected row remains its current theme fill. Pulse/motion parity is still open. These images are same-state behavior evidence, not pixel-equality evidence.

Additional [reference initial geometry](evidence/inspector-reference-pass247.json), [detail geometry](evidence/inspector-reference-pass247-details.json) and [actual platform fonts](evidence/inspector-reference-pass247-fonts.json) were captured by the same run. The interaction changes do not alter Round102's font/layout baseline.

## Real production replay and limits

The opt-in nativeacceptance executable also booted the actual isolated Host and production mainView with an existing retained session. Its [capture](evidence/native-main-pass247.png) and [metadata](evidence/native-main-pass247.png.json) show Host/mux/session list/workspace/history ready, five messages, and successful settings/describe (62,434 bytes), pluginManager/listBundles (126,903 bytes) and schedule/catalog (2 bytes) reads. The screenshot's missing-credential result belongs to that pre-existing retained history. Model requests in this invocation: 0.

The readiness evidence was generated before the final visible-projection/lifecycle cleanup. Those final changes have frame regressions and a rebuilt production binary; the readiness capture is not claimed to come from that final exact executable. The real driver stopped its Host after capture but hung during application exit and was cancelled. This remains a readiness/replay pass and an unresolved shutdown failure, not a completed end-to-end desktop pass. No production or default GUI launch was used to change protocol registration; the native whaleisle URL registration still points at the stable default executable.

## Verification

Final affected check passed:

```text
go test ./internal/ui/frame ./internal/ui/chat ./internal/store/session -count=1
# frame 18.981s, chat 4.346s, session 6.835s
go vet ./internal/ui/frame ./internal/ui/chat ./internal/store/session
```

After the final first-render cleanup adjustment, focused inspector/tool routing regressions passed in 6.499s, then frame vet and production build passed:

```text
go test ./internal/ui/frame -run 'TestInspector|TestSessionInspector|TestInspectToolCall|TestPendingPick|TestToolCallInspector|TestWorkspaceResourceMenu' -count=1
go vet ./internal/ui/frame
CGO_ENABLED=0 go build -o whaleisle-pass247.exe ./cmd/whaleisle
```

The [interaction regressions](../../internal/ui/frame/session_inspector_interaction_test.go) cover equal/empty confirmation resets, draft Clear+Cancel, manual filtered folds, filtered arrows/count, included/context/excluded reveal, unrelated and target-fold retention, filtered center index, one-shot/repeated picks, ordinary selection tail behavior, detail ratio/removed records, breadcrumbs, hidden/float/fullscreen navigation, current shared projection, absent-page cleanup, single routed queue, replacing old picker, persisted dock opening, first-render binding handoff and session/mode invalidation. Existing [tool action](../../internal/ui/frame/inspect_tool_call_test.go), [filter](../../internal/ui/frame/session_inspector_controls_test.go), [table](../../internal/ui/frame/session_inspector_table_test.go) and full frame tests also passed.

Opt-in fixture and nativeacceptance binaries compiled. Node syntax check and the reference six-state run exited 0; the Windows six-state run printed PASS for all six states and exited 0. Probe-only failures were corrected before accepting evidence: the first embedded browser filter expression lacked a closing brace; PowerShell reserves the name Filter and Windows PowerShell interpreted a literal middle dot without a BOM incorrectly. The final Windows probe uses Apply-InspectorQuery and a character-code count prefix. A unit scenario initially had an unnamed split node which made find("") resolve the container; it now uses a real resource-tree ID. These were not counted as passing runs.

The final [default executable](../../whaleisle.exe) was copied from pass247. Both SHA256 values are 437025B162F3D7D0D4B96E38B258FF90377C026832D499487C328B9011EED757 (85,014,016 bytes). Fixture/test hooks are absent from the production build. Owned fixture, reference and acceptance processes were closed; protected original PID 20012 / backend PID 16008 at 3080 remained alive. Git whitespace checking passed.

Independent read-only review verified reference semantics and caught the prior-picker handoff defect, which was corrected with a regression. A subsequent narrow read-only review found no concrete blocker in the final rendered/Chat cleanup and first-binding handoff. Reviewers did not execute tests or applications; command evidence above belongs to the lead.

## Remaining parity

The overall goal stays active. Stream lifecycle fold defaults and reset-on-settlement, collapsed summaries, stable live/durable identity, fold anchors/trailing extent, 40px near-tail behavior, 700ms row/Chat highlight and reduced motion still need work. Native serialized Chat objects do not implement typed lazy accessors, cycles, paging50 or complete reference target matching. Reference DOM-hover picker, no-Chat/blur feedback, input/listbox/active-descendant semantics, breadcrumb sticky overflow and some typography/selection colors remain. Full-window shell/material/layout differences, repaint cost, shortcut preference disk reads, previous toolkit baseline failures and the reproduced shutdown hang remain open. This round is not a claim of complete inspector or product parity.
