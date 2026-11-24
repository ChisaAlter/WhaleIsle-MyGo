package chat

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// CommandCardView keeps the complete Host output available even when the
// compact summary is collapsed. It never interprets output as a tool/action.
func CommandCardView(c *ui.Context, msg session.Message, m Model, st ToolCardStyle) {
	command := msg.Command
	if command == nil {
		return
	}
	if command.Compaction != nil {
		ManualCompactionCardView(c, msg, m, st)
		return
	}
	t := design.For(c)
	title := i18n.T(i18n.KeyCommandTitle)
	if command.Name != "" {
		title = "/" + command.Name
	}
	status, color := i18n.T(i18n.KeyCommandRunning), t.Primary
	switch command.State {
	case "success":
		status, color = i18n.T(i18n.KeyCommandDone), t.SuccessForeground
	case "error":
		status, color = i18n.T(i18n.KeyCommandFailed), t.DestructiveForeground
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
	chatCardSurface(c, ui.Column(c).Fill().Gap(design.Sp(1.5)).Padding(design.Sp(2), design.Sp(2.5)), design.RadiusSM).Children(func() {
		ui.Row(c).Fill().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(design.TextXS.Size).Bold().TextColor(t.CardForeground)
			ui.Text(c, summary).FontSize(design.TextXS.Size).TextColor(color).Grow(1)
			if hasDetails && m.ThinkingExpanded != nil {
				label := i18n.T(i18n.KeyPlanBoardExpand)
				if expanded {
					label = i18n.T(i18n.KeyPlanBoardCollapse)
				}
				if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
					if *m.ThinkingExpanded == nil {
						*m.ThinkingExpanded = make(map[string]bool)
					}
					(*m.ThinkingExpanded)[key] = !expanded
				}
			}
			if command.Text != "" {
				if coss.Button(c, i18n.T(i18n.KeyChatCopy), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS, Icon: coss.IconCopy}).Clicked() {
					mygo.Clipboard.WriteText(command.Text)
				}
			}
		})
		if expanded || (hasDetails && m.ThinkingExpanded == nil) {
			if strings.TrimSpace(command.Args) != "" {
				ui.Text(c, title+command.Args).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
			}
			if command.Text != "" {
				ui.Text(c, command.Text).FontSize(design.TextXS.Size).TextColor(t.CardForeground)
			}
		}
	})
}
