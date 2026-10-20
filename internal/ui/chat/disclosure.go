package chat

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/widgets"
)

// Conversation row chrome. The shared widgets.DisclosureRow owns the 24px seat,
// chevron turn, hover fill, focus ring and body animation; this file only maps
// the transcript's data onto it.

// disclosureStyle carries the surface palette for row chrome.
type disclosureStyle struct {
	FG     ui.Color
	Muted  ui.Color
	Hover  ui.Color
	Accent ui.Color
	Size   float32
}

func (s disclosureStyle) size() float32 {
	if s.Size <= 0 {
		return 13
	}
	return s.Size
}

// rowStatus is the five-state status language shared by tool, turn and goal
// rows, mapped onto widgets.State.
type rowStatus int

const (
	statusNone rowStatus = iota
	statusIdle
	statusRunning
	statusDone
	statusWarning
	statusError
)

// widgetState maps the transcript status onto the shared state dot.
func widgetState(status rowStatus) widgets.State {
	switch status {
	case statusRunning:
		return widgets.StateRunning
	case statusDone:
		return widgets.StateDone
	case statusWarning:
		return widgets.StateWarning
	case statusError:
		return widgets.StateError
	default:
		return widgets.StateIdle
	}
}

func statusLabel(status rowStatus) string {
	switch status {
	case statusRunning:
		return "running"
	case statusDone:
		return "done"
	case statusWarning:
		return "warning"
	case statusError:
		return "error"
	default:
		return "idle"
	}
}

// disclosureRow paints one interactive row. The whole row is the hit target
// and the caller owns the open state, as the Original DisclosureRow does.
type disclosureRow struct {
	Key        string
	Icon       *ui.SVG
	IconColor  ui.Color
	Title      string
	TitleColor ui.Color
	Summary    string
	Trailing   string
	Status     rowStatus
	Open       bool
	Expandable bool
	Running    bool
	OnToggle   func()
}

func paintDisclosureRow(c *ui.Context, d disclosureRow, st disclosureStyle) {
	titleColor := d.TitleColor
	if titleColor.A == 0 {
		titleColor = st.FG
	}
	summaryColor := st.Muted
	options := widgets.DisclosureRowOptions{
		Icon:         d.Icon,
		Title:        d.Title,
		Summary:      d.Summary,
		Size:         st.size(),
		Color:        titleColor,
		SummaryColor: summaryColor,
		Running:      d.Running,
		OnToggle:     func(bool) { d.OnToggle() },
	}
	if d.Status != statusNone {
		options.Status = statusLabel(d.Status)
		state := widgetState(d.Status)
		options.Leading = func() { widgets.StateDot(c, state) }
	}
	if d.Trailing != "" {
		trailing := d.Trailing
		options.Trailing = func() {
			ui.Text(c, trailing).FontSize(st.size() - 1).TextColor(summaryColor).NoWrap()
		}
	}
	if d.Summary != "" {
		// The whole line reads as one announcement: title then summary.
		options.Label = d.Title + " " + d.Summary
	}
	if d.Expandable && d.OnToggle != nil {
		open := d.Open
		options.Open = &open
	}
	root := widgets.DisclosureRow(c, options)
	if d.Key != "" {
		root.Key("row:" + d.Key)
	}
}
