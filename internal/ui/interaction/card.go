// Package interaction renders pending Host waterfall prompts above the composer.
package interaction

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the card to the interaction store snapshot.
type Model struct {
	Snapshot   interaction.ViewState
	ThemeID    string
	SystemDark bool
	Store      *interaction.Store
	// OnAutoAllowChanged is optional persistence when the user toggles auto-allow.
	OnAutoAllowChanged func(on bool)
}

// View paints auto-allow banner and pending approval / question cards.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	snap := m.Snapshot
	// Idle in manual mode: omit strip. Auto-allow on → banner (toggle to manual).
	// Pending → approval / question cards.
	if snap.Banner == "" && len(snap.Pending) == 0 && snap.QuestionStatus == "" && !snap.AutoAllow {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	// Parent frame already constrains width to composerCardW; keep vertical pad only.
	ui.Column(c).Background(field).Padding(8, 0, 0, 0).Gap(8).Children(func() {
		ui.Box(c).Fill().Height(1).Background(border)
		if snap.AutoAllow {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				msg := i18n.T(i18n.KeyInteractionAutoAllowOn)
				if snap.Banner != "" {
					msg = i18n.T(i18n.KeyInteractionAutoAllowed) + ": " + snap.Banner
				}
				ui.Text(c, msg).FontSize(12).TextColor(muted).Grow(1)
				if ui.Button(c, i18n.T(i18n.KeyInteractionManual)).Clicked() {
					m.Store.SetAutoAllow(false)
					if m.OnAutoAllowChanged != nil {
						m.OnAutoAllowChanged(false)
					}
				}
			})
		}
		if snap.QuestionStatus != "" {
			text := questionStatusText(snap.QuestionStatus)
			ui.Text(c, text).FontSize(12).TextColor(muted)
		}
		for _, p := range snap.Pending {
			pendingCard(c, p, snap, m.Store, m.OnAutoAllowChanged, fg, muted, accent)
		}
	})
}

func pendingCard(c *ui.Context, p interaction.Pending, snap interaction.ViewState, store *interaction.Store, onAuto func(bool), fg, muted, accent ui.Color) {
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.PlanReview != nil {
		planReviewCard(c, p, card, store, fg, muted, accent)
		return
	}
	ui.Column(c).Padding(8, 8, 8, 8).Gap(6).Background(accent.Alpha(0.08)).Radius(12).Children(func() {
		ui.Text(c, p.Title).FontSize(13).Bold().TextColor(fg)
		if p.Event == interaction.EventUserQuestions {
			if p.Continued {
				ui.Text(c, questionStatusText("continued")).FontSize(11).TextColor(muted)
			}
			if card, ok := snap.QuestionCards[p.EventID]; ok {
				questionsBody(c, p.EventID, card, store, fg, muted)
			}
		} else {
			if p.Detail != "" {
				ui.Text(c, p.Detail).FontSize(12).TextColor(muted)
			}
			if p.CallID != "" {
				ui.Text(c, i18n.T(i18n.KeyInteractionCallID)+": "+p.CallID).FontSize(11).TextColor(muted)
			}
			if p.Args != "" {
				ui.Column(c).Padding(6, 8, 6, 8).Background(accent.Alpha(0.06)).Gap(4).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, p.Args).FontSize(11).TextColor(fg).Grow(1)
						if ui.Button(c, i18n.T(i18n.KeyInteractionCopyArgs)).FontSize(11).Clicked() {
							if strings.TrimSpace(p.Args) != "" {
								mygo.Clipboard.WriteText(p.Args)
							}
						}
					})
				})
			}
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FontSize(12).TextColor(muted)
		}
		ui.Row(c).Gap(8).Children(func() {
			if p.Sending {
				ui.Button(c, i18n.T(i18n.KeyInteractionContinue)).Disabled(true)
				return
			}
			if p.Error != "" {
				if ui.Button(c, i18n.T(i18n.KeyBootRetry)).Clicked() {
					store.Retry(p.EventID)
				}
				return
			}
			switch p.Event {
			case interaction.EventUserQuestions:
				canSubmit := store.QuestionsCanSubmit(p.EventID)
				label := i18n.T(i18n.KeyInteractionContinue)
				if p.Continued {
					label = questionStatusText("submit")
				}
				btn := ui.Button(c, label)
				if !canSubmit {
					btn = btn.Disabled(true)
				}
				if btn.Clicked() && canSubmit {
					store.ResolveQuestions(p.EventID)
				}
				if !p.Continued {
					if ui.Button(c, i18n.T(i18n.KeyInteractionDeny)).Clicked() {
						store.ResolveDeny(p.EventID)
					}
				}
			default:
				if ui.Button(c, i18n.T(i18n.KeyInteractionAllow)).Clicked() {
					store.ResolveAllow(p.EventID)
				}
				if ui.Button(c, i18n.T(i18n.KeyInteractionDeny)).Clicked() {
					store.ResolveDeny(p.EventID)
				}
			}
		})
	})
}

