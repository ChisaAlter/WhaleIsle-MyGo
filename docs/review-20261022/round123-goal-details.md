# Round123 · 目标详情卡片与 Inspect 入口

本轮实现已安装鲸屿的目标（goal）结果详情卡与统一的 Inspect 入口，构建为 [pass267](<C:/AI/WhaleIsle-MyGo/whaleisle-pass267.exe>) 并更新 [默认程序](<C:/AI/WhaleIsle-MyGo/whaleisle.exe>)。整体「布局、UI、功能、动效完全一致」目标仍在进行；上一轮提交为 `5358dd9f`，参考 Electron、后端服务及 GUI 3080 未改动。

原版的 `create_goal` / `get_goal` / `update_goal` 不走通用 IN/OUT，而是把**记录下来的**目标渲染成一张紧凑卡片：目标正文一行、`状态`、`执行轮次`、可选 `受阻原因`，字段为 `5.5em + 1fr` 两列网格；卡片 `margin 4/0/4/4`、`.5px` 描边、`radius 16`、`overflow:hidden`，列表最大高 320 并可独立滚动。折叠行摘要直接使用目标正文（空目标用「没有目标」），不再重复参数。实现见 [目标详情模型](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>)、[卡片渲染](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details.go>)、[菜单文案](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>) 与 [接入](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_lifecycle.go>)。

模型严格跟随原版 `details-card-model` 的 `goalDetail`：[回归](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 含 35 个由当前安装版模型 + 真实 `toolDefinition` 与中英文词典在 Node VM 生成的 golden。`goal:null` 直接得到空状态且**不校验** activation；其余要求 `id`/`objective` 非空白、`revision`/`roundsStarted`/`maxGoalRounds` 为非负安全整数、`phase` 与 `activation` 枚举合法；`blockedReason` 若出现必须是含非空白 `code`/`message` 的对象。正文与受阻原因保留原始（不 trim）拼写；轮次按 JS 语义取整，允许 `0` 与「已执行轮次大于上限」。根调用只读保留的原始 call head；嵌套收据自带 name/arguments，因此即使已进入结果窗口、或先前的 start 名称不同，仍能正确出卡。模型不改动历史记录，且每次派生返回独立切片。

| 细节 | 实现 |
|---|---|
| 目标图标 | 仅 goal 三工具使用安装版 `IconGoalOutlineRegular` 路径，其余工具字形不变 |
| 卡片容器 | 4/0/4/4 外边距、`.5px` 描边、radius16、`overflow:hidden`、`code-block` 底色、13/20 字体 |
| 条目 | `padding 10px 14px`；多条之间 `.5px` 分隔线；标题 `pre-wrap` + 任意处折行 |
| 字段 | 标题后 `margin-top 6`，`5.5em` + `1fr`、列间距 12，行距 4；label 用 caption 色，值 `pre-wrap`/任意处折行 |
| 空状态 | 单行「没有目标」，不渲染字段 |
| 滚动 | 列表 `max-height 320`，独立滚动，不撑开聊天流 |
| Inspect 预留 | 有 Inspect 时条目/空状态右侧预留 88；无 Inspect 时保留 14 |

Inspect 现在由**所有**有结果的工具行共用，而不是目标专用：原版在展开体末尾渲染，目标卡片则绝对定位在右上 `12/12`。按钮 `.5px` 描边、圆角 999、11/16 字号、`2px 8px` 内边距、图标 12；默认 `opacity 0`，整行 hover 或按钮键盘聚焦时以 100ms `cubic-bezier(.4,0,.2,1)` 淡入，hover 底色为安装版 `--dsw-alias-interactive-bg-hover-solid`（浅 `#f1f3f5` / 深 `#353638`）。隐藏透明度不取消键盘可达性，Space/Enter 与点击都会触发，并保持展开状态。文案改为原版 `Inspect` / `查看`。

原生验证使用只读记录 JSON 的 fixture，回调仅记录 payload，不连接 Host、不修改目标、不写剪贴板：[最终记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-native-final-state.json>)（PID 24140，已关闭并验证退出）。覆盖：折叠行 24 高 → 展开后出现状态/轮次/受阻原因；Inspect 通过点击、Space、Enter、UIA Invoke 各触发一次，四次都携带正确 `native-goal` / `goal-message` / `207` 且不折叠外层；空目标卡片可展开；320 宽与深色主题下保持同一调用身份；通用工具仍在 IN/OUT 之后以普通流布局显示 Inspect 并回传 `native-generic`。更早的两次失败记录（PID 42652、16416）分别是探针用整数物理像素直接比对逻辑坐标、以及 PowerShell 以 GBK 解码 `·` 字号差异，均属探针问题，产品代码未改。

[布局预设回归](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 验证 420 宽卡片：两字段卡片外层高 100（含 8 外边距），空状态 50，超长内容被 320 上限裁为 330；label 起点 x19、值 x102.5、标题 y15、首个字段 y41；长列表滚动后字段确实位移且不影响后续行。截图见 [浅色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-native-software-goal-dark-false.png>)、[深色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-native-software-goal-dark-true.png>)。

**参考基准在中途发生了变化。** 会话进行中安装版鲸屿更新了 Web 产物（`index-l5jToo6l.css` / `index-wKUmVWbt.js` → `index-dU47c3Cy.css` / `index-mpwUGNSV.js`），`ui-tool` 与 `locale` 模块哈希也随之变化。我重新核对了当前安装版：[权威对照](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-installed-authority.json>) 显示本轮卡片/Inspect 依赖的 **12/12 条规则仍逐字存在**（含 88 预留、5.5em 网格、320 上限、`.5px` 描边、hover 淡入与实色 hover token），`details-card-model.js` 哈希未变（`61babee6…`），中英文文案逐项一致（`Status/Rounds/Blocker/Awaiting continuation` ↔ `状态/执行轮次/受阻原因/等待继续`，`Inspect/查看`）。旧资产名已不存在，先前准备的浏览器参考脚本因此**在 Chrome 启动前**就停止，本轮**没有**进行浏览器验收，也未创建 Chrome 进程；相关的过时参考脚本与旧源码快照已删除，避免被误当作现行权威。

因此本轮证据边界是：安装版**源码级**规则与文案一致 + Node VM 模型 golden + 原生布局/交互记录。没有浏览器计算样式、真实 Host Inspect 面板跳转或目标实时状态比对。原生 `.5px` 描边按整设备像素实现（DPR1 每边 1），因此卡片外框可能比 CSS 分数边框高 1 左右，属未消除的已知差异。

回归：Chat 60.735s、Frame 13.292s、i18n 0.571s、Session 7.591s 全通过，`mygo vet` 生命周期检查与受影响包 `go vet` 通过，见 [最终检查](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-final-checks.json>)。未修改 MyGo 框架，无全框架通过声明。默认与版本化产物均为 86391808 字节、SHA256 `0DEC05F1796761714F2CBEBDE5EB1429A258ADA65A74B6FD97F8A835383F6C4E`，见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass267-build.json>)；无二进制提交。

整体仍待后续：其余细节族（定时任务、作业、终端、团队、会话、Cordis、workflow/ralph 等）的详情模型与 `ToolDetails` 通用条目（badge/description/lines/groups/unchanged/code），Inspect 面板自身布局，以及既有的字体与基线、选择与复制、滚动条、tooltip、presence 动效、Host 主页/物理输入、外壳与材质、完整启动关闭、真机媒体等。卡片级 `.5px` 边框取整、隐藏按钮的可访问性朗读顺序、以及新安装版整体回归也仍需继续核对，不能据本轮判定全产品对齐完成。
