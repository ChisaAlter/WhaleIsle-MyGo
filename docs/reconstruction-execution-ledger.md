# 鲸屿MyGo固定执行总表（2026-10-16）

这是执行清单，不重新审查或重编号。按固定编号并发执行，协调共享文件，每项推送后单项汇报。

## 固定范围与计数

完整能力基线：鲸屿MyGo功能对照-20261016.json / .txt。主目标265项：代码链已实现226、缺失10、部分实现26、证据不足3；另有平台4项（缺失3、部分实现1）。这些数字保持原始快照，不是当前进度百分比。
本执行表46项：B01–B36 原版缺口36项＋A01新增发现1项＋E01–E03澄清3项＋P01–P04平台4项＋N01–N02新增需求2项。另列 V01–V06 后置门槛6项。
B项当前状态：代码已推送13项、实现中4项（B02、B03、B04、B06）、待实现19项，待推送0项；合计23项尚未关闭。A01本地已写但上传阻塞。
基线按可见用户能力归组，不宣称穷尽隐藏插件入口、所有导出RPC或隐藏CLI。226项源码调用链已实现也不等于运行验收通过。

## 执行顺序

1. 固定 B01–B36 保留基线出现顺序和原始 ID，不改写基线状态以冒充运行完成。
2. 已写模块 B13、B31、B32、B35、B36 均已推送；A01 仍待解除既有特定上传动作阻塞，不计发布完成。
3. 按固定编号并发执行，当前 B02、B03、B04、B06 四项实现中；共享文件协调修改，避免覆盖其他模块。每项完成并确认推送后立即单项汇报，不等待整批。代码已推送项不因后置实机门槛未做而反复重写。
4. 单项关闭须满足具体行为准则并登记提交和验证范围；代码推送不等于 GUI/真实服务验收完成。
5. 新发现只追加新编号，不能替换或重排 B 基线；A01 与 B05 分开。E 为澄清/证据项，P 为平台项，N 为新增需求。
6. 逐项实现先行，V01–V06 在功能完成后进行最终验证和组装；不提前发起泛化审查/测试循环。

## 完整执行项

### B01 · chat.batch · 会话多选与批量归档/删除
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：可进入/退出会话多选，选中可见项；批量归档/删除有确认、进度及逐项失败汇总，保留失败项的可重试状态。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：原版实际挂载多选模式、选中可见项、批量确认/进度/逐项失败汇总。Native侧栏Model与行操作只接受单个sessionID，主框架也只有单项回调；未找到多选状态或批量操作流程。归档和删除共享多选renderer，本项合并计数。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-workspace/src/client/rows/SessionSelection.tsx:217；vendor/deepseek-harness/packages/client/ui-workspace/src/client/rows/SessionSelection.tsx:236；vendor/deepseek-harness/packages/client/ui-workspace/src/client/rows/WorkspaceBrowser.tsx:1545
- 原生证据：internal/ui/sidebar/sidebar.go:24；internal/ui/sidebar/sidebar.go:356；internal/ui/frame/frame.go:507

### B02 · chat.stop · 停止当前响应（含键盘入口）
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：在适用输入焦点和进行中响应状态下，双 Esc 序列触发当前会话 session/cancel；单 Esc 的编辑/候选关闭行为不被误触发停止覆盖。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：停止按钮已接session/cancel。原版双Esc stop序列未在native输入、frame或应用快捷键路由中实现；Escape仅取消编辑/模型命令，不应把整项停止功能算missing。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-conversation/src/client/apply.ts:417；vendor/deepseek-harness/packages/client/ui-conversation/src/client/stop-sequence.ts:1
- 原生证据：internal/ui/frame/frame.go:772；internal/store/session/store.go:972；internal/ui/composer/draft_input.go:124；internal/shortcuts/shortcuts.go:18

### B03 · chat.completion · 斜杠/@候选键盘选择与关闭
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：斜杠/@候选打开时 Enter 采纳当前高亮项；Esc 或 Shift+Tab 关闭且不改草稿，并保持关闭状态直至重新触发，不能直接发送消息。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：Native有上下键、Tab及点击选取；原版Enter采纳当前高亮以及Esc/Shift+Tab关闭不改草稿的仲裁缺失。Native Enter继续走submit，suggestOpen每帧从文本重算且无dismiss状态。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-input-trigger/src/client/controller.ts:264；vendor/deepseek-harness/packages/client/ui-input-trigger/src/client/controller.ts:277
- 原生证据：internal/ui/composer/composer.go:938；internal/ui/composer/draft_input.go:124；internal/ui/composer/draft_input.go:159

### B04 · chat.approval · 工具审批、参数展示、允许/拒绝与快捷键
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：审批可操作且焦点不在编辑区域时，Enter 允许、Esc 拒绝当前审批；沿用审批目标校验、忙碌与失败重试状态，不能误作用于别的会话或输入。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：Waterfall审批和允许一次/拒绝/失败重试已真实接通，参数可复制。原版非编辑焦点区域Enter允许、Esc拒绝的审批专用键盘处理缺失；native只找到按钮ResolveAllow/ResolveDeny调用，没有相应键盘路由。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-approval/src/client/index.ts:93；vendor/deepseek-harness/packages/client/ui-approval/src/client/ApprovalPanel.tsx:59；vendor/deepseek-harness/packages/client/ui-approval/src/client/ApprovalPanel.tsx:70
- 原生证据：internal/ui/frame/frame.go:1058；internal/ui/interaction/card.go:83；internal/store/interaction/store.go:139；internal/store/interaction/store.go:466

