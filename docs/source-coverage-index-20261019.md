# 原版模块源码覆盖索引（2026-10-19）

**全部原版模块是否已经逐功能、逐状态对照：否。** 下表覆盖当前验收清单登记的全部 ID；这证明范围登记完整，不证明原版每条动态插件贡献都已穷尽，也不把本轮分配项收尾写成全功能完成。

原版源码只读位置为 `C:/AI/Deepseek-Harness-Desktop`。下表沿用验收表缩写：D=`src/`，V=`vendor/`，P=`vendor/deepseek-harness/packages/client/`。Go 路径以本仓库为根。

等级只描述实际源码阅读：`behavior-compared` 是该行已列子路径的行为对照，不是整行穷尽；`entrypoint-sampled` 是入口或局部状态抽查；`not-reviewed` 是尚无足够阅读证据。各专用记录保留精确的已审子路径和未审边界。缺真实运行证据不能改写为缺实现。

必须分开保存四种结论：源码实现与覆盖、普通回归、真实 Host 操作、Windows 实机与原版同态视觉验收。历史 Host/截图 receipts 只适用于各自记录的版本；不能继承到当前候选。

当前源码差异状态：

- 已关闭的本轮差异及本地提交见 [实现批次记录](source-parity-implementation-batch-20261019.md)，包括生命周期、逐卡动作、附件退休、Next Prompt、Skills Hub、Bot/Room/IM 等具体路径。
- Market 图库、README GIF/WebP、限定 AVIF 序列及静态 SVG 扩展已提交源码；后续 Schedule/Jobs、question 收据与草稿、恢复 checkpoint、桌宠穿透、安装事务也已有具体源码和待运行回归。不能将这些提交标为验收通过。
- 原版 128/1440 GPU fluid、Beam 尚不一致的图层，以及仍不支持的 SVG 构造属于代码/效果差异；既有 GPU 深度工作优先级暂停仍单独保留，不伪装成截图工具阻塞。
- 普通回归 run02：source `221dba0`，`../evidence/source-parity-unified-20261019T161408Z/` 保存日志和 receipts；仓库全量 exit 1，受影响 UI SDK 全包 exit 0。实际编译、AVIF、字面文本与焦点等问题和旧契约/时钟夹具分别处理；未构建新候选，没有全产品通过结论。
- 真实 Windows 原版/Go 截图及物理输入验收仍缺支持的原生捕获/输入能力。账户、设备、模型、系统安装等各自的前提和未验收状态继续保留在 [完整验收矩阵](native-full-function-acceptance-20261019.md)。现用程序、已交付目录和用户配置未替换。

覆盖等级按各专用记录与其后续补充汇总，每个标签只覆盖链接中已读的选定子路径。即使某行选定行为已比较，其未审子状态、动态贡献或平台行为仍可能是 `not-reviewed`；不因总表每行都有证据就声称全功能完成。后续真实阅读可以更新等级，编译或测试数量不能。

