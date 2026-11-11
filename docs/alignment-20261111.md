# 鲸屿 MyGo 对齐执行

目标：完成 2026-11-10 对比确认的全部差距。用户 2026-11-11 明确要求 Project 先放缓、重新设计；本轮不继续其页面重排，只处理共用 Host 升级所需的兼容接线，保留现有项目数据和入口。

## 当前范围

| 项目 | 状态与完成条件 |
| --- | --- |
| Host / 插件版本 | rc.4 完整构建入口已串起安装版 Host、新版手机 Electron ABI 依赖、旧 `dshbot` 0.2.0、main 四插件及含 4 项兼容修正的 `dsh-acp` 0.1.0。默认 `runtime/` 仍保留 rc.2 未切换；当前完整交付输入更新为 `tmp/alignment-20261111/runtime-complete-policy`（历史 `runtime-complete-fixed` 保留，旧手工 `runtime-rc4` 不作交付输入） |
| 队列排序 | 源码与内存窗口交互通过：真实拖放、方向键、失败反馈；rc.4 Host/Store 排序一致，原生窗口综合验收待做 |
| 跨工作区置顶 | 源码与内存窗口交互通过：顶部区域、去重、工作区标签、折叠恢复、搜索展开和可见批量选择；原生窗口综合验收待做 |
| 手机端 | 独立 rc.4 运行时与 Whale UI 已通过浏览器检查；真实本地中继/daemon/Go Host 配对、发送、断线、重载后历史与草稿恢复通过。实体手机/公网仍待验收 |
| 主动任务 | 已接持久任务、Inbox/Outbox、Host 执行/恢复/调度、GitHub 来源、结果动态、Cordis callback 工具动作来源识别（拦截同名第三方伪造 `read/write`）、具体工具审批与 ACP `auto-allow` 任务回合压制、普通续聊恢复、以及审批后飞书结果投递，均通过真实 rc.4 Host 验证。L3–L5 更细业务动作策略、真实飞书账号外发、OS 通知中心显示/点击与真实外部模型仍待验收 |
| 后台资料任务 / IM | 已接入本体 main 插件与原生任务面板；真实 Host 资料快照/执行/回传/停止/恢复通过，飞书授权边界与冷会话绑定通过。真实飞书账号收发待授权和验收 |
| ACP | 已接入原生管理/登录/配置界面、Composer Agent→模型选择、T3 ACP 装配（4 项兼容修正）、MCP stdio 桥（`t3-mcp-bridge.cjs`）真实工具消费、会话停止/重启 `session/resume`、以及主动任务回合 `auto-allow` 统一门禁约束。真实第三方 ACP Agent CLI 安装、OAuth/账号与外部模型计费使用仍待验收 |
| 更新分发 | 已基于 `runtime-complete-policy` 生成完整打包目录与正式安装包 `tmp/alignment-20261111/package-rc4-policy/WhaleIsle-Setup-0.0.0-dev.exe`（含 `runtime-files.nsh`、blockmap、`SHA256SUMS.txt`），包内重定位运行时核心链路、隐藏原生主窗口及隔离 `APP_ID` 的静默安装/覆盖升级/卸载已通过。正式代码签名、远端分发源消费及覆盖用户现有安装仍待确认 |
| 实际使用验收 | 待做：真实外部模型、窗口真实输入/IME、正常用户退出竞争、覆盖真实安装版升级；优先 API 与隐藏窗口，不静默接管用户桌面 |
| Project 重设计 | 按用户要求放缓，另行推进设计；不以旧版页面对齐替代新设计 |

## 本次源码对比的口径

本体已发布基线是 0.3.14 / Harness rc.4；main 对比点为 `c9d805a7de0e0b7b80d5768cd30ab489275d1b3b`；ACP 另属于 `feat/acp-agents` 的 `18994faa03` 加未提交改动。ACP 当前在 `dsh-acp/`，不计入已发布版的稳定能力。MyGo 对比对象为当前含未提交实现的工作树，不等于用户安装版本。

- **已完成本轮补齐的实现：** 主动任务服务、原生界面、Host 执行/恢复、自动调度、GitHub 来源、本地结果交付、Cordis callback 工具来源识别、具体工具审批、审批后飞书投递，以及 ACP 外部 Agent 装配、原生配置/登录、Composer 模型入口、MCP stdio 桥消费和主动任务 `auto-allow` 压制均已在下述批次落地（L2–L5 当前仍共用受门禁约束的工作区写上限，未拆更细业务分级）。既有普通定时任务、dshbot 任务和后台资料整理保留其原有入口。
- **运行时与安装包集成进展：** `scripts/build-harness.ps1 -RuntimeSourceResources ... -AcpSource ...` 已对 rc.4 统一完成 Host、手机、旧 dshbot、main 四插件和 T3 ACP（含 4 项兼容修正）装配；`scripts/package.ps1 -RuntimeRoot` 已产出完整打包树与正式 `WhaleIsle-Setup-0.0.0-dev.exe`。默认 `runtime/` 与用户已安装环境尚未切换，等待最终授权。
- **已有有效实现：** rc.4 核心会话、队列与闲置释放、顶部置顶、手机沙盒与本地重连、后台资料任务、IM 冷历史绑定、主动任务来源门禁与 ACP 协议/MCP/策略闭环，证据见下文。它们的外部设备/真实账号/原生操作缺口单列，不能整体列为未实现，也不把本地夹具验证冒充为真实外部账号验收。
- **仍缺使用证据：** 当前 rc.4 候选的真实外部模型、OS 输入/IME、焦点/弹层、正常退出、实体手机/公网、真实 IM 收发，以及完整安装升级回退；复杂插件富内容、Pad、Office、桌宠/语音、SSH 等旧对齐域的实际使用缺口也未被本轮定向检查自动关闭。
- **旧审查数字不作当前完成率：** 367 项、758 动作与 R3/R4 修复记录有重叠且基线较旧，本轮没有重新执行全量动作，不据此报告“还剩几项”或百分比。Project 继续按既定要求暂缓重设计。
- **本体已有与共同建设分开：** 本体 main 的主动服务仍以持久状态、策略函数、调度和 IPC 为基础；其架构文档列出的完整自主助理闭环是目标，不能全部算作已发布能力。MyGo 本轮的执行器、工具审批桥和来源轮询属于继续完成既定目标；ACP 则是本体开发分支差距。

## 已有有效证据

11 月 10 日 rc.2 的真实 Host + 本地模型核心链、审批往返与隐藏原生窗口历史呈现通过，见 [核心验证](core-flow-verification-20261110.md)。该证据不能代替 rc.4 或新增功能验收。

