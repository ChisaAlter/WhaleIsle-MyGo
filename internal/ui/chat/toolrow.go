package chat

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Tool rows follow the Original ToolRow: one 24px disclosure line 鈥?kind icon,
// localized title, 2px separator, truncated argument summary, status dot and
// chevron 鈥?and a structured body that only mounts once the row is open.

const (
	toolRowBlockMaxHeight = 220
	toolRowSummaryRunes   = 120
)

func toolKindTitle(kind ToolKind) string {
	switch kind {
	case ToolKindBash:
		return i18n.T(i18n.KeyChatToolRun)
	case ToolKindRead:
		return i18n.T(i18n.KeyChatToolRead)
	case ToolKindWrite:
		return i18n.T(i18n.KeyChatToolWrite)
	case ToolKindSearch:
		return i18n.T(i18n.KeyChatToolSearch)
	case ToolKindWeb:
		return i18n.T(i18n.KeyChatToolWeb)
	case ToolKindTodo:
		return i18n.T(i18n.KeyChatToolTodo)
	case ToolKindAsk:
		return i18n.T(i18n.KeyChatToolAsk)
	case ToolKindSubagent:
		return i18n.T(i18n.KeyChatToolAgent)
	default:
		return i18n.T(i18n.KeyChatToolGeneric)
	}
}

// toolRowStatus maps the lifecycle state onto the shared status language.
func toolRowStatus(status string) rowStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "done", "completed":
		return statusDone
	case "call", "running", "preparing":
		return statusRunning
	case "error", "failed":
		return statusError
	case "stopped", "cancelled", "canceled":
		return statusWarning
	case "":
		return statusNone
	default:
		return statusIdle
	}
}

// toolRowStyle adapts the tool card palette to the disclosure chrome.
func toolRowStyle(st ToolCardStyle) disclosureStyle {
	size := st.FontSize
	if size <= 0 {
		size = 12
	}
	hover := st.Field
	if hover.A == 0 {
		hover = st.Muted.Alpha(.10)
	}
	return disclosureStyle{FG: st.FG, Muted: st.Muted, Hover: hover, Accent: st.Accent, Size: size + 1}
}

// argumentFields lists the JSON keys each tool family puts first in its row
// summary, in the Original's order of relevance.
var argumentFields = map[ToolKind][]string{
	ToolKindBash:     {"command", "cmd", "script", "description", "arguments"},
	ToolKindRead:     {"file_path", "path", "file", "target_file"},
	ToolKindWrite:    {"file_path", "path", "file", "target_file"},
	ToolKindSearch:   {"pattern", "query", "regex", "q", "path"},
	ToolKindWeb:      {"url", "query", "href", "search_query"},
	ToolKindTodo:     {"todos", "items", "plan"},
	ToolKindAsk:      {"question", "header", "prompt"},
	ToolKindSubagent: {"description", "prompt", "agent", "task"},
	ToolKindGeneric:  {"command", "path", "query", "url", "description", "arguments"},
}

// toolArgSummary picks the row's one-line argument summary. JSON arguments are
// read by field; free-form arguments collapse to their first line.
func toolArgSummary(arguments, name string) string {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" {
		return ""
	}
	kind, _ := ClassifyToolName(name)
	if strings.HasPrefix(arguments, "{") || strings.HasPrefix(arguments, "[") {
		var payload map[string]any
		if json.Unmarshal([]byte(arguments), &payload) == nil {
			for _, key := range argumentFields[kind] {
				value, ok := payload[key]
				if !ok {
					continue
				}
				if text := jsonValueSummary(value); text != "" {
					return clipRunes(text, toolRowSummaryRunes)
				}
			}
			// A command-style payload may nest arguments as a JSON string.
			if nested, ok := payload["arguments"].(string); ok && nested != "" {
				return toolArgSummary(nested, name)
			}
		}
	}
	return clipRunes(collapseSpace(arguments), toolRowSummaryRunes)
}

func jsonValueSummary(value any) string {
	switch v := value.(type) {
	case string:
		return collapseSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case []any:
		if len(v) == 1 {
			return jsonValueSummary(v[0])
		}
		return strconv.Itoa(len(v)) + " items"
	case map[string]any:
		if command, ok := v["command"].(string); ok {
			return collapseSpace(command)
		}
		return ""
	default:
		return ""
	}
}

func collapseSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// paintToolRow draws the single-line row plus its body when open.
func paintToolRow(c *ui.Context, kind ToolKind, title, summary string, status rowStatus, st ToolCardStyle, key string, shared *map[string]bool, body func()) {
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(2)
	open := false
	if shared != nil && key != "" {
		open = (*shared)[key]
	} else if key != "" {
		open = *rowOpenState(root, key, false)
	}
	root.Children(func() {
		paintDisclosureRow(c, disclosureRow{
			Key:        "tool-row:" + key,
			Icon:       toolKindIcon(kind),
			Title:      title,
			Summary:    summary,
			KeepSummary: true,
			Status:     status,
			Running:    status == statusRunning,
			Open:       open,
			Expandable: body != nil,
			OnToggle: func() {
				if body == nil {
					return
				}
				if shared != nil && key != "" {
					if *shared == nil {
						*shared = map[string]bool{}
					}
					(*shared)[key] = !open
					return
				}
				state := rowOpenState(root, key, false)
				*state = !*state
			},
		}, toolRowStyle(st))
		if body != nil && open {
			ui.Column(c).FillWidth().MinWidth(0).Padding(0, 0, 2, 6).Children(body)
		}
	})
}

