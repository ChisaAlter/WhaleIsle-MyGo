# 重建进度记录（2026-10-17）

依据已有审计、冻结交付和明确发布回执整理；没有新代码审查、测试或真机验收。完整记录见 [JSON](reconstruction-progress-20261017.json)。

- 历史工作项与能力索引存在重叠，不是全项目分母，不给总体完成百分比。
- source-implemented、已冻结、已推送、CI 通过和真机验收分别记录。
- 当前缺口只见 remaining；旧发现保留在 historical_finding / audit_states / superseded_progress。
- 范围：Windows/macOS 桌面及已有 Android 远程；Linux 不纳入产品验收。

## 当前关键状态

### WT05 · partial

- 已有源码：Read-only native grid, formula source, selection/TSV copy and available XLSX styles.
- 已有源码：Frozen layout stage adds merged intersections, saved/default row/column sizes, base XF font/number-format/alignment inheritance and wrapText.
- 已有源码：Frozen owner stage routes to existing exact-resource original workbook renderer when its capability is live; native choice and revocation fallback retained.
- 已有源码：Frozen hidden-dimensions stage: XLSX row/column hidden and zeroHeight flags, explicit overrides, no grid space/header for hidden dimensions, merged-cell selection preserves source anchor and raw TSV.
- 已有源码：Read saved fillId/borderId including base-XF inheritance and explicit apply flags
- 已有源码：Render saved solid fill RGB; native selection highlighting remains visible
- 已有源码：Resolve source RGB/ARGB, named theme slots from xl/theme/theme1.xml including saved system lastClr, and workbook-supplied indexedColors
- 已有源码：Read four source border edges and render supported solid/width, dotted/dashed/dash-dot and double variants
- 已有源码：Disclose unsupported source paint in formula metadata; unresolvable color never receives a guessed substitute
- 已有源码：Preserve stored values/formulas, readonly selection/copy, hidden dimensions and bounded paging
- 当前剩余：Native tint adjustments, automatic/system colors without saved fallback, absent default indexed palettes and nonstandard theme relationship targets are unresolved rather than guessed
- 当前剩余：Gradient/pattern fills, diagonal/slanted borders are not rendered
- 当前剩余：Merged outlines use the anchor style; collecting per-edge border contributions from covered cells is not implemented
- 当前剩余：Styles for wholly absent cells inherited only from row/column definitions are not synthesized into the current sparse cell projection
- 当前剩余：Native automatic row height/auto-fit is not implemented.
- 阶段 layout：verified-pushed；484a428700474d746ca365eaf4153e270ab361a7
- 阶段 owner：verified-pushed；3b8474fed7729c73a3f4dadbb568c0aecba96943
- 阶段 hidden-dimensions：verified-pushed；25bc4844b99b50b150742081af971aec44df7f34
- 阶段 source-fill-border：verified-pushed；fadb4803df326caaec5d246f29d693c522eca306
- 阶段 native-grid：verified-pushed；25be9a89299b2a0802848f0cf3dce5d6ea929437

### CP08 · partial

- 已有源码：Conversation body accepts files through existing scoped composer intake, with drop feedback.
- 已有源码：Persistent navigation rail accepts files using existing composer FileDropTarget and addComposerFiles real intake
- 已有源码：Capture exact Store/selectedID/draft/ComposerHostGeneration; reject stale or global-panel callback
- 已有源码：Main child Conversation disables parent navigation drop; resource docks and WebViews outside target subtree retain own routes
- 已有源码：Native innermost target routing ensures no parent/child duplicate file delivery; typed internal dock drag not attachments
- 已有源码：Caption/titlebar safe file-drop target is published, using the main composer owner guard. Arbitrary unconsumed regions still have no global composer fallback.
- 当前剩余：Arbitrary unconsumed resource/child/WebView regions lack authoritative composer ownership in Window.OnFileDrop; no global fallback was added.
- 阶段 conversation-target：verified-pushed；cdabe6a1618795627d68b1ca4b88b6c31fc955c4
- 阶段 navigation-rail：verified-pushed；8bbff06d7a5fde1886a1753ddf5faedf559c6a5e
- 阶段 caption-safe-target：verified-pushed；3280cb6a0922d4e4573b7ba16fc6bfab7f8604ef