开始时主仓 `31915c16`，有 141 个已修改文件与 28 个未跟踪文件（包括上一轮测试收尾修复与报告）。本体安装版 0.3.14 / Host rc.4；本体 checkout `feat/acp-agents`，ACP 含正在开发的未提交改动。两处均保留已有工作。

本目标尚未触发 CI；历史累计 CI 次数未核实，最终触发前再查。本轮先产品实现，再准备和运行受影响的验证。

## 11 月 11 日进展

- 新运行时位于 `runtime-rc4/`，旧 `runtime/` 保留，默认启动尚未切换。支持两套精确 Host pin 与 readiness 门禁，rc.4 API 已重生成；自定义 session projection decoder 保留在手写文件。
- `tmp/alignment-20261111/rc4-core-result.json`：真实 rc.4 Host + 本机 HTTP/SSE 模型，创建、发送、文件工具、停止、续聊、重启恢复通过（36.25 秒，6 次本地请求）。该旧版工具首条 detail 硬编码为 rc.2，实际传入的是 `runtime-rc4`；工具输出已修正，未为修改标签重复验证。
- `rc4-queue-idle-result.json`：队列移动后 Host/Store 同序，follow 中闲置会话保留，切走后释放 Agent，切回恢复历史通过（10.906 秒，2 次本地请求）。两个流程均完成自有 Host/模型进程与端口清理。
- 置顶/拖动新增行为检查、原有键盘排序及失败提示、`TestPublishedResultProjectionUnion` 已通过。`tmp/alignment-20261111/ui/` 四张内存窗口截图已目视检查，覆盖明暗、窄侧栏、工作区标签和拖放插入线；这些不替代 OS 输入/IME 验收。
- `scripts/installed-mobile-runtime.mjs` 从本体安装文件补齐独立 rc.4 手机运行时，沿用已装 SQLite/PTY 的 Electron ABI，避免拿普通 Node 加载 Electron native addon。普通页面维持 DENY，只有同源 Whale UI viewer 可被沙盒框架嵌入。
- 手机运行时 native probe（Electron 44.9.1 / Node 24.22.0 / ABI 149）和 Go 的实际资源加载校验通过。`mobile-ui-fixed-state/mobile-whale-result.json`：独立无头 Chrome 真实加载沙盒 viewer，输入值回传、计算、源码切换、流式更新保留状态、390/320 宽度通过；两张截图已检查。首次检查错误地读取旧平面 state，按当前 `whale-ui-state` envelope 修正工具后通过，产品 renderer 无须修补。
- `scripts/session-runtime-dshbot.mjs` 将旧 dshbot 0.2.0 的完整独立依赖复制到 rc.4，并仅为 8 项既有 DSH peer 添加精确 rc.4 版本。`rc4-dshbot-hidden-preset-result.json`：实际 Host 注册隐藏群聊预设、创建两个联系人和群聊、两个成员真实本地模型回复、Host 重启后目录与历史恢复通过（14.475 秒，5 请求）。首次工具从用户选择列表寻找隐藏 preset 导致失败；改用 `agentPresets/read` 按 ID 检查。没有改变预设 hidden 属性。
- `rc4-phone-observed-disconnect-result.json`：真实 Go Host、Go 手机桥、原始远程 daemon、本地 Cloudflare relay 和无头 Chrome，配对、桌面历史、手机发送、主动断开实际 WebSocket 后观测离线与恢复、重载后凭据重连及草稿恢复通过（15.913 秒，3 次本地请求）。浏览器结果位于 `core-host-prompt-3446069083/phone-browser-result.json`；截图处于抽屉关闭动画中，不据此认证手机静态外观。工具已补充等待抽屉关闭，未为截图重复整个有效检查。
- 手机联调发现 Composer 的无会话发送只传 cwd，未传已注册的 workspaceId，产生不属于任何工作区的会话。已在 `CaptureComposerTarget` / `CaptureComposerGestureTarget` 捕获已知 Host 工作区身份，并在创建时使用；未登记目录继续保留原调用契约。对应真实手机链及请求归属/跨导航/未登记目录回归通过。
- 手机首次工具误把 Away 模式当作本机 :3180 托管；已用独立生产 SPA server 模拟 Away 页面托管。随后目录检查定位到上述真实归属错误；失败收据保留，没有放宽手机会话过滤。所有自有 Host、模型、中继、浏览器及远程 daemon 已退出。
- `scripts/native-release-catalog.mjs` 的安装运行时识别已补齐 rc.4 pin/profile；完整打包、安装升级及默认运行时切换仍待最终集成。
- 完整交付仍缺主动任务、ACP、更新分发和安装版验收；后台资料任务/IM 的当前进展见下。rc.4 核心链通过不能据此切换或发布不完整运行时。

## 后台资料任务与 IM

- `scripts/session-runtime-desktop-plugins.mjs` 已从本体 main `c9d805a7de0e0b7b80d5768cd30ab489275d1b3b` 导入 `dsh-whale`、`dsh-im`、`dsh-task-control`、`dsh-platform-session` 的完整包及私有依赖。只接受这四包相对目标 revision 无差异的源码；原包在 `runtime-rc4/desktop-plugins-before-c9d805a7de0e/`，本体源码只读。
- 原生“后台资料整理”面板可选择文本文件、填写目标和完成条件、查看来源/结果/投递状态、停止与继续、打开执行记录，并指定/撤销飞书任务授权人。任务执行复用本体的授权输入快照和受限 worker，关闭面板不终止后台任务。新建表单在接受任务后收起，优先显示任务结果。
- `rc4-whale-tasks-delivery-result.json`：真实 rc.4 Host + 原生 typed RPC + 本地 HTTP/SSE 模型通过资料读取、带引用报告、会话投递、取消模型请求、源文件改动后继续原快照、重启后 interrupted 且不自动重跑、明确继续及 owner grant/revoke（26.054 秒，8 次本地请求）。Host、模型、监听均已清理。首次检查把 stop+resume 的 generation 误期望为 2，按实际两次递增修正；第二次在结果尚投递时立刻建下一任务被 Host 拒绝，前端和工具现均等投递结束。
- Native Store/RPC 定向检查通过：不明提交结果保留草稿且需先刷新、旧 Host 的回执和取消不能影响新 Host、停止/继续只传 taskId、owner 必须明确具体身份。原生 UI 指针操作与明/暗/420 宽度渲染检查通过，截图在 `tmp/alignment-20261111/ui/whale-task-*.png`，已目视检查。
- `desktop-im-source-stdout.log`：从同一 revision 复制的 14 项任务授权/投递/会话归属检查针对 staged 源码通过；使用替身 SDK，没有真实飞书外发。另 `desktop-im-live-binding-stdout.log` 的 opt-in 真实 Host 检查通过：绑定冷历史会话、保留原会话身份和内容、没有创建替代会话（6.35 秒）。该隔离检查刻意禁用 web runtime，出现 `harness-dev-context` 等待 webApp 的提示；不影响被检查的 Host RPC。
- 一次 `imonboard` 名称筛选未匹配任何测试，不计为通过。当前原生绑定接口仍是 `bot.workspace.set` / `bot.preset.set`；本次没有修改原有登录或账户配置。

