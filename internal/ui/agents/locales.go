package agents

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// AgentsPanelLocalesZH and AgentsPanelLocalesEN synchronize @deepseek-ai/dsh-client-ui-agents-panel.
var AgentsPanelLocalesZH = map[string]string{
	"panel.title":            "智能体",
	"guide.title":            "代理与后台任务",
	"guide.description":      "查看当前会话的子代理和后台任务。",
	"empty.title":            "还没有子代理",
	"empty.body":             "当前会话生成子代理后，会显示在这里。",
	"list.aria":              "当前会话的子代理",
	"jobs.title":             "后台任务",
	"jobs.aria":              "当前会话的后台任务",
	"jobs.status.running":    "运行中",
	"jobs.status.stopping":   "正在停止",
	"jobs.status.completed":  "已完成",
	"jobs.status.failed":     "失败",
	"jobs.status.killed":     "已终止",
	"jobs.status.stopped":    "已停止",
	"jobs.status.terminated": "已终止",
	"jobs.stop":              "停止",
	"jobs.stopping":          "正在停止…",
	"type.label":             "代理",
	"empty":                  "当前会话暂无子代理或后台任务",
	"loading":                "正在加载代理与后台任务…",
	"error":                  "无法读取代理或后台任务",
	"retry":                  "重试",
	"section.subagents":      "子代理",
	"section.jobs":           "后台任务",
	"activity.running":       "正在运行",
	"activity.completed":     "已完成",
	"activity.inactive":      "当前未运行",
	"mode.oneShot":           "一次性",
	"mode.continuable":       "可继续",
	"mode.external":          "外部任务",
	"mode.unknown":           "模式未知",
}

var AgentsPanelLocalesEN = map[string]string{
	"panel.title":            "Agents",
	"guide.title":            "Agents and jobs",
	"guide.description":      "Review subagents and background jobs in this session.",
	"empty.title":            "No agents yet",
	"empty.body":             "When this session spawns subagents, they show up here.",
	"list.aria":              "Subagents in this session",
	"jobs.title":             "Background jobs",
	"jobs.aria":              "Background jobs in this session",
	"jobs.status.running":    "running",
	"jobs.status.stopping":   "stopping",
	"jobs.status.completed":  "completed",
	"jobs.status.failed":     "failed",
	"jobs.status.killed":     "killed",
	"jobs.status.stopped":    "stopped",
	"jobs.status.terminated": "terminated",
	"jobs.stop":              "Stop",
	"jobs.stopping":          "Stopping…",
	"type.label":             "Agents",
	"empty":                  "No subagents or background jobs in this session",
	"loading":                "Loading agents and background jobs…",
	"error":                  "Unable to load agents or background jobs",
	"retry":                  "Retry",
	"section.subagents":      "Subagents",
	"section.jobs":           "Background jobs",
	"activity.running":       "running",
	"activity.completed":     "completed",
	"activity.inactive":      "not running",
	"mode.oneShot":           "one-shot",
	"mode.continuable":       "continuable",
	"mode.external":          "External task",
	"mode.unknown":           "unknown mode",
}

