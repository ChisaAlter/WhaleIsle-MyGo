# B04: native Browser recording on macOS and Linux

## Implemented path

The Windows-only blockers were the encoder implementation build tag, the UI's OS gate, and the SDK's Windows-only isolated helper. Native Browser PNG capture already exists (`WKWebView.takeSnapshot` and WebKitGTK snapshot); it was not replaced.

Only an explicit **Record WebM** click creates a new hidden helper. `PlatformAvailable` advertises that an attempt is implemented, not that a codec exists. The helper probes MediaRecorder WebM support and canvas capture; its first frame checks manual `requestFrame`, and failures return through the existing visible action error. There is no automatic recording or startup codec probe.

The existing serialized PNG → canvas → MediaRecorder path now compiles on Windows/macOS/Linux amd64/arm64. Stop ends capture and finishes encoding. The result stays in bounded memory until the user chooses Export; the existing native save dialog and exclusive file creation remain. Navigation, switching tabs, window close and cancellation discard the pending result. Limits remain 8 fps, 1280×720 output, 2 minutes, 64 MiB, no audio. This records the current webpage viewport, not the desktop or microphone.

## Restricted blank helper

On non-Windows, `IsolatedPreview` accepts only the empty encoder case. All LocalPDF/LocalPDFViewer/LocalHTML/LocalMedia URLs, native child configurations and UI surfaces remain unsupported. This does **not** enable arbitrary HTML preview on these platforms.

- macOS uses public `WKWebsiteDataStore.nonPersistentDataStore` and verifies `isPersistent == false`.
- Linux uses a separate `webkit_web_context_new_ephemeral` and verifies `webkit_web_context_is_ephemeral`; missing APIs fail creation.
- Both load a single application-owned document at `about:blank`, with a preinstalled CSP denying resources, connections, scripts from the document, frames, workers, objects and forms. The native application evaluates only the existing fixed encoder code; inputs are bounded PNG bytes, never source-page HTML, URLs or scripts.
- No Host RPC handler, user/preload scripts, titlebar bridge or custom URL scheme is registered. File/HTML loads, outside navigation, new windows and permission requests are refused. Response handling cannot start downloads.
- Ordinary Browser profiles and normal app windows retain their existing configuration. Windows retains its existing isolated WebView2 implementation.

This is a narrowly trusted encoder, not a general untrusted-code sandbox. No external service or network proxy is introduced.

## Runtime requirements and limitations

Safari/WebKit 18.4 added MediaRecorder WebM VP8/VP9 support. Older macOS WebKit can therefore fail the WebM probe even when WKWebView itself is available. System WebKit version, canvas manual capture and actual encoder support are tested at runtime, not inferred from the OS version alone. Browser's separate persistent workspace profile still requires macOS 14+.

Linux needs WebKitGTK's ephemeral APIs (2.16+) and this SDK's asynchronous JS API (2.40+), plus a functioning WebM encoder in the installed media stack. No codec is bundled or installed. Missing APIs/codec return an actionable recording error; there is no alternate format or shared-profile fallback. Hidden-webview throttling or native capture/encoder failure can still fail a recording and must be tested on actual desktops.

Public API references:
- https://developer.apple.com/documentation/webkit/wkwebsitedatastore/nonpersistent()
- https://webkitgtk.org/reference/webkit2gtk/2.16.2/WebKitWebContext.html
- https://webkit.org/blog/16574/webkit-features-in-safari-18-4/

## Validation performed

One feature-targeted validation round, CGO disabled:
- `go test ./internal/platform ./internal/linux -run TestBlankEncoder -count=1`: passed. Policy tests and substituted native-function fixtures; no native libraries, network or browser stores were used.
- Darwin arm64 and Linux arm64 backend test-binary compilation: passed, not executed.
- `go test ./internal/desktop/browserrecording -count=1`: passed, synthetic encoder/capture fixtures.
- `node internal/desktop/browserrecording/helper_test.cjs`: passed synthetic transfer, limits, backpressure, cancellation and missing-codec checks. No real MediaRecorder or playback.
- Windows amd64, Darwin arm64 and Linux amd64 builds of `internal/desktop/browserrecording` and `internal/ui/browserpreview`: passed.

No GUI, real codec, capture-to-playback round trip or native network-isolation runtime test was available. These results do not establish native runtime acceptance. No full SDK/application test suite was run for this module, and no actual user browser data was touched.