func questionsBody(c *ui.Context, eventID string, card interaction.QuestionCardView, store *interaction.Store, fg, muted ui.Color) {
	for _, row := range card.Rows {
		ui.Column(c).Gap(4).Children(func() {
			label := row.Question
			if label == "" {
				label = row.Header
			}
			if label != "" {
				ui.Text(c, label).FontSize(12).TextColor(fg)
			}
			if row.Detail != "" {
				ui.Text(c, row.Detail).FontSize(12).TextColor(muted)
			}
			// The protocol permits a custom answer even when choices are supplied.
			defer func() {
				ptr := store.QuestionCustomPtr(eventID, row.ID)
				inp := ui.TextInput(c, ptr).Fill()
				if inp.Changed() {
					store.SetQuestionCustom(eventID, row.ID, *ptr)
				}
			}()
			if len(row.OptionLabels) == 0 {
				return
			}
			if row.MultiSelect {
				for index, opt := range row.OptionLabels {
					opt := opt
					checked := store.QuestionOptionCheckedPtr(eventID, row.ID, opt)
					box := ui.Checkbox(c, checked, opt)
					if index < len(row.OptionDescriptions) && row.OptionDescriptions[index] != "" {
						ui.Text(c, row.OptionDescriptions[index]).FontSize(11).TextColor(muted)
					}
					if box.Changed() {
						store.SetQuestionOptionChecked(eventID, row.ID, opt, *checked)
					}
				}
				return
			}
			if len(row.OptionLabels) <= 4 {
				for index, opt := range row.OptionLabels {
					opt := opt
					checked := store.QuestionOptionCheckedPtr(eventID, row.ID, opt)
					box := ui.Checkbox(c, checked, opt)
					if index < len(row.OptionDescriptions) && row.OptionDescriptions[index] != "" {
						ui.Text(c, row.OptionDescriptions[index]).FontSize(11).TextColor(muted)
					}
					if box.Changed() && *checked {
						store.SetQuestionOptionChecked(eventID, row.ID, opt, true)
					}
				}
				return
			}
			ptr := store.QuestionSingleSelectPtr(eventID, row.ID)
			sel := ui.Select(c, ptr, row.OptionLabels)
			if sel.Changed() {
				store.SetQuestionSingleSelect(eventID, row.ID, *ptr)
			}
		})
	}
}

func questionStatusText(status string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	switch status {
	case "continued":
		if zh {
			return "等待已结束，任务已继续；你仍可补充回答。"
		}
		return "The wait ended and work continued. You can still send a late answer."
	case "submit":
		if zh {
			return "发送补充回答"
		}
		return "Send late answer"
	case "submitted":
		if zh {
			return "补充回答已提交。"
		}
		return "Late answer submitted."
	default:
		if zh {
			return "此问题已不再接受补充回答。"
		}
		return "This question no longer accepts a late answer."
	}
}
