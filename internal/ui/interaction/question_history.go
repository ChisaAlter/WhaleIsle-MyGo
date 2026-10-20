package interaction

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func questionHistory(c *ui.Context, records []interaction.QuestionHistoryView, store CardStore, fg, muted, accent ui.Color) {
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
	choice := ui.Select(c, &selected, labels)
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
		ui.Text(c, label).Bold().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
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
				for _, selected := range r.Selected {
					if selected == option {
						marker = "✓ "
						break
					}
				}
				ui.Text(c, marker+option).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
				if i < len(r.OptionDescriptions) && r.OptionDescriptions[i] != "" {
					ui.Text(c, r.OptionDescriptions[i]).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
			}
			answer := append([]string(nil), r.Selected...)
			if r.Custom != "" {
				answer = append(answer, r.Custom)
			}
			text := strings.Join(answer, " · ")
			if text == "" {
				text = i18n.T(i18n.KeyChatQuestionSkipped)
			}
			ui.Text(c, text).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if ui.Button(c, i18n.T(i18n.KeyChatQuestionCloseHist)).Clicked() {
			store.ReviewQuestionHistory(v.AgentID, "")
		}
	})
}
