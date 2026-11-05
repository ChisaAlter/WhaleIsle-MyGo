# 原版功能补齐执行记录

## 目标与范围

用户于 2026-11-02 授权“全部对照鲸屿补齐这些功能”。实现依据为已安装 Whale Isle 0.3.13、Host/client 0.2.1-rc.2，以及本目录稳定 367 项和 758 个细分动作；二者有重叠，不相加计功能。原项目 dirty 未发布增量单列，不擅自作为已发布合同。当前工作树已有轨迹诊断改动保留。

沿用纯 Go/MyGo 原生交互与原 Host 协议；不以默认打开整套 Host 网页替代原生功能。现有受限文档/媒体/动态内容容器按原有用途复用。原项目、原安装和其中 vendor 只读；本项目 runtime 仅同步明确授权的原安装已发布文件并保留备份。用户真实配置/数据不改，不自动重放消息。

## 当前阶段与历史执行记录

**2026-11-03 已撤回整体“实现完成、进入验收”的判断。** 用户截图证明主界面仍有状态流、入口映射与功能目录缺失；`.8` 不是实现就绪版本。下列勾选仅保留各次局部实现及检查的历史事实，不代表该域完整界面或最终产品完成。当前先重核并修复原版实际界面合同，用户已明确停止测试、模型请求、桌面验收和新候选构建。

当前各域源码纠偏与未覆盖面统一见 [界面实现纠偏](UI-REPAIR-20261103.md)。已接入顶部/权限/团队/资源、模型与插件、文件工作面，以及随后找到的回合导航、问题失败恢复、计划审批和自动化入口修复；这些源码没有进入 `.8`，也没有恢复验证。

用户随后要求继续清查。只读第二轮已整合 [54 项当前未对齐项](UI-AUDIT-R2-20261103.md)及两端源码证据。用户现已授权“全部推进开发”：54 项全部进入实现，包含 P2；同时完成此前已定位的自动化日期/时钟交互。测试、测试准备、桌面操作、模型请求、构建与发布仍停止，不生成新候选。

### 第二轮开发责任

**第二轮 54 项（P1 25、P2 29）的产品源码和消费者接线已落盘。** 此前已定位的自动化日期/时钟交互，以及开发中定位到的同名外部 feature writer 取消/失败回滚也已补入。当前为源码实现结果，未编译或运行；不代表全部鲸屿功能已对齐或可以恢复验收。

各域只写自己产品文件，跨域不并发改同一文件。root 独占 `main.go` 和本记录，负责主装配；各域报告逐项保留改动、消费入口和运行未知。

| Owner | 范围 | 交付记录 |
| --- | --- | --- |
| audit_conversation_remaining_r2 | CONV-R2-001–008；消息、轨迹、审批、附件及对话 slots | [对话实现](../../../tmp/parity-implementation-20261103/implementation-r2-conversation.md) |
| audit_account_usage_r2 | R2-AU-001–014；账号、用量、敏感文本 | [账号与用量实现](../../../tmp/parity-implementation-20261103/implementation-r2-account-usage.md) |
| audit_settings_views_r2 | SETV-R2-001–006；一般设置与接受/取消语义 | [设置实现](../../../tmp/parity-implementation-20261103/implementation-r2-settings.md) |
| audit_authoring_tools_r2 | R2-AUTH-001–003；Skills 与预设 | [作者工具实现](../../../tmp/parity-implementation-20261103/implementation-r2-authoring.md) |
| audit_dynamic_pad_r2 | DYN-R2-001–006、R2-WS-001/002；Pad、公开桥、Models 扩展位 | [动态扩展与 Pad 实现](../../../tmp/parity-implementation-20261103/implementation-r2-dynamic-pad.md) |
| audit_remote_lifecycle_r2 | R2-LIFE-001–008；快捷键、导入、更新及远程 | [生命周期实现](../../../tmp/parity-implementation-20261103/implementation-r2-lifecycle.md) |
| audit_pet_assistant_r2 | R2-PETA-001–005；桌宠、Whale 与语音 | [桌宠与助理实现](../../../tmp/parity-implementation-20261103/implementation-r2-pet.md) |
| repair_frame_entry_parity | R2-WS-003/004 与上述 frame/sidebar 消费者 | [框架消费者实现](../../../tmp/parity-implementation-20261103/implementation-r2-frame.md) |
| repair_automation_consumers | 已定位的自动化日期、时钟、焦点与反馈 | [自动化实现](../../../tmp/parity-implementation-20261103/implementation-r2-automation.md) |

以上域报告位于 `tmp/parity-implementation-20261103/`。`pluginui` 由 dynamic owner 单独写，`frame/sidebar` 由 frame owner 单独写；Models、账号、Pet 和 Skills 设置分别由各域 owner 写，一般设置不覆盖它们。聊天主消息由 conversation owner 写，秘密识别/状态 helper 由 account owner 写并通过接口交接。

### 主装配已消费，不以接口声明代替可见入口

- `main.go` 的 Skills 组手势捕获确切可见成员和 catalog，编辑分组使用无损数组；账户授权/退出由根级常驻 owner 负责，登录链接复制及错误码进入账户页。
- Whale、Features、Remote 实际开关复用保护事务。外部多字段 desktop patch 保留原协议，消费异步实际重启结果；取消、拒绝或失败仅回滚本次仍归属的 feature 字段，其它设置和后来的写入不覆盖。
- 完整会话框架与外层全局页共同绑定 shell/workspace registry；Models 两个扩展位、主/子可见审批、picker 选择后 adopt 均已接到真实消费者。匿名计划使用源当前 active key、完整问题 schema 和原生已选请求共同定界，不把 transport ID 当请求身份。
- 主窗绑定后启动唯一更新调度；异步导入、窗口关闭和最终退出有实际取消入口。自动下载使用共享 typed 偏好和原版缺省值。
- 页面关闭先交当前 DialogBase 的原 close owner，再处理真实焦点资源或窗口；非 frame 路由也消费同一按键身份。刷新只执行资源已有能力，字号键持久修改字体偏好。
- 账号奖励只在真实 native surface 已呈现可见 toast 后通知账户 owner；`Toast.OnShown` 不把入队、capture、隐藏或最小化当成呈现。
- 自动化主页面使用同一已取得的 session snapshot 提供原会话状态；右栏以真实 viewer 身份消费，日期/时钟、焦点和失败反馈由实际任务编辑器接入。

本轮各 owner 及 root 只进行产品源码编辑、`gofmt`、直接 diff 和消费者阅读；未新增测试或 QA 夹具，未编译、运行、桌面操作、模型请求、CI、打包或发布。当前 dirty 源码尚未生成新 EXE，`.8` 不包含这些修复。

### 继续条件与尚未证明的范围

测试暂停仍有效，不能据此生成新候选或把 367 项/758 个重叠动作批量改绿。原始“轨迹点击后下划线不动”的窗口根因仍未知；独立轨迹账本修复不替代这一定位。编译/类型链接、普通与窄窗口、命中与 IME、真实 Host/账号/模型/设备、升级及安装恢复均没有本轮运行结论。

第二轮未穷尽的 IM 平台/媒体与未知投递、AI Office 专属业务、复杂桌宠物理/照料/动画、动态第三方行为组合，继续保留原清查的未知范围。可选 models.dev 元数据补全未纳入本轮公开 slot 修复。完整对齐目标未签署完成；恢复被暂停的验证需用户另行授权。

### 旧局部实现及运行事实

