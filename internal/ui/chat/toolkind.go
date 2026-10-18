package chat

import "strings"

// ToolKind is a coarse UI category for tool cards.
type ToolKind string

const (
	ToolKindGeneric  ToolKind = "tool"
	ToolKindBash     ToolKind = "bash"
	ToolKindRead     ToolKind = "read"
	ToolKindWrite    ToolKind = "write"
	ToolKindSearch   ToolKind = "search"
	ToolKindWeb      ToolKind = "web"
	ToolKindTodo     ToolKind = "todo"
	ToolKindAsk      ToolKind = "ask"
	ToolKindSubagent ToolKind = "subagent"
)

// ClassifyToolName maps a tool/call name to a display kind + short glyph label.
func ClassifyToolName(name string) (ToolKind, string) {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.TrimPrefix(n, "tool/")
	switch {
	case strings.Contains(n, "bash") || strings.Contains(n, "shell") || strings.Contains(n, "terminal") || n == "exec":
		return ToolKindBash, "bash"
	case strings.Contains(n, "todo") || strings.Contains(n, "task_list"):
		return ToolKindTodo, "todo"
	case strings.Contains(n, "subagent") || n == "task" || strings.HasPrefix(n, "task_") || strings.Contains(n, "agent"):
		return ToolKindSubagent, "agent"
	case n == "ask" || strings.HasPrefix(n, "ask_") || strings.Contains(n, "question") || strings.Contains(n, "confirm"):
		return ToolKindAsk, "ask"
	case strings.Contains(n, "read") || strings.Contains(n, "cat") || strings.Contains(n, "open_file"):
		return ToolKindRead, "read"
	case strings.Contains(n, "write") || strings.Contains(n, "edit") || strings.Contains(n, "apply_patch") || strings.Contains(n, "str_replace"):
		return ToolKindWrite, "write"
	case strings.Contains(n, "search") || strings.Contains(n, "grep") || strings.Contains(n, "glob") || strings.Contains(n, "find"):
		return ToolKindSearch, "search"
	case strings.Contains(n, "web") || strings.Contains(n, "fetch") || strings.Contains(n, "browser") || strings.Contains(n, "http"):
		return ToolKindWeb, "web"
	default:
		return ToolKindGeneric, "tool"
	}
}
