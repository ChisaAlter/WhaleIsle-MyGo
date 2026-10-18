# Conversation file drops (CP08)

The original ComposerAttachments.tsx:32–34 installs document-level file listeners;
ui-attachment/src/client/drop-events.ts:36–91 accepts only Files transfers and
rejects dropped directories through its entry metadata.

MyGo now registers the main Conversation column, including history and blank
space, as a native file target. The existing composer target remains innermost,
so a drop has one destination. Sidebar/resource panes are outside this target.
A subagent replacing the main conversation does not register the parent target.
The target key contains the selected session ID; queued events from the previous
session cannot be replayed against the next session's draft. The intake also
checks the currently selected ID at render time.

The bridge uses Element.DroppedFiles/FileDragOver, not the typed application
Dock drag route. Real native drag-over shows a pass-through hint over the chat
area; it does not fabricate hover from pointer movements. Modal overlays retain
normal hit-test ownership, so file drops over their backdrop cannot target the
underlying conversation. The existing composer target retains its current UI.

Both drop destinations reuse CE's complete-batch regular-file filtering,
128-file limit, busy-state guard, and OnAttachFiles callback. Existing CP06
local-reference versus upload routing is unchanged. Explicit folder references
remain separate. No clipboard text interpretation or filesystem writes occur.

Validation: one focused go test invocation for composer/frame passed synthetic
file intake, directory/busy rejection, one-time consumption, hover-only behavior,
and a headless real typed Dock drag which did not activate native-file intake.
Native Explorer/Finder drag/drop and modal behavior have source-level evidence,
not physical Windows/macOS acceptance testing. No real clipboard/files were used.
