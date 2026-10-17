# CH12 native API response bridge

The native helper supports one additional fixed command, `api-response`, with a
bounded typed JSON body. `Session.ReplyOpenUIAPI(openui.APIResponse) error` checks
the exact instance and the shared protocol validator. It accepts only a nonempty
opaque request ID, status 200–599 and JSON body fitting the existing 64 KiB event
envelope. It does not add character/length restrictions to the request ID.

Responses use a FIFO with the existing 16-slot event transport budget. A full
queue applies backpressure to the independent resource worker until space or
Close, without coalescing, dropping responses, or limiting lifetime request
counts. Close wakes waiting producers and discards buffered responses. A stalled
write has the existing five-second helper-close watchdog; a broken transport
closes the helper. A missing `openUIAPI` capability bit fails visibly.

Windows and external macOS serialize the body with existing theme/program pipe
writers. Inline macOS sends one command followed by the existing bounded event
batch and frame; API responses continue to drain even when its pixel seat is
hidden. Ordinary HTML/capture mode cannot receive the command.

The only script is a fixed invocation of
`window.__whaleOpenUIAPIResponse(response)`, requiring `true`. Strict typed
validation and reserialization reject extra command fields and trailing JSON.
The trusted parent shell checks its exact instance and pending request ID,
consumes that pending request and posts the response to its opaque iframe. An
unknown, duplicated or stale request is rejected there. The native layer does
not create an additional unbounded request-ID registry.

There is no arbitrary Eval/RPC operation, Host credential forwarding or native
network permission. Only the separately implemented Go resource adapter knows
how to call the three fixed public service operations; the iframe remains
network-disabled. No public service query or location transmission was used
for verification.

## Verification

- One focused previewprocess run: `go test ./internal/desktop/previewprocess
  -run 'TestOpenUI|TestPacket|TestDocument' -count=1`, PASS 0.194s.
- Tests cover ordinary/old-helper/closed/wrong-instance admission, bounds, opaque
  Unicode request IDs, owned byte copies, FIFO/backpressure/Close, more than 16
  lifetime requests, framed body round trips, fixed script/extra field rejection,
  failure closure and source-level macOS/inline command invariants.
- Windows amd64 and Darwin arm64 Go package test cross-compiles passed with
  CGO disabled. They do not compile the Objective-C helper.
- Objective-C SDK build, signing and target-OS GUI execution remain unverified.

This is the basic 64 KiB response stage. Larger image data URL transport, if
implemented, is a separate increment and must not change this frozen payload.
