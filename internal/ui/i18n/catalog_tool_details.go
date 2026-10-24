package i18n

const (
	KeyToolDetailState        = "chat.tool.detail.state"
	KeyToolDetailGoalEmpty    = "chat.tool.detail.goal.empty"
	KeyToolDetailGoalRounds   = "chat.tool.detail.goal.rounds"
	KeyToolDetailGoalReason   = "chat.tool.detail.goal.reason"
	KeyToolDetailGoalActive   = "chat.tool.detail.goal.active"
	KeyToolDetailGoalPaused   = "chat.tool.detail.goal.paused"
	KeyToolDetailGoalBlocked  = "chat.tool.detail.goal.blocked"
	KeyToolDetailGoalComplete = "chat.tool.detail.goal.complete"
	KeyToolDetailGoalDisarmed = "chat.tool.detail.goal.disarmed"

	KeyToolDetailScheduleCount     = "chat.tool.detail.schedule.count"
	KeyToolDetailScheduleEmpty     = "chat.tool.detail.schedule.empty"
	KeyToolDetailScheduleDeleted   = "chat.tool.detail.schedule.deleted"
	KeyToolDetailScheduleOnce      = "chat.tool.detail.schedule.once"
	KeyToolDetailScheduleEvery     = "chat.tool.detail.schedule.every"
	KeyToolDetailScheduleDaily     = "chat.tool.detail.schedule.daily"
	KeyToolDetailScheduleWeekly    = "chat.tool.detail.schedule.weekly"
	KeyToolDetailScheduleCron      = "chat.tool.detail.schedule.cron"
	KeyToolDetailScheduleWhen      = "chat.tool.detail.schedule.when"
	KeyToolDetailScheduleFrequency = "chat.tool.detail.schedule.frequency"
	KeyToolDetailScheduleScheduled = "chat.tool.detail.schedule.scheduled"
	KeyToolDetailScheduleOverdue   = "chat.tool.detail.schedule.overdue"

	KeyToolDetailDays        = "chat.tool.detail.days"
	KeyToolDetailHours       = "chat.tool.detail.hours"
	KeyToolDetailMinutes     = "chat.tool.detail.minutes"
	KeyToolDetailSeconds     = "chat.tool.detail.seconds"
	KeyToolDetailWeekdayJoin = "chat.tool.detail.weekday.join"

	KeyToolDetailWeekday1 = "chat.tool.detail.weekday.1"
	KeyToolDetailWeekday2 = "chat.tool.detail.weekday.2"
	KeyToolDetailWeekday3 = "chat.tool.detail.weekday.3"
	KeyToolDetailWeekday4 = "chat.tool.detail.weekday.4"
	KeyToolDetailWeekday5 = "chat.tool.detail.weekday.5"
	KeyToolDetailWeekday6 = "chat.tool.detail.weekday.6"
	KeyToolDetailWeekday7 = "chat.tool.detail.weekday.7"

	KeyToolDetailRecordedResult  = "chat.tool.detail.recorded_result"
	KeyToolDetailEmpty           = "chat.tool.detail.empty"
	KeyToolDetailJobsCount       = "chat.tool.detail.jobs.count"
	KeyToolDetailFieldType       = "chat.tool.detail.field.type"
	KeyToolDetailReceiptCancel   = "chat.tool.detail.receipt.cancel"
	KeyToolDetailReceiptFinished = "chat.tool.detail.receipt.finished"
	KeyToolDetailOutputTruncated = "chat.tool.detail.output.truncated"

	// Generic entry vocabulary: the installed `unchanged` container and the
	// shared CodeBlock toolbar.
	KeyToolDetailUnchanged = "chat.tool.detail.unchanged"
	KeyToolDetailCodeLabel = "chat.tool.detail.code.label"
	KeyToolDetailCopy      = "chat.tool.detail.code.copy"
	KeyToolDetailCopied    = "chat.tool.detail.code.copied"
	KeyToolDetailWrap      = "chat.tool.detail.code.wrap"
	KeyToolDetailUnwrap    = "chat.tool.detail.code.unwrap"

	// Agent/team control receipts: counts, receipts, wait outcomes and the
	// projected-record field vocabulary.
	KeyToolDetailAgentsCount      = "chat.tool.detail.agents.count"
	KeyToolDetailTasksCount       = "chat.tool.detail.tasks.count"
	KeyToolDetailTasksNextPage    = "chat.tool.detail.tasks.next_page"
	KeyToolDetailReceiptDelivered = "chat.tool.detail.receipt.delivered"
	KeyToolDetailReceiptInterrupt = "chat.tool.detail.receipt.interrupt"
	KeyToolDetailReceiptStarted   = "chat.tool.detail.receipt.started"
	KeyToolDetailWaitNoProgress   = "chat.tool.detail.wait.no_progress"
	KeyToolDetailWaitTitle        = "chat.tool.detail.wait.title"
	KeyToolDetailWaitTimeout      = "chat.tool.detail.wait.timeout"
	KeyToolDetailWaitChanged      = "chat.tool.detail.wait.changed"
	KeyToolDetailModelsTitle      = "chat.tool.detail.models.title"
	KeyToolDetailAgentReply       = "chat.tool.detail.agent.reply"
	KeyToolDetailNone             = "chat.tool.detail.none"
	KeyToolDetailYes              = "chat.tool.detail.yes"
	KeyToolDetailNo               = "chat.tool.detail.no"
	KeyToolDetailMoreInInspect    = "chat.tool.detail.more_in_inspect"
	KeyToolDetailRowInput         = "chat.tool.detail.row.input"
	KeyToolDetailRowOutput        = "chat.tool.detail.row.output"

	KeyToolDetailFieldID             = "chat.tool.detail.field.id"
	KeyToolDetailFieldRevision       = "chat.tool.detail.field.revision"
	KeyToolDetailFieldPlatform       = "chat.tool.detail.field.platform"
	KeyToolDetailFieldProvider       = "chat.tool.detail.field.provider"
	KeyToolDetailFieldModel          = "chat.tool.detail.field.model"
	KeyToolDetailFieldRole           = "chat.tool.detail.field.role"
	KeyToolDetailFieldContext        = "chat.tool.detail.field.context"
	KeyToolDetailFieldOwner          = "chat.tool.detail.field.owner"
	KeyToolDetailFieldReady          = "chat.tool.detail.field.ready"
	KeyToolDetailFieldDependencies   = "chat.tool.detail.field.dependencies"
	KeyToolDetailFieldWriteScopes    = "chat.tool.detail.field.write_scopes"
	KeyToolDetailFieldWarnings       = "chat.tool.detail.field.warnings"
	KeyToolDetailFieldDiagnostics    = "chat.tool.detail.field.diagnostics"
	KeyToolDetailFieldMethods        = "chat.tool.detail.field.methods"
	KeyToolDetailFieldInputSchema    = "chat.tool.detail.field.input_schema"
	KeyToolDetailFieldOutputSchema   = "chat.tool.detail.field.output_schema"
	KeyToolDetailFieldCurrentPackage = "chat.tool.detail.field.current_package"
	KeyToolDetailFieldNextPackage    = "chat.tool.detail.field.next_package"
	KeyToolDetailFieldLatestRun      = "chat.tool.detail.field.latest_run"
	KeyToolDetailFieldPackages       = "chat.tool.detail.field.packages"
	KeyToolDetailFieldRegistrations  = "chat.tool.detail.field.registrations"
	KeyToolDetailFieldProps          = "chat.tool.detail.field.props"
	KeyToolDetailFieldData           = "chat.tool.detail.field.data"
	KeyToolDetailFieldSource         = "chat.tool.detail.field.source"
	KeyToolDetailFieldContent        = "chat.tool.detail.field.content"
	KeyToolDetailFieldMessage        = "chat.tool.detail.field.message"
	KeyToolDetailFieldMessageID      = "chat.tool.detail.field.message_id"
	KeyToolDetailFieldRoot           = "chat.tool.detail.field.root"
	KeyToolDetailFieldPID            = "chat.tool.detail.field.pid"
	KeyToolDetailFieldTime           = "chat.tool.detail.field.time"
	KeyToolDetailFieldSeq            = "chat.tool.detail.field.seq"
	KeyToolDetailFieldTurn           = "chat.tool.detail.field.turn"
	KeyToolDetailFieldStep           = "chat.tool.detail.field.step"
	KeyToolDetailFieldCallID         = "chat.tool.detail.field.call_id"
	KeyToolDetailFieldAgents         = "chat.tool.detail.field.agents"
	KeyToolDetailFieldResult         = "chat.tool.detail.field.result"
	KeyToolDetailFieldParent         = "chat.tool.detail.field.parent"
	KeyToolDetailFieldDepth          = "chat.tool.detail.field.depth"
	KeyToolDetailFieldAgent          = "chat.tool.detail.field.agent"
	KeyToolDetailFieldJob            = "chat.tool.detail.field.job"
	KeyToolDetailFieldPreviousStatus = "chat.tool.detail.field.previous_status"
	KeyToolDetailFieldTask           = "chat.tool.detail.field.task"

	// Inspection families: Cordis counts, workflow/ralph reports, session-query
	// searches and traces.
	KeyToolDetailProvidersCount        = "chat.tool.detail.providers.count"
	KeyToolDetailPluginsCount          = "chat.tool.detail.plugins.count"
	KeyToolDetailWorkflowScript        = "chat.tool.detail.workflow.script"
	KeyToolDetailRalphComplete         = "chat.tool.detail.ralph.complete"
	KeyToolDetailRalphBlocker          = "chat.tool.detail.ralph.blocker"
	KeyToolDetailRalphLimit            = "chat.tool.detail.ralph.limit"
	KeyToolDetailReportNextSteps       = "chat.tool.detail.report.next_steps"
	KeyToolDetailMatchesCount          = "chat.tool.detail.matches.count"
	KeyToolDetailMatchesCapped         = "chat.tool.detail.matches.capped"
	KeyToolDetailFieldAvailability     = "chat.tool.detail.field.availability"
	KeyToolDetailFieldBestMatch        = "chat.tool.detail.field.best_match"
	KeyToolDetailFieldTarget           = "chat.tool.detail.field.target"
	KeyToolDetailFieldSurface          = "chat.tool.detail.field.surface"
	KeyToolDetailTraceReplacedBy       = "chat.tool.detail.trace.replaced_by"
	KeyToolDetailTraceReplacementChain = "chat.tool.detail.trace.replacement_chain"
	KeyToolDetailTraceReplaces         = "chat.tool.detail.trace.replaces"
	KeyToolDetailTraceSources          = "chat.tool.detail.trace.sources"
	KeyToolDetailTraceDerived          = "chat.tool.detail.trace.derived"
	KeyToolDetailTraceAncestors        = "chat.tool.detail.trace.ancestors"
	KeyToolDetailTraceDescendants      = "chat.tool.detail.trace.descendants"
	KeyToolDetailEventNeighbors        = "chat.tool.detail.event.neighbors"

	// Terminal sessions, language-server locations and todo row markers.
	KeyToolDetailTerminalsCount = "chat.tool.detail.terminals.count"
	KeyToolDetailFieldExitCode  = "chat.tool.detail.field.exit_code"
	KeyToolDetailFieldSignal    = "chat.tool.detail.field.signal"
	KeyToolDetailLocationsCount = "chat.tool.detail.locations.count"
	KeyToolDetailLocation       = "chat.tool.detail.location"
	KeyToolDetailTodoAdded      = "chat.tool.detail.todo.added"
	KeyToolDetailTodoUpdated    = "chat.tool.detail.todo.updated"
	KeyToolDetailTodoRemoved    = "chat.tool.detail.todo.removed"
	// The todo_write diff card: captions, counts and the empty-list copy.
	KeyToolDetailTodoInitial      = "chat.tool.detail.todo.initial"
	KeyToolDetailTodoCompare      = "chat.tool.detail.todo.compare"
	KeyToolDetailTodoUnavailable  = "chat.tool.detail.todo.unavailable"
	KeyToolDetailTodoNoChanges    = "chat.tool.detail.todo.no_changes"
	KeyToolDetailTodoAddedCount   = "chat.tool.detail.todo.added_count"
	KeyToolDetailTodoUpdatedCount = "chat.tool.detail.todo.updated_count"
	KeyToolDetailTodoRemovedCount = "chat.tool.detail.todo.removed_count"
	KeyToolDetailTodoMoved        = "chat.tool.detail.todo.moved"
	KeyToolDetailTodoEmpty        = "chat.tool.detail.todo.empty"

	KeyToolDetailStatusRunning      = "chat.tool.detail.status.running"
	KeyToolDetailStatusIdle         = "chat.tool.detail.status.idle"
	KeyToolDetailStatusReady        = "chat.tool.detail.status.ready"
	KeyToolDetailStatusInactive     = "chat.tool.detail.status.inactive"
	KeyToolDetailStatusProvisioning = "chat.tool.detail.status.provisioning"
	KeyToolDetailStatusFailed       = "chat.tool.detail.status.failed"
	KeyToolDetailStatusCompleted    = "chat.tool.detail.status.completed"
	KeyToolDetailStatusPending      = "chat.tool.detail.status.pending"
	KeyToolDetailStatusProgress     = "chat.tool.detail.status.progress"
	KeyToolDetailStatusDeleted      = "chat.tool.detail.status.deleted"
	KeyToolDetailStatusKilled       = "chat.tool.detail.status.killed"
	KeyToolDetailStatusBlocked      = "chat.tool.detail.status.blocked"
	KeyToolDetailStatusAccepted     = "chat.tool.detail.status.accepted"
	KeyToolDetailStatusQueued       = "chat.tool.detail.status.queued"
	KeyToolDetailStatusExited       = "chat.tool.detail.status.exited"
)

