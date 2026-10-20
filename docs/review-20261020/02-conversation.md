# 02 · 对话渲染与消息组件（conversation）对比审查

审查日期：2026-10-20
基准：运行中原版 Electron 0.3.9（3080）/ 原生 Go 实现（3090）
范围：原版 `ui-conversation`、`ui-chat`、`ui-approval`、`ui-user-questions`、`ui-tool`、`ui-message-edit`、`ui-message-feedback`、`ui-deliverables`、`ui-goal`、`ui-plan`、`ui-jobs`、`ui-attachment`、`ui-trajectory`、`ui-diff`；原生 `internal/ui/{chat,interaction,widgets/markdown,diffsurface}` 与 `internal/ui/frame` 中消息相关文件。

## 取证说明（先读）

1. 提供的原生截图 `evidence/native-go-3090.png` 是一个**空会话**（画面只有 `New chat · session-…`、`对话/轨迹` 两个 tab、一行 `workspace ready:` 和一行 `选择会话后开始对话`），因此**消息级视觉无法用截图对齐**。本文件的对话层结论以两端源码 + 原生既有快照测试为准；需要补一张“有对话的原生截图”才能闭环（见三、待验证 1）。
2. 原版源码有两份：安装版 rc.2 构建产物 `C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\<pkg>\lib\client.js` 与源码树 `runtime\vendor\deepseek-harness\packages\client\<pkg>\src`（alpha.1）。本文件用 `runtime\vendor\…\src`（可读 TSX/CSS），凡与运行版可能不一致的条目标注 `version-drift`。
3. 原生侧的证据大多是 1:1 对齐原版 token 的实现（例如气泡色 `rgb(237,243,254)` / `rgb(44,44,46)`、宽度轴 `clamp(680, 64%, 920)`、`axis*0.702` 气泡宽度上限、宽度拖拽轨），这些**已经对齐，不计入差距**；差距集中在“块级排版、工具/思考/交互卡片的形态、视觉原语与本地化”。

---

## 一、5 个最致命差距

### 1. Markdown 正文被压成“一段富文本”，块级排版整体消失（P0）

原版正文是一条完整的排版阶梯：`h1–h3` 有 32/16 外边距，`p` 16px，`ul` 左缩进 18px、`li` 间距 6px，`blockquote` 有 2px 左边线与 14px 缩进，行内代码是带 0.5px 边框 + 圆角 + 0.875em 的 chip，宽表可以“破列”占满滚动区（`ui-primitives/src/markdown/MarkdownText.module.css:16-47,102-118,151-155,163-174,192-200`；`ui-chat/src/client/chat/AssistantMarkdown.module.css:17-21,33-43`）。

原生把所有块都拆成同一列 `Gap(4)` 的 span：标题只是“字号 + 5/+3/+1、字重 700”的行内 span，列表是 `"• "` / `"1. "` 文本 + 两个空格缩进，引用只有“斜体 + muted”，表格是每格固定 `Width(180)` + 1px 边框的网格，行高从没被显式设置（`internal/ui/widgets/markdown/blocks.go:21,102-131,133-173`；`markdown.go:274-287,313-352,142-145`）。结果：整篇回答没有段落呼吸、标题看不出层级、引用没有“引”的形状、表格被钉死 180px（长内容换行、与列宽无关），与原版是两种阅读体验。这是**每次对话都在发生**的断裂。

### 2. 工具调用被渲染成一张带边框卡片 + 160/200 字原始 JSON（P0）

原版一次工具调用 = **单行 disclosure**：16px 前置图标 + 标题 13/24 + 2px 圆点分隔 + 摘要（可省略号截断）+（差异时）`+N/-N` 计数；整行可点开，展开后按工具类型落入结构化卡片：`TerminalBlock`（终端）、`DiffBlock`（差异）、`ReadBlock`（文件读取）、`SearchBlock`（检索）、`WebBlock`（网页）、图片 gallery、`ToolDetails`（`ui-tool/src/client/tool/components/ToolRow.tsx:257-343,360-378`；`ToolRow.module.css:1-6,96-108,173,199`；`ToolRow.tsx:250-254`）。