## 主动任务持久化与原生收件箱

- 新 `internal/desktop/proactive/` 复用本体 `task.created`、`task.transitioned`、审批和调度事件格式，保存至 `userData/assistant/proactive-events.ndjson`。一个服务序列化 UI 与控制接口写入，审批/任务/关联定时器一次替换落盘，成功后才发布内存状态；损坏记录或已发生外部写入时拒绝覆盖。
- `POST /desktop/proactive` 已接原有认证控制服务器，支持任务创建/读取/状态、Inbox/Outbox、审批提案和调度创建/取消/显式 poll。Host 持有的 token 不代表用户同意，`approval/decide` 返回 403；原生界面才调用具体 approval ID 的 Decide。等待审批时拒绝普通状态跳转，定时 poll 也不消费该任务。
- 助理页新增“任务收件箱”：记录任务、查看来源/详情/状态/证据与安排、批准/拒绝指定操作、查看决策历史及取消记录。写入在 UI 线程外执行，失败保留草稿；离开原鲸鱼娘设置表面时停止该表面的轮询。当前不提供未接通的“自动开始执行”按钮。
- `TestJournalRestartApprovalsAndSchedule` 等检查已实际写入、重开与读取临时磁盘日志，覆盖超过 64 KiB 的正文、重复操作、批准后恢复、单次只触发一次、循环时间、取消关联审批与安排、并发提案、外部写冲突和损坏记录保留。`TestProactiveControlRoundTripAndUserDecision` 运行真实 loopback HTTP 服务器，验证匿名拒绝、提案到 Outbox、控制接口不能自批、用户决定后 HTTP 与磁盘一致。
- `TestDesktopProactiveJournalCompatibility` 已 opt-in 使用本体实际 `src/main/proactive/service.js` 与 Node 24.21.0：本体读取 Go 创建/批准/安排的记录，再由本体创建任务/审批，Go 重开后拒绝并恢复成功。环境参数为 `WHALEISLE_PROACTIVE_DESKTOP_SOURCE=C:\AI\Deepseek-Harness-Desktop` 与 `WHALEISLE_PROACTIVE_NODE=C:\AI\tools\node-v24.21.0-win-x64\node.exe`。这是隔离日志互读，未启动用户本体或调用模型。
- `TestProactivePanelPersistAndRetainDraftOnConflict` 通过原生主装配回调验证保存与冲突保稿；`TestProactiveInboxDecisionAndRendering` 通过实际 MyGo Tester 指针点击、生产服务写盘、重开决策检查。800 宽/420 窄、明暗截图 `tmp/alignment-20261111/ui/proactive-inbox-*.png` 已目视检查。首次通过后补了中文风险/历史文案，原测试的子串断言把“你已批准此操作”错当作残留按钮；改为精确按钮文字断言后通过，未改变审批逻辑。
- 本存储/面板批次不覆盖 OS 输入/IME；随后执行与恢复实现见下一节，不能将本存储批次的结果单独用作主动系统全交付证据。

## 主动任务实际执行与恢复

- `internal/desktop/proactive/execution.go` / `host.go` / `runner.go` 已接入实际 Session API。原生“开始执行”先把 Host 家目录、工作目录/已注册 workspaceId、Session ID、request ID 和提示词写盘，再创建/跟随/提交会话。执行结果进入“待核对结果”，用户可查看会话、明确继续或确认完成；Host 控制路由不能自行写入原生执行状态或授权可执行日程。
- 原生任务创建可选 L1–L5。L1 经 `internal/harness/proactive-readonly.mjs` 的认证、只向下收紧权限的桥接设置 `read-only` + `ask`；L2–L5 使用已有 `workspace-write` + `ask`。这只是运行权限上限，不表示 L1–L5 的所有动作策略已接入工具调用，也不授予外部发送/发布权限。默认权限与用户其他会话不变。
- 日程执行由与当前 Host 同生命周期的 Runner 处理，面板关闭仍运行。每次到期原子创建独立子任务和执行身份，再消耗一次性日程或推进循环日程；错过多次合并一次，同一日程有未定执行或待批操作时不叠加。已有协议型 `schedule/poll` 不消耗原生执行日程。
- 接收过或结果不明的请求仅核对原 Session，不自动重发；只有尚未提交的 prepared 执行可发送。历史按 `source.rpcId` 定位原回合，长历史会分页。运行中 Host 重启后若没有已完成结果，显示“需要补充”，明确“继续任务”才产生新 request ID，沿用原 Session。
- 停止先撤销本地观察并等待其退出，再取消具体 Host Session。审批提案同样暂停具体执行，保留 `waiting_approval`；批准本身不自动发模型请求。现有退出/停止/更新任务保护期间冻结 Runner 的新任务/定时准入，取消生命周期操作后恢复。Runner 退出等待所属观察完成，避免旧 Host 回执改写重开的服务。
- `rc4-proactive-create-result.json`：真实 rc.4 Host、原生执行器、本地 HTTP/SSE 模型，实际文件读取、只读投影、Host 重启读取已完成结果且请求数不增加、明确续聊同 Session/新 request ID、停止真实 HTTP 请求、真实定时器生成独立子任务、历史两个 request ID 各仅一次通过。耗时 12.995 秒、6 次本地模型请求，所属 Host/模型/监听均关闭。
- `rc4-proactive-interrupted-result.json`：运行中停止并重启实际 Host，恢复 `waiting_input`、没有新增模型请求，明确继续后原会话完成，通过（10.646 秒、2 请求）。`rc4-proactive-approval-result.json`：审批请求取消实际模型 HTTP 请求，任务停留 waiting_approval，用户决定后明确 Continue 才完成下一轮，通过（6.321 秒、2 请求）。所有外发仅合成目标，未发送真实消息。
- 首次集成失败为 follow 的 `address` 使用普通 `[]byte` 编成 base64，改用 generated `SessionFollowRequest` 保留 JSON；第二次失败为 `session/projections` 不存在时返回 null 而不是 error，按本体契约补创建分支。失败收据 `rc4-proactive-first-result.json` / `rc4-proactive-follow-result.json` 保留，均为模型请求 0，不抹去失败。
- `execution_test.go` 定向检查覆盖循环安排跨重开、未定执行不重叠、待审批阻止下一次、取消、外 Host 不可认领、迟到结果不覆写新请求、Host 不可自授可执行日程、生命周期冻结。原生创建级别/冲突保稿、开始/继续/完成/打开执行会话/安排按钮已走 MyGo Tester；明暗与 420 窄屏截图 `ui/proactive-create-narrow.png`、`proactive-schedule-*.png`、`proactive-result-dark.png` 已查看。单行时间字段的过高文本框改为已有 TextInput 后定向复验通过。
- `tmp/alignment-20261111/whaleisle.exe` 最终构建通过，未切默认运行时、安装、推送或触发 CI。上述为 rc.4 本地模型与内存窗口证据，尚非 OS 输入/IME、真实外部模型或当前安装包验收。
- 本执行批次结束时，来源自动发现、结果主动投递及 Policy Evaluate 与工具执行的完整联动仍缺。随后来源与本地交付见下一节；Outbox 仍是具体提案/人工决定，并非所有工具执行的统一权限门禁。