### B05 · chat.questions · 交互问答、补答、回看与逐题跳过
- 基线状态：部分实现
- 当前状态：代码已推送；实机验收待后置
- 完成准则：每题可明确 Skip，提交该题 selected:[]；允许其他题正常作答并跳过剩余题，补答与回看保留跳过语义。推荐默认选项另见 A01。
- 提交：e565c754209ed0b3fd61701cc19c5f379a5527dd
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：单/多选、自填、隐藏重开、超时继续后的RPC补答及历史回看均有实现。但原版可逐题Skip并提交selected:[]；native answerValid强制每题至少选择/自填，没有skip状态或入口，不能部分作答后跳过其余题。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。 此提交仅覆盖逐题跳过，不覆盖推荐默认项 A01。
- 原版证据：vendor/deepseek-harness/packages/client/ui-user-questions/src/client/index.ts:382；vendor/deepseek-harness/packages/client/ui-user-questions/src/client/QuestionComposer.tsx:343；vendor/deepseek-harness/packages/client/ui-user-questions/src/client/QuestionComposer.tsx:422
- 原生证据：internal/ui/interaction/card.go:169；internal/store/interaction/questions.go:99；internal/store/interaction/projected_questions.go:246；internal/store/interaction/question_history.go:35

### B06 · chat.tool-file-links · 从工具结果直接打开文件并定位行号
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：read 工具结果提供可点击路径并定位实际起始行；edit/write 提供打开目标文件入口，工具卡回调贯通到正确 Host/工作区文件资源。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：原版read结果可点文件路径并定位调用的起始行，edit/write也有直接打开文件入口。Native工具通用参数/结果可完整阅读，但ToolImageGalleryOptions只传图片/URL回调，没有文件路径/行号打开通路；工具卡未接这些文件定位动作。原生通用展示本身不算缺失，此项只记录缺少交互动作，文件dock其它入口不能替代。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-tool/src/client/apply.ts:46；vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/toolviews/read-row.tsx:15；vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/components/ToolRow.tsx:192
- 原生证据：internal/ui/chat/tool_lifecycle.go:16；internal/ui/chat/tool_lifecycle.go:70；internal/ui/chat/tool_images.go:13

### B07 · chat.retry · 模型重试状态链与等待倒计时
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：llm/retry 与 retry-started 按 retryId 关联，显示次数、最大次数和等待倒计时，正确进入开始或取消状态，不把旧重试显示为仍等待。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：原版llm/retry及retry-started按retryId关联，展示次数/最大次数/等待倒计时/开始或取消状态。Native这些事件仅被时间或用量计算读取，通用extractTextish不会构造该状态，chat渲染链无对应重试卡；Host仍可能实际重试，缺的是用户可见状态能力。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/register.ts:31；vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/retry.ts:41；vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/MessageItem.tsx:42
- 原生证据：internal/store/session/classify.go:155；internal/store/session/follow.go:283；internal/store/session/turn_usage.go:193

### B08 · chat.max-tokens · 输出达到token上限的明确终止提示
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：turn/end reason=max-tokens 显示输出达到 token 上限的专用终止说明，历史与流式结束后均不再泛化为正常完成。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：原版turn/end reason=max-tokens有专用说明。Native保留turn/end但completedTurnStatus未处理max-tokens，进入默认Completed，用户看不到被上限截断这一终止原因。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/register.ts:33；vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/turn-max-tokens.ts:41；vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/MessageItem.tsx:440
- 原生证据：internal/store/session/follow.go:259；internal/ui/chat/chat.go:163；internal/ui/chat/performance_usage.go:13

### B09 · chat.terminal-error · 终止错误原因、错误码与特殊中止说明
- 基线状态：部分实现
- 当前状态：代码已推送；实机验收待后置
- 完成准则：turn/end 的 error.message、error.code 及 signed-out 等特殊 aborted 原因可被用户看到，历史终止卡保留这些原因。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：Native能显示error/failed状态，但turn/end分支只输出统一失败标签，用量后即return，不展示reason.error.message/code；aborted hook如signed-out亦未细分。原版终止错误renderer会显示具体原因和错误码。另有网络/prompt报错横幅，不补足历史turn/end错误。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/turn-error.ts:32；vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/MessageItem.tsx:435
- 原生证据：internal/store/session/follow.go:259；internal/ui/chat/chat.go:163；internal/ui/chat/performance_usage.go:13

### B10 · chat.system-prompt · 模型可见系统提示词与更新记录披露
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：从 system/message 和 request/header 组合展示实际有效模型系统提示词，反映替换/更新记录及 series 锚点，不能仅显示原始零散文本。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：Native对有text/message字段的system事件可通用显示，也有原始事件检查器；但未实现system/message与request/header联合生成完整有效prompt、替换/更新及series锚点披露。仅通用文本不等价于原版有效模型请求提示词视图。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/request-prompt.ts:63；vendor/deepseek-harness/packages/client/ui-chat/src/client/conversation-nodes/request-prompt.ts:95；vendor/deepseek-harness/packages/client/ui-chat/src/client/chat/SystemPromptRow.tsx:25
- 原生证据：internal/store/session/classify.go:155；internal/store/session/follow.go:283；internal/ui/chat/chat.go:248

