# 快捷键弹窗对齐（第 3 阶段）

日期：2026-10-22
参考：`reference/electron/vendor/deepseek-harness/packages/client/ui-shortcuts/src/client/Reference.module.css`、
`.../Reference.tsx`、`.../locales.ts` 与已安装包 `C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\ui-shortcuts\lib\types\client\locales.js`
当前：`docs/review-20261022/evidence/settings-shortcuts-dialog-1440.png`（列表态）、
`settings-shortcuts-editor-1440.png`（行内编辑态），均为生产 `ModalView` 无头渲染，1440×920。

上一轮只对齐了“设置·通用”页里的快捷键**入口行**（标题/说明/`编辑快捷键`按钮/弹窗打开方式），
弹窗内部仍是 MyGo 的页面式编辑器。本轮把弹窗内部改成参考的“速查列表”结构。

## 1. 差异清单

| ID | 参考（源码声明） | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D8 | 弹窗 `width:min(480px,100%); height:600px; radius:24px; padding:0`；标题区 46 DIP，标题 16/500，右上 28×28 关闭按钮（`Reference.module.css:52-77`） | `bodyDialog` 通用壳：720×720 上限、`radius 20`、内边距 20，标题右侧是文字按钮 `关闭` | 弹窗尺寸、圆角、头部节奏与参考不同 |
| D9 | 每行 42 DIP：命令名 13/18 + 右侧 26 DIP 键位胶囊（圆角 10、`bg-module-platform`）；点击行就在原地展开录键器（`Reference.module.css:173-206`） | 每个命令一个常开区块（约 150 DIP）：`已保存：…；默认：…` 一行、整宽文本输入、`录制按键…`、`解绑`、`使用默认值`、错误行 | 列表被撑长，37 条命令无法扫读，与参考的“速查表”交互完全不同 |
| D10 | 检索区 60 DIP：36 DIP、圆角 12 的输入框，左侧搜索图标，占位 `搜索快捷键`，非空时右侧圆形清除按钮（`Reference.module.css:95-127`） | 普通 `TextInput`（自带边框/内边距），占位 `搜索命令或快捷键`，无图标、无清除 | 检索区高度与视觉不一致 |
| D11 | 无页内说明段；`应用操作`分组不显示组标题（`Reference.tsx:145-153`）；空状态 `没有匹配的快捷键` | `应用与主页快捷键` 标题 + 两段长说明 + `应用缩放：N%` 与三个缩放按钮 + 冲突汇总句 + 每条命令的帮助文字 | 弹窗像设置页而不是参考的速查弹窗，内容被挤到滚动区外 |
| D12 | 底部 52 DIP：左 `恢复全部默认`（12px、刷新图标、无修改时禁用），右 `N 项已自定义`（`Reference.module.css:137-164`） | 页面末尾 `放弃草稿并刷新` / `保存快捷键` / `恢复全部默认值…` 三个普通按钮 | 底部动作与参考不同，且没有“已自定义 N 项”提示 |
| D13 | 重置确认标题 `恢复全部默认快捷键？`，说明“将恢复当前平台的默认快捷键。你修改或移除的快捷键都会恢复，其他平台不受影响。”，按钮 `取消` / `恢复默认`（`locales.js:14-16`） | 说明为“确认恢复全部应用和主页绑定并立即保存？将替换已知命令的自定义绑定与未保存草稿，未知命令保留。”，按钮 `确认恢复并保存` | 文案漂移 |
| D14 | 每行操作即改即存（`save({type:'set'|'reset'|'clear'})`），成功后关闭行内编辑器并提示 `已修改`；没有全局保存按钮 | 行内 `移除/使用默认值` 立即保存，但录键和普通编辑要靠底部 `保存快捷键`；失败时全局 `e.Err` | 交互模型不一致，且非共享编辑器行内保存不过冲突校验 |

