# CH12: bounded OpenUI helper events

The existing Windows AppContainer and signed macOS legacy helper can carry a
separate, explicit OpenUI mode. `Document.OpenUI` is trusted application input;
`openui.Shell` builds the top-level shell in the parent before document delivery.
The existing length-prefixed HTML body carries the render data. Only an OpenUI
mode bit is added to the 4096-byte protocol header. A ready-handshake capability
bit rejects old helpers that would display the viewer without delivering events.

## Boundary

- The isolated iframe retains opaque origin, no network/worker/native bindings,
  no Host credentials and no workspace paths. The OpenUI shell is owned by the
  separate OpenUI adapter; this bridge does not broaden its policy.
- A fixed native expression drains only `window.__whaleOpenUIDrain()` in the
  trusted top-level shell. No page-supplied script, operation name, URL or
  arbitrary RPC becomes a native command.
- At most 16 events per drain and 64 KiB per event; event bodies are independently
  framed, never squeezed into the protocol header. Shell error/overflow fails
  closed. Native receipt calls `openui.ValidateEvent` for the expected instance.
- `Session.Events()` exposes a capacity-16 channel of copied JSON. It closes and
  discards pending events when the session closes. Overflow, stale instance,
  malformed JSON or disallowed event type closes the helper session.
- Events are renderer reports/user intents, not permission to call the Host.
  Consumers must still check their session/Host/message owner ticket and action
  policy. Native code performs no Host request, clipboard action or URL open.
- Ordinary HTML has no event channel. Its shell and theme protocol remain in use.

## Platforms and lifecycle

Windows polls the trusted shell every 150 ms with a bounded evaluation timeout,
then continues the existing size reporting. macOS inline mode emits at most one
bounded event batch before each requested frame; the parent permits only one
request/frame pair in flight. macOS external windows poll every 150 ms on the
main run loop. Neither mode adds a general page binding.

This increment does not add `Session.UpdateOpenUI`: program changes require an
explicit rebuild with validated state retained by the owner. OpenUI capture mode
is rejected. Linux OpenUI is explicitly unavailable in this increment rather
than silently using an incompatible ordinary HTML shell.

## Verification

Focused protocol tests cover body/header separation, trusted-shell boundaries,
batch/event bounds, stale instance and privileged event rejection, copied queue
ownership, overflow/close cleanup, exact nonce/phase, truncation, event/frame
interleaving and old-helper capability rejection. Ordinary protocol regression
cases were also exercised. The initial native-source test had an incorrect
relative fixture path; it was corrected and the affected test rerun. Final
OpenUI cases passed (0.011 s).

Go package cross-compilation passed for Windows amd64 and Darwin arm64 with
CGO disabled. The macOS Objective-C helper was source-checked only: no macOS SDK,
signed helper build, WebView2/AppContainer launch, live pinned viewer bundle or
target-platform runtime acceptance was performed. Those remain runtime-unknown.
