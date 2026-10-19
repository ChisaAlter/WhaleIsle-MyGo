# 原版全量功能验收矩阵（2026-10-19）

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
| BOOT-02 | 真实 Host/Node/插件闭包选择：`V/harness-upstream.json`、原版 `scripts/production-runtime.js`、`scripts/runtime-instance-graph.js`、安装资源 | `internal/harness/`、`internal/paths/`、`scripts/session-runtime.mjs`、构建/打包脚本 | **实际通过：独立 runtime stage/verify；Host 实际未验收** | 主线确认 rc.2 闭包及 3143 条内部链接校验成功；候选实际选中路径、Host 握手和 UI 数据读取仍须分别记录 |
| BOOT-03 | 会话日志严格准入和版本恢复：原版 session/persistence/format catalog | `internal/harness/session_runtime.go`、session runtime admission | **实际通过：三份既有快照 rc.2 cold read** | 仅只读 API；不能推定 Go resume/follow/UI 通过；未知 required event 仍须拒绝 |
| BOOT-04 | 内核退出、启动恢复、安全模式、重启：`D/main/harness-controller.js`、`plugin-recovery-actions.js`、launcher | `startup_recovery.go`、`internal/startuprec/`、harness supervisor | 仅单测/夹具 | crash/恢复/重试及失败记录的真实生命周期未验证，不重启用户实例 |
| BOOT-05 | 旧数据导入、类别选择、冲突和中断恢复：`D/main/data-import*`、`import-transaction.js` | `internal/desktop/importer/`、`import_flow.go`、`import_recovery.go` | 有实现；实际未验证 | 用隔离源/目标验证完整事务、取消、恢复和源数据不变 |
| BOOT-06 | 恢复草稿、选择、文件标签、视图布局：原版各 store/persist | `internal/recovery/`、`cmd/whaleisle/recovery.go`、workspace sessions | 仅单测/夹具 | 当前版本实际退出/重开未验证；禁止把保存布局当成会话已恢复 |

