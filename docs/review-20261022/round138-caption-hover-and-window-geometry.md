# Round138 · 标题栏按钮 hover 填充与真实窗口几何取证

本轮两件事：给 Go 绘制的窗口三键补上原生 hover 填充（关闭 #e81123 + 白色字形，最小化/最大化用交互 hover 底色），以及加一个可重复的**真实窗口**几何探针并把 pass272 的实测矩形落盘。不涉及启动器与 chat。

## 1. 原生对照

- [window-controls.css:39-46](<C:/AI/WhaleIsle-MyGo/reference/electron/src/renderer/window-controls.css>)：`button:hover { background: var(--dsw-alias-interactive-bg-hover) }`；`button[data-act="close"]:hover { background: #e81123; color: #fff }`。常态是 `background: transparent; color: var(--dsw-alias-label-primary)`（`:15-30`）。
- [harness-chrome-inject.js:116-122](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/harness-chrome-inject.js>) 注入到 Harness 标题栏的同一份规则：hover 用交互底色，关闭 hover 用 #e81123 + 白字。
- 原生字形是 12×12 的 SVG（✕、—、▢）；MyGo 是 Go 绘制，位置/命中区保持 Top 12、32×32、右侧 8/40/72。

## 2. 实现

[caption_buttons.go](<C:/AI/WhaleIsle-MyGo/internal/ui/frame/caption_buttons.go>)：

- 三键统一走 `control(...)`，`Draw` 里先 `button.Hovered()` 判定：hover 时 `p.Fill(r, fill, 8)`（与按钮 8 DIP 圆角一致）再画字形；常态不填充，字形仍用 ink，外观与之前一致。
- 关闭：hover 填充 `#e81123`（`captionCloseHover`），字形转白；最小化/最大化：hover 填充 `c.Theme().SurfaceHover`（appearance 帧把 `theme.Colors.Hover` 写进 `SurfaceHover`，即 `--dsw-alias-interactive-bg-hover`），字形保持 ink。
- 顺带修掉一个既有缺陷：`ui.Painter.Line` 只画**轴对齐**线段（paint.go:874 取包围盒），旧的「✕」其实是一条 8×1 横杠。现在用 `captionDiagonal` 沿对角线打 1 DIP 点画真正的 ✕（同样是 ink/白色字形）。
- 为了能在 headless 下测试，把绘制体拆成 `drawCaptionButtons(c, ink, captionWindowActions{Close, Minimize, ToggleMaximize, Maximized})`；`captionWindowButtons` 仍负责平台/原生 overlay 判定并取 `mygo.FocusedWindow()`，签名不变。命中区常量化为 `captionButtonSize=32`、`captionButtonTop=12`。原生 overlay（`WindowsCaptionButtons` 窗口选项）未动。

## 3. 回归（ui.NewTester 指针采样）

[caption_buttons_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/frame/caption_buttons_test.go>)：

- 先用 `tt.Find("关闭"/"最大化"/"最小化")` 取到按钮盒，断言与原生几何完全一致（close/max/min = 360/328/296 起，y=12，32×32）。
- 常态帧：三个按钮的填充采样点都是页面色（透明），且各自有 ink 字形。
- hover 关闭：填充点精确等于 `#e81123`（±1），字形出现纯白像素；同时最小化/最大化的填充点仍是页面色（hover 不外溢）。
- hover 最小化/最大化：填充点等于 `SurfaceHover` 叠在页面色上的合成值（测试里按 source-over 现算期望），字形仍是 ink；另一个按钮不被染色。
- 指针移开：三个按钮恢复透明。
- hover 不吞点击：三键各点一次，回放计数 1/1/1。
- 另测最大化态仍画「还原」双框字形（墨迹像素多于单框），hover 改动未破坏。

命令（`. .\scripts\env.ps1`）：

- `go test ./internal/ui/frame -count=1` → ok，13.2s；定点 `-run CaptionButtons` 两条 PASS，2.5s。
- `go vet ./internal/ui/frame` → 退出 0。

## 4. 真实窗口几何探针

新增 [scripts/window-geometry-probe.ps1](<C:/AI/WhaleIsle-MyGo/scripts/window-geometry-probe.ps1>)：用临时 profile（写入 `welcome-shown` 标记、无工作区，直接进启动器路径）启动给定 exe，轮询到首个带标题的可见顶层窗口后静置 2.5s，用 **EnumWindows + GetWindowThreadProcessId + GetWindowText + GetWindowRect + IsWindowVisible + IsIconic** 枚举该 PID 的顶层窗口，按 per-monitor-v2 物理像素（`SetThreadDpiAwarenessContext(-4)`）记录 title/class/rect/visible/minimized/dpi，写 JSON，然后在 finally 里关闭并强杀进程、删除临时 profile。**全程不调用 PrintWindow/BitBlt**（该类窗口会挂起，已验证）。

参数：`-Exe`、`-Out`（必填），`-TimeoutSeconds`（默认 60）、`-SettleMilliseconds`（默认 2500）、`-UserData`（可选，复用指定 profile）。

pass272 实测（`.\whaleisle.exe`，临时 profile，PID 32864）：

| 窗口 | class | 物理矩形 | 状态 | dpi |
|---|---|---|---|---|
| Whale Isle | MyGoWindow | 2160×1380 @ (200,0) | 可见 | 144 |
| （无标题） | MyGoApp | 202×56 @ (0,0) | 隐藏 | 144 |
| （无标题） | PseudoConsoleWindow | 0×0 | 可见 | 144 |
| MSCTFIME UI | MSCTFIME UI | 0×0 | 隐藏 | 144 |
| Default IME | IME | 0×0 | 隐藏 | 144 |

结论：主窗口 2160×1380 物理像素 ÷ 1.5（144 DPI）= **1440×920 DIP**，与 `mainWindowSize` 一致（G02 的窗口尺寸在真实窗口上得到确认）；这一次启动落在主窗口上（启动器是主窗口内面板，不是独立窗口），没有额外顶层窗口。原始记录见 [window-geometry-pass272.json](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/window-geometry-pass272.json>)。

## 5. 边界与未完成

- 探针记录的是**物理像素**矩形；DIP 需要按每窗口的 dpi 折算（上表已折算）。窗口矩形包含不可见边框/阴影（GetWindowRect 语义），与 DIP 内容区不是同一口径。
- 探针不截图、不读像素、不注入输入；它验证的是窗口存在与几何，不是视觉对比。
- 本次只测了 1 个 DPI（150%）与一次启动；多 DPI、最小化/最大化、托盘隐藏等状态需要再跑（脚本已支持 `-SettleMilliseconds`/`-TimeoutSeconds` 与复用 profile）。
- hover 填充是 Go 绘制层，原生 caption overlay 未改；`ui.Painter` 没有旋转/路径 API，✕ 用 1 DIP 点列近似，放大到高 DPI 时边缘会比原生 SVG 略硬（已记录）。
- 未提交、未构建 passN exe。