原生每个 tool 事件都画一张 `Background(field)+Border(1)+Radius(6)+Padding(8,10)` 的卡片，首行是“彩色文字块 chip（bash/read/write…）+ 工具名加粗 + 状态英文词”，下面直接贴 `truncateToolText(...,160/200)` 的原始参数/结果，参数与结果又各自是 `"▸ 参数 / Arguments"` + 一段等宽文本来展开（`internal/ui/chat/toolcard.go:138-153`；`toolkind.go:20-44`；`tool_lifecycle.go:54-96,123-148`）。一轮 20 次工具调用就是 20 个三行以上的边框盒子，密度、层次、可读性全部与“单行 24px 行高”的原版相反——这是“界面和原版差距很大”最主要的来源。

### 3. 缺少正版视觉原语（图标按钮 / DisclosureRow / StateDot / TextShimmer），折叠与状态一律退化为“文字按钮 + ASCII 箭头”（P0）

原版的次要行统一由 `DisclosureRow` 承载：图标 + 标题 + 圆点分隔 + 摘要，整行可点，运行中有 `TextShimmer` 扫光，状态用 `StateDot`（思考行 `ReasoningRow.tsx:88-102` + `ReasoningRow.module.css:6-9,59-66`；工具行 `ToolRow.tsx:360-378`；回合过程 `TurnProcessNodeView.tsx:36-58`；用量 `TurnUsageDisclosure.tsx:33-50`）。`@keyframes` 只出现在 `TurnNavigator`、`TextShimmer`、`StateDot`、`TypingFxLayer`、`ComposerBeam` 这些原语里，说明“动效”本身就是这套设计系统的一层。

原生的折叠/状态控件没有任何 shimmer、StateDot 或持续动画：整个 chat 包里唯一的定时重绘只用于复制对勾回收（`message_actions.go:52`）和 `workflow.go:235` 的环形进度，运行中的思考/工具/过程行都是静态文字。折叠控件全部是文字：`"▸ 思考"`、`"▸ 参数 / Arguments"`、`"▸ 工作步骤 / Work details"`、`"▸ 本轮用量 / Turn usage"`、`"▸ 收起问题 / Minimize question"`、`"▸ 上下文注入 · …"`，箭头是字符 `▸ ▾ ⌃ ⌄ ×`（`internal/ui/chat/thinking.go:32-45`；`tool_lifecycle.go:127-137`；`transcript_preferences.go:237-250`；`performance_usage.go:35-45`；`internal/ui/interaction/question_navigation.go:25-34`；`context_injection.go:15-31`；`todopanel.go:62-66`）。观感上就是“能跑的原型”，不是原版那套 24px 行高、图标对齐、悬停扫光的成品。

### 4. 中英硬拼接：UI 文案绕过 locale 字典，直接显示「中文 / English」（P0）

原版所有产品文案都走 `t()`：`ui-chat/src/client/locale.ts`（20KB）与 `markdown-labels.ts`，组件里只有 key（`AssistantMarkdown.tsx:60-61`；`MessageIconActions.tsx:85-108`）。

原生明明有 `internal/ui/i18n`（`i18n.go:1027-1031,1961-1965` 提供双语字典、`i18n.Locale()` 可用），但 chat + interaction 两包的 **28 个非测试源文件里有 176 处**硬编码 `"中文 / English"` 双写字符串（`Select-String -Pattern '[\u4e00-\u9fff] / [A-Za-z]'`），例如：`参数 / Arguments`、`结果 / Result`、`检查此调用 / Inspect this call`、`正在读取文件 / Reading files`、`本轮用量 / Turn usage`、`发送中 / Sending…`、`已接受，等待会话更新 / Accepted, waiting for conversation update`、`收起问题 / Minimize question`、`在 Host 打开 / Open on Host`（`internal/ui/chat/transcript_preferences.go:220-228`；`tool_lifecycle.go:71-83,137`；`performance_usage.go:39-60`；`pending_submission.go:28-31`；`turn_deliverables.go:135,241-246`；`interaction/question_navigation.go:25-76`）。用户在任何语言下都会看到两套语言叠在一行里，这是最廉价、也最伤专业感的差距。

### 5. 空对话态退化 + 内部状态串直接进 UI（P0）

