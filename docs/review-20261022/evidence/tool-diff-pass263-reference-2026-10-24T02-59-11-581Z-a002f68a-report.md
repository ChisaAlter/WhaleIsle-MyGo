# Round119 installed DiffBlock reference

PASSED: 10/10 cases; 50/50 assertions. [Metrics](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a.json>) · [Unique installed source](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-source.json>) · [Runner](<C:/AI/WhaleIsle-MyGo/scripts/tool-diff-reference-chrome.mjs>).

Manual installed DiffBlock/CodeToolbar/FoldToggle DOM, direct ToolRow.bodyWrap containment, verbatim installed CSS and current light/dark tokens, synthetic diffs and explicit local handlers. Exact installed jsdiff/localHunks/buildRows/copyText and emitted/bundled diffCardModel execute source-only in VM.

| Surface | Installed behavior and measured geometry |
|---|---|
| Caller/card | ToolRow direct bodyWrap, no bodyScroll260 cap. Caller cap9; primitive16. Margin4 0 4 4; code11px/19px; radius16; border0. |
| Toolbar |42px; pad10 18 8 22;font11/18; controls24x24/icons14/actions gap4. Common extension canonical language else localized Code block; no title/status passed by DiffBlock. |
| Body/rows |Grid minmax(max-content,1fr);pad6 0 20;overflow-x:auto. All rows stretch to common content width. Row border-box,min-height19,pad0 22,pre; blank addition still19. Wrapped grid minmax(0,1fr),pre-wrap/anywhere. |
| Light colors |Add #22c55e bg#22c55e14;del#ec1313 bg#ec131314;card#f9fafb;context/path#61666b. |
| Dark colors |Add #22c55e bg#22c55e1f;del#f25a5a bg#f25a5a1f;card#1b1b1c;context/path#cfd3d6. Both themes add/del inset3px stripe same as foreground. |
| Folding |Flatten paths/gaps/content first;9 no fold;10 head5/fold1/tail4. At9 body197/card239;10 folded body216/card258. Expand all then collapse control; no outer260 cap. |
| Copy |Full source VM row text includes folded paths/gaps and every line. del=-space,add=+space,context=two spaces;path/gap verbatim;LF joined. Local memory success Copied/check1000ms, copied guard;refusal leaves Copy. |
| Empty/language |Empty diffs renders null. Unchanged and empty-create single diff retain path-only with Copy. Same ts/TSX label typescript;different/absent languages localized fallback. |

Source-only VM runs exact installed jsdiff/localHunks/buildRows/contentLines/copyText/totals against 22 cases including LF/CRLF/bareCR, ambiguous LCS, distant context gaps,256 edit boundary,full-fragment fallback and large CJK/astral/combining Unicode without truncation. Diff-card model runs 64 rows independently through emitted functions and current ui-tool client.js fragment. It validates preparation/root/running/settled dispatch,write fallback,edit applied metadata,legacy Generic settlement,recorded-path remapping and malformed metadata; accepted current edge cases are preserved. 14 emitted/bundled language cases and current en/zh label adapters also execute.

The reference is manual DOM with verbatim installed CSS and explicit local handlers. No React,Shiki,Host,file operations,OS clipboard,native button activation,motion,tooltips or application backend executes. Browser input is CDP only after exact PID/profile/port verification. Protected Electron20012/backend16008/LockApp12516/GUI3080 untouched. One Chrome startup;no accepting rerun.

Source integrity true;DPR 1;cleanup true;combined source+metrics JSON 197340 bytes (<250000).

[light](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-light.png>) · [dark](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-dark.png>) · [copied](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-copied.png>) · [expanded](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-expanded.png>) · [wrapped](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-diff-pass263-reference-2026-10-24T02-59-11-581Z-a002f68a-wrapped.png>)
