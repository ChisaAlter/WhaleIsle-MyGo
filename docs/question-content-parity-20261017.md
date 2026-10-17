# CH62 / CH84: question content parity

Original source is read-only, pinned at `a8c931da37376ce7dc7989e6f30b0db885e06d26`. Root reported CH56 published at `983de31` before this change.

## Original contract
- `vendor/deepseek-harness/packages/client/ui-user-questions/src/client/QuestionComposer.tsx:450-453`: independent plain-text header and question heading.
- `QuestionComposer.tsx:506-507`: rich question detail through shared MarkdownText.
- `QuestionComposer.tsx:510-543`: all options and complete plain-text descriptions.
- `QuestionComposer.module.css:17-20,160-166,177-192,273-306`: bounded card, scrolling body, non-shrinking wrapping option rows.
- `vendor/deepseek-harness/packages/client/ui-primitives/src/markdown/CodeBlock.tsx:157-170`: code defaults to wrapped and exposes copy/wrap controls.

## Native changes
- `internal/ui/interaction/question_content.go:9-26`: complete width-constrained prose and reused safe Markdown renderer for detail. No new Markdown parser, external navigation handler, or network fetch.
- `internal/ui/widgets/markdown/markdown.go:24-26` and `blocks.go:49-77`: caller-scoped `CodeWrapDefault`, enabled only for question detail. Existing other consumers retain their default; user toggles persist per mounted block. Clipboard retains exact parsed code bytes including trailing newlines.
- `internal/ui/interaction/card.go:115-127`: header and question remain independent, fully wrapping and available while minimized.
- `internal/ui/interaction/question_editing.go:61-113`: local scrollport (at most 520 DIP / half window height), footer outside the scrollport; shrinkable checkbox labels handle long unbroken identifiers. All descriptions are rendered for every option count.
- Existing pagination, option Enter, single-select advancement, multiline/IME, Sending locks, and scoped focus/timer behavior are retained. `question_history.go` is untouched.

CH56 had already restored separate header content and removed the old >4-option description omission. CH84 additionally closes confirmed long-unbroken-label clipping and long-body footer reachability defects.

## Verification
Red evidence before fixes:
- `TestQuestionDetailMarkdownCodeCopyAndWrap`: missing Markdown code toolbar, then default wrap mismatch.
- `TestQuestionContentLongUnbrokenOptionFitsNarrowCard`: 264-character unbroken label remained 20 DIP high.
- `TestQuestionContentLongBodyScrollKeepsFooterReachable`: Submit at Y=1285 in a 700 DIP-high window.

Final focused command (MyGo only):
`XDG_CACHE_HOME=/tmp/whaleisle-font-cache GOTOOLCHAIN=local GOPROXY=off GOMODCACHE=/workspace/scratch/2d36477d56ec/.gomodcache GOCACHE=/workspace/scratch/2d36477d56ec/.toolchain/gocache /workspace/scratch/2d36477d56ec/.toolchain/go/bin/go test ./internal/store/interaction ./internal/ui/interaction ./internal/ui/widgets/markdown -count=1`

Passed: store interaction 0.115s; UI interaction 0.185s; Markdown 0.046s. Coverage includes code clipboard bytes, wrap toggle/default, multiline edit/Sending freeze, full six-option content, unbroken label wrapping, 700 DIP window scroll/footer reachability, and existing question/approval regressions.

An earlier combined run passed store/UI interaction but exposed old `TestSpansBoldAndCode`, which asserted `Consolas` although production code already reads configured code font (`monospace` by default). Only that test expectation was corrected to current Code family/size; no production font policy changed. Earlier failure is retained here rather than represented as an initially clean run.

## Limits
Source implementation and headless focused verification only. No full-project suite, original execution, real Host integration, or target-platform GUI acceptance. Shared Markdown typography and English toolbar labels remain those of the existing component. This change does not expand settled/reply history or add an external link opener.