## 会话、消息和输入

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| CHAT-01 | 会话列表、搜索、分组/排序、置顶：`P/ui-workspace/src/client/index.ts`、Rows/navigation/pin-order | `internal/ui/sidebar/`、workspace navigation、session store | 仅单测/夹具；复用 65 映射 | 实际大量会话、菜单动作身份、搜索和持久排序未验证 |
| CHAT-02 | 新建、改名、归档/恢复、删除及分支来源：`P/ui-workspace/`、`ui-session/` | sidebar/session menu、session lifecycle、ancestry | 有实现；实际未验证 | 各动作分别验收；删除等仅在隔离数据中验证，不能以菜单能打开代替 |
| CHAT-03 | history load/resume/follow、切换与重连：`P/ui-session/`、ConversationMainPanel/ConversationSession | `internal/store/session/`、frame conversation phase | **核心未验收** | 真实快照只读成功尚未闭环到当前 Go 界面；需消息完整性、顺序、加载/错误/重试和晚到隔离 |
| CHAT-04 | 空白 Hero、工作区/目录入口和首条消息过渡：HeroShell/InputBar | `internal/ui/frame/`、composer、workspacepicker | 仅单测/夹具；复用 65 映射 | 实际 Host 权威空白、非空历史不能显示欢迎态；当前新会话未验证 |
| CHAT-05 | 中文输入、选区、撤销/重做、换行、焦点：InputBar editor | `internal/ui/composer/draft_*`、MyGo Windows text input | 仅单测/夹具 | 实际 Windows IME 预编辑、候选提交、Escape 和 resize 后候选几何未验证 |
| CHAT-06 | 草稿/附件保留、输入区尺寸与拖动：InputBar、draft/resize preferences | composer resize/attachments/state、recovery | 仅单测/夹具；复用 65 映射 | 当前 Host 的已存 resize 值、关闭/切换/重开后保留未验证，不清用户配置 |
| CHAT-07 | 发送、流式回复、queue/steer、停止与忙碌 Enter：submission settings/InputBar | session composer submit/pending、composer actions | 仅单测/夹具 | 实际模型路径未验证；取消后不重复发送、错误恢复、队列顺序需独立验收 |
| CHAT-08 | Slash/@命令、补全、skill/文件引用：`ui-input-trigger`、`ui-commands`、`ui-skill`、`ui-reference` | composer suggestions/commands、file mentions、skill controls | 有实现；实际未验证 | 所有原版命令来源、选择/取消、权限和 Host 切换未逐入口验证 |
| CHAT-09 | model/preset/权限选择及菜单：`ui-model-selection`、`ui-agent-preset`、`ui-permission-presets` | composer model/permission commands、preset seat | 仅单测/夹具 | 实际路由、当前选择及保存/重新加载未验证；与设置页去重复用同一能力 |
| CHAT-10 | 文件/图片上传、粘贴、拖放、移除：`ui-file-upload`、`ui-attachment` | composer attachment/paste、clipboardimage、attachment cache | 有实现；实际未验证 | 原生剪贴板、文件对话框、大文件/错误/取消、发送前后身份未验证 |
| CHAT-11 | 语音输入/录制取消：原版 InputBar 工具与 desktop voice contribution | voicecapture/voiceinput、composer voice controller | 有实现；实际未验证 | 麦克风/转写能力及取消未验证；不得以入口存在判正常 |
| CHAT-12 | Markdown、代码、思考、流式文本、工具消息：`ui-chat`、`ui-tool` | `internal/ui/chat/`、MyGo Markdown、tool lifecycle | 部分定向回归；完整消息实际未验证 | 多消息类型、流式更新、代码交互、长文/虚拟列表/定位分别验收 |
| CHAT-13 | 消息复制/引用/编辑重发/分支/反馈：`ui-message-edit`、`ui-message-feedback`、MessageIconActions | chat actions/edit/quote/history/feedback | 仅单测/夹具；65 映射部分覆盖 | 真实消息身份与 mutation、失败重试未验证 |
| CHAT-14 | 图像、引用、交付物、工具文件链接：`ui-reference`、`ui-deliverables`、`ui-attachment` | chat citations/images/deliverables/tool files | 有实现；实际未验证 | 实际附件读取、路径授权、错误/历史只读和点击目标未验证 |
| CHAT-15 | Workflow run/phase/member 专用呈现：`ui-workflow-run/index.ts`、workflow-definition | `internal/workflowrun/`、`ui/chat/workflow.go` | **已有实现，实际未验证** | 不能因 65 表缺行误报未实现；需 running/completed/failed/cancelled/interrupted、成员导航和空 phase |
| CHAT-16 | 子代理、层级、导航与详情：`ui-subagent`、`ui-agents-panel` | `internal/ui/agents/`、session agent navigation | 有实现；实际未验证 | 子会话只读、交互、附件、取消、父子回退及失效身份 |
| CHAT-17 | 轨迹、请求头、上下文、压缩/历史范围：`ui-trajectory`、conversation contributions | chat trajectory/context/manual compaction、session inspector | 有实现；实际未验证 | 原版视图项、范围/定位、长历史、压缩前后语义和只读模式 |
| CHAT-18 | Goal/Plan/Todo、下一提示、使用量/费用：`ui-goal`、`ui-plan`、conversation contributions | chat plan/todo/performance、sessioncost、goal read state | 仅单测/夹具 | Optional goal 修复已回归；真实空 goal Hero、读错归属及计划/提示生命周期未验证 |
| CHAT-19 | 工具权限批准/拒绝、原因与作用域：`ui-approval` | `internal/ui/interaction/approval*`、interaction store | 有实现与测试；实际未验证 | 原请求目标、一次/持续批准、只读/过期、取消及历史不得重新提交 |
| CHAT-20 | 用户问题、编辑答案、等待/历史、计划审核：`ui-user-questions` QuestionComposer/PlanReviewPanel | interaction questions/plan_review、chat question_reply/submitted_plan | 有实现与测试；实际未验证 | 多选/自填、提交/撤回、计划选项、已结束/过期/Host 切换 |
| CHAT-21 | Visual reply 预览、发布、历史读取、隔离交互/导出：`ui-visual-replies` | chat visual_reply、htmlpreview、visual reply cmd adapters | readiness 补丁/单测；真实功能未验证 | rc.2 副本全部七处 readiness checks；模型视觉能力、独立渲染及发布未执行，不能绕 gate |

