# Round235 · 第二层 wash 排查（未完成闭环）

结论：本片把「平铺画布」的门从设置编辑器移到了**配置直读**（`wallpaperCanvasFlatFor`），并修好了因首次运行默认变化而误报的 config 测试；但**实时窗口仍未读到遮罩值**（本轮直采 9 点全为 21,21,23，cyan 0），说明门没有被真实渲染路径命中或窗口底色覆盖了根画布。round233 的 (43,54,50)/(39,84,57)/(37,100,129) 在本次临时 profile 上未复现（该环境无渐变/无壁纸），因此第二层的具体来源仍待用带探针的干净会话确认。

## 1. 已确认的事实

- 临时 profile 的 `config.json` 在运行后**没有**被写入 `goWallpaperBackground`（无显式 gradient），也没有 `goWallpaperPath`；即 schema rewrite 不是本轮样本的洗色来源。
- `theme.Resolve("deepseek.dark")` → `scheme="dark"`、`BG "#151517"`，因此暗色判定本身正确。
- `wp probe`（`WHALEISLE_BACKDROP_PROBE=1`，可写入 `WHALEISLE_BACKDROP_LOG`）在真实窗口里**没有输出**，说明 `wallpaperPainter` 的探针分支未执行——这也解释了为什么前两轮无法用日志定位；需要继续在 mainView 入口记录一次（而不是 painter 内）。
- 直采 9 点（1440×860 后）：全部 21,21,23，青色 0。即真实窗口画的是**未遮罩的 base**，`FlatBackdropFill` 没有作用到该路径。

## 2. 代码改动

- [cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：新增 `wallpaperCanvasFlatFor(path)`，不依赖设置编辑器，直接读 `cfgStore.WallpaperBackground()` 并用 `NormalizeEffect` 判定；探针扩展为输出 normalized/active/canvasFlat/scheme，并支持写日志文件。
- [cmd/whaleisle/main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)：根画布改为 `darkCanvas && state.wallpaperCanvasFlatFor(wallpaperPath)` → `chrome.FlatBackdropFill`，与窗口底色 `maskedCanvasHex` 共用同一判定。
- [internal/config/first_run_test.go](<C:/AI/WhaleIsle-MyGo/internal/config/first_run_test.go>)：首次运行默认改为 none/空色后，原断言「effect 与 colors 不同于首次运行」会误报；改为断言存储的 `backgroundEffect` 仍是显式（`EffectExplicit`）。

## 3. 测试与门禁

- `go test ./internal/config ./internal/ui/chrome ./cmd/whaleisle -count=1` → 3.349s / 3.985s / 21.334s 全 ok；`go vet` 同三包 → 退出 0。
- 既有 `TestFlatBackdropFill`（不透明+数值+light）、`TestWallpaperCanvasFill`（暗色遮罩/亮色不变/显式 gradient 绕过/有图绕过）、`TestPresetDoesNotPaintWithoutEffect` 仍全绿。
- 直采证据：[round235-live-flat.png](<evidence/screens/round235-live-flat.png>)（窗口 1440×860，9 点 21,21,23，cyan 0）。

## 4. 残余 / 下一步（明确）

1. **真实路径探针**：在 `mainView` 顶层记录一次（path/effect/scheme/canvasFlat/`colors.BG`/实际 Background 传入值），确认根画布是否被后续 `DrawOver` 或窗口 `BackgroundColor` 覆盖；目前 painter 内探针不执行，说明该分支在真实渲染中未被调用或提前 return。
2. **who paints 21**：若根画布已遮罩而屏幕仍是 21，则是窗口底色/合成层（DirectComposition 的 `BackgroundColor` 在 `Transparent:true` 下的行为）盖过了内容，需要在窗口层做一次不透明填充（例如在 `WindowFramePainter` 之前铺满 client rect）。
3. round233 的青色（43,54,50/37,100,129）需要在其原始 profile 上重跑探针才能定位是 gradient 还是别的 wash；本轮在干净 profile 下未复现。
