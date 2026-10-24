package i18n

// Row-owned copy follows the installed conversation locale. Existing shared
// tool labels keep their older wording for other surfaces.
const (
	KeyToolRowPreparing = "chat.tool.row.preparing"
	KeyToolRowIOGroup   = "chat.tool.row.io_group"
)

func init() {
	Register(KeyToolRowPreparing, "正在准备调用", "Preparing tool call")
	Register(KeyToolRowIOGroup, "输入与输出", "Tool input/output")
	for _, entry := range []struct{ key, zh, en string }{
		{"search", "搜索", "Search"},
		{"read", "读取", "Read"},
		{"bash", "运行命令", "Bash"},
		{"write", "写入", "Write"},
		{"edit", "编辑", "Edit"},
		{"code", "代码", "Code"},
		{"generic", "工具调用", "Tool call"},
		{"inspect", "查询 Cordis 环境", "Query Cordis environment"},
		{"runCordis", "运行 Cordis 插件", "Run Cordis Plugin"},
		{"stopCordis", "停止 Cordis 插件", "Stop Cordis Plugin"},
		{"removeCordis", "移除 Cordis 插件", "Remove Cordis Plugin"},
		{"pwsh", "运行命令", "Pwsh"},
		{"readImage", "读取图片", "Read image"},
		{"todo", "更新任务清单", "Update to-do list"},
		{"ask", "提问", "Ask question"},
		{"createGoal", "创建目标", "Create goal"},
		{"getGoal", "查看目标", "View goal"},
		{"updateGoal", "更新目标", "Update goal"},
		{"createSchedule", "创建定时任务", "Create reminder"},
		{"listSchedules", "查看定时任务", "List reminders"},
		{"deleteSchedule", "删除定时任务", "Delete reminder"},
		{"updateSchedule", "修改定时任务", "Update reminder"},
		{"inspectProviders", "检查提供方", "Inspect providers"},
		{"queryRuntime", "查询运行时", "Query runtime"},
		{"inspectPlugins", "检查动态插件", "Inspect plugins"},
		{"workflow", "运行工作流", "Run workflow"},
		{"ralph", "运行循环工作流", "Run ralph loop"},
		{"readEvent", "读取事件", "Read event"},
		{"searchEvents", "搜索事件", "Search events"},
		{"traceEvent", "追踪事件", "Trace event"},
		{"searchSessions", "搜索会话", "Search sessions"},
		{"traceSession", "追踪会话", "Trace session"},
		{"listModels", "查看可用模型", "List models"},
		{"subagent", "创建子智能体", "Create subagent"},
		{"listAgents", "查看子智能体", "List subagents"},
		{"sendMessage", "发送消息", "Send message"},
		{"interruptAgent", "中断智能体", "Interrupt agent"},
		{"listJobs", "查看后台任务", "List background jobs"},
		{"readJob", "读取任务输出", "Read job output"},
		{"killJob", "取消后台任务", "Cancel background job"},
		{"openTerminal", "创建终端", "Open terminal"},
		{"readTerminal", "读取终端", "Read terminal"},
		{"listTerminals", "查看终端", "List terminals"},
		{"signalTerminal", "发送终端信号", "Signal terminal"},
		{"closeTerminal", "关闭终端", "Close terminal"},
		{"lsp", "查询代码符号", "Query code symbols"},
		{"spawnTeammate", "创建队友", "Create teammate"},
		{"createTeamTask", "创建团队任务", "Create team task"},
		{"getTeamTask", "读取团队任务", "Read team task"},
		{"updateTeamTask", "更新团队任务", "Update team task"},
		{"listTeamTasks", "查看团队任务", "List team tasks"},
		{"waitAgent", "等待子智能体", "Wait for subagent"},
		{"grep", "搜索文件内容", "Grep"},
		{"glob", "查找文件", "Glob"},
		{"webSearch", "网页搜索", "Search"},
		{"webFetch", "网页获取", "Fetch"},
	} {
		Register("chat.tool.row.title."+entry.key, entry.zh, entry.en)
	}
}
