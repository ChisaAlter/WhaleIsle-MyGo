package chat

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// These are exact wire names from the installed tool-call model. Keyed search
// and web providers refine titles without changing the generic model's variant.
var toolRowTitleNames = map[string]string{
	"cordis_package_inspect": "inspect", "cordis_runtime_inspect": "inspect",
	"cordis_run": "runCordis", "cordis_stop": "stopCordis", "cordis_undefine": "removeCordis",
	"pwsh": "pwsh", "read_image": "readImage", "todo_write": "todo", "ask_user_question": "ask",
	"create_goal": "createGoal", "get_goal": "getGoal", "update_goal": "updateGoal",
	"schedule_create": "createSchedule", "schedule_list": "listSchedules", "schedule_delete": "deleteSchedule", "schedule_update": "updateSchedule",
	"cordis_inspect_list": "inspectProviders", "cordis_inspect_query": "queryRuntime", "cordis_inspect_self": "inspectPlugins",
	"workflow": "workflow", "ralph": "ralph",
	"session_event_read": "readEvent", "session_event_search": "searchEvents", "session_event_trace": "traceEvent", "session_search": "searchSessions", "session_trace": "traceSession",
	"list_subagent_models": "listModels", "subagent": "subagent", "list_agents": "listAgents", "send_message": "sendMessage", "interrupt_agent": "interruptAgent",
	"job_list": "listJobs", "job_output": "readJob", "job_kill": "killJob",
	"terminal_open": "openTerminal", "terminal_read": "readTerminal", "terminal_list": "listTerminals", "terminal_signal": "signalTerminal", "terminal_close": "closeTerminal",
	"lsp": "lsp", "spawn_teammate": "spawnTeammate", "team_task_create": "createTeamTask", "team_task_get": "getTeamTask", "team_task_update": "updateTeamTask", "team_task_list": "listTeamTasks", "wait_agent": "waitAgent",
	"grep": "grep", "glob": "glob", "web_search": "webSearch", "web_fetch": "webFetch",
}

func toolRowVariant(name string) string {
	switch name {
	case "bash", "pwsh":
		return "bash"
	case "read", "read_image", "web_fetch", "cordis_package_inspect", "cordis_runtime_inspect":
		return "read"
	case "web_search", "grep", "glob":
		return "search"
	case "write":
		return "write"
	case "edit":
		return "edit"
	case "run_code":
		return "code"
	default:
		return "others"
	}
}

func toolRowTitleName(name string) string {
	if title, ok := toolRowTitleNames[name]; ok {
		return title
	}
	// Only the legacy shell title aliases bash. Classification remains exact.
	if name == "shell" {
		return "bash"
	}
	if variant := toolRowVariant(name); variant != "others" {
		return variant
	}
	return "generic"
}

func toolRowTitle(name string) string {
	if toolRowTitleName(name) == "search" {
		return i18n.T(i18n.KeyChatToolSearch)
	}
	return i18n.T("chat.tool.row.title." + toolRowTitleName(name))
}

func toolRowStatusLabel(status string) string {
	switch status {
	case "preparing":
		return i18n.T(i18n.KeyToolRowPreparing)
	case "running", "call":
		return i18n.T(i18n.KeyChatToolRunning)
	case "error", "failed":
		return i18n.T(i18n.KeyChatToolFailed)
	case "stopped", "cancelled", "canceled":
		return i18n.T(i18n.KeyChatToolStopped)
	default:
		return ""
	}
}

// toolRowSummary uses full argument text, never the truncated Detail field.
// Preparing calls currently expose only raw JSON, not the original lazy argument
// view; incomplete payloads therefore retain their literal first line.
func toolRowSummary(info ToolCardInfo) string {
	if info.terminal != nil && terminalDescription(info.terminal) != "" {
		return terminalDescription(info.terminal)
	}
	if (info.Status == "error" || info.Status == "failed") && info.Output != "" {
		return toolRowFirstLine(info.Output)
	}
	base := toolRowArgumentSummary(toolRowVariant(info.Name), info.Arguments)
	if toolRowTitleName(info.Name) != "generic" {
		return base
	}
	if base == "" {
		return info.Name
	}
	if info.Name == "" {
		return base
	}
	return info.Name + " · " + base
}

func toolRowFirstLine(text string) string {
	if end := strings.IndexByte(text, '\n'); end >= 0 {
		return text[:end]
	}
	return text
}

type toolRowArgumentField struct {
	key   string
	value json.RawMessage
}

// Keep object insertion order for the final first-string fallback. A decoded
// Go map cannot retain the wire order. Duplicate keys retain their original
// position and final value, matching JSON.parse's object update semantics.
func toolRowArgumentFields(raw string) ([]toolRowArgumentField, bool) {
	if !json.Valid([]byte(raw)) {
		return nil, false
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	token, err := decoder.Token()
	if err != nil {
		return nil, false
	}
	delim, object := token.(json.Delim)
	if !object || (delim != '{' && delim != '[') {
		return nil, false
	}
	var fields []toolRowArgumentField
	positions := map[string]int{}
	for decoder.More() {
		key := ""
		if delim == '{' {
			token, err = decoder.Token()
			if err != nil {
				return nil, false
			}
			key = token.(string)
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, false
		}
		if index, exists := positions[key]; exists && delim == '{' {
			fields[index].value = value
		} else {
			positions[key] = len(fields)
			fields = append(fields, toolRowArgumentField{key, value})
		}
	}
	return fields, true
}

func toolRowArgumentString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var text string
	if json.Unmarshal(raw, &text) != nil || text == "" {
		return "", false
	}
	return text, true
}

func toolRowArgumentSummary(variant, raw string) string {
	fields, parsedObject := toolRowArgumentFields(raw)
	if !parsedObject {
		return toolRowFirstLine(raw)
	}
	field := func(key string) json.RawMessage {
		for _, field := range fields {
			if field.key == key {
				return field.value
			}
		}
		return nil
	}
	pick := func(keys ...string) (string, bool) {
		for _, key := range keys {
			if value, ok := toolRowArgumentString(field(key)); ok {
				return toolRowFirstLine(value), true
			}
		}
		return "", false
	}
	file := variant == "read" || variant == "write" || variant == "edit"
	// The original argument view gives file_path priority only when complete.
	// Every successfully decoded string in this complete JSON payload is complete.
	primary := ""
	if file {
		primary, _ = pick("file_path")
	} else {
		primary, _ = pick("description")
	}
	if primary != "" {
		return primary
	}
	if variant == "search" {
		var queries []json.RawMessage
		if json.Unmarshal(field("queries"), &queries) == nil {
			var lines []string
			for _, query := range queries {
				if value, ok := toolRowArgumentString(query); ok {
					lines = append(lines, toolRowFirstLine(value))
				}
			}
			if len(lines) > 0 {
				return strings.Join(lines, ", ")
			}
		}
	}
	var keys []string
	switch variant {
	case "bash":
		keys = []string{"description", "command"}
	case "read":
		keys = []string{"path", "file_path", "url"}
	case "search":
		keys = []string{"query", "pattern", "url"}
	case "write", "edit":
		keys = []string{"path", "file_path"}
	case "code":
		keys = []string{"description"}
	}
	if value, ok := pick(keys...); ok {
		return value
	}
	for _, field := range fields {
		if value, ok := toolRowArgumentString(field.value); ok {
			return toolRowFirstLine(value)
		}
	}
	return toolRowFirstLine(raw)
}