### B11 · chat.goal · 会话目标创建/编辑/暂停/恢复/清除与激活状态
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：目标状态独立保留 activation 与 blockedReason；active+disarmed 呈现 Resume 并接恢复动作，正确区分暂停/激活及阻塞原因。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：原版goal dock及/goal有真实注册；native PlanBoard目标区在frame挂载，goals/create/edit/pause/resume/clear真实RPC均存在，通用/goal执行也在。但目标条未保留独立activation与blockedReason：原版active+disarmed显示Resume，native Phase=active时一律显示Pause，goals/get返回的activation被有值phase覆盖。缺的是解除激活后的直接恢复入口/正确激活状态及阻塞原因提示，不能说目标功能或所有恢复渠道缺失。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-goal/src/client/index.ts:68；vendor/deepseek-harness/packages/client/ui-goal/src/client/index.ts:92；vendor/deepseek-harness/packages/client/ui-goal/src/client/GoalBar.tsx:132；vendor/deepseek-harness/packages/goal/command-goal/src/index.ts:191
- 原生证据：internal/ui/frame/frame.go:1014；internal/ui/chat/planboard.go:185；internal/ui/chat/planboard.go:224；internal/store/session/goals.go:193；internal/store/session/planboard.go:21

### B12 · workspace.tree-auto-refresh · Host文件树目录变更自动刷新及开关
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：提供 Host 文件树自动刷新开关；启用后订阅目录 changes 并使缓存失效/刷新，关闭、换 Host/工作区及销毁时终止旧订阅。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：原版真实UI开关setAutoRefresh且createWatch建立workspaceFiles.changes流。native DirectoryTree.Get命中缓存直接返回，只在手工Refresh/Cancel/binding变化清缓存，没有目录watch或自动刷新控制。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-files/src/client/index.ts:84-91；vendor/deepseek-harness/packages/client/ui-sidebar-files/src/client/FilesBody.tsx:190-221；vendor/deepseek-harness/packages/client/ui-sidebar-files/src/client/face.ts:44-55,184-199
- 原生证据：internal/desktop/workspacefs/tree.go:20-57；internal/ui/files/files.go:169-176

### B13 · workspace.dock-move-tabs · 通用资源多页签分组及跨面板移动/重排
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：任意资源可在同一 pane 形成多页签，可组内排序、跨 pane 移动并切换活动项；迁移保留资源身份和状态，不限文件标签。
- 提交：c316c973134b107b76992f496ea79c78f425e4e4
- 验证范围：TestWorkspaceDockTabsState 已通过（0.019s）；未跑全库，未作真实 Windows GUI/真实账户集成验收。
- 基线说明：原版TabStrip真实drag源、placeTab/dropTab执行资源tab组内排序或跨pane迁移。native leaf只有Kind和单份资源state，没有通用tab列表、tab迁移或重排；文件专属tab不能替代任意资源的同pane多标签。按一个资源组织能力计缺。
- 进度说明：通用页签已实际推送确认；此前 import cycle 已修复后通过指定检查。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-right/src/client/shell/SidebarRight.tsx:163-176；vendor/deepseek-harness/packages/client/ui-dockkit/src/components/TabPanel.tsx:222-325
- 原生证据：internal/ui/frame/workspace_panes.go:28-45,94-106,337-431；internal/ui/files/tabs.go:279-304

### B14 · workspace.dock-float · 通用资源浮动、移动缩放、最小化及重新停靠
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：任意可停靠资源可浮动、移动、缩放、最小化并重新停靠，保持 live resource 及状态；终端、计划、agents、diff、inspector 均走通用能力。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：原版任意可停靠资源拖出即可float，保留live resource并能unfloat/移动/缩放。native有文件只读快照浮窗，但无通用pane FloatRect或float/unfloat动作，terminal/plan/agents/diff/inspector无法等价浮动。文件/媒体专用预览由另一域审查，不重复计。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-right/src/client/shell/SidebarRight.tsx:173-178；vendor/deepseek-harness/packages/client/ui-dockkit/src/components/TabLayout.tsx:43-118；vendor/deepseek-harness/packages/client/ui-dockkit/src/components/TabMenu.tsx:1-6
- 原生证据：internal/ui/frame/workspace_panes.go:13-45；internal/ui/files/floating.go:13-24,60-90

### B15 · workspace.dock-session-restore · 按会话保留并恢复独立资源布局
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：按会话保存独立资源分组/拆分/活动项与资源状态；切回会话恢复对应布局，不再复用唯一全局 WorkspacePanes。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：native能够Export/Restore单一全局WorkspacePanes，但没有原版bySession资源布局。切会话共用split/kind/active布局，diff root变化清state。文件各workspace草稿仍可保留，不将其误说全部丢失。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-right/src/client/shell/SidebarRight.tsx:363；vendor/deepseek-harness/packages/client/ui-sidebar-right/src/client/persistence.ts:63-98
- 原生证据：cmd/whaleisle/main.go:2879-2882；internal/recovery/store.go:22-41；cmd/whaleisle/recovery.go:55-60,156；internal/ui/frame/workspace_diff.go:22-29,54-62

### B16 · BR07 · 本机开发服务器发现与快捷打开
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：发现本机正在运行的开发服务器，提供发现列表、刷新/定期更新及可点击打开入口，正确呈现服务器消失或探测失败状态。
- 提交：321f42c2b3bcddaa838a719e874a3fdbd7fefb93
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：没有端口探测、发现列表、定期刷新或快捷打开入口；检索 internal/cmd 未发现等价发现调用。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/main/preview.js:233-245,1351-1354；vendor/deepseek-harness/packages/client/ui-preview/src/client/PreviewPanel.tsx:303-323,803-823
- 原生证据：internal/ui/browserpreview/controller.go:216-380；internal/ui/browserpreview/tools.go:177-317

