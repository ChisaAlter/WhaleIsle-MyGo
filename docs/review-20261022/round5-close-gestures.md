# 第 5 阶段：快捷键弹窗的关闭手势与插件安装卡的选项行

日期：2026-10-22
参考：`reference/electron/vendor/deepseek-harness/packages/client/ui-shortcuts/src/client/Reference.tsx`
（`:96-108` `closeReference`：先收行内编辑器、再关弹窗；`:155-159` 内容区 `onPointerDownCapture` 在非交互区域按下时 `closeEditor()`；`:162` 关闭盒与 `:204-210` 底栏重置都调用同一关闭入口，重置前 `setTarget(null)`）、
`.../packages/client/ui-primitives/src/Modal.tsx`（`:70` 遮罩 `onClick={onClose}`）、`Editor.tsx`（`:163-180` 行内控件只有 `恢复默认`/`移除`/录键器，标签本身不是控件）、
`.../packages/client/ui-plugin-manager/src/client/PluginManagerPage.module.css`（`:714` `.optionsRow` 两端对齐、`:723-759` `.registryToggle` 28 高带 0.5 描边、`:592-615` `.guideToggle` 12.5 文字开关 + 160ms 旋转的箭头）、`PluginManagerPage.tsx`（`:1005-1029` 两个开关的次序与文案）。
当前：生产渲染 + `ui.Tester` 键盘/指针事件（`internal/ui/settings/shortcut_dialog_affordance_test.go`、`internal/ui/pluginmanager/install_options_row_test.go`）。
本轮的弹窗表面、尺寸、行节奏沿用第 4 轮的几何结论，未再改动。

## 1. 差异清单

| ID | 参考（源码声明） | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D22 | 关闭盒调用 `closeReference`：`target !== null` 时先 `closeEditor()`，弹窗保持打开（`Reference.tsx:102-108`、`:162`） | 关闭盒直接置 `*open = false`，行内编辑器与弹窗一起消失 | 一次点击丢掉两层，与参考的两段式关闭相反；编辑器被强拆时不走 `CancelRecording` 的收敛 |
| D23 | Escape 由 Modal `onClose` 进入同一个 `closeReference`，同样先收编辑器（`Reference.tsx:153`、`Editor.tsx:89-95`） | `DialogBase` 的 backdrop `OverlayShortcut(0, KeyEscape)` 直接关弹窗 | Esc 一次退出两层；编辑中按 Esc 期望是“退出该行” |
| D24 | 遮罩 `onClick={onClose}` 也走 `closeReference`（`Modal.tsx:70`） | 点遮罩直接关弹窗 | 点外一次退出两层 |
| D25 | 底栏“恢复全部默认”在打开确认前 `setTarget(null)`（`Reference.tsx:206`） | `RequestReset` 只清 `Recording`，`target` 仍指向该行 | 确认框打开时其下仍是行内编辑器，Esc 的层级语义不清 |
| D26 | 编辑态的行标签是普通文本，不是控件，也不可聚焦（`Reference.tsx:188`） | 标签是 `ui.Button`，成为编辑态第 4 个 Tab 停靠点与一个可宣读控件 | 键盘用户在行内多停一站，与参考的 3 个控件（恢复默认/移除/录键器）不同 |
| D27 | 安装卡的空态：`.optionsRow` 一行两端对齐，左边是 `.guideToggle`（16 DIP 箭头 + 12.5 DIP 次要色文字，无描边无填充，展开时文案变“收起引导”、箭头 160ms 转 180°），右边是 `.registryToggle`（28 高、0.5 描边、圆角、选中名 500 字重主要色 + 末尾箭头） | 两个整宽 32 高描边按钮上下堆叠，文字居中，没有箭头；文案是“安装引导”“安装源：<名字>” | 选项行占掉两整行、像表单按钮而不是参考的开关；注册表选择没有“当前值”的层级 |

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/settings/shortcuts.go` | 新增 `shortcutCloseReference(open, e, target)`：有行内编辑器时只 `CancelRecording` 并清空 `target`，否则关闭弹窗；关闭盒改为调用它。弹窗内容在 `*target != ""` 时把 `Escape` 注册到 panel（非 overlay，位于焦点链上，优先于 backdrop 的关窗处理），并接管遮罩点击（`backdrop.Clicked()` 时收编辑器并恢复 `*open = true`，使第一下只退一层）。底栏“恢复全部默认”在 `RequestReset` 前清 `target`；清空后弹窗不再注册 panel 级 Escape，确认框独占 Esc。行内编辑器的标签由 `ui.Button` 改为不可聚焦的 `ui.Box` + `Clicked()`（保留“点标签收起编辑器”的指针路径，去掉多出的 Tab 停靠点）。 |
| `internal/ui/settings/shortcut_dialog_affordance_test.go` | 新增 6 条回归：关闭盒、Escape、遮罩、录键中 Escape、重置确认、标签非 Tab 停靠点。 |
| `internal/ui/pluginmanager/install_registry.go` | 拆出 `installOptionsRow`（`.optionsRow`：一行、两端对齐、可换行）、`installGuideToggle`（`.guideToggle`：16 DIP 箭头在前、12.5 DIP、无描边、悬停转主要色、展开换文案）、`installRegistryToggle`（`.registryToggle`：28 高、0.5 描边、圆角 10、次要色标题 + 500 字重主要色选中名 + 末尾箭头）；`installGuidePanel` / `installRegistryPanel` 承接展开内容与浮层。新增 `installChevronAngle`（160ms ease-out 旋转，`c.After(16ms)` 续帧）与参考 `IconChevronDownOutlineRegular` 的箭头路径。 |
| `internal/ui/pluginmanager/dialogs.go` | 空态调用改成选项行 + 引导面板 + 注册表面板三步；顺带修正该文件既有的 import 次序（gofmt）。 |
| `internal/ui/pluginmanager/install_options_row_test.go` | 新增 4 条回归：选项行几何（引导 18 高、左 24；注册表 28 高、右 24；同一中线）、引导展开换文案、箭头 160ms 的 0→180→0 缓动、注册表展开露出选项。 |

## 3. 手势对照

| 手势（行内编辑器已打开） | 参考 | 当前 |
| --- | --- | --- |
| 关闭盒 | 收起编辑器，弹窗保持；再点一次关弹窗 | 同左（`TestShortcutDialogCloseBoxClosesTheRowEditorFirst`） |
| Escape | 收起编辑器，弹窗保持；再按一次关弹窗 | 同左（`TestShortcutDialogEscapeClosesTheRowEditorFirst`、`...ArmedRecorderEscape...`） |
| 遮罩点击 | 收起编辑器，弹窗保持；再点一次关弹窗 | 同左（`TestShortcutDialogMaskPressClosesTheRowEditorFirst`） |
| 底栏“恢复全部默认” | 先清行内编辑器，再开确认框 | 同左（`TestShortcutDialogResetConfirmationKeepsEscape`） |
| 点行标签 | 编辑器收起（指针落在非交互区域） | 同左，但标签已不再可聚焦（`TestShortcutInlineEditorLabelIsNotATabStop`） |

Esc 在录键器持有焦点时同样只退一层：MyGo 先把已注册为 Shortcut 的按键送给 panel，再轮到聚焦元素的 `HandleInput`，所以录键器不会把 Escape 当成非法组合键并弹出“审阅/重试”两条恢复动作。

### 安装卡选项行（D27，1440×920，DIP）

| 项目 | 参考声明 | 当前实测 |
| --- | --- | --- |
| 选项行 | `.optionsRow`：一行、`space-between`、可换行 | 一行，引导开关与注册表 chip 同中线（中心差 ≤1） |
| 引导开关 | `.guideToggle`：12.5 文字、无边框、自左对齐、箭头在前 | 132.5×18 @ x=464（卡片左内边距 24） |
| 引导文案 | `installGuideToggle` / `installGuideHide` | “插件安装引导和示例” / “收起引导” |
| 箭头 | 16 DIP 图标，展开时 `rotate(180deg)`，`160ms ease` | 0°→180° 缓动 160ms（`TestInstallChevronAngleEasesOverTheReferenceDuration`） |
| 注册表 chip | `.registryToggle`：28 高、0.5 描边、圆角、`bg-layer-2`、选中名 500 字重主要色 | 171×28 @ 右缘 976（卡片右内边距 24） |
| 卡片高度 | 内容自适应 | 560×283（原 560×334，两个 32 高按钮换成一行 18/28 控件） |

## 4. 验证

```sh
go test ./internal/ui/... -count=1        # 34 个包通过，custompanel 无测试
go test ./internal/shortcuts -count=1     # 通过
go vet ./internal/ui/settings             # 无输出
go build -o whaleisle-pass5.exe ./cmd/whaleisle
```

差异确认方式：新增回归在修复前全部失败（`the close box dismissed the reference instead of closing the row editor`、`Escape dismissed the reference instead of closing the row editor`）；遮罩分支单独停用后 `TestShortcutDialogMaskPressClosesTheRowEditorFirst` 复现同一失败；把标签改回 `ui.Button` 后 `TestShortcutInlineEditorLabelIsNotATabStop` 报 `the edited row's label is a tab stop the reference does not have`。修复后全部通过。

