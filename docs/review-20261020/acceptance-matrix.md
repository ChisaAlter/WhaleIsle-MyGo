# MyGo鲸屿 · 代码 + 实机验收矩阵（2026-10-21）

验收原则：每条都要有**真实操作**（UIA 驱动运行实例或真实 Host RPC）+ 证据（截图/日志/测试名）。
状态：PASS / FAIL / BLOCKED（注明外部依赖）/ PENDING。
证据目录：`docs/review-20261020/evidence/acceptance/`。注：标注「代码级」的行表示已用代码 + 单元/接线测试验证，未驱动实机。

| ID | 区域 | 操作 | 预期 | 状态 | 证据/备注 |
| --- | --- | --- | --- | --- | --- |
| A1 | 会话 | 新建会话 | 出现新会话、可输入 | PASS（实机 2026-10-21） | `A1-new-session.png`：点“+ 新会话”后侧栏新增并选中“新会话”，主区回到 hero 空态，composer 可输入（随后 A2 实机发送验证） |
| A2 | 会话 | 输入并发送消息 | 消息上屏，进入运行态 | PASS（代码级，实机受 API key 阻塞） | 发送链路：composer 发送按钮 `internal/ui/composer/composer.go:564-581` 四分支（model 命令 → `OnSubmitGesture` → `OnSubmit` → `OnSend`），回车同序 `draft_input.go:198-236`；cmd 已接 `OnSubmitComposerGesture`（`cmd/whaleisle/main.go:6491`）。实机“你好”失败＝Host 无 key，已改为友好卡片：`chat/failure_notice.go:classifyTurnFailure` 识别 `no API key / 未配置 API Key / AUTH / QUOTA / INTERRUPTED`，标题「还没有配置 API Key」+ 说明 +「打开模型设置」（`chat/chat.go:249` 调用；frame `OnOpenModelSettings` → `cmd/whaleisle/main_navigation.go` 打开设置模型页），原始 Host 错误只在「详情」里（可展开、等宽可选中），不丢信息；测试 `TestClassifyTurnFailure*`、`TestTurnFailureCard*` |
| A3 | 会话 | 模型流式回复 | 增量文本、停止按钮可用 | BLOCKED | 无 DEEPSEEK_API_KEY |
| A4 | 会话 | 工具调用/权限确认 | 工具卡可展开、允许/拒绝生效 | BLOCKED | 依赖模型 |
| A5 | 会话 | 复制/编辑/反馈消息 | 入口可见、操作生效 | PASS（代码级） | 复制 `chat/message_actions.go:128`（`messageCopyAction`，剪贴板走 `c.WriteClipboard`）；示例/转发 `messageQuoteMenu`（:121）；编辑 `chat/user_message.go:158-178`（仅最新可编辑用户消息，`m.OnEditMessage` 已由 frame 接 `ComposerState.BeginEdit`，`frame/frame.go:991`）；反馈 `chat/message_actions.go:130` → `chat/feedback_editor.go:17 authoritativeFeedbackActions`（store `RateMessage/ClearMessageFeedback`，`frame/frame.go:960-965`）；分支 `:132` 走 `OnForkTurn`。无死入口 |
| A6 | 会话 | Plan/Todo/Goal 面板 | 打开、状态正确 | PASS（代码级） | Plan/Goal 面板 `frame/frame.go:1045-1105` 挂载 `chat.PlanBoardView`（仅 `PlanMode/Goal/Todos` 非空时渲染，goal 六个动作接 `Store.CreateGoal/EditGoal/CompleteGoal/ClearGoal/PauseGoal/ResumeGoal`）；Todo dock `frame/frame.go:694`（`chat.TodoPanelView`，Todos 为空自隐藏）；队列 dock `frame/frame.go:1167`（`QueueDockView`）；状态来自 `snap.PlanBoard`（Host 事件投影） |
| A7 | 会话 | 搜索/切换/归档 | 列表与过滤正确 | BLOCKED（实机） | `A7-search-sessions.png`：会话列表与切换可用（多会话、当前选中高亮）；侧栏搜索框无法被 UIA SendInput 写入，改用 UIA `set`（ValuePattern）后实例卡死（见“实机驱动记录”） |
| B1 | 文件 | 打开工作区文件树并读取文件 | 内容正确、可切换 | **PASS（实机，已修）** / PASS（代码级） | 实机复验：无工作区会话的右栏不再请求远程 mirror，显示「此会话没有工作区文件夹 / 在左侧选择或添加项目目录后，这里会显示工作区文件。」证据：`A2-failure-card.png` 右侧。旧构建的 `resolve-mirror 404` 截图保留在 `A6-plan-todo.png` 作为修复前证据。代码级证据见右侧原记录：`workspaceFiles/{list,read,readBytes,stat,changes}`（`internal/desktop/workspacefs/remote_list.go:85`、`text_page.go:49`），本地大文件分页走 `ReadFileText`；文件树本身**没有** create/rename/隐藏开关命令——与原版一致（`runtime/vendor/.../ui-files/src/client/FileTree.tsx` 只有 drag-mention），并由 `files/functional_acceptance_test.go:34,126` 反向断言不得出现这些标签。证据：`go test ./internal/ui/files ./internal/desktop/workspacefs` |
| B2 | 文件 | 编辑并保存/冲突提示 | 保存成功或冲突可见 | PASS（代码级） | 保存链完整：编辑器观察 → `SaveQueue` → `SaveTextChecked` 重读比对（`workspacefs/save_queue.go:206-246`）→ 真实落盘；冲突返回 `SaveConflictError`（"File changed on disk. Draft kept…"）并保留草稿与基线、脏关闭被拦截（`files/autosave_test.go:TestAutosaveConflictRecoveryAndBindingChange`）；远端/截断预览按 `files/autosave.go:17-35` 明确拒绝保存。**本轮修复**：终端链接打开的文件此前没有写入口（`cmd/whaleisle/terminal_target.go` 的 `files.Model` 缺 `OnSaveEdit`），现抽出 `terminalFilesModel` 接上 JoinRoot+WriteFileText，并补 `cmd/whaleisle/pass9_chain_test.go`（写盘成功 + 越界路径被拒）。**剩余差异（记 RPC 缺，不造假）**：远程工作区保存无 RPC —— dsh 只暴露读侧 `workspaceFiles/*`；远程预览草稿按 `autosave.go:29` 提示走会话远程保存，本仓库未伪造写接口 |
| B3 | 终端 | 打开终端并执行命令 | 输出正确、可中断 | PASS（代码级） | PTY 由原生 Ghostty/ConPTY 提供（`internal/ui/terminal/panel.go`、`runtime.go`），中断即终端内 Ctrl+C（emulator 键位）；`native_acceptance_windows_test.go:TestNativeConPTYLifecycle` 跑 pinned DLL + `cmd /D` 的 sentinel/cwd/resize/active-close/reopen/自然退出；分组上限 4（`workspace.go:88`），split H/V、maximize/restore、restart、多会话俱在；创建失败经 `Panel.Err` 上屏并在切换/关闭时清除（`workspace.go:57,165`）。证据：`go test ./internal/ui/terminal` |
| B4 | Git | 状态/提交/推送 | 真实仓库操作有结果或明确失败 | PASS（代码级） | 真实 git 子进程（`internal/desktop/git`：status/diff/commit/branch/publish），子进程环境固定 `GIT_TERMINAL_PROMPT=0` + 空 ASKPASS/GCM 非交互（`git/environment.go`），凭据不可用时返回明确失败而不是挂死；推送前必须 `PreparePush` 出 `PushPlan`（branch/remote/target/URL + 默认分支警告）再在应用内确认（`cmd/whaleisle/git_push_review.go`），工作区差异的暂存/取消暂存/暂存 diff/还原（含不可恢复文案）与折叠/分栏/折行在 `internal/ui/diffsurface`。证据：`go test ./internal/desktop/git ./internal/ui/diffsurface` |
| C1 | 设置 | 打开设置并切换全部 section | 每页渲染、无死控件 | PASS（代码级） | `sections.go` 计划顺序 13 项；`body_presentation_test.go` 覆盖 general/interface/appearance/pet/about × 6 状态；全仓 settings/usage/pluginmanager/market 仅 4 处字面 `Disabled(true)`，均为条件分支（凭证无 Host、桌面保存无 Host、技能无 Hub、自绘开关 disabled 参数），无永久死按钮 |
| C2 | 设置 | 通用/外观/界面修改并重启 | 值持久化 | PASS（代码级） | host 偏好经 `BodySettingsAccess.OnChange/OnPatch` → `Store.BindDescribedPreferences()` 落盘；主题经 `theme.NewEditor(...).Save()`；实机重启未验（本轮不驱动实机） |
| C3 | 设置 | 模型 provider/API key/env | 保存、检测、错误可见 | PARTIAL | 保存/取消/错误可见已实现（`model_configuration.go` + `modelNamespaceBody`，`state.Err` 上屏）。**缺口**：参考页的「未检测到环境变量」橙色提示未实现 —— 需要 store 提供 env 探测字段（BLOCKED，非本域可补） |
| C4 | 设置 | 技能/MCP/插件列表 | 真实数据、可操作 | PARTIAL | 数据均来自真实 RPC（`skills/list`、`mcpServers/list`+`Connection.Tools`、`pluginManager/listPlugins`），启停走 `setInvocation`/`setDisabled`；本轮修掉 MCP「已连接但零工具」静默空行（现在显式「没有工具」）。**缺口**：MCP **tools/call 全仓无 RPC**（只能列不能调）；插件“依赖安装”无动作（仅「已声明，但安装文件缺失」文案） |
| C5 | 设置 | 账户/关于/更新 | 真实状态、按钮有响应 | PASS（代码级） | 账户 sign-in/sign-out/authorize/cancel 均已渲染并接回调；余额来自 `accountdetails` wallets（不可用时显式「余额暂不可用」）；关于版本来自 `harnessVersionSnapshot`（本轮已由 disabled Select 改只读文本）；更新走 `downloadNativeRelease`；「打开配置文件」本轮接 `mygo.Shell.OpenPath` |
| D1 | 任务 | 定时任务创建/编辑/运行/取消 | 全链路有效 | PASS（代码级） | 创建：cmd/main.go OnNewTask → panel 创建表单，落库 schedstore.Create；编辑：OnSaveTitle/OnSavePrompt → store.UpdateTitle/UpdatePrompt（store.go:348/357）已接，失败走 notifyUser 可见；删除/刷新/历史：本轮补齐 OnDelete→store.Delete(store.go:282)、OnRefresh→store.Refresh(store.go:152)、OnHistory→store.History(store.go:248)；投递：store.DeliveryHistory(history.go:68) + panel 历史卡；运行/取消：Host 端调度（面板展示下次运行时间/启用态筛选），仓库没有 “Run now” RPC，不伪造 |
| D2 | 代理 | 子代理面板/中断 | 列表真实、可中断 | PASS（代码级） | 列表：frame 以 session.Store 快照渲染（frame.go agents.View / Store: navigation）；中断：panel.go:88/165 行内「中断」→ agents.Model.OnInterrupt，已接 session.Store.InterruptChild（store.go:1160，空 mode 落 “cancel”）；状态刷新：store 事件/快照驱动，busy 时按钮禁用 |
| D3 | 代理 | Agent 预设选择/说明 | 生效、说明可读 | PASS（代码级） | 管理器由 session store 构造（main.go session.NewAgentPresetManager）；点卡片 = settings/agent_presets.go 内 a.Manager.SaveDefault(row.ID)（已常驻 BFF）；说明来自 Host 行（Name/Description），本轮把卡片重排为参考样式且 guide 仍走真实文本 |
| E1 | 鲸鱼娘 | 设置页读写 | 名字/称呼/人设/记忆/技能生效 | PASS（代码级） | 数据层 internal/desktop/whale（catalog/settings.update/memory.replace/memory.clear/skills.toggle，11 条 httptest）；设置页 settings/pet_whale.go （名字·称呼失焦 Patch、IM、人设行内编辑、记忆对话框、技能 44×24 开关）已有 tester 断言；cmd 接线 settings_body_wiring.go:115 对齐七个回调 |
| E2 | 鲸鱼娘 | 打开聊天并发送 | 真实会话 | BLOCKED | 依赖模型 |
| E3 | 桌宠 | 显示/拖动/喂食/互动 | 窗口与状态正确 | PENDING | |
| F1 | 远程 | 网关设置/配对码/SSH | 保存、刷新、TOFU 提示 | PASS（代码级） | 网关：`settings_body.go` 单字段提交（中继/模式/端口各只提交本字段，`pass6_remote_test.go` 四例）+ `config/remote_gateway_test.go` 校验与 stale 拒绝；配对码刷新：页面「刷新」→ `OnRotateToken`（`main.go:5379`）→ `phoneremote.DeviceAction("rotate")` → 原运行时 `rotateToken`，`pass6_remote_test.go` 断言调用/错误上屏，异步 DeviceAction 的 Busy/Error/Message 也在设置页显示（`pass9_acceptance_test.go:TestGatewayRefreshShowsRuntimeActionState`）；**不假装已连接**：`pass9_acceptance_test.go:TestPhoneOfferRequiresLiveRelay` 证明 relayConnected=false 时隐藏复制/打开按钮并显示「无可用配对码」；SSH：`sshremote/known_hosts_test.go` 覆盖 TOFU/未知主机/密钥变更拒绝，`cmd/whaleisle/ssh_connect_operation_test.go` 覆盖指纹提示→Accept→重试，本地转发 `forward.go` + `tunnel_loopback_test.go`，`main.go:5573-5627` 接线。真机配对/公网中继 = BLOCKED（无手机、无 relay、无 dshd-remote 组件） |
| F2 | IM | 启用 dsh-im/查看渠道 | 插件启用、渠道真实 | BLOCKED（依赖 dsh-im 安装） | 代码级已验：`im/enable_plugin_test.go` + `main.go:5317`（pluginManager/setBundleEnabled 后刷新）；渠道/历史来自 `im/list`·`im/history`·`pluginInventory` 真实 RPC（`store/im/store_test.go`）；发送状态本轮改为真实结果：pending→sent/failed/local，失败消息不再显示成已送达（`store/im` 新增状态断言 + `ui/im/pass9_acceptance_test.go`）；空态按真实能力区分「连接 Host」与「IM 插件未就绪」。实机渠道/收发 BLOCKED：dsh-im 未安装 |
| F3 | Office | 打开 Office 配置 | 真实配置或明确依赖 | BLOCKED（依赖 dsh-im/Office 服务） | 代码级已验：`officeSettingsView` 读 `store/office` 快照（pluginrpc），配置校验/Review/Confirm 全程真实 RPC（`store/office/office_test.go`）；空态按 `HarnessReady`/`v.Available` 区分「连接 Host」「dsh-im 插件未就绪」「未读取到配置」，不再出现「原版插件页面仍可使用」；`im/office_test.go` 覆盖依赖文案与「打开插件页」。实机 Office 连接 BLOCKED：需 dsh-im 插件与 Office 服务端 |
| G1 | 视觉 | 首屏与安装版对比 | 布局/背景/字体一致 | PARTIAL | 见 `visual-acceptance.md`：7 PASS / 1 DIFF（Project 空态，本轮已补）/ 3 UNVERIFIED（composer 与 hero 被第三方窗口遮挡） |
| G2 | 视觉 | 13 个设置页快照对比 | 结构一致 | PARTIAL | 仅 models 有同源全窗截图；8 页有 ours 但未逐张开图，4 页缺 ours 截图 |
| G3 | 视觉 | 动画/玻璃/指针特效 | 无闪烁、行为一致 | PARTIAL | bloom/软边/玻璃 PASS（有测试与定格）；指针特效 UNVERIFIED（静态图无法定格） |

