# Round231 · 不透明画布：透明窗口泄漏桌面的修复

结论：round230 的实时 tint（hero 42,50,48、caption 29,62,78、4/9 cyan、逐次抖动）来自两处叠加：①主窗口 `Transparent: true`，凡未画满的像素都让桌面透上来；②`FlatBackdropFill` 的合成结果 alpha 不是 255（`ui.RGBA(...,0.5)` 的 Over 结果仍是半透明），于是即便画了遮罩，桌面色仍会混进来。本轮把平铺画布强制为**不透明**并同时接到根画布与窗口底色。

## 1. 证据

- 实时截图（首次运行 profile，暗色，无壁纸）采样：hero(900,400)=21,21,23（原始 base、未被遮罩），窗口外桌面点(200,400)=40,40,40、(1400,400)=21,21,23 —— 未画满区域读到的就是桌面/其它窗口的颜色，且与 hero 不同 → 透明泄漏成立。
- 代码：`cmd/whaleisle/main.go:2345` `Transparent: true` + `BackgroundColor: th.BackgroundColor()`；`mainView` 的 `wallpaperCanvasFill(...)` 用了 `systemDark`（OS 主题）而不是解析后的主题 scheme，导致配置 `deepseek.dark`、系统为亮色时遮罩根本不生效（实时 hero 停在 21）。
- `FlatBackdropFill` 之前返回 `ui.RGBA(0,0,0,0.5).Over(base)`：mygo 的 alpha 为 0–255，`0.5` 只是 0.5/255，Over 后的颜色仍带 alpha，透明窗口下继续与桌面混合（这正是 ±2 抖动）。

## 2. 修复

1. [chrome/glass.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/glass.go>)：`FlatBackdropFill` 先做 `rgba(0,0,0,0.5)` 合成，再**强制 alpha=255**，返回完全不透明的遮罩底色；`TestFlatBackdropFill` 增加 alpha==255 断言，仍校验 dark ≈(10,10,11)、light 原样。
2. [cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：新增 `flatDarkCanvasPreview(cfgStore, cfg, th)`（暗色 scheme + 无 `goWallpaperPath` + 非显式 gradient）与 `maskedCanvasHex`（把遮罩后的颜色作为窗口底色 hex）。
3. [cmd/whaleisle/main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)：窗口创建处暗色平铺时 `bgColor = maskedCanvasHex(bgColor)`（不透明窗口底），根画布改用解析后的主题 scheme（`th.Scheme` 而非 OS 的 `systemDark`）决定是否遮罩；圆角/标题按钮/overlay 仍由 `WindowsFrameCornerRadius`/DirectComposition 负责，没有关闭 `Transparent`，壁纸与显式 gradient 分支不受影响，light 不变。

## 3. 验证状态

- 单测：`TestFlatBackdropFill`（不透明度 + 数值 + light）、`TestWallpaperCanvasFill`（dark 平铺遮罩、light 不变、显式 gradient 绕过、有图绕过）、`TestPresetDoesNotPaintWithoutEffect` 全通过。
- 门禁：`go test ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 4.797s / 17.805s / 25.017s 全 ok；`go vet` 同三包 → 退出 0。
- **实时窗口采样未能完成**：本轮两次拉起临时构建（`WHALEISLE_USER_DATA` 临时 profile、1440×860 SetWindowPos）后 `MainWindowHandle` 为空/进程提前退出（二次启动 15s 内无主窗口，疑似单实例锁或端口占用），只拿到首次运行的一张整屏截图（evidence/screens/round231-live-attempt.png，窗口 600×700，hero=21,21,23，桌面点 40,40,40 → 泄漏证据）。最终 9 点验证需要在一次干净启动里重跑（见残余）。

## 4. 残余 / 下一步

1. 用一次干净会话重跑实时验证：启动 → SetWindowPos 1440×860 → 采样 9 点，期望 hero/card ≈12,12,12、cyan 0/9、caption/sidebar 为各自 token 合成。
2. 若仍见 tint，检查 `BackgroundColor` 在 DirectComposition 下的实际用法（可能需要窗口级 `SetBackground` 或整窗 `Draw` 覆盖），以及 `systemDark` 其它使用点是否也需要改为 `th.Scheme`。
3. caption 业务蓝（native 17,31,48）与 glassOpacity 60 面板合成仍未复刻（round228/229 已记录）。
