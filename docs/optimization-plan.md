# 鲸屿 MyGo 优化实施方案

更新：2026-11-13。状态：方案已制定，以下批次尚未实施。

技术基线：[核心架构与技术路线](../ARCHITECTURE.md)。本方案规定执行顺序与交付批次，不另立一套架构；指标数值、插件兼容合同与数据权威以架构文档为准。

## 1. 先让核心工作顺，再兑现原生优势

本轮优化要交付一个可以日常编码、支持 DSH 自定义扩展、资源收益可解释的客户端。优先级是：

1. **操作可靠：** 输入、发送、停止、切换和审批不丢状态、不误作用于其他会话。
2. **交互连续：** 流式输出、滚动、文件与 Diff 不互相拖慢，反馈、焦点和恢复符合用户预期。
3. **生态完整：** 标准 DSH 插件无需 Go 专用版本；自定义左侧空间、会话 View 和右栏有真实可用路径。
4. **收益可测：** 整个必要进程树的资源与响应优于对照，而不是只优化主 EXE 的数字。

推荐按三个交付轮次推进：**核心可日用 → 扩展可持续 → 原生收益与发布**。每批完成产品修改后再补定向测试和实际 QA；不先造一套优化平台，也不先做全站视觉重绘。

## 2. 当前依据：哪些已存在，哪些仍需定位

以下来自当前源码与已有诊断，不能直接视为运行瓶颈或故障根因。

| 观察 | 判断与行动 |
| --- | --- |
| `mainView` 中集中许多帧观察、生命周期及跨模块协调 | 按完整工作流收敛归属，优先检查这些调用是否在无状态变化时重复工作；不靠拆文件数衡量改善 |
| 侧栏 `whaleAssistantNavLabel` 在缓存到期时会同步 `Stat` / `ReadFile` | 是可确认的 UI 路径 I/O 清理点；移到已有设置观察流程，绘制只取快照 |
| Session 已有 `viewCache`、`FrameSnapshot`，聊天已有 transcript memo 和增量前缀复用 | 优先复用、检查失效范围与实际命中；不再平行引入第二套缓存 |
| 普通 `Snapshot` 与帧快照有不同拷贝和生命周期合同 | 优化渲染调用时维持原合同；不能把只读共享帧数据交给异步任务或改变普通调用方的数据隔离 |
| 已有帧采样、pprof、快照统计、启动跟踪和 watchdog | 使用现有观测定位成本；只有现有数据无法回答具体问题时才补必要观测 |
| `pluginui` 同时承接贡献目录、交互与 DOM 镜像，已有精确 Host 网关与 WebView | 保留可用协议与安全传输，迁移通用镜像，不整体推倒重做 |
| 工作区创建、聊天提示与设置动作存在未提交修复 | 先辨认这些改动的目标与有效证据，再纳入首批；未验证修改不当作已关闭问题 |
| 11 月 11 日点击卡死诊断尚未复现，观察的入口曾指向旧构建 | 保留未解决状态，定位到受影响实例后再修；不以换绘制引擎或新增超时重试猜测根因 |

代码依据见文末。新版本出现反证时，只更新受影响批次的判断，不重跑无关项目。

## 3. 批次与依赖

| 批次 | 交付结果 | 依赖 | 优先级 |
| --- | --- | --- | --- |
| 0 | 明确本次实现输入与真实运行对象 | 无 | 开工准备 |
| 1 | 会话、草稿与交互形成稳定工作流 | 0 | 核心 P0 |
| 2 | 输入和长对话响应稳定，UI 路径无已确认阻塞工作 | 1 的状态归属 | 核心 P0 |
| 3 | 文件、Diff 与主工作台操作连贯 | 1；复用 2 的呈现约束 | 核心 P0/P1 |
| 4 | 真实 DSH 左侧扩展空间完整可用 | 1 的操作归属；核心批次形成可用候选 | 生态 P0 |
| 5 | 更多扩展面可用，逐项退役通用镜像 | 4 | 生态 P1 |
| 6 | 原生收益有同条件测量，候选具备升级与恢复路径 | 1–5 对应交付稳定 | 产品 P1 |

批次 1–3 对应架构阶段 A，4 对应 B，5 对应 C，6 对应 D。共享工作流、扩展协调或 Host 生命周期在同一时期由一个负责人维护；独立呈现工作可以在 Interface 稳定后分别推进，最终在同一候选验收。

## 4. 第一轮：核心可日用

### 批次 0：明确输入，不重建验收工程

开工时读取当前改动、实际启动入口、EXE 版本、选定 Host/Node 和启用的插件。复用已有日志、候选与相关有效证据；在隔离配置下做必要观察，区分源码、旧安装和受影响实例。

**批次 0 交付物是本方案文末的盘点记录**：工作区未提交改动的分组与归属批次、每项改动的现有证据层、编译/测试基线、未解决问题清单。盘点完成后才能进入批次 1 的代码合并，避免把来历不明的改动混入新实现。

已有候选不能代表后续改动后的行为。实现开始后按实际需要构建新候选，不修改正在使用的 profile，不把全量复制运行时和全量重测作为每次改动的前置步骤。

**完成条件：** 受影响对象、已有未解决反馈、输入版本与本批保留项清楚；仍未知的卡死根因明确保留。对当前可确认的改进直接进入实现，不等待所有历史故障都复现。

### 批次 1：收敛会话、草稿、提交与审批

**用户结果：** 连续输入、发送、停止和切换会话后，内容与操作始终落在正确对象上。

实施：

