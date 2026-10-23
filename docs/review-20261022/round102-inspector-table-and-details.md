# Round 102 — Session inspector table, filter and raw details

## Changed behavior

The native inspector now uses the reference's compact single toolbar and fixed-height table instead of three control strips and large event buttons. The toolbar contains a presentation select, crosshair picker, total or matching/total count, conditional earlier-history action and Follow Latest. Raw Log has Index / Type / T/S / Time (UTC) / Data columns. The first four columns are 64 / 190 / 65 / 100 DIP, the remaining column fills available width, header and data rows are 30 DIP, and a shared horizontal viewport preserves the 660 DIP minimum in narrow docks. Chat Group omits the UTC column. Epoch zero and sequence zero display correctly; stream children show their own chunk index and inherit their root location. Native Host ownership, the retained log window, disabled capture gate, actual Message projection and raw event bytes remain authoritative.

Type filtering now opens from the 13px funnel in the Type header, in a 290px popup with compact input, candidates and Clear / Cancel / Apply controls. Existing confirmed-vs-draft behavior, keyboard candidate navigation, IME guard and Escape cancellation remain. Matching uses the reference's surrounding-star and whitespace normalization. Filtering/mode changes reset scroll state. Mode changes also reset details ratio, navigation and tree state.

The table indents by 20 DIP per level and displays branch disclosures, reference-style compact data previews, contextual ancestor coloring, selection and nested fixed ancestor rows. The fixed band belongs to the vertical scroll owner, so wheel/PageDown over its controls continue scrolling. The original flow position of a pinned record contains only an invisible 30 DIP placeholder, avoiding duplicate live record controls. This is a native rendering approach, not a claim to reproduce every reference folding/anchoring behavior.

Details occupy 40% of the complete inspector height by default, with the 5 DIP splitter and existing 15–80% resize limits. The raw-data toolbar and scroll body are compact; the native raw-copy action is retained as a small secondary icon. JSON objects/arrays use Object / Array(n), plain field/index names, integer-key-first then insertion key ordering, and 16 DIP indentation. Containers initially expand, including empty containers. Key/value colors use the resolved palette. Raw scalar precision is retained, including integers beyond JavaScript's exact range; this deliberately remains different from JSON.parse-derived values.

On this Windows installation the actual Electron inspector's monospace face was measured through CSS.getPlatformFontsForNode as NSimSun; the selector uses Segoe UI. The native inspector selects NSimSun with monospace fallback on Windows only. This is confined to this surface. Font rasterization is still renderer-dependent.

Implementation: [table and toolbar](../../internal/ui/frame/session_inspector_table.go), [preview formatter](../../internal/ui/frame/session_inspector_format.go), [pane integration](../../internal/ui/frame/session_inspector_pane.go), [filter/splitter](../../internal/ui/frame/session_inspector_controls.go), [details](../../internal/ui/frame/session_inspector_details.go), [JSON tree](../../internal/ui/frame/session_inspector_tree.go), and [retained record metadata](../../internal/store/session/session_inspector.go).

## Accessibility corrections found in review

Independent read-only review caught two regressions before commit: fixed rows originally duplicated controls and sat outside Y-scroll ancestry; the Type-header filter was hidden by the toolkit's leaf ColumnHeader semantics. Both were corrected. A follow-up review found that exposed selected-row state did not implement SelectionItem.Select. Record rows now handle click, Enter and Space through the same selection function; real Windows UIA Select on an initially unselected record changes IsSelected and opens Raw data.

The small [toolkit accessibility correction](../../third_party/mygo/ui/access.go) treats ColumnHeader as a container while retaining its text-derived name, and marks explicitly checked Row elements selectable. [Toolkit regressions](../../third_party/mygo/ui/access_test.go) verify header button traversal/action and custom-row state. Generic ListItem wrappers, selection-container relationships and full reference table accessibility structure remain follow-up work.

## Reference and actual measurements

Reference checkout and installed bundle were read only. The independent Electron profile used 3083/debugger 19444. The original user GUI/backend stayed at 3080 with PID 16008 under PID 20012 and was never stopped. The reference script evaluates an in-memory copy of the installed inspector factory and directly mounts its real components with local fixture callbacks; it does not register a plugin or change the bundle. The native component fixture connects to no Host.

Both component windows read the same seven [fixture records](../../internal/ui/frame/testdata/inspector-records.json). Reproduction uses the [Electron probe](../../scripts/inspector-reference-fixture.mjs) and [opt-in native desktop fixture](../../internal/ui/frame/session_inspector_fixture_windows_test.go).

