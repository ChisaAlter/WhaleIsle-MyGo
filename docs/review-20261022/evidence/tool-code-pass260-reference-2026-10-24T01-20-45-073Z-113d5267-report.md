# Round116 installed run_code measurement

FAILED: 9/10 cases; 55/56 assertions. [Compact measurements](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267.json>) · [Unique CSS/maps/source hashes and source-only table](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267-source.json>) · [Runner](<C:/AI/WhaleIsle-MyGo/scripts/tool-code-reference-chrome.mjs>).

Verbatim installed ToolRow, CodeBlock, CodeCard/CodeToolbar and DisclosureRow/TextShimmer CSS with current installed theme tokens. Manually assembled installed containment and class relationships, synthetic run_code arguments/results. Local fixture disclosure/wrap/copy handlers; no React, app/session/model/host, actual Shiki tokenizer, tooltip portal or motion execution.

| Installed baseline | Exact declaration / measured use |
|---|---|
| ToolRow header | 24px high; code block begins +4px horizontally and +4px below header. |
| Code scroll owner | bodyScroll max-height:260px; overflow-y:auto; content-box. The codeBody override only supplies margin:4px 0 4px 4px. |
| CodeBlock | border:0; radius-lg=16px; width596 at row600. Default wrapped=true; no line numbers. |
| Toolbar | height42; padding10px 18px 8px 22px; gap12; text11px/18px; actions gap4; native buttons24×24, glyph14. |
| Header containment | bannerWrap sticky top0/z6; CodeToolbar inside; language typescript; title/status omitted. content display:contents. |
| Program pre | 11px/19px; padding6px 22px 20px; pre-wrap/word-break:normal/overflow-wrap:anywhere. Unwrapped uses pre/overflow-wrap:normal and horizontal auto overflow. |
| OUT card | Only OUT, no IN or divider/copy. Margin4px 0 4px 4px; declared .5px border uses1px at DPR1; radius16. Font11px/16px. |
| OUT section | grid max-content 1fr; gap14; padding12px 16px; max-height150px content-box gives174px outer section,176px bordered card. Sticky label top0. |

Measured short program: pre 596×64, CodeBlock 596×106, code viewport 600×114. Long program viewport 260px and OUT section 174px scroll independently; trusted wheel retained the other scroll offset and the toolbar remained at the code-scrollport top.

The source-only formatToolBody table executes the two exact installed pure functions in a VM without model imports: empty raw is null, invalid JSON remains raw, a nonempty code string is returned verbatim, and empty/missing/nonstring code falls back to pretty JSON. Thus code:"" still draws a CodeBlock; raw empty suppresses it. Output empty string still draws a label and40px OUT section; output null omits IO.

The wrap control keeps aria-label="Wrap lines" while aria-pressed and nowrap/wrap glyph change. Tooltip label metadata changes between "Do not wrap lines" and "Wrap lines". Copy reads pre.textContent after removal of exactly one terminal LF; it copies the program without the JSON envelope or line-number text, then shows Copied/check for1000ms after a successful local adapter result. Repeated clicks during feedback are ignored; a refused adapter result keeps Copy. No system clipboard access occurred.

Disclosure keyboard assertions cover only the explicit fixture div keydown handlers for Enter/Space. Tab focuses the first native toolbar button and exposes the1px ring with2px offset. Native-button Enter activation was deliberately not asserted because keydown/keyup without a char event can miss that Chrome default action. Tooltip portals, React, usePresence/motion, Shiki tokenization/grammar loading/streaming, and the original GUI/model/host remain unexecuted. The colored sample uses honest manual spans with exact installed light token colors (#d6336c keyword, #1c7ed6 constant, #495057 punctuation).

Source files and prior runners unchanged: true. Startup count: 1; measured DPR: 1. Cleanup: true (all observed owned PIDs exited, loopback port released, owned profile removed). Protected Electron20012/backend16008/LockApp12516 and port3080 were not used or mutated.

[initial](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267-initial.png>) · [wrap](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267-unwrapped.png>) · [copy](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267-copied.png>) · [scroll](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-code-pass260-reference-2026-10-24T01-20-45-073Z-113d5267-scroll.png>)

Failure retained: default-wrapped-and-click-unwrapped-controls expected the unwrapped pre outer height to be45px. Saved measurements show wrapped height121px with no horizontal overflow, then unwrapped height50px with horizontal overflow. The unwrapped pre has clientHeight45px, scrollHeight45px, scrollWidth1114px, clientWidth316px, no border, line-height19px and vertical padding6px+20px. Its5px horizontal scrollbar accounts for50px outer height. The resulting CodeBlock is316×92px and code-scroll wrapper320×100px. The assertion omitted scrollbar thickness; its failure and original acceptance JSON remain unchanged. No acceptance rerun occurred.

Copy feedback reset was measured at1001.9ms. Independent wheel offsets were code/output0/0→220/0→220/220. All18 recorded source files retained identical hashes,43 observed input events were trusted, and the only page request was the owned file fixture. The source JSON is71,546bytes and measurement JSON72,027bytes,143,573bytes combined with27 unique computed-style records.
