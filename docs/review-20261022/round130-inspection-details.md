# Round130 · 检查类详情卡（inspection-details 族）

本轮补上安装版 `inspectionDetails` 一族：`cordis_inspect_list/query/self`、`workflow`、`ralph`、`session_search`、`session_event_search`、`session_event_read`、`session_trace`、`session_event_trace`。这些工具此前全部退回通用 IN/OUT 证据。`lsp` 与 `terminal_list` 仍延后（见边界）。

## 1. 原生对照

模型在 [inspection-details-model.ts](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/inspection-details-model.ts>)：`cordisDetails` 19–34 行、`workflowDetails` 36–72 行、`dateText` 9–17 行、`textFields` 74–90 行、`searchDetails` 92–117 行、`eventReadDetails` 119–134 行、`traceDetails` 136–150 行、派发 160–179 行（含 `hasSpillNotice` 短路）。实体投影与字段名表沿用 round128 的 [detail-model-shared.ts:66-150](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/models/detail-model-shared.ts>)。安装版同一套规则在 <C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\ui-tool\lib\client.js> 内（只读对照）。harness 期望用例 [details-models.client.spec.ts:120-219](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/tests/details-models.client.spec.ts>) 锁定了本轮每张卡片。

| 工具 | 原生规则 | MyGo |
|---|---|---|
| `cordis_inspect_list` | `providers` 必须是数组，按实体投影，汇总「N 个检查提供方」 | 同 |
| `cordis_inspect_query` | 需要字符串 `provider`/`method` 与 `data` 键，汇总 `provider.method` | 同；`data` 按实体投影 |
| `cordis_inspect_self` | 有 `plugins` 数组时按插件列表（汇总「N 个动态插件」）；否则投影整条记录，汇总取参数 `pluginId` → `packageId` → 记录 `mode` | 同 |
| `workflow` | `workflow "名称" completed (N agents).\nReturn value:\n<JSON>`；头部条目为名称 + 完成徽标 + 「启动智能体数」，参数 `code` 进「工作流脚本」组（javascript 代码块），返回值按实体投影，汇总为名称 | 同；代码块用 round127 的 CodeBlock（语言条 + 换行 + 复制） |
| `ralph` | 以 `\nFinal report:\n` 切分；报告必须是含 `summary` 字符串、`evidence`/`nextSteps` 字符串数组、`blocker` 字符串的记录；头部措辞决定徽标（完成/受阻/轮次上限）；`nextSteps` 进「待完成事项」组（行列表），参数 `objective` 进「任务内容」组，`rounds` 与 `blocker` 进字段，`evidence` 进行列表 | 同；行列表/组均用 round127 词汇 |
| `session_search` | 空结果文案或按 `N. ` 分块；`Session <id> — <title>`，`Snippet:` 作描述，`Created` 等已知字段本地化（`Created` 转日期），`Result cap reached.` 换成封顶标题行 | 同 |
| `session_event_search` | 空结果文案；`seq N \| type \| surface \| time` 行，有摘要时标题用摘要、副标题 `type · #N`，`Time` 本地化、`Surface` 进字段 | 同 |
| `session_event_read` | `Session <id> — <title>` + `Target event seq N:` + ```` ```json ```` 块；事件需为含 `type` 与 `data` 记录的记录；`Seq`/`Time` 字段、`data` 按实体投影、尾部相邻事件进「前后事件」组，汇总 `type · #N` | 同 |
| `session_trace` | 首段字段、「Ancestors (nearest first):」「Descendants:」两段；`- none` 显示「无」，否则逐行去掉 `- ` 前缀进行列表；未知段落整卡退回 | 同；行列表保留缩进 |
| `session_event_trace` | 单条记录卡（标题、副标题、字段） | 同；与 `session_trace` 按工具名区分，不靠段落数猜测 |

日期：原生 `Intl.DateTimeFormat(locale, {dateStyle:'medium', timeStyle:'short'})`。用 Node 24 在 `TZ=Asia/Shanghai` 复核了两个随包语言：en = `Oct 24, 2026, 3:59 PM`，zh-CN = `2026年10月24日 15:59`；数字按毫秒时间戳解析，超出 `|ms| < 8.64e15` 视为无效日期并保留原文，无效 ISO 也保留原文（与原生 `String(value)` 一致）。

## 2. 实现

新增 [tool_details_inspection_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_inspection_model.go>)：日期与字段行解析、Cordis/报告/搜索/事件/追踪五个推导组，全部复用 round128 的保序 JSON 解码与 `toolDetailsInspectionItems` 投影。派发在 [tool_details_model.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>) 里紧跟在控制收据之后、目标/定时/作业族之前，对应原生 `controlDetails(...) ?? inspectionDetails(...)` 的位置；`hasSpillNotice` 直接复用 `terminalModelHasSpillNotice`。文案（检查提供方/动态插件计数、工作流脚本、ralph 三种徽标、待完成事项、匹配计数与封顶、可用状态/最相关事件/目标事件/记录状态、追踪与前后事件）登记在 [catalog_tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)，中英取自安装版词典。

## 3. 回归

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 的 `TestToolDetailsModelInspectionCordis`（列表/查询/插件/模式/参数优先级与三类非法输入）、`TestToolDetailsModelInspectionWorkflowAndRalph`（头部字段、脚本组、返回值投影、三种徽标与四类非法报告）、`TestToolDetailsModelInspectionSearches`（会话与事件命中、摘要、封顶、本地化时间、三种非法输入）、`TestToolDetailsModelInspectionReadsAndTraces`（事件卡三段、追踪三段与 `- none`、行列表缩进、无效日期原文、spill 短路、未知段落拒绝）。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 的 `TestToolDetailsViewInspectionReports`（ralph 行列表 + 两个组折叠/展开、workflow 脚本组点开后出现 javascript 与复制动作、无 OUT 回退）。
- 既有门禁 `TestToolDetailsModelCallAndContentGates` 里 `cordis_inspect_self` 原本被当作「未支持名字」，本轮它已成为合法工具，改为精确名字 `cordis_inspect_self `（尾随空格）继续校验名字精确匹配。

命令与结果（`. .\scripts\env.ps1`）：

- `go test ./internal/ui/chat ./internal/ui/i18n ./internal/ui/frame -count=1` → chat 62.733s、i18n 0.311s、frame 15.051s，全部 ok。
- `go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- 显式延后：`lsp`（悬停需要 `markdown` 条目、引用列表需要可点击 `location` 条目，两者 round127 都还没有；本轮不发明新条目类型）与 `terminal_list`（属控制收据族，已在 round128 记录延后）。
- 日期文本只对 en / zh-CN 用安装版格式；MyGo 的语言枚举只有这两种，其它语言记录保留原文（原生会按传入 locale 格式化，这是既有平台差异）。
- 数值到文本沿用 round128 的最短形式；仅极端小数/超安全整数可能与 `JSON.stringify` 不同。
- 本轮实现阶段未构建 exe；收口时统一构建 pass270（见 §5），仍未跑原生窗口验收。卡片文案逐条对照 harness 期望用例，几何复用既有详情卡布局与 `ui.NewTester`。

## 5. 构建

默认与版本化产物均为 **86,862,848 字节**、SHA256 `E3D63B46DEE1F0949943759CA060388A590A1044A3D5F337A342C8788D250D3A`（pass270，覆盖 round127–130），见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/round130-build.json>)。`go build ./internal/... ./cmd/...` 退出 0。
