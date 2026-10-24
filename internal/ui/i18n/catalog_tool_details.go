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
}
