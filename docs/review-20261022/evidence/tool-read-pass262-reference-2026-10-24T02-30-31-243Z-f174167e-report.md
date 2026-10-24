# Round118 installed ReadBlock reference

PASSED: 10/10 cases; 60/60 assertions. [Compact metrics](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e.json>) · [Unique installed source/CSS/model table](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e-source.json>) · [Runner](<C:/AI/WhaleIsle-MyGo/scripts/tool-read-reference-chrome.mjs>).

Manual installed ReadBlock/CodeCard/CodeToolbar/FoldToggle and ToolRow containment, verbatim current installed CSS and baseline light tokens, synthetic read windows. Explicit local fold/wrap/copy handlers and an in-memory clipboard adapter. Exact installed pure read-card functions execute source-only in VM, without importing the application.

| Surface | Verified installed CSS / behavior |
|---|---|
| Caller | CHAT_READ_MAX_LINES=8; primitive own default16. ToolRow.bodyWrap flex column directly contains ReadBlock; no bodyScroll260 cap. Caller readBody margin4px 0 4px 4px. |
| Card | code font11px/19px; border0; radius-lg16px. Initial data-code-wrap=false. At620px row width, read card616px. |
| Toolbar | 42px high; padding10px 18px 8px 22px; font11px/18px text family; heading/header gap12px; actions gap4px; native controls24×24, glyph14. |
| Language/title/status | Language code family; unknown/absent grammar displays Code block. Title/status text family11px/18px tertiary; title ellipsis, title attribute exact; status nowrap margin-right8px. Status iff returned count<totalLines. |
| Body | padding6px 22px 20px; font11px/19px; overflow-x:auto. No pre element or code block number counter. |
| Read row | flex; min-height:1lh resolves19px, even empty text; white-space:pre. Content min-width0 and primary color. |
| Gutter | max(number digits across full window,3)ch; margin-right12px; right aligned, tertiary, user-select:none and aria-hidden=true. Four-digit window uses4ch for every row. |
| Fold | hidden=returned-maxLines; collapsed head ceil(max/2), tail remainder. At8 cap, head4/fold/tail4. Native button font inherit11/19, height19; width100%; padding-left calc(gutter+12px); border0; tertiary then secondary on hover. |
| Wrap | false initially; trusted click enables pre-wrap plus content overflow-wrap:anywhere, producing multiple19px lines without horizontal overflow. aria-label remains Wrap lines; aria-pressed/tooltip/glyph change. |
| Copy | Full returned window raw text joined LF, including hidden middle and preserved final empty-text line; no gutter/banner/fold chrome. Empty file omits copy; one empty-text row still has Copy. Successful local adapter shows Copied/check for1000ms; feedback flag blocks another click; refused adapter leaves Copy. |

Measured short: body 616×64, card 616×106, root 138px. Empty file body/card 26/68px; single blank-text row body 45px. Gutter3ch=18.140625px;4ch=24.1875px. 14of100 window retains37..40/fold6/47..50 at rest; expand shows37..50, then collapse control. Expanded body 311px and collapsed 197px.

Source-only validation executes the exact emitted readMeta/validReadCall/readCardModel/readCallLine, parser and path helpers, then independently executes the current ui-tool client.js read-model fragment against all 53 cases. Only a settled root success read with valid object arguments, one text result, exact LF file envelope and valid persisted metadata becomes a read card. Numbering must strictly increase from at least offset and stay within totalLines; gaps are allowed. Metadata text/path controls the card; it is not cross-checked against envelope body/path or argument offset. Empty metadata path and an empty window with offset past total are accepted by installed validation. Table records these facts without imposing new validation. No application model imports or Host execution occur.

Installed authority: JZ ReadBlock, r4 CodeToolbar, Vw FoldToggle in the installed web JS; CSS suffix1ay0u/1pq26; ui-tool ToolRow/read-family/read-row/read-card/label adapters and current theme. Read-only reference TypeScript explains symbols; installed bytes/maps are authoritative.

The fixture explicitly implements local fold/wrap/copy transitions and a memory clipboard. It does not run React, Shiki/highlightLines, lazy grammar loading, viewport observers, actual motion/presence, Tooltip portals, original callbacks, real file operations, application backend or Host dispatch. CDP Tab only tests focus; no native Enter/Space activation claim. Pending clipboard races are not tested; only the copied success flag duplicate guard. No OS clipboard is accessed.

Source bytes unchanged: true; startup count 1; measured DPR 1; cleanup true. All observed owned Chrome PIDs exited, the exact debugging port released and the owned profile removed when cleanup passes. Protected Electron20012/backend16008/LockApp12516 and GUI3080 untouched. Combined JSON 182205 bytes, target<250000.

[initial](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e-initial.png>) · [copied](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e-copied.png>) · [expanded](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e-expanded.png>) · [wrapped](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-read-pass262-reference-2026-10-24T02-30-31-243Z-f174167e-wrapped.png>)