- 在现有 Session、Composer、Interaction 中收敛工作流归属；主程序只连接导航与界面，不另建一份会话状态。
- 完成新建/复用空会话的目录与身份处理，整合已有冲突修复；区分 Host 已受理、明确失败和结果未知，不靠重复创建掩盖问题。
- 统一提交快照、草稿编辑版本、附件归属和清稿条件；切换、断线、取消与失败保留用户后续输入。
- 收敛发送/停止/审批/提问入口，确保当前动作与请求、会话、连接身份一致；保留后台任务和其他会话的正常执行。
- 本地提示归属到其具体页面/动作，修复过期提示与设置页串扰，不把全局诊断串作为页面状态。
- 输入法组合期间按现有文本输入合同处理 Enter、Esc、撤销和快捷键；只修实际行为差距，不重写输入引擎。

落点：`internal/store/session`、`internal/store/interaction`、`internal/ui/composer`、`cmd/whaleisle` 的会话与恢复接线。

**分批落地节奏（单批过大风险的对策）：**

- **1a 会话创建与提示归属**（已有未提交修复覆盖大半）：工作区/空会话目录与身份、session/conflict 重试、提示串扰过滤。交付以现有测试 + 同一候选真实路径为准。
- **1b 草稿、提交与审批归属**：提交快照/草稿版本/附件归属/清稿条件；发送/停止/审批/提问入口身份一致。这是本批剩余的主要实现面。
- **1c 输入法合同**：IME 组合期间 Enter/Esc/撤销/快捷键，只修实测差距。

1a 可先独立构建候选验证，不等 1b/1c 全部完成；1b 是完成条件的主体。

**完成条件：** 两个会话之间带草稿/附件切换、提交中继续编辑、失败后重提、停止后续聊、断线重连和一次性审批在同一候选真实可用；没有丢稿、错会话、重复提交或重复应答。新建目录冲突的既有修复有当前行为证据。

### 批次 2：清理帧热路径，改善输入与长对话

**用户结果：** 模型持续输出时仍能输入、阅读旧消息、选择复制与切换；没有周期性卡顿或自动滚动抢位置。

实施：

- 把已确认的侧栏文件读取迁到已有设置观察流程，以快照发布名称；无网络/磁盘工作进入绘制回调。
- 复用帧采样、pprof 和快照统计定位当前候选的主要耗时，再处理超额计算、锁持有与重复通知。
- 复用 `FrameSnapshot`、Store 修订缓存和 transcript memo；缩小失效范围，避免侧栏或用量变化重建不相关消息。
- 仅对实际变化的消息做解析与重排，保留虚拟列表和稳定消息身份；渲染通知可合并，Host 事件、持久消息、审批与错误不能丢弃。
- 维持普通快照深拷贝和帧快照只读合同；异步操作携带独立输入，不持有帧内共享视图。
- 翻页、折叠、流输出分别保留阅读锚点；用户离开底部后不强制追流。光标、选区和焦点不因布局更新而重置。

落点：`mainView`、`internal/ui/sidebar`、Session 的帧快照与投影、`internal/ui/chat`、`internal/ui/widgets/markdown`。

**完成条件：** 固定长历史与持续流场景满足架构第 11 节的交互目标；旧消息阅读、翻页、折叠与选区保持正确。若仍有长帧，记录实际耗时归属，继续处理影响使用的具体成本，不以缓存命中数代替体验通过。

### 批次 3：把文件与 Diff 做成同一个工作环境

**用户结果：** 从聊天打开文件、查看改动、保存和回到对话的过程连贯，未保存内容不会被刷新或切换覆盖。

实施：

- 统一“会话 → 文件/行 → Diff → 返回”的目标身份和导航，复用现有文件标签、草稿、外部变化及保存冲突保护。
- Git 查询与文件扫描按工作区及请求归属更新；较早请求不能覆盖新工作区结果。窗口内选择与后台操作各有明确归属。
- 精简 Diff 默认工具区：主路径突出文件、变更和暂存/还原，仓库级、分支级操作进入明确次级入口；保留既有功能和风险确认。
- 对聊天、输入区、侧栏、Diff 做一轮聚焦的密度、层级和状态反馈统一；不同时更换品牌、主题系统或全部页面。
- 终端保持既有原生路径，修受影响的焦点与快捷键交接，不另换终端实现。

落点：`internal/ui/files`、`internal/ui/diffsurface`、`internal/desktop/workspacefs`、`internal/desktop/git`、主程序导航与窗口接线。

**完成条件：** 打开引用文件并定位、编辑后切换/返回、外部修改冲突、Diff 暂存/还原、终端焦点和返回聊天真实通过；没有隐式保存、提交或推送。实际工作台呈现经过检查，不以静态组件存在代替整体使用验证。

## 5. 第二轮：扩展可持续

### 批次 4：交付 DSH 左侧扩展空间的完整链路

**用户结果：** 可以创建或安装标准 DSH 插件，在左侧打开自定义空间，切回聊天时原有工作保持不变。

实施：

- 在现有 `pluginui` / `pluginslots` 上收敛扩展协调 Interface，向原生导航发布实际有效贡献和动作身份；不新造插件管理器或“空间列表 RPC”。
- 在隔离发行输入中接入架构固定的新 DSH 契约与匹配 Node，保留旧运行时 Adapter；不静默切换日用版本。
- 新版使用 `sidebar.panellist` + `main`，旧版侧栏区域使用其原 owner；导航和排序来自实际注册结果，不按包名猜测。
- 自定义页面保留原 DSH Client Runtime / Cordis / React 组合，通过现有鉴权网关与 WebView 承载；不要求插件作者提供 Go UI。
- 接通“更多 → 创建你的插件”的 Creator 草稿、附件保留与用户确认；仅准备草稿，不自动发送或安装。
- 实现切换空间、重载、禁用/卸载、Host 重启时的路由恢复与动作撤销；保留旧会话、草稿与文件编辑状态。

