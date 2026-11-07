package chat

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// ChatLocalesZH and ChatLocalesEN synchronize @deepseek-ai/dsh-client-ui-chat locales.
var ChatLocalesZH = map[string]string{
	"openui.loading":                        "正在载入交互式回复…",
	"openui.sending":                        "正在发送…",
	"openui.desktopOnly":                    "请在 Whale Isle 桌面端打开此交互式回复。",
	"openui.emptyViewer":                    "Whale UI 渲染器内容为空。",
	"openui.loadFailed":                     "无法载入交互式回复。",
	"openui.stateFailed":                    "无法恢复或保存交互状态。",
	"openui.dismiss":                        "关闭错误提示",
	"openui.invalidAction":                  "Whale UI 操作参数无效。",
	"openui.invalidResource":                "不支持此 Whale UI 资源请求。",
	"openui.invalidBundle":                  "不支持此 Whale UI 资源包。",
	"openui.invalidResult":                  "保存的 Whale UI 工具结果无效。",
	"openui.invalidFilename":                "下载文件名无效。",
	"openui.invalidMediaType":               "下载文件的媒体类型无效。",
	"openui.invalidDownload":                "下载数据地址无效。",
	"openui.panel":                          "Whale UI 面板",
	"openui.panelPublished":                 "已更新 Whale UI 面板",
	"message.stepProcess.readImage":         "正在读取图片",
	"message.stepProcess.write":             "正在写入文件",
	"message.stepProcess.code":              "正在运行代码",
	"message.stepProcess.webSearch":         "正在搜索网页",
	"message.stepProcess.prepare.read":      "准备读取文件",
	"message.stepProcess.prepare.readImage": "准备读取图片",
	"message.stepProcess.prepare.write":     "准备写入文件",
	"message.stepProcess.prepare.search":    "准备搜索代码",
	"message.stepProcess.prepare.edit":      "准备编辑文件",
	"message.stepProcess.prepare.commands":  "准备运行命令",
	"message.stepProcess.prepare.code":      "准备运行代码",
	"message.stepProcess.prepare.webSearch": "准备搜索网页",
	"message.stepProcess.prepare.webFetch":  "准备访问网页",
	"message.stepProcess.prepare.subagents": "准备协调子智能体",
	"message.stepProcess.prepare.plan":      "准备更新计划",
	"message.stepProcess.prepare.questions": "准备提问",
	"message.stepProcess.prepare.tools":     "准备调用工具",
	"message.stepProcess.done.readImage":    "已读取图片",
	"message.stepProcess.done.write":        "已写入文件",
	"message.stepProcess.done.code":         "运行了代码",
	"message.stepProcess.done.webSearch":    "已搜索网页",
	"message.stepProcess.joinTwo":           "{first}并{second}",
	"message.stepProcess.more":              "{title}等",
	"message.turnProcess.projectReplied":    "本轮回复结束",
	"message.turnProcess.projectRepliedTook": "本轮回复结束，用时 ",
	"image.open":                            "查看大图",
	"image.close":                           "关闭图片预览",
	"duration.compactSeconds":               "{seconds}秒",
	"duration.compactMinutes":               "{minutes}分{seconds}秒",
	"stats.counts":                          "{turns} 轮 {steps} 步",
	"stats.dialog.usageTitle":               "Token 用量",
	"stats.dialog.llmTime":                  "模型用时",
	"stats.dialog.toolTime":                 "工具调用用时",
	"stats.dialog.ttft":                     "首 token 平均（TTFT）",
	"stats.dialog.speed":                    "输出速度（TPS）",
	"chat.loadingHistory":                   "载入历史…",
	"chat.loadError":                        "历史加载失败：{message}（{code}）",
	"chat.toBottom":                         "回到底部",
	"chat.deepDiving":                       "深度求索中",
	"chat.deepDivingFor":                    "深度求索中，用时 {duration} ···",
	"chat.turnNavigation.turn":              "第 {turn} 轮",
	"settings.links.newTab":                 "默认浏览器",
	"settings.collapse.nextInput":           "下次有新消息时",
	"settings.transcript.verbose":           "完全展开",
	"fileOpen.unknown":                      "无法打开此文件",
	"message.extraBlock":                    "附加内容块",
	"message.compaction":                    "上下文已压缩",
	"message.compaction.running":            "正在压缩…",
	"message.compaction.expand":             "点击查看压缩摘要",
	"message.compaction.unavailable":        "压缩摘要不可用",
	"message.unknownSurface":                "未知 surface 事件：{type}",
	"message.turnProcess.toolCalls.one":     "{count} 次工具调用",
	"message.turnProcess.toolCalls.other":   "{count} 次工具调用",
	"message.turnProcess.subagents.one":     "{count} 个 subagent",
	"message.turnProcess.subagents.other":   "{count} 个 subagent",
	"message.turnProcess.thoughtForAWhile":  "已思考",
	"message.branchUnavailable":             "仅可从已完成轮次的最后一条消息分支",
	"message.retry.active":                  "正在重试模型请求",
	"message.retry.delay":                   "重试延迟：",
	"message.retry.failure":                 "失败原因：",
	"message.failure.interrupted":           "DSH 进程在本轮完成前退出（例如被强制结束、崩溃或关闭超时），重新启动后本轮已中止。未完成的工具调用结果未知。",
	"message.maxTokens.hint":                "回答被截断，已有输出保留在对话中。发送“继续”可让模型接着输出。",
	"message.turnUsage.model":               "提供方 / 模型",
	"command.failed":                        "指令失败",
	"clock.md":                              "{m}月{d}日",
	"clock.ymd":                             "{y}年{m}月{d}日",
}

