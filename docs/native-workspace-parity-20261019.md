# Windows 主工作区还原记录（2026-10-19）

本轮按原版 TSX/CSS 的组件树、布局、状态条件和事件行为还原 MyGo 原生 UI。源码等价实现、原生回归和最终原版实机对照分别记录；编译或测试通过不等于视觉验收通过。

## 实际基线与保护范围

- 现用 Go 仓库：`C:\AI\WhaleIsle-MyGo`，`main@b7fb9f9`。PID24668 使用仓库根目录 `whaleisle.exe`；嵌入构建信息为 Go1.27.2、b7fb9f9、vcs.modified=true，文件写入时间08:53:05（本机+08:00）。
- 原版进程：PID32964，`C:\软件\Whale Isle\Whale Isle.exe`；FileVersion0.3.9，ProductVersion0.3.9.0。原版源码只读参考为 `C:\AI\Deepseek-Harness-Desktop@e353e8e13f`，未假定该源码 SHA 等于已安装 EXE 的发布源码。
- 隔离工作副本：`C:\Users\48818\Documents\Codex\2026-10-19\task-2\WhaleIsle-MyGo`；分支 `codex/native-workspace-parity-20261019`；origin 指向 `https://github.com/ChisaAlter/WhaleIsle-MyGo.git`。
- 原仓库三处未提交崩溃修复（main.go、quickchat_view.go、resize.go）逐字保存在独立基线提交 `aeb77c5`；未覆盖原工作副本，未动原版、鯨 code、PR#2 或远端 main。
- 两个现用实例 PID/启动时间在本轮只读复核中保持不变。没有发送模型请求、读取凭据、删除会话、清配置、改网络/安全设置、发布或触发 CI。

## 原版规则 → 原生实现

以下原版路径前缀均为 `vendor/deepseek-harness/packages/client/`。

| 原版规格 | Go 对应 | 本轮处理 |
|---|---|---|
| `ui-layout/columns.ts`；conversation column W=clamp(680,column×0.64,920)，卡片W+32 | frame.go、conversation_width.go | 保留已有列契约；Hero/active使用同一输入卡边界，不重复扣Hero24侧距；保留手动宽度 |
| `ui-conversation/.../ConversationMainPanel.tsx`；`ConversationSession.tsx`；sessions/session.ts | session/store.go、follow.go、lifecycle_events.go、composer_pending.go、composer_submit.go；conversation_phase.go | 历史loading/ready/error独立于消息数；Host summaryBlank例外；按Host/会话保存PromptAttempted；过期generation隔离；共同错误与重试 |
| `ConversationRoot.module.css`、`HeroShell.module.css` | frame.go、conversation_phase.go、conversation_header.go、session_ancestry.go | Hero底内距32，标题26/32字重500、鱼34、gap10/7；标题/操作/工具/角落分组；560DIP阈值收起文字；使用原版SVG |
| `InputBar.module.css`、`InputBar.tsx` | composer.go、draft_input.go、font_preferences.go、tool_actions.go、command_launcher.go | 原生/后备输入稳定key；14字体/24行高，变更为fontSize+10；自然最小36/52；工具回现有+菜单；运行态Send/Stop规则；窄栏模型/发送分组 |
| 原版非空附件列表条件 | attachment_display.go | 修正非nil DraftState零附件仍挂载210DIP轨道的直接根因；错误反馈独立保留；不通过高度上限或重置配置掩盖 |
| `ui-workspace/rows/Rows.module.css`、`Rows.tsx`、`WorkspaceBrowser.tsx` | sidebar.go、session_menu.go、workspace_heading.go、workspace_groups.go | 会话32、工作区34、状态槽16×20；选中仅换底色；操作进菜单；搜索/多选同排，Esc清空收起；无重复工作区层级 |
| `ui-theme/styles/base.css`、`design-platform.css`、`derive.ts` | theme/preferences.go、theme.go；cmd/theme_font_preferences.go | 原版完整字体回退；保留显式覆盖；pressed/hover交互token；当前深色accent#7aaaff |
| 原版GoalBar无目标返回null | frame.go | 无实际计划/目标/队列/todo/编辑器时不挂载占位计划面板 |
| `MessageIconActions`、`TurnTailNodeView`、`FeedbackDialog` | chat/message_actions.go、message_icons.go、feedback_editor.go、feedback_dialog.go | 尾部28DIP图标座/gap8，历史hover/focus显现；按持久回答聚合复制；488DIP反馈模态替代常驻诊断编辑器 |
| 原版dock/pane header及上下文菜单 | workspace_panes.go、workspace_dock_tabs.go、workspace_dock_drop.go | 38DIP标题栏、28DIP标签/角落工具、溢出菜单；方向/浮动/重置回菜单，拖拽期间才显示落点 |

## 验证及证据

执行器使用已缓存的 Windows Go1.27.2：`C:\Users\48818\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.2.windows-amd64\bin\go.exe`。`Program Files\Go` 的基础版本实际为1.27.0，未降低仓库版本要求。GOMODCACHE、GOCACHE、GOTMPDIR 均放在本任务目录；原缓存缺少完整校验元数据的依赖由普通 Go 模块下载补齐，未修改系统网络设置。

- session整包：通过5.736s；包括真实本机假HTTP/WebSocket、回放、错误重试、快速选择切换、首次发送失败。沙箱首轮拒绝回环连接的结果保留为环境失败；允许本机测试服务后通过，未修改测试断言制造通过。
- sidebar整包：通过2.118s；新增层级、取消置顶、重命名路径定向通过1.349s。
- theme整包：通过1.717s；最终cmd整包通过3.832s，含字体/原生交互token/5DIP滚动条映射。
- composer：最终整包通过2.807s；覆盖零附件空白、工具菜单、自然/保存尺寸、运行态Send/Stop、输入身份与IME client选择范围。首轮菜单单击缺陷已修复后验证。
- frame：最终整包通过6.514s，重新生成宽/窄、明/暗、右栏、长标题、Hero/active/settling/error证据；覆盖空计划占位、同宽输入卡、焦点保持及dock交互。
- chat：最终整包通过3.945s；包括原版消息尾部图标、反馈模态布局与开关/失败草稿/键盘焦点/提交中关闭。
- feedback：最新定向生命周期回归通过3.018s；关闭后重开草稿不被旧请求覆盖，重复成功通知不吞失。
- `d7d07fc` Windows/amd64候选构建通过；嵌入Go1.27.2、vcs.revision=d7d07fcbca9894771e53f5ebc8a52447c5a60f80、vcs.modified=false。独立截图程序PID35932退出0；stderr为空。

证据目录在工作副本外：`..\evidence\frame`（原生软件渲染器PNG/JSON，包含400/900/1500中栏、Hero→首条消息、长祖先标题、settling、失败、窄/宽窗与明暗主题）；`..\evidence\native`（独立MyGo进程调用内置Window.CapturePage所得PNG）。

原生捕获实测客户区1200×800DIP、图像1800×1200px、scale1.5。测试窗口Hidden=true，未抢占现用会话；无Host、凭据或模型。它证明本机原生布局/字体栅格路径可运行，不是原版同状态对照。可复现捕获源保留在忽略目录 `.toolchain/native-parity/main.go`。

已实际检查最终Hero、普通会话明/暗布局、400DIP长祖先顶栏和反馈窗口。Go修复前后同尺寸/同DPI的Hero对照显示：零附件210DIP空轨道和双语常驻工具条消除，输入卡恢复紧凑自然高度，助手预设回到工作区行右侧。主工作区渲染中右栏不再占用多排方向/重置工具，消息尾部回到图标行。软件渲染证据不是现用窗口或原版截图。

Library附件（`d7d07fc`检查点，均为隔离fixture；对应元数据见 `..\evidence\library-artifacts-final.json`）：

| 画面 | file_id | library_file_id |
|---|---|---|
| 原生Hero 150% | file_0000000026f881fd818bd0cd9ada4fd8 | libfile_8ef81c087e14819188949b069373a1bb |
| 主工作区软件渲染 | file_00000000afd481fd82284b4886a6519e | libfile_0e4f16b08fc08191be8ab49cfbdc2944 |
| 反馈模态软件渲染 | file_000000000d4081f8a9dfa32a8439ac6d | libfile_2b9e5fb4dc908191a5a4e434b2f85158 |

Standards与Spec独立只读审查覆盖完整 `aeb77c5..d7d07fc`。曾提出的纯图片回答动作栏问题经原版projection+view联合核对后撤回，未据此增加原版没有的动作。

另核实同一持久回答“文本＋图片”被Snapshot拆行为同SourceMessageID/Seq的块后，UI分支条件错误读取末尾图片空Body，而Store的原始事件允许分支。已仅在UI条件判断中还原该同一事件，再调用既有CanForkTurn；独立后续图片、工具和错误边界保持禁用。实际分支仍由Store复核。定向回归通过2.309s，包含实际按钮回调与真实endSeq。

## 本地检查候选

- 产品提交：`d7d07fc`（主工作区还原）、`ce4efcb`（同生命周期混合内容分支修正）。后者两个chat文件的Standards与Spec复查均无发现。
- 最终EXE：`..\evidence\whaleisle-native-parity-ce4efcb.exe`。Windows/amd64，Go1.27.2；嵌入 `vcs.revision=ce4efcba6d190ec772553e4e6c8f78c1d591702f`、`vcs.modified=false`。
- SHA256：`A8447152F95CF7F499A29A53A733DFCD5F952C34A8978D88EA5E2F24A66E6217`。
- 该原生EXE已构建，未启动为用户工作实例，也未作安装包验收。实际窗口证据来自同分支独立离线fixture；最终新增修正只影响混合内容分支的可用性，未改变上述Hero/普通文本/反馈窗口画面。
- `go.mod`、`go.sum`、`.github`无修改；最终构建前工作区干净。现用Go PID24668和原版主PID32964经只读CIM复核仍使用原路径/原启动时间。
- 本轮需求实现与自动/隔离操作验证有证据；同状态原版实机视觉验收仍未完成。该候选供主工作区检查，不标记完整产品还原或可合并。

## 明确保留的缺口

- 未获得原版与最终Go版的同窗口/同DPI/同字体/同页面数据实机对照，未签署像素或视觉验收。当前会话缺少computer-use所需node_repl，CUA原生API禁用；未绕过这些限制。原版只读源码仍用于推进实现，未把缺截图当作实施前提。
- 当前用户Host接受的composer手动高度/宽度未能确认：只读检查已知公开settings.json未发现对应持久化key；没有清除设置。回归分别覆盖自然高度和保存高度220DIP。
- 原生IME client身份及选区测试已通过，实际Windows中文输入设备/输入法候选窗交互尚未验证。
- 现有原生API不能精确表达原版鱼标的CSS旋转中心/位移/缓动、滚动条仅悬停滑块显现、反馈遮罩模糊和composer锚定通知位置；未为这些绘制差异修改工具包或搭新框架。
- 其他页面（设置、插件、任务、账号等）本轮未逐页还原；富内容/复杂工具卡、非空计划的完整视觉、全部原版动画尚不能据本轮测试宣称完成。
- CI本轮触发0次，未push；历史累计未通过次数未知，未用本轮任务重置它。

## 第二阶段清点：高频页面与模态

基线 `0b85f86`，保留 `ce4efcb` EXE。先由四条只读审计线检查实际注册、完整组件和桌面覆盖，再划定以下页面批次；此表的“未完成”不是权限阻塞。原版组件路径除注明外，前缀为 `vendor/deepseek-harness/packages/client/`。

### 按严重程度排序的主要差异

| 优先级 | 原版组件 → Go实现 | 清点结果与本批边界 |
|---|---|---|
| 高 | ui-settings-general SettingsRoot/CSS → settings/modal.go/settings.go | 实际CSS为800宽、最大800且viewport−48高、188导航、54内容标题栏；Go为1120×820、240导航和通栏标题。本批恢复外壳、15项注册顺序、活动项、滚动和统一错误显示 |
| 高 | ui-plugin-manager + ui-plugin-market → main_navigation.go / ui/market | 原版侧栏Plugins管理器与桌面精选市场不同；Go合并入口且没有pluginMarket/调用。管理器package/row/item配置数据也缺失，保留为独立待完成范围，不以换标签冒充完成 |
| 高 | ui-settings-market MarketSection/CSS + src/main/dsh-market-desktop.js → ui/market | Go通栏按钮、无活动页签、单列无卡片、空搜索无反馈、详情/确认在正文流中。本批恢复800列、原版页签、响应式卡片、状态及有界模态；精选目录后端不变 |
| 高 | ui-settings-plugin-inventory → settings/plugin_inventory.go | Go无卡片层级且展开符为文字占位；本批恢复双列/窄列卡片、状态点、Loader标识、展开详情和长文本边界 |
| 高 | ui-schedule TaskManagerPage/TaskDetail → schedule/Panel | Go列表、操作排、编辑、历史全部堆在同一滚动区；本批恢复960目录、47%详情、760以下详情接管、Rule/Records、独立滚动、脏草稿保存条及明确loading/empty/error |
| 高 | ui-workspace WorkspacePicker → workspacepicker/picker.go + cmd/workspace_navigation.go | Go居中520弹窗并常驻路径注册字段；本批改为Hero触发控件锚定菜单，工作区身份选择和固定底部Add/No-folder；重命名/移除有界模态 |
| 高 | ui-directory-picker-browse DirectoryBrowser → dirpick/panel.go + main.go旧overlay | Go固定420外壳、列表Column限高但不滚动，返回后才开窗，迟到结果能重新打开。本批恢复680×500有界模态、立即loading、取消/Host身份栅栏、真实列表滚动和固定底部 |
| 中 | ui-jobs JobListAction → frame/jobs.go / jobflow/controller.go | 原版500×≤480锚定popover，挂载即观察并显示实时计数；Go为880居中modal、仅打开时观察且关闭清空。本批不改，单独后续生命周期批次 |
| 中 | ui-schedule ScheduleCatalogAction → chat/schedule_context.go | 原版28图标/336锚定菜单/零项隐藏；Go为消息区常驻任务条，有溢出风险。本批不混入共享顶栏改造 |
| 中 | 原版项目/资源浏览组件 → ui/projects/view.go等 | Go固定200列表和多排操作，项目详情跨历史/材料/团队，尚未逐状态还原。本批仅清点，不泛化为整项目页面改造 |
| 中 | MarketExtras → market详情/历史 | README暂为文本；截图轮播资产和可展开完整操作日志缺少对应数据，当前有限历史不能冒充完整原版记录 |

### 原版实际设置注册清单（共15项）

依据完整index注册与本地插件，按order排列。`archived-sessions`仅有图标映射，未发现真实注册，不另造页面。

| 原版section / order | Go实现 | 本批以外的正文缺口 |
|---|---|---|
| account / −10，仅credential-stored | settings/account*.go | 正文部分对齐；未登录入口保留在General，活动登录仍可返回 |
| general / 0 | desktop.go、通用schema、自定义指令/权限 | 桌面与Host功能分组仍需恢复 |
| appearance / 5 | appearance.go、theme/wallpaper | 正文未完整视觉对照 |
| interface / 6 | interface.go | 原版多项界面功能行缺少专用呈现 |
| models / 10 | models.go | 本批未深入正文审计，不宣称对齐 |
| plugins / 15 | plugins.go、plugin_inventory.go | 本批清单卡片之外的配置/管理层级未完整 |
| skills / 16 | skills.go | 堆叠表单与原版列表/详情不同 |
| market / 17，桌面API gate | ui/market | 本批恢复设置内嵌及精选市场呈现，通用插件发现另列 |
| mcp / 18 | mcp*.go | 正文部分迁移，未逐模态对齐 |
| remote / 18 | ui/remote | Go混合SSH/手机页且离开设置；原版Gateway/Channels分组待恢复 |
| agent-presets / 20 | agent_presets*.go | 正文与弹层部分迁移 |
| usage-stats / 25，本地vendor/dsh-usage-panel | ui/usage | Go外部页且混入账号内容，设置内组合待恢复 |
| pet / 40，桌面bridge gate | pet*.go | 保留能力条件；正文部分迁移 |
| remote-workspace / 45，本地vendor/dsh-remote | 通用namespace；机器UI在ui/remote | 页面组合不对应原版 |
| about / 90 | about.go | Go增加的管理元信息密度与原版不同 |

