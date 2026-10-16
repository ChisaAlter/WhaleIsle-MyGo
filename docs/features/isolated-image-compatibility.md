# Bounded isolated image compatibility preview

The Files image toolbar adds an explicit compatibility-preview action for
GIF, ICO, AVIF and SVG. Default presentation now uses bounded native image
decoding, preserving animated GIF content, SVG artwork and native zoom/pan.
The explicit compatibility action remains this isolated live WebView, with
its original stricter SVG policy unchanged.
This increment reuses the existing static document capability and Windows
WebView surface; it does not install codecs or open a privileged page.

## Authority and isolation

The action reads the saved image through the initiating session's os.Root,
with a 16 MiB maximum (1 MiB for SVG), stable-file checks and an immutable
snapshot. Drafts are not saved, uploaded or submitted. A fixed HTML template
contains a base64 data image; user SVG bytes are never inserted as HTML.

The existing static DocumentShell sandbox and LocalHTMLURL page enforce
no scripts, network, application bindings, popups, downloads or external
navigation. The only served URL is a random read-only document capability.
The lease expires after 30 minutes and is revoked with its owner/session.
Image snapshots deliberately have no interactive HTML source or root identity:
CanRunScripts hides the promotion control and InteractiveSnapshot rejects
them independently. Browser pick/capture behavior is otherwise unchanged.

## Supported bounded subset

All raster paths apply a 32-million-pixel budget and at most 256 frames/items.

- GIF: complete descriptors and sub-blocks, logical canvas and every frame;
  cumulative frame pixels and full-canvas compositing across all frames must
  fit the pixel budget. LZW output is validated incrementally in a small
  buffer without retaining decoded frames.
- ICO: every directory entry, extent and actual PNG/DIB dimensions are checked,
  including cumulative pixels. Supported DIB entries use the uncompressed
  40-byte Windows header with 1/4/8/24/32 bpp. PNG entries have bounded chunks,
  checked CRCs, and no APNG or compressed metadata.
- AVIF: bounded ISO BMFF metadata with one unprotected primary av01 item,
  one mdat extent and explicit associated ispe/av1C properties. Sequences,
  grids, alpha/auxiliary images, references, ambiguous layouts and unsupported
  properties are rejected. Display still depends on the installed WebView
  runtime's AVIF decoder; opening the snapshot is not proof of successful
  image decoding.
- SVG: bounded static shapes, text, gradients, clipping/masks and local
  references. XML tokens, nesting, attributes, numeric geometry and expanded
  reference graphs are limited. References must resolve locally, be acyclic
  and stay under the expansion budget. Root dimensions must be bounded pixels
  or a supported viewBox. Scripts/events, animation, filters, foreign content,
  nested images/data, external references, CSS imports/escapes/computed
  resource functions and ambiguous stylesheet resource references are rejected.

These are byte/container/resource gates, not a proof of the browser codec's
internal allocation behavior. AVIF elementary-stream dimensions and PNG zlib
output are not decoded by this gate. Arbitrary complex SVG and every WebView
version are expressly outside the compatibility claim. Unsupported content
fails with a visible reason; external opening is a separate user action and
is not counted as successful compatibility preview.

## Delivery check

Source review covered the native/static snapshot API, ownership cancellation,
absence of interactive promotion, bounded parsers and immutable data markup.
No real workspace image, external opener, WebView profile or model was run.
Integration owns the one incremental Windows compile check; runtime visual and
decoder compatibility remain unaccepted.
