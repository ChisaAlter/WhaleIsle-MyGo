# 功能对照矩阵与可修项修复（2026-10-26/27）

> 2026-11-02 校正：下文是历史阶段记录，其“20 项已实现”不能继续作为当前完成或可用性结论。全量生产接线审查已发现新会话归属、中文删除取消、设置、MCP 等实际断路；顶部单行布局还被错误局部测试固化。当前原安装版已为 0.3.13，不再是下文 0.3.10 基线。以 [本轮全量审查](full-audit-20261102/) 为当前问题清单，保留下文原始证据的真实作用域。

最初一轮对 MyGo 与 Electron「鲸屿本体」做**功能面**对照，并按「代码可补」原则修复了两处无效/缺失交互；下文按阶段保留初始结果和 2026-10-27 后续证据。
当前来源基线是实际安装版 Whale Isle **0.3.10** 及其当前运行包；ui-theme 包的实际版本为 **0.2.1-rc.2**，不能把旧 `reference/electron/` 0.3.3 快照或桌面版本号视作同一版本。初始阶段只读源码，后续已用原安装可执行文件在 fresh 自有 profile／workspace 下实际启动第二个 Electron 实例，取得同尺基线；未修改安装目录、真实账号资料或既有用户实例。版本、隔离条件、有效图及失败记录见 [Electron 基线记录](../../tmp/electron-parity-fixture-20261027/RESULTS.md)和 [ui-theme 来源清单](../../tmp/builtin-theme-parity-20261027/source.json)。
本矩阵不替代像素级合同 [acceptance-ui-parity.md](../acceptance-ui-parity.md)，也不把历史截图或文档自述当作本轮新证据；下列状态数量沿用功能面口径，不因补充阶段说明而改验收勾选或阈值。

## 1. 口径

- **已实现**：入口、主流程、失败路径和持久化均在当前源码中闭合，且至少有一层自动测试或历史实机证据。
- **部分**：主流程可用，但仍有明确的代码边界或未在本环境完成的验收。
- **缺失**：无入口或仅有占位。
- **无法验证**：本环境缺真机、证书、账号或其他外部条件，**不等于**功能缺失。
- 证据路径均相对仓库根；`cmd/whaleisle` 与 `internal/desktop/...` 是本轮测试门禁范围。

## 2. 矩阵（25 项）

