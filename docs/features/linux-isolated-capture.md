# Linux isolated capture and interactive byte viewer

The explicit Browser Preview script action can launch a separate trusted helper
inside bubblewrap with a private packaged root filesystem, network/PID/user/IPC/
UTS/mount namespaces, private Xvfb and private D-Bus. Capture remains supported.
No host display, Wayland socket, home or workspace is mounted. The default static
preview, fixed blank encoder and global readiness remain unchanged.

A trusted companion `whaleisle-preview.runtime` directory must contain the
minimal Linux runtime (WebKitGTK/GTK libraries, fonts, `/usr/bin/Xvfb`,
`/usr/bin/dbus-run-session`, `/usr/bin/xdotool` and their dependencies). The package
script checks those executables; it does not build a runtime from host paths.
The root is read-only; `/tmp`, `/run`, and `/home` are private tmpfs mounts.
Missing packages, namespace/startup failures, or invalid IPC fail closed and
leave the static preview intact. Capture-only launch does not require xdotool.

The helper checks namespace identities before GUI initialization and the parent
sends the frozen document only after the nonce handshake. WebKit has no app
bindings, application protocols or privileged services. Permissions, downloads,
new windows and non-blank navigation are denied. The interactive document uses
the same opaque iframe and restrictive CSP as the Windows interactive helper,
including local form handlers but no form transmission or runtime file/network
resource loading. Prepared inline images/fonts remain available.

## Separate-window interaction

The parent creates a native MyGo pixel viewer, never a webview containing the
untrusted document. The contained WebKit window is visible only on Xvfb. A real
validated first PNG is required before interactive startup succeeds. Subsequent
frames are requested at idle 10 Hz, with one request in flight, an 8 MiB encoded
limit and exact width-by-700 dimensions checked before decoding. UI publication
also applies backpressure. Each pipe exchange has a five-second watchdog; close,
owner cancellation, renderer failure or the thirty-minute lease tears down the
helper tree and viewer. At most 64 pending input/theme commands are retained;
intermediate pointer motion is skipped while commands are queued, so high-rate
motion cannot exhaust the queue. Discrete input overflow closes the session
rather than dropping a button release.

Typed pointer coordinates/button transitions, vertical wheel steps, committed
UTF-8 text (up to 1024 bytes), and a fixed navigation/edit key allowlist are sent
through the parent-only pipe. Packaged xdotool targets the single visible helper
window discovered by PID on the private display. There are no arbitrary shell,
JavaScript, path, host-window-ID or clipboard commands. Modifier shortcuts,
composition display, clipboard transfer, horizontal wheel, accessibility tree
transport and inline card embedding are not implemented. Inline requests still
fail closed; the GTK embedded seat is unchanged. macOS remains unsupported.

## Verification boundary

The focused `TestInteractiveWireBounds` and
`TestInteractiveMotionBackpressurePreservesTransitions` tests compile the Linux previewprocess
package and checks frame roundtrip/dimensions plus input size, coordinate and
command bounds. This is source-level implementation, not packaged-runtime GUI
acceptance. A real runtime test must cover initial pixels, buttons/drag release,
scroll, text and allowed keys, cancellation/window close, malformed frames,
renderer hang/exit, and resource/network isolation. No Ready promotion is made.
