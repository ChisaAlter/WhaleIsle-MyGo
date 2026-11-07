package interaction

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func questionHistoryAnsweredCountText(rows []interaction.QuestionHistoryRow) string {
	if len(rows) == 0 {
		return ""
	}
	answered := 0
	for _, r := range rows {
		if len(r.Selected) > 0 || strings.TrimSpace(r.Custom) != "" {
			answered++
		}
	}
	if i18n.Locale() == i18n.LocaleZH {
		return fmt.Sprintf("%d/%d 已回答", answered, len(rows))
	}
	return fmt.Sprintf("%d/%d answered", answered, len(rows))
}

func questionHistory(c *ui.Context, records []interaction.QuestionHistoryView, store CardStore, colors fonttheme.Colors, fg, muted, accent ui.Color) {
	if len(records) == 0 {
		return
	}
	labels := []string{i18n.T(i18n.KeyChatQuestionHistory)}
	selected := labels[0]
	var current *interaction.QuestionHistoryView
	for i, r := range records {
		label := r.Title + " [" + r.CallID + "]"
		labels = append(labels, label)
		if r.Open {
			selected = label
			current = &records[i]
		}
	}
	choice := ui.Select(c, &selected, labels).Tooltip(questionStatusText("reply.label"))
	if choice.Changed() {
		id := ""
		for i, label := range labels[1:] {
			if label == selected {
				id = records[i].CallID
				break
			}
		}
		store.ReviewQuestionHistory(records[0].AgentID, id)
	}
	if current == nil {
		return
	}
	v := *current
	ui.Column(c).FillWidth().Padding(8).Gap(6).Background(accent.Alpha(0.06)).Radius(12).Children(func() {
		label := i18n.T(i18n.KeyChatQuestionRecorded)
		if v.Queued {
			label = i18n.T(i18n.KeyChatQuestionQueued)
		}
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).Bold().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Grow(1)
			if summary := questionHistoryAnsweredCountText(v.Rows); summary != "" {
				ui.Text(c, summary).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
		})
		for _, r := range v.Rows {
			if r.Header != "" && r.Header != r.Question {
				ui.Text(c, r.Header).FontSize(fonttheme.InterfaceSize(11)).Bold().TextColor(muted)
			}

			ui.Text(c, r.Question).FontSize(fonttheme.InterfaceSize(12)).Bold().TextColor(fg)
			if r.Detail != "" {
				ui.Text(c, r.Detail).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
			}
			for i, option := range r.OptionLabels {
				marker := "○ "
				for _, sel := range r.Selected {
					if sel == option {
						marker = "✓ "
						break
					}
				}
				display := interaction.RecommendedQuestionLabel(option)
				recommended := (i < len(r.Recommended) && r.Recommended[i]) || display != option
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, marker+display).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
					if recommended {
						ui.Text(c, i18n.T(i18n.KeyChatQuestionRecommended)).FontSize(10).FixedLineHeight(16).FontWeight(600).TextColor(accent).Padding(0, 4).Radius(4).Background(accent.Alpha(.12))
					}
				})
				if i < len(r.OptionDescriptions) && r.OptionDescriptions[i] != "" {
					ui.Text(c, r.OptionDescriptions[i]).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			}
			var answer []string
			for _, sel := range r.Selected {
				answer = append(answer, interaction.RecommendedQuestionLabel(sel))
			}
			if r.Custom != "" {
				answer = append(answer, r.Custom)
			}
			text := strings.Join(answer, " · ")
			if text == "" {
				text = i18n.T(i18n.KeyChatQuestionSkipped)
			}
			ui.Text(c, text).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionCloseHist), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Tooltip(questionStatusText("reply.close")).Clicked() {
			store.ReviewQuestionHistory(v.AgentID, "")
		}
	})
}
