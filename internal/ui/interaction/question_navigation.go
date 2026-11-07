package interaction

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func questionsBody(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store CardStore, colors theme.Colors, fg, muted ui.Color) bool {
	if len(card.Rows) == 0 {
		return false
	}
	frame := ui.Column(c.Key("question-wizard:" + p.AgentID + ":" + p.EventID)).FillWidth().MinHeight(0).Shrink(1).Gap(6)
	flow := ui.Local(frame, "question-editor-flow", func() questionEditorFlow { return questionEditorFlow{} })
	index := min(max(card.Index, 0), len(card.Rows)-1)
	row := card.Rows[index]
	focused := false
	frame.Children(func() {
		ui.Row(c).FillWidth().Shrink(0).Gap(8).Wrap().Children(func() {
			// The pager below owns the progress readout; the header keeps only the
			// minimize/review affordances.
			ui.Box(c).Grow(1).MinWidth(0)
			label := i18n.T(i18n.KeyChatQuestionMinimize)
			tooltip := questionStatusText("nav.minimize")
			if flow.minimized {
				label = i18n.T(i18n.KeyChatQuestionExpand)
				tooltip = questionStatusText("nav.maximize")
			}
			if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Tooltip(tooltip).Clicked() {
				flow.minimized = !flow.minimized
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionReview), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
				flow.summary = !flow.summary
			}
		})
		if flow.summary {
			ui.Scroll(c).FillWidth().MinHeight(0).Shrink(1).Children(func() { questionDraftSummary(c, p, card, store, colors, flow, muted) })
		}
		if flow.minimized {
			return
		}
		focused = questionAnswerBody(c, p, row, index, store, flow, fg, muted)
		if flow.feedback != "" {
			ui.Text(c, questionFeedback(flow.feedback)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Row(c).FillWidth().Shrink(0).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			if prev := questionPagerButton(c, questionPagerPrev, i18n.T(i18n.KeyChatQuestionPrev), !p.Sending && index > 0); prev.Clicked() {
				store.NavigateQuestion(p.AgentID, p.EventID, index-1)
				flow.feedback = ""
			}
			ui.Text(c, fmt.Sprintf("%d / %d", index+1, len(card.Rows))).FontSize(theme.InterfaceSize(14)).FontWeight(500).TextColor(muted)
			if next := questionPagerButton(c, questionPagerNext, i18n.T(i18n.KeyChatQuestionNext), !p.Sending && index+1 < len(card.Rows)); next.Clicked() {
				store.NavigateQuestion(p.AgentID, p.EventID, index+1)
				flow.feedback = ""
			}
			if row.Skipped {
				ui.Text(c, i18n.T(i18n.KeyChatQuestionSkipped)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				if widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionUndoSkip), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
					store.SetQuestionSkipped(p.EventID, row.ID, false)
					flow.feedback = ""
				}
			}
			ui.Box(c).Grow(1).MinWidth(0)
			if skip := interactionGhostButton(c, colors, i18n.T(i18n.KeyChatQuestionSkip), p.Sending); skip.Clicked() {
				flow.focusID, flow.feedback = store.SkipQuestionAndContinue(p.AgentID, p.EventID, row.ID)
			}
			label := i18n.T(i18n.KeyChatQuestionStep)
			if index+1 == len(card.Rows) {
				label = i18n.T(i18n.KeyChatQuestionSubmit)
				if p.Continued {
					label = questionStatusText("submit")
				}
			}
			answered := len(row.Selected) > 0 || strings.TrimSpace(row.Custom) != ""
			submit := interactionPrimaryButton(c, colors, label, p.Sending || !answered)
			if submit.Clicked() {
				flow.focusID, flow.feedback = store.ContinueQuestion(p.AgentID, p.EventID, row.ID)
			}
		})
	})
	return focused
}

func questionAnsweredCountText(rows []interaction.QuestionRowView) string {
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

// Optional draft recap is a navigation aid, never a mandatory submit gate.
// Settled/queued history remains the separate read-only recorded-answer view.
func questionDraftSummary(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store CardStore, colors theme.Colors, flow *questionEditorFlow, muted ui.Color) {
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		if summary := questionAnsweredCountText(card.Rows); summary != "" {
			ui.Text(c, summary).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		for index, row := range card.Rows {
			label := row.Header
			if label == "" {
				label = row.Question
			}
			values := make([]string, 0, len(row.Selected)+1)
			for _, sel := range row.Selected {
				values = append(values, interaction.RecommendedQuestionLabel(sel))
			}
			if row.Custom != "" {
				values = append(values, row.Custom)
			}
			answer := strings.Join(values, " · ")
			if answer == "" {
				answer = i18n.T(i18n.KeyChatQuestionUnanswer)
				if row.Skipped {
					answer = i18n.T(i18n.KeyChatQuestionSkipped)
				}
			}
			ui.Text(c, fmt.Sprintf("%d. %s: %s", index+1, label, answer)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			if widgets.Button(c, colors, fmt.Sprintf(i18n.T(i18n.KeyChatQuestionReviewN), index+1), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
				if store.NavigateQuestion(p.AgentID, p.EventID, index) {
					flow.summary, flow.minimized = false, false
					flow.feedback = ""
				}
			}
		}
	})
}

func questionFeedback(status string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	switch status {
	case "incomplete":
		if zh {
			return "请先完成这道问题。"
		}
		return "Complete or skip the remaining questions before submitting."
	case "unavailable":
		if zh {
			return "当前无法提交，请稍候再试。"
		}
		return "Cannot submit right now; try again in a moment."
	case "resubmit":
		if zh {
			return "回答未送达，工作已继续，请再提交一次。"
		}
		return "The answer did not arrive before work continued; submit it again."
	case "sent":
		if zh {
			return "回答已发送；面板未能关闭。"
		}
		return "Reply sent; the panel could not close."
	default:
		if zh {
			return "请选择一个选项或填写自定义答案。"
		}
		return "Answer this question before continuing."
	}
}
