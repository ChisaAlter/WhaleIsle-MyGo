# MyGo 鲸屿 · 死控件与假数据全量审计（Pass 7-A）

- 审计日期：2026-10-21
- 范围：`internal/ui/**`（646 个非测试文件）、`cmd/whaleisle` 接线、`internal/desktop` 数据源、`internal/dsh` API
- 方式：只读静态扫描 + 逐条读代码确认（未改任何代码，未跑 git 写命令）
- 原始统计：`.Disabled(` 共 **1118** 处；字面量 `Disabled(true)` **25** 处；`Disabled(... == nil)` 形式 **147** 处；`Clicked() && m.OnX != nil`（看起来可点、回调 nil 时静默无效）**191** 处。

> 结论先说：真正"永久灰掉/点了没反应"的控件数量不大（下面 A1–A3 合计约 20 条），根因集中在三处：
> **(1) 少数回调在 cmd/frame 从未接线**；**(2) 部分整块 UI 定义了却从未挂载**（死视图）；**(3) 远程 / IM / 鲸鱼娘 / 市场 / WebView 等页面本身仍是 stub，数据源未接**。
> 其余 1000+ 处 `Disabled` 都是正常的可用性门控（空输入、busy、只读、未连接）。

---

## A. 死控件清单

### A1 永久禁用：回调从未接线（用户能看见，点了没反应/永远灰）

| # | 位置 | 界面位置 / 文案 | 原因 | 参考版 | 修复需要 |
|---|---|---|---|---|---|
| A1-1 | `internal/ui/settings/settings.go:122`（tooltip `:124`） | 设置弹窗右上「打开配置文件」outline 小按钮，灰显，hover 提示「此构建尚未接入配置文件打开」 | ① 回调 nil：`Disabled(m.OnOpenConfig == nil)`，`OnOpenConfig` 在 `cmd/whaleisle`、`internal/ui/frame` 里**从未赋值**（全仓仅 `settings` 包内出现） | 原版可点（打开 config.json） | 新增 cmd handler：`internal/paths` 加 `ConfigFile()`（现只有 `DshHome()`/`UserDataDir()`），打开用现成的 `mygo.Shell.OpenExternal`；或改为在设置内嵌编辑器 |
| A1-2 | `internal/ui/schedule/schedule.go:126`、`internal/ui/schedule/schedule.go:134` | 调度任务编辑态「保存标题」「保存提示词」按钮，**整块不渲染**（`if m.OnSaveTitle != nil && ...`） | ① 回调 nil：`OnSaveTitle` / `OnSavePrompt` 从未赋值 | 原版可点 | 直接接线：`internal/store/schedule` 已有 `(*Store).UpdateTitle(ctx, sessionID, id, title)`（`store.go:348`）与 `UpdatePrompt`（`store.go:357`）；在 `cmd/whaleisle/schedule_navigation.go` 里补 `OnSaveTitle/OnSavePrompt` |
| A1-3 | `internal/ui/agents/panel.go:84`、`:163` | 子代理面板行内「中断」按钮，**整块不渲染**（`runningN > 0 && m.OnInterrupt != nil`） | ① 回调 nil：`OnInterrupt` 从未赋值 | 原版可点 | 接线：`internal/store/session` 已有 `(*Store).InterruptChild(ctx, parentID, childID, mode)`（`store.go:1160`） |
| A1-4 | `internal/ui/browserpreview/panel.go:75` | 浏览器预览面板「在外部打开」，**不渲染** | ① 回调 nil：`OnOpenExternal` 从未赋值 | 原版可点 | 接线：`mygo.Shell.OpenExternal(url)`（`cmd/whaleisle/main.go:5062` 已有同样调用） |
| A1-5 | `internal/ui/chat/tool_lifecycle.go:89`（数据来源 `chat/chat.go:296`） | 工具卡「检查此调用」按钮，**不渲染** | ① 回调 nil：`OnInspectToolCall` 从未赋值；`chat.Model.OnInspectCall` → `ToolImageGalleryOptions.OnInspectCall` 全链路已就绪 | 原版可点（打开调用详情） | 接到 frame 的 session inspector：`frame/session_inspector_pane.go` 已有 `InspectorTarget`/`prepareSessionInspector`，需要暴露"按 CallID 打开"入口 |

