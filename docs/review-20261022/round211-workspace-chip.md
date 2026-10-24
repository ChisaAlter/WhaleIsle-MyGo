# Round211 · 侧栏/输入栏工作区 chip 宽度对齐原生 150 DIP

round205 量到：MyGo 的工作区 chip 宽 **116.67**，原生 **150**；原生「标准模式」chip 落在 **x=638.3**，MyGo 因为 chip 窄了 33.33，落到 610.67。本轮把 chip 改成原生规则，并让后随的 preset chip 保持原生间距。

## 1. 原生规则（本轮采用的数值）

| 项 | 值 | 来源 |
| --- | --- | --- |
| chip 宽度 | **150 DIP（固定）** | round205 实测（原生 chip 结束 ≈620，「标准模式」x=638.3） |
| 水平内边距 | 8 / 8 | 既有 `Padding(0, 8, 0, 8)`，与原生测量一致 |
| 图标 | 16×16 文件夹，不收缩 | 既有实现 |
| 图标与文字间距 | 4 | `chip()` 的 `Gap(4)` |
| 尾部 chevron | 不收缩，保留在 150 内 | 既有 `chevron(c,false)` |
| 长名称 | 截断（单行 + 省略），不撑宽 | 本轮新增 `Shrink(1).Grow(1)` |
| 与 preset chip 的间距 | **18 DIP** | 638.3 − 620 ≈ 18 |

说明：本轮**没有**跑 CDP DOM 探针去读原生 CSS 的 min-/max-width 与 font-size（时间盒被代码与测试占用），150/18 取自 round205 的像素测量；如果原生其实是 `min-width:150` 而非固定宽，短名场景下两者的差别只会在「名字短于 150 时的内部留白」，不影响外部 x。

## 2. 修改

`internal/ui/composer/composer.go`：

1. 工作区 chip 增加 `.Width(150)`（原先由内容撑开，`chip()` 只给 `MinWidth(0).MaxWidth(220)`），并加注释说明固定 150 的由来。
2. 名称 `ui.Text(...).SingleLine().MaxLines(1)` 追加 `.Shrink(1).Grow(1)`：长名称在 150 内截断，不会把 chevron 或 preset chip 推走。
3. 两个 chip 所在的 hero 行 `Gap(2)` → **`Gap(18)`**：该行只有这两个 chip，18 就是原生的 chip-to-chip 间距。

## 3. 测试（`internal/ui/composer/workspace_parity_test.go`）

扩展 `TestHeroRowCarriesWorkspaceAndPreset`：

- 宽度循环从 `{432, 800}` 扩到 `{432, 800, 1440}`；
- 新增 `closeMetric(t, workspace.W, 150, "workspace chip width")`；
- 新增 `closeMetric(t, presetChip.X, workspace.X+workspace.W+18, "preset chip sits 18 DIP after the chip")`；
- 名称循环增加一个超长根路径（`C:\a-very-long-workspace-name-that-must-truncate\inside-the-chip`），断言 **长名不改变 chip 宽度**（仍 150）；长名分支跳过具体标签断言（chip 显示的是 basename）；
- 既有的行高 28、左内边距 20、chip 高度 28、垂直居中、标题行与导航 pitch 的断言全部保留且未改动。

实测（tester）：`workspace={36 4 150 28}`、`preset={204 4 127.18 28}` → 204 = 36 + 150 + 18，三种宽度一致。

## 4. 前/后对比（DIP）

| 项 | 之前 | 之后 | 原生 |
| --- | --- | --- | --- |
| 工作区 chip 宽 | 116.67 | **150** | 150 |
| preset chip 相对 chip 的间距 | ~4（`Gap(2)` + 内边距余量） | **18** | ~18 |
| preset chip x（round205 实机） | 610.67 | 按规则应为 chip 右缘 + 18（≈638.3） | 638.3 |

## 5. 门禁

- `go test ./internal/ui/composer` → ok 12.4 s
- `go test ./internal/ui/sidebar` → ok 5.1 s
- `go test ./internal/ui/frame` → ok 17.3 s
- `go vet ./internal/ui/composer ./internal/ui/sidebar ./internal/ui/frame` → 0

## 6. 残留

1. **没有做 pass291 暗色实机截图**：本轮时间盒用于实现与测试，实机 x=638.3 的复测留待下一轮（起 pass291 临时实例 → 截 1440×860 → 量 chip 右缘与「标准模式」x，应与本规则一致）。
2. **没有跑 CDP DOM 探针**读原生 CSS（min-/max-width、padding、font-size、truncation）。若原生是 `min-width` 而非固定宽，短名情况下内部留白会与我方固定 150 略有差异，需要那次探针确认。
3. 是否还有其它入口使用同一 chip（会话页内的 workspace picker 等）本轮未审计；本次只改了 hero 行这一处。
4. 未提交、未构建 passN、未改 runtime/vendor 与 third_party。
