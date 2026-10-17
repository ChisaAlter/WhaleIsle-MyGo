# CH12: same-helper OpenUI streaming updates

This increment builds on the bounded event bridge. `Session.UpdateOpenUI(render)`
accepts only the session's existing instance ID and a valid Render (program at
most 256 KiB, input state at most 48 KiB). Ordinary HTML/capture sessions do not
have an update queue. Older helpers lacking the `openUIUpdate` ready capability
return an explicit unsupported-update error.

Admission is synchronous and non-blocking on renderer work. One pending update
is retained; newer updates replace it. Identical updates are deduplicated and
transmission is limited to 10 Hz. Close drops the pending update. Transport
failure/timeout closes the helper session rather than silently losing a stream.
A successful method return means admitted/coalesced, not a renderer ack.

## State and transport

The fixed `render` command carries only instanceId, program and streaming in a
bounded JSON body. State and theme are deliberately absent. The existing trusted
shell's `__whaleOpenUIRender` changes only program/streaming and preserves its
latest viewer state/theme; it does not recreate the iframe. The 2 MiB wire bound
accounts for JSON escaping of the bounded program, not a larger program budget.

Windows serializes render packets and theme packets under one command mutex.
The helper validates strict typed JSON and invokes only the fixed shell method.
macOS external windows use the same rate-limited pump; inline mode integrates
updates into the existing one-request/one-frame loop. Each render response still
contains at most the bounded events followed by its frame. Input is neither
reinterpreted nor granted new authority. The native helper adds no Host access,
network/clipboard/URL operation or caller-selected evaluation function.

## Verification and remaining runtime limits

One focused final invocation passed new update fixtures and existing event/wire
cases (0.216 s). Fixtures cover latest-only coalescing, 10 Hz cadence, same-instance
ownership, state/theme absence, close and duplicate cleanup, old-helper rejection,
program/state/wire bounds, strict JSON and safe fixed-script serialization,
failed transport cleanup and native inline-source contracts.

The Go package cross-compiled for Windows amd64 and Darwin arm64 with CGO disabled.
The macOS Objective-C helper remains source-checked only, without the macOS SDK,
signing or target execution. Real viewer/runtime acceptance and the full project
suite were not run. Linux OpenUI and capture remain explicitly unsupported here.

The platform worker separately owns shell state-retention tests, chat/owner
lifecycle, inline layout and parent-window handle wiring. Its follow-up must be
integrated with this increment; the frozen phase-one snapshot is unchanged.
