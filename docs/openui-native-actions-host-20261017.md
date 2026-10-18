# OpenUI native clipboard and HTTP(S) actions: Host source wiring

## Behavior

The independently authored embedded viewer can request plain-text clipboard
writes and opening an absolute HTTP(S) link from its actual UI event handlers.
CopyToClipboard and Code copy share the same clipboard action. Existing MyGo
operations run on the application's UI thread, outside the contained renderer:

- Clipboard: `mygo.Clipboard.Write(transfer.TextData(text))`, whose returned
  error is observed. The older `WriteText` convenience method discards errors
  and is deliberately not used here.
- Link: the existing `internal/desktop/webview.OpenExternal` HTTP(S)-only route,
  which calls `mygo.Shell.OpenExternal` and returns its actual native result.

There is no renderer-native binding, OS capability grant, custom scheme,
clipboard read, filesystem action, extra confirmation dialog or application
execution endpoint. Existing OS behavior and any OS-provided confirmation are
unchanged. Assistant/form-submission events retain their existing Host
confirmation flow.

## Default-disabled capability and ownership

`Render.HostActions` defaults to false. The chat entrypoint enables it only on
Windows/macOS when `WHALEISLE_OPENUI_VIEWER` is empty and the actual viewer bytes
equal this build's embedded runtime. Every explicit legacy override remains
disabled, including one that happens to contain identical bytes. The paired
trusted shell and Session independently check the actual owned viewer bytes;
the renderer's `ready.runtime` declaration is not sufficient.

The owned frontend checks the synchronous trusted UI event before requesting
an action and checks active user activation when that browser API is present.
The trusted shell checks the exact viewer contentWindow and opaque origin.
HtmlView child messages and generic iframe messages cannot grant actions.
The request's `userGesture` boolean is not, by itself, native attestation: this
boundary relies on the pinned owned runtime's event-handling contract plus the
existing isolated-source and owner checks.

The Host additionally checks the original store owner and active helper before
dispatch, repeats those checks on the UI thread, claims the exact pending
payload once, and checks the owner again immediately before the native call.
Closing the helper clears all pending action claims. A revoked session, unknown
instance, unclaimed/altered payload or repeated claim cannot execute an action.

## Wire contract

The existing `whale-openui` version-1 event envelope and 64-KiB complete-event
limit are retained. New typed events are:

- `type: clipboard`, `text`, `userGesture: true`, `sequence`, `requestId`
- `type: open-url`, `url`, `userGesture: true`, `sequence`, `requestId`

`sequence` is a positive JavaScript-safe integer, strictly increasing for this
helper session. `requestId` must equal `native-action-<sequence>`. The Session
tracks the largest admitted sequence and exact pending payload; both replayed
events and duplicate execution claims are rejected. The paired shell reserves
that ID prefix from resource requests.

Clipboard text is never truncated. Empty text is allowed; invalid UTF-8, NUL
(which would truncate native Windows plain text), and oversized envelopes are
rejected. Links require a nonempty hostname, lowercase `http` or `https`, no
userinfo credentials, no opaque URL form, and no raw control characters or
surrounding whitespace. The frontend submits a normalized absolute URL; the
Host validates it independently and the existing product route validates it
again.

Replies reuse the existing nonce-checked, bounded FIFO API-response transport:
`{instanceId, requestId, status, body: {ok, error?}}`. The paired shell recognizes
its typed pending action and forwards `action-response` with the same request
ID and action type. Resource responses keep their existing path. No helper
packet kind or native message bridge is added.

Only a nil error from the actual clipboard/native URL call yields status 200
and `ok: true`. Native errors, ownership rejection and an unavailable UI loop
yield `ok: false`; error text is bounded to 4096 UTF-8 bytes. For URL opening,
success means the OS accepted the default-browser dispatch, not that the page
loaded. A missing/late acknowledgement must not be presented as success.

## Platforms and delivery limits

Windows and macOS use their existing interactive-preview event and reply paths.
Linux and unsupported platforms remain unavailable, as the OpenUI helper bridge
was already unavailable there. A helper lacking the existing API-response
capability is refused before a native action is claimed or executed.

This module requires the paired owned-runtime/shell native-action package and
must be published with it. Before this change, clipboard requests were rejected
by the shell and open-url requests were ignored; explicit legacy overrides keep
that restriction.

Only source inspection, Go formatting and frozen-file hash verification were
performed. No tests were added or run, no native build was attempted, and no
clipboard, browser-open, application, untrusted HTML or native platform behavior
was executed. Runtime/platform acceptance remains outstanding.