参考文案来源：`ui-shortcuts/lib/types/client/locales.js`（`title`、`description`、`search`、`close`、`reset-all`、`modified-count`、`reset-title`、`reset-description`、`cancel`、`reset`、`unbound`、`empty`、`record`、`record-help`）。

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/chrome/dialog_surface.go` | 抽出 `DialogSurfaceBox{Width,Height,Radius,Padding}` 与 `DialogSurfaceSized`；原 `DialogSurface`/`DialogSurfaceTheme` 以 `Radius 28`、无固定高度继续工作，40 处既有弹窗不受影响 |
| `internal/ui/settings/shortcuts.go` | 重写弹窗：480×600、圆角 24、无内边距；46 DIP 标题区（16/500 + 28×28 关闭）、60 DIP 检索区（36 高、圆角 12、搜索图标、`搜索快捷键`、清除按钮）、42 DIP 行（13px 标签 + 26 DIP 键位胶囊）、52 DIP 底部（`恢复全部默认` + `N 项已自定义`）；点击行进入行内编辑器（`恢复默认`/`移除`/录键器），保存成功自动收起；仅在脏/读取失败时出现 `重新读取` |
| `internal/ui/settings/shortcut_recording.go` | 录键器改为参考的 26 DIP 键位输入样式（`按下快捷键` / `按下快捷键…`），保存成功回调关闭行内编辑器 |
| `internal/ui/shortcuts/recording.go` | `SaveRow` 对非共享编辑器也做整套冲突校验，确保行内录键不会写入冲突键位 |
| `internal/ui/settings/interface.go` | 移除“界面”页内嵌的整块快捷键编辑器（参考只在“通用”注册 `settings.general.item` 入口行） |
| `internal/ui/settings/shortcut_entry_test.go` | 重写为弹窗级回归：参考结构与旧文案、行内录键单行保存、`移除/恢复默认` 单行落盘、失败保留接受值、底部计数与重置确认、检索与空状态、1440×920 两态截图 |
| `internal/ui/settings/general_reference_test.go`、`shortcut_reload_test.go`、`physical_recording_release_test.go` | 入口/关闭/重新读取/录键器断言跟随新结构 |

## 3. 几何对照（1440×920，DIP）

测量：`tmp/measure_shortcut_dialog.py`（生产渲染 PNG 的对话框外框、文字带起始 y 与行距）。

| 项目 | 参考声明 | 当前实测 | Δ |
| --- | ---: | ---: | ---: |
| 弹窗外框 | 480×600 @ 居中 | 478×598 @ (481,161) | ≤1（0.5 描边） |
| 圆角 | 24 | 24 | 0 |
| 标题区高 | 46 | 46（161→207） | 0 |
| 检索区高 | 60 | 60（207→267） | 0 |
| 列表首行标签 | 约 281 | 281..292 | 0 |
| 行距 | 42 | 42（连续 11 行实测均为 42） | 0 |
| 键位胶囊高 | 26 | 26 | 0 |
| 底部高 | 52 | 52（706→758） | 0 |
| 底部文案 y | — | 701..707 | — |

## 4. 验证

```sh
go test ./internal/ui/settings -count=1
go test ./internal/shortcuts -count=1
go test ./internal/ui/... -count=1
go vet ./internal/ui/settings ./internal/ui/chrome ./internal/shortcuts
go build -o whaleisle-pass123.exe ./cmd/whaleisle
```

结果：`internal/ui/...` 全部包通过（首次全量跑时 `widgets/TestStateDotRunningBreathes` 曾因并行负载失败一次，单独与复跑均通过）；`internal/shortcuts` 通过；`go vet` 无输出；`go build` 成功。
截图：`settings-shortcuts-dialog-1440.png`、`settings-shortcuts-editor-1440.png`。

## 5. 同轮附带：特效设置弹窗改用共享表面

覆盖矩阵里 `settings/effect_preferences` 一直标 ⛔（“16px、无玻璃”）。实际代码已是 28 DIP 圆角，但面板直接 `Background(theme.Background)+Clip`，遮罩是手写黑色层：没有共享弹窗的淡入、玻璃材质、菜单投影与 0.5 DIP 外描边，和参考弹窗的材质不一致。

- `internal/ui/settings/effect_preferences.go`：面板改为 `chrome.DialogSurfaceTheme(...)`，保留 620×min(760, 85% 视口) 的尺寸，去掉手写遮罩。
- 证据：`docs/review-20261022/evidence/settings-effect-dialog.png`（900×950 无头渲染，含合成保存失败状态）。
- 覆盖矩阵对应行改为 ✅。
- 验证：`go test ./internal/ui/settings`（含 `TestEffectEditorAtomicSaveRetainsFailureAndChangesHost`、`TestInterfaceGearOpensTheEffectDialog`）通过。

## 6. 残留

1. **弹窗垂直位置**：参考在居中基础上 `translateY(30px)`（`Reference.module.css:59`），当前 `DialogBase` 严格居中，实测顶部 161 vs 参考 191。需要共享弹窗支持统一偏移后再改，未在本轮动 `DialogBase`。
2. **行悬停提示**：参考悬停时在键位胶囊左侧淡入铅笔图标（`.rowActions`），当前只换行底色（悬停 `Tooltip` 说明动作）。
3. **退出行内编辑**：参考支持点击别处 / Esc 关闭；当前靠再次点击行标签或保存成功后自动收起，录键中额外提供 `取消`（多一个控件）。
4. **`重新读取`**：参考底部只有重置与计数；当前在草稿脏或读取失败时额外显示 `重新读取`，保留原生“配置被外部改写”的恢复路径。
5. **分组**：参考有 `应用操作/消息输入/菜单与弹层/审批区域` 四组，原生没有消息输入与审批区命令，按参考“空组不显示标题”的规则只渲染应用操作组，因此列表没有组标题。
6. **应用缩放簇**：从弹窗移除；能力仍由 `window.zoomIn/zoomOut/actualSize`（`Mod+=`/`Mod+-`/`Mod+0`）提供，并在同一弹窗列表中可见。
7. **实机截图**：本轮仍未取得 MyGo 实机窗口图（Host 门禁与 150% DPI 老问题），证据为生产 `ModalView` 无头渲染 + 测试。也尝试直接截取运行中的鲸屿本体（0.3.9）快捷键弹窗：窗口可 `SetWindowPos`，但侧栏 `账号菜单` 的 UIA 元素不支持 `InvokePattern`，无法在不注入鼠标的情况下进入设置；改用 `Ctrl+,` 发送按键也未送达前台。故参考侧仍以源码 CSS/文案为准。
8. 覆盖矩阵中其余待迁移弹窗：`pluginmanager` 安装/来源/市场选择、`settings/onboarding`、`market/details`。
