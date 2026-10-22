# 第 6 阶段：插件安装卡的固定 footer 与引导面板

日期：2026-10-22
参考：`reference/electron/vendor/deepseek-harness/packages/client/ui-plugin-manager/src/client/PluginManagerPage.tsx`
（`:954-974` 的 `Modal` 只把 `header`/`description`/`children` 放进 `contentClassName={installContent}`，而 `footer` 是 Modal 的独立节点，
内容为 `.installFooter`：`<p className={css.installSafety} role="note">` 的两行安全说明 + `<Button variant="primary" className={css.wide}>`）、
`.../PluginManagerPage.module.css`（`:518-522` `.installContent{min-height:0;overflow-y:auto}`；`:531-537` `.installFooter` 20 DIP 列间距；
`:637-661` `.installSafety` 12 圆角、`padding:10px 12px`、warn-tertiary 底色、14 DIP 图标 `margin-top:2px`、`.installSafetyText` 4 DIP 行距；
`:864-869` `.wide{width:100%;height:40px;justify-content:center;border-radius:12px}`）、
`.../packages/client/ui-primitives/src/Modal.module.css`（`:136-142` `.footer` 横向 24 内边距、`.dialog` 底部 24 内边距、内容与 footer 之间 20 间距）、
`.../src/client/locales.ts`（`:77-78` `installGuideSafety`/`installUpgradeNotice`、`:90-91` `installRun`/`installChecking`）、
`.../src/icons/index.tsx`（`:590-601` `IconWarningOutlineRegular`：16 DIP 圆圈 + 感叹号，1 DIP 描边）。
当前：生产渲染 + `ui.Tester` 几何/像素事件（`internal/ui/pluginmanager/install_card_footer_test.go`）。

## 1. 差异清单

