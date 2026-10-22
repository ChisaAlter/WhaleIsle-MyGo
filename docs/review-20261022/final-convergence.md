# 收口报告：MyGo鲸屿 ↔ 鲸屿本体

日期：2026-10-22  
参考：Electron 鲸屿 0.3.9 安装版（`C:\软件\Whale Isle`）的 `app.asar` 与
`reference/electron/vendor/deepseek-harness` 源码（CSS 变量、Modal、PluginManagerPage、locales）。  
验证方式：生产渲染 `ui.Tester` 的几何/像素断言 + 键盘指针事件 + `go test ./internal/ui/...`，
参考侧以源码声明为准（实机窗口受 Host 门禁与 150% DPI 限制，见残留 R7）。

本报告汇总第 2–6 轮与收口轮的 34 条差异（D1–D34），接在前 30 轮审计（A1–B21，见 `final-report.md`）之后。
逐轮细节留在 `round2-general.md` … `round6-install-footer.md`。

## 1. 差异清单（D1–D34）

状态：✅ 已修复并有回归测试；🟡 已接共享表面但无逐项参考对照（原生功能）。

| ID | 差异 | 状态 | 证据 |
| --- | --- | --- | --- |
| D1 | 设置·通用页 10 行在 800 DIP 模态内被裁到最后一项不可见 | ✅ | `round2-general.md` |
| D2 | 设置行控件列固定 180 DIP 并 `FillWidth`，说明列被压 | ✅ | 同上 |
| D3 | 权限行文案漂移（`新任务默认权限`） | ✅ | 同上 |
| D4 | 语言行多出 `界面语言` + 说明 | ✅ | 同上 |
| D5 | 三条设置行标题/说明与参考不一致 | ✅ | 同上 |
| D6 | 快捷键在页面内联展开，参考是 Setting-Cell + 弹窗 | ✅ | 同上 |
| D7 | 行内控件宽度节奏与参考不同 | ✅ | 同上 |
| D8 | 快捷键弹窗 720×720/20 圆角/文字关闭，参考 480×600/24 圆角/28 关闭盒 | ✅ | `round3-shortcuts.md` |
| D9 | 每条命令常开编辑器（约 150 DIP），参考 42 DIP 行 + 原地录键 | ✅ | 同上 |
| D10 | 检索区无图标/清除、占位与高度不同 | ✅ | 同上 |
| D11 | 页内长说明、缩放簇、逐条帮助文字挤掉列表 | ✅ | 同上 |
| D12 | 底栏三个普通按钮，参考 `恢复全部默认` + `N 项已自定义` | ✅ | 同上 |
| D13 | 重置确认文案漂移 | ✅ | 同上 |
| D14 | 行编辑要按底部“保存”，参考即改即存 | ✅ | 同上 |
| D15 | 快捷键弹窗未按参考下移 30 DIP（161 vs 191） | ✅ | `round4-dialog-surface.md` |
| D16 | 行悬停没有 12 DIP 铅笔暗示 | ✅ | 同上 |
| D17 | 行内编辑器多 `取消` 与可见说明行 | ✅ | 同上 |
| D18 | 插件安装弹窗 640×720/20 圆角/无淡入/文字关闭 | ✅ | 同上 |
| D19 | 索引源弹窗底部重复 `关闭`、正文重复标题 | ✅ | 同上 |
| D20 | 市场确认弹窗 20 圆角/无淡入/无头部关闭 | ✅ | 同上 |
| D21 | 市场详情卡遮罩 `.35`、头部 16 等距、自绘描边 | ✅ | 同上 |
| D22 | 关闭盒一次点掉行内编辑器 + 弹窗两层 | ✅ | `round5-close-gestures.md` |
| D23 | Esc 直接关弹窗，未先收行内编辑器 | ✅ | 同上 |
| D24 | 遮罩点击同样一次退两层 | ✅ | 同上 |
| D25 | 重置确认打开时行内编辑器未先 `setTarget(null)` | ✅ | 同上 |
| D26 | 编辑态行标签是可聚焦控件（第 4 个 Tab 停靠点） | ✅ | 同上 |
| D27 | 安装卡空态两个整宽按钮，参考 `.optionsRow` 左右两端开关 | ✅ | 同上 |
| D28 | 安全说明与主按钮随正文滚动，短窗下主按钮被裁出卡片 | ✅ | `round6-install-footer.md` |
| D29 | 安全说明是一行纯文本，无 warn 面板/图标/两段文案 | ✅ | 同上 |
| D30 | 主按钮用主题蓝 `ui.PrimaryButton`，参考 `.wide` 整宽 40 高 | ✅ | 同上 |
| D31 | 检查中主操作消失，状态行与动作分离 | ✅ | 同上 |
| D32 | 引导是平铺三行文字，参考 `.guide` 面板 + 示例行 + “填入示例” | ✅ | 同上 |
| D33 | 市场确认卡仍是自写一行安全短句，与 spec 卡两种口径 | ✅ | 本文 §2.1 |
| D34 | 原生生命周期/选择卡无入场动画、遮罩固定 `.4` | ✅ | 本文 §2.2 |

