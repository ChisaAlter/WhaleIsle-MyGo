# Round239 · 真实路由的 wash 门控（验证仍未闭环）

结论：本轮在真实绘制入口（`wallpaperPainter`）加了「只有本地显式 `backgroundEffect` 才允许 gradient」的门——Host 外观文档带来的渐变种子不再能作画；探针日志确认 `path="" effect="none" localExplicit=false active=false`，即渐变 painter 已经关闭。但**直接屏幕采样仍读到 21,21,23**：临时会话的 `MainWindowHandle` 是 600×700 的 launcher 窗口（不是绘制 mainView 的那个 surface），强行 SetWindowPos 后截到的仍是 launcher 的底；mygo 的 `Window.CapturePage()` 在本构建返回空，无法从内部取图。

## 1. 图层追踪

| 层 | 位置 | 探针结果 |
|---|---|---|
| 根画布 | main.go（`canvasFill` + Draw 填充） | `view: scheme="dark" systemDark=true darkCanvas=true path="" flat=true bg="#151517" fill={10 10 11 255}` |
| 壁纸/环境 painter | `wallpaperPainter` → `chrome.WallpaperBackgroundPainter` | 门控后 `painter: path="" effect="none" localExplicit=false active=false`（不绘制） |
| 环境渐变 | `chrome/background.go` 渐变分支 | 仅 `Effect=="gradient"` 时进入；已被门控 |
| 帧/边框 | `chrome.WindowFramePainter` | 只描 1px ring，不填充 |
| boot/sky 渐变 | `internal/ui/boot/backdrop.go` | 仅启动页使用，主窗口路由不经过 |
| 其它渐变色带 | `chrome/glass.go:605` | 玻璃高光带，需要已有表面才画 |

由此，round237 的绿-青-紫空间渐变最可能来自 **Host 外观文档自带的 gradient 种子**（`theme_font_preferences.go` 接受 Host 外观后 `g.Prefs = h.Background`），本轮已在 painter 入口按「本地是否显式保存过 backgroundEffect」把它关掉；该门对 wallpaper/gradient 显式选择与 light 主题不生效。

## 2. 改动

[cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：`wallpaperPainter` 解析后，若 `path == ""` 且 `cfgStore.WallpaperBackground()` 的 `EffectExplicit` 为 false，则把 prefs 的 Effect 强制为 `none` 并记录 `painter:` 探针（写 `WHALEISLE_BACKDROP_LOG`）。这样「Host 种子渐变」不会再画，而用户在 MyGo 里显式选过渐变（本地配置有 `backgroundEffect` 键）时仍照旧绘制；有壁纸时不受影响。

## 3. 验证状态

- 探针（`WHALEISLE_BACKDROP_LOG`）：
  ```
  view: scheme="dark" systemDark=true darkCanvas=true path="" flat=true bg="#151517" fill={10 10 11 255}
  painter: path="" effect="none" localExplicit=false active=false
  ```
- 直接截屏（`WHALEISLE_FORCE_MAIN=1`，1440×860 拉伸）：9 点仍 **21,21,23**、cyan 0（`evidence/screens/round239-live-flat.png`）。枚举窗口确认临时会话同时存在 launcher 窗口（600×700）与主窗口，`MainWindowHandle` 指向 launcher，因此截到的是 launcher 的底色而不是 mainView 的遮罩画布；本轮未能拿到 mainView 的真实像素。
- mygo `Window.CapturePage()` 在该构建返回空（未写出文件），内部取图不可用。
- 门禁：`go test ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 3.862s / 14.133s / 21.787s 全 ok；`go vet` 同三包 → 退出 0。

## 4. 残余 / 下一步

1. 用正常 harness 启动（非 launcher、非 FORCE_MAIN）并在 pass298 同环境复采；若仍有绿-青-紫，则在 `wallpaperPainter` 的探针日志里核对 `effect` 是否真的为 `none`（若为 none 仍渐变色，则渐变来自另一个绘制者，需要抓 `p.FillGradient` 调用栈）。
2. 修正验证工具：按标题/尺寸挑主窗口（排除 launcher），或实现可用的内部截图（mygo CapturePage 在当前构建为空，需查 WebView2 是否已初始化）。
3. caption 业务蓝与 glassOpacity 60 仍待处理（round228/229）。