| ID | 原版登记入口 | Go 登记入口 | 当前源码阅读等级 | 详细范围 |
|---|---|---|---|---|
| BOOT-01 | 启动器、workspace/启动偏好、内核准备：`D/renderer/launcher*`、`D/main/window.js` | `internal/ui/launcher/`、`internal/ui/boot/`、`cmd/whaleisle/main.go` | behavior-compared | 仅启动/保护性生命周期/导入界面已修子路径；非全部启动异常穷尽 |
| BOOT-02 | 真实 Host/Node/插件闭包选择：`V/harness-upstream.json`、原版 `scripts/production-runtime.js`、`scripts/runtime-instance-graph.js`、安装资源 | `internal/harness/`、`internal/paths/`、`scripts/session-runtime.mjs`、构建/打包脚本 | behavior-compared | [选定范围与未审边界](source-coverage-runtime-recovery-20261019.md) |
| BOOT-03 | 会话日志严格准入和版本恢复：原版 session/persistence/format catalog | `internal/harness/session_runtime.go`、session runtime admission | behavior-compared | [选定范围与未审边界](source-coverage-runtime-recovery-20261019.md) |
| BOOT-04 | 内核退出、启动恢复、安全模式、重启：`D/main/harness-controller.js`、`plugin-recovery-actions.js`、launcher | `startup_recovery.go`、`internal/startuprec/`、harness supervisor | behavior-compared | 仅启动/保护性生命周期/导入界面已修子路径；非全部启动异常穷尽 |
| BOOT-05 | 旧数据导入、类别选择、冲突和中断恢复：`D/main/data-import*`、`import-transaction.js` | `internal/desktop/importer/`、`import_flow.go`、`import_recovery.go` | behavior-compared | 仅启动/保护性生命周期/导入界面已修子路径；非全部启动异常穷尽 |
| BOOT-06 | 恢复草稿、选择、文件标签、视图布局：原版各 store/persist | `internal/recovery/`、`cmd/whaleisle/recovery.go`、workspace sessions | behavior-compared | [选定范围与未审边界](source-coverage-runtime-recovery-20261019.md) |
| CHAT-01 | 会话列表、搜索、分组/排序、置顶：`P/ui-workspace/src/client/index.ts`、Rows/navigation/pin-order | `internal/ui/sidebar/`、workspace navigation、session store | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-02 | 新建、改名、归档/恢复、删除及分支来源：`P/ui-workspace/`、`ui-session/` | sidebar/session menu、session lifecycle、ancestry | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-03 | history load/resume/follow、切换与重连：`P/ui-session/`、ConversationMainPanel/ConversationSession | `internal/store/session/`、frame conversation phase | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-04 | 空白 Hero、工作区/目录入口和首条消息过渡：HeroShell/InputBar | `internal/ui/frame/`、composer、workspacepicker | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-05 | 中文输入、选区、撤销/重做、换行、焦点：InputBar editor | `internal/ui/composer/draft_*`、MyGo Windows text input | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-06 | 草稿/附件保留、输入区尺寸与拖动：InputBar、draft/resize preferences | composer resize/attachments/state、recovery | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-07 | 发送、流式回复、queue/steer、停止与忙碌 Enter：submission settings/InputBar | session composer submit/pending、composer actions | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-08 | Slash/@命令、补全、skill/文件引用：`ui-input-trigger`、`ui-commands`、`ui-skill`、`ui-reference` | composer suggestions/commands、file mentions、skill controls | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-09 | model/preset/权限选择及菜单：`ui-model-selection`、`ui-agent-preset`、`ui-permission-presets` | composer model/permission commands、preset seat | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-10 | 文件/图片上传、粘贴、拖放、移除：`ui-file-upload`、`ui-attachment` | composer attachment/paste、clipboardimage、attachment cache | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-11 | 语音输入/录制取消：原版 InputBar 工具与 desktop voice contribution | voicecapture/voiceinput、composer voice controller | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-12 | Markdown、代码、思考、流式文本、工具消息：`ui-chat`、`ui-tool` | `internal/ui/chat/`、MyGo Markdown、tool lifecycle | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-13 | 消息复制/引用/编辑重发/分支/反馈：`ui-message-edit`、`ui-message-feedback`、MessageIconActions | chat actions/edit/quote/history/feedback | behavior-compared | [选定范围与未审边界](source-coverage-conversation-interactions-20261019.md) |
| CHAT-14 | 图像、引用、交付物、工具文件链接：`ui-reference`、`ui-deliverables`、`ui-attachment` | chat citations/images/deliverables/tool files | behavior-compared | [选定范围与未审边界](source-coverage-conversation-interactions-20261019.md)；[桌面 Files 与 Host 文档分流纠正](source-review-native-file-references-20261019.md)，不能以历史引用一律认定 Host-only |
| CHAT-15 | Workflow run/phase/member 专用呈现：`ui-workflow-run/index.ts`、workflow-definition | `internal/workflowrun/`、`ui/chat/workflow.go` | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-16 | 子代理、层级、导航与详情：`ui-subagent`、`ui-agents-panel` | `internal/ui/agents/`、session agent navigation | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-17 | 轨迹、请求头、上下文、压缩/历史范围：`ui-trajectory`、conversation contributions | chat trajectory/context/manual compaction、session inspector | entrypoint-sampled | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-18 | Goal/Plan/Todo、下一提示、使用量/费用：`ui-goal`、`ui-plan`、conversation contributions | chat plan/todo/performance、sessioncost、goal read state | behavior-compared | [选定范围与未审边界](source-coverage-conversation-20261019.md) |
| CHAT-19 | 工具权限批准/拒绝、原因与作用域：`ui-approval` | `internal/ui/interaction/approval*`、interaction store | behavior-compared | [选定范围与未审边界](source-coverage-conversation-interactions-20261019.md) |
| CHAT-20 | 用户问题、编辑答案、等待/历史、计划审核：`ui-user-questions` QuestionComposer/PlanReviewPanel | interaction questions/plan_review、chat question_reply/submitted_plan | behavior-compared | [选定范围与未审边界](source-coverage-conversation-interactions-20261019.md) |
| CHAT-21 | Visual reply 预览、发布、历史读取、隔离交互/导出：`ui-visual-replies` | chat visual_reply、htmlpreview、visual reply cmd adapters | behavior-compared | [选定范围与未审边界](source-coverage-conversation-interactions-20261019.md)；错误分类和初始高度另见 [后续记录](source-coverage-visual-reply-20261019.md) |
| WORK-01 | 工作区切换、添加、改名、移除、无目录：`ui-workspace/WorkspacePicker` | `ui/workspacepicker/`、workspace navigation | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-02 | Miller 目录浏览、路径编辑、隐藏项、创建和失败重试：`ui-directory-picker-browse` | `ui/dirpick/`、directory_picker.go | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-03 | 文件树、展开/刷新、路径/文件引用及操作菜单：`ui-files/apply.ts`、FileTree | `ui/files/` tree/source actions、workspacefs | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-04 | 编辑、保存、自动保存、外部修改和脏关闭确认：FileSaveCoordinator/FileClosePrompt | files autosave/reload/draft、file editor/controller | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md)；原生未绘制编辑缓冲及 IME 选区提交快照已补源码与回归准备，实际验证待执行 |
| WORK-05 | 文本/代码/Markdown 预览、行定位/搜索/换行/只读：documentpreview text/code/markdown registrations | files text_read/text_preview/wrap/document_surface | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-06 | HTML、图像、PDF、Office、Excel 预览：documentpreview index.ts 的各类型注册 | files media/image/table、pdfpreview、mediapreview、webview | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md)；桌面 Files 图片保持本地读取，AVIF 复用序列时间轴；[分流纠正](source-review-native-file-references-20261019.md) |
| WORK-07 | 标签、停靠/浮动、工具区尺寸、关闭与恢复：`ui-sidebar-right`、`ui-dockkit` | frame workspace panes、files tabs/floating、workspace sessions | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-08 | Browser 标签、URL 导航、标题、历史、录制/失败：`ui-sidebar-browser`、desktop guests | browserpreview、webview、browser_workspace、browserrecording | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-09 | Git 状态、diff、分支选择/创建：`ui-git/apply.ts`、BranchMenu/CreateBranchDialog | desktop/git、diffsurface、Git cmd adapters | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-10 | Git commit/publish 及授权/失败：CommitDialog/PublishDialog | desktop/git、git action wiring | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-11 | 会话工作区终端、split/tab、输入/resize/隐藏：`ui-user-terminal` TerminalDrawer/Pane | `ui/terminal/`、Ghostty/ConPTY、terminalruntime | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| WORK-12 | Host 终端、shell/会话选择和权限：`ui-sidebar-terminal`、settings-shell | hostterminal store、terminal_target、host terminal UI | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| SET-01 | SettingsRoot/navigation/onboarding：`ui-settings-general`、`ui-settings-account` | settings modal/navigation/sections、account onboarding | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-02 | account：`ui-settings-account/AccountSection` | settings account/signout、accountlogin/logout | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-03 | general：GeneralSection、instructions/permission contributions | settings desktop/custom_instructions/permission_defaults、settings store | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-04 | general 子项：shortcuts、developer tools、session log upload | settings_shortcuts、desktop preferences、session-log-deepseek row | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-05 | appearance：AppearanceSection/font/wallpaper/cursor | settings appearance/theme_font/wallpaper/pointer、ui/theme | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-06 | interface：InterfaceSection、Beam/TypingFx | settings interface/effect_preferences、composer effects | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-07 | models：ModelsSection/provider forms/discovery | models、model_configuration、provider_profiles/discovery | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-08 | plugins：PluginsSettingsSection、inventory/client modules | settings plugins/plugin_body/plugin_inventory | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-09 | skills：SkillsSection/groups/editor/catalog | settings skills、skill editor/directory、skill stores | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-10 | market：MarketSection/desktop override | ui/market、settings embedded content | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md)；图库已实现，未据此提升整个 Market 覆盖等级 |
| SET-11 | mcp：McpSection/import/catalog/auth | settings mcp/form/import/catalog、mcpconfig | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-12 | remote：RemoteSection/GatewaySettingsTab/Channels | ui/remote、ui/im/settings、phone/SSH adapters | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-13 | agent-presets：AgentPresetSection/guide/seat | settings agent_presets/guide、preset manager | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-14 | usage-stats：`V/dsh-usage-panel/.../StatsSection` | ui/usage embedded/stats/dimensions/time_range | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-15 | pet：PetSection/Whale contribution | settings pet/look、assistant_whale_settings | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-16 | remote-workspace：`V/dsh-remote` settings contribution | ui/remote/settings_body、remote workspace adapters | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-17 | about：AboutSection/current version/update entries | settings about、update controller | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| SET-18 | served 插件配置：subagent/web-search/agent-loop/shell 等 | settings plugin_body/model_body、captured namespace binding | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| PLUG-01 | 独立 profile Plugins：`ui-plugin-manager/PluginManagerPage` | internal/pluginmanager、ui/pluginmanager、native_plugin_manager | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| PLUG-02 | 通用 Discover/detail/source/pagination：`ui-plugin-market` | pluginmanager discover/dialogs | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| PLUG-03 | install/progress/log/cancel/uncertain reconciliation；依据：manager-store/install terminal、plugin market | pluginmanager install/controller、event wiring | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| PLUG-04 | 精选 Market tabs/detail/README/install/update/remove：`ui-settings-market`、`D/main/dsh-market-desktop.js` | ui/market、desktop/market | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| PLUG-05 | README 媒体播放/失败 alt；依据：原版 Markdown/media | market readme_images、Markdown/SVG/image support | behavior-compared | [选定范围与未审边界](source-coverage-settings-20261019.md)；动画格式与 SVG 具体能力边界见 [媒体记录](source-coverage-media-20261019.md) |
| TASK-01 | Jobs header/roster/output：`ui-jobs/JobListAction` | jobflow、session/jobs、frame/jobs | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| TASK-02 | Jobs 两次确认 kill、输出展开、clear；依据：同上 JobListAction | jobflow control/state | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| TASK-03 | schedule catalog/detail/history：`ui-schedule` | ui/schedule、schedule store | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| TASK-04 | schedule create/edit/run/cancel/delete；依据：TaskManager/TaskDetail | schedule controller/actions | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| PROJ-01 | Project rows/menu/modal/directory：`V/dsh-project/client/client.js` | ui/projects/sidebar/modal、projectflow | entrypoint-sampled | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| PROJ-02 | task cards/queue/work/team/observer；依据：原版 ProjectTaskCard、请求投影 | projects observer/task_cards、project request adapters | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| PROJ-03 | notes/preferences/charter/materials/documents/deliveries；依据：原版 Project routes | projectflow materials、projects documents/deliveries/assets | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| PROJ-04 | reports/history/process/activity/diff；依据：原版 Project routes | projects history/process/activity/detail、diff | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| PROJ-05 | rename/archive/request lifecycle/dirty transitions；依据：原版 Project lifecycle | projectflow lifecycle、projects lifecycle | behavior-compared | [选定范围与未审边界](source-coverage-project-tasks-20261019.md) |
| REMOTE-01 | remote machines/workspaces/files/tools；依据：ui-settings-remote、`V/dsh-remote` client | ui/remote、remote store、remote workbook | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| REMOTE-02 | phone Gateway/pairing/devices、LAN/relay topology；依据：原版 GatewaySettingsTab、`D/relay` | phoneremote、gateway/phone device editors | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| REMOTE-03 | SSH tunnel/identity/host validation；依据：原版 SSH/remote desktop contributions | desktop/sshremote、remote SSH wiring | entrypoint-sampled | [选定范围与未审边界](source-coverage-settings-20261019.md) |
| REMOTE-04 | Channels/accounts/IM configuration and messaging；依据：`V/dsh-im`、`D/main/dsh-im-desktop.js` | ui/im、im store/settings | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md)；限 channel 配置/状态及 Office 控制面；IM delivery/全消息类型/平台扩展未完整比较 |
| BOT-01 | Whale settings、任务、routine/trigger；依据：`V/dsh-whale`、`V/dshbot` client | ui/assistant tasks/routines/Whale settings、stores | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md) |
| BOT-02 | Bot/联系人、头像、可见性、复制/删除、目标/记忆；依据：dshbot client registrations | assistant bot_*/contacts、cmd controllers | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md) |
| BOT-03 | rooms/members/runtime/sender policy/custom panels/notifications；依据：dshbot/Whale contributions | assistant room_*、custompanel、botnotify/bridgealerts | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md)；限 Room/主要 thread/Bot 通知规则；custom-panel 原版入口仍 not-reviewed，Bridge 提醒仅抽样 |
| PET-01 | 桌宠开关/皮肤/位置/行为/拖动/点击穿透；依据：`D/main/desktop-pet*`、desktop-live2d、renderer pet* | desktop/pet、pet settings/cmd adapters | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md)；限固定设置和透明区输入门控；完整渲染、动画、跨屏/DPI 未完成对照 |
| PET-02 | pet chat/look/care/growth/personality/notifications；依据：原版 pet-chat/stats/growth/settings | desktop/pet、quickchat、pet controllers | behavior-compared | [选定范围与未审边界](source-coverage-assistant-pet-20261019.md)；限菜单、聊天路由及 care/stats/growth；direct fallback 和完整通知链仅抽样 |
| DESK-01 | 文件/运行/视图/帮助菜单：`D/main/menu.js` | application_menu.go、application shortcuts | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-02 | tray show/launcher/settings/market/restart/pet/quit：tray-menu.js | desktop_tray/menu/notice | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-03 | caption/drag/resize/snap、窗口焦点/关闭/退出确认；依据：window/chrome/close-behavior/native-window-motion | ui/chrome、frame、taskguard、native window | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-04 | 全局/应用快捷键、命令参考、菜单分发；依据：shortcuts.js、ui-shortcuts Editor/Reference | shortcuts/shortcutprefs、unified/application shortcuts | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-05 | deep links、单实例、开机启动、外部打开；依据：原版 main/preload desktop integrations | deeplink、MyGo app integration、open-in-app | entrypoint-sampled | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-06 | 更新检查/下载/取消/提示、安装/回滚；依据：`D/main/update*`、update-dialog | desktop/update、update_attention、startuprec | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| DESK-07 | installer brand/language/welcome/license/finish/uninstall；依据：原版 installer/packaging resources | scripts/nsis、installer branding | behavior-compared | [选定范围与未审边界](source-coverage-files-desktop-20261019.md) |
| SAVE-01 | 普通配置与未知字段保留、并发原版/Go写入；依据：`D/main/config.js` | internal/config、workspace preferences merge | behavior-compared | [选定范围与未审边界](source-coverage-persistence-20261019.md) |
| SAVE-02 | 凭据加密/引用、账户登出及安全存储状态；依据：原版 config-credentials/account/profile operations | credentials、credentialrefs、accountlogout | behavior-compared | [选定范围与未审边界](source-coverage-persistence-20261019.md) |
| SAVE-03 | Host settings 原子修改、失败草稿保留、namespace owner；依据：原版 settings/config-form stores | store/settings/body_preferences/plugin_body | entrypoint-sampled | [选定范围与未审边界](source-coverage-persistence-20261019.md) |
| SAVE-04 | 草稿/附件/文件/布局/预览/terminal/shortcut 的保存与恢复；依据：原版各 runtime share/persist/draft | recovery、config、workspace sessions、files drafts/preferences | entrypoint-sampled | [选定范围与未审边界](source-coverage-runtime-recovery-20261019.md)；恢复/checkpoint 已对照选定行为，Browser/全部 viewer/终端与迁移仍有未审范围 |