- [x] 数据安全和日常主链：取消删除、身份/草稿/工作区、新会话、登录/credential、顶部两层与可达导航。
- [x] 聊天、轨迹、工具/审批/问题、子会话与队列。
- [x] 设置、模型、费用、用户文本掩码。
- [x] 插件/市场/技能/MCP、动态贡献与 Pad，含公开 composer.bar 附件接口（保留文末明确兼容边界）。
- [x] 文件/交付物/Git/终端/浏览器。
- [x] Project/Team/自动化/Whale/IM。
- [x] 壁纸/桌宠/语音/图像及生命周期/导入/更新/远程。
- [x] 整合源码定向检查、已列生产链本机运行与离屏/软件呈现检查，含最后公开附件接口的差异检查。
- [x] 2026-11-03 生成独立完整验收候选，不运行安装器、不替换原安装或旧预览。
- [x] 新候选runtime与同版生产Go主链：真实文件工具回传、停止、下一轮、Host重启后的完整历史恢复均通过（本机模拟模型）。
- [x] 原 Pad 模型工具静态卡 CRUD 真实运行，工具结果回传 provider，并同步到生产 Chat/Pad Store 与 SSE。
- [x] 修复真实 Host 拒收问题/审批回执及 legacy 问题卡不关闭；原问答/审批链、Pad 数据源/刷新/undo/重启恢复均已通过真实运行。
- [x] RPC、问题关闭权威与 control reader 三项修复先纳入 `.3`；Host 终端、五组 API 合同及 Provider UX 纳入 `.4`，Windows worktree/旧回执恢复及手机端口预留纳入 `.5`；连同下列子会话修复均保留于当前 `.8`，实际隔离入口与 Host 握手通过。
- [x] 同候选 Pad 后台 agent 的成功/真实失败/下一轮恢复与公开 Run API，以及 Electron/Go Windows 安全存储双向互操作。
- [x] 修复首次自动选会话/新建/分叉后的 control reader 漏启；已发布 opt-in timed 问题的真实超时、继续、迟到回答、无限等待与旧 delivery 隔离通过。
- [x] Pad 原 15 分钟自动卡实际触发；运行中停止取消真实模型流，新 Host generation 恢复且不采用旧提交。
- [x] 本地 MCP 生产表单/Controller → 真实 SDK stdio 工具 → provider/持久 Chat Store；停用后下一轮工具撤销和删除通过。
- [x] Pad 固定 command 源手动拒绝不执行、原 fingerprint 批准后真实读取、刷新及删除停止通过。
- [x] Project/原生 Team 真实委派和自动交付、绝对时间计划自动执行及恢复/删除；Host 终端 attach/PTY/保留重连；scoped 文件 stat；受影响结果合同与正常/窄原生软件布局。
- [x] Project development 独立 worktree 的写入/读回、停止 handback 和明确续接、历史终态归纳交付；Windows 长路径检出与旧 preparing 回执修复。
- [x] 同一 60 秒循环计划连续自动执行两次；第三次到期前删除，在原服务仍在线时越过原触发窗口确认不再运行。
- [x] managed Host 为手机固定 SPA 端口预留；普通 FreePort 合同保持，真实 Host 与自有 companion 同时监听通过。
- [x] 原生 Provider Controller 对真实 Host 完成目录读取、合成凭据保存、同 profile 重启恢复与清除；Project 多需求排序、修订隔离、取消及交付后的自动推进通过。
- [x] 子会话实际链发现的运行态、control inbox、idle disposal 与停止 mode 四处产品断点已修，真实队列/编辑/移除/steer/取消/同 child 续聊/历史重开和父子隔离通过；统一纳入 `.6`，历史 `.5` EXE 不含这些修复。
- [x] 两个本地群聊成员的真实 runtime、独立 root 身份、明确打开工具历史和同线程队列回复通过；无产品修改。
- [x] `.6` 实际普通文件/PNG 准备、原收据与上传进度、模型 Files/Messages、真实 read 工具、持久引用、正常重启后原字节读取及显式重新上传通过。
- [x] 实际 fork 的累计计费与全局继承前缀去重、合成价格保存/同 profile 重启恢复、JSON/CSV 和 UTC 日期范围导出通过；恢复未重发模型请求。
- [x] 原 Host 实际 v4 压缩会话扫描、保留单选、生产导入事务、目标 Host 恢复与同 ID 续聊通过；未选不导入、源保真及既有目标历史保留。
- [x] Files 实际 scoped Unicode 分页、根目录 fallback、watch 及自动刷新关闭/手动刷新/重新开启通过；空 root path 产品故障已修并纳入 `.7`。
- [x] 预设保存后普通新建实际采用 persona/tools，已有会话保持原身份，同 profile 重启恢复及再新建通过；无产品增量。
- [x] 原命令目录/准确错误、真实 `/compact` 摘要与替换检查点、明确下一条及正常重启恢复后续聊通过；无产品增量。
- [x] 两目录工作区与三个明确归属 Session 的实际注册纠错、重命名、顺序、pin、archive/undo、仅归档对象删除、注册移除保留文件/会话及 config/Host 重启恢复通过；发布 replace 完整帧漏消费已修。
- [x] Skills 项目/用户根真实编写、完整编辑、调用标志、原 skill 工具与用户显式调用、撤销、取消/确认删除及同 profile 恢复通过；调用标签/删除后缓存/cold promote 目录恢复三处产品断点已修。
- [x] Goal 原创建/暂停/编辑/恢复/完成/清除和重启 disarm，以及 Plan 原 mode 持久恢复、默认 legacy review 专用批准→实际退出→明确下一轮通过；无产品改动。
- [x] 本轮 Skills 三处断点及 workspace replace 完整帧修复统一纳入 `.8`，完整载荷与私有入口实际 EXE/Host 握手通过。
- [ ] 同一新候选的实际 Windows 呈现、物理交互与外部依赖验收。
- [ ] 逐条验收后生成最终交付物；必要检查未通过不触发 CI、不推送、不打最终安装包。

## 唯一写入与集成责任

root 负责主框架/顶部和共享接口、整合判断与本记录；`integrate_main_parity` 已交付主壳装配，后续 `main.go` 消费由 root 单独负责。各域实现者独占下表的业务文件；跨域消费者通过明确接口交主装配，不并发编辑 main.go。目录边界内还需服从发送给实现者的精确排除项。

| 域 | 实现责任 |
| --- | --- |
| 工作区与侧栏 | Session/Workspace/目录选择状态与侧栏用户动作；root 装配 |
| 聊天 | chat/composer/agents 及对应 child/trajectory Store |
| 设置/账号 | 设置表单/模型/凭据及账号登录、用量 |
| 插件 | 管理/市场/技能/MCP及动态表面；不接管 Pad |
| Pad | 新原生 Pad 客户端/Store/面板与默认装配方案；root 处理 main/共享 overlay |
| 工作面 | files/diff/terminal/browser 及其 desktop 实现 |
| 编排 | Project/Team/schedule/jobs/Whale/IM |
| 桌宠 | pet/voice/wallpaper/theme effects，以及对应设置文件 |
| 生命周期 | launcher/import/update/remote/tray/task protection |

先完成相应产品实现，再新增/修改定向测试。既有有效证据不为安心重跑。验收记录必须分开代码实现、组件/Host运行、可见原生界面和真实外部依赖；不能将 wired 或编译成功转成可用。用户虽已允许仅操作新隔离候选，但随后明确要求先停止测试，当前仅进行源码对照和实现修复。

## 当前状态

当前保留的最近候选是 **`0.2.1-rc.4+parity.20261103.8`**，不是就绪交付物。它保留此前各域整合、Files scoped 根目录、Windows worktree/旧回执恢复、手机端口预留与子会话四处修复，并纳入当时 Skills 调用/删除缓存、cold promote 目录恢复及 workspace replace 完整帧修复。历史私有入口曾启动该 EXE，认证 Host ready **5,589.946ms**，随后存活 10 秒，测试进程/监听均回收；私有 Git 根正确、application.log 0B。当前界面纠偏源码尚未进入该程序，旧启动收据不作为就绪结论。

**全量实现与验收均未完成。** `.8` 的局部 Host/启动收据保留，但用户截图已否定整体就绪：配置命令被误当首轮对话、团队按钮错开右栏、会话更多被插件菜单替代、右栏注册目录与资源挂载不全、内部状态文案外露。当前源码正在纠偏，尚未构建进新 EXE；367 项与 758 个动作分母不变，不批量改绿。

**磁盘阻点已解除：** 用户批准后，08:15 仅删除清单内 `.1–.6/whaleisle` 六个旧程序载荷，执行前复核精确绝对路径及活动进程引用；不遍历 reparse 目标。`.7/.8`、全部既有 profile、源码/报告/补丁和旧预览均保留。清理前容量已在本轮之外回升至21,325,705,216B，完成时31,814,238,208B（约29.6GiB），不把全部容量变化归于本次删除。见[清理收据](../../../tmp/parity-implementation-20261103/candidate8/old-payload-cleanup-result.json)。