func init() {
	Register(KeyToolDetailState, "状态", "Status")
	Register(KeyToolDetailGoalEmpty, "没有目标", "No goal")
	Register(KeyToolDetailGoalRounds, "执行轮次", "Rounds")
	Register(KeyToolDetailGoalReason, "受阻原因", "Blocker")
	Register(KeyToolDetailGoalActive, "进行中", "Active")
	Register(KeyToolDetailGoalPaused, "已暂停", "Paused")
	Register(KeyToolDetailGoalBlocked, "受阻", "Blocked")
	Register(KeyToolDetailGoalComplete, "已完成", "Completed")
	Register(KeyToolDetailGoalDisarmed, "等待继续", "Awaiting continuation")

	Register(KeyToolDetailScheduleCount, "%d 个定时任务", "%d reminders")
	Register(KeyToolDetailScheduleEmpty, "没有定时任务", "No reminders")
	Register(KeyToolDetailScheduleDeleted, "已删除", "Deleted")
	Register(KeyToolDetailScheduleOnce, "单次", "Once")
	Register(KeyToolDetailScheduleEvery, "每 %s", "Every %s")
	Register(KeyToolDetailScheduleDaily, "每天 %s（%s）", "Every day at %s (%s)")
	Register(KeyToolDetailScheduleWeekly, "每周 %s %s（%s）", "Weekly on %s at %s (%s)")
	Register(KeyToolDetailScheduleCron, "Cron %s（%s）", "Cron %s (%s)")
	Register(KeyToolDetailScheduleWhen, "计划时间", "Scheduled for")
	Register(KeyToolDetailScheduleFrequency, "重复", "Repeat")
	Register(KeyToolDetailScheduleScheduled, "等待触发", "Scheduled")
	Register(KeyToolDetailScheduleOverdue, "已到期，等待会话恢复", "Overdue, awaiting session resume")

	Register(KeyToolDetailDays, "%d 天", "%d d")
	Register(KeyToolDetailHours, "%d 小时", "%d h")
	Register(KeyToolDetailMinutes, "%d 分钟", "%d min")
	Register(KeyToolDetailSeconds, "%d 秒", "%d s")
	Register(KeyToolDetailWeekdayJoin, "、", ", ")

	Register(KeyToolDetailWeekday1, "周一", "Mon")
	Register(KeyToolDetailWeekday2, "周二", "Tue")
	Register(KeyToolDetailWeekday3, "周三", "Wed")
	Register(KeyToolDetailWeekday4, "周四", "Thu")
	Register(KeyToolDetailWeekday5, "周五", "Fri")
	Register(KeyToolDetailWeekday6, "周六", "Sat")
	Register(KeyToolDetailWeekday7, "周日", "Sun")

	Register(KeyToolDetailRecordedResult, "调用结果", "Recorded result")
	Register(KeyToolDetailEmpty, "暂无结果", "No results")
	Register(KeyToolDetailJobsCount, "%d 个后台任务", "%d background jobs")
	Register(KeyToolDetailFieldType, "类型", "Type")
	Register(KeyToolDetailReceiptCancel, "已请求取消", "Cancellation requested")
	Register(KeyToolDetailReceiptFinished, "任务已结束", "Already finished")
	Register(KeyToolDetailOutputTruncated, "输出已截断", "Output truncated")

	// Values recorded from the installed locale dictionaries: `todo.diff.unchanged`
	// and the common `copy`/`copied`/`codeBlock.*` entries.
	Register(KeyToolDetailUnchanged, "%d 项未变化", "%d unchanged")
	Register(KeyToolDetailCodeLabel, "代码块", "Code block")
	Register(KeyToolDetailCopy, "复制", "Copy")
	Register(KeyToolDetailCopied, "复制成功", "Copied")
	Register(KeyToolDetailWrap, "自动换行", "Wrap lines")
	Register(KeyToolDetailUnwrap, "取消自动换行", "Do not wrap lines")

	// Agent/team control receipts and the projected-record field vocabulary,
	// recorded from the installed conversation dictionary.
	Register(KeyToolDetailAgentsCount, "%d 个智能体", "%d agents")
	Register(KeyToolDetailTasksCount, "%d 个团队任务", "%d team tasks")
	Register(KeyToolDetailTasksNextPage, "后面还有任务；下一页位置为 %s", "More tasks available; next cursor is %s")
	Register(KeyToolDetailReceiptDelivered, "消息已送达", "Message delivered")
	Register(KeyToolDetailReceiptInterrupt, "已请求中断", "Interrupt requested")
	Register(KeyToolDetailReceiptStarted, "已启动", "Started")
	Register(KeyToolDetailWaitNoProgress, "没有正在运行的子智能体", "No active subagents")
	Register(KeyToolDetailWaitTitle, "子智能体状态", "Subagent activity")
	Register(KeyToolDetailWaitTimeout, "等待超时", "Wait timed out")
	Register(KeyToolDetailWaitChanged, "检测到变化", "Change detected")
	Register(KeyToolDetailModelsTitle, "可用模型", "Available models")
	Register(KeyToolDetailAgentReply, "智能体回复", "Agent response")
	Register(KeyToolDetailNone, "无", "None")
	Register(KeyToolDetailYes, "是", "Yes")
	Register(KeyToolDetailNo, "否", "No")
	Register(KeyToolDetailMoreInInspect, "另有 %d 项，可在「查看」中读取", "%d more items available in Inspect")
	Register(KeyToolDetailRowInput, "输入", "Input")
	Register(KeyToolDetailRowOutput, "输出", "Output")

	Register(KeyToolDetailFieldID, "ID", "ID")
	Register(KeyToolDetailFieldRevision, "版本", "Revision")
	Register(KeyToolDetailFieldPlatform, "平台", "Platform")
	Register(KeyToolDetailFieldProvider, "提供方", "Provider")
	Register(KeyToolDetailFieldModel, "模型", "Model")
	Register(KeyToolDetailFieldRole, "角色", "Role")
	Register(KeyToolDetailFieldContext, "上下文", "Context")
	Register(KeyToolDetailFieldOwner, "负责人", "Owner")
	Register(KeyToolDetailFieldReady, "可开始", "Ready")
	Register(KeyToolDetailFieldDependencies, "前置任务", "Dependencies")
	Register(KeyToolDetailFieldWriteScopes, "文件范围", "Write scopes")
	Register(KeyToolDetailFieldWarnings, "提示", "Warnings")
	Register(KeyToolDetailFieldDiagnostics, "诊断", "Diagnostics")
	Register(KeyToolDetailFieldMethods, "方法", "Methods")
	Register(KeyToolDetailFieldInputSchema, "输入 Schema", "Input schema")
	Register(KeyToolDetailFieldOutputSchema, "输出 Schema", "Output schema")
	Register(KeyToolDetailFieldCurrentPackage, "当前包", "Current package")
	Register(KeyToolDetailFieldNextPackage, "待运行的包", "Next package")
	Register(KeyToolDetailFieldLatestRun, "最近运行", "Latest run")
	Register(KeyToolDetailFieldPackages, "版本包", "Packages")
	Register(KeyToolDetailFieldRegistrations, "注册项", "Registrations")
	Register(KeyToolDetailFieldProps, "属性", "Props")
	Register(KeyToolDetailFieldData, "数据", "Data")
	Register(KeyToolDetailFieldSource, "来源", "Source")
	Register(KeyToolDetailFieldContent, "内容", "Content")
	Register(KeyToolDetailFieldMessage, "消息", "Message")
	Register(KeyToolDetailFieldMessageID, "消息 ID", "Message ID")
	Register(KeyToolDetailFieldRoot, "根节点", "Root")
	Register(KeyToolDetailFieldPID, "进程 ID", "Process ID")
	Register(KeyToolDetailFieldTime, "时间", "Time")
	Register(KeyToolDetailFieldSeq, "事件序号", "Event sequence")
	Register(KeyToolDetailFieldTurn, "轮次", "Turn")
	Register(KeyToolDetailFieldStep, "步骤", "Step")
	Register(KeyToolDetailFieldCallID, "调用 ID", "Call ID")
	Register(KeyToolDetailFieldAgents, "启动智能体数", "Agents started")
	Register(KeyToolDetailFieldResult, "结果", "Result")
	Register(KeyToolDetailFieldParent, "父级", "Parent")
	Register(KeyToolDetailFieldDepth, "层级", "Depth")
	Register(KeyToolDetailFieldAgent, "智能体 ID", "Agent ID")
	Register(KeyToolDetailFieldJob, "任务 ID", "Job ID")
	Register(KeyToolDetailFieldPreviousStatus, "中断前状态", "Previous status")
	Register(KeyToolDetailFieldTask, "任务内容", "Task")

	Register(KeyToolDetailStatusRunning, "运行中", "Running")
	Register(KeyToolDetailStatusIdle, "空闲", "Idle")
	Register(KeyToolDetailStatusReady, "就绪", "Ready")
	Register(KeyToolDetailStatusInactive, "未运行", "Inactive")
	Register(KeyToolDetailStatusProvisioning, "准备中", "Provisioning")
	Register(KeyToolDetailStatusFailed, "失败", "Failed")
	Register(KeyToolDetailStatusCompleted, "已完成", "Completed")
	Register(KeyToolDetailStatusPending, "待处理", "Pending")
	Register(KeyToolDetailStatusProgress, "进行中", "In progress")
	Register(KeyToolDetailStatusDeleted, "已删除", "Deleted")
	Register(KeyToolDetailStatusKilled, "已取消", "Cancelled")
	Register(KeyToolDetailStatusBlocked, "受阻", "Blocked")
	Register(KeyToolDetailStatusAccepted, "已接收", "Accepted")
	Register(KeyToolDetailStatusQueued, "已入队", "Queued")
	Register(KeyToolDetailStatusExited, "已退出", "Exited")

	// Inspection families, recorded from the installed conversation dictionary.
	Register(KeyToolDetailProvidersCount, "%d 个检查提供方", "%d inspect providers")
	Register(KeyToolDetailPluginsCount, "%d 个动态插件", "%d dynamic plugins")
	Register(KeyToolDetailWorkflowScript, "工作流脚本", "Workflow script")
	Register(KeyToolDetailRalphComplete, "智能体报告完成", "Worker reported completion")
	Register(KeyToolDetailRalphBlocker, "智能体报告受阻", "Worker reported a blocker")
	Register(KeyToolDetailRalphLimit, "已达到轮次上限", "Round limit reached")
	Register(KeyToolDetailReportNextSteps, "待完成事项", "Remaining work")
	Register(KeyToolDetailMatchesCount, "%d 条匹配", "%d matches")
	Register(KeyToolDetailMatchesCapped, "已达到结果上限，可缩小搜索范围", "Result limit reached; narrow the search for more")
	Register(KeyToolDetailFieldAvailability, "可用状态", "Availability")
	Register(KeyToolDetailFieldBestMatch, "最相关事件", "Best match")
	Register(KeyToolDetailFieldTarget, "目标事件", "Target event")
	Register(KeyToolDetailFieldSurface, "记录状态", "Record status")
	Register(KeyToolDetailTraceReplacedBy, "被替换为", "Replaced by")
	Register(KeyToolDetailTraceReplacementChain, "替换链", "Replacement chain")
	Register(KeyToolDetailTraceReplaces, "替换的事件", "Replaced events")
	Register(KeyToolDetailTraceSources, "引用来源", "Source events")
	Register(KeyToolDetailTraceDerived, "派生事件", "Derived events")
	Register(KeyToolDetailTraceAncestors, "祖先会话", "Ancestor sessions")
	Register(KeyToolDetailTraceDescendants, "后代会话", "Descendant sessions")
	Register(KeyToolDetailEventNeighbors, "前后事件", "Surrounding events")
	Register(KeyToolDetailTerminalsCount, "%d 个终端", "%d terminals")
	Register(KeyToolDetailFieldExitCode, "退出码", "Exit code")
	Register(KeyToolDetailFieldSignal, "信号", "Signal")
	Register(KeyToolDetailLocationsCount, "%d 个位置", "%d locations")
	Register(KeyToolDetailLocation, "第 %s 行，第 %s 列", "Line %s, column %s")
	Register(KeyToolDetailTodoAdded, "新增", "Added")
	Register(KeyToolDetailTodoUpdated, "状态变化", "Status changed")
	Register(KeyToolDetailTodoRemoved, "移除", "Removed")
	Register(KeyToolDetailTodoInitial, "首次记录", "Initial list")
	Register(KeyToolDetailTodoCompare, "与上次清单相比", "Changes since the previous list")
	Register(KeyToolDetailTodoUnavailable, "旧清单不可用", "Previous list unavailable")
	Register(KeyToolDetailTodoNoChanges, "清单没有变化", "No changes to the list")
	Register(KeyToolDetailTodoAddedCount, "新增 %d", "%d added")
	Register(KeyToolDetailTodoUpdatedCount, "更新 %d", "%d updated")
	Register(KeyToolDetailTodoRemovedCount, "移除 %d", "%d removed")
	Register(KeyToolDetailTodoMoved, "顺序调整", "Reordered")
	Register(KeyToolDetailTodoEmpty, "任务清单为空", "The to-do list is empty")
}
