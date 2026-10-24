# Round126 · 缺失功能补齐（Welcome / 通知 / 外部编辑器 / WhaleBridge 窗口 / AI 提交信息）

本轮把 10-22 功能审查登记里能独立成窗口/入口的 5 项完全缺失补成可运行实现，并把更新过程从「只有按钮」补到「进度行 + 技术详情 + journal 证据」。构建为 [pass269](<C:/AI/WhaleIsle-MyGo/whaleisle-pass269.exe>) 并更新 [默认程序](<C:/AI/WhaleIsle-MyGo/whaleisle.exe>)。整体「布局、UI、功能、动效完全一致」目标仍在进行；参考 Electron 本体、后端服务与 GUI 3080 未改动（只读对照）。上一轮提交为 `25c23896`。

## 1. 补全项与参考对照

| 审查项 | 参考（Electron 0.3.9） | MyGo 实现 | 端口差异 |
|---|---|---|---|
| #3 Welcome 首启窗口 | `src/main/welcome-window.js`、`welcome-backend.js`、`renderer/welcome.html` + `welcome-layout.css` | [窗口](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/welcome_window.go>)、[页面](<C:/AI/WhaleIsle-MyGo/internal/ui/welcome/welcome.go>)、`welcome.*` 文案 | 无独立 BrowserWindow；同一 720×560 面板，登录/API Key/跳过三条路径，完成后写 `welcome-seen` 标记再进 launcher |
| #4 系统通知 | `src/main/system-notifications.js` | [system_notifications.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/system_notifications.go>) | Electron `Notification` 换成托盘气泡；不注册 `Electron.lnk`/AppID，开发边界与打包一致 |
| #10 外部编辑器打开 | `src/main/editors.js`（VS Code/Cursor/Windsurf/VSCodium 等 + `--goto` 行列） | [editors](<C:/AI/WhaleIsle-MyGo/internal/desktop/editors/editors.go>)、[open_editor.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/open_editor.go>)、文件预览「打开方式」 | 探测顺序、安装路径 glob 与 `--goto` 参数按原版；不启动编辑器时只做文件系统/PATH 探测 |
| #11 WhaleBridge 账号窗 | `src/main/whalebridge-window.js`（1120×780，只加载本机管理页） | [whalebridge_window.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/whalebridge_window.go>)、[management.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/components/management.go>) | 复用共享 WebView2 Host 而不是第二个 BrowserWindow；管理地址经 `WhaleBridgeManagementAddress` 校验，拒绝其它来源 |
| #14 AI 提交信息生成 | `src/main/git-generate.js`（45s 预算、启发式、清洗、未跟踪文件合并摘要） | [commit_generation.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/git/commit_generation.go>)、[git_generate.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/git_generate.go>)、标题栏提交条「自动生成」 | 45s 请求预算之外加本地 Git 读取余量，总 60s；草稿在生成期间保持可编辑，结果落到下一帧草稿，避免同帧保存覆盖 |
| 更新过程 | `src/main/update-dialog.js`、`update-overlay.js`、`update-journal.js` | [updateprogress.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/updateprogress.go>)、[update_progress.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/update_progress.go>)、`update-journal/` | 不做第二个 overlay 窗口：同一个更新在标题栏下一行显示 checking/downloading/staged，安装决定仍留给确认卡；journal 失败只丢证据，不阻塞更新 |
| 详情卡 badge | 安装版 `ToolDetails` 的 `STATUS_KEYS` 词表 | [tool_details_badge.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_badge.go>) | 只本地化自有词表，未知状态保留原文拼写 |

Welcome 只在「无 API Key 且无 welcome-seen 标记」且非静默/后台启动时出现；完成或跳过后调用同一条 `beginLauncherBoot`，launcher/boot 只会跑一次，不会出现两条启动路径。WhaleBridge 的 `open`/`manage` 先确保组件实例已启动，再打开内嵌管理窗，安装动作保持原状不接管。

## 2. 回归

新增/扩展测试：

