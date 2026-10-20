# 04 · 功能界面（设置 / 文件 / 终端 / Git / 用量 / 市场 / 远程 / 定时 / Project / 桌宠）

审查日期 2026-10-20 · 审查者 surfaces-reviewer · 方法见 [README.md](README.md)

本轮范围：`settings`、`files`、`diffsurface`、`terminal`、`usage`、`market`、`pluginmanager`、`remote`、`schedule`、`projects`、`im`、`browserpreview`、`sessioncost`、`desktop/pet`。
共 **10 条致命差距 + 25 条明细 + 8 条待验证**。

## 证据约定

为表格可读，路径使用以下缩写（首次出现处均给出完整路径）：

| 缩写 | 实际路径 |
| --- | --- |
| `client/` | `runtime\vendor\deepseek-harness\packages\client\` |
| `vendor/` | `runtime\vendor\` |
| `dsh-remote/` | `runtime\vendor\dsh-remote\plugin-src\client\` |
| `pet-live2d/` | `reference\electron\src\renderer\` |

- 原版 = alpha.1 可读 TSX/CSS（`client/*/src`）+ 安装版 rc.2 的包清单；原生 = 当前源码树 `internal\ui\*`、`internal\desktop\pet`。
- 原生侧一律 Go 文件 UTF-8 读取，行号取自 `read`/`[IO.File]::ReadAllLines`（1 基）。
- 未取到源码证据的（无法定位原版实现）一律进"三、待验证"，不写进明细表。

**版本提示（version-drift）**：`ui-plugin-market`、`ui-deepseek-model-label-1017`、`ui-secret-mask`、`ui-visual-replies` 只存在于安装版 rc.2 的 `client/` 清单，不在 alpha.1 源码树。本文件中市场条目以 `ui-settings-market`（alpha.1 有）为准。

---

## 一、最致命的 10 条差距

### 1. 设置页的行式布局被"字段堆叠表单"整体取代（P0）

原版每一条设置是**一行**：左列标题 14px/22 + 描述 12px/18 tertiary，右列控件（Switch 或右对齐 SettingsSelect），行内 `padding:16px 0`，行间 `border-bottom:0.5px solid --dsw-alias-border-l2`，末行分隔线由 section 自己剥掉。

- 证据：`runtime\vendor\deepseek-harness\packages\client\ui-settings-general\src\client\CloseBehaviorRow.module.css:3-32`（row / rowText / title / desc 全套）、同目录 `GeneralSection.module.css:10-12`（末行去线）。
- 原生：通用命名空间页走 `fieldRow`，把标题行、重置按钮、描述、控件**纵向堆**在一起，没有行分隔线、没有 16px 行高节奏。
- 证据：`internal\ui\settings\settings.go:383-399`（`ui.Column.Gap(4)` + 标题 12px + 右侧"重置为继承值" + 描述 11px + 控件在下一行）。
- 影响：**每一个**非专属分区（`general`、`permission`、`agent-loop`、`web-search`、`subagent`、`shell`、`conversation`、`session-log` …，见 `settings.go:226-272` 的分支表）都变成一张 JSON 表单，用户在设置里看到的是"调试表单"，不是设置页。这是"界面差距很大"最直接的来源。
- 修复方向：把 `bodyPreferenceRow`（已实现，`internal\ui\settings\body_preferences.go:33-73`，行内 16/0 padding + 0.5px 分隔线 + 180px 右列）提升为**唯一**设置行原语，`fieldRow` 改为调用它；布尔/枚举/文本分别落到 Switch / SettingsSelect 等价物 / 行内输入。

### 2. 布尔设置用无标签复选框，且控件在标签下方（P0）

- 原版：`Switch` 36×20、`padding:2px`、`border-radius:999px`、选中品牌色、thumb 位移 16px、`aria-checked` 驱动外观；行内右对齐。
  证据：`client/ui-primitives/src/Switch.module.css:5-21`、`:70-99`、`Switch.tsx:55-71`。
- 原生：`ui.Checkbox(c, checked, "")` —— 第三参数是**空字符串**，控件自带标签为空，标题只在上方 12px 文字里，控件独占一行。
  证据：`internal/ui/settings/settings.go:406-414`。
- 视觉后果：开关不是"开关"，是表单勾选框；同一屏里多个勾选框独占行、挤在左列，右侧整片空白。
- 修复方向：`settings` 包内加 `switchRow(...)`（`ui.Switch` 已可用，正确用法见 `body_preferences.go:150`、`pet.go:117`），`fieldRow` 的 `FieldBool` 分支改走它并补 `Label`。

### 3. 危险操作全部弹 Windows 原生 MessageBox（P1）

- 原版：所有确认/危险操作是应用内 `Modal`/`RiskConfirmation`，带主题、带说明正文、带主次按钮。
  证据：`client/ui-git/src/client/CommitDialog.tsx:74-95`（Modal + footer 三按钮）、`client/ui-primitives/src/RiskConfirmation.tsx`。
- 原生：设置"完整替换"、Git 推送核对、插件导入、桌宠重置、项目归档、任务保护等全部 `mygo.Dialog.Message(...)`，即系统消息框；`cmd/whaleisle` + `internal` 内共 **64 处**。
  证据：`cmd/whaleisle/main.go:5007-5018`（设置 Replace 确认）、`internal/ui/settings/pet_reset.go:11`、`cmd/whaleisle/git_push_review.go:39`、`cmd/whaleisle/task_protection.go:72`。
- 视觉后果：中文系统对话框（宋体、系统蓝按钮）盖在主题化窗口上，是"Electron 观感 vs 系统工具观感"的分界线；且 `mygo.Dialog.Error("Usage export", …)`（`cmd/whaleisle/main.go:5108-5115`）连标题都是英文。
- 修复方向：在 `internal/ui/chrome` 或 `widgets` 补应用内 `ConfirmDialog`（复用 `ui.DialogBase`，与 `settings/modal.go:21-28` 同款），把应用内语义的确认（设置/插件/Git/项目/桌宠）迁过去；文件选择、卸载器启动保留系统对话框。

### 4. 文件多标签：竖排文字按钮 + "▸/•" 前缀，替代 28px 横向 chip 条（P0）

- 原版：dockkit `TabStrip` —— 28px 高、gap 4、chip `min 80 / max 170 / padding 0 10 / 13px`、active 用 `--dsw-alias-markdown-tag` 填充、关闭按钮**覆盖**在标题最后 24px 且只在 hover/active/focus 出现（宽度不跳动）、标题超宽用 16px 渐变遮罩而非省略号、可拖拽换位、有 `+` 新建 chip、溢出菜单（关闭/复制/分屏）。
  证据：`client/ui-dockkit/src/components/dockkit.module.css:237-248`（strip）、`:357-376`（chip）、`:403-413`（遮罩）、`:443-448`（close 显形）、`:482-485`（active 填充）、`:451-465`（addTab）；`TabPanel.tsx:245`（`role="tablist"`）、`:316-321`（close tooltip）。
- 原生：`paintTabs` 是**竖向** `Column.Gap(2).Padding(4,8,4,8)`，每个标签一个 11px 文字 `Button`，活动态靠字符串前缀 `"▸ "`、脏标记靠 `" •"`、关闭是另一个独立 `"×"` 按钮。
  证据：`internal/ui/files/tabs.go:316-339`（`:321` 纵向列、`:329` 的 `"▸ "`、`:332` 的 11px、`:335` 的 `"×"`）。
- 影响：文件页的主结构（横向标签）不存在了 —— 用户看到的是一列文本；关闭按钮独占行宽、脏标记是字符而不是圆点/加粗，且没有拖拽、溢出菜单、分屏。
- 同源问题：同类"用字符当状态"的写法还有用量排行（`ranking.go:47` 的 `"● "` 前缀表选中，见致命 #7）、设置左栏（无此问题）。这是一条可统一治理的**反模式**：控件状态必须走 `Selected/Checked` 属性，不得拼字符串。
- 修复方向：重写 `paintTabs` 为横向 `Row`：28px chip、active 底色、hover 才出 close、标题单行 + 遮罩溢出、末尾 `+`；脏标记改为 chip 内 8px 圆点。

### 5. Diff 只有前景色，没有行底色、行号槽、hunk 头卡片（P0）

- 原版：每个 hunk 一个圆角头（`bg-layer-2`，上圆角 8），行 `min-height:20px`、`padding:0 8px`、gutter 12px 数字列，`+`/`-` 行用 `color-mix(状态色 8%)` 铺底，等宽字体。
  证据：`client/ui-diff/src/client/DiffPanel.module.css:76-120`（`.hunks` 代码字体、`.hunkHeader` 圆角、`.line` 20px、`.gutter` 12px、`.add`/`.del` 8% 底）。
- 原生：`diff.Spans` 只给每行一个 `ui.Span{Color:…}`，整份 patch 塞进一个 `RichText`；没有行容器，因此**无法**有行底色、行悬停、行号槽、hunk 卡片。
  证据：`internal/ui/widgets/diff/diff.go:78-92`（仅 color）、`:96-106`（单块 RichText）；`internal/ui/diffsurface/patch_view.go:133-151`（split 模式也只是给文本上色）。
- 视觉后果：diff 页读起来像彩色终端输出，而原版是带色块分层的审阅视图 —— 这是"专业感"差距最重的一屏。
- 修复方向：新增 `widgets/diff/rows.go`，按行产出 `ui.Row`（gutter 12px + 文本），`+`/`-` 行加 `Background(状态色 8%)`；hunk 头单独一行带 `Radius(6)` 与 `bg-layer-2`；`patch_view.go` 双栏复用同一行原语。

### 6. 设置导航多了一个"实现命名空间"分区，文案也不同（P1）

- 原版导航 = `settings.section` 账本的投影：13 项（登录后 +账号 = 14 项）、无分组标题、无搜索框，图标逐项不同 —— 账号(-10，仅登录后)、通用(0)、外观(5)、界面(6)、模型(10)、插件(15)、技能(16)、市场(17)、远程(18)、MCP(18)、智能体预设(20)、用量统计(25)、桌宠(40)、关于(90)。
  证据：`client/ui-settings-general/src/client/index.ts:297-379`（订单值）、`client/ui-settings-mcp/src/client/index.ts:49-56`、`client/ui-settings-skills/src/client/index.ts:67-74`、`client/ui-settings-plugins/src/client/index.ts:76-84`、`client/ui-settings-models/src/client/index.ts:175-185`、`client/ui-settings-market/src/client/index.ts:69-76`、`client/ui-settings-remote/src/client/index.ts:126-134`、`client/ui-theme/src/client/index.ts:1086-1095`、`client/ui-agent-preset/src/client/index.ts:264-271`、`vendor/dsh-usage-panel/src/client/index.tsx:50-55`（order 25）；导航渲染 `SettingsRoot.tsx:105-124`（只有 title + rows，**无搜索框**）、图标表 `:39-54`。
- 原生：15 项，多出 `远程工作区`（原版只是"界面设置"里的一行 `remoteWorkspace.title`，见 `client/ui-settings-general/src/client/locales.ts:55`），且账号/通用/桌宠/远程工作区四项共用同一个齿轮图标。
  证据：`internal/ui/settings/navigation.go:81-95`（`:94` 的 `add(SectionRemoteWorkspace, "SettingsOutline", …)`；`:81`、`:93` 同样用 `"SettingsOutline"`）、`navigation_icons.go:6-96`（13 个正确图标都在表里，映射选错）。
- 文案不一致：原版 `通用设置/界面设置/智能体预设/鲸鱼娘/用量统计/账户` ↔ 原生 `通用/界面/Agent 预设/桌宠/用量/账号`（`internal/ui/i18n/i18n.go:1493-1508` 对 `client/ui-settings-general/src/client/locales.ts:29,35,58`）；面板标题原版"设置"（`locales.ts:25`）↔ 原生"Harness 设置"（`internal/ui/i18n/i18n.go:1252`）。
- 修复方向：删掉 `SectionRemoteWorkspace` 一级入口、回到 Interface 的行；`navigation.go` 图标改为按 id 映射既有图标表；对齐 5 个文案键与面板标题。

### 7. 用量页是"诊断文本墙"，原版是可视化仪表盘（P0）

- 原版：KPI 卡（总/输入/输出/会话数/命中率/费用，`useCountUp(…, 900)` 数字滚动）、GitHub 贡献日历（周列、月标签、Mon/Wed/Fri 行标签、全窗四分位分级、点选 + Shift 区间、hover tooltip 含当日费用）、模型环图、项目排行卡、会话卡、导出菜单、计费设置弹窗。
  证据：`vendor/dsh-usage-panel/src/client/components/KpiCards.tsx:59-64`、`Heatmap.tsx:1-7`（设计说明）、`:34`（LABELED_ROWS = Mon/Wed/Fri）、`:43-46`（四分位阈值）、`:51-56`（区间选择）、`:58-80`（tooltip 含费用）；同目录 `ModelDonut.tsx`、`ProjectRankCard.tsx`、`ExportMenu.tsx`、`BillingSettingsModal.tsx` 均存在。
- 原生：几乎全是 `ui.Text` + `fmt.Sprintf` 的信息行，包括 `Coverage: %d/%d · Failed %d · Pending %d · %s · UTC`、`Events %d · Retries %d · Compaction %.0f tokens · Main %d · Subagent %d`；日期范围是一排 `"7 days"/"14 days"/"30 days"` 按钮（中文界面里直接是英文）；导出是四个 `"导出 json/days/models/sessions"` 按钮；排行退化为 `model · provider · 12345 tokens` 纯文本行，排序选中态用 `"● "` 前缀拼字符串。
  证据：`internal/ui/usage/stats.go:48-52`、`:55`、`:70-79`、`:87-97`、`:100-105`；排行 `ranking.go:12-30`（`:22` 文本行、`:27` "已显示 %d / %d"）、`:39-56`（`:47` 的 `"● "` 前缀选中态）。
- 图表：柱图是 22px 宽纯色方块（无坐标轴、无 tooltip、无网格），热力图是**单行** 18×18 方格 + 9px 标签（不是 7×N 周历）。
  证据：`internal/ui/usage/chart.go:19-49`（`chartH=96`、`Width(22)`、10px 标签）、`heatmap.go:16-51`（`ui.Row` 单行、`Size(18,18)`、9px）。
- 修复方向：统计细节收进"诊断"折叠区；补 `usage/kpi.go`（四张卡 + 数字滚动）、`usage/calendar.go`（7 行周历 + 月标签 + 四分位配色 + 点选区间）、`usage/donut.go`；日期范围改分段控件，导出改单个菜单按钮。

### 8. 全局文案中英双写，未走 locale 路由（P0）

- 原版：所有产品文案在 typed 字典里（`locales.ts`），组件只拿 `t()`；仓库规则明确禁止硬编码（`runtime\vendor\deepseek-harness\packages\client\AGENTS.md` "Client UI copy is locale-owned"）。
- 原生：`internal/ui` 里共 **955 处** 形如 `"保存只提交已修改字段… / Save submits only changed fields…"` 的双语字符串（扫描口径见附 B；其中含少量注释与路径误计，量级不受影响），运行时两种语言同时打印。按本审查范围：settings 57、sessioncost 23、im 19、usage 12、schedule 12、remote 11、projects 5、files 2、terminal 2、market 2。
- 代表证据：`internal/ui/settings/settings.go:322`、`:351`、`:355-359`；`internal/ui/schedule/timing_picker.go:199`、`:215`、`:292`；`internal/ui/remote/phone.go:143`、`:146`、`:148`；`internal/ui/usage/stats.go:83-85`。
- 后果：中文用户每屏都能看到英文尾巴，英文用户看到中文前缀 —— 比"翻译不准"更伤，因为它在排版上就是噪音（行高、换行、按钮宽度全被撑坏）。
- 修复方向：新增 `i18n.T2(zhKey, enKey)`（或批量生成 key）机械替换这 955 处；本轮优先 settings / usage / remote / schedule / im / sessioncost。

### 9. Git 提交流程是内联一排文字按钮，原版是 520px 提审对话框（P1）

- 原版：`CommitDialog` 宽 `min(520px,100%)`，内含分支行（分支名 13/500 + 默认分支告警）、大文件告警卡、文件清单（可选复选、每行 `+N/-N`、可点击打开文件）、合计 `+/-`、大号 message textarea、footer 三个按钮（取消 / 提交到新分支 / 提交）。
  证据：`client/ui-git/src/client/CommitDialog.module.css:1-13`（520 + review 卡）、`CommitDialog.tsx:69-70`（合计）、`:74-95`（Modal + footer）、`:150-179`（文件行 + `+/-`）、`:191-199`（message）。
- 原生：整个 Git 工作台是文字按钮栈 —— 范围切换 `"全部修改（对比 HEAD）"/"全部未暂存"/"全部已暂存"`，每个文件行下面挂 10px 文字按钮 `"丢弃此文件…"/"暂存此文件"/"查看已暂存"/"取消暂存"`，提交说明是一个内联 `TextInput`，动作是 `"核对所选文件并提交…"`。
  证据：`internal/ui/diffsurface/workbench.go:143-196`（`:166` 路径按钮 11px、`:170-193` 10px 动作）、`commit_selection.go:252-353`（`:311` 内联输入、`:346-353` 三个"核对…"按钮）。
- 修复方向：把 `commit_selection.go` 的核对态搬到 `ui.DialogBase` 弹层（520px），文件清单用行 + 右对齐 `+N/-N` + 末行合计。

### 10. 终端没有标签条：单 pane 无标题，多 pane 用 144px 右侧竖列表（P1）

- 原版：终端在 dock 标签里（28px chip、OSC 标题），工具条是绝对定位的 28px 圆角药丸（`top:8 right:8`，0.5px 边框、`radius: md(12)`、内部分隔线 0.5px、每个动作 28×28），字体 13px/20px 等宽。
  证据：`client/ui-user-terminal/src/client/TerminalWorkspace.module.css:11-13`（13/20 代码字体）、`:25-66`（toolbar 药丸 + 28px action + `.rule`）；标题 `client/ui-sidebar-terminal/src/client/TerminalTitle.tsx`；OSC 颜色继承 `terminal-theme.ts:19-64`。
- 原生：单 pane 时**没有标签、没有标题**，只有右上角浮层动作条；多 pane 时右侧多出一条 144px 竖列，里面是 `"组 1"`、`"终端 1"` + `"×"` 文本按钮。
  证据：`internal/ui/terminal/panel.go:298-315`（`:304` 仅 `len(panes)>1` 才画竖列、`:313` 单 pane 浮层）、`workspace_chrome.go:53-89`（`:55` `Width(144)`、`:61` `"组 %d"`、`:77` `"终端 %d"`、`:80` `"×"`）。
- 修复方向：pane 列表改为横向 28px chip（与文件标签共用原语），删竖列；标题优先取 `term.Title()`（已有 `terminalTitle`，`panel.go:147-160`），退化时用 `终端 N`。

---

## 二、明细表

> 严重度：P0 一眼可见的观感断裂 / 每日必用功能缺失；P1 结构或交互明显不同但主流程可用；P2 细节。

| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 设置 · 面板容器 | 居中模态 800×800、`radius-panel`(28)、`box-shadow: --dsw-elevation-prominent`、遮罩 `--dsw-alias-bg-mask-1` + `backdrop-filter`；`client/ui-settings-general/src/client/SettingsRoot.module.css:90-135` | `DialogBase` 800×800、`Radius(28)`、遮罩黑 24%（暗色 50%）；`internal/ui/settings/modal.go:19-28`、`:33-35` | 遮罩用硬编码黑而非主题 mask token；暗色 50% 与主题无关；无 elevation 阴影替身 | P2 | `chrome` 暴露 Mask/Elevation token 并在 `modal.go` 使用 |
| 2 | 设置 · 行式偏好 | 标题左、控件右、行 16/0、0.5px 分隔线、末行去线；`client/ui-settings-general/src/client/CloseBehaviorRow.module.css:3-32`、`GeneralSection.module.css:10-12` | `fieldRow` 纵向堆：标题 12 + 重置按钮 + 描述 11 + 控件下一行，无分隔线；`internal/ui/settings/settings.go:383-459` | 见致命 #1 —— 整页不是"设置行" | P0 | `fieldRow` 改走 `bodyPreferenceRow` |
| 3 | 设置 · 布尔控件 | Switch 36×20 胶囊、选中品牌色、thumb 位移 16、`aria-checked`；`client/ui-primitives/src/Switch.module.css:5-21`、`:97-99` | `ui.Checkbox(c, checked, "")`（空标签、独占一行）；`internal/ui/settings/settings.go:406-414` | 见致命 #2 | P0 | 改 `ui.Switch` + `Label` |
| 4 | 设置 · 保存反馈 | 单按钮：未 dirty 禁用、保存中换文案、失败在按钮左侧出状态行；`client/ui-primitives/src/settings-form/SettingsForm.tsx:56-71`；git/插件/定时均有 toast（`client/ui-git/src/client/GitProgressToast.tsx`、`client/ui-plugin-manager/src/client/PluginRefreshToast.tsx`、`client/ui-schedule/src/client/DeleteToast.tsx`） | 三个并排按钮（保存 / 完整替换 / 放弃草稿）+ 两段 11px 说明，结果是一行 11px 灰字置顶；`internal/ui/settings/settings.go:336-359`、`:219-221`；`c.AddToast` 全仓仅 6 处（`internal/ui/settings/preference_notices.go:28` 等），保存/安装/提交均无 toast | 反馈不在操作处、按钮分不清主次；成功反馈没有统一出口 | P1 | 主按钮唯一 + 就地状态 + 统一 Toast 出口 |
| 5 | 设置 · 危险确认 | 应用内 Modal + RiskConfirmation；`client/ui-git/src/client/CommitDialog.tsx:74-95` | 系统 MessageBox（64 处）；`cmd/whaleisle/main.go:5007-5018`、`cmd/whaleisle/task_protection.go:72` | 见致命 #3 | P1 | 补应用内 ConfirmDialog |
| 6 | 设置 · 导航 | 13 项（+账号 14）、逐项不同图标、无搜索框；`SettingsRoot.tsx:39-54`、`:105-124` | 15 项、4 项共用齿轮、多出"远程工作区"；`internal/ui/settings/navigation.go:81-95` | 见致命 #6 | P1 | 删多余分区 + 修图标映射 |
| 7 | 设置 · 通用页主体 | 通用页 = 一组偏好行（语言、Enter 发送、关闭行为、开机启动、自动恢复、开发者工具）；`client/ui-settings-general/src/client/index.ts:313-350` | 通用页 = 3 个横排文字按钮 + 命名空间搜索框 + 命名空间下拉 + 刷新按钮，再接所选命名空间的 schema 表单；`internal/ui/settings/navigation.go:101-165`、`settings.go:226-228` | 偏好行被"命名空间浏览器"取代；用户要靠下拉框在实现命名空间里找设置 | P0 | 通用页改回固定偏好行；命名空间浏览降为默认折叠的"高级"区 |
| 8 | 设置 · 入口区 | 侧栏底部 `settings.launcher`（账号头像 + 名字 + 菜单）+ 设置行 + 更新/连接指示器；`SettingsRoot.tsx:296-329`、`client/ui-settings/src/client/contract/slots.ts:17` | 侧栏底部只有"启动器"文字按钮 + 设置行 + 一个状态圆点，无账号块；截图 `evidence/native-go-3090.png` 左下角"启动器"行 对 `evidence/original-electron-3080.png` 左下角（头像 + "Ayase" 行） | 身份块缺失，被替换成打开另一个窗口的入口 | P1 | 恢复 `settings.launcher` 槽位语义 |
| 9 | 文件 · 多标签 | 横向 28px chip 条（min80/max170/13px），active 底色，hover 出 close，可拖拽 / 溢出菜单 / `+`；`client/ui-dockkit/src/components/dockkit.module.css:237-248,357-376,443-448,482-485` | 竖排 11px 文字按钮，`"▸ "` 表活动、`" •"` 表脏、独立 `"×"`；`internal/ui/files/tabs.go:316-339` | 见致命 #4 | P0 | 重写 `paintTabs` |
| 10 | 文件 · 视图切换 | 工具栏幽灵按钮并列：源码/渲染、折行、浮动预览、评论选行、Save（primary）；`client/ui-files/src/client/FilePreview.tsx:567-644`（`:583-597` markdown 切换、`:622-633` 评论） | 三态**循环**（分屏 → 编辑 → 高亮）+ 工具条 11px 文字按钮；`internal/ui/files/files.go:451-472`、`:608-628` | 原版是并列按钮，原生是循环切换，用户看不出有哪几种视图 | P1 | 改分段控件（3 项） |
| 11 | Diff · 行样式 | hunk 圆角头 `bg-layer-2`、gutter 12px、行 20px、`+/-` 8% 状态色底、代码字体；`client/ui-diff/src/client/DiffPanel.module.css:76-120` | 单块 RichText，仅前景色；`internal/ui/widgets/diff/diff.go:78-92`、`internal/ui/diffsurface/patch_view.go:133-151` | 见致命 #5 | P0 | 行原语 + 底色 + hunk 头 |
| 12 | Diff · 工具条 | scope 是 2px/8px 的 `radius-sm` chip（12px 字），刷新是 24px 图标按钮；`client/ui-diff/src/client/DiffPanel.module.css:18-50` | 4 个 11px 文字按钮（"切换为分栏/启用折行/全部折叠/全部展开"）+ 每文件 10px "展开/折叠"；`internal/ui/diffsurface/patch_view.go:23-44`、`:57-63` | 无图标、无 chip 形态，工具条读起来像调试开关 | P1 | 图标按钮 + 分段控件 |
| 13 | 终端 · 标签与标题 | 28px chip + OSC 标题（`TerminalTitle.tsx`），工具条为 28px 圆角药丸含 0.5px 分隔；`client/ui-user-terminal/src/client/TerminalWorkspace.module.css:25-66` | 单 pane 无标题；多 pane 右侧 144px 竖列（"组 N"/"终端 N"/"×"）；`internal/ui/terminal/panel.go:298-315`、`workspace_chrome.go:53-89` | 见致命 #10 | P1 | 横向 chip 条 |
| 14 | 终端 · 字体/配色 | 主题 token 驱动 xterm（`--dsw-font-family-terminal`、13/20），并吃 OSC 4/10/11/12 程序改色；`TerminalWorkspace.module.css:11-13`、`client/ui-sidebar-terminal/src/client/terminal-theme.ts:19-64` | 原生 Ghostty `myterm.LightTheme/DarkTheme` 覆盖 Background/Foreground/Cursor/Selection，字号字体取 `theme.CurrentPreferences().Terminal`；`internal/ui/terminal/panel.go:257-274` | ANSI 16 色仍用框架默认，不随主题 accent；程序 OSC 改色支持未知 | P2 | 由主题派生 ANSI 调色板 |
| 15 | Git · 提交 | 520px 对话框：分支行 / 告警 / 文件清单(+N/-N) / 合计 / message / footer；`client/ui-git/src/client/CommitDialog.tsx:74-199`、`CommitDialog.module.css:1-13` | 内联 TextInput + 三个"核对…"文字按钮，文件动作是 10px 文字按钮；`internal/ui/diffsurface/commit_selection.go:311-353`、`workbench.go:166-193` | 见致命 #9 | P1 | 弹层 + 结构化清单 |
| 16 | 用量 · 主体 | KPI 卡（900ms 计数动画）+ 贡献日历 + 环图 + 排行卡 + 导出菜单 + 计费弹窗；`vendor/dsh-usage-panel/src/client/components/KpiCards.tsx:59-64`、`Heatmap.tsx:34-56` | 统计数字用 `ui.Text` 逐行打印，含 `Coverage: … UTC`、`Events … Retries …`；`internal/ui/usage/stats.go:48-52,100-105` | 见致命 #7 | P0 | 仪表盘化 |
| 17 | 用量 · 图表形态 | 周历（7 行 × N 周、月标签、Mon/Wed/Fri 行标签、四分位分级、点选区间、tooltip 含费用）；`vendor/dsh-usage-panel/src/client/components/Heatmap.tsx:1-7,34,43-80` | 单行 18×18 方格（9px 标签）+ 22px 方块柱（96px 高、10px 标签）；`internal/ui/usage/heatmap.go:16-51`、`chart.go:19-49` | 无坐标轴 / tooltip / 图例；热力图不是日历 | P1 | 重画两类图 |
| 18 | 用量 · 文案 | 全量走 `locales.ts`（408 行字典）；`vendor/dsh-usage-panel/src/client/locales.ts` | 硬编码英文 `"7 days"/"14 days"/"30 days"`、`"Export json"`、`"Coverage:"`、`"Input … Output …"`；`internal/ui/usage/stats.go:48-52,70-79,87-97` | 中文界面混入英文标签与导出格式名 | P1 | 走 i18n key |
| 19 | 插件市场 · 卡片 | `grid-template-columns: repeat(auto-fill, minmax(280px,1fr))`、gap 12、卡 `radius: md(12)`、`padding:12px 14px`、`border 0.5 l2`、hover 变 `border-l3`（有过渡）、描述 2 行 clamp、foot 行放分类标签/已安装标记/动作；`client/ui-settings-market/src/client/MarketSection.module.css:217-421` | 手算列数（≥572 才 2 列）、`Radius(8)`、`Border(.5)`、`Padding(12,14)`、标题 14 Bold，无 hover 边框过渡；`internal/ui/market/presentation.go:105-139` | 半径 / 列数规则 / hover 反馈与 680px 详情弹层均不同，卡片密度低于原版 | P1 | 自动列 + radius 12 + hover 边框 |
| 20 | 插件列表 · 行卡片 | 48×48 图标瓦片（`border 0.5 l3`、`radius-lg`）、标题 14/500、描述 13 单行 clamp、状态胶囊 18px/10px、启用 Switch、分组标题 + 计数 + info tooltip、安装后 2400ms 高亮环；`client/ui-plugin-manager/src/client/PluginManagerPage.module.css:128-244`、`:174-195` | 文字段落（`已安装插件包`、`运行时未连接…`）+ 一个裸列表；`internal/ui/settings/plugins.go:49-90` | 行卡片体系完全缺失，插件页像日志 | P1 | 复用市场卡片原语做 48px 图标行 |
| 21 | 定时任务 · 时间选择 | 三列滚动时钟（列宽 52、行 32、`max-height:216`、选中 `inset 0 0 0 1px` 环）+ 月历 DatePicker（独立 module）；`client/ui-schedule/src/client/ClockPicker.module.css:2-48`、同目录 `DatePicker.module.css` | 日期是 `"日期 / Date: 2026-10-20"` 文字按钮 + `‹ 2026-10 ›` + 7 列 28px 日格 + 11px 周标签 + `"关闭 / Close"`；时钟是三列 Select；文案双写 `"每隔 / Every"`、`"分钟 / Minute"`、`"当前时间规则只读 / Timing is read-only"`；`internal/ui/schedule/timing_picker.go:150-178,213-265,292,434,446` | 选择器是表单控件拼装而非自绘滚动选择器；日历无今天/禁用态；双语拼接撑开标签宽度 | P1 | 自绘月历 + 滚动时钟 + i18n |
| 22 | 远程 · 机器/文件页与配对 | 卡片 `border 1px l2 / radius 10 / bg-layer-1 / padding 12 / gap 10`，卡标题 13/600；机器行是**带边框的卡**（`padding 8 10`、`radius 8`，当前机 `border: brand` + `interactive-bg-active` 填充，名 13/600 + 副行 12）；表单是 `108px + 1fr` 网格、标签右对齐 12px；文件树行 `padding 3 4 / radius 5 / 12.5px`、折叠符 14px、徽标 11px；审计区是 `max-height:220` 的边框盒、等宽 11.5/1.6；配对卡用本地 `uqr` 编码出二维码 SVG；`dsh-remote/styles.js:7-76`（上述数值）、`dsh-remote/RemoteExplorerBody.jsx`、`dsh-remote/RemoteSettingsSection.jsx`；配对：`client/ui-settings-remote/src/client/qr.ts:1-15`、`RemoteSection.tsx:302-340` | 卡片 `Radius(8)/Border(1)/Padding(10,12)`，机器列表只是 `Column.Gap(8).Padding(8,0)`（无逐行边框、无当前态品牌边）；文件树行 `Height(26)`、缩进 `depth*14`；审计区是 `Scroll.Height(180)` + 11px 纯文本（无边框、非等宽）；配对有 QR 分支但无图时提示"二维码不可用，可复制临时配对链接"，状态行直接打印 `"offer v2 · ready"`；`internal/ui/remote/remote.go:1-368`（卡片/标题/滚动）、`file_workspace.go`（树/弹层 480px `Radius(12)`）、`tools.go:180`（`Height(180)` 审计）、`machines.go`（无卡）、`phone.go:138,142-152` | 机器卡与"当前机"标识缺失；树/审计的容器盒缺失；协议调试串进 UI；QR 依赖运行时位图而非本地编码 | P1 | 机器卡补边框 + 当前态品牌色；审计与树套边框盒；删 `offer v2`；本地生成 QR |
| 23 | 桌宠 · 窗口模型 | 全屏透明覆盖层 + 点击穿透开关（`setInteractive`），宠物绘制 240×260，粒子/饭碗可超出宠物矩形（`reportRoam` 上报矩形）；`pet-live2d/pet-live2d.js:59-60,116-117,1731-1740` | 独立 220 宽、最小高 438 的无边框窗口，`Transparent/AlwaysOnTop/SkipTaskbar`，webview 载 `whale-pet://localhost/index.html`；`internal/desktop/pet/companion.go:187` | "全屏穿透覆盖"退化为"会挡桌面的小窗口"；粒子/饭碗无法越界 | P1 | 改全屏透明层 + 命中区上报（`hit_regions.go` 已有基础） |
| 24 | 桌宠 · 渲染管线 | ONNX 神经化身（`avatar/model.onnx`）+ anime4k 超分 + 物理 + 漫游，资源 5.16MB；`pet-live2d/pet-live2d.js:51-52,305-417`、`pet-live2d/pet-live2d.html:276-302` | 资源 1.38MB（`pet-rig.js` 16KB、`pet-runtime.js` 20KB、`shell.png` 522KB、`tail.png` 505KB）；THA4 神经资源要求 exe 旁存在 `pet-render-assets`，缺失即报 "THA4 assets unavailable" 退回 rig 贴图；`internal/desktop/pet/neural.go:20-59`、`companion.go:353-402` | 当前源码树 exe 旁**没有** `pet-render-assets`（已核 `whaleisle.exe` 所在目录），默认跑贴图模式 | P1 | 打包随 exe 分发资源 + 补形象/模型选择 |
| 25 | Project · 观察器形态 | **非模态浮动观察器**：可折叠/展开并保留用户调整过的尺寸与位置，按 Session/tab 持久化；主界面仍是 Lead 对话，可边看观察器边继续说话；材料侧栏用 **Results / Notes / Project instructions** 三个页签，Results 先给结论/验证/遗留问题/文件，详情折叠；`resources\vendor\dsh-project\README.md:57`、`:59`、`:61`、`:63` | **模态** `DialogBase`：遮罩黑 40%、面板 `min(760, w-32)`、最大高 80%、`Radius(28)`，关闭才能回到对话；材料是四个文字页签 `工作成果/项目规则/偏好/笔记` + `ProjectID / Path` 面包屑 + `N / 256000 UTF-8 bytes` 计数器 + 240px 文本框；`internal/ui/projects/modal.go:170-176`、`materials.go:29-76` | 观察器从"浮动、可缩、不挡对话"退化为"模态弹窗 + 路由栈"；材料页签结构与 Results 优先层级缺失；字节计数器是内部实现细节外露 | P1 | 改浮动面板 + 尺寸/位置持久化；材料页签收敛为 3 个并按 Results 优先级排序 |

---

## 三、待验证

1. **Project 的可读基准只有 README**：安装版 `C:\软件\Whale Isle\resources\vendor\dsh-project` 只有 `client/` + `lib/`（构建产物）+ `README.md`（81 行），无 TSX/CSS；本仓库 `runtime\vendor\` 下**没有** `dsh-project`（只有 `dsh-platform-session`、`dsh-task-control`、`dsh-im`、`dsh-remote`、`dsh-usage-panel`、`dsh-whale`、`dshbot`）。本文件第 25 条以该 README 的界面段落（`:57-65`）为基准；若需要逐像素比对，须补 vendored 源码或提供运行截图。
2. **设置面板阴影/背板 token**：原版 `--dsw-elevation-prominent`、`--dsw-alias-bg-mask-1`、`--dsw-alias-bg-layer-2` 的终值取决于主题 CSS 变量，本轮只核到 `--dsw-radius-panel:28px`（`client/ui-theme/src/styles/base.css:26-29`）；`--dsw-elevation-*` 未定位到定义处，原生阴影参数未对比。
3. **文件预览工具栏结构**：原生有 `internal/ui/files/preview_toolbar.go`（434 行）与 `toolbar_path.go`（60 行），是否已有面包屑（原版 `FilePreview.tsx:570-582` 的 chevron 路径）本轮未逐一核对渲染分支，需实测截图确认。
4. **Markdown 预览保真度**：原版用 `MarkdownText`（`client/ui-primitives/src/MarkdownText.tsx`，含 katex / 表格 / 脚注）；原生有 markdown 渲染器与 `table_preview.go`、`svgimage`，但表格/公式/脚注的实际差异需逐样例对比。
5. **文件树交互**：原版 `FileTree` 支持搜索走全树 DFS + 缓存（`FilesPanel.tsx:99-108`）、拖动生成 @ 提及（`client/ui-files/src/client/fileTreeDragMention.ts`）、右键"在应用中打开"（`client/ui-open-in-app`）；原生 `internal/ui/files/tree_actions.go`（150 行）+ `workspacefs.Search` 是否等价需实测。
6. **IM 渠道页与桌宠助理块**：`runtime\vendor\dsh-im\src` 只有 `.mjs` 渠道实现（钉钉/飞书等，非 UI），无 TSX；`dsh-whale\client` 只有 35KB 构建产物 `client.js`；两者界面无法取证。可读的只有 `dsh-remote\plugin-src\client\*.jsx|js`（已用于第 22 条）。IM/桌宠助理块若要继续，需以安装版运行界面截图为准（对照 `internal/ui/im` 933 行、`internal/ui/settings/pet.go` 186 行）。
7. **浏览器预览 / 计划预览 / 自定义面板**：`internal/ui/browserpreview`（3641 行）对应原版 `ui-preview`（22 文件，含 `DeviceToolbar`、`browserRecording`、annotation），本轮未展开；建议单开一条，重点核设备工具栏、录制条、批注层。
8. **桌宠设置项完整性**：原版 `PetSection` 通过 `settings.pet.item` 接收 `dsh-whale` 的助理字段（`client/ui-settings-general/src/client/index.ts:361-372`）；原生 `pet.go` 有"聊天与能力 / 桌面形象与行为"两页签（`:42-48`），但助理字段由 `dsh-whale` 的 `client.js`（35KB 构建产物）注入，实际字段覆盖面需在运行版对比。

---

### 附 A：本轮核对过、判定已对齐（不必排期）

- 设置面板尺寸/圆角（800×800、28）、左栏 188/40/12、行内 `9px 16px 9px 12px`、标题 16/500、内容区 `padding:0 24 24`：`internal/ui/settings/settings.go:184-215` 对 `client/ui-settings-general/src/client/SettingsRoot.module.css:117-272`。
- 设置行原语 `bodyPreferenceRow` 的几何（14/12 字号、16/0 行距、0.5px 分隔线、180px 右列、420px 断点）：`internal/ui/settings/body_preferences.go:33-73` 对 `CloseBehaviorRow.module.css:3-32`。问题只在"没有全域使用"（致命 #1）。
- 终端工具条药丸（28px、`Border(0.5)`、5 个动作）：`internal/ui/terminal/workspace_chrome.go:16-51` 对 `TerminalWorkspace.module.css:25-66`，仅圆角 6 vs 12 的 P2 偏差。
- 文件"所选行添加到对话"与"引用整文件"能力：`internal/ui/files/source_actions.go:142-146`、`references.go:18-25` 已具备原版 `FilePreview.tsx:622-633` 的语义。
- 桌宠设置页的两页签骨架（360px 宽页签、16/500 小标题、14/12 行）：`internal/ui/settings/pet.go:41-107` 对 `client/ui-settings-general/src/client/PetSection.module.css:4-78`。

### 附 B：本轮的量化扫描（可复现）

| 扫描 | 命令口径 | 结果 |
| --- | --- | --- |
| 双语硬编码 | `internal/ui` 下同时含非 ASCII、` / `、引号的行 | 955 处 |
| 系统对话框 | `mygo.Dialog.(Message\|Confirm\|Open\|Save\|Error)` | 64 处（应用内 + cmd） |
| Toast | `c.AddToast` / `ToastBase(` | 6 处 / 0 处 |
| Tooltip / Popover / 右键菜单 | `.Tooltip(` / `Popover(` / `ContextMenu(` | 156 / 3 / 12 |

结论：Tooltip 覆盖面足够；**Popover 与 toast 严重不足**，这是"原版有悬浮卡/短暂反馈，原生没有"的系统性来源，排期时应与致命 #3、#4 同批处理。
