# Round203 · 壁纸效果字段（blur / pixelate / transparentTheme）语义与对齐

结论：三个字段的**取值语义**（0–100 百分比、非有限数归零、取整、夹紧）与**映射**（blur→`p/100*40`px、pixelate→`1+p/100*19` 倍、transparent 抬升到 20）本轮已按安装版实现并可测；`transparentTheme` 的遮罩/玻璃透明度此前已生效。还剩两处结构性差异：MyGo 在**源图像素**上做像素化（原生在**屏幕空间**按倍率重采样），以及透明主题的模糊抬升只在渲染层生效、不回写 Host。

## 1. 原生语义（安装版与参考源码）

| 项 | 原生值 | 出处 |
|---|---|---|
| 夹紧 | 非有限数（NaN/Inf）→ 0；否则 `Math.round` 后夹到 0–100 | `ui-theme/lib/client.js:1128-1136 clampWallpaperEffect()` |
| 模糊 | `blurPx = clamp(p) / 100 * 40`（0–40px），写到根元素 `--dsh-wallpaper-blur` | 同文件 `1137-1144 wallpaperBlurPx()`、`1578` |
| 像素化 | `factor = p<=0 ? 1 : 1 + p/100*19`（1–20）；位图按 `cssSize / factor` 绘制，CSS 再拉伸回并用 `image-rendering: pixelated` | `1145-1153 wallpaperPixelFactor()`；`wallpaper.css:11-29` |
| 默认值 | `wallpaperBlur: 0`、`wallpaperPixelate: 0`、`transparentTheme: false`、`backgroundEffect: "gradient"` | `client.js:4038-4052 DEFAULT_THEME_SETTINGS` |
| 图层 | `#dsh-wallpaper-inner`：四周 `-48px` 出血（96px），`background-size: cover`，`filter: blur(var(--dsh-wallpaper-blur, 0px))`，`image-rendering: pixelated` | `ui-theme/src/styles/wallpaper.css:11-29` |
| 遮罩 | `#dsh-wallpaper::after` 叠 `--dsw-alias-bg-mask-1`；`html[data-dsh-transparent]` 时改为 `transparent`（不压暗，颜色直接透出） | `wallpaper.css:40-55` |
| 透明主题 | 生效且有壁纸时，把低于 20 的 blur **一次性**抬到 20 并**持久化**（`queueWrite`），之后用户可再调低 | `client.js:8352-8366 nudgeTransparentBlur()` |
| 窗口层 | 注入的 `#dsh-wallpaper` 只负责圆角/裁剪，外层还有 1px hairline ring | `harness-chrome-inject.js:156-190` |

## 2. MyGo 实现

- [background.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background.go>) 新增三个纯函数，逐条对应原生：
  - `clampWallpaperEffect(value)`：NaN/Inf→0，`math.Round`，夹 0–100；
  - `wallpaperBlurRadius(percent)`：`round(clamp/100*40)`，供 `blurBitmap` 的半径；
  - `wallpaperPixelFactor(percent)`：`clamp<=0 ? 1 : 1+clamp/100*19`。
- `ConfigureBackdrop` 先夹紧 `Blur/Pixelate` 再参与透明抬升判断（原来直接比较原始值，越界/非有限数会漏判）；`backgroundBitmap` 的块大小改为 `round(wallpaperPixelFactor(...))`（原来 `1+int(p/100*19)` 截断且未夹紧），模糊半径改为 `wallpaperBlurRadius(...)`（原来 `int(p*.4)` 截断、未夹紧）。
- `transparentTheme`：`ConfigureBackdrop` 在生效且 `Blur<20` 时把**绘制用**的 blur 抬到 20，关掉后恢复；`WallpaperBackgroundPainter` 在 `p.Transparent` 时跳过 `bg-mask-1` 压暗（对应 `html[data-dsh-transparent] #dsh-wallpaper::after{background:transparent}`）；面板透明度继续走既有 `effectivePrefs`/`MaterialAlpha`（0% 玻璃）。
- 字段来源：`wallpapers.Preferences` 的 JSON 标签就是 Host 键名（`wallpaperBlur`、`wallpaperPixelate`、`transparentTheme`），由 Host 设置 JSON 读入；本树没有 `theme.ReadHostSettings` 这个入口，故未新增读取路径（避免猜默认值：默认取 `wallpapers.Defaults()`，与上面的原生默认一致）。

## 3. 测试与截图

- [background_effects_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background_effects_test.go>) `TestWallpaperEffectSemantics`：11 组映射（0/8/12/50/100/150/-5/NaN/+Inf/0.4/99.6）验证 blur 半径与 1–20 倍率，并钉住块大小取整（0→1、8→3、12→3、50→11、100→20、150→20）。
- [wallpaper_effect_snapshot_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/wallpaper_effect_snapshot_test.go>) `TestWallpaperEffectSnapshot`：暗色种子壁纸 1440×860，导出 0/0 与 8/12 两张（`WALLPAPER_EFFECT_SNAPSHOT_DIR` 指定目录，默认跳过）。
- 截图（本次已导出）：
  - [round203-wallpaper-dark-blur0-pixel0.png](<evidence/screens/round203-wallpaper-dark-blur0-pixel0.png>)
  - [round203-wallpaper-dark-blur8-pixel12.png](<evidence/screens/round203-wallpaper-dark-blur8-pixel12.png>)（可见平滑模糊 + 块状像素化）

## 4. 残余（结构性）

1. **像素化作用域**：原生对「cover 裁好的屏幕位图」按 `factor` 缩小再放大，屏幕上的块大小恒为 `factor` px；MyGo 在**解码后的源图**上按块取中心色再裁切，屏幕块大小 = `factor × 源图→窗口缩放比`。因此同一 12% 在 320×200 小图与 4K 大图上的观感不同（截图用的是 320×200 种子图，块更细）。真正的对齐需要在 `WallpaperBackgroundPainter` 里先做 cover 裁切/缩放，再按屏幕像素块化（会改动缓存键与 surfaces 采样，留作后续）。
2. **模糊核**：MyGo 是可分离 running-sum 盒式模糊（半径 `round(p*0.4)`），原生是 CSS 高斯 `blur(p/100*40 px)`；且原生有 48px 出血消除边缘光晕，MyGo 无出血，边缘窗口会截断。视觉接近但非逐像素一致。
3. **透明抬升不持久化**：原生把 20 写回 Host（一次性），MyGo 只在渲染层抬升（关掉透明主题即恢复用户值）。chrome 层没有 Host 写权限，若要持久化需在 cmd 的设置保存路径里补一次 `nudge`。
4. 页面级合成：原生壁纸是页面 `#root` 之下的图层、面板在其上；MyGo 在窗口绘制前画壁纸、再由各表面采样（`Backdrop*`/`Glass`），透明度合成本身已等价，但上面两条（块作用域、出血）仍会造成观感差异。
