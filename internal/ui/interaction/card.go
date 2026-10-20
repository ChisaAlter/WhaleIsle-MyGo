// Package interaction renders pending Host waterfall prompts above the composer.
package interaction

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	Store                CardStore
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
	if snap.Banner == "" && len(snap.Pending) == 0 && len(snap.QuestionHistory) == 0 && snap.QuestionStatus == "" && snap.QuestionDraftError == "" && !snap.AutoAllow {
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
		if snap.QuestionDraftError != "" {
			ui.Text(c, snap.QuestionDraftError).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		}
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

func pendingCard(c *ui.Context, p interaction.Pending, snap interaction.ViewState, store CardStore, onAuto func(bool), fg, muted, accent ui.Color, onOpenPlan func(planpreview.Document)) bool {
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.PlanReview != nil {
		planReviewCard(c, p, card, store, fg, muted, accent, onOpenPlan)
		return false
	}
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.Hidden {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, p.Title).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
			if ui.Button(c, i18n.T(i18n.KeyChatQuestionReopen)).Clicked() {
				store.ReopenQuestion(p.EventID)
			}
		})
		return false
	}
	focused := false
	t := c.Theme()
	approval := p.Event == interaction.EventApproval
	tint := accent.Alpha(0.06)
	if approval {
		tint = t.Warning.Alpha(0.07)
	}
	card := interactionCardSeat(c, "approval-card:"+p.AgentID+":"+p.EventID, approval, tint)
	var keys *approvalKeys
	if approval {
		keys = ui.Local(card, "approval-keys", func() approvalKeys { return approvalKeys{} })
		card.Focusable().Label("Approval details: " + p.EventID).HandleInput(approvalHandler(keys, p, store, false))
	}
	card.Children(func() {
		title := p.Title
		if q, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && q.Index >= 0 && q.Index < len(q.Rows) {
			row := q.Rows[q.Index]
			if row.Header != "" && row.Question != "" {
				questionText(c, row.Header, 11, muted)
			}
			if row.Question != "" {
				title = row.Question
			} else if row.Header != "" {
				title = row.Header
			}
		}
		if approval {
			interactionCardHeader(c, widgets.StateWarning, i18n.T(i18n.KeyChatApprovalWaiting), p.CallID, muted)
		} else {
			// The pager below already reports "1 / 3"; the header names the wait.
			interactionCardHeader(c, widgets.StateRunning, i18n.T(i18n.KeyChatQuestionWaiting), "", muted)
		}
		ui.Text(c, title).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(15)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
		if p.Event == interaction.EventUserQuestions {
			if p.Continued {
				ui.Text(c, questionStatusText("continued")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			if card, ok := snap.QuestionCards[p.EventID]; ok {
				questionWaitStatus(c, p, card, store, muted)
				focused = questionsBody(c, p, card, store, fg, muted)
			}
		} else {
			approvalReason(c, p, muted)
			if command := approvalCommand(p.Args); command != "" {
				commandBlock(c, t, command)
			} else if p.CallID != "" {
				ui.Text(c, i18n.T(i18n.KeyInteractionCallID)+": "+p.CallID).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			approvalArguments(c, p, store, keys, fg, muted, accent)
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FontSize(theme.InterfaceSize(12)).TextColor(t.Danger)
		}
		ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Wrap().Children(func() {
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
				if hide := interactionSecondaryButton(c, t, i18n.T(i18n.KeyChatQuestionHide)); hide.Clicked() {
					store.HideQuestion(p.EventID)
				}

				if !p.Continued {
					if deny := interactionDangerButton(c, t, i18n.T(i18n.KeyInteractionDeny)); deny.Clicked() {
						store.ResolveDeny(p.EventID)
					}
				}
			default:
				deny := interactionDangerButton(c, t, i18n.T(i18n.KeyInteractionDeny))
				allow := interactionPrimaryButton(c, t, i18n.T(i18n.KeyInteractionAllow))
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
