# 设置·通用页对齐（第 2 阶段）

日期：2026-10-22  
参考：`docs/review-20261020/evidence/compare/ref-settings-general-1440.png`（鲸屿 0.3.9，1440×920）  
当前：`docs/review-20261022/evidence/settings-general-aligned-1440.png`（生产 `ModalView` 无头渲染，1440×920）  
对照图：`docs/review-20261022/evidence/settings-general-compare-1440.png`

本轮只处理“设置 → 通用”这一组可逐项验证的差异；每次修改都先量参考图，再改代码、补测试、重新渲染。

## 1. 差异清单

| ID | 参考 | 现状（修复前） | 影响 |
| --- | --- | --- | --- |
| D1 | 通用页 10 行在 800 DIP 高的模态内完整显示，行距约 76 DIP | 页面 `Gap(10)` 叠加行自身内边距，每行约 86 DIP；第 10 行“关闭窗口时”被模态底边裁掉 | 内容溢出、最后一项不可见 |
| D2 | 选择器列约 135 DIP，控件按内容取宽并右对齐 | 控件列固定 180 DIP，所有 Select 被 `FillWidth` 拉满；说明列被压缩，关闭说明换行 | 控件过宽、说明多余换行 |
| D3 | 权限行：`权限` / `选择新会话的默认权限模式` | `新任务默认权限` / `只影响以后创建的任务；当前会话权限保持不变。` | 文案漂移，行更高 |
| D4 | 语言行只有 `语言` | `界面语言` + `选择界面使用的语言。` | 多余说明 |
| D5 | `工作步骤展示` / `选择希望看到多少工具调用细节`；`工作步骤收起时机` / `选择何时自动收起工作步骤`；`网页链接默认打开方式` / `对话中网页链接的打开位置` | `工作详情` / `选择任务过程展示的详细程度。`；`工作详情收起时机`（无说明）；`链接打开方式`（无说明） | 文案漂移、说明缺失 |
| D6 | 快捷键是普通 Setting-Cell：标题 + 说明 + 右侧 `编辑快捷键` 按钮，点击打开弹窗 | 折叠行 `键盘快捷键`，编辑区直接展开在页面里 | 结构与交互不同，页面被撑高 |
| D7 | 行内控件保持各自宽度：权限约 135，语言/展示/收起等约 84 | 所有 Select 同宽 180（修复 D2 后仍为 140 全宽） | 与参考的视觉节奏不同 |

参考文案来源（安装包内字典）：

- `resources/vendor/deepseek-harness/packages/client/ui-permission-presets/lib/types/client/locales.js`
- `.../ui-chat/lib/types/client/locale.js`（`settings.transcript.*`、`settings.collapse.*`、`settings.links.*`）
- `.../ui-shortcuts/lib/types/client/locales.js`（`title`、`description`、`view`）

## 2. 修复清单

| 文件 | 修改 |
| --- | --- |
| `internal/ui/settings/settings.go` | 通用页的 `formPane` 间距由 10 改为 5，其它设置页保持 10；避免裁切并让 10 行在 800 DIP 内对齐参考 |
| `internal/ui/settings/body_preferences.go` | 控制列 180 → 140，并 `AlignItems(ui.End)`；Select 去掉 `FillWidth`，改为 `MinWidth(84) / MaxWidth(140)`，按内容取宽 |
| `internal/ui/settings/desktop.go` | 语言行改 `语言`（去说明）；三段文案改为参考字典；`linkOpening` 补说明；语言/关闭/启动 Select 改为 84–140 内容宽 |
| `internal/ui/settings/permission_defaults.go` | 权限行标题/说明与参考字典一致；Select 改内容宽 |
| `internal/ui/settings/shortcuts.go` | 新增 `bodyShortcutPreference`：Setting-Cell + `编辑快捷键`（132 DIP）按钮，点击用共享 `bodyDialog` 打开快捷键编辑；移除页面内折叠 |
| `internal/ui/settings/general_reference_test.go` | 新增中文文案/旧文案回归、快捷键弹窗打开/关闭、生产 `ModalView` 1440×920 渲染 |
| `internal/ui/settings/body_presentation_test.go` | 固定控制列宽度 140 的几何回归；宽/窄布局都保留 |
| `internal/ui/settings/navigation_test.go` | 语言行断言改为参考的 `Language` |

## 3. 行位置对照（1440×920，DIP）

| 行 | 参考 y | 当前 y | Δ |
| --- | ---: | ---: | ---: |
| 权限 | 136 | 134 | −2 |
| 语言 | 220 | 217 | −3 |
| 字号大小 | 281 | 281 | 0 |
| 工作步骤展示 | 358 | 361 | +3 |
| 工作步骤收起时机 | 434 | 437 | +3 |
| 显示代码工作视图 | 510 | 512 | +2 |
| 快捷键 | 586 | 588 | +2 |
| 网页链接默认打开方式 | 662 | 664 | +2 |
| 繁忙时的发送行为 | 739 | 739 | 0 |
| 关闭窗口时 | 816 | 815 | −1 |

测量脚本：`tmp/compare_bands.py`（从两张 1440×920 图的正文列提取文字带起始 y）。

## 4. 验证

```sh
go test ./internal/ui/settings
go test ./internal/ui/...
go build -o whaleisle-pass121.exe ./cmd/whaleisle
```

通过：`internal/ui/...` 全部包；`go build` 成功。  
截图：`settings-general-aligned-1440.png`、`settings-general-compare-1440.png`。

## 5. 残留

1. **实机窗口截图未在本轮取得**：开发版启动器对 Host 有 provenance/visual-readiness 门禁，非隔离实例又会被单实例锁接管；手动窗口在本机 150% DPI 下超出屏幕，UIA 坐标不可靠。故用生产 `ModalView` 的 1440×920 无头渲染作为像素证据，文案/弹窗交互由测试覆盖。下一轮可在 `settingsacceptance` 测试钩子可用的环境中补实机图。
2. **快捷键弹窗内部结构**仍是 MyGo 的编辑器（搜索、逐条录制、保存），不是参考的“快捷键速查/分类列表”弹窗；本轮只对齐入口行、标题、说明、按钮与打开方式。
3. **控件最大宽 140** 对超长枚举值（如长模型名、长权限名）可能截断；需在真实 Host 枚举下复核，必要时改为按内容测量并夹在 140。
4. **背景饱和度/壁纸**：当前 profile 的极光底色明显比参考默认 dune 更艳；这受 Host `ui-theme` 状态影响，不能仅靠改默认值断言一致，留待单独核对渲染公式或 Host 状态。
5. **其它设置页**（外观、界面、模型等）仅受控制列宽度变化的通用影响，未逐页重截图；`go test ./internal/ui/...` 通过，下一轮应逐页对照。
6. 覆盖矩阵中仍待迁移/待核的弹窗：`settings/effect_preferences`、`pluginmanager` 安装/来源/市场选择、`settings/onboarding`、`market/details`。