原版空会话渲染居中 hero：34px 鱼标 + 26/32 标题 + “预览”胶囊徽章，composer 居中（`ui-conversation/src/client/skeleton/EmptyHero.tsx:102-166`；原生另实现了同名 hero：`internal/ui/frame/conversation_phase.go:73-91`，说明这条能力存在）。

但证据截图里实际看到的是：`y≈87` 一行灰字 **`workspace ready: …`**，`y≈115` 一行 13px 灰字 `选择会话后开始对话` 顶在最上方。原因三处：
- `internal/store/session/workspace_rpc.go:37` 把 `s.lastAction = "workspace ready: " + title`，`chat.go:396-398` 无条件把它画成正文（连内部英文短语一起）。
- 消息为空时只画一行 `KeyChatEmpty` + 16px padding（`chat.go:140-142`，字典值 `i18n.go:1027`）。
- hero 的开启条件很窄：只有 `SelectedID=="" || SelectedSummaryBlank`，或 `HistoryReady && 消息为空` 才进 `conversationHero`；`Messages==0 && !HistoryReady` 落进 `conversationSettling`，而 settling 阶段只画一个空的 `conversation-phase-space` 盒子（`conversation_phase.go:32-53`；`frame.go:863-864,1090-1091`）。也就是说：只要这一屏被判成 active（截图里就是这种情况——有会话、有 对话/轨迹 tab、但 0 条消息），用户看到的就是“一行 13px 灰字 + 一行内部状态串”，既不是 hero，也没有骨架。原版在同样状态下永远是居中 hero，不会出现任何宿主内部字符串。

---

## 二、明细表

| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
|---|---|---|---|---|---|---|
| 1 | 正文块级排版 | `h1–h3` 32/16 外边距、`p` 16px、`ul` 18px 缩进 + `li` 6px、`blockquote` 2px 左线、行内代码 chip（`MarkdownText.module.css:16-47,102-118,151-155,163-174`） | 全块挤进 `Gap(4)` 一列；标题只放大字号（+5/+3/+1）、列表是 `"• "` 文本 + 空格缩进、引用仅斜体、无行高（`blocks.go:21`；`markdown.go:274-287,329-352,142-145`） | 段落呼吸、标题阶梯、引用与列表形状全部丢失 | P0 | markdown 增加“块级视图”分支：按块类型给 16/32 外边距、24px 行高、引用左边线、真实 `ol/ul` 缩进；`DocumentLayout` 已有骨架可直接复用 |
| 2 | 表格 | 自动列宽 + `tableScroll` 横向滚动；≥4 列可破列占满滚动区（`MarkdownText.module.css:192-200`；`AssistantMarkdown.module.css:33-43`） | 每格硬编码 `Width(180)` + 1px 边框，表头仅 700 字重，无表头底色/对齐（`blocks.go:136-172`） | 宽表被强行折行，窄表浪费 180px；表头不可辨识 | P1 | 先测内容宽度再分配列宽，表头加背景 token，保留“超出则横向滚动”策略 |
| 3 | 代码块 | 语言标签 + shiki 主题色；工具栏是**图标**（换行/复制）+ tooltip + `aria-pressed`，复制后 1s 换对勾；支持行号；流式增量高亮（`CodeBlock.tsx:97-149,156-170,187-208`；`CodeToolbar.tsx:42-59`） | 语言文本 + 两个文字按钮 `Copy code` / `Wrap lines`，无行号、无复制反馈、固定 chroma `github`（浅色）样式（`blocks.go:97-131`；`highlight.go:20-31`） | 代码块是“灰盒子 + 两个文本按钮”；深色主题下高亮仍是浅色配色 | P0 | 改用图标工具条 + copy 态；深/浅两套 chroma 主题；加可选行号；语言未知时回落到 `code` 标签 |
| 4 | 工具卡片形态 | 单行 disclosure：图标 + 标题 + 圆点 + 截断摘要（13/24），整行点击展开（`ToolRow.tsx:360-378`；`ToolRow.module.css:1-6`） | 每个调用一张边框卡片（`Padding(8,10)+Border(1)+Radius(6)`），首行“文字 chip + 工具名 + 状态词”，第二行长 JSON（`toolcard.go:138-153`） | 一轮十几次调用就是十几张卡片，信息密度与原版相反 | P0 | 新增 `DisclosureRow` 等价原语（图标/标题/圆点/摘要/chevron），工具卡改为单行 + 点击展开 |
| 5 | 工具结构化 body | 终端块、差异块、读取块、检索块、网页块、图片 gallery、明细卡（`ToolRow.tsx:257-343`） | 参数/结果一律 `Text` 原文（JSON 缩进），靠 `"▸ 参数 / Arguments"` 展开（`tool_lifecycle.go:78-81,123-148`） | 命令输出、diff、检索命中都没有专用渲染，用户要读 JSON | P1 | 按 `toolkind.go` 已有分类接 `diff.View`（已在用）、终端等宽块、读取行号块、检索命中列表 |
| 6 | 工具状态与错误态 | `StateDot`（done/ongoing/error）+ 错误行摘要用 error 色、`data-state` 驱动样式（`ToolRow.tsx:104-113,237-246`；`ToolRow.module.css:96-108,244-246`） | 状态是英文裸词 `call/ok/error/running` 上色显示，无圆点、无错误首行摘要，色值写死 `RGB(224,102,102)`（`toolcard.go:110-118,142`；`toolcard.go:241-243`） | 状态不可扫读、错误不醒目、深色主题色值不跟随 | P1 | 引入 StateDot 等价物；状态词进 i18n；错误色走 `state-error` token |
| 7 | 思考过程 | 图标 + “思考” + 圆点 + shimmer 摘要，折叠时固定 24px；展开体缩进、头部 sticky（`ReasoningRow.tsx:88-102`；`ReasoningRow.module.css:6-9,17-22,73-76`） | 带边框卡片 + `"▸ 思考"` 文字按钮 + `"复制"` 文字按钮 + 单行预览（`thinking.go:37-53`） | 多一层边框、少一个图标、无扫光、无“思考用了几秒” | P1 | 与 #3/#4 共用 DisclosureRow；折叠态锁 24px；把耗时接进标题右侧 |
| 8 | 运行中状态 | 行内实时计时“深度思考中，用时 5 分 35 秒” + 分色分隔线 + 小鲸鱼 + shimmer，`tabular-nums`（`RunningStatus.tsx:20-40`；`ChatView.module.css:136-175`） | 只有过程分组里一句 `正在分析请求 / Analyzing request`（或“正在运行命令…”），无计时、无扫光、无图标（`transcript_preferences.go:188-201`） | 用户无法判断“跑了多久/还活着吗”，与原版最显眼的运行反馈缺失 | P1 | 增加常驻 running 行：图标 + 计时（`c.AnimationFrame()` 已有先例，见 `conversation_enter.go:50`）+ 分隔线 |
| 9 | 每轮结束状态 | 回合折叠头部是“用时 2 分 13 秒”+ chevron（`TurnProcessNodeView.tsx:21-58`） | `formatTurnStatus` 生成「已完成，用时 …」但**只被测试调用**，UI 里从不绘制；turn/end 只画用量行与失败详情（`chat.go:196-207,401-416`；`turn_status_test.go:12-21`） | 成功回合没有“完成/耗时”这一行，读者失去节奏锚点 | P1 | 在 turn/end 分支先画状态+耗时（复用 `formatTurnStatus`），再画用量行 |
| 10 | 折叠控件形态 | 统一 DisclosureRow：图标按钮/整行点击，24px 行高（`ToolRow.tsx:360-378`；`TurnUsageDisclosure.tsx:33-50`） | 全部是文字按钮 + ASCII 箭头 `▸ ▾ ⌃ ⌄ ×`（`thinking.go:32-45`；`tool_lifecycle.go:127-137`；`transcript_preferences.go:237-250`；`todopanel.go:62-66`；`trajectory.go:213-218`） | 控件语言与原版不一致，字形在不同字体下错位 | P1 | 提供 `ui.Disclosure` 原语（图标 + 整行点击 + chevron），批量替换文字箭头 |
| 11 | 本地化 | 全部文案走 `t()` 字典（`ui-chat/src/client/locale.ts`；`AssistantMarkdown.tsx:60-61`） | 28 个非测试源文件里 176 处 `"中文 / English"` 双写（`transcript_preferences.go:220-228`；`tool_lifecycle.go:71-83`；`pending_submission.go:28-31`；`interaction/question_navigation.go:25-76`；`turn_deliverables.go:135,241-246`） | 任何语言下都显示两种语言，专业感直接崩 | P0 | 这些 key 迁到 `i18n.go` 双字典，按 `Locale()` 取单语；CI 加一条“对话层禁止 `" / "` 双写”的 grep 门禁 |
| 12 | 空对话态 | 居中 hero（34px 鱼标 + 26/32 标题 + 预览徽章），composer 居中（`EmptyHero.tsx:102-166`） | `KeyChatEmpty` 13px 灰字 + `Padding(16)` 贴左上；`conversationSettling` 阶段什么都不画（`chat.go:140-142`；`conversation_phase.go:49-51`） | 新建会话第一屏是“一行小字”，不是品牌化的 hero | P0 | `Messages==0 && !PromptAttempted` 一律走 hero；settling 补骨架而不是空白 |
| 13 | 内部状态串泄漏 | 原版正文不出现宿主内部短语；状态条只呈现可读结果 | `workspace_rpc.go:37` 生成 `workspace ready: <title>`，`chat.go:396-398` 无条件绘制（截图 x≈288,y≈87） | 英文调试串直接暴露给用户 | P0 | `LastAction` 只接受用户向文案；内部短语在 store 层过滤或改为 i18n key |
| 14 | 审批卡 | warn 边框卡（`radius-xl` + shadow）：warn 色条 + StateDot + “等待你的确认”、15/24 标题、等宽命令、右对齐 outline/primary 按钮、拒绝按钮 hover 转危险色（`ApprovalPanel.tsx:59-85`；`ApprovalPanel.module.css:8-29,56-66`） | 淡 accent 底 + `Radius(12)` 的小卡，13px 粗标题，参数用普通文本，按钮是默认样式（`interaction/card.go:111-116,130-145,149-185`） | 危险/确认语义全靠正文，没有“待批准”这一层的视觉重量 | P1 | 加 warn 色条 + StateDot + 主/次按钮；命令参数走等宽字体；Enter/Esc 已有（`approval_keys.go:34-64`）保持 |
| 15 | 提问卡 | eyebrow + `h2` 标题 + 图标按钮（最小化/关闭）+ 选项行（序号徽章、推荐徽章、描述、选中态）+ 自定义作答 + footer 翻页器 `1 / 3` + 一个主按钮（`QuestionComposer.tsx:438-445,498-538,559,590-608`） | 标题 + 框架默认 `Checkbox` 列表（描述另起 11px 行）+ 一排文字按钮：收起/查看答案摘要/上一题/下一题/跳过/下一步，进度是 11px `"1 / 3"` 文本（`interaction/question_editing.go:71-107`；`question_navigation.go:23-76`） | 没有选项卡与选中态，动作区是 6 个平级文字按钮 | P1 | 选项做成可选卡片（序号 + 标签 + 描述 + 推荐徽章）；动作区收敛为“翻页器 + 主按钮” |
| 16 | 计划审批 / plan review | 独立面板卡：strip + `h3` 标题 + 摘要 + 错误位 + 右对齐 outline（讨论）/ primary（批准）按钮（`ui-user-questions/src/client/PlanReviewPanel.tsx:58-86`） | 卡片内直接内联 `planpreview.View` + “讨论修改/批准计划”文字按钮（`plan_review.go:19-71`） | 结构接近，但按钮层级与卡片样式同 #14 | P1 | 与 #14 共用审批卡外壳；批准按钮用 primary |
| 17 | 用户附件（文件） | 附件行渲染文件卡：类型图标 28px + 文件名 + “EXT · size”，240×64、`radius-xl`、0.5px 边框（`MessageItem.tsx:199-224`；`MessageItem.module.css:320-364`） | 没有任何文件卡渲染路径：durable `session.Message` 无附件字段（`internal/store/session/composer_submit.go:18` 的 `ComposerAttachment` 只服务 composer/队列），正文只按图片引用 + 文本处理（`user_message.go:18-40,44-122`） | 用户发了 PDF/zip，气泡里只剩文件名文本 | P1 | 解析 file 附件块并渲染文件卡（可复用 composer 侧附件显示） |
| 18 | 图片 | 单图按自然宽高比（clamp 0.25–4）最大 240×240，`radius-xl` 边框 + hover 底色 + 失败态 danger 卡 + spinner；多图 64px 瓦片；点击开 lightbox（`MessageImage.tsx:52-63,108-110`；`MessageImage.module.css:18-32,46-58,71-82`） | 固定 `Width(360) × Height(220)` + `Fit(Contain)` + 下方 11px 文件名；无圆角、无边框、无加载/失败态、多图逐张纵向堆叠（`image.go:176-191`） | 每张图都被塞进同一个 16:10 盒子（大量留白），聊天流被图片切碎 | P1 | 按图片元数据算宽高比并限制最大边；加圆角/边框/hover；多图走瓦片 |
| 19 | 本地乐观回显 | 提交后立刻以**和最终消息一致的气泡**回显（含气泡底色、时间、动作条），`data-submission-echo`（`MessageItem.tsx:277-321`） | 纯文本 + 附件名 + 一行状态 `发送中 / Sending…`，无气泡（`pending_submission.go:9-36`） | 用户刚发出的消息“样子会变两次” | P2 | 复刻 user bubble 组件，回显阶段直接复用；状态改成气泡下方的小字 |
| 20 | 队列 / 插话 | 队列 dock 每条一行：preview 文本 + 图标按钮（保存/取消/编辑/删除/插话，`aria-label` 而非文字）；插话（steering）在 transcript 里以同款气泡 + 标记出现（`ui-conversation/src/client/queue/QueueDock.tsx:178-304`；`MessageItem.tsx:246-267`） | 只在“计划板”展开后以文本行 + 编辑/插话/删除 三个文字按钮出现；插话只改变状态串“插话 · 发送中…”（`planboard.go:121-158`；`pending_submission.go:32-34`） | 队列不可见（要开面板），动作是三个文字按钮，插话没有气泡标记 | P1 | 队列放回 composer dock 且动作用图标；steering 走 user bubble + 角标 |
| 21 | todo / goal 面板 | todo 在 composer dock（图标 + 13/24 标题 + 计数 + chevron，展开后 StateDot 行）（`TodoPanel.tsx:69-97`）；goal 是 36px 半透明菜单面 strip + 28px 图标按钮 + InlineEditor（`GoalBar.module.css:19-41,113-131`） | 两者被塞进底部“计划板”里做小节标题（10/11px），todo 用 `☑ ⌃ ⌄` 文本 + 写死 `rgba(255,255,255,0.72)` 背景，goal 用 5 个 11px 文字按钮（`todopanel.go:44-91`；`planboard.go:100-118,185-258`） | 位置、字号、控件都不对；goal 需要先开面板才看得到 | P1 | todo/goal 各自回到 composer context stack，用图标 + 计数 + 图标动作 |
| 22 | 回合导航 | 右侧浮动“回合轨”：28px 宽、10px 刻度、悬停出预览卡、渐隐上下端（`TurnNavigator.module.css:21-46,80-100`；`TurnNavigator.tsx:178-232`） | 顶部一行 `ui.Select` 下拉（“回合目录”）+ 取消跳转/重试/回到最新 文字按钮（`history_navigation.go:64-138`） | 导航从“空间定位”退化为“表单选择” | P2 | 用绝对定位轨 + hover 预览实现；下拉保留作为窄屏回退 |
| 23 | 用量行 | 动作行里的胶囊“用量 5.2M tok · 10:40”（截图 x≈666-745,y≈431）+ 展开 `dl` 明细（`TurnUsageDisclosure.tsx:33-52,56-80`） | 正文里一行 11px 文本 `▸ 本轮用量 / Turn usage · N tokens · 缓存命中 / Cache x%`，展开后 5 行未分组的数字文本（`performance_usage.go:25-61`） | 数字没有千分位/紧凑格式（`5.2M`），且挤在正文流里 | P2 | 做成动作行胶囊；数字走 `formatTokens` 等价格式；明细用 dt/dd 两列 |
| 24 | 消息间距与阅读列宽 | 滚动区 `padding:16px` + 两侧 clearance+16；列宽 = content width（+32 给 composer）；行间距 6/12/16（`ChatView.module.css:22-34,54-70,75-101`） | 每条消息再套 `Padding(12,16,8,16)`，列宽 = content width 但正文被再缩 32px；行距固定 ≈20px；右侧额外 37px 滚动条沟（`chat.go:155,360-371`） | 阅读列比原版窄 32px、行距与原版节奏无关 | P2 | 把 16px 侧内边距移到列容器（只算一次），并按“过程行 6 / 回答 12 / 回合头 16”分级 |
| 25 | 轨迹台账 | 真实 `<table>`：30px 粘性 `th` 表头（sidebar 底色 + 0.5px 下边线）、事件列 122/84px（中文）+ 自适应内容列、`col-resize` 拖列宽、虚拟占位行、插入历史的“加载更早”行（`TrajectoryTable.tsx:2610-2659`；`TrajectoryTable.module.css:114-140,766,876`） | `ui.List` 行，固定 30px 的按钮行：角色列（84/122px）+ 单行预览 + 时间；无表头、无列宽手势（`trajectory.go:174-194`；`trajectory_record_view.go:13-29`） | 台账没有表头与列宽手势，列定位依赖固定像素硬编码 | P2 | 补 30px 粘性表头与列宽手势；分组行（回合/请求）用粘性标题 |

