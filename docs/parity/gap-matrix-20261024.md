# 原生 ↔ MyGo 差距矩阵（2026-10-24）

HEAD：`4c08edfc`；参考物：Electron 鲸屿 0.3.9（安装版只读）+ `reference/electron` 只读快照。
本文件与 `gap-matrix-20261024.json` 由同一份数据生成，两边的行、状态、计数一致。

## 0. 口径与取证方法

- 原生证据一律给 `文件:行号`（`src/main/**`、`vendor/deepseek-harness/packages/client/**`），安装版包体只作旁证。
- MyGo 证据一律给实际代码路径与行号；**不以 `docs/status.md` 或 round 文档自述为准**——本轮发现多处文档结论已被后续代码推翻（见 §7）。
- 状态四档：**已对齐**（源码值可逐项对上）、**差异**（有实现但值/结构/承载方式不同）、**缺失**（无对应实现）、**未取证/未逐项取证**（存在但不能确认等价，不猜）。
- 工作量：S ≤ 半天、M ≈ 1–2 天、L ≥ 3 天（含回归与证据）。
- 实机窗口像素对照受 Host 门禁与 150% DPI 限制（`final-convergence.md` R7），窗口内观感类行只能到源码级结论。

> 审计备注：审计期间工作树出现并行改动——新增 `internal/ui/chat/tool_details_entries.go`（未提交）并改 `tool_details.go`/`tool_details_model.go`/`i18n/catalog_tool_details.go`，内容正是 G14 的 lines/groups/unchanged。本矩阵按 HEAD `4c08edfc` 判定，该改动合入后需重核 G14。

## 1. 计数

| 项 | 数量 |
| --- | --- |
| 矩阵行数 | 54 |
| 缺失 | 6 |
| 差异（含结构/渲染降级） | 11 |
| 部分对齐 | 5 |
| 未取证 / 未逐项取证 | 6 |
| 已对齐（源码级证据） | 25 |
| round126 及之后已闭环项 | 12 |

> 结论先行：`docs/status.md`「审查修正」的「11 缺失」在 HEAD 只剩 3 项为真缺失——关闭遮罩（G40）、崩溃报告（G41）、媒体权限语义（G42，属差异）；native 也没有 OS 级 `globalShortcut`（`shortcuts.js` 只做应用内 + 菜单加速键），应用菜单/在线壁纸/手机远程运行时/task-control/桌宠窗均有代码。