### B17 · BR08 · 可视网页标注并附加到聊天草稿
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：网页标注支持多元素、区域框选、自由笔/擦除、局部样式编辑、评论与局部截图附件，结果可附加到当前聊天草稿。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：已有单元素选择→有界文本草稿。缺多元素、区域框选、自由笔/擦除、局部样式编辑、标注评论与局部截图附件；原版这些都属于真实 pick 界面。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/main/preview-guest-preload.js:330-1125；vendor/deepseek-harness/packages/client/ui-preview/src/client/pickMarkdown.ts:18-31；vendor/deepseek-harness/packages/client/ui-preview/src/client/PreviewPanel.tsx:636-650
- 原生证据：internal/desktop/webview/inspect.go:31-63；internal/desktop/webview/picker.js:1-130；internal/ui/browserpreview/actions.go:82-157

### B18 · BR10 · 网页录制并导出视频
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：网页录制可完成超出当前 2 分钟截断的长录制并导出视频；对齐原版 512MiB 保存上限或明确等价容量策略，不静默缩减到当前限制。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：已有真实无音频 WebM 编码导出；native 限2分钟/64MiB/1280×720/8fps，原版服务保存上限512MiB且录制路径无相同2分钟截断；长录制能力缩减。不因实现采用截图流判缺失。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-preview/src/client/browserRecording.ts:221-360；src/main/preview.js:1007-1035
- 原生证据：internal/ui/browserpreview/recording.go:65-145,154-242；internal/desktop/browserrecording/recording.go:17-22,94-112

### B19 · BR12 · 聊天上方可交互浏览器小浮窗并恢复同一页面
- 基线状态：缺失
- 当前状态：待实现
- 完成准则：聊天上方小浮窗承载同一个可交互 live guest，支持拖动缩放及恢复原面板，恢复保留页面上下文而非重新抓帧。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：native 只有抓帧 PiP 和文档浮窗，没有原版独立 mini-player：移动同一 live guest 到聊天上方、交互、拖动缩放、恢复原面板。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-preview/src/client/DshdMiniPlayer.tsx:51-98,178-179；vendor/deepseek-harness/packages/client/ui-preview/src/client/PreviewPanel.tsx:325-343,763-772
- 原生证据：internal/ui/browserpreview/picture_in_picture.go:29-61；internal/ui/browserpreview/controller.go:212-380

### B20 · HT02 · 已保存 HTML 交互预览
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：已保存 HTML 从正常 Host 资源入口也可进入隔离交互预览，携带必要来源/根身份并按安全策略允许脚本；补齐行内交互而非仅本地 fallback 独立窗口。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：正常已连接且选中会话时 MediaPreviewSource 返回 reader（不区分本机Host），故主入口走 OpenRemote；该 Document 没有 source/rootIdentity/identity，CanRunScripts=false，脚本按钮隐藏。仅本地fallback具备交互；且为独立窗口而非原版inline。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/html/HtmlBody.tsx:37-96；vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/html/pack.ts:32-74
- 原生证据：cmd/whaleisle/workspace_document_preview.go:96-100,149-158；internal/store/session/media_preview.go:11-25；internal/desktop/htmldocument/remote.go:76-82；internal/desktop/htmldocument/interactive.go:25-38；internal/desktop/htmldocument/image.go:20-28；internal/ui/browserpreview/html.go:63-65；internal/ui/browserpreview/interactive_html.go:91-176

### B21 · HT04 · 生成 HTML 发布回复的行内交互与保状态展开
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：生成 HTML 回复可自动行内交互，各回复独立持有浏览上下文；展开/收回不重载丢状态，并支持动态高度桥。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：可显式打开独立隔离交互窗口；缺自动行内互动、同一浏览上下文保状态展开/收回、多张回复同时独立保有页面、动态高度桥。静态PNG不能替代。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-visual-replies/src/client/PublishedPage.tsx:52-137；vendor/deepseek-harness/packages/client/ui-visual-replies/src/client/VisualReplyFrame.tsx:27-60；vendor/chisacode-remote/packages/app/src/components/generative-html-preview-frame.web.tsx:8-27
- 原生证据：internal/ui/chat/visual_reply.go:8-90；cmd/whaleisle/visual_reply.go:122-167；internal/desktop/previewprocess/helper_windows.go:154-224

### B22 · HT05 · 生成 HTML 随应用主题实时更新
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：生成 HTML 预览在应用主题改变时收到实时主题 tokens 并更新当前页面，无需重新打开，持续 IPC 不因后续主题消息关闭。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：只有打开时的light/dark和固定palette；IPC只收一次document，后续任意数据关闭，没有主题更新；原版发送实时当前主题tokens。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-visual-replies/src/client/VisualReplyFrame.tsx:17-40,59
- 原生证据：internal/desktop/htmlpreview/shell.go:32-54；internal/desktop/previewprocess/helper_windows.go:211-224；cmd/whaleisle/visual_reply.go:138-157