### A2 硬编码 / 永久占位禁用（`Disabled(true)`，无任何状态能让它变可点）

| # | 位置 | 界面位置 / 文案 | 原因 | 参考版 | 修复需要 |
|---|---|---|---|---|---|
| A2-1 | `internal/ui/settings/mcp_catalog.go:121` | 设置 · MCP · 目录搜索框（placeholder「搜索名称、ID、命令或 URL」） | ④ 永久占位：`ui.TextInput(...).Disabled(true)`，无 tooltip、无筛选实现 | 原版可输入 | 本地过滤：目录行已在内存（`mcpCatalogRows`），加 `query` 过滤即可，无需 RPC |
| A2-2 | `internal/ui/settings/mcp_catalog.go:123` | 设置 · MCP · 筛选下拉（全部/已启用/已禁用） | ④ 永久占位：`ui.Select(...).Disabled(true)` | 原版可切 | 同上，纯本地状态过滤 |
| A2-3 | `internal/ui/pluginmanager/view.go:915` | 插件库卡片角标「不兼容」被做成 disabled 按钮 | ④ 语义错误：状态徽章做成了按钮 | 原版是文本徽章 | 改成 `Badge`/`Text`，不做成按钮（属于样式修复，非功能） |
| A2-4 | `internal/ui/terminal/panel.go:216` | 终端右键菜单「快捷键」项，永久灰 | ④ 占位：`menu.Item(...).Disabled(true)` | 原版可点（打开快捷键设置） | 打开设置 → 快捷键页（`cmd/whaleisle/settings_shortcuts.go` 已有页面） |
| A2-5 | `internal/ui/remote/settings_body.go:404` | 远程设置某段 segmented 控件，常灰 + 0.5 透明度 | ③/④：取决于远端配置字段，当前恒为假 | 待按原版核对 | 复核该字段来源（`remote` store 是否有对应 RPC）后再决定接还是删 |
| A2-6 | `internal/ui/settings/about.go:76` | 关于页版本号「下拉」展示 | ④ 展示型控件误用 Select | 原版是文本 | 改文本，不做成下拉 |

> 说明：`internal/ui/chrome/gitcommit.go:36` 的 commit 按钮在 `commitEnabled=false` 时灰显，属**合理门控**（无改动/非 Git 仓库），但视觉上是「死按钮」，建议 hover 给原因（已有 `CommitLabel` 机制可扩展）。

### A3 回调 nil 时整块消失（不是"灰"，是"没有"——用户感知为功能缺失）

| # | 位置 | 缺失的界面元素 | 原因 | 参考版 | 修复需要 |
|---|---|---|---|---|---|
| A3-1 | `internal/ui/settings/skills.go:100-104` | 技能页「浏览技能库」按钮：`HubQuery==nil || OnSearchHub==nil` 时灰 + tooltip「此构建未接入技能搜索」 | ② 取决于接线：cmd 已赋值 `OnSearchHub`/`HubQuery`，**未连接 Host 或 hub 未就绪时才会灰** → 属合理门控，但 tooltip 文案会被误读为"未实现" | 原版可点 | 文案改为「需要连接 Host」；确认 hub RPC 在未连接时不报错 |
| A3-2 | `internal/ui/settings/settings.go:30` 注释 + `:122` | 同上 A1-1 | ① | — | 同 A1-1 |
| A3-3 | `internal/ui/remote/remote.go:343` | 远程 · SSH 段落状态列恒为「未实现（无 SSH 探测）」 | ④ 永久占位文案（不是控件） | 原版有实时探测 | 接 SSH 探测（`internal/ssh`? 目前无）或删掉该列，只保留配置就绪检查 |
| A3-4 | `internal/ui/remote/remote.go:204` | 远程 · SSH 段落标题「SSH 远程工作区（配置就绪检查；无实时 SSH 隧道）」 | ④ 自述 stub | — | 与 A3-3 一并处理 |
| A3-5 | `internal/ui/im/office.go:54` | IM · Office 页「尚未读取配置。请连接 Host 并刷新；原版插件页面仍可使用。」 | ② 数据源未接（office 配置 RPC 未接） | 原版可编辑 | 接 `internal/store/office` 的读接口；当前 store 已有 `office` 包，需补 cmd 侧 wiring |
| A3-6 | `internal/ui/im/im.go:95`（`KeyIMStubNote`） | IM 页顶部自述「刷新时探测 im/list·im/history·im/send；无响应则查 pluginInventory」 | ② 半接：RPC 探测式，插件不响应即空 | 原版插件页完整 | 需要 dsh-im 插件 RPC 或改为「打开插件页」入口 |
| A3-7 | `internal/ui/settings/pet_whale.go:42` | 设置 · 鲸鱼娘「Host 尚未提供鲸鱼娘设置。」 | ② Host 无对应 RPC（`pluginrpc.Whale*`） | 原版有 | 需要 Host 侧 whale settings RPC；短期应隐藏整个区块而不是显示占位 |
| A3-8 | `internal/ui/webview/panel.go:74` | WebView 面板正文「可在本窗口上方覆盖无边框 WebView2 子窗口…」 | ④ 自述 stub | 原版是真实网页 | 已有 `webview` 宿主能力则接真实加载；否则隐藏面板 |
| A3-9 | `internal/ui/usage/usage.go:114` | 用量页空态「尚未加载」 | ② 数据未拉取时的默认态 | 原版加载中/有数据 | 若 `usage` store 的 `Refresh` 未在页面打开时触发，需补触发（store 已有 RPC） |

