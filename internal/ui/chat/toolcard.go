package chat

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/dsh/events"
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
			info.Detail = truncateToolText(ptcDetail(ev.Data), 200)
			if len(info.Images) > 0 {
				info.Detail = toolImageText(ev.Data.Content, true)
			}
		}
	default:
		info.Name = strings.TrimPrefix(eventType, "tool/")
		info.Detail = truncateToolText(extractLooseText(raw), 160)
	}
	if info.Name == "" {
		info.Name = "tool"
	}
	return info
}

// ToolCardView renders a bordered compact tool card with a kind chip.
func ToolCardView(c *ui.Context, eventType string, raw json.RawMessage, st ToolCardStyle, gallery ...ToolImageGalleryOptions) {
	info := ParseToolCard(eventType, raw)
	kind, chip := ClassifyToolName(info.Name)
	size := st.FontSize
	if size <= 0 {
		size = 12
	}
	statusColor := st.Muted
	switch info.Status {
	case "ok":
		statusColor = st.OK
	case "error", "failed":
		statusColor = st.Err
	case "call", "running":
		statusColor = st.Accent
	}
	chipBG := st.Border
	switch kind {
	case ToolKindBash:
		chipBG = ui.RGB(45, 55, 72)
	case ToolKindRead:
		chipBG = ui.RGB(49, 70, 89)
	case ToolKindWrite:
		chipBG = ui.RGB(70, 55, 40)
	case ToolKindSearch:
		chipBG = ui.RGB(55, 60, 90)
	case ToolKindWeb:
		chipBG = ui.RGB(40, 70, 70)
	case ToolKindTodo:
		chipBG = ui.RGB(55, 75, 55)
	case ToolKindAsk:
		chipBG = ui.RGB(75, 60, 40)
	case ToolKindSubagent:
		chipBG = ui.RGB(65, 50, 80)
	}
	ui.Column(c).Padding(8, 10, 8, 10).Gap(4).Fill().Background(st.Field).Border(1, st.Border).Radius(6).Children(func() {
		ui.Row(c).Gap(8).Fill().AlignItems(ui.Center).Children(func() {
			ui.Text(c, chip).FontSize(size-2).Bold().TextColor(st.FG).Background(chipBG).Padding(2, 6, 2, 6).Radius(4)
			ui.Text(c, info.Name).FontSize(size).Bold().TextColor(st.FG).Grow(1)
			ui.Text(c, info.Status).FontSize(size - 1).TextColor(statusColor)
		})
		if len(info.Images) > 0 && len(gallery) > 0 {
			toolImageGallery(c, info.Images, st, gallery[0])
		}
		if len(gallery) > 0 {
			markdown.LinkActions(c, info.Links, markdown.Style{FontSize: size, Accent: st.Accent}, gallery[0].OnOpenURL)
		}
		if info.Detail != "" {
			ui.Text(c, info.Detail).FontSize(size - 1).TextColor(st.Muted)
		}
	})
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