落点：`internal/desktop/pluginui`、`internal/desktop/pluginslots`、`internal/desktop/webview`、`internal/dsh`、侧栏与主面板路由。

**完成条件：** 一个实际 DSH 前端插件完成安装/启用、注册、原始页面交互、切换、HMR、禁用/卸载及重启恢复；核心草稿不受损，旧动作无效。新版与所保留旧版各按其真实契约验证。该闭环只证明这些路径，不扩称全部插槽已兼容。

### 批次 5：迁移其余扩展面与通用镜像

**用户结果：** 设置、会话 View、右侧工具和复杂界面扩展都有可用路径，不因不能原生重画而被隐藏。

实施：

- 按“独立插件页面 → 设置/右栏 → 会话 View/消息内容 → root/hoc”推进，复用原 owner 的上下文和服务。
- 为不能独立嵌入的组合交付完整 DSH 兼容工作台；实现原生与 Web 的单一提交/应答归属，草稿和附件按明确合同交接。
- 待处理交互不能安全交接时留在原 owner 完成，不隐式允许、取消或重发；专属桌面服务缺失时显示具体兼容限制。
- 每替代一组 `mirror_*` 调用，验证其原有操作后才删除相应镜像路径和重复运行时；不先整包删除 `pluginui`。
- 窗口焦点、弹层、裁剪、DPI、快捷键与关闭在真实兼容表面上检查，不继续扩大“DOM 事件转发”来模拟全部控件行为。

**完成条件：** 各类已承诺扩展面有真实路径；至少一个依赖整体组合的扩展可用；两套呈现不重复操作、不抢交互。保留的原生 Adapter、兼容表面与尚缺平台服务有清晰范围。

**验收对象指名规则：** 批次 4 收尾时在进展记录中从实际已注册的扩展贡献里指名一个依赖整体组合（会话 View/root 组合，不能独立嵌入）的真实 DSH 插件作为本批验收对象；未指名不开工，避免验收时挑最容易的路径凑数。

## 6. 第三轮：原生收益与发布

### 批次 6：优化实际成本，完成恢复交付

在前两轮的稳定候选上，按架构第 11 节对照本体标准模式、本体精简模式与 MyGo；使用相同功能、插件和等价数据。

优化按测量结果选择，不预设“换 Go 一定更快”：

- 如主开销在扩展容器，先合并重复 Client Runtime 与隐藏界面组合，再检查可抽取的依赖；需要后台运行的插件保持运行。
- 如主开销在原生投影与绘制，处理实际重建、分配与布局热点，不重复改动已有效的缓存。
- 如主开销在 Host/插件，分别报告共同成本与 MyGo 可控制的增量；不能通过停用必要插件制造收益。
- 启动按首帧、可输入、Host ready、历史恢复分段优化；非必要初始化延后，必要能力不能标假 ready。
- 退出、更新与容器故障继续服从任务保护和数据恢复；安装、升级与回退覆盖当前实际交付输入，不清空 profile。

**完成条件：** 架构定义的响应、长对话和核心内存目标有当前测量；含前端插件的成本单独报告；当前候选的安装、恢复与受影响实际行为通过。缺项保持未通过，不将验证责任转给 CI。

**测量不达标的退出规则：** 若同条件对照下 MyGo 未优于本体（或优势落在测量噪声内），如实报告数值并三选一向用户请示：(a) 对最大可归因成本做一轮有界优化后复测；(b) 按实测发布，只宣称已测得的结果、撤销未证实的收益宣称；(c) 暂缓发布，回到架构层面重估路线。默认不静默发布带性能宣称的候选。

## 7. 交付与范围控制

每批按“实现 → 受影响的自动检查 → 实际操作/呈现 → 修复确认 → 稳定交接”推进。复用已有有效证据；新增改动只补相关覆盖。最终发布范围稳定且本地必要 QA 通过后才触发最终 CI，同一交付的历史未通过次数按原规则累计。

实施状态只需在本方案追加简短进展，或沿用已有权威记录并在此链接：当前候选、实际结果、未解决问题、下一步。无需再建一套并行总台账。

本方案不安排：Project 页面重设计、桌宠大改、全面换肤、第二套插件协议或市场、直接改本体仓库、自动升级日用运行时。核心功能或扩展兼容的必要相邻修复仍在对应批次范围内。

**首批执行范围确定为批次 1。** 批次 0 是其必要输入整理；先完成会话/草稿/交互与已有相关修复的收敛，再推进有依据的帧热路径优化。方案编写不代表已经授权执行这些程序修改。

## 8. 当前实现依据

- [主程序装配与帧入口](../cmd/whaleisle/main.go)、[已有 UI 性能诊断](../cmd/whaleisle/ui_diagnostics.go)
- [Session Store 与帧快照](../internal/store/session/store.go)、[快照隔离合同](../internal/store/session/view_clone.go)、[工作区会话创建](../internal/store/session/workspace_session_creation.go)
- [草稿绑定退役](../internal/ui/composer/draft_binding.go)、[草稿输入](../internal/ui/composer/draft_input.go)
- [聊天投影缓存](../internal/ui/chat/chat.go)、[侧栏及名称读取](../internal/ui/sidebar/sidebar.go)
- [文件关闭与保存](../internal/ui/files/close_save.go)、[Diff 工作台](../internal/ui/diffsurface/workbench.go)
- [插件 owner](../internal/desktop/pluginui/owner.go)、[当前导航合同](../internal/desktop/pluginui/navigation.go)、[鉴权网关](../internal/desktop/pluginslots/gateway.go)