**本轮阶段调整：** 已确定真实模型为官方 `deepseek-flash`、总费用人民币1元，CNB 为只创建公开空仓 `ayasealter/WhaleIsleMyGo`；没有执行付费请求或建仓写请求。用户要求先停测试后，私密卡等待和模型准备已停止；这些授权不被当作当前执行任务。Computer Use 刷新会话后仍报 native pipe 不存在，当前也不再重试桌面连接。先完成界面实现纠偏，不要求用户继续代为验收。

**本轮本机主链：** Project development 的实际独立 worktree、写入、停止/handback、明确续接和两个终态的完整归纳交付通过；同一 60 秒循环计划已自动执行两次，删除后原第三次窗口没有再触发。手机进程链查出默认 Host 与固定 SPA 同用 3180 的实际规划错误，源码已修并跑通真实 Host 端口预留，但旧预览仍占该固定端口，完整手机链未执行。具体版本、原始失败与范围见下方。

**新增实际证据与当前修复：** `.5` runtime 上 Provider 的真实配置合同、Project 多需求队列及本地群聊/成员 runtime 已通过，未改这三域产品。子会话真实链另修四处产品问题：历史 turn/end 覆盖新 start、未消费 control inbox、正常 idle disposal 撤读、停止错误发送 cancel 而非 address mode。修复后当前生产 Store 的完整子会话链已通过，冻结源码已纳入 `.6`；历史 `.5` EXE 不含这些修复，原失败保留。

**`.6` 后续真实链补证：** 普通文件和 PNG 已经原 Host/模型协议/持久历史往返，并完成正常 Host 重启后的读取和显式重新上传；真实 fork 的用量去重、合成计费设置恢复与导出通过；原 Host 生成的真实 v4 压缩会话经生产 importer 单选导入后，目标 Host 已恢复原工具结果并在同 ID 完成下一轮。三域无产品修改，无需重建或重复启动入口检查；这些实际链不代替 EXE 窗口或真实旧用户 profile 接管。

**本轮新缺陷与三条实际链：** Files 在没有本地挂载时把 scoped 根目录空 path 发给 Host，被 `path is required` 拒绝；已仅为 scoped root 改发 `.`，旧无 scope 合同保持。修复后真实目录/watch/自动刷新切换通过，`.6` EXE 仍含原缺陷，修复已纳入统一 `.7`，实际私有入口通过。预设保存/新会话实际组成/重启恢复，以及原 `/compact` 摘要/检查点/明确续聊/同 profile 恢复均通过，后两域未改产品。具体版本和首失败见下方分项。

**本轮 Skills/工作区与 Goal/Plan：** Skills 标签/删除缓存和冷恢复三处真实断点已修，项目/用户技能作者、原工具调用、用户显式调用、撤销/删除、同 profile 自动恢复均通过；工作区/会话管理、移除注册保留目录/会话与配置恢复通过，另补发布 replace 帧消费。四个产品文件的冻结增量已纳入 `.8`。Goal 生命周期与重启 disarm、Plan 持久 mode 和默认 legacy review 明确批准后实际退出/下一轮也通过，没有为这些有效证据重复运行或单独构建。具体 scope/原失败见下方。

**Host 终端协议与布局修复已纳入 `.4`：** 旧 `.3` 使用已撤下的 follow/write/resize，首轮 Render 为 0，见[原始失败](../../../tmp/parity-implementation-20261103/host-terminal-flow/attempt1.json)。现已迁移原发布 attach 双向流/provider ack，生产 HostView 实际消费 19 个帧，输入/resize/同 ID 同进程重连/关闭回收通过；见[修复与真实结果](../../../tmp/parity-implementation-20261103/host-terminal-flow/RESULT.md)。该次 800×420 surface 只剩 1 行终端，进一步定位并修复管理控件挤占正文：同尺寸现在为 784×354/19 行，360×420 窄 pane 仍 41×19；管理 popup 不改变正文 Bounds，root 已实际查看正常/窄与管理软件图。见[几何、渲染与完整入口](../../../tmp/parity-implementation-20261103/host-terminal-layout/RESULT.md)。真实链采用当前修复源码与未改变的 `.3` runtime/DLL；没有把软件图冒充 Windows 可见窗口验收。

**API 合同核对发现的 5 组问题已修入 `.4`：** 群成员 session/list 必填键修为 `_request`；文件 stat 加 frame-bound Session scope；奖励 arbitrary metadata/服务端标题保真；migration-required hint 不取得 live queue/question/model/permissions 权威；模型容量缺省/显式值不再丢失。严格 envelope、迟到回执、真实 Host stat 与受影响类型/Store/原生组件定向均通过；主包标题消费与 projection 三分支 roundtrip 亦通过。群成员的真实公开合同后续另已补证，见下方本地群聊结果；EXE 窗口跳转仍未验。API 生成器另已兼容实际发布的 lazy 元数据，31 包/163 端点正常生成；没有盲目覆盖全部类型。见[消费者审查](../../../tmp/parity-implementation-20261103/api-contract-drift/consumer-audit.md)、[三项结果合同修复](../../../tmp/parity-implementation-20261103/api-contract-drift/result-contract-fix/RESULT.md)与[生成器结果](../../../tmp/parity-implementation-20261103/api-contract-drift/GENERATOR.md)。

**原生模型页的 Provider 能力消费已修：** 默认端点、Host 端点环境来源、原生鉴权引用顺序与已检测/缺失/未知状态均保留；显式引用优先，OAuth-only 且无显式引用时不再提供或接受 key 写入。普通 key、自定义 route 和空 `{}` 目录采用保持可用，没有另造登录系统。生产 Controller/codec 与原生 headless 控件的 11 个定向目标已有有效通过证据；root 已查看 1000/400 宽软件图。原始 Models source 贡献入口不变，真实账号和可见候选仍未验。见[实现、合同与两次失败的处置](../../../tmp/parity-implementation-20261103/provider-capability-ux/RESULT.md)。

**Provider 真实配置合同补证通过：** 当前生产 Controller + `.5` Host 实际读取 43 个 configurable providers、OpenAI 端点/native auth 和 Codex OAuth-only metadata；在全新私有 profile 用 revision-guarded settings/mutate 绑定引用，credentials/set 保存固定合成值，正常停启同一 profile 后 presence/引用仍在，再以 credentials/unset 清除并确认草稿/确认态归零。OAuth-only 拒写亦经真实目录验证。首次运行 20,349ms、cleanup 全 true/tripwire 0，没有 Session、模型请求或真实账号；这补上配置消费者合同，不代替真实 provider 鉴权或可见窗口。见[实际配置、恢复与清除](../../../tmp/parity-implementation-20261103/provider-config-live/RESULT.md)。

**`.2` 控制流缺陷已修入 `.3`：** 连接时未选会话导致首次 `WatchControl` 返回，后续自动选择/新建/分叉只启动历史流，同 ID 选择又直接返回；真实 Host 已为 continued/asOfSeq 25，Go Store 却停在 seq 2。这是产品断链，现由 Session Store 共享锁内准入确保 reader，复用原取消、client/mux、Host generation 与父 context；同 ID 幂等调用保留 controlNote 和 follow 代际。实际 timed 链没有在 QA 额外补 watch，2 次工具调用均回传并持久入生产 Chat Store。见[修复、实际结果与首次失败](../../../tmp/parity-implementation-20261103/timed-question-flow/RESULT.md)。

11 月 2 日补齐阶段未生成主程序；11 月 3 日已另建下述新候选。原有轨迹诊断预览和旧安装器仍不包含本轮新源码。

### 2026-11-03 当前候选与有效证据

