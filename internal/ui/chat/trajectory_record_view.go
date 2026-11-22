package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// paintTrajectoryRecord renders one TrajectoryTable row: the fixed event
// gutter (turn rail + #n marker), a colored kindTag badge (icon + label),
// then the title with muted inline detail and the `→ result` continuation
// for tool calls (TrajectoryTable.tsx/TrajectoryCell.tsx).
func paintTrajectoryRecord(c *ui.Context, row trajRow, p trajectoryPalette, turnStart bool) {
	eventWidth := float32(84)
	if i18n.Locale() == i18n.LocaleEN {
		eventWidth = 122
	}
	kind := trajCellKind(row)
	icon, ink, fill := trajKindBadge(p, kind)
	ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Children(func() {
		// Turn gutter — the original's .turn rail and .turnLabel.
		ui.Box(c).Width(28).FillHeight().Children(func() {
			if row.TurnKey != "" {
				ui.Box(c).Absolute().Left(12).Top(2).Bottom(2).Width(2).Background(p.turnAccent)
			}
			if turnStart && row.TurnNumber > 0 {
				ui.Text(c, fmt.Sprintf("#%d", row.TurnNumber)).FontSize(theme.InterfaceSize(10)).TextColor(p.caption).SingleLine().Absolute().Left(16).Top(8)
			}
		})
		// .event column — 84px zh / 122px en — carries the icon-only kindTag
		// badge (the original renders the 16px tinted glyph, no label).
		ui.Box(c).Width(eventWidth).MinWidth(eventWidth).Children(func() {
			ui.Box(c).Width(16).Height(16).Radius(4).Background(fill).Children(func() {
				ui.Icon(c, icon).Width(11).Height(11).TextColor(ink).Absolute().Left(2.5).Top(2.5)
			})
		})
		ui.Box(c).Width(8)
		title, detail := trajectoryRecordTitle(row, kind)
		ui.Text(c, title).FontSize(theme.InterfaceSize(12)).FontWeight(500).TextColor(p.fg).SingleLine()
		if detail != "" {
			ui.Text(c, " "+detail).FontSize(theme.InterfaceSize(12)).TextColor(p.muted).SingleLine().Grow(1).MinWidth(0)
		}
		if row.Result != "" {
			ui.Text(c, " → "+row.Result).FontSize(theme.InterfaceSize(12)).TextColor(p.muted).SingleLine().Grow(1).MinWidth(0)
		}
		if detail == "" && row.Result == "" {
			ui.Box(c).Grow(1).MinWidth(0)
		}
		if row.Time != "" && row.Timing != nil && (kind == "message" || kind == "tool") {
			ui.Text(c, row.Time).FontSize(theme.InterfaceSize(11)).TextColor(p.caption).SingleLine().Padding(0, 8, 0, 0)
		}
	})
}

// trajectoryRecordTitle splits the row's display text into the bold-ish
// lead and the muted inline detail the original paints side by side.
func trajectoryRecordTitle(row trajRow, kind string) (string, string) {
	switch kind {
	case "tool":
		name := ""
		if row.Source.Tool != nil {
			name = row.Source.Tool.Name
		}
		if name == "" {
			fields := strings.Fields(row.Tools)
			if len(fields) > 0 {
				name = fields[0]
			}
		}
		if name == "" {
			name = trajectoryText("工具调用", "Tool call")
		}
		detail := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(row.Tools), name))
		if detail == "" {
			detail = row.Input
		}
		return name, detail
	case "user", "system", "context":
		return trajectoryRecordPreview(row), ""
	default:
		return trajectoryRecordPreview(row), ""
	}
}

func trajectoryRecordPreview(row trajRow) string {
	values := []string{row.Output, row.Input, row.Think, row.Tools}
	switch row.Kind {
	case trajectoryRequestKind:
		values = []string{row.Input, row.Model}
	case session.MessageKindTool:
		values = []string{row.Tools, row.Input, row.Output}
	case session.MessageKindThinking:
		values = []string{row.Think, row.Input}
	}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.Join(strings.Fields(value), " ")
		}
	}
	return ""
}

// Wide-window inspector follows the source clamp and keeps at least280DIP
// for the ledger. The narrow-window source variant overlays the ledger.
func trajectoryInspectorWidth(width float32) float32 {
	return min(max(320, width*.38), 440, max(0, width-280))
}
