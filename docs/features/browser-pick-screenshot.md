# Browser preview: element selection and screenshots

The native browser toolbar now exposes **选择元素**, **网页截图**, **取消**, and **显示截图**. Both the full browser panel and the docked workspace browser use the same controller.

## Element selection

A user gesture captures the initiating session, composer draft, native browser tab, application window, and Host generation. A temporary transparent picker highlights the element under the pointer. Clicking appends a bounded annotation to that session's current ordinary draft; Escape, Cancel, navigation, tab changes, closing/hiding the preview, a different session/workspace, or Host replacement invalidates the operation. Finishing an old asynchronous result cannot clear a newer operation or write to another draft.

Annotations include URL, a CSS selector, element tag, role/label, bounded text and viewport bounds. They are explicitly marked as page content for review. Input/select/textarea values, editable content and executable HTML are excluded. A selected cross-origin iframe is represented by its frame element; its private document is never traversed. Open shadow roots are inspected up to a bounded depth. A session must already be selected; the feature does not create or select one, upload attachments, focus the composer or send a message. In-progress message editing, sending and IME composition prevent insertion.

The picker runs only in the ordinary browser preview. Local PDF/HTML snapshots and authenticated plugin gateways keep their separate policies. Native page bindings remain disabled. Selection is an untrusted DOM annotation, not a verified command or application capability.

## Screenshots

The screenshot button captures the current visible browser viewport, saves a PNG under the application's user cache folder at `WhaleIsle/browser-screenshots`, and reveals the exact generated file in the system file manager. A subsequent **显示截图** action reveals it again. The path and filename are generated locally; page text cannot choose a path. Nothing is automatically attached or uploaded. The cache may be cleared by the operating system; move screenshots elsewhere when they should be retained.

Capture has a 15-second caller deadline, one outstanding native capture per Host, a 4096-DIP per-axis viewport limit, a 16-megapixel output limit and a 32-MiB PNG limit. Windows WebView capture reads only the logical stream bytes, bounded to 64 MiB before copying, rather than copying global-memory allocation padding. Cancelling or replacing the initiating context prevents saving/delivery; any file created before a stale result is discarded. A native capture already in progress is allowed to finish, while repeat captures remain blocked until it does.

## Rendering lifecycle

The owning main frame calls BeginFrame/EndFrame. The controller marks itself rendered only in the actual browser view. This preserves an active docked picker across paints while hiding and cancelling the native browser when the right pane is closed, too narrow, or another full panel is drawn.

## Verification status

Behavior and privacy-boundary tests have been added. No runtime, device, acceptance, compilation or test execution was performed for this change: the cloud shell was unavailable and the requested work excluded device/acceptance testing. Code integration must preserve the main frame hooks and use the action-aware controller factory at every construction site.