原版可访问源码足以确定上述页面范围；没有因缺原版当前窗口截图停止实现。后续将按这份明确清单分批处理，不把所有页面归入本轮已完成。

### 第二阶段本批验证与独立审查

本批已实现上述设置外壳、桌面精选市场/清单、自动化任务目录/详情、锚定工作区选择/管理和有界目录浏览器；15项设置正文、独立Plugins管理器、Jobs、Projects与目录Miller多栏/隐藏项/路径编辑不在完成范围。项目页原版依据补充为本地 `vendor/dsh-project/client/client.js`（36px项目行、hover/focus菜单和760px/80vh项目模态），Go对应 `ui/projects`；保留下一独立页面批次。

一次整批门禁（Go1.27.2、`-count=1`）通过：

| 页面/主工作区门禁 | 结果 |
|---|---|
| settings / market / schedule | 2.771s / 1.930s / 1.908s |
| workspacepicker / dirpick | 1.692s / 1.196s |
| frame / composer / chat | 5.286s / 2.496s / 3.461s |
| sidebar / theme / session store | 2.142s / 1.191s / 5.455s |
| cmd/whaleisle | 2.957s |

检查点后的Pet/账号返回入口两行修正，受影响外部导航/账号返回定向回归另通过2.680s，其他页面代码未变。

两条独立审查覆盖完整 `git diff 0b85f86`，包含新增文件；随后只复查受影响修正。Standards指出任务筛选丢失、外部设置与Pet入口残留Market选择，均已修复；Spec指出目录错误缺Retry，已补保存失败目标路径并重试。组合fixture还实际发现嵌套滚动造成市场正文零尺寸，已修成由设置承载滚动的自然内容列。最终两条代码审查均无确认的剩余P1/P2，不代替原版视觉验收。

`..\evidence\secondary` 保存20张PNG，已检查设置市场正常/窄窗、任务目录/详情/窄窗/记录/删除、目录短窗、工作区长列表/重命名、插件清单窄列及详情窄窗。未见本批关键操作越界或互相覆盖；截图均为隔离数据。`..\evidence\frame-phase2` 保存主工作区防退化渲染。部分次级正文仍保留原Go控件样式/文案，不能由本批结构恢复推定为完整视觉对齐。

目录测试只用模拟HTTP，市场目录为内存夹具；没有创建真实任务/目录、安装插件、访问真实Host或发送模型请求。`go.mod`、`go.sum`和`.github`未变；没有CI、推送或用户实例操作。

### 第二阶段本地候选与原生画面

- 产品提交 `e565dcdcc9a8bf44a0037cd3b7f36cfb90860172`；候选 `..\evidence\whaleisle-native-pages-e565dcd.exe`。Windows/amd64、Go1.27.2，嵌入该SHA、`vcs.modified=false`。
- 候选SHA256：`F440CB97C7CB542B30099A2AE8C8D2A81E918FF192A2A8FBE3233AFFD96FC0DD`。
- 之前 `whaleisle-native-parity-ce4efcb.exe` 的SHA256仍为 `A8447152F95CF7F499A29A53A733DFCD5F952C34A8978D88EA5E2F24A66E6217`，未覆盖。
- 独立原生截图进程PID33432退出0，stderr为空；`..\evidence\native-pages-e565dcd\native-settings-market.png`，客户区1200×800DIP、图像1800×1200px、scale1.5。已实际查看设置外壳、15项模拟能力注册导航和市场正文；使用原生窗口/字体栅格，Hidden=true，无真实Host、用户配置、账号凭据或模型。夹具源码 `.toolchain/secondary-parity/main.go` 为任务内忽略文件。
- Library原生截图：`file_00000000887881fdba723cc24f992cd5` / `libfile_a942226cdd8c81919b383c9d3313d8ea`；文件名 `WhaleIsle-MyGo-isolated-native-settings-market-e565dcd-150pct.png`。这是隔离实例，不是现用程序或原版对照。
- 本轮候选未替换、启动为用户工作实例，也未进行安装包验收。原版同状态视觉对照、实际Windows IME及本节列明的后续页面仍未完成；无审批等待阻塞。

## Third-batch integration checkpoint

The fixed remaining scope is settings bodies, Jobs, Projects, the full Host directory browser and independent Plugins. Work continues on the same dedicated branch from 0376ce3; both earlier candidate binaries remain untouched. Source contracts and the third-batch ownership plan are recorded in the existing plan document.

Jobs/header, directory browser, compact Projects/modal routes, the profile Plugins controller/page and the fifteen registered settings bodies now have native implementations. Independent review identified substantive gaps beyond outer page containers: compact Skills/MCP controls, rich provider editing, preset grid/card ownership, Observer task-card modes and install-event wiring. These corrections are part of this batch, rather than a deferred checklist. The final affected-package gate has now passed; clean build provenance is recorded below.

The five Standards findings have dedicated corrections: Host-keyed local drafts, a single atomic captured snapshot/writer, directory UI-only merging after asynchronous commits, retained failed effect edits, uncertain-install reconciliation and content-sized Project modals. Root also binds plugin-manager changed/log/state events to the captured Host owner, and wires source-defined Project queue commands with the displayed request revision and before-request identities. No guessed queue-read endpoint was added: the existing Project projection contains its queue.

Isolated captures include Jobs (evidence/phase3-inspect), Projects (evidence/phase3-projects), General/Interface/Appearance/Pet/About (phase3-settings-snapshots), feature bodies (phase3-body-snapshots), and composed setting-modal checks (evidence/phase3-composition). These are synthetic data, not installed-app certification. A hidden native Account fixture was captured at1200x800DIP/1800x1200px/scale1.5; the attempted second page in that same hidden window was a cached duplicate, explicitly invalidated as evidence. A fresh-process Plugins capture succeeded and was inspected; its evidence details follow below.

Read-only process observation at2026-10-19T02:57:56Z found zero whaleisle.exe processes and the original Whale Isle.exe main PID31876 with direct Node child29456. No process was restarted or stopped. The initial Go PID24668 is historical discovery evidence, not the current running state. Exact observation is retained in evidence/phase3-runtime-observation.json. The executable-adjacent Go runtime candidate lacks plugin-market and dsh-project source; this does not establish the selected runtime of a nonexistent Go process. Generic Discover reports actual unavailable/error results without falling back to curated Market. Actual original-window same-state capture remains unavailable in this execution surface.

### Third-batch final verification

The full fixed source-backed scope is implemented: fifteen registered settings bodies (including the embedded Remote, Remote Workspace, Channels and Usage bodies), mounted Jobs/header actions, compact Projects/sidebar/modal and Observer task-card modes, the full Host directory browser, and the independent profile Plugins manager. This is a native MyGo implementation. Curated Market remains a separate route. Generic runtime capability errors remain explicit.

The integrator closed the final functional review gaps: queue mutation captures its Host/route/request owner atomically; ordinary Project rows can open their listed directory without requiring a delivery card; first-entry task work details read the request, team projection and exact work details through existing read contracts. Cached non-current cards require refresh before current-card actions. These paths have focused fake-transport regressions. Source-known model contribution namespaces retain structured editors and captured binding ownership.

Verification used Go1.27.2, CGO_ENABLED=0 and isolated caches, with -count=1. No real Host/model/configuration was used. All 22 affected packages passed across the final gates:

| Packages | Result |
|---|---|
| settings store / market / composer / chat / theme | PASS: 1.286s / 2.004s / 3.819s / 4.335s / 1.267s |
| remote / usage / im / assistant / pluginmanager / ui-pluginmanager | PASS: 2.281s / 2.083s / 1.864s / 2.719s / 1.213s / 1.698s |
| session store / workspacepicker / dirpick / schedule / jobflow / projectflow | PASS: 6.628s / 0.813s / 0.703s / 1.003s / 0.296s / 0.597s |
| Final five after the last integration changes: cmd / projects / frame / sidebar / settings | PASS: 6.362s / 3.249s / 7.040s / 2.449s / 4.558s |

Additional atomic settings binding regressions passed after the final store binding change (1.136s). Two intermediate cmd compile failures were mistakes in newly added fake test fixtures (method name and install RPC expectation), corrected before the final complete cmd package pass; they are not omitted as passing runs. No CI was dispatched. git diff --check passed.

Independent Standards and Spec reviewers both report zero remaining P1/P2 within the bounded changed scope. This is code/source review, not an assertion of original-app pixel parity. Evidence/phase3-final contains 247 synthetic PNGs; presets were regenerated after the alpha correction, and nine inspected Observer images were synchronized from the corrected source set. Evidence/frame-phase3 contains 41 workspace-state PNGs.

The valid hidden native captures are Account and the fresh-process Plugins settings body at1200x800DIP /1800x1200px /scale1.5. The duplicate cached Account image from the first attempted Plugins capture remains explicitly invalid in evidence/native-phase3-working/capture-verification.json. The valid fresh Plugins SHA256 is ECDB765E76AF57B2CFC994EF7ECAF3A43D7A86AE5FDDB6341CDA4AA227198943. These capture binaries were built from the working tree before final commit and use isolated synthetic data; the separately built clean application candidate has its own provenance below.

Library retained the inspected fresh image as file_00000000f95481fd87c52b1153f803e6 / libfile_9d6e6dd3f8c08191a04e1e30ae7260ac. Its returned version xattr was written to the local file; evidence/library-artifact-phase3.json records the upload.

Remaining acceptance limits: original same-state native screenshot comparison is unavailable because this execution surface cannot capture native user windows. No restart or replacement of user instances occurred. Windows IME, installed-candidate behavior, real Host plugin installation and actual project/remote mutations remain unverified. The nominated executable-adjacent Go runtime lacks original plugin-market/dsh-project sources; selected runtime availability cannot be inferred without an actual running Go instance. Missing capabilities are reported rather than simulated. Original repositories, whale-code, user configuration, main, PR#2, network/security settings and remote branches remain untouched.
### Third-batch clean local candidate

- Product/source commit: `bc95dd10d8501c4f5fc5314fc426b4cb10fe1b20` on `codex/native-workspace-parity-20261019`; clean tree at build time.
- Build: `go build -trimpath -o ..\evidence\whaleisle-native-complete-bc95dd1.exe ./cmd/whaleisle` succeeded. Embedded Go1.27.2/windows/amd64/CGO0 metadata reports that exact revision and `vcs.modified=false`.
- Candidate SHA256: `CC651CF18A9E531063ABD88210EEEB8A5E45307A00A68431B0E8AD82E22C3448`. Build metadata/manifest are alongside the EXE.
- Rechecked both earlier EXEs: e565dcd remains `F440CB97C7CB542B30099A2AE8C8D2A81E918FF192A2A8FBE3233AFFD96FC0DD`; ce4efcb remains `A8447152F95CF7F499A29A53A733DFCD5F952C34A8978D88EA5E2F24A66E6217`.
- This candidate was neither installed nor launched against user data. It is a local EXE build, not an installer or a certified complete runtime bundle. No remote publication, push, release, main merge, PR#2 merge or CI occurred.
## Windows follow-up: component acceptance ledger

This ledger supersedes page-level completion wording above. It maps the original components within the agreed main-workspace and fixed secondary-page scope to rules, implementation, regressions and evidence. A passing renderer or fake-transport test does not certify the installed desktop, original pixel parity, real IME, real Host or installation. Subcomponents are listed separately where they have different state or action ownership. This is not an inventory of every TSX implementation file or of uninstalled third-party contributions.

Reference prefixes: `P/` = original `vendor/deepseek-harness/packages/client/`; `V/` = original `vendor/`. Go `U/` = `internal/ui/`; tests with a bare name are in the corresponding implementation package unless a full path is given. Evidence `E/` = this task's `evidence/` outside the repository. Existing images remain bound to their recorded fixture source versions; the follow-up changes invalidate the old sidebar metadata/edit-hint images only. All PNG references below are software fixtures except explicitly labelled hidden native captures.

### Main workspace components

