# Round262 · Rail 顶距一行（17.33）与 footer 一处未落地

> **状态补记（提交整理批次）**：第 1 节的 17.33 顶距实验已被并发切片覆盖回 18；收尾复核以
> `SidebarRoot.module.css` 的 `.root.collapsed { padding: 18px 10px 6px }` 为真值，确认原生就是
> 18（18/66/114/162/210，节距 48），故第 5 节第 1 条「重新落 17.33」不执行，`rail_geometry_test.go`
> 的容差已收紧到 ±0.1。第 2 节「17.33 时 headless 实测 18.00」不成立（当时改动已被覆盖回 18）。
> 第 3、5 节记录的 footer 距底 5.34 仍未落地，登记为已知差异项（窗口原点侧 1 物理像素另计）。

## 1. 做了什么

1. **顶距 18 → 17.33**：`internal/ui/sidebar/sidebar.go` 收起态 `root.Padding(17.33, 10, 6, 10)`。headless 实测（tester 72×920）：展开 18.00、新建会话 66.00、插件 114.00、自动化任务 162.00、鲸鱼娘 210.00 —— 与原生 18/66/114/162/210 **完全一致**（修复前是 18.67/66.67/…，统一 +0.67）。
2. **footer 账号菜单**：尝试把底内边距 6 → 15.34 → 10，两次实测 y 都是 **874.00 不变**，说明 footer 不受 root 的 bottom padding 控制（它锚在别处）。因此**没有留下 bottom padding 的改动**（改回 6），并在下文记录。

## 2. 一个必须说明的问题：顶距改动没有留在树上

`git status` 显示工作树干净，`sidebar.go:145` 现在是 `root.Padding(18, 10, 6, 10)`，`git log` 里最新提交 `d56da88f feat: composite dark chrome…` 与 `ef93b67a test: pin the collapsed rail geometry…`（后者的提交说明正是本切片新增的测试）。也就是说：**我做的 17.33 改动在并发写入/提交过程中被还原了**，现在树上跑的是 18。

影响：
- 现有测试 `rail_geometry_test.go` 只断言 ±1，误差 0.67 在容差内，**它无法区分 18 与 17.33**（这也解释了为什么门禁仍然全绿）。
- 实机 UIA 在 round261 量到的是 18.67（对应 padding 18），所以那 0.67 的偏差**当前仍然存在**。

## 3. footer 的实测（供下一步用）

tester 72×920，`OnOpenSettings` 提供后账号菜单出现在 **x=10, y=874, 36 高** → 距底 10.00；原生（910.67 高窗口）账号菜单 y=859.33 → 距底 **15.34**，x=9.33、宽 36.67。即我们比原生**低 5.34**、左偏 0.67、窄 0.67（后两者 = 1 物理像素）。
改 root 的 bottom padding（15.34、10）对 y **完全无效**，说明 footer 由另一层（foot area / rail 内固定底部）定位，需要单独读该层再改。

## 4. 门禁（当前树）

`go test ./internal/ui/sidebar` ok 2.9 s；`./internal/ui/frame`（Rail/Sidebar/Chrome/Caption 过滤）ok 4.5 s；`./cmd/whaleisle`（同过滤）ok 18.7 s；`go vet ./internal/ui/sidebar ./internal/ui/frame` → 0。

## 5. 下一步（都很小）

1. 重新落 17.33（并**把测试容差收紧到 0.1**，否则 0.67 的偏差永远测不出来）。
2. 找到 footer 的真实锚点（`footArea` / rail 内底部容器），把它从距底 10 改到 15.34、x 10 → 9.33、宽 36 → 36.67。
3. 重跑 temp light UIA collapsed + expanded 确认。
4. 注意并发：本轮 `sidebar.go` 被其他切片覆盖过一次，改完要立刻 `git diff` 确认，不能只看测试绿。