## 工作区、文档、Git 和终端

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| WORK-01 | 工作区切换、添加、改名、移除、无目录：`ui-workspace/WorkspacePicker` | `ui/workspacepicker/`、workspace navigation | 仅单测/夹具；65 映射 | 真实工作区状态和授权未验证 |
| WORK-02 | Miller 目录浏览、路径编辑、隐藏项、创建和失败重试：`ui-directory-picker-browse` | `ui/dirpick/`、directory_picker.go | 仅单测/夹具；65 映射 | 真实读目录/创建目录/网络路径/权限结果未验证 |
| WORK-03 | 文件树、展开/刷新、路径/文件引用及操作菜单：`ui-files/apply.ts`、FileTree | `ui/files/` tree/source actions、workspacefs | 有实现；实际未验证 | 实际文件目标、权限、长目录和刷新/晚到结果 |
| WORK-04 | 编辑、保存、自动保存、外部修改和脏关闭确认：FileSaveCoordinator/FileClosePrompt | files autosave/reload/draft、file editor/controller | 有实现；实际未验证 | 隔离文件中核对字节、冲突、取消和关闭/切换后编辑归属 |
| WORK-05 | 文本/代码/Markdown 预览、行定位/搜索/换行/只读：documentpreview text/code/markdown registrations | files text_read/text_preview/wrap/document_surface | 有实现；实际未验证 | 格式、编码、分页、大文件、失败重试、历史文件的读权限 |
| WORK-06 | HTML、图像、PDF、Office、Excel 预览：documentpreview index.ts 的各类型注册 | files media/image/table、pdfpreview、mediapreview、webview | 有实现；格式支持未逐项实际验收 | 逐格式成功/损坏/大资源、滚动/缩放/打开外部、关闭释放；部分 decoder/渲染能力需明确列支持表 |
| WORK-07 | 标签、停靠/浮动、工具区尺寸、关闭与恢复：`ui-sidebar-right`、`ui-dockkit` | frame workspace panes、files tabs/floating、workspace sessions | 仅单测/夹具；65 映射部分覆盖 | 实机拖动/焦点、同/跨会话目标、重启恢复 |
| WORK-08 | Browser 标签、URL 导航、标题、历史、录制/失败：`ui-sidebar-browser`、desktop guests | browserpreview、webview、browser_workspace、browserrecording | 有实现；实际未验证 | HTTP(S) 导航/脚本能力、关闭/切换、进程释放和失败状态 |
| WORK-09 | Git 状态、diff、分支选择/创建：`ui-git/apply.ts`、BranchMenu/CreateBranchDialog | desktop/git、diffsurface、Git cmd adapters | 有实现；实际未验证 | 用隔离仓库分别检查 dirty/conflict/no-repo、身份与取消 |
| WORK-10 | Git commit/publish 及授权/失败：CommitDialog/PublishDialog | desktop/git、git action wiring | 有实现；实际未验证 | 外部写操作单列验收，不能由 diff 可读推定正常 |
| WORK-11 | 会话工作区终端、split/tab、输入/resize/隐藏：`ui-user-terminal` TerminalDrawer/Pane | `ui/terminal/`、Ghostty/ConPTY、terminalruntime | 有实现；实际未验证 | shell 启动、cwd、输入输出、拆分/关闭和 PTY 释放；隐藏不等于终止 |
| WORK-12 | Host 终端、shell/会话选择和权限：`ui-sidebar-terminal`、settings-shell | hostterminal store、terminal_target、host terminal UI | 有实现；实际未验证 | 与本地 PTY 去重但保持不同 target；真实 Host 读/写与权限未验证 |

## 全部设置 section 与共享能力

