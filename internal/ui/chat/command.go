package chat

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// CommandCardView keeps the complete Host output available even when the
// compact summary is collapsed. It never interprets output as a tool/action.
func CommandCardView(c *ui.Context, msg session.Message, m Model, st ToolCardStyle) {
	command := msg.Command
	if command == nil {
		return
	}
	title := i18n.T(i18n.KeyCommandTitle)
	if command.Name != "" {
		title = "/" + command.Name
	}
	status, color := i18n.T(i18n.KeyCommandRunning), st.Accent
	switch command.State {
	case "success":
		status, color = i18n.T(i18n.KeyCommandDone), st.OK
	case "error":
		status, color = i18n.T(i18n.KeyCommandFailed), st.Err
	}
	summary := strings.SplitN(command.Text, "\n", 2)[0]
	if summary == "" {
		summary = status
	}
	runes := []rune(summary)
	if len(runes) > 160 {
		summary = string(runes[:160]) + "…"
	}
	key := m.Snapshot.SelectedID + ":" + msg.ID
	expanded := false
	if m.ThinkingExpanded != nil && *m.ThinkingExpanded != nil {
		expanded = (*m.ThinkingExpanded)[key]
	}
	hasDetails := strings.TrimSpace(command.Args) != "" || (command.Text != "" && command.Text != summary)
	ui.Column(c).Fill().Gap(6).Padding(8, 10).Background(st.Field).Border(1, st.Border).Radius(6).Children(func() {
		ui.Row(c).Fill().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(12).Bold().TextColor(st.FG)
			ui.Text(c, summary).FontSize(12).TextColor(color).Grow(1)
			if hasDetails && m.ThinkingExpanded != nil {
				label := i18n.T(i18n.KeyPlanBoardExpand)
				if expanded {
					label = i18n.T(i18n.KeyPlanBoardCollapse)
				}
				if ui.Button(c, label).FontSize(11).Clicked() {
					if *m.ThinkingExpanded == nil {
						*m.ThinkingExpanded = make(map[string]bool)
					}
					(*m.ThinkingExpanded)[key] = !expanded
				}
			}
			if command.Text != "" {
				if ui.Button(c, i18n.T(i18n.KeyChatCopy)).FontSize(11).Clicked() {
					mygo.Clipboard.WriteText(command.Text)
				}
			}
		})
		if expanded || (hasDetails && m.ThinkingExpanded == nil) {
			if strings.TrimSpace(command.Args) != "" {
				ui.Text(c, title+command.Args).FontSize(12).TextColor(st.Muted)
			}
			if command.Text != "" {
				ui.Text(c, command.Text).FontSize(12).TextColor(st.FG)
			}
		}
	})
}