// SubagentLocalesZH and SubagentLocalesEN synchronize @deepseek-ai/dsh-client-ui-subagent.
var SubagentLocalesZH = map[string]string{
	"type.label":                "子代理",
	"empty":                     "当前会话暂无子代理",
	"loading":                   "正在加载子代理…",
	"loading.label":             "正在加载子智能体…",
	"error":                     "无法读取子代理",
	"load.error":                "无法加载子智能体",
	"retry":                     "重试",
	"catalog.migrationRequired": "需要迁移",
	"branch.collapse":           "收起 {label} 的下级子智能体",
	"branch.expand":             "展开 {label} 的下级子智能体",
	"section.running":           "运行中",
	"section.completed":         "已完成",
	"activity.running":          "正在运行",
	"activity.completed":        "已完成",
	"activity.inactive":         "当前未运行",
	"mode.oneShot":              "一次性",
	"mode.continuable":          "可继续",
	"mode.external":             "外部任务",
	"mode.unknown":              "模式未知",
	"count.running.one":         "{count} 个子智能体，正在运行",
	"count.running.other":       "{count} 个子智能体，正在运行",
	"count.live.one":            "{n} 个运行中",
	"count.live.other":          "{n} 个运行中",
	"count.total.one":           "{n} 个子代理",
	"count.total.other":         "{n} 个子代理",
	"switcher.aria":             "切换子智能体：{title}",
	"tree.aria":                 "子智能体会话",
	"open.main":                 "在主对话打开",
	"open.sidebar":              "在侧边栏打开",
	"open.sidebar.aria":         "在侧边栏打开 {label}",
	"back.parent":               "返回父会话",
	"back.parentNamed":          "返回 {title}",
	"ancestor.current":          "当前：{title}",
	"ancestor.unnamed":          "未命名会话",
	"sidebar.chat":              "聊天",
	"sidebar.missing":           "未找到该子代理会话",
	"readonly.oneShot.title":    "一次性子智能体记录",
	"readonly.title":            "此子智能体暂时只读",
	"readonly.badge.title":      "只读查看",
	"readonly.oneShot.body":     "一次性任务不支持后续消息，可在这里查看完整执行记录。",
	"readonly.unknown.body":     "读取子会话后才能确定是否可继续。",
	"readonly.body":             "父会话当前不在线，重新打开父会话后即可继续发送消息。",
	"tokens.thousand":           "{value}K",
	"tokens.million":            "{value}M",
	"tokens.total":              "{value} tok",
	"duration.seconds":          "{seconds}秒",
	"duration.minutes":          "{minutes}分{seconds}秒",
	"duration.hours":            "{hours}小时{minutes}分{seconds}秒",
	"duration.days":             "{days}天",
	"duration.daysHours":        "{days}天{hours}小时",
	"duration.months":           "约{months}个月",
	"duration.monthsDays":       "约{months}个月{days}天",
	"duration.years":            "约{years}年",
	"duration.yearsMonths":      "约{years}年{months}个月",
	"duration.exactDays":        "{days}天{hours}小时{minutes}分{seconds}秒",
	"duration.exactTitle":       "精确用时：{duration}",
}

var SubagentLocalesEN = map[string]string{
	"type.label":                "Subagent",
	"empty":                     "No subagents in this session",
	"loading":                   "Loading subagents…",
	"loading.label":             "Loading subagents…",
	"error":                     "Unable to load subagents",
	"load.error":                "Unable to load subagents",
	"retry":                     "Retry",
	"catalog.migrationRequired": "Migration required",
	"branch.collapse":           "Collapse {label} descendants",
	"branch.expand":             "Expand {label} descendants",
	"section.running":           "Running",
	"section.completed":         "Completed",
	"activity.running":          "running",
	"activity.completed":        "completed",
	"activity.inactive":         "not running",
	"mode.oneShot":              "one-shot",
	"mode.continuable":          "continuable",
	"mode.external":             "External task",
	"mode.unknown":              "unknown mode",
	"count.running.one":         "{count} subagent running",
	"count.running.other":       "{count} subagents running",
	"count.live.one":            "{n} running",
	"count.live.other":          "{n} running",
	"count.total.one":           "{n} subagent",
	"count.total.other":         "{n} subagents",
	"switcher.aria":             "Switch subagent: {title}",
	"tree.aria":                 "Subagent sessions",
	"open.main":                 "Open in main view",
	"open.sidebar":              "Open in sidebar",
	"open.sidebar.aria":         "Open {label} in sidebar",
	"back.parent":               "Back to parent",
	"back.parentNamed":          "Back to {title}",
	"ancestor.current":          "Current: {title}",
	"ancestor.unnamed":          "Untitled session",
	"sidebar.chat":              "Chat",
	"sidebar.missing":           "Subagent session not found",
	"readonly.oneShot.title":    "One-shot subagent record",
	"readonly.title":            "This subagent is read-only for now",
	"readonly.badge.title":      "Read-only",
	"readonly.oneShot.body":     "One-shot tasks do not accept follow-ups; review the full execution record here.",
	"readonly.unknown.body":     "Read the child session to determine whether it can be continued.",
	"readonly.body":             "The parent session is offline; reopen it to continue sending messages.",
	"tokens.thousand":           "{value}K",
	"tokens.million":            "{value}M",
	"tokens.total":              "{value} tok",
	"duration.seconds":          "{seconds}s",
	"duration.minutes":          "{minutes}m {seconds}s",
	"duration.hours":            "{hours}h {minutes}m {seconds}s",
	"duration.days":             "{days}d",
	"duration.daysHours":        "{days}d {hours}h",
	"duration.months":           "~{months}mo",
	"duration.monthsDays":       "~{months}mo {days}d",
	"duration.years":            "~{years}y",
	"duration.yearsMonths":      "~{years}y {months}mo",
	"duration.exactDays":        "{days}d {hours}h {minutes}m {seconds}s",
	"duration.exactTitle":       "Exact duration: {duration}",
}