## GitHub 来源与本地结果交付

- 新 `internal/desktop/proactive/sources.go` / `github.go` / `monitor.go` 已接仓库订阅、暂停/恢复、立即检查与 15 分钟轮询。使用本机 GitHub CLI 的现有认证，只执行仓库元数据与 issues 的 GET；issues 响应包含 PR，按稳定 issue ID 去重，不对同一 PR 再请求 pulls 列表。全部分页成功后才原子保存新增任务和成功时间；失败保留原游标。来源暂停/重配会取消在途读取，旧 revision 回执不可导入。
- 原生“GitHub 来源”允许 owner/name 或 github.com 仓库地址，来源配置写盘后应用重启继续轮询。默认没有订阅；新增事项只进入收件箱，不自动调用模型。来源修改可更新尚未处理的提案；已经执行的任务保持启动时目标/会话/结果，另显示最新来源内容与关闭状态。
- 真实检查发现本体仓库已从 `ChisaAlter/Deepseek-Harness-Desktop` 改名为 `ChisaAlter/WhaleIsle`（GitHub repository ID `1333141598`）。首次严格按旧路径校验合法重定向结果失败；现先读取所选仓库 metadata，再按正式 full_name 校验每个事项链接，稳定 issue ID 使别名订阅不产生重复任务。未改本体 Git remote 或源码。
- `TestGitHubLiveReadOnlyDiscovery` 对 MyGo 仓库真实 GET 返回 0 项，证明空列表通路；随后对本体旧名称真实 GET，经上述改名修复后读到 5 个开放事项，5 项均进入隔离 Inbox，模型请求 0。没有订阅到用户实际 profile，也没有修改远端议题/PR。网页工具不可用，协议依据本体源、CLI 帮助与实际 API 响应核对。
- 新 `notices.go` 将完成摘要、Task ID、Session ID、未读状态与执行完成一次写盘，Monitor 主动消费到应用内动态及系统通知适配器。应用内可查看、标已读、查看历史；助手入口显示未读数。桌面提醒只含简短标题/打开提示，全文保留应用内；桌面回执与用户已读分开。崩溃前 sending 回执恢复为 unknown，不自动重发；只有显式重试重新提交。
- `rc4-proactive-delivery-result.json`：真实 rc.4 Host + 本地 HTTP/SSE 模型 + 生产 Runner/Monitor，完成结果进入持久未读动态，确切 Session/摘要及后续已读状态在重开后保持，通过（16.210 秒、1 个本地模型请求）。未调用 OS toast 或外部消息，所属 Host/模型/监听均关闭。
- 定向检查通过：连续分页/大整数 ID/PR 类型/外仓链接拒绝/坏页不提交半列表、发现去重/跨重开/来源更新保留执行/失败游标/暂停迟到响应、真实 Monitor 取消与恢复、仓库别名去重；结果一次落盘/重复完成不重复通知/发送结果不明不重试/不可用保留未读/关闭时保留 unknown。原生来源回调、暂停不清另一份草稿、失败保稿和无窗口降级也通过。
- 原生界面“查看动态/标为已读/重试提醒/添加/暂停/立即检查”实际 MyGo Tester 点击通过，800/420 宽和暗色截图 `ui/proactive-sources-*.png` 已查看。仍未验证操作系统通知中心的真实显示/点击；隔离运行继续禁用系统注册，不把记录了通知回执等同于用户看到提醒。
- Monitor 在主窗口装配后启动，Host 暂停不停止已选来源；正常退出清理停止 Monitor。App OnQuit 仅取消，避免在 UI 线程等待通知回调；系统通知权限等待可取消，不阻塞正常退出。主程序最新构建与最终差异检查见本批收尾。

## 工具审批与执行回执

- 新 `internal/desktop/proactive/tools.go` 与嵌入的 `internal/harness/proactive-tools.mjs` 将 native task/request/session 绑定到实际 Host `tools/execute`。每次提案展示具体工具、完整参数及工作目录；批准只认该调用，发出执行前记录一次性 lease，随后保存 succeeded/failed/unknown 回执。拒绝作为工具结果返回模型；取消或重启使未执行提案失效，已发出但结果不明的操作不自动重放。
- 原生工具审批直接决定等待中的调用，不再取消其 Host 会话；原有非工具提案仍保留暂停执行的行为。L1 拒绝写入；L2–L5 允许工作目录内的 `write/edit`，其他未分类操作要求具体批准，原 Host 权限检查继续生效。
- `rc4-proactive-tool-gate-result.json`：真实 rc.4 Host + 本地 HTTP/SSE 模型，批准前无副作用、精确参数批准后只执行一次、下一次调用重新审批、拒绝无副作用、L2 实际文件写入/L1 无文件、Host 重启后旧批准不能恢复执行均通过（16.167 秒、9 次本地模型请求）。外发替身仅写隔离目录的本地记录，没有发送真实消息；所属 Host/模型/监听均已关闭。
- 定向 Go 检查及 Node 2 项检查通过：精确参数/大整数收件 ID、重开后单次执行、范围与取消、超过 64 KiB 的认证请求、尾随 JSON 拒绝、普通/分叉会话不误绑、子会话绑定继承。子代理继承目前只由替身检查覆盖，跨进程实际子代理仍未验。
- `ui/proactive-tool-approval-light.png`、`proactive-tool-approval-narrow.png` 和 `proactive-tool-effect-dark.png` 已实际查看，完整参数与未知执行回执可读；等待原因已补中文并查看更新后的窄窗图。主程序 `tmp/alignment-20261111/whaleisle.exe` 构建通过。
- **普通续聊兼容已修复：** 工具 overlay 依据当前 `turn/start` 后的 `user/message.source.rpcId` 判断是否为任务回合。任务中的插话保留门禁，后续普通用户回合恢复原 Host 权限，明确继续任务重新绑定新 request；任务子会话继续保留其原 owner。`rc4-proactive-turn-binding-result.json` 已验证实际 Go `SubmitComposer` 在完成的任务会话中写文件，随后任务 Continue 再次要求批准，拒绝无新增副作用；原批准/拒绝/L1-L2/重启检查同时通过（16.211 秒、13 次本地模型请求）。所有自有进程/监听关闭，Node 回合/插话/子会话检查 3 项通过。
- 工具动作目前仍按名称识别，第三方同名工具的来源识别需收尾，不能把原 Host 工具的通过外推到任意插件。真实跨进程子代理也未被本次普通续聊检查覆盖。
- 审批后主动结果投递在后述批次接入已有授权飞书会话；OS 通知显示/点击、真实模型与真实 IM 收发仍缺实际验收。完整主动系统尚未交付。

