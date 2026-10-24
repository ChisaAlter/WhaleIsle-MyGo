# Round222 · Hero chip 行：原生 32.67 内缩 + 3.33 chip 间距

round217 的 UIA 实测（1440×860）：

| 元素 | 原生 | MyGo（修复前） | 差 |
| --- | --- | --- | --- |
| workspace chip | x=502, w=150 | x=489.33, w=150 | 行内缩 32.67 vs 20 |
| 标准模式 chip | x=655.33 | x=675.33 | chip 间距 3.33 vs 36 |

## 1. 修改（`internal/ui/composer/composer.go`）

1. hero 行左侧内缩：`Padding(0, 16, 0, 20)` → **`Padding(0, 16, 0, 32.67)`**（32.67 = 502 − 469.33，即原生 chip 相对行左边距）。
2. 两个 chip 的间距：`Gap(18)` → **`Gap(3.33)`**（655.33 − (502 + 150) = 3.33）。
3. chip 宽度 150、长名截断（`.Shrink(1).Grow(1)`）保持不变。

## 2. 测试

- `internal/ui/composer/workspace_parity_test.go`（扩展既有用例）：三档宽度 432/800/1440，断言 `workspace.W == 150`、`workspace.X-row.X == 32.67`、`preset.X == workspace.X+150+3.33`（间距容差收紧到 3–3.7），以及不溢出；长名用例仍要求宽度不变。tester 结果：`workspace={48.67 4 150 28}`、`preset={202 4 127.18 28}`。
- 新增 `internal/ui/frame/hero_chip_row_test.go`：在完整的 1440 宽外壳里找 hero 行，断言同样的相对量（32.67 / 150 / 3.33，容差 0.01）并检查不溢出。实测 `row={536.5 455.5 641 28}`、`workspace={569.17 455.5 150 28}`、`preset={722.5 455.5 101.47 28}`。

## 3. 绝对 x=502 的现状（重要）

修复后行内相对量已与原生一致，但完整外壳里 chip 的绝对 x 是 **569.17**（原生 502）：两者相差 **+67.17**，且 preset 也恰好同偏移（722.5 vs 655.33）。也就是说 chip 行本身已经对齐，剩下的整段位移来自 **composer 卡片的位置/宽度**（本外壳 card.x=536.5、w=641；要让 chip 落在 502，卡片 x 需约 469.33）。这是一个独立的「composer 卡片水平居中/宽度」问题，不属于本轮 chip 行内缩，已记入残留。

## 4. 门禁

- `go test ./internal/ui/frame` → ok 13.4 s；`./internal/ui/composer` → ok 8.0 s；`./internal/ui/sidebar` → ok 2.8 s。
- `go vet ./internal/ui/frame ./internal/ui/composer ./internal/ui/sidebar` → 0。
  （中途曾因其他同事正在编辑的 `internal/ui/theme/host_preferences.go:73 undefined: strings` 让 vet 失败一次，20 s 后重跑即 0；非本切片改动。）

## 5. 残留

1. **composer 卡片水平位置**：+67.17 的整体偏移未处理；这决定 chip 的绝对 x 是否等于 502（以及 preset 是否等于 655.33）。需要单独量卡片宽度/居中规则（round217 的 UIA 数据可继续用）。
2. **未做 pass293 实机截图**：pass293 构建于 05:26，早于本轮修改，直接截图看到的仍是旧几何；要实机验证需用当前树构建临时 exe 再截（约 10 分钟）。
3. 右侧内缩（16）沿用旧值，本轮没有原生数据支撑，未改。
4. 未提交、未构建 passN、未改 runtime/vendor 与 third_party；无临时文件残留。