这些源码位置用于确定复用与改造落点，运行效果由对应批次的当前候选证据确认。

## 9. 进展记录

### 2026-11-13 批次 0 盘点

工具链：go1.27.2 windows/amd64。工作区未提交改动盘点（19 个修改 + 7 个新增/未跟踪）：

| 改动组 | 文件 | 归属批次 | 现有证据 |
| --- | --- | --- | --- |
| 工作区会话创建：cwd 精确匹配、session/conflict 重试（`SessionId=nil` 重发）、Select 时清理 lastAction | `internal/store/session/store.go`、`workspace_session_creation.go` + 测试 | 1a | `[static]` 包测试通过；真实路径未走 |
| 提示归属与串扰：settingsActionText 扩充、新增 models/skills 动作过滤器、chat 内部动作前缀扩充、会话错误本地化、Notice 空消息关 toast | `internal/ui/settings/actions.go`、`cmd/whaleisle/settings_action_filter.go`、`internal/ui/chat/chat.go`、`queue_dock.go`、`internal/ui/widgets/notice.go` + 测试 | 1a | `[static]` 包测试通过；设置页真实呈现未走 |
| 桌宠测试 mock 跟进运行时（append/matchMedia/pet-sprite.js/sprite 尺寸） | `scripts/pet-*.test.mjs` | 方案外（既成事实） | `[mock]` node 测试 |
| 文档指向新架构 | `README*.md`、`docs/plan.md` | 无 | 已读核对 |

编译与测试基线：`go test ./internal/store/session ./internal/ui/chat ./internal/ui/settings ./internal/ui/widgets ./cmd/whaleisle` 全部通过（2026-11-13，本机）。以上为 `[static]` 层证据，不构成 §5 意义上的实机验证。

未解决保留项：

- 11 月 11 日点击卡死根因未复现，保持未解决（见 §2）。
- `Select` 时清空 `lastAction` 的语义边界（"session archived" 豁免清单是否完备）需要真实会话行为证据。
- settings 动作过滤器为字符串前缀合同，新 Host 动作出现时可能漏过滤，批次 1 落地时需确认是否有更稳的归属信号。

### 2026-11-13 批次 1a 状态

- 新候选：`tmp/opt-20261113/whaleisle.exe`（含上述未提交修复，go build 通过）。
- `[process]` 隔离启动冒烟：`WHALEISLE_ISOLATED=1` + 独立 `WHALEISLE_USER_DATA`，启动 trace 至 `window.created` ≈416ms，进程存活至超时退出；未接管协议/单实例锁，未触碰在用 profile。
- 源码核对：批次 1b 的归属机制已存在于现行实现——`ComposerTarget` 钉住 client+hostGeneration（`composer_submit.go`）、附件按 SessionID+HostScope 校验、提交回声/退役/队列放置完整（`composer_pending.go`）、审批按 `SourceRequestID` 归属且过期 owner 返回 expired outcome（`interaction/store.go`）、提问回答与 live waterfall 互斥（`projected_questions.go`）。1b 剩余工作以"完成条件"列出的 7 个真实场景验证为主，不做猜测性重构。
- 真实路径 QA（需要连接 Host 的候选运行）：工作区目录冲突重试、设置页提示归属、草稿切换/失败重提等仍为 `Not Run (no interactive target)`。

### 2026-11-13 批次 1a 修正 + 批次 2 首项

**修正（product-defect）：** `tools/session-recovery-check` 首次实跑发现工作区路径会话创建成功但不发布 `lastAction="session created"`（`Store.CreateSession` → `WorkspaceSessionTarget.Select` 无完成信号；新增的选择清空使其完全静默）。已在 `internal/store/session/store.go` 的创建 goroutine 按 ReportFailure 同款围栏补发；`"session created"` 本就在内部动作过滤表，不产生界面泄漏。

**批次 2 侧栏 I/O 迁移（已实现）：** `whaleAssistantNavLabel` 的 `Stat`/`ReadFile` 绘制路径 I/O 移除，改为 `observeWhaleNavLabel`——Host ready + 鲸鱼娘启用时经 `pluginrpc.WhaleSettings` 后台获取名称，按 Host 键快照发布，30s 周期补抓外部修改，设置页 name/identity 写成功后直接更新快照。绘制只读 `Model.AssistantName`。落点：`internal/ui/sidebar/sidebar.go`、`internal/ui/frame/frame.go`、`cmd/whaleisle/assistant_whale_settings.go`、`main.go`。

**证据（同日，真实 owned Host + 候选）：**

| 检查 | 层 | 结果 |
| --- | --- | --- |
| `go test ./internal/store/session ./internal/ui/sidebar ./internal/ui/frame ./cmd/whaleisle` | `[static]` | 全绿 |
| `tools/functional-workspace-check`（stage=runtime/，真实 Host，5 例） | `[rpc]` | 全过：首用工作区初始化、目录选择/创建、注册/重命名/移除保文件 |
| `tools/session-recovery-check`（真实 Host + 真实 mainView native save/restore，18 项） | `[rpc]`/`[process]` | 全过：create/rename/pin/archive/fork/delete、重启存活、草稿磁盘往返、native 保存/恢复 |

未解决保留项：交互式点击路径（双会话草稿切换、提交中编辑、失败重提 UI 手势）仍 `Not Run (no interactive target)`；批准/提问的 live Agent 触发按工具声明未验证。

