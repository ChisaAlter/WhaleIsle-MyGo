# Round207 · 壁纸像素化的屏幕空间语义

结论：像素化改为**先按窗口尺寸 cover 裁切，再按整数倍降采样/最近邻放大**，屏幕上的块大小不再随源图分辨率变化（320×200 与 1440×860 在 factor 3 下实测都是 **3px**）；模糊升级为 3 次盒式卷积逼近 CSS gaussian，并引入原生同款 **48px 出血**再裁回。玻璃面板的采样仍用源分辨率位图，避免材质数值回归。

## 1. 改动前 / 后的屏幕块大小（factor 3，窗口 300×200）

| 源图 | 改前（源图像素块 3px × cover 缩放） | 改后（屏幕空间） |
|---|---|---|
| 320×200 | 3 × (300/320) ≈ **2.8px** | **3px** |
| 1440×860 | 3 × (300/1440) ≈ **0.63px**（几乎看不见） | **3px** |

改后数值由 `TestWallpaperScreenBlockSize` 在**渲染出的图像**上逐像素探测：同一归一化渐变源分别以 320×200 和 1440×860 提供，渲染 300×200，沿中线统计同色游程的众数，两者都必须等于 3。

## 2. 方法

- 新增 [wallpaper_screen.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/wallpaper_screen.go>)：
  - `backgroundScreenBitmap(path, p, w, h)`：目标尺寸 = 窗口像素；先画到 `w+96 × h+96` 的画布（`screenBleed = 48`，对应原生 `WALLPAPER_BLEED`，模糊才有真实邻居、不产生边缘暗环）；`coverFit()` 双线性做 cover 裁切；`pixelateBlocks()` 以 `round(1+p/100*19)` 为块大小（与原生 factor 相同、整数化后最近邻放大）；`blurThreePass()` 三次盒式（每次半宽 `b = round((sqrt(1+4σ²)-1)/2)`，`σ = radius/2`，与 CSS `blur(radius)` 的 σ 对齐）；最后裁掉 48px 出血。
  - 缓存键：`path + targetW + targetH + blur + pixelate`（`backgroundState.targetW/targetH`），源图解码结果按 path 单独缓存（`decoded`），缩放窗口不再重新读盘/解码。
  - `backgroundSamplingBitmap()`：玻璃材质取样的源分辨率位图（沿用改动前的块化/单次模糊），因此 `TestGlassSurfaceReplacesTheSolidBackground` 等材质数值保持不变。
- [background.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background.go>) 的绘制闭包每帧用实际矩形调用 `backgroundScreenBitmap(path, p, r.W, r.H)`，再 `painter.Image(bmp, r, ui.Cover)`（位图已是窗口尺寸，cover 等价 1:1）；`s.pixels` 只由采样函数写入，绘制函数不再覆盖。

## 3. 测试与截图

- `TestWallpaperScreenBlockSize`（[wallpaper_screen_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/wallpaper_screen_test.go>)）：两种源图、factor 3、渲染后像素探测，断言众数块宽相等且为 3。
- `TestWallpaperEffectSemantics`（round203）继续钉住百分比→半径/倍率映射与夹紧。
- 快照重新导出（外观因 3 次模糊与出血而变化）：
  - [round203-wallpaper-dark-blur0-pixel0.png](<evidence/screens/round203-wallpaper-dark-blur0-pixel0.png>)
  - [round203-wallpaper-dark-blur8-pixel12.png](<evidence/screens/round203-wallpaper-dark-blur8-pixel12.png>)
- 门禁：`go test ./internal/ui/chrome ./internal/ui/theme ./cmd/whaleisle -count=1` → chrome ok（全套通过）、theme 2.154s、cmd 21.564s；`go vet` 同三包 → 退出 0。未提交、未构建 passN，未改原生应用、runtime/vendor、third_party。

## 4. 残余

1. **高斯精确度**：3 次盒式卷积与 Chromium 的 CSS gaussian 只在方差上对齐（σ = radius/2），逐像素不完全相同；若要更接近需按半径分配三档盒宽或做真实高斯核，成本更高。
2. **出血**：已实现 48px 出血+裁回，但 MyGo 的位图是 CPU 端 `image.RGBA`，没有 GPU 合成器的采样差异；边缘表现应接近，未做逐像素对照。
3. **面板取样**：玻璃材质仍基于源分辨率位图（保持材质数值稳定），因此面板“透出的纹理”不是屏幕空间块化的那一份；观感差异很小，但严格逐像素不一致。
4. **透明主题**：blur 抬升 20 仍只在渲染层生效、不回写 Host（与 round203 相同，chrome 层无 Host 写权限）。