像素侧核对：

- 遮罩点击只退一层时，弹窗重绘与静止态逐像素一致（差异仅来自指针离开行后消失的 tooltip：bbox (744,43)-(875,74)），说明 `*open = true` 的接管没有让 200ms 淡入重播。
- 标签换成 `ui.Box` 前后，`settings-shortcuts-editor-1440.png` 与 `settings-shortcuts-dialog-1440.png` 的 SHA-256 完全相同（纯语义改动，无视觉位移）。
- 安装卡选项行：`docs/review-20261022/evidence/plugins-install-idle-1440.png` 重新生成（44068 字节；原 43069），`plugins-sources-1440.png`、`plugins-market-choice-1440.png`、`plugins-install-failed-1440.png` 重渲染后与既有证据逐字节一致，说明重构只动了空态选项行。

## 5. 残留

1. **实机截图**：仍以生产 `ui.Tester` 渲染与键盘/指针事件为证据；本机 Host 门禁与 DPI 下未取得实机窗口截图。
2. ~~**安装卡的操作区**~~：第 6 轮已收口，安全说明与主按钮移到固定 footer，只有正文滚动（`round6-install-footer.md`）。
3. ~~**引导面板的内层结构**~~：第 6 轮已收口，展开的引导按 `.guide`/`.guideItem` 改成带底与发丝线的圆角面板、示例行（“示例：”+ 等宽包名）与“填入示例”按钮（`round6-install-footer.md`）。
4. **`重新读取`**：参考底栏只有重置与计数；原生在草稿脏或读取失败时仍显示 `重新读取`，保留配置被外部改写的恢复路径（第 4 轮遗留）。
5. **其余 🟡 弹窗**：projects/dirpick/workspacepicker/chat/usage/boot 尚未逐项接共享表面并补 1440×920 证据（第 4 轮遗留）。
6. **分组标题**：参考有应用/输入/菜单/审批四组；原生没有输入与审批命令，只渲染应用组（第 4 轮遗留）。