### 2026-11-13 批次 2 帧耗基线（隔离实例，`WHALEISLE_UI_SAMPLE` + CPU pprof）

空载基线（无会话数据的隔离实例，90s）：148 帧、0 个 >16ms 慢帧、`lastView` ≈504µs、`snapBuilds=2`/`snapHits=6`（无逐帧重建）、goroutine 1 停放于 mygo 事件循环。75s CPU 采样 219.87s 样本中 99.65% 为 `runtime.cgocall`（阻塞等待），主项是 `startNativeUIRecovery` 的 JSON 解码阻塞读与窗口 composition watch——属等待非空转。

结论：空载无热点；批次 2 的"固定长历史+持续流"场景热点需要带真实会话数据与流式输出的候选运行，当前无可驱动负载，保持 `Not Run (no workload)`，不做猜测性优化。

### 2026-11-13 批次 1 交互 QA（真实 native 桌面 + 真实 owned Host + 本地确定性模型）

新增 `cmd/whaleisle/composer_flow_acceptance_windows_test.go`（build tag `composerflowacceptance`）：同一候选进程内起真实隔离桌面（真实 `mainView` 经 `CapturePage` 渲染）、真实 owned Host（32197）、进程内 Anthropic-SSE 确定性模型 fixture（`DEEPSEEK_BASE_URL` 指向 loopback，`QA_WAIT` 保持流直到取消）。驱动与按钮相同的产品入口：`submitComposer`/`Select`/`CancelRunning`/草稿状态机，不注入 OS 输入事件。

**结果（`tmp/opt-20261113/composer-qa6/proof.json`，pid 6140，单进程断言全部通过）：**

| 场景 | 层 | 结果 |
| --- | --- | --- |
| 工作区会话创建 ×2（真实 `CreateSession` → `session created` 信号） | `[process]` | pass |
| 草稿文本+附件跨会话切换保留（A 文本+1 附件 / B 文本） | `[process]` | pass |
| 带附件真实发送 + 提交中继续打字：用户消息恰好入列一次、助手流式回复到达、`typed while sending` 未被清除 | `[process]`/`[rpc]` | pass |
| 发送中停止：`session/cancel` accepted while streaming | `[rpc]` | pass |
| 过期编辑序号提交被拒：草稿+错误保留（`d.Error` 中文提示，文本未丢） | `[process]` | pass |
| 拒绝后重提：清空 EditSeq 再提交被接受并收到回复 | `[rpc]` | pass |

**过程中发现/修复的 harness 层问题（非产品缺陷）：** ① native 恢复看门狗以同可执行文件拉起 `exe --native-ui-recovery-watchdog <pid>`，继承 QA 环境变量后会重复执行测试流程并用自身的 readiness-timeout 覆盖主进程 proof——TestMain 已加看门狗参数防护；② 模型 fixture 每条 SSE `data` 必须带 `"type": <事件名>`，否则 deepseek provider 报 `MALFORMED_RESPONSE / SSE event type mismatch`（已按 `tools/core-host-check` fixture 修正）；③ `PromptBusy` 只在 admission/编辑 RPC 期间为真，运行中 turn 应观测 `Items[i].Running`。

**剩余 `Not Run`：** 一次性审批（需审批策略 fixture，本 fixture 未触发 approval pending）；IME/真实键鼠输入（`real-input-uia-setvalue` 已证明 UIA `ValuePattern.SetValue` 通道真实可达草稿，OS 键鼠/输入法组合态仍未覆盖）；断线重连的会话内恢复（草稿/附件跨进程重启已由 session-recovery-check 覆盖，会话内断线恢复未在本测试路径）。除 `real-input-uia-setvalue` 外其余步骤仍不得标 `[real-ui]`。

### 2026-11-13 批次 2 帧管线全链路仪器化验证（真实流式负载，`composer-qa17`）

为定位"store 驱动刷新期间 mainView 零重建"的疑点，沿 `win.Invalidate → contentChanged → conn.Changed(=rt.changed) → windowHost.requestFrame → Surface.RequestFrame(InvalidateRect) → WM_PAINT → runFrame` 全链加了计数器（`mygo.TestInvalidateCounters` / `mygo.TestFrameCounters`，vendored mygo 新增 `testhooks_counters.go` + `internal/windows` 计数点）。

**中流区间数据（workload 第 2 个长流进行到 1.5s 时快照，`visible=true minimized=false holdsFrames=false`）：**

| 环节 | 计数 | 判读 |
| --- | --- | --- |
| `win.Invalidate` 调用 | 331→707（653 次 CAS 合并丢弃） | 正常 coalescing |
| `contentChanged` 实跑 | 94→180 | postMain 任务正常执行 |
| `conn.Changed`（`rt.changed`） | 94→180 | 引擎侧全到 |
| `Surface.RequestFrame`/`InvalidateRect` | 107→222 | 请求全部到达 surface |
| `WM_PAINT` 到达 | 25→54 | OS 按节奏派发（合并属预期） |
| mainView 构建 | 25→54（与 paint 1:1） | **每个到达的 paint 都建了帧** |

**结论：帧管线无缺陷。** 流式期间 store 更新→invalidate→paint→构建全链路正常工作；之前观测到的"帧持平"是诊断采样落在流结束后的区间（无更新即无帧，属正确行为），非 `InvalidateRect` 未派发或 `holdsFrames` 卡死。

### 2026-11-13 批次 2 持续流帧耗基线（`composer-qa20`，可见窗口 + `WHALEISLE_UI_SAMPLE` 250ms + workload=3 长流回合）

