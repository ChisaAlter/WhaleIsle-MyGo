package agents

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestAgentsSubagentJobsTeamLocalesParity(t *testing.T) {
	defer i18n.SetLocale(i18n.Locale())

	i18n.SetLocale(i18n.LocaleZH)
	if got := AgentsPanelLocale("guide.title"); got != "代理与后台任务" {
		t.Fatalf("zh guide.title = %q", got)
	}
	if got := AgentsPanelLocale("empty.title"); got != "还没有子代理" {
		t.Fatalf("zh empty.title = %q", got)
	}
	if got := AgentsPanelLocale("jobs.status.killed"); got != "已终止" {
		t.Fatalf("zh jobs.status.killed = %q", got)
	}
	if got := SubagentLocale("loading.label"); got != "正在加载子智能体…" {
		t.Fatalf("zh loading.label = %q", got)
	}
	if got := SubagentLocale("catalog.migrationRequired"); got != "需要迁移" {
		t.Fatalf("zh catalog.migrationRequired = %q", got)
	}
	if got := FormatSubagentBranchToggle("Worker", false); got != "收起 Worker 的下级子智能体" {
		t.Fatalf("zh FormatSubagentBranchToggle = %q", got)
	}
	if got := FormatSubagentRunningCount(2); got != "2 个子智能体，正在运行" {
		t.Fatalf("zh FormatSubagentRunningCount = %q", got)
	}
	if got := FormatJobOutputToggleAria("build", false); got != "展开 build 的实时输出" {
		t.Fatalf("zh FormatJobOutputToggleAria = %q", got)
	}
	if got := FormatJobOutputStreamError("EOF"); got != "实时输出流中断：EOF" {
		t.Fatalf("zh FormatJobOutputStreamError = %q", got)
	}
	if got := FormatAgentTeamTrigger(3); got != "智能体团队，3 名成员运行中" {
		t.Fatalf("zh FormatAgentTeamTrigger = %q", got)
	}
	if got := childReadOnlyTitle(session.AgentNavigationView{Address: session.AgentSessionAddress{Mode: "one-shot"}}); got != "一次性子智能体记录" {
		t.Fatalf("zh childReadOnlyTitle(one-shot) = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := AgentsPanelLocale("guide.description"); got != "Review subagents and background jobs in this session." {
		t.Fatalf("en guide.description = %q", got)
	}
	if got := SubagentLocale("load.error"); got != "Unable to load subagents" {
		t.Fatalf("en load.error = %q", got)
	}
	if got := FormatSubagentBranchToggle("Worker", true); got != "Expand Worker descendants" {
		t.Fatalf("en FormatSubagentBranchToggle = %q", got)
	}
	if got := FormatSubagentRunningCount(1); got != "1 subagent running" {
		t.Fatalf("en FormatSubagentRunningCount(1) = %q", got)
	}
	if got := FormatJobTerminalExpandAria(42); got != "Expand 42 collapsed output lines" {
		t.Fatalf("en FormatJobTerminalExpandAria = %q", got)
	}
	if got := FormatAgentTeamRange(1, 5, 10); got != "1–5 / 10" {
		t.Fatalf("en FormatAgentTeamRange = %q", got)
	}
	if got := childReadOnlyTitle(session.AgentNavigationView{Address: session.AgentSessionAddress{Mode: "continuable"}}); got != "This subagent is read-only for now" {
		t.Fatalf("en childReadOnlyTitle(continuable) = %q", got)
	}
}