### A4 死视图：函数已实现但从未挂载（用户永远看不到）

> 判定方法：全仓（含测试）对该函数的引用数为 0（仅定义本身）。

| # | 位置 | 功能 | 现状 |
|---|---|---|---|
| A4-1 | `internal/ui/chat/feedback.go:12-79` | 旧「消息反馈行」（👍/👎 + 不喜欢原因） | 定义 / 模型 / 渲染函数都在，**从不调用**；实际生效的是 `chat/feedback_editor.go:17 authoritativeFeedbackActions`（由 `message_actions.go:130` 挂载）。→ 删除旧实现，避免误判"反馈没接" |
| A4-2 | `internal/ui/frame/frame.go:1491 paintRightTab` | 右栏 tab 条 | 从不调用；右栏切换实际走 `workspace_resource_menu.go` / `workspace_panes.go`。→ 删除或补回 tab 条（若原版有 tab） |
| A4-3 | `internal/ui/files/source_actions.go:110 paintSourceActions` | 源码预览动作条（选区引用到对话等） | 从不调用；实际动作条是 `files/preview_toolbar.go:114 paintPreviewToolbar`。→ 删除 |
| A4-4 | `internal/ui/files/references.go:13 paintReferenceButtons` | 文件「引用到对话」按钮组 | 同上（被 preview_toolbar 取代）→ 删除或确认功能是否丢失 |
| A4-5 | `internal/ui/settings/account_details.go:11 accountDetailsPane` | 账户详情面板 | 从不调用；账户页走 `settings/account.go` → 需确认原版「账户详情」入口是否存在 |
| A4-6 | `internal/ui/projects/filter.go:25 projectFilterControls` | 项目页搜索/筛选输入框（`TextInput` + 「清除搜索」） | 从不调用；**过滤逻辑本身是活的**（`projects/sidebar.go:22` 调用 `filteredProjects(m.Listing.Projects, m.SearchQuery, m.History)`），但没有任何输入框能改 `SearchQuery` → 用户没有项目搜索入口 |
| A4-7 | `internal/ui/remote/file_actions.go:23 fileRowActions` | 远程文件行操作（旧实现） | 从不调用；**功能已由 `remote/files.go:85-137` 的新动作条覆盖**（重新读取 / 编辑 / 保存到远端 / 下载到会话镜像）→ 仅需删除旧代码，不是功能缺失 |
| A4-8 | `internal/ui/settings/mcp.go:140 mcpRowDetail`、`internal/ui/settings/mcp_catalog.go:65 mcpCatalogDetails` | MCP 服务器行详情面板 | 从不调用 → MCP 行点开无详情 |
| A4-9 | `internal/ui/assistant/panel.go:181 historyBlock` | 助手历史块 | 从不调用 → 助手面板无历史列表 |
| A4-10 | `internal/ui/diffsurface/repository_actions.go:231 reviewCommitPush`、`:254 confirmReviewedCommit` | 提交/推送前的复核流 | 从不调用 → 复核确认流未挂载（相关 UI 可能只走了简化路径） |
| A4-11 | `internal/ui/chat/chat.go:499 showMessageEdit` | 消息编辑入口 | 仅测试引用 → 生产路径无"编辑已发送消息"入口（`OnEditMessage` 已在 frame 接线，`EditingSeq` 也在，缺的是按钮） |

