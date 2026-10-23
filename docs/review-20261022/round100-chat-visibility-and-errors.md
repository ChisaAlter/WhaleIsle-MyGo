# Round 100 — Main-chat visibility, tool notices and terminal errors

## Reference-backed changes

The assembled Electron chat hides ordinary context injections, system prompts and permission-command nodes. A context containing any tool addition/removal stays visible. A nonhuman message that actually woke the current turn has a separate source-specific notice. The previous native chat rendered every context block as a generic disclosure, including runtime/time inputs.

The native main-chat projection now follows those visibility rules. The full store snapshot, inspector inputs, raw events, recalled-session labels and invoked-skill facts remain available. Filtering happens only when deriving the main transcript order, before its selected detail level is applied.

The waking flag is derived from the retained event window, using the inbox's current claim at each message. It includes the reference's idle first-step steer case and does not infer waking from source kind, the first context in a turn or an identity that was claimed in the past. The fold models canceled/requeued identities, separate next-turn/next-step queues, omitted removedCount, JavaScript splice clamping/truncation, partial windows, explicit coordinates and binding changes. Its cache belongs to the retained inspector binding and invalidates when event bytes change, including a corrected event at the same sequence number. Explicit sequence zero is retained; an absent sequence is not treated as zero.

All-tool contexts now show a named, noninteractive singleton notice or a multi-tool disclosure with localized counts and actual name lists. The implementation uses the reference SVG paths and row/body geometry, with 100 ms opacity, a 200 ms exit-presence hold and reduced-motion bypass. Mixed content stays on the generic context path.

The previous native missing-key/auth/quota cards were an invention: the reference uses an inline error row with a status dot, title, reason and optional separate code. Native terminal errors now use that form, including the signed-out hook's task-stopped notice. Stable AUTH/account/quota codes select the reference's localized copy; an unrecognized or missing code keeps the actual provider reason. Raw errors are retained. The duplicate selected-session shell feedback from live api-session/error was removed; per-session ContactError remains stored.

Implementation is in [chat visibility](../../internal/ui/chat/chat_visibility.go), [tool notices](../../internal/ui/chat/context_tools.go), [turn-trigger notices](../../internal/ui/chat/turn_trigger.go), [claim projection](../../internal/store/session/turn_triggers.go) and [failure rows](../../internal/ui/chat/failure_notice.go).

## Reference authority

Read-only sources in the pinned Electron checkout:

- [Assembled visibility contract](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/contract/chat-visibility.ts).
- [Message waking classification](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/message.ts) and [inbox claim/splice semantics](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/inbox.ts).
- [MessageItem terminal error row](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/MessageItem.tsx), [error CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/MessageItem.module.css), and [stable error projection](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/event-projection.ts).
- [Tool/context row](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextInjectionRow.tsx), [row CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/ContextInjectionRow.module.css), [turn-trigger card](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/TurnTriggerNodeView.tsx), and [trigger CSS](../../reference/electron/vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/TurnTriggerNodeView.module.css).

## Verification

Final integrated source, after removing all build-only diagnostic code, passed:

```text
go test ./internal/store/session ./internal/ui/chat ./internal/ui/frame ./internal/ui/i18n -count=1
go test ./cmd/whaleisle -run '^TestWorkspaceCaption|^TestMainNavigation|^TestConnectionUnavailable' -count=1
CGO_ENABLED=0 go build -o whaleisle-pass244.exe ./cmd/whaleisle
git diff --check
```

The default whaleisle executable was updated from that successful pass244 build. No diagnostic build tag is used by the final binary.

Regressions exercise all four main-chat detail levels, unchanged full snapshots and inspector nodes, retained recall/skill facts, stable-code failure copy, uncoded reasons, sign-out versus ordinary completion, narrow/wide inline errors, tool singleton/multi/mixed content, geometry, expand/collapse and motion. Claim-fold tests cover replay, live and prepended history, requeue/cancellation, human claims, message-time ownership, coordinate scoping, sequence zero and changed retained claims. The full-frame resource-menu regression selects a session and reaches the inspector controls/list/object path; its store has no raw records, so the live detail verification below is separate evidence.

## Live Windows evidence

An independent installed Electron profile used port 3083/debugger 19444. The existing user application on port 3080 was not stopped or altered. Both the reference and native session use the same restricted prompt and deepseek-official/deepseek-flash missing-credential outcome. Native replays the prompt sent in Round 99; the reference prompt was sent once in this round. Their timestamps differ. No credentials were read, copied or changed, and there is no successful model reply.

- [Installed reference](evidence/missing-key-reference-pass244.png): ordinary runtime/time context is absent, with an inline provider reason and MISSING_CREDENTIAL code.
- [Native pass244](evidence/missing-key-native-pass244.png): the same prompt/error semantics, no ordinary context rows, no duplicate shell error and no invented failure-card actions. The reference capture is 1600×900 CSS pixels; native is 2400×1350 physical pixels at 150% DPI, corresponding to 1600×900 logical units. These are behavior comparisons, not a claim of whole-window pixel equality.
- [Native retained inspector detail](evidence/retained-context-inspector-pass244.png): the main chat remains filtered while the right resource dock shows original raw event rows. Selecting user/message sequence 11 opens the original time-context JSON. The screenshot is from a pass244-equivalent diagnostic build whose only extra source periodically writes Go stacks; that source was removed before the final integrated build. The earlier screenshot incorrectly labelled as inspector evidence was replaced with this inspected capture.

### Native verification limits and diagnostics

The first live inspector attempt lost visible dock content and its original native process later disappeared. No matching Windows crash event or panic stack was recovered. Subsequent diagnostics exposed two automation issues: DPI-unaware resize changed the logical viewport, and resizing before UIA invocation invalidated retained accessibility elements. Reading a zero-bounds startup window also produced stale startup text. Reacquiring the exact current button without resizing allowed opening the retained inspector and selecting its real event detail.

The final managed diagnostic process remained running and responsive through raw-row enumeration, event selection and screenshot capture; its captured stderr was empty. This proves that path can work with the selected real session, but does not explain the earlier process exits. Repeated stack samples show UI progress and many samples in Direct3D Present/repaint; they do not establish a deadlocked inspector or a single blocked GPU call. No speculative inspector-loop or graphics patch is included in this round.

## Remaining parity work

This round does not establish complete Whale Isle parity. The same-state screenshots still differ in caption/tab arrangement, sidebar content, transcript placement and overall surface details. The reference also has a separate assistant status text, “处理失败”, absent from this native capture.

Generic mixed-context disclosure is still the older native layout. Trigger mixed-content rendering, the reference's 20,000 UTF-16-character budget/localized truncation suffix, detailed semantic color tokens and all trigger motion states still need exact alignment. The current trigger body clips by native bytes and has a simpler fallback for mixed content. Original durable content remains intact in the inspector.

Separate follow-up candidates from the read-only inspector review are fractional-sequence row-key collision/cycle handling and repeated object-graph allocation for large histories. Neither was shown to cause the selected integer-sequence session's earlier exits. The intermittent native exit and high repaint cost remain diagnostic follow-ups, not solved defects or grounds for an overall completion claim. Sample 09 also confirms synchronous shortcut-preference file reads during menu synchronization on the UI thread; consolidating that accepted preference snapshot is a separate follow-up, not an explanation for the earlier exits.
