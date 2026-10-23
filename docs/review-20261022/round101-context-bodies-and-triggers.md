# Round 101 — Context bodies, disclosure geometry and trigger notices

## Changed behavior

Native context disclosures now resolve the same six presentations as the reference: instructions, catalog, snapshot, notice, relay and recall. Each presentation validates its complete source before specializing. Unrecognized or unreadable sources retain the generic text, unknown blocks and source fields. Adjacent text blocks join without an inserted separator; unknown blocks retain their original position. Snapshot sections replace all original content, while catalogs replace text but retain unknown blocks, as the reference does.

Display limits use 20,000 UTF-16 code units independently for each joined text run, snapshot section, source value and JSON preview. Catalogs display at most 200 entries while validating the complete list. Localized truncation suffixes match the reference. Display clipping never edits the retained event. JSON previews follow JavaScript property ordering, duplicate-key replacement, IEEE-754 number formatting and decoded strings; literal backslash-u text stays distinct from actual Unicode line separators. There is no invented Copy JSON action.

The shared disclosure uses the reference's 24px row, font-size delta, leading box, separator geometry, 22px indentation, 141px code-well cap, padding and colors. Enter and Space toggle the keyboard-focused row, and its visible focus ring is enabled. Hover colors use 100ms CSS timing. Opening fades over 100ms; closing keeps an inert paint copy and its layout for the reference's 200ms presence interval. Removed controls are absent from input and accessibility during that hold. Reduced motion bypasses it.

The code well now wraps ordinary text to its available width and permits horizontal scrolling to fixed-width overflowing catalog names and source keys. The small toolkit addition [ScrollOverflowX](../../third_party/mygo/ui/scroll_overflow.go) retains finite layout width and measures descendant overflow, stopping at nested clip/scroll boundaries. Existing ScrollBoth retains its unbounded canvas behavior.

Trigger notices use their source-specific icon/title, the reference's border/background tokens, root hover, ordinary interface font and 12px/18px time/explanation/content. The title uses the font's normal line spacing rather than an invented 1.5 multiplier. Trigger bodies call the notice presentation directly, ignoring source form and source fields. Their body appears/disappears immediately, with the reference's 240px scrolling cap. Date-aware message clocks include hours/minutes on prior dates and use the locale's same-year or other-year template. The same clock is used by user/question/assistant action chrome.

Theme palettes now distinguish border-l1, code-block and trigger background roles. Custom families derive secondary labels with the reference formula and retain platform tertiary/caption values unless explicitly overridden. Underlying hover/code-block overrides feed trigger aliases; explicit trigger overrides win. Child-agent read-only history passes the complete current palette to its context renderer.

Implementation: [body resolver/rendering](../../internal/ui/chat/context_body.go), [shared disclosure](../../internal/ui/chat/context_disclosure.go), [context row](../../internal/ui/chat/context_injection.go), [tool notices](../../internal/ui/chat/context_tools.go), [trigger notice](../../internal/ui/chat/turn_trigger.go), [clock](../../internal/ui/chat/message_clock.go), [semantic palette](../../internal/ui/theme/family_palette.go), [child history caller](../../internal/ui/agents/navigation.go).

## Reference authority

The pinned Electron checkout was read only. No reference source, installed bundle or user session was edited.

- [Context body components](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextBody.tsx) and [CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextBody.module.css).
- [Context row](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextInjectionRow.tsx), [code well CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextInjectionRow.module.css), [shared disclosure](../../reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src/DisclosureRow.tsx) and [hover/geometry CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src/DisclosureRow.module.css).
- [JSON block](../../reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src/markdown/JsonBlock.tsx) and [CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src/markdown/JsonBlock.module.css).
- [Trigger](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/TurnTriggerNodeView.tsx), [trigger CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/TurnTriggerNodeView.module.css) and [message clock](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/message-chrome.ts).
- [Theme derivation](../../reference/electron/vendor/deepseek-harness/packages/client/ui-theme/src/derive.ts), [platform tokens](../../reference/electron/vendor/deepseek-harness/packages/client/ui-theme/src/styles/design-platform.css) and [typography tokens](../../reference/electron/vendor/deepseek-harness/packages/client/ui-theme/src/styles/gradient-shadow-text.css).

## Verification actually run

Final affected package checks passed:

```text
go test ./internal/ui/chat ./internal/ui/theme ./internal/ui/frame ./internal/ui/i18n -count=1
go test ./internal/ui/agents ./internal/ui/chat -count=1
go test ./internal/ui/chat -run '^TestContextDisclosureGeometryKeyboardAndInertExit$' -count=1
go test ./internal/ui/chat -run '^TestContextBodyHorizontalWheel|^TestTurnTrigger' -count=1
go vet ./internal/ui/chat ./internal/ui/theme ./internal/ui/agents
CGO_ENABLED=0 go build -o whaleisle-pass245.exe ./cmd/whaleisle
```

From the toolkit directory, these passed:

