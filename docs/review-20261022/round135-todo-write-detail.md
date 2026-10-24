# Round135 · todo_write 详情卡（上一版清单重建）

本轮补上 round132 状态/变化记号唯一缺的生产者：`todo_write` 的差异卡。原生把它放在工具行里（不是 `detailsCardModel` 的族派发），需要「这次调用之前的清单」；MyGo 的会话事件窗口里同样有 `todo/write` 事件与调用 id，所以按原生方式重建上一版清单，而不是只做首版卡片。

## 1. 原生对照

- [todo-diff-model.ts](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/todo-diff-model.ts>)：`todosDetail` 校验（`details-card-model.ts:33-47`）→ 上一版条目 → 按标题配对，计算 added / updated（状态变化或位置变化）/ removed，其余进 `unchanged`；汇总为「N added · N updated · N removed」，无变化为「No changes to the list」；标题行 `todo.diff.initial` / `todo.diff.compare` / `todo.diff.unavailable`；空列表在「首版且空」时用 `detail.todo.empty`。
- [todo-history.ts](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/todo-history.ts>)：`todo/write` 事件是清单的权威记录；每个 root `tool/call`（`callId`）或 PTC `tool/ptc-dispatch-start`（`subCallId`）绑定到它之前最近一次 `todo/write` 的状态；基线三种形态——有清单（比较）、`todos: undefined`（首版）、无节点（历史未加载，卡片写「旧清单不可用」）。
- 记号与容器复用 round131/132 的条目词汇（`ToolDetails.tsx:46-75`、`data-change` 样式、`unchanged` 容器）。
- 安装版文案：`todo.diff.initial`「首次记录」、`compare`「与上次清单相比」、`unavailable`「旧清单不可用」、`noChanges`「清单没有变化」、`added`「新增 {count}」、`updated`「更新 {count}」、`removed`「移除 {count}」、`movedItem`「顺序调整」、`detail.todo.empty`「任务清单为空」，以及已登记的 `addedItem`/`updatedItem`/`removedItem` 与 `unchanged`。

## 2. 历史来源与实现

- 来源：`session.ViewState.Messages` 里保留了 `todo/write`、`tool/call`、`tool/ptc-dispatch-start` 原始事件（与计划面板同源）。新增 [tool_details_todo_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_todo_model.go>) 的 `todoBaselinesFor(messages, historyHasMore)`：顺序扫描，维护「最近一次 `todo/write` 的清单」，为每个 `todo_write` 调用记录首个基线；`[chat.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/chat.go>)` 每帧建一次索引，按 `msg.Tool.CallID` 经 `ToolImageGalleryOptions.Todo` 传给 `[tool_lifecycle.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_lifecycle.go>)` 的详情推导。
- 模型：`toolDetailsTodoFor(call, baseline)` 复刻原生差异算法（标题配对、`retainedPositions` 位置比较、previousStatus、removed 按上一版顺序追加、unchanged 收集、汇总拼接）。参数校验同样严格：`todos` 必须是数组、每项 `content` 非空（按 JS trim 集，含 NBSP/BOM）、`status` 限三种、标题重复整卡退回。
- 文案新增在 [catalog_tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)，中文用原生前缀式计数（「新增 1」），英文后缀式（"1 added"）。
- 可逆子集：没有事件窗口的界面（例如只读/独立表面）传入 `nil` 基线，得到原生「首版」卡片（全部记为 added、标题「首次记录」），不需要猜测历史。

## 3. 回归

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 的 `TestToolDetailsModelTodoDiff`（首版、旧清单不可用、混合 added/updated/removed/unchanged、纯顺序调整、无变化、空清单、八种非法参数、重复标题、非 todo_write）、`TestToolDetailsModelTodoHistoryIndex`（root/nested 基线、`hasMore` 截断窗口）、`TestToolDetailsModelTodoCopyChinese`（中文标题/计数/顺序调整/不可用）。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 的 `TestToolDetailsViewTodoDiff`：`+`/`−` 记号、`Pending → Completed` 尾随文本、`1 unchanged` 折叠与展开、无 OUT 回退；删除线用像素对照验证（同一标题在删除行与普通行渲染不同，>20 像素差）。
- 既有族 golden 全部未改；`ToolImageGalleryOptions.Todo` 默认 nil，其他入口行为不变。

命令与结果（`. .\scripts\env.ps1`）：

- `go test ./internal/ui/chat ./internal/ui/i18n -count=1` → chat 81.952s、i18n 0.422s，全部 ok。
- `go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- 历史只在已加载窗口内重建：`HistoryHasMore` 为真且窗口内没有更早的 `todo/write` 时显示「旧清单不可用」，与原生的未加载分支一致；更早分页加载后会随索引自动重算。
- PTC 嵌套调用以 `tool/ptc-dispatch-start` 的 `subCallId` 为键；若该事件不在窗口内则退回首版/不可用判断。
- 原生没有 `todo_read` 工具（任务描述里提到），本轮只实现 `todo_write`；不存在的工具保持原 OUT。
- 差异卡每帧从 `Snapshot.Messages` 重建索引，与既有消息投影同为 O(n)；重复调用时没有额外缓存。
- 本轮未构建 exe、未跑原生窗口验收；文案与算法按原生模型逐条对照，几何沿用既有卡片与 `ui.NewTester`。