// JobsLocalesZH and JobsLocalesEN synchronize @deepseek-ai/dsh-client-ui-jobs.
var JobsLocalesZH = map[string]string{
	"row.expandAria":      "展开 {label} 的实时输出",
	"row.collapseAria":    "收起 {label} 的实时输出",
	"duration.seconds":    "{seconds}秒",
	"duration.minutes":    "{minutes}分{seconds}秒",
	"output.error":        "实时输出流中断：{error}",
	"terminal.expandAria": "展开被折叠的 {n} 行输出",
	"trigger.label":       "后台任务",
	"panel.aria":          "后台任务",
	"panel.close":         "关闭后台任务面板",
	"section.live":        "运行中",
	"section.done":        "已结束",
	"status.running":      "运行中",
	"status.stopping":     "正在停止",
	"status.completed":    "已完成",
	"status.failed":       "失败",
	"status.killed":       "已停止",
	"duration.title.live": "已运行 {duration}",
	"duration.title.done": "耗时 {duration}",
	"output.empty.live":   "正在等待输出…",
	"output.empty.done":   "无输出",
	"terminal.copy":       "复制输出",
	"terminal.copied":     "已复制",
	"terminal.expand":     "展开其余 {n} 行",
	"terminal.collapse":   "收起",
	"terminal.exitCode":   "退出码 {code}",
	"kill.stop":           "停止任务 {label}",
	"kill.stopping":       "正在停止…",
	"kill.failed":         "停止失败，点击重试",
	"count.live.one":      "{n} 个运行中",
	"count.live.other":    "{n} 个运行中",
	"count.idle.one":      "{n} 个任务",
	"count.idle.other":    "{n} 个任务",
}

var JobsLocalesEN = map[string]string{
	"row.expandAria":      "Show live output of {label}",
	"row.collapseAria":    "Hide live output of {label}",
	"duration.seconds":    "{seconds}s",
	"duration.minutes":    "{minutes}m {seconds}s",
	"output.error":        "live output stream interrupted: {error}",
	"terminal.expandAria": "Expand {n} collapsed output lines",
	"trigger.label":       "Background tasks",
	"panel.aria":          "Background tasks",
	"panel.close":         "Close background tasks",
	"section.live":        "Running",
	"section.done":        "Finished",
	"status.running":      "running",
	"status.stopping":     "stopping",
	"status.completed":    "completed",
	"status.failed":       "failed",
	"status.killed":       "stopped",
	"duration.title.live": "Running for {duration}",
	"duration.title.done": "Took {duration}",
	"output.empty.live":   "Waiting for output…",
	"output.empty.done":   "No output",
	"terminal.copy":       "Copy output",
	"terminal.copied":     "Copied",
	"terminal.expand":     "Show {n} more lines",
	"terminal.collapse":   "Show less",
	"terminal.exitCode":   "exit {code}",
	"kill.stop":           "Stop task {label}",
	"kill.stopping":       "Stopping…",
	"kill.failed":         "Failed to stop; click to retry",
	"count.live.one":      "{n} running",
	"count.live.other":    "{n} running",
	"count.idle.one":      "{n} task",
	"count.idle.other":    "{n} tasks",
}

