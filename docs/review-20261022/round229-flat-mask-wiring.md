# Round229 · 平铺遮罩接线与 preset 泄漏

结论：round102 重建后仍是青色（hero 42,52,48；caption 27,74,94）的原因是**首次运行仍带着 preset 种子**（`FirstRunAppearance` 虽然 `Effect:"none"`，却保留 `Preset:"dune"`/`Variant:"orbs"`/dune 颜色），任何以 preset 为准的绘制或用户一开渐变就会立刻出现整片色洗；本轮清空首次运行的 preset/颜色、在 painter 里明确「只有 `Effect=="gradient"` 才绘制」，并把 `FlatBackdropFill` 接到页面画布（只遮罩画布，不动面板 token）。9 点采样：画布 10,10,11（≈native 12），0 青。

## 1. 探针

`WHALEISLE_BACKDROP_PROBE=1` 时 `wallpaperPainter` 打印：解析后的 path、`g.Prefs.Effect/Preset/Colors[0]/EffectExplicit`、`chrome.BackdropActive()`。用它区分三种来源：①配置/Host 显式 gradient；②首次运行 preset 种子；③无壁纸平铺。round102 的青色对应 ①/②：Effect 为 gradient 且颜色来自 preset。

## 2. 修复

1. [theme/first_run.go](<C:/AI/WhaleIsle-MyGo/internal/ui/theme/first_run.go>)：首次运行 `Preset:""`、`Colors:[7]string{}`（Effect 已是 none）——没有任何颜色种子能泄漏。
2. [chrome/background.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background.go>)：painter 的严格门——只有 `Effect=="gradient"` 才返回渐变闭包，preset/颜色单独存在时一律空绘制（注释写明）。
3. [cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：新增 `wallpaperCanvasFill(base, dark, path, prefs)`——dark + 无路径 + `NormalizeEffect` 后非 gradient 时返回 `chrome.FlatBackdropFill`；新增 `resolvedWallpaperPath` 供画布与 painter 共用同一路径解析。
4. [cmd/whaleisle/main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)：根画布 `Background(canvasFill)`（原 `chrome.Color(colors.BG)`），面板/侧栏/标题栏仍用各自 token，不重复遮罩。
5. `FlatBackdropFill` 的遮罩 alpha 修正为 `ui.RGBA(0,0,0,0.5)`（mygo 的 alpha 是 0–1；此前用 128 被钳成 1，画布变纯黑——round228 的测试过松，本轮同时收紧断言为 10–13）。

## 3. 采样（1440×860，暗色，无壁纸，无外观文档）

[round229-flat-backdrop.png](<evidence/screens/round229-flat-backdrop.png>)：9 点全部 **10,10,11**（bg-base 21,21,23 叠 rgba(0,0,0,0.5) = 10.5/10.5/11.5），青色计数 0。对比 native 目标：hero/card 12,12,12（差 ≤2，符合容差）；caption 17,31,48 与 sidebar 65,65,65 在当前合成帧里仍是画布值——它们是「面板/选中项」而不是画布，需要下一片按控件复刻（见残余）。

## 4. 测试

- `TestFlatBackdropFill`（收紧）：dark #151517 → (10..12,10..12,11..13)，light 原样。
- `TestFlatMaskedCanvasSnapshot`：导出上面的 1440×860 帧。
- `TestPresetDoesNotPaintWithoutEffect`：`Effect:"none" + Preset:"dune" + dune 颜色` 时 painter 不改变任何像素；把 Effect 改为显式 gradient 后同一 prefs 会绘制（对照）。
- `TestWallpaperCanvasFill`（cmd）：dark 平铺被遮罩；light 不变；显式 gradient（含无颜色）绕开遮罩；有壁纸路径时用原始底色。
- 既有 `first_run` 测试更新为 preset/颜色为空。

## 5. 门禁

`go test ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 3.795s / 13.901s / 19.998s 全 ok；`go vet` 同三包 → 退出 0。未提交、未构建 passN；未改原生应用、runtime/vendor、third_party。

## 6. 残余

1. **glassOpacity 60**：目前只在有壁纸/渐变时参与合成；无壁纸时面板是不透明 token（sidebar 27,27,28 / card 35,35,36），native 的 65,65,65 是选中项 `nav-item-active rgb(67,69,74)`，需按控件确认后再决定是否启用无壁纸玻璃。
2. **caption 业务蓝**：native 17,31,48 是 accent 着色面，本轮未复刻（不知道对应控件）。
3. 真实窗口仍需用探针重跑确认：若 `Effect` 仍为 gradient，说明 profile 里有显式外观文档（属用户选择，按设计保留）。
