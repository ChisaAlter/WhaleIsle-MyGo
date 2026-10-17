# CH12 native image response bodies

This increment extends only fixed OpenUI API response bodies to the helper's
existing 25 MiB document-body budget. The `api-response` command and
`Session.ReplyOpenUIAPI` interface remain unchanged; iframe events stay at 64 KiB,
state at 48 KiB, and the protocol header at 4096 bytes. It does not grant network
access to the iframe/helper or introduce an Eval/RPC command.

Windows and macOS helpers advertise `openUIAPIBytes: 26214400` in readiness.
Missing or unrecognized byte capability retains the basic helper's 64 KiB limit.
The Go admission path rejects an oversized response for that older helper before
writing its pipe, rather than sending an unsupported body. A new helper strictly
validates the typed JSON and invokes the same fixed parent-shell response
function, whose exact instance and pending request ID checks still apply.

The FIFO retains the existing 16 transport slots and now also accounts for a
maximum 25 MiB of queued response bodies in total. Either exhausted resource
applies backpressure; it does not truncate the response or impose a lifetime
request count. One encoding-admission slot prevents waiting producers from
creating unbounded additional wire copies. Dequeue releases its byte charge;
Close discards buffers and wakes both queue and encoding waiters.

Inline macOS continues its single command/event-batch/frame sequence. An API
response remains deliverable when the pixel seat is hidden. Public image fetch,
bitmap validation, data URL construction and owner-context enforcement belong
to the separate Go adapter stage; no service or location request was made here.

## Focused verification

- One focused `go test ./internal/desktop/previewprocess -run 'TestOpenUIAPI'
  -count=1`: PASS 0.486s.
- A 13 MiB synthetic image-shaped JSON response round-trips through the bounded
  body protocol. A second 13 MiB response waits until the first leaves the FIFO.
  The same large payload remains invalid as an iframe event.
- Existing response fixtures cover typed validation, opaque IDs, FIFO/Close,
  fixed script and capability rejection; the additional fixture rejects a large
  reply for the older 64 KiB helper and accepts it after the advertised upgrade.
- Windows amd64 and Darwin arm64 Go package test cross-compiles passed with CGO
  disabled. Objective-C SDK build/signing and target-OS rendering remain pending.

The basic response-stage frozen payload remains unchanged. This increment must
be published with the adapter stage defining `openui.MaxAPIResponseBytes` and
validating API responses against that shared 25 MiB budget.