- Welcome：`TestWelcomeRequiredGate`（有 key、有标记、静默启动三种跳过）、`TestWriteWelcomeMarkerIsIdempotent`、`TestWelcomeKeyShape`。
- 通知：`TestSystemNotificationsAllowed`、`TestDevelopmentAppMatchesThePackagingBoundary`。
- 编辑器：`TestAllMatchesElectronEditorList`、`TestArgsGotoLineColumnAndDirectPath`、`TestLocatePrefersExecutableOverBatchShim`、`TestLocateSkipsBatchShimWithoutExecutable`、`TestLocateExpandsInstallPathsAndGlobsNewestChannel`、`TestAvailableKeepsDefinitionOrder`、`TestOpenWithRejectsUnknownAndMissingEditor`、`TestLocateUnixInstallPath`。
- WhaleBridge：`TestWhaleBridgeManagementAddress`、`TestSameManagementOrigin`。
- 生成：`TestCommitGenerationOriginalHeuristic`、`TestCommitGenerationOriginalRequestAndSanitize`、`TestCommitGenerationConfiguredFailureAndCancellation`、`TestMergeGenerationSummaryNamesUntrackedFiles`、`TestGenerateWorktreeCommitMessageFailsWithoutChanges`。
- 更新：`TestUpdateDownloadProgressCopy`、`TestUpdateAvailableDetailKeepsTheEvidenceBounded`、`TestUpdateJournalRecordsCheckAndDownloadMilestones`、`TestUpdateJournalStaysBestEffort`、`TestNewJournalWritesTheStartedMilestone`、`TestJournalRequiresAnAbsoluteAndKnownAction`、`TestJournalStateWhitelistsFields`、`TestJournalSkipsRepeatedStates`、`TestJournalToleratesNil`。

顺带修正两处过期断言：`settings_sections_test.go` 仍按旧「UI language」检查设置总览页；round2 已按参考把该行改为 `Language`（`KeySettingsDesktopLocale` 只留给 Interface 专页），测试改为断言参考文案，避免把已对齐的界面判成回退。

全量受影响包：Chat 97.7s、Frame 25.2s、i18n 0.6s、Chrome 4.4s、Welcome 1.7s、editors 0.5s、git 113.9s、components 2.6s、update 2.1s、hostterminal 1.9s、cmd/whaleisle 37.7s，全部 ok；`go vet` 受影响的 11 个包退出 0；`go build ./internal/... ./cmd/...` 退出 0。证据见 [检查记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/round126-checks.json>)。

`go build ./...` 仍会撞上 `tmp/` 里历史遗留的 `dumpparse.go`、`gofmtcheck/`、`resize.fixed.go`（多包同目录），与本轮无关；以 `./internal/... ./cmd/...` 作为构建门禁。

## 3. 构建

默认与版本化产物均为 **86,703,104 字节**、SHA256 `7BAF783A3737CE464C7336FDE6961B6177DFEF331E3CDEF6F4319C54F9AC77F5`，见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/round126-build.json>)。无二进制提交。

## 4. 边界与未完成

- 本轮没有跑真实窗口验收：Welcome/WhaleBridge 窗需要真实凭据与本机组件实例，证据是无头 `ui.Tester` 的行为与几何断言；实机窗口仍受 Host 门禁与 150% DPI 限制。
- 通知是托盘气泡而不是系统通知中心条目，图标/点击行为与 Electron 不完全等价；这是 MyGo 平台能力差异，不是漏接。
- 编辑器只在 Windows 探测并测试了安装路径；`TestLocateUnixInstallPath` 只覆盖路径规则，没有 mac/linux 实机。
- 生成提交信息依赖 harness 模型；测试用假模型覆盖启发式、失败与取消，没有真实模型调用验收。
- 审查登记里仍有 **11 项完全缺失**（桌宠/Live2D、手机远程服务端、全局快捷键、崩溃报告、媒体权限、关闭遮罩、应用菜单、dsh-remote RPC、task-control peer、在线壁纸目录、relay/远程 shell）与 **7 项空壳**（组件 install、phoneremote、壁纸内容层、market/remote 注释、im 通路、usage 注释、terminal 已修）；ToolDetails 通用条目、作业/团队/会话/Cordis/workflow/ralph 详情族、Inspect 面板、字体基线、选择复制、滚动条、tooltip、presence 动效、Host 主页/物理输入、外壳材质、完整启停仍未对齐。
- 按验收合同 `docs/acceptance-ui-parity.md` 仍是**零双签**：本轮只减少缺失项，不代表全产品对齐完成。
