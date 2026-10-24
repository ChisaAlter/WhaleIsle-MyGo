# Round128 · 智能体/团队控制收据（control-details 族）

本轮接 round127 的通用条目词汇，补上安装版 `controlDetails` 里智能体与团队协作那一族收据：`list_agents`、`spawn_teammate`、`team_task_create/get/update/list`、`send_message`、`interrupt_agent`、`wait_agent`、`subagent`、`list_subagent_models`。这些卡片此前在 MyGo 里全部退回通用 IN/OUT 证据。`terminal_list`、`lsp` 与 `terminal_*` 收据不在本轮范围，仍然只走原有终端卡片或原始证据。

## 1. 原生对照

模型在 [control-details-model.ts](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/control-details-model.ts>)：`agentList` 22–41 行、`receipt` 15–20 行、派发 110–160 行（`list_agents`、`spawn_teammate`、`team_task_*`、`send_message`、`interrupt_agent`、`wait_agent`、`subagent`、`list_subagent_models`）。实体投影 `inspectionItems` 在 [detail-model-shared.ts:100-150](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/detail-model-shared.ts>)，字段名表在同文件 66–81 行。安装版把这些规则压进 <C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\ui-tool\lib\client.js>（只读对照）。harness 自带期望 [details-models.client.spec.ts:56-117](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/tests/details-models.client.spec.ts>) 锁定了本轮每张卡片的具体文案。

| 工具 | 原生规则 | MyGo 实现 |
|---|---|---|
| `list_agents` | 结果是数组时按实体投影，汇总「N 个智能体」；`(no subagents)` 给 0；否则逐行 `id [state]`，可选 ` parent=… depth=…` 与 ` — title`，任一malformed 行整卡退回 | 三种分支齐全；标题缺失时用 target 作标题且无副标题，有标题时 target 进副标题；`parent`/`depth` 走字段「父级」「层级」 |
| `spawn_teammate` | `json.member` 必须是记录，成员按实体投影，汇总为参数 `name` | 同；真实成员行带 `target/role/diagnostics`，没有标题键，按原生回退标题「结果」 |
| `team_task_create/get/update` | `json` 必须是记录且有字符串 `subject`，整条任务投影，汇总为 subject | 同；`id/revision/status/ownerName/ready/blockedBy/writeScopes/writeScopeWarnings` 按字段名表本地化，数组进嵌套组 |
| `team_task_list` | `tasks` 必须是数组，`nextCursor` 只接受数字，汇总为「N 个团队任务」，有游标时标题行换成「后面还有任务；下一页位置为 N」 | 同；空数组按投影规则得到一条「暂无结果」行 |
| `send_message` | `status` 为 `queued` → 徽标「已入队」warning；`accepted` → 「消息已送达」success；否则文本必须等于 `message delivered to agent <target>`；描述是参数 `message` | 同；target 按 `target/agent_id/sessionId/job_id` 顺序取第一个非空串 |
| `interrupt_agent` | `previousStatus` 字符串时加字段「中断前状态」并本地化其状态词；否则文本必须等于 `interrupt requested for agent <target>` | 同 |
| `wait_agent` | `timedOut` 必须是布尔；`noProgress.message` 时卡片标题为「没有正在运行的子智能体」、描述为该消息；否则收据标题「子智能体状态」，徽标「等待超时」/「检测到变化」neutral | 同 |
| `subagent` | `started background subagent job X` → 「已启动」info + 字段「任务 ID」；`started subagent X` → 「智能体 ID」；其余文本进「智能体回复」卡，任务提示放「任务内容」组 | 同；回复正文本轮放在描述里（原生用 Markdown 条目，见边界） |
| `list_subagent_models` | 每行按第一个 ` — ` 切成模型名与说明，无分隔符时只有说明；汇总取参数 `model`，其次 `provider`，最后「可用模型」 | 同 |

字段名表（`FIELD_KEYS`）连同 `detail.none/yes/no`、`moreInInspect`、`row.input/output` 一起按安装版词典逐字登记，扩展自有的字段名保持原拼写（例如成员行的 `target`）。

## 2. 实现

新增 [tool_details_control_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_control_model.go>)：一个保序 JSON 解码器（`toolDetailsValue`，保留对象键顺序，供实体投影与 JSON 代码块使用）、`toolDetailsInspectionItems`（标题/描述/状态徽标、标量字段、嵌套组、`inputSchema/outputSchema` 代码组、`arguments` 字符串展开、40 行上限、5 层深度后转 JSON 代码）、`toolDetailsReceipt` 与九个族的推导函数。派发在 [tool_details_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>) 中放在目标/定时/作业族之前，与原生 `controlDetails(...) ?? …` 的顺序一致；`job_list/job_output/job_kill` 仍走原有实现，输出不变。

[目录文案](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)：新增计数、收据、等待、模型列表、空值/布尔、Inspector 余量、`row.input/output` 与 40 个字段名，中文与英文取自安装版 `ui-conversation`/`client locale` 词典。

## 3. 回归

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 的 `TestToolDetailsModelAgentList`（行格式、父级/层级、JSON roster、空文本、三条无效行）、`TestToolDetailsModelSpawnAndTeamTasks`（spawn 两种行、非法 member、任务 create/get/update、列表游标与三类非法输入、空任务列表的「暂无结果」行）、`TestToolDetailsModelAgentReceipts`（queued/accepted/纯文本送达、中断两种形态与非法输入、wait 三种结果与非法 timedOut）、`TestToolDetailsModelSubagent`（job/agent 两种启动收据、回复卡与任务组、模型列表两种逗号形态与摘要回退）、`TestToolDetailsModelInspectionProjection`（标量/null/布尔、记录投影、40 行上限与 Inspector 余量、深度转 JSON 代码）。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 的 `TestToolDetailsViewTeamTaskList`（游标标题行、subject/description/徽标、Revision/Owner/Ready 字段、Dependencies/Write scopes 折叠后点开、无 OUT 回退）与 `TestToolDetailsViewAgentRoster`（Build/Review、Running 徽标、Parent/Depth 字段）。

命令与结果（`. .\scripts\env.ps1`）：

- `go test ./internal/ui/chat ./internal/ui/i18n ./internal/ui/frame -count=1` → chat 71.069s、i18n 0.577s、frame 18.386s，全部 ok。
- `go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- 明确延后：`terminal_list` 与 `lsp` 两张列表卡（以及本轮未涉及的 `terminal_open/read/signal/close` 收据）。终端族已有自己的卡片，`lsp` 需要 `location` 可点文件与 Markdown 悬停两种条目，属于后续轮次。
- `subagent` 的回复正文原生走 Markdown 条目；本轮条目词汇还没有 `markdown`，所以正文放在描述里（保留原文与换行），是已登记的可见差异。
- 数值到文本沿用 Go 的最短形式；与 `JSON.stringify` 只在极端小数（如 1e-7）或超出安全整数的数字上可能不同，这些收据目前不产生此类值。
- 本轮没有构建 exe、没有原生窗口验收；卡片文案由 harness 自带期望用例逐条对照，几何沿用 round125/round127 的卡片布局与 `ui.NewTester`。
