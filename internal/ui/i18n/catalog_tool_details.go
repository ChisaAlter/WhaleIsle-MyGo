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
}