### CH12 · partial

- 已有源码：Genuine original OpenUI runtime bundle located; no arbitrary replacement bundle used.
- 已有源码：Stage 1 source integration and companion native helper event bridge published per parent confirmation: isolated protocol/resource/state handling, exact session owner, code-fence entry and native event bridge.
- 已有源码：Stage 2 inline embedding and same-instance program/streaming updates are published: retains live shell state/theme, deduplicates identical updates, latest-only queue bounded to one pending value and at most 10 Hz.
- 已有源码：Native stage 2 Windows/Darwin Go cross-compiles and targeted protocol/shell/UI checks are recorded; this does not certify physical inline input/clipping/lifecycle.
- 已有源码：Active forms use their owner lifetime instead of an arbitrary deadline; arbitrary global renderer and persistent-state count ceilings were removed, with per-message resource bounds retained.
- 已有源码：Typed public resource adapters and exact-owner API request/response routing are published. Only fixed supported services and typed parameters are accepted; arbitrary URLs and general iframe networking remain unavailable.
- 已有源码：Image production code resolves supported Wikimedia HTTPS image results to validated image data URLs while retaining source/attribution. API response transport advertises its actual larger budget independently of unchanged event/state limits.
- 已有源码：Independently authored OpenUI SDK-based runtime is embedded by default from maintainable tools/openui-runtime source; original installation or user-supplied viewer is no longer required. 53 registered original-contract components and 59 application functions are present, with exact variant limitations documented rather than claiming complete behavioral parity.
- 已有源码：Owned raster Map uses fixed ArcGIS tile data URLs and the existing typed OSRM route adapter; parent /api/map-tile route is implemented without granting iframe network or worker access. Source, lockfile, production asset, provenance and complete notices for 79 browser dependencies are committed together.
- 已有源码：Code and Markdown fenced code use local AST syntax highlighting with explicit grammar aliases and full-source fallback; no code execution or clipboard added.
- 已有源码：Six chart variants expose real hover/focus data details and Arrow/Home/End/Escape navigation, including original values, series, stacked ranges or proportions.
- 已有源码：Public adapter successful JSON responses are reused across viewers with original-scope insertion-order caches: wiki 500, normalized image queries 500, normalized route sets 200; each hit receives current instance/request IDs and failures are not cached.
- 已有源码：Owned raster Map implements continuous anchored zoom and pan for mouse and touch gestures, preserving actual projected coordinates and tile-source selection.
- 当前剩余：HtmlView and YouTube remain absent. Clipboard/external-link actions and external Image/Markdown image URLs remain unsupported. Query/Mutation business providers are not supplied.
- 当前剩余：Other component variants, state/function edge semantics and action confirmation differences are detailed in tools/openui-runtime/COMPATIBILITY.md; registry counts are not full parity. Undefined stacked pie/donut/scatter combinations are parameter boundaries, not proven original-only missing features.
- 当前剩余：Public JSON process caching and owned raster tile caching are implemented. Original browser HTTP bitmap-cache behavior is not claimed; the optional legacy MapLibre bundle remains unshipped/unsupported.
- 当前剩余：Only latest received state is persisted; undelivered queued state can be lost on close. Native compilation and actual renderer/resource/state/action acceptance for the newly embedded runtime remain unverified.
- 阶段 stage-1-runtime-and-native-bridge：verified-pushed；abf32a3a15c31e007b19210298ade8c42a78e3e0
- 阶段 inline-and-streaming：verified-pushed；551649d8afe62ff3964dc94e1e0f610faacea2d8
- 阶段 owner-lifetime：verified-pushed；b30c5c4b3c5265c54be262f63cfeea58e89e27f7
- 阶段 remove-count-ceilings：verified-pushed；1edab767c7d759d5804d432fb3bbd9b686a41bed
- 阶段 typed-public-resource-api：verified-pushed；ea4e4782dac48f6a3669688b8c4e36f12e042d7f
- 阶段 fixed-source-display-images：production-pushed-tests-unpublished；a0b0a6bc9b87a0fa4fc64c3a723736385685a398
- 阶段 independently-authored-embedded-runtime：verified-pushed；86104c1f965d5b326d5a8dd01f75364940fe073e
- 阶段 code-markdown-highlighting：verified-pushed；04684a6a3983da3771134e7eb0fc21cf078d16ea
- 阶段 chart-tooltip-keyboard：verified-pushed；a6d384a515a52e10bc7643e9654f55161ff77d06
- 阶段 public-adapter-process-cache：verified-pushed；93555b5a91686df4edd489e311bc99b261d24757
- 阶段 map-wheel-pinch：verified-pushed；407ffc53dfec71c55bca9de25c1102817791ad8e