**帧分布（t=0–48s，783 帧 / 54 慢帧 = 6.9%）：**

| 区间 | 场景 | 帧增量 | 慢帧增量 | lastView |
| --- | --- | --- | --- | --- |
| 0–10s | 启动+双会话建立 | ~73 | 6 | ≤3ms |
| 10–15s | 首回合流式（带附件） | +103 | **+21** | ≤7.5ms |
| 15–25s | 取消/拒绝/重提+流式 | +347 | +20 | ≤5ms |
| 25–30s | workload 长历史流式 | +198 | **0** | ≤4ms |
| 30–48s | 收尾+空闲 | +22 | 6（单次 27ms） | 27ms 尖峰 |

快照记忆化：`snapBuilds=715 / snapHits=6283`（~90% 命中）；`refresh=1098` 次调用无 nil-win；流式期间实测 ~30–45fps 派发。

**判读：** 慢帧集中在首回合（字体/布局缓存冷启动特征），历史最长的 workload 段反而 0 慢帧——无随历史长度恶化的热点。单次 27ms 尖峰出现在收尾。批次 2 的"先测量"要求已满足：**无病态热点**；首回合冷启动构建与偶发尖峰是有界微优化候选，不做猜测性重构。

**harness 修复（本轮）：** 会话创建/取消等待从 `LastAction` 瞬态字符串改为稳定状态观测（`SelectedID`/`Items[].Running`）+ `seenActions` 采样记录——修掉了三次假超时 flake（瞬态动作在 100ms 轮询间被覆盖）。

### 2026-11-13 批次 1 交互验收闭环（`composer-qa25`，可见窗口 + workload=2，10/10 通过）

同一候选二进制在真实隔离桌面（真实 `mainView` + 真实 owned Host + loopback 确定性模型）完成全部批次 1 交互场景：

| 场景 | 结果 | 证据层 |
| --- | --- | --- |
| 双会话创建（`session created` 信号） | ✅ | `[process]` |
| 草稿文本+附件跨会话切换不丢 | ✅ | `[process]` |
| 带附件发送+提交中编辑（消息恰入列一次、新打字保留） | ✅ | `[process]` |
| 流式中 `session/cancel` | ✅ | `[process]` |
| 过期编辑序号被拒（草稿+错误保留） | ✅ | `[process]` |
| 拒绝后重提成功 | ✅ | `[process]` |
| **一次性审批**：`pwsh` 工具 `sandbox_permissions:"danger-full-access"` 升级 → `approveEscalation` → `approval/request` waterfall → `interaction.Store.Pending` → `ResolveAllow` 恰一次 → 工具执行 → `tool_result` → pending 清空 | ✅ | `[process]`（真实 Host 沙箱升级审批，非 fixture 注入事件） |
| workload 持续流+帧链路（mid: `cc=580 req=1770 paint=1231 frames=501`） | ✅ | `[process]` |
| UIA `ValuePattern.SetValue` 写草稿 | ✅ | `[real-ui]` |
| `WM_CHAR` Unicode 输入（"你好"经真实 wndproc→`TextInput`） | ✅ | `[real-ui]` |

**审批触发结论（供复用）：** 本部署没有任何内置工具经 `tools/pre-execute` 返回 `ask`（fs 工具未挂审批门）；真实审批路径有二——①shell 工具（`pwsh`/`bash`）的 `sandbox_permissions` 升级参数走 `approveEscalation`；②ACP agent `session/request_permission`。harness 用①拿到确定性端到端审批。`one-time-approval` 三次 inconclusive 的根因分别是：工具选择未按优先级（选了 `edit`）、`rw_exec` 远程工具无升级参数、选中工具 schema 不广告升级参数——均已修正。

**`real-input-unichar` 语义边界：** `WM_CHAR` 是 IME 提交后的真实投递通道（真实 wndproc→`platform.TextInput`→草稿），但不经过 OS 输入上下文（HIMC）——**真 IME 组合态验证仍 `Not Run`**，合成 `WM_IME_COMPOSITION` 无法伪造组合串。

**间歇性 detached-send 卡死（已定位并修复）：** qa19/qa22/qa27/qa29 复现"发送静默无回执"（`send=false derr="" att=1 modelRequests=0`，草稿从未 detach）。在 `submitComposer`/`submitDetachedComposer` 各静默早退点加 `composer submit dropped` 诊断日志后，qa29 现场抓到真凶：**`requires-workspace` 门**——新会话 `SelectedCwd` 已是真实工作区且 `WorkspaceReady=true`，但 `Workspaces[].SessionIDs` 名册刷新滞后于会话创建（竞态窗口），`RequiresWorkspace` 误判无工作区 → `pickAndApplyWorkspace` 静默吞掉发送（`composer submit dropped: requires-workspace hero=true sel=... cwd=<workspace> wsReady=true workspaces=1`）。修复：`internal/ui/composer/workspace.go` 增加"`SelectedCwd` 匹配**已注册** `WorkspaceRow.Path`（`session.HostPathKey` 归一化）即豁免"——名册就绪时 cwd 属注册内根 = 归属证据，未注册的 cwd（如已删工作区 `C:/removed`）仍要求准入，"ready 名册为权威、不复活已删工作区"语义不变；`workspace_test.go` 新增 roster-lag/斜杠大小写/deleted-cwd 用例。同时 `composer_detached.go` 飞行管线包 2 分钟有界 ctx（`PrepareAttachment`/`Submit`/`AwaitSubmissionRetirement` 原用 `context.Background()` 可静默无限等待），并修复 `AbandonSubmission` 覆盖原始 stage 错误导致 `CompleteDetachedSend` 误判成功静默丢草稿的问题。`composer submit dropped` 诊断日志保留为生产面包屑。

