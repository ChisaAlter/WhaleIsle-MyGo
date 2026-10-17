# Windows OpenUI HtmlView frame admission: source-only wiring

## Scope

The contained Windows helper opts into this policy only after receiving the
nonce-checked parent document packet with `openUI` enabled and completing the
existing token, job and browser-process identity checks. Ordinary HTML never
calls the opt-in API and retains its existing one-srcdoc policy. The change
does not modify AppContainer capabilities, OS tokens, network rules, native
bindings, host objects, web messages, permissions or resource filtering.

The trusted shell has one opaque OpenUI viewer. The viewer may contain immediate
opaque `HtmlView` children. Frame identities are retained COM `IUnknown`
identities, not `window.name`, URL fragments, HTML attributes or messages.

- The global frame-navigation callback first cancels every OpenUI frame request.
- The first actual viewer frame gets one exact `about:srcdoc` admission.
- Only frames reported by that viewer's own `ICoreWebView2Frame7.FrameCreated`
  callback receive a direct-child navigation handler. Each gets one exact
  `about:srcdoc` admission. Each HtmlView also receives a Frame7 descendant guard.
  Deeper actual frames are tracked recursively with deny-only NavigationStarting
  handlers: they can never clear cancellation, including for `about:srcdoc`.
  Normal HtmlView removal/recreation is unaffected.
- All additional roots, deeper frames, unknown frames, non-srcdoc destinations,
  redirects and second document navigations remain cancelled. The explicit
  descendant guard does not rely on legacy global event delivery for depth 3.
- A destroyed frame loses its retained identity/subscriptions. The viewer slot
  is never recycled. Updating HtmlView HTML must recreate its iframe, as the
  paired owned runtime does. Height/title changes can retain the same frame.
- Window cleanup revokes the frame policy and releases event subscriptions and
  COM references before the controller is closed.

## Feature reporting and older WebView2 runtimes

`Page.EnableIsolatedOpenUIFrames` and `Page.IsolatedOpenUIFrameSupport` are
Host-only optional-backend APIs; neither is bound to web content. The latter
reports unknown until the actual viewer has been tracked. Frame7 absence or
failure to register its child event leaves the basic viewer's Frame2 policy
active and HtmlView disabled. An admitted HtmlView must itself expose Frame7
and successfully register its descendant guard before its srcdoc is admitted.

The helper then invokes the existing controlled top-shell Eval path with the
fixed `__whaleOpenUIHtmlViewSupport(true|false)` call and checks its return value.
The paired shell/runtime default `render.htmlView` to false. An unsupported
HtmlView displays an explicit unavailable message without creating its iframe.
Only the trusted Host supplies this feature flag; nested documents cannot
negotiate or grant support. Other OpenUI components remain usable without
Frame7. Missing identity/navigation APIs or failed essential event installation
fail the preview closed instead of relaxing isolation.

## Native ABI provenance

All new vtable indices and interface IDs were read directly from Microsoft's
`Microsoft.Web.WebView2` package version `1.0.3537.50`, member
`build/native/include/WebView2.h`. This was a source-header inspection only;
the package was not installed or executed and is not added to the project.

Official package:
https://api.nuget.org/v3-flatcontainer/microsoft.web.webview2/1.0.3537.50/microsoft.web.webview2.1.0.3537.50.nupkg

| Interface | IID | Used vtable slots |
| --- | --- | --- |
| IUnknown | 00000000-0000-0000-c000-000000000046 | QueryInterface 0; Release 2 |
| ICoreWebView2_4 | 20d02d59-6df2-42dc-bd06-f98a694b1302 | add_FrameCreated 73; remove_FrameCreated 74 |
| ICoreWebView2FrameCreatedEventArgs | 4d6e7b5e-9baa-11eb-a8b3-0242ac130003 | get_Frame 3 |
| ICoreWebView2Frame2 | 7a6a5834-d185-4dbf-b63f-4a9bc43107d4 | inherited add_Destroyed 8/remove_Destroyed 9; add_NavigationStarting 11/remove_NavigationStarting 12 |
| ICoreWebView2Frame7 | 3598cfa2-d85d-5a9f-9228-4dde1f59ec64 | add_FrameCreated 30; remove_FrameCreated 31 |

Microsoft documents Frame7's event as reporting direct descendant iframes,
introduced in WebView2 `1.0.3240.44`:
https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2frame7?view=webview2-1.0.3537.50

Microsoft documents the global FrameNavigationStarting handlers as running
before the frame-specific NavigationStarting handlers, sharing the same event
arguments. That ordering permits the default cancellation to be cleared only by
the exact tracked frame's handler:
https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2frame2?view=webview2-1.0.3537.50

The Frame7 descendant guard also avoids depending solely on the legacy global
event at deeper levels. Microsoft's issue tracker contains an older runtime
report of global navigation events missing nested frames:
https://github.com/MicrosoftEdge/WebView2Feedback/issues/4605

## Remaining verification limits

This is a source-only implementation. No tests were added or executed; no Go
compiler/native build, browser, WebView2 instance, untrusted HTML or application
was run. Source and official ABI references were inspected, and the frozen
sources are hash-listed with their pre-edit base hashes. `gofmt` was unavailable
in this workspace; no formatter or toolchain was installed.

Actual Windows amd64/arm64 compilation, runtime event ordering, first-srcdoc
startup, child creation/removal/recreation, deep-frame rejection and older-runtime
fallback remain unverified. This source change is not an isolation attestation
or a claim that native acceptance is complete. It must ship together with the
paired owned HtmlView runtime/shell capability wiring; publishing only this
Windows portion would leave the fixed shell setter unavailable.

Nested iframe/object/frame content inside HtmlView is unsupported. Deeper frame
documents are rejected by their own synchronous NavigationStarting callback,
rather than relying solely on legacy global event delivery. A missing essential
interface or failed callback installation closes the preview as unavailable;
closing on an installation failure is not a demonstrated zero-execution-window
guarantee. The source relies on WebView2's documented frame-creation/navigation
lifecycle and callback semantics; those guarantees remain unverified on an
actual native runtime in this source-only delivery.
