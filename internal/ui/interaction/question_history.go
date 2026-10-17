package interaction

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/interaction"
	fonttheme "whaleisle/internal/ui/theme"
)

func questionHistory(c *ui.Context, records []interaction.QuestionHistoryView, store *interaction.Store, fg, muted, accent ui.Color) {
	if len(records) == 0 {
		return
	}
	labels := []string{"查看回答历史 / Review answer history"}
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
		label := "已记录回答（只读） / Recorded answers (read-only)"
		if v.Queued {
			label = "已排队回答（只读） / Queued answers (read-only)"
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
				text = "未作答 / Skipped"
			}
			ui.Text(c, text).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
		}
		if ui.Button(c, "关闭历史 / Close history").Clicked() {
			store.ReviewQuestionHistory(v.AgentID, "")
		}
	})
}