| # | 功能面 | MyGo 现状 | 证据路径 | 与本体差异（一句） | 验证等级 |
| --- | --- | --- | --- | --- | --- |
| 1 | 会话与流式输出 | 已实现 | `internal/store/session/`（`store.go`、`follow.go`、`assistant_stream.go`、`page.go`）、`internal/ui/chat/chat.go`、`internal/ui/composer/` | 未发现功能级差异；persistence 深层字段仍有 `json.RawMessage` 回退（已知技术债） | 代码 + 单测 + 真实 Host 历史证据 |
| 2 | 工具调用 / 计划 / 队列 | 已实现 | `internal/ui/chat/toolrow.go`、`tool_details_*.go`、`planboard.go`、`queue_dock.go`；`internal/store/session/planboard.go`、`queue_editor.go`、`queue_projection.go` | 未发现功能级差异；队列编辑/插话/移除与工具详情族均可用 | 代码 + 单测 |
| 3 | 设置 19 个专页 | 已实现 | `internal/ui/settings/sections.go`（20 个 SectionID，含 `other`）、`settings/*.go`、`cmd/whaleisle/settings_body_wiring.go` | 页面与控件齐全；逐页「控件×空/载/错/成功」截图穷举仍属验收合同未完成项 | 代码 + 单测；像素验收未做 |
| 4 | 插件市场 | 已实现 | `internal/desktop/market/`、`internal/ui/market/`、`internal/ui/pluginmanager/` | 未发现功能级差异；安装需 Host 与网络资产，离线只验证状态/错误路径 | 代码 + 单测 + 历史实机 |
| 5 | 插件槽位 | 部分 | `internal/desktop/pluginslots/`、`pluginui/`、[P2 生产入口 proof](../../tmp/plugin-inline-production-p2/main-66dd7b511bad47959870f80b29d1e8e3/captures/result.json)、[双尾退出](../../tmp/plugin-inline-production-p2/main-66dd7b511bad47959870f80b29d1e8e3/cleanup.json)、[定向滚动结果](../../tmp/plugin-inline-production-p2/scroll-a6b803ac7fe34d2eaaa4e43894f85939/captures/result.json)、[滚动退出](../../tmp/plugin-inline-production-p2/scroll-a6b803ac7fe34d2eaaa4e43894f85939/cleanup.json) | P2 已经真实 mainView／Host follow／显式激活，以单一原 browser owner 的位图在两个原生轮尾显示并验证各自点击；普通 112-DIP 轮尾定向滚动、原始控件 Unicode／完整中文预编辑取消提交及 fresh bitmap、3200-CSS-px 长行的负Y与可见子裁片已实测；显式portal的原节点显示／可信点击／关闭后fresh bitmap已通过并自然退出；系统IME／物理输入、任意富控件、可访问性和整体像素等价仍有边界 | 代码 + 受控 Host fixture + 实际双轮尾点击／普通轮尾定向滚动 PASS／自然退出；整体仍部分 |
| 6 | 自动化任务 | 已实现 | `internal/store/schedule/`（`store.go`、`edit.go`、`history.go`）、`internal/ui/schedule/`（`panel.go`、`presentation.go`、`timing_picker.go`） | 未发现功能级差异；删除/保存/投递记录与只读结束态均有门禁 | 代码 + 单测 |
| 7 | 技能 / MCP | 已实现 | `internal/ui/settings/skills.go`、`skill_hub.go`、`mcp.go`、`mcp_form.go`、`mcp_import.go`、`internal/desktop/mcpconfig/` | 未发现功能级差异；MCP 登录/重试/工具探测按 Host 能力降级 | 代码 + 单测 |
| 8 | 远程手机 | 部分 | `internal/desktop/phoneremote/`（`runtime.go`、`bridge.cjs`、`shell_routes.cjs`、`host_actions.go`）、`docs/parity/android-remote-interoperability.md` | 配对/QR/LAN 网关/远程 shell+git 路由已接通；无 Android 真机、mDNS 发现、TLS/SSH 主机与断线重连矩阵 | 代码 + 隔离 loopback live 测试（见 §5） |
| 9 | 系统通知 | 部分 | `cmd/whaleisle/system_notifications*.go`、`internal/desktop/notifications/`、`scripts/nsis/notification-integration.nsh` | WinRT 激活器、AUMID、快捷方式与点击分发已实现；真实通知中心呈现/点击与旧版 `Electron.lnk` 迁移未做物理验收 | 代码 + 单测 + NSIS fixture；真机未验 |
| 10 | 托盘 | 已实现 | `cmd/whaleisle/desktop_tray*.go`（菜单模板与 `tray-menu.js` 逐项一致）、`trayicon.go` | 未发现功能级差异；首次隐藏提示与运行任务确认沿用既有行为 | 代码 + 单测 + 历史实机 |
| 11 | 桌宠 | 部分 | `internal/desktop/pet/`、`cmd/whaleisle/pet*.go`、[桌宠分阶段记录](../features/pet-renderer-parity.md)、[cadence 实测](../../tmp/pet-cadence-gui/result.json)、[脚点／唤醒边界实测](../../tmp/pet-boundary-gui-round2/result.json) | 后续已补脸／呼吸与拾起，task59 单 Z 粒子、task64 每可见 RAF 缓存呈现及 task72 绝对 2 CSS px 脚点／严格唤醒 expiry 均完成源码、回归和实际 WebGPU＋Anime4K 验证；长期性能、外部硬件和全窗像素仍未验 | 代码 + 多轮自有神经 GUI／左右与拾起公式／150% 与 100% 双屏；59／64／72 已验证，完整硬件／像素仍部分 |
| 12 | 快捷聊天 | 已实现 | `internal/desktop/pet/quickchat*.go`（248×360，min 248×220）、`cmd/whaleisle/pet_quickchat.go`、`docs/features/pet-quick-chat.md` | 未发现功能级差异；真实 Host 发送/模型切换/取消的端到端仍属运行时边界 | 代码 + 协议单测 + 隔离 GUI |
| 13 | 文件预览 | 已实现 | `internal/ui/files/`（`files.go`、`preview_toolbar.go`、`text_*`、`table_*`、`image_*`、`document_surface.go`）、`internal/desktop/workspacefs/` | 未发现功能级差异；本轮修掉非文本预览的无效「跳转到行」入口 | 代码 + 大量单测/验收 fixture |
| 14 | 终端 | 已实现 | `internal/ui/terminal/`、`internal/store/hostterminal/`、`internal/desktop/terminalruntime/`、`docs/features/host-terminal-*.md` | 未发现功能级差异；Host follow 仅 snapshot/output/state 三帧，写/缩放走一元 RPC | 代码 + 单测 + 真实 ConPTY 用例 |
| 15 | Git | 已实现 | `internal/desktop/git/`（status/diff/branches/commit_selection/discard/pr/publish）、`internal/ui/frame/changes_review.go`、`internal/ui/diffsurface/` | 未发现功能级差异；选择性提交用临时 index，丢弃单文件有复核令牌 | 代码 + 真实临时仓库测试 |
| 16 | Office 文档 | 已实现 | `cmd/whaleisle/office_document.go`、`internal/desktop/pdfpreview/`、`internal/ui/files/table_preview.go` | 采用 Host/本机转换到 PDF 后预览，非进程内 docx 排版；转换失败/超时/过大/源变化均有提示 | 代码 + 单测；真实 Office 渲染依赖 Host |
| 17 | 图片预览 | 已实现 | `internal/ui/files/image_read.go`、`image_view.go`、`internal/desktop/imagepreview/`、`svgrender/` | 未发现功能级差异；GIF/ICO/SVG/AVIF 为受限原生解码，拒绝序列等边界已明示 | 代码 + 单测 |
| 18 | 搜索 | 已实现 | `internal/store/session/search.go`、`internal/ui/sidebar/session_search.go`、`internal/ui/files/filter.go`、`internal/harness/extra_overlay.go` | 未发现功能级差异；正文搜索失败时保留标题/工作区匹配并显示错误 | 代码 + 单测 |
| 19 | 快捷键 | 已实现 | `internal/shortcuts/`、`internal/shortcutprefs/`、`internal/ui/frame/shortcuts.go`、`internal/ui/settings/shortcuts.go` | 未发现功能级差异；仅实现页面命令与 Host keybindings，不迁移 Electron 全部命令 | 代码 + 单测 |
| 20 | 主题 / 外观 | 已实现 | [实际主题来源](../../tmp/builtin-theme-parity-20261027/source.json)、[字体与渲染策略冻结](../../tmp/skin-parity-20261027/freeze.json)、[布局冻结](../../tmp/skin-layout-parity-20261027/freeze.json)、[主题卡冻结](../../tmp/theme-card-freeze-20261027.json)、[设置整包回归](../../tmp/composer-settings-installed-20261027/full-settings-3.log) | 九族主题及 382 个 override 已对齐；Codex flatChrome、Claude 角色字体、720-DIP 会话轴、卡片／气泡／侧栏几何、主题两半与卡片操作已实现并通过回归。渐变与设置玻璃完整回归及有界缓存开销已测，run6 最新源码六页实测含两 skin 的同 owner 串行 Host 设置、720轴／字体角色及后续开关／滚动占位／侧栏／空会话入口修复；与最终生产构建573个共同输入hash一致。全页像素合同未完成 | 代码 + 完整包／定向回归 + 六页真实采集；未认证全像素 |
| 21 | 更新 | 部分 | `internal/desktop/update/`、`internal/ui/chrome/updateprogress.go`、`cmd/whaleisle/native_releases.go`、`update_journal.go` | 检查/下载/增量/journal/attention 已实现；安装包未签名、未上传，catalog HTTPS 仅目标位 | 代码 + 单测 + 隔离安装事务；签名/发布外部阻塞 |
| 22 | 多工作区 | 已实现 | `internal/ui/workspacepicker/`、`internal/store/session/workspace_navigation.go`、`workspace_groups.go`、`workspace_preferences.go` | 未发现功能级差异；注册、重命名、移除、分组与恢复均走 Host 身份 | 代码 + 单测 |
| 23 | 导入 / 导出 | 已实现 | `internal/desktop/importer/`、`internal/ui/frame/archive_undo.go`、`internal/store/usage`（统计导出）、`cmd/whaleisle/import_flow.go` | 未发现功能级差异；Electron 数据导入为事务化应用，失败保留原状态 | 代码 + 单测 + 隔离安装/导入证据 |
| 24 | 远程 Host / SSH | 已实现 | `internal/desktop/sshremote/`（known_hosts 钉扎）、`internal/store/remote/`、`internal/desktop/phoneremote/git_*.go` | 未发现功能级差异；远程 shell/git 走 daemon 路由，需真实 SSH 目标做端到端 | 代码 + 单测；真实网络未验 |
| 25 | 崩溃 / 单实例 / 关闭保护 | 已实现 | `cmd/whaleisle/crash_report.go`、`third_party/mygo/single_instance.go`、`internal/ui/chrome/closing.go`、`internal/desktop/taskguard/` | 未发现功能级差异；renderer 崩溃源在 Go 架构中不存在，由 web-boot 覆盖 | 代码 + 单测 + 历史实机 |