### B23 · DO05 · 图片每标签缩放/适宽与滚动查看
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：每个图片资源标签独立保存缩放值，提供放大/缩小/适宽及大图平移滚动，切换标签恢复对应查看位置。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：只有固定contain缩放以适应区域；没有原版按标签保存的缩放值、放大缩小/适宽及查看大图的平移滚动。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/image/ImageBody.tsx:61-73,122-148；vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/zoom/ZoomControls.tsx:1
- 原生证据：internal/ui/files/media.go:82-116；internal/desktop/imagepreview/image.go:57-60；internal/ui/files/floating.go:124-132

### B24 · DO07 · Markdown资源预览含YAML元数据字段
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：Markdown 资源 viewer 解析首部 YAML 元数据并显示字段列表，正文继续正常渲染，不能把元数据直接当普通正文。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：普通Markdown已渲染；未实现该资源viewer专有的首部YAML解析与字段列表，原文会按Markdown正文处理。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/markdown/MarkdownBody.tsx:27-77
- 原生证据：internal/ui/files/files.go:493-499；internal/ui/widgets/markdown/markdown.go:160

### B25 · DO08 · 资源文件预览实现选择与扩展优先级
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：资源预览器具有动态注册与扩展优先级，多个可用 viewer 可供选择，选中实现直接服务同一原生资源 tab。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：内置类型与源码/渲染模式已有；native资源窗格没有原版documentPreviews动态registry和多个viewer候选菜单。网页插件载体不等于把扩展预览器接进native同一资源tab。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/document/registry.ts:35-55,97-130；vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/TextPreview.tsx:246-268
- 原生证据：internal/desktop/workspacefs/docs.go:22-39；internal/ui/files/files.go:344-448；internal/desktop/pluginslots/client.js:1

### B26 · DO10 · 文本资源按页继续读取完整大文件
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：通用文本资源 viewer 提供继续读取下一页入口，累积展示超过首 1MiB 的内容，维持读取位置和结束/失败状态。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：原版通用资源viewer有loadNext累积text-pages，native Files固定前1MiB/远端单read无继续页入口。原版desktop Files同样1MiB，故差异仅针对通用资源viewer，不误称原Files编辑器回归。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/TextPreview.tsx:235-241,300-340
- 原生证据：internal/ui/files/text_preview.go:10-27；internal/ui/files/text_read.go:105-120；internal/desktop/workspacefs/remote_list.go:72-91

### B27 · SCHEDULE-05 · 可视化时间/日期/时区与 Cron 构建编辑
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：定时任务编辑提供可视日期/时间选择、近期时区及 Cron 构建与预览，保存可用规则并支持读取现有规则回填编辑。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：规则读写具备，但 native 是 RFC3339、秒数、IANA 文本及逗号星期/Cron 字符串输入；未移植原日期/时钟选择器、近期时区和 Cron 可视化构建/预览。不能按 RPC 已有算这一用户操作体验完整。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-schedule/src/client/TaskDetail.tsx:29-30；vendor/deepseek-harness/packages/client/ui-schedule/src/client/TaskDetail.tsx:1543；vendor/deepseek-harness/packages/client/ui-schedule/src/client/DatePicker.tsx:1；vendor/deepseek-harness/packages/client/ui-schedule/src/client/ClockPicker.tsx:1；vendor/deepseek-harness/packages/client/ui-schedule/src/client/recent-time-zones.ts:1
- 原生证据：internal/ui/schedule/panel.go:313-350

### B28 · life-cancel-recovery · 在启动页取消自动恢复
- 基线状态：缺失
- 当前状态：代码已推送；实机验收待后置
- 完成准则：启动页提供取消本次自动恢复入口，真正取消待执行/进行中的恢复流程，避免取消后仍自动重启；无需离开启动页整体停止 Host。
- 提交：0d537dbaf3d213de91546d601d4195d86e0a6000
- 验证范围：定向检查已通过；未做真实 Windows GUI/真实账户集成验收。
- 基线说明：启动页没有cancel-restart等价动作；只能离开启动页到启动器整体停止Host。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/main/ipc.js:212-214
- 原生证据：internal/ui/boot/boot.go:18-26,68-84;internal/harness/supervisor.go:169-218

### B29 · life-boot-log · 查看并导出启动诊断日志
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：启动诊断导出包含版本、时间、snapshot 及日志行；取消保存对话框不生成默认日志文件。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：只导出日志行，没有原版版本/时间/snapshot诊断封套；取消保存对话框仍会写默认日志文件。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/main/ipc.js:216-232;src/main/boot-log-dump.js:1
- 原生证据：internal/harness/logexport.go:17-49;cmd/whaleisle/main.go:7046-7055

### B30 · launcher-offline-plugin-repair · 启动器离线插件修复（禁用、启用、移除）
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：Host 未就绪时可在启动器离线禁用指定/可疑插件并重试，也能单项重新启用和移除坏插件；不依赖先启动 ready Host 或恢复模式。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：Host未就绪时禁用指定或可疑插件并重试：原生精确修复依赖ready Host的pluginManager；原版支持launcher configIO离线禁用后启动。原生须先恢复模式启动。；启动器单项重新启用插件：启动器库存无单项启用动作；正常Host设置页有插件启停，不能替代离线修复。；启动器移除坏插件：有正常插件管理，但恢复列表没有原版离线removePlugin入口。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/launcher/launcher-service.js:812-888;src/main/ipc-launcher.js:168-172；src/launcher/launcher-service.js:891-903;src/main/ipc-launcher.js:174；src/launcher/launcher-service.js:624-672;src/main/ipc-launcher.js:176
- 原生证据：cmd/whaleisle/startup_diagnostics.go:116-211;internal/ui/launcher/startup_diagnostics.go:43-50；internal/ui/launcher/startup_diagnostics.go:60-75;cmd/whaleisle/settings_plugins.go:1

