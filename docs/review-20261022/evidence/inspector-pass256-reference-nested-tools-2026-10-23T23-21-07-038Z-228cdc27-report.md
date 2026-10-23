# Round112 scoped installed nested-tool picker evidence

The single real acceptance execution passed 11/11 cases and 65/65 assertions on Node v24.21.0 and installed Chrome/155.0.8059.85. Ten cases and 59 assertions executed in Chrome; one separately labelled case and six assertions executed in a Node VM. Run ID: `2026-10-23T23-21-07-038Z-228cdc27`. The acceptance run started at `2026-10-23T23:21:07.040Z` and finished at `2026-10-23T23:21:10.280Z`.

The new runner is [inspector-reference-nested-tools.mjs](../../../scripts/inspector-reference-nested-tools.mjs). The full measured values, original extracted source slices and trusted input audit are in the [JSON evidence](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27.json); the concise case listing and extraction hashes are in the [text evidence](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27.txt). The retained [fixture](../../../.devdata/r112-nested-picker-2026-10-23T23-21-07-038Z-228cdc27.html) contains the exact installed picker/functions and CSS.

The Chrome fixture is a controlled file document, not the full original application, React rendering, live session data, original GUI, or Electron Chromium. Seven original installed functions, `POINTER_ACTIONS`, and `InspectorChatPicker` execute verbatim. Only the two exact installed generated picker CSS rules are applied, together with fixture-only layout and theme. Original `SessionLogModel.pick` and forward-target helpers execute separately in the Node VM; fixture source/accept/settledRoots/rows/records adapters provide matcher state. Full SessionLogModel lifecycle is outside this evidence scope.

The outer original Chat node uses native alias `tool:root`. Its recursive structure is `root -> child -> grandchild`, plus `root -> sibling`, with four identical rendered tool labels. The fixture reproduces the installed ToolCallTree relationship: each call wrapper contains its own card plus recursive `data-subcalls` descendants. The child CallIDs remain part of the same outer Chat node.

| CallID | Actual preview `(left, top, width, height)` | Actual picked nodeKey | Original Chat row |
| --- | --- | --- | --- |
| grandchild | `(126, 244, 715, 56)` | `tool:root` | `node:tool:root:` |
| child | `(95, 184, 746, 118)` | `tool:root` | `node:tool:root:` |
| root | `(64, 124, 777, 238)` | `tool:root` | `node:tool:root:` |
| sibling | `(95, 306, 746, 56)` | `tool:root` | `node:tool:root:` |

The grandchild leaf preview is exactly its own CallID wrapper. The child preview contains the grandchild and excludes the sibling. The root preview contains the complete tree. The child indentation gutter `(120,272)` picks `child`; the gap between the child subtree and sibling `(468,304)` and the gap before the first child `(468,182)` pick `root`. Every pick retains `nodeKey: tool:root`. Exact original `matchChatNodeRow` resolves each accepted CallID to the owning root Chat row.

Real CDP clicks were trusted native Chrome events caught and prevented by the installed mask. Accepted and rejected picks produced no underlying pointer/mouse/click action. A rejected child kept the same gesture armed; the next grandchild click completed it with a distinct CallID. The underlying grandchild button worked after the mask released. The clip-child preview independently matched the intersection of its wrapper, step clipper, transcript/composer and browser viewport. A mounted offscreen call retained a match identity but had no `visibleChatRect`. Arming, hover, selection, diagnostic reads and forward enumeration did not change transcript markup, layout, scroll extents or scroll position.

The original Log reverse matcher received child/grandchild/sibling CallIDs enriched with root anchor sequence `100`, turn `7` and step `2`. The fixture log contained root `tool/call` at sequence `100` and three real-type `tool/ptc-dispatch-start` entries at `110`, `120`, and `130`, with no exact child `tool/call`. All three reverse picks returned `event:100`. A separately identified exact `tool/call` control at `150` returned `event:150`, preserving the original priority rule. No matching semantics or source text were changed to force child PTC reverse selection.

For forward selection, original `sessionLogChatTarget` described PTC child sequence `110` as `callId: child, rootCallId: root`. Original `resolveChatRevealTargets` retained those identities and added `nodeKey: tool:root`. Original browser `findChatTargets` placed the exact `childCall` wrapper before both the outer root Chat node and root-call fallback.

Screenshots show the actual isolated Chrome state: [leaf](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27-leaf.png), [child subtree](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27-subtree.png), [root tree](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27-root.png), and [clipped nested call](inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27-clipped.png).