- **完整候选 `.8`：** [候选和隔离启动说明](../../../dist/0.2.1-rc.4-parity-20261103.8/README.md)。Go 1.27.2，源码 `b8b93ed8` 加当前整合 dirty；EXE 为 103,964,672 字节。完整 rc.2 Host、25,722 个 runtime 物理文件、3,146 个内部链接、原生 terminal/THA4 和 preview/integration helper 均随包；完整管线再次验证闭包和 SQLite/PTY ABI。未生成安装器，未触发 CI。见[构建与集成收据](../../../tmp/parity-implementation-20261103/candidate8/result.json)、[构建日志](../../../tmp/parity-implementation-20261103/candidate8/build.log)。
- **当前实际隔离入口/EXE PASS-SCOPED：** 执行交付 `Start-Isolated.ps1 -Hidden`，全新私有 profile/工作目录，无预注册工作区或复制账号。原生窗口创建、认证 Host ready **5,589.946ms**，随后 10 秒存活；父 PowerShell HOME/APPDATA 未变。私有 Git 根正确，application.log 0B；仅回收自有测试进程树及 listener。隐藏启动不证明可见布局、点击或正常 UI 退出。见[当前结果](../../../tmp/parity-implementation-20261103/candidate8/startup-result.json)。
- **入口修正：** 敏感变量与 Git 上下文只从子进程清除，不改调用者环境；私有 Git 根已实际核对。首次完整入口因本机存在两个 `git.exe`，把 `Get-Command` 多个结果拼成无效路径，尚未启动 EXE 或写配置；现按 PATH 首个匹配选择，首失败和空私有目录保留在 `candidate2/entry-first-*` 与候选旁的 `acceptance-data-entry-rejected`。原 helper 的合成 secret/父环境检查见[环境结果](../../../tmp/parity-implementation-20261103/isolated-entry-environment-result.json)。

下列 Host 链通过生产 Go Store 和真实随包 Host，模型为本机协议模拟器；不经过 EXE 窗口发送。`.8` 复用此前相同 runtime 输入与终端 pin，构建管线重新验证完整闭包与实际原生 ABI；各链的具体源码/runtime 版本见原报告，没有为换版本重跑未改变的有效行为检查。

