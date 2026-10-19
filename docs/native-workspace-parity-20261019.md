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
