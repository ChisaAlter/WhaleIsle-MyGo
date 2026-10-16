# H02 — Dynamic HTML preview and non-Windows runtime gap

Status: **open; not feature-equivalent**. Source comparison on 2026-10-16.
This document records the remaining implementation boundary; it does not enable
a renderer or change Host capabilities.

## Pinned reference

Original: ChisaAlter/WhaleIsle at
`e9f584be6d3b491ce777bb6823f30bc2bbc6f195`.

- [src/main/html-preview.js](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/src/main/html-preview.js)
  accepts prepared HTML, width 240–1600 and light/dark appearance. It creates a
  fresh in-memory Electron/Chromium partition with no preload or Node access,
  blocks requests other than data: and about:blank, denies navigation, popups,
  permissions and downloads, runs the document, waits for fonts/images and
  layout, then returns PNG base64, width, full content height, captured height
  (at most 4000), and bounded console messages. Cancellation and a 20-second
  deadline destroy the capture window. The implementation has no Windows-only
  branch; this is source-contract evidence, not proof of execution on every OS.
- [generative-html-preview-frame.web.tsx](https://github.com/ChisaAlter/WhaleIsle/blob/e9f584be6d3b491ce777bb6823f30bc2bbc6f195/vendor/chisacode-remote/packages/app/src/components/generative-html-preview-frame.web.tsx)
  renders an inline iframe using srcDoc and sandbox="allow-scripts".
  An independent desktop window is therefore not the same presentation surface.

MyGo baseline: `82d2662a88a9a13ac62016b8f36d274080ccaa97`.

## Implemented Windows capture path

The independent helper now has a bounded capture mode, in addition to its
existing explicit interactive window. CaptureExperimental sends the same
already-authorized snapshot through the existing AppContainer handshake.
The helper uses MyGo Window.CapturePage and the existing HTML measurement,
4000px crop, PNG normalization and console collection code. It keeps its
browser alive until the parent has checked the process tree and received the
result; the parent closes the whole job before returning the capture.

The IPC returns PNG base64, width, full content height, captured height and
console messages. Its size is bounded, and the receiver checks nonce, phase,
dimensions and PNG header against metadata. The overall capture has a
20-second deadline. No Host credentials, filesystem paths or URLs are sent.

Production callers are connected:
- The authenticated /desktop/html-preview POST handler keeps existing input
  validation, then invokes the separate helper renderer.
- The published HTML image-preview action reuses its authorized durable read
  and invokes that same helper renderer.
- Interactive preview still opens its separate window; it is not replaced by
  the screenshot mode.

No SDK changes were needed. The existing isolation/job policy, static fallback
and global Readiness/Probe semantics are unchanged. A successful helper call
is not silently promoted to global in-process rendering availability. The Host
visual-generation readiness gate therefore still requires its existing runtime
verification; the new capture endpoint alone does not declare that feature ready.

## Exact remaining gaps

1. **No non-Windows process backend.**
   `previewprocess/launcher_other.go` returns
   "interactive AppContainer preview requires Windows amd64 or arm64".
   `cmd/whaleisle-preview/main_other.go` prints the same reason and exits 1.
   The HTML-tab script action is disabled outside supported Windows targets;
   published interactive-reply callbacks are not offered there.
2. **No non-Windows isolated browser primitive.**
   `third_party/mygo/window.go` rejects IsolatedPreview on non-Windows before
   native construction. `htmlpreview/capture_other.go` returns unavailable
   for both capture and probing. Ordinary WebKitGTK/WKWebView windows or their
   screenshot APIs cannot satisfy the missing isolation contract by themselves.
3. **Windows runtime parity is still unverified.** The current helper requires
   the complete WebView2 process collection to preserve its expected SID and
   zero capabilities. No Windows runtime was available in this task, and a
   Linux unit test cannot show that a supported WebView2 runtime meets that
   requirement or renders successfully.
4. **The interaction surface differs.** The current helper is a separate
   one-document window with opening appearance; it does not implement the
   original inline interactive surface or live appearance updates.

## Remaining platform work

There is no existing cross-platform isolated helper backend. Closing the
non-Windows gap requires platform-specific isolated browser/process support,
not a switch in a controller. Ordinary application webviews, additional Host
RPCs, arbitrary workspace paths, network access or native execution must not
be substituted for that missing backend.

## Focused verification

New fixtures passed in one targeted invocation:

```
go test ./internal/desktop/previewprocess ./internal/desktop/htmlpreview -run 'TestCaptureResultIPC|TestCaptureModeAndOptIn|TestProcessRendererRoute' -count=1
```

Coverage: result/console round trip, bad nonce/width/PNG/truncated or oversized
result rejection, capture mode round trip, disabled/cancelled requests, handler
input validation, and unchanged readiness. Existing package fixtures had also
passed before the implementation; they were not rerun as a full suite.

Windows amd64 helper and main application cross-compilation use
-buildvcs=false because this recovered source checkout has no Git metadata.
These are build checks, not Windows execution. No native browser/AppContainer
runtime or non-Windows dynamic runtime was exercised. H02 remains partially
open for the platform and runtime gaps above.