var ChatLocalesEN = map[string]string{
	"openui.loading":                        "Loading interactive reply…",
	"openui.sending":                        "Sending…",
	"openui.desktopOnly":                    "Open this interactive reply in Whale Isle desktop.",
	"openui.emptyViewer":                    "The Whale UI renderer document is empty.",
	"openui.loadFailed":                     "Could not load the interactive reply.",
	"openui.stateFailed":                    "Could not restore or save the interaction state.",
	"openui.dismiss":                        "Dismiss error",
	"openui.invalidAction":                  "The Whale UI action arguments are invalid.",
	"openui.invalidResource":                "This Whale UI resource request is unsupported.",
	"openui.invalidBundle":                  "This Whale UI resource bundle is unsupported.",
	"openui.invalidResult":                  "The saved Whale UI tool result is invalid.",
	"openui.invalidFilename":                "The download filename is invalid.",
	"openui.invalidMediaType":               "The download media type is invalid.",
	"openui.invalidDownload":                "The download data URL is invalid.",
	"openui.panel":                          "Whale UI panel",
	"openui.panelPublished":                 "Whale UI panel updated",
	"message.stepProcess.readImage":         "Reading images",
	"message.stepProcess.write":             "Writing files",
	"message.stepProcess.code":              "Running code",
	"message.stepProcess.webSearch":         "Searching the web",
	"message.stepProcess.prepare.read":      "Preparing to read files",
	"message.stepProcess.prepare.readImage": "Preparing to read images",
	"message.stepProcess.prepare.write":     "Preparing to write files",
	"message.stepProcess.prepare.search":    "Preparing to search code",
	"message.stepProcess.prepare.edit":      "Preparing to edit files",
	"message.stepProcess.prepare.commands":  "Preparing to run commands",
	"message.stepProcess.prepare.code":      "Preparing to run code",
	"message.stepProcess.prepare.webSearch": "Preparing to search the web",
	"message.stepProcess.prepare.webFetch":  "Preparing to visit web pages",
	"message.stepProcess.prepare.subagents": "Preparing to coordinate subagents",
	"message.stepProcess.prepare.plan":      "Preparing to update the plan",
	"message.stepProcess.prepare.questions": "Preparing questions",
	"message.stepProcess.prepare.tools":     "Preparing tool calls",
	"message.stepProcess.done.readImage":    "Read images",
	"message.stepProcess.done.write":        "Wrote files",
	"message.stepProcess.done.code":         "Ran code",
	"message.stepProcess.done.webSearch":    "Searched the web",
	"message.stepProcess.joinTwo":           "{first} and {second}",
	"message.stepProcess.more":              "{title}, etc.",
	"message.turnProcess.projectReplied":    "Response finished",
	"message.turnProcess.projectRepliedTook": "Response finished in ",
	"image.open":                            "View full image",
	"image.close":                           "Close image preview",
	"duration.compactSeconds":               "{seconds}s",
	"duration.compactMinutes":               "{minutes}m{seconds}s",
	"stats.counts":                          "{turns} turns {steps} steps",
	"stats.dialog.usageTitle":               "Token usage",
	"stats.dialog.llmTime":                  "LLM time",
	"stats.dialog.toolTime":                 "Tool time",
	"stats.dialog.ttft":                     "Avg time to first token (TTFT)",
	"stats.dialog.speed":                    "Tokens per second (TPS)",
	"chat.loadingHistory":                   "Loading history…",
	"chat.loadError":                        "Failed to load history: {message} ({code})",
	"chat.toBottom":                         "Back to bottom",
	"chat.deepDiving":                       "Deep diving",
	"chat.deepDivingFor":                    "Deep diving for {duration} ···",
	"chat.turnNavigation.turn":              "Turn {turn}",
	"settings.links.newTab":                 "Default Browser",
	"settings.collapse.nextInput":           "On next message",
	"settings.transcript.verbose":           "Verbose",
	"fileOpen.unknown":                      "Couldn’t open this file",
	"message.extraBlock":                    "Extra content block",
	"message.compaction":                    "Context compacted",
	"message.compaction.running":            "Compacting context…",
	"message.compaction.expand":             "View compaction summary",
	"message.compaction.unavailable":        "Compaction summary unavailable",
	"message.unknownSurface":                "Unknown surface event: {type}",
	"message.turnProcess.toolCalls.one":     "{count} tool call",
	"message.turnProcess.toolCalls.other":   "{count} tool calls",
	"message.turnProcess.subagents.one":     "{count} subagent",
	"message.turnProcess.subagents.other":   "{count} subagents",
	"message.turnProcess.thoughtForAWhile":  "Thought for a while",
	"message.branchUnavailable":             "Available only on the last message of a completed turn",
	"message.retry.active":                  "Retrying model request",
	"message.retry.delay":                   "Retry delay: ",
	"message.retry.failure":                 "Failure reason: ",
	"message.failure.interrupted":           "The DSH process exited before this turn finished (for example, it was killed, crashed, or timed out while shutting down), so the turn was closed when DSH restarted. Unfinished tool calls have unknown outcomes.",
	"message.maxTokens.hint":                "The reply was cut off; earlier output is preserved in the conversation. Send \"continue\" to let the model resume.",
	"message.turnUsage.model":               "Provider / model",
	"command.failed":                        "Command failed",
	"clock.md":                              "{m}/{d}",
	"clock.ymd":                             "{y}-{m}-{d}",
}

