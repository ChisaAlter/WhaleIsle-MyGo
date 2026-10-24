# Round221 · 首次运行/无壁纸的平铺底色默认值

结论：round216 的「无图即平铺」没有生效，因为效果值在更上游被种成了 `gradient`（Host 默认结构与首次运行外观），而且 `IsFirstRun()` 会覆盖配置里显式写的 `backgroundEffect:"none"`。本轮把「未显式选择」与「显式选择」分开：默认/种子/空串 → 平铺底色；显式 `gradient`（配置或 Host）→ 保留渐变；显式 `none` 一定生效。

## 1. 优先级表（无壁纸图时）

| 来源 | 值 | 结果 |
|---|---|---|
| 默认 `wallpapers.Defaults()` | `Effect:"none"`, `EffectExplicit:false` | 平铺主题底色 |
| 首次运行 `theme.FirstRunAppearance()` | `Effect:"none"`（保留 dune/orbs 供以后显式选择） | 平铺 |
| Host 文档没有 `backgroundEffect` 键 | 种子为空串，`BackgroundEffectExplicit:false` | 平铺（HostSettings.Background() 把空串归一为 none） |
| Host 文档 `backgroundEffect:"gradient"` | 显式 → `true` | 渐变 |
| Host 文档 `backgroundEffect:"none"` | 显式 → `true` | 平铺（显式 none 胜出） |
| 本地配置 `goWallpaperBackground.backgroundEffect:"none"` | 键存在 → `EffectExplicit:true` | 平铺（不再被 `IsFirstRun()` 覆盖） |
| 本地配置 `backgroundEffect:"gradient"` | 显式 → `true` | 渐变 |
| 有 seed 图 / Host 图 | 任意 | 图片优先，效果保留 |

## 2. 修复

- [wallpapers/preferences.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/wallpapers/preferences.go>)：新增 `EffectExplicit bool `json:"-"`` 与 `NormalizeEffect(path, prefs)`——路径为空且未显式选择（或效果为空串）时把 `Effect` 归一为 `none`；有图直接返回原值。
- [theme/host_preferences.go](<C:/AI/WhaleIsle-MyGo/internal/ui/theme/host_preferences.go>)：`ReadHostSettings` 的种子不再预置 `gradient`+aurora（改为空串/空色/空 preset），并从原始 JSON 判断 `backgroundEffect` 键是否存在（`BackgroundEffectExplicit`）；`HostSettings.Background()` 把空串归一为 `none`。
- [theme/first_run.go](<C:/AI/WhaleIsle-MyGo/internal/ui/theme/first_run.go>)：`FirstRunAppearance` 的 `Effect` 改为 `none`（dune 预设与颜色保留，供用户显式开启）。
- [config/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/internal/config/wallpaper_background.go>)：读取 `goWallpaperBackground` 时检查 `backgroundEffect` 键是否出现，出现即 `EffectExplicit=true`（含 `"none"`）。
- [chrome/background.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background.go>)：`ConfigureBackdrop` 入口先 `wallpapers.NormalizeEffect`，再决定 `active`（因此渐变只有显式时才 active）。
- [cmd/whaleisle/wallpaper_background.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/wallpaper_background.go>)：`wallpaperBackdropPrefs` 改为委托 `NormalizeEffect`；`wallpaperEditor` 的首轮覆盖只保留 `!HasWallpaperBackground()`（去掉 `IsFirstRun()`，配置里的显式 `none` 不再被覆盖）。

## 3. 测试

- `TestNormalizeEffect`（wallpapers）：未设置/空串/非显式 gradient → none；显式 gradient → gradient；显式 none → none；有图保留效果。
- `TestFirstRunFlatBackdrop` + `TestHostBackgroundExplicitness`（theme）：首次运行为 none；Host 无键 → 空/非显式且 Background() 归一 none；Host `gradient`/`none` 均标为显式并保留。
- `TestWallpaperBackdropPrefs`（cmd）：空/空白路径 → none；显式 gradient 无壁纸 → gradient；seed 图与 Host 图 → 保留效果；Host 空 → 平铺；Host 非空 → 图片优先。
- `TestNoWallpaperFlatBackdrop`（chrome）：暗/亮主题下平铺 painter 与不画 painter 的中心像素一致，aurora 对照仍会改变像素。
- 既有用例按新语义修正：`background_gradient_test` 改为显式 gradient 场景、`first_run_test` 期望 none、`glass_test`/`transparent_test`/`background_motion_test` 的渐变场景补 `EffectExplicit=true`。

## 4. 截图与数值

- [round221-first-run-dark-flat.png](<evidence/screens/round221-first-run-dark-flat.png>)（1440×860，暗色）：采样 hero(1000,430)、sidebar(100,430)、footer(720,820)、card(300,200)、top(200,120) 全部 **#151517**，无青色带。
- `evidence/accept-D-MAIN-210/baseline.png` 仍不在工作区（round216 已记录），无法逐点对比；按 round210 记录 native 为平铺深色，本轮 MyGo 已是同一平铺底色。

## 5. 残余

1. 首次运行的 `Preset:"dune"`/颜色仍随 profile 保存，只是 `Effect:"none"` 不绘制；用户显式开启渐变时直接可用。
2. `EffectExplicit` 是进程内标记（`json:"-"`），保存后重新读取靠「键是否存在」推断；旧配置若显式写过 `backgroundEffect` 会被正确视为显式。
3. 透明主题的 blur 抬升仍只在渲染层生效（与 round203/207 相同）。
