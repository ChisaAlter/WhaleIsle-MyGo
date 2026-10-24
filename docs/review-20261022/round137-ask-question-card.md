# Round137 · ask_user_question 对话记录卡（已回答/未回答/未作答）

本轮补上 `ask_user_question` 在对话历史里的记录卡。`confirm_ask.go` 只是提问队列（当前待回答的编辑器位置），历史行此前只显示通用 IN/OUT。原生把这张卡放在 ask 工具行内：由调用参数的 `questions` 与结果里的 `answers` 按稳定 id 配对，未回答的题显示「未回答」，取消/中断的整组显示判定语。

## 1. 原生对照

- 卡片组件：[AskQuestionCard.tsx](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/components/AskQuestionCard.tsx>)（answered 用 `<dl>` 逐题 `dt` 问题 + `dd` 答案行；一题无答案显示 `skippedLabel`；unanswered 先判定语再列问题）；样式 [AskQuestionCard.module.css](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/components/AskQuestionCard.module.css>)（卡片 16/20 内边距、0.5px border-l1、radius lg、gap 16、max-height 360 滚动；问题 tertiary、答案 primary、14px/24；未回答列表 padding-left 20、gap 8）。
- 数据来源：[toolviews/ask-question-row.tsx](<C:/AI/WhaleIsle-MyGo/reference/electron/vendor/deepseek-harness/packages/client/ui-tool/src/client/tool/toolviews/ask-question-row.tsx>)：`questionEntries` 要求非空数组、每项字符串 `id`/`question`、id 唯一（46–121 行）；`answerEntries` 要求 `{answers:[{id, selected:string[], custom?:string}]}`（46–65 行）；`pairAnswers` 要求数量相等、id 唯一且每题有答案，答案行 = `selected` + 非空 `custom`（123–148 行）；判定语来自错误码 `ASK_CANCELLED` → `ask.cancelledDetail`、`ASK_ABORTED` → `ask.interruptedDetail`（258–271 行），未作答标签 `ask.skipped`。
- 安装版文案：`ask.skipped`「未回答 / Not answered」、`ask.cancelledDetail`「本轮已取消，未提交回答 / This question set was cancelled before answers were submitted.」、`ask.interruptedDetail`「本轮已中断，未提交回答 / This question set was interrupted before answers were submitted.」（`ui-conversation/lib/client.js`）。
- 原生行为经 `ask-question-row.client.spec.tsx:388-491` 固定：三题（含多选、自定义、"跳过"）各列答案行与「未回答」；配对不安全（少答、多答、重复 id）时整张卡退回通用证据；取消/中断没有可查看的回答。

## 2. 实现

- 新增 [ask_question_card.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/ask_question_card.go>)：`askQuestionCard{Kind, Questions, SkippedLabel, Verdict}`；`askQuestionCardFor(call)` 从调用参数取问题（用 round128 保序 JSON 解码，拒绝空数组/缺字段/重复 id），从结果取答案并严格配对；失败结果按 `data.error.code` 生成未回答卡（仅 `ASK_CANCELLED`/`ASK_ABORTED`，其它错误保持通用 OUT）。
- 行接入：[toolcard.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/toolcard.go>) 的 `ToolCardInfo.ask`、[tool_lifecycle.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_lifecycle.go>) 为每个配对完成的行推导卡片、[toolrow.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/toolrow.go>) 的结构化正文最前面绘制卡片并返回 `ownsCopy`；卡片存在时不再画通用 IN/OUT 与链接动作，行仍可展开（`expandable` 计入 `info.ask`）。
- 颜色/字号：卡片背景 `bg-base`、描边 `border-l1`、问题 tertiary、答案/判定语 primary，字号取聊天字号（默认 14），行高字号 +10（默认 24）。文案登记在 [catalog_tool_details.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)。

## 3. 回归

- 模型：[tool_details_model_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 的 `TestToolDetailsModelAskQuestionAnswered`（两题：两个答案行/一题跳过；自定义答案追加、空自定义不追加）、`TestToolDetailsModelAskQuestionFallbacks`（空问题、重复问题 id、缺 question、缺/非数组 answers、少答、未知 id、重复 id、非字符串 selected/custom、非 JSON、非取消类错误、其它工具，共 12 种）、`TestToolDetailsModelAskQuestionUnanswered`（取消/中断判定语与中文文案）。
- 视图：[tool_details_view_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 的 `TestToolDetailsViewAskQuestionAnswered`（问题、多答案行、自定义答案、跳过标签、无 OUT、20+1+20=45px 文本内沿）、`TestToolDetailsViewAskQuestionUnanswered`（判定语 + 问题、无 OUT）、`TestToolDetailsViewAskQuestionMalformedFallback`（非 JSON 结果保持 OUT 证据、无卡片）。
- 既有族 golden 未改；`ask_user_question` 之外的工具不进入该分支。

命令与结果（`. .\scripts\env.ps1`）：

- `go test ./internal/ui/chat ./internal/ui/i18n -count=1` → chat 66.472s、i18n 0.423s，全部 ok。
- `go vet ./internal/ui/chat ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- 未实现原生的 pending/closed 投影分支：这两态需要 `userQuestions` 会话投影（哪些调用仍可回答、哪些已被迟到回复结算）。MyGo 的聊天层目前没有该投影，所以记录里 `{pending:true,callId}` 形状的结果仍显示通用证据，而不是「已继续工作，仍可回答 / 已结束」卡片。
- 行摘要与「回答 / 查看回答」动作没有移植：本卡只做展开后的记录卡；原生 answered 行还会把摘要改成「N/M 已回答」并提供只读面板，这属于面板/队列功能（`confirm_ask.go` 的职责），不在本轮范围。
- 只读（TextOnly）座位不绘制该卡，遵循现有「结构化正文不进入只读证据座位」的约定。
- 本轮未构建 exe、未跑原生窗口验收；文案与校验规则按原组件逐条对照，几何用 `ui.NewTester`。