// ChatPlanLocalesZH and ChatPlanLocalesEN synchronize @deepseek-ai/dsh-client-ui-plan in chat.
var ChatPlanLocalesZH = map[string]string{
	"preview.document":           "计划 · Markdown",
	"preview.open":               "在侧边栏打开计划",
	"preview.full":               "查看全文",
	"preview.openNamed":          "打开计划：{title}",
	"preview.loading":            "正在读取计划…",
	"preview.failed":             "无法读取计划",
	"preview.invalidAddress":     "计划地址无效",
	"preview.historyUnavailable": "无法读取会话历史",
	"preview.unavailable":        "计划预览不可用",
	"chip.exitFailed":            "退出计划模式失败",
}

var ChatPlanLocalesEN = map[string]string{
	"preview.document":           "Plan · Markdown",
	"preview.open":               "Open plan in sidebar",
	"preview.full":               "View full plan",
	"preview.openNamed":          "Open plan: {title}",
	"preview.loading":            "Loading plan…",
	"preview.failed":             "Could not load plan",
	"preview.invalidAddress":     "Invalid plan address",
	"preview.historyUnavailable": "Session history is unavailable",
	"preview.unavailable":        "Plan preview is unavailable",
	"chip.exitFailed":            "Failed to exit plan mode",
}

// SkillLocalesZH and SkillLocalesEN synchronize @deepseek-ai/dsh-client-ui-skill.
var SkillLocalesZH = map[string]string{
	"row.running":   "正在加载 skill",
	"row.preparing": "准备加载技能",
	"row.failed":    "skill 加载失败",
	"row.stopped":   "skill 加载已中止",
}

var SkillLocalesEN = map[string]string{
	"row.running":   "Loading skill",
	"row.preparing": "Preparing to load a skill",
	"row.failed":    "Skill load failed",
	"row.stopped":   "Skill load stopped",
}

// GoalLocalesZH and GoalLocalesEN synchronize @deepseek-ai/dsh-client-ui-goal.
var GoalLocalesZH = map[string]string{
	"commandInput.aria": "指令输入",
}

var GoalLocalesEN = map[string]string{
	"commandInput.aria": "Command input",
}

