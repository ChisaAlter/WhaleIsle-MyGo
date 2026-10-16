# macOS isolated preview: legacy compatibility helper

Status: source implementation; Darwin/arm64 Go launcher cross-compiled. Native
Cocoa compilation, signing and macOS runtime acceptance have not been performed.
This backend uses the public but deprecated single-process WebKit WebView API.
It is not WKWebView-equivalent, and modern HTML/CSS/JavaScript compatibility is
unverified. Global rendering readiness is unchanged.

## Implemented flow

`previewprocess.OpenExperimental` on macOS resolves the dedicated companion at
`Contents/Helpers/WhaleIsle Preview.app/Contents/MacOS/whaleisle-preview` relative
to the host executable. It rejects other paths, symlink substitution, unsigned
or ad-hoc builds, mismatched signing teams and unexpected entitlements.

The parent verifies an Apple-issued signature and the exact App Sandbox-only
entitlement dictionary. It records the signed helper CDHash, launches with a
minimal environment and only standard IPC/diagnostic descriptors, then checks
the live process against that CDHash through Security.framework before sending
any document. The helper independently checks its own live signature,
entitlements and same-team host identity. App Sandbox enforcement comes from
macOS activation of the signed executable, not a hand-written sandbox profile
or an ephemeral web data store. No private API is used.

The legacy WebView runtime class must exist. The helper receives the existing
opaque-iframe document shell through bounded byte IPC and enables JavaScript
only inside the separate process. It denies external/file resource requests,
redirects, page navigation, popups, dialogs, file panels and downloads; Java and
plugins are disabled. The helper has no network or file-selection entitlement
and exposes no native page bridge. The renderer stays in the helper process,
avoiding WKWebView's independently launched WebContent/Networking services.

Interactive mode shows a separate native window with mouse and keyboard input.
Theme messages update the existing document. Capture mode reads the shell's
bounded readiness/height state, resizes to at most 4000 pixels, and returns PNG
bytes through the existing validated capture protocol. Parent cancellation,
startup failure, pipe closure, window closure or the thirty-minute lifetime
limit terminates the helper. Inline embedding is not implemented in this first
compatibility increment and fails with an explicit separate-window message.

## Build and package

On a Mac with the native SDK, run:

    sh scripts/build-preview-macos.sh arm64 /path/to/new-output

Use `amd64` for an Intel target. The build produces an unsigned `.app` and a
copy of `preview.entitlements`. The script does not launch the helper, sign it,
access a keychain, configure credentials, install anything or notarize it.
Use the existing external signing workflow and the host's signing team, keeping
exactly `com.apple.security.app-sandbox=true`. Do not add network, inherited
sandbox, user-selected-file, debugging or JIT exceptions to make a failing
runtime start. If the target's legacy engine cannot run under these constraints,
that target is unsupported and must retain the static fallback.

The existing `stage` operation in `scripts/package-macos.mjs` accepts
`previewHelperBundle` pointing to the already-signed `.app`. Omission preserves
existing package behavior. The supplied helper must have identifier
`org.whaleisle.preview`, executable `whaleisle-preview`, package type `APPL`,
matching Mach-O architecture, an Apple-issued signature, and exactly the
App Sandbox entitlement. Symlinks and special files are rejected.

The bundle is copied with `ditto` into `Contents/Helpers/WhaleIsle Preview.app`
and verified again. Its signing identity is recorded in the package manifest.
The archive operation rechecks the helper and same-team host signature before
creating the archive. The external signing workflow must preserve nested helper
entitlements; do not blanket deep-sign it using host entitlements.

## Acceptance still required

The Go launcher cross-compile does not validate the Objective-C helper or its
SDK declarations. A real signed Mac must establish:

- Native helper compilation and public WebView availability on supported OSes
- Startup under App Sandbox-only entitlements, including any legacy JSC/JIT constraints
- Real HTML/JS, iframe sandbox, forms, canvas and CSS compatibility
- Correct high-DPI PNG pixels, dimensions, console output and input behavior
- Network/file attempts blocked by the OS, not just navigation delegates/CSP
- Cancellation, window close, process identity and malformed IPC behavior

Missing capabilities or initialization errors preserve the static fallback.
No Windows behavior or default in-process HTML renderer is changed.

## Sources

- WebKit describes legacy WebView as single-process:
  https://webkit.org/debugging-webkit/
- Apple marks the legacy APIs deprecated:
  https://developer.apple.com/documentation/webkit/setting-up-a-web-view-legacy
- Current WK service initialization has network-access prerequisites; this
  backend deliberately does not use WKWebView:
  https://github.com/WebKit/WebKit/blob/main/Source/WebKit/Shared/EntryPointUtilities/Cocoa/XPCService/XPCServiceEntryPoint.mm