### PR07 · source-implemented

- 已有源码：Keyboard calendar navigation and price modal dismissal.
- 已有源码：Frozen roving focus stage adds one Tab stop, latest-day first focus, remembered arrow focus on return and Shift-click range-end replacement/shrinking.
- 已有源码：Frozen week-column stage: Monday-first weekday rows and week columns, left/right ±7 days and up/down ±1 day, focus scrolling, original month-marker rules; roving and Shift range selection retained.
- 阶段 calendar-price-actions：verified-pushed；d77ddf6c58c0dc4b9186265b4165447a24715ecc
- 阶段 calendar-roving：verified-pushed；73479a9853eb56fd7309f7952918d7a523e8ffac
- 阶段 week-columns：verified-pushed；51031f5b22c87c31baea5831660dc1cc5cad8e15
- 阶段 pricing-fixture：verified-pushed；574300e2c58090a97e6a6e24178b0ee56af06d12

### PR06 · source-implemented

- 已有源码：Frozen supervisor-driven compact boot scene with loading state, genuine error/retry controls, supplied log drawer and ready collapse.
- 已有源码：Frozen curtain stage: caption-below boot overlay, finite 600ms split reveal on Ready, reduced-motion handling and real controls retained; light/dark sky/ocean/stars decoration.
- 阶段 boot-curtain：verified-pushed；ff920db5383493f6f865efd560f602aadd1b0034
- 阶段 startup-scene：verified-pushed；6b4903e69bda4b401286e5f8725b2d49e09aa71e

### AX203 · source-implemented

- 已有源码：Published optional multi-workspace scope allows explicit selections from verified live local Host/Project roots; default authority remains unchanged, selections are run-scoped/revocable, and no actual user grant was enabled during development.
- 阶段 explicit-optional-workspace-scope：verified-pushed；4f5bc31ec2f4205c91f8abef59e962c641944e77

### CH68 · partial

- 已有源码：Opened child panes retain exact occurrence/root/Host/client/address read leases across main-parent navigation. Exact close, root/ancestor/child removal, Host replacement and authority reset revoke the owner without pane-ID aliasing.
- 已有源码：Explicit Send, image changes and single-child Stop use the original retained owner and live original-parent availability. Approval, denial, question, plan, retry and explicit submission use an occurrence-bound interaction facade with delivery-time revalidation.
- 已有源码：Persisted nonlegacy direct-root readers restore before their tabs paint after authoritative catalog verification. Newly saved nested readers persist ID-only verified lineage and validate every edge before following; finite acyclic lineage has no added depth cap.
- 已有源码：Verified restored readers remain read-only in background/shared paths. Each explicit gesture can obtain its own revalidated action context without requiring Reopen or permanently upgrading the lease; automatic approval is unavailable.
- 当前剩余：Old nested records without saved verified lineage cannot be reconstructed automatically; legacy records without verified Host identity require explicit opening from a current catalog. No guessed ancestry or global tree scan is used.
- 阶段 independent-owner-readonly-retention：verified-pushed；b56162e3a3910eee7d9274d6385955ca38e54157
- 阶段 hidden-reader-retention：verified-pushed；fc526d303168afd7c2443bdac7b00d0e644f81d2
- 阶段 explicit-retained-prompt：verified-pushed；05d6342b298a56de4673b27c709cb8c71b2ea597
- 阶段 explicit-retained-interrupt：verified-pushed；9f3b9714b02919da6f311f7ee34ceeef727c5057
- 阶段 never-painted-restoration：verified-pushed；e2539b3a2c2fd6ec70a852abb974b9df173ab245
- 阶段 verified-saved-lineage：verified-pushed；57868a63e4041a4bb9dc134248b951838331170b
- 阶段 finite-lineage-without-depth-cap：verified-pushed；3a4e41bbacb27a682c186504a54eff68d0c85aa1
- 阶段 explicit-restored-gesture：production-pushed-tests-unpublished；0e70ae5a3f4fad583a939666a45455a17100cd04
- 阶段 retained-interaction-facade：verified-pushed；98ad22eaa2823f6a5ed02218e01b73453b916f58