// AgentTeamLocalesZH and AgentTeamLocalesEN synchronize @deepseek-ai/dsh-experimental-client-ui-agent-team.
var AgentTeamLocalesZH = map[string]string{
	"action":                           "团队",
	"title":                            "智能体团队",
	"members":                          "成员",
	"tasks":                            "共享任务",
	"emptyTasks":                       "暂无共享任务，可以通过对话创建",
	"loading":                          "正在加载团队…",
	"unavailable":                      "Team 暂不可用",
	"retry":                            "重试",
	"dismissError":                     "关闭提示",
	"failure":                          "团队持久记录无效：{message}",
	"currentSession":                   "当前会话",
	"openMember":                       "打开成员会话",
	"model":                            "模型",
	"owner":                            "Owner",
	"unowned":                          "未分配",
	"blockedBy":                        "依赖",
	"writeScopes":                      "写入范围",
	"ready":                            "可开始",
	"blocked":                          "被依赖阻塞",
	"expand":                           "展开",
	"collapse":                         "收起",
	"status.provisioning":              "准备中",
	"status.running":                   "运行中",
	"status.inactive":                  "待命",
	"status.stopped":                   "已结束",
	"status.failed":                    "失败",
	"status.turnRunning":               "回合运行中",
	"task.pending":                     "待处理",
	"task.in_progress":                 "进行中",
	"task.completed":                   "已完成",
	"project.coordinator":              "协调者",
	"project.work":                     "工作",
	"project.assignee":                 "负责人",
	"project.openHistory":              "查看工作过程",
	"project.keepObserving":            "持续观察",
	"project.returnToPopover":          "回到浮层",
	"project.close":                    "关闭",
	"project.reopenFromProject":        "请从来源项目的团队按钮重新打开。",
	"project.unavailableInProject":     "团队暂不可用，请从来源项目重试。",
	"project.sidebarUnavailable":       "暂时无法打开团队侧栏，请重试。",
	"project.scopeLabel":               "团队记录范围",
	"project.scopeCurrent":             "当前与近期",
	"project.scopeAll":                 "全部记录",
	"project.recentNote":               "包含近期登记的历史工作。",
	"project.searchLabel":              "查找成员或工作",
	"project.searchPlaceholder":        "输入名称或工作内容",
	"project.emptyCurrentWork":         "暂无相关工作。以往工作可在全部记录中查看。",
	"project.emptyAllWork":             "尚未安排后台工作，在主对话中提出需求即可。",
	"project.emptyFilteredMembers":     "没有匹配成员，可尝试其他关键词。",
	"project.emptyFilteredWork":        "没有匹配工作，可尝试其他关键词。",
	"project.memberPagination":         "成员分页",
	"project.workPagination":           "工作分页",
	"project.previousPage":             "上一页",
	"project.nextPage":                 "下一页",
	"project.range":                    "{start}–{end} / {total}",
	"project.unknownPrerequisite":      "前置工作不可用",
	"project.idle":                     "空闲",
	"project.backgroundCommandRunning": "后台命令运行中",
	"project.backgroundCommandStale":   "后台命令上次仍在运行",
	"project.activityStale":            "团队活动暂未确认；后台命令记录为上次确认状态。",
	"project.trigger":                  "智能体团队，{count} 名成员运行中",
	"project.triggerUnknown":           "智能体团队，运行状态待确认",
	"project.triggerHint":              "{count} 名成员正在运行，含后台命令。点击查看。",
	"project.triggerUnknownHint":       "团队活动暂未确认。点击查看保留的记录。",
}