- **文件工具/停止/恢复 PASS：** 原 8 步在 23,047ms 完成：首轮回复、真实 read 结果回到 provider 与持久 Chat Store、31ms 停止、下一轮，以及 Host 重启后 4 个用户输入/全部回复/工具结果恢复。最初超时来自 QA 目录误继承父 Git 仓库，修正独立 Git 根与真实 turn 结束观察后通过；30 秒预算和原断言未放宽。见[结果与根因](../../../tmp/parity-implementation-20261103/tool-roundtrip-fix/result.md)、[报告](../../../tmp/parity-implementation-20261103/tool-roundtrip-fix/report.json)。
- **附件真实 Host/模型/恢复 PASS：** 当前生产 ComposerTarget + `.6` Host 上传原名普通文件 8,229B，真实 receipt 与请求体进度 0→11203/11203（非原文件大小）；64×48/129B PNG 沿原内联准入、实际 Files→Messages 到模型。普通文件保持原名/大小/只读句柄，原 read 实际取回 98 行、结果到模型；ChatStore 仅一个对应 user event、持久 file/image 引用完整。PNG 经生产 FetchAttachment 原字节读回；普通文件按原合同由只读存储副本和工具读取，不能用仅支持 image 的 session/attachment 假报不兼容。正常停启同 profile 后忘记缓存再读仍原字节，旧 receipt/错 Session/旧 owner 均拒绝，显式重新 prepare 获新 receipt 后下一轮通过。20.967s、4 次本机模型请求，无产品修改；标题请求误分类首失败保留，owned 服务全清理、tripwire 0。不覆盖 EXE picker/拖放或真实 provider。见[实际附件合同与原始收据](../../../tmp/parity-implementation-20261103/attachment-real-flow/RESULT.md)。
- **用量/fork 去重/计费导出 PASS：** 当前 Session/usage Store + `.6` Host 实际提交一次、ForkSession、明确新输入，保存合成价格。fork 累计 billedUsage 为 miss=330（含 cacheWrite）、hit=90、output=60；全局两个会话实际去重后 input=300/output=60/cacheRead=90/cacheWrite=30，原面板 total=input+output=360。按自有合成每百万 token 价格算 459 分，非真实账单。同一 owned profile 正常重启后投影/价格/去重均恢复；生产 json/models/days/sessions 导出及 UTC 日期范围通过，models CSV 为准确 route、459/CNY/1 priced/0 unpriced，恢复模型请求 0。生成段 11.393s、续段 9.974s，无产品修改。两次 QA 失败分别是标题分类和误将 cache-inclusive 480 当面板 total；已按原发布 totalsFrom 校正观察，未重跑有效消息/fork/保存。所有 owned 服务已终态、tripwire 0；可见页面/选择器与真实账单仍未验。见[真实聚合、原合同与恢复续段](../../../tmp/parity-implementation-20261103/usage-real-flow/RESULT.md)。
- **Files 根目录修复、实际分页/watch PASS：** 当前生产 DirectoryTree 在无本地挂载时实际走 scoped Host；`.6` 首轮完整读回 4,501 行/139,530B Unicode 文本、3 页 cursor 0→2001→4001、版本稳定，随后根目录失败 `path is required`。原发布 workspaceFiles/inspect 拒绝空 path，现仅 scoped root 改为 Session-relative `.`；legacy 空路径入口不变，定向 0.132s。修复后真实 Host 创建/改名事件自动刷新，关闭自动刷新实际终止订阅、手动刷新有效、重新开启真实订阅并显示新条目；最终 starts=2/callbacks=5，10.422s、模型请求 0、cleanup 全 true。有效分页未重复。此冻结源码已纳入 `.7`，`.6` EXE 原失败保留；不证明可见文件树或 symlink 权限。见[修复、原失败与真实 watch](../../../tmp/parity-implementation-20261103/files-host-flow/RESULT.md)。
- **工作区/会话实际管理与恢复 PASS：** 当前 Store + `.7` Host 对文件路径的真实注册拒绝保留同一 picker，原位纠正后 Host ID/path、Store 及 config 一致。两个 owned Git 目录、A1/A2/B1 三个明确成员，重命名、工作区移位、pin/unpin、archive/undo 及归档后精确删除 A2 通过；删除 B 注册仅在回执和 follow remove 都到达后关闭 B1，文件原字节与 B1 保留。真实 config 的 tree/manual/show/archive/collapse 及正常同 profile 重启恢复通过，A1 解除归档和 B1 明确重开有效，模型请求0。首段26.441s的 FAIL 是 QA 漏 archive-before-delete，原 schema/UI均拒绝活动对象删除；不改产品，复用同 owned profile 续段21.226s通过，不重复有效管理动作。另补 published workspace/follow replace→完整 admitted registry，协议新旧目标0.176s；不冒充真实权限撤销或可见窗口。见[实际链、原失败和精确修复](../../../tmp/parity-implementation-20261103/workspace-real-flow/RESULT.md)。
- **Skills 实际作者/调用/撤销/恢复 PASS（修复后源码）：** 当前 Store/Composer + `.7` Host 在 owned 项目/用户根创建技能，完整中文正文/description/groups/whenToUse 编辑与保存、模型调用开关和 user-only 成功。原 skill 工具两次实际读回已编辑正文；用户 slash 显式调用注入正文，撤销 user flag 后新会话不注入且 Picker移除。正常同 profile 重启发现 cold follow 先返回而 Agent promote 在后，完整目录停留全局空 fallback；现由真实 api-session/added available 捕获 owner、重新读取并退休旧 fallback，无定时重试/模型启动。另修调用开关只改 catalog 的旧标签及删除后 Picker残留；删除 winner重新读真实同名低优先级结果。两技能均取消删除保留→确认删除同步。首段36.733s有效创建/编辑/首轮保留，原产品失败不覆盖；修复后仅续原profile11.916s通过。定向0.168s、受影响既有目标0.211s；4完成轮/6主请求/3标题，cleanup全true/tripwire0。源码已纳入`.8`，历史`.7` EXE不含新修复；不是窗口、打开目录系统操作或 Skills Hub验收。见[三处根因及实际恢复续段](../../../tmp/parity-implementation-20261103/skills-real-flow/RESULT.md)。
- **预设保存/实际组成/恢复 PASS：** 生产 Manager 将 owned 默认 standard 保存为 minimal，普通 CreateComposerSession 未传 override，模型实际收到 minimal persona 和 13 tools；原 standard 会话仍原 coding persona、41 tools，真实 read 到模型。正常同 profile 重启后，新 Store/Manager 恢复默认及两会话身份，再普通新建仍 minimal；6 轮/9 主请求/3 标题，24.900s、cleanup 全 true/tripwire 0。两预设均 workspace-write，minimal 不是只读权限预设。结束恢复 owned 原默认，无产品改动；source consumer + `.6` Host 不代替窗口设置/Creator/真实 provider。见[实际采用与持久化](../../../tmp/parity-implementation-20261103/preset-real-flow/RESULT.md)。
- **Goal/Plan 实际状态/审批/退出 PASS：** 生产 Session/Interaction Stores + `.7` Host 原 /goal 创建 active/armed并实际自动 get_goal；原生 PauseGoal真实 user-cancel，不等普通助手完成。edit同ID/revision增加、resume读取新目标，正常重启保持durable active但activation disarmed、请求4→4无重放；原生CompleteGoal→complete，/goal clear→nil。原 /plan mode 同 profile 持久恢复亦无重放。独立Plan原profile续段只发新唯一输入，持久 read 结果已进模型、不重读文件；原 exit_plan_mode→专用 Approve重复点击仅1真实回执→下一实际step完整Plan guidance移除、41工具定义逐字节不变，submitted document精确1卡，明确下一轮正常结束。最后3请求/1工具/1结果/1回执，7份执行cleanup全true。原失败是QA工具/RPC envelope、pause合同、system消息、Web/CLI来源、requestId幂等和unnamed/named review混用，均保留；按正确发布合同复用有效段，没有改产品/Host/preset/journal。默认Web是unnamed legacy，named Discuss/Reopen不由此证明；EXE窗口及真实provider仍未验。见[组合收据与所有原始来源](../../../tmp/parity-implementation-20261103/goal-plan-real-flow/RESULT.md)。
- **原命令/手动压缩/恢复 PASS：** 生产 Store + `.6` Host 实际读取 compact/export/feedback/goal/permission/plan 目录，读取与错误不请求模型；invalid /compact usage 和 unknown 反馈准确。真实 read 历史后经 ExecuteCommandLine('/compact') 产生唯一 start39/summary41/replace42/end43，command/done44 success；原生成功 card 完整摘要、7 items/~2702 tokens。显式下一条实际使用摘要，旧压缩输入不再发送；正常同 profile 重启原记录/成功 card 保持、请求数6→6无重放，明确恢复后新输入仍使用同摘要。21.512s、7 请求=5 conversation+1 title+1 summary，cleanup 全 true/tripwire 0。三次 QA 协议观察错误与原失败保留，未改产品/Host/journal或放宽断言；不声称完整其他命令矩阵或 EXE 点击已过。见[真实命令与压缩闭环](../../../tmp/parity-implementation-20261103/command-real-flow/RESULT.md)。
- **问题/审批 PASS：** 默认 legacy 问题的原选项与自由文本返回、拒答取消、原 read-only preset、拒绝无文件副作用、单次允许精确写入、之后无升级写入仍被拒绝。7 个真实工具结果全部回传 provider 并持久入 Chat Store。两个产品根因已修入 `.2`：标准 `Client.Call` 回执信封，以及 legacy/timed 关闭权威区分；投影问题包括无限等待仍保留原 Host 权威。dsh 关联检查 1.409s，第二修复后 interaction 整包 0.290s。见[完整结果与首失败](../../../tmp/parity-implementation-20261103/interaction-flow/RESULT.md)、[实际报告](../../../tmp/parity-implementation-20261103/interaction-flow/result.json)。
- **Timed 问题 PASS：** 已发布 opt-in 模式实际 timeout=1 → ASK_TIMED_OUT → continued → 晚答 → settled；timeout=-1 超过默认 1 秒仍 open。真实 Host 撤销并重放 delivery 后，旧答案在 HTTP 前失效，新答案成功，成功回执不提前撤销待投影卡。修复后生产 Stores + `.2` runtime 共 2 个工具结果、6 次本机模型请求，定向 Session 检查 0.161s；修复源码已打入 `.3`。见[真实链结果](../../../tmp/parity-implementation-20261103/timed-question-flow/RESULT.md)。
- **子会话队列/停止/续聊 PASS（修复后源码）：** 原 `.5` Host 的 subagent 工具真实创建 child 并 read；当前生产 Store 以完整 parent/child/mode 地址和 token follow/发送，在忙时 queue、读取全文编辑和移除，实际持久 inbox 与父队列隔离。显式 steer 后真实下一步工具与完成消息可见；parent-owned stop 确实取消模型 HTTP，同一 child 再续聊、历史重开和原 parent 下一轮均通过，子输入未串入父 user transcript。四处已证实产品根因如前述；正常 idle child removed 只保留持久读取，deleted、root/unknown removed、关闭 pane 和 Host 变更仍撤销。12.401s、14 次本机请求、3 个真实工具/结果；两组受影响定向检查 0.174s/0.175s。产品与 QA 观察器的原失败均保留，全部 owned 服务/reader 已终态，tripwire 0。`.5` payload 未改，源码纳入新候选，不能把本收据当 `.5` EXE 或窗口已过。见[完整实际链与精确增量](../../../tmp/parity-implementation-20261103/child-session-flow/RESULT.md)。
- **本地群聊/成员 runtime PASS：** `.5` 真实 Host/dshbot + 当前生产 Client、RoomRuntimeReader、Session Store 创建两个合成成员和一个 room；runtime 实际 running→passed，原 `_request` session/list 确认成员各为独立 root、非个人/room 会话。明确打开成员读到 8 行持久工具历史；主群原 legacy-0 为 3 行，GroupReplyTarget queue 明确回复后同线程为 6 行，4 对实际 ask_participant call/result。17.499s、11 次本机请求（8 成员、3 标题），原失败为 QA provider ID 与编译字段错误，修正后主链一次通过。无产品修改，owned 进程/监听/reader 全部终态、无出网；EXE 窗口与外部平台仍未验。见[实际群聊与逐项收据](../../../tmp/parity-implementation-20261103/group-runtime-flow/RESULT.md)。
- **MCP 本地 stdio PASS：** `.3` 随包 Host 与生产 Controller 实际添加/连接、discover/test；模型调用成熟 SDK server 的固定 owned 文件只读工具，精确结果回到 provider 与持久 Chat Store。停用后同一会话下一轮模型请求已无该工具，删除记录成功；3 个自有进程均由保留的 Windows handle 证明退出。首次运行通过，1 个工具结果、4 次本地模型请求，监听全部关闭、tripwire 0，未改产品。外部 MCP/OAuth/HTTP transport 与 EXE 点击仍未验。见[实际结果](../../../tmp/parity-implementation-20261103/mcp-local-flow/RESULT.md)。
- **Project/原生 Team 只读委派 PASS：** `.3` 真实 Host + 生产 Client/Session Store 完成原 Lead 创建/重命名、request/todo/delegate、worker 真实读文件/生成材料/report、native settlement 自动唤醒原 Lead，再读取匹配报告并交付。只有一次 QA human submit；工作 done、worker idle、Team task completed、需求 completed、交付 delivered/applicable，最终正文持久化。12.220 秒、12 次本机请求、9 个真实工具调用；8 个结果回 provider，finish:true 的 report 另由真实报告/结算/交付引用证明。首轮 QA 误认动态 system context 为末尾工具结果已纠正，未改产品。此只读收据不覆盖 development/停止或多需求队列；后续两项分别补证，实际窗口仍未覆盖。见[真实结果](../../../tmp/parity-implementation-20261103/project-real-flow/RESULT.md)。
- **Project development/独立 worktree/停止续接 PASS：** 修复 Windows 新 worktree 在有效长路径上继承 GIT_DIR 失败，保持原目录/分支/基线/receipt；只初始化刚创建且仅有 `.git` 的目录，已有或中断目录不 reset。包括没有 checkoutPending 的旧 preparing 回执，未检出或 dirty 均不能假报 ready；真实 Git 定向通过 2.023s、1.723s。当前 Client/Store/修复后 bridge + `.4` runtime 实际执行同成员、同工作、同 worktree 的 write/read、停止和明确新消息续接、原生结算和持久化交付。provider 取消 5ms，stop RPC 40ms，目录 handback/idle 124ms。旧 aborted 与新 completed 终态均被精确归纳后 pendingSummary=false，changes final=true、精确一文件 +1/-1；源目录 dirty/untracked/index 保留。原失败及 QA 游标/legacy 标志误读均保留，未重复有效 stop/write 链。多需求队列另见下一项；任意 shell、外部审批和实际窗口仍未覆盖。见[修复与完整实际结果](../../../tmp/parity-implementation-20261103/project-development-flow/RESULT.md)。
- **Project 多需求队列/自动推进 PASS：** 当前 Client/Session Store + `.5` 真实 Host 创建 A active、B/C discussing；公开 enqueue 和 move 得到 C→B。第二次且最后一次用户源修订 C 为 revision2，身份/队位不变；旧 C1 cancel 被原子拒绝且 C2 队列不变，有效取消 B。A 经真实 worker/read/report/native settlement 和持久化交付完成后，Host 原生自动提升 C2，无第三次用户提交；C2 再实际委派/读取/报告/交付。最终 A/C completed、B cancelled、队列为空，B 无工作或成员。首次实际运行 14.950s，25 次本机请求/19 个工具/17 个结果，两个 finish:true 报告另有接受/结算/交付引用证明。无产品修改，cleanup 全 true/tripwire 0；不重复有效 development/stop 链，不代替窗口或外部依赖。见[实际队列与逐项收据](../../../tmp/parity-implementation-20261103/project-queue-flow/RESULT.md)。
- **自动定时任务 PASS：** 原 schedule_create 创建正常绝对 at，未再 Submit/Run/Refresh，真实时钟在目标后 38ms 持久投递、约 88ms 到达模型；原 read 读取独占 marker，结果和回复进入原会话，生产 Schedule Store 自动观察 active → inactive。原 profile 重启后规则、消息与执行历史读回，公开删除后目录为 0，恢复未新增模型请求。2 个真实工具结果、5 次本机模型请求。首轮 QA 错把非人类来源的 context 呈现当消息缺失；原始 source.kind=schedule 完整，修观察器后续持久化检查，未重跑有效自动链。此单次收据不覆盖循环/取消，后续补证见下一项；日历/DST、跨停机补发及窗口仍未覆盖。见[完整结果](../../../tmp/parity-implementation-20261103/schedule-real-flow/RESULT.md)。
- **循环计划/未到期取消 PASS：** `.4` runtime 正常公开 every_seconds=60，同一计划在 03:42:32、03:43:32 自动 read/回复并保存两条历史，始终 active 且推进下一周期；第二次完成后、第三次目标前约 59.387 秒经生产 Store 删除。保持同 Host/模型在线至第三次目标后 5.074 秒，无新投递/请求/工具，目录仍空。3 个工具调用/结果、7 次本机请求，首次通过，无产品修改，cleanup 全 true/tripwire 0。上一条单次 at/恢复结果未重复；日历/DST、跨停机和可见窗口仍未覆盖。见[实际周期与取消窗口](../../../tmp/parity-implementation-20261103/schedule-recurring-flow/RESULT.md)。
- **Pad 静态卡 PASS：** `pad_create/get/list/update/get/delete/list` 的 7 个工具结果全部回传并持久入 Chat Store；HTTP/SSE/生产 Pad Store 数量 1 → 1 → 0、版本 1 → 2 一致。见[静态工具结果](../../../tmp/parity-implementation-20261103/pad-static-flow/RESULT.md)。
- **Pad 数据/undo/恢复 PASS：** 真实拒绝后 HTTP 源执行 0 次；错 fingerprint 返回 409；原 fingerprint 批准后才首读/刷新。公开 undo 令版本 1 → 2 → 3、原内容和批准状态恢复；正常 Host 停启后同 ID、版本、批准源与缓存恢复，刷新继续有效。删除后超过原 interval 不再取数。消费修复后 Go 回执，未绕过审批；命令源和后台 agent 见下项，窗口绘制仍未验。见[完整结果](../../../tmp/parity-implementation-20261103/pad-data-flow/RESULT.md)、[报告](../../../tmp/parity-implementation-20261103/pad-data-flow/result.json)。
- **Pad 后台 agent/Run API PASS：** `.2` 的原 Host 实际创建三轮后台 agent：真实 read → submit 成功；自有文件不存在时真实 read 失败 → report_failure，保留旧数据；恢复文件后下一轮成功并清除错误。running refresh 去重、`agent:null` 清除计划有效，HTTP/SSE/生产 Store 一致。公开 Run API 拒绝空 prompt，合法 prompt/system 无工具调用正常回传。8 个真实工具结果、15 次本机 provider 请求，服务全部释放。卡片 JS `dsh.run` 桥仍单列，不由 API 成功代替。见[结果](../../../tmp/parity-implementation-20261103/pad-agent-flow/RESULT.md)。
- **Pad 自动调度/运行中停止 PASS：** 同一原卡 everyMinutes=15，恢复后公开 due 未重置，无手动 refresh、新卡或时钟修改；真实 scan 在 due 后 10,519ms 触发 read → submit，HTTP/SSE/Go Store 同值 idle。另在真实模型 SSE 尚未完成时停止 Host，5,335ms 内停止流程完成并观察到 HTTP 取消，旧 submit 为 0；新 generation 恢复中断状态，下轮实际读到新 marker。自有卡已删除、监听全部释放、tripwire 0。首轮 helper 独有 20 秒 SSE 总超时失败保留；核对生产默认无总 Timeout 后，只改 helper，续同一卡的真实等待，没有重跑有效 active-stop。见[完整结果](../../../tmp/parity-implementation-20261103/pad-agent-lifecycle/RESULT.md)。
- **Pad 固定 command 源 PASS；卡片 JS 桥 NOT-RUN：** `.3` 实际 `pad_create` 后由生产 Interaction Store 手动拒绝，公开 refresh 与超过 interval 的观察均为执行 0 次；原 fingerprint 批准后，原 runCommand/spawn 两次读取独占 marker，批准/刷新结果进入 HTTP/SSE/Store，两个子进程均退出 0，删除后不再执行。首次运行通过，未改产品。`dsh.run` 卡片 JS 桥必须经真实 Parent/原生 PluginSurface 建立 card owner，没有公开 headless 卡入口；现有语法测试与富内容 DOM fixture 不能替代，留待原生窗口验收。见[分项结果](../../../tmp/parity-implementation-20261103/pad-bridge-command/RESULT.md)，整体报告为 PARTIAL，不把 command 通过扩记为 JS 桥通过。
- **Windows 安全存储双向互操作 PASS：** 从 `.2` 随包 Electron 44.5.0 runtime 复制出独占运行目录，只加载本次自有 app.asar，不载入原版应用代码/用户数据，不修改 Electron fuses，不创建 BrowserWindow。真实 `safeStorage.encryptString` 生成 `v10` 信封与自身 Local State，生产 Go `credentials.Load` 完整读回六字段及嵌套设备字段；Go `Save` 后，真实 Electron `decryptString` 又完整读回更新后的合成值。两个 Electron 与 Go 进程均正常退出，无残留。见[机器报告](../../../tmp/parity-implementation-20261103/credentials-interop/result.json)、[清理](../../../tmp/parity-implementation-20261103/credentials-interop/cleanup.json)。这闭合格式互操作，不代替真实旧 profile 接管或账号登录。首次 QA 目录构建被既有 `tmp/go.mod` 隔离规则拒绝，改用单文件诊断入口；未改模块配置或产品。
- **真实压缩历史导入/恢复/续聊 PASS：** `.6` Host 实际生成两个可区分的 session.v4.jsonl.zstd，源 read 工具结果确已回传并持久化，正常停止后才扫描。生产 RecoverInterrupted/AcquireExternalMaintenance/BuildCatalog/RescanCatalog/ApplyCatalog 保留单选，Copied=1、Skipped/Failed=0，未选源不在目标、既有目标原字节保留，事务回收后维护入口正常。目标原 Host + 新 Store 准确恢复两个源 user/assistant 正文及真实工具结果；同一 ID 下一轮模型实际收到原上下文、完成持久化回复，既有目标历史仍可打开。源 62 个条目的完整清单、bytes/metadata/链接目标不变。QA junction/目录枚举元数据错误和 env.ps1 覆盖局部变量的首次失败保留；改用正确 Lstat 与调用变量，复用已停稳真实源，不重复生成。最终目标续段 21.211s、全部 owned 服务/reader/进程终态，tripwire 0，无产品修改。不包含真实账号/profile、OAuth、插件下载或可见窗口接管。见[实际格式、事务与恢复收据](../../../tmp/parity-implementation-20261103/import-real-flow/RESULT.md)。
- **原生终端 backend PASS：** 实际随包 DLL + 生产 MyGo ConPTY 创建 cmd，验证固定输入/完整输出、独立 cwd、resize 后真实 `mode con` 为 92 列 × 28 行、自然退出码 0、Done/Close 和进程释放，356ms。不是 terminal 窗口、IME 或 Host terminal 验收。见[终端结果](../../../tmp/parity-implementation-20261103/terminal-runtime/RESULT.md)。
- **Host 保留终端协议/PTY PASS（修复后源码）：** 当前 Store 与 `.3` 原 runtime/DLL 完成真实 attach，原 HostView/Attachment/Render.Data/Acknowledge 消费 19 帧；固定输入/cwd、真实 mode con resize、隐藏保留、同 terminal ID/cmd PID 重连和旧 attachment 拒写、关闭后列表/进程归零，10,475ms。三个受影响包定向通过；无模型或可见窗口。修复后首轮 QA 的贪婪 OSC 剥离误吞真实输出，离线原帧证明后只改观察器，失败保留。正文挤占另已修复，正常/窄 pane 的实际软件几何与图像见前述布局结果；未为无关布局改动重跑有效 Host 链。见[完整结果](../../../tmp/parity-implementation-20261103/host-terminal-flow/RESULT.md)。
- **补充文件 stat PASS（修复后源码）：** 两个活跃 main 回调均使用同 frame 的 fileClient/fileSessionID，新 scoped helper 实际传 workspaceFileScopeId/path，保留旧导出入口合同。workspacefs、Files owner/迟到保护、cmd 格式/envelope 定向均首次通过；单次 `.3` 真实 Host 对独占 27B 文件返回准确路径、非空版本和大小，10,594ms，清理全部完成、无外网或模型。原已正确的文本分页、目录和媒体链未重复。见[完整结果与已消费装配](../../../tmp/parity-implementation-20261103/scoped-file-stat/RESULT.md)。
- **手机资源：** 首候选生产 `ResolveResources` 消费者一次通过 1.310s、无 Skip；该目标不启动 Node。各次完整构建管线均实际加载 SQLite/PTY 绑定并完成内存查询；未创建手机 daemon 或配对设备。见[消费者结果](../../../tmp/parity-implementation-20261103/mobile-consumer/mobile-result.md)和当前构建日志。
- **手机端口自冲突已修入 `.5`；完整进程链仍阻塞：** 默认 Host 与发布 LAN SPA 都使用 3180。managed Host 现在在原端口扫描中保留 MobileWebPort，普通 FreePort/无保留项 caller 不变；两个定向目标通过 0.132s，main 编译通过。修复后 Supervisor + `.4` 真实 Host 请求并保留自有 55033、实际 Host ready 于 55035，55033 仍可真实绑定 companion，认证 session/list 成功，10,121ms，cleanup 全 true。04:27:37 恢复检查实际监听仍为 127.0.0.1:3180/PID 37664，独占绑定也返回 AddressAlreadyInUse；一次空查询不足以判定释放。没有启动本轮手机服务或新 profile，daemon/SPA/offer/link/设备管理链全部 NOT-RUN，需旧预览所有者正常释放端口后恢复，未关闭或重启它。见[源码修复与真实分配](../../../tmp/parity-implementation-20261103/phone-host-port/RESULT.md)、[固定端口的生产依据和新探测](../../../tmp/parity-implementation-20261103/phone-runtime-local/RESULT.md)。
- **构建工具：** `package.ps1 -StageOnly` 保留完整载荷检查但不运行 NSIS；`-GoExecutable` 一致使用指定缓存 Go，原默认 `go` 保持。早期 PATH Go1.27.0 的离线 toolchain 失败和空 stage 保留；改用已有 Go1.27.2 后成功，未下载工具或改系统安装。
- **剩余依赖：** watcher 连续目标仍因文件 symlink 权限缺失未通过，未原样重复 `EPERM`，见[权限结果](../../../tmp/parity-implementation-20261103/symlink-result.md)。LIFE-025 缺最终验收安装器与实际发布，见[发布交接](../../../tmp/parity-implementation-20261103/release-source-result.md)。模型/费用、CNB空仓目标和仅隔离候选桌面范围已获明确确认；随后用户要求先停止测试，当前不执行模型、桌面或发布工作。

