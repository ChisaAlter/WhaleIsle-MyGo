package chat

import "github.com/egoist/mygo/ui"

var (
	toolIconChevronDown = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1" xmlns="http://www.w3.org/2000/svg"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor" /></svg>`))
	toolIconChevronUp   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1" xmlns="http://www.w3.org/2000/svg"><path d="M12 10L8.70711 6.70711C8.31658 6.31658 7.68342 6.31658 7.29289 6.70711L4 10" stroke="currentColor" /></svg>`))
	toolIconCode        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1" xmlns="http://www.w3.org/2000/svg"><path d="M6.27612 1.5L4.52612 14.5M11.4739 1.5L9.72388 14.5M2.39868 5.5H14.0681M1.93188 10.5H13.6013" stroke="currentColor" /></svg>`))
)

func toolBodyKind(name string) ToolKind {
	switch name {
	case "bash", "pwsh":
		return ToolKindBash
	case "read", "read_image":
		return ToolKindRead
	case "write", "edit":
		return ToolKindWrite
	case "grep", "glob":
		return ToolKindSearch
	case "web_search", "web_fetch":
		return ToolKindWeb
	case "todo_write":
		return ToolKindTodo
	default:
		return ToolKindGeneric
	}
}

func toolKindIconForName(name string) *ui.SVG {
	switch toolRowVariant(name) {
	case "bash":
		return toolIconTerminal
	case "read":
		return toolIconRead
	case "write", "edit":
		return toolIconWrite
	case "search":
		return toolIconSearch
	case "code":
		return toolIconCode
	default:
		return toolIconGeneric
	}
}