### WT08 · source-implemented

- 已有源码：Stage 1 frozen: remote absolute-path copy and remote Reveal/Open With target resolution, narrow toolbar wrapping, original noninteractive directory/filename path label and browser chrome localization.
- 已有源码：Stage 2 frozen: existing Go-owned specialized Browser tools and emitted feedback use English/Chinese, including size/zoom/profile/annotation/recording/media/preview/restoration/DevTools.
- 阶段 toolbar-chrome：verified-pushed；7b94b14eefa031e403a3d3000cb3be512b2a5676
- 阶段 tool-localization：verified-pushed；4b94fd81267ac5fac606b9d4f3b909db27a2bb09

## 真实桌面验收

所有相关 work item 的 runtime_status 仍为 runtime-unverified。原生窗口、输入/裁切、Finder/Explorer 拖放、实际 Host 模块可用性、Android 远程交互等未因单元检查或 CI 成功而获得真机验收。CH12 stage 1 成功 CI 不代表 stage 2；两阶段 CI 分别记录。

## 发布回执补齐

发布状态只依据完整提交号和已核实的远端记录。源码交付、提交、原生编译和真实运行验收分别记录；缺少独立提交证据的条目保留空值，不以猜测补齐。

## 已记录工作项索引

以下列出本次汇总的全部工作项。它们包含拆分项与历史范围外项，不能用条数计算全项目完成率；详细证据、剩余差异及运行状态以 JSON 为准。