## 完整 rc.4 运行时装配

- 新 `scripts/session-runtime-complete.mjs` 将已有四条准备路径串到 `build-harness.ps1`：安装资源 Host → 手机 Electron ABI 依赖 → dshbot 0.2.0 保留 → main `c9d805a7de0e` 四插件。`package.ps1 -RuntimeRoot` 指定整套输入并检查手机、dshbot、四插件入口及同一来源 revision；保留原 rc.2 调用方式。

```powershell
./scripts/build-harness.ps1 `
  -RuntimeSourceResources 'C:\软件\Whale Isle\resources' `
  -RuntimeDestination '<新的运行时目录>' `
  -DesktopSourceRoot 'C:\AI\Deepseek-Harness-Desktop' `
  -LegacyVendor 'C:\AI\WhaleIsle-MyGo\runtime\vendor'
./scripts/package.ps1 -RuntimeRoot '<上述运行时目录>' -OutputRoot '<新的打包目录>' -StageOnly
```

- 第一次统一装配发现既有 `session-runtime-desktop-plugins.mjs` 将没有 `files` 字段的包当作空列表，导致 `dsh-platform-session` 仅复制 package.json。已修为无 allowlist 时复制包内容并检查入口，复制包及其相对 import 的实际 Node 检查通过。旧 `runtime-rc4` 也有此遗漏，后续不要继续使用该目录作为完整候选；此前核心/手机/任务的证据仍仅限其实际覆盖路径。
- 失败日志 `runtime-complete-build.log` 与失败目录 `runtime-complete` 保留。修复后的统一构建在新目录 `tmp/alignment-20261111/runtime-complete-fixed` 完成，日志 `runtime-complete-fixed-build.log`；手机 native probe 加载 SQLite/PTY 成功，Host rc.4、main 四插件与 dshbot 均完整。
- `rc4-complete-plugins-result.json`：从新统一产物启动真实 Host，平台插件认证路由正确返回隔离空账户，资料任务读取/报告/会话回传、停止请求、原快照继续、重启显式恢复与授权控制通过（52.835 秒、8 次本地模型请求）；自有 Host/模型/监听均退出。
- 正式 `package.ps1 -RuntimeRoot .../runtime-complete-fixed -StageOnly` 成功生成 `tmp/alignment-20261111/package-rc4-complete/whaleisle`（版本 `0.0.0-dev`），含主程序、辅助程序、原生终端、字体/桌宠资源、完整 Host/手机/插件、依赖恢复助手和安装文件清单。日志 `package-rc4-complete.log`；这是完整打包目录，尚未构建 Setup、签名、安装或发布。
- `rc4-packaged-core-result.json` 从包内 `whaleisle/runtime` 启动，平台认证路由、创建会话、实际文件读取、流式停止、停止后续聊、重启后持久历史通过（12.428 秒、6 次本地模型请求），自有 Host/模型/监听均退出。此次验证针对重新定位后的打包运行时，不冒充已安装程序验收。
- 同源码 `native-acceptance.exe` 使用包内 runtime 在隔离数据目录启动真实隐藏原生主窗口，Host/mux/workspace ready，settings/describe、pluginManager/listBundles、schedule/catalog 返回成功；`packaged-native/native.png` 已查看，窗口初始界面完整显示。收据 `native.png.json`，无模型请求，进程退出且 32481 无监听。该检查为冷启动空账户，不覆盖鼠标/IME、正常退出或正式安装版；测试收尾仍直接销毁自有窗口。

## 审批后飞书结果投递

- 新 `internal/desktop/proactive/delivery.go` 持久保存具体任务回合、Host 归属、原授权飞书会话、完整正文、审批与消息回执。原生“发送结果到飞书”选择目标并编辑正文，提交到 Outbox 后由用户批准；拒绝投递保留已完成的任务结果。HTTP Host 控制面不能创建或批准原生投递，只能认领一次已批准 ID 并报告结果。
- 新嵌入模块 `internal/harness/proactive-delivery.mjs` 复用 main `whaleTasks` 中 `dsh-im` 注册的生产 Feishu adapter。可选目标来自曾提交 `/task` 的已授权会话，沿用原 bot、actor、chatId 和 incarnation；不要求重新输入账号或另建 SDK。发送前重新核对渠道/owner epoch/Host 生命周期准入；接收方和正文来自 Go 保存的批准记录，调用方不能另传正文或替换目标。
- Runner 在批准后消费投递，发送前写一次性 lease；重复请求不能再发。明确未发送记录 failed；发送已发起但回执缺失、连接中断或重启时为 unknown，均不自动重试。迟到的同 lease 消息回执可结清 unknown。正在发送计入桌面退出/更新工作保护；维护冻结期间不开始新投递。任务继续不会覆盖尚未完成的投递。
- `rc4-proactive-feishu-delivery-result.json`：从 `runtime-complete-fixed` 启动真实 rc.4 Host、生产 WhaleTasks 和生产 `createFeishuTaskBinding`，末端 SDK 由本地文件记录器替代。验证审批前无发送、拒绝无发送、具体目标/正文一次送达记录器并持久回执、重复 ID 拒绝、预览后撤销 owner 阻止发送、缺失消息 ID 为 unknown、实际 Host 重启后 unknown 不重放（13.980 秒、1 次本地模型请求）。记录器只写隔离 `qa-feishu-effects.jsonl`，未向任何真实飞书会话外发；所有自有 Host/模型/监听关闭。
- Go 存储检查覆盖跨重开、错误 Host 拒绝、pending/ready/sending 的取消与继续边界、已确认任务的投递、Host 伪造提案拒绝、传输失败不重试。生产 proactive 包全套及原生面板定向检查通过；Node 工具门禁 3 项通过。正常用户退出/更新期间的实际投递竞争仍需在整体生命周期验收中覆盖。
- 原生 UI 实际 Tester 点击完成选会话→提交审批→批准→消息回执；`ui/proactive-delivery-light.png`、`proactive-delivery-narrow.png`、`proactive-delivery-receipt-dark.png` 已查看，目标和正文完整可读。`cmd/whaleisle/assistant_proactive_delivery.go` 对异步会话列表保留 Host 所有权，切 Host 后旧回执不可覆盖新列表。
- 最新主程序构建 `tmp/alignment-20261111/whaleisle.exe` 通过。上一节 `package-rc4-complete` 是此次投递修改之前的打包快照，未重新打包；后续 ACP 与策略收尾后再统一生成最终安装候选。真实飞书收发仍需用户指定目标并授权外发；当前仅完成生产适配器加隔离 SDK 的验证。

