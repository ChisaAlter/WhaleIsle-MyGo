package chat

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/events"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// ToolCardStyle colors for compact tool rows.
type ToolCardStyle struct {
	FG       ui.Color
	Muted    ui.Color
	Border   ui.Color
	Field    ui.Color
	Accent   ui.Color
	OK       ui.Color
	Err      ui.Color
	FontSize float32
}

// ToolCardInfo is parsed display data for one persistence tool event.
type ToolCardInfo struct {
	Name   string
	Status string
	Detail string
	Images []string
	Links  []markdown.Link
	// Arguments and Output keep the full payloads for the expanded body. Detail
	// stays the one-line summary the collapsed row uses.
	Arguments string
	Output    string
}

// ParseToolCard extracts name, status, and truncated args/result from raw JSON.
func ParseToolCard(eventType string, raw json.RawMessage) ToolCardInfo {
	info := ToolCardInfo{Name: eventType, Status: "event"}
	if len(raw) == 0 {
		return info
	}
	switch eventType {
	case events.TypeToolCall:
		var ev events.ToolCall
		if json.Unmarshal(raw, &ev) == nil {
			info.Name = ev.Data.Name
			if info.Name == "" {
				info.Name = "tool"
			}
			info.Status = "call"
			info.Links = markdown.Links(ev.Data.Arguments)
			info.Arguments = ev.Data.Arguments
			info.Detail = truncateToolText(ev.Data.Arguments, 160)
		}
	case events.TypeToolResult:
		var ev events.ToolResult
		if json.Unmarshal(raw, &ev) == nil {
			info.Name = toolResultName(ev)
			info.Status = toolResultStatus(ev)
			info.Images = toolImageReferences(ev.Data.Message.Content)
			if info.Status == "error" {
				info.Images = nil
			}
			info.Links = markdown.Links(toolResultText(ev))
			info.Output = toolResultText(ev)
			info.Detail = truncateToolText(toolResultText(ev), 200)
			if len(info.Images) > 0 {
				info.Detail = toolResultText(ev)
			}
		}
	case events.TypeToolPtcDispatchStart:
		var ev events.ToolPtcDispatchStart
		if json.Unmarshal(raw, &ev) == nil {
			info.Name = ev.Data.Name
			info.Status = "running"
			info.Arguments = mustJSON(ev.Data.Arguments)
			info.Detail = truncateToolText(mustJSON(ev.Data.Arguments), 160)
		}
	case events.TypeToolPtcDispatch:
		var ev events.ToolPtcDispatch
		if json.Unmarshal(raw, &ev) == nil {
			info.Name = ev.Data.Name
			info.Status = ptcStatus(ev.Data.IsError)
			if !ev.Data.IsError {
				info.Images = toolImageReferences(ev.Data.Content)
			}
			info.Links = markdown.Links(ptcDetail(ev.Data))
			info.Output = ptcDetail(ev.Data)
			info.Detail = truncateToolText(ptcDetail(ev.Data), 200)
			if len(info.Images) > 0 {
				info.Detail = toolImageText(ev.Data.Content, true)
			}
		}
	default:
		info.Name = strings.TrimPrefix(eventType, "tool/")
		info.Detail = truncateToolText(extractLooseText(raw), 160)
		info.Output = extractLooseText(raw)
	}
	if info.Name == "" {
		info.Name = "tool"
	}
	return info
}

// ToolCardView renders a bordered compact tool card with a kind chip.
func ToolCardView(c *ui.Context, eventType string, raw json.RawMessage, st ToolCardStyle, gallery ...ToolImageGalleryOptions) {
	toolCardInfoView(c, ParseToolCard(eventType, raw), st, gallery...)
}

// toolCardInfoView paints one tool call as the Original ToolRow does: a single
// 24px disclosure line (kind icon, title, argument summary, status dot,
// chevron) whose structured body mounts only when opened.
func toolCardInfoView(c *ui.Context, info ToolCardInfo, st ToolCardStyle, gallery ...ToolImageGalleryOptions) {
	toolCardInfoViewExtra(c, info, st, nil, gallery...)
}