// toolNumberedBlock renders file content with a muted line-number gutter, as the
// Original ReadBlock does.
func toolNumberedBlock(c *ui.Context, key, body string, st ToolCardStyle) {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return
	}
	lines := strings.Split(body, "\n")
	if len(lines) > 400 {
		lines = lines[:400]
	}
	var out []ui.Span
	font := theme.RenderPreferences().Code.Family
	size := CodeSizeFor(st)
	for index, line := range lines {
		out = append(out, ui.Span{Text: padLine(index+1, len(lines)) + "  ", Size: size, Color: st.Muted.Alpha(.7), Font: font})
		out = append(out, ui.Span{Text: line, Size: size, Color: st.FG, Font: font})
		if index < len(lines)-1 {
			out = append(out, ui.Span{Text: "\n", Size: size, Font: font})
		}
	}
	ui.Box(c).Key("tool-file:"+key).FillWidth().MinWidth(0).Background(st.Muted.Alpha(.06)).Radius(6).
		Padding(6, 8, 6, 8).Children(func() {
		ui.Scroll(c).FillWidth().MaxHeight(toolRowBlockMaxHeight).Children(func() {
			ui.RichText(c, out...).Font(font).FixedLineHeight(size + 6).TextColor(st.FG)
		})
	})
}

func padLine(n, total int) string {
	width := len(strconv.Itoa(total))
	text := strconv.Itoa(n)
	for len(text) < width {
		text = " " + text
	}
	return text
}

// toolStructuredBody renders the per-family detail the Original toolviews own:
// a terminal card for commands, a numbered file card for reads, the query plus
// matches for searches and the payload for web/todo calls. It reports whether
// the family was handled (the caller otherwise falls back to the argument and
// result disclosures) and whether that body already owns the copy action.
func toolStructuredBody(c *ui.Context, kind ToolKind, info ToolCardInfo, st ToolCardStyle, key string, opts ToolImageGalleryOptions) (handled, ownsCopy bool) {
	switch kind {
	case ToolKindBash:
		command := toolArgField(info.Arguments, argumentFields[kind])
		if command == "" && info.Output == "" {
			return false, false
		}
		// One terminal seat holds the command and its output, as the Original
		// TerminalBlock does; long lines scroll inside it instead of widening
		// the bubble.
		terminalBlock(c, terminalBlockStyle{FG: st.FG, Muted: st.Muted, Field: st.Field, Size: CodeSizeFor(st)},
			key, command, info.Output, func(text string) { c.WriteClipboard(text) })
		return true, true
	case ToolKindRead:
		path := toolArgField(info.Arguments, argumentFields[kind])
		if path == "" && info.Output == "" {
			return false, false
		}
		if path != "" {
			ui.Text(c, i18n.T(i18n.KeyChatToolPath)).FontSize(11).TextColor(st.Muted)
			ui.Text(c, path).FontSize(11).Font(theme.RenderPreferences().Code.Family).TextColor(st.FG).Selectable()
		}
		toolNumberedBlock(c, key+":content", info.Output, st)
		return true, false
	case ToolKindSearch:
		query := toolArgField(info.Arguments, argumentFields[kind])
		if query == "" && info.Output == "" {
			return false, false
		}
		// Hits keep path:line plus the highlighted snippet in one scrolled seat.
		searchHitsBlock(c, toolSeatStyle{FG: st.FG, Muted: st.Muted, Field: st.Field, Size: CodeSizeFor(st)},
			key+":hits", query, info.Output, func(text string) { c.WriteClipboard(text) })
		return true, true
	case ToolKindWeb:
		if info.Output == "" && info.Web == nil {
			return false, false
		}
		openURL := opts.OnOpenURL
		if opts.TextOnly {
			openURL = nil
		}
		webResultBlock(c, toolSeatStyle{FG: st.FG, Muted: st.Muted, Field: st.Field, Accent: st.Accent, Size: CodeSizeFor(st)},
			key+":page", info.Web, info, openURL, func(text string) { c.WriteClipboard(text) })
		return true, true
	case ToolKindTodo:
		if info.Output == "" {
			return false, false
		}
		todoBlock(c, toolSeatStyle{FG: st.FG, Muted: st.Muted, Field: st.Field, Size: CodeSizeFor(st)},
			key+":plan", info.Output, func(text string) { c.WriteClipboard(text) })
		return true, true
	default:
		return false, false
	}
}

// toolField returns the first non-empty string field among keys.
func toolArgField(arguments string, keys []string) string {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" || !strings.HasPrefix(arguments, "{") {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal([]byte(arguments), &payload) != nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := payload[key]; ok {
			if text := jsonValueSummary(raw); text != "" {
				return text
			}
		}
	}
	return ""
}

// CodeSizeFor shares the markdown code ladder with chat bodies.
func CodeSizeFor(st ToolCardStyle) float32 {
	size := st.FontSize - 1
	if size < 11 {
		return 11
	}
	return size
}
