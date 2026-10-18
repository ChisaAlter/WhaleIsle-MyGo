# CH77 visible child image demand loading

Read-only child history now mounts image seats for declared attachment image
references in the bounded currently loaded message window. Tool references use
the existing validated image-content parser; body/image projection references
must explicitly use attachment:. HTTP/file URLs and arbitrary strings are not
automatically resolved.

Only an image seat intersecting Painter.NativeSurfaceClip starts an OnRead
request. This uses the actual paint clip, including scroll ancestry and modal
blocking, rather than assuming every mounted history row is visible. Requests
reuse the existing child-addressed callback, cache snapshot and explicit retry
buttons. Each navigation occurrence permits at most two outstanding automatic
requests and 32 attempted images total; remaining images retain manual Preview.
Already cached, busy or failed resources never auto-retry. Navigation ownership
keys reset with the existing scoped history view.

Successfully loaded local images render inline and click into the scoped gallery
implemented by CH77. The gallery still navigates only loaded local images. No
credential, fetch or authorization implementation changed.

One focused fixture passed: TestChildVisibleImageRequests (0.340s). A real
headless Scroll viewport requested only the visible declared attachment, skipped
a remote URL and offscreen image, deduplicated paint passes, retained failures
for manual retry, and enforced concurrency/total budgets. Empty modal-blocked
clip was checked directly. No real Host or remote image was accessed; native
Windows/macOS scroll/clip acceptance was not performed.
