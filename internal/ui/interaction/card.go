// Package interaction renders pending Host waterfall prompts above the composer.
package interaction

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/theme"
)

// Model wires the card to the interaction store snapshot.
type Model struct {
	// Scope each mounted parent/child surface and its page shortcuts independently.
	ScopeAgentID         string
	DisablePageShortcuts bool
	OnOpenPlanResource   func(planpreview.Document)
	Snapshot             interaction.ViewState
	ThemeID              string
	SystemDark           bool
	Store                *interaction.Store
	// OnAutoAllowChanged is optional persistence when the user toggles auto-allow.
	OnAutoAllowChanged func(on bool)
}

// View paints auto-allow banner and pending approval / question cards.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	focusedAgent, focusedEvent := "", ""
	defer func() {
		if m.ScopeAgentID != "" {
			m.Store.SyncQuestionFocusForAgent(m.ScopeAgentID, focusedEvent)
		} else {
			m.Store.SyncQuestionFocus(focusedAgent, focusedEvent)
		}
	}()
	snap := m.Snapshot
	// Idle in manual mode: omit strip. Auto-allow on → banner (toggle to manual).
	// Pending → approval / question cards.
	if snap.Banner == "" && len(snap.Pending) == 0 && len(snap.QuestionHistory) == 0 && snap.QuestionStatus == "" && !snap.AutoAllow {
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
	strip := ui.Column(c).Background(field).Padding(8, 0, 0, 0).Gap(8)
	previous := ui.Local(strip, "page-approval-target", func() interaction.Pending { return interaction.Pending{} })
	if !m.DisablePageShortcuts {
		approvalPageShortcuts(c, previous, snap, func() interaction.ViewState {
			return m.Store.SnapshotForAgent(previous.AgentID)
		}, m.Store.ResolveAllow, m.Store.ResolveDeny)
	}
	strip.Children(func() {
		ui.Box(c).Fill().Height(1).Background(border)
		if snap.AutoAllow {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				msg := i18n.T(i18n.KeyInteractionAutoAllowOn)
				if snap.Banner != "" {
					msg = i18n.T(i18n.KeyInteractionAutoAllowed) + ": " + snap.Banner
				}
				ui.Text(c, msg).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
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
			ui.Text(c, text).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
		questionHistory(c, snap.QuestionHistory, m.Store, fg, muted, accent)
		for _, p := range snap.Pending {
			if pendingCard(c, p, snap, m.Store, m.OnAutoAllowChanged, fg, muted, accent, m.OnOpenPlanResource) {
				focusedAgent, focusedEvent = p.AgentID, p.EventID
			}
		}
	})
}

func pendingCard(c *ui.Context, p interaction.Pending, snap interaction.ViewState, store *interaction.Store, onAuto func(bool), fg, muted, accent ui.Color, onOpenPlan func(planpreview.Document)) bool {
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.PlanReview != nil {
		planReviewCard(c, p, card, store, fg, muted, accent, onOpenPlan)
		return false
	}
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.Hidden {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, p.Title).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
			if ui.Button(c, "重新打开问题 / Reopen questions").Clicked() {
				store.ReopenQuestion(p.EventID)
			}
		})
		return false
	}
	focused := false
	card := ui.Column(c.Key("approval-card:"+p.AgentID+":"+p.EventID)).Padding(8, 8, 8, 8).Gap(6).Background(accent.Alpha(0.08)).Radius(12)
	var keys *approvalKeys
	if p.Event == interaction.EventApproval {
		keys = ui.Local(card, "approval-keys", func() approvalKeys { return approvalKeys{} })
		card.Focusable().Label("Approval details: " + p.EventID).HandleInput(approvalHandler(keys, p, store, false))
	}
	card.Children(func() {
		title := p.Title
		if q, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && q.Index >= 0 && q.Index < len(q.Rows) {
			if q.Rows[q.Index].Header != "" {
				title = q.Rows[q.Index].Header
			} else if q.Rows[q.Index].Question != "" {
				title = q.Rows[q.Index].Question
			}
		}
		ui.Text(c, title).FontSize(theme.InterfaceSize(13)).Bold().TextColor(fg)
		if p.Event == interaction.EventUserQuestions {
			if p.Continued {
				ui.Text(c, questionStatusText("continued")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			if card, ok := snap.QuestionCards[p.EventID]; ok {
				questionWaitStatus(c, p, card, store, muted)
				focused = questionsBody(c, p, card, store, fg, muted)
			}
		} else {
			if p.Detail != "" {
				ui.Text(c, p.Detail).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
			if p.CallID != "" {
				ui.Text(c, i18n.T(i18n.KeyInteractionCallID)+": "+p.CallID).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			approvalArguments(c, p, store, keys, fg, muted, accent)
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FontSize(theme.InterfaceSize(12)).TextColor(muted)
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
				if ui.Button(c, "暂时隐藏 / Hide for now").Clicked() {
					store.HideQuestion(p.EventID)
				}

				if !p.Continued {
					if ui.Button(c, i18n.T(i18n.KeyInteractionDeny)).Clicked() {
						store.ResolveDeny(p.EventID)
					}
				}
			default:
				allow := ui.Button(c, i18n.T(i18n.KeyInteractionAllow))
				deny := ui.Button(c, i18n.T(i18n.KeyInteractionDeny))
				if keys != nil {
					allow.HandleInput(approvalHandler(keys, p, store, true))
					deny.HandleInput(approvalHandler(keys, p, store, true))
				}
				if allow.Clicked() {
					store.ResolveAllow(p.EventID)
				}
				if deny.Clicked() {
					store.ResolveDeny(p.EventID)
				}
			}
		})
	})

	return focused
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
