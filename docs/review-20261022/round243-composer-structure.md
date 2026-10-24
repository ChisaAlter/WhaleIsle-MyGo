# Round243 · Composer 结构模型（原生 rect 推导）与两次失败的实现尝试

本轮目标：用原生 rect 推导 composer 的卡高/内缩/工具栏/发送/鱼徽标结构，再实现并 ±1 验证。**结论：结构模型已得，但实现两次都被布局约束挡住，未留下代码改动**（工作树保持干净、全部测试绿）。

## 1. 模型（由 round242 light rects 推导，DIP，1440×860）

| 元素 | 原生 rect | 推导出的规则 |
| --- | --- | --- |
| card | top 409.33（已知），left 469.33，w 774.39 | **高度 155.34**：draft 顶 417.33 = card 顶 + 8；send 底 556.67 + 8 底内边距 = 564.67 |
| draft | (482, 417.33, 745.33, 52.67) | 左内缩 **12.67**（482−469.33），右内缩 **16.39**（774.39−12.67−745.33），顶 8 |
| toolbar 行 | send (1188.67, 522, 34.67, 34.67) | draft 底 470 → send 顶 **522 = +52**；send 右缘 1223.34，比 draft 右缘(1227.33)左 **4.0**，距 card 右缘 **20.38** |
| 鱼徽标 | (936, 321.33, 48.67, 15.33) | 我方 (928.67, 320.67, 62.67, 20)：宽 **×0.777**、高 **×0.767** → 是**字号/行高**问题（若线性，等于 12→≈9.3），padding 只解释 14 总宽中的小部分 |

与我方现状的差：卡高 114 vs 155.34（+41.34）、draft 全宽 774 vs 745.33（左 0/右 0 内缩）、send y 479.33 vs 522（−42.67）、send x 1202 vs 1188.67（+13.33）、鱼 62.67×20 vs 48.67×15.33。

## 2. 两次实现尝试（已回滚）

1. **给 hero 工具栏行加 `Margin(top: 42.67)`**（`Composer trailing controls`）：几何上正好把 send 从 479.33 推到 522。结果：`TestComposerCurrentCardGeometry` 失败——在 800×200 的 tester 里卡片被压扁，`primary height=27.33 want 34`，说明卡高不是自适应的，工具栏上抬只会挤压行内控件。
2. **再加 `MinHeight(155.34)`**（原生卡高）：仍然 `primary height=27.33`、`bottom inset=−4.66`，因为父容器给出的高度不足时子行仍被压缩；同时 `TestHeroEmptyComposerMatchesReferenceGeometry`、`TestStatefulEmptyAttachmentRailDoesNotDisplaceDraft` 一并失败。

两次都已 `git checkout`/反向替换回滚；随后 `go test ./internal/ui/composer ./internal/ui/frame ./internal/ui/chat ./internal/ui/sidebar` 全绿，`git status --porcelain internal/ui/composer` 为空。

## 3. 为什么不能"直接改数字"

卡高、工具栏位置、draft 内缩三者互锁：卡高由 hero 外壳（round232 的 foot/stack 居中）与父容器共同决定，`MinHeight` 不足以让 Composer 在受限高度里撑开；而一旦卡高变化，hero 行、card top 409.33、chip 行（round222）都会跟着移动。正确顺序是：
1. 先在 hero 外壳里把卡片高度显式化（给 composer 一个确定的可用高度，而不是依赖内容压缩）；
2. 再把 draft 包进 `Padding(0, 16.39, 0, 12.67)` 的容器（盒子 745.33 宽，而不是靠文字 padding）；
3. 再把工具栏行的上边距设为 52−34−… 的实测差值、右内边距设为 20.38；
4. 最后按字体度量把鱼徽标字号从 `InterfaceSize(12)` 降到能给出 48.67×15.33 的值（约 0.78×），并复核行高；
5. 每步同步更新 `presentation_test.go`、`hero_*_test.go`、`draft_*_test.go` 中钉死的矩形。

## 4. 门禁（回滚后）

- `go test ./internal/ui/composer` ok 7.7 s；`./internal/ui/frame` ok 14.4 s；`./internal/ui/chat` ok 66.9 s；`./internal/ui/sidebar` ok 3.4 s。
- `go vet ./internal/ui/composer ./internal/ui/chat ./internal/ui/frame` → 0。

## 5. 残留

1. 上述 5 步实现未做；本轮只交付了可执行的模型与两次失败记录（避免把压扁的卡片当成"修好了"）。
2. **没有做原生 CDP DOM 探针**：本机原生窗口没有开远程调试端口，本轮改用「安装包内 CSS bundle + 已有 UIA rect」推导；CSS bundle 是压缩单行且 composer 规则疑似在 JS chunk 里，未提取到 `.hero`/`.input-bar` 选择器，因此 padding/line-height 的**CSS 原值**仍未直接读到，只有 rect 推导值。
3. 未做 temp light UIA 验证（没有代码改动可验证）。
4. 未提交、未构建 passN、未改 runtime/vendor 与 third_party。
