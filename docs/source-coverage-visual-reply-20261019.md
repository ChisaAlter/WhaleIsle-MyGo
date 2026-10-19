# CHAT21 visual reply error and initial-height source follow-up

This bounded follow-up implements two source-confirmed differences. It is not full CHAT21 acceptance. No tests, build, preview helper, real Host, page, model request, service, listener, or external request was executed. Only owned source formatting and whitespace checks ran. Regression fixtures are prepared, not passed.

## Original evidence

Read-only original: `C:/AI/Deepseek-Harness-Desktop/vendor/deepseek-harness/packages/client/ui-visual-replies/src/client/PublishedPage.tsx`. Its `pageFailure` suppresses reload for `session/not-found`, plus exactly these `session/attachment-invalid` detail reasons: `ATTACHMENT_NOT_REFERENCED`, `VISUAL_REPLY_TOO_LARGE`, `VISUAL_REPLY_INVALID_UTF8`, `ATTACHMENT_NOT_FOUND`, `ATTACHMENT_CORRUPT`, `INVALID_ATTACHMENT_REF`. Unknown reasons and transport failures retain manual reload. A generic “unauthorized” string is not one of these rules.

The same file passes `model.height` to `VisualReplyFrame`. `model.ts` uses `presentation.height ?? 480` and accepts finite numbers from 80 through 2000; `VisualReplyFrame.tsx` starts its local height from that value and later accepts bounded, rounded size messages from the actual iframe. Original `src/html.ts` injects the page's measuring/ResizeObserver bridge. These are original pinned source observations, not claims based on a newer browser version.

## Implemented source

- `9cf03fc`: `internal/store/session/visual_reply_snapshot.go` preserves `dsh.Error` and classifies precisely the original permanent Host failures. Local invalid reference/content/size/UTF-8/byte-count results use a concrete permanent content-error type. Ordinary Host absence and session/generation change remain transient. `cmd/whaleisle/visual_reply_inline.go` supplies its existing Retry action only for a retryable error.
- `e6e842c`: publication metadata retains initial height (default 480, including explicit null; fractional declarations round up to native pixels). `ReadVisualReplyPageSnapshot` returns that metadata and authorized HTML from one read; existing `ReadVisualReplySnapshot` is a compatibility wrapper and does not duplicate the RPC. Inline state uses declared height until the helper has a valid content measurement; repainting never rereads metadata or replaces the mounted page.
- The same height commit adds optional `previewprocess.Document.InitialHeight`. Zero preserves the previous 700-pixel viewport for existing callers. Values 80–2000 travel in the existing document packet's height field, are validated at send/receive boundaries, and initialize the Windows helper viewport before its shell loads. There is no new message kind, RPC, helper launch entry or resource permission.

The helper's apparent `height || 360` fallback was traced rather than assumed to be a permanent override: the ordinary shell starts with height zero but does not become ready until an actual valid `preview-ready`/size report arrives. Windows waits for that readiness before reporting loaded; subsequent size packets contain the bounded measured content height. Initial viewport metadata now reaches the actual browser before that measurement, including pages whose dimensions depend on viewport height.

Existing Files/OpenUI callers keep zero initial-height behavior. Non-Windows pixel viewers still use their existing fixed 700-pixel rendering viewport; that platform boundary is outside this Windows follow-up. No claim of cross-platform height parity is made.

## Prepared verification and remaining acceptance

Prepared fixtures are `internal/store/session/visual_reply_failure_test.go`, `cmd/whaleisle/visual_reply_failure_test.go`, and `internal/desktop/previewprocess/initial_height_test.go`. They cover wrapped typed Host failures, unknown/transient reasons, concrete invalid local content, Host absence versus owner retirement, the actual inline Retry callback gate, native/PTC metadata/default/bounds, a metadata read without duplicate RPC, legacy zero-height protocol compatibility, invalid sender/receiver heights, and the Windows source connection from viewport initialization to later size measurement.

The reader fixtures use a custom in-memory `http.RoundTripper`; they do not need a port or live server. Protocol fixtures use byte buffers. The main inline fixture seeds an already failed occurrence and never reads or launches a surface. The Windows source assertion establishes wiring only. Parent-owned unified tests/build and actual isolated native rendering acceptance remain outstanding; no visual, interaction or process-containment QA pass is asserted here.
