package agents

import (
	"fmt"
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
	"jobs.status.completed":  "已完成",
	"jobs.status.failed":     "失败",
	"jobs.status.killed":     "已终止",
	"jobs.status.terminated": "已终止",
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
	"jobs.status.completed":  "completed",
	"jobs.status.failed":     "failed",
	"jobs.status.killed":     "killed",
	"jobs.status.terminated": "terminated",
}

// SubagentLocalesZH and SubagentLocalesEN synchronize @deepseek-ai/dsh-client-ui-subagent.
var SubagentLocalesZH = map[string]string{
	"duration.seconds":          "{seconds}秒",
	"duration.minutes":          "{minutes}分{seconds}秒",
	"duration.hours":            "{hours}小时{minutes}分",
	"duration.days":             "{days}天",
	"duration.years":            "约{years}年",
	"duration.exactTitle":       "总活跃耗时：{duration}",
	"tokens.thousand":           "{value}K",
	"tokens.million":            "{value}M",
	"loading.label":             "正在加载子智能体…",
	"load.error":                "无法加载子智能体",
	"catalog.migrationRequired": "需要迁移",
	"branch.collapse":           "收起 {label} 的下级子智能体",
	"branch.expand":             "展开 {label} 的下级子智能体",
	"count.running.one":         "{count} 个子智能体，正在运行",
	"count.running.other":       "{count} 个子智能体，正在运行",
	"switcher.aria":             "切换子智能体：{title}",
	"tree.aria":                 "子智能体会话",
	"readonly.oneShot.title":    "一次性子智能体记录",
	"readonly.title":            "此子智能体暂时只读",
}

var SubagentLocalesEN = map[string]string{
	"duration.seconds":          "{seconds}s",
	"duration.minutes":          "{minutes}m {seconds}s",
	"duration.hours":            "{hours}h {minutes}m",
	"duration.days":             "{days}d",
	"duration.years":            "~{years}y",
	"duration.exactTitle":       "Total active duration: {duration}",
	"tokens.thousand":           "{value}K",
	"tokens.million":            "{value}M",
	"loading.label":             "Loading subagents…",
	"load.error":                "Unable to load subagents",
	"catalog.migrationRequired": "Migration required",
	"branch.collapse":           "Collapse {label} descendants",
	"branch.expand":             "Expand {label} descendants",
	"count.running.one":         "{count} subagent running",
	"count.running.other":       "{count} subagents running",
	"switcher.aria":             "Switch subagent: {title}",
	"tree.aria":                 "Subagent sessions",
	"readonly.oneShot.title":    "One-shot subagent record",
	"readonly.title":            "This subagent is read-only for now",
}

// JobsLocalesZH and JobsLocalesEN synchronize @deepseek-ai/dsh-client-ui-jobs.
var JobsLocalesZH = map[string]string{
	"row.expandAria":      "展开 {label} 的实时输出",
	"row.collapseAria":    "收起 {label} 的实时输出",
	"kill.failed":         "停止失败",
	"duration.seconds":    "{seconds}秒",
	"duration.minutes":    "{minutes}分{seconds}秒",
	"output.error":        "实时输出流中断：{error}",
	"terminal.expandAria": "展开被折叠的 {n} 行输出",
}

var JobsLocalesEN = map[string]string{
	"row.expandAria":      "Show live output of {label}",
	"row.collapseAria":    "Hide live output of {label}",
	"kill.failed":         "Stop failed",
	"duration.seconds":    "{seconds}s",
	"duration.minutes":    "{minutes}m {seconds}s",
	"output.error":        "live output stream interrupted: {error}",
	"terminal.expandAria": "Expand {n} collapsed output lines",
}

// AgentTeamLocalesZH and AgentTeamLocalesEN synchronize @deepseek-ai/dsh-experimental-client-ui-agent-team.
var AgentTeamLocalesZH = map[string]string{
	"loading":         "正在加载团队…",
	"project.trigger": "智能体团队，{count} 名成员运行中",
	"project.range":   "{start}–{end} / {total}",
}

var AgentTeamLocalesEN = map[string]string{
	"loading":         "Loading Team…",
	"project.trigger": "Agent Team, {count} members running",
	"project.range":   "{start}–{end} / {total}",
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

// FormatSubagentSwitcherAria formats switcher.aria.
func FormatSubagentSwitcherAria(title string) string {
	return strings.ReplaceAll(SubagentLocale("switcher.aria"), "{title}", title)
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

// FormatAgentTeamTrigger formats project.trigger.
func FormatAgentTeamTrigger(running int) string {
	return strings.ReplaceAll(AgentTeamLocale("project.trigger"), "{count}", fmt.Sprintf("%d", running))
}

// FormatAgentTeamRange formats project.range.
func FormatAgentTeamRange(start, end, total int) string {
	s := AgentTeamLocale("project.range")
	s = strings.ReplaceAll(s, "{start}", fmt.Sprintf("%d", start))
	s = strings.ReplaceAll(s, "{end}", fmt.Sprintf("%d", end))
	s = strings.ReplaceAll(s, "{total}", fmt.Sprintf("%d", total))
	return s
}