## ACP 原生接入与真实 Host 验证（11 月 11 日续）

- 新增 `internal/desktop/acp/`、`cmd/whaleisle/settings_acp.go` 和原生 `settings/acp*.go`：Agent 增删/重置、启停、独立账号、目录搜索、命令/参数/独立配置目录/环境变量、ask/自动允许读取/自动允许策略、模型配置/收藏/隐藏、会话断开。调用既有 ACP RPC，不改本体工作区。新建 Agent 使用显式唯一 ID，允许中文显示名，不能覆盖已有 ID；测试 ACP 使用当前草稿。配置更新保留未知字段、配置模型与发现模型的区别；后台刷新不覆盖未保存草稿。请求发出前复制完整参数，Host 切换/停止取消旧请求，旧回执不能覆盖新面板。
- 登录 UI 已接凭据登录/仅保存、浏览器同意、设备码、交互凭据、回调、取消、重新授权和退出（可明确选择删除 API 密钥）。这些生命周期控件已做原生 Tester 操作检查；真实第三方 OAuth/设备码与账号登录尚未验收。环境变量/凭据输入不回显。
- Composer 增加 Agent→模型入口，ACP 与 Harness 模型分开，保留原模型/推理强度回调；补齐方向键导航和当前模型已下架时的返回。现有 grouped model picker 定向检查及新增 ACP 键盘检查通过。
- `scripts/session-runtime-acp.mjs` 从明确选择的本体 ACP 包复制发布文件，记录 lexical checkout HEAD、实际源目录、workingTree 和文件清单；保留 `dsh-acp` junction 指向仓库外实际源的事实。`build-harness.ps1 -AcpSource <dsh-acp>` 接入此步骤，ACP 仍为可选组件，旧 rc.2 使用路径不要求安装它。完整运行时校验在存在 ACP 时验证其清单和兼容修正，缺 T3 bundle 的 Host 准备会失败。当前本体 HEAD 仍为 `18994faa036d8f30de4b148c96cb9a7523874dc0`，脏工作区不是发布版本。
- 真实集成暴露并在 MyGo 装配步骤修复三项本体包缺口：① `dsh:…` 内部实例 ID 不符合 T3 slug schema（同一探测仅改实例 ID 后成功）；改内部 ID 为 `dsh-…`，保留 `acp:` provider 路由。② prompt 完成时未等待 T3 事件队列，快速回复可能成为空白；完成前调用 `api.drainEvents()`。③ Host Stop 只中止本地队列等待，没有发送 ACP cancel；由包内 `mygo-turn-cancellation.js` 转发取消，下一轮等待旧 prompt 收尾，不再读取旧队列。这些修正由仓库脚本持续装配，不改 `C:/AI/Deepseek-Harness-Desktop` 或其 junction 的源文件。
- 有效端到端证据：`tmp/alignment-20261111/rc4-acp-cancel-fixed-result.json`，真实 rc.4 Host + 实际 staged T3 ACP 插件 + 本地独立 JSONL ACP 子进程，19.070 秒。覆盖握手、生产 Go Store 选择外部 provider/model/high 推理强度、连续两轮流式持久回复、权限拒绝无文件效果、允许一次恰有一次文件效果、改名称/收藏/隐藏不拆活动会话、实际取消后续聊、Host 重启恢复 DSH 历史并发送 v2 `session/resume`、断开/禁用保留会话历史。HTTP 模型请求为 0，所有内置外部 Agent 在 Host 启动前已禁用；未使用真实第三方 Agent/模型/账号。
- 之前失败的 `rc4-acp-first` / `v2` / `v2-negotiated` / `slug` / `drain` / `events-fixed` / `lifecycle` 记录保留。包含初版 v1/v2 夹具错误、上述三个产品缺口，以及一次把本地化推理强度显示名当作原始 ID 的观察器错误；最终检查改读生产 `ModelSelection.ReasoningEffort`，未削弱 high 传到 ACP 进程的断言。`rc4-acp-probe-diagnostic` 临时日志补丁已恢复，最终通过使用正式装配修正。
- 原生 UI 证据 `ui/acp-settings-light.png`、`ui/acp-settings-narrow.png`、`ui/acp-login-dark.png` 已查看，窄窗控件和表单完整可读。Go ACP client、设置/登录/新建 ID、Composer 键盘和主程序 Host 所有权检查通过；装配/cancel 的 Node 检查通过；主程序最新构建 `tmp/alignment-20261111/whaleisle.exe` 通过。最终 Host/模型监听关闭，另按命令路径检查本批及旧 ACP 夹具子进程，存活数为 0。
- 运行态验收对象仍是冻结的 `runtime-complete-fixed/vendor/dsh-acp`。完整装配入口另在新目录 `acp-stage-smoke/vendor/dsh-acp` 实际复制并校验 13 个文件及三项修正；此较新工作区快照没有替换已验收的冻结对象。当前安装版、默认 `runtime/` 和前一节 `package-rc4-complete` 均未更新，后者不能作为本批 ACP/飞书投递代码的交付包。本批未提交/推送/触发 CI。
- 上一阶段尚需：真实 Agent 的安装与 OAuth/模型/工具使用，MCP 桥真实工具消费，主动任务工具来源与 ACP 自动允许策略的统一约束，以及与整应用退出/升级的生命周期验收。其中 MCP 桥真实工具消费、工具来源识别、ACP 自动允许压制与正式安装包产物已在下一节收尾落地；真实第三方 Agent 账号仍单列待验。

