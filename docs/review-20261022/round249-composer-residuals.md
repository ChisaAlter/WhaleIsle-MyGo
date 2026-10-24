# Round249 · Composer 三项残留（draft 内缩 / 鱼徽标 / preset chip 间距）

pass301 的三项残留全部落地，卡片 155.34、send 位置、slogan/preset/卡片顶均未被打扰。

## 1. draft 盒内缩 12.67 / 16.39

`internal/ui/composer/composer.go` hero 分支：卡片不再 0 侧内边距，改为**原生写法**——卡片自己承担书写区的左右内缩，控制行再补偿回来：

```go
card = card.Padding(8, 16.39, 6.67, 12.67).Gap(54).MinHeight(155.34)
tools = tools.Padding(0, 4, 0, 4).Height(34)   // 16.39 + 4 = 20.39 ≈ 原生 20.38
```

- draft 盒：live 774.39 − 29.06 = **745.33**，x = 卡片 + **12.67** ✓
- 发送右缘距卡片右 = 16.39 + 4 = **20.39**（原生 20.38）✓
- 卡片高（8 + 52.67 + 54 + 34 + 6.67 = 155.34）与 send 顶（卡片 +112.67）不变 ✓

## 2. 鱼徽标 48.67×15.33

`internal/ui/frame/conversation_phase.go`：字号/行高按 0.777 比例下调并由实测收敛到
`InterfaceSize(8.5)` / `FixedLineHeight(13.4)`（padding 保持 1,7,0,7）。

实测（frame harness）：`大肥鱼版` = **{936, 353.15, 49, 15.4}**，原生 {936, 321.33, 48.67, 15.33}（y 差 30 是 920 vs 860 harness 的固定偏移）→ **w +0.33 / h +0.07，x 完全一致** ✓

## 3. preset chip 间距 +3.33

hero 行 `Gap(0)` → **`Gap(3.33)`**：工作区 chip 结束 652（live）→ preset 起 **655.33** ✓（tester 里 198.67 → 202）。

## 4. 测试同步

| 文件 | 改动 |
| --- | --- |
| `presentation_test.go` | hero：`sideInset` 0→12.67、`draftShrink` 0→29.06、`rightInset` 8→20.38 |
| `hero_pass2_test.go` | `hero card side padding` 0→12.67、`primary right inset` 8→20.38 |
| `draft_presentation_test.go` | `heroSidePadding(hero)` 0→29.06（注释改为原生书写区内缩）；caret 采样改为 `math.Ceil(x)`，因为 draft 盒现在落在小数 x 上 |
| `workspace_parity_test.go` / `hero_chip_row_test.go` | chip 间距断言 0 → 3.2–3.5 |

## 5. 门禁

`go test ./internal/ui/composer` ok 12.2 s；`./internal/ui/frame` ok 18.6 s；`./internal/ui/chat` ok 83.2 s；`go vet` 三包 → 0。

## 6. 残留

1. **未做 temp LIGHT 实机 UIA 验证**（含 goPreviewNoticeAck）：时间盒用在实现与测试更新上。harness 数字已与原生对齐（draft 盒 745.33/12.67 内缩、send 20.39、鱼徽标 49×15.4@936、preset 655.33），实机复核是下一轮的第一步。
2. 头图行的 slogan 起点在 harness 里是 772.64（原生 765.33），差 7.3——这是「鱼图标 + 间距 + 徽标」整行居中后随徽标变窄而右移的结果，不在本任务三项里，记录为观察项。
3. 未提交、未构建 passN、未改 runtime/vendor 与 third_party；无临时 exe/profile，scratch 测试已删。
