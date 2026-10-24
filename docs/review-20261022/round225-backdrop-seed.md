# Round225 · 背景路径的来源审计与 Host 壁纸显式化

结论：paint 期出现非空路径的唯一入口是 **Host 文档的 `wallpaperImage`**（或本地 `goWallpaperPath`）；本轮用探针固定了这一点，并把 Host 壁纸改成「只有文档里真的带 `wallpaperImage` 键才生效」，种子/默认值不再能覆盖本地平铺底色。首次运行（配置空 + Host 无键）现在路径为空、`Effect` 归一 `none`，9 点采样 0 青色。

## 1. 探针与来源审计

`wallpaperPainter` 新增环境变量开关 `WHALEISLE_BACKDROP_PROBE=1`：每次解析后打印

```
backdrop probe: path=%q hostAccepted=%v hostImageLen=%d configSource=%q
```

来源链路（逐一核对）：

| 入口 | 代码 | 污染可能 |
|---|---|---|
| paint 路径 | `main.go:3262 theme.WindowWallpaperPath(cfg.WallpaperPath)` → `wallpaperPainter` | 仅当 `goWallpaperPath` 非空 |
| Host 壁纸 | `theme_font_preferences.go:90 s.hostWallpaperImage = h.WallpaperImage` | **已修**：现在只有 Host 文档含 `wallpaperImage` 键才赋值 |
| Host 效果 | `theme.ReadHostSettings` 种子 | round221 已改为空串 + `BackgroundEffectExplicit` |
| 首次运行 | `theme.FirstRunAppearance` | round221 已改为 `Effect:"none"` |
| 配置默认 | `wallpapers.Defaults()` / `config.WallpaperBackground()` | `Effect:"none"`、`EffectExplicit` 按键存在判定 |
| 内置图片种子 | 仓库内不存在（无 wallpaper 资源、无 `go:embed` 壁纸） | 无 |

工作区里的 `.devdata/config.json`、`.devdata-clean/config.json` 的 `goWallpaperPath` 均为空；`internal/ui/chrome/assets` 只有 logo/rail-wash（rail-wash 采样为灰绿 189,192,198，不是青色）。因此 round223 的青色只能来自 Host 文档注入的默认 `wallpaperImage`（或外部写入的 `goWallpaperPath`），而不是 aurora 渐变——`active = path != "" || Effect == "gradient"` 中当 `Effect` 为 none 时只剩路径分支。

## 2. 修复

- `theme.HostSettings` 新增 `WallpaperImageExplicit`（`json:"-"`），`ReadHostSettings` 只在原始 JSON 含 `wallpaperImage` 键时置真；`settings.HostThemeState` 同步携带该字段。
- `cmd/whaleisle/theme_font_preferences.go`：仅当 `h.WallpaperImageExplicit` 才写 `s.hostWallpaperImage` / `s.desktopDraft.wallpaperPath`；种子空值不再清空或覆盖本地壁纸。
- 效果链路沿用 round221 的 `NormalizeEffect`：无路径 + 未显式 → `none`；显式 gradient/Host 图仍照常绘制（用户图片不受影响）。

## 3. 测试

- `TestHostWallpaperExplicitness`（theme）：无 `wallpaperImage` 键 → 空且非显式；带键 → 显式保留。
- 既有 `TestNormalizeEffect`、`TestWallpaperBackdropPrefs`、`TestNoWallpaperFlatBackdrop`（暗/亮像素探针 + aurora 对照）继续覆盖空路径/显式效果/种子图。
- 9 点渲染探针：见下。

## 4. 截图与 9 点采样

[round225-first-run-dark-flat.png](<evidence/screens/round225-first-run-dark-flat.png>)（1440×860，暗色）：

| 点 | RGB |
|---|---|
| (720,200) hero | 21,21,23 |
| (300,200) card | 21,21,23 |
| (100,430) sidebar | 21,21,23 |
| (1000,430) right | 21,21,23 |
| (720,600) | 21,21,23 |
| (300,700) caption 区 | 21,21,23 |
| (720,760) footer 区 | 21,21,23 |
| (300,820) footer 左 | 21,21,23 |
| (720,820) footer 中 | 21,21,23 |

青色计数 **0**（判据 `G>R+25 && B>R+25`）。与 native round223 数值对比：native hero 12,12,12、caption 17,31,48、sidebar 65,65,65——MyGo 现在是单一主题底色 #151517，不含任何青/蓝渐变；剩余差异是底色明度（#151517 vs #0c0c0c）与面板/亚克力层，需要另立主题色对齐项，不是背景路径问题。

## 5. 残余

1. 主题底色 #151517 与 native 采样 #0c0c0c 仍有 ~9 级差（`theme.Resolve` 的 dark BG），属主题色板问题。
2. native caption 17,31,48 / sidebar 65,65,65 是「面板在壁纸/底色上的合成值」，MyGo 目前面板是不透明底色叠加；若要对齐需要逐面板材质校准。
3. `WHALEISLE_BACKDROP_PROBE` 只在显式设置环境变量时打印，默认无日志。
