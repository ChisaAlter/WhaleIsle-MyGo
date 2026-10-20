package chat

import "github.com/egoist/mygo/ui"

// Tool-kind artwork for the conversation rows. The Original ships one icon per
// tool family; these 16-unit outline glyphs stand in for that vocabulary until
// the tool packages publish their own marks.
var (
	toolIconTerminal = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <rect x="1.5" y="2.5" width="13" height="11" rx="2" stroke="currentColor" />
    <path d="M4.5 6L7 8L4.5 10" stroke="currentColor" />
    <path d="M8.5 10H11.5" stroke="currentColor" />
  </svg>`))
	toolIconRead = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M4 1.5H9.5L13 5V14.5H4V1.5Z" stroke="currentColor" />
    <path d="M9.5 1.5V5H13" stroke="currentColor" />
  </svg>`))
	toolIconWrite = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M2.5 13.5L3.5 10.2L10.8 2.9L13.1 5.2L5.8 12.5L2.5 13.5Z" stroke="currentColor" />
    <path d="M9.6 4.1L11.9 6.4" stroke="currentColor" />
  </svg>`))
	toolIconSearch = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <circle cx="6.8" cy="6.8" r="4.6" stroke="currentColor" />
    <path d="M10.4 10.4L14 14" stroke="currentColor" />
  </svg>`))
	toolIconWeb = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <circle cx="8" cy="8" r="6" stroke="currentColor" />
    <path d="M2 8H14" stroke="currentColor" />
    <path d="M8 2C9.9 3.9 10.9 5.9 10.9 8S9.9 12.1 8 14C6.1 12.1 5.1 10.1 5.1 8S6.1 3.9 8 2Z" stroke="currentColor" />
  </svg>`))
	toolIconTodo = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M2 4.5L3.6 6L6 3.5" stroke="currentColor" />
    <path d="M2 11.5L3.6 13L6 10.5" stroke="currentColor" />
    <path d="M8.5 4.5H14" stroke="currentColor" />
    <path d="M8.5 11.5H14" stroke="currentColor" />
  </svg>`))
	toolIconAsk = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <path d="M13.5 8.2C13.5 10.8 11 12.9 8 12.9C7.3 12.9 6.6 12.8 6 12.6L2.5 14L3.5 11.4C2.6 10.5 2 9.4 2 8.2C2 5.6 4.5 3.5 8 3.5S13.5 5.6 13.5 8.2Z" stroke="currentColor" />
  </svg>`))
	toolIconAgent = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <circle cx="4" cy="4" r="2.2" stroke="currentColor" />
    <circle cx="12" cy="12" r="2.2" stroke="currentColor" />
    <path d="M4 6.2V9C4 10.1 4.9 11 6 11H9.6" stroke="currentColor" />
  </svg>`))
	toolIconGeneric = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke-width="1">
    <circle cx="8" cy="8" r="2.4" stroke="currentColor" />
    <path d="M8 1.6V3.4M8 12.6V14.4M1.6 8H3.4M12.6 8H14.4M3.5 3.5L4.8 4.8M11.2 11.2L12.5 12.5M12.5 3.5L11.2 4.8M4.8 11.2L3.5 12.5" stroke="currentColor" />
  </svg>`))
)

func toolKindIcon(kind ToolKind) *ui.SVG {
	switch kind {
	case ToolKindBash:
		return toolIconTerminal
	case ToolKindRead:
		return toolIconRead
	case ToolKindWrite:
		return toolIconWrite
	case ToolKindSearch:
		return toolIconSearch
	case ToolKindWeb:
		return toolIconWeb
	case ToolKindTodo:
		return toolIconTodo
	case ToolKindAsk:
		return toolIconAsk
	case ToolKindSubagent:
		return toolIconAgent
	default:
		return toolIconGeneric
	}
}