## 实机驱动

`powershell -NoProfile -ExecutionPolicy Bypass -File .devdata\uia.ps1 -Action <texts|invoke|set|type|click|shot> ...`
窗口按进程名 `whaleisle*` 自动定位；`type` 用 SendInput Unicode 写入自定义编辑框。

## 已知外部阻塞

- 无 `DEEPSEEK_API_KEY`：A3/A4/E2 无法真实跑模型；错误文案需友好化。
- dsh-im/Office 需要插件安装；IM 页已提供启用动作。
- 远程 F1 真机部分：无手机、无公网 relay、无 dshd-remote 组件安装；网关/配对码/TOFU/转发只做代码级验证，真机配对记 BLOCKED。

## C 域代码验收补充（Pass 9-D，2026-10-21）

逐 section 行为差异（对照 `ui-settings-*` / `ui-plugin-manager` / `ui-settings-market`，**代码级**，未驱动实机）：

| 区域 | 参考行为 | 本仓库实现 | 结论 |
| --- | --- | --- | --- |
| 账户 | 登录/登出、余额与钱包、bonus、用量入口 | `account.go`：sign-in/authorize/cancel/sign-out 全渲染；余额来自 `accountdetails` wallets，不可用显式提示；bonus 有独立钱包行 | 一致（登录态未实机验） |
| Provider / API key | 供应商卡、密钥输入、**未检测到环境变量**提示、取消/保存 | 密钥输入+保存/取消+错误上屏已实现；**env 检测提示缺**（需 store 字段） | 部分 |
| 快捷键 | 逐条编辑、冲突提示、恢复默认 | `shortcuts.go` 有编辑/恢复默认与确认弹窗；冲突提示来自 `shortcuts` 编辑器 | 一致 |
| 开发者工具 / 会话日志 | host 偏好开关 | `desktop.go` 走 `ui-settings/enabled`、`session-log-deepseek/enabled` | 一致 |
| MCP | 服务器行、状态、工具列表 | 行/状态真实；工具列表来自 `Connection.Tools`；**tools/call 无 RPC**；本轮修「零工具」静默空行 | 部分 |
| 插件 | 列表、启停、依赖提示 | 列表/启停真实（`setDisabled`）；依赖仅文案「已声明，但安装文件缺失」，**无安装动作** | 部分 |
| 市场 | 自有市场 → 改官方 | 设置侧「市场」入口已移除（task-48），能力保留在 `internal/ui/market` | 已按新方向处理 |
| 更新 / 关于 | 版本只读、在线安装徽标、打开配置/运行目录 | 版本只读文本、徽标在 `OnInstallUpdate` 时是按钮、运行目录 `OpenInFileManager`、配置文件 `Shell.OpenPath` | 一致 |
| 用量 | 导出、价格详情、范围 | 导出接 `OnExportStats("json")`；价格详情为**估算**并在文案里写明「不是账单或实时报价」；无「设置」按钮（本版本无范围对话框） | 一致 |

