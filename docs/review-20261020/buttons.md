# 按钮样式对齐（Wave 11）

用户反馈：「按键样式还有比较大的差距」。本文记录原版按钮的取值来源、原生实现与替换范围。

## 1. 原版取值（可读源码）

`runtime/vendor/deepseek-harness/packages/client/ui-primitives/src/Button.module.css`：

| 属性 | md | sm |
| --- | --- | --- |
| 高度 | 36 DIP | 28 DIP |
| 字号 / 行高 | 14 / 22 | 12 / 18 |
| 圆角 | `--dsw-radius-md` = 12 | `--dsw-radius-sm` = 8 |
| 水平内边距 | 14 | 10 |
| 图标槽 | 16 × 16，gap 4 | 同左 |
| disabled | opacity 0.4，not-allowed | 同左 |

变体（同一文件）：

- **primary**：`--dsw-alias-button-primary-fill` = `--dsw-alias-brand-primary`（浅色 `rgb(15,17,21)`，深色 `rgb(249,250,251)`），文字 `--dsw-alias-label-primary-foreground`；hover `--dsw-alias-button-primary-hover`（浅 `rgb(67,69,74)`，深 `rgb(235,238,242)`）。**是黑/白单色，不是蓝色。**
- **ghost**：无面；hover `--dsw-alias-interactive-bg-hover`，active `--dsw-alias-interactive-bg-active`。
- **outline**：`border: 0.5px solid var(--dsw-alias-border-l3)`（浅 `rgba(0,0,0,.12)`，深 `rgba(255,255,255,.16)`），hover 同上。
- **toolbar**：`--dsw-alias-button-tool-bar-fill` / `-hover`（`rgba(84,85,87,.5/.6)`）。
- **info**：`--dsw-alias-button-info-fill/hover`（deepseek 蓝，仅强调处使用）。
- **danger**：`--dsw-alias-state-error-primary`（浅 `rgb(236,19,19)`，深 `rgb(242,90,90)`）。

结论：此前 MyGo `Button` 的默认脸是「Surface + 1px 边框 + radius 6 + 13px」，主按钮又是 Accent 蓝，和原版三处都不同：**圆角、字号/高度、主按钮的单色语义**。

## 2. 原生实现

`internal/ui/widgets/button.go`：

```go
widgets.Button(c, colors, label, widgets.ButtonSpec{
    Variant: widgets.ButtonPrimary, // Primary/Ghost/Outline/Toolbar/Info/Danger
    Size:    widgets.ButtonSmall,   // Medium(36/14/12) / Small(28/12/8)
    Disabled: cond,
    Leading: func(c *ui.Context) { /* 可选 13–16 DIP 图标 */ },
})
```

- 基于 `ui.ButtonBase`（不套 MyGo 主题脸），点击仍用 `.Clicked()`。
- hover/pressed 在构建帧读取指针状态；ghost/outline 显式 `BorderWidth(0)`，避免继承主题 1px 框。
- disabled 使用 0.4 透明度；outline 只画 0.5 DIP 边框。
- 主题新增 token：`BrandPrimary`、`BrandPrimaryHover`、`BorderL3`、`ToolbarFill`、`ToolbarHover`、`Danger`、`DangerHover`（`internal/ui/theme/theme.go`）。

测试：

- `TestButtonUsesHarnessMetrics`：md 36 / sm 28。
- `TestPrimaryButtonIsTheBrandMonochromeFill`：主按钮面 = `BrandPrimary`，且不等于 info 蓝；12 DIP 圆角切掉角落。
- `TestDisabledButtonDimsToFortyPercent`。
- `TestGhostButtonHasNoFace` / `TestGhostButtonDrawsOnlyItsLabel`：ghost 不画底、不画框。
- `TestButtonGallery`（`BUTTON_GALLERY_DIR` 门控）：`buttons-light.png` / `buttons-dark.png`。

## 3. 替换范围（约 290 处）

| 区域 | 数量 | 代表 |
| --- | --- | --- |
| files / terminal / diffsurface / usage / remote | 31 / 13 / 42 / 30 / 64 | 工具栏 Toolbar sm、保存 Primary、取消 Ghost、丢弃/删除 Danger、关闭终端确认 Danger md |
| 设置 / 市场 / 插件列表 / 定时任务 | ~27 | 确认框 md 取消 Ghost + 确认 Primary、设置动作条 Primary/Danger md、安装 Primary sm、定时任务保存 md + 新建 Primary sm（含 plus 图标） |
| 审批 / 提问 / 对话 | 19 / 19 / 60 | 允许·下一步·提交 Primary sm、拒绝·讨论修改 Outline sm、跳过·隐藏·加载更早 Ghost sm、停止交互·清空 Danger sm |
| 启动器 | 3 个 helper | 主按钮 / 边框按钮 / 小按钮统一走新配方（影响全部版本页、导入页、组件页动作） |

画廊：`evidence/after/buttons-{light,dark}.png`、`surface-{market,plugins,schedule}.png`、`surface-remote-*.png`、`surface-project.png`、`transcript-approval-card.png`、`transcript-question-card.png`，均已用新按钮重生成。

## 4. 未转换（如实记录）

1. 规则排除：tab、chip、segmented、icon-only 按钮、菜单项、输入框、Switch、网格单元格。它们有自己的原版配方，不属于 `Button`。
2. 签名缺主题参数的 helper：`files/image_view.go`（4 处缩放）、`terminal/host_rename.go` 等 6 处、`diffsurface/git_progress.go`、`remote/file_actions.go` 的条件式确认按钮、chat 的 `SubmittedPlanCardView` / `image.go` 失败卡与灯箱。需要继续把 `theme.Colors` 往上传一层。
3. 「拒绝」用 Outline sm 而非「outline + 红色文字」组合；原版红 hover 需要的变体尚未加入原语。
4. composer 的 chips（权限/模型/预设）与自绘发送按钮保持原样——它们对应原版的 chip 与 send 配方，不是标准按钮。
5. `settings` 仍有约 200 个 `ui.Button` 属于 chip/icon/close/菜单类，按规则没有动；若要「设置页全量换肤」，需先定 chip 与动作按钮的判定规则。
