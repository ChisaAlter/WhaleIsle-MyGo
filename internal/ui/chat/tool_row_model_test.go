package chat

import (
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestToolRowModelTitlesAndVariants(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, tc := range []struct{ name, variant, en, zh string }{
		{"bash", "bash", "Bash", "运行命令"},
		{"shell", "others", "Bash", "运行命令"},
		{"pwsh", "bash", "Pwsh", "运行命令"},
		{"read", "read", "Read", "读取"},
		{"read_image", "read", "Read image", "读取图片"},
		{"write", "write", "Write", "写入"},
		{"edit", "edit", "Edit", "编辑"},
		{"run_code", "code", "Code", "代码"},
		{"grep", "search", "Grep", "搜索文件内容"},
		{"glob", "search", "Glob", "查找文件"},
		{"web_search", "search", "Search", "网页搜索"},
		{"web_fetch", "read", "Fetch", "网页获取"},
		{"cordis_package_inspect", "read", "Query Cordis environment", "查询 Cordis 环境"},
		{"cordis_runtime_inspect", "read", "Query Cordis environment", "查询 Cordis 环境"},
		{"cordis_run", "others", "Run Cordis Plugin", "运行 Cordis 插件"},
		{"cordis_stop", "others", "Stop Cordis Plugin", "停止 Cordis 插件"},
		{"cordis_undefine", "others", "Remove Cordis Plugin", "移除 Cordis 插件"},
		{"todo_write", "others", "Update to-do list", "更新任务清单"},
		{"ask_user_question", "others", "Ask question", "提问"},
		{"create_goal", "others", "Create goal", "创建目标"},
		{"get_goal", "others", "View goal", "查看目标"},
		{"update_goal", "others", "Update goal", "更新目标"},
		{"schedule_create", "others", "Create reminder", "创建定时任务"},
		{"schedule_list", "others", "List reminders", "查看定时任务"},
		{"schedule_delete", "others", "Delete reminder", "删除定时任务"},
		{"schedule_update", "others", "Update reminder", "修改定时任务"},
		{"cordis_inspect_list", "others", "Inspect providers", "检查提供方"},
		{"cordis_inspect_query", "others", "Query runtime", "查询运行时"},
		{"cordis_inspect_self", "others", "Inspect plugins", "检查动态插件"},
		{"workflow", "others", "Run workflow", "运行工作流"},
		{"ralph", "others", "Run ralph loop", "运行循环工作流"},
		{"session_event_read", "others", "Read event", "读取事件"},
		{"session_event_search", "others", "Search events", "搜索事件"},
		{"session_event_trace", "others", "Trace event", "追踪事件"},
		{"session_search", "others", "Search sessions", "搜索会话"},
		{"session_trace", "others", "Trace session", "追踪会话"},
		{"list_subagent_models", "others", "List models", "查看可用模型"},
		{"subagent", "others", "Create subagent", "创建子智能体"},
		{"list_agents", "others", "List subagents", "查看子智能体"},
		{"send_message", "others", "Send message", "发送消息"},
		{"interrupt_agent", "others", "Interrupt agent", "中断智能体"},
		{"job_list", "others", "List background jobs", "查看后台任务"},
		{"job_output", "others", "Read job output", "读取任务输出"},
		{"job_kill", "others", "Cancel background job", "取消后台任务"},
		{"terminal_open", "others", "Open terminal", "创建终端"},
		{"terminal_read", "others", "Read terminal", "读取终端"},
		{"terminal_list", "others", "List terminals", "查看终端"},
		{"terminal_signal", "others", "Signal terminal", "发送终端信号"},
		{"terminal_close", "others", "Close terminal", "关闭终端"},
		{"lsp", "others", "Query code symbols", "查询代码符号"},
		{"spawn_teammate", "others", "Create teammate", "创建队友"},
		{"team_task_create", "others", "Create team task", "创建团队任务"},
		{"team_task_get", "others", "Read team task", "读取团队任务"},
		{"team_task_update", "others", "Update team task", "更新团队任务"},
		{"team_task_list", "others", "List team tasks", "查看团队任务"},
		{"wait_agent", "others", "Wait for subagent", "等待子智能体"},
		{"read_file", "others", "Tool call", "工具调用"},
		{"Read", "others", "Tool call", "工具调用"},
		{"plugin:read", "others", "Tool call", "工具调用"},
	} {
		if got := toolRowVariant(tc.name); got != tc.variant {
			t.Errorf("%s variant=%q want%q", tc.name, got, tc.variant)
		}
		for _, locale := range []struct{ id, want string }{{i18n.LocaleEN, tc.en}, {i18n.LocaleZH, tc.zh}} {
			i18n.SetLocale(locale.id)
			if got := toolRowTitle(tc.name); got != locale.want {
				t.Errorf("%s %s title=%q want%q", tc.name, locale.id, got, locale.want)
			}
		}
	}
}

func TestToolRowModelArgumentSummaries(t *testing.T) {
	for _, tc := range []struct{ name, args, want string }{
		{"bash", `{"command":"ls","description":"List source files"}`, "List source files"},
		{"pwsh", `{"command":"Get-Item","description":"  Preserve  spacing\t "}`, "  Preserve  spacing\t "},
		{"read", `{"description":"Never primary for files","path":"old.txt","file_path":"new.txt"}`, "new.txt"},
		{"read_image", `{"file_path":"images/a.png","description":"Image"}`, "images/a.png"},
		{"edit", `{"file_path":2,"path":"fallback.txt"}`, "fallback.txt"},
		{"write", `{"file_path":"","path":"fallback.txt"}`, "fallback.txt"},
		{"read", `{"url":"https://example.test","path":"local.txt"}`, "local.txt"},
		{"web_search", `{"queries":["first\nrest",null,"","second"],"query":"ignored"}`, "first, second"},
		{"web_search", `{"queries":["ignored"],"description":"Primary description"}`, "Primary description"},
		{"grep", `{"pattern":"fallback","query":"first"}`, "first"},
		{"grep", `{"queries":[null,7,""],"pattern":"fallback"}`, "fallback"},
		{"glob", `{"pattern":"*.go","description":"Find source files"}`, "Find source files"},
		{"web_fetch", `{"description":"Fetch ignored","url":"https://example.test"}`, "https://example.test"},
		{"run_code", `{"code":"program","description":"Compile\nrest"}`, "Compile"},
		{"get_goal", `{"arguments":"not recursively parsed"}`, "not recursively parsed"},
		{"custom_wire", `{"number":12,"last":"wire-first","alpha":"map-first"}`, "custom_wire · wire-first"},
		{"custom_wire", `{"z":"old","a":"later","z":"updated"}`, "custom_wire · updated"},
		{"custom_wire", `{"description":"   "}`, "custom_wire ·    "},
		{"custom_wire", `{"nested":{"description":"not flattened"},"fallback":"text"}`, "custom_wire · text"},
		{"custom_wire", `[null,5,"first\nrest","second"]`, "custom_wire · first"},
		{"read", `[1,2]`, `[1,2]`},
		{"read", `"quoted scalar"`, `"quoted scalar"`},
		{"read", `null`, `null`},
		{"read", `{}`, `{}`},
		{"read", "  literal   first\r\nsecond", "  literal   first\r"},
		{"read", "\nsecond", ""},
		{"bash", "{\"description\":\"streaming\nrest", "{\"description\":\"streaming"},
		{"read", `{"file_path":"unfinished`, `{"file_path":"unfinished`},
		{"read", "", ""},
		{"get_goal", "", ""},
		{"unknown", "", "unknown"},
		{"Read", `{"file_path":"x"}`, "Read · x"},
	} {
		if got := toolRowSummary(ToolCardInfo{Name: tc.name, Arguments: tc.args, Detail: "Never a summary source"}); got != tc.want {
			t.Errorf("%s args%q summary=%q want%q", tc.name, tc.args, got, tc.want)
		}
	}
	long := strings.Repeat("完整摘要 ", 80)
	if got := toolRowSummary(ToolCardInfo{Name: "bash", Arguments: `{"description":"` + long + `"}`}); got != long {
		t.Fatalf("long summary clipped: %d want%d", len(got), len(long))
	}
	for i := 0; i < 40; i++ {
		if got := toolRowSummary(ToolCardInfo{Name: "unknown", Arguments: `{"z":"first","a":"second"}`}); got != "unknown · first" {
			t.Fatalf("object fallback lost insertion order: %q", got)
		}
	}
}

func TestToolRowModelSettlementAndStatus(t *testing.T) {
	for _, tc := range []struct{ status, output, want string }{
		{"error", "  first  error\r\nsecond", "  first  error\r"},
		{"failed", "failure\nsecond", "failure"},
		{"error", "\nsecond", ""},
		{"error", "", "wire · base"},
		{"stopped", "failure", "wire · base"},
		{"cancelled", "failure", "wire · base"},
		{"ok", "result", "wire · base"},
	} {
		got := toolRowSummary(ToolCardInfo{Name: "wire", Status: tc.status, Arguments: `{"description":"base"}`, Output: tc.output, Detail: "Do not use Detail"})
		if got != tc.want {
			t.Errorf("%s output%q summary=%q want%q", tc.status, tc.output, got, tc.want)
		}
	}
	if got := toolRowSummary(ToolCardInfo{Name: "read", Status: "failed", Detail: "Detail-only failure"}); got != "" {
		t.Fatalf("Detail became error output: %q", got)
	}
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, tc := range []struct{ status, zh, en string }{
		{"preparing", "正在准备调用", "Preparing tool call"},
		{"running", "运行中", "Running"}, {"call", "运行中", "Running"},
		{"error", "失败", "Failed"}, {"failed", "失败", "Failed"},
		{"stopped", "已停止", "Stopped"}, {"cancelled", "已停止", "Stopped"},
		{"ok", "", ""}, {"", "", ""}, {"event", "", ""},
	} {
		i18n.SetLocale(i18n.LocaleZH)
		if got := toolRowStatusLabel(tc.status); got != tc.zh {
			t.Errorf("%s zh status=%q want%q", tc.status, got, tc.zh)
		}
		i18n.SetLocale(i18n.LocaleEN)
		if got := toolRowStatusLabel(tc.status); got != tc.en {
			t.Errorf("%s en status=%q want%q", tc.status, got, tc.en)
		}
	}
}