### B31 · components-catalog · 发现可安装通用组件目录
- 基线状态：部分实现
- 当前状态：已实现；定向通过；待推送
- 完成准则：组件目录同时发现用户已有组件和 bundled components/samples 中可安装的通用组件，并正确区分已安装与可安装条目。
- 提交：无已发布提交
- 验证范围：指定定向检查已通过；未跑全库，未作真实 GUI 验收，尚未推送。
- 基线说明：只扫描已有userData/components目录，不扫描原版bundled components/samples目录。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/launcher/components/registry.js:14-34;src/launcher/components/index.js:119-134
- 原生证据：internal/desktop/components/list.go:43-63,124-158

### B32 · components-generic-lifecycle · 通用组件生命周期管理（安装、启停、更新、回滚、卸载）
- 基线状态：缺失
- 当前状态：已实现；定向通过；待推送
- 完成准则：非 WhaleBridge 通用组件具备安装、启动、停止、更新、回滚、卸载的真实执行路由，操作对象、版本和失败状态可追溯。
- 提交：无已发布提交
- 验证范围：指定定向检查已通过；未跑全库，未作真实 GUI 验收，尚未推送。
- 基线说明：通用组件安装：非WhaleBridge安装显式unsupported。；通用组件启动：非WhaleBridge启动显式unsupported。；通用组件停止：非WhaleBridge停止显式unsupported。；通用组件更新：非WhaleBridge无更新执行路由。；通用组件回滚：非WhaleBridge无回滚执行路由。；通用组件卸载：非WhaleBridge无卸载执行路由。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：src/launcher/components/index.js:383;src/main/ipc-components.js:44；src/launcher/components/index.js:429;src/main/ipc-components.js:49；src/launcher/components/index.js:434;src/main/ipc-components.js:51；src/launcher/components/index.js:449;src/main/ipc-components.js:58；src/launcher/components/index.js:501;src/main/ipc-components.js:67；src/launcher/components/index.js:547;src/main/ipc-components.js:75
- 原生证据：internal/desktop/components/lifecycle.go:28-62；internal/desktop/components/lifecycle.go:28-62;cmd/whaleisle/main.go:3215-3277

### B33 · package-windows · Windows完整原生安装包组装
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：Windows 原生打包必须包含顶层 Host Node 等必需运行时；缺失则阻止成功 marker/成包，不产生表面成功的残缺包。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：Node目录缺失仅提示后继续，仍可写.runtime-staged，可能得到缺顶层Host Node的包。独立候选校验工具存在。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：package.json:20-42,166-191
- 原生证据：scripts/package.ps1:29-111;scripts/nsis/whaleisle.nsi:35-51

### B34 · package-pet · 发行包收录THA4资源
- 基线状态：部分实现
- 当前状态：待实现
- 完成准则：默认发行流程将 THA4 所需资源加入 stage 并由安装包收录；清空 stage 后仍重新装配资源，不能静默回退替代。
- 提交：无已发布提交
- 验证范围：未开始本项新增实现的检查；功能完成后再进入对应验收门槛。
- 基线说明：package.ps1清空stage且未调用pet-render-assets，NSIS只按marker收录；默认流程遗漏THA4并触发回退。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：package.json:77-90;src/main/desktop-live2d.js:1
- 原生证据：scripts/package.ps1:25-28;scripts/pet-render-assets.mjs:1-35;scripts/nsis/whaleisle.nsi:53-69;internal/desktop/pet/neural.go:33-58

### B35 · SP-11 · 账户联系我们反馈入口
- 基线状态：缺失
- 当前状态：已实现；定向通过；待推送
- 完成准则：账户/关于等明确入口可打开反馈表，并按原版能力提供 UID、版本、屏幕及设备上下文；数据传送遵循用户授权与隐私要求。
- 提交：无已发布提交
- 验证范围：指定定向检查已通过；未跑全库，未作真实 GUI 验收，尚未推送。
- 基线说明：原版contactUs打开带UID/版本/屏幕/设备上下文的反馈表；native账户、关于和应用菜单无此操作，只有仓库/文档链接。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-settings-account/src/client/AccountMenu.tsx:85
- 原生证据：internal/ui/settings/account.go:64；internal/ui/settings/about.go:123

### B36 · SP-12 · 账户运行状态通知（登录过期、需登录、额度耗尽）
- 基线状态：部分实现
- 当前状态：已实现；定向通过；待推送
- 完成准则：消费 deepseek-account/session-expired、model-sign-in-required、ACCOUNT_QUOTA 三类事件，提供跨界面主动提醒及对应重新登录/直接充值入口。
- 提交：无已发布提交
- 验证范围：指定定向检查已通过；未跑全库，未作真实 GUI 验收，尚未推送。
- 基线说明：手动登录/资料余额/用量/充值及奖励通知已实现；缺少三种运行中主动通知：deepseek-account/session-expired、model-sign-in-required，以及ACCOUNT_QUOTA frame-wide提醒/直接充值。已与chat审查交叉确认，未发现这三种事件的等价通知消费者。
- 进度说明：按父任务已确认的执行进度登记；本表生成不新增代码审查或产品测试。
- 原版证据：vendor/deepseek-harness/packages/client/ui-settings-account/src/client/AccountMenu.tsx:41；vendor/deepseek-harness/packages/client/ui-settings-account/src/client/AccountQuotaNotice.tsx:72
- 原生证据：cmd/whaleisle/main.go:1834；internal/ui/settings/account.go:64；internal/ui/chat/usage_label.go:18；cmd/whaleisle/account_return.go:39