| ID | 参考（源码声明） | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D28 | 安全说明与主按钮在 Modal 的 `footer` 槽里，`.installContent` 滚动时 footer 不动（`PluginManagerPage.tsx:960-974`、`PluginManagerPage.module.css:518-522`） | 两者都在 `ui.Scroll` 的正文末尾，随输入区/引导面板一起滚动；卡片被窗口夹短时主按钮会被裁到卡片外 | “确认来源后点安装”这一步会随内容滚走；短窗口下根本点不到 |
| D29 | `.installSafety`：warn-tertiary 底色（`rgb(254,245,231)` / 深色 `rgb(39,36,31)`）、12 圆角、`padding:10px 12px`、14 DIP 警告图标、两行 12/18 说明（`PluginManagerPage.module.css:637-661`） | 一行 12 DIP 纯文本、无底色无图标，文案是自写的短句 | 安全提示在视觉上不成立，用户读不到“来源不明会损坏 Whale Isle / 泄露数据”与“暂不支持自动更新” |
| D30 | 主按钮是 `.wide`：整宽、40 高、12 圆角、`variant="primary"`（单色主按钮）（`PluginManagerPage.module.css:864-869`） | 用 `ui.PrimaryButton`（主题强调蓝）夹在正文按钮流里，宽度随文字 | 同一张卡里两种主按钮语言；按钮宽度/高度与参考的整宽主操作不符 |
| D31 | 检查中按钮文案换成 `installChecking`“正在检查…”并禁用，前面带 ongoing 状态点（`PluginManagerPage.tsx:969-972`） | 正文另起一行 `正在检查…` 状态文本 + 一个“取消检查并返回编辑”按钮，主按钮消失 | 主操作位置在检查中空掉，状态与动作分离 |
| D32 | 展开后的引导是 `.guide` 面板：0.5 描边 + 16 圆角 + `bg-layer-1` 底、`padding:8px 14px`、20 DIP 列间距；内含 `.guideList`（`.guideItem`：标题 13/20/500、`.guideHint` 12/18 上方 6 DIP、`.guideExample`“示例：”+ 等宽字体包名、右侧 `variant="outline" size="sm"` 的“填入示例”按钮，`align-self:center`）（`PluginManagerPage.tsx:1032-1059`、`PluginManagerPage.module.css:619-711`） | 标题、说明与一个**以示例本身为文案**的普通按钮平铺在正文里，没有面板、没有“示例：”标签、没有等宽包名，也没有独立动作 | 引导看起来像三行说明而不是可操作的示例卡；“填入示例”这个动作在界面语言里不存在 |

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/pluginmanager/dialogs.go` | `pluginDialog` 增加 `footer func()`（nil = 无 footer），footer 作为卡片最后一行、`Shrink(0)`，位于滚动体之外；`installDialog` 在 `idle`/`checking` 且非 `githubRecovery`、非 `Confirmed` 时挂上 `installFooter`，并把正文滚动体的底部内边距从 24 改为 0（由 footer 的 20 + 24 承担）；正文移除安全说明文本、`正在检查…` 状态行与 `安装` 主按钮，只保留“取消检查并返回编辑”这条原生恢复路径。 |
| `internal/ui/pluginmanager/install_footer.go` | 新增 `installSafetyNote`（`.installSafety`：12 圆角 warn 面板、10/12 内边距、14 DIP `IconWarningOutlineRegular`、`margin-top:2`、两行 12/18、4 DIP 行距）、`installSafetyCopy`（参考两句安全/升级文案）、`installWarnFill`/`installSafetyInk`（warn-tertiary / label-secondary 的四个 token 值）、`installFooter`（`padding:20 24 24`、20 DIP 列间距、`.wide` 整宽 40 高 primary；检查中换 `正在检查…` + `widgets.StateDot` 并禁用）。 |
| `internal/ui/pluginmanager/install_card_footer_test.go` | 新增 6 条回归：footer 在两个窗口高度下与卡片底边保持同一距离（且正文真的溢出）、`.installSafety` 的 58 DIP 文字内缩/4 DIP 行距/warn 底色/12 圆角像素、`.wide` 512×40、检查中只出现 `正在检查…`、日志页没有 spec 页的 footer、英文长文案下 footer 仍把主按钮留在卡片内。 |
| `internal/ui/pluginmanager/dialogs_surface_test.go` | 空态断言改到参考文案（`请确认插件来源可信`、`暂不支持自动更新`）。 |
| `internal/ui/pluginmanager/install_registry.go` | `installGuidePanel` 改为 `.guide` 面板（0.5 描边 `BorderL3`、16 圆角、`colors.Field` 底、`padding:8 14`、`gap:2`），拆出 `installGuideItem`（`.guideItem`：`padding:8 0`、10 DIP 间距、`align-items:center`；标题 13/20/500、`.guideHint` 12/18 + `Margin(6,0,0,0)`、示例行“示例：”+ `Font("monospace")` 包名）与 `installGuideExample`（参考 `GUIDE_EXAMPLES` 唯一一条的标题/提示/示例）；右侧改为 `widgets.Button` 的 outline/sm“填入示例”，无障碍名 `填入示例 <包名>`，点击仍是 `EditInstall`。镜像恢复提示留在面板下方。 |
| `internal/ui/pluginmanager/install_guide_panel_test.go` | 新增回归：标题内缩 38（正文 24 + 面板 14）、标题 20 高、提示距标题 6、示例行距提示 0 且与标签同一行、填入按钮 28 高/右缘落在面板 14 内边距/与整行同为中线、面板底色为 `rgb(245,246,247)` 且与卡片白底不同；顺带产出 `plugins-install-guide-1440.png`。 |

## 3. 几何（1440×920，DIP）

| 项目 | 参考声明 | 当前实测 |
| --- | --- | --- |
| 卡片 | 560 宽、内容自适应 | 560×507.9 @ (440, 206.0) |
| 安全面板 | footer 内、左右各 24 内缩 | 左 464（卡片 +24），文字起于 498（+24 padding 12 + gap 8 + 图标 14 = +58） |
| 安全文案 | 12/18 两段、行距 4 | 每段 466 宽、36 高（各两行），第二段比第一段低 40 |
| 面板底色/圆角 | warn-tertiary / 12 | `rgb(254, 245, 231)`；(466, 534) 处角点为卡片底色 |
| 主按钮 | `.wide` 整宽 40 高 | 512×40 @ (464, 650)，底边 690 = 卡片底 713.9 − 24 |
| 说明与按钮间距 | 面板 10 底距 + footer 20 间距 | 30（实测差 ≤1.5） |

### 引导面板（D32，1440×920 展开态，DIP）

| 项目 | 参考声明 | 当前实测 |
| --- | --- | --- |
| 面板 | 0.5 描边、16 圆角、`bg-layer-1` 底、`padding:8px 14px` | 左 464（正文 24）；文字左 478.5（卡片 +38.5）；标题顶 418，面板顶 410（+8） |
| 标题 | 13/20/500 | 404×20，500 字重、主要色 |
| 提示 | 12/18，上方 6 | 404×36（两行），距标题底 6 |
| 示例行 | `示例：` + 等宽包名，与提示同级 0 间距 | 标签 36×20、包名 140.6×20，同一行；距离提示底 0 |
| 填入按钮 | `outline`/`sm`，`align-self:center` | 69×28 @ (892.5, 445)，右缘 961.5 = 卡片右 1000 − 24 − 14；中线 459 与三行文字（418–500）一致 |
| 面板底色 | `--dsw-alias-bg-layer-1` | `rgb(245, 246, 247)`（本仓 `colors.Field`），与卡片白底不同 |

## 4. 验证

```sh
go test ./internal/ui/... -count=1        # 34 个包通过，custompanel 无测试
go test ./internal/ui/pluginmanager/ -count=1
go vet ./internal/ui/pluginmanager        # 无输出
go build ./internal/... ./cmd/...
```

修复前复现：把 `internal/ui/pluginmanager/dialogs.go` 退回上一提交后重跑新增回归，四条失败——`the trust note is missing`（正文里仍是旧的短句 `请确认来源可信。插件以当前 profile 的权限运行。`，`dsh-plugin-whale-pet` 之后才是它），
`the checking card lost the trust note`（检查中仍是 `正在检查…` 状态行 + `取消检查并返回编辑`，没有 footer）。
引导面板同理：把 `dialogs.go` 与 `install_registry.go` 一起退回后，`the guide's title starts 24.0 DIP into the card, want the body's 24 plus the panel's 14`。修复后全部通过。