## A. 主壳与布局

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G01 | 标题栏高度 / 拖拽带 / 交通灯留白 | src/main/chrome.js:9 `TITLEBAR_HEIGHT = 48`；:334/:341 hideNativeMenu 后由注入 chrome 画标题带 | internal/ui/chrome/titlebar.go:36 `TitleBarHeight = 12`，:84-93 只画 12 DIP 行；main.go:2201 传给窗口 | layout | 差异（待实机取证） | 1440×920 实机截图量标题带高度与拖拽区域；对照 native 注入 CSS 的 48；实机窗口受 Host 门禁与 150% DPI 限制（final-convergence R7） | S |
| G02 | 主窗口几何 / 最小尺寸 / 外框圆角 / 透明 | src/main/window.js:209-213（1440×920，min 960×640）；shell-silhouette-radius.test.js:14-28（`--dsw-radius-xl: 20px`，最大化降 0） | cmd/whaleisle/main.go:2193-2204（TitleBarHidden、Transparent、WindowsFrameCornerRadius 20、CaptionButtons 32×32/Top8/Right8/CornerRadius8）+ mainWindowSize（1440,920,960,640） | layout | 已对齐 | 读窗口选项与 mainWindowSize；实机边界圆角抽查 | - |
| G03 | 原生菜单栏可见性 | src/main/chrome.js:48 `autoHideMenuBar: true`，:65-70 `setAutoHideMenuBar(true)` + `setMenuBarVisibility(false)` | cmd/whaleisle/application_menu.go:34 `mygo.App.SetMenu(...)`；全仓无 SetMenuBarVisibility/AutoHideMenuBar/HideMenu 调用 | layout | 差异（风险，未取证） | Windows 实机看是否出现系统菜单栏并把内容下压；若出现则补隐藏调用 | S |
| G04 | 启动器窗口（独立 1060×660 / 无边框透明） | src/main/window.js:906-909（launcher 1060×660，min 860×560，show:false，transparent） | internal/ui/launcher/panel.go（主窗内面板）；cmd/whaleisle 仅 4 处 NewWindow（main/welcome/whalebridge/phone PR），无 launcher 窗口 | layout | 差异（结构） | 空数据目录冷启动量首屏窗口尺寸/圆角/标题带；与 native launcher.html+launcher.css 1440 对照 | L |
| G05 | Boot 页与揭幕（boot.html / boot-reveal） | src/renderer/boot.html、boot.css、boot-reveal.css、boot.js；src/renderer/boot-recovery.js | internal/ui/boot/boot.go、scene.go、curtain.go、backdrop.go（主窗口内页面；curtain.go:10-75 自绘退场幕） | animation | 差异（结构） | 冷启动录屏对比 boot 尺寸/揭幕时序 | M |
| G06 | 三轨几何（sidebar 264-420/280/56、center 400、details 300、45%-70%） | vendor/deepseek-harness/packages/client/ui-layout columns.ts 契约（MyGo 注释引用同一契约） | internal/ui/frame/frame.go:56-67 常量与 ComputeColumns；:70-90 让步算法 | layout | 已对齐 | ui.Tester 几何断言 + 1440/960 两档列宽 | - |
| G07 | 窄窗断点（PHONE_MAX 768、抽屉 320、SIDEBAR_AUTO_COLLAPSE 1024） | ui-layout columns.ts（同一契约） | internal/ui/frame/sidebar_columns.go:12-15（phoneMax 768、phoneDrawer 320）、:22-38 narrow/drawer、:54-58 drawerWidth min(320, vw-48) | layout | 已对齐 | 390/768/960/1024/1440 五档截图（narrow_layout_test.go 已有断言） | - |
| G08 | 会话列宽与两侧 rails / 拖拽提交 | ui-conversation 列宽契约（MyGo 注释引用 desktop ui-layout） | internal/ui/frame/conversation_width.go:15-45（640 下限、column-176、gesture 预览/提交）、:48-56 rails | layout | 已对齐 | 拖拽 divider 后宽度持久化与 960 窄窗回退 | - |
| G09 | 右侧 / 底部工作区（文件树、预览、终端、浮动面板） | ui-sidebar-right、ui-sidebar-files、ui-surfaces、ui-dockkit（vendor client 包） | internal/ui/frame/workspace_panes.go、workspace_dock_tabs.go、workspace_float.go、workspace_layout_contract.go | layout | 未逐项取证 | 逐面板 1440 截图 + 停靠/浮动/最小化三态；round99-113 有专项回归但未逐像素对照 | M |
| G10 | 标题栏右侧控件组（⚙/面板/列/主页/···/Git 徽标） | ui-titlebar 包 + src/main/chrome.js 注入的 caption 槽 | internal/ui/chrome/titlebar.go、workspace_trailing.go、workspace_trailing_icons.go、chrome/gitcommit.go | layout | 未逐项取证 | 逐图标命中区/顺序/悬停态对照；tooltip 与 presence 动效 | M |
| G11 | 共享弹层材质（玻璃/描边/投影/200ms 入场） | Modal.module.css:19-49（28 圆角、modalEnter 200ms、`--dsw-alias-bg-mask-1`） | internal/ui/chrome/dialog_surface.go、dialog_decor.go、glass.go；D34 回归 TestProjectRenameDialogFadesInLikeTheModal | animation | 已对齐 | 首帧/半程/末帧像素采样 + 深色遮罩 .5 | - |
| G12 | 设置模态 28px 玻璃与顶部关闭盒 | Modal.module.css:70-78（22/14/12/24 头部、16/24/500 标题） | internal/ui/settings/modal.go（28px 玻璃；覆盖矩阵记 ➖ 参考即全屏） | layout | 已对齐 | 1440×920 设置页截图对照头部间距 | - |
## B. 会话与工具卡

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G13 | 工具行 chrome + 悬停 Inspect 药丸 | ui-tool components/ToolRow.tsx:93-96（hover 显示）、:344-348（inspectButton） | internal/ui/chat/toolcard.go:201-202（toolInspectButton）、tool_row_chrome.go、toolrow.go | function | 已对齐 | 悬停显示/隐藏、点按跳转会话检查器定位同一 seq | - |
| G14 | ToolDetails 通用条目字段族（lines / markdown / groups / status / change / previousStatus / location / unchanged） | ui-tool components/ToolDetails.tsx:10-26（ToolDetailItem 全字段）、:88-104（lines、markdown、code、groups 折叠）、:129-136（unchanged 折叠块） | internal/ui/chat/tool_details_model.go:34-44 仅 Title/Fields/Badge/Subtitle/Description/Code；无 Lines/Markdown/Groups/Status/Change/Location/PreviousStatus/Unchanged | missing | 缺失（字段族） | 构造含 groups/unchanged/lines 的详情结果（嵌套 diff、todo 历史、未变更段）看是否落入通用正文而非分组折叠 | M |
| G15 | 详情族：goal / schedule（create/update/list/delete） | ui-tool models/details-card-model.ts:49（goalDetail）、:101（scheduleItem）、:82（短时钟）、:97（枚举分隔） | internal/ui/chat/tool_details_model.go:82-93（分派）、:107-163（goal）、:237-372（schedule 全字段/时区/星期/删除收据） | function | 已对齐 | fixture：空 goal、blocked reason、overdue、weekly/cron、删除收据；日期格式 en/zh | - |
| G16 | 详情族：agents / team / terminal 控制收据（spawn_teammate、team_task_*、interrupt_agent、wait_agent、terminal_list/open/close/signal、lsp） | ui-tool models/control-details-model.ts:43（jobList）、:56（terminalList）、:102-122（agent/team 分派）、:176-194（job/terminal 收据） | 缺：仅 internal/ui/chat/tool_row_model.go:18-24 的「工具名→标签」映射；tool_details_model.go:81-102 无这些分支 | missing | 缺失 | 运行 team_task_list / terminal_list / interrupt_agent 后看是否只有通用 badge-描述-lines 正文 | L |
| G17 | 详情族：session 查询 / 检查（session_search、session_event_*、cordis_inspect_*） | ui-tool models/inspection-details-model.ts:19-33（cordis 三命令）、:92-173（session/search 结果表） | 缺（仅 tool_row_model.go:13-20 标签映射）；会话检查器 internal/ui/frame/session_inspector_* 是原生面板，不产出该详情族 | missing | 缺失 | cordis_inspect_query / session_event_search 结果展开对比列数与摘要 | L |
| G18 | 详情族：workflow / ralph 报告（含 completion / blocker 徽标） | ui-tool models/inspection-details-model.ts:36-38（workflow 正则）、:58-61（Ralph completed/blocker 徽标） | 缺（tool_row_model.go:19 仅标签）；workflow 运行另有 internal/ui/chat/workflow.go:48 的独立运行卡 | missing | 缺失 | workflow 完成 + ralph 两种报告各跑一次看正文结构 | M |
| G19 | 卡片族：read / diff / search / web / image / terminal / run_code | ui-tool models/read-card-model.ts、diff-card-model.ts:7（CHAT_DIFF_MAX_LINES=9）、search-card-model.ts、web-card-model.ts、image-card-model.ts、terminal-card-model.ts、tool-call-model.ts:61（run_code→code） | internal/ui/chat/tool_read_model.go、tool_diff_model.go/tool_diff_rows.go、tool_search_model.go、tool_web_model.go、tool_image_model.go、terminal_model.go、tool_code.go（round114-122 逐族收口） | function | 已对齐（分族回归在） | 每族 fixture 快照 + 折叠/复制/打开文件；round114-122 已有逐族证据 | - |
| G20 | todo / plan / ask_user_question 卡 | ui-tool models/todo-diff-model.ts、ask-question-card-model.ts、components/AskQuestionCard.tsx | internal/ui/chat/planboard.go、question_projection.go、question_history.go、frame/question_projection.go | function | 已对齐（未逐项对照） | todo diff 三态、计划卡、问答卡空/已答态 | M |
| G21 | Inspect / trajectory 面板（选中、遮罩、嵌套工具几何、流式生命周期） | ui-tool components/ToolCallTree.tsx、models/inspection-details-model.ts；ui-trajectory 包 | internal/ui/frame/session_inspector_pane.go、session_inspector_tree.go、session_inspector_pick.go、session_inspector_motion.go、internal/ui/chat/session_inspector*.go（round102-113） | function | 已对齐（部分） | 从工具行点 Inspect → 命中同一 seq、嵌套卡片几何、Esc/遮罩关闭顺序 | M |
| G22 | 子代理 / 工作流成员 / 会话祖先卡 | ui-subagent、ui-workflow-run、ui-session 包 | internal/ui/chat/workflow.go、workflow_disclosure.go、workflow_member.go、session_ancestry.go | function | 已对齐（未逐项对照） | 工作流运行卡展开、成员行、子会话返回 | M |
| G23 | 图片灯箱 / 子会话图片画廊 / 附件查看器 | ui-attachment、ui-tool image-card-model.ts；原生 20px 媒体层 | internal/ui/chat/child_image_gallery.go、image.go、attachments.go；composer/attachment_display.go（20px/裁剪） | layout | 已对齐（🟡 无参考同款） | 打开/关闭/Esc/点遮罩；缩放与裁剪 | S |
## C. 设置

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G24 | 设置分区覆盖（account/general/appearance/interface/models/plugins/skills/market/remote/mcp/agent-presets/pet/about/usage-stats/remote-workspace/agent-loop/web-search/subagent/shell） | packages/client/ui-settings*（ui-settings、-account、-agent-loop、-general、-market、-mcp、-models、-plugin-inventory、-plugins、-remote、-session-log、-shell、-skills、-subagent、-web-search） | internal/ui/settings/sections.go:14-31（19 个 SectionID）、:38-60 planSectionOrder；各页 settings/*.go（about/account/appearance/interface/models/plugins/skills/market/remote/mcp/agent_presets/pet/desktop…） | function | 已对齐（页面在，深度未逐页取证） | 逐页 1440 截图 + 控件数/空态；round2-6 只覆盖通用/快捷键/插件/市场/安装卡 | M |
| G25 | 通用页行结构、控件列宽、文案（D1-D7） | 设置通用页参考布局（final-convergence D1-D7 记录） | internal/ui/settings/preference_row.go、desktop.go、settings.go；round2 提交 32ec0ad7/42b7fda1 | layout | 已对齐 | 800 DIP 模态内 10 行全可见；权限行文案 `新任务默认权限` | - |
| G26 | 快捷键弹窗 480×600 / 24 圆角 / 行内录键 / 关闭手势 | final-convergence D8-D17、R4-R5（参考速查表、`恢复全部默认` + N 项已自定义） | internal/ui/settings/shortcuts.go、shortcut_recording.go、shortcut_icons.go；frame/shortcuts.go | layout | 已对齐 | 四类关闭手势先收行内编辑器；重置确认文案；底栏计数 | - |
| G27 | 设置各专页深度（模型配置、账号登录、用量、远程、市场、插件、技能、MCP、Agent 预设、关于、宠物） | 对应 ui-settings-* 包 | settings/models.go、model_configuration.go、account.go、body_preferences.go、catalog_pass6_remote.go、catalog_pass4a/b.go、mcp*.go、agent_presets.go、about.go、pet.go | function | 未逐项取证 | 每页控件 × 状态（空/有值）+ 保存回执；插件/市场安装会改环境，需可回滚环境（R8） | L |
| G28 | 字体基线 / 滚动条 / 选择复制 / tooltip / presence 动效 | ui-theme tokens、web-styling.md、motion.css:83-94（FlipText） | round126 自述仍未对齐；有 composer/model_flip_text.go（400ms）与 widgets 动效，但字体基线/滚动条/选择复制无逐项对照 | animation | 未逐项取证 | 同字号文本像素宽、滚动条宽/圆角、选区颜色、tooltip 延迟/位移 | M |
## D. 外围窗口

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G29 | Welcome 首启窗 600×700（无边框/透明/不可缩放） | src/main/welcome-window.js:26-33（600×700、useContentSize、center、resizable:false、frame:false、transparent） | cmd/whaleisle/welcome_window.go:85-92（600×700、TitleBarHeight 36、WindowsCaptionButtons 32×32/Top8/Right8） | layout | 已对齐 | 无凭据冷启动看窗口尺寸/居中/关闭路径；marker 幂等 | S |
| G30 | WhaleBridge 账号窗 1120×780（仅本机管理地址） | src/main/whalebridge-window.js:17（1120×780，min 380×520，hidden titleBar + overlay 32） | cmd/whaleisle/whalebridge_window.go:100；internal/desktop/components/management.go 地址校验 | layout | 已对齐 | 地址白名单拒绝外源；窗口尺寸/标题带 | S |
| G31 | 桌宠渲染（Live2D/ORT 超分 vs 自绘 canvas） | src/main/desktop-pet.js、desktop-live2d.js（:8 依赖 pet-stats）、src/renderer/pet-live2d/（ort/、anime4k.js、anime4k.LICENSE、states/、avatar/、rig/）、pet-physics.js、pet-wander.js | internal/desktop/pet/companion.go:188（220 宽、Frameless/Transparent/AlwaysOnTop/SkipTaskbar）、assets/pet-rig.js（注释：改编自 pet-live2d.js，纯 canvas，无 SDK/模型）、assets/rig（manifest.json+shell.png+tail.png）、tha4.js | function | 差异（渲染降级） | 桌宠外观逐帧对照：模型分层/物理摆动/超分锐度；native 用 ORT+anime4k，MyGo 单层 PNG 刚体 | L |
| G32 | 桌宠行为族（成长/统计/漫步/物理/记忆/喂食） | pet-growth.js、pet-stats.js、pet-wander.js、pet-physics.js、pet-chat.js、pet-dsh-watch.js | internal/desktop/pet/growth.go、growth_clock.go、behavior.go、motion.go、motion_state.go、feed.go、care_animation.go、neural.go、interaction.go、notifications.go | function | 部分 | 同一成长种子跑 7 天曲线；漫步路径/物理回弹；抚摸/喂食/休息动画 | M |
| G33 | 桌宠快捷聊天窗 | src/main/pet-chat.js（独立 BrowserWindow；尺寸常量未取到） | internal/desktop/pet/quickchat.go:91-93（420×580，Parent=pet 窗，AlwaysOnTop）、quickchat_view.go、quickchat_protocol.go | layout | 未取证 | 从桌宠开聊天窗量尺寸/贴边方向/native 对照 | S |
| G34 | 手机 PR 复核窗 660×720 | src/main/dshd-remote.js / remote.js 侧窗（参数未逐项取证） | cmd/whaleisle/phone_pr_review.go:51（660×720，min 520×520，Parent） | layout | 未取证 | 打开手机 PR 复核窗量尺寸/父子关系 | S |
## E. 系统集成

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G35 | 托盘菜单项序 / 文案 / 宠物 checkbox / 分隔线 | src/main/tray-menu.js:17-33（显示窗口/打开启动器/设置…/插件市场/重启 Harness/桌面宠物/分隔/退出） | cmd/whaleisle/desktop_tray_menu.go:29-36（同序，checkbox `pet`，restart 置灰条件） | copy | 已对齐 | 中英双语菜单逐项；禁用态（组件未就绪） | - |
| G36 | 托盘图标栅格尺寸 | src/main/tray.js:10 createTray、:20 `image.resize({ width: 24, height: 24 })` | cmd/whaleisle/main.go:7511 `scalePNG(data, 32)`；trayicon.go:12-30 CatmullRom | layout | 差异 | 托盘区截图量图标物理像素（32 vs 24）、高 DPI 模糊度 | S |
| G37 | 应用菜单命令覆盖与加速键同步 | src/main/menu.js:5 buildMenu、:9-17 commandItem（按配置 revision 取 accelerator）、:26-113 模板（文件/编辑/视图/窗口/帮助/mac 应用菜单） | cmd/whaleisle/application_menu.go:34 SetMenu、:125 buildApplicationMenu、:153 view 组、application_commands.go:26-39 命令表 + shortcuts.Resolve | function | 部分（项覆盖与运行时加速键刷新未逐项对照） | 逐项对照 native 模板命令 ID/文案/加速键；配置修订后菜单 keycap 是否刷新 | M |
| G38 | 快捷键服务（协议、录制、导入、菜单抑制、CAS 重载） | src/main/shortcuts.js:31 PROTOCOL_URL、:33 CHANNELS、:51 ariaToAccelerator、:102 menuAccelerators、:231 acceleratorFor、:190 ipc get；无 OS 级 globalShortcut | internal/shortcuts/{shortcuts.go,physical.go,recording.go,shared.go}、internal/shortcutprefs/document.go、cmd/whaleisle/{shortcut_reload_*,unified_shortcuts_test}.go | function | 部分（无 globalShortcut，native 亦无；菜单加速键同步未取证） | 改键 → 菜单 keycap 与页面命令同时生效；冲突拒绝；导入外部键表 | M |
| G39 | 通知呈现（系统通知中心 vs 托盘气泡） | src/main/system-notifications.js:10-15（Electron Notification + 平台/开发边界） | cmd/whaleisle/system_notifications.go（托盘气泡；round126 明确不注册 Electron.lnk/AppID） | function | 差异（平台能力） | 通知中心条目/图标/点击行为；开发与打包边界 | M |
| G40 | 关闭遮罩（全屏 curtain + 24 圆角卡 + 0.85s spinner） | src/main/closing-overlay.js:1-52（z-index 2147483647、卡 24 圆角/24 padding、spinner 36px/0.85s linear、标题 16/24、说明 12/18）；src/main/window.js:486 覆盖窗 | 缺：cmd/whaleisle/quit_lifecycle.go:16-70 只有 quit 保护/清理；internal/ui/boot/curtain.go 是 boot 退场幕，不是关闭遮罩 | missing | 缺失 | 有运行任务时关窗：native 先出遮罩再确认/退出；MyGo 直接确认框 | M |
| G41 | 崩溃报告（crash-<ts>-<source>.log，保留 10 份，恢复对话框引用文件名） | src/main/crash-report.js:19 CRASH_REPORTS_RETAINED=10、:106 writeCrashReport、:119 pruneCrashReports | 缺：全仓无 crash-*.log 写入（仅 internal/config/harness_restart.go 的普通重启策略）；cmd/whaleisle/startup_recovery.go 无崩溃文件引用 | missing | 缺失 | 注入一次 fatal（host/web-boot/renderer/main 四源）看 userData 是否落盘 + 恢复对话框点名 | M |
| G42 | 媒体权限（仅主 harness 帧、仅音频、仅 loopback） | src/main/media-permissions.js:32-39（media 仅 primary 主帧 + 127.0.0.1/localhost + mediaType audio + macOS 授权；notifications 受 systemNotificationsAllowed 约束） | internal/desktop/webview/host.go:361 `SetPermissionHandler(... return false)`（一律拒绝）；麦克风走 native voicecapture 旁路 | function | 差异（WebView 内麦克风被拒） | harness 页面请求 getUserMedia(audio) 是否成功；notification 权限是否随开关 | S |
| G43 | 任务保护 / 关闭到托盘 / 退出确认 | src/main/task-protection.js、close-behavior.js、closing-overlay 流程 | internal/desktop/taskguard、cmd/whaleisle/task_protection.go、portable_close_policy_test.go、desktop_tray.go:153 | function | 已对齐 | 有 running 会话时关窗/托盘退出必须确认；无任务时 Hide 进托盘 | - |
## F. 远程与网络

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G44 | 手机远程服务端（HTTP 服务 + 配对 + QR + 设备动作） | src/main/mobile-web-server.js:7 MOBILE_WEB_PORT=3180、mobile-web.js、mobile-web/ 目录 | internal/desktop/phoneremote/runtime.go:1-60（listening/relayConnected/pairing URL+PNG/devices）、bridge.cjs（子进程委托 vendored @chisacode/server 与 dshd 远端）、web.go、device_actions.go；UI internal/ui/remote/phone.go | function | 差异（委托 vendored 守护进程，非自研服务端） | 无 Node runtime 时能力退化；配对/QR/设备动作在真实手机上跑一轮 | M |
| G45 | relay 客户端（帧编解码、握手、重连退避） | src/main/relay-client.js:5（relay-frames encode/attachFrameReader）、:294 socket.write(encodeFrame)；shared/relay-frames、relay-auth、lan | 无自研 relay 客户端；bridge.cjs:21/82/96 仅消费守护进程 snapshot 的 relayConnected 标志 | function | 差异（依赖代理） | 断网/握手超时/重连退避行为；relay 直连路径是否可达 | M |
| G46 | dsh-remote RPC（机器注册表、远端文件、remote shell、远程 git dispatch） | src/main/dsh-remote-desktop.js、remote-shell.js、remote.js、dshd-git-dispatch.js、dshd-git-tunnel.js | internal/store/remote/{store.go,registry,files.go,directory*.go,removal.go}、internal/desktop/sshremote/*（隧道/known_hosts）、cmd/whaleisle/remote_*.go；remote shell / git dispatch 未逐项取证 | function | 部分 | 远端目录/文件/删除/草稿保存 + remote shell 会话 + git dispatch 各跑一条 | M |
| G47 | task-control peer（peer 文件 schemaVersion 1 + 取消端点） | src/main/task-control-peer.js:52-60（writePeerFile、schemaVersion 1、tmp+rename 原子写） | internal/desktop/taskcontrol/control.go:1-40（Work/Inspection/Coverage，NeedsConsent 语义）；cmd/whaleisle/task_protection.go 接线 | function | 已对齐 | 伪造 peer 文件 → 运行中任务必须要求确认；覆盖缺失不得当 idle | - |
| G48 | 在线壁纸目录（Bing 今日/归档、Wallhaven SFW、自定义 JSON） | src/main/wallpaper-catalog.js:1（Bing today/year、Wallhaven SFW、custom JSON）、:5-6（UA/4MB 上限）、:24 bingCatalogUrls | internal/desktop/wallpapers/catalog.go:40-42（bing/wallhaven Sources）、:16-38（Source/Item/Query/Result）、images.go；UI internal/ui/theme/gallery.go、host_wallpaper_sources.go:25 | function | 已对齐 | 两个源各拉一页 + 下载应用 + 翻页/搜索/分类 | S |
| G49 | 更新过程（标题栏进度行 + 技术详情 + journal 证据） | src/main/update-dialog.js、update-overlay.js、update-journal.js | internal/ui/chrome/updateprogress.go、cmd/whaleisle/update_progress.go、update_journal.go、internal/desktop/update/*（checking/downloading/staged；journal 失败不阻塞） | function | 已对齐（round126） | 假 feed 走 checking→downloading→staged；journal 写失败仍完成更新 | - |
## G. 动效与文案

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G50 | 动效 token：popover 时长/位移、overlay 位移、退场时长 | ui-theme base.css:16-22（overlay 200ms、overlay-out 100ms、popover 160ms、swap 100ms、flip 400ms、distance-overlay 8px、distance-popover 4px）、motion.css:30/44（translateY 距离）、:12-31（out 用 fast） | internal/ui/chrome/motion.go:11-21（OverlayDuration 200ms、PopoverDuration 150ms、popoverRise 6、overlayRise 2）；退场是否走 100ms 未取证 | animation | 差异 | 录屏逐帧量菜单/弹层位移与时长；reduce-motion 下归零；退场 100ms | S |
| G51 | 触发器文案翻转（权限/模型/推理）与 presence | motion.css:83-94（flip-out/in 400ms）、docs/motion.md 原则 3 | internal/ui/composer/model_flip_text.go、model_flip_projection.go、effect_timing.go:11（400ms） | animation | 已对齐（部分：权限/推理档未取证） | 切换模型/权限/推理强度看翻转而非闪现 | S |
| G52 | 界面文案与双语（i18n 漂移） | vendor client locales.ts / 各包 locale 字典；D3/D4（权限行、语言行） | internal/ui/i18n/catalog_*.go（chat/cmd/composer/context/surfaces/terminal/tool_* 分片）、settings 文案键；tray/menu 用 resumeText 内联中英 | copy | 部分（已修 D3/D4；未做全量键对齐扫描） | 抽取 native 全部可见文案键与 MyGo catalog 做差集；切换 en/zh 全表面巡检 | L |
| G53 | 工具详情卡状态词表（STATUS_KEYS 本地化） | ui-tool ToolDetails badge（安装版 STATUS_KEYS 词表） | internal/ui/chat/tool_details_badge.go（自有词表本地化，未知状态保留原文） | copy | 已对齐（round126） | running/completed/failed/unknown 四态 + 未知原词 | - |
## H. 口径

| ID | 表面 | 原生证据 | MyGo 现状 | 类型 | 状态 | 验证方法 | 量 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| G54 | docs/status.md「审查修正」11 缺失 / 7 空壳清单的时效性 | 以 reference/electron/src 与安装版为准 | 11 项中至少 6 项在 HEAD 已可运行：应用菜单（application_menu.go:34）、在线壁纸（wallpapers/catalog.go）、手机远程运行时（phoneremote/runtime.go）、task-control peer（taskcontrol/control.go）、桌宠窗（pet/companion.go:188）、dsh-remote 注册表/文件（store/remote）；7 空壳中 market/remote/usage/chat 注释亦已修正 | function | 文档过时（需 Lead 决定是否更新 status.md） | 按本矩阵逐条重跑 grep | S |

## 6. round126 及之后已闭环（不再计入缺失）

| 项 | 证据 | 残余 |
| --- | --- | --- |
| Welcome 首启窗口 | cmd/whaleisle/welcome_window.go:85-92 + internal/ui/welcome；round126 | 无（600×700 与 native welcome-window.js:26-33 一致） |
| 系统通知 | cmd/whaleisle/system_notifications.go；round126 | 托盘气泡而非通知中心条目（G39）；不注册 Electron.lnk/AppID |
| 外部编辑器打开 | internal/desktop/editors/editors.go + cmd/whaleisle/open_editor.go；round126 | 仅 Windows 实机验证；mac/linux 只有路径规则测试 |
| WhaleBridge 账号窗 | cmd/whaleisle/whalebridge_window.go:100 + components/management.go；round126 | 无（1120×780 与 native :17 一致） |
| AI 提交信息 / PR 文案生成 | internal/desktop/git/commit_generation.go + cmd/whaleisle/git_generate.go；round126 | 总预算 60s（native 45s 生成预算）；未做真实模型调用验收 |
| 更新进度行 / 技术详情 / journal | chrome/updateprogress.go + update_progress.go；round126 | 同一标题栏行而非第二个 overlay 窗口（有意收缩） |
| 应用菜单（原「缺失 #9」） | cmd/whaleisle/application_menu.go:34 SetMenu、:125 buildApplicationMenu | 菜单栏可见性未隐藏（G03）；命令覆盖待逐项（G37） |
| 在线壁纸目录（原「缺失 #15」） | internal/desktop/wallpapers/catalog.go:40-42 Bing/Wallhaven + theme/host_wallpaper_sources.go:25 | 逐源翻页/搜索未实机验收（G48） |
| 手机远程运行时（原「缺失 #2」部分） | internal/desktop/phoneremote/runtime.go + bridge.cjs 委托 vendored 服务 | 非自研服务端；无 Node runtime 时退化（G44） |
| task-control peer（原「缺失 #13」） | internal/desktop/taskcontrol/control.go + cmd/whaleisle/task_protection.go | 无（peer 文件 schemaVersion 1 对齐） |
| 桌宠窗口（原「缺失 #1」部分） | internal/desktop/pet/companion.go:188 透明置顶无边框 220 宽 + quickchat 窗 | 渲染降级为纯 canvas rig，无 ORT/anime4k 与 Live2D 模型（G31） |
| dsh-remote 注册表/远端文件（原「缺失 #12/#16」部分） | internal/store/remote/* + internal/desktop/sshremote/* | remote shell / git dispatch 未逐项取证（G46）；relay 无自研客户端（G45） |

## 6b. 原「7 空壳」（docs/status.md:250-261）逐项核对

| 项 | 证据 | 残余/结论 |
| --- | --- | --- |
| 空壳 1：components/lifecycle.go:23 install 动作为空壳 | cmd/whaleisle/component_changes.go、launcher_components_refresh.go 走 install 控制面；market/plugin 安装另有路径 | lifecycle.go 本轮未逐行复核，标记 unverified |
| 空壳 2：internal/desktop/phoneremote 整包空壳 | 已推翻：runtime.go 有 listening/relay/QR/devices/device actions，bridge.cjs 守护进程 + web.go | 非自研服务端（G44） |
| 空壳 3：chrome/wallpaper.go:18 壁纸内容层占位 | 已推翻：chrome/background.go:14/40 ConfigureBackdrop + theme/gallery.go + host_wallpaper_sources.go + desktop/wallpapers | 内容层能否真正铺满窗口背景取决于 MyGo 窗口背景 API，需实机截图 |
| 空壳 4：internal/ui/market 注释 stub | 注释已修正；market 面板列表/安装/启停/卸载在 internal/ui/market/* + internal/desktop/market/* | 无 |
| 空壳 5：internal/ui/remote 注释 stub / dsh-remote RPC | 注释已修正；remote/machines.go、directory.go、files.go 已接 internal/store/remote | remote shell/git dispatch 未逐项取证（G46） |
| 空壳 6：internal/store/im 的 im/* 通路 | 上游 vendor 无 im/* typert；本地 IM 走 cordis 通道（docs/status.md:191-193） | 完整 IM 仍靠 dsh-im WebView（上游/平台边界） |
| 空壳 7：usage/chat 包注释 stub | 注释已修正；internal/store/usage/* 与 internal/ui/chat/* 均为实现 | 无 |
| 原 #8：terminal/follow 流（已修） | internal/store/hostterminal 已按 harness 线协议（terminal/follow + 一元 write/resize）接线（docs/status.md:273） | 已闭环，不计缺失 |

## 7. 文档口径漂移（与旧文档冲突处，以代码为准）

- docs/status.md:229-248 的 11 项缺失中，应用菜单、在线壁纸目录、手机远程运行时、task-control peer、桌宠窗口、dsh-remote 注册表在 HEAD 已有可运行代码（见 G54 与 §6）
- docs/review-20261022/round126-missing-features.md:9 称 Welcome 为「720×560 面板」，实际 cmd/whaleisle/welcome_window.go:87-88 为 600×700（与 native 一致）
- round126-missing-features.md:46 仍列「作业/团队/会话/Cordis/workflow/ralph 详情族」未对齐；其中作业（job_list/output/kill）已在 tool_details_model.go:94-99,548-630 实现，团队/会话/Cordis/workflow/ralph 确实缺失
- docs/status.md:187 称壁纸「仅本地目录画廊」，与 internal/desktop/wallpapers/catalog.go:40-42 的 Bing/Wallhaven 不符

## 8. 下一轮 Top-10 队列

1. G14 ToolDetails 字段族补齐（lines/markdown/groups/status/change/location/unchanged）——所有详情卡的嵌套结构都依赖它
2. G16 agents/team/terminal 控制收据详情族（control-details-model.ts:43-194）
3. G17 session 查询 / cordis 检查详情族（inspection-details-model.ts:19-33,92-173）
4. G18 workflow / ralph 报告详情族（inspection-details-model.ts:36-61）
5. G40 关闭遮罩（closing-overlay.js:1-52：24 圆角卡 + 0.85s spinner）
6. G41 崩溃报告（crash-report.js:19,106,119：crash-*.log、保留 10、恢复对话框点名）
7. G42 媒体权限（media-permissions.js:32-39 vs webview/host.go:361 一律拒绝）
8. G04 启动器独立窗口 1060×660（window.js:906-909）与 G05 boot 独立窗口/揭幕
9. G31 桌宠渲染（Live2D + ORT/anime4k）对齐，当前为单层 canvas rig
10. G50 动效 token 校正（popover 160ms/4px、overlay 8px、退场 100ms）与 G01 标题带 48 vs 12 实机取证

## 9. 未能取证 / 明确不猜

- 实机窗口截图/尺寸：Host provenance 与 150% DPI 下窗口超出屏幕、UIA 坐标不可靠（final-convergence R7），故 G01/G03/G04/G05 的窗口内观感只有源码证据，没有像素对照
- G33 桌宠快捷聊天窗 native 尺寸/贴边方向：pet-chat.js 顶部未含窗口尺寸常量
- G34 手机 PR 复核窗 native 对应尺寸：dsh-remote-desktop.js/remote.js 的窗口创建参数未逐行取证
- G45 relay 直连是否在 vendored 守护进程内真实可用：仅见 bridge.cjs 消费 relayConnected 标志
- G46 remote shell / 远程 git dispatch 的 MyGo 接线深度：仅确认 store/remote 的注册表与文件域
- G37 应用菜单命令覆盖：只核对了 MyGo 现有分组（application_menu.go:125-160），未逐条对照 menu.js:26-113 全模板
- G28 字体基线/滚动条/选择复制/tooltip：无逐项对照证据，仅为 round126 自述

## 10. 结论

主壳几何（1440×920/960×640、20px 圆角、264–420 侧栏、768/1024 断点）、会话与工具卡主体（read/diff/search/web/image/terminal/run_code/goal/schedule/jobs/inspect）、设置容器与专页、托盘菜单、欢迎/WhaleBridge 窗口、任务保护已能用源码逐项对上。
真正拉开「原生感」的是三处：**详情卡的通用字段族与四类控制/检查详情族**（G14、G16–G18）、**关闭与崩溃两条兜底体验**（G40、G41）、**桌宠渲染层级**（G31）；其次是启动器/boot 的窗口承载方式（G04、G05）与动效 token 的几个数值（G50）。
按本矩阵，**整体仍未达到验收合同的「零差异」**：`docs/acceptance-ui-parity.md` 仍是零双签，且上表 6 类缺失/差异需要代码改动而非文档修正。