「像真的但其实是默认值」排查：未发现伪造数据。唯一的默认值来源是 `emptyDash`（空数据显示 `—`）与 `providerCopy` 兜底文案；用量价格表是**原版固定官方价格**且页面明确标注年份与「估算」。

## 实机驱动记录（2026-10-21，i18n-sweeper / UIA）

实例：`whaleisle-pass9`（pid 13512），窗口标题 `MyGo鲸屿`，1613×1044 @ (345,87)。

| 流程 | 实机结果 | 证据 |
| --- | --- | --- |
| A1 新建会话 | **PASS**：侧栏新增并选中“新会话”，主区回到 hero，composer 可输入 | `A1-new-session.png` |
| A2 发送消息 | 消息上屏（含时间戳/复制/编辑）→ 立即“● 运行失败”（无 API key） | `A2-send-message.png`、`A2-typed.png` |
| A5 复制 | **PASS**：UIA `invoke` 后系统剪贴板收到消息全文 | `A5-copy-message.png` |
| A5 编辑 | **PASS**：“正在重新编辑此消息”，草稿回填，可“取消” | `A5-edit-message.png` |
| A6 Plan/Todo/Goal | **BLOCKED**：无模型回合，无 plan/todo 内容；右侧面板只有“文件/预览” | `A6-plan-todo.png` |
| A7 会话列表/切换 | 列表与切换可用（搜索框无法驱动，见下） | `A7-search-sessions.png` |
| B1 文件树 | **FAIL**：`dsh-remote resolve-mirror?sessionId=session-00e8a956-…: invalid JSON (HTTP 404)`，只有“重试远程工作区解析” | `A6-plan-todo.png`（同屏右侧） |