var AgentTeamLocalesEN = map[string]string{
	"action":                           "Team",
	"title":                            "Agent Team",
	"members":                          "Members",
	"tasks":                            "Shared tasks",
	"emptyTasks":                       "No shared tasks yet. Create them through the conversation.",
	"loading":                          "Loading Team…",
	"unavailable":                      "Team is unavailable",
	"retry":                            "Retry",
	"dismissError":                     "Dismiss notice",
	"failure":                          "Invalid team persistence record: {message}",
	"failure.persisted":                "Invalid persisted Team record: {message}",
	"currentSession":                   "Current chat",
	"openMember":                       "Open member conversation",
	"model":                            "Model",
	"owner":                            "Owner",
	"unowned":                          "Unowned",
	"blockedBy":                        "Blocked by",
	"writeScopes":                      "Write scopes",
	"ready":                            "Ready",
	"blocked":                          "Blocked by dependencies",
	"expand":                           "Show more",
	"collapse":                         "Show less",
	"status.provisioning":              "Provisioning",
	"status.running":                   "Running",
	"status.inactive":                  "Standing by",
	"status.stopped":                   "Ended",
	"status.failed":                    "Failed",
	"status.turnRunning":               "Turn running",
	"task.pending":                     "Pending",
	"task.in_progress":                 "In progress",
	"task.completed":                   "Completed",
	"project.coordinator":              "Coordinator",
	"project.work":                     "Work",
	"project.assignee":                 "Assignee",
	"project.openHistory":              "View work history",
	"project.keepObserving":            "Keep observing",
	"project.returnToPopover":          "Return to popover",
	"project.close":                    "Close",
	"project.reopenFromProject":        "Reopen this Team from its source Project.",
	"project.unavailableInProject":     "Team is unavailable. Retry from its source Project.",
	"project.sidebarUnavailable":       "The Team Sidebar cannot open right now. Try again.",
	"project.scopeLabel":               "Team record scope",
	"project.scopeCurrent":             "Current and recent",
	"project.scopeAll":                 "All records",
	"project.recentNote":               "Includes recently recorded historical work.",
	"project.searchLabel":              "Find a member or work",
	"project.searchPlaceholder":        "Search names or work descriptions",
	"project.emptyCurrentWork":         "No related work. Earlier work is available in All records.",
	"project.emptyAllWork":             "No background work yet. Describe what you need in the main conversation.",
	"project.emptyFilteredMembers":     "No matching members. Try another keyword.",
	"project.emptyFilteredWork":        "No matching work. Try another keyword.",
	"project.memberPagination":         "Member pages",
	"project.workPagination":           "Work pages",
	"project.previousPage":             "Previous",
	"project.nextPage":                 "Next",
	"project.range":                    "{start}–{end} / {total}",
	"project.unknownPrerequisite":      "Prerequisite unavailable",
	"project.idle":                     "Idle",
	"project.backgroundCommandRunning": "Background command running",
	"project.backgroundCommandStale":   "Background command last seen running",
	"project.activityStale":            "Team activity is unconfirmed; background command records show their last confirmed state.",
	"project.trigger":                  "Agent Team, {count} members running",
	"project.triggerUnknown":           "Agent Team, activity unconfirmed",
	"project.triggerHint":              "{count} members running, including background commands. Click to inspect.",
	"project.triggerUnknownHint":       "Team activity is unconfirmed. Click to inspect retained records.",
}

// AgentsPanelLocale returns the localized string for a dsh-client-ui-agents-panel key.
func AgentsPanelLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := AgentsPanelLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := AgentsPanelLocalesZH[key]; ok {
		return v
	}
	return key
}

// SubagentLocale returns the localized string for a dsh-client-ui-subagent key.
func SubagentLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := SubagentLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := SubagentLocalesZH[key]; ok {
		return v
	}
	return key
}

// JobsLocale returns the localized string for a dsh-client-ui-jobs key.
func JobsLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := JobsLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := JobsLocalesZH[key]; ok {
		return v
	}
	return key
}

// AgentTeamLocale returns the localized string for a dsh-experimental-client-ui-agent-team key.
func AgentTeamLocale(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := AgentTeamLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := AgentTeamLocalesZH[key]; ok {
		return v
	}
	return key
}

func formatSubagentTemplate(key string, kv ...string) string {
	text := SubagentLocale(key)
	for i := 0; i+1 < len(kv); i += 2 {
		text = strings.ReplaceAll(text, "{"+kv[i]+"}", kv[i+1])
	}
	return text
}

// FormatSubagentBranchToggle formats branch.collapse / branch.expand.
func FormatSubagentBranchToggle(label string, collapsed bool) string {
	key := "branch.collapse"
	if collapsed {
		key = "branch.expand"
	}
	return strings.ReplaceAll(SubagentLocale(key), "{label}", label)
}

// FormatSubagentRunningCount formats count.running.one / count.running.other.
func FormatSubagentRunningCount(count int) string {
	key := "count.running.other"
	if count == 1 {
		key = "count.running.one"
	}
	return strings.ReplaceAll(SubagentLocale(key), "{count}", fmt.Sprintf("%d", count))
}

// FormatSubagentLiveCount formats count.live.one / count.live.other.
func FormatSubagentLiveCount(n int) string {
	key := "count.live.other"
	if n == 1 {
		key = "count.live.one"
	}
	return strings.ReplaceAll(SubagentLocale(key), "{n}", strconv.Itoa(n))
}

