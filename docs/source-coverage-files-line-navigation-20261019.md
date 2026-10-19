# Files source-line navigation coverage — 2026-10-19

This is a narrow source audit and regression preparation record. It is not runtime acceptance or a census of every Files/document-preview contribution.

Original read-only source: `C:\AI\Deepseek-Harness-Desktop`, `vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/TextPreview.tsx` 136–158. Its navigation effect retains an unhandled target revision, loads another page while the requested line exceeds loadedThrough and EOF is false, and answers navigation only once covered or at EOF. Loading/failure suppresses another automatic request.

The Go gap was confirmed in `internal/ui/files/source_actions.go`: requestLineNow clamped to the already loaded prefix. Initial text-read completion consumed pendingLine immediately; successful next-page application never continued it. Merely exposing the existing next-page button did not preserve an out-of-page navigation.

Source correction: `a00c31309b8d1fede7511b94b9f74fac03b852be` (four owned product files).

- The same FileTab retains the raw one-based requested line through initial read and serial page appends. A successful page either covers it, reaches EOF and clamps, or requests the next page.
- A page failure keeps the loaded prefix, cursor and original target. No paint loop retries it. The existing next-page/reload action can retry.
- Tab departure cancels its queue generation and marks only a cancelled pending page for resume on returning to the same tab. Failure is not marked for automatic resume.
- Existing textReadQueue ownership remains BindingID, workspace root, session, path, tab occurrence readID and generation. Existing cursor-advance checks remain. A resource-version mismatch rejects the append and retires old pending navigation; Host Bind also resets it.
- Paged resources retain their existing read-only save guard. No new RPC, page control or file-discovery behavior was added.

Prepared `internal/ui/files/text_line_paging_regression_test.go` covers an initial prefix followed by two pages, EOF clamp, failure plus explicit retry, cancelled same-owner generation arriving late, hidden-tab suppression, occurrence mismatch, resource-version retirement and Host rebinding. Readers return in-memory TextPage fixtures; no real workspace file, Host, account or application is opened.

Only gofmt was executed. Tests, build and application/runtime verification were deliberately not run. These named source chains are S2 (source implemented); actual scrolling, native selection and live Host paging remain unverified.