像素侧核对：

- `plugins-install-idle-1440.png` 重新生成（44068 → 53719 字节）：新增 warn 面板与整宽 40 高按钮。
- `plugins-install-guide-1440.png` 新增（64433 字节）：展开态的面板、示例行与“填入示例”。
- `plugins-sources-1440.png`、`plugins-market-choice-1440.png`、`plugins-install-failed-1440.png` 重渲染后与既有证据**逐字节一致**（SHA-256 相同），说明 `pluginDialog` 的 footer 参数没有影响其三条调用路径。

## 5. 残留

1. **`label-secondary`**：共享 `theme.Colors` 没有这个 token（本仓把 secondary 折到 tertiary），`installSafetyInk` 直接写 `rgb(97,102,107)` / 深色 `rgb(207,211,214)`；自定义主题族下这两行仍是固定灰。
2. **面板描边**：`.guide` 的 `border-l4`（0.16）用 `BorderL3`（0.12）代替，0.5 DIP 发丝线在浅色下比参考略淡。
3. **检查中的恢复路径**：参考 footer 只有禁用的“正在检查…”，没有“取消检查并返回编辑”；原生保留这条按钮作为卡死时的退路，位置在正文。
4. **市场选择卡的安全线**：`确认安装插件` 卡（原生功能，参考无对应弹窗）正文仍是一行旧短句；两张卡现在用了两种安全文案，下一轮统一。
5. **`重新读取` 与底栏语义**、**其余 🟡 弹窗**、**分组标题**：仍沿用第 5 轮残留（未在本轮处理）。
6. **实机截图**：仍以 `ui.Tester` 渲染为证据，未取得实机窗口截图。