### 规模

| 状态 | 数量 |
| --- | --- |
| 总项 | 25 |
| 已实现 | 20 |
| 部分 | 5（#5、#8、#9、#11、#21） |
| 缺失 | 0 |
| 整项无法验证 | 0 |
| 存在未验证边界的项 | 8（#3、#5、#8、#9、#11、#13、#21、#25） |

> 计数口径：以「功能面是否可用」判状态，以「是否完成本环境验收」单列边界；不把缺真机/证书的项误记为缺失。

## 3. 本轮修复

### 3.1 非文本预览的「跳转到行…」是无效入口（已修）

- 现象：图片、音频/视频、PDF、表格分页预览的标题栏右键菜单仍提供「跳转到行…」。接受后只写入 `revealLine`，这些预览没有行模型，界面无任何变化。
- 修复：新增 `previewSupportsLineNavigation`，仅在异步文本路径（且非图片/媒体）上生成菜单项；已打开的跳转行也做同一门禁。
- 文件：`internal/ui/files/preview_toolbar.go`
- 验证：`go test ./internal/ui/files -run '^TestPreviewLineNavigationIsOfferedOnlyForText$' -count=1`；同一测试同时断言 `a.txt`/`a.md` 有菜单项，`a.png`/`a.pdf`/`a.xlsx`/`a.mp4` 没有。
- 证据：`tmp/parity-fix-check.log`（PASS）。

### 3.2 聊天消息右键菜单缺少复制、无引用回调时整菜单消失（已修）

- 现象：消息右键菜单此前只在 `OnQuoteMessage != nil` 时创建，且只有「引用」；不接线引用回调时菜单完全不存在，键盘/右键用户无法复制消息。
- 修复：改为 `messageActionMenu`，可复制消息始终提供「复制」，引用仅在回调存在时出现；复制走上下文服务，与悬浮复制按钮同源。
- 文件：`internal/ui/chat/message_actions.go`、`internal/ui/chat/user_message.go`
- 验证：`go test ./internal/ui/chat -run '^TestMessageContextMenuCopiesWithoutQuoteHandler$' -count=1`（无引用回调时菜单含 Copy、不含 Quote、剪贴板为消息正文），并回归既有 `TestAssistantTailOneCompactRowKeepsLifecycleTextAndBoundary`。
- 证据：`tmp/parity-fix-check.log`（PASS）。

### 3.3 S27/S28 的早期局部断言与 task62 源码纠正

早期 [remaining-gaps](remaining-gaps-20261026.md) 第 11 项列「composer 772 vs 774.4、导航基线 1.7 DIP、hero 纵向 seat 未定义」。当时以 [chat_chrome_residual_test.go](../../internal/ui/frame/chat_chrome_residual_test.go) 的 774.4-DIP 卡宽、22-DIP 导航行盒及 250-DIP hero 栈等本地断言通过为依据，曾将整项记为闭环，证据仍保留在 [旧局部测试日志](../../tmp/parity-chrome-check.log)。