参考声明位置（供复核）：`Modal.module.css:19-49`（遮罩 + `modalEnter` 200ms + 28 圆角面板）、
`Modal.module.css:70-78`（头部 22/14/12/24 + 16/24/500 标题）、`PluginManagerPage.tsx:954-974`
（`.installContent` 滚动 + footer 槽）、`PluginManagerPage.tsx:1032-1059`（`.guide` 面板）、
`pluginmanager/locales.ts:77-78, 90-91`（两句安全/升级文案、`installChecking`）。

## 2. 本轮修复（D33–D34）

### 2.1 D33 市场确认卡改用参考的 `.installSafety` 面板

- 现状：`确认安装插件` 卡（原生功能，参考无对应弹窗）正文是一行 12 DIP 灰字
  `请确认来源可信。插件以当前 profile 的权限运行。`，与 spec 卡的 warn 面板两种口径。
- 修复：`marketChoiceDialog` 的安全行改为 `installSafetyNote(c)` —— 与 spec 卡完全相同：
  12 圆角、10/12 内边距、14 DIP `IconWarningOutlineRegular`、两行参考文案（`installGuideSafety`
  + `installUpgradeNotice`）、4 DIP 行距、`rgb(254,245,231)` / 深色 `rgb(39,36,31)` 底。
- 回归：`TestMarketChoiceCarriesTheReferenceSafetyNote` —— 参考文案在位、旧短句消失、
  文字左缩 58（正文 24 + 面板 12 + 间距 8 + 图标 14）、文字宽 466（512 − 12 − 12 − 8 − 14）、
  面板左缘像素为 warn 底色。修复前失败：`the market card lost the reference's trust note`
  （文本里仍是那行旧短句）。
- 证据：`plugins-market-choice-1440.png` 42231 → 51494 字节。
- 深色：`TestInstallSafetyNotePaintsTheDarkTint` 在 `deepseek.dark` + `ui.DarkTheme()` 下断言
  面板左缘为 `rgb(39,36,31)`、卡片本身不是该色；证据 `plugins-install-dark-1440.png`（51246 字节）。

### 2.2 D34 原生生命周期/确认卡补上参考 Modal 的 200ms 入场

- 参考：`Modal.module.css:25-31` 的 `.mask::after` 与 `:34-49` 的 `.dialog` 都跑
  `modalEnter var(--ds-transition-duration)`（200ms，缓入缓出），遮罩用 `--dsw-alias-bg-mask-1`。
- 现状：这些卡手写 `back.Background(ui.RGBA(0,0,0,.4))`，入场即在第一帧全不透明，
  遮罩固定 `.4`（不随明暗主题变），与共享弹窗表面（`DialogSurface`：淡入 + `.24/.5`）不一致。
