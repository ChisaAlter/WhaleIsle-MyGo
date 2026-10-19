# 原版全量功能验收矩阵（2026-10-19）

> Integration update, Windows full-function pass: this ledger now distinguishes actual owned Host/filesystem/native evidence from fixtures, live model/device/account prerequisites and unavailable original-window pixel comparison. The branch includes terminal duplex, browser callback, PDF navigation, session generation/recovery, rich-history parsers, SSH ownership, compact preview and attachment fixes. A complete isolated local folder is being built from the final clean source; package startup evidence is recorded separately and does not certify all91 rows. E/ means the task evidence directory beside this repository.

本表承接 [原版还原记录与既有 65 条组件映射](native-workspace-parity-20261019.md#windows-follow-up-component-acceptance-ledger)，为“全部功能正常用”登记实际验收范围。源码存在、单测通过、页面截图和编译通过都不能单独认证产品功能可用；不能由同组一个成功动作推定整组通过。本表不是新增功能需求，不重复派发已有修复任务。

当前工作分支为 `codex/session-runtime-compat-20261019`。记录时已知核心恢复产品为 `e537058`（runtime）、`7e54fa4`（可选 goal）、`85bde0d`（视图配置合并），构建证据见原记录的 Core availability 章。正在执行的主线仍由集成负责人维护，本表不另开运行时、会话或配置修复。

## 状态与共同验收规则

- **实际通过**：绑定具体构建、运行时、数据状态和实际操作。只认证该操作，不扩展到其他功能。
- **仅单测/夹具**：有既有定向回归或合成场景证据，实际产品路径未验收。
- **未验证**：有对应实现，但未取得本功能实际操作证据，或旧证据不能绑定当前候选。
- **部分实现**：源码规则/格式仍有明确差异；标明缺失部分，不写成工具阻塞。
- **外部依赖阻塞**：只能用于当前仍缺的具体前置条件，必须写清操作和原因；不能沿用已解除的阻塞。

“仅单测/夹具”是目前证据上限，不表示该行每个子动作都已有完整回归。既有 65 映射已登记的测试与截图继续以原表为准；本表新增的业务范围需要把具体结果逐项回填，不能由测试文件存在推定其执行通过。

**前置条件更新**：此前受限执行环境的 junction EPERM 是历史结果。新的已批准普通用户执行环境已通过固定 Node 24.21 的工作区 junction 和自有 loopback 检查；主线随后确认 runtime 定向用例 **9/9 通过、最终退出码 0**，`core-recovery-runtime-rc2` 完整 stage/verify 通过并验证 3143 条内部链接。因此本表不再把 junction 权限或 staging 记为当前阻塞。实际 Host/桌面验证仍由主线执行，不能由 staging 成功推定用户实例已恢复。

真实数据证据目前限三份已有压缩 v4 快照的严格校验与 rc.2 cold read，源文件字节未变。该证据不涵盖 Go UI 的 resume、follow、消息显示或新模型请求。旧 `dcfb14c` 独立候选曾隐藏启动并退出；当前核心恢复 EXE 不能继承为已实际启动。

每行均需覆盖适用的：首次进入/已有数据、加载/空/失败/重试、权限不足/只读、忙碌/取消、切换会话或 Host 后的晚到响应、关闭重开，以及保存冲突/重启恢复。持续状态还需核对流中断、重连和取消后无重复执行。持久写操作需隔离数据；付费请求、用户实例、账户和破坏性操作不能为补表而擅自执行。

路径简写：`P/` = 原版 `vendor/deepseek-harness/packages/client/`；`V/` = 原版 `vendor/`；`D/` = 原版 `src/`。Go 路径相对本仓库。原版参照仅只读；实际安装 runtime 的来源版本与源码 checkout 分别记录。

## 启动、运行时与恢复

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| BOOT-01 | 启动器、workspace/启动偏好、内核准备：`D/renderer/launcher*`、`D/main/window.js` | `internal/ui/launcher/`、`internal/ui/boot/`、`cmd/whaleisle/main.go` | 仅单测/夹具 | 当前候选启动到准备、就绪、失败/重试的实际操作未验证 |
| BOOT-02 | 真实 Host/Node/插件闭包选择：`V/harness-upstream.json`、原版 `scripts/production-runtime.js`、`scripts/runtime-instance-graph.js`、安装资源 | `internal/harness/`、`internal/paths/`、`scripts/session-runtime.mjs`、构建/打包脚本 | Actual isolated rc.2 stage + Host/native startup PASS | 3143 internal links, exact Node/runtime gate, real mux/workspace/history; E/core-host-check-resume.json and E/native-core-final-e21f4b8.png. Final portable package validation pending. |
| BOOT-03 | 会话日志严格准入和版本恢复：原版 session/persistence/format catalog | `internal/harness/session_runtime.go`、session runtime admission | Actual strict v4 cold read + real Host resume PASS | Three owned required-event snapshots selected and followed into Go messages; three user snapshots cold-read only and unchanged. Unknown required events remain rejected. E/core-host-check-resume.json. |
| BOOT-04 | 内核退出、启动恢复、安全模式、重启：`D/main/harness-controller.js`、`plugin-recovery-actions.js`、launcher | `startup_recovery.go`、`internal/startuprec/`、harness supervisor | 仅单测/夹具 | crash/恢复/重试及失败记录的真实生命周期未验证，不重启用户实例 |
| BOOT-05 | 旧数据导入、类别选择、冲突和中断恢复：`D/main/data-import*`、`import-transaction.js` | `internal/desktop/importer/`、`import_flow.go`、`import_recovery.go` | Actual isolated import/cancel/conflict/resume PASS (partial) | Three synthetic v4 sessions copied with selection/skip/overwrite and cancel after a committed item; original source unchanged; imported real Host resume passed. Native dialog and crash-at-directory-swap not run. E/functional-session-recovery-20261019.json. |
| BOOT-06 | 恢复草稿、选择、文件标签、视图布局：原版各 store/persist | `internal/recovery/`、`cmd/whaleisle/recovery.go`、workspace sessions | Actual two-process native save/open recovery PASS | Production saveResume then automatic openResume restored selection/history, unsent draft+local attachment, files/trajectory/todo/sidebar and unsaved editor bytes without flushing disk. Not a user-instance restart. E/functional-session-recovery-20261019.json. |

## 会话、消息和输入

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| CHAT-01 | 会话列表、搜索、分组/排序、置顶：`P/ui-workspace/src/client/index.ts`、Rows/navigation/pin-order | `internal/ui/sidebar/`、workspace navigation、session store | Actual Host list/search/pin/restart PASS (partial) | Three synthetic imports, content search, pin/unpin and restart persistence passed; large-list/grouping/reorder and native pointer menu parity remain unverified. E/functional-session-recovery-20261019.json. |
| CHAT-02 | 新建、改名、归档/恢复、删除及分支来源：`P/ui-workspace/`、`ui-session/` | sidebar/session menu、session lifecycle、ancestry | Actual owned Host CRUD/fork/restart PASS (partial) | Create/rename/archive/restore and delete only an owned archived fork; stale create/fork/reuse/scratch generations fenced. Native menu gestures not accepted. E/functional-session-recovery-20261019.json. |
| CHAT-03 | history load/resume/follow、切换与重连：`P/ui-session/`、ConversationMainPanel/ConversationSession | `internal/store/session/`、frame conversation phase | Actual owned persisted-history follow + native display PASS | Real Host list=3, all three histories ready, final native message screenshot; user running EXE not replaced. Full original-window visual comparison remains unavailable. |
| CHAT-04 | 空白 Hero、工作区/目录入口和首条消息过渡：HeroShell/InputBar | `internal/ui/frame/`、composer、workspacepicker | 仅单测/夹具；复用 65 映射 | 实际 Host 权威空白、非空历史不能显示欢迎态；当前新会话未验证 |
| CHAT-05 | 中文输入、选区、撤销/重做、换行、焦点：InputBar editor | `internal/ui/composer/draft_*`、MyGo Windows text input | 仅单测/夹具 | 实际 Windows IME 预编辑、候选提交、Escape 和 resize 后候选几何未验证 |
| CHAT-06 | 草稿/附件保留、输入区尺寸与拖动：InputBar、draft/resize preferences | composer resize/attachments/state、recovery | Actual native draft/attachment/layout recovery PASS (partial) | Independent save/open restored draft, attachment Path/Name and layout; new card is source-sized240x64DIP. User saved resize setting and manual drag/IME not touched or accepted. E/functional-session-recovery-20261019.json; final card evidence tracked separately. |
| CHAT-07 | 发送、流式回复、queue/steer、停止与忙碌 Enter：submission settings/InputBar | session composer submit/pending、composer actions | 仅单测/夹具 | 实际模型路径未验证；取消后不重复发送、错误恢复、队列顺序需独立验收 |
| CHAT-08 | Slash/@命令、补全、skill/文件引用：`ui-input-trigger`、`ui-commands`、`ui-skill`、`ui-reference` | composer suggestions/commands、file mentions、skill controls | Actual default-session skill inventory/editor PASS (partial) | Default original preset with an owned idle session: create/edit/flags/refresh/UserOnly/review-delete passed; strict artifact audit metadata-only. Slash/@/file completion and skill invocation unverified. E/functional-default-session-skills-20261019.json. |
| CHAT-09 | model/preset/权限选择及菜单：`ui-model-selection`、`ui-agent-preset`、`ui-permission-presets` | composer model/permission commands、preset seat | Actual empty preset refresh/default persistence PASS (partial) | Real preset catalog and owned future default CAS save/restart/restore passed; stale option receipts fenced. No live model catalog, permission switch or Agent execution. E/functional-local-settings-composer-20261019.json. |
| CHAT-10 | 文件/图片上传、粘贴、拖放、移除：`ui-file-upload`、`ui-attachment` | composer attachment/paste、clipboardimage、attachment cache | Actual owned upload/recovery/compact native card PASS (partial) | Text receipt/PNG preparation, duplicate refusal, same-owner reuse/cross-session fresh receipt, JSON restore reprepare, failed file retry/remove passed; final native card240x64/image64x64 verified. Host correctly refused fetching an unreferenced staged blob. No prompt, clipboard, picker/drop gesture. E/functional-attachments-composer-20261019.json. |
| CHAT-11 | 语音输入/录制取消：原版 InputBar 工具与 desktop voice contribution | voicecapture/voiceinput、composer voice controller | 有实现；实际未验证 | 麦克风/转写能力及取消未验证；不得以入口存在判正常 |
| CHAT-12 | Markdown、代码、思考、流式文本、工具消息：`ui-chat`、`ui-tool` | `internal/ui/chat/`、MyGo Markdown、tool lifecycle | Actual saved rich history + native render PASS (partial) | Strict persisted text/code/table/thinking fixtures followed through real Host and production controls; six native states captured. Streaming/model generation, long-list navigation and original pixel parity unverified. E/functional-rich-history-20261019.json. |
| CHAT-13 | 消息复制/引用/编辑重发/分支/反馈：`ui-message-edit`、`ui-message-feedback`、MessageIconActions | chat actions/edit/quote/history/feedback | Actual owned completed-turn fork PASS (partial) | Real Host fork of synthetic completed history; saved rich history Copy/disclosure controls verified. Live edit/replay/feedback and system clipboard gestures unverified. E/functional-session-recovery-20261019.json and E/functional-rich-history-20261019.json. |
| CHAT-14 | 图像、引用、交付物、工具文件链接：`ui-reference`、`ui-deliverables`、`ui-attachment` | chat citations/images/deliverables/tool files | Actual saved attachment/image/deliverable native display PASS (partial) | Fixed standard nested ImageBlock reference; real Host image fetch/decode and native bitmap/deliverable preview with disabled unavailable handoff. No prompt/upload-consumption or full deliverable lifecycle. E/functional-rich-history-20261019.json. |
| CHAT-15 | Workflow run/phase/member 专用呈现：`ui-workflow-run/index.ts`、workflow-definition | `internal/workflowrun/`、`ui/chat/workflow.go` | Actual saved Workflow native display PASS (partial) | Dedicated failed run/completed member/empty phase and unavailable-member guards passed on real Host history. Running/cancel/interruption and model-backed Workflow execution not run. E/functional-rich-history-20261019.json. |
| CHAT-16 | 子代理、层级、导航与详情：`ui-subagent`、`ui-agents-panel` | `internal/ui/agents/`、session agent navigation | 有实现；实际未验证 | 子会话只读、交互、附件、取消、父子回退及失效身份 |
| CHAT-17 | 轨迹、请求头、上下文、压缩/历史范围：`ui-trajectory`、conversation contributions | chat trajectory/context/manual compaction、session inspector | 有实现；实际未验证 | 原版视图项、范围/定位、长历史、压缩前后语义和只读模式 |
| CHAT-18 | Goal/Plan/Todo、下一提示、使用量/费用：`ui-goal`、`ui-plan`、conversation contributions | chat plan/todo/performance、sessioncost、goal read state | 仅单测/夹具 | Optional goal 修复已回归；真实空 goal Hero、读错归属及计划/提示生命周期未验证 |
| CHAT-19 | 工具权限批准/拒绝、原因与作用域：`ui-approval` | `internal/ui/interaction/approval*`、interaction store | Actual saved approval-audit native display PASS (partial) | Historical approval audit did not recreate a live actionable approval card; live approve/deny/scope/expiry lifecycle not executed. E/functional-rich-history-20261019.json. |
| CHAT-20 | 用户问题、编辑答案、等待/历史、计划审核：`ui-user-questions` QuestionComposer/PlanReviewPanel | interaction questions/plan_review、chat question_reply/submitted_plan | Actual saved answered/continued question native display PASS (partial) | Fixed current surfaceOp append reply recognition; continued supplement+normal composer and settled reply/history/Copy passed. Minimize/reopen targeted regression passed. No live answer submission/plan review. E/functional-rich-history-20261019.json. |
| CHAT-21 | Visual reply 预览、发布、历史读取、隔离交互/导出：`ui-visual-replies` | chat visual_reply、htmlpreview、visual reply cmd adapters | readiness 补丁/单测；真实功能未验证 | rc.2 副本全部七处 readiness checks；模型视觉能力、独立渲染及发布未执行，不能绕 gate |

## 工作区、文档、Git 和终端

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| WORK-01 | 工作区切换、添加、改名、移除、无目录：`ui-workspace/WorkspacePicker` | `ui/workspacepicker/`、workspace navigation | Actual isolated Host/controller/FS PASS | Register/select/persist-failure retry/stale-token/rename/cancel/remove verified; removal preserved owned files. Native picker interaction not exercised. E/functional-workspace-host-20261019.json. |
| WORK-02 | Miller 目录浏览、路径编辑、隐藏项、创建和失败重试：`ui-directory-picker-browse` | `ui/dirpick/`、directory_picker.go | Actual directoryPicker RPC/FS + headless Miller PASS (partial) | Empty/missing directory, create/retry and invalid-name rejection passed. Hidden-path and path-edit assertions were not executed; native pointer dialog/ACL remain unverified. E/functional-files-git-20261019.json corrects the earlier raw receipt. |
| WORK-03 | 文件树、展开/刷新、路径/文件引用及操作菜单：`ui-files/apply.ts`、FileTree | `ui/files/` tree/source actions、workspacefs | Actual filesystem + production DirectoryTree PASS (partial) | External create/rename followed by production refresh, expand/collapse and root boundary checks; app-owned create/rename commands and real context-menu pointer use not exercised. E/functional-files-git-20261019.json. |
| WORK-04 | 编辑、保存、自动保存、外部修改和脏关闭确认：FileSaveCoordinator/FileClosePrompt | files autosave/reload/draft、file editor/controller | Actual product save/autosave/conflict/dirty recovery PASS | Real Tabs/SaveQueue and bytes, external-write refusal, failed-save draft retention, Keep/discard and recovery serialization; close prompt driven headlessly. E/functional-files-git-20261019.json. |
| WORK-05 | 文本/代码/Markdown 预览、行定位/搜索/换行/只读：documentpreview text/code/markdown registrations | files text_read/text_preview/wrap/document_surface | Actual async readers/paging/line/missing retry PASS (partial) | UTF-8 text/code/Markdown source loads and edit guards; complete rendered Markdown visual/selection parity remains unverified. E/functional-files-git-20261019.json. |
| WORK-06 | HTML、图像、PDF、Office、Excel 预览：documentpreview index.ts 的各类型注册 | files media/image/table、pdfpreview、mediapreview、webview | Actual native format reads + WebView2 HTML/PDF + Host Office conversion PASS (partial) | PNG/JPEG/BMP/GIF/ICO/SVG/AVIF/WebP; XLS/XLSX/CSV/TSV; actual HTML sandbox and PDF canvas/text/page2/150percent screenshots; DOCX/XLSX/PPTX via bundled LibreOffice. Fixed PDF TDZ and clamped-page navigation. Legacy DOC/PPT, encrypted inputs and ACL not run. E/functional-files-git-20261019.json and E/office-runtime-provenance-20261019.json. |
| WORK-07 | 标签、停靠/浮动、工具区尺寸、关闭与恢复：`ui-sidebar-right`、`ui-dockkit` | frame workspace panes、files tabs/floating、workspace sessions | Actual Windows native browser workspace PASS (partial) | Fixed synchronous navigation-finish Eval deadlock by posting after callback. Real second navigation/back/forward/localStorage/workspace isolation/close passed. Original pixel parity/manual drag and recording unverified. E/functional-browser-workspace-20261019.json. |
| WORK-08 | Browser 标签、URL 导航、标题、历史、录制/失败：`ui-sidebar-browser`、desktop guests | browserpreview、webview、browser_workspace、browserrecording | Actual native browser hide/close and owned storage isolation PASS (partial) | Production WebView native lifecycle and workspace partition isolation verified; floating drag/manual resize/recording not accepted. E/functional-browser-workspace-20261019.json. |
| WORK-09 | Git 状态、diff、分支选择/创建：`ui-git/apply.ts`、BranchMenu/CreateBranchDialog | desktop/git、diffsurface、Git cmd adapters | Actual owned no-remote Git workflow PASS | Init, untracked diff, scoped stage/unstage/reviewed commit, branch create/switch/compare, dirty collision, cancel, stale review and invalid path verified through product APIs. Native pointer use not verified. E/functional-files-git-20261019.json. |
| WORK-10 | Git commit/publish 及授权/失败：CommitDialog/PublishDialog | desktop/git、git action wiring | Actual local reviewed commit + publish validation PASS (partial) | No remote configured, no remote publication attempted. Real remote authentication/push require separate authorization; local invalid publish review verified. E/functional-files-git-20261019.json. |
| WORK-11 | 会话工作区终端、split/tab、输入/resize/隐藏：`ui-user-terminal` TerminalDrawer/Pane | `ui/terminal/`、Ghostty/ConPTY、terminalruntime | Actual Windows NativeDLL + ConPTY PASS | Pinned DLL, cmd /D /Q sentinel/cwd/resize/close/reopen/natural exit. Manual IME/keyboard/clipboard and all other shells remain unverified. E/functional-terminal-20261019.json. |
| WORK-12 | Host 终端、shell/会话选择和权限：`ui-sidebar-terminal`、settings-shell | hostterminal store、terminal_target、host terminal UI | Actual rc.2 Host + production MyGo HostView PASS | Fixed obsolete follow/write/resize protocol to attach duplex. Output/fit resize/rename/control takeover/read-only/reconnect/hide/close/reopen/exit/resource cleanup passed; original-window pixel parity not claimed. E/functional-terminal-20261019.json. |

## 全部设置 section 与共享能力

设置注册 ID/order 复用原记录的 15 项库存；早期页面差异文字已过时，不作为当前缺陷结论。每项均需验证读取、编辑、失败保留、取消/返回、保存和重新加载。Host contribution 必须以实际 served namespace 为准。

| ID | 原版 section/依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| SET-01 | SettingsRoot/navigation/onboarding：`ui-settings-general`、`ui-settings-account` | settings modal/navigation/sections、account onboarding | 仅单测/夹具；65 映射 | 当前构建实际打开、所有入口回退焦点、窄窗/空服务 |
| SET-02 | account：`ui-settings-account/AccountSection` | settings account/signout、accountlogin/logout | 仅单测/夹具 | 登录/浏览器返回、账户流、额度、登出、失败恢复未真实验收 |
| SET-03 | general：GeneralSection、instructions/permission contributions | settings desktop/custom_instructions/permission_defaults、settings store | Actual safe Host preference save/restart/restore PASS (partial) | Only ui-conversation.busyEnter was changed, verified after restart, restored and verified after a second restart. Other general settings not accepted by extrapolation. E/functional-plugins-settings-tasks-20261019.json. |
| SET-04 | general 子项：shortcuts、developer tools、session log upload | settings_shortcuts、desktop preferences、session-log-deepseek row | 有实现；部分测试 | 快捷键编辑/冲突/恢复、开发工具开关、日志上传分别未实际验收；不擅自上传 |
| SET-05 | appearance：AppearanceSection/font/wallpaper/cursor | settings appearance/theme_font/wallpaper/pointer、ui/theme | 仅单测/夹具；部分实现 | 字体回退/显式覆盖、主题/壁纸生效和原生像素未验收；完整 GPU fluid 仍未实现 |
| SET-06 | interface：InterfaceSection、Beam/TypingFx | settings interface/effect_preferences、composer effects | 仅单测/夹具；部分实现 | 动画持久化与数字状态有回归，剩余图形层/实机动画不可签一致 |
| SET-07 | models：ModelsSection/provider forms/discovery | models、model_configuration、provider_profiles/discovery | Actual static provider configuration read PASS (partial) | Read two providers/43 configurable entries without discovery/auth/model calls; account, credentials, live discovery and selection not run. E/functional-default-session-skills-20261019.json. |
| SET-08 | plugins：PluginsSettingsSection、inventory/client modules | settings plugins/plugin_body/plugin_inventory | Actual plugin inventory + owned fixture lifecycle PASS (partial) | Inventory20 and no-dependency/no-script local install/disable/remove persisted across restarts. Native settings UI and dependency-bearing packages remain unverified. |
| SET-09 | skills：SkillsSection/groups/editor/catalog | settings skills、skill editor/directory、skill stores | Actual owned skill filesystem + default-session scope PASS (partial) | Create/read/edit/flags/delete and isolated restart persistence passed, plus default idle-session inventory path. Native picker/IME, external install/update/ACL not run. E/functional-local-settings-composer-20261019.json and E/functional-default-session-skills-20261019.json. |
| SET-10 | market：MarketSection/desktop override | ui/market、settings embedded content | 仅单测/夹具 | 与通用 Discover 分开；真实 catalog 和安装管理见 PLUG 行 |
| SET-11 | mcp：McpSection/import/catalog/auth | settings mcp/form/import/catalog、mcpconfig | Actual own stdio MCP lifecycle + restart persistence PASS (partial) | Production controller import/create/test/enable/health/edit/disable/delete; original SDK initialize and paged tools/list; seven owned children exited, zero tools/call. Native GUI/HTTP/OAuth/third-party services unverified. E/functional-local-mcp-20261019.json. |
| SET-12 | remote：RemoteSection/GatewaySettingsTab/Channels | ui/remote、ui/im/settings、phone/SSH adapters | 仅单测/夹具 | 网络及设备验收见 REMOTE 行；不能仅凭设置表单通过 |
| SET-13 | agent-presets：AgentPresetSection/guide/seat | settings agent_presets/guide、preset manager | Actual owned preset catalog/default CAS persistence PASS (partial) | Declarative empty preset catalog/default save/restart/restore passed; no live binding/Agent start or full native authoring lifecycle. E/functional-local-settings-composer-20261019.json. |
| SET-14 | usage-stats：`V/dsh-usage-panel/.../StatsSection` | ui/usage embedded/stats/dimensions/time_range | 仅单测/夹具 | 日期/维度/空/错误/刷新与真实统计未验证 |
| SET-15 | pet：PetSection/Whale contribution | settings pet/look、assistant_whale_settings | 仅单测/夹具 | 与实际桌宠/Whale 操作分开；开关生效和原版桥接状态 |
| SET-16 | remote-workspace：`V/dsh-remote` settings contribution | ui/remote/settings_body、remote workspace adapters | 仅单测/夹具 | 独立 workspace 能力与本机目录不能混为一项 |
| SET-17 | about：AboutSection/current version/update entries | settings about、update controller | 仅单测/夹具 | 真实版本/来源、外链、更新动作未验收 |
| SET-18 | served 插件配置：subagent/web-search/agent-loop/shell 等 | settings plugin_body/model_body、captured namespace binding | Actual served describe25 + one namespace persistence PASS (partial) | Real Go schema decoder and ui-conversation busyEnter save/restore across two Host restarts. Other served namespaces still require field-specific acceptance. |

## 插件、任务与 Projects

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| PLUG-01 | 独立 profile Plugins：`ui-plugin-manager/PluginManagerPage` | internal/pluginmanager、ui/pluginmanager、native_plugin_manager | Actual inventory20 + owned profile lifecycle PASS (partial) | Real production controller and Host; native navigation/details screenshot not accepted. E/functional-plugins-settings-tasks-20261019.json. |
| PLUG-02 | 通用 Discover/detail/source/pagination：`ui-plugin-market` | pluginmanager discover/dialogs | Actual original optional-bundle local catalog PASS; default fresh profile HTTP404 | Own fresh profile enabled the already bundled original module via manager (no install); own-loopback indexes/catalog/page/search/cache/refresh/typed errors and restart persistence passed, sources restored and bundle disabled. Remote market/native UI remain unverified. E/functional-plugin-market-local-20261019.json. |
| PLUG-03 | install/progress/log/cancel/uncertain reconciliation；依据：manager-store/install terminal、plugin market | pluginmanager install/controller、event wiring | Actual local install/disable/remove + restart persistence PASS (partial) | Original pnpm12.4.1 exact local binary required. Only no-script/no-dependency own fixture; cancel/uncertain reconciliation and remote dependencies not exercised. |
| PLUG-04 | 精选 Market tabs/detail/README/install/update/remove：`ui-settings-market`、`D/main/dsh-market-desktop.js` | ui/market、desktop/market | Actual installed inventory20 PASS (partial) | Remote catalog/download/README/install/update not verified. Inventory is not remote-market acceptance. |
| PLUG-05 | README 媒体播放/失败 alt；依据：原版 Markdown/media | market readme_images、Markdown/SVG/image support | 部分实现 | GIF 有实现；完整动画 WebP 与未支持 SVG 构造不能呈现为已一致；真实播放未验收 |
| TASK-01 | Jobs header/roster/output：`ui-jobs/JobListAction` | jobflow、session/jobs、frame/jobs | Actual rc.2 jobs roster/output/cursor/mount PASS | Production jobflow + real Mux NativeJobs wire, two original public-registry owned jobs, UTF-8 output and resume cursor; popup follow closed while mounted roster stayed live. Native user GUI/Store lifecycle not claimed. E/functional-jobs-20261019.json. |
| TASK-02 | Jobs 两次确认 kill、输出展开、clear；依据：同上 JobListAction | jobflow control/state | Actual cancel/kill/settlement/clear semantics PASS | Cancel sends0kill; confirm sends1 requested receipt, later producer exit proves killed; clear hides only settled rows locally and leaves Host records intact. Unmount releases logical streams; child/Host/listener closed. E/functional-jobs-20261019.json. |
| TASK-03 | schedule catalog/detail/history：`ui-schedule` | ui/schedule、schedule store | Actual future own schedule catalog/history PASS | One2099 task, history empty, edit persisted across real restart. No task execution/delivery. E/functional-plugins-settings-tasks-20261019.json. |
| TASK-04 | schedule create/edit/run/cancel/delete；依据：TaskManager/TaskDetail | schedule controller/actions | Actual original service create + Go edit/delete/restart PASS (partial) | Inert own fixture called public Schedule.create (not a new RPC); Go captured record edit/delete worked. Native UI create, Run/cancel/model/delivery remain unverified. |
| PROJ-01 | Project rows/menu/modal/directory：`V/dsh-project/client/client.js` | ui/projects/sidebar/modal、projectflow | Actual Project Host bridge + Go client PASS (backend) | Fresh list/create/directory identity reuse/rename/detail original Lead verified. Native menu/modal/layout unverified. E/functional-projects-20261019.json. |
| PROJ-02 | task cards/queue/work/team/observer；依据：原版 ProjectTaskCard、请求投影 | projects observer/task_cards、project request adapters | Actual empty request/queue/work reads PASS (partial) | No request, delegation, queue mutation or model execution. Nonempty task/team/observer projections remain unverified. |
| PROJ-03 | notes/preferences/charter/materials/documents/deliveries；依据：原版 Project routes | projectflow materials、projects documents/deliveries/assets | Actual material and docs Host roundtrip PASS (partial) | Notes user/generated separation, preferences and charter persisted; owned static seeded docs read/list passed. Worker artifacts/deliveries and native UI remain unverified. |
| PROJ-04 | reports/history/process/activity/diff；依据：原版 Project routes | projects history/process/activity/detail、diff | Actual safe empty request/work reads PASS (partial) | Nonempty model-backed reports/process/history/activity/diff remain unverified; seeded file reads are recorded separately. |
| PROJ-05 | rename/archive/request lifecycle/dirty transitions；依据：原版 Project lifecycle | projectflow lifecycle、projects lifecycle | Actual archive/restore + restart identity/material persistence PASS | Archived material write rejected; original Lead and own materials/docs retained after two Host starts. Request lifecycle and native dirty transitions remain unverified. |

## Remote、Whale/Bot 和桌宠

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| REMOTE-01 | remote machines/workspaces/files/tools；依据：ui-settings-remote、`V/dsh-remote` client | ui/remote、remote store、remote workbook | Existing owned HTTP fixture only; real remote machine NOT_RUN | No external machine credentials or remote workspace/files/tools lifecycle acceptance. E/functional-local-remote-20261019.json. |
| REMOTE-02 | phone Gateway/pairing/devices、LAN/relay topology；依据：原版 GatewaySettingsTab、`D/relay` | phoneremote、gateway/phone device editors | 有实现；实际未验证 | 真实设备/网络未验收；legacy token 不能当新配对凭据 |
| REMOTE-03 | SSH tunnel/identity/host validation；依据：原版 SSH/remote desktop contributions | desktop/sshremote、remote SSH wiring | Actual own SSH server/forward lifecycle PASS (partial) | Host-key reject/explicit accept/policy overrides, local/reverse UTF8 echo, cancellation/peer exit/port reuse passed; UI prompt ownership fenced. No user keys/system SSH changes or real remote machine. E/functional-local-remote-20261019.json and E/functional-ssh-ui-operation-20261019.json. |
| REMOTE-04 | Channels/accounts/IM configuration and messaging；依据：`V/dsh-im`、`D/main/dsh-im-desktop.js` | ui/im、im store/settings | 有实现；实际未验证 | 平台授权、配置与对话/投递分开验收，不擅自发送 |
| BOT-01 | Whale settings、任务、routine/trigger；依据：`V/dsh-whale`、`V/dshbot` client | ui/assistant tasks/routines/Whale settings、stores | Actual owned assistant metadata CRUD/restart PASS (partial) | Strict expected receipt adapted; safe task/routine/trigger metadata passed with no execution. No run/cancel/model/notification delivery. E/functional-assistant-20261019.json. |
| BOT-02 | Bot/联系人、头像、可见性、复制/删除、目标/记忆；依据：dshbot client registrations | assistant bot_*/contacts、cmd controllers | Actual own bot/contact metadata lifecycle PASS (partial) | Owned contact/bot visibility/copy/delete/goal/memory operations and persistence passed; strict five-session metadata-only artifact audit. No real messages/model. E/functional-assistant-20261019.json. |
| BOT-03 | rooms/members/runtime/sender policy/custom panels/notifications；依据：dshbot/Whale contributions | assistant room_*、custompanel、botnotify/bridgealerts | Actual owned room registration/create/restart PASS (partial) | Profile-local original dshbot room preset adapter registers required tools and exact original room definition before ACTIVE. Metadata routes passed; real members/messages/notifications not sent. E/functional-assistant-20261019.json. |
| PET-01 | 桌宠开关/皮肤/位置/行为/拖动/点击穿透；依据：`D/main/desktop-pet*`、desktop-live2d、renderer pet* | desktop/pet、pet settings/cmd adapters | 部分单测；实际未验证 | 原生透明窗口、多屏/缩放、位置恢复、原版资源类型；不能凭 live2d 命名判断算法缺失 |
| PET-02 | pet chat/look/care/growth/personality/notifications；依据：原版 pet-chat/stats/growth/settings | desktop/pet、quickchat、pet controllers | 有实现；实际未验证 | 真实聊天/设备/成长数据/通知未验证；禁止擅自付费调用 |

## 桌面集成、安装和持久化

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| DESK-01 | 文件/运行/视图/帮助菜单：`D/main/menu.js` | application_menu.go、application shortcuts | 有实现与测试；实际未验证 | 工作区/launcher/settings/market/MCP/skills、restart/reload、zoom/fullscreen/devtools/help/about逐动作 |
| DESK-02 | tray show/launcher/settings/market/restart/pet/quit：tray-menu.js | desktop_tray/menu/notice | 有实现与测试；实际未验证 | 实际托盘、close-to-tray/显示与退出；隔离启动主动跳过此项 |
| DESK-03 | caption/drag/resize/snap、窗口焦点/关闭/退出确认；依据：window/chrome/close-behavior/native-window-motion | ui/chrome、frame、taskguard、native window | 有实现；实机未验证 | 同 DPI 与字体/窗口尺寸；活跃任务、窗口最小化/关闭/重开 |
| DESK-04 | 全局/应用快捷键、命令参考、菜单分发；依据：shortcuts.js、ui-shortcuts Editor/Reference | shortcuts/shortcutprefs、unified/application shortcuts | 有实现与测试；实际未验证 | 原生键盘冲突、输入框/IME优先级、禁用/重绑定及持久化 |
| DESK-05 | deep links、单实例、开机启动、外部打开；依据：原版 main/preload desktop integrations | deeplink、MyGo app integration、open-in-app | 有实现；实际未验证 | 不能由隔离 mode 认证用户系统集成；不更改系统注册以补表 |
| DESK-06 | 更新检查/下载/取消/提示、安装/回滚；依据：`D/main/update*`、update-dialog | desktop/update、update_attention、startuprec | Actual owned local updater HTTP/files lifecycle PASS (partial) | Check/available/current/download/hash rejection/cancel/partial/range/patch exercised against inert owned payloads. Publisher signature, OS install/rollback and live update service not run. E/functional-local-updater-20261019.jsonl. |
| DESK-07 | installer brand/language/welcome/license/finish/uninstall；依据：原版 installer/packaging resources | scripts/nsis、installer branding | 静态/定向证据；实际安装未验证 | 打包工具、包构建、可逆安装/卸载分别记录，不以资源检查推定安装成功 |
| SAVE-01 | 普通配置与未知字段保留、并发原版/Go写入；依据：`D/main/config.js` | internal/config、workspace preferences merge | Actual native concurrent preference preservation PASS (partial) | Second writer closeToTray retained while production per-Host grouping changed/restored. Same-field conflict covered by targeted regression. E/native-core-final-e21f4b8.png.json. |
| SAVE-02 | 凭据加密/引用、账户登出及安全存储状态；依据：原版 config-credentials/account/profile operations | credentials、credentialrefs、accountlogout | Actual Windows DPAPI synthetic file roundtrip/tamper PASS (partial) | Real CryptProtectData/UnprotectData, encrypted Save/Load, synthetic Chromium key interoperability and tamper rejection; no user credentials accessed. Account logout/provider migration remain unverified. E/functional-storage-windows-20261019.jsonl. |
| SAVE-03 | Host settings 原子修改、失败草稿保留、namespace owner；依据：原版 settings/config-form stores | store/settings/body_preferences/plugin_body | Actual served setting restart persistence PASS (partial) | Only ui-conversation.busyEnter save/restart/restore/restart exercised; other namespaces and owner-switch failure behavior remain targeted regressions, not full runtime acceptance. |
| SAVE-04 | 草稿/附件/文件/布局/预览/terminal/shortcut 的保存与恢复；依据：原版各 runtime share/persist/draft | recovery、config、workspace sessions、files drafts/preferences | Actual two-process draft/file/layout + browser partition PASS (partial) | Unsaved editor buffer restored without disk flush; draft/local attachment/selection/layout recovered. Browser within-workspace storage persisted and cross-workspace separated. Terminal handles/old shell/model mutations are not replayed. E/functional-session-recovery-20261019.json; E/functional-browser-workspace-20261019.json. |

## 明确差异和优先级

1. 兼容 runtime 完整 stage/verify 已通过，最高优先级转为隔离实例的真实 Host 握手、会话加载、resume/follow、错误重试、消息显示和视图配置保存。全量矩阵不抢修复主线，不把只读快照成功放大为 UI 通过。
2. 65 映射外需要补验收证据的主要能力是消息类型/Workflow/审批问题、文件编辑及全部预览格式、Git、两种终端、Whale/Bot、桌面集成、安装和逐路径持久化。以上大多已有实现，不能凭缺台账声明未实现。
3. 已知部分实现：完整源 128/1440 GPU 流体与其资源/多遍契约尚未实施；剩余 Beam 图形层/最终像素未一致；README 的完整动画 WebP及不支持 SVG 构造仍有缺口。这些是代码/格式能力差异，不是缺截图导致的工具阻塞。GPU/GIF 后续开发当前暂停。
4. 原版同状态视觉、原生 Windows IME、真实 PTY/系统菜单/托盘/设备/安装和外部服务行为仍没有当前产品认证。可用工具的变化需要更新具体状态，不能永久沿用历史阻塞。

## 本次审计事实与维护

本次只读检查了原版 AppFrame/SidebarRoot、workspace 注册、document preview 各类型注册、Git titlebar 注册、Workflow/用户问题注册、15 settings section 库存和桌面 menu/tray；核对 Go 对应实现存在。未发现可仅凭本轮证据确认的新协议错误，因此未新开重复修复任务。发现 `ui/chat/workflow.go`、`internal/workflowrun/` 等既存实现后明确标成“实际未验证”，没有误报缺失。

原版 `ui-sidebar-documentpreview/src/client/index.ts` 的注册清单明确包括 text/Markdown/HTML/image/PDF/code/Office/Excel；不同格式必须分别验收。原版 `ui-workflow-run/src/client/index.ts` 注册独立 conversation chat node；普通 tool 消息测试不能代替其验收。原版 menu/tray 也应逐动作登记，隔离隐藏启动不会触发这些能力。

各实现/回归的详细链接继续维护在原 65 映射和核心恢复章，不复制一份旧测试结果。后续给某行登记“实际通过”时补入：产品 SHA、runtime pin/产物身份、隔离数据/实际前提、操作、观察结果和证据路径。无结果、取消或只编译保持未验证；主线成功后由集成负责人更新 BOOT/CHAT/SAVE 状态。

### 现有证据复用索引

以下链接用于定位已有实现回归，不代表本轮重新执行，也不能替代实际产品验收：

| 范围 | 现有具体回归/证据入口 | 尚不能认证的部分 |
|---|---|---|
| BOOT、CHAT-03/18、SAVE | [核心恢复与三份快照记录](native-workspace-parity-20261019.md#core-availability-session-runtime-optional-goal-and-view-preferences)、[session runtime 回归](../internal/harness/session_runtime_test.go)、[视图偏好](../cmd/whaleisle/view_preferences_test.go)、[配置合并](../internal/config/workspace_preferences_merge_test.go) | 当前候选真实 Host、UI resume/follow、跨进程保存 |
| CHAT-01/04/06、WORK-01/02/07、SET、TASK、PROJ | [65 条组件映射](native-workspace-parity-20261019.md#windows-follow-up-component-acceptance-ledger) 中的具体测试及软件/隐藏原生夹具路径 | 合成数据截图与旧候选不能认证当前真实业务操作 |
| CHAT-15/16 | [workflow 模型](../internal/workflowrun/model_test.go)、[成员流](../internal/store/session/workflow_member_stream_test.go) | 实际 workflow 各终态、跨 phase/member 定位 |
| CHAT-19/20 | [approval context](../internal/store/interaction/approval_context_test.go)、[问题编辑](../internal/store/interaction/question_editing_test.go)、[问题历史](../internal/store/interaction/question_history_test.go) | 实际权限操作、提交、过期与历史只读闭环 |
| WORK-03..06/09 | [文件 workflow](../internal/ui/files/workflow_test.go)、[Git workflow](../internal/desktop/git/workflow_test.go) | 每种预览格式、文件真实保存冲突、Git 实际 mutation |
| WORK-11/12 | [Host terminal selection](../internal/store/hostterminal/selection_test.go)、[occurrence close](../internal/store/hostterminal/occurrence_close_test.go) | 原生 PTY、真实 Host 终端和资源释放 |
| PLUG、TASK | [插件事件](../internal/pluginmanager/events_test.go)、[README](../internal/ui/market/readme_test.go)、[SVG](../internal/ui/market/readme_svg_test.go)、[Jobs mounted](../internal/jobflow/mounted_test.go)、[schedule edit](../internal/store/schedule/edit_test.go) | 真实 registry/install、运行任务、调度执行 |
| PROJ | [请求队列](../internal/projectflow/request_queue_test.go)、[documents](../internal/projectflow/documents_test.go)、[history](../internal/projectflow/history_test.go)、[lifecycle](../internal/projectflow/lifecycle_test.go) | 当前 runtime 的实际模块握手、读写与归档 |
| REMOTE、BOT、PET、DESK、SAVE | 对应目录现有回归；[phone runtime](../internal/desktop/phoneremote/runtime_test.go)、[SSH host key](../internal/desktop/sshremote/known_hosts_test.go)、[Whale section](../internal/ui/assistant/whale_settings_section_test.go)、[pet recovery](../internal/desktop/pet/recovery_test.go)、[persistence](../internal/config/persistence_test.go) | 真设备、系统入口、通知、安装及重启恢复逐项未验收 |
