# Round246 · Hero 栈回到原生座位（卡片长高后的 −20.67 补偿）

## 1. 算术

round244b 把 hero 卡片从 134 长到 **155.34**（+41.34）。hero 栈是「上 grow / 内容 / 下 grow / 38-DIP foot」的居中布局，foot 恒定意味着**抬升量恒为 19**；总内容高 +41.34 后，居中重新分配，栈整体上移 **41.34 / 2 = 20.67**，这正是 round245 实测的 slogan 294（原生 314.67）、preset 352.67（373.33）、card top 388.67（409.33）。

补法（按 Lead 指定）：在栈上方加一个**固定 41.34-DIP 块**，foot 保持 38。新增高度使居中重算，栈下移 41.34 / 2 = **20.67**，精确抵消。

## 2. 实现

`internal/ui/frame/frame.go`（会话区域 children 的最前面）：

```go
if emptyHero || phase == conversationSettling {
    ui.Box(c).Key("conversation-hero-top-block").Height(41.34).Shrink(0)
}
```

只对 hero / settling 生效，active 会话与标题行、chip、caption、导航都不受影响；卡片 155.34、草稿 52.67、send +112.67 全部保持（round244b 的几何未被触碰）。

## 3. 验证（frame harness 1440×920，与 1440×860 差 30 DIP 的固定偏移）

| 元素 | 修复前 | 修复后 | 860 基准（harness −30） | 原生 860 |
| --- | --- | --- | --- | --- |
| slogan y | 324.18 | **344.85** | 314.85 | 314.67 |
| preset 行 y | 382.48 | **403.15** | 373.15 | 373.33 |
| card top | 418.48 | **439.15** | 409.15 | 409.33 |
| card 高 | 155.34 | 155.34 | — | 155.34 |

三者在 860 基准上都与原生差 **≤0.18 DIP**（远好于 ±1）；preset 行→卡片间距仍是 8。

## 4. 测试

新增 `internal/ui/frame/hero_stack_seat_test.go`：在 1440×920 harness 里断言 slogan y 344.67、preset y 403.33、preset 高 28、card top 439.33、card 高 155.34（均 ±1），并断言 preset 行到卡片间距 8；注释写明与 860 实机值的 −30 关系，避免下一个人误改。

## 5. 门禁

- `go test ./internal/ui/frame` ok 17.4 s；`./internal/ui/composer` ok 11.1 s；`./internal/ui/chat` ok 78.4 s。
- `go vet ./internal/ui/frame ./internal/ui/composer ./internal/ui/chat` → 0。
- 本轮临时对照测试已删除，无临时 exe / profile 残留。

## 6. 残留

1. **没有做 temp light binary + 直接 UIA 实机验证**（含 `goPreviewNoticeAck` 关闭预览模态）：时间盒用在结构与测试上，实机 ±1 仍待一次运行；harness 数字与原生差 ≤0.18，但严格说那是 headless。
2. round244b 遗留的 **draft 左右内缩 12.67/16.39** 与 **鱼徽标 48.67×15.33** 仍未做。
3. 未提交、未构建 passN、未改 runtime/vendor 与 third_party。