### A5 已排除：合理门控（抽样说明，避免误判）

| 位置 | 控件 | 门控条件 | 判定 |
|---|---|---|---|
| `internal/ui/composer/composer.go:560` | 发送按钮 | 空输入 / sending | 合理 |
| `internal/ui/interaction/question_options.go:36` | 选项卡片 | 已提交 / sending | 合理 |
| `internal/ui/settings/model_configuration.go:39-369`（16 处） | 模型配置全部输入 | `!s.Writable || s.Saving` | 合理（Host 只读时确实不能改） |
| `internal/ui/settings/body_preferences.go:180-232` | 通用设置字段 | `!ClientReady || !Writable || Loading` | 合理 |
| `internal/ui/pluginmanager/view.go:156-1017` | 插件页按钮 | Loading/Busy/未连接/只读源 | 合理 |
| `internal/ui/chat/feedback.go:83,99,119` | 反馈按钮 | 提交中 | 合理（但该行是死视图，见 A4-1） |

---

## B. 假数据 / 占位内容清单

| # | 位置 | 展示内容 | 真实数据是否可得 | 建议 |
|---|---|---|---|---|
| B-1 | `internal/ui/usage/pricing.go:41` | 固定「原版官方价格表（2026-08-17）」，CNY / 百万 token，用于费用估算 | 无定价 RPC；只有本地自定义价格 + 内置表 | **保留但标注来源与日期**（文案已有"估算不是账单"）；若 Host 后续提供计价 RPC 再接 |
| B-2 | `internal/ui/settings/agent_preset_guide.go:103-148` | 「预置模式」页 5 段中英示例提示词（含"修复一个问题""批量检查配置文件""添加一个界面"长文） | 官方文案，非用户数据 | 属设计内容，建议挪到 i18n catalog 或标记为示例，避免被当成真实任务 |
| B-3 | `internal/ui/launcher/panel.go:726` | 组件行徽章「示例 / Sample」（`row.Source == "bundled"`） | 真实（来源标记） | 文案易误解为假数据，建议改为「内置 / Bundled」 |
| B-4 | `internal/ui/pluginmanager/artwork.go:64` | 插件艺术图占位「预览暂不可用。」 | 预览需远端素材或本地缓存 | 改成骨架占位 + 重试按钮（现在点了没后续） |
| B-5 | `internal/ui/theme/editor.go:34` | 状态串 `"预览中，尚未保存 / Preview, not saved"`（中英同串） | — | 违反 i18n 规范，改为双语 catalog key |
| B-6 | `internal/ui/i18n/i18n.go:1078-1079` | `KeyFrameRightStub`「右栏（M7：文件 / Git）」、`KeyFrameBottomStub`「底栏（M7：终端）」 | 已过时（右栏已实现） | 未渲染但仍在目录 → 删除死 key |
| B-7 | `internal/ui/i18n/i18n.go:1594,1715,1770,1795` | `KeyRemoteStubNote` / `KeyRemotePhoneQRPlaceholder` / `KeyIMConnectionPlaceholder` / `KeyMarketStubNote` 等"本页是 stub"自述文案 | 部分未渲染 | 删除未渲染的，保留的改成面向用户的功能说明（不要暴露"stub/未接入"给用户） |
| B-8 | `internal/ui/browserpreview/tool_state.go:17` | `browserDevicePresets` 设备预设表 | 合法静态配置 | 保留 |
| B-9 | `internal/ui/chat/feedback_dialog.go:14` | `feedbackCategories` 反馈分类（task-result 等） | 合法静态枚举 | 保留（但需确认与 Host 侧分类一致） |