> 统计：25 条（P0 6 条：#1/#3/#4/#11/#12/#13；P1 14 条；P2 5 条）。

---

## 三、待验证项

1. **需要一张“有消息”的原生截图**。证据截图是空会话，消息气泡/正文/工具卡的最终像素无法核对。建议用测试快照（`internal/ui/chat/software_snapshot_test.go:70` 已能生成 `chat-composer-<width>-dark-<bool>-attachments-<bool>` 的软件快照）或新截一张 3090 的多轮会话图，再校验本文 P0-1/P0-2 的实际观感。
2. **`ui.List` 的行高与虚拟化能力**：原生消息列表用 `ui.List(c, list, len(msgs))` + `FollowEnd`（`chat.go:145-152`）；原版是 `use-chat-viewport` 自研虚拟化 + `use-scroll-follow` 锚定（`ui-chat/src/client/chat/use-chat-viewport.ts`、`use-scroll-follow.ts`）。没查到“长列表（>500 行）滚动是否掉帧、插入历史时是否跳动”的直接证据，待实测。
3. **轨迹 tab 的细节**（`version-drift` 风险）：原版 rc.2 的 `TrajectoryTable.tsx` 有 3500+ 行、含工具栏视图切换/搜索索引/分组与虚拟占位；原生只有工具栏（时长/回合/调用折叠 + 160px 搜索框）、概览条、台账 + 检查器（`trajectory.go:115-320`）。逐项对照（列宽拖拽、搜索结果高亮计数、行分组、请求头 tab）需要再用运行版截图核实。
4. **`version-drift` 候选**：本文引用的 `RunningStatus`（深度思考计时）、`TurnNavigator`（回合轨）、`StatsPills`/`TurnUsageDisclosure`（用量胶囊）在 alpha.1 源码中均存在，但运行版为 rc.2，其文案/尺寸可能已改；排期前应到 `C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\ui-chat\lib\client.js` 抽查一次字符串与 token。
5. **消息进入/更新动画**：原版 `hero → 对话` 的 300ms 位移已经有原生实现（`internal/ui/frame/conversation_enter.go:10-17,119-172`），但“新消息逐条出现”“流式追加时的重排抑制”在原版源码中没有找到对应的 `@keyframes`（仅 `TextShimmer`/`StateDot`/`TypingFxLayer`），因此**不下结论**；若用户感知到“原生打字更抖”，应在 `ui.List` 增量更新路径上单独排查。
6. **审批卡的键盘可达性**：原生已实现 Enter=允许 / Esc=拒绝、含 IME 组合态与 `button` 焦点排除（`interaction/approval_keys.go:19-86`），与原版 `ApprovalPanel.tsx:46-58` 语义一致；此项**不算差距**，但审批卡的 `aria`/焦点环需要在真实键盘走查中确认（原生用的是 `Label("Approval details: …")`，见 `card.go:115`）。
