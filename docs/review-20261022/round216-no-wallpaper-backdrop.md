# Round216 · 无壁纸时的平铺底色

结论：原生配置没有壁纸图（`preference: dark`，`wallpaperImage` 取默认空串），页面因此显示主题底色（#151517）；MyGo 之前在没有图片时仍绘制 aurora 渐变，这是 D-MAIN 的主要色差。本轮让「无图」走平铺底色：config 与 Host 都为空 → 不画渐变/不跑动画；有 seed 图或 Host 非空时照旧绘制图片。

## 1. 原生引用

- `.devdata/dsh-home/profiles/web/cordis.patch.yml` 的 `ui-theme` 配置只有 `preference: dark`，没有 `wallpaperImage` → 取 ui-theme 默认 `wallpaperImage: ""`（`lib/client.js` `DEFAULT_THEME_SETTINGS`）。
- 原生 `applyWallpaperLayer` 在没有 data-URL 图时不创建 `#dsh-wallpaper-inner` 位图，页面只显示主题 `--dsw-alias-bg-base`；round210 的 baseline 里 hero 区域就是平铺深色。

## 2. 修复

[wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>) 新增 `wallpaperBackdropPrefs(path, prefs)`：解析后的路径为空（config 空且 Host 空，`wallpaperSourcePath` 已保证 Host 非空时 Host 值优先）时把 `Effect` 置为 `"none"`，`chrome.WallpaperBackgroundPainter` 于是不绘制任何图层，窗口露出主题底色；有图时保留用户选择的 `gradient`。`visible=false` 的既有分支、transparent/blur 路径都未改动。

## 3. 测试

- `TestWallpaperBackdropPrefs`（[wallpaper_backdrop_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_backdrop_test.go>)）：空/空白路径 → `none`；seed 路径与 Host data URL → 保留 `gradient`；Host 空 → 平铺；Host 非空 → 图片优先。
- `TestNoWallpaperFlatBackdrop`（[wallpaper_flat_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/wallpaper_flat_test.go>)）：暗/亮两套主题各渲染一帧，平铺 painter 与不画 painter 的中心像素完全一致；同一位置换成 aurora 时必须变化（对照，证明探针有效）。
- `TestNoWallpaperFlatSnapshot`：`NO_WALLPAPER_SNAPSHOT_DIR` 导出 1440×860 暗色无壁纸帧。

## 4. 截图与数值

- 导出：[round216-no-wallpaper-dark-flat.png](<evidence/screens/round216-no-wallpaper-dark-flat.png>)（1440×860）。
- 采样：hero(1000,430)、sidebar(100,430)、card(300,200)、top(720,60) 全为 **#151517**（暗色主题 BG），即完全平铺。
- `evidence/accept-D-MAIN-210/baseline.png` 当前不在工作区（目录里没有该文件），无法逐点对照；按 round210 记录，native hero 为平铺深色 rgb(12,12,12) 量级，本轮 MyGo 已无渐变/色相差异，剩余差异应只剩文字/AA/布局。

## 5. 残余

1. 用户在设置里显式选择「渐变背景」但没有壁纸时，现在也会回落为平铺底色（与原生当前配置一致，但牺牲了该组合；若产品要保留需区分「用户显式选择」与「首次运行默认」）。
2. `theme.FirstRunAppearance()` 仍返回 gradient/dune，被本修复覆盖为平铺；如需彻底一致应把首次运行默认改为 none（在 theme 包，超出本片范围）。
3. 透明主题/模糊路径不变（有图时仍走 round203/207 的实现）。
