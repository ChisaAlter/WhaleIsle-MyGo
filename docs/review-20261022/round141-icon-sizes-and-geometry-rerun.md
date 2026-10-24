# Round141 · 托盘/窗口图标尺寸（G36）与 pass273 几何复核

本轮把两个图标光栅对齐原生，并在改为「启动器独立窗口」的 pass273 上复跑几何探针确认尺寸。

## 1. 原生对照

- 托盘：[tray.js:15-21](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/tray.js>) 取 `iconImage()`，**仅 win32** 再 `resize({width:24,height:24})`；其它平台保持原图。
- 窗口图标：[window.js:177-188](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/window.js>) `iconImage()`：win32 优先 `icon.ico`，否则把 `icon.png` 缩到 **48×48**；非 win32 用原图（macOS 窗口本身不显示图标）。

## 2. 改动

[trayicon.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/trayicon.go>)（新增常量与两个纯函数，便于单测）：

| 项 | 之前 | 现在 |
|---|---|---|
| 托盘图标 | 所有平台都 `scalePNG(data, 32)`，得 32×32 | `resizeTrayIcon(GOOS, data)`：**win32 → 24×24**；darwin/linux 直接返回原图（对齐 tray.js 只在 win32 resize 的条件） |
| 主窗口图标 | 未设置（Windows 用 exe 自带图标） | `applyWindowIcon(win)` → `win.SetIcon(48×48 PNG)`，仅 win32 |
| 欢迎窗口图标 | 未设置 | 同上 |
| 启动器窗口图标 | 未设置（pass273 起为独立窗口） | 同上 |

- 缩放仍是 `draw.CatmullRom`（`scalePNG` 未改质量路径），源字节只读：已目标尺寸时原样返回，不动入参。
- 资源读取抽成 `readResourceIcon()`（工作目录或 exe 目录的 `resources/icon.png`），托盘与窗口图标共用；缺失时两者都是「无图标」而不是崩溃。
- 接线只在窗口创建后加一行：`main.go` 主窗口、`welcome_window.go`、`launcher_window.go`；未触碰启动器流程本身。

## 3. 回归

[trayicon_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/trayicon_test.go>) 在原有用例上追加：

- `TestTrayIconResizeTarget`：win32 → 24×24（并断言常量就是原生 24）；darwin/linux → 原字节不动；源字节未被改写；已是 24 的输入原样返回（不重复编码）。
- `TestWindowIconResizeTarget`：win32 → 48×48；darwin/linux → nil（交给系统应用图标）；源字节未改写；缩放后四个象限颜色仍互不相同（Catmull-Rom 没有退化成平均色）。
- `TestScalePNGRejectsBrokenInput`：非 PNG 输入在 `scalePNG`/`resizeTrayIcon`/`windowIconPNG` 都返回错误而不是 panic。

命令（`. .\scripts\env.ps1`）：

- `go test ./cmd/whaleisle -count=1` → ok，21.2s（含图标 6 条）。
- `go vet ./cmd/whaleisle` → 退出 0。

## 4. pass273 几何复核

用 [scripts/window-geometry-probe.ps1](<C:/AI/WhaleIsle-MyGo/scripts/window-geometry-probe.ps1>) 对 `.\whaleisle.exe`（pass273，临时 profile + `welcome-shown`，无工作区）复跑，记录在 [window-geometry-pass273-rerun.json](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/window-geometry-pass273-rerun.json>)：

| 窗口 | class | 物理矩形 | 状态 | DIP（÷1.5） |
|---|---|---|---|---|
| Whale Isle | MyGoWindow | 1590×990 @ (485,189) | 可见（启动器） | **1060×660** |
| Whale Isle | MyGoWindow | 2160×1380 @ (200,0) | 隐藏（主窗口） | **1440×920** |
| （无标题） | MyGoApp | 202×56 @ (0,0) | 隐藏 | — |
| （无标题） | PseudoConsoleWindow | 0×0 | 可见 | — |
| MSCTFIME UI / Default IME | — | 0×0 | 隐藏 | — |

结论：pass273 的**启动器 1590×990 物理 = 1060×660 DIP**、**主窗口 2160×1380 物理 = 1440×920 DIP** 都保持，与原生 launcher 1060×660 与主窗口 1440×920 一致；dpi=144（150%）下两者同时存在，启动器可见、主窗口隐藏，和 pass273 的「启动器独立窗口」设计相符。图标改动不影响窗口尺寸。

**探针健壮性修复（本轮实证发现）**：上一次探针强杀实例后留下 `%TEMP%\mygo-*-1-*.sock`（mygo 单实例 socket，[single_instance.go:157-165](<C:/AI/WhaleIsle-MyGo/third_party/mygo/single_instance.go>)）。Windows 下该残留文件会让后续启动把命令行交给「已死实例」并立即 exit 0，探针于是记录 `status=error, windows=[]`（我先后复现了三次）。探针现在会在**没有 whaleisle 运行**时清掉超过 2 分钟的 `mygo-*.sock` 再启动；清理后复跑即得到上表。这是 mygo 单实例在 Windows 上的一个真实缺陷（强杀后一段时间内无法再启动），不在本 slice 的写入范围，记录待办。

## 5. 边界与未完成

- 原生 win32 优先用 `icon.ico`；MyGo 的 `resources/` 只有 `icon.png`，`Window.SetIcon` 也只接受 PNG，所以两条路径都走 PNG→48。若以后补 `.ico`，需要在 mygo 侧支持 ICO 解码（或继续用 48 PNG，视觉等价）。
- 托盘 24×24 是**物理像素**尺寸；Windows 通知区在 150% 缩放下会自行再缩放/选取合适尺寸，原生 Electron 同样是先缩到 24 再交给系统，因此两边口径一致，但实际像素仍由 shell 决定。
- 探针只覆盖 150% DPI 与一次启动；最小化/最大化/托盘隐藏态未测。
- 未提交、未构建 passN exe。
