# Round133 · 标题带上沿与弹层动效 token

本轮按 [chrome 指标核对](<C:/AI/WhaleIsle-MyGo/docs/parity/chrome-metrics-20261024.md>) 修两处可证实的差异：标题带按钮上沿（G02，4 DIP）和 popover/overlay 的时长与位移（G50）。G01「标题带 48 vs 12」经核对是**测量错误**，不动 48 的带高。

## 1. 事实核对（为何不是 36 DIP）

- `chrome.js:9 TITLEBAR_HEIGHT = 48` 只在 `:376` 导出、全仓无消费者，是死常量；主窗口真正的 48 DIP 带来自 `harness-chrome-inject.js:11,88` 的控件盘与 `AppFrame.module.css:277-284` 的拖拽矩形。
- MyGo 的带也是 48：`internal/ui/frame/workspace_caption.go:31,36,114`，由 `TestSharedCaptionIsOneRowAcrossEveryRoute` 渲染断言；`chrome/titlebar.go:36 TitleBarHeight = 12` 只是平台标题保留值，不是可见带高。
- `window-controls.css` 的 30×30 胶囊在任何实际页面都会被 `boot.css:50-59` 与 `launcher.css:33-42` 覆盖成 32×32、gap 0、padding 12/8/4；主窗口从不加载该文件。两侧真实控件盒一致：32×32、radius 8、right 8、glyph 12。

## 2. 修改

| 项 | 原生 | 修改前 | 修改后 |
|---|---|---|---|
| 控件上沿 | y=12..44（48 盘 + 12 上 / 4 下内边距） | `Top: 8`（y=8..40） | `Top: 12`：`cmd/whaleisle/main.go` 窗口选项 + `internal/ui/frame/caption_buttons.go` 的 Go 三连按钮同步 |
| popover 时长 | 0.16s（`base.css:18`） | 150ms | `internal/ui/widgets/motion.go`、`internal/ui/chrome/motion.go` → 160ms |
| popover 位移 | `translateY(4px)` 进出同距（`base.css:22`、`motion.css:44`） | 进 6 / 出 3 | 进 4 / 出 4（`popoverRise = 4`） |
| overlay 位移 | `translateY(8px) scale(0.96)`（`base.css:21,23`） | 2 | 8；`scale` 无法表达（`ui.Motion` 无 scale 字段） |

`internal/ui/widgets/motion_test.go` 的 150ms 断言同步改为 160ms；`chrome/motion_test.go` 用常量比较，无需改值。

验证：`go test ./internal/ui/widgets ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 1.7s / 3.4s / 12.6s / 18.1s 全部 ok。未跑实机像素（Host 门禁 + 150% DPI；`PrintWindow` 会卡死 WebView2）。

## 3. 残留（已登记，不在本轮盲改）

- overlay 退出仍是 200ms（原生 `overlay-out` 100ms）：`ui.ElementTransition` 只有一个 Duration，`Reveal` 进出共用，需要先把退场时长拆开。
- overlay 的 `scale(0.96)` 需要 `ui.Motion` 增加 scale 字段（工具包能力，不在 app 层伪造）。
- 标题/视图标签行中心仍在 24（原生 28）：`workspace_caption.go` 需 `Padding(12,0,0,0)` 让内容占 y=12..44；会牵动 hero/标签/金色截图，留作单独一轮。
- `browserpreview` 的 `TopInset = chrome.TitleBarHeight(12) + chrome` 与真实 48 带不一致（`media/pdf/html.go:23-26`、`controller.go:128`），未取证是否遮挡，先记录。
- 关闭按钮 hover 仍无 `#e81123` 填充（原生 `harness-chrome-inject.js:120-123`）。