// toolCardInfoViewExtra adds owner content (diff/file cards, inspect action)
// inside the disclosure body, so a collapsed row keeps one line.
func toolCardInfoViewExtra(c *ui.Context, info ToolCardInfo, st ToolCardStyle, extra func(), gallery ...ToolImageGalleryOptions) {
	opts := ToolImageGalleryOptions{}
	if len(gallery) > 0 {
		opts = gallery[0]
	}
	kind, _ := ClassifyToolName(info.Name)
	size := st.FontSize
	if size <= 0 {
		size = 12
	}
	key := opts.ID
	if key == "" {
		key = info.Name + ":" + info.Status + ":" + info.Detail
	}
	summary := toolArgSummary(info.Arguments, info.Name)
	if summary == "" {
		summary = toolArgSummary(info.Detail, info.Name)
	}
	if toolRowStatus(info.Status) == statusError {
		if line := firstDetailLine(info.Detail); line != "" {
			summary = line
		}
	}
	body := func() {
		if len(info.Images) > 0 && !opts.TextOnly {
			toolImageGallery(c, info.Images, st, opts)
		}
		if !opts.TextOnly {
			markdown.LinkActions(c, info.Links, markdown.Style{FontSize: size, Accent: st.Accent}, opts.OnOpenURL)
		}
		if extra != nil {
			extra()
		}
		handled, ownsCopy := toolStructuredBody(c, kind, info, st, key)
		if !handled {
			if info.Arguments != "" {
				toolLifecycleDetail(c, key+":args", i18n.T(i18n.KeyChatToolArguments), prettyToolPayload(info.Arguments), st, nil, opts.TextOnly)
			}
			if info.Output != "" {
				toolLifecycleDetail(c, key+":result", i18n.T(i18n.KeyChatToolResult), info.Output, st, nil, opts.TextOnly)
			}
		}
		// One copy seat keeps the whole call (arguments + output) reachable, as
		// the Original detail cards expose their own clipboard action. The
		// terminal seat already carries its own, so it does not get a second.
		if payload := toolCopyPayload(info.Arguments, info.Output); payload != "" && !ownsCopy {
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				copySeat := ui.Button(c, i18n.T(i18n.KeyChatToolCopy)).FontSize(10).Label(i18n.T(i18n.KeyChatToolCopy)).
					Height(22).Padding(0, 8, 0, 8).Radius(6).Border(0.5, st.Border).
					Background(ui.Transparent).TextColor(st.Muted)
				if copySeat.Clicked() {
					if opts.TextOnly {
						c.Services().WriteClipboard(payload)
					} else {
						c.WriteClipboard(payload)
					}
				}
			})
		}
	}
	paintToolRow(c, kind, toolKindTitle(kind), summary, toolRowStatus(info.Status), st, key, opts.Expanded, body)
}

// toolCopyPayload joins the call evidence for the copy action.
func toolCopyPayload(arguments, output string) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(arguments) != "" {
		parts = append(parts, arguments)
	}
	if strings.TrimSpace(output) != "" {
		parts = append(parts, output)
	}
	return strings.Join(parts, "\n")
}

func firstDetailLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return clipRunes(collapseSpace(trimmed), toolRowSummaryRunes)
		}
	}
	return ""
}

func toolResultName(ev events.ToolResult) string {
	if ev.Data.Message.ToolCallId != "" {
		return ev.Data.Message.ToolCallId
	}
	return "result"
}

func toolResultStatus(ev events.ToolResult) string {
	if ev.Data.Error != nil || (ev.Data.Message.IsError != nil && *ev.Data.Message.IsError) {
		return "error"
	}
	return "ok"
}

func toolResultText(ev events.ToolResult) string {
	parts := []string{toolImageText(ev.Data.Message.Content, len(toolImageReferences(ev.Data.Message.Content)) > 0 && toolResultStatus(ev) != "error")}
	if ev.Data.Error != nil && ev.Data.Error.Reason != nil {
		parts = append(parts, *ev.Data.Error.Reason)
	}
	return strings.Join(parts, "\n")
}

func ptcStatus(isErr bool) string {
	if isErr {
		return "error"
	}
	return "ok"
}

func ptcDetail(d events.ToolPtcDispatchData) string {
	if d.IsError && d.Error != nil && d.Error.Reason != nil {
		return *d.Error.Reason
	}
	if len(d.Content) > 0 {
		return toolImageText(d.Content, !d.IsError && len(toolImageReferences(d.Content)) > 0)
	}
	return mustJSON(d.Arguments)
}

func extractLooseText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	for _, key := range []string{"text", "content", "message", "arguments"} {
		if v, ok := obj[key]; ok {
			if t := extractLooseText(v); t != "" {
				return t
			}
		}
	}
	return ""
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func truncateToolText(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// DefaultToolCardStyle builds colors from theme chrome helpers.
func DefaultToolCardStyle(fg, muted, border, field, accent ui.Color) ToolCardStyle {
	return ToolCardStyle{
		FG:       fg,
		Muted:    muted,
		Border:   border,
		Field:    field,
		Accent:   accent,
		OK:       ui.RGB(61, 153, 112),
		Err:      ui.RGB(224, 102, 102),
		FontSize: 12,
	}
}