这个整体闭环结论已撤回：实际安装版 0.3.10 源码表明，当时 155.34-DIP composer 卡和 250-DIP hero 栈含补偿尺寸，本地断言通过不能证明与参考相同。task62 已改回来源定义的 **114-DIP 卡片**（8 顶部＋52 输入区＋12 gap＋42 工具栏），**208-DIP 自然视觉栈**及其内部 **32-DIP 底部 padding**，移除额外 41.34 顶部／38 底部补偿。其他仍有独立来源的横向轴与导航结论不凭这次纠正一并作废。

新 composer 定向测试通过 4.157s；随后 frame 首次执行仍命中旧 32.67／3.33 inset 断言，这份失败保留在 [verify.log](../../tmp/composer-hero-source-20261027/verify.log)。按来源更新该旧断言后，frame 通过 7.469s，见 [最终 frame 验证](../../tmp/composer-hero-source-20261027/verify-frame.log)，覆盖 864→920 高度时整体移动 28 DIP 等关系。源码、来源提取和前后差分保留在 [task62 证据目录](../../tmp/composer-hero-source-20261027/)。这确立来源尺寸与定向行为；最新整页 actual／diff 和全状态双签仍待执行，不以几何单测替代整体像素验收。

### 3.4 冷启动分段计时的清单描述已过时（文档更正）

`remaining-gaps` 第 4 项称「无埋点解释」。当前 `internal/startuptrace` 已提供 opt-in 分段（`runtime.validate`、`runtime.profile_prepare`、`runtime.launch_resolve`、`host.plan`、`host.spawned`、`host.handshake`、`host.ready` 等，见 `internal/harness/prepare.go`、`internal/harness/supervisor.go`）。仍缺的是**用这套埋点对 112.548s 冷启动重采一份 receipt**，而不是代码缺少计时点。该项保留待办，但证据描述已修正。

## 4. 未验证边界（不得据此报通过）

| # | 边界 | 原因 |
| --- | --- | --- |
| 1 | Android 真机、mDNS/LAN 发现、TLS/SSH、relay 与断线重连 | 无真机/第二台网络设备；隔离的本机 loopback 测试通过，不代表这些设备与网络场景已验收（见 §5） |
| 2 | macOS Keychain ACL/TCC/音频设备/Apple SDK | 无 macOS 硬件；仅交叉构建与注入测试 |
| 3 | Windows 通知中心真实点击与视觉模糊 | 需要安装态与真实系统通知中心；不得用 API 读回替代截图/点击 |
| 4 | 代码签名与发布 | 无证书、未上传；`Get-AuthenticodeSignature = NotSigned` |
| 5 | 桌宠 OEM 缩放组合、连续拖拽、长时 FPS/内存/功耗 | 需要外部硬件与长时间采样 |
| 6 | 全应用像素合同（阈值 0、控件×状态穷举、双签） | 合同 [acceptance-ui-parity.md](../acceptance-ui-parity.md) 仍零双签；后续已取得实际安装版同尺基线和自有原生 GUI 证据，但完整成对 actual／diff、全状态穷举及人工双签未完成 |
| 7 | 插件任意槽位／React 控件的完整等价 | P2 实际主入口双轮尾显示／点击及普通轮尾定向滚动已证；后续原生生产行输入路径完整Unicode／中文预编辑取消提交，真实3200-CSS-px长行裁片及显式portal可信点击均已通过。末轮使用真实main创建的单owner与测试专用native单视口投影，不冒称原主List物理输入、系统IME、任意rich控件或全客户端像素，tool／composer仍需原owner挂载 |
| 8 | 原账号长会话、受控帧延迟与完整输入响应 | 已有经生产 mainView 的 2,000 行／400 卡原生 run02／04／06、profile03／05 及实际 HWND 草稿 Q；native06 p50／p95=21.272／30.2765ms，仍有 CPU／分配成本，且与 peer WebView2 采图输入重叠，非独占或精确 A/B。CPU phase／present-call 不等于 DWM／FPS，Q 冒烟不等于键盘到显示延迟；见 [延迟诊断](ui-latency-diagnosis-20261026.md) |

## 5. 测试结果