## 工具来源识别、ACP 策略收紧与正式安装包构建（11 月 11 日收尾）

- **Cordis callback 来源识别与同名伪造拦截：** 新增嵌入模块 `internal/harness/proactive-provenance.mjs`（由 `launch.go` preload 先行加载），基于 `realpathSync` + `pathToFileURL` 预注册内置 `tool-fs`（`harness:fs`）、`tool-fs-search`（`harness:fs-search`）与 `dsh-acp`（`desktop:acp`）的 Cordis `apply` 回调，在 `ToolRuntime.prototype.register` 绑定实际定义来源；未登记插件一律归为 `plugin:unclassified`。`internal/desktop/proactive/tools.go` 收紧为仅当 `call.Source` 为 `harness:fs` / `harness:fs-search` 时才按内置文件工具放行，防止第三方或会话级 scoped 工具注册同名 `read/write` 绕过门禁。原生审批卡片 `internal/ui/assistant/proactive.go` 新增来源标签展示，截图 `tmp/alignment-20261111/ui/proactive-tool-approval-light.png`（800px）与 `proactive-tool-approval-narrow.png`（420px）已查看确认无裁切。
- **ACP 主动任务策略统一约束（第 4 项装配修正）：** 新增 `scripts/acp-task-policy.mjs` 并接入 `scripts/session-runtime-acp.mjs` 的 `t3-native-task-policy`（注入 `lib/t3-session.js` 与 `lib/index.js`，写出 `lib/mygo-task-policy.js`），不改动 `C:\AI\Deepseek-Harness-Desktop` 及其 junction 源文件。在主动任务绑定回合，强制压制 ACP Agent 的 `permissionPolicy: 'auto-allow'`、T3 `acpPermissionDisposition === 'allow'` 免审快径、未授权客户端直连写文件/创建终端及 elicitation 自动接受；仅凭 Go Outbox 单次批准产生的 `consumeAuthorization(exec)` 放行一次外部权限请求，拒绝或取消时经 `rejectTrackedAcpTool`（含 `entry.taskRejected` 竞态保护）即时回绝挂起请求；同会话后续普通用户回合自动恢复原 `auto-allow` 策略。
- **端到端真实 Host 证据 1（来源识别与同名伪造门禁）：** `tmp/alignment-20261111/rc4-proactive-provenance-gate-result.json`，从新完整运行时启动真实 rc.4 Host + 本地 HTTP/SSE 模型，通过 `@deepseek-ai/dsh-scope` 在会话层注册覆盖内置工具的同名 `read/write`（上报 `plugin:unclassified`），验证 L1 直接拒绝伪造 `read`、L2 将伪造 `write` 拦截进 Outbox 等待显式批准，连同精确参数单次执行、拒绝零副作用、L1/L2 内置写入边界、普通续聊与重启旧批准失效共 8 项检查全绿（82.953 秒，17 次本地模型请求，`ownedHostStopped: true`，`ownedListenerClosed: true`）。首次因 Windows PowerShell 5.1 下 `ProcessStartInfo.ArgumentList` 为 `$null` 留下空目录 `rc4-proactive-provenance-environment`（保留未删），已在 `Run-Isolated.ps1` 补回退分支修复。
- **端到端真实 Host 证据 2（ACP MCP stdio 桥 + 任务回合 `auto-allow` 压制）：** `tmp/alignment-20261111/rc4-acp-policy-mcp-result.json`，真实 rc.4 Host + staged T3 ACP + 本地外部 ACP 子进程，共 12 项检查全绿（24.496 秒，0 次 HTTP 模型请求）：外部子进程真实 `spawn` 包内 `t3-mcp-bridge.cjs` stdio 进程完成 `initialize`、`tools/list`、`whaleisle_skill_list` 及 `whaleisle_present` 交付物落盘；同一 ACP 会话配置 `permissionPolicy=auto-allow` 后，L1 任务回合自动拒绝外部写权限，L2 任务回合进入 Outbox（显示 `source=desktop:acp`），拒绝零副作用、批准一次写一次，随后同会话普通用户续聊自动恢复 `auto-allow` 直写。测试结束后核查监听关闭且 `acp-fixture.mjs` / `t3-mcp-bridge.cjs` 子进程存活数为 0。
- **单测与宿主环境隔离修复：** Node 单测 `proactive-tools.test.mjs` 与 `session-runtime-acp.test.mjs` 6/6 通过；Go `internal/desktop/proactive`、`internal/harness`、`internal/desktop/acp`、`internal/ui/...`、`scripts/nsis`、`scripts/session-runtime-links` 全部通过；`cmd/whaleisle/main_navigation_parity_test.go` 补充 `t.Setenv(paths.EnvUserData, t.TempDir())` 消除读取宿主机 `%APPDATA%` 鲸鱼娘会话的污染后，`go test ./cmd/whaleisle/... -count=1` 全量通过（40.585 秒）。
- **最新完整 rc.4 运行时与正式 NSIS 安装包产物：**
  - 新统一运行时 `tmp/alignment-20261111/runtime-complete-policy`（日志 `runtime-complete-policy-build.log`）通过 `session-runtime-complete.mjs verify`：含 Host `0.2.1-rc.4`、Electron 44.9.1 ABI 149 手机依赖、`dshbot` `0.2.0`、main `c9d805a7de0e` 四插件及含 4 项兼容修正的 `dsh-acp` `0.1.0`。旧 `runtime-complete-fixed` 与 `package-rc4-complete` 原样保留备查，不再作为最新交付输入。
  - `scripts/package.ps1 -RuntimeRoot .../runtime-complete-policy -OutputRoot .../package-rc4-policy`（日志 `package-rc4-policy.log`）完整生成阶段目录 `whaleisle/`（28,846 个物理文件、3,147 个 junction、`runtime-links.exe`、Ghostty 终端、9 项 THA4 资产、字体许可，`whaleisle.exe` SHA256 `89b128839624a0c58a736b3d71ca87cab822210be0abe8b4069de46dd1830cb9`）、`runtime-files.nsh`（7,695,290 字节）、正式安装包 `WhaleIsle-Setup-0.0.0-dev.exe`（316,372,500 字节，SHA256 `7ba53f8ca712c4c2daa58ac4ea83574c1d7f3299e81cbba63ec5952581962cf1`）、`.blockmap.json`（1,123,292 字节）与 `SHA256SUMS.txt`。
  - 重定位包内运行时与隐藏原生主窗口验收：`rc4-packaged-policy-core-result.json` 从 `package-rc4-policy/whaleisle/runtime` 启动真实 Host 通过全部 9 项核心链路与重启恢复检查（50.605 秒，6 次本地模型请求，进程/端口全清）；`packaged-native-policy-run/native.png` 与 `native.png.json` 在隐藏原生主窗口完成冷启动、Host/mux/workspace ready、`settings/describe`、`pluginManager/listBundles`（127,787 字节，含 `dsh-acp`）、`schedule/catalog` 读取及并发配置持久化校验（`preferencesConcurrentEdit: true`，32483 端口已释放；首次未配 `autoStartDesktop` 的目录 `packaged-native-policy` 保留）。