| Original component | Layout and state rule | Go implementation | Regression | Evidence / remaining acceptance |
|---|---|---|---|---|
| P/ui-layout columns + ConversationRoot.module.css | Conversation clamp680/64%/920; card adds32; preserve narrow concession | U/frame/frame.go, conversation_width.go | [TestConversationContentWidthAxis](../internal/ui/frame/frame_test.go#L44); [TestTranscriptWidthPreferenceAndNarrowConcession](../internal/ui/frame/conversation_width_test.go#L9) | E/frame-phase3/conversation-hero-400-resize0.png; same-state original still blocked |
| ConversationMainPanel / HeroShell | Unknown/loading history is not blank; Hero only after authoritative blank state | U/frame/conversation_phase.go; store/session follow lifecycle | [TestConversationPhaseSeparatesHistoryReadinessFromEmpty](../internal/ui/frame/conversation_phase_test.go#L19); [TestConversationSettlingAndFailureUseSharedStatus](../internal/ui/frame/conversation_phase_test.go#L106) | E/frame-phase3/conversation-settling.png, conversation-follow-error.png |
| ConversationSession title/ancestry | Breadcrumb and utilities share title band;400px retains every action | U/frame/conversation_header.go, session_ancestry.go | [TestConversationHeader400KeepsLongTitleAndEveryUtilityInside](../internal/ui/frame/conversation_header_test.go#L14); [TestConversationAncestrySharesTitleBandWithUtilities](../internal/ui/frame/conversation_phase_test.go#L143) | E/frame-phase3/conversation-header-400.png |
| InputBar input/card | Natural height,336px input cap; compact tools and primary-send slot share axis | U/composer/composer.go, draft_input.go | [TestComposerCurrentCardGeometry](../internal/ui/composer/presentation_test.go#L21); [TestShippedDraftClientGrowsAndClipsAtOriginal336pxCap](../internal/ui/composer/draft_presentation_test.go#L62); [TestComposerNarrowToolbarKeepsPrimaryAndActionsVisible](../internal/ui/composer/presentation_test.go#L131) | Existing composer/frame fixture matrix; no current real-window certification |
| InputBar tool/primary state | Empty running draft stops; nonempty sends; child-agent Stop remains independent | U/composer/tool_actions.go, composer.go | [TestRunningComposerPrimaryPreservesSendAndStopRoles](../internal/ui/composer/workspace_parity_test.go#L52); [TestComposerPrimaryEmptyFailedAndAttachmentOnlyAdmission](../internal/ui/composer/workspace_parity_test.go#L215) | Existing composer behavior regressions, not repeated |
| InputBar editRow | Float8px above card, right8, close22; draft bounds/focus do not move | U/composer/composer.go edit status overlay | NEW [TestComposerEditStatusFloatsWithoutMovingDraft](../internal/ui/composer/edit_geometry_test.go#L14) | E/windows-gap-audit/composer-edit-status-320.png and -800.png, inspected; corrected in4004010 |
| InputBar native input owner | IME preedit owns Enter/Escape; UTF16 selection and marked owner survive redraw/resize/attachments | third_party/mygo/internal/windows/surface.go, surface_input.go; U/composer/draft_client.go, draft_input.go | [TestNativeDraftInputRetainsCompositionAcrossFrames](../internal/ui/composer/draft_input_lifetime_test.go#L12); [TestBusyEnterNativeGestureIMEAndFocus](../internal/ui/composer/busy_enter_test.go#L10); [TestShippedDraftClientUTF16SelectionAndIMEGeometryAfterResize](../internal/ui/composer/draft_presentation_test.go#L88); [TestAttachmentTransitionsRetainNativeIMEOwner](../internal/ui/composer/workspace_parity_test.go#L193) | E/windows-followup-runtime/runtime-observation.json; source/previous tests only; real Chinese candidate window, commit and resize BLOCKED |
| Rows session line/menu | Height32; status slot16x20; hover/focus shows overflow menu with captured session | U/sidebar/sidebar.go, session_menu.go | [TestSessionRowsStayCompactAcrossSelectionAndMenu](../internal/ui/sidebar/native_rows_test.go#L12); [TestSessionMenusKeepRowIdentityAndCallbacks](../internal/ui/sidebar/native_rows_test.go#L50) | Existing sidebar tests; no permanent action strip |
| Rows time/PinnedIndicator | Nonblank rest shows original relative-time buckets and pinned glyph; hover/menu substitutes actions | U/sidebar/session_metadata.go, sidebar.go | NEW [TestSessionMetadataSourceTimeBuckets](../internal/ui/sidebar/session_metadata_test.go#L15); [TestSessionMetadataRestHoverMenuAndBlank](../internal/ui/sidebar/session_metadata_test.go#L36) | E/windows-gap-audit/sidebar-metadata-rest.png and -hover.png, inspected; exact original IconPinFillRegular vector in dcfb14c |
| WorkspaceBrowser groups | Workspace/session scroll shares Project rows; footer stays outside scroll | U/sidebar/workspace_groups.go, sidebar.go; U/projects/sidebar.go | [TestManyProjectsKeepSettingsFooterInsideSidebar](../internal/ui/sidebar/project_scroll_test.go#L12) | Existing40-Project fixture; compact rows no longer displace settings footer |
| MessageIconActions / FeedbackDialog |28px icons; history hover/focus; feedback draft and captured turn retained | U/chat/message_actions.go, feedback_dialog.go, feedback_editor.go | [TestAssistantTailOneCompactRowKeepsLifecycleTextAndBoundary](../internal/ui/chat/message_actions_test.go#L18); [TestEarlierUserActionsRevealOnHoverAndKeyboardWithoutReflow](../internal/ui/chat/message_actions_test.go#L129); [TestFeedbackModalSourceGeometryAndWorkflow](../internal/ui/chat/feedback_dialog_test.go#L23) | task-root/feedback-dialog-evidence and prior native workspace evidence; real feedback submission deliberately not exercised |
| Dock pane header/tab/menu |38px header;28px tabs; compact menus retain resource identity and close/maximize actions | U/frame/workspace_panes.go, workspace_dock_tabs.go | [TestWorkspacePaneChromeHeightAndNarrowActions](../internal/ui/frame/workspace_panes_chrome_test.go#L37); [TestWorkspacePaneMenuRetainsActionsAndResourceIdentity](../internal/ui/frame/workspace_panes_chrome_test.go#L63); [TestWorkspacePaneChromeMaximizeAndClose](../internal/ui/frame/workspace_panes_chrome_test.go#L94) | Prior source-backed pane fixtures; real multi-window drag remains uncertified |
| ui-theme font/platform defaults | Original Segoe UI/PingFang SC/Microsoft YaHei defaults, explicit overrides preserved | U/theme/preferences.go, theme.go | [TestEmptyPreferencesRestoreOriginalTextFallbacks](../internal/ui/theme/preferences_test.go#L9); [TestNormalizePreservesExplicitTextPreferences](../internal/ui/theme/preferences_test.go#L18); [TestApplyInterfaceUsesFallbackStackAndPreservesPalette](../internal/ui/theme/preferences_test.go#L26) | Native hidden captures show Windows text; identical original font/DPI/state comparison BLOCKED |

### Settings shell and component bodies

| Original component | Layout and state rule | Go implementation | Regression | Evidence / remaining acceptance |
|---|---|---|---|---|
| P/ui-settings-general SettingsRoot |800 shell,188 rail,54 body title; fit viewport and return focus | U/settings/modal.go, settings.go | [TestSettingsModalFitsViewport](../internal/ui/settings/modal_test.go#L37); [TestSettingsModalReturnsWorkspaceFocus](../internal/ui/settings/modal_test.go#L9) | E/phase3-composition/settings-market-{600,1000}.png; original same-state window still blocked |
| Registered section navigation |15 actual registrations, correct order/gates; dedicated route beats stale schema selection | U/settings/navigation.go, sections.go; cmd/whaleisle/settings_sections.go | [TestSettingsDedicatedSelectionWinsOverStaleSchemaSelection](../internal/ui/settings/navigation_test.go#L50); [TestSettingsUnsignedAccountStillHasLoginEntry](../internal/ui/settings/navigation_test.go#L62) | Existing navigation fixtures, current source inventory recorded above |
| P/ui-settings-general/src/client/GeneralSection.tsx | Compact preference rows; technical desktop/runtime controls disclosed | U/settings/desktop.go, body_helpers.go | [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(general/*); [TestSettingsPreferenceRowsReflowLongCopy](../internal/ui/settings/body_presentation_test.go#L166) | E/phase3-final/settings-body-general-{wide,narrow,loading,error,empty,read-only}.png |
| General custom instructions | Text draft belongs to Host; failed writes retain edits and accepted values replace acknowledged drafts | U/settings/custom_instructions.go, body_preferences.go; store/settings/body_preferences.go | [TestBodyTextDraftsStayWithHostAndAdoptAcknowledgedValues](../internal/ui/settings/body_presentation_test.go#L183); [TestBodyPreferenceWritesCrossNamespaceWithoutAcknowledgingDrafts](../internal/store/settings/body_preferences_test.go#L15) | Existing body/atomic-write fixtures; real Host save not attempted |
| General permission defaults | Only advertised enums; current Host ownership; no new permissions accepted | U/settings/permission_defaults.go; store/settings/body_preferences.go | [TestBodyPreferenceRejectsUnadvertisedEnumAndStaleHost](../internal/store/settings/body_preferences_test.go#L64) | Body state matrix; actual permission changes forbidden in this validation |
| P/ui-settings-general/src/client/InterfaceSection.tsx | Local language/tray/login and Host controls remain in settings body; narrow rows stack | U/settings/interface.go; cmd/whaleisle/settings_body_wiring.go | [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(interface/*); [TestSettingsPreferenceRowsReflowLongCopy](../internal/ui/settings/body_presentation_test.go#L166) | E/phase3-final/settings-body-interface-*.png; isolated mode skips login registration |
| Interface composer Beam editor | Structured source style/presets; paired fields one atomic mutation; failure retains dialog/draft | U/settings/effect_preferences.go, body_preferences.go; U/composer/settings_preview.go | [TestEffectEditorAtomicSaveRetainsFailureAndChangesHost](../internal/ui/settings/body_presentation_test.go#L247); [TestEffectConfigurationRoundTripAndSourceConstraints](../internal/ui/settings/body_presentation_test.go#L317); [TestBodyPreferenceCoupledEffectsAreOneValidatedMutation](../internal/store/settings/body_preferences_test.go#L95) | E/phase3-final/settings-effect-composerBeam-failed-save.png; native preview fixture only |
| Interface typing effect editor | Structured color/effect/caret presets; original style bounds; Host replacement closes old editor | Same effect editor and preview modules | Same effect tests with typingFx subcase | E/phase3-final/settings-effect-typingFx-failed-save.png; real Host apply not attempted |
| P/ui-theme/src/client/AppearanceSection.tsx theme/font | Dedicated body, font/scheme overrides, read-only/error states | U/settings/appearance.go, theme_font_preferences.go; U/theme | [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(appearance/*); [TestNormalizePreservesExplicitTextPreferences](../internal/ui/theme/preferences_test.go#L18) | E/phase3-final/settings-body-appearance-*.png |
| Appearance wallpaper/pointer settings | Dedicated disclosure/gallery and editable local preferences | U/settings/wallpaper_background.go, pointer_effects.go | [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(appearance/*) | Body layout covered; exact original fluid renderer is NOT certified: current UI explicitly describes a native approximation |
| P/ui-settings-account/src/client/AccountSection.tsx | Account/profile/balance cards; loading/error/no-permission distinct; no credential use in fixtures | U/settings/account.go, account_signout.go | [TestFeatureBodySoftwareSnapshots](../internal/ui/settings/feature_body_presentation_test.go#L143)(account/*); [TestAccountAccessFieldsPresent](../internal/ui/settings/account_test.go#L41) | E/phase3-final/settings-account-*.png; E/native-phase3-working/native-settings-account.png is hidden native synthetic data |
| P/ui-settings-models/src/client/ModelsSection.tsx roster | Compact provider cards; active row owns its action/menu | U/settings/models.go, provider_profiles.go | [TestFeatureBodySoftwareSnapshots](../internal/ui/settings/feature_body_presentation_test.go#L143)(models/*) | E/phase3-final/settings-models-*.png; real discovery/account use not performed |
| Models structured configuration | Open source-defined provider editors; bind namespace+Host and retain form drafts | U/settings/model_body.go, model_configuration.go; store/settings/plugin_body.go; cmd/whaleisle/settings_body_wiring.go | [TestFeatureModelsKeepStructuredEditorsAndBoundDrafts](../internal/ui/settings/feature_body_presentation_test.go#L52); [TestPluginNamespaceBodySavesBoundNamespaceWithoutSelectingRail](../internal/store/settings/plugin_body_test.go#L20); [TestPluginNamespaceOldCallbacksCannotWriteNewHost](../internal/store/settings/plugin_body_test.go#L78) | E/phase3-final/settings-models-structured-{360,780}.png; no arbitrary diagnostic form replacing known editors |
| P/ui-settings-plugins/src/client/PluginsSettingsSection.tsx | Installed profile settings remain separate from curated Market and separate manager | U/settings/plugins.go, plugin_body.go | [TestPluginNamespaceBodySavesBoundNamespaceWithoutSelectingRail](../internal/store/settings/plugin_body_test.go#L20); [TestSettingsFeatureSectionKeepsShellAndGeneralControls](../internal/ui/settings/navigation_test.go#L11) | Native Plugins SETTINGS capture E/native-phase3-working/plugins-fresh/native-settings-plugins.png; not the independent manager |
| P/ui-settings-plugin-inventory/src/client/PluginInventorySettingsTab.tsx | Reflow cards, Loader IDs/status, per-entry expansion and retry | U/settings/plugin_inventory.go | [TestPluginInventoryCardsReflowAndExpandTheirOwnEntry](../internal/ui/settings/plugin_inventory_presentation_test.go#L13); [TestSP43ClientModulesStatusAndRetry](../internal/ui/settings/plugin_inventory_test.go#L11) | E/phase3-final/plugin-inventory-{wide,narrow,expanded}.png |
| P/ui-settings-skills/src/client/SkillsSection.tsx | Compact controls/groups; immutable bundled rows and captured skill identity | U/settings/skills.go | [TestFeatureCompactSkillControlsAndGroups](../internal/ui/settings/feature_body_presentation_test.go#L86); [TestFeatureBodySoftwareSnapshots](../internal/ui/settings/feature_body_presentation_test.go#L143)(skills/*) | E/phase3-final/settings-skills-*.png; actual skill installation/delete not exercised |
| P/ui-settings-mcp/src/client/McpSection.tsx | Compact server rows, editor/import/catalog and authorization gate | U/settings/mcp.go, mcp_form.go, mcp_import.go, mcp_catalog.go | [TestFeatureMCPCompactControlsAndAuthorizationGate](../internal/ui/settings/feature_body_presentation_test.go#L116); [TestMCPRowDetailParts](../internal/ui/settings/mcp_test.go#L24) | E/phase3-final/settings-mcp-*.png; actual server authorization/test not exercised |
| P/ui-agent-preset/src/client/AgentPresetSection.tsx | Adaptive equal-width cards/min268; low-alpha selection; independent guide reader | U/settings/agent_presets.go, agent_preset_guide.go | [TestAgentPresetSettingsParity](../internal/ui/settings/agent_preset_parity_test.go#L13); [TestFeatureBodySoftwareSnapshots](../internal/ui/settings/feature_body_presentation_test.go#L143)(agent-presets/*) | E/phase3-final/settings-agent-presets-data-{360,780}.png; inspected after alpha fix |
| P/ui-settings-market/src/client/MarketSection.tsx catalog | Separate curated tabs and responsive cards, embedded in same settings shell | U/market/presentation.go; cmd/whaleisle/settings_sections.go | [TestMarketTabsRetainStoreAndEmbeddedRefresh](../internal/ui/market/presentation_test.go#L126); [TestSettingsMarketUsesModalShellAndReturnsToGeneral](../cmd/whaleisle/settings_sections_test.go#L87) | E/phase3-final/market-cards-{wide,narrow}.png; E/phase3-composition/settings-market-*.png |
| Market details/install confirmation | Nested dialog retains selected item and explicit install options | U/market/presentation.go, management.go | [TestMarketConfirmationIsModalAndEscapeCancels](../internal/ui/market/presentation_test.go#L42); [TestMarketDetailsSnapshotFixture](../internal/ui/market/presentation_test.go#L77) | E/phase3-final/market-details-*.png, market-confirmation.png; README now uses native document layout, inline links and cached absolute media; see the source-gap batch below for regressions and remaining format/cascade limits |
| P/ui-settings-remote/src/client/{RemoteSection,RemoteSettingsSection,GatewaySettingsTab}.tsx | Gateway and machine/tool groups stay in settings modal; v2 pairing differs from legacy token | U/remote/settings_body.go, gateway.go, machines.go, tools.go | [TestRemoteSettingsGroupingAndDraftOwnership](../internal/ui/remote/settings_body_test.go#L21); cmd [TestSettingsEmbeddedPagesStayInModalAcrossWidths](../cmd/whaleisle/settings_sections_test.go#L34) | E/phase3-final/settings-remote-*.png; E/phase3-composition/settings-composition-remote-*.png; no pairing/network/security change |
| Remote Channels configuration | Embedded platform configuration only; no transcript substituted for settings | U/im/settings.go; cmd/whaleisle/settings_sections.go | [TestEmbeddedChannelsConfiguration](../internal/ui/im/settings_test.go#L10) | Existing Channels fixture; actual service connection forbidden/not performed |
| V/dsh-remote/plugin-src/client/RemoteSettingsSection.jsx | Separate remote-workspace registered body and captured target | U/remote/settings_body.go; cmd/whaleisle/settings_sections.go | [TestRemoteSettingsGroupingAndDraftOwnership](../internal/ui/remote/settings_body_test.go#L21); [TestSettingsEmbeddedPagesStayInModalAcrossWidths](../cmd/whaleisle/settings_sections_test.go#L34) | E/phase3-final/settings-workspace-*.png; E/phase3-composition/settings-composition-remote-workspace-*.png |
| V/dsh-usage-panel/src/client/StatsSection.tsx | Statistics-only embedded body, account settings excluded, original dimension grouping | U/usage/embedded.go, stats.go, statistics_dimensions.go, time_range.go | [TestEmbeddedUsageHasOnlyStatistics](../internal/ui/usage/embedded_test.go#L81); [TestUsageBodySoftwareSnapshots](../internal/ui/usage/embedded_test.go#L96) | E/phase3-final/settings-usage-*.png; E/phase3-composition/settings-composition-usage-stats-*.png |
| P/ui-settings-general/src/client/PetSection.tsx | Dedicated preferences/behavior body; busy/error/read-only distinct | U/settings/pet.go, pet_look.go | [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(pet/*) | E/phase3-final/settings-body-pet-*.png; live desktop pet behavior not certified |
| Pet Whale settings contribution | Reuse Host/generation-owned editor without session creation or model request | U/assistant/whale_settings_section.go; cmd/assistant_whale_settings.go | [TestWhaleSettingsSectionSourceRowsAndHostDialogIsolation](../internal/ui/assistant/whale_settings_section_test.go#L15); [TestWhaleSettingsNativeFormRequiresExplicitClear](../internal/ui/assistant/whale_settings_test.go#L10) | Existing inert settings fixtures; real Host apply blocked by missing runtime |
| P/ui-settings-general/src/client/AboutSection.tsx | Compact version/home/update body, long path containment, storage warnings | U/settings/about.go | [TestAboutStorageWarningsAndLinks](../internal/ui/settings/about_test.go#L10); [TestSettingsBodySourcePagesAndStates](../internal/ui/settings/body_presentation_test.go#L85)(about/*) | E/phase3-final/settings-body-about-*.png; no update/download initiated |

### Jobs, directory browser, Projects and independent Plugins

| Original component | Layout and state rule | Go implementation | Regression | Evidence / remaining acceptance |
|---|---|---|---|---|
| P/ui-jobs/src/client/JobListAction.tsx | Roster observes while mounted;500/max480 anchored popover; closing output does not dispose roster | internal/jobflow/controller.go; store/session/jobs.go; U/frame/jobs.go | [TestMountedRosterOutlivesPopupAndRemountFencesOldStream](../internal/jobflow/mounted_test.go#L11); [TestJobsHeaderPopoverRosterOutputStopAndFinished](../internal/ui/frame/header_jobs_test.go#L118) | E/phase3-final/jobs-{live,narrow,output}.png; real job kill was not requested |
| Jobs output/error/finished controls | Retain output cursor; retry failure; captured pending kill not repeated when popup closes | internal/jobflow; U/frame/jobs.go | [TestJobsReadErrorRetryLoadingAndEmptyDisposal](../internal/ui/frame/header_jobs_test.go#L254); [TestPendingKillSurvivesPopupCloseWithoutDuplicate](../internal/jobflow/mounted_test.go#L74) | E/phase3-final/jobs-{loading,error,empty,output-error}.png; fake transports only |
| P/ui-schedule ScheduleCatalogAction |28px header trigger;336 anchored menu; zero/one/many schedules and owning session | U/frame/conversation_header.go; U/chat/schedule_context.go | [TestScheduleHeaderZeroOneManyOwnerAndPopoverBounds](../internal/ui/frame/header_schedule_test.go#L23); [TestScheduleHeaderDeletionUsesCapturedOwningSession](../internal/ui/frame/header_schedule_test.go#L114) | Existing header snapshots in E/phase3-inspect; real schedule delete not exercised |
| P/ui-schedule TaskManagerPage / TaskDetail |960 split catalog,47% detail; narrow nested detail; lazy Records and retained draft | U/schedule/presentation.go | [TestScheduleCatalogDetailGeometryAndLazyRecords](../internal/ui/schedule/presentation_test.go#L168); [TestScheduleDraftRetainedAndEndedRuleReadable](../internal/ui/schedule/presentation_test.go#L227); [TestScheduleSaveAndDeleteKeepCapturedOwner](../internal/ui/schedule/presentation_test.go#L330) | E/secondary/schedule-{catalog,detail,narrow-detail,records,delete}.png |
| P/ui-workspace WorkspacePicker | Hero anchor, scroll list with pinned add/no-folder actions; rename uses workspace identity | U/workspacepicker/picker.go; cmd/workspace_navigation.go | [TestPickerAnchorsAndUsesWorkspaceIdentity](../internal/ui/workspacepicker/picker_test.go#L17); [TestPickerLongListKeepsActionsPinned](../internal/ui/workspacepicker/picker_test.go#L58); [TestRenameFocusAndBusyDismissal](../internal/ui/workspacepicker/picker_test.go#L87) | E/secondary/workspace-picker-long-list.png, workspace-rename.png |
| P/ui-directory-picker-browse/src/client/DirectoryBrowser.tsx |680x500 shell; Miller columns, pinned footer, hidden-directory filter | U/dirpick/panel.go, state.go | [TestMillerPaneGeometryAndPinnedFooter](../internal/ui/dirpick/panel_test.go#L26); [TestHiddenToggleSelectionAndPrefixFiltering](../internal/ui/dirpick/panel_test.go#L74) | E/phase3-final/directory-picker-miller-{short,wide}.png |
| Directory path editor/preview | Typed path preview waits for correct legs; leaving focus/Escape cancels; preserve Host spelling/raw whitespace | U/dirpick/state.go; cmd/directory_picker.go | [TestPathEditorFocusLeavingCardCancelsDraft](../internal/ui/dirpick/panel_test.go#L250); [TestDirectoryBrowserTypedPreviewWaitsBothLegsAndCancelKeepsLanding](../cmd/whaleisle/directory_picker_test.go#L464); [TestDirectoryBrowserEnteredPathPreservesWhitespaceAndErrorsStaySilentForPreview](../cmd/whaleisle/directory_picker_test.go#L576) | Existing pure/fake directory regressions; actual Host filesystem reads blocked |
| Directory nested create/error/retry |380 nested dialog, parent inert, raw name passed exactly; failed path retry and late-close fences | U/dirpick/panel.go; cmd/directory_picker.go | [TestNestedCreateRawNameAndParentInertness](../internal/ui/dirpick/panel_test.go#L126); [TestDirectoryBrowserCreateFailureRetryRawNameAndLanding](../cmd/whaleisle/directory_picker_test.go#L531); [TestDirectoryBrowserRenderMergePreservesSameRequestCommit](../cmd/whaleisle/directory_picker_test.go#L681) | E/phase3-final/directory-picker-{create,empty}.png; no user directory was created |
| V/dsh-project/client/client.js ProjectRow / SidebarPage |36px rows; hover/focus menu; Projects inside shared sidebar scroll | U/projects/sidebar.go; U/sidebar/sidebar.go | [TestCompactProjectRowsKeepMenusAndDirectoryIdentity](../internal/ui/projects/presentation_test.go#L216); [TestManyProjectsKeepSettingsFooterInsideSidebar](../internal/ui/sidebar/project_scroll_test.go#L12) | Existing project/sidebar fixtures; actual runtime lacks staged Project module |
| ProjectHeader and modal routes | Content-sized modal capped760/80vh, separate scrolling body and pinned footer; route restores origin focus | U/projects/modal.go, view.go | [TestProjectPagesUseBoundedModalStateMatrix](../internal/ui/projects/presentation_test.go#L29); [TestProjectNestedRouteReturnsToScrollAndOriginFocus](../internal/ui/projects/presentation_test.go#L362) | E/phase3-final/projects-*-{wide,narrow,narrow-dark}.png |
| ProjectTaskCard strip/card/full | Retain each request mode/tab, completion reset, progress/delivery summaries; full details max420 | U/projects/task_cards.go, observer.go | [TestObserverTaskCardsRetainModesAndRouteActions](../internal/ui/projects/task_cards_test.go#L44); [TestObserverTaskCardWideNarrowStateMatrix](../internal/ui/projects/task_cards_test.go#L141) | E/phase3-final/projects-observer-{strip,card,full}-*.png; corrected compact footer inspected |
| Project request queue/menu | Revision and before-request IDs come from displayed projection; mutation captures Host/route/request owner atomically | internal/projectflow/requests.go; cmd/project_request_queue.go, project_requests.go | [TestRequestQueueUsesSourceProjectionAndRevisionContract](../internal/projectflow/request_queue_test.go#L16); [TestProjectQueueUsesDisplayedRevisionAndBeforeIdentity](../cmd/whaleisle/project_request_queue_test.go#L9); [TestProjectQueueSubmissionKeepsRenderedHostAndRoute](../cmd/whaleisle/project_request_queue_test.go#L42) | Fake-transport tests; actual queue mutation not performed |
| ProjectTaskWork first-entry detail | Read current request/team/work without a prior Team visit; stale cached cards cannot execute current-card actions | cmd/project_request_detail.go; U/projects/task_cards.go | [TestRequestWorkDetailLoadsTeamOnFirstTaskEntryWithoutMutations](../cmd/whaleisle/project_request_detail_test.go#L17); [TestObserverCardAssetsAndWorkRequireMatchingControllerContext](../internal/ui/projects/task_cards_test.go#L251) | Existing task-card fixtures; current real Host unavailable |
| Project materials: notes/preferences/charter | Bound file paths/16KB rule file; retain failed edits; archived project is read-only | internal/projectflow/materials.go; U/projects/materials.go, material_collection.go | [TestProjectMaterialEditing](../internal/projectflow/materials_test.go#L12); [TestProjectMaterialTransitionKeepsDraftOnFailure](../internal/ui/projects/presentation_test.go#L258) | E/phase3-final/projects-{notes,preferences,rules,material-read-only,material-save-error}*.png |
| Project documents and deliveries | Separate loading/empty/read-error/detail routes, exact captured file references | U/projects/documents.go, deliveries.go, assets.go | [TestProjectPagesUseBoundedModalStateMatrix](../internal/ui/projects/presentation_test.go#L29); cmd [TestSidebarProjectDirectoryNeedsKnownProjectWithoutDeliveryCard](../cmd/whaleisle/project_request_detail_test.go#L55) | E/phase3-final/projects-{documents-empty,document-text,document-error,deliveries-empty,delivery-detail}-*.png |
| Project reports/history/process/activity/diff | Recorded report and process are separate routes; current revision gates activity/work diff | U/projects/history.go, historical_process.go, activity.go, detail.go | [TestProjectPagesUseBoundedModalStateMatrix](../internal/ui/projects/presentation_test.go#L29); [TestWorkDetailDiffEntryAndReturn](../internal/ui/projects/detail_diff_test.go#L10); [TestObserverFullCardsFenceNoArgumentActionsAndSerializeReads](../internal/ui/projects/task_cards_test.go#L202) | E/phase3-final/projects-{report-detail,report-error,history,process,activity,diff}-*.png; real files not opened |
| Project rename/archive lifecycle | Captured project identity; busy guards and dirty editor confirmation | U/projects/lifecycle.go; internal/projectflow | [TestProjectLifecycleDialogsProtectTargetAndBusyState](../internal/ui/projects/presentation_test.go#L308); [TestProjectLifecycleIsNotRequestArchive](../internal/projectflow/lifecycle_test.go#L11) | E/phase3-final/projects-{rename,archive}-narrow.png; user projects untouched |
| P/ui-plugin-manager installed profile manager | Independent page, profile/bundle/plugin identity, bounded cards; no curated fallback on missing capability | internal/pluginmanager; U/pluginmanager/view.go; cmd/native_plugin_manager.go | [TestManagerNativeNavigationConfigurationsAndNarrowLayout](../internal/ui/pluginmanager/view_test.go#L92); [TestManagerDisconnectedPageNeverShowsCuratedCatalog](../internal/ui/pluginmanager/view_test.go#L190); [TestManagerJoinsLoaderIdentityAndKeepsCardsOnReadFailure](../internal/pluginmanager/controller_test.go#L98) | E/phase3-final/plugins-{installed-wide,installed-narrow,disconnected}.png; not the native Plugins settings capture |
| Generic Plugin Market Discover/detail/sources | Named typed original endpoint, source-aware query/pages; unavailable capability remains explicit | internal/pluginmanager/discover.go; U/pluginmanager/dialogs.go | [TestManagerDiscoverDetailsAndSourcesStaySeparate](../internal/ui/pluginmanager/view_test.go#L156); [TestDiscoverUsesNamedQueryAndRejectsMixedIndexPages](../internal/pluginmanager/controller_test.go#L277); [TestDiscoverMissingEndpointStaysDistinctFromCuratedMarket](../internal/pluginmanager/controller_test.go#L316) | E/phase3-final/plugins-{discover-wide,discover-detail,discover-sources-narrow}.png; staged endpoint missing, real read-only call BLOCKED |
| Plugin install terminal/progress/cancel | Captured install/Host IDs; bounded runs/streams/argv/cwd/exit; terminal events cannot revive operation | internal/pluginmanager/install.go; cmd/native_plugin_manager.go | [TestNativePluginEventsFollowCapturedInstallAndHost](../cmd/whaleisle/native_plugin_manager_test.go#L18); [TestInstallProgressCannotReviveTerminalOrReopenCancellation](../internal/pluginmanager/events_test.go#L19); [TestInstallTerminalShowsRunMetadataInNarrowDialog](../internal/ui/pluginmanager/view_test.go#L70) | E/phase3-final/plugins-install-terminal-narrow.png; no real plugin installation or cancellation |

### Windows startup and installation chain

| Original component / condition | Rule | Go implementation | Regression / actual check | Evidence / result |
|---|---|---|---|---|
| Separate acceptance instance | Temporary files must not claim installed protocol/login entries, lock, SDK cache or shared acceptance hook | NEW cmd/isolated_desktop.go; three integration callsites and startup in main.go | NEW four TestIsolatedDesktop* tests PASS2.865s |4004010 then final dcfb14c candidate, explicit WHALEISLE_ISOLATED=1 and absolute temporary WHALEISLE_USER_DATA |
| Isolated native process startup | Hidden process survives initial load; candidate path and temporary config hash match; stop only owned PID | Actual Windows candidate + fresh inert configuration and empty vendor |4004010 PID41892 alive4s,stderr empty,config unchanged,own PID absent after stop | E/windows-followup-runtime/run-20261019-140153/observation.json; this does not certify UI/IME/Host |
| Native original/candidate screenshots | Same viewport/DPI/font/data state, read-only capture | Current CUA capability check | apps=[]; cua.listWindows is not a function; node_repl absent | E/windows-followup-tools/native-window-capability.json; BLOCKED, no alternate screenshot/OS helper attempted |
| Host read-only runtime | Exact staged build and original Project/plugin-market capabilities required; do not substitute user Host | paths.VendorDir and current Host artifacts | No selected runtime near candidate/checkout; old vendor lacks manifest and required modules | E/windows-followup-runtime/runtime-observation.json; BLOCKED, no real Host request made |
| Original runtime extraResources | Ship complete matching runtime, preview helper and resource provenance | scripts/package.ps1:79-133; scripts/nsis/whaleisle.nsi:36-80 | Existing package-release-inputs-fixture.mjs and component-samples-fixture.mjs not repeated | E/windows-followup-package/package-conditions.json; missing NSIS/runtime/same-version helper; no package build attempted |
| Original installer welcome/license/finish/languages | Original package.json177-194 + build/installer.nsh brand pages and Chinese/English flow | scripts/nsis/whaleisle.nsi plus branding/ now restore source page order and resources | scripts/installer-branding-fixture.mjs source/resource regression | Source implementation and static contract PASS; NSIS compiler/installation still unavailable |
| Same appId cover upgrade/uninstall | Maintain installation identity, preserve user data, inspect shortcuts/registry and uninstall behavior | scripts/nsis/whaleisle.nsi16,23-25,84-114; importer | Existing [TestCoverUpgradePathInvariants](../internal/desktop/importer/upgrade_path_test.go#L14), static only | TEMP /D installation still writes same HKCU/app shortcuts; prohibited on this account; requires already-available Windows Sandbox/VM |

Follow-up source fixes are limited to the two source-proven UI omissions and safe isolated startup. Prior22-package results are retained rather than rerun wholesale. New affected gates: composer2.836s, sidebar final vector2.453s; four isolated startup tests2.865s. Source edits, fixture checks and actual process checks are separate entries. Full visual parity, real IME, full Host capabilities and installation remain open; source-known README rich rendering, fluid-effect fidelity and installer pages are not hidden by page-level completion labels.
### Final Windows follow-up disposition

- PASS: product changes in4004010 and exact pinned vector in dcfb14c; affected composer/sidebar packages and four new isolated tests passed as listed above. The earlier22-package gate was not rerun wholesale.
- PASS: final `whaleisle-native-windows-dcfb14c.exe` is Go1.27.2/windows/amd64/CGO0, embeds source `dcfb14c32d3b1bfdcf4b07d5533c023261ab9e26`, `vcs.modified=false`. SHA256 `2C448A7810A7A5D9D82DAEAD4F923E73343A93B285EE3D9438CC34E517EFF12C`. All previous candidate files remain separate.
- PASS: final artifact isolated hidden startup PID19200 survived4 seconds; exact process path matched,stderr0 bytes, temporary config hash remained `3592644D1A152FF52B76C7FDAD2AA2C61DA6B93834F90AEE0D839828094229A4`; only own PID stopped and confirmed absent. Evidence `E/windows-followup-runtime/final-20261019-140433/observation.json`. Empty workspace/empty vendor and autoStartDesktop=false requested no Host startup. This pass covers startup/cleanup, not UI rendering or Host behavior.
- SOURCE GAPS at dcfb14c / not passed then: original installer brand/welcome/license/finish/language flow; curated Market README rich Markdown/media; exact fluid-effect renderer fidelity. The source-gap batch below subsequently implements the first two with explicit remaining limits; fluid fidelity stays open. No broad original-equivalence sign-off is made.
- TOOL BLOCKED: native original/candidate screenshot and actual Chinese IME controls. CUA has no native apps or Windows window method; node_repl is not exposed. No screenshot helper, injection API or permission workaround was attempted.
- RUNTIME BLOCKED: genuine Host read-only Project/Plugins/directory operations need matching staged Windows runtime. None is selected beside this candidate/checkout; old vendor lacks current manifest/modules. No call was made against the user's running Host and no model request was sent.
- INSTALLATION BLOCKED: no local NSIS, complete runtime or matching preview helper. Package scripts would rebuild/remove staging; they were not executed. TEMP installation would still alter this account's original appId registry and shortcuts, so installation/uninstallation was not attempted.
- DENIED, not retried: default-sandbox narrow CIM child-process query for the first isolated PID41892 lacked access; final candidate used its owned process object and no CIM. This does not certify absence of all possible child processes. Original `git -C` metadata access was also denied; the source worker's already available ordinary file reads supplied the component references without requesting elevation. No auto-review escalation or new permission was requested.

Minimal remaining manual/prerequisite checklist:

1. In an already available Windows Sandbox/VM or another approved isolated Windows account, provide the matching verified runtime and preview helper plus existing NSIS for offline staging/content inspection. Do not run TEMP smoke installation on the active user account. No tool installation is requested by this task.
2. When native capture/input is available, compare original and final candidate at the same client size,DPI,font and synthetic page state. Check idle/selected/hover/menu sidebar, empty/loading/error/history conversation, edit hint and narrow composer first.
3. In that isolated candidate, test Chinese preedit Enter (must not send), Escape (candidate cancellation), candidate selection/commit, and candidate-window caret placement after input resize. Do not submit a model prompt.
4. With a matching isolated Host, perform only read operations for profile inventory/Discover, Project listing/detail and directory browse. Real plugin installs, job termination, queue writes and other mutations remain outside this validation.

The30-minute check was already scheduled by the parent conversation. This task creates no duplicate automation, and no CI/push/merge/release/system installation occurs.
## Source-gap batch after88838f0

This batch continues the same isolated branch. It resolves the two implementable source gaps from the65-row ledger: curated Market rich README and installer brand/language flow. Fluid fidelity is separately assessed without changing its current approximation or calling it equivalent. The existing source and Windows startup evidence for dcfb14c remains historical; this batch does not repeat the22-package gate, native capability probe, rejected CIM query or installed-app startup test.

### README component chain

| Original component and layout/state rule | Go implementation | Regression and evidence |
|---|---|---|
| MarketExtras.tsx MarketDetails: nonempty README mounts MarkdownText; no document section for empty content | U/market/readme.go and presentation.go; same bounded native modal, no browser replacement | [TestMarketReadmeNativeDocumentAndExplicitInlineLink](../internal/ui/market/readme_test.go#L34); [TestMarketReadmeEmptyDoesNotMountSection](../internal/ui/market/readme_test.go#L70); [TestMarketRichReadmeDetailsWideAndNarrow](../internal/ui/market/readme_test.go#L226) |
| MarkdownAnchor: inline explicit HTTP(S)/mailto navigation; no relative/fragment/base-URL guessing; markup is not executable HTML | U/widgets/markdown Style.OnOpenURL and LiteralHTML; main.go binds system opener only on user activation | [TestDocumentInlineURLsKeepTextFlowAndRequireActivation](../internal/ui/widgets/markdown/document_test.go#L22); [TestDocumentLiteralHTMLAndFormattedLinkContents](../internal/ui/widgets/markdown/document_test.go#L60); [TestDocumentURLTargetsKeepMathSpanOffsets](../internal/ui/widgets/markdown/document_test.go#L105) |
| Original linked image preserves enclosing anchor destination, including empty alt and repeated image URLs | Native image link box derives its destination from AST ancestry and isolates each occurrence's retained state; formulas keep link metadata through SVG and fallback | [TestDocumentLinkedImageKeepsItsOriginalInlineDestination](../internal/ui/widgets/markdown/document_test.go#L123); [TestDocumentEmptyAltImageCannotLoseOrBorrowEnclosingLink](../internal/ui/widgets/markdown/document_test.go#L187); [TestDocumentRepeatedImageURLsRetainSeparateOccurrences](../internal/ui/widgets/markdown/document_test.go#L218); [TestDocumentFormulaURLsSurviveRenderedAndPendingMath](../internal/ui/widgets/markdown/document_test.go#L163) |
| MarkdownText typography: content-size delta; h1/h2/h3=21/19/18 and30/28/26 lines, weights700; h4-6/body=14/24 with heading/strong600 | Opt-in U/widgets/markdown/document_layout.go; default legacy Chat paths are unchanged | [TestMarketReadmeSourceHeadingParagraphAndQuoteGeometry](../internal/ui/market/readme_test.go#L79) at14/18px; targeted legacy Chat and full Files package below |
| Block margins:32/16 headings,16px paragraphs, adjacent CSS margins collapse; h4-6 before list8; first/last child outer margin0 | Document walker calculates max adjacent margins and suppresses outer margins | [TestDocumentLayoutResetsOuterMarginsAndCollapsesInteriorMargins](../internal/ui/widgets/markdown/document_test.go#L149); same market geometry regression |
| Lists:18px content indent,6px item gap,8px loose paragraphs,4px nested-list top; horizontal rule32px margins and0.5px height | Native document list and rule geometry, active theme Border token | [TestMarketReadmeSourceListAndRuleGeometry](../internal/ui/market/readme_test.go#L113) |
| Blockquote left border2 caption + left padding14; inherit normal body color/style instead of muted italics | Document quote uses current active theme Caption and body style | Market heading/paragraph/quote geometry; source caption light#ADB2B8/dark#81858C |
| MarkdownImage: only absolute HTTP(S) remote images; natural size/max100%, aspect ratio,12px radius; relative/file/data retain authored tertiary italic alt | U/market/readme_images.go shares existing imgcache.Cache, native bitmap/basic SVG; RenderImage hook retains document order including table cells | [TestMarketReadmeRemoteImageLoadsInPlaceWithoutNavigation](../internal/ui/market/readme_images_test.go#L65); [TestMarketReadmeCachedImagesKeepNaturalSizeAndAspectRatio](../internal/ui/market/readme_images_test.go#L122); [TestMarketReadmeImageFailureAndUnsupportedDestinationsRetainAlt](../internal/ui/market/readme_images_test.go#L155); [TestDocumentImagesStayInOrderIncludingTableCells](../internal/ui/widgets/markdown/document_test.go#L77) |

Original source references were ordinary read-only files under `vendor/deepseek-harness/packages/client/`: ui-settings-market/src/client/MarketExtras.tsx and MarketSection.module.css; ui-primitives/src/markdown/render.tsx and MarkdownText.module.css; ui-theme/src/styles/gradient-shadow-text.css66-106, base.css28 and design-platform.css. The backend supplies raw README without a base URL, so no guessed branch/URL field or new request endpoint was added.

Inspected software evidence in `E/market-readme-native/`: market-rich-readme-wide.png, market-rich-readme-narrow.png, market-readme-image-loaded.png and market-readme-image-failed.png. The final package gate regenerated these after outer-margin, alt-style and image-radius corrections. All remote-image tests use an in-memory HTTP RoundTripper and temporary cache; no public network, model or user Host was used.

README limits at519d31b included the unresolved Market h3 cascade; the next batch below resolves its source loading order. Native basic SVG still lacks browser-complete text/filter/embedded-image behavior. Media audit below corrects the earlier blanket first-frame statement: GIF returns its first frame, whereas standard animated WebP fails decoding and retains alt. These are explicit limits, not full browser or original-window pixel parity.

### Installer component chain

| Original component and rule | Go/NSIS implementation | Regression and evidence |
|---|---|---|
| package.json build.nsis + build/installer.nsh: original right header, install/uninstall side image and ICO | scripts/nsis/whaleisle.nsi MUI2 branding macros; scripts/nsis/branding/ exact original resource bytes | scripts/installer-branding-fixture.mjs checks SHA256,24-bit BMP150x57/164x314 and ICO; source-assets.json records each observed byte hash |
| Assisted install pages: welcome, license, directory, install progress, finish; original long titles, project link and Run checkbox | MUI_PAGE_WELCOME/LICENSE/DIRECTORY/INSTFILES/FINISH with original MIT license | Same Node fixture checks ordering, per-page macros and exact license/resource provenance |
| Uninstall welcome, progress, finish; source three-line welcome title | MUI_UNPAGE_WELCOME/INSTFILES/FINISH | Same fixture verifies separate uninstall welcome macro application |
| zh_CN then en_US; original does not add language selection step | MUI_LANGUAGE SimpChinese/English, Unicode; no extra language dialog | Same fixture checks language order and absence of inserted selector |

`node scripts/installer-branding-fixture.mjs` passed. Evidence `E/windows-followup-package/installer-branding-source-check.json` is explicitly PASS_SOURCE_AND_RESOURCE_CONTRACT_ONLY. Original Git metadata access remained denied and was not retried; resource provenance therefore claims observed file hashes and source rules, not a freshly verified original commit. No package script, makensis, installer or uninstaller ran. Missing NSIS, matching runtime/helper and safe isolated Windows installation environment still block actual compilation, page rendering, upgrade and uninstall acceptance. Original same-appId registry/shortcut effects remain unchanged; TEMP installation is still prohibited on the active account.

### Fluid and related animation assessment

`E/windows-fluid-assessment.json` records four separate original components at88838f0, their Go implementation, equations/state rules and missing evidence. This historical assessment precedes the implemented solver/timeline batch below.

| Original component | Already represented | Specific remaining difference / feasible path |
|---|---|---|
| cursor-fx.ts pixel trail |24px base cells, cap4000, lifetime/spacing, reduced motion and inert overlay | Source random palette vs deterministic cycling; retune/resize preservation and fractional cell rules differ; dedicated numerical/state regressions not yet present |
| cursor-fluid.ts splash solver | Pointer dye/momentum, palette/size/speed inputs, bilinear advection and premultiplied image | Original aspect-correct128 simulation and1440 dye fields, curl/vorticity3/divergence, pressure decay.1 and20 Jacobi passes, gradient subtraction are missing from current96x64 approximation. Splat force6000*speed/100 and click10x, rational dissipation,16.666ms cap, shading/alpha=maxRGB,4s idle and resize preservation also differ |
| TypingFxLayer / typing-fx.ts | Insert owner/IME commit and echo limits | Source drop35% and flash20% alpha1 differ from Go.65/.8; piecewise CSS easing and relative caret clock not exact. These are directly implementable timing changes, not native-tool blockers |
| ComposerBeam | State/direction/mask/night controls | Four radial lights, separate bloom, white highlight,420ms transition and distinct CSS easing are not fully reproduced |

Native UI is not intrinsically incapable of the source fluid. A strict CPU equation port can feed Painter.Image, but source1440-short-edge dye performance has not been measured. A performant source-sized GPU port needs persistent half-float ping-pong textures and multiple simulation passes inside vendored MyGo/D3D11; current Painter.Effect exposes a single fragment/backdrop and20 scalar parameters rather than that stateful contract. Existing shader/blur infrastructure offers an implementation route. This is concrete unimplemented renderer/solver work, not a reason to label the current approximation exact or to defer the directly calculable timing/color fixes to screenshot availability.

### Bounded verification

- Final shared Markdown and Market packages: PASS1.866s /2.486s after all independent review corrections, including empty-alt/repeated images, formula links and source list/rule geometry. The dedicated list/rule geometry plus two affected image exports passed2.288s; the three image/formula regressions passed1.790s before this final combined gate.
- Existing Files renderer package: PASS3.961s. Targeted legacy Chat file-mention/read-only-history/thinking-Markdown cases: PASS2.654s. Cmd embedded-Market integration: PASS2.867s.
- Installer source/resource Node fixture: PASS. git diff --check: PASS. No NSIS build or install claim.
- Independent Spec review closed its two P2 findings; Standards closed its three P2 findings after bounded read-only delta review. No remaining P1/P2 was reported for those corrected deltas. These reviews do not close the explicit source/runtime fidelity limits above.
- Two early shared-widget fixture failures matched plain paragraph labels incorrectly because RichText exposes merged paragraph text; corrected to observe actual merged bounds and click the measured inert text positions. A shared renderer compile error used a nonexistent Painter.FillRect API; corrected to existing Painter.Fill before the final passing gate. These were development failures, not omitted passing runs.
- No network/security configuration, user data, credentials, installed process, original repository, whale-code, remote branch, CI or scheduled automation was changed.

### Clean Windows artifact for this batch

Product commit `519d31b9bffc3c8b19060a4854baf7e5b2cf2ddb` on `codex/native-workspace-parity-20261019` contains the README, installer resources, regressions and updated mapping. `go build -trimpath` succeeded from a clean checkout using Go1.27.2, Windows/amd64, CGO_ENABLED=0. Embedded `vcs.revision` matches that commit and `vcs.modified=false`.

New artifact: `E/whaleisle-native-readme-519d31b.exe`,73701888 bytes, SHA256 `7FED9915B51D6816FCB8C6F3B108662C67D0D342961C5B8FDE0A43A9BBC24D4E`. Adjacent `.build.json` and `.buildinfo.txt` retain machine-readable provenance. Prior candidate EXEs were preserved. This candidate was not launched, installed or claimed to inherit dcfb14c's actual startup observation; its runtime/IME/native visual acceptance remains open. No NSIS installer was built.

## Solver, timeline and h3 batch after3f643ef

Execution continued on the same isolated branch, with numerical solver, animation timing and README cascade in separate file ownership. The integrator owns the background scheduling, input hooks and engine destruction. Source first, implementation second, focused tests/profile after product implementation. No extra product setting or dependency was added.

| Original component / state rule | Current native implementation | Regression / direct evidence |
|---|---|---|
| cursor-fluid.ts independent128 simulation and1440 dye grids; CLAMP_TO_EDGE bilinear fields | NEW third_party/mygo/ui/fluid_solver.go accepts separate resolutions, retains independent velocity/dye/pressure/curl/divergence buffers; live Resize resamples readable velocity/dye | [TestSplashSolverSplatResizeAndDisplay](../third_party/mygo/ui/fluid_solver_test.go#L113); [TestSplashSolverSeededFiniteAndDeterministic](../third_party/mygo/ui/fluid_solver_test.go#L157); original profile measured below, not enabled as default |
| Source curl, vorticity3, divergence with reflected boundary velocity, pressure*.1,20 Jacobi iterations and gradient subtraction | Same ordered CPU numerical passes; float32 rather than original half-float | [TestSplashSolverShaderPasses](../third_party/mygo/ui/fluid_solver_test.go#L35); [TestSplashSolverQuiescent](../third_party/mygo/ui/fluid_solver_test.go#L11); source shader read-only review |
| Source additive splat,radius.002 at100%,force6000*speed,RGB.15,click10x; latest normalized event delta; clock-selected random palette | Solver Splat plus NEW decorative_fluid_worker.go; input.go records latest event without input consumption, source aspect correction and seeded worker random state | [TestSplashWorkerSeededInputsAndBoundedCoalescing](../third_party/mygo/ui/decorative_fluid_worker_test.go#L124); [TestSplashSolverSplatResizeAndDisplay](../third_party/mygo/ui/fluid_solver_test.go#L113) |
| Source dt<=.016666,velocity rational dissipation2,dye3.5*speed/100,normal-gradient shading and alpha=maxRGB | Solver Step/Render; center-grid sampling optimization retains formulas and compares against general sampler; display texels use physical DPI-scaled size, pointer normalization stays in DIPs | [TestSplashSolverShaderPasses](../third_party/mygo/ui/fluid_solver_test.go#L35); [TestSplashSolverCenteredRenderMatchesSampler](../third_party/mygo/ui/fluid_solver_test.go#L209)(max1/255 output byte difference); seeded finite/premultiplied regression |
| Source hidden pause,4s idle tail, live retune, field-preserving resize and disposal | One event-driven background worker per active native splash; one in-flight job and one latest pending input, UI copies into one retained bitmap; paint-only refresh, engine close releases worker; native held flag resumes on SurfaceShown | [TestSplashNativePaintSchedulingRetuneAndDisposal](../third_party/mygo/ui/decorative_fluid_worker_test.go#L29); [TestSplashIdleHiddenAndCloseDoNotSchedule](../third_party/mygo/ui/decorative_fluid_worker_test.go#L85); no actual user process or GPU benchmark |
| cursor-fx.ts random palette; rounded cell size controls both interpolation and stamp; live size updates retain cells | decorative_effects.go removes fixed palette cycling and size-reset, uses source rounded spacing and gray empty-palette fallback | [TestPointerTrailRetuneRandomPaletteAndRoundedSpacing](../third_party/mygo/ui/decorative_fluid_worker_test.go#L242) |
| TypingFxLayer source400ms token,per-property stop/ease,drop35%/flash20% alpha1,relative step-end caret | U/composer/effect_timing.go and typing_effects.go; source(.4,0,.2,1) inversion; same native input/IME adapter | [TestEffectTimingCSSBezier](../internal/ui/composer/effect_timing_test.go#L17); [TestEffectTimingTypingKeyframes](../internal/ui/composer/effect_timing_test.go#L34); [TestEffectTimingCaretScheduling](../internal/ui/composer/effect_timing_test.go#L64); [TestEffectTimingTypingLifecycle](../internal/ui/composer/effect_timing_test.go#L198) |
| ComposerBeam source distinct CSS ease/ease-in-out/bezier,420ms transition and interrupted reversal; independent spin/hue/breath activation | beamTimeline per composer card and settings preview, child clocks, source palette closure; no full gradient/raster replacement | [TestEffectTimingBeamActivationAndReducedMotion](../internal/ui/composer/effect_timing_test.go#L85); [TestEffectTimingBeamTransitionReversal](../internal/ui/composer/effect_timing_test.go#L125); [TestEffectTimingBeamChildActivation](../internal/ui/composer/effect_timing_test.go#L144); [TestEffectTimingBeamTrajectories](../internal/ui/composer/effect_timing_test.go#L175) |
| Market h3 cascade: same specificity, later-loaded .readme h3 overrides only size14px/line22px; weight700/margins32/16 retained | Market opts into absolute DocumentH3FontSize/LineHeight, shared Markdown defaults and Chat untouched | [TestMarketReadmeH3CascadeKeepsAbsoluteMetricsAndSharedMargins](../internal/ui/market/readme_test.go#L162); [TestMarketReadmeH3CascadeWrapsAtFourteenPixels](../internal/ui/market/readme_test.go#L197); existing source heading/quote geometry |

The h3 source chain is ui-primitives/tsdown.config.ts staticLinked -> web/src/seed.ts20 primitives import -> boot.ts module loader -> Market clientBundle -> tsdown.client.ts42 styleInjectionModule appends a style tag when its factory runs. Both .markdown h3 and .readme h3 have specificity(0,1,1); MarketSection.module.css436-439 therefore wins14px/22px. This resolves the source cascade question without claiming a captured original window.

### Measured CPU/rendering decision

Windows amd64,i7-11800H,Go1.27.2,CGO0,1440x900 viewport; single local benchmark runs, not cross-device guarantees. Values include shaded display unless noted. `E/windows-fluid-implementation.json` retains measurements, scopes and profile paths.

| Workload | Observed time | Allocation / retained data |
|---|---|---|
| Original128/1440 CPU profile,Step+Render |1458.256ms/frame |80,465,920B numerical +13,271,040B output;0 steady alloc/op; excluded from default |
| Initial64/64 CPU profile before targeted sampling optimization |4.168ms/frame |365,568B numerical +26,112B output;0 steady alloc/op |
| Final64/64 Step+Render |2.146ms/frame |0B/op,0alloc/op; same retained buffers |
| Final64/64 Splat+Step+Render |1.496ms/frame |0B/op,0alloc/op; different workload/sample run, not directly subtracted from previous row |
| Background request through result,354 warmed frames |mean1.561ms,p50=1.534ms,p95=2.509ms,max3.062ms |0B/op,0alloc/op; includes channel and thread scheduling |
| UI waiting bitmap paint / receive+copy+Changed+paint |46.49ns /548.2ns |0B/op,0alloc/op; copy includes26,112 output bytes |
| Prior3f643ef96x64 approximation,Splat+paint |0.379ms/frame |24,752B/op,4alloc/op; temporary exact old-source benchmark fixture removed after measurement |

Decision: native default remains explicitly a bounded64/64 CPU profile with a maximum60Hz job rate, running off-main. One retained UI copy adds26,112B at this aspect, for about417,792B field/output/UI pixels plus small controller overhead; disabled/unused state starts no worker. This meets the planned measured4ms worker/0.5ms UI/4MiB buffer budgets on this machine, including the sampled worst frame. It does **not** prove total active CPU equivalence: the physical solver costs more CPU than the old approximation. No GPU/display-driver duration or actual desktop utilization was measured. Output upload uses one stable bitmap ID rather than a newly allocated image each frame.

Source128/1440 is intentionally not selected for the default because CPU time and93,736,960B field/output cost are unacceptable for this path. This resolution difference is explicit, not an automatic hidden fallback. A performant full-resolution port still needs stateful half-float multi-pass GPU rendering across D3D11 and other MyGo renderers. Existing Painter.Effect is stateless/one-pass with20 scalar parameters; no source-sized GPU path was added or certified. Native source appearance remains unverified.

### Media audit and remaining differences

Original render.tsx708-735 uses browser img, without first-frame extraction. Current ui/bitmap.go uses image.Decode; Go image/gif returns image[0]. x/image@v0.46.0/webp lacks ANIM/ANMF decoding, so animated WebP fails and README retains alt. ui/svg.go and internal/svg/compile.go support basic vector content, but omit text/image/filter/pattern/marker/animate; parsing may succeed with incomplete painting. SVG raster has a4096 single-edge cap. No decoder or dependency was installed/changed, and no complete-media equivalence is claimed.

Typing glyph scaling still differs from CSS centered transforms. Beam radial/inset lights, white highlight, separate conic bloom/blur and three-layer dynamic opacity retarget remain approximate. GPU full-resolution fluid, half-float rounding and actual same-state original/native capture remain open implementation/acceptance work. These source/renderer limits are distinct from unavailable native capture/IME/Host/NSIS prerequisites recorded earlier.

### Verification and environment checkpoint

- Latest SDK numerical/native scheduling/repaint targeted gate: PASS1.115s. Actual input, retune, portrait resize, idle/hidden/resume, disable/close and deterministic output are exercised; no live window or user Host was used.
- Final affected application packages: Composer PASS2.486s, shared Markdown PASS1.162s, Market PASS1.582s. Includes existing native input/IME ownership regressions; real Windows IME is still unverified.
- Independent Standards and Spec bounded reviews found0 new confirmed P1/P2; Standards also rechecked the native-event delta/copy benchmark correction. The integrator then corrected and directly tested hidden SurfaceShown resumption.
- One test-only compile failure referenced nonexistent Scene.Background; corrected to existing Scene.Clear before passing. A one-off tool transport disconnect caused only read commands to fail; subsequent Get-Location, SDK tests and benchmarks exited0. No environment restart or permission escalation was performed.
- No CI/push/merge/release/install, unknown executable download, network/security modification, rejected Git/CIM retry, user-process restart or user configuration change occurred. Earlier EXEs are preserved; new clean build provenance follows separately.

Final source check corrected display shading at high DPI: source uses drawingBuffer texels, so the worker receives w*p.scale/h*p.scale while pointer coordinates remain normalized from DIPs. [TestSplashNativeDPIUsesPhysicalDisplayTexels](../third_party/mygo/ui/decorative_fluid_worker_test.go#L153) PASS1.261s confirms 1440x900 at200% sends2880x1800 display dimensions. This is a focused source geometry regression, not an actual Windows DPI capture. The productf9f4d2d EXE was built before this correction and is preserved as historical; the next clean candidate includes it.

### Solver candidate provenance and requested branch publication

Product f9f4d2d adds the solver/timeline/h3 batch; bf347ff12f557b9077a75869244e0964465f3fda adds the focused DPI correction. New current artifact E/whaleisle-native-fluid-bf347ff.exe was built with Go1.27.2,Windows/amd64,CGO0,trimpath from clean bf347ff,embedded vcs.modified=false. SHA256 73C895B57420A583B82FD4F693F015814E053723DD40FBB3B6FABFAC1772850C. Adjacent .build.json/.buildinfo.txt preserve source identity. f9f4d2d and all earlier EXEs are retained; neither new candidate was launched or installed. Successful compile is not source-window visual acceptance.

The user subsequently requested commit/push. Origin was checked by an actual successful command and exactly matches https://github.com/ChisaAlter/WhaleIsle-MyGo.git, independently verified by the parent as the user's private repository with push permission. Publication scope is only codex/native-workspace-parity-20261019; no force push, main merge, PR creation or release. The current three workflow push filters target main/audit-security or the older reconstruction branch, not this branch; no CI dispatch is authorized here. Review of task-added paths and common credential signatures found no credentials/temporary executable/profile/database files in the branch diff; evidence executables and profiles remain outside tracked source. Push outcome and remote-head equality are recorded by the final execution report, not assumed from authorization.

## Renderer protection and media batch after b426c7a

This batch continues on the isolated branch and is local only. The preceding branch push was verified at b426c7acacfa365a07f73bc17767dbd0e0a3461c; its authorization does not cover these changes. Source-equation CPU fluid remains explicitly reduced resolution (64/64), not exact original visual reproduction. The historical 1.458s/frame original-resolution CPU measurement remains excluded from usable integration.

| Original component / current gap | Implemented scope | Direct regression |
|---|---|---|
| Fluid native integration could allocate without an aspect bound or submit at60Hz on slow hardware | Pre-allocation 4MiB ceiling for retained numerical fields plus worker/UI RGBA buffers; extreme geometry releases the decoration owner while preserving preferences. Start-to-start pacing is max(16.67ms, measured work/0.24), with a single pending input and stable bitmap | [Memory and extreme geometry](../third_party/mygo/ui/decorative_fluid_worker_test.go#L170); [slow processing, deadline and stable upload](../third_party/mygo/ui/decorative_fluid_worker_test.go#L207) |
| ComposerBeam stroke, inner glow and bloom each retarget their own opacity | Independent420ms CSS-ease transitions from current values, source rounding/clamping, reversal shortening only for true reversals; bloom-zero fade tail and source60s night sampling | [Independent retargets](../internal/ui/composer/beam_opacity_test.go#L15); [interruptions/reversals](../internal/ui/composer/beam_opacity_test.go#L43); [breathing](../internal/ui/composer/beam_opacity_test.go#L73); [night](../internal/ui/composer/beam_opacity_test.go#L112); [source rounding](../internal/ui/composer/beam_opacity_test.go#L142) |
| MyGo SVG parse success may still omit unsupported README content | Explicit supported static SVG/CSS subset gate before parsing; known unsupported constructs take existing authored-alt failure path. Supported shapes/gradients/masks/clips, simple CSS and unconditional switch stay accepted | [Supported subset](../internal/ui/market/readme_svg_test.go#L13); [actual switch rendering](../internal/ui/market/readme_svg_test.go#L28); [15 unsupported categories and actual alt layout](../internal/ui/market/readme_svg_test.go#L55) |

The fluid cap is retained fields/two pixel buffers, not RSS, GPU texture/mipmap memory or transient resize peak. Pacing targets processing time equal to at most24% of one core; it does not measure desktop utilization or guarantee source frame rate. Geometry rejection does not silently lower resolution or clear user configuration. Waiting uses the native timer when beyond the next frame. Existing hidden/resume, idle, disable and close ownership remains covered. Beam radial/inset coverage, white highlight, separate conic bloom/blur and centered glyph scaling still differ; no replacement approximate raster was added. The SVG gate is a known-capability boundary, not a full conformance proof or full SVG implementation.

### Concrete GPU implementation boundary

Read-only inspection found reusable pass machinery but no existing complete persistent float-field path. No GPU command, hidden window, device installation or driver change was performed for this audit.

| Backend | Safe implementation primitives already present | Missing code contract; why existing backdrop targets cannot serve the solver |
|---|---|---|
| D3D11 | internal/gpu/d3d11/d3d11.go pass:853, readBackdrop:830 unbind, bindState:810 restore; shader.go compile:88 uses existing system HLSL compiler | newTarget:896 is BGRA8 RTV/SRV; newTexture:603 exposes format but only creates SRV, not an empty float render target. pass binds one sampled input at t4 with blur-specific48B constants. Backdrop pairs are transient/rebuilt on growth and 8bit clamps signed velocity |
| OpenGL | internal/gpu/gl/gl.go pass ordering:769, main state:747, compile/link:474 | newTexture:537 fixes UNSIGNED_BYTE, fitBackdrop:813 allocates RGBA8, pass:804 binds one input/unit0 and transient target. Needs float-renderable/filter capability checks, persistent FBO fields and two-input passes |
| Metal | internal/gpu/metal/metal.go newTexture:661 accepts format, RGBA16Float:67; passPipeline:607 creates unblended format-specific pipelines | pass:883 is one-input; releaseTextures:999 / trim:1059 discards backdrop after2s idle. Solver requires retained state and source4s behavior, separate lifecycle and two-input bindings |

Required narrow shared implementation: a fluid frame contract and renderer-owned persistent velocity/dye/pressure ping-pong fields; ordered source passes, multiple sampled inputs, resize resampling, final composition, close/device-loss recreation and per-backend float-render/filter checks. Existing scene.Effect/EffectOp carries only five vec4 values and a single backdrop; scene.Image is CPU RGBA8. Restore viewport/scissor/pipeline/sampler/bindings and unbind both inputs after solving; disable blending during numerical passes. Do not build a generic GPU framework or borrow blur targets.

Windows ui/window.go recreates a failed renderer; GL and Metal also have context/device failure paths that a new owner must handle. D3D11 RenderOffscreen (offscreen.go:16) uses a hidden HWND; existing TestDrawsAsTheCPURenderer is a reusable retained-renderer test pattern, not a current float-field test. Device creation alone can avoid HWND but does not supply cross-frame field/readback/performance acceptance. WARP correctness would not certify hardware GPU latency. These are unimplemented code interfaces, not missing authorization. Full GPU fluid remains unfinished.

### Animated-media implementation boundary

GIF remains first frame. Standard image/gif can decode delay/disposal/loop information, but safe source-equivalent integration still needs (1) a public UI-thread stable-bitmap update that increments scene.Image.version and invalidates mipmaps, (2) deterministic element unmount/close cancellation, and (3) bounded cancellable decode/composition instead of expanding every canvas frame. NewBitmap may wrap packed RGBA pixels but creates a new image identity; mutating pixels without Changed leaves GPU/mipmaps stale. ui.Local pruning does not call a close hook. Painter.After is reusable for visible scheduling, but does not solve upload or ownership. These are code implementation dependencies, not permission or screenshot-tool blockers. Animated WebP decoding and full SVG rendering remain unsupported.

The original lockfile pins Electron44.3.0, whose [DEPS](https://raw.githubusercontent.com/electron/electron/v44.3.0/DEPS) selects Chromium152.0.7977.78. The inspected installed node_modules version was44.2.0/Chromium152.0.7977.76; the two references are distinct. Chromium's [DeferredImageDecoder](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/third_party/blink/renderer/platform/graphics/deferred_image_decoder.cc) maps frame durations <=10ms to100ms. Its [ImageAnimationController](https://raw.githubusercontent.com/chromium/chromium/152.0.7977.78/cc/trees/image_animation_controller.cc) also coordinates visible drivers, BeginFrame, first-loop and long-gap behavior. A naive per-frame timer is not accepted as browser-equivalent playback. No approximate animation player was added to claim completion.

### Checks and acceptance limits

- New native cap/pacing regressions PASS1.461s; affected SDK native lifecycle/repaint gate plus handoff benchmark PASS1.928s. A final test-only fix removed a metadata read before channel completion in the disable branch; that affected lifecycle case passed1.265s. No race detector was run with this CGO0 Windows toolchain.
- UI handoff benchmark on the same i7-11800H: waiting56.49ns/op, receive/copy/Changed/paint529.6ns/op; both0B/op,0alloc/op. These are CPU fixture operations, not GPU upload/desktop frame measurements. Solver equations did not change; prior costly benchmarks were not repeated.
- Final Composer package PASS3.155s and Market package PASS1.880s. Independent Standards review found0 confirmed P1/P2 in pacing/memory/ownership. Spec review found an over-restrictive SVG switch gate; the unconditional switch was restored, actual rendering tested and the P2 closed by bounded delta review. No remaining confirmed P1/P2 for reviewed changes.
- Initial positive comma-selector fixture failed because the parser emits QualifiedRuleGrammar per comma; handling was corrected before passing. These are recorded development failures, not acceptance passes. git diff --check passed.
- Native same-state original/candidate screenshots and actual Windows IME still need a supported native capture/input surface. Real Host acceptance needs an isolated matching Host/data fixture. Installer acceptance needs the NSIS runtime and an isolated installation account/environment. None is treated as a substitute for the unfinished GPU/media code above. No installed process/configuration, original repo, whale-code, security/network settings, denied operation, remote branch or CI was touched this batch.

### Local candidate provenance

Product commit054fde1e7c98f9bba9f666b14cc646df429033aa contains this protection/opacity/SVG batch and regressions. Offline Windows/amd64 build with Go1.27.2, CGO0 and trimpath passed from clean source; embedded vcs.revision matches054fde1 and vcs.modified=false. Candidate E/whaleisle-native-guard-054fde1.exe is73844224 bytes; SHA256 2B7B8A801E43F54C10D57CA3FC7253860FE6DF9390C554F52559FD34FFA4240C. Adjacent .exe.build.json and .exe.buildinfo.txt retain provenance, and E/windows-fluid-implementation.json appends this batch without replacing historical bf347ff metrics or b426c7a publication. No launch/install or runtime/visual acceptance is claimed. Prior candidates remain intact. This product and subsequent evidence commit are local only, with no new remote verification or push.

## Core availability: session runtime, optional goal and view preferences

The user reported all selected sessions erroring, then two independent errors in a new-session screenshot. GPU/GIF work was paused before product edits; its plan is safely committed on codex/gpu-gif-plan-20261019@16fde68. The hotfix branch is codex/session-runtime-compat-20261019, based on b881c9d. The original native-parity branch and all old EXEs remain intact. No push is authorized for this batch.

### Session reader root cause and repair

The parent directly inspected the screenshot: resume and follow both report required, non-ignorable job/registered at seq20 unknown to this harness. Disk inspection found C:/AI/WhaleIsle-MyGo/runtime/vendor/deepseek-harness at0.2.1-alpha.1, matching the historical reference pin5badb15009ae1756c3afe0ae0cef1faafc290ccc. The installed original C:/软件/Whale Isle/resources/vendor and original source use0.2.1-rc.2, pin687f282e8625c6b8d11ff58211a90524be2e4317. Both claim logical session format4. The rc.2 built vocabulary adds job/registered, job/settled, conversation/next-prompt-request, team/control and working-directory/change. Actual alpha validateStoredEvents rejected the same synthetic seq20 job event with the screenshot error; actual installed rc.2 accepted it, while still rejecting unknown required events.

The disk Go EXE currently embeds b426c7a with vcs.modified=true (SHA256162695545B30034096D7730F8DA99AC01C023E42D96FD60783BDCE9CC6393DE0). Process observations in this restricted surface did not expose an actual Go/Host process path. That is not proof that the user's instance is absent; actual running JS/Node selection remains unknown. CIM and original Git metadata queries previously denied were not retried.

Code tracing places the shared failure before either response: resume and follow both reach observeSession -> readColdSessionLog -> validateStoredEvents. The old build-harness custom path populated only missing runtime directories; rebuilding a new EXE therefore left the old alpha reader beside it. Product e537058 repairs this boundary:

- Resolve checks the actual compiled event registry, including configured standard source launchers; PlanStart rejects an old reader before preparing the shared profile. No session inspection, migration or unknown-event ignoring is added to startup.
- scripts/session-runtime.mjs explicitly inspects, stages and verifies a complete released rc.2 vendor dependency graph into a fresh destination, plus its matching Node. It executes the actual strict reader against metadata-only fixtures, verifies the source pin, relocates internal links, hashes artifacts and rejects mixed versions. It never starts Host or reads user sessions.
- The copied visual-reply bundle retains the existing per-operation HTML preview readiness gate. The exact released input/output hashes and all seven awaited checks are pinned; older904/e9 safety checks are preserved. This is a product patch only on the copied runtime, not a system permission change or modification of the original.
- build-harness rejects the obsolete reference or a mixed existing tree and supports an explicit RuntimeSourceResources/fresh RuntimeDestination. Packaging preserves the selected runtime's Node hash, verifies the copied output, and only then continues toward a ready marker. Explicit mismatched Node overrides are errors. Independent Spec/Standards review identified this Node-selection mismatch; both closed the correction after bounded delta review.

### Actual existing-session read evidence

Three existing compressed v4 snapshots were read without writing their files, starting a backend writer or Host, or requesting a model. Complete Zstd frames were decoded using the original pure scanner; actual alpha/rc.2 validators and rc.2 readColdSessionLog were exercised through an in-memory read handle. Only schema metadata and counts were returned; no private title, session ID, message body or credential was output. All read handles closed and source bytes matched before/after each read.

| Snapshot | Stored events | User / assistant message events | Alpha reader | rc.2 cold read |
|---|---|---|---|---|
|1|1316|33 /166|required job/registered refused|accepted;1319 events including in-memory interrupted-turn closers|
|2|324|8 /45|same refusal|accepted;327 events|
|3|1108|18 /134|same refusal|accepted;1110 events|

This verifies real existing event data through the read API, not new MyGo UI resume/follow or message rendering acceptance. An initial one-shot Zstd probe read only the first frame/header and was not treated as empty history. A direct TypeScript import needed Node's process-local experimental transform-types flag for parameter properties; no tool/component installation or system setting change was made. E/session-runtime-compat-20261019.json retains sanitized measurements and exact limitations.

### Separate goal and configuration failures

Goal absent was generated by Go itself after a successful empty goals/get response. The original API returns GoalView | undefined; no goal is ordinary optional state. Background success now folds the goal without changing LastAction. Real read errors retain goals/get ownership in ViewState.GoalReadError, bound to client/session/follow generation, cleared on success and isolated across selection/Host changes. Write-action failures retain their existing action-error path. No empty goal panel or new entry point is added.

View preference retry formerly repeated Save against the same stale config baseline, so an unrelated original/Go/background config update could make every retry fail. A dedicated persistent adapter now retains the caller's acknowledged baseline per Host. Saving rereads disk and three-way merges only changed groupBy/orderBy fields and individual collapsed/sessionOrders keys, retaining other Hosts, unknown keys and external settings. Same-field divergent changes remain explicit ErrConflict with pending local state; CAS is not disabled. Each rename attempt checks the fresh baseline; a conflict stops immediately instead of blindly retrying. Rebinding the same config store retains adapter identity. Actual cross-process atomicity beyond the retained compare-before-rename check is not claimed.

Goal repair is local commit7e54fa4; view merge is85bde0d. Bounded Spec review found the first-edit default-value boundary: absent groupBy/orderBy must compare as the UI's workspace/manual defaults, otherwise an unrelated collapse looked like a group change. The shared comparison now normalizes base/current/next, its new regression passed, and Spec closed the P2. Standards reported no confirmed P1/P2 for this goal/view delta. No remaining confirmed review finding is being presented as a pass for the unavailable real Host/UI gate.

Direct regressions: [built vocabulary vs stale source](../internal/harness/session_runtime_test.go#L30), [preparation preflight](../internal/harness/session_runtime_test.go#L71), [Node binding](../internal/harness/session_runtime_test.go#L130); [optional goal](../internal/store/session/goal_read_error_test.go#L49), [read-error ownership](../internal/store/session/goal_read_error_test.go#L86), [late goal error](../internal/store/session/goal_read_error_test.go#L145); [per-field/Host merge](../internal/config/workspace_preferences_merge_test.go#L43), [conflict/retry](../internal/config/workspace_preferences_merge_test.go#L94), [rename CAS](../internal/config/workspace_preferences_merge_test.go#L150), [first-edit default normalization](../cmd/whaleisle/view_preferences_test.go#L124).

### Checks and Host smoke matrix

- Runtime-focused Go regression PASS2.119s after Node-binding correction; PowerShell parse checks passed. Readiness byte-patch/mock fixture passed. Seven distinct Node runtime cases passed; two link cases are BLOCKED by EPERM, so that suite is not green. A broader harness gate and old socket-based goal groups produced no completion result and were cancelled; their cause remains unconfirmed and they are not passes. No identical-condition rerun or CI dispatch followed.
- Five new in-memory, no-socket goal-read cases passed2.593s, including optional results, real errors, owner changes/late responses and write-error separation. Temporary configuration/cmd targets passed3.663s/3.142s, including concurrent fields/Hosts, same-field conflicts, retry/reopen, repeated saves and a write appearing between rename attempts. User configuration was not modified.
- After default normalization, only affected view-preference targets were repeated: config PASS1.747s / cmd PASS2.755s. Existing broad gates were not rerun. Runtime Node-provenance and view-default P2 findings are both corrected and closed by bounded review.

| Entry / operation | Read-only smoke scope | Actual status here | Persistent/paid actions withheld |
|---|---|---|---|
| Host connection | Unauthenticated GET loopback:3080 root, no body/credential read | Unavailable from this restricted surface; does not establish user Host is stopped | Start/restart/replace existing Host |
| Sessions | Strict reader and cold read of existing snapshots | Actual results above; UI follow/resume/display unverified | Prompt, new session, resume activation against user data, history mutation/delete |
| New-session goal | goals/get optional/error ownership | No-socket actual client/store fixture passed; actual Host/Hero unverified | Goal set/clear and model execution |
| Workspace | workspace/follow, directoryPicker/list | Authenticated Host not available; not executed | initializeDefault/create/rename/delete/trust/order writes |
| Plugins | Existing manager profile/inventory queries | Authenticated Host not available; not executed | Install/enable/remove/update |
| Tasks | schedule/catalog/list/history; job/list/follow | Authenticated Host not available; not executed | Create/run/retry/cancel/delete or scheduled model delivery |
| Settings | settings/describe and read-only inspection | Authenticated Host not available; not executed | update/mutate/replace, account/credential changes |

### Exact staging blocker and minimum next operation

The installed source graph was checked read-only:25375 regular files,6310 real directories,3143 links; no external, ancestor or broken target was found. Two temporary fixtures failed at fs.symlink(target,link,'junction') with EPERM before reaching production stage. Targets were the internal node_modules/@deepseek-ai/dsh-session link and external-link-rejection fixture under sandbox AC/Temp/session-runtime-fixture-G6L3Do and -09K9U4. The exact token/policy cause is unknown. No junction creation was retried, no elevation/security change was requested, and the real stage copy was not attempted after this evidence. No successful runtime marker or complete-deployment claim was made.

Minimum prerequisite: a normal authorized Windows execution surface able to create links inside a fresh workspace runtime directory. Run the existing script's two link cases and explicit stage/verify there, with no administrator or policy change. Then a separately owned isolated candidate and matching data fixture can perform the read-only smoke matrix and actual message-display check. Switching or restarting the user's active instance remains outside this execution. These missing operations are separate from completed source corrections; a rebuilt EXE alone does not upgrade the adjacent Host.

### Core-recovery build provenance

The three separate product commits are e537058 (Host/runtime),7e54fa4 (goal state),85bde0d (view merge). Clean committed7075a152e395565faa9174a5c69ac56fce8aa905 also includes their evidence report. Offline Go1.27.2 Windows/amd64 CGO0 trimpath build passed from that clean revision; embedded vcs.revision matches and vcs.modified=false. New E/whaleisle-core-recovery-7075a15.exe is73894912 bytes, SHA25646A56ABB231077F8E906463A790EB1C55DAE07694259AC7DCBFC597A7E755206. Adjacent .exe.build.json/.exe.buildinfo.txt preserve metadata. It was not launched, installed or pushed. It is a compile-verified shell, not a complete compatible runtime deployment. Prior EXEs, user files/processes, original and whale-code are unchanged.

## Integrated functional recovery milestone (local only)

This section supersedes the earlier AppContainer staging blocker, without rewriting its historical failures. Branch remains `codex/session-runtime-compat-20261019`; no push, release, main merge, installed replacement, user-instance restart or original/whale-code modification occurred.

The sanctioned ordinary-user execution channel was inspected with the pinned installed Node v24.21.0; it is not an administrator token or a system-setting change. A new workspace-only junction probe passed realpath/sentinel read and removed its owned link. An owned ephemeral HTTP listener/GET passed. Under that verified context the complete runtime script passed all nine cases, then staged and verified rc.2 into `task-2/core-recovery-runtime-rc2`: 25,375 copied ordinary files, 6,310 real directories, 3,143 relocated internal links, exact upstream pin 687f282e8625c6b8d11ff58211a90524be2e4317, pinned matching Node and retained visual-reply readiness checks. Old sandbox EPERM/EACCES observations no longer describe the current prerequisite.

Actual parallel implementation and integration:

| Local commit | Product behavior | Evidence and limit |
|---|---|---|
| 10e1398 | Plugin inventory and registry replies stay with their owner/dialog; schedule writes/history use connection epochs; market confirmations retire on Host changes | New fake-transport regressions and plugin/market packages passed. The two existing schedule socket cases, previously denied in AppContainer, passed 1.210s in the verified ordinary-user channel. These are regressions, not live plugin installations or scheduled executions. |
| a4af895 | Sidebar normal/archive empty states and return action; browse archive filter and batch targets agree; file rows recover compact geometry, hover reference and context actions | Sidebar/files targeted behavior and geometry checks passed. OS opener, actual file write/conflict and every preview format remain unaccepted. |
| bcb3d3e | Approval/question takeover shares one composer seat; draft and selection survive; workspace-less Hero becomes a directory-selection card and cannot submit | Composer package and frame behavior checks passed; two takeover images in E/composer-structure-20261019 are explicitly UI Tester fixtures. Actual Host approval/question lifecycle remains unverified without an agent request. |
| 3d043c2 | Collapsed caption restores New session, reveals conversation across exclusive routes while keeping drafts; manual and catalog market actions recheck the rendered Host before worker admission | Targeted frame/cmd checks passed, including reconnect and route state. Spec review found and closed the catalog two-phase race and exclusive-route visibility gap. |
| 6d507ff | A new authoritative workspace baseline retires only its own matching prior reader error, retaining later user-action errors | Real ready-window capture exposed the stale feedback; session regression passed 4.950s and real native rerun below confirms the correction. |

`17f97cb` adds the [full original-function acceptance matrix](native-full-function-acceptance-20261019.md), covering actual source entries and distinguishing real, fixture-only, incomplete and unverified work. Its 91 IDs are scope inventory, not a completion count.

### Real Windows native window evidence

The opt-in `nativeacceptance` build runs the production `main()` on MyGo's required process main thread, with a hidden 1200x800-DIP window, fresh isolated config/workspace, explicit staged vendor/Node and no credentials or prompts. It skips protocol/login integration through the existing isolated mode. Every exit marks only this owned shell quitting and stops its Supervisor, preventing close-to-tray from retaining it.

E/native-core-recovery-blank.png first proved production Host startup, but captured the launcher page. E/native-core-workspace-3d043c2.png was an intermediate connecting page and is not accepted as ready. Waiting for actual mux, session list and workspace readiness produced E/native-core-workspace-ready.png, which exposed stale workspace-follow text; it was not treated as a pass for a clean main page. After the focused correction, E/native-core-workspace-6d507ff.png and its adjacent JSON record Host/mux/list/workspace ready, with the stale text absent. The real native invocation returned PASS and the owned process exited. This verifies blank main-workspace startup, not existing-history rendering or original pixel parity.

Two earlier acceptance setup mistakes are retained as failures: unquoted PowerShell test arguments exited during flag parsing; calling App.Run from a test worker violated MyGo's main-goroutine requirement. The first actual QA profile lacked its configured directory and correctly failed; reusing that failed profile correctly paused auto-start. A separate fresh profile with its directory created passed, preserving the failed profile rather than clearing recovery configuration. These are QA setup corrections, not user-state resets.

Read-only actual glyph probing through the vendored MyGo/DirectWrite path resolved the default stack's Chinese to Microsoft YaHei and English to Segoe UI, with SimSun as a separate control. No font parser/stack bug was established, so no speculative SDK/font change was made. The probe does not claim every captured glyph was instrumented.

Core history gate now has real isolated evidence: `495a4ff` adds the opt-in checker using production Supervisor/Prepare and Go Store. Three physical v4 Zstandard snapshots each contain 22 events, including required job/registered at seq20. Actual session/list and Go Store both return three; all three production agent-resume resolver calls and WebSocket follows return HistoryReady and one matching message. The successful E/core-host-check-resume.json records 12.416s, zero model calls and confirmed owned Host/listener cleanup. The initial single-Zstandard-frame fixture was invalid: rc.2 requires a header-only first frame, then event frames. Its metadata reader intentionally omitted that malformed test artifact; correcting the owned fixture closed discovery without a product/history-loss patch. Previous failed-run reports are retained.

Production hidden native E/native-core-history.png confirms the selected message is painted in the conversation page, with HistoryReady, messageCount1 and no goal-read/workspace error. Actual settings/describe, pluginManager/listBundles and schedule/catalog reads also passed against that Host; adjacent JSON stores counts/results without bodies or credentials. This is an end-to-end compatible-runtime/history/display check using inert isolated data, alongside the earlier real-user-snapshot read-only cold evidence. It is not proof that the user's still-running old EXE/Host has changed.

That screenshot exposed a clipped status row. Full Frame.View reproduction identified the real constraint: without a terminal the conversation retained a fixed, initially stale height relative to the caption, clipping the footer. e21f4b8 uses the parent's remaining height in that case; open-terminal allocation is retained, and the composer seat follows original flex:none. The complete regression failed against the old fixed height and passed the correction across short/long history and stats/cost configurations. Final actual E/native-core-final-e21f4b8.png shows the restored message and full bottom row. Its metadata records outer window1200x800 DIP, actual content1185x792 DIP and PNG1778x1188 pixels. The same run passed a real production preference write after an independent config.Store edit, switched/restored grouping and freshly reread disk to confirm that external field was retained. This is owned QA configuration only. All three read endpoints still passed, zero model requests were issued, and the invocation exited PASS.

The now-working ordinary-user process view also confirmed the user's Go PID7800 at C:/AI/WhaleIsle-MyGo/whaleisle.exe. Its observed disk SHA256 remains162695545B30034096D7730F8DA99AC01C023E42D96FD60783BDCE9CC6393DE0, revisionb426c7a with modified=true; disk modification07:22:35UTC precedes process start07:22:44UTC. Original Whale Isle processes remain running. Neither was changed; loaded Host JS is still not inferred from the executable path. A later AppContainer exec transport disconnect affected two read attempts; an actual ordinary-user Get-Location passed and work continued there without restarting any user process or changing security settings.

Model requests, live account/remote/device flows, package installation/system integration and matching original-window screenshots are not covered by these results. The matrix retains their individual unverified/partial states. GPU/GIF depth work remains paused.

### Final clean candidate and remaining acceptance boundary

Clean committed16b22b2b81d6aa8af6c9f637b4708177cc2014f4 built offline as Windows/amd64 GUI (-H=windowsgui), Go1.27.2, CGO0, trimpath, embedded vcs.modified=false. New E/whaleisle-core-recovery-16b22b2.exe is73954816 bytes, SHA256B4DBAAEE7E5160597174D62A9BBDA5CF74C49A1AA967044584B19E233C30D9CB. Build metadata and full Go buildinfo are adjacent. The complete matching rc.2 runtime remains separately staged at task-2/core-recovery-runtime-rc2; this EXE is not an installed package and must not be substituted alone beside the old alpha runtime.

The ordinary clean EXE itself was started hidden only on the handed-off synthetic profile. It wrote a healthy production startup marker09:33:52UTC, after this candidate's build09:32:20UTC and after the previous native image09:30:15UTC. The script's original timeout is preserved in .exe.isolated-start.json: ConvertFrom-Json produced a DateTime, then reparsing its culture string shifted the comparison to01:33:52UTC. A read-only review proved this runner false negative; .exe.isolated-start-review.json records both the original result and actual fresh healthy marker. No redundant launch followed. Its owned PID4052/tree was stopped and the owned listener was confirmed closed; user PID7800 was excluded and untouched.

The actual final native view and concurrent view-preference write passed on the same production source, as detailed above. Independent Spec/Standards review findings in this batch were corrected and closed; no review or unit count is being used to certify the untested product scope.

Matching original-window/DPI/state capture and hover/menu operation remain unaccepted. The supported Computer Use skill, C:/Users/48818/.codex/plugins/cache/openai-bundled/computer-use/26.1013.40934/skills/computer-use/SKILL.md, requires reading guidance before native control: "You MUST read this before controlling Windows apps." Its guidance resource returned "skill environment is not available for this callback"; the enabled tool inventory has no node_repl/@oai/sky entry, while the CUA surface declares native APIs disabled. The original is running and remained read-only, but source comparison is not relabeled as real original-window acceptance. Candidate self-capture through its own MyGo API is a separate supported route and succeeded.

The full-function matrix retains live model/paid agent actions, accounts/credentials, real remote devices, system integrations/installers and unexercised page lifecycles as unverified or partial. No such action was silently authorized or simulated as a pass. Current user processes/configuration/session files are unchanged, all verified source work is in local branch commits, no push/release/CI/main merge or PR#2 merge occurred, and GPU/GIF work remains paused. This is a concrete core-workspace recovery milestone, not completion of the entire original desktop acceptance.

## Earlier complete-folder native recovery checkpoint (2026-10-19, source9232f52)

The complete local directory is `../WhaleIsle-MyGo-local-9232f52/`; its only user entry is `WhaleIsle.exe`. Desktop and preview were built from clean source `9232f52c7f371b6312911b14c2caf6d6e3756b6c` (Go VCS modified=false), with the verified rc.2 runtime, Node24.21, private pnpm12.4.1, Ghostty DLL/license, pet assets, preview helper and component samples. This is a fixed-path local folder with internally relocated junctions; it is not an installer or an archive for relocation to another machine.

Ordinary package startup from the task directory passed without WHALEISLE_USER_DATA, VENDOR, NODE or ISOLATED overrides. The QA process used only an owned HOME and OS hidden startup hint, so this proves the ordinary production entry/bootstrap/Host path rather than an OS-visible-window gesture. Startup recorded ready after about63 seconds; FreePort selected3082 while the already running apps retained3080/3081. Matching private Node PID23344 and desktop PID31356 exited through the application's managed stop, and3082 closed. The generated private test data was moved into E/portable-startup-profile-9232f52; the delivered folder has no data directory and creates its own on first launch. E/portable-ordinary-start-9232f52.json records the exact invocation and cleanup. An initial cleanup assumption about3081 failed the path ownership guard before any stop request; that pre-existing process was untouched.

Source comparison and actual native captures exposed and fixed three remaining space problems: file cards now use240x64DIP and image cards64x64DIP; text preview uses one source-matched38DIP read-only header or36DIP desktop-file header; new editor panels default to a single editor. Existing explicit Split/Highlight modes and their saved numeric values are preserved. Deferred header/tree menus retain exact Host/session/root/tab/read-generation ownership, and Save still compares the disk baseline before writing. The old screenshot's359 physical-pixel header span is approximately239DIP at1.5scale, not359DIP. Original window pixels remain unavailable.

The 91-row full-function ledger now records actual isolated Host, native, filesystem and local-service evidence separately from fixtures and remaining live-model, account, device, OS gesture and original-window gates. Actual terminal duplex, PDF/WebView navigation, session CRUD/import/recovery, saved rich message types, files/Office/Git, skills/presets/MCP/plugins/jobs/schedules, assistant metadata, SSH, DPAPI and local updater checks are bounded by their receipts. Six saved rich native states are in E/rich-history; no prompt/model/approval/question response, workflow run, real IM or notification delivery was issued. This milestone does not claim all91 features or original pixel parity passed.

Product review: E/independent-compact-native-review-20261019.json found no confirmed product P1/P2 in compact toolbar/card changes. Its test-readiness note was fixed in40c7309. Targeted source/default tests preserve explicit restored editor modes. Failed local QA attempts remain in evidence; no CI run, remote push, PR merge, release publication, installed/user instance replacement or user configuration reset occurred.

Checkpoint follow-up: restored full-width native trajectory event/content rows and selected-only inspector; the layout/copy regression and nested tab checks passed with independent review. Settings13 actual native states, usage strict persisted-corpus/exports/cache, directory hidden/path/newfolder/retry, application shortcut ownership and source font/reduced-motion fixes are now documented in the full-function matrix. Directory Retry was additionally verified for Tab/Enter/Escape; its portal focus scope is explicit, rather than a pointer-only focus exception. The final EXE is refreshed only after these changes are committed and the source is clean.

Phone resources were also completed in the local folder:25569 entries,3143 relocated links, closure SHA256 3a24c27571c2bef7d95baac64d0cf234d8ab1aa0c615cb17bba1a6c409cd008c; no test data or Miniflare tools copied. The final resource resolver gate passed without starting a service. Actual original daemon/local Worker/E2EE proof is separate from that packaging gate, and neither certifies a real phone, public relay or voice assets. E/phone-final-component-receipt/acceptance-receipt.json is the packaging receipt.


## Final checkpoint follow-up and remaining shutdown boundary

The complete folder keeps its historical9232f52 name; BUILD.json and portable-package.json identify the current binaries. Source033d61c corrects the short trajectory ledger that the actual d7df1ff capture showed centered vertically. The full-height application column now anchors rows below the timeline; geometry/copy/nested-tab regressions passed2.898s.

The d7df1ff ordinary entry booted a private Host and its managed stop request reached resume save, but protected quit did not complete. The exact inspection/confirmation branch remains uncollected: task-control can require native consent or block on unavailable coverage, and the startup marker does not imply admission readiness. Do not carry the old9232f52 graceful-exit result forward. The native computer-use capability remains unavailable, so no OS confirmation was submitted and no guard was weakened. Both owned paths/start times were checked before isolated forced cleanup; their listener closed and test data was archived. E/portable-ordinary-start-d7df1ff.json retains this partial result and the initial time-comparison guard rejection before signaling.

Final integration uses E/checkpoint-final-build.json, E/native-workspace-checkpoint-final.json/.png and E/portable-ordinary-start-checkpoint-final.json. Original pixel parity, protected OS confirmation and the full91-row feature scope remain subject to their specific gates.


## Verified final complete-folder checkpoint

Clean Windows desktop and preview source `97e04277fff09938d56759fac3edca4533f21d9a` (`vcs.modified=false`, Go1.27.2/CGO0) is installed in `../WhaleIsle-MyGo-local-9232f52/`. Unique entry is `WhaleIsle.exe`; keep the complete fixed-path directory. Desktop SHA256 `d1256f16b7c866a900607ef13f235acb8b950c4265bb9647aa7b024ce10bb845`; preview SHA256 `4a2613ab5ee5293fb78865f511f66de8a7ecabb545e3e6bfa0d50780ab5cb1b3`.

Actual final ordinary entry booted with no userdata/vendor/Node/QA overrides, from the unrelated task directory, using only a fresh owned HOME and OS Hidden startup hint. Ready marker arrived in15.082 seconds; private Node47712 used3082. This proves ordinary bootstrap/Host readiness, not OS visible-window gestures or protected graceful quit. Only this run's exact path/start-time-matched desktop17700 and Node47712 were force-cleaned;3082 closed, and private data was moved to E/portable-startup-profile-checkpoint-final. The delivered folder has no data directory.

Final production native restore passed, content1185x792DIP / PNG1778x1188 at1.5 scale. The image was viewed: short two-column trajectory starts directly below the timeline, unselected inspector is absent, compact composer retains the owned draft and file card, and the right pane has one source editor with an unsaved buffer. Saved session-pane binding, attachment, draft, layout and source mode1 were verified; workspace disk bytes stayed unchanged and model requests were0. This capture uses the own-window public MyGo API, not an original/Go desktop pixel comparison.

Evidence: E/checkpoint-final-summary.json, E/checkpoint-final-build.json, E/native-workspace-checkpoint-final.json/.png and E/portable-ordinary-start-checkpoint-final.json. Build source remains the SHA above if a later commit only records these acceptance results. Protected quit, original-window comparison and other stated91-row limits remain open; no CI/push/merge/release or user-instance replacement was performed.
