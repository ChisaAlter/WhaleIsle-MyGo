package chat

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func QuestionReplyView(c *ui.Context, msg session.Message, scope string, palette theme.Colors, fg, muted ui.Color) {
	data := session.QuestionReplyDetails(msg)
	root := ui.Column(c.Key(fmt.Sprintf("question-reply:%s:%s:%g", scope, msg.ID, msg.Seq))).FillWidth().Gap(5)
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		if stdButton(c, palette, i18n.T(i18n.KeyChatQuestionEarlier), buttonGhost, false).Label("Question reply details " + msg.ID).Clicked() {
			*open = !*open
		}
		if !*open {
			values := []string{}
			for _, q := range data.Questions {
				values = append(values, data.AnswerValues(q.ID)...)
			}
			summary := strings.Join(values, ", ")
			if summary == "" {
				summary = i18n.T(i18n.KeyChatQuestionSkipped)
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
						answer := i18n.T(i18n.KeyChatQuestionSkipped)
						if len(values) > 0 {
							answer = strings.Join(values, ", ")
						}
						ui.Text(c, i18n.T(i18n.KeyChatQuestionAnswer) + ": "+answer).FontSize(12).TextColor(fg).Selectable()
					}
				})
			})
		}
		ui.Row(c).Gap(8).Children(func() {
			if data.Time > 0 {
				ui.Text(c, formatMessageClock(data.Time, c.Now())).FontSize(11).TextColor(muted)
			}
			if stdButton(c, palette, i18n.T(i18n.KeyChatQuestionCopyReply), buttonGhost, false).Label("Copy question reply " + msg.ID).Clicked() {
				c.WriteClipboard(questionReplyClipboard(data))
			}
		})
	})
}

// questionReplyClipboard keeps the copied reply in the active locale. The store
// projection still carries legacy bilingual text, so the UI owns the copy
// surface it renders.
func questionReplyClipboard(v session.QuestionReplyView) string {
	if len(v.Questions) == 0 {
		return v.Text
	}
	blocks := make([]string, 0, len(v.Questions))
	for _, q := range v.Questions {
		heading := q.Question
		if q.Header != "" && q.Header != q.Question {
			heading = q.Header + " — " + heading
		}
		answer := i18n.T(i18n.KeyChatQuestionSkipped)
		if values := v.AnswerValues(q.ID); len(values) > 0 {
			answer = i18n.T(i18n.KeyChatQuestionAnswer) + ": " + strings.Join(values, ", ")
		}
		blocks = append(blocks, heading+"\n"+answer)
	}
	return strings.Join(blocks, "\n\n")
}