// CordisLocalesZH and CordisLocalesEN synchronize @deepseek-ai/dsh-client-ui-cordis.
var CordisLocalesZH = map[string]string{
	"row.defineTitle":       "注册 Cordis 插件",
	"a11y.preparing":        "准备调用 Cordis 工具",
	"row.updateTitle":       "更新 Cordis 插件",
	"purpose.missing":       "(未填写用途)",
	"status.idle":           "待激活",
	"status.clientPending":  "Client 待激活",
	"run.removed":           "包已不存在",
	"run.superseded":        "已有更新的运行卡片，请查看下方",
	"panel.hint":            "运行控制在左下角设置上方的 Cordis 面板",
	"panel.approvals.aria":  "Cordis 审批",
	"panel.empty":           "还没有定义任何插件",
	"panel.readFailed":      "读取插件清单失败：{message}",
	"panel.retryAvailable":  "插件清单待重新读取",
	"panel.next":            "待切换：{packageId}",
	"action.approveOnce":    "仅允许此版本",
	"action.approvePlugin":  "允许此插件的后续版本",
	"render.failedAbdicated": "{slot} 渲染失败，已恢复默认界面：",
	"render.failedHeld":     "{slot} 渲染失败：",
	"a11y.defining":         "正在定义插件",
	"a11y.failed":           "定义失败",
	"a11y.stopped":          "定义已中断",
	"body.source":           "插件代码",
}

var CordisLocalesEN = map[string]string{
	"row.defineTitle":       "Register Cordis Plugin",
	"a11y.preparing":        "Preparing a Cordis tool call",
	"row.updateTitle":       "Update Cordis Plugin",
	"purpose.missing":       "(no purpose given)",
	"status.idle":           "Ready",
	"status.clientPending":  "Client ready to activate",
	"run.removed":           "This package no longer exists",
	"run.superseded":        "A newer run card is available below",
	"panel.hint":            "Run controls live in the Cordis panel above Settings",
	"panel.approvals.aria":  "Cordis approvals",
	"panel.empty":           "No plugins defined yet",
	"panel.readFailed":      "Reading the plugin inventory failed: {message}",
	"panel.retryAvailable":  "The plugin inventory needs a fresh read.",
	"panel.next":            "Next: {packageId}",
	"action.approveOnce":    "Allow this version only",
	"action.approvePlugin":  "Allow future versions of this plugin",
	"render.failedAbdicated": "Rendering failed in {slot}; the default UI was restored:",
	"render.failedHeld":     "Rendering failed in {slot}:",
	"a11y.defining":         "Defining the plugin",
	"a11y.failed":           "Definition failed",
	"a11y.stopped":          "Definition interrupted",
	"body.source":           "Plugin source",
}

