package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Original TrajectoryTable uses one role column (84px in Chinese, 122px
// otherwise) and one content column, with 30px single-line record rows.
func paintTrajectoryRecord(c *ui.Context, row trajRow, fg, muted, accent ui.Color) {
	eventWidth := float32(84)
	if i18n.Locale() == i18n.LocaleEN {
		eventWidth = 122
	}
	ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(eventWidth).MinWidth(0).Padding(0, 8, 0, 8).Children(func() {
			ui.Text(c, trajectoryRecordRole(row)).FontSize(theme.InterfaceSize(12)).TextColor(accent).SingleLine().FillWidth().MinWidth(0)
		})
		ui.Row(c).Grow(1).MinWidth(0).Padding(0, 8, 0, 8).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, trajectoryRecordPreview(row)).FontSize(theme.InterfaceSize(12)).TextColor(fg).SingleLine().Grow(1).MinWidth(0)
			if row.Time != "" && row.Timing != nil {
				ui.Text(c, row.Time).FontSize(theme.InterfaceSize(11)).TextColor(muted).SingleLine()
			}
		})
	})
}

func trajectoryRecordRole(row trajRow) string {
	if row.Kind == trajectoryRequestKind {
		return trajectoryText("请求", "Request")
	}
	if row.Kind == session.MessageKindText {
		switch row.Role {
		case "user":
			return trajectoryText("用户", "User")
		case "assistant":
			return trajectoryText("助手", "Assistant")
		case "system":
			return trajectoryText("系统", "System")
		}
	}
	return kindLabelZH(row.Kind)
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