// FormatSubagentTotalCount formats count.total.one / count.total.other.
func FormatSubagentTotalCount(n int) string {
	key := "count.total.other"
	if n == 1 {
		key = "count.total.one"
	}
	return strings.ReplaceAll(SubagentLocale(key), "{n}", strconv.Itoa(n))
}

// FormatSubagentSwitcherAria formats switcher.aria.
func FormatSubagentSwitcherAria(title string) string {
	return strings.ReplaceAll(SubagentLocale("switcher.aria"), "{title}", title)
}

// FormatSubagentOpenSidebarAria formats open.sidebar.aria.
func FormatSubagentOpenSidebarAria(label string) string {
	return strings.ReplaceAll(SubagentLocale("open.sidebar.aria"), "{label}", label)
}

// FormatSubagentBackParentNamed formats back.parentNamed.
func FormatSubagentBackParentNamed(title string) string {
	return strings.ReplaceAll(SubagentLocale("back.parentNamed"), "{title}", title)
}

// FormatSubagentAncestorCurrent formats ancestor.current.
func FormatSubagentAncestorCurrent(title string) string {
	return strings.ReplaceAll(SubagentLocale("ancestor.current"), "{title}", title)
}

// FormatSubagentTokens mirrors formatTokens(value, t) in ui-subagent/lib/client.js.
func FormatSubagentTokens(value float64) string {
	var formatted string
	switch {
	case value >= 1_000_000:
		formatted = strings.TrimSuffix(fmt.Sprintf("%.1f", value/1_000_000), ".0") + "M"
	case value >= 1_000:
		formatted = strings.TrimSuffix(fmt.Sprintf("%.1f", value/1_000), ".0") + "K"
	default:
		formatted = strconv.Itoa(int(math.Round(value)))
	}
	return strings.ReplaceAll(SubagentLocale("tokens.total"), "{value}", formatted)
}

// FormatSubagentDuration mirrors formatDuration(ms, t) in ui-subagent/lib/client.js:
// seconds, minutes, hours, days, daysHours, months, monthsDays, years, yearsMonths.
func FormatSubagentDuration(ms float64) string {
	totalSeconds := int(math.Floor(math.Max(0, ms) / 1000))
	totalMinutes := totalSeconds / 60
	totalHours := totalMinutes / 60
	days := totalHours / 24
	hours := totalHours % 24
	minutes := totalMinutes % 60
	seconds := totalSeconds % 60

	if days >= 365 {
		years := days / 365
		months := (days % 365) / 30
		if months == 0 {
			return formatSubagentTemplate("duration.years", "years", strconv.Itoa(years))
		}
		return formatSubagentTemplate("duration.yearsMonths", "years", strconv.Itoa(years), "months", strconv.Itoa(months))
	}
	if days >= 30 {
		months := days / 30
		remDays := days % 30
		if remDays == 0 {
			return formatSubagentTemplate("duration.months", "months", strconv.Itoa(months))
		}
		return formatSubagentTemplate("duration.monthsDays", "months", strconv.Itoa(months), "days", strconv.Itoa(remDays))
	}
	if days > 0 {
		if hours == 0 {
			return formatSubagentTemplate("duration.days", "days", strconv.Itoa(days))
		}
		return formatSubagentTemplate("duration.daysHours", "days", strconv.Itoa(days), "hours", strconv.Itoa(hours))
	}
	if totalHours > 0 {
		return formatSubagentTemplate("duration.hours",
			"hours", strconv.Itoa(totalHours),
			"minutes", strconv.Itoa(minutes),
			"seconds", strconv.Itoa(seconds),
		)
	}
	if totalMinutes > 0 {
		return formatSubagentTemplate("duration.minutes",
			"minutes", strconv.Itoa(totalMinutes),
			"seconds", strconv.Itoa(seconds),
		)
	}
	return formatSubagentTemplate("duration.seconds", "seconds", strconv.Itoa(seconds))
}

