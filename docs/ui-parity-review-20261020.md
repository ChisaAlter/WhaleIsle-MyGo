# MyGo鲸屿 原生重写 vs 原版鲸屿：全面对比审查（2026-10-20）

> 审查对象：`C:\AI\WhaleIsle-MyGo`（Go + MyGo 原生 UI）
> 对照基准：运行中的 Electron 版鲸屿（3080）+ 其前端源码（alpha.1 可读源码 / rc.2 构建产物）
> 分册：[01 外壳与导航](review-20261020/01-shell-nav.md) · [02 对话与消息](review-20261020/02-conversation.md) · [03 输入区](review-20261020/03-composer.md) · [04 功能界面](review-20261020/04-surfaces.md) · [05 视觉与特效](review-20261020/05-visual-system.md)

> **修复进展（2026-10-20，Wave 1-4）**：本报告 103 条里的 P0 已全部有实现，Wave 4 补上了最后两个系统性缺口：**真实 DPI**（runtime 把 host 设备像素归一成 DIP + `host.pixels()` 能力位；150% 下 1440/1280/960 px 实机截图里 hero 徽章与 composer 卡均完整）和**材质可读性**（玻璃对比度守卫：4 组合正文最低 7.85:1、次要 3.01:1；菜单/市场/设置 hairline 改 DrawOver）。本地化：UI 284 + store/desktop 375 个纯中文 key 英文化，双语双写保持 0，剩余 19 处均为 Excel 本地化日期解析表与品牌名。文件标签 "+" 的真实 picker 已接。已落地明细见 [fix-plan](fix-plan-20261020.md)，截图见 `review-20261020/evidence/after/`。未关闭：同步式生命周期 MessageBox（卸载/更新/任务保护）的异步化迁移、实机深色+壁纸同条件对照、100/125% DPI 与多显示器验收、消息级组件实机走查。

## 0. 一句话结论

**不是"功能没做完"，而是"做成了另一种产品形态"**：数据结构、接口、大量几何/色值已经 1:1 对齐（气泡色、宽度轴、发送盘 34px、输入卡 80% 填充都能对上），但**设计系统这一层没有搬过来**——原语（DisclosureRow / StateDot / Switch / 菜单材质 / motion token）、块级排版、统一本地化、应用内对话框。于是每个界面都"差一档"，累积成用户说的"差距很大"。

## 1. 数量总览

| 分册 | 条目 | P0 | P1 | P2 |
| --- | --- | --- | --- | --- |
| 01 外壳与导航 | 22 | 3 | 10 | 9 |
| 02 对话与消息 | 25 | 6 | 14 | 5 |
| 03 输入区 | 17 | 2 | 5 | 10 |
| 04 功能界面 | 25 | 6 | 17 | 2 |
| 05 视觉与特效 | 14 | 2 | 5 | 7 |
| **合计** | **103** | **19** | **51** | **33** |

去重后（hero、本地化、材质等跨册重复）约 **90 条差距、14 条 P0**。全部条目在两个分册里都能落到 `file:line`，没有"感觉上"的结论。

## 2. 五条根因

### 根因 1：设计系统原语没有搬（影响面最大）

原版所有次要行、状态、折叠都由一批原语承载：`DisclosureRow`（图标 + 标题 + 圆点 + 摘要，整行可点，24px 行高）、`StateDot`（idle/running/done/error 四态）、`TextShimmer`（运行中扫光）、`Switch`（36×20）、菜单面（`--dsw-menu-backdrop-filter`）。原生几乎全部退化为**文字按钮 + ASCII 箭头**：`"▸ 思考"`、`"▸ 参数 / Arguments"`、`"▸ 收起问题"`、`"▸ 本轮用量"`（`internal/ui/chat/thinking.go:32-45`、`tool_lifecycle.go:127-137`、`performance_usage.go:35-45`、`interaction/question_navigation.go:25-34`）。

连锁后果：工具调用变边框卡片、设置变字段表单、状态变英文裸词、悬停/按下无反馈。**这是"像原型不像成品"的总开关。**

### 根因 2：本地化退化为「中文 / English」双语硬拼

`internal/ui` 里 **955 处** 双写字符串（surfaces 扫描），其中 chat + interaction 两包 28 个源文件就有 **176 处**；settings 57、sessioncost 23、im 19、usage 12。明明有 `internal/ui/i18n` 双字典却没用，用户在任何语言下都看到两种语言叠在一行（例：`含子代理计费 Token / Billed tokens incl. subagents · 命中 / Hit 0`，`internal/ui/sessioncost/view.go:65,83`）。

这是**最廉价、覆盖面最广**的观感差距，也是一次机械改动能全站受益的部分。

### 根因 3：系统控件混进主题化窗口

