# P02/P03 controlled PDF reader

Reference: WhaleIsle e9f584be6d3b491ce777bb6823f30bc2bbc6f195,
vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/pdf/
(pdf.tsx, runtime.ts, assets.ts, text.ts, store.ts). The reference uses PDF.js
6.3.289, an owned module Worker, offline binary resources, TextLayerBuilder,
visible-page updates and tab-local fit-width/fixed zoom. The Desktop iframe
fallback is not the reference implementation for this work.

The MyGo reader uses the same pinned upstream engine and text layer, with a
small product-owned DOM reader rather than the original React body. It retains
all page geometry, lazily renders viewport pages, and publishes actual page,
fixed/fit-width zoom, scroll offsets and text-selection endpoints through a
bounded same-document fragment. WebView2 SourceChanged feeds the existing
Host state. No new native message bridge or HTTP write endpoint is introduced.
Workspace snapshots store only numeric reading state; capabilities and document
bytes remain transient. While a floating reader is active its state is
authoritative; closing it restores the owner tab's reader.

LocalPDFViewerURL is separate from the legacy script-disabled PDF capability.
Only the immutable reader, version-matched engine/worker/text-layer assets and
the current PDF are served. The native policy allows exact listed GET/HEAD
resources and reader-only navigation. No Host credentials/bindings/objects,
user preloads, subframes, downloads, file access or external resource access.
No PDF scripting manager, annotation script execution, XFA or QuickJS assets
are installed. Original PDF/HTML/media restrictions remain in effect.

The embedded resource provenance records the official npm integrity, upstream
version, original reference commit and SHA-256 hashes. Packaging has no runtime
npm, CDN, directory server or arbitrary resource fallback. PDF preview remains
Windows/WebView2-only; unsupported platforms fail explicitly.

## Evidence and limits

- JavaScript syntax check: passed.
- Targeted Go PDF resource/ownership, exact capability policy, state round-trip,
  lease and floating-window fixtures: passed.
- Windows amd64 CGO=0 compilation of the affected PDF, Browser preview and
  WebView Host packages: passed.
- Actual PDF.js worker/canvas/text selection execution: not verified here.
  Local headless Chromium could not start because the execution sandbox denied
  its socket creation; the elevated attempt failed before launch on a read-only
  runtime mount. No sandbox policy was changed or bypassed.
- Windows WebView2 GUI, real native SourceChanged delivery, visual layout,
  mouse selection and rendering fidelity: not run. Build and state fixtures
  must not be presented as GUI acceptance.

The reader explicitly reports expired documents, encrypted PDFs, rendering or
worker failures; retry and system-open remain available. Preview bounds are
5000 pages and 32 million canvas pixels per page. It does not add PDF editing,
annotation scripts or arbitrary document network access.