- [Reference initial capture](evidence/inspector-reference-pass246.png), [details capture](evidence/inspector-reference-pass246-details.png), [initial geometry](evidence/inspector-reference-pass246.json), [details geometry](evidence/inspector-reference-pass246-details.json), [actual font faces](evidence/inspector-reference-pass246-fonts.json).
- [Native initial capture](evidence/inspector-native-pass246.png), [details capture](evidence/inspector-native-pass246-details.png), [filter capture](evidence/inspector-native-pass246-filter.png). Native images are physical pixels at Windows 150% scale. The final fixture outer window is 1193×912 physical pixels; its content measures about 780.7×600 logical units. Thin black outer margins are fixture frame area. Normalize the actual content rectangle and scale before comparing with the 780×600 CSS reference.
- UIA measured native header/data height 45 physical = 30 DIP; first column widths 96 / 285 / 97 / 150 physical correspond to 64 / 190 / approximately 65 / 100 DIP with rounding. Native toolbar is 34 DIP versus reference 34.667 CSS px. Native details separator begins at logical y=360; reference details also begin at y=360. Remaining fractional border/font metrics are not counted as exact equality.
- [Real production mainView replay](evidence/native-main-pass246.png) and [readiness metadata](evidence/native-main-pass246.png.json): selected retained session has history ready, five projected messages and 21 inspector records. Host, mux, session list and workspace are ready. settings/describe, pluginManager/listBundles and schedule/catalog reads passed. Model requests: 0. The displayed missing-credential failure belongs to the pre-existing retained session, not a new request.

The production replay uses the existing nativeacceptance entry and actual isolated Host, not a component mock. It generated the readiness image/metadata and stopped its Host, but hung during final application exit and was cancelled. It is evidence of readiness/replay, not a green end-to-end shutdown test. The capture precedes the final action-only Row change; that final change has headless regressions plus real Windows UIA action verification. Earlier transient captures of the launcher or closed popup were replaced by the actual requested states. Unquoted PowerShell test flags were once split into incorrect arguments; only the quoted-argument replay produced the final readiness evidence.

## Verification run

Passed affected application checks:

```text
go test ./internal/ui/frame ./internal/store/session ./internal/ui/chat -count=1
go test ./internal/ui/frame -run 'TestInspector|TestSessionInspector|TestWorkspaceResourceMenuOpensRetainedInspectorThroughFrame' -count=1
go vet ./internal/ui/frame ./internal/store/session
go test -tags inspectorfixture -c -o .devdata/r102-inspector-fixture.exe ./internal/ui/frame
CGO_ENABLED=0 go build -o whaleisle-pass246.exe ./cmd/whaleisle
git diff --check
```

The full affected run passed frame 32.009s / session 14.731s / chat 9.613s. The final action-only change passed the focused inspector/frame route run in 5.342s and vet/build afterward. Tests cover ordered previews, deltas/depth limits, epoch/identity/location, columns and row height, row/details actions, 40% split, 5px rail, narrow horizontal reach, filter ancestor context, mode resets, and 82-row fixed-band wheel/PageDown with a single record-control instance.

From the toolkit directory, these passed:

```text
go test ./ui -run 'TestAccessibilityOfColumnHeaderControls|TestAccessibilityOfCheckedCustomRow|TestAccessibilityOfTables|TestRole|TestAccessibilityTree' -count=1
go vet ./ui
CGO_ENABLED=0 GOOS=windows go build ./ui
CGO_ENABLED=0 GOOS=linux go build ./ui
CGO_ENABLED=0 GOOS=darwin go build ./ui
```

Actual Windows operations verified filter accessibility/discovery and popup, record details/Close, unique accessible record count, selected SelectionItem state, and SelectionItem.Select action on an initially unselected record. A UIA action returning successfully was not treated as a state assertion; the final Select probe re-read IsSelected and Raw data.

The default [executable](../../whaleisle.exe) was updated from the final pass246 build; SHA256 is 2BE7FCF7ABA6F2AE21C6EEDB99AA5A6834915672B1932D491F109333E58A24E7. Fixture tags are absent. A non-isolated review launch temporarily registered the native whaleisle protocol to the numbered candidate; this native-owned registration now points at the stable default executable. The original dsh GUI/backend remained running. Owned review processes were closed afterward.

## Remaining parity work

This round is partial inspector parity and does not complete the product goal. Active filtering still forces matching descendants open, so manual folding of contextual branches does not yet follow the reference. Stream lifecycle fold defaults/automatic close, collapsed reasoning summaries, live-to-durable identity continuity, folding anchors/trailing space, the near-tail threshold, 700ms row/navigation highlight and reduced-motion behavior need implementation. Native Chat Group uses actual native Message JSON and relationships, not the reference runtime object's typed lazy/accessor/cycle/paging model. Breadcrumb sticky/reveal behavior and JSON empty-container Empty children/scalar canonicalization differ. Picker gesture remains the native explicit choose action, not reference DOM hover masking.

The inspector popup still uses toolkit theme colors, and small geometry/typography metrics remain. Full-window title/tab/sidebar/material matching, high repaint cost, shortcut-preference disk reads, intermittent exits, this round's exit hang and prior toolkit allocation/grapheme-ellipsis baseline failures remain open. No full toolkit or full repository green claim is made. The overall goal stays active.