### 2026-11-13 批次 3 首批工作面证据（`composer-qa31`，可见窗口，12/12 通过）

QA 工作区在 fixture 阶段 `git init` + 基线提交；场景经产品真实入口驱动：

| 场景 | 结果 | 证据层 |
| --- | --- | --- |
| 文件打开+预览：`state.layout.Workspace.OpenFileResource(files.Model{真实 ReadRemoteContext/ReadTextPageContext}, "qa-note.txt")` → 真实 `WorkspaceFileState` tab → 异步 workspacefs 读取 → `state.filePreview` 含文件内容 | ✅ | `[process]`（dock 路由+tab 生命周期+读取管线全真，非直接设 state 字段） |
| Diff 面板：修改已跟踪文件 → `rightPaneTab="diff"` 路由 → `paintWorkbench` 渲染期同步 `git.DiffHEAD` → `state.diffText` 含新增行 | ✅ | `[process]`（真实 git repo + 真实 DiffHEAD 输出） |

**harness 修正（非产品缺陷）：** 此前 qa28 两场景超时——①共享 230s deadline 被 workload 耗尽导致等待立即返回；②只设 `state.selectedFile` 未调 `OpenFileResource`，tab/读取从未启动。已改为 `win.Update` 内驱动真入口 + 每场景独立 45s 预算。

**qa30/qa33 间歇异常（已定位并修复）：** `session 2 created` 两次超时。根因：`WorkspaceSessionTarget.currentLocked` 栅栏（followGeneration/cwd/workspaceErr/workspaceCurrentLocked）在会话 1 收尾竞态中失稳 → `Create` 返回 "owner changed" → `ReportFailure` 同一栅栏再判一次拒绝写入 `lastAction`——**纯静默丢弃**。修复：`ReportFailure` 拒绝支路增加诊断面包屑；测试侧重发手势+`seenActions`/`LastAction` 诊断输出。qa34 起 `createdSignalSampled=true` 连续通过。

### 2026-11-13 批次 3 文件+Diff 工作面全绿（`composer-qa45`，可见窗口，16/16 通过）

新增场景均经产品真实入口驱动：

| 场景 | 结果 | 证据层 |
| --- | --- | --- |
| `file-edit-switch-return`：文件 tab 注入 dirty 草稿（`FileTab.Draft != Preview` + `Editing`，三级状态一致写入）→ `rightPaneTab` diff → `OpenFileResource` 重新激活文件 dock tab → 权威 `FileTab` 草稿+编辑态存活，磁盘未隐式写 | ✅ | `[process]`（权威 `pane.Files.Tabs` 状态，非镜像字段） |
| `external-edit-conflict`：外部 `os.WriteFile` 修改已打开文件 → 资源观察者检测到 → 冲突状态"草稿和已加载内容已保留" + `readBlocked` + 草稿保留 + 磁盘未被隐式覆盖 | ✅ | `[process]`（观察者路径=产品真实检测路径；显式 `OpenPresentedPath` reread 为备选路径，preview 采纳差异如实记录为 detail 不断言） |
| `diff-stage-reflect`：外部 `git add` → `git.Changes` 报 staged + scoped diff 含改动 + commit 数不变（无隐式 commit） | ✅ | `[process]`（真实 git；Diff 菜单项 UI 驱动仍为声明缺口） |
| `terminal-route`：`restartTerminal`（真实打开手势，非 `ensureTerminal` 刷新路径）→ panel 挂载 → 路由回 files → panel 仍在 | ✅ | `[process]` |

**harness 修正（非产品缺陷，3 处）：** ①`"files"` 路由经 `selectKind` 激活的是文件**浏览器** pane（`workspace_panes.go:140` 专门跳过带资源的 pane）——返回文件 tab 的真实手势是 `OpenFileResource`/`activateDockTab`，此前断言活动 pane 投影镜像导致误报；②顶层 `state.fileEditDraft` 等是逐帧投影（`finishFileProjection` 用活动 pane 的 `WorkspaceFileState` publish，diff 激活时恒为空）——断言必须落在权威 `FileTab`/`pane.Files`；③`state.fileTabs` 是逐帧拷贝，`textReader`/`readTab`/`binding` 值字段不与 `pane.Files.Tabs` 共享——读操作必须绑 pane 实例。

**已知时序现象（观察中，未升级为产品缺陷）：** `WorkspaceFileState.capture` 会把镜像标量（含 `Selected`）写回 pane 状态；投影切换期间若镜像被另一 pane 的空 `Files` publish 清空，`s.Selected` 可能瞬时为 `""`（qa43 观测到 `selectedTab` 一度为 nil、观察者退役）。`FileTab` 权威态不受影响且 `selectTab`/`restore` 会自愈——记为观察项，若用户路径可复现编辑区空白再升级为 `product-defect`。

**批次 3 剩余 `Not Run`：** Diff 暂存/还原菜单项 UI 驱动（`[real-ui]` 缺口，无外部驱动点）；终端焦点交接细节；隐式 save/commit/push 不存在性的完整检查（draft 未隐式保存、stage 未隐式 commit 已证）。

### 2026-11-14 批次 4 首批：左栏扩展空间"更多"菜单链路（`composer-qa48`，rc.4 vendor + 配套 Node，可见窗口，19/19 通过）

按 `ARCHITECTURE.md` §5/§6 新契约实施 `sidebar.panellist.actions`（"更多"菜单）端到端接入：

