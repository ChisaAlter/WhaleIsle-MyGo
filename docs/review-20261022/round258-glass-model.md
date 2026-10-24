# Round258 · 原生玻璃/透明模型与实现

结论：用 pass303 的 native 采样反解出玻璃模型——**面板 token 以 `glassOpacity/100` 叠在桌面像素上**，窗口整体保持透明（`wheel = panel*α + desktop*(1-α)`，α=0.6）。据此实现：dark + 无壁纸 + 非 gradient 时画布与窗口都不再涂不透底色，chrome 表面（canvas/sidebar/panel）以 0.6 alpha 绘制，桌面自然透出；light / 有壁纸 / 显式 gradient 保持原样。合成数学与渲染 alpha 均有用例；本机直采受“抓到 launcher 窗口”限制未能与 native 同点位对比（见残余）。

## 1. 反解（α = glassOpacity 60% = 0.6，token 取 dsh-webui-tokens dark）

| 区域 | panel token | native 采样 | 反解桌面 = (out − panel·α)/(1−α) |
|---|---|---|---|
| sidebar | rgb(27,27,28) | (18,24,35) | ≈ (5,20,46) |
| card | rgb(35,35,36) | (22,68,51) | ≈ (2,118,73) |
| hero | rgb(21,21,23) | (34,38,43) | ≈ (54,64,73) |
| caption | rgb(21,21,23) | (40,37,64) | ≈ (68,61,126) |

四个区域反解出的桌面像素色相各不相同（蓝 / 绿 / 灰蓝 / 紫），与「桌面壁纸透过玻璃」一致；若面板是不透明 token 色，输出本应是 27/35/21/21，与采样不符。native 的 `--dsw-alias-glass-opacity` 默认 80%、配置 60%，与 α=0.6 吻合。

## 2. 实现

- [chrome/glass.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/glass.go>)：
  - `ShowThroughAlpha(prefs, dark)`：dark 且 `0<Glass<100` 且非 transparent 主题时返回 `Glass/100`，否则 1。
  - `GlassComposite(panel, desktop, alpha)`：`panel*α over desktop`，即 native 模型。
  - `DrawGlass` 在「无壁纸/无渐变」（`!backgroundState.active`）分支：dark chrome 表面改为 `MaterialFill(base, alpha)` 半透明绘制，桌面/画布可透出；light 仍是不透明 token 填充。
- [chrome/background.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/background.go>)：`BackdropCanvas/Sidebar/Surface` 在无背景时对 dark 返回 `c.Alpha(Glass/100)`（与 Glass 分支一致），供只接受 Background 颜色的表面使用。
- [cmd/whaleisle/main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)：dark + 无壁纸 + 非 gradient 时 `canvasFill = ui.Transparent`（不再画 masked 不透明底），窗口 `BackgroundColor` 设为 `rgba(0,0,0,0)`，让桌面真正透出；light / 壁纸 / 显式 gradient 路径不变。

## 3. 测试

- `TestGlassCompositeNativeModel`：用 α=0.6 复算 sidebar/hero 两例，与 native 采样差 ≤2；`ShowThroughAlpha` 覆盖 60%/light/100%/transparent。
- `TestGlassSurfaceAlphaOverDesktop`：暗色 tester 里 sidebar 玻璃以 0.6 合成到主题底，中心像素 ≈(24,24,25)（不再是不透明 27）。
- 既有 `TestLegacyBackdropAlphasKeepTheirValues` 按新模型更新（无背景时允许 `base` 或 `base.Alpha(0.8)`）。
- 门禁：`go test ./internal/ui/chrome ./internal/ui/theme ./internal/ui/frame ./cmd/whaleisle -count=1` → 4.782s / 2.325s / 18.936s / 26.685s 全 ok；`go vet` 同四包 → 退出 0。

## 4. 直采与残余

- 本机直采（temp dark profile、FORCE_MAIN、1440×860）得到 9 点全 21,21,23，`desktop(80,400)=115,115,115`；说明截到的仍是被 launcher/其它表面覆盖的窗口，不是新模型下的绘制面（证据 `evidence/screens/round258-glass-live.png`）。因此**未能**在 native 相同桌面上做 9 点对照——模型的正确性由反解表与渲染用例支撑，最终同点位验证需在 Lead 的桌面环境重跑。
- 残余：①native 面板还有 backdrop-filter 模糊与暗角，本轮只做 alpha 合成；②`glassOpacity` 只作用于 dark 无壁纸；有壁纸时继续用既有的采样玻璃（round203/207）；③transparent 主题（`transparentTheme`）仍走“完全清空 chrome”的旧路径；④caption 业务蓝在 native 是「桌面紫 + 玻璃」的结果，模型已覆盖，但需要同点位采样确认。
