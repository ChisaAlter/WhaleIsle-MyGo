# Round228 · 暗色底与玻璃合成：token 核对与结论

结论：MyGo 的暗色 token 与原生完全一致（base 就是 `rgb(21,21,23)`），round225 的「统一 #151517」不是调色板错误；native 9 点里的 hero 12,12,12 来自 base 叠加 `--dsw-alias-bg-mask-1`（rgba(0,0,0,0.5)）的**合成**，sidebar 65,65,65 与 `--dsw-specific-sidebar-nav-item-active`（rgb(67,69,74)）只差 2–4，caption 17,31,48 是带业务蓝的着色面。把 #0c0c0c 直接写进 `BG` 会被既有的 token 对照测试（`tokens_test.go` 要求 dark bg-base = rgb(21,21,23)）判定为回归，并破坏侧栏/面板的相对明度，因此本轮**没有**改调色板，而是补上了原生遮罩合成的纯函数与测试，并把逐面合成留给下一片（需要 native 区域图对照）。

## 1. 原生暗色 token（`reference/electron/src/shared/dsh-webui-tokens.css`，`body[data-ds-dark-theme]`）

| token | 值 | MyGo `deepseekDark` |
|---|---|---|
| `--dsw-alias-bg-base` | rgb(21,21,23) | `BG "#151517"` ✓ |
| `--dsw-alias-bg-layer-1` | rgb(35,35,36) | `Field "#232324"` ✓ |
| `--dsw-alias-bg-layer-2` | rgb(44,44,46) | `Bubble "rgb(44, 44, 46)"` ✓ |
| `--dsw-alias-bg-mask-1` | rgba(0,0,0,0.5) | 未建模（见 §2） |
| `--dsw-alias-bg-module-platform` | rgb(53,54,56) | light 的 Field 用它；dark 的 Field 用 layer-1 ✓ |
| `--dsw-specific-sidebar-fill` | rgb(27,27,28) | `Sidebar "#1b1b1c"` ✓ |
| `--dsw-specific-sidebar-nav-item-active` | rgb(67,69,74) | Hover/Pressed 用白色 alpha，选中项 ≈ 67,69,74 |
| `--dsw-alias-glass-opacity`（默认） | 80% | `glass` 滑块 80，配置 60 |
| `--dsw-alias-state-business-primary` | rgb(103,158,254) / harness 解析为 122,170,255 | `Accent "#7aaaff"` ✓（按 harness 级联值） |

## 2. 合成公式与 9 点解释

- hero/card `(12,12,12)`：`rgb(21,21,23)` 被 `rgba(0,0,0,0.5)` 遮罩 → (10.5,10.5,11.5)，经 sRGB/取整即 12,12,12。本片新增 `chrome.FlatBackdropFill(base, dark)` 精确复现该合成（dark: `rgba(0,0,0,128).Over(base)`，light 原样返回），单测 `TestFlatBackdropFill` 断言 dark #151517 → ≤(13,13,14) 且 light 不变。
- sidebar `(65,65,65)`：与 `nav-item-active rgb(67,69,74)` 相距 2–4（±2 容差内近似），说明采样点落在**选中/悬停项**而不是 sidebar 填色（填色是 27,27,28）。
- caption `(17,31,48)`：R 低于 base 21、G/B 明显偏高，是**业务蓝着色面**（accent 122,170,255 与暗底混合）而不是纯玻璃：`base*0.5 + accent*α` 在 α≈0.12–0.16 时落在该区间。
- 因此「把 base 改成 #0c0c0c」只能对上 hero 一点，却会让 sidebar/选中项/文字对比整体偏移，且与 token 表冲突。

## 3. 测试与门禁

- 新增 `TestFlatBackdropFill`（[flat_base_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chrome/flat_base_test.go>)）：dark 遮罩合成到 ~(11,11,12)，light 返回自身。
- 既有 `tokens_test.go` 的 dark bg-base = rgb(21,21,23) 保持通过（这也是本轮拒绝改 `BG` 的依据）。
- 门禁：`go test ./internal/ui/theme ./internal/ui/chrome ./internal/ui/frame ./cmd/whaleisle -count=1` → 2.017s / 5.123s / 14.816s / 21.450s 全 ok；`go vet` 同四包 → 退出 0。
- 未提交、未构建 passN；未改原生应用、runtime/vendor、third_party。

## 4. 截图

本片无视觉改动（安全起见未接线），沿用 [round225-first-run-dark-flat.png](<evidence/screens/round225-first-run-dark-flat.png>)（9 点全 21,21,23，0 青色）。下一片接线后应得到 hero ≈12,12,12。

## 5. 残余（下一片的明确任务）

1. **遮罩接线**：在主窗口绘制路径用 `FlatBackdropFill` 取代裸 `colors.BG`（仅 dark + 无壁纸 + 无渐变），并让 sidebar/面板继续用不透明白底 token（侧栏选中项用 67,69,74）。需要 native 区域图（每个采样点的控件）确认后一次接线，避免把遮罩铺到玻璃面板上。
2. **玻璃不透明度 60**：`glassOpacity` 目前只在有壁纸/渐变时参与（`MaterialAlpha`），无壁纸时面板是不透明 token 色；若要 matched native，需要无壁纸也按 60% 合成，代价是大量既有视觉测试的期望值要重算。
3. **caption 着色面**：native caption 的蓝色来自业务蓝着色；MyGo 目前 caption 是纯文字色，需要确认该区域在原生里是哪种控件（状态条/提示条）再复刻。
