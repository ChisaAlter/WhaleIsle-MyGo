# Round244b · Composer 高度/工具栏实现（步骤 1+3 落地，步骤 2+4 未做）

按 round243 的 5 步顺序实现，本轮完成了 **步骤 1（显式可用高度 → 卡片 155.34）** 和 **步骤 3（工具栏/发送位置）**，测试同步更新、门禁全绿。步骤 2（draft 左右内缩 12.67/16.39）和步骤 4（鱼徽标字号）没有做，原因与实测数据见 §3、§4。

## 1. 结构实现（internal/ui/composer/composer.go）

| 项 | 之前 | 现在 | 原生目标 |
| --- | --- | --- | --- |
| hero 卡片高度 | 134（8+72+12+34+8） | **155.34** | 155.34 ✓ |
| hero 草稿高度 | 72（3 行自适应） | **52.67**（`ta.Height(52.67)`） | 52.67 ✓ |
| 草稿↔工具栏间距 | 12 | **54** | 52（原生草稿 34.67 高、我方 34，差额放进底内边距） |
| 卡片底内边距 | 8 | **6.67** | 8（同上，吸收 0.67 的圆整差） |
| 卡片高度下限 | — | `MinHeight(155.34)`（空草稿时也钉住） | 155.34 ✓ |
| 发送顶（相对卡片顶） | +90 | **+112.67** | +112.67 ✓（=522−409.33） |
| 发送底内边距 | 10 | **8.67** | 8（差 0.67，见上） |
| 发送尺寸 | 34×34 | 34×34（未改） | 34.67×34.67（在 ±1 内） |

headless 实测（tester 800 宽，h=200/300/420 三档一致）：`card={16 40 768 155.34}`、`draft={16 48 768 52.67}`、`send={742 152.67 34 34}` → 发送顶 = 卡片顶 + 112.67，与原生 522（卡片顶 409.33）完全一致；卡片高 155.34 与原生一致。

## 2. 测试同步（钉死新几何）

- `hero_pass2_test.go`：`hero card height` 114 → **155.34**，注释改为 round242 的 8+52.67+54+34+6.67 拆解。
- `presentation_test.go`：hero 的 `bottomInset` 10 → **8.67**（发送底 = 卡片底 − 8.67）。
- `workspace_parity_test.go`：hero 空草稿高度 72 → **52.67**，卡片高度上限 134 → **155.34**。
- docked（非 hero）路径未动，`TestComposerCurrentCardGeometry` 的 docked 分支仍用 8/8 原值。

## 3. 未做：步骤 2（draft 内缩 12.67/16.39）

原生草稿盒是 745.33 宽、距卡片左 12.67、右 16.39；我方仍是 `FillWidth()` 铺满卡片（只有 12 的文字 padding）。要改需把草稿包进带左右 padding 的容器，会牵动 `draft_presentation_test.go` 等一批钉死草稿矩形的用例；本轮时间盒已在步骤 1+3 用尽，故未做——**不是忘了，是刻意留到能一次改完并全部更新测试的下一轮**。

## 4. 未做：步骤 4（鱼徽标 48.67×15.33）

`internal/ui/frame/conversation_phase.go` 的徽标仍是 `InterfaceSize(12)` + `Padding(1,7,0,7)`，实测 62.67×20。目标 48.67×15.33 需要约 0.777 的字号/行高比（≈9.3），属于 frame 侧且会与 slogan/preset 行高互相影响，本轮未改。

## 5. 实测验证（不完整，如实记录）

用当前树构建临时 light 包 `tmp/whaleisle-244b.exe`（已删除）、独立 profile（主题 light、端口 3091）启动，窗口 2160×1380（=1440×860 @150%）；

- **UIA 直接测量失败**：启动后应用弹出「预览版说明」模态（`docs/review-20261022/evidence/round244b/light-hero-244b.png` 可见），模态把 composer 挡在下面，UIA 树里没有 `Composer draft` / `发送消息`，因此没有取得发送/草稿的实机矩形。
- 因此本轮**只有 headless 数字**（与原生完全一致的 +112.67 / 155.34 / 52.67），实机 ±1 验证需在无模态的首启流程下重跑（先写 `welcome-shown` 之外还需关闭 preview notice，或先点「继续」再测）。

## 6. 门禁

- `go test ./internal/ui/composer` ok 9.5 s；`./internal/ui/frame` ok 15.1 s；`./internal/ui/chat` ok 68.7 s；`./internal/ui/sidebar` ok 3.0 s。
- `go vet`（四包）→ 0。
- 临时 exe 已删除、profile 与 socket 已清理、无 whaleisle 进程残留。

## 7. 残留

1. 步骤 2（draft 盒内缩）与步骤 4（鱼徽标字号）未做。
2. 实机 UIA ±1 未验证（被首启模态挡住），下轮用无模态 profile 重测 send/draft/card top。
3. 未提交、未构建 passN、未改 runtime/vendor 与 third_party。
