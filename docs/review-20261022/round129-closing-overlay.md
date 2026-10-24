# Round129 · 关闭遮罩（G40）

本轮补上原生退出时覆盖主窗口的「关闭中」层。矩阵 `docs/parity/gap-matrix-20261024.md` 把 G40 记为缺失：MyGo 的 `finalizeQuit` 直接 `sup.Stop()`，窗口停在最后一帧，用户看不到「正在停止本机 Harness 服务」的反馈；原生在停服前先显示这层遮罩。

## 1. 原生对照

| 原生 | 位置 | MyGo |
|---|---|---|
| 整窗遮罩：`position: fixed; inset: 0`、主题底、居中 | `reference/electron/src/main/closing-overlay.js:4-20` | `internal/ui/chrome/closing.go` `DrawClosingOverlay`，`ui.Overlay` + `Absolute` 全窗、`Modal` 捕输入 |
| 卡片：48 内边距？否——`min-width:240; padding:24; border:1px; radius:24; gap:8`，双层阴影 `0 0 1px rgba(0,0,0,.2)` + `0 12px 32px rgba(0,0,0,.08)` | 同文件 `:21-32` | 同文件常量与 `Shadow`/`Stroke` |
| 旋转环：36×36、2px 边框、`border-top-color` 主题蓝、`0.85s linear infinite` | `:33-40` | 36×36 圆环、2px、顶 90° 弧为主题 accent；**旋转未做**（见 §4） |
| 文案：标题 16/24 500，`margin:8px 0 0`；说明 12/18 muted | `:41-52` | `FixedLineHeight(24/18)` + `Margin(8,0,0,0)`，避免把 24 当成字体倍数 |
| 文案值：`关闭中` / `正在停止本机 Harness 服务，请稍候`（en：`Closing` / `Stopping the local Harness service…`） | `:59-70` | `i18n.KeyClosingTitle` / `KeyClosingDetail`，中英逐字登记 |
| 顺序：先 restore/show/focus、再插遮罩、最后 `harness.shutdown()`；绘制有 500ms 上限且失败不阻塞 | `reference/electron/src/main/index.js:1428-1437`、`closing-overlay.js:128-158` | `finalizeQuit` → `stopWithClosingOverlay(win, sup)`：置标志、投递一次 UI 线程任务（restore/show/focus + 重绘）、调用方最多等 500ms 那一帧，然后**始终** `sup.Stop()` |

## 2. 实现

- 状态：`mainState.closingOverlay`；`mainView` 第一条语句是 `defer state.paintClosingOverlayLayer(c, closingColors)`，因此无论从聊天、boot、launcher 还是设置弹层返回，遮罩都最后画、盖住整窗。
- 配色：`closingColors` 取 `mainView` 已解析的 `colors`（已含用户固定 scheme 与系统深浅色），不用 `mygo.Theme.IsDark()`——后者在 `ui.Tester` 无头环境会阻塞，且固定浅色主题时会被系统深色带偏。
- 退出：`finalizeQuit` 用 `stopWithClosingOverlay` 取代裸 `sup.Stop()`。标志、restore/show/focus 与重绘在同一个投递任务里完成；等待发生在调用方 goroutine，投递本身卡住也不会把关闭拖死；nil 窗口、隐藏窗口、绘制失败、超时都继续停服。
- 不新增顶层窗口：原生是往页面注入 DOM，MyGo 用同一主窗口的最上层 `Overlay`，语义与层级一致。

## 3. 回归与证据

- `go test ./cmd/whaleisle ./internal/ui/chrome ./internal/ui/i18n ./internal/ui/frame -count=1` → cmd 20.9s、chrome 3.1s、i18n 0.3s、frame 14.8s，全部 ok；同范围 `go vet` 退出 0。
- cmd 4 项：`TestClosingOverlayIsUpBeforeHarnessStop`（supervisor 到 `Stopped` 时标志已置）、`TestClosingOverlayPaintDeadlineDoesNotHoldShutdown`（40ms 测试预算内返回 false 且仍停服）、`TestClosingOverlayFrameRequestHandlesAMissingWindow`、`TestPaintClosingOverlayLayerFollowsTheTheme`。
- chrome 6 项：中英文案逐字、720×480 下 zh/en 几何（卡片 `max(240, 最宽行+48)`、高 150、spinner 36 位于卡片 +24 且水平居中、标题顶 = spinner 36 + 8 + 8、说明顶 = 标题 24 + 8、行高 24/18、整卡与文案居中、24 圆角露出底色）、短文案的 240 下限、整窗覆盖（页面底色不再可见）、明暗主题与 spinner 顶部弧 accent 像素、空文案不画。
- 渲染证据：[浅色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/round129-closing-overlay-light.png>)、[深色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/round129-closing-overlay-dark.png>)（`ui.Tester` 输出，13,184 / 11,747 字节）。

## 4. 边界与未完成

- **旋转未做**：原生 `0.85s linear infinite`；MyGo 只画静态环。遮罩只活到 500ms 绘制预算 + 停服，进程随即退出，第二帧在多数退出路径上到不了屏幕。若后续引入退出期动效时钟，再补逐帧旋转；矩阵 G40 的动画差异按此登记。
- 无实机窗口截图：`PrintWindow(PW_RENDERFULLCONTENT)` 会让 WebView2 窗口卡死（本轮实测 `Responding=False`，只能杀进程），实机像素证据仍受 Host 门禁与 150% DPI 限制。
- 本轮不校验真实 harness 停止时长；测试用假 supervisor 覆盖「先遮罩后停服」的顺序与超时边界。
- 矩阵 `gap-matrix-20261024.md` 是 4c08edfc 时点的快照，G40 行未改；以本文与 `docs/status.md` 为准。
