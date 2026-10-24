# 原生 ↔ MyGo 差距矩阵（2026-10-25 刷新）

HEAD：`9290c9b9`；参考：Electron 鲸屿 0.3.9（安装版只读）+ `reference/electron` + 安装版 rc.2 runtime。
本文件与 `gap-matrix-20261025.json` 同源生成；2026-10-24 版（`gap-matrix-20261024.*`）保留为历史快照。

## 0. 口径

- 以 docs/status.md:223-352（round126-159 登记）、round127-157 文档、docs/parity/chrome-metrics-20261024.md 为闭环证据来源
- 每条闭环结论都回读 MyGo 代码/脚本/实机证据文件（window-geometry-pass273/274.json、harness-start-pass275/276/278/279.json、native-web-shot.mjs、round 文档的 file:line）
- 新 sweep：枚举 reference/electron/src/main 的 284 个模块，按 24 个能力面归并，逐个在 internal/、cmd/ 中找对应实现；找不到即记为 open 并给验证方法
- 状态三档：closed（有代码+轮次证据或实机证据）/ open（无实现、实现不同、或未取证）/ measurement-error（原矩阵判定本身错误）
- 不把「文档自述」当闭环证据：凡只有文档没有代码或实机证据的，记 open/unverified

## 1. 计数

| 项 | 数量 |
| --- | --- |
| 行数 | 89 |
| closed | 63 |
| open | 25 |
| measurement-error | 1 |
| 高严重度 | 3 |
| 中严重度 | 9 |
| 低严重度 | 13 |
| info | 64 |

按状态×严重度：

| 状态 | high | medium | low | info |
| --- | --- | --- | --- | --- |
| open | 3 | 9 | 13 | 0 |
| closed | 0 | 0 | 0 | 63 |
| measurement-error | 0 | 0 | 0 | 1 |

## 2. 本轮确认闭环的要点

- round127-132 把 ToolDetails 通用字段族与 agent/team、inspection（cordis/workflow/ralph/session）、markdown/location/todo、terminal_list/lsp 全部补齐（G14/G16/G17/G18/G20）
- round129 关闭遮罩（G40）落地；round131 崩溃报告（G41）与媒体权限（G42）落地
- round133/136/148 把动效 token 校正为 popover 160ms/4px、overlay 进 200 出 100、scale .96、DesignEase（G50），并确认 G01 是测量错误
- round139/147/151/153 启动器成为独立 1060×660 窗口、图标与欢迎窗指标对齐（G04/G29）
- round143 修主窗首帧自锁、round145 修单实例陈旧 socket、round146/150 跑通 start→boot→ready
- round152/155 打通原生 Web UI 参考管线（token URL + CDP 截图 + DOM rect），round156 把 composer 收到 114 DIP
- round157/158 远程/中继矩阵：G3 relay token 注入、G2 LAN 配对就绪修复
- round159 桌宠可见性根因（Hidden:true 导致自定义 scheme 中止）修复，实机 240×438 DIP