设置注册 ID/order 复用原记录的 15 项库存；早期页面差异文字已过时，不作为当前缺陷结论。每项均需验证读取、编辑、失败保留、取消/返回、保存和重新加载。Host contribution 必须以实际 served namespace 为准。

| ID | 原版 section/依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| SET-01 | SettingsRoot/navigation/onboarding：`ui-settings-general`、`ui-settings-account` | settings modal/navigation/sections、account onboarding | 仅单测/夹具；65 映射 | 当前构建实际打开、所有入口回退焦点、窄窗/空服务 |
| SET-02 | account：`ui-settings-account/AccountSection` | settings account/signout、accountlogin/logout | 仅单测/夹具 | 登录/浏览器返回、账户流、额度、登出、失败恢复未真实验收 |
| SET-03 | general：GeneralSection、instructions/permission contributions | settings desktop/custom_instructions/permission_defaults、settings store | 仅单测/夹具 | 指令和权限默认的实际生效、设置并发合并、Host 隔离 |
| SET-04 | general 子项：shortcuts、developer tools、session log upload | settings_shortcuts、desktop preferences、session-log-deepseek row | 有实现；部分测试 | 快捷键编辑/冲突/恢复、开发工具开关、日志上传分别未实际验收；不擅自上传 |
| SET-05 | appearance：AppearanceSection/font/wallpaper/cursor | settings appearance/theme_font/wallpaper/pointer、ui/theme | 仅单测/夹具；部分实现 | 字体回退/显式覆盖、主题/壁纸生效和原生像素未验收；完整 GPU fluid 仍未实现 |
| SET-06 | interface：InterfaceSection、Beam/TypingFx | settings interface/effect_preferences、composer effects | 仅单测/夹具；部分实现 | 动画持久化与数字状态有回归，剩余图形层/实机动画不可签一致 |
| SET-07 | models：ModelsSection/provider forms/discovery | models、model_configuration、provider_profiles/discovery | 仅单测/夹具 | 真实账户/模型发现、默认路由、credential refs、连接错误及只读身份 |
| SET-08 | plugins：PluginsSettingsSection、inventory/client modules | settings plugins/plugin_body/plugin_inventory | 仅单测/夹具；65 映射 | 与独立 manager 分开；实际加载器、设置保存、授权与刷新 |
| SET-09 | skills：SkillsSection/groups/editor/catalog | settings skills、skill editor/directory、skill stores | 仅单测/夹具 | 安装/更新/删除/目录权限及实际生效未验证 |
| SET-10 | market：MarketSection/desktop override | ui/market、settings embedded content | 仅单测/夹具 | 与通用 Discover 分开；真实 catalog 和安装管理见 PLUG 行 |
| SET-11 | mcp：McpSection/import/catalog/auth | settings mcp/form/import/catalog、mcpconfig | 仅单测/夹具 | 实际服务启动/授权/health、导入/保存/删除及权限未验证 |
| SET-12 | remote：RemoteSection/GatewaySettingsTab/Channels | ui/remote、ui/im/settings、phone/SSH adapters | 仅单测/夹具 | 网络及设备验收见 REMOTE 行；不能仅凭设置表单通过 |
| SET-13 | agent-presets：AgentPresetSection/guide/seat | settings agent_presets/guide、preset manager | 仅单测/夹具 | 选择/创建/编辑/删除和 Hero/session seat 实际一致性 |
| SET-14 | usage-stats：`V/dsh-usage-panel/.../StatsSection` | ui/usage embedded/stats/dimensions/time_range | 仅单测/夹具 | 日期/维度/空/错误/刷新与真实统计未验证 |
| SET-15 | pet：PetSection/Whale contribution | settings pet/look、assistant_whale_settings | 仅单测/夹具 | 与实际桌宠/Whale 操作分开；开关生效和原版桥接状态 |
| SET-16 | remote-workspace：`V/dsh-remote` settings contribution | ui/remote/settings_body、remote workspace adapters | 仅单测/夹具 | 独立 workspace 能力与本机目录不能混为一项 |
| SET-17 | about：AboutSection/current version/update entries | settings about、update controller | 仅单测/夹具 | 真实版本/来源、外链、更新动作未验收 |
| SET-18 | served 插件配置：subagent/web-search/agent-loop/shell 等 | settings plugin_body/model_body、captured namespace binding | 有实现与单测 | 实际 served 字段、默认值/校验/权限、Host 切换和单次原子保存需逐 namespace 验证 |

