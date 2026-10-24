# Round184 · F1（标题栏窗口按钮上移）+ F2（侧栏标题行上移）

**方向更正**：任务描述写作「native y=12 → 现在 8」，但 round182 的实测与算术都是反的：原生 `最小化/最大化/关闭` 在 **y=8, h=32 → bottom 40.0**，MyGo 在 **y=12.67 → bottom 44.67**（round182 表 129 行，列序 native | MyGo | 差）。因此 F1 是把 MyGo 从 **12 → 8**，本轮按证据执行。

## F1 标题栏窗口按钮：12 → 8（bottom 44.67 → 40.0）

| 位置 | 之前 | 之后 |
| --- | --- | --- |
| `internal/ui/frame/caption_buttons.go:18` `captionButtonTop` | 12 | **8** |
| `cmd/whaleisle/main.go:2343` `WindowsCaptionButtonOptions{… Top: …}` | 12 | **8** |

32×32 尺寸与 0 间距未动；注释同步更新为「32x32 at Top 8 (native y=8, h=32, bottom 40)」。`caption_buttons_test.go` 用的是 `captionButtonTop` 常量而非硬编码 12，因此 hit/hover/glyph 用例不受影响（frame 包 25.1 s 全绿）。

## F2 侧栏标题行：整体上移 6.67 DIP

`internal/ui/sidebar/workspace_heading.go`：

```go
// 行内 5 个控件（工作区/多选会话/视图选项/添加工作区/搜索会话）一起上移
const workspaceHeadingLift float32 = 6.67

header := ui.Row(c).FillWidth().Height(36)….AlignItems(ui.Center).
	OffsetAfterLayout(func(ui.Rect) ui.Point { return ui.Point{Y: -workspaceHeadingLift} })
```

用 `OffsetAfterLayout` 而不是改上方容器的 padding：只有这一行（含全部五个控件）上移，下面的会话树/滚动区不受影响（`TestWorkspaceSearchIsInlineAndEscapeClears` 的相对位移断言仍通过）。

实测（headless，`ui.NewTester` 1440 宽布局，`chat_chrome_metrics_test.go`）：`视图选项` 由 **Y=256.0 → 249.33**，原生 250 → **误差 0.67 DIP（<1）**。因此该行五个控件全部进入 1 DIP 窗口。

## 测试与门禁

- 更新 `internal/ui/frame/chat_chrome_metrics_test.go`：
  - `W/H` 改为容差比较（`|W-28|,|H-28| ≤ 0.01`）——`OffsetAfterLayout` 会引入 28.000015 这类浮点噪声，原来的精确比较会误报；
  - 新增断言 `|Y-250| ≤ 1`（原生标题行 y=250），把 F2 的修复钉死。
- `go test ./internal/ui/frame` → ok 25.1 s；`go test ./internal/ui/sidebar` → ok 8.4 s；`go test ./cmd/whaleisle -run "Chrome|Caption|Sidebar"` → ok 18.8 s。
- `go vet ./internal/ui/frame ./internal/ui/sidebar` → 0；`go build ./cmd/whaleisle` → 0；另构建临时 exe `tmp/whaleisle-f1f2.exe`（未覆盖任何 passN）。

## 关于 within-1-DIP 计数（原 5/22）

两处结构性偏差已从「+4.67 / +6.67 的整块位移」变为「0 / 0.67」，因此：

- 侧栏标题行 5 个控件（工作区、多选会话、视图选项、添加工作区、搜索会话）应从「全部超出」变为**全部 within 1 DIP**；
- 窗口按钮三连（最小化/最大化/关闭）的 caption bottom 回到原生 40.0，caption-relative 空间不再被整体压缩 4.67。
- **精确的新计数没有重算**：需要重新抓一次 pass286 暗色 1440×860 并用 `scripts/acceptance-structure.py`（带 geom 目录）跑 structure-relative，本轮时间盒内没做这一步，因此不给出未经验证的数字。按结构推断至少 +5（标题行）且 caption 一项归零。

## 残留

1. 未做 pass286 实机截图对照：临时 exe 已构建，但「抓图 → structure-relative → 计数」流程需要一次完整 capture，留待下一轮（命令与上轮一致）。
2. `新建会话` 行的 `dy +54 / dw +83.33`（round182 表 139 行）与本次两处无关，仍是独立的 sidebar 结构差异。
3. 原生窗口只做只读查询；本轮未改 runtime/vendor 或 third_party；临时 exe 在 gitignored `tmp/`。