64 处危险操作走 `mygo.Dialog.Message` 系统消息框（`cmd/whaleisle/main.go:5007-5018` 设置替换、`git_push_review.go:39`、`task_protection.go:72` 等），一屏宋体系统弹框就把主题化观感打断；布尔设置用 `ui.Checkbox(c, checked, "")` 空标签勾选框（`internal/ui/settings/settings.go:406-414`）。原版全部是应用内 Modal / RiskConfirmation / Switch。

### 根因 4：首屏与外壳的"整体感"

- 空态 hero 没接回：只剩一行 13px 灰字 + 贴底输入卡（`internal/ui/chat/chat.go:140-143`），并且把 store 内部串 `workspace ready: <title>` 画进正文（`internal/store/session/workspace_rpc.go:37` → `chat.go:396-398`）。
- 标题带结构错：52px caption + 13px 标签行两行（`frame/workspace_caption.go:31,62-66`），原版 rc.2 是单行 48px；无标题时直接拼 `New chat · session-`（`internal/store/session/title.go:17-24`）。
- 侧栏品牌锁小一档（32/15-400 vs 38/18-700），左下**没有账号区**（`sidebar/sidebar.go:144-156,287-302`）。
- boot 从整窗海洋场景退化为左侧文本表单（`internal/ui/boot/boot.go:55-101`）。

### 根因 5：材质与动效是"近似"，不是"同一套"

原版是透明窗口 + 壁纸 + 每层 alpha token + 菜单 `backdrop-filter: blur(40px) saturate(150%)` + `motion.css` 统一 420ms/150ms 过渡；原生把壁纸**预模糊成一张 bitmap**，再给每个表面固定 alpha（`internal/ui/chrome/background.go:23-73`），没有表面级背景模糊，也没有 motion token。指针特效、金属漆只有开关没有渲染（`settings/pointer_effects.go:14-16`、`wallpapers/preferences.go:10`）；composer beam 只在响应运行中绘制（`composer/composer.go:167`）。

## 3. 14 条 P0（去重后，建议按此排期）

| # | 域 | 问题 | 关键证据 | 修复方向 |
| --- | --- | --- | --- | --- |
| 1 | 外壳 | 空态 hero 未接回，settling 阶段空白 | `chat/chat.go:140-143`；`frame/conversation_phase.go:32-53` | 新建 hero 组件（鱼标 + 26/32 标题 + workspace chip + 居中限宽 composer） |
| 2 | 外壳 | 内部状态串泄漏正文 | `store/session/workspace_rpc.go:37` → `chat.go:396-398` | store 层过滤内部短语，只放用户向文案 |
| 3 | 外壳 | 侧栏品牌锁偏小 + 无账号区 | `sidebar/sidebar.go:144-156,287-302` | 按 Brand 18/700 + 38px 头像重刻；footer 加账号行与菜单 |
| 4 | 外壳 | 标题带两行、标题泄露 session id、中英混排 | `frame/workspace_caption.go:31,62-66`；`store/session/title.go:17-24` | 单行 48px；未命名回落本地化"新会话" |
| 5 | 对话 | Markdown 块级排版消失（标题/列表/引用/表格） | `widgets/markdown/blocks.go:21`；`markdown.go:274-352` | 按块类型输出外边距/行高/引用线/自动列宽 |
| 6 | 对话 | 工具调用是边框卡片 + 原始 JSON | `chat/toolcard.go:138-153` | 单行 DisclosureRow + 终端/差异/读取/检索专用 body |
| 7 | 对话 | 缺设计原语（DisclosureRow/StateDot/Shimmer） | `chat/thinking.go:32-45` 等 | 建 `widgets` 原语层，批量替换文字箭头 |
| 8 | 对话 | 代码块固定浅色 chroma、文字按钮、无行号/复制反馈 | `widgets/markdown/blocks.go:97-131`；`highlight.go:20-31` | 深浅两套 chroma、图标工具条、复制态 |
| 9 | 全局 | 955 处「中文 / English」双写（chat 176） | `sessioncost/view.go:65,83`；`chat/tool_lifecycle.go:71-83` | 迁 i18n + grep 门禁 |
| 10 | 设置 | 行式布局变字段堆叠；布尔项空标签勾选框 | `settings/settings.go:383-414` | 统一 `bodyPreferenceRow` 原语 + Switch |
| 11 | 文件 | 多标签变竖排文字按钮（▸/• 前缀） | `files/tabs.go:316-339` | 28px 横向 chip 条：active 底、hover 出 close、可拖拽 |
| 12 | 差异 | Diff 只有前景色，无行底色/行号槽/hunk 头 | `diffsurface`；对照 `DiffPanel.module.css:76-120` | 行 20px、gutter 12px、±8% 行底色、hunk 圆角头 |
| 13 | 用量 | 诊断文本墙替代 KPI/热力图/环图 | `usage/stats.go:48-105`；`chart.go:19-49` | KPI 卡 + 贡献日历 + 环图 + 排行卡 |
| 14 | 视觉 | 假玻璃 + 默认外观不对（浅色、无壁纸） | `chrome/background.go:23-73`；`.devdata/config.json:21,43` | 表面级背景模糊（或分层合成）+ 默认深色壁纸 |