| ID | 工作项 | 源码状态 | 已核实提交 |
|---|---|---|---|
| AX201 | Desktop phone gateway mode, listen port, bind scope and legacy relay address editors | 有实现证据 | [7588816f](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7588816fec18c56d82f3fdb99154ca1de25c0aa0) |
| AX202 | Unmodified native Android stacked commit/push/PR creation | 有实现证据 | [16ff6d90](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/16ff6d9087b14a2839edd1fda34da863daff265d) |
| AX203 | Phone Git on a phone-selected workspace independent of desktop selection | 有实现证据 | [4f5bc31e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/4f5bc31ec2f4205c91f8abef59e962c641944e77) |
| AX204 | Historical original Android patch compatibility documentation (out of MyGo scope) | 历史范围外 | 未核对独立提交 |
| B04 | B04 | 有实现证据 | [0a64b564](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/0a64b5644848b60b9b3d4bd9aa213b540e4151c8) |
| CH01 | Archived search, restore and delete | 有实现证据 | [e8bae5ee](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/e8bae5ee77ea7d5850034e7c82e67a4fb22edb6f) |
| CH02 | Context occupancy and composition | 有实现证据 | [5456453e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/5456453e672357d71428d609a761fc6ab05a3fac) |
| CH03 | Injected context distinct from human speech | 有实现证据 | [7598ee85](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7598ee854839985111bca0c2bdbe6ec673c3e68d) |
| CH04 | Full trajectory search/details/copy | 有实现证据 | [de4e6fbe](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/de4e6fbed47cce4f7ebeace2e381c9ec75f6eb33) |
| CH05 | Automatic compaction checkpoint and summary | 有实现证据 | [fff62bcd](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fff62bcd6f11a318aae45518c5c6272e5462941d) |
| CH06 | Searchable provider-grouped model picker | 有实现证据 | [1969c738](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/1969c73860dd71bad333cb2dd11a667855df80c3) |
| CH07 | Drag reorder session rows | 有实现证据 | [375654b1](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/375654b1cbfe4420664b4576e9925d0bdd9622d3) |
| CH08 | Transcript width handles and persisted preference | 有实现证据 | [83f23f86](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/83f23f86c2655517020d58d838c07f5e2546b9a5) |
| CH09 | Markdown reasoning rendering | 有实现证据 | [e33c95fa](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/e33c95fadf46e32dd5ede75a17d9e4acb774f278) |
| CH10 | Advanced trajectory schema/request/range inspection | 有实现证据 | [fb1b0a62](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fb1b0a62c21a8334d319e7e2ab2a5658f37c7b7b) |
| CH10.inspect-call | Inspect a tool call directly in trajectory | 有实现证据 | [facab7db](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/facab7db9840986fbe0911e7128cc9ff8ca5d0a3) |
| CH11 | Background completion unread indicator | 有实现证据 | [871fcd2d](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/871fcd2d029074ddd5f497ec66b00caed547bcf4) |
| CH12 | Assistant OpenUI code-fence interactive runtime | 部分实现 | [407ffc53](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/407ffc53dfec71c55bca9de25c1102817791ad8e) |
| CH13 | Sidebar attention state for approval/question/plan review | 有实现证据 | [871fcd2d](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/871fcd2d029074ddd5f497ec66b00caed547bcf4) |
| CH14 | Undo archive from completion notification | 有实现证据 | [d11d1bd8](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/d11d1bd8d4802a13aae0f65c9e697fc94a74d26a) |
| CH15 | Main session ancestry breadcrumb navigation | 有实现证据 | [5b0a5f2d](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/5b0a5f2d60943efeab77b319dd23d3ca8f1d1837) |
| CH50 | Timed questions: countdown, focus pause, first-edit hold, take-time action | 有实现证据 | [0a64b564](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/0a64b5644848b60b9b3d4bd9aa213b540e4151c8) |
| CH51 | External catalog entries remain labeled and non-openable | 有实现证据 | [7ca90540](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7ca9054005af4dca714031e6c06971e5dc487716) |
| CH52 | Approve once or reject, with delivery failure retry | 有实现证据 | 未核对独立提交 |
| CH53 | Scoped approval Enter/Escape and IME/button exclusions | 有实现证据 | 未核对独立提交 |
| CH54 | Localized approval reason follows UI language | 有实现证据 | [1bc8ea0a](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/1bc8ea0af1f4b8bd919c02605993c2aafda1c39f) |
| CH55 | Question multiline custom answer and Enter/Shift+Enter flow | 有实现证据 | [581ac5b1](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/581ac5b1dbf3504bfc69cc4debc9221f2090de74) |
| CH56 | Question wizard progress, previous/next, auto-advance, collapse/expand | 有实现证据 | [983de311](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/983de31143abc686bab47d6dc6ecef6eef52343a) |
| CH57 | Hide/reopen a question while preserving its draft and waiter | 有实现证据 | 未核对独立提交 |
| CH58 | Freeze question input while answer/cancel is in flight | 有实现证据 | [581ac5b1](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/581ac5b1dbf3504bfc69cc4debc9221f2090de74) |
| CH59 | Continue timed task and submit a recovered late answer | 有实现证据 | 未核对独立提交 |
| CH60 | Read back settled/queued answers with full question context | 有实现证据 | [5e61bba7](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/5e61bba730aeb0c315e2e4943e76b31cbddd7799) |
| CH61 | Expandable, copyable structured late-reply transcript bubble | 有实现证据 | [5e61bba7](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/5e61bba730aeb0c315e2e4943e76b31cbddd7799) |
| CH62 | Markdown-rich question detail, code wrapping and code copy | 有实现证据 | [1bc8ea0a](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/1bc8ea0af1f4b8bd919c02605993c2aafda1c39f) |
| CH63 | Plan review: preview/open plan, discuss changes, explicit approve, retry | 有实现证据 | 未核对独立提交 |
| CH64 | Current-session agent roster, labels, modes, running state and refresh | 有实现证据 | 未核对独立提交 |
| CH65 | Open addressed child as the main Conversation | 有实现证据 | [fc706e71](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fc706e71c2365f09e08cbd85421763b96c2c6203) |
| CH66 | Observe child live assistant/thinking/tool output with reconnect | 有实现证据 | 未核对独立提交 |
| CH67 | Read earlier child history and return to latest | 有实现证据 | 未核对独立提交 |
| CH68 | Open independent child chat aside, retain/restore and close it | 部分实现 | [0e70ae5a](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/0e70ae5a3f4fad583a939666a45455a17100cd04) |
| CH69 | Handle continuable-child approval/questions within its Conversation | 有实现证据 | [581ac5b1](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/581ac5b1dbf3504bfc69cc4debc9221f2090de74) |
| CH70 | Continue a child with text/images through its dedicated route | 有实现证据 | [f536bdcf](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/f536bdcf7f123d13a70213baf16611156a58a291) |
| CH71 | Parent availability gates child continuation while Stop remains available | 有实现证据 | [58ba107b](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/58ba107b2b1dd0f009e912f5d1a405e78dc316b0) |
| CH72 | Interrupt a running continuable child | 有实现证据 | 未核对独立提交 |
| CH73 | Browse nested agents, parent/sibling lineage, expand branches | 有实现证据 | [6ca3f24e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/6ca3f24eaf284f0cfdf1b930f312778416b1c3a1) |
| CH74 | Agent activity completion, token total and active duration metrics | 有实现证据 | [f98a5be0](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/f98a5be065159b6791c12db314c14607cd275e53) |
| CH75 | Keyboard-accessible catalog tree and root counts | 有实现证据 | [f98a5be0](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/f98a5be065159b6791c12db314c14607cd275e53) |
| CH76 | Read child workspace references, follow links, paginate text safely | 有实现证据 | 未核对独立提交 |
| CH77 | Preview child image attachments with child authorization | 有实现证据 | [6ea15607](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/6ea156071cf9d2aa3a78048b922465eb30a9fcb1) |
| CH78 | Open child HTML/PDF documents through addressed preview | 有实现证据 | 未核对独立提交 |
| CH79 | Read child spreadsheets with workbook controls | 有实现证据 | 未核对独立提交 |
| CH80 | Inspect child presented files/changes, diff, open/reveal and preview | 有实现证据 | 未核对独立提交 |
| CH81 | View current-session background jobs and status/detail | 有实现证据 | 未核对独立提交 |
| CH82 | Withdraw stale approvals and keep them isolated to their session | 有实现证据 | 未核对独立提交 |
| CH83 | Explain one-shot/unknown child read-only status | 有实现证据 | [58ba107b](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/58ba107b2b1dd0f009e912f5d1a405e78dc316b0) |
| CH84 | Question headers and every option description remain visible | 有实现证据 | [1bc8ea0a](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/1bc8ea0af1f4b8bd919c02605993c2aafda1c39f) |
| CP01 | Move/select the text caret with arrows and standard boundary/word editing | 有实现证据 | [24480770](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/24480770fecc0cea8feafa4c3b36c11897358346) |
| CP02 | Undo and redo draft edits through standard keyboard gestures | 有实现证据 | [24480770](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/24480770fecc0cea8feafa4c3b36c11897358346) |
| CP03 | Establish an undo boundary when a draft is sent | 有实现证据 | [24480770](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/24480770fecc0cea8feafa4c3b36c11897358346) |
| CP04 | Click a source-owned reference chip to open its preview | 有实现证据 | [5613d8c1](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/5613d8c14e1698df82ea5c6d367720ef904b0110) |
| CP05 | Open the command list from the composer + control | 有实现证据 | [9b267839](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9b267839961e18628901d3f13cf0aae5b30a0687) |
| CP06 | Preserve local non-image files as workspace/path references | 有实现证据 | [7f8959e4](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7f8959e45031a7045068c2934d68f177c1b6dc45) |
| CP07 | Paste generic clipboard files | 有实现证据 | 未核对独立提交 |
| CP08 | Drop files anywhere on the conversation document with overlay feedback | 部分实现 | [3280cb6a](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/3280cb6a0922d4e4573b7ba16fc6bfab7f8604ef) |
| CP09 | Reject dropped/pasted directories before staging attachments | 有实现证据 | 未核对独立提交 |
| CP10 | Preview draft image thumbnails and full original image | 有实现证据 | [9b267839](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9b267839961e18628901d3f13cf0aae5b30a0687) |
| CP11 | Browse many draft attachments in a horizontal rail | 有实现证据 | [9b267839](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9b267839961e18628901d3f13cf0aae5b30a0687) |
| CP12 | Show file type, byte size, upload state and progress per attachment | 有实现证据 | [93a27b25](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/93a27b25b065a5fd32cd01df190896db7b651121) |
| CP13 | Remove an attachment, including one still uploading | 有实现证据 | [b9988d09](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/b9988d09675ebafb07ab725820c0e42fc5ac83c3) |
| CP14 | Block sending failed uploads until explicit retry or removal | 有实现证据 | [b9988d09](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/b9988d09675ebafb07ab725820c0e42fc5ac83c3) |
| CP15 | Enforce projected image count, per-image bytes and total image bytes at intake | 有实现证据 | [b9988d09](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/b9988d09675ebafb07ab725820c0e42fc5ac83c3) |
| CP16 | Bare /permission opens the same preset picker | 有实现证据 | [01994d75](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/01994d757907c05f8cf266e378212a1a12eb3e9e) |
| CP17 | Select distinct permission presets with identical display names | 有实现证据 | [6e897181](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/6e89718101f476e336ab0e22c9129d632882806f) |
| CP18 | Effective plan-mode status chip with one-click exit | 有实现证据 | [47f8d3b8](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/47f8d3b8382c53fe4d338d52d721f95893720b17) |
| CP19 | Plan-specific composer prompt hint follows effective target mode | 有实现证据 | [ff41ea9e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/ff41ea9e2dcf92a4a3481b11237eca63e1eae649) |
| CP20 | Find and insert a user-invocable skill from the composer | 有实现证据 | [9b267839](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9b267839961e18628901d3f13cf0aae5b30a0687) |
| CP21 | Search skill names using ordered-subsequence ranking | 有实现证据 | [ff41ea9e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/ff41ea9e2dcf92a4a3481b11237eca63e1eae649) |
| CP22 | Invalidate composer skill choices after preset, session or connection changes | 有实现证据 | [e6418fc3](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/e6418fc31718561fb5f477c24c07a39eb4e34877) |
| CP23 | Stage a preset before session materialization; apply only to a blank session | 有实现证据 | [7ee411a4](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7ee411a4a712b87afdc0029a53dc901f9cce4d04) |
| CP24 | Explain preset choices and surface in-flight selection/refusal | 有实现证据 | [7ee411a4](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7ee411a4a712b87afdc0029a53dc901f9cce4d04) |
| CP25 | Select distinct agent presets with identical display names | 有实现证据 | [6e897181](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/6e89718101f476e336ab0e22c9129d632882806f) |
| CP26 | Escape cancels the active voice operation | 有实现证据 | [9de19861](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9de19861fb54c0346eac0d5903b8e790415dda5d) |
| CP27 | Unavailable voice button opens setup explanation and plugin settings | 有实现证据 | [9de19861](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9de19861fb54c0346eac0d5903b8e790415dda5d) |
| CP28 | Voice hardware support beyond Windows | 有实现证据 | [75c32137](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/75c321376227bb03a4619a6c669fd01af77a8bf5) |
| PR01 | Existing encrypted macOS credential migration unsupported | 有实现证据 | [9cd7d229](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/9cd7d22955a1ea581c99406c3ab118b723d40984) |
| PR02 | Authenticated embedded Platform usage/top-up and external popup compatibility | 有实现证据 | [fd1c03e7](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fd1c03e70848585262b33d7e33644e1a052a6265) |
| PR03 | Update confirmation surface and redundant-confirmation flow diverge | 有实现证据 | [6cf77164](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/6cf77164cf2bd08ab97c9b59c912b6e53e8fd1b1) |
| PR04 | Upstream desktop onboarding intentionally disabled by original | 有实现证据 | 未核对独立提交 |
| PR05 | Settings modal structure, dismissal/focus and motion replaced by full-content page | 有实现证据 | [755737c6](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/755737c6f49330e358cd59fd1835c594e891b446) |
| PR06 | Supervisor boot scene, log drawer and ready transition | 有实现证据 | [ff920db5](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/ff920db5383493f6f865efd560f602aadd1b0034) |
| PR07 | Usage heatmap orientation/keyboard and price-modal layout divergence | 有实现证据 | [51031f5b](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/51031f5b22c87c31baea5831660dc1cc5cad8e15) |
| PR08 | Windows schedule default timezone falls back to UTC | 有实现证据 | [73de5bfc](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/73de5bfcb7fa08648dba0f6ff39f078a0264e277) |
| WT01 | Saved-file always-on-top separate preview window | 有实现证据 | [a1f5ce4e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/a1f5ce4e3bf688ade87b6fc9a564958c5e3ca5cd) |
| WT02 | Explicit resource-tab close terminates Host process and releases occurrence | 有实现证据 | [8f7aa79f](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/8f7aa79f0700045b91a5f2c5c0516b32a3b9a498) |
| WT03 | Type candidates with Enter/Escape apply/cancel | 有实现证据 | [fdcff91e](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fdcff91ee0ed10689adec96b1c6450d4b9e37d23) |
| WT04 | macOS current-page Developer Tools | 部分实现 | [7cca7fae](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/7cca7fae8730aa31780817ae3559a7ce4e4dfba5) |
| WT05 | Spreadsheet grid, sheet tabs, formula bar, formatting/merged-cell presentation | 部分实现 | [fadb4803](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/fadb4803df326caaec5d246f29d693c522eca306) |
| WT06 | Document dirty/version notification; per-resource automatic refresh toggle | 有实现证据 | [a91d7df9](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/a91d7df9ca33b150260603f36f850998e00cacd7) |
| WT07 | Drag outside surface floats tab; edge/split drop target hints | 有实现证据 | [e105e245](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/e105e245258b412ed24399b515f792dd3a7eb964) |
| WT08 | File breadcrumb toolbar and direct source editing presentation | 有实现证据 | [4b94fd81](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/4b94fd81267ac5fac606b9d4f3b909db27a2bb09) |

## 本轮发布与验证边界

- 已核实开发分支头：`407ffc53dfec71c55bca9de25c1102817791ad8e`；提交条数仅为发布记录数。
- 所有真实平台验收仍单列为 unknown，不由源码、冻结或 CI 结果推定。
- CH12 尚未发布的测试文件：`internal/desktop/openui/api_test.go`。
- CH12 尚未发布的测试文件：`internal/desktop/openui/api_images_test.go`。
- CH12 尚未发布的测试文件：`internal/desktop/previewprocess/openui_api_test.go`。
- CH12 API/图片增量尚未做原生 SDK 编译；此前 inline/streaming 成功 CI 不覆盖该增量。
- CH68 尚未发布的测试文件：`internal/store/session/agent_restored_gesture_test.go`。
- CH68 尚未发布的测试文件：`internal/store/interaction/scoped_gesture_test.go`。
- OpenUI 已默认内嵌已许可SDK加自有源码的运行时及栅格地图；原版应用包未再分发。组件变体与剩余行为见 [兼容清单](../../tools/openui-runtime/COMPATIBILITY.md)，真实运行仍待验证。