// TrajectoryLocalesZH and TrajectoryLocalesEN synchronize @deepseek-ai/dsh-client-ui-trajectory.
var TrajectoryLocalesZH = map[string]string{
	"toolbar.aria":                        "轨迹工具栏",
	"toolbar.useActualDuration":           "使用实际时长",
	"toolbar.useEqualWidth":               "使用等宽操作",
	"toolbar.actualTime":                  "实际时间",
	"toolbar.expandTurns":                 "展开所有轮次",
	"toolbar.collapseTurns":               "收起所有轮次",
	"toolbar.expandCalls":                 "展开所有调用",
	"toolbar.collapseCalls":               "收起所有调用",
	"toolbar.search":                      "搜索轨迹",
	"kind.subtool":                        "子工具",
	"kind.sub":                            "子项",
	"turn.label":                          "第 {turn} 轮",
	"section.betweenTurns":                "轮次之间",
	"group.step":                          "第 {step} 步",
	"timing.stepStartUnavailable":         "步骤开始时间不可用",
	"timing.usageUnavailable":             "用量不可用",
	"timing.outputTokensUnavailable":      "输出 token 数不可用",
	"timing.durationTooShort":             "时长过短",
	"timing.ttft":                         "首 token 延迟",
	"timing.throughput":                   "吞吐量",
	"timing.source":                       "计时来源",
	"timing.sessionTimestamps":            "会话时间戳",
	"timing.sessionTimestampsRunning":     "会话时间戳（运行中）",
	"timing.request":                      "请求计时",
	"unit.seconds":                        "{value} 秒",
	"usage.notReported":                   "未报告用量",
	"options.notRecorded":                 "未记录选项",
	"options.json":                        "请求选项 JSON",
	"source.pluginNamed":                  "插件 · {plugin}",
	"source.goalRound":                    "目标 · Round {round}",
	"source.notRecorded":                  "未记录来源",
	"source.messageJson":                  "消息来源 JSON",
	"tab.rawOutput":                       "原始输出",
	"record.schemaUnavailable":            "Schema 不可用",
	"record.resultJson":                   "结果 JSON",
	"record.parametersJson":               "参数 JSON",
	"record.namedParametersJson":          "{name} 参数 JSON",
	"record.payloadJson":                  "参数 JSON",
	"record.outputJson":                   "结果 JSON",
	"code.copyOutput":                     "复制输出",
	"record.systemPromptMissing":          "本次请求没有系统提示词",
	"record.toolsMissing":                 "本次请求没有工具",
	"block.openSummary":                   "打开第 {index} 个块的工具调用概述",
	"block.openSummaryTitle":              "打开工具调用概述",
	"block.label":                         "块 #{index} {type}",
	"history.loadingEarlier":              "正在加载更早的历史…",
	"history.loadingEarlierAria":          "正在加载更早的历史…",
	"history.clickToLoadEarlier":          "点击加载更早的历史",
	"request.label":                       "请求 #{request}",
	"request.labelCompaction":             "请求 #{request} · 压缩",
	"request.compaction":                  "压缩 · {section}",
	"request.retryProgress":               "{retry}/{maximum}",
	"request.collapsedSummary":            "已收起的{kind}概述，{summary}",
	"request.rowAria":                     "{request}{kind}，{content}",
	"request.rowAriaCompaction":           "请求 {request}，压缩",
	"summary.toolCalls.one":               "{count} 个工具调用",
	"summary.toolCalls.other":             "{count} 个工具调用",
	"summary.steps.one":                   "{count} 个步骤",
	"summary.steps.other":                 "{count} 个步骤",
	"details.resize":                      "调整事件详情宽度",
	"details.resizeTitle":                 "拖动调整大小；双击恢复默认值。",
	"details.close":                       "关闭详情",
	"details.purpose":                     "用途",
	"details.subtoolCalls":                "子工具调用",
	"details.lastAttemptError":            "上次尝试错误",
	"details.assistantMessage":            "助手消息",
	"timeline.overviewAria":               "时间线概览；水平拖动可聚焦事件",
	"timeline.started":                    "开始于 {time}",
	"layout.compacting":                   "正在压缩上下文…",
	"layout.compactionFailed":             "上下文压缩失败",
	"layout.compacted":                    "上下文已压缩",
	"layout.imageCount":                   "图片 ×{count}",
	"layout.fileAttachments":              "文件 ×{count}",
	"layout.initialSystemPrompt":          "初始系统提示词",
	"layout.systemPromptUpdated":          "系统提示词已更新",
	"layout.systemPromptAndToolsUpdated":  "系统提示词和工具已更新",
	"layout.compactionInterrupted":        "上下文压缩在完成前被中断。",
}