// FormatSubagentExactDuration mirrors formatExactDuration(ms, t) in ui-subagent/lib/client.js:
// returns FormatSubagentDuration when days == 0, otherwise duration.exactDays.
func FormatSubagentExactDuration(ms float64) string {
	totalSeconds := int(math.Floor(math.Max(0, ms) / 1000))
	totalMinutes := totalSeconds / 60
	totalHours := totalMinutes / 60
	days := totalHours / 24
	hours := totalHours % 24
	minutes := totalMinutes % 60
	seconds := totalSeconds % 60
	if days == 0 {
		return FormatSubagentDuration(ms)
	}
	return formatSubagentTemplate("duration.exactDays",
		"days", strconv.Itoa(days),
		"hours", strconv.Itoa(hours),
		"minutes", strconv.Itoa(minutes),
		"seconds", strconv.Itoa(seconds),
	)
}

// FormatSubagentExactDurationTitle formats duration.exactTitle.
func FormatSubagentExactDurationTitle(duration string) string {
	return strings.ReplaceAll(SubagentLocale("duration.exactTitle"), "{duration}", duration)
}

// FormatJobOutputToggleAria formats row.expandAria / row.collapseAria.
func FormatJobOutputToggleAria(label string, expanded bool) string {
	key := "row.expandAria"
	if expanded {
		key = "row.collapseAria"
	}
	return strings.ReplaceAll(JobsLocale(key), "{label}", label)
}

// FormatJobOutputStreamError formats output.error.
func FormatJobOutputStreamError(errText string) string {
	return strings.ReplaceAll(JobsLocale("output.error"), "{error}", errText)
}

// FormatJobTerminalExpandAria formats terminal.expandAria.
func FormatJobTerminalExpandAria(lines int) string {
	return strings.ReplaceAll(JobsLocale("terminal.expandAria"), "{n}", fmt.Sprintf("%d", lines))
}

// FormatJobsDurationTitle formats duration.title.live or duration.title.done.
func FormatJobsDurationTitle(live bool, duration string) string {
	key := "duration.title.done"
	if live {
		key = "duration.title.live"
	}
	return strings.ReplaceAll(JobsLocale(key), "{duration}", duration)
}

// FormatJobsTerminalExpand formats terminal.expand.
func FormatJobsTerminalExpand(n int) string {
	return strings.ReplaceAll(JobsLocale("terminal.expand"), "{n}", strconv.Itoa(n))
}

// FormatJobsTerminalExitCode formats terminal.exitCode.
func FormatJobsTerminalExitCode(code int) string {
	return strings.ReplaceAll(JobsLocale("terminal.exitCode"), "{code}", strconv.Itoa(code))
}

// FormatJobsKillStop formats kill.stop.
func FormatJobsKillStop(label string) string {
	return strings.ReplaceAll(JobsLocale("kill.stop"), "{label}", label)
}

// FormatJobsLiveCount formats count.live.one or count.live.other.
func FormatJobsLiveCount(n int) string {
	key := "count.live.other"
	if n == 1 {
		key = "count.live.one"
	}
	return strings.ReplaceAll(JobsLocale(key), "{n}", strconv.Itoa(n))
}

// FormatJobsIdleCount formats count.idle.one or count.idle.other.
func FormatJobsIdleCount(n int) string {
	key := "count.idle.other"
	if n == 1 {
		key = "count.idle.one"
	}
	return strings.ReplaceAll(JobsLocale(key), "{n}", strconv.Itoa(n))
}

// FormatAgentTeamFailure formats failure.
func FormatAgentTeamFailure(message string) string {
	return strings.ReplaceAll(AgentTeamLocale("failure"), "{message}", message)
}

// FormatAgentTeamTrigger formats project.trigger.
func FormatAgentTeamTrigger(running int) string {
	return strings.ReplaceAll(AgentTeamLocale("project.trigger"), "{count}", fmt.Sprintf("%d", running))
}

// FormatAgentTeamTriggerHint formats project.triggerHint.
func FormatAgentTeamTriggerHint(count int) string {
	return strings.ReplaceAll(AgentTeamLocale("project.triggerHint"), "{count}", strconv.Itoa(count))
}

// FormatAgentTeamRange formats project.range.
func FormatAgentTeamRange(start, end, total int) string {
	s := AgentTeamLocale("project.range")
	s = strings.ReplaceAll(s, "{start}", fmt.Sprintf("%d", start))
	s = strings.ReplaceAll(s, "{end}", fmt.Sprintf("%d", end))
	s = strings.ReplaceAll(s, "{total}", fmt.Sprintf("%d", total))
	return s
}