## 插件、任务与 Projects

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| PLUG-01 | 独立 profile Plugins：`ui-plugin-manager/PluginManagerPage` | internal/pluginmanager、ui/pluginmanager、native_plugin_manager | 仅单测/夹具；65 映射 | bundle/plugin/profile 身份、detail/settings/navigation、加载失败与刷新 |
| PLUG-02 | 通用 Discover/detail/source/pagination：`ui-plugin-market` | pluginmanager discover/dialogs | 仅单测/夹具 | 实际端点是否 served 由完整 runtime 验证；不可回退成精选市场冒充成功 |
| PLUG-03 | install/progress/log/cancel/uncertain reconciliation；依据：manager-store/install terminal、plugin market | pluginmanager install/controller、event wiring | 仅单测/夹具 | 真实安装、取消不确定结果、重连及 loader 收敛未执行 |
| PLUG-04 | 精选 Market tabs/detail/README/install/update/remove：`ui-settings-market`、`D/main/dsh-market-desktop.js` | ui/market、desktop/market | 有实现与定向回归 | 实际 registry/package/download/install 未验收；README 支持与媒体子集需明确 |
| PLUG-05 | README 媒体播放/失败 alt；依据：原版 Markdown/media | market readme_images、Markdown/SVG/image support | 部分实现 | GIF 有实现；完整动画 WebP 与未支持 SVG 构造不能呈现为已一致；真实播放未验收 |
| TASK-01 | Jobs header/roster/output：`ui-jobs/JobListAction` | jobflow、session/jobs、frame/jobs | 仅单测/夹具；65 映射 | mounted roster、输出 cursor/错误、live→settled、重连 |
| TASK-02 | Jobs 两次确认 kill、输出展开、clear；依据：同上 JobListAction | jobflow control/state | 仅单测/夹具 | 真任务动作未执行；clear 与 kill 不混淆；关闭 popup 不重复 mutation；不凭读取重试推定可重跑任务 |
| TASK-03 | schedule catalog/detail/history：`ui-schedule` | ui/schedule、schedule store | 仅单测/夹具；65 映射 | 真实 catalog/history、时区/规则/结束态及延迟加载 |
| TASK-04 | schedule create/edit/run/cancel/delete；依据：TaskManager/TaskDetail | schedule controller/actions | 仅单测/夹具 | 隔离调度真实验收；不能擅自触发模型或投递 |
| PROJ-01 | Project rows/menu/modal/directory：`V/dsh-project/client/client.js` | ui/projects/sidebar/modal、projectflow | 仅单测/夹具；65 映射 | 当前 runtime 模块/桥 ready 后实际加载、回退、目录目标 |
| PROJ-02 | task cards/queue/work/team/observer；依据：原版 ProjectTaskCard、请求投影 | projects observer/task_cards、project request adapters | 仅单测/夹具 | 请求 revision、队列 before-ID、首次详情、缓存非当前卡及晚到隔离 |
| PROJ-03 | notes/preferences/charter/materials/documents/deliveries；依据：原版 Project routes | projectflow materials、projects documents/deliveries/assets | 仅单测/夹具 | 实际文件读写、16KB 限制、失败保留、归档只读 |
| PROJ-04 | reports/history/process/activity/diff；依据：原版 Project routes | projects history/process/activity/detail、diff | 仅单测/夹具 | 当前与历史来源、真实文件/报告读取、可用动作与失效 revision |
| PROJ-05 | rename/archive/request lifecycle/dirty transitions；依据：原版 Project lifecycle | projectflow lifecycle、projects lifecycle | 仅单测/夹具 | Project archive 与 request archive 不混淆；实际 mutation 未执行 |