var TrajectoryLocalesEN = map[string]string{
	"toolbar.aria":                        "Trajectory toolbar",
	"toolbar.useActualDuration":           "Use actual duration",
	"toolbar.useEqualWidth":               "Use equal-width operations",
	"toolbar.actualTime":                  "Actual time",
	"toolbar.expandTurns":                 "Expand turns",
	"toolbar.collapseTurns":               "Collapse turns",
	"toolbar.expandCalls":                 "Expand calls",
	"toolbar.collapseCalls":               "Collapse calls",
	"toolbar.search":                      "Search trajectory",
	"kind.subtool":                        "SUBTOOL",
	"kind.sub":                            "Sub",
	"turn.label":                          "Turn {turn}",
	"section.betweenTurns":                "Between turns",
	"group.step":                          "Step {step}",
	"timing.stepStartUnavailable":         "Step start unavailable",
	"timing.usageUnavailable":             "Usage unavailable",
	"timing.outputTokensUnavailable":      "Output tokens unavailable",
	"timing.durationTooShort":             "Duration too short",
	"timing.ttft":                         "TTFT",
	"timing.throughput":                   "Throughput",
	"timing.source":                       "Timing source",
	"timing.sessionTimestamps":            "Session timestamps",
	"timing.sessionTimestampsRunning":     "Session timestamps (running)",
	"timing.request":                      "Request Timing",
	"unit.seconds":                        "{value} s",
	"usage.notReported":                   "Usage not reported",
	"options.notRecorded":                 "Options not recorded",
	"options.json":                        "Request options JSON",
	"source.pluginNamed":                  "Plugin · {plugin}",
	"source.goalRound":                    "Goal · Round {round}",
	"source.notRecorded":                  "Source not recorded",
	"source.messageJson":                  "Message source JSON",
	"tab.rawOutput":                       "Raw Output",
	"record.schemaUnavailable":            "Schema unavailable",
	"record.resultJson":                   "Result JSON",
	"record.parametersJson":               "parameters JSON",
	"record.namedParametersJson":          "{name} parameters JSON",
	"record.payloadJson":                  "Payload JSON",
	"record.outputJson":                   "Result JSON",
	"code.copyOutput":                     "Copy output",
	"record.systemPromptMissing":          "No system prompt in this request",
	"record.toolsMissing":                 "No tools in this request",
	"block.openSummary":                   "Open Block #{index} tool call summary",
	"block.openSummaryTitle":              "Open tool call summary",
	"block.label":                         "Block #{index} {type}",
	"history.loadingEarlier":              "Loading earlier history…",
	"history.loadingEarlierAria":          "Loading earlier history…",
	"history.clickToLoadEarlier":          "Click to load earlier history",
	"request.label":                       "Request #{request}",
	"request.labelCompaction":             "Request #{request} · Compaction",
	"request.compaction":                  "Compaction · {section}",
	"request.retryProgress":               "{retry} of {maximum}",
	"request.collapsedSummary":            "Collapsed {kind} summary, {summary}",
	"request.rowAria":                     "{request}{kind}, {content}",
	"request.rowAriaCompaction":           "Request {request}, compaction",
	"summary.toolCalls.one":               "{count} tool call",
	"summary.toolCalls.other":             "{count} tool calls",
	"summary.steps.one":                   "{count} step",
	"summary.steps.other":                 "{count} steps",
	"details.resize":                      "Resize event details",
	"details.resizeTitle":                 "Drag to resize. Double-click to reset.",
	"details.close":                       "Close details",
	"details.purpose":                     "Purpose",
	"details.subtoolCalls":                "Subtool calls",
	"details.lastAttemptError":            "Last attempt error",
	"details.assistantMessage":            "Assistant Message",
	"timeline.overviewAria":               "Timeline overview; drag horizontally to focus events",
	"timeline.started":                    "Started {time}",
	"layout.compacting":                   "Compacting context…",
	"layout.compactionFailed":             "Compaction failed",
	"layout.compacted":                    "Context compacted",
	"layout.imageCount":                   "Images ×{count}",
	"layout.fileAttachments":              "Files ×{count}",
	"layout.initialSystemPrompt":          "Initial System Prompt",
	"layout.systemPromptUpdated":          "System Prompt Updated",
	"layout.systemPromptAndToolsUpdated":  "System Prompt and Tools Updated",
	"layout.compactionInterrupted":        "Compaction was interrupted before completion.",
}

// VisualRepliesLocalesZH and VisualRepliesLocalesEN synchronize @deepseek-ai/dsh-client-ui-visual-replies.
var VisualRepliesLocalesZH = map[string]string{
	"settings.saveFailed":      "无法保存设置，请重试。",
	"settings.unavailable":     "当前连接无法更改此设置。",
	"card.expand":              "展开页面",
	"card.collapse":            "收起页面",
	"card.hideSource":          "隐藏源码",
	"card.sourceTitle":         "HTML 源码",
	"card.saveFailed":          "无法保存页面，请重试。",
	"card.loading":             "正在加载页面…",
	"card.loadFailed":          "页面加载失败，请重新加载。",
	"card.loadNotReferenced":   "这个页面不属于当前会话，无法读取。",
	"card.loadTooLarge":        "页面超过 25 MiB 的读取上限，无法加载。",
	"card.loadInvalidUtf8":     "页面不是有效的 UTF-8 HTML，无法读取。",
	"card.loadMissing":         "页面附件已不存在，无法读取。",
	"card.loadCorrupt":         "页面附件已损坏，无法读取。",
	"card.loadInvalidRef":      "页面附件引用无效，无法读取。",
	"card.loadSessionMissing":  "此会话已不存在，无法读取页面。",
	"card.frameTitle":          "视觉回复：{title}",
	"card.desktopOnly":         "请在鲸屿桌面端打开此页面以交互预览。",
}

var VisualRepliesLocalesEN = map[string]string{
	"settings.saveFailed":      "Could not save this setting. Try again.",
	"settings.unavailable":     "This setting cannot be changed through the current connection.",
	"card.expand":              "Expand page",
	"card.collapse":            "Collapse page",
	"card.hideSource":          "Hide source",
	"card.sourceTitle":         "HTML source",
	"card.saveFailed":          "Could not save the page. Try again.",
	"card.loading":             "Loading page…",
	"card.loadFailed":          "The page could not be loaded. Reload to try again.",
	"card.loadNotReferenced":   "This page is not referenced by the current conversation and cannot be read.",
	"card.loadTooLarge":        "The page exceeds the 25 MiB read limit and cannot be loaded.",
	"card.loadInvalidUtf8":     "The page is not valid UTF-8 HTML and cannot be read.",
	"card.loadMissing":         "The page attachment no longer exists and cannot be read.",
	"card.loadCorrupt":         "The page attachment is damaged and cannot be read.",
	"card.loadInvalidRef":      "The page attachment reference is invalid and cannot be read.",
	"card.loadSessionMissing":  "This conversation no longer exists, so the page cannot be read.",
	"card.frameTitle":          "Visual reply: {title}",
	"card.desktopOnly":         "Open this page in the Whale Isle desktop app for an interactive preview.",
}