### 已消费的交付与证据

以下报告位于 `tmp/parity-implementation-20261102/`。通过表示报告所列本地、隔离或离屏检查通过，不表示实际用户窗口、模型、设备、安装或外部服务已通过。

| 域 | 产品与检查结果 | 报告 |
| --- | --- | --- |
| 工作区/侧栏 | 注册、成员、恢复、删除失败、草稿/导航 fences；production Store → 真实隔离 Host → 实际 sidebar filter 的新建会话链通过，自有 Host 已退出 | `workspace-result.md` |
| 聊天/轨迹/输入 | 异步 @、持久 child 草稿/队列/全量正文、Goal/plan owner、上传字节、secret mask、轨迹详情/折叠/diff/usage、工具收据与完整 canonical composer 席位；定向检查和三张离屏图已交 | `chat-result.md` |
| 设置/账号 | 登录 origin/metadata、凭据引用、live/per-ID 模型编辑、供应商目录、默认预设、语言和费用；Store/原生离屏交互通过 | `settings-result.md` |
| 插件/市场/技能/MCP | Discover、新 MCP 表单、GitHub/prerelease/历史日志、公开位置消费者、第三方 root/plugins.tab、typed composer、原 owner 富内容与无损大正文；Go 检查及真实离线 Chrome DOM/Canvas/WebGL/媒体/600行尾部动作通过；公开 File/drop 725,000 字节、实际 File/Blob 预览与类型保真通过；11 月 3 日另补 MCP 实际 stdio 调用、停用撤销和删除，兼容条件见下 | `plugins-result.md`；新 MCP 结果见上方 |
| Inspector | 真实 Group/member、ID/T/S、原生对象读者、惰性树/分页/导航；定向检查通过 | `inspector-result.md` |
| Pad | 原生坞/卡片/审批/模型/布局/state 定向检查通过；11 月 3 日已补静态工具、审批/HTTP、undo/重启/删除、后台 agent/Run API、真实 15 分钟自动触发、运行中取消恢复及固定 command 源。窗口、卡片 JS 桥仍未验 | `pad-result.md`；新证据见上方五个 Pad 结果 |
| 工作面 | 文件/图片/OpenWith、Git PR/错误、终端链接、独立 Project root、mini、多 terminal occurrence、插件 right-pane 生命周期；定向检查通过 | `workbench-result.md` |
| 编排 | Project/普通 Team、四个 bridge 动作、归档/历史/材料、Hero 草稿入口、日程/Whale/IM；Go → Node → 临时 Git 与六包检查通过；11 月 3 日另有真实原生 Team、development worktree/停止续接/交付、单次和循环自动任务/恢复/历史/未到期取消证据 | `orchestration-result.md`；新 Project/定时结果见上方 |
| 桌宠/语音/壁纸 | 计粮来源/迁移、完整皮肤图集、通知/快捷卡、壁纸事务、共享原生特效、PCM 电平；定向检查通过，旧 watcher symlink fixture 的 EPERM 单列 | `pet-result.md` |
| 生命周期/导入/远程 | 原生更新中心暂停恢复、对象级退出明细、profile lock、导入重扫/格式、typed mirror、长期手机 link、SSH 晚到错误和代理/PAC；本地定向检查通过，已发布 runtime 闭包同步保留恢复材料 | `lifecycle-result.md` |
| 顶部/主框架 | 48 DIP 原生 caption + 独立页签行、正常/窄窗与键盘、真实动态 handle、Goal dock、数据安全确认、Pad 正文留位；完整 frame 检查 7.453s；新增第三方 root caption 与窗口原语定向 0.179s 通过 | `root-result.md` |
| 主程序装配 | 已交接口实际消费；主体 cmd/workspace/profile 1.332s、review 改变的目标检查、About+MainParity 0.256s、root 原生对话准入 0.202s、pet/wallpaper 原目标 0.204s 均通过；已查看 1200/400 宽 root 离屏图。新增 File 流/receipt/reader/stale/清理及 retry 0.279s、原 File.type 保真 0.241s 通过 | `main-result.md` |