| 工作面 | 结果 | 证据层 |
| --- | --- | --- |
| `sidebar.panellist.actions` 加入 JS/Go 公共插槽白名单；新增 `Owner.PublicSlotRegistrations`（catalog 级注册观察——actions 注册不挂 DOM，不能用 `Entries()` 门控） | ✅ | `[static]` + `[process]`（qa48 观测 `registrations=1 bundledNative=true`） |
| bundle/native 判定修正：`ui-entry-customization` 等 bundle 注册以 `whaleBundledMarker` registrant 呈现时同样视为内置（此前仅认显式包名） | ✅ | `[process]`（`bundledNative=true`） |
| Pin 为内存态（`nativePlugins.navigation.unpinned`）：取消固定→更多菜单、固定→回栏、`plugins` 永不可取消、owner/page 重置恢复注册默认、不持久化 | ✅ | `[process]`（`extension-panel-pins`：unpin/repin/pluginsPinned 全 true）+ 单测 4 项 |
| 侧栏"更多"行+菜单（展开态 `sidebar.go` + 收起态 `rail.go` 共用组装）；门控=有 actions 注册才显示（rc.2 无注册→无更多入口=正确旧契约降级） | ✅ | `[static]` + 单测 `TestSidebarPanelMoreGateFollowsRegistrations` |
| "Create your plugin"→`startCreatorPluginDraft`：草稿播种+cordis 预设+**不发送**（`whenDraftExists:'append'` 语义：非空草稿追加、保留附件/引用） | ✅ | `[process]`（`extension-panel-creator`：草稿含"带左侧栏入口"+`agentPresetPick=cordis`+`model.requests` 不变） |
| 选中插件视图撤销时区分 HMR 重建与真禁用/卸载：2.5s 宽限期，逾期 toast"扩展已不可用"后关闭 | ✅ | `[static]` |

**新增 QA 场景（qa48，`WHALEISLE_VENDOR=runtime-rc4\vendor` + `runtime-rc4\node\node.exe`）：** `extension-panel-more`/`extension-panel-pins`/`extension-panel-creator` 三个 mark 全过；16 项存量场景在 rc.4 下全过（含 workload-streams 帧耗与 UIA 真实输入）。

**rc.4 运行要求实测：** rc.4 Host 校验 Node 来源——系统 Node 24.20 被拒（`selected Node differs from the verified Host session runtime`），必须使用 `runtime-rc4\node\node.exe` 配套捆绑 Node。该约束已属 `sessionRuntimeProfile` 语义，如实记录。

**harness 修正（非产品缺陷）：** ①`startCreatorPluginDraft` 内部自带 `win.Update`——QA 从 `win.Update` 回调内调用会 UI 线程重入死锁（两次 ~20min 无超时卡死的根因）；②`mark` 增加 stderr 进度日志，驱动不再是无输出的黑盒。

**批次 4 剩余 `Not Run`：** `[real-ui]` 真实菜单点击（展开/收起态更多行渲染与点击需 UI 树驱动，现有断言停在状态/组成层）；actions 注册的撤销→菜单消失路径；HMR 瞬时空→宽限期→恢复的真实插件路径（需真实可重载插件）；rc.2 回归跑（旧契约降级分支逻辑已证，rc.2 全量套件未重跑）。

### 2026-11-14 批次 4 深化：控件层验证 + 门控契约修正 + rc.2 降级回归（`composer-qa49`）

**门控契约修正（实测上游源码定性）：** `ui-sidebar` 的 `hasEntryActions && <PanelMoreMenu/>`（`client.js:551`）证明"更多"启动器仅在 `entriesOfSlot("sidebar.panellist.actions").length > 0` 时存在。此前实现附带"有未固定项即显示"的兜底偏离契约，已修正为 `bundled || (有注册 && 有未固定行)`：bundled 注册自带 browse/create 内容即可开门；foreign 注册算门控但只在有未固定行提供可渲染内容时才开门（外来 action owner 组件无法原生渲染=如实降级，见下）。

**控件层验证（`ui.NewTester` 真实命中测试层，4 个新用例）：** 点击"更多"行→真实菜单项顺序（未固定行/固定到左侧栏子菜单/分隔/浏览更多插件/创建你的插件）→`ChooseMenuItem` 逐项驱动：`h-notes` 打开且不回钉、`OnOpenMarket`、`OnCreatePluginDraft("leftbar")`、子菜单回钉；行右键→"从左侧栏取消固定"→回调；无注册→"更多"行不存在；rail 折叠态同路径。

**HMR 宽限期单测化：** 撤销视图决策抽出为 `pluginViewMissingDecision`（纯函数）：活桥首失开始计时、2.5s 内保持静默打开、逾期通知一次并关闭、死桥立即静默关闭（无 HMR 提示）——4 个断言点全覆盖。

**rc.2 降级回归（`composer-qa49`，默认 vendor + 默认 Node）：** `extension-panel-more` 实测走旧契约分支——"no sidebar.panellist.actions registrations (old-contract runtime: correctly no More)"；pins/creator 两 mark 亦过。`external-edit-conflict` 一次超时（qa45 同 runtime 通过、qa48 rc.4 通过）——既有观察项抖动，与本批改动正交。

**环境约束补记：** `WHALEISLE_NODE` 显式覆盖在两个 runtime 上均被 Host 拒（`selected Node differs from the verified Host session runtime`）；rc.4 的正确指定方式是 `runtime-rc4\node\node.exe` 配套捆绑 Node，rc.2/默认树不传该变量走自身解析。
