# Round132 · 详情条目收尾与遗留卡片（markdown/location/status 记号、job_output 代码卡、terminal_list、lsp）

本轮把 `ToolDetails` 条目词汇补完，并清掉两处遗留卡片：markdown 正文、可点击文件标题、todo 行的状态/变化记号；`job_output` 改用共享 CodeBlock 外观；新增 `terminal_list` 与 `lsp`。目标/定时/作业/智能体团队/检查族的既有输出不变。

## 1. 原生对照

- 条目顺序与各条目：`ToolDetails.tsx:43-107`（标题行 46–75、subtitle/description 76–77、fields 78–87、lines 88、markdown 89、code 90–96、groups 97–104）。
- 可点击位置：`ToolDetails.tsx:58-63`（`item.location !== undefined && onOpenFile !== undefined` 时标题渲染成 `.path` 按钮，点击 `onOpenFile(path, {line})`）；样式 `.path`（`flex:1; text-align:start; cursor:pointer`，hover 下划线）。
- 状态记号：`ToolDetails.tsx:49-57`（change=added → `+`，removed → `−`，completed → 勾图标，in_progress → 播放图标，pending → 10×10 方框）；`:65-73` 尾随状态文本（`previous → status`，`change=updated` 且无 previous 时追加 ` · 变化标签`）；CSS `.status`（14×18，16px/18，flex-start）、`.pending`（1px 边框、radius 2、10×10）、`.statusText`（12px、nowrap、`margin-inline-start:auto`）、`.heading:has(.statusText) .text`（标题 `flex-basis:12em`）、`[data-change=added] .status` success、`[data-change=removed] .status` error 且标题 tertiary + 删除线。语义来源 `todo-diff-model.ts`（status/previousStatus/change）。
- markdown 条目：`.prose`（`margin-top:8px`、13px、随根色继承）。
- `job_output`：`control-details-model.ts:170` 用 CodeBlock 渲染输出（本轮 MyGo 改为同一外观，语言缺失时用共享文案「代码块 / Code block」作语言条）。
- `terminal_list`：`control-details-model.ts:56-74`（`id (name)? [type] running|exited code=X signal=Y pid=N`；running 徽标，exited 徽标按 exit code 0 取 success；字段 Type、PID、Exit code、Signal，signal=`null` 丢弃）。
- `lsp`：`control-details-model.ts:76-98`（需要非空 `file_path` 与数字 `line`/`character`；hover 为 `file:line:character` 标题 + markdown + 位置；`No results.` 为 0 位置空卡；其余行为 `path:row:column` 行，副标题「第 row 行，第 column 列」，URI 不加位置链接、Windows 盘符不算 URI）。
- 安装版词典：terminals.count「%d 个终端 / %d terminals」、locations.count「%d 个位置 / %d locations」、location「第 %s 行，第 %s 列 / Line %s, column %s」、field.exitCode「退出码 / Exit code」、field.signal「信号 / Signal」、todo.diff.addedItem「新增 / Added」、updatedItem「状态变化 / Status changed」、removedItem「移除 / Removed」（`ui-conversation/lib/client.js`）。

## 2. 实现

- 模型（[tool_details_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>)）：`toolDetailItem` 增加 `Markdown *string`、`Location *toolDetailLocation{Path, Line *int}`、`Status *toolDetailStatus{Value, Label}`、`PreviousStatus *string`、`Change *toolDetailChange{Value, Label}`；零值即「未记录」，既有族 golden 不受影响。
- 视图（[tool_details_entries.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_entries.go)、[tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details.go>)）：标题行按 状态记号 → 标题（有位置且 owner 给了打开回调时是按钮，含 `path:line` 标签、hover 下划线）→ 徽标 → 尾随状态文本；带状态文本时标题最小 12em；removed 行标题 tertiary + 删除线；markdown 用 `transcriptMarkdownStyle`（13px）渲染；代码条目一律使用共享 `markdown.ToolCodeBlock`（语言条、换行开关、复制），缺语言时用「代码块 / Code block」；`toolDetailsCard` 增加文件打开回调并透传到条目（调用方 `toolrow.go` 传 `info.openFile`）。
- 卡片（[tool_details_control_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_control_model.go>)）：`terminal_list` 与 `lsp` 两个推导函数接在控制收据派发里，顺序与原生一致（终端列表在 lsp 之前）。
- 文案（[catalog_tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)）：终端/位置计数、位置行列、退出码/信号、三种 todo 变化标签。

## 3. 回归

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 新增 `TestToolDetailsModelTerminalList`（三种状态行、signal=null、exit 0 成功色、空列表、四类非法行）、`TestToolDetailsModelLSP`（五个引用行含 URI/盘符、hover 的位置与 markdown、空结果、省略号行只计有标题行、四类非法参数）、`TestToolDetailsModelVocabularyKindsGolden`（新字段整卡 golden）；`TestToolDetailsGenericEntryModelGolden` 更新为新字段全为 null 的 golden；既有 job/goal/schedule/agent/team/inspection golden 未变。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 新增 `TestToolDetailsViewVocabularyKinds`（`+`/`−`/`→`、previous 状态文本、updated 后缀规则、markdown 正文、点击标题打开 `src/a.ts:12`）与 `TestToolDetailsViewTerminalListAndLSP`（终端行与 lsp 引用行、点击打开 `src/a.ts:2`）；`TestToolDetailsViewJobOutput` 改为断言 CodeBlock 外观（Copy/Code block/Wrap lines），几何测试更新代码卡高度（42px 语言条 + 6/20 内边距 + 两行 19px = 106px）。
- 命令与结果（`. .\scripts\env.ps1`）：`go test ./internal/ui/chat ./internal/ui/i18n ./internal/ui/frame -count=1` → chat 62.083s、i18n 0.650s、frame 12.768s，全部 ok（首次全量 66.085s / 0.596s / 14.217s）；`go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- `todo_write` 详情**族**仍未接线：本轮的 status/previousStatus/change/unchanged 词汇已就绪并有测试，但原生 `todosDetail` 需要从会话历史重建上一版清单（`todo-history.ts`），属于另一轮的工作；在此之前 todo 结果仍走原有计划块。
- 位置点击沿用 MyGo 既有文件打开约定：把 `path:line` 文本交给 `info.openFile`（与「文件卡」按钮一致），原生是 `openFile(path, {line})` 两参形态。
- `lsp` 悬停正文用共享 Markdown 视图渲染（含代码复制），但原生 `MarkdownText` 的脚注等细节由该视图自行处理，未逐项对照。
- 本轮未构建 exe、未跑原生窗口验收；几何与文案按安装版 CSS 与 harness 期望用例，用 `ui.NewTester` 复核。
