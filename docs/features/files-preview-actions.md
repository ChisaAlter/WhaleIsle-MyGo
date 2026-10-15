# Files source, selection, line navigation and image actions

This increment follows the original `packages/ui-files` contracts at
`90465b67ec16765631d54c195af19d9ff3c3a770`. It is a code-level delivery,
not runtime acceptance.

## Source and rendered Markdown

Markdown opens in Source by default. A per-tab Source/Rendered control renders
the current draft with the existing native Markdown widget; switching never
saves, reloads or discards a draft. Its supported HTTP(S) links use the existing
explicit-click opener. The renderer does not execute HTML or scripts.

Source previews, editable text and split/highlight-only source panes use native
TextArea selection, with syntax colors attached to original source rune ranges.
Markdown source is no longer silently rendered by the syntax-preview path.
Existing edit modes and dirty-tab save/conflict handling remain intact.

## Selection and jump to line

Add to chat uses complete selected lines, a one-based L-range and relative path,
followed by a fenced text block, matching the original file action. It reads the
native selection rather than searching the clipboard or collecting the whole
file. Backtick-containing paths/content receive safe delimiters. A stale
selection after a draft change, reload or tab switch cannot be appended.

The shared composer action validates the active session, sending state and IME
composition. It appends only to that session's current draft, preserves its text
and attachments, and does not submit, upload or call a model.

Go to line accepts a positive line number, clamps it to the file, switches
Markdown to Source and scrolls the selected line into view. LF, CRLF, CR and
non-ASCII text use native rune offsets. Conversation-file clicks additionally
recognize `:line`, `:line:column`, `#Lline` and `#Lline-Lend`; the first
line is revealed. Existing literal filenames win before suffix interpretation.
Local resolution retains workspace/symlink confinement and foreign-path/Host
fallback behavior. URL links are not rerouted into Files.

## Image preview

Local PNG/JPEG/WebP/BMP and GIF first frames use the existing native decoder.
SVG uses MyGo's native vector renderer. ICO files with an embedded PNG frame
show the largest such frame. No new codec or external process is installed.
Image previews are not exposed as editable binary text. Pre-existing unsaved
text drafts, including SVG drafts, remain visible and editable.

Reads are limited to regular workspace files, 16 MiB compressed data
(1 MiB for SVG), and 32 million decoded raster pixels. The decoded visible
image is cached; reload/activation invalidates it. Inactive image bitmaps are
released from tab state.

Known limits remain explicit in the UI:
- AVIF and legacy ICO without PNG frames need the existing external opener.
- GIF is a still first-frame preview.
- Native SVG omits text, embedded images, filters and other unsupported elements.
- An image available only through the session's text-reader fallback cannot be
  decoded locally. Remote text-file fallback is unchanged.

No workspace reads, saves, external-open actions or model requests were
executed as part of implementing this module. A bounded source review covered
native API signatures, selection offsets, dirty-tab preservation, line suffix
resolution and image resource limits; the cumulative compiler check is owned
by integration.