> 输入区两条 P0 并入上表 #9/#14 的范围内：#1 dock 状态行双语调试串、#2 权限 pill 直接显示 `workspace-write`（证书 `composer/preset_picker.go:16-24`）。

## 4. 优先级路线

### Phase A：观感翻身（建议先做，2-3 周单人）

1. **设计原语包**（5-7 天）：`DisclosureRow`、`StateDot`、`Switch`、行式设置原语、应用内 `ConfirmDialog`、菜单面材质、motion token。做完 7/10/12 三条 P0 会一起松动。
2. **首屏三件套**（3-4 天）：hero 接回（#1/#2）、标题带 48px + 本地化（#4）、品牌锁 + 账号区（#3）。
3. **本地化清理 + 门禁**（2-3 天）：955 处批量迁移，加"禁止 `" / "` 双写"的 grep 门禁；同时清掉 `workspace ready:` 与英文裸词。
4. **材质与默认外观**（4-6 天）：表面级背景模糊或分层合成、菜单/弹层模糊、默认深色 + 壁纸；指针特效与金属漆要么实现要么隐藏开关。

### Phase B：结构还原（2-3 周）

工具流（#6/#7 + 结构化 body）→ Markdown 块级 + 代码块（#5/#8）→ 设置行式（#10）→ 文件 chip 标签（#11）→ Diff（#12）→ 用量仪表盘（#13）→ 终端标签、Git 提交对话框、64 处系统 MessageBox 迁移。

### Phase C：打磨（约 2 周）

动效 token 全站落地（hover/pressed/弹层进出/peer 过渡）→ 指针特效 trail/splash、金属漆、beam idle 态 → 窄宽抽屉、滚动条、DPI 三档 → boot 场景、launcher 尺寸统一。

### 性价比排序（如果只能做一件事）

**先做本地化（#9）+ 原语层（#7）**：两项都不改布局，但能让 40+ 条 P1/P2 同时改善，是"看起来像成品"的最短路径。

## 5. 版本漂移与"假缺口"（排期前必读）

- 原生按 **alpha.1** 源码还原，线上产品是 **rc.2**。已确认的漂移：
  - rc.2 会话头部只有一行（无「对话/轨迹」标签行）；原生按 alpha.1 复刻了两行（01 #6）。
  - rc.2 `ContextMeter` 在 trailing 行，alpha.1 在 dock——原生位置与 rc.2 一致，修样式时不要挪回去（03 待验证 8）。
  - rc.2 新增 `ui-deepseek-model-label-1017`、`ui-secret-mask`、`ui-plugin-market`；原生对应能力可能按旧包实现。
- 已核对**不算差距**的项（避免返工）：气泡色/宽度轴/发送盘 34px/输入卡 80% 填充/placeholder 文案与色值/审批卡 Enter-Esc/输入框 max-height 336px。
- 证据限制：原生截图是**空会话 + 浅色**，原版是**深色 + 壁纸 + 运行中**；消息级、beam、hover 的最终像素还没有同条件对照，见各分册"待验证"。

## 6. 验收建议

1. **同条件成对截图**：深色 + 同一壁纸、1440×900 @1.5x、同一会话状态（空态 / 3 轮对话 / 工具展开 / 运行中），每个 Phase 前后各一组。
2. **两条自动化门禁**：i18n 双写 grep；`mygo.Dialog` 白名单（只允许文件选择/卸载等系统语义）。
3. **一条结构门禁**：`internal/ui` 禁止出现 `"▸ "`、`"• "`、`"▸"` 等字符状态（改走控件属性），防止新代码继续复制反模式。

## 7. 分册索引

| 分册 | 内容 | 条目 |
| --- | --- | --- |
| [01-shell-nav.md](review-20261020/01-shell-nav.md) | 窗口外壳、caption、侧栏、会话行、标题带、boot、启动器、窄宽 | 22 |
| [02-conversation.md](review-20261020/02-conversation.md) | Markdown、代码块、工具卡、思考、审批/提问、附件、队列、todo/goal、轨迹 | 25 |
| [03-composer.md](review-20261020/03-composer.md) | 输入条几何、发送/停止、权限/模型 chip、`/` `@` 弹层、dock 状态行、附件条 | 17 |
| [04-surfaces.md](review-20261020/04-surfaces.md) | 设置、文件/标签/编辑器、Diff、终端、Git、用量、市场、远程、定时、Project、桌宠 | 25 |
| [05-visual-system.md](review-20261020/05-visual-system.md) | 玻璃/材质、token、字体、动效、指针特效、金属漆、beam、滚动条、DPI | 14 |
| [README.md](review-20261020/README.md) | 审查方法、基准、模板、局限 | — |

证据截图：[原版 3080](review-20261020/evidence/original-electron-3080.png) · [原生 3090](review-20261020/evidence/native-go-3090.png)
