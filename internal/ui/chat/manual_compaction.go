package chat

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

// ManualCompactionCardView renders the one landed checkpoint and its command
// settlement together. Errors after a landed checkpoint remain visible; neither
// structured counts nor generated summary text can replace that error output.
func ManualCompactionCardView(c *ui.Context, msg session.Message, m Model, st ToolCardStyle) {
	command := msg.Command
	if command == nil || command.Compaction == nil {
		return
	}
	compact := command.Compaction
	status, statusColor := i18n.T(i18n.KeyCommandRunning), st.Accent
	switch command.State {
	case "success":
		status, statusColor = i18n.T(i18n.KeyCommandDone), st.OK
	case "error":
		status, statusColor = i18n.T(i18n.KeyCommandFailed), st.Err
	}
	label := i18n.T(i18n.KeyChatCompactionSummaryMissing)
	if compact.ShadowedItemCount != nil && compact.ShadowedTokenCount != nil {
		label = fmt.Sprintf(i18n.T(i18n.KeyChatCompactionCounts), *compact.ShadowedItemCount, *compact.ShadowedTokenCount)
	} else if command.Text != "" {
		label = strings.SplitN(command.Text, "\n", 2)[0]
	}
	runes := []rune(label)
	if len(runes) > 160 {
		label = string(runes[:160]) + "…"
	}
	key := m.Snapshot.SelectedID + ":" + msg.ID
	expanded := false
	if m.ThinkingExpanded != nil && *m.ThinkingExpanded != nil {
		expanded = (*m.ThinkingExpanded)[key]
	}
	hasDetails := compact.Summary != "" || strings.TrimSpace(command.Args) != "" ||
		(command.Text != "" && command.Text != label)
	ui.Column(c).Fill().Gap(6).Padding(8, 10).Background(st.Field).Border(1, st.Border).Radius(6).Children(func() {
		ui.Row(c).Fill().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatManualCompaction)).FontSize(12).Bold().TextColor(st.FG).Grow(1)
			ui.Text(c, status).FontSize(11).TextColor(statusColor)
			if hasDetails && m.ThinkingExpanded != nil {
				action := i18n.T(i18n.KeyPlanBoardExpand)
				if expanded {
					action = i18n.T(i18n.KeyPlanBoardCollapse)
				}
				if stdButton(c, st.Colors, action, buttonGhost, false).Clicked() {
					if *m.ThinkingExpanded == nil {
						*m.ThinkingExpanded = make(map[string]bool)
					}
					(*m.ThinkingExpanded)[key] = !expanded
				}
			}
			copyText := command.Text
			if compact.Summary != "" {
				if copyText != "" {
					copyText += "\n\n"
				}
				copyText += compact.Summary
			}
			if copyText != "" && stdButton(c, st.Colors, i18n.T(i18n.KeyChatCopy), buttonGhost, false).Clicked() {
				mygo.Clipboard.WriteText(copyText)
			}
		})
		ui.Text(c, label).FontSize(12).TextColor(st.Muted)
		// A failed save/cancel after replacement must not look like success.
		if command.State == "error" && command.Text != "" {
			ui.Text(c, command.Text).FontSize(12).TextColor(st.Err)
		}
		if expanded || (hasDetails && m.ThinkingExpanded == nil) {
			if strings.TrimSpace(command.Args) != "" {
				ui.Text(c, "/compact"+command.Args).FontSize(12).TextColor(st.Muted)
			}
			if command.State != "error" && command.Text != "" && command.Text != label {
				ui.Text(c, command.Text).FontSize(12).TextColor(st.FG)
			}
			if compact.Summary != "" {
				style := markdown.Style{FontSize: 12, FG: st.FG, Muted: st.Muted, CodeBG: st.Field, Accent: st.Accent}
				if m.OnOpenCitationURL != nil {
					markdown.ViewInteractive(c, compact.Summary, style, m.OnOpenCitationURL)
				} else {
					markdown.View(c, compact.Summary, style)
				}
			} else {
				ui.Text(c, i18n.T(i18n.KeyChatCompactionSummaryMissing)).FontSize(11).TextColor(st.Muted)
			}
		}
	})
}