The runner allocated CDP port `52363` through an exclusive loopback bind, spawned Chrome PID `8184` with `stdio: ignore`, and verified CDP browser PID, exact unique profile and exact port against the launch. The only page request was its owned file document. All 54 observed input events had `isTrusted: true`; no remote page request, runtime exception, or protocol error was recorded. `Browser.close` targeted only the verified owned browser. The browser exited with code zero, all nine observed owned process IDs were gone, the CDP port was released, and the exact run profile directory was removed after path and ownership verification. Original PID `20012`, backend PID `16008`, GUI `http://127.0.0.1:3080`, and their profiles were never targeted. Fixture and evidence are retained; the new profile-base directory is empty.

Exact SHA256 values:

| Artifact | SHA256 |
| --- | --- |
| Runner | `873edc9aaa48678c6c14e62aa220ae2aaf6de803530bfe5f396125d7791fb3f0` |
| JSON evidence | `fae392a523f796fa064301fccc03ebeb5d5e8b87d5603c2b552465b275037304` |
| Text evidence | `177f56a6bf3e0b717c1ebbcf6ac02c125e8fac80a33ee8268388425189d57143` |
| Retained fixture | `802b346eed5449948c65528180c7d9e4f4d9afdaee2ca9a0e0a86acf30052d00` |
| Installed Inspector bundle | `62443553c0dec20f1d1a21b02a55822877c734059ea1c18d5a414ddafac657a0` |
| Exact applied picker CSS | `d63382b629c63048449b6c2fa464a03dfa4d9a7522f31fbe7315fee04572ba07` |
| Installed ToolCallTree | `a506bbb34deff21a498c309155f633a013dc947a352cd8cf89a51cb7eb268835` |
| Installed Chrome executable | `de999bce6a1c954df71a8f15bf751b336172f55ac1a0325bb0580cbc2121268c` |

Every verbatim region's source lines, UTF-16 offsets, byte count and SHA256 are retained in JSON. Installed Inspector bundle, installed ToolCallTree, existing Round111 runner and new Round112 runner were hashed before and after execution and remained identical. No existing script/evidence or native implementation file was edited; no install, model request, app server, commit, or application build was run.

The first syntax check preceded insertion of acceptance cases; the second checked the completed runner. Both syntax-only checks passed. Exactly one browser acceptance run executed. Job `pwsh-ec820f2452dc4f1e` completed with exit code zero and was collected before this report. The passing acceptance was not retried.

Before runner creation, one oversized PowerShell setup call was rejected by process creation with the exact output below. It did not launch Chrome or execute the proposed command, and no acceptance evidence existed yet. The setup was split into shorter scoped commands; the original installed source and matching semantics were unaffected.

```text
Error: spawn ENAMETOOLONG
```

An initial targeted edit was rejected by file-observation policy because the just-created copy had not yet been read by the read tool. It returned the following message. The file was then read and the same scoped edit applied; no code ran during the rejected edit.

```text
Error: cannot modify "C:\AI\WhaleIsle-MyGo\scripts\inspector-reference-nested-tools.mjs": file has not been read — read the file, then retry
```

Collected acceptance output:

```text
PASSED nested-fixture-wrapper-shape-and-mask-no-layout-change: 5/5 assertions
PASSED grandchild-leaf-preview-callid-and-chat-root-record: 7/7 assertions
PASSED child-subtree-preview-callid-and-chat-root-record: 8/8 assertions
PASSED root-full-tree-preview-callid-and-chat-root-record: 8/8 assertions
PASSED sibling-leaf-preview-callid-and-chat-root-record: 7/7 assertions
PASSED nested-gaps-use-nearest-containing-parent: 7/7 assertions
PASSED rejected-child-stays-armed-then-grandchild-accepts: 4/4 assertions
PASSED clipped-nested-call-offscreen-and-no-layout-change: 6/6 assertions
PASSED node-vm-original-log-pick-root-anchor-fallback-with-ptc-start: 6/6 assertions
PASSED forward-original-findChatTargets-exact-child-before-root: 3/3 assertions
PASSED page-input-and-original-source-integrity: 4/4 assertions
{"status":"passed","passedCases":11,"failedCases":0,"notVerifiedCases":0,"passedAssertions":65,"failedAssertions":0,"chrome":"Chrome/155.0.8059.85","cleanup":true,"evidence":"C:\\AI\\WhaleIsle-MyGo\\docs\\review-20261022\\evidence\\inspector-pass256-reference-nested-tools-2026-10-23T23-21-07-038Z-228cdc27.json"}
[status: completed, exit code: 0]
```