各报告记录实际命令、源版本、首次失败原因和有效证据范围；不重复未改能力的检查，也不把编译或创建 browser stub 算作真实交互。

归并补证：pet 的主包行为目标先前仅被在途编译问题挡住、从未执行。首次实际执行发现 Whale catalog 旧 fixture 固定 rpcId 被 identity 校验拒绝，及 wallpaper 纯配置解析调用 UI 主题服务使 headless 无 UI loop 时阻塞；`main-pet-checks.log` 保留失败。原 pet owner 已修 fixture 回显和 UI ready 准入，唯一 cmd owner 只续原受影响目标，`main-pet-followup-checks.log` 记录 PASS 0.204s；不以 MainParity 编译替代该证据，也不以 headless 通过替代真实桌宠/设备/外观验收。

### 独立审查已修的真实集成缺陷

- 插件简写槽名统一到真实 canonical；reasoning 的真实外层 seq 与 user.content Factory occurrence 已消费，不再只写回调却永远不命中。
- 动态 conversation 返回 Chat/Trajectory 同步原真实 selector；折叠页签 More 与正常行共用完整动态入口和动作。
- Pad `ask(..., false)` 仅填草稿、不调用模型；insert 替换当前 UTF-16 光标选区，保留已有草稿冲突判断。
- 独立 Project Files 分开打开时的 admission 与长期 access；切聊天不会撤掉同 Host/client/Project/root 的已开文件权限。