## A 主壳

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G01 | 标题带 48 DIP / 内容 y=12..44 | chrome.js:9 TITLEBAR_HEIGHT=48 | chrome/titlebar.go；内容带 y=12..44 | measurement-error | info | slice-chrome | round133/136 | layout | 原矩阵按 12 vs 48 记差异，实为带高本就 48、按钮内缩 12；round136 已坐 y=12..44 | S |
| G02 | 主窗 1440×920 / 96×640 min / 20 圆角 / 透明 | window.js:209-213；silhouette test:14-28 | main.go:2193-2204；探针 2160×1380 px = 1440×920 DIP | closed | info | slice-chrome | round136/138 | layout | window-geometry-pass273/274.json + round138 换算 | S |
| G03 | 原生菜单栏隐藏（autoHideMenuBar/setMenuBarVisibility） | chrome.js:48,65-70 | application_menu.go:34 SetMenu；无可见性调用 | open | low | slice-chrome | round134 自述仍未断言 | layout | 需 Windows 实机看菜单栏是否下压内容；headless 不可断言 | S |
| G04 | 启动器独立 1060×660 窗口 | window.js:906-909 | cmd/whaleisle 启动器窗口；探针 1590×990 px | closed | info | slice-launcher | round139/147/151/153 | layout | window-geometry-pass273；round151 图标、round147 首页指标 25/26 | S |
| G05 | Boot 揭幕 600ms / 切片 | renderer/boot.html,boot-reveal.css | internal/ui/boot | closed | info | slice-launcher | round140/148 | animation | round140 600ms+DesignEase；round148 overlay 进 200/出 100 | S |
| G06 | 三轨几何 | ui-layout columns.ts | frame/frame.go:56-67 | closed | info | slice-chrome | - | layout | ui.Tester 几何断言 | S |
| G07 | 768/1024 断点 | ui-layout columns.ts | frame/sidebar_columns.go:12-15 | closed | info | slice-chrome | - | layout | narrow_layout_test | S |
| G08 | 会话列宽/rails | ui-conversation | frame/conversation_width.go | closed | info | slice-chrome | round152/156 | layout | composer 宽 772 vs 774.4 为残项（见 S38） | S |
| G09 | 右侧/底部工作区（文件树/预览/终端/浮动） | ui-sidebar-right/ui-dockkit | frame/workspace_*.go | open | medium | slice-workspace | - | layout | 逐面板 1440 截图 + 停靠/浮动/最小化三态未系统化 | M |
| G10 | 标题栏右侧控件组逐图标 | ui-titlebar + chrome 注入 | chrome/workspace_trailing*.go | open | low | slice-chrome | round133/138 只覆盖 caption 按钮 | layout | 逐图标命中区/顺序/悬停未逐项 | M |
| G11 | 弹层材质 28/200ms | Modal.module.css:19-49 | chrome/dialog_surface.go | closed | info | slice-chrome | final-convergence D34 | animation | 像素采样回归 | S |
| G12 | 设置模态玻璃 | Modal.module.css:70-78 | settings/modal.go | closed | info | slice-settings | - | layout | 设置页截图 | S |
## B 工具卡

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G13 | 工具行 + Inspect 药丸 | ToolRow.tsx:93-96,344-348 | chat/toolcard.go:201-202 | closed | info | slice-chat | - | function | 悬停/跳转回归 | S |
| G14 | ToolDetails 字段族 lines/groups/unchanged/markdown/location/status/change | ToolDetails.tsx:10-26,88-136 | chat/tool_details_entries.go + tool_details_model.go | closed | info | slice-chat | round127/132 | missing | entries 文件 + 条目测试 | M |
| G15 | goal/schedule 详情族 | details-card-model.ts:49,101 | tool_details_model.go:82-372 | closed | info | slice-chat | round123/125 | function | fixture | S |
| G16 | agent/team 控制收据族 | control-details-model.ts:102-194 | round128 实现（tool_details_model.go 扩展） | closed | info | slice-chat | round128 | missing | list_agents/spawn_teammate/team_task_* 卡 | M |
| G17 | session/cordis 检查族 | inspection-details-model.ts:19-33,92-173 | round130 实现 | closed | info | slice-chat | round130 | missing | cordis_inspect_*/session_* 卡 | M |
| G18 | workflow/ralph 族 | inspection-details-model.ts:36-61 | round130 实现 | closed | info | slice-chat | round130 | missing | workflow/ralph 报告卡 | M |
| G19 | read/diff/search/web/image/terminal/run_code 卡 | 各 card-model | chat/tool_*_model.go | closed | info | slice-chat | round114-122 | function | 逐族 fixture | S |
| G20 | todo/plan/ask 卡 | todo-diff-model/ask-question-card | round135/137 + planboard.go | closed | info | slice-chat | round135/137 | function | 三态 + 未作答标签 | M |
| G21 | Inspect/trajectory 面板 | ToolCallTree.tsx + ui-trajectory | frame/session_inspector_*.go | open | low | slice-chat | round102-113 | function | 原生面板在，逐项对照未完成的嵌套几何残项 | M |
| G22 | 子代理/工作流成员/会话祖先 | ui-subagent/ui-workflow-run | chat/workflow*.go, session_ancestry.go | closed | info | slice-chat | round128/130 | function | 卡片展开/成员行 | S |
| G23 | 图片灯箱/画廊/附件 | ui-attachment | chat/child_image_gallery.go 等 | closed | info | slice-chat | - | layout | 打开/关闭/缩放 | S |
## C 设置

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G24 | 设置分区与页面 | ui-settings* | settings/sections.go:14-31 + 各页 | closed | info | slice-settings | round142/144 | function | 19 SectionID 全在 | S |
| G25 | 通用页行结构 | final-convergence D1-D7 | settings/preference_row.go | closed | info | slice-settings | round2 | layout | 800 DIP 模态 | S |
| G26 | 快捷键弹窗 | D8-D17 | settings/shortcuts*.go | closed | info | slice-settings | round3-5 | layout | 四类关闭手势 | S |
| G27 | 各专页深度（模型/账号/用量/远程/市场/插件/技能/MCP/预设/关于/宠物） | ui-settings-* | settings/*.go | open | medium | slice-settings | round142/144 仅文案 | function | 控件×状态未逐页；安装类会改环境 | L |
| G28 | 字体基线/滚动条/选择复制/tooltip | ui-theme tokens/web-styling | widgets/markdown、chrome | open | medium | slice-chrome | - | animation | 同字号像素宽/滚动条宽/选区色/tooltip 延迟 | M |
## D 外围窗

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G29 | Welcome 600×700 网格 | welcome-window.js:26-33 + welcome-layout.css | welcome_window.go + ui/welcome | closed | info | slice-launcher | round126/147/153 | layout | 17 项指标测试 | S |
| G30 | WhaleBridge 1120×780 | whalebridge-window.js:17 | whalebridge_window.go:100 | closed | info | slice-shell | round126 | layout | 地址白名单 | S |
| G31a | 桌宠窗口可见性与几何 240×438 | desktop-live2d.js 240×260 足迹 | pet/companion.go；实机 360×657 px | closed | info | slice-pet | round154/159 | layout | 实机截图 + hit-region | S |
| G31b | 桌宠渲染层（Live2D + ORT/anime4k vs canvas rig） | renderer/pet-live2d/（ort/,anime4k.js,states/,avatar/,rig/） | pet/assets/pet-rig.js（纯 canvas） | open | medium | slice-pet | round154/159 明确保留 | function | 逐帧对照模型分层/物理/锐度 | L |
| G32 | 桌宠行为族（成长/统计/漫步/物理/喂食/记忆） | pet-growth/stats/wander/physics/chat | pet/growth*.go,behavior.go,motion.go,feed.go,neural.go | open | medium | slice-pet | - | function | 成长曲线/漫步路径/物理回弹未逐项 | M |
| G33 | 桌宠快捷聊天窗 | pet-chat.js | pet/quickchat.go:91-93（420×580） | open | low | slice-pet | - | layout | native 尺寸/贴边未取证 | S |
| G34 | 手机 PR 复核窗 | dshd-remote/remote | phone_pr_review.go:51（660×720） | open | low | slice-remote | - | layout | 未与 native 窗口参数对照 | S |
## E 系统

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G35 | 托盘菜单项序/文案 | tray-menu.js:17-33 | desktop_tray_menu.go:29-36 | closed | info | slice-shell | - | copy | 中英逐项 | S |
| G36 | 托盘图标 24 / 窗口图标 48 | tray.js:20 | main.go:7511 scalePNG(data,32)→round141 24+48 | closed | info | slice-shell | round141 | layout | 图标尺寸探针 | S |
| G37 | 应用菜单逐项 + keycap | menu.js:5-113 | application_menu.go:34,125,251 | closed | info | slice-shell | round134 | function | 命令 ID/文案/keycap 只取已接受 revision | M |
| G38 | 快捷键服务（录制/导入/冲突/CAS） | shortcuts.js:31,51,102,231 | internal/shortcuts/* + shortcutprefs | closed | info | slice-shell | round134/145 | function | 改键→菜单 keycap 同步 | M |
| G39 | 通知呈现（系统通知中心 vs 托盘气泡） | system-notifications.js:10-15 | system_notifications.go（托盘气泡） | open | medium | slice-shell | round126 明确平台差 | function | 通知中心条目/点击/AUMID 未对齐 | M |
| G40 | 关闭遮罩 24 圆角/0.85s spinner | closing-overlay.js:1-52 | chrome/closing.go + finalizeQuit（500ms 预算） | closed | info | slice-shell | round129 | missing | 浅/深渲染图；spinner 静态环为残项 | M |
| G41 | 崩溃报告四源/保留 10 | crash-report.js:19,106,119 | crash_report.go（host/main/web-boot；无 renderer 源） | closed | info | slice-shell | round131 | missing | 文件名/保留/权限/失败不掩盖 | M |
| G42 | 媒体权限（主帧/loopback/audio） | media-permissions.js:32-39 | webview/permissions.go + host.go | closed | info | slice-shell | round131 | function | camera/geo 拒绝；帧/mediaType 无信息为残项 | S |
| G43 | 任务保护/关闭到托盘 | task-protection.js,close-behavior.js | taskguard + task_protection.go | closed | info | slice-shell | - | function | running 必须确认 | S |
## F 远程

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G44 | 手机远程运行时（G1: 无 mobile runtime） | mobile-web-server.js:7 | phoneremote/runtime.go + bridge.cjs | open | high | slice-remote | round157 G1 | function | 需真实 daemon/手机；离线不可验 | M |
| G45 | relay（G3 token 已修；G5 LAN 发现 / G6 TLS/SSH） | relay-client.js:5,294 | bridge.cjs relayConnected；G3 注入 daemon 配置 | open | high | slice-remote | round157/158 | function | LAN 发现与 TLS/SSH 需真实网络 | M |
| G46 | dsh-remote 注册表/文件（remote shell/git dispatch） | dsh-remote-desktop.js,remote-shell.js | store/remote + sshremote | open | medium | slice-remote | round157 | function | remote shell/git dispatch 未接线取证 | M |
| G47 | task-control peer | task-control-peer.js:52-60 | taskcontrol/control.go | closed | info | slice-remote | - | function | peer 文件 schemaVersion 1 | S |
| G48 | 在线壁纸目录 | wallpaper-catalog.js:1,24 | wallpapers/catalog.go:40-42 | closed | info | slice-settings | - | function | Bing/Wallhaven 源 | S |
| G49 | 更新进度/journal | update-*.js | chrome/updateprogress.go + update/* | closed | info | slice-shell | round126 | function | checking/downloading/staged | S |
## G 动效

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G50 | 动效 token（popover 160/4、overlay 200/8/100/scale .96） | base.css:16-22, motion.css:30,44 | chrome/motion.go + widgets DesignEase + Motion.Scale/ExitDuration | closed | info | slice-chrome | round133/136/148 | animation | 进 200/出 100、scale .96、popover 160/4 | S |
| G51 | FlipText 400ms | motion.css:83-94 | composer/model_flip_*.go | closed | info | slice-chat | round142 | animation | 切换模型/权限翻转 | S |
## G 文案

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G52 | i18n 漂移（91 key / 46 处设置漂移已修；cmd 254 处 inline 未审） | vendor locales + settings copy | i18n/catalog_*.go；settings providerCopy | open | medium | slice-i18n | round142/144 | copy | cmd/ inline 文案全量扫描未做 | L |
| G53 | 详情卡 STATUS_KEYS | ToolDetails badge | chat/tool_details_badge.go | closed | info | slice-chat | round126 | copy | 四态+未知原词 | S |
| G54 | 文档口径（旧 11 缺失/7 空壳清单） | reference src | docs/status.md 已按 round126-159 更新 | closed | info | lead | status.md 2026-10-25 | function | 按本矩阵复核 | S |
## S 新增

| ID | 表面 | 原生证据 | MyGo 证据 | 状态 | 严重度 | owner | 轮次/证据 | 类型 | 验证 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S01 | harness 页面文件下载（文件名消毒 + 落盘） | index.js:1359 will-download + download-path.js:1-20 | 主页面无 OnWillDownload；仅 webview/host.go:365 与 preview helper 阻止 | open | medium | slice-shell | - | function | 在 harness 页触发一次下载看落盘/命名；离线可用 data: 链接验证 | S |
| S02 | 插件安装前磁盘空间预检 | install-space.js:6-11（256MiB 保留/statfs） | 无 statfs 预检 | open | low | slice-market | - | function | 小容量卷安装看错误文案 | S |
| S03 | DevTools 窗口级 Ctrl+Shift+I + 打包门控 | devtools-shortcut.js:1-14 | 菜单 RoleToggleDevTools（application_menu.go:251）+ config OpenDevTools；无快捷键 | open | low | slice-shell | - | function | 打包态按 Ctrl+Shift+I 是否开/是否需 openDevTools | S |
| S04 | 浏览器预览缩放档位 | preview-zoom.js:4-6（Chrome 17 档） | browserpreview 无 zoom（仅 PDF zoom） | open | low | slice-preview | - | function | 预览页 Ctrl+±/菜单缩放 | S |
| S05 | 插件崩溃取证（OOM/EACCES 分类） | plugin-forensics.js:1-12 | launcher pluginRecovery 状态（部分） | open | low | slice-launcher | - | function | 注入插件崩溃看归类文案 | M |
| S06 | 首启「预览版说明」600×344 | (round152 新增，native 无同款) | round152 实现 + goPreviewNoticeAck | closed | info | slice-chat | round152 | function | 首启截图 + 持久化 | S |
| S07 | 单实例陈旧 socket 接管 | （架构差异） | round145 single_instance 修复 | closed | info | slice-shell | round145 | function | 强杀后重启可用 | S |
| S08 | harness start→ready / runtime staging | harness-controller.js | round146/150 rc.2 staging + 就绪 | closed | info | slice-runtime | round146/150 | function | harness-start-pass275/278/279.json ready=True | M |
| S09 | 主窗首帧自锁（Responding=False） | （缺陷） | round143 Locked 访问器 | closed | info | slice-shell | round143 | function | 探针 Responding=True | S |
| S10 | startup 深链 settings?section= | settings-jump.js:1-25 | deeplink/parse.go:74-84 | closed | info | slice-shell | - | function | whaleisle://settings?section=desktop | S |
| S11 | harness 浏览器会话 cookie 兑换 | harness-browser-auth.js:1-12 | internal/harness/auth.go dsh-auth- cookie | closed | info | slice-runtime | - | function | 一次 token 换 cookie 后加载 | S |
| S12 | 打开 DSH home / 工作区到资源管理器 | open-dsh-home.js | round134 菜单「在资源管理器中打开工作区」 | closed | info | slice-shell | round134 | function | 菜单项存在 + 打开路径 | S |
| S13 | 插件目录链接（junction/symlink） | desktop-plugin-link.js | pluginslots junction_*.go | closed | info | slice-market | - | function | 软链/重装后加载；身份比较(ino/dev)未逐项 | S |
| S14 | 预览注解 / 画中画 | preview-annotation-keyboard.js, preview-pip-*.js | webview/annotation.go + browserpreview/picture_in_picture.go | closed | info | slice-preview | - | function | 标注键盘流 / PiP 帧 | S |
| S15 | HTML 预览隔离与脚本授权 | preview-*.js, preview-guest-protocol | htmlpreview/renderer_service + htmldocument/interactive.go | closed | info | slice-preview | - | function | 授权变更即失效 | S |
| S16 | 会话搜索 overlay（session-query） | session-search-overlay.js | harness/extra_overlay.go:11 session-search patch | closed | info | slice-chat | - | function | Ctrl+F 搜索面板可用 | S |
| S17 | 导入 / 归档 / 撤销 | data-import.js, import-transaction.js | importer/* + frame/archive_undo.go | closed | info | slice-import | - | function | 导入事务 + 归档撤销 | S |
| S18 | 团队/工作流/子代理 | *.js + ui-* | chat/workflow*.go（round128/130） | closed | info | slice-chat | round128/130 | function | 卡片与收据 | S |
| S19 | 终端（terminal/follow 线协议） | pty.js | store/hostterminal（follow+一元 write/resize） | closed | info | slice-terminal | status.md:273 | function | 终端打开/输入/缩放 | M |
| S20 | 浏览器预览本地服务/工作区范围 | preview-workspace.js, local-url.js | browserpreview/local_servers.go, workspace_scope.go | closed | info | slice-preview | - | function | 本地端口预览 + 作用域恢复 | S |
| S21 | 窗口 AppUserModelID / 快捷方式身份 | window-app-details.js:5-8 | mygo.json identifier + 图标 | closed | info | slice-shell | - | function | 安装后任务栏分组；仅静态身份核对，未实机看分组 | S |
| S22 | 首次隐藏到托盘的提示（once-only） | background-notice.js:1-25 | desktop_tray_notice.go trayHideNotice | closed | info | slice-shell | - | function | 首次隐藏弹提示 + marker | S |
| S23 | 市场更新检查（30min TTL） | marketplace-updates.js:1-10 | market/details.go:45-49（30m） | closed | info | slice-market | - | function | 插件有新版本标记 | S |
| S24 | 更新注意提示（attention） | update-attention.js | update_attention.go | closed | info | slice-shell | - | function | 更新提示与忽略 | S |
| S25 | 设置文案 623 处审计 | settings copy | round144 46 处修正 | closed | info | slice-settings | round144 | copy | providerCopy 审计 | M |
| S26 | 启动器图标/品牌 16×16 path + whale-head | launcher.html 资源 | round151 | closed | info | slice-launcher | round151 | layout | 逐字节/逐 path | S |
| S27 | 桌面 hero 纵向 seat / 首屏徽标 | round152 实测 | round152 徽标「大肥鱼版」；hero seat 未定 | open | low | slice-chat | round152 | layout | 需桌面 hero 参考；浏览器参考不可直接用 | S |
| S28 | 聊天 chrome 残项（composer 772 vs 774.4、导航基线 1.7 DIP） | round156 实测 | round156 已修 main；残项未清 | open | low | slice-chat | round156 | layout | 像素差 2.4 DIP / 基线 1.7 | S |
| S29 | renderer 崩溃源（四源之一） | crash-report.js SOURCES 含 renderer | MyGo 无 renderer 进程 | closed | info | slice-shell | round131 | function | 架构差异，web-boot 覆盖渲染失败 | S |
| S30 | 通知 AUMID / Electron.lnk 注册 | legacy-notification-shortcut.js | 未注册（round126 明确） | open | low | slice-shell | round126 | function | 通知中心归属应用名 | M |
| S31 | 验收合同零双签（流程门禁） | docs/acceptance-ui-parity.md | 仍零双签 | open | high | lead | status.md:227 | function | 需用户双签；非代码项 | - |
| S32 | 原生参考管线（token URL+CDP 截图/DOM rect） | (round152/155 工具) | scripts/native-web-shot.mjs + window-capture.ps1 | closed | info | slice-verify | round152/155 | function | 可复现出图 | S |
| S33 | Project overlay 挂载（staged vendor 缺 current-host-build.json） | dsh-project bridge | round150 接受安装版 readiness | closed | info | slice-runtime | round150 | function | 首 patch = desktop-project.patch.yml | M |
| S34 | 性能：per-profile project stage 成本 / current-host 校验只认源码构建 | (round150 残留) | 未解决 | open | low | slice-runtime | round150 | function | 启动耗时与校验口径 | M |

## 3. 剩余 Top-20（按影响排序）

1. G44 手机远程 G1：无 mobile runtime（phoneremote/bridge.cjs 依赖 vendored daemon）——需真机 slice；证据 round157
2. G45 远程 G5 LAN 发现（mDNS/同网段）——round157 未做
3. G45 远程 G6 TLS/SSH 通道——round157 未做
4. G31b 桌宠渲染层：Live2D + ORT/anime4k vs 单层 canvas rig（round154/159 明确保留）
5. G39 通知呈现：系统通知中心 vs 托盘气泡（round126 平台差）
6. S01 harness 页面下载：无 will-download 处理，native index.js:1359 + download-path.js
7. G46 dsh-remote 的 remote shell / 远程 git dispatch 未接线取证
8. G28 字体基线/滚动条/选择复制/tooltip 未逐项对照（像素级）
9. G09 右侧/底部工作区逐面板未系统化对照
10. G27 设置各专页控件×状态未逐页（安装类需可回滚环境）
11. G52 cmd/ 254 处 inline 文案未审计（round142 仅聊天/输入/设置）
12. G03 应用菜单栏可见性未在实机断言（headless 不可断言）
13. S05 插件崩溃取证（OOM/EACCES 分类）仅有 launcher 状态
14. S03 DevTools 快捷键窗口级 + 打包门控缺失
15. S04 浏览器预览缩放档位缺失
16. S02 安装前磁盘空间预检缺失
17. G32 桌宠行为族数值/曲线未逐项
18. G10 标题栏右侧控件逐图标命中区/悬停未逐项
19. S27/S28 聊天 chrome 残项（hero 纵向 seat、composer 772 vs 774.4、导航基线 1.7 DIP）
20. S31 验收合同零双签（流程门禁，owner lead）

## 4. 离线无法验证

- G44 G1：手机远程缺 mobile runtime，需要真实 daemon + 手机端联调，离线只能看 bridge.cjs 的 ready 判定
- G45 G5：LAN 发现需要同网段第二台设备/mDNS，离线无法证明
- G45 G6：TLS/SSH 远程通道需要真实证书与 SSH 主机
- G39/G30：通知中心条目与 AUMID 归属需要安装后的系统通知中心观察，开发态不注册 Electron.lnk
- S01：文件下载落盘需要可写的 Downloads 与真实 harness 页面（可 data: 链接近似验证，但仍需一次运行）
- G03：Windows 系统菜单栏可见性在 headless/UIA 下不可断言，需要实机截图（round134 已登记）
- G09/G10/G27/G28：逐面板/逐页/字体基线需实机像素对照，当前仅有部分 round149-156 基线
- G33/G34：pet-chat 与 phone PR 窗的 native 创建参数未在参考源码中定位，无法离线对照
- G46：remote shell / git dispatch 需要第二台机器或 SSH 目标，离线只能核对注册表与文件域
- S05：插件崩溃取证需要真实插件崩溃（OOM/EACCES），离线只能用单测注入
- 真实 harness 端到端：需要 runtime/rc.2 staging 与端口 3099，本会话不复用 3080 的 GUI 会话，未跑

## 5. 结论

round126-159 把 2026-10-24 矩阵里 6 条 hard missing 全部关掉（G14/G16/G17/G18/G40/G41/G42），并修掉 G01 这条测量错误；当前剩余以 **远程/中继的真实运行时**（G44/G45/G46，需真机）、**桌宠渲染层**（G31b）、**通知呈现**（G39）、**设置/工作区/字体的逐项像素对照**（G09/G10/G27/G28）为主，另有本轮 sweep 新发现的 **文件下载（S01）、安装空间预检（S02）、DevTools 快捷键（S03）、预览缩放（S04）、插件取证（S05）** 五个小口子。
验收合同 `docs/acceptance-ui-parity.md` 仍零双签（S31），这属于流程门禁而不是代码差距；在它双签前不能宣称 1:1 完成。

