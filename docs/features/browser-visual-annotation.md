# B17: native browser visual annotation

## Original contract and source

Ported from Host commit `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`, `src/main/preview-guest-preload.js` (`startAnnotation`, originally lines 330–1147), `preview-pick-helpers.js`, and `preview-pick-label.js`. This restores the original editor rather than representing annotation tools as count fields.

The ordinary web browser toolbar's **可视批注** opens a shadow-root editor with:

- Select and Shift multi-select, labeled outlines, marquee element selection and empty-area regions.
- Draw freehand strokes and Erase selected elements, regions, or strokes.
- Comment composer; expandable, draggable editor with font family/size/weight/line height, text/background/border colors, opacity, radius, border width, width/height with aspect lock, padding, margin, and gap.
- Local style preview, restoring original inline values and priorities on remove/cancel/completion.
- Review/Enter prepares the annotation and viewport-clamped cropped PNG, retaining outlines/strokes in the image and hiding editor chrome. Ctrl/Meta+Enter also only prepares; no send route exists.

The native toolbar then exposes **查看批注截图**, **复制批注**, **批注和截图加入草稿**, and **取消**. Only the native append button inserts the annotation text and hands the locally generated PNG to the existing composer attachment-preparation/upload flow. It never submits a prompt, focuses a composer, selects a session, or creates a session. Text and image remain in the initiating draft for review and normal send.

## Boundaries and lifecycle

The original Electron IPC was removed. Host-initiated `EvalContext` polls a bounded JSON string under a random operation key; all renderer content is treated as untrusted data. There are no new Host RPC routes, file access capabilities, native page bindings, external fetches, page-selected file paths, or page-selected upload destinations. A page cannot trigger attachment/upload merely by completing/spoofing the renderer state: native confirmation remains required.

The existing capture path enforces viewport/pixel/PNG limits. The crop is intersected with decoded PNG bounds, with CSS viewport-to-image scaling. Collections, strings, coordinates, and serialized results have explicit limits. Form values, editable content, and executable HTML are not collected. A transparent shield prevents interactions with embedded cross-origin documents; their iframe elements can be annotated without traversing their contents. Style controls reject URL/expression syntax.

Session/draft/window/Host ownership remains captured by `browserActionTarget`; navigation identity is retained for staged results too. Cancellation, stale ownership, navigation, hide, or tab change discards uncommitted captures and restores temporary styles. Once attached, the composer owns the image through its existing attachment lifecycle.

## Verification status

Source review and gofmt only. No tests, builds, GUI launches, Host launches, screenshot acceptance, or device acceptance performed in this module pass. The root-owned focused synthetic check remains pending. Suggested checks: bounded JSON decode, invalid geometry/collection rejection, crop clamping and CSS/image scale, stale navigation invalidation, cancelled temporary-file cleanup, and explicit native confirmation before attachment.

## Narrow offline regression fixture

The same-package `internal/desktop/webview/annotation_test.go` and `internal/ui/browserpreview/annotation_test.go` directly exercise production validation/crop functions and the final native-confirmation callback boundary. They check camelCase payload decoding, size/coordinate rejection, CSS-to-image crop scale and edge clamping, stale owner/navigation predicates, one-shot native attachment, and uncommitted-file cleanup. Narrow source checks bind renderer staging and native confirmation to the integration source. No copied or reconstructed implementation is used. Tests were prepared but not executed by the module worker.

Focused command: `go test ./internal/desktop/webview ./internal/ui/browserpreview -run '^TestB17' -count=1`.


Capture order is intentional: renderer hides editor/toolbar and stages data without teardown; native capture finishes and cropping uses those pixels; deferred cleanup then restores styles. Capture-pending Escape is blocked inside the renderer so it cannot restore CSS between result polling and capture; the native Cancel action still cancels the context and runs cleanup. Errors use the same deferred cleanup.
