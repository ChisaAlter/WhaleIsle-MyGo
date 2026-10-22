# 第 4 阶段：弹窗表面统一与快捷键弹窗收尾

日期：2026-10-22
参考：`reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src/Modal.module.css`（通用 Modal：`--dsw-radius-panel: 28px`、`--dsw-alias-bg-mask-1: .24/.5`、`modalEnter` 200ms 淡入、`padding: 22px 14px 12px 24px` 头、24 内边距体）、
`.../packages/client/ui-plugin-manager/src/client/PluginManagerPage.module.css`（`.installDialog { width: min(560px,100%); max-height: min(800px,100%) }`）、
`.../packages/client/ui-shortcuts/src/client/Reference.module.css`（`translateY(30px)`、`.rowActions`、`.inlineAction`）。
当前：`docs/review-20261022/evidence/settings-shortcuts-dialog-1440.png`、`settings-shortcuts-hover-1440.png`、
`plugins-install-idle-1440.png`、`plugins-install-failed-1440.png`、`plugins-sources-1440.png`、`plugins-market-choice-1440.png`（均 1440×920 生产渲染）。

## 1. 差异清单

| ID | 参考（源码声明） | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D15 | 快捷键弹窗在居中位置再 `transform: translateY(30px)`（`Reference.module.css:59`），1440×920 下 600 DIP 面板顶部 190 | `DialogBase` 严格居中，实测顶部 161 | 弹窗整体偏高 30 DIP |
| D16 | 每个可编辑行在键位胶囊左侧有 20×26 的 `.rowActions`，内含 12 DIP 铅笔，`opacity: 0`，行悬停时显示（`Reference.module.css:190-194`） | 行悬停只换底色，没有“可编辑”的铅笔提示 | 悬停反馈缺少参考的编辑暗示 |
| D17 | 行内编辑器只有 `恢复默认`、`移除`、录键器；说明文字只在 sr-only 中；离开行/选择别的行即收起（`Editor.tsx:163-180`、`Reference.tsx:155-159`） | 录键器下多一个 `取消` 按钮，另有一行可见说明 `松开组合键即保存。` | 行内多两个控件，行高被撑开，与参考的行内节奏不同 |
| D18 | 插件安装弹窗走通用 Modal：`width: min(560px,100%)`、`max-height: min(800px,100%)`、28 圆角、淡入、头部 22/14/12/24 + 28 DIP 关闭按钮、体 24 内边距（`PluginManagerPage.module.css:510-516`、`Modal.module.css:34-49`） | 640×720 固定尺寸、20 圆角、手写 `.35` 遮罩、无淡入（`backdrop.Background` 被 `DialogDecor` 覆盖为 `.24` 但不渐变）、头部 20/20/12/20 + 文字按钮 `关闭` | 插件安装弹窗与其它弹窗材质、尺寸、头部节奏都不一致 |
| D19 | 同一 Modal 语言；标题在头部，正文不再重复标题 | `插件索引源`：560 宽内容自适应但 20 圆角、无淡入、底部额外 `关闭` 按钮、正文里再画一层带标题的卡片（标题重复） | 弹窗像内嵌页面，标题重复 |
| D20 | 同一 Modal 语言；头部关闭 + 底部右对齐 `取消/安装` | `确认安装插件`：20 圆角、无淡入、无头部关闭、整体 20 内边距 | 与参考的确认卡结构不同 |
| D21 | 通用 Modal：28 圆角、`--dsw-alias-bg-mask-1`、200ms 淡入、22/14/12/24 头部 + 28 关闭盒、24 体边距、elevation-stroke（`Modal.module.css:19-49`、`:124-130`） | `market/details` 已是 28 圆角 + 玻璃 + 淡入，但遮罩写死 `.35`、头部 16 等距 + 文字按钮（重新加载/关闭）、关闭不是 28 盒、体边距 16、并自己再画一条 12 半径描边 | 详情卡与其它弹窗的遮罩深度、头部节奏和描边半径不一致 |

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/chrome/dialog_surface.go` | `DialogSurfaceBox` 增加 `OffsetY`：在居中基础上用 `Margin(2*OffsetY,0,0,0)` 下移，40 处既有弹窗保持 0 不受影响 |
| `internal/ui/settings/shortcuts.go` | 弹窗 `OffsetY: 30`；头部关闭按钮 `Margin(-2,0,0,0)` 使其与标题共享中线（参考 20 vs 22）；行增加 20×26 悬停铅笔槽（隐藏时仍占位，胶囊不位移）；行内编辑器移除 `取消` 与可见说明 |
| `internal/ui/settings/shortcut_recording.go` | 移除录键器的 `取消` 控件，保留 `审阅/重试` 两条失败恢复动作 |
| `internal/ui/settings/shortcut_icons.go` | 新增参考 12 DIP 铅笔（`ui-shortcuts/src/client/Icons.tsx` 的 `edit` 路径原样） |
| `internal/ui/pluginmanager/dialogs.go` | 新增 `pluginDialog`：共享表面 + 参考头部（22/14/12/24、16/500 标题、28 DIP 关闭盒 14 内距）+ 560 宽内容自适应（上限 `h-48`）；安装卡片 640→560、20→28 圆角、改为淡入/玻璃/投影/0.5 描边；索引源与市场确认卡片同迁移，索引源去掉底部 `关闭` |
| `internal/ui/pluginmanager/view.go` | `sourcesEditor` 去掉重复标题和内层边框卡片 |
| `internal/ui/market/presentation.go` | 详情卡改走 `DialogSurfaceSized`（680 宽、`min(720,h-48)` 高、共享淡入/动态遮罩/投影/0.5 描边）；头部 22/14/12/24 + 28 关闭盒；`重新加载详情` 移入正文；删除私有 12 半径描边 |
| `internal/ui/settings/shortcut_dialog_affordance_test.go` | 新增：30 DIP 下移与关闭盒几何、悬停铅笔显隐/位置、行内无取消与可见说明、切换行/关行不写盘、悬停态截图 |
| `internal/ui/settings/shortcut_entry_test.go` | 跟随移除可见说明（改用录键器与行内动作存在性判断编辑态） |
| `internal/ui/pluginmanager/dialogs_surface_test.go` | 新增四张卡片的共享表面回归：560 宽/居中/24 让位、28 关闭盒与 14 右距、内容自适应高度、日志态不溢出、截图 |

## 3. 几何对照（1440×920，DIP）

| 项目 | 参考声明 | 当前实测 | Δ |
| --- | ---: | ---: | ---: |
| 快捷键弹窗外框 | 480×600 @ 顶部 190 | 478×598 @ (481,191) | ≤1（0.5 描边） |
| 快捷键弹窗标题区 | 46 | 46（191→237） | 0 |
| 关闭盒 | 28×28，顶部 20、右 14 | 顶部 210（=191+19.5）、右 14 | ≤1 |
| 检索区 | 60 | 60（237→297） | 0 |
| 行 | 42，胶囊 26 | 42（连续 11 行），胶囊 26 | 0 |
| 悬停铅笔槽 | 20×26，与胶囊间距 8，图标 12 | 20×26，胶囊文字距槽右 20（8+12） | 0 |
| 底部 | 52 | 52（736→788） | 0 |
| 插件卡宽度 | min(560, 100%) | 560 @ x=440 | 0 |
| 安装卡（空态） | 内容自适应，上限 800 | 560×334 @ y=293 | — |
| 安装卡（失败/日志） | 内容自适应，上限 800 | 560×276 @ y=322（无空白工作高度） | — |
| 索引源卡 | 同上 | 560×192 @ y=364 | — |
| 市场确认卡 | 同上 | 560×262 @ y=329 | — |
| 插件卡头部 | 22/14/12/24，关闭盒顶部 22、右 14 | 关闭盒顶部 22、右 14、28×28 | 0 |
| 行情详情卡（1000×800） | 680 宽、`min(720,h-48)` 高、24 体边距 | 680×720 @ (160,40) | 0 |

测量：`tmp/measure_shortcut_dialog.py`（快捷键弹窗）；插件卡片由白色像素外接框测得（外框 560 宽）。

## 4. 验证

```sh
go test ./internal/ui/... -count=1      # 34 个包通过，custompanel 无测试
go test ./internal/shortcuts -count=1   # 通过
go vet ./internal/ui/settings ./internal/ui/chrome ./internal/ui/pluginmanager ./internal/shortcuts
go build -o whaleisle-pass4.exe ./cmd/whaleisle
```

新增回归：`TestShortcutDialogDropsDownLikeTheReference`、`TestShortcutDialogRowHoverRevealsEditPencil`、
`TestShortcutInlineEditorHasNoCancelControl`、`TestShortcutDialogHoverSnapshot`、
`TestInstallDialogUsesTheSharedModalSurface`、`TestInstallDialogSizesTheLogScreenToItsContent`、
`TestSourcesDialogUsesTheSharedModalSurface`、`TestMarketChoiceDialogUsesTheSharedModalSurface`。
行情详情沿用 `TestMarketDetailsSnapshotFixture`、`TestMarketDetailsDrawsItsHairlineInTheDark`、`TestMarketReadmeStaysInsideTheDetailsPanel`；
快照改为先推进叠加层时钟再落图（`market-details-wide.png` 不再停在淡入第 0 帧）。
`go vet` 无输出；`go build` 成功。

## 5. 残留

1. ~~**Esc 先收行内编辑**~~：第 5 轮已收口，关闭盒/Esc/遮罩/重置四类手势都先收编辑器（`round5-close-gestures.md`）。
2. ~~**点外收起**~~：第 5 轮已收口，遮罩点击先收编辑器、第二次才关弹窗；行标签仍可点击收起但不再可聚焦。
3. **`重新读取`**：参考底部只有重置与计数；原生在草稿脏或读取失败时仍显示 `重新读取`，保留配置被外部改写的恢复路径。
4. **分组标题**：参考有应用/输入/菜单/审批四组；原生没有输入与审批命令，按“空组不显示标题”只渲染应用组。
5. **安装卡的操作区**：参考把安全说明与主按钮放在固定 footer；原生仍把动作放在滚动体内（视觉一致，滚动时按钮会随内容移动）。
6. ~~**安装引导/索引源开关**~~：第 5 轮已收口，空态选项行按 `.optionsRow`/`.guideToggle`/`.registryToggle` 改成左侧 12.5 文字开关（带 160ms 旋转箭头）与右侧 28 高描边 chip（`round5-close-gestures.md`）。
7. **`settings/onboarding`**：参考本身是全屏 overlay（`OnboardingSurface.module.css`：`position: fixed; inset: 0`，无圆角），原生也是 100%×100%、圆角 0，故覆盖矩阵改标 ➖，不再作为“待接共享表面”。
8. **实机截图**：仍以生产 `ModalView`/`ui.Tester` 无头渲染 + 测试为证据；实机窗口受 Host 门禁与 DPI 限制未取得。