### A01 · chat.questions.recommended-default · 交互问答仅默认选中首个推荐选项
- 基线状态：基线之后新增发现
- 当前状态：本地已写；上传阻塞，未发布
- 完成准则：仅当存在推荐选项时，默认选中第一个推荐选项；无推荐项不自动选第一项，用户选择和逐题 Skip 保持独立语义。
- 提交：无已发布提交
- 验证范围：单个定向 fixture 已通过（0.004s）；未作 GUI/完整集成检查。
- 阻塞：上传在已授权的一次重试后，工具仍返回 cancelled；没有已发布提交。当前需恢复该特定上传动作许可；仅确认工具返回取消，不能推断用户主动取消。本表不另发授权请求。
- 原版证据：未提供新增源码定位；不虚构证据。
- 原生证据：未提供新增源码定位；不虚构证据。

### E01 · BR14 · 网页文件上传/文件选择
- 基线状态：证据不足
- 当前状态：待澄清/补证据
- 完成准则：在真实 Windows WebView2 页面验证 file input 能打开文件选择器、选择并交付文件及取消行为；依据结果判定可关闭还是新增具体修复，不能仅因缺实机证据判缺失。
- 提交：无已发布提交
- 验证范围：未补充运行证据；运行类确认留待对应后置门槛，不提前启动一轮审查。
- 基线说明：原版依赖 Chromium 原生 file input，native依赖 WebView2 默认行为；没有运行或足够显式文件选择接线证据，不能判缺失。
- 原版证据：src/main/preview.js:532-590
- 原生证据：internal/desktop/webview/host.go:197-235；third_party/mygo/internal/windows/webview.go:1

### E02 · HT07 · 视觉生成启用/运行就绪与实际renderer一致性
- 基线状态：证据不足
- 当前状态：待澄清/补证据
- 完成准则：厘清启用 Probe 与实际 AppContainer helper 路径的 readiness 合同，使设置启用状态与实际 renderer 可用性一致；真实 Windows 结果用于确认是否存在功能故障。
- 提交：无已发布提交
- 验证范围：未补充运行证据；运行类确认留待对应后置门槛，不提前启动一轮审查。
- 基线说明：启用仍Probe进程内隔离窗口，真实POST使用独立AppContainer helper；helper成功不更新GET readiness。确有双路径不一致，但未运行不能断言Windows一定无法启用；不计确定缺失。
- 原版证据：vendor/deepseek-harness/packages/client/ui-visual-replies/src/config.ts:1-30；vendor/deepseek-harness/packages/client/ui-visual-replies/src/tools.ts:49-70
- 原生证据：cmd/whaleisle/settings_visual_replies.go:15-44；internal/desktop/htmlpreview/readiness.go:18-80；internal/desktop/htmlpreview/capture_windows.go:24-43；internal/harness/current_host.go:110-111

### E03 · launcher-slim · 独立轻量启动器管理另一桌面安装
- 基线状态：证据不足
- 当前状态：待澄清/补证据
- 完成准则：明确是否要求独立轻量启动器产品形态；若要求，具备独立管理另一桌面安装的入口与生命周期；若不要求，记录明确范围结论，不把应用内 Host 管理算缺失。
- 提交：无已发布提交
- 验证范围：未补充运行证据；运行类确认留待对应后置门槛，不提前启动一轮审查。
- 基线说明：原生是应用内启动器，未找到独立launcher产品形态；是否必须保留该形态需明确，不将已实现Host管理计为缺失。
- 原版证据：package.json:25;electron-builder.launcher.yml:1;src/launcher/launcher-service.js:380-414,931
- 原生证据：internal/ui/launcher/panel.go:1;cmd/whaleisle/native_releases.go:36

### P01 · PF01 · macOS/Linux隔离HTML/PDF/媒体文档表面
- 基线状态：缺失
- 当前状态：后续平台待实现
- 完成准则：macOS/Linux 具备隔离 HTML、PDF 和媒体文档表面及资源读取/交互路径，不再直接返回不支持。
- 提交：无已发布提交
- 验证范围：非 Windows 后续范围，未实机验证，不混入 Windows 36 项缺口。
- 基线说明：这些isolated local文档Windows以外明确拒绝；普通浏览器backend存在不能替代。
- 原版证据：src/main/html-preview.js:1；src/renderer/file-preview.js:59-75
- 原生证据：internal/desktop/previewprocess/launcher_other.go:11-13；third_party/mygo/window.go:396-397

### P02 · PF02 · Linux网页明暗外观覆盖
- 基线状态：缺失
- 当前状态：后续平台待实现
- 完成准则：Linux 浏览器可手动指定明/暗外观并更新当前页面，而非只能跟随系统。
- 提交：无已发布提交
- 验证范围：非 Windows 后续范围，未实机验证，不混入 Windows 36 项缺口。
- 基线说明：Linux明确BrowserColorSchemeAvailable=false，跟随系统，不能手工覆盖。
- 原版证据：src/main/preview.js:1283-1286
- 原生证据：third_party/mygo/internal/linux/browser_profile.go:128；internal/ui/browserpreview/tools.go:280