以下是最初一轮功能修复的隐藏运行器结果，不是当前源码全部后续任务的最新大包验收（日志在 `tmp/`，后续证据另列）：

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/ui/files ./internal/ui/chat ./internal/ui/frame -count=1 -timeout=10m` | PASS（files 6.235s / chat 61.857s / frame 9.419s） |
| `go test ./cmd/whaleisle ./internal/desktop/... -count=1 -timeout=10m` | 除 `internal/desktop/phoneremote` 的 `TestLiveLanServeSPAHealthAndPairing` 外全部 PASS；该失败在**改动前基线**同样存在 |
| 失败复现 | `tmp/parity-baseline.log`（改动前）与 `tmp/parity-gate.log`（改动后）均为 `runtime stopped before ready: status="offline"`，约 37–43s；原始错误未记录，原因不能确定。后续隔离结果和修复见下方 |

最初该轮未执行 Electron 启动、真机、安装包重打、通知中心点击或像素合同双签。后续已实际启动 fresh 自有 Electron／Host 并取得同一安装基线的有效截图与 DOM，不能继续概括为「本轮从未启动 Electron」；后续启动不追认旧失败图，具体有效／排除项见下方。真机／发布／像素合同等边界仍按 §4 分别记录。

### 2026-10-27 后续复核

- 实际安装版 0.3.10 以原可执行文件、fresh 自有 profile／workspace 启动，取得 1440×864 DIP／2160×1296 px／DPI 1.5／zoom 1 的浅色 zh-CN 同尺基线；这是请求 dark 后观察到的实际 light，不以目录名冒充主题。有效项为一个空桌面 WebContents 图、General／Appearance／Interface 三个完整 PrintWindow client 图及对应 DOM。空桌面 composed-client 图仍缺，名为 final-empty 的两张图实际上仍在 Interface 页面，已排除；初始 1441-DIP 尺寸失败和 guard 失败均保留，不计作通过。自有 Electron PID29608／Host 自然退出，未控制既有用户实例、改写安装文件或发送模型请求。截图不是 DWM 实屏，也没有因此取得完整 paired actual／diff。详见 [有效证据与排除项](../../tmp/electron-parity-fixture-20261027/RESULTS.md)。
- P2 后续沿生产 `mainView → withPluginSessionFrame → 原 turn/end callback → production Owner` 路径，以实际 Host follow、显式激活和一个原 browser owner 显示两个轮尾 `1@2`／`2@5`。定向原生点击后两者均为 `Local 1／Shared 2／Host title: P1 Host shared 2`；[complete.json](../../tmp/plugin-inline-production-p2/main-66dd7b511bad47959870f80b29d1e8e3/captures/complete.json) 为 PASS，PID21856 [cleanup.json](../../tmp/plugin-inline-production-p2/main-66dd7b511bad47959870f80b29d1e8e3/cleanup.json) 确认退出且无 listener。较早 main-eb 驱动清理失败不当作自然退出成功，早期两轮滚动失败也保留。后续长历史含 28 行 user 正文，native／source anchor 定向滚动后的两个普通 112-DIP 原轮尾仍各为 Local 1／Shared 2／Host title 2；[滚动结果](../../tmp/plugin-inline-production-p2/scroll-a6b803ac7fe34d2eaaa4e43894f85939/captures/result.json)记录两个 connected／visible 轮尾及 2160×1296 图尺寸，[完成记录](../../tmp/plugin-inline-production-p2/scroll-a6b803ac7fe34d2eaaa4e43894f85939/captures/complete.json) PASS，[退出记录](../../tmp/plugin-inline-production-p2/scroll-a6b803ac7fe34d2eaaa4e43894f85939/cleanup.json)确认 PID32012 自然退出且无 listener。此证只覆盖普通高度轮尾定向滚动；超过 viewport 的单插件裁片、portals、键盘／IME及整体像素仍待验，task75 后续有独立 16 组 DPI×zoom 和实际 AppRun resize／repaint 证据，见本节 round6 记录；这不使早期插件图变为像素验收。
- task59／64 已完成神经 Z 粒子和逐可见 RAF 缓存呈现，实际 WebGPU＋Anime4K 在 4.226s 内获得 66 张不同 RGBA／66 次推理和 prepare；705 次缓存呈现包含 RAF 与完成回调，不是模型 FPS。输入／内容姿态和命中差为零，矩阵差最大 0.000493128px，自有进程退出且 profile 删除，见 [cadence 结果](../../tmp/pet-cadence-gui/result.json)。task72 随后完成绝对脚点和严格 wake expiry：43 神经＋30 兼容检查、四个脚本套件及 full pet 3.458s 通过；实际 WebGPU／Anime4K [round2 结果](../../tmp/pet-boundary-gui-round2/result.json)在 DPR1.5 下输入／alpha 包络差为零，最大 feet 差 0.001334101 CSS px，普通静止画面的实际 BCR／hit／脚点偏差为零，PID35524 自然退出且 profile 删除。首轮 PID34132 因 tap 动作尚未结束被误称 idle 的失败仍保留；修正夹具等待真正静止，未放宽容差或改写原生产数学。这关闭已知两 CSS px／一 RAF 缺陷，不把它扩大成模型栅格、外部硬件、长期功耗／内存、DWM 或全窗零像素验收。
- task67 依据实际 ui-theme 0.2.1-rc.2 的源码与运行 bundle 核对九族：DeepSeek、Codex、Claude、Midnight、Celadon、Violet、Amber、Paper、Contrast。Codex／Claude 的浅深 seed 及 99＋99＋92＋92=382 个 override 已补，见 [来源与 bundle 相等记录](../../tmp/builtin-theme-parity-20261027/source.json)。后续 task74 已完成 selected-half skin 保留、Codex flatChrome 与 Claude Inter／Source Serif 按角色字体；task79 已完成 720-DIP 轴、输入卡／气泡／侧栏、canvas 单层绘制、默认 hero badge 和原版图标。task77 补齐主题卡 88-DIP preview、独立半边选择、hover／focus actions 和窄窗网格，行为检查 6.195s 通过。task78 设置整包 12.173s 通过；theme／composer／frame／sidebar 完整包及 task85 修正测试逐像素重复取整图后的完整 chat 22.056s 亦通过。来源、哈希及具体日志见上表；这不是新两 skin 实机采集或完整像素验收的替代。
- task68／69／70 在 profile05 定位后分别补 UI 自有精确衍生 memo、同 pass 的 Store／版本校验读取复用和无 context 技能快路。联合 native06 保留相同 2,000 raw 行（400 个 Files、输入 TurnDelivery 为零）→400 投影卡、可见三卡／两个最终回答及真实草稿 Q 断言，未启用 profiler，全部通过；callback p50／p95=21.272／30.2765ms，whole-process 平均 58,823.78 次分配／12,163.707KB，39／100 帧观察到 GC，最大 callback 85.4366ms 保留。peer Host 已就绪后的 WebView2 采图／输入活动与该批重叠，非本方或机器独占，不作单项收益或精确 A/B。run04 的 1008.1365ms p95 和全部慢尾未覆盖，profile05 自启动累计 allocs 不与 native06 每帧数直接相除归因。详见 [延迟诊断](ui-latency-diagnosis-20261026.md)及 [run06 报告](../../tmp/markdown-native-long-run-06-summary.md)；原账号长会话和 DWM／完整键盘延迟仍未在此验证。
- 已安装版桌面布局在小于 1024 DIP 时暂时收起侧栏，且保留标题栏工具簇；小于 768 DIP 时右侧工作区临时全屏。MyGo 已对齐这些断点、标题密度和账户按钮身份，并用实际 `ui.Tester` 回归窗口切换、右栏开关、焦点及草稿。侧栏的竖屏抽屉按所属屏幕方向判断；macOS 仅完成交叉编译和无 GUI 测试。
- 较早阶段，同标签的会话插件实例补了轮次、工具调用或必要实例序号，区分合法轮次 0 与缺失轮次；显式启用后，从唯一匹配的轮尾打开原局部 browser seat。当时尚非原位双尾，定向证据 [freeze.log](../../tmp/plugin-turn-tail-a1/freeze.log) 保留；后续 P2 的单 owner／双原生轮尾实际显示和点击证据见本节上方，不再以旧 seat 限制概括当前全部实现。
- 原版手机远程默认 30 秒的 daemon 就绪期限接近隔离冷启动测得的 29.879 秒；Go 桥接现给予 60 秒，并让停止操作能取消启动。仅配置 relay 为 loopback 的隔离真实运行在修改前通过 SPA、健康检查和配对路由（`tmp/phone-lan-diagnostic-20261027/focused-r4.log`）；修改后通过模拟启动和停止回归。旧失败没有保存底层异常，不能认定其唯一原因。live 测试现在需显式启用并使用自有 loopback relay；Android 真机仍未测试。
- 隔离桌面测试对归档、搜索筛选、重启及工作区状态进行了初次运行和重启验证（`tmp/sidebar-live-acceptance-20261027/execute-probe-09.log`）。集成回归 `tmp/parity-final-integrated-20261027.log`、插件及手机远程组件回归 `tmp/lead-phone-component-20261027.log` 通过；macOS frame 测试只交叉编译（`tmp/parity-cross-frame-20261027.log`）。
- 消息代码块复制保留源制表符（`tmp/markdown-code-copy-final.log`）；新桌宠资料默认优先 THA4，旧省略字段仍按关闭处理，开发构建可显式指定经过固定清单验证的资源（`tmp/pet-neural-check.log`）。修改后在自有 Windows 窗口复测 THA4/Anime4K 首帧及左右朝向同帧参考公式（`tmp/pet-neural-live-20261027/result.json`、`tmp/pet-facing-gui/result.json`）；设备、其他姿态与长期性能并非全量验证。
- 进一步修复聊天投影在虚拟列表前的两处二次扫描：过程分组改为按轮索引，回答操作改为按来源逐次拼接正文。100 组随机旧函数对照、定向语义回归与聊天包测试通过；同进程三次合成基准在 8,000 行下的中位数分别从 167.439→8.448 ms 和 76.908→1.102 ms。回答操作增加约 0.330 MB/次分配；快照深复制未改。完整口径与限制见 `docs/parity/ui-latency-diagnosis-20261026.md`。
- 坐标、文件提及及聊天投影缓存已补精确来源身份，避免等长事件替换、原地修改或元数据更新后复用旧结果；坐标最多保留 16,384 条 / 8 MiB，时间戳与工作流使用独立字段副本。2,000 行工具/工作流热缓存约 0.10–0.12 ms、零分配；复杂元数据仍有安全重建边界。见 `tmp/composer-source-memos-20261027/freeze-check.log`。
- `cmd/whaleisle` 对已确认的用户可见固定文案迁移了 238 个字面量、227 个原源码行至 193 个双语 key，最终范围 84 个生产文件。格式参数、`%w`、权限 sentinel 及真实调用分支回归通过；后续批次 69 份快照做归一化 AST 比对，未把日志、协议或候选数等同缺陷数。见 `tmp/cmd-inline-i18n-20261027/closing.log`、`summary.txt`。
- 当时 round4 整合 `tmp/parity-round4-check-20261027.log` 全部通过（不是后续全部任务的最新大包证明）：44 个 Node body/sleep/direction/renderer 用例、4 个脚本套件，以及 config、dsh、手机远程、插件、pet、session、全部 UI 和 cmd 包。上一轮的 `TestCreateSessionSelectsNewID` 在选择发布后立即断言后台 list 已发出，是异步时序假设；仅测试改用 list-start/响应门和最终通知，连续 50 次及 session 全包通过。原失败日志 `tmp/parity-round3-check-20261027.log` 保留，不重写成通过。
- round4 当时源码 GUI 构建 SHA-256 `9B1FE1DA0F4E11BCA836BB299D4783EE46E88FF8640C755D281E7A556C21FE64` 已直接启动，owned PID 38148 / Host PID 20744；主窗口截图、`last-desktop-start.ok=true`、trace ready 和 loopback 接受均已核对，未打包。新隔离 profile 的冷启动为 49.251s，慢启动尚未解决。证据 `tmp/current-parity-round4-20261027/verification.json`、`main-window.png`、`startup-trace.json`。
- 开发构建使用新的隔离资料目录，本轮不使用真实用户资料。这些结果不代表逐页或像素级对照通过。

### 2026-10-27 round6 已核验证据

- 首启任务63已改为一次性、代次感知的会话／草稿选择，并移除 Ready 前隐式 EnsureWorkspace。Windows Documents 使用 KnownFolder，隔离运行必须通过自有 cordis patch 指定 Documents，不能只依赖 HOME；见 [首启复核](../../tmp/startup-selection-20261027/RESULTS.md)。
- task75 修复 repaint 的物理／逻辑坐标重复换算和浮点取整多一像素。16 组 DPI×zoom 矩阵及实际 Windows AppRun 的隐藏窗口 resize／动画 repaint 验证通过，见 [原生回执](../../tmp/composer-native-scale-20261027/native-proof.json)；CapturePage 是原生渲染回读，不是 DWM/FPS 证据。
- Project 默认关闭、设置提交后准确重启 Host 已闭环。实际生产 mainView 开关 off→on→off，Ready generation 1→3→5，patch 数 0→1→0，两次 Busy，原空会话与两个草稿 marker、HistoryReady／ModelKnown 均保留；47.717s 完成且未发送模型请求。同步 store 通知重入主锁的真实死锁已修复，旧 RED 和后续 GREEN 都保留。见 [实际 lifecycle 回执](../../tmp/project-settings-lifecycle-20261027/actual-run4/receipts/complete.json)。
- task82 导航按实际 Whale overlay 配置门控，缺配置或关闭时移除鲸鱼娘入口。真实 Prepare 的八组组合和四启动阶段与集成导航／旧 rebind 回归通过，见 [源码及验证说明](../../tmp/assistant-navigation-gate-20261027/review.md)。SkipUserPlugins 或 disabled 列表不会移除该 immutable builtin，未另造不符合实际 consumer 的 gate。
- 原生 run3 已取得同尺空桌面及三个设置页的 native-render／PrintWindow client 图。采集进程自然消失、listener 为零、stdout PASS、stderr 为零；PowerShell 退出码不可读仍保持 null。原误判保留，补充 [恢复回执](../../tmp/parity-round6-visual-20261027-run3/recovered-receipt.json) 明确哪些退出事实来自历史控制流推断。未缩放、未遮罩、阈值零的 [四页 diff](../../tmp/parity-round6-diff-20261027-run3/diff-summary.json) 已生成；91.21%–99.90% 是动态背景、资料与采图路径共同造成的像素观测，不能作为缺陷比例或验收 PASS。上述图早于 task77／78／79／81 最终修复；后续 run4 六页真实采集已完成，见下项。
- run4 已用固定 collector `8BC3D432…` 在仓库外独立 profile 取得空桌面、通用／外观／界面设置、Codex／Claude 六页的 native-render 与 PrintWindow client 双图，均为 2160×1296／144 DPI。见 [完成回执](../../../WhaleIsle-MyGo-visual-round6-run4/captures/complete.json) 与 [清理回执](../../../WhaleIsle-MyGo-visual-round6-run4/cleanup.json)：PID 25264 自然退出、listener 为零、stdout PASS、stderr 为零，退出码仍诚实记录 null。双 skin 保持同一个空会话，真实 Host 同 owner 的半边写入依次 r2 codex/deepseek→r3 codex/codex→r4 claude/codex→r5 claude/claude，每步 Flush 且已接受；Claude 的 Inter／Source Serif 4 为绘制回退，未伪造原始字体偏好，代码字体独立，内容轴 720／卡片 752×114 DIP。未遮罩、未缩放、阈值零的 [run4 四页差分](../../tmp/parity-round6-diff-20261027-run4/diff-summary.json) 为 85.65%–96.50%，仅作像素观测，不作为缺陷比例或验收通过。run4 是历史固定源码证据，不包含之后的插件缓存／输入、Git89、侧栏92、设置93、header94 修复；这些增量须在最终源码构建中另行验证。
- run4 后三个已确证增量通过回归：Git89 的 branch/action 标签按各自 hover／键盘焦点／菜单独立展开，离开整簇重置，保留原200ms／ReduceMotion与所有操作；完整 chrome 4.250s、最终真实按钮／菜单／键盘 fixture 2.588s，见 [Git 验证](../../tmp/git-titlebar-labels-20261027/review.md)。侧栏92恢复 tertiary 分区标题、默认工作区14／400／20及归档12／500／18，保留两 skin 13字体与空归档展开；独立来源绘制逐像素、归档交互2.175s、完整 sidebar3.988s，见 [侧栏日志](../../tmp/sidebar-section-typography-20261027/verify-01.log)。设置93区分 labelled36×20品牌色与 input44×28命中区域／36×20轨道，并只在实际溢出时预留5DIP gutter；完整 settings11.211s，见 [设置日志](../../tmp/composer-settings-followup-20261027/verify-2.log)。
- header94按原生命周期隐藏空会话的页顶操作，保留原 leading／corner；会话插件显式操作移到当前会话行菜单，旧菜单执行时重新核对目标，外侧导航只绑定原 owner 操作、不重复采图。完整 frame12.099s与cmd导航定向4.966s通过，见 [回执](../../tmp/conversation-blank-header-20261027-freeze.json)。四个完整 slot 名称的旧测试按钮挤占800-DIP测试宽度导致原失败保留；fixture改为28×28语义按钮并加入leading顺序，未放宽几何界限。run5 实际采集通过 blank caption／launcher 隐藏检查，但发现空会话原先不挂行菜单；旧单测未设 Blank=true，因而漏检入口。已为当前空会话单独保留插件 Open／Stop 菜单，仍不增加一般行操作和时间。补充 Blank=true 测试，完整 sidebar 3.524s，见 [补测及冻结](../../tmp/conversation-blank-sidebar-20261027-freeze.json)。run5 失败原始记录保留且自然退出／listener0，不计六页通过。随后 run6 以修正后的完整源码 collector `CDFFE7CA…` 取得六页12图，同尺寸／144 DPI、complete PASS、自然退出／listener0／stderr0，见 [完成记录](../../../WhaleIsle-MyGo-visual-round6-run6/captures/complete.json) 和 [退出记录](../../../WhaleIsle-MyGo-visual-round6-run6/cleanup.json)。实际 [空页入口](../../../WhaleIsle-MyGo-visual-round6-run6/captures/empty-desktop.json) 同时证明 blank caption、launcher缺席、当前空会话菜单只有「会话插件」、browser owner仍未创建；双 skin 也无多余页顶操作，真实 Host 仍同 owner 串行四次写入。六图均已人工查看，设置开关分支、侧栏字体／卡片及两 skin 布局可见。同尺、不缩放、不遮罩、阈值零的 [run6 四页差分](../../tmp/parity-round6-diff-20261027-run6/diff-summary.json) 已生成；85.25%–96.38%／MAE7.998–8.662只报告像素观测，动态相位、资料和空页采图路径不一致，不能当缺陷比例或通过分数。不可读退出码继续保留 null，未宣称 DWM 实屏或全像素一致。
- task77／78／79／81／82／85 已冻结。task81 的完整 chrome／cmd 相关回归及 Darwin／Linux arm64 CGO0 构建通过，见 [收尾日志](../../tmp/ambient-gradient-parity-20261027/final-check-fixed.log)。保持原 Gaussian 核的优化测得约 4.662ms／新背景帧、2.708ms／设置玻璃 miss、260ns／cache hit（0B／0alloc），见 [实测](../../tmp/ambient-gradient-parity-20261027/benchmark-convolution.log)；该次 benchmark 在末次 ceil→floor 预算边界修复前，不能冒充最终小一行栅格的重新计时。128 长边栅格、3σ 截断和 30Hz 相位均属已披露近似，设置滤镜重建 gradient／canvas／modal mask，不包括任意前景控件，未认证浏览器逐像素相等。插件原始控件六步 Unicode／中文预编辑取消与提交、可信收据及提交后的 fresh bitmap 已取得，见 [严格输入六步](../../tmp/plugin-inline-production-p2/text-1484abb25d2e417885775018d04a8eb3/captures/steps.json)；native row 路径、候选 caret 坐标不等同于全主窗口物理输入或系统 IME 候选窗。full／visible 裁片、uniform bitmap、原行高度／source-offset scroll 及显式 owner portal 的生产 wire 已完成且相关三包回归通过。末轮 [实际功能收据](../../tmp/plugin-inline-production-p2/text-81de49b8421947ad892cbc36dce249f3/captures/integrated-original-owner.json) 为原始高行按钮／portal各click1、全部可信事件、两个原节点各mount1／unmount{}／sameOriginalNodes=true；fullH3200／Y−1484.67与visibleY71.67／H662.33，portal打开图及关闭后的fresh图已查看，finalBitmapCurrent=true，[complete](../../tmp/plugin-inline-production-p2/text-81de49b8421947ad892cbc36dce249f3/captures/complete.json) PASS、[cleanup](../../tmp/plugin-inline-production-p2/text-81de49b8421947ad892cbc36dce249f3/cleanup.json) PID9076正常退出／无listener。前一功能已通过但测试收尾nil panic／白图保留，不作为整轮成功。最终整合批次70个 Node用例／4脚本及17个Go包通过；cmd包旧 Git标签测试的「Commit and push」与原版现有「Commit & push」字面不符，仅改这一处测试值；24组完整frame验证21.813s、完整cmd28.875s通过，原严格review owner／无操作／Host调用断言保持，见 [末轮整包验证](../../tmp/git-titlebar-frame-final-20261027/verify.log)。因此18个Go包、70个Node用例和4脚本均已通过，首次失败日志保留。最终 round6 生产程序已 CGO0／windowsgui 构建，SHA `22B04F69EB747ECBBBC214AE105DA2B21F5D2353A20EF987848A51BDC07DC962`；637清单条目／576唯一输入在构建前后稳定，与run6采集的573个共同生产输入hash一致，见 [构建来源复核](../../tmp/parity-round6-production-provenance.json)。插件最终实测已自然退出。该程序已在外部fresh自有profile直接启动，PID36336／HWND4982914，1个Bundled Node Host PID25236／127.0.0.1:58769，启动trace4458.959ms Ready，源码／缺失文件／exehash与启动记录均复核一致；见 [启动与只读窗口验证](../../../WhaleIsle-MyGo-latest-round6-20261027/verification.json)。该时间仅为此次已有runtime缓存环境测值，不替代冷启动对照。实际client为2160×1295／144DPI、outer2160×1296，未强制resize成采集viewport；主窗图已查看，无欢迎页。旧自有PID37320在新实例verified Ready后仅用CloseMainWindow正常退出，未force，见 [替换收尾](../../../WhaleIsle-MyGo-latest-round6-20261027/previous-owned-window-lifecycle.json)；原安装应用PID6308与真实资料未触碰。未打包。