// ChatAgentTeamLocalesZH and ChatAgentTeamLocalesEN synchronize @deepseek-ai/dsh-experimental-client-ui-agent-team.
var ChatAgentTeamLocalesZH = map[string]string{
	"loading":         "正在加载团队…",
	"project.trigger": "智能体团队，{count} 名成员运行中",
	"project.range":   "{start}–{end} / {total}",
}

var ChatAgentTeamLocalesEN = map[string]string{
	"loading":         "Loading Team…",
	"project.trigger": "Agent Team, {count} members running",
	"project.range":   "{start}–{end} / {total}",
}

// CotTranslationLocalesZH and CotTranslationLocalesEN synchronize @deepseek-ai/dsh-experimental-client-ui-cot-translation.
var CotTranslationLocalesZH = map[string]string{
	"provider":            "翻译服务",
	"google":              "Google",
	"deepseekAccount":     "DeepSeek Flash · 账号（付费）",
	"deepseekOfficial":    "DeepSeek Flash · 官方 API（付费）",
	"providerUnavailable": "此服务暂不可用，请配置 DeepSeek 账号或官方 API 服务，或选择可用的翻译服务",
	"paidNotice":          "每个未缓存片段都会通过所选账号或官方 API 凭据单独发送一次付费的 DeepSeek Flash 请求；重新展开会复用已保存的译文，更改服务、语言、原文或请求设置可能发送新的付费请求",
	"anonymousNotice":     "Bing 和 Google 翻译无需额外登录或 API key，但匿名接口可能拒绝或限制请求",
	"targetLanguage":      "目标语言",
	"targetLanguageHint":  "auto 使用界面语言；也可填写语言代码，例如 zh、en、ja",
	"privacy":             "展开的思考内容会发送给所选翻译服务，可能包含私有代码或对话细节。",
	"translation":         "查看译文",
	"original":            "查看原文",
	"failed":              "翻译暂不可用，已显示原文",
	"translating":         "正在翻译思考内容",
	"invalidProvider":     "请选择翻译服务",
	"invalidLanguage":     "请填写 auto 或语言代码，例如 zh、en、ja",
	"unavailable":         "翻译设置暂不可用",
	"readOnly":            "本部署的设置为只读",
	"saveFailed":          "本部署没有接受这些值，已保留供你修改",
	"unwrap":              "不换行",
	"footnotes":           "脚注",
}

var CotTranslationLocalesEN = map[string]string{
	"provider":            "Translation service",
	"google":              "Google",
	"deepseekAccount":     "DeepSeek Flash — Account (paid)",
	"deepseekOfficial":    "DeepSeek Flash — Official API (paid)",
	"providerUnavailable": "This provider is unavailable. Configure your DeepSeek account or Official API provider, or choose an available service.",
	"paidNotice":          "Each uncached fragment sends a separate paid DeepSeek Flash request through the selected account or Official API credentials. Reopening reasoning reuses saved results; changing the provider, language, source text, or request settings may send a new paid request.",
	"anonymousNotice":     "Bing and Google need no translation login or API key. Their anonymous endpoints may reject or limit requests.",
	"targetLanguage":      "Target language",
	"targetLanguageHint":  "auto follows the UI language; language codes such as zh, en, and ja also work.",
	"privacy":             "Expanded reasoning is sent to the selected service. It may contain private code or conversation details.",
	"translation":         "View translation",
	"original":            "View original",
	"failed":              "Translation failed. Showing the original.",
	"translating":         "Translating reasoning",
	"invalidProvider":     "Choose a translation service.",
	"invalidLanguage":     "Enter auto or a language code, such as zh, en, or ja.",
	"unavailable":         "Translation preferences are unavailable.",
	"readOnly":            "This deployment stores settings read-only.",
	"saveFailed":          "The deployment did not accept these values; they were left for you to correct.",
	"unwrap":              "Unwrap lines",
	"footnotes":           "Footnotes",
}

