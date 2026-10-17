package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
)

func QuestionReplyView(c *ui.Context, msg session.Message, scope string, fg, muted ui.Color) {
	data := session.QuestionReplyDetails(msg)
	root := ui.Column(c.Key(fmt.Sprintf("question-reply:%s:%s:%g", scope, msg.ID, msg.Seq))).FillWidth().Gap(5)
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		if ui.Button(c, "之前问题的回复 / Reply to earlier questions").Label("Question reply details " + msg.ID).Clicked() {
			*open = !*open
		}
		if !*open {
			values := []string{}
			for _, q := range data.Questions {
				values = append(values, data.AnswerValues(q.ID)...)
			}
			summary := strings.Join(values, ", ")
			if summary == "" {
				summary = "未作答 / Skipped"
				if len(data.Questions) == 0 {
					summary = msg.Body
				}
			}
			ui.Text(c, clipRunes(summary, 160)).FontSize(12).TextColor(muted)
		} else {
			ui.Scroll(c).FillWidth().MaxHeight(320).Children(func() {
				ui.Column(c).FillWidth().Gap(5).Children(func() {
					if len(data.Questions) == 0 {
						ui.Text(c, data.Text).FontSize(12).TextColor(fg).Selectable()
					}
					for _, q := range data.Questions {
						if q.Header != "" && q.Header != q.Question {
							ui.Text(c, q.Header).FontSize(11).Bold().TextColor(muted)
						}
						ui.Text(c, q.Question).FontSize(12).Bold().TextColor(fg)
						if q.Detail != "" {
							ui.Text(c, q.Detail).FontSize(12).TextColor(muted)
						}
						for _, option := range q.Options {
							text := option.Label
							if option.Description != "" {
								text += " — " + option.Description
							}
							ui.Text(c, text).FontSize(12).TextColor(muted)
						}
						values := data.AnswerValues(q.ID)
						answer := "未作答 / Skipped"
						if len(values) > 0 {
							answer = strings.Join(values, ", ")
						}
						ui.Text(c, "回答 / Answer: "+answer).FontSize(12).TextColor(fg).Selectable()
					}
				})
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if data.Time > 0 {
				ui.Text(c, formatEventTime(data.Time)).FontSize(11).TextColor(muted)
			}
			if ui.Button(c, "复制回复 / Copy reply").Label("Copy question reply " + msg.ID).Clicked() {
				c.WriteClipboard(data.ClipboardText())
			}
		})
	})
}