**实例卡死（阻塞后续全部流程）**：对侧栏搜索框执行 UIA `set`（ValuePattern）抛出 “Unsupported Pattern” 后，实例在数秒内失去响应：`IsHungAppWindow=True`、`WM_NULL` 3s 超时、等待 15s 未恢复。按要求**未重启**，故 B2/B3/B4、C1–C5、D1–D3、E1/E3、F1/F2、G1–G3 均记 BLOCKED。证据：`instance-hung.txt`。建议由实例 owner 重启后按本表续跑。

**实机与代码级记录不一致处（需复验）**：
1. 运行实例是旧构建：A2 失败提示仍是原始英文 `llm-deepseek: no API key for provider route …`，没有 `classifyTurnFailure` 的“还没有配置 API Key”卡片与“打开模型设置”入口 → 需用当前代码重建后复验 A2/A3/A4。
2. 无目录（scratch）会话的文件树走 `dsh-remote resolve-mirror` 并 404（B1）；建议单独排查无工作区会话的远程解析路径。
3. 实机点击侧栏“设置”无跳转（多次重试无效），因此 C/D/E/F 域无法进入；是否与后面的卡死同源待复验。

**uia.ps1 使用经验（供后续驱动复用）**：
- 必须显式传 `-ProcessId`：脚本按窗口标题匹配的字符串在 ANSI 读取下是乱码（`*椴稿笨*`），自动定位必然失败。
- 自定义绘制按钮：`invoke` 能触发“复制/设置/新会话”，但鼠标 `click` 对它们无效；“发送”反之只有 `click` 生效。两者需试探。
- `type`（SendInput Unicode）在 composer 可用但有丢字（实测 21 字符丢 2 个，故 A2 消息体为 `aceptance A2 mesage`）；在侧栏搜索框完全不生效。
- **不要用 `set`**：搜索框上抛 Unsupported Pattern 并导致实例卡死。
- `shot` 截取窗口矩形，若被其它窗口覆盖会拍到别的内容（本次有一次拍到 Chrome）；操作前先用 `SetForegroundWindow` 置前。

