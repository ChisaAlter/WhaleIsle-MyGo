# Round 98 — Git caption review survives the conversation frame

## Confirmed problem

The top caption's enabled Commit / Commit and push / Push / Publish repository actions opened a Workbench review on the main-state adapter. `frame.View` had already copied that adapter into the active resource pane. At the end of the frame, `finishDiffProjection` published the pane's earlier copy over the main Workbench, losing the newly opened review. MyGo settles a consumed click with another build before painting, so the modal could disappear without a visible flash.

The earlier Round 95–96 conclusion that the observed no-op necessarily meant a disabled control was incorrect. Live inspection of pass241 showed Push was enabled and advertised InvokePattern. Focus-ring and opacity changes did not repair this state overwrite.

## Implementation

- In the conversation frame's projection adapter, wrap the existing caption producer and capture its Workbench result into the occurrence that owned the adapter before the caption was rendered. Do so before resource panes render and before final writeback. Preserve the original session/Host admission checks and callbacks.
- Keep global NavigationShell behavior unchanged; it does not run the conversation's pane projection.
- Put disabled-action reasons on enabled presentation wrappers around the primary Git action and menu rows. MyGo does not show tooltips owned by disabled elements.
- Remove the additional `.Opacity(.45)` introduced in Round 96. MyGo already multiplies disabled painting by `.48`; the extra multiplier made the controls approximately `.216` opaque.
- Update stale frame tests/snapshot lookup to the existing account entry's current accessible label. No account product behavior changed.

## Regression evidence

New `TestWorkspaceCaptionReviewSurvivesFrameProjection` exercises the complete conversation `frame.View` and the global `NavigationShell` control. Four actions × six page/dock combinations cover closed dock, visible diff and split diff scenarios, with three extra frames after opening and another frame after Close.

Before the fix, all 12 conversation scenarios lost their review while global controls remained open. After the fix, all scenarios pass. The split conversation assertions also preserve the other pane's private message draft and ensure it acquires no caption review or remote/publish action. Fixtures only initialize temporary repositories and read their status; opening/closing never starts an operation, prepares a commit/push plan or contacts model endpoints.

`TestWorkspaceTrailingDisabledReasonsRemainVisibleOnHover` verifies the disabled primary action and disabled Push menu item show their actual reason after pointer dwell and dispatch no Git callback. The clock is advanced in memory.

Checks passed:

```text
go test ./internal/ui/frame ./internal/ui/chrome ./internal/ui/diffsurface -count=1
go test ./cmd/whaleisle -run '^TestWorkspaceCaption' -count=1
go build -o whaleisle-pass242.exe ./cmd/whaleisle
git diff --check
```

The additional split isolation assertions were then rerun successfully with:

```text
go test ./cmd/whaleisle -run '^TestWorkspaceCaptionReviewSurvivesFrameProjection$' -count=1
```

## Live verification

- Build: pass242, same isolated `.devdata-clean` profile, PID 30180 at verification.
- Queried ControlType.Button and IsEnabled, then invoked the enabled caption action through InvokePattern rather than matching a text child or guessing screen coordinates.
- The Git review remained visible across later UIA inspection. Its Close button dismissed it and subsequent inspection found no review.
- No Prepare review / Confirm / Push / Commit execution action was invoked.
- Screenshot: [Open review](evidence/git-caption-review-pass242.png).
- Screenshot: [Review closed](evidence/git-caption-review-closed-pass242.png).

The clean test repository had no selected file changes at review load, so the existing Workbench presented the remote-action review. This verifies review lifetime; it does not establish exact Electron confirmation wording or layout parity.

## Still open

The custom Composer draft exposes ValuePattern without a writable accessibility value action. Therefore the earlier UIA SetValue failure alone does not prove that normal keyboard typing is read-only. Round 97 did not establish a sent prompt or a model response. A real chat roundtrip and the custom editor's accessibility support remain to be verified.

Exact Electron default-branch confirmation flow, complete same-state visual comparisons, and the remaining application-wide function/motion coverage are still unfinished. The overarching parity goal remains active.