- 修复：`chrome` 新增 `DialogMask(c)` 与 `DialogReveal(c, backdrop, panel)`：
  遮罩 `.24`（浅）/ `.5`（深），遮罩与面板同进度 `EaseInOut` 200ms 淡入（`Reveal` + `RevealMask`），
  减少动态时直接 1。接到 7 处手写卡片：projects 重命名、停止并归档、项目页面弹窗（2）、
  撤销要求改动、dirpick 新建文件夹、workspacepicker 改名/移除确认。
- 回归：`TestProjectRenameDialogFadesInLikeTheModal` —— 用 `tt.SetTime` 采样：
  首帧卡片位置仍是页面底色 255（未绘制）、半程为中间值、结束落在 `Field` 的 245；
  遮罩 255 → 224 → 194（= 255 × 0.76，即 24% 黑）。修复前失败：`the card paints 245 at the
  start of the enter, want the page's 255`。
- 证据：`projects-rename-fade-1440.png`（半程，卡片半透明、遮罩更浅）、`projects-rename-1440.png`、
  以及重新生成的 `projects-rename-narrow.png`、`projects-archive-narrow.png`、
  `directory-picker-create.png`、`workspace-rename.png`（遮罩由 `.4` 变为 `.24`）。
- 测试基建：新增 `settleDialogReveal`（`SetTime(start)` 再 `SetTime(start+2s)`）在 4 个既有快照
  夹具里先让入场结束再编码，避免证据图拍到首帧透明卡片。

## 3. 修复清单（第 2–6 轮与收口轮提交）

| 轮次 | 提交 | 范围 |
| --- | --- | --- |
| 2 | `32ec0ad7`、`42b7fda1` | 设置·通用页行结构/控件宽度/文案（D1–D7）；输入框不再在聚焦/悬停时画边框 |
| 3 | `08d8723c` | 快捷键弹窗重建为参考速查表（D8–D14） |
| 4 | `9f5db8ac`、`41f5989b`、`01c69d27`、`047f3aaf` | 效果编辑面板；快捷键弹窗下移 30 + 悬停铅笔（D15–D17）；插件三卡与市场详情卡接共享 Modal（D18–D21） |
| 5 | `f0f29069`、`6ee7e615` | 四类关闭手势先收行内编辑器、标签不可聚焦（D22–D26）；安装卡选项行（D27） |
| 6 | `51b322cf`、`32d6e55f`、`e8fa9f8e` | 安装卡固定 footer（D28–D31）；`.guide` 引导面板（D32）；英文长文案下的 footer 回归 |
| 收口 | `a8da76b8`、`d49b6929`、`c2bed406` | 市场卡安全面板（D33）；深色安全面板回归；原生卡 200ms 入场（D34） |

更早的共享表面迁移（`b4f29a6e` 设置弹窗、`7170d61d` projects、`40d0bbfe` dirpick/workspacepicker、
`a9dc0220` 媒体/用量、`42cada12` 启动抽屉、`22654d76` 排期删除、`b92a5bae` assistant/market、
`6c74d134` onboarding、`21c6a522` 通用确认框）见 `full-audit.md` 与 `coverage-matrix.md`。

## 4. 残留清单

