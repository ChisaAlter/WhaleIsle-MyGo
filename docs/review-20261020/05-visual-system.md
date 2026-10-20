# 05 视觉系统与特效：原生 vs 原版

审查人：Lead｜2026-10-20｜证据基准见 [README](README.md)

## 先声明截图条件不同

| | 原版（3080） | 原生（3090） |
| --- | --- | --- |
| 主题 | 深色 | 浅色 `deepseek` |
| 背景 | 用户壁纸（动漫图） | 无壁纸，`goWallpaperPath` 为空，背景是 gradient/orbs |
| 尺寸 | 1441×912（1.5× DPI） | 1298×822 |

所以**像素级差异不能直接算差距**。下表以源码实现为准，截图只用于观察"观感结果"。

## 五个最致命差距

1. **表面材质是"假玻璃"（P0）**。原版是透明窗口 + 壁纸 + 每层 alpha token + 菜单 `backdrop-filter: blur(40px) saturate(150%)`（`reference/electron/src/shared/dsh-webui-tokens.css:58-59,111-112`、`ui-theme/src/styles/gradient-shadow-text.css:20`），窗口材质由桌面注入层控制（`reference/electron/src/renderer/assets/window-material.css:3-11`）。原生则把壁纸**预先**高斯模糊成一张 bitmap，再给每个表面一个固定 alpha（`internal/ui/chrome/background.go:23-73`：canvas 最多 45%、surface 直接用 `Glass/100`、sidebar 另有一套曲线），**不对表面之下实际内容取样**，也没有菜单/弹层模糊。叠加结果是"一层层发白的卡片"，而不是"透过玻璃看壁纸"。
2. **默认外观与原版不同，放大了差距（P0）**。原生开发实例是浅色 + 无壁纸 + `gradient/orbs` 背景（`.devdata/config.json:21,43`），原版是深色 + 壁纸。首启默认值和"恢复用户外观设置"的链路需要对齐，否则用户第一眼就是两套观感。
3. **指针特效只有开关，没有渲染（P1）**。原版有完整的光标特效系统（`ui-theme/src/cursor-fx.ts` 16 KB、`cursor-fluid.ts` 35 KB、`client/cursor-fx-presets.ts`）；原生只有偏好字段与设置页（`internal/ui/settings/pointer_effects.go:14-16`、`internal/ui/theme/host_preferences.go:33-39`），全仓库没有任何消费 `CursorEffect` 的绘制代码——打开开关不会有任何效果。
4. **按钮金属漆同样空转（P1/P2）**。`metallicPaintEnabled` 只出现在偏好与设置页（`wallpapers/preferences.go:10`、`settings/pointer_effects.go:39`、`wallpaper_background.go:491`），没有渲染实现；原版有 `ui-theme/src/styles/metallic-paint.css`。
5. **动效缺一层 token 体系（P1）**。原版 `ui-theme/src/styles/motion.css:12-94` 定义 overlay/popover/swap/flip 的时长与 easing，弹层、切换、翻面全站复用；原生没有等价物，弹层与面板切换基本是立即重绘（beam 是唯一的例外，见下表 #12）。

## 明细表

| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 表面 alpha 模型 | 每层用 `color-mix(... 80%, transparent)` 等 token，透明窗口让壁纸透出；`dsh-webui-tokens.css:58-59,111-112` | 固定曲线：canvas 最高 45%，surface=`Glass/100`，sidebar 另一套；`chrome/background.go:23-73` | 没有"同一层材料"的语义，透明随表面不同而漂移；浅色下几乎不透 | P0 | 建 surface material token（canvas/sidebar/panel/menu/terminal 各一档），与壁纸亮度联动 |
| 2 | 菜单/弹层背景模糊 | `--dsw-menu-backdrop-filter: blur(40px) saturate(150%)`，`gradient-shadow-text.css:20`；组件用 `backdrop-filter: var(...)`，如 `ui-chat/.../stat-dialog.module.css:27` | 弹层只有暗色遮罩 `ui.RGBA(0,0,0,.4)`，面板不透明；`settings/modal.go:21-27`、`ui/market/presentation.go:287-288` | 弹层"糊"在背景上，没有玻璃层次 | P0 | 给 Painter 增加背景模糊采样（或用分层窗口/离屏纹理），先覆盖菜单/弹层 |
| 3 | 壁纸渲染管线 | `ui-theme/src/styles/wallpaper.css:28,91,150,197,249`（blur 变量、遮罩、渐变叠加）；`wallpaper.ts` 27 KB（来源/裁切/收藏） | `chrome/background.go:77-120` 只做 Cover + 可选 blur/pixelate；`theme/wallpaper.go` 仅 905 B | 缺遮罩层、渐变叠加、透明主题联动 | P1 | 把 wallpaper.css 的图层结构翻译成 Painter 分层（底图→模糊→遮罩→渐变→内容） |
| 4 | 透明主题 | `html[data-dsh-transparent]` 分支（`reference/electron/src/shared/harness-desktop-forks.js:99-100`） | 仅把 alpha 归零；`background.go:28-30,63-65` | 透明主题下并没有真正"透出桌面"，只是不画底色 | P1 | 透明需要窗口 alpha + 桌面采样，取决于 MyGo 窗口能力，先做能力验证 |
| 5 | 背景效果 | `effect-presets.ts`、`wallpaper.ts`（gradient/orbs/aurora…） | `background.go:99-120` 已实现 gradient + 4 种 variant（aurora/rays/chaos） | 数量与参数接近，缺预设库与编辑器 | P2 | 对照 `effect-presets.ts` 补齐预设 |
| 6 | 主题 token 深度 | `design-platform.css` 425 行静态 token + alias 层 + `derive.ts` | `theme/theme.go:38-56` 17 个语义色 + `family_palette.go:32-89` 派生 | 原生语义少，边界场景（浮层/禁用/选中）只能就近取色 | P2 | 按原版 alias 表扩到 40-60 个语义 token |
| 7 | 字体 | Montserrat 三字重 woff2（`ui-theme/src/styles/montserrat-*.woff2`）+ 完整字体回退 | 系统字体 + `InterfaceSize` 缩放；`theme/editor_font.go` 只管编辑器 | 品牌字与数字排版不同 | P2 | 内置 Montserrat 子集，品牌与指标数字使用 |
| 8 | 渐变/金属文字 | `gradient-shadow-text.css` 14 KB（渐变文字、阴影文字、动画） | 品牌是位图 avatar + 纯色文本；`chrome/brand.go:28`、`sidebar/sidebar.go:152-154` | 原版首页/启动器的标题质感缺失 | P2 | 先做静态渐变文字（线性渐变填充） |
| 9 | 动效 token | `motion.css:12-94`（overlay/popover/swap/flip + `--ds-ease-in-out`） | 无对应层；仅 beam 自绘时间轴 | 弹层、切换、拖拽无过渡 | P1 | 在 `ui/widgets` 加 motion token，统一时长/缓动 |
| 10 | 指针特效 | `cursor-fx.ts`、`cursor-fluid.ts`、`cursor-fx-presets.ts` | 仅设置项，无渲染；`settings/pointer_effects.go:14-16` | 开关无效 | P1 | 顶层 overlay 层 + Painter 粒子/拖尾，先 trail 再 splash |
| 11 | 金属漆按钮 | `metallic-paint.css` | 仅开关；`wallpapers/preferences.go:10` | 开关无效 | P2 | 悬停时给主按钮叠加移动高光 |
| 12 | Composer beam | `ui-conversation/src/client/ComposerBeam.module.css`（bloom blur、conic 遮罩、420 ms opacity 过渡）；`ComposerBeam.tsx` | 几何与 bloom 已实现（`composer/beam.go:44-141`、`beam_raster.go`、`beam_blur.go`），但只在响应运行中显示：`composer.go:167` | 空闲态没有原版的静态描边/光泽过渡 | P1 | 对齐 ComposerBeam 的 active/idle 两态与 420 ms 过渡 |
| 13 | 滚动条 | `ui-theme/src/styles/scrollbar.css`（定制宽度/圆角/悬停） | 未定制（待实机确认） | 滚动条观感不同 | P2 | MyGo 暴露滚动条样式后补齐 |
| 14 | DPI/缩放 | 原版截图 1.5×，布局按 DIP | `theme.InterfaceSize` 缩放；缩放行为待实机 | 未验证 | P2 | 在 100/125/150% 三档截图对比 |

## 建议优先级（供总报告合并）

1. **材质层（#1/#2/#3）**：这是"一眼不像"的最大来源；先解决菜单/弹层模糊，再解决面板分层。
2. **默认外观（#2 结论）**：首启深色 + 默认壁纸/效果，恢复用户设置时保持原样。
3. **动效层（#9/#12）**：统一 token 后再补指针特效（#10）。
4. **品牌细节（#6/#7/#8/#11/#13/#14）**：在结构与交互对齐后统一收口。

## 待验证

- MyGo 是否能对"表面下层内容"做实时采样/模糊（决定 #1/#2 的实现方式，可能需要分层窗口或离屏合成）。
- ~~原生窗口是否真的无边框~~ **已由 shell 审查更正**：原生已经是无边框 + 自绘窗口按钮（`cmd/whaleisle/main.go:2164-2168`），截图里的"系统标题栏"其实是 1px 恒色硬描边（`internal/ui/chrome/frame_ring.go:12-25`）+ 12px 全空 caption 条（主路由 `Height(0)`）叠加浅色背景造成的错觉；真正的差距是外壳整体感，不是 OS 外壳。详见 [01-shell-nav.md](01-shell-nav.md)。
- 滚动条与 DPI 行为需实机截图。

## 修复后更正（2026-10-20，Wave 1/2）

- **#3 指针特效"未实现"不成立（已关闭）**：MyGo 框架本身已渲染 trail/splash 并已接线（`third_party/mygo/ui/decorative_effects.go:47`、`ui/paint.go:60`、`cmd/whaleisle/pointer_effects.go`、`cmd/whaleisle/main.go:2921`）。Wave 2 补了 `chrome.CursorEffectConfig/ApplyCursorEffects` 做钳制、accent 回退与 reduced-motion 规则（`internal/ui/chrome/cursor.go`，`cursor_test.go` 用渲染器证明 trail 上屏、禁用/reduced 不画）。
- **#4 金属漆"空转"不成立（已关闭）**：框架对每个 `ButtonBase` 已上光（`third_party/mygo/ui/base.go:27`、`paintMetallicButton`）；Wave 2 另加 `chrome.MetallicSheen` 供自绘按钮使用。
- **#2 壁纸默认模糊**：原版 `DEFAULT_WALLPAPER_EFFECT=0`（`runtime/vendor/.../ui-theme/src/wallpaper.ts:10`），默认 blur 本来就是 0；"观感不同"来自深色 + `bg-mask-1` 遮罩 + 环境渐变。Wave 2 已补遮罩图层与首启深色（`chrome/background.go`、`theme/first_run.go`）。
- **#1 假玻璃已部分解决**：新增材质模型与壁纸采样（`chrome/glass.go`：`Material`/`SampleBackdrop`/`DrawGlass`/`MenuSurface`），侧栏与右栏已改为采样玻璃；圆角面板（composer 卡、弹层）尚未接线，因为 `DrawGlass` 还不接受圆角裁剪（列在 fix-plan Wave 3）。