完整独立审查见 `tmp/parity-implementation-20261102/review-candidate.md`。审查核对全量分母并发现六项实际整合缺陷，不代表逐一执行全部动作。

### 相邻闭环与仍然存在的条件

- 富编辑/文件/日期/图像/媒体/Canvas/React portal 使用同原挂载 owner 的受限富内容 seat，不打开整套 Host 默认页面。实际无网络 Chrome headless 检查通过；独立 seat 审查发现的撤回、过期 ack、portal owner、逐节点 clip、hidden 与强 mask 问题均已由原 owner 修复并复读。见 `tmp/parity-implementation-20261102/rich-seat-review.md`。Windows/MyGo HWND、IME、原生文件/日期弹层、隐藏 source 的绘制帧 ack 和多 clip/modal 恢复仍未实测。
- 只接受明确 aria-controls 或可回溯到 exact root 的 React portal；无任何可证明归属的手工 DOM popup 不能靠时间猜 owner。任意 Electron/preload/私有编辑器与浏览器 attachment registry 对象不通用兼容；产品显示实际模块 ID/错误/本页重试。公开草稿/选区/选择附件/移除/提交、composer.bar 的直接 File/drop/resolve/retry 均已有真实消费者，不归入上述私有边界。目录引用继续走原生桌面路径入口；不将 File.path、私有 registry ID 当路径授权。
- 公开 File 流复用同一原生 cache/attachments、捕获 target 与 PrepareAttachmentProgress；同源临时 cookie/page/binding/revision 复验，失败或过期未采用的文件清理，读取只按当前 opaque PreparationID。HTTP 流不放大通用512KiB socket上限，不发明原版普通文件大小/数量限制，图片沿真实 Host limits。主包690,000字节原文件名上传/receipt、双GIF批次、取消/错误/过期清理、重试及自定义/显式空File.type检查通过；浏览器725,000字节drop/重连reader/真实Blob预览另有实际软件渲染证据。见各报告最后两组差异日志。
- 地图保留 raster、YouTube 保留显式激活 seat；Genui 远程资源仅 public address、支持 MIME 与16MiB范围。原版 HtmlView 本身也禁止直接 connect/外部 CSS，不能继续把它们当迁移缺失；实际40–12000高度/主题/表面合同沿用原组件。
- remote mirror → 工作区实际 UI 采用及配置提交边界已增加 machine/Host/generation/receipt 复验；main 受影响检查通过 0.268s，不把旧镜像路径迟到采纳到新 Host。
- Project 目录行贡献已按 Available、picker:false、真实目录已有项目/拒绝条件修正；复用异步预检缓存，其他目录扩展零值合同保留，新目标检查通过。
- About 已在无 Host/重启后恢复同一更新中心，错误与暂停/ready状态可见，不猜 shell-only；三项 UI 与两项主包 fixture 已通过。第三方 root 保留原生 caption，并对目录/工作区 picker、显式 native 设置和恢复路由让行。
- Configurable-provider 的 nativeAuth/defaultBaseURL/baseURLEnv 已在原生编辑器实际消费，定向 UI 与 Controller 证据见前述结果；没有冒充真实 provider 登录验收。未有活跃 UI 调用的旧 Cordis/helper 仍是潜在合同风险，不等同于当前 IM 运行失败，详见上述消费者审查。

### 低磁盘期间的有限只读核对

本轮没有产品或新增实际运行验收进展。只核对上一轮已暴露的文件/目录相邻合同，避免在未恢复空间时制造另一套修复/夹具：

- 主文件入口始终提供 `ReadTextPageContext`（`main.go:6535–6540`），当前 tab 的选择/重载优先异步分页（`files.go:560–569`、`reload.go:45–48`、`text_read.go:125–131,174–181`）；图像和工作簿通过捕获的 Session 调用 `mediapreview.ReadSnapshot`（`main_parity_integration.go:334–353`、`remote_workbook.go:15–28`、`mediapreview/reader.go:82–106`）。这次没有确认新的活跃 scoped 读取缺陷，不扩改无生产调用的 `ReadRemoteBytes` 或改变旧导出入口合同。
- 原 DirectoryPicker 的 API 实际属于 `api/workspace-controller`；首次省略 path 以打开 Host 用户主目录是原合同，不能把 Files 根路径修复的 `.` 套用过去。list/create 参数、原路径/空白名称、隐藏项/虚拟盘符根、真实 main/controller 入口与 `.8` 发布 browse backend 相符，未找到新源码断点。生产入口见 `cmd/whaleisle/directory_picker.go:170–177,391–420`、`workspace_navigation.go:125–143`，原发布合同见 `api/workspace-controller/lib/types/directory-picker.js:42–46,113–120`、`host/directory-picker-browse/lib/index.js:219–223,266–288`。未运行 Host 或原生浏览/创建，不升级验收状态。

历史阻塞：该只读阶段结束时空间109,985,792B（约105MiB），且尚无清理授权，连续第3轮没有产品/运行进展后曾标记 blocked。此后用户明确批准精确清理，已完成并解除磁盘阻点；又按用户最新指示停止测试并转回界面实现纠偏。当前状态以上方新记录为准，原有阴性源码核对及历史收据不转写为全量完成。

### 验收与发布依赖

- 当前不执行电脑控制/真实用户 profile、账号、模型、IM、SSH、设备和安装操作；这些路径保持未验，不能据源码宣称实际可用。
- 用户报告的旧 MyGo rc.3 点击“轨迹”不命中的精确根因仍 unknown；新原生控件的软件点击/键盘/布局已过，不等价于受影响 Windows 窗口已过。
- LIFE-025 原生发布源/CNB 部署需发布侧完成；未配置线路已明确禁用，不把空 URL 当可用。
- FLOW-023、FLOW-046、LIFE-039 属原项目未发布 dirty 增量，不混入 0.3.13 已发布基线，也不复制半套协议。
- 本轮无 CI、push 或最终安装包。独立完整候选已生成；后续使用同一候选验收正常/窄窗、工作区→会话→发送/工具/审批→停止→下一轮→恢复及复杂插件/媒体的实际窗口路径。旧预览和本地软件测试不能代替这一步。
