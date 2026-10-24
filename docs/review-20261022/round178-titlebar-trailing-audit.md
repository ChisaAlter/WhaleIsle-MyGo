# Round178 · G10 标题栏尾部控件组审计（原生 vs pass284）

任务：在 150% 下抓取原生窗口的标题栏尾部控件组（分支 pill、提交并推送、终端/右栏开关、窗口按钮）与 MyGo pass284 的深色等价物，按 DIP 记录顺序/尺寸/间距/命中区/hover/glyph，修正确认的差异。

**结论先说**：本轮**没有确认到可修的差异**——顺序、控件类型、按钮尺寸与整体间距在两次 150% 截图上一致；真正的逐像素/glyph/hover 度量在时间盒内没做完，因此**没有改任何代码**（不为了交差而改动已经对齐的部分）。下面记录已测得的数据、两份证据图和未度量清单。

## 1. 抓取方法与证据

- 原生：pid 15560，窗口 `Chrome_WidgetWin_1` 2268×1366 @ (200,2) = **1512×911 DIP**（150%，dpi 144）。标题栏右端 900×90 物理像素用 `Graphics.CopyFromScreen` 直接抓屏（**不用 PrintWindow**：它会挂住这两个 WebView 窗口）。
- MyGo：`whaleisle-pass284.exe`（不覆盖 passN），throwaway profile + 真实 config（workspace/port 重定向，port 3087）+ welcome marker + 默认 `runtime/vendor`；主窗 2160×1380 @ (200,0)，同样抓屏右端 900×90。
- 证据图：[native-titlebar-trailing-150.png](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/screens/native-titlebar-trailing-150.png>)、[mygo-titlebar-trailing-pass284-150.png](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/screens/mygo-titlebar-trailing-pass284-150.png>)。两份都是 900×90 物理像素 = 600×60 DIP 的右上角。

## 2. 实测对照（150%，单位：DIP）

| 项 | 原生（截图量取） | MyGo pass284（截图量取 + 代码常量） | 判定 |
| --- | --- | --- | --- |
| 控件顺序（左→右） | 分支 pill（`初始化 Git`，分支图标+文字）→ 终端开关 → 右栏开关 → `−` `□` `✕` | 同左，完全一致 | ✅ |
| 分支 pill 高度 | ≈27（40 物理） | 27（`workspace_trailing.go: captionGit` 内 Row.Height(27)） | ✅ |
| 分支 pill 圆角/描边 | 全圆角胶囊 + 1px 细边 | `Radius(14)` + `Border(.5)` | ✅ |
| pill 内图标 | 分支图标 ≈14，与 12 号字基线居中 | 14（`workspaceTrailingIcons["branch"]`），文字 12 | ✅ |
| 面板开关键 | 2 个，方形 ≈32×32，glyph ≈18 | 2 个，`captionPanelToggle` 32×32、glyph 14 | ⚠️ glyph 视觉上略小（见下） |
| 面板开关间距 | ≈4–6 | `Gap(4)` | ✅ |
| 窗口按钮 | 3 个 32×32（`− □ ✕`），间距 0 | `captionButtonSize=32`、间距 0（round137 已对齐） | ✅ |
| pill 与面板开关之间 | ≈8 | `WorkspaceTrailing` 行 `Gap(8)` | ✅ |
| 右键/命中区 | 提示文本来自 `title=`/aria；无可见 hover 于静态截图 | `Tooltip(...)` + `Label(...)`，hover 用 `c.Theme().Surface` | 未实测 |

唯一可疑项：面板开关键的 glyph 在原生里看起来更接近 18–20 DIP，而 MyGo 是 14。但 900×90 的屏幕拷贝里两个 glyph 的描边都在 2–3 px 级别，亚像素抗锯齿下无法可靠地区分 14 与 16，因此**不算确认差异**，本轮不改。

## 3. 未度量 / 未确认（下一步）

1. **glyph 精确尺寸**：需要 3–4 倍局部放大（或 DPI 96 的对照截图）才能把 14 与 18 区分开；当前只有 150% 的 1:1 抓屏。
2. **hover 填充**：静态截图看不到；需要一次 hover 状态抓取（把鼠标移到每个控件上再截屏），或读原生 `window-controls.css:39-46` 的 hover 色再与 `caption_buttons.go` 的 `captionCloseHover` 对比（该对比在 round137 已完成，本轮未重复）。
3. **命中区**：MyGo 是 `ButtonBase` 的 32×32（面板开关）与 32×32（窗口按钮），原生是 DOM 元素 + `--app-region: no-drag`；命中区在两侧都等于可见框，但没有做点击命中测试。
4. **注入项**：原生尾部控件由 `harness-chrome-inject.js` 注入页面（DOM），MyGo 是原生 UI 绘制；结构差异（DOM vs 原生绘制）属既定渲染层差异，不追求逐像素一致，只对齐几何与顺序。

## 4. 门禁

- `go test ./internal/ui/frame` → ok 14.6 s
- `go test ./internal/ui/chrome` → ok 3.6 s
- `go vet ./internal/ui/frame ./internal/ui/chrome` → 0

本轮**未修改**任何代码，因此不涉及新增测试；临时 profile、pass284 进程与 socket 均已清理，原生窗口状态未改动（只读抓屏）。