| ID | 残留 | 原因 / 影响 | 下一步 |
| --- | --- | --- | --- |
| R1 | `label-secondary` token 缺失 | 共享 `theme.Colors` 把 secondary 折到 tertiary；`installSafetyInk` 写死 `rgb(97,102,107)`/`rgb(207,211,214)`，自定义主题族下两行仍是固定灰 | 在 `theme.Colors` 补 secondary 并让 chrome/widgets 一并使用 |
| R2 | `.guide` 描边 0.12 代替参考的 0.16（border-l4） | 浅色下 0.5 DIP 发丝线比参考略淡 | 主题加 `BorderL4`，面板改用 |
| R3 | 检查中的“取消检查并返回编辑” | 参考 footer 只有禁用的“正在检查…”；原生保留正文这条退路防止卡死 | 保持；若参考后续补取消入口再对齐 |
| R4 | 快捷键底栏 `重新读取` | 参考只有 `恢复全部默认` + 计数；原生在草稿脏/读取失败时显示，用于丢弃草稿或重读被外部改写的配置 | 保持为原生恢复路径；已在矩阵注明 |
| R5 | 快捷键分组标题 | 参考四组（应用/输入/菜单/审批），原生只有应用组命令，按“空组不显示标题”只渲染应用组 | 有了输入/审批命令再补组标题 |
| R6 | 其余 🟡 弹窗（projects 页面弹窗、request_changes、chat 图片画廊/可视回复/引用/反馈、usage 价格、boot 日志、composer 附件） | 都是原生功能，参考无同款；已接共享表面/20px 媒体层与 200ms 淡入，但没有逐项 1440×920 参考对照 | 逐卡补 1440 证据即可，无参考差异可修 |
| R7 | 实机窗口截图 | Host provenance/visual-readiness 门禁；150% DPI 下窗口超出屏幕、UIA 坐标不可靠 | 门禁允许时替换无头证据（安装卡、快捷键弹窗、projects 卡） |
| R8 | 插件启用/安装、市场安装、Git 提交、排期创建、工作区文件操作未执行 | 会改本机环境，只验证到打开/取消 | 在可回滚环境执行一次并记录 |
| R9 | 菜单子菜单只画 `›` 未展开；真实鼠标坐标注入不稳定 | 第 1 阶段遗留，点击路径由 accessibility action 覆盖 | 需要真实指针注入方案 |
| R10 | 市场确认卡的动作仍在正文（`取消/安装`），不是 footer | 参考无市场卡；三张卡内部结构不完全一致 | 若要求三卡同构，可把动作移入 `pluginDialog` footer |

## 5. 验证

```sh
go test ./internal/ui/... -count=1          # 34 个包全部 ok（custompanel 无测试）
go test ./internal/ui/pluginmanager/ -count=1
go test ./internal/ui/projects/ ./internal/ui/dirpick/ ./internal/ui/workspacepicker/ ./internal/ui/chrome/ -count=1
go vet ./internal/ui/pluginmanager ./internal/ui/chrome ./internal/ui/projects
go build ./internal/... ./cmd/...           # 退出码 0
```

`gofmt -l` 在本仓对所有 CRLF 文件都会报（历史行尾），不作为门禁；改动文件编译与 `go vet` 干净。

证据（`docs/review-20261022/evidence/`，SHA-256 前 16 位）：

| 文件 | 字节 | SHA-256 |
| --- | --- | --- |
| `plugins-market-choice-1440.png` | 51494 | `DBCEC4C86BC4BE77` |
| `plugins-install-idle-1440.png` | 53719 | `F9B1068198A3EB4A` |
| `plugins-install-dark-1440.png` | 51246 | `8E0AC6BE6EB5943C` |
| `projects-rename-1440.png` | 41481 | `85663C8C8D28F7F1` |
| `projects-rename-fade-1440.png` | 40535 | `5B7DCB8752BA159B` |
| `projects-rename-narrow.png` | 17782 | `27E0B2D8D93C79A7` |
| `projects-archive-narrow.png` | 23034 | `8CD5956634DE5C6A` |
| `directory-picker-create.png` | 29389 | `B648A6D0911EFF82` |
| `workspace-rename.png` | 13223 | `D921C58B9BC8514B` |

关键几何（1440×920，DIP）：

- 市场卡：560×339.8 @ (440, 290.1)，安全面板文字起 498（+58）、466×36（两行）。
- 安装卡：560×507.9 @ (440, 206)；安全面板左 464、文字起 498；`.wide` 512×40 @ (464, 650)，
  底边恰为卡片底 −24；引导面板标题内缩 38、示例行与“填入示例”同一中线。
- 重命名卡：380 宽 @ x=530；遮罩 0.24 → 194/255；面板首帧未绘制，200ms 后落在 245。

## 6. 结论

参考源码里可判定的布局、颜色、层级、文案、弹窗材质与入场动效差异已收敛为 34 条并全部有回归或证据；
剩下的 10 条是 token 缺口、原生独有的恢复路径、无参考对应物的原生弹窗，以及实机窗口/真实执行受环境限制的部分。
