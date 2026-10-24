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
}
