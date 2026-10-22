# 第 5 阶段：快捷键弹窗的关闭手势

日期：2026-10-22
参考：`reference/electron/vendor/deepseek-harness/packages/client/ui-shortcuts/src/client/Reference.tsx`
（`:96-108` `closeReference`：先收行内编辑器、再关弹窗；`:155-159` 内容区 `onPointerDownCapture` 在非交互区域按下时 `closeEditor()`；`:162` 关闭盒与 `:204-210` 底栏重置都调用同一关闭入口，重置前 `setTarget(null)`）、
`.../packages/client/ui-primitives/src/Modal.tsx`（`:70` 遮罩 `onClick={onClose}`）、`Editor.tsx`（`:163-180` 行内控件只有 `恢复默认`/`移除`/录键器，标签本身不是控件）。
当前：生产渲染 + `ui.Tester` 键盘/指针事件（`internal/ui/settings/shortcut_dialog_affordance_test.go`）。
本轮所有改动都落在关闭手势上：弹窗表面、尺寸、行节奏沿用第 4 轮的几何结论，未再改动。

## 1. 差异清单

| ID | 参考（源码声明） | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D22 | 关闭盒调用 `closeReference`：`target !== null` 时先 `closeEditor()`，弹窗保持打开（`Reference.tsx:102-108`、`:162`） | 关闭盒直接置 `*open = false`，行内编辑器与弹窗一起消失 | 一次点击丢掉两层，与参考的两段式关闭相反；编辑器被强拆时不走 `CancelRecording` 的收敛 |
| D23 | Escape 由 Modal `onClose` 进入同一个 `closeReference`，同样先收编辑器（`Reference.tsx:153`、`Editor.tsx:89-95`） | `DialogBase` 的 backdrop `OverlayShortcut(0, KeyEscape)` 直接关弹窗 | Esc 一次退出两层；编辑中按 Esc 期望是“退出该行” |
| D24 | 遮罩 `onClick={onClose}` 也走 `closeReference`（`Modal.tsx:70`） | 点遮罩直接关弹窗 | 点外一次退出两层 |
| D25 | 底栏“恢复全部默认”在打开确认前 `setTarget(null)`（`Reference.tsx:206`） | `RequestReset` 只清 `Recording`，`target` 仍指向该行 | 确认框打开时其下仍是行内编辑器，Esc 的层级语义不清 |
| D26 | 编辑态的行标签是普通文本，不是控件，也不可聚焦（`Reference.tsx:188`） | 标签是 `ui.Button`，成为编辑态第 4 个 Tab 停靠点与一个可宣读控件 | 键盘用户在行内多停一站，与参考的 3 个控件（恢复默认/移除/录键器）不同 |

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/settings/shortcuts.go` | 新增 `shortcutCloseReference(open, e, target)`：有行内编辑器时只 `CancelRecording` 并清空 `target`，否则关闭弹窗；关闭盒改为调用它。弹窗内容在 `*target != ""` 时把 `Escape` 注册到 panel（非 overlay，位于焦点链上，优先于 backdrop 的关窗处理），并接管遮罩点击（`backdrop.Clicked()` 时收编辑器并恢复 `*open = true`，使第一下只退一层）。底栏“恢复全部默认”在 `RequestReset` 前清 `target`；清空后弹窗不再注册 panel 级 Escape，确认框独占 Esc。行内编辑器的标签由 `ui.Button` 改为不可聚焦的 `ui.Box` + `Clicked()`（保留“点标签收起编辑器”的指针路径，去掉多出的 Tab 停靠点）。 |
| `internal/ui/settings/shortcut_dialog_affordance_test.go` | 新增 5 条回归：关闭盒、Escape、遮罩、录键中 Escape、重置确认、标签非 Tab 停靠点。 |

## 3. 手势对照

| 手势（行内编辑器已打开） | 参考 | 当前 |
| --- | --- | --- |
| 关闭盒 | 收起编辑器，弹窗保持；再点一次关弹窗 | 同左（`TestShortcutDialogCloseBoxClosesTheRowEditorFirst`） |
| Escape | 收起编辑器，弹窗保持；再按一次关弹窗 | 同左（`TestShortcutDialogEscapeClosesTheRowEditorFirst`、`...ArmedRecorderEscape...`） |
| 遮罩点击 | 收起编辑器，弹窗保持；再点一次关弹窗 | 同左（`TestShortcutDialogMaskPressClosesTheRowEditorFirst`） |
| 底栏“恢复全部默认” | 先清行内编辑器，再开确认框 | 同左（`TestShortcutDialogResetConfirmationKeepsEscape`） |
| 点行标签 | 编辑器收起（指针落在非交互区域） | 同左，但标签已不再可聚焦（`TestShortcutInlineEditorLabelIsNotATabStop`） |

Esc 在录键器持有焦点时同样只退一层：MyGo 先把已注册为 Shortcut 的按键送给 panel，再轮到聚焦元素的 `HandleInput`，所以录键器不会把 Escape 当成非法组合键并弹出“审阅/重试”两条恢复动作。

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

## 5. 残留

1. **实机截图**：仍以生产 `ui.Tester` 渲染与键盘/指针事件为证据；本机 Host 门禁与 DPI 下未取得实机窗口截图。
2. **安装卡的操作区**：参考把安全说明与主按钮放在固定 footer；原生仍放在滚动体内（第 4 轮遗留）。
3. **`重新读取`**：参考底栏只有重置与计数；原生在草稿脏或读取失败时仍显示 `重新读取`，保留配置被外部改写的恢复路径（第 4 轮遗留）。
4. **其余 🟡 弹窗**：projects/dirpick/workspacepicker/chat/usage/boot 尚未逐项接共享表面并补 1440×920 证据（第 4 轮遗留）。
5. **分组标题**：参考有应用/输入/菜单/审批四组；原生没有输入与审批命令，只渲染应用组（第 4 轮遗留）。