```text
go test ./ui -run '^TestScrollOverflowX|^TestScroll|^TestTransition|^TestSelectableContainer|^TestCrossParagraph' -count=1
go vet ./ui
CGO_ENABLED=0 GOOS=windows go build ./ui
CGO_ENABLED=0 GOOS=linux go build ./ui
CGO_ENABLED=0 GOOS=darwin go build ./ui
```

Regressions cover form validation/replacement, exact content keys, primitive and malformed foreign blocks, independent UTF-16 clipping including a split surrogate pair, unchanged raw events, JS numeric/key ordering, ASCII whitespace collapse with NBSP preservation, adjacent JSON margins, shared selection, message-scoped state, keyboard focus/toggling, current body right edge, inert exit and stable layout, trigger hover/body/source separation and date formatting. The real horizontal-wheel integration test proves fixed metadata moves horizontally while the paragraph remains wrapped. Toolkit checks additionally reach X/Y overflow, stop at a nested clip and retain existing ScrollBoth behavior.

The first integrated run found a literal-backslash JSON bug and an obsolete source-whitespace expectation; both were corrected and the affected checks passed afterward. A mistyped test filter once reported no tests; the actual named disclosure regression was then run and passed. The manual native fixture initially called App.Run from a testing goroutine; its opt-in TestMain now runs the window on the main goroutine. These failed attempts are not counted as passing evidence.

The default local executable was updated from the successful pass245 build. No fixture build tag is enabled in that binary. The full toolkit suite was not repeated. The previously recorded allocation and grapheme-ellipsis failures remain separate unresolved work.

## Live Windows evidence and limits

The independent installed Electron profile used backend 3083 and debugger 19444. The existing user GUI on 3080, owned by its original node process, remained untouched. The native component window never connects to a Host. Both component views read the same [fixture events](../../internal/ui/chat/testdata/context-components.json).

The reference probe evaluates a copy of the installed chat factory in that isolated renderer, exposing its internal components only in the evaluation result. It uses the installed React, primitives, dictionary and CSS; source/bundle files are unchanged. It directly mounts the real ContextInjectionRow and TurnTriggerNodeView with fixed props and opens the top-level disclosures. This is an actual desktop component comparison, not a normal conversation/Host-path test. The [reference probe script](../../scripts/context-reference-fixture.mjs) and [opt-in native fixture](../../internal/ui/chat/context_fixture_windows_test.go) preserve the reproduction path.

- [Reference component capture](evidence/context-components-reference-pass245.png): 900×800 CSS pixels at 1× emulated device scale.
- [Native component capture](evidence/context-components-native-pass245.png): 1350×1200 physical pixels at 150% Windows scale, corresponding to 900×800 logical units. The thin black outer edge belongs to the isolated hidden-titlebar fixture window. Keyboard focus remains on the last invoked trigger; the reference capture was made with no matching keyboard-focus state, so its absence of an outline is not an equality claim.
- [Reference measured geometry](evidence/context-components-reference-geometry-pass245.json) and [fixture record](evidence/context-components-fixture-pass245.json).
- [Formal native pass245 main window](evidence/native-main-pass245.png): replayed selected session, filtered main chat, existing missing-credential result, composer and retained inspector event list. No model request was sent in this round. The backend eventually listened on 3081. Initial accessibility enumeration failed once while the tree was changing; a later enumeration completed and a direct PrintWindow capture succeeded.

The first three generic row header positions match after DPI conversion; body indentation and instruction/catalog heights also match. The reference tool body is 54 CSS px; native is the same logical height. The reference trigger header measures about 42.667 CSS px at the fixture's 14px inherited normal font; native measures about 42 DIP after the title correction. Renderer font metrics and fractional border snapping still differ. The screenshot text appearance is not accepted as proof of an actual selected font face; a read-only audit found the current DWrite stack parser and default aliases resolve to Segoe UI, not Times, so no speculative font-resolver patch was made.

## Remaining work

This round does not establish complete product parity. The component fixture is deliberately independent of transcript visibility and the Host event projection; the ordinary context rows shown there are still hidden in normal main chat under the previous round's rules. Live sampling covered instructions, catalog, generic mixed blocks, tools and a schedule trigger. Snapshot/notice/relay/recall, 18px settings, reduced-motion, clipping limits and exit timing have focused native regressions and reference-source review, but not all combinations have real-window captures.

Full-window caption/tab/sidebar placement, the inspector's button-like event rows, overall materials, typography rasterization and some status text still differ. The captured trigger header retains a subpixel metric difference. JSON string decoding uses Go strings: a source JSON lone surrogate is displayed as U+FFFD, whereas JavaScript can preserve that code unit before stringify; the raw event remains intact. Closing this rare preview edge requires an explicit UTF-16 string representation rather than claiming the formatter is universally identical.

The prior intermittent native exits, high repaint cost, synchronous shortcut-preference reads, and toolkit baseline failures remain follow-ups. The successful pass245 replay/capture does not identify their cause or close them. The overall parity goal stays active.