## Remote、Whale/Bot 和桌宠

| ID | 原版功能与依据 | Go 对应实现 | 实际证据状态 | 缺口与验收点 |
|---|---|---|---|---|
| REMOTE-01 | remote machines/workspaces/files/tools；依据：ui-settings-remote、`V/dsh-remote` client | ui/remote、remote store、remote workbook | 有实现；实际未验证 | 连接/断开/鉴权、远端目标、读/写/权限、切换和重连 |
| REMOTE-02 | phone Gateway/pairing/devices、LAN/relay topology；依据：原版 GatewaySettingsTab、`D/relay` | phoneremote、gateway/phone device editors | 有实现；实际未验证 | 真实设备/网络未验收；legacy token 不能当新配对凭据 |
| REMOTE-03 | SSH tunnel/identity/host validation；依据：原版 SSH/remote desktop contributions | desktop/sshremote、remote SSH wiring | 有实现；实际未验证 | host key、取消/断开、错误、配置重新加载；不更改用户网络/安全 |
| REMOTE-04 | Channels/accounts/IM configuration and messaging；依据：`V/dsh-im`、`D/main/dsh-im-desktop.js` | ui/im、im store/settings | 有实现；实际未验证 | 平台授权、配置与对话/投递分开验收，不擅自发送 |
| BOT-01 | Whale settings、任务、routine/trigger；依据：`V/dsh-whale`、`V/dshbot` client | ui/assistant tasks/routines/Whale settings、stores | 有实现；实际未验证 | 不只验收 settings；创建/编辑/运行/取消及 owner/lifecycle |
| BOT-02 | Bot/联系人、头像、可见性、复制/删除、目标/记忆；依据：dshbot client registrations | assistant bot_*/contacts、cmd controllers | 有实现；实际未验证 | 单项业务能力和实际持久化未被 65 组件台账覆盖 |
| BOT-03 | rooms/members/runtime/sender policy/custom panels/notifications；依据：dshbot/Whale contributions | assistant room_*、custompanel、botnotify/bridgealerts | 有实现；实际未验证 | 真实成员、路由、消息/通知及关闭/切换取消；不能以 UI 卡片显示代替 |
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
| DESK-06 | 更新检查/下载/取消/提示、安装/回滚；依据：`D/main/update*`、update-dialog | desktop/update、update_attention、startuprec | 部分单测；实际未验证 | 网络/更新器/安装与回滚未执行，明确版本和包来源 |
| DESK-07 | installer brand/language/welcome/license/finish/uninstall；依据：原版 installer/packaging resources | scripts/nsis、installer branding | 静态/定向证据；实际安装未验证 | 打包工具、包构建、可逆安装/卸载分别记录，不以资源检查推定安装成功 |
| SAVE-01 | 普通配置与未知字段保留、并发原版/Go写入；依据：`D/main/config.js` | internal/config、workspace preferences merge | 定向回归通过；实际多应用写入未验证 | 当前 per-Host 三路合并/CAS修复不等于跨进程完整验收；同字段冲突须保留 |
| SAVE-02 | 凭据加密/引用、账户登出及安全存储状态；依据：原版 config-credentials/account/profile operations | credentials、credentialrefs、accountlogout | 有实现与测试；实际未验证 | 不读取或输出用户凭据；用隔离凭据验证写入/迁移/清理 |
| SAVE-03 | Host settings 原子修改、失败草稿保留、namespace owner；依据：原版 settings/config-form stores | store/settings/body_preferences/plugin_body | 定向回归通过；实际 Host 未验证 | 每个 served namespace 的写入/重读与 Host 变化，不能用单个成功 schema 推全体 |
| SAVE-04 | 草稿/附件/文件/布局/预览/terminal/shortcut 的保存与恢复；依据：原版各 runtime share/persist/draft | recovery、config、workspace sessions、files drafts/preferences | 部分测试；当前候选实际重开未验证 | 持久状态、仅内存状态和不可重放操作分开；不得复播旧 shell/model mutation |

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