// ChatLocale returns the localized string for a dsh-client-ui-chat key.
func ChatLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := ChatLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := ChatLocalesZH[key]; ok {
		return v
	}
	return key
}

// ChatPlanLocale returns the localized string for a dsh-client-ui-plan key.
func ChatPlanLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := ChatPlanLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := ChatPlanLocalesZH[key]; ok {
		return v
	}
	return key
}

// SkillLocale returns the localized string for a dsh-client-ui-skill key.
func SkillLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := SkillLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := SkillLocalesZH[key]; ok {
		return v
	}
	return key
}

// SkillRowStateTitle maps a skill tool call state to its localized row title.
func SkillRowStateTitle(state string) string {
	switch state {
	case "preparing":
		return SkillLocale("row.preparing")
	case "running":
		return SkillLocale("row.running")
	case "failed", "error":
		return SkillLocale("row.failed")
	case "stopped", "cancelled", "canceled":
		return SkillLocale("row.stopped")
	default:
		return SkillLocale("row.running")
	}
}

// GoalLocale returns the localized string for a dsh-client-ui-goal key.
func GoalLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := GoalLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := GoalLocalesZH[key]; ok {
		return v
	}
	return key
}

// CordisLocale returns the localized string for a dsh-client-ui-cordis key.
func CordisLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := CordisLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := CordisLocalesZH[key]; ok {
		return v
	}
	return key
}

// TrajectoryLocale returns the localized string for a dsh-client-ui-trajectory key.
func TrajectoryLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := TrajectoryLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := TrajectoryLocalesZH[key]; ok {
		return v
	}
	return key
}

// VisualRepliesLocale returns the localized string for a dsh-client-ui-visual-replies key.
func VisualRepliesLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := VisualRepliesLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := VisualRepliesLocalesZH[key]; ok {
		return v
	}
	return key
}

// VisualReplyLoadError resolves a visual reply error code to its localized message.
func VisualReplyLoadError(code string) string {
	switch code {
	case "not_referenced", "loadNotReferenced":
		return VisualRepliesLocale("card.loadNotReferenced")
	case "too_large", "loadTooLarge":
		return VisualRepliesLocale("card.loadTooLarge")
	case "invalid_utf8", "loadInvalidUtf8":
		return VisualRepliesLocale("card.loadInvalidUtf8")
	case "missing", "loadMissing":
		return VisualRepliesLocale("card.loadMissing")
	case "corrupt", "loadCorrupt":
		return VisualRepliesLocale("card.loadCorrupt")
	case "invalid_ref", "loadInvalidRef":
		return VisualRepliesLocale("card.loadInvalidRef")
	case "session_missing", "loadSessionMissing":
		return VisualRepliesLocale("card.loadSessionMissing")
	default:
		return VisualRepliesLocale("card.loadFailed")
	}
}

// ChatAgentTeamLocale returns the localized string for a dsh-experimental-client-ui-agent-team key.
func ChatAgentTeamLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := ChatAgentTeamLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := ChatAgentTeamLocalesZH[key]; ok {
		return v
	}
	return key
}

// CotTranslationLocale returns the localized string for a dsh-experimental-client-ui-cot-translation key.
func CotTranslationLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := CotTranslationLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := CotTranslationLocalesZH[key]; ok {
		return v
	}
	return key
}

// FormatStepProcessJoinTwo formats message.stepProcess.joinTwo.
func FormatStepProcessJoinTwo(first, second string) string {
	s := ChatLocale("message.stepProcess.joinTwo")
	s = strings.ReplaceAll(s, "{first}", first)
	s = strings.ReplaceAll(s, "{second}", second)
	return s
}

// FormatStepProcessMore formats message.stepProcess.more.
func FormatStepProcessMore(title string) string {
	return strings.ReplaceAll(ChatLocale("message.stepProcess.more"), "{title}", title)
}

// FormatVisualReplyFrameTitle formats card.frameTitle.
func FormatVisualReplyFrameTitle(title string) string {
	return strings.ReplaceAll(VisualRepliesLocale("card.frameTitle"), "{title}", title)
}

// FormatTrajectoryTurnLabel formats turn.label.
func FormatTrajectoryTurnLabel(turn int) string {
	return strings.ReplaceAll(TrajectoryLocale("turn.label"), "{turn}", fmt.Sprintf("%d", turn))
}

// FormatTrajectoryStepLabel formats group.step.
func FormatTrajectoryStepLabel(step int) string {
	return strings.ReplaceAll(TrajectoryLocale("group.step"), "{step}", fmt.Sprintf("%d", step))
}
