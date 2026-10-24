# Round127 · ToolDetails 通用条目（lines / groups / unchanged / code）

本轮补齐 `ToolDetails` 的通用条目词汇。round125 完成的是各工具族（goal/schedule/image/web/search/diff/read/terminal/run_code），round126 只补了 `STATUS_KEYS` 徽标词表；条目里的 `lines`（行列表）、`groups`（嵌套条目）、`unchanged`（未变化容器）与带工具条的代码块一直没有实现。本轮只做词汇本身，不改任何既有工具族的输出，也不碰 Inspect 面板。

## 1. 原生对照

渲染器与模型：[ToolDetails.tsx](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/components/ToolDetails.tsx>)（条目顺序 43–107 行：标题行 → subtitle → description → fields → lines → markdown → code → groups；`unchanged` 在列表之后 129–136 行）；样式 [ToolDetails.module.css](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/components/ToolDetails.module.css>)；`unchanged` 的产出见 [todo-diff-model.ts:76](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/todo-diff-model.ts>)，空列表时不产出该容器。安装版把同一份规则压进 <C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\ui-tool\lib\client.js:1507>，逐条与源码一致（只读对照，未改动安装目录）。

| 部件 | 安装版取值 |
|---|---|
| 条目盒 | `padding: 10px 14px`；`.item + .item { border-top: .5px solid border-l2 }`；Inspect 88px 只作用于 `data-inspect:not([data-caption]) > .list > .item`、`.empty`、`.caption` |
| lines | `ul.lines { margin: 6px 0 0; padding-inline-start: 18px; white-space: pre-wrap }`，`li + li { margin-top: 4px }`，圆点标记 |
| groups | `details.group { margin-top: 8px; border-top: .5px solid border-l2 }`；`> summary { gap: 6px; padding: 8px 0 0; color: label-caption }`，右向 chevron 打开时 `rotate(90deg)`；`.group .list { max-height: none }`；`.group .item { padding: 9px 0 2px 20px }` |
| 组内纯代码条目 | `.root .group .item:has(> .code:only-child) { padding: 4px 0 0 20px }` |
| code | `.root .item > .code { margin: 8px 0 0; max-height: 240px; overflow: auto }`；纯代码条目 `padding: 0` 且 code margin 0；CodeBlock 自带语言条、换行开关与复制 |
| unchanged | `details.unchanged { border-top: .5px solid border-l2; color: label-tertiary }`；`> summary { gap: 6px; padding: 9px 14px }`，同样右向 chevron；不吃 Inspect 预留 |
| badge | 5px 圆点 + 5px 间距 + 12px 标签（round126 已实现，本轮沿用） |

文案取自安装版词典（`packages/client/locale/lib/client.js` 与 `ui-conversation/lib/client.js`）：`copy` = 复制 / Copy，`copied` = 复制成功 / Copied，`codeBlock.title` = 代码块 / Code block，`codeBlock.wrap` = 自动换行 / Wrap lines，`codeBlock.unwrap` = 取消自动换行 / Do not wrap lines，`todo.diff.unchanged` = `{count} 项未变化` / `{count} unchanged`。

## 2. 实现

模型（[tool_details_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>)）：`toolDetailItem` 增加 `Lines []string` 与 `Groups []toolDetailGroup`；新增 `toolDetailGroup{Label, Items}` 与 `toolDetailsUnchanged{Label, Items}`；`toolDetailsModel` 增加 `Unchanged *toolDetailsUnchanged`。`toolDetailsUnchangedFor(items)` 生成本地化计数标签，空列表返回 `nil`，因此“未变化”容器不可能以 0 计数出现在界面上。条目字段的 `nil` 与空切片语义保留：`nil Lines` 不画列表，`Fields: []` 与 `Fields: nil` 都不画字段。

视图（新文件 [tool_details_entries.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_entries.go>)）：`toolDetailsLines` 画 18px 起排的圆点行列表与 4px 行距；`toolDetailsGroupView` 画 8px 上边距的细线、6px 间距的 chevron 摘要与 9/0/2/20 的子条目缩进；`toolDetailsUnchangedView` 画 9/14 摘要与普通条目内边距的子项；两者都是 `RoleDisclosure` 的键盘可达摘要，展开状态存在 `ui.Local`，不占 Inspect 预留。`toolDetailsCode` 在条目带语言时改用共享的 `markdown.ToolCodeBlock`（语言条 + 换行开关 + 复制），否则保持 round125 的纯文本可选中区域。`toolDetailsItem` 增加 `nested` 参数以获得组内 4/0/0/20 与 9/0/2/20 两种内边距，并把配色收进 `toolDetailsPalette`。

i18n（[catalog_tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)）：新增 `chat.tool.detail.unchanged`、`chat.tool.detail.code.{label,copy,copied,wrap,unwrap}`，中文与英文按安装版逐字登记。

## 3. 回归

新增测试：

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 的 `TestToolDetailsUnchangedSuppressionAndLabel`（nil/空切片抑制、en/zh 计数、保留条目）、`TestToolDetailsGenericEntryModelGolden`（把 badge/description/lines/code/嵌套 groups/unchanged 整份模型序列化成 golden，锁定 `nil` 与空切片的区别与代码语言）、`TestToolDetailsFamilyModelsStayOnTheirOwnEntries`（job/goal/schedule 成功记录不产生任何通用条目，无效记录仍退回原始 OUT）。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 的 `TestToolDetailsViewGenericEntriesDiscloseAndCopy`（顺序、折叠时隐藏、点开后再收起、复制动作存在、unchanged 展开）、`TestToolDetailsViewGenericEntryGeometry`（19px 内容左沿、37px 行文本、24px 行节奏、组摘要 28px、unchanged 摘要 38px、子行 57px/9+6 偏移、展开增高 37px）、`TestToolDetailsViewGenericEntryGroupCodeOnly`（纯代码子条目 4/0/0/20 与 42px 语言条）、`TestToolDetailsViewUnchangedEmptyAndSuppressed`（空列表 + 抑制、空列表 + unchanged 增高 38.5px）。

命令与结果（`. .\scripts\env.ps1` 后执行）：

- `go test ./internal/ui/chat ./internal/ui/i18n ./internal/ui/frame -count=1` → chat 63.960s、i18n 0.206s、frame 13.050s，全部 ok（前一次全量运行为 chat 81.418s / i18n 0.681s / frame 18.653s）。
- `go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- 本轮未构建任何 exe，也没有跑真实窗口验收：几何断言取自安装版 CSS 在 1x 下的取值，由 `ui.NewTester` 复核，没有原生窗口截图对照。
- `job_output` 的代码块仍是无语言条的纯文本区域。原生对它同样用 CodeBlock 渲染，但任务要求不改既有工具族行为，本轮以“记录里有语言才启用工具条”来区分，属于已登记的差异，留给作业族后续对齐。
- 条目词汇里的 `status`/`previousStatus`/`change`（todo 清单的勾/三角/删除线）与 `markdown`、`location`（可点开文件）不在本轮范围；Inspect 面板仍整块未移植。
- 详情族（作业/团队/会话/Cordis/workflow/ralph/todo）尚未产出 `lines`/`groups`/`unchanged`，本轮只提供它们要用的词汇与测试。
