# Round236 · 窗口层填充与 mainView 探针（仍未落到屏幕）

结论：探针证明 `mainView` 计算并传入了遮罩色 `{10 10 11 255}`（`scheme="dark" systemDark=true darkCanvas=true path="" flat=true bg="#151517"`），但**直采屏幕仍是 21,21,23**。因此遮罩值在渲染管线里被覆盖或根本没有进入被截取的那个窗口/表面；本轮把填充上移到 Draw 层（`p.Fill(r, canvasFill)`，不只是 `Background`），仍未改变截取值。

## 1. 探针日志（WHALEISLE_BACKDROP_LOG）

```
view: scheme="dark" systemDark=true darkCanvas=true path="" flat=true bg="#151517" fill={10 10 11 255 [0 0 0]}
```

- `wallpaperCanvasFlatFor` 为 true，`chrome.FlatBackdropFill` 返回不透明 (10,10,11)。
- 首次窗口的 `flatDarkCanvasPreview` 同样为 true，`bgColor` 被设为 `#0a0a0b`（窗口 BackgroundColor）。
- 之前 painter 内的探针不触发：因为本轮临时 profile 启动到的是 **launcher 窗口**（600×700），`launcherFirst` 为真时 mainView 不走 painter；新增 `WHALEISLE_FORCE_MAIN=1` 诊断开关后 mainView 才出现并输出上述日志。

## 2. 改动

- [cmd/whaleisle/main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)：
  - mainView 入口记录 path/scheme/systemDark/darkCanvas/flat/bg/fill（写 `WHALEISLE_BACKDROP_LOG`）。
  - 根 Column 增加 **Draw 层不透明填充**：`darkCanvas && flatCanvas` 时 `p.Fill(r, canvasFill, 0)`（原实现只有 `Background(canvasFill)`），确保在布局/边框之外也铺满 client。
  - 新增 `WHALEISLE_FORCE_MAIN=1` 诊断开关（跳过 launcher-first），用于无 harness 的干净会话直采。
- [cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：`appendBackdropLog` 提取为共享辅助；`wallpaperCanvasFlatFor` 直读 cfgStore（round235）。

## 3. 结果

- 直采 1440×860：9 点仍全部 **21,21,23**，cyan 0（[round236-live-flat.png](<evidence/screens/round236-live-flat.png>)）。
- 由于探针已证明传入值正确，剩余可能在：①截取的是另一个表面/窗口（MainWindowHandle 指向的窗口不是实际绘制 surface）；②`Transparent:true` 下 DirectComposition 用窗口背景（或 mygo 主题底色）覆盖了 content 的 Draw；③mainView 的画面被某个全屏占位层（未连接 harness 的 onboarding/空态）用 base 21 覆盖。
- 门禁：`go test ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 3.548s / 14.528s / 22.050s 全 ok；`go vet` 同三包 → 退出 0。

## 4. 下一步（按证据顺序）

1. 在 `p.Fill` 与窗口 BackgroundColor 之外，用 `EnumWindows` 逐个窗口截取（而不只 MainWindowHandle），确认可视窗口的 HWND 与绘制 surface 一致。
2. 若 Draw 填充生效但屏幕仍是 21，则问题在 mygo 的 Transparent 合成：需要在窗口创建时使用非 Transparent + 自绘圆角，或使用 mygo 提供的 `SetBackground`/surface 接口（需查 mygo 文档，不改第三方源码）。
3. 用连接了 harness 的正常启动（非 FORCE_MAIN、非 launcher）复采，排除空态占位层。
