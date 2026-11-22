package chat

import (
	"fmt"
	"regexp"
	"strconv"
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
		// .event cell — eventColumn wide; 36px turn gutter on the left with
		// the rail, the top-left #n turnLabel chip, and the icon-only
		// kindTag badge right-aligned in the kindSlot.
		ui.Box(c).Width(eventWidth).MinWidth(eventWidth).FillHeight().Children(func() {
			if row.TurnKey != "" {
				ui.Box(c).Absolute().Left(13).Top(1).Bottom(1).Width(1).Background(p.turnAccent)
			}
			if turnStart && row.TurnNumber > 0 {
				ui.Row(c).Absolute().Left(0).Top(0).Height(10).Padding(1, 5, 1, 5).Background(p.modulePlatform).Radius(2).Children(func() {
					ui.Text(c, fmt.Sprintf("#%d", row.TurnNumber)).FontSize(theme.InterfaceSize(8)).TextColor(p.caption).SingleLine()
				})
			}
			ui.Box(c).Absolute().Right(4).Top(7).Width(16).Height(16).Radius(4).Background(fill).Children(func() {
				ui.Icon(c, icon).Width(11).Height(11).TextColor(ink).Absolute().Left(2.5).Top(2.5)
			})
		})
		ui.Box(c).Width(4)
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
	case "system", "context":
		// The original titles the record, not the prompt text
		// (layout.ts → layout.initialSystemPrompt / systemPromptUpdated).
		if row.Kind == session.MessageKindSystemPrompt {
			if row.Source.EventType == "system-prompt-update" {
				return TrajectoryLocale("layout.systemPromptUpdated"), ""
			}
			return TrajectoryLocale("layout.initialSystemPrompt"), ""
		}
		return trajectoryRecordPreview(row), ""
	case "user":
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
			return trajectoryPlainText(value)
		}
	}
	return ""
}

var trajectoryLinkRe = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)

// trajectoryPlainText collapses preview markdown to the plain text the
// original renders — headings, block markers, fences, emphasis/code ticks
// and [text](url) links lose their syntax, underscores stay literal
// (inline_code renders as inline_code).
func trajectoryPlainText(value string) string {
	value = strings.ReplaceAll(value, "```", "")
	lines := make([]string, 0, 4)
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		line = strings.TrimSpace(strings.TrimPrefix(line, ">"))
		if len(line) > 2 && line[1] == ' ' && (line[0] == '-' || line[0] == '*' || line[0] == '+') {
			line = strings.TrimSpace(line[2:])
		} else if dot := strings.Index(line, ". "); dot > 0 && dot <= 3 {
			if _, err := strconv.Atoi(line[:dot]); err == nil {
				line = strings.TrimSpace(line[dot+2:])
			}
		}
		lines = append(lines, line)
	}
	joined := trajectoryLinkRe.ReplaceAllString(strings.Join(lines, " "), "$1")
	joined = strings.NewReplacer("**", "", "__", "", "~~", "", "`", "", "*", "").Replace(joined)
	return strings.Join(strings.Fields(joined), " ")
}

// Wide-window inspector follows the source clamp and keeps at least280DIP
// for the ledger. The narrow-window source variant overlays the ledger.
func trajectoryInspectorWidth(width float32) float32 {
	return min(max(320, width*.38), 440, max(0, width-280))
}