### P03 · PF03 · macOS/Linux GIF动画与完整SVG/ICO图像渲染
- 基线状态：部分实现
- 当前状态：后续平台待实现
- 完成准则：macOS/Linux 图片 viewer 支持 GIF 动画及完整 SVG/ICO 渲染，覆盖字体、嵌图与滤镜，不仅首帧或简化 fallback。
- 提交：无已发布提交
- 验证范围：非 Windows 后续范围，未实机验证，不混入 Windows 36 项缺口。
- 基线说明：非Windows不走隔离img；fallback GIF仅首帧、SVG字体/嵌图/滤镜可能省略。
- 原版证据：vendor/deepseek-harness/packages/client/ui-sidebar-documentpreview/src/client/image/ImageBody.tsx:17-27,139-148
- 原生证据：internal/ui/files/media.go:49-50,82-83,176

### P04 · SP-14 · macOS系统钥匙串凭据持久化
- 基线状态：缺失
- 当前状态：后续平台待实现
- 完成准则：macOS 凭据通过系统钥匙串持久化；Linux 同根因限制在同项明确处理或范围说明，不额外重复计数。
- 提交：无已发布提交
- 验证范围：非 Windows 后续范围，未实机验证，不混入 Windows 36 项缺口。
- 基线说明：原版safeStorage使用OS钥匙串，native非Windows后端直接返回不支持。Linux同根因附注，不单独计数。
- 原版证据：src/main/config.js:374
- 原生证据：internal/credentials/dpapi_other.go:9；internal/credentials/storage_status.go:16

### N01 · REQ-PANEL-01 · 用户自定义右侧 Panel 与可复用 Panel 库
- 基线状态：新增需求，非原版重构缺口
- 当前状态：待实现
- 完成准则：用户可定义、保存、选择、修改及复用右侧 Panel 模板，具备持久化模板库与实际渲染/操作入口，不能以固定内建扩展 tab 替代。
- 提交：无已发布提交
- 验证范围：未实现、未验收；不混入固定基线265项。
- 基线说明：原版存在内建扩展 tab/固定 Bot 页面与 Project observer，不等于用户能定义/保存/选择/修改 Panel 模板库。扫描 native cmd/internal 未见此类 schema/store/editor/执行入口。此为新增要求，不纳入原版重构缺口。
- 原版证据：vendor/dshbot/client/client.js:4228-4695；vendor/dsh-project/client/client.js:1354-1358
- 原生证据：internal/ui/assistant/panel.go:76-145；cmd/whaleisle/main.go:6103-6113

### N02 · REQ-PANEL-02 · 每个 Bot 独立 Panel 实例与状态隔离
- 基线状态：新增需求，非原版重构缺口
- 当前状态：待实现
- 完成准则：按 Panel definition ID × Bot ID 创建独立实例并持久化状态；切换 Bot 恢复各自实例，修改不串扰其他 Bot。依赖 N01。
- 提交：无已发布提交
- 验证范围：未实现、未验收；不混入固定基线265项。
- 基线说明：已有 canonical session/各 Bot 控制状态隔离不等于可实例化自定义 Panel；未见 panel definition ID × bot ID 实例持久化与切换恢复链。依赖上一新增能力。
- 原版证据：vendor/dshbot/client/client.js:7247-7283
- 原生证据：internal/ui/assistant/panel.go:76-145；cmd/whaleisle/assistant_conversation.go:32-136

## 功能完成后的验证与组装门槛

### V01 · 真实 Windows GUI
- 状态：功能完成后再执行
- 通过条件：功能模块完成并集成后，实际运行 Windows GUI，确认输入/焦点/布局/交互及会话切换行为，记录环境与失败项。
- 当前验证：未执行。

### V02 · Host 与真实账户桥接
- 状态：功能完成后再执行
- 通过条件：验证真实 Host 连接、账户会话和事件流，以及需要真实服务的插件/账户桥接；记录版本、授权边界和服务条件。
- 当前验证：未执行。

### V03 · 文件、Git、SSH 与 PTY
- 状态：功能完成后再执行
- 通过条件：验证本机及已授权远端文件操作、Git 工作流、SSH 和交互 PTY 的端到端路径、取消/错误及资源释放。
- 当前验证：未执行。

### V04 · 媒体、HTML、PDF 与编解码
- 状态：功能完成后再执行
- 通过条件：验证真实资源打开、隔离交互、导出/录制、PDF 和媒体编解码，补齐 E01/E02 所需实机证据。
- 当前验证：未执行。

### V05 · 桌宠 GPU 与音频
- 状态：功能完成后再执行
- 通过条件：在目标 GPU/音频环境确认 THA4 资产加载、渲染与音频路径，记录依赖及回退行为。
- 当前验证：未执行。

### V06 · 最终移动端与发行集成
- 状态：功能完成后再执行
- 通过条件：组装最终发行候选，核对 Windows 必需运行时/组件/桌宠资源和安装更新链；确认移动端供给、ABI 与实际联动，形成最终交付记录。
- 当前验证：未执行。

## 范围和记录约定

原版基线提交：e9f584be6d3b491ce777bb6823f30bc2bbc6f195
原生基线提交：1a941e1f2bb756c152048275fa0e02fed6c21999
本表进度采用已确认的执行记录；生成本表没有更改产品代码、运行产品测试、上传或提交。后续实施只更新执行状态、具体变更、提交与验证范围，保留基线证据。
详细基线覆盖限制随 JSON 一并保留。平台项独立安排，未知项不自动判缺失；新增 Panel 要求不伪装成原版已有能力。