## MyGo 品牌命名统一与独立共存安装身份（11 月 11 日补充）

- **中英文产品名称统一追加 `MyGo`：**
  - [product.go](file:///c:/AI/WhaleIsle-MyGo/internal/product/product.go) 与 [mygo.json](file:///c:/AI/WhaleIsle-MyGo/mygo.json)：统一设置中文名 `Name = "鲸屿 MyGo"`、英文名 `EnglishName = "Whale Isle MyGo"`，保留 `LegacyNativeName = "MyGo鲸屿"` 以兼容既有原生窗口/WebView 缓存目录不迁移。
  - [sidebar.go](file:///c:/AI/WhaleIsle-MyGo/internal/ui/sidebar/sidebar.go) 与 [sidebar_expanded_test.go](file:///c:/AI/WhaleIsle-MyGo/internal/ui/frame/sidebar_expanded_test.go)：侧栏左上角品牌区改为 `鲸屿 MyGo` + `WHALE ISLE MYGO`（副标题保留 `BASED ON DEEPSEEK HARNESS`），微调字号与间距并新增 `TestSidebarBrandFitsDefaultAndMinWidths` 覆盖默认 `280 DIP` 与最小 `264 DIP` 宽度，确认无省略号截断、不与右侧折叠按钮碰撞（截图：`tmp/alignment-20261111/brand-preview/sidebar-brand-280.png`、`sidebar-brand-264.png`）。
  - 全局多语言与窗口文案同步：涵盖 [i18n.go](file:///c:/AI/WhaleIsle-MyGo/internal/ui/i18n/i18n.go)（中英文 `KeyAppName`、欢迎页、关于页、启动器标题等）、引导页 [onboarding.go](file:///c:/AI/WhaleIsle-MyGo/internal/ui/settings/onboarding.go)、启动器窗口 `鲸屿 MyGo Launcher`、恢复窗口 `Whale Isle MyGo · Recovery`、桌宠 `Whale Isle MyGo companion`、插件安全提示、文档预览提示及 Project 数据目录提示 [hero.go](file:///c:/AI/WhaleIsle-MyGo/internal/ui/projects/hero.go)。
- **与 Electron 本体共存（安装独立、数据共享）：**
  - **独立安装身份与目录：** `product.AppID = "ai.deepseek.harness.gui.mygo"`、`product.InstallDir = "WhaleIsle-MyGo"`。[whaleisle.nsi](file:///c:/AI/WhaleIsle-MyGo/scripts/nsis/whaleisle.nsi) 默认安装至 `%LOCALAPPDATA%\Programs\WhaleIsle-MyGo`，桌面与开始菜单快捷方式为 `鲸屿 MyGo.lnk`，安装包输出名为 `WhaleIsle-MyGo-Setup-${VERSION}.exe`；macOS [package-macos.mjs](file:///c:/AI/WhaleIsle-MyGo/scripts/package-macos.mjs) 同步为 `Whale Isle MyGo.app` 与 `ai.deepseek.harness.gui.mygo`；卸载注册表、系统通知 `AppUserModelID` 与 WebView YouTube Origin 同步对齐 `product.AppID`。
  - **保留共享用户数据目录：** `config.DataDirName = "Deepseek-Harness-Desktop"` 保持不变，直接共享现有配置与会话历史；双端同时启动时通过 `claimDesktopProfile` 回退 `native` 子目录锁并监听独立端口 `3180`。
  - **打包冒烟去重提速：** [smoke-nsis.ps1](file:///c:/AI/WhaleIsle-MyGo/scripts/smoke-nsis.ps1) 调用 `package.ps1` 时改为 `-StageOnly`，消除单次冒烟重复执行两遍 LZMA 压缩的耗时问题。
- **验证证据：**
  - Go 单测与契约检查：`internal/ui/...`（含 `layoutstability` Toast 修正）、`cmd/whaleisle`、`internal/desktop/importer`（`TestCoverUpgradePathInvariants`）、`appuninstall`、`externallauncher`、`notifications`、`pet`、`update`、`webview`、`harness`、`scripts/nsis` 及 Node `macos-preview-bundle.test.mjs`、`package-release-inputs-fixture.mjs` 全部通过。
  - 隐藏原生主窗口真实验收：`tmp/alignment-20261111/packaged-native-mygo-run2/native.png` 与 `native.png.json`（真实 rc.4 Host/mux/workspace ready，`preferencesConcurrentEdit: true`，`2160x1295` 截图中左上角 `鲸屿 MyGo` + `WHALE ISLE MYGO` 清晰无截断）。
  - 隔离静默安装/覆盖升级/卸载冒烟：`tmp/alignment-20261111/nsis-smoke-mygo.log`（`WhaleIsle-MyGo-Smoke-8177194c93b9480eab9b0bc2cf06bf1a.exe` 静默安装、覆盖升级、静默卸载及注册表/目录零残留通过）。

## 当前接续点

- 最新开发可执行文件 `tmp/alignment-20261111/whaleisle.exe`（SHA256 `1a5b4b0262ab21392fe23f7e7866d5ba9c416d008a7224c42a8ad8fd5358f24b`）及快速验证包 `tmp/alignment-20261111/nsis-smoke-mygo/` 已构建验证完毕；避免长时间后台复制/LZMA 压缩阻塞，正式全量离线安装包可在发布前按需在终端单次执行打包。
- 未提交、未推送、未发布、未触发 CI，默认 `runtime/` 仍保留 rc.2 未静默切换。
- 代码与本地隔离端到端验证已全部收口；剩余需用户授权或真机参与的实测边界包括：① 真实第三方 ACP Agent CLI 安装、OAuth 登录与计费模型调用；② 真实外部模型 API 对话验收；③ 真实飞书账号与指定会话外发验收；④ 实体手机与公网中继连通验收；⑤ 操作系统原生通知中心真实弹窗/点击、真实窗口键盘/IME 输入与正常退出竞争；⑥ 正式代码签名、远端更新源发布及确认本机安装。Project 继续按用户要求暂缓重设计。
