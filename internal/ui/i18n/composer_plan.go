package i18n

const (
	KeyComposerPlaceholderPlan       = "composer.placeholder_plan"
	KeyComposerPlaceholderSteerQueue = "composer.placeholder_steer_queue"
)

func init() {
	zhCN[KeyComposerPlaceholderPlan] = "描述你的任务以生成计划"
	en[KeyComposerPlaceholderPlan] = "Describe your task to generate a plan"
	zhCN[KeyComposerPlaceholderSteerQueue] = "Cmd/Ctrl+Enter 插话发送全部排队消息"
	en[KeyComposerPlaceholderSteerQueue] = "Cmd/Ctrl+Enter steers all queued messages"
}