## Lead 实机复验（pass12/13，UIA 驱动 + 当前构建）

| 项 | 结果 | 证据/说明 |
| --- | --- | --- |
| A1 新建会话 | PASS（实机） | `A1-new-session.png`；新会话回 hero，composer 可输入 |
| A2 发送消息/失败提示 | PASS（实机，模型调用 BLOCKED） | 发送后回合失败；当前构建显示「● 还没有配置 API Key」+「打开模型设置」+「详情」，原始英文只在详情里；`A2-failure-card.png`。无 `DEEPSEEK_API_KEY`，流式回复仍 BLOCKED |
| A2 失败卡动作 | PASS（实机） | UIA invoke「打开模型设置」→ 设置·模型页；导航无「市场」 |
| A5 复制/编辑 | PASS（实机） | 复制写入剪贴板；编辑回填草稿并可取消；`A5-copy-message.png`、`A5-edit-message.png` |
| A7 会话列表/切换 | PASS（实机） | 侧栏列表与切换可用；搜索框 SendInput 不生效，未验证过滤 |
| B1 文件树（无工作区会话） | PASS（实机，已修） | 旧构建报 `dsh-remote resolve-mirror 404`；现在右栏显示「此会话没有工作区文件夹 / 在左侧选择或添加项目目录后，这里会显示工作区文件。」`A2-failure-card.png` 右侧可见 |
| C1 设置导航 | PASS（实机） | 设置弹窗打开，左栏 通用/外观/模型/插件/技能/MCP/远程/用量/关于 渲染；「市场」已从导航移除；`C1-settings-models.png` |
| C3 模型页 | PASS（实机） | 供应商/密钥/识图/当前任务模型渲染；无 key 时提示「档案或凭证状态尚未验证，请刷新；不可据此判断模型可用。」 |
| B3 终端 / B4 Git | PASS（代码级） | 见 B3/B4 行；本轮实机未跑（桌面锁定，鼠标点击不可达） |
| D1–D3/E1/F1–F3 | 代码级结论不变 | 见对应行；实机依赖模型/插件/设备 |
| G1–G3 | PARTIAL | 见 `visual-acceptance.md`：首屏 7 PASS/1 DIFF（Project 空态本轮已补）/3 UNVERIFIED；设置页只有 models 有同源截图；指针特效 UNVERIFIED |

