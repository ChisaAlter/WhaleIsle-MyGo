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
	RenderSlot SlotRenderer
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
func View(c *ui.Context, m Model, resident ...ui.Rect) {
	previous := residentComposerBounds
	if len(resident) > 0 {
		residentComposerBounds = resident[0]
	} else {
		residentComposerBounds = ui.Rect{}
	}
	defer func() { residentComposerBounds = previous }()
	interactionView(c, m)
}

func interactionView(c *ui.Context, m Model) {
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
	strip := ui.Column(c).Padding(8, 0, 0, 0).Gap(8)
	_, _ = border, field
	previous := ui.Local(strip, "page-approval-target", func() interaction.Pending { return interaction.Pending{} })
	if !m.DisablePageShortcuts {
		approvalPageShortcuts(c, previous, snap, func() interaction.ViewState {
			return m.Store.SnapshotForAgent(previous.AgentID)
		}, m.Store.ResolveAllow, m.Store.ResolveDeny)
	}
	strip.Children(func() {
		// no opaque border line above composer
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
				if widgets.Button(c, colors, i18n.T(i18n.KeyInteractionManual), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
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
		questionHistory(c, snap.QuestionHistory, m.Store, colors, fg, muted, accent)
		// The source publishes one active interaction per Agent. Child histories
		// may show other native cards too, but they must not borrow that outlet
		// (especially when anonymous reviews have identical question schemas).
		slotOwners := make(map[string]bool)
		for _, p := range snap.Pending {
			var render SlotRenderer
			card := snap.QuestionCards[p.EventID]
			if !p.Continued && !card.Hidden && !card.PlanReviewHidden && !slotOwners[p.AgentID] {
				slotOwners[p.AgentID] = true
				render = m.RenderSlot
			}
			if pendingCard(c, p, snap, m.Store, m.OnAutoAllowChanged, colors, fg, muted, accent, m.OnOpenPlanResource, render) {
				focusedAgent, focusedEvent = p.AgentID, p.EventID
			}
		}
	})
}

func pendingCard(c *ui.Context, p interaction.Pending, snap interaction.ViewState, store CardStore, onAuto func(bool), colors theme.Colors, fg, muted, accent ui.Color, onOpenPlan func(planpreview.Document), renderers ...SlotRenderer) bool {
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.PlanReview != nil {
		planReviewCard(c, p, card, store, colors, fg, muted, accent, onOpenPlan, renderers...)
		return false
	}
	if card, ok := snap.QuestionCards[p.EventID]; p.Event == interaction.EventUserQuestions && ok && card.Hidden {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, p.Title).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1)
			if summary := questionAnsweredCountText(card.Rows); summary != "" {
				ui.Text(c, summary).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
			reopen := widgets.Button(c, colors, i18n.T(i18n.KeyChatQuestionReopen), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall})
			if reopen.Clicked() {
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
	if approval {
		card.Background(ui.Transparent).Draw(chrome.Glass(chrome.Color(colors.InputMajor), chrome.MaterialPanel, 20))
	} else {
		layer := ui.RGBA(255, 255, 255, .8)
		if c.Theme().Dark {
			layer = ui.RGBA(44, 44, 46, .8)
		}
		card.Background(ui.Transparent).Draw(chrome.Glass(layer, chrome.MaterialPanel, 20))
	}
	var keys *approvalKeys
	if approval {
		keys = ui.Local(card, "approval-keys", func() approvalKeys { return approvalKeys{} })
		card.Focusable().Label("Approval details: " + p.EventID).HandleInput(approvalHandler(keys, p, store, false))
	}
	card.Children(func() {
		if approval {
			ui.Row(c).FillWidth().Shrink(0).Padding(4, 16).Gap(8).Background(t.Warning.Alpha(.12)).Children(func() {
				state := widgets.StateWarning
				if p.Sending {
					state = widgets.StateRunning
				}
				widgets.StateDot(c, state)
				ui.Text(c, i18n.T(i18n.KeyChatApprovalWaiting)).FontSize(13).FixedLineHeight(18).TextColor(t.Warning)
			})
			body := ui.Scroll(c.Key("approval-body:"+p.EventID)).Label(approvalDetailAriaLabel()).FillWidth().Grow(1).MinHeight(0).Padding(0, 16).Gap(6)
			if residentComposerBounds.H <= 0 {
				body.MaxHeight(336)
			}
			body.Children(func() {
				ui.Text(c, p.Title).FillWidth().MinWidth(0).FontSize(15).FixedLineHeight(24).FontWeight(500).TextColor(fg)
				approvalReason(c, p, muted)
				renderInteractionSlot(c, p, "conversation.approval.detail", func() {
					if p.ApprovalContext != "" {
						ui.Text(c, p.ApprovalContext).FillWidth().MinWidth(0).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					}
					if command := approvalCommand(p.Args); command != "" {
						// The command seat already carries the exact string, so the raw
						// payload is not repeated as a second open block; the copy action
						// under the command still yields it.
						var copyInput func(ui.InputEvent) bool
						if keys != nil {
							copyInput = approvalHandler(keys, p, store, true)
						}
						commandBlock(c, t, command, func(text string) { c.WriteClipboard(text) }, copyInput)
					}
					if p.CallID != "" {
						ui.Text(c, i18n.T(i18n.KeyInteractionCallID)+": "+p.CallID).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					}
					if approvalCommand(p.Args) == "" {
						approvalArguments(c, p, store, keys, fg, muted, accent)
					}
				}, renderers...)
			})
		} else {
			title := p.Title
			if q, ok := snap.QuestionCards[p.EventID]; ok && q.Index >= 0 && q.Index < len(q.Rows) {
				row := q.Rows[q.Index]
				if row.Header != "" && row.Question != "" {
					ui.Text(c, row.Header).FontSize(11).FixedLineHeight(16).TextColor(muted).Padding(14, 24, 0, 24).Shrink(0)
				}
				if row.Question != "" {
					title = row.Question
				} else if row.Header != "" {
					title = row.Header
				}
			}
			ui.Text(c, title).FillWidth().MinWidth(0).FontSize(16).FixedLineHeight(22).FontWeight(500).TextColor(fg).Shrink(0).Padding(20, 24, 0, 24)
			if q, ok := snap.QuestionCards[p.EventID]; ok {
				questionWaitStatus(c, p, q, store, colors, muted)
				focused = questionsBody(c, p, q, store, colors, fg, muted)
			}
		}
		if p.Error != "" {
			ui.Text(c, p.Error).FontSize(theme.InterfaceSize(12)).TextColor(t.Danger)
		}
		ui.Row(c).FillWidth().Shrink(0).Padding(4, 16).Gap(8).Justify(ui.End).Wrap().Children(func() {
			if p.Sending {
				widgets.Button(c, colors, i18n.T(i18n.KeyInteractionContinue), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: true})
				return
			}
			if p.Error != "" {
				if retry := widgets.Button(c, colors, i18n.T(i18n.KeyBootRetry), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}); retry.Clicked() {
					store.Retry(p.EventID)
				}
				if p.Event != interaction.EventUserQuestions {
					return
				}
			}
			switch p.Event {
			case interaction.EventUserQuestions:
				if hide := interactionGhostButton(c, colors, i18n.T(i18n.KeyChatQuestionHide), false).Tooltip(questionStatusText("nav.close")); hide.Clicked() {
					store.HideQuestion(p.EventID)
				}

				if !p.Continued {
					if deny := interactionOutlineButton(c, colors, i18n.T(i18n.KeyInteractionDeny), false).Tooltip(questionStatusText("nav.cancel")); deny.Clicked() {
						store.ResolveDeny(p.EventID)
					}
				}
			default:
				renderInteractionSlot(c, p, "conversation.approval.actions", func() {}, renderers...)
				deny := interactionOutlineButton(c, colors, i18n.T(i18n.KeyInteractionDeny), false)
				deny.Children(func() {
					ui.Box(c).Fill().PassThrough().Label(i18n.T(i18n.KeyParityApprovalReject))
				})
				allow := interactionPrimaryButton(c, colors, i18n.T(i18n.KeyInteractionAllow), false)
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
	case "waiting", "ask.waiting":
		if zh {
			return "等待回答"
		}
		return "waiting"
	case "pending", "ask.pending":
		if zh {
			return "已继续工作，仍可回答"
		}
		return "continued; answer still available"
	case "pendingDetail", "ask.pendingDetail":
		if zh {
			return "等待中的问题仍可在输入框中回答"
		}
		return "These pending questions remain answerable from the composer."
	case "closed", "ask.closed":
		if zh {
			return "已结束"
		}
		return "closed"
	case "closedDetail", "ask.closedDetail":
		if zh {
			return "该问题已结束，结果见下方对话"
		}
		return "This question is closed; its outcome is in the conversation below."
	case "cancelled", "ask.cancelled":
		if zh {
			return "已取消"
		}
		return "cancelled"
	case "cancelledDetail", "ask.cancelledDetail":
		if zh {
			return "本轮已取消，未提交回答"
		}
		return "This question set was cancelled before answers were submitted."
	case "interrupted", "ask.interrupted":
		if zh {
			return "已中断"
		}
		return "interrupted"
	case "interruptedDetail", "ask.interruptedDetail":
		if zh {
			return "本轮已中断，未提交回答"
		}
		return "This question set was interrupted before answers were submitted."
	case "skipped", "ask.skipped":
		if zh {
			return "未回答"
		}
		return "Not answered"
	case "resubmit", "error.resubmit":
		if zh {
			return "回答未送达，工作已继续，请再提交一次。"
		}
		return "The answer did not arrive before work continued; submit it again."
	case "unavailable", "error.unavailable":
		if zh {
			return "当前无法提交，请稍候再试。"
		}
		return "Cannot submit right now; try again in a moment."
	case "sent", "status.sent":
		if zh {
			return "回答已发送；面板未能关闭。"
		}
		return "Reply sent; the panel could not close."
	case "review.skipped":
		if zh {
			return "这道问题当时被跳过。"
		}
		return "This question was skipped."
	case "reply.label":
		if zh {
			return "回答先前等待中的问题"
		}
		return "Reply to earlier pending questions"
	case "reply.open":
		if zh {
			return "展开问题详情"
		}
		return "Open question details"
	case "reply.close":
		if zh {
			return "收起问题详情"
		}
		return "Close question details"
	case "nav.minimize":
		if zh {
			return "收起问题卡片"
		}
		return "Collapse the question card"
	case "nav.maximize":
		if zh {
			return "展开问题卡片"
		}
		return "Expand the question card"
	case "nav.cancel":
		if zh {
			return "放弃整组问题"
		}
		return "Dismiss all questions"
	case "nav.close":
		if zh {
			return "收起问题面板，可从工具调用重新打开"
		}
		return "Close the panel — reopen it from the tool call"
	case "custom.placeholder":
		if zh {
			return "输入你的答案"
		}
		return "Type your answer"
	default:
		if zh {
			return "此问题已不再接受补充回答。"
		}
		return "This question no longer accepts a late answer."
	}
}