> 未发现 `internal/ui` / `cmd` 里存在 fixture / mock / 张三式假数据；产品面上的"假"主要来自上面这些 **stub 页面**（远程 SSH、IM/Office、鲸鱼娘、WebView、市场自述）与**占位文案**。

---

## C. 优先级

### P0（用户高频页面：设置通用 / 对话 / 调度 / 子代理 / 远程 / IM / 鲸鱼娘 / 用量）

| 序 | 条目 | 影响 |
|---|---|---|
| P0-1 | A1-1 设置「打开配置文件」永久灰 | 设置页第一眼可见的失效按钮 |
| P0-2 | A1-2 调度「保存标题 / 保存提示词」缺失 | 调度任务无法改名/改提示词（store 已具备能力，纯接线） |
| P0-3 | A1-3 子代理「中断」缺失 | 无法中止跑飞的子代理（store 已具备能力，纯接线） |
| P0-4 | A3-3 / A3-4 远程 SSH「未实现（无 SSH 探测）」 | 远程页核心承诺缺失 |
| P0-5 | A3-5 / A3-6 IM 与 Office 配置页空 + 自述 stub | IM 页要点开才有数据 |
| P0-6 | A3-7 鲸鱼娘「Host 尚未提供鲸鱼娘设置」 | 设置页主推页面为空 |
| P0-7 | A3-9 用量页默认「尚未加载」 | 用量页首屏空态（若无自动刷新） |
| P0-8 | A4-6 项目搜索框未挂载（过滤逻辑已在，缺输入框） | 项目页无法搜索标题（高频操作） |
| P0-9 | A2-1 / A2-2 MCP 目录搜索与筛选永久禁用（设置页高频） | 搜索框和筛选下拉永远灰 |
| P0-10 | A1-5 工具卡「检查此调用」缺失 | 对话里调试工具调用的入口消失 |

### P1（其它页面 / 清理项）

| 序 | 条目 |
|---|---|
| P1-1 | A4-7 远程文件行操作旧实现（新实现已覆盖，删旧代码） |
| P1-2 | A2-4 终端菜单「快捷键」永久灰 |
| P1-3 | A2-3 插件「不兼容」做成灰按钮（应改徽章） |
| P1-4 | A2-6 关于页版本做成禁用下拉（应改文本） |
| P1-5 | A3-1 技能「浏览技能库」tooltip 误导（「未接入」实为未连接） |
| P1-6 | A1-4 浏览器预览「在外部打开」缺失 |
| P1-7 | A4-1/2/3/4/5/7/8/9/10/11 死视图清理（避免后续误判/误导） |
| P1-8 | A2-5 远程设置段控件常灰待复核 |
| P1-9 | B-3/B-4/B-5/B-6/B-7 文案与占位清理 |

---

## D. 修复顺序建议（按成本/收益）

1. **纯接线即可修（各 1–5 行）**：A1-2 调度保存（`store.UpdateTitle/UpdatePrompt` 已有）、A1-3 子代理中断（`store.InterruptChild` 已有）、A1-4 外部打开（`mygo.Shell.OpenExternal` 已有）、A1-1 打开配置文件（`paths` 补一个路径函数即可）。
2. **本地状态即可修**：A2-1/A2-2 MCP 搜索与筛选（内存过滤）。
3. **需要新入口/新 RPC**：A1-5 工具调用详情（inspector 按 CallID 打开）、A3-3 SSH 探测、A3-5 Office 配置读写、A3-7 鲸鱼娘 RPC。
4. **删/改文案即可**：A4 死视图、B-6/B-7 死 key、A2-3/A2-6 误用控件、A3-1 误导 tooltip。

---

## E. 待复核（本轮未完全确认，留给下一轮）

- `internal/ui/remote/settings_body.go:404` 的常灰 segment 对应哪个配置字段、原版行为如何。
- `internal/ui/projects/filter.go` 的筛选在原版是行内筛选还是顶部筛选（决定 A4-6 是"删除"还是"挂回"）。
- `internal/ui/market/*` 的市场数据来源是否为真实 RPC（`internal/store/market`）还是静态样例——本轮只确认了 `KeyMarketStubNote` 未被渲染。
- `internal/ui/frame/frame.go:1491 paintRightTab` 是否对应原版右栏 tab 条（若原版有 tab，该项从"删除"改为"补挂载"）。