### 本轮据此修复

1. 失败回合（turn/end）现在复用 `paintTurnFailure`：原先只显示「运行失败」，必须展开才看到英文原因；现在标题/说明本地化，原始 Host 文本只在「详情」。
2. 失败卡的「打开模型设置」此前在聊天视图是 nil 回调（只有导航模型接了），已把 `openModelSettings` 同时接到主 `frameModel`，实机点击可达模型页。
3. 无工作区（scratch）会话的右侧文件栏不再调用远程 `resolve-mirror`，改为真实空态说明。
4. 设置左栏移除残留的「市场」入口（设置文档里的市场 section 仍可深链打开）。
5. 侧栏 Project 空态：折叠时显示「添加项目目录，开始持续对话。」，无项目时不再显示搜索/归档筛选。

### 仍未通过 / BLOCKED

- 模型流式、工具卡/权限、Plan/Todo 真实内容、鲸鱼娘聊天：无 `DEEPSEEK_API_KEY`。
- IM/Office：未安装 dsh-im；启用按钮与空态已接。
- 远程真机配对/公网 relay、桌宠多屏/DPI：需设备/桌面解锁环境。
- 设置 general/appearance/skills/market 缺 ours 真机截图；视觉像素级对比仍不完整。
### 第二轮实机（pass15，设置/插件/任务/鲸鱼娘）

