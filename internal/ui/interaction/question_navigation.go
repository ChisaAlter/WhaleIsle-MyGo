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
	frame := ui.Column(c.Key("question-wizard:" + p.AgentID + ":" + p.EventID)).FillWidth().Gap(6)
	flow := ui.Local(frame, "question-editor-flow", func() questionEditorFlow { return questionEditorFlow{} })
	index := min(max(card.Index, 0), len(card.Rows)-1)
	row := card.Rows[index]
	focused := false
	frame.Children(func() {
		ui.Row(c).FillWidth().Gap(8).Wrap().Children(func() {
			// The pager below owns the progress readout; the header keeps only the
			// minimize/review affordances.
			ui.Box(c).Grow(1).MinWidth(0)
			label := i18n.T(i18n.KeyChatQuestionMinimize)
			if flow.minimized {
				label = i18n.T(i18n.KeyChatQuestionExpand)
			}
			if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
				flow.minimized = !flow.minimized
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionReview), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: p.Sending}).Clicked() {
				flow.summary = !flow.summary
			}
		})
		if flow.summary {
			questionDraftSummary(c, p, card, store, colors, flow, muted)
		}
		if flow.minimized {
			return
		}
		focused = questionAnswerBody(c, p, row, index, store, flow, fg, muted)
		if flow.feedback != "" {
			ui.Text(c, questionFeedback(flow.feedback)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		}
		ui.Row(c).FillWidth().Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			if prev := questionPagerButton(c, questionPagerPrev, i18n.T(i18n.KeyChatQuestionPrev), !p.Sending && index > 0); prev.Clicked() {
				store.NavigateQuestion(p.AgentID, p.EventID, index-1)
				flow.feedback = ""
			}
			ui.Text(c, fmt.Sprintf("%d / %d", index+1, len(card.Rows))).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
			if skip := interactionGhostButton(c, colors, i18n.T(i18n.KeyChatQuestionSkip), p.Sending || p.Error != ""); skip.Clicked() {
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
			submit := interactionPrimaryButton(c, colors, label, p.Sending || p.Error != "" || !answered)
			if submit.Clicked() {
				flow.focusID, flow.feedback = store.ContinueQuestion(p.AgentID, p.EventID, row.ID)
			}
		})
	})
	return focused
}

// Optional draft recap is a navigation aid, never a mandatory submit gate.
// Settled/queued history remains the separate read-only recorded-answer view.
func questionDraftSummary(c *ui.Context, p interaction.Pending, card interaction.QuestionCardView, store CardStore, colors theme.Colors, flow *questionEditorFlow, muted ui.Color) {
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		for index, row := range card.Rows {
			label := row.Header
			if label == "" {
				label = row.Question
			}
			values := append([]string(nil), row.Selected...)
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
	if i18n.Locale() == i18n.LocaleZH {
		if status == "incomplete" {
			return "请回答或跳过其余问题后提交。"
		}
		return "请先回答此题再继续。"
	}
	if status == "incomplete" {
		return "Complete or skip the remaining questions before submitting."
	}
	return "Answer this question before continuing."
}