| 项 | 结果 | 证据/说明 |
| --- | --- | --- |
| C1 设置 13 区 | PASS（实机） | 通用/外观/界面/模型/插件/技能/MCP/远程/Agent 预设/用量/桌宠/关于 均可进入并渲染；左侧「市场」已移除。证据：`C1-settings-models.png`（设置模型页）；关于页实机文本见本轮记录 |
| C4 插件页 | PASS（实机，带差异） | 真实 inventory：内置（子代理/网页搜索/代理循环/终端）+ 实验性卡片 + 开关；`C4-plugins.png`。差异：卡片的 Open/Enable 可访问名仍是英文 |
| C5 关于/更新 | PASS（实机，已修） | About 渲染版本 `dev`、来源、复制版本；打开即检查更新。修复前弹窗直接显示 `not_configured` + 英文内部信息；现在显示「当前构建没有配置更新源（开发版）…」；`updateCheckMessage` 测试覆盖四种状态 |
| D1 自动化任务 | PASS（实机空态） | 「暂无任务。在会话中创建的任务会显示在这里。」+「新建自动化任务」；`D1-schedule.png` |
| E1 鲸鱼娘设置 | PASS（实机） | 「鲸鱼娘 · 人设与长期记忆」「模型、IM 与技能」「她的技能 · 已启用 0 / 0」「长期记忆/MEMORY.md」「名字与称呼」；真实 Host 路径显示；`E1-whale.png` |
| B3 终端 / E3 桌宠拖动 | BLOCKED（实机） | 桌面处于锁屏，鼠标点击/E2E 拖动不可达；面板菜单按钮不支持 InvokePattern。终端 PTY 与桌宠状态仍只有代码级/单测证据 |

本轮修复：关于页自动检查更新的弹窗不再泄露 `not_configured`/内部英文，改为产品文案（`cmd/whaleisle/update_check_lifecycle.go`，`update_check_message_test.go`）。
### 实机发现的 P0 崩溃（已修）

打开 设置 → 远程 → 返回 时应用进程 panic 退出：

```
panic: ui: a ListState shown by two Lists
chat.View → ui.List(...).Rows
settings.ModalView → SectionContent → paintMainContent("")
```

根因：`settingsSectionModel` 先把 `ActiveSection=remote` 和 SectionContent 装到模型上，但回调执行时又去读**实时**的 `settingsEmbeddedSection`；`OnClose` 已把它清空，于是内容区用 `""` 重画了主聊天，与页面自身的 `ChatList` 同帧复用，触发框架的 ListState 唯一性保护。

修复：构建模型时捕获 section，传给回调使用；新增 `TestEmbeddedSectionContentKeepsTheSectionItWasBuiltFor`。实机复验：设置→远程→返回，进程存活并回到会话（pass16）。
### 第三轮实机（pass17/18）

| 项 | 结果 | 证据/说明 |
| --- | --- | --- |
| B3 终端抽屉 | PASS（实机，PTY 已起） | 「切换终端抽屉」打开 PowerShell 1，提示符 `PS C:\AI\WhaleIsle-MyGo\.devdata\dsh-home\no-workspace>`；关闭/新终端/分栏/最大化菜单真实；`B3-terminal.png`。命令输入受桌面锁屏限制，命令执行仍以 ConPTY 测试为准 |
| B4 Git 入口 | PARTIAL（实机） | 顶栏「切换分支」「推送」「Git 操作选项」存在；无仓库会话打开分支筛选显示「没有匹配分支」，不伪造分支 |
| C5 关于页 | PASS（实机，已改） | 参考版打开关于**不弹更新框**；移除了 mount 时自动检查，保留显式「检查更新」按钮；测试改为断言初始不调用、点击后才调用 |
| C4 插件页 | PASS（实机，已修样式） | 「添加插件」旁的 ⌄ 触发器此前用 PrimaryButton，渲染成独立蓝色方块；改为普通按钮，保持分段按钮观感 |

本轮修复：About 不再自动弹更新对话框；插件页添加按钮分段样式；插件卡可访问名本地化（打开/启用）。